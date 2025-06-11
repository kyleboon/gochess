package db

import (
	"os"
	"testing"
)

func TestImportPGN_WithFEN(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gochess-test-db-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := New(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	// Test with a PGN file that has a FEN, which should be ignored
	pgnPath := "../../testdata/invalid_fen.pgn"
	count, errors := db.ImportPGN(pgnPath)

	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, but got %d: %v", len(errors), errors)
	}

	if count != 1 {
		t.Fatalf("expected 1 game to be imported, but got %d", count)
	}
}
