package logging

import (
	"io"
	"log/slog"
	"os"
)

// Level represents the logging level
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Format represents the log output format
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Config holds the logging configuration
type Config struct {
	Level  Level
	Format Format
	Output io.Writer
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() Config {
	return Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: os.Stderr,
	}
}

// New creates a new slog.Logger with the given configuration
func New(cfg Config) *slog.Logger {
	// Convert our Level to slog.Level
	var level slog.Level
	switch cfg.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create the appropriate handler based on format
	var handler slog.Handler
	if cfg.Format == FormatJSON {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	return slog.New(handler)
}

// Default returns a logger with default configuration
func Default() *slog.Logger {
	return New(DefaultConfig())
}

// NewWithLevel creates a new logger with the specified level
func NewWithLevel(level Level) *slog.Logger {
	cfg := DefaultConfig()
	cfg.Level = level
	return New(cfg)
}

// Discard returns a logger that discards all output (useful for testing)
func Discard() *slog.Logger {
	cfg := DefaultConfig()
	cfg.Output = io.Discard
	return New(cfg)
}
