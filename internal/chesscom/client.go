// Package chesscom provides a client for the Chess.com API.
package chesscom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kyleboon/gochess/internal/logging"
)

const (
	baseURL = "https://api.chess.com/pub"
)

// Client represents a Chess.com API client.
type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new Chess.com API client.
func NewClient() *Client {
	return NewClientWithLogger(logging.Default())
}

// NewClientWithLogger creates a new Chess.com API client with a custom logger.
func NewClientWithLogger(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// GetPlayerGames fetches a list of games for a specific user in a given month and year.
func (c *Client) GetPlayerGames(ctx context.Context, username string, year, month int) (*GamesResponse, error) {
	url := fmt.Sprintf("%s/player/%s/games/%d/%02d", baseURL, username, year, month)
	c.logger.Info("fetching player games", "username", username, "year", year, "month", month, "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request", "error", err, "url", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed", "error", err, "url", url)
		return nil, fmt.Errorf("failed to fetch games: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response", "statusCode", resp.StatusCode, "url", url)

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("unexpected status code from Chess.com API", "statusCode", resp.StatusCode, "url", url)
		return nil, fmt.Errorf("chess.com API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err, "url", url)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var games GamesResponse
	if err := json.Unmarshal(body, &games); err != nil {
		c.logger.Error("failed to unmarshal JSON response", "error", err, "url", url)
		return nil, fmt.Errorf("failed to unmarshal games response: %w", err)
	}

	c.logger.Info("successfully fetched player games", "username", username, "year", year, "month", month, "gamesCount", len(games.Games))
	return &games, nil
}

// GetPlayerGamesPGN downloads the PGN file containing all games for a specific user in a given month and year.
func (c *Client) GetPlayerGamesPGN(ctx context.Context, username string, year, month int) (string, error) {
	url := fmt.Sprintf("%s/player/%s/games/%d/%02d/pgn", baseURL, username, year, month)
	c.logger.Info("fetching player games PGN", "username", username, "year", year, "month", month, "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request", "error", err, "url", url)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed", "error", err, "url", url)
		return "", fmt.Errorf("failed to fetch PGN: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response", "statusCode", resp.StatusCode, "url", url)

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("unexpected status code from Chess.com API", "statusCode", resp.StatusCode, "url", url)
		return "", fmt.Errorf("chess.com API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err, "url", url)
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Info("successfully fetched PGN", "username", username, "year", year, "month", month, "pgnSize", len(body))
	return string(body), nil
}

// GetArchivedMonths returns a list of monthly archives available for a player.
func (c *Client) GetArchivedMonths(ctx context.Context, username string) (*ArchivesResponse, error) {
	url := fmt.Sprintf("%s/player/%s/games/archives", baseURL, username)
	c.logger.Info("fetching archived months", "username", username, "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request", "error", err, "url", url)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed", "error", err, "url", url)
		return nil, fmt.Errorf("failed to fetch archives: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response", "statusCode", resp.StatusCode, "url", url)

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("unexpected status code from Chess.com API", "statusCode", resp.StatusCode, "url", url)
		return nil, fmt.Errorf("chess.com API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read response body", "error", err, "url", url)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var archives ArchivesResponse
	if err := json.Unmarshal(body, &archives); err != nil {
		c.logger.Error("failed to unmarshal JSON response", "error", err, "url", url)
		return nil, fmt.Errorf("failed to unmarshal archives response: %w", err)
	}

	c.logger.Info("successfully fetched archived months", "username", username, "archiveCount", len(archives.Archives))
	return &archives, nil
}
