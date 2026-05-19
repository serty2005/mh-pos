# POS Backend Spec

Статус: актуальный backend contract для frozen cashier pilot.

Код и тесты являются источником истины. Этот документ не описывает будущие API как реализованные.

## Runtime Modules

Реализовано сейчас:

- POS Edge HTTP API на `chi`.
- SQLite repository/migration/runtime verification.
- Application services для auth, staff shifts, cash sessions, floor/menu/catalog reads, order, pricing, precheck, payment/check, master-data ingest, outbox.
- Manual persistence implementation в infrastructure repositories.
- Selected modifiers runtime для order lines, pricing, precheck/check snapshots.
- API чтения активных заказов зала для статусов столов и панели активных заказов POS.
- Runtime-поля курса подачи и комментария строки заказа, которые не влияют на финансовые итоги.
- Service catalog items as sellable POS items.

Не реализовано сейчас:

- `sqlc` как текущий persistence implementation;
- ClickHouse runtime;
- inventory consumption engine;
- Cloud Inventory Worker и costing engine;
- payment processor module;
- fiscal adapter.

## Public Routes

Реализовано сейчас в `pos-backend/internal/pos/api/router.go`:

- `GET /health`
- `POST /api/v1/auth/pin-login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/session`
- `POST /api/v1/system/pair`
- `GET /api/v1/system/pairing-status`
- `GET /api/v1/system/provisioning-status`
- `POST /api/v1/system/provisioning/register-cloud`
- `POST /api/v1/system/provisioning/pair-via-license`
- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/catalog/items`
- `GET /api/v1/menu/items`
- `POST /api/v1/employee-shifts/open`
- `POST /api/v1/employee-shifts/{id}/close`
- `GET /api/v1/employee-shifts/current`
- `GET /api/v1/employee-shifts/recent`
- `POST /api/v1/orders`
- `GET /api/v1/orders/current`
- `GET /api/v1/orders/active`
- `GET /api/v1/orders/closed`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/modifiers`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/details`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `GET /api/v1/orders/{id}/pricing`
- `POST /api/v1/orders/{id}/discounts`
- `POST /api/v1/orders/{id}/surcharges`
- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/orders/{id}/prechecks`
- `POST /api/v1/orders/{id}/close`
- `GET /api/v1/prechecks/{id}`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/reprint`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/checks/{id}`
- `POST /api/v1/checks/{id}/reprint`
- `POST /api/v1/checks/{id}/cancellations`
- `POST /api/v1/checks/{id}/refunds`
- `POST /api/v1/payments/{id}/refund`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `GET /api/v1/cash-shifts/current`
- `POST /api/v1/cash-drawer-events`
- `GET /api/v1/sync/outbox`
- `GET /api/v1/sync/local-events`
- `GET /api/v1/sync/status`
- `POST /api/v1/sync/retry-failed`
- `POST /api/v1/sync/master-data/snapshots`
- `POST /api/v1/sync/master-data/{stream}`
- `GET /api/v1/storage/status`
- `POST /api/v1/storage/retention/dry-run`

## Current/Optional Reads

Реализовано сейчас:

- `GET /api/v1/employee-shifts/current` возвращает `200` с JSON `null`, когда у authenticated employee нет открытой личной смены.
- `GET /api/v1/cash-shifts/current` и `GET /api/v1/orders/current?table_id=...` пока используют `404 NOT_FOUND` как empty state, когда текущая сущность отсутствует.
- Эти empty states не retryable и не означают инфраструктурный сбой. Cashier UI превращает `200 null` или optional `404` в `null` через optional read helper и показывает состояние "нет открытой смены/кассовой смены/активного заказа".
- `GET /api/v1/employee-shifts/current` ищет текущую личную смену по authenticated employee в restaurant context, а не по устройству.
- `GET /api/v1/cash-shifts/current` ищет открытую кассовую смену по устройству из authenticated request context.
- `node_device_id` в query/header остается частью authenticated device/session metadata; он не выбирает personal employee shift вместо actor session.

## Cashier Flow

Реализовано сейчас:

1. Кассир открывает личную смену сотрудника.
2. Кассир открывает кассовую смену устройства.
3. Кассир создает заказ на стол.
4. Кассир добавляет, меняет количество и списывает активные строки заказа, включая selected modifiers для menu items с modifier groups.
5. Кассир может сохранять metadata курса подачи и комментария строки через backend; эти поля возвращаются вместе со строкой и не меняют итоги.
6. Кассир может читать все активные заказы зала через `GET /api/v1/orders/active?hall_id=...`, поэтому статусы столов и панель активных заказов используют backend-данные, а не UI mock-данные.
7. Кассир может применять backend-authoritative discount/surcharge commands и читать pricing preview.
8. Кассир выпускает пречек.
9. Backend блокирует заказ и создает immutable financial precheck snapshot.
10. Кассир проводит одну или несколько оплат через `precheck_id`.
11. Backend создает final check только после полной оплаты.
12. Кассир или менеджер может повторно напечатать копию precheck/check из immutable snapshot.
13. Авторизованный оператор может записать cancellation/refund operation; текущий cashier UI использует check-level ledger routes для full whole-check и partial `order_line`/quantity cancellation/refund и оставляет compatibility payment refund route как fallback для закрытых заказов.

Read contract закрытых заказов:

- Реализовано сейчас: `GET /api/v1/orders/closed` принимает `limit`, `offset`, `business_date_local`, `from_business_date_local`, `to_business_date_local`, `shift_id`, `device_id`, `check_id`.
- Default `limit` = `50`, max `limit` = `100`; отрицательный `offset` и невалидные business date фильтры отклоняются.
- Сортировка stable newest-first: close timestamp, затем `id`.
- API без фильтра возвращает только bounded latest page, а не всю историю.

Operational activity/sync read contract:

- Реализовано сейчас: `GET /api/v1/sync/outbox` принимает optional `limit`; backend repository возвращает bounded default page `100`, если limit пустой, отрицательный, нулевой или больше `500`.
- Реализовано сейчас: `GET /api/v1/sync/local-events` принимает optional `limit` и `event_type`; backend repository возвращает bounded default page `100`, если limit пустой, отрицательный, нулевой или больше `500`, и сортирует по `created_at DESC, id DESC`.
- POS UI sync/activity drawer использует `limit=5` для outbox и local events; эти reads не являются бесконечным журналом отчетности.
- `GET /api/v1/sync/status` агрегирует counts by outbox status and does not return payload history.

Контракт lifecycle локального storage:

- Реализовано сейчас: `GET /api/v1/storage/status` возвращает read-only operational snapshot локальной SQLite БД: page stats (`page_count`, `page_size_bytes`, `freelist_count`, estimated size), high-level table counts, диапазон business date закрытых чеков, закрытые заказы по business date, outbox counts by status/direction и число blocking Edge -> Cloud outbox messages.
- Реализовано сейчас: `POST /api/v1/storage/retention/dry-run` принимает `cutoff_business_date_local` в формате `YYYY-MM-DD` и считает документы с `checks.business_date_local < cutoff`, которые могли бы войти в будущую archive/retention policy.
- Оба endpoint требуют operator session с `pos.sync.view`; UI visibility не является security boundary.
- Текущий retention mode равен `dry_run_only`: response всегда помечает destructive apply как unsupported, ledger/snapshots как protected и возвращает block reason `dry_run_only_no_archive_policy`. Если есть non-sent `edge_to_cloud` outbox messages, добавляется `pending_edge_to_cloud_outbox`.
- Не реализовано сейчас: физическое удаление, перенос в archive tables/files, restore/read path из архива, VACUUM как часть HTTP lifecycle flow.

## Precheck Contract

Реализовано сейчас:

- `IssuePrecheckCommand` contains `order_id`.
- Precheck can be issued only for `open` order.
- Order device must match command device.
- Only one active issued precheck per order is allowed.
- Snapshot contains active lines with selected modifiers, currency, discounts, surcharges, taxes, totals and calculation breakdown at issue time.
- Order becomes `locked`.
- `CancelPrecheckCommand` requires `precheck_id`, `manager_employee_id`, `manager_pin`, `cancellation_reason`.
- Cancel requires operator permission `pos.precheck.cancel.request` and manager permission `pos.precheck.cancel`.
- Cancel writes manager override audit and returns order to `open`.
- Reprint requires `pos.precheck.reprint` and valid immutable snapshot.

Pricing contract:

- `GET /api/v1/orders/{id}/pricing` returns backend-calculated preview for open/current order state.
- `POST /api/v1/orders/{id}/discounts` supports line/order discounts with `amount_kind = percentage|fixed` and required `application_index`.
- `POST /api/v1/orders/{id}/surcharges` supports `service_charge`, `pb1_service_fee` and `manual` surcharge foundation with `amount_kind = percentage|fixed` and required `application_index`.
- Canonical pipeline is `order lines subtotal -> unified ordered modifiers by application_index -> taxable base -> taxes -> grand total`.
- Discounts and surcharges share one `application_index` space per calculation snapshot; duplicate indexes are rejected.
- Surcharge is a separate domain operation and is not represented as a negative discount.
- Tax Always Last is enforced by the calculation pipeline.
- Rounding policy is deterministic integer half-up minor units (`integer_half_up_minor_units_v1`).
- Tax foundation supports `tax_profiles`, `tax_rules`, percentage/fixed components, inclusive/exclusive mode, compound foundation and tax exempt profile foundation; inclusive tax is included in `tax_total`, but does not increase grand total.
- Precheck issue persists immutable breakdown in `precheck_lines`, `precheck_discounts`, `precheck_surcharges`, `precheck_taxes`.
- Menu price/tax rule changes after precheck issue do not mutate old precheck/check snapshots.
- Selected modifiers are priced by backend calculation and persisted in `order_line_modifiers`, precheck/check snapshots and reprint payloads. `POST /orders/{id}/lines` and `PATCH /orders/{id}/lines/{line_id}/modifiers` validate required/min/max, active group/option, option-group membership, menu item link and non-negative option price. Modifier update is allowed only for active lines of open editable orders without active precheck/finalized check and writes `OrderLineModifiersUpdated` to outbox/local events.
- Service items use the same order/pricing/precheck/check flow as other sellable menu items and do not imply recipe semantics.
- Catalog item lifecycle/availability audit: Edge runtime хранит active catalog/menu read models из Cloud lifecycle status. Temporary unavailability не реализована как глобальный catalog item status; запланировано далее моделировать ее как overlay menu/restaurant/terminal-group, если это будет принято.
- UOM audit: текущий runtime хранит string units (`base_unit`, recipe `unit`, stock move `unit`). Separate UOM reference model with machine `code`, `name`, `short_name` and translations не реализована сейчас, поэтому новый runtime code не должен считать display labels canonical UOM codes.

## Payment, Check, Cancellation And Refund Contract

Реализовано сейчас:

- Payment capture endpoint is `POST /api/v1/prechecks/{id}/payments`.
- Capture command accepts `method`, `amount`, `currency` and optional provider metadata.
- Supported methods are `cash`, `card`, `other`.
- Payment requires open cash session for the same device, shift and restaurant.
- Payment updates `prechecks.paid_total`.
- Payment updates `prechecks.remaining_total`.
- Partial payments are allowed; overpayment is rejected.
- Full payment creates one final check and closes order.
- Check snapshot includes immutable precheck snapshot and payment snapshot.
- Check reprint/refund use immutable snapshots and do not re-read current menu modifier data.
- Reprint check requires `pos.check.reprint`.
- Cancellation endpoint is `POST /api/v1/checks/{id}/cancellations`.
- Refund endpoint is `POST /api/v1/checks/{id}/refunds`.
- Ledger endpoints принимают `command_id`, `operation_kind`, `inventory_disposition`, reason и optional item scopes. Текущий cashier UI отправляет whole-check commands без item list, поэтому backend записывает `whole_check` из immutable check snapshot; для partial line/quantity UI отправляет `items[]` со scope `order_line`, `order_line_id`, `quantity`, `amount`, `currency` и `tax_amount`.
- Compatibility refund endpoint is `POST /api/v1/payments/{id}/refund`; it records a refund operation with payment allocation and does not mutate finalized payment/check/precheck totals.
- Compatibility refund requires the captured payment to belong to an order that already has a finalized check. Captured partial payment on a still-issued precheck is not refundable through this endpoint.
- Cancellation uses permission `pos.precheck.cancel`; refund uses permission `pos.payment.refund`.
- Cancellation requires the original personal shift to be open, the current cash session to belong to that shift and the same `business_date_local`.
- Refund requires the original personal shift to be closed or the current `business_date_local` to differ from the check business date; a current open cash session is still required.
- Financial operations are append-only rows in `financial_operations` and `financial_operation_items`.
- Operation type is `cancellation` or `refund`; operation kind is `full` or `partial`.
- Item scopes are `whole_check`, `order_line`, `modifier_line`, `service_charge`, `tip`, `payment`.
- Backend rejects over-cancel, over-refund, over-line-amount, over-line-quantity and over-payment-allocation scenarios.
- Operation snapshot embeds immutable check snapshot and operation items.
- Inventory disposition is explicit: `no_stock_effect`, `return_to_stock`, `write_off_waste`, `manual_review`; financial operation does not mutate stock tables.
- Current POS Edge events are `CancellationRecorded` and `RefundRecorded`. New refund runtime does not emit legacy `PaymentRefunded`/`CheckRefunded`; those names remain Cloud-accepted legacy sync event types only.

Не реализовано сейчас:

- automatic PSP authorization/capture/refund;
- fiscal receipt creation;
- refund manager PIN policy beyond current RBAC permission check;
- cashier UI for modifier/service/tip partial cancellation/refund;
- automatic stock return/write-off;
- separate `business_day` and `fiscal_shift` runtime aggregates.

## Shift, Business Date And Fiscal Boundary

Реализовано сейчас:

- Runtime имеет personal employee shift (`shifts`) и device cash session (`cash_sessions`).
- `cashier_shift` в текущем коде представлен personal employee shift aggregate/table `shifts`; отдельного объекта с именем `cashier_shift` нет.
- `business_date_local` хранится у shifts, cash sessions, payments, checks и financial operations.
- Отдельной сущности `business_day` нет; текущая business date вычисляется из restaurant settings.
- Отдельной сущности `fiscal_shift` нет.
- Final check является finalized internal POS document после полной оплаты.
- Cancellation/refund являются compensating operations и не переписывают finalized check/payment.

Вне текущего объема:

- fiscalized receipt state;
- correction document flow;
- fiscal document reprint vs ordinary POS snapshot reprint distinction in runtime;
- reopen business day policy as a first-class operation.

## Master Data Ingest

Реализовано сейчас:

- `ApplyMasterData` принимает `full_snapshot` и `incremental`.
- `full_snapshot` требует `full_snapshot_reason` со значением `terminal_restaurant_changed` или `node_role_changed`.
- Backup hook для full snapshot существует, если он настроен.
- Поддерживаемые POS Edge ingest streams:
  - `restaurants`
  - `devices`
  - `staff`
  - `floor`
  - `catalog`
  - `menu`
  - `pricing_policy`
- `catalog` применяет catalog folders, folder parameters, catalog tags, item tags, item kinds `dish`, `good`, `semi_finished`, `service` и modifier groups/options/bindings.
- `menu` применяет menu items, menu item `item_type` и effective menu item modifier group links после применения menu items.
- Cloud publication package для POS Edge должен соответствовать typed ingest DTO `ApplyMasterDataCommand`: modifier groups/options/bindings публикуются top-level массивами с `restaurant_id` и без Cloud lifecycle fields, а `menu_item_modifier_groups` остается link-only (`menu_item_id`, `modifier_group_id`, `sort_order`).
- `required`, `min_count`, `max_count`, `active` принадлежат top-level `modifier_groups[]`; эти поля не публикуются внутри `menu_item_modifier_groups[]` и не встраиваются как rich `menu_items[].modifier_groups[]` в ingest payload. Inventory/recipe expansion для modifiers не выполняется в POS order/pricing/precheck/check runtime.
- `restaurants` применяет Cloud-authored settings и `active`; опубликованный active restaurant сохраняется в Edge read model как active row.
- `pricing_policy` применяет `tax_profiles`, `tax_rules`, `service_charge_rules` и automatic discount/surcharge `pricing_policies` с sync metadata.
- Strict JSON decode отклоняет неизвестные request fields; unsupported stream names отклоняются до partial apply.

Только основа:

- Domain constants и SQLite state знают о `recipes` и `inventory_reference`, но `mastersync.Service` пока не применяет эти streams.
- Cloud schema foundation для recipes/inventory-adjacent data не делает их поддерживаемыми POS Edge ingest payloads.

## Pricing, Modifiers And Inventory Boundaries

Discounts/taxes:

- Реализовано сейчас: separate `Pricing` policy area, backend authoritative calculation, unified ordered discount/surcharge pipeline, tax-last invariant, immutable precheck breakdown and no UI authoritative totals.
- Реализовано сейчас: Cloud-authored tax/service-charge/automatic discount-surcharge reference rows применяются через `pricing_policy`.
- Реализовано сейчас: selected modifiers participate in line/order totals and immutable snapshots.
- Реализовано сейчас: service items are priced as normal sellable order lines.
- Запланировано далее: полный Cloud-authored pricing UI workflow и policy-id-backed manual runtime adjustments.
- Ручные order discounts/surcharges остаются Edge operational commands и требуют pricing permissions; manual policy exceptions требуют отдельный permission/audit boundary до поддержки.

Modifiers:

- Реализовано сейчас: Cloud modifier group/option/binding data is published and ingested into Edge read model.
- Реализовано сейчас: selected modifiers are accepted in add-line commands, stored on order lines, priced by backend and copied into precheck/check snapshots.
- Реализовано сейчас: cashier UI exposes modifier selection for menu items with modifier groups.
- Запланировано далее: modifier print/reporting/audit polish if pilot acceptance requires it.

Recipes/inventory:

- Запланировано далее: POS Edge работает как генератор events и UI ввода; он не создает `StockDocument`, `StockMove`, stock balance или costing rows.
- Запланировано далее: Edge SQLite содержит only read-only `recipe_versions`, `recipe_lines` and bidirectional `stop_lists`; legacy Edge-side stock tables должны быть удалены из целевого baseline.
- Запланировано далее: Cloud Inventory Worker обрабатывает `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `RefundRecorded`, `CancellationRecorded`, `StopListUpdated`.
- Запланировано далее: Cloud PostgreSQL хранит `stock_ledger` with `unit_cost_minor`, `total_cost_minor` and `costing_status`; ClickHouse получает batch projection `olap_stock_moves`.
- Реализовано сейчас: cancellation/refund ledger хранит явный `inventory_disposition`, но текущий POS runtime не мутирует stock tables.
- Не реализовано сейчас: Cloud Inventory Worker, stop-list sync, recipe expansion, modifier linked catalog item stock consumption, automatic stock return/write-off.
- Профильный целевой contract: `docs/backend/INVENTORY-COSTING-SPEC.md`.

## RBAC

Реализовано сейчас:

- UI visibility is UX only.
- Backend app-layer permissions are authoritative.
- Payment permissions are method-specific:
  - `pos.payment.cash`
  - `pos.payment.card.manual`
  - `pos.payment.other`
- Pricing permissions:
  - `pos.pricing.view`
  - `pos.pricing.discount.apply`
  - `pos.pricing.surcharge.apply`
- Refund uses `pos.payment.refund`.
- Check cancellation ledger uses `pos.precheck.cancel` in current backend service.
- Precheck cancel uses split request/approval:
  - `pos.precheck.cancel.request`
  - `pos.precheck.cancel`
- Reprint permissions:
  - `pos.precheck.reprint`
  - `pos.check.reprint`

## Provisioning

Реализовано сейчас:

- `POST /api/v1/system/provisioning/pair-via-license` принимает одноразовый license pairing code, получает Cloud URL, restaurant id, node token и node device id из License Server.
- Если локальный Edge еще не находится в статусе `paired`, а license response содержит другой `node_device_id`, локальная provisioning identity переключается на `node_device_id` из license response. Это поддерживает Cloud UI flow без ручного ввода пользователем device ID.
- Если локальный Edge уже находится в статусе `paired`, повторный `pair-via-license` и повторный assignment polling являются идемпотентными empty work: backend возвращает текущий paired status без повторного resolve/download/apply/pair и без нового `EdgeNodePaired` local event/outbox row.
- После успешного resolve Edge скачивает Cloud snapshot, применяет master data и переводит provisioning status в `paired`.

## Error And Logging Contract

Реализовано сейчас:

- HTTP panic recovery returns safe JSON error.
- Request audit log records method/path/status/duration and masked IDs.
- Sensitive data such as PINs, tokens and raw payment-sensitive payloads must not be logged.
- Stable permission/error behavior is enforced in backend services; UI must not expose raw Go/SQL errors.
