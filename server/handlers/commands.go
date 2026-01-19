package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// SemanticSearchRequest represents a natural language query
type SemanticSearchRequest struct {
	Query string `json:"query"`
}

// SemanticSearchResponse returns candidates
type SemanticSearchResponse struct {
	Candidates []CommandCandidate `json:"candidates"`
	Message    string             `json:"message"`
	Cached     bool               `json:"cached"`
}

// CommandCandidate represents a suggested command
type CommandCandidate struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	UseCases    []string `json:"use_cases"`
	Keywords    []string `json:"keywords"` // Keep for compatibility
}

// HandleSemanticSearch proxies natural language queries to Gemini with Caching
// POST /api/commands/search
func HandleSemanticSearch(db *sql.DB, geminiAPIKey string) http.HandlerFunc {
	// Ensure cache table exists on startup (or first request)
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

		if req.Query == "" {
			http.Error(w, "Query is required", http.StatusBadRequest)
			return
		}

		// 1. Check Cache
		queryHash := hashQuery(req.Query)
		if cachedCandidates, err := getCachedResponse(db, queryHash); err == nil {
			response := SemanticSearchResponse{
				Candidates: cachedCandidates,
				Message:    fmt.Sprintf("Found %d candidates (cached)", len(cachedCandidates)),
				Cached:     true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// 2. Call Gemini (if not cached)
		candidates, err := searchWithGemini(geminiAPIKey, req.Query)
		if err != nil {
			log.Printf("Gemini search failed: %v", err)
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}

		// 3. Cache Result
		if len(candidates) > 0 {
			go cacheResponse(db, queryHash, candidates) // Async cache
		}

		response := SemanticSearchResponse{
			Candidates: candidates,
			Message:    fmt.Sprintf("Found %d candidates", len(candidates)),
			Cached:     false,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// searchWithGemini queries Gemini API for commands
func searchWithGemini(apiKey, query string) ([]CommandCandidate, error) {
	// TODO: Replace with actual Gemini API call
	// For now, retaining the mock logic for demonstration/fallback
	// In production, this would make an HTTP request to Google AI Studio
	
	_ = apiKey
	
	queryLower := strings.ToLower(query)
	// Determine if we should delay to simulate network call (removed for speed if mock)
	
	// Mock logic
	if strings.Contains(queryLower, "firewall") || strings.Contains(queryLower, "port") {
		return []CommandCandidate{
			{Name: "ufw", Description: "Uncomplicated Firewall", Category: "security", UseCases: []string{"enable firewall", "allow port 80"}, Keywords: []string{"ufw"}},
			{Name: "iptables", Description: "Administration tool for IPv4 packet filtering", Category: "security", UseCases: []string{"block ip", "nat"}, Keywords: []string{"iptables"}},
		}, nil
	}
	
	// Default mock - simulating "I don't know" or generic
	// In real implementation, this returns actual LLM predictions
	return []CommandCandidate{
		{Name: "echo", Description: "Display a line of text", Category: "general", UseCases: []string{"print variable", "write to file"}, Keywords: []string{"echo"}},
	}, nil
}


// --- Caching Helpers ---

func ensureCacheTable(db *sql.DB) {
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS query_cache (
			query_hash TEXT PRIMARY KEY,
			response_json TEXT,
			created_at INTEGER
		)
	`)
}

func hashQuery(query string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(strings.ToLower(query))))
	return hex.EncodeToString(hash[:])
}

func getCachedResponse(db *sql.DB, hash string) ([]CommandCandidate, error) {
	var jsonStr string
	var createdAt int64
	
	// Cache valid for 24 hours
	expiry := time.Now().Add(-24 * time.Hour).Unix()
	
	err := db.QueryRow("SELECT response_json, created_at FROM query_cache WHERE query_hash = ? AND created_at > ?", hash, expiry).Scan(&jsonStr, &createdAt)
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
	jsonBytes, _ := json.Marshal(candidates)
	_, _ = db.Exec(`
		INSERT OR REPLACE INTO query_cache (query_hash, response_json, created_at)
		VALUES (?, ?, ?)
	`, hash, string(jsonBytes), time.Now().Unix())
}
