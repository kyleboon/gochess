package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
)

func TestImportPGN_WithECOClassification(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewWithLogger(dbPath, logging.Discard())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create a test PGN file with a known opening (Italian Game)
	pgnContent := `[Event "Test Game"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player A"]
[Black "Player B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. O-O Nf6 5. d3 1-0
`

	pgnPath := filepath.Join(tempDir, "test.pgn")
	if err := os.WriteFile(pgnPath, []byte(pgnContent), 0644); err != nil {
		t.Fatalf("Failed to create test PGN file: %v", err)
	}

	// Import the PGN file
	ctx := context.Background()
	imported, errors := db.ImportPGN(ctx, pgnPath)

	if len(errors) > 0 {
		t.Fatalf("Import errors: %v", errors)
	}

	if imported != 1 {
		t.Errorf("Expected 1 game imported, got %d", imported)
	}

	// Verify ECO code was stored
	var ecoCode, openingName string
	err = db.conn.QueryRowContext(ctx, `
		SELECT eco_code, opening_name FROM games WHERE white = ?
	`, "Player A").Scan(&ecoCode, &openingName)

	if err != nil {
		t.Fatalf("Failed to query game: %v", err)
	}

	if ecoCode == "" {
		t.Error("Expected ECO code to be set, but it was empty")
	}

	if openingName == "" {
		t.Error("Expected opening name to be set, but it was empty")
	}

	// The opening should be an Italian Game variant (C50-C59)
	if len(ecoCode) > 0 && ecoCode[0] != 'C' {
		t.Errorf("Expected Italian Game (C-series ECO), got %s: %s", ecoCode, openingName)
	}

	t.Logf("Classified as: %s - %s", ecoCode, openingName)

	// Verify positions also have ECO codes
	var posCount int
	err = db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM positions WHERE eco_code = ? AND opening_name = ?
	`, ecoCode, openingName).Scan(&posCount)

	if err != nil {
		t.Fatalf("Failed to query positions: %v", err)
	}

	if posCount == 0 {
		t.Error("Expected positions to have ECO codes, but found none")
	}

	t.Logf("Found %d positions with ECO classification", posCount)
}

func TestImportPGN_MultipleDifferentOpenings(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewWithLogger(dbPath, logging.Discard())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create a test PGN file with multiple games with different openings
	pgnContent := `[Event "Game 1"]
[Site "Test"]
[Date "2024.01.01"]
[Round "1"]
[White "Player A"]
[Black "Player B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0

[Event "Game 2"]
[Site "Test"]
[Date "2024.01.01"]
[Round "2"]
[White "Player C"]
[Black "Player D"]
[Result "1-0"]

1. e4 c5 2. Nf3 1-0

[Event "Game 3"]
[Site "Test"]
[Date "2024.01.01"]
[Round "3"]
[White "Player E"]
[Black "Player F"]
[Result "1-0"]

1. d4 d5 2. c4 1-0
`

	pgnPath := filepath.Join(tempDir, "test.pgn")
	if err := os.WriteFile(pgnPath, []byte(pgnContent), 0644); err != nil {
		t.Fatalf("Failed to create test PGN file: %v", err)
	}

	// Import the PGN file
	ctx := context.Background()
	imported, errors := db.ImportPGN(ctx, pgnPath)

	if len(errors) > 0 {
		t.Fatalf("Import errors: %v", errors)
	}

	if imported != 3 {
		t.Errorf("Expected 3 games imported, got %d", imported)
	}

	// Verify each game has different ECO codes
	rows, err := db.conn.QueryContext(ctx, `
		SELECT white, eco_code, opening_name FROM games ORDER BY white
	`)
	if err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}
	defer rows.Close()

	games := make(map[string]struct {
		eco     string
		opening string
	})

	for rows.Next() {
		var white, eco, opening string
		if err := rows.Scan(&white, &eco, &opening); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		games[white] = struct {
			eco     string
			opening string
		}{eco, opening}

		if eco == "" {
			t.Errorf("Game by %s has no ECO code", white)
		}

		t.Logf("%s: %s - %s", white, eco, opening)
	}

	// Game 1 should be Ruy Lopez (C60)
	if game1, ok := games["Player A"]; ok {
		if len(game1.eco) > 0 && game1.eco[0] != 'C' {
			t.Errorf("Expected Ruy Lopez (C-series), got %s", game1.eco)
		}
	}

	// Game 2 should be Sicilian (B20)
	if game2, ok := games["Player C"]; ok {
		if len(game2.eco) > 0 && game2.eco[0] != 'B' {
			t.Errorf("Expected Sicilian Defense (B-series), got %s", game2.eco)
		}
	}

	// Game 3 should be Queen's Gambit (D06)
	if game3, ok := games["Player E"]; ok {
		if len(game3.eco) > 0 && game3.eco[0] != 'D' {
			t.Errorf("Expected Queen's Gambit (D-series), got %s", game3.eco)
		}
	}
}
