package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/themobileprof/clipilot/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func TestNewDetector(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	detector := NewDetector(database.Conn())
	if detector == nil {
		t.Fatal("Expected non-nil detector")
	}

	if detector.keywordThresh != 0.6 {
		t.Errorf("Expected default keyword threshold 0.6, got %f", detector.keywordThresh)
	}
	if detector.llmThresh != 0.6 {
		t.Errorf("Expected default LLM threshold 0.6, got %f", detector.llmThresh)
	}
	if detector.onlineEnabled {
		t.Error("Expected online mode to be disabled by default")
	}
}

func TestSetThresholds(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	detector := NewDetector(database.Conn())

	detector.SetThresholds(0.8, 0.7)
	if detector.keywordThresh != 0.8 {
		t.Errorf("Expected keyword threshold 0.8, got %f", detector.keywordThresh)
	}
	if detector.llmThresh != 0.7 {
		t.Errorf("Expected LLM threshold 0.7, got %f", detector.llmThresh)
	}
}

func TestSetOnlineEnabled(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	detector := NewDetector(database.Conn())

	detector.SetOnlineEnabled(true)
	if !detector.onlineEnabled {
		t.Error("Expected online mode to be enabled")
	}

	detector.SetOnlineEnabled(false)
	if detector.onlineEnabled {
		t.Error("Expected online mode to be disabled")
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "install mysql database",
			expected: []string{"install", "package", "get", "mysql", "database"}, // Correct (5) - added 'get'
		},
		{
			input:    "setup git configuration",
			expected: []string{"setup", "git", "configuration"}, // Correct (3)
		},
		{
			input:    "copy_file from source to destination",
			expected: []string{"copy", "duplicate", "cp", "source", "destination"}, // Correct (5) - added 'cp', removed 'file'
		},
		{
			input:    "THE and FOR with", // Stop words
			expected: []string{},
		},
		{
			input:    "Setup Git Config", // Case insensitive
			expected: []string{"setup", "git", "config", "configuration"}, // Correct (4)
		},
		{
			input:    "ab cd ef", // Too short tokens
			expected: []string{"ab", "cd", "ef"},
		},
		{
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		result := tokenize(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("For input %q: expected %d tokens, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for i, token := range result {
			if token != tt.expected[i] {
				t.Errorf("For input %q: expected token %q at position %d, got %q", tt.input, tt.expected[i], i, token)
			}
		}
	}
}

func TestDetectWithNoModules(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	detector := NewDetector(database.Conn())

	result, err := detector.Detect("nonexistent_random_cmd_12345")
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.ModuleID != "" {
		t.Errorf("Expected empty module ID, got %s", result.ModuleID)
	}

	if result.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0, got %f", result.Confidence)
	}

	if len(result.Candidates) != 0 {
		t.Errorf("Expected 0 candidates, got %d", len(result.Candidates))
	}
}

func TestDetectWithCommand(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test command (Detect() now searches commands, not modules)
	_, err := database.Conn().Exec(`
		INSERT INTO commands (name, description, has_man)
		VALUES (?, ?, 1)
	`, "testcmd", "A test command for testing")
	if err != nil {
		t.Fatalf("Failed to insert test command: %v", err)
	}

	detector := NewDetector(database.Conn())

	// Test detection - should find the command
	result, err := detector.Detect("testcmd")
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Candidates) == 0 {
		t.Error("Expected at least one candidate")
	}

	if result.Confidence == 0.0 {
		t.Error("Expected non-zero confidence")
	}

	// Verify the result is a command, not a module
	if len(result.Candidates) > 0 && !strings.HasPrefix(result.Candidates[0].ModuleID, "cmd:") {
		t.Errorf("Expected command result (cmd: prefix), got: %s", result.Candidates[0].ModuleID)
	}
}

// TestKeywordSearchModule tests the original keywordSearch which still searches modules
func TestKeywordSearchModule(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test module
	_, err := database.Conn().Exec(`
		INSERT INTO modules (id, name, version, description, tags, installed, json_content)
		VALUES (?, ?, ?, ?, ?, 1, '{}')
	`, "test_module", "Test Module", "1.0.0", "A test module for testing", "test,module")
	if err != nil {
		t.Fatalf("Failed to insert test module: %v", err)
	}

	// Insert patterns
	patterns := []struct {
		pattern string
		weight  float64
		ptype   string
	}{
		{"test", 1.5, "keyword"},
		{"module", 1.0, "keyword"},
		{"testing", 1.0, "keyword"},
	}

	for _, p := range patterns {
		_, err := database.Conn().Exec(`
			INSERT INTO intent_patterns (module_id, pattern, weight, pattern_type)
			VALUES (?, ?, ?, ?)
		`, "test_module", p.pattern, p.weight, p.ptype)
		if err != nil {
			t.Fatalf("Failed to insert pattern %s: %v", p.pattern, err)
		}
	}

	detector := NewDetector(database.Conn())

	// Test keywordSearch (not Detect) - this still searches modules
	result, err := detector.keywordSearch("run test module")
	if err != nil {
		t.Fatalf("keywordSearch failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Candidates) == 0 {
		t.Error("Expected at least one candidate")
	}

	if result.Confidence == 0.0 {
		t.Error("Expected non-zero confidence")
	}
}

func TestKeywordSearchEmptyInput(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	detector := NewDetector(database.Conn())

	result, err := detector.keywordSearch("")
	if err != nil {
		t.Fatalf("keywordSearch failed: %v", err)
	}

	if len(result.Candidates) != 0 {
		t.Errorf("Expected 0 candidates for empty input, got %d", len(result.Candidates))
	}
}

func BenchmarkTokenize(b *testing.B) {
	input := "install mysql database server with secure configuration"
	for i := 0; i < b.N; i++ {
		_ = tokenize(input)
	}
}

func BenchmarkDetect(b *testing.B) {
	database, err := db.New(":memory:")
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	// Insert test data
	_, err = database.Conn().Exec(`
		INSERT INTO modules (id, name, version, description, tags, installed, json_content)
		VALUES (?, ?, ?, ?, ?, 1, '{}')
	`, "bench_module", "Benchmark Module", "1.0.0", "A module for benchmarking", "test,benchmark")
	if err != nil {
		b.Fatalf("Failed to insert module: %v", err)
	}

	_, err = database.Conn().Exec(`
		INSERT INTO intent_patterns (module_id, pattern, weight, pattern_type)
		VALUES (?, ?, ?, ?)
	`, "bench_module", "benchmark", 1.5, "keyword")
	if err != nil {
		b.Fatalf("Failed to insert pattern: %v", err)
	}

	detector := NewDetector(database.Conn())
	input := "run benchmark test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect(input)
	}
}
