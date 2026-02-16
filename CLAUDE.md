# claude-tmux

A bash hook script that colors tmux tabs based on Claude Code session state.

## Tech Stack

- Bash
- tmux — tab coloring via window-option overrides

## Project Structure

```
claude-tmux-hook.sh   # Hook script — the entire product
README.md             # Installation and usage docs
LICENSE               # MIT license
```

## Architecture

`claude-tmux-hook.sh` is configured as a Claude Code hook. It receives the event name as `$1`, drains stdin (Claude Code pipes a JSON payload regardless), captures the tmux target via `tmux display-message`, then sets tmux tab colors based on the event type.

### Tab Coloring

| Event | State | Tab Color |
|-------|-------|-----------|
| `user-prompt-submit`, `pre-tool-use`, `post-tool-use`, `post-tool-use-failure` | Busy | Yellow |
| `permission-request`, `notification-permission`, `notification-elicitation` | Waiting | Blue |
| `stop`, `notification-idle`, `session-start` | Idle | Green |
| `session-end` | — | Reset (unset all overrides) |

### `@claude-state` User Option

Each event sets a `@claude-state` window option (`busy`, `waiting`, or `idle`) so users can build custom tmux format strings using `#{@claude-state}` conditionals instead of relying on the built-in coloring.

### Powerline Edge Detection

The `set_tab_color` helper reads the global `status-bg` (or parses it from `status-style`) to construct Powerline-compatible triangle edges that blend with the user's status bar theme. Falls back to `terminal` if unset.

## Commands

- `bash -n claude-tmux-hook.sh` — Syntax check the hook script
