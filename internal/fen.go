package internal

import (
	"errors"
	"strconv"
	"strings"
)

// ParseFen parses a FEN string and returns a Board
func ParseFen(fen string) (*Board, error) {
	parts := strings.Fields(fen)
	if len(parts) != 6 {
		return nil, errors.New("invalid FEN: expected 6 space-separated fields")
	}

	board := &Board{}

	if err := parsePiecePlacement(board, parts[0]); err != nil {
		return nil, err
	}

	if err := parseActiveColor(board, parts[1]); err != nil {
		return nil, err
	}

	if err := parseCastling(board, parts[2]); err != nil {
		return nil, err
	}

	if err := parseEnPassant(board, parts[3]); err != nil {
		return nil, err
	}

	halfmove, err := strconv.Atoi(parts[4])
	if err != nil {
		return nil, errors.New("invalid halfmove clock in FEN")
	}
	board.Rule50 = halfmove

	fullmove, err := strconv.Atoi(parts[5])
	if err != nil {
		return nil, errors.New("invalid fullmove number in FEN")
	}
	board.MoveNr = fullmove

	return board, nil
}

func parsePiecePlacement(board *Board, placement string) error {
	ranks := strings.Split(placement, "/")
	if len(ranks) != 8 {
		return errors.New("invalid piece placement: expected 8 ranks")
	}

	for i := range board.Piece {
		board.Piece[i] = NoPiece
	}

	for rank := 7; rank >= 0; rank-- {
		rankStr := ranks[7-rank]
		file := 0

		for _, char := range rankStr {
			if file >= 8 {
				return errors.New("invalid piece placement: too many pieces in rank")
			}

			if char >= '1' && char <= '8' {
				// Skip empty squares
				file += int(char - '0')
			} else {
				// Place piece
				piece := pieceFromChar(char)
				if piece == NoPiece {
					return errors.New("invalid piece character in FEN")
				}

				square := Square(file, rank)
				board.Piece[square] = piece
				file++
			}
		}

		if file != 8 {
			return errors.New("invalid piece placement: rank doesn't have 8 squares")
		}
	}

	return nil
}

func parseActiveColor(board *Board, color string) error {
	if color == "w" {
		board.SideToMove = White
	} else if color == "b" {
		board.SideToMove = Black
	} else {
		return errors.New("invalid active color in FEN: expected 'w' or 'b'")
	}
	return nil
}

func parseCastling(board *Board, castling string) error {
	// Initialize castling rights
	for i := range board.CastleSq {
		board.CastleSq[i] = NoSquare
	}

	if castling == "-" {
		return nil // No castling rights
	}

	for _, char := range castling {
		switch char {
		case 'K':
			board.CastleSq[WhiteOO] = H1
		case 'Q':
			board.CastleSq[WhiteOOO] = A1
		case 'k':
			board.CastleSq[BlackOO] = H8
		case 'q':
			board.CastleSq[BlackOOO] = A8
		default:
			return errors.New("invalid castling availability in FEN")
		}
	}

	return nil
}

func parseEnPassant(board *Board, enPassant string) error {
	if enPassant == "-" {
		board.EpSquare = NoSquare
		return nil
	}

	square := squareFromString(enPassant)
	if square == NoSquare {
		return errors.New("invalid en passant target square in FEN")
	}

	board.EpSquare = square
	return nil
}

func (b *Board) Fen() string {
	var sb strings.Builder

	// 1. Piece placement
	for rank := 7; rank >= 0; rank-- {
		emptyCount := 0

		for file := 0; file < 8; file++ {
			sq := Square(file, rank)
			piece := b.Piece[sq]

			if piece == NoPiece {
				emptyCount++
			} else {
				if emptyCount > 0 {
					sb.WriteString(strconv.Itoa(emptyCount))
					emptyCount = 0
				}

				sb.WriteRune(PieceRunes[piece])
			}
		}

		if emptyCount > 0 {
			sb.WriteString(strconv.Itoa(emptyCount))
		}

		if rank > 0 {
			sb.WriteRune('/')
		}
	}

	// 2. Active color
	sb.WriteRune(' ')
	if b.SideToMove == White {
		sb.WriteRune('w')
	} else {
		sb.WriteRune('b')
	}

	// 3. Castling availability
	sb.WriteRune(' ')

	castlingCount := 0
	if b.CastleSq[WhiteOO] == H1 {
		sb.WriteRune('K')
		castlingCount++
	}
	if b.CastleSq[WhiteOOO] == A1 {
		sb.WriteRune('Q')
		castlingCount++
	}
	if b.CastleSq[BlackOO] == H8 {
		sb.WriteRune('k')
		castlingCount++
	}
	if b.CastleSq[BlackOOO] == A8 {
		sb.WriteRune('q')
		castlingCount++
	}

	if castlingCount == 0 {
		sb.WriteRune('-')
	}

	// 4. En passant target square
	sb.WriteRune(' ')
	if b.EpSquare == NoSquare {
		sb.WriteRune('-')
	} else {
		sb.WriteString(b.EpSquare.String())
	}

	// 5. Halfmove clock
	sb.WriteRune(' ')
	sb.WriteString(strconv.Itoa(b.Rule50))

	// 6. Fullmove number
	sb.WriteRune(' ')
	sb.WriteString(strconv.Itoa(b.MoveNr))

	return sb.String()
}
