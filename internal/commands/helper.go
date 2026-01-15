package commands

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// CommandHelper provides interactive help for system commands
// It uses the simple key->value table (commands) and extracts
// detailed info from man pages on demand
type CommandHelper struct {
	db *sql.DB
}

// NewCommandHelper creates a new command helper
func NewCommandHelper(db *sql.DB) *CommandHelper {
	return &CommandHelper{db: db}
}

// DescribeCommand provides an interactive description of a command
// This implements the workflow:
// 1. Show command name and brief description (from DB)
// 2. Offer options: usage, examples, options (from man), or execute
func (ch *CommandHelper) DescribeCommand(cmdName string) error {
	// Get command info from database
	info, err := ch.getCommandInfo(cmdName)
	if err != nil {
		return fmt.Errorf("command '%s' not found: %w", cmdName, err)
	}

	// Display basic info
	fmt.Printf("\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📦 Command: %s\n", info.Name)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📝 %s\n", info.Description)
	fmt.Printf("\n")

	// Show options menu
	return ch.showOptionsMenu(info)
}

// getCommandInfo retrieves command info from database
func (ch *CommandHelper) getCommandInfo(name string) (*CommandInfo, error) {
	var info CommandInfo
	err := ch.db.QueryRow(`
		SELECT name, COALESCE(description, ''), COALESCE(has_man, 0)
		FROM commands WHERE name = ?
	`, name).Scan(&info.Name, &info.Description, &info.HasMan)

	if err == sql.ErrNoRows {
		// Check if command exists but not indexed
		if ch.commandExists(name) {
			return &CommandInfo{
				Name:        name,
				Description: "Command exists but description not indexed",
				HasMan:      ch.hasManPage(name),
			}, nil
		}
		return nil, fmt.Errorf("command not found")
	}
	return &info, err
}

// showOptionsMenu displays interactive options for the command
func (ch *CommandHelper) showOptionsMenu(info *CommandInfo) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("What would you like to do?")
		fmt.Println()
		fmt.Println("  [u] Show usage/synopsis")
		fmt.Println("  [e] Show examples")
		fmt.Println("  [o] Show options/flags")
		fmt.Println("  [m] Open full man page")
		fmt.Println("  [r] Run/type the command")
		fmt.Println("  [q] Go back")
		fmt.Println()
		fmt.Print("Choice: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.ToLower(strings.TrimSpace(input))

		switch input {
		case "u", "usage":
			ch.showUsage(info.Name)
		case "e", "examples":
			ch.showExamples(info.Name)
		case "o", "options":
			ch.showOptions(info.Name)
		case "m", "man":
			ch.openManPage(info.Name)
		case "r", "run":
			return ch.runCommand(info.Name, reader)
		case "q", "quit", "back", "":
			return nil
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
		fmt.Println()
	}
}

// showUsage extracts and displays the SYNOPSIS section from man
func (ch *CommandHelper) showUsage(cmdName string) {
	fmt.Printf("\n📋 Usage for %s:\n", cmdName)
	fmt.Println("─────────────────────────────────────")

	section := ch.extractManSection(cmdName, "SYNOPSIS")
	if section == "" {
		// Fallback: try --help
		section = ch.extractHelpUsage(cmdName)
	}

	if section == "" {
		fmt.Println("Usage information not available.")
		fmt.Println("Try: man " + cmdName + " or " + cmdName + " --help")
	} else {
		fmt.Println(section)
	}
	fmt.Println()
}

// showExamples extracts and displays the EXAMPLES section from man
func (ch *CommandHelper) showExamples(cmdName string) {
	fmt.Printf("\n💡 Examples for %s:\n", cmdName)
	fmt.Println("─────────────────────────────────────")

	section := ch.extractManSection(cmdName, "EXAMPLES")
	if section == "" {
		section = ch.extractManSection(cmdName, "EXAMPLE")
	}

	if section == "" {
		// Generate basic example
		fmt.Printf("No examples in man page. Basic usage:\n")
		fmt.Printf("\n  %s --help\n", cmdName)
		fmt.Printf("  %s [options] [arguments]\n", cmdName)
	} else {
		fmt.Println(section)
	}
	fmt.Println()
}

// showOptions extracts and displays the OPTIONS section from man
func (ch *CommandHelper) showOptions(cmdName string) {
	fmt.Printf("\n⚙️  Options for %s:\n", cmdName)
	fmt.Println("─────────────────────────────────────")

	section := ch.extractManSection(cmdName, "OPTIONS")
	if section == "" {
		section = ch.extractManSection(cmdName, "FLAGS")
	}

	if section == "" {
		// Fallback: try --help
		output, err := exec.Command(cmdName, "--help").CombinedOutput()
		if err == nil && len(output) > 0 {
			// Extract lines that look like options
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "-") {
					fmt.Println(line)
				}
			}
		} else {
			fmt.Println("Options not available. Try: " + cmdName + " --help")
		}
	} else {
		fmt.Println(section)
	}
	fmt.Println()
}

// openManPage opens the full man page in the pager
func (ch *CommandHelper) openManPage(cmdName string) {
	fmt.Printf("\nOpening man page for %s...\n", cmdName)

	cmd := exec.Command("man", cmdName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to open man page: %v\n", err)
		fmt.Println("Try: man " + cmdName)
	}
}

// runCommand allows user to type and execute the command
func (ch *CommandHelper) runCommand(cmdName string, reader *bufio.Reader) error {
	fmt.Printf("\n🚀 Run %s\n", cmdName)
	fmt.Println("─────────────────────────────────────")
	fmt.Println("Type your command (or press Enter to run with no arguments):")
	fmt.Printf("\n$ %s ", cmdName)

	args, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	args = strings.TrimSpace(args)

	// Build full command
	fullCmd := cmdName
	if args != "" {
		fullCmd = cmdName + " " + args
	}

	fmt.Printf("\n⚡ Executing: %s\n", fullCmd)
	fmt.Println("─────────────────────────────────────")

	// Execute via shell for proper argument parsing
	cmd := exec.Command("sh", "-c", fullCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			fmt.Printf("\n⚠️  Command exited with code %d\n", exitErr.ExitCode())
		} else {
			fmt.Printf("\n❌ Command failed: %v\n", err)
		}
	} else {
		fmt.Println("\n✅ Command completed successfully")
	}

	return nil
}

// extractManSection extracts a specific section from man page
func (ch *CommandHelper) extractManSection(cmdName, sectionName string) string {
	// Run man with col to strip formatting
	cmd := exec.Command("sh", "-c", fmt.Sprintf("man %s 2>/dev/null | col -bx", cmdName))
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	text := string(output)
	lines := strings.Split(text, "\n")

	// Find section start
	sectionRegex := regexp.MustCompile(`(?i)^` + sectionName + `\s*$`)
	startIdx := -1
	for i, line := range lines {
		if sectionRegex.MatchString(strings.TrimSpace(line)) {
			startIdx = i + 1
			break
		}
	}

	if startIdx == -1 {
		return ""
	}

	// Extract until next section (line starting without whitespace)
	var result []string
	nextSectionRegex := regexp.MustCompile(`^[A-Z][A-Z ]+$`)

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Stop at next section header
		if nextSectionRegex.MatchString(trimmed) && trimmed != sectionName {
			break
		}

		result = append(result, line)

		// Limit output length
		if len(result) > 50 {
			result = append(result, "  ... (truncated, use 'm' for full man page)")
			break
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

// extractHelpUsage extracts usage from --help output
func (ch *CommandHelper) extractHelpUsage(cmdName string) string {
	output, err := exec.Command(cmdName, "--help").CombinedOutput()
	if err != nil {
		// Try -h
		output, err = exec.Command(cmdName, "-h").CombinedOutput()
		if err != nil {
			return ""
		}
	}

	lines := strings.Split(string(output), "\n")
	var usageLines []string

	inUsage := false
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "usage:") || strings.HasPrefix(lower, "usage") {
			inUsage = true
		}

		if inUsage {
			usageLines = append(usageLines, line)
			// Stop after a few lines
			if len(usageLines) >= 5 {
				break
			}
			// Stop at empty line after first usage
			if len(usageLines) > 1 && strings.TrimSpace(line) == "" {
				break
			}
		}
	}

	return strings.Join(usageLines, "\n")
}

// commandExists checks if a command exists in PATH
func (ch *CommandHelper) commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// hasManPage checks if a man page exists for the command
func (ch *CommandHelper) hasManPage(name string) bool {
	cmd := exec.Command("man", "-w", name)
	err := cmd.Run()
	return err == nil
}

// QuickHelp shows a brief one-liner for a command (non-interactive)
func (ch *CommandHelper) QuickHelp(cmdName string) (string, error) {
	var description string
	err := ch.db.QueryRow(`
		SELECT COALESCE(description, '')
		FROM commands WHERE name = ?
	`, cmdName).Scan(&description)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("command not found: %s", cmdName)
	}
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s - %s", cmdName, description), nil
}

// SuggestCommand suggests commands similar to the given name
func (ch *CommandHelper) SuggestCommand(input string) ([]CommandInfo, error) {
	rows, err := ch.db.Query(`
		SELECT name, COALESCE(description, ''), COALESCE(has_man, 0)
		FROM commands
		WHERE name LIKE ? OR description LIKE ?
		LIMIT 5
	`, "%"+input+"%", "%"+input+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []CommandInfo
	for rows.Next() {
		var info CommandInfo
		if err := rows.Scan(&info.Name, &info.Description, &info.HasMan); err != nil {
			continue
		}
		suggestions = append(suggestions, info)
	}

	return suggestions, nil
}
