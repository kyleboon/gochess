package rampart

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/kyleboon/gochess/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMoveGeneration tests that the move generator produces the correct moves for each test case
func TestMoveGeneration(t *testing.T) {
	// Find test data files
	testFiles, err := LoadAllTestFiles("../../testdata/rampart")
	require.NoError(t, err, "Failed to load test files")
	require.NotEmpty(t, testFiles, "No test files found")

	// Count total test cases
	var totalTests, passedTests int

	// Test each file
	for category, testFile := range testFiles {
		t.Run(category, func(t *testing.T) {
			for i, testCase := range testFile.TestCases {
				totalTests++
				testName := fmt.Sprintf("Case_%d", i+1)
				if testCase.Start.Description != "" {
					testName = fmt.Sprintf("%s_%s", testName, strings.ReplaceAll(testCase.Start.Description, " ", "_"))
				}

				t.Run(testName, func(t *testing.T) {
					// Parse the starting FEN position
					board, err := internal.ParseFen(testCase.Start.FEN)
					require.NoError(t, err, "Failed to parse starting FEN: %s", testCase.Start.FEN)

					// Generate legal moves
					moves := board.LegalMoves()

					// Create a map of resulting FEN positions from our move generator
					generatedFENs := make(map[string]bool)
					moveToFEN := make(map[string]string) // Used for better error reporting

					for _, move := range moves {
						resultBoard := board.MakeMove(move)
						fen := resultBoard.Fen()
						generatedFENs[fen] = true

						// Store the move in algebraic notation for error reporting
						san := move.San(board)
						moveToFEN[san] = fen
					}

					// Create a map of expected FEN positions
					expectedFENs := make(map[string]bool)
					expectedMoves := make(map[string]bool)

					for _, expected := range testCase.Expected {
						expectedFENs[expected.FEN] = true
						expectedMoves[expected.Move] = true
					}

					// Check if we have the correct number of moves
					assert.Equal(t, len(expectedFENs), len(generatedFENs),
						"Number of generated positions doesn't match expected")

					// Check that all expected positions are generated
					missingFENs := []string{}
					for expectedFEN := range expectedFENs {
						if !generatedFENs[expectedFEN] {
							missingFENs = append(missingFENs, expectedFEN)
						}
					}

					// Check for positions we generated but weren't expected
					unexpectedFENs := []string{}
					for generatedFEN := range generatedFENs {
						if !expectedFENs[generatedFEN] {
							unexpectedFENs = append(unexpectedFENs, generatedFEN)
						}
					}

					// Check if the move names match
					generatedMoves := make(map[string]bool)
					for _, move := range moves {
						// Get the algebraic notation of the move
						san := move.San(board)
						// Remove checkmate symbol (#) as our implementation might not add this
						san = strings.TrimSuffix(san, "#")
						generatedMoves[san] = true
					}

					missingMoves := []string{}
					for expectedMove := range expectedMoves {
						cleanMove := strings.TrimSuffix(expectedMove, "#")
						if !generatedMoves[cleanMove] {
							missingMoves = append(missingMoves, expectedMove)
						}
					}

					unexpectedMoves := []string{}
					for generatedMove := range generatedMoves {
						found := false
						for expectedMove := range expectedMoves {
							cleanExpectedMove := strings.TrimSuffix(expectedMove, "#")
							if cleanExpectedMove == generatedMove {
								found = true
								break
							}
						}
						if !found {
							unexpectedMoves = append(unexpectedMoves, generatedMove)
						}
					}

					// Sort for consistent output
					sort.Strings(missingFENs)
					sort.Strings(unexpectedFENs)
					sort.Strings(missingMoves)
					sort.Strings(unexpectedMoves)

					// Check if test passed
					if len(missingFENs) == 0 && len(unexpectedFENs) == 0 {
						passedTests++
					}

					// Report errors
					if len(missingFENs) > 0 {
						t.Errorf("Missing expected positions (%d): %v", len(missingFENs), missingFENs)
					}
					if len(unexpectedFENs) > 0 {
						t.Errorf("Unexpected positions generated (%d): %v", len(unexpectedFENs), unexpectedFENs)
					}
					if len(missingMoves) > 0 {
						t.Errorf("Missing expected moves (%d): %v", len(missingMoves), missingMoves)
					}
					if len(unexpectedMoves) > 0 {
						t.Errorf("Unexpected moves generated (%d): %v", len(unexpectedMoves), unexpectedMoves)
					}
				})
			}
		})
	}

	t.Logf("Passed %d/%d tests (%.1f%%)", passedTests, totalTests, float64(passedTests)/float64(totalTests)*100)
}
