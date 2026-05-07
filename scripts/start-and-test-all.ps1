param(
    [switch]$SkipDocker,
    [switch]$SkipUI,
    [switch]$SkipBootstrap,
    [int]$StartupTimeoutSec = 120
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$cloudDir = Join-Path $repoRoot "cloud-backend"
$posDir = Join-Path $repoRoot "pos-backend"
$uiDir = Join-Path $repoRoot "pos-ui"
$bootstrapScript = Join-Path $repoRoot "scripts\bootstrap-pos-demo.ps1"
$pidFile = Join-Path $repoRoot ".dev-stack-pids.json"

$cloudDsn = "postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
$dockerName = "mh-pos-cloud-postgres"

function Write-Step([string]$Message) {
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Wait-HttpOk([string]$Url, [int]$TimeoutSec) {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $resp = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 3
            if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 300) {
                return $true
            }
        } catch {
        }
        Start-Sleep -Seconds 2
    }
    return $false
}

function Start-GoService([string]$Name, [string]$WorkDir, [string]$GoCmd, [hashtable]$EnvVars) {
    $exports = @()
    foreach ($k in $EnvVars.Keys) {
        $v = $EnvVars[$k].ToString().Replace("'", "''")
        $exports += "`$env:$k='$v'"
    }
    $prefix = ""
    if ($exports.Count -gt 0) {
        $prefix = ($exports -join "; ") + "; "
    }
    $script = $prefix + $GoCmd
    $proc = Start-Process -FilePath "powershell.exe" `
        -ArgumentList @("-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", $script) `
        -WorkingDirectory $WorkDir `
        -WindowStyle Normal `
        -PassThru
    Write-Host "Started $Name (PID=$($proc.Id))"
    return $proc
}

function Ensure-DockerPostgres() {
    Write-Step "Checking Docker PostgreSQL container: $dockerName"
    $exists = docker ps -a --filter "name=^${dockerName}$" --format "{{.Names}}" 2>$null
    if (-not $exists) {
        Write-Host "Container not found. Creating..."
        docker run --name $dockerName -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=mh_pos_cloud -p 5432:5432 -d postgres:16 | Out-Null
    } else {
        $running = docker ps --filter "name=^${dockerName}$" --format "{{.Names}}"
        if (-not $running) {
            Write-Host "Container found but stopped. Starting..."
            docker start $dockerName | Out-Null
        } else {
            Write-Host "Container is already running."
        }
    }
}

$started = [ordered]@{
    cloud_backend = $null
    pos_backend   = $null
    pos_ui        = $null
}

if (-not $SkipDocker) {
    Ensure-DockerPostgres
}

Write-Step "Starting cloud-backend"
$started.cloud_backend = Start-GoService `
    -Name "cloud-backend" `
    -WorkDir $cloudDir `
    -GoCmd "go run ./cmd/cloud-api" `
    -EnvVars @{ CLOUD_POSTGRES_DSN = $cloudDsn }

Write-Step "Waiting for cloud health endpoint"
if (-not (Wait-HttpOk -Url "http://localhost:8090/health" -TimeoutSec $StartupTimeoutSec)) {
    throw "cloud-backend did not become healthy in ${StartupTimeoutSec}s"
}

Write-Step "Starting pos-backend"
$started.pos_backend = Start-GoService `
    -Name "pos-backend" `
    -WorkDir $posDir `
    -GoCmd "go run ./cmd/pos-edge" `
    -EnvVars @{
        POS_DEV_TOOLS      = "1"
        POS_CLOUD_SYNC_URL = "http://localhost:8090/api/v1/sync/edge-events"
    }

Write-Step "Waiting for POS health endpoint"
if (-not (Wait-HttpOk -Url "http://localhost:8080/health" -TimeoutSec $StartupTimeoutSec)) {
    throw "pos-backend did not become healthy in ${StartupTimeoutSec}s"
}

if (-not $SkipUI) {
    Write-Step "Starting pos-ui"
    $started.pos_ui = Start-Process -FilePath "npm.cmd" `
        -ArgumentList @("run", "dev") `
        -WorkingDirectory $uiDir `
        -WindowStyle Normal `
        -PassThru
    Write-Host "Started pos-ui (PID=$($started.pos_ui.Id))"

    Write-Step "Waiting for UI endpoint"
    if (-not (Wait-HttpOk -Url "http://localhost:5173" -TimeoutSec $StartupTimeoutSec)) {
        throw "pos-ui did not become healthy in ${StartupTimeoutSec}s"
    }
}

if (-not $SkipBootstrap) {
    Write-Step "Running POS demo bootstrap"
    & $bootstrapScript | Out-Host
}

Write-Step "Running sync smoke checks"
Invoke-RestMethod http://localhost:8080/api/v1/sync/status | Out-Null
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=5 | Out-Null
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=5 | Out-Null

$pidPayload = @{
    cloud_backend_pid = if ($started.cloud_backend) { $started.cloud_backend.Id } else { $null }
    pos_backend_pid   = if ($started.pos_backend) { $started.pos_backend.Id } else { $null }
    pos_ui_pid        = if ($started.pos_ui) { $started.pos_ui.Id } else { $null }
    created_at        = (Get-Date).ToString("s")
} | ConvertTo-Json -Depth 3
$pidPayload | Set-Content -Path $pidFile -Encoding UTF8

Write-Host ""
Write-Host "Done. Services started and baseline checks passed." -ForegroundColor Green
Write-Host "Cloud health: http://localhost:8090/health"
Write-Host "POS health:   http://localhost:8080/health"
if (-not $SkipUI) {
    Write-Host "POS UI:       http://localhost:5173"
}
Write-Host "PID file:     $pidFile"
Write-Host ""
Write-Host "Stop example:"
Write-Host "  Get-Content $pidFile | ConvertFrom-Json"
Write-Host "  Stop-Process -Id <cloud_backend_pid>,<pos_backend_pid>,<pos_ui_pid> -Force"
