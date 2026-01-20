package intent

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/themobileprof/clipilot/internal/commands"
	"github.com/themobileprof/clipilot/internal/interfaces"
	"github.com/themobileprof/clipilot/internal/journey"
	"github.com/themobileprof/clipilot/internal/models"
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

// IsOnlineEnabled returns whether online fallback is enabled
func (d *Detector) IsOnlineEnabled() bool {
	return d.onlineEnabled
}

// Detect performs intent detection using live system search (man/apropos)
func (d *Detector) Detect(input string) (*models.IntentResult, error) {
	logger := journey.GetLogger()
	
	candidates := make(map[string]models.Candidate)
	
	// Layer 1: Live System Search (apropos)
	startMan := time.Now()
	manResults, err := d.searchSystemManPages(input)
	countMan := 0
	topScoreMan := 0.0
	
	if err == nil {
		for _, c := range manResults {
			countMan++
			if c.Score > topScoreMan { topScoreMan = c.Score }
			candidates[c.ModuleID] = c
		}
	}
	logger.AddStep("apropos", countMan, topScoreMan, time.Since(startMan), "")

	// Layer 2: Keyword Search (Modules & Cached Commands)
	// This covers intent_patterns and FTS on local DB
	kwResults, err := d.keywordSearch(input)
	if err == nil {
		for _, c := range kwResults.Candidates {
			// Don't overwrite higher score from apropos if exists, but usually keyword is better structured
			if existing, ok := candidates[c.ModuleID]; ok {
				if c.Score > existing.Score {
					candidates[c.ModuleID] = c
				}
			} else {
				candidates[c.ModuleID] = c
			}
		}
		// logger.AddStep("keyword", len(kwResults.Candidates), 0, 0, "") 
	}

	// Layer 3: Common Commands Catalog (DB) - Intent-based fallback
	// If live search found nothing or low confidence, search our curated catalog
	maxScore := topScoreMan
	if maxScore < 0.6 {
		startDB := time.Now()
		// Simple LIKE search on catalog
		catalogResults, err := d.searchCommonCommands(input, 5)
		if err == nil {
			for _, cmd := range catalogResults {
				// Avoid duplicates if already found installed
				if _, ok := candidates["cmd:"+cmd.Name]; ok {
					continue
				}
				
				score := 0.7 // Good fallback
				if cmd.Priority > 80 { score += 0.1 }
				
				c := models.Candidate{
					ModuleID:    "common:" + cmd.Name,
					Name:        cmd.Name + " (not installed)",
					Description: cmd.Description,
					Score:       score,
					Tags:        []string{"catalog", cmd.Category},
				}
				candidates[c.ModuleID] = c
			}
		}
		logger.AddStep("catalog", len(catalogResults), 0.7, time.Since(startDB), "")
	}

	// Layer 3: Online Semantic Search Fallback (if enabled)
	if d.onlineEnabled && len(candidates) == 0 {
		startRemote := time.Now()
		remoteResults, err := d.remoteSearch(input)
		if err == nil {
			for _, c := range remoteResults {
				candidates[c.ModuleID] = c
			}
		}
		logger.AddStep("remote", len(remoteResults), 0, time.Since(startRemote), "")
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
	
	logger.SetFinalCandidates(finalCandidates)

	if len(finalCandidates) > 0 {
		return &models.IntentResult{
			ModuleID:   finalCandidates[0].ModuleID,
			Confidence: finalCandidates[0].Score,
			Candidates: finalCandidates,
			Method:     "live_search",
		}, nil
	}

	return &models.IntentResult{
		ModuleID:   "",
		Confidence: 0.0,
		Method:     "none",
		Candidates: []models.Candidate{},
	}, nil
}

// searchSystemManPages searches using 'apropos' for each keyword
func (d *Detector) searchSystemManPages(input string) ([]models.Candidate, error) {
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return nil, nil
	}

	// Count occurrences of each command across all token searches
	// matches stores command -> {description, hitCount}
	type matchInfo struct {
		description string
		hits        int
		exact       bool
	}
	matches := make(map[string]*matchInfo)

	for _, token := range tokens {
		// Run apropos -s 1,8 <token> to search user and admin commands
		// -s 1,8 helps filter out C function calls (section 2,3)
		cmd := exec.Command("apropos", "-s", "1,8", token)
		output, err := cmd.Output()
		if err != nil {
			continue 
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" { continue }

			// Parse: name (section) - description
			parts := strings.SplitN(line, " - ", 2)
			if len(parts) != 2 { continue }

			left := parts[0]
			desc := parts[1]

			// Extract command name from "name (1)"
			// There might be multiple: "bzcmp, bzdiff (1)"
			names := strings.Split(left, ",")
			for _, name := range names {
				name = strings.TrimSpace(name)
				idx := strings.Index(name, "(")
				if idx != -1 {
					name = strings.TrimSpace(name[:idx])
				}
				
				if _, ok := matches[name]; !ok {
					matches[name] = &matchInfo{description: desc, hits: 0}
				}
				matches[name].hits++
				
				// Check for exact match with token
				if strings.EqualFold(name, token) {
					matches[name].exact = true
				}
			}
		}
	}

	// Validated candidates
	candidates := []models.Candidate{}
	
	for name, info := range matches {
		// Verify installation
		path, err := exec.LookPath(name)
		if err != nil {
			continue // Skip if not actually executable
		}
		
		// calculate score
		score := 0.4
		
		// Boost for exact token match
		if info.exact {
			score += 0.2
		}
		
		// Boost for multiple token hits (if query had multiple tokens)
		if len(tokens) > 1 {
			score += 0.15 * float64(info.hits - 1)
		}
		
		// Boost for priority commands (using catalog data)
		var priority int
		if err := d.db.QueryRow("SELECT priority FROM common_commands WHERE name = ?", name).Scan(&priority); err == nil {
			score += float64(priority) / 300.0 // Priority 100 adds ~0.33
		}
		
		// Boost for exact match with input line
		if strings.EqualFold(input, name) {
			score = 1.0
		}

		// Cap score
		if score > 1.0 { score = 1.0 }

		// Filter low scores
		if score < 0.5 { continue }

		candidates = append(candidates, models.Candidate{
			ModuleID:    "cmd:" + name,
			Name:        name, 
			Description: info.description,
			Score:       score,
			Tags:        []string{"command", "installed"},
		})
		
		// Ensure path is used
		_ = path 
	}

	return candidates, nil
}

// keywordSearch performs token-based search against intent_patterns
func (d *Detector) keywordSearch(input string) (*models.IntentResult, error) {
	tokens := tokenize(input)
	if len(tokens) == 0 {
		return &models.IntentResult{Candidates: []models.Candidate{}}, nil
	}

	// 1. Search Intent Patterns for Modules
	query := `
		SELECT m.id, m.name, m.version, m.description, p.weight
		FROM intent_patterns p
		JOIN modules m ON p.module_id = m.id
		WHERE p.pattern LIKE ?
	`
	
	candidates := []models.Candidate{}
	seen := make(map[string]bool)

	for _, token := range tokens {
		rows, err := d.db.Query(query, "%"+token+"%")
		if err != nil {
			continue
		}
		// Don't defer inside loop, close manually
		
		for rows.Next() {
			var modID, modName, modVer, modDesc string
			var weight float64
			if err := rows.Scan(&modID, &modName, &modVer, &modDesc, &weight); err != nil {
				continue
			}

			if seen[modID] { continue }
			seen[modID] = true

			candidates = append(candidates, models.Candidate{
				ModuleID:    modID,
				Name:        modName,
				Description: modDesc,
				Score:       0.7 * weight,
				Tags:        []string{"module", "keyword"},
			})
		}
		rows.Close()
	}

	// 2. Also search commands
	cmdRes, err := d.keywordSearchCommands(input)
	if err == nil && cmdRes != nil {
		candidates = append(candidates, cmdRes.Candidates...)
	}

	// Calculate top confidence
	var maxConf float64
	for _, c := range candidates {
		if c.Score > maxConf {
			maxConf = c.Score
		}
	}

	return &models.IntentResult{
		Candidates: candidates,
		Confidence: maxConf,
		Method:     "hybrid",
	}, nil
}

// keywordSearchCommands performs search using the FTS-enabled Indexer
func (d *Detector) keywordSearchCommands(input string) (*models.IntentResult, error) {
	indexer := commands.NewIndexer(d.db)
	
	candidates := []models.Candidate{}

	// 1. Search Installed Commands (using FTS)
	cmdResults, err := indexer.SearchCommands(input, 10)
	foundNames := make(map[string]bool)
	if err == nil {
		for _, cmd := range cmdResults {
			foundNames[cmd.Name] = true
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
			// Check if already installed
			isInstalled := false
			if _, err := exec.LookPath(cmd.Name); err == nil {
				isInstalled = true
			}

			// If already found in system index, skip to avoid duplicates
			// UNLESS the catalog has a better description? 
			// For now, let's prefer the system index implementation but if missing, use catalog.
			if foundNames[cmd.Name] {
				continue
			}

			if isInstalled {
				// Installed but not in system index (or low rank there)
				// Use catalog entry but mark as installed command
				candidates = append(candidates, models.Candidate{
					ModuleID:    "cmd:" + cmd.Name,
					Name:        cmd.Name,
					Description: cmd.Description,
					Score:       0.9, 
					Tags:        []string{"command", "catalog"},
				})
				continue
			}

			// Not installed
			name := cmd.Name
			description := cmd.Description
			
			name += " (not installed)"
			description += " - " + cmd.GetInstallCommand()
			score := 0.65 // Slightly lower but still visible

			candidates = append(candidates, models.Candidate{
				ModuleID:    "common:" + cmd.Name,
				Name:        name,
				Description: description,
				Score:       score,
				Tags:        []string{"catalog", cmd.Category},
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
		"why": true, "does": true, "did": true, "is": true,
		"are": true, "was": true, "were": true, "be": true,
		"my": true, "your": true, "his": true, "her": true,
		"me": true, "i": true, "you": true, "it": true,
		"want": true, "need": true, "like": true, "to": true,
		"do": true, "a": true, "an": true, "in": true,
		"on": true, "at": true, "from": true, "by": true,
	}

	// Synonyms for better man page matching
	synonyms := map[string][]string{
		"folder":   {"directory"},
		"folders":  {"directory"},
		"dir":      {"directory"},
		"make":     {"create", "build"}, // 'make' matches 'make', 'create' matches 'mkdir'
		"delete":   {"remove"},
		"del":      {"remove"},
		"remove":   {"delete"},
		"move":     {"rename"},
		"copy":     {"duplicate"},
		"edit":     {"editor", "modify", "change"},
		"download": {"fetch", "get", "retrieve"},
		"web":      {"internet", "url"},
		"ram":      {"memory"},
		"mem":      {"memory"},
		"storage":  {"disk", "space"},
		"space":    {"disk"},
		"disk":     {"storage"},
		"unzip":    {"extract", "archive", "zip"},
		"zip":      {"archive", "compress"},
		"config":   {"configuration"},
		"install":  {"package"},
		"kill":     {"terminate", "process"},
		"list":     {"show", "display"},
	}

	filtered := []string{}
	seen := make(map[string]bool)

	addToken := func(t string) {
		if !seen[t] {
			filtered = append(filtered, t)
			seen[t] = true
		}
	}

	for _, token := range tokens {
		if len(token) < 2 { // Allow 2-char tokens (e.g. 'cp', 'ls', 'ln')
			continue
		}
		if stopWords[token] {
			continue
		}
		
		addToken(token)

		// Add synonyms
		if syns, ok := synonyms[token]; ok {
			for _, s := range syns {
				addToken(s)
			}
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

// remoteSearch calls the server's semantic search endpoint
func (d *Detector) remoteSearch(query string) ([]models.Candidate, error) {
	serverURL := os.Getenv("CLIPILOT_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	apiURL := fmt.Sprintf("%s/api/commands/search", serverURL)
	
	reqBody, _ := json.Marshal(map[string]string{
		"query": query,
	})
	
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error: %s", resp.Status)
	}
	
	// Define response structure locally to match server response
	type geminiEnhancement struct {
		EnhancedDescription string   `json:"enhanced_description"`
		Keywords            []string `json:"keywords"`
		Category            string   `json:"category"`
		UseCases            []string `json:"use_cases"`
	}
	
	var data struct {
		Candidates []geminiEnhancement `json:"candidates"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	
	candidates := []models.Candidate{}
	for _, c := range data.Candidates {
		// Use first keyword as command name if possible, or fallback
		name := "unknown"
		if len(c.Keywords) > 0 {
			name = c.Keywords[0]
		}
		
		candidates = append(candidates, models.Candidate{
			ModuleID:    "remote:" + name,
			Name:        name + " (AI Suggested)",
			Description: c.EnhancedDescription,
			Score:       0.9, // High confidence for AI result
			Tags:        append(c.Keywords, "ai-suggested"),
		})
	}
	
	return candidates, nil
}
