# claude-tmux — Specification & Build Plan

A Go TUI tool (using Bubbletea) that runs as a tmux popup, listing active Claude Code sessions and letting you jump between them.

---

## Core Functionality

Find all running Claude Code sessions via a hooks-based event log, display them in a navigable list with live status, and switch to the selected pane on enter.

---

## Feature List

### F1 — Hooks-based Session Discovery

Detect running Claude Code sessions via structured event logging.

- A bash hook script (`hooks/claude-tmux-hook.sh`) is configured in `~/.claude/settings.json` for all Claude Code lifecycle events.
- The hook receives the event name as `$1`, reads the JSON payload from stdin, extracts `session_id`, `cwd`, and `tool_name` via `jq`.
- It captures the Claude PID via `$PPID` and the tmux target via `tmux display-message` (if `$TMUX` is set).
- Each event is appended as a single JSON line to `~/.claude-tmux/events.log`.

### F2 — Event Log Reading & State Derivation

Build session state from the event stream.

- `session.ReadSessions()` opens `~/.claude-tmux/events.log` and scans line by line.
- Each line is parsed as JSON into a `RawEvent` struct.
- Events are applied to a `map[string]*Session` keyed by session ID:
  - `session-start` → create session entry
  - `session-end` → delete session entry
  - Other events → update Status and Action per the mapping table
- Dead sessions (where `ClaudePID` is no longer alive) are pruned via `syscall.Kill(pid, 0)`.
- The log is rotated on startup: truncated to last 500 lines if exceeding 1000.

### F3 — Session List Model

Each session entry contains:
- **SessionID**: the Claude Code session UUID
- **ClaudePID**: the Claude process PID (for liveness checks)
- **Working directory**: shortened path (`~` for home, truncate long paths)
- **tmux target**: `session:window.pane` string (empty if detached)
- **Project name**: last path component of the working directory
- **Status**: busy, idle, waiting, or unknown
- **Action**: current activity (tool name, "Thinking…", "Permission", "Input")

Sessions are sorted alphabetically by tmux session name, then by window index.

### F4 — TUI (Bubbletea)

Interactive terminal list with keyboard navigation.

**Display per session (single line):**
```
  claude-tmux   work:2   ~/projects/personal              Bash
> claude-tmux   work:1   ~/projects/personal/configs    Thinking…    ← selected
  claude-tmux   univ:8   ~/projects/universe/admin
```

Format: `{status} {project_name}   {tmux_session}:{window}   {shortened_path}   {action}`

Selected line highlighted with cursor indicator and accent color.

**Keybindings:**
| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `G` / `g` | Jump to bottom / top |
| `Enter` | Jump to selected session |
| `q` / `Esc` / `Ctrl-C` | Quit |
| `/` | Enter filter mode (type to filter list) |

**Filter mode:**
- Text input at the top of the list
- Filters sessions by substring match against project name, path, or tmux target
- `Esc` clears filter and returns to normal mode
- `Enter` jumps to first (or selected) match

### F5 — tmux Jump

Switch to the selected session's tmux pane.

- Quit the TUI (return alt screen).
- Determine if running inside tmux (`$TMUX` env var).
  - **Inside tmux**: exec `tmux switch-client -t {target}`
  - **Outside tmux**: exec `tmux attach-session -t {target}`
- Use `syscall.Exec` to replace the process so tmux popup closes cleanly.

### F6 — Refresh

Keep the list current while the popup is open.

- On startup, perform log rotation and initial read.
- Set a tick interval (750ms) to re-read the event log and update the list.
- Preserve cursor position across refreshes (match by SessionID).
- A separate 150ms tick drives the spinner animation for busy sessions.

---

## Non-Goals (MVP)

These are explicitly deferred:

- CPU/memory usage display
- Process kill functionality
- Theme system / color configuration
- Config file support
- Support for non-Claude agents (codex, cursor, etc.)
- Mouse support

---

## Project Structure

```
claude-tmux/
├── cmd/
│   └── claude-tmux/
│       └── main.go              # Entry point, CLI flags
├── hooks/
│   └── claude-tmux-hook.sh      # Bash hook for Claude Code events
├── internal/
│   ├── tui/
│   │   ├── model.go             # Bubbletea model, Update, View
│   │   └── styles.go            # Lipgloss style definitions
│   ├── session/
│   │   ├── session.go           # Session data type, sorting
│   │   └── events.go            # Event log reader & state derivation
│   └── tmux/
│       └── jump.go              # tmux switch/attach logic
├── docs/
│   ├── README.md                # Project README with setup instructions
│   └── SPEC.md                  # This file
├── go.mod
├── go.sum
├── Makefile
└── CLAUDE.md
```

### Package Responsibilities

**`cmd/claude-tmux`** — Thin entry point. Parses flags (`-v`/`--version`). Calls `tui.Run()`.

**`internal/session`** — All session discovery logic. No TUI dependency.

- `session.go` — `Session` struct definition, `Status` type, sorting, path utils.
- `events.go` — `ReadSessions()` reads `~/.claude-tmux/events.log`, derives session state from events. `RotateLog()` truncates on startup.

**`internal/tui`** — Bubbletea model, view, and styles.

- `model.go` — `model` struct, `Init()`, `Update()`, `View()`, `Run()`. Two modes: `modeNormal`, `modeFilter`.
- `styles.go` — Lipgloss style constants for all visual elements.

**`internal/tmux`** — tmux command execution.

- `jump.go` — `Jump(target string) error` — determines inside/outside tmux, execs the switch.

---

## Key Design Decisions

1. **Hooks over process scanning** — Claude Code hooks provide authoritative session state directly, eliminating fragile regex heuristics on pane content and process tree walks.
2. **JSON lines event log** — Simple append-only format. Single-line `echo >>` is atomic enough for concurrent writers. No file locking needed.
3. **Go + Bubbletea** — matches jeb-todo-md patterns, Charm ecosystem for TUI.
4. **syscall.Exec for jump** — replaces the TUI process entirely so tmux popup closes cleanly without a lingering shell.
5. **Separate session package** — keeps discovery logic testable and independent of TUI.
6. **750ms refresh** — file read is cheap; faster polling than the old 2s process scan gives more responsive status updates.
7. **Log rotation on startup** — prevents unbounded growth without a separate daemon.
