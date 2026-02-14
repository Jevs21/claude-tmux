package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Status represents the activity state of a Claude Code session.
type Status int

const (
	StatusUnknown Status = iota
	StatusIdle
	StatusBusy
)

// Session represents a running Claude Code process mapped to a tmux pane.
type Session struct {
	PID         int
	PPID        int
	WorkDir     string
	ProjectName string
	TmuxTarget  string // "session:window.pane" or empty if detached
	TmuxSession string
	WindowIndex int
	PaneIndex   int
	Status      Status
}

// DisplayPath returns the working directory with the home directory replaced by ~
// and long paths truncated.
func (s Session) DisplayPath() string {
	return ShortenPath(s.WorkDir)
}

// Jumpable returns true if the session is mapped to a tmux pane.
func (s Session) Jumpable() bool {
	return s.TmuxTarget != ""
}

// DisplayTarget returns a formatted tmux target string for display.
func (s Session) DisplayTarget() string {
	if s.TmuxTarget == "" {
		return "detached"
	}
	return fmt.Sprintf("%s:%d", s.TmuxSession, s.WindowIndex)
}

// ShortenPath replaces the home directory with ~ and truncates long paths.
func ShortenPath(path string) string {
	homeDir, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, homeDir) {
		path = "~" + path[len(homeDir):]
	}

	sep := string(filepath.Separator)
	parts := strings.Split(path, sep)

	// Filter out empty parts from leading separator
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}

	if len(nonEmpty) <= 4 {
		return path
	}

	// Show first component + ... + last two components
	prefix := ""
	if strings.HasPrefix(path, sep) {
		prefix = sep
	}
	shortened := prefix + nonEmpty[0] + sep + "..." +
		sep + strings.Join(nonEmpty[len(nonEmpty)-2:], sep)
	return shortened
}

// SortSessions sorts sessions by tmux session name, then window index.
// Detached sessions (no tmux target) are sorted to the end.
func SortSessions(sessions []Session) {
	sort.Slice(sessions, func(i, j int) bool {
		a, b := sessions[i], sessions[j]

		// Detached sessions go to the end
		if a.TmuxTarget == "" && b.TmuxTarget != "" {
			return false
		}
		if a.TmuxTarget != "" && b.TmuxTarget == "" {
			return true
		}

		// Sort by tmux session name
		if a.TmuxSession != b.TmuxSession {
			return a.TmuxSession < b.TmuxSession
		}

		// Then by window index
		return a.WindowIndex < b.WindowIndex
	})
}
