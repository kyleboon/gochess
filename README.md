# GoChess

A comprehensive chess library and toolset written in Go that helps you manage, analyze, and study your chess games.

## Features

### Game Management
- **Automatic Import**: Download and import games from Chess.com and Lichess with a single command
- **Smart Tracking**: Automatically fetches only new games since your last import
- **SQLite Database**: Store and query thousands of games efficiently
- **PGN Support**: Import, export, and manage PGN files

### Analysis (Coming Soon)
- Analyze PGN files using Stockfish or other UCI-compatible engines
- Calculate centipawn loss for each move
- Identify and annotate inaccuracies, mistakes, and blunders
- Generate summary statistics for each player and game

### Chess Engine
- Full chess move generation and validation
- FEN (Forsyth-Edwards Notation) support
- Legal move detection and position evaluation

## Project Structure

- `cmd/`: Main applications
  - `gochess/`: The chess analysis tool executable
- `internal/`: Private application code
  - `pgn/`: PGN parsing and annotation
  - `engine/`: UCI engine communication
  - `analysis/`: Game analysis logic
- `pkg/`: Library code that may be used by external applications

## Getting Started

### Prerequisites

- Go 1.23 or later
- (Optional) Stockfish chess engine for analysis features

### Installation

```bash
go install github.com/kyleboon/gochess/cmd/gochess@latest
```

Or build from source:

```bash
git clone https://github.com/kyleboon/gochess.git
cd gochess
go build ./cmd/gochess
```

## Quick Start

### 1. Initialize Configuration

Set up your configuration with usernames for automatic game import:

```bash
gochess config init
```

This will interactively prompt you for:
- Database location (default: `~/.gochess/games.db`)
- Chess.com username (optional)
- Lichess username and API token (optional)

### 2. Import Your Games

Once configured, importing your games is as simple as:

```bash
gochess import
```

This command will:
- Fetch new games from all configured sources (Chess.com and/or Lichess)
- Only download games since your last import (incremental updates)
- Store them in your local database
- Track the import time for future runs

Run it daily, weekly, or whenever you want to update your game collection!

### 3. Explore Your Games

List games in your database:

```bash
# List recent games
gochess db list

# Filter by player
gochess db list --white "YourUsername"

# Show a specific game
gochess db show --id 123

# Export games to PGN
gochess db export --output games.pgn
```

Get statistics:

```bash
# Overall statistics
gochess db stats

# Stats for a specific player
gochess db stats --player "YourUsername"
```

## Advanced Usage

### Configuration Management

```bash
# Show current configuration
gochess config show

# Add a user
gochess config add-user --platform lichess --username your-username

# Remove a user
gochess config remove-user --platform chesscom
```

### Import Options

```bash
# Force full re-import (ignore last import time)
gochess import --full

# Show detailed errors during import
gochess import --verbose
```

### Manual Downloads

You can still use the platform-specific commands for more control:

```bash
# Chess.com: Download specific month
gochess chesscom download --username player --year 2024 --month 12

# Chess.com: Download all history
gochess chesscom download --username player --all-history --import-db

# Lichess: Download with date range
gochess lichess download --username player --since 2024-01-01 --import-db

# Lichess: Download with filters
gochess lichess download --username player --perf-type blitz --rated true
```

### Database Operations

```bash
# Import PGN files directly
gochess db import --pgn games.pgn

# Clear database
gochess db clear

# List with pagination
gochess db list --limit 50 --offset 100
```

## Configuration File

The configuration is stored at `~/.gochess/config.yaml`:

```yaml
database_path: /Users/you/.gochess/games.db
chesscom:
  username: your-chesscom-username
lichess:
  username: your-lichess-username
  api_token: your-optional-api-token
last_import:
  chesscom:your-chesscom-username: 2024-12-09T10:30:00Z
  lichess:your-lichess-username: 2024-12-09T10:31:15Z
```

You can edit this file manually or use the `gochess config` commands.

## License

TBD