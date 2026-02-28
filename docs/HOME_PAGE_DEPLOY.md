# Home Page Deployment Guide

## Changes Made

Updated [server/templates/home.html](../server/templates/home.html) to emphasize Clio's key features:

### Key Messaging Now Prominent:
- ✅ **"Perfect for beginners"** - Natural language interface, no command memorization
- ✅ **Works offline** - After initial sync, runs completely locally
- ✅ **Android (Termux) support** - Full support highlighted as first-class platform
- ✅ **Example usage** - Shows real conversational examples with Clio

### Hero Section Updated:
**Before:** "CLI Automation Made Simple"  
**After:** "Clio: Your AI CLI Assistant"

**Before:** Generic description about workflows  
**After:** "Perfect for beginners. Just describe what you want in plain English—Clio handles the commands. Works offline on Linux, macOS, and Android (Termux)."

### Features Section:
Changed from "How It Works" (3 generic steps) to practical benefits:
1. **Natural Language** - For beginners, no syntax to learn
2. **Offline Ready** - Works without internet
3. **Works Everywhere** - Including Android/Termux

### New Example Usage Section:
Shows conversational examples:
- 💬 "install python development tools" → 🤖 Installs Python, pip, virtualenv
- 💬 "backup my documents folder" → 🤖 Creates timestamped backup
- 💬 "find large files over 1GB" → 🤖 Scans and lists files

---

## Deploying to Production

### Option 1: GitHub Actions (Recommended)

The updated templates are already in your Git repository. When you push to `main` or trigger a manual deploy:

```bash
git add server/templates/home.html
git commit -m "Update home page to emphasize Clio for beginners, offline, and Termux"
git push origin main
```

Then trigger manual deployment:
1. Go to GitHub Actions → "Manual Deploy (Hotfix)"
2. Click "Run workflow"
3. Select environment: **production**
4. Reason: "Deploy updated home page with Clio emphasis"
5. Click "Run workflow"

### Option 2: Docker Deployment Verification

The [Dockerfile.registry](../Dockerfile.registry) already includes templates:

```dockerfile
# Line 29-30
COPY --from=builder /build/server/static ./static
COPY --from=builder /build/server/templates ./templates
```

When the Docker image builds, it WILL include your updated `home.html`.

### Option 3: Manual Docker Build (Testing)

```bash
# Build Docker image locally
docker build -f Dockerfile.registry -t clipilot-registry:test .

# Run locally to test
docker run -p 8082:8080 \
  -e ADMIN_PASS=dev123 \
  clipilot-registry:test

# Visit http://localhost:8082 to verify changes
```

---

## Verification Checklist

After deployment, visit your production URL and verify:

- [ ] Hero title shows: **"Clio: Your AI CLI Assistant"**
- [ ] Subtitle mentions: **"Perfect for beginners"**
- [ ] Subtitle mentions: **"Works offline"**
- [ ] Subtitle mentions: **"Android (Termux)"**
- [ ] Features section shows: Natural Language, Offline Ready, Works Everywhere
- [ ] Example usage section visible with conversational examples
- [ ] Terminal preview shows multiple Clio commands
- [ ] Quickstart mentions "no experience needed"

---

## Troubleshooting

### "Still seeing old page"

**Browser cache:**
```bash
Ctrl+Shift+R  # Hard refresh (Chrome/Firefox)
Cmd+Shift+R   # Hard refresh (macOS)
```

**Docker not restarting:**
```bash
# SSH to production server
docker logs clipilot-registry  # Check if running
docker restart clipilot-registry  # Force restart
```

**Volume persistence issue:**
```bash
# Templates are baked into image, not in volume
# Volume only stores data/uploads/, not templates/
docker pull your-dockerhub-username/clipilot-registry:latest
docker stop clipilot-registry && docker rm clipilot-registry
# Run with new image (see deploy.yml lines 85-93)
```

### "Deployment workflow failed"

Check GitHub Actions logs:
1. Build step: Ensure `COPY server/templates ./templates` succeeded
2. SSH step: Ensure `docker pull` downloaded latest image
3. Container logs: `docker logs clipilot-registry`

---

## Testing Locally

Changes are live on your local server:
```bash
curl http://localhost:8080/ | grep "Clio: Your AI"
# Should output: <h1 class="hero-title">Clio: Your AI CLI Assistant</h1>
```

Visit: http://localhost:8080/

---

## Related Files

- [server/templates/home.html](../server/templates/home.html) - Main landing page
- [server/static/style.css](../server/static/style.css) - Material Design styles (unchanged)
- [Dockerfile.registry](../Dockerfile.registry) - Docker build (includes templates)
- [.github/workflows/deploy.yml](../.github/workflows/deploy.yml) - Deployment workflow
- [TRANSFORMATION_SUMMARY.md](../TRANSFORMATION_SUMMARY.md) - Complete transformation docs
