package session

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Scan discovers all running Claude Code processes and returns them as Sessions.
// It filters out child processes (where the parent is also a Claude process).
func Scan() ([]Session, error) {
	psOutput, err := runPS()
	if err != nil {
		return nil, fmt.Errorf("failed to run ps: %w", err)
	}

	sessions := parseProcesses(psOutput)

	// Resolve working directories
	for i := range sessions {
		workDir := resolveWorkDir(sessions[i].PID)
		if workDir != "" {
			sessions[i].WorkDir = workDir
			sessions[i].ProjectName = filepath.Base(workDir)
		}
	}

	return sessions, nil
}

// runPS executes ps and returns the raw output.
func runPS() (string, error) {
	cmd := exec.Command("ps", "-axo", "pid,ppid,comm")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// parseProcesses parses ps output and returns Claude sessions with children filtered out.
func parseProcesses(psOutput string) []Session {
	type processEntry struct {
		pid  int
		ppid int
	}

	var claudeProcesses []processEntry
	claudePIDs := make(map[int]bool)

	scanner := bufio.NewScanner(strings.NewReader(psOutput))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
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
		comm := fields[2]

		// Match processes where the command name is "claude"
		commBase := filepath.Base(comm)
		if commBase == "claude" {
			claudeProcesses = append(claudeProcesses, processEntry{pid: pid, ppid: ppid})
			claudePIDs[pid] = true
		}
	}

	// Filter out children: if a claude process's parent is also claude, skip it
	var sessions []Session
	for _, proc := range claudeProcesses {
		if claudePIDs[proc.ppid] {
			continue
		}
		sessions = append(sessions, Session{
			PID:  proc.pid,
			PPID: proc.ppid,
		})
	}

	return sessions
}

// resolveWorkDir attempts to find the working directory of a process.
// Uses lsof on macOS, falls back to /proc on Linux.
func resolveWorkDir(pid int) string {
	// Try lsof first (works on macOS and Linux)
	workDir := resolveWorkDirLsof(pid)
	if workDir != "" {
		return workDir
	}

	// Fallback: try /proc (Linux)
	return resolveWorkDirProc(pid)
}

// resolveWorkDirLsof uses lsof to find the current working directory.
func resolveWorkDirLsof(pid int) string {
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-Fn", "-a", "-d", "cwd")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// lsof -Fn output format: lines starting with 'n' contain the name (path)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "n") && len(line) > 1 {
			return line[1:]
		}
	}
	return ""
}

// resolveWorkDirProc reads the /proc filesystem for the working directory (Linux).
func resolveWorkDirProc(pid int) string {
	procPath := fmt.Sprintf("/proc/%d/cwd", pid)
	target, err := filepath.EvalSymlinks(procPath)
	if err != nil {
		return ""
	}
	return target
}
