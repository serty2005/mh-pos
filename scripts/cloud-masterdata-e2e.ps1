param(
  [string]$CloudApiBase = "http://localhost:8090/api/v1",
  [string]$PosApiBase = "http://localhost:8080/api/v1",
  [string]$NodeDeviceId = "cloud-e2e-edge-node-1",
  [string]$ClientDeviceId = "cloud-e2e-client-1",
  [string]$EmployeePin = "1357"
)

$ErrorActionPreference = "Stop"

function Invoke-JsonPost {
  param(
    [string]$Uri,
    [object]$Body
  )
  Invoke-RestMethod -Method Post -Uri $Uri -ContentType "application/json" -Body ($Body | ConvertTo-Json -Depth 12)
}

function Invoke-JsonPatch {
  param(
    [string]$Uri,
    [object]$Body
  )
  Invoke-RestMethod -Method Patch -Uri $Uri -ContentType "application/json" -Body ($Body | ConvertTo-Json -Depth 12)
}

Write-Host "Checking Cloud health..."
Invoke-RestMethod -Uri ($CloudApiBase -replace "/api/v1$", "/health") | Out-Null

Write-Host "Checking POS Edge health..."
Invoke-RestMethod -Uri ($PosApiBase -replace "/api/v1$", "/health") | Out-Null

$suffix = [guid]::NewGuid().ToString("N").Substring(0, 8)
$restaurant = Invoke-JsonPost "$CloudApiBase/restaurants" @{
  name = "Cloud E2E Bistro $suffix"
  timezone = "Europe/Moscow"
  currency = "RUB"
  business_day_mode = "standard"
  business_day_boundary_local_time = "04:00"
}

$permissions = @{
  "pos.employee_shift.open" = $true
  "pos.employee_shift.view_current" = $true
  "pos.catalog.view" = $true
  "pos.menu.view" = $true
  "pos.order.create" = $true
  "pos.order.view" = $true
  "pos.order.add_line" = $true
  "pos.precheck.issue" = $true
  "pos.precheck.view" = $true
  "pos.payment.cash" = $true
  "pos.check.view" = $true
  "pos.sync.view" = $true
}

$role = Invoke-JsonPost "$CloudApiBase/roles" @{
  restaurant_id = $restaurant.id
  name = "cloud-e2e-cashier-$suffix"
  permissions_json = ($permissions | ConvertTo-Json -Compress)
}

$employee = Invoke-JsonPost "$CloudApiBase/employees" @{
  restaurant_id = $restaurant.id
  role_id = $role.id
  name = "Cloud E2E Cashier"
  pin = $EmployeePin
}

$catalog = Invoke-JsonPost "$CloudApiBase/catalog/items" @{
  restaurant_id = $restaurant.id
  type = "dish"
  name = "Cloud E2E Tea"
  sku = "CLOUD-E2E-TEA-$suffix"
  base_unit = "portion"
}

$menu = Invoke-JsonPost "$CloudApiBase/menu/items" @{
  restaurant_id = $restaurant.id
  catalog_item_id = $catalog.id
  name = "Cloud E2E Tea"
  price = 15000
  currency = "RUB"
  availability_json = "{}"
}

$publication = Invoke-JsonPost "$CloudApiBase/restaurants/$($restaurant.id)/master-data/publish" @{
  published_by = "cloud-masterdata-e2e"
  node_device_id = $NodeDeviceId
}

Write-Host "Published package $($publication.id) for restaurant $($restaurant.id)"

$snapshot = Invoke-RestMethod -Uri "$CloudApiBase/restaurants/$($restaurant.id)/edge-nodes/$NodeDeviceId/master-data/snapshot"

Write-Host "Applying Cloud -> Edge master-data snapshot..."
Invoke-RestMethod `
  -Method Post `
  -Uri "$PosApiBase/sync/master-data/snapshots" `
  -ContentType "application/json" `
  -Body ($snapshot | ConvertTo-Json -Depth 20) | Out-Null

Write-Host "Pairing POS Edge node with Cloud-created restaurant..."
$pairingCode = "MHPOS:$($restaurant.id):$NodeDeviceId"
Invoke-JsonPost "$PosApiBase/system/pair" @{ pairing_code = $pairingCode } | Out-Null

Write-Host "Logging in with Cloud-created employee PIN..."
$login = Invoke-JsonPost "$PosApiBase/auth/pin-login" @{
  node_device_id = $NodeDeviceId
  client_device_id = $ClientDeviceId
  pin = $EmployeePin
}

$headers = @{
  "X-Node-Device-ID" = $NodeDeviceId
  "X-Client-Device-ID" = $ClientDeviceId
  "X-Session-ID" = $login.session.id
  "X-Actor-Employee-ID" = $login.actor.employee_id
}

Write-Host "Checking POS Edge offline read model..."
$catalogItems = Invoke-RestMethod -Headers $headers -Uri "$PosApiBase/catalog/items"
$menuItems = Invoke-RestMethod -Headers $headers -Uri "$PosApiBase/menu/items"

if (-not ($catalogItems | ConvertTo-Json -Depth 8).Contains($catalog.id)) {
  throw "Cloud-created catalog item was not visible on POS Edge."
}
if (-not ($menuItems | ConvertTo-Json -Depth 8).Contains($menu.id)) {
  throw "Cloud-created menu item was not visible on POS Edge."
}
if ($login.actor.employee_id -ne $employee.id) {
  throw "PIN login did not resolve Cloud-created employee. Expected $($employee.id), got $($login.actor.employee_id)."
}

Write-Host "Checking that master-data apply did not create master-data operational outbox rows..."
$outbox = Invoke-RestMethod -Headers $headers -Uri "$PosApiBase/sync/outbox?limit=100"
$outboxJson = $outbox | ConvertTo-Json -Depth 20
foreach ($forbidden in @("RestaurantCreated", "RoleCreated", "EmployeeCreated", "CatalogItemCreated", "MenuItemCreated")) {
  if ($outboxJson.Contains($forbidden)) {
    throw "Unexpected master-data operational event in Edge outbox: $forbidden"
  }
}

Write-Host "Cloud master-data E2E completed."
[pscustomobject]@{
  restaurant_id = $restaurant.id
  node_device_id = $NodeDeviceId
  employee_id = $employee.id
  role_id = $role.id
  catalog_item_id = $catalog.id
  menu_item_id = $menu.id
  publication_id = $publication.id
}
