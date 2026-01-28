# Release Distribution

Multi-platform package distribution using GoReleaser.

## Quick Start

```bash
# Create and push a tag
git tag v0.1.0
git push origin v0.1.0
# GitHub Actions will automatically build and release
```

## Distribution Channels

| Channel | Platform | Config | Notes |
|---------|----------|--------|-------|
| **GitHub Release** | All | Built-in | Binary downloads + checksums |
| **Homebrew** | macOS/Linux | `brews` | Requires `homebrew-tap` repo |
| **Scoop** | Windows | `scoops` | Requires `scoop-bucket` repo |
| **APT (deb)** | Debian/Ubuntu | `nfpms` | Via GitHub Release |
| **RPM** | RHEL/Fedora | `nfpms` | Via GitHub Release |
| **Chocolatey** | Windows | `chocolateys` | Requires API key |
| **AUR** | Arch Linux | `aurs` | Requires SSH key |

## Setup Requirements

### 1. Homebrew Tap

Create repo `YoungY620/homebrew-tap`:
```bash
# Users install with:
brew tap YoungY620/homebrew-tap
brew install memo
```

### 2. Scoop Bucket

Create repo `YoungY620/scoop-bucket`:
```bash
# Users install with:
scoop bucket add memo https://github.com/YoungY620/scoop-bucket
scoop install memo
```

### 3. Chocolatey (Optional)

1. Create account at https://chocolatey.org
2. Get API key
3. Add `CHOCOLATEY_API_KEY` to GitHub secrets

```powershell
# Users install with:
choco install memo
```

### 4. APT/DEB (Manual Repo)

For a proper APT repository, you need:
1. Host `.deb` files on a server
2. Create `Packages` and `Release` files
3. Sign with GPG

Simple alternative - download from GitHub:
```bash
# Download latest .deb
curl -LO https://github.com/YoungY620/memo/releases/latest/download/memo_*_linux_amd64.deb
sudo dpkg -i memo_*.deb
```

### 5. MSI Installer (Optional)

Add to `.goreleaser.yaml`:
```yaml
# Requires WiX Toolset on Windows runner
msi:
  - id: memo-msi
    name: memo
    wxs: ./build/wix/memo.wxs
```

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
| `.goreleaser.yaml` | GoReleaser config |
| `.github/workflows/release.yaml` | CI/CD pipeline |
