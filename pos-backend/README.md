# MyHoReCa POS Edge Backend

Foundation for the POS Edge Backend: a local JSON API service with SQLite persistence, domain invariants, `local_event_log`, and a sync outbox. This repository intentionally implements only the POS Edge Backend. POS UI, Cloud Backend, Back Office UI, PostgreSQL, fiscalization, reporting, integrations, recipes, and inventory are out of scope for this phase.

## Stack

- Go 1.26
- Modular monolith, Clean Architecture, DDD-lite
- SQLite with `modernc.org/sqlite`
- HTTP JSON API with `chi`
- Docker Compose with a named SQLite volume

## Run Locally On Windows

From `pos-backend`:

```powershell
go mod tidy
go run ./cmd/pos-edge
```

The service listens on `http://localhost:8080`.

Useful environment variables:

```powershell
$env:POS_HTTP_ADDR=":8080"
$env:POS_SQLITE_PATH="data/pos-edge.db"
$env:POS_SQLITE_MIGRATIONS_DIR="migrations/sqlite"
```

VSCode setup: open the `pos-backend` folder, install the official Go extension, run `Go: Install/Update Tools`, then use the integrated terminal for `go test ./...` and `go run ./cmd/pos-edge`.

## Docker

```powershell
docker compose up --build
```

SQLite is stored in the `pos_edge_sqlite` Docker volume. The API is available on `http://localhost:8080`.

## API Smoke Test

```powershell
curl http://localhost:8080/health
```

Create the basic data:

```powershell
$bootstrapDeviceID = "bootstrap-$env:COMPUTERNAME"

$restaurant = curl -s -X POST http://localhost:8080/api/v1/restaurants -H "Content-Type: application/json" -d "{`"device_id`":`"$bootstrapDeviceID`",`"name`":`"Demo Bistro`",`"timezone`":`"Europe/Moscow`",`"currency`":`"RUB`"}" | ConvertFrom-Json
$role = curl -s -X POST http://localhost:8080/api/v1/roles -H "Content-Type: application/json" -d "{`"device_id`":`"$bootstrapDeviceID`",`"name`":`"cashier`",`"permissions_json`":`"{\`"pos\`":true}`"}" | ConvertFrom-Json
$device = curl -s -X POST http://localhost:8080/api/v1/devices/register -H "Content-Type: application/json" -d "{`"device_id`":`"$bootstrapDeviceID`",`"restaurant_id`":`"$($restaurant.id)`",`"device_code`":`"POS-1`",`"name`":`"Main terminal`",`"type`":`"windows-pos`"}" | ConvertFrom-Json
$employee = curl -s -X POST http://localhost:8080/api/v1/employees -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"restaurant_id`":`"$($restaurant.id)`",`"role_id`":`"$($role.id)`",`"name`":`"Anna`",`"pin_hash`":`"demo-hash`"}" | ConvertFrom-Json
$catalog = curl -s -X POST http://localhost:8080/api/v1/catalog/items -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"type`":`"dish`",`"name`":`"Soup`",`"sku`":`"SOUP-001`",`"base_unit`":`"portion`"}" | ConvertFrom-Json
$menu = curl -s -X POST http://localhost:8080/api/v1/menu/items -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"catalog_item_id`":`"$($catalog.id)`",`"name`":`"Soup`",`"price`":35000,`"currency`":`"RUB`"}" | ConvertFrom-Json
```

Open a shift, create and pay an order:

```powershell
$shift = curl -s -X POST http://localhost:8080/api/v1/shifts/open -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"restaurant_id`":`"$($restaurant.id)`",`"opened_by_employee_id`":`"$($employee.id)`",`"opening_cash_amount`":100000}" | ConvertFrom-Json
$order = curl -s -X POST http://localhost:8080/api/v1/orders -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"table_name`":`"A1`",`"guest_count`":2}" | ConvertFrom-Json
curl -s -X POST "http://localhost:8080/api/v1/orders/$($order.id)/lines" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"menu_item_id`":`"$($menu.id)`",`"quantity`":2}"
$check = curl -s -X POST "http://localhost:8080/api/v1/orders/$($order.id)/check" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"discount_total`":0,`"tax_total`":0}" | ConvertFrom-Json
curl -s -X POST "http://localhost:8080/api/v1/checks/$($check.id)/payments" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"method`":`"cash`",`"amount`":$($check.total),`"currency`":`"RUB`"}"
curl -s -X POST "http://localhost:8080/api/v1/orders/$($order.id)/close" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`"}"
curl -s http://localhost:8080/api/v1/sync/outbox
curl -s "http://localhost:8080/api/v1/sync/local-events?limit=50"
curl -s "http://localhost:8080/api/v1/sync/local-events?limit=50&event_type=OrderCreated"
```

Bootstrap note: before the real POS device is registered, bootstrap writes use a stable local bootstrap id such as `bootstrap-$env:COMPUTERNAME` as `device_id`. After `/devices/register` returns the real device aggregate id, all regular POS writes should use `$device.id`.

Outbox note: `pos_sync_outbox.device_id` is always non-empty. `restaurant_id` may be `NULL` for Phase 1 global dictionaries such as roles, catalog items, and menu items because they are not restaurant-scoped yet; this is intentional and separate from the mandatory `device_id` observability contract.

Local events note: write use cases store a matching local event in `local_event_log` in the same SQLite transaction as the outbox row. The same `command_id` is stored in `local_event_log`, in `pos_sync_outbox`, and in the `SyncEnvelope` JSON payload together with `event_id`, aggregate metadata, `device_id`, optional `restaurant_id`, optional `shift_id`, and the domain payload. The read-only endpoint `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated` is for operational inspection and does not change write semantics.

## Tests

```powershell
go test ./...
```
