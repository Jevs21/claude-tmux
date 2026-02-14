package session

import "testing"

func TestDetectStatus_SpinnerWithEllipsis(t *testing.T) {
	paneContent := "✻ Fiddle-faddling…\n"
	status := detectStatus(paneContent)
	if status != StatusBusy {
		t.Errorf("expected StatusBusy, got %d", status)
	}
}

func TestDetectStatus_DifferentSpinnerChars(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{"star", "✽ Reading files…\n"},
		{"asterisk", "✳ Writing code…\n"},
		{"dot", "· Thinking…\n"},
		{"sixPointStar", "✶ Analyzing…\n"},
		{"cross", "✢ Processing…\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status := detectStatus(testCase.content)
			if status != StatusBusy {
				t.Errorf("expected StatusBusy for %q, got %d", testCase.content, status)
			}
		})
	}
}

func TestDetectStatus_PromptIdle(t *testing.T) {
	paneContent := "some output\n❯ \n"
	status := detectStatus(paneContent)
	if status != StatusIdle {
		t.Errorf("expected StatusIdle, got %d", status)
	}
}

func TestDetectStatus_CompletionMessageWithPrompt(t *testing.T) {
	// Completion message does NOT end with ellipsis, plus prompt is visible → idle
	paneContent := "✻ Worked for 2m 17s\n❯ \n"
	status := detectStatus(paneContent)
	if status != StatusIdle {
		t.Errorf("expected StatusIdle, got %d", status)
	}
}

func TestDetectStatus_EmptyContent(t *testing.T) {
	status := detectStatus("")
	if status != StatusUnknown {
		t.Errorf("expected StatusUnknown, got %d", status)
	}
}

func TestDetectStatus_WhitespaceOnly(t *testing.T) {
	status := detectStatus("   \n  \n\n")
	if status != StatusUnknown {
		t.Errorf("expected StatusUnknown, got %d", status)
	}
}

func TestDetectStatus_CompletionMessageWithoutPrompt(t *testing.T) {
	// Completion message without ellipsis and no prompt → unknown
	paneContent := "✻ Worked for 2m 17s\n"
	status := detectStatus(paneContent)
	if status != StatusUnknown {
		t.Errorf("expected StatusUnknown for completion message without prompt, got %d", status)
	}
}

func TestDetectStatus_SpinnerTakesPrecedenceOverPrompt(t *testing.T) {
	// If both spinner activity and prompt are visible, spinner wins (busy)
	paneContent := "❯ claude\n✻ Working on something…\n"
	status := detectStatus(paneContent)
	if status != StatusBusy {
		t.Errorf("expected StatusBusy when both spinner and prompt present, got %d", status)
	}
}

func TestDetectStatus_WaitingWithTwoOptions(t *testing.T) {
	paneContent := "Do you want to proceed?\n❯ 1. Yes\n  2. No\n"
	status := detectStatus(paneContent)
	if status != StatusWaiting {
		t.Errorf("expected StatusWaiting for yes/no prompt, got %d", status)
	}
}

func TestDetectStatus_WaitingWithThreeOptions(t *testing.T) {
	paneContent := "Allow access to /tmp/foo?\n❯ 1. Allow once\n  2. Allow always\n  3. Deny\n"
	status := detectStatus(paneContent)
	if status != StatusWaiting {
		t.Errorf("expected StatusWaiting for permission prompt, got %d", status)
	}
}

func TestDetectStatus_WaitingCursorOnSecondOption(t *testing.T) {
	paneContent := "Choose an option:\n  1. Option A\n❯ 2. Option B\n  3. Option C\n"
	status := detectStatus(paneContent)
	if status != StatusWaiting {
		t.Errorf("expected StatusWaiting when cursor is on second option, got %d", status)
	}
}

func TestDetectStatus_NotWaitingNumberedListWithoutSelector(t *testing.T) {
	// A markdown numbered list in Claude output with an idle ❯ prompt should be idle, not waiting.
	// The ❯ is on its own prompt line, not prefixing a numbered option.
	paneContent := "Here are some steps:\n1. First step\n2. Second step\n3. Third step\n❯ \n"
	status := detectStatus(paneContent)
	if status != StatusIdle {
		t.Errorf("expected StatusIdle for numbered list without selector on option, got %d", status)
	}
}

func TestDetectStatus_NotWaitingSingleOption(t *testing.T) {
	// Only one numbered option found — not enough to be an interactive menu
	paneContent := "❯ 1. Yes\n"
	status := detectStatus(paneContent)
	if status != StatusIdle {
		t.Errorf("expected StatusIdle for single option, got %d", status)
	}
}

func TestDetectStatus_WaitingTakesPrecedenceOverIdle(t *testing.T) {
	// Both ❯ prompt and numbered options are present — waiting wins over idle
	paneContent := "Some output\n❯ 1. Allow\n  2. Deny\n❯ \n"
	status := detectStatus(paneContent)
	if status != StatusWaiting {
		t.Errorf("expected StatusWaiting to take precedence over idle, got %d", status)
	}
}

func TestDetectStatus_WaitingRealClaudeCodePermissionPrompt(t *testing.T) {
	// Real Claude Code permission prompt format with tool description block
	paneContent := " Bash command\n\n" +
		"   git add CLAUDE.md internal/session/session.go internal/session/status.go internal/session/status_test.go\n" +
		"   internal/tui/model.go internal/tui/styles.go\n" +
		"   Stage all modified files\n\n" +
		" Do you want to proceed?\n" +
		" ❯ 1. Yes\n" +
		"   2. Yes, and don't ask again for git add commands in /Users/user/projects/claude-tmux\n" +
		"   3. No\n"
	status := detectStatus(paneContent)
	if status != StatusWaiting {
		t.Errorf("expected StatusWaiting for real Claude Code permission prompt, got %d", status)
	}
}

func TestDetectStatus_BusyTakesPrecedenceOverWaiting(t *testing.T) {
	// Spinner activity + numbered options — busy wins
	paneContent := "✻ Working…\n❯ 1. Allow\n  2. Deny\n"
	status := detectStatus(paneContent)
	if status != StatusBusy {
		t.Errorf("expected StatusBusy to take precedence over waiting, got %d", status)
	}
}
