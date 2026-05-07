param(
    [switch]$StopDocker,
    [switch]$KeepPidFile
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$pidFile = Join-Path $repoRoot ".dev-stack-pids.json"
$dockerName = "mh-pos-cloud-postgres"

function Stop-IfRunning([Nullable[int]]$ProcessId, [string]$Name) {
    if (-not $ProcessId) {
        return
    }
    try {
        $proc = Get-Process -Id $ProcessId -ErrorAction Stop
        Stop-Process -Id $ProcessId -Force -ErrorAction Stop
        Write-Host "Stopped $Name (PID=$ProcessId)"
    } catch {
        Write-Host "$Name (PID=$ProcessId) is not running"
    }
}

if (-not (Test-Path $pidFile)) {
    Write-Host "PID file not found: $pidFile"
    if ($StopDocker) {
        Write-Host "Stopping Docker container: $dockerName"
        docker stop $dockerName | Out-Null
    }
    Get-Process pos-edge -ErrorAction SilentlyContinue | Stop-Process -Force
    Get-Process cloud-api -ErrorAction SilentlyContinue | Stop-Process -Force
    Get-Process go -ErrorAction SilentlyContinue | Stop-Process -Force
    exit 0
}

$pids = Get-Content $pidFile | ConvertFrom-Json

Stop-IfRunning -ProcessId $pids.pos_ui_pid -Name "pos-ui"
Stop-IfRunning -ProcessId $pids.pos_backend_pid -Name "pos-backend"
Stop-IfRunning -ProcessId $pids.cloud_backend_pid -Name "cloud-backend"

if ($StopDocker) {
    Write-Host "Stopping Docker container: $dockerName"
    try {
        docker stop $dockerName | Out-Null
    } catch {
        Write-Host "Docker container $dockerName is not running or missing"
    }
}

if (-not $KeepPidFile) {
    Remove-Item $pidFile -Force
    Write-Host "Removed PID file: $pidFile"
} else {
    Write-Host "PID file kept: $pidFile"
}


Write-Host "Done."
