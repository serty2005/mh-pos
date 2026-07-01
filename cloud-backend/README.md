# MyHoReCa Cloud Backend

Cloud backend для POS/RMS платформы: прием Edge operational events, PostgreSQL runtime projections и foundation Cloud-authored master data.

Профильный контракт Cloud Backend описан в `../docs/backend/CLOUD-BACKEND-SPEC.md`.

Текущий scope:

- Go HTTP entrypoint: `cmd/cloud-api`;
- PostgreSQL bootstrap и migrations;
- `GET /health`;
- `POST /api/v1/sync/edge-events`;
- идемпотентный прием POS Edge `SyncEnvelope`;
- хранение raw envelope;
- operational event journal в PostgreSQL (`cloud_operational_events`);
- deterministic runtime projections для event type stats и shift finance/refund foundation.
- реализовано сейчас: Cloud-owned production-oriented CRUD API для ресторанов, залов/столов, ролей, сотрудников/PIN credentials, tenant-level catalog items, restaurant-scoped menu items и versioned master-data publications;
- реализовано сейчас: restaurant menu items являются overrides поверх tenant catalog identity и хранят stable `menu_item_id`, `catalog_item_id`, category/menu folder, tag, active tax profile, availability и runtime status;
- реализовано сейчас: publication workflow создает deterministic Cloud -> Edge packages для stream `restaurants`, `staff`, `floor`, `catalog`, restaurant-effective `menu` и сохраняет их в `cloud_master_data_packages`;
- реализовано сейчас: generic Cloud -> Edge package storage/validation поддерживает stream `pricing_policy` для tax/service-charge reference payloads; full Cloud UI/publication workflow для pricing/tax остается запланирован далее;
- реализовано сейчас: device provisioning поддерживает Cloud Approve и License Code flow для чистого подключения POS Edge без dev bootstrap;
- реализовано сейчас: Cloud UI API responses по сотрудникам и публикациям не возвращают PIN и `pin_hash`; PIN hash присутствует только внутри sync-ready staff package для device/system delivery на Edge.
- реализовано сейчас: Cloud sync receiver принимает inventory event catalog, пишет durable `inventory_event_queue`, а Cloud Inventory Worker создает Cloud-owned `stock_documents` и `stock_ledger` для нормализованных item payloads.

## Запуск

Запусти локальный PostgreSQL:

```powershell
docker run --name mh-pos-cloud-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=mh_pos_cloud -p 5432:5432 -d postgres:16
```

```powershell
cd cloud-backend
$env:CLOUD_CONFIG_PATH="config/cloud-api.json" # optional; файл имеет приоритет над env
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go mod tidy
go test ./...
go run ./cmd/cloud-api
```

Значения по умолчанию:

```text
CLOUD_HTTP_ADDR=:8090
CLOUD_POSTGRES_MIGRATIONS_DIR=migrations/postgres
CLOUD_POSTGRES_BACKUP_DIR=data/cloud-backups
CLOUD_PUBLIC_URL=http://localhost:8090
LICENSE_SERVER_URL=http://localhost:8095
MH_POS_VERSION=0.1.13
```

`CLOUD_POSTGRES_DSN` обязателен.

Реализовано сейчас: Cloud Backend также читает optional `config/cloud-api.json`; пример полного файла находится в `config/cloud-api.example.json`. Если `CLOUD_CONFIG_PATH` задан явно, файл обязателен. Порядок приоритета: defaults -> env -> JSON-файл. Общий контракт описан в `../docs/backend/RUNTIME-CONFIG.md`.

Реализовано сейчас: PostgreSQL использует managed migrations из `migrations/postgres`; в pre-pilot режиме активен один схлопнутый baseline `001_init.sql`, который содержит receiver storage, projection tables, Cloud-owned master-data authority schema, restaurants API tables, provisioning tables, refund event catalog, refund finance projection columns, `pricing_policy` package stream, `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs` и `stop_lists`.
Реализовано сейчас: `schema_migrations` хранит имя SQL file, checksum и status; уже примененный baseline не выполняется повторно.
Реализовано сейчас: до первого клиента существующие dev/test БД не поддерживаются как data-preserving upgrade path и пересоздаются из baseline. Если active baseline меняется, `MH_POS_VERSION` повышается, чтобы startup policy не принимала checksum drift как ту же runtime-версию; для local/dev recovery предпочтительно пересоздать Cloud PostgreSQL volume из актуального baseline.
Реализовано сейчас: startup policy использует `db_runtime_versions`; checksum drift при той же версии останавливает startup, а `DB version > MH_POS_VERSION` завершает startup fail-fast.
Реализовано сейчас: schema verification проверяет required runtime storage, включая receiver journal/raw payload tables, projection tables, provisioning packages, currency reference catalog и Cloud master-data authority tables.
Запланировано далее: projection query endpoints для dashboards не блокируют startup verification.
Вне текущего объема: ручной SQL repair вне startup migration framework; для local/dev recovery предпочтительно пересоздать БД или запустить приложение с корректным `CLOUD_POSTGRES_MIGRATIONS_DIR`.

## Master Data Authority

Реализовано сейчас: Cloud является источником истины для production-oriented справочников сотрудников, ролей, tenant-level каталога и restaurant menu overrides. POS Edge не становится production CRUD для этих сущностей; Edge получает published state через Cloud -> Edge package/snapshot delivery и использует локальную read model offline.

Реализовано сейчас: один `catalog_item_id` может использоваться несколькими ресторанами через разные `menu_item_id`. Изменение name/price/tag/tax/menu folder/availability/runtime status у одного menu item не меняет другое меню; Edge package `menu` содержит только menu items выбранного ресторана.

Cloud master-data production API для будущего `cloud-ui`:

```text
POST  /api/v1/restaurants
GET   /api/v1/restaurants
GET   /api/v1/restaurants/{id}
PATCH /api/v1/restaurants/{id}
POST  /api/v1/restaurants/{id}/archive
POST  /api/v1/roles
GET   /api/v1/roles?restaurant_id=...
GET   /api/v1/roles/{id}
PATCH /api/v1/roles/{id}
POST  /api/v1/roles/{id}/archive
POST  /api/v1/employees
GET   /api/v1/employees?restaurant_id=...
GET   /api/v1/employees/{id}
PATCH /api/v1/employees/{id}
POST  /api/v1/employees/{id}/suspend
POST  /api/v1/employees/{id}/activate
POST  /api/v1/employees/{id}/archive
POST  /api/v1/employees/{id}/pin
POST  /api/v1/employees/{id}/pin/rotate
POST  /api/v1/halls
GET   /api/v1/halls?restaurant_id=...
PATCH /api/v1/halls/{id}
POST  /api/v1/halls/{id}/archive
POST  /api/v1/tables
GET   /api/v1/tables?restaurant_id=...
PATCH /api/v1/tables/{id}
POST  /api/v1/tables/{id}/archive
POST  /api/v1/catalog/items
GET   /api/v1/catalog/items?restaurant_id=...
GET   /api/v1/catalog/items/{id}
PATCH /api/v1/catalog/items/{id}
POST  /api/v1/catalog/items/{id}/archive
POST  /api/v1/menu/items
GET   /api/v1/menu/items?restaurant_id=...
GET   /api/v1/menu/items/{id}
PATCH /api/v1/menu/items/{id}
POST  /api/v1/menu/items/{id}/archive
POST  /api/v1/master-data/menu/categories
GET   /api/v1/master-data/menu/categories?restaurant_id=...
PATCH /api/v1/master-data/menu/categories/{id}
POST  /api/v1/master-data/menu/categories/{id}/archive
GET   /api/v1/restaurants/{id}/master-data/publication-state
GET   /api/v1/restaurants/{id}/master-data/packages/latest
GET   /api/v1/restaurants/{id}/master-data/packages/{package_id}
GET   /api/v1/restaurants/{id}/master-data/delivery-status
GET   /api/v1/restaurants/{id}/edge-nodes/{node_device_id}/master-data/snapshot
```

Совместимые legacy/foundation routes `/api/v1/master-data/...` сохранены для текущих тестов и low-level сценариев, но новый production onboarding path документируется через top-level routes выше.

Реализовано сейчас: user-facing manual publish endpoint удален. До Edge assignment delivery packages не создаются. При assignment/pairing Cloud собирает current full batch для конкретного `node_device_id`; после подтвержденных Cloud master-data commits обновляется только latest row в `cloud_master_data_packages` для назначенных Edge и stream. Event backlog на каждое изменение не ведется.

Реализовано сейчас: `GET /api/v1/restaurants/{id}/master-data/publication-state` до первого assignment/automatic refresh возвращает `200` с JSON `null`, чтобы Cloud UI показывал ожидаемый empty state без browser-console 404 noise.

Реализовано сейчас: employee lifecycle поддерживает `active`, `suspended`, `archived`; role assignment обновляет permission snapshot для sync-safe POS usage; PIN rotation увеличивает credential version. API responses не возвращают PIN или `pin_hash`.

Реализовано сейчас: catalog foundation разделяет `cloud_catalog_items`, `cloud_dishes`, `cloud_goods`, `cloud_semi_finished_products`, `cloud_services`, `cloud_recipe_items`, `cloud_modifier_groups`, `cloud_modifier_options`; canonical catalog item kinds: `dish`, `good`, `semi_finished`, `service`. Menu foundation хранит draft/published/archived lifecycle, price, category placement, availability, основу будущего station routing и будущих multi-location assignments. Cloud catalog kinds публикуются в POS Edge catalog stream без legacy `ingredient` mapping; categories пока хранятся в Cloud foundation, но не публикуются в Edge package до появления поддержанного Edge ingest contract.

Реализовано сейчас: raw PIN и `pin_hash` не возвращаются Cloud UI-facing API responses. API responses используют безопасное `pin_configured`; `pin_hash` присутствует только в device/system snapshot package для offline PIN auth на POS Edge. PIN должен быть уникален в рамках ресторана среди всех сотрудников не в статусе `archived`; `active` и `suspended` сотрудники удерживают PIN, archived сотрудник не блокирует повторное использование. При нарушении API возвращает `409` с `X-Error-Code: PIN_ALREADY_EXISTS`.

## Device Provisioning

Реализовано сейчас: Cloud API возвращает безопасный error envelope `{ "error": { "code", "message_key", "details", "correlation_id" } }` и всегда выставляет `X-Error-Code` для ошибок.

Option A, Cloud Approve:

```text
POST /api/v1/devices/register
GET  /api/v1/devices/unassigned
GET  /api/v1/restaurants/{restaurant_id}/devices
POST /api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign
GET  /api/v1/devices/{node_device_id}/assignment-status
```

`assign` проверяет active restaurant, создает/обновляет assigned edge node, при необходимости публикует master-data и возвращает snapshot URL. `assignment-status` после ручного назначения возвращает `restaurant_id`, snapshot URL и одноразовый node token; Cloud не переопределяет `cloud_url`, с которым Edge уже зарегистрировался, и хранит только hash/verifier.

Option B, License Server Code:

```text
POST /api/v1/restaurants/{restaurant_id}/devices/generate-pairing-code
POST /api/v1/devices/pairing/consume
```

Cloud генерирует короткий одноразовый code и `pairing_id`, сохраняет hash и derived decrypt key, регистрирует invitation в `LICENSE_SERVER_URL` и возвращает plaintext code только в response. POS Edge после resolve в License Server вызывает encrypted consume в Cloud; только consume назначает Edge node и выдает node token. Если License Server недоступен, возвращается `503 LICENSE_SERVER_UNAVAILABLE`.

Вне текущего объема: production authorization perimeter Cloud API. Текущие endpoints предназначены для dev/pilot perimeter и не смешиваются с POS operator auth.

## Локальная проверка receiver-а

```powershell
Invoke-RestMethod http://localhost:8090/health
```

Для полного локального заполнения данных и привязки POS Edge используется единый seed-скрипт из корня репозитория:

```powershell
python ..\scripts\seed-dev-system.py --cloud-base http://localhost:8090 --pos-base http://localhost:8080 --license-base http://localhost:8095
```
Минимальное тело, эквивалентное curl-запросу `POST /api/v1/sync/edge-events`:

```powershell
$body = @{
  version = "1"
  event_id = "demo-cloud-replay-event-1"
  command_id = "demo-cloud-replay-command-1"
  event_type = "OrderCreated"
  aggregate_type = "Order"
  aggregate_id = "demo-order-cloud-1"
  restaurant_id = "demo-restaurant"
  device_id = "demo-edge-node-1"
  shift_id = "demo-shift-cloud-1"
  occurred_at = "2026-05-07T09:00:00Z"
  payload = @{
    origin = "edge_device"
    data = @{
      id = "demo-order-cloud-1"
      edge_order_id = "demo-edge-order-cloud-1"
      restaurant_id = "demo-restaurant"
      device_id = "demo-edge-node-1"
      shift_id = "demo-shift-cloud-1"
      status = "open"
      table_name = "A1"
      guest_count = 2
      opened_at = "2026-05-07T09:00:00Z"
      created_at = "2026-05-07T09:00:00Z"
      updated_at = "2026-05-07T09:00:00Z"
    }
  }
} | ConvertTo-Json -Depth 8

Invoke-RestMethod -Method Post http://localhost:8090/api/v1/sync/edge-events -ContentType "application/json" -Body $body
Invoke-RestMethod -Method Post http://localhost:8090/api/v1/sync/edge-events -ContentType "application/json" -Body $body
```

Повторный duplicate replay возвращает тот же стабильный ack. Реализовано сейчас: Cloud хранит raw accepted envelopes, append-safe operational event journal и минимальные deterministic projections для runtime ingestion. Запланировано далее: richer projection query APIs.

## Локальный E2E Prototype: получить pairing code и войти в POS UI

реализовано сейчас: Cloud участвует в локальном прототипе как идемпотентный receiver envelope-ов.

1. Запусти Cloud с PostgreSQL:

```powershell
cd cloud-backend
$env:CLOUD_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable"
go run ./cmd/cloud-api
```

2. После полного seed проверь, что Cloud получает POS outbox события автоматически через sync sender worker. Seed summary содержит `restaurant_id` и `node_device_id` для ручных запросов, но отдельный replay-скрипт больше не поддерживается.

реализовано сейчас: POS outbox operational events автоматически доставляются в Cloud POS sender worker-ом, когда `POS_SYNC_SENDER_ENABLED=true`, а `POS_CLOUD_SYNC_URL` указывает на этот receiver.

Проверка PostgreSQL:

```powershell
docker exec -it mh-pos-cloud-postgres psql -U postgres -d mh_pos_cloud -c "select event_type, count(*) from cloud_operational_events group by event_type order by event_type;"
docker exec -it mh-pos-cloud-postgres psql -U postgres -d mh_pos_cloud -c "select idempotency_key, event_type, cloud_received_at from cloud_edge_event_receipts order by cloud_received_at desc limit 10;"
```

## Проверки

```powershell
cd cloud-backend
go test ./...
```

Стандартные тесты используют in-memory repository для service и HTTP replay checks. PostgreSQL runtime storage реализован в `internal/cloudsync/infra/postgres`, инициализируется через managed SQL baseline, получает advisory lock на время upgrade и проходит schema verification до запуска HTTP server.

## Контракт

См. `../docs/sync/edge-cloud-contracts-v1.md`.

## Sync API update 2026-05-07

Реализовано сейчас endpoints:
- `POST /api/v1/sync/edge-events`
- `POST /api/v1/sync/edge-events/batch` (item-level ACK)
- `PUT /api/v1/provisioning/master-data/{stream}` (store Cloud -> Edge package)
- `GET /api/v1/provisioning/master-data/{stream}?node_device_id=...` (resolve package for Edge import)

`sync_mode` по умолчанию считается `incremental`. `full_snapshot` package принимается только с `full_snapshot_reason = terminal_restaurant_changed` или `node_role_changed`.

Реализовано сейчас storage:
- `cloud_projection_event_type_stats`
- `cloud_projection_shift_finance`
- `cloud_projection_financial_operations`
- `cloud_sync_problem_events`
- `cloud_master_data_packages`
- `cloud_master_data_delivery_states`

Реализовано сейчас: `CLOUD_SYNC_MAX_CLOUD_PACKAGES_PER_EXCHANGE` ограничивает число Cloud -> Edge packages в одном authenticated `sync/exchange` response. Ошибочные Edge batch/exchange items получают item-level ACK и сохраняются в `cloud_sync_problem_events`, не блокируя прием остальных items.

Реализовано сейчас: scheduled `sync/exchange` доставляет Cloud package только если package `cloud_version` новее Edge-provided stream checkpoint. `cloud_master_data_delivery_states` хранит per-Edge Cloud version, Edge ACK version, lag, last sync, safe error code и bounded retry state. Ошибка assembly после authority commit не откатывает commit и повторяется при следующем exchange. Cloud UI читает `GET /api/v1/restaurants/{id}/master-data/delivery-status` и не вызывает publish API.

Реализовано сейчас financial operation sync behavior:
- `CancellationRecorded` and `RefundRecorded` are current Edge -> Cloud financial operation events.
- `PaymentRefunded` and `CheckRefunded` remain legacy inbound-only event types for older Edge payloads.
- Cloud stores raw payloads and operational journal rows idempotently for current and legacy events.
- `cloud_projection_shift_finance` tracks coarse refund counters/totals from `RefundRecorded` and legacy refund events.
- `cloud_projection_financial_operations` stores detailed current `CancellationRecorded`/`RefundRecorded` operation projection with operation/check/shift/date/type/disposition/reason/snapshot metadata; legacy refund events do not populate this primary ledger projection.
- `GET /api/v1/reporting/financial-operations` exposes a bounded read-only reporting view over this projection with restaurant/date/type/shift/original shift/check filters, `limit`/`offset`, checksum metadata and no raw sync payload or snapshot JSON.

## Pricing policy publication

Статус: реализовано сейчас.

Cloud master-data publication включает `pricing_policy` stream для Cloud-authored discounts/surcharges. Published policy payload содержит `manual`, `requires_permission`, `application_index` и amount fields, которые POS Edge использует как authoritative runtime source.
