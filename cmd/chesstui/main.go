package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyleboon/gochess/internal"
)

type appState int

const (
	stateMainGame appState = iota
	stateChesscomDownloader
)

type model struct {
	state         appState
	board         *internal.Board
	moveList      []string
	gameOver      bool
	checkmate     bool
	stalemate     bool
	lastMove      internal.Move
	moveDelay     time.Duration
	autoPlaying   bool
	moveViewport  viewport.Model
	width, height int
	chesscomModel chesscomModel
}

func initialModel() model {
	// Initialize board with starting position
	board, _ := internal.ParseFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	// Create a viewport for the move list with default dimensions
	// These will be updated when we receive the window size
	vp := viewport.New(40, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4"))

	// Enable mouse wheel for scrolling
	vp.YPosition = 0

	return model{
		state:         stateMainGame,
		board:         board,
		moveList:      []string{},
		gameOver:      false,
		moveDelay:     5 * time.Millisecond,
		autoPlaying:   true,
		moveViewport:  vp,
		width:         80, // Default width
		height:        24, // Default height
		chesscomModel: initialChesscomModel(),
	}
}

func (m model) Init() tea.Cmd {
	if m.state == stateMainGame {
		return makeRandomMove(m.moveDelay)
	} else if m.state == stateChesscomDownloader {
		return m.chesscomModel.Init()
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global keys and messages first
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.state == stateMainGame {
			// Update viewport dimensions based on window size
			m.moveViewport = viewport.New(m.width/2, m.height-6)
			m.moveViewport.Style = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(1, 2)
		} else if m.state == stateChesscomDownloader {
			// Pass window size to Chess.com model
			var cmd tea.Cmd
			chessModel, cmd := m.chesscomModel.Update(msg)
			if cm, ok := chessModel.(chesscomModel); ok {
				m.chesscomModel = cm
			}
			return m, cmd
		}

		return m, nil

	case loadGameMsg:
		// Handle loading a game from Chess.com
		if m.state == stateChesscomDownloader && msg.game != nil {
			// Parse the game and create a new board
			if msg.game.Root != nil && msg.game.Root.Board != nil {
				m.board = msg.game.Root.Board
				m.moveList = []string{}
				m.gameOver = false
				m.checkmate = false
				m.stalemate = false
				m.state = stateMainGame
				return m, nil
			}
		}
	}

	// Handle state-specific updates
	if m.state == stateMainGame {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case " ", "enter":
				if m.gameOver {
					return initialModel(), nil
				}
				if !m.autoPlaying {
					return m.Update(randomMoveMsg{})
				}
			case "p":
				m.autoPlaying = !m.autoPlaying
				if m.autoPlaying {
					return m, makeRandomMove(m.moveDelay)
				}
			case "d":
				// Switch to Chess.com downloader
				m.state = stateChesscomDownloader
				m.chesscomModel = initialChesscomModel()
				return m, m.chesscomModel.Init()
			}

			// Let the viewport handle keyboard events for scrolling
			var cmd tea.Cmd
			m.moveViewport, cmd = m.moveViewport.Update(msg)
			return m, cmd

		case randomMoveMsg:
			// Check if game is already over
			if m.gameOver {
				return m, nil
			}

			// Get all legal moves
			moves := m.board.LegalMoves()

			// Check for game end
			if len(moves) == 0 || m.board.HasInsufficientMaterial() {
				check, _ := m.board.IsCheckOrMate()
				m.gameOver = true
				m.checkmate = check
				m.stalemate = !check || m.board.HasInsufficientMaterial()
				return m, nil
			}

			// Choose a random move
			randIndex := rand.Intn(len(moves))
			move := moves[randIndex]

			// Add to move list
			moveStr := move.San(m.board)
			if m.board.SideToMove == internal.White {
				m.moveList = append(m.moveList, fmt.Sprintf("%d. %s", m.board.MoveNr, moveStr))
			} else {
				lastIdx := len(m.moveList) - 1
				if lastIdx >= 0 {
					m.moveList[lastIdx] = m.moveList[lastIdx] + " " + moveStr
				}
			}

			// Make the move
			m.board = m.board.MakeMove(move)
			m.lastMove = move

			// Check if game ended after this move
			check, mate := m.board.IsCheckOrMate()
			if mate {
				m.gameOver = true
				m.checkmate = check
				m.stalemate = !check
				return m, nil
			}

			if m.board.Rule50 >= 50 {
				m.gameOver = true
				m.checkmate = false
				m.stalemate = true
				return m, nil
			}

			// Check for insufficient material (another draw condition)
			if m.board.HasInsufficientMaterial() {
				m.gameOver = true
				m.checkmate = false
				m.stalemate = true
				return m, nil
			}

			// Schedule next move if auto-playing
			if m.autoPlaying {
				return m, makeRandomMove(m.moveDelay)
			}
		}
	} else if m.state == stateChesscomDownloader {
		// Delegate to Chess.com model
		var cmd tea.Cmd
		chessModel, cmd := m.chesscomModel.Update(msg)

		// Check if we should return to the main game
		if cmd == nil {
			// See if we have a game to load
			if m.chesscomModel.gameToLoad != nil {
				// Get the game and make sure moves are parsed
				game := m.chesscomModel.gameToLoad
				if game.Root != nil && game.Root.Board != nil {
					m.board = game.Root.Board
					m.moveList = []string{}
					m.gameOver = false
					m.checkmate = false
					m.stalemate = false
					m.state = stateMainGame
					return m, nil
				}
			}

			// If no game to load, just return to main screen
			m.state = stateMainGame
			return m, nil
		}

		if cm, ok := chessModel.(chesscomModel); ok {
			m.chesscomModel = cm
		}
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.state == stateChesscomDownloader {
		return m.chesscomModel.View()
	}

	// Style definitions
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 1)

	darkSquare := lipgloss.NewStyle().
		Background(lipgloss.Color("#8B4513")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1).
		Align(lipgloss.Center).
		Width(3)

	lightSquare := lipgloss.NewStyle().
		Background(lipgloss.Color("#F5DEB3")).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 1).
		Align(lipgloss.Center).
		Width(3)

	highlightSquare := lipgloss.NewStyle().
		Background(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1).
		Align(lipgloss.Center).
		Width(3)

	boardContainerStyle := lipgloss.NewStyle().
		MarginRight(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 0)

	// Generate the chess board
	var boardOutput string
	boardOutput += "  a b c d e f g h\n"

	for rank := 7; rank >= 0; rank-- {
		boardOutput += fmt.Sprintf("%d ", 8-rank)
		for file := 0; file < 8; file++ {
			sq := internal.Square(file, rank)
			piece := m.board.Piece[sq]

			var style lipgloss.Style
			if (rank+file)%2 == 0 {
				style = lightSquare
			} else {
				style = darkSquare
			}

			// Highlight last move
			if m.lastMove != internal.NullMove {
				// For highlighting, we'll compare by square index
				fromSq := m.lastMove.From
				toSq := m.lastMove.To

				if sq == fromSq || sq == toSq {
					style = highlightSquare
				}
			}

			var pieceChar string
			switch piece.Type() {
			case internal.Pawn:
				if piece.Color() == internal.White {
					pieceChar = "♙"
				} else {
					pieceChar = "♟"
				}
			case internal.Knight:
				if piece.Color() == internal.White {
					pieceChar = "♘"
				} else {
					pieceChar = "♞"
				}
			case internal.Bishop:
				if piece.Color() == internal.White {
					pieceChar = "♗"
				} else {
					pieceChar = "♝"
				}
			case internal.Rook:
				if piece.Color() == internal.White {
					pieceChar = "♖"
				} else {
					pieceChar = "♜"
				}
			case internal.Queen:
				if piece.Color() == internal.White {
					pieceChar = "♕"
				} else {
					pieceChar = "♛"
				}
			case internal.King:
				if piece.Color() == internal.White {
					pieceChar = "♔"
				} else {
					pieceChar = "♚"
				}
			default:
				pieceChar = " "
			}

			boardOutput += style.Render(pieceChar)
		}
		boardOutput += fmt.Sprintf(" %d\n", 8-rank)
	}
	boardOutput += "  a b c d e f g h"

	// Create a string of moves
	moveListStr := ""
	for _, move := range m.moveList {
		moveListStr += move + "\n"
	}

	// Prepare the viewport with the move list
	m.moveViewport.SetContent(moveListStr)

	// Status information
	var statusInfo string
	if m.gameOver {
		if m.checkmate {
			winner := "Black"
			if m.board.SideToMove == internal.Black {
				winner = "White"
			}
			statusInfo = fmt.Sprintf("Checkmate! %s wins.", winner)
		} else if m.stalemate {
			statusInfo = "Draw by stalemate or insufficient material."
		}
		statusInfo += " Press Enter to start a new game."
	} else {
		toMove := "White"
		if m.board.SideToMove == internal.Black {
			toMove = "Black"
		}
		statusInfo = fmt.Sprintf("To move: %s | Press 'p' to pause/resume auto-play | Press 'd' for Chess.com downloader", toMove)

		check, _ := m.board.IsCheckOrMate()
		if check {
			statusInfo = fmt.Sprintf("Check! %s to move", toMove)
		}
	}

	// Combine the board and the move list side by side
	boardSection := boardContainerStyle.Render(boardOutput)
	moveSection := fmt.Sprintf("%s\n%s", titleStyle.Render("Move List"), m.moveViewport.View())

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		titleStyle.Render("GoChess TUI"),
		lipgloss.JoinHorizontal(lipgloss.Top, boardSection, moveSection),
		infoStyle.Render(statusInfo),
	)
}

// Message and command for random move generation
type randomMoveMsg struct{}

func makeRandomMove(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return randomMoveMsg{}
	})
}

func main() {
	rand.Seed(time.Now().UnixNano())
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
