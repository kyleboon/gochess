package lichess

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

// DownloadGames downloads games for a Lichess user
func DownloadGames(c *cli.Context) error {
	username := c.String("username")
	output := c.String("output")
	importDB := c.Bool("import-db")
	dbPath := c.String("database")
	verbose := c.Bool("verbose")
	apiToken := c.String("api-token")

	// Optional date range filters
	since := c.String("since")
	until := c.String("until")
	max := c.Int("max")

	// Optional game filters
	vs := c.String("vs")
	rated := c.String("rated")
	perfType := c.String("perf-type")
	color := c.String("color")

	client := NewClient()

	// Set API token if provided
	if apiToken != "" {
		client.SetAPIToken(apiToken)
	}

	// Expand database path
	dbPath = expandPath(dbPath)

	// Build parameters
	params := DefaultGamesParams(username)

	// Parse date range
	if since != "" {
		sinceTime, err := parseTimeString(since)
		if err != nil {
			return fmt.Errorf("failed to parse --since: %w", err)
		}
		sinceMillis := sinceTime.UnixMilli()
		params.Since = &sinceMillis
	}

	if until != "" {
		untilTime, err := parseTimeString(until)
		if err != nil {
			return fmt.Errorf("failed to parse --until: %w", err)
		}
		untilMillis := untilTime.UnixMilli()
		params.Until = &untilMillis
	}

	if max > 0 {
		params.Max = &max
	}

	if vs != "" {
		params.Vs = vs
	}

	if rated != "" {
		if rated == "true" {
			ratedBool := true
			params.Rated = &ratedBool
		} else if rated == "false" {
			ratedBool := false
			params.Rated = &ratedBool
		}
	}

	if perfType != "" {
		params.PerfType = perfType
	}

	if color != "" {
		params.Color = color
	}

	fmt.Printf("Fetching games for %s from Lichess...\n", username)

	// Get the PGN data
	pgn, err := client.GetPlayerGamesPGN(c.Context, params)
	if err != nil {
		return fmt.Errorf("failed to fetch games: %w", err)
	}

	if pgn == "" {
		fmt.Printf("No games found for %s\n", username)
		return nil
	}

	// Count games (rough estimate based on [Event tags)
	gameCount := strings.Count(pgn, "[Event ")

	// If we're importing to DB
	if importDB {
		// Create a temporary file to store the PGN for import
		tmpfile, err := os.CreateTemp("", "lichess-*.pgn")
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}
		tmpPath := tmpfile.Name()
		defer os.Remove(tmpPath) // Clean up

		// Write PGN to temporary file
		if _, err := tmpfile.WriteString(pgn); err != nil {
			tmpfile.Close()
			return fmt.Errorf("failed to write to temporary file: %w", err)
		}
		tmpfile.Close()

		// Open database
		fmt.Printf("Opening database at %s...\n", dbPath)
		database, err := db.New(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer database.Close()

		// Import the PGN file
		count, errors := database.ImportPGN(c.Context, tmpPath)

		// Print import results
		if len(errors) > 0 && verbose {
			fmt.Printf("Encountered %d errors during import:\n", len(errors))
			for i, err := range errors {
				fmt.Printf("  Error %d: %s\n", i+1, err)
				if pgnErr, ok := err.(*db.PGNImportError); ok && pgnErr.PGNText != "" {
					fmt.Printf("    PGN: %s\n", pgnErr.PGNText)
				}
			}
		} else if len(errors) > 0 {
			fmt.Printf("Encountered %d errors during import. Use --verbose to see details.\n", len(errors))
		}

		fmt.Printf("Successfully imported %d games from Lichess\n", count)

		// Get current game count in database
		currentCount, err := database.GetGameCount(c.Context)
		if err == nil {
			fmt.Printf("Total games in database: %d\n", currentCount)
		}

		// If the user also wants to output to a file, do that too
		if output != "" {
			fmt.Println("\nAdditionally saving PGN to file...")
			outputFile, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer outputFile.Close()

			_, err = outputFile.WriteString(pgn)
			if err != nil {
				return fmt.Errorf("failed to write to output file: %w", err)
			}

			fmt.Printf("Saved PGN to %s\n", output)
		}

		return nil
	}

	// Handle output to file or stdout
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

	// Write PGN to output
	fmt.Fprintln(outputWriter, pgn)

	if output != "" {
		fmt.Printf("Downloaded %d games for %s to %s\n", gameCount, username, output)
	} else {
		fmt.Printf("Downloaded %d games for %s\n", gameCount, username)
	}

	return nil
}

// parseTimeString parses various time string formats into time.Time
// Supported formats: YYYY-MM-DD, YYYY-MM, YYYY
func parseTimeString(timeStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006-01",
		"2006",
	}

	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time string %q (supported formats: YYYY-MM-DD, YYYY-MM, YYYY)", timeStr)
}
