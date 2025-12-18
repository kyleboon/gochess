package db

import (
	"regexp"
	"strings"

	"github.com/kyleboon/gochess/internal"
	"github.com/kyleboon/gochess/internal/pgn"
)

// extractMoveStrings extracts SAN move notation from a parsed game
// It traverses the game tree and generates SAN notation for each move
func extractMoveStrings(game *pgn.Game) []string {
	moves := make([]string, 0, 100)

	if game.Root == nil || game.Root.Next == nil {
		return moves
	}

	// Walk through the game tree and convert each move to SAN
	board := game.Root.Board
	for node := game.Root.Next; node != nil; node = node.Next {
		san := moveToSAN(node.Move, board)
		moves = append(moves, san)
		board = node.Board
	}

	return moves
}

// moveToSAN converts an internal.Move to Standard Algebraic Notation
// This is a simplified implementation that generates basic SAN without full disambiguation
func moveToSAN(m internal.Move, b *internal.Board) string {
	if m == internal.NullMove {
		return "--"
	}

	// Check if this is castling
	piece := b.Piece[m.From]
	if piece.Type() == internal.King {
		// Detect castling by king movement
		fromFile := m.From.File()
		toFile := m.To.File()
		if fromFile == 4 { // King starts on e-file
			if toFile == 6 {
				return "O-O" // Kingside
			} else if toFile == 2 {
				return "O-O-O" // Queenside
			}
		}
	}

	pieceType := piece.Type()
	var san strings.Builder

	// Add piece letter (nothing for pawns)
	if pieceType != internal.Pawn {
		san.WriteRune(internal.PieceRunes[pieceType])
	}

	// Check for captures
	isCapture := b.Piece[m.To] != internal.NoPiece

	// For pawn captures, we need the file
	if pieceType == internal.Pawn && isCapture {
		san.WriteString(m.From.String()[:1]) // Just the file (a-h)
	}

	// Add capture symbol
	if isCapture {
		san.WriteRune('x')
	}

	// Add destination square
	san.WriteString(m.To.String())

	// Add promotion
	if m.Promotion != internal.NoPiece {
		san.WriteRune('=')
		san.WriteRune(internal.PieceRunes[m.Promotion])
	}

	// Note: We skip check/checkmate symbols for simplicity
	// This is acceptable for ECO classification which primarily uses early game moves

	return san.String()
}

// extractMovesFromPGN extracts move strings from raw PGN text
// This is an alternative approach that parses the PGN text directly
func extractMovesFromPGN(pgnText string) []string {
	moves := make([]string, 0, 100)

	// Remove comments (anything in braces or parentheses)
	re := regexp.MustCompile(`\{[^}]*\}|\([^)]*\)`)
	pgnText = re.ReplaceAllString(pgnText, "")

	// Remove result markers
	pgnText = strings.ReplaceAll(pgnText, "1-0", "")
	pgnText = strings.ReplaceAll(pgnText, "0-1", "")
	pgnText = strings.ReplaceAll(pgnText, "1/2-1/2", "")
	pgnText = strings.ReplaceAll(pgnText, "*", "")

	// Split by whitespace
	tokens := strings.Fields(pgnText)

	// Extract moves (skip move numbers like "1.", "2.", etc.)
	movePattern := regexp.MustCompile(`^\d+\.+$`)
	for _, token := range tokens {
		if !movePattern.MatchString(token) && token != "" {
			// Remove any annotations like !?, !, ??, etc.
			move := strings.TrimRight(token, "!?")
			if move != "" {
				moves = append(moves, move)
			}
		}
	}

	return moves
}
