# Routix CLI installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/ramusaaa/routix/main/install.ps1 | iex
#
# Or with a specific version:
# $env:ROUTIX_VERSION="v0.4.0"; irm https://raw.githubusercontent.com/ramusaaa/routix/main/install.ps1 | iex

param(
    [string]$Version = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Module  = "github.com/ramusaaa/routix/cmd/routix"
$Binary  = "routix"
$Default = "v0.4.0"

# ─── helpers ────────────────────────────────────────────────────────────────

function Write-Step  { Write-Host ":: $args" -ForegroundColor Cyan }
function Write-Ok    { Write-Host " ok $args" -ForegroundColor Green }
function Write-Warn  { Write-Host "warn $args" -ForegroundColor Yellow }
function Write-Fail  { Write-Host "err  $args" -ForegroundColor Red; exit 1 }

# ─── preflight ───────────────────────────────────────────────────────────────

Write-Step "Checking prerequisites..."

if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Fail "Go is not installed. Download it from https://go.dev/dl/ and re-run this script."
}

$goVersionOutput = & go version
if ($goVersionOutput -match "go(\d+)\.(\d+)") {
    $goMajor = [int]$Matches[1]
    $goMinor = [int]$Matches[2]
    if ($goMajor -lt 1 -or ($goMajor -eq 1 -and $goMinor -lt 21)) {
        Write-Fail "Go 1.21 or newer is required (found $goMajor.$goMinor). Update at https://go.dev/dl/"
    }
    Write-Ok "Go $goMajor.$goMinor"
} else {
    Write-Warn "Could not parse Go version. Continuing..."
}

# ─── resolve version ─────────────────────────────────────────────────────────

Write-Step "Resolving latest version..."

# Use env var override if set
if ($env:ROUTIX_VERSION) { $Version = $env:ROUTIX_VERSION }

if ([string]::IsNullOrEmpty($Version)) {
    try {
        $apiUrl  = "https://api.github.com/repos/ramusaaa/routix/releases/latest"
        $headers = @{ "User-Agent" = "routix-installer" }
        $release = Invoke-RestMethod -Uri $apiUrl -Headers $headers -TimeoutSec 10
        $Version = $release.tag_name
    } catch {
        Write-Warn "Could not fetch latest version from GitHub, using $Default"
        $Version = $Default
    }
}

Write-Ok "Installing $Version"

# ─── install ─────────────────────────────────────────────────────────────────

Write-Step "Running go install..."

$installTarget = "${Module}@${Version}"

try {
    & go install $installTarget
    if ($LASTEXITCODE -ne 0) { throw "go install exited with code $LASTEXITCODE" }
} catch {
    Write-Warn "Tagged version failed, trying @latest..."
    & go install "${Module}@latest"
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "Installation failed. Check your internet connection and try again."
    }
}

# ─── path setup ──────────────────────────────────────────────────────────────

Write-Step "Checking PATH..."

$GoPath = & go env GOPATH
if ([string]::IsNullOrEmpty($GoPath)) { $GoPath = "$env:USERPROFILE\go" }
$GoBin  = Join-Path $GoPath "bin"

$userPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")

if ($userPath -notlike "*$GoBin*") {
    Write-Step "Adding $GoBin to user PATH..."
    $newPath = "$userPath;$GoBin"
    [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    $env:PATH = "$env:PATH;$GoBin"
    Write-Ok "PATH updated"
} else {
    Write-Ok "$GoBin already in PATH"
}

# ─── verify ──────────────────────────────────────────────────────────────────

Write-Step "Verifying installation..."

$binaryPath = Join-Path $GoBin "$Binary.exe"

if (Test-Path $binaryPath) {
    Write-Ok "Installed at $binaryPath"
} else {
    Write-Fail "Installation failed. Binary not found at $binaryPath"
}

# ─── done ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  Routix is ready." -ForegroundColor White
Write-Host ""
Write-Host "  Get started:"
Write-Host ""
Write-Host "    routix new my-api"
Write-Host "    cd my-api"
Write-Host "    routix serve"
Write-Host ""
Write-Host "  Run 'routix help' to see all commands."
Write-Host ""
Write-Host "  Note: Open a new terminal window for PATH changes to take effect." -ForegroundColor Yellow
Write-Host ""
