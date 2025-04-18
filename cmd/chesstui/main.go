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

type model struct {
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
		board:        board,
		moveList:     []string{},
		gameOver:     false,
		moveDelay:    5 * time.Millisecond,
		autoPlaying:  true,
		moveViewport: vp,
		width:        80, // Default width
		height:       24, // Default height
	}
}

func (m model) Init() tea.Cmd {
	return makeRandomMove(m.moveDelay)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update viewport dimensions based on window size
		m.moveViewport = viewport.New(m.width/2, m.height-6)
		m.moveViewport.Style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)
		
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
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
		Padding(0, 1)

	// Render the title
	title := titleStyle.Render("Random Chess Game")

	// Render the board
	boardContent := ""
	files := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	fileLabel := "  "
	for _, f := range files {
		fileLabel += fmt.Sprintf(" %s ", f)
	}
	boardContent += fileLabel + "\n"

	for rank := 7; rank >= 0; rank-- {
		rankLine := fmt.Sprintf("%d ", rank+1)
		for file := 0; file < 8; file++ {
			sq := internal.Square(file, rank)
			piece := m.board.Piece[sq]

			// Determine piece character to display
			pieceChar := " "
			if piece != internal.NoPiece {
				pieceChar = string(internal.Glyphs[piece])
			}

			// Determine square style (dark/light) and highlight for last move
			squareStyle := lightSquare
			if (file+rank)%2 == 1 {
				squareStyle = darkSquare
			}

			// Highlight last move squares
			if !m.gameOver && (sq == m.lastMove.From || sq == m.lastMove.To) {
				squareStyle = highlightSquare
			}

			rankLine += squareStyle.Render(pieceChar)
		}
		rankLine += fmt.Sprintf(" %d", rank+1)
		boardContent += rankLine + "\n"
	}

	boardContent += fileLabel

	// Game status
	statusContent := ""
	if m.gameOver {
		if m.checkmate {
			// Determine winner
			winner := "Black"
			if m.board.SideToMove == internal.Black {
				winner = "White"
			}
			statusContent += infoStyle.Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Checkmate! %s wins!", winner))
		} else if m.stalemate {
			statusContent += infoStyle.Foreground(lipgloss.Color("#FFFF00")).Render("Stalemate! Game drawn.")
		}
		statusContent += "\n" + infoStyle.Render("Press Enter or Space to play again.")
	} else {
		sideToMove := "White"
		if m.board.SideToMove == internal.Black {
			sideToMove = "Black"
		}
		statusContent += infoStyle.Render(fmt.Sprintf("Side to move: %s", sideToMove))

		if m.autoPlaying {
			statusContent += "\n" + infoStyle.Render("Auto-playing moves (press 'p' to pause)")
		} else {
			statusContent += "\n" + infoStyle.Render("Paused (press 'p' to resume auto-play or Enter for next move)")
		}
	}
	
	statusContent += "\n" + infoStyle.Render("Press 'q' to quit")

	// Left panel with board and status
	leftPanel := boardContainerStyle.Render(boardContent + "\n\n" + statusContent)
	
	// Prepare move list content for the viewport
	moveListContent := ""
	for i, move := range m.moveList {
		moveNumber := i/2 + 1
		if i%2 == 0 {
			moveListContent += fmt.Sprintf("%d. %s", moveNumber, move)
		} else {
			moveListContent += fmt.Sprintf(" %s\n", move)
		}
	}
	
	// If the last move was white's move (odd number of moves), add a newline
	if len(m.moveList)%2 != 0 {
		moveListContent += "\n"
	}
	
	// Update viewport content
	m.moveViewport.SetContent(moveListContent)
	
	// Create and style the moves panel title
	moveListTitle := titleStyle.Render("Move List")
	
	// Combine everything
	rightPanel := lipgloss.JoinVertical(
		lipgloss.Left,
		moveListTitle,
		m.moveViewport.View(),
	)
	
	// Final layout - fixed board on left, scrollable moves on right
	layout := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		rightPanel,
	)
	
	// Add the title above the layout
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		layout,
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

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
	}
}
