package chesscom

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

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

// DownloadGames downloads games for a Chess.com user
func DownloadGames(c *cli.Context) error {
	username := c.String("username")
	year := c.Int("year")
	month := c.Int("month")
	format := c.String("format")
	output := c.String("output")

	client := NewClient()

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

// ConvertToDatabase fetches and converts games to a PGN database
func ConvertToDatabase(username string, year, month int) (*GamesResponse, error) {
	client := NewClient()
	return client.GetPlayerGames(username, year, month)
}
