# Edge / Cloud Sync Contracts v1

Статус: актуализировано под frozen cashier pilot.

## Direction Model

Реализовано сейчас:

- POS Edge owns cashier operational runtime data.
- Cloud owns master/reference/configuration authoring.
- POS Edge can continue cashier runtime while Cloud is unavailable.
- Directional ownership matrix is maintained in `docs/sync/directional-sync-ownership.md`.

## SyncExchange v1

Реализовано сейчас:

- `POST /api/v1/sync/exchange` является приоритетным Cloud-Edge циклом для POS Edge worker.
- Endpoint требует `Authorization: Bearer <node_token>`; Cloud проверяет token hash из `cloud_edge_nodes`, `node_device_id`, assigned restaurant и status `assigned`.
- Legacy `POST /api/v1/sync/edge-events` и `POST /api/v1/sync/edge-events/batch` остаются совместимыми путями приема Edge events.

Request:

```json
{
  "protocol_version": "sync_exchange.v1",
  "node_device_id": "edge-node-id",
  "restaurant_id": "restaurant-id",
  "edge_events": [
    {
      "client_item_id": "pos_sync_outbox.id",
      "payload": {
        "version": "1",
        "event_id": "event-id",
        "command_id": "command-id",
        "event_type": "OrderCreated",
        "aggregate_type": "Order",
        "aggregate_id": "order-id",
        "restaurant_id": "restaurant-id",
        "device_id": "edge-node-id",
        "node_device_id": "edge-node-id",
        "occurred_at": "2026-05-07T10:00:00Z",
        "payload": {
          "origin": "edge_device",
          "data": {}
        }
      }
    }
  ],
  "streams": [
    {
      "stream_name": "catalog",
      "last_cloud_version": 42,
      "checkpoint_token": "catalog:42"
    }
  ]
}
```

Response:

```json
{
  "protocol_version": "sync_exchange.v1",
  "status": "partial",
  "edge_acks": [
    {
      "client_item_id": "pos_sync_outbox.id",
      "status": "accepted",
      "ack": {
        "status": "accepted",
        "event_id": "event-id"
      }
    }
  ],
  "stream_results": [
    {
      "stream_name": "catalog",
      "status": "changed",
      "cloud_version": 43,
      "checkpoint_token": "catalog:43"
    }
  ],
  "cloud_packages": [
    {
      "stream_name": "catalog",
      "node_device_id": "edge-node-id",
      "restaurant_id": "restaurant-id",
      "sync_mode": "incremental",
      "cloud_version": 43,
      "checkpoint_token": "catalog:43",
      "payload_json": {
        "catalog_items": []
      }
    }
  ]
}
```

Правила:

- `edge_events` ограничен 100 items; один envelope ограничен 2 MiB; body endpoint ограничен 8 MiB.
- Поддерживаемые exchange streams: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`.
- ACK statuses: `accepted`, `rejected`, `retryable`; rejected/retryable items возвращают стабильный `error_code` и `message_key`.
- Если stream package отсутствует, `stream_results.status = "not_found"` и HTTP остается успешным.
- Unknown stream отклоняет весь request как `400 VALIDATION_FAILED` до приема Edge events.
- Edge revision больше Cloud revision отклоняет весь request как `409 SYNC_REVISION_AHEAD` до приема Edge events.
- Равная revision с другим checkpoint отклоняет весь request как `409 SYNC_CHECKPOINT_CONFLICT` до приема Edge events.
- POS Edge применяет `cloud_packages` через существующий `mastersync.Service`; stream data и `cloud_master_sync_state` фиксируются одной SQLite transaction boundary.
- Если Edge не смог применить Cloud package, outbox ACK не коммитится как `sent`; следующий exchange безопасно повторяет Edge events через Cloud idempotency.

## Cloud -> Edge Master Data Ingest

POS Edge endpoints:

```text
POST /api/v1/sync/master-data/snapshots
POST /api/v1/sync/master-data/{stream}
```

Supported POS Edge apply streams in `mastersync.Service`:

```text
restaurants
devices
staff
floor
catalog
menu
pricing_policy
```

Request body shape currently supported by POS Edge:

```json
{
  "restaurant_id": "restaurant-id",
  "stream": "catalog",
  "sync_mode": "incremental",
  "full_snapshot_reason": "",
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
  "folders": [],
  "folder_parameters": [],
  "tags": [],
  "item_tags": [],
  "menu_items": [],
  "modifier_groups": [],
  "modifier_options": [],
  "modifier_bindings": [],
  "menu_item_modifier_groups": [],
  "tax_profiles": [],
  "tax_rules": [],
  "service_charge_rules": [],
  "pricing_policies": []
}
```

Правила:

- `sync_mode` по умолчанию равен `incremental`.
- Поддерживаемые значения: `incremental` и `full_snapshot`.
- `full_snapshot` требует `full_snapshot_reason` со значением `terminal_restaurant_changed` или `node_role_changed`.
- Unsupported streams отклоняются.
- `catalog` применяет `catalog_items` с canonical `item_type`/`type` values `dish`, `good`, `semi_finished`, `service`, а также `folders`, `folder_parameters`, `tags`, `item_tags`, `modifier_groups`, `modifier_options` и `modifier_bindings`.
- `menu` применяет `menu_items` и effective `menu_item_modifier_groups` links после применения menu items; для старого explicit `stream: "catalog"` link-only payload остается accepted, если referenced menu item уже существует.
- Cloud publication package для POS Edge является typed ingest DTO, а не Cloud rich projection. `modifier_groups[]` содержит только поля, которые принимает POS Edge: `id`, `restaurant_id`, `name`, `required`, `min_count`, `max_count`, `active`.
- `modifier_options[]` содержит `id`, `restaurant_id`, `modifier_group_id`, `name`, `price_minor`, `active`.
- `modifier_bindings[]` содержит `id`, `restaurant_id`, `modifier_group_id`, `target_type`, `target_id`, `sort_order`, `active`.
- `menu_item_modifier_groups[]` является link-only массивом и содержит только `menu_item_id`, `modifier_group_id`, `sort_order`. Правила обязательности и count limits остаются в top-level `modifier_groups[]`.
- `menu_items[]` в ingest payload не содержит embedded rich `modifier_groups[]`; POS runtime read model собирает modifiers из top-level groups/options/links после применения snapshot.
- `restaurants` применяет Cloud-authored настройки ресторана и `active`; опубликованный active restaurant должен попадать в Edge read model как `active = true`.
- `pricing_policy` применяет Cloud-authored `tax_profiles`, `tax_rules`, `service_charge_rules` и automatic discount/surcharge `pricing_policies` в Edge read-model tables с sync metadata.
- Unsupported JSON fields отклоняются strict decode; неизвестные stream names не применяются.
- `recipes`, `inventory_reference` и `stop_lists` пока не являются поддерживаемыми POS Edge apply payloads.

Только основа:

- Cloud schema содержит recipe/inventory-adjacent publication foundation.
- Целевая inventory architecture требует Cloud -> Edge packages для read-only `recipe_versions`/`recipe_lines` и `stop_lists`.
- Эти streams нельзя документировать как реализованный POS Edge runtime ingest, пока `mastersync.Service` не применяет их payloads.

## Edge -> Cloud Operational Events

Реализовано сейчас:

- POS Edge пишет local operational events и outbox rows для cashier runtime commands.
- Детали sender/cloud receiver являются implementation-specific; документация не должна обещать Cloud reporting semantics для events, которых нет в подтвержденном Edge -> Cloud catalog.

Текущий POS Edge emitted catalog в domain boundary включает:

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
CancellationRecorded
RefundRecorded
CheckCreated
CheckReprinted
OrderClosed
AuthSessionStarted
AuthSessionRevoked
DeviceRegistered
```

Local-only POS Edge events that are not Edge -> Cloud operational contracts:

```text
StockDocumentPosted
```

Целевой Cloud-centric inventory Edge -> Cloud catalog, запланировано далее:

```text
CheckClosed
ItemServed
StockReceiptCaptured
InventoryCountCaptured
ProductionCompleted
RefundRecorded
CancellationRecorded
StopListUpdated
```

`StockDocumentPosted` не входит в целевой catalog: POS Edge не должен формировать stock documents/moves.

Legacy inbound-only event types, которые Cloud receiver продолжает валидировать для старых Edge payloads:

```text
PaymentRefunded
CheckRefunded
```

Cancellation/refund sync behavior:

- `CancellationRecorded` и `RefundRecorded` являются текущими Edge -> Cloud operational events для append-only financial operation ledger.
- Whole-check и partial `order_line`/quantity cancellation/refund UI, а также compatibility payment refund пишут те же текущие ledger events: `CancellationRecorded` для cancellation и `RefundRecorded` для refund. Переданный UI `command_id` остается idempotency key; `inventory_disposition` и operation `items[]` остаются payload data и не являются stock movement event.
- `PaymentRefunded` и `CheckRefunded` остаются валидируемыми legacy event types, но новый POS Edge refund flow пишет `RefundRecorded`.
- Cloud receiver валидирует эти event types, сохраняет raw envelope/journal rows и обновляет event-type stats.
- `GET /api/v1/sync/edge-events` реализовано сейчас как безопасный Cloud UI/API журнал receipt metadata: `restaurant_id`, `device_id`, `event_type`, aggregate metadata, timestamps и SHA-256 raw payload; raw payload в ответ не включается.
- `cloud_edge_event_receipts.event_type` принимает весь текущий catalog и legacy inbound-only types, чтобы runtime schema не расходилась с Go validation contract.
- Cloud shift finance foundation обновляет coarse refund counters from `RefundRecorded` (`checks_refunded_count`, `checks_refunded_total`) and legacy `PaymentRefunded`/`CheckRefunded` counters where such envelopes are received.
- Shift finance projection не является полной ledger projection для cancellation/refund; detailed reporting by operation item scope, inventory disposition, approval and original shift must read stored raw/journal payloads until a dedicated financial operation projection exists.
- `GET /api/v1/orders/closed` pagination/filtering является POS local read API behavior; оно не меняет Edge -> Cloud event payloads или Cloud receiver contracts.
- `GET /api/v1/storage/status` и `POST /api/v1/storage/retention/dry-run` являются локальными POS operational read/dry-run API. Они не создают sync envelopes; dry-run только сообщает non-sent `edge_to_cloud` outbox rows как blocking state для будущей destructive retention/archive policy.
- `GET /api/v1/sync/outbox`, `GET /api/v1/sync/local-events` и POS UI activity/sync drawer читают только bounded local windows; они не являются sync cleanup или archive contract.
- Manual `StockDocumentPosted` остается local-only в текущем POS Edge и не принимается/не проецируется Cloud receiver. В целевой inventory architecture этот event должен быть retired.

### Inventory Event Payloads Target

Запланировано далее. Все payloads передаются внутри стандартного sync envelope в `payload.data`.

`CheckClosed` является финальным batch-delta trigger:

```json
{
  "check_id": "018f0000-0000-7000-8000-000000000001",
  "order_id": "018f0000-0000-7000-8000-000000000002",
  "business_date_local": "2026-05-19",
  "closed_at": "2026-05-19T12:40:00Z",
  "items": [
    {
      "order_line_id": "018f0000-0000-7000-8000-000000000010",
      "catalog_item_id": "018f0000-0000-7000-8000-000000000020",
      "quantity": "2.000",
      "unit_code": "PC",
      "modifiers": [
        {
          "order_line_modifier_id": "018f0000-0000-7000-8000-000000000030",
          "modifier_option_id": "018f0000-0000-7000-8000-000000000031",
          "quantity": "1.000"
        }
      ]
    }
  ]
}
```

`ItemServed` фиксирует KDS-подачу и дедуплицируется с `CheckClosed`:

```json
{
  "served_event_id": "018f0000-0000-7000-8000-000000000101",
  "order_id": "018f0000-0000-7000-8000-000000000002",
  "order_line_id": "018f0000-0000-7000-8000-000000000010",
  "catalog_item_id": "018f0000-0000-7000-8000-000000000020",
  "quantity": "1.000",
  "unit_code": "PC",
  "served_at": "2026-05-19T12:25:00Z"
}
```

`RefundRecorded` и `CancellationRecorded` должны передавать disposition по каждой строке:

```json
{
  "operation_id": "018f0000-0000-7000-8000-000000000501",
  "operation_type": "refund",
  "check_id": "018f0000-0000-7000-8000-000000000001",
  "business_date_local": "2026-05-19",
  "recorded_at": "2026-05-19T14:00:00Z",
  "items": [
    {
      "order_line_id": "018f0000-0000-7000-8000-000000000010",
      "catalog_item_id": "018f0000-0000-7000-8000-000000000020",
      "quantity": "1.000",
      "inventory_disposition": "return_to_stock"
    },
    {
      "order_line_id": "018f0000-0000-7000-8000-000000000011",
      "catalog_item_id": "018f0000-0000-7000-8000-000000000021",
      "quantity": "1.000",
      "inventory_disposition": "write_off_waste"
    }
  ]
}
```

`StopListUpdated` синхронизируется в обе стороны:

```json
{
  "stop_list_id": "018f0000-0000-7000-8000-000000000601",
  "restaurant_id": "018f0000-0000-7000-8000-000000000004",
  "catalog_item_id": "018f0000-0000-7000-8000-000000000020",
  "available_quantity": "0.000",
  "active": true,
  "source": "edge",
  "reason": "ingredient_unavailable",
  "updated_at": "2026-05-19T12:05:00Z"
}
```

## Financial Payload Boundaries

Реализовано сейчас:

- Payloads `PaymentCaptured`, `CheckCreated`, `CancellationRecorded` и `RefundRecorded` включают backend-owned `business_date_local`, если он есть у source aggregate.
- Precheck/check reprint использует immutable snapshot payload, включая selected modifiers с name, quantity, unit price и total price.
- Payment ссылается на `precheck_id`, а не на legacy `check_id`.
- `RefundRecorded`/`CancellationRecorded` payload содержит immutable operation snapshot with embedded check snapshot, selected modifiers and item scopes; Cloud raw/journal receipt не должен отбрасывать modifier data из snapshot payload.

Не реализовано сейчас:

- inventory consumption events;
- KDS/stock input events;
- stop-list sync;
- Cloud Inventory Worker projection from these events;
- stock movement events for refund/cancellation disposition;
- Cloud receipt/projection contract for manual stock document events;
- PSP/fiscal event streams.

## Запланированные Границы

Запланировано до пилота только при отдельном принятии:

- Cloud-authored pricing/tax UI и полный publication workflow поверх generic `pricing_policy` package storage/apply;
- modifier print/reporting projections if pilot acceptance requires them;

После пилота:

- KDS/Production events as Edge business facts, not Edge stock moves;
- Cloud-generated inventory stock documents/moves;
- PSP/fiscal integration events;
- richer financial operation reporting projections and optional ClickHouse acceleration.

## Pricing policy stream completeness

Статус: реализовано сейчас.

Cloud -> Edge stream `pricing_policy` публикует JSON package с ключами `tax_profiles`, `tax_rules`, `service_charge_rules` и `pricing_policies`. Для текущего Cloud authoring surface `pricing_policies` содержит опубликованные скидки/надбавки, включая `id`, `restaurant_id`, `kind`, `scope`, `amount_kind`, `amount_minor`, `value_basis_points`, `application_index`, `manual`, `requires_permission` и `active`. Edge strict ingest сохраняет эти поля и отклоняет неизвестные поля по существующему strict decode contract.
