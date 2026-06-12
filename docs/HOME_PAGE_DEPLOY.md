# Home Page Deployment Guide

## Changes Made

Updated [server/templates/home.html](../server/templates/home.html) and [server/templates/modules.html](../server/templates/modules.html) to reflect the current Clio architecture.

### Key Messaging Now Prominent:
- ✅ **Plain English → shell commands** — interactive REPL with `>>` prompt, confirm-before-run
- ✅ **Setup wizards vs automation modules** — clearly separated (matches `catalog` / `setup` / `modules` in Clio)
- ✅ **Offline-first** — local catalog, cached modules, optional remote search via this registry
- ✅ **Termux / Android** — `clio-run-module`, lite profile, `sync` / `sync full`
- ✅ **Registry integration** — modules download on demand from `/api/v1/modules`

### Hero Section Updated:
**Title:** "Clio: Your Terminal Assistant"

**Subtitle:** Offline-first CLI assistant. Ask in plain English; modules download from this registry on demand. Linux, macOS, and Termux.

### Primary Use Case: Command Search

**Section:** "Most Common Use: Finding Commands"

Shows 4 command search examples:
- 💬 "how do I compress a file?" → 🤖 `tar -czf archive.tar.gz file` with explanation
- 💬 "what command shows disk space?" → 🤖 `df -h` with usage examples
- 💬 "how to change file permissions?" → 🤖 `chmod` with practical examples
- 💬 "how do I search inside files?" → 🤖 `grep -r "pattern" .` with explanation

### Secondary Use Case: Automation

**Section:** "Automation Workflows"

Shows 3 multi-step automation examples:
- 💬 "install docker" → 🤖 Detects OS, adds repo, installs, starts service
- 💬 "backup my documents folder" → 🤖 Creates timestamped backup
- 💬 "setup python dev environment" → 🤖 Installs Python, pip, virtualenv, packages

### Features Section:
1. **Command Search** (First feature) - Most common use, search man pages
2. **Offline Ready** - Works without internet
3. **Works Everywhere** - Including Android/Termux

---

## Automatic Clio Install Script Sync

The deployment workflows now **automatically fetch and upload** the latest Clio install script during deployment:

### What Happens on Deploy:

1. **Fetch**: Downloads `install.sh` from `https://raw.githubusercontent.com/themobileprof/clio/main/install.sh`
2. **Extract Version**: Reads `VERSION=` from the script
3. **Create API Key**: Generates temporary CI/CD API key in the database
4. **Upload**: POSTs to `/api/install-script/upload` with Bearer token auth
5. **Activate**: Script becomes available at **clipilot.themobileprof.com/clio**

### Benefits:
- ✅ Always serves the latest Clio install script
- ✅ No manual upload needed after Clio updates
- ✅ Automatic on every deployment (hotfix and release)
- ✅ Graceful failure - deployment continues if upload fails

### Implementation:
Both workflows updated:
- [.github/workflows/deploy.yml](../.github/workflows/deploy.yml) - Manual hotfix deployments
- [.github/workflows/release.yml](../.github/workflows/release.yml) - Tag-based releases

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
