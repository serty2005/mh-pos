param(
  [Alias("CloudApiBase")]
  [string]$CloudBaseUrl = "http://localhost:8090",
  [Alias("PosApiBase", "EdgeApiBase")]
  [string]$EdgeBaseUrl = "http://localhost:8080",
  [string]$UiBaseUrl = "http://localhost:5173",
  [string]$NodeDeviceId = "",
  [string]$CashierPin = "1111",
  [string]$ManagerPin = "2222",
  [switch]$RunRuntimeSmoke,
  [switch]$VerboseOutput
)

$ErrorActionPreference = "Stop"

& "$PSScriptRoot/bootstrap-production-way.ps1" `
  -CloudBaseUrl $CloudBaseUrl `
  -EdgeBaseUrl $EdgeBaseUrl `
  -UiBaseUrl $UiBaseUrl `
  -NodeDeviceId $NodeDeviceId `
  -CashierPin $CashierPin `
  -ManagerPin $ManagerPin `
  -RunRuntimeSmoke:$RunRuntimeSmoke `
  -VerboseOutput:$VerboseOutput
