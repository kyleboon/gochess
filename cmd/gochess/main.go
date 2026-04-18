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
	defaultDepth    = 18
	defaultLines    = 1
	defaultLogLevel = "info"
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
						Usage:   "Output format (table, csv, or tui)",
						Value:   "table",
					},
					&cli.BoolFlag{
						Name:  "tui",
						Usage: "Use pretty TUI output (same as --format=tui)",
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
								Aliases: []string{"p"},
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
				Usage: "Analyze chess positions with a UCI engine",
				Subcommands: []*cli.Command{
					{
						Name:  "position",
						Usage: "Analyze a single position",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "fen",
								Usage: "FEN string of the position to analyze",
							},
							&cli.IntFlag{
								Name:  "game-id",
								Usage: "Game ID to load position from database",
							},
							&cli.IntFlag{
								Name:  "move",
								Usage: "Ply (half-move) number within the game (used with --game-id)",
								Value: 0,
							},
							&cli.StringFlag{
								Name:    "engine",
								Aliases: []string{"e"},
								Usage:   "Path to UCI chess engine executable",
							},
							&cli.IntFlag{
								Name:    "depth",
								Aliases: []string{"d"},
								Usage:   "Analysis depth",
								Value:   defaultDepth,
							},
							&cli.IntFlag{
								Name:  "lines",
								Usage: "Number of principal variations (MultiPV)",
								Value: defaultLines,
							},
							&cli.BoolFlag{
								Name:  "save",
								Usage: "Save the evaluation to the database (requires --game-id)",
							},
						},
						Action: analyzePositionAction,
					},
				},
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
							&cli.BoolFlag{
								Name:  "tui",
								Usage: "Use interactive TUI browser",
							},
						},
						Action: listCommandRouter,
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

func statsCommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	playerFilter := c.StringSlice("player")
	showAll := c.Bool("all")
	format := c.String("format")
	useTUI := c.Bool("tui")

	// If --tui flag is set, use TUI format
	if useTUI {
		format = "tui"
	}

	// Route to TUI if requested
	if format == "tui" {
		return statsTUICommand(c)
	}

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
		fmt.Println("Name,Games,Wins,Losses,Draws,WinRate,WhiteGames,BlackGames,WhiteWins,BlackWins,WhiteWinRate,BlackWinRate,BulletGames,BlitzGames,RapidGames,ClassicalGames")
		for _, s := range stats {
			fmt.Printf("%s,%d,%d,%d,%d,%.1f%%,%d,%d,%d,%d,%.1f%%,%.1f%%,%d,%d,%d,%d\n",
				s.Name, s.Games, s.Wins, s.Losses, s.Draws, s.WinRate,
				s.WhiteGames, s.BlackGames, s.WhiteWins, s.BlackWins,
				s.WhiteWinRate, s.BlackWinRate,
				s.BulletGames, s.BlitzGames, s.RapidGames, s.ClassicalGames)
		}

	default:
		// Table output (default)
		fmt.Printf("%-20s %-6s %-6s %-6s %-6s %-8s %-8s %-8s\n",
			"PLAYER", "GAMES", "WINS", "LOSSES", "DRAWS", "WIN RATE", "AS WHITE", "AS BLACK")
		fmt.Println(repeatString("-", 88))

		for _, s := range stats {
			// Truncate long player names
			name := s.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}

			// Format win rates by color
			whiteRate := fmt.Sprintf("%.1f%%", s.WhiteWinRate)
			blackRate := fmt.Sprintf("%.1f%%", s.BlackWinRate)

			fmt.Printf("%-20s %-6d %-6d %-6d %-6d %-7.1f%% %-8s %-8s\n",
				name, s.Games, s.Wins, s.Losses, s.Draws, s.WinRate,
				whiteRate, blackRate)
		}

		// Show detailed statistics if only showing one player
		if len(stats) == 1 {
			s := stats[0]
			fmt.Printf("\nDetailed statistics for %s:\n", s.Name)

			// Win rates by color
			fmt.Printf("\n  Performance by Color:\n")
			fmt.Printf("    As White: %d games, %d-%d-%d (W-L-D), %.1f%% win rate\n",
				s.WhiteGames, s.WhiteWins, s.WhiteLosses, s.WhiteDraws, s.WhiteWinRate)
			fmt.Printf("    As Black: %d games, %d-%d-%d (W-L-D), %.1f%% win rate\n",
				s.BlackGames, s.BlackWins, s.BlackLosses, s.BlackDraws, s.BlackWinRate)

			// Time control breakdown
			if s.BulletGames > 0 || s.BlitzGames > 0 || s.RapidGames > 0 || s.ClassicalGames > 0 {
				fmt.Printf("\n  Games by Time Control:\n")
				if s.BulletGames > 0 {
					fmt.Printf("    Bullet:    %d games (%.1f%%)\n",
						s.BulletGames, float64(s.BulletGames)/float64(s.Games)*100)
				}
				if s.BlitzGames > 0 {
					fmt.Printf("    Blitz:     %d games (%.1f%%)\n",
						s.BlitzGames, float64(s.BlitzGames)/float64(s.Games)*100)
				}
				if s.RapidGames > 0 {
					fmt.Printf("    Rapid:     %d games (%.1f%%)\n",
						s.RapidGames, float64(s.RapidGames)/float64(s.Games)*100)
				}
				if s.ClassicalGames > 0 {
					fmt.Printf("    Classical: %d games (%.1f%%)\n",
						s.ClassicalGames, float64(s.ClassicalGames)/float64(s.Games)*100)
				}
			}
		}
	}

	// Get and display opening statistics (only in table format)
	if format == "table" {
		var openingStats []db.OpeningStats
		if len(players) > 0 {
			openingStats, err = database.GetOpeningStatsFiltered(c.Context, players)
		} else {
			openingStats, err = database.GetOpeningStats(c.Context)
		}

		if err != nil {
			fmt.Printf("\nWarning: Failed to get opening statistics: %v\n", err)
		} else if len(openingStats) > 0 {
			fmt.Printf("\nOpening Statistics:\n")

			// Show top 10 most played openings
			displayCount := 10
			if len(openingStats) < displayCount {
				displayCount = len(openingStats)
			}

			fmt.Printf("\n  Top %d Most Played Openings:\n", displayCount)
			fmt.Printf("  %-6s %-40s %-6s %-7s %-8s %-8s\n", "ECO", "OPENING", "GAMES", "WIN%%", "AS WHITE", "AS BLACK")
			fmt.Println("  " + repeatString("-", 90))

			for i := 0; i < displayCount; i++ {
				op := openingStats[i]
				// Truncate long opening names
				name := op.OpeningName
				if len(name) > 40 {
					name = name[:37] + "..."
				}

				whiteRate := fmt.Sprintf("%.1f%%", op.WhiteWinRate)
				blackRate := fmt.Sprintf("%.1f%%", op.BlackWinRate)

				fmt.Printf("  %-6s %-40s %-6d %-6.1f%% %-8s %-8s\n",
					op.ECOCode, name, op.Games, op.WinRate, whiteRate, blackRate)
			}

			// Show detailed stats for single player
			if len(stats) == 1 && len(openingStats) > 0 {
				s := stats[0]
				fmt.Printf("\n  Opening Performance for %s:\n", s.Name)

				// Best and worst openings (with minimum 3 games)
				var best, worst *db.OpeningStats
				for i := range openingStats {
					if openingStats[i].Games >= 3 {
						if best == nil || openingStats[i].WinRate > best.WinRate {
							best = &openingStats[i]
						}
						if worst == nil || openingStats[i].WinRate < worst.WinRate {
							worst = &openingStats[i]
						}
					}
				}

				if best != nil {
					fmt.Printf("    Best opening:  %s (%s) - %d games, %.1f%% win rate\n",
						best.ECOCode, best.OpeningName, best.Games, best.WinRate)
				}
				if worst != nil && worst.ECOCode != best.ECOCode {
					fmt.Printf("    Worst opening: %s (%s) - %d games, %.1f%% win rate\n",
						worst.ECOCode, worst.OpeningName, worst.Games, worst.WinRate)
				}
			}
		}
	}

	// Get and display position statistics (only in table format)
	if format == "table" {
		uniqueCount, topPositions, err := database.GetPositionStats(c.Context)
		if err != nil {
			fmt.Printf("\nWarning: Failed to get position statistics: %v\n", err)
		} else if uniqueCount > 0 {
			fmt.Printf("\nPosition Statistics:\n")
			fmt.Printf("  Unique positions: %d\n", uniqueCount)

			if len(topPositions) > 0 {
				fmt.Printf("\n  Top 10 Most Common Positions (after move 10):\n")
				fmt.Printf("  %-6s %-6s %-6s %-6s %-6s %-30s %s\n", "COUNT", "WHITE%", "BLACK%", "DRAW%", "ECO", "OPENING", "FEN")
				fmt.Println("  " + repeatString("-", 110))

				for _, pos := range topPositions {
					// Truncate very long FEN strings for display
					fen := pos.FEN
					if len(fen) > 40 {
						fen = fen[:37] + "..."
					}

					// Truncate long opening names
					opening := pos.OpeningName
					if len(opening) > 30 {
						opening = opening[:27] + "..."
					}

					// Default to "-" if ECO code is not available
					eco := pos.ECOCode
					if eco == "" {
						eco = "-"
					}
					if opening == "" {
						opening = "-"
					}

					fmt.Printf("  %-6d %-6.1f %-6.1f %-6.1f %-6s %-30s %s\n",
						pos.Count, pos.WhiteWinPct, pos.BlackWinPct, pos.DrawPct, eco, opening, fen)
				}
			}
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

// listCommandRouter routes to either TUI or normal list command
func listCommandRouter(c *cli.Context) error {
	if c.Bool("tui") {
		return gameListTUICommand(c)
	}
	return db.ListCommand(c)
}
