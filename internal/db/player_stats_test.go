package db

import (
	"context"
	"os"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlayerStats(t *testing.T) {
	// Create a temporary database
	tempDir, err := os.MkdirTemp("", "gochess-test-player-stats-")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := tempDir + "/test.db"
	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Test with empty database
	t.Run("Empty database", func(t *testing.T) {
		stats, err := db.GetPlayerStats(ctx)
		require.NoError(t, err)
		assert.Empty(t, stats, "Should have no stats in empty database")
	})

	// Import games with various time controls and results
	pgnContent := `[Event "Bullet Game 1"]
[Site "Chess.com"]
[Date "2024.01.01"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]
[TimeControl "60+0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O 1-0

[Event "Blitz Game 1"]
[Site "Chess.com"]
[Date "2024.01.02"]
[White "Bob"]
[Black "Alice"]
[Result "0-1"]
[TimeControl "300+0"]

1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. Bg5 0-1

[Event "Rapid Game 1"]
[Site "Chess.com"]
[Date "2024.01.03"]
[White "Alice"]
[Black "Bob"]
[Result "1/2-1/2"]
[TimeControl "900+10"]

1. e4 e5 2. Nf3 Nc6 1/2-1/2

[Event "Classical Game 1"]
[Site "Chess.com"]
[Date "2024.01.04"]
[White "Bob"]
[Black "Alice"]
[Result "1/2-1/2"]
[TimeControl "1800+30"]

1. d4 Nf6 2. c4 e6 1/2-1/2

[Event "Bullet Game 2"]
[Site "Lichess"]
[Date "2024.01.05"]
[White "Alice"]
[Black "Charlie"]
[Result "1-0"]
[TimeControl "120+1"]

1. e4 e5 2. Nf3 1-0
`

	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	count, errs := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs)
	assert.Equal(t, 5, count)

	// Test with imported games
	t.Run("With imported games", func(t *testing.T) {
		stats, err := db.GetPlayerStats(ctx)
		require.NoError(t, err)
		assert.Len(t, stats, 3, "Should have stats for 3 players")

		// Find Alice's stats (should be most active with 4 games)
		var alice *PlayerStats
		for i := range stats {
			if stats[i].Name == "Alice" {
				alice = &stats[i]
				break
			}
		}
		require.NotNil(t, alice, "Should have stats for Alice")

		// Alice's overall stats
		assert.Equal(t, 5, alice.Games, "Alice played 5 games")
		assert.Equal(t, 3, alice.Wins, "Alice won 3 games")
		assert.Equal(t, 0, alice.Losses, "Alice lost 0 games")
		assert.Equal(t, 2, alice.Draws, "Alice drew 2 games")
		assert.InDelta(t, 60.0, alice.WinRate, 0.1, "Alice's win rate should be 60%")

		// Alice's stats by color
		assert.Equal(t, 3, alice.WhiteGames, "Alice played 3 games as white")
		assert.Equal(t, 2, alice.BlackGames, "Alice played 2 games as black")
		assert.Equal(t, 2, alice.WhiteWins, "Alice won 2 games as white")
		assert.Equal(t, 1, alice.BlackWins, "Alice won 1 game as black")
		assert.Equal(t, 0, alice.WhiteLosses, "Alice lost 0 games as white")
		assert.Equal(t, 0, alice.BlackLosses, "Alice lost 0 games as black")
		assert.Equal(t, 1, alice.WhiteDraws, "Alice drew 1 game as white")
		assert.Equal(t, 1, alice.BlackDraws, "Alice drew 1 game as black")

		// Win rates by color
		assert.InDelta(t, 66.7, alice.WhiteWinRate, 0.1, "Alice's white win rate should be ~66.7%")
		assert.InDelta(t, 50.0, alice.BlackWinRate, 0.1, "Alice's black win rate should be 50%")

		// Alice's time control breakdown
		assert.Equal(t, 2, alice.BulletGames, "Alice played 2 bullet games")
		assert.Equal(t, 1, alice.BlitzGames, "Alice played 1 blitz game")
		assert.Equal(t, 2, alice.RapidGames, "Alice played 2 rapid games (900+10 and 1800+30)")
		assert.Equal(t, 0, alice.ClassicalGames, "Alice played 0 classical games")

		// Find Bob's stats
		var bob *PlayerStats
		for i := range stats {
			if stats[i].Name == "Bob" {
				bob = &stats[i]
				break
			}
		}
		require.NotNil(t, bob, "Should have stats for Bob")

		// Bob's overall stats
		assert.Equal(t, 4, bob.Games, "Bob played 4 games")
		assert.Equal(t, 0, bob.Wins, "Bob won 0 games")
		assert.Equal(t, 2, bob.Losses, "Bob lost 2 games")
		assert.Equal(t, 2, bob.Draws, "Bob drew 2 games")

		// Bob's stats by color
		assert.Equal(t, 2, bob.WhiteGames, "Bob played 2 games as white")
		assert.Equal(t, 2, bob.BlackGames, "Bob played 2 games as black")
		assert.Equal(t, 0, bob.WhiteWins, "Bob won 0 games as white")
		assert.Equal(t, 0, bob.BlackWins, "Bob won 0 games as black")

		// Bob's time control breakdown
		assert.Equal(t, 1, bob.BulletGames, "Bob played 1 bullet game")
		assert.Equal(t, 1, bob.BlitzGames, "Bob played 1 blitz game")
		assert.Equal(t, 2, bob.RapidGames, "Bob played 2 rapid games (900+10 and 1800+30)")
		assert.Equal(t, 0, bob.ClassicalGames, "Bob played 0 classical games")
	})

	// Test filtering by player
	t.Run("Filter by specific player", func(t *testing.T) {
		stats, err := db.GetPlayerStatsFiltered(ctx, []string{"Alice"})
		require.NoError(t, err)
		assert.Len(t, stats, 1, "Should have stats for only Alice")
		assert.Equal(t, "Alice", stats[0].Name)
		assert.Equal(t, 5, stats[0].Games)
	})

	// Test filtering by multiple players
	t.Run("Filter by multiple players", func(t *testing.T) {
		stats, err := db.GetPlayerStatsFiltered(ctx, []string{"Alice", "Bob"})
		require.NoError(t, err)
		assert.Len(t, stats, 2, "Should have stats for Alice and Bob")
	})
}

func TestCategorizeTimeControl(t *testing.T) {
	tests := []struct {
		name     string
		tc       string
		expected string
	}{
		{"Empty string", "", "unknown"},
		{"Bullet - 60+0", "60+0", "bullet"},
		{"Bullet - 120+1", "120+1", "bullet"},
		{"Blitz - 180+0", "180+0", "blitz"},
		{"Blitz - 300+0", "300+0", "blitz"},
		{"Blitz - 300+3", "300+3", "blitz"},
		{"Rapid - 600+0", "600+0", "rapid"},
		{"Rapid - 900+10", "900+10", "rapid"},
		{"Rapid - 1800+0", "1800+0", "rapid"},
		{"Classical - 3600+0", "3600+0", "classical"},
		{"Classical - 5400+30", "5400+30", "classical"},
		{"Daily - 1/86400", "1/86400", "classical"},
		{"Daily - 3/259200", "3/259200", "classical"},
		{"Invalid format", "invalid", "unknown"},
		{"No increment", "300", "blitz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeTimeControl(tt.tc)
			assert.Equal(t, tt.expected, result, "Time control %q should be categorized as %s", tt.tc, tt.expected)
		})
	}
}

func TestGetPlayerStats_Sorting(t *testing.T) {
	// Create a temporary database
	tempDir, err := os.MkdirTemp("", "gochess-test-sorting-")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	dbPath := tempDir + "/test.db"
	db, err := NewWithLogger(dbPath, logging.Discard())
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Import games where Alice plays more than Bob
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.02"]
[White "Alice"]
[Black "Charlie"]
[Result "1-0"]

1. e4 e5 1-0

[Event "Game 3"]
[Site "Test"]
[Date "2024.01.03"]
[White "Alice"]
[Black "Dave"]
[Result "1-0"]

1. e4 e5 1-0
`

	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	require.NoError(t, err)

	_, errs := db.ImportPGN(ctx, pgnFile)
	require.Empty(t, errs)

	stats, err := db.GetPlayerStats(ctx)
	require.NoError(t, err)

	// Alice should be first (3 games), others have 1 game each
	assert.Equal(t, "Alice", stats[0].Name, "Most active player should be first")
	assert.Equal(t, 3, stats[0].Games, "Alice should have 3 games")
}
