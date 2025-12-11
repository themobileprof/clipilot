package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	OnlineMode       bool       `yaml:"online_mode"`
	AutoConfirm      bool       `yaml:"auto_confirm"`
	APIKey           string     `yaml:"api_key,omitempty"`
	APIEndpoint      string     `yaml:"api_endpoint,omitempty"`
	DBPath           string     `yaml:"db_path"`
	TelemetryEnabled bool       `yaml:"telemetry_enabled"`
	ColorOutput      bool       `yaml:"color_output"`
	Thresholds       Thresholds `yaml:"thresholds"`
}

// Thresholds holds confidence thresholds for intent detection
type Thresholds struct {
	KeywordSearch float64 `yaml:"keyword_search"`
	LocalLLM      float64 `yaml:"local_llm"`
}

// Default returns the default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		OnlineMode:       false,
		AutoConfirm:      false,
		APIKey:           "",
		APIEndpoint:      "",
		DBPath:           filepath.Join(homeDir, ".clipilot", "clipilot.db"),
		TelemetryEnabled: false,
		ColorOutput:      false,
		Thresholds: Thresholds{
			KeywordSearch: 0.6,
			LocalLLM:      0.6,
		},
	}
}

// Load reads configuration from file, creating with defaults if it doesn't exist
func Load(path string) (*Config, error) {
	// If file doesn't exist, create it with defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := Default()
		if err := cfg.Save(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// Read existing file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Default() // Start with defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".clipilot", "config.yaml")
}
