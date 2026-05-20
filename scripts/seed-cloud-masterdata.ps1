param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ArgsFromCaller
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
& python (Join-Path $scriptDir "seed-cloud-masterdata.py") @ArgsFromCaller
exit $LASTEXITCODE
