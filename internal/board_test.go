package internal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPieceColor(t *testing.T) {
	assert.Equal(t, White, Piece(WP).Color())
	assert.Equal(t, Black, Piece(BP).Color())
	assert.Equal(t, White, Piece(WN).Color())
	assert.Equal(t, Black, Piece(BN).Color())
	assert.Equal(t, White, Piece(WB).Color())
	assert.Equal(t, Black, Piece(BB).Color())
	assert.Equal(t, White, Piece(WR).Color())
	assert.Equal(t, Black, Piece(BR).Color())
	assert.Equal(t, White, Piece(WQ).Color())
	assert.Equal(t, Black, Piece(BQ).Color())
	assert.Equal(t, White, Piece(WK).Color())
	assert.Equal(t, Black, Piece(BK).Color())
	assert.Equal(t, NoPiece, Piece(NoPiece).Color())
}

func TestPieceType(t *testing.T) {
	assert.Equal(t, Pawn, Piece(WP).Type())
	assert.Equal(t, Pawn, Piece(BP).Type())
	assert.Equal(t, Knight, Piece(WN).Type())
	assert.Equal(t, Knight, Piece(BN).Type())
	assert.Equal(t, Bishop, Piece(WB).Type())
	assert.Equal(t, Bishop, Piece(BB).Type())
	assert.Equal(t, Rook, Piece(WR).Type())
	assert.Equal(t, Rook, Piece(BR).Type())
	assert.Equal(t, Queen, Piece(WQ).Type())
	assert.Equal(t, Queen, Piece(BQ).Type())
	assert.Equal(t, King, Piece(WK).Type())
	assert.Equal(t, King, Piece(BK).Type())
	assert.Equal(t, NoPiece, Piece(NoPiece).Type())
}

func TestPieceFromChar(t *testing.T) {
	assert.Equal(t, Piece(WP), pieceFromChar('P'))
	assert.Equal(t, Piece(BP), pieceFromChar('p'))
	assert.Equal(t, Piece(WN), pieceFromChar('N'))
	assert.Equal(t, Piece(BN), pieceFromChar('n'))
	assert.Equal(t, Piece(WB), pieceFromChar('B'))
	assert.Equal(t, Piece(BB), pieceFromChar('b'))
	assert.Equal(t, Piece(WR), pieceFromChar('R'))
	assert.Equal(t, Piece(BR), pieceFromChar('r'))
	assert.Equal(t, Piece(WQ), pieceFromChar('Q'))
	assert.Equal(t, Piece(BQ), pieceFromChar('q'))
	assert.Equal(t, Piece(WK), pieceFromChar('K'))
	assert.Equal(t, Piece(BK), pieceFromChar('k'))
	assert.Equal(t, Piece(NoPiece), pieceFromChar('.'))
}

func TestSquare(t *testing.T) {
	assert.Equal(t, A1, Square(0, 0))
	assert.Equal(t, B2, Square(1, 1))
	assert.Equal(t, C3, Square(2, 2))
	assert.Equal(t, D4, Square(3, 3))
	assert.Equal(t, E5, Square(4, 4))
	assert.Equal(t, F6, Square(5, 5))
	assert.Equal(t, G7, Square(6, 6))
	assert.Equal(t, H8, Square(7, 7))
}

func TestFile(t *testing.T) {
	assert.Equal(t, FileA, A1.File())
	assert.Equal(t, FileB, B2.File())
	assert.Equal(t, FileC, C3.File())
	assert.Equal(t, FileD, D4.File())
	assert.Equal(t, FileE, E5.File())
	assert.Equal(t, FileF, F6.File())
	assert.Equal(t, FileG, G7.File())
	assert.Equal(t, FileH, H8.File())
}

func TestRank(t *testing.T) {
	assert.Equal(t, Rank1, A1.Rank())
	assert.Equal(t, Rank2, B2.Rank())
	assert.Equal(t, Rank3, C3.Rank())
	assert.Equal(t, Rank4, D4.Rank())
	assert.Equal(t, Rank5, E5.Rank())
	assert.Equal(t, Rank6, F6.Rank())
	assert.Equal(t, Rank7, G7.Rank())
	assert.Equal(t, Rank8, H8.Rank())
}

func TestRelativeRank(t *testing.T) {
	assert.Equal(t, Rank1, A1.RelativeRank(White))
	assert.Equal(t, Rank8, A1.RelativeRank(Black))
	assert.Equal(t, Rank2, B2.RelativeRank(White))
	assert.Equal(t, Rank7, B2.RelativeRank(Black))
	assert.Equal(t, Rank3, C3.RelativeRank(White))
	assert.Equal(t, Rank6, C3.RelativeRank(Black))
	assert.Equal(t, Rank4, D4.RelativeRank(White))
	assert.Equal(t, Rank5, D4.RelativeRank(Black))
	assert.Equal(t, Rank5, E5.RelativeRank(White))
	assert.Equal(t, Rank4, E5.RelativeRank(Black))
	assert.Equal(t, Rank6, F6.RelativeRank(White))
	assert.Equal(t, Rank3, F6.RelativeRank(Black))
	assert.Equal(t, Rank7, G7.RelativeRank(White))
	assert.Equal(t, Rank2, G7.RelativeRank(Black))
	assert.Equal(t, Rank8, H8.RelativeRank(White))
	assert.Equal(t, Rank1, H8.RelativeRank(Black))
}

func TestSquareString(t *testing.T) {
	assert.Equal(t, "-", NoSquare.String())
	assert.Equal(t, "a1", A1.String())
	assert.Equal(t, "b2", B2.String())
	assert.Equal(t, "c3", C3.String())
	assert.Equal(t, "d4", D4.String())
	assert.Equal(t, "e5", E5.String())
	assert.Equal(t, "f6", F6.String())
	assert.Equal(t, "g7", G7.String())
	assert.Equal(t, "h8", H8.String())
}

func TestSquareFromString(t *testing.T) {
	assert.Equal(t, A1, squareFromString("a1"))
	assert.Equal(t, B2, squareFromString("b2"))
	assert.Equal(t, C3, squareFromString("c3"))
	assert.Equal(t, D4, squareFromString("d4"))
	assert.Equal(t, E5, squareFromString("e5"))
	assert.Equal(t, F6, squareFromString("f6"))
	assert.Equal(t, G7, squareFromString("g7"))
	assert.Equal(t, H8, squareFromString("h8"))
}

func TestBoardMy(t *testing.T) {
	b := &Board{
		SideToMove: White,
	}
	assert.Equal(t, Piece(WP), b.my(Pawn))
	assert.Equal(t, Piece(WR), b.my(Rook))
	assert.Equal(t, Piece(WQ), b.my(Queen))
	assert.Equal(t, Piece(WK), b.my(King))
	assert.Equal(t, Piece(WN), b.my(Knight))
	assert.Equal(t, Piece(WB), b.my(Bishop))
}

func TestBoardOpp(t *testing.T) {
	b := &Board{
		SideToMove: Black,
	}
	assert.Equal(t, Piece(WP), b.opp(Pawn))
	assert.Equal(t, Piece(WR), b.opp(Rook))
	assert.Equal(t, Piece(WQ), b.opp(Queen))
	assert.Equal(t, Piece(WK), b.opp(King))
	assert.Equal(t, Piece(WN), b.opp(Knight))
	assert.Equal(t, Piece(WB), b.opp(Bishop))
}

func TestSquareValue(t *testing.T) {
	assert.Equal(t, 0, int(A1))
	assert.Equal(t, 8, int(A2))
	assert.Equal(t, 16, int(A3))
	assert.Equal(t, 24, int(A4))
	assert.Equal(t, 32, int(A5))
	assert.Equal(t, 40, int(A6))
	assert.Equal(t, 48, int(A7))
	assert.Equal(t, 56, int(A8))
	assert.Equal(t, 1, int(B1))
	assert.Equal(t, 9, int(B2))
	assert.Equal(t, 17, int(B3))
	assert.Equal(t, 25, int(B4))
	assert.Equal(t, 33, int(B5))
	assert.Equal(t, 41, int(B6))
	assert.Equal(t, 49, int(B7))
	assert.Equal(t, 57, int(B8))
}

func TestSquareColor(t *testing.T) {
	assert.Equal(t, Black, A1.Color())
	assert.Equal(t, White, B1.Color())
	assert.Equal(t, Black, C1.Color())
	assert.Equal(t, White, D1.Color())
	assert.Equal(t, Black, E1.Color())
	assert.Equal(t, White, F1.Color())
	assert.Equal(t, Black, G1.Color())
	assert.Equal(t, White, H1.Color())
	assert.Equal(t, White, A2.Color())
	assert.Equal(t, Black, B2.Color())
	assert.Equal(t, White, C2.Color())
	assert.Equal(t, Black, D2.Color())
	assert.Equal(t, White, E2.Color())
	assert.Equal(t, Black, F2.Color())
	assert.Equal(t, White, G2.Color())
	assert.Equal(t, Black, H2.Color())
}

func TestGetPieceTypes(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		color    int
		expected []GamePiece
	}{
		{
			name:  "Starting position - White pieces",
			fen:   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			color: White,
			expected: []GamePiece{{WR, A1}, {WN, B1}, {WB, C1}, {WQ, D1}, {WK, E1}, {WB, F1}, {WN, G1}, {WR, H1},
				{WP, A2}, {WP, B2}, {WP, C2}, {WP, D2}, {WP, E2}, {WP, F2}, {WP, G2}, {WP, H2}},
		},
		{
			name:  "Starting position - Black pieces",
			fen:   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			color: Black,
			expected: []GamePiece{{BP, A7}, {BP, B7}, {BP, C7}, {BP, D7}, {BP, E7}, {BP, F7}, {BP, G7}, {BP, H7},
				{BR, A8}, {BN, B8}, {BB, C8}, {BQ, D8}, {BK, E8}, {BB, F8}, {BN, G8}, {BR, H8},
			},
		},
		{
			name:     "Endgame with few pieces - White",
			fen:      "8/5k2/8/8/8/8/4K3/4Q3 w - - 0 1",
			color:    White,
			expected: []GamePiece{{WQ, E1}, {WK, E2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := ParseFen(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}

			got := board.GetPieceTypes(tt.color)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetPieceTypes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasInsufficientMaterial(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		expected bool
	}{
		{
			name:     "Starting position",
			fen:      "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			expected: false,
		},
		{
			name:     "King vs King",
			fen:      "4k3/8/8/8/8/8/8/4K3 w - - 0 1",
			expected: true,
		},
		{
			name:     "King vs King and Knight",
			fen:      "4k3/8/8/8/8/8/5N2/4K3 w - - 0 1",
			expected: true,
		},
		{
			name:     "King vs King and Bishop",
			fen:      "4k3/8/8/8/8/8/5B2/4K3 w - - 0 1",
			expected: true,
		},
		{
			name:     "King and Bishop vs King and Bishop (same color squares)",
			fen:      "4k3/6b1/8/8/8/8/5B2/4K3 w - - 0 1",
			expected: true,
		},
		{
			name:     "King and Bishop vs King and Bishop (different color squares)",
			fen:      "4k3/7b/8/8/8/8/5B2/4K3 w - - 0 1",
			expected: true,
		},
		{
			name:     "King and two Knights vs King",
			fen:      "4k3/8/8/8/8/5N2/5N2/4K3 w - - 0 1",
			expected: false, // Two knights can force mate
		},
		{
			name:     "King and Pawn vs King",
			fen:      "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
			expected: false, // Pawn can promote
		},
		{
			name:     "King and Rook vs King",
			fen:      "4k3/8/8/8/8/8/8/4KR2 w - - 0 1",
			expected: false, // Rook can deliver mate
		},
		{
			name:     "King and Queen vs King",
			fen:      "4k3/8/8/8/8/8/8/4KQ2 w - - 0 1",
			expected: false, // Queen can deliver mate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := ParseFen(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}

			got := board.HasInsufficientMaterial()
			if got != tt.expected {
				t.Errorf("HasInsufficientMaterial() = %v, want %v", got, tt.expected)
			}
		})
	}
}
