# DDD Context Map

Статус: актуальная context map для frozen cashier pilot.

Статусы:

- `реализовано сейчас`
- `foundation only`
- `запланировано до пилота`
- `запланировано далее`
- `после пилота`
- `вне текущего объема`

## Context Ownership

| Context | Owns | Статус сейчас | До пилота | После пилота / вне текущего объема | Не владеет |
| --- | --- | --- | --- | --- | --- |
| `Organization` | restaurant identity, business-day config, devices | реализовано сейчас: restaurants/devices read model and pairing/provisioning foundation | harden provisioning acceptance | multi-tenant admin depth | order/payment facts |
| `Staff / Shift` | employees, roles, auth session actor context, personal shifts, cash sessions, cash drawer events, manager override audit | реализовано сейчас | RBAC matrix final acceptance | HR/timekeeping | payment processor/fiscal state |
| `Floor` | halls, tables | реализовано сейчас: Cloud-owned read model on Edge | table UX polish if needed | reservations | order lifecycle |
| `Catalog` | catalog item identity, menu-visible item identity, units, SKU, menu publication identity | реализовано сейчас for simple catalog/menu read model; foundation only for modifiers/recipes | modifier publication only if accepted | allergens/nutrition/media-heavy catalog | pricing, tax policy, order facts |
| `Pricing` | price policy, discounts, surcharges, manual override policy, rounding inputs for totals | реализовано сейчас: calculation engine, line/order discounts, separate surcharge foundation, unified ordered modifier pipeline by `application_index`, tax-last invariant, integer rounding, pricing preview, `pricing_policy` reference ingest for service-charge rules | policy-id-backed runtime adjustments if pilot acceptance requires central policy | advanced promos, dynamic pricing, loyalty coupling | catalog identity, tax/fiscal law mapping, UI authoritative totals |
| `Fiscal / Tax` | tax profiles/policy, fiscal/legal receipt mapping, fiscal adapter boundary | реализовано сейчас: tax profile/rule foundation, inclusive/exclusive percentage/fixed components, precheck tax breakdown, Cloud -> Edge `pricing_policy` ingest for tax profiles/rules | regional/legal policy hardening if pilot needs tax | real fiscalization adapter | PSP/payment processor, order lifecycle |
| `Order` | orders, order lines, order status, precheck issue intent | реализовано сейчас for `Order -> Precheck` runtime | modifiers/order line snapshot only if accepted | split/merge/transfer/courses | inventory mutation, payment facts, pricing policy ownership |
| `Precheck` | immutable precheck snapshot, order lock/cancel lifecycle | реализовано сейчас | snapshot expansion for modifiers/pricing if accepted | print device orchestration | payment processor and stock movement |
| `Payment` | captured payments, payment attempts, payment methods, refund status | реализовано сейчас for manual/trusted `cash/card/other`, refund and confirmed refund operational sync | richer refund ledger projection if reporting needs it | real PSP integration | fiscalization, order line pricing, inventory |
| `Check` | final paid document after full precheck payment, immutable check snapshot | реализовано сейчас | fiscal/tax fields only if policy exists | legal fiscal receipt adapter | PSP authorization and stock consumption |
| `Inventory` | stock documents, stock moves, stock balances, item costs, consumption policy | foundation only | consumption trigger only if accepted as pilot blocker | full recipe expansion, KDS/DishServed trigger, AVCO/FIFO/batches | order/payment/check direct mutation |
| `Production` | KDS tickets, stations, cooking/dish served lifecycle | вне текущего объема | none unless pilot changes | KDS runtime | financial close and inventory direct writes |
| `CRM` | customer identity/preferences/history | вне текущего объема except `guest_count` | none | full CRM | order/payment lifecycle |
| `Loyalty` | bonuses/coupons/customer promos | вне текущего объема | none | full loyalty engine | backend authoritative totals unless integrated through Pricing |
| `Accounting / Finance` | reporting/reconciliation/profit views | foundation only in Cloud projections | pilot reports if needed | ERP/accounting integration, ClickHouse acceleration | runtime capture/check mutation |

## Mandatory Boundaries

- `Catalog` does not own pricing or tax policy.
- `Pricing` does not own catalog identity, recipe composition, order lifecycle or payment facts.
- `Fiscal / Tax` does not own PSP/payment processor state.
- `Payment` and fiscalization are separate boundaries.
- `Order` does not write inventory directly.
- `Inventory` changes only through stock documents / stock moves.
- UI never owns authoritative discount/tax/totals.
- Cloud owns master data authoring; POS Edge uses local read models and ingest.

## Current Event/Sync Reality

Реализовано сейчас:

- Edge cashier operations write local events/outbox for core runtime.
- Cloud -> Edge `mastersync.Service` applies only `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.
- Reprint events use immutable snapshots.
- Refund backend flow writes local event/outbox records; `PaymentRefunded` and `CheckRefunded` are confirmed Edge -> Cloud operational events accepted by Cloud receiver/projections.

Foundation only:

- Domain constants mention `recipes` and `inventory_reference`.
- SQLite state can store sync state for those streams.
- Apply path for those streams is not implemented in `mastersync.Service`.

## Pre-Pilot Boundary Decisions

Pricing/Discounts:

- реализовано сейчас: separate backend calculation boundary, unified ordered discount/surcharge pipeline, Tax Always Last, deterministic integer rounding and precheck breakdown persistence;
- реализовано сейчас: Cloud-authored tax/service-charge reference rows use Edge ingest only;
- запланировано далее: policy-id-backed runtime adjustments and manual override boundaries;
- backend authoritative calculation;
- tax profile separate from catalog.

Modifiers:

- foundation only today;
- Cloud owns modifier master data;
- selected modifiers must become order/precheck/check snapshot data before feature is called cashier-ready.

Recipes/Inventory:

- foundation only today;
- consumption after final check is recommended pilot policy if no KDS/DishServed trigger is introduced;
- semi-finished fallback expansion requires explicit approved policy.

Payment/Fiscal:

- manual/trusted capture is current runtime;
- payment processor boundary is planned, not implemented;
- fiscal adapter boundary is planned, not implemented.
