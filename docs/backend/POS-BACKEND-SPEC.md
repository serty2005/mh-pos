# POS Backend Spec

Статус: актуальный backend contract для frozen cashier pilot.

Код и тесты являются источником истины. Этот документ не описывает будущие API как реализованные.

## Runtime Modules

Реализовано сейчас:

- POS Edge HTTP API на `chi`.
- SQLite repository/migration/runtime verification.
- Application services для auth, staff shifts, cash sessions, floor/menu/catalog reads, order, pricing, precheck, payment/check, master-data ingest, outbox.
- Manual persistence implementation в infrastructure repositories.

Не реализовано сейчас:

- `sqlc` как текущий persistence implementation;
- ClickHouse runtime;
- POS order modifiers runtime;
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
4. Cashier adds/changes/voids active order lines.
5. Cashier can apply backend-authoritative discount/surcharge commands and read pricing preview.
6. Cashier issues precheck.
7. Backend locks order and creates immutable financial precheck snapshot.
8. Cashier captures one or more payments through `precheck_id`.
9. Backend creates final check only after full payment.
10. Cashier/manager can reprint precheck/check copy from immutable snapshot.
11. Authorized operator can refund captured payment.

## Precheck Contract

Реализовано сейчас:

- `IssuePrecheckCommand` contains `order_id`.
- Precheck can be issued only for `open` order.
- Order device must match command device.
- Only one active issued precheck per order is allowed.
- Snapshot contains active lines, currency, discounts, surcharges, taxes, totals and calculation breakdown at issue time.
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

## Payment, Check And Refund Contract

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
- Reprint check requires `pos.check.reprint`.
- Refund endpoint is `POST /api/v1/payments/{id}/refund`.
- Refund requires `pos.payment.refund`, open cash session and same device/shift/restaurant.
- Refund changes payment status from `captured` to `refunded`.
- Refund decreases precheck `paid_total`; if a check exists, it decreases check `paid_total` and can mark check `refunded`.

Не реализовано сейчас:

- automatic PSP authorization/capture/refund;
- fiscal receipt creation;
- refund manager PIN policy beyond current RBAC permission check;
- full refund ledger model beyond current payment attempt/status behavior.

## Master Data Ingest

Реализовано сейчас:

- `ApplyMasterData` accepts `full_snapshot` and `incremental`.
- `full_snapshot` requires `full_snapshot_reason` of `terminal_restaurant_changed` or `node_role_changed`.
- Backup hook exists for full snapshot when configured.
- Supported POS Edge ingest streams:
  - `restaurants`
  - `devices`
  - `staff`
  - `floor`
  - `catalog`
  - `menu`

Foundation only:

- Domain constants and SQLite state know about `recipes` and `inventory_reference`, but `mastersync.Service` does not apply those streams yet.
- Cloud schema foundation for modifiers/recipes/inventory-adjacent data does not make them supported POS Edge ingest payloads.

## Pricing, Modifiers And Inventory Boundaries

Discounts/taxes:

- Реализовано сейчас: separate `Pricing` policy area, backend authoritative calculation, unified ordered discount/surcharge pipeline, tax-last invariant, immutable precheck breakdown and no UI authoritative totals.
- Запланировано далее: Cloud-authored pricing/tax rule publication into Edge runtime.

Modifiers:

- Foundation only: Cloud modifier group/option tables and menu item assignments.
- Не реализовано сейчас: selected modifiers in order lines, precheck/check snapshots and cashier UI.
- Запланировано до пилота only if accepted: order line snapshot support and backend price impact calculation.

Recipes/inventory:

- Foundation only: SQLite recipe and stock tables.
- Не реализовано сейчас: recipe expansion, modifier-to-recipe expansion, automatic stock consumption.
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
- Precheck cancel uses split request/approval:
  - `pos.precheck.cancel.request`
  - `pos.precheck.cancel`
- Reprint permissions:
  - `pos.precheck.reprint`
  - `pos.check.reprint`

## Error And Logging Contract

Реализовано сейчас:

- HTTP panic recovery returns safe JSON error.
- Request audit log records method/path/status/duration and masked IDs.
- Sensitive data such as PINs, tokens and raw payment-sensitive payloads must not be logged.
- Stable permission/error behavior is enforced in backend services; UI must not expose raw Go/SQL errors.
