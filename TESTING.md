# CLIPilot Testing

## 🚀 Quick Start

### Using the Test Runner Script (Recommended)
```bash
# Run all tests
./scripts/test.sh

# Quick test (no race detector)
./scripts/test.sh quick

# Generate coverage report
./scripts/test.sh coverage

# Run specific package
./scripts/test.sh db
./scripts/test.sh intent
./scripts/test.sh modules

# Full CI/CD suite
./scripts/test.sh ci

# Show all options
./scripts/test.sh help
```

### Using Go Test Directly
```bash
# All unit tests
go test ./...

# With coverage
go test -cover ./...

# With race detection
go test -race ./...

# Integration tests
go test -tags=integration ./...

# Specific package
go test ./internal/db
```

## 📊 Current Test Status

- ✅ **52 tests** total (43 unit + 9 integration)
- ✅ **69.7% coverage** overall
- ✅ **All tests passing**
- ✅ **CI/CD integrated** via GitHub Actions
- ✅ **Race detection** enabled

### Coverage by Package
| Package | Coverage | Status |
|---------|----------|--------|
| `internal/db` | 66.1% | ✅ Good |
| `internal/engine` | 48.6% | ⚠️ Needs improvement |
| `internal/intent` | 77.9% | ✅ Excellent |
| `internal/modules` | 83.8% | ✅ Excellent |
| `pkg/config` | 72.0% | ✅ Good |

## 📁 Test Files

```
internal/
├── db/db_test.go           # 11 tests - database operations
├── engine/runner_test.go   # 9 tests - flow execution
├── intent/keyword_test.go  # 7 tests - intent detection
└── modules/loader_test.go  # 10 tests - module loading

pkg/
└── config/config_test.go   # 6 tests - configuration

integration_test.go         # 9 integration tests
```

## 🎯 Before Submitting PR

Run the full test suite:
```bash
./scripts/test.sh ci
```

Or manually:
```bash
go test -v -race -cover ./...
go test -tags=integration ./...
```

## 📚 Documentation

- [Testing Guide](docs/TESTING_GUIDE.md) - Comprehensive testing documentation
- [Test Coverage Report](docs/TEST_COVERAGE.md) - Detailed coverage status
- [Implementation Summary](docs/TEST_IMPLEMENTATION_SUMMARY.md) - What was built

## 🔍 Running Specific Tests

```bash
# Single package
go test ./internal/db
go test -v ./internal/intent

# Single test function
go test ./internal/db -run TestNew
go test ./internal/intent -run TestDetect

# Tests matching pattern
go test ./... -run "TestConcurrent"

# Benchmarks
go test -bench=. ./internal/intent
```

## 🐛 Debugging Failed Tests

```bash
# Verbose output
go test -v ./internal/db

# With race detector
go test -v -race ./internal/db

# Check for temp files (if cleanup fails)
ls -la /tmp/clipilot-test-*

# Run with custom timeout
go test -timeout 30s ./internal/db
```

## ⚡ Performance Testing

```bash
# Run benchmarks
./scripts/test.sh bench

# Or manually
go test -bench=. -benchmem ./internal/intent
go test -bench=. ./internal/modules
```

## 📈 Coverage Report

Generate HTML coverage report:
```bash
./scripts/test.sh coverage
# Opens coverage.html in your browser
```

Or manually:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
xdg-open coverage.html  # Linux
open coverage.html      # macOS
```

## 🎓 Writing New Tests

### Basic Test Template
```go
package yourpackage

import (
    "testing"
)

func TestYourFunction(t *testing.T) {
    // Setup
    input := "test input"
    expected := "expected output"
    
    // Execute
    result, err := YourFunction(input)
    
    // Assert
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Database Test Template
```go
func TestWithDatabase(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Your test logic here
}
```

### Table-Driven Test Template
```go
func TestMultipleCases(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "output1"},
        {"case2", "input2", "output2"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Function(tt.input)
            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

## 🔄 CI/CD Integration

Tests run automatically via GitHub Actions on:
- ✅ Push to `main` or `develop` branches
- ✅ Pull requests
- ✅ Version tags (releases)

Workflow file: [`.github/workflows/test.yml`](.github/workflows/test.yml)

### What CI/CD Runs
1. **Unit Tests** - All packages with race detection
2. **Coverage** - Upload to Codecov
3. **Build Verification** - CLI and registry binaries
4. **Linting** - golangci-lint
5. **Integration Tests** - Optional, manual trigger

## 🧪 Test Categories

### Unit Tests (43 tests)
Test individual functions and components in isolation.

**Database Tests (11)**
- Database creation and migration
- Settings CRUD operations
- State management
- Query logging
- Concurrent access

**Engine Tests (9)**
- Runner initialization
- Flow execution
- Condition evaluation
- Command execution
- Logging

**Intent Detection Tests (7)**
- Tokenization
- Keyword search
- Threshold configuration
- Benchmarks

**Module Tests (10)**
- YAML parsing
- Module import
- Module retrieval
- Listing

**Config Tests (6)**
- Configuration loading
- Saving
- Default values

### Integration Tests (9 tests)
Test full workflows and build processes.

- CLI binary build and execution
- Registry server build
- Module loading workflows
- Cross-platform compilation
- Docker image building
- Module YAML validation

## 🎯 Testing Best Practices

1. ✅ **Isolation** - Tests don't depend on each other
2. ✅ **Cleanup** - Always clean up temp files and connections
3. ✅ **Clear Names** - Test names describe what they test
4. ✅ **Fast** - Unit tests complete in <100ms
5. ✅ **Deterministic** - Always produce same results
6. ✅ **Error Paths** - Test both success and failure cases
7. ✅ **Table-Driven** - Use tables for multiple inputs

## 🚨 Common Issues

### Issue: Race Condition Detected
```bash
# Run with race detector to identify
go test -race ./...
```

### Issue: Test Hangs
```bash
# Add timeout
go test -timeout 10s ./internal/db
```

### Issue: Temp Files Not Cleaned
```bash
# Check for leftover files
ls -la /tmp/clipilot-test-*

# Clean manually
rm -rf /tmp/clipilot-test-*
```

## 📞 Getting Help

- Check [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) for detailed info
- See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines  
- Open an issue on GitHub for test-related problems

---

## Registry Server Testing

For testing the web-based module registry:

### Test in Browser (Easiest Method)

1. **Open the home page**: http://localhost:8080
   - You should see the CLIPilot Registry landing page
   - Check for: hero section, feature cards, quick start guide

2. **Browse modules**: http://localhost:8080/modules
   - Should show "0 modules available" (initially empty)
   - Grid layout should be visible

3. **Login**: http://localhost:8080/login
   - Username: `admin`
   - Password: `test123` (from .env file)
   - Should redirect to upload page after successful login

4. **Upload a module**: http://localhost:8080/upload
   - Use the test module: `/tmp/test_module.yaml`
   - Click "Choose File" → Select the file → Click "Upload Module"
   - Should see: ✓ Module uploaded successfully!

5. **View uploaded module**: http://localhost:8080/modules
   - Should now show your uploaded module
   - Click "Download" to verify it downloads correctly

6. **Check your modules**: http://localhost:8080/my-modules
   - Should list modules you uploaded
   - Shows upload date and download count

## 🧪 Automated Tests (Command Line)

### Test 1: Server Health Check
```bash
# Check home page
curl -s http://localhost:8080/ | grep -q "CLIPilot Registry" && echo "✓ Home page OK" || echo "✗ Home page failed"

# Check API endpoint
curl -s http://localhost:8080/api/modules | grep -q "^\[" && echo "✓ API OK" || echo "✗ API failed"

# Check modules page
curl -s http://localhost:8080/modules | grep -q "Browse Modules" && echo "✓ Modules page OK" || echo "✗ Failed"
```

### Test 2: Authentication Flow
```bash
# Test login
curl -c cookies.txt -X POST http://localhost:8080/login \
  -d "username=admin&password=test123" \
  -L -s -o /dev/null -w "HTTP %{http_code}\n"
# Should return: HTTP 200

# Test protected page without auth (should redirect to login)
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost:8080/upload
# Should return: HTTP 303 or 200 (if already logged in browser)

# Test protected page with auth
curl -b cookies.txt -s http://localhost:8080/upload | grep -q "Upload Module" && echo "✓ Auth OK" || echo "✗ Auth failed"
```

### Test 3: Module Upload
```bash
# Create test module
cat > /tmp/test_module.yaml << 'EOF'
name: test_hello_world
version: "1.0.0"
description: Simple hello world test module
tags:
  - test
  - hello
metadata:
  author: Test User
  license: MIT
flows:
  main:
    start: greet
    steps:
      greet:
        type: instruction
        message: "Hello from CLIPilot!"
        next: finish
      finish:
        type: terminal
        message: "✓ Test complete!"
EOF

# Login first
curl -c cookies.txt -L -X POST http://localhost:8080/login \
  -d "username=admin&password=test123" -s -o /dev/null

# Upload module
RESPONSE=$(curl -b cookies.txt -X POST http://localhost:8080/api/upload \
  -F "module=@/tmp/test_module.yaml" -s)

echo "$RESPONSE" | grep -q "success" && echo "✓ Upload OK" || echo "✗ Upload failed: $RESPONSE"

# Check if module appears in API
curl -s http://localhost:8080/api/modules | grep -q "test_hello_world" && echo "✓ Module listed" || echo "✗ Module not found"
```

### Test 4: Module Download
```bash
# Get module ID (assuming it's 1)
MODULE_ID=$(curl -s http://localhost:8080/api/modules | jq -r '.[0].id' 2>/dev/null || echo "1")

# Download module
curl -s http://localhost:8080/modules/$MODULE_ID -o /tmp/downloaded.yaml

# Verify it's valid YAML
if [ -f /tmp/downloaded.yaml ]; then
  grep -q "test_hello_world" /tmp/downloaded.yaml && echo "✓ Download OK" || echo "✗ Download failed"
else
  echo "✗ Module not found"
fi
```

### Test 5: Validation Tests
```bash
# Test invalid YAML upload
echo "invalid: [yaml" > /tmp/invalid.yaml
curl -b cookies.txt -X POST http://localhost:8080/api/upload \
  -F "module=@/tmp/invalid.yaml" -s | grep -q "Invalid YAML" && echo "✓ Validation OK" || echo "✗ Validation failed"

# Test duplicate upload (upload same module twice)
curl -b cookies.txt -X POST http://localhost:8080/api/upload \
  -F "module=@/tmp/test_module.yaml" -s | grep -q "already exists" && echo "✓ Duplicate check OK" || echo "✗ Duplicate check failed"
```

## 🔍 Manual Testing Checklist

### UI Testing
- [ ] Home page loads and displays correctly
- [ ] Navigation menu works (Home, Browse, Login links)
- [ ] Module grid displays properly (responsive)
- [ ] Login form accepts credentials
- [ ] Upload form has file input and specifications
- [ ] ChatGPT prompt is visible and copyable
- [ ] My Modules page shows user's uploads
- [ ] Logout redirects to home page
- [ ] Static CSS loads (check styling)

### Functionality Testing
- [ ] Can login with correct credentials
- [ ] Cannot login with wrong credentials (shows error)
- [ ] Cannot access /upload without authentication
- [ ] Can upload valid YAML module
- [ ] Cannot upload invalid YAML (shows error)
- [ ] Cannot upload duplicate module (shows error)
- [ ] Uploaded module appears in /modules list
- [ ] Can download module (increments counter)
- [ ] Download counter increases on each download
- [ ] API returns correct JSON
- [ ] Logout clears session

### Error Handling
- [ ] Invalid login shows error message
- [ ] Missing module file shows error
- [ ] Invalid YAML shows specific error
- [ ] Duplicate upload shows conflict error
- [ ] 404 for non-existent module
- [ ] Protected pages redirect to login

## 🎯 Test with CLIPilot CLI

### Test Module Installation
```bash
# Install module from registry
clipilot modules install 1

# Should see: ✓ Module test_hello_world (v1.0.0) installed successfully!

# List installed modules
clipilot modules list

# Should include test_hello_world

# Run the module
clipilot run test_hello_world
```

## 📊 Performance Testing

### Load Test (Optional)
```bash
# Install Apache Bench if needed
# sudo apt-get install apache2-utils

# Test API endpoint
ab -n 100 -c 10 http://localhost:8080/api/modules

# Test home page
ab -n 100 -c 10 http://localhost:8080/
```

## 🐛 Debugging

### Check Server Logs
```bash
# Server output should be in the terminal where you started it
# Look for errors or warnings

# Check if server is running
curl -s http://localhost:8080/ > /dev/null && echo "Server is running" || echo "Server is down"

# Check which port it's using
ps aux | grep registry
netstat -tlnp | grep 8080 || ss -tlnp | grep 8080
```

### Check Database
```bash
# List uploaded modules in database
sqlite3 data/registry.db "SELECT id, name, version, downloads FROM modules;"

# Check file storage
ls -lh data/uploads/
```

### Common Issues

**Problem:** Port already in use
```bash
# Solution: Use different port
PORT=9090 ADMIN_PASSWORD=test123 ./bin/registry
```

**Problem:** Templates not found
```bash
# Solution: Run from project root
cd /home/samuel/sites/clipilot
./bin/registry
```

**Problem:** Permission denied on data directory
```bash
# Solution: Fix permissions
chmod 755 data/
```

## 🎨 Visual Testing

### Browser DevTools Testing

1. **Open DevTools** (F12)
2. **Network Tab**: 
   - Check all resources load (HTML, CSS, static files)
   - Verify API calls return correct status codes
   - Check response payloads

3. **Console Tab**:
   - Should have no JavaScript errors
   - Check upload progress feedback

4. **Application Tab**:
   - Verify session cookie is set after login
   - Check cookie expiration (24 hours)

### Responsive Design Testing

Test on different viewport sizes:
- Desktop: 1920x1080
- Tablet: 768x1024
- Mobile: 375x667

Check:
- Navigation collapses properly
- Module grid adjusts to screen size
- Forms are usable on mobile
- Text is readable

## ✅ Success Criteria

Your registry is working correctly if:

✓ Server starts without errors  
✓ Home page loads with proper styling  
✓ Can login with credentials  
✓ Can upload valid YAML module  
✓ Uploaded module appears in browser  
✓ Can download module (YAML file)  
✓ API returns valid JSON  
✓ Validation rejects invalid YAML  
✓ Duplicate detection works  
✓ Session persists across requests  
✓ Logout clears session  
✓ CLI can install modules from registry  

## 🛑 Stopping the Test Server

```bash
# Find the registry process
ps aux | grep registry

# Kill it
pkill -f registry

# Or if you have the PID
kill <PID>

# Clean up test data (optional)
rm -rf data/
rm -f cookies.txt /tmp/test_module.yaml /tmp/downloaded.yaml
```

## 📝 Test Report Template

After testing, document your results:

```
## Test Results - [Date]

### Environment
- Server Version: 1.0.0
- Platform: [Linux/macOS]
- Port: 8080

### Tests Passed
- [ ] UI loads correctly
- [ ] Authentication works
- [ ] Module upload works
- [ ] Module download works
- [ ] API endpoints work
- [ ] Validation works
- [ ] CLI integration works

### Issues Found
1. [Issue description]
2. [Issue description]

### Performance
- API response time: [X]ms
- Upload time: [X]s
- Page load time: [X]s
```

---

**Current Status**: Server running at http://localhost:8080
**Credentials**: admin / test123
**Test Module**: /tmp/test_module.yaml

**Start testing now!** → Open http://localhost:8080 in your browser
