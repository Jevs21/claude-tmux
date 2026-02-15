package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Jump switches to the given tmux pane target (e.g. "session:window.pane").
// It uses syscall.Exec to replace the current process so that the tmux popup
// closes cleanly after the switch.
func Jump(target string) error {
	tmuxBin, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	if insideTmux() {
		return syscall.Exec(tmuxBin, []string{"tmux", "switch-client", "-t", target}, os.Environ())
	}

	return syscall.Exec(tmuxBin, []string{"tmux", "attach-session", "-t", target}, os.Environ())
}

// insideTmux returns true if the current process is running inside a tmux session.
func insideTmux() bool {
	return os.Getenv("TMUX") != ""
}
