# POS Data And Migrations

Статус: актуальный data/migration contract для текущего cashier runtime, целевого полного пилота и замороженных принципов ClickHouse event archive.

## Canonical Policy

Реализовано сейчас:

- POS Edge uses SQLite as local OLTP/source of truth.
- Cloud backend uses PostgreSQL as Cloud OLTP/source of truth.
- Active pre-pilot migration path uses one managed baseline SQL file per runtime module and runtime startup migration/verification.
- Existing dev/test databases are recreated from the baseline; data-preserving upgrade migrations are outside the current pre-client scope.
- Manual ad-hoc SQL is not canonical upgrade path.
- Реализовано сейчас: persistence implementation использует handwritten repository code, а не подтвержденный `sqlc`.

Запланировано далее:

- `sqlc` may be evaluated after schema and package boundaries stabilize.
- ClickHouse должен быть добавлен как immutable archive для всех business events и OLAP/reporting accelerator. Он не является transactional source of truth и не входит в POS transaction path.
- Все `event_id` для Edge POS, KDS и Cloud domain events должны быть UUIDv7.

## POS Edge SQLite

Managed SQL files, реализовано сейчас:

- `pos-backend/migrations/sqlite/001_init.sql`

Таблицы, реализованные сейчас:

- `restaurants`, `devices`, `edge_node_identity`, `edge_provisioning_state`, `client_devices`
- `roles`, `employees`, `auth_sessions`
- `halls`, `tables`
- `catalog_items`, `catalog_folders`, `catalog_folder_parameters`, `catalog_tags`, `catalog_item_tags`, `menu_items`, `modifier_groups`, `modifier_options`, `menu_item_modifier_groups`, `tax_profiles`, `tax_rules`
- `shifts`, `cash_sessions`, `cash_drawer_events`
- `orders`, `order_lines`
- `prechecks`, `precheck_lines`, `precheck_discounts`, `precheck_surcharges`, `precheck_taxes`, `payments`, `payment_attempts`, `checks`
- `order_line_modifiers`, `order_line_discounts`, `order_level_discounts`, `order_surcharges`, `service_charge_rules`, `pricing_policies`
- `financial_operations`, `financial_operation_items`
- `manager_override_audit`
- `local_event_log`, `pos_sync_outbox`, `cloud_master_sync_state`

Cashier runtime invariants:

- `orders.status` includes `open`, `locked`, `closed`, `cancelled`.
- Список закрытых заказов поддержан индексами bounded query: `orders_closed_restaurant_closed_at`, `orders_closed_shift_closed_at`, `orders_closed_device_closed_at`, `checks_business_date_closed_at`, `checks_order_id_closed_at`.
- Activity/sync reads поддержаны bounded queries and indexes: `local_event_log_created_at`, `local_event_log_event_type_created_at`, `local_event_log_command_id_created_at`, `pos_sync_outbox_status_sequence_no`, `pos_sync_outbox_pending_retry_sequence`, `pos_sync_outbox_processing_locked_at`, `pos_sync_outbox_command_id_created_at`.
- `prechecks` has immutable `snapshot`, `version`, `currency_code`, `paid_total`, `remaining_total`, `discount_total`, `surcharge_total`, `tax_total`, `total`.
- `precheck_*` breakdown tables persist lines, discounts, surcharges and tax components for audit/reprint/sync replay.
- `payments` references `precheck_id`, not legacy `check_id`.
- `checks` references `order_id` and stores immutable `snapshot`.
- `order_lines.course` и `order_lines.comment` хранят введенные оператором metadata комментария и курса подачи для cashier UI и не участвуют в расчете цены.
- `order_line_modifiers` stores selected modifiers for active order lines; precheck/check snapshots preserve selected modifiers for reprint/refund.
- `catalog_items.type` supports canonical `dish`, `good`, `semi_finished`, `service`; legacy `ingredient` is not accepted by current active catalog v2 path.
- `financial_operations` and `financial_operation_items` are append-only ledger tables for cancellation/refund; they do not mutate finalized payment/check/precheck rows.
- `financial_operation_items.scope` supports `whole_check`, `order_line`, `modifier_line`, `service_charge`, `tip`, `payment`.
- `financial_operations.inventory_disposition` stores `no_stock_effect`, `return_to_stock`, `write_off_waste` or `manual_review`; it is not an automatic stock movement.
- `financial_operations_check_type_created_at`, `financial_operations_restaurant_business_date_type_created_at`, `financial_operations_original_shift_created_at` and `financial_operations_check_created_at` support bounded ledger reads used by `GET /api/v1/checks/{id}/financial-operations` and `GET /api/v1/financial-operations`.
- `payments_business_date_shift_created_at`, `orders_closed_restaurant_created_at`, `local_event_log_occurred_at` and `pos_sync_outbox_created_at` are growth-control indexes for bounded operational reads; they do not imply physical retention/delete.
- `business_date_local` is stored for shifts, cash sessions, payments, checks and financial operations.
- Целевой Cloud-centric inventory contract запрещает POS Edge создавать складские документы и проводки.
- Текущий inventory baseline удалил Edge-side `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, `purchase_receipts`, `purchase_receipt_lines`; прежний Edge-side manual stock method был pre-pilot foundation и больше не является runtime path.
- Для локальной проверки stop-list Edge хранит только read-only recipes и двусторонний overlay `stop_lists`.

## Recipe/Inventory Runtime Boundary

Целевая POS Edge SQLite схема для inventory upgrade:

- `recipe_versions`
- `recipe_lines`
- `stop_lists`

Правила:

- `recipe_versions` и `recipe_lines` являются Cloud-owned read-only reference data для KDS UI и локальной проверки stop-list.
- `stop_lists` является двусторонним overlay: менеджер может обновить stop-list на Edge или в Cloud admin UI.
- POS Edge не создает `StockDocument`, `StockMove`, stock balance или costing rows.
- `StockDocumentPosted` не входит в целевой Edge -> Cloud operational catalog.
- Все складские документы, движения, остатки и себестоимость создаются только Cloud Inventory Worker.

Не реализовано сейчас:

- целевой `stop_lists` sync Edge <-> Cloud;
- recipe expansion and retro costing DAG.

Modifier acceptance реализовано сейчас только в order/pricing/precheck/check storage path: `order_line_modifiers` и `precheck_line_modifiers` сохраняют selected modifiers для replay/audit/reprint. В целевой inventory architecture Edge не знает о `ModifierOption.linked_catalog_item_id`; эту связь применяет только Cloud Inventory Worker.

UOM/status audit:

- unit fields остаются строками; UOM reference table with separate `code`, `name`, `short_name` and translations не реализована сейчас;
- catalog item `active` / Cloud lifecycle `status` не являются temporary availability overlays;
- temporary unavailability для продаж моделируется через `stop_lists`, а не через global catalog item lifecycle value.

## Cloud PostgreSQL

Managed SQL files, реализовано сейчас:

- `cloud-backend/migrations/postgres/001_init.sql`

Запланировано далее для Cloud event ingestion:

- PostgreSQL хранит `inbox_events` как transactional приемную очередь Cloud API.
- Cloud API после приема Edge outbox batch сохраняет events в `inbox_events` и отвечает `200 OK` без синхронной записи в ClickHouse.
- `inbox_events` должен иметь processing flag `processed_for_olap`.
- Async Batch Forwarder читает `inbox_events`, собирает batch от 1 000 до 100 000 rows и экспортирует их в ClickHouse.
- После successful export worker помечает события как `processed_for_olap = true`.

`001_init.sql` provides foundation for:

- roles and employees;
- menu categories;
- catalog folders and inherited folder parameters;
- catalog tags and item-tag assignments;
- catalog items with canonical kinds `dish`, `good`, `semi_finished`, `service`;
- dishes, goods, semi-finished products and services;
- recipe items;
- modifier groups/options and binding rules;
- menu items;
- menu item modifier group assignments;
- pricing policies for automatic discounts/surcharges;
- menu location assignments;
- master-data publications.

Ограничение текущей основы:

- Cloud recipe/inventory-adjacent foundation is not equal to POS Edge recipe/inventory runtime support.
- POS Edge `ApplyMasterData` сейчас принимает `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`.
- `catalog` stream applies catalog folders/tags/items, service catalog items and modifier groups/options/bindings/effective menu-item links; `menu` stream applies menu items.
- Cloud хранит menu categories отдельно от catalog folders; catalog publication не использует menu categories как замену folder hierarchy.
- `recipes` и `inventory_reference` могут существовать в constants/schema state, но пока не поддерживаются `mastersync.Service` apply path.

Реализовано сейчас для Cloud-centric inventory:

- PostgreSQL baseline содержит Cloud-owned `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists`.
- `stock_ledger` хранит `unit_cost_minor`, `total_cost_minor`, `costing_status`, `source_event_id`, `source_event_type`, `occurred_at` и `business_date_local`.
- Cloud Inventory Worker пишет документы и ledger из нормализованных item payloads.
- `modifier_options` получает optional `linked_catalog_item_id`; POS Edge не применяет это поле в order/pricing runtime.
- ClickHouse `raw_business_events` наполняется только Async Batch Forwarder из PostgreSQL `inbox_events` и является бессрочным архивом business events.
- ClickHouse `olap_stock_moves` наполняется только batch projection из PostgreSQL/ClickHouse event data и не является transactional source of truth.

## ClickHouse Immutable Event Store

Запланировано далее как замороженный принцип:

```text
Edge Outbox
  -> Cloud API (PostgreSQL inbox_events)
  -> Async Batch Forwarder
  -> ClickHouse raw_business_events
```

Запрещено выполнять synchronous dual-write в PostgreSQL и ClickHouse при обработке HTTP/sync request от кассы.

Целевая таблица ClickHouse: `raw_business_events`.

Engine:

```sql
MergeTree
```

Обязательные колонки:

| Column | Type | Правило |
| --- | --- | --- |
| `event_id` | UUID | UUIDv7 |
| `tenant_id` | UUID | tenant boundary |
| `restaurant_id` | UUID | restaurant boundary |
| `device_id` | UUID | source Edge/KDS device |
| `employee_id` | UUID | actor employee |
| `event_type` | String | domain event type |
| `occurred_at` | DateTime64 | extracted from UUIDv7 |
| `payload` | String | full original event body as JSON string |

Sorting key:

```sql
ORDER BY (tenant_id, event_type, event_id)
```

Partitioning:

```sql
PARTITION BY toYYYYMM(occurred_at)
```

Схема использует толстые metadata и JSON payload. Новые колонки под отдельные event types не добавляются.

## Retention And Archiving

PostgreSQL `inbox_events` является delivery queue и short-term operational buffer. ClickHouse `raw_business_events` является бессрочным archive для business events.

Правила:

- `processed_for_olap = false` events нельзя удалять из PostgreSQL.
- `processed_for_olap = true` events можно удалять из PostgreSQL после retention window.
- Базовый retention window для PostgreSQL `inbox_events`: 3 месяца.
- ClickHouse `raw_business_events` хранит events бессрочно.
- Удаление processed events из PostgreSQL не является loss of history, потому что historical event trail сохраняется в ClickHouse.

## Discount, Tax And Pricing Data

Реализовано сейчас:

- `Pricing` является отдельным runtime boundary от Order, Payment и Catalog.
- `order_line_discounts` хранит line/order discount commands для открытых orders и требует `application_index`.
- `order_surcharges` хранит manual/service/PB1 surcharge commands для открытых orders и требует `application_index`.
- `order_line_discounts.application_index` и `order_surcharges.application_index` используют одно ordered modifier space на order; application code и SQLite triggers отклоняют duplicate indexes между discount/surcharge tables там, где возможно.
- `service_charge_rules` является schema foundation для managed service-charge policy.
- `tax_profiles` и `tax_rules` хранят tax profile/rule foundation.
- `tax_profiles`, `tax_rules` и `service_charge_rules` включают Cloud -> Edge sync metadata: `cloud_version`, `cloud_updated_at`, `cloud_deleted_at`, `last_synced_at`.
- `menu_items.tax_profile_id` и `order_lines.tax_profile_id` позволяют snapshot tax policy без смешивания tax behavior с Catalog.
- `order_lines.course` и `order_lines.comment` являются POS runtime metadata для уже добавленной строки и не меняют pricing pipeline.
- `prechecks` и `checks` содержат `currency_code`, `discount_total`, `surcharge_total`, `tax_total`, `total`, `paid_total`, `remaining_total`.
- Precheck breakdown persistence использует `precheck_lines`, `precheck_discounts`, `precheck_surcharges`, `precheck_taxes`; discount/surcharge breakdown rows сохраняют `application_index`.
- Canonical calculation pipeline: `order lines subtotal -> unified ordered modifiers by application_index -> taxable base -> taxes -> grand total`.
- Taxes всегда считаются после всех discount/surcharge modifiers.
- `pricing_policy` применяет Cloud-authored tax/service-charge reference rows and automatic discount/surcharge `pricing_policies` как incremental или full snapshot payloads.
- Edge operational manual adjustments остаются runtime commands на открытых orders и требуют backend permissions. Будущие policy-backed manual adjustments должны ссылаться на synced policy ids; manual policy exceptions требуют отдельный permission/audit boundary до поддержки.
- Inclusive tax хранится в tax breakdown/tax total, но не увеличивает grand total; `tax_added_minor` в line breakdown фиксирует tax part, который добавлен к payable total.
- Rounding policy является deterministic integer half-up minor units; persistent money values остаются `INTEGER` minor units.

Запланировано далее:

- Cloud-authored policy authoring UI and policy-id-backed manual runtime adjustments;
- regional fiscal/legal tax adapter only after pilot foundation is stable.

Вне текущего runtime:

- UI-side financial calculation.

## Cancellation/Refund Ledger Data

Реализовано сейчас:

- `financial_operations` stores one append-only operation with type `cancellation` or `refund`, kind `full` or `partial`, amount, currency, `business_date_local`, original shift, current shift, check/precheck ids and immutable operation snapshot.
- `financial_operation_items` stores item allocations for whole check, order line, modifier line, service charge, tip and payment scope.
- SQLite triggers reject update/delete for both financial operation tables.
- Backend records `CancellationRecorded` and `RefundRecorded` outbox/local events.
- Backend exposes `GET /api/v1/checks/{id}/financial-operations` and `GET /api/v1/financial-operations` as read-only bounded ledger surfaces for activity detail and local reporting filters.
- Legacy payment refund route writes the same ledger through payment scope instead of updating payment/check/precheck statuses.
- Cashier UI whole-check и partial `order_line`/quantity cancellation/refund использует те же ledger endpoints, отправляет явный `inventory_disposition` и не требует schema changes или mutable status columns у finalized payments/checks. Line/quantity UI опирается на immutable check/precheck snapshot и пишет `financial_operation_items` со scope `order_line`; modifier/service/tip UI не реализован сейчас.
- Storage archive export сохраняет `financial_operations`, `financial_operation_items` и immutable snapshots как protected data в JSONL artifact без пересчета или мутации source rows.

- Реализовано сейчас в Cloud: `cloud_projection_financial_operations` хранит текущие projections для `CancellationRecorded`/`RefundRecorded` из raw/journal receipt с operation/check/precheck/shift/date/type/disposition/reason/snapshot metadata. Текущая validation financial operation payload требует совпадение payload `restaurant_id`/`device_id` с envelope, `precheck_id`, `reason` и immutable snapshot. Legacy `PaymentRefunded`/`CheckRefunded` не заполняют эту detailed projection.

Не реализовано сейчас:

- public Cloud HTTP reporting API/UI over financial operation projection;
- fiscal/correction document storage;
- automatic inventory stock moves from `inventory_disposition`.
- physical local delete/compaction policy для закрытых заказов.

## POS Edge Local Storage Lifecycle

Реализовано сейчас:

- Backend предоставляет read-only основу lifecycle через `GET /api/v1/storage/status` и `POST /api/v1/storage/retention/dry-run`.
- Backend предоставляет manifest-only archive/export plan через `POST /api/v1/storage/archive/export-plan`.
- Backend предоставляет export-only archive readiness через `POST /api/v1/storage/archive/export`.
- Backend предоставляет non-destructive archive read/lookup preview через `POST /api/v1/storage/archive/read-plan` и `POST /api/v1/storage/archive/lookup`.
- Backend предоставляет read-only apply-plan verification через `POST /api/v1/storage/archive/apply-plan`; destructive apply/delete остается disabled.
- Status использует SQLite PRAGMA `page_count`, `page_size`, `freelist_count`, `journal_mode` для безопасной оценки размера без чтения файловой системы.
- Status считает high-level объемы runtime tables: orders/order lines/modifiers, prechecks and breakdown tables, payments/attempts, checks, financial operation ledger, shifts, cash sessions, local events and outbox. Legacy Edge stock foundation tables удалены из целевого baseline и не входят в status counts.
- Status агрегирует closed orders по `checks.business_date_local` и возвращает oldest/newest closed check business date.
- Dry-run считает candidate rows только для closed orders с `checks.business_date_local < cutoff_business_date_local`; cutoff должен использовать формат `YYYY-MM-DD`.
- Dry-run не пишет и не удаляет строки. `financial_operations`, `financial_operation_items`, immutable precheck/check snapshots, local events и outbox остаются protected.
- Non-sent `edge_to_cloud` outbox rows возвращаются как blocking state для любой будущей destructive retention policy.
- Archive export-plan использует тот же cutoff rule `< cutoff_business_date_local`, возвращает `mode = manifest_only`, `result_mode = plan_only`, `destructive_apply_supported = false`, `blocked = true`, `archive_set`, protected flags для financial ledger, immutable snapshots, local events и outbox, active/open blockers (`active_orders`, `open_shifts`, `open_cash_sessions`), blocking outbox count и deterministic manifest `storage-archive-manifest-v1`.
- Archive export-plan не создает files/directories и не мутирует `orders`, `prechecks`, `payments`, `checks`, `financial_operations`, `financial_operation_items`, `local_event_log` или `pos_sync_outbox`.
- Archive export отбирает только closed orders с `checks.business_date_local <= cutoff_business_date_local`; невалидный или будущий cutoff отклоняется safe `INVALID` error.
- Archive export создает отдельную директорию export с `archive.jsonl` и `manifest.json`. Default runtime path задается `POS_SQLITE_ARCHIVE_DIR`; если он не задан в POS entrypoint, используется `archives` рядом с active SQLite data directory, а не внутри `.db` file.
- `archive.jsonl` является typed JSONL: каждая строка содержит `table` и `row`. Включаются closed orders, order lines/modifiers/discounts/surcharges, prechecks and breakdown tables, payments/payment attempts, checks, financial operations/items.
- `manifest.json` содержит version, `mode = export_only`, `result_mode = export_only`, generated_at, cutoff, reason, archive_id, archive/manifest paths, SHA-256 archive hash, counts, table list, business-date range, SQLite/source metadata, source node/device metadata если она есть в runtime, `runtime_rows_deleted = false`, protected flags and destructive block reasons.
- `local_event_log` и `pos_sync_outbox` включаются только как summary/reference rows без `payload_json`; manifest/counts явно показывают reference counts. Это сохраняет связь с outbox/local events без выноса потенциально sensitive payload history в архивный файл.
- Export-only archive не удаляет и не мутирует `orders`, `prechecks`, `payments`, `checks`, `financial_operations`, `financial_operation_items`, `local_event_log`, `pos_sync_outbox` или другие runtime tables.
- Export-only archive может создать файл даже при destructive block. `destructive_apply_supported = false`, `blocked = true`; non-sent `edge_to_cloud` outbox rows по candidate aggregates добавляют `pending_edge_to_cloud_outbox_for_archive_scope`.
- Archive read-plan принимает `manifest_path` и optional `archive_path`, разрешает только пути внутри configured archive dir, проверяет manifest version, SHA-256 archive file, counts manifest vs JSONL и наличие snapshot payload в `prechecks`/`checks`, затем возвращает summary/verification без business payload и без мутации runtime SQLite.
- Archive lookup принимает `manifest_path`, `archive_path` и ровно один ключ `check_id` или `order_id`, перед lookup выполняет ту же verification и streaming-способом возвращает immutable check/precheck snapshot preview и related counts. Lookup не пишет archive tables, runtime tables, `local_event_log` или `pos_sync_outbox` и не раскрывает summary payloads outbox/local events.
- Archive apply-plan читает manifest и archive JSONL, проверяет `version = pos_storage_archive_export_v1`, streaming-считает SHA-256 и rows по table, сверяет counts с manifest и текущим eligible runtime scope по `checks.business_date_local <= cutoff_business_date_local`, проверяет наличие snapshot payload в `prechecks` и `checks`.
- Archive apply-plan всегда возвращает `result_mode = apply_blocked`, `destructive_apply_supported = false`, `runtime_rows_deleted = false`; blockers включают missing/invalid manifest, version mismatch, SHA mismatch, manifest/runtime counts mismatch, pending Edge -> Cloud outbox, open operational boundary, missing runtime restore/apply path и disabled destructive apply policy.
- Apply-plan не отключает FK/triggers, не пишет archive tables, не удаляет runtime rows и не запускает `VACUUM`/compaction.
- Local event/outbox UI reads are bounded operational windows only; they are not cleanup/archive mechanisms and do not remove sync data.

Не реализовано сейчас:

- archive tables;
- restore в active SQLite для archived closed checks;
- physical delete of closed orders/checks/prechecks/payments/financial ledger rows;
- destructive apply policy;
- automatic VACUUM/compaction from HTTP API.

## Modifier Data

Реализовано сейчас:

- Cloud has `cloud_modifier_groups`, `cloud_modifier_options`, `cloud_menu_item_modifier_groups` and modifier binding foundation.
- POS Edge stores synced modifier groups/options/menu item links in read-model tables.
- POS Edge stores selected modifiers in `order_line_modifiers`.
- Precheck/check immutable snapshots include selected modifiers and their financial effect.
- Modifier price is included in backend authoritative calculation and immutable precheck/check snapshots.

Запланировано далее:

- modifier-to-recipe expansion belongs to recipe/inventory runtime;
- richer reporting/print formatting can be added after pilot acceptance.

## Recipe And Inventory Data

Запланировано до полного пилота:

- Recipes are versioned in Edge read-only tables `recipe_versions` and `recipe_lines`.
- Edge локально использует recipes только для KDS UI и проверки stop-list при добавлении позиции.
- Edge inventory mutation tables удалены из целевой SQLite схемы.
- Cloud Inventory Worker создает Cloud-owned stock documents and `stock_ledger` from Edge/KDS business events.
- `CheckClosed` запускает batch delta consumption после сверки с `ItemServed`.
- `RefundRecorded` и `CancellationRecorded` должны содержать operation-level `inventory_disposition`; отдельный `items[].inventory_disposition` в текущем payload не реализован.
- `StopListUpdated` синхронизируется Edge <-> Cloud.
- UOM reference model with separate code/display fields remains запланировано далее.
- `ProductionCompleted` создает `PRODUCTION`: приход заготовки и расход сырья.
- semi-finished fallback expansion.
- costing recalculation.
- ClickHouse `olap_stock_moves`.
- Cloud OLAP API читает ClickHouse projections и не участвует в transactional command validation.

Вне текущего runtime:

- automatic recipe consumption after check;
- automatic return-to-stock/write-off after cancellation/refund;
- KDS `ItemServed` inventory trigger.

## Migration Safety

Required behavior:

- startup must run schema upgrade before business runtime access;
- DB version newer than runtime version must fail fast;
- schema verification must check critical tables/columns/indexes before HTTP runtime;
- after the first client deployment, existing DB upgrade must have backup path and explicit data-preserving migration files;
- destructive SQLite cleanup/reset must be explicit, audited and documented before being exposed in UI/admin flows.

## Pricing policy audit columns

Статус: реализовано сейчас.

Managed SQLite baseline `001_init.sql` хранит `pricing_policy_id` в `order_line_discounts`, `order_surcharges`, `precheck_discounts` и `precheck_surcharges`. Это связывает runtime adjustment и immutable precheck snapshot с Cloud-authored policy для audit, replay и проверки, что размер/тип/scope/application order были скопированы из опубликованного справочника.

Статус: реализовано сейчас.

Edge table `pricing_policies` хранит `manual` из Cloud payload, чтобы POS runtime мог явно отклонять manual override policies в обычном pilot selection flow.
