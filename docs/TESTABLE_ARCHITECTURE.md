# Testable Architecture - Interface-Driven Design

## Overview

CLIPilot follows an **interface-driven architecture** where all core components are accessed through well-defined interfaces. This enables:

- ✅ **Easy mocking** for unit tests
- ✅ **Dependency injection** for flexibility
- ✅ **Loose coupling** between components
- ✅ **Better testability** without external dependencies
- ✅ **Implementation swapping** without breaking consumers

## Core Boundaries

All critical system boundaries are defined as Go interfaces in `internal/interfaces/interfaces.go`:

### 1. IntentClassifier
**Purpose**: Detects user intent from natural language input

**Interface**:
```go
type IntentClassifier interface {
    Detect(input string) (*models.IntentResult, error)
    SetThresholds(keyword, llm float64)
    SetOnlineEnabled(enabled bool)
}
```

**Implementation**: `internal/intent.Detector`

**Used By**: REPL, CLI commands

**Testability**: Easy to mock different detection strategies (keyword-only, LLM-based, hybrid)

---

### 2. ModuleStore
**Purpose**: Handles module loading, storage, and retrieval

**Interface**:
```go
type ModuleStore interface {
    LoadFromFile(path string) (*models.Module, error)
    ImportModule(module *models.Module) error
    GetModule(moduleID string) (*models.Module, error)
    ListModules() ([]models.Module, error)
}
```

**Implementation**: `internal/modules.Loader`

**Used By**: Engine runner, CLI module commands, registry sync

**Testability**: Can mock module loading without database or filesystem access

---

### 3. FlowRunner
**Purpose**: Executes module flows with their steps

**Interface**:
```go
type FlowRunner interface {
    Run(moduleID string) error
    SetDryRun(enabled bool)
    SetAutoYes(enabled bool)
}
```

**Implementation**: `internal/engine.Runner`

**Used By**: REPL query handler, CLI run commands

**Testability**: Can test flow execution logic without actual command execution

---

### 4. Executor
**Purpose**: Executes individual steps within a flow

**Interface**:
```go
type Executor interface {
    ExecuteCommand(command string) (string, error)
    ValidateCommand(command string) error
}
```

**Implementation**: (TODO: Extract from Runner)

**Used By**: Flow runner

**Testability**: Can mock command execution for testing flow logic without shell access

---

### 5. StateStore
**Purpose**: Manages execution state and session data

**Interface**:
```go
type StateStore interface {
    GetState(sessionID, key string) (string, error)
    SetState(sessionID, key, value string) error
    DeleteState(sessionID string) error
    GetAllState(sessionID string) (map[string]string, error)
}
```

**Implementation**: (TODO: Extract from DB)

**Used By**: Flow runner for inter-step communication

**Testability**: Can test state-dependent logic with in-memory mock

---

### 6. PlatformDetector
**Purpose**: Identifies the operating system and environment

**Interface**:
```go
type PlatformDetector interface {
    DetectOS() (string, error)
    DetectDistro() (string, error)
    IsTermux() bool
    GetPackageManager() (string, error)
}
```

**Implementation**: (TODO: Create implementation)

**Used By**: Module execution for platform-specific logic

**Testability**: Can test Termux-specific behavior on non-Termux systems

---

### 7. LLMClient
**Purpose**: Provides access to language model inference

**Interface**:
```go
type LLMClient interface {
    Classify(input string, candidates []models.Candidate) (*models.IntentResult, error)
    IsAvailable() bool
}
```

**Implementation**: (TODO: Create tiny LLM and online LLM implementations)

**Used By**: Intent detector Layer 2 and Layer 3

**Testability**: Can test intent detection without LLM access

---

## Testing with Interfaces

### Basic Pattern

```go
func TestWithMocks(t *testing.T) {
    // Create mocks
    mockStore := mocks.NewMockModuleStore()
    mockRunner := &mocks.MockFlowRunner{}
    
    // Configure behavior
    mockStore.GetModuleFunc = func(id string) (*models.Module, error) {
        return &models.Module{ID: id}, nil
    }
    
    // Test your code with mocks
    // ...
}
```

### Example: Testing Runner with Mock Module Store

```go
func TestRunnerWithMocks(t *testing.T) {
    // Setup
    mockStore := mocks.NewMockModuleStore()
    testModule := &models.Module{
        ID: "test.module",
        Flows: map[string]*models.Flow{
            "main": {
                Start: "step1",
                Steps: map[string]*models.Step{
                    "step1": {Type: "terminal", Message: "Done"},
                },
            },
        },
    }
    
    mockStore.GetModuleFunc = func(id string) (*models.Module, error) {
        return testModule, nil
    }
    
    // Create runner with mock
    db := setupTestDB(t) // Minimal DB for logging
    runner := engine.NewRunner(db, mockStore)
    runner.SetDryRun(true)
    
    // Test
    err := runner.Run("test.module")
    if err != nil {
        t.Errorf("Run failed: %v", err)
    }
}
```

### Example: Testing Intent Detection with Mock LLM

```go
func TestIntentWithMockLLM(t *testing.T) {
    mockLLM := &mocks.MockLLMClient{}
    
    // Simulate LLM unavailable
    mockLLM.IsAvailableFunc = func() bool {
        return false
    }
    
    // Test fallback behavior
    // ...
}
```

### Example: Testing Platform-Specific Logic

```go
func TestTermuxBehavior(t *testing.T) {
    mockPlatform := &mocks.MockPlatformDetector{}
    
    // Simulate Termux environment
    mockPlatform.IsTermuxFunc = func() bool {
        return true
    }
    
    mockPlatform.GetPackageManagerFunc = func() (string, error) {
        return "pkg", nil
    }
    
    // Test Termux-specific logic
    // ...
}
```

## Mock Implementations

All mocks are provided in `internal/mocks/mocks.go`. Each mock:

- Implements the corresponding interface
- Has configurable function fields for custom behavior
- Has sensible defaults for common use cases
- Includes state tracking for assertions

### Available Mocks

```go
// Module management
type MockModuleStore struct { ... }

// Intent detection
type MockIntentClassifier struct { ... }

// Flow execution
type MockFlowRunner struct { ... }

// Command execution
type MockExecutor struct { ... }

// State management
type MockStateStore struct { ... }

// Platform detection
type MockPlatformDetector struct { ... }

// LLM access
type MockLLMClient struct { ... }

// Database connection
type MockDatabaseConnection struct { ... }
```

## Benefits of Interface-Driven Design

### 1. Unit Test Isolation
Test components in complete isolation without:
- Database setup
- Filesystem access
- Network calls
- External processes
- LLM services

### 2. Faster Tests
Mocks execute instantly compared to:
- Database queries (10-100ms)
- File I/O (5-50ms)
- Network calls (100-1000ms)
- LLM inference (500-2000ms)

### 3. Predictable Tests
Mock behavior is deterministic, eliminating:
- Network flakiness
- Filesystem race conditions
- Database state pollution
- External service outages

### 4. Edge Case Testing
Easy to test error scenarios:
```go
mockStore.GetModuleFunc = func(id string) (*models.Module, error) {
    return nil, errors.New("database connection lost")
}
```

### 5. Implementation Swapping
Switch implementations without changing consumers:
- Keyword → Tiny LLM → Online LLM
- SQLite → PostgreSQL → DynamoDB
- Local → S3 module storage

## Architecture Verification

All implementations include compile-time interface checks:

```go
// Ensures Loader implements ModuleStore
var _ interfaces.ModuleStore = (*Loader)(nil)

// Ensures Runner implements FlowRunner
var _ interfaces.FlowRunner = (*Runner)(nil)

// Ensures Detector implements IntentClassifier
var _ interfaces.IntentClassifier = (*Detector)(nil)
```

If implementation doesn't match interface, compilation fails immediately.

## Migration Guide

### Before (Concrete Dependencies)
```go
type Runner struct {
    loader *modules.Loader  // Concrete type
}

func NewRunner(db *sql.DB) *Runner {
    return &Runner{
        loader: modules.NewLoader(db),
    }
}
```

**Problem**: Can't test Runner without real database and filesystem

### After (Interface Dependencies)
```go
type Runner struct {
    loader interfaces.ModuleStore  // Interface
}

func NewRunner(db *sql.DB, loader interfaces.ModuleStore) *Runner {
    return &Runner{
        loader: loader,
    }
}
```

**Solution**: Inject any ModuleStore implementation (real or mock)

## Best Practices

### 1. Keep Interfaces Small
Each interface should have a single, clear responsibility. Prefer multiple small interfaces over one large interface.

### 2. Define Interfaces at Boundaries
Put interfaces where components connect, not inside components.

### 3. Accept Interfaces, Return Structs
```go
// Good
func NewRunner(store interfaces.ModuleStore) *Runner

// Bad
func NewRunner(store *Loader) interfaces.FlowRunner
```

### 4. Use Interface Assertions
Always verify implementations match interfaces:
```go
var _ interfaces.ModuleStore = (*Loader)(nil)
```

### 5. Test with Mocks First
Write tests with mocks before implementing:
1. Define interface
2. Create mock
3. Write tests with mock
4. Implement real version
5. Verify tests still pass

## TODO: Remaining Work

### Phase 1: Extract Interfaces (DONE ✅)
- [x] Define core interfaces
- [x] Create mock implementations
- [x] Add interface assertions to existing code
- [x] Write example tests with mocks

### Phase 2: Refactor Database Layer
- [ ] Create `StateStore` implementation
- [ ] Create `SettingsManager` implementation
- [ ] Create `Logger` implementation
- [ ] Update `internal/db` to implement `DatabaseConnection`

### Phase 3: Implement Missing Components
- [ ] Create `Executor` implementation (extract from Runner)
- [ ] Create `PlatformDetector` implementation
- [ ] Create `LLMClient` implementations (tiny + online)

### Phase 4: Update Dependencies
- [ ] Update `Runner` to use `Executor` interface
- [ ] Update `Detector` to use `LLMClient` interface
- [ ] Update all code to use `StateStore` instead of direct DB

### Phase 5: Comprehensive Mock Tests
- [ ] Add mock tests for all packages
- [ ] Test error scenarios with mocks
- [ ] Test edge cases without external dependencies
- [ ] Increase overall coverage to 80%+

## Testing Commands

```bash
# Run tests with new mocks
go test ./internal/engine/...

# Verify interface compliance
go build ./internal/...

# Check mock coverage
go test -cover ./internal/mocks/...

# Run all tests
./scripts/test.sh quick
```

## See Also

- [TESTING_GUIDE.md](TESTING_GUIDE.md) - Comprehensive testing guide
- [TEST_COVERAGE.md](TEST_COVERAGE.md) - Current coverage status
- `internal/interfaces/interfaces.go` - Interface definitions
- `internal/mocks/mocks.go` - Mock implementations
- `internal/engine/runner_mocks_test.go` - Example mock usage

---

**Status**: Phase 1 complete. Interfaces defined, mocks created, examples provided.

**Next Step**: Extract StateStore, Logger, and Executor from current implementations.
