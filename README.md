# MyHoReCa POS / RMS

Монорепозиторий edge-first POS/RMS платформы для HoReCa.

Текущий фокус репозитория - перевод уже существующего POS Edge foundation к Architecture Lock v1.3. Целевая финансовая модель v1.3:

```text
Order -> Precheck -> Payment -> Check
```

`Precheck` - рабочий финансовый snapshot для гостя. `Check` - только финальный неизменяемый расчетный документ после полной оплаты precheck.

Важно: проект еще не был запущен в production. Реальных production БД с клиентскими данными нет, поэтому production data migration до первого запуска не требуется. Изменения v1.3 проектируются как first-launch schema/logic.

## Текущее состояние

Репозиторий уже содержит рабочий foundation:

- `pos-backend/` - локальный POS Edge backend на Go + SQLite;
- SQLite runtime gate для POS Edge: startup fail-fast проверяет фактические `sqlite_version()`, `journal_mode=WAL`, `synchronous=NORMAL`, `foreign_keys=ON`, `busy_timeout >= 5000`;
- `cloud-backend/` - минимальный Cloud Sync Receiver на Go + PostgreSQL;
- Cloud PostgreSQL startup schema path использует один управляемый canonical SQL file `cloud-backend/migrations/postgres/001_sync_receiver.sql`; версия/состояние фиксируются в runtime version tables;
- runtime module/database version contract включен: на старте модулей проверяются `db_runtime_versions`, re-runnable canonical SQL file, checksum tracking в `schema_migrations`, backup-before-upgrade, schema verification и fail-fast при `DB version > app version`;
- approved frontend MVP: отдельный пакет `pos-ui` на Vue 3 + TypeScript + Quasar + Vue Router + Pinia + `@tanstack/vue-query` + `vue-i18n` + Zod; `/pair`, `/login`, `/lock` и рабочий POS Terminal Core `/pos` для single-terminal cashier flow реализованы;
- `local_event_log`;
- `pos_sync_outbox`;
- `SyncEnvelope` foundation;
- PIN auth/session foundation: `POST /api/v1/auth/pin-login`, `GET /api/v1/auth/session`, `POST /api/v1/auth/logout`;
- реализовано сейчас: PIN login rate limiting returns `429 Too Many Requests` after repeated invalid attempts for the same `node_device_id + client_device_id` window;
- strict lock/logout model: UI lock или auto-lock вызывает backend logout, session становится `revoked`, новый PIN создает новую session;
- operator auth enforcement для business/operator flows: active employee session, `actor_employee_id`, `session_id`, matching `client_device_id` и permissions там, где нужны;
- system/device flows (`sync`, pairing/status, diagnostics/hardware callbacks в будущих фазах) не требуют employee session и должны авторизоваться отдельным device/system path;
- Edge Node pairing foundation: `POST /api/v1/system/pair`, `GET /api/v1/system/pairing-status`;
- identity split: `node_device_id` обозначает Edge Backend и назначается pairing flow; `client_device_id` обозначает frontend-клиент, генерируется `pos-ui` в `localStorage` и auto-registers на Edge;
- actor/session/client/node metadata в write commands, `local_event_log`, `pos_sync_outbox` и `SyncEnvelope`: `node_device_id`, `client_device_id`, `actor_employee_id`, `session_id`;
- halls/tables foundation для выбора стола в POS/Waiter UI;
- read-only endpoint активного заказа по столу для cashier terminal: `GET /api/v1/orders/current?table_id=...`;
- order line editing foundation: изменение количества и void позиции без физического удаления;
- personal employee shifts, cash shifts (`cash_sessions`) and cash drawer events;
- public precheck issue/read/list/cancel flow: `POST /api/v1/orders/{id}/precheck`, `GET /api/v1/prechecks/{id}`, `GET /api/v1/orders/{id}/prechecks`, `POST /api/v1/prechecks/{id}/cancel`;
- manager override для `CancelPrecheck`: локальная PBKDF2 PIN verification, actor permission `pos.precheck.cancel.request`, approver permission `pos.precheck.cancel`, audit trail `manager_override_audit`;
- precheck-based payment capture: `POST /api/v1/prechecks/{id}/payments`, partial payments, required open cash shift, automatic final `Check` после полной оплаты и automatic order close;
- foundation финальных чеков и оплат;
- `payment_attempts`;
- retry-safe sync outbox foundation со status/claim/retry metadata и явным `sync_direction`;
- production-like POS Edge -> Cloud sender worker с direction gate, automatic retry/backoff, stale lock reclaim и idempotent resend;
- Cloud operational event journal для принятых Edge runtime events;
- directional sync ownership foundation: Cloud owns master/reference/configuration data, Edge owns operational POS runtime data; matrix в `docs/sync/directional-sync-ownership.md`;
- Cloud -> Edge master-data ingest API on POS Edge: `POST /api/v1/sync/master-data/snapshots` and `POST /api/v1/sync/master-data/{stream}` for `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`; applies `cloud_sync` payloads transactionally without creating Edge -> Cloud outbox rows;
- operational sync endpoints для просмотра outbox, local events, aggregated status и ручного retry failed/suspended messages.

Честное состояние текущего кода: POS Edge backend уже выполняет runtime flow `Order -> Precheck -> Payment -> Check`. `IssuePrecheck` переводит order в `locked`; публичный `CancelPrecheck` требует manager employee id, PIN и reason, пишет audit trail и возвращает unpaid active issued precheck в `open`; payment capture идет через `precheck_id`, а final `Check` создается только после полной оплаты. Публичные compatibility endpoints для старой check/payment модели удалены. KDS/Waiter UI еще не готовы: backend сейчас дает только halls/tables и базовое редактирование order lines.

## Структура монорепозитория

```text
.
|-- AGENTS.md                 # архитектурные правила и быстрый вход для AI-агентов
|-- README.md                 # карта монорепозитория
|-- SPECv1.3.md               # целевая спецификация Architecture Lock v1.3
|-- ROADMAP.md                # roadmap перехода к MVP v1.3
|-- pos-backend/              # POS Edge Backend, текущая основная кодовая база
|   |-- README.md             # запуск, Docker, smoke test, текущий API и first-launch schema
|   |-- cmd/pos-edge/         # entrypoint локального POS backend сервиса
|   |-- internal/platform/    # общая инфраструктура: clock, http, idgen, sqlite, tx
|   |-- internal/pos/         # POS bounded context
|   |   |-- api/              # HTTP router и thin handlers
|   |   |-- app/              # use cases, транзакции, orchestration
|   |   |-- domain/           # доменные модели, ошибки и инварианты
|   |   |-- ports/            # интерфейсы репозиториев
|   |   `-- infra/sqlite/     # SQLite реализации репозиториев
|   |-- migrations/sqlite/    # single managed canonical SQLite init/upgrade SQL file
|   |-- docker/               # Dockerfile
|   |-- docker-compose.yml    # локальный запуск через Docker Compose
|   `-- docs/                 # отчеты и проектные документы по backend
|-- cloud-backend/            # минимальный Cloud Sync Receiver foundation
|   |-- README.md             # запуск и тесты cloud receiver
|   |-- cmd/cloud-api/        # entrypoint Cloud API
|   `-- migrations/postgres/  # single managed canonical PostgreSQL SQL file
|-- pos-ui/                   # Vue 3 + Quasar POS Terminal Core
|-- docs/sync/                # sync contracts
|-- .codex/skills/            # локальные skills для Codex
|-- pack_go_files.py          # вспомогательный скрипт упаковки Go-файлов
`-- project_dump.py           # вспомогательный скрипт дампа проекта
```

Планируемые, но еще не реализованные части монорепозитория:

- `device-adapters/` - адаптеры принтеров, терминалов и другого оборудования.
- `backoffice-ui/` - будущий web UI для управления и отчетности.

## Как работать с репозиторием

Перед изменениями прочитай:

- [AGENTS.md](AGENTS.md)
- [SPECv1.3.md](SPECv1.3.md)
- [ROADMAP.md](ROADMAP.md)

Эти документы фиксируют edge-first, offline-first, Clean Architecture, транзакции для write операций, outbox в той же транзакции и целевую модель `Order -> Precheck -> Payment -> Check`.

Для POS Edge backend:

```powershell
cd pos-backend
go mod tidy
go test ./...
go run ./cmd/pos-edge
```

Сервис по умолчанию слушает `http://localhost:8080`.

Для Cloud Sync Receiver:

```powershell
cd cloud-backend
go test ./...
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

Для POS UI:

```powershell
cd pos-ui
npm install
npm run dev
```

Dev server слушает `http://localhost:5173` и ходит в POS Edge backend `http://localhost:8080/api/v1` по умолчанию.

## Локальный E2E Prototype Quickstart

реализовано сейчас: локально можно поднять минимальную связку `pos-ui -> pos-backend -> cloud-backend` и пройти cashier flow вручную.

1. Подними PostgreSQL для Cloud:

```powershell
docker run --name mh-pos-cloud-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=mh_pos_cloud -p 5432:5432 -d postgres:16
```

2. Запусти Cloud receiver:

```powershell
cd cloud-backend
go mod tidy
go test ./...
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

3. Запусти POS Edge backend с dev tools:

```powershell
cd pos-backend
go mod tidy
go test ./...
$env:POS_DEV_TOOLS="1"
go run ./cmd/pos-edge
```

4. В новом терминале из корня репозитория создай demo данные:

```powershell
.\scripts\bootstrap-pos-demo.ps1
```

Скрипт вызывает dev/local endpoint `POST /api/v1/dev/bootstrap-demo` и печатает:

- `Pairing code`: `MHPOS:<restaurant_id>:demo-edge-node-1`
- `Cashier PIN`: `1111`
- `Manager PIN`: `2222`
- `Manager employee`: employee id для cancel precheck override

5. Запусти POS UI:

```powershell
cd pos-ui
npm install
npm run dev
```

6. Открой `http://localhost:5173` и пройди ручной сценарий:

```text
pairing -> login -> open personal shift -> open cash shift -> select hall/table -> create order -> add lines -> change quantity -> void line -> issue precheck -> cancel unpaid precheck with manager override -> issue precheck again -> capture payment -> final check -> close cash shift -> close personal shift -> lock/logout
```

7. Проверь POS sync state:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=20

$managerLogin = Invoke-RestMethod -Method Post http://localhost:8080/api/v1/auth/pin-login -ContentType "application/json" -Body (@{
  node_device_id = "demo-edge-node-1"
  client_device_id = "sync-cli-1"
  pin = "2222"
} | ConvertTo-Json)

$syncHeaders = @{
  "X-Node-Device-ID" = "demo-edge-node-1"
  "X-Client-Device-ID" = "sync-cli-1"
  "X-Session-ID" = $managerLogin.session.id
  "X-Actor-Employee-ID" = $managerLogin.actor.employee_id
}

Invoke-RestMethod -Headers $syncHeaders http://localhost:8080/api/v1/sync/outbox?limit=20
Invoke-RestMethod -Headers $syncHeaders http://localhost:8080/api/v1/sync/status
Invoke-RestMethod -Method Post -Headers $syncHeaders http://localhost:8080/api/v1/sync/retry-failed
```

8. Проверь Cloud receiver и автоматический POS sender:

```powershell
Invoke-RestMethod http://localhost:8090/health
Invoke-RestMethod -Headers $syncHeaders http://localhost:8080/api/v1/sync/status
```

Runtime POS actions автоматически перемещают operational outbox rows в Cloud, когда `POS_SYNC_SENDER_ENABLED=true`, а `POS_CLOUD_SYNC_URL` указывает на Cloud. Configuration/bootstrap rows, которые не являются допустимыми Edge -> Cloud operational events, имеют `sync_direction = cloud_to_edge` или `local_only` и помечаются `suspended` с явной sync-direction причиной.

Cloud-authored master data for local/dev checks can be applied directly to POS Edge without running a full Cloud master backend:

```powershell
Invoke-RestMethod -Method Post http://localhost:8080/api/v1/sync/master-data/catalog -ContentType "application/json" -Body (@{
  node_device_id = "demo-edge-node-1"
  sync_mode = "incremental"
  cloud_version = 1
  catalog_items = @(@{
    id = "cloud-demo-dish-1"
    type = "dish"
    name = "Cloud Demo Dish"
    sku = "CLOUD-DEMO-DISH"
    base_unit = "portion"
    active = $true
  })
} | ConvertTo-Json -Depth 5)
```

This endpoint is Cloud -> Edge ingest, not a POS runtime mutation route. It updates master tables and `cloud_master_sync_state`; it does not enqueue Edge -> Cloud outbox rows.

Проверка PostgreSQL:

```powershell
docker exec -it mh-pos-cloud-postgres psql -U postgres -d mh_pos_cloud -c "select event_type, count(*) from cloud_operational_events group by event_type order by event_type;"
docker exec -it mh-pos-cloud-postgres psql -U postgres -d mh_pos_cloud -c "select event_type, count(*) from cloud_edge_event_receipts group by event_type order by event_type;"
```

`.\scripts\send-cloud-test-envelope.ps1 -ReplayTwice` по-прежнему проверяет duplicate replay напрямую против Cloud. `.\scripts\dev-smoke.ps1` выполняет health checks, POS demo bootstrap, POS sync endpoint checks и Cloud envelope replay, но не стартует серверы за тебя.

реализовано сейчас: `.\scripts\start-and-test-all.ps1` запускает Cloud, POS Edge и UI в отдельных видимых PowerShell окнах, пишет PID file `.dev-stack-pids.json`, сохраняет логи в `logs/dev-stack/`, проверяет порты `8090/8080/5173`, показывает tail логов при health timeout и выполняет authenticated sync smoke через demo manager session. Остановка выполняется через `.\scripts\stop-and-test-all.ps1`, который использует PID file и завершает process tree без небезопасного wildcard kill.

## Локальный E2E Prototype: получить pairing code и войти в POS UI

реализовано сейчас: local developer flow использует реальные POS backend endpoints и реальный MVP pairing code.

1. Запусти Cloud:

```powershell
cd cloud-backend
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

2. Запусти POS с dev tools:

```powershell
cd pos-backend
$env:POS_DEV_TOOLS="1"
go run ./cmd/pos-edge
```

3. Из корня репозитория получи demo credentials:

```powershell
.\scripts\bootstrap-pos-demo.ps1
```

Используй возвращенный `pairing_code` на `http://localhost:5173/pair`, затем войди на `/login` с cashier PIN `1111`. Скрипт также возвращает `restaurant_id`, `node_device_id`, employee ids, `hall_id`, `table_ids` и `menu_item_ids`.

4. Проверь Cloud replay с реальными bootstrap IDs:

```powershell
$demo = .\scripts\bootstrap-pos-demo.ps1
.\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
```

5. Проверь локальное POS sync state:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=10

$managerLogin = Invoke-RestMethod -Method Post http://localhost:8080/api/v1/auth/pin-login -ContentType "application/json" -Body (@{
  node_device_id = "demo-edge-node-1"
  client_device_id = "sync-cli-2"
  pin = "2222"
} | ConvertTo-Json)

$syncHeaders = @{
  "X-Node-Device-ID" = "demo-edge-node-1"
  "X-Client-Device-ID" = "sync-cli-2"
  "X-Session-ID" = $managerLogin.session.id
  "X-Actor-Employee-ID" = $managerLogin.actor.employee_id
}

Invoke-RestMethod -Headers $syncHeaders http://localhost:8080/api/v1/sync/status
Invoke-RestMethod -Headers $syncHeaders http://localhost:8080/api/v1/sync/outbox?limit=10
```

реализовано сейчас: POS Edge автоматически доставляет Edge -> Cloud operational outbox rows в локальный Cloud receiver, когда sender включен. Недоступность Cloud не блокирует POS runtime writes.

## Основные контуры

### POS Edge Backend

Где лежит: `pos-backend/`

Назначение:

- локальное хранение POS данных в SQLite;
- JSON API для POS UI;
- доменные инварианты заказов, смен, cash shifts и текущего financial foundation;
- edge foundation для `local_event_log`, `SyncEnvelope` и sync outbox;
- operational access к sync outbox, local events, aggregated sync status и manual retry failed/suspended;
- financial foundation для precheck payments, final checks, `payment_attempts`, cash shifts и cash drawer events;
- foundation для будущих рецептов, склада и учета.

Текущее состояние: публичный runtime `Order -> Precheck -> Payment -> Check` включен. `Precheck` является рабочим финансовым snapshot, payment привязан к precheck, а `Check` создается автоматически только после полной оплаты.

Архитектура внутри backend:

```text
domain -> app -> ports -> infra
```

Короткое правило: domain не знает про HTTP, SQLite, `database/sql` и инфраструктуру; use cases управляют транзакциями; handlers остаются тонкими.

### Cloud Backend

Где лежит: `cloud-backend/`

Назначение:

- принимать POS Edge `SyncEnvelope`;
- выполнять idempotent receive/dedupe;
- хранить raw envelope до будущих Cloud projections.

Cloud не является зависимостью для критических POS операций: локальный кассовый узел должен продолжать работать offline.

### UI

Approved frontend MVP - отдельный пакет `pos-ui` на Vue 3 + TypeScript + Quasar + Vue Router + Pinia + `@tanstack/vue-query` + `vue-i18n` + Zod. Старые предположения про React/Vite UI считаются устаревшими. Tailwind не используется. Frontend не является source of truth и не содержит бизнес-решений.

`pos-ui` уже содержит рабочий shell и POS Terminal Core:

- `/pair` - ввод pairing code и вызов `POST /api/v1/system/pair`;
- `/login` - реальный `POST /api/v1/auth/pin-login`;
- `/lock` - реальный `POST /api/v1/auth/logout` и очистка локального session state;
- `/pos` - cashier surface для одного терминала: личная смена сотрудника, кассовая смена, выбор зала/стола, активный заказ, позиции меню, изменение/void позиций, выпуск/отмена пречека, cash/trusted card payment и отображение final check.

Identity model: `node_device_id` - Edge Node backend identity, назначается pairing/provisioning. `client_device_id` - конкретный UI-клиент, в MVP генерируется frontend через `crypto.randomUUID()` и хранится в `localStorage`; backend auto-registers новый client. `device_id` остается domain/storage field для POS Edge node identity в operational payload, а новые transport examples используют явные `node_device_id` и `client_device_id`.

## Проверки

Основная проверка POS Edge:

```powershell
cd pos-backend
go test ./...
```

Проверка Cloud receiver:

```powershell
cd cloud-backend
go test ./...
```

## Где искать

- Целевая спецификация: `SPECv1.3.md`
- Roadmap MVP: `ROADMAP.md`
- Архитектурные правила: `AGENTS.md`
- Запуск POS Edge backend: `pos-backend/README.md`
- Запуск Cloud receiver: `cloud-backend/README.md`
- HTTP маршруты POS Edge: `pos-backend/internal/pos/api/router.go`
- Публичный API/use cases жизненного цикла precheck и payment: `pos-backend/internal/pos/api/router.go`, `pos-backend/internal/pos/app/precheck/service.go`, `pos-backend/internal/pos/app/check/service.go`
- Use cases: `pos-backend/internal/pos/app/`
- Доменные модели: `pos-backend/internal/pos/domain/`
- Репозитории SQLite: `pos-backend/internal/pos/infra/sqlite/`
- Схема БД: `pos-backend/migrations/sqlite/`
- Sync contracts: `docs/sync/edge-cloud-contracts-v1.md`

## Статус

- Architecture Lock: v1.3.
- Целевая финансовая модель: `Order -> Precheck -> Payment -> Check`.
- Production data migration before first launch: не требуется.
- SQLite clean install: активный migration path состоит из одного canonical `001_init.sql`, который сразу создает текущую runtime-схему без `payments.check_id`; служебные version tables создаются кодом startup upgrade framework.
- POS Edge SQLite runtime contract: functional minimum `>= 3.37.0`, production WAL pilot baseline `>= 3.51.3` или pinned backport `3.50.7/3.44.6`; backend завершается при несоответствии.
- POS Edge code: публичный runtime `Order -> Precheck -> Payment -> Check` включен; old check/payment compatibility endpoints удалены.
- `local_event_log` уже является частью edge foundation, хранит `command_id` той же write-операции, что и outbox rows (одна write-операция может породить несколько events), и доступен read-only через `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated`.
- Sync outbox имеет retry-safe поля `sequence_no`, `sync_direction`, `attempts`, `next_retry_at`, `locked_at`, `locked_by`, `sent_at`, `last_error` и статусы `pending`, `processing`, `sent`, `failed`, `suspended`.
- Sync outbox доступен через `GET /api/v1/sync/outbox`, aggregated status через `GET /api/v1/sync/status`, manual retry failed/suspended через `POST /api/v1/sync/retry-failed`; эти operator endpoints требуют заголовки `X-Node-Device-ID`, `X-Client-Device-ID`, `X-Session-ID`, `X-Actor-Employee-ID` и соответствующие permissions (`pos.sync.view`, `pos.sync.retry_failed`).
- Edge financial foundation включает публичные precheck issue/read/list/cancel endpoints, precheck payment endpoint, `manager_override_audit`, `payment_attempts`, automatic final checks, `cash_sessions` как кассовые смены, `cash_drawer_events`, PIN auth/session foundation, halls/tables API и базовые HTTP endpoints для cash shift/drawer workflows.
- Cloud-owned master-data foundation запрещает Edge runtime mutation restaurants/devices metadata/roles/employees/halls/tables/catalog/menu/recipes/inventory reference data. POS Edge использует локальную read model offline; Cloud-authored master data applies through `/api/v1/sync/master-data/snapshots` or `/api/v1/sync/master-data/{stream}`; dev seed/admin write routes require `POS_DEV_TOOLS=1`.
- Auth/device foundation включает pairing status/pair endpoints, `POST /api/v1/auth/logout`, revoked sessions, client device registry, `node_device_id`/`client_device_id` metadata в local events/outbox/SyncEnvelope.
- Pairing verifier хранится в keyed format `pairing.hmac-sha256.v1`; plaintext pairing code не сохраняется.
- PIN login должен однозначно определить одного active employee в paired restaurant; дубли active PIN отклоняются как conflict.
- Закрытие личной смены сотрудника в POS Edge запрещено при открытых заказах или active cash shift.
- Cloud: минимальный `cloud-backend/` Sync Receiver реализован; Cloud не является зависимостью для критических POS Edge операций.
- POS UI: `pos-ui` на Vue 3 + Quasar реализует `pairing -> login -> pos -> lock/logout` и POS Terminal Core для single-terminal cashier flow.
- Источник истины для активных POS операций: локальный POS Edge Node.

## Error handling contract (реализовано сейчас)

реализовано сейчас:

- POS backend возвращает безопасный error envelope `{ "error": { "code", "message_key", "details", "correlation_id" } }`;
- backend пишет internal cause и panic stack только в structured logs, а UI получает stable code и i18n key;
- `X-Error-Code` содержит stable code для audit middleware;
- `401 SESSION_REVOKED` очищает UI session state и ведет к controlled login flow;
- `403 PERMISSION_DENIED` показывает modal "Недостаточно прав" без logout;
- `409`, `429`, `5xx`, network/timeout различаются в UI и не показываются raw stack/SQL/Go errors;
- POS UI использует global error dialog store и `vue-i18n` keys для user-facing ошибок;
- TanStack mutations не имеют опасного auto-retry для финансовых write commands.

Каталог кодов: `docs/backend/POS-ERROR-CATALOG.md`.

## Runtime logging config

реализовано сейчас:

- POS Edge log level env: `POS_LOG_LEVEL`
- Cloud Backend log level env: `CLOUD_LOG_LEVEL`
- Supported values: `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`
- Default: `INFO`

PowerShell example:

```powershell
$env:POS_LOG_LEVEL="DEBUG"
$env:CLOUD_LOG_LEVEL="INFO"
```

### Worker telemetry

реализовано сейчас:

- POS sync sender writes structured non-HTTP telemetry events with normalized fields (`operation`, `action`, `result`, `error_code`).
- TRACE can be enabled with `POS_LOG_LEVEL=TRACE` for lifecycle-level diagnostics of the sender worker.

## Permission model (реализовано сейчас)

реализовано сейчас:

- backend enforces canonical RBAC permission ids in app-layer for critical cashier runtime operations;
- role permissions are still stored as JSON on roles, but authorization checks use stable ids;
- canonical role profiles are implemented for `cashier`, `senior_cashier`, `waiter`, `manager`, `kitchen`, `support_admin`;
- read/runtime APIs (`employee-shifts/current|recent`, `cash-shifts/current`, `halls`, `tables`, `catalog`, `menu`, `orders/current|{id}`, `prechecks`, `checks`) require explicit operator permissions;
- order close uses `pos.order.close`;
- cash drawer event recording requires backend permission `pos.cash_drawer.record_event`;
- payment capture is split by method: `pos.payment.cash`, `pos.payment.card.manual`, `pos.payment.other`;
- precheck cancel override uses split permissions: actor requires `pos.precheck.cancel.request`, approver requires `pos.precheck.cancel`;
- operator-triggered `GET /api/v1/sync/outbox`, `GET /api/v1/sync/status`, `GET /api/v1/sync/local-events` require `pos.sync.view`;
- operator-triggered `POST /api/v1/sync/retry-failed` requires `pos.sync.retry_failed`;
- POS UI visibility is wired to the same backend permission ids; backend remains the security boundary;
- failed authorization returns `forbidden` without leaking PIN or PIN hash data.

вне текущего объема:

- operations that do not exist in the current runtime surface are not part of RBAC hardening until they have route/use-case/contracts and tests.

## Currency precision (реализовано сейчас)

реализовано сейчас:

- POS backend validates currency codes using canonical active ISO 4217 catalog and rejects unsupported codes;
- catalog explicitly covers full active ISO list (including SEA currencies such as `IDR`, `THB`, `VND`, `MYR`, `SGD`, `PHP`);
- runtime precision is currency-code driven and supports 0/2/3/4 minor units where defined by ISO catalog;
- POS UI uses precision-aware `minor <-> major` conversion helper by currency code (instead of fixed `/100`);
- Cloud provisioning supports `currencies` stream and Cloud startup upserts canonical active ISO currency reference into `cloud_currency_reference`.

## Runtime DB versioning and backup

реализовано сейчас:

- shared product/runtime version env: `MH_POS_VERSION` (default `0.1.1`) for both POS and Cloud modules;
- POS startup uses `db_runtime_versions` + `schema_migrations` and creates SQLite backup before schema/data upgrade (`POS_SQLITE_BACKUP_DIR`);
- Cloud startup uses `db_runtime_versions` + `schema_migrations` and creates PostgreSQL JSONL backup snapshot before schema/data upgrade (`CLOUD_POSTGRES_BACKUP_DIR`);
- missing `db_runtime_versions` is treated as the oldest DB state, not as an immediate crash;
- active migration path uses one managed canonical SQL file per module; the file is re-runnable on version upgrade;
- `schema_migrations` tracks the active SQL file with SHA-256 checksum; checksum drift at the same DB/app version fails fast;
- required tables/columns/indexes are verified before HTTP server/workers start;
- `DB version > MH_POS_VERSION` fails fast because downgrade is not supported;
- planned next: UI/admin SQLite cleanup/reset operation with mandatory backup and explicit confirmation for collision/corruption recovery.

## SQLite maintenance (реализовано сейчас)

реализовано сейчас:

- обычный startup не выполняет `VACUUM`;
- `VACUUM`, `VACUUM INTO`, `PRAGMA optimize`, `PRAGMA wal_checkpoint(TRUNCATE)` доступны как explicit maintenance через `scripts/maintain-sqlite.ps1`;
- `VACUUM`/`VACUUM INTO` требуют `-Force`;
- `VACUUM INTO` не перезаписывает существующий target file;
- операции выполняются вне active write transaction.

Пример:

```powershell
.\scripts\maintain-sqlite.ps1 -DatabasePath "pos-backend\data\pos-edge.db" -Optimize -WalCheckpoint
.\scripts\maintain-sqlite.ps1 -DatabasePath "pos-backend\data\pos-edge.db" -VacuumInto "pos-backend\data\snapshots\pos-edge.compact.db" -Force
```
