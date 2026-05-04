# Отчет по фазе 1: POS Edge Backend Foundation

Дата: 2026-05-04

## Вердикт

Фаза 1 закрыта как foundation POS Edge Backend.

Реализован запускаемый локальный backend-сервис для POS Edge Node с SQLite persistence, JSON API, базовыми доменными моделями, application use cases, sync outbox и тестами ключевых доменных инвариантов.

Это не финальная продуктовая версия POS-системы, а корректный базовый слой, на котором можно продолжать развитие POS Edge Backend без смены архитектурного направления.

## Scope

В рамках фазы реализован только POS Edge Backend.

Не реализовывались и намеренно оставлены вне scope:

- POS UI
- Cloud Backend
- Back Office UI
- PostgreSQL
- складской учет
- рецепты
- DishServed
- фискализация
- внешние интеграции
- reporting

## Архитектура

Проект построен как modular monolith с Clean Architecture и DDD-lite разделением:

- `cmd/pos-edge` - точка входа сервиса.
- `internal/platform` - инфраструктурные компоненты: HTTP helpers, SQLite bootstrap/migrations, clock, idgen, tx manager.
- `internal/pos/domain` - доменные модели и доменные ошибки.
- `internal/pos/app` - use cases и бизнес-инварианты.
- `internal/pos/ports` - repository interfaces.
- `internal/pos/infra/sqlite` - SQLite repository implementation.
- `internal/pos/api` - HTTP handlers и routing.
- `migrations/sqlite` - SQLite schema migrations.

Доменные модели не зависят от HTTP, SQLite, `database/sql` или деталей транспорта.

Use cases управляют транзакциями, проверяют инварианты и пишут sync outbox в той же транзакции, что и основное изменение.

HTTP handlers тонкие: принимают JSON, вызывают application service, возвращают JSON response.

## Технологии

- Go 1.26.2
- `github.com/go-chi/chi/v5 v5.2.5`
- `modernc.org/sqlite v1.50.0`
- SQLite
- Docker / Docker Compose
- JSON API

## Реализованные таблицы

SQLite migration `migrations/sqlite/001_init.sql` создает:

- `restaurants`
- `devices`
- `roles`
- `employees`
- `catalog_items`
- `menu_items`
- `shifts`
- `orders`
- `order_lines`
- `checks`
- `payments`
- `pos_sync_outbox`
- `schema_migrations`

Добавлены базовые constraints и индексы, включая уникальный частичный индекс на одну открытую смену для device:

- `shifts_one_open_per_device`
- `pos_sync_outbox_status_created_at`

Денежные значения хранятся как integer minor units.

## Реализованные API

Health:

- `GET /health`

Restaurants:

- `POST /api/v1/restaurants`
- `GET /api/v1/restaurants`

Devices:

- `POST /api/v1/devices/register`
- `GET /api/v1/devices`

Employees:

- `POST /api/v1/employees`
- `GET /api/v1/employees`
- `PATCH /api/v1/employees/{id}/archive`

Roles:

- `POST /api/v1/roles`
- `GET /api/v1/roles`

Catalog:

- `POST /api/v1/catalog/items`
- `GET /api/v1/catalog/items`

Menu:

- `POST /api/v1/menu/items`
- `GET /api/v1/menu/items`

Shifts:

- `POST /api/v1/shifts/open`
- `POST /api/v1/shifts/{id}/close`
- `GET /api/v1/shifts/current?device_id=...`

Orders:

- `POST /api/v1/orders`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/lines`
- `POST /api/v1/orders/{id}/close`

Checks:

- `POST /api/v1/orders/{id}/check`
- `GET /api/v1/checks/{id}`

Payments:

- `POST /api/v1/checks/{id}/payments`

Outbox:

- `GET /api/v1/sync/outbox`
- `POST /api/v1/sync/outbox/{id}/mark-sent`
- `POST /api/v1/sync/outbox/{id}/mark-failed`

## Доменные инварианты

Реализованы проверки:

- у device не может быть больше одной открытой смены;
- нельзя создать заказ без открытой смены;
- нельзя закрыть смену с открытыми заказами;
- нельзя добавить line в закрытый заказ;
- нельзя создать check для закрытого заказа;
- нельзя закрыть заказ без check;
- нельзя закрыть заказ без полной оплаты;
- нельзя переплатить check;
- нельзя создать menu item для archived catalog item;
- справочники не удаляются, employee архивируется через `active = false`.

Часть инвариантов дополнительно поддержана на уровне SQLite constraints.

## Sync Outbox

Каждое write-действие application layer пишет запись в `pos_sync_outbox`.

Реализованные command types:

- `RestaurantCreated`
- `DeviceRegistered`
- `RoleCreated`
- `EmployeeCreated`
- `EmployeeArchived`
- `CatalogItemCreated`
- `MenuItemCreated`
- `ShiftOpened`
- `ShiftClosed`
- `OrderCreated`
- `OrderLineAdded`
- `CheckCreated`
- `PaymentCaptured`
- `OrderClosed`

Outbox содержит:

- `command_id`
- `restaurant_id`
- `device_id`
- `aggregate_type`
- `aggregate_id`
- `command_type`
- `payload_json`
- `status`
- `attempts`
- `last_error`
- timestamps

Запись outbox создается атомарно с основной write-операцией.

## Docker

Добавлены:

- `docker/Dockerfile`
- `docker-compose.yml`

Команда запуска:

```powershell
docker compose up --build
```

SQLite хранится в named volume `pos_edge_sqlite`.

## Документация

Добавлен `README.md` для POS Edge Backend:

- локальный запуск на Windows;
- VSCode рекомендации;
- Docker Compose запуск;
- smoke test;
- curl examples для базового POS workflow;
- запуск тестов.

## Тестирование

Добавлены минимальные application-level тесты:

- нельзя открыть две смены на одном device;
- нельзя создать заказ без открытой смены;
- нельзя закрыть смену с открытыми заказами;
- нельзя добавить line в закрытый заказ;
- нельзя переплатить check;
- outbox запись создается при write-действии.

Проверки на 2026-05-04:

```powershell
go test ./...
go vet ./...
go build ./cmd/pos-edge
```

Результат: все проверки прошли успешно.

Также выполнен runtime smoke test сервиса с запросом `GET /health`; результат: `health ok`.

## Известные ограничения

Текущая фаза сознательно минимальна:

- нет authentication/authorization;
- нет idempotency storage по `command_id`, кроме unique constraint outbox;
- нет sync worker, только outbox foundation и API mark-sent/mark-failed;
- нет pagination/filtering для большинства list endpoints;
- нет API для архивирования catalog/menu справочников;
- нет полноценной модели налогов, скидок, возвратов и void/refund workflow;
- нет OpenAPI спецификации;
- нет production observability.

Эти ограничения не блокируют закрытие foundation-фазы, но должны быть учтены в следующем планировании.

## Рекомендованный следующий этап

Фаза 2 может быть посвящена укреплению POS Edge Backend:

- request idempotency по `command_id`;
- authentication для локального POS Edge Node;
- расширенные repository фильтры и pagination;
- sync worker интерфейс и batch claiming для outbox;
- OpenAPI contract;
- structured logging и health/readiness;
- API-level integration tests;
- отдельные use cases для архивирования справочников;
- подготовка контрактов будущей синхронизации с Cloud Backend.
