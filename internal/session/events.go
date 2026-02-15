package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// RawEvent represents a single JSON line from the events log.
type RawEvent struct {
	Timestamp int64  `json:"ts"`
	SessionID string `json:"sid"`
	Event     string `json:"event"`
	PID       int    `json:"pid"`
	CWD       string `json:"cwd"`
	TmuxInfo  string `json:"tmux"`
	ToolName  string `json:"tool"`
}

// eventLogPath returns the path to the events log file.
func eventLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".claude-tmux", "events.log")
}

// ReadSessions reads the event log and derives the current set of active sessions.
// It returns an empty slice if the log file does not exist.
func ReadSessions() ([]Session, error) {
	logPath := eventLogPath()
	if logPath == "" {
		return nil, nil
	}

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open event log: %w", err)
	}
	defer file.Close()

	sessionMap := make(map[string]*Session)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event RawEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip malformed lines gracefully
			continue
		}

		if event.SessionID == "" {
			continue
		}

		eventTime := time.Unix(event.Timestamp, 0)

		switch event.Event {
		case "session-start":
			tmuxSession, windowIndex, paneIndex := parseTmuxTarget(event.TmuxInfo)
			sessionMap[event.SessionID] = &Session{
				SessionID:   event.SessionID,
				ClaudePID:   event.PID,
				WorkDir:     event.CWD,
				ProjectName: filepath.Base(event.CWD),
				TmuxTarget:  event.TmuxInfo,
				TmuxSession: tmuxSession,
				WindowIndex: windowIndex,
				PaneIndex:   paneIndex,
				Status:      StatusIdle,
				LastUpdate:  eventTime,
			}

		case "session-end":
			delete(sessionMap, event.SessionID)

		default:
			session, exists := sessionMap[event.SessionID]
			if !exists {
				// Event for an unknown session — create a minimal entry
				tmuxSession, windowIndex, paneIndex := parseTmuxTarget(event.TmuxInfo)
				session = &Session{
					SessionID:   event.SessionID,
					ClaudePID:   event.PID,
					WorkDir:     event.CWD,
					ProjectName: filepath.Base(event.CWD),
					TmuxTarget:  event.TmuxInfo,
					TmuxSession: tmuxSession,
					WindowIndex: windowIndex,
					PaneIndex:   paneIndex,
				}
				sessionMap[event.SessionID] = session
			}

			session.LastUpdate = eventTime

			// Update CWD if provided (it may change during a session)
			if event.CWD != "" {
				session.WorkDir = event.CWD
				session.ProjectName = filepath.Base(event.CWD)
			}
			// Note: TmuxTarget is intentionally NOT updated from regular events.
			// It is only set from session-start or initial creation of unknown sessions.
			// Updating it from every event would overwrite the correct pane assignment
			// with whatever pane happened to be focused when the hook fired.
			if event.PID != 0 {
				session.ClaudePID = event.PID
			}

			applyEventStatus(session, event)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read event log: %w", err)
	}

	// Prune dead sessions: check if ClaudePID is still alive
	for sessionID, session := range sessionMap {
		if session.ClaudePID > 0 && !isProcessAlive(session.ClaudePID) {
			delete(sessionMap, sessionID)
		}
	}

	// Deduplicate sessions sharing the same tmux pane.
	// If two sessions claim the same non-empty TmuxTarget, keep only the one
	// with the most recent LastUpdate. This handles stale log data from before
	// the hook fix that ensures each pane reports its own target.
	tmuxTargetOwner := make(map[string]string) // TmuxTarget -> SessionID of best candidate
	for sessionID, session := range sessionMap {
		if session.TmuxTarget == "" {
			continue
		}
		existingID, exists := tmuxTargetOwner[session.TmuxTarget]
		if !exists {
			tmuxTargetOwner[session.TmuxTarget] = sessionID
			continue
		}
		existingSession := sessionMap[existingID]
		if session.LastUpdate.After(existingSession.LastUpdate) {
			// Current session is newer; evict the older one
			delete(sessionMap, existingID)
			tmuxTargetOwner[session.TmuxTarget] = sessionID
		} else {
			// Existing session is newer or equal; evict the current one
			delete(sessionMap, sessionID)
		}
	}

	// Convert map to sorted slice
	sessions := make([]Session, 0, len(sessionMap))
	for _, session := range sessionMap {
		sessions = append(sessions, *session)
	}
	SortSessions(sessions)

	return sessions, nil
}

// applyEventStatus updates a session's Status and Action based on an event.
func applyEventStatus(session *Session, event RawEvent) {
	switch event.Event {
	case "user-prompt-submit":
		session.Status = StatusBusy
		session.Action = "Thinking\u2026"
	case "pre-tool-use":
		session.Status = StatusBusy
		if event.ToolName != "" {
			session.Action = event.ToolName
		}
	case "post-tool-use", "post-tool-use-failure":
		session.Status = StatusBusy
		session.Action = ""
	case "stop":
		session.Status = StatusIdle
		session.Action = ""
	case "permission-request":
		session.Status = StatusWaiting
		session.Action = "Permission"
	case "notification-idle":
		session.Status = StatusIdle
		session.Action = ""
	case "notification-permission":
		session.Status = StatusWaiting
		session.Action = "Permission"
	case "notification-elicitation":
		session.Status = StatusWaiting
		session.Action = "Input"
	}
}

// parseTmuxTarget splits a tmux target string like "work:2.0" into its components.
// Returns empty string and zeros if the target is empty or malformed.
func parseTmuxTarget(target string) (sessionName string, windowIndex int, paneIndex int) {
	if target == "" {
		return "", 0, 0
	}

	colonIndex := strings.LastIndex(target, ":")
	if colonIndex < 0 {
		return target, 0, 0
	}

	sessionName = target[:colonIndex]
	remainder := target[colonIndex+1:]

	dotIndex := strings.Index(remainder, ".")
	if dotIndex < 0 {
		windowIndex, _ = strconv.Atoi(remainder)
		return sessionName, windowIndex, 0
	}

	windowIndex, _ = strconv.Atoi(remainder[:dotIndex])
	paneIndex, _ = strconv.Atoi(remainder[dotIndex+1:])
	return sessionName, windowIndex, paneIndex
}

// isProcessAlive checks if a process with the given PID is still running.
func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

// RotateLog truncates the event log to the last 500 lines if it exceeds 1000 lines.
// Called on startup to prevent unbounded log growth.
func RotateLog() error {
	logPath := eventLogPath()
	if logPath == "" {
		return nil
	}

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open event log for rotation: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read event log for rotation: %w", err)
	}

	if len(lines) <= 1000 {
		return nil
	}

	// Keep last 500 lines
	keepLines := lines[len(lines)-500:]
	content := strings.Join(keepLines, "\n") + "\n"

	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write rotated event log: %w", err)
	}

	return nil
}
