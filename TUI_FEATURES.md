# GoChess TUI (Terminal User Interface) Features

GoChess now includes beautiful, interactive TUI components powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)!

## Features

### 1. Pretty Statistics Display

Use the `--tui` flag or `--format=tui` option with the `stats` command to see colorful, well-formatted statistics:

```bash
# Using the --tui flag
gochess stats --tui

# Using the --format flag
gochess stats --format=tui

# Filter by player
gochess stats --player YourUsername --tui
```

**Features:**
- Color-coded win rates (green for good, red for poor)
- Beautifully formatted tables with alternating row colors
- Performance by color (White/Black statistics)
- Time control breakdown
- Top 10 opening statistics with win rates
- Position frequency analysis

### 2. Interactive Game Browser

Use the `--tui` flag with the `db list` command to browse games interactively:

```bash
# Browse all games
gochess db list --tui

# Filter and browse
gochess db list --white "Magnus Carlsen" --tui
gochess db list --event "Titled Tuesday" --tui
```

**Features:**
- Navigate with arrow keys (↑/↓)
- Search/filter games with `/`
- Press `Enter` to view full game details
- Press `q` to quit or go back
- Beautiful color-coded results (wins, losses, draws)
- Displays ELO ratings, opening names, and ECO codes

### 3. Progress Indicators

Import operations now show progress with spinners and progress bars:

```bash
gochess import
```

**Features:**
- Animated spinner while importing
- Progress bar showing completion percentage
- Real-time error reporting
- Color-coded success/error messages

## Color Scheme

The TUI uses a carefully chosen color palette:

- **Primary (Purple)**: Titles and headers
- **Success (Green)**: Wins and positive stats
- **Warning (Orange)**: Draws and moderate stats
- **Error (Red)**: Losses and poor stats
- **Accent (Cyan)**: Important values and highlights
- **Muted (Gray)**: Help text and less important info

## Keyboard Controls

### Game List Browser
- `↑`/`↓` or `j`/`k`: Navigate up/down
- `/`: Search/filter
- `Enter`: View game details
- `q`: Quit or go back
- `Ctrl+C`: Force quit

### Stats Viewer
- Just displays the output (non-interactive)

## Examples

### View Your Statistics with Pretty Colors
```bash
gochess stats --tui
```

### Browse Your Latest Games Interactively
```bash
gochess db list --limit 50 --tui
```

### View Opening Performance
```bash
gochess stats --player YourName --tui
```

## Implementation Details

The TUI components are located in `internal/tui/`:

- `styles.go`: Color palette and reusable styles
- `stats.go`: Stats rendering functions
- `gamelist.go`: Interactive game list browser
- `progress.go`: Progress indicators and spinners

### Adding TUI to New Commands

To add TUI features to a new command:

1. Import the tui package: `import "github.com/kyleboon/gochess/internal/tui"`
2. Add a `--tui` flag to your command
3. Use the rendering functions from the `tui` package
4. For interactive components, create a Bubble Tea model

Example:
```go
if c.Bool("tui") {
    // Use TUI rendering
    fmt.Println(tui.RenderPlayerStats(stats, count))
} else {
    // Use plain text output
    fmt.Printf("Player: %s\n", stats.Name)
}
```

## Dependencies

The TUI features require these additional dependencies:

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling library
- `github.com/charmbracelet/bubbles` - Pre-built TUI components

These are automatically included when you build from source.

## Future Enhancements

Potential future TUI features:

- [ ] Interactive game replay with board visualization
- [ ] Real-time progress for analysis operations
- [ ] Interactive opening explorer
- [ ] Dashboard view combining stats, games, and positions
- [ ] Customizable color themes
- [ ] Mouse support for clicking on games

## Troubleshooting

### Colors not showing?
Make sure your terminal supports ANSI colors. Most modern terminals do.

### Layout looks broken?
Ensure your terminal window is at least 100 columns wide and 24 rows tall for optimal display.

### TUI not working?
Try running without `--tui` to use the plain text output instead:
```bash
gochess stats  # Plain text output
```
