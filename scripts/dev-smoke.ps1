param(
  [Alias("PosApiBase", "EdgeApiBase")]
  [string]$EdgeBaseUrl = "http://localhost:8080",
  [Alias("CloudApiBase")]
  [string]$CloudBaseUrl = "http://localhost:8090",
  [string]$UiBaseUrl = "http://localhost:5173",
  [switch]$VerboseOutput
)

$ErrorActionPreference = "Stop"

Write-Host "Checking Cloud health..."
Invoke-RestMethod -Uri "$CloudBaseUrl/health" | Out-Null

Write-Host "Checking POS health..."
Invoke-RestMethod -Uri "$EdgeBaseUrl/health" | Out-Null

Write-Host "Running production-way bootstrap and runtime smoke..."
$bootstrap = & "$PSScriptRoot/bootstrap-production-way.ps1" `
  -CloudBaseUrl $CloudBaseUrl `
  -EdgeBaseUrl $EdgeBaseUrl `
  -UiBaseUrl $UiBaseUrl `
  -RunRuntimeSmoke `
  -VerboseOutput:$VerboseOutput

Write-Host "Sending and replaying Cloud test envelope..."
& "$PSScriptRoot/send-cloud-test-envelope.ps1" `
  -CloudApiBase $CloudBaseUrl `
  -RestaurantId $bootstrap.restaurant_id `
  -NodeDeviceId $bootstrap.node_device_id `
  -ReplayTwice

Write-Host "Smoke checks completed."
Write-Host "Use POS UI: $($bootstrap.ui_base_url)"
Write-Host "Use cashier PIN on /login: $($bootstrap.cashier_pin)"
Write-Host "Use manager PIN for refund/manager flows: $($bootstrap.manager_pin)"

$bootstrap
