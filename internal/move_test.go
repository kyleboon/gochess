package internal

import (
	"testing"
)

func TestAlgebraicNotation(t *testing.T) {
	// Test setup: Create a board with the starting position
	board, err := ParseFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatalf("Failed to create starting board position: %v", err)
	}

	// Test parsing the move e4 (pawn from e2 to e4)
	move, err := board.ParseMove("e4")
	if err != nil {
		t.Fatalf("Failed to parse move e4: %v", err)
	}

	// Verify the move properties
	if move.From != E2 {
		t.Errorf("Expected from square to be E2, got %s", move.From.String())
	}
	if move.To != E4 {
		t.Errorf("Expected to square to be E4, got %s", move.To.String())
	}
	if move.Promotion != NoPiece {
		t.Errorf("Expected no promotion piece, got %v", move.Promotion)
	}

	// Verify the piece at the from square is a white pawn
	if board.Piece[move.From] != WP {
		t.Errorf("Expected piece at %s to be white pawn, got %v", move.From.String(), board.Piece[move.From])
	}

	// Make the move
	newBoard := board.MakeMove(move)

	// Verify the move was made correctly
	if newBoard.Piece[E2] != NoPiece {
		t.Errorf("Expected no piece at E2 after move, got %v", newBoard.Piece[E2])
	}
	if newBoard.Piece[E4] != WP {
		t.Errorf("Expected white pawn at E4 after move, got %v", newBoard.Piece[E4])
	}

	// Verify the side to move changed
	if newBoard.SideToMove != Black {
		t.Errorf("Expected side to move to be Black after move, got %v", newBoard.SideToMove)
	}

	// Verify that the algebraic notation of the move is correct
	san := move.San(newBoard)
	if san != "e4" {
		t.Errorf("Expected SAN notation to be 'e4', got '%s'", san)
	}
}
