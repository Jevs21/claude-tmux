package tmux

import (
	"os"
	"testing"
)

func TestInsideTmux(t *testing.T) {
	// Save original value and restore it after the test
	originalTmuxEnv, originalWasSet := os.LookupEnv("TMUX")
	t.Cleanup(func() {
		if originalWasSet {
			os.Setenv("TMUX", originalTmuxEnv)
		} else {
			os.Unsetenv("TMUX")
		}
	})

	t.Run("TMUX env set returns true", func(t *testing.T) {
		os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
		if !insideTmux() {
			t.Error("expected insideTmux() to return true when TMUX is set")
		}
	})

	t.Run("TMUX empty returns false", func(t *testing.T) {
		os.Setenv("TMUX", "")
		if insideTmux() {
			t.Error("expected insideTmux() to return false when TMUX is empty")
		}
	})

	t.Run("TMUX unset returns false", func(t *testing.T) {
		os.Unsetenv("TMUX")
		if insideTmux() {
			t.Error("expected insideTmux() to return false when TMUX is unset")
		}
	})
}
