# claude-tmux

A bash hook script that exposes Claude Code session state to tmux via a window option.

## Tech Stack

- Bash
- tmux — `@claude-state` window option

## Project Structure

```
claude-tmux-hook.sh   # Hook script — the entire product
README.md             # Installation and usage docs
LICENSE               # MIT license
```

## Architecture

`claude-tmux-hook.sh` is configured as a Claude Code hook. It receives the target state as `$1` (`busy`, `waiting`, `idle`, or `reset`), drains stdin (Claude Code pipes a JSON payload regardless), and sets or unsets a `@claude-state` tmux window option. The event-to-state mapping is handled in the `~/.claude/settings.json` hooks config, not in the script itself.

### `@claude-state` User Option

The hook sets `@claude-state` on the current tmux window. Users reference `#{@claude-state}` in their `tmux.conf` format strings to build whatever visual treatment they prefer (background colors, text indicators, icons, etc.). On `reset`, the option is unset and the tab reverts to its default appearance.

### States

| State     | Meaning                              |
|-----------|--------------------------------------|
| `busy`    | Claude is working (tools, thinking)  |
| `waiting` | Permission or input needed           |
| `idle`    | Ready for next prompt                |
| `reset`   | Unsets `@claude-state` (session end) |

## Commands

- `bash -n claude-tmux-hook.sh` — Syntax check the hook script
