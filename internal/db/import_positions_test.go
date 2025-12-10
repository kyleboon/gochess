package db

import (
	"context"
	"os"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportPGN_WithPositions tests that positions are stored when importing PGN
func TestImportPGN_WithPositions(t *testing.T) {
	// Create a temporary database
	tempDir, err := os.MkdirTemp("", "gochess-test-positions-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := tempDir + "/test.db"
	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer db.Close()

	// Create a test PGN file
	pgnContent := `[Event "Test Game"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 1-0
`

	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	// Import the PGN
	ctx := context.Background()
	count, errs := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs, "Import should succeed without errors")
	assert.Equal(t, 1, count, "Should import 1 game")

	// Verify the game was imported
	gameCount, err := db.GetGameCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, gameCount, "Should have 1 game in database")

	// Query positions from the database
	var positionCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM positions").Scan(&positionCount)
	require.NoError(t, err)

	// The game has 13 moves (plies), so we expect 14 positions (including final position)
	// Actually: 1.e4 e5 2.Nf3 Nc6 3.Bb5 a6 4.Ba4 Nf6 5.O-O Be7 6.Re1 b5 7.Bb3
	// That's 13 plies, so 14 positions total
	assert.Equal(t, 14, positionCount, "Should have 14 positions stored")

	// Verify we can query a specific position (the starting position)
	var fen, nextMove string
	var moveNumber int
	err = db.conn.QueryRow(`
		SELECT move_number, fen, next_move
		FROM positions
		WHERE move_number = 0
		LIMIT 1
	`).Scan(&moveNumber, &fen, &nextMove)
	require.NoError(t, err)

	assert.Equal(t, 0, moveNumber, "First position should be move number 0")
	assert.Equal(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", fen,
		"First position should be starting position")
	assert.Equal(t, "e2e4", nextMove, "First move should be e2e4")
}

// TestImportPGN_PositionsForMultipleGames tests position storage for multiple games
func TestImportPGN_PositionsForMultipleGames(t *testing.T) {
	// Create a temporary database
	tempDir, err := os.MkdirTemp("", "gochess-test-multi-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := tempDir + "/test.db"
	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer db.Close()

	// Create a test PGN file with 2 games
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.02"]
[White "Player3"]
[Black "Player4"]
[Result "0-1"]

1. d4 d5 2. c4 e6 3. Nc3 0-1
`

	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	// Import the PGN
	ctx := context.Background()
	count, errs := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs, "Import should succeed without errors")
	assert.Equal(t, 2, count, "Should import 2 games")

	// Query total positions from the database
	var totalPositions int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM positions").Scan(&totalPositions)
	require.NoError(t, err)

	// Game 1: 3 plies = 4 positions
	// Game 2: 5 plies = 6 positions
	// Total: 10 positions
	assert.Equal(t, 10, totalPositions, "Should have 10 total positions")

	// Verify positions are associated with the correct games
	rows, err := db.conn.Query(`
		SELECT game_id, COUNT(*) as pos_count
		FROM positions
		GROUP BY game_id
		ORDER BY game_id
	`)
	require.NoError(t, err)
	defer rows.Close()

	gameCounts := make(map[int]int)
	for rows.Next() {
		var gameID, count int
		err = rows.Scan(&gameID, &count)
		require.NoError(t, err)
		gameCounts[gameID] = count
	}

	assert.Equal(t, 2, len(gameCounts), "Should have positions for 2 games")
	assert.Equal(t, 4, gameCounts[1], "Game 1 should have 4 positions")
	assert.Equal(t, 6, gameCounts[2], "Game 2 should have 6 positions")
}

// TestImportPGN_SkipDuplicatePositions tests that duplicate games don't store duplicate positions
func TestImportPGN_SkipDuplicatePositions(t *testing.T) {
	// Create a temporary database
	tempDir, err := os.MkdirTemp("", "gochess-test-dup-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := tempDir + "/test.db"
	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer db.Close()

	// Create a test PGN file
	pgnContent := `[Event "Test Game"]
[Site "Test"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 1-0
`

	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Import the same game twice
	count1, errs1 := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs1)
	assert.Equal(t, 1, count1, "Should import 1 game on first import")

	count2, errs2 := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs2)
	assert.Equal(t, 0, count2, "Should skip duplicate game on second import")

	// Verify we still only have positions for one game
	var positionCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM positions").Scan(&positionCount)
	require.NoError(t, err)

	assert.Equal(t, 3, positionCount, "Should have 3 positions (not duplicated)")
}
