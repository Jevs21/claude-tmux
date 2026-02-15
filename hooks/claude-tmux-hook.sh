#!/usr/bin/env bash
# claude-tmux-hook.sh — Claude Code hook that appends structured events
# to ~/.claude-tmux/events.log for the claude-tmux TUI to read.
#
# Usage: Configured in ~/.claude/settings.json as a hook command.
# Receives event name as $1, reads JSON payload from stdin.
# Requires: jq

set -euo pipefail

EVENT_NAME="${1:-unknown}"
LOG_DIR="${HOME}/.claude-tmux"
LOG_FILE="${LOG_DIR}/events.log"

# Ensure log directory exists
mkdir -p "${LOG_DIR}"

# Read JSON payload from stdin (Claude Code pipes it)
PAYLOAD="$(cat)"

# Extract fields from payload via jq
SESSION_ID="$(echo "${PAYLOAD}" | jq -r '.session_id // empty' 2>/dev/null || true)"
TOOL_NAME="$(echo "${PAYLOAD}" | jq -r '.tool_name // empty' 2>/dev/null || true)"
CWD="$(echo "${PAYLOAD}" | jq -r '.cwd // empty' 2>/dev/null || true)"

# Capture the Claude process PID (our parent)
CLAUDE_PID="${PPID}"

# Capture tmux target if running inside tmux
TMUX_TARGET=""
if [ -n "${TMUX:-}" ]; then
    TMUX_TARGET="$(tmux display-message -t "$TMUX_PANE" -p '#{session_name}:#{window_index}.#{pane_index}' 2>/dev/null || true)"
fi

# Build and append the JSON event line (atomic single-line append)
TIMESTAMP="$(date +%s)"
echo "{\"ts\":${TIMESTAMP},\"sid\":\"${SESSION_ID}\",\"event\":\"${EVENT_NAME}\",\"pid\":${CLAUDE_PID},\"cwd\":\"${CWD}\",\"tmux\":\"${TMUX_TARGET}\",\"tool\":\"${TOOL_NAME}\"}" >> "${LOG_FILE}"
