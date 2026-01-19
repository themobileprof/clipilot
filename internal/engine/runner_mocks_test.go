package engine

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/themobileprof/clipilot/internal/mocks"
	"github.com/themobileprof/clipilot/internal/models"
	_ "modernc.org/sqlite"
)

// TestRunnerWithMockModuleStore demonstrates using mocks for testing
func TestRunnerWithMockModuleStore(t *testing.T) {
	// Setup mock module store
	mockStore := mocks.NewMockModuleStore()

	// Create a test module
	testModule := &models.Module{
		ID:          "org.test.example",
		Name:        "test_module",
		Version:     "1.0.0",
		Description: "Test module",
		Flows: map[string]*models.Flow{
			"main": {
				Start: "step1",
				Steps: map[string]*models.Step{
					"step1": {
						Type:    "terminal",
						Message: "Test complete",
					},
				},
			},
		},
	}

	// Configure mock to return our test module
	mockStore.GetModuleFunc = func(moduleID string) (*models.Module, error) {
		if moduleID == "org.test.example" {
			return testModule, nil
		}
		return nil, nil
	}

	// Create temp database for logging
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize tables
	_, _ = db.Exec(`
		CREATE TABLE logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT,
			input TEXT,
			resolved_module TEXT,
			confidence REAL,
			method TEXT,
			status TEXT,
			error_message TEXT,
			duration_ms INTEGER,
			ts INTEGER DEFAULT (strftime('%s', 'now'))
		)
	`)

	// Create runner with mock store
	runner := NewRunner(db, mockStore)
	runner.SetDryRun(true) // Dry run to avoid actual command execution

	// Run the module
	err = runner.Run("org.test.example")
	if err != nil {
		t.Errorf("Runner.Run() failed: %v", err)
	}
}

// TestRunnerErrorHandlingWithMocks demonstrates testing error scenarios
func TestRunnerErrorHandlingWithMocks(t *testing.T) {
	mockStore := mocks.NewMockModuleStore()

	// Configure mock to return an error
	mockStore.GetModuleFunc = func(moduleID string) (*models.Module, error) {
		return nil, sql.ErrNoRows
	}

	// Create temp database
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	runner := NewRunner(db, mockStore)

	// Try to run non-existent module
	err = runner.Run("nonexistent.module")
	if err == nil {
		t.Error("Expected error for non-existent module, got nil")
	}
}

// TestMockModuleStoreOperations tests the mock store itself
func TestMockModuleStoreOperations(t *testing.T) {
	store := mocks.NewMockModuleStore()

	// Test ImportModule and GetModule
	testModule := &models.Module{
		ID:      "test.module",
		Name:    "Test",
		Version: "1.0.0",
	}

	err := store.ImportModule(testModule)
	if err != nil {
		t.Errorf("ImportModule failed: %v", err)
	}

	retrieved, err := store.GetModule("test.module")
	if err != nil {
		t.Errorf("GetModule failed: %v", err)
	}

	if retrieved.ID != testModule.ID {
		t.Errorf("Retrieved module ID mismatch: got %s, want %s", retrieved.ID, testModule.ID)
	}

	// Test ListModules
	modules, err := store.ListModules()
	if err != nil {
		t.Errorf("ListModules failed: %v", err)
	}

	if len(modules) != 1 {
		t.Errorf("ListModules returned %d modules, want 1", len(modules))
	}
}

// TestMockIntentClassifier demonstrates testing with mock intent detection
func TestMockIntentClassifier(t *testing.T) {
	mock := &mocks.MockIntentClassifier{}

	// Configure custom behavior
	mock.DetectFunc = func(input string) (*models.IntentResult, error) {
		return &models.IntentResult{
			ModuleID:   "detected.module",
			Confidence: 0.95,
			Method:     "mock",
			Candidates: []models.Candidate{
				{
					ModuleID: "detected.module",
					Score:    0.95,
				},
			},
		}, nil
	}

	// Test detection
	result, err := mock.Detect("test input")
	if err != nil {
		t.Errorf("Detect failed: %v", err)
	}

	if result.ModuleID != "detected.module" {
		t.Errorf("Expected module ID 'detected.module', got %s", result.ModuleID)
	}

	if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", result.Confidence)
	}
}

// TestMockStateStore demonstrates state management testing
func TestMockStateStore(t *testing.T) {
	store := mocks.NewMockStateStore()

	sessionID := "test-session-123"

	// Test SetState
	err := store.SetState(sessionID, "key1", "value1")
	if err != nil {
		t.Errorf("SetState failed: %v", err)
	}

	// Test GetState
	value, err := store.GetState(sessionID, "key1")
	if err != nil {
		t.Errorf("GetState failed: %v", err)
	}

	if value != "value1" {
		t.Errorf("Expected value 'value1', got %s", value)
	}

	// Test GetAllState
	allState, err := store.GetAllState(sessionID)
	if err != nil {
		t.Errorf("GetAllState failed: %v", err)
	}

	if len(allState) != 1 {
		t.Errorf("Expected 1 state entry, got %d", len(allState))
	}

	// Test DeleteState
	err = store.DeleteState(sessionID)
	if err != nil {
		t.Errorf("DeleteState failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetState(sessionID, "key1")
	if err == nil {
		t.Error("Expected error after deletion, got nil")
	}
}
