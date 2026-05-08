# MyHoReCa Cloud Backend

Минимальный Cloud Sync Receiver для POS/RMS платформы.

Текущий scope:

- Go HTTP entrypoint: `cmd/cloud-api`;
- PostgreSQL bootstrap и migrations;
- `GET /health`;
- `POST /api/v1/sync/edge-events`;
- идемпотентный прием POS Edge `SyncEnvelope`;
- хранение raw envelope;
- operational event journal в PostgreSQL (`cloud_operational_events`);
- deterministic runtime projections для event type stats и shift finance foundation.

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
MH_POS_VERSION=0.1.1
```

`CLOUD_POSTGRES_DSN` обязателен.

implemented now: PostgreSQL использует ordered managed migrations из `migrations/postgres`: `001_sync_receiver.sql` задает baseline receiver storage, `002_projection_event_type_stats.sql` создает/ремонтирует required runtime projection table `cloud_projection_event_type_stats`, `003_runtime_schema_repair.sql` довыравнивает весь implemented-now runtime schema set для старых БД.
implemented now: `schema_migrations` хранит имя SQL file, checksum и status; уже примененные migrations не выполняются повторно, а новая ordered migration записывается в history после успешного apply.
implemented now: если `schema_migrations` отсутствует, содержит старую запись без checksum или не имеет новой ordered repair migration, Cloud применяет idempotent managed SQL, довыравнивает недостающие runtime-таблицы и только после успешного apply записывает checksum/status.
implemented now: startup policy использует `db_runtime_versions`; если таблица версий отсутствует, БД считается самой старой, перед safe upgrade существующей схемы создается JSONL backup snapshot таблиц `public`, а `DB version > MH_POS_VERSION` завершает startup fail-fast.
implemented now: schema verification проверяет только required runtime storage, включая `cloud_projection_event_type_stats`, `cloud_projection_shift_finance`, receiver journal/raw payload tables, provisioning packages и currency reference catalog.
planned next: projection query endpoints для dashboards не блокируют startup verification.
out of scope: ручной SQL repair вне startup migration framework; для local/dev recovery предпочтительно пересоздать БД или запустить приложение с корректным `CLOUD_POSTGRES_MIGRATIONS_DIR`.

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

Повторный duplicate replay возвращает тот же стабильный ack. implemented now: Cloud хранит raw accepted envelopes, append-safe operational event journal и минимальные deterministic projections для runtime ingestion. planned next: richer projection query APIs.

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

implemented now endpoints:
- `POST /api/v1/sync/edge-events`
- `POST /api/v1/sync/edge-events/batch` (item-level ACK)
- `PUT /api/v1/provisioning/master-data/{stream}` (store Cloud -> Edge package)
- `GET /api/v1/provisioning/master-data/{stream}?node_device_id=...` (resolve package for Edge import)

`sync_mode` по умолчанию считается `incremental`. `full_snapshot` package принимается только с `full_snapshot_reason = terminal_restaurant_changed` или `node_role_changed`.

implemented now storage:
- `cloud_projection_event_type_stats`
- `cloud_projection_shift_finance`
- `cloud_master_data_packages`
