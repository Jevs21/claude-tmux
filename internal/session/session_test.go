package session

import (
	"os"
	"testing"
)

func TestShortenPath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory replaced",
			input:    homeDir + "/projects/personal",
			expected: "~/projects/personal",
		},
		{
			name:     "short path unchanged",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "long path truncated",
			input:    "/Users/someone/projects/personal/deep/nested/path",
			expected: "/Users/.../nested/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenPath(tt.input)
			if result != tt.expected {
				t.Errorf("ShortenPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSortSessions(t *testing.T) {
	sessions := []Session{
		{SessionID: "aaa", TmuxSession: "work", TmuxTarget: "work:2.0", WindowIndex: 2},
		{SessionID: "bbb", TmuxTarget: ""}, // detached
		{SessionID: "ccc", TmuxSession: "dev", TmuxTarget: "dev:0.0", WindowIndex: 0},
		{SessionID: "ddd", TmuxSession: "work", TmuxTarget: "work:0.0", WindowIndex: 0},
	}

	SortSessions(sessions)

	// Expected order: dev:0, work:0, work:2, detached
	expectedSessionIDs := []string{"ccc", "ddd", "aaa", "bbb"}
	for i, expectedSessionID := range expectedSessionIDs {
		if sessions[i].SessionID != expectedSessionID {
			t.Errorf("position %d: expected SessionID %s, got %s", i, expectedSessionID, sessions[i].SessionID)
		}
	}
}

func TestSessionJumpable(t *testing.T) {
	jumpable := Session{TmuxTarget: "work:0.0"}
	if !jumpable.Jumpable() {
		t.Error("expected session with TmuxTarget to be jumpable")
	}

	detached := Session{TmuxTarget: ""}
	if detached.Jumpable() {
		t.Error("expected session without TmuxTarget to not be jumpable")
	}
}

func TestSessionDisplayTarget(t *testing.T) {
	attached := Session{TmuxSession: "work", TmuxTarget: "work:2.0", WindowIndex: 2}
	if attached.DisplayTarget() != "work:2" {
		t.Errorf("expected 'work:2', got %q", attached.DisplayTarget())
	}

	detached := Session{}
	if detached.DisplayTarget() != "detached" {
		t.Errorf("expected 'detached', got %q", detached.DisplayTarget())
	}
}

func TestShortenPath_EdgeCases(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory exactly",
			input:    homeDir,
			expected: "~",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "single component",
			input:    "/usr",
			expected: "/usr",
		},
		{
			name:     "exactly 4 non-empty components no truncation",
			input:    "/usr/local/share/man",
			expected: "/usr/local/share/man",
		},
		{
			name:     "5 components triggers truncation",
			input:    "/usr/local/share/man/man1",
			expected: "/usr/.../man/man1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenPath(tt.input)
			if result != tt.expected {
				t.Errorf("ShortenPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSortSessions_EdgeCases(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		var sessions []Session
		SortSessions(sessions)
		if len(sessions) != 0 {
			t.Errorf("expected empty slice, got %d sessions", len(sessions))
		}
	})

	t.Run("single session", func(t *testing.T) {
		sessions := []Session{
			{SessionID: "only", TmuxSession: "work", TmuxTarget: "work:0.0", WindowIndex: 0},
		}
		SortSessions(sessions)
		if sessions[0].SessionID != "only" {
			t.Errorf("expected 'only', got %q", sessions[0].SessionID)
		}
	})

	t.Run("all detached", func(t *testing.T) {
		sessions := []Session{
			{SessionID: "a", TmuxTarget: ""},
			{SessionID: "b", TmuxTarget: ""},
			{SessionID: "c", TmuxTarget: ""},
		}
		SortSessions(sessions)
		// All detached — should not panic, all remain
		if len(sessions) != 3 {
			t.Errorf("expected 3 sessions, got %d", len(sessions))
		}
	})

	t.Run("same tmux session different windows", func(t *testing.T) {
		sessions := []Session{
			{SessionID: "w3", TmuxSession: "dev", TmuxTarget: "dev:3.0", WindowIndex: 3},
			{SessionID: "w1", TmuxSession: "dev", TmuxTarget: "dev:1.0", WindowIndex: 1},
			{SessionID: "w2", TmuxSession: "dev", TmuxTarget: "dev:2.0", WindowIndex: 2},
		}
		SortSessions(sessions)
		expectedOrder := []string{"w1", "w2", "w3"}
		for i, expectedID := range expectedOrder {
			if sessions[i].SessionID != expectedID {
				t.Errorf("position %d: expected %q, got %q", i, expectedID, sessions[i].SessionID)
			}
		}
	})

	t.Run("alphabetical session names", func(t *testing.T) {
		sessions := []Session{
			{SessionID: "s3", TmuxSession: "work", TmuxTarget: "work:0.0", WindowIndex: 0},
			{SessionID: "s1", TmuxSession: "alpha", TmuxTarget: "alpha:0.0", WindowIndex: 0},
			{SessionID: "s2", TmuxSession: "dev", TmuxTarget: "dev:0.0", WindowIndex: 0},
		}
		SortSessions(sessions)
		expectedOrder := []string{"s1", "s2", "s3"}
		for i, expectedID := range expectedOrder {
			if sessions[i].SessionID != expectedID {
				t.Errorf("position %d: expected %q, got %q", i, expectedID, sessions[i].SessionID)
			}
		}
	})
}
