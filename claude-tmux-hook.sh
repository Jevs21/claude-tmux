#!/usr/bin/env bash
# claude-tmux-hook.sh — Claude Code hook that colors tmux tabs based on
# session state.
#
# Usage: Configured in ~/.claude/settings.json as a hook command.
# Receives event name as $1.

set -euo pipefail

EVENT_NAME="${1:-unknown}"

# Drain stdin to avoid broken pipe (Claude Code pipes the payload regardless)
cat > /dev/null

# Capture tmux target if running inside tmux
TMUX_TARGET=""
if [ -n "${TMUX:-}" ]; then
    TMUX_TARGET="$(tmux display-message -t "$TMUX_PANE" -p '#{session_name}:#{window_index}.#{pane_index}' 2>/dev/null || true)"
fi

# Apply tmux tab coloring based on session state.
# Sets @claude-state as a user option on the window so users can build custom
# format strings with #{@claude-state} conditionals if they prefer full control.
# Also sets window-local format overrides using Powerline-compatible styling,
# reading the status-bg color to construct proper triangle edges.
# On session-end: unsets all overrides so the tab reverts to global defaults.
if [ -n "${TMUX_TARGET}" ]; then
    WINDOW_TARGET="${TMUX_TARGET%.*}"

    # Resolve status bar background color for Powerline triangle edges.
    # Try status-bg first, then parse it from status-style.
    STATUS_BG=$(tmux show-option -gqv status-bg 2>/dev/null)
    if [ -z "${STATUS_BG}" ] || [ "${STATUS_BG}" = "default" ]; then
        STATUS_STYLE=$(tmux show-option -gqv status-style 2>/dev/null)
        STATUS_BG=$(printf '%s' "${STATUS_STYLE}" | sed -n 's/.*bg=\([^ ,]*\).*/\1/p')
    fi
    : "${STATUS_BG:=terminal}"

    # Helper: set indicator tab formats for a given color.
    # Active tab (window-status-current-format) gets visible Powerline arrow edges.
    # Inactive tabs (window-status-format) get flat edges that blend with the
    # status bar, matching the convention used by most Powerline themes.
    set_tab_color() {
        local tab_bg="$1" tab_fg="$2"
        tmux set-window-option -t "${WINDOW_TARGET}" window-status-format \
            "#[fg=${STATUS_BG},bg=${STATUS_BG}]#[fg=${tab_fg},bg=${tab_bg}] #I:#W #[fg=${STATUS_BG},bg=${STATUS_BG}]" 2>/dev/null || true
        tmux set-window-option -t "${WINDOW_TARGET}" window-status-current-format \
            "#[fg=${tab_bg},bg=${STATUS_BG}]#[fg=${tab_fg},bg=${tab_bg},bold] #I:#W #[fg=${tab_bg},bg=${STATUS_BG}]" 2>/dev/null || true
    }

    case "${EVENT_NAME}" in
        user-prompt-submit|pre-tool-use|post-tool-use|post-tool-use-failure)
            tmux set-window-option -t "${WINDOW_TARGET}" @claude-state "busy" 2>/dev/null || true
            set_tab_color "yellow" "black"
            ;;
        permission-request|notification-permission|notification-elicitation)
            tmux set-window-option -t "${WINDOW_TARGET}" @claude-state "waiting" 2>/dev/null || true
            set_tab_color "blue" "black"
            ;;
        stop|notification-idle|session-start)
            tmux set-window-option -t "${WINDOW_TARGET}" @claude-state "idle" 2>/dev/null || true
            set_tab_color "green" "black"
            ;;
        session-end)
            # Fully remove all overrides when the session ends.
            tmux set-window-option -t "${WINDOW_TARGET}" -u @claude-state 2>/dev/null || true
            tmux set-window-option -t "${WINDOW_TARGET}" -u window-status-format 2>/dev/null || true
            tmux set-window-option -t "${WINDOW_TARGET}" -u window-status-current-format 2>/dev/null || true
            ;;
    esac
fi
