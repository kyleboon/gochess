package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPositionStats(t *testing.T) {
	// Create a temporary database
	tempDir, err := os.MkdirTemp("", "gochess-test-position-stats-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := tempDir + "/test.db"
	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Test with empty database
	t.Run("Empty database", func(t *testing.T) {
		uniqueCount, topPositions, err := db.GetPositionStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, uniqueCount, "Should have 0 unique positions in empty database")
		assert.Empty(t, topPositions, "Should have no top positions in empty database")
	})

	// Import some games with positions
	// Games need to be long enough (>10 moves) to have positions counted in stats
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Nb8 10. d4 Nbd7 11. Nbd2 Bb7 12. Bc2 Re8 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.02"]
[White "Player3"]
[Black "Player4"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Na5 10. Bc2 c5 11. d4 Qc7 12. Nbd2 Nc6 1-0

[Event "Game 3"]
[Site "Test"]
[Date "2024.01.03"]
[White "Player5"]
[Black "Player6"]
[Result "1-0"]

1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. cxd5 exd5 5. Bg5 Be7 6. e3 c6 7. Bd3 Nbd7 8. Qc2 Nh5 9. Bxe7 Qxe7 10. Nge2 Nb6 11. O-O-O Bd7 12. Kb1 O-O-O 1-0
`

	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	count, errs := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs)
	assert.Equal(t, 3, count)

	// Test with imported games
	t.Run("With imported games", func(t *testing.T) {
		uniqueCount, topPositions, err := db.GetPositionStats(ctx)
		require.NoError(t, err)

		// Should have many unique positions (only counting positions after move 10)
		assert.Greater(t, uniqueCount, 0, "Should have at least some unique positions")

		// Should have top positions
		assert.NotEmpty(t, topPositions, "Should have some top positions")
		assert.LessOrEqual(t, len(topPositions), 10, "Should return at most 10 top positions")

		// Since we only count positions after move 10, and the games diverge before that,
		// we won't necessarily have positions appearing in multiple games
		// Just verify that we have valid position data
		for _, pos := range topPositions {
			assert.Greater(t, pos.Count, 0, "All positions should have count > 0")
			assert.NotEmpty(t, pos.FEN, "All positions should have a FEN string")
		}

		// Verify counts are in descending order
		for i := 1; i < len(topPositions); i++ {
			assert.GreaterOrEqual(t, topPositions[i-1].Count, topPositions[i].Count,
				"Top positions should be sorted by count (descending)")
		}

		// Verify all FENs are non-empty
		for i, pos := range topPositions {
			assert.NotEmpty(t, pos.FEN, "Position %d FEN should not be empty", i)
			assert.Greater(t, pos.Count, 0, "Position %d count should be positive", i)
		}
	})
}

func TestGetPositionStats_LargeDataset(t *testing.T) {
	// Create a temporary database
	tempDir, err := os.MkdirTemp("", "gochess-test-large-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := tempDir + "/test.db"
	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Import many games with same moves but different dates to test the top positions limit
	// Games need to be long enough (>10 moves) to have positions counted in stats
	pgnContent := ""
	for i := 0; i < 20; i++ {
		pgnContent += `[Event "Repeated Game"]
[Site "Test"]
[Date "2024.01.` + fmt.Sprintf("%02d", i+1) + `"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Nb8 10. d4 Nbd7 11. Nbd2 1-0

`
	}

	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	count, errs := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs)
	assert.Equal(t, 20, count)

	uniqueCount, topPositions, err := db.GetPositionStats(ctx)
	require.NoError(t, err)

	// With 20 identical games, we should have only a few unique positions (after move 10)
	assert.LessOrEqual(t, uniqueCount, 10, "Should have few unique positions with identical games")

	// Should still limit to 10 results
	assert.LessOrEqual(t, len(topPositions), 10, "Should return at most 10 top positions")

	// The most common position (after move 10) should appear 20 times
	// Note: we only count positions after move 10 (20 half-moves)
	if len(topPositions) > 0 {
		assert.Equal(t, 20, topPositions[0].Count,
			"Most common position after move 10 should appear 20 times (once per game)")
	}
}
