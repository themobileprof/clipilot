package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Test database creation
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created: %s", dbPath)
	}

	// Test connection is valid
	if err := db.conn.Ping(); err != nil {
		t.Errorf("Database connection is not valid: %v", err)
	}
}

func TestMigrate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify tables exist after migration
	tables := []string{"modules", "intent_patterns", "steps", "logs", "state", "settings", "dependencies"}
	for _, table := range tables {
		var count int
		err := db.conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query table %s: %v", table, err)
		}
		if count == 0 {
			t.Errorf("Table %s does not exist after migration", table)
		}
	}
}

func TestSettings(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test setting a value
	key := "test_key"
	value := "test_value"
	if err := db.SetSetting(key, value); err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	// Test retrieving the value
	retrieved, err := db.GetSetting(key)
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}
	if retrieved != value {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}

	// Test updating the value
	newValue := "new_value"
	if err := db.SetSetting(key, newValue); err != nil {
		t.Fatalf("Failed to update setting: %v", err)
	}

	retrieved, err = db.GetSetting(key)
	if err != nil {
		t.Fatalf("Failed to get updated setting: %v", err)
	}
	if retrieved != newValue {
		t.Errorf("Expected %s, got %s", newValue, retrieved)
	}

	// Test non-existent key
	retrieved, err = db.GetSetting("nonexistent")
	if err != nil {
		t.Fatalf("Unexpected error for non-existent key: %v", err)
	}
	if retrieved != "" {
		t.Errorf("Expected empty string for non-existent key, got %s", retrieved)
	}
}

func TestState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	sessionID := "test_session"
	key := "test_state_key"
	value := "test_state_value"

	// Test setting state
	if err := db.SetState(sessionID, key, value); err != nil {
		t.Fatalf("Failed to set state: %v", err)
	}

	// Test getting state
	retrieved, err := db.GetState(key)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if retrieved != value {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}
}

func TestLogQuery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test logging a query
	sessionID := "test_session"
	input := "test input"
	resolvedModule := "test_module"
	method := "keyword"
	confidence := 0.85

	logID, err := db.LogQuery(sessionID, input, resolvedModule, method, confidence)
	if err != nil {
		t.Fatalf("Failed to log query: %v", err)
	}
	if logID == 0 {
		t.Error("Expected non-zero log ID")
	}

	// Test updating log status
	status := "completed"
	errorMsg := ""
	durationMs := int64(150)

	if err := db.UpdateLogStatus(logID, status, errorMsg, durationMs); err != nil {
		t.Fatalf("Failed to update log status: %v", err)
	}

	// Verify the log was updated
	var retrievedStatus string
	var retrievedDuration int64
	err = db.conn.QueryRow("SELECT status, duration_ms FROM logs WHERE id = ?", logID).Scan(&retrievedStatus, &retrievedDuration)
	if err != nil {
		t.Fatalf("Failed to query log: %v", err)
	}
	if retrievedStatus != status {
		t.Errorf("Expected status %s, got %s", status, retrievedStatus)
	}
	if retrievedDuration != durationMs {
		t.Errorf("Expected duration %d, got %d", durationMs, retrievedDuration)
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "concurrent_key"
			value := "value"
			if err := db.SetSetting(key, value); err != nil {
				t.Errorf("Concurrent write %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
