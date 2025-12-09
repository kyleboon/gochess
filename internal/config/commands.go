package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// InitCommand initializes a new configuration file interactively
func InitCommand(c *cli.Context) error {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config file already exists at %s\n", configPath)
		fmt.Print("Do you want to overwrite it? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborting.")
			return nil
		}
	}

	// Get default database path
	dbPath, err := DefaultDatabasePath()
	if err != nil {
		return err
	}

	fmt.Println("\n=== GoChess Configuration Setup ===")
	fmt.Printf("Default database path: %s\n", dbPath)
	fmt.Print("Press Enter to use default, or enter a custom path: ")

	reader := bufio.NewReader(os.Stdin)
	customPath, _ := reader.ReadString('\n')
	customPath = strings.TrimSpace(customPath)
	if customPath != "" {
		dbPath = customPath
	}

	cfg := &Config{
		DatabasePath: dbPath,
		LastImport:   make(map[string]time.Time),
	}

	// Ask about Chess.com
	fmt.Print("\nDo you want to track Chess.com games? (y/N): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		fmt.Print("Enter your Chess.com username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		if username != "" {
			cfg.ChessCom = &ChessComConfig{
				Username: username,
			}
		}
	}

	// Ask about Lichess
	fmt.Print("\nDo you want to track Lichess games? (y/N): ")
	response, _ = reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		fmt.Print("Enter your Lichess username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		if username != "" {
			cfg.Lichess = &LichessConfig{
				Username: username,
			}

			fmt.Print("Enter your Lichess API token (optional, press Enter to skip): ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)
			if token != "" {
				cfg.Lichess.APIToken = token
			}
		}
	}

	// Validate that at least one source is configured
	if !cfg.HasAnySource() {
		fmt.Println("\nWarning: No game sources configured. You can add them later with 'gochess config add-user'.")
	}

	// Save the config
	if err := cfg.SaveDefault(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
	fmt.Println("You can now run 'gochess import' to download and import your games!")

	return nil
}

// ShowCommand displays the current configuration
func ShowCommand(c *cli.Context) error {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	cfg, err := Load(configPath)
	if err != nil {
		return err
	}

	fmt.Println("=== GoChess Configuration ===")
	fmt.Printf("Config file: %s\n", configPath)
	fmt.Printf("Database: %s\n", cfg.DatabasePath)

	if cfg.ChessCom != nil && cfg.ChessCom.Username != "" {
		fmt.Println("\nChess.com:")
		fmt.Printf("  Username: %s\n", cfg.ChessCom.Username)
		if lastImport, ok := cfg.GetLastImport("chesscom", cfg.ChessCom.Username); ok {
			fmt.Printf("  Last import: %s\n", lastImport.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("  Last import: never\n")
		}
	}

	if cfg.Lichess != nil && cfg.Lichess.Username != "" {
		fmt.Println("\nLichess:")
		fmt.Printf("  Username: %s\n", cfg.Lichess.Username)
		if cfg.Lichess.APIToken != "" {
			fmt.Printf("  API Token: [configured]\n")
		}
		if lastImport, ok := cfg.GetLastImport("lichess", cfg.Lichess.Username); ok {
			fmt.Printf("  Last import: %s\n", lastImport.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("  Last import: never\n")
		}
	}

	if !cfg.HasAnySource() {
		fmt.Println("\nNo game sources configured.")
	}

	return nil
}

// AddUserCommand adds a user to the configuration
func AddUserCommand(c *cli.Context) error {
	platform := c.String("platform")
	username := c.String("username")
	token := c.String("token")

	if platform == "" {
		return fmt.Errorf("--platform is required (chesscom or lichess)")
	}
	if username == "" {
		return fmt.Errorf("--username is required")
	}

	platform = strings.ToLower(platform)
	if platform != "chesscom" && platform != "lichess" {
		return fmt.Errorf("invalid platform %q (must be 'chesscom' or 'lichess')", platform)
	}

	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	// Load existing config or create new one
	cfg, err := Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new config with defaults
			dbPath, err := DefaultDatabasePath()
			if err != nil {
				return err
			}
			cfg = &Config{
				DatabasePath: dbPath,
				LastImport:   make(map[string]time.Time),
			}
		} else {
			return err
		}
	}

	// Add the user
	switch platform {
	case "chesscom":
		cfg.ChessCom = &ChessComConfig{
			Username: username,
		}
		fmt.Printf("Added Chess.com user: %s\n", username)
	case "lichess":
		cfg.Lichess = &LichessConfig{
			Username: username,
			APIToken: token,
		}
		fmt.Printf("Added Lichess user: %s\n", username)
		if token != "" {
			fmt.Println("API token configured")
		}
	}

	// Save the config
	if err := cfg.SaveDefault(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration updated at %s\n", configPath)
	return nil
}

// RemoveUserCommand removes a user from the configuration
func RemoveUserCommand(c *cli.Context) error {
	platform := c.String("platform")

	if platform == "" {
		return fmt.Errorf("--platform is required (chesscom or lichess)")
	}

	platform = strings.ToLower(platform)
	if platform != "chesscom" && platform != "lichess" {
		return fmt.Errorf("invalid platform %q (must be 'chesscom' or 'lichess')", platform)
	}

	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	cfg, err := Load(configPath)
	if err != nil {
		return err
	}

	// Remove the user
	switch platform {
	case "chesscom":
		if cfg.ChessCom != nil {
			username := cfg.ChessCom.Username
			cfg.ChessCom = nil
			fmt.Printf("Removed Chess.com user: %s\n", username)
		} else {
			fmt.Println("No Chess.com user configured")
		}
	case "lichess":
		if cfg.Lichess != nil {
			username := cfg.Lichess.Username
			cfg.Lichess = nil
			fmt.Printf("Removed Lichess user: %s\n", username)
		} else {
			fmt.Println("No Lichess user configured")
		}
	}

	// Save the config
	if err := cfg.SaveDefault(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration updated at %s\n", configPath)
	return nil
}
