package intent

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/themobileprof/clipilot/internal/commands"
	"github.com/themobileprof/clipilot/internal/interfaces"
	"github.com/themobileprof/clipilot/pkg/models"
)

// Detector handles intent detection using multiple strategies
type Detector struct {
	db            *sql.DB
	keywordThresh float64
	llmThresh     float64
	onlineEnabled bool
	hybridMatcher *HybridMatcher
	hybridEnabled bool
}

// NewDetector creates a new intent detector
func NewDetector(db *sql.DB) *Detector {
	return &Detector{
		db:            db,
		keywordThresh: 0.6,
		llmThresh:     0.6,
		onlineEnabled: false,
		hybridMatcher: nil,
		hybridEnabled: false,
	}
}

// Ensure Detector implements IntentClassifier interface
var _ interfaces.IntentClassifier = (*Detector)(nil)

// SetThresholds updates confidence thresholds
func (d *Detector) SetThresholds(keyword, llm float64) {
	d.keywordThresh = keyword
	d.llmThresh = llm
}

// EnableHybrid enables the hybrid offline intelligence matcher
func (d *Detector) EnableHybrid() error {
	if d.hybridMatcher == nil {
		d.hybridMatcher = NewHybridMatcher(d.db)
	}

	if err := d.hybridMatcher.Load(); err != nil {
		return fmt.Errorf("failed to load hybrid matcher: %w", err)
	}

	d.hybridEnabled = true
	return nil
}

// DisableHybrid disables hybrid matching
func (d *Detector) DisableHybrid() {
	d.hybridEnabled = false
}

// IsHybridEnabled returns whether hybrid matching is enabled
func (d *Detector) IsHybridEnabled() bool {
	return d.hybridEnabled && d.hybridMatcher != nil
}

// SetOnlineEnabled enables/disables online LLM fallback
func (d *Detector) SetOnlineEnabled(enabled bool) {
	d.onlineEnabled = enabled
}

// Detect performs intent detection using a multi-layered approach with result fusion
// 1. Hybrid Offline Intelligence (TF-IDF scores)
// 2. Database FTS Search (Exact/Prefix matches)
// 3. Result Merging & Re-ranking
func (d *Detector) Detect(input string) (*models.IntentResult, error) {
	candidates := make(map[string]models.Candidate)
	
	// Layer 1: Hybrid offline intelligence (TF-IDF + intent + category)
	if d.hybridEnabled && d.hybridMatcher != nil {
		hybridResult, err := d.hybridMatcher.Match(input)
		if err == nil {
			for _, c := range hybridResult.Candidates {
				// Normalize score for fusion
				candidates[c.ModuleID] = c
			}
		}
	}

	// Layer 2: FTS Search (Database Layer) via Indexer
	// This uses the robust FTS5 tables we added
	ftsResults, err := d.keywordSearchCommands(input)
	if err == nil {
		for _, c := range ftsResults.Candidates {
			if existing, ok := candidates[c.ModuleID]; ok {
				// boost existing score if found in FTS (Confirmation Boost)
				existing.Score = existing.Score + 0.5 // Significant boost for dual-match
				if existing.Score > 1.0 { existing.Score = 1.0 } // Cap at 1.0 (approximated)
				
				// Take description from FTS if it looks better (longer)
				if len(c.Description) > len(existing.Description) {
					existing.Description = c.Description
				}
				candidates[c.ModuleID] = existing
			} else {
				// Add new FTS-only match
				candidates[c.ModuleID] = c
			}
		}
	}

	// Layer 3: Online LLM fallback (if enabled)
	if d.onlineEnabled && len(candidates) == 0 {
		// TODO: implement online LLM call
		_ = d.onlineEnabled 
	}

	// Convert map back to list
	finalCandidates := []models.Candidate{}
	for _, c := range candidates {
		finalCandidates = append(finalCandidates, c)
	}

	// Sort by score
	for i := 0; i < len(finalCandidates); i++ {
		for j := i + 1; j < len(finalCandidates); j++ {
			if finalCandidates[j].Score > finalCandidates[i].Score {
				finalCandidates[i], finalCandidates[j] = finalCandidates[j], finalCandidates[i]
			}
		}
	}

	// Return top result
	if len(finalCandidates) > 0 {
		return &models.IntentResult{
			ModuleID:   finalCandidates[0].ModuleID,
			Confidence: finalCandidates[0].Score,
			Candidates: finalCandidates,
			Method:     "hybrid_merged",
		}, nil
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
    // This function can remain as is for finding MODULES by pattern
    // Ideally we should inject FTS here too for module descriptions, but let's focus on commands first
	return d.keywordSearchCommands(input) // Reuse our enhanced search
}

// keywordSearchCommands performs search using the FTS-enabled Indexer
func (d *Detector) keywordSearchCommands(input string) (*models.IntentResult, error) {
	indexer := commands.NewIndexer(d.db)
	
	candidates := []models.Candidate{}

	// 1. Search Installed Commands (using FTS)
	cmdResults, err := indexer.SearchCommands(input, 10)
	if err == nil {
		for _, cmd := range cmdResults {
			candidates = append(candidates, models.Candidate{
				ModuleID:    "cmd:" + cmd.Name,
				Name:        cmd.Name,
				Description: cmd.Description,
				Score:       0.8, // High baseline for FTS match
				Tags:        []string{"command"},
			})
		}
	}

	// 2. Search Common Commands Catalog (using FTS)
	commonResults, err := indexer.SearchCommonCommands(input, 5)
	if err == nil {
		for _, cmd := range commonResults {
			candidates = append(candidates, models.Candidate{
				ModuleID:    "common:" + cmd.Name,
				Name:        cmd.Name + " (not installed)",
				Description: cmd.Description + " - " + cmd.GetInstallCommand(),
				Score:       0.6, // Slightly lower for uninstalled
				Tags:        []string{"installable", cmd.Category},
			})
		}
	}

	return &models.IntentResult{
		Candidates: candidates,
		Method:     "fts",
	}, nil
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
