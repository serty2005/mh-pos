param(
    [string]$DatabasePath = "pos-backend\data\pos-edge.db",
    [switch]$Optimize,
    [switch]$WalCheckpoint,
    [switch]$Vacuum,
    [string]$VacuumInto,
    [switch]$Force
)

$ErrorActionPreference = "Stop"
$OutputEncoding = [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
$env:PYTHONIOENCODING = "utf-8"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$posDir = Join-Path $repoRoot "pos-backend"
$dbPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $DatabasePath))

if (-not (Test-Path -LiteralPath $dbPath)) {
    throw "SQLite база данных не найдена: $dbPath"
}

$argsList = @("run", "./cmd/sqlite-maintenance", "-db", $dbPath)
if ($Optimize) {
    $argsList += "-optimize"
}
if ($WalCheckpoint) {
    $argsList += "-wal-checkpoint"
}
if ($Vacuum) {
    $argsList += "-vacuum"
}
if ($VacuumInto) {
    $target = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $VacuumInto))
    if (Test-Path -LiteralPath $target) {
        throw "Целевой файл VACUUM INTO уже существует: $target"
    }
    $argsList += @("-vacuum-into", $target)
}
if ($Force) {
    $argsList += "-force"
}

Write-Host "Целевая SQLite база: $dbPath"
Write-Host "Операции: $($argsList -join ' ')"
Write-Host "VACUUM/VACUUM INTO выполняются только как явные maintenance-операции и не входят в обычный POS write flow."

Push-Location $posDir
try {
    & go @argsList
} finally {
    Pop-Location
}
