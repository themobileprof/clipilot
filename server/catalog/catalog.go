package catalog

import (
	_ "embed"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed common_commands.yaml
var embeddedYAML []byte

// CommandEntry is one row from the common commands catalog.
type CommandEntry struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	Category      string `yaml:"category"`
	Keywords      string `yaml:"keywords"`
	PkgPackage    string `yaml:"pkg_package"`
	Priority      int    `yaml:"priority"`
	AlternativeTo string `yaml:"alternative_to"`
}

// SearchResult is a scored catalog hit for API responses.
type SearchResult struct {
	Entry CommandEntry
	Score float64
}

var (
	entries     []CommandEntry
	entriesOnce sync.Once
)

// essentials fill gaps in the YAML for Termux / student queries.
var essentials = []CommandEntry{
	{Name: "ping", Description: "Test network connectivity", Category: "networking", Keywords: "network, internet, data, wifi, connect, working", Priority: 95},
	{Name: "pwd", Description: "Print current working directory", Category: "file-management", Keywords: "where, directory, folder, path, location", Priority: 90},
	{Name: "ps", Description: "List running processes", Category: "system", Keywords: "process, running, apps, programs, tasks", Priority: 90},
	{Name: "kill", Description: "Stop a running process", Category: "system", Keywords: "stop, stuck, jam, hang, frozen, process", Priority: 88},
	{Name: "pkg", Description: "Termux package manager", Category: "system", Keywords: "install, update, upgrade, package, termux", PkgPackage: "pkg", Priority: 92},
}

func loadEntries() []CommandEntry {
	entriesOnce.Do(func() {
		_ = yaml.Unmarshal(embeddedYAML, &entries)
		entries = append(entries, essentials...)
	})
	return entries
}

// Search finds commands matching a natural-language query.
func Search(query string) []SearchResult {
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return nil
	}

	var results []SearchResult
	for _, entry := range loadEntries() {
		score := scoreEntry(entry, tokens, query)
		if score >= minScore {
			results = append(results, SearchResult{Entry: entry, Score: score})
		}
	}

	sortResults(results)
	if len(results) > 5 {
		results = results[:5]
	}
	return results
}

const minScore = 2.5

func scoreEntry(entry CommandEntry, tokens []string, rawQuery string) float64 {
	q := strings.ToLower(rawQuery)
	name := strings.ToLower(entry.Name)
	desc := strings.ToLower(entry.Description)
	kw := strings.ToLower(entry.Keywords)
	cat := strings.ToLower(entry.Category)

	var score float64

	// Full query contains command name
	if strings.Contains(q, name) {
		score += 4
	}

	for _, tok := range tokens {
		if tok == name {
			score += 6
		}
		if strings.Contains(name, tok) && len(tok) >= 3 {
			score += 2
		}
		if strings.Contains(kw, tok) {
			score += 3
		}
		if strings.Contains(desc, tok) {
			score += 1.5
		}
		if strings.Contains(cat, tok) {
			score += 1
		}
	}

	// Boost high-priority common tools slightly
	score += float64(entry.Priority) / 100.0

	// Connectivity checks: "data no dey work", "internet not working"
	if hasToken(tokens, "network") && (hasToken(tokens, "work") || strings.Contains(q, "working")) {
		if name == "ping" || name == "curl" {
			score += 5
		}
		if name == "nmap" {
			score -= 3
		}
	}

	return score
}

func hasToken(tokens []string, want string) bool {
	for _, t := range tokens {
		if t == want {
			return true
		}
	}
	return false
}

func sortResults(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}

// UseCase returns a practical usage hint for the client.
func UseCase(entry CommandEntry, os string) string {
	switch entry.Name {
	case "cp":
		return "cp source dest"
	case "mv":
		return "mv old new"
	case "rm":
		return "rm file"
	case "ls":
		return "ls -la"
	case "df":
		return "df -h"
	case "free":
		return "free -h"
	case "grep":
		return "grep -r pattern ."
	case "find":
		return "find . -name 'pattern'"
	case "tar":
		return "tar -xzvf archive.tar.gz"
	case "unzip":
		return "unzip file.zip"
	case "chmod":
		return "chmod +x script.sh"
	case "git":
		return "git clone <url>"
	default:
		if entry.PkgPackage != "" && (os == "linux" || strings.Contains(os, "android")) {
			return "pkg install " + entry.PkgPackage + "  # Termux"
		}
		return entry.Name + " --help"
	}
}
