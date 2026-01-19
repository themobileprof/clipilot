package commands

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SyncEnhancedCommands syncs local commands with server's enhanced descriptions
func SyncEnhancedCommands(db *sql.DB, registryURL string) error {
	// Gather local commands
	commands, err := getLocalCommands(db)
	if err != nil {
		return fmt.Errorf("failed to get local commands: %w", err)
	}

	fmt.Printf("📤 Sending %d commands to registry...\n", len(commands))

	// Prepare sync request
	reqBody := map[string]interface{}{
		"commands": commands,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call sync endpoint
	resp, err := http.Post(
		registryURL+"/api/commands/sync",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to sync with registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error: %s", string(body))
	}

	// Parse response
	var syncResp struct {
		Enhanced []struct {
			Name                string   `json:"name"`
			Description         string   `json:"description"`
			EnhancedDescription string   `json:"enhanced_description"`
			Keywords            []string `json:"keywords"`
			Category            string   `json:"category"`
			UseCases            []string `json:"use_cases"`
		} `json:"enhanced"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Apply enhancements to local database
	enhanced := 0
	for _, cmd := range syncResp.Enhanced {
		err := applyEnhancement(db, cmd.Name, cmd.EnhancedDescription, cmd.Keywords, cmd.Category)
		if err == nil {
			enhanced++
		}
	}

	fmt.Printf("✓ %s\n", syncResp.Message)
	fmt.Printf("✓ Applied %d enhancements locally\n", enhanced)

	return nil
}

// getLocalCommands retrieves all commands from local database
func getLocalCommands(db *sql.DB) ([]map[string]string, error) {
	rows, err := db.Query(`
		SELECT name, description
		FROM commands
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commands := []map[string]string{}
	for rows.Next() {
		var name, description string
		if err := rows.Scan(&name, &description); err != nil {
			continue
		}

		commands = append(commands, map[string]string{
			"name":        name,
			"description": description,
		})
	}

	return commands, nil
}

// applyEnhancement updates local command with enhanced description
func applyEnhancement(db *sql.DB, name, enhancedDesc string, keywords []string, category string) error {
	// Merge keywords into description for better search
	searchableDesc := enhancedDesc
	if len(keywords) > 0 {
		// Add keywords that aren't already in description
		for _, keyword := range keywords {
			if !strings.Contains(strings.ToLower(searchableDesc), strings.ToLower(keyword)) {
				searchableDesc += " " + keyword
			}
		}
	}

	_, err := db.Exec(`
		UPDATE commands 
		SET description = ?
		WHERE name = ?
	`, searchableDesc, name)

	return err
}
