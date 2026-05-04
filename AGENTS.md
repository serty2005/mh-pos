# AGENTS.md

## 🧠 Purpose

Этот документ определяет архитектурные принципы, правила разработки и доменные инварианты системы POS/RMS.

Он обязателен для:
- разработчиков
- AI-агентов (Codex, ChatGPT и др.)
- code review
- архитектурных решений

Любые изменения архитектуры должны быть согласованы с этим документом.

---

# 🧭 System Overview

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

📌 **Главная цель:** работать без интернета.

---

### 2. Cloud Platform (будет позже)

- Cloud Backend (Go + PostgreSQL)
- Back Office UI
- Reporting
- Integrations
- Fiscalization

📌 **Главная цель:** учет, аналитика и консистентность данных.

---

# 🎯 Architectural Principles

## 1. Edge-first

- POS должен работать без сети
- Интернет — это улучшение, не зависимость
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

### POS Edge Node — source of truth для:
- активных заказов
- чеков (до sync)
- оплат
- смен

### Cloud Backend — source of truth для:
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

```

domain → app → ports → infra

```

Запрещено:

- domain → infra
- domain → database/sql
- domain → http

---

# 🧱 POS Edge Backend Scope

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
- sync outbox

НЕ включает (пока):

- inventory
- recipes
- cloud sync
- fiscalization

---

# 📦 Domain Rules

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
  - restaurant_id
  - device_id
  - shift_id

### Ограничения:

- нельзя редактировать закрытый заказ
- нельзя добавить позицию в закрытый заказ

---

## Чеки и оплаты

Связи:

```

Order → Check → Payments

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

# 🔄 Sync Model (Foundation)

## Outbox Pattern

Все действия пишутся в:

```

pos_sync_outbox

```

Каждый command:

- command_id
- device_id
- aggregate_type
- aggregate_id
- payload
- status

## Принципы:

- idempotent
- retry-safe
- append-only

---

# ⚖️ Invariants (Critical)

НАРУШЕНИЕ = BUG

- ❌ нельзя открыть 2 смены на device
- ❌ нельзя создать заказ без смены
- ❌ нельзя закрыть смену с открытыми заказами
- ❌ нельзя изменить закрытый заказ
- ❌ нельзя закрыть заказ без оплаты
- ❌ нельзя переплатить чек
- ❌ нельзя удалять справочники
- ❌ нельзя пропустить запись в outbox

---

# 🧠 ID Strategy

Использовать:

- ULID или UUID

Правила:

- генерируются backend
- глобально уникальны
- не зависят от БД

---

# 🕒 Time

- всегда хранить в UTC
- использовать `created_at`, `updated_at`
- бизнес-время хранить отдельно (например `opened_at`)

---

# 💾 Database Rules (SQLite)

- SQLite — primary storage для POS
- использовать транзакции ВСЕГДА для write операций
- не делать “частичных” записей

---

# 🔐 Transactions

Каждый use case:

```

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

# 🧪 Testing Rules

Минимум:

- инварианты
- бизнес-ограничения
- idempotency foundation

Обязательно тестировать:

- duplicate actions
- invalid state transitions
- boundary cases

---

# 🧰 Coding Standards

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

# 🚫 Anti-Patterns

Запрещено:

- ❌ бизнес-логика в handlers
- ❌ прямые SQL в use cases
- ❌ mutable состояния без контроля
- ❌ глобальные синглтоны
- ❌ shared mutable state
- ❌ скрытые зависимости
- ❌ Redis как source of truth
- ❌ микросервисы на MVP
- ❌ event sourcing как primary модель

---

# 📊 Observability (Foundation)

Каждый request должен иметь:

- request_id
- device_id
- tenant_id (в будущем)

Логировать:

- ошибки
- ключевые действия
- outbox события

---

# 🔮 Future Extensions (Design Constraints)

Текущий код должен позволять добавить без переписывания:

- cloud sync
- inventory (stock ledger)
- recipes + versions
- DishServed → write-off
- fiscalization
- multi-tenant cloud
- reporting

---

# 🧭 Decision Rules

При любом архитектурном выборе:

1. Работает ли это offline?
2. Будет ли это idempotent?
3. Не ломает ли это инварианты?
4. Можно ли это синхронизировать позже?
5. Не добавляет ли это скрытую связанность?

Если ответ “нет” → решение неверное.

---

# 🧩 Naming Conventions

Использовать:

- `Order`, не `OrderEntity`
- `CreateOrder`, не `CreateOrderUseCaseImpl`
- `Repository`, не `RepoManager`
- `ID`, не `Id`

---

# 📌 Final Rule

Если есть сомнение:

👉 выбирай **простоту + явность + инварианты**

а не “гибкость” или “универсальность”.

---

# 📎 Status

- Version: 1.0
- Scope: POS Edge Backend (Phase 1)
- Cloud: not implemented yet