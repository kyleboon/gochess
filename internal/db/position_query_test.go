package db

import (
	"context"
	"os"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDBWithGame(t *testing.T) (*DB, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "gochess-posquery-test-")
	require.NoError(t, err)

	database, err := NewWithLogger(tempDir+"/test.db", logging.Discard())
	require.NoError(t, err)

	// Import a game with positions
	pgnContent := `[Event "Test Tournament"]
[Site "Test Location"]
[Date "2024.01.15"]
[Round "1"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0
`
	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	count, errs := database.ImportPGN(context.Background(), pgnFile)
	require.Empty(t, errs)
	require.Equal(t, 1, count)

	return database, tempDir
}

func TestGetPositionByGameAndMove(t *testing.T) {
	database, tempDir := setupTestDBWithGame(t)
	defer os.RemoveAll(tempDir)
	defer database.Close()

	ctx := context.Background()

	t.Run("valid position", func(t *testing.T) {
		// Move 0 is the starting position
		gp, err := database.GetPositionByGameAndMove(ctx, 1, 0)
		require.NoError(t, err)
		require.NotNil(t, gp)
		assert.Equal(t, 1, gp.GameID)
		assert.Equal(t, 0, gp.MoveNumber)
		assert.NotEmpty(t, gp.FEN)
		assert.Equal(t, "Alice", gp.White)
		assert.Equal(t, "Bob", gp.Black)
		assert.Equal(t, "Test Tournament", gp.Event)
		assert.Equal(t, "2024.01.15", gp.Date)
	})

	t.Run("position not found", func(t *testing.T) {
		_, err := database.GetPositionByGameAndMove(ctx, 1, 999)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "position not found")
	})

	t.Run("game not found", func(t *testing.T) {
		_, err := database.GetPositionByGameAndMove(ctx, 999, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "position not found")
	})
}

func TestGetPositionsForGame(t *testing.T) {
	database, tempDir := setupTestDBWithGame(t)
	defer os.RemoveAll(tempDir)
	defer database.Close()

	ctx := context.Background()

	t.Run("all positions for game", func(t *testing.T) {
		positions, err := database.GetPositionsForGame(ctx, 1)
		require.NoError(t, err)
		// 1. e4 e5 2. Nf3 Nc6 3. Bb5 = starting pos + 5 half-moves = 6 positions
		require.True(t, len(positions) >= 5, "expected at least 5 positions, got %d", len(positions))

		// Verify ordering
		for i := 1; i < len(positions); i++ {
			assert.True(t, positions[i].MoveNumber >= positions[i-1].MoveNumber,
				"positions should be ordered by move number")
		}

		// Each position should have game metadata
		for _, p := range positions {
			assert.Equal(t, "Alice", p.White)
			assert.Equal(t, "Bob", p.Black)
			assert.NotEmpty(t, p.FEN)
		}
	})

	t.Run("no positions for nonexistent game", func(t *testing.T) {
		positions, err := database.GetPositionsForGame(ctx, 999)
		require.NoError(t, err)
		assert.Empty(t, positions)
	})
}

func TestUpdatePositionEvaluation(t *testing.T) {
	database, tempDir := setupTestDBWithGame(t)
	defer os.RemoveAll(tempDir)
	defer database.Close()

	ctx := context.Background()

	t.Run("update evaluation", func(t *testing.T) {
		// Get a position first
		gp, err := database.GetPositionByGameAndMove(ctx, 1, 0)
		require.NoError(t, err)
		assert.Nil(t, gp.Evaluation)

		// Update it
		err = database.UpdatePositionEvaluation(ctx, gp.PositionID, 0.35)
		require.NoError(t, err)

		// Verify it was updated
		gp2, err := database.GetPositionByGameAndMove(ctx, 1, 0)
		require.NoError(t, err)
		require.NotNil(t, gp2.Evaluation)
		assert.InDelta(t, 0.35, *gp2.Evaluation, 0.001)
	})

	t.Run("update nonexistent position", func(t *testing.T) {
		err := database.UpdatePositionEvaluation(ctx, 999999, 1.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "position not found")
	})
}
