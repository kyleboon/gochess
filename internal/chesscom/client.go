// Package chesscom provides a client for the Chess.com API.
//
// Rate Limiting:
// According to Chess.com API documentation, serial access is unlimited.
// However, parallel requests may trigger rate limiting, resulting in a
// "429 Too Many Requests" response. This client implements automatic retry
// with exponential backoff when rate limiting occurs.
//
// To avoid rate limiting:
//   - Make requests sequentially (serial access is unlimited)
//   - Avoid running multiple instances of the client in parallel
//   - If you receive 429 responses, the client will automatically retry
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

	// Default retry configuration
	defaultMaxRetries     = 3
	defaultInitialBackoff = 1 * time.Second
	defaultMaxBackoff     = 30 * time.Second
	defaultBackoffFactor  = 2.0
)

// RetryConfig holds the retry configuration for handling rate limiting.
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retry attempts
	InitialBackoff time.Duration // Initial backoff duration
	MaxBackoff     time.Duration // Maximum backoff duration
	BackoffFactor  float64       // Exponential backoff multiplier
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     defaultMaxRetries,
		InitialBackoff: defaultInitialBackoff,
		MaxBackoff:     defaultMaxBackoff,
		BackoffFactor:  defaultBackoffFactor,
	}
}

// Client represents a Chess.com API client.
//
// Note on Concurrency and Rate Limiting:
// The Chess.com API allows unlimited serial access but may rate limit
// parallel requests. This client does NOT use internal locking, so if you
// need to make parallel requests from multiple goroutines, you should
// implement external coordination to avoid rate limiting.
type Client struct {
	httpClient  *http.Client
	logger      *slog.Logger
	retryConfig RetryConfig
}

// NewClient creates a new Chess.com API client with default settings.
func NewClient() *Client {
	return NewClientWithLogger(logging.Default())
}

// NewClientWithLogger creates a new Chess.com API client with a custom logger.
func NewClientWithLogger(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:      logger,
		retryConfig: DefaultRetryConfig(),
	}
}

// SetRetryConfig sets custom retry configuration for the client.
func (c *Client) SetRetryConfig(config RetryConfig) {
	c.retryConfig = config
}

// doRequestWithRetry executes an HTTP request with automatic retry on 429 responses.
// It implements exponential backoff according to the client's retry configuration.
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	backoff := c.retryConfig.InitialBackoff

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Execute the request
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		// If not rate limited, return the response
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Close the body before retrying
		resp.Body.Close()

		// If this was the last attempt, return the error
		if attempt == c.retryConfig.MaxRetries {
			c.logger.Error("max retries exceeded for rate limited request",
				"url", req.URL.String(),
				"attempts", attempt+1,
				"maxRetries", c.retryConfig.MaxRetries)
			return nil, fmt.Errorf("rate limited after %d retries (HTTP 429)", c.retryConfig.MaxRetries)
		}

		// Log the retry
		c.logger.Warn("rate limited, retrying after backoff",
			"url", req.URL.String(),
			"attempt", attempt+1,
			"backoff", backoff,
			"statusCode", http.StatusTooManyRequests)

		// Wait with exponential backoff
		select {
		case <-time.After(backoff):
			// Continue to next retry
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		// Calculate next backoff duration with exponential increase
		backoff = time.Duration(float64(backoff) * c.retryConfig.BackoffFactor)
		if backoff > c.retryConfig.MaxBackoff {
			backoff = c.retryConfig.MaxBackoff
		}

		// For subsequent attempts, we need to clone the request since the body may have been read
		// However, for GET requests (which all our API calls are), this is not an issue
	}

	// Should never reach here due to the loop logic, but added for safety
	return nil, fmt.Errorf("unexpected retry loop exit")
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

	resp, err := c.doRequestWithRetry(ctx, req)
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

	resp, err := c.doRequestWithRetry(ctx, req)
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

	resp, err := c.doRequestWithRetry(ctx, req)
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
