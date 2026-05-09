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
- system/device flows (`sync`, `system/pair`, pairing status, provisioning status/register/license, будущие diagnostics/hardware callbacks) не требуют employee session;
- lock screen = backend logout через `POST /api/v1/auth/logout`; session становится `revoked`, новый PIN login создает новую session;
- `node_device_id` обозначает POS Edge Backend / Edge Node; на чистом Edge он генерируется один раз, сохраняется локально и затем подтверждается Cloud provisioning/pairing;
- `client_device_id` обозначает frontend-клиент, в MVP генерируется `pos-ui` и auto-registers на Edge;
- `device_id` остается domain/storage field для POS Edge node identity в operational payload; новый transport contract использует явные `node_device_id` и `client_device_id`.

Sync/outbox foundation поддерживает retry-safe состояние очереди: `pending`, `processing`, `sent`, `failed`, `suspended`, локальный `sequence_no`, явный `sync_direction`, attempts/retry metadata, processing locks, stale lock reclaim на app-level и manual retry failed/suspended. Реализовано сейчас: POS Edge запускает background sender worker, который claim'ит Edge -> Cloud operational rows, отправляет `SyncEnvelope` в Cloud, делает idempotent resend, automatic retry с exponential backoff, crash recovery через stale lock reclaim и direction gate для Cloud-managed/configuration событий.

Реализовано сейчас: master/reference/configuration data является Cloud-owned. POS Edge хранит локальную read model для offline POS flow, но Edge runtime не редактирует restaurants, devices metadata, roles, employees, halls, tables, catalog, menu, recipes и inventory reference data. Cloud-authored master data применяется через `POST /api/v1/sync/master-data/snapshots` или `POST /api/v1/sync/master-data/{stream}` с origin `cloud_sync`; dev seed/admin mutation routes требуют `POS_DEV_TOOLS=1` и используют `system_seed`. Ownership matrix: `../docs/sync/directional-sync-ownership.md`.

Реализовано сейчас: production path для ресторанов, сотрудников, ролей, залов/столов, каталога и меню - Cloud CRUD API, Cloud-authored publication/package, затем Cloud -> Edge ingest. POS Edge остается offline read-model consumer; local/demo bootstrap и mutation helpers нужны только для dev/local smoke flow.

Проект еще не был запущен в production. Реальных production БД с клиентскими данными нет, поэтому production data migration до первого запуска не требуется. SQLite clean install использует managed baseline `migrations/sqlite/001_init.sql`, а старые pre-pilot БД довыравниваются ordered repair migration `002_runtime_schema_repair.sql`; runtime version/checksum metadata создается кодом startup framework.

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

`MigrateDirWithPolicy` управляет ordered SQL files в `migrations/sqlite`: `001_init.sql` задает clean baseline, `002_runtime_schema_repair.sql` довыравнивает implemented-now runtime columns для старых pre-pilot БД. В стартовой схеме сразу присутствуют `prechecks`, `payments.precheck_id`, `auth_sessions` со status `active/revoked`, `edge_node_identity`, `client_devices`, `halls`, `tables`, `orders.table_id`, Cloud -> Edge metadata columns на master tables, `cloud_master_sync_state`, retry-safe поля `pos_sync_outbox` (`sequence_no`, `sync_direction`, `attempts`, `next_retry_at`, `locked_at`, `locked_by`, `sent_at`, `last_error`), actor/session/node/client metadata (`node_device_id`, `client_device_id`, `actor_employee_id`, `session_id`) в `local_event_log`, `pos_sync_outbox` и `SyncEnvelope`, `local_event_log.command_id`, `manager_override_audit`, constraints precheck lifecycle и outbox. Ручной ad-hoc ремонт БД не является canonical path.

Реализовано сейчас: startup policy поддерживает `db_runtime_versions` и `schema_migrations`; если version table отсутствует, БД считается самой старой, перед safe schema/data upgrade существующей БД выполняется SQLite online backup artifact `.db` в `POS_SQLITE_BACKUP_DIR`, checksum drift при той же версии останавливает startup, а `DB version > MH_POS_VERSION` не поддерживается и завершается fail-fast.
Реализовано сейчас: POS Edge использует единую продуктовую версию `MH_POS_VERSION`, общую для всех модулей решения.
Реализовано сейчас: любые операции изменения схемы/структуры данных БД (создание, изменение, удаление таблиц/полей/ключей/индексов и сопутствующие DDL-действия) выполняются только программно кодом сервиса при старте через migration policy.
Реализовано сейчас: ручной путь применения SQL-скриптов для runtime-обновления БД не является поддерживаемым сценарием и не рассматривается как canonical upgrade path; после managed SQL apply POS выполняет schema verification обязательных таблиц/колонок/индексов до старта HTTP server и sync worker. Запланировано далее: административная UI-операция очистки/пересоздания SQLite для случаев коллизий, повреждения БД или неустранимого конфликта загрузки данных; операция должна требовать backup, явное подтверждение и admin/support permission.

Write transactions в POS Edge открываются через `BEGIN IMMEDIATE`, чтобы writer lock бралась в начале транзакционного use case.

## SQLite maintenance

Реализовано сейчас:

- `VACUUM`, `VACUUM INTO`, `PRAGMA optimize` и `PRAGMA wal_checkpoint(TRUNCATE)` являются явными maintenance/snapshot операциями.
- Они не запускаются автоматически на каждом startup и не выполняются внутри active write transaction.
- `VACUUM`/`VACUUM INTO` требуют явный `-Force`.
- Wrapper из корня репозитория: `.\scripts\maintain-sqlite.ps1`.

Пример:

```powershell
.\scripts\maintain-sqlite.ps1 -DatabasePath "pos-backend\data\pos-edge.db" -Optimize -WalCheckpoint
.\scripts\maintain-sqlite.ps1 -DatabasePath "pos-backend\data\pos-edge.db" -Vacuum -Force
```

## Запуск локально на Windows

Из `pos-backend`:

```powershell
go mod tidy
go run ./cmd/pos-edge
```

Сервис слушает `http://localhost:8080`.

Полезные environment variables:

```powershell
$env:POS_CONFIG_PATH="config/pos-edge.json" # optional; файл имеет приоритет над env
$env:POS_HTTP_ADDR=":8080"
$env:POS_SQLITE_PATH="data/pos-edge.db"
$env:POS_SQLITE_MIGRATIONS_DIR="migrations/sqlite"
$env:POS_SQLITE_BACKUP_DIR="data/backups"
$env:MH_POS_VERSION="0.1.1"
$env:POS_SYNC_SENDER_ENABLED="true"
$env:POS_CLOUD_SYNC_URL="http://localhost:8090" # можно также указать legacy sender endpoint .../api/v1/sync/edge-events
$env:LICENSE_SERVER_URL="http://localhost:8095"
$env:POS_SYNC_SENDER_BATCH_SIZE="25"
$env:POS_SYNC_SENDER_POLL_INTERVAL="2s"
$env:POS_SYNC_SENDER_RECLAIM_AFTER="5m"
$env:POS_SYNC_SENDER_SEND_TIMEOUT="10s"
$env:POS_DEV_TOOLS="1" # только для локального demo bootstrap/dev seed/admin master-data helpers
```

Реализовано сейчас: POS Edge также читает optional `config/pos-edge.json`; пример полного файла находится в `config/pos-edge.example.json`. Если `POS_CONFIG_PATH` задан явно, файл обязателен. Порядок приоритета: defaults -> env -> JSON-файл. Общий контракт описан в `../docs/backend/RUNTIME-CONFIG.md`.

VSCode setup: открой папку `pos-backend`, установи официальный Go extension, выполни `Go: Install/Update Tools`, затем используй integrated terminal для `go test ./...` и `go run ./cmd/pos-edge`.

## Docker

```powershell
docker compose up --build
```

SQLite хранится в Docker volume `pos_edge_sqlite`. API доступен на `http://localhost:8080`.

## API Smoke Test

Реализовано сейчас: локальный demo bootstrap доступен только при `POS_DEV_TOOLS=1`.

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
- `GET /api/v1/system/provisioning-status`
- `POST /api/v1/system/provisioning/register-cloud`
- `POST /api/v1/system/provisioning/pair-via-license`
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
- `GET /api/v1/employee-shifts/current`
- `POST /api/v1/employee-shifts/open`
- `POST /api/v1/employee-shifts/{id}/close`
- `GET /api/v1/cash-shifts/current`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `POST /api/v1/dev/bootstrap-demo` dev/local only, требует `POS_DEV_TOOLS=1`

Cloud -> Edge master-data ingest endpoints:

- `POST /api/v1/sync/master-data/snapshots`
- `POST /api/v1/sync/master-data/{stream}`

Реализовано сейчас: supported POS Edge ingest streams: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`. Payload принимает `node_device_id`, `sync_mode` (`incremental` по умолчанию или явный `full_snapshot`), optional `full_snapshot_reason`, optional `checkpoint_token`, `cloud_version`, optional `cloud_updated_at` и stream arrays (`restaurants`, `devices`, `roles`, `employees`, `halls`, `tables`, `catalog_items`, `menu_items`). Explicit `full_snapshot` разрешен только для `terminal_restaurant_changed` или `node_role_changed`; ingest пишет master tables и `cloud_master_sync_state` в одной транзакции и не создает Edge -> Cloud outbox rows.

Cloud production delivery path:

- Cloud создает master data через `/api/v1/restaurants`, `/api/v1/roles`, `/api/v1/employees`, `/api/v1/catalog/items`, `/api/v1/menu/items`;
- Cloud создает минимальный floor через `/api/v1/halls` и `/api/v1/tables`, потому что cashier order flow требует зал и стол;
- Cloud публикует package через `POST /api/v1/restaurants/{id}/master-data/publish`;
- Cloud отдает Edge-ready payload через `GET /api/v1/restaurants/{id}/edge-nodes/{node_device_id}/master-data/snapshot`;
- POS Edge применяет payload через `POST /api/v1/sync/master-data/snapshots`.

Zero-to-Cashier provisioning path:

- Option A: при `POS_CLOUD_SYNC_URL=http://localhost:8090` Edge регистрируется в Cloud через `POST /api/v1/devices/register`, хранит локальный статус `pending_admin_approval`, polling получает assignment, скачивает snapshot и вызывает существующий local pair use case.
- Option B: UI или оператор отправляет license code в `POST /api/v1/system/provisioning/pair-via-license`; Edge resolve-ит code через `LICENSE_SERVER_URL`, сохраняет cloud config/token, скачивает snapshot, применяет master-data и переходит в статус `paired`.
- `node_device_id` не равен `client_device_id`: первый принадлежит Edge Backend и хранится в SQLite, второй принадлежит браузеру/UI и auto-registers при PIN login.

`..\scripts\cloud-masterdata-e2e.ps1` проверяет этот путь локально: Cloud CRUD -> publish -> POS Edge ingest -> POS catalog/menu read -> PIN login Cloud-created employee.

Вне текущего объема: direct POS Edge apply for `currencies` stream. Cloud backend already owns canonical ISO 4217 currency reference/provisioning, but POS Edge currently validates currencies from its local canonical catalog rather than importing currency packages through master-data ingest.

Master-data write endpoints for restaurants/devices/roles/employees/halls/tables/catalog/menu are implemented as application-layer seed/cloud-sync write use cases. HTTP mutation routes are dev-only seed/admin helpers behind `POS_DEV_TOOLS=1`; normal POS runtime should use read endpoints and the Cloud -> Edge ingest endpoints above. Реализовано сейчас: эти endpoints не являются production API для справочников и не должны использоваться как onboarding ресторана.

Operational sync endpoints: `GET /api/v1/sync/outbox?limit=50`, `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated`, `GET /api/v1/sync/status`, `POST /api/v1/sync/retry-failed`. `retry-failed` не отправляет данные в Cloud и не меняет business state; он только возвращает `failed`/`suspended` outbox rows в `pending`.

## Error contract

Реализовано сейчас:

- Все API ошибки возвращаются в безопасном envelope `{ "error": { "code", "message_key", "details", "correlation_id" } }`.
- Internal cause, SQL details и panic stack пишутся только в backend logs.
- `X-Error-Code` дублирует stable `code` для audit middleware.
- Revoked session возвращает `401 SESSION_REVOKED`.
- Permission deny возвращает `403 PERMISSION_DENIED`.
- Wrong client/session context возвращает `403 SESSION_CONTEXT_MISMATCH`.
- PIN, manager PIN и PIN hash не возвращаются в error payloads и не должны логироваться.

Каталог ошибок: `../docs/backend/POS-ERROR-CATALOG.md`.

POS UI package: `../pos-ui` содержит Vue 3 + Quasar shell и рабочий POS Terminal Core на `/pos` для single-terminal cashier flow. См. `pos-ui/README.md`.

## Локальный E2E Prototype: получить pairing code и войти в POS UI

Реализовано сейчас: `POST /api/v1/dev/bootstrap-demo` доступен только для dev/local и требует `POS_DEV_TOOLS=1`.
Реализовано сейчас: Cloud API заменяет bootstrap scripts как backend path создания ресторана, сотрудников, ролей, PIN credential, каталога и меню. Запланировано далее: `cloud-ui` будет построен поверх этого Cloud API.

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
$login = Invoke-RestMethod -Method Post http://localhost:8080/api/v1/auth/pin-login -ContentType "application/json" -Body (@{
  node_device_id = $demo.node_device_id
  client_device_id = "dev-readme-client"
  pin = "2222"
} | ConvertTo-Json)
$headers = @{
  "X-Node-Device-ID" = $demo.node_device_id
  "X-Client-Device-ID" = "dev-readme-client"
  "X-Session-ID" = $login.session.id
  "X-Actor-Employee-ID" = $login.actor.employee_id
}
Invoke-RestMethod http://localhost:8080/api/v1/sync/status -Headers $headers
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=10 -Headers $headers
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=10 -Headers $headers
```

Реализовано сейчас: production-like sync sender worker включен по умолчанию и отправляет operational Edge -> Cloud events в `POS_CLOUD_SYNC_URL`. Для изолированной локальной отладки установи `POS_SYNC_SENDER_ENABLED=false`.

Проверка Cloud PostgreSQL после runtime POS flow:

```powershell
docker exec -it mh-pos-cloud-postgres psql -U postgres -d mh_pos_cloud -c "select event_type, count(*) from cloud_operational_events group by event_type order by event_type;"
```

## Проверки

```powershell
go test ./...
```
