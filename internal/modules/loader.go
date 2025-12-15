package modules

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/themobileprof/clipilot/pkg/models"
	"gopkg.in/yaml.v3"
)

// Loader handles module loading and storage
type Loader struct {
	db *sql.DB
}

// NewLoader creates a new module loader
func NewLoader(db *sql.DB) *Loader {
	return &Loader{db: db}
}

// LoadFromFile reads and parses a module YAML file
func (l *Loader) LoadFromFile(path string) (*models.Module, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read module file: %w", err)
	}

	var module models.Module
	if err := yaml.Unmarshal(data, &module); err != nil {
		return nil, fmt.Errorf("failed to parse module YAML: %w", err)
	}

	// Populate step keys from map keys
	for flowName, flow := range module.Flows {
		for stepKey, step := range flow.Steps {
			step.Key = stepKey
		}
		_ = flowName // Unused for now
	}

	return &module, nil
}

// ImportModule imports a module into the database
func (l *Loader) ImportModule(module *models.Module) error {
	tx, err := l.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error as we might commit
	}()

	// Serialize module to JSON for storage
	jsonContent, err := json.Marshal(module)
	if err != nil {
		return fmt.Errorf("failed to marshal module: %w", err)
	}

	// Insert or update module
	_, err = tx.Exec(`
		INSERT INTO modules (id, name, version, description, tags, provides, requires, size_kb, installed, json_content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = ?,
			version = ?,
			description = ?,
			tags = ?,
			provides = ?,
			requires = ?,
			size_kb = ?,
			installed = 1,
			json_content = ?,
			updated_at = strftime('%s', 'now')
	`,
		module.ID, module.Name, module.Version, module.Description,
		strings.Join(module.Tags, ","),
		strings.Join(module.Provides, ","),
		strings.Join(module.Requires, ","),
		module.SizeKB,
		string(jsonContent),
		// For ON CONFLICT UPDATE
		module.Name, module.Version, module.Description,
		strings.Join(module.Tags, ","),
		strings.Join(module.Provides, ","),
		strings.Join(module.Requires, ","),
		module.SizeKB,
		string(jsonContent),
	)
	if err != nil {
		return fmt.Errorf("failed to insert module: %w", err)
	}

	// Clear old steps and patterns
	if _, err := tx.Exec("DELETE FROM steps WHERE module_id = ?", module.ID); err != nil {
		return fmt.Errorf("failed to delete old steps: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM intent_patterns WHERE module_id = ?", module.ID); err != nil {
		return fmt.Errorf("failed to delete old patterns: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM dependencies WHERE module_id = ?", module.ID); err != nil {
		return fmt.Errorf("failed to delete old dependencies: %w", err)
	}

	// Insert steps
	for flowName, flow := range module.Flows {
		order := 0
		for stepKey, step := range flow.Steps {
			extraJSON, _ := json.Marshal(map[string]interface{}{
				"condition": step.Condition,
				"validate":  step.Validate,
				"map":       step.Map,
				"based_on":  step.BasedOn,
			})

			_, err = tx.Exec(`
				INSERT INTO steps (module_id, flow_name, step_key, type, message, command, run_module, order_num, next_step, extra_json)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, module.ID, flowName, stepKey, step.Type, step.Message, step.Command, step.RunModule, order, step.Next, string(extraJSON))
			if err != nil {
				return fmt.Errorf("failed to insert step %s: %w", stepKey, err)
			}
			order++
		}
	}

	// Generate and insert intent patterns
	patterns := generatePatterns(module)
	for _, pattern := range patterns {
		_, err = tx.Exec(`
			INSERT INTO intent_patterns (module_id, pattern, weight, pattern_type)
			VALUES (?, ?, ?, ?)
		`, module.ID, pattern.Pattern, pattern.Weight, pattern.Type)
		if err != nil {
			return fmt.Errorf("failed to insert pattern: %w", err)
		}
	}

	// Insert dependencies
	for _, req := range module.Requires {
		_, err = tx.Exec(`
			INSERT INTO dependencies (module_id, requires_module_id)
			VALUES (?, ?)
		`, module.ID, req)
		if err != nil {
			return fmt.Errorf("failed to insert dependency: %w", err)
		}
	}

	return tx.Commit()
}

// GetModule retrieves a module by ID
func (l *Loader) GetModule(moduleID string) (*models.Module, error) {
	var jsonContent string
	err := l.db.QueryRow("SELECT json_content FROM modules WHERE id = ? AND installed = 1", moduleID).Scan(&jsonContent)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("module not found: %s", moduleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query module: %w", err)
	}

	var module models.Module
	if err := json.Unmarshal([]byte(jsonContent), &module); err != nil {
		return nil, fmt.Errorf("failed to unmarshal module: %w", err)
	}

	return &module, nil
}

// ListModules returns all installed modules
func (l *Loader) ListModules() ([]models.Module, error) {
	rows, err := l.db.Query(`
		SELECT id, name, version, description, tags, provides, requires, size_kb
		FROM modules WHERE installed = 1
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query modules: %w", err)
	}
	defer rows.Close()

	var modules []models.Module
	for rows.Next() {
		var m models.Module
		var tags, provides, requires string
		err := rows.Scan(&m.ID, &m.Name, &m.Version, &m.Description, &tags, &provides, &requires, &m.SizeKB)
		if err != nil {
			return nil, fmt.Errorf("failed to scan module: %w", err)
		}
		if tags != "" {
			m.Tags = strings.Split(tags, ",")
		}
		if provides != "" {
			m.Provides = strings.Split(provides, ",")
		}
		if requires != "" {
			m.Requires = strings.Split(requires, ",")
		}
		modules = append(modules, m)
	}

	return modules, nil
}

// Pattern represents an intent pattern
type Pattern struct {
	Pattern string
	Weight  float64
	Type    string
}

// generatePatterns creates intent patterns from module metadata
func generatePatterns(module *models.Module) []Pattern {
	patterns := []Pattern{}

	// Add name tokens
	nameTokens := tokenize(module.Name)
	for _, token := range nameTokens {
		patterns = append(patterns, Pattern{
			Pattern: token,
			Weight:  1.5,
			Type:    "keyword",
		})
	}

	// Add description tokens
	descTokens := tokenize(module.Description)
	for _, token := range descTokens {
		patterns = append(patterns, Pattern{
			Pattern: token,
			Weight:  1.0,
			Type:    "keyword",
		})
	}

	// Add tags with higher weight
	for _, tag := range module.Tags {
		patterns = append(patterns, Pattern{
			Pattern: strings.ToLower(tag),
			Weight:  2.0,
			Type:    "tag",
		})
	}

	// Add module ID components
	idParts := strings.Split(module.ID, ".")
	for _, part := range idParts {
		if len(part) > 2 {
			patterns = append(patterns, Pattern{
				Pattern: strings.ToLower(part),
				Weight:  1.0,
				Type:    "keyword",
			})
		}
	}

	return patterns
}

// tokenize breaks text into searchable tokens
func tokenize(text string) []string {
	text = strings.ToLower(text)
	// Replace common separators with spaces
	text = strings.ReplaceAll(text, "_", " ")
	text = strings.ReplaceAll(text, "-", " ")

	tokens := strings.Fields(text)

	// Filter short tokens
	filtered := []string{}
	for _, token := range tokens {
		if len(token) >= 3 {
			filtered = append(filtered, token)
		}
	}

	return filtered
}
