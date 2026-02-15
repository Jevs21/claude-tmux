package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeTestLog creates a temporary event log file and sets up the environment
// so eventLogPath() resolves to it. Returns a cleanup function.
func writeTestLog(t *testing.T, lines []string) (cleanup func()) {
	t.Helper()

	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, ".claude-tmux")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create temp log dir: %v", err)
	}

	logFile := filepath.Join(logDir, "events.log")
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp log: %v", err)
	}

	// Override HOME so eventLogPath() resolves to our temp dir
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	return func() {
		os.Setenv("HOME", originalHome)
	}
}

func TestParseTmuxTarget(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSession string
		expectedWindow  int
		expectedPane    int
	}{
		{
			name:            "standard target",
			input:           "work:2.0",
			expectedSession: "work",
			expectedWindow:  2,
			expectedPane:    0,
		},
		{
			name:            "multi-digit window and pane",
			input:           "dev:10.3",
			expectedSession: "dev",
			expectedWindow:  10,
			expectedPane:    3,
		},
		{
			name:            "empty target",
			input:           "",
			expectedSession: "",
			expectedWindow:  0,
			expectedPane:    0,
		},
		{
			name:            "session name only no colon",
			input:           "work",
			expectedSession: "work",
			expectedWindow:  0,
			expectedPane:    0,
		},
		{
			name:            "window without pane",
			input:           "work:2",
			expectedSession: "work",
			expectedWindow:  2,
			expectedPane:    0,
		},
		{
			name:            "session name with colon in name",
			input:           "my:session:2.1",
			expectedSession: "my:session",
			expectedWindow:  2,
			expectedPane:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionName, windowIndex, paneIndex := parseTmuxTarget(tt.input)
			if sessionName != tt.expectedSession {
				t.Errorf("sessionName: got %q, want %q", sessionName, tt.expectedSession)
			}
			if windowIndex != tt.expectedWindow {
				t.Errorf("windowIndex: got %d, want %d", windowIndex, tt.expectedWindow)
			}
			if paneIndex != tt.expectedPane {
				t.Errorf("paneIndex: got %d, want %d", paneIndex, tt.expectedPane)
			}
		})
	}
}

func TestReadSessions_MissingLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestReadSessions_MalformedLinesSkipped(t *testing.T) {
	// Use current PID so the process is alive
	currentPID := os.Getpid()
	cleanup := writeTestLog(t, []string{
		"not json at all",
		fmt.Sprintf(`{"ts":1707900000,"sid":"abc","event":"session-start","pid":%d,"cwd":"/tmp/proj","tmux":"work:0.0","tool":""}`, currentPID),
		`{malformed json`,
		"",
		fmt.Sprintf(`{"ts":1707900001,"sid":"abc","event":"stop","pid":%d,"cwd":"/tmp/proj","tmux":"work:0.0","tool":""}`, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].SessionID != "abc" {
		t.Errorf("expected session ID 'abc', got %q", sessions[0].SessionID)
	}
	if sessions[0].Status != StatusIdle {
		t.Errorf("expected StatusIdle after stop, got %d", sessions[0].Status)
	}
}

func TestReadSessions_FullLifecycle(t *testing.T) {
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/home/user/proj","tmux":"work:0.0","tool":""}`, ts, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"user-prompt-submit","pid":%d,"cwd":"/home/user/proj","tmux":"work:0.0","tool":""}`, ts+1, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"pre-tool-use","pid":%d,"cwd":"/home/user/proj","tmux":"work:0.0","tool":"Bash"}`, ts+2, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"post-tool-use","pid":%d,"cwd":"/home/user/proj","tmux":"work:0.0","tool":"Bash"}`, ts+3, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"stop","pid":%d,"cwd":"/home/user/proj","tmux":"work:0.0","tool":""}`, ts+4, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-end","pid":%d,"cwd":"/home/user/proj","tmux":"work:0.0","tool":""}`, ts+5, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Session ended, so it should be removed
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after session-end, got %d", len(sessions))
	}
}

func TestReadSessions_MultipleConcurrentSessions(t *testing.T) {
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj/alpha","tmux":"work:0.0","tool":""}`, ts, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s2","event":"session-start","pid":%d,"cwd":"/proj/beta","tmux":"dev:1.0","tool":""}`, ts+1, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"user-prompt-submit","pid":%d,"cwd":"/proj/alpha","tmux":"work:0.0","tool":""}`, ts+2, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s2","event":"pre-tool-use","pid":%d,"cwd":"/proj/beta","tmux":"dev:1.0","tool":"Read"}`, ts+3, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Build a map by session ID for easier assertions
	sessionsByID := make(map[string]Session)
	for _, s := range sessions {
		sessionsByID[s.SessionID] = s
	}

	s1 := sessionsByID["s1"]
	if s1.Status != StatusBusy {
		t.Errorf("s1: expected StatusBusy, got %d", s1.Status)
	}
	if s1.Action != "Thinking\u2026" {
		t.Errorf("s1: expected action 'Thinking\u2026', got %q", s1.Action)
	}
	if s1.ProjectName != "alpha" {
		t.Errorf("s1: expected project 'alpha', got %q", s1.ProjectName)
	}

	s2 := sessionsByID["s2"]
	if s2.Status != StatusBusy {
		t.Errorf("s2: expected StatusBusy, got %d", s2.Status)
	}
	if s2.Action != "Read" {
		t.Errorf("s2: expected action 'Read', got %q", s2.Action)
	}
	if s2.TmuxSession != "dev" {
		t.Errorf("s2: expected tmux session 'dev', got %q", s2.TmuxSession)
	}
}

func TestReadSessions_StateTransitions(t *testing.T) {
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	tests := []struct {
		name           string
		events         []string
		expectedStatus Status
		expectedAction string
	}{
		{
			name: "session-start is idle",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
			},
			expectedStatus: StatusIdle,
			expectedAction: "",
		},
		{
			name: "user-prompt-submit is busy with thinking",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"user-prompt-submit","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
			},
			expectedStatus: StatusBusy,
			expectedAction: "Thinking\u2026",
		},
		{
			name: "pre-tool-use is busy with tool name",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"pre-tool-use","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":"Edit"}`, ts+1, currentPID),
			},
			expectedStatus: StatusBusy,
			expectedAction: "Edit",
		},
		{
			name: "post-tool-use is busy with no action",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"post-tool-use","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
			},
			expectedStatus: StatusBusy,
			expectedAction: "",
		},
		{
			name: "post-tool-use-failure is busy",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"post-tool-use-failure","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
			},
			expectedStatus: StatusBusy,
			expectedAction: "",
		},
		{
			name: "stop is idle",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"user-prompt-submit","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"stop","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+2, currentPID),
			},
			expectedStatus: StatusIdle,
			expectedAction: "",
		},
		{
			name: "permission-request is waiting",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"permission-request","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
			},
			expectedStatus: StatusWaiting,
			expectedAction: "Permission",
		},
		{
			name: "notification-idle is idle",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"user-prompt-submit","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"notification-idle","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+2, currentPID),
			},
			expectedStatus: StatusIdle,
			expectedAction: "",
		},
		{
			name: "notification-permission is waiting",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"notification-permission","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
			},
			expectedStatus: StatusWaiting,
			expectedAction: "Permission",
		},
		{
			name: "notification-elicitation is waiting with input",
			events: []string{
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts, currentPID),
				fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"notification-elicitation","pid":%d,"cwd":"/proj","tmux":"w:0.0","tool":""}`, ts+1, currentPID),
			},
			expectedStatus: StatusWaiting,
			expectedAction: "Input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := writeTestLog(t, tt.events)
			defer cleanup()

			sessions, err := ReadSessions()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sessions) != 1 {
				t.Fatalf("expected 1 session, got %d", len(sessions))
			}
			if sessions[0].Status != tt.expectedStatus {
				t.Errorf("status: got %d, want %d", sessions[0].Status, tt.expectedStatus)
			}
			if sessions[0].Action != tt.expectedAction {
				t.Errorf("action: got %q, want %q", sessions[0].Action, tt.expectedAction)
			}
		})
	}
}

func TestReadSessions_EventWithoutSessionStart(t *testing.T) {
	// Events arriving for an unknown session should create a minimal entry
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":%d,"sid":"orphan","event":"pre-tool-use","pid":%d,"cwd":"/proj/orphan","tmux":"work:1.0","tool":"Bash"}`, ts, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].SessionID != "orphan" {
		t.Errorf("expected session ID 'orphan', got %q", sessions[0].SessionID)
	}
	if sessions[0].Status != StatusBusy {
		t.Errorf("expected StatusBusy, got %d", sessions[0].Status)
	}
	if sessions[0].Action != "Bash" {
		t.Errorf("expected action 'Bash', got %q", sessions[0].Action)
	}
}

func TestReadSessions_EmptySessionIDSkipped(t *testing.T) {
	cleanup := writeTestLog(t, []string{
		`{"ts":1707900000,"sid":"","event":"session-start","pid":999,"cwd":"/proj","tmux":"w:0.0","tool":""}`,
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for empty sid, got %d", len(sessions))
	}
}

func TestRotateLog(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, ".claude-tmux")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create temp log dir: %v", err)
	}

	// Create a log with 1100 lines
	var lines []string
	for i := 0; i < 1100; i++ {
		lines = append(lines, fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"pre-tool-use","pid":1,"cwd":"/proj","tmux":"w:0.0","tool":"line-%d"}`, 1707900000+i, i))
	}

	logFile := filepath.Join(logDir, "events.log")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write log: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	if err := RotateLog(); err != nil {
		t.Fatalf("RotateLog error: %v", err)
	}

	// Read back and count lines
	rotatedContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read rotated log: %v", err)
	}

	rotatedLines := strings.Split(strings.TrimSpace(string(rotatedContent)), "\n")
	if len(rotatedLines) != 500 {
		t.Errorf("expected 500 lines after rotation, got %d", len(rotatedLines))
	}

	// Verify the last line is the last original line (line-1099)
	if !strings.Contains(rotatedLines[499], "line-1099") {
		t.Errorf("expected last line to contain 'line-1099', got %q", rotatedLines[499])
	}

	// Verify the first kept line is line-600 (1100 - 500)
	if !strings.Contains(rotatedLines[0], "line-600") {
		t.Errorf("expected first kept line to contain 'line-600', got %q", rotatedLines[0])
	}
}

func TestRotateLog_NoRotationNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, ".claude-tmux")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create temp log dir: %v", err)
	}

	// Create a log with only 50 lines
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"stop","pid":1,"cwd":"/proj","tmux":"w:0.0","tool":""}`, 1707900000+i))
	}

	logFile := filepath.Join(logDir, "events.log")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write log: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	if err := RotateLog(); err != nil {
		t.Fatalf("RotateLog error: %v", err)
	}

	// File should be unchanged
	rotatedContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	rotatedLines := strings.Split(strings.TrimSpace(string(rotatedContent)), "\n")
	if len(rotatedLines) != 50 {
		t.Errorf("expected 50 lines (no rotation), got %d", len(rotatedLines))
	}
}

func TestRotateLog_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Should not error when file doesn't exist
	if err := RotateLog(); err != nil {
		t.Fatalf("RotateLog should not error on missing file: %v", err)
	}
}

func TestReadSessions_DeadProcessPruned(t *testing.T) {
	// Use a PID that is almost certainly not alive
	deadPID := 2147483647

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":1707900000,"sid":"dead","event":"session-start","pid":%d,"cwd":"/proj","tmux":"work:0.0","tool":""}`, deadPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions (dead PID pruned), got %d", len(sessions))
	}
}

func TestReadSessions_TmuxPaneDeduplication(t *testing.T) {
	// Two alive sessions claiming the same tmux target — only the most recently
	// updated one should survive.
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	cleanup := writeTestLog(t, []string{
		// Older session on work:0.0
		fmt.Sprintf(`{"ts":%d,"sid":"old-session","event":"session-start","pid":%d,"cwd":"/proj/old","tmux":"work:0.0","tool":""}`, ts, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"old-session","event":"user-prompt-submit","pid":%d,"cwd":"/proj/old","tmux":"work:0.0","tool":""}`, ts+1, currentPID),
		// Newer session also claiming work:0.0 (stale log data scenario)
		fmt.Sprintf(`{"ts":%d,"sid":"new-session","event":"session-start","pid":%d,"cwd":"/proj/new","tmux":"work:0.0","tool":""}`, ts+10, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"new-session","event":"pre-tool-use","pid":%d,"cwd":"/proj/new","tmux":"work:0.0","tool":"Read"}`, ts+11, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after dedup, got %d", len(sessions))
	}
	if sessions[0].SessionID != "new-session" {
		t.Errorf("expected surviving session to be 'new-session', got %q", sessions[0].SessionID)
	}
	if sessions[0].ProjectName != "new" {
		t.Errorf("expected project 'new', got %q", sessions[0].ProjectName)
	}
}

func TestReadSessions_TmuxPaneDedup_DifferentPanesPreserved(t *testing.T) {
	// Sessions on different tmux targets should NOT be deduped.
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj/a","tmux":"work:0.0","tool":""}`, ts, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s2","event":"session-start","pid":%d,"cwd":"/proj/b","tmux":"work:0.1","tool":""}`, ts+1, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions on different panes, got %d", len(sessions))
	}
}

func TestReadSessions_TmuxTargetNotOverwrittenByLaterEvents(t *testing.T) {
	// Verify that a session's tmux target set at session-start is not overwritten
	// by subsequent events that might carry a different tmux value.
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"work:0.0","tool":""}`, ts, currentPID),
		// Later event carries a different tmux target (simulating the old hook bug)
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"pre-tool-use","pid":%d,"cwd":"/proj","tmux":"work:1.0","tool":"Bash"}`, ts+1, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	// TmuxTarget should still be the original from session-start
	if sessions[0].TmuxTarget != "work:0.0" {
		t.Errorf("expected tmux target 'work:0.0' (from session-start), got %q", sessions[0].TmuxTarget)
	}
	if sessions[0].WindowIndex != 0 {
		t.Errorf("expected window index 0, got %d", sessions[0].WindowIndex)
	}
	if sessions[0].PaneIndex != 0 {
		t.Errorf("expected pane index 0, got %d", sessions[0].PaneIndex)
	}
}

func TestReadSessions_TmuxTargetParsedCorrectly(t *testing.T) {
	currentPID := os.Getpid()

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":1707900000,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj","tmux":"mywork:3.1","tool":""}`, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.TmuxSession != "mywork" {
		t.Errorf("expected tmux session 'mywork', got %q", s.TmuxSession)
	}
	if s.WindowIndex != 3 {
		t.Errorf("expected window index 3, got %d", s.WindowIndex)
	}
	if s.PaneIndex != 1 {
		t.Errorf("expected pane index 1, got %d", s.PaneIndex)
	}
	if s.TmuxTarget != "mywork:3.1" {
		t.Errorf("expected tmux target 'mywork:3.1', got %q", s.TmuxTarget)
	}
}

func TestReadSessions_CWDUpdatedMidSession(t *testing.T) {
	currentPID := os.Getpid()
	ts := time.Now().Unix()

	cleanup := writeTestLog(t, []string{
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"session-start","pid":%d,"cwd":"/proj/alpha","tmux":"work:0.0","tool":""}`, ts, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"user-prompt-submit","pid":%d,"cwd":"/proj/alpha","tmux":"work:0.0","tool":""}`, ts+1, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"pre-tool-use","pid":%d,"cwd":"/proj/beta","tmux":"work:0.0","tool":"Bash"}`, ts+2, currentPID),
		fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"stop","pid":%d,"cwd":"/proj/beta","tmux":"work:0.0","tool":""}`, ts+3, currentPID),
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].WorkDir != "/proj/beta" {
		t.Errorf("expected WorkDir '/proj/beta', got %q", sessions[0].WorkDir)
	}
	if sessions[0].ProjectName != "beta" {
		t.Errorf("expected ProjectName 'beta', got %q", sessions[0].ProjectName)
	}
}

func TestReadSessions_EmptyLogFile(t *testing.T) {
	// Create an empty events.log file (0 bytes)
	cleanup := writeTestLog(t, []string{})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions from empty log, got %d", len(sessions))
	}
}

func TestReadSessions_OnlySessionEnd(t *testing.T) {
	// A session-end for an ID that was never started should be a harmless no-op
	cleanup := writeTestLog(t, []string{
		`{"ts":1707900000,"sid":"ghost","event":"session-end","pid":12345,"cwd":"/proj","tmux":"work:0.0","tool":""}`,
	})
	defer cleanup()

	sessions, err := ReadSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for session-end only, got %d", len(sessions))
	}
}

func TestRotateLog_ExactBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, ".claude-tmux")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create temp log dir: %v", err)
	}

	// Create a log with exactly 1001 lines (just over the 1000 threshold)
	var lines []string
	for i := 0; i < 1001; i++ {
		lines = append(lines, fmt.Sprintf(`{"ts":%d,"sid":"s1","event":"pre-tool-use","pid":1,"cwd":"/proj","tmux":"w:0.0","tool":"line-%d"}`, 1707900000+i, i))
	}

	logFile := filepath.Join(logDir, "events.log")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write log: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	if err := RotateLog(); err != nil {
		t.Fatalf("RotateLog error: %v", err)
	}

	rotatedContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read rotated log: %v", err)
	}

	rotatedLines := strings.Split(strings.TrimSpace(string(rotatedContent)), "\n")
	if len(rotatedLines) != 500 {
		t.Errorf("expected 500 lines after rotation, got %d", len(rotatedLines))
	}

	// First kept line should be line 501 (index 501, since 1001-500=501)
	if !strings.Contains(rotatedLines[0], "line-501") {
		t.Errorf("expected first kept line to contain 'line-501', got %q", rotatedLines[0])
	}

	// Last line should be the original last line (line-1000)
	if !strings.Contains(rotatedLines[499], "line-1000") {
		t.Errorf("expected last line to contain 'line-1000', got %q", rotatedLines[499])
	}
}
