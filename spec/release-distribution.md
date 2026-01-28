# Release Distribution

macOS package distribution via Homebrew using GoReleaser.

## Status

- [x] GoReleaser config (macOS only)
- [x] GitHub Actions workflow
- [ ] Create `YoungY620/homebrew-tap` repository
- [ ] Test Homebrew installation

## Quick Start

```bash
# Create and push a tag
git tag v0.1.0
git push origin v0.1.0
# GitHub Actions will automatically build and release
```

## Installation (after setup)

```bash
brew tap YoungY620/tap
brew install memo
```

## Setup

### 1. Create Homebrew Tap Repository

Create a new GitHub repo: `YoungY620/homebrew-tap`

GoReleaser will automatically push Formula to this repo on release.

### 2. GitHub Secrets

The `GITHUB_TOKEN` is automatically provided by GitHub Actions.

For pushing to homebrew-tap, ensure the token has `repo` scope.

## Local Testing

```bash
# Install goreleaser
brew install goreleaser

# Test build (no publish)
goreleaser build --snapshot --clean

# Test full release (no publish)
goreleaser release --snapshot --clean
```

## Files

| File | Purpose |
|------|---------|
| `.goreleaser.yaml` | GoReleaser config (macOS only) |
| `.github/workflows/release.yaml` | CI/CD pipeline |

## Output

Each release produces:
- `memo_<version>_darwin_amd64.tar.gz` (Intel Mac)
- `memo_<version>_darwin_arm64.tar.gz` (Apple Silicon)
- `checksums.txt`
- Homebrew Formula (pushed to tap repo)
