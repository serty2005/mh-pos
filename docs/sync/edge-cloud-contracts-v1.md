# Edge / Cloud Sync Contracts v1

Статус: актуализировано под текущий cashier runtime и целевой полный пилот.

## Direction Model

Реализовано сейчас:

- POS Edge owns cashier operational runtime data.
- Cloud owns master/reference/configuration authoring.
- POS Edge can continue cashier runtime while Cloud is unavailable.
- Directional ownership matrix is maintained in `docs/sync/directional-sync-ownership.md`.
- Реализовано сейчас: Cloud публикует recipe/stop-list reference data в streams `recipes` и `inventory_reference`, POS Edge применяет их через текущий master-data ingest pipeline для offline sale blocking.

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
        "event_id": "018f0000-0000-7000-8000-000000000001",
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
- POS Edge worker выполняет строгий periodic cycle без random jitter. Если pending Edge -> Cloud backlog достигает configured high-watermark, worker запускает следующую итерацию без ожидания poll interval.
- Cloud ограничивает число `cloud_packages` в одном `sync/exchange` response настройкой `CLOUD_SYNC_MAX_CLOUD_PACKAGES_PER_EXCHANGE`; большие Cloud -> Edge изменения передаются несколькими последовательными exchange-сессиями.
- Если POS Edge получил bounded Cloud -> Edge response с числом packages не меньше `POS_SYNC_SENDER_CLOUD_PACKAGE_BURST_THRESHOLD`, следующий Cloud pull выполняется без ожидания `POS_SYNC_SENDER_CLOUD_PULL_INTERVAL`.
- `event_id` для всех Edge POS/KDS business events должен быть UUIDv7.
- Поддерживаемые exchange streams: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`, `recipes`, `inventory_reference`.
- ACK statuses: `accepted`, `rejected`, `retryable`; rejected/retryable items возвращают стабильный `error_code` и `message_key`.
- Если stream package отсутствует, `stream_results.status = "not_found"` и HTTP остается успешным.
- Unknown stream отклоняет весь request как `400 VALIDATION_FAILED` до приема Edge events.
- Edge revision больше Cloud revision отклоняет весь request как `409 SYNC_REVISION_AHEAD` до приема Edge events.
- Равная revision с другим checkpoint отклоняет весь request как `409 SYNC_CHECKPOINT_CONFLICT` до приема Edge events.
- POS Edge применяет `cloud_packages` через существующий `mastersync.Service`; stream data и `cloud_master_sync_state` фиксируются одной SQLite transaction boundary.
- POS Edge применяет каждый Cloud package только после полного приема HTTP response и successful JSON decode. Ошибочный package извлекается из текущей порции, фиксируется как `cloud_master_sync_state.status = "failed"` с `last_error`, не ломает применение остальных packages и не блокирует Edge -> Cloud ACK.
- Если весь Cloud exchange request не принят транспортно или авторизационно, Edge outbox ACK не коммитится как `sent`; следующий exchange безопасно повторяет Edge events через Cloud idempotency.

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
recipes
inventory_reference
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
- `folder_parameters[]`, `tags[]` и `item_tags[]` содержат `restaurant_id`, потому что POS Edge сохраняет эти справочники с restaurant-scoped identity и отклоняет записи без явного restaurant context.
- `modifier_options[]` содержит `id`, `restaurant_id`, `modifier_group_id`, `name`, `price_minor`, `active`.
- `modifier_bindings[]` содержит `id`, `restaurant_id`, `modifier_group_id`, `target_type`, `target_id`, `sort_order`, `active`.
- `menu_item_modifier_groups[]` является link-only массивом и содержит только `menu_item_id`, `modifier_group_id`, `sort_order`. Правила обязательности и count limits остаются в top-level `modifier_groups[]`.
- `menu_items[]` в ingest payload не содержит embedded rich `modifier_groups[]`; POS runtime read model собирает modifiers из top-level groups/options/links после применения snapshot.
- `restaurants` применяет Cloud-authored настройки ресторана и `active`; опубликованный active restaurant должен попадать в Edge read model как `active = true`.
- `pricing_policy` применяет Cloud-authored `tax_profiles`, `tax_rules`, `service_charge_rules` и automatic discount/surcharge `pricing_policies` в Edge read-model tables с sync metadata.
- `recipes` применяет `recipe_versions` и `recipe_lines`.
- `inventory_reference` применяет active/inactive `stop_lists` overlay rows.
- Unsupported JSON fields отклоняются strict decode; неизвестные stream names не применяются.

Только основа:

- Cloud schema и publication workflow реально публикуют `recipes`/`inventory_reference` в `cloud_master_data_packages` как часть одного детерминированного publication snapshot.
- `scripts/seed-dev-system.py` создает recipe/stop-list examples и публикует их в Edge; runtime sale-blocking проверяется профильными POS backend tests.
- Smoke suite `pos_stop_list_sale_blocking` покрывает Cloud authoring -> publication -> Edge import -> блокировку продажи в POS runtime.

## Edge -> Cloud Operational Events

Реализовано сейчас:

- POS Edge пишет local operational events и outbox rows для cashier runtime commands.
- Детали sender/cloud receiver являются implementation-specific; документация не должна обещать Cloud reporting semantics для events, которых нет в подтвержденном Edge -> Cloud catalog.

Замороженный принцип для дальнейшей реализации:

```text
Edge Outbox
  -> Cloud API (PostgreSQL inbox_events)
  -> Async Batch Forwarder
  -> ClickHouse raw_business_events
```

- Cloud API принимает Edge outbox batch, сохраняет events в PostgreSQL `inbox_events` и отвечает `200 OK` без synchronous ClickHouse write.
- Async Batch Forwarder экспортирует `inbox_events` в ClickHouse batch size от 1 000 до 100 000 rows.
- После successful export event в PostgreSQL помечается `processed_for_olap = true`.
- `processed_for_olap = true` events старше 3 месяцев можно удалить из PostgreSQL.
- ClickHouse `raw_business_events` хранит все business events бессрочно.
- Synchronous dual-write в PostgreSQL и ClickHouse запрещен.

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
CheckClosed
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

Дополнительный Cloud-centric inventory Edge -> Cloud catalog, запланировано далее:

```text
KitchenTicketStatusChanged
ItemServed
StockReceiptCaptured
CatalogItemChangeSuggested
RecipeChangeSuggested
InventoryCountCaptured
ProductionCompleted
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
- Cloud receiver валидирует для текущих `CancellationRecorded`/`RefundRecorded` operation id, edge operation id, совпадение payload `restaurant_id`/`device_id` с envelope, check id, precheck id, original/current shift ids, amount, currency, business date, operation-level inventory disposition, reason и immutable snapshot; затем сохраняет raw envelope/journal rows, обновляет event-type stats и поддерживает detailed projection `cloud_projection_financial_operations`.
- Ошибочные items в batch/exchange не останавливают прием остальных items: Cloud возвращает item-level `rejected`/`retryable` ACK и сохраняет проблемный raw item в `cloud_sync_problem_events` для последующего анализа.
- `GET /api/v1/sync/edge-events` реализовано сейчас как безопасный Cloud UI/API журнал receipt metadata: `restaurant_id`, `device_id`, `event_type`, aggregate metadata, timestamps и SHA-256 raw payload; raw payload в ответ не включается.
- `cloud_edge_event_receipts.event_type` принимает весь текущий catalog и legacy inbound-only types, чтобы runtime schema не расходилась с Go validation contract.
- Cloud shift finance foundation обновляет coarse refund counters from `RefundRecorded` (`checks_refunded_count`, `checks_refunded_total`) and legacy `PaymentRefunded`/`CheckRefunded` counters where such envelopes are received.
- Shift finance projection не является полной ledger projection для cancellation/refund; detailed reporting foundation теперь читает `cloud_projection_financial_operations` для current `CancellationRecorded`/`RefundRecorded`. Public Cloud reporting HTTP API/UI остается запланировано далее.
- `GET /api/v1/orders/closed` pagination/filtering является POS local read API behavior; оно не меняет Edge -> Cloud event payloads или Cloud receiver contracts.
- `GET /api/v1/storage/status`, `POST /api/v1/storage/retention/dry-run`, `POST /api/v1/storage/archive/export-plan`, `POST /api/v1/storage/archive/export`, `POST /api/v1/storage/archive/verify`, `POST /api/v1/storage/archive/read-plan`, `POST /api/v1/storage/archive/lookup`, `POST /api/v1/storage/archive/apply-plan` и `POST /api/v1/storage/archive/apply-readiness` являются локальными POS operational lifecycle API. Они используют exclusive cutoff rule `checks.business_date_local < cutoff_business_date_local` и не создают sync envelopes. Dry-run, manifest-only export-plan, export-only archive, verify/read-plan/lookup и apply-readiness не мутируют runtime rows. Apply-readiness возвращает `result_mode = apply_readiness_only` и `ready_for_destructive_apply = true` только при verified archive, clean scoped `edge_to_cloud` outbox и отсутствии open operational boundaries. Apply-plan при тех же verified/safety условиях выполняет локальный destructive apply + `VACUUM` и возвращает `result_mode = destructive_apply`, `runtime_rows_deleted = true`; иначе возвращает `apply_blocked`.
- `GET /api/v1/sync/outbox`, `GET /api/v1/sync/local-events` и POS UI activity/sync drawer читают только bounded local windows; они не являются sync cleanup или archive contract.
- Manual `StockDocumentPosted` исторически был local-only pre-pilot Edge event, не принимался и не проецировался Cloud receiver; при Cloud-centric inventory cutover этот Edge runtime path удален.
- `GET /api/v1/inventory/stock-ledger` является Cloud bounded read-only endpoint для проверки результата Cloud Inventory Worker по accepted inventory events; endpoint не раскрывает raw sync payload и не является ClickHouse/OLAP contract.

### Inventory Event Payloads Target

Реализовано сейчас: POS Edge генерирует `CheckClosed` при создании final check после полной оплаты; payload строится из immutable `check.Snapshot` и передается внутри стандартного sync envelope в `payload.data`. Cloud receiver принимает `CheckClosed` и `ItemServed`, а Cloud Inventory Worker создает `SALE` stock documents/ledger rows идемпотентно и дедуплицирует уже обработанный `ItemServed` при последующем `CheckClosed`.

Запланировано до полного пилота для генерации на POS Edge/KDS и остальных inventory/proposal payloads, необходимых полному Cloud Inventory Engine и ClickHouse OLAP: `KitchenTicketStatusChanged`, `ItemServed`, `StockReceiptCaptured`, `CatalogItemChangeSuggested`, `RecipeChangeSuggested`, `InventoryCountCaptured`, `ProductionCompleted`, `StopListUpdated`, `RefundRecorded`, `CancellationRecorded`. Все payloads передаются внутри стандартного sync envelope в `payload.data`.

`CheckClosed` является финальным batch-delta trigger:

```json
{
  "check_id": "018f0000-0000-7000-8000-000000000001",
  "order_id": "018f0000-0000-7000-8000-000000000002",
  "precheck_id": "018f0000-0000-7000-8000-000000000003",
  "restaurant_id": "018f0000-0000-7000-8000-000000000004",
  "business_date_local": "2026-05-19",
  "closed_at": "2026-05-19T12:40:00Z",
  "items": [
    {
      "order_line_id": "018f0000-0000-7000-8000-000000000010",
      "catalog_item_id": "018f0000-0000-7000-8000-000000000020",
      "quantity": "2.000",
      "unit_code": "PC",
      "required_for_inventory": true,
      "modifiers": [
        {
          "modifier_group_id": "018f0000-0000-7000-8000-000000000030",
          "modifier_option_id": "018f0000-0000-7000-8000-000000000031",
          "name": "Extra sauce",
          "quantity": "1.000",
          "unit_code": "PC"
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

Реализовано сейчас в Cloud Inventory Worker:

- `ItemServed` пишет `stock_ledger` movement для конкретного `order_line_id`;
- replay того же `ItemServed` не создает второй stock document;
- `CheckClosed` после уже обработанного `ItemServed` списывает только положительную unserved delta;
- replay того же `CheckClosed` не создает второй stock document.

Запланировано далее: POS Edge/KDS endpoints и UI для генерации `KitchenTicketStatusChanged`/`ItemServed`, ClickHouse export, balances, recipe expansion и полный costing engine.

`KitchenTicketStatusChanged` фиксирует advanced KDS lifecycle без прямой складской проводки:

```json
{
  "status_event_id": "018f0000-0000-7000-8000-000000000111",
  "restaurant_id": "018f0000-0000-7000-8000-000000000004",
  "order_id": "018f0000-0000-7000-8000-000000000002",
  "order_line_id": "018f0000-0000-7000-8000-000000000010",
  "station_id": "kitchen-hot",
  "from_status": "accepted",
  "to_status": "in_progress",
  "changed_by_employee_id": "018f0000-0000-7000-8000-000000000112",
  "changed_at": "2026-05-19T12:12:00Z",
  "reason": null
}
```

Допустимые KDS statuses полного пилота: `new`, `accepted`, `in_progress`, `hold`, `ready`, `served`, `recall`, `cancelled`. Переход `served` должен сопровождаться `ItemServed`.

`StockReceiptCaptured`, `CatalogItemChangeSuggested` и `RecipeChangeSuggested` создают Cloud worker review/apply flow:

```json
{
  "event_type": "RecipeChangeSuggested",
  "suggestion_id": "018f0000-0000-7000-8000-000000000311",
  "restaurant_id": "018f0000-0000-7000-8000-000000000004",
  "recipe_version_id": "018f0000-0000-7000-8000-000000000312",
  "prep_time_delta_minutes": 5,
  "changes": [
    {
      "action": "replace_ingredient",
      "from_catalog_item_id": "018f0000-0000-7000-8000-000000000314",
      "to_catalog_item_id": "018f0000-0000-7000-8000-000000000315",
      "quantity": "0.120",
      "unit_code": "KG",
      "loss_percent": "3.00"
    }
  ]
}
```

Cloud worker не применяет `CatalogItemChangeSuggested`/`RecipeChangeSuggested` к master data без explicit policy или manager approve. `RecipeChangeSuggested.prep_time_delta_minutes` валидируется по `recipe_suggestion_max_time_delta_minutes`.

`RefundRecorded` и `CancellationRecorded` сейчас передают operation-level disposition, а не disposition по каждой строке:

```json
{
  "operation_id": "018f0000-0000-7000-8000-000000000501",
  "operation_type": "refund",
  "check_id": "018f0000-0000-7000-8000-000000000001",
  "inventory_disposition": "return_to_stock",
  "business_date_local": "2026-05-19",
  "recorded_at": "2026-05-19T14:00:00Z",
  "items": [
    {
      "order_line_id": "018f0000-0000-7000-8000-000000000010",
      "catalog_item_id": "018f0000-0000-7000-8000-000000000020",
      "quantity": "1.000"
    },
    {
      "order_line_id": "018f0000-0000-7000-8000-000000000011",
      "catalog_item_id": "018f0000-0000-7000-8000-000000000021",
      "quantity": "1.000"
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
  "conflict_policy": "most_restrictive",
  "reason": "ingredient_unavailable",
  "updated_at": "2026-05-19T12:05:00Z"
}
```

## Financial Payload Boundaries

Реализовано сейчас:

- Payloads `PaymentCaptured`, `CheckCreated`, `CancellationRecorded` и `RefundRecorded` включают backend-owned `business_date_local`, если он есть у source aggregate.
- Precheck/check reprint использует immutable snapshot payload, включая selected modifiers с name, quantity, unit price и total price.
- Payment ссылается на `precheck_id`, а не на legacy `check_id`.
- `PaymentCaptured`, `CheckCreated` и `CheckClosed` используют в envelope текущую кассовую смену оплаты; исходная личная смена заказа остается в order payload и не переписывается.
- `RefundRecorded`/`CancellationRecorded` payload содержит immutable operation snapshot with embedded check snapshot, selected modifiers and item scopes; Cloud raw/journal receipt не должен отбрасывать modifier data из snapshot payload. Текущий validation contract требует operation-level `inventory_disposition`; `items[].inventory_disposition` не является реализованным полем.

Не реализовано сейчас:

- Edge-origin stop-list edit sync/conflict policy;
- KDS runtime для генерации `KitchenTicketStatusChanged` / `ItemServed` / `ProductionCompleted`;
- proposal events `CatalogItemChangeSuggested` и `RecipeChangeSuggested`;
- recipe expansion, modifier linked catalog item consumption и retro costing DAG;
- ClickHouse forwarder/projection export and bounded OLAP API;
- PSP/fiscal event streams.

Запланировано до полного пилота:

- advanced KDS генерирует `KitchenTicketStatusChanged`, `ItemServed` и cooking events;
- chef receipt/catalog/recipe proposal flows генерируют `StockReceiptCaptured`, `CatalogItemChangeSuggested` и `RecipeChangeSuggested`;
- stop-list changes синхронизируются через Cloud -> Edge packages и, если включен Edge manager input, через `StopListUpdated`;
- Cloud Inventory Worker расширяется до полного receipts, counts, production, refund/cancellation dispositions, balances and costing engine;
- ClickHouse pipeline экспортирует `raw_business_events` and `olap_stock_moves`, а Cloud OLAP API читает bounded aggregates.

## Запланированные Границы

Запланировано до полного пилота:

- Cloud-authored pricing/tax UI и полный publication workflow поверх generic `pricing_policy` package storage/apply;
- Cloud authoring workflow для `recipes`/`inventory_reference` package generation;
- `CheckClosed`/`KitchenTicketStatusChanged`/`ItemServed` pilot inventory and KDS facts;
- `CatalogItemChangeSuggested`/`RecipeChangeSuggested` review queues;
- full inventory event catalog and Cloud Inventory Engine;
- ClickHouse `raw_business_events`, `olap_stock_moves`, retry/backfill/export state and OLAP API;
- stop-list sale blocking smoke через offline Edge.

После полного пилота:

- hardware bump-bar/printer integrations and rich BI dashboards beyond bounded OLAP/KDS metrics;
- PSP/fiscal integration events;
- public Cloud reporting UI beyond pilot OLAP API.

## Pricing policy stream completeness

Статус: реализовано сейчас.

Cloud -> Edge stream `pricing_policy` публикует JSON package с ключами `tax_profiles`, `tax_rules`, `service_charge_rules` и `pricing_policies`. Для текущего Cloud authoring surface `pricing_policies` содержит опубликованные скидки/надбавки, включая `id`, `restaurant_id`, `kind`, `scope`, `amount_kind`, `amount_minor`, `value_basis_points`, `application_index`, `manual`, `requires_permission` и `active`. Edge strict ingest сохраняет эти поля и отклоняет неизвестные поля по существующему strict decode contract.
