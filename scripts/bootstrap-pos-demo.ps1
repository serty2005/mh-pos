param(
  [Alias("CloudApiBase")]
  [string]$CloudBaseUrl = "http://localhost:8090",
  [Alias("PosApiBase", "EdgeApiBase")]
  [string]$EdgeBaseUrl = "http://localhost:8080",
  [string]$UiBaseUrl = "http://localhost:5173",
  [string]$RestaurantName = "",
  [string]$NodeDeviceId = "",
  [string]$CashierPin = "1111",
  [string]$ManagerPin = "2222",
  [switch]$RunRuntimeSmoke,
  [switch]$VerboseOutput
)

$ErrorActionPreference = "Stop"

Write-Warning "scripts/bootstrap-pos-demo.ps1 is deprecated. It now delegates to production-way Cloud -> Edge provisioning and does not use POS Edge dev bootstrap."

& "$PSScriptRoot/bootstrap-production-way.ps1" `
  -CloudBaseUrl $CloudBaseUrl `
  -EdgeBaseUrl $EdgeBaseUrl `
  -UiBaseUrl $UiBaseUrl `
  -RestaurantName $RestaurantName `
  -NodeDeviceId $NodeDeviceId `
  -CashierPin $CashierPin `
  -ManagerPin $ManagerPin `
  -RunRuntimeSmoke:$RunRuntimeSmoke `
  -VerboseOutput:$VerboseOutput
