# GoChess TODO List

## 🔥 High Priority - Core Features

### Position-Based Search
- [ ] Add `positions` table to database schema
  - Columns: `id`, `game_id`, `move_number`, `fen`, `evaluation`, `next_move`
  - Index on FEN for fast lookups
  - Store positions at every move or key positions only?
- [ ] Extend PGN import to parse moves and generate positions
  - Walk through game moves and generate FEN for each position
  - Store positions during import process
- [ ] Implement position search command
  - `gochess db search-position --fen "rnbqkbnr/pp1ppppp/8/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2"`
  - Return all games that reached this position
- [ ] Add position-specific statistics
  - Win/loss/draw rates from specific positions
  - "After 1.e4 c5 2.Nf3, I win 60%, draw 30%, lose 10%"

### Opening Classification & Search
- [ ] Research ECO code database/classification system
  - Option 1: Embed ECO opening database (JSON/CSV)
  - Option 2: Use lichess opening explorer API
  - Option 3: Classify by move sequences (custom)
- [ ] Add `opening_eco`, `opening_name`, `opening_variation` columns to `games` table
- [ ] Implement opening detection during import
  - Match first 8-12 moves against ECO database
  - Store opening classification in database
- [ ] Add opening search/filter to `list` command
  - `gochess db list --opening "Sicilian Defense"`
  - `gochess db list --eco "B20"`
- [ ] Add opening statistics to `stats` command
  - Performance by opening (win/loss/draw rates)
  - Most/least successful openings
  - Opening frequency distribution

### Lichess Integration
- [x] Research Lichess API endpoints
  - Games export endpoint: `https://lichess.org/api/games/user/{username}`
  - Uses date range filtering instead of monthly archives
  - Rate limiting: ~120 requests/minute with API token
- [x] Create `internal/lichess` package
  - Mirror structure of `internal/chesscom`
  - Client with logger and retry logic
  - Support for downloading game archives
- [x] Implement Lichess API client
  - `GetPlayerGamesPGN()` - Download games in PGN format
  - Supports date range filtering (since/until timestamps)
  - Handle authentication (API token support for private games)
  - Implement rate limiting/retry (429 detection like Chess.com)
- [x] Add Lichess CLI commands
  - `gochess lichess download --username <user>` with date range and filter options
  - Supports `--since`, `--until`, `--max`, `--vs`, `--rated`, `--perf-type`, `--color`
  - Supports `--import-db` for direct database import
  - Supports `--api-token` for authenticated requests
- [ ] Add Lichess support to TUI (optional)
  - Similar to Chess.com browser
  - Switch between Chess.com/Lichess sources

---

## 🎯 Medium Priority - Analysis & Statistics

### Advanced Game Analysis
- [ ] Integrate Stockfish engine (or other UCI engines)
  - Research UCI protocol implementation in Go
  - Option 1: Shell out to stockfish binary
  - Option 2: Use notnil/chess (has UCI support)
- [ ] Add `analysis` table to database
  - Columns: `id`, `game_id`, `move_number`, `evaluation`, `best_move`, `mate_in`, `depth`
- [ ] Implement analysis command
  - `gochess analyze --game-id 123 --engine stockfish --depth 20`
  - Store evaluations in database
- [ ] Calculate accuracy scores
  - Compare moves played vs engine recommendations
  - Store accuracy percentage per game
- [ ] Detect mistakes/blunders/inaccuracies
  - Define thresholds (centipawn loss)
  - Tag moves in analysis table

### Advanced Filtering & Search
- [ ] Implement multi-criteria search
  - Combine player, opening, date range, result, ELO range
  - `gochess db search --white "Player" --opening "Sicilian" --result "1-0" --min-elo 1800`
- [ ] Add time control filtering
  - Filter by bullet/blitz/rapid/classical
- [ ] Add result-based filtering improvements
  - Filter by termination reason (checkmate, resignation, timeout, etc.)

### Opening Repertoire Analysis
- [ ] Build opening tree from player's games
  - Visualize what openings are played and how often
  - Identify repertoire gaps
- [ ] Generate repertoire statistics
  - Success rate by variation depth
  - Comparison with database/master games
- [ ] Suggest repertoire improvements
  - "You've never played the Najdorf variation of the Sicilian"

---

## 🔧 Medium Priority - Enhancements

### Export Improvements
- [ ] Implement export all games (not just by ID)
  - `gochess db export --output all-games.pgn`
  - Support filtering during export
- [ ] Add export by position
  - `gochess db export --position-fen "..." --output sicilian-games.pgn`
- [ ] Add export by opening
  - `gochess db export --opening "Sicilian Defense" --output sicilian.pgn`
- [ ] Support multiple export formats
  - PGN (existing)
  - JSON
  - CSV (for spreadsheet analysis)

### TUI Enhancements
- [ ] Add game replay mode with analysis overlay
  - Show engine evaluation as you step through moves
  - Highlight mistakes/blunders
- [ ] Add opening tree visualization in TUI
- [ ] Add position search from TUI
- [ ] Improve Chess.com/Lichess browser UX
  - Better loading states
  - Progress indicators for bulk downloads

### Performance Optimizations
- [ ] Add database indexes for common queries
  - Composite index on (white, black) for player searches
  - Index on opening_eco for opening searches
  - Full-text search index on event/site?
- [ ] Optimize PGN import for large files
  - Stream parsing instead of loading entire file
  - Batch inserts for better performance
- [ ] Add import progress tracking
  - Progress bar for large imports
  - ETA calculation

---

## 🐛 Bugs & Technical Debt

### Known Issues
- [ ] Fix `TestImportPGN_WithFEN` test failure
  - Currently expects 0 errors but gets FEN validation error
  - Either fix FEN validation or update test expectations
  - Related to: `testdata/invalid_fen.pgn`
- [ ] Perft tests don't catch insufficient material scenarios
  - Add specific test cases for KvK+minor piece endgames
  - Possibly extend perft to validate game state

### Code Quality Improvements
- [ ] Add more comprehensive error messages for PGN import failures
  - Include line numbers and context
  - Better error categorization
- [ ] Improve database migration strategy
  - Current approach uses `addColumnIfNotExists()` which is fragile
  - Consider proper migration system (e.g., golang-migrate)
- [ ] Add configuration file support
  - Store default database path, API keys, engine paths
  - `~/.gochess/config.yaml` or similar

---

## 💡 Nice to Have - Future Features

### Tactical Pattern Detection
- [ ] Implement pattern recognition
  - Pins, forks, skewers, discovered attacks
  - Sacrifices, deflections, decoys
- [ ] Add pattern search
  - "Find all games where I won material with a knight fork"
- [ ] Generate tactical puzzles from user's games
  - Extract critical positions
  - Create training mode

### Social & Sharing
- [ ] Export games with annotations
  - Add commentary to exported PGN
  - Include engine analysis as variations
- [ ] Generate game summaries/reports
  - "Your November 2024 Chess Report"
  - Statistics, best games, worst blunders
- [ ] Share games via web service
  - Generate shareable links
  - Lichess study integration?

### Study Mode
- [ ] Create study collections
  - Group games by theme, opening, or purpose
  - Add personal notes and tags
- [ ] Spaced repetition for openings
  - Quiz mode for repertoire training
  - Track what you know vs need to review
- [ ] Opening trainer
  - Practice opening lines from your repertoire
  - Randomized positions from your games

### Advanced Analysis
- [ ] Game comparison tool
  - Compare two games side-by-side
  - Identify similar positions/themes
- [ ] Opening preparation helper
  - Analyze opponent's repertoire (if games available)
  - Suggest preparation based on their tendencies
- [ ] Time usage analysis
  - Track time spent per move/phase
  - Identify time trouble patterns

---

## 🏗️ Infrastructure & Tooling

### Testing
- [ ] Add integration tests for end-to-end workflows
  - Full import → search → export cycle
  - API download → database → analysis
- [ ] Add benchmarks for critical paths
  - Move generation performance
  - Database query performance
  - PGN parsing speed
- [ ] Increase test coverage
  - Target: >80% coverage for internal packages
  - Add edge case tests

### Documentation
- [ ] Create user guide / tutorial
  - Getting started guide
  - Common workflows
  - Configuration examples
- [ ] Add API documentation
  - Godoc for all exported functions
  - Usage examples in comments
- [ ] Create video demo/walkthrough
  - Showcase key features
  - Common use cases

### Distribution
- [ ] Create pre-built binaries for releases
  - macOS (arm64, amd64)
  - Linux (amd64, arm64)
  - Windows (amd64)
- [ ] Add Homebrew formula (macOS)
- [ ] Add installation script
- [ ] Create Docker image
  - Include Stockfish for analysis
  - Pre-configured environment

---

## 📊 Priority Matrix

### Immediate Next Steps (Start Here)
1. **Lichess Integration** - Expands game import sources
2. **Opening Classification** - Enables opening-based search and stats
3. **Position Storage** - Foundation for position-based features

### After Foundation is Complete
4. **Position Search** - Leverage position storage
5. **Opening Statistics** - Leverage opening classification
6. **Export Improvements** - Make data more accessible

### Later Enhancements
7. **Engine Integration** - Adds analysis capabilities
8. **TUI Improvements** - Better user experience
9. **Advanced Features** - Tactical patterns, study mode, etc.

---

## 🤝 Contributing

Want to contribute? Pick an item from this list, assign it to yourself, and submit a PR!

**High Impact, Lower Effort:**
- Lichess API integration (mirror Chess.com structure)
- Opening classification (ECO database integration)
- Export all games functionality

**High Impact, Higher Effort:**
- Position storage and search
- Engine integration (Stockfish UCI)
- Opening tree visualization

---

## 📝 Notes

- This list is a living document - add items as they come up
- Mark items complete with `[x]` when done
- Link to related issues/PRs where applicable
- Update CLAUDE_CONTEXT.md when architecture changes

---

*Last Updated*: 2025-12-09
