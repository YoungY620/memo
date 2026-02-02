# Memo installer for Windows
# Usage: irm https://raw.githubusercontent.com/YoungY620/memo/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "YoungY620/memo"
$InstallDir = "$env:USERPROFILE\.local\bin"

Write-Host "Platform: windows-amd64"

# Get latest version
$Response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Latest = $Response.tag_name
Write-Host "Latest version: $Latest"

# Check current version
$ExePath = "$InstallDir\memo.exe"
if (Test-Path $ExePath) {
    $Output = & $ExePath --version 2>&1
    if ($Output -match "memo\s+(.+)") {
        $Current = $Matches[1].Trim()
        $NormCurrent = $Current -replace "^v", ""
        $NormLatest = $Latest -replace "^v", ""
        if ($NormCurrent -eq $NormLatest) {
            Write-Host "Already up to date: $Latest"
            exit 0
        }
        Write-Host "Current version: $Current"
    }
}

# Download and install
$Url = "https://github.com/$Repo/releases/download/$Latest/memo-windows-amd64.exe"
Write-Host "Downloading: $Url"
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Invoke-WebRequest -Uri $Url -OutFile $ExePath
Write-Host "Installed: $ExePath"

# Check PATH
if (-not ($env:PATH -split ";" | Where-Object { $_ -eq $InstallDir })) {
    Write-Host "Note: Add to PATH: `$env:PATH += `";$InstallDir`""
}
