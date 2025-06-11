package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	defaultDBPath = "~/.gochess/games.db"
)

// expandPath expands the tilde in file paths to the user's home directory
func expandPath(path string) string {
	if path == "" {
		return ""
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// ImportCommand imports PGN files to the SQLite database
func ImportCommand(c *cli.Context) error {
	pgnPath := c.String("pgn")
	dbPath := expandPath(c.String("database"))

	// Check if PGN file exists
	fileInfo, err := os.Stat(pgnPath)
	if err != nil {
		return fmt.Errorf("error accessing PGN file: %w", err)
	}

	// Open database connection
	fmt.Printf("Opening database at %s...\n", dbPath)
	db, err := New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get start count
	startCount, err := db.GetGameCount()
	if err != nil {
		return fmt.Errorf("failed to get initial game count: %w", err)
	}

	// Import games
	startTime := time.Now()

	if fileInfo.IsDir() {
		// If input is a directory, import all PGN files in it
		fmt.Printf("Importing all PGN files from directory: %s\n", pgnPath)
		
		var totalImported int
		var allErrors []error
		
		err := filepath.Walk(pgnPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip directories and non-PGN files
			if info.IsDir() || filepath.Ext(path) != ".pgn" {
				return nil
			}
			
			fmt.Printf("Importing file: %s\n", path)
			imported, errors := db.ImportPGN(path)
			totalImported += imported
			allErrors = append(allErrors, errors...)
			
			fmt.Printf("  Imported %d games\n", imported)
			return nil
		})
		
		if err != nil {
			return fmt.Errorf("error walking directory: %w", err)
		}
		
		// Report import errors
		if len(allErrors) > 0 {
			fmt.Printf("Encountered %d errors during import\n", len(allErrors))
			if c.Bool("verbose") {
				for _, errInstance := range allErrors {
					if pgnErr, ok := errInstance.(*PGNImportError); ok && pgnErr.PGNText != "" {
						fmt.Printf("  - Error in PGN:\n%s\n  - Message: %v\n", pgnErr.PGNText, pgnErr.OriginalError)
					} else {
						fmt.Printf("  - %v\n", errInstance)
					}
				}
			}
		}
		
		fmt.Printf("Total games imported: %d\n", totalImported)
	} else {
		// Import single file
		fmt.Printf("Importing PGN file: %s\n", pgnPath)
		imported, errors := db.ImportPGN(pgnPath)
		
		// Report import errors
		if len(errors) > 0 {
			fmt.Printf("Encountered %d errors during import\n", len(errors))
			
			// Always show a summary of error types
			errorCounts := make(map[string]int)
			for _, err := range errors {
				errorType := "Unknown error"
				
				// Extract the error type from the error message
				errStr := err.Error()
				if len(errStr) > 0 {
					// Get the first part of the error message (up to the colon or end)
					colonIdx := strings.Index(errStr, ":")
					if colonIdx > 0 {
						errorType = errStr[:colonIdx]
					} else {
						errorType = errStr
					}
				}
				
				errorCounts[errorType]++
			}
			
			// Print error type summary
			fmt.Println("Error summary:")
			for errorType, count := range errorCounts {
				fmt.Printf("  - %s: %d occurrences\n", errorType, count)
			}
			
			// Print detailed errors if verbose
			if c.Bool("verbose") {
				fmt.Println("\nDetailed errors:")
				for i, errInstance := range errors {
					// Only show first 10 detailed errors if there are many
					if i >= 10 && len(errors) > 12 {
						fmt.Printf("  ... and %d more errors (use --verbose for full details)\n", len(errors)-10)
						break
					}
					if pgnErr, ok := errInstance.(*PGNImportError); ok && pgnErr.PGNText != "" {
						fmt.Printf("  %d. Error in PGN:\n%s\n  Message: %v\n", i+1, pgnErr.PGNText, pgnErr.OriginalError)
					} else {
						fmt.Printf("  %d. %v\n", i+1, errInstance)
					}
				}
			} else if len(errors) <= 5 {
				// If there are only a few errors, show them even without verbose
				fmt.Println("\nErrors:")
				for i, errInstance := range errors {
					if pgnErr, ok := errInstance.(*PGNImportError); ok && pgnErr.PGNText != "" {
						fmt.Printf("  %d. Error in PGN:\n%s\n  Message: %v\n", i+1, pgnErr.PGNText, pgnErr.OriginalError)
					} else {
						fmt.Printf("  %d. %v\n", i+1, errInstance)
					}
				}
			} else {
				// Just show the first 3 errors
				fmt.Println("\nFirst few errors:")
				for i := 0; i < 3 && i < len(errors); i++ {
					errInstance := errors[i]
					if pgnErr, ok := errInstance.(*PGNImportError); ok && pgnErr.PGNText != "" {
						fmt.Printf("  %d. Error in PGN:\n%s\n  Message: %v\n", i+1, pgnErr.PGNText, pgnErr.OriginalError)
					} else {
						fmt.Printf("  %d. %v\n", i+1, errInstance)
					}
				}
				fmt.Println("  Use --verbose flag to see all errors")
			}
		}
		
		fmt.Printf("Games imported: %d\n", imported)
	}

	// Get end count
	endCount, err := db.GetGameCount()
	if err != nil {
		return fmt.Errorf("failed to get final game count: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Import completed in %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Database now contains %d games (added %d new games)\n", 
		endCount, endCount-startCount)

	return nil
}

// ListCommand lists games in the database
func ListCommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	limit := c.Int("limit")
	offset := c.Int("offset")
	
	// Prepare search criteria
	criteria := make(map[string]string)
	if white := c.String("white"); white != "" {
		criteria["white"] = white
	}
	if black := c.String("black"); black != "" {
		criteria["black"] = black
	}
	if event := c.String("event"); event != "" {
		criteria["event"] = event
	}
	if date := c.String("date"); date != "" {
		criteria["date"] = date
	}
	
	// Open database connection
	db, err := New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Get total count
	count, err := db.GetGameCount()
	if err != nil {
		return fmt.Errorf("failed to get game count: %w", err)
	}
	
	// Search games
	games, err := db.SearchGames(criteria, limit, offset)
	if err != nil {
		return fmt.Errorf("failed to search games: %w", err)
	}
	
	// Display results
	fmt.Printf("Database contains %d total games\n", count)
	fmt.Printf("Showing games %d to %d of matched results:\n\n", offset+1, offset+len(games))
	
	for i, game := range games {
		fmt.Printf("%d. [%s] %s vs %s (%s) - %s\n", 
			i+offset+1, 
			game["date"], 
			game["white"], 
			game["black"],
			game["event"],
			game["result"])
	}
	
	if len(games) == limit {
		fmt.Printf("\nMore results available. Use --offset %d to see the next page.\n", offset+limit)
	}
	
	return nil
}

// ShowCommand shows details of a specific game
func ShowCommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	id := c.Int("id")
	
	// Open database connection
	db, err := New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Get the game
	game, err := db.GetGameByID(id)
	if err != nil {
		return fmt.Errorf("failed to get game: %w", err)
	}
	
	// Display game details
	fmt.Printf("Game #%d\n", id)
	fmt.Printf("Event: %s\n", game["event"])
	fmt.Printf("Site: %s\n", game["site"])
	fmt.Printf("Date: %s\n", game["date"])
	fmt.Printf("White: %s (%d)\n", game["white"], game["white_elo"])
	fmt.Printf("Black: %s (%d)\n", game["black"], game["black_elo"])
	fmt.Printf("Result: %s\n", game["result"])
	fmt.Printf("Time Control: %s\n", game["time_control"])
	
	// Show all tags
	fmt.Printf("\nAll Tags:\n")
	tags := game["tags"].(map[string]string)
	for name, value := range tags {
		fmt.Printf("  %s: %s\n", name, value)
	}
	
	// Show PGN
	if c.Bool("pgn") {
		fmt.Printf("\nPGN:\n%s\n", game["pgn_text"])
	}
	
	return nil
}

// ExportCommand exports games to PGN format
func ExportCommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	output := c.String("output")
	id := c.Int("id")
	
	// Open database connection
	db, err := New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Set up output writer
	var outputWriter *os.File
	if output == "" {
		outputWriter = os.Stdout
	} else {
		var err error
		outputWriter, err = os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputWriter.Close()
	}
	
	// Export specific game or all games
	if id > 0 {
		// Export single game
		game, err := db.GetGameByID(id)
		if err != nil {
			return fmt.Errorf("failed to get game: %w", err)
		}
		
		pgnText := game["pgn_text"].(string)
		fmt.Fprintln(outputWriter, pgnText)
		
		if output != "" {
			fmt.Printf("Exported game #%d to %s\n", id, output)
		}
	} else {
		// TODO: Implement exporting all games or filtered games
		return fmt.Errorf("exporting all games not yet implemented - please specify a game ID")
	}
	
	return nil
}

// ClearCommand clears all games from the database
func ClearCommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	
	// Ask for confirmation unless --force is specified
	if !c.Bool("force") {
		fmt.Printf("WARNING: This will delete ALL games from the database at %s\n", dbPath)
		fmt.Print("Are you sure you want to continue? [y/N]: ")
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return nil
		}
	}
	
	// Open database connection
	fmt.Printf("Opening database at %s...\n", dbPath)
	db, err := New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Get initial count
	count, err := db.GetGameCount()
	if err != nil {
		return fmt.Errorf("failed to get initial game count: %w", err)
	}
	
	if count == 0 {
		fmt.Println("Database is already empty")
		return nil
	}
	
	// Clear all games
	fmt.Println("Clearing all games from database...")
	if err := db.ClearGames(); err != nil {
		return fmt.Errorf("failed to clear games: %w", err)
	}
	
	fmt.Printf("Successfully deleted %d games from the database\n", count)
	return nil
}

// StatsCommand shows statistics for players in the database
func StatsCommand(c *cli.Context) error {
	dbPath := expandPath(c.String("database"))
	player := c.String("player")
	format := c.String("format")
	
	// Open database connection
	fmt.Printf("Opening database at %s...\n", dbPath)
	db, err := New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Get game count
	count, err := db.GetGameCount()
	if err != nil {
		return fmt.Errorf("failed to get game count: %w", err)
	}
	
	if count == 0 {
		fmt.Println("Database is empty")
		return nil
	}
	
	// Get player statistics
	fmt.Println("Calculating player statistics...")
	stats, err := db.GetPlayerStats()
	if err != nil {
		return fmt.Errorf("failed to get player statistics: %w", err)
	}
	
	// Filter by player if specified
	if player != "" {
		var filteredStats []PlayerStats
		playerLower := strings.ToLower(player)
		
		for _, s := range stats {
			if strings.Contains(strings.ToLower(s.Name), playerLower) {
				filteredStats = append(filteredStats, s)
			}
		}
		
		stats = filteredStats
		
		if len(stats) == 0 {
			fmt.Printf("No players found matching '%s'\n", player)
			return nil
		}
	}
	
	// Display statistics
	fmt.Printf("Database contains %d games with %d players\n\n", count, len(stats))
	
	switch format {
	case "csv":
		// CSV output
		fmt.Println("Name,Games,Wins,Losses,Draws,WinRate,WhiteGames,BlackGames,WhiteWins,BlackWins")
		for _, s := range stats {
			fmt.Printf("%s,%d,%d,%d,%d,%.1f%%,%d,%d,%d,%d\n",
				s.Name, s.Games, s.Wins, s.Losses, s.Draws, s.WinRate,
				s.WhiteGames, s.BlackGames, s.WhiteWins, s.BlackWins)
		}
	
	default:
		// Table output (default)
		fmt.Printf("%-20s %-6s %-6s %-6s %-6s %-8s %-6s %-6s\n",
			"PLAYER", "GAMES", "WINS", "LOSSES", "DRAWS", "WIN RATE", "WHITE", "BLACK")
		fmt.Println(strings.Repeat("-", 72))
		
		for _, s := range stats {
			// Truncate long player names
			name := s.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}
			
			fmt.Printf("%-20s %-6d %-6d %-6d %-6d %-7.1f%% %-6d %-6d\n",
				name, s.Games, s.Wins, s.Losses, s.Draws, s.WinRate,
				s.WhiteGames, s.BlackGames)
		}
		
		// Show color statistics
		if len(stats) > 0 && player != "" {
			fmt.Printf("\nDetailed statistics for %s:\n", stats[0].Name)
			fmt.Printf("  As White: %d games, %d wins (%.1f%%)\n", 
				stats[0].WhiteGames, stats[0].WhiteWins, 
				safeDiv(float64(stats[0].WhiteWins), float64(stats[0].WhiteGames))*100)
			fmt.Printf("  As Black: %d games, %d wins (%.1f%%)\n", 
				stats[0].BlackGames, stats[0].BlackWins,
				safeDiv(float64(stats[0].BlackWins), float64(stats[0].BlackGames))*100)
		}
	}
	
	return nil
}

// safeDiv performs division but handles division by zero gracefully
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
