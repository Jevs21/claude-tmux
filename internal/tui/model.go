package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jevs21/claude-tmux/internal/session"
	"github.com/Jevs21/claude-tmux/internal/tmux"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const refreshInterval = 750 * time.Millisecond
const spinnerInterval = 150 * time.Millisecond

// spinnerFrames are the characters used for the busy status animation,
// matching the spinner characters Claude Code uses.
var spinnerFrames = []rune{'✻', '✽', '✳', '·', '✶', '✢'}

// mode represents the current TUI interaction mode.
type mode int

const (
	modeNormal mode = iota
	modeFilter
)

// model is the Bubbletea model for the session list TUI.
type model struct {
	sessions     []session.Session
	filtered     []session.Session
	cursor       int
	mode         mode
	filterInput  textinput.Model
	filterText   string
	err          error
	width        int
	height       int
	jumpTarget   string // set when user selects a session to jump to
	spinnerFrame int    // current index into spinnerFrames for busy animation
}

// sessionsMsg carries the result of an async session read.
type sessionsMsg struct {
	sessions []session.Session
	err      error
}

// tickMsg triggers a periodic refresh.
type tickMsg time.Time

// spinnerTickMsg triggers a spinner frame advance.
type spinnerTickMsg time.Time

// scanCmd reads sessions from the event log.
func scanCmd() tea.Msg {
	sessions, err := session.ReadSessions()
	if err != nil {
		return sessionsMsg{err: err}
	}
	return sessionsMsg{sessions: sessions}
}

// tickCmd returns a command that sends a tickMsg after the refresh interval.
func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// spinnerTickCmd returns a command that sends a spinnerTickMsg for animation.
func spinnerTickCmd() tea.Cmd {
	return tea.Tick(spinnerInterval, func(t time.Time) tea.Msg {
		return spinnerTickMsg(t)
	})
}

func initialModel() model {
	filterInput := textinput.New()
	filterInput.Placeholder = "filter sessions..."
	filterInput.CharLimit = 64

	return model{
		filterInput: filterInput,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(scanCmd, tickCmd(), spinnerTickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case sessionsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		previousSessionID := m.selectedSessionID()
		m.sessions = msg.sessions
		m.applyFilter()
		m.restoreCursorBySessionID(previousSessionID)
		return m, nil

	case tickMsg:
		return m, tea.Batch(scanCmd, tickCmd())

	case spinnerTickMsg:
		m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
		return m, spinnerTickCmd()

	case tea.KeyMsg:
		switch m.mode {
		case modeNormal:
			return m.updateNormal(msg)
		case modeFilter:
			return m.updateFilter(msg)
		}
	}

	return m, nil
}

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "enter":
		if len(m.filtered) > 0 && m.filtered[m.cursor].Jumpable() {
			m.jumpTarget = m.filtered[m.cursor].TmuxTarget
			return m, tea.Quit
		}
		return m, nil

	case "/":
		m.mode = modeFilter
		m.filterInput.SetValue(m.filterText)
		cmd := m.filterInput.Focus()
		return m, cmd

	case "G":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
		return m, nil

	case "g":
		m.cursor = 0
		return m, nil
	}

	return m, nil
}

func (m model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.filterText = ""
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.applyFilter()
		m.clampCursor()
		return m, nil

	case "enter":
		m.mode = modeNormal
		m.filterText = m.filterInput.Value()
		m.filterInput.Blur()
		m.applyFilter()
		m.clampCursor()

		// Jump to first match if available
		if len(m.filtered) > 0 && m.filtered[m.cursor].Jumpable() {
			m.jumpTarget = m.filtered[m.cursor].TmuxTarget
			return m, tea.Quit
		}
		return m, nil

	case "ctrl+c":
		return m, tea.Quit
	}

	// Pass keystrokes to the text input
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterText = m.filterInput.Value()
	m.applyFilter()
	m.clampCursor()
	return m, cmd
}

func (m model) View() string {
	var builder strings.Builder

	// Header
	sessionCount := len(m.filtered)
	headerText := fmt.Sprintf("Claude Sessions (%d)", sessionCount)
	builder.WriteString(headerStyle.Render(headerText))
	builder.WriteString("\n")

	// Filter input (shown in filter mode or when filter is active)
	if m.mode == modeFilter {
		builder.WriteString(filterPromptStyle.Render("/ "))
		builder.WriteString(m.filterInput.View())
		builder.WriteString("\n\n")
	} else if m.filterText != "" {
		builder.WriteString(filterPromptStyle.Render("filter: "))
		builder.WriteString(helpStyle.Render(m.filterText))
		builder.WriteString("\n\n")
	}

	// Error state
	if m.err != nil {
		builder.WriteString(emptyStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		builder.WriteString("\n")
		return builder.String()
	}

	// Empty state
	if len(m.filtered) == 0 {
		if len(m.sessions) == 0 {
			builder.WriteString(emptyStyle.Render("No Claude sessions found"))
		} else {
			builder.WriteString(emptyStyle.Render("No sessions match filter"))
		}
		builder.WriteString("\n")
		return builder.String()
	}

	// Calculate column widths for alignment
	maxProjectWidth := 0
	maxTargetWidth := 0
	maxActionWidth := 0
	for _, s := range m.filtered {
		if len(s.ProjectName) > maxProjectWidth {
			maxProjectWidth = len(s.ProjectName)
		}
		displayTarget := s.DisplayTarget()
		if len(displayTarget) > maxTargetWidth {
			maxTargetWidth = len(displayTarget)
		}
		if len(s.Action) > maxActionWidth {
			maxActionWidth = len(s.Action)
		}
	}

	// Session list
	for i, s := range m.filtered {
		isSelected := i == m.cursor

		// Render status indicator
		statusIndicator := m.renderStatusIndicator(s.Status, isSelected)

		var line string
		if isSelected {
			cursor := cursorStyle.Render("> ")
			projectName := projectSelectedStyle.
				Width(maxProjectWidth).
				Render(s.ProjectName)
			target := tmuxTargetSelectedStyle.
				Width(maxTargetWidth).
				Render(s.DisplayTarget())
			if !s.Jumpable() {
				target = detachedSelectedStyle.
					Width(maxTargetWidth).
					Render(s.DisplayTarget())
			}
			displayPath := pathSelectedStyle.Render(s.DisplayPath())
			line = cursor + statusIndicator + projectName + "  " + target + "  " + displayPath

			if s.Action != "" {
				actionText := actionSelectedStyle.
					Width(maxActionWidth).
					Render(s.Action)
				line += "  " + actionText
			}
		} else {
			cursor := "  "
			projectName := projectStyle.
				Width(maxProjectWidth).
				Render(s.ProjectName)
			target := tmuxTargetStyle.
				Width(maxTargetWidth).
				Render(s.DisplayTarget())
			if !s.Jumpable() {
				target = detachedStyle.
					Width(maxTargetWidth).
					Render(s.DisplayTarget())
			}
			displayPath := pathStyle.Render(s.DisplayPath())
			line = cursor + statusIndicator + projectName + "  " + target + "  " + displayPath

			if s.Action != "" {
				actionText := actionStyle.
					Width(maxActionWidth).
					Render(s.Action)
				line += "  " + actionText
			}
		}

		builder.WriteString(line)
		builder.WriteString("\n")
	}

	// Footer help
	builder.WriteString("\n")
	if m.mode == modeFilter {
		builder.WriteString(helpStyle.Render("enter: jump  esc: clear filter  ctrl+c: quit"))
	} else {
		builder.WriteString(helpStyle.Render("j/k: navigate  enter: jump  /: filter  q: quit"))
	}

	return builder.String()
}

// renderStatusIndicator returns a styled status character with a trailing space.
func (m model) renderStatusIndicator(status session.Status, isSelected bool) string {
	switch status {
	case session.StatusBusy:
		char := string(spinnerFrames[m.spinnerFrame])
		if isSelected {
			return statusBusySelectedStyle.Render(char) + " "
		}
		return statusBusyStyle.Render(char) + " "
	case session.StatusWaiting:
		if isSelected {
			return statusWaitingSelectedStyle.Render("?") + " "
		}
		return statusWaitingStyle.Render("?") + " "
	case session.StatusIdle:
		if isSelected {
			return statusIdleSelectedStyle.Render("●") + " "
		}
		return statusIdleStyle.Render("●") + " "
	default:
		if isSelected {
			return statusUnknownSelectedStyle.Render("●") + " "
		}
		return statusUnknownStyle.Render("●") + " "
	}
}

// applyFilter updates the filtered session list based on the current filter text.
func (m *model) applyFilter() {
	if m.filterText == "" {
		m.filtered = m.sessions
		return
	}

	filterLower := strings.ToLower(m.filterText)
	var filtered []session.Session
	for _, s := range m.sessions {
		searchText := strings.ToLower(
			s.ProjectName + " " + s.TmuxTarget + " " + s.WorkDir,
		)
		if strings.Contains(searchText, filterLower) {
			filtered = append(filtered, s)
		}
	}
	m.filtered = filtered
}

// clampCursor ensures the cursor is within valid bounds.
func (m *model) clampCursor() {
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// selectedSessionID returns the SessionID of the currently selected session, or empty if none.
func (m model) selectedSessionID() string {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		return m.filtered[m.cursor].SessionID
	}
	return ""
}

// restoreCursorBySessionID attempts to restore the cursor to the session with the given ID.
func (m *model) restoreCursorBySessionID(sessionID string) {
	if sessionID == "" {
		m.clampCursor()
		return
	}
	for i, s := range m.filtered {
		if s.SessionID == sessionID {
			m.cursor = i
			return
		}
	}
	m.clampCursor()
}

// Run starts the Bubbletea TUI program and handles the jump after exit.
func Run() error {
	// Rotate log on startup to prevent unbounded growth
	_ = session.RotateLog()

	m := initialModel()
	program := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// After the TUI exits, check if we need to jump
	if finalM, ok := finalModel.(model); ok && finalM.jumpTarget != "" {
		return tmux.Jump(finalM.jumpTarget)
	}

	return nil
}
