package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kyleboon/gochess/internal/chesscom"
	"github.com/kyleboon/gochess/internal/db"
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
		Commands: []*cli.Command{
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
					{
						Name:    "stats",
						Aliases: []string{"st"},
						Usage:   "Show win rate statistics for players in the database",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "database",
								Aliases: []string{"db"},
								Usage:   "Path to database file",
								Value:   "~/.gochess/games.db",
							},
							&cli.StringFlag{
								Name:    "player",
								Aliases: []string{"p"},
								Usage:   "Filter statistics for a specific player",
							},
							&cli.StringFlag{
								Name:    "format",
								Aliases: []string{"f"},
								Usage:   "Output format (table or csv)",
								Value:   "table",
							},
						},
						Action: db.StatsCommand,
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
