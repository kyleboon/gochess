package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kyleboon/gochess/internal/eco"
	"github.com/kyleboon/gochess/internal/logging"
	"github.com/kyleboon/gochess/internal/pgn"
)

// DB represents a SQLite database connection for chess data
type DB struct {
	conn   *sql.DB
	logger *slog.Logger
	ecoDB  *eco.Database
}

// New creates a new SQLite database connection
func New(dbPath string) (*DB, error) {
	return NewWithLogger(dbPath, logging.Default())
}

// NewWithLogger creates a new SQLite database connection with a custom logger
func NewWithLogger(dbPath string, logger *slog.Logger) (*DB, error) {
	logger.Debug("opening database", "path", dbPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		logger.Error("failed to create database directory", "path", dbPath, "error", err)
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Error("failed to open database connection", "path", dbPath, "error", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize ECO database
	ecoDB, err := eco.NewDatabaseWithLogger(logger)
	if err != nil {
		_ = conn.Close()
		logger.Error("failed to initialize ECO database", "error", err)
		return nil, fmt.Errorf("failed to initialize ECO database: %w", err)
	}

	db := &DB{
		conn:   conn,
		logger: logger,
		ecoDB:  ecoDB,
	}

	if err := db.createTables(); err != nil {
		_ = conn.Close()
		logger.Error("failed to create database tables", "error", err)
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Debug("database opened successfully", "path", dbPath)
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

	// Add ECO opening classification columns
	err = db.addColumnIfNotExists("games", "eco_code TEXT")
	if err != nil {
		return fmt.Errorf("failed to add eco_code column: %w", err)
	}

	err = db.addColumnIfNotExists("games", "opening_name TEXT")
	if err != nil {
		return fmt.Errorf("failed to add opening_name column: %w", err)
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

	// Create positions table for position-based search
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS positions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER NOT NULL,
			move_number INTEGER NOT NULL,
			fen TEXT NOT NULL,
			next_move TEXT,
			evaluation REAL,
			FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create positions table: %w", err)
	}

	// Add ECO opening classification columns to positions
	err = db.addColumnIfNotExists("positions", "eco_code TEXT")
	if err != nil {
		return fmt.Errorf("failed to add eco_code column to positions: %w", err)
	}

	err = db.addColumnIfNotExists("positions", "opening_name TEXT")
	if err != nil {
		return fmt.Errorf("failed to add opening_name column to positions: %w", err)
	}

	// Create index on common search fields
	_, err = db.conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_games_players ON games(white, black);
		CREATE INDEX IF NOT EXISTS idx_games_date ON games(date);
		CREATE INDEX IF NOT EXISTS idx_tags ON tags(tag_name, tag_value);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_game_hash ON games(game_hash);
		CREATE INDEX IF NOT EXISTS idx_games_eco ON games(eco_code);
		CREATE INDEX IF NOT EXISTS idx_positions_fen ON positions(fen);
		CREATE INDEX IF NOT EXISTS idx_positions_game_id ON positions(game_id);
		CREATE INDEX IF NOT EXISTS idx_positions_eco ON positions(eco_code);
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

// processParseErrors converts parsing errors into PGNImportErrors with associated game text
func processParseErrors(parseErrors []error, pgnData *PGNData) []error {
	allErrors := make([]error, 0, len(parseErrors))

	for _, errInstance := range parseErrors {
		var originalPgnError error
		var errorLine int
		isPgnParseError := false

		// Type assertion to get *pgn.ParseError
		if pe, ok := errInstance.(*pgn.ParseError); ok {
			originalPgnError = pe
			errorLine = pe.Line
			isPgnParseError = true
		} else {
			originalPgnError = errInstance
		}

		foundGameText := ""
		if isPgnParseError && pgnData != nil && pgnData.GameTexts != nil {
			// Find which game text contains the error line
			currentLine := 1
			for _, gameText := range pgnData.GameTexts {
				lineCount := strings.Count(gameText, "\n")
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

	return allErrors
}

// validateGameTags checks if all required tags are present in a game
func validateGameTags(game *pgn.Game) error {
	requiredTags := []string{"Event", "Site", "Date", "White", "Black", "Result"}
	missingTags := []string{}

	for _, tag := range requiredTags {
		if _, ok := game.Tags[tag]; !ok || game.Tags[tag] == "" {
			missingTags = append(missingTags, tag)
		}
	}

	if len(missingTags) > 0 {
		return fmt.Errorf("missing required tags: %s", strings.Join(missingTags, ", "))
	}

	return nil
}

// checkDuplicateGame checks if a game with the given hash already exists in the database
func checkDuplicateGame(tx *sql.Tx, gameHash string) (bool, error) {
	var existingID int
	err := tx.QueryRow("SELECT id FROM games WHERE game_hash = ?", gameHash).Scan(&existingID)
	switch err {
	case nil:
		// Game exists
		return true, nil
	case sql.ErrNoRows:
		// Game doesn't exist
		return false, nil
	default:
		// Unexpected error
		return false, err
	}
}

// insertGameRecord inserts a game and its tags into the database and returns the game ID
func insertGameRecord(ctx context.Context, tx *sql.Tx, stmtGame, stmtTag *sql.Stmt, game *pgn.Game, gameText, gameHash, ecoCode, openingName string) (int64, error) {
	// Parse ELO ratings
	whiteElo, _ := strconv.Atoi(game.Tags["WhiteElo"])
	blackElo, _ := strconv.Atoi(game.Tags["BlackElo"])

	// Insert game
	res, err := stmtGame.Exec(
		game.Tags["Event"], game.Tags["Site"], game.Tags["Date"], game.Tags["Round"],
		game.Tags["White"], game.Tags["Black"], game.Tags["Result"],
		whiteElo, blackElo, game.Tags["TimeControl"],
		gameText, gameHash, ecoCode, openingName,
	)
	if err != nil {
		return 0, fmt.Errorf("error inserting game: %w", err)
	}

	gameID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting last insert ID: %w", err)
	}

	// Insert tags
	for name, value := range game.Tags {
		if _, err := stmtTag.Exec(gameID, name, value); err != nil {
			return 0, fmt.Errorf("error inserting tag %s: %w", name, err)
		}
	}

	return gameID, nil
}

// insertPositions inserts all positions for a game into the database
func insertPositions(ctx context.Context, tx *sql.Tx, stmtPosition *sql.Stmt, gameID int64, positions []Position, ecoCode, openingName string) error {
	for _, pos := range positions {
		_, err := stmtPosition.Exec(gameID, pos.MoveNumber, pos.FEN, pos.NextMove, nil, ecoCode, openingName)
		if err != nil {
			return fmt.Errorf("error inserting position at move %d: %w", pos.MoveNumber, err)
		}
	}
	return nil
}

// ImportPGN imports games from a PGN file into the database
func (db *DB) ImportPGN(ctx context.Context, filePath string) (int, []error) {
	db.logger.Info("starting PGN import", "file", filePath)
	allErrors := make([]error, 0)

	// Parse PGN file using our adapter that handles different PGN formats and preserves the move text
	pgnData, parseErrors := ParsePGNFileWithMoves(filePath)
	db.logger.Debug("PGN file parsed", "file", filePath, "parseErrors", len(parseErrors))

	// Process PGN parsing errors
	if len(parseErrors) > 0 {
		allErrors = append(allErrors, processParseErrors(parseErrors, pgnData)...)
	}

	// Check if we parsed any games
	if pgnData.PgnDB == nil || len(pgnData.PgnDB.Games) == 0 {
		db.logger.Warn("no games found in PGN file", "file", filePath, "errors", len(allErrors))
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
	db.logger.Debug("games parsed successfully", "file", filePath, "totalGames", len(pgnDB.Games))

	// Begin transaction
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		db.logger.Error("failed to begin transaction", "error", err)
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to begin transaction: %w", err), PGNText: ""})
		return 0, allErrors
	}
	db.logger.Debug("transaction started")
	defer func() {
		if err := recover(); err != nil {
			_ = tx.Rollback()
			panic(err)
		}
	}()

	// Prepare statements
	stmtGame, err := tx.PrepareContext(ctx, `
		INSERT INTO games (
			event, site, date, round, white, black, result,
			white_elo, black_elo, time_control, pgn_text, game_hash, eco_code, opening_name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to prepare game statement: %w", err), PGNText: ""})
		return 0, allErrors
	}
	defer func() { _ = stmtGame.Close() }()

	stmtTag, err := tx.PrepareContext(ctx, `
		INSERT INTO tags (game_id, tag_name, tag_value)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to prepare tag statement: %w", err), PGNText: ""})
		return 0, allErrors
	}
	defer func() { _ = stmtTag.Close() }()

	stmtPosition, err := tx.PrepareContext(ctx, `
		INSERT INTO positions (game_id, move_number, fen, next_move, evaluation, eco_code, opening_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to prepare position statement: %w", err), PGNText: ""})
		return 0, allErrors
	}
	defer func() { _ = stmtPosition.Close() }()

	// Count of successfully imported games
	importedCount := 0

	// Process each game
	for i, game := range pgnDB.Games {
		var currentGameText string
		if i < len(gameTexts) {
			currentGameText = gameTexts[i]
		} else {
			err := fmt.Errorf("could not find PGN text for game index %d", i)
			allErrors = append(allErrors, &PGNImportError{OriginalError: err, PGNText: ""})
			continue
		}

		// Validate required tags
		if err := validateGameTags(game); err != nil {
			allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("game %d: %w", i+1, err), PGNText: currentGameText})
			continue
		}

		// Ensure FEN tag exists - add standard starting position if missing
		if _, hasFen := game.Tags["FEN"]; !hasFen {
			game.Tags["FEN"] = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
		}

		// Calculate game hash
		moveText := ExtractMoveText(currentGameText)
		gameHash := CalculateGameHash(game, moveText)

		// Check if game already exists
		isDuplicate, err := checkDuplicateGame(tx, gameHash)
		if err != nil {
			dbErr := fmt.Errorf("error checking duplicate for game %d (event: %s): %w", i+1, game.Tags["Event"], err)
			allErrors = append(allErrors, &PGNImportError{OriginalError: dbErr, PGNText: currentGameText})
			continue
		}
		if isDuplicate {
			// Game already exists, skip
			continue
		}

		// Parse moves for ECO classification and position extraction
		var ecoCode, openingName string
		if err := pgnDB.ParseMoves(game); err != nil {
			db.logger.Warn("failed to parse moves for game",
				"event", game.Tags["Event"], "error", err)
			// Don't fail the import if move parsing fails
		} else {
			// Classify opening using ECO database
			if game.Root != nil && game.Root.Next != nil {
				// Extract SAN moves from the game tree
				moveStrs := extractMoveStrings(game)
				if len(moveStrs) > 0 {
					ecoCode, openingName, _ = db.ecoDB.Classify(moveStrs)
					if ecoCode != "" {
						db.logger.Debug("opening classified",
							"eco", ecoCode, "opening", openingName, "event", game.Tags["Event"])
					}
				}
			}
		}

		// Insert game and tags
		gameID, err := insertGameRecord(ctx, tx, stmtGame, stmtTag, game, currentGameText, gameHash, ecoCode, openingName)
		if err != nil {
			dbErr := fmt.Errorf("game %d (event: %s): %w", i+1, game.Tags["Event"], err)
			allErrors = append(allErrors, &PGNImportError{OriginalError: dbErr, PGNText: currentGameText})
			continue
		}

		// Extract and insert positions (if moves were parsed successfully)
		if game.Root != nil && game.Root.Next != nil {
			positions := ExtractPositions(game)
			if len(positions) > 0 {
				if err := insertPositions(ctx, tx, stmtPosition, gameID, positions, ecoCode, openingName); err != nil {
					db.logger.Warn("failed to insert positions for game",
						"game_id", gameID, "event", game.Tags["Event"], "error", err)
					// Don't fail the import if position storage fails
				} else {
					db.logger.Debug("positions inserted", "game_id", gameID, "count", len(positions))
				}
			}
		}

		importedCount++
	}

	// Commit transaction
	db.logger.Debug("committing transaction", "importedGames", importedCount, "errors", len(allErrors))
	err = tx.Commit()
	if err != nil {
		db.logger.Error("failed to commit transaction", "error", err, "importedGames", importedCount)
		_ = tx.Rollback()
		allErrors = append(allErrors, &PGNImportError{OriginalError: fmt.Errorf("failed to commit transaction: %w", err), PGNText: ""})
		return importedCount, allErrors
	}

	db.logger.Info("PGN import completed", "file", filePath, "imported", importedCount, "errors", len(allErrors))
	return importedCount, allErrors
}

// GetGameCount returns the total number of games in the database
func (db *DB) GetGameCount(ctx context.Context) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM games").Scan(&count)
	if err != nil {
		db.logger.Error("failed to get game count", "error", err)
		return 0, fmt.Errorf("failed to get game count: %w", err)
	}
	db.logger.Debug("game count retrieved", "count", count)
	return count, nil
}

// SearchGames searches for games matching the specified criteria
func (db *DB) SearchGames(ctx context.Context, criteria map[string]string, limit, offset int) ([]map[string]interface{}, error) {
	db.logger.Debug("searching games", "criteria", criteria, "limit", limit, "offset", offset)

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
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		db.logger.Error("failed to execute search query", "error", err)
		return nil, fmt.Errorf("failed to search games: %w", err)
	}
	defer func() { _ = rows.Close() }()

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
		db.logger.Error("error iterating search results", "error", err)
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	db.logger.Debug("search completed", "resultsFound", len(games))
	return games, nil
}

// GetGameByID retrieves a game by its ID
func (db *DB) GetGameByID(ctx context.Context, id int) (map[string]interface{}, error) {
	// Query the game
	row := db.conn.QueryRowContext(ctx, "SELECT * FROM games WHERE id = ?", id)

	var gameID int
	var event, site, date, round, white, black, result string
	var whiteElo, blackElo int
	var timeControl, pgnText, gameHash string
	var createdAt string
	var ecoCode, openingName sql.NullString

	err := row.Scan(
		&gameID, &event, &site, &date, &round, &white, &black, &result,
		&whiteElo, &blackElo, &timeControl, &pgnText, &createdAt, &gameHash, &ecoCode, &openingName,
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

	// Add ECO fields if available
	if ecoCode.Valid {
		game["eco_code"] = ecoCode.String
	}
	if openingName.Valid {
		game["opening_name"] = openingName.String
	}
	
	// Get all tags
	rows, err := db.conn.QueryContext(ctx, "SELECT tag_name, tag_value FROM tags WHERE game_id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

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
func (db *DB) ClearGames(ctx context.Context) error {
	// Begin transaction
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := recover(); err != nil {
			_ = tx.Rollback()
			panic(err)
		}
	}()

	// Delete all child tables first (due to foreign key constraints)
	_, err = tx.Exec("DELETE FROM tags")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete tags: %w", err)
	}

	_, err = tx.Exec("DELETE FROM positions")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete positions: %w", err)
	}

	// Delete all games
	_, err = tx.Exec("DELETE FROM games")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete games: %w", err)
	}

	// Reset the auto-increment counters
	_, err = tx.Exec("DELETE FROM sqlite_sequence WHERE name='games' OR name='tags' OR name='positions'")
	if err != nil {
		_ = tx.Rollback()
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
	Name          string  // Player's name
	Games         int     // Total games played
	Wins          int     // Total wins
	Losses        int     // Total losses
	Draws         int     // Total draws
	WinRate       float64 // Win rate as a percentage
	WhiteGames    int     // Games played as white
	BlackGames    int     // Games played as black
	WhiteWins     int     // Wins as white
	BlackWins     int     // Wins as black
	WhiteLosses   int     // Losses as white
	BlackLosses   int     // Losses as black
	WhiteDraws    int     // Draws as white
	BlackDraws    int     // Draws as black
	WhiteWinRate  float64 // Win rate as white (0-100)
	BlackWinRate  float64 // Win rate as black (0-100)
	BulletGames   int     // Games in bullet time control
	BlitzGames    int     // Games in blitz time control
	RapidGames    int     // Games in rapid time control
	ClassicalGames int    // Games in classical/daily time control
}

// OpeningStats represents statistics for a chess opening
type OpeningStats struct {
	ECOCode      string  // ECO code (e.g., "C50")
	OpeningName  string  // Opening name (e.g., "Italian Game")
	Games        int     // Total games with this opening
	Wins         int     // Wins with this opening
	Losses       int     // Losses with this opening
	Draws        int     // Draws with this opening
	WinRate      float64 // Win rate as a percentage (0-100)
	WhiteGames   int     // Games where player was white
	BlackGames   int     // Games where player was black
	WhiteWins    int     // Wins as white
	BlackWins    int     // Wins as black
	WhiteWinRate float64 // Win rate as white (0-100)
	BlackWinRate float64 // Win rate as black (0-100)
}

// categorizeTimeControl categorizes a time control string into bullet/blitz/rapid/classical
func categorizeTimeControl(tc string) string {
	if tc == "" {
		return "unknown"
	}

	// Parse time control formats like "180+0", "600+5", "1/86400", etc.
	// For Chess.com/Lichess, typical formats are:
	// - Bullet: < 3 minutes (180 seconds)
	// - Blitz: 3-10 minutes (180-600 seconds)
	// - Rapid: 10-60 minutes (600-3600 seconds)
	// - Classical: > 60 minutes or daily/correspondence

	// Handle daily/correspondence formats (like "1/86400")
	if strings.Contains(tc, "/") {
		return "classical"
	}

	// Parse base time in seconds
	parts := strings.Split(tc, "+")
	if len(parts) == 0 {
		return "unknown"
	}

	baseTime, err := strconv.Atoi(parts[0])
	if err != nil {
		return "unknown"
	}

	// Categorize based on base time
	if baseTime < 180 {
		return "bullet"
	} else if baseTime < 600 {
		return "blitz"
	} else if baseTime < 3600 {
		return "rapid"
	}
	return "classical"
}

// GetPlayerStats retrieves statistics for all players in the database
func (db *DB) GetPlayerStats(ctx context.Context) ([]PlayerStats, error) {
	return db.GetPlayerStatsFiltered(ctx, nil)
}

// GetPlayerStatsFiltered retrieves statistics for specific players in the database
// If players is nil or empty, returns stats for all players
func (db *DB) GetPlayerStatsFiltered(ctx context.Context, players []string) ([]PlayerStats, error) {
	// Build query based on whether we're filtering
	var query string
	var args []interface{}

	if len(players) == 0 {
		// Query all games
		query = `
			SELECT white, black, result, time_control
			FROM games
			WHERE white != '' AND black != ''
		`
	} else {
		// Query games for specific players
		placeholders := make([]string, len(players))
		args = make([]interface{}, len(players))
		for i, player := range players {
			placeholders[i] = "?"
			args[i] = player
		}
		playerList := strings.Join(placeholders, ",")
		query = fmt.Sprintf(`
			SELECT white, black, result, time_control
			FROM games
			WHERE (white IN (%s) OR black IN (%s))
			AND white != '' AND black != ''
		`, playerList, playerList)
		args = append(args, args...) // Duplicate args for both IN clauses
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Map to track player statistics
	playerStats := make(map[string]*PlayerStats)

	// Create a set of filtered players for quick lookup
	filterSet := make(map[string]bool)
	for _, player := range players {
		filterSet[player] = true
	}
	isFiltered := len(players) > 0

	// Process each game
	for rows.Next() {
		var white, black, result string
		var timeControl sql.NullString
		if err := rows.Scan(&white, &black, &result, &timeControl); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Categorize time control
		tc := ""
		if timeControl.Valid {
			tc = categorizeTimeControl(timeControl.String)
		}

		// When filtering, only track stats for the filtered players
		trackWhite := !isFiltered || filterSet[white]
		trackBlack := !isFiltered || filterSet[black]

		// Initialize player stats if not yet tracked and should be tracked
		if trackWhite {
			if _, ok := playerStats[white]; !ok {
				playerStats[white] = &PlayerStats{Name: white}
			}
		}
		if trackBlack {
			if _, ok := playerStats[black]; !ok {
				playerStats[black] = &PlayerStats{Name: black}
			}
		}

		// Update game counts for tracked players
		if trackWhite {
			playerStats[white].Games++
			playerStats[white].WhiteGames++
			// Update time control counts
			switch tc {
			case "bullet":
				playerStats[white].BulletGames++
			case "blitz":
				playerStats[white].BlitzGames++
			case "rapid":
				playerStats[white].RapidGames++
			case "classical":
				playerStats[white].ClassicalGames++
			}
		}
		if trackBlack {
			playerStats[black].Games++
			playerStats[black].BlackGames++
			// Update time control counts
			switch tc {
			case "bullet":
				playerStats[black].BulletGames++
			case "blitz":
				playerStats[black].BlitzGames++
			case "rapid":
				playerStats[black].RapidGames++
			case "classical":
				playerStats[black].ClassicalGames++
			}
		}

		// Update win/loss/draw counts based on result
		switch result {
		case "1-0": // White win
			if trackWhite {
				playerStats[white].Wins++
				playerStats[white].WhiteWins++
			}
			if trackBlack {
				playerStats[black].Losses++
				playerStats[black].BlackLosses++
			}
		case "0-1": // Black win
			if trackBlack {
				playerStats[black].Wins++
				playerStats[black].BlackWins++
			}
			if trackWhite {
				playerStats[white].Losses++
				playerStats[white].WhiteLosses++
			}
		case "1/2-1/2": // Draw
			if trackWhite {
				playerStats[white].Draws++
				playerStats[white].WhiteDraws++
			}
			if trackBlack {
				playerStats[black].Draws++
				playerStats[black].BlackDraws++
			}
		default: // Unknown result or ongoing game
			// Skip updating win/loss/draw counts
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Calculate win rates and convert to slice
	results := make([]PlayerStats, 0, len(playerStats))
	for _, stats := range playerStats {
		// Calculate overall win rate
		if stats.Games > 0 {
			stats.WinRate = float64(stats.Wins) / float64(stats.Games) * 100.0
		}

		// Calculate win rate as white
		if stats.WhiteGames > 0 {
			stats.WhiteWinRate = float64(stats.WhiteWins) / float64(stats.WhiteGames) * 100.0
		}

		// Calculate win rate as black
		if stats.BlackGames > 0 {
			stats.BlackWinRate = float64(stats.BlackWins) / float64(stats.BlackGames) * 100.0
		}

		results = append(results, *stats)
	}

	// Sort by number of games (most active players first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Games > results[j].Games
	})

	return results, nil
}

// GetOpeningStats retrieves statistics for all openings in the database
func (db *DB) GetOpeningStats(ctx context.Context) ([]OpeningStats, error) {
	return db.GetOpeningStatsFiltered(ctx, nil)
}

// GetOpeningStatsFiltered retrieves statistics for specific players' games with openings
// If players is nil or empty, returns stats for all players
func (db *DB) GetOpeningStatsFiltered(ctx context.Context, players []string) ([]OpeningStats, error) {
	// Build query based on whether we're filtering by players
	var query string
	var args []interface{}

	if len(players) == 0 {
		// Query all games with ECO codes
		query = `
			SELECT eco_code, opening_name, white, black, result
			FROM games
			WHERE eco_code IS NOT NULL AND eco_code != ''
			AND white != '' AND black != ''
		`
	} else {
		// Query games for specific players
		placeholders := make([]string, len(players))
		args = make([]interface{}, len(players))
		for i, player := range players {
			placeholders[i] = "?"
			args[i] = player
		}
		playerList := strings.Join(placeholders, ",")
		query = fmt.Sprintf(`
			SELECT eco_code, opening_name, white, black, result
			FROM games
			WHERE (white IN (%s) OR black IN (%s))
			AND eco_code IS NOT NULL AND eco_code != ''
			AND white != '' AND black != ''
		`, playerList, playerList)
		args = append(args, args...) // Duplicate args for both IN clauses
	}

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query games: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Map to track opening statistics (keyed by ECO code)
	openingStats := make(map[string]*OpeningStats)

	// Create a set of filtered players for quick lookup
	filterSet := make(map[string]bool)
	for _, player := range players {
		filterSet[player] = true
	}
	isFiltered := len(players) > 0

	// Process each game
	for rows.Next() {
		var ecoCode, openingName, white, black, result string
		if err := rows.Scan(&ecoCode, &openingName, &white, &black, &result); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Initialize opening stats if not yet tracked
		if _, ok := openingStats[ecoCode]; !ok {
			openingStats[ecoCode] = &OpeningStats{
				ECOCode:     ecoCode,
				OpeningName: openingName,
			}
		}

		stats := openingStats[ecoCode]

		// When filtering, only count games where the filtered player(s) participated
		whiteIsTracked := !isFiltered || filterSet[white]
		blackIsTracked := !isFiltered || filterSet[black]

		// Skip if neither player is being tracked (when filtering)
		if isFiltered && !whiteIsTracked && !blackIsTracked {
			continue
		}

		// When filtering, we track stats separately for white and black
		// For non-filtered (all games), both flags are true

		// Update game counts
		// If both players are tracked, count the game twice (once for each)
		// If only one is tracked, count it once
		if whiteIsTracked {
			stats.Games++
			stats.WhiteGames++
		}
		if blackIsTracked && (!whiteIsTracked || isFiltered) {
			// Don't double-count games in non-filtered mode
			if !whiteIsTracked {
				stats.Games++
			}
			stats.BlackGames++
		}

		// Update win/loss/draw counts based on result and player color
		switch result {
		case "1-0": // White win
			if whiteIsTracked {
				stats.Wins++
				stats.WhiteWins++
			}
			if blackIsTracked && !whiteIsTracked {
				stats.Losses++
			}
		case "0-1": // Black win
			if blackIsTracked {
				stats.Wins++
				stats.BlackWins++
			}
			if whiteIsTracked && !blackIsTracked {
				stats.Losses++
			}
		case "1/2-1/2": // Draw
			if whiteIsTracked {
				stats.Draws++
			}
			if blackIsTracked && !whiteIsTracked {
				stats.Draws++
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Calculate win rates and convert to slice
	results := make([]OpeningStats, 0, len(openingStats))
	for _, stats := range openingStats {
		// Calculate overall win rate
		if stats.Games > 0 {
			stats.WinRate = float64(stats.Wins) / float64(stats.Games) * 100.0
		}

		// Calculate win rate as white
		if stats.WhiteGames > 0 {
			stats.WhiteWinRate = float64(stats.WhiteWins) / float64(stats.WhiteGames) * 100.0
		}

		// Calculate win rate as black
		if stats.BlackGames > 0 {
			stats.BlackWinRate = float64(stats.BlackWins) / float64(stats.BlackGames) * 100.0
		}

		results = append(results, *stats)
	}

	// Sort by number of games (most common openings first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Games > results[j].Games
	})

	return results, nil
}

// PositionFrequency represents a position and how often it occurs
type PositionFrequency struct {
	FEN            string  // The position in FEN notation
	Count          int     // Number of times this position appears in the database
	WhiteWins      int     // Number of games where white won from this position
	BlackWins      int     // Number of games where black won from this position
	Draws          int     // Number of games that were drawn from this position
	WhiteWinPct    float64 // Percentage of games white won (0-100)
	BlackWinPct    float64 // Percentage of games black won (0-100)
	DrawPct        float64 // Percentage of games drawn (0-100)
	ECOCode        string  // Most common ECO code for this position (if available)
	OpeningName    string  // Most common opening name for this position (if available)
}

// GetPositionStats retrieves statistics about positions in the database
// Only includes positions reached after move 10 (move_number >= 20 half-moves)
func (db *DB) GetPositionStats(ctx context.Context) (uniqueCount int, topPositions []PositionFrequency, err error) {
	// Get count of unique positions (all moves)
	err = db.conn.QueryRowContext(ctx, "SELECT COUNT(DISTINCT fen) FROM positions").Scan(&uniqueCount)
	if err != nil {
		db.logger.Error("failed to get unique position count", "error", err)
		return 0, nil, fmt.Errorf("failed to get unique position count: %w", err)
	}

	// Get top 10 most common positions (after move 10) with win statistics and ECO codes
	// For each position, we get the most common ECO code (MODE) that appears with it
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			p.fen,
			COUNT(*) as frequency,
			SUM(CASE WHEN g.result = '1-0' THEN 1 ELSE 0 END) as white_wins,
			SUM(CASE WHEN g.result = '0-1' THEN 1 ELSE 0 END) as black_wins,
			SUM(CASE WHEN g.result = '1/2-1/2' THEN 1 ELSE 0 END) as draws,
			p.eco_code,
			p.opening_name
		FROM positions p
		JOIN games g ON p.game_id = g.id
		WHERE p.move_number >= 20
		GROUP BY p.fen
		ORDER BY frequency DESC
		LIMIT 10
	`)
	if err != nil {
		db.logger.Error("failed to query top positions", "error", err)
		return 0, nil, fmt.Errorf("failed to query top positions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	topPositions = make([]PositionFrequency, 0, 10)
	for rows.Next() {
		var pos PositionFrequency
		var ecoCode, openingName sql.NullString
		if err := rows.Scan(&pos.FEN, &pos.Count, &pos.WhiteWins, &pos.BlackWins, &pos.Draws, &ecoCode, &openingName); err != nil {
			return 0, nil, fmt.Errorf("failed to scan position row: %w", err)
		}

		// Set ECO code and opening name if available
		if ecoCode.Valid {
			pos.ECOCode = ecoCode.String
		}
		if openingName.Valid {
			pos.OpeningName = openingName.String
		}

		// Calculate percentages
		if pos.Count > 0 {
			pos.WhiteWinPct = float64(pos.WhiteWins) / float64(pos.Count) * 100
			pos.BlackWinPct = float64(pos.BlackWins) / float64(pos.Count) * 100
			pos.DrawPct = float64(pos.Draws) / float64(pos.Count) * 100
		}

		topPositions = append(topPositions, pos)
	}

	if err := rows.Err(); err != nil {
		db.logger.Error("error iterating position rows", "error", err)
		return 0, nil, fmt.Errorf("error iterating position rows: %w", err)
	}

	db.logger.Debug("position stats retrieved", "uniqueCount", uniqueCount, "topPositionsCount", len(topPositions))
	return uniqueCount, topPositions, nil
}

// GamePosition represents a position within a game, with game metadata.
type GamePosition struct {
	PositionID int
	GameID     int
	MoveNumber int
	FEN        string
	NextMove   string
	Evaluation *float64
	White      string
	Black      string
	Event      string
	Date       string
}

// GetPositionByGameAndMove retrieves a single position for a game at a specific ply.
func (db *DB) GetPositionByGameAndMove(ctx context.Context, gameID, moveNumber int) (*GamePosition, error) {
	row := db.conn.QueryRowContext(ctx, `
		SELECT p.id, p.game_id, p.move_number, p.fen, p.next_move, p.evaluation,
		       g.white, g.black, g.event, g.date
		FROM positions p
		JOIN games g ON p.game_id = g.id
		WHERE p.game_id = ? AND p.move_number = ?
	`, gameID, moveNumber)

	var gp GamePosition
	var nextMove sql.NullString
	var evaluation sql.NullFloat64
	err := row.Scan(
		&gp.PositionID, &gp.GameID, &gp.MoveNumber, &gp.FEN,
		&nextMove, &evaluation,
		&gp.White, &gp.Black, &gp.Event, &gp.Date,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("position not found: game %d, move %d", gameID, moveNumber)
		}
		return nil, fmt.Errorf("failed to get position: %w", err)
	}
	if nextMove.Valid {
		gp.NextMove = nextMove.String
	}
	if evaluation.Valid {
		v := evaluation.Float64
		gp.Evaluation = &v
	}
	return &gp, nil
}

// GetPositionsForGame retrieves all positions for a game, ordered by move number.
func (db *DB) GetPositionsForGame(ctx context.Context, gameID int) ([]GamePosition, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT p.id, p.game_id, p.move_number, p.fen, p.next_move, p.evaluation,
		       g.white, g.black, g.event, g.date
		FROM positions p
		JOIN games g ON p.game_id = g.id
		WHERE p.game_id = ?
		ORDER BY p.move_number
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var positions []GamePosition
	for rows.Next() {
		var gp GamePosition
		var nextMove sql.NullString
		var evaluation sql.NullFloat64
		if err := rows.Scan(
			&gp.PositionID, &gp.GameID, &gp.MoveNumber, &gp.FEN,
			&nextMove, &evaluation,
			&gp.White, &gp.Black, &gp.Event, &gp.Date,
		); err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		if nextMove.Valid {
			gp.NextMove = nextMove.String
		}
		if evaluation.Valid {
			v := evaluation.Float64
			gp.Evaluation = &v
		}
		positions = append(positions, gp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating positions: %w", err)
	}
	return positions, nil
}

// UpdatePositionEvaluation updates the evaluation column for a position.
func (db *DB) UpdatePositionEvaluation(ctx context.Context, positionID int, evaluation float64) error {
	result, err := db.conn.ExecContext(ctx, `
		UPDATE positions SET evaluation = ? WHERE id = ?
	`, evaluation, positionID)
	if err != nil {
		return fmt.Errorf("failed to update evaluation: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("position not found: %d", positionID)
	}
	return nil
}
