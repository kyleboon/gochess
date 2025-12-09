# Work Log: Lichess Integration

## Date: 2025-12-09

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
