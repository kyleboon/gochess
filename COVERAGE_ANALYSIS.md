# GoChess Test Coverage Analysis

**Generated:** 2025-12-12
**Last Updated:** 2025-12-12 (after API client tests)
**Overall Coverage:** 38.8% of statements (was 34.0% → 36.5% → 38.8%)

## Summary by Package

| Package | Coverage | Status | Priority |
|---------|----------|--------|----------|
| internal/logging | 90.5% | ✓ Excellent | Low |
| internal (core) | 77.3% | ✓ Good | Low |
| internal/rampart | 62.5% | ✓ Acceptable | Low |
| internal/pgn | 53.1% | ~ Needs work | Medium |
| internal/db | 53.4% | ✓ Acceptable | Medium |
| internal/lichess | 32.8% | ~ Improved | Medium |
| internal/config | 21.4% | ~ Improved | Medium |
| internal/chesscom | 18.8% | ~ Improved | Medium |
| cmd/gochess | 0.0% | ✗ No tests | Medium |

---

## High Priority Gaps

### 1. internal/db (48.3% coverage)

**Well Tested:**
- ✓ ImportPGN and position storage
- ✓ GetPlayerStats and GetPositionStats
- ✓ Position extraction (ExtractPositions)
- ✓ Hash calculation and validation

**Missing Tests:**
- ✗ ParsePGNFile - file parsing adapter
- ✗ All CLI command functions (db_cmd.go)
  - ImportCommand, ListCommand, ShowCommand
  - ExportCommand, ClearCommand

**Completed:**
1. ✅ Added tests for GetGameByID (retrieval, not found, multiple games)
2. ✅ Added tests for ClearGames (empty db, clearing data, cascade deletion, re-import)
3. ✅ Fixed bug in GetGameByID (missing game_hash in Scan)

**Recommended Actions:**
1. Add integration test for ParsePGNFile
2. CLI commands can remain untested (UI layer)

---

### 2. internal/chesscom (18.8% coverage)

**Well Tested:**
- ✓ HTTP retry logic (doRequestWithRetry - 90%)
- ✓ Client constructors (NewClientWithLogger)
- ✓ RetryConfig
- ✓ GetPlayerGames - API method with httptest ✅
- ✓ GetPlayerGamesPGN - PGN download ✅
- ✓ GetArchivedMonths - archive listing ✅

**Missing Tests:**
- ✗ All converter functions (GamesToDatabase, PGNToDatabase)
- ✗ All CLI functions (chesscom_cmd.go)
- ✗ ImportFromConfig - config-based import

**Completed:**
1. ✅ Added integration tests with httptest for all API methods (GetPlayerGames, GetPlayerGamesPGN, GetArchivedMonths)
2. ✅ Enhanced Client with baseURL field for testing

**Recommended Actions:**
1. Test converter functions with sample game data
2. Test ImportFromConfig workflow
3. CLI functions can remain untested

---

### 3. internal/lichess (32.8% coverage)

**Well Tested:**
- ✓ HTTP retry logic (doRequestWithRetry - 90%)
- ✓ Client constructors
- ✓ Query parameter building (buildQueryParams - 88.9%)
- ✓ GetPlayerGamesPGN - main download method ✅
- ✓ Date range filtering ✅
- ✓ API token authentication ✅
- ✓ Filter parameters (rated, perfType, color, max) ✅
- ✓ Rate limit retry (429 handling) ✅

**Missing Tests:**
- ✗ All CLI functions (lichess_cmd.go)
- ✗ ImportFromConfig - config-based import

**Completed:**
1. ✅ Added comprehensive integration tests with httptest for GetPlayerGamesPGN
2. ✅ Enhanced Client with baseURL field for testing
3. ✅ Tests cover: basic params, date ranges, API tokens, filters, empty responses, 404s, 429s

**Recommended Actions:**
1. Test ImportFromConfig workflow
2. CLI functions can remain untested

---

### 4. internal/config (14.3% coverage)

**Well Tested:**
- ✓ GetLastImport and SetLastImport (100%)
- ✓ Basic Load/Save operations (70-75%)

**Missing Tests:**
- ✗ All CLI commands (commands.go)
  - InitCommand, ShowCommand
  - AddUserCommand, RemoveUserCommand

**Completed:**
1. ✅ Added tests for LoadOrDefault (existing config, default when missing)
2. ✅ Added tests for SaveDefault (creates directory, saves correctly)
3. ✅ Added tests for ClearAllLastImports (clears all, can set after)
4. ✅ Fixed bug in LoadOrDefault (error wrapping with %w for errors.Is)

**Recommended Actions:**
1. CLI commands can remain untested

---

## Medium Priority Gaps

### 5. internal/pgn (53.1% coverage)

**Well Tested:**
- ✓ Core parsing (readGame - 82.9%)
- ✓ Token parsing (parseMoves - 100%)
- ✓ Lexer (most functions 100%)

**Missing Tests:**
- ✗ variation() function - 45% coverage
- ✗ nag() function - NAG annotations (0%)
- ✗ Several game tree methods (Plies, NewVariation, Variations, IsRoot)
- ✗ NAG-related methods (AddNag, DropNag)

**Recommended Actions:**
1. Add tests for variation handling
2. Add tests for NAG (Numeric Annotation Glyph) support
3. Game tree methods may not be critical if unused

---

### 6. cmd/gochess (0% coverage)

**Note:** CLI commands are typically not unit tested. Integration/E2E tests are more appropriate but not critical.

**If Testing Desired:**
- Test main command routing
- Test flag parsing
- Test error handling for invalid inputs

**Recommendation:** Low priority - CLI is a thin layer over tested library code.

---

## Low Priority Gaps

### 7. internal/rampart (62.5% coverage)

**Note:** Test data utilities. Current coverage is acceptable.

---

## Recommended Testing Priorities

### Immediate (High Value, Low Effort) ✅ COMPLETED
1. ~~**internal/db/GetGameByID** - Simple SELECT test~~ ✅ Done
2. ~~**internal/db/ClearGames** - Database clearing test~~ ✅ Done
3. ~~**internal/config file operations** - LoadOrDefault, SaveDefault~~ ✅ Done

### Short Term (High Value, Medium Effort) ✅ COMPLETED
4. ~~**internal/chesscom API methods** - Integration tests with httptest~~ ✅ Done
5. ~~**internal/lichess API methods** - Integration tests with httptest~~ ✅ Done
6. **internal/chesscom/lichess converters** - Data conversion tests

### Medium Term (Medium Value, Higher Effort)
7. **internal/config ImportFromConfig** - End-to-end workflow tests
8. **internal/pgn variations** - PGN game tree handling
9. **internal/db ParsePGNFile** - File parsing integration

### Lower Priority
10. CLI command functions (low ROI for testing effort)

---

## Coverage Goals

**Current:** 38.8% (was 34.0% → 36.5% → 38.8%)
**Quick Wins Progress:** +2.5% (db & config tests)
**API Client Tests Progress:** +2.3% (chesscom & lichess tests)
**Total Improvement:** +4.8%
**Target:** 60%+ (achievable with remaining recommendations)

**Breakdown:**
- Core libraries (internal/*): Target 70%+ ✓ (currently 77.3%)
- API clients (chesscom/lichess): Target 50%+ (currently 32.8%/18.8%)
- Database (internal/db): Target 65%+ (currently 53.4%, close!)
- CLI: Remain at 0% (acceptable)

---

## Testing Strategy

### For API Clients (chesscom/lichess)
```go
func TestClient_GetPlayerGames(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return mock response
    }))
    defer server.Close()

    client := NewClientWithLogger(server.URL, logging.Discard())
    // Test API call
}
```

### For Database Operations
```go
func TestGetGameByID(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Import test game
    // Retrieve by ID
    // Verify fields
}
```

### For Config Operations
```go
func TestLoadOrDefault(t *testing.T) {
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yaml")

    // Test with missing file
    // Test with existing file
    // Test with invalid YAML
}
```

---

## Conclusion

The project has good coverage of core chess logic (77.3%) and excellent coverage of logging (90.5%).

**Recent Progress:**
- Overall coverage improved from 34.0% → 38.8% (+4.8%)
- Chess.com client: 5.4% → 18.8% (+13.4%)
- Lichess client: 18.4% → 32.8% (+14.4%)
- Database operations: Added GetGameByID and ClearGames tests
- Config operations: Added LoadOrDefault, SaveDefault, ClearAllLastImports tests

**Remaining gaps:**
1. **API integration layers** (chesscom: 18.8%, lichess: 32.8%) - Still need converter function tests
2. **Configuration management** (21.4%) - Need ImportFromConfig tests
3. **Database query functions** (53.4%) - Close to target, may need ParsePGNFile tests

**Next steps** to reach 60%+ coverage:
1. Test converter functions (GamesToDatabase, PGNToDatabase)
2. Test ImportFromConfig workflow (end-to-end)
3. Additional PGN variation handling tests
