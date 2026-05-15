# POS Data And Migrations

Статус: актуальный data/migration contract для frozen cashier pilot.

## Canonical Policy

Реализовано сейчас:

- POS Edge uses SQLite as local OLTP/source of truth.
- Cloud backend uses PostgreSQL as Cloud OLTP/source of truth.
- Active pre-pilot migration path uses one managed baseline SQL file per runtime module and runtime startup migration/verification.
- Existing dev/test databases are recreated from the baseline; data-preserving upgrade migrations are outside the current pre-client scope.
- Manual ad-hoc SQL is not canonical upgrade path.
- Current persistence implementation is handwritten repository code, not confirmed `sqlc`.

Запланировано далее:

- `sqlc` may be evaluated after schema and package boundaries stabilize.
- ClickHouse may be added only as Cloud OLAP/reporting accelerator, not source of truth.

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
- `prechecks` has immutable `snapshot`, `version`, `currency_code`, `paid_total`, `remaining_total`, `discount_total`, `surcharge_total`, `tax_total`, `total`.
- `precheck_*` breakdown tables persist lines, discounts, surcharges and tax components for audit/reprint/sync replay.
- `payments` references `precheck_id`, not legacy `check_id`.
- `checks` references `order_id` and stores immutable `snapshot`.
- `order_line_modifiers` stores selected modifiers for active order lines; precheck/check snapshots preserve selected modifiers for reprint/refund.
- `catalog_items.type` supports canonical `dish`, `good`, `semi_finished`, `service`; legacy `ingredient` is not accepted by current active catalog v2 path.
- `financial_operations` and `financial_operation_items` are append-only ledger tables for cancellation/refund; they do not mutate finalized payment/check/precheck rows.
- `financial_operation_items.scope` supports `whole_check`, `order_line`, `modifier_line`, `service_charge`, `tip`, `payment`.
- `financial_operations.inventory_disposition` stores `no_stock_effect`, `return_to_stock`, `write_off_waste` or `manual_review`; it is not an automatic stock movement.
- `business_date_local` is stored for shifts, cash sessions, payments, checks and financial operations.
- `stock_moves` are append-only by trigger.

## Таблицы Только Основы

SQLite: реализована только основа:

- `recipe_versions`
- `recipe_lines`
- `purchase_receipts`
- `purchase_receipt_lines`
- `stock_documents`
- `stock_moves`
- `stock_balances`
- `item_costs`

Эти таблицы не означают готовый inventory runtime. Текущий код не подтверждает:

- automatic stock consumption;
- recipe expansion;
- modifier-to-recipe expansion;
- cashier-facing inventory mutation flow;
- app services that post stock documents from final checks.

## Cloud PostgreSQL

Managed SQL files, реализовано сейчас:

- `cloud-backend/migrations/postgres/001_init.sql`

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
- Cloud хранит menu categories отдельно от catalog folders; catalog publication не использует menu categories как замену folder hierarchy.
- `recipes` и `inventory_reference` могут существовать в constants/schema state, но пока не поддерживаются `mastersync.Service` apply path.

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
- Legacy payment refund route writes the same ledger through payment scope instead of updating payment/check/precheck statuses.

Не реализовано сейчас:

- separate refund projection tables in Cloud;
- fiscal/correction document storage;
- automatic inventory stock moves from `inventory_disposition`.

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

Реализована только основа:

- Recipes are versioned in SQLite via `recipe_versions` and `recipe_lines`.
- Cloud has recipe item foundation.
- Stock movement foundation exists through stock documents/moves/balances/costs.
- Cancellation/refund ledger can record intended inventory disposition, but stock tables stay unchanged until an explicit Inventory service posts stock documents/moves.

Запланировано далее:

- consumption trigger policy;
- stock document posting services;
- snapshot data sufficient for inventory/fiscal/reporting replay.

Вне текущего runtime:

- automatic recipe consumption after check;
- automatic return-to-stock/write-off after cancellation/refund;
- KDS/DishServed inventory trigger;
- semi-finished fallback expansion.

## Migration Safety

Required behavior:

- startup must run schema upgrade before business runtime access;
- DB version newer than runtime version must fail fast;
- schema verification must check critical tables/columns/indexes before HTTP runtime;
- after the first client deployment, existing DB upgrade must have backup path and explicit data-preserving migration files;
- destructive SQLite cleanup/reset must be explicit, audited and documented before being exposed in UI/admin flows.
