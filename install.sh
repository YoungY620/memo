#!/bin/bash
# Memo installer script
# Usage: curl -sSL https://raw.githubusercontent.com/YoungY620/memo/main/install.sh | bash

set -e

REPO="YoungY620/memo"
BINARY="memo"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    mingw*|msys*|cygwin*) OS="windows" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest release tag
LATEST=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "Failed to get latest release"
    exit 1
fi

# Build download URL
if [ "$OS" = "windows" ]; then
    FILENAME="${BINARY}-${OS}-${ARCH}.exe"
else
    FILENAME="${BINARY}-${OS}-${ARCH}"
fi
URL="https://github.com/$REPO/releases/download/$LATEST/$FILENAME"

echo "Downloading $BINARY $LATEST for $OS/$ARCH..."
curl -sSL "$URL" -o "/tmp/$BINARY"
chmod +x "/tmp/$BINARY"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "Installed $BINARY $LATEST to $INSTALL_DIR/$BINARY"
echo "Run '$BINARY --version' to verify"
