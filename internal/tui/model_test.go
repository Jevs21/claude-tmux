package tui

import (
	"strings"
	"testing"

	"github.com/Jevs21/claude-tmux/internal/session"
)

func TestApplyFilter(t *testing.T) {
	baseSessions := []session.Session{
		{SessionID: "s1", ProjectName: "claude-tmux", TmuxTarget: "work:0.0", WorkDir: "/home/user/projects/claude-tmux"},
		{SessionID: "s2", ProjectName: "api-server", TmuxTarget: "dev:1.0", WorkDir: "/home/user/projects/api-server"},
		{SessionID: "s3", ProjectName: "frontend", TmuxTarget: "work:2.0", WorkDir: "/home/user/projects/frontend"},
	}

	tests := []struct {
		name           string
		filterText     string
		expectedCount  int
		expectedIDs    []string
	}{
		{
			name:          "empty filter returns all sessions",
			filterText:    "",
			expectedCount: 3,
			expectedIDs:   []string{"s1", "s2", "s3"},
		},
		{
			name:          "match on ProjectName",
			filterText:    "api-server",
			expectedCount: 1,
			expectedIDs:   []string{"s2"},
		},
		{
			name:          "match on TmuxTarget",
			filterText:    "dev:1",
			expectedCount: 1,
			expectedIDs:   []string{"s2"},
		},
		{
			name:          "match on WorkDir",
			filterText:    "frontend",
			expectedCount: 1,
			expectedIDs:   []string{"s3"},
		},
		{
			name:          "case-insensitive matching",
			filterText:    "CLAUDE-TMUX",
			expectedCount: 1,
			expectedIDs:   []string{"s1"},
		},
		{
			name:          "no matches returns empty filtered list",
			filterText:    "nonexistent",
			expectedCount: 0,
			expectedIDs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				sessions:   baseSessions,
				filterText: tt.filterText,
			}
			m.applyFilter()

			if len(m.filtered) != tt.expectedCount {
				t.Fatalf("expected %d filtered sessions, got %d", tt.expectedCount, len(m.filtered))
			}
			for i, expectedID := range tt.expectedIDs {
				if m.filtered[i].SessionID != expectedID {
					t.Errorf("position %d: expected SessionID %q, got %q", i, expectedID, m.filtered[i].SessionID)
				}
			}
		})
	}
}

func TestClampCursor(t *testing.T) {
	tests := []struct {
		name           string
		filtered       []session.Session
		cursorBefore   int
		expectedCursor int
	}{
		{
			name: "cursor in bounds unchanged",
			filtered: []session.Session{
				{SessionID: "s1"},
				{SessionID: "s2"},
				{SessionID: "s3"},
			},
			cursorBefore:   1,
			expectedCursor: 1,
		},
		{
			name: "cursor beyond end clamped to last index",
			filtered: []session.Session{
				{SessionID: "s1"},
				{SessionID: "s2"},
			},
			cursorBefore:   5,
			expectedCursor: 1,
		},
		{
			name:           "empty filtered list with cursor 0 stays 0",
			filtered:       []session.Session{},
			cursorBefore:   0,
			expectedCursor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				filtered: tt.filtered,
				cursor:   tt.cursorBefore,
			}
			m.clampCursor()
			if m.cursor != tt.expectedCursor {
				t.Errorf("expected cursor %d, got %d", tt.expectedCursor, m.cursor)
			}
		})
	}
}

func TestSelectedSessionID(t *testing.T) {
	sessions := []session.Session{
		{SessionID: "s1"},
		{SessionID: "s2"},
		{SessionID: "s3"},
	}

	tests := []struct {
		name       string
		filtered   []session.Session
		cursor     int
		expectedID string
	}{
		{
			name:       "valid cursor returns correct SessionID",
			filtered:   sessions,
			cursor:     1,
			expectedID: "s2",
		},
		{
			name:       "cursor out of bounds returns empty string",
			filtered:   sessions,
			cursor:     10,
			expectedID: "",
		},
		{
			name:       "empty filtered list returns empty string",
			filtered:   []session.Session{},
			cursor:     0,
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				filtered: tt.filtered,
				cursor:   tt.cursor,
			}
			result := m.selectedSessionID()
			if result != tt.expectedID {
				t.Errorf("expected %q, got %q", tt.expectedID, result)
			}
		})
	}
}

func TestRestoreCursorBySessionID(t *testing.T) {
	sessions := []session.Session{
		{SessionID: "s1"},
		{SessionID: "s2"},
		{SessionID: "s3"},
	}

	tests := []struct {
		name           string
		filtered       []session.Session
		sessionID      string
		cursorBefore   int
		expectedCursor int
	}{
		{
			name:           "session found sets cursor to its index",
			filtered:       sessions,
			sessionID:      "s3",
			cursorBefore:   0,
			expectedCursor: 2,
		},
		{
			name:           "session not found clamps cursor",
			filtered:       sessions,
			sessionID:      "missing",
			cursorBefore:   10,
			expectedCursor: 2,
		},
		{
			name:           "empty sessionID clamps cursor",
			filtered:       sessions,
			sessionID:      "",
			cursorBefore:   10,
			expectedCursor: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				filtered: tt.filtered,
				cursor:   tt.cursorBefore,
			}
			m.restoreCursorBySessionID(tt.sessionID)
			if m.cursor != tt.expectedCursor {
				t.Errorf("expected cursor %d, got %d", tt.expectedCursor, m.cursor)
			}
		})
	}
}

func TestRenderStatusIndicator(t *testing.T) {
	tests := []struct {
		name             string
		status           session.Status
		isSelected       bool
		expectedContains string
	}{
		{
			name:             "StatusBusy contains spinner character",
			status:           session.StatusBusy,
			isSelected:       false,
			expectedContains: "✻",
		},
		{
			name:             "StatusWaiting contains question mark",
			status:           session.StatusWaiting,
			isSelected:       false,
			expectedContains: "?",
		},
		{
			name:             "StatusIdle contains bullet",
			status:           session.StatusIdle,
			isSelected:       false,
			expectedContains: "●",
		},
		{
			name:             "StatusUnknown contains bullet",
			status:           session.StatusUnknown,
			isSelected:       false,
			expectedContains: "●",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{spinnerFrame: 0}
			result := m.renderStatusIndicator(tt.status, tt.isSelected)

			if !strings.Contains(result, tt.expectedContains) {
				t.Errorf("expected result to contain %q, got %q", tt.expectedContains, result)
			}

			// All status indicators should end with a trailing space
			if !strings.HasSuffix(result, " ") {
				t.Errorf("expected trailing space, got %q", result)
			}
		})
	}
}

