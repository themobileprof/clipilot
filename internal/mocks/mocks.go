package mocks

import (
	"database/sql"
	"fmt"

	"github.com/themobileprof/clipilot/internal/interfaces"
	"github.com/themobileprof/clipilot/pkg/models"
)

// MockModuleStore is a mock implementation of ModuleStore for testing
type MockModuleStore struct {
	LoadFromFileFunc func(path string) (*models.Module, error)
	ImportModuleFunc func(module *models.Module) error
	GetModuleFunc    func(moduleID string) (*models.Module, error)
	ListModulesFunc  func() ([]models.Module, error)
	modules          map[string]*models.Module
}

// NewMockModuleStore creates a new mock module store
func NewMockModuleStore() *MockModuleStore {
	return &MockModuleStore{
		modules: make(map[string]*models.Module),
	}
}

func (m *MockModuleStore) LoadFromFile(path string) (*models.Module, error) {
	if m.LoadFromFileFunc != nil {
		return m.LoadFromFileFunc(path)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockModuleStore) ImportModule(module *models.Module) error {
	if m.ImportModuleFunc != nil {
		return m.ImportModuleFunc(module)
	}
	m.modules[module.ID] = module
	return nil
}

func (m *MockModuleStore) GetModule(moduleID string) (*models.Module, error) {
	if m.GetModuleFunc != nil {
		return m.GetModuleFunc(moduleID)
	}
	if mod, ok := m.modules[moduleID]; ok {
		return mod, nil
	}
	return nil, fmt.Errorf("module not found: %s", moduleID)
}

func (m *MockModuleStore) ListModules() ([]models.Module, error) {
	if m.ListModulesFunc != nil {
		return m.ListModulesFunc()
	}
	var result []models.Module
	for _, mod := range m.modules {
		result = append(result, *mod)
	}
	return result, nil
}

// Ensure MockModuleStore implements ModuleStore interface
var _ interfaces.ModuleStore = (*MockModuleStore)(nil)

// MockIntentClassifier is a mock implementation of IntentClassifier for testing
type MockIntentClassifier struct {
	DetectFunc           func(input string) (*models.IntentResult, error)
	SetThresholdsFunc    func(keyword, llm float64)
	SetOnlineEnabledFunc func(enabled bool)
}

func (m *MockIntentClassifier) Detect(input string) (*models.IntentResult, error) {
	if m.DetectFunc != nil {
		return m.DetectFunc(input)
	}
	return &models.IntentResult{
		ModuleID:   "test.module",
		Confidence: 0.8,
		Method:     "mock",
		Candidates: []models.Candidate{},
	}, nil
}

func (m *MockIntentClassifier) SetThresholds(keyword, llm float64) {
	if m.SetThresholdsFunc != nil {
		m.SetThresholdsFunc(keyword, llm)
	}
}

func (m *MockIntentClassifier) SetOnlineEnabled(enabled bool) {
	if m.SetOnlineEnabledFunc != nil {
		m.SetOnlineEnabledFunc(enabled)
	}
}

// Ensure MockIntentClassifier implements IntentClassifier interface
var _ interfaces.IntentClassifier = (*MockIntentClassifier)(nil)

// MockFlowRunner is a mock implementation of FlowRunner for testing
type MockFlowRunner struct {
	RunFunc        func(moduleID string) error
	SetDryRunFunc  func(enabled bool)
	SetAutoYesFunc func(enabled bool)
	runCalled      bool
	lastModuleID   string
}

func (m *MockFlowRunner) Run(moduleID string) error {
	m.runCalled = true
	m.lastModuleID = moduleID
	if m.RunFunc != nil {
		return m.RunFunc(moduleID)
	}
	return nil
}

func (m *MockFlowRunner) SetDryRun(enabled bool) {
	if m.SetDryRunFunc != nil {
		m.SetDryRunFunc(enabled)
	}
}

func (m *MockFlowRunner) SetAutoYes(enabled bool) {
	if m.SetAutoYesFunc != nil {
		m.SetAutoYesFunc(enabled)
	}
}

// Ensure MockFlowRunner implements FlowRunner interface
var _ interfaces.FlowRunner = (*MockFlowRunner)(nil)

// MockExecutor is a mock implementation of Executor for testing
type MockExecutor struct {
	ExecuteCommandFunc  func(command string) (string, error)
	ValidateCommandFunc func(command string) error
}

func (m *MockExecutor) ExecuteCommand(command string) (string, error) {
	if m.ExecuteCommandFunc != nil {
		return m.ExecuteCommandFunc(command)
	}
	return "mock output", nil
}

func (m *MockExecutor) ValidateCommand(command string) error {
	if m.ValidateCommandFunc != nil {
		return m.ValidateCommandFunc(command)
	}
	return nil
}

// Ensure MockExecutor implements Executor interface
var _ interfaces.Executor = (*MockExecutor)(nil)

// MockStateStore is a mock implementation of StateStore for testing
type MockStateStore struct {
	GetStateFunc    func(sessionID, key string) (string, error)
	SetStateFunc    func(sessionID, key, value string) error
	DeleteStateFunc func(sessionID string) error
	GetAllStateFunc func(sessionID string) (map[string]string, error)
	state           map[string]map[string]string // sessionID -> key -> value
}

func NewMockStateStore() *MockStateStore {
	return &MockStateStore{
		state: make(map[string]map[string]string),
	}
}

func (m *MockStateStore) GetState(sessionID, key string) (string, error) {
	if m.GetStateFunc != nil {
		return m.GetStateFunc(sessionID, key)
	}
	if session, ok := m.state[sessionID]; ok {
		if val, ok := session[key]; ok {
			return val, nil
		}
	}
	return "", fmt.Errorf("state not found")
}

func (m *MockStateStore) SetState(sessionID, key, value string) error {
	if m.SetStateFunc != nil {
		return m.SetStateFunc(sessionID, key, value)
	}
	if m.state[sessionID] == nil {
		m.state[sessionID] = make(map[string]string)
	}
	m.state[sessionID][key] = value
	return nil
}

func (m *MockStateStore) DeleteState(sessionID string) error {
	if m.DeleteStateFunc != nil {
		return m.DeleteStateFunc(sessionID)
	}
	delete(m.state, sessionID)
	return nil
}

func (m *MockStateStore) GetAllState(sessionID string) (map[string]string, error) {
	if m.GetAllStateFunc != nil {
		return m.GetAllStateFunc(sessionID)
	}
	if session, ok := m.state[sessionID]; ok {
		return session, nil
	}
	return make(map[string]string), nil
}

// Ensure MockStateStore implements StateStore interface
var _ interfaces.StateStore = (*MockStateStore)(nil)

// MockPlatformDetector is a mock implementation of PlatformDetector for testing
type MockPlatformDetector struct {
	DetectOSFunc          func() (string, error)
	DetectDistroFunc      func() (string, error)
	IsTermuxFunc          func() bool
	GetPackageManagerFunc func() (string, error)
}

func (m *MockPlatformDetector) DetectOS() (string, error) {
	if m.DetectOSFunc != nil {
		return m.DetectOSFunc()
	}
	return "linux", nil
}

func (m *MockPlatformDetector) DetectDistro() (string, error) {
	if m.DetectDistroFunc != nil {
		return m.DetectDistroFunc()
	}
	return "ubuntu", nil
}

func (m *MockPlatformDetector) IsTermux() bool {
	if m.IsTermuxFunc != nil {
		return m.IsTermuxFunc()
	}
	return false
}

func (m *MockPlatformDetector) GetPackageManager() (string, error) {
	if m.GetPackageManagerFunc != nil {
		return m.GetPackageManagerFunc()
	}
	return "apt", nil
}

// Ensure MockPlatformDetector implements PlatformDetector interface
var _ interfaces.PlatformDetector = (*MockPlatformDetector)(nil)

// MockLLMClient is a mock implementation of LLMClient for testing
type MockLLMClient struct {
	ClassifyFunc    func(input string, candidates []models.Candidate) (*models.IntentResult, error)
	IsAvailableFunc func() bool
}

func (m *MockLLMClient) Classify(input string, candidates []models.Candidate) (*models.IntentResult, error) {
	if m.ClassifyFunc != nil {
		return m.ClassifyFunc(input, candidates)
	}
	return &models.IntentResult{
		ModuleID:   "test.module",
		Confidence: 0.9,
		Method:     "llm",
		Candidates: candidates,
	}, nil
}

func (m *MockLLMClient) IsAvailable() bool {
	if m.IsAvailableFunc != nil {
		return m.IsAvailableFunc()
	}
	return true
}

// Ensure MockLLMClient implements LLMClient interface
var _ interfaces.LLMClient = (*MockLLMClient)(nil)

// MockDatabaseConnection is a mock implementation of DatabaseConnection for testing
type MockDatabaseConnection struct {
	ConnFunc    func() *sql.DB
	CloseFunc   func() error
	MigrateFunc func() error
	conn        *sql.DB
}

func (m *MockDatabaseConnection) Conn() *sql.DB {
	if m.ConnFunc != nil {
		return m.ConnFunc()
	}
	return m.conn
}

func (m *MockDatabaseConnection) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func (m *MockDatabaseConnection) Migrate() error {
	if m.MigrateFunc != nil {
		return m.MigrateFunc()
	}
	return nil
}

// Ensure MockDatabaseConnection implements DatabaseConnection interface
var _ interfaces.DatabaseConnection = (*MockDatabaseConnection)(nil)
