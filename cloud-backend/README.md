# MyHoReCa Cloud Backend

Minimal Cloud Sync Receiver for the POS/RMS platform.

Current scope:

- Go HTTP entrypoint: `cmd/cloud-api`
- PostgreSQL bootstrap and migrations
- `GET /health`
- `POST /api/v1/sync/edge-events`
- idempotent receive of POS Edge `SyncEnvelope`
- raw envelope storage before future projections

## Run

Start local PostgreSQL:

```powershell
docker run --name mh-pos-cloud-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=mh_pos_cloud -p 5432:5432 -d postgres:16
```

```powershell
cd cloud-backend
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go mod tidy
go test ./...
go run ./cmd/cloud-api
```

Defaults:

```text
CLOUD_HTTP_ADDR=:8090
CLOUD_POSTGRES_MIGRATIONS_DIR=migrations/postgres
```

`CLOUD_POSTGRES_DSN` is required.

## Local Receiver Smoke Test

```powershell
Invoke-RestMethod http://localhost:8090/health
..\scripts\send-cloud-test-envelope.ps1 -ReplayTwice
```

To replay an envelope using IDs from the POS demo bootstrap:

```powershell
$demo = ..\scripts\bootstrap-pos-demo.ps1
..\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
```

Minimal curl-equivalent body for `POST /api/v1/sync/edge-events`:

```powershell
$body = @{
  version = "1"
  event_id = "demo-cloud-replay-event-1"
  command_id = "demo-cloud-replay-command-1"
  event_type = "OrderCreated"
  aggregate_type = "Order"
  aggregate_id = "demo-order-cloud-1"
  restaurant_id = "demo-restaurant"
  device_id = "demo-edge-node-1"
  shift_id = "demo-shift-cloud-1"
  occurred_at = "2026-05-07T09:00:00Z"
  payload = @{
    origin = "edge_device"
    data = @{
      id = "demo-order-cloud-1"
      edge_order_id = "demo-edge-order-cloud-1"
      restaurant_id = "demo-restaurant"
      device_id = "demo-edge-node-1"
      shift_id = "demo-shift-cloud-1"
      status = "open"
      table_name = "A1"
      guest_count = 2
      opened_at = "2026-05-07T09:00:00Z"
      created_at = "2026-05-07T09:00:00Z"
      updated_at = "2026-05-07T09:00:00Z"
    }
  }
} | ConvertTo-Json -Depth 8

Invoke-RestMethod -Method Post http://localhost:8090/api/v1/sync/edge-events -ContentType "application/json" -Body $body
Invoke-RestMethod -Method Post http://localhost:8090/api/v1/sync/edge-events -ContentType "application/json" -Body $body
```

Duplicate replay returns the same stable ack. Cloud currently stores raw accepted envelopes; full Cloud projections and production sender worker are out of scope for this slice.

## Local E2E Prototype: получить pairing code и войти в POS UI

implemented now: Cloud participates in the local prototype as the idempotent envelope receiver.

1. Start Cloud with PostgreSQL:

```powershell
cd cloud-backend
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

2. After POS bootstrap, verify replay with real IDs:

```powershell
$demo = ..\scripts\bootstrap-pos-demo.ps1
..\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
```

out of scope: POS outbox is not automatically delivered to Cloud by a production sender worker.

## Test

```powershell
cd cloud-backend
go test ./...
```

The default tests use the in-memory repository for service and HTTP replay checks. PostgreSQL runtime storage is implemented in `internal/cloudsync/infra/postgres` and initialized by `migrations/postgres`.

## Contract

See `../docs/sync/edge-cloud-contracts-v1.md`.
