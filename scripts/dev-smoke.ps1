param(
  [string]$PosApiBase = "http://localhost:8080/api/v1",
  [string]$CloudApiBase = "http://localhost:8090"
)

$ErrorActionPreference = "Stop"

Write-Host "Checking Cloud health..."
Invoke-RestMethod -Uri "$CloudApiBase/health"

Write-Host "Checking POS health..."
Invoke-RestMethod -Uri ($PosApiBase -replace "/api/v1$", "/health")

Write-Host "Bootstrapping POS demo data..."
$bootstrap = & "$PSScriptRoot/bootstrap-pos-demo.ps1" -PosApiBase $PosApiBase

Write-Host "Checking POS sync endpoints..."
Invoke-RestMethod -Uri "$PosApiBase/sync/status"
Invoke-RestMethod -Uri "$PosApiBase/sync/local-events?limit=5"
Invoke-RestMethod -Uri "$PosApiBase/sync/outbox?limit=5"

Write-Host "Sending and replaying Cloud test envelope..."
& "$PSScriptRoot/send-cloud-test-envelope.ps1" -CloudApiBase $CloudApiBase -ReplayTwice

Write-Host "Smoke checks completed. Open POS UI and use pairing code:"
Write-Host $bootstrap.pairing_code
