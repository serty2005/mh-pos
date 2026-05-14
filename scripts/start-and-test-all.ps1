param(
    [switch]$SkipDocker,
    [switch]$SkipUI,
    [switch]$SkipLicense,
    [switch]$SkipBootstrap,
    [switch]$PreserveLocalData,
    [int]$StartupTimeoutSec = 120
)

$ErrorActionPreference = "Stop"
$OutputEncoding = [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
$env:PYTHONIOENCODING = "utf-8"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$cloudDir = Join-Path $repoRoot "cloud-backend"
$posDir = Join-Path $repoRoot "pos-backend"
$uiDir = Join-Path $repoRoot "pos-ui"
$licenseDir = Join-Path $repoRoot "license-server"
$bootstrapScript = Join-Path $repoRoot "scripts\bootstrap-production-way.ps1"
$pidFile = Join-Path $repoRoot ".dev-stack-pids.json"
$logsDir = Join-Path $repoRoot "logs\dev-stack"

$cloudDsn = "postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
$dockerName = "mh-pos-cloud-postgres"

function Write-Step([string]$Message) {
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Test-PortOpen([int]$Port) {
    $client = New-Object System.Net.Sockets.TcpClient
    try {
        $async = $client.BeginConnect("127.0.0.1", $Port, $null, $null)
        if (-not $async.AsyncWaitHandle.WaitOne(300)) {
            return $false
        }
        $client.EndConnect($async)
        return $true
    } catch {
        return $false
    } finally {
        $client.Close()
    }
}

function Assert-PortFree([int]$Port, [string]$Name) {
    if (Test-PortOpen -Port $Port) {
        throw "${Name}: порт $Port уже занят. Останови текущий процесс или выполни scripts\stop-and-test-all.ps1 перед запуском нового стека."
    }
}

function Remove-LocalSqlite([string]$BasePath, [string]$Name) {
    $resolvedParent = Resolve-Path -LiteralPath (Split-Path -Parent $BasePath) -ErrorAction SilentlyContinue
    if (-not $resolvedParent) {
        return
    }
    $repoFull = [System.IO.Path]::GetFullPath($repoRoot)
    $parentFull = [System.IO.Path]::GetFullPath($resolvedParent.Path)
    if (-not $parentFull.StartsWith($repoFull, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove ${Name} SQLite outside repo: $parentFull"
    }
    foreach ($path in @($BasePath, "$BasePath-wal", "$BasePath-shm")) {
        if (Test-Path -LiteralPath $path) {
            Remove-Item -LiteralPath $path -Force
            Write-Host "Удален ${Name} SQLite файл: $path"
        }
    }
}

function Show-LogTail([string]$Name, [string]$LogPath) {
    Write-Host "Последние строки лога для ${Name}: $LogPath" -ForegroundColor Yellow
    if (Test-Path -LiteralPath $LogPath) {
        Get-Content -LiteralPath $LogPath -Tail 80 -Encoding UTF8 | ForEach-Object { Write-Host $_ }
    } else {
        Write-Host "Файл лога пока не создан."
    }
}

function Wait-HttpOk([string]$Url, [int]$TimeoutSec, [string]$Name, [string]$LogPath) {
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
    Show-LogTail -Name $Name -LogPath $LogPath
    return $false
}

function Start-ServiceWindow([string]$Name, [string]$WorkDir, [string]$Command, [hashtable]$EnvVars, [string]$LogPath) {
    $exports = @(
        '$OutputEncoding = [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()',
        '$env:PYTHONIOENCODING = "utf-8"'
    )
    foreach ($k in $EnvVars.Keys) {
        $v = $EnvVars[$k].ToString().Replace("'", "''")
        $exports += ('$env:' + $k + "='$v'")
    }
    $escapedLog = $LogPath.Replace("'", "''")
    $script = ($exports -join "; ") + "; " + $Command + " 2>&1 | Tee-Object -FilePath '$escapedLog'"
    $proc = Start-Process -FilePath "powershell.exe" `
        -ArgumentList @("-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", $script) `
        -WorkingDirectory $WorkDir `
        -WindowStyle Hidden `
        -PassThru
    Write-Host "Запущен $Name (PID=$($proc.Id), log=$LogPath)"
    return $proc
}

function Ensure-DockerPostgres() {
    Write-Step "Проверяю Docker PostgreSQL container: $dockerName"
    $exists = docker ps -a --filter "name=^${dockerName}$" --format "{{.Names}}" 2>$null
    if (-not $exists) {
        Write-Host "Container не найден. Создаю..."
        docker run --name $dockerName -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=mh_pos_cloud -p 5432:5432 -d postgres:16 | Out-Null
    } else {
        $running = docker ps --filter "name=^${dockerName}$" --format "{{.Names}}"
        if (-not $running) {
            Write-Host "Container найден, но остановлен. Запускаю..."
            docker start $dockerName | Out-Null
        } else {
            Write-Host "Container уже запущен."
        }
    }
}

function Wait-DockerPostgresReady([int]$TimeoutSec) {
    Write-Step "Жду готовность Docker PostgreSQL"
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        docker exec $dockerName pg_isready -U postgres -d mh_pos_cloud 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            return
        }
        Start-Sleep -Seconds 2
    }
    throw "Docker PostgreSQL не перешел в ready за ${TimeoutSec}s"
}

function Stop-StartedProcess([object]$Process, [string]$Name) {
    if (-not $Process) {
        return
    }
    try {
        Get-Process -Id $Process.Id -ErrorAction Stop | Out-Null
        taskkill.exe /PID $Process.Id /T /F | Out-Null
        Write-Host "Остановлен $Name после ошибки (PID=$($Process.Id))" -ForegroundColor Yellow
    } catch {
    }
}

if (Test-Path -LiteralPath $pidFile) {
    throw "PID file уже существует: $pidFile. Выполни scripts\stop-and-test-all.ps1 перед запуском нового локального стека."
}

New-Item -ItemType Directory -Force -Path $logsDir | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $posDir "data") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $licenseDir "data") | Out-Null

Assert-PortFree -Port 8090 -Name "cloud-backend"
Assert-PortFree -Port 8080 -Name "pos-backend"
if (-not $SkipLicense) {
    Assert-PortFree -Port 8095 -Name "license-server"
}
if (-not $SkipUI) {
    Assert-PortFree -Port 5173 -Name "pos-ui"
}

$cloudLog = Join-Path $logsDir "cloud-backend.log"
$posLog = Join-Path $logsDir "pos-backend.log"
$uiLog = Join-Path $logsDir "pos-ui.log"
$licenseLog = Join-Path $logsDir "license-server.log"

$started = [ordered]@{
    cloud_backend = $null
    license_server = $null
    pos_backend   = $null
    pos_ui        = $null
}

if (-not $SkipDocker) {
    Ensure-DockerPostgres
    Wait-DockerPostgresReady -TimeoutSec $StartupTimeoutSec
}

if (-not $PreserveLocalData) {
    Write-Step "Очищаю локальные SQLite БД dev stack"
    Remove-LocalSqlite -BasePath (Join-Path $posDir "data\pos-edge.db") -Name "POS Edge"
    Remove-LocalSqlite -BasePath (Join-Path $licenseDir "data\license-server.db") -Name "License Server"
}

try {
if (-not $SkipLicense) {
    Write-Step "Запускаю license-server"
    $started.license_server = Start-ServiceWindow `
        -Name "license-server" `
        -WorkDir $licenseDir `
        -Command "go run ./cmd/license-api" `
        -EnvVars @{} `
        -LogPath $licenseLog

    Write-Step "Жду health endpoint license-server"
    if (-not (Wait-HttpOk -Url "http://localhost:8095/health" -TimeoutSec $StartupTimeoutSec -Name "license-server" -LogPath $licenseLog)) {
        throw "license-server не перешел в healthy за ${StartupTimeoutSec}s"
    }
}

Write-Step "Запускаю cloud-backend"
$started.cloud_backend = Start-ServiceWindow `
    -Name "cloud-backend" `
    -WorkDir $cloudDir `
    -Command "go run ./cmd/cloud-api" `
    -EnvVars @{
        CLOUD_POSTGRES_DSN = $cloudDsn
        CLOUD_PUBLIC_URL   = "http://localhost:8090"
        LICENSE_SERVER_URL = if ($SkipLicense) { "" } else { "http://localhost:8095" }
    } `
    -LogPath $cloudLog

Write-Step "Жду health endpoint cloud-backend"
if (-not (Wait-HttpOk -Url "http://localhost:8090/health" -TimeoutSec $StartupTimeoutSec -Name "cloud-backend" -LogPath $cloudLog)) {
    throw "cloud-backend не перешел в healthy за ${StartupTimeoutSec}s"
}

Write-Step "Запускаю pos-backend"
$started.pos_backend = Start-ServiceWindow `
    -Name "pos-backend" `
    -WorkDir $posDir `
    -Command "go run ./cmd/pos-edge" `
    -EnvVars @{
        POS_CLOUD_SYNC_URL = "http://localhost:8090"
        LICENSE_SERVER_URL = if ($SkipLicense) { "" } else { "http://localhost:8095" }
    } `
    -LogPath $posLog

Write-Step "Жду health endpoint POS"
if (-not (Wait-HttpOk -Url "http://localhost:8080/health" -TimeoutSec $StartupTimeoutSec -Name "pos-backend" -LogPath $posLog)) {
    throw "pos-backend не перешел в healthy за ${StartupTimeoutSec}s"
}

if (-not $SkipUI) {
    Write-Step "Запускаю pos-ui"
    $started.pos_ui = Start-ServiceWindow `
        -Name "pos-ui" `
        -WorkDir $uiDir `
        -Command "npm.cmd run dev" `
        -EnvVars @{} `
        -LogPath $uiLog

    Write-Step "Жду UI endpoint"
    if (-not (Wait-HttpOk -Url "http://localhost:5173" -TimeoutSec $StartupTimeoutSec -Name "pos-ui" -LogPath $uiLog)) {
        throw "pos-ui не перешел в healthy за ${StartupTimeoutSec}s"
    }
}

$bootstrap = $null
if (-not $SkipBootstrap) {
    Write-Step "Выполняю production-way Cloud -> Edge bootstrap"
    $bootstrap = & $bootstrapScript -RunRuntimeSmoke
    $bootstrap | Out-Host
}

if ($bootstrap) {
    Write-Step "Выполняю authenticated POS sync smoke checks"
    $clientDeviceId = "dev-smoke-client"
    $loginBody = @{
        node_device_id   = $bootstrap.node_device_id
        client_device_id = $clientDeviceId
        pin              = $bootstrap.manager_pin
    } | ConvertTo-Json
    $login = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/auth/pin-login" -ContentType "application/json" -Body $loginBody
    $headers = @{
        "X-Node-Device-ID"     = $bootstrap.node_device_id
        "X-Client-Device-ID"   = $clientDeviceId
        "X-Session-ID"         = $login.session.id
        "X-Actor-Employee-ID"  = $login.actor.employee_id
    }
    Invoke-RestMethod -Uri "http://localhost:8080/api/v1/sync/status" -Headers $headers | Out-Null
    Invoke-RestMethod -Uri "http://localhost:8080/api/v1/sync/local-events?limit=5" -Headers $headers | Out-Null
    Invoke-RestMethod -Uri "http://localhost:8080/api/v1/sync/outbox?limit=5" -Headers $headers | Out-Null
}

$pidPayload = @{
    cloud_backend_pid = if ($started.cloud_backend) { $started.cloud_backend.Id } else { $null }
    license_server_pid = if ($started.license_server) { $started.license_server.Id } else { $null }
    pos_backend_pid   = if ($started.pos_backend) { $started.pos_backend.Id } else { $null }
    pos_ui_pid        = if ($started.pos_ui) { $started.pos_ui.Id } else { $null }
    logs_dir          = $logsDir
    created_at        = (Get-Date).ToString("s")
} | ConvertTo-Json -Depth 3
$pidPayload | Set-Content -Path $pidFile -Encoding UTF8

Write-Host ""
Write-Host "Готово. Сервисы запущены, базовые проверки прошли." -ForegroundColor Green
Write-Host "Cloud health: http://localhost:8090/health"
if (-not $SkipLicense) {
    Write-Host "License health: http://localhost:8095/health"
}
Write-Host "POS health:   http://localhost:8080/health"
if (-not $SkipUI) {
    Write-Host "POS UI:       http://localhost:5173"
}
Write-Host "Логи:         $logsDir"
Write-Host "PID file:     $pidFile"
Write-Host "Остановка:"
Write-Host '  powershell -ExecutionPolicy Bypass -File .\scripts\stop-and-test-all.ps1'
} catch {
    Stop-StartedProcess -Process $started.pos_ui -Name "pos-ui"
    Stop-StartedProcess -Process $started.pos_backend -Name "pos-backend"
    Stop-StartedProcess -Process $started.cloud_backend -Name "cloud-backend"
    Stop-StartedProcess -Process $started.license_server -Name "license-server"
    throw
}
