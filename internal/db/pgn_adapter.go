package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kyleboon/gochess/internal/pgn"
)

// PGNData represents a parsed PGN file with original content preserved
type PGNData struct {
	// PgnDB is the parsed PGN database
	PgnDB *pgn.DB
	// GameTexts contains the original text of each game, including moves
	GameTexts []string
}

// ParsePGNFile parses a PGN file with additional pre-processing for compatibility
// with different PGN formats (like Chess.com exports)
func ParsePGNFile(filePath string) (*pgn.DB, []error) {
	// Read PGN file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read PGN file: %w", err)}
	}

	// Pre-process the data to handle Chess.com specific formats
	processedData := preprocessPGN(string(data))

	// Parse PGN
	pgnDB := &pgn.DB{}
	parseErrors := pgnDB.Parse(processedData)

	return pgnDB, parseErrors
}

// ParsePGNFileWithMoves parses a PGN file and preserves the complete game text including moves
func ParsePGNFileWithMoves(filePath string) (*PGNData, []error) {
	// Read PGN file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read PGN file: %w", err)}
	}

	// Pre-process the data to handle Chess.com specific formats
	processedData := preprocessPGN(string(data))

	// Split into individual games to preserve complete text
	games := splitGames(processedData)

	// Parse PGN
	pgnDB := &pgn.DB{}
	parseErrors := pgnDB.Parse(processedData)

	// Create result structure with both parsed data and original text
	result := &PGNData{
		PgnDB:     pgnDB,
		GameTexts: games,
	}

	return result, parseErrors
}

// preprocessPGN handles common PGN format variations to improve compatibility
func preprocessPGN(data string) string {
	// Preprocessing is no longer needed as the parser now handles FEN tags directly.
	return data
}

// splitGames attempts to split a PGN string into individual games
func splitGames(data string) []string {
	// First try to split by event tag since this typically indicates the start of a new game
	// Using regex to find game boundaries more accurately
	regex := regexp.MustCompile(`(?m)^\[Event `) // Matches [Event at start of line
	
	// Find all positions where [Event appears at the start of a line
	matches := regex.FindAllStringIndex(data, -1)
	
	if len(matches) == 0 {
		// If no matches, check if the whole thing is a single game
		if strings.Contains(data, "[White ") && strings.Contains(data, "[Black ") {
			return []string{data}
		}
		return nil
	}
	
	// Extract each game using the positions we found
	games := make([]string, len(matches))
	for i, match := range matches {
		startPos := match[0]
		endPos := len(data)
		
		// If this isn't the last game, set end position to the start of the next game
		if i < len(matches)-1 {
			endPos = matches[i+1][0]
		}
		
		// Extract this game
		games[i] = data[startPos:endPos]
	}
	
	// Clean up each game
	for i, game := range games {
		// Ensure each game has proper formatting
		game = strings.TrimSpace(game)
		
		// Make sure the game has required tags
		if !strings.Contains(game, "[White ") || !strings.Contains(game, "[Black ") {
			// This is not a complete game, remove it
			games[i] = ""
			continue
		}
		
		games[i] = game
	}
	
	// Remove any empty games
	validGames := make([]string, 0, len(games))
	for _, game := range games {
		if game != "" {
			validGames = append(validGames, game)
		}
	}
	
	return validGames
}

// ExtractMoveText extracts just the moves portion of a PGN game
func ExtractMoveText(gameText string) string {
	// Find the end of the last tag section
	lastTagEnd := strings.LastIndex(gameText, "]")
	if lastTagEnd == -1 || lastTagEnd >= len(gameText)-1 {
		return ""
	}
	
	// Extract everything after the last tag
	moveText := gameText[lastTagEnd+1:]
	
	// Clean up the move text
	moveText = strings.TrimSpace(moveText)
	
	return moveText
}

// CalculateGameHash generates a unique hash for a game based on its key properties
func CalculateGameHash(game *pgn.Game, moveText string) string {
	// Create a normalized string with the most important game information
	// This should be reasonably unique for each game, but ignore irrelevant differences
	
	// Get essential tags
	white := game.Tags["White"]
	black := game.Tags["Black"]
	date := game.Tags["Date"]
	result := game.Tags["Result"]
	
	// Clean up move text by removing clock annotations and comments
	// This helps ensure the same game doesn't get different hashes due to annotations
	cleanMoves := CleanMoveText(moveText)
	
	// Create a string to hash with the most identifying information
	toHash := fmt.Sprintf("%s|%s|%s|%s|%s", white, black, date, result, cleanMoves)
	
	// Calculate SHA-256 hash
	hash := sha256.Sum256([]byte(toHash))
	
	// Convert to hex string
	return hex.EncodeToString(hash[:])
}

// CleanMoveText removes clock annotations, comments, and unnecessary whitespace
// to create a normalized move text for hashing
func CleanMoveText(moveText string) string {
	// Remove clock annotations like {[%clk 0:01:23.4]}
	clockRegex := regexp.MustCompile(`\{\[\%clk [^\}]*\}\}`)
	moveText = clockRegex.ReplaceAllString(moveText, "")
	
	// Remove all comments (text in curly braces)
	commentRegex := regexp.MustCompile(`\{[^\}]*\}`)
	moveText = commentRegex.ReplaceAllString(moveText, "")
	
	// Remove NAGs ($1, $2, etc.)
	nagRegex := regexp.MustCompile(`\$\d+`)
	moveText = nagRegex.ReplaceAllString(moveText, "")
	
	// Normalize whitespace
	moveText = strings.Join(strings.Fields(moveText), " ")
	
	return moveText
}
