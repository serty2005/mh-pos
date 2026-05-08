# Контракты синхронизации Edge -> Cloud v1

Документ описывает реализованный сейчас sync contract между POS Edge и Cloud Sync Receiver.

## Модель направлений

Реализовано сейчас: Edge -> Cloud отправляет только runtime operational events. POS Edge остается источником истины для кассовых runtime-операций и продолжает работать, когда Cloud недоступен.

Реализовано сейчас: Cloud -> Edge является направлением для master/reference/configuration данных. POS Edge принимает Cloud-authored master-data snapshots/incrementals через dedicated ingest API; этот flow не входит в Edge -> Cloud sender.

Реализовано сейчас: подробная ownership matrix зафиксирована в `docs/sync/directional-sync-ownership.md`.

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

Реализовано сейчас: `pos_sync_outbox.sync_direction` хранит явное направление строки: `edge_to_cloud`, `cloud_to_edge` или `local_only`. Sender отправляет только operational rows с `sync_direction = edge_to_cloud`.

Реализовано сейчас: Cloud -> Edge master-data ingestion доступен на POS Edge через backend API `POST /api/v1/sync/master-data/snapshots` и `POST /api/v1/sync/master-data/{stream}`. Apply flow выполняется application-layer сервисом `internal/pos/app/mastersync`, пишет master/read-model rows и `cloud_master_sync_state` в одной транзакции, использует origin `cloud_sync` и не создает Edge -> Cloud outbox/local events. По умолчанию apply является `incremental`; перед explicit `full_snapshot` apply POS Edge создает recoverable SQLite online backup artifact `.db`.

Реализовано сейчас: Cloud -> Edge state хранится на Edge в `cloud_master_sync_state` и sync metadata columns master tables (`cloud_version`, `cloud_updated_at`, `cloud_deleted_at`, `last_synced_at`).

## Cloud -> Edge master-data ingest

Реализовано сейчас: POS Edge принимает Cloud-authored full snapshot или incremental payload.

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
  "full_snapshot_reason": "terminal_restaurant_changed",
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

`sync_mode` может быть `incremental` или `full_snapshot`; если поле не передано, backend использует `incremental`. `full_snapshot` допустим только с `full_snapshot_reason = terminal_restaurant_changed` или `node_role_changed`. `/snapshots` выводит streams из непустых массивов. `/{stream}` применяет только stream из URL; пустой stream допустим для `incremental` checkpoint update, а пустой `full_snapshot` отклоняется до backup и до записи state.

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

Реализовано сейчас: stream apply order внутри multi-stream snapshot идет как restaurants, devices, staff, floor, catalog, menu, чтобы SQLite foreign keys могли быть удовлетворены тем же payload.

Реализовано сейчас: для `full_snapshot` POS Edge выполняет pre-validation payload, затем SQLite online backup, затем transaction apply master rows и `cloud_master_sync_state`. Ошибка backup завершает request fail-fast без частичного apply. Backup directory задается `POS_SQLITE_BACKUP_DIR`; default находится рядом с SQLite DB в `backups`.

## POS Sender

Реализовано сейчас: `pos-backend` запускает background sender worker, когда `POS_SYNC_SENDER_ENABLED` равно true. По умолчанию sender включен.

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

Реализовано сейчас: Cloud принимает этот Edge -> Cloud operational catalog:

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
PrecheckReprinted
PrecheckCancelled
PaymentCaptured
CheckCreated
CheckReprinted
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

Реализовано сейчас: financial payloads для `PaymentCaptured` и `CheckCreated` включают backend-owned `business_date_local`. `CheckCreated` также включает `closed_at` и immutable `snapshot`.

Реализовано сейчас: reprint events `PrecheckReprinted` и `CheckReprinted` используют immutable snapshot payload:

```json
{
  "document_type": "check",
  "source_id": "check-id",
  "copy_marker": "COPY",
  "actor_employee_id": "employee-id",
  "reprinted_at": "2026-05-05T10:00:00Z",
  "snapshot": {}
}
```

## Хранение в Cloud

Реализовано сейчас: Cloud append-safe сохраняет принятые envelopes в:

- `cloud_edge_event_receipts`;
- `cloud_edge_event_raw_payloads`;
- `cloud_operational_events`.

`cloud_edge_event_raw_payloads` сохраняет полный raw envelope. `cloud_operational_events` является operational replay journal для projections.

Реализовано сейчас: item-level ACKs поддерживаются batch endpoint `POST /api/v1/sync/edge-events/batch`; Cloud receiver пишет deterministic projections `cloud_projection_event_type_stats` и `cloud_projection_shift_finance` во время accepted event ingest.

Запланировано далее: projection query APIs и более богатые reporting projections для dashboards.

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

## Обновление sync contract 2026-05-07

Реализовано сейчас:
- Cloud поддерживает item-level ACK batch ingest endpoint `POST /api/v1/sync/edge-events/batch`.
- POS sender поддерживает batch delivery и маппит per-item ACK status (`accepted`, `rejected`, `retryable`) в outbox lifecycle (`sent`, `suspended`, `failed/pending retry`).
- Cloud пишет deterministic projections при accepted operational events:
  - `cloud_projection_event_type_stats`
  - `cloud_projection_shift_finance`
- Cloud предоставляет production-oriented provisioning/import package endpoints для Cloud -> Edge master/reference/configuration delivery:
  - `PUT /api/v1/provisioning/master-data/{stream}`
  - `GET /api/v1/provisioning/master-data/{stream}?node_device_id=...`
- Cloud хранит provisioning payloads в `cloud_master_data_packages`.
- Provisioning stream catalog на Cloud включает: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `currencies`.
- `currencies` stream payload использует canonical active ISO 4217 catalog (`currency_code`, `currency_alpha_code`, `minor_unit`, display flags) и валидируется до apply.
- POS Edge реализует Cloud -> Edge master-data ingest streams: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.
- POS Edge создает SQLite online backup перед Cloud -> Edge explicit `full_snapshot` master-data apply; `incremental` apply backup не создает.
- Cloud/POS contract разрешает `full_snapshot` только для `terminal_restaurant_changed` или `node_role_changed`; обычные package updates должны быть `incremental`.
- POS Edge `currencies` apply находится вне текущего объема до отдельного Edge import path, storage contract и тестов; Edge runtime сейчас валидирует валюты по локальному canonical catalog.

Запланировано далее:
- добавить authorization policy для provisioning endpoints в production perimeter;
- добавить projection query APIs для ops dashboards.
