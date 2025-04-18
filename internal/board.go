package internal

const (
	White = iota
	Black
)

const (
	NoPiece = iota << 1
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

const (
	WP = White | Pawn
	WN = White | Knight
	WB = White | Bishop
	WR = White | Rook
	WQ = White | Queen
	WK = White | King
	BP = Black | Pawn
	BN = Black | Knight
	BB = Black | Bishop
	BR = Black | Rook
	BQ = Black | Queen
	BK = Black | King
)

type Piece uint8

func (p Piece) Color() int { return int(p) & 0x01 }
func (p Piece) Type() int  { return int(p) &^ 0x01 }

var PieceRunes = []rune{
	'.', ',',
	'P', 'p',
	'N', 'n',
	'B', 'b',
	'R', 'r',
	'Q', 'q',
	'K', 'k',
}

var Glyphs = []rune{
	'.', ',',
	0x2659, 0x265F,
	0x2658, 0x265E,
	0x2657, 0x265D,
	0x2656, 0x265C,
	0x2655, 0x265B,
	0x2654, 0x265A,
}

func pieceFromChar(c rune) Piece {
	for i := WP; i < len(PieceRunes); i++ {
		if PieceRunes[i] == c {
			return Piece(i)
		}
	}
	return NoPiece
}

const (
	A1, B1, C1, D1, E1, F1, G1, H1 Sq = 8*iota + 0, 8*iota + 1, 8*iota + 2,
		8*iota + 3, 8*iota + 4, 8*iota + 5, 8*iota + 6, 8*iota + 7
	A2, B2, C2, D2, E2, F2, G2, H2
	A3, B3, C3, D3, E3, F3, G3, H3
	A4, B4, C4, D4, E4, F4, G4, H4
	A5, B5, C5, D5, E5, F5, G5, H5
	A6, B6, C6, D6, E6, F6, G6, H6
	A7, B7, C7, D7, E7, F7, G7, H7
	A8, B8, C8, D8, E8, F8, G8, H8
	NoSquare Sq = -1
)

var squareNames = []string{
	"a1", "b1", "c1", "d1", "e1", "f1", "g1", "h1",
	"a2", "b2", "c2", "d2", "e2", "f2", "g2", "h2",
	"a3", "b3", "c3", "d3", "e3", "f3", "g3", "h3",
	"a4", "b4", "c4", "d4", "e4", "f4", "g4", "h4",
	"a5", "b5", "c5", "d5", "e5", "f5", "g5", "h5",
	"a6", "b6", "c6", "d6", "e6", "f6", "g6", "h6",
	"a7", "b7", "c7", "d7", "e7", "f7", "g7", "h7",
	"a8", "b8", "c8", "d8", "e8", "f8", "g8", "h8",
}

const (
	FileA = iota
	FileB
	FileC
	FileD
	FileE
	FileF
	FileG
	FileH
)

const (
	Rank1 = iota
	Rank2
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
)

type Sq int8

func Square(file, rank int) Sq {
	if file < FileA || file > FileH || rank < Rank1 || rank > Rank8 {
		return NoSquare
	}

	return Sq(rank*8 + file)
}

// File returns the square's file (0-7).
func (sq Sq) File() int {
	return int(sq) % 8
}

// Rank returns the square's rank (0-7).
func (sq Sq) Rank() int { return int(sq) / 8 }

// RelativeRank returns the square's rank relative to the given player (0-7).
func (sq Sq) RelativeRank(color int) int {
	if color == White {
		return sq.Rank()
	}
	return 7 - sq.Rank()
}

// String returns the algebraic notation of the square (a1, e5, etc.).
func (sq Sq) String() string {
	if sq == NoSquare {
		return "-"
	}
	return squareNames[sq]
}

func squareFromString(s string) Sq {
	if len(s) != 2 || s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return NoSquare
	}
	return Square(int(s[0])-'a', int(s[1])-'1')
}

const (
	queenSide = iota << 1
	kingSide
	WhiteOO  = White | kingSide
	BlackOO  = Black | kingSide
	WhiteOOO = White | queenSide
	BlackOOO = Black | queenSide
)

type Board struct {
	Piece      [64]Piece // piece placement (NoPiece, WP, BP, WN, BN, ...)
	SideToMove int       // White or Black
	MoveNr     int       // fullmove counter (1-based)
	Rule50     int       // halfmove counter for the 50-move rule (counts from 0-100)
	EpSquare   Sq        // en-passant square (behind capturable pawn)
	CastleSq   [4]Sq     // rooks that can castle; e.g. CastleSq[WhiteOO]
	checkFrom  Sq        // squares the opponent's castling king moved through;
	checkTo    Sq        //      [A1,A1] if opp did not castle last turn.
}

type GamePiece struct {
	piece Piece
	sq    Sq
}

func (sq Sq) Color() int { return (sq.File() + sq.Rank() + 1) % 2 }

func (b *Board) my(piece int) Piece  { return Piece(b.SideToMove | piece) }
func (b *Board) opp(piece int) Piece { return Piece(b.SideToMove ^ 1 | piece) }

// MakeMove returns a copy of the Board with move m applied.
func (b Board) MakeMove(m Move) *Board {
	epSquare := b.EpSquare // remember en passant square

	// these are reset by making a move
	b.EpSquare = NoSquare
	b.checkFrom, b.checkTo = A1, A1

	switch {
	case m == NullMove:
		// do nothing
	case b.Piece[m.From] == b.my(King) && b.Piece[m.To] == b.my(Rook): // castling
		wing := kingSide
		if m.To < m.From {
			wing = queenSide
		}
		rf, kf, rt, kt, _, _ := b.castleSquares(wing)
		b.Piece[rf] = NoPiece
		b.Piece[kf] = NoPiece
		b.Piece[rt] = b.my(Rook)
		b.Piece[kt] = b.my(King)
		if kf < kt {
			b.checkFrom, b.checkTo = kf, kt
		} else {
			b.checkFrom, b.checkTo = kt, kf
		}
		b.CastleSq[b.SideToMove|kingSide] = NoSquare
		b.CastleSq[b.SideToMove|queenSide] = NoSquare
		b.Rule50++
	default:
		piece := b.Piece[m.From]
		if piece.Type() == Pawn {
			switch dy := m.To.Rank() - m.From.Rank(); {
			case dy == 2 || dy == -2:
				b.EpSquare = Square(m.From.File(), m.From.Rank()+dy/2)
			case m.To == epSquare:
				// move the captured pawn to the ep-square, so
				// that Rule50 is updated correctly below
				b.Piece[Square(m.To.File(), m.From.Rank())] = NoPiece
				b.Piece[epSquare] = b.opp(Pawn)
			case m.To.RelativeRank(b.SideToMove) == Rank8:
				b.Piece[m.From] = m.Promotion
			}
		}
		// update castling rights
		for i, sq := range b.CastleSq {
			if sq == m.From || sq == m.To {
				b.CastleSq[i] = NoSquare
			}
		}
		if piece.Type() == King {
			b.CastleSq[b.SideToMove|kingSide] = NoSquare
			b.CastleSq[b.SideToMove|queenSide] = NoSquare
		}
		// update the 50-move rule counter
		if piece.Type() == Pawn || b.Piece[m.To] != NoPiece {
			b.Rule50 = 0
		} else {
			b.Rule50++
		}
		// move the piece
		b.Piece[m.To] = b.Piece[m.From]
		b.Piece[m.From] = NoPiece
	}
	// switch side to move
	if b.SideToMove ^= 1; b.SideToMove == White {
		b.MoveNr++
	}
	return &b
}

// find locates a piece in the given range of squares.
func (b *Board) find(piece Piece, sq0, sq1 Sq) Sq {
	dir := Sq(1)
	if sq0 > sq1 {
		dir = -1
	}
	for sq := sq0; sq != sq1+dir; sq += dir {
		if b.Piece[sq] == piece {
			return sq
		}
	}
	return NoSquare
}

// GetPieceTypes returns a map of piece types to their counts for the given color.
// For example, if White has 2 bishops and 1 queen, the result would be:
// map[Bishop:2 Queen:1]
// Kings are excluded from the results as they are always present.
func (b *Board) GetPieceTypes(color int) []GamePiece {
	pieces := []GamePiece{}

	for sq := A1; sq <= H8; sq++ {
		piece := b.Piece[sq]
		if piece == NoPiece || piece.Color() != color {
			continue
		}

		pieces = append(pieces, GamePiece{piece, sq})
	}

	return pieces
}

func (b *Board) HasInsufficientMaterial() bool {
	whitePieces := b.GetPieceTypes(White)
	blackPieces := b.GetPieceTypes(Black)

	if len(whitePieces) > 3 || len(blackPieces) > 3 {
		return false
	}

	// if white has a rook, a pawn or a queen
	whiteHasLightColoredBishop := false
	whiteHasDarkColoredBishop := false
	whiteKnightCount := 0
	for _, piece := range whitePieces {
		if piece.piece == WR || piece.piece == WP || piece.piece == WQ {
			return false
		}
		if piece.piece == WB {
			if piece.sq.Color() == 0 {
				whiteHasLightColoredBishop = true
			} else {
				whiteHasDarkColoredBishop = true
			}
		}
		if piece.piece == WN {
			whiteKnightCount++
		}
	}

	if whiteHasLightColoredBishop && whiteHasDarkColoredBishop {
		return false
	}

	if whiteKnightCount > 1 {
		return false
	}

	if (whiteHasLightColoredBishop || whiteHasDarkColoredBishop) && whiteKnightCount >= 1 {
		return false
	}

	// if black has a rook, a pawn or a queen
	blackHasLightColoredBishop := false
	blackHasDarkColoredBishop := false
	blackKnightCount := 0
	for _, piece := range blackPieces {
		if piece.piece == BR || piece.piece == BP || piece.piece == BQ {
			return false
		}
		if piece.piece == WB {
			if piece.sq.Color() == 0 {
				blackHasLightColoredBishop = true
			} else {
				blackHasDarkColoredBishop = true
			}
		}
		if piece.piece == WN {
			blackKnightCount++
		}
	}

	if blackHasLightColoredBishop && blackHasDarkColoredBishop {
		return false
	}

	if blackKnightCount > 1 {
		return false
	}

	if (blackHasLightColoredBishop || blackHasDarkColoredBishop) && blackKnightCount >= 1 {
		return false
	}

	return true
}
