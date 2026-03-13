package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// AnalysisOptions configures the engine analysis.
type AnalysisOptions struct {
	Depth   int // search depth (default 20)
	MultiPV int // number of lines to report (default 1)
}

// Score represents an engine evaluation score.
type Score struct {
	Centipawns int
	Mate       int
	IsMate     bool
}

// String returns a human-readable score string.
func (s Score) String() string {
	if s.IsMate {
		if s.Mate > 0 {
			return fmt.Sprintf("#%d", s.Mate)
		}
		return fmt.Sprintf("#%d", s.Mate)
	}
	sign := "+"
	cp := s.Centipawns
	if cp < 0 {
		sign = "-"
		cp = -cp
	}
	return fmt.Sprintf("%s%d.%02d", sign, cp/100, cp%100)
}

// AnalysisLine represents a single principal variation from the engine.
type AnalysisLine struct {
	Rank  int      // MultiPV rank (1-based)
	Score Score    // evaluation score
	Depth int      // search depth reached
	Moves []string // principal variation moves (UCI notation)
	Nodes int64    // nodes searched
	NPS   int64    // nodes per second
}

// AnalysisResult holds the complete analysis output.
type AnalysisResult struct {
	FEN   string
	Lines []AnalysisLine
	Depth int
}

// Analyze runs a position analysis and returns the result.
func (e *Engine) Analyze(ctx context.Context, fen string, opts AnalysisOptions) (*AnalysisResult, error) {
	if opts.Depth <= 0 {
		opts.Depth = 20
	}
	if opts.MultiPV <= 0 {
		opts.MultiPV = 1
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Set MultiPV if more than 1 line requested
	if opts.MultiPV > 1 {
		if err := e.sendLocked(fmt.Sprintf("setoption name MultiPV value %d", opts.MultiPV)); err != nil {
			return nil, err
		}
	}

	// Send position
	if err := e.sendLocked(fmt.Sprintf("position fen %s", fen)); err != nil {
		return nil, err
	}

	// Start search
	if err := e.sendLocked(fmt.Sprintf("go depth %d", opts.Depth)); err != nil {
		return nil, err
	}

	// Read until "bestmove"
	lines, err := e.readUntilLocked(ctx, "bestmove")
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Determine if we need to flip the score (engine reports from side-to-move's
	// perspective, but the convention is to show from White's perspective).
	blackToMove := false
	if fields := strings.Fields(fen); len(fields) >= 2 && fields[1] == "b" {
		blackToMove = true
	}

	// Parse info lines — keep only the deepest entry per MultiPV rank
	best := make(map[int]*AnalysisLine) // rank -> best line at target depth
	for _, raw := range lines {
		al, err := parseInfoLine(raw)
		if err != nil || al == nil {
			continue
		}
		prev, ok := best[al.Rank]
		if !ok || al.Depth >= prev.Depth {
			best[al.Rank] = al
		}
	}

	// Normalize scores to White's perspective
	if blackToMove {
		for _, al := range best {
			if al.Score.IsMate {
				al.Score.Mate = -al.Score.Mate
			} else {
				al.Score.Centipawns = -al.Score.Centipawns
			}
		}
	}

	// Collect results ordered by rank
	result := &AnalysisResult{
		FEN:   fen,
		Depth: opts.Depth,
	}
	for rank := 1; rank <= opts.MultiPV; rank++ {
		if al, ok := best[rank]; ok {
			result.Lines = append(result.Lines, *al)
		}
	}

	return result, nil
}

// parseInfoLine parses a UCI "info" line into an AnalysisLine.
// Returns nil, nil for non-info lines (e.g. "bestmove").
func parseInfoLine(line string) (*AnalysisLine, error) {
	if !strings.HasPrefix(line, "info ") {
		return nil, nil
	}

	tokens := strings.Fields(line)
	al := &AnalysisLine{Rank: 1} // default rank if multipv not present

	for i := 1; i < len(tokens); i++ {
		switch tokens[i] {
		case "depth":
			if i+1 < len(tokens) {
				i++
				v, err := strconv.Atoi(tokens[i])
				if err != nil {
					return nil, fmt.Errorf("parse depth: %w", err)
				}
				al.Depth = v
			}
		case "multipv":
			if i+1 < len(tokens) {
				i++
				v, err := strconv.Atoi(tokens[i])
				if err != nil {
					return nil, fmt.Errorf("parse multipv: %w", err)
				}
				al.Rank = v
			}
		case "score":
			if i+1 < len(tokens) {
				i++
				switch tokens[i] {
				case "cp":
					if i+1 < len(tokens) {
						i++
						v, err := strconv.Atoi(tokens[i])
						if err != nil {
							return nil, fmt.Errorf("parse score cp: %w", err)
						}
						al.Score = Score{Centipawns: v}
					}
				case "mate":
					if i+1 < len(tokens) {
						i++
						v, err := strconv.Atoi(tokens[i])
						if err != nil {
							return nil, fmt.Errorf("parse score mate: %w", err)
						}
						al.Score = Score{Mate: v, IsMate: true}
					}
				}
			}
		case "nodes":
			if i+1 < len(tokens) {
				i++
				v, err := strconv.ParseInt(tokens[i], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parse nodes: %w", err)
				}
				al.Nodes = v
			}
		case "nps":
			if i+1 < len(tokens) {
				i++
				v, err := strconv.ParseInt(tokens[i], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("parse nps: %w", err)
				}
				al.NPS = v
			}
		case "pv":
			// Everything after "pv" is the move list
			al.Moves = tokens[i+1:]
			i = len(tokens) // break out of loop
		}
	}

	// Skip info lines without depth (e.g. "info string ...")
	if al.Depth == 0 {
		return nil, nil
	}

	return al, nil
}
