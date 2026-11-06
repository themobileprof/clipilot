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
3. **Store**: Saves to `/var/lib/clio-registry/uploads/install_scripts/`
4. **Activate**: Script becomes available at `/clio`

Implementation: [scripts/remote-deploy.sh](../scripts/remote-deploy.sh), invoked by [.github/workflows/ci.yml](../.github/workflows/ci.yml).

---

## Deploying to Production

Deploy from a checkout:

```bash
./scripts/ship.sh
```

That builds the binary, copies it to the server, and restarts the `clio-registry` user service. GitHub Actions does not deploy while the account is locked for a billing issue.

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
sudo systemctl status clio-registry
sudo journalctl -u clio-registry -n 50
curl http://127.0.0.1:8080/health
```

### "Deploy failed"

Run `./scripts/ship.sh` again and read the health-check output. The service log on the server is `journalctl --user -u clio-registry -n 50`.

---

## Testing Locally

```bash
go build -o clio-server ./cmd/registry
./clio-server
curl http://localhost:8080/ | grep "Clio"
```

---

## Related Files

- [server/templates/home.html](../server/templates/home.html) - Main landing page
- [server/static/style.css](../server/static/style.css) - Material Design styles
- [.github/workflows/ci.yml](../.github/workflows/ci.yml) - CI/CD workflow
- [scripts/remote-deploy.sh](../scripts/remote-deploy.sh) - Server-side deploy script
