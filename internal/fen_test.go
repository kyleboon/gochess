package internal

import (
	"testing"
)

func TestParseFen(t *testing.T) {
	tests := []struct {
		name    string
		fen     string
		wantErr bool
		check   func(*Board) bool
	}{
		{
			name:    "Starting position",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			wantErr: false,
			check: func(b *Board) bool {
				// Check pieces
				if b.Piece[A1] != WR || b.Piece[B1] != WN || b.Piece[C1] != WB || b.Piece[D1] != WQ ||
					b.Piece[E1] != WK || b.Piece[F1] != WB || b.Piece[G1] != WN || b.Piece[H1] != WR {
					return false
				}
				for file := FileA; file <= FileH; file++ {
					if b.Piece[Square(file, Rank2)] != WP {
						return false
					}
				}
				for rank := Rank3; rank <= Rank6; rank++ {
					for file := FileA; file <= FileH; file++ {
						if b.Piece[Square(file, rank)] != NoPiece {
							return false
						}
					}
				}
				for file := FileA; file <= FileH; file++ {
					if b.Piece[Square(file, Rank7)] != BP {
						return false
					}
				}
				if b.Piece[A8] != BR || b.Piece[B8] != BN || b.Piece[C8] != BB || b.Piece[D8] != BQ ||
					b.Piece[E8] != BK || b.Piece[F8] != BB || b.Piece[G8] != BN || b.Piece[H8] != BR {
					return false
				}

				// Check other properties
				if b.SideToMove != White {
					return false
				}
				if b.CastleSq[WhiteOO] != H1 || b.CastleSq[WhiteOOO] != A1 ||
					b.CastleSq[BlackOO] != H8 || b.CastleSq[BlackOOO] != A8 {
					return false
				}
				if b.EpSquare != NoSquare {
					return false
				}
				if b.Rule50 != 0 {
					return false
				}
				if b.MoveNr != 1 {
					return false
				}

				return true
			},
		},
		{
			name:    "Middle game position",
			fen:     "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
			wantErr: false,
			check: func(b *Board) bool {
				// Check key pieces in this position
				if b.Piece[C4] != WB || b.Piece[F3] != WN || b.Piece[C6] != BN || b.Piece[F6] != BN {
					return false
				}
				if b.SideToMove != White {
					return false
				}
				if b.CastleSq[WhiteOO] != H1 || b.CastleSq[WhiteOOO] != A1 ||
					b.CastleSq[BlackOO] != H8 || b.CastleSq[BlackOOO] != A8 {
					return false
				}
				if b.EpSquare != NoSquare {
					return false
				}
				if b.Rule50 != 4 {
					return false
				}
				if b.MoveNr != 4 {
					return false
				}
				return true
			},
		},
		{
			name:    "Position with en passant square",
			fen:     "rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2",
			wantErr: false,
			check: func(b *Board) bool {
				if b.EpSquare != D6 {
					return false
				}
				if b.SideToMove != White || b.MoveNr != 2 || b.Rule50 != 0 {
					return false
				}
				return true
			},
		},
		{
			name:    "Position with no castling rights",
			fen:     "rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w - - 0 2",
			wantErr: false,
			check: func(b *Board) bool {
				if b.CastleSq[WhiteOO] != NoSquare || b.CastleSq[WhiteOOO] != NoSquare ||
					b.CastleSq[BlackOO] != NoSquare || b.CastleSq[BlackOOO] != NoSquare {
					return false
				}
				return true
			},
		},
		{
			name:    "Position with black to move",
			fen:     "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			wantErr: false,
			check: func(b *Board) bool {
				if b.SideToMove != Black || b.EpSquare != E3 {
					return false
				}
				return true
			},
		},
		{
			name:    "Invalid FEN - wrong number of parts",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq -",
			wantErr: true,
			check:   func(b *Board) bool { return true },
		},
		{
			name:    "Invalid FEN - wrong number of ranks",
			fen:     "rnbqkbnr/pppppppp/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			wantErr: true,
			check:   func(b *Board) bool { return true },
		},
		{
			name:    "Invalid FEN - invalid active color",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR x KQkq - 0 1",
			wantErr: true,
			check:   func(b *Board) bool { return true },
		},
		{
			name:    "Invalid FEN - invalid castling rights",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w XQkq - 0 1",
			wantErr: true,
			check:   func(b *Board) bool { return true },
		},
		{
			name:    "Invalid FEN - invalid en passant square",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq z3 0 1",
			wantErr: true,
			check:   func(b *Board) bool { return true },
		},
		{
			name:    "Invalid FEN - invalid halfmove clock",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - x 1",
			wantErr: true,
			check:   func(b *Board) bool { return true },
		},
		{
			name:    "Invalid FEN - invalid fullmove number",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 y",
			wantErr: true,
			check:   func(b *Board) bool { return true },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := ParseFen(tt.fen)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFen() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !tt.check(b) {
				t.Errorf("ParseFen() board state incorrect for FEN: %s", tt.fen)
			}
		})
	}
}

func TestBoardFen(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		wantFen  string // If omitted, expected to be identical to input fen
	}{
		{
			name:    "Starting position",
			fen:     "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name:    "Middle game position",
			fen:     "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
		},
		{
			name:    "Position with en passant",
			fen:     "rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2",
		},
		{
			name:    "Position with black to move",
			fen:     "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
		{
			name:    "Position with no castling rights",
			fen:     "rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w - - 0 2",
		},
		{
			name:    "Late game position",
			fen:     "4k3/8/8/8/8/8/4P3/4K3 w - - 5 39",
		},
		{
			name:    "Position with multiple empty squares",
			fen:     "8/3k4/8/8/3K4/8/8/8 b - - 10 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the FEN string to create a board
			board, err := ParseFen(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}
			
			// Convert board back to FEN string
			gotFen := board.Fen()
			
			// Determine expected FEN
			expectedFen := tt.fen
			if tt.wantFen != "" {
				expectedFen = tt.wantFen
			}
			
			// Compare result
			if gotFen != expectedFen {
				t.Errorf("Board.Fen() = %v, want %v", gotFen, expectedFen)
			}
		})
	}
}

func TestFenRoundTrip(t *testing.T) {
	fens := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
		"rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2",
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w - - 0 2",
		"4k3/8/8/8/8/8/4P3/4K3 w - - 5 39",
		"8/3k4/8/8/3K4/8/8/8 b - - 10 50",
	}

	for _, fen := range fens {
		t.Run(fen, func(t *testing.T) {
			// Parse FEN → Board → FEN
			board, err := ParseFen(fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}
			
			// Convert board back to FEN
			gotFen := board.Fen()
			
			// FENs should be identical after round trip
			if gotFen != fen {
				t.Errorf("FEN round trip failed:\nOriginal: %v\nGot:      %v", fen, gotFen)
			}
		})
	}
}
