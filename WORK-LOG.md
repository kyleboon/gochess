# Work Log

## Session: Configuration & Import Simplification
**Date:** 2024-12-09

### Goal
Simplify the import workflow by adding configuration management and a unified import command. The goal is to make importing games as simple as running `gochess import` without needing to specify usernames, platforms, or manage date ranges manually.

### Implementation

#### 1. Configuration System (`internal/config/`)
Created a new configuration package to manage persistent settings:

**Files Created:**
- `internal/config/config.go` - Core configuration structures and I/O
- `internal/config/config_test.go` - Comprehensive test coverage
- `internal/config/commands.go` - CLI command implementations

**Key Features:**
- YAML-based configuration at `~/.gochess/config.yaml`
- Stores usernames for Chess.com and Lichess
- Tracks API tokens (optional for Lichess)
- Records last import timestamp per platform/user for incremental updates
- Helper functions for loading, saving, and querying config

**Configuration Structure:**
```yaml
database_path: /Users/you/.gochess/games.db
chesscom:
  username: your-username
lichess:
  username: your-username
  api_token: optional-token
last_import:
  chesscom:your-username: 2024-12-09T10:30:00Z
  lichess:your-username: 2024-12-09T10:31:15Z
```

#### 2. Configuration Commands
Added `gochess config` command suite with subcommands:
- `config init` - Interactive setup wizard
- `config show` - Display current configuration
- `config add-user --platform <p> --username <u>` - Add tracked users
- `config remove-user --platform <p>` - Remove tracked users

#### 3. Smart Import Functions
Enhanced both Chess.com and Lichess packages with config-aware import:

**Lichess (`internal/lichess/lichess_cmd.go`):**
- `ImportFromConfig()` - Imports using config settings
- Uses `since` parameter to fetch only games after last import
- Automatically updates last import timestamp after successful import

**Chess.com (`internal/chesscom/chesscom_cmd.go`):**
- `ImportFromConfig()` - Imports using config settings
- Skips monthly archives before the last import month
- Automatically updates last import timestamp after successful import

#### 4. Unified Import Command
Created `cmd/gochess/import_cmd.go` with unified `gochess import` command:

**Features:**
- Imports from all configured sources automatically
- Incremental by default (only new games)
- `--full` flag for complete re-import
- `--verbose` flag for detailed error output
- Summary statistics after import

**Workflow:**
```bash
# One-time setup
gochess config init

# Daily use - just run this!
gochess import

# Force full re-import
gochess import --full
```

#### 5. Documentation Updates
- Updated `README.md` with Quick Start guide
- Added configuration examples and advanced usage
- Updated `TODO.md` with completed items
- Comprehensive inline documentation in all new code

### Testing
- All existing tests pass
- Added comprehensive tests for config package
- Manual testing of config commands
- Build verification successful

### Changes Summary
- **New Files:** 4 files (`config.go`, `config_test.go`, `commands.go`, `import_cmd.go`)
- **Modified Files:** 5 files (`main.go`, `lichess_cmd.go`, `chesscom_cmd.go`, `README.md`, `TODO.md`)
- **Lines Added:** ~600 lines of production code + tests
- **Backward Compatibility:** All existing commands still work

### User Impact
**Before:**
```bash
# Complex, manual workflow
gochess chesscom download --username player --year 2024 --month 12 --import-db --database ~/.gochess/games.db
gochess lichess download --username player --since 2024-12-01 --import-db --database ~/.gochess/games.db --api-token <token>
```

**After:**
```bash
# One-time setup
gochess config init

# Simple daily workflow
gochess import
```

### Next Steps
- Consider adding automatic scheduled imports (cron-like feature)
- Add import hooks for notifications
- Consider config file validation and migration

---

## Session: Lichess Integration
**Date:** 2024-12-09 (Earlier)

### Goal
Add Lichess API integration to enable downloading and importing games from Lichess.org, mirroring the existing Chess.com integration.

---

## Implementation Steps

### Phase 1: Research & API Understanding
- [ ] Research Lichess API endpoints and authentication
  - Endpoint: `https://lichess.org/api/games/user/{username}`
  - Response format: ndjson (newline-delimited PGN)
  - Parameters: `since`, `until`, `max`, `ongoing`, `rated`
  - Rate limiting: ~120 requests/minute with API token
  - Authentication: Optional `Authorization: Bearer <token>` header

### Phase 2: Core Implementation
- [ ] Create `internal/lichess` package structure
  - `types.go` - Response types (minimal for PGN)
  - `client.go` - HTTP client with retry logic
  - `client_test.go` - Unit tests with httptest

- [ ] Implement Lichess API client
  - `NewClient()` and `NewClientWithLogger()` constructors
  - `GetPlayerGames(ctx, username, since, until)` method
  - `doRequestWithRetry()` for 429 handling
  - Exponential backoff retry logic
  - Context-aware operations
  - Structured logging with `log/slog`

### Phase 3: CLI Integration
- [ ] Add Lichess CLI commands to `cmd/gochess/main.go`
  - `lichess download` command
  - Flags: `--username`, `--since`, `--until`, `--import-db`, `--database`, `--output`
  - Support for `--all-history` flag
  - Auto-import to database when `--import-db` is set

### Phase 4: Testing
- [ ] Write comprehensive unit tests
  - HTTP client with `httptest.Server`
  - Error handling (rate limits, network errors)
  - PGN parsing of Lichess format
  - Duplicate game detection
  - Context cancellation

### Phase 5: Documentation
- [ ] Update documentation
  - Update `CLAUDE_CONTEXT.md` with Lichess section
  - Mark items complete in `TODO.md`
  - Update `README.md` with Lichess examples

### Phase 6: Optional TUI Integration
- [ ] Add Lichess to TUI browser (if desired)
  - Source selector for Chess.com vs Lichess
  - Reuse existing game browser UI

---

## Progress Log

### Session 1 - Core Implementation Complete
- ✅ Created WORK-LOG.md with implementation plan
- ✅ Researched Lichess API endpoints and authentication
- ✅ Created `internal/lichess` package structure
  - `models.go` - GamesParams struct and DefaultGamesParams
  - `client.go` - Client with retry logic and GetPlayerGamesPGN
  - `client_test.go` - Comprehensive test coverage
  - `lichess_cmd.go` - CLI command implementation
- ✅ Implemented Lichess API client with exponential backoff retry
- ✅ Added unit tests (all passing)
  - TestClient_RetryOn429
  - TestClient_RetryContextCancellation
  - TestRetryConfig_ExponentialBackoff
  - TestDefaultRetryConfig
  - TestGetPlayerGamesPGN
  - TestBuildQueryParams
  - TestSetAPIToken
  - TestDefaultGamesParams
- ✅ Created Lichess CLI commands
  - Added `lichess download` command to main.go
  - Supports date range filtering (--since, --until)
  - Supports game filtering (--vs, --rated, --perf-type, --color, --max)
  - Supports direct database import (--import-db)
  - Supports API token authentication (--api-token)
- Next: Update documentation

---

## Notes
- Lichess API is more permissive than Chess.com (no archives, uses date ranges)
- PGN format is directly compatible with existing import functionality
- Follow same patterns as Chess.com client for consistency

---

## Key Differences: Chess.com vs Lichess

| Feature | Chess.com | Lichess |
|---------|-----------|---------|
| Archive Structure | Monthly archives | Date range filtering |
| Response Format | JSON with games array | ndjson (streaming PGN) |
| Rate Limiting | Unknown, serial safe | ~120 req/min with token |
| Authentication | Public only | Optional token for private games |
| API Endpoint | `/pub/player/{user}/games/{YYYY}/{MM}` | `/api/games/user/{user}?since={ts}` |

---

*Last Updated*: 2025-12-09
