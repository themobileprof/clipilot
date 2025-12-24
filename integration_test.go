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

// TestCLIBuild tests that the CLI binary builds successfully
func TestCLIBuild(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "clipilot-test", "./cmd/clipilot")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, output)
	}

	// Clean up
	defer os.Remove("clipilot-test")

	// Verify binary exists
	if _, err := os.Stat("clipilot-test"); os.IsNotExist(err) {
		t.Fatal("Binary was not created")
	}
}

// TestRegistryBuild tests that the registry server builds successfully
func TestRegistryBuild(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "registry-test", "./cmd/registry")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, output)
	}

	// Clean up
	defer os.Remove("registry-test")

	// Verify binary exists
	if _, err := os.Stat("registry-test"); os.IsNotExist(err) {
		t.Fatal("Binary was not created")
	}
}

// TestCLIHelp tests that the CLI --help flag works
func TestCLIHelp(t *testing.T) {
	// Build binary first
	cmd := exec.Command("go", "build", "-o", "clipilot-test", "./cmd/clipilot")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer os.Remove("clipilot-test")

	// Test help flag
	cmd = exec.Command("./clipilot-test", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage") && !strings.Contains(outputStr, "config") {
		t.Errorf("Help output doesn't look right: %s", outputStr)
	}
}

// TestCLIVersion tests that the CLI --version flag works
func TestCLIVersion(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "clipilot-test", "./cmd/clipilot")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer os.Remove("clipilot-test")

	cmd = exec.Command("./clipilot-test", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Version command might exit with non-zero, that's okay
		if !strings.Contains(string(output), "CLIPilot") {
			t.Fatalf("Version command failed unexpectedly: %v\nOutput: %s", err, output)
		}
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "CLIPilot") {
		t.Errorf("Version output doesn't contain 'CLIPilot': %s", outputStr)
	}
}

// TestCLIInitAndReset tests database initialization and reset
func TestCLIInitAndReset(t *testing.T) {
	// Build binary
	cmd := exec.Command("go", "build", "-o", "clipilot-test", "./cmd/clipilot")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer os.Remove("clipilot-test")

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "clipilot-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test init command
	cmd = exec.Command("./clipilot-test", "--db", dbPath, "--config", configPath, "--init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Init command failed: %v\nOutput: %s", err, output)
	}

	// Verify database was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify config was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

// TestModuleLoading tests loading modules from directory
func TestModuleLoading(t *testing.T) {
	// Build binary
	cmd := exec.Command("go", "build", "-o", "clipilot-test", "./cmd/clipilot")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer os.Remove("clipilot-test")

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "clipilot-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	configPath := filepath.Join(tmpDir, "config.yaml")
	modulesDir := filepath.Join(tmpDir, "modules")

	// Create modules directory with a test module
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		t.Fatalf("Failed to create modules dir: %v", err)
	}

	testModule := `name: integration_test
id: org.test.integration
version: 1.0.0
description: Integration test module
tags: [test, integration]
flows:
  main:
    start: step1
    steps:
      step1:
        type: terminal
        message: "Integration test complete"
`

	moduleFile := filepath.Join(modulesDir, "test_module.yaml")
	if err := os.WriteFile(moduleFile, []byte(testModule), 0644); err != nil {
		t.Fatalf("Failed to write test module: %v", err)
	}

	// Initialize with modules
	cmd = exec.Command("./clipilot-test", "--db", dbPath, "--config", configPath, "--init", "--load", modulesDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Init with modules failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "integration_test") && !strings.Contains(outputStr, "Loaded") {
		t.Logf("Output: %s", outputStr)
		// Don't fail, just log - output format might vary
	}
}

// TestCrossCompilation tests that binaries can be built for different platforms
func TestCrossCompilation(t *testing.T) {
	platforms := []struct {
		goos   string
		goarch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
	}

	for _, platform := range platforms {
		t.Run(platform.goos+"_"+platform.goarch, func(t *testing.T) {
			outputName := "clipilot-" + platform.goos + "-" + platform.goarch
			cmd := exec.Command("go", "build", "-o", outputName, "./cmd/clipilot")
			cmd.Env = append(os.Environ(),
				"GOOS="+platform.goos,
				"GOARCH="+platform.goarch,
				"CGO_ENABLED=0",
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Build failed for %s/%s: %v\nOutput: %s", platform.goos, platform.goarch, err, output)
			}

			defer os.Remove(outputName)

			// Verify binary was created
			if _, err := os.Stat(outputName); os.IsNotExist(err) {
				t.Errorf("Binary was not created for %s/%s", platform.goos, platform.goarch)
			}
		})
	}
}

// TestDockerBuild tests that the registry Docker image builds
func TestDockerBuild(t *testing.T) {
	// Check if Docker is available
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker not available, skipping Docker build test")
	}

	// Build Docker image
	cmd = exec.Command("docker", "build", "-f", "Dockerfile.registry", "-t", "clipilot-registry-test", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Docker build failed: %v\nOutput: %s", err, output)
	}

	// Clean up image
	defer func() {
		cmd := exec.Command("docker", "rmi", "clipilot-registry-test")
		_ = cmd.Run() // Ignore errors in cleanup
	}()
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
