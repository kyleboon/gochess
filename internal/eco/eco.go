package eco

import (
	"bufio"
	_ "embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/kyleboon/gochess/internal/logging"
)

// Opening represents a chess opening classification
type Opening struct {
	ECOCode string
	Name    string
	PGN     string
	Moves   []string // Parsed PGN moves
}

// Database holds all ECO opening classifications
type Database struct {
	openings []Opening
	logger   *slog.Logger
}

//go:embed data/a.tsv
var aTSV string

//go:embed data/b.tsv
var bTSV string

//go:embed data/c.tsv
var cTSV string

//go:embed data/d.tsv
var dTSV string

//go:embed data/e.tsv
var eTSV string

// NewDatabase creates a new ECO database with embedded opening data
func NewDatabase() (*Database, error) {
	return NewDatabaseWithLogger(logging.Default())
}

// NewDatabaseWithLogger creates a new ECO database with a custom logger
func NewDatabaseWithLogger(logger *slog.Logger) (*Database, error) {
	db := &Database{
		openings: make([]Opening, 0, 4000),
		logger:   logger,
	}

	// Load all embedded TSV files
	files := []string{aTSV, bTSV, cTSV, dTSV, eTSV}
	for _, content := range files {
		if err := db.loadTSV(content); err != nil {
			return nil, err
		}
	}

	// Sort by move count (descending) - longer sequences first for accurate matching
	sort.Slice(db.openings, func(i, j int) bool {
		return len(db.openings[i].Moves) > len(db.openings[j].Moves)
	})

	logger.Debug("ECO database loaded", "count", len(db.openings))
	return db, nil
}

// loadTSV parses a TSV content string and adds openings to the database
func (db *Database) loadTSV(content string) error {
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Skip header line
	if !scanner.Scan() {
		return fmt.Errorf("empty TSV content")
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			db.logger.Warn("invalid TSV line", "line", line)
			continue
		}

		opening := Opening{
			ECOCode: parts[0],
			Name:    parts[1],
			PGN:     parts[2],
			Moves:   parsePGNMoves(parts[2]),
		}

		db.openings = append(db.openings, opening)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading TSV: %w", err)
	}

	return nil
}

// parsePGNMoves converts PGN notation to a list of moves
// Example: "1. e4 e5 2. Nf3" -> ["e4", "e5", "Nf3"]
func parsePGNMoves(pgn string) []string {
	moves := make([]string, 0)
	parts := strings.Fields(pgn)

	for _, part := range parts {
		// Skip move numbers (e.g., "1.", "2.")
		if strings.HasSuffix(part, ".") {
			continue
		}
		// Remove check/checkmate symbols
		move := strings.TrimRight(part, "+#")
		if move != "" {
			moves = append(moves, move)
		}
	}

	return moves
}

// Classify finds the best matching ECO opening for a sequence of moves
// Returns the ECO code, opening name, and whether a match was found
func (db *Database) Classify(moves []string) (string, string, bool) {
	if len(moves) == 0 {
		return "", "", false
	}

	// Find longest matching opening
	// Openings are already sorted by move count (descending)
	for _, opening := range db.openings {
		if matchesMoves(opening.Moves, moves) {
			return opening.ECOCode, opening.Name, true
		}
	}

	return "", "", false
}

// matchesMoves checks if the opening moves match the beginning of the game moves
func matchesMoves(openingMoves, gameMoves []string) bool {
	if len(openingMoves) > len(gameMoves) {
		return false
	}

	for i, move := range openingMoves {
		if !movesEqual(move, gameMoves[i]) {
			return false
		}
	}

	return true
}

// movesEqual compares two moves for equality, handling algebraic notation variations
func movesEqual(a, b string) bool {
	// Normalize by removing annotation symbols
	a = strings.TrimRight(a, "+#!?")
	b = strings.TrimRight(b, "+#!?")

	return a == b
}

// GetOpening retrieves an opening by ECO code
func (db *Database) GetOpening(ecoCode string) (Opening, bool) {
	for _, opening := range db.openings {
		if opening.ECOCode == ecoCode {
			return opening, true
		}
	}
	return Opening{}, false
}

// GetByName searches for openings by name (case-insensitive substring match)
func (db *Database) GetByName(name string) []Opening {
	name = strings.ToLower(name)
	results := make([]Opening, 0)

	for _, opening := range db.openings {
		if strings.Contains(strings.ToLower(opening.Name), name) {
			results = append(results, opening)
		}
	}

	return results
}
