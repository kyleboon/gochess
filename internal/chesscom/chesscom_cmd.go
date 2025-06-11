package chesscom

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyleboon/gochess/internal/db"
	"github.com/urfave/cli/v2"
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

// ListArchives lists available archives for a Chess.com user
func ListArchives(c *cli.Context) error {
	username := c.String("username")
	client := NewClient()

	fmt.Printf("Fetching available archives for %s...\n", username)

	archives, err := client.GetArchivedMonths(username)
	if err != nil {
		return fmt.Errorf("failed to fetch archives: %w", err)
	}

	fmt.Printf("Available archives for %s:\n", username)
	for _, archive := range archives.Archives {
		fmt.Println(archive)
	}

	return nil
}

// downloadAndImportMonthlyGames downloads and optionally imports games for a specific month
// If externalDB is provided, it will be used instead of opening a new connection
func downloadAndImportMonthlyGames(username string, year, month int, format, output, dbPath string, importDB, verbose bool, externalDB *db.DB) (int, error) {
	client := NewClient()
	fmt.Printf("Fetching games for %s (%d/%02d)...\n", username, year, month)
	
	// Get the PGN data
	pgn, err := client.GetPlayerGamesPGN(username, year, month)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch PGN for %d/%02d: %w", year, month, err)
	}
	
	// If we're importing to DB
	if importDB {
		// Create a temporary file to store the PGN for import
		tmpfile, err := os.CreateTemp("", "chesscom-*.pgn")
		if err != nil {
			return 0, fmt.Errorf("failed to create temporary file: %w", err)
		}
		tmpPath := tmpfile.Name()
		defer os.Remove(tmpPath) // Clean up

		// Write PGN to temporary file
		if _, err := tmpfile.WriteString(pgn); err != nil {
			tmpfile.Close()
			return 0, fmt.Errorf("failed to write to temporary file: %w", err)
		}
		tmpfile.Close()

		// Use the external database if provided, otherwise open a new connection
		var database *db.DB
		var dbOpened bool
		
		if externalDB != nil {
			// Use the provided database connection
			database = externalDB
		} else if dbPath != "" {
			// Open a new database connection
			database, err = db.New(dbPath)
			if err != nil {
				return 0, fmt.Errorf("failed to open database: %w", err)
			}
			dbOpened = true
			defer func() {
				if dbOpened {
					database.Close()
				}
			}()
		} else {
			return 0, fmt.Errorf("no database connection available")
		}

		// Import the PGN file
		count, errors := database.ImportPGN(tmpPath)

		// Print import results
		if len(errors) > 0 && verbose {
			fmt.Printf("Encountered %d errors during import of %d/%02d:\n", len(errors), year, month)
			for i, err := range errors {
				fmt.Printf("  Error %d: %s\n", i+1, err)
				if pgnErr, ok := err.(*db.PGNImportError); ok && pgnErr.PGNText != "" {
					fmt.Printf("    PGN: %s\n", pgnErr.PGNText)
				}
			}
		} else if len(errors) > 0 {
			fmt.Printf("Encountered %d errors during import of %d/%02d. Use --verbose to see details.\n", len(errors), year, month)
		}

		// Return the count of imported games
		return count, nil
	}

	// Handle non-import output
	if output != "" {
		// If we're writing to a file and not importing, create a month-specific file
		monthlyOutput := output
		if strings.Contains(output, "*") {
			// Replace * with year-month
			monthlyOutput = strings.ReplaceAll(output, "*", fmt.Sprintf("%d-%02d", year, month))
		}
		
		outputFile, err := os.Create(monthlyOutput)
		if err != nil {
			return 0, fmt.Errorf("failed to create output file %s: %w", monthlyOutput, err)
		}
		defer outputFile.Close()
		
		// Write the PGN data
		_, err = outputFile.WriteString(pgn)
		if err != nil {
			return 0, fmt.Errorf("failed to write to output file: %w", err)
		}
		
		fmt.Printf("Downloaded PGN games for %s (%d/%02d) to %s\n", username, year, month, monthlyOutput)
	}
	
	return 0, nil
}

// DownloadGames downloads games for a Chess.com user
func DownloadGames(c *cli.Context) error {
	username := c.String("username")
	year := c.Int("year")
	month := c.Int("month")
	format := c.String("format")
	output := c.String("output")
	importDB := c.Bool("import-db")
	dbPath := c.String("database")
	verbose := c.Bool("verbose")
	allHistory := c.Bool("all-history")

	client := NewClient()

	// Expand database path
	dbPath = expandPath(dbPath)

	// Handle downloading all historical games
	if allHistory {
		// Fetch all available archives
		fmt.Printf("Fetching available archives for %s...\n", username)
		archives, err := client.GetArchivedMonths(username)
		if err != nil {
			return fmt.Errorf("failed to fetch archives: %w", err)
		}

		fmt.Printf("Found %d months of archives for %s\n", len(archives.Archives), username)
		
		// Open database once if importing
		var database *db.DB
		if importDB {
			fmt.Printf("Opening database at %s...\n", dbPath)
			database, err = db.New(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer database.Close()
		}
		
		totalGames := 0
		skippedMonths := 0
		
		// Process each archive
		for i, archiveURL := range archives.Archives {
			// Extract year and month from the URL
			// Format is https://api.chess.com/pub/player/{username}/games/{year}/{month}
			parts := strings.Split(archiveURL, "/")
			if len(parts) < 2 {
				continue
			}
			
			archiveYear, err := parseArchiveYear(parts[len(parts)-2])
			if err != nil {
				fmt.Printf("Warning: Could not parse year from archive URL %s: %v\n", archiveURL, err)
				skippedMonths++
				continue
			}
			
			archiveMonth, err := parseArchiveMonth(parts[len(parts)-1])
			if err != nil {
				fmt.Printf("Warning: Could not parse month from archive URL %s: %v\n", archiveURL, err)
				skippedMonths++
				continue
			}
			
			fmt.Printf("\nProcessing archive %d/%d: %d/%02d\n", i+1, len(archives.Archives), archiveYear, archiveMonth)
			
			// Download and process games for this month
			monthlyGames, err := downloadAndImportMonthlyGames(
				username, 
				archiveYear, 
				archiveMonth, 
				format, 
				output, 
				"", // Empty dbPath because we already opened the database
				importDB, 
				verbose,
				database, // Pass the database connection
			)
			
			if err != nil {
				fmt.Printf("Error processing %d/%02d: %v\n", archiveYear, archiveMonth, err)
				skippedMonths++
				continue
			}
			
			totalGames += monthlyGames
		}
		
		// Summary
		fmt.Printf("\n====== DOWNLOAD SUMMARY ======\n")
		fmt.Printf("Total archives processed: %d\n", len(archives.Archives) - skippedMonths)
		if skippedMonths > 0 {
			fmt.Printf("Skipped archives: %d\n", skippedMonths)
		}
		
		if importDB {
			fmt.Printf("Total games imported: %d\n", totalGames)
			
			// Get current game count in database
			currentCount, err := database.GetGameCount()
			if err == nil {
				fmt.Printf("Total games in database: %d\n", currentCount)
			}
		}
		
		return nil
	}
	
	// Handle regular single-month download
	if importDB {
		// Use our reusable function to handle the download and import
		count, err := downloadAndImportMonthlyGames(
			username,
			year,
			month,
			format,
			output,
			dbPath,
			importDB,
			verbose,
			nil, // No external DB for single-month case
		)
		
		if err != nil {
			return fmt.Errorf("failed to download and import games: %w", err)
		}
		
		// Print success message
		fmt.Printf("Successfully imported %d games from Chess.com\n", count)

		// If the user still wants to output to a file or stdout, we'll do that too
		if output != "" || format != "pgn" {
			// Continue with the normal download operation
			fmt.Println("\nAdditionally processing requested output format...")
		} else {
			// Otherwise we're done
			return nil
		}
	}

	// Handle output to file or stdout as before
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

	switch format {
	case "pgn":
		pgn, err := client.GetPlayerGamesPGN(username, year, month)
		if err != nil {
			return fmt.Errorf("failed to fetch PGN: %w", err)
		}

		fmt.Fprintln(outputWriter, pgn)

		if output != "" {
			fmt.Printf("Downloaded PGN games for %s (%d/%02d) to %s\n",
				username, year, month, output)
		}

	case "json":
		games, err := client.GetPlayerGames(username, year, month)
		if err != nil {
			return fmt.Errorf("failed to fetch games: %w", err)
		}

		// Just output the raw JSON for now
		for i, game := range games.Games {
			fmt.Fprintf(outputWriter, "Game %d:\n", i+1)
			fmt.Fprintf(outputWriter, "  URL: %s\n", game.URL)
			fmt.Fprintf(outputWriter, "  White: %s (Rating: %d)\n", game.White.Username, game.White.Rating)
			fmt.Fprintf(outputWriter, "  Black: %s (Rating: %d)\n", game.Black.Username, game.Black.Rating)
			fmt.Fprintf(outputWriter, "  Result: %s-%s\n", game.White.Result, game.Black.Result)
			fmt.Fprintf(outputWriter, "  Time Control: %s\n", game.TimeControl)
			fmt.Fprintf(outputWriter, "  End Time: %s\n", game.GetEndTime().Format(time.RFC3339))
			fmt.Fprintf(outputWriter, "  Rated: %v\n", game.Rated)
			fmt.Fprintf(outputWriter, "  PGN: %s\n\n", game.PGN)
		}

		if output != "" {
			fmt.Printf("Downloaded %d games for %s (%d/%02d) to %s\n",
				len(games.Games), username, year, month, output)
		}

	case "summary":
		games, err := client.GetPlayerGames(username, year, month)
		if err != nil {
			return fmt.Errorf("failed to fetch games: %w", err)
		}

		fmt.Fprintf(outputWriter, "Games for %s (%d/%02d):\n", username, year, month)
		fmt.Fprintf(outputWriter, "Total games: %d\n\n", len(games.Games))

		for i, game := range games.Games {
			fmt.Fprintf(outputWriter, "Game %d: %s vs %s\n",
				i+1, game.White.Username, game.Black.Username)
			fmt.Fprintf(outputWriter, "  Result: %s-%s\n", game.White.Result, game.Black.Result)
			fmt.Fprintf(outputWriter, "  Time Control: %s\n", game.TimeControl)
			fmt.Fprintf(outputWriter, "  Date: %s\n\n", game.GetEndTime().Format("2006-01-02"))
		}

		if output != "" {
			fmt.Printf("Downloaded summary of %d games for %s (%d/%02d) to %s\n",
				len(games.Games), username, year, month, output)
		}

	default:
		return fmt.Errorf("unknown format %q, supported formats: pgn, json, summary", format)
	}

	return nil
}

// Helper functions for parsing archive URLs

// parseArchiveYear extracts the year from an archive URL part
func parseArchiveYear(yearStr string) (int, error) {
	var year int
	_, err := fmt.Sscanf(yearStr, "%d", &year)
	return year, err
}

// parseArchiveMonth extracts the month from an archive URL part
func parseArchiveMonth(monthStr string) (int, error) {
	var month int
	_, err := fmt.Sscanf(monthStr, "%d", &month)
	return month, err
}

// ConvertToDatabase fetches and converts games to a PGN database
func ConvertToDatabase(username string, year, month int) (*GamesResponse, error) {
	client := NewClient()
	return client.GetPlayerGames(username, year, month)
}
