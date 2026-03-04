package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ImportProgress represents progress of an import operation
type ImportProgress struct {
	Current    int
	Total      int
	CurrentMsg string
	Errors     []string
}

// ImportProgressModel is a Bubble Tea model for showing import progress
type ImportProgressModel struct {
	spinner  spinner.Model
	progress ImportProgress
	done     bool
	quitting bool
}

// NewImportProgressModel creates a new import progress model
func NewImportProgressModel() ImportProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return ImportProgressModel{
		spinner: s,
	}
}

// Init initializes the model
func (m ImportProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// progressMsg is sent when progress is updated
type progressMsg ImportProgress

// doneMsg is sent when the import is complete
type doneMsg struct{}

// UpdateProgress returns a command to update progress
func UpdateProgress(p ImportProgress) tea.Cmd {
	return func() tea.Msg {
		return progressMsg(p)
	}
}

// Done returns a command to mark import as complete
func Done() tea.Cmd {
	return func() tea.Msg {
		return doneMsg{}
	}
}

// Update handles messages
func (m ImportProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case progressMsg:
		m.progress = ImportProgress(msg)
		return m, nil

	case doneMsg:
		m.done = true
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the model
func (m ImportProgressModel) View() string {
	if m.quitting {
		return "Import cancelled.\n"
	}

	if m.done {
		return m.renderComplete()
	}

	return m.renderProgress()
}

// renderProgress renders the progress view
func (m ImportProgressModel) renderProgress() string {
	var b strings.Builder

	// Title
	title := TitleStyle.Render("♔ Importing Chess Games")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Spinner and current message
	b.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), m.progress.CurrentMsg))
	b.WriteString("\n")

	// Progress bar
	if m.progress.Total > 0 {
		pct := float64(m.progress.Current) / float64(m.progress.Total) * 100
		bar := renderProgressBar(m.progress.Current, m.progress.Total, 50)
		b.WriteString(fmt.Sprintf("%s %.1f%% (%d/%d)\n",
			bar, pct, m.progress.Current, m.progress.Total))
	}

	// Errors (if any)
	if len(m.progress.Errors) > 0 {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Errors: %d", len(m.progress.Errors))))
		b.WriteString("\n")
		for i, err := range m.progress.Errors {
			if i >= 5 {
				b.WriteString(fmt.Sprintf("  ... and %d more\n", len(m.progress.Errors)-5))
				break
			}
			b.WriteString(fmt.Sprintf("  - %s\n", err))
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Press 'ctrl+c' to cancel"))

	return BorderStyle.Render(b.String())
}

// renderComplete renders the completion view
func (m ImportProgressModel) renderComplete() string {
	var b strings.Builder

	successCount := m.progress.Total - len(m.progress.Errors)

	if len(m.progress.Errors) == 0 {
		// All successful
		b.WriteString(SuccessStyle.Render(fmt.Sprintf("✓ Successfully imported %d games!", successCount)))
	} else if successCount > 0 {
		// Some successful
		b.WriteString(DrawStyle.Render(fmt.Sprintf("⚠ Imported %d games with %d errors",
			successCount, len(m.progress.Errors))))
	} else {
		// All failed
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("✗ Import failed: %d errors", len(m.progress.Errors))))
	}

	if len(m.progress.Errors) > 0 {
		b.WriteString("\n\n")
		b.WriteString("Errors:\n")
		for i, err := range m.progress.Errors {
			if i >= 10 {
				b.WriteString(fmt.Sprintf("... and %d more\n", len(m.progress.Errors)-10))
				break
			}
			b.WriteString(fmt.Sprintf("  - %s\n", err))
		}
	}

	return b.String() + "\n"
}

// renderProgressBar renders a progress bar
func renderProgressBar(current, total, width int) string {
	if total == 0 {
		return ""
	}

	filled := int(float64(current) / float64(total) * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	style := lipgloss.NewStyle().Foreground(ColorSuccess)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorTextMuted)

	return style.Render(bar[:filled]) + emptyStyle.Render(bar[filled:])
}

// RenderSimpleProgress renders a simple progress message (non-interactive)
func RenderSimpleProgress(current, total int, message string) string {
	var b strings.Builder

	spinner := SpinnerStyle.Render("⣾")
	b.WriteString(fmt.Sprintf("%s %s\n", spinner, message))

	if total > 0 {
		pct := float64(current) / float64(total) * 100
		bar := renderProgressBar(current, total, 40)
		b.WriteString(fmt.Sprintf("%s %.1f%% (%d/%d)\n", bar, pct, current, total))
	}

	return b.String()
}
