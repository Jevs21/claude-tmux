package session

import (
	"os/exec"
	"strings"
)

// spinnerChars are the characters Claude Code uses for its activity spinner.
var spinnerChars = []rune{'✻', '✽', '✳', '·', '✶', '✢'}

// CaptureStatuses captures tmux pane content for each session and sets the
// Status field based on whether the session appears busy or idle.
func CaptureStatuses(sessions []Session) {
	for i := range sessions {
		if sessions[i].TmuxTarget == "" {
			sessions[i].Status = StatusUnknown
			continue
		}
		paneContent := capturePaneContent(sessions[i].TmuxTarget)
		sessions[i].Status = detectStatus(paneContent)
	}
}

// capturePaneContent runs tmux capture-pane to get the visible content of a pane.
func capturePaneContent(tmuxTarget string) string {
	cmd := exec.Command("tmux", "capture-pane", "-t", tmuxTarget, "-p")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}

// detectStatus examines pane content to determine if a Claude session is busy or idle.
//
// A session is considered busy if a line starts with a spinner character followed
// by text ending with an ellipsis (…), which indicates an active spinner like
// "✻ Fiddle-faddling…". Completion messages like "✻ Worked for 2m 17s" do NOT
// end with ellipsis and are not treated as busy.
//
// A session is considered idle if the prompt character (❯) is visible.
//
// Otherwise the status is unknown.
func detectStatus(paneContent string) Status {
	if paneContent == "" {
		return StatusUnknown
	}

	spinnerCharSet := make(map[rune]bool, len(spinnerChars))
	for _, ch := range spinnerChars {
		spinnerCharSet[ch] = true
	}

	lines := strings.Split(paneContent, "\n")

	hasSpinnerActivity := false
	hasPrompt := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// Check for active spinner: line starts with spinner char + space + text + ellipsis
		runes := []rune(trimmedLine)
		if len(runes) >= 3 && spinnerCharSet[runes[0]] && runes[1] == ' ' && runes[len(runes)-1] == '…' {
			hasSpinnerActivity = true
		}

		// Check for prompt character
		if strings.Contains(trimmedLine, "❯") {
			hasPrompt = true
		}
	}

	if hasSpinnerActivity {
		return StatusBusy
	}
	if hasPrompt {
		return StatusIdle
	}
	return StatusUnknown
}
