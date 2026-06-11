package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/themobileprof/clipilot/server/catalog"
)

// SemanticSearchRequest is the Clio client search body.
type SemanticSearchRequest struct {
	Query string `json:"query"`
	OS    string `json:"os,omitempty"`
	Arch  string `json:"arch,omitempty"`
}

// SemanticSearchResponse is returned to Clio (and legacy clients).
type SemanticSearchResponse struct {
	Candidates []CommandCandidate `json:"candidates"`
	Results    []CommandCandidate `json:"results,omitempty"` // legacy alias for Clio
	Message    string             `json:"message"`
	Cached     bool               `json:"cached"`
	Source     string             `json:"source,omitempty"` // catalog | gemini
}

// CommandCandidate represents a suggested command.
type CommandCandidate struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	UseCases    []string `json:"use_cases"`
	Keywords    []string `json:"keywords"`
	Usage       string   `json:"usage,omitempty"`
}

// HandleSemanticSearch serves POST /api/commands/search for the Clio client.
func HandleSemanticSearch(db *sql.DB, geminiAPIKey string) http.HandlerFunc {
	ensureCacheTable(db)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SemanticSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Query) == "" {
			http.Error(w, "Query is required", http.StatusBadRequest)
			return
		}

		cacheKey := hashQuery(req.Query, req.OS, req.Arch)
		if cached, err := getCachedResponse(db, cacheKey); err == nil {
			writeSearchResponse(w, cached, true, "")
			return
		}

		candidates, source := searchCommands(req.Query, req.OS, geminiAPIKey)
		if len(candidates) == 0 {
			http.Error(w, "No matching commands found", http.StatusNotFound)
			return
		}

		go cacheResponse(db, cacheKey, candidates)

		writeSearchResponse(w, candidates, false, source)
	}
}

func searchCommands(query, os, geminiAPIKey string) ([]CommandCandidate, string) {
	hits := catalog.Search(query)
	if len(hits) > 0 && hits[0].Score >= 4.0 {
		return catalogHitsToCandidates(hits, os), "catalog"
	}

	if geminiAPIKey != "" {
		if candidates, err := searchWithGemini(geminiAPIKey, query, os, hits); err == nil && len(candidates) > 0 {
			return candidates, "gemini"
		}
		log.Printf("Gemini search failed, using catalog fallback")
	}

	if len(hits) > 0 {
		return catalogHitsToCandidates(hits, os), "catalog"
	}
	return nil, ""
}

func catalogHitsToCandidates(hits []catalog.SearchResult, os string) []CommandCandidate {
	out := make([]CommandCandidate, 0, len(hits))
	for _, hit := range hits {
		usage := catalog.UseCase(hit.Entry, os)
		kw := strings.Split(hit.Entry.Keywords, ",")
		for i := range kw {
			kw[i] = strings.TrimSpace(kw[i])
		}
		out = append(out, CommandCandidate{
			Name:        hit.Entry.Name,
			Description: hit.Entry.Description,
			Category:    hit.Entry.Category,
			UseCases:    []string{usage},
			Keywords:    kw,
			Usage:       usage,
		})
	}
	return out
}

func writeSearchResponse(w http.ResponseWriter, candidates []CommandCandidate, cached bool, source string) {
	msg := fmt.Sprintf("Found %d candidates", len(candidates))
	if cached {
		msg += " (cached)"
	}
	resp := SemanticSearchResponse{
		Candidates: candidates,
		Results:    candidates,
		Message:    msg,
		Cached:     cached,
		Source:     source,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// searchWithGemini calls Gemini Flash when catalog confidence is low.
func searchWithGemini(apiKey, query, os string, hints []catalog.SearchResult) ([]CommandCandidate, error) {
	var hintLines strings.Builder
	for i, h := range hints {
		if i >= 3 {
			break
		}
		hintLines.WriteString(fmt.Sprintf("- %s: %s\n", h.Entry.Name, h.Entry.Description))
	}

	prompt := fmt.Sprintf(`You help Nigerian students on Termux (Android, arm64) find Linux shell commands.
Reply with ONLY a JSON array (max 3 items). Each item: {"name":"command","description":"brief","usage":"example"}.
No markdown. Prefer standard tools (ls, cp, df, free, pkg, git). Never suggest "echo" unless user wants to print text.

User query: %q
OS: %s
Catalog hints:
%s`, query, os, hintLines.String())

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"maxOutputTokens": 512,
		},
	}
	jsonBody, _ := json.Marshal(body)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<10))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(raw))
	}

	return parseGeminiCandidates(raw)
}

func parseGeminiCandidates(raw []byte) ([]CommandCandidate, error) {
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &geminiResp); err != nil {
		return nil, err
	}
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty gemini response")
	}

	text := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var parsed []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Usage       string `json:"usage"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, err
	}

	out := make([]CommandCandidate, 0, len(parsed))
	for _, p := range parsed {
		name := strings.Fields(strings.TrimSpace(p.Name))[0]
		if name == "" || name == "echo" {
			continue
		}
		usage := p.Usage
		if usage == "" {
			usage = name
		}
		out = append(out, CommandCandidate{
			Name:        name,
			Description: p.Description,
			Category:    "remote",
			UseCases:    []string{usage},
			Usage:       usage,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid commands in gemini response")
	}
	return out, nil
}

// --- Caching ---

func ensureCacheTable(db *sql.DB) {
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS query_cache (
			query_hash TEXT PRIMARY KEY,
			response_json TEXT,
			created_at INTEGER
		)
	`)
}

func hashQuery(query, os, arch string) string {
	normalized := strings.TrimSpace(strings.ToLower(query)) + "|" + os + "|" + arch
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func getCachedResponse(db *sql.DB, hash string) ([]CommandCandidate, error) {
	var jsonStr string
	var createdAt int64
	expiry := time.Now().Add(-7 * 24 * time.Hour).Unix()

	err := db.QueryRow(
		`SELECT response_json, created_at FROM query_cache WHERE query_hash = ? AND created_at > ?`,
		hash, expiry,
	).Scan(&jsonStr, &createdAt)
	if err != nil {
		return nil, err
	}

	var candidates []CommandCandidate
	if err := json.Unmarshal([]byte(jsonStr), &candidates); err != nil {
		return nil, err
	}
	return candidates, nil
}

func cacheResponse(db *sql.DB, hash string, candidates []CommandCandidate) {
	jsonBytes, err := json.Marshal(candidates)
	if err != nil {
		return
	}
	_, _ = db.Exec(`
		INSERT OR REPLACE INTO query_cache (query_hash, response_json, created_at)
		VALUES (?, ?, ?)
	`, hash, string(jsonBytes), time.Now().Unix())
}
