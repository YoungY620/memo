# Memo MCP launcher for Windows — auto-installs memo if missing, then starts the MCP server.
#
# Usage in MCP config (Windows):
#   {
#     "mcpServers": {
#       "memo": {
#         "command": "powershell",
#         "args": ["-Command", "& { try { iex (irm 'https://raw.githubusercontent.com/YoungY620/memo/main/run-mcp.ps1') } catch { if (Get-Command memo -ErrorAction SilentlyContinue) { & memo mcp @args } else { throw 'memo not found and install failed' } } }"]
#       }
#     }
#   }
#
# How it works:
#   - All install output goes to stderr (stdout is reserved for MCP JSON-RPC)
#   - Invoke-WebRequest does not consume stdin, so no special handling needed
#   - If download fails but memo is already installed, falls back to local binary

$ErrorActionPreference = "Stop"

$Repo = "YoungY620/memo"
$InstallDir = "$env:USERPROFILE\.local\bin"
$MemoBin = "$InstallDir\memo.exe"

function Log($msg) { [Console]::Error.WriteLine($msg) }

function Find-Memo {
    # Check PATH first
    $inPath = Get-Command memo -ErrorAction SilentlyContinue
    if ($inPath) { return $inPath.Source }
    # Check install dir
    if (Test-Path $MemoBin) { return $MemoBin }
    return $null
}

function Install-Memo {
    Log "memo not found, installing..."

    try {
        $Response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $Latest = $Response.tag_name
    } catch {
        Log "Failed to fetch latest version from GitHub"
        return $false
    }

    $Url = "https://github.com/$Repo/releases/download/$Latest/memo-windows-amd64.exe"
    Log "Downloading memo $Latest for windows-amd64..."
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

    try {
        Invoke-WebRequest -Uri $Url -OutFile $MemoBin
    } catch {
        Log "Failed to download memo from $Url"
        return $false
    }

    Log "Installed memo to $MemoBin"
    return $true
}

# Try to find existing memo, install if missing
$memoPath = Find-Memo
if (-not $memoPath) {
    if (-not (Install-Memo)) {
        Log "ERROR: Could not install memo. Install manually: https://github.com/$Repo"
        exit 1
    }
    $memoPath = Find-Memo
    if (-not $memoPath) {
        Log "ERROR: memo installed but not found in PATH or $InstallDir"
        exit 1
    }
}

# Start memo mcp, passing through stdio
& $memoPath mcp @args
exit $LASTEXITCODE
