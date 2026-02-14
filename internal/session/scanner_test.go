package session

import (
	"testing"
)

func TestParseProcesses(t *testing.T) {
	psOutput := `  PID  PPID COMM
    1     0 launchd
  100     1 zsh
  200   100 claude
  300   200 claude
  400     1 zsh
  500   400 claude
  600     1 vim
`

	sessions := parseProcesses(psOutput)

	// Should find 2 top-level claude processes (PID 200 and 500)
	// PID 300 is a child of PID 200 (both claude), so it should be filtered out
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	pids := make(map[int]bool)
	for _, s := range sessions {
		pids[s.PID] = true
	}

	if !pids[200] {
		t.Error("expected PID 200 to be included")
	}
	if !pids[500] {
		t.Error("expected PID 500 to be included")
	}
	if pids[300] {
		t.Error("expected PID 300 (child) to be filtered out")
	}
}

func TestParseProcessesNoClaude(t *testing.T) {
	psOutput := `  PID  PPID COMM
    1     0 launchd
  100     1 zsh
  200   100 vim
`

	sessions := parseProcesses(psOutput)
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestParseProcessesEmptyOutput(t *testing.T) {
	sessions := parseProcesses("")
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}
