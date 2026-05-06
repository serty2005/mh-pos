# MyHoReCa POS Edge Backend

POS Edge Backend - локальный JSON API сервис на Go + SQLite для кассового узла. Он должен работать offline, сохранять критические операции локально и писать `local_event_log` + `pos_sync_outbox` в той же транзакции, что и бизнес-изменение.

## Architecture Lock v1.3

Целевая финансовая модель проекта:

```text
Order -> Precheck -> Payment -> Check
```

`Precheck` - рабочий финансовый snapshot для гостя. `Check` - только финальный неизменяемый расчетный документ после полной оплаты precheck.

Текущее состояние кода: backend выполняет runtime flow `Order -> Precheck -> Payment -> Check`. `IssuePrecheck` создает issued precheck и переводит order в `locked`; публичный `CancelPrecheck` требует manager employee id, PIN и reason, проверяет локальный PBKDF2 `pin_hash`, пишет `manager_override_audit` и возвращает unpaid active issued precheck в `open`; payment capture идет через `precheck_id`, поддерживает partial payments и создает final `Check` только после полной оплаты. Legacy `POST /api/v1/checks/{id}/payments` отключен и не обходит precheck flow.

Sync/outbox foundation уже поддерживает retry-safe состояние очереди: `pending`, `processing`, `sent`, `failed`, `suspended`, локальный `sequence_no`, attempts/retry metadata, processing locks, stale lock reclaim на app-level и manual retry failed/suspended. Реальный Cloud sender/worker в этой итерации не реализован.

Проект еще не был запущен в production. Реальных production БД с клиентскими данными нет, поэтому production data migration до первого запуска не требуется. Изменения схемы v1.3 нужно проектировать как first-launch schema.

## Stack

- Go 1.26
- Modular monolith, Clean Architecture, DDD-lite
- SQLite with `modernc.org/sqlite`
- HTTP JSON API with `chi`
- Docker Compose with a named SQLite volume

## SQLite Runtime Contract

При старте POS Edge backend открывает SQLite fail-fast: выставляет и затем проверяет фактические runtime параметры.

Обязательный baseline:

- `sqlite_version()` доступен;
- functional minimum: `SQLite >= 3.37.0`;
- production WAL pilot baseline: `SQLite >= 3.51.3` или pinned backport `3.50.7/3.44.6`;
- `PRAGMA journal_mode = WAL`;
- `PRAGMA synchronous = NORMAL`;
- `PRAGMA foreign_keys = ON`;
- `PRAGMA busy_timeout >= 5000`.

Если runtime не соответствует baseline, backend завершается до применения migrations и запуска HTTP server.

## Запуск Локально На Windows

Из `pos-backend`:

```powershell
go mod tidy
go run ./cmd/pos-edge
```

Сервис слушает `http://localhost:8080`.

Полезные environment variables:

```powershell
$env:POS_HTTP_ADDR=":8080"
$env:POS_SQLITE_PATH="data/pos-edge.db"
$env:POS_SQLITE_MIGRATIONS_DIR="migrations/sqlite"
```

VSCode setup: открой папку `pos-backend`, установи официальный Go extension, выполни `Go: Install/Update Tools`, затем используй integrated terminal для `go test ./...` и `go run ./cmd/pos-edge`.

## Docker

```powershell
docker compose up --build
```

SQLite хранится в Docker volume `pos_edge_sqlite`. API доступен на `http://localhost:8080`.

## API Smoke Test

Этот smoke test проверяет текущий публичный `Order -> Precheck -> Payment -> Check` slice.

```powershell
curl http://localhost:8080/health
```

Создать базовые данные:

```powershell
$bootstrapDeviceID = "bootstrap-$env:COMPUTERNAME"

$restaurant = curl -s -X POST http://localhost:8080/api/v1/restaurants -H "Content-Type: application/json" -d "{`"device_id`":`"$bootstrapDeviceID`",`"name`":`"Demo Bistro`",`"timezone`":`"Europe/Moscow`",`"currency`":`"RUB`"}" | ConvertFrom-Json
$role = curl -s -X POST http://localhost:8080/api/v1/roles -H "Content-Type: application/json" -d "{`"device_id`":`"$bootstrapDeviceID`",`"name`":`"cashier`",`"permissions_json`":`"{\`"pos\`":true}`"}" | ConvertFrom-Json
$device = curl -s -X POST http://localhost:8080/api/v1/devices/register -H "Content-Type: application/json" -d "{`"device_id`":`"$bootstrapDeviceID`",`"restaurant_id`":`"$($restaurant.id)`",`"device_code`":`"POS-1`",`"name`":`"Main terminal`",`"type`":`"windows-pos`"}" | ConvertFrom-Json
$employee = curl -s -X POST http://localhost:8080/api/v1/employees -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"restaurant_id`":`"$($restaurant.id)`",`"role_id`":`"$($role.id)`",`"name`":`"Anna`",`"pin_hash`":`"demo-hash`"}" | ConvertFrom-Json
$catalog = curl -s -X POST http://localhost:8080/api/v1/catalog/items -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"type`":`"dish`",`"name`":`"Soup`",`"sku`":`"SOUP-001`",`"base_unit`":`"portion`"}" | ConvertFrom-Json
$menu = curl -s -X POST http://localhost:8080/api/v1/menu/items -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"catalog_item_id`":`"$($catalog.id)`",`"name`":`"Soup`",`"price`":35000,`"currency`":`"RUB`"}" | ConvertFrom-Json
```

Проверить текущий публичный precheck flow:

```powershell
$shift = curl -s -X POST http://localhost:8080/api/v1/shifts/open -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"restaurant_id`":`"$($restaurant.id)`",`"opened_by_employee_id`":`"$($employee.id)`",`"opening_cash_amount`":100000}" | ConvertFrom-Json
$cashSession = curl -s -X POST http://localhost:8080/api/v1/cash-sessions/open -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"restaurant_id`":`"$($restaurant.id)`",`"opened_by_employee_id`":`"$($employee.id)`",`"opening_cash_amount`":100000}" | ConvertFrom-Json
curl -s -X POST http://localhost:8080/api/v1/cash-drawer-events -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"created_by_employee_id`":`"$($employee.id)`",`"event_type`":`"cash_count`",`"amount`":100000}"
$order = curl -s -X POST http://localhost:8080/api/v1/orders -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"table_name`":`"A1`",`"guest_count`":2}" | ConvertFrom-Json
curl -s -X POST "http://localhost:8080/api/v1/orders/$($order.id)/lines" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"menu_item_id`":`"$($menu.id)`",`"quantity`":2}"
$precheck = curl -s -X POST "http://localhost:8080/api/v1/orders/$($order.id)/precheck" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`"}" | ConvertFrom-Json
curl -s "http://localhost:8080/api/v1/prechecks/$($precheck.id)"
curl -s "http://localhost:8080/api/v1/orders/$($order.id)/prechecks"
$payment = curl -s -X POST "http://localhost:8080/api/v1/prechecks/$($precheck.id)/payments" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"method`":`"cash`",`"amount`":$($precheck.total),`"currency`":`"RUB`"}" | ConvertFrom-Json
curl -s -X POST "http://localhost:8080/api/v1/cash-sessions/$($cashSession.id)/close" -H "Content-Type: application/json" -d "{`"device_id`":`"$($device.id)`",`"closed_by_employee_id`":`"$($employee.id)`",`"closing_cash_amount`":100000}"
curl -s http://localhost:8080/api/v1/sync/outbox
curl -s http://localhost:8080/api/v1/sync/status
curl -s -X POST http://localhost:8080/api/v1/sync/retry-failed -H "Content-Type: application/json" -d "{}"
curl -s "http://localhost:8080/api/v1/sync/local-events?limit=50"
curl -s "http://localhost:8080/api/v1/sync/local-events?limit=50&event_type=PrecheckIssued"
```

Bootstrap note: до регистрации реального POS device bootstrap writes используют стабильный локальный bootstrap id вроде `bootstrap-$env:COMPUTERNAME` как `device_id`. После `/devices/register` все regular POS writes должны использовать `$device.id`.

Outbox note: `pos_sync_outbox.device_id` всегда непустой. `restaurant_id` может быть `NULL` для Phase 1 global dictionaries вроде roles, catalog items и menu items, потому что они пока не restaurant-scoped. Это намеренно и отдельно от обязательного `device_id` observability contract. Outbox rows получают монотонный локальный `sequence_no` и статусы `pending`, `processing`, `sent`, `failed`, `suspended`; retry-safe metadata включает `attempts`, `next_retry_at`, `locked_at`, `locked_by`, `sent_at`, `last_error`. App-level foundation умеет claim pending batch по `sequence_no` с учетом `next_retry_at`, reclaim stale `processing` locks, переводить failed/suspended обратно в pending и агрегировать sync status. Cloud sender/worker пока не реализован.

Local events note: write use cases сохраняют matching local events в `local_event_log` в той же SQLite transaction, что и outbox rows. Один business command может породить несколько domain events с одним `command_id` (например full payment пишет `PaymentCaptured`, `CheckCreated`, `OrderClosed`); `command_id` хранится в `local_event_log`, в `pos_sync_outbox` и в каждом `SyncEnvelope` JSON payload вместе с `event_id`, aggregate metadata, `device_id`, optional `restaurant_id`, optional `shift_id` и domain payload. Read-only endpoint `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated` нужен для operational inspection и не меняет write semantics.

Financial foundation note: текущий `CapturePayment` принимает `precheck_id`, сохраняет `payments` и первую строку `payment_attempts` в той же transaction, что и precheck paid-total update, `local_event_log` и `pos_sync_outbox`. При partial payment precheck остается `issued`; при полной оплате в той же transaction создается final `Check`, precheck становится terminal `closed`, order закрывается, а события `PaymentCaptured`, `CheckCreated` и `OrderClosed` пишутся в local event log и outbox с тем же `command_id`.

Precheck foundation note: в схеме есть `prechecks` lifecycle foundation с `version`, `supersedes_precheck_id`, `paid_total`, terminal status `cancelled/superseded/closed`, domain model, repository interface/SQLite implementation и app service. `IssuePrecheck` транзакционно создает precheck, переводит order в `locked`, пишет `local_event_log` и `pos_sync_outbox`, и доступен публично через `POST /api/v1/orders/{id}/precheck`. `GET /api/v1/prechecks/{id}` и `GET /api/v1/orders/{id}/prechecks` читают prechecks. `CancelPrecheck` доступен через `POST /api/v1/prechecks/{id}/cancel`, требует manager override, запрещает cancel при `paid_total > 0`, возвращает order в `open`, пишет `manager_override_audit`, `PrecheckCancelled`, local event и outbox в одной transaction. PIN не пишется в audit/outbox/local event и не отправляется в Cloud.

Cash session endpoints: `POST /api/v1/cash-sessions/open`, `POST /api/v1/cash-sessions/{id}/close`, `GET /api/v1/cash-sessions/current?device_id=...`, `POST /api/v1/cash-drawer-events`. Закрытие смены запрещено, пока на device есть active cash session; cash session нужно закрыть до `POST /api/v1/shifts/{id}/close`.

Operational sync endpoints: `GET /api/v1/sync/outbox?limit=50`, `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated`, `GET /api/v1/sync/status`, `POST /api/v1/sync/retry-failed`. `retry-failed` не отправляет данные в Cloud и не меняет business state; он только возвращает `failed`/`suspended` outbox rows в `pending`.

## Текущие Financial Endpoints

См. `internal/pos/api/router.go`. На текущий момент financial endpoints:

- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/prechecks/{id}`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/orders/{id}/prechecks`
- `POST /api/v1/orders/{id}/check`
- `POST /api/v1/checks/{id}/payments`
- `GET /api/v1/checks/{id}`

`POST /api/v1/orders/{id}/check` оставлен как deprecated dev alias и вызывает `IssuePrecheck`, не создает legacy check напрямую. `POST /api/v1/checks/{id}/payments` оставлен только как deprecated compatibility route и возвращает conflict с указанием использовать `/api/v1/prechecks/{id}/payments`. `GET /api/v1/checks/{id}` читает только final checks, которые появляются после полной оплаты precheck.

## Tests

```powershell
go test ./...
```
