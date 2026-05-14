# POS Data And Migrations

Статус: актуальный data/migration contract для frozen cashier pilot.

## Canonical Policy

Реализовано сейчас:

- POS Edge uses SQLite as local OLTP/source of truth.
- Cloud backend uses PostgreSQL as Cloud OLTP/source of truth.
- Active migration path uses managed SQL files and runtime startup migration/verification.
- Manual ad-hoc SQL is not canonical upgrade path.
- Current persistence implementation is handwritten repository code, not confirmed `sqlc`.

Запланировано далее:

- `sqlc` may be evaluated after schema and package boundaries stabilize.
- ClickHouse may be added only as Cloud OLAP/reporting accelerator, not source of truth.

## POS Edge SQLite

Managed files:

- `pos-backend/migrations/sqlite/001_init.sql`
- `pos-backend/migrations/sqlite/002_runtime_schema_repair.sql`

Таблицы, реализованные сейчас:

- `restaurants`, `devices`, `edge_node_identity`, `edge_provisioning_state`, `client_devices`
- `roles`, `employees`, `auth_sessions`
- `halls`, `tables`
- `catalog_items`, `menu_items`, `tax_profiles`, `tax_rules`
- `shifts`, `cash_sessions`, `cash_drawer_events`
- `orders`, `order_lines`
- `prechecks`, `precheck_lines`, `precheck_discounts`, `precheck_surcharges`, `precheck_taxes`, `payments`, `payment_attempts`, `checks`
- `order_line_discounts`, `order_level_discounts`, `order_surcharges`, `service_charge_rules`
- `manager_override_audit`
- `local_event_log`, `pos_sync_outbox`, `cloud_master_sync_state`

Cashier runtime invariants:

- `orders.status` includes `open`, `locked`, `closed`, `cancelled`.
- `prechecks` has immutable `snapshot`, `version`, `currency_code`, `paid_total`, `remaining_total`, `discount_total`, `surcharge_total`, `tax_total`, `total`.
- `precheck_*` breakdown tables persist lines, discounts, surcharges and tax components for audit/reprint/sync replay.
- `payments` references `precheck_id`, not legacy `check_id`.
- `checks` references `order_id` and stores immutable `snapshot`.
- `business_date_local` is stored for shifts, cash sessions, payments and checks.
- `stock_moves` are append-only by trigger.

## Foundation Only Tables

SQLite foundation only:

- `recipe_versions`
- `recipe_lines`
- `purchase_receipts`
- `purchase_receipt_lines`
- `stock_documents`
- `stock_moves`
- `stock_balances`
- `item_costs`

These tables are not proof of a finished inventory runtime. Current code does not confirm:

- automatic stock consumption;
- recipe expansion;
- modifier-to-recipe expansion;
- cashier-facing inventory mutation flow;
- app services that post stock documents from final checks.

## Cloud PostgreSQL

Managed files currently present:

- `001_sync_receiver.sql`
- `002_projection_event_type_stats.sql`
- `003_runtime_schema_repair.sql`
- `004_master_data_authority.sql`
- `005_master_data_restaurants_api.sql`
- `006_zero_to_cashier_provisioning.sql`

`004_master_data_authority.sql` provides foundation for:

- roles and employees;
- categories;
- catalog items with kinds `dish`, `good`, `raw_material`, `semi_finished`;
- dishes, goods and semi-finished products;
- recipe items;
- modifier groups/options;
- menu items;
- menu item modifier group assignments;
- menu location assignments;
- master-data publications.

Foundation warning:

- Cloud modifier/recipe/catalog foundation is not equal to POS Edge runtime support.
- POS Edge `ApplyMasterData` currently ingests only `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.
- `recipes` and `inventory_reference` may exist in constants/schema state, but they are not supported by `mastersync.Service` apply path yet.

## Discount, Tax And Pricing Data

Реализовано сейчас:

- `Pricing` is a separate runtime boundary from Order, Payment and Catalog.
- `order_line_discounts` stores line/order discount commands for open orders and requires `application_index`.
- `order_surcharges` stores manual/service/PB1 surcharge commands for open orders and requires `application_index`.
- `order_line_discounts.application_index` and `order_surcharges.application_index` use one ordered modifier space per order; application code and SQLite triggers reject duplicate indexes across discount/surcharge tables where possible.
- `service_charge_rules` is schema foundation for managed service-charge policy.
- `tax_profiles` and `tax_rules` store tax profile/rule foundation.
- `menu_items.tax_profile_id` and `order_lines.tax_profile_id` allow tax policy snapshotting without mixing tax behavior into Catalog.
- `prechecks` and `checks` contain `currency_code`, `discount_total`, `surcharge_total`, `tax_total`, `total`, `paid_total`, `remaining_total`.
- Precheck breakdown persistence uses `precheck_lines`, `precheck_discounts`, `precheck_surcharges`, `precheck_taxes`; discount/surcharge breakdown rows persist `application_index`.
- Canonical calculation pipeline is `order lines subtotal -> unified ordered modifiers by application_index -> taxable base -> taxes -> grand total`.
- Taxes are always calculated after all discount/surcharge modifiers.
- Inclusive tax is stored in tax breakdown/tax total but does not increase grand total; `tax_added_minor` in line breakdown records the tax part that was added to payable total.
- Rounding policy is deterministic integer half-up minor units; persistent money values remain `INTEGER` minor units.

Запланировано далее:

- Cloud-authored policy data published to Edge;
- regional fiscal/legal tax adapter only after pilot foundation is stable.

Вне текущего runtime:

- UI-side financial calculation.

## Modifier Data

Foundation only:

- Cloud has `cloud_modifier_groups`, `cloud_modifier_options`, `cloud_menu_item_modifier_groups`.
- POS Edge order/precheck/check tables do not currently store selected modifiers.

Запланировано до пилота if accepted:

- Edge read model and publication payload for modifier groups/options.
- Order line selected modifiers snapshot.
- Modifier price delta included in backend authoritative calculation.

## Recipe And Inventory Data

Foundation only:

- Recipes are versioned in SQLite via `recipe_versions` and `recipe_lines`.
- Cloud has recipe item foundation.
- Stock movement foundation exists through stock documents/moves/balances/costs.

Запланировано далее:

- consumption trigger policy;
- stock document posting services;
- snapshot data sufficient for inventory/fiscal/reporting replay.

Вне текущего runtime:

- automatic recipe consumption after check;
- KDS/DishServed inventory trigger;
- semi-finished fallback expansion.

## Migration Safety

Required behavior:

- startup must run schema upgrade before business runtime access;
- DB version newer than runtime version must fail fast;
- schema verification must check critical tables/columns/indexes before HTTP runtime;
- existing DB upgrade must have backup path;
- destructive SQLite cleanup/reset must be explicit, audited and documented before being exposed in UI/admin flows.
