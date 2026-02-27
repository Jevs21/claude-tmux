# claude-tmux

A Claude Code [hook](https://docs.anthropic.com/en/docs/claude-code/hooks) that exposes session state to tmux — so you can color tabs, add indicators, or build any status display you want.

<div style="display: flex; gap: 10px;">
  <img src="https://github.com/user-attachments/assets/5730548d-9aa3-4174-b98d-25a6655d2f93" width="48%"/>
  <img src="https://github.com/user-attachments/assets/12f4ad83-c9d6-44ca-b3d6-2bce31129708" width="48%"/>
</div>

## How It Works

The hook sets a `@claude-state` window option (`busy`, `waiting`, `idle`) on each Claude Code event. Your tmux config reads this option via `#{@claude-state}` to drive whatever visual treatment you prefer — background colors, status text, icons, etc.

On session end, the option is unset and the tab reverts to its default appearance.

### States

| State     | Meaning                              |
|-----------|--------------------------------------|
| `busy`    | Claude is working (tools, thinking)  |
| `waiting` | Permission or input needed           |
| `idle`    | Ready for your next prompt           |
| `reset`   | Unsets `@claude-state` (session end) |

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

Append the state as text to tabs running a Claude session. Works with any theme since it doesn't touch colors.

```tmux
set -g window-status-current-format ' #I:#W#{?#{@claude-state}, [#{@claude-state}],} '
set -g window-status-format ' #I:#W#{?#{@claude-state}, [#{@claude-state}],} '
```

Result: `0:zsh [busy]` during a session, `0:zsh` otherwise.

### Background color per state

Changes tab background per state. Tabs without a Claude session keep your default style.

```tmux
set -g window-status-format \
  '#{?#{==:#{@claude-state},busy},#[bg=yellow fg=black],#{?#{==:#{@claude-state},waiting},#[bg=blue fg=white],#{?#{==:#{@claude-state},idle},#[bg=green fg=black],}} #I:#W '
set -g window-status-current-format \
  '#{?#{==:#{@claude-state},busy},#[bg=yellow fg=black bold],#{?#{==:#{@claude-state},waiting},#[bg=blue fg=white bold],#{?#{==:#{@claude-state},idle},#[bg=green fg=black bold],}}} #I:#W '
```

### My config

A full working config with a transparent status bar and per-state coloring on both active and inactive tabs.

```tmux
set -g status-style bg=default
set -g status-bg default
set -g status-left-length 50
set -g status-left "#[fg=black,bg=colour245,bold]  %b %d, W%V ~ #H ~ #S  #[fg=default,bg=default]   "
set -g status-right ""

set -g window-status-style bg=default,fg=colour245
set -g window-status-format '#{?#{==:#{@claude-state},waiting},#[fg=colour196]#[bold],#{?#{==:#{@claude-state},busy},#[fg=colour208],#{?#{==:#{@claude-state},idle},#[fg=colour114],#[fg=colour245]}}}    #I:#W    '
set -g window-status-current-format '#[fg=black,bold]#{?#{==:#{@claude-state},busy},#[bg=colour208],#{?#{==:#{@claude-state},waiting},#[bg=colour196],#{?#{==:#{@claude-state},idle},#[bg=colour114],#[bg=colour141]}}}  » #I:#W «  '
```

## License

![License](https://img.shields.io/badge/license-MIT-green)
