param(
  [Parameter(Mandatory = $true)]
  [string]$ConfigPath,
  [Parameter(Mandatory = $true)]
  [string]$PresetPath,
  [Parameter(Mandatory = $true)]
  [string]$Version,
  [Parameter(Mandatory = $true)]
  [ValidateRange(1, 65535)]
  [int]$PosHttpPort,
  [Parameter(Mandatory = $true)]
  [string]$CloudSyncUrl,
  [Parameter(Mandatory = $true)]
  [string]$LicenseServerUrl,
  [string]$WebWallpaperConfigPath = ""
)

$ErrorActionPreference = "Stop"

function Read-JsonConfig($Path) {
  if (-not (Test-Path -LiteralPath $Path)) {
    return [pscustomobject]@{}
  }
  $Raw = Get-Content -LiteralPath $Path -Raw -Encoding UTF8
  if ([string]::IsNullOrWhiteSpace($Raw)) {
    return [pscustomobject]@{}
  }
  return $Raw | ConvertFrom-Json
}

function Set-JsonProperty($Object, $Name, $Value) {
  if ($Object.PSObject.Properties.Name -contains $Name) {
    $Object.$Name = $Value
    return
  }
  $Object | Add-Member -NotePropertyName $Name -NotePropertyValue $Value
}

function Write-Utf8NoBom($Path, $Content) {
  $Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText($Path, $Content, $Utf8NoBom)
}

if ([string]::IsNullOrWhiteSpace($CloudSyncUrl)) {
  throw "CloudSyncUrl must not be empty"
}
if ([string]::IsNullOrWhiteSpace($LicenseServerUrl)) {
  throw "LicenseServerUrl must not be empty"
}

$SourcePath = $ConfigPath
if (-not (Test-Path -LiteralPath $SourcePath)) {
  $SourcePath = $PresetPath
}

$Config = Read-JsonConfig $SourcePath
Set-JsonProperty $Config "MH_POS_VERSION" $Version
Set-JsonProperty $Config "POS_HTTP_ADDR" "127.0.0.1:$PosHttpPort"
Set-JsonProperty $Config "POS_CLOUD_SYNC_URL" $CloudSyncUrl
Set-JsonProperty $Config "LICENSE_SERVER_URL" $LicenseServerUrl
Set-JsonProperty $Config "POS_SQLITE_PATH" "data/pos-edge.db"
Set-JsonProperty $Config "POS_SQLITE_MIGRATIONS_DIR" "migrations/sqlite"
Set-JsonProperty $Config "POS_SQLITE_BACKUP_DIR" "data/backups"
Set-JsonProperty $Config "POS_SQLITE_ARCHIVE_DIR" "data/archives"
Set-JsonProperty $Config "POS_UI_DIST_DIR" "ui/pos-ui"

$ConfigDir = Split-Path -Parent $ConfigPath
New-Item -ItemType Directory -Force $ConfigDir | Out-Null
$Json = $Config | ConvertTo-Json -Depth 8
Write-Utf8NoBom $ConfigPath $Json

if (-not [string]::IsNullOrWhiteSpace($WebWallpaperConfigPath)) {
  $WebWallpaperConfig = [pscustomobject]@{
    URL = "http://127.0.0.1:$PosHttpPort/"
    Monitors = @()
    Audio = [pscustomobject]@{
      ID = ""
      Name = ""
      Active = $false
    }
  }
  $WebWallpaperDir = Split-Path -Parent $WebWallpaperConfigPath
  New-Item -ItemType Directory -Force $WebWallpaperDir | Out-Null
  $WebWallpaperJson = $WebWallpaperConfig | ConvertTo-Json -Depth 8
  Write-Utf8NoBom $WebWallpaperConfigPath $WebWallpaperJson
}
