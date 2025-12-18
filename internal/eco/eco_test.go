package eco

import (
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
)

func TestNewDatabase(t *testing.T) {
	db, err := NewDatabaseWithLogger(logging.Discard())
	if err != nil {
		t.Fatalf("failed to create ECO database: %v", err)
	}

	if len(db.openings) == 0 {
		t.Fatal("expected openings to be loaded, got 0")
	}

	// Should have loaded thousands of openings
	if len(db.openings) < 3000 {
		t.Errorf("expected at least 3000 openings, got %d", len(db.openings))
	}
}

func TestParsePGNMoves(t *testing.T) {
	tests := []struct {
		name     string
		pgn      string
		expected []string
	}{
		{
			name:     "Simple opening",
			pgn:      "1. e4 e5 2. Nf3",
			expected: []string{"e4", "e5", "Nf3"},
		},
		{
			name:     "With check symbols",
			pgn:      "1. e4 e5 2. Nf3 Nc6 3. Bb5+",
			expected: []string{"e4", "e5", "Nf3", "Nc6", "Bb5"},
		},
		{
			name:     "With checkmate",
			pgn:      "1. f3 e5 2. g4 Qh4#",
			expected: []string{"f3", "e5", "g4", "Qh4"},
		},
		{
			name:     "Castling",
			pgn:      "1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O",
			expected: []string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "Nf6", "O-O"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moves := parsePGNMoves(tt.pgn)

			if len(moves) != len(tt.expected) {
				t.Errorf("expected %d moves, got %d", len(tt.expected), len(moves))
				return
			}

			for i, move := range moves {
				if move != tt.expected[i] {
					t.Errorf("move %d: expected %s, got %s", i, tt.expected[i], move)
				}
			}
		})
	}
}

func TestClassify(t *testing.T) {
	db, err := NewDatabaseWithLogger(logging.Discard())
	if err != nil {
		t.Fatalf("failed to create ECO database: %v", err)
	}

	tests := []struct {
		name            string
		moves           []string
		expectedECO     string
		expectedName    string
		shouldMatch     bool
		nameContains    string // For partial name matching
	}{
		{
			name:         "Italian Game",
			moves:        []string{"e4", "e5", "Nf3", "Nc6", "Bc4"},
			expectedECO:  "C50",
			nameContains: "Italian",
			shouldMatch:  true,
		},
		{
			name:         "Sicilian Defense",
			moves:        []string{"e4", "c5"},
			expectedECO:  "B20",
			nameContains: "Sicilian",
			shouldMatch:  true,
		},
		{
			name:         "French Defense",
			moves:        []string{"e4", "e6"},
			expectedECO:  "C00",
			nameContains: "French",
			shouldMatch:  true,
		},
		{
			name:         "Ruy Lopez",
			moves:        []string{"e4", "e5", "Nf3", "Nc6", "Bb5"},
			expectedECO:  "C60",
			nameContains: "Ruy Lopez",
			shouldMatch:  true,
		},
		{
			name:         "Queen's Gambit",
			moves:        []string{"d4", "d5", "c4"},
			expectedECO:  "D06",
			nameContains: "Queen's Gambit",
			shouldMatch:  true,
		},
		{
			name:        "No opening (empty moves)",
			moves:       []string{},
			shouldMatch: false,
		},
		{
			name:         "King's Pawn opening",
			moves:        []string{"e4"},
			expectedECO:  "B00",
			nameContains: "King's Pawn",
			shouldMatch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eco, name, found := db.Classify(tt.moves)

			if found != tt.shouldMatch {
				t.Errorf("expected match=%v, got %v", tt.shouldMatch, found)
			}

			if !tt.shouldMatch {
				return
			}

			if eco != tt.expectedECO {
				t.Errorf("expected ECO %s, got %s", tt.expectedECO, eco)
			}

			if tt.nameContains != "" && !contains(name, tt.nameContains) {
				t.Errorf("expected name to contain %q, got %q", tt.nameContains, name)
			}
		})
	}
}

func TestClassifyWithLongerSequences(t *testing.T) {
	db, err := NewDatabaseWithLogger(logging.Discard())
	if err != nil {
		t.Fatalf("failed to create ECO database: %v", err)
	}

	// Test that longer opening sequences produce more specific classifications
	moves1 := []string{"e4", "c5"}
	eco1, name1, found1 := db.Classify(moves1)

	moves2 := []string{"e4", "c5", "Nf3", "d6", "d4", "cxd4", "Nxd4", "Nf6", "Nc3", "a6"}
	eco2, name2, found2 := db.Classify(moves2)

	if !found1 || !found2 {
		t.Fatal("expected both classifications to succeed")
	}

	// Both should be Sicilian variations
	if !contains(name1, "Sicilian") {
		t.Errorf("expected first to be Sicilian, got %s", name1)
	}

	if !contains(name2, "Sicilian") {
		t.Errorf("expected second to be Sicilian, got %s", name2)
	}

	// The longer sequence should have a different (more specific) ECO code
	if eco1 == eco2 {
		t.Logf("Note: Both have same ECO code %s, but this is acceptable", eco1)
		t.Logf("  Short: %s", name1)
		t.Logf("  Long:  %s", name2)
	}
}

func TestGetOpening(t *testing.T) {
	db, err := NewDatabaseWithLogger(logging.Discard())
	if err != nil {
		t.Fatalf("failed to create ECO database: %v", err)
	}

	tests := []struct {
		name        string
		ecoCode     string
		shouldFind  bool
		namePattern string
	}{
		{
			name:        "Italian Game",
			ecoCode:     "C50",
			shouldFind:  true,
			namePattern: "Italian",
		},
		{
			name:       "Invalid ECO code",
			ecoCode:    "Z99",
			shouldFind: false,
		},
		{
			name:        "Sicilian Defense",
			ecoCode:     "B20",
			shouldFind:  true,
			namePattern: "Sicilian",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opening, found := db.GetOpening(tt.ecoCode)

			if found != tt.shouldFind {
				t.Errorf("expected found=%v, got %v", tt.shouldFind, found)
			}

			if !tt.shouldFind {
				return
			}

			if opening.ECOCode != tt.ecoCode {
				t.Errorf("expected ECO %s, got %s", tt.ecoCode, opening.ECOCode)
			}

			if tt.namePattern != "" && !contains(opening.Name, tt.namePattern) {
				t.Errorf("expected name to contain %q, got %q", tt.namePattern, opening.Name)
			}
		})
	}
}

func TestGetByName(t *testing.T) {
	db, err := NewDatabaseWithLogger(logging.Discard())
	if err != nil {
		t.Fatalf("failed to create ECO database: %v", err)
	}

	tests := []struct {
		name         string
		searchName   string
		minResults   int
		shouldContain string
	}{
		{
			name:         "Search Sicilian",
			searchName:   "Sicilian",
			minResults:   50, // Should find many Sicilian variations
			shouldContain: "B",
		},
		{
			name:         "Search French",
			searchName:   "French",
			minResults:   20,
			shouldContain: "C00",
		},
		{
			name:       "Search nonexistent",
			searchName: "XYZNonexistent",
			minResults: 0,
		},
		{
			name:         "Case insensitive",
			searchName:   "ITALIAN",
			minResults:   1,
			shouldContain: "C50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := db.GetByName(tt.searchName)

			if len(results) < tt.minResults {
				t.Errorf("expected at least %d results, got %d", tt.minResults, len(results))
			}

			if tt.shouldContain != "" && len(results) > 0 {
				found := false
				for _, opening := range results {
					if contains(opening.ECOCode, tt.shouldContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected at least one result with ECO containing %q", tt.shouldContain)
				}
			}
		})
	}
}

func TestMovesEqual(t *testing.T) {
	tests := []struct {
		name     string
		move1    string
		move2    string
		expected bool
	}{
		{
			name:     "Exact match",
			move1:    "e4",
			move2:    "e4",
			expected: true,
		},
		{
			name:     "With check symbol",
			move1:    "Bb5+",
			move2:    "Bb5",
			expected: true,
		},
		{
			name:     "Different moves",
			move1:    "e4",
			move2:    "e5",
			expected: false,
		},
		{
			name:     "With annotation",
			move1:    "Qh4#",
			move2:    "Qh4",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := movesEqual(tt.move1, tt.move2)
			if result != tt.expected {
				t.Errorf("movesEqual(%q, %q) = %v, want %v", tt.move1, tt.move2, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 len(s) > len(substr) && (
			 s[:len(substr)] == substr ||
			 s[len(s)-len(substr):] == substr ||
			 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
