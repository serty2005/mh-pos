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
- Service catalog items as sellable POS items.

Не реализовано сейчас:

- `sqlc` как текущий persistence implementation;
- ClickHouse runtime;
- inventory consumption engine;
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
- `GET /api/v1/orders/{id}`
- `GET /api/v1/orders/closed`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
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

## Cashier Flow

Реализовано сейчас:

1. Cashier opens employee shift.
2. Cashier opens cash session for device.
3. Cashier creates order for table.
4. Cashier adds/changes/voids active order lines, including selected modifiers for menu items with modifier groups.
5. Cashier can apply backend-authoritative discount/surcharge commands and read pricing preview.
6. Cashier issues precheck.
7. Backend locks order and creates immutable financial precheck snapshot.
8. Cashier captures one or more payments through `precheck_id`.
9. Backend creates final check only after full payment.
10. Cashier/manager can reprint precheck/check copy from immutable snapshot.
11. Authorized operator can record cancellation/refund operation; current cashier UI uses the compatibility payment refund route for closed orders.

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
- Selected modifiers are priced by backend calculation and persisted in order/precheck/check snapshots.
- Service items use the same order/pricing/precheck/check flow as other sellable menu items and do not imply recipe semantics.

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
- Compatibility refund endpoint is `POST /api/v1/payments/{id}/refund`; it records a refund operation with payment allocation and does not mutate finalized payment/check/precheck totals.
- Cancellation uses permission `pos.precheck.cancel`; refund uses permission `pos.payment.refund`.
- Cancellation requires the original personal shift to be open, the current cash session to belong to that shift and the same `business_date_local`.
- Refund requires the original personal shift to be closed or the current `business_date_local` to differ from the check business date; a current open cash session is still required.
- Financial operations are append-only rows in `financial_operations` and `financial_operation_items`.
- Operation type is `cancellation` or `refund`; operation kind is `full` or `partial`.
- Item scopes are `whole_check`, `order_line`, `modifier_line`, `service_charge`, `tip`, `payment`.
- Backend rejects over-cancel, over-refund, over-line-quantity and over-payment-allocation scenarios.
- Operation snapshot embeds immutable check snapshot and operation items.
- Inventory disposition is explicit: `no_stock_effect`, `return_to_stock`, `write_off_waste`, `manual_review`; financial operation does not mutate stock tables.
- Current events are `CancellationRecorded` and `RefundRecorded`. `PaymentRefunded` and `CheckRefunded` remain accepted legacy sync event types.

Не реализовано сейчас:

- automatic PSP authorization/capture/refund;
- fiscal receipt creation;
- refund manager PIN policy beyond current RBAC permission check;
- cashier UI for line/quantity/modifier/service/tip partial cancellation/refund;
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
- `catalog` применяет catalog folders, folder parameters, catalog tags, item tags and item kinds `dish`, `good`, `semi_finished`, `service`.
- `menu` применяет menu items, menu item `item_type`, modifier groups/options and menu item modifier group links.
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

- Реализована только основа: SQLite recipe and stock tables.
- Не реализовано сейчас: recipe expansion, modifier-to-recipe expansion, automatic stock consumption.
- Реализовано сейчас: cancellation/refund ledger stores explicit `inventory_disposition`, but does not mutate stock tables.
- Не реализовано сейчас: automatic stock return on refund/cancellation.
- Запланировано далее: stock documents/moves app services and consumption policy.

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
- Если локальный Edge уже находится в статусе `paired`, конфликтующий `node_device_id` из license response отклоняется как конфликт состояния.
- После успешного resolve Edge скачивает Cloud snapshot, применяет master data и переводит provisioning status в `paired`.

## Error And Logging Contract

Реализовано сейчас:

- HTTP panic recovery returns safe JSON error.
- Request audit log records method/path/status/duration and masked IDs.
- Sensitive data such as PINs, tokens and raw payment-sensitive payloads must not be logged.
- Stable permission/error behavior is enforced in backend services; UI must not expose raw Go/SQL errors.
