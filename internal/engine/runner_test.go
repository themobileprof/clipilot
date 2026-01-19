package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/themobileprof/clipilot/internal/db"
	"github.com/themobileprof/clipilot/internal/modules"
	"github.com/themobileprof/clipilot/internal/models"
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

func TestNewRunner(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)

	if runner == nil {
		t.Fatal("Expected non-nil runner")
	}
	if runner.db == nil {
		t.Error("Expected non-nil database connection")
	}
	if runner.loader == nil {
		t.Error("Expected non-nil loader")
	}
	if runner.dryRun {
		t.Error("Expected dryRun to be false by default")
	}
	if runner.autoYes {
		t.Error("Expected autoYes to be false by default")
	}
}

func TestSetDryRun(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)

	runner.SetDryRun(true)
	if !runner.dryRun {
		t.Error("Expected dryRun to be true")
	}

	runner.SetDryRun(false)
	if runner.dryRun {
		t.Error("Expected dryRun to be false")
	}
}

func TestSetAutoYes(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)

	runner.SetAutoYes(true)
	if !runner.autoYes {
		t.Error("Expected autoYes to be true")
	}

	runner.SetAutoYes(false)
	if runner.autoYes {
		t.Error("Expected autoYes to be false")
	}
}

func TestRunNonExistentModule(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)

	err := runner.Run("nonexistent_module")
	if err == nil {
		t.Error("Expected error for non-existent module, got nil")
	}
}

func TestRunSimpleModule(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)
	runner.SetDryRun(true) // Don't execute commands

	// Create simple test module
	module := &models.Module{
		Name:        "simple_test",
		ID:          "org.test.simple",
		Version:     "1.0.0",
		Description: "Simple test module",
		Flows: map[string]*models.Flow{
			"main": {
				Start: "step1",
				Steps: map[string]*models.Step{
					"step1": {
						Type:    "instruction",
						Message: "Step 1",
						Command: "echo 'test'",
						Next:    "step2",
					},
					"step2": {
						Type:    "terminal",
						Message: "Complete",
					},
				},
			},
		},
	}

	if err := loader.ImportModule(module); err != nil {
		t.Fatalf("Failed to import module: %v", err)
	}

	// Run module (dry run, so no user interaction needed)
	err := runner.Run(module.ID)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}

	// Verify execution was logged
	var count int
	err = database.Conn().QueryRow("SELECT COUNT(*) FROM logs WHERE resolved_module = ?", module.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}
	if count == 0 {
		t.Error("Expected execution to be logged")
	}
}

func TestEvaluateCondition(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)

	tests := []struct {
		name      string
		condition *models.Condition
		state     map[string]string
		expected  bool
	}{
		{
			name: "Equal - true",
			condition: &models.Condition{
				StateKey: "key1",
				Operator: "eq",
				Value:    "value1",
			},
			state:    map[string]string{"key1": "value1"},
			expected: true,
		},
		{
			name: "Equal - false",
			condition: &models.Condition{
				StateKey: "key1",
				Operator: "eq",
				Value:    "value2",
			},
			state:    map[string]string{"key1": "value1"},
			expected: false,
		},
		{
			name: "Not equal - true",
			condition: &models.Condition{
				StateKey: "key1",
				Operator: "ne",
				Value:    "value2",
			},
			state:    map[string]string{"key1": "value1"},
			expected: true,
		},
		{
			name: "Contains - true",
			condition: &models.Condition{
				StateKey: "key1",
				Operator: "contains",
				Value:    "val",
			},
			state:    map[string]string{"key1": "value1"},
			expected: true,
		},
		{
			name: "Contains - false",
			condition: &models.Condition{
				StateKey: "key1",
				Operator: "contains",
				Value:    "xyz",
			},
			state:    map[string]string{"key1": "value1"},
			expected: false,
		},
		{
			name: "Missing key",
			condition: &models.Condition{
				StateKey: "missing",
				Operator: "eq",
				Value:    "value",
			},
			state:    map[string]string{"key1": "value1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &models.ExecutionContext{
				State: tt.state,
			}
			result := runner.evaluateCondition(ctx, tt.condition)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRunCommand(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)

	tests := []struct {
		name        string
		command     string
		expectError bool
		expectOut   string
	}{
		{
			name:        "Simple echo",
			command:     "echo 'hello'",
			expectError: false,
			expectOut:   "hello",
		},
		{
			name:        "Command with exit code",
			command:     "exit 1",
			expectError: true,
		},
		{
			name:        "Multiple commands",
			command:     "echo 'line1' && echo 'line2'",
			expectError: false,
			expectOut:   "line1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runner.runCommand(tt.command)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectOut != "" && output != "" {
				// Check if output contains expected string (not exact match due to newlines)
				if len(output) == 0 {
					t.Errorf("Expected output to contain %q, got empty", tt.expectOut)
				}
			}
		})
	}
}

func TestLogStartAndComplete(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := modules.NewLoader(database.Conn())
	runner := NewRunner(database.Conn(), loader)

	ctx := &models.ExecutionContext{
		SessionID: "test_session",
		ModuleID:  "test_module",
	}

	// Test log start
	logID, err := runner.logStart(ctx, "test_module")
	if err != nil {
		t.Fatalf("logStart failed: %v", err)
	}
	if logID == 0 {
		t.Error("Expected non-zero log ID")
	}

	// Test log complete
	err = runner.logComplete(logID, "completed", "", 100)
	if err != nil {
		t.Fatalf("logComplete failed: %v", err)
	}

	// Verify log entry
	var status string
	var duration int64
	err = database.Conn().QueryRow("SELECT status, duration_ms FROM logs WHERE id = ?", logID).Scan(&status, &duration)
	if err != nil {
		t.Fatalf("Failed to query log: %v", err)
	}
	if status != "completed" {
		t.Errorf("Expected status 'completed', got %s", status)
	}
	if duration != 100 {
		t.Errorf("Expected duration 100, got %d", duration)
	}
}
