# GoChess Roadmap: CLI ChessBase

## Vision

GoChess is a personal CLI chess database and analysis tool — a command-line ChessBase. It stores your game history from Chess.com and Lichess, provides engine analysis, identifies weaknesses in your play, and lets you export or share your work.

The roadmap is organized around **workflows** — each milestone delivers a complete, useful answer to a question you'd ask after playing chess.

---

## Current State

The data pipeline is solid: import from Chess.com/Lichess, PGN parsing, SQLite storage with duplicate detection, position extraction, ECO opening classification, and UCI engine communication all work. The gap is in the **experience layer** — the CLI output isn't actionable, filtering is limited, analysis beyond single positions isn't implemented, and there's no way to share results back to platforms.

---

## Milestone 1: "How did I play today?"

**Goal**: After a chess session, run one or two commands and immediately understand how you did.

### Import Summary

Rework `gochess import` to print a session summary after importing:
- Number of games imported
- Win/loss/draw breakdown
- Rating change (if available from platform data)
- Openings played with results

Instead of "imported 12 games", output something like: "5 games imported — 3W 1L 1D, +12 rating in blitz. You played the Caro-Kann 3 times (2W 1L)."

### Enhanced Filtering for `gochess db list`

Add combinable filters:
- `--since` / `--until` — date range filtering (`--since today`, `--since 2024-01-01`)
- `--opening` / `--eco` — opening name or ECO code (`--opening "Sicilian"`, `--eco B20`)
- `--time-control` — bullet, blitz, rapid, classical
- `--result` — win, loss, draw

All filters are combinable: `gochess db list --since today --time-control blitz --result loss`.

### Reworked Stats Output

Redesign `gochess stats` to lead with actionable information:
- Current win/loss streaks
- Worst-performing openings (by win rate, minimum game threshold)
- Performance breakdown by time control
- Less wall-of-numbers, more narrative structure

### Improved Game Display

`gochess db show --id 123` displays a single game with:
- Move list in a readable format
- Opening name and ECO code
- Game metadata (players, ratings, time control, result, date)
- Termination reason

### Dependencies

None — this milestone builds on existing functionality.

---

## Milestone 2: "Where did I go wrong?"

**Goal**: Review games with engine analysis, find mistakes, and get accuracy scores like chess.com's game review.

### Full Game Analysis

`gochess analyze game --id 123` runs Stockfish over an entire game:
- For each move, calculate the evaluation before and after
- Compute centipawn loss per move (delta between best engine eval and played move eval)
- Classify moves based on centipawn loss thresholds:
  - Excellent: 0-10 cp loss
  - Good: 10-25 cp loss
  - Inaccuracy: 25-50 cp loss
  - Mistake: 50-100 cp loss
  - Blunder: 100+ cp loss
- Thresholds should be configurable in settings
- Store all analysis results in the database (don't re-analyze)

### Batch Analysis

`gochess analyze game --last N` analyzes the most recent N games. Progress indicator for long-running analysis. Configurable engine depth (default: depth 18, balancing speed and accuracy).

### Analysis Output

Per-game summary:
- Overall accuracy percentage (based on centipawn loss distribution)
- Count of blunders, mistakes, inaccuracies
- Critical moments: top 3 worst moves with the engine's preferred line and eval difference
- Separate accuracy for opening/middlegame/endgame phases

### Annotated Game Display

When `gochess db show --id 123` is called for an analyzed game:
- Show move quality markers alongside each move
- Display evaluation changes at critical moments
- Highlight the worst moves with engine alternatives

### Stats Integration

`gochess stats` gains a new analysis section:
- Average accuracy by time control
- Accuracy trend over recent games
- Most common mistake types / game phases where mistakes cluster

### Dependencies

- Requires Stockfish (or any UCI engine) installed on the system
- Builds on existing `internal/engine/` UCI integration
- Position storage from import (already implemented)

---

## Milestone 3: "What should I study?"

**Goal**: Identify weaknesses in your repertoire and find patterns across games to focus study time.

### Opening Repertoire View

`gochess repertoire` displays your opening tree:
- Which openings you play as White and Black
- Win rate per opening line
- How deep your preparation goes (where you start deviating or losing accuracy)
- Highlight gaps: openings where you score poorly or face responses you don't handle well
- Filter by color (`--color white`), time control, date range

### Position Search

`gochess db search --fen "rnbqkbnr/..."` finds all games that reached a specific position:
- Aggregate results (W/L/D count and percentages)
- List matching games with basic metadata
- Optionally filter by player color, time control, date range

### Opening Deep-Dive

`gochess stats --opening "Sicilian"` or `--eco B20` drills into a specific opening:
- Your overall record in this opening
- Average accuracy (if games are analyzed)
- Most common opponent responses and your results against each
- Where you tend to go wrong (move number where accuracy drops)

### Game Phase Analysis

Using engine analysis data from Milestone 2, tag performance by game phase:
- Opening phase: moves 1-15 (or until out of book)
- Middlegame: moves 15-40
- Endgame: move 40+

Surface in stats: "Your opening play is strong (avg 3 cp loss) but you lose 15 centipawns on average in endgames."

### Dependencies

- Milestone 2 (engine analysis) for accuracy-based insights
- Position search works without analysis but is less useful
- ECO classification (already implemented)

---

## Milestone 4: "Share my work"

**Goal**: Get games and analysis out of the local database into shareable formats and platforms.

### Bulk Export with Filters

Rework `gochess db export` to support:
- `--all` to export entire database
- Same filter flags as `db list`: `--player`, `--opening`, `--eco`, `--since`, `--until`, `--result`, `--time-control`
- Output to file (`--output games.pgn`) or stdout

### Multiple Export Formats

- `--format pgn` (default) — standard PGN
- `--format json` — structured JSON for programmatic consumption
- `--format csv` — tabular format for spreadsheets

### Annotated PGN Export

`gochess db export --analyzed` produces PGN with engine annotations:
- Evaluation comments on each move (`{ +0.35 }`)
- Move quality NAGs (`?!` for inaccuracy, `??` for blunder, `!` for excellent)
- Engine's preferred line as a variation on critical moves
- Produces standard PGN that any chess tool (ChessBase, Lichess, chess.com) can read

### Lichess Study Integration

`gochess lichess study create`:
- Create a new Lichess study from selected games
- Select games by ID list, filter flags, or opening
- Include analysis annotations if available
- Returns the study URL
- Requires API token (already in config)

`gochess lichess study sync`:
- Update an existing study with newly analyzed games
- Track which games have been pushed to avoid duplicates
- Takes a study ID parameter

### Dependencies

- Milestone 2 for annotated exports
- Lichess API token (already supported in config)
- Lichess study API access

---

## Milestone 5: "Track my progress"

**Goal**: See how you're improving over time across weeks and months.

### Performance Trends

`gochess stats --trend` shows:
- Accuracy over time (weekly or monthly buckets)
- Win rate by month
- Rating progression (from imported game metadata)
- Broken down by time control
- Presented as a text-based chart or structured table

### Period Comparison

`gochess stats --compare` compares two time periods:
- Default: this month vs last month
- Custom: `--compare "2026-01 vs 2026-03"`
- Metrics: win rate, accuracy, opening diversity, blunder frequency, average centipawn loss

### Weakness Aggregation

`gochess stats --weaknesses` surfaces persistent problem areas:
- Openings with consistently poor results (minimum game threshold to avoid noise)
- Game phases where centipawn loss is highest
- Time controls where you underperform
- Draws from analysis data across full history

### Goal Tracking

Lightweight goal system:
- Set goals in config: `gochess config set-goal --accuracy 85 --time-control blitz`
- Goals displayed in stats output with current progress
- Simple threshold-based tracking, not gamification

### Dependencies

- Milestones 1-2 (imported and analyzed games for meaningful data)
- Sufficient game history for trends to be meaningful

---

## Milestone Order and Rationale

```
M1 (UX) --> M2 (Analysis) --> M3 (Study) --> M4 (Share) --> M5 (Trends)
```

- **M1 first** because the tool needs to be pleasant to use before adding features. Every subsequent milestone benefits from better output and filtering.
- **M2 before M3** because repertoire and phase analysis depend on engine evaluation data.
- **M3 before M4** because you need to identify what's interesting before sharing it.
- **M4 before M5** because sharing is more immediately useful than long-term trending.
- **M5 last** because trend data becomes meaningful only after you've been using the tool consistently with analysis.

---

## Out of Scope (Future Considerations)

These are interesting but not part of this roadmap:

- **Tactical pattern detection** — pins, forks, skewers identification
- **Puzzle generation** — generate training puzzles from your blunders
- **Spaced repetition** — opening trainer with review scheduling
- **Chess.com study/export** — chess.com API is more limited than Lichess for write operations
- **Multiplayer/server mode** — this is a personal CLI tool
- **GUI** — the TUI is the display layer; no plans for a graphical interface
