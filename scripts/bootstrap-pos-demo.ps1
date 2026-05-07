param(
  [string]$PosApiBase = "http://localhost:8080/api/v1"
)

$ErrorActionPreference = "Stop"

$healthUri = $PosApiBase -replace "/api/v1$", "/health"

try {
  Invoke-RestMethod -Uri $healthUri | Out-Null
} catch {
  throw "POS backend is not reachable at $healthUri. Start pos-backend first, then run this script again."
}

try {
  $result = Invoke-RestMethod `
    -Method Post `
    -Uri "$PosApiBase/dev/bootstrap-demo" `
    -ContentType "application/json"
} catch {
  $message = $_.Exception.Message
  if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
    $message = $_.ErrorDetails.Message
  }
  if ($message -match "dev bootstrap is disabled|dev bootstrap disabled|forbidden") {
    throw "Dev bootstrap is disabled. Start POS backend with `$env:POS_DEV_TOOLS=`"1`" and run this script again."
  }
  throw
}

Write-Host "Demo bootstrap completed"
Write-Host "Restaurant ID:      $($result.restaurant_id)"
Write-Host "Node device ID:     $($result.node_device_id)"
Write-Host "Pairing code:       $($result.pairing_code)"
Write-Host "Cashier PIN:        $($result.cashier_pin)"
Write-Host "Manager PIN:        $($result.manager_pin)"
Write-Host "Cashier employee:   $($result.cashier_employee_id)"
Write-Host "Manager employee:   $($result.manager_employee_id)"
Write-Host "Hall ID:            $($result.hall_id)"
Write-Host "Tables:             $($result.table_ids -join ', ')"
Write-Host "Menu items:         $($result.menu_item_ids -join ', ')"

$result
