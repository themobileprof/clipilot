package intent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/themobileprof/clipilot/internal/ml"
	"github.com/themobileprof/clipilot/pkg/models"
)

// SemanticClassifier provides semantic intent classification using embeddings
type SemanticClassifier struct {
	engine        *ml.EmbeddingEngine
	cache         *ml.EmbeddingCache
	db            *sql.DB
	moduleEmbeds  map[string][]float32 // moduleID -> embedding
	commandEmbeds map[string][]float32 // command name -> embedding
	threshold     float64
	loaded        bool
	mu            sync.RWMutex
}

// NewSemanticClassifier creates a new semantic classifier
func NewSemanticClassifier(db *sql.DB) *SemanticClassifier {
	homeDir, _ := os.UserHomeDir()
	modelDir := filepath.Join(homeDir, ".clipilot", "models")

	return &SemanticClassifier{
		engine:        ml.NewEmbeddingEngine(modelDir),
		cache:         ml.NewEmbeddingCache(),
		db:            db,
		moduleEmbeds:  make(map[string][]float32),
		commandEmbeds: make(map[string][]float32),
		threshold:     0.5, // Minimum similarity threshold
		loaded:        false,
	}
}

// Load initializes the semantic classifier and pre-computes embeddings
func (sc *SemanticClassifier) Load() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.loaded {
		return nil
	}

	// Load the embedding engine
	if err := sc.engine.Load(); err != nil {
		return fmt.Errorf("failed to load embedding engine: %w", err)
	}

	// Try to load cached embeddings first
	if err := sc.loadCachedEmbeddings(); err != nil {
		// Cache miss - compute embeddings
		if err := sc.computeModuleEmbeddings(); err != nil {
			return fmt.Errorf("failed to compute module embeddings: %w", err)
		}
		if err := sc.computeCommandEmbeddings(); err != nil {
			// Non-fatal - commands are optional
			fmt.Printf("Warning: failed to compute command embeddings: %v\n", err)
		}
		// Save to cache for next time
		if err := sc.saveCachedEmbeddings(); err != nil {
			fmt.Printf("Warning: failed to cache embeddings: %v\n", err)
		}
	}

	sc.loaded = true
	return nil
}

// Classify performs semantic intent classification for both modules and commands
func (sc *SemanticClassifier) Classify(input string) (*models.IntentResult, error) {
	sc.mu.RLock()
	if !sc.loaded {
		sc.mu.RUnlock()
		return nil, fmt.Errorf("semantic classifier not loaded")
	}
	sc.mu.RUnlock()

	// Get embedding for input
	inputEmbed, err := sc.engine.Embed(input)
	if err != nil {
		return nil, fmt.Errorf("failed to embed input: %w", err)
	}

	// Find similar modules
	candidates := sc.findSimilarModules(inputEmbed)

	// Find similar commands
	cmdCandidates := sc.findSimilarCommands(inputEmbed)
	candidates = append(candidates, cmdCandidates...)

	// Sort by score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Build result
	result := &models.IntentResult{
		Candidates: candidates,
		Method:     "semantic",
	}

	if len(candidates) > 0 && candidates[0].Score >= sc.threshold {
		result.ModuleID = candidates[0].ModuleID
		result.Confidence = float64(candidates[0].Score)
	}

	return result, nil
}

// ClassifyCommands performs semantic search for COMMANDS only (Layer 1)
// This is the primary search method - modules are accessed directly via 'run <module_id>'
func (sc *SemanticClassifier) ClassifyCommands(input string) (*models.IntentResult, error) {
	sc.mu.RLock()
	if !sc.loaded {
		sc.mu.RUnlock()
		return nil, fmt.Errorf("semantic classifier not loaded")
	}
	sc.mu.RUnlock()

	// Get embedding for input
	inputEmbed, err := sc.engine.Embed(input)
	if err != nil {
		return nil, fmt.Errorf("failed to embed input: %w", err)
	}

	// Find similar commands ONLY (not modules)
	candidates := sc.findSimilarCommands(inputEmbed)

	// Sort by score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Build result
	result := &models.IntentResult{
		Candidates: candidates,
		Method:     "semantic",
	}

	if len(candidates) > 0 && candidates[0].Score >= sc.threshold {
		result.ModuleID = candidates[0].ModuleID
		result.Confidence = float64(candidates[0].Score)
	}

	return result, nil
}

// findSimilarModules returns modules similar to the query embedding
func (sc *SemanticClassifier) findSimilarModules(queryEmbed []float32) []models.Candidate {
	var candidates []models.Candidate

	for moduleID, embed := range sc.moduleEmbeds {
		similarity := ml.CosineSimilarity(queryEmbed, embed)

		if similarity >= float32(sc.threshold) {
			// Get module details from DB
			var name, description, tags string
			err := sc.db.QueryRow(`
				SELECT name, description, COALESCE(tags, '')
				FROM modules WHERE id = ? AND installed = 1
			`, moduleID).Scan(&name, &description, &tags)

			if err != nil {
				continue
			}

			tagList := []string{}
			if tags != "" {
				tagList = strings.Split(tags, ",")
			}

			candidates = append(candidates, models.Candidate{
				ModuleID:    moduleID,
				Name:        name,
				Description: description,
				Score:       float64(similarity),
				Tags:        tagList,
			})
		}
	}

	return candidates
}

// findSimilarCommands returns commands similar to the query embedding
func (sc *SemanticClassifier) findSimilarCommands(queryEmbed []float32) []models.Candidate {
	var candidates []models.Candidate

	for cmdName, embed := range sc.commandEmbeds {
		similarity := ml.CosineSimilarity(queryEmbed, embed)

		if similarity >= float32(sc.threshold) {
			// Get command details from DB
			var description string
			err := sc.db.QueryRow(`
				SELECT COALESCE(description, '') FROM commands WHERE name = ?
			`, cmdName).Scan(&description)

			if err != nil {
				continue
			}

			candidates = append(candidates, models.Candidate{
				ModuleID:    "cmd:" + cmdName,
				Name:        cmdName,
				Description: description,
				Score:       float64(similarity),
				Tags:        []string{"command"},
			})
		}
	}

	return candidates
}

// computeModuleEmbeddings generates embeddings for all installed modules
func (sc *SemanticClassifier) computeModuleEmbeddings() error {
	rows, err := sc.db.Query(`
		SELECT id, name, description, COALESCE(tags, '')
		FROM modules WHERE installed = 1
	`)
	if err != nil {
		return fmt.Errorf("failed to query modules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, description, tags string
		if err := rows.Scan(&id, &name, &description, &tags); err != nil {
			continue
		}

		// Create rich text representation for embedding
		// Combine name, description, and tags for better semantic matching
		text := fmt.Sprintf("%s: %s. Tags: %s", name, description, tags)

		embed, err := sc.engine.Embed(text)
		if err != nil {
			fmt.Printf("Warning: failed to embed module %s: %v\n", id, err)
			continue
		}

		sc.moduleEmbeds[id] = embed
	}

	return nil
}

// computeCommandEmbeddings generates embeddings for system commands
func (sc *SemanticClassifier) computeCommandEmbeddings() error {
	// Only embed commands with descriptions (quality filter)
	rows, err := sc.db.Query(`
		SELECT name, description
		FROM commands
		WHERE description IS NOT NULL AND description != ''
		LIMIT 500
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

		// Combine name and description for embedding
		text := fmt.Sprintf("%s: %s", name, description)

		embed, err := sc.engine.Embed(text)
		if err != nil {
			continue // Skip commands that fail to embed
		}

		sc.commandEmbeds[name] = embed
	}

	return nil
}

// loadCachedEmbeddings loads pre-computed embeddings from database
func (sc *SemanticClassifier) loadCachedEmbeddings() error {
	// Load module embeddings
	rows, err := sc.db.Query(`
		SELECT module_id, embedding FROM module_embeddings
	`)
	if err != nil {
		return err // Table might not exist yet
	}
	defer rows.Close()

	for rows.Next() {
		var moduleID string
		var embeddingJSON []byte
		if err := rows.Scan(&moduleID, &embeddingJSON); err != nil {
			continue
		}

		var embedding []float32
		if err := json.Unmarshal(embeddingJSON, &embedding); err != nil {
			continue
		}

		sc.moduleEmbeds[moduleID] = embedding
	}

	// Load command embeddings
	rows2, err := sc.db.Query(`
		SELECT command_name, embedding FROM command_embeddings
	`)
	if err != nil {
		return nil // Table might not exist, that's okay
	}
	defer rows2.Close()

	for rows2.Next() {
		var cmdName string
		var embeddingJSON []byte
		if err := rows2.Scan(&cmdName, &embeddingJSON); err != nil {
			continue
		}

		var embedding []float32
		if err := json.Unmarshal(embeddingJSON, &embedding); err != nil {
			continue
		}

		sc.commandEmbeds[cmdName] = embedding
	}

	// Return success only if we loaded some embeddings
	if len(sc.moduleEmbeds) > 0 || len(sc.commandEmbeds) > 0 {
		return nil
	}
	return fmt.Errorf("no cached embeddings found")
}

// saveCachedEmbeddings saves computed embeddings to database
func (sc *SemanticClassifier) saveCachedEmbeddings() error {
	tx, err := sc.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Save module embeddings
	for moduleID, embedding := range sc.moduleEmbeds {
		embJSON, err := json.Marshal(embedding)
		if err != nil {
			continue
		}

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO module_embeddings (module_id, embedding, updated_at)
			VALUES (?, ?, strftime('%s', 'now'))
		`, moduleID, embJSON)
		if err != nil {
			// Table might not exist - create it
			_, _ = tx.Exec(`
				CREATE TABLE IF NOT EXISTS module_embeddings (
					module_id TEXT PRIMARY KEY,
					embedding BLOB,
					updated_at INTEGER
				)
			`)
			_, _ = tx.Exec(`
				INSERT OR REPLACE INTO module_embeddings (module_id, embedding, updated_at)
				VALUES (?, ?, strftime('%s', 'now'))
			`, moduleID, embJSON)
		}
	}

	// Save command embeddings
	for cmdName, embedding := range sc.commandEmbeds {
		embJSON, err := json.Marshal(embedding)
		if err != nil {
			continue
		}

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO command_embeddings (command_name, embedding, updated_at)
			VALUES (?, ?, strftime('%s', 'now'))
		`, cmdName, embJSON)
		if err != nil {
			// Table might not exist - create it
			_, _ = tx.Exec(`
				CREATE TABLE IF NOT EXISTS command_embeddings (
					command_name TEXT PRIMARY KEY,
					embedding BLOB,
					updated_at INTEGER
				)
			`)
			_, _ = tx.Exec(`
				INSERT OR REPLACE INTO command_embeddings (command_name, embedding, updated_at)
				VALUES (?, ?, strftime('%s', 'now'))
			`, cmdName, embJSON)
		}
	}

	return tx.Commit()
}

// SetThreshold sets the minimum similarity threshold
func (sc *SemanticClassifier) SetThreshold(threshold float64) {
	sc.threshold = threshold
}

// IsLoaded returns whether the classifier is loaded
func (sc *SemanticClassifier) IsLoaded() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.loaded
}

// Close releases resources
func (sc *SemanticClassifier) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.engine != nil {
		if err := sc.engine.Close(); err != nil {
			return err
		}
	}
	sc.loaded = false
	return nil
}

// RefreshEmbeddings recomputes embeddings for new/updated modules
func (sc *SemanticClassifier) RefreshEmbeddings() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if err := sc.computeModuleEmbeddings(); err != nil {
		return fmt.Errorf("failed to refresh module embeddings: %w", err)
	}

	if err := sc.computeCommandEmbeddings(); err != nil {
		fmt.Printf("Warning: failed to refresh command embeddings: %v\n", err)
	}

	return sc.saveCachedEmbeddings()
}

// GetStats returns statistics about the classifier
func (sc *SemanticClassifier) GetStats() map[string]int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return map[string]int{
		"modules":  len(sc.moduleEmbeds),
		"commands": len(sc.commandEmbeds),
	}
}

// Legacy TinyLLM for backward compatibility
// Deprecated: Use SemanticClassifier instead

// TinyLLM provides local LLM-based intent classification
// Deprecated: This is a placeholder - use SemanticClassifier instead
type TinyLLM struct {
	modelPath string
	loaded    bool
}

// NewTinyLLM creates a new tiny LLM classifier
// Deprecated: Use NewSemanticClassifier instead
func NewTinyLLM(modelPath string) *TinyLLM {
	return &TinyLLM{
		modelPath: modelPath,
		loaded:    false,
	}
}

// Load loads the LLM model into memory
// Deprecated: Use SemanticClassifier.Load instead
func (llm *TinyLLM) Load() error {
	return nil
}

// Classify runs classification on input text
// Deprecated: Use SemanticClassifier.Classify instead
func (llm *TinyLLM) Classify(input string, candidates []string) (label string, confidence float64, err error) {
	return "", 0.0, fmt.Errorf("deprecated: use SemanticClassifier instead")
}

// Unload releases model from memory
func (llm *TinyLLM) Unload() error {
	llm.loaded = false
	return nil
}

// IsLoaded returns whether the model is loaded
func (llm *TinyLLM) IsLoaded() bool {
	return llm.loaded
}
