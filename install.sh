#!/bin/sh
# Memo installer for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/YoungY620/memo/main/install.sh | sh

set -e

REPO="YoungY620/memo"
INSTALL_DIR="$HOME/.local/bin"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac
PLATFORM="${OS}-${ARCH}"
echo "Platform: $PLATFORM"

# Get latest version
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
echo "Latest version: $LATEST"

# Check current version
if [ -x "$INSTALL_DIR/memo" ]; then
    CURRENT=$("$INSTALL_DIR/memo" --version 2>/dev/null | awk '{print $2}' || echo "")
    if [ "$CURRENT" = "$LATEST" ] || [ "v$CURRENT" = "$LATEST" ]; then
        echo "Already up to date: $LATEST"
        exit 0
    fi
    echo "Current version: $CURRENT"
fi

# Download and install
URL="https://github.com/$REPO/releases/download/$LATEST/memo-$PLATFORM"
echo "Downloading: $URL"
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" -o "$INSTALL_DIR/memo"
chmod +x "$INSTALL_DIR/memo"
echo "Installed: $INSTALL_DIR/memo"

# Check PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "Note: Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac
