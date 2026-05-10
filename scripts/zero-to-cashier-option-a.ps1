param(
  [string]$CloudBaseUrl = "http://localhost:8090",
  [string]$EdgeBaseUrl = "http://localhost:8080",
  [string]$UiBaseUrl = "http://localhost:5173",
  [string]$EmployeePin = "1111",
  [string]$ManagerPin = "2222",
  [switch]$RunRuntimeSmoke,
  [switch]$VerboseOutput
)

$ErrorActionPreference = "Stop"

& "$PSScriptRoot/bootstrap-production-way.ps1" `
  -CloudBaseUrl $CloudBaseUrl `
  -EdgeBaseUrl $EdgeBaseUrl `
  -UiBaseUrl $UiBaseUrl `
  -CashierPin $EmployeePin `
  -ManagerPin $ManagerPin `
  -RunRuntimeSmoke:$RunRuntimeSmoke `
  -VerboseOutput:$VerboseOutput
