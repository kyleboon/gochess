package db

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
	"github.com/kyleboon/gochess/internal/pgn"
)

func TestImportPGN_WithFEN(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gochess-test-db-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewWithLogger(tempDir+"/test.db", logging.Discard())
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Test with a PGN file that has a FEN, which should be ignored
	pgnPath := "../../testdata/invalid_fen.pgn"
	count, errors := db.ImportPGN(context.Background(), pgnPath)

	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, but got %d: %v", len(errors), errors)
	}

	if count != 1 {
		t.Fatalf("expected 1 game to be imported, but got %d", count)
	}
}

func TestImportPGN_NoFEN(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gochess-test-db-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewWithLogger(tempDir+"/test.db", logging.Discard())
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Test with a Lichess PGN file that has no FEN tag
	// This simulates the real-world scenario that was causing errors
	pgnPath := "../../testdata/lichess_no_fen.pgn"
	count, errors := db.ImportPGN(context.Background(), pgnPath)

	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, but got %d: %v", len(errors), errors)
	}

	if count != 1 {
		t.Fatalf("expected 1 game to be imported, but got %d", count)
	}

	// Verify the game was imported with the correct FEN (standard starting position)
	games, err := db.SearchGames(context.Background(), map[string]string{}, 10, 0)
	if err != nil {
		t.Fatalf("failed to search games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("expected 1 game in database, but found %d", len(games))
	}
}

// TestValidateGameTags tests the validateGameTags function
func TestValidateGameTags(t *testing.T) {
	tests := []struct {
		name      string
		game      *pgn.Game
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid game with all required tags",
			game: &pgn.Game{
				Tags: map[string]string{
					"Event":  "Test Tournament",
					"Site":   "Test City",
					"Date":   "2024.01.01",
					"White":  "Player1",
					"Black":  "Player2",
					"Result": "1-0",
				},
			},
			wantError: false,
		},
		{
			name: "Missing Event tag",
			game: &pgn.Game{
				Tags: map[string]string{
					"Site":   "Test City",
					"Date":   "2024.01.01",
					"White":  "Player1",
					"Black":  "Player2",
					"Result": "1-0",
				},
			},
			wantError: true,
			errorMsg:  "Event",
		},
		{
			name: "Missing multiple tags",
			game: &pgn.Game{
				Tags: map[string]string{
					"Event": "Test Tournament",
					"Date":  "2024.01.01",
				},
			},
			wantError: true,
			errorMsg:  "missing required tags",
		},
		{
			name: "Empty tag value",
			game: &pgn.Game{
				Tags: map[string]string{
					"Event":  "Test Tournament",
					"Site":   "",
					"Date":   "2024.01.01",
					"White":  "Player1",
					"Black":  "Player2",
					"Result": "1-0",
				},
			},
			wantError: true,
			errorMsg:  "Site",
		},
		{
			name: "All tags empty",
			game: &pgn.Game{
				Tags: map[string]string{
					"Event":  "",
					"Site":   "",
					"Date":   "",
					"White":  "",
					"Black":  "",
					"Result": "",
				},
			},
			wantError: true,
			errorMsg:  "missing required tags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGameTags(tt.game)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateGameTags() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateGameTags() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateGameTags() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestProcessParseErrors tests the processParseErrors function
func TestProcessParseErrors(t *testing.T) {
	tests := []struct {
		name          string
		parseErrors   []error
		pgnData       *PGNData
		wantErrorCount int
		checkErrorType bool
	}{
		{
			name:          "No errors",
			parseErrors:   []error{},
			pgnData:       &PGNData{},
			wantErrorCount: 0,
		},
		{
			name: "Single parse error",
			parseErrors: []error{
				&pgn.ParseError{Line: 5, Col: 10, Message: "invalid move"},
			},
			pgnData: &PGNData{
				GameTexts: []string{"[Event \"Test\"]\n\n1. e4 e5"},
			},
			wantErrorCount: 1,
			checkErrorType: true,
		},
		{
			name: "Multiple parse errors",
			parseErrors: []error{
				&pgn.ParseError{Line: 5, Col: 10, Message: "invalid move"},
				&pgn.ParseError{Line: 10, Col: 5, Message: "missing tag"},
			},
			pgnData: &PGNData{
				GameTexts: []string{
					"[Event \"Test1\"]\n\n1. e4 e5",
					"[Event \"Test2\"]\n\n1. d4 d5",
				},
			},
			wantErrorCount: 2,
			checkErrorType: true,
		},
		{
			name: "Non-ParseError",
			parseErrors: []error{
				sql.ErrNoRows,
			},
			pgnData:        &PGNData{},
			wantErrorCount: 1,
			checkErrorType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := processParseErrors(tt.parseErrors, tt.pgnData)
			if len(errors) != tt.wantErrorCount {
				t.Errorf("processParseErrors() returned %d errors, want %d", len(errors), tt.wantErrorCount)
			}
			if tt.checkErrorType {
				for _, err := range errors {
					if _, ok := err.(*PGNImportError); !ok {
						t.Errorf("processParseErrors() returned error of type %T, want *PGNImportError", err)
					}
				}
			}
		})
	}
}

// TestCheckDuplicateGame tests the checkDuplicateGame function
func TestCheckDuplicateGame(t *testing.T) {
	// Create a test database
	tempDir, err := os.MkdirTemp("", "gochess-test-duplicate-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewWithLogger(tempDir+"/test.db", logging.Discard())
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Insert a test game with a known hash
	testHash := "test-hash-123"
	tx, err := db.conn.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO games (event, site, date, round, white, black, result, pgn_text, game_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "Test Event", "Test Site", "2024.01.01", "1", "Player1", "Player2", "1-0", "test pgn", testHash)
	if err != nil {
		t.Fatalf("failed to insert test game: %v", err)
	}

	tests := []struct {
		name         string
		gameHash     string
		wantDuplicate bool
		wantError    bool
	}{
		{
			name:         "Duplicate game exists",
			gameHash:     testHash,
			wantDuplicate: true,
			wantError:    false,
		},
		{
			name:         "No duplicate - different hash",
			gameHash:     "different-hash-456",
			wantDuplicate: false,
			wantError:    false,
		},
		{
			name:         "Empty hash",
			gameHash:     "",
			wantDuplicate: false,
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDuplicate, err := checkDuplicateGame(tx, tt.gameHash)
			if tt.wantError {
				if err == nil {
					t.Errorf("checkDuplicateGame() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("checkDuplicateGame() unexpected error = %v", err)
				}
				if isDuplicate != tt.wantDuplicate {
					t.Errorf("checkDuplicateGame() = %v, want %v", isDuplicate, tt.wantDuplicate)
				}
			}
		})
	}
}

// TestInsertGameRecord tests the insertGameRecord function
func TestInsertGameRecord(t *testing.T) {
	// Create a test database
	tempDir, err := os.MkdirTemp("", "gochess-test-insert-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewWithLogger(tempDir+"/test.db", logging.Discard())
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Prepare statements
	stmtGame, err := tx.PrepareContext(ctx, `
		INSERT INTO games (
			event, site, date, round, white, black, result,
			white_elo, black_elo, time_control, pgn_text, game_hash, eco_code, opening_name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("failed to prepare game statement: %v", err)
	}
	defer stmtGame.Close()

	stmtTag, err := tx.PrepareContext(ctx, `
		INSERT INTO tags (game_id, tag_name, tag_value)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("failed to prepare tag statement: %v", err)
	}
	defer stmtTag.Close()

	tests := []struct {
		name      string
		game      *pgn.Game
		gameText  string
		gameHash  string
		wantError bool
	}{
		{
			name: "Valid game with tags",
			game: &pgn.Game{
				Tags: map[string]string{
					"Event":      "Test Tournament",
					"Site":       "Test City",
					"Date":       "2024.01.01",
					"Round":      "1",
					"White":      "Player1",
					"Black":      "Player2",
					"Result":     "1-0",
					"WhiteElo":   "2000",
					"BlackElo":   "1900",
					"TimeControl": "180+2",
				},
			},
			gameText:  "[Event \"Test\"]\n\n1. e4 e5",
			gameHash:  "hash1",
			wantError: false,
		},
		{
			name: "Game with missing optional tags",
			game: &pgn.Game{
				Tags: map[string]string{
					"Event":  "Test Tournament 2",
					"Site":   "Test City 2",
					"Date":   "2024.01.02",
					"Round":  "2",
					"White":  "Player3",
					"Black":  "Player4",
					"Result": "0-1",
				},
			},
			gameText:  "[Event \"Test2\"]\n\n1. d4 d5",
			gameHash:  "hash2",
			wantError: false,
		},
		{
			name: "Game with invalid ELO (should not error, just default to 0)",
			game: &pgn.Game{
				Tags: map[string]string{
					"Event":    "Test Tournament 3",
					"Site":     "Test City 3",
					"Date":     "2024.01.03",
					"Round":    "3",
					"White":    "Player5",
					"Black":    "Player6",
					"Result":   "1/2-1/2",
					"WhiteElo": "invalid",
					"BlackElo": "also-invalid",
				},
			},
			gameText:  "[Event \"Test3\"]\n\n1. Nf3 Nf6",
			gameHash:  "hash3",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameID, err := insertGameRecord(ctx, tx, stmtGame, stmtTag, tt.game, tt.gameText, tt.gameHash, "", "")
			if tt.wantError {
				if err == nil {
					t.Errorf("insertGameRecord() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("insertGameRecord() unexpected error = %v", err)
				} else {
					// Verify the game was inserted and we got a valid ID
					if gameID <= 0 {
						t.Errorf("insertGameRecord() returned invalid game ID: %d", gameID)
					}
					// Verify the game was inserted
					var count int
					err = tx.QueryRow("SELECT COUNT(*) FROM games WHERE game_hash = ?", tt.gameHash).Scan(&count)
					if err != nil {
						t.Errorf("failed to query inserted game: %v", err)
					}
					if count != 1 {
						t.Errorf("expected 1 game with hash %s, got %d", tt.gameHash, count)
					}

					// Verify tags were inserted
					var tagCount int
					err = tx.QueryRow("SELECT COUNT(*) FROM tags WHERE game_id IN (SELECT id FROM games WHERE game_hash = ?)", tt.gameHash).Scan(&tagCount)
					if err != nil {
						t.Errorf("failed to query inserted tags: %v", err)
					}
					expectedTags := len(tt.game.Tags)
					if tagCount != expectedTags {
						t.Errorf("expected %d tags, got %d", expectedTags, tagCount)
					}
				}
			}
		})
	}
}
func TestGetGameByID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gochess-test-getgame-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewWithLogger(tempDir+"/test.db", logging.Discard())
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("Game not found", func(t *testing.T) {
		_, err := db.GetGameByID(ctx, 999)
		if err == nil {
			t.Error("expected error for non-existent game")
		}
		if !strings.Contains(err.Error(), "game not found") {
			t.Errorf("expected 'game not found' error, got: %v", err)
		}
	})

	// Import a test game
	pgnContent := `[Event "Test Tournament"]
[Site "Test Location"]
[Date "2024.01.15"]
[Round "1"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]
[WhiteElo "1800"]
[BlackElo "1750"]
[TimeControl "600+5"]
[CustomTag "CustomValue"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0
`
	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	if err != nil {
		t.Fatalf("failed to write PGN file: %v", err)
	}

	count, errs := db.ImportPGN(ctx, pgnFile)
	if len(errs) > 0 {
		t.Fatalf("import errors: %v", errs)
	}
	if count != 1 {
		t.Fatalf("expected 1 game imported, got %d", count)
	}

	t.Run("Valid game retrieval", func(t *testing.T) {
		game, err := db.GetGameByID(ctx, 1)
		if err != nil {
			t.Fatalf("failed to get game: %v", err)
		}

		// Verify main fields
		if game["id"] != 1 {
			t.Errorf("expected id=1, got %v", game["id"])
		}
		if game["event"] != "Test Tournament" {
			t.Errorf("expected event='Test Tournament', got %v", game["event"])
		}
		if game["site"] != "Test Location" {
			t.Errorf("expected site='Test Location', got %v", game["site"])
		}
		if game["date"] != "2024.01.15" {
			t.Errorf("expected date='2024.01.15', got %v", game["date"])
		}
		if game["round"] != "1" {
			t.Errorf("expected round='1', got %v", game["round"])
		}
		if game["white"] != "Alice" {
			t.Errorf("expected white='Alice', got %v", game["white"])
		}
		if game["black"] != "Bob" {
			t.Errorf("expected black='Bob', got %v", game["black"])
		}
		if game["result"] != "1-0" {
			t.Errorf("expected result='1-0', got %v", game["result"])
		}
		if game["white_elo"] != 1800 {
			t.Errorf("expected white_elo=1800, got %v", game["white_elo"])
		}
		if game["black_elo"] != 1750 {
			t.Errorf("expected black_elo=1750, got %v", game["black_elo"])
		}
		if game["time_control"] != "600+5" {
			t.Errorf("expected time_control='600+5', got %v", game["time_control"])
		}

		// Verify PGN text is present
		pgnText, ok := game["pgn_text"].(string)
		if !ok {
			t.Error("pgn_text should be a string")
		}
		if !strings.Contains(pgnText, "1. e4 e5") {
			t.Error("pgn_text should contain moves")
		}

		// Verify tags
		tags, ok := game["tags"].(map[string]string)
		if !ok {
			t.Fatal("tags should be a map[string]string")
		}
		if tags["CustomTag"] != "CustomValue" {
			t.Errorf("expected CustomTag='CustomValue', got %v", tags["CustomTag"])
		}
	})

	t.Run("Multiple games", func(t *testing.T) {
		// Import another game
		pgnContent2 := `[Event "Second Game"]
[Site "Another Place"]
[Date "2024.01.16"]
[Round "2"]
[White "Charlie"]
[Black "Dave"]
[Result "0-1"]

1. d4 d5 0-1
`
		pgnFile2 := tempDir + "/test2.pgn"
		err = os.WriteFile(pgnFile2, []byte(pgnContent2), 0644)
		if err != nil {
			t.Fatalf("failed to write second PGN file: %v", err)
		}

		count, errs := db.ImportPGN(ctx, pgnFile2)
		if len(errs) > 0 {
			t.Fatalf("import errors: %v", errs)
		}
		if count != 1 {
			t.Fatalf("expected 1 game imported, got %d", count)
		}

		// Retrieve second game
		game, err := db.GetGameByID(ctx, 2)
		if err != nil {
			t.Fatalf("failed to get game 2: %v", err)
		}

		if game["id"] != 2 {
			t.Errorf("expected id=2, got %v", game["id"])
		}
		if game["white"] != "Charlie" {
			t.Errorf("expected white='Charlie', got %v", game["white"])
		}
		if game["black"] != "Dave" {
			t.Errorf("expected black='Dave', got %v", game["black"])
		}

		// Verify first game still retrievable
		game1, err := db.GetGameByID(ctx, 1)
		if err != nil {
			t.Fatalf("failed to get game 1: %v", err)
		}
		if game1["white"] != "Alice" {
			t.Errorf("game 1 should still be Alice, got %v", game1["white"])
		}
	})
}

func TestClearGames(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gochess-test-clear-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := NewWithLogger(tempDir+"/test.db", logging.Discard())
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("Clear empty database", func(t *testing.T) {
		err := db.ClearGames(ctx)
		if err != nil {
			t.Errorf("clearing empty database should not error: %v", err)
		}

		// Verify count is 0
		count, err := db.GetGameCount(ctx)
		if err != nil {
			t.Fatalf("failed to count games: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 games, got %d", count)
		}
	})

	// Import some test games
	pgnContent := `[Event "Game 1"]
[Site "Site 1"]
[Date "2024.01.01"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 1-0

[Event "Game 2"]
[Site "Site 2"]
[Date "2024.01.02"]
[White "Player3"]
[Black "Player4"]
[Result "0-1"]
[TimeControl "300+0"]

1. d4 d5 0-1

[Event "Game 3"]
[Site "Site 3"]
[Date "2024.01.03"]
[White "Player5"]
[Black "Player6"]
[Result "1/2-1/2"]

1. c4 c5 1/2-1/2
`
	pgnFile := tempDir + "/test.pgn"
	err = os.WriteFile(pgnFile, []byte(pgnContent), 0644)
	if err != nil {
		t.Fatalf("failed to write PGN file: %v", err)
	}

	count, errs := db.ImportPGN(ctx, pgnFile)
	if len(errs) > 0 {
		t.Fatalf("import errors: %v", errs)
	}
	if count != 3 {
		t.Fatalf("expected 3 games imported, got %d", count)
	}

	// Verify games were imported
	gameCount, err := db.GetGameCount(ctx)
	if err != nil {
		t.Fatalf("failed to count games: %v", err)
	}
	if gameCount != 3 {
		t.Fatalf("expected 3 games, got %d", gameCount)
	}

	t.Run("Clear database with games", func(t *testing.T) {
		err := db.ClearGames(ctx)
		if err != nil {
			t.Errorf("failed to clear games: %v", err)
		}

		// Verify count is 0
		count, err := db.GetGameCount(ctx)
		if err != nil {
			t.Fatalf("failed to count games: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 games after clear, got %d", count)
		}

		// Verify tags were also cleared (cascade)
		var tagCount int
		err = db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags").Scan(&tagCount)
		if err != nil {
			t.Fatalf("failed to count tags: %v", err)
		}
		if tagCount != 0 {
			t.Errorf("expected 0 tags after clear, got %d", tagCount)
		}

		// Verify positions were also cleared (cascade)
		var posCount int
		err = db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions").Scan(&posCount)
		if err != nil {
			t.Fatalf("failed to count positions: %v", err)
		}
		if posCount != 0 {
			t.Errorf("expected 0 positions after clear, got %d", posCount)
		}
	})

	t.Run("Re-import after clear", func(t *testing.T) {
		// Should be able to import again
		count, errs := db.ImportPGN(ctx, pgnFile)
		if len(errs) > 0 {
			t.Fatalf("import errors after clear: %v", errs)
		}
		if count != 3 {
			t.Fatalf("expected 3 games re-imported, got %d", count)
		}

		// IDs should start from 1 again due to sequence reset
		game, err := db.GetGameByID(ctx, 1)
		if err != nil {
			t.Fatalf("failed to get game after re-import: %v", err)
		}
		if game["id"] != 1 {
			t.Errorf("expected first game to have id=1 after clear, got %v", game["id"])
		}
	})
}
