# Chess Game Analysis Tool Specification

## Overview
A command-line application written in Go that analyzes chess games from PGN files using the Stockfish engine (or other UCI-compatible engines). The tool evaluates moves, identifies inaccuracies, mistakes, and blunders, calculates centipawn loss, and outputs an annotated PGN with embedded analysis.

## Core Requirements

### Input/Output
- **Input**: PGN file (single or multi-game)
- **Output**: Annotated PGN with embedded analysis to standard output
- **Output Format**: Standard PGN with annotations embedded

### Analysis Features
- Calculate centipawn loss for each move
- Identify and annotate inaccuracies, mistakes, and blunders
- Calculate average centipawn loss for each player
- Identify and annotate brilliant moves
- Generate summary statistics for each player and game

### Engine Configuration
- Default to Stockfish, with ability to use other UCI-compatible engines
- Configurable engine path via command-line option
- Default to single-threaded analysis with options to use multiple cores
- Dynamic analysis timing similar to Lichess (minimum depth, time per move, additional time for critical positions)

### Move Classification Thresholds (Default Lichess-style)
- Inaccuracy: ~20-50 centipawn loss
- Mistake: ~50-100 centipawn loss
- Blunder: >100 centipawn loss
- All thresholds configurable via command-line options

## Technical Specifications

### Command Line Interface
- Go-style flags (e.g., `-depth 20`)
- Support for standard input/output redirection

### Required Parameters
- `-pgn`: Path to PGN file (required)
- `-engine`: Path to UCI chess engine executable (optional, default: search in PATH)

### Optional Parameters
- `-depth`: Minimum analysis depth (optional, default: 18)
- `-time`: Maximum time per move in seconds (optional, default: 3)
- `-threads`: Number of CPU threads (optional, default: 1)
- `-inaccuracy`: Centipawn threshold for inaccuracies (optional, default: 20)
- `-mistake`: Centipawn threshold for mistakes (optional, default: 50)
- `-blunder`: Centipawn threshold for blunders (optional, default: 100)
- `-log`: Log level (info, debug, trace) (optional, default: info)

### Engine Communication
- Implement UCI protocol for engine communication
- Set appropriate UCI options for Stockfish

### PGN Processing
- Handle both single-game and multi-game PGN files
- Preserve original PGN headers
- Add new analysis-related headers
- Replace any existing annotations with engine analysis

### Analysis Process
1. Parse PGN file
2. Initialize engine
3. For each game:
   a. Process moves sequentially
   b. For each position:
      i. Set position in engine
      ii. Run analysis with configured depth/time
      iii. Record evaluation and best line
      iv. Calculate centipawn loss from previous position
      v. Classify move quality (inaccuracy, mistake, blunder, brilliant)
   c. Generate summary statistics
   d. Add annotations and statistics to PGN
4. Output annotated PGN to standard out

### Progress Visualization
- Display current move being analyzed
- Show overall percentage complete

### Annotation Format
- For normal moves: Brief evaluation
- For inaccuracies, mistakes, and blunders:
  - Engine evaluation score
  - Categorization
  - Best engine line
- For brilliant moves: Recognition and evaluation
- Include up to 3 equally strong alternative lines when appropriate

### Summary Statistics (PGN Headers)
- `[WhiteInaccuracies "X"]`
- `[WhiteMistakes "X"]`
- `[WhiteBlunders "X"]`
- `[WhiteAverageCentipawnLoss "X.X"]`
- `[BlackInaccuracies "X"]`
- `[BlackMistakes "X"]`
- `[BlackBlunders "X"]`
- `[BlackAverageCentipawnLoss "X.X"]`
- `[WhiteBrilliantMoves "X"]`
- `[BlackBrilliantMoves "X"]`

## Error Handling

### PGN Parsing Errors
- Exit with clear error messages for corrupt PGN files
- Validate PGN structure before beginning analysis

### Engine Errors
- Retry engine communication up to 3 times on failure
- Exit with appropriate error message if engine fails to respond within configured time
- Provide clear error messages for engine initialization failures

### Logging
- Implement tiered logging (info, debug, trace)
- Log engine communication details at trace level
- Log analysis progress at info level
- Log errors at all levels

## Architecture

### Components
1. **PGN Parser**: Read and validate PGN files
2. **UCI Interface**: Communicate with the chess engine
3. **Analysis Engine**: Coordinate analysis and evaluation
4. **PGN Annotator**: Add annotations to PGN based on analysis
5. **Command-line Interface**: Parse flags and handle user input
6. **Logger**: Handle logging at different levels

### Data Flow
1. Read PGN from file
2. Parse PGN into internal representation
3. Initialize and configure chess engine
4. For each position, send position to engine and collect evaluation
5. Process evaluations to identify move quality
6. Generate annotations and statistics
7. Output annotated PGN

## Testing Plan

### Unit Tests
- PGN parsing and validation
- UCI communication protocol
- Move evaluation and classification
- Annotation generation
- Command-line argument parsing

### Integration Tests
- End-to-end processing of sample PGN files
- Testing with different engines
- Testing with various flag combinations

### Validation Tests
- Verify analysis quality against known benchmarks
- Compare output to Lichess analysis for consistency

## Future Enhancements (Not for Initial Implementation)
1. Opening theory recognition
2. Move range/position-specific analysis
3. Performance metrics output
4. Win probability percentage
5. Export to CSV or other formats
6. Support for chess variants
7. Batch processing of multiple files

## Development Priorities
1. Implement UCI communication with Stockfish
2. Build PGN parsing and annotation
3. Implement move evaluation and classification
4. Add summary statistics
5. Develop logging and error handling
6. Create progress visualization
7. Optimize performance

This specification provides a comprehensive guide for developing a Go application that analyzes chess games using the Stockfish engine and outputs annotated PGN files with detailed analysis.
