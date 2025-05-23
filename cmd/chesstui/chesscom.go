package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kyleboon/gochess/internal/chesscom"
	"github.com/kyleboon/gochess/internal/pgn"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Padding(0, 1).
			Bold(true)
)

type chesscomStatus int

const (
	inputUsername chesscomStatus = iota
	loadingArchives
	selectArchive
	loadingGames
	displayGames
	loadingPGN
	displayError
)

type chesscomModel struct {
	status        chesscomStatus
	usernameInput textinput.Model
	archives      list.Model
	games         list.Model
	errorMsg      string
	spinner       spinner.Model
	viewport      viewport.Model
	pgn           string
	client        *chesscom.Client
	width         int
	height        int
	pgnDB         *pgn.DB
	gameToLoad    *pgn.Game
}

type archiveItem struct {
	url  string
	date string
}

func (i archiveItem) Title() string       { return i.date }
func (i archiveItem) Description() string { return "" }
func (i archiveItem) FilterValue() string { return i.date }

type gameItem struct {
	game  chesscom.Game
	index int
}

func (i gameItem) Title() string {
	whitePlayer := i.game.White.Username
	blackPlayer := i.game.Black.Username
	result := fmt.Sprintf("%s - %s", i.game.White.Result, i.game.Black.Result)
	timeControl := i.game.TimeControl
	return fmt.Sprintf("#%d: %s vs %s (%s) [%s]",
		i.index+1, whitePlayer, blackPlayer, result, timeControl)
}

func (i gameItem) Description() string {
	endTime := time.Unix(i.game.EndTime, 0).Format("2006-01-02 15:04:05")
	eco := ""
	if i.game.ECO != "" {
		eco = fmt.Sprintf("ECO: %s, ", i.game.ECO)
	}
	return fmt.Sprintf("%sTime Control: %s, Played on: %s",
		eco, i.game.TimeControl, endTime)
}

func (i gameItem) FilterValue() string {
	return fmt.Sprintf("%s %s", i.game.White.Username, i.game.Black.Username)
}

func initialChesscomModel() chesscomModel {
	// Username input
	ti := textinput.New()
	ti.Placeholder = "Enter Chess.com username"
	ti.Focus()
	ti.CharLimit = 30
	ti.Width = 30

	// Initialize spinner for loading states
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	// List models
	archivesList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	archivesList.Title = "Available Archives"
	archivesList.SetShowStatusBar(false)
	archivesList.SetFilteringEnabled(false)
	archivesList.Styles.Title = titleStyle

	gamesList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	gamesList.Title = "Games"
	gamesList.SetShowStatusBar(true)
	gamesList.SetFilteringEnabled(true)
	gamesList.Styles.Title = titleStyle

	// Viewport for PGN display
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4"))

	return chesscomModel{
		status:        inputUsername,
		usernameInput: ti,
		archives:      archivesList,
		games:         gamesList,
		spinner:       s,
		viewport:      vp,
		client:        chesscom.NewClient(),
		width:         80,
		height:        24,
	}
}

func (m chesscomModel) Init() tea.Cmd {
	return textinput.Blink
}

func fetchArchives(username string) tea.Cmd {
	return func() tea.Msg {
		client := chesscom.NewClient()
		archives, err := client.GetArchivedMonths(username)
		if err != nil {
			return fetchArchivesErrMsg{err: err}
		}
		return fetchArchivesMsg{archives: archives}
	}
}

func fetchGames(archiveURL string) tea.Cmd {
	return func() tea.Msg {
		// Parse URL to get year and month
		// Example URL: https://api.chess.com/pub/player/username/games/2023/01
		parts := strings.Split(archiveURL, "/")
		if len(parts) < 2 {
			return fetchGamesErrMsg{err: fmt.Errorf("invalid archive URL")}
		}

		month, _ := strconv.Atoi(parts[len(parts)-1])
		year, _ := strconv.Atoi(parts[len(parts)-2])
		username := parts[len(parts)-4]

		client := chesscom.NewClient()
		games, err := client.GetPlayerGames(username, year, month)
		if err != nil {
			return fetchGamesErrMsg{err: err}
		}
		return fetchGamesMsg{games: games}
	}
}

func fetchPGN(archiveURL string) tea.Cmd {
	return func() tea.Msg {
		// Parse URL to get year and month
		parts := strings.Split(archiveURL, "/")
		if len(parts) < 2 {
			return fetchPGNErrMsg{err: fmt.Errorf("invalid archive URL")}
		}

		month, _ := strconv.Atoi(parts[len(parts)-1])
		year, _ := strconv.Atoi(parts[len(parts)-2])
		username := parts[len(parts)-4]

		client := chesscom.NewClient()
		pgnData, err := client.GetPlayerGamesPGN(username, year, month)
		if err != nil {
			return fetchPGNErrMsg{err: err}
		}

		// Parse PGN
		db := &pgn.DB{}
		errs := db.Parse(pgnData)
		if len(errs) > 0 {
			return fetchPGNErrMsg{err: errs[0]}
		}

		return fetchPGNMsg{pgnData: pgnData, db: db}
	}
}

type fetchArchivesMsg struct {
	archives *chesscom.ArchivesResponse
}

type fetchArchivesErrMsg struct {
	err error
}

type fetchGamesMsg struct {
	games *chesscom.GamesResponse
}

type fetchGamesErrMsg struct {
	err error
}

type fetchPGNMsg struct {
	pgnData string
	db      *pgn.DB
}

type fetchPGNErrMsg struct {
	err error
}

type loadGameMsg struct {
	game *pgn.Game
}

func (m chesscomModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.status != inputUsername {
				return initialChesscomModel(), nil
			}
			return m, nil // Return nil to indicate quitting

		case "q", "ctrl+c":
			return m, nil // Return nil to indicate quitting

		case "enter":
			switch m.status {
			case inputUsername:
				username := strings.TrimSpace(m.usernameInput.Value())
				if username == "" {
					return m, nil
				}
				m.status = loadingArchives
				return m, tea.Batch(
					fetchArchives(username),
					m.spinner.Tick,
				)

			case selectArchive:
				if len(m.archives.Items()) == 0 {
					return m, nil
				}

				i, ok := m.archives.SelectedItem().(archiveItem)
				if !ok {
					return m, nil
				}

				m.status = loadingGames
				return m, tea.Batch(
					fetchGames(i.url),
					m.spinner.Tick,
				)

			case displayGames:
				if len(m.games.Items()) == 0 {
					return m, nil
				}

				i, ok := m.games.SelectedItem().(gameItem)
				if !ok {
					return m, nil
				}

				// Load selected PGN into viewport
				m.viewport.SetContent(i.game.PGN)

				// Setup for loading the game
				m.status = loadingPGN
				return m, fetchPGN(i.game.URL)
			}
		}

	case loadGameMsg:
		m.gameToLoad = msg.game
		return m, nil // Return nil to indicate quitting

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update viewport dimensions
		m.viewport.Width = m.width
		m.viewport.Height = m.height - 4

		// Update list dimensions
		m.archives.SetSize(m.width, m.height-6)
		m.games.SetSize(m.width, m.height-6)

		return m, nil

	case fetchArchivesMsg:
		m.status = selectArchive

		// Convert archives to list items
		var items []list.Item
		for _, url := range msg.archives.Archives {
			// Parse date from URL
			parts := strings.Split(url, "/")
			if len(parts) >= 2 {
				year := parts[len(parts)-2]
				month := parts[len(parts)-1]
				// Convert month number to name
				monthInt, _ := strconv.Atoi(month)
				monthName := time.Month(monthInt).String()

				items = append(items, archiveItem{
					url:  url,
					date: fmt.Sprintf("%s %s", monthName, year),
				})
			}
		}

		// Sort items in reverse chronological order (newest first)
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}

		m.archives.SetItems(items)

		return m, nil

	case fetchArchivesErrMsg:
		m.status = displayError
		m.errorMsg = fmt.Sprintf("Error fetching archives: %v", msg.err)
		return m, nil

	case fetchGamesMsg:
		m.status = displayGames

		// Convert games to list items
		var items []list.Item
		for i, game := range msg.games.Games {
			items = append(items, gameItem{
				game:  game,
				index: i,
			})
		}

		m.games.SetItems(items)

		return m, nil

	case fetchGamesErrMsg:
		m.status = displayError
		m.errorMsg = fmt.Sprintf("Error fetching games: %v", msg.err)
		return m, nil

	case fetchPGNMsg:
		m.status = loadingPGN
		m.pgn = msg.pgnData
		m.pgnDB = msg.db

		// If there are games, load the first one
		if len(msg.db.Games) > 0 {
			// Parse moves for the game
			err := msg.db.ParseMoves(msg.db.Games[0])
			if err != nil {
				m.status = displayError
				m.errorMsg = fmt.Sprintf("Error parsing moves: %v", err)
				return m, nil
			}

			return m, tea.Sequence(
				func() tea.Msg {
					return loadGameMsg{game: msg.db.Games[0]}
				},
			)
		}

		return m, nil

	case fetchPGNErrMsg:
		m.status = displayError
		m.errorMsg = fmt.Sprintf("Error fetching PGN: %v", msg.err)
		return m, nil

	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Handle updates for sub-components
	switch m.status {
	case inputUsername:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
		cmds = append(cmds, cmd)

	case selectArchive:
		m.archives, cmd = m.archives.Update(msg)
		cmds = append(cmds, cmd)

	case displayGames:
		m.games, cmd = m.games.Update(msg)
		cmds = append(cmds, cmd)

		// Also update viewport for viewing PGN content
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m chesscomModel) View() string {
	switch m.status {
	case inputUsername:
		return fmt.Sprintf(
			"\n%s\n\n%s\n\n%s\n\n%s",
			titleStyle.Render("Chess.com Game Downloader"),
			"Enter a Chess.com username to download games:",
			m.usernameInput.View(),
			"Press Enter to continue, Esc to quit",
		)

	case loadingArchives:
		return fmt.Sprintf(
			"\n%s\n\n%s %s\n\n",
			titleStyle.Render("Chess.com Game Downloader"),
			m.spinner.View(),
			fmt.Sprintf("Loading archives for %s...", m.usernameInput.Value()),
		)

	case selectArchive:
		return fmt.Sprintf(
			"\n%s\n\n%s\n\n%s",
			titleStyle.Render("Chess.com Game Downloader"),
			m.archives.View(),
			"Select an archive and press Enter to view games, Esc to go back",
		)

	case loadingGames:
		return fmt.Sprintf(
			"\n%s\n\n%s %s\n\n",
			titleStyle.Render("Chess.com Game Downloader"),
			m.spinner.View(),
			"Loading games...",
		)

	case displayGames:
		selectedGame := ""
		if i, ok := m.games.SelectedItem().(gameItem); ok {
			selectedGame = i.game.PGN
		}

		// Split the view - games list on top, PGN viewer on bottom
		if selectedGame != "" {
			m.viewport.SetContent(selectedGame)
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n\n%s",
				titleStyle.Render("Chess.com Game Downloader"),
				m.games.View(),
				subtitleStyle.Render("Game PGN:"),
				m.viewport.View(),
			)
		}

		return fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			titleStyle.Render("Chess.com Game Downloader"),
			m.games.View(),
			"Select a game to view its PGN, Enter to load into board, Esc to go back",
		)

	case displayError:
		return fmt.Sprintf(
			"\n%s\n\n%s\n\n%s",
			titleStyle.Render("Chess.com Game Downloader"),
			errorStyle.Render(fmt.Sprintf("Error: %s", m.errorMsg)),
			"Press Esc to go back",
		)

	default:
		return "Loading..."
	}
}
