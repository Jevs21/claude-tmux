# claude-tmux

A Claude Code [hook](https://docs.anthropic.com/en/docs/claude-code/hooks) that exposes session state to tmux — so you can color tabs, add indicators, or build any status display you want.

![License](https://img.shields.io/badge/license-MIT-green)

## How It Works

The hook sets a `@claude-state` window option (`busy`, `waiting`, `idle`) on each Claude Code event. Your tmux config reads this option via `#{@claude-state}` to drive whatever visual treatment you prefer — background colors, status text, icons, etc.

On session end, the option is unset and the tab reverts to its default appearance.

### States

| State     | Meaning                              |
|-----------|--------------------------------------|
| `busy`    | Claude is working (tools, thinking)  |
| `waiting` | Permission or input needed           |
| `idle`    | Ready for your next prompt           |
| *(unset)* | No active Claude session in this tab |

## Prerequisites

- **tmux** — required (the hook sets tmux window options)

## Installation

### 1. Clone the repo

```bash
git clone https://github.com/Jevs21/claude-tmux.git
```

### 2. Configure the hook

Add the following to your `~/.claude/settings.json` (adjust the path to where you cloned the repo):

```jsonc
{
  "hooks": {
    "SessionStart": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh idle" }] }],
    "SessionEnd": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh reset" }] }],
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh busy" }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh idle" }] }],
    "PreToolUse": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh busy" }] }],
    "PostToolUse": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh busy" }] }],
    "PostToolUseFailure": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh busy" }] }],
    "PermissionRequest": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh waiting" }] }],
    "Notification": [
      { "matcher": "idle_prompt", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh idle" }] },
      { "matcher": "permission_prompt", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh waiting" }] },
      { "matcher": "elicitation_dialog", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh waiting" }] }
    ]
  }
}
```

> **Note:** If you already have hooks configured, merge the entries above into your existing `hooks` object.

### 3. Add to your tmux config

The hook only sets the `@claude-state` option — your `~/.tmux.conf` decides how to display it. See the [examples](#examples) below, then reload your config:

```bash
tmux source-file ~/.tmux.conf
```

## Examples

All examples use the `#{@claude-state}` window option. The key building blocks:

- `#{@claude-state}` — the raw state string (`busy`, `waiting`, `idle`, or empty)
- `#{?#{@claude-state},<if-set>,<if-unset>}` — conditional: is there an active Claude session?
- `#{==:#{@claude-state},busy}` — compare against a specific state

### Text label

The simplest option — append the state as text to tabs running a Claude session. Works with any theme since it doesn't touch colors or styling.

```tmux
set -g window-status-current-format ' #I:#W#{?#{@claude-state}, [#{@claude-state}],} '
set -g window-status-format ' #I:#W#{?#{@claude-state}, [#{@claude-state}],} '
```

Result: `0:zsh [busy]` during a session, `0:zsh` otherwise.

### Background color on active tab

Changes the active tab background per state. Tabs without a Claude session keep your default style.

```tmux
set -g window-status-current-format \
  '#{?#{==:#{@claude-state},busy},#[bg=yellow fg=black],#{?#{==:#{@claude-state},waiting},#[bg=blue fg=white],#{?#{==:#{@claude-state},idle},#[bg=green fg=black],}}} #I:#W '
```

### Background color on all tabs

Same as above but also colors inactive tabs, so you can see the state of every Claude session at a glance.

```tmux
set -g window-status-format \
  '#{?#{==:#{@claude-state},busy},#[bg=yellow fg=black],#{?#{==:#{@claude-state},waiting},#[bg=blue fg=white],#{?#{==:#{@claude-state},idle},#[bg=green fg=black],}} #I:#W '
set -g window-status-current-format \
  '#{?#{==:#{@claude-state},busy},#[bg=yellow fg=black bold],#{?#{==:#{@claude-state},waiting},#[bg=blue fg=white bold],#{?#{==:#{@claude-state},idle},#[bg=green fg=black bold],}}} #I:#W '
```

### Minimal dot indicator

Prepends a colored dot to tabs with an active session. Least intrusive — doesn't change your tab styling at all.

```tmux
set -g window-status-format \
  '#{?#{==:#{@claude-state},busy},#[fg=yellow]● ,#{?#{==:#{@claude-state},waiting},#[fg=blue]● ,#{?#{==:#{@claude-state},idle},#[fg=green]● ,}}}#[default]#I:#W '
set -g window-status-current-format \
  '#{?#{==:#{@claude-state},busy},#[fg=yellow]● ,#{?#{==:#{@claude-state},waiting},#[fg=blue]● ,#{?#{==:#{@claude-state},idle},#[fg=green]● ,}}}#[default]#I:#W '
```

## License

MIT
