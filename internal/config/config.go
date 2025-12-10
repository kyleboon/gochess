package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the gochess configuration
type Config struct {
	DatabasePath string                   `yaml:"database_path"`
	LogLevel     string                   `yaml:"log_level,omitempty"`
	ChessCom     *ChessComConfig          `yaml:"chesscom,omitempty"`
	Lichess      *LichessConfig           `yaml:"lichess,omitempty"`
	LastImport   map[string]time.Time     `yaml:"last_import,omitempty"`
}

// ChessComConfig holds Chess.com specific configuration
type ChessComConfig struct {
	Username string `yaml:"username"`
}

// LichessConfig holds Lichess specific configuration
type LichessConfig struct {
	Username string `yaml:"username"`
	APIToken string `yaml:"api_token,omitempty"`
}

// DefaultConfigPath returns the default path to the config file
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".gochess", "config.yaml"), nil
}

// DefaultDatabasePath returns the default path to the database
func DefaultDatabasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".gochess", "games.db"), nil
}

// Load reads the configuration from the specified path
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s (run 'gochess config init' to create one)", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize LastImport map if nil
	if cfg.LastImport == nil {
		cfg.LastImport = make(map[string]time.Time)
	}

	return &cfg, nil
}

// LoadOrDefault loads the configuration from the default path, or returns a default config if not found
func LoadOrDefault() (*Config, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	cfg, err := Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config
			dbPath, err := DefaultDatabasePath()
			if err != nil {
				return nil, err
			}
			return &Config{
				DatabasePath: dbPath,
				LastImport:   make(map[string]time.Time),
			}, nil
		}
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to the specified path
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveDefault writes the configuration to the default path
func (c *Config) SaveDefault() error {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	return c.Save(configPath)
}

// GetLastImport returns the last import time for a given platform/username combination
func (c *Config) GetLastImport(platform, username string) (time.Time, bool) {
	key := fmt.Sprintf("%s:%s", platform, username)
	t, ok := c.LastImport[key]
	return t, ok
}

// SetLastImport sets the last import time for a given platform/username combination
func (c *Config) SetLastImport(platform, username string, t time.Time) {
	if c.LastImport == nil {
		c.LastImport = make(map[string]time.Time)
	}
	key := fmt.Sprintf("%s:%s", platform, username)
	c.LastImport[key] = t
}

// HasAnySource returns true if at least one source is configured
func (c *Config) HasAnySource() bool {
	return (c.ChessCom != nil && c.ChessCom.Username != "") ||
		(c.Lichess != nil && c.Lichess.Username != "")
}

// GetLogLevel returns the configured log level, defaulting to "error" if not set
func (c *Config) GetLogLevel() string {
	if c.LogLevel == "" {
		return "error"
	}
	return c.LogLevel
}
