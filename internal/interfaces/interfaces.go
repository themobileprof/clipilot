package interfaces

import (
	"database/sql"

	"github.com/themobileprof/clipilot/pkg/models"
)

// IntentClassifier detects user intent from natural language input
type IntentClassifier interface {
	// Detect analyzes input and returns the most likely module and candidates
	Detect(input string) (*models.IntentResult, error)
	// SetThresholds configures confidence thresholds for detection layers
	SetThresholds(keyword, llm float64)
	// SetOnlineEnabled enables or disables online LLM fallback
	SetOnlineEnabled(enabled bool)
}

// ModuleStore handles module loading, storage, and retrieval
type ModuleStore interface {
	// LoadFromFile reads and parses a module YAML file
	LoadFromFile(path string) (*models.Module, error)
	// ImportModule saves a module to the database
	ImportModule(module *models.Module) error
	// GetModule retrieves a module by ID
	GetModule(moduleID string) (*models.Module, error)
	// ListModules returns all installed modules
	ListModules() ([]models.Module, error)
}

// FlowRunner executes module flows with their steps
type FlowRunner interface {
	// Run executes a module flow by ID
	Run(moduleID string) error
	// SetDryRun enables or disables dry-run mode (shows commands without executing)
	SetDryRun(enabled bool)
	// SetAutoYes enables or disables automatic confirmation prompts
	SetAutoYes(enabled bool)
}

// Executor executes individual steps within a flow
type Executor interface {
	// ExecuteCommand runs a shell command and returns output
	ExecuteCommand(command string) (string, error)
	// ValidateCommand checks if a command is safe to execute
	ValidateCommand(command string) error
}

// StateStore manages execution state and session data
type StateStore interface {
	// GetState retrieves a state value for a session
	GetState(sessionID, key string) (string, error)
	// SetState stores a state value for a session
	SetState(sessionID, key, value string) error
	// DeleteState removes state data for a session
	DeleteState(sessionID string) error
	// GetAllState retrieves all state for a session
	GetAllState(sessionID string) (map[string]string, error)
}

// PlatformDetector identifies the operating system and environment
type PlatformDetector interface {
	// DetectOS returns the operating system name
	DetectOS() (string, error)
	// DetectDistro returns the Linux distribution name (if applicable)
	DetectDistro() (string, error)
	// IsTermux checks if running in Termux environment
	IsTermux() bool
	// GetPackageManager returns the system package manager name
	GetPackageManager() (string, error)
}

// LLMClient provides access to language model inference
type LLMClient interface {
	// Classify performs intent classification using LLM
	Classify(input string, candidates []models.Candidate) (*models.IntentResult, error)
	// IsAvailable checks if the LLM service is accessible
	IsAvailable() bool
}

// Logger handles execution logging and history
type Logger interface {
	// LogStart records the beginning of an execution
	LogStart(sessionID, input, resolvedModule string, confidence float64, method string) (int64, error)
	// LogComplete records the completion of an execution
	LogComplete(logID int64, status, errorMsg string, durationMs int64) error
	// GetLogs retrieves recent execution logs
	GetLogs(limit int) ([]models.LogEntry, error)
}

// SettingsManager handles configuration and settings persistence
type SettingsManager interface {
	// GetSetting retrieves a configuration value
	GetSetting(key string) (string, error)
	// SetSetting stores a configuration value
	SetSetting(key, value, description string) error
	// DeleteSetting removes a configuration value
	DeleteSetting(key string) error
	// ListSettings returns all configuration values
	ListSettings() (map[string]string, error)
}

// DatabaseConnection provides low-level database access
type DatabaseConnection interface {
	// Conn returns the underlying sql.DB connection
	Conn() *sql.DB
	// Close closes the database connection
	Close() error
	// Migrate runs database migrations
	Migrate() error
}
