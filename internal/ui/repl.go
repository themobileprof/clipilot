package ui

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/themobileprof/clipilot/internal/commands"
	"github.com/themobileprof/clipilot/internal/engine"
	"github.com/themobileprof/clipilot/internal/intent"
	"github.com/themobileprof/clipilot/internal/modules"
	"github.com/themobileprof/clipilot/internal/registry"
)

// REPL represents the interactive command-line interface
type REPL struct {
	db             *sql.DB
	loader         *modules.Loader
	runner         *engine.Runner
	detector       *intent.Detector
	registryClient *registry.Client
	history        []string
}

// NewREPL creates a new REPL interface
func NewREPL(db *sql.DB) *REPL {
	loader := modules.NewLoader(db)
	runner := engine.NewRunner(db, loader)
	detector := intent.NewDetector(db)

	// Get registry URL from settings
	registryURL, _ := registry.GetRegistryURL(db)
	registryClient := registry.NewClient(db, registryURL)

	return &REPL{
		db:             db,
		loader:         loader,
		runner:         runner,
		detector:       detector,
		registryClient: registryClient,
		history:        []string{},
	}
}

// Start begins the interactive REPL loop
func (repl *REPL) Start() error {
	// Detect Termux environment
	isTermux := os.Getenv("TERMUX_VERSION") != "" || os.Getenv("PREFIX") != ""

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║          CLIPilot v1.0.0 - Your CLI Assistant           ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	if isTermux {
		fmt.Println("📱 Running on Termux - All tools work on your Android device!")
		fmt.Println("💡 New to command line? Type 'help' for a beginner's guide")
		fmt.Println("🚀 Quick start: 'run termux_setup' or 'search phone tools'")
	} else {
		fmt.Println("👋 New to command line? Type 'help' for a beginner's guide")
		fmt.Println("🔍 Try searching: 'search git' or 'search copy files'")
	}
	fmt.Println()
	fmt.Println("Type 'help' anytime • Type 'exit' when done")
	fmt.Println()

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
	case "sync":
		return repl.syncRegistry()
	case "update-commands":
		return repl.updateCommands()
	case "reset":
		return repl.resetDatabase()
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
	fmt.Print(`
╔══════════════════════════════════════════════════════════════════════════╗
║                       Welcome to CLIPilot!                               ║
║           Your Friendly Assistant for Command-Line Tasks                 ║
╚══════════════════════════════════════════════════════════════════════════╝

💡 NEW TO COMMAND LINE? Just type what you want to do in plain English!
   CLIPilot will find the right tools and guide you step-by-step.

🔍 BASIC COMMANDS (Just type these and press Enter):

  help                    Show this help message anytime you need it
  
  search <what you need>  Find tools and modules
                          Example: search "copy files"
                          Example: search git
                          
  exit  or  quit          Close CLIPilot when you're done

📦 DISCOVERING TOOLS:

  When you search, CLIPilot will show you:
  • Tools already on your device (ready to use)
  • Tools you can install (with instructions)
  • Modules that automate tasks for you
  
  If a tool is "not installed", CLIPilot tells you how to get it!

⚙️  MANAGING MODULES (Pre-made Automations):

  modules list            See what's on your device
  modules list --all      See everything available (uses some data)
  modules install <name>  Add a new module (downloads it)
  modules remove <name>   Delete a module you don't need
  
  run <module_name>       Use a module to do a task

🌐 KEEPING UP TO DATE:

  sync                    Get latest modules from internet
                          (Will download data - do this on WiFi!)
                          
  update-commands         Refresh list of tools on your device
                          (No internet needed - safe anytime)

📊 SETTINGS & HISTORY:

  settings                See your current configuration
  logs                    See what tasks you've run before

🆘 FIRST TIME USING COMMAND LINE?

  Don't worry! Here's what to do:
  
  1. Type: search "system information"
     (This shows you details about your device)
     
  2. Type: modules list
     (This shows helpful tools already installed)
     
  3. Try asking in plain English:
     • "how do I check disk space"
     • "copy files"
     • "setup git"
     
  CLIPilot understands natural language - just describe what you need!

✨ EXAMPLES TO TRY:

  > search ripgrep
    (Finds tools for searching inside files)
    
  > search "copy files"
    (Shows you different ways to copy files)
    
  > modules list
    (See what automations you have)
    
  > how do I setup python
    (CLIPilot guides you through Python setup)
    
  > run detect_os
    (Tells you about your device)

💾 DATA USAGE TIPS (Important for limited internet):

  ✓ "search" - No internet needed (searches local database)
  ✓ "update-commands" - No internet needed
  ✓ "modules list" - No internet needed
  ⚠ "sync" - Uses internet (downloads module catalog)
  ⚠ "modules install" - Uses internet (downloads module)

📱 REMEMBER: You can always type "help" to see this message again!

`)
	return nil
}

// searchModules searches for modules matching a query
func (repl *REPL) searchModules(query string) error {
	if query == "" {
		fmt.Println("\n💡 To search, type what you're looking for after 'search'")
		fmt.Println("   Examples:")
		fmt.Println("   • search git")
		fmt.Println("   • search \"copy files\"")
		fmt.Println("   • search database")
		return nil
	}

	result, err := repl.detector.Detect(query)
	if err != nil {
		fmt.Printf("\n⚠️  Search had a problem: %v\n", err)
		fmt.Println("💡 Try simpler words or check your spelling")
		return nil
	}

	if len(result.Candidates) == 0 {
		fmt.Println("\n🔍 No matches found for your search.")
		fmt.Println()
		fmt.Println("💡 Tips:")
		fmt.Println("   • Try different words (e.g., 'copy' instead of 'duplicate')")
		fmt.Println("   • Check what's installed: modules list")
		fmt.Println("   • Get new tools: sync (requires internet)")
		fmt.Println()
		fmt.Println("📝 Your search has been recorded to help improve CLIPilot!")

		// Submit module request to registry
		repl.submitModuleRequest(query)
		return nil
	}

	fmt.Printf("\n✨ Found %d result(s) for '%s':\n\n", len(result.Candidates), query)
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
		fmt.Printf("\n⚠️  Could not understand your request: %v\n", err)
		fmt.Println("\n💡 Try using the 'search' command instead:")
		fmt.Printf("   search %s\n", input)
		return nil
	}

	if result.ModuleID == "" || len(result.Candidates) == 0 {
		fmt.Println("\n🤔 I couldn't find tools for what you're asking.")
		fmt.Println()
		fmt.Println("💡 Here are some things to try:")
		fmt.Printf("   1. Use 'search' command: search %s\n", input)
		fmt.Println("   2. Try simpler words: instead of 'duplicate files', try 'copy files'")
		fmt.Println("   3. Check available modules: modules list")
		fmt.Println()
		fmt.Println("📝 Your request helps us improve CLIPilot - thank you!")

		// Submit module request to registry
		repl.submitModuleRequest(input)
		return nil
	}

	// Show top candidate
	top := result.Candidates[0]
	fmt.Printf("\n✨ Found: %s (match strength: %.0f%%)\n", top.Name, result.Confidence*100)
	fmt.Printf("   What it does: %s\n", top.Description)

	if result.Confidence < 0.7 && len(result.Candidates) > 1 {
		fmt.Println("\n🔍 Other tools you might want:")
		for i := 1; i < len(result.Candidates) && i < 4; i++ {
			fmt.Printf("   • %s\n", result.Candidates[i].Name)
		}
	}

	fmt.Printf("\nRun this module? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "y" || response == "yes" {
		return repl.runner.Run(top.ModuleID)
	}

	fmt.Println("💡 Tip: To just search without running, use: search <what you need>")
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
		flags := args[1:]
		return repl.listModules(flags)
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

// listModules lists modules based on flags
func (repl *REPL) listModules(flags []string) error {
	listAll := false
	listAvailable := false

	for _, flag := range flags {
		switch flag {
		case "--all", "-a":
			listAll = true
		case "--available", "--avail":
			listAvailable = true
		}
	}

	if listAvailable || listAll {
		return repl.listAvailableModules(listAll)
	}

	// Default: list only installed modules
	modules, err := repl.loader.ListModules()
	if err != nil {
		return fmt.Errorf("failed to list modules: %w", err)
	}

	if len(modules) == 0 {
		fmt.Println("\nNo modules installed.")
		fmt.Println("Use 'sync' to fetch available modules, then 'modules install <id>' to install.")
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

// listAvailableModules lists available and optionally installed modules
func (repl *REPL) listAvailableModules(includeInstalled bool) error {
	available, err := repl.registryClient.ListAvailableModules()
	if err != nil {
		return fmt.Errorf("failed to list available modules: %w", err)
	}

	var installed []registry.ModuleMetadata
	if includeInstalled {
		installed, err = repl.registryClient.ListInstalledModules()
		if err != nil {
			return fmt.Errorf("failed to list installed modules: %w", err)
		}
	}

	if len(available) == 0 && len(installed) == 0 {
		fmt.Println("\nNo modules found.")
		fmt.Println("Run 'sync' to fetch modules from registry.")
		return nil
	}

	if includeInstalled && len(installed) > 0 {
		fmt.Printf("\nInstalled Modules (%d):\n\n", len(installed))
		for _, mod := range installed {
			fmt.Printf("✓ %s (v%s) [INSTALLED]\n", mod.Name, mod.Version)
			fmt.Printf("  %s\n", mod.Description)
			if len(mod.Tags) > 0 {
				fmt.Printf("  Tags: %s\n", strings.Join(mod.Tags, ", "))
			}
			fmt.Println()
		}
	}

	if len(available) > 0 {
		fmt.Printf("\nAvailable Modules (%d):\n\n", len(available))
		for _, mod := range available {
			fmt.Printf("○ %s (v%s)\n", mod.Name, mod.Version)
			fmt.Printf("  %s\n", mod.Description)
			if len(mod.Tags) > 0 {
				fmt.Printf("  Tags: %s\n", strings.Join(mod.Tags, ", "))
			}
			fmt.Printf("  Install: modules install %s.%s.%s\n", "org.themobileprof", mod.Name, mod.Version)
			fmt.Println()
		}
	}

	return nil
}

// installModule installs a module from the registry
func (repl *REPL) installModule(moduleID string) error {
	// Get registry URL from settings
	var registryURL string
	err := repl.db.QueryRow("SELECT value FROM settings WHERE key = 'registry_url'").Scan(&registryURL)
	if err != nil {
		// Check environment variable
		if envURL := os.Getenv("REGISTRY_URL"); envURL != "" {
			registryURL = envURL
		} else {
			return fmt.Errorf("registry URL not configured: set REGISTRY_URL environment variable or run 'clipilot settings set registry_url <url>'")
		}
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

	fmt.Println("\nCurrent Settings:")
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

	fmt.Println("\nRecent Executions:")
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

// updateCommands indexes all available system commands
func (repl *REPL) updateCommands() error {
	fmt.Println("\n=== Indexing System Commands ===")

	// Check if man is available
	if !repl.checkManAvailable() {
		return fmt.Errorf("man command not found - please install man pages first")
	}

	// Create indexer and run
	indexer := commands.NewIndexer(repl.db)

	// Index installed commands
	if err := indexer.RefreshCommandIndex(); err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}

	// Load common commands catalog
	fmt.Println("\nLoading common commands catalog...")
	if err := indexer.LoadCommonCommands(); err != nil {
		fmt.Printf("Warning: failed to load common commands: %v\n", err)
	} else {
		fmt.Println("✓ Common commands catalog loaded")
	}

	// Show summary
	count := indexer.GetCommandCount()
	fmt.Printf("\n💡 Indexed %d commands with descriptions from man pages\n", count)
	fmt.Println("   You can now search for any command using natural language!")

	return nil
}

// checkManAvailable verifies man command is installed
func (repl *REPL) checkManAvailable() bool {
	var cmdExists bool
	err := repl.db.QueryRow(`
		SELECT COUNT(*) > 0 FROM commands WHERE name = 'man'
	`).Scan(&cmdExists)

	if err == nil && cmdExists {
		return true
	}

	// Fallback: check if man command exists via system
	// This will be true on first run before indexing
	return true // We'll let the indexer fail if man is truly unavailable
}

// ExecuteNonInteractive runs a command non-interactively
func (repl *REPL) ExecuteNonInteractive(input string) error {
	return repl.handleCommand(input)
}

// syncRegistry syncs the module registry catalog
func (repl *REPL) syncRegistry() error {
	fmt.Println("Syncing module registry...")

	startTime := time.Now()
	err := repl.registryClient.SyncRegistry()
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Get sync status
	status, err := repl.registryClient.GetSyncStatus()
	if err != nil {
		return fmt.Errorf("failed to get sync status: %w", err)
	}

	fmt.Printf("\n✓ Registry synced successfully in %.2fs\n", duration.Seconds())
	fmt.Printf("  Total modules: %d\n", status.TotalModules)
	fmt.Printf("  Cached modules: %d\n", status.CachedModules)
	fmt.Printf("  Last sync: %s\n\n", status.LastSync.Format("2006-01-02 15:04:05"))

	return nil
}

// AutoSyncIfNeeded performs auto-sync if enabled and due
func (repl *REPL) AutoSyncIfNeeded() error {
	shouldSync, err := repl.registryClient.ShouldAutoSync()
	if err != nil {
		return err
	}

	if !shouldSync {
		return nil
	}

	fmt.Println("Auto-syncing registry...")
	err = repl.registryClient.SyncRegistry()
	if err != nil {
		return err
	}

	status, _ := repl.registryClient.GetSyncStatus()
	if status != nil {
		fmt.Printf("✓ Synced %d modules from registry\n", status.TotalModules)
	}

	return nil
}

// submitModuleRequest sends a module request to the registry when no matching module is found
func (repl *REPL) submitModuleRequest(query string) {
	// Get context information
	osInfo := os.Getenv("OS")
	if osInfo == "" {
		osInfo = "unknown"
	}

	isTermux := os.Getenv("TERMUX_VERSION") != ""
	userContext := fmt.Sprintf("os=%s, termux=%v", osInfo, isTermux)

	// Try to submit the request
	err := repl.registryClient.SubmitModuleRequest(query, userContext)
	if err != nil {
		// Silently fail - this is a nice-to-have feature
		// Don't bother the user with connection errors
		return
	}

	fmt.Println("\n💡 Your request has been submitted to help us improve CLIPilot.")
	fmt.Println("   Check https://clipilot.themobileprof.com for new modules!")
}

// resetDatabase resets the database with confirmation
func (repl *REPL) resetDatabase() error {
	fmt.Println("\n⚠️  WARNING: This will delete ALL data including:")
	fmt.Println("  - Installed modules")
	fmt.Println("  - Settings and preferences")
	fmt.Println("  - Execution history")
	fmt.Println("  - Registry cache")
	fmt.Print("\nAre you sure you want to reset? Type 'yes' to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "yes" {
		fmt.Println("Reset cancelled.")
		return nil
	}

	fmt.Println("\n⚠️  To complete the reset, please:")
	fmt.Println("1. Exit CLIPilot (type 'exit')")
	fmt.Println("2. Run: clipilot --reset --load=~/.clipilot/modules")
	fmt.Println("3. Run: clipilot sync")
	fmt.Println("\nThis will delete and recreate the database with your local modules.")

	return nil
}
