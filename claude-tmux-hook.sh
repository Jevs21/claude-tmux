#!/usr/bin/env bash
# claude-tmux-hook.sh — Claude Code hook that exposes session state via tmux.
#
# Usage: Configured in ~/.claude/settings.json as a hook command.
# Receives the target state as $1 (busy, waiting, idle, or reset).
# The hooks config maps Claude Code events to these states.

set -euo pipefail

STATE="${1:-}"

# Drain stdin to avoid broken pipe (Claude Code pipes the payload regardless)
cat > /dev/null

# Exit early if not in tmux or no state provided
if [ -z "${TMUX:-}" ] || [ -z "${STATE}" ]; then
    exit 0
fi

WINDOW_TARGET=$(tmux display-message -t "$TMUX_PANE" -p '#{session_name}:#{window_index}' 2>/dev/null) || exit 0

if [ "${STATE}" = "reset" ]; then
    tmux set-window-option -t "${WINDOW_TARGET}" -u @claude-state 2>/dev/null || true
else
    tmux set-window-option -t "${WINDOW_TARGET}" @claude-state "${STATE}" 2>/dev/null || true
fi
