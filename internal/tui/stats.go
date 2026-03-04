package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kyleboon/gochess/internal/db"
)

// RenderPlayerStats renders player statistics in a beautiful TUI format
func RenderPlayerStats(stats []db.PlayerStats, totalGames int) string {
	var b strings.Builder

	// Title
	title := TitleStyle.Render("♔ Player Statistics ♔")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Total games info
	info := StatLabelStyle.Render("Total Games:") + " " + StatValueStyle.Render(fmt.Sprintf("%d", totalGames))
	b.WriteString(info)
	b.WriteString("\n\n")

	// Table header
	headerCols := []string{
		HeaderStyle.Copy().Width(20).Render("PLAYER"),
		HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("GAMES"),
		HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("WINS"),
		HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("LOSSES"),
		HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("DRAWS"),
		HeaderStyle.Copy().Width(10).Align(lipgloss.Right).Render("WIN RATE"),
		HeaderStyle.Copy().Width(10).Align(lipgloss.Right).Render("AS WHITE"),
		HeaderStyle.Copy().Width(10).Align(lipgloss.Right).Render("AS BLACK"),
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerCols...)
	b.WriteString(header)
	b.WriteString("\n")

	// Table rows
	for i, s := range stats {
		// Alternate row colors
		rowStyle := RowStyle
		if i%2 == 1 {
			rowStyle = RowAltStyle
		}

		// Truncate long names
		name := s.Name
		if len(name) > 18 {
			name = name[:15] + "..."
		}

		// Format win rate with color
		winRateText := fmt.Sprintf("%.1f%%", s.WinRate)
		var winRateStyle lipgloss.Style
		switch {
		case s.WinRate >= 60:
			winRateStyle = WinStyle
		case s.WinRate >= 45:
			winRateStyle = lipgloss.NewStyle().Foreground(ColorTextBright)
		case s.WinRate >= 30:
			winRateStyle = DrawStyle
		default:
			winRateStyle = LossStyle
		}

		// Format color-specific rates
		whiteRateText := fmt.Sprintf("%.1f%%", s.WhiteWinRate)
		blackRateText := fmt.Sprintf("%.1f%%", s.BlackWinRate)

		cols := []string{
			rowStyle.Copy().Width(20).Render(name),
			rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%d", s.Games)),
			rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(WinStyle.Render(fmt.Sprintf("%d", s.Wins))),
			rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(LossStyle.Render(fmt.Sprintf("%d", s.Losses))),
			rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(DrawStyle.Render(fmt.Sprintf("%d", s.Draws))),
			rowStyle.Copy().Width(10).Align(lipgloss.Right).Render(winRateStyle.Render(winRateText)),
			rowStyle.Copy().Width(10).Align(lipgloss.Right).Render(whiteRateText),
			rowStyle.Copy().Width(10).Align(lipgloss.Right).Render(blackRateText),
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// If single player, show detailed stats
	if len(stats) == 1 {
		b.WriteString("\n")
		b.WriteString(renderDetailedPlayerStats(stats[0]))
	}

	return BorderStyle.Render(b.String())
}

// renderDetailedPlayerStats renders detailed statistics for a single player
func renderDetailedPlayerStats(s db.PlayerStats) string {
	var b strings.Builder

	subtitle := SubtitleStyle.Render(fmt.Sprintf("Detailed Statistics for %s", s.Name))
	b.WriteString(subtitle)
	b.WriteString("\n")

	// Performance by color
	b.WriteString(StatLabelStyle.Render("Performance by Color:"))
	b.WriteString("\n")

	whiteRecord := fmt.Sprintf("%d-%d-%d (W-L-D)", s.WhiteWins, s.WhiteLosses, s.WhiteDraws)
	b.WriteString(fmt.Sprintf("  As White: %s games, %s, %s win rate\n",
		StatValueStyle.Render(fmt.Sprintf("%d", s.WhiteGames)),
		whiteRecord,
		WinStyle.Render(fmt.Sprintf("%.1f%%", s.WhiteWinRate))))

	blackRecord := fmt.Sprintf("%d-%d-%d (W-L-D)", s.BlackWins, s.BlackLosses, s.BlackDraws)
	b.WriteString(fmt.Sprintf("  As Black: %s games, %s, %s win rate\n",
		StatValueStyle.Render(fmt.Sprintf("%d", s.BlackGames)),
		blackRecord,
		WinStyle.Render(fmt.Sprintf("%.1f%%", s.BlackWinRate))))

	// Time control breakdown
	if s.BulletGames > 0 || s.BlitzGames > 0 || s.RapidGames > 0 || s.ClassicalGames > 0 {
		b.WriteString("\n")
		b.WriteString(StatLabelStyle.Render("Games by Time Control:"))
		b.WriteString("\n")

		if s.BulletGames > 0 {
			pct := float64(s.BulletGames) / float64(s.Games) * 100
			b.WriteString(fmt.Sprintf("  Bullet:    %s games (%.1f%%)\n",
				StatValueStyle.Render(fmt.Sprintf("%d", s.BulletGames)), pct))
		}
		if s.BlitzGames > 0 {
			pct := float64(s.BlitzGames) / float64(s.Games) * 100
			b.WriteString(fmt.Sprintf("  Blitz:     %s games (%.1f%%)\n",
				StatValueStyle.Render(fmt.Sprintf("%d", s.BlitzGames)), pct))
		}
		if s.RapidGames > 0 {
			pct := float64(s.RapidGames) / float64(s.Games) * 100
			b.WriteString(fmt.Sprintf("  Rapid:     %s games (%.1f%%)\n",
				StatValueStyle.Render(fmt.Sprintf("%d", s.RapidGames)), pct))
		}
		if s.ClassicalGames > 0 {
			pct := float64(s.ClassicalGames) / float64(s.Games) * 100
			b.WriteString(fmt.Sprintf("  Classical: %s games (%.1f%%)\n",
				StatValueStyle.Render(fmt.Sprintf("%d", s.ClassicalGames)), pct))
		}
	}

	return b.String()
}

// RenderOpeningStats renders opening statistics in a beautiful TUI format
func RenderOpeningStats(openings []db.OpeningStats, playerName string, limit int) string {
	var b strings.Builder

	subtitle := SubtitleStyle.Render("♟ Opening Statistics")
	b.WriteString(subtitle)
	b.WriteString("\n")

	if len(openings) == 0 {
		b.WriteString(HelpStyle.Render("No opening statistics available"))
		return b.String()
	}

	// Show top N openings
	displayCount := limit
	if len(openings) < displayCount {
		displayCount = len(openings)
	}

	b.WriteString(fmt.Sprintf("  Top %d Most Played Openings:\n", displayCount))
	b.WriteString("\n")

	// Table header
	headerCols := []string{
		HeaderStyle.Copy().Width(6).Render("ECO"),
		HeaderStyle.Copy().Width(35).Render("OPENING"),
		HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("GAMES"),
		HeaderStyle.Copy().Width(10).Align(lipgloss.Right).Render("WIN%"),
		HeaderStyle.Copy().Width(10).Align(lipgloss.Right).Render("AS WHITE"),
		HeaderStyle.Copy().Width(10).Align(lipgloss.Right).Render("AS BLACK"),
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerCols...)
	b.WriteString("  " + header)
	b.WriteString("\n")

	// Table rows
	for i := 0; i < displayCount; i++ {
		op := openings[i]
		rowStyle := RowStyle
		if i%2 == 1 {
			rowStyle = RowAltStyle
		}

		// Truncate long opening names
		name := op.OpeningName
		if len(name) > 33 {
			name = name[:30] + "..."
		}

		// Format win rate with color
		winRateText := fmt.Sprintf("%.1f%%", op.WinRate)
		var winRateStyle lipgloss.Style
		switch {
		case op.WinRate >= 60:
			winRateStyle = WinStyle
		case op.WinRate >= 45:
			winRateStyle = lipgloss.NewStyle().Foreground(ColorTextBright)
		case op.WinRate >= 30:
			winRateStyle = DrawStyle
		default:
			winRateStyle = LossStyle
		}

		cols := []string{
			rowStyle.Copy().Width(6).Render(op.ECOCode),
			rowStyle.Copy().Width(35).Render(name),
			rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(fmt.Sprintf("%d", op.Games)),
			rowStyle.Copy().Width(10).Align(lipgloss.Right).Render(winRateStyle.Render(winRateText)),
			rowStyle.Copy().Width(10).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f%%", op.WhiteWinRate)),
			rowStyle.Copy().Width(10).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f%%", op.BlackWinRate)),
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
		b.WriteString("  " + row)
		b.WriteString("\n")
	}

	// Show best/worst for single player
	if playerName != "" && len(openings) > 0 {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Opening Performance for %s:\n", StatValueStyle.Render(playerName)))

		var best, worst *db.OpeningStats
		for i := range openings {
			if openings[i].Games >= 3 {
				if best == nil || openings[i].WinRate > best.WinRate {
					best = &openings[i]
				}
				if worst == nil || openings[i].WinRate < worst.WinRate {
					worst = &openings[i]
				}
			}
		}

		if best != nil {
			b.WriteString(fmt.Sprintf("    Best:  %s (%s) - %s games, %s win rate\n",
				WinStyle.Render(best.ECOCode),
				best.OpeningName,
				StatValueStyle.Render(fmt.Sprintf("%d", best.Games)),
				WinStyle.Render(fmt.Sprintf("%.1f%%", best.WinRate))))
		}
		if worst != nil && worst.ECOCode != best.ECOCode {
			b.WriteString(fmt.Sprintf("    Worst: %s (%s) - %s games, %s win rate\n",
				LossStyle.Render(worst.ECOCode),
				worst.OpeningName,
				StatValueStyle.Render(fmt.Sprintf("%d", worst.Games)),
				LossStyle.Render(fmt.Sprintf("%.1f%%", worst.WinRate))))
		}
	}

	return b.String()
}

// RenderPositionStats renders position statistics in a beautiful TUI format
func RenderPositionStats(uniqueCount int, topPositions []db.PositionFrequency) string {
	var b strings.Builder

	subtitle := SubtitleStyle.Render("♞ Position Statistics")
	b.WriteString(subtitle)
	b.WriteString("\n")

	if uniqueCount == 0 {
		b.WriteString(HelpStyle.Render("No position statistics available"))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  Unique positions: %s\n", StatValueStyle.Render(fmt.Sprintf("%d", uniqueCount))))

	if len(topPositions) > 0 {
		b.WriteString("\n")
		b.WriteString("  Top 10 Most Common Positions (after move 10):\n")
		b.WriteString("\n")

		// Table header
		headerCols := []string{
			HeaderStyle.Copy().Width(6).Align(lipgloss.Right).Render("COUNT"),
			HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("WHITE%"),
			HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("BLACK%"),
			HeaderStyle.Copy().Width(8).Align(lipgloss.Right).Render("DRAW%"),
			HeaderStyle.Copy().Width(6).Render("ECO"),
			HeaderStyle.Copy().Width(25).Render("OPENING"),
			HeaderStyle.Copy().Width(40).Render("FEN"),
		}
		header := lipgloss.JoinHorizontal(lipgloss.Top, headerCols...)
		b.WriteString("  " + header)
		b.WriteString("\n")

		// Table rows
		for i, pos := range topPositions {
			rowStyle := RowStyle
			if i%2 == 1 {
				rowStyle = RowAltStyle
			}

			// Truncate long strings
			fen := pos.FEN
			if len(fen) > 38 {
				fen = fen[:35] + "..."
			}

			opening := pos.OpeningName
			if len(opening) > 23 {
				opening = opening[:20] + "..."
			}

			eco := pos.ECOCode
			if eco == "" {
				eco = "-"
			}
			if opening == "" {
				opening = "-"
			}

			cols := []string{
				rowStyle.Copy().Width(6).Align(lipgloss.Right).Render(fmt.Sprintf("%d", pos.Count)),
				rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(WinStyle.Render(fmt.Sprintf("%.1f", pos.WhiteWinPct))),
				rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(LossStyle.Render(fmt.Sprintf("%.1f", pos.BlackWinPct))),
				rowStyle.Copy().Width(8).Align(lipgloss.Right).Render(DrawStyle.Render(fmt.Sprintf("%.1f", pos.DrawPct))),
				rowStyle.Copy().Width(6).Render(eco),
				rowStyle.Copy().Width(25).Render(opening),
				rowStyle.Copy().Width(40).Render(fen),
			}
			row := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
			b.WriteString("  " + row)
			b.WriteString("\n")
		}
	}

	return b.String()
}
