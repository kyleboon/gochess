# Claude Context: GoChess Project

## Project Purpose

GoChess is a chess library and toolset written in Go that provides:
- Chess move generation and validation
- PGN (Portable Game Notation) parsing and management
- FEN (Forsyth-Edwards Notation) support
- SQLite-based game database with import/export capabilities
- Chess.com API integration for downloading game archives
- Lichess API integration for downloading game archives
- CLI tools for game management and analysis

**Target Users**: Chess enthusiasts, developers building chess applications, analysts working with chess databases.

---

## Project Architecture

### Core Design Principles

1. **Separation of Concerns**: Chess logic, database, API client, and UI are in separate packages
2. **Idiomatic Go**: Follows Go best practices, uses standard library where possible
3. **Testing First**: Comprehensive test coverage with table-driven tests
4. **Context-Aware**: All I/O operations accept `context.Context` for cancellation/timeout
5. **Structured Logging**: Uses `log/slog` for observable, parseable logs

### Package Structure

```
gochess/
├── cmd/
│   └── gochess/        # Main CLI application
├── internal/
│   ├── board.go        # Chess board representation and logic
│   ├── move.go         # Move parsing and notation (algebraic, UCI)
│   ├── move_gen.go     # Legal move generation
│   ├── fen.go          # FEN parsing and validation
│   ├── perft.go        # Performance testing for move generation
│   ├── db/             # SQLite database layer
│   ├── pgn/            # PGN parsing and database
│   ├── chesscom/       # Chess.com API client
│   ├── lichess/        # Lichess API client
│   └── logging/        # Structured logging configuration
├── testdata/           # Test fixtures (PGN files, FEN positions)
└── README.md
```

---

## Key Architectural Decisions

### 1. **Board Representation**
- Uses bitboards for piece locations (64-bit integers)
- Efficient for move generation and position evaluation
- Pieces stored as: `WP`, `WN`, `WB`, `WR`, `WQ`, `WK` (white), `BP`, `BN`, etc. (black)

### 2. **Database Design**
- SQLite for portability and zero-config
- Two tables: `games` (main data) and `tags` (metadata)
- Game hash (`game_hash`) for duplicate detection
- Stores complete PGN text for perfect round-tripping

**Schema Highlights**:
```sql
games: id, event, site, date, white, black, result, white_elo, black_elo,
       time_control, pgn_text, game_hash, created_at

tags: id, game_id, tag_name, tag_value
```

### 3. **Context Propagation**
- All HTTP requests accept `context.Context`
- All database operations accept `context.Context`
- Enables graceful cancellation and timeout handling

### 4. **Error Handling**
- Errors wrapped with `fmt.Errorf("...: %w", err)` for stack traces
- Custom error types: `PGNImportError` for import failures with context
- Never panic in library code; return errors

### 5. **Logging Strategy**
- **Library code** (internal/*): Uses `log/slog` with structured key-value pairs
- **CLI output** (cmd/*): Uses `fmt.Printf` for user-facing messages
- **Tests**: Use `logging.Discard()` to suppress log noise
- **Levels**: Debug (verbose), Info (operations), Warn (retries), Error (failures)

---

## Important Patterns and Conventions

### Testing Patterns

1. **Table-Driven Tests**: All tests use `tests := []struct{...}` pattern
2. **Descriptive Names**: Test names explain the scenario being tested
3. **Temporary Databases**: Use `os.MkdirTemp()` for isolated test databases
4. **Discard Logger**: `logging.Discard()` prevents test output pollution

**Example**:
```go
func TestValidateGameTags(t *testing.T) {
    tests := []struct {
        name      string
        game      *pgn.Game
        wantError bool
        errorMsg  string
    }{
        {name: "Valid game", game: validGame, wantError: false},
        // ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateGameTags(tt.game)
            // assertions
        })
    }
}
```

### Code Organization

1. **Single Responsibility**: Functions should do one thing
   - Example: `ImportPGN` was 192 lines → split into 4 helper functions
2. **Dependency Injection**: Constructors accept logger/config
   - `NewWithLogger()` for custom loggers
   - `NewClientWithLogger()` for HTTP client
3. **Exported vs Unexported**:
   - Exported: Public API (PascalCase)
   - Unexported: Internal helpers (camelCase)

### Naming Conventions

- **Files**: `snake_case.go` (e.g., `move_gen.go`, `sqlite_test.go`)
- **Packages**: Short, lowercase, no underscores (e.g., `chesscom`, `logging`)
- **Constructors**: `New()` or `NewWithX()` for customization
- **Tests**: `TestFunctionName_Scenario` or `TestFunctionName` with subtests

---

## Current Focus Areas

### Recently Completed (Latest Session)

1. **Lichess Integration**: Complete API client for downloading games from Lichess
2. **CLI Commands**: Added `gochess lichess download` with comprehensive filtering options
3. **Date Range Support**: Lichess uses date ranges (since/until) instead of monthly archives
4. **API Token Support**: Optional authentication for private games and higher rate limits
5. **Comprehensive Tests**: Full test coverage for Lichess client with 8 test cases

### Current State

- **All high-priority code quality issues resolved**
- **Test coverage**: Comprehensive unit tests for core functionality
- **Production-ready**: Context support, logging, error handling, retry logic
- **Known test failures**: `TestImportPGN_WithFEN` (pre-existing, unrelated to recent changes)

### Technical Debt / Future Work

1. **Perft Tests**: Don't catch insufficient material scenarios (edge case)
2. **FEN Validation**: Some invalid FENs in testdata cause test failures
3. **Export All Games**: `db export` only exports single game by ID
4. **Rate Limiting**: Currently no protection against multiple parallel processes hitting Chess.com API

---

## Important Quirks and Gotchas

### 1. **Chess.com API Rate Limiting**
⚠️ **Serial access is unlimited, parallel requests may trigger HTTP 429**
- The client automatically retries with exponential backoff
- Default: 3 retries, starting at 1s, max 30s backoff
- No internal mutex - multiple client instances may conflict
- Solution: Make requests sequentially (current behavior in `--all-history`)

### 1a. **Lichess API Rate Limiting**
⚠️ **~120 requests per minute with API token**
- The client automatically retries with exponential backoff (same as Chess.com)
- Default: 3 retries, starting at 1s, max 30s backoff
- Uses date range filtering instead of monthly archives
- Optional API token for private games and higher rate limits

### 2. **PGN Import Behavior**
- **Duplicate Detection**: Uses hash of moves + metadata
- **FEN Handling**: Custom FENs are ignored; standard starting position assumed
- **Error Handling**: Imports continue even if some games fail
- **Transaction Safety**: All-or-nothing at file level (commit only on success)

### 3. **Database Migration**
- `game_hash` column added via `addColumnIfNotExists()` for backward compatibility
- Existing databases are automatically migrated on first connection
- No explicit migration system; schema changes handled in `createTables()`

### 4. **Board Representation Edge Cases**
- **Bitboard Squares**: 0-63, rank 1 = 0-7, rank 8 = 56-63
- **Piece Constants**: Don't confuse `WB` (white bishop) with `BB` (black bishop)
- **Move Generation**: Does not check for insufficient material (use `HasInsufficientMaterial()`)

### 5. **Testing Quirks**
- Database tests create real SQLite files (not mocked)
- HTTP client tests use `httptest.Server` for realistic simulation
- Perft tests are slow at depth 6+ (use `-short` flag to skip)

### 6. **Logging vs User Output**
```go
// WRONG: Using logger for user-facing messages
logger.Info("Imported 42 games")  // Goes to stderr, not formatted

// RIGHT: Use fmt for CLI output
fmt.Printf("Imported %d games\n", count)

// RIGHT: Use logger for operations/debugging
logger.Info("import completed", "count", count, "errors", len(errs))
```

### 7. **Context Patterns**
```go
// CLI commands: Use c.Context from urfave/cli
func ImportCommand(c *cli.Context) error {
    db.ImportPGN(c.Context, path)  // ✓
}

// HTTP clients: Use context from caller
client.GetPlayerGames(ctx, username, year, month)  // ✓

// Tests: Usually context.Background()
db.ImportPGN(context.Background(), testFile)  // ✓
```

---

## Development Workflow

### Building
```bash
go build ./...                    # Build all packages
go build ./cmd/gochess           # Build main CLI
```

### Testing
```bash
go test ./...                     # Run all tests
go test ./internal/db -v          # Verbose output for specific package
go test -short ./...              # Skip slow tests (perft)
go test -run TestImportPGN ./internal/db  # Run specific test
```

### Running
```bash
# CLI examples
./gochess db import --pgn games.pgn --database ~/.gochess/games.db
./gochess chesscom download --username player --year 2024 --month 12 --import-db
./gochess db list --database ~/.gochess/games.db
```

### Git Conventions
- Descriptive commit messages with context
- Include "Generated with Claude Code" footer for AI-assisted changes
- Co-Authored-By: Claude for AI contributions
- Commit related changes together (not per-file)

---

## Common Tasks

### Adding a New Database Operation

1. Add method to `DB` struct in `internal/db/sqlite.go`
2. Accept `context.Context` as first parameter
3. Use `db.logger` for structured logging
4. Return errors, don't panic
5. Write table-driven tests in `internal/db/sqlite_test.go`
6. Use `logging.Discard()` in tests

### Adding a New Chess.com API Endpoint

1. Define response struct in `internal/chesscom/models.go`
2. Add method to `Client` in `internal/chesscom/client.go`
3. Use `doRequestWithRetry()` for automatic 429 handling
4. Log with `c.logger.Info()` before/after requests
5. Add tests in `internal/chesscom/client_test.go` with `httptest`

### Adding a New Lichess API Endpoint

1. Define request params in `internal/lichess/models.go` (if needed)
2. Add method to `Client` in `internal/lichess/client.go`
3. Use `doRequestWithRetry()` for automatic 429 handling
4. Log with `c.logger.Info()` before/after requests
5. Add tests in `internal/lichess/client_test.go` with `httptest`

### Adding a New CLI Command

1. Define command in `cmd/gochess/main.go`
2. Use urfave/cli/v2 framework
3. User output: `fmt.Printf()` for messages
4. Pass `c.Context` to database/HTTP operations
5. Handle errors gracefully with helpful messages

---

## Key Dependencies

- **CLI Framework**: `github.com/urfave/cli/v2`
- **SQLite Driver**: `github.com/mattn/go-sqlite3` (requires CGO)
- **Testing**: `github.com/stretchr/testify` (assertions)

**Standard Library**:
- `log/slog` - Structured logging
- `context` - Cancellation/timeouts
- `net/http` - HTTP client
- `database/sql` - Database abstraction

---

## Performance Characteristics

- **Move Generation**: ~1-2 million positions/second (depth 5 perft)
- **PGN Import**: ~1000-5000 games/second (depends on game length)
- **Database Queries**: Fast with proper indexing (indexed on white, black, date, event)
- **Chess.com API**: Rate limited; sequential requests are unlimited

---

## Quick Reference: File Locations

| Need to... | Look in... |
|------------|-----------|
| Modify board logic | `internal/board.go` |
| Add move generation | `internal/move_gen.go` |
| Parse PGN files | `internal/pgn/parse.go` |
| Database operations | `internal/db/sqlite.go` |
| Chess.com API | `internal/chesscom/client.go` |
| Lichess API | `internal/lichess/client.go` |
| Logging config | `internal/logging/logger.go` |
| CLI commands | `cmd/gochess/main.go` |
| Test fixtures | `testdata/` |

---

## Version Information

- **Go Version**: 1.23+ required (uses new features)
- **Toolchain**: go1.24.1 or later
- **Platform**: Cross-platform (tested on macOS, should work on Linux/Windows)
- **CGO**: Required for SQLite (mattn/go-sqlite3)

---

## Getting Help

- **Code Comments**: Most complex functions have detailed comments
- **Tests as Documentation**: See `*_test.go` files for usage examples
- **This File**: Update when making architectural changes
- **Git History**: Well-documented commits explain "why" behind changes

---

*Last Updated*: 2025-12-09 (after adding Lichess integration)
