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
hooks/claude-tmux-hook.sh        # Bash hook for Claude Code event capture
internal/tui/model.go             # Bubbletea model & TUI logic (package tui)
internal/tui/styles.go            # Lipgloss style constants (package tui)
internal/session/session.go       # Session data type, sorting, path utils (package session)
internal/session/events.go        # Event log reader & state derivation (package session)
internal/tmux/jump.go             # tmux switch/attach via syscall.Exec (package tmux)
docs/README.md                    # Project README with setup instructions
docs/SPEC.md                      # Feature spec and build plan
```

## Architecture

### Hooks-based Session Monitoring

Session state is derived from a structured event log written by a Claude Code hook script, rather than process scanning or pane capture.

```
Hook fires (Claude Code) -> bash script appends JSON line -> ~/.claude-tmux/events.log
                                                                    ^
                                              Go TUI reads file on 750ms tick
```

### Hook Script

`hooks/claude-tmux-hook.sh` is configured as a Claude Code hook. It receives the event name as `$1` and reads the JSON payload from stdin. It extracts `session_id`, `cwd`, and `tool_name` via `jq`, captures the Claude PID via `$PPID` and the tmux target via `tmux display-message`, then appends a single JSON line to `~/.claude-tmux/events.log`.

### Event Log Format

One JSON object per line in `~/.claude-tmux/events.log`:

```json
{"ts":1707900000,"sid":"d2abb274-...","event":"session-start","pid":12345,"cwd":"/path","tmux":"work:2.0","tool":""}
{"ts":1707900005,"sid":"d2abb274-...","event":"pre-tool-use","pid":12345,"cwd":"/path","tmux":"work:2.0","tool":"Bash"}
```

### State Derivation

`session.ReadSessions()` reads the event log and derives session states:

| Event | Status | Action |
|-------|--------|--------|
| `session-start` | Idle | -- |
| `user-prompt-submit` | Busy | Thinking... |
| `pre-tool-use` | Busy | {tool_name} |
| `post-tool-use` / `post-tool-use-failure` | Busy | -- |
| `stop` | Idle | -- |
| `permission-request` | Waiting | Permission |
| `notification-idle` | Idle | -- |
| `notification-permission` | Waiting | Permission |
| `notification-elicitation` | Waiting | Input |
| `session-end` | (remove session) | -- |

Dead sessions (where `ClaudePID` is no longer alive) are pruned automatically. The log is rotated on startup (truncated to 500 lines if exceeding 1000).

### TUI Modes

Two modes via a state machine in `model.Update()`:

- **modeNormal** -- Browse with j/k, enter to jump, / to filter, q to quit
- **modeFilter** -- Text input filters sessions by substring match against project name, path, or tmux target

### Jump Mechanism

After TUI exits, if a session was selected, `tmux.Jump()` uses `syscall.Exec` to replace the process with `tmux switch-client -t <target>` (inside tmux) or `tmux attach-session -t <target>` (outside tmux). This ensures the tmux popup closes cleanly.

### Session Status Display

Busy sessions display an animated yellow spinner in the TUI (150ms frame interval) with the current action (tool name or "Thinking...") shown in a dim italic column. Waiting sessions show a blue `?` with "Permission" or "Input". Idle sessions show a green dot. Unknown/detached sessions show a dim dot.

### Refresh

A 750ms tick re-reads the event log and updates the list. Cursor position is preserved by matching on SessionID across refreshes. A separate 150ms tick drives the spinner animation for busy sessions.

## Commands

- `go build -o claude-tmux ./cmd/claude-tmux` - Build
- `go test ./...` - Run tests
- `./claude-tmux` - Run the TUI
- `./claude-tmux --version` - Show version
- `make install-hook` - Show hook installation instructions

## tmux Integration

Add to `~/.tmux.conf`:
```
bind-key a display-popup -E "claude-tmux"
```

## Hook Installation

The hook script requires `jq`. Install the hook by adding the configuration to `~/.claude/settings.json` (see `make install-hook` for the full config block).

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
