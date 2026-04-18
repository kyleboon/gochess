package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kyleboon/gochess/internal/chesscom"
	"github.com/kyleboon/gochess/internal/config"
	"github.com/kyleboon/gochess/internal/db"
	"github.com/kyleboon/gochess/internal/lichess"
	"github.com/kyleboon/gochess/internal/logging"
	"github.com/urfave/cli/v2"
)

// ImportCommand is the unified import command that imports from all configured sources
func ImportCommand(c *cli.Context) error {
	verbose := c.Bool("verbose")
	full := c.Bool("full")

	// Load configuration
	cfg, err := config.LoadOrDefault()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine log level: CLI flag takes precedence over config file
	// If the flag was set explicitly, use it; otherwise use config
	logLevel := cfg.GetLogLevel()
	if c.IsSet("log-level") {
		logLevel = c.String("log-level")
	}

	// Create logger with the determined log level
	logger := createLogger(logLevel)

	// Check if any sources are configured
	if !cfg.HasAnySource() {
		fmt.Println("No game sources configured.")
		fmt.Println("Run 'gochess config init' to set up your configuration.")
		return nil
	}

	// If --full is specified, clear last import times
	if full {
		fmt.Println("Full import requested - fetching all available games...")
		cfg.LastImport = make(map[string]time.Time)
	}

	// Open database
	fmt.Printf("Opening database at %s...\n", cfg.DatabasePath)
	database, err := db.NewWithLogger(cfg.DatabasePath, logger)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	totalGames := 0
	hasErrors := false

	// Import from Chess.com if configured
	if cfg.ChessCom != nil && cfg.ChessCom.Username != "" {
		fmt.Println("\n=== Importing from Chess.com ===")
		count, err := chesscom.ImportFromConfig(c.Context, cfg, database, logger, verbose)
		if err != nil {
			fmt.Printf("Error importing from Chess.com: %v\n", err)
			hasErrors = true
		} else {
			totalGames += count
		}
	}

	// Import from Lichess if configured
	if cfg.Lichess != nil && cfg.Lichess.Username != "" {
		fmt.Println("\n=== Importing from Lichess ===")
		count, err := lichess.ImportFromConfig(c.Context, cfg, database, logger, verbose)
		if err != nil {
			fmt.Printf("Error importing from Lichess: %v\n", err)
			hasErrors = true
		} else {
			totalGames += count
		}
	}

	// Get current game count in database
	currentCount, err := database.GetGameCount(c.Context)
	if err == nil {
		fmt.Printf("\n=== Import Summary ===\n")
		fmt.Printf("Games imported this session: %d\n", totalGames)
		fmt.Printf("Total games in database: %d\n", currentCount)
	}

	if hasErrors {
		fmt.Println("\nSome imports failed. Use --verbose to see more details.")
		return fmt.Errorf("some imports failed")
	}

	if totalGames == 0 {
		fmt.Println("\nNo new games to import.")
	}

	return nil
}

// createLogger creates a logger with the specified log level
func createLogger(logLevelStr string) *slog.Logger {
	var level logging.Level
	switch logLevelStr {
	case "debug":
		level = logging.LevelDebug
	case "warn":
		level = logging.LevelWarn
	case "error":
		level = logging.LevelError
	default:
		level = logging.LevelInfo
	}
	return logging.NewWithLevel(level)
}
