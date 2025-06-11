package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kyleboon/gochess/internal/pgn"
)

// DB represents a SQLite database connection for chess data
type DB struct {
	conn *sql.DB
}

// New creates a new SQLite database connection
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// createTables creates the necessary tables if they don't exist
func (db *DB) createTables() error {
	// Enable foreign keys
	_, err := db.conn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create games table (original schema, without game_hash)
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event TEXT,
			site TEXT,
			date TEXT,
			round TEXT,
			white TEXT,
			black TEXT,
			result TEXT,
			white_elo INTEGER,
			black_elo INTEGER,
			time_control TEXT,
			pgn_text TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create games table: %w", err)
	}

	// Add game_hash column if it doesn't exist (handles migration for existing databases)
	// SQLite ALTER TABLE is limited - we can't add a UNIQUE constraint directly
	err = db.addColumnIfNotExists("games", "game_hash TEXT")
	if err != nil {
		return fmt.Errorf("failed to add game_hash column: %w", err)
	}

	// Create tags table for additional metadata
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER NOT NULL,
			tag_name TEXT NOT NULL,
			tag_value TEXT NOT NULL,
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create tags table: %w", err)
	}

	// Create index on common search fields
	_, err = db.conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_games_players ON games(white, black);
		CREATE INDEX IF NOT EXISTS idx_games_date ON games(date);
		CREATE INDEX IF NOT EXISTS idx_tags ON tags(tag_name, tag_value);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_game_hash ON games(game_hash);
	`)
	
	return err
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist
func (db *DB) addColumnIfNotExists(table, columnDef string) error {
	// Extract column name from the definition
	parts := strings.Fields(columnDef)
	if len(parts) == 0 {
		return fmt.Errorf("invalid column definition: %s", columnDef)
	}
	columnName := parts[0]
	
	// Check if the column already exists
	var dummy string
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT 1", columnName, table)
	err := db.conn.QueryRow(query).Scan(&dummy)
	
	if err != nil {
		// Column doesn't exist, add it
		if strings.Contains(err.Error(), "no such column") {
			// Add the column
			alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, columnDef)
			_, err := db.conn.Exec(alterQuery)
			if err != nil {
				return fmt.Errorf("failed to add column %s: %w", columnName, err)
			}
			return nil
		}
		// If the error is something else (like empty table), we can ignore it
	}
	
	// Column already exists or table is empty
	return nil
}

// ImportPGN imports games from a PGN file into the database
func (db *DB) ImportPGN(filePath string) (int, []error) {
	allErrors := make([]error, 0)

	// Parse PGN file using our adapter that handles different PGN formats and preserves the move text
	pgnData, parseErrors := ParsePGNFileWithMoves(filePath)

	// Process PGN parsing errors
	if len(parseErrors) > 0 {
		for _, errInstance := range parseErrors {
			var originalPgnError error
			var errorLine int
			isPgnParseError := false

			// Type assertion to get *pgn.ParseError, which implements the error interface
			if pe, ok := errInstance.(*pgn.ParseError); ok {
				originalPgnError = pe
				errorLine = pe.Line
				isPgnParseError = true
			} else {
				originalPgnError = errInstance // Not a pgn.ParseError we can get line from
			}

			foundGameText := ""
			if isPgnParseError && pgnData != nil && pgnData.GameTexts != nil {
				// The parser gives a line number relative to the entire file content it parsed.
				// We need to find which of our split game texts contains that line.
				// We do this by tracking the cumulative line count.
				currentLine := 1
				for _, gameText := range pgnData.GameTexts {
					lineCount := strings.Count(gameText, "\n")
					// The preprocessor joins games with "\n\n"
					separatorLines := 2
					if errorLine >= currentLine && errorLine < currentLine+lineCount+separatorLines {
						foundGameText = gameText
						break
					}
					currentLine += lineCount + separatorLines
				}
			}
			allErrors = append(allErrors, &PGNImportError{OriginalError: originalPgnError, PGNText: foundGameText})
		}
	}

	// Check if we parsed any games
	if pgnData.PgnDB == nil || len(pgnData.PgnDB.Games) == 0 {
		if len(allErrors) > 0 { // If there were parse errors, return them
			return 0, allErrors
		}
		// If no parse errors and no games, then it's genuinely "no games found"
		noGamesErr := fmt.Errorf("no games found in PGN file: %s", filePath)
		return 0, []error{&PGNImportError{OriginalError: noGamesErr, PGNText: ""}}
	}

	// Get parsed games and their complete text
	pgnDB := pgnData.PgnDB
	gameTexts := pgnData.GameTexts

	// Begin transaction
	tx, err := db.conn.Begin()
	if err != nil {
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to begin transaction: %w", err), PGNText: ""})
		return 0, allErrors
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	// Prepare statements
	stmtGame, err := tx.Prepare(`
		INSERT INTO games (
			event, site, date, round, white, black, result, 
			white_elo, black_elo, time_control, pgn_text, game_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to prepare game statement: %w", err), PGNText: ""})
		return 0, allErrors
	}
	defer stmtGame.Close()

	stmtTag, err := tx.Prepare(`
		INSERT INTO tags (game_id, tag_name, tag_value) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to prepare tag statement: %w", err), PGNText: ""})
		return 0, allErrors
	}
	defer stmtTag.Close()

	// Count of successfully imported games
	importedCount := 0

	// Process each game
	for i, game := range pgnDB.Games {
		var currentGameText string
		if i < len(gameTexts) {
			currentGameText = gameTexts[i]
		} else {
			// This case should ideally not happen if parsing was successful
			// and gameTexts corresponds to pgnDB.Games
			err := fmt.Errorf("could not find PGN text for game index %d", i)
			allErrors = append(allErrors, &PGNImportError{OriginalError: err, PGNText: ""})
			continue
		}

		// Validate required tags
		missingTags := []string{}
		for _, requiredTag := range []string{"Event", "Site", "Date", "White", "Black", "Result"} {
			if _, ok := game.Tags[requiredTag]; !ok || game.Tags[requiredTag] == "" {
				missingTags = append(missingTags, requiredTag)
			}
		}
		if len(missingTags) > 0 {
			err := fmt.Errorf("game %d is missing required tags: %s", i+1, strings.Join(missingTags, ", "))
			allErrors = append(allErrors, &PGNImportError{OriginalError: err, PGNText: currentGameText})
			continue
		}

		// Ensure FEN tag exists - add standard starting position if missing
		if _, hasFen := game.Tags["FEN"]; !hasFen {
			game.Tags["FEN"] = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
		}

		// Calculate game hash
		moveText := ExtractMoveText(currentGameText)
		gameHash := CalculateGameHash(game, moveText)

		// Check if game_hash already exists
		var existingID int
		err := tx.QueryRow("SELECT id FROM games WHERE game_hash = ?", gameHash).Scan(&existingID)
		if err == nil {
			// Game with this hash already exists, skip.
			continue
		} else if err != sql.ErrNoRows {
			// Unexpected error during hash check.
			dbErr := fmt.Errorf("error checking game_hash for game %d (event: %s, hash: %s): %w", i+1, game.Tags["Event"], gameHash, err)
			allErrors = append(allErrors, &PGNImportError{OriginalError: dbErr, PGNText: currentGameText})
			continue
		}

		// Game hash does not exist (sql.ErrNoRows), proceed with insert.
		// Parse ELO ratings
		whiteElo, _ := strconv.Atoi(game.Tags["WhiteElo"])
		blackElo, _ := strconv.Atoi(game.Tags["BlackElo"])

		// Insert game
		res, err := stmtGame.Exec(
			game.Tags["Event"], game.Tags["Site"], game.Tags["Date"], game.Tags["Round"],
			game.Tags["White"], game.Tags["Black"], game.Tags["Result"],
			whiteElo, blackElo, game.Tags["TimeControl"],
			currentGameText, gameHash,
		)
		if err != nil {
			dbErr := fmt.Errorf("error inserting game %d (event: %s): %w", i+1, game.Tags["Event"], err)
			allErrors = append(allErrors, &PGNImportError{OriginalError: dbErr, PGNText: currentGameText})
			continue
		}

		gameID, err := res.LastInsertId()
		if err != nil {
			dbErr := fmt.Errorf("error getting last insert ID for game %d: %w", i+1, err)
			allErrors = append(allErrors, &PGNImportError{OriginalError: dbErr, PGNText: currentGameText})
			continue
		}

		// Insert tags
		for name, value := range game.Tags {
			if _, err := stmtTag.Exec(gameID, name, value); err != nil {
				dbErr := fmt.Errorf("error inserting tag for game ID %d (tag: %s): %w", gameID, name, err)
				allErrors = append(allErrors, &PGNImportError{OriginalError: dbErr, PGNText: currentGameText})
				// Continue to insert other tags
			}
		}
		importedCount++
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to commit transaction: %w", err), PGNText: ""})
		return importedCount, allErrors
	}

	return importedCount, allErrors
}

// GetGameCount returns the total number of games in the database
func (db *DB) GetGameCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get game count: %w", err)
	}
	return count, nil
}

// SearchGames searches for games matching the specified criteria
func (db *DB) SearchGames(criteria map[string]string, limit, offset int) ([]map[string]interface{}, error) {
	// Build the query
	query := "SELECT id, event, site, date, white, black, result FROM games WHERE 1=1"
	var args []interface{}
	
	// Add search criteria
	for field, value := range criteria {
		switch field {
		case "white", "black", "event", "site", "date", "result":
			query += fmt.Sprintf(" AND %s LIKE ?", field)
			args = append(args, "%"+value+"%")
		}
	}
	
	// Add limit and offset
	query += " ORDER BY date DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	
	// Execute query
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search games: %w", err)
	}
	defer rows.Close()
	
	// Process results
	var games []map[string]interface{}
	for rows.Next() {
		var id int
		var event, site, date, white, black, result string
		
		if err := rows.Scan(&id, &event, &site, &date, &white, &black, &result); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		
		game := map[string]interface{}{
			"id":     id,
			"event":  event,
			"site":   site,
			"date":   date,
			"white":  white,
			"black":  black,
			"result": result,
		}
		
		games = append(games, game)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return games, nil
}

// GetGameByID retrieves a game by its ID
func (db *DB) GetGameByID(id int) (map[string]interface{}, error) {
	// Query the game
	row := db.conn.QueryRow("SELECT * FROM games WHERE id = ?", id)
	
	var gameID int
	var event, site, date, round, white, black, result string
	var whiteElo, blackElo int
	var timeControl, pgnText string
	var createdAt string
	
	err := row.Scan(
		&gameID, &event, &site, &date, &round, &white, &black, &result,
		&whiteElo, &blackElo, &timeControl, &pgnText, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("game not found: %d", id)
		}
		return nil, fmt.Errorf("failed to scan game: %w", err)
	}
	
	// Build the game data
	game := map[string]interface{}{
		"id":           gameID,
		"event":        event,
		"site":         site,
		"date":         date,
		"round":        round,
		"white":        white,
		"black":        black,
		"result":       result,
		"white_elo":    whiteElo,
		"black_elo":    blackElo,
		"time_control": timeControl,
		"pgn_text":     pgnText,
		"created_at":   createdAt,
	}
	
	// Get all tags
	rows, err := db.conn.Query("SELECT tag_name, tag_value FROM tags WHERE game_id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()
	
	tags := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags[name] = value
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}
	
	game["tags"] = tags
	
	return game, nil
}

// ClearGames removes all games from the database
func (db *DB) ClearGames() error {
	// Begin transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	// Delete all tags first (due to foreign key constraints)
	_, err = tx.Exec("DELETE FROM tags")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete tags: %w", err)
	}

	// Delete all games
	_, err = tx.Exec("DELETE FROM games")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete games: %w", err)
	}

	// Reset the auto-increment counters
	_, err = tx.Exec("DELETE FROM sqlite_sequence WHERE name='games' OR name='tags'")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to reset sequence: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// PlayerStats represents statistics for a player
type PlayerStats struct {
	Name       string  // Player's name
	Games      int     // Total games played
	Wins       int     // Total wins
	Losses     int     // Total losses
	Draws      int     // Total draws
	WinRate    float64 // Win rate as a percentage
	WhiteGames int     // Games played as white
	BlackGames int     // Games played as black
	WhiteWins  int     // Wins as white
	BlackWins  int     // Wins as black
}

// GetPlayerStats retrieves statistics for all players in the database
func (db *DB) GetPlayerStats() ([]PlayerStats, error) {
	// Query all games in the database
	rows, err := db.conn.Query(`
		SELECT white, black, result 
		FROM games
		WHERE white != '' AND black != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer rows.Close()
	
	// Map to track player statistics
	playerStats := make(map[string]*PlayerStats)
	
	// Process each game
	for rows.Next() {
		var white, black, result string
		if err := rows.Scan(&white, &black, &result); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		
		// Initialize player stats if not yet tracked
		if _, ok := playerStats[white]; !ok {
			playerStats[white] = &PlayerStats{Name: white}
		}
		if _, ok := playerStats[black]; !ok {
			playerStats[black] = &PlayerStats{Name: black}
		}
		
		// Update game counts
		playerStats[white].Games++
		playerStats[white].WhiteGames++
		playerStats[black].Games++
		playerStats[black].BlackGames++
		
		// Update win/loss/draw counts based on result
		switch result {
		case "1-0": // White win
			playerStats[white].Wins++
			playerStats[white].WhiteWins++
			playerStats[black].Losses++
		case "0-1": // Black win
			playerStats[black].Wins++
			playerStats[black].BlackWins++
			playerStats[white].Losses++
		case "1/2-1/2": // Draw
			playerStats[white].Draws++
			playerStats[black].Draws++
		default: // Unknown result or ongoing game
			// Skip updating win/loss/draw counts
		}
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	// Calculate win rates and convert to slice
	result := make([]PlayerStats, 0, len(playerStats))
	for _, stats := range playerStats {
		// Calculate win rate
		if stats.Games > 0 {
			stats.WinRate = float64(stats.Wins) / float64(stats.Games) * 100.0
		}
		
		result = append(result, *stats)
	}
	
	// Sort by number of games (most active players first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Games > result[j].Games
	})
	
	return result, nil
}
