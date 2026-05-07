# MyHoReCa POS / RMS

Монорепозиторий edge-first POS/RMS платформы для HoReCa.

Текущий фокус репозитория - перевод уже существующего POS Edge foundation к Architecture Lock v1.3. Целевая финансовая модель v1.3:

```text
Order -> Precheck -> Payment -> Check
```

`Precheck` - рабочий финансовый snapshot для гостя. `Check` - только финальный неизменяемый расчетный документ после полной оплаты precheck.

Важно: проект еще не был запущен в production. Реальных production БД с клиентскими данными нет, поэтому production data migration до первого запуска не требуется. Изменения v1.3 проектируются как first-launch schema/logic.

## Текущее Состояние

Репозиторий уже содержит рабочий foundation:

- `pos-backend/` - локальный POS Edge backend на Go + SQLite;
- SQLite runtime gate для POS Edge: startup fail-fast проверяет фактические `sqlite_version()`, `journal_mode=WAL`, `synchronous=NORMAL`, `foreign_keys=ON`, `busy_timeout >= 5000`;
- `cloud-backend/` - минимальный Cloud Sync Receiver на Go + PostgreSQL;
- approved frontend MVP: отдельный пакет `pos-ui` на Vue 3 + TypeScript + Quasar + Vue Router + Pinia + `@tanstack/vue-query` + `vue-i18n` + Zod; `/pair`, `/login`, `/lock` и рабочий POS Terminal Core `/pos` для single-terminal cashier flow реализованы;
- `local_event_log`;
- `pos_sync_outbox`;
- `SyncEnvelope` foundation;
- PIN auth/session foundation: `POST /api/v1/auth/pin-login`, `GET /api/v1/auth/session`, `POST /api/v1/auth/logout`;
- strict lock/logout model: UI lock или auto-lock вызывает backend logout, session становится `revoked`, новый PIN создает новую session;
- operator auth enforcement для business/operator flows: active employee session, `actor_employee_id`, `session_id`, matching `client_device_id` и permissions там, где нужны;
- system/device flows (`sync`, pairing/status, diagnostics/hardware callbacks в будущих фазах) не требуют employee session и должны авторизоваться отдельным device/system path;
- Edge Node pairing foundation: `POST /api/v1/system/pair`, `GET /api/v1/system/pairing-status`;
- identity split: `node_device_id` обозначает Edge Backend и назначается pairing flow; `client_device_id` обозначает frontend-клиент, генерируется `pos-ui` в `localStorage` и auto-registers на Edge;
- actor/session/client/node metadata в write commands, `local_event_log`, `pos_sync_outbox` и `SyncEnvelope`: `node_device_id`, `client_device_id`, `actor_employee_id`, `session_id`;
- halls/tables foundation для выбора стола в POS/Waiter UI;
- read-only endpoint активного заказа по столу для cashier terminal: `GET /api/v1/orders/current?table_id=...`;
- order line editing foundation: изменение количества и void позиции без физического удаления;
- shifts, cash sessions, cash drawer events;
- public precheck issue/read/list/cancel flow: `POST /api/v1/orders/{id}/precheck`, `GET /api/v1/prechecks/{id}`, `GET /api/v1/orders/{id}/prechecks`, `POST /api/v1/prechecks/{id}/cancel`;
- manager override для `CancelPrecheck`: локальная PBKDF2 PIN verification, permission `precheck.cancel`, audit trail `manager_override_audit`;
- precheck-based payment capture: `POST /api/v1/prechecks/{id}/payments`, partial payments, automatic final `Check` после полной оплаты и automatic order close;
- final checks/payments foundation;
- `payment_attempts`;
- retry-safe sync outbox foundation with status/claim/retry metadata;
- operational sync endpoints for outbox inspection, local events, aggregated status and manual retry of failed/suspended messages.

Честное состояние текущего кода: POS Edge backend уже выполняет runtime flow `Order -> Precheck -> Payment -> Check`. `IssuePrecheck` переводит order в `locked`; публичный `CancelPrecheck` требует manager employee id, PIN и reason, пишет audit trail и возвращает unpaid active issued precheck в `open`; payment capture идет через `precheck_id`, а final `Check` создается только после полной оплаты. Deprecated `POST /api/v1/orders/{id}/check` остается dev alias для `IssuePrecheck`; legacy `POST /api/v1/checks/{id}/payments` отключен и не обходит precheck flow. KDS/Waiter UI еще не готовы: backend сейчас дает только halls/tables и базовое редактирование order lines.

## Структура Монорепозитория

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
|   |-- migrations/sqlite/    # canonical SQLite first-launch init schema
|   |-- docker/               # Dockerfile
|   |-- docker-compose.yml    # локальный запуск через Docker Compose
|   `-- docs/                 # отчеты и проектные документы по backend
|-- cloud-backend/            # минимальный Cloud Sync Receiver foundation
|   |-- README.md             # запуск и тесты cloud receiver
|   |-- cmd/cloud-api/        # entrypoint Cloud API
|   `-- migrations/postgres/  # PostgreSQL bootstrap и migrations
|-- pos-ui/                   # Vue 3 + Quasar POS Terminal Core
|-- docs/sync/                # sync contracts
|-- .codex/skills/            # локальные skills для Codex
|-- pack_go_files.py          # вспомогательный скрипт упаковки Go-файлов
`-- project_dump.py           # вспомогательный скрипт дампа проекта
```

Планируемые, но еще не реализованные части монорепозитория:

- `device-adapters/` - адаптеры принтеров, терминалов и другого оборудования.
- `backoffice-ui/` - будущий web UI для управления и отчетности.

## Как Работать С Репозиторием

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

## Local E2E Prototype Quickstart

implemented now: локально можно поднять минимальную связку `pos-ui -> pos-backend -> cloud-backend` и пройти cashier flow вручную.

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
pairing -> login -> open shift -> open cash session -> select hall/table -> create order -> add lines -> change quantity -> void line -> issue precheck -> cancel unpaid precheck with manager override -> issue precheck again -> capture payment -> final check -> close cash session -> close shift -> lock/logout
```

7. Проверь POS sync state:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=20
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=20
Invoke-RestMethod http://localhost:8080/api/v1/sync/status
Invoke-RestMethod -Method Post http://localhost:8080/api/v1/sync/retry-failed
```

8. Проверь Cloud receiver:

```powershell
Invoke-RestMethod http://localhost:8090/health
.\scripts\send-cloud-test-envelope.ps1 -ReplayTwice
```

`.\scripts\dev-smoke.ps1` выполняет health checks, POS demo bootstrap, POS sync endpoint checks и Cloud envelope replay, но не стартует серверы за тебя.

## Local E2E Prototype: получить pairing code и войти в POS UI

implemented now: local developer flow uses real POS backend endpoints and a real MVP pairing code.

1. Start Cloud:

```powershell
cd cloud-backend
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

2. Start POS with dev tools:

```powershell
cd pos-backend
$env:POS_DEV_TOOLS="1"
go run ./cmd/pos-edge
```

3. From repo root, get demo credentials:

```powershell
.\scripts\bootstrap-pos-demo.ps1
```

Use the returned `pairing_code` on `http://localhost:5173/pair`, then log in on `/login` with cashier PIN `1111`. The script also returns `restaurant_id`, `node_device_id`, employee ids, `hall_id`, `table_ids`, and `menu_item_ids`.

4. Check Cloud replay with real bootstrap IDs:

```powershell
$demo = .\scripts\bootstrap-pos-demo.ps1
.\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
```

5. Check local POS sync state:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/sync/status
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=10
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=10
```

out of scope: production POS-to-Cloud sender worker is not implemented in this prototype slice.

## Основные Контуры

### POS Edge Backend

Где лежит: `pos-backend/`

Назначение:

- локальное хранение POS данных в SQLite;
- JSON API для POS UI;
- доменные инварианты заказов, смен, cash sessions и текущего financial foundation;
- edge foundation для `local_event_log`, `SyncEnvelope` и sync outbox;
- operational access к sync outbox, local events, aggregated sync status и manual retry failed/suspended;
- financial foundation для precheck payments, final checks, `payment_attempts`, cash sessions и cash drawer events;
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
- `/pos` - cashier surface для одного терминала: смена, кассовая сессия, выбор зала/стола, активный заказ, позиции меню, изменение/void позиций, выпуск/отмена пречека, cash/trusted card payment и отображение final check.

Identity model: `node_device_id` - Edge Node backend identity, назначается pairing/provisioning. `client_device_id` - конкретный UI-клиент, в MVP генерируется frontend через `crypto.randomUUID()` и хранится в `localStorage`; backend auto-registers новый client. Старый `device_id` в текущих API/таблицах остается backward-compatible alias для `node_device_id`, но новая документация и новый код должны использовать явное разделение.

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

## Где Искать

- Целевая спецификация: `SPECv1.3.md`
- Roadmap MVP: `ROADMAP.md`
- Архитектурные правила: `AGENTS.md`
- Запуск POS Edge backend: `pos-backend/README.md`
- Запуск Cloud receiver: `cloud-backend/README.md`
- HTTP маршруты POS Edge: `pos-backend/internal/pos/api/router.go`
- Public precheck lifecycle/payment API/use cases: `pos-backend/internal/pos/api/router.go`, `pos-backend/internal/pos/app/precheck/service.go`, `pos-backend/internal/pos/app/check/service.go`
- Use cases: `pos-backend/internal/pos/app/`
- Доменные модели: `pos-backend/internal/pos/domain/`
- Репозитории SQLite: `pos-backend/internal/pos/infra/sqlite/`
- Схема БД: `pos-backend/migrations/sqlite/`
- Sync contracts: `docs/sync/edge-cloud-contracts-v1.md`

## Статус

- Architecture Lock: v1.3.
- Target financial model: `Order -> Precheck -> Payment -> Check`.
- Production data migration before first launch: не требуется.
- SQLite clean install: активный migration path состоит из canonical `001_init.sql`, который сразу создает текущую runtime-схему без legacy `payments.check_id`.
- POS Edge SQLite runtime contract: functional minimum `>= 3.37.0`, production WAL pilot baseline `>= 3.51.3` или pinned backport `3.50.7/3.44.6`; backend завершается при несоответствии.
- POS Edge code: public `Order -> Precheck -> Payment -> Check` runtime enabled; legacy check payment endpoint is disabled.
- `local_event_log` уже является частью edge foundation, хранит `command_id` той же write-операции, что и outbox rows (одна write-операция может породить несколько events), и доступен read-only через `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated`.
- Sync outbox имеет retry-safe поля `sequence_no`, `attempts`, `next_retry_at`, `locked_at`, `locked_by`, `sent_at`, `last_error` и статусы `pending`, `processing`, `sent`, `failed`, `suspended`.
- Sync outbox доступен через `GET /api/v1/sync/outbox`, aggregated status через `GET /api/v1/sync/status`, manual retry failed/suspended через `POST /api/v1/sync/retry-failed`.
- Edge financial foundation включает публичные precheck issue/read/list/cancel endpoints, precheck payment endpoint, `manager_override_audit`, `payment_attempts`, automatic final checks, `cash_sessions`, `cash_drawer_events`, PIN auth/session foundation, halls/tables API и базовые HTTP endpoints для cash session/drawer workflows.
- Auth/device foundation включает pairing status/pair endpoints, `POST /api/v1/auth/logout`, revoked sessions, client device registry, `node_device_id`/`client_device_id` metadata в local events/outbox/SyncEnvelope.
- Закрытие смены в POS Edge запрещено при открытых заказах или active cash session.
- Cloud: минимальный `cloud-backend/` Sync Receiver реализован; Cloud не является зависимостью для критических POS Edge операций.
- POS UI: `pos-ui` на Vue 3 + Quasar реализует `pairing -> login -> pos -> lock/logout` и POS Terminal Core для single-terminal cashier flow.
- Source of truth для активных POS операций: локальный POS Edge Node.
