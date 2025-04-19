package internal

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// PerftStats tracks various statistics during perft
type PerftStats struct {
	Nodes           int
	Captures        int
	EnPassant       int
	Castles         int
	Promotions      int
	Checks          int
	DiscoveryChecks int
	DoubleChecks    int
	Checkmates      int
}

// PerftExpected contains expected perft results for validation
var PerftExpected = []PerftStats{
	{Nodes: 1},                              // depth 0
	{Nodes: 20},                             // depth 1
	{Nodes: 400},                            // depth 2
	{Nodes: 8902, Captures: 34, Checks: 12}, // depth 3
	{Nodes: 197281, Captures: 1576, Checks: 469, Checkmates: 8},                                                                       // depth 4
	{Nodes: 4865609, Captures: 82719, EnPassant: 258, Checks: 27351, DiscoveryChecks: 6, Checkmates: 347},                             // depth 5
	{Nodes: 119060324, Captures: 2812008, EnPassant: 5248, Checks: 809099, DiscoveryChecks: 329, DoubleChecks: 46, Checkmates: 10828}, // depth 6
	// For depths 7-9, we won't validate all stats as they take too long to compute
	{Nodes: 3195901860},    // depth 7
	{Nodes: 84998978956},   // depth 8
	{Nodes: 2439530234167}, // depth 9
}

// isDiscoveryCheck determines if a move is a discovery check
func isDiscoveryCheck(b *Board, move Move) bool {
	// A discovery check occurs when a piece moves out of the way to reveal an attack on the king
	oldBoard := b.Copy()
	newBoard := b.MakeMove(move)

	// Get the opponent's king position
	kingPos := newBoard.find(newBoard.opp(King), A1, H8)
	if kingPos == NoSquare {
		return false
	}

	// Check if it's check in the new position
	check, _ := newBoard.IsCheckOrMate()
	if !check {
		return false
	}

	// The piece that moved shouldn't be the one giving check
	pieceType := oldBoard.Piece[move.From].Type()
	kingFile, kingRank := kingPos.File(), kingPos.Rank()
	moveToFile, moveToRank := move.To.File(), move.To.Rank()

	switch pieceType {
	case Queen:
		// If the queen is aligned with the king, it might be giving the check directly
		if kingFile == moveToFile || kingRank == moveToRank ||
			abs(kingFile-moveToFile) == abs(kingRank-moveToRank) {
			return false
		}
	case Rook:
		// If the rook is aligned with the king, it might be giving the check directly
		if kingFile == moveToFile || kingRank == moveToRank {
			return false
		}
	case Bishop:
		// If the bishop is aligned with the king, it might be giving the check directly
		if abs(kingFile-moveToFile) == abs(kingRank-moveToRank) {
			return false
		}
	case Knight:
		// Check if knight is giving the check
		dx, dy := abs(kingFile-moveToFile), abs(kingRank-moveToRank)
		if (dx == 1 && dy == 2) || (dx == 2 && dy == 1) {
			return false
		}
	case Pawn:
		// Check if pawn is giving the check
		dx := abs(kingFile - moveToFile)
		if dx <= 1 && moveToRank-kingRank == oldBoard.SideToMove*2-1 {
			return false
		}
	}

	return true
}

// isDoubleCheck determines if a move results in a double check
func isDoubleCheck(b *Board, move Move) bool {
	newBoard := b.MakeMove(move)

	// Get the opponent's king position
	kingPos := newBoard.find(newBoard.opp(King), A1, H8)
	if kingPos == NoSquare {
		return false
	}

	// Count how many pieces are giving check
	checkCount := 0
	for i, piece := range newBoard.Piece {
		if piece == NoPiece || piece.Color() != newBoard.SideToMove {
			continue
		}

		sq := Sq(i)

		// Check if this piece is attacking the king
		switch piece.Type() {
		case Pawn:
			// Pawns attack diagonally
			if abs(sq.File()-kingPos.File()) == 1 &&
				sq.Rank()-kingPos.Rank() == newBoard.SideToMove*2-1 {
				checkCount++
			}
		case Knight:
			// Knights attack in an L shape
			dx, dy := abs(sq.File()-kingPos.File()), abs(sq.Rank()-kingPos.Rank())
			if (dx == 1 && dy == 2) || (dx == 2 && dy == 1) {
				checkCount++
			}
		case Bishop:
			// Bishops attack diagonally
			if abs(sq.File()-kingPos.File()) == abs(sq.Rank()-kingPos.Rank()) {
				// Check that the path is clear
				if isPathClear(newBoard, sq, kingPos) {
					checkCount++
				}
			}
		case Rook:
			// Rooks attack in straight lines
			if sq.File() == kingPos.File() || sq.Rank() == kingPos.Rank() {
				// Check that the path is clear
				if isPathClear(newBoard, sq, kingPos) {
					checkCount++
				}
			}
		case Queen:
			// Queens attack like bishops and rooks combined
			if sq.File() == kingPos.File() || sq.Rank() == kingPos.Rank() ||
				abs(sq.File()-kingPos.File()) == abs(sq.Rank()-kingPos.Rank()) {
				// Check that the path is clear
				if isPathClear(newBoard, sq, kingPos) {
					checkCount++
				}
			}
		}

		if checkCount >= 2 {
			return true
		}
	}

	return false
}

// isPathClear checks if there are no pieces between the two squares
func isPathClear(b *Board, from, to Sq) bool {
	dx := sign(to.File() - from.File())
	dy := sign(to.Rank() - from.Rank())

	x, y := from.File(), from.Rank()
	for {
		x += dx
		y += dy
		sq := Square(x, y)
		if sq == to {
			return true
		}
		if b.Piece[sq] != NoPiece {
			return false
		}
	}
}

func sign(x int) int {
	if x < 0 {
		return -1
	} else if x > 0 {
		return 1
	}
	return 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// perftWithStats performs a perft search with detailed statistics
func perftWithStats(b *Board, depth int, stats *PerftStats) {
	if depth == 0 {
		stats.Nodes++
		return
	}

	moves := b.LegalMoves()
	if depth == 1 {
		for _, move := range moves {
			stats.Nodes++

			// Count captures
			if b.Piece[move.To] != NoPiece {
				stats.Captures++
			}

			// Count en passant captures
			if move.To == b.EpSquare && b.Piece[move.From].Type() == Pawn {
				stats.EnPassant++
			}

			// Count castling moves
			if b.Piece[move.From].Type() == King && abs(move.From.File()-move.To.File()) > 1 {
				stats.Castles++
			}

			// Count promotions
			if move.Promotion != NoPiece {
				stats.Promotions++
			}

			// Make the move to check for checks and checkmates
			newBoard := b.MakeMove(move)
			check, mate := newBoard.IsCheckOrMate()
			if check {
				stats.Checks++

				// Check for discovery and double checks
				if isDiscoveryCheck(b, move) {
					stats.DiscoveryChecks++
				}

				if isDoubleCheck(b, move) {
					stats.DoubleChecks++
				}

				if mate {
					stats.Checkmates++
				}
			}
		}
		return
	}

	for _, move := range moves {
		newBoard := b.MakeMove(move)
		perftWithStats(newBoard, depth-1, stats)
	}
}

func TestPerft(t *testing.T) {
	// We'll test starting from the initial position
	board, err := ParseFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	assert.NoError(t, err)

	// Run perft for depths 0-4 with full validation
	// Higher depths would take too long for standard tests
	maxTestDepth := 4

	for depth := 0; depth <= maxTestDepth; depth++ {
		t.Run(fmt.Sprintf("Perft(%d)", depth), func(t *testing.T) {
			stats := &PerftStats{}

			// Use a timeout for safety
			done := make(chan bool)

			go func() {
				startTime := time.Now()
				perftWithStats(board, depth, stats)
				duration := time.Since(startTime)
				nodesPerSecond := float64(stats.Nodes) / duration.Seconds()

				t.Logf("Perft(%d): %d nodes in %v (%.2f nodes/s)",
					depth, stats.Nodes, duration, nodesPerSecond)

				done <- true
			}()

			timeout := 2 * time.Minute
			if depth <= 3 {
				timeout = 10 * time.Second
			}

			select {
			case <-done:
				// Test passed, validate results
				expectedStats := PerftExpected[depth]
				assert.Equal(t, expectedStats.Nodes, stats.Nodes, "Node count mismatch")

				if depth >= 3 {
					assert.Equal(t, expectedStats.Captures, stats.Captures, "Capture count mismatch")
					assert.Equal(t, expectedStats.Checks, stats.Checks, "Check count mismatch")
				}

				if depth >= 4 {
					assert.Equal(t, expectedStats.Checkmates, stats.Checkmates, "Checkmate count mismatch")
				}

				if depth >= 5 {
					assert.Equal(t, expectedStats.EnPassant, stats.EnPassant, "En passant count mismatch")
					assert.Equal(t, expectedStats.DiscoveryChecks, stats.DiscoveryChecks, "Discovery check count mismatch")
				}

				if depth >= 6 {
					assert.Equal(t, expectedStats.DoubleChecks, stats.DoubleChecks, "Double check count mismatch")
				}

			case <-time.After(timeout):
				t.Fatalf("Perft(%d) timed out after %v", depth, timeout)
			}
		})
	}
}

// For running individual perft tests at specific depths
func TestPerftAtDepth(t *testing.T) {
	// Skip this in normal testing
	t.Skip("This test is too slow for regular testing. Unskip to run manually.")

	board, err := ParseFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	assert.NoError(t, err)

	depth := 6 // Change this to the desired depth
	stats := &PerftStats{}

	startTime := time.Now()
	perftWithStats(board, depth, stats)
	duration := time.Since(startTime)

	t.Logf("Perft(%d) results:", depth)
	t.Logf("  Nodes:           %d", stats.Nodes)
	t.Logf("  Captures:        %d", stats.Captures)
	t.Logf("  En Passant:      %d", stats.EnPassant)
	t.Logf("  Castles:         %d", stats.Castles)
	t.Logf("  Promotions:      %d", stats.Promotions)
	t.Logf("  Checks:          %d", stats.Checks)
	t.Logf("  Discovery Checks: %d", stats.DiscoveryChecks)
	t.Logf("  Double Checks:   %d", stats.DoubleChecks)
	t.Logf("  Checkmates:      %d", stats.Checkmates)
	t.Logf("Time: %v (%.2f nodes/s)", duration, float64(stats.Nodes)/duration.Seconds())
}

// TestPerftDivide runs a perft divide test - showing the node count for each move at the root
func TestPerftDivide(t *testing.T) {
	t.Skip("This test is informational only. Unskip to run manually.")

	board, err := ParseFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	assert.NoError(t, err)

	depth := 5 // Change this to the desired depth

	moves := board.LegalMoves()
	var totalNodes int

	t.Logf("Perft Divide at depth %d:", depth)
	for _, move := range moves {
		stats := &PerftStats{}
		newBoard := board.MakeMove(move)

		perftWithStats(newBoard, depth-1, stats)
		t.Logf("  %s: %d", move.San(board), stats.Nodes)
		totalNodes += stats.Nodes
	}

	t.Logf("Total nodes: %d", totalNodes)
}
