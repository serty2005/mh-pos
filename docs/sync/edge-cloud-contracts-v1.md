# Контракты синхронизации Edge -> Cloud v1

Документ описывает implemented now sync contract между POS Edge и Cloud Sync Receiver.

## Модель направлений

implemented now: Edge -> Cloud отправляет только runtime operational events. POS Edge остается источником истины для кассовых runtime-операций и продолжает работать, когда Cloud недоступен.

implemented now: Cloud -> Edge является направлением для master/reference/configuration данных. POS Edge принимает Cloud-authored master-data snapshots/incrementals через dedicated ingest API; этот flow не входит в Edge -> Cloud sender.

implemented now: подробная ownership matrix зафиксирована в `docs/sync/directional-sync-ownership.md`.

Cloud-managed/configuration сущности:

- рестораны;
- метаданные организации;
- сотрудники и права;
- меню, каталог и категории;
- залы и столы;
- налоговые профили, цены и настройки;
- provisioning/configuration;
- справочная основа inventory.

Edge-managed operational сущности и события:

- аудит auth/session;
- события состояния устройств;
- личные смены сотрудников и кассовые смены;
- события денежного ящика;
- заказы и изменения заказов;
- пречеки;
- оплаты;
- финальные чеки;
- manager override, audit и business events.

POS sender содержит direction gate. Если строка `pos_sync_outbox` не является Edge runtime operational event, sender не отправляет ее POST-запросом в Cloud; вместо тихого drop он переводит строку в `suspended` с явной причиной.

implemented now: `pos_sync_outbox.sync_direction` хранит явное направление строки: `edge_to_cloud`, `cloud_to_edge` или `local_only`. Sender отправляет только operational rows с `sync_direction = edge_to_cloud`.

implemented now: Cloud -> Edge master-data ingestion доступен на POS Edge через backend API `POST /api/v1/sync/master-data/snapshots` и `POST /api/v1/sync/master-data/{stream}`. Apply flow выполняется application-layer сервисом `internal/pos/app/mastersync`, пишет master/read-model rows и `cloud_master_sync_state` в одной транзакции, использует origin `cloud_sync` и не создает Edge -> Cloud outbox/local events.

implemented now: Cloud -> Edge state хранится на Edge в `cloud_master_sync_state` и sync metadata columns master tables (`cloud_version`, `cloud_updated_at`, `cloud_deleted_at`, `last_synced_at`).

## Cloud -> Edge master-data ingest

implemented now: POS Edge принимает Cloud-authored full snapshot или incremental payload.

Endpoints:

```text
POST /api/v1/sync/master-data/snapshots
POST /api/v1/sync/master-data/{stream}
Content-Type: application/json
```

Supported `{stream}` values:

```text
restaurants
devices
staff
floor
catalog
menu
```

Request body shape:

```json
{
  "node_device_id": "edge-node-device-id",
  "restaurant_id": "restaurant-id",
  "sync_mode": "full_snapshot",
  "checkpoint_token": "optional-cloud-checkpoint",
  "cloud_version": 42,
  "cloud_updated_at": "2026-05-07T10:00:00Z",
  "restaurants": [],
  "devices": [],
  "roles": [],
  "employees": [],
  "halls": [],
  "tables": [],
  "catalog_items": [],
  "menu_items": []
}
```

`sync_mode` может быть `full_snapshot` или `incremental`; если поле не передано, backend использует `full_snapshot`. `/snapshots` infers streams from non-empty arrays. `/{stream}` applies only the stream from the URL and may apply an empty stream to update checkpoint state.

Response body:

```json
{
  "node_device_id": "edge-node-device-id",
  "applied_at": "2026-05-07T10:01:00Z",
  "applied_streams": ["catalog"],
  "counts": {"catalog": 1},
  "sync_states": []
}
```

implemented now: stream apply order inside a multi-stream snapshot is restaurants, devices, staff, floor, catalog, menu so SQLite foreign keys can be satisfied by the same payload.

## POS Sender

implemented now: `pos-backend` запускает background sender worker, когда `POS_SYNC_SENDER_ENABLED` равно true. По умолчанию sender включен.

Переменные окружения:

```text
POS_SYNC_SENDER_ENABLED=true
POS_CLOUD_SYNC_URL=http://localhost:8090/api/v1/sync/edge-events
POS_SYNC_SENDER_ID=pos-sync-sender-main
POS_SYNC_SENDER_BATCH_SIZE=25
POS_SYNC_SENDER_POLL_INTERVAL=2s
POS_SYNC_SENDER_RECLAIM_AFTER=5m
POS_SYNC_SENDER_SEND_TIMEOUT=10s
```

Модель доставки:

```text
at-least-once delivery
idempotent Cloud receive
sequence-aware local claiming
automatic retry with exponential backoff
processing lock reclaim after crash
non-retryable failures become suspended
```

Поведение retry:

- retryable network/HTTP 429/5xx ошибки возвращают строку в `pending`;
- `attempts` увеличивается;
- `next_retry_at` использует exponential backoff от 1 минуты до 30 минут;
- после более чем 20 попыток строка переходит в `suspended`;
- устаревшие `processing` locks переclaim'иваются worker-ом;
- `POST /api/v1/sync/retry-failed` возвращает `failed`/`suspended` строки в `pending` для ручного восстановления.

## Endpoint приема

```text
POST /api/v1/sync/edge-events
Content-Type: application/json
```

Тело запроса - один `SyncEnvelope`.

implemented now: Cloud принимает этот Edge -> Cloud operational catalog:

```text
ShiftOpened
ShiftClosed
CashSessionOpened
CashSessionClosed
CashDrawerEventRecorded
OrderCreated
OrderLineAdded
OrderLineQuantityChanged
OrderLineVoided
PrecheckIssued
PrecheckCancelled
PaymentCaptured
CheckCreated
OrderClosed
AuthSessionStarted
AuthSessionRevoked
DeviceRegistered
```

## SyncEnvelope

```json
{
  "version": "1",
  "event_id": "edge-generated-event-id",
  "command_id": "edge-command-id",
  "event_type": "OrderCreated",
  "aggregate_type": "Order",
  "aggregate_id": "order-id",
  "restaurant_id": "restaurant-id",
  "device_id": "node-device-id",
  "node_device_id": "node-device-id",
  "client_device_id": "client-device-id",
  "actor_employee_id": "employee-id",
  "session_id": "session-id",
  "shift_id": "shift-id",
  "occurred_at": "2026-05-05T09:00:00Z",
  "payload": {
    "origin": "edge_device",
    "data": {}
  }
}
```

Связи:

```text
command_id
  генерируется один раз на Edge write use case
  может быть общим для нескольких local events из одного write use case
  сохраняется в строках local_event_log
  сохраняется в строках pos_sync_outbox
  копируется в каждый SyncEnvelope.command_id

event_id
  генерируется один раз для локального Edge event
  сохраняется в local_event_log
  копируется в SyncEnvelope.event_id

edge_event_id
  Cloud-side имя для SyncEnvelope.event_id
  используется в Cloud idempotency key
```

`payload.data` - JSON-представление соответствующего Edge domain object или event payload.

## Хранение в Cloud

implemented now: Cloud append-safe сохраняет принятые envelopes в:

- `cloud_edge_event_receipts`;
- `cloud_edge_event_raw_payloads`;
- `cloud_operational_events`.

`cloud_edge_event_raw_payloads` сохраняет полный raw envelope до будущей projection logic. `cloud_operational_events` является operational replay journal для последующих projections.

planned next: item-level ACKs и более богатые Cloud projections.

## Правила идемпотентности

Текущий MVP использует instance-per-tenant, поэтому `organization_id` или `tenant_id` пока не входят в implemented key.

```text
idempotency_key = restaurant_id + ":" + device_id + ":" + edge_event_id
edge_event_id = SyncEnvelope.event_id
```

Поведение replay:

```text
same idempotency_key + same raw envelope hash -> return original ack
same idempotency_key + different raw envelope hash -> reject as conflict
```

## Ack

Cloud возвращает HTTP `202 Accepted` и для первого успешного приема, и для безопасного duplicate replay.

```json
{
  "status": "accepted",
  "idempotency_key": "restaurant-id:device-id:edge-event-id",
  "cloud_receipt_id": "cloud-generated-receipt-id",
  "command_id": "edge-command-id",
  "event_id": "edge-generated-event-id",
  "edge_event_id": "edge-generated-event-id",
  "envelope_version": "1",
  "cloud_received_at": "2026-05-05T10:00:00Z",
  "raw_payload_sha256_hex": "..."
}
```

Ack стабилен при replay: повторный POST того же envelope возвращает те же `cloud_receipt_id`, timestamps, ids и payload hash.

## Sync Contract Update 2026-05-07

implemented now:
- Cloud supports item-level ACK batch ingest endpoint `POST /api/v1/sync/edge-events/batch`.
- POS sender supports batch delivery and maps per-item ACK status (`accepted`, `rejected`, `retryable`) to outbox lifecycle (`sent`, `suspended`, `failed/pending retry`).
- Cloud writes richer deterministic projections on accepted operational events:
  - `cloud_projection_event_type_stats`
  - `cloud_projection_shift_finance`
- Cloud exposes production-oriented provisioning/import package endpoints for Cloud -> Edge master/reference/configuration delivery:
  - `PUT /api/v1/provisioning/master-data/{stream}`
  - `GET /api/v1/provisioning/master-data/{stream}?node_device_id=...`
- Cloud stores provisioning payloads in `cloud_master_data_packages`.
- Provisioning stream catalog on Cloud includes: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `currencies`.

next:
- add authorization policy for provisioning endpoints in production perimeter;
- add projection query APIs for ops dashboards.
