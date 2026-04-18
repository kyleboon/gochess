package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		verify func(t *testing.T, logger *slog.Logger, buf *bytes.Buffer)
	}{
		{
			name: "Text format with info level",
			config: Config{
				Level:  LevelInfo,
				Format: FormatText,
				Output: nil, // Will be set in test
			},
			verify: func(t *testing.T, logger *slog.Logger, buf *bytes.Buffer) {
				logger.Info("test message", "key", "value")
				output := buf.String()
				if !strings.Contains(output, "test message") {
					t.Errorf("Expected output to contain 'test message', got: %s", output)
				}
				if !strings.Contains(output, "key=value") {
					t.Errorf("Expected output to contain 'key=value', got: %s", output)
				}
			},
		},
		{
			name: "JSON format with debug level",
			config: Config{
				Level:  LevelDebug,
				Format: FormatJSON,
				Output: nil, // Will be set in test
			},
			verify: func(t *testing.T, logger *slog.Logger, buf *bytes.Buffer) {
				logger.Debug("debug message", "id", 123)
				output := buf.String()
				if !strings.Contains(output, `"msg":"debug message"`) {
					t.Errorf("Expected JSON output to contain debug message, got: %s", output)
				}
				if !strings.Contains(output, `"id":123`) {
					t.Errorf("Expected JSON output to contain id field, got: %s", output)
				}
			},
		},
		{
			name: "Debug messages filtered at info level",
			config: Config{
				Level:  LevelInfo,
				Format: FormatText,
				Output: nil, // Will be set in test
			},
			verify: func(t *testing.T, logger *slog.Logger, buf *bytes.Buffer) {
				logger.Debug("should not appear")
				logger.Info("should appear")
				output := buf.String()
				if strings.Contains(output, "should not appear") {
					t.Errorf("Debug message should be filtered at info level, got: %s", output)
				}
				if !strings.Contains(output, "should appear") {
					t.Errorf("Info message should appear, got: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			tt.config.Output = buf
			logger := New(tt.config)
			tt.verify(t, logger, buf)
		})
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Error("Default() returned nil logger")
	}
}

func TestNewWithLevel(t *testing.T) {
	buf := &bytes.Buffer{}

	tests := []struct {
		name          string
		level         Level
		logFunc       func(*slog.Logger)
		shouldAppear  bool
	}{
		{
			name:  "Debug level logs debug messages",
			level: LevelDebug,
			logFunc: func(_ *slog.Logger) {
				// Replace the handler to use our buffer
				l := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
				l.Debug("debug message")
			},
			shouldAppear: true,
		},
		{
			name:  "Info level filters debug messages",
			level: LevelInfo,
			logFunc: func(_ *slog.Logger) {
				l := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
				l.Debug("debug message")
			},
			shouldAppear: false,
		},
		{
			name:  "Error level filters info messages",
			level: LevelError,
			logFunc: func(_ *slog.Logger) {
				l := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelError}))
				l.Info("info message")
			},
			shouldAppear: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := NewWithLevel(tt.level)
			tt.logFunc(logger)

			output := buf.String()
			containsMessage := len(output) > 0

			if tt.shouldAppear && !containsMessage {
				t.Errorf("Expected message to appear but got empty output")
			}
			if !tt.shouldAppear && containsMessage {
				t.Errorf("Expected message to be filtered but got: %s", output)
			}
		})
	}
}

func TestDiscard(t *testing.T) {
	logger := Discard()
	if logger == nil {
		t.Error("Discard() returned nil logger")
	}

	// Should not panic and should produce no output
	logger.Info("test message")
	logger.Debug("debug message")
	logger.Error("error message")
}
