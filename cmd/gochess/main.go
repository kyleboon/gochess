package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kyleboon/gochess/internal/chesscom"
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
