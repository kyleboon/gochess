package internal

import (
	"testing"
)

// BenchmarkMoveGeneration measures how many legal moves can be generated per second
func BenchmarkMoveGeneration(b *testing.B) {
	// Load a set of diverse positions to get a better average
	positions := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",                   // Starting position
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",       // Position 2 (complex middlegame)
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",                                  // Position 3 (endgame)
		"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",           // Position 4 (complex with many captures)
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",                  // Position 5 (middlegame tactics)
		"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10",   // Position 6 (symmetric)
	}

	var totalMoves int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cycle through positions
		pos := positions[i%len(positions)]
		
		// Parse FEN and generate legal moves
		board, err := ParseFen(pos)
		if err != nil {
			b.Fatalf("Failed to parse FEN %s: %v", pos, err)
		}
		
		moves := board.LegalMoves()
		totalMoves += len(moves)
	}
	
	// Report moves per second in the benchmark output
	b.ReportMetric(float64(totalMoves)/b.Elapsed().Seconds(), "moves/s")
}

// BenchmarkComplexPosition benchmarks move generation on a particularly complex position
func BenchmarkComplexPosition(b *testing.B) {
	// This position has many queens and knights, which have complex move patterns
	complex := "R6R/3Q4/1Q4Q1/4Q3/2Q4Q/Q4Q2/pp1Q4/kBNN1KB1 w - - 0 1"
	
	board, err := ParseFen(complex)
	if err != nil {
		b.Fatalf("Failed to parse FEN: %v", err)
	}
	
	var totalMoves int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		moves := board.LegalMoves()
		totalMoves += len(moves)
	}
	
	// Position-specific metrics
	b.ReportMetric(float64(totalMoves)/b.Elapsed().Seconds(), "moves/s")
	b.ReportMetric(float64(totalMoves)/float64(b.N), "moves/position")
}

// BenchmarkPerft1 benchmarks perft(1) across positions (1-ply move generation)
func BenchmarkPerft1(b *testing.B) {
	positions := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",                   // Starting position
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",       // Position 2
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",                                  // Position 3
		"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",           // Position 4
	}
	
	var totalNodes int
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := positions[i%len(positions)]
		board, _ := ParseFen(pos)
		nodes := perft(board, 1)
		totalNodes += nodes
	}
	
	b.ReportMetric(float64(totalNodes)/b.Elapsed().Seconds(), "nodes/s")
}

// Perft (performance test) - counts the number of leaf nodes at a given depth
func perft(board *Board, depth int) int {
	if depth == 0 {
		return 1
	}
	
	moves := board.LegalMoves()
	if depth == 1 {
		return len(moves)
	}
	
	var nodes int
	for _, move := range moves {
		newBoard := board.MakeMove(move)
		nodes += perft(newBoard, depth-1)
	}
	
	return nodes
}
