package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test loading non-existent config (should create default)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify default values
	if cfg.OnlineMode {
		t.Error("Expected OnlineMode to be false by default")
	}
	if cfg.AutoConfirm {
		t.Error("Expected AutoConfirm to be false by default")
	}
	if cfg.TelemetryEnabled {
		t.Error("Expected TelemetryEnabled to be false by default")
	}
	if cfg.ColorOutput {
		t.Error("Expected ColorOutput to be false by default")
	}
	if cfg.Thresholds.KeywordSearch != 0.6 {
		t.Errorf("Expected KeywordSearch threshold 0.6, got %f", cfg.Thresholds.KeywordSearch)
	}
	if cfg.Thresholds.LocalLLM != 0.6 {
		t.Errorf("Expected LocalLLM threshold 0.6, got %f", cfg.Thresholds.LocalLLM)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Test loading existing config
	cfg2, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load existing config failed: %v", err)
	}

	// Verify values match
	if cfg2.OnlineMode != cfg.OnlineMode {
		t.Error("OnlineMode mismatch after reload")
	}
	if cfg2.Thresholds.KeywordSearch != cfg.Thresholds.KeywordSearch {
		t.Error("KeywordSearch threshold mismatch after reload")
	}
}

func TestLoadExistingConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create custom config
	customConfig := `online_mode: true
auto_confirm: true
api_key: "test_key"
api_endpoint: "http://test.example.com"
thresholds:
  keyword_search: 0.8
  local_llm: 0.7
db_path: /tmp/test.db
telemetry_enabled: true
color_output: true
`

	if err := os.WriteFile(configPath, []byte(customConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify custom values
	if !cfg.OnlineMode {
		t.Error("Expected OnlineMode to be true")
	}
	if !cfg.AutoConfirm {
		t.Error("Expected AutoConfirm to be true")
	}
	if cfg.APIKey != "test_key" {
		t.Errorf("Expected APIKey 'test_key', got %s", cfg.APIKey)
	}
	if cfg.APIEndpoint != "http://test.example.com" {
		t.Errorf("Expected APIEndpoint 'http://test.example.com', got %s", cfg.APIEndpoint)
	}
	if cfg.Thresholds.KeywordSearch != 0.8 {
		t.Errorf("Expected KeywordSearch 0.8, got %f", cfg.Thresholds.KeywordSearch)
	}
	if cfg.Thresholds.LocalLLM != 0.7 {
		t.Errorf("Expected LocalLLM 0.7, got %f", cfg.Thresholds.LocalLLM)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("Expected DBPath '/tmp/test.db', got %s", cfg.DBPath)
	}
	if !cfg.TelemetryEnabled {
		t.Error("Expected TelemetryEnabled to be true")
	}
	if !cfg.ColorOutput {
		t.Error("Expected ColorOutput to be true")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create invalid YAML
	invalidYAML := `invalid: [yaml
online_mode: true
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load should fail
	_, err = Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config
	cfg := &Config{
		OnlineMode:  true,
		AutoConfirm: true,
		APIKey:      "save_test_key",
		Thresholds: Thresholds{
			KeywordSearch: 0.9,
			LocalLLM:      0.8,
		},
		DBPath:           "/tmp/save_test.db",
		TelemetryEnabled: false,
		ColorOutput:      true,
	}

	// Save config
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load and verify
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loaded.OnlineMode != cfg.OnlineMode {
		t.Error("OnlineMode mismatch")
	}
	if loaded.AutoConfirm != cfg.AutoConfirm {
		t.Error("AutoConfirm mismatch")
	}
	if loaded.APIKey != cfg.APIKey {
		t.Error("APIKey mismatch")
	}
	if loaded.Thresholds.KeywordSearch != cfg.Thresholds.KeywordSearch {
		t.Error("KeywordSearch threshold mismatch")
	}
	if loaded.DBPath != cfg.DBPath {
		t.Error("DBPath mismatch")
	}
	if loaded.TelemetryEnabled != cfg.TelemetryEnabled {
		t.Error("TelemetryEnabled mismatch")
	}
	if loaded.ColorOutput != cfg.ColorOutput {
		t.Error("ColorOutput mismatch")
	}
}

func TestDefaultConfigValues(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Load will create default config if it doesn't exist
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify defaults
	if cfg.OnlineMode {
		t.Error("Expected OnlineMode false")
	}
	if cfg.AutoConfirm {
		t.Error("Expected AutoConfirm false")
	}
	if cfg.Thresholds.KeywordSearch != 0.6 {
		t.Errorf("Expected KeywordSearch 0.6, got %f", cfg.Thresholds.KeywordSearch)
	}
	if cfg.Thresholds.LocalLLM != 0.6 {
		t.Errorf("Expected LocalLLM 0.6, got %f", cfg.Thresholds.LocalLLM)
	}
	if cfg.TelemetryEnabled {
		t.Error("Expected TelemetryEnabled false")
	}
	if cfg.ColorOutput {
		t.Error("Expected ColorOutput false")
	}

	// DBPath should contain .clipilot
	if cfg.DBPath == "" {
		t.Error("Expected DBPath to be set")
	}
}
