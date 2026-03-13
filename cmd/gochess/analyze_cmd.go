package main

import (
	"fmt"

	"github.com/kyleboon/gochess/internal/config"
	"github.com/kyleboon/gochess/internal/db"
	"github.com/kyleboon/gochess/internal/engine"
	"github.com/kyleboon/gochess/internal/logging"
	"github.com/urfave/cli/v2"
)

func analyzePositionAction(c *cli.Context) error {
	fen := c.String("fen")
	gameID := c.Int("game-id")
	moveNumber := c.Int("move")
	enginePath := c.String("engine")
	depth := c.Int("depth")
	lines := c.Int("lines")
	save := c.Bool("save")

	// Determine log level
	logLevel := logging.LevelError
	if c.IsSet("log-level") {
		logLevel = logging.Level(c.String("log-level"))
	}
	logger := logging.NewWithLevel(logLevel)

	// Load config for defaults
	cfg, err := config.LoadOrDefault()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve engine path: flag > config > error
	if enginePath == "" {
		enginePath = cfg.GetEnginePath()
	}
	if enginePath == "" {
		return fmt.Errorf("engine path required: use --engine flag or configure with 'gochess config init'")
	}

	// Resolve engine options from config
	var engineOpts engine.Options
	if cfg.Engine != nil {
		engineOpts.Threads = cfg.Engine.Threads
		engineOpts.Hash = cfg.Engine.Hash
	}

	// Resolve FEN: --fen flag or --game-id + --move from DB
	var gamePos *db.GamePosition
	if fen == "" {
		if gameID <= 0 {
			return fmt.Errorf("either --fen or --game-id is required")
		}

		dbPath := expandPath(cfg.DatabasePath)
		database, err := db.NewWithLogger(dbPath, logger)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		gamePos, err = database.GetPositionByGameAndMove(c.Context, gameID, moveNumber)
		if err != nil {
			return fmt.Errorf("failed to get position: %w", err)
		}
		fen = gamePos.FEN
	}

	// Print game info if loaded from DB
	if gamePos != nil {
		fmt.Printf("Game: %s vs %s (%s, %s)\n", gamePos.White, gamePos.Black, gamePos.Event, gamePos.Date)
		fmt.Printf("Position at ply %d\n", gamePos.MoveNumber)
	}
	fmt.Printf("FEN: %s\n", fen)

	// Start engine
	fmt.Printf("\nAnalyzing at depth %d with %d line(s)...\n", depth, lines)

	eng, err := engine.NewWithOptions(c.Context, enginePath, logger, engineOpts)
	if err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}
	defer eng.Close()

	// Run analysis
	result, err := eng.Analyze(c.Context, fen, engine.AnalysisOptions{
		Depth:   depth,
		MultiPV: lines,
	})
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Display results
	fmt.Printf("\nAnalysis (depth %d):\n\n", result.Depth)
	for _, line := range result.Lines {
		moves := ""
		if len(line.Moves) > 5 {
			moves = fmt.Sprintf("%s", joinMoves(line.Moves[:5]))
		} else {
			moves = fmt.Sprintf("%s", joinMoves(line.Moves))
		}
		fmt.Printf("  %d. %-8s %s\n", line.Rank, line.Score.String(), moves)
	}

	// Optionally save evaluation to DB
	if save && gamePos != nil && len(result.Lines) > 0 {
		dbPath := expandPath(cfg.DatabasePath)
		database, err := db.NewWithLogger(dbPath, logger)
		if err != nil {
			return fmt.Errorf("failed to open database for saving: %w", err)
		}
		defer database.Close()

		score := result.Lines[0].Score
		var eval float64
		if score.IsMate {
			if score.Mate > 0 {
				eval = 999.0
			} else {
				eval = -999.0
			}
		} else {
			eval = float64(score.Centipawns) / 100.0
		}

		if err := database.UpdatePositionEvaluation(c.Context, gamePos.PositionID, eval); err != nil {
			return fmt.Errorf("failed to save evaluation: %w", err)
		}
		fmt.Printf("\nEvaluation %.2f saved to database.\n", eval)
	}

	return nil
}

func joinMoves(moves []string) string {
	result := ""
	for i, m := range moves {
		if i > 0 {
			result += " "
		}
		result += m
	}
	return result
}
