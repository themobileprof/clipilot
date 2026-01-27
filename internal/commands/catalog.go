package commands

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	yaml "gopkg.in/yaml.v3"
)

//go:embed common_commands.yaml
var embeddedCatalogData []byte

// CommonCommand represents a command from the catalog
type CommonCommand struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	Category      string `yaml:"category"`
	Keywords      string `yaml:"keywords"`
	AptPackage    string `yaml:"apt_package"`
	PkgPackage    string `yaml:"pkg_package"`
	DnfPackage    string `yaml:"dnf_package"`
	BrewPackage   string `yaml:"brew_package"`
	ArchPackage   string `yaml:"arch_package"`
	AlternativeTo string `yaml:"alternative_to"`
	Homepage      string `yaml:"homepage"`
	Priority      int    `yaml:"priority"`
}

// Global instance to avoid reloading
var (
	globalCatalog *Catalog
	catalogOnce   sync.Once
)

// Catalog holds the in-memory list of common commands
type Catalog struct {
	Commands []CommonCommand
	byName   map[string]*CommonCommand
}

// GetCatalog returns the singleton catalog instance, initializing it if needed
func GetCatalog() (*Catalog, error) {
	var err error
	catalogOnce.Do(func() {
		globalCatalog = &Catalog{
			byName: make(map[string]*CommonCommand),
		}
		
		// Parse embedded YAML
		if len(embeddedCatalogData) == 0 {
			err = fmt.Errorf("embedded common commands data is empty")
			return
		}

		if parseErr := yaml.Unmarshal(embeddedCatalogData, &globalCatalog.Commands); parseErr != nil {
			err = fmt.Errorf("failed to parse embedded catalog: %w", parseErr)
			return
		}

		// Index by name for fast lookups
		for i := range globalCatalog.Commands {
			cmd := &globalCatalog.Commands[i]
			globalCatalog.byName[cmd.Name] = cmd
		}
	})
	return globalCatalog, err
}

// SearchResult represents a matching command
type SearchResult struct {
	Command   *CommonCommand
	Score     float64
	Installed bool
}

// Search performs an efficient in-memory search
func (c *Catalog) Search(query string) []SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	tokens := strings.Fields(query)
	results := []SearchResult{}

	if len(tokens) == 0 {
		return results
	}

	// Cache installed check results for this search session (optimization)
	// Cache installed check results for this search session (optimization)
	installedCache := make(map[string]bool)
	checkInstalled := func(name string) bool {
		if val, ok := installedCache[name]; ok {
			return val
		}

		// Termux/Android Crash Fix:
		// Go >1.20 exec.LookPath calls faccessat2 which is blocked by seccomp on some Android versions.
		// We use a manual PATH walk with os.Stat which uses safer stat/fstat syscalls.
		pathEnv := os.Getenv("PATH")
		found := false
		for _, dir := range filepath.SplitList(pathEnv) {
			if dir == "" {
				dir = "."
			}
			path := filepath.Join(dir, name)
			info, err := os.Stat(path)
			// Check if file exists, is not a dir, and is executable (bit 0111)
			if err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
				found = true
				break
			}
		}

		installedCache[name] = found
		return found
	}

	for i := range c.Commands {
		cmd := &c.Commands[i]
		cmdName := strings.ToLower(cmd.Name)
		score := 0.0

		// 1. Exact Name Match (Highest Priority)
		if cmdName == query {
			score = 1.0
		} else if strings.Contains(cmdName, query) {
			score = 0.8
		}

		// 2. Keyword Matching
		// Check how many query tokens match the command's keywords or description
		matches := 0
		cmdText := strings.ToLower(cmd.Keywords + " " + cmd.Description + " " + cmd.Category)
		
		for _, token := range tokens {
			if strings.Contains(cmdName, token) {
				matches++
				continue
			}
			if strings.Contains(cmdText, token) {
				matches++
			}
		}

		// Calculate Jaccard-like similarity for keywords
		if matches > 0 {
			keywordScore := float64(matches) / float64(len(tokens)+2) // Simple normalization
			if keywordScore > score {
				score = keywordScore
			}
		}

		// Threshold
		if score > 0.3 {
			// Check installation status
			isInstalled := checkInstalled(cmd.Name)
			
			// Boost score if installed, as it's more actionable
			if isInstalled {
				score += 0.2
			}
			
			// Cap score
			if score > 1.0 { score = 1.0 }

			results = append(results, SearchResult{
				Command:   cmd,
				Score:     score,
				Installed: isInstalled,
			})
		}
	}
	
	// Sort by score desc
	// Basic bubble sort for small list is faster than sort.Slice overhead
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > 5 {
		results = results[:5]
	}

	return results
}
