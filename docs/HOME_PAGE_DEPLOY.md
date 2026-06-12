# Home Page Deployment Guide

## Changes Made

Updated [server/templates/home.html](../server/templates/home.html) and [server/templates/modules.html](../server/templates/modules.html) to reflect the current Clio architecture.

### Key Messaging Now Prominent:
- ✅ **Plain English → shell commands** — interactive REPL with `>>` prompt, confirm-before-run
- ✅ **Setup wizards vs automation modules** — clearly separated (matches `catalog` / `setup` / `modules` in Clio)
- ✅ **Offline-first** — local catalog, cached modules, optional remote search via this registry
- ✅ **Termux / Android** — `clio-run-module`, lite profile, `sync` / `sync full`
- ✅ **Registry integration** — modules download on demand from `/api/v1/modules`

---

## Automatic Clio Install Script Sync

The deployment workflow automatically fetches and uploads the latest Clio install script during deployment:

1. **Fetch**: Downloads `install.sh` from `https://raw.githubusercontent.com/themobileprof/clio/main/install.sh`
2. **Extract Version**: Reads `VERSION=` from the script
3. **Store**: Saves to `/var/lib/clipilot-registry/uploads/install_scripts/`
4. **Activate**: Script becomes available at `/clio`

Implementation: [scripts/remote-deploy.sh](../scripts/remote-deploy.sh), invoked by [.github/workflows/ci.yml](../.github/workflows/ci.yml).

---

## Deploying to Production

Push to `main` to deploy automatically:

```bash
git add server/templates/home.html
git commit -m "Update home page messaging"
git push origin main
```

The CI/CD workflow builds the binary, copies it to the server, and restarts the `clipilot-registry` systemd service.

---

## Verification Checklist

After deployment, visit your production URL and verify:

- [ ] Hero title shows: **"Clio: Your AI CLI Assistant"**
- [ ] Subtitle mentions offline and Termux support
- [ ] Features section shows command search, offline ready, works everywhere
- [ ] Example usage section visible with conversational examples

---

## Troubleshooting

### "Still seeing old page"

Hard refresh the browser (`Ctrl+Shift+R` / `Cmd+Shift+R`).

Check the service on the production server:

```bash
sudo systemctl status clipilot-registry
sudo journalctl -u clipilot-registry -n 50
curl http://127.0.0.1:8080/health
```

### "Deployment workflow failed"

Check GitHub Actions logs for the **Deploy** job in `.github/workflows/ci.yml`.

---

## Testing Locally

```bash
go build -o clipilot-server ./cmd/registry
./clipilot-server
curl http://localhost:8080/ | grep "Clio"
```

---

## Related Files

- [server/templates/home.html](../server/templates/home.html) - Main landing page
- [server/static/style.css](../server/static/style.css) - Material Design styles
- [.github/workflows/ci.yml](../.github/workflows/ci.yml) - CI/CD workflow
- [scripts/remote-deploy.sh](../scripts/remote-deploy.sh) - Server-side deploy script
