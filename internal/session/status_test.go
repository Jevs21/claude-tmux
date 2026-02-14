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
