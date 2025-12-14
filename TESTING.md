# CLIPilot Registry - Testing Guide

## 🚀 Quick Test (Visual)

The registry server is already running at **http://localhost:8080**

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
