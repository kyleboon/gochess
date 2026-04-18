package engine

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
)

// Engine manages a UCI chess engine process.
type Engine struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	scan   *bufio.Scanner
	mu     sync.Mutex
	logger *slog.Logger
}

// Options holds UCI engine options to set after initialization.
type Options struct {
	Threads int
	Hash    int // hash table size in MB
}

// New starts a UCI engine process and waits for "uciok".
func New(ctx context.Context, path string, logger *slog.Logger) (*Engine, error) {
	return NewWithOptions(ctx, path, logger, Options{})
}

// NewWithOptions starts a UCI engine process, sets options, and waits for "uciok".
func NewWithOptions(ctx context.Context, path string, logger *slog.Logger, opts Options) (*Engine, error) {
	cmd := exec.CommandContext(ctx, path)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("engine stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("engine stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("engine start: %w", err)
	}

	e := &Engine{
		cmd:    cmd,
		stdin:  stdinPipe,
		scan:   bufio.NewScanner(stdoutPipe),
		logger: logger,
	}

	// Send "uci" and wait for "uciok"
	if err := e.send("uci"); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	if _, err := e.readUntil(ctx, "uciok"); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("engine did not respond with uciok: %w", err)
	}

	// Set options
	if opts.Threads > 0 {
		if err := e.SetOption("Threads", fmt.Sprintf("%d", opts.Threads)); err != nil {
			_ = cmd.Process.Kill()
			return nil, err
		}
	}
	if opts.Hash > 0 {
		if err := e.SetOption("Hash", fmt.Sprintf("%d", opts.Hash)); err != nil {
			_ = cmd.Process.Kill()
			return nil, err
		}
	}

	// Wait for engine to be ready
	if err := e.IsReady(ctx); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}

	return e, nil
}

// NewFromStreams creates an Engine from pre-existing streams (for testing).
// The caller is responsible for providing a writer that the engine reads from
// and a reader that the engine writes to.
func NewFromStreams(stdin io.WriteCloser, stdout io.Reader, logger *slog.Logger) *Engine {
	return &Engine{
		stdin:  stdin,
		scan:   bufio.NewScanner(stdout),
		logger: logger,
	}
}

// Close sends "quit" and waits for the engine process to exit.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_ = e.sendLocked("quit")
	_ = e.stdin.Close()

	if e.cmd != nil {
		return e.cmd.Wait()
	}
	return nil
}

// IsReady sends "isready" and waits for "readyok".
func (e *Engine) IsReady(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.sendLocked("isready"); err != nil {
		return err
	}
	_, err := e.readUntilLocked(ctx, "readyok")
	return err
}

// SetOption sends a "setoption" command to the engine.
func (e *Engine) SetOption(name, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	cmd := fmt.Sprintf("setoption name %s value %s", name, value)
	return e.sendLocked(cmd)
}

// send writes a command to the engine's stdin (acquires lock).
func (e *Engine) send(cmd string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sendLocked(cmd)
}

// sendLocked writes a command to the engine's stdin (caller must hold lock).
func (e *Engine) sendLocked(cmd string) error {
	e.logger.Debug("engine send", "cmd", cmd)
	_, err := fmt.Fprintf(e.stdin, "%s\n", cmd)
	if err != nil {
		return fmt.Errorf("engine send %q: %w", cmd, err)
	}
	return nil
}

// readUntil reads lines until one starts with the given prefix (acquires lock).
// Returns all lines read including the matching line.
func (e *Engine) readUntil(ctx context.Context, prefix string) ([]string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.readUntilLocked(ctx, prefix)
}

// readUntilLocked reads lines until one starts with the given prefix (caller must hold lock).
func (e *Engine) readUntilLocked(ctx context.Context, prefix string) ([]string, error) {
	var lines []string
	for {
		select {
		case <-ctx.Done():
			return lines, ctx.Err()
		default:
		}

		if !e.scan.Scan() {
			if err := e.scan.Err(); err != nil {
				return lines, fmt.Errorf("engine read: %w", err)
			}
			return lines, fmt.Errorf("engine: unexpected EOF waiting for %q", prefix)
		}

		line := e.scan.Text()
		e.logger.Debug("engine recv", "line", line)
		lines = append(lines, line)

		if strings.HasPrefix(line, prefix) {
			return lines, nil
		}
	}
}
