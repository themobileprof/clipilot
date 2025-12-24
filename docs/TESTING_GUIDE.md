# CLIPilot Test Suite Documentation

## Overview

CLIPilot uses a comprehensive test suite integrated into the CI/CD pipeline to ensure code quality and reliability across all supported platforms.

## Test Structure

### Unit Tests (`*_test.go`)

Unit tests are located alongside the code they test:

```
internal/
├── db/
│   └── db_test.go              # Database layer tests (66.1% coverage)
├── engine/
│   └── runner_test.go          # Flow execution tests (48.6% coverage)
├── intent/
│   └── keyword_test.go         # Intent detection tests (77.9% coverage)
└── modules/
    └── loader_test.go          # Module loading tests (83.8% coverage)

pkg/
└── config/
    └── config_test.go          # Configuration tests (72.0% coverage)
```

### Integration Tests (`integration_test.go`)

Integration tests verify end-to-end functionality:
- CLI binary building and execution
- Module loading workflows
- Cross-platform compilation
- Docker image building
- Module YAML validation

**Note:** Integration tests require the `integration` build tag:
```bash
go test -tags=integration ./...
```

## Running Tests

### All Unit Tests
```bash
go test ./...
```

### With Coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.txt -covermode=atomic ./...
```

### With Race Detection
```bash
go test -race ./...
```

### Integration Tests Only
```bash
go test -tags=integration ./...
```

### Specific Package
```bash
go test ./internal/db
go test ./internal/intent -v
```

### Single Test
```bash
go test ./internal/db -run TestNew
go test ./internal/intent -run TestDetect
```

### Benchmarks
```bash
go test -bench=. ./internal/intent
go test -bench=. -benchmem ./internal/modules
```

## CI/CD Integration

### GitHub Actions Workflows

#### `.github/workflows/test.yml`
Runs on every push and PR to `main` and `develop`:

1. **Test Job** (Matrix: Ubuntu + macOS, Go 1.24)
   - Checkout code
   - Set up Go
   - Cache Go modules
   - Download dependencies
   - Run tests with race detector and coverage
   - Upload coverage to Codecov

2. **Build Job**
   - Build CLI binary (`./bin/clipilot`)
   - Build registry server (`./bin/registry`)
   - Test binaries (--version, --help)

3. **Lint Job**
   - Run golangci-lint with 5-minute timeout
   - Check code formatting and style

### Coverage Requirements

Current coverage by package:
- `internal/db`: 66.1%
- `internal/engine`: 48.6%
- `internal/intent`: 77.9%
- `internal/modules`: 83.8%
- `pkg/config`: 72.0%

**Target:** Maintain >60% coverage overall, >70% for core packages (intent, modules).

## Test Patterns

### Database Tests

All database tests use temporary SQLite databases:

```go
func setupTestDB(t *testing.T) (*db.DB, func()) {
    tmpDir, err := os.MkdirTemp("", "clipilot-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }

    dbPath := filepath.Join(tmpDir, "test.db")
    database, err := db.New(dbPath)
    if err != nil {
        t.Fatalf("Failed to create database: %v", err)
    }

    cleanup := func() {
        database.Close()
        os.RemoveAll(tmpDir)
    }

    return database, cleanup
}
```

### Module Tests

Module tests verify YAML parsing and database import:

```go
func TestLoadFromFile(t *testing.T) {
    // Create test YAML
    yamlContent := `name: test_module
id: org.test.test_module
version: 1.0.0
...`
    
    // Load and verify
    module, err := loader.LoadFromFile(testFile)
    if err != nil {
        t.Fatalf("LoadFromFile failed: %v", err)
    }
    
    // Assert fields
    if module.Name != "test_module" {
        t.Errorf("Expected name 'test_module', got %s", module.Name)
    }
}
```

### Intent Detection Tests

Test tokenization and keyword matching:

```go
func TestTokenize(t *testing.T) {
    tests := []struct {
        input    string
        expected []string
    }{
        {"install mysql database", []string{"install", "mysql", "database"}},
        {"THE and FOR with", []string{}}, // Stop words filtered
    }
    
    for _, tt := range tests {
        result := tokenize(tt.input)
        // Assert length and content
    }
}
```

## Writing New Tests

### 1. Add Test File

Create `*_test.go` in the same package:
```bash
touch internal/newpackage/newpackage_test.go
```

### 2. Follow Naming Convention

- Test functions: `func TestFunctionName(t *testing.T)`
- Benchmark functions: `func BenchmarkFunctionName(b *testing.B)`
- Table-driven tests for multiple cases

### 3. Use Test Helpers

- `setupTestDB()` for database tests
- `t.Helper()` for test utilities
- `t.Run()` for subtests
- `t.Cleanup()` for deferred cleanup

### 4. Test Error Cases

Always test both success and failure paths:
```go
func TestFunction(t *testing.T) {
    // Test success
    result, err := Function(validInput)
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    // Test error
    _, err = Function(invalidInput)
    if err == nil {
        t.Error("Expected error, got nil")
    }
}
```

## Before Submitting PR

Run the full test suite locally:

```bash
# Unit tests with race detection
go test -v -race -cover ./...

# Integration tests
go test -tags=integration ./...

# Linting (optional, CI will run it)
golangci-lint run --timeout=5m
```

Expected output:
- ✅ All tests pass
- ✅ No race conditions detected
- ✅ Coverage maintained or improved
- ✅ No linting errors

## Debugging Test Failures

### Verbose Output
```bash
go test -v ./internal/db
```

### Run Single Test
```bash
go test -v -run TestNew ./internal/db
```

### Check Race Conditions
```bash
go test -race ./internal/db
```

### Enable Debug Logging
```bash
export CLIPILOT_LOG_LEVEL=debug
go test -v ./...
```

### Inspect Test Database
If a test fails with database issues:
```bash
# Tests create temp DBs in /tmp/clipilot-test-*
ls -la /tmp/clipilot-test-*

# If test doesn't clean up, inspect manually
sqlite3 /tmp/clipilot-test-*/test.db
.tables
SELECT * FROM modules;
```

## Continuous Integration Details

### Workflow Triggers
- **Push to main/develop:** Full test + build + lint
- **Pull Request:** Same as push
- **Version tag (v*):** Release workflow (includes tests)

### Test Matrix
- **Operating Systems:** Ubuntu Latest, macOS Latest
- **Go Versions:** 1.24
- **Timeout:** 10 minutes per job

### Failure Handling
- Tests must pass before merge
- Coverage cannot decrease by >5%
- Race conditions block merge
- Linting errors block merge

## Coverage Reports

View detailed coverage on Codecov:
- https://codecov.io/gh/themobileprof/clipilot

Generate local HTML coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

## Performance Testing

Run benchmarks to detect performance regressions:
```bash
# Run all benchmarks
go test -bench=. ./...

# Compare with baseline
go test -bench=. -benchmem ./internal/intent > new.txt
benchcmp baseline.txt new.txt
```

## Testing Best Practices

1. **Isolation:** Tests should not depend on each other
2. **Cleanup:** Always clean up temp files and connections
3. **Parallelization:** Use `t.Parallel()` for independent tests
4. **Table-driven:** Use table-driven tests for multiple inputs
5. **Clear names:** Test names should describe what they test
6. **Fast:** Keep unit tests under 100ms each
7. **Deterministic:** Tests should always produce same results

## Known Issues

### SQLite Concurrency
SQLite is configured for single connection (best for file-based DB). Tests use separate temp databases to avoid conflicts.

### Android/Termux Testing
Integration tests for Android builds require cross-compilation. Manual testing needed on actual devices.

### Docker Tests
Docker build tests are skipped if Docker is not available (CI environments have Docker installed).

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Codecov Documentation](https://docs.codecov.com/)
- [golangci-lint](https://golangci-lint.run/)

---

**Questions?** Open an issue or check [CONTRIBUTING.md](../CONTRIBUTING.md)
