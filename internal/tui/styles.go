package tui

import "charm.land/lipgloss/v2"

// Color palette
var (
	colorPrimary   = lipgloss.Color("12") // bright blue
	colorSecondary = lipgloss.Color("10") // bright green
	colorAccent    = lipgloss.Color("13") // bright magenta
	colorMuted     = lipgloss.Color("245")
	colorBg        = lipgloss.Color("0")
	colorFg        = lipgloss.Color("15")
	colorWarning   = lipgloss.Color("11") // bright yellow
	colorError     = lipgloss.Color("9")  // bright red
	colorDim       = lipgloss.Color("241")
)

// Common styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorMuted).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	ManagerTagStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	VersionStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	DescStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(colorError)

	WarningStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	DialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 3).
			Width(50)

	CursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(colorPrimary)
)
