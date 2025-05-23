package chesscom

import "time"

// ArchivesResponse represents the response from the archives endpoint.
type ArchivesResponse struct {
	Archives []string `json:"archives"`
}

// GamesResponse represents the response from the games endpoint.
type GamesResponse struct {
	Games []Game `json:"games"`
}

// Game represents a chess game from the Chess.com API.
type Game struct {
	URL        string `json:"url"`
	PGN        string `json:"pgn"`
	TimeControl string `json:"time_control"`
	EndTime    int64  `json:"end_time"`
	Rated      bool   `json:"rated"`
	FEN        string `json:"fen"`
	TimeClass  string `json:"time_class"`
	Rules      string `json:"rules"`
	White      Player `json:"white"`
	Black      Player `json:"black"`
	ECO        string `json:"eco"`
	
	// Optional accuracy field (may not be present in all responses)
	Accuracies *Accuracies `json:"accuracies,omitempty"`
}

// Player represents a player in a Chess.com game.
type Player struct {
	Rating   int    `json:"rating"`
	Result   string `json:"result"`
	ID       string `json:"@id"`
	Username string `json:"username"`
	UUID     string `json:"uuid"`
}

// Accuracies represents the accuracy ratings from the Chess.com API.
type Accuracies struct {
	White float64 `json:"white"`
	Black float64 `json:"black"`
}

// GetEndTime returns the game's end time as a time.Time value.
func (g *Game) GetEndTime() time.Time {
	return time.Unix(g.EndTime, 0)
}
