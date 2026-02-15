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

# Extract fields from payload via jq (single invocation)
IFS=$'\t' read -r SESSION_ID TOOL_NAME CWD < <(
    echo "${PAYLOAD}" | jq -r '[.session_id // "", .tool_name // "", .cwd // ""] | @tsv' 2>/dev/null || printf '\t\t'
)

# Capture the Claude process PID (our parent)
CLAUDE_PID="${PPID}"

# Capture tmux target if running inside tmux
TMUX_TARGET=""
if [ -n "${TMUX:-}" ]; then
    TMUX_TARGET="$(tmux display-message -t "$TMUX_PANE" -p '#{session_name}:#{window_index}.#{pane_index}' 2>/dev/null || true)"
fi

# Build and append the JSON event line (uses jq to safely handle special characters)
TIMESTAMP="$(date +%s)"
jq -n --argjson ts "$TIMESTAMP" --arg sid "$SESSION_ID" \
    --arg event "$EVENT_NAME" --argjson pid "$CLAUDE_PID" \
    --arg cwd "$CWD" --arg tmux "$TMUX_TARGET" --arg tool "$TOOL_NAME" \
    '{ts:$ts, sid:$sid, event:$event, pid:$pid, cwd:$cwd, tmux:$tmux, tool:$tool}' -c >> "${LOG_FILE}"

# Rotate log if it exceeds 1000 lines to prevent unbounded growth.
# The Go-side RotateLog() is a secondary safety net; this is the primary cap.
if (( $(wc -l < "$LOG_FILE") > 1000 )); then
    tail -500 "$LOG_FILE" > "${LOG_FILE}.tmp" && mv "${LOG_FILE}.tmp" "$LOG_FILE"
fi
