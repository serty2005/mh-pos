# AGENTS.md

## Быстрый Старт Для Агентов

Перед любыми изменениями держи в голове главные инварианты: POS должен работать offline, все write use case выполняются в транзакции, `local_event_log` и outbox пишутся в той же транзакции, закрытые заказы не меняются, смена на device может быть только одна активная.

### Карта Репозитория

```text
.
|-- README.md                 # карта монорепозитория и команды входа
|-- AGENTS.md                 # этот документ: правила архитектуры и навигация
|-- pos-backend/              # текущая основная кодовая база
|   |-- README.md             # запуск, Docker, smoke test, примеры API
|   |-- cmd/pos-edge/         # main() локального POS Edge Backend
|   |-- internal/platform/    # clock, http helpers, idgen, sqlite, tx
|   |-- internal/pos/api/     # HTTP router и thin handlers
|   |-- internal/pos/app/     # use cases, транзакции, outbox orchestration
|   |-- internal/pos/domain/  # бизнес-модели, ошибки, инварианты
|   |-- internal/pos/ports/   # интерфейсы репозиториев
|   |-- internal/pos/infra/   # реализации портов, сейчас SQLite
|   |-- migrations/sqlite/    # schema migrations, включая local_event_log и pos_sync_outbox
|   `-- docs/                 # проектные отчеты backend
|-- .codex/skills/            # локальные Codex skills
|-- pack_go_files.py          # вспомогательный скрипт упаковки Go-файлов
`-- project_dump.py           # вспомогательный скрипт дампа проекта
```

### Куда Идти За Чем

- Запустить backend: `pos-backend/README.md`
- Найти HTTP endpoints: `pos-backend/internal/pos/api/router.go`
- Добавить или изменить use case: `pos-backend/internal/pos/app/<context>/service.go`
- Проверить бизнес-правило: `pos-backend/internal/pos/domain/<context>/`
- Добавить repository contract: `pos-backend/internal/pos/ports/`
- Реализовать SQLite storage: `pos-backend/internal/pos/infra/sqlite/`
- Изменить схему БД: `pos-backend/migrations/sqlite/`
- Проверить read-only sync endpoints: `GET /api/v1/sync/outbox`, `GET /api/v1/sync/local-events`
- Проверить архитектурные import rules: `pos-backend/internal/pos/architecture_test.go`
- Проверить schema/invariant tests: `pos-backend/internal/pos/infra/sqlite/schema_test.go`
- Посмотреть отчет по текущей фазе: `pos-backend/docs/phase-1-report.md`

### Команды

```powershell
cd pos-backend
go test ./...
go run ./cmd/pos-edge
docker compose up --build
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
- AI-агентов (Codex, ChatGPT и др.)
- code review
- архитектурных решений

Любые изменения архитектуры должны быть согласованы с этим документом.

---

# System Overview

Система построена как **edge-first POS/RMS платформа**.

## Основные компоненты:

### 1. POS Edge Node (локальный кассовый узел)

Работает на:

- Windows
- Linux
- Android

Содержит:

- POS UI (интерфейс)
- POS Edge Backend (Go + SQLite)
- device adapters (принтеры и т.д.)
- sync outbox

Главная цель: работать без интернета.

---

### 2. Cloud Platform (будет позже)

- Cloud Backend (Go + PostgreSQL)
- Back Office UI
- Reporting
- Integrations
- Fiscalization

Главная цель: учет, аналитика и консистентность данных.

---

# Architectural Principles

## 1. Edge-first

- POS должен работать без сети
- Интернет - это улучшение, не зависимость
- Все критические операции выполняются локально

---

## 2. Offline-first

- все действия пишутся локально
- синхронизация асинхронная
- повторная отправка безопасна

---

## 3. Eventual Consistency

- cloud и edge могут временно расходиться
- консистентность достигается через sync

---

## 4. Source of Truth

### POS Edge Node - source of truth для:

- активных заказов
- чеков (до sync)
- оплат
- смен

### Cloud Backend - source of truth для:

- меню
- сотрудников
- цен
- рецептов
- склада
- отчетности

---

## 5. Modular Monolith

- нет микросервисов на MVP
- модули изолированы
- явные границы контекстов

---

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

Текущая реализация включает:

- restaurants
- devices
- employees
- roles
- catalog
- menu
- orders
- checks
- payments
- shifts
- `local_event_log`
- sync outbox
- foundation для recipes/inventory/accounting в схеме и repository layer

НЕ включает (пока):

- POS UI
- cloud sync
- fiscalization
- production-grade inventory workflows
- public API для inventory/recipes

---

# Domain Rules

## Справочники

- не удаляются
- имеют `active` / `archived`
- имеют стабильные ID
- изменения должны быть sync-safe

---

## Заказы

- создаются локально
- имеют `edge_order_id`
- принадлежат:
  - `restaurant_id`
  - `device_id`
  - `shift_id`

### Ограничения:

- нельзя редактировать закрытый заказ
- нельзя добавить позицию в закрытый заказ

---

## Чеки и оплаты

Связи:

```text
Order -> Check -> Payments
```

Правила:

- чек обязателен для закрытия заказа
- заказ нельзя закрыть без оплаты
- нельзя переплатить чек (без политики)

---

## Смены

- у device только одна активная смена
- заказ требует активную смену
- нельзя закрыть смену с открытыми заказами

---

# Sync Model (Foundation)

## Outbox Pattern

Все write-действия пишутся в local edge foundation:

```text
local_event_log
pos_sync_outbox
```

`local_event_log` хранит локальные события для операционного аудита и будущей синхронизации. `pos_sync_outbox` хранит команды/события для retry-safe отправки. Одна write-операция использует один `command_id`: он хранится в `local_event_log`, в `pos_sync_outbox` и внутри `SyncEnvelope` payload.

Read-only operational endpoints:

```text
GET /api/v1/sync/outbox?limit=50
GET /api/v1/sync/local-events?limit=50&event_type=OrderCreated
```

Каждый command:

- `command_id`
- `event_id`
- `envelope_version`
- `device_id`
- `aggregate_type`
- `aggregate_id`
- `payload`
- `status`

## Принципы:

- idempotent
- retry-safe
- append-only

---

# Invariants (Critical)

НАРУШЕНИЕ = BUG

- нельзя открыть 2 смены на device
- нельзя создать заказ без смены
- нельзя закрыть смену с открытыми заказами
- нельзя изменить закрытый заказ
- нельзя закрыть заказ без оплаты
- нельзя переплатить чек
- нельзя удалять справочники
- нельзя пропустить запись в outbox

---

# ID Strategy

Использовать:

- ULID или UUID

Правила:

- генерируются backend
- глобально уникальны
- не зависят от БД

---

# Time

- всегда хранить в UTC
- использовать `created_at`, `updated_at`
- бизнес-время хранить отдельно, например `opened_at`

---

# Database Rules (SQLite)

- SQLite - primary storage для POS
- использовать транзакции ВСЕГДА для write операций
- не делать частичных записей

---

# Transactions

Каждый use case:

```text
BEGIN
business logic
repository writes
outbox write
COMMIT
```

Запрещено:

- писать в outbox вне транзакции
- split транзакции

---

# Testing Rules

Минимум:

- инварианты
- бизнес-ограничения
- idempotency foundation

Обязательно тестировать:

- duplicate actions
- invalid state transitions
- boundary cases

---

# Coding Standards

## Общие

- код должен быть читаемым
- без магии
- без overengineering

---

## Domain Layer

- только бизнес-логика
- без инфраструктуры
- явные ошибки

---

## Use Cases

- orchestrate domain
- управляют транзакциями
- не содержат SQL

---

## Repositories

- только интерфейсы в ports
- реализация в infra

---

## HTTP Layer

- thin handlers
- validation + mapping
- без бизнес-логики

---

# Anti-Patterns

Запрещено:

- бизнес-логика в handlers
- прямые SQL в use cases
- mutable состояния без контроля
- глобальные синглтоны
- shared mutable state
- скрытые зависимости
- Redis как source of truth
- микросервисы на MVP
- event sourcing как primary модель

---

# Observability (Foundation)

Каждый request должен иметь:

- `request_id`
- `device_id`
- `tenant_id` (в будущем)

Логировать:

- ошибки
- ключевые действия
- outbox события

---

# Future Extensions (Design Constraints)

Текущий код должен позволять добавить без переписывания:

- cloud sync
- inventory (stock ledger)
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

Если ответ "нет" - решение неверное.

---

# Naming Conventions

Использовать:

- `Order`, не `OrderEntity`
- `CreateOrder`, не `CreateOrderUseCaseImpl`
- `Repository`, не `RepoManager`
- `ID`, не `Id`

---

# Final Rule

Если есть сомнение:

выбирай **простоту + явность + инварианты**

а не "гибкость" или "универсальность".

В конце каждой итерации агент обязан синхронизировать `AGENTS.md` и `README.md` с фактическим состоянием репозитория, если структура, статус или доступные backend capabilities изменились.

---

# Status

- Version: 1.2
- Scope: POS Edge Backend (Phase 1)
- Edge foundation: `local_event_log` + `pos_sync_outbox`
- Operational read-only endpoints: sync outbox and local events
- Cloud: not implemented yet
