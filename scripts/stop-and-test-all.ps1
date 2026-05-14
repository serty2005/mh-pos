param(
    [switch]$StopDocker,
    [switch]$KeepPidFile
)

$ErrorActionPreference = "Stop"
$OutputEncoding = [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
$env:PYTHONIOENCODING = "utf-8"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$pidFile = Join-Path $repoRoot ".dev-stack-pids.json"
$dockerName = "mh-pos-cloud-postgres"

function Stop-IfRunning([Nullable[int]]$ProcessId, [string]$Name) {
    if (-not $ProcessId) {
        return
    }
    try {
        Get-Process -Id $ProcessId -ErrorAction Stop | Out-Null
        taskkill.exe /PID $ProcessId /T /F | Out-Null
        Write-Host "Остановлен $Name (PID=$ProcessId)"
    } catch {
        Write-Host "$Name (PID=$ProcessId) уже не запущен"
    }
}

if (-not (Test-Path $pidFile)) {
    Write-Host "PID file не найден: $pidFile"
    if ($StopDocker) {
        Write-Host "Останавливаю Docker container: $dockerName"
        docker stop $dockerName | Out-Null
    }
    Write-Host "Без PID file нет безопасно известного process tree. При необходимости останови вручную по порту/процессу."
    exit 0
}

$pids = Get-Content $pidFile | ConvertFrom-Json

Stop-IfRunning -ProcessId $pids.pos_ui_pid -Name "pos-ui"
Stop-IfRunning -ProcessId $pids.pos_backend_pid -Name "pos-backend"
Stop-IfRunning -ProcessId $pids.cloud_backend_pid -Name "cloud-backend"
Stop-IfRunning -ProcessId $pids.license_server_pid -Name "license-server"

if ($StopDocker) {
    Write-Host "Останавливаю Docker container: $dockerName"
    try {
        docker stop $dockerName | Out-Null
    } catch {
        Write-Host "Docker container $dockerName не запущен или отсутствует"
    }
}

if (-not $KeepPidFile) {
    Remove-Item $pidFile -Force
    Write-Host "Удален PID file: $pidFile"
} else {
    Write-Host "PID file сохранен: $pidFile"
}


Write-Host "Готово."
