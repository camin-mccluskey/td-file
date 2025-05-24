package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	FilePath    string `yaml:"file_path"`
	FilePattern string `yaml:"file_pattern"`
	BaseDir     string `yaml:"base_directory"`
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	// Try XDG config directory first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "td-file", "config.yaml"), nil
	}

	// Fall back to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Join(home, ".config", "td-file")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// LoadConfig loads the configuration from the appropriate location
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &Config{
			BaseDir:     filepath.Join(os.Getenv("HOME"), "Documents", "todos"),
			FilePattern: "todos-{YYYY-MM-DD}.md",
		}

		if err := SaveConfig(defaultConfig); err != nil {
			return nil, err
		}
		return defaultConfig, nil
	}

	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to the appropriate location
func SaveConfig(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func ResolveTodoPath(cfg *Config) (string, error) {
	if cfg.FilePath != "" {
		return cfg.FilePath, nil
	}
	if cfg.FilePattern != "" {
		filename := strings.ReplaceAll(cfg.FilePattern, "{YYYY-MM-DD}", time.Now().Format("2006-01-02"))
		if cfg.BaseDir != "" {
			return filepath.Join(cfg.BaseDir, filename), nil
		}
		return filename, nil
	}
	return "", fmt.Errorf("no file_path or file_pattern specified in config")
}
