package chesscom

import (
	"fmt"
	"strings"

	"github.com/kyleboon/gochess/internal/pgn"
)

// GamesToDatabase converts Chess.com games to a pgn.DB.
func GamesToDatabase(games *GamesResponse) (*pgn.DB, []error) {
	db := &pgn.DB{}
	var errs []error

	for _, game := range games.Games {
		pgnData := game.PGN
		if err := db.Parse(pgnData); err != nil && len(err) > 0 {
			errs = append(errs, fmt.Errorf("failed to parse game PGN: %v", err))
		}
	}

	// Parse moves for all games
	for _, game := range db.Games {
		if err := db.ParseMoves(game); err != nil {
			errs = append(errs, fmt.Errorf("failed to parse moves: %v", err))
		}
	}

	return db, errs
}

// PGNToDatabase converts a PGN string (containing multiple games) to a pgn.DB.
func PGNToDatabase(pgnData string) (*pgn.DB, []error) {
	db := &pgn.DB{}
	errs := db.Parse(pgnData)

	// Parse moves for all games
	for _, game := range db.Games {
		if err := db.ParseMoves(game); err != nil {
			errs = append(errs, fmt.Errorf("failed to parse moves: %v", err))
		}
	}

	return db, errs
}

// FilterGames filters games by various criteria.
func FilterGames(db *pgn.DB, options FilterOptions) *pgn.DB {
	result := &pgn.DB{}

	for _, game := range db.Games {
		if shouldIncludeGame(game, options) {
			result.Games = append(result.Games, game)
		}
	}

	return result
}

// FilterOptions defines criteria for filtering games.
type FilterOptions struct {
	WhitePlayer   string
	BlackPlayer   string
	Result        string
	TimeControl   string
	MinimumRating int
	MaximumRating int
	StartDate     string
	EndDate       string
}

// shouldIncludeGame determines if a game should be included based on filter options.
func shouldIncludeGame(game *pgn.Game, options FilterOptions) bool {
	// If no filters are set, include the game
	if options == (FilterOptions{}) {
		return true
	}

	// Check white player
	if options.WhitePlayer != "" {
		if white, ok := game.Tags["White"]; !ok || !strings.EqualFold(white, options.WhitePlayer) {
			return false
		}
	}

	// Check black player
	if options.BlackPlayer != "" {
		if black, ok := game.Tags["Black"]; !ok || !strings.EqualFold(black, options.BlackPlayer) {
			return false
		}
	}

	// Check result
	if options.Result != "" {
		if result, ok := game.Tags["Result"]; !ok || result != options.Result {
			return false
		}
	}

	// Check time control
	if options.TimeControl != "" {
		if tc, ok := game.Tags["TimeControl"]; !ok || tc != options.TimeControl {
			return false
		}
	}

	// Add more filter criteria as needed

	return true
}
