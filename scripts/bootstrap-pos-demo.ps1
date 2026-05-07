param(
  [string]$PosApiBase = "http://localhost:8080/api/v1"
)

$ErrorActionPreference = "Stop"

# Скрипт ожидает, что POS backend запущен с POS_DEV_TOOLS=1.
$result = Invoke-RestMethod `
  -Method Post `
  -Uri "$PosApiBase/dev/bootstrap-demo" `
  -ContentType "application/json"

Write-Host "Demo bootstrap completed"
Write-Host "Restaurant ID:      $($result.restaurant_id)"
Write-Host "Node device ID:     $($result.node_device_id)"
Write-Host "Pairing code:       $($result.pairing_code)"
Write-Host "Cashier PIN:        $($result.cashier_pin)"
Write-Host "Manager PIN:        $($result.manager_pin)"
Write-Host "Manager employee:   $($result.manager_employee_id)"
Write-Host "Hall ID:            $($result.hall_id)"
Write-Host "Tables:             $($result.table_ids -join ', ')"
Write-Host "Menu items:         $($result.menu_item_ids -join ', ')"

$result
