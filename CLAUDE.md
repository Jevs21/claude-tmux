# claude-tmux

A bash hook script that colors tmux tabs based on Claude Code session state.

## Tech Stack

- Bash
- [jq](https://jqlang.github.io/jq/) — JSON parsing in the hook script
- tmux — tab coloring via window-option overrides

## Project Structure

```
hooks/claude-tmux-hook.sh   # Hook script — the entire product
docs/README.md              # Installation and usage docs
LICENSE                     # MIT license
```

## Architecture

### Hook Script

`hooks/claude-tmux-hook.sh` is configured as a Claude Code hook. It receives the event name as `$1` and reads the JSON payload from stdin. It extracts `session_id`, `cwd`, and `tool_name` via `jq`, captures the Claude PID via `$PPID` and the tmux target via `tmux display-message`, then:

1. Appends a single JSON line to `~/.claude-tmux/events.log`
2. Sets tmux tab colors based on the event type

### Event Log Format

One JSON object per line in `~/.claude-tmux/events.log`:

```json
{"ts":1707900000,"sid":"d2abb274-...","event":"session-start","pid":12345,"cwd":"/path","tmux":"work:2.0","tool":""}
{"ts":1707900005,"sid":"d2abb274-...","event":"pre-tool-use","pid":12345,"cwd":"/path","tmux":"work:2.0","tool":"Bash"}
```

The log is rotated (truncated to 500 lines) when it exceeds 1000 lines.

### Tab Coloring

The hook sets tmux tab colors based on session state:

| Event | State | Tab Color |
|-------|-------|-----------|
| `user-prompt-submit`, `pre-tool-use`, `post-tool-use`, `post-tool-use-failure` | Busy | Yellow |
| `permission-request`, `notification-permission`, `notification-elicitation` | Waiting | Blue |
| `stop`, `notification-idle`, `session-start` | Idle | Green |
| `session-end` | — | Reset (unset all overrides) |

### `@claude-state` User Option

Each event sets a `@claude-state` window option (`busy`, `waiting`, or `idle`) so users can build custom tmux format strings using `#{@claude-state}` conditionals instead of relying on the built-in coloring.

### `STATUS_BG` Detection

The `set_tab_color` helper reads the global `status-bg` (or parses it from `status-style`) to construct Powerline-compatible triangle edges that blend with the user's status bar theme. Falls back to `terminal` if unset.

### `set_tab_color` Helper

`set_tab_color <bg> <fg>` sets both `window-status-format` (inactive) and `window-status-current-format` (active) on the window target:

- **Active tab**: Powerline arrow edges (``, ``) with bold text
- **Inactive tab**: Flat edges that blend with the status bar background

On `session-end`, all window-option overrides are unset so the tab reverts to global defaults.

## Commands

- `bash -n hooks/claude-tmux-hook.sh` — Syntax check the hook script

## Hook Installation

The hook script requires `jq`. Install the hook by adding the configuration to `~/.claude/settings.json` (see `docs/README.md` for the full config block).
