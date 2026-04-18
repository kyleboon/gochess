package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kyleboon/gochess/internal/db"
	"github.com/kyleboon/gochess/internal/tui"
	"github.com/urfave/cli/v2"
)

// gameListTUICommand shows an interactive game list browser
func gameListTUICommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	limit := c.Int("limit")
	offset := c.Int("offset")

	// Prepare search criteria
	criteria := make(map[string]string)
	if white := c.String("white"); white != "" {
		criteria["white"] = white
	}
	if black := c.String("black"); black != "" {
		criteria["black"] = black
	}
	if event := c.String("event"); event != "" {
		criteria["event"] = event
	}
	if date := c.String("date"); date != "" {
		criteria["date"] = date
	}

	// Open database connection
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Query games with filters
	gamesMaps, err := database.SearchGames(c.Context, criteria, limit, offset)
	if err != nil {
		return fmt.Errorf("failed to search games: %w", err)
	}

	if len(gamesMaps) == 0 {
		fmt.Println("No games found matching the criteria")
		return nil
	}

	// Convert maps to Game structs
	games := make([]tui.Game, len(gamesMaps))
	for i, gameMap := range gamesMaps {
		games[i] = tui.MapToGame(gameMap)
	}

	// Start the TUI
	model := tui.NewGameListModel(games)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}
