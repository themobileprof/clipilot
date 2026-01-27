#!/bin/bash
# CLIPilot Test Runner Script
# Provides quick commands for running different test scenarios

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}       CLIPilot Test Suite Runner${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Function to run tests and show results
run_test() {
    local test_name=$1
    local test_cmd=$2
    
    echo -e "${YELLOW}Running: ${test_name}${NC}"
    echo "Command: $test_cmd"
    echo ""
    
    if eval "$test_cmd"; then
        echo -e "${GREEN}✅ ${test_name} PASSED${NC}"
        echo ""
        return 0
    else
        echo -e "${RED}❌ ${test_name} FAILED${NC}"
        echo ""
        return 1
    fi
}

# Parse command line arguments
case "${1:-all}" in
    all)
        echo "Running static analysis (vet)..."
        if ! run_test "Go Vet" "go vet ./..."; then
            echo -e "${YELLOW}⚠️  Static analysis failed. Fix issues above.${NC}"
            exit 1
        fi
        
        echo "Running all unit tests..."
        run_test "All Unit Tests" "go test ./..."
        ;;

    lint)
        echo "Running linter..."
        LINT_CMD=""
        if [ -f "./bin/golangci-lint" ]; then
            LINT_CMD="./bin/golangci-lint run"
        elif command -v golangci-lint &> /dev/null; then
            LINT_CMD="golangci-lint run"
        fi

        if [ -n "$LINT_CMD" ]; then
             run_test "Linter" "$LINT_CMD"
        else
             echo -e "${YELLOW}⚠️  golangci-lint not found.${NC}"
             echo -e "To install: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v1.64.4"
             # Fallback to vet
             run_test "Go Vet" "go vet ./..."
        fi
        ;;
    
    coverage)
        echo "Running tests with coverage..."
        run_test "Coverage" "go test -cover ./..."
        echo ""
        echo "Generating HTML coverage report..."
        go test -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
        echo -e "${GREEN}✅ Coverage report generated: coverage.html${NC}"
        ;;
    
    race)
        echo "Running tests with race detector..."
        run_test "Race Detection" "go test -race ./..."
        ;;
    
    integration)
        echo "Running integration tests..."
        run_test "Integration Tests" "go test -tags=integration ./..."
        ;;
    
    bench)
        echo "Running benchmarks..."
        run_test "Benchmarks" "go test -bench=. -benchmem ./internal/intent ./internal/modules"
        ;;
    
    db)
        echo "Running database tests..."
        run_test "Database Tests" "go test -v ./internal/db"
        ;;
    
    engine)
        echo "Running engine tests..."
        run_test "Engine Tests" "go test -v ./internal/engine"
        ;;
    
    intent)
        echo "Running intent detection tests..."
        run_test "Intent Tests" "go test -v ./internal/intent"
        ;;
    
    modules)
        echo "Running module loader tests..."
        run_test "Module Tests" "go test -v ./internal/modules"
        ;;
    
    config)
        echo "Running config tests..."
        run_test "Config Tests" "go test -v ./internal/config"
        ;;
    
    ci)
        echo "Running CI/CD test suite (full)..."
        echo ""
        run_test "Unit Tests" "go test -v ./..."
        run_test "Race Detection" "go test -race ./..."
        run_test "Coverage" "go test -coverprofile=coverage.txt -covermode=atomic ./..."
        run_test "Build CLI" "go build -o bin/clipilot ./cmd/clipilot"
        run_test "Build Registry" "go build -o bin/registry ./cmd/registry"
        echo ""
        echo -e "${GREEN}✅ Full CI/CD test suite PASSED${NC}"
        ;;
    
    quick)
        echo "Running quick test (no race detector)..."
        run_test "Quick Tests" "go test ./internal/..."
        ;;
    
    verbose)
        echo "Running verbose tests..."
        run_test "Verbose Tests" "go test -v ./..."
        ;;
    
    help|--help|-h)
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  all         - Run all unit tests (default)"
        echo "  coverage    - Run with coverage and generate HTML report"
        echo "  race        - Run with race detector"
        echo "  integration - Run integration tests"
        echo "  bench       - Run benchmarks"
        echo "  db          - Run database tests only"
        echo "  engine      - Run engine tests only"
        echo "  intent      - Run intent detection tests only"
        echo "  modules     - Run module loader tests only"
        echo "  config      - Run config tests only"
        echo "  ci          - Run full CI/CD test suite"
        echo "  quick       - Quick test (no race detector)"
        echo "  verbose     - Run with verbose output"
        echo "  help        - Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0              # Run all unit tests"
        echo "  $0 coverage     # Generate coverage report"
        echo "  $0 ci           # Run full CI/CD suite"
        echo "  $0 db           # Test database package only"
        ;;
    
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        echo "Run '$0 help' for usage information"
        exit 1
        ;;
esac

echo ""
echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}              Test Run Complete${NC}"
echo -e "${BLUE}================================================${NC}"
