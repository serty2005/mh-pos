# AGENTS.md

## Быстрый Старт Для Агентов

**⚠️ ВАЖНЫЕ ПРАВИЛА ОКРУЖЕНИЯ (WINDOWS + POWERSHELL):**
- Для поиска, чтения и анализа файлов **используйте Python** (а не системные bash/powershell утилиты).
- При открытии любых файлов через Python **всегда принудительно указывайте кодировку UTF-8** (`open(filepath, mode, encoding='utf-8')`). Окружение работает на русскоязычной Windows, и чтение без `utf-8` приведет к нечитаемым символам (mojibake) и поломке кириллицы.
- Все комментарии в генерируемом коде должны быть строго на **русском языке**.


Перед любыми изменениями держи в голове главные инварианты: POS должен работать offline, все write use case выполняются в транзакции, `local_event_log` и outbox пишутся в той же транзакции, закрытые заказы не меняются, смена на device может быть только одна активная.

Architecture Lock v1.3 фиксирует целевую финансовую модель:

```text
Order -> Precheck -> Payment -> Check
```

`Precheck` - рабочий финансовый snapshot для гостя. `Check` - только финальный неизменяемый расчетный документ после полной оплаты precheck. Новая логика должна двигаться к `IssuePrecheck`, а не к старому `CreateCheck`.

Важно: проект еще не был запущен в production. Реальных production БД с клиентскими данными нет, поэтому production data migration до первого запуска не требуется. Изменения v1.3 нужно проектировать как first-launch schema/logic, а не как миграцию исторических данных.

Текущее состояние кода нужно отделять от следующих будущих улучшений: `pos-backend` уже включает публичный runtime `Order -> Precheck -> Payment -> Check`. `IssuePrecheck` (`POST /api/v1/orders/{id}/precheck`, `GET /api/v1/prechecks/{id}`, `GET /api/v1/orders/{id}/prechecks`) создает issued precheck и переводит order в `locked`; публичный `CancelPrecheck` (`POST /api/v1/prechecks/{id}/cancel`) требует manager employee id, PIN и reason, проверяет локальный PBKDF2 `pin_hash`, пишет `manager_override_audit`, `local_event_log` и outbox в одной транзакции и возвращает unpaid active issued precheck order в `open`; payment capture идет через `POST /api/v1/prechecks/{id}/payments`, поддерживает partial payments и автоматически создает final `Check` + закрывает order после полной оплаты. Deprecated `POST /api/v1/orders/{id}/check` остается alias к `IssuePrecheck`; legacy `POST /api/v1/checks/{id}/payments` отключен. Sync/outbox foundation уже включает retry-safe schema/app/API состояние очереди, но полноценный Cloud sender/worker еще не реализован.

### Карта Репозитория

```text
.
|-- README.md                 # карта монорепозитория и команды входа
|-- AGENTS.md                 # этот документ: правила архитектуры и навигация
|-- SPECv1.3.md               # целевая архитектурная спецификация MVP-0 / first launch
|-- ROADMAP_MVP.md            # рабочий roadmap перехода к v1.3
|-- pos-backend/              # текущая основная кодовая база POS Edge
|   |-- README.md             # запуск, Docker, smoke test, текущий API и first-launch schema
|   |-- cmd/pos-edge/         # main() локального POS Edge Backend
|   |-- internal/platform/    # clock, http helpers, idgen, sqlite, tx
|   |-- internal/pos/api/     # HTTP router и thin handlers
|   |-- internal/pos/app/     # use cases, транзакции, outbox orchestration
|   |-- internal/pos/domain/  # бизнес-модели, ошибки, инварианты
|   |-- internal/pos/ports/   # интерфейсы репозиториев
|   |-- internal/pos/infra/   # реализации портов, сейчас SQLite
|   |-- migrations/sqlite/    # canonical first-launch SQLite init schema
|   `-- docs/                 # проектные отчеты backend
|-- cloud-backend/            # минимальный Cloud Sync Receiver foundation
|   |-- README.md             # запуск и тесты cloud receiver
|   |-- cmd/cloud-api/        # main() Cloud API
|   `-- migrations/postgres/  # PostgreSQL bootstrap и migrations
|-- docs/sync/                # sync contracts
|-- .codex/skills/            # локальные Codex skills
|-- pack_go_files.py          # вспомогательный скрипт упаковки Go-файлов
`-- project_dump.py           # вспомогательный скрипт дампа проекта
```

### Куда Идти За Чем

- Целевая архитектура v1.3: `SPECv1.3.md`
- Порядок MVP-итераций: `ROADMAP_MVP.md`
- Запустить POS Edge backend: `pos-backend/README.md`
- Запустить Cloud Sync Receiver: `cloud-backend/README.md`
- Найти текущие HTTP endpoints: `pos-backend/internal/pos/api/router.go`
- Добавить или изменить use case: `pos-backend/internal/pos/app/<context>/service.go`
- Проверить бизнес-правило: `pos-backend/internal/pos/domain/<context>/`
- Добавить repository contract: `pos-backend/internal/pos/ports/`
- Реализовать SQLite storage: `pos-backend/internal/pos/infra/sqlite/`
- Изменить схему БД: `pos-backend/migrations/sqlite/`
- Проверить sync endpoints: `GET /api/v1/sync/outbox`, `GET /api/v1/sync/local-events`, `GET /api/v1/sync/status`, `POST /api/v1/sync/retry-failed`
- Проверить архитектурные import rules: `pos-backend/internal/pos/architecture_test.go`
- Проверить schema/invariant tests: `pos-backend/internal/pos/infra/sqlite/schema_test.go`

### Команды

```powershell
cd pos-backend
go test ./...
go run ./cmd/pos-edge
docker compose up --build

cd ../cloud-backend
go test ./...
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

### Правила Навигации По Слоям

- `domain`: только бизнес-логика, типы, ошибки, инварианты. Никакого HTTP, SQL, `database/sql`, infra.
- `app`: orchestration use cases, транзакции, вызовы портов, запись outbox. Никаких прямых SQL.
- `ports`: интерфейсы репозиториев.
- `infra`: реализации портов и работа с SQLite.
- `api`: HTTP mapping, request validation, response mapping. Без бизнес-логики.
- `platform`: технические primitives, не POS business logic.

### Когда Добавляешь Write Use Case

1. Проверь доменный инвариант.
2. Открой транзакцию.
3. Выполни repository writes.
4. Запиши local event в `local_event_log` и command/event в `pos_sync_outbox` в той же транзакции.
5. Закоммить транзакцию.
6. Добавь тест на invalid state transition или boundary case.

Запрещено писать в outbox вне транзакции или делать split транзакции.

---

## Purpose

Этот документ определяет архитектурные принципы, правила разработки и доменные инварианты системы POS/RMS.

Он обязателен для:

- разработчиков
- AI-агентов
- code review
- архитектурных решений

Любые изменения архитектуры должны быть согласованы с `SPECv1.3.md`, `ROADMAP_MVP.md` и этим документом.

---

# System Overview

Система построена как **edge-first POS/RMS платформа**.

## 1. POS Edge Node

Локальный кассовый узел работает на Windows/Linux/Android и содержит:

- POS UI
- POS Edge Backend (Go + SQLite)
- device adapters
- sync outbox

Главная цель: работать без интернета.

## 2. Cloud Platform

В репозитории уже есть минимальный `cloud-backend/` Sync Receiver foundation. Это не runtime dependency для критических POS операций.

Cloud в целевой архитектуре отвечает за:

- учет и аналитику
- справочники
- рецепты и склад
- sync receiver
- reporting
- integrations
- fiscalization в будущих фазах

---

# Architectural Principles

## 1. Edge-first

- POS должен работать без сети.
- Интернет - это улучшение, не зависимость.
- Все критические операции выполняются локально.

## 2. Offline-first

- Все действия пишутся локально.
- Синхронизация асинхронная.
- Повторная отправка безопасна.

## 3. Eventual Consistency

- Cloud и Edge могут временно расходиться.
- Консистентность достигается через sync.

## 4. Source of Truth

### POS Edge Node - source of truth для:

- активных заказов
- prechecks до синхронизации
- локальных оплат
- финальных checks до синхронизации
- смен и cash sessions

### Cloud Backend - source of truth для:

- меню
- сотрудников
- цен
- рецептов
- склада
- отчетности
- долгосрочных проекций после sync

## 5. Modular Monolith

- Нет микросервисов на MVP.
- Модули изолированы.
- Явные границы контекстов.

## 6. Clean Architecture

Зависимости:

```text
domain -> app -> ports -> infra
```

Запрещено:

- domain -> infra
- domain -> database/sql
- domain -> http

---

# POS Edge Backend Scope

Текущая реализация включает legacy foundation:

- restaurants
- devices
- employees
- roles
- catalog
- menu
- orders
- публичный `Order -> Precheck` API slice и prechecks lifecycle foundation
- precheck-based payments и final checks после полной оплаты
- payment_attempts
- shifts
- cash_sessions
- cash_drawer_events
- `local_event_log`
- sync outbox
- foundation для recipes/inventory/accounting в схеме и repository layer

Текущая реализация еще НЕ включает:

- POS UI
- production-grade inventory workflows
- fiscalization
- public API для inventory/recipes

---

# Target Financial Model v1.3

```text
Order -> Precheck -> Payment -> Check
```

## Order

`Order` - рабочая сущность официанта и кухни.

Правила:

- заказ создается локально и требует активную смену;
- заказ принадлежит `restaurant_id`, `device_id`, `shift_id`;
- в `open` можно добавлять позиции;
- после active precheck заказ должен быть locked;
- закрытый заказ нельзя редактировать.

## Precheck

`Precheck` - заблокированный финансовый snapshot для гостя.

Правила:

- создается из текущего состояния order;
- фиксирует позиции, скидки, налоги и totals;
- активным может быть только один `issued` precheck на order;
- precheck нельзя редактировать;
- изменение order после precheck требует отмены текущего precheck через manager override;
- precheck пишется в `local_event_log` и outbox в той же транзакции.

## Payment

`Payment` - immutable финансовый факт.

Правила:

- payment привязан к precheck;
- payment нельзя удалять или редактировать;
- ошибка исправляется refund/reversal/correction событием;
- нельзя переплатить precheck без явной политики tips/overpayment;
- capture payment пишет local event и outbox в той же транзакции.
- full payment может породить несколько событий с одним `command_id`: `PaymentCaptured`, `CheckCreated`, `OrderClosed`.

## Check

`Check` - финальный неизменяемый расчетный документ.

Правила:

- check создается только после полной оплаты precheck;
- check нельзя использовать как рабочий счет гостя;
- check нельзя создать вручную до оплаты;
- после final check заказ закрывается;
- check immutable.

---

# Смены И Cash Sessions

- У device только одна активная смена.
- Заказ требует активную смену.
- Cash session нельзя открыть без активной смены.
- Cash drawer event нельзя записать без active cash session.
- Нельзя закрыть смену с open/locked заказами.
- Нельзя закрыть смену с active cash session.

---

# Sync Model

Все write-действия пишутся в local edge foundation:

```text
local_event_log
pos_sync_outbox
```

`local_event_log` хранит локальные события для операционного аудита и будущей синхронизации. `pos_sync_outbox` хранит команды/события для retry-safe отправки. Одна write-операция использует один `command_id`: он хранится в `local_event_log`, в `pos_sync_outbox` и внутри `SyncEnvelope` payload.

Operational sync endpoints:

```text
GET /api/v1/sync/outbox?limit=50
GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated
GET /api/v1/sync/status
POST /api/v1/sync/retry-failed
```

`retry-failed` не отправляет данные в Cloud и не меняет бизнес-сущности. Он только возвращает outbox rows со статусом `failed`/`suspended` в `pending` для ручного повторного sync.

Каждый command:

- `command_id`
- `event_id`
- `envelope_version`
- `sequence_no`
- `device_id`
- `aggregate_type`
- `aggregate_id`
- `payload`
- `status`: `pending`, `processing`, `sent`, `failed`, `suspended`
- retry/lock metadata: `attempts`, `next_retry_at`, `locked_at`, `locked_by`, `sent_at`, `last_error`

Принципы:

- idempotent
- retry-safe
- append-only
- `processing` locks должны быть reclaimable, а `failed`/`suspended` можно вручную вернуть в `pending`

---

# Invariants (Critical)

НАРУШЕНИЕ = BUG

- нельзя открыть 2 смены на device
- нельзя создать заказ без смены
- нельзя закрыть смену с открытыми заказами
- нельзя закрыть смену с active cash session
- нельзя изменить закрытый заказ
- нельзя изменить locked order без отмены active precheck по правилам manager override
- нельзя редактировать issued precheck
- нельзя принять payment без precheck в целевой модели v1.3
- нельзя создать check до полной оплаты precheck
- нельзя использовать check как рабочий счет гостя
- нельзя переплатить precheck без явной политики
- нельзя открыть cash session без активной смены
- нельзя записать cash drawer event без active cash session
- нельзя удалять справочники
- нельзя пропустить запись в `local_event_log`
- нельзя пропустить запись в outbox

---

# Database Rules

- SQLite - primary storage для POS Edge.
- При открытии POS Edge backend выполняет fail-fast SQLite runtime gate: проверяет `sqlite_version()`, `journal_mode=WAL`, `synchronous=NORMAL`, `foreign_keys=ON`, `busy_timeout >= 5000`.
- SQLite runtime baseline: functional minimum `>= 3.37.0`; production WAL pilot baseline `>= 3.51.3` или явно разрешенный pinned backport `3.50.7/3.44.6`.
- Использовать транзакции всегда для write операций; SQLite write transactions открываются через `BEGIN IMMEDIATE`.
- Не делать частичных записей.
- До первого production launch не нужна миграция реальных production данных.
- Активный SQLite migration path для первого пилота - один canonical `001_init.sql`, который сразу создает текущую runtime-схему `Order -> Precheck -> Payment -> Check`.
- Изменения схемы v1.3 проектируются как first-launch schema, пока нет клиентских production БД; не добавлять backward compatibility вокруг старых dev-миграций.

---

# Testing Rules

Минимум:

- инварианты
- бизнес-ограничения
- idempotency foundation
- transactional `local_event_log` + outbox

Обязательно тестировать:

- duplicate actions
- invalid state transitions
- boundary cases
- запрет создания check до полной оплаты precheck в целевой v1.3 реализации
- запрет редактирования issued precheck

---

# Coding Standards

## Общие

- код должен быть читаемым;
- без магии;
- без overengineering;
- документация, планы и task comments по проекту пишутся на русском языке, если пользователь не попросил иначе;
- имена Go-пакетов, SQL-таблиц, JSON fields, enum values и endpoints остаются на английском.

## Domain Layer

- только бизнес-логика;
- без инфраструктуры;
- явные ошибки.

## Use Cases

- orchestrate domain;
- управляют транзакциями;
- не содержат SQL.

## Repositories

- только интерфейсы в ports;
- реализация в infra.

## HTTP Layer

- thin handlers;
- validation + mapping;
- без бизнес-логики.

---

# Anti-Patterns

Запрещено:

- бизнес-логика в handlers;
- бизнес-логика во frontend;
- прямые SQL в use cases;
- mutable состояния без контроля;
- глобальные синглтоны;
- shared mutable state;
- скрытые зависимости;
- Redis как source of truth;
- микросервисы на MVP;
- event sourcing как primary модель;
- Cloud как runtime dependency для POS writes;
- production data migration до первого запуска;
- развивать новый код вокруг старого `CreateCheck`;
- создавать `Check` до полной оплаты;
- редактировать `Precheck`;
- удалять или редактировать `Payment`.

---

# Observability

Каждый request должен иметь:

- `request_id`
- `device_id`
- `tenant_id` (в будущем)

Логировать:

- ошибки
- ключевые действия
- outbox события

---

# Future Extensions

Текущий код должен позволять добавить без переписывания:

- cloud sync
- inventory stock ledger
- recipes + versions
- DishServed -> write-off
- fiscalization
- multi-tenant cloud
- reporting

---

# Decision Rules

При любом архитектурном выборе:

1. Работает ли это offline?
2. Будет ли это idempotent?
3. Не ломает ли это инварианты?
4. Можно ли это синхронизировать позже?
5. Не добавляет ли это скрытую связанность?
6. Соответствует ли это модели `Order -> Precheck -> Payment -> Check`?

Если ответ "нет" - решение неверное.

---

# Naming Conventions

Использовать:

- `Order`, не `OrderEntity`
- `CreateOrder`, не `CreateOrderUseCaseImpl`
- `Repository`, не `RepoManager`
- `ID`, не `Id`
- `IssuePrecheck` для целевого precheck flow
- `Check` только для final check semantics

---

# Final Rule

Если есть сомнение:

выбирай **простоту + явность + инварианты**

а не "гибкость" или "универсальность".

В конце каждой итерации агент обязан синхронизировать `AGENTS.md`, `README.md` и при изменении backend API `pos-backend/README.md` с фактическим состоянием репозитория, если структура, статус или доступные capabilities изменились.

---

# Status

- Version: 1.3 Architecture Lock
- Scope: POS Edge Backend + minimal Cloud Sync Receiver foundation
- Target financial model: `Order -> Precheck -> Payment -> Check`
- Current POS Edge code: public `Order -> Precheck -> Payment -> Check` runtime enabled; legacy check payment endpoint is disabled
- SQLite clean install: active migration path contains canonical `001_init.sql`; it creates `prechecks`, `payments.precheck_id`, retry-safe `pos_sync_outbox`, `local_event_log.command_id` and `manager_override_audit` immediately, without legacy `payments.check_id`
- Edge foundation: `local_event_log` + retry-safe `pos_sync_outbox` + cash sessions + payment attempts + prechecks lifecycle foundation with public issue/read/list/cancel endpoints, manager override audit, precheck payments and automatic final check generation
- Operational sync endpoints: outbox, local events, aggregated sync status and manual retry failed/suspended
- Cloud: minimal `cloud-backend/` Sync Receiver implemented; Cloud is not a runtime dependency for critical POS Edge writes

---

# Cloud Sync Receiver Status

Implemented Cloud scope:

- Go entrypoint: `cloud-backend/cmd/cloud-api`
- PostgreSQL bootstrap and migrations: `cloud-backend/migrations/postgres`
- Health endpoint: `GET /health`
- Edge event receive endpoint: `POST /api/v1/sync/edge-events`
- Accepted current legacy Edge events: `ShiftOpened`, `ShiftClosed`, `OrderCreated`, `OrderLineAdded`, `CheckCreated`, `PaymentCaptured`, `OrderClosed`, `CashSessionOpened`, `CashSessionClosed`, `CashDrawerEventRecorded`
- Idempotent insert/dedupe using `restaurant_id:device_id:edge_event_id`
- Raw full `SyncEnvelope` storage
- Stable ack on duplicate replay

Important invariant: Cloud receiver must not become a runtime dependency for critical POS Edge writes. Edge still works offline; Cloud only receives outbox events asynchronously.
