package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

type config struct {
	pgnPath     string
	enginePath  string
	depth       int
	timePerMove int
	threads     int
	inaccuracy  int
	mistake     int
	blunder     int
	logLevel    string
}

func parseFlags() *config {
	cfg := &config{}

	flag.StringVar(&cfg.pgnPath, "pgn", "", "Path to PGN file (required)")
	flag.StringVar(&cfg.enginePath, "engine", "", "Path to UCI chess engine executable (optional, default: search in PATH)")
	flag.IntVar(&cfg.depth, "depth", defaultDepth, "Minimum analysis depth")
	flag.IntVar(&cfg.timePerMove, "time", defaultTimePerMove, "Maximum time per move in seconds")
	flag.IntVar(&cfg.threads, "threads", defaultThreads, "Number of CPU threads")
	flag.IntVar(&cfg.inaccuracy, "inaccuracy", defaultInaccuracy, "Centipawn threshold for inaccuracies")
	flag.IntVar(&cfg.mistake, "mistake", defaultMistake, "Centipawn threshold for mistakes")
	flag.IntVar(&cfg.blunder, "blunder", defaultBlunder, "Centipawn threshold for blunders")
	flag.StringVar(&cfg.logLevel, "log", defaultLogLevel, "Log level (info, debug, trace)")

	flag.Parse()

	if cfg.pgnPath == "" {
		fmt.Fprintln(os.Stderr, "Error: PGN file path is required")
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func main() {
	cfg := parseFlags()

	log.SetPrefix("[gochess] ")

	// TODO: Implement the analysis pipeline:
	// 1. Parse PGN file
	// 2. Initialize engine
	// 3. Process games
	// 4. Output annotated PGN

	fmt.Printf("Analyzing PGN file: %s\n", cfg.pgnPath)
	fmt.Printf("Using engine: %s\n", cfg.enginePath)

	// Just for demonstration
	fmt.Println("\n=== Running Example ===")
}
