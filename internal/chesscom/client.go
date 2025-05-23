// Package chesscom provides a client for the Chess.com API.
package chesscom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.chess.com/pub"
)

// Client represents a Chess.com API client.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Chess.com API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetPlayerGames fetches a list of games for a specific user in a given month and year.
func (c *Client) GetPlayerGames(username string, year, month int) (*GamesResponse, error) {
	url := fmt.Sprintf("%s/player/%s/games/%d/%02d", baseURL, username, year, month)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch games: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chess.com API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var games GamesResponse
	if err := json.Unmarshal(body, &games); err != nil {
		return nil, fmt.Errorf("failed to unmarshal games response: %w", err)
	}

	return &games, nil
}

// GetPlayerGamesPGN downloads the PGN file containing all games for a specific user in a given month and year.
func (c *Client) GetPlayerGamesPGN(username string, year, month int) (string, error) {
	url := fmt.Sprintf("%s/player/%s/games/%d/%02d/pgn", baseURL, username, year, month)

	fmt.Printf("Fetching PGN for %s (%d/%02d)\n", username, year, month)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PGN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chess.com API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// GetArchivedMonths returns a list of monthly archives available for a player.
func (c *Client) GetArchivedMonths(username string) (*ArchivesResponse, error) {
	url := fmt.Sprintf("%s/player/%s/games/archives", baseURL, username)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch archives: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chess.com API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var archives ArchivesResponse
	if err := json.Unmarshal(body, &archives); err != nil {
		return nil, fmt.Errorf("failed to unmarshal archives response: %w", err)
	}

	return &archives, nil
}
