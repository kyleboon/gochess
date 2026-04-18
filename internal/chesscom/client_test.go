package chesscom

import (
	"context"
	"net/http"
	"net/http/httptest"
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
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"archives":[]}`))
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
			// Note: We test the retry logic directly using httptest server instead
			// of mocking the baseURL constant
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

func TestClient_GetPlayerGames(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request path
			expectedPath := "/pub/player/testuser/games/2024/01"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			// Return sample JSON response
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"games": [
					{
						"url": "https://www.chess.com/game/live/123",
						"pgn": "[Event \"Live Chess\"]\n1. e4 e5",
						"time_control": "600",
						"end_time": 1609459200,
						"white": {"username": "player1", "rating": 1500},
						"black": {"username": "player2", "rating": 1510}
					}
				]
			}`))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL + "/pub"
		games, err := client.GetPlayerGames(context.Background(), "testuser", 2024, 1)

		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if games == nil {
			t.Fatal("expected games response, got nil")
		}
		if len(games.Games) != 1 {
			t.Errorf("expected 1 game, got %d", len(games.Games))
		}
	})

	t.Run("404 Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL + "/pub"
		_, err := client.GetPlayerGames(context.Background(), "nonexistent", 2024, 1)

		if err == nil {
			t.Error("expected error for 404 response")
		}
	})
}

func TestClient_GetPlayerGamesPGN(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedPGN := "[Event \"Test\"]\n1. e4 e5 1-0\n"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/pub/player/testuser/games/2024/01/pgn"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(expectedPGN))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL + "/pub"
		pgn, err := client.GetPlayerGamesPGN(context.Background(), "testuser", 2024, 1)

		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if pgn != expectedPGN {
			t.Errorf("expected PGN %q, got %q", expectedPGN, pgn)
		}
	})

	t.Run("404 Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL + "/pub"
		_, err := client.GetPlayerGamesPGN(context.Background(), "nonexistent", 2024, 1)

		if err == nil {
			t.Error("expected error for 404 response")
		}
	})
}

func TestClient_GetArchivedMonths(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/pub/player/testuser/games/archives"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"archives": [
					"https://api.chess.com/pub/player/testuser/games/2024/01",
					"https://api.chess.com/pub/player/testuser/games/2024/02"
				]
			}`))
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL + "/pub"
		archives, err := client.GetArchivedMonths(context.Background(), "testuser")

		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if len(archives.Archives) != 2 {
			t.Errorf("expected 2 archives, got %d", len(archives.Archives))
		}
	})

	t.Run("404 Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClientWithLogger(logging.Discard())
		client.baseURL = server.URL + "/pub"
		_, err := client.GetArchivedMonths(context.Background(), "nonexistent")

		if err == nil {
			t.Error("expected error for 404 response")
		}
	})
}
