package ui

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/themobileprof/clipilot/internal/engine"
	"github.com/themobileprof/clipilot/internal/intent"
	"github.com/themobileprof/clipilot/internal/modules"
)

// REPL represents the interactive command-line interface
type REPL struct {
	db       *sql.DB
	loader   *modules.Loader
	runner   *engine.Runner
	detector *intent.Detector
	history  []string
}

// NewREPL creates a new REPL interface
func NewREPL(db *sql.DB) *REPL {
	loader := modules.NewLoader(db)
	runner := engine.NewRunner(db, loader)
	detector := intent.NewDetector(db)

	return &REPL{
		db:       db,
		loader:   loader,
		runner:   runner,
		detector: detector,
		history:  []string{},
	}
}

// Start begins the interactive REPL loop
func (repl *REPL) Start() error {
	fmt.Println("CLIPilot v1.0.0 - Lightweight CLI Assistant")
	fmt.Println("Type 'help' for available commands, 'exit' to quit\n")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		repl.history = append(repl.history, input)

		if err := repl.handleCommand(input); err != nil {
			if err.Error() == "exit" {
				fmt.Println("Goodbye!")
				return nil
			}
			fmt.Printf("Error: %v\n\n", err)
		}
	}
}

// handleCommand processes a single command
func (repl *REPL) handleCommand(input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "help":
		return repl.showHelp()
	case "exit", "quit":
		return fmt.Errorf("exit")
	case "search":
		return repl.searchModules(strings.Join(args, " "))
	case "run":
		if len(args) == 0 {
			return fmt.Errorf("usage: run <module_id>")
		}
		return repl.runner.Run(args[0])
	case "modules":
		if len(args) == 0 {
			return fmt.Errorf("usage: modules <list|install|remove>")
		}
		return repl.handleModulesCommand(args)
	case "settings":
		return repl.showSettings()
	case "logs":
		return repl.showLogs()
	default:
		// Treat as natural language query
		return repl.handleQuery(input)
	}
}

// showHelp displays help information
func (repl *REPL) showHelp() error {
	fmt.Println(`
Available Commands:
  help                    - Show this help message
  search <query>          - Search for modules matching query
  run <module_id>         - Execute a specific module
  modules list            - List all installed modules
  modules install <id>    - Download and install a module
  modules remove <id>     - Remove an installed module
  settings                - Show current settings
  logs                    - View execution history
  exit, quit              - Exit CLIPilot

Natural Language:
  You can also type natural language queries, and CLIPilot will
  try to find and suggest relevant modules.

Examples:
  > install mysql
  > setup docker
  > how to configure git
  > run detect_os
`)
	return nil
}

// searchModules searches for modules matching a query
func (repl *REPL) searchModules(query string) error {
	if query == "" {
		return fmt.Errorf("please provide a search query")
	}

	result, err := repl.detector.Detect(query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(result.Candidates) == 0 {
		fmt.Println("No modules found matching your query.")
		fmt.Println("Try different keywords or check installed modules with: modules list")
		return nil
	}

	fmt.Printf("\nFound %d module(s):\n\n", len(result.Candidates))
	for i, candidate := range result.Candidates {
		if i >= 10 {
			break // Limit to top 10
		}
		fmt.Printf("%d. %s (ID: %s)\n", i+1, candidate.Name, candidate.ModuleID)
		fmt.Printf("   %s\n", candidate.Description)
		fmt.Printf("   Score: %.2f | Tags: %s\n\n", candidate.Score, strings.Join(candidate.Tags, ", "))
	}

	return nil
}

// handleQuery processes natural language queries
func (repl *REPL) handleQuery(input string) error {
	result, err := repl.detector.Detect(input)
	if err != nil {
		return fmt.Errorf("query processing failed: %w", err)
	}

	if result.ModuleID == "" || len(result.Candidates) == 0 {
		fmt.Println("I couldn't find a relevant module for your query.")
		fmt.Println("Try rephrasing or use 'search <keywords>' to find modules.")
		return nil
	}

	// Show top candidate
	top := result.Candidates[0]
	fmt.Printf("\nFound: %s (confidence: %.2f)\n", top.Name, result.Confidence)
	fmt.Printf("Description: %s\n", top.Description)

	if result.Confidence < 0.7 && len(result.Candidates) > 1 {
		fmt.Println("\nOther possible matches:")
		for i := 1; i < len(result.Candidates) && i < 4; i++ {
			fmt.Printf("  - %s\n", result.Candidates[i].Name)
		}
	}

	fmt.Printf("\nRun this module? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "y" || response == "yes" {
		return repl.runner.Run(top.ModuleID)
	}

	fmt.Println("Cancelled.")
	return nil
}

// handleModulesCommand handles module management commands
func (repl *REPL) handleModulesCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: modules <list|install|remove>")
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		return repl.listModules()
	case "install":
		if len(args) < 2 {
			return fmt.Errorf("usage: modules install <module_id>")
		}
		return repl.installModule(args[1])
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("usage: modules remove <module_id>")
		}
		return repl.removeModule(args[1])
	default:
		return fmt.Errorf("unknown subcommand: %s", subcommand)
	}
}

// listModules lists all installed modules
func (repl *REPL) listModules() error {
	modules, err := repl.loader.ListModules()
	if err != nil {
		return fmt.Errorf("failed to list modules: %w", err)
	}

	if len(modules) == 0 {
		fmt.Println("No modules installed.")
		fmt.Println("Use 'modules install <id>' to install modules.")
		return nil
	}

	fmt.Printf("\nInstalled Modules (%d):\n\n", len(modules))
	for _, mod := range modules {
		fmt.Printf("• %s (v%s)\n", mod.Name, mod.Version)
		fmt.Printf("  ID: %s\n", mod.ID)
		fmt.Printf("  %s\n", mod.Description)
		if len(mod.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(mod.Tags, ", "))
		}
		fmt.Println()
	}

	return nil
}

// installModule installs a module from the registry
func (repl *REPL) installModule(moduleID string) error {
	// Get registry URL from settings
	var registryURL string
	err := repl.db.QueryRow("SELECT value FROM settings WHERE key = 'registry_url'").Scan(&registryURL)
	if err != nil {
		registryURL = "http://localhost:8080" // Default registry URL
	}

	fmt.Printf("Downloading module %s from registry...\n", moduleID)

	// Download module from registry
	url := fmt.Sprintf("%s/modules/%s", registryURL, moduleID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download module: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("module not found in registry")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned error: %s", resp.Status)
	}

	// Read module YAML
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read module data: %w", err)
	}

	// Save to temporary file
	tmpFile, err := os.CreateTemp("", "clipilot-module-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Load and import module
	module, err := repl.loader.LoadFromFile(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to load module: %w", err)
	}

	if err := repl.loader.ImportModule(module); err != nil {
		return fmt.Errorf("failed to import module: %w", err)
	}

	fmt.Printf("✓ Module %s (v%s) installed successfully!\n", module.Name, module.Version)
	return nil
}

// removeModule removes a module
func (repl *REPL) removeModule(moduleID string) error {
	_, err := repl.db.Exec("UPDATE modules SET installed = 0 WHERE id = ?", moduleID)
	if err != nil {
		return fmt.Errorf("failed to remove module: %w", err)
	}
	fmt.Printf("Module %s removed.\n", moduleID)
	return nil
}

// showSettings displays current settings
func (repl *REPL) showSettings() error {
	rows, err := repl.db.Query(`
		SELECT key, value, description
		FROM settings
		ORDER BY key
	`)
	if err != nil {
		return fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	fmt.Println("\nCurrent Settings:\n")
	for rows.Next() {
		var key, value, description string
		if err := rows.Scan(&key, &value, &description); err != nil {
			continue
		}
		fmt.Printf("• %s = %s\n", key, value)
		if description != "" {
			fmt.Printf("  %s\n", description)
		}
	}
	fmt.Println()

	return nil
}

// showLogs displays recent execution logs
func (repl *REPL) showLogs() error {
	rows, err := repl.db.Query(`
		SELECT ts, resolved_module, status, confidence, method, duration_ms
		FROM logs
		ORDER BY ts DESC
		LIMIT 20
	`)
	if err != nil {
		return fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	fmt.Println("\nRecent Executions:\n")
	for rows.Next() {
		var ts int64
		var module, status, method string
		var confidence float64
		var duration sql.NullInt64

		if err := rows.Scan(&ts, &module, &status, &confidence, &method, &duration); err != nil {
			continue
		}

		durationStr := "N/A"
		if duration.Valid {
			durationStr = fmt.Sprintf("%dms", duration.Int64)
		}

		fmt.Printf("• %s | Status: %s | Duration: %s\n", module, status, durationStr)
		fmt.Printf("  Method: %s | Confidence: %.2f\n", method, confidence)
	}
	fmt.Println()

	return nil
}

// ExecuteNonInteractive runs a command non-interactively
func (repl *REPL) ExecuteNonInteractive(input string) error {
	return repl.handleCommand(input)
}
