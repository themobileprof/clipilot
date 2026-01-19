package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CommandSyncRequest represents commands submitted by a user
type CommandSyncRequest struct {
	Commands []UserCommand `json:"commands"`
}

// UserCommand represents a command from user's system
type UserCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CommandSyncResponse returns enhanced descriptions
type CommandSyncResponse struct {
	Enhanced []EnhancedCommand `json:"enhanced"`
	Message  string            `json:"message"`
}

// EnhancedCommand represents a command with enhanced description
type EnhancedCommand struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	EnhancedDescription string   `json:"enhanced_description"`
	Keywords            []string `json:"keywords"`
	Category            string   `json:"category"`
	UseCases            []string `json:"use_cases"`
}

// HandleCommandSync handles bidirectional command sync
// POST /api/commands/sync
func HandleCommandSync(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CommandSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Identify which commands need enhancement (not already in enhanced_commands)
		newCommands := []UserCommand{}
		for _, cmd := range req.Commands {
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM enhanced_commands WHERE name = ?)", cmd.Name).Scan(&exists)
			if err == nil && !exists {
				newCommands = append(newCommands, cmd)
			}
		}

		// Submit only NEW commands to server for future enhancement
		submittedCount := 0
		for _, cmd := range newCommands {
			err := submitCommand(db, cmd)
			if err == nil {
				submittedCount++
			}
		}

		// Fetch enhanced descriptions for user's commands (only returns enhanced ones)
		enhanced, err := getEnhancedCommands(db, req.Commands)
		if err != nil {
			http.Error(w, "Failed to fetch enhancements", http.StatusInternalServerError)
			return
		}

		response := CommandSyncResponse{
			Enhanced: enhanced,
			Message:  fmt.Sprintf("Synced %d commands, received %d enhancements (%d new discoveries)", len(req.Commands), len(enhanced), submittedCount),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// submitCommand adds a command to the submissions queue
func submitCommand(db *sql.DB, cmd UserCommand) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO command_submissions (command_name, user_description)
		VALUES (?, ?)
	`, cmd.Name, cmd.Description)
	return err
}

// getEnhancedCommands fetches enhanced descriptions for given commands
func getEnhancedCommands(db *sql.DB, commands []UserCommand) ([]EnhancedCommand, error) {
	if len(commands) == 0 {
		return []EnhancedCommand{}, nil
	}

	// Build IN clause
	placeholders := make([]string, len(commands))
	args := make([]interface{}, len(commands))
	for i, cmd := range commands {
		placeholders[i] = "?"
		args[i] = cmd.Name
	}

	query := fmt.Sprintf(`
		SELECT name, description, 
		       COALESCE(enhanced_description, description) as enhanced_description,
		       COALESCE(keywords, '') as keywords,
		       COALESCE(category, '') as category,
		       COALESCE(use_cases, '') as use_cases
		FROM enhanced_commands
		WHERE name IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	enhanced := []EnhancedCommand{}
	for rows.Next() {
		var cmd EnhancedCommand
		var keywordsStr, useCasesStr string

		err := rows.Scan(
			&cmd.Name,
			&cmd.Description,
			&cmd.EnhancedDescription,
			&keywordsStr,
			&cmd.Category,
			&useCasesStr,
		)
		if err != nil {
			continue
		}

		// Parse comma-separated strings
		if keywordsStr != "" {
			cmd.Keywords = strings.Split(keywordsStr, ",")
		}
		if useCasesStr != "" {
			cmd.UseCases = strings.Split(useCasesStr, ",")
		}

		enhanced = append(enhanced, cmd)
	}

	return enhanced, nil
}

// HandleEnhanceCommand enhances a command using Gemini
// POST /api/commands/enhance (admin only)
func HandleEnhanceCommand(db *sql.DB, geminiAPIKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// TODO: Add admin authentication check

		var req struct {
			CommandName string `json:"command_name"`
			Description string `json:"description"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Enhance using Gemini
		enhanced, err := enhanceWithGemini(geminiAPIKey, req.CommandName, req.Description)
		if err != nil {
			http.Error(w, fmt.Sprintf("Enhancement failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Save to database
		_, err = db.Exec(`
			INSERT INTO enhanced_commands 
			(name, description, enhanced_description, keywords, category, use_cases, source, last_enhanced, enhancement_model)
			VALUES (?, ?, ?, ?, ?, ?, 'ai', ?, 'gemini-1.5-flash')
			ON CONFLICT(name) DO UPDATE SET
				enhanced_description = excluded.enhanced_description,
				keywords = excluded.keywords,
				category = excluded.category,
				use_cases = excluded.use_cases,
				last_enhanced = excluded.last_enhanced,
				version = version + 1,
				updated_at = strftime('%s', 'now')
		`, req.CommandName, req.Description, enhanced.EnhancedDescription,
			strings.Join(enhanced.Keywords, ","),
			enhanced.Category,
			strings.Join(enhanced.UseCases, ","),
			time.Now().Unix())

		if err != nil {
			http.Error(w, "Failed to save enhancement", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(enhanced)
	}
}

// GeminiEnhancement represents the AI-enhanced command data
type GeminiEnhancement struct {
	EnhancedDescription string   `json:"enhanced_description"`
	Keywords            []string `json:"keywords"`
	Category            string   `json:"category"`
	UseCases            []string `json:"use_cases"`
}

// enhanceWithGemini calls Gemini API to enhance command description
func enhanceWithGemini(apiKey, commandName, description string) (*GeminiEnhancement, error) {
	prompt := fmt.Sprintf(`You are enhancing command descriptions for a CLI assistant. Be CONSERVATIVE and CONCISE.

Command: %s
Original Description: %s

Provide:
1. Enhanced description (max 100 chars, add 3-5 searchable keywords)
2. Keywords (5-7 words users might search for)
3. Category (one of: networking, process, filesystem, system, development, security, general)
4. Use cases (3-5 brief phrases like "check open ports", "monitor connections")

Keep everything SHORT and ACCURATE. Focus on discoverability, not explanation.

Return ONLY valid JSON:
{
  "enhanced_description": "...",
  "keywords": ["word1", "word2", ...],
  "category": "...",
  "use_cases": ["phrase1", "phrase2", ...]
}`, commandName, description)

	// TODO: Actual Gemini API call
	// For now, return a mock enhancement
	_ = apiKey
	_ = prompt

	// Mock response - replace with actual Gemini call
	return &GeminiEnhancement{
		EnhancedDescription: description + " (enhanced placeholder)",
		Keywords:            []string{"keyword1", "keyword2"},
		Category:            "general",
		UseCases:            []string{"use case 1", "use case 2"},
	}, nil
}
