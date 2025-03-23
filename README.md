# Chess Game Analysis Tool

A command-line application written in Go that analyzes chess games from PGN files using the Stockfish engine. The tool evaluates moves, identifies inaccuracies, mistakes, and blunders, calculates centipawn loss, and outputs an annotated PGN with embedded analysis.

## Features

- Analyze PGN files using Stockfish or other UCI-compatible engines
- Calculate centipawn loss for each move
- Identify and annotate inaccuracies, mistakes, and blunders
- Calculate average centipawn loss for each player
- Identify and annotate brilliant moves
- Generate summary statistics for each player and game

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

- Go 1.21 or later
- Stockfish chess engine (or another UCI-compatible engine)

### Installation

```bash
go install github.com/kyleboon/gochess/cmd/gochess@latest
```

### Usage

```bash
gochess -pgn game.pgn -engine /path/to/stockfish
```

For more options:
```bash
gochess -help
```

## License

TBD