#!/usr/bin/env bash
# claude-tmux-hook.sh — Claude Code hook that colors tmux tabs based on
# session state and appends structured events to ~/.claude-tmux/events.log.
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

# Apply tmux tab coloring based on session state.
# Sets @claude-state as a user option on the window so users can build custom
# format strings with #{@claude-state} conditionals if they prefer full control.
# Saves the user's original format strings on first color change, then prepends
# a #[fg=...,bg=...] directive to preserve content while adding color.
# On session-end: restores original formats and cleans up all user options.
if [ -n "${TMUX_TARGET}" ]; then
    WINDOW_TARGET="${TMUX_TARGET%.*}"

    # Save the user's original window-level format overrides on first color
    # change. Uses @claude-saved-fmt / @claude-saved-curr-fmt user options.
    # The sentinel __global__ means the window had no local override.
    if [ -z "$(tmux show-window-option -t "${WINDOW_TARGET}" -v @claude-saved-fmt 2>/dev/null)" ]; then
        orig_fmt="$(tmux show-window-option -t "${WINDOW_TARGET}" -v window-status-format 2>/dev/null)" || true
        orig_curr_fmt="$(tmux show-window-option -t "${WINDOW_TARGET}" -v window-status-current-format 2>/dev/null)" || true
        tmux set-window-option -t "${WINDOW_TARGET}" @claude-saved-fmt "${orig_fmt:-__global__}" 2>/dev/null || true
        tmux set-window-option -t "${WINDOW_TARGET}" @claude-saved-curr-fmt "${orig_curr_fmt:-__global__}" 2>/dev/null || true
    fi

    # Helper: prepend a color directive to the user's format string.
    # Reads the saved original (or global fallback) and prefixes it with
    # #[fg=...,bg=...] so the color is visible without replacing content.
    set_tab_color() {
        local tab_bg="$1" tab_fg="$2"
        local base_fmt base_curr_fmt saved

        saved="$(tmux show-window-option -t "${WINDOW_TARGET}" -v @claude-saved-fmt 2>/dev/null)"
        if [ "${saved}" = "__global__" ]; then
            base_fmt="$(tmux show-option -gv window-status-format 2>/dev/null)"
        else
            base_fmt="${saved}"
        fi

        saved="$(tmux show-window-option -t "${WINDOW_TARGET}" -v @claude-saved-curr-fmt 2>/dev/null)"
        if [ "${saved}" = "__global__" ]; then
            base_curr_fmt="$(tmux show-option -gv window-status-current-format 2>/dev/null)"
        else
            base_curr_fmt="${saved}"
        fi

        tmux set-window-option -t "${WINDOW_TARGET}" window-status-format \
            "#[fg=${tab_fg},bg=${tab_bg}]${base_fmt}" 2>/dev/null || true
        tmux set-window-option -t "${WINDOW_TARGET}" window-status-current-format \
            "#[fg=${tab_fg},bg=${tab_bg},bold]${base_curr_fmt}" 2>/dev/null || true
    }

    case "${EVENT_NAME}" in
        user-prompt-submit|pre-tool-use|post-tool-use|post-tool-use-failure)
            tmux set-window-option -t "${WINDOW_TARGET}" @claude-state "busy" 2>/dev/null || true
            set_tab_color "yellow" "black"
            ;;
        permission-request|notification-permission|notification-elicitation)
            tmux set-window-option -t "${WINDOW_TARGET}" @claude-state "waiting" 2>/dev/null || true
            set_tab_color "blue" "white"
            ;;
        stop|notification-idle|session-start)
            tmux set-window-option -t "${WINDOW_TARGET}" @claude-state "idle" 2>/dev/null || true
            set_tab_color "green" "black"
            ;;
        session-end)
            # Restore original format strings saved on first color change.
            saved_fmt="$(tmux show-window-option -t "${WINDOW_TARGET}" -v @claude-saved-fmt 2>/dev/null)" || true
            saved_curr_fmt="$(tmux show-window-option -t "${WINDOW_TARGET}" -v @claude-saved-curr-fmt 2>/dev/null)" || true

            if [ "${saved_fmt}" = "__global__" ] || [ -z "${saved_fmt}" ]; then
                tmux set-window-option -t "${WINDOW_TARGET}" -u window-status-format 2>/dev/null || true
            else
                tmux set-window-option -t "${WINDOW_TARGET}" window-status-format "${saved_fmt}" 2>/dev/null || true
            fi
            if [ "${saved_curr_fmt}" = "__global__" ] || [ -z "${saved_curr_fmt}" ]; then
                tmux set-window-option -t "${WINDOW_TARGET}" -u window-status-current-format 2>/dev/null || true
            else
                tmux set-window-option -t "${WINDOW_TARGET}" window-status-current-format "${saved_curr_fmt}" 2>/dev/null || true
            fi

            # Clean up all user options.
            tmux set-window-option -t "${WINDOW_TARGET}" -u @claude-state 2>/dev/null || true
            tmux set-window-option -t "${WINDOW_TARGET}" -u @claude-saved-fmt 2>/dev/null || true
            tmux set-window-option -t "${WINDOW_TARGET}" -u @claude-saved-curr-fmt 2>/dev/null || true
            ;;
    esac
fi

# Rotate log if it exceeds 1000 lines to prevent unbounded growth.
if (( $(wc -l < "$LOG_FILE") > 1000 )); then
    tail -500 "$LOG_FILE" > "${LOG_FILE}.tmp" && mv "${LOG_FILE}.tmp" "$LOG_FILE"
fi
