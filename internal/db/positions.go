package db

import (
	"github.com/kyleboon/gochess/internal"
	"github.com/kyleboon/gochess/internal/pgn"
)

// Position represents a chess position at a specific point in a game
type Position struct {
	MoveNumber int     // Half-move (ply) number, starting from 0
	FEN        string  // Position in FEN notation
	NextMove   string  // Move played from this position in SAN notation
}

// ExtractPositions walks through a parsed PGN game and extracts all positions.
// Returns a slice of positions, where each position includes the FEN before
// the move and the move that was played from that position.
func ExtractPositions(game *pgn.Game) []Position {
	positions := make([]Position, 0, 80) // Average game has ~40 moves * 2 = 80 plies

	// Start from the initial position (root node has starting position)
	if game.Root == nil || game.Root.Board == nil {
		return positions
	}

	// The root node contains the starting position
	startingBoard := game.Root.Board
	moveNumber := 0

	// Add starting position if there's a first move
	if game.Root.Next != nil {
		positions = append(positions, Position{
			MoveNumber: moveNumber,
			FEN:        startingBoard.Fen(),
			NextMove:   formatMove(game.Root.Next.Move, startingBoard),
		})
		moveNumber++
	}

	// Walk through the main variation (linked list of nodes)
	for node := game.Root.Next; node != nil; node = node.Next {
		// node.Board is the position AFTER the move was played
		// We want to store positions with the move that will be played

		if node.Next != nil {
			// There's another move after this one, so store this position
			// with the next move to be played
			positions = append(positions, Position{
				MoveNumber: moveNumber,
				FEN:        node.Board.Fen(),
				NextMove:   formatMove(node.Next.Move, node.Board),
			})
			moveNumber++
		} else {
			// This is the final position of the game (no next move)
			positions = append(positions, Position{
				MoveNumber: moveNumber,
				FEN:        node.Board.Fen(),
				NextMove:   "", // No move played from final position
			})
		}
	}

	return positions
}

// formatMove converts a Move to Standard Algebraic Notation (SAN)
// This is a simplified implementation that just uses UCI notation for now
// TODO: Implement proper SAN formatting with disambiguation
func formatMove(m internal.Move, b *internal.Board) string {
	if m == internal.NullMove {
		return "--"
	}

	// For now, use simple UCI notation (e.g., "e2e4", "e7e8q")
	// This should ideally be converted to proper SAN (e.g., "e4", "e8=Q")
	move := m.From.String() + m.To.String()

	if m.Promotion != internal.NoPiece {
		// Add promotion piece in lowercase (UCI style)
		piece := internal.PieceRunes[m.Promotion]
		if piece >= 'A' && piece <= 'Z' {
			piece = piece + 32 // Convert to lowercase
		}
		move += string(piece)
	}

	return move
}
