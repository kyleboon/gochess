package lichess

// GamesParams holds the parameters for fetching games from Lichess.
type GamesParams struct {
	// Username is the Lichess username to fetch games for (required)
	Username string

	// Since filters games played after this timestamp (Unix milliseconds, optional)
	Since *int64

	// Until filters games played before this timestamp (Unix milliseconds, optional)
	Until *int64

	// Max limits the number of games to fetch (optional, default is all games)
	Max *int

	// Vs filters games against a specific opponent username (optional)
	Vs string

	// Rated filters for rated games only when true (optional)
	Rated *bool

	// PerfType filters by game type: ultraBullet, bullet, blitz, rapid, classical, correspondence (optional)
	PerfType string

	// Color filters games by color: white or black (optional)
	Color string

	// Analyzed filters for games that have been analyzed (optional)
	Analyzed *bool

	// Moves includes the moves in PGN format (default: true)
	Moves bool

	// Tags includes PGN tags (default: true)
	Tags bool

	// Clocks includes clock comments in PGN (default: true)
	Clocks bool

	// Evals includes evaluation comments in PGN (default: true)
	Evals bool

	// Opening includes opening information (default: true)
	Opening bool

	// Ongoing includes ongoing games (default: false)
	Ongoing bool

	// Finished includes finished games (default: true)
	Finished bool

	// Sort defines the sort order: dateAsc or dateDesc (default: dateDesc)
	Sort string
}

// DefaultGamesParams returns a GamesParams struct with sensible defaults.
func DefaultGamesParams(username string) GamesParams {
	return GamesParams{
		Username: username,
		Moves:    true,
		Tags:     true,
		Clocks:   true,
		Evals:    true,
		Opening:  true,
		Ongoing:  false,
		Finished: true,
		Sort:     "dateDesc",
	}
}
