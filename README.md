# claude-tmux

A Claude Code [hook](https://docs.anthropic.com/en/docs/claude-code/hooks) that colors your tmux tabs based on session state — yellow when Claude is working, blue when it needs permission or input, green when idle.

![License](https://img.shields.io/badge/license-MIT-green)

## How It Works

The hook script fires on Claude Code events and sets tmux window-option overrides to color the tab for that pane. It also sets a `@claude-state` user option so you can build custom tmux format strings if you prefer full control.

### Tab Colors

| Color  | State   | Meaning                           |
|--------|---------|-----------------------------------|
| Yellow | Busy    | Claude is working (tools, thinking) |
| Blue   | Waiting | Permission or input needed        |
| Green  | Idle    | Session waiting for user input    |
| Reset  | Ended   | Tab reverts to global defaults    |

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
    "SessionStart": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh session-start" }] }],
    "SessionEnd": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh session-end" }] }],
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh user-prompt-submit" }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh stop" }] }],
    "PreToolUse": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh pre-tool-use" }] }],
    "PostToolUse": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh post-tool-use" }] }],
    "PostToolUseFailure": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh post-tool-use-failure" }] }],
    "PermissionRequest": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh permission-request" }] }],
    "Notification": [
      { "matcher": "idle_prompt", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh notification-idle" }] },
      { "matcher": "permission_prompt", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh notification-permission" }] },
      { "matcher": "elicitation_dialog", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/claude-tmux-hook.sh notification-elicitation" }] }
    ]
  }
}
```

> **Note:** If you already have hooks configured, merge the entries above into your existing `hooks` object.

## Custom Theming with `@claude-state`

The hook sets a `@claude-state` window option (`busy`, `waiting`, or `idle`) on each event. You can use this in your own tmux status format strings instead of relying on the built-in tab coloring:

```tmux
# Example: show state text in status bar
set -g window-status-format '#I:#W #{?#{==:#{@claude-state},busy},⚡,#{?#{==:#{@claude-state},waiting},❓,}}'
```

## Powerline Compatibility

The hook auto-detects your `status-bg` color (or parses it from `status-style`) to construct Powerline-compatible triangle edges that blend with your status bar theme.

## License

MIT
