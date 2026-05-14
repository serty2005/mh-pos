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
- `pos-backend/migrations/sqlite/003_pricing_policy_sync_foundation.sql`

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
- `007_refund_and_pricing_policy_hardening.sql`

`004_master_data_authority.sql` provides foundation for:

- roles and employees;
- categories;
- catalog items with canonical kinds `dish`, `good`, `ingredient`, `semi_finished`;
- dishes, goods and semi-finished products;
- recipe items;
- modifier groups/options;
- menu items;
- menu item modifier group assignments;
- menu location assignments;
- master-data publications.

Foundation warning:

- Cloud modifier/recipe/catalog foundation is not equal to POS Edge runtime support.
- POS Edge `ApplyMasterData` сейчас принимает `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`.
- Cloud хранит menu categories как master-data foundation, но текущий publication payload не включает `categories`, пока POS Edge не имеет поддерживаемого category ingest contract.
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
- `pricing_policy` применяет Cloud-authored tax/service-charge reference rows как incremental или full snapshot payloads; он не включает modifiers runtime или advanced Cloud-owned pricing logic.
- Edge operational adjustments остаются runtime commands на открытых orders. Будущие policy-backed adjustments должны ссылаться на synced policy ids; manual policy exceptions требуют отдельный permission/audit boundary до поддержки.
- Inclusive tax хранится в tax breakdown/tax total, но не увеличивает grand total; `tax_added_minor` в line breakdown фиксирует tax part, который добавлен к payable total.
- Rounding policy является deterministic integer half-up minor units; persistent money values остаются `INTEGER` minor units.

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
