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
