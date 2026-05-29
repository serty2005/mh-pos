# Kitchen Processes Implementation Spec

Статус: профильная спецификация кухонного контура. POS Edge order queue, ticket lifecycle, recipe read, local kitchen proposals, Edge-side kitchen stock input routes, Cloud proposal review/apply, Cloud-side `StockWriteOffCaptured` processing и профильный kitchen/process smoke помечены как реализовано сейчас.

Дата фиксации: 2026-05-27.

## Назначение

Документ задает целевой контракт реализации кухонных процессов для RMS-POS:

- backend-authoritative статусы кухонных заказов и отдельных блюд;
- факт `served`, фиксируемый работником кухни;
- просмотр техкарт и отправка предложений изменений;
- приемка товара на склад как Edge business event с Cloud-owned документами учета;
- предложения новых товаров, блюд и техкарт от кухни;
- кухонные события складского движения: ревизия, списание с причиной, приготовление заготовки;
- отдельный kitchen mode в `pos-ui-g`.

Источник истины для фактического runtime остается код и тесты. Эта спецификация описывает, что нужно реализовать следующим шагом, и не помечает будущие API как уже работающие.

## Текущая Готовность К Реализации

Реализовано сейчас:

- POS Edge уже имеет таблицы `kitchen_tickets` и `kitchen_ticket_events`.
- POS Edge уже создает kitchen ticket на активную order line и поддерживает lifecycle `new -> accepted -> in_progress -> ready -> served` с ветками `hold`, `recall`, `cancelled`, включая повторный цикл `served -> recall -> start -> ready -> serve`.
- POS Edge уже предоставляет `GET /api/v1/kitchen/order-queue` с grouped order read model и вычисляемым `kitchen_order_status`.
- `POST /api/v1/kitchen/tickets/{id}/{action}` уже пишет `KitchenTicketStatusChanged`; action `serve` дополнительно пишет `ItemServed`.
- Replay того же kitchen `command_id` идемпотентно возвращает текущее состояние ticket без второго `kitchen_ticket_events`/outbox event.
- Повторная подача после `recall` с новым `command_id` пишет новый `ItemServed` с `ticket_id`, `serve_sequence` и optional `supersedes_served_event_id`.
- Cloud sync contract уже принимает `KitchenTicketStatusChanged`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted`, `StopListUpdated`, `RefundRecorded`, `CancellationRecorded`.
- Cloud Inventory Worker уже создает `stock_documents` и `stock_ledger` для `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted`, stock-effect `RefundRecorded`/`CancellationRecorded`; `StopListUpdated` пока не создает stock movement.
- Cloud -> Edge ingest уже поддерживает streams `catalog`, `menu`, `recipes`, `inventory_reference`; POS Edge SQLite уже содержит `catalog_items`, `recipe_versions`, `recipe_lines`, `stop_lists`.
- POS Edge уже имеет `GET /api/v1/catalog/items`, который можно использовать для полного каталога, а не только текущего меню.
- POS Edge уже применяет `warehouse_reference` из `inventory_reference` и использует default warehouse для kitchen stock command validation.
- POS Edge уже предоставляет `POST /api/v1/kitchen/stock-receipts`, `POST /api/v1/kitchen/inventory-counts`, `POST /api/v1/kitchen/stock-write-offs`, `POST /api/v1/kitchen/productions`.
- Эти kitchen stock routes пишут только immutable local event/outbox envelopes `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted`; POS Edge не создает `stock_documents`, `stock_moves`, `stock_ledger`, `stock_balances`.
- Replay того же stock `command_id` для того же event type возвращает сохраненный `id`, `warehouse_id`, `event_type` и `replayed = true` без второго `local_event_log`/outbox event.
- POS Edge уже предоставляет `GET /api/v1/kitchen/catalog/items/{catalog_item_id}/recipe`, `POST /api/v1/kitchen/catalog-suggestions`, `POST /api/v1/kitchen/recipe-suggestions`, `GET /api/v1/kitchen/proposals`.
- Recipe read возвращает active recipe version, строки техкарты с названиями ингредиентов из `catalog_items` и локальные proposals по этой техкарте.
- Catalog/recipe suggestion routes сохраняют `kitchen_proposals` со status `pending_sync`, пишут `CatalogItemChangeSuggested`/`RecipeChangeSuggested` в `local_event_log` и `pos_sync_outbox`, поддерживают replay по `command_id` и не мутируют `catalog_items`, `recipe_versions`, `recipe_lines`.
- `RecipeChangeSuggested.prep_time_delta_minutes` валидируется POS Edge лимитом `POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES` с default `120`.
- `pos-ui-g` уже имеет отдельный terminal mode `kds` с backend-backed kitchen runtime: нижний quick access `Заказы`/`Склад`/`Кухня`, верхние вкладки внутри разделов, queue/ready order tiles, stock forms и kitchen proposals/recipe screens.

Не реализовано сейчас:

- Компенсирующий пересчет уже обработанного `ItemServed`, если recall/serve-again пришел после создания первого stock document.
- Edge-side stop-list edit/conflict policy.

## Архитектурные Решения

Рекомендуемый подход: расширять текущую Cloud-centric inventory architecture через Edge business events.

POS Edge остается локальным runtime для ввода кухонных фактов и offline KDS. Cloud остается единственным владельцем складских документов, ledger, себестоимости, справочника каталога, техкарт и manager review/apply решений.

POS Edge не создает `StockDocument`, `StockMove`, `StockLedger` и не считает себестоимость. POS Edge пишет immutable outbox events с оператором, устройством, сменой, business date и payload snapshot. Cloud принимает события идемпотентно, пишет их в PostgreSQL inbox/journal для ACK, checkpoint и retry, затем async forwarder переносит кухонный event trail в ClickHouse `raw_business_events`. Складской анализатор полного контура читает ordered kitchen/inventory stream из ClickHouse, строит эффективное последнее состояние по строке/тикету и только после этого создает Cloud-owned складские документы и ledger в PostgreSQL.

## Scope Текущей Реализации

Реализовано сейчас:

- POS Edge backend: kitchen stock command/input layer, warehouse validation, recipe read, local catalog/recipe proposal status read model, RBAC and outbox events for receipt, count, write-off, production, catalog suggestions and recipe suggestions.

Запланировано далее:

- Cloud backend: новые sync event contracts, receiver validation, proposal review tables/routes, inventory worker для write-off и warehouse sequence.
- POS UI `pos-ui-g`: полный kitchen mode с тремя нижними разделами и вкладками внутри рабочих экранов реализован; дальнейшие шаги касаются UX-полировки, расширенных validation hints и e2e покрытия.
- Cloud UI: manager review для catalog/recipe suggestions, approve/reject и публикация измененных справочников.
- Документация и тесты по backend, sync, UI и smoke.

Вне текущего объема:

- Hardware bump-bar и kitchen printer orchestration.
- Procurement planning, supplier contracts, ERP/accounting integrations.
- POS-side расчет остатков или себестоимости.
- Автоматическое применение предложений кухни без manager approve, кроме явно включенной Cloud policy.
- Мобильная версия kitchen mode; кухонный экран проектируется для desktop/tablet.

## Domain Model

### Kitchen Order Queue

Кухонный заказ является read model поверх order + kitchen tickets. Он не заменяет `orders.status`, потому что `orders.status` уже занят финансовым lifecycle `open`, `locked`, `closed`, `cancelled`.

Добавляется вычисляемый `kitchen_order_status`:

- `queued` - все активные tickets в `new`;
- `accepted` - все активные tickets в `accepted` или часть уже начата;
- `in_progress` - хотя бы один active ticket в `in_progress`, нет готовых/served;
- `partially_ready` - часть active tickets `ready`, часть еще готовится;
- `ready` - все active tickets `ready`;
- `partially_served` - часть active tickets `served`, часть еще не served;
- `served` - все active tickets `served`;
- `cancelled` - все tickets отменены;
- `mixed` - состояние не попало в простую группу, например сочетание `hold`, `recall`, `cancelled` и active work.

Этот статус возвращается API очереди и UI, но не хранится в `orders.status`.

### Kitchen Ticket

Существующий `kitchen.Ticket` остается основной моделью блюда/строки. Статусы:

- `new`;
- `accepted`;
- `in_progress`;
- `hold`;
- `ready`;
- `served`;
- `recall`;
- `cancelled`.

Факт `served` фиксируется только через backend action `serve` и записывает:

- строку в `kitchen_ticket_events`;
- outbox event `KitchenTicketStatusChanged`;
- outbox event `ItemServed`.

Replay того же `command_id` остается идемпотентным и не создает второй `ItemServed`. Новый рабочий цикл после возврата блюда в готовку является отдельным бизнес-фактом: повар может выполнить `served -> recall -> start -> ready -> serve`, и последующий `serve` с новым `command_id` должен создать новый `ItemServed` с новым `served_event_id`.

Для повторной подачи `ItemServed` должен содержать `serve_sequence` по `order_line_id`/`ticket_id` и optional `supersedes_served_event_id`. Cloud-анализатор считает эффективным только последнее неотозванное событие по строке и тикету. Складское списание не должно удваиваться из-за первого `ItemServed`, если после него был `recall` и новая подача: итоговая складская проекция должна соответствовать последнему эффективному `ItemServed`, а прежний факт остается в ClickHouse как audit/timing event.

### Recipe Read Model

POS Edge хранит read-only Cloud recipes:

- `recipe_versions`;
- `recipe_lines`;
- связанный `catalog_items`.

Для кухни нужно добавить backend read endpoint, который возвращает техкарту в удобном виде:

- блюдо или заготовка;
- active recipe version;
- yield quantity/unit;
- ингредиенты с `catalog_item_id`, названием из полного каталога, quantity, unit, loss percent;
- sync metadata `cloud_version`, `cloud_updated_at`;
- локальные pending предложения по этой техкарте.

Edge не применяет recipe proposal локально. До Cloud approve UI показывает proposal как pending/change requested/rejected/approved.

## POS Edge Backend API

Все mutating routes требуют `command_id`, operator session, `node_device_id`, `client_device_id`, `session_id`. Kitchen status actions сначала проверяют replay в `kitchen_ticket_events`: повтор того же `command_id` для того же ticket/action возвращает успешный идемпотентный ответ без новых событий; reuse `command_id` для другой команды остается safe conflict через общий outbox/idempotency guard. Kitchen stock commands проверяют replay через `pos_sync_outbox`: повтор того же `command_id` и event type возвращает сохраненный результат без повторной записи local/outbox events.

### Kitchen Queue

Реализовано сейчас:

```text
GET /api/v1/kitchen/order-queue?status=&station=&limit=&offset=
GET /api/v1/kitchen/tickets?status=&station=&limit=&offset=
POST /api/v1/kitchen/tickets/{ticket_id}/accept
POST /api/v1/kitchen/tickets/{ticket_id}/start
POST /api/v1/kitchen/tickets/{ticket_id}/hold
POST /api/v1/kitchen/tickets/{ticket_id}/ready
POST /api/v1/kitchen/tickets/{ticket_id}/serve
POST /api/v1/kitchen/tickets/{ticket_id}/recall
POST /api/v1/kitchen/tickets/{ticket_id}/cancel
```

Целевые переходы:

- `new -> accepted|cancelled`;
- `accepted -> in_progress|hold|cancelled`;
- `in_progress -> hold|ready|cancelled`;
- `hold -> in_progress|cancelled`;
- `ready -> served|recall`;
- `served -> recall`;
- `recall -> in_progress|cancelled`.

`recall` из `served` означает, что уже поданное блюдо вернули в работу. Он не удаляет предыдущий `ItemServed`, а делает его неэффективным для последующей складской аналитики до новой подачи.

`GET /kitchen/order-queue` возвращает grouped read model:

```json
{
  "orders": [
    {
      "order_id": "order-1",
      "edge_order_id": "edge-order-1",
      "table_name": "A1",
      "shift_id": "shift-1",
      "kitchen_order_status": "in_progress",
      "created_at": "2026-05-27T09:00:00Z",
      "last_status_changed_at": "2026-05-27T09:05:00Z",
      "elapsed_seconds": 420,
      "tickets": []
    }
  ],
  "limit": 50,
  "offset": 0
}
```

Правила:

- default `limit = 50`, max `limit = 100`;
- сортировка: oldest active order first по самому раннему active ticket `created_at`, затем `order_id`;
- `served` и `cancelled` по умолчанию скрываются из active queue, но доступны фильтром;
- timestamps рассчитываются backend-side, UI не делает authoritative timing.
- `status` у `order-queue` фильтрует вычисляемый `kitchen_order_status`, а `station` фильтрует `station_routing_key`.
- `GET /api/v1/kitchen/order-queue` требует `pos.kitchen.view`; ticket status actions требуют `pos.kitchen.status.change`.

### Full Catalog For Kitchen

Использовать существующий:

```text
GET /api/v1/catalog/items
```

Доработки:

- route должен возвращать весь локально синхронизированный каталог, включая `dish`, `good`, `semi_finished`, `service`, а не только menu sellable items;
- kitchen UI фильтрует типы по сценарию, но backend остается источником данных;
- Cloud publication должна включать все active catalog items ресторана, даже если item не входит в текущее меню.

### Recipe Read And Suggestion

Реализовано сейчас:

```text
GET /api/v1/kitchen/catalog/items/{catalog_item_id}/recipe
POST /api/v1/kitchen/recipe-suggestions
GET /api/v1/kitchen/proposals?kind=&status=&limit=&offset=
```

`POST /kitchen/recipe-suggestions` пишет outbox event `RecipeChangeSuggested`.

`GET /kitchen/catalog/items/{catalog_item_id}/recipe` возвращает backend read model, который `pos-ui-g` читает без compatibility-подмены:

```json
{
  "catalog_item": {
    "id": "dish-1",
    "type": "dish",
    "name": "Суп дня",
    "base_unit": "portion",
    "active": true
  },
  "recipe_version": {
    "id": "recipe-1",
    "dish_catalog_item_id": "dish-1",
    "yield_qty": 1,
    "yield_unit": "portion",
    "active": true
  },
  "ingredients": [
    {
      "line_id": "line-1",
      "catalog_item_id": "good-1",
      "catalog_item_name": "Морковь",
      "quantity": "120",
      "unit_code": "G",
      "loss_percent": "3"
    }
  ],
  "proposals": []
}
```

Payload:

```json
{
  "command_id": "cmd-1",
  "suggestion_id": "optional-client-id",
  "proposal_group_id": "optional-group-id",
  "recipe_version_id": "recipe-1",
  "owner_catalog_item_id": "dish-1",
  "owner_catalog_suggestion_id": "",
  "action": "update_recipe",
  "prep_time_delta_minutes": 5,
  "changes": [
    {
      "line_id": "line-1",
      "action": "replace_ingredient",
      "from_catalog_item_id": "old-good-1",
      "to_catalog_item_id": "new-good-1",
      "quantity": "0.120",
      "unit_code": "KG",
      "loss_percent": "3.00"
    }
  ],
  "reason": "supplier_substitution"
}
```

Допустимые `action`:

- `create_recipe`;
- `update_recipe`;
- `replace_ingredient`;
- `add_ingredient`;
- `remove_ingredient`;
- `change_quantity`;
- `change_loss_percent`;
- `change_prep_time`.

Правила:

- `prep_time_delta_minutes` проверяется POS Edge конфигом `POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES`;
- для `create_recipe` допускается `owner_catalog_suggestion_id`, если блюдо еще не принято в Cloud catalog;
- Edge сохраняет локальный proposal status `pending_sync`, затем обновляет его по Cloud -> Edge proposal feedback stream;
- Edge не меняет `recipe_versions` и `recipe_lines` до следующей Cloud publication после manager approve.

### Catalog Suggestions

Реализовано сейчас:

```text
POST /api/v1/kitchen/catalog-suggestions
```

Payload:

```json
{
  "command_id": "cmd-1",
  "proposal_group_id": "group-1",
  "action": "create",
  "catalog_item_id": "",
  "kind": "good",
  "name": "Базилик свежий",
  "sku": "",
  "base_unit": "G",
  "kitchen_type": "cold",
  "accounting_category": "ingredient",
  "reason": "supplier_delivery"
}
```

Допустимые `kind`: `dish`, `good`, `semi_finished`, `service`.

Для сборки нового блюда работник кухни отправляет два связанных события:

- `CatalogItemChangeSuggested` с `kind = dish`;
- `RecipeChangeSuggested` с тем же `proposal_group_id` и `action = create_recipe`.

Cloud создает единый review item "новое блюдо с техкартой". Manager может принять catalog item, затем создать menu item отдельным Cloud flow.

### Stock Receipt

Реализовано сейчас:

```text
POST /api/v1/kitchen/stock-receipts
```

Payload:

```json
{
  "command_id": "cmd-1",
  "receipt_id": "receipt-1",
  "warehouse_id": "warehouse-main",
  "supplier_counterparty_id": "supplier-1",
  "supplier_name_snapshot": "ООО Поставщик",
  "document_number": "UPD-100",
  "document_date": "2026-05-27",
  "received_at": "2026-05-27T08:20:00Z",
  "business_date_local": "2026-05-27",
  "currency": "RUB",
  "items": [
    {
      "line_id": "line-1",
      "catalog_item_id": "good-1",
      "catalog_suggestion_id": "",
      "name_snapshot": "Картофель",
      "quantity": "10.000",
      "unit_code": "KG",
      "unit_cost_minor": 6500,
      "line_total_minor": 65000,
      "currency": "RUB"
    }
  ]
}
```

Правила:

- Каждая строка в текущем Edge runtime содержит существующий `catalog_item_id`, потому что текущий Cloud sync contract для `StockReceiptCaptured` еще валидирует `catalog_item_id` как обязательный.
- Связка receipt line с `catalog_suggestion_id` остается запланировано далее вместе с `CatalogItemChangeSuggested` receiver/review flow.
- POS Edge не создает учетный документ. Он пишет `StockReceiptCaptured`.
- Cloud создает `stock_documents.document_type = PURCHASE`.
- Строки с нерешенным `catalog_suggestion_id` попадают в Cloud status `pending_catalog_resolution`; ledger для них создается только после manager approve и связывания suggestion с catalog item.
- `line_total_minor` является обязательным, потому что пользователь явно передает сумму построчно.

### Inventory Count

Реализовано сейчас:

```text
POST /api/v1/kitchen/inventory-counts
```

Payload:

```json
{
  "command_id": "cmd-1",
  "count_id": "count-1",
  "warehouse_id": "warehouse-main",
  "counted_at": "2026-05-27T21:00:00Z",
  "business_date_local": "2026-05-27",
  "items": [
    {
      "line_id": "line-1",
      "catalog_item_id": "good-1",
      "counted_quantity": "3.250",
      "unit_code": "KG"
    }
  ]
}
```

Cloud создает `stock_documents.document_type = INVENTORY_COUNT`. Для полного учета разницы Cloud worker должен считать delta относительно Cloud balance, а не принимать Edge count как готовое движение.

### Stock Write-Off

POS Edge реализовано сейчас пишет `StockWriteOffCaptured` как local event/outbox envelope. Cloud Go sync contract и Inventory Worker для этого event реализованы сейчас.

Реализовано сейчас:

```text
POST /api/v1/kitchen/stock-write-offs
```

Payload:

```json
{
  "command_id": "cmd-1",
  "write_off_id": "write-off-1",
  "warehouse_id": "warehouse-main",
  "written_off_at": "2026-05-27T16:30:00Z",
  "business_date_local": "2026-05-27",
  "reason_code": "spoilage",
  "reason": "истек срок годности",
  "items": [
    {
      "line_id": "line-1",
      "catalog_item_id": "good-1",
      "quantity": "1.500",
      "unit_code": "KG"
    }
  ]
}
```

Cloud worker создает `stock_documents.document_type = WASTE`, `stock_ledger.movement_type = OUT`.

### Production Completed

Реализовано сейчас:

```text
POST /api/v1/kitchen/productions
```

Payload:

```json
{
  "command_id": "cmd-1",
  "production_id": "production-1",
  "warehouse_id": "warehouse-main",
  "semi_finished_catalog_item_id": "semi-1",
  "quantity": "5.000",
  "unit_code": "KG",
  "completed_at": "2026-05-27T10:15:00Z",
  "business_date_local": "2026-05-27"
}
```

Cloud создает `stock_documents.document_type = PRODUCTION`: приход заготовки и расход сырья по опубликованной техкарте заготовки.

## POS Edge SQLite

Реализовано сейчас:

- `warehouse_reference` - Cloud -> Edge read model складов: `id`, `restaurant_id`, `name`, `kind`, `is_default`, `active`, sync metadata.

Реализовано сейчас:

- `kitchen_proposals` - локальные предложения catalog/recipe с status `draft`, `pending_sync`, `synced`, `approved`, `rejected`, `changes_requested`, `failed`.

Запланировано далее:

- `kitchen_stock_receipt_drafts` и `kitchen_stock_receipt_lines` - локальные draft только до отправки; после успешной отправки immutable snapshot остается в outbox/local event.
- `kitchen_inventory_count_drafts` и lines.
- `kitchen_write_off_drafts` и lines.
- `kitchen_production_drafts`.

Не добавлять:

- `stock_documents`;
- `stock_moves`;
- `stock_ledger`;
- `stock_balances`;
- `item_costs`.

## Sync Event Contracts

Реализовано сейчас на POS Edge generated payloads:

```text
CatalogItemChangeSuggested
RecipeChangeSuggested
StockWriteOffCaptured
```

Реализовано сейчас для Cloud receiver/review/apply runtime: Cloud-side typed validation и review queues для `CatalogItemChangeSuggested`/`RecipeChangeSuggested`, а также Cloud-side worker processing для `StockWriteOffCaptured`.

Расширить существующие:

- `StockReceiptCaptured`: `warehouse_id`, `supplier_counterparty_id`, `supplier_name_snapshot`, `document_number`, `document_date`, `line_id`, `catalog_suggestion_id`, `name_snapshot`, `line_total_minor`.
- `InventoryCountCaptured`: `warehouse_id`, `line_id`.
- `ProductionCompleted`: `warehouse_id`.
- `StopListUpdated`: `warehouse_id` optional, `conflict_policy`.
- `KitchenTicketStatusChanged`: `status_event_id`, `restaurant_id`, `changed_by_employee_id`, optional `reason`.
- `ItemServed`: `ticket_id`, `serve_sequence`, optional `supersedes_served_event_id`.

Inventory-relevant event types:

```text
CheckClosed
ItemServed
StockReceiptCaptured
InventoryCountCaptured
StockWriteOffCaptured
ProductionCompleted
RefundRecorded
CancellationRecorded
StopListUpdated
```

Proposal event types:

```text
CatalogItemChangeSuggested
RecipeChangeSuggested
```

Proposal events сохраняются в Cloud raw/journal и proposal review tables, но не попадают в `inventory_event_queue`, пока не содержат stock movement.

Все кухонные события, включая `KitchenTicketStatusChanged`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted`, `CatalogItemChangeSuggested` и `RecipeChangeSuggested`, должны попадать в ClickHouse `raw_business_events`. ClickHouse является источником event trail для kitchen timing, audit, effective served-state analysis и последующей складской аналитики. PostgreSQL inbox/journal используется для приема, ACK, retry и idempotency, но не должен быть единственным аналитическим источником кухонной истории.

## Cloud Backend

### Proposal Review Tables

Реализовано сейчас:

- `cloud_catalog_suggestions`;
- `cloud_recipe_suggestions`;
- `cloud_recipe_suggestion_changes`;
- `cloud_suggestion_review_events`.

Статусы (реализовано сейчас):

- `pending`;
- `approved`;
- `rejected`;
- `changes_requested`;

Cloud routes:

```text
GET /api/v1/master-data/catalog-suggestions?restaurant_id=&status=&limit=&offset=
POST /api/v1/master-data/catalog-suggestions/{id}/approve
POST /api/v1/master-data/catalog-suggestions/{id}/reject
POST /api/v1/master-data/catalog-suggestions/{id}/request-changes
GET /api/v1/master-data/recipe-suggestions?restaurant_id=&status=&limit=&offset=
POST /api/v1/master-data/recipe-suggestions/{id}/approve
POST /api/v1/master-data/recipe-suggestions/{id}/reject
POST /api/v1/master-data/recipe-suggestions/{id}/request-changes
```

Approve rules:

- `CatalogItemChangeSuggested` creates/updates Cloud catalog only during approve/apply transaction.
- `RecipeChangeSuggested` creates a new Cloud recipe version or updates recipe items only during approve/apply transaction.
- For linked new dish proposal, Cloud applies catalog suggestion before recipe suggestion in the same transaction.
- After approve/apply Cloud creates a new master-data publication. POS Edge receives updated `catalog`, `recipes`, and `proposal_feedback`.

### Proposal Feedback Stream

Реализовано сейчас Cloud -> Edge stream:

```text
proposal_feedback
```

Payload содержит:

- `catalog_suggestions[]`;
- `recipe_suggestions[]`;
- `proposal_group_id`;
- `status`;
- `reviewed_by_employee_id`;
- `review_comment`;
- `cloud_version`;
- `cloud_updated_at`.

POS Edge применяет stream только к локальным `kitchen_proposals`. Он не меняет catalog/recipe read model через feedback stream.

### Warehouse Sequence

Реализовано сейчас для строгой последовательности складских изменений:

- `warehouse_id` в `inventory_event_queue`, `stock_documents`, `stock_ledger`;
- индекс `(restaurant_id, warehouse_id, occurred_at, id)`;
- worker claim не должен параллельно обрабатывать два pending events одного `(restaurant_id, warehouse_id)`;
- порядок применения: `occurred_at ASC`, затем `created_at ASC`, затем `event_id ASC`;
- replay accepted event не создает новый document из-за unique `(source_event_id, source_event_type)`.

Для кухонных событий ClickHouse `raw_business_events` хранит event trail. Реализовано сейчас: перед созданием складского документа Worker пропускает superseded `ItemServed`, если superseding served fact уже принят Cloud до обработки очереди. Запланировано далее: компенсирующий пересчет, если первый served fact уже успел создать stock document до recall/serve-again.

Если POS Edge еще не получил Cloud warehouse reference, backend использует default warehouse из локального `inventory_reference`; если его нет, mutating kitchen stock routes возвращают safe validation error `KITCHEN_WAREHOUSE_REQUIRED`.

## RBAC

Реализовано сейчас backend permissions:

```text
pos.kitchen.view
pos.kitchen.status.change
pos.catalog.view
pos.kitchen.catalog.view
pos.kitchen.recipe.view
pos.kitchen.recipe.suggest
pos.kitchen.catalog.suggest
pos.kitchen.stock.receipt
pos.kitchen.stock.inventory_count
pos.kitchen.stock.write_off
pos.kitchen.production.complete
pos.kitchen.stop_list.update
```

Роль `kitchen` получает:

- `pos.employee_shift.view_current`;
- `pos.catalog.view`;
- `pos.kitchen.view`;
- `pos.kitchen.status.change`;
- `pos.kitchen.catalog.view`;
- `pos.kitchen.recipe.view`;
- `pos.kitchen.recipe.suggest`;
- `pos.kitchen.catalog.suggest`;
- `pos.kitchen.stock.receipt`;
- `pos.kitchen.stock.inventory_count`;
- `pos.kitchen.stock.write_off`;
- `pos.kitchen.production.complete`.

Для локального seed/smoke kitchen role дополнительно публикуется с `pos.employee_shift.open`, `pos.employee_shift.close` и `pos.employee_shift.recent`, чтобы `scripts/seed-dev-system.py --run-kitchen-process-smoke` мог открыть персональную смену оператора кухни через POS Edge HTTP API. Чтение proposal status выполняется через `GET /api/v1/kitchen/proposals` под `pos.kitchen.view`; отдельного permission ID для чтения proposal status сейчас нет.

`pos.kitchen.stop_list.update` выдать только manager/kitchen-lead профилю, если владелец пилота разрешает Edge-side stop-list input.

## pos-ui-g Kitchen Mode

Реализовано сейчас: `currentMode === 'kds'` в `pos-ui-g` является backend-backed kitchen mode, а не placeholder.

Основной экран при входе: очередь заказов.

Нижний quick access внутри kitchen mode содержит минимальный набор разделов:

- `orders` - заказы;
- `stock` - склад;
- `kitchen` - кухня.

Навигация внутри каждого раздела выполняется верхними вкладками.

Раздел `orders`:

- `queue` - очередь заказов;
- `ready` - готовые к выдаче.

Раздел `stock`:

- `receipt` - приемка товара;
- `count` - ревизия;
- `writeoff` - списание;
- `production` - заготовки;

Раздел `kitchen`:

- `recipes` - техкарты;
- `suggestions` - catalog/recipe предложения;
- `my_proposals` - мои предложения и статусы.

### Queue UI

Order tile:

- крупный номер/`edge_order_id`;
- стол;
- elapsed time от первого active ticket;
- last status change time;
- цвет/тон по kitchen_order_status;
- блюда внутри карточки;
- для каждого блюда: название, количество, course/comment, station, status, таймер статуса, доступные actions.

Форматы отображения:

- `order_tiles` - основной режим;
- `status_columns` - группировка tickets по статусам;
- `compact_list` - плотный список для маленького экрана.

UI не меняет статус оптимистично. После action UI перечитывает backend truth.

### Stock Forms

Общие правила:

- catalog picker использует полный `GET /catalog/items`;
- ввод количества и цены сохраняет decimal strings / integer minor units без float totals в UI;
- supplier/counterparty, document date и line total обязательны для receipt;
- reason обязателен для write-off;
- связка receipt line с pending `catalog_suggestion_id` остается запланировано далее; текущий UI использует существующий `catalog_item_id`, как требует текущий Edge/Cloud stock contract;
- все пользовательские labels/errors идут через `pos-ui-g/src/shared/i18n`.

## Error Contract

Реализовано сейчас для Edge-side kitchen stock input validation:

- `KITCHEN_WAREHOUSE_REQUIRED`;
- `KITCHEN_RECEIPT_LINE_ITEM_REQUIRED`;
- `KITCHEN_RECEIPT_LINE_TOTAL_REQUIRED`;
- `KITCHEN_WRITE_OFF_REASON_REQUIRED`;
- `KITCHEN_INVENTORY_COUNT_EMPTY`;
- `KITCHEN_PRODUCTION_RECIPE_REQUIRED`.

Реализовано сейчас для recipe/proposal routes:

- `KITCHEN_RECIPE_NOT_FOUND`;
- `KITCHEN_RECIPE_SUGGESTION_LIMIT_EXCEEDED`;

Запланировано далее для Cloud proposal review feedback routes:

- `KITCHEN_PROPOSAL_NOT_FOUND`;
- `KITCHEN_PROPOSAL_ALREADY_REVIEWED`.

UI показывает safe localized messages и support code, не raw Go/SQL errors.

## Тестирование

POS Backend:

```bash
cd pos-backend
go mod tidy
go test ./...
```

Покрыть:

- kitchen order queue grouping and aggregate status;
- all valid/invalid ticket transitions and idempotent replay;
- replay of the same `serve command_id` writes one `ItemServed`;
- `served -> recall -> start -> ready -> serve` with a new `command_id` writes a new `ItemServed` with incremented `serve_sequence`;
- full catalog read for kitchen;
- recipe read endpoint;
- catalog suggestion event;
- recipe suggestion event and max prep-time delta;
- stock receipt event with existing catalog item;
- stock receipt event with pending catalog suggestion;
- inventory count event;
- stock write-off event with required reason;
- production completed event;
- RBAC denial for each mutating route.

Cloud Backend:

```bash
cd cloud-backend
go mod tidy
go test ./...
```

Покрыть:

- validation of new sync event types;
- receiver stores proposal events without inventory queue;
- receiver queues stock receipt/count/write-off/production;
- worker creates `PURCHASE`, `INVENTORY_COUNT`, `WASTE`, `PRODUCTION`;
- warehouse sequential processing;
- ClickHouse kitchen event trail is exported and used to determine latest effective `ItemServed`;
- recalled served events do not produce duplicate effective stock consumption after повторная подача;
- proposal approve/reject/request-changes;
- approve creates publication and feedback stream;
- replay idempotency.

POS UI `pos-ui-g`:

```bash
cd pos-ui-g
npm install
npm run build
```

Покрыть unit/component tests:

- kitchen API schemas;
- order queue rendering;
- action availability by status and permission;
- receipt line totals;
- proposal status rendering;
- safe error display.

Playwright smoke:

- cashier/waiter creates order with dish;
- kitchen sees order tile;
- kitchen moves line `accept -> start -> ready -> serve`;
- kitchen recalls served line and serves it again; Cloud effective served projection uses the latest `ItemServed`;
- outbox contains `KitchenTicketStatusChanged` and `ItemServed`;
- kitchen captures receipt with existing item;
- kitchen creates new good proposal and receipt line linked to it;
- kitchen creates new dish + recipe proposal group;
- kitchen submits inventory count, write-off and production;
- Cloud receives events and creates expected stock ledger rows for stock events.

Реализовано сейчас как HTTP профильный smoke: `scripts/seed-dev-system.py --run-kitchen-process-smoke` покрывает перечисленный kitchen/process сценарий без browser automation. Если вместе включены `--run-minimal-flow` и `--run-kitchen-process-smoke`, итоговый JSON содержит отдельные секции `minimal_flow` и `kitchen_process_smoke`; kitchen/process ветка должна идти под kitchen PIN, а не под manager PIN.

## Документация После Реализации

Обновить в том же PR:

- `SPECv1.3.md`;
- `ROADMAP.md`;
- `docs/backend/POS-BACKEND-SPEC.md`;
- `docs/backend/CLOUD-BACKEND-SPEC.md`;
- `docs/backend/INVENTORY-COSTING-SPEC.md`;
- `docs/backend/POS-ERROR-CATALOG.md`;
- `docs/sync/edge-cloud-contracts-v1.md`;
- `docs/sync/directional-sync-ownership.md`;
- `docs/ui/POS-UI-SPEC.md`;
- `docs/ui/POS-UI-RBAC.md`;
- `docs/ui/CLOUD-UI-SPEC.md`.

## Implementation Milestones

1. POS Edge kitchen read/write backend: queue, recipes, proposals, stock events, RBAC, tests.
2. Cloud sync contracts and inventory worker: new events, write-off, warehouse sequence, tests.
3. Cloud proposal review/apply and feedback stream.
4. `pos-ui-g` kitchen mode: queue first, then stock/proposal forms.
5. Cloud UI review surfaces.
6. End-to-end smoke and documentation alignment.

## Оставшиеся Решения Перед Кодом

Требуется подтвердить:

- default warehouse model: один Cloud-authored default kitchen warehouse на ресторан или несколько складов уже в первом шаге;
- может ли обычная роль `kitchen` менять stop-list, или это только manager/kitchen-lead;
- нужен ли partial `serve` для части количества одной order line, или вся kitchen ticket quantity считается served целиком;
- нужен ли отдельный compensating stock document при `served -> recall`, или достаточно delayed/effective consumption на основании последнего `ItemServed`;
- нужно ли Cloud auto-apply для catalog suggestions, или только ручной approve manager.
