param(
  [string]$CloudApiBase = "http://localhost:8090",
  [switch]$ReplayTwice
)

$ErrorActionPreference = "Stop"

$restaurantId = "demo-restaurant"
$deviceId = "demo-edge-node-1"
$eventId = "demo-cloud-replay-event-1"
$occurredAt = "2026-05-07T09:00:00Z"

$envelope = [ordered]@{
  version = "1"
  event_id = $eventId
  command_id = "demo-cloud-replay-command-1"
  event_type = "OrderCreated"
  aggregate_type = "Order"
  aggregate_id = "demo-order-cloud-1"
  restaurant_id = $restaurantId
  device_id = $deviceId
  shift_id = "demo-shift-cloud-1"
  occurred_at = $occurredAt
  payload = [ordered]@{
    origin = "edge_device"
    data = [ordered]@{
      id = "demo-order-cloud-1"
      edge_order_id = "demo-edge-order-cloud-1"
      restaurant_id = $restaurantId
      device_id = $deviceId
      shift_id = "demo-shift-cloud-1"
      status = "open"
      table_name = "A1"
      guest_count = 2
      opened_at = $occurredAt
      created_at = $occurredAt
      updated_at = $occurredAt
    }
  }
}

$body = $envelope | ConvertTo-Json -Depth 8

function Send-Envelope {
  Invoke-RestMethod `
    -Method Post `
    -Uri "$CloudApiBase/api/v1/sync/edge-events" `
    -ContentType "application/json" `
    -Body $body
}

Write-Host "Sending Cloud test envelope..."
$first = Send-Envelope
$first

if ($ReplayTwice) {
  Write-Host "Replaying the same envelope..."
  $second = Send-Envelope
  $second
}
