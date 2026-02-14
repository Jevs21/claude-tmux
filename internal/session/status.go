package session

import (
	"os/exec"
	"strconv"
	"strings"
	"unicode"
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

// parseNumberedOption checks if a line represents a numbered option (e.g., "1. Yes"
// or "❯ 1. Yes"). It strips leading whitespace and an optional ❯ prefix, then looks
// for a pattern of digit(s) + period + space + text. Returns the option number and
// whether the line had a ❯ prefix.
func parseNumberedOption(line string) (optionNumber int, hasSelector bool, matched bool) {
	trimmedLine := strings.TrimLeftFunc(line, unicode.IsSpace)
	if trimmedLine == "" {
		return 0, false, false
	}

	// Check for and strip ❯ prefix
	if strings.HasPrefix(trimmedLine, "❯") {
		hasSelector = true
		trimmedLine = strings.TrimPrefix(trimmedLine, "❯")
		trimmedLine = strings.TrimLeftFunc(trimmedLine, unicode.IsSpace)
	}

	// Look for "N. text" pattern: digit(s), period, space, then text
	dotIndex := strings.Index(trimmedLine, ". ")
	if dotIndex < 1 {
		return 0, false, false
	}

	numberPart := trimmedLine[:dotIndex]
	parsedNumber, err := strconv.Atoi(numberPart)
	if err != nil {
		return 0, false, false
	}

	return parsedNumber, hasSelector, true
}

// detectNumberedOptions scans lines for an interactive numbered option menu.
// Returns true if at least options 1 and 2 are found AND at least one line has
// a ❯ selector prefix before its number.
func detectNumberedOptions(lines []string) bool {
	foundOptions := make(map[int]bool)
	hasSelectorOnOption := false

	for _, line := range lines {
		optionNumber, hasSelector, matched := parseNumberedOption(line)
		if !matched {
			continue
		}
		foundOptions[optionNumber] = true
		if hasSelector {
			hasSelectorOnOption = true
		}
	}

	return foundOptions[1] && foundOptions[2] && hasSelectorOnOption
}

// detectStatus examines pane content to determine if a Claude session is busy,
// waiting for input, or idle.
//
// A session is considered busy if a line starts with a spinner character followed
// by text ending with an ellipsis (…), which indicates an active spinner like
// "✻ Fiddle-faddling…". Completion messages like "✻ Worked for 2m 17s" do NOT
// end with ellipsis and are not treated as busy.
//
// A session is considered waiting if the pane contains an interactive numbered
// option menu (at least options 1 and 2 present, with a ❯ selector on one of them).
//
// A session is considered idle if the prompt character (❯) is visible.
//
// Priority: busy > waiting > idle > unknown.
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
	if detectNumberedOptions(lines) {
		return StatusWaiting
	}
	if hasPrompt {
		return StatusIdle
	}
	return StatusUnknown
}
