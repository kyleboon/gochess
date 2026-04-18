package main

import (
	"fmt"

	"github.com/kyleboon/gochess/internal/config"
	"github.com/kyleboon/gochess/internal/db"
	"github.com/kyleboon/gochess/internal/tui"
	"github.com/urfave/cli/v2"
)

// statsTUICommand renders stats with the TUI interface
func statsTUICommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	playerFilter := c.StringSlice("player")
	showAll := c.Bool("all")

	// Load config to get configured users
	cfg, err := config.LoadOrDefault()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Open database connection
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Get game count
	count, err := database.GetGameCount(c.Context)
	if err != nil {
		return fmt.Errorf("failed to get game count: %w", err)
	}

	if count == 0 {
		fmt.Println("Database is empty")
		return nil
	}

	// Determine which players to show stats for
	var players []string
	if len(playerFilter) > 0 {
		players = playerFilter
	} else if !showAll {
		if cfg.ChessCom != nil && cfg.ChessCom.Username != "" {
			players = append(players, cfg.ChessCom.Username)
		}
		if cfg.Lichess != nil && cfg.Lichess.Username != "" {
			players = append(players, cfg.Lichess.Username)
		}

		if len(players) == 0 {
			fmt.Println("No configured users found. Use --all to show all players or configure users with 'gochess config add-user'")
			return nil
		}
	}

	// Get player statistics
	var stats []db.PlayerStats
	if len(players) > 0 {
		stats, err = database.GetPlayerStatsFiltered(c.Context, players)
	} else {
		stats, err = database.GetPlayerStats(c.Context)
	}
	if err != nil {
		return fmt.Errorf("failed to get player statistics: %w", err)
	}

	if len(stats) == 0 {
		if len(players) > 0 {
			fmt.Printf("No games found for players: %v\n", players)
		} else {
			fmt.Println("No player statistics available")
		}
		return nil
	}

	// Render player stats with TUI
	fmt.Println(tui.RenderPlayerStats(stats, count))

	// Get and display opening statistics
	var openingStats []db.OpeningStats
	if len(players) > 0 {
		openingStats, err = database.GetOpeningStatsFiltered(c.Context, players)
	} else {
		openingStats, err = database.GetOpeningStats(c.Context)
	}

	if err != nil {
		fmt.Printf("\nWarning: Failed to get opening statistics: %v\n", err)
	} else if len(openingStats) > 0 {
		playerName := ""
		if len(stats) == 1 {
			playerName = stats[0].Name
		}
		fmt.Println()
		fmt.Println(tui.RenderOpeningStats(openingStats, playerName, 10))
	}

	// Get and display position statistics
	uniqueCount, topPositions, err := database.GetPositionStats(c.Context)
	if err != nil {
		fmt.Printf("\nWarning: Failed to get position statistics: %v\n", err)
	} else if uniqueCount > 0 {
		fmt.Println()
		fmt.Println(tui.RenderPositionStats(uniqueCount, topPositions))
	}

	return nil
}
