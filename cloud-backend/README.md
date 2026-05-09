# MyHoReCa Cloud Backend

Cloud backend для POS/RMS платформы: прием Edge operational events, PostgreSQL runtime projections и foundation Cloud-authored master data.

Текущий scope:

- Go HTTP entrypoint: `cmd/cloud-api`;
- PostgreSQL bootstrap и migrations;
- `GET /health`;
- `POST /api/v1/sync/edge-events`;
- идемпотентный прием POS Edge `SyncEnvelope`;
- хранение raw envelope;
- operational event journal в PostgreSQL (`cloud_operational_events`);
- deterministic runtime projections для event type stats и shift finance foundation.
- реализовано сейчас: Cloud-owned production-oriented CRUD API для ресторанов, залов/столов, ролей, сотрудников/PIN credentials, catalog items, menu items и versioned master-data publications;
- реализовано сейчас: publication workflow создает deterministic Cloud -> Edge packages для stream `restaurants`, `staff`, `floor`, `catalog`, `menu` и сохраняет их в `cloud_master_data_packages`;
- реализовано сейчас: device provisioning поддерживает Cloud Approve и License Code flow для чистого подключения POS Edge без dev bootstrap;
- реализовано сейчас: Cloud UI API responses по сотрудникам и публикациям не возвращают PIN и `pin_hash`; PIN hash присутствует только внутри sync-ready staff package для device/system delivery на Edge.

## Запуск

Запусти локальный PostgreSQL:

```powershell
docker run --name mh-pos-cloud-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=mh_pos_cloud -p 5432:5432 -d postgres:16
```

```powershell
cd cloud-backend
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
MH_POS_VERSION=0.1.1
```

`CLOUD_POSTGRES_DSN` обязателен.

Реализовано сейчас: PostgreSQL использует ordered managed migrations из `migrations/postgres`: `001_sync_receiver.sql` задает baseline receiver storage, `002_projection_event_type_stats.sql` создает/ремонтирует required runtime projection table `cloud_projection_event_type_stats`, `003_runtime_schema_repair.sql` довыравнивает весь required receiver/projection/provisioning schema set для старых БД, `004_master_data_authority.sql` добавляет Cloud-owned master-data authority schema, `005_master_data_restaurants_api.sql` добавляет `cloud_restaurants`, cloud-version metadata и partial unique SKU policy для неархивных catalog items, `006_zero_to_cashier_provisioning.sql` добавляет Cloud halls/tables и device provisioning таблицы.
Реализовано сейчас: `schema_migrations` хранит имя SQL file, checksum и status; уже примененные migrations не выполняются повторно, а новая ordered migration записывается в history после успешного apply.
Реализовано сейчас: если `schema_migrations` отсутствует, содержит старую запись без checksum или не имеет новой ordered repair migration, Cloud применяет idempotent managed SQL, довыравнивает недостающие runtime-таблицы и только после успешного apply записывает checksum/status.
Реализовано сейчас: startup policy использует `db_runtime_versions`; если таблица версий отсутствует, БД считается самой старой, перед safe upgrade существующей схемы создается JSONL backup snapshot таблиц `public`, а `DB version > MH_POS_VERSION` завершает startup fail-fast.
Реализовано сейчас: schema verification проверяет required runtime storage, включая receiver journal/raw payload tables, projection tables, provisioning packages, currency reference catalog и Cloud master-data authority tables.
Запланировано далее: projection query endpoints для dashboards не блокируют startup verification.
Вне текущего объема: ручной SQL repair вне startup migration framework; для local/dev recovery предпочтительно пересоздать БД или запустить приложение с корректным `CLOUD_POSTGRES_MIGRATIONS_DIR`.

## Master Data Authority

Реализовано сейчас: Cloud является источником истины для production-oriented справочников сотрудников, ролей, каталога и меню. POS Edge не становится production CRUD для этих сущностей; Edge получает published state через Cloud -> Edge package/snapshot delivery и использует локальную read model offline.

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
POST  /api/v1/restaurants/{id}/master-data/publish
GET   /api/v1/restaurants/{id}/master-data/publication-state
GET   /api/v1/restaurants/{id}/master-data/packages/latest
GET   /api/v1/restaurants/{id}/master-data/packages/{package_id}
GET   /api/v1/restaurants/{id}/edge-nodes/{node_device_id}/master-data/snapshot
```

Совместимые legacy/foundation routes `/api/v1/master-data/...` сохранены для текущих тестов и low-level сценариев, но новый production onboarding path документируется через top-level routes выше.

Реализовано сейчас: publication endpoint не делает каждое сохранение live. Он создает versioned publication (`version`, `cloud_version`, `published_at`, `published_by`, `package_sha256`) и deterministic packages для `restaurants`, `staff`, `floor`, `catalog`, `menu`. Generated packages сохраняются в `cloud_master_data_packages`, после чего Edge может получить их через provisioning/import path или через Edge-ready snapshot endpoint.

Реализовано сейчас: employee lifecycle поддерживает `active`, `suspended`, `archived`; role assignment обновляет permission snapshot для sync-safe POS usage; PIN rotation увеличивает credential version. API responses не возвращают PIN или `pin_hash`.

Реализовано сейчас: catalog foundation разделяет `cloud_catalog_items`, `cloud_dishes`, `cloud_goods`, `cloud_semi_finished_products`, `cloud_recipe_items`, `cloud_modifier_groups`, `cloud_modifier_options`; menu foundation хранит draft/published/archived lifecycle, price, category placement, availability, основу будущего station routing и будущих multi-location assignments. Cloud catalog type `semi_finished` публикуется в текущий POS Edge-compatible catalog stream как `ingredient`; это совместимое foundation-решение до отдельного расширения Edge ingest enum.

Реализовано сейчас: raw PIN и `pin_hash` не возвращаются Cloud UI-facing API responses. API responses используют безопасное `pin_configured`; `pin_hash` присутствует только в device/system snapshot package для offline PIN auth на POS Edge. PIN должен быть уникален в рамках ресторана среди всех сотрудников не в статусе `archived`; `active` и `suspended` сотрудники удерживают PIN, archived сотрудник не блокирует повторное использование. При нарушении API возвращает `409` с `X-Error-Code: PIN_ALREADY_EXISTS`.

## Device Provisioning

Реализовано сейчас: Cloud API возвращает безопасный error envelope `{ "error": { "code", "message_key", "details", "correlation_id" } }` и всегда выставляет `X-Error-Code` для ошибок.

Option A, Cloud Approve:

```text
POST /api/v1/devices/register
GET  /api/v1/devices/unassigned
POST /api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign
GET  /api/v1/devices/{node_device_id}/assignment-status
```

`assign` проверяет active restaurant, создает/обновляет assigned edge node, при необходимости публикует master-data и возвращает snapshot URL. `assignment-status` после назначения возвращает `restaurant_id`, `cloud_url`, snapshot URL и одноразовый node token; Cloud хранит только hash/verifier.

Option B, License Server Code:

```text
POST /api/v1/restaurants/{restaurant_id}/devices/generate-pairing-code
```

Cloud генерирует короткий одноразовый code и node token, сохраняет hashes, регистрирует code в `LICENSE_SERVER_URL` и возвращает plaintext code только в response. Если License Server недоступен, возвращается `503 LICENSE_SERVER_UNAVAILABLE`.

Вне текущего объема: production authorization perimeter Cloud API. Текущие endpoints предназначены для dev/pilot perimeter и не смешиваются с POS operator auth.

## Локальный smoke test receiver-а

```powershell
Invoke-RestMethod http://localhost:8090/health
..\scripts\send-cloud-test-envelope.ps1 -ReplayTwice
```

Replay envelope с ID из POS demo bootstrap:

```powershell
$demo = ..\scripts\bootstrap-pos-demo.ps1
..\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
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

2. После POS bootstrap проверь replay с реальными ID:

```powershell
$demo = ..\scripts\bootstrap-pos-demo.ps1
..\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
```

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

Стандартные тесты используют in-memory repository для service и HTTP replay checks. PostgreSQL runtime storage реализован в `internal/cloudsync/infra/postgres`, инициализируется через ordered managed SQL files, получает advisory lock на время upgrade и проходит schema verification до запуска HTTP server.

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
- `cloud_master_data_packages`
