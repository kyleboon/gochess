package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Game represents a chess game for display purposes
type Game struct {
	ID               int
	Event            string
	Site             string
	Date             string
	White            string
	Black            string
	Result           string
	WhiteElo         int
	BlackElo         int
	TimeControl      string
	PGNText          string
	ECOCode          string
	OpeningName      string
	OpeningVariation string
}

// gameItem represents a game in the list
type gameItem struct {
	game Game
}

func (i gameItem) Title() string {
	return fmt.Sprintf("%s vs %s", i.game.White, i.game.Black)
}

func (i gameItem) Description() string {
	result := i.game.Result
	if result == "" {
		result = "?"
	}

	eloInfo := ""
	if i.game.WhiteElo > 0 || i.game.BlackElo > 0 {
		eloInfo = fmt.Sprintf(" (%d vs %d)", i.game.WhiteElo, i.game.BlackElo)
	}

	return fmt.Sprintf("%s - %s%s - %s", i.game.Date, result, eloInfo, i.game.Event)
}

func (i gameItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s", i.game.White, i.game.Black, i.game.Event, i.game.Date)
}

// GameListModel represents the game list browser
type GameListModel struct {
	list     list.Model
	games    []Game
	selected *Game
	quitting bool
	width    int
	height   int
}

// NewGameListModel creates a new game list browser
func NewGameListModel(games []Game) GameListModel {
	items := make([]list.Item, len(games))
	for i, game := range games {
		items[i] = gameItem{game: game}
	}

	const defaultWidth = 100
	const defaultHeight = 20

	delegate := list.NewDefaultDelegate()

	// Customize delegate colors
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(ColorPrimary).
		BorderForeground(ColorPrimary).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(ColorAccent).
		BorderForeground(ColorPrimary)

	l := list.New(items, delegate, defaultWidth, defaultHeight)
	l.Title = "♔ Chess Games Browser"
	l.Styles.Title = TitleStyle
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(ColorTextMuted)
	l.Styles.HelpStyle = HelpStyle

	return GameListModel{
		list:   l,
		games:  games,
		width:  defaultWidth,
		height: defaultHeight,
	}
}

// Init initializes the model
func (m GameListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m GameListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			// Select the current game
			i, ok := m.list.SelectedItem().(gameItem)
			if ok {
				m.selected = &i.game
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the model
func (m GameListModel) View() string {
	if m.quitting {
		return "Thanks for using GoChess!\n"
	}

	// If a game is selected, show its details
	if m.selected != nil {
		return m.renderGameDetails()
	}

	return m.list.View()
}

// renderGameDetails renders the details of the selected game
func (m GameListModel) renderGameDetails() string {
	var b strings.Builder

	game := m.selected

	// Title
	title := TitleStyle.Render(fmt.Sprintf("♔ Game #%d", game.ID))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Players
	b.WriteString(SubtitleStyle.Render("Players"))
	b.WriteString("\n")

	whiteElo := ""
	if game.WhiteElo > 0 {
		whiteElo = fmt.Sprintf(" (%d)", game.WhiteElo)
	}
	blackElo := ""
	if game.BlackElo > 0 {
		blackElo = fmt.Sprintf(" (%d)", game.BlackElo)
	}

	fmt.Fprintf(&b, "  White: %s%s\n", StatValueStyle.Render(game.White), whiteElo)
	fmt.Fprintf(&b, "  Black: %s%s\n", StatValueStyle.Render(game.Black), blackElo)

	// Result with color
	resultStyle := lipgloss.NewStyle()
	switch game.Result {
	case "1-0":
		resultStyle = WinStyle
	case "0-1":
		resultStyle = LossStyle
	case "1/2-1/2":
		resultStyle = DrawStyle
	}
	fmt.Fprintf(&b, "  Result: %s\n", resultStyle.Render(game.Result))

	// Game info
	b.WriteString("\n")
	b.WriteString(SubtitleStyle.Render("Game Information"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Event: %s\n", game.Event)
	fmt.Fprintf(&b, "  Site: %s\n", game.Site)
	fmt.Fprintf(&b, "  Date: %s\n", game.Date)
	if game.TimeControl != "" {
		fmt.Fprintf(&b, "  Time Control: %s\n", game.TimeControl)
	}
	if game.ECOCode != "" {
		fmt.Fprintf(&b, "  Opening: %s - %s (%s)\n",
			StatValueStyle.Render(game.ECOCode),
			game.OpeningName,
			game.OpeningVariation)
	}

	// PGN (if available and not too long)
	if game.PGNText != "" {
		b.WriteString("\n")
		b.WriteString(SubtitleStyle.Render("PGN"))
		b.WriteString("\n")

		// Truncate very long PGN
		pgn := game.PGNText
		const maxPGNLength = 500
		if len(pgn) > maxPGNLength {
			pgn = pgn[:maxPGNLength] + "..."
		}

		pgnStyle := lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true).
			Width(m.width - 4)
		b.WriteString(pgnStyle.Render(pgn))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Press 'q' to go back, 'ctrl+c' to quit"))

	return BorderStyle.Render(b.String())
}

// GetSelectedGame returns the currently selected game
func (m GameListModel) GetSelectedGame() *Game {
	return m.selected
}

// MapToGame converts a map[string]interface{} from the database to a Game struct
func MapToGame(m map[string]interface{}) Game {
	game := Game{}

	if id, ok := m["id"].(int); ok {
		game.ID = id
	}
	if event, ok := m["event"].(string); ok {
		game.Event = event
	}
	if site, ok := m["site"].(string); ok {
		game.Site = site
	}
	if date, ok := m["date"].(string); ok {
		game.Date = date
	}
	if white, ok := m["white"].(string); ok {
		game.White = white
	}
	if black, ok := m["black"].(string); ok {
		game.Black = black
	}
	if result, ok := m["result"].(string); ok {
		game.Result = result
	}
	if whiteElo, ok := m["white_elo"].(int); ok {
		game.WhiteElo = whiteElo
	}
	if blackElo, ok := m["black_elo"].(int); ok {
		game.BlackElo = blackElo
	}
	if timeControl, ok := m["time_control"].(string); ok {
		game.TimeControl = timeControl
	}
	if pgnText, ok := m["pgn_text"].(string); ok {
		game.PGNText = pgnText
	}
	if ecoCode, ok := m["eco_code"].(string); ok {
		game.ECOCode = ecoCode
	}
	if openingName, ok := m["opening_name"].(string); ok {
		game.OpeningName = openingName
	}

	return game
}
