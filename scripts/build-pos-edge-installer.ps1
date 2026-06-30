param(
  [string]$Version = "0.1.9",
  [ValidateSet("amd64", "386")]
  [string]$Arch = "amd64",
  [ValidateRange(1, 65535)]
  [int]$PosHttpPort = 8080,
  [string]$LicenseServerUrl = "https://license.example.com",
  [string]$CloudSyncUrl = "https://cloud.example.com",
  [string]$OutDir,
  [string]$WebWallpaperExe = ""
)

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
if (-not $OutDir) {
  $OutDir = Join-Path $Root "dist\pos-edge-installer"
}
if ($WebWallpaperExe -and -not (Test-Path -LiteralPath $WebWallpaperExe)) {
  throw "WebWallpaperExe was not found: $WebWallpaperExe"
}

function Require-Command($Name) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "$Name is required in PATH"
  }
}

function Write-Utf8NoBom($Path, $Content) {
  $Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText($Path, $Content, $Utf8NoBom)
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
$Config.POS_HTTP_ADDR = "127.0.0.1:$PosHttpPort"
$Config.POS_SQLITE_PATH = "data/pos-edge.db"
$Config.POS_SQLITE_MIGRATIONS_DIR = "migrations/sqlite"
$Config.POS_SQLITE_BACKUP_DIR = "data/backups"
$Config.POS_SQLITE_ARCHIVE_DIR = "data/archives"
$Config.POS_UI_DIST_DIR = "ui/pos-ui"
Write-Utf8NoBom (Join-Path $Stage "config\pos-edge.install.json") ($Config | ConvertTo-Json -Depth 8)
Copy-Item (Join-Path $Root "installer\windows\apply-pos-edge-settings.ps1") (Join-Path $Stage "config\apply-pos-edge-settings.ps1")

Copy-Item -Recurse (Join-Path $Root "pos-backend\migrations\sqlite") (Join-Path $Stage "migrations\sqlite")
Copy-Item -Recurse (Join-Path $Root "pos-ui-g\dist") (Join-Path $Stage "ui\pos-ui")

@"
@echo off
cd /d "%~dp0"
set POS_CONFIG_PATH=config\pos-edge.json
pos-edge.exe
"@ | Set-Content -Encoding ASCII (Join-Path $Stage "start-pos-edge.cmd")

if ($WebWallpaperExe) {
  Copy-Item -LiteralPath $WebWallpaperExe -Destination (Join-Path $Stage "webwallpaper\gowebwallpaper.exe")
  $WebWallpaperExample = @"
{
  "URL": "http://127.0.0.1:$PosHttpPort/",
  "Monitors": [],
  "Audio": {
    "ID": "",
    "Name": "",
    "Active": false
  }
}
"@
  Write-Utf8NoBom (Join-Path $Stage "webwallpaper\config.pos-edge.example.json") $WebWallpaperExample
}

makensis `
  "/DAPP_VERSION=$Version" `
  "/DAPP_ARCH=$Arch" `
  "/DDEFAULT_POS_HTTP_PORT=$PosHttpPort" `
  "/DDEFAULT_LICENSE_SERVER_URL=$LicenseServerUrl" `
  "/DDEFAULT_CLOUD_SYNC_URL=$CloudSyncUrl" `
  "/DSTAGE_DIR=$Stage" `
  "/DOUT_FILE=$Installer" `
  (Join-Path $Root "installer\nsis\pos-edge.nsi")

Write-Host "Installer written to $Installer"
