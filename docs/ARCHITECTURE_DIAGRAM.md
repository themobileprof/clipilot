# CLIPilot Architecture - Interface Boundaries

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLI Entry Point                              │
│                      (cmd/clipilot/main.go)                         │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         REPL Interface                               │
│                       (internal/ui/repl.go)                         │
└───────┬──────────────┬──────────────┬───────────────────────────────┘
        │              │              │
        ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│   Intent     │ │    Flow      │ │   Module     │
│  Classifier  │ │   Runner     │ │    Store     │  ◄── INTERFACES
│              │ │              │ │              │     (Testable Boundaries)
└──────┬───────┘ └──────┬───────┘ └──────┬───────┘
       │                │                │
       │                │                │
       ▼                ▼                ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│   Detector   │ │    Runner    │ │    Loader    │
│  (keyword)   │ │  (executor)  │ │   (YAML)     │  ◄── IMPLEMENTATIONS
│              │ │              │ │              │
└──────┬───────┘ └──────┬───────┘ └──────┬───────┘
       │                │                │
       │                │                │
       ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Database Layer                                │
│                   (internal/db/db.go)                               │
│                                                                      │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐               │
│  │  State Store │ │    Logger    │ │   Settings   │               │
│  │  (sessions)  │ │   (history)  │ │   Manager    │               │
│  └──────────────┘ └──────────────┘ └──────────────┘               │
└─────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════

                     TESTING WITH MOCKS

Production Flow:
  CLI → REPL → IntentClassifier (Detector) → Database
                    ↓
              ModuleStore (Loader) → Database
                    ↓
              FlowRunner (Runner) → Command Execution

Test Flow:
  Test → MockIntentClassifier (no database)
           ↓
       MockModuleStore (in-memory)
           ↓
       MockFlowRunner (no commands)

═══════════════════════════════════════════════════════════════════════

                  INTERFACE DEPENDENCY GRAPH

┌─────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  IntentClassifier ──uses──▶ ModuleStore (for keyword search)       │
│         │                                                            │
│         └──uses──▶ LLMClient (for layer 2/3 detection)             │
│                                                                      │
│  FlowRunner ──uses──▶ ModuleStore (to load flows)                  │
│      │                                                               │
│      ├──uses──▶ Executor (to run commands)                         │
│      │                                                               │
│      ├──uses──▶ StateStore (for inter-step data)                   │
│      │                                                               │
│      └──uses──▶ Logger (for execution history)                     │
│                                                                      │
│  ModuleStore ──uses──▶ DatabaseConnection                          │
│                                                                      │
│  Executor ──uses──▶ PlatformDetector (for OS-specific logic)       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════

                    MOCK INJECTION EXAMPLE

// Production
func main() {
    db := db.New(dbPath)
    loader := modules.NewLoader(db)          // Real implementation
    detector := intent.NewDetector(db)        // Real implementation
    runner := engine.NewRunner(db, loader)    // Real implementation
    
    repl := ui.NewREPL(db, detector, runner, loader)
    repl.Start()
}

// Testing
func TestIntentToExecution(t *testing.T) {
    mockLoader := mocks.NewMockModuleStore()  // Mock implementation
    mockDetector := &mocks.MockIntentClassifier{}
    mockRunner := &mocks.MockFlowRunner{}
    
    // Configure mocks
    mockDetector.DetectFunc = func(input string) (*models.IntentResult, error) {
        return &models.IntentResult{ModuleID: "test.module"}, nil
    }
    
    mockRunner.RunFunc = func(moduleID string) error {
        // Verify correct module is executed
        assert.Equal(t, "test.module", moduleID)
        return nil
    }
    
    // Test without real database, filesystem, or command execution
    repl := ui.NewREPL(nil, mockDetector, mockRunner, mockLoader)
    err := repl.ExecuteNonInteractive("test query")
    assert.NoError(t, err)
}

═══════════════════════════════════════════════════════════════════════

                  IMPLEMENTATION STATUS

✅ IMPLEMENTED (Phase 1):
   • IntentClassifier interface + intent.Detector implementation
   • ModuleStore interface + modules.Loader implementation  
   • FlowRunner interface + engine.Runner implementation
   • All mock implementations
   • Example tests with mocks
   • Compile-time interface verification

🔄 TODO (Phase 2):
   • Extract StateStore from DB
   • Extract Logger from DB
   • Extract SettingsManager from DB
   • Extract Executor from Runner
   • Implement PlatformDetector
   • Implement LLMClient (tiny + online)
   • Update Runner to use Executor interface
   • Update Detector to use LLMClient interface

═══════════════════════════════════════════════════════════════════════

                   TESTING BENEFITS

Before (Concrete Dependencies):
  ❌ Tests require real database
  ❌ Tests require filesystem access
  ❌ Tests execute actual shell commands
  ❌ Tests are slow (100-1000ms per test)
  ❌ Tests are flaky (network, filesystem)
  ❌ Hard to test edge cases

After (Interface Dependencies):
  ✅ Tests use in-memory mocks
  ✅ No filesystem or network needed
  ✅ No command execution
  ✅ Tests are fast (<1ms per test)
  ✅ Tests are deterministic
  ✅ Easy to test any scenario

═══════════════════════════════════════════════════════════════════════

Key Files:
  • internal/interfaces/interfaces.go - All interface definitions
  • internal/mocks/mocks.go - All mock implementations
  • internal/engine/runner_mocks_test.go - Example mock usage
  • docs/TESTABLE_ARCHITECTURE.md - Complete guide
  • docs/INTERFACE_IMPLEMENTATION_COMPLETE.md - Status report

Commands:
  go test ./internal/engine -run Mock  # Run mock-based tests
  go test ./...                         # Run all tests
  ./scripts/test.sh quick              # Quick test run
```
