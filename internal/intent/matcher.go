package intent

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/themobileprof/clipilot/internal/models"
)

// HybridMatcher implements the offline intelligence matching system
type HybridMatcher struct {
	db         *sql.DB
	normalizer *TextNormalizer
	tfidf      *TFIDFEngine
	threshold  float64
}

// NewHybridMatcher creates a new hybrid matcher
func NewHybridMatcher(db *sql.DB) *HybridMatcher {
	return &HybridMatcher{
		db:         db,
		normalizer: NewTextNormalizer(),
		tfidf:      NewTFIDFEngine(),
		threshold:  0.3, // Minimum score for relevance
	}
}

// Load indexes all commands into the TF-IDF engine
func (m *HybridMatcher) Load() error {
	// Load installed commands
	rows, err := m.db.Query(`
		SELECT name, description
		FROM commands
		ORDER BY name
	`)
	if err != nil {
		return fmt.Errorf("failed to query commands: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, description string
		if err := rows.Scan(&name, &description); err != nil {
			continue
		}

		// Infer category from command name and description
		category := inferCategory(name, description)

		// Extract keywords from description
		keywords := extractKeywords(description)

		m.tfidf.AddDocument(Document{
			ID:          "cmd:" + name,
			Name:        name,
			Category:    category,
			Description: description,
			Keywords:    keywords,
		})
	}

	// Load common commands (installable suggestions)
	rows, err = m.db.Query(`
		SELECT name, description, category, COALESCE(keywords, '')
		FROM common_commands
		ORDER BY priority DESC
		LIMIT 100
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name, description, category, keywordStr string
			if err := rows.Scan(&name, &description, &category, &keywordStr); err != nil {
				continue
			}

			keywords := strings.Split(keywordStr, ",")
			if keywordStr == "" {
				keywords = extractKeywords(description)
			}

			m.tfidf.AddDocument(Document{
				ID:          "common:" + name,
				Name:        name,
				Category:    category,
				Description: description,
				Keywords:    keywords,
			})
		}
	}

	// Build TF-IDF index
	m.tfidf.BuildIndex()

	return nil
}

// Match finds the best commands for a user query
func (m *HybridMatcher) Match(query string) (*models.IntentResult, error) {
	// Step 1: Normalize query
	normalized, intent := m.normalizer.Normalize(query)

	// Step 2: TF-IDF search
	tfidfResults := m.tfidf.Search(normalized, 20)

	// Step 3: Hybrid scoring
	candidates := make([]models.Candidate, 0, len(tfidfResults))
	for _, result := range tfidfResults {
		// Compute hybrid score
		score := m.computeHybridScore(result, intent, normalized)

		if score < m.threshold {
			continue
		}

		// Determine if command is installed or installable
		cmdType := "command"
		cmdName := result.Document.Name
		if strings.HasPrefix(result.Document.ID, "common:") {
			cmdName += " (not installed)"
			cmdType = "installable"
		}

		candidates = append(candidates, models.Candidate{
			ModuleID:    result.Document.ID,
			Name:        cmdName,
			Description: result.Document.Description,
			Score:       score,
			Tags:        []string{cmdType, result.Document.Category},
		})
	}

	// Step 4: Fallback if no good matches
	if len(candidates) == 0 {
		candidates = m.getFallbackSuggestions(intent, normalized)
	}

	// Build result
	result := &models.IntentResult{
		Candidates: candidates,
		Method:     "hybrid",
	}

	if len(candidates) > 0 {
		result.ModuleID = candidates[0].ModuleID
		result.Confidence = candidates[0].Score
	}

	return result, nil
}

// computeHybridScore combines multiple signals
func (m *HybridMatcher) computeHybridScore(result ScoredDocument, intent, query string) float64 {
	// Base TF-IDF score (0.45 weight)
	tfidfScore := result.Score * 0.45

	// Intent match bonus (0.35 weight)
	intentScore := m.computeIntentScore(result.Document, intent) * 0.35

	// Category boost (0.20 weight)
	categoryScore := m.computeCategoryScore(result.Document, query) * 0.20

	return tfidfScore + intentScore + categoryScore
}

// computeIntentScore checks if command matches the intent
func (m *HybridMatcher) computeIntentScore(doc Document, intent string) float64 {
	// Check if command name or description contains intent-related words
	text := strings.ToLower(doc.Name + " " + doc.Description)

	intentKeywords := map[string][]string{
		"show":      {"list", "show", "display", "print", "get"},
		"find":      {"find", "search", "locate", "grep", "which"},
		"kill":      {"kill", "stop", "terminate", "pkill"},
		"monitor":   {"monitor", "watch", "tail", "top", "stat"},
		"start":     {"start", "run", "launch", "service"},
		"install":   {"install", "add", "setup"},
		"remove":    {"remove", "delete", "uninstall", "rm"},
		"configure": {"config", "set", "change", "edit"},
		"test":      {"test", "ping", "check", "verify"},
	}

	keywords := intentKeywords[intent]
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return 1.0
		}
	}

	return 0.3 // Partial credit
}

// computeCategoryScore boosts relevant categories
func (m *HybridMatcher) computeCategoryScore(doc Document, query string) float64 {
	categoryKeywords := map[string][]string{
		"networking":  {"network", "port", "socket", "tcp", "udp", "ip", "dns", "route"},
		"process":     {"process", "pid", "cpu", "memory", "thread", "job"},
		"filesystem":  {"file", "directory", "disk", "mount", "path", "folder"},
		"system":      {"system", "info", "kernel", "hardware", "device"},
		"security":    {"user", "permission", "secure", "auth", "key", "firewall"},
		"development": {"git", "code", "build", "compile", "debug", "python", "node"},
	}

	query = strings.ToLower(query)
	for category, keywords := range categoryKeywords {
		if doc.Category == category {
			for _, keyword := range keywords {
				if strings.Contains(query, keyword) {
					return 1.0
				}
			}
		}
	}

	return 0.5 // Neutral
}

// getFallbackSuggestions returns top commands when primary search fails
func (m *HybridMatcher) getFallbackSuggestions(intent, query string) []models.Candidate {
	// Try category-based suggestions
	categories := m.guessCategoriesFromQuery(query)

	candidates := []models.Candidate{}
	for _, category := range categories {
		rows, err := m.db.Query(`
			SELECT name, description
			FROM commands
			WHERE category = ?
			ORDER BY name
			LIMIT 3
		`, category)

		if err == nil {
			for rows.Next() {
				var name, description string
				if err := rows.Scan(&name, &description); err == nil {
					candidates = append(candidates, models.Candidate{
						ModuleID:    "cmd:" + name,
						Name:        name,
						Description: description,
						Score:       0.25,
						Tags:        []string{"command", category},
					})
				}
			}
			rows.Close()
		}

		if len(candidates) >= 5 {
			break
		}
	}

	return candidates
}

// guessCategoriesFromQuery infers likely categories from query words
func (m *HybridMatcher) guessCategoriesFromQuery(query string) []string {
	query = strings.ToLower(query)
	categories := []string{}

	categoryHints := map[string][]string{
		"networking":  {"network", "port", "socket", "connect", "tcp", "udp", "dns"},
		"process":     {"process", "cpu", "memory", "running", "pid", "kill"},
		"filesystem":  {"file", "directory", "disk", "space", "mount", "folder"},
		"system":      {"system", "hardware", "kernel", "device", "info"},
		"development": {"git", "code", "build", "python", "node", "compile"},
	}

	for category, hints := range categoryHints {
		for _, hint := range hints {
			if strings.Contains(query, hint) {
				categories = append(categories, category)
				break
			}
		}
	}

	if len(categories) == 0 {
		categories = []string{"general", "system"}
	}

	return categories
}

// extractKeywords extracts searchable keywords from text
func extractKeywords(text string) []string {
	text = strings.ToLower(text)
	words := strings.Fields(text)

	// Filter out very common words
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true,
		"command": true, "utility": true, "tool": true,
	}

	keywords := []string{}
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// inferCategory attempts to categorize a command based on its name and description
func inferCategory(name, description string) string {
	text := strings.ToLower(name + " " + description)

	// Category detection rules (in priority order)
	if strings.Contains(text, "network") || strings.Contains(text, "socket") ||
		strings.Contains(text, "port") || strings.Contains(text, "tcp") ||
		strings.Contains(text, "udp") || strings.Contains(text, "dns") ||
		strings.Contains(text, "route") || strings.Contains(text, "ping") ||
		name == "ss" || name == "netstat" || name == "nc" || name == "nmap" {
		return "networking"
	}

	if strings.Contains(text, "process") || strings.Contains(text, "pid") ||
		strings.Contains(text, "cpu") || strings.Contains(text, "memory") ||
		strings.Contains(text, "thread") || name == "ps" || name == "top" ||
		name == "htop" || name == "kill" || name == "pkill" {
		return "process"
	}

	if strings.Contains(text, "file") || strings.Contains(text, "directory") ||
		strings.Contains(text, "disk") || strings.Contains(text, "mount") ||
		strings.Contains(text, "copy") || strings.Contains(text, "move") ||
		name == "ls" || name == "cp" || name == "mv" || name == "rm" ||
		name == "mkdir" || name == "find" || name == "du" || name == "df" {
		return "filesystem"
	}

	if strings.Contains(text, "git") || strings.Contains(text, "code") ||
		strings.Contains(text, "build") || strings.Contains(text, "compile") ||
		strings.Contains(text, "debug") || strings.Contains(text, "python") ||
		strings.Contains(text, "node") || strings.Contains(text, "java") ||
		name == "git" || name == "gcc" || name == "make" || name == "npm" {
		return "development"
	}

	if strings.Contains(text, "user") || strings.Contains(text, "permission") ||
		strings.Contains(text, "secure") || strings.Contains(text, "auth") ||
		strings.Contains(text, "key") || strings.Contains(text, "firewall") ||
		name == "chmod" || name == "chown" || name == "sudo" || name == "ssh" {
		return "security"
	}

	if strings.Contains(text, "system") || strings.Contains(text, "kernel") ||
		strings.Contains(text, "hardware") || strings.Contains(text, "device") ||
		name == "uname" || name == "lsb_release" || name == "dmesg" {
		return "system"
	}

	if strings.Contains(text, "database") || strings.Contains(text, "sql") ||
		name == "mysql" || name == "psql" || name == "mongo" || name == "redis" {
		return "database"
	}

	return "general"
}
