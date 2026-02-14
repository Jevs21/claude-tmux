package tui

import "github.com/charmbracelet/lipgloss"

// Gruvbox-inspired color palette
var (
	colorForeground = lipgloss.Color("#ebdbb2")
	colorDim        = lipgloss.Color("#928374")
	colorAccent     = lipgloss.Color("#fe8019") // orange
	colorGreen      = lipgloss.Color("#b8bb26")
	colorBlue       = lipgloss.Color("#83a598")
	colorAqua       = lipgloss.Color("#8ec07c")
	colorSelectedBg = lipgloss.Color("#3c3836")
	colorRed        = lipgloss.Color("#fb4934")
	colorYellow     = lipgloss.Color("#fabd2f")
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorForeground).
			Background(colorSelectedBg)

	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	normalStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	projectStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	projectSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorGreen).
				Background(colorSelectedBg)

	tmuxTargetStyle = lipgloss.NewStyle().
			Foreground(colorBlue)

	tmuxTargetSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorBlue).
				Background(colorSelectedBg)

	pathStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	pathSelectedStyle = lipgloss.NewStyle().
				Foreground(colorAqua).
				Background(colorSelectedBg)

	filterPromptStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	emptyStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	detachedStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Italic(true)

	detachedSelectedStyle = lipgloss.NewStyle().
				Foreground(colorRed).
				Italic(true).
				Background(colorSelectedBg)

	statusBusyStyle = lipgloss.NewStyle().
			Foreground(colorYellow)

	statusBusySelectedStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Background(colorSelectedBg)

	statusIdleStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	statusIdleSelectedStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Background(colorSelectedBg)

	statusUnknownStyle = lipgloss.NewStyle().
				Foreground(colorDim)

	statusUnknownSelectedStyle = lipgloss.NewStyle().
					Foreground(colorDim).
					Background(colorSelectedBg)
)
