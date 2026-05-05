# MyHoReCa POS / RMS

Монорепозиторий edge-first POS/RMS платформы для HoReCa.

Текущий фокус репозитория - POS Edge Backend: локальный backend кассового узла на Go + SQLite, который должен работать без интернета, сохранять критические операции локально и готовить данные к будущей синхронизации с cloud.

## Структура Монорепозитория

```text
.
|-- AGENTS.md                 # архитектурные правила и быстрый вход для AI-агентов
|-- README.md                 # карта монорепозитория
|-- pos-backend/              # POS Edge Backend, текущая основная кодовая база
|   |-- README.md             # запуск, Docker, smoke test и API-примеры backend
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
|-- .codex/skills/            # локальные skills для Codex
|-- pack_go_files.py          # вспомогательный скрипт упаковки Go-файлов
`-- project_dump.py           # вспомогательный скрипт дампа проекта
```

Планируемые, но еще не реализованные части монорепозитория:

- `pos-ui/` - локальный UI кассового узла.
- `device-adapters/` - адаптеры принтеров, терминалов и другого оборудования.
- `cloud-backend/` - будущий cloud backend на Go + PostgreSQL.
- `backoffice-ui/` - будущий web UI для управления и отчетности.

## Как Работать С Репозиторием

Перед изменениями прочитай [AGENTS.md](AGENTS.md). Он фиксирует обязательные принципы: edge-first, offline-first, Clean Architecture, транзакции для write операций и outbox в той же транзакции.

Для backend-разработки переходи в `pos-backend`:

```powershell
cd pos-backend
go mod tidy
go test ./...
go run ./cmd/pos-edge
```

Сервис по умолчанию слушает `http://localhost:8080`.

Полные команды запуска, Docker и API smoke test описаны в [pos-backend/README.md](pos-backend/README.md).

## Основные Контуры

### POS Edge Backend

Где лежит: `pos-backend/`

Назначение:

- локальное хранение POS данных в SQLite;
- JSON API для POS UI;
- доменные инварианты заказов, чеков, оплат, смен и cash sessions;
- edge foundation для `local_event_log`, `SyncEnvelope` и sync outbox;
- read-only operational access к sync outbox и local events;
- financial foundation для `payment_attempts`, cash sessions и cash drawer events;
- foundation для будущих рецептов, склада и учета.

Архитектура внутри backend:

```text
domain -> app -> ports -> infra
```

Короткое правило: domain не знает про HTTP, SQLite, `database/sql` и инфраструктуру; use cases управляют транзакциями; handlers остаются тонкими.

### Cloud И UI

Cloud backend, back office UI и POS UI пока не реализованы. Не добавляй cloud как зависимость для критических POS операций: локальный кассовый узел должен продолжать работать offline.

## Проверки

Основная проверка сейчас:

```powershell
cd pos-backend
go test ./...
```

Тесты включают архитектурные ограничения import-ов и доменные/SQLite инварианты.

## Где Искать

- Запуск backend: `pos-backend/README.md`
- Архитектурные правила: `AGENTS.md`
- HTTP маршруты: `pos-backend/internal/pos/api/router.go`
- Use cases: `pos-backend/internal/pos/app/`
- Доменные модели: `pos-backend/internal/pos/domain/`
- Репозитории SQLite: `pos-backend/internal/pos/infra/sqlite/`
- Схема БД: `pos-backend/migrations/sqlite/`
- Отчет по фазе backend: `pos-backend/docs/phase-1-report.md`

## Статус

- Phase 1: POS Edge Backend foundation.
- `local_event_log` уже является частью edge foundation, хранит `command_id` той же write-операции, что и outbox, и доступен read-only через `GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated`.
- Sync outbox доступен через `GET /api/v1/sync/outbox`.
- Edge financial foundation включает `payment_attempts`, `cash_sessions`, `cash_drawer_events` и базовые HTTP endpoints для cash session/drawer workflows.
- Cloud: минимальный `cloud-backend/` Sync Receiver реализован; Cloud не является зависимостью для критических POS Edge операций.
- POS UI: не реализован.
- Source of truth для активных POS операций: локальный POS Edge Node.

## Phase 0/2 Sync Receiver Update

Репозиторий теперь включает минимальный `cloud-backend/`:

- Go Cloud API entrypoint: `cloud-backend/cmd/cloud-api`
- PostgreSQL bootstrap и migrations: `cloud-backend/migrations/postgres`
- health endpoint: `GET /health`
- Edge event receiver: `POST /api/v1/sync/edge-events`
- idempotent receive/dedupe для текущих POS Edge events, включая cash session/drawer foundation events
- raw `SyncEnvelope` storage до будущих Cloud projections

Контракты синхронизации и idempotency rules зафиксированы в `docs/sync/edge-cloud-contracts-v1.md`.

Cloud backend команды:

```powershell
cd cloud-backend
go test ./...
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```
