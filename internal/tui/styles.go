package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Brand colors
	ColorPrimary   = lipgloss.Color("#5A56E0")
	ColorSecondary = lipgloss.Color("#7C4DFF")
	ColorActive    = lipgloss.Color("#00E676")
	ColorInactive  = lipgloss.Color("#9E9E9E")
	ColorFailed    = lipgloss.Color("#FF1744")
	ColorWarning   = lipgloss.Color("#FFAB00")
	ColorBorder    = lipgloss.Color("#3E3E5E")
	ColorFocused   = lipgloss.Color("#7C4DFF")
	ColorBgDark    = lipgloss.Color("#1A1A24")
	ColorText      = lipgloss.Color("#ECEFF1")

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	// Panes
	PaneBaseStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	PaneFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorFocused).
				Padding(0, 1)

	// Status Badges
	BadgeActive = lipgloss.NewStyle().
			Foreground(ColorActive).
			Bold(true)

	BadgeInactive = lipgloss.NewStyle().
			Foreground(ColorInactive)

	BadgeFailed = lipgloss.NewStyle().
			Foreground(ColorFailed).
			Bold(true)

	// Status line & Footer
	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#90A4AE")).
			Background(lipgloss.Color("#263238")).
			Padding(0, 1)

	KeyHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
)
