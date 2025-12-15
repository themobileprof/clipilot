package intent

import (
	"database/sql"
	"fmt"
	"strings"

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

	// Build scoring map
	scores := make(map[string]float64)
	weights := make(map[string]float64)

	// Query patterns for each token
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
	for moduleID, score := range scores {
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
