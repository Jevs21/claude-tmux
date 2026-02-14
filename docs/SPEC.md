# claude-tmux — Specification & Build Plan

A Go TUI tool (using Bubbletea) that runs as a tmux popup, listing active Claude Code sessions and letting you jump between them.

---

## Core Functionality (MVP)

Find all running `claude` processes, map each to its tmux pane, display them in a navigable list, and switch to the selected pane on enter.

---

## Feature List

### F1 — Process Discovery

Detect running Claude Code sessions by scanning system processes.

- Run `ps -axo pid,ppid,comm,command` and match processes where `comm` == `claude`.
- Filter out child processes: build a PID→PPID map; if a matched process's parent is also a matched process, exclude the child.
- Collect per-process: PID, PPID, working directory (from `lsof -p <pid> -Fn` cwd entry, or `/proc/<pid>/cwd` on Linux).

### F2 — tmux Pane Mapping

Map each Claude process to the tmux pane it's running in.

- Run `tmux list-panes -a -F "#{pane_pid} #{session_name} #{window_index} #{pane_index}"` once to get all pane PIDs.
- Build a map of `panePID → tmuxTarget` where `tmuxTarget` is `session:window.pane`.
- For each Claude process, walk up the process tree (PID→PPID, up to 25 hops) until a PID matches a tmux pane PID.
- If no match found, the session is "detached" (still shown, but not jumpable).

### F3 — Session List Model

Build the data model that the TUI renders.

Each session entry contains:
- **PID**: the Claude process PID
- **Working directory**: shortened path (`~` for home, truncate long paths)
- **tmux target**: `session:window.pane` string (empty if detached)
- **Project name**: last path component of the working directory
- **Status**: busy, idle, or unknown (determined by pane capture)

Sessions are sorted alphabetically by tmux session name, then by window index.

### F4 — TUI (Bubbletea)

Interactive terminal list with keyboard navigation.

**Display per session (single line):**
```
  claude-tmux   work:2   ~/projects/personal
> claude-tmux   work:1   ~/projects/personal/configs    ← selected
  claude-tmux   univ:8   ~/projects/universe/admin
```

Format: `{project_name}   {tmux_session}:{window}   {shortened_path}`

Selected line highlighted with cursor indicator and accent color.

**Keybindings:**
| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
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

### F6 — Refresh on Focus

Keep the list current while the popup is open.

- On startup, perform a full scan (F1 + F2).
- Set a tick interval (e.g. 2 seconds) to re-scan and update the list.
- Preserve cursor position across refreshes (match by PID).

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
├── internal/
│   ├── tui/
│   │   ├── model.go             # Bubbletea model, Update, View
│   │   └── styles.go            # Lipgloss style definitions
│   ├── session/
│   │   ├── scanner.go           # Process discovery (F1)
│   │   ├── tmux.go              # tmux pane mapping (F2)
│   │   ├── session.go           # Session data type (F3)
│   │   └── status.go            # Pane capture & busy/idle detection
│   └── tmux/
│       └── jump.go              # tmux switch/attach logic (F5)
├── docs/
│   └── SPEC.md                  # This file
├── go.mod
├── go.sum
├── Makefile
├── CLAUDE.md
└── README.md
```

### Package Responsibilities

**`cmd/claude-tmux`** — Thin entry point. Parses flags (`-v`/`--version`). Calls `tui.Run()`.

**`internal/session`** — All session discovery logic. No TUI dependency.

- `session.go` — `Session` struct definition, `Status` type, sorting.
- `scanner.go` — `Scan() ([]Session, error)` — runs ps, parses output, filters children, resolves working directories.
- `tmux.go` — `MapPanes(sessions []Session) []Session` — queries tmux, walks process trees, attaches pane targets.
- `status.go` — `CaptureStatuses(sessions []Session)` — captures tmux pane content, detects busy/idle/unknown status via spinner and prompt patterns.

**`internal/tui`** — Bubbletea model, view, and styles.

- `model.go` — `model` struct, `Init()`, `Update()`, `View()`, `Run()`. Two modes: `ModeNormal`, `ModeFilter`.
- `styles.go` — Lipgloss style constants for selected/unselected/header/filter.
- `keys.go` — Key constants and help text.

**`internal/tmux`** — tmux command execution.

- `jump.go` — `Jump(target string) error` — determines inside/outside tmux, execs the switch.

---

## Build Plan

Ordered implementation steps. Each step produces testable, runnable output.

### Step 1 — Project Scaffold

- `go mod init github.com/Jevs21/claude-tmux`
- Create directory structure
- `Makefile` (build, test, clean)
- Minimal `main.go` that prints version

### Step 2 — Session Data Types

File: `internal/session/session.go`

```go
type Session struct {
    PID         int
    PPID        int
    WorkDir     string
    ProjectName string
    TmuxTarget  string   // "session:window.pane" or empty
    TmuxSession string   // just the session name
    WindowIndex int
    PaneIndex   int
}
```

- `ShortenPath(path string) string` — replaces home dir with `~`, truncates long paths
- `Sort(sessions []Session)` — sort by tmux session, then window index

### Step 3 — Process Scanner

File: `internal/session/scanner.go`

- `Scan() ([]Session, error)`
- Exec `ps -axo pid,ppid,comm,command`
- Parse output, match `comm == "claude"`
- Build PID→PPID map, filter child processes
- Resolve working directory via `lsof -p <pid>`
- Return `[]Session` with PID, PPID, WorkDir, ProjectName populated

Write tests with mock ps output.

### Step 4 — tmux Pane Mapping

File: `internal/session/tmux.go`

- `MapPanes(sessions []Session) []Session`
- Exec `tmux list-panes -a -F "#{pane_pid} #{session_name} #{window_index} #{pane_index}"`
- Parse into map `panePID → (session, window, pane)`
- For each session, walk PID→PPID tree to find matching pane PID
- Populate TmuxTarget, TmuxSession, WindowIndex, PaneIndex
- Process tree data comes from a single `ps -axo pid,ppid` call

### Step 5 — tmux Jump

File: `internal/tmux/jump.go`

- `Jump(target string) error`
- Check `$TMUX` env var
- Inside tmux: `syscall.Exec("tmux", ["tmux", "switch-client", "-t", target])`
- Outside tmux: `syscall.Exec("tmux", ["tmux", "attach-session", "-t", target])`

### Step 6 — TUI Styles

File: `internal/tui/styles.go`

Define lipgloss styles:
- `selectedStyle` — bold, accent foreground, background highlight
- `normalStyle` — dim foreground
- `headerStyle` — bold, accent color
- `filterStyle` — for the filter input prompt
- `pathStyle` — dimmed path color
- `tmuxStyle` — distinct color for tmux target column
- `emptyStyle` — message when no sessions found

Color palette: gruvbox-inspired (matches rpai default).

### Step 7 — TUI Model & View

File: `internal/tui/model.go`

Model:
```go
type model struct {
    sessions      []session.Session
    filtered      []session.Session  // subset after filter
    cursor        int
    mode          mode               // ModeNormal | ModeFilter
    filterInput   textinput.Model
    filterText    string
    err           error
    width         int
    height        int
}
```

Messages:
- `tea.KeyMsg` — keyboard input
- `sessionsMsg` — result of async scan
- `tickMsg` — periodic refresh trigger

Update:
- `ModeNormal`: j/k navigation, enter to jump, `/` to filter, q to quit
- `ModeFilter`: text input, esc to cancel, enter to jump

View:
- Header line: `Claude Sessions (N)`
- Session list: one line per session, columns aligned
- Footer: keybinding hints
- Empty state: "No Claude sessions found"

### Step 8 — Periodic Refresh

- `tickCmd()` returns a `tea.Tick` command at 2-second intervals
- On tick, re-run `Scan()` + `MapPanes()` as a `tea.Cmd`
- On `sessionsMsg`, update session list, reapply filter, preserve cursor by PID

### Step 9 — Integration & Polish

- Wire everything together in `main.go`
- `tea.WithAltScreen()` for clean tmux popup behavior
- Test in tmux popup: `tmux display-popup -E "claude-tmux"`
- Handle edge cases: no tmux, no sessions, terminal resize
- `CLAUDE.md` project documentation

### Step 10 — Release Tooling

- `.goreleaser.yml` (darwin amd64/arm64, linux amd64/arm64)
- `.github/workflows/test.yml`
- `.github/workflows/release.yml`

---

## Key Design Decisions

1. **Go + Bubbletea** — matches jeb-todo-md patterns, Charm ecosystem for TUI.
2. **Process scanning via ps/lsof** — no external dependencies, works on macOS and Linux.
3. **syscall.Exec for jump** — replaces the TUI process entirely so tmux popup closes cleanly without a lingering shell.
4. **Separate session package** — keeps discovery logic testable and independent of TUI.
5. **Single-line session display** — compact for tmux popup, scannable at a glance.
6. **Filter over search** — type-ahead filtering is faster than a separate search mode for small lists.
7. **2-second refresh** — balances freshness with low overhead. Process scanning is cheap.
