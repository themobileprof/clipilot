# Releasing CLIPilot

This document describes how to create a new release of CLIPilot.

## Release Process

### 1. Prepare the Release

1. Ensure all changes are committed and pushed to `main`
2. Update version numbers if needed (in `cmd/clipilot/main.go`)
3. Update CHANGELOG.md with release notes

### 2. Create a Release Tag

```bash
# Tag the release (use semantic versioning)
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag
git push origin v1.0.0
```

### 3. Automated Build

The GitHub Actions workflow (`.github/workflows/release.yml`) will automatically:
- Build binaries for multiple platforms:
  - Linux: amd64, arm64, armv7
  - macOS: amd64 (Intel), arm64 (Apple Silicon)
- Create compressed archives (`.tar.gz`)
- Generate SHA256 checksums
- Package module files
- Create a GitHub Release with all artifacts

### 4. Verify the Release

1. Go to: https://github.com/themobileprof/clipilot/releases
2. Find the new release
3. Verify all binary artifacts are present:
   - `clipilot-linux-amd64.tar.gz`
   - `clipilot-linux-arm64.tar.gz`
   - `clipilot-linux-armv7.tar.gz`
   - `clipilot-darwin-amd64.tar.gz`
   - `clipilot-darwin-arm64.tar.gz`
   - `clipilot-modules.tar.gz`
   - Corresponding `.sha256` checksum files

### 5. Test the Release

Test the installation script with the new release:

```bash
# Test one-line installer
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash

# Verify it works
clipilot --version
```

## Manual Release (if needed)

If you need to build and release manually:

```bash
# Build for current platform
go build -ldflags="-s -w" -o clipilot ./cmd/clipilot

# Cross-compile for other platforms
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o clipilot-linux-amd64 ./cmd/clipilot
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o clipilot-linux-arm64 ./cmd/clipilot
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o clipilot-darwin-amd64 ./cmd/clipilot
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o clipilot-darwin-arm64 ./cmd/clipilot

# Create archives
for binary in clipilot-*; do
    tar czf "${binary}.tar.gz" "$binary"
    sha256sum "${binary}.tar.gz" > "${binary}.tar.gz.sha256"
done

# Create modules archive
cd modules
tar czf ../clipilot-modules.tar.gz *.yaml
cd ..
sha256sum clipilot-modules.tar.gz > clipilot-modules.tar.gz.sha256

# Upload to GitHub Release manually
# Use the GitHub web interface or gh CLI
gh release create v1.0.0 *.tar.gz *.sha256 --title "v1.0.0" --notes "Release notes here"
```

## Version Numbering

Follow semantic versioning (semver):
- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.1.0): New features, backwards compatible
- **Patch** (v1.0.1): Bug fixes, backwards compatible

## Troubleshooting

### Build fails with CGO errors
- For Linux builds, ensure cross-compilation tools are available
- For macOS builds, disable CGO: `CGO_ENABLED=0`

### Release workflow doesn't trigger
- Ensure you pushed the tag: `git push origin v1.0.0`
- Check GitHub Actions logs in the repository

### Users report "unsupported architecture"
- Add support in `.github/workflows/release.yml`
- Update `install.sh` to handle the new architecture
