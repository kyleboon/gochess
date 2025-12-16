// Package lichess provides a client for the Lichess API.
//
// Rate Limiting:
// Lichess API allows approximately 120 requests per minute with proper authentication.
// This client implements automatic retry with exponential backoff when rate limiting occurs.
//
// To avoid rate limiting:
//   - Use an API token for authenticated requests (higher limits)
//   - Make requests with reasonable delays between calls
//   - If you receive 429 responses, the client will automatically retry
package lichess

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kyleboon/gochess/internal/logging"
)

const (
	baseURL = "https://lichess.org/api"

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

// Client represents a Lichess API client.
//
// Note on Authentication:
// For public games, no authentication is required. For private games or
// higher rate limits, provide an API token via SetAPIToken().
type Client struct {
	httpClient  *http.Client
	logger      *slog.Logger
	retryConfig RetryConfig
	apiToken    string
	baseURL     string // Base URL for API requests (exposed for testing)
}

// NewClient creates a new Lichess API client with default settings.
func NewClient() *Client {
	return NewClientWithLogger(logging.Default())
}

// NewClientWithLogger creates a new Lichess API client with a custom logger.
func NewClientWithLogger(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Longer timeout for streaming responses
		},
		logger:      logger,
		retryConfig: DefaultRetryConfig(),
		baseURL:     baseURL,
	}
}

// SetRetryConfig sets custom retry configuration for the client.
func (c *Client) SetRetryConfig(config RetryConfig) {
	c.retryConfig = config
}

// SetAPIToken sets the API token for authenticated requests.
// This enables access to private games and provides higher rate limits.
func (c *Client) SetAPIToken(token string) {
	c.apiToken = token
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
	}

	// Should never reach here due to the loop logic, but added for safety
	return nil, fmt.Errorf("unexpected retry loop exit")
}

// GetPlayerGamesPGN fetches games for a specific user in PGN format.
// The games are returned as a single PGN string with newline separators.
// Use GamesParams to configure filters and options for the request.
func (c *Client) GetPlayerGamesPGN(ctx context.Context, params GamesParams) (string, error) {
	// Build the URL with query parameters
	apiURL := fmt.Sprintf("%s/games/user/%s", c.baseURL, params.Username)
	queryParams := c.buildQueryParams(params)

	if len(queryParams) > 0 {
		apiURL += "?" + queryParams
	}

	c.logger.Info("fetching player games from Lichess",
		"username", params.Username,
		"url", apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		c.logger.Error("failed to create HTTP request", "error", err, "url", apiURL)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header if token is set
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	}

	// Set accept header for PGN format
	req.Header.Set("Accept", "application/x-chess-pgn")

	resp, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		c.logger.Error("HTTP request failed", "error", err, "url", apiURL)
		return "", fmt.Errorf("failed to fetch games: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received HTTP response",
		"statusCode", resp.StatusCode,
		"url", apiURL)

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("unexpected status code from Lichess API",
			"statusCode", resp.StatusCode,
			"url", apiURL)
		return "", fmt.Errorf("lichess API returned status code %d", resp.StatusCode)
	}

	// Lichess returns games as ndjson (newline-delimited PGN)
	// We need to read the entire stream
	var pgnBuilder strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	// Increase buffer size for large games
	const maxCapacity = 512 * 1024 // 512KB per line
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	gameCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			pgnBuilder.WriteString(line)
			pgnBuilder.WriteString("\n")

			// Count games (each game ends with a blank line after result)
			if strings.HasPrefix(line, "[Event ") {
				gameCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		c.logger.Error("failed to read response stream", "error", err, "url", apiURL)
		return "", fmt.Errorf("failed to read response stream: %w", err)
	}

	pgn := pgnBuilder.String()
	c.logger.Info("successfully fetched PGN from Lichess",
		"username", params.Username,
		"pgnSize", len(pgn),
		"gameCount", gameCount)

	return pgn, nil
}

// buildQueryParams constructs the query string from GamesParams.
func (c *Client) buildQueryParams(params GamesParams) string {
	queryParams := url.Values{}

	if params.Since != nil {
		queryParams.Add("since", strconv.FormatInt(*params.Since, 10))
	}
	if params.Until != nil {
		queryParams.Add("until", strconv.FormatInt(*params.Until, 10))
	}
	if params.Max != nil {
		queryParams.Add("max", strconv.Itoa(*params.Max))
	}
	if params.Vs != "" {
		queryParams.Add("vs", params.Vs)
	}
	if params.Rated != nil {
		queryParams.Add("rated", strconv.FormatBool(*params.Rated))
	}
	if params.PerfType != "" {
		queryParams.Add("perfType", params.PerfType)
	}
	if params.Color != "" {
		queryParams.Add("color", params.Color)
	}
	if params.Analyzed != nil {
		queryParams.Add("analyzed", strconv.FormatBool(*params.Analyzed))
	}

	// Boolean flags (defaults are handled in DefaultGamesParams)
	queryParams.Add("moves", strconv.FormatBool(params.Moves))
	queryParams.Add("tags", strconv.FormatBool(params.Tags))
	queryParams.Add("clocks", strconv.FormatBool(params.Clocks))
	queryParams.Add("evals", strconv.FormatBool(params.Evals))
	queryParams.Add("opening", strconv.FormatBool(params.Opening))
	queryParams.Add("ongoing", strconv.FormatBool(params.Ongoing))
	queryParams.Add("finished", strconv.FormatBool(params.Finished))

	if params.Sort != "" {
		queryParams.Add("sort", params.Sort)
	}

	return queryParams.Encode()
}
