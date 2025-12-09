package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_SaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "gochess-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create a config
	cfg := &Config{
		DatabasePath: "/path/to/games.db",
		ChessCom: &ChessComConfig{
			Username: "testuser",
		},
		Lichess: &LichessConfig{
			Username: "testuser2",
			APIToken: "secret-token",
		},
		LastImport: map[string]time.Time{
			"lichess:testuser2": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// Save config
	err = cfg.Save(configPath)
	require.NoError(t, err)

	// Load config
	loaded, err := Load(configPath)
	require.NoError(t, err)

	// Verify loaded config
	assert.Equal(t, cfg.DatabasePath, loaded.DatabasePath)
	assert.Equal(t, cfg.ChessCom.Username, loaded.ChessCom.Username)
	assert.Equal(t, cfg.Lichess.Username, loaded.Lichess.Username)
	assert.Equal(t, cfg.Lichess.APIToken, loaded.Lichess.APIToken)
	assert.True(t, cfg.LastImport["lichess:testuser2"].Equal(loaded.LastImport["lichess:testuser2"]))
}

func TestConfig_LoadNonExistent(t *testing.T) {
	// Try to load non-existent config
	_, err := Load("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config file not found")
}

func TestConfig_GetLastImport(t *testing.T) {
	cfg := &Config{
		LastImport: map[string]time.Time{
			"lichess:user1": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			"chesscom:user2": time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name     string
		platform string
		username string
		wantTime time.Time
		wantOk   bool
	}{
		{
			name:     "existing lichess entry",
			platform: "lichess",
			username: "user1",
			wantTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantOk:   true,
		},
		{
			name:     "existing chesscom entry",
			platform: "chesscom",
			username: "user2",
			wantTime: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			wantOk:   true,
		},
		{
			name:     "non-existent entry",
			platform: "lichess",
			username: "user3",
			wantTime: time.Time{},
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotOk := cfg.GetLastImport(tt.platform, tt.username)
			assert.Equal(t, tt.wantOk, gotOk)
			if tt.wantOk {
				assert.True(t, tt.wantTime.Equal(gotTime))
			}
		})
	}
}

func TestConfig_SetLastImport(t *testing.T) {
	cfg := &Config{}

	// Set a last import time
	now := time.Now()
	cfg.SetLastImport("lichess", "testuser", now)

	// Verify it was set
	gotTime, gotOk := cfg.GetLastImport("lichess", "testuser")
	assert.True(t, gotOk)
	assert.True(t, now.Equal(gotTime))

	// Set another one
	later := now.Add(24 * time.Hour)
	cfg.SetLastImport("chesscom", "testuser2", later)

	// Verify both exist
	gotTime1, gotOk1 := cfg.GetLastImport("lichess", "testuser")
	assert.True(t, gotOk1)
	assert.True(t, now.Equal(gotTime1))

	gotTime2, gotOk2 := cfg.GetLastImport("chesscom", "testuser2")
	assert.True(t, gotOk2)
	assert.True(t, later.Equal(gotTime2))
}

func TestConfig_HasAnySource(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "no sources",
			cfg:  &Config{},
			want: false,
		},
		{
			name: "chesscom only",
			cfg: &Config{
				ChessCom: &ChessComConfig{
					Username: "user1",
				},
			},
			want: true,
		},
		{
			name: "lichess only",
			cfg: &Config{
				Lichess: &LichessConfig{
					Username: "user2",
				},
			},
			want: true,
		},
		{
			name: "both sources",
			cfg: &Config{
				ChessCom: &ChessComConfig{
					Username: "user1",
				},
				Lichess: &LichessConfig{
					Username: "user2",
				},
			},
			want: true,
		},
		{
			name: "empty chesscom",
			cfg: &Config{
				ChessCom: &ChessComConfig{
					Username: "",
				},
			},
			want: false,
		},
		{
			name: "empty lichess",
			cfg: &Config{
				Lichess: &LichessConfig{
					Username: "",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.HasAnySource()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultPaths(t *testing.T) {
	configPath, err := DefaultConfigPath()
	require.NoError(t, err)
	assert.Contains(t, configPath, ".gochess")
	assert.Contains(t, configPath, "config.yaml")

	dbPath, err := DefaultDatabasePath()
	require.NoError(t, err)
	assert.Contains(t, dbPath, ".gochess")
	assert.Contains(t, dbPath, "games.db")
}
