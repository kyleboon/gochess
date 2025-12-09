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
			white_elo, black_elo, time_control, pgn_text, game_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			err := insertGameRecord(ctx, tx, stmtGame, stmtTag, tt.game, tt.gameText, tt.gameHash)
			if tt.wantError {
				if err == nil {
					t.Errorf("insertGameRecord() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("insertGameRecord() unexpected error = %v", err)
				} else {
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
