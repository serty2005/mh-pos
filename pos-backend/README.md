# MyHoReCa POS Edge Backend

POS Edge Backend - локальный JSON API сервис на Go + SQLite для кассового узла. Он должен работать offline, сохранять критические операции локально и писать `local_event_log` + `pos_sync_outbox` в той же транзакции, что и бизнес-изменение.

## Architecture Lock v1.3

Целевая финансовая модель проекта:

```text
Order -> Precheck -> Payment -> Check
```

`Precheck` - рабочий финансовый snapshot для гостя. `Check` - только финальный неизменяемый расчетный документ после полной оплаты precheck.

Текущее состояние кода: backend выполняет runtime flow `Order -> Precheck -> Payment -> Check`. `IssuePrecheck` создает issued precheck и переводит order в `locked`; публичный `CancelPrecheck` требует manager employee id, PIN и reason, проверяет локальный PBKDF2 `pin_hash`, пишет `manager_override_audit` и возвращает unpaid active issued precheck в `open`; payment capture идет через `precheck_id`, поддерживает partial payments и создает final `Check` только после полной оплаты. Backend также включает строгую PIN auth/session/logout foundation, Edge Node pairing, client device auto-registration, actor/session/node/client metadata, halls/tables API, read endpoint текущего активного заказа по столу, order line quantity/void API и рассчитанные backend-ом order totals в `GET /api/v1/orders/{id}`. Публичные compatibility endpoints для старой check/payment модели удалены.

Граница auth/device:

- operator/business flows требуют active employee session, `actor_employee_id`, `session_id`, matching `client_device_id` и permissions там, где нужны;
- system/device flows (`sync`, `system/pair`, pairing status, будущие diagnostics/hardware callbacks) не требуют employee session;
- lock screen = backend logout через `POST /api/v1/auth/logout`; session становится `revoked`, новый PIN login создает новую session;
- `node_device_id` обозначает POS Edge Backend / Edge Node и приходит через pairing;
- `client_device_id` обозначает frontend-клиент, в MVP генерируется `pos-ui` и auto-registers на Edge;
- `device_id` остается domain/storage field для POS Edge node identity в operational payload; новый transport contract использует явные `node_device_id` и `client_device_id`.

Sync/outbox foundation поддерживает retry-safe состояние очереди: `pending`, `processing`, `sent`, `failed`, `suspended`, локальный `sequence_no`, явный `sync_direction`, attempts/retry metadata, processing locks, stale lock reclaim на app-level и manual retry failed/suspended. implemented now: POS Edge запускает background sender worker, который claim'ит Edge -> Cloud operational rows, отправляет `SyncEnvelope` в Cloud, делает idempotent resend, automatic retry с exponential backoff, crash recovery через stale lock reclaim и direction gate для Cloud-managed/configuration событий.

implemented now: master/reference/configuration data является Cloud-owned. POS Edge хранит локальную read model для offline POS flow, но Edge runtime не редактирует restaurants, devices metadata, roles, employees, halls, tables, catalog, menu, recipes и inventory reference data. Application services принимают такие writes только от `cloud_sync` или `system_seed`; public HTTP Edge mutation возвращает `403 Forbidden`. Ownership matrix: `../docs/sync/directional-sync-ownership.md`.

Проект еще не был запущен в production. Реальных production БД с клиентскими данными нет, поэтому production data migration до первого запуска не требуется. SQLite clean install является canonical first-launch source of truth: активный migration path содержит один `migrations/sqlite/001_init.sql`, который сразу создает текущую runtime-схему без `payments.check_id`.

## Стек

- Go 1.26
- Modular monolith, Clean Architecture, DDD-lite
- SQLite с `modernc.org/sqlite`
- HTTP JSON API на `chi`
- Docker Compose с именованным SQLite volume

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

## SQLite First Launch Schema

`MigrateDir` применяет canonical `001_init.sql` на чистую БД. В этой стартовой схеме сразу присутствуют `prechecks`, `payments.precheck_id`, `auth_sessions` со status `active/revoked`, `edge_node_identity`, `client_devices`, `halls`, `tables`, `orders.table_id`, Cloud -> Edge metadata columns на master tables, `cloud_master_sync_state`, retry-safe поля `pos_sync_outbox` (`sequence_no`, `sync_direction`, `attempts`, `next_retry_at`, `locked_at`, `locked_by`, `sent_at`, `last_error`), actor/session/node/client metadata (`node_device_id`, `client_device_id`, `actor_employee_id`, `session_id`) в `local_event_log`, `pos_sync_outbox` и `SyncEnvelope`, `local_event_log.command_id`, `manager_override_audit`, constraints precheck lifecycle и outbox. Историческая dev-цепочка миграций не является обязательной частью первого пилотного запуска.

Write transactions в POS Edge открываются через `BEGIN IMMEDIATE`, чтобы writer lock бралась в начале транзакционного use case.

## Запуск локально на Windows

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
$env:POS_SYNC_SENDER_ENABLED="true"
$env:POS_CLOUD_SYNC_URL="http://localhost:8090/api/v1/sync/edge-events"
$env:POS_SYNC_SENDER_BATCH_SIZE="25"
$env:POS_SYNC_SENDER_POLL_INTERVAL="2s"
$env:POS_SYNC_SENDER_RECLAIM_AFTER="5m"
$env:POS_SYNC_SENDER_SEND_TIMEOUT="10s"
$env:POS_DEV_TOOLS="1" # только для локального demo bootstrap
```

VSCode setup: открой папку `pos-backend`, установи официальный Go extension, выполни `Go: Install/Update Tools`, затем используй integrated terminal для `go test ./...` и `go run ./cmd/pos-edge`.

## Docker

```powershell
docker compose up --build
```

SQLite хранится в Docker volume `pos_edge_sqlite`. API доступен на `http://localhost:8080`.

## API Smoke Test

implemented now: локальный demo bootstrap доступен только при `POS_DEV_TOOLS=1`.

```powershell
Invoke-RestMethod http://localhost:8080/health
..\scripts\bootstrap-pos-demo.ps1
```

Bootstrap создает `Demo Bistro`, paired Edge Node `demo-edge-node-1`, cashier/manager roles, сотрудников с PIN `1111` и `2222`, зал, столы и несколько menu items. Ответ содержит `pairing_code` и `manager_employee_id` для ручного UI flow.

Проверка PIN login после bootstrap:

```powershell
$demo = ..\scripts\bootstrap-pos-demo.ps1
$clientDeviceID = [guid]::NewGuid().ToString()
Invoke-RestMethod -Method Post http://localhost:8080/api/v1/auth/pin-login -ContentType "application/json" -Body (@{
  node_device_id = $demo.node_device_id
  client_device_id = $clientDeviceID
  pin = "1111"
} | ConvertTo-Json)
```

Operational sync endpoints:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=20
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=20
Invoke-RestMethod http://localhost:8080/api/v1/sync/status
Invoke-RestMethod -Method Post http://localhost:8080/api/v1/sync/retry-failed
```

Для выпуска precheck используй `POST /api/v1/orders/{id}/precheck`, для payment capture - `POST /api/v1/prechecks/{id}/payments`.

## Доступные API Endpoints

Финансовые endpoints:

- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/prechecks/{id}`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/orders/{id}/prechecks`
- `GET /api/v1/checks/{id}`

Auth/device и POS UI endpoints:

- `GET /api/v1/system/pairing-status`
- `POST /api/v1/system/pair`
- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`
- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/catalog/items`
- `GET /api/v1/menu/items`
- `POST /api/v1/orders`
- `GET /api/v1/orders/current?table_id=...`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `GET /api/v1/shifts/current`
- `POST /api/v1/shifts/open`
- `POST /api/v1/shifts/{id}/close`
- `GET /api/v1/cash-sessions/current`
- `POST /api/v1/cash-sessions/open`
- `POST /api/v1/cash-sessions/{id}/close`
- `POST /api/v1/dev/bootstrap-demo` dev/local only, требует `POS_DEV_TOOLS=1`

Master-data write endpoints for restaurants/devices/roles/employees/halls/tables/catalog/menu are implemented as application-layer seed/cloud-sync write use cases, but public Edge runtime HTTP calls to them return `403 Forbidden`. Use `POST /api/v1/dev/bootstrap-demo` for local demo bootstrap until production Cloud -> Edge provisioning is implemented.

Operational sync endpoints: `GET /api/v1/sync/outbox?limit=50`, `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated`, `GET /api/v1/sync/status`, `POST /api/v1/sync/retry-failed`. `retry-failed` не отправляет данные в Cloud и не меняет business state; он только возвращает `failed`/`suspended` outbox rows в `pending`.

POS UI package: `../pos-ui` содержит Vue 3 + Quasar shell и рабочий POS Terminal Core на `/pos` для single-terminal cashier flow. См. `pos-ui/README.md`.

## Локальный E2E Prototype: получить pairing code и войти в POS UI

implemented now: `POST /api/v1/dev/bootstrap-demo` доступен только для dev/local и требует `POS_DEV_TOOLS=1`.

```powershell
cd pos-backend
$env:POS_DEV_TOOLS="1"
go run ./cmd/pos-edge
```

Из корня репозитория:

```powershell
$demo = .\scripts\bootstrap-pos-demo.ps1
$demo.pairing_code
```

Возвращенный `pairing_code` имеет формат `MHPOS:<restaurant_id>:<node_device_id>` и принимается `POST /api/v1/system/pair` и POS UI `/pair`. Cashier PIN `1111` выполняет вход через `POST /api/v1/auth/pin-login` с возвращенным `node_device_id`.

Проверь локальные sync endpoints:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/sync/status
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=10
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=10
```

implemented now: production-like sync sender worker включен по умолчанию и отправляет operational Edge -> Cloud events в `POS_CLOUD_SYNC_URL`. Для изолированной локальной отладки установи `POS_SYNC_SENDER_ENABLED=false`.

Проверка Cloud PostgreSQL после runtime POS flow:

```powershell
docker exec -it mh-pos-cloud-postgres psql -U postgres -d mh_pos_cloud -c "select event_type, count(*) from cloud_operational_events group by event_type order by event_type;"
```

## Проверки

```powershell
go test ./...
```
