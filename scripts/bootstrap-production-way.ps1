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
$OutputEncoding = [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
$env:PYTHONIOENCODING = "utf-8"

function Normalize-BaseUrl([string]$Url) {
  return ($Url.TrimEnd("/") -replace "/api/v1$", "")
}

$CloudBaseUrl = Normalize-BaseUrl $CloudBaseUrl
$EdgeBaseUrl = Normalize-BaseUrl $EdgeBaseUrl
$cloudApi = $CloudBaseUrl + "/api/v1"
$cloudMasterDataApi = $cloudApi + "/master-data"
$edgeApi = $EdgeBaseUrl + "/api/v1"
$clientDeviceId = "production-way-smoke-client"
$commandSequence = 0

function Write-Step([string]$Message) {
  if ($VerboseOutput) {
    Write-Host "==> $Message" -ForegroundColor Cyan
  }
}

function Convert-ToJsonBody([object]$Body) {
  return ($Body | ConvertTo-Json -Depth 40)
}

function Get-HttpErrorBody {
  param([object]$ErrorRecord)

  $responseBody = ""
  $statusCode = ""

  if ($ErrorRecord.ErrorDetails -and -not [string]::IsNullOrWhiteSpace($ErrorRecord.ErrorDetails.Message)) {
    $responseBody = $ErrorRecord.ErrorDetails.Message
  }

  try {
    if ($ErrorRecord.Exception.Response) {
      $statusCode = [int]$ErrorRecord.Exception.Response.StatusCode
      $stream = $ErrorRecord.Exception.Response.GetResponseStream()
      if ($stream) {
        $reader = New-Object System.IO.StreamReader($stream)
        $bodyFromStream = $reader.ReadToEnd()
        if (-not [string]::IsNullOrWhiteSpace($bodyFromStream)) {
          $responseBody = $bodyFromStream
        }
      }
    }
  } catch {
  }

  return [pscustomobject]@{
    StatusCode = $statusCode
    Body = $responseBody
  }
}

function Invoke-JsonPost {
  param(
    [string]$Uri,
    [object]$Body = @{},
    [hashtable]$Headers = @{},
    [int[]]$ExpectedStatus = @(200, 201)
  )

  $json = Convert-ToJsonBody $Body

  try {
    $response = Invoke-WebRequest `
      -Method Post `
      -Uri $Uri `
      -ContentType "application/json" `
      -Body $json `
      -Headers $Headers `
      -UseBasicParsing `
      -ErrorAction Stop

    if ($ExpectedStatus -notcontains [int]$response.StatusCode) {
      throw "Unexpected HTTP $($response.StatusCode) from $Uri`: $($response.Content)"
    }

    if ([string]::IsNullOrWhiteSpace($response.Content)) {
      return $null
    }

    return $response.Content | ConvertFrom-Json
  } catch {
    $details = Get-HttpErrorBody $_

    Write-Host ""
    Write-Host "POST failed:" -ForegroundColor Red
    Write-Host "URI: $Uri" -ForegroundColor Red
    Write-Host "Status: $($details.StatusCode)" -ForegroundColor Red
    Write-Host "Request body:" -ForegroundColor Yellow
    Write-Host $json
    Write-Host "Response body:" -ForegroundColor Yellow
    Write-Host $details.Body
    Write-Host ""

    if (-not [string]::IsNullOrWhiteSpace($details.Body)) {
      throw "POST $Uri failed. HTTP $($details.StatusCode). Response body: $($details.Body)"
    }

    throw "POST $Uri failed. $($_.Exception.Message)"
  }
}

function Invoke-JsonGet {
  param(
    [string]$Uri,
    [hashtable]$Headers = @{},
    [int[]]$ExpectedStatus = @(200)
  )

  try {
    $response = Invoke-WebRequest `
      -Method Get `
      -Uri $Uri `
      -Headers $Headers `
      -UseBasicParsing `
      -ErrorAction Stop

    if ($ExpectedStatus -notcontains [int]$response.StatusCode) {
      throw "Unexpected HTTP $($response.StatusCode) from $Uri`: $($response.Content)"
    }

    if ([string]::IsNullOrWhiteSpace($response.Content)) {
      return $null
    }

    return $response.Content | ConvertFrom-Json
  } catch {
    $details = Get-HttpErrorBody $_
    if (-not [string]::IsNullOrWhiteSpace($details.Body)) {
      throw "GET $Uri failed. HTTP $($details.StatusCode). Response body: $($details.Body)"
    }
    throw "GET $Uri failed. $($_.Exception.Message)"
  }
}

function New-CommandId([string]$Prefix) {
  $script:commandSequence += 1
  return "cmd-production-way-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())-$($script:commandSequence)-$Prefix"
}

function New-PermissionsJson([string[]]$Permissions) {
  $map = [ordered]@{}
  foreach ($permission in ($Permissions | Sort-Object -Unique)) {
    $map[$permission] = $true
  }
  return ($map | ConvertTo-Json -Compress)
}

function New-AuthHeaders($Login, [string]$NodeId) {
  return @{
    "X-Node-Device-ID"    = $NodeId
    "X-Client-Device-ID"  = $clientDeviceId
    "X-Session-ID"        = $Login.session.id
    "X-Actor-Employee-ID" = $Login.actor.employee_id
  }
}

function Assert-JsonContains([object]$Value, [string]$Needle, [string]$Message) {
  $json = $Value | ConvertTo-Json -Depth 40
  if (-not $json.Contains($Needle)) {
    throw $Message
  }
}

function Set-JsonProperty([object]$Object, [string]$Name, [object]$Value) {
  if ($Object.PSObject.Properties[$Name]) {
    $Object.$Name = $Value
  } else {
    $Object | Add-Member -NotePropertyName $Name -NotePropertyValue $Value
  }
}

function Remove-JsonProperty([object]$Object, [string]$Name) {
  if ($null -ne $Object -and $Object.PSObject.Properties[$Name]) {
    $Object.PSObject.Properties.Remove($Name)
  }
}

function Convert-CloudSnapshotForEdge([object]$Snapshot, [string]$RestaurantId, [string]$NodeId) {
  $payload = $Snapshot

  foreach ($candidate in @("payload", "snapshot", "master_data", "data")) {
    if ($payload.PSObject.Properties[$candidate]) {
      $payload = $payload.$candidate
      break
    }
  }

  # Cloud publication may include rich read-model fields that current POS Edge DTOs
  # do not accept under strict JSON decode. Keep canonical modifier links through
  # top-level arrays, but strip presentation/Cloud-only fields known to break ingest.
  if ($payload.PSObject.Properties["menu_items"] -and $null -ne $payload.menu_items) {
    foreach ($item in $payload.menu_items) {
      Remove-JsonProperty $item "modifier_groups"
    }
  }

  if ($payload.PSObject.Properties["modifier_groups"] -and $null -ne $payload.modifier_groups) {
    foreach ($group in $payload.modifier_groups) {
      # Current Edge runtime rejects Cloud modifier group field "required".
      # The bootstrap creates optional groups only, so omitting it is safe for smoke.
      Remove-JsonProperty $group "required"
    }
  }

  Set-JsonProperty $payload "node_device_id" $NodeId

  if ([string]::IsNullOrWhiteSpace($payload.restaurant_id)) {
    Set-JsonProperty $payload "restaurant_id" $RestaurantId
  }

  if ([string]::IsNullOrWhiteSpace($payload.sync_mode)) {
    Set-JsonProperty $payload "sync_mode" "incremental"
  }

  if ($payload.sync_mode -eq "full_snapshot" -and [string]::IsNullOrWhiteSpace($payload.full_snapshot_reason)) {
    Set-JsonProperty $payload "full_snapshot_reason" "terminal_restaurant_changed"
  }

  if ($null -eq $payload.cloud_version) {
    Set-JsonProperty $payload "cloud_version" 1
  }

  return $payload
}

function Invoke-RuntimeSmoke {
  param(
    [string]$RestaurantId,
    [string]$NodeId,
    [string]$CashierEmployeeId,
    [string]$ManagerEmployeeId,
    [string]$TableId,
    [string[]]$MenuItemIds,
    [string]$ModifierGroupId,
    [string]$ModifierOptionId
  )

  Write-Step "Checking cashier PIN login"
  $cashierLogin = Invoke-JsonPost "$edgeApi/auth/pin-login" @{
    node_device_id = $NodeId
    client_device_id = $clientDeviceId
    pin = $CashierPin
  } -ExpectedStatus @(201)

  if ($cashierLogin.actor.employee_id -ne $CashierEmployeeId) {
    throw "Cashier PIN resolved unexpected employee_id=$($cashierLogin.actor.employee_id)"
  }

  Write-Step "Checking manager PIN login for refund-capable smoke"
  $managerLogin = Invoke-JsonPost "$edgeApi/auth/pin-login" @{
    node_device_id = $NodeId
    client_device_id = $clientDeviceId
    pin = $ManagerPin
  } -ExpectedStatus @(201)

  if ($managerLogin.actor.employee_id -ne $ManagerEmployeeId) {
    throw "Manager PIN resolved unexpected employee_id=$($managerLogin.actor.employee_id)"
  }

  $headers = New-AuthHeaders $managerLogin $NodeId

  Write-Step "Opening employee shift and cash shift"
  $shift = Invoke-JsonPost "$edgeApi/employee-shifts/open" @{
    command_id = New-CommandId "open-employee-shift"
    restaurant_id = $RestaurantId
    opened_by_employee_id = $ManagerEmployeeId
  } -Headers $headers -ExpectedStatus @(201)

  $cashShift = Invoke-JsonPost "$edgeApi/cash-shifts/open" @{
    command_id = New-CommandId "open-cash-shift"
    restaurant_id = $RestaurantId
    opened_by_employee_id = $ManagerEmployeeId
    opening_cash_amount = 0
  } -Headers $headers -ExpectedStatus @(201)

  Write-Step "Creating order with dish, modifier and service lines"
  $order = Invoke-JsonPost "$edgeApi/orders" @{
    command_id = New-CommandId "create-order"
    restaurant_id = $RestaurantId
    table_id = $TableId
    table_name = "Smoke T1"
    guest_count = 1
  } -Headers $headers -ExpectedStatus @(201)

  Invoke-JsonPost "$edgeApi/orders/$($order.id)/lines" @{
    command_id = New-CommandId "add-dish-with-modifier"
    menu_item_id = $MenuItemIds[0]
    quantity = 1
    selected_modifiers = @(
      @{
        modifier_group_id = $ModifierGroupId
        modifier_option_id = $ModifierOptionId
        quantity = 1
      }
    )
  } -Headers $headers -ExpectedStatus @(201) | Out-Null

  Invoke-JsonPost "$edgeApi/orders/$($order.id)/lines" @{
    command_id = New-CommandId "add-second-dish"
    menu_item_id = $MenuItemIds[1]
    quantity = 1
  } -Headers $headers -ExpectedStatus @(201) | Out-Null

  Invoke-JsonPost "$edgeApi/orders/$($order.id)/lines" @{
    command_id = New-CommandId "add-service-line"
    menu_item_id = $MenuItemIds[2]
    quantity = 1
  } -Headers $headers -ExpectedStatus @(201) | Out-Null

  Write-Step "Issuing precheck and capturing full payment"
  $precheck = Invoke-JsonPost "$edgeApi/orders/$($order.id)/precheck" @{
    command_id = New-CommandId "issue-precheck"
  } -Headers $headers -ExpectedStatus @(201)

  if ($precheck.total -le 0) {
    throw "Precheck total must be positive, got $($precheck.total)"
  }

  $precheckReprint = Invoke-JsonPost "$edgeApi/prechecks/$($precheck.id)/reprint" @{
    command_id = New-CommandId "reprint-precheck"
    reason = "production-way smoke precheck reprint"
  } -Headers $headers -ExpectedStatus @(201)

  if (-not $precheckReprint.snapshot) {
    throw "Expected precheck reprint snapshot."
  }

  $payment = Invoke-JsonPost "$edgeApi/prechecks/$($precheck.id)/payments" @{
    command_id = New-CommandId "capture-payment"
    method = "cash"
    amount = $precheck.total
    currency = "RUB"
  } -Headers $headers -ExpectedStatus @(201)

  if ($payment.status -ne "captured") {
    throw "Expected captured payment, got $($payment.status)"
  }

  $closedOrder = Invoke-JsonGet "$edgeApi/orders/$($order.id)" -Headers $headers

  if ($closedOrder.status -ne "closed" -or -not $closedOrder.check -or $closedOrder.check.status -ne "paid") {
    throw "Expected closed paid order after full payment."
  }

  $checkReprint = Invoke-JsonPost "$edgeApi/checks/$($closedOrder.check.id)/reprint" @{
    command_id = New-CommandId "reprint-check"
    reason = "production-way smoke check reprint"
  } -Headers $headers -ExpectedStatus @(201)

  if (-not $checkReprint.snapshot) {
    throw "Expected check reprint snapshot."
  }

  Write-Step "Recording same-shift full check cancellation"
  $cancelOrder = Invoke-JsonPost "$edgeApi/orders" @{
    command_id = New-CommandId "create-cancellation-order"
    restaurant_id = $RestaurantId
    table_id = $TableId
    table_name = "Smoke T1"
    guest_count = 1
  } -Headers $headers -ExpectedStatus @(201)

  Invoke-JsonPost "$edgeApi/orders/$($cancelOrder.id)/lines" @{
    command_id = New-CommandId "add-cancellation-line"
    menu_item_id = $MenuItemIds[1]
    quantity = 1
  } -Headers $headers -ExpectedStatus @(201) | Out-Null

  $cancelPrecheck = Invoke-JsonPost "$edgeApi/orders/$($cancelOrder.id)/precheck" @{
    command_id = New-CommandId "issue-cancellation-precheck"
  } -Headers $headers -ExpectedStatus @(201)

  $cancelPayment = Invoke-JsonPost "$edgeApi/prechecks/$($cancelPrecheck.id)/payments" @{
    command_id = New-CommandId "capture-cancellation-payment"
    method = "cash"
    amount = $cancelPrecheck.total
    currency = "RUB"
  } -Headers $headers -ExpectedStatus @(201)

  if ($cancelPayment.status -ne "captured") {
    throw "Expected captured payment for cancellation check, got $($cancelPayment.status)"
  }

  $cancelClosedOrder = Invoke-JsonGet "$edgeApi/orders/$($cancelOrder.id)" -Headers $headers

  if ($cancelClosedOrder.status -ne "closed" -or -not $cancelClosedOrder.check -or $cancelClosedOrder.check.status -ne "paid") {
    throw "Expected closed paid cancellation order before ledger operation."
  }

  $cancellation = Invoke-JsonPost "$edgeApi/checks/$($cancelClosedOrder.check.id)/cancellations" @{
    command_id = New-CommandId "record-cancellation"
    operation_kind = "full"
    inventory_disposition = "manual_review"
    reason = "production-way smoke same-shift cancellation"
  } -Headers $headers -ExpectedStatus @(201)

  if ($cancellation.operation_type -ne "cancellation" -or $cancellation.operation_kind -ne "full" -or $cancellation.status -ne "recorded") {
    throw "Expected recorded full cancellation operation, got operation_type=$($cancellation.operation_type), kind=$($cancellation.operation_kind), status=$($cancellation.status)"
  }

  $cancelOrderAfter = Invoke-JsonGet "$edgeApi/orders/$($cancelOrder.id)" -Headers $headers

  if ($cancelOrderAfter.status -ne "closed" -or $cancelOrderAfter.check.status -ne "paid") {
    throw "Cancellation ledger operation must not mutate closed order/check status."
  }

  Write-Step "Closing original shift before refund boundary"
  Invoke-JsonPost "$edgeApi/cash-shifts/$($cashShift.id)/close" @{
    command_id = New-CommandId "close-cash-shift-before-refund"
    closed_by_employee_id = $ManagerEmployeeId
    closing_cash_amount = 0
  } -Headers $headers -ExpectedStatus @(200) | Out-Null

  Invoke-JsonPost "$edgeApi/employee-shifts/$($shift.id)/close" @{
    command_id = New-CommandId "close-employee-shift-before-refund"
    closed_by_employee_id = $ManagerEmployeeId
  } -Headers $headers -ExpectedStatus @(200) | Out-Null

  Write-Step "Opening refund shift and cash shift"
  $refundShift = Invoke-JsonPost "$edgeApi/employee-shifts/open" @{
    command_id = New-CommandId "open-refund-employee-shift"
    restaurant_id = $RestaurantId
    opened_by_employee_id = $ManagerEmployeeId
  } -Headers $headers -ExpectedStatus @(201)

  $refundCashShift = Invoke-JsonPost "$edgeApi/cash-shifts/open" @{
    command_id = New-CommandId "open-refund-cash-shift"
    restaurant_id = $RestaurantId
    opened_by_employee_id = $ManagerEmployeeId
    opening_cash_amount = 0
  } -Headers $headers -ExpectedStatus @(201)

  Write-Step "Recording refund operation and checking local sync artifacts"
  $refund = Invoke-JsonPost "$edgeApi/checks/$($closedOrder.check.id)/refunds" @{
    command_id = New-CommandId "record-refund"
    operation_kind = "full"
    inventory_disposition = "no_stock_effect"
    reason = "production-way smoke refund"
  } -Headers $headers -ExpectedStatus @(201)

  if ($refund.operation_type -ne "refund" -or $refund.operation_kind -ne "full" -or $refund.status -ne "recorded") {
    throw "Expected recorded full refund operation, got operation_type=$($refund.operation_type), kind=$($refund.operation_kind), status=$($refund.status)"
  }

  $orderAfterRefund = Invoke-JsonGet "$edgeApi/orders/$($order.id)" -Headers $headers

  if ($orderAfterRefund.status -ne "closed" -or $orderAfterRefund.check.status -ne "paid") {
    throw "Refund ledger operation must not mutate closed order/check status."
  }

  $outbox = Invoke-JsonGet "$edgeApi/sync/outbox?limit=50" -Headers $headers
  $localEvents = Invoke-JsonGet "$edgeApi/sync/local-events?limit=50" -Headers $headers

  Assert-JsonContains $outbox $refund.id "Expected refund operation event in Edge outbox."
  Assert-JsonContains $outbox $cancellation.id "Expected cancellation operation event in Edge outbox."
  Assert-JsonContains $localEvents $order.id "Expected order events in Edge local event log."

  Write-Step "Closing refund shift and cash shift"
  Invoke-JsonPost "$edgeApi/cash-shifts/$($refundCashShift.id)/close" @{
    command_id = New-CommandId "close-refund-cash-shift"
    closed_by_employee_id = $ManagerEmployeeId
    closing_cash_amount = 0
  } -Headers $headers -ExpectedStatus @(200) | Out-Null

  Invoke-JsonPost "$edgeApi/employee-shifts/$($refundShift.id)/close" @{
    command_id = New-CommandId "close-refund-employee-shift"
    closed_by_employee_id = $ManagerEmployeeId
  } -Headers $headers -ExpectedStatus @(200) | Out-Null

  return [pscustomobject]@{
    shift_id = $refundShift.id
    cash_shift_id = $refundCashShift.id
    original_shift_id = $shift.id
    original_cash_shift_id = $cashShift.id
    order_id = $order.id
    precheck_id = $precheck.id
    precheck_reprint_source_id = $precheckReprint.source_id
    payment_id = $payment.id
    check_id = $closedOrder.check.id
    check_reprint_source_id = $checkReprint.source_id
    cancellation_order_id = $cancelOrder.id
    cancellation_check_id = $cancelClosedOrder.check.id
    cancellation_operation_id = $cancellation.id
    cancellation_status = $cancellation.status
    refund_operation_id = $refund.id
    refund_status = $refund.status
  }
}

Write-Step "Checking Cloud and POS Edge health"
Invoke-JsonGet ($CloudBaseUrl.TrimEnd("/") + "/health") | Out-Null
Invoke-JsonGet ($EdgeBaseUrl.TrimEnd("/") + "/health") | Out-Null

$suffix = [guid]::NewGuid().ToString("N").Substring(0, 8)

if ([string]::IsNullOrWhiteSpace($RestaurantName)) {
  $RestaurantName = "Production Way Bistro $suffix"
}

$provisioningStatus = Invoke-JsonGet "$edgeApi/system/provisioning-status"

if ([string]::IsNullOrWhiteSpace($NodeDeviceId)) {
  $NodeDeviceId = $provisioningStatus.node_device_id
}

if ([string]::IsNullOrWhiteSpace($NodeDeviceId)) {
  throw "POS Edge did not return node_device_id from /system/provisioning-status."
}

if ($provisioningStatus.node_device_id -and $NodeDeviceId -ne $provisioningStatus.node_device_id) {
  throw "NodeDeviceId must match local POS Edge identity. Provided $NodeDeviceId, Edge returned $($provisioningStatus.node_device_id)."
}

$cashierPermissions = New-PermissionsJson @(
  "pos.employee_shift.open",
  "pos.employee_shift.close",
  "pos.employee_shift.view_current",
  "pos.employee_shift.recent",
  "pos.cash_session.open",
  "pos.cash_session.view_current",
  "pos.catalog.view",
  "pos.floor.view",
  "pos.menu.view",
  "pos.order.create",
  "pos.order.view",
  "pos.order.add_line",
  "pos.order.change_quantity",
  "pos.order.void_line",
  "pos.order.close",
  "pos.precheck.issue",
  "pos.precheck.view",
  "pos.precheck.reprint",
  "pos.payment.cash",
  "pos.payment.card.manual",
  "pos.check.view"
)

$managerPermissions = New-PermissionsJson @(
  "pos.employee_shift.open",
  "pos.employee_shift.close",
  "pos.employee_shift.view_current",
  "pos.employee_shift.recent",
  "pos.cash_session.open",
  "pos.cash_session.close",
  "pos.cash_session.view_current",
  "pos.cash_drawer.record_event",
  "pos.catalog.view",
  "pos.floor.view",
  "pos.menu.view",
  "pos.order.create",
  "pos.order.view",
  "pos.order.add_line",
  "pos.order.change_quantity",
  "pos.order.void_line",
  "pos.order.close",
  "pos.precheck.issue",
  "pos.precheck.view",
  "pos.precheck.reprint",
  "pos.precheck.cancel.request",
  "pos.precheck.cancel",
  "pos.payment.cash",
  "pos.payment.card.manual",
  "pos.payment.other",
  "pos.payment.refund",
  "pos.check.view",
  "pos.check.reprint",
  "pos.sync.view",
  "pos.sync.retry_failed"
)

Write-Step "Creating Cloud-owned restaurant, roles, employees, floor and menu"

$restaurant = Invoke-JsonPost "$cloudApi/restaurants" @{
  name = $RestaurantName
  timezone = "Europe/Moscow"
  currency = "RUB"
  business_day_mode = "standard"
  business_day_boundary_local_time = "04:00"
} -ExpectedStatus @(201)

$cashierRole = Invoke-JsonPost "$cloudApi/roles" @{
  restaurant_id = $restaurant.id
  name = "cashier-$suffix"
  permissions_json = $cashierPermissions
} -ExpectedStatus @(201)

$managerRole = Invoke-JsonPost "$cloudApi/roles" @{
  restaurant_id = $restaurant.id
  name = "manager-$suffix"
  permissions_json = $managerPermissions
} -ExpectedStatus @(201)

$cashier = Invoke-JsonPost "$cloudApi/employees" @{
  restaurant_id = $restaurant.id
  role_id = $cashierRole.id
  name = "Production Cashier"
  pin = $CashierPin
} -ExpectedStatus @(201)

$manager = Invoke-JsonPost "$cloudApi/employees" @{
  restaurant_id = $restaurant.id
  role_id = $managerRole.id
  name = "Production Manager"
  pin = $ManagerPin
} -ExpectedStatus @(201)

$hall = Invoke-JsonPost "$cloudApi/halls" @{
  restaurant_id = $restaurant.id
  name = "Main Hall"
} -ExpectedStatus @(201)

$table = Invoke-JsonPost "$cloudApi/tables" @{
  restaurant_id = $restaurant.id
  hall_id = $hall.id
  name = "T1"
  seats = 2
} -ExpectedStatus @(201)

$catalogTea = Invoke-JsonPost "$cloudApi/catalog/items" @{
  restaurant_id = $restaurant.id
  type = "dish"
  name = "Production Tea"
  sku = "PROD-WAY-TEA-$suffix"
  base_unit = "portion"
} -ExpectedStatus @(201)

$catalogSoup = Invoke-JsonPost "$cloudApi/catalog/items" @{
  restaurant_id = $restaurant.id
  type = "dish"
  name = "Production Soup"
  sku = "PROD-WAY-SOUP-$suffix"
  base_unit = "portion"
} -ExpectedStatus @(201)

$catalogService = Invoke-JsonPost "$cloudApi/catalog/items" @{
  restaurant_id = $restaurant.id
  type = "service"
  name = "Production Service"
  sku = "PROD-WAY-SERVICE-$suffix"
  base_unit = "service"
} -ExpectedStatus @(201)

$menuTea = Invoke-JsonPost "$cloudApi/menu/items" @{
  restaurant_id = $restaurant.id
  catalog_item_id = $catalogTea.id
  name = "Production Tea"
  price = 15000
  currency = "RUB"
  availability_json = "{}"
} -ExpectedStatus @(201)

$menuSoup = Invoke-JsonPost "$cloudApi/menu/items" @{
  restaurant_id = $restaurant.id
  catalog_item_id = $catalogSoup.id
  name = "Production Soup"
  price = 25000
  currency = "RUB"
  availability_json = "{}"
} -ExpectedStatus @(201)

$menuService = Invoke-JsonPost "$cloudApi/menu/items" @{
  restaurant_id = $restaurant.id
  catalog_item_id = $catalogService.id
  name = "Production Service"
  price = 5000
  currency = "RUB"
  availability_json = "{}"
} -ExpectedStatus @(201)

$modifierGroup = Invoke-JsonPost "$cloudMasterDataApi/modifiers/groups" @{
  restaurant_id = $restaurant.id
  name = "Production Add-ons"
  required = $false
  min_count = 0
  max_count = 2
} -ExpectedStatus @(201)

$modifierOption = Invoke-JsonPost "$cloudMasterDataApi/modifiers/options" @{
  restaurant_id = $restaurant.id
  modifier_group_id = $modifierGroup.id
  name = "Lemon"
  price_minor = 3000
} -ExpectedStatus @(201)

$modifierBinding = Invoke-JsonPost "$cloudMasterDataApi/modifiers/bindings" @{
  restaurant_id = $restaurant.id
  modifier_group_id = $modifierGroup.id
  target_type = "menu_item"
  target_id = $menuTea.id
  sort_order = 1
} -ExpectedStatus @(201)

Write-Step "Publishing master-data package and applying Edge-ready snapshot"

$publication = Invoke-JsonPost "$cloudApi/restaurants/$($restaurant.id)/master-data/publish" @{
  published_by = "bootstrap-production-way"
  node_device_id = $NodeDeviceId
} -ExpectedStatus @(201)

$cloudSnapshot = Invoke-JsonGet "$cloudApi/restaurants/$($restaurant.id)/edge-nodes/$NodeDeviceId/master-data/snapshot"
$edgeSnapshot = Convert-CloudSnapshotForEdge $cloudSnapshot $restaurant.id $NodeDeviceId
Invoke-JsonPost "$edgeApi/sync/master-data/snapshots" $edgeSnapshot -ExpectedStatus @(200) | Out-Null

Write-Step "Creating production provisioning/license code when available"

$pairingCode = $null

try {
  $pairing = Invoke-JsonPost "$cloudApi/restaurants/$($restaurant.id)/devices/generate-pairing-code" @{
    node_device_id = $NodeDeviceId
    display_name = "POS Terminal 1"
    expires_in_minutes = 30
  } -ExpectedStatus @(201)

  $pairingCode = $pairing.pairing_code

  $paired = Invoke-JsonPost "$edgeApi/system/provisioning/pair-via-license" @{
    pairing_code = $pairingCode
  } -ExpectedStatus @(200)

  if ($paired.node_device_id) {
    $NodeDeviceId = $paired.node_device_id
  }
} catch {
  Write-Step "License code flow is unavailable; using Cloud approve assignment plus direct snapshot ingest"

  try {
    Invoke-JsonPost "$edgeApi/system/provisioning/register-cloud" @{
      cloud_url = $CloudBaseUrl.TrimEnd("/")
      display_name = "POS Terminal 1"
      app_version = "local-smoke"
    } -ExpectedStatus @(200) | Out-Null
  } catch {
    Write-Step "Edge cloud registration was not available before Cloud assignment; continuing with explicit assignment"
  }

  Invoke-JsonPost "$cloudApi/restaurants/$($restaurant.id)/devices/$NodeDeviceId/assign" @{} -ExpectedStatus @(200) | Out-Null

  for ($i = 0; $i -lt 20; $i++) {
    $polled = Invoke-JsonGet "$edgeApi/system/provisioning-status"
    if ($polled.paired) {
      break
    }
    Start-Sleep -Seconds 1
  }

  $pairingCode = "Cloud-approved:$NodeDeviceId"
}

Write-Step "Checking Edge pairing status, read model and PIN login"

$pairingStatus = Invoke-JsonGet "$edgeApi/system/pairing-status"

$managerLoginForRead = Invoke-JsonPost "$edgeApi/auth/pin-login" @{
  node_device_id = $NodeDeviceId
  client_device_id = $clientDeviceId
  pin = $ManagerPin
} -ExpectedStatus @(201)

$readHeaders = New-AuthHeaders $managerLoginForRead $NodeDeviceId

$halls = Invoke-JsonGet "$edgeApi/halls?restaurant_id=$($restaurant.id)" -Headers $readHeaders
$tables = Invoke-JsonGet "$edgeApi/tables?restaurant_id=$($restaurant.id)&hall_id=$($hall.id)" -Headers $readHeaders
$menuItems = Invoke-JsonGet "$edgeApi/menu/items" -Headers $readHeaders

Assert-JsonContains $halls $hall.id "Cloud-created hall is not visible on POS Edge."
Assert-JsonContains $tables $table.id "Cloud-created table is not visible on POS Edge."
Assert-JsonContains $menuItems $menuTea.id "First Cloud-created menu item is not visible on POS Edge."
Assert-JsonContains $menuItems $menuSoup.id "Second Cloud-created menu item is not visible on POS Edge."
Assert-JsonContains $menuItems $menuService.id "Cloud-created service menu item is not visible on POS Edge."
Assert-JsonContains $menuItems $modifierOption.id "Cloud-created modifier option is not visible on POS Edge menu item."

$runtimeSmoke = $null

if ($RunRuntimeSmoke) {
  $runtimeSmoke = Invoke-RuntimeSmoke `
    -RestaurantId $restaurant.id `
    -NodeId $NodeDeviceId `
    -CashierEmployeeId $cashier.id `
    -ManagerEmployeeId $manager.id `
    -TableId $table.id `
    -MenuItemIds @($menuTea.id, $menuSoup.id, $menuService.id) `
    -ModifierGroupId $modifierGroup.id `
    -ModifierOptionId $modifierOption.id
}

$summary = [pscustomobject]@{
  restaurant_id = $restaurant.id
  node_device_id = $NodeDeviceId
  pairing_code = $pairingCode
  cashier_pin = $CashierPin
  manager_pin = $ManagerPin
  cashier_employee_id = $cashier.id
  manager_employee_id = $manager.id
  hall_id = $hall.id
  table_ids = @($table.id)
  catalog_item_ids = @($catalogTea.id, $catalogSoup.id, $catalogService.id)
  menu_item_ids = @($menuTea.id, $menuSoup.id, $menuService.id)
  modifier_group_id = $modifierGroup.id
  modifier_option_id = $modifierOption.id
  modifier_binding_id = $modifierBinding.id
  publication_id = $publication.id
  cloud_base_url = $CloudBaseUrl.TrimEnd("/")
  edge_base_url = $EdgeBaseUrl.TrimEnd("/")
  ui_base_url = $UiBaseUrl.TrimEnd("/")
  pairing_status = $pairingStatus
  runtime_smoke = $runtimeSmoke
}

Write-Host "Production-way bootstrap completed"
Write-Host ($summary | ConvertTo-Json -Depth 40)
$summary
