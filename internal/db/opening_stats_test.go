package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOpeningStats(t *testing.T) {
	// Create a temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	t.Run("Empty database", func(t *testing.T) {
		stats, err := db.GetOpeningStats(ctx)
		require.NoError(t, err)
		assert.Empty(t, stats, "Should have no stats in empty database")
	})

	// Import games with different openings
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.02"]
[Round "1"]
[White "Bob"]
[Black "Alice"]
[Result "0-1"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 0-1

[Event "Game 3"]
[Site "Test"]
[Date "2024.01.03"]
[Round "1"]
[White "Alice"]
[Black "Charlie"]
[Result "1/2-1/2"]

1. e4 c5 2. Nf3 1/2-1/2

[Event "Game 4"]
[Site "Test"]
[Date "2024.01.04"]
[Round "1"]
[White "Alice"]
[Black "Dave"]
[Result "1-0"]

1. e4 c5 2. Nf3 d6 1-0

[Event "Game 5"]
[Site "Test"]
[Date "2024.01.05"]
[Round "1"]
[White "Alice"]
[Black "Eve"]
[Result "1-0"]

1. d4 d5 2. c4 1-0
`

	pgnPath := filepath.Join(tempDir, "test.pgn")
	err = os.WriteFile(pgnPath, []byte(pgnContent), 0644)
	require.NoError(t, err)

	_, errs := db.ImportPGN(ctx, pgnPath)
	require.Empty(t, errs)

	t.Run("Get all opening stats", func(t *testing.T) {
		stats, err := db.GetOpeningStats(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, stats, "Should have opening statistics")

		// Should have at least 2 different openings (Ruy Lopez, Sicilian, Queen's Gambit)
		assert.GreaterOrEqual(t, len(stats), 2, "Should have multiple openings")

		// Verify stats are sorted by game count (most common first)
		for i := 1; i < len(stats); i++ {
			assert.GreaterOrEqual(t, stats[i-1].Games, stats[i].Games,
				"Stats should be sorted by game count descending")
		}

		// Check that each opening has valid statistics
		for _, s := range stats {
			assert.NotEmpty(t, s.ECOCode, "ECO code should not be empty")
			assert.NotEmpty(t, s.OpeningName, "Opening name should not be empty")
			assert.Greater(t, s.Games, 0, "Should have at least one game")
			assert.Equal(t, s.Games, s.Wins+s.Losses+s.Draws,
				"Win+Loss+Draw should equal total games")

			// Win rate should be between 0 and 100
			assert.GreaterOrEqual(t, s.WinRate, 0.0)
			assert.LessOrEqual(t, s.WinRate, 100.0)
		}
	})

	t.Run("Filter by specific player", func(t *testing.T) {
		stats, err := db.GetOpeningStatsFiltered(ctx, []string{"Alice"})
		require.NoError(t, err)
		require.NotEmpty(t, stats, "Alice should have opening statistics")

		// Alice plays in all 5 games, so should have stats
		totalGames := 0
		for _, s := range stats {
			totalGames += s.Games
		}
		assert.Equal(t, 5, totalGames, "Alice played in 5 games")
	})

	t.Run("Filter by multiple players", func(t *testing.T) {
		stats, err := db.GetOpeningStatsFiltered(ctx, []string{"Bob", "Charlie"})
		require.NoError(t, err)
		require.NotEmpty(t, stats, "Should have stats for Bob and Charlie")

		// Bob plays in 2 games, Charlie in 1 game
		totalGames := 0
		for _, s := range stats {
			totalGames += s.Games
		}
		assert.Equal(t, 3, totalGames, "Bob and Charlie played in 3 games total")
	})
}

func TestGetOpeningStats_WinRates(t *testing.T) {
	// Create a temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Import games where Alice always wins with Sicilian
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 c5 2. Nf3 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.02"]
[Round "1"]
[White "Alice"]
[Black "Charlie"]
[Result "1-0"]

1. e4 c5 2. Nf3 d6 1-0

[Event "Game 3"]
[Site "Test"]
[Date "2024.01.03"]
[Round "1"]
[White "Alice"]
[Black "Dave"]
[Result "1-0"]

1. e4 c5 2. Nf3 Nc6 1-0
`

	pgnPath := filepath.Join(tempDir, "test.pgn")
	err = os.WriteFile(pgnPath, []byte(pgnContent), 0644)
	require.NoError(t, err)

	_, errs := db.ImportPGN(ctx, pgnPath)
	require.Empty(t, errs)

	stats, err := db.GetOpeningStatsFiltered(ctx, []string{"Alice"})
	require.NoError(t, err)
	require.NotEmpty(t, stats)

	// Debug: print all stats
	t.Logf("Found %d opening stats for Alice", len(stats))
	for _, s := range stats {
		t.Logf("  %s (%s): %d games, %d wins", s.ECOCode, s.OpeningName, s.Games, s.Wins)
	}

	// Should have Sicilian Defense stats (collect all Sicilian variations)
	totalSicilianGames := 0
	totalSicilianWins := 0
	for i := range stats {
		if stats[i].ECOCode[0] == 'B' { // Sicilian is B-series
			totalSicilianGames += stats[i].Games
			totalSicilianWins += stats[i].Wins
		}
	}

	assert.Equal(t, 3, totalSicilianGames, "Alice played 3 Sicilian games total")
	assert.Equal(t, 3, totalSicilianWins, "Alice won all 3 Sicilian games")
}

func TestGetOpeningStats_ColorBreakdown(t *testing.T) {
	// Create a temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Import games where Alice plays same opening as both colors
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.02"]
[Round "1"]
[White "Bob"]
[Black "Alice"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0

[Event "Game 3"]
[Site "Test"]
[Date "2024.01.03"]
[Round "1"]
[White "Alice"]
[Black "Charlie"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 Nf6 1-0
`

	pgnPath := filepath.Join(tempDir, "test.pgn")
	err = os.WriteFile(pgnPath, []byte(pgnContent), 0644)
	require.NoError(t, err)

	_, errs := db.ImportPGN(ctx, pgnPath)
	require.Empty(t, errs)

	stats, err := db.GetOpeningStatsFiltered(ctx, []string{"Alice"})
	require.NoError(t, err)
	require.NotEmpty(t, stats)

	// Debug: print all stats
	t.Logf("Found %d opening stats for Alice", len(stats))
	for _, s := range stats {
		t.Logf("  %s (%s): %d games (%d white, %d black), %d wins, %d losses",
			s.ECOCode, s.OpeningName, s.Games, s.WhiteGames, s.BlackGames, s.Wins, s.Losses)
	}

	// Should have Ruy Lopez stats (collect all variations)
	totalGames := 0
	totalWhiteGames := 0
	totalBlackGames := 0
	totalWins := 0
	totalLosses := 0
	totalWhiteWins := 0
	totalBlackWins := 0

	for i := range stats {
		if stats[i].ECOCode[0] == 'C' { // Ruy Lopez is C-series
			totalGames += stats[i].Games
			totalWhiteGames += stats[i].WhiteGames
			totalBlackGames += stats[i].BlackGames
			totalWins += stats[i].Wins
			totalLosses += stats[i].Losses
			totalWhiteWins += stats[i].WhiteWins
			totalBlackWins += stats[i].BlackWins
		}
	}

	assert.Equal(t, 3, totalGames, "Alice played 3 Ruy Lopez games")
	assert.Equal(t, 2, totalWhiteGames, "2 games as white")
	assert.Equal(t, 1, totalBlackGames, "1 game as black")
	assert.Equal(t, 2, totalWins, "2 wins (both as white)")
	assert.Equal(t, 1, totalLosses, "1 loss (as black)")
	assert.Equal(t, 2, totalWhiteWins, "2 wins as white")
	assert.Equal(t, 0, totalBlackWins, "0 wins as black")
}
