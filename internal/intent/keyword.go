package intent

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/themobileprof/clipilot/internal/interfaces"
	"github.com/themobileprof/clipilot/pkg/models"
)

// Detector handles intent detection using multiple strategies
type Detector struct {
	db            *sql.DB
	keywordThresh float64
	llmThresh     float64
	onlineEnabled bool
}

// NewDetector creates a new intent detector
func NewDetector(db *sql.DB) *Detector {
	return &Detector{
		db:            db,
		keywordThresh: 0.6,
		llmThresh:     0.6,
		onlineEnabled: false,
	}
}

// Ensure Detector implements IntentClassifier interface
var _ interfaces.IntentClassifier = (*Detector)(nil)

// SetThresholds updates confidence thresholds
func (d *Detector) SetThresholds(keyword, llm float64) {
	d.keywordThresh = keyword
	d.llmThresh = llm
}

// SetOnlineEnabled enables/disables online LLM fallback
func (d *Detector) SetOnlineEnabled(enabled bool) {
	d.onlineEnabled = enabled
}

// Detect performs intent detection using the hybrid pipeline
func (d *Detector) Detect(input string) (*models.IntentResult, error) {
	// Layer 1: Keyword/DB search
	result, err := d.keywordSearch(input)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	if result.Confidence >= d.keywordThresh {
		result.Method = "keyword"
		return result, nil
	}

	// Layer 2: Tiny Local LLM (TODO: implement)
	// llmResult, err := d.localLLM(input, result.Candidates)
	// if err == nil && llmResult.Confidence >= d.llmThresh {
	//     return llmResult, nil
	// }

	// Layer 3: Online LLM fallback (if enabled)
	if d.onlineEnabled {
		// TODO: implement online LLM call
		// return d.onlineLLM(input, result.Candidates)
		_ = d.onlineEnabled // Suppress empty branch warning until implementation
	}

	// Return best candidate from keyword search with low confidence
	if len(result.Candidates) > 0 {
		return result, nil
	}

	return &models.IntentResult{
		ModuleID:   "",
		Confidence: 0.0,
		Method:     "none",
		Candidates: []models.Candidate{},
	}, nil
}

// keywordSearch performs token-based search against intent_patterns
func (d *Detector) keywordSearch(input string) (*models.IntentResult, error) {
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return &models.IntentResult{
			Candidates: []models.Candidate{},
		}, nil
	}

	// Build scoring map for both modules and commands
	scores := make(map[string]float64)
	weights := make(map[string]float64)
	commandMatches := make(map[string]models.Candidate) // Store command matches separately

	// First, check for direct command matches (highest priority)
	for _, token := range tokens {
		rows, err := d.db.Query(`
			SELECT name, description, has_man
			FROM commands
			WHERE name = ? OR name LIKE ?
			LIMIT 5
		`, token, token+"%")
		if err == nil {
			for rows.Next() {
				var name, description string
				var hasMan bool
				if err := rows.Scan(&name, &description, &hasMan); err == nil {
					// Commands get very high weight (3.0) to prioritize over modules
					cmdID := "cmd:" + name
					scores[cmdID] = 3.0
					commandMatches[cmdID] = models.Candidate{
						ModuleID:    cmdID,
						Name:        name,
						Description: description,
						Score:       3.0,
						Tags:        []string{"command"},
					}
				}
			}
			rows.Close()
		}
	}

	// Query module patterns for each token
	for _, token := range tokens {
		rows, err := d.db.Query(`
			SELECT DISTINCT p.module_id, p.weight, p.pattern_type
			FROM intent_patterns p
			WHERE p.pattern LIKE ?
		`, "%"+token+"%")
		if err != nil {
			return nil, fmt.Errorf("pattern query failed: %w", err)
		}

		for rows.Next() {
			var moduleID string
			var weight float64
			var patternType string
			if err := rows.Scan(&moduleID, &weight, &patternType); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan failed: %w", err)
			}

			// Boost weight for exact matches and tags
			boost := 1.0
			if patternType == "tag" {
				boost = 1.5
			}

			scores[moduleID] += weight * boost
			weights[moduleID] += weight * boost
		}
		rows.Close()
	}

	// Normalize scores and get candidates
	candidates := []models.Candidate{}

	// Add command matches first (highest priority)
	for _, cmd := range commandMatches {
		candidates = append(candidates, cmd)
	}

	// Add module matches
	for moduleID, score := range scores {
		// Skip command IDs (already added)
		if strings.HasPrefix(moduleID, "cmd:") {
			continue
		}

		normalizedScore := score / float64(len(tokens))

		// Get module details
		var name, description, tags string
		err := d.db.QueryRow(`
			SELECT name, description, COALESCE(tags, '')
			FROM modules
			WHERE id = ? AND installed = 1
		`, moduleID).Scan(&name, &description, &tags)
		if err != nil {
			continue // Skip if module not found
		}

		tagList := []string{}
		if tags != "" {
			tagList = strings.Split(tags, ",")
		}

		candidates = append(candidates, models.Candidate{
			ModuleID:    moduleID,
			Name:        name,
			Description: description,
			Score:       normalizedScore,
			Tags:        tagList,
		})
	}

	// If no strong matches, check common commands catalog for suggestions
	if len(candidates) == 0 || (len(candidates) > 0 && candidates[0].Score < 1.5) {
		commonCmds, err := d.searchCommonCommands(input, 3)
		if err == nil && len(commonCmds) > 0 {
			for _, cmd := range commonCmds {
				candidates = append(candidates, models.Candidate{
					ModuleID:    "common:" + cmd.Name,
					Name:        cmd.Name + " (not installed)",
					Description: cmd.Description + " - " + cmd.InstallCmd,
					Score:       float64(cmd.Priority) / 100.0 * 2.0, // Convert priority to score
					Tags:        []string{"installable", cmd.Category},
				})
			}
		}
	}

	// Sort candidates by score (descending)
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].Score > candidates[i].Score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Build result
	result := &models.IntentResult{
		Candidates: candidates,
		Method:     "keyword",
	}

	if len(candidates) > 0 {
		result.ModuleID = candidates[0].ModuleID
		result.Confidence = candidates[0].Score
	}

	return result, nil
}

// tokenize breaks input into searchable tokens
func tokenize(text string) []string {
	text = strings.ToLower(text)
	// Remove punctuation
	text = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' || r == '_' || r == '-' {
			return r
		}
		return ' '
	}, text)

	// Replace separators with spaces
	text = strings.ReplaceAll(text, "_", " ")
	text = strings.ReplaceAll(text, "-", " ")

	tokens := strings.Fields(text)

	// Filter short tokens and common stop words
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true,
		"how": true, "can": true, "what": true, "where": true,
	}

	filtered := []string{}
	for _, token := range tokens {
		if len(token) >= 3 && !stopWords[token] {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// searchCommonCommands searches installable commands from catalog
func (d *Detector) searchCommonCommands(query string, limit int) ([]commonCommandSuggestion, error) {
	query = strings.ToLower(query)

	rows, err := d.db.Query(`
		SELECT name, description, category, priority,
		       apt_package, pkg_package, dnf_package, brew_package, arch_package
		FROM common_commands
		WHERE name LIKE ? OR description LIKE ? OR keywords LIKE ?
		ORDER BY priority DESC, name
		LIMIT ?
	`, "%"+query+"%", "%"+query+"%", "%"+query+"%", limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []commonCommandSuggestion{}
	for rows.Next() {
		var cmd commonCommandSuggestion
		var aptPkg, pkgPkg, dnfPkg, brewPkg, archPkg sql.NullString

		err := rows.Scan(
			&cmd.Name, &cmd.Description, &cmd.Category, &cmd.Priority,
			&aptPkg, &pkgPkg, &dnfPkg, &brewPkg, &archPkg,
		)
		if err != nil {
			continue
		}

		// Determine install command for current OS
		cmd.InstallCmd = getInstallCommand(
			aptPkg.String, pkgPkg.String, dnfPkg.String, brewPkg.String, archPkg.String,
		)

		if cmd.InstallCmd != "" {
			results = append(results, cmd)
		}
	}

	return results, nil
}

type commonCommandSuggestion struct {
	Name        string
	Description string
	Category    string
	Priority    int
	InstallCmd  string
}

// getInstallCommand returns OS-specific install command
func getInstallCommand(aptPkg, pkgPkg, dnfPkg, brewPkg, archPkg string) string {
	// Check for Termux
	if os.Getenv("TERMUX_VERSION") != "" || os.Getenv("PREFIX") != "" {
		if pkgPkg != "" {
			return "pkg install " + pkgPkg
		}
	}

	// Check for apt (Debian/Ubuntu)
	if _, err := exec.LookPath("apt"); err == nil {
		if aptPkg != "" {
			return "sudo apt install " + aptPkg
		}
	}

	// Check for dnf (Fedora/RHEL)
	if _, err := exec.LookPath("dnf"); err == nil {
		if dnfPkg != "" {
			return "sudo dnf install " + dnfPkg
		}
	}

	// Check for brew (macOS)
	if _, err := exec.LookPath("brew"); err == nil {
		if brewPkg != "" {
			return "brew install " + brewPkg
		}
	}

	// Check for pacman (Arch Linux)
	if _, err := exec.LookPath("pacman"); err == nil {
		if archPkg != "" {
			return "sudo pacman -S " + archPkg
		}
	}

	return ""
}
