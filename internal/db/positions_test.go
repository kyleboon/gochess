package db

import (
	"testing"

	"github.com/kyleboon/gochess/internal/pgn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPositions(t *testing.T) {
	tests := []struct {
		name           string
		pgnText        string
		expectedCount  int
		checkFirstPos  bool
		firstPosFEN    string
		firstPosMove   string
		checkLastPos   bool
		lastPosFEN     string
		lastPosMove    string
	}{
		{
			name: "Simple game with a few moves",
			pgnText: `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0`,
			expectedCount: 6, // Starting position + 5 moves = 6 positions
			checkFirstPos: true,
			firstPosFEN:   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			firstPosMove:  "e2e4",
			checkLastPos:  true,
			// After 3. Bb5, the game ends, so last position has no next move
		},
		{
			name: "Very short game (scholar's mate)",
			pgnText: `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Bc4 Nc6 3. Qh5 Nf6 4. Qxf7# 1-0`,
			expectedCount: 8, // 7 moves + final position = 8
			checkFirstPos: true,
			firstPosFEN:   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			firstPosMove:  "e2e4",
		},
		{
			name: "Game with no moves (just tags)",
			pgnText: `[Event "Test"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "*"]

*`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the PGN
			pgnDB := &pgn.DB{}
			parseErrors := pgnDB.Parse(tt.pgnText)
			require.Empty(t, parseErrors, "Failed to parse PGN: %v", parseErrors)
			require.NotNil(t, pgnDB, "PGN database should not be nil")
			require.Greater(t, len(pgnDB.Games), 0, "Should have at least one game")

			game := pgnDB.Games[0]

			// Parse the moves
			err := pgnDB.ParseMoves(game)
			require.NoError(t, err, "Failed to parse moves")

			// Extract positions
			positions := ExtractPositions(game)

			// Check count
			assert.Equal(t, tt.expectedCount, len(positions), "Unexpected number of positions")

			if tt.expectedCount == 0 {
				return
			}

			// Check first position if requested
			if tt.checkFirstPos {
				assert.Equal(t, 0, positions[0].MoveNumber, "First position should have move number 0")
				assert.Equal(t, tt.firstPosFEN, positions[0].FEN, "First position FEN mismatch")
				assert.Equal(t, tt.firstPosMove, positions[0].NextMove, "First position next move mismatch")
			}

			// Check last position if requested
			if tt.checkLastPos && len(positions) > 0 {
				lastPos := positions[len(positions)-1]
				assert.Equal(t, "", lastPos.NextMove, "Last position should have no next move")
			}

			// Verify move numbers are sequential
			for i, pos := range positions {
				assert.Equal(t, i, pos.MoveNumber, "Move numbers should be sequential")
			}

			// Verify all positions have valid FEN strings (basic check)
			for i, pos := range positions {
				assert.NotEmpty(t, pos.FEN, "Position %d should have a FEN string", i)
				// FEN should have 6 parts separated by spaces
				assert.Equal(t, 6, len(splitFEN(pos.FEN)), "Position %d FEN should have 6 fields", i)
			}
		})
	}
}

func TestExtractPositions_NilGame(t *testing.T) {
	game := &pgn.Game{}
	positions := ExtractPositions(game)
	assert.Empty(t, positions, "Should return empty slice for nil root")
}

// Helper function to split FEN string
func splitFEN(fen string) []string {
	fields := make([]string, 0, 6)
	current := ""
	for _, char := range fen {
		if char == ' ' {
			if current != "" {
				fields = append(fields, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		fields = append(fields, current)
	}
	return fields
}
