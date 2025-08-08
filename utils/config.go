// config.go - Configuration management
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ScanPaths     []string `json:"scan_paths"`
	SkipDirs      []string `json:"skip_dirs"`
	Editor        string   `json:"editor"`
	MaxDepth      int      `json:"max_depth"`
	IncludeHidden bool     `json:"include_hidden"`
}

func GetDefaultConfig() Config {
	user, _ := os.UserHomeDir()
	return Config{
		ScanPaths: []string{
			filepath.Join(user, "Documents"),
			filepath.Join(user, "Desktop"),
			filepath.Join(user, "Projects"),
		},
		SkipDirs: []string{
			"node_modules", "dist", "build", "target", "vendor", ".git",
			".vscode", ".idea", "bin", "obj", "out", "tmp", "temp",
			"logs", "cache", ".next", ".nuxt", "coverage",
		},
		Editor:        "code",
		MaxDepth:      5,
		IncludeHidden: false,
	}
}

func LoadConfig() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	configPath := filepath.Join(homeDir, ".gitseeker", "config.json")

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return Config{}, fmt.Errorf("failed to create config directory: %w", err)
	}

	// If config doesn't exist, create it with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := GetDefaultConfig()
		if err := SaveConfig(defaultConfig); err != nil {
			return Config{}, fmt.Errorf("failed to save default config: %w", err)
		}
		return defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func SaveConfig(config Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".gitseeker", "config.json")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}		

	return nil
}
