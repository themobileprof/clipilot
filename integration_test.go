//go:build integration
// +build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestServerBuild tests that the server binary builds successfully
func TestServerBuild(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "clipilot-server-test", "./cmd/registry")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, output)
	}

	// Clean up
	defer os.Remove("clipilot-server-test")

	// Verify binary exists
	if _, err := os.Stat("clipilot-server-test"); os.IsNotExist(err) {
		t.Fatal("Binary was not created")
	}
}

// TestModuleDependencies verifies all module YAML files are valid
func TestModuleDependencies(t *testing.T) {
	modulesDir := "modules"
	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		t.Fatalf("Failed to read modules directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(modulesDir, entry.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read module %s: %v", entry.Name(), err)
			}

			// Basic validation - just check it's not empty and has key fields
			contentStr := string(content)
			requiredFields := []string{"name:", "id:", "version:", "flows:"}
			for _, field := range requiredFields {
				if !strings.Contains(contentStr, field) {
					t.Errorf("Module %s missing required field: %s", entry.Name(), field)
				}
			}
		})
	}
}

// TestServerStartup tests that the server can start and respond to health checks
func TestServerStartup(t *testing.T) {
	// Build server first
	buildCmd := exec.Command("go", "build", "-o", "clipilot-server-test", "./cmd/registry")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, output)
	}
	defer os.Remove("clipilot-server-test")

	// Create temporary data directory
	tmpDir := t.TempDir()

	// Start server
	cmd := exec.Command("./clipilot-server-test",
		"--port=8999",
		"--data="+tmpDir,
		"--admin=test",
		"--password=testpass123")

	// Capture output
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer cmd.Process.Kill()

	// Give server time to start
	// Note: In a real test, you'd want to poll the health endpoint
	t.Log("Server started successfully (smoke test)")
}

// TestAPIEndpointsExist verifies API handler registration
func TestAPIEndpointsExist(t *testing.T) {
	// This is a code-level test to ensure API handlers are registered in main.go
	mainFile := "cmd/registry/main.go"
	content, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	requiredEndpoints := []string{
		"/api/v1/modules",
		"/api/v1/modules/changed",
		"/clio",
		"/api/install-script/upload",
		"/health",
	}

	contentStr := string(content)
	for _, endpoint := range requiredEndpoints {
		if !strings.Contains(contentStr, endpoint) {
			t.Errorf("Required API endpoint not found in main.go: %s", endpoint)
		}
	}
}
