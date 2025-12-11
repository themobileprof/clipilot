package models

// Module represents a complete module definition
type Module struct {
	ID          string           `yaml:"id" json:"id"`
	Name        string           `yaml:"name" json:"name"`
	Version     string           `yaml:"version" json:"version"`
	Description string           `yaml:"description" json:"description"`
	Tags        []string         `yaml:"tags" json:"tags"`
	Provides    []string         `yaml:"provides" json:"provides"`
	Requires    []string         `yaml:"requires" json:"requires"`
	SizeKB      int              `yaml:"size_kb" json:"size_kb"`
	Flows       map[string]*Flow `yaml:"flows" json:"flows"`
	Metadata    ModuleMetadata   `yaml:"metadata" json:"metadata"`
}

// ModuleMetadata contains module authorship and licensing info
type ModuleMetadata struct {
	Author  string `yaml:"author" json:"author"`
	License string `yaml:"license" json:"license"`
	URL     string `yaml:"url,omitempty" json:"url,omitempty"`
}

// Flow represents a workflow with steps
type Flow struct {
	Start string           `yaml:"start" json:"start"`
	Steps map[string]*Step `yaml:"steps" json:"steps"`
}

// Step represents a single step in a flow
type Step struct {
	Key       string            `yaml:"-" json:"key"`     // Populated from map key
	Type      string            `yaml:"type" json:"type"` // action, instruction, branch, terminal
	Message   string            `yaml:"message,omitempty" json:"message,omitempty"`
	Command   string            `yaml:"command,omitempty" json:"command,omitempty"`
	RunModule string            `yaml:"run_module,omitempty" json:"run_module,omitempty"`
	BasedOn   string            `yaml:"based_on,omitempty" json:"based_on,omitempty"` // For branch type
	Map       map[string]string `yaml:"map,omitempty" json:"map,omitempty"`           // For branch type
	Next      string            `yaml:"next,omitempty" json:"next,omitempty"`
	Validate  []Validation      `yaml:"validate,omitempty" json:"validate,omitempty"`
	Condition *Condition        `yaml:"condition,omitempty" json:"condition,omitempty"`
}

// Validation represents a step validation rule
type Validation struct {
	CheckCommand string `yaml:"check_command,omitempty" json:"check_command,omitempty"`
	ParseOutput  string `yaml:"parse_output,omitempty" json:"parse_output,omitempty"`
	Expected     string `yaml:"expected,omitempty" json:"expected,omitempty"`
	ErrorMessage string `yaml:"error_message,omitempty" json:"error_message,omitempty"`
}

// Condition represents a conditional execution rule
type Condition struct {
	StateKey string `yaml:"state_key" json:"state_key"`
	Operator string `yaml:"operator" json:"operator"` // eq, ne, gt, lt, contains
	Value    string `yaml:"value" json:"value"`
}

// IntentResult represents the result of intent detection
type IntentResult struct {
	ModuleID   string
	Confidence float64
	Method     string // keyword, llm_local, llm_online, manual
	Candidates []Candidate
}

// Candidate represents a potential module match
type Candidate struct {
	ModuleID    string
	Name        string
	Description string
	Score       float64
	Tags        []string
}

// ExecutionContext holds runtime state during module execution
type ExecutionContext struct {
	SessionID   string
	ModuleID    string
	FlowName    string
	CurrentStep string
	State       map[string]string
	DryRun      bool
	LogID       int64
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Passed  bool
	Message string
	Type    string
}
