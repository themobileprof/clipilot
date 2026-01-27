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
	"github.com/themobileprof/clipilot/internal/utils/safeexec"
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
	
	// Layer 1: Keyword Search (Modules & Cached/Common Commands)
	// This covers intent_patterns and FTS on local DB (Curated List).
	// We prioritize this because it uses our clean, vetted descriptions.
	kwResults, err := d.keywordSearch(input)
	var topScoreKW float64
	if err == nil {
		for _, c := range kwResults.Candidates {
			if c.Score > topScoreKW { topScoreKW = c.Score }
			candidates[c.ModuleID] = c
		}
	}

	// Optimization: If we found a high-confidence match (e.g. curated command installed locally),
	// bypass the slow and potentially fragile system man page search.
	// Common commands (cp, ls, git) will typically score >= 0.8 via FTS if installed.
	runApropos := true
	if topScoreKW >= 0.85 {
		runApropos = false
		logger.AddStep("keyword_hit", len(kwResults.Candidates), topScoreKW, 0, "bypassed_man")
	}

	var topScoreMan float64
	// Layer 2: Live System Search (apropos) - Fallback for unknown commands
	if runApropos {
		startMan := time.Now()
		manResults, err := d.searchSystemManPages(input)
		countMan := 0
		
		if err == nil {
			for _, c := range manResults {
				countMan++
				if c.Score > topScoreMan { topScoreMan = c.Score }
				// Only add if better than existing
				if existing, ok := candidates[c.ModuleID]; ok {
					if c.Score > existing.Score {
						candidates[c.ModuleID] = c
					}
				} else {
					candidates[c.ModuleID] = c
				}
			}
		}
		logger.AddStep("apropos", countMan, topScoreMan, time.Since(startMan), "")
	}

	// Layer 3: Common Commands Catalog (DB) - Intent-based fallback
	// If live search found nothing or low confidence, search our curated catalog for NOT installed suggestions
	maxScore := topScoreKW
	if topScoreMan > maxScore { maxScore = topScoreMan }

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
	// Helper to safely resolve paths on Termux (avoiding faccessat2 crash)
	// safeResolve logic removed in favor of centralized safeexec package

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
		
		// Use safeexec.LookPath instead of exec.LookPath default
		binPath, err := safeexec.LookPath("apropos")
		if err != nil {
			continue // apropos not found
		}

		cmd := exec.Command(binPath, "-s", "1,8", token)
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
		path, err := safeexec.LookPath(name)
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

	// 1. Search Common Commands Catalog (In-Memory)
	// PRIMARY SOURCE: Quick, reliable, works even if DB is broken
	foundNames := make(map[string]bool)
	if catalog, err := commands.GetCatalog(); err == nil {
		catalogResults := catalog.Search(input)
		for _, res := range catalogResults {
			cmd := res.Command
			isInstalled := res.Installed

			if isInstalled {
				// Installed command found in catalog
				candidates = append(candidates, models.Candidate{
					ModuleID:    "cmd:" + cmd.Name,
					Name:        cmd.Name,
					Description: cmd.Description,
					Score:       res.Score, 
					Tags:        []string{"command", "catalog"},
				})
				foundNames[cmd.Name] = true
				continue
			}

			// Not installed - we still add high-confidence matches as suggestions
			// but mark them clearly
			if res.Score > 0.8 {
				candidates = append(candidates, models.Candidate{
					ModuleID:    "common:" + cmd.Name,
					Name:        cmd.Name + " (not installed)",
					Description: cmd.Description,
					Score:       res.Score - 0.1, // Slight penalty
					Tags:        []string{"catalog", "installable"},
				})
			}
		}
	}

	// 2. Search Installed System Commands (SQLite FTS)
	// SECONDARY SOURCE: Covers the thousands of other system commands not in catalog
	// We only run this to find *other* commands.
	cmdResults, err := indexer.SearchCommands(input, 10)
	if err == nil {
		for _, cmd := range cmdResults {
			// Skip if we already found a better version in the catalog
			if foundNames[cmd.Name] {
				continue
			}
			
			candidates = append(candidates, models.Candidate{
				ModuleID:    "cmd:" + cmd.Name,
				Name:        cmd.Name,
				Description: cmd.Description,
				Score:       0.8, // High baseline for FTS match
				Tags:        []string{"command"},
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
		// Generic nouns that cause false positives
		"file": true, "files": true, "output": true,
	}

	// Synonyms for better man page matching
	synonyms := map[string][]string{
		"folder":   {"directory", "dir"},
		"folders":  {"directory"},
		"dir":      {"directory"},
		"make":     {"create", "build", "mkdir"}, 
		"delete":   {"remove", "rm"},
		"del":      {"remove", "rm"},
		"remove":   {"delete", "rm"},
		"move":     {"rename", "mv"},
		"copy":     {"duplicate", "cp"},
		"edit":     {"editor", "modify", "change", "nano", "vim"},
		"download": {"fetch", "get", "retrieve", "wget", "curl"},
		"web":      {"internet", "url"},
		"ram":      {"memory"},
		"mem":      {"memory"},
		"storage":  {"disk", "space"},
		"space":    {"disk"},
		"disk":     {"storage"},
		"unzip":    {"extract", "archive", "zip"},
		"zip":      {"archive", "compress"},
		"config":   {"configuration"},
		"install":  {"package", "get"},
		"kill":     {"terminate", "process"},
		"list":     {"show", "display", "ls"},
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
	catalog, err := commands.GetCatalog()
	if err != nil {
		return nil, err
	}

	results := catalog.Search(query)
	var suggestions []commonCommandSuggestion

	for _, r := range results {
		cmd := r.Command
		suggestions = append(suggestions, commonCommandSuggestion{
			Name:        cmd.Name,
			Description: cmd.Description,
			Category:    cmd.Category,
			Priority:    cmd.Priority,
			InstallCmd: getInstallCommand(
				cmd.AptPackage, cmd.PkgPackage, cmd.DnfPackage, 
				cmd.BrewPackage, cmd.ArchPackage,
			),
		})
	}
	return suggestions, nil
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
	if _, err := safeexec.LookPath("apt"); err == nil {
		if aptPkg != "" {
			return "sudo apt install " + aptPkg
		}
	}

	// Check for dnf (Fedora/RHEL)
	if _, err := safeexec.LookPath("dnf"); err == nil {
		if dnfPkg != "" {
			return "sudo dnf install " + dnfPkg
		}
	}

	// Check for brew (macOS)
	if _, err := safeexec.LookPath("brew"); err == nil {
		if brewPkg != "" {
			return "brew install " + brewPkg
		}
	}

	// Check for pacman (Arch Linux)
	if _, err := safeexec.LookPath("pacman"); err == nil {
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
