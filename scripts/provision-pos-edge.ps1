param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$ArgsFromCaller
)

$ErrorActionPreference = "Stop"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
& python (Join-Path $scriptDir "provision-pos-edge.py") @ArgsFromCaller
exit $LASTEXITCODE
