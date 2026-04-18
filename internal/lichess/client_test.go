package lichess

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kyleboon/gochess/internal/logging"
)

func TestClient_RetryOn429(t *testing.T) {
	tests := []struct {
		name           string
		failCount      int
		maxRetries     int
		expectSuccess  bool
		expectAttempts int
	}{
		{
			name:           "Success on first attempt",
			failCount:      0,
			maxRetries:     3,
			expectSuccess:  true,
			expectAttempts: 1,
		},
		{
			name:           "Success after 1 retry",
			failCount:      1,
			maxRetries:     3,
			expectSuccess:  true,
			expectAttempts: 2,
		},
		{
			name:           "Success after 2 retries",
			failCount:      2,
			maxRetries:     3,
			expectSuccess:  true,
			expectAttempts: 3,
		},
		{
			name:           "Failure after max retries",
			failCount:      4,
			maxRetries:     3,
			expectSuccess:  false,
			expectAttempts: 4, // Initial attempt + 3 retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int32

			// Create a test server that returns 429 for the first failCount requests
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count := atomic.AddInt32(&requestCount, 1)
				if count <= int32(tt.failCount) {
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				// Success response
				w.Header().Set("Content-Type", "application/x-chess-pgn")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("[Event \"Test Game\"]\n\n"))
			}))
			defer server.Close()

			// Create client with custom retry config and fast backoff for testing
			client := NewClientWithLogger(logging.Discard())
			client.retryConfig = RetryConfig{
				MaxRetries:     tt.maxRetries,
				InitialBackoff: 10 * time.Millisecond,
				MaxBackoff:     50 * time.Millisecond,
				BackoffFactor:  2.0,
			}

			// Create a request to the test server
			ctx := context.Background()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			// Execute request with retry
			resp, err := client.doRequestWithRetry(ctx, req)

			// Verify the number of attempts
			actualAttempts := int(atomic.LoadInt32(&requestCount))
			if actualAttempts != tt.expectAttempts {
				t.Errorf("expected %d attempts, got %d", tt.expectAttempts, actualAttempts)
			}

			// Verify success/failure
			if tt.expectSuccess {
				if err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if resp == nil {
					t.Error("expected response, got nil")
				} else {
					_ = resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						t.Errorf("expected status 200, got %d", resp.StatusCode)
					}
				}
			} else {
				if err == nil {
					t.Error("expected error, got success")
				}
				if resp != nil {
					t.Error("expected nil response on failure")
				}
			}
		})
	}
}

func TestClient_RetryContextCancellation(t *testing.T) {
	var requestCount int32

	// Create a test server that always returns 429
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	// Create client with slow backoff
	client := NewClientWithLogger(logging.Discard())
	client.retryConfig = RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
		BackoffFactor:  2.0,
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Create and execute request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = client.doRequestWithRetry(ctx, req)

	// Should get context cancelled error
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}

	// Should have made at least one request but not all retries
	attempts := int(atomic.LoadInt32(&requestCount))
	if attempts == 0 {
		t.Error("expected at least one request attempt")
	}
	if attempts > 3 {
		t.Errorf("expected context cancellation to stop retries early, got %d attempts", attempts)
	}
}

func TestRetryConfig_ExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		BackoffFactor:  2.0,
	}

	// Simulate backoff calculation
	backoff := config.InitialBackoff
	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
	}

	for i := 0; i < 4; i++ {
		if backoff != expected[i] {
			t.Errorf("attempt %d: expected backoff %v, got %v", i, expected[i], backoff)
		}

		// Calculate next backoff
		backoff = time.Duration(float64(backoff) * config.BackoffFactor)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", config.MaxRetries)
	}

	if config.InitialBackoff != 1*time.Second {
		t.Errorf("expected InitialBackoff=1s, got %v", config.InitialBackoff)
	}

	if config.MaxBackoff != 30*time.Second {
		t.Errorf("expected MaxBackoff=30s, got %v", config.MaxBackoff)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("expected BackoffFactor=2.0, got %f", config.BackoffFactor)
	}
}

func TestGetPlayerGamesPGN(t *testing.T) {
	tests := []struct {
		name        string
		params      GamesParams
		serverPGN   string
		expectError bool
	}{
		{
			name:   "Fetch single game",
			params: DefaultGamesParams("testuser"),
			serverPGN: `[Event "Rated Blitz game"]
[Site "https://lichess.org/abc123"]
[Date "2024.12.09"]
[White "testuser"]
[Black "opponent"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0

`,
			expectError: false,
		},
		{
			name:   "Fetch multiple games",
			params: DefaultGamesParams("testuser"),
			serverPGN: `[Event "Rated Blitz game"]
[Site "https://lichess.org/abc123"]
[Date "2024.12.09"]
[White "testuser"]
[Black "opponent1"]
[Result "1-0"]

1. e4 e5 1-0

[Event "Rated Rapid game"]
[Site "https://lichess.org/def456"]
[Date "2024.12.08"]
[White "opponent2"]
[Black "testuser"]
[Result "0-1"]

1. d4 d5 0-1

`,
			expectError: false,
		},
		{
			name:        "Empty response",
			params:      DefaultGamesParams("testuser"),
			serverPGN:   "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the endpoint
				if !strings.HasPrefix(r.URL.Path, "/api/games/user/") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				// Verify accept header
				if r.Header.Get("Accept") != "application/x-chess-pgn" {
					t.Errorf("unexpected accept header: %s", r.Header.Get("Accept"))
				}

				w.Header().Set("Content-Type", "application/x-chess-pgn")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.serverPGN))
			}))
			defer server.Close()

			// Create client and replace baseURL for testing
			client := NewClientWithLogger(logging.Discard())

			// We need to make the request to our test server
			// Temporarily modify the params to use test server URL
			ctx := context.Background()
			apiURL := server.URL + "/api/games/user/" + tt.params.Username
			queryParams := client.buildQueryParams(tt.params)
			if len(queryParams) > 0 {
				apiURL += "?" + queryParams
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Accept", "application/x-chess-pgn")

			resp, err := client.doRequestWithRetry(ctx, req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			// Read response
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				t.Fatalf("failed to read response: %v", readErr)
			}

			pgn := string(body)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got success")
				}
			} else {
				if err != nil {
					t.Errorf("expected success, got error: %v", err)
				}
				if pgn != tt.serverPGN {
					t.Errorf("PGN mismatch\nexpected:\n%s\ngot:\n%s", tt.serverPGN, pgn)
				}
			}
		})
	}
}

func TestBuildQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		params   GamesParams
		expected map[string]string
	}{
		{
			name:   "Default params",
			params: DefaultGamesParams("testuser"),
			expected: map[string]string{
				"moves":    "true",
				"tags":     "true",
				"clocks":   "true",
				"evals":    "true",
				"opening":  "true",
				"ongoing":  "false",
				"finished": "true",
				"sort":     "dateDesc",
			},
		},
		{
			name: "With optional filters",
			params: GamesParams{
				Username: "testuser",
				Max:      intPtr(10),
				Vs:       "opponent",
				Rated:    boolPtr(true),
				PerfType: "blitz",
				Color:    "white",
				Moves:    true,
				Tags:     true,
				Clocks:   false,
				Evals:    false,
				Opening:  true,
				Ongoing:  false,
				Finished: true,
				Sort:     "dateAsc",
			},
			expected: map[string]string{
				"max":      "10",
				"vs":       "opponent",
				"rated":    "true",
				"perfType": "blitz",
				"color":    "white",
				"moves":    "true",
				"tags":     "true",
				"clocks":   "false",
				"evals":    "false",
				"opening":  "true",
				"ongoing":  "false",
				"finished": "true",
				"sort":     "dateAsc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithLogger(logging.Discard())
			queryString := client.buildQueryParams(tt.params)

			// Parse the query string and verify each expected parameter
			for key, expectedValue := range tt.expected {
				if !strings.Contains(queryString, key+"="+expectedValue) {
					t.Errorf("expected query param %s=%s not found in %s", key, expectedValue, queryString)
				}
			}
		})
	}
}

func TestSetAPIToken(t *testing.T) {
	client := NewClient()

	if client.apiToken != "" {
		t.Error("expected empty API token on new client")
	}

	token := "test_token_123"
	client.SetAPIToken(token)

	if client.apiToken != token {
		t.Errorf("expected API token %s, got %s", token, client.apiToken)
	}
}

func TestDefaultGamesParams(t *testing.T) {
	username := "testuser"
	params := DefaultGamesParams(username)

	if params.Username != username {
		t.Errorf("expected username %s, got %s", username, params.Username)
	}

	if !params.Moves {
		t.Error("expected Moves to be true")
	}

	if !params.Tags {
		t.Error("expected Tags to be true")
	}

	if !params.Clocks {
		t.Error("expected Clocks to be true")
	}

	if !params.Evals {
		t.Error("expected Evals to be true")
	}

	if !params.Opening {
		t.Error("expected Opening to be true")
	}

	if params.Ongoing {
		t.Error("expected Ongoing to be false")
	}

	if !params.Finished {
		t.Error("expected Finished to be true")
	}

	if params.Sort != "dateDesc" {
		t.Errorf("expected Sort to be dateDesc, got %s", params.Sort)
	}
}

// Helper functions for pointer types
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func TestClient_GetPlayerGamesPGN(t *testing.T) {
	t.Run("Success with basic params", func(t *testing.T) {
		// Note: The client filters out blank lines, so expected PGN has no blank lines
		expectedPGN := `[Event "Rated Blitz game"]
[Site "https://lichess.org/abc123"]
[Date "2024.01.01"]
[White "player1"]
[Black "player2"]
[Result "1-0"]
1. e4 e5 2. Nf3 Nc6 1-0
[Event "Rated Rapid game"]
[Site "https://lichess.org/def456"]
[Date "2024.01.02"]
[White "player3"]
[Black "player4"]
[Result "0-1"]
1. d4 d5 0-1
`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify path
			if !strings.Contains(r.URL.Path, "/games/user/testuser") {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			// Verify headers
			if r.Header.Get("Accept") != "application/x-chess-pgn" {
				t.Errorf("expected Accept header for PGN")
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(expectedPGN))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL
		params := GamesParams{
			Username: "testuser",
		}

		pgn, err := client.GetPlayerGamesPGN(context.Background(), params)

		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if pgn != expectedPGN {
			t.Errorf("PGN mismatch.\nExpected:\n%s\nGot:\n%s", expectedPGN, pgn)
		}
	})

	t.Run("Success with date range", func(t *testing.T) {
		since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
		until := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC).UnixMilli()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify query parameters
			query := r.URL.Query()
			if query.Get("since") != strconv.FormatInt(since, 10) {
				t.Errorf("expected since=%d, got %s", since, query.Get("since"))
			}
			if query.Get("until") != strconv.FormatInt(until, 10) {
				t.Errorf("expected until=%d, got %s", until, query.Get("until"))
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[Event \"Test\"]\n1. e4 e5 1-0\n"))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL
		params := GamesParams{
			Username: "testuser",
			Since:    &since,
			Until:    &until,
		}

		_, err := client.GetPlayerGamesPGN(context.Background(), params)

		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
	})

	t.Run("Success with API token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify authorization header
			auth := r.Header.Get("Authorization")
			expectedAuth := "Bearer test-token-123"
			if auth != expectedAuth {
				t.Errorf("expected Authorization: %s, got: %s", expectedAuth, auth)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[Event \"Test\"]\n1. e4 e5 1-0\n"))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL
		client.SetAPIToken("test-token-123")

		params := GamesParams{
			Username: "testuser",
		}

		_, err := client.GetPlayerGamesPGN(context.Background(), params)

		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
	})

	t.Run("Success with filters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			// Verify filter parameters
			if query.Get("rated") != "true" {
				t.Errorf("expected rated=true, got %s", query.Get("rated"))
			}
			if query.Get("perfType") != "blitz" {
				t.Errorf("expected perfType=blitz, got %s", query.Get("perfType"))
			}
			if query.Get("color") != "white" {
				t.Errorf("expected color=white, got %s", query.Get("color"))
			}
			if query.Get("max") != "50" {
				t.Errorf("expected max=50, got %s", query.Get("max"))
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[Event \"Test\"]\n1. e4 e5 1-0\n"))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL

		rated := true
		max := 50

		params := GamesParams{
			Username: "testuser",
			Rated:    &rated,
			PerfType: "blitz",
			Color:    "white",
			Max:      &max,
		}

		_, err := client.GetPlayerGamesPGN(context.Background(), params)

		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
	})

	t.Run("Empty PGN response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(""))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL
		params := GamesParams{
			Username: "newuser",
		}

		pgn, err := client.GetPlayerGamesPGN(context.Background(), params)

		if err != nil {
			t.Fatalf("expected success with empty PGN, got error: %v", err)
		}
		if pgn != "" {
			t.Errorf("expected empty string, got: %s", pgn)
		}
	})

	t.Run("404 Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL
		params := GamesParams{
			Username: "nonexistent",
		}

		_, err := client.GetPlayerGamesPGN(context.Background(), params)

		if err == nil {
			t.Error("expected error for 404 response")
		}
	})

	t.Run("429 Rate Limit", func(t *testing.T) {
		var requestCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			// Second request succeeds
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[Event \"Test\"]\n1. e4 e5 1-0\n"))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL
		client.SetRetryConfig(RetryConfig{
			MaxRetries:     2,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     50 * time.Millisecond,
			BackoffFactor:  2.0,
		})

		params := GamesParams{
			Username: "testuser",
		}

		pgn, err := client.GetPlayerGamesPGN(context.Background(), params)

		if err != nil {
			t.Fatalf("expected success after retry, got error: %v", err)
		}
		if pgn == "" {
			t.Error("expected non-empty PGN")
		}
		if requestCount != 2 {
			t.Errorf("expected 2 requests (1 fail + 1 retry), got %d", requestCount)
		}
	})
}
