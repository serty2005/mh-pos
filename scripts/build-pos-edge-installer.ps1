param(
  [string]$Version = "0.1.9",
  [ValidateSet("amd64", "386")]
  [string]$Arch = "amd64",
  [Parameter(Mandatory = $true)]
  [string]$LicenseServerUrl,
  [Parameter(Mandatory = $true)]
  [string]$CloudSyncUrl,
  [string]$OutDir,
  [string]$WebWallpaperExe = ""
)

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
if (-not $OutDir) {
  $OutDir = Join-Path $Root "dist\pos-edge-installer"
}

function Require-Command($Name) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "$Name is required in PATH"
  }
}

Require-Command "go"
Require-Command "npm"
Require-Command "makensis"

$Stage = Join-Path $OutDir "stage-$Arch"
$Installer = Join-Path $OutDir "myhoreca-pos-edge-$Version-$Arch-setup.exe"
Remove-Item -Recurse -Force $Stage -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $Stage | Out-Null

Push-Location (Join-Path $Root "pos-ui-g")
try {
  npm install
  $env:VITE_POS_API_BASE = "/api/v1"
  npm run build
} finally {
  Remove-Item Env:\VITE_POS_API_BASE -ErrorAction SilentlyContinue
  Pop-Location
}

Push-Location (Join-Path $Root "pos-backend")
try {
  $env:CGO_ENABLED = "0"
  $env:GOOS = "windows"
  $env:GOARCH = $Arch
  go build -trimpath -ldflags="-s -w" -o (Join-Path $Stage "pos-edge.exe") ./cmd/pos-edge
} finally {
  Remove-Item Env:\CGO_ENABLED, Env:\GOOS, Env:\GOARCH -ErrorAction SilentlyContinue
  Pop-Location
}

New-Item -ItemType Directory -Force `
  (Join-Path $Stage "config"), `
  (Join-Path $Stage "migrations"), `
  (Join-Path $Stage "ui"), `
  (Join-Path $Stage "webwallpaper") | Out-Null

$Config = Get-Content (Join-Path $Root "pos-backend\config\pos-edge.windows.json") -Raw | ConvertFrom-Json
$Config.MH_POS_VERSION = $Version
$Config.LICENSE_SERVER_URL = $LicenseServerUrl
$Config.POS_CLOUD_SYNC_URL = $CloudSyncUrl
$Config.POS_HTTP_ADDR = "127.0.0.1:8080"
$Config.POS_SQLITE_PATH = "data/pos-edge.db"
$Config.POS_SQLITE_MIGRATIONS_DIR = "migrations/sqlite"
$Config.POS_SQLITE_BACKUP_DIR = "data/backups"
$Config.POS_SQLITE_ARCHIVE_DIR = "data/archives"
$Config.POS_UI_DIST_DIR = "ui/pos-ui"
$Config | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 (Join-Path $Stage "config\pos-edge.install.json")

Copy-Item -Recurse (Join-Path $Root "pos-backend\migrations\sqlite") (Join-Path $Stage "migrations\sqlite")
Copy-Item -Recurse (Join-Path $Root "pos-ui-g\dist") (Join-Path $Stage "ui\pos-ui")

@"
@echo off
cd /d "%~dp0"
set POS_CONFIG_PATH=config\pos-edge.json
pos-edge.exe
"@ | Set-Content -Encoding ASCII (Join-Path $Stage "start-pos-edge.cmd")

if ($WebWallpaperExe) {
  Copy-Item $WebWallpaperExe (Join-Path $Stage "webwallpaper\gowebwallpaper.exe")
  @"
{
  "URL": "http://127.0.0.1:8080/",
  "Monitors": [],
  "Audio": {
    "ID": "",
    "Name": "",
    "Active": false
  }
}
"@ | Set-Content -Encoding UTF8 (Join-Path $Stage "webwallpaper\config.pos-edge.example.json")
}

makensis `
  "/DAPP_VERSION=$Version" `
  "/DAPP_ARCH=$Arch" `
  "/DSTAGE_DIR=$Stage" `
  "/DOUT_FILE=$Installer" `
  (Join-Path $Root "installer\nsis\pos-edge.nsi")

Write-Host "Installer written to $Installer"
