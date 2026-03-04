package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7B61FF")
	ColorSecondary = lipgloss.Color("#FF6B9D")
	ColorAccent    = lipgloss.Color("#00D9FF")

	// Status colors
	ColorSuccess = lipgloss.Color("#00D787")
	ColorWarning = lipgloss.Color("#FFB86C")
	ColorError   = lipgloss.Color("#FF5555")
	ColorInfo    = lipgloss.Color("#8BE9FD")

	// Text colors
	ColorText       = lipgloss.Color("#F8F8F2")
	ColorTextMuted  = lipgloss.Color("#6272A4")
	ColorTextBright = lipgloss.Color("#FFFFFF")

	// Background colors
	ColorBg       = lipgloss.Color("#282A36")
	ColorBgDark   = lipgloss.Color("#1E1F29")
	ColorBgLight  = lipgloss.Color("#44475A")
	ColorBgAccent = lipgloss.Color("#373844")
)

// Styles
var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBg)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	// Table styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorTextBright).
			Background(ColorBgLight).
			Bold(true).
			Padding(0, 1)

	RowStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	RowAltStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBgAccent).
			Padding(0, 1)

	// Stats styles
	StatLabelStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Width(20).
			Align(lipgloss.Left)

	StatValueStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	// Win/Loss/Draw colors
	WinStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	LossStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	DrawStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	// Border styles
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// Progress/Loading
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// Error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError)

	// Success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true).
			MarginTop(1)
)

// Width helpers for responsive layouts
const (
	MinWidth  = 80
	MaxWidth  = 120
	MinHeight = 24
)

// FormatPercentage formats a win rate with color
func FormatPercentage(rate float64) string {
	var style lipgloss.Style
	switch {
	case rate >= 60:
		style = WinStyle
	case rate >= 45:
		style = lipgloss.NewStyle().Foreground(ColorTextBright)
	case rate >= 30:
		style = DrawStyle
	default:
		style = LossStyle
	}
	text := lipgloss.NewStyle().Width(6).Align(lipgloss.Right).Render(lipgloss.NewStyle().Render(""))
	return style.Render(text)
}
