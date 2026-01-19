package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/themobileprof/clipilot/internal/db"
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

func TestNewLoader(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := NewLoader(database.Conn())
	if loader == nil {
		t.Fatal("Expected non-nil loader")
	}
	if loader.db == nil {
		t.Error("Expected non-nil database connection")
	}
}

func TestLoadFromFile(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test YAML file
	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	yamlContent := `name: test_module
id: org.test.test_module
version: 1.0.0
description: A test module
tags:
  - test
  - example
provides:
  - test_capability
requires:
  - dependency_module
size_kb: 5

flows:
  main:
    start: step1
    steps:
      step1:
        type: instruction
        message: "Test step 1"
        next: step2
      step2:
        type: terminal
        message: "Complete"
`

	testFile := filepath.Join(tmpDir, "test_module.yaml")
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(database.Conn())
	module, err := loader.LoadFromFile(testFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify module fields
	if module.Name != "test_module" {
		t.Errorf("Expected name 'test_module', got %s", module.Name)
	}
	if module.ID != "org.test.test_module" {
		t.Errorf("Expected ID 'org.test.test_module', got %s", module.ID)
	}
	if module.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", module.Version)
	}
	if module.Description != "A test module" {
		t.Errorf("Expected description 'A test module', got %s", module.Description)
	}
	if len(module.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(module.Tags))
	}
	if len(module.Provides) != 1 || module.Provides[0] != "test_capability" {
		t.Errorf("Expected provides 'test_capability', got %v", module.Provides)
	}
	if len(module.Requires) != 1 || module.Requires[0] != "dependency_module" {
		t.Errorf("Expected requires 'dependency_module', got %v", module.Requires)
	}
	if module.SizeKB != 5 {
		t.Errorf("Expected size_kb 5, got %d", module.SizeKB)
	}

	// Verify flows
	if len(module.Flows) != 1 {
		t.Fatalf("Expected 1 flow, got %d", len(module.Flows))
	}
	flow, exists := module.Flows["main"]
	if !exists {
		t.Fatal("Expected 'main' flow to exist")
	}
	if flow.Start != "step1" {
		t.Errorf("Expected start step 'step1', got %s", flow.Start)
	}
	if len(flow.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(flow.Steps))
	}
}

func TestLoadFromFileInvalidYAML(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create invalid YAML
	invalidYAML := `name: test
invalid: [yaml
`
	testFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(testFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(database.Conn())
	_, err = loader.LoadFromFile(testFile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadFromFileNonExistent(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := NewLoader(database.Conn())
	_, err := loader.LoadFromFile("/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestImportModule(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := NewLoader(database.Conn())

	// Create test module
	module := &models.Module{
		Name:        "import_test",
		ID:          "org.test.import_test",
		Version:     "1.0.0",
		Description: "Test import",
		Tags:        []string{"test", "import"},
		Provides:    []string{"test_feature"},
		Requires:    []string{"dependency"},
		SizeKB:      10,
		Flows: map[string]*models.Flow{
			"main": {
				Start: "step1",
				Steps: map[string]*models.Step{
					"step1": {
						Type:    "instruction",
						Message: "Test step",
						Next:    "",
					},
				},
			},
		},
	}

	// Import module
	if err := loader.ImportModule(module); err != nil {
		t.Fatalf("ImportModule failed: %v", err)
	}

	// Verify module was imported
	var count int
	err := database.Conn().QueryRow("SELECT COUNT(*) FROM modules WHERE id = ?", module.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query modules: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 module, got %d", count)
	}

	// Verify patterns were created
	err = database.Conn().QueryRow("SELECT COUNT(*) FROM intent_patterns WHERE module_id = ?", module.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query patterns: %v", err)
	}
	if count == 0 {
		t.Error("Expected patterns to be created")
	}

	// Verify steps were created
	err = database.Conn().QueryRow("SELECT COUNT(*) FROM steps WHERE module_id = ?", module.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query steps: %v", err)
	}
	if count == 0 {
		t.Error("Expected steps to be created")
	}
}

func TestGetModule(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := NewLoader(database.Conn())

	// Create and import test module
	originalModule := &models.Module{
		Name:        "get_test",
		ID:          "org.test.get_test",
		Version:     "1.0.0",
		Description: "Test get",
		Tags:        []string{"test"},
		Flows: map[string]*models.Flow{
			"main": {
				Start: "step1",
				Steps: map[string]*models.Step{
					"step1": {
						Type:    "terminal",
						Message: "Done",
					},
				},
			},
		},
	}

	if err := loader.ImportModule(originalModule); err != nil {
		t.Fatalf("ImportModule failed: %v", err)
	}

	// Get module
	retrievedModule, err := loader.GetModule(originalModule.ID)
	if err != nil {
		t.Fatalf("GetModule failed: %v", err)
	}

	// Verify fields
	if retrievedModule.Name != originalModule.Name {
		t.Errorf("Expected name %s, got %s", originalModule.Name, retrievedModule.Name)
	}
	if retrievedModule.ID != originalModule.ID {
		t.Errorf("Expected ID %s, got %s", originalModule.ID, retrievedModule.ID)
	}
	if retrievedModule.Version != originalModule.Version {
		t.Errorf("Expected version %s, got %s", originalModule.Version, retrievedModule.Version)
	}
}

func TestGetModuleNonExistent(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := NewLoader(database.Conn())

	_, err := loader.GetModule("nonexistent_module")
	if err == nil {
		t.Error("Expected error for non-existent module, got nil")
	}
}

func TestListModules(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	loader := NewLoader(database.Conn())

	// Import multiple modules
	modules := []*models.Module{
		{
			Name:    "module1",
			ID:      "org.test.module1",
			Version: "1.0.0",
			Flows:   map[string]*models.Flow{},
		},
		{
			Name:    "module2",
			ID:      "org.test.module2",
			Version: "2.0.0",
			Flows:   map[string]*models.Flow{},
		},
	}

	for _, mod := range modules {
		if err := loader.ImportModule(mod); err != nil {
			t.Fatalf("Failed to import module %s: %v", mod.Name, err)
		}
	}

	// List modules
	listed, err := loader.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(listed))
	}

	// Verify modules are in list
	found := make(map[string]bool)
	for _, mod := range listed {
		found[mod.ID] = true
	}
	for _, mod := range modules {
		if !found[mod.ID] {
			t.Errorf("Module %s not found in list", mod.ID)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		minLen   int
		expected []string
	}{
		{"test_module_name", 3, []string{"test", "module", "name"}},
		{"simple", 3, []string{"simple"}},
		{"ab", 3, []string{}},
		{"copy-file", 3, []string{"copy", "file"}},
		{"install MySQL Database", 3, []string{"install", "mysql", "database"}},
	}

	for _, tt := range tests {
		result := tokenize(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("tokenize(%q): expected %d tokens, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for i, token := range result {
			if token != tt.expected[i] {
				t.Errorf("tokenize(%q): expected token %q at position %d, got %q", tt.input, tt.expected[i], i, token)
			}
		}
	}
}

func BenchmarkImportModule(b *testing.B) {
	database, err := db.New(":memory:")
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	loader := NewLoader(database.Conn())
	module := &models.Module{
		Name:        "bench_module",
		ID:          "org.bench.module",
		Version:     "1.0.0",
		Description: "Benchmark module",
		Tags:        []string{"benchmark", "test"},
		Flows: map[string]*models.Flow{
			"main": {
				Start: "step1",
				Steps: map[string]*models.Step{
					"step1": {Type: "instruction", Message: "Test", Next: "step2"},
					"step2": {Type: "terminal", Message: "Done"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = loader.ImportModule(module)
	}
}
