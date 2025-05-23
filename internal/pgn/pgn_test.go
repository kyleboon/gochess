package pgn

import (
	"strings"
	"testing"
)

func TestParsePgn(t *testing.T) {
	// Add FEN tag with standard starting position
	pgnText := `[Event "Live Chess"]
[Site "Chess.com"]
[Date "2025.04.07"]
[Round "-"]
[White "kyle_b81"]
[Black "danpin"]
[Result "0-1"]
[CurrentPosition "6r1/4k3/8/2N2p2/3PpP1P/2P1R3/5K2/7r w - -"]
[Timezone "UTC"]
[ECO "B13"]
[ECOUrl "https://www.chess.com/openings/Caro-Kann-Defense-Exchange-Variation-3...cxd5-4.Bd3-Nf6"]
[UTCDate "2025.04.07"]
[UTCTime "14:10:29"]
[WhiteElo "1472"]
[BlackElo "1466"]
[TimeControl "600"]
[Termination "danpin won on time"]
[StartTime "14:10:29"]
[EndDate "2025.04.07"]
[EndTime "14:30:45"]
[Link "https://www.chess.com/game/live/137123783766"]
[FEN "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"]

1. e4 {[%clk 0:09:57]} 1... c6 {[%clk 0:09:59.9]} 2. d4 {[%clk 0:09:54.6]} 2... d5 {[%clk 0:09:56.2]} 3. exd5 {[%clk 0:09:53.9]} 3... cxd5 {[%clk 0:09:54.6]} 4. Bd3 {[%clk 0:09:52.6]} 4... Nf6 {[%clk 0:09:52.3]} 5. Bf4 {[%clk 0:09:48.5]} 5... Nc6 {[%clk 0:09:47.5]} 6. c3 {[%clk 0:09:46.6]} 6... a6 {[%clk 0:09:39.1]} 7. Nf3 {[%clk 0:09:26.4]} 7... e6 {[%clk 0:09:32.3]} 8. O-O {[%clk 0:09:24]} 8... Be7 {[%clk 0:09:25.1]} 9. Qc2 {[%clk 0:09:20.5]} 9... Bd6 {[%clk 0:09:14.5]} 10. Bxd6 {[%clk 0:09:17.9]} 10... Qxd6 {[%clk 0:09:13]} 11. Re1 {[%clk 0:09:16.1]} 11... Ne7 {[%clk 0:09:09.8]} 12. Nbd2 {[%clk 0:09:08.4]} 12... h6 {[%clk 0:09:01.2]} 13. Ne5 {[%clk 0:08:20.6]} 13... b5 {[%clk 0:08:23.9]} 14. a3 {[%clk 0:07:43.4]} 14... Bb7 {[%clk 0:07:51.2]} 15. Re3 {[%clk 0:07:40.4]} 15... O-O {[%clk 0:07:17.8]} 16. Rg3 {[%clk 0:07:21.9]} 16... Nf5 {[%clk 0:07:06.8]} 17. Bxf5 {[%clk 0:07:12.6]} 17... exf5 {[%clk 0:06:59]} 18. Qxf5 {[%clk 0:07:04.3]} 18... Bc8 {[%clk 0:06:48.4]} 19. Qf4 {[%clk 0:06:54.3]} 19... Qd8 {[%clk 0:05:54]} 20. Qxh6 {[%clk 0:06:39]} 20... Ng4 {[%clk 0:05:15.3]} 21. Qf4 {[%clk 0:06:29.2]} 21... Qf6 {[%clk 0:03:55.2]} 22. Qxf6 {[%clk 0:06:20.3]} 22... Nxf6 {[%clk 0:03:52.8]} 23. f3 {[%clk 0:05:27.4]} 23... Bf5 {[%clk 0:03:50.6]} 24. Re1 {[%clk 0:05:19.1]} 24... Rae8 {[%clk 0:03:35.4]} 25. Nb3 {[%clk 0:04:57.2]} 25... Nh5 {[%clk 0:03:10.7]} 26. Rg5 {[%clk 0:04:51.4]} 26... g6 {[%clk 0:03:08.3]} 27. Kf1 {[%clk 0:02:47.5]} 27... Nf4 {[%clk 0:02:39.1]} 28. g3 {[%clk 0:01:45.6]} 28... Nh3 {[%clk 0:02:26.9]} 29. Rxf5 {[%clk 0:01:26]} 29... gxf5 {[%clk 0:02:22.1]} 30. Nc5 {[%clk 0:01:17.3]} 30... Kg7 {[%clk 0:02:04.8]} 31. Ncd7 {[%clk 0:01:11.4]} 31... Rh8 {[%clk 0:01:54.2]} 32. Kg2 {[%clk 0:01:03.5]} 32... Ng5 {[%clk 0:01:52.6]} 33. f4 {[%clk 0:01:00]} 33... Ne4 {[%clk 0:01:50]} 34. Re2 {[%clk 0:00:49.2]} 34... f6 {[%clk 0:01:40.9]} 35. Nf3 {[%clk 0:00:42.8]} 35... Re7 {[%clk 0:01:31.6]} 36. Nb6 {[%clk 0:00:38]} 36... Rd8 {[%clk 0:01:19.7]} 37. h4 {[%clk 0:00:36.1]} 37... Kg6 {[%clk 0:01:12.2]} 38. Nd2 {[%clk 0:00:33.4]} 38... a5 {[%clk 0:00:57.2]} 39. Nxe4 {[%clk 0:00:29.2]} 39... dxe4 {[%clk 0:00:55.3]} 40. Re3 {[%clk 0:00:27.3]} 40... b4 {[%clk 0:00:39]} 41. axb4 {[%clk 0:00:25.7]} 41... axb4 {[%clk 0:00:37.9]} 42. Nc4 {[%clk 0:00:17.9]} 42... bxc3 {[%clk 0:00:35.8]} 43. bxc3 {[%clk 0:00:16.5]} 43... Ra7 {[%clk 0:00:32.7]} 44. Kh3 {[%clk 0:00:12.8]} 44... Rh8 {[%clk 0:00:24.1]} 45. g4 {[%clk 0:00:10.9]} 45... fxg4+ {[%clk 0:00:22.7]} 46. Kxg4 {[%clk 0:00:10.1]} 46... f5+ {[%clk 0:00:20.6]} 47. Kg3 {[%clk 0:00:08.8]} 47... Ra1 {[%clk 0:00:18.3]} 48. Ne5+ {[%clk 0:00:06.4]} 48... Kf6 {[%clk 0:00:16.7]} 49. Nd7+ {[%clk 0:00:05.3]} 49... Ke7 {[%clk 0:00:15.2]} 50. Nc5 {[%clk 0:00:04.5]} 50... Rg8+ {[%clk 0:00:11.6]} 51. Kf2 {[%clk 0:00:02.9]} 51... Rh1 {[%clk 0:00:09.8]} 0-1`

	t.Run("Parse PGN Tags", func(t *testing.T) {
		db := &DB{}
		errors := db.Parse(pgnText)

		if len(errors) > 0 {
			t.Fatalf("Expected no parse errors, got %v", errors)
		}

		if len(db.Games) != 1 {
			t.Fatalf("Expected 1 game, got %d", len(db.Games))
		}

		game := db.Games[0]

		// Verify essential tags
		expectedTags := map[string]string{
			"Event":       "Live Chess",
			"Site":        "Chess.com",
			"Date":        "2025.04.07",
			"White":       "kyle_b81",
			"Black":       "danpin",
			"Result":      "0-1",
			"WhiteElo":    "1472",
			"BlackElo":    "1466",
			"TimeControl": "600",
			"FEN":         "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		}

		for tag, expectedValue := range expectedTags {
			if value, ok := game.Tags[tag]; !ok || value != expectedValue {
				t.Errorf("Expected tag %s to be %s, got %s", tag, expectedValue, value)
			}
		}
	})

	t.Run("Parse Moves", func(t *testing.T) {
		db := &DB{}
		errors := db.Parse(pgnText)
		if len(errors) > 0 {
			t.Fatalf("Expected no parse errors, got %v", errors)
		}

		game := db.Games[0]

		// Verify plies before parsing moves
		if game.plies != 102 {
			t.Errorf("Expected 102 plies, got %d", game.plies)
		}

		// Parse the moves
		err := db.ParseMoves(game)
		if err != nil {
			t.Fatalf("Error parsing moves: %v", err)
		}

		// Verify root is not nil
		if game.Root == nil {
			t.Fatal("Game root is nil")
		}

		// Verify first move (1. e4)
		firstNode := game.Root.Next
		if firstNode == nil {
			t.Fatal("First move is nil")
		}

		move := firstNode.Move
		// Get move details using the Uci method
		uci := move.Uci(firstNode.Parent.Board)
		if !strings.HasPrefix(uci, "e2e4") {
			t.Errorf("Expected first move to be e2-e4, got %s", uci)
		}

		// Check that the game tree has the correct number of nodes
		node := game.Root
		nodeCount := 0
		for node.Next != nil {
			node = node.Next
			nodeCount++
		}

		// The last move should be 51...Rh1
		if nodeCount != 102 {
			t.Errorf("Expected 102 nodes, got %d", nodeCount)
		}

		// Verify the last move
		lastNode := node
		lastMove := lastNode.Move
		// Use Uci method to check the move
		lastUci := lastMove.Uci(lastNode.Parent.Board)
		if !strings.HasPrefix(lastUci, "a1h1") {
			t.Errorf("Expected last move to be a1-h1, got %s", lastUci)
		}
	})

	t.Run("Test Game Result", func(t *testing.T) {
		db := &DB{}
		db.Parse(pgnText)

		if len(db.Games) != 1 {
			t.Fatalf("Expected 1 game, got %d", len(db.Games))
		}

		game := db.Games[0]
		if game.Tags["Result"] != "0-1" {
			t.Errorf("Expected result 0-1, got %s", game.Tags["Result"])
		}

		// Test termination tag
		if game.Tags["Termination"] != "danpin won on time" {
			t.Errorf("Expected termination 'danpin won on time', got %s", game.Tags["Termination"])
		}
	})
}

// Test that chess.com clock notation is properly handled
func TestChesscomClockNotation(t *testing.T) {
	// Short PGN excerpt with chess.com clock notation
	pgnText := `[Event "Live Chess"]
[Site "Chess.com"]
[Result "0-1"]
[FEN "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"]

1. e4 {[%clk 0:09:57]} 1... c6 {[%clk 0:09:59.9]} 2. d4 {[%clk 0:09:54.6]} 0-1`

	db := &DB{}
	errors := db.Parse(pgnText)

	if len(errors) > 0 {
		t.Fatalf("Expected no parse errors with chess.com clock notation, got %v", errors)
	}

	if len(db.Games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(db.Games))
	}

	game := db.Games[0]
	err := db.ParseMoves(game)
	if err != nil {
		t.Fatalf("Error parsing moves with clock notation: %v", err)
	}

	// Verify that we can read 3 moves
	node := game.Root
	moveCount := 0
	for node.Next != nil {
		node = node.Next
		moveCount++
	}

	if moveCount != 3 {
		t.Errorf("Expected 3 moves with clock notation, got %d", moveCount)
	}
}
