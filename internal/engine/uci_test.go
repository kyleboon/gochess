package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/kyleboon/gochess/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInfoLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *AnalysisLine
		wantErr bool
	}{
		{
			name: "centipawn score with pv",
			line: "info depth 20 multipv 1 score cp 35 nodes 1234567 nps 2000000 pv d2d3 d7d6 c1e3",
			want: &AnalysisLine{
				Rank:  1,
				Depth: 20,
				Score: Score{Centipawns: 35},
				Nodes: 1234567,
				NPS:   2000000,
				Moves: []string{"d2d3", "d7d6", "c1e3"},
			},
		},
		{
			name: "negative centipawn score",
			line: "info depth 15 multipv 2 score cp -42 nodes 500000 nps 1500000 pv e7e5 d2d4",
			want: &AnalysisLine{
				Rank:  2,
				Depth: 15,
				Score: Score{Centipawns: -42},
				Nodes: 500000,
				NPS:   1500000,
				Moves: []string{"e7e5", "d2d4"},
			},
		},
		{
			name: "mate score positive",
			line: "info depth 18 multipv 1 score mate 3 nodes 100000 nps 1000000 pv e1g1 d8h4 g2g3",
			want: &AnalysisLine{
				Rank:  1,
				Depth: 18,
				Score: Score{Mate: 3, IsMate: true},
				Nodes: 100000,
				NPS:   1000000,
				Moves: []string{"e1g1", "d8h4", "g2g3"},
			},
		},
		{
			name: "mate score negative",
			line: "info depth 12 multipv 1 score mate -2 nodes 50000 nps 800000 pv a1a2 b3b1",
			want: &AnalysisLine{
				Rank:  1,
				Depth: 12,
				Score: Score{Mate: -2, IsMate: true},
				Nodes: 50000,
				NPS:   800000,
				Moves: []string{"a1a2", "b3b1"},
			},
		},
		{
			name: "missing multipv defaults to rank 1",
			line: "info depth 10 score cp 15 pv e2e4",
			want: &AnalysisLine{
				Rank:  1,
				Depth: 10,
				Score: Score{Centipawns: 15},
				Moves: []string{"e2e4"},
			},
		},
		{
			name: "non-info line returns nil",
			line: "bestmove e2e4 ponder e7e5",
			want: nil,
		},
		{
			name: "info string line (no depth) returns nil",
			line: "info string NNUE evaluation using nn-...",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInfoLine(tt.line)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want.Rank, got.Rank)
			assert.Equal(t, tt.want.Depth, got.Depth)
			assert.Equal(t, tt.want.Score, got.Score)
			assert.Equal(t, tt.want.Nodes, got.Nodes)
			assert.Equal(t, tt.want.NPS, got.NPS)
			assert.Equal(t, tt.want.Moves, got.Moves)
		})
	}
}

func TestScoreString(t *testing.T) {
	tests := []struct {
		name  string
		score Score
		want  string
	}{
		{
			name:  "positive centipawns",
			score: Score{Centipawns: 35},
			want:  "+0.35",
		},
		{
			name:  "negative centipawns",
			score: Score{Centipawns: -150},
			want:  "-1.50",
		},
		{
			name:  "zero centipawns",
			score: Score{Centipawns: 0},
			want:  "+0.00",
		},
		{
			name:  "large positive centipawns",
			score: Score{Centipawns: 523},
			want:  "+5.23",
		},
		{
			name:  "mate in 3",
			score: Score{Mate: 3, IsMate: true},
			want:  "#3",
		},
		{
			name:  "mated in 2",
			score: Score{Mate: -2, IsMate: true},
			want:  "#-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.score.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

// mockEngine simulates UCI responses using io.Pipe for testing.
func mockEngine(t *testing.T, responses []string) (*Engine, func()) {
	t.Helper()

	// engineStdinR is what we read from the engine's perspective (commands from us)
	// engineStdinW is what we write commands to
	engineStdinR, engineStdinW := io.Pipe()

	// engineStdoutR is what we read engine output from
	// engineStdoutW is what the engine writes responses to
	engineStdoutR, engineStdoutW := io.Pipe()

	logger := logging.Discard()
	e := NewFromStreams(engineStdinW, engineStdoutR, logger)

	// Goroutine: read commands and write responses
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = engineStdoutW.Close() }()

		scanner := strings.Join(responses, "\n")
		_, _ = io.WriteString(engineStdoutW, scanner+"\n")
	}()

	cleanup := func() {
		_ = engineStdinR.Close()
		_ = engineStdinW.Close()
		<-done
	}

	// Consume stdin in background to prevent blocking
	go func() {
		_, _ = io.Copy(io.Discard, engineStdinR)
	}()

	return e, cleanup
}

func TestMockEngine_IsReady(t *testing.T) {
	e, cleanup := mockEngine(t, []string{"readyok"})
	defer cleanup()

	err := e.IsReady(context.Background())
	require.NoError(t, err)
}

func TestMockEngine_Analyze(t *testing.T) {
	responses := []string{
		"info depth 1 multipv 1 score cp 10 nodes 100 nps 10000 pv e2e4",
		"info depth 2 multipv 1 score cp 15 nodes 500 nps 25000 pv e2e4 e7e5",
		"info depth 2 multipv 2 score cp 8 nodes 500 nps 25000 pv d2d4 d7d5",
		"info depth 3 multipv 1 score cp 20 nodes 2000 nps 50000 pv e2e4 e7e5 g1f3",
		"info depth 3 multipv 2 score cp 12 nodes 2000 nps 50000 pv d2d4 d7d5 c2c4",
		"bestmove e2e4 ponder e7e5",
	}
	e, cleanup := mockEngine(t, responses)
	defer cleanup()

	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	result, err := e.Analyze(context.Background(), fen, AnalysisOptions{
		Depth:   3,
		MultiPV: 2,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, fen, result.FEN)
	assert.Equal(t, 3, result.Depth)
	require.Len(t, result.Lines, 2)

	// Line 1 should be the deepest for multipv 1
	assert.Equal(t, 1, result.Lines[0].Rank)
	assert.Equal(t, 3, result.Lines[0].Depth)
	assert.Equal(t, Score{Centipawns: 20}, result.Lines[0].Score)
	assert.Equal(t, []string{"e2e4", "e7e5", "g1f3"}, result.Lines[0].Moves)

	// Line 2 should be the deepest for multipv 2
	assert.Equal(t, 2, result.Lines[1].Rank)
	assert.Equal(t, 3, result.Lines[1].Depth)
	assert.Equal(t, Score{Centipawns: 12}, result.Lines[1].Score)
	assert.Equal(t, []string{"d2d4", "d7d5", "c2c4"}, result.Lines[1].Moves)
}

func TestMockEngine_AnalyzeWithMate(t *testing.T) {
	responses := []string{
		"info depth 5 multipv 1 score mate 2 nodes 5000 nps 100000 pv d1h5 g7g6 h5f7",
		"bestmove d1h5",
	}
	e, cleanup := mockEngine(t, responses)
	defer cleanup()

	// Black to move: engine reports mate 2 from Black's perspective,
	// but we normalize to White's perspective, so it becomes mate -2 (White is getting mated).
	fen := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	result, err := e.Analyze(context.Background(), fen, AnalysisOptions{Depth: 5})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Lines, 1)

	assert.True(t, result.Lines[0].Score.IsMate)
	assert.Equal(t, -2, result.Lines[0].Score.Mate)
	assert.Equal(t, "#-2", result.Lines[0].Score.String())
}

func TestMockEngine_AnalyzeContextCanceled(t *testing.T) {
	// Engine that never responds with bestmove
	engineStdinR, engineStdinW := io.Pipe()
	engineStdoutR, engineStdoutW := io.Pipe()

	logger := logging.Discard()
	e := NewFromStreams(engineStdinW, engineStdoutR, logger)

	// Write some info lines but never bestmove
	go func() {
		_, _ = io.WriteString(engineStdoutW, "info depth 1 score cp 10 pv e2e4\n")
		// Don't write bestmove — let context cancel
	}()

	// Consume stdin
	go func() {
		_, _ = io.Copy(io.Discard, engineStdinR)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := e.Analyze(ctx, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", AnalysisOptions{Depth: 10})
	assert.Error(t, err)

	_ = engineStdinR.Close()
	_ = engineStdinW.Close()
	_ = engineStdoutW.Close()
}
