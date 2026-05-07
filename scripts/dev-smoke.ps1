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

Write-Host "Pairing code: $($bootstrap.pairing_code)"
Write-Host "Cashier PIN:  $($bootstrap.cashier_pin)"

Write-Host "Checking POS sync status..."
$syncStatus = Invoke-RestMethod -Uri "$PosApiBase/sync/status"
$syncStatus

Write-Host "Checking POS local events..."
$localEvents = Invoke-RestMethod -Uri "$PosApiBase/sync/local-events?limit=10"
$localEvents

Write-Host "Checking POS outbox..."
$outbox = Invoke-RestMethod -Uri "$PosApiBase/sync/outbox?limit=10"
$outbox

Write-Host "Sending and replaying Cloud test envelope..."
& "$PSScriptRoot/send-cloud-test-envelope.ps1" `
  -CloudApiBase $CloudApiBase `
  -RestaurantId $bootstrap.restaurant_id `
  -NodeDeviceId $bootstrap.node_device_id `
  -ReplayTwice

Write-Host "Smoke checks completed."
Write-Host "Use this pairing code in POS UI /pair: $($bootstrap.pairing_code)"
Write-Host "Use this cashier PIN on /login: $($bootstrap.cashier_pin)"

$bootstrap
