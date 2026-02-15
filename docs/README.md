# claude-tmux

[![CI](https://github.com/Jevs21/claude-tmux/actions/workflows/ci.yml/badge.svg)](https://github.com/Jevs21/claude-tmux/actions/workflows/ci.yml)

A TUI tool for monitoring and jumping between running Claude Code sessions in tmux.

![Go](https://img.shields.io/badge/Go-1.24-blue) ![License](https://img.shields.io/badge/license-MIT-green)

## How It Works

claude-tmux uses Claude Code [hooks](https://docs.anthropic.com/en/docs/claude-code/hooks) to track session state. A hook script appends structured JSON events to `~/.claude-tmux/events.log` whenever Claude Code starts, stops, uses tools, or requests permission. The TUI reads this log on a 750ms tick to display live session status.

```
Claude Code hook fires → bash script appends JSON line → ~/.claude-tmux/events.log
                                                                  ↑
                                                    TUI reads on 750ms tick
```

## Prerequisites

- **Go 1.24+** — to build from source
- **tmux** — required for session switching
- **jq** — required by the hook script to parse JSON payloads

## Installation

```bash
git clone https://github.com/Jevs21/claude-tmux.git
cd claude-tmux
make build
```

This produces a `claude-tmux` binary in the project root. Move it somewhere on your `$PATH`:

```bash
cp claude-tmux ~/.local/bin/   # or /usr/local/bin/
```

## Hook Setup

The hook script (`hooks/claude-tmux-hook.sh`) must be registered with Claude Code. Add the following to your `~/.claude/settings.json` (adjust the path to where you cloned the repo):

```jsonc
{
  "hooks": {
    "SessionStart": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh session-start" }] }],
    "SessionEnd": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh session-end" }] }],
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh user-prompt-submit" }] }],
    "Stop": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh stop" }] }],
    "PreToolUse": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh pre-tool-use" }] }],
    "PostToolUse": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh post-tool-use" }] }],
    "PostToolUseFailure": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh post-tool-use-failure" }] }],
    "PermissionRequest": [{ "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh permission-request" }] }],
    "Notification": [
      { "matcher": "idle_prompt", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh notification-idle" }] },
      { "matcher": "permission_prompt", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh notification-permission" }] },
      { "matcher": "elicitation_dialog", "hooks": [{ "type": "command", "command": "/path/to/claude-tmux/hooks/claude-tmux-hook.sh notification-elicitation" }] }
    ]
  }
}
```

You can also run `make install-hook` to print this config with the correct absolute path filled in.

> **Note:** If you already have hooks configured, merge the entries above into your existing `hooks` object.

## tmux Keybinding

Add a keybinding to your `~/.tmux.conf` to launch claude-tmux as a popup:

```tmux
bind-key a display-popup -E "claude-tmux"
```

Then reload tmux config: `tmux source-file ~/.tmux.conf`

## Usage

Launch directly or via your tmux keybinding:

```bash
claude-tmux            # launch the TUI
claude-tmux --version  # show version
```

### Keybindings

| Key       | Mode   | Action                     |
|-----------|--------|----------------------------|
| `j` / `k` | Normal | Navigate down / up          |
| `G` / `g` | Normal | Jump to bottom / top        |
| `Enter`   | Normal | Jump to selected session    |
| `/`       | Normal | Enter filter mode           |
| `q` / `Esc` | Normal | Quit                     |
| `Enter`   | Filter | Apply filter and jump       |
| `Esc`     | Filter | Clear filter, return to list|
| `Ctrl+C`  | Any    | Force quit                  |

### Session Status Indicators

| Indicator | Status  | Meaning                          |
|-----------|---------|----------------------------------|
| `●` (green)  | Idle    | Session waiting for user input |
| `◐` (yellow) | Busy    | Claude is working (with spinner) |
| `?` (blue)   | Waiting | Permission or input needed     |
| `○` (dim)    | Unknown | Session detached or unresponsive |
