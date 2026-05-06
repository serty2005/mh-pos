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
- `local_event_log`;
- `pos_sync_outbox`;
- `SyncEnvelope` foundation;
- shifts, cash sessions, cash drawer events;
- public precheck issue/read/list/cancel flow: `POST /api/v1/orders/{id}/precheck`, `GET /api/v1/prechecks/{id}`, `GET /api/v1/orders/{id}/prechecks`, `POST /api/v1/prechecks/{id}/cancel`;
- manager override для `CancelPrecheck`: локальная PBKDF2 PIN verification, permission `precheck.cancel`, audit trail `manager_override_audit`;
- precheck-based payment capture: `POST /api/v1/prechecks/{id}/payments`, partial payments, automatic final `Check` после полной оплаты и automatic order close;
- final checks/payments foundation;
- `payment_attempts`;
- read-only sync endpoints.

Честное состояние текущего кода: POS Edge backend уже выполняет runtime flow `Order -> Precheck -> Payment -> Check`. `IssuePrecheck` переводит order в `locked`; публичный `CancelPrecheck` требует manager employee id, PIN и reason, пишет audit trail и возвращает unpaid active issued precheck в `open`; payment capture идет через `precheck_id`, а final `Check` создается только после полной оплаты. Deprecated `POST /api/v1/orders/{id}/check` остается dev alias для `IssuePrecheck`; legacy `POST /api/v1/checks/{id}/payments` отключен и не обходит precheck flow.

## Структура Монорепозитория

```text
.
|-- AGENTS.md                 # архитектурные правила и быстрый вход для AI-агентов
|-- README.md                 # карта монорепозитория
|-- SPECv1.3.md               # целевая спецификация Architecture Lock v1.3
|-- ROADMAP_MVP.md            # roadmap перехода к MVP v1.3
|-- pos-backend/              # POS Edge Backend, текущая основная кодовая база
|   |-- README.md             # запуск, Docker, smoke test и текущее legacy API состояние
|   |-- cmd/pos-edge/         # entrypoint локального POS backend сервиса
|   |-- internal/platform/    # общая инфраструктура: clock, http, idgen, sqlite, tx
|   |-- internal/pos/         # POS bounded context
|   |   |-- api/              # HTTP router и thin handlers
|   |   |-- app/              # use cases, транзакции, orchestration
|   |   |-- domain/           # доменные модели, ошибки и инварианты
|   |   |-- ports/            # интерфейсы репозиториев
|   |   `-- infra/sqlite/     # SQLite реализации репозиториев
|   |-- migrations/sqlite/    # SQLite schema migrations
|   |-- docker/               # Dockerfile
|   |-- docker-compose.yml    # локальный запуск через Docker Compose
|   `-- docs/                 # отчеты и проектные документы по backend
|-- cloud-backend/            # минимальный Cloud Sync Receiver foundation
|   |-- README.md             # запуск и тесты cloud receiver
|   |-- cmd/cloud-api/        # entrypoint Cloud API
|   `-- migrations/postgres/  # PostgreSQL bootstrap и migrations
|-- docs/sync/                # sync contracts
|-- .codex/skills/            # локальные skills для Codex
|-- pack_go_files.py          # вспомогательный скрипт упаковки Go-файлов
`-- project_dump.py           # вспомогательный скрипт дампа проекта
```

Планируемые, но еще не реализованные части монорепозитория:

- `pos-ui/` - локальный UI кассового узла.
- `device-adapters/` - адаптеры принтеров, терминалов и другого оборудования.
- `backoffice-ui/` - будущий web UI для управления и отчетности.

## Как Работать С Репозиторием

Перед изменениями прочитай:

- [AGENTS.md](AGENTS.md)
- [SPECv1.3.md](SPECv1.3.md)
- [ROADMAP_MVP.md](ROADMAP_MVP.md)

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

## Основные Контуры

### POS Edge Backend

Где лежит: `pos-backend/`

Назначение:

- локальное хранение POS данных в SQLite;
- JSON API для POS UI;
- доменные инварианты заказов, смен, cash sessions и текущего financial foundation;
- edge foundation для `local_event_log`, `SyncEnvelope` и sync outbox;
- read-only operational access к sync outbox и local events;
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

POS UI и back office UI пока не реализованы.

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
- Roadmap MVP: `ROADMAP_MVP.md`
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
- POS Edge SQLite runtime contract: functional minimum `>= 3.37.0`, production WAL pilot baseline `>= 3.51.3` или pinned backport `3.50.7/3.44.6`; backend завершается при несоответствии.
- POS Edge code: public `Order -> Precheck -> Payment -> Check` runtime enabled; legacy check payment endpoint is disabled.
- `local_event_log` уже является частью edge foundation, хранит `command_id` той же write-операции, что и outbox rows (одна write-операция может породить несколько events), и доступен read-only через `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated`.
- Sync outbox доступен через `GET /api/v1/sync/outbox`.
- Edge financial foundation включает публичные precheck issue/read/list/cancel endpoints, precheck payment endpoint, `manager_override_audit`, `payment_attempts`, automatic final checks, `cash_sessions`, `cash_drawer_events` и базовые HTTP endpoints для cash session/drawer workflows.
- Закрытие смены в POS Edge запрещено при открытых заказах или active cash session.
- Cloud: минимальный `cloud-backend/` Sync Receiver реализован; Cloud не является зависимостью для критических POS Edge операций.
- POS UI: не реализован.
- Source of truth для активных POS операций: локальный POS Edge Node.
