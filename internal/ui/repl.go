package ui

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/themobileprof/clipilot/internal/commands"
	"github.com/themobileprof/clipilot/internal/engine"
	"github.com/themobileprof/clipilot/internal/intent"
	"github.com/themobileprof/clipilot/internal/journey"
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
	cmdHelper      *commands.CommandHelper
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

	// Create command helper
	cmdHelper := commands.NewCommandHelper(db)

	return &REPL{
		db:             db,
		loader:         loader,
		runner:         runner,
		detector:       detector,
		registryClient: registryClient,
		cmdHelper:      cmdHelper,
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
	case "describe", "desc":
		if len(args) == 0 {
			return fmt.Errorf("usage: describe <command_name>")
		}
		return repl.cmdHelper.DescribeCommand(args[0])
	case "model":
		return repl.handleModelCommand(args)
	case "modules":
		if len(args) == 0 {
			return fmt.Errorf("usage: modules <list|install|remove>")
		}
		return repl.handleModulesCommand(args)
	case "sync":
		return repl.syncRegistry()
	case "update-commands":
		return repl.updateCommands()
	case "sync-commands":
		return repl.syncCommands()
	case "reset":
		return repl.resetDatabase()
	case "uninstall":
		return repl.uninstallCLIPilot()
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
  
  describe <command>      Get detailed help for any command
                          Example: describe rsync
                          Shows usage, examples, options (from man pages)
                          
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

🧠 OFFLINE INTELLIGENCE (Pure Go, No CGO):

  model status            Check if hybrid matcher is enabled
  model enable            Enable TF-IDF + intent + category matching
  model disable           Use only keyword search
  model refresh           Rebuild TF-IDF index after adding commands

  When enabled, searches use:
  • Text normalization (verb lemmatization, stop words)
  • Intent extraction (show, find, kill, monitor, etc.)
  • TF-IDF similarity (like LLM, but deterministic)
  • Category boost (networking, process, filesystem, etc.)
  • Fallback ladder (never returns "I don't know")

🌐 KEEPING UP TO DATE:

  sync                    Get latest modules from internet
                          (Will download data - do this on WiFi!)
                          
  update-commands         Refresh list of tools on your device
                          (No internet needed - safe anytime)
                          
  sync-commands           Sync with registry for enhanced descriptions
                          (Sends your commands, receives enhancements)

📊 SETTINGS & HISTORY:

  settings                See your current configuration
  logs                    See what tasks you've run before
  
  uninstall               Remove CLIPilot from your system
                          (Don't worry - it will ask for confirmation!)

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
    
  > describe tar
    (Shows usage, examples, options for tar command)
    
  > modules list
    (See what automations you have)
    
  > how do I setup python
    (CLIPilot guides you through Python setup)
    
  > run detect_os
    (Tells you about your device)

💾 DATA USAGE TIPS (Important for limited internet):

  ✓ "search" - No internet needed (searches local database)
  ✓ "describe" - No internet needed (uses local man pages)
  ✓ "update-commands" - No internet needed
  ✓ "modules list" - No internet needed
  ⚠ "sync" - Uses internet (downloads module catalog)
  ⚠ "modules install" - Uses internet (downloads module)
  ⚠ "model download" - Uses internet (~23 MB download)

📱 REMEMBER: You can always type "help" to see this message again!

`)
	return nil
}

// searchModules searches for modules matching a query
// searchModules searches for commands (and shows module info separately)
// Note: Despite the name, this now primarily searches commands since Detect() focuses on commands
func (repl *REPL) searchModules(query string) error {
	if query == "" {
		fmt.Println("\n💡 To search, type what you're looking for after 'search'")
		fmt.Println("   Examples:")
		fmt.Println("   • search git")
		fmt.Println("   • search \"copy files\"")
		fmt.Println("   • search database")
		return nil
	}

	// Start journey logging
	journey.GetLogger().StartNewJourney(query)

	result, err := repl.detector.Detect(query)
	if err != nil {
		journey.GetLogger().EndJourney("error:" + err.Error())
		fmt.Printf("\n⚠️  Search had a problem: %v\n", err)
		fmt.Println("💡 Try simpler words or check your spelling")
		return nil
	}

	if len(result.Candidates) == 0 {
		fmt.Println("\n🔍 No matches found for your search.")
		fmt.Println()
		fmt.Println("💡 Tips:")
		fmt.Println("   • Try different words (e.g., 'copy' instead of 'duplicate')")
		fmt.Println("   • Check available modules: modules list")
		fmt.Println("   • Index system commands: update-commands")
		fmt.Println("   • Get new modules: sync (requires internet)")
		fmt.Println()
		fmt.Println("📝 Your search has been recorded to help improve CLIPilot!")
		journey.GetLogger().EndJourney("no_matches")
		return nil
	}

	fmt.Printf("\n✨ Found %d result(s) for '%s':\n\n", len(result.Candidates), query)
	for i, candidate := range result.Candidates {
		if i >= 10 {
			break // Limit to top 10
		}

		// Check if it's a command or installable
		if strings.HasPrefix(candidate.ModuleID, "cmd:") {
			cmdName := strings.TrimPrefix(candidate.ModuleID, "cmd:")
			fmt.Printf("%d. %s [COMMAND]\n", i+1, cmdName)
		} else if strings.HasPrefix(candidate.ModuleID, "common:") {
			cmdName := strings.TrimPrefix(candidate.ModuleID, "common:")
			fmt.Printf("%d. %s [NOT INSTALLED]\n", i+1, cmdName)
		} else {
			fmt.Printf("%d. %s (ID: %s)\n", i+1, candidate.Name, candidate.ModuleID)
		}

		fmt.Printf("   %s\n", candidate.Description)
		fmt.Printf("   Score: %.2f | Tags: %s\n\n", candidate.Score, strings.Join(candidate.Tags, ", "))
	}

	fmt.Println("💡 To run a command: Just type it in your terminal")
	fmt.Println("💡 For detailed help: describe <command_name>")
	fmt.Println("💡 To run a module: run <module_id>")

	return nil
}

// handleQuery processes natural language queries - shows COMMANDS and interactive help
// Note: Modules are now accessed directly via 'run <module_id>', not through natural language queries
func (repl *REPL) handleQuery(input string) error {
	// Start journey logging
	logger := journey.GetLogger()
	logger.StartNewJourney(input)

	result, err := repl.detector.Detect(input)
	if err != nil {
		fmt.Printf("\n⚠️  Could not understand your request: %v\n", err)
		fmt.Println("\n💡 Try using the 'describe' command for command help:")
		fmt.Printf("   describe <command_name>\n")
		journey.GetLogger().EndJourney("error:" + err.Error())
		return nil
	}

	if result.ModuleID == "" || len(result.Candidates) == 0 {
		fmt.Println("\n🤔 I couldn't find commands for what you're asking.")
		fmt.Println()
		fmt.Println("💡 Here are some things to try:")
		fmt.Println("   1. Be more specific: 'how to copy files' → 'copy files'")
		fmt.Println("   2. Use 'search' to find commands: search copy")
		fmt.Println("   3. Install missing tools: update-commands")
		fmt.Println("   4. Check available modules: modules list")
		fmt.Println()
		fmt.Println("📝 Your request helps us improve CLIPilot - thank you!")
		journey.GetLogger().EndJourney("no_matches")
		return nil
	}

	// Show command results with descriptions
	top := result.Candidates[0]

	// Check if it's a command (cmd:) or installable (common:)
	if strings.HasPrefix(top.ModuleID, "cmd:") {
		cmdName := strings.TrimPrefix(top.ModuleID, "cmd:")
		
		// Conversational opening based on confidence
		if result.Confidence >= 0.8 {
			fmt.Printf("\nI found the perfect tool for that: `%s`\n", cmdName)
			fmt.Printf("It helps you: %s\n", top.Description)
		} else if result.Confidence >= 0.6 {
			fmt.Printf("\nYou can use `%s` to do that.\n", cmdName)
			fmt.Printf("Description: %s\n", top.Description)
		} else {
			fmt.Printf("\nI'm not 100%% sure, but `%s` might be what you need.\n", cmdName)
			fmt.Printf("It is used to: %s\n", top.Description)
		}

		// Show other related commands if confidence is low
		if result.Confidence < 0.7 && len(result.Candidates) > 1 {
			fmt.Println("\nYou might also find these useful:")
			for i := 1; i < len(result.Candidates) && i < 4; i++ {
				name := result.Candidates[i].Name
				if strings.HasSuffix(name, " (not installed)") {
					continue // Skip installable suggestions for now
				}
				fmt.Printf("   • `%s` - %s\n", name, result.Candidates[i].Description)
			}
		}

		// Interactive menu
		fmt.Printf("\nHere is how I can help you with `%s`:\n", cmdName)
		fmt.Println("  (Not what you need? Choose option 4 to search again)")
		fmt.Println("  1. Show me examples and usage (recommended)")
		fmt.Println("  2. Run this command (interactive)")
		fmt.Println("  3. Just show me the command (and exit assistant)")
		fmt.Println("  4. No, search for a different command")
		fmt.Println("  0. Cancel")
		fmt.Print("\nChoice [1-4, 0]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		switch response {
		case "1":
			// Show detailed help using CommandHelper
			journey.GetLogger().EndJourney("describe:" + cmdName)
			return repl.cmdHelper.DescribeCommand(cmdName)
		case "2":
			// Run command
			journey.GetLogger().EndJourney("run:" + cmdName)
			return repl.cmdHelper.RunCommand(cmdName, reader)
		case "3":
			// Just show the command and exit
			journey.GetLogger().EndJourney("show:" + cmdName)
			fmt.Printf("\nYou can run it like this:\n")
			fmt.Printf("  %s --help\n\n", cmdName)
			return fmt.Errorf("exit")
		case "4":
			// Prompt for new search
			journey.GetLogger().EndJourney("retry_search")
			fmt.Print("\nWhat would you like to search for instead? ")
			newQuery, _ := reader.ReadString('\n')
			newQuery = strings.TrimSpace(newQuery)
			if newQuery != "" {
				return repl.handleQuery(newQuery)
			}
			return nil
		case "0", "":
			journey.GetLogger().EndJourney("cancel")
			return nil
		default:
			fmt.Println("Invalid choice. Use 'describe " + cmdName + "' for detailed help.")
			return nil
		}
	} else if strings.HasPrefix(top.ModuleID, "common:") {
		// Installable command suggestion
		cmdName := strings.TrimPrefix(top.ModuleID, "common:")
		fmt.Printf("\nThat looks like a job for `%s`, but it's not installed yet.\n", cmdName)
		fmt.Printf("Description: %s\n\n", top.Description)
		return nil
	}

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
	fmt.Println("\n=== Updating Command Catalog ===")

	// Create indexer
	indexer := commands.NewIndexer(repl.db)

	// Load common commands catalog
	fmt.Println("Loading common commands catalog (for ranking priority)...")
	if err := indexer.LoadCommonCommands(); err != nil {
		fmt.Printf("Warning: failed to load common commands: %v\n", err)
	} else {
		fmt.Println("✓ Common commands catalog loaded")
	}

	fmt.Println("\n💡 System commands are now discovered in real-time using 'apropos'.")
	fmt.Println("   No need to index them manually anymore!")

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

// syncCommands syncs local commands with registry's enhanced descriptions
func (repl *REPL) syncCommands() error {
	fmt.Println("\n=== Syncing Command Descriptions ===")
	fmt.Println("This will:")
	fmt.Println("  • Send your command list to the registry")
	fmt.Println("  • Receive AI-enhanced descriptions")
	fmt.Println("  • Update your local database")
	fmt.Println()

	// Get registry URL
	registryURL, err := registry.GetRegistryURL(repl.db)
	if err != nil {
		return fmt.Errorf("registry not configured: run 'sync' first")
	}

	// Perform sync
	if err := commands.SyncEnhancedCommands(repl.db, registryURL); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Println("\n💡 Run 'model refresh' to rebuild the search index with new descriptions")
	return nil
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

// handleModelCommand manages the offline intelligence system
func (repl *REPL) handleModelCommand(args []string) error {
	if len(args) == 0 {
		return repl.showModelStatus()
	}

	switch args[0] {
	case "status":
		return repl.showModelStatus()
	case "enable":
		return repl.enableHybrid()
	case "disable":
		repl.detector.DisableHybrid()
		fmt.Println("✓ Hybrid offline intelligence disabled")
		return nil
	case "refresh":
		return repl.refreshHybrid()
	case "online":
		if len(args) < 2 {
			return fmt.Errorf("usage: model online <on|off>")
		}
		if args[1] == "on" {
			repl.detector.SetOnlineEnabled(true)
			fmt.Println("✓ Online semantic search enabled (will use server fallback)")
		} else if args[1] == "off" {
			repl.detector.SetOnlineEnabled(false)
			fmt.Println("✓ Online semantic search disabled")
		} else {
			return fmt.Errorf("invalid argument: use 'on' or 'off'")
		}
		return nil
	default:
		fmt.Println("Model commands:")
		fmt.Println("  model status    - Show offline intelligence status")
		fmt.Println("  model enable    - Enable hybrid TF-IDF matcher")
		fmt.Println("  model disable   - Disable offline intelligence")
		fmt.Println("  model refresh   - Rebuild TF-IDF index")
		fmt.Println("  model online    - Toggle online semantic search (on/off)")
		return nil
	}
}

// showModelStatus displays offline intelligence system information
func (repl *REPL) showModelStatus() error {
	fmt.Println("\n📊 Offline Intelligence Status")
	fmt.Println("─────────────────────────────────────")

	// Check if enabled
	if repl.detector.IsHybridEnabled() {
		fmt.Println("Status:      ✓ Enabled (Hybrid TF-IDF Matcher)")
		fmt.Println("Method:      Pure Go, no CGO, no external models")
		fmt.Println("Features:    • Text normalization")
		fmt.Println("             • Intent extraction")
		fmt.Println("             • TF-IDF similarity")
		fmt.Println("             • Category boost")
	} else {
		fmt.Println("Status:      ○ Disabled (using keyword search)")
		fmt.Println("             Run 'model enable' to activate")
	}

	// Check online status
	if repl.detector.IsOnlineEnabled() {
		fmt.Println("Online:      ✓ Enabled (Server Fallback)")
	} else {
		fmt.Println("Online:      ○ Disabled")
		fmt.Println("             Run 'model online on' to activate")
	}

	// Get command stats
	var cmdCount int
	_ = repl.db.QueryRow("SELECT COUNT(*) FROM commands").Scan(&cmdCount)

	var commonCount int
	_ = repl.db.QueryRow("SELECT COUNT(*) FROM common_commands").Scan(&commonCount)

	fmt.Printf("\nIndexed Commands:\n")
	fmt.Printf("  Installed:  %d\n", cmdCount)
	fmt.Printf("  Catalog:    %d\n", commonCount)

	fmt.Println()
	return nil
}

// enableHybrid enables the hybrid offline intelligence matcher
func (repl *REPL) enableHybrid() error {
	fmt.Println("\n🔄 Enabling offline intelligence...")

	if err := repl.detector.EnableHybrid(); err != nil {
		return fmt.Errorf("failed to enable hybrid matcher: %w", err)
	}

	fmt.Println("✓ Hybrid matcher enabled!")
	fmt.Println("  Using TF-IDF + intent detection + category boost")
	fmt.Println("  No external models or CGO required")
	fmt.Println()

	return nil
}

// refreshHybrid rebuilds the TF-IDF index
func (repl *REPL) refreshHybrid() error {
	fmt.Println("\n🔄 Refreshing hybrid matcher...")

	if !repl.detector.IsHybridEnabled() {
		return fmt.Errorf("hybrid matcher not enabled - run 'model enable' first")
	}

	// Reload the index
	repl.detector.DisableHybrid()
	if err := repl.detector.EnableHybrid(); err != nil {
		return fmt.Errorf("failed to refresh hybrid matcher: %w", err)
	}

	fmt.Println("✓ Hybrid matcher refreshed!")
	return nil
}

// uninstallCLIPilot uninstalls CLIPilot with user confirmation
func (repl *REPL) uninstallCLIPilot() error {
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║            CLIPilot Uninstall                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Detect environment
	isTermux := os.Getenv("TERMUX_VERSION") != "" || os.Getenv("PREFIX") != ""

	// Determine installation paths
	var installDir string
	if isTermux {
		installDir = os.Getenv("PREFIX") + "/bin"
	} else {
		// Check common locations
		possiblePaths := []string{
			"/usr/local/bin/clipilot",
			filepath.Join(os.Getenv("HOME"), ".local", "bin", "clipilot"),
			filepath.Join(os.Getenv("HOME"), "bin", "clipilot"),
		}
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				installDir = filepath.Dir(path)
				break
			}
		}
	}

	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".clipilot")
	binaryPath := filepath.Join(installDir, "clipilot")

	// Show what will be removed
	fmt.Println("📋 Found the following CLIPilot components:")
	fmt.Println()

	foundBinary := false
	foundData := false

	if installDir != "" {
		if _, err := os.Stat(binaryPath); err == nil {
			fmt.Printf("  ✓ Binary: %s\n", binaryPath)
			foundBinary = true
		}
	}

	if _, err := os.Stat(dataDir); err == nil {
		fmt.Printf("  ✓ Data directory: %s\n", dataDir)
		foundData = true
	}

	if !foundBinary && !foundData {
		fmt.Println("  ○ No CLIPilot components found")
		fmt.Println()
		fmt.Println("CLIPilot does not appear to be installed.")
		return nil
	}

	fmt.Println()
	fmt.Println("⚠️  This will permanently delete:")
	if foundBinary {
		fmt.Println("   • CLIPilot binary")
	}
	if foundData {
		fmt.Println("   • All your modules and settings")
		fmt.Println("   • Command index and search history")
		fmt.Println("   • Execution logs")
	}
	fmt.Println()
	fmt.Println("🔴 This action cannot be undone!")
	fmt.Println()

	// Confirm
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Continue with uninstallation? [y/N]: ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "y" && response != "yes" {
		fmt.Println("\nUninstallation cancelled.")
		return nil
	}

	fmt.Println()
	fmt.Println("🗑️  Uninstalling CLIPilot...")
	fmt.Println()

	// Create a shell script that will execute after CLIPilot exits
	tmpScript := filepath.Join(os.TempDir(), "clipilot_uninstall.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# CLIPilot self-uninstall cleanup script
# This script runs after CLIPilot exits

sleep 1  # Wait for CLIPilot to fully exit

echo "Removing CLIPilot components..."

# Remove binary
if [ -f "%s" ]; then
    if rm -f "%s" 2>/dev/null; then
        echo "✓ Removed binary from %s"
    else
        if command -v sudo >/dev/null 2>&1; then
            if sudo rm -f "%s" 2>/dev/null; then
                echo "✓ Removed binary (using sudo)"
            else
                echo "✗ Failed to remove binary - you may need to run: sudo rm %s"
            fi
        else
            echo "✗ Failed to remove binary - you may need to remove it manually"
        fi
    fi
fi

# Remove data directory
if [ -d "%s" ]; then
    if rm -rf "%s" 2>/dev/null; then
        echo "✓ Removed data directory"
    else
        echo "✗ Failed to remove data directory - you may need to run: rm -rf %s"
    fi
fi

echo ""
echo "✅ CLIPilot has been uninstalled!"
echo ""
echo "📝 Note: Packages installed using CLIPilot modules were not removed."
echo ""
echo "Thank you for trying CLIPilot! 👋"
echo "To reinstall: https://github.com/themobileprof/clipilot"
echo ""

# Clean up this script
rm -f "$0"
`, binaryPath, binaryPath, installDir, binaryPath, binaryPath, dataDir, dataDir, dataDir)

	if err := os.WriteFile(tmpScript, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to create uninstall script: %w", err)
	}

	fmt.Println("📝 Uninstall script created. CLIPilot will now exit and complete uninstallation.")
	fmt.Println()

	// Execute the cleanup script in background and exit
	cmd := exec.Command("bash", tmpScript)
	if err := cmd.Start(); err != nil {
		os.Remove(tmpScript)
		return fmt.Errorf("failed to start uninstall: %w", err)
	}

	// Exit CLIPilot
	os.Exit(0)
	return nil
}
