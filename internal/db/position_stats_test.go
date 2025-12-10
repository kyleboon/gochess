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
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.02"]
[White "Player3"]
[Black "Player4"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 1-0

[Event "Game 3"]
[Site "Test"]
[Date "2024.01.03"]
[White "Player5"]
[Black "Player6"]
[Result "1-0"]

1. d4 d5 2. c4 1-0
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

		// Should have many unique positions
		assert.Greater(t, uniqueCount, 0, "Should have at least some unique positions")

		// Should have top positions
		assert.NotEmpty(t, topPositions, "Should have some top positions")
		assert.LessOrEqual(t, len(topPositions), 10, "Should return at most 10 top positions")

		// First position should be the starting position (most common)
		// All three games start from the standard position
		if len(topPositions) > 0 {
			expectedStartFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
			assert.Equal(t, expectedStartFEN, topPositions[0].FEN,
				"Most common position should be the starting position")
			assert.Equal(t, 3, topPositions[0].Count,
				"Starting position should appear 3 times (once per game)")
		}

		// The position after 1.e4 e5 should appear twice (games 1 and 2)
		foundE4E5Position := false
		for _, pos := range topPositions {
			if pos.Count == 2 && pos.FEN == "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2" {
				foundE4E5Position = true
				break
			}
		}
		assert.True(t, foundE4E5Position, "Should find position after 1.e4 e5 appearing twice")

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
	pgnContent := ""
	for i := 0; i < 20; i++ {
		pgnContent += `[Event "Repeated Game"]
[Site "Test"]
[Date "2024.01.` + fmt.Sprintf("%02d", i+1) + `"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 1-0

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

	// With 20 identical games, we should have only a few unique positions
	assert.LessOrEqual(t, uniqueCount, 10, "Should have few unique positions with identical games")

	// Should still limit to 10 results
	assert.LessOrEqual(t, len(topPositions), 10, "Should return at most 10 top positions")

	// The most common position should appear 20 times
	if len(topPositions) > 0 {
		assert.Equal(t, 20, topPositions[0].Count,
			"Starting position should appear 20 times (once per game)")
	}
}
