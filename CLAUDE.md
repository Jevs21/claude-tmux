# claude-tmux

A TUI tool for navigating between running Claude Code sessions in tmux.

## Tech Stack

- Go 1.24
- [Bubbletea](https://github.com/charmbracelet/bubbletea) v1 - TUI framework (Elm architecture)
- [Bubbles](https://github.com/charmbracelet/bubbles) - textinput component for filtering
- [Lipgloss](https://github.com/charmbracelet/lipgloss) v1 - terminal styling

## Project Structure

```
cmd/claude-tmux/main.go          # Thin entry point (package main)
internal/tui/model.go             # Bubbletea model & TUI logic (package tui)
internal/tui/styles.go            # Lipgloss style constants (package tui)
internal/session/session.go       # Session data type, sorting, path utils (package session)
internal/session/scanner.go       # Process discovery via ps (package session)
internal/session/tmux.go          # tmux pane mapping via process tree walk (package session)
internal/tmux/jump.go             # tmux switch/attach via syscall.Exec (package tmux)
docs/SPEC.md                      # Feature spec and build plan
```

## Architecture

### Session Discovery

`session.Scan()` runs `ps -axo pid,ppid,comm` to find processes where comm == "claude". Child processes (where parent is also claude) are filtered out. Working directories are resolved via `lsof -p <pid>` (macOS) or `/proc/<pid>/cwd` (Linux).

### tmux Mapping

`session.MapPanes()` runs `tmux list-panes -a` to get all pane PIDs, then walks each Claude process up the PID→PPID tree (max 25 hops) to find a matching tmux pane. Sessions without a pane match are shown as "detached".

### TUI Modes

Two modes via a state machine in `model.Update()`:

- **modeNormal** — Browse with j/k, enter to jump, / to filter, q to quit
- **modeFilter** — Text input filters sessions by substring match against project name, path, or tmux target

### Jump Mechanism

After TUI exits, if a session was selected, `tmux.Jump()` uses `syscall.Exec` to replace the process with `tmux switch-client -t <target>` (inside tmux) or `tmux attach-session -t <target>` (outside tmux). This ensures the tmux popup closes cleanly.

### Refresh

A 2-second tick re-scans processes and updates the list. Cursor position is preserved by matching on PID across refreshes.

## Commands

- `go build -o claude-tmux ./cmd/claude-tmux` - Build
- `go test ./...` - Run tests
- `./claude-tmux` - Run the TUI
- `./claude-tmux --version` - Show version

## tmux Integration

Add to `~/.tmux.conf`:
```
bind-key a display-popup -E "claude-tmux"
```

## Keybindings

| Key | Mode | Action |
|-----|------|--------|
| j/k | Normal | Navigate up/down |
| G/g | Normal | Jump to bottom/top |
| enter | Normal | Jump to selected session |
| / | Normal | Enter filter mode |
| q/esc | Normal | Quit |
| enter | Filter | Apply filter and jump |
| esc | Filter | Clear filter |
| ctrl+c | Any | Force quit |
