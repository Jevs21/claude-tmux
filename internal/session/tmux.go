package session

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// paneInfo holds the tmux pane metadata for a single pane.
type paneInfo struct {
	SessionName string
	WindowIndex int
	PaneIndex   int
}

// MapPanes maps each Session to its tmux pane by walking the process tree.
// It queries tmux for all panes, then for each session walks up the PID→PPID
// chain to find a matching tmux pane PID.
func MapPanes(sessions []Session) []Session {
	paneMap, err := getTmuxPanes()
	if err != nil {
		// tmux not running or not available — return sessions as-is (detached)
		return sessions
	}

	processTree, err := getProcessTree()
	if err != nil {
		return sessions
	}

	for i := range sessions {
		info, found := findPaneForPID(sessions[i].PID, paneMap, processTree)
		if found {
			sessions[i].TmuxSession = info.SessionName
			sessions[i].WindowIndex = info.WindowIndex
			sessions[i].PaneIndex = info.PaneIndex
			sessions[i].TmuxTarget = fmt.Sprintf(
				"%s:%d.%d",
				info.SessionName,
				info.WindowIndex,
				info.PaneIndex,
			)
		}
	}

	return sessions
}

// getTmuxPanes queries tmux for all panes and returns a map of panePID → paneInfo.
func getTmuxPanes() (map[int]paneInfo, error) {
	cmd := exec.Command(
		"tmux", "list-panes", "-a",
		"-F", "#{pane_pid} #{session_name} #{window_index} #{pane_index}",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes failed: %w", err)
	}

	paneMap := make(map[int]paneInfo)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		panePID, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		windowIndex, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		paneIndex, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}

		paneMap[panePID] = paneInfo{
			SessionName: fields[1],
			WindowIndex: windowIndex,
			PaneIndex:   paneIndex,
		}
	}

	return paneMap, nil
}

// getProcessTree returns a map of PID → PPID for all processes.
func getProcessTree() (map[int]int, error) {
	cmd := exec.Command("ps", "-axo", "pid,ppid")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}

	tree := make(map[int]int)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		tree[pid] = ppid
	}

	return tree, nil
}

// findPaneForPID walks up the process tree from the given PID to find a tmux pane.
// Returns the pane info and true if found, or zero value and false if not.
func findPaneForPID(pid int, paneMap map[int]paneInfo, processTree map[int]int) (paneInfo, bool) {
	currentPID := pid
	const maxHops = 25

	for i := 0; i < maxHops; i++ {
		if info, found := paneMap[currentPID]; found {
			return info, true
		}

		parentPID, exists := processTree[currentPID]
		if !exists || parentPID <= 1 {
			break
		}
		currentPID = parentPID
	}

	return paneInfo{}, false
}
