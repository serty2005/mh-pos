param(
  [string]$CloudApiBase = "http://localhost:8090/api/v1",
  [string]$PosApiBase = "http://localhost:8080/api/v1",
  [string]$EmployeePin = "1111",
  [string]$ClientDeviceId = "zero-to-cashier-b"
)

$ErrorActionPreference = "Stop"

function Invoke-JsonPost($Uri, $Body) {
  Invoke-RestMethod -Method Post -Uri $Uri -ContentType "application/json" -Body ($Body | ConvertTo-Json -Depth 20)
}

$suffix = Get-Date -Format "yyyyMMddHHmmss"
$restaurant = Invoke-JsonPost "$CloudApiBase/restaurants" @{
  name = "Zero Cashier B $suffix"
  timezone = "Europe/Moscow"
  currency = "RUB"
  business_day_mode = "standard"
  business_day_boundary_local_time = "04:00"
}
$role = Invoke-JsonPost "$CloudApiBase/roles" @{
  restaurant_id = $restaurant.id
  name = "cashier-$suffix"
  permissions_json = (@{ permissions = @("pos.employee_shift.open","pos.employee_shift.close","pos.employee_shift.view_current","pos.employee_shift.recent","pos.cash_session.open","pos.cash_session.close","pos.cash_session.view_current","pos.floor.view","pos.catalog.view","pos.menu.view","pos.order.create","pos.order.view","pos.order.add_line","pos.order.change_quantity","pos.order.void_line","pos.precheck.issue","pos.precheck.view","pos.payment.cash","pos.check.view") } | ConvertTo-Json -Compress)
}
$employee = Invoke-JsonPost "$CloudApiBase/employees" @{ restaurant_id = $restaurant.id; role_id = $role.id; name = "Zero Cashier"; pin = $EmployeePin }
$hall = Invoke-JsonPost "$CloudApiBase/halls" @{ restaurant_id = $restaurant.id; name = "Main Hall" }
$table = Invoke-JsonPost "$CloudApiBase/tables" @{ restaurant_id = $restaurant.id; hall_id = $hall.id; name = "T1"; seats = 2 }
$catalog = Invoke-JsonPost "$CloudApiBase/catalog/items" @{ restaurant_id = $restaurant.id; type = "dish"; name = "Zero Coffee"; sku = "ZERO-B-COFFEE-$suffix"; base_unit = "portion" }
$menu = Invoke-JsonPost "$CloudApiBase/menu/items" @{ restaurant_id = $restaurant.id; catalog_item_id = $catalog.id; name = "Zero Coffee"; price = 18000; currency = "RUB"; availability_json = "{}" }

$status = Invoke-RestMethod "$PosApiBase/system/provisioning-status"
$code = Invoke-JsonPost "$CloudApiBase/restaurants/$($restaurant.id)/devices/generate-pairing-code" @{
  node_device_id = $status.node_device_id
  display_name = "POS Terminal 1"
  expires_in_minutes = 30
}
$paired = Invoke-JsonPost "$PosApiBase/system/provisioning/pair-via-license" @{ pairing_code = $code.pairing_code }
if (-not $paired.paired) { throw "Edge did not become paired via license code." }

$login = Invoke-JsonPost "$PosApiBase/auth/pin-login" @{ node_device_id = $paired.node_device_id; client_device_id = $ClientDeviceId; pin = $EmployeePin }
$headers = @{ "X-Node-Device-ID" = $paired.node_device_id; "X-Client-Device-ID" = $ClientDeviceId; "X-Session-ID" = $login.session.id; "X-Actor-Employee-ID" = $login.actor.employee_id }
$menuItems = Invoke-RestMethod -Headers $headers "$PosApiBase/menu/items"
$tables = Invoke-RestMethod -Headers $headers "$PosApiBase/tables?restaurant_id=$($restaurant.id)&hall_id=$($hall.id)"
if (-not (($menuItems | ConvertTo-Json -Depth 10).Contains($menu.id))) { throw "Menu item not visible on Edge." }
if (-not (($tables | ConvertTo-Json -Depth 10).Contains($table.id))) { throw "Table not visible on Edge." }

[pscustomobject]@{ restaurant_id = $restaurant.id; node_device_id = $paired.node_device_id; employee_id = $employee.id; pairing_code = $code.pairing_code; menu_item_id = $menu.id; table_id = $table.id }
