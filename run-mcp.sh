#!/bin/sh
# Memo MCP launcher — auto-installs memo if missing, then starts the MCP server.
#
# Usage in MCP config (Linux/macOS):
#   {
#     "mcpServers": {
#       "memo": {
#         "command": "sh",
#         "args": ["-c", "eval \"$(curl -fsSL https://raw.githubusercontent.com/YoungY620/memo/main/run-mcp.sh 2>/dev/null || echo '')\""]
#       }
#     }
#   }
#
# How it works:
#   - All install output goes to stderr (stdout is reserved for MCP JSON-RPC)
#   - Uses eval "$(curl ...)" so stdin is NOT consumed (unlike curl | sh)
#   - exec replaces this shell process, preserving stdio for MCP
#
# Fallback behavior:
#   - memo already installed      -> starts immediately
#   - GitHub down, memo installed  -> curl fails silently, local binary starts
#   - GitHub down, memo missing    -> exit 1 with clear error

REPO="YoungY620/memo"
INSTALL_DIR="$HOME/.local/bin"
MEMO_BIN="$INSTALL_DIR/memo"

# All output to stderr — stdout is for MCP JSON-RPC
log() { echo "$@" >&2; }

# Check if memo is already in PATH or install dir
find_memo() {
    if command -v memo >/dev/null 2>&1; then
        command -v memo
        return 0
    fi
    if [ -x "$MEMO_BIN" ]; then
        echo "$MEMO_BIN"
        return 0
    fi
    return 1
}

install_memo() {
    log "memo not found, installing..."

    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) log "Unsupported architecture: $ARCH"; return 1 ;;
    esac
    PLATFORM="${OS}-${ARCH}"

    LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$LATEST" ]; then
        log "Failed to fetch latest version from GitHub"
        return 1
    fi

    URL="https://github.com/$REPO/releases/download/$LATEST/memo-$PLATFORM"
    log "Downloading memo $LATEST for $PLATFORM..."
    mkdir -p "$INSTALL_DIR"
    if ! curl -fsSL "$URL" -o "$MEMO_BIN" 2>/dev/null; then
        log "Failed to download memo from $URL"
        return 1
    fi
    chmod +x "$MEMO_BIN"
    log "Installed memo to $MEMO_BIN"
}

# Try to find existing memo, install if missing
MEMO_PATH=$(find_memo) || {
    install_memo || {
        log "ERROR: Could not install memo. Install manually: https://github.com/$REPO"
        exit 1
    }
    MEMO_PATH=$(find_memo) || {
        log "ERROR: memo installed but not found in PATH or $INSTALL_DIR"
        exit 1
    }
}

# Replace this process with memo mcp, preserving stdio
exec "$MEMO_PATH" mcp "$@"
