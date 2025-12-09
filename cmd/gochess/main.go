package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kyleboon/gochess/internal/chesscom"
	"github.com/kyleboon/gochess/internal/config"
	"github.com/kyleboon/gochess/internal/db"
	"github.com/kyleboon/gochess/internal/lichess"
	"github.com/urfave/cli/v2"
)

const (
	defaultDepth       = 18
	defaultTimePerMove = 3
	defaultThreads     = 1
	defaultInaccuracy  = 20
	defaultMistake     = 50
	defaultBlunder     = 100
	defaultLogLevel    = "info"
)

func main() {
	app := &cli.App{
		Name:  "gochess",
		Usage: "Chess utilities and analysis tools",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set log level (debug, info, warn, error)",
				Value:   defaultLogLevel,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "import",
				Usage: "Import games from all configured sources",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Show detailed error messages",
					},
					&cli.BoolFlag{
						Name:    "full",
						Aliases: []string{"f"},
						Usage:   "Import full history (ignore last import time)",
					},
				},
				Action: ImportCommand,
			},
			{
				Name:    "stats",
				Aliases: []string{"st"},
				Usage:   "Show statistics for configured players",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "database",
						Aliases: []string{"db"},
						Usage:   "Path to database file",
						Value:   "~/.gochess/games.db",
					},
					&cli.StringSliceFlag{
						Name:    "player",
						Aliases: []string{"p"},
						Usage:   "Filter statistics for specific player(s) (can be used multiple times)",
					},
					&cli.BoolFlag{
						Name:  "all",
						Usage: "Show statistics for all players in database (not just configured users)",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Usage:   "Output format (table or csv)",
						Value:   "table",
					},
				},
				Action: statsCommand,
			},
			{
				Name:  "chesscom",
				Usage: "Interact with Chess.com API",
				Subcommands: []*cli.Command{
					{
						Name:  "archives",
						Usage: "List available archives for a user",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "username",
								Aliases:  []string{"u"},
								Usage:    "Chess.com username",
								Required: true,
							},
						},
						Action: chesscom.ListArchives,
					},
					{
						Name:  "download",
						Usage: "Download games for a user",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "username",
								Aliases:  []string{"u"},
								Usage:    "Chess.com username",
								Required: true,
							},
							&cli.IntFlag{
								Name:    "year",
								Aliases: []string{"y"},
								Usage:   "Year of games to download",
								Value:   time.Now().Year(),
							},
							&cli.IntFlag{
								Name:    "month",
								Aliases: []string{"m"},
								Usage:   "Month of games to download (1-12)",
								Value:   int(time.Now().Month()),
							},
							&cli.StringFlag{
								Name:    "format",
								Aliases: []string{"f"},
								Usage:   "Output format (pgn, json, or summary)",
								Value:   "summary",
							},
							&cli.StringFlag{
								Name:    "output",
								Aliases: []string{"o"},
								Usage:   "Output file path (default: stdout)",
							},
							&cli.BoolFlag{
								Name:  "import-db",
								Usage: "Import games directly into the database",
							},
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file (for import-db option)",
								Value:   "~/.gochess/games.db",
							},
							&cli.BoolFlag{
								Name:    "verbose",
								Aliases: []string{"v"},
								Usage:   "Show detailed error messages",
							},
							&cli.BoolFlag{
								Name:    "all-history",
								Aliases: []string{"a"},
								Usage:   "Download all available game history (ignores year/month options)",
							},
						},
						Action: chesscom.DownloadGames,
					},
				},
			},
			{
				Name:  "lichess",
				Usage: "Interact with Lichess API",
				Subcommands: []*cli.Command{
					{
						Name:  "download",
						Usage: "Download games for a user",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "username",
								Aliases:  []string{"u"},
								Usage:    "Lichess username",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "since",
								Aliases: []string{"s"},
								Usage:   "Download games since this date (YYYY-MM-DD, YYYY-MM, or YYYY)",
							},
							&cli.StringFlag{
								Name:    "until",
								Usage:   "Download games until this date (YYYY-MM-DD, YYYY-MM, or YYYY)",
							},
							&cli.IntFlag{
								Name:    "max",
								Aliases: []string{"n"},
								Usage:   "Maximum number of games to download",
							},
							&cli.StringFlag{
								Name:  "vs",
								Usage: "Filter games against a specific opponent",
							},
							&cli.StringFlag{
								Name:  "rated",
								Usage: "Filter by rated games (true/false)",
							},
							&cli.StringFlag{
								Name:  "perf-type",
								Usage: "Filter by game type (ultraBullet, bullet, blitz, rapid, classical, correspondence)",
							},
							&cli.StringFlag{
								Name:  "color",
								Usage: "Filter by color (white/black)",
							},
							&cli.StringFlag{
								Name:    "output",
								Aliases: []string{"o"},
								Usage:   "Output file path (default: stdout)",
							},
							&cli.BoolFlag{
								Name:  "import-db",
								Usage: "Import games directly into the database",
							},
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file (for import-db option)",
								Value:   "~/.gochess/games.db",
							},
							&cli.StringFlag{
								Name:  "api-token",
								Usage: "Lichess API token for private games (optional)",
							},
							&cli.BoolFlag{
								Name:    "verbose",
								Aliases: []string{"v"},
								Usage:   "Show detailed error messages",
							},
						},
						Action: lichess.DownloadGames,
					},
				},
			},
			{
				Name:  "config",
				Usage: "Manage gochess configuration",
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "Initialize configuration interactively",
						Action: config.InitCommand,
					},
					{
						Name:  "show",
						Usage: "Show current configuration",
						Action: config.ShowCommand,
					},
					{
						Name:  "add-user",
						Usage: "Add a user to track",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "platform",
								Aliases:  []string{"p"},
								Usage:    "Platform (chesscom or lichess)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "username",
								Aliases:  []string{"u"},
								Usage:    "Username to track",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "token",
								Aliases: []string{"t"},
								Usage:   "API token (Lichess only, optional)",
							},
						},
						Action: config.AddUserCommand,
					},
					{
						Name:  "remove-user",
						Usage: "Remove a tracked user",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "platform",
								Aliases:  []string{"p"},
								Usage:    "Platform (chesscom or lichess)",
								Required: true,
							},
						},
						Action: config.RemoveUserCommand,
					},
				},
			},
			{
				Name:  "analyze",
				Usage: "Analyze a PGN file with a chess engine",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "pgn",
						Aliases:  []string{"p"},
						Usage:    "Path to PGN file (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "engine",
						Aliases: []string{"e"},
						Usage:   "Path to UCI chess engine executable (optional, default: search in PATH)",
					},
					&cli.IntFlag{
						Name:    "depth",
						Aliases: []string{"d"},
						Usage:   "Minimum analysis depth",
						Value:   defaultDepth,
					},
					&cli.IntFlag{
						Name:    "time",
						Aliases: []string{"t"},
						Usage:   "Maximum time per move in seconds",
						Value:   defaultTimePerMove,
					},
					&cli.IntFlag{
						Name:  "threads",
						Usage: "Number of CPU threads",
						Value: defaultThreads,
					},
					&cli.IntFlag{
						Name:  "inaccuracy",
						Usage: "Centipawn threshold for inaccuracies",
						Value: defaultInaccuracy,
					},
					&cli.IntFlag{
						Name:  "mistake",
						Usage: "Centipawn threshold for mistakes",
						Value: defaultMistake,
					},
					&cli.IntFlag{
						Name:  "blunder",
						Usage: "Centipawn threshold for blunders",
						Value: defaultBlunder,
					},
					&cli.StringFlag{
						Name:    "log",
						Aliases: []string{"l"},
						Usage:   "Log level (info, debug, trace)",
						Value:   defaultLogLevel,
					},
				},
				Action: analyzeAction,
			},
			{
				Name:  "db",
				Usage: "Manage PGN database",
				Subcommands: []*cli.Command{
					{
						Name:  "import",
						Usage: "Import PGN files into the database",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "pgn",
								Aliases:  []string{"p"},
								Usage:    "Path to PGN file or directory of PGN files",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file",
								Value:   "~/.gochess/games.db",
							},
							&cli.BoolFlag{
								Name:    "verbose",
								Aliases: []string{"v"},
								Usage:   "Show detailed error messages",
							},
						},
						Action: db.ImportCommand,
					},
					{
						Name:  "list",
						Usage: "List games in the database",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file",
								Value:   "~/.gochess/games.db",
							},
							&cli.StringFlag{
								Name:    "white",
								Aliases: []string{"w"},
								Usage:   "Filter by white player",
							},
							&cli.StringFlag{
								Name:    "black",
								Aliases: []string{"b"},
								Usage:   "Filter by black player",
							},
							&cli.StringFlag{
								Name:    "event",
								Aliases: []string{"e"},
								Usage:   "Filter by event",
							},
							&cli.StringFlag{
								Name:    "date",
								Aliases: []string{"d"},
								Usage:   "Filter by date",
							},
							&cli.IntFlag{
								Name:    "limit",
								Aliases: []string{"n"},
								Usage:   "Maximum number of results",
								Value:   20,
							},
							&cli.IntFlag{
								Name:  "offset",
								Usage: "Result offset (for pagination)",
								Value: 0,
							},
						},
						Action: db.ListCommand,
					},
					{
						Name:  "show",
						Usage: "Show details of a specific game",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:     "id",
								Usage:    "Game ID",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file",
								Value:   "~/.gochess/games.db",
							},
							&cli.BoolFlag{
								Name:    "pgn",
								Usage:   "Show PGN text",
								Value:   true,
							},
						},
						Action: db.ShowCommand,
					},
					{
						Name:  "export",
						Usage: "Export games to PGN format",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file",
								Value:   "~/.gochess/games.db",
							},
							&cli.IntFlag{
								Name:  "id",
								Usage: "Export specific game by ID (if not specified, export all games)",
							},
							&cli.StringFlag{
								Name:    "output",
								Aliases: []string{"o"},
								Usage:   "Output file path (default: stdout)",
							},
						},
						Action: db.ExportCommand,
					},
					{
						Name:    "clear",
						Aliases: []string{"c"},
						Usage:   "Clear all games from the database",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file",
								Value:   "~/.gochess/games.db",
							},
							&cli.BoolFlag{
								Name:    "force",
								Aliases: []string{"f"},
								Usage:   "Clear without confirmation prompt",
							},
						},
						Action: db.ClearCommand,
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func analyzeAction(c *cli.Context) error {
	pgnPath := c.String("pgn")
	enginePath := c.String("engine")
	depth := c.Int("depth")
	timePerMove := c.Int("time")
	threads := c.Int("threads")
	inaccuracy := c.Int("inaccuracy")
	mistake := c.Int("mistake")
	blunder := c.Int("blunder")
	logLevel := c.String("log")

	log.SetPrefix("[gochess] ")

	// TODO: Implement the analysis pipeline:
	// 1. Parse PGN file
	// 2. Initialize engine
	// 3. Process games
	// 4. Output annotated PGN

	fmt.Printf("Analyzing PGN file: %s\n", pgnPath)
	fmt.Printf("Using engine: %s\n", enginePath)
	fmt.Printf("Settings: depth=%d, time=%ds, threads=%d, log=%s\n",
		depth, timePerMove, threads, logLevel)
	fmt.Printf("Thresholds: inaccuracy=%d, mistake=%d, blunder=%d\n",
		inaccuracy, mistake, blunder)

	// Just for demonstration
	fmt.Println("\n=== Running Example ===")

	return nil
}

func statsCommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	playerFilter := c.StringSlice("player")
	showAll := c.Bool("all")
	format := c.String("format")

	// Load config to get configured users
	cfg, err := config.LoadOrDefault()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Open database connection
	fmt.Printf("Opening database at %s...\n", dbPath)
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

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
		// User explicitly specified players
		players = playerFilter
	} else if !showAll {
		// Use configured users by default
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
	// else showAll is true, so players remains nil/empty and we get all players

	// Get player statistics
	fmt.Println("Calculating player statistics...")
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

	// Display statistics
	fmt.Printf("Database contains %d games\n\n", count)

	switch format {
	case "csv":
		// CSV output
		fmt.Println("Name,Games,Wins,Losses,Draws,WinRate,WhiteGames,BlackGames,WhiteWins,BlackWins")
		for _, s := range stats {
			fmt.Printf("%s,%d,%d,%d,%d,%.1f%%,%d,%d,%d,%d\n",
				s.Name, s.Games, s.Wins, s.Losses, s.Draws, s.WinRate,
				s.WhiteGames, s.BlackGames, s.WhiteWins, s.BlackWins)
		}

	default:
		// Table output (default)
		fmt.Printf("%-20s %-6s %-6s %-6s %-6s %-8s %-6s %-6s\n",
			"PLAYER", "GAMES", "WINS", "LOSSES", "DRAWS", "WIN RATE", "WHITE", "BLACK")
		fmt.Println(repeatString("-", 72))

		for _, s := range stats {
			// Truncate long player names
			name := s.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}

			fmt.Printf("%-20s %-6d %-6d %-6d %-6d %-7.1f%% %-6d %-6d\n",
				name, s.Games, s.Wins, s.Losses, s.Draws, s.WinRate,
				s.WhiteGames, s.BlackGames)
		}

		// Show detailed statistics if only showing one player
		if len(stats) == 1 {
			fmt.Printf("\nDetailed statistics for %s:\n", stats[0].Name)
			fmt.Printf("  As White: %d games, %d wins (%.1f%%)\n",
				stats[0].WhiteGames, stats[0].WhiteWins,
				safeDiv(float64(stats[0].WhiteWins), float64(stats[0].WhiteGames))*100)
			fmt.Printf("  As Black: %d games, %d wins (%.1f%%)\n",
				stats[0].BlackGames, stats[0].BlackWins,
				safeDiv(float64(stats[0].BlackWins), float64(stats[0].BlackGames))*100)
		}
	}

	return nil
}

// expandPath expands the tilde in file paths to the user's home directory
func expandPath(path string) string {
	if path == "" {
		return ""
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// repeatString repeats a string n times
func repeatString(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// safeDiv performs division but handles division by zero gracefully
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
