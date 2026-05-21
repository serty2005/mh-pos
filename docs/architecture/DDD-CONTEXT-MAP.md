# DDD Context Map

Статус: актуальная context map для frozen cashier pilot.

Статусы:

- `реализовано сейчас`
- `реализована только основа`
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
| `Catalog` | catalog item identity, menu-visible item identity, services, folders, tags, units, SKU, menu publication identity, modifier master data | реализовано сейчас for catalog/menu read model, folders/tags/services and modifier publication/ingest | print/reporting polish for modifiers if accepted | allergens/nutrition/media-heavy catalog | pricing, tax policy, order facts |
| `Pricing` | price policy, discounts, surcharges, manual override policy, rounding inputs for totals | реализовано сейчас: calculation engine, line/order discounts, synced automatic discount/surcharge policies, selected modifier pricing, separate surcharge foundation, unified ordered modifier pipeline by `application_index`, tax-last invariant, integer rounding, pricing preview, `pricing_policy` reference ingest for service-charge rules | policy-id-backed manual runtime adjustments if pilot acceptance requires central policy | advanced promos, dynamic pricing, loyalty coupling | catalog identity, tax/fiscal law mapping, UI authoritative totals |
| `Fiscal / Tax` | tax profiles/policy, fiscal/legal receipt mapping, fiscal adapter boundary | реализовано сейчас: tax profile/rule foundation, inclusive/exclusive percentage/fixed components, precheck tax breakdown, Cloud -> Edge `pricing_policy` ingest for tax profiles/rules | regional/legal policy hardening if pilot needs tax | real fiscalization adapter | PSP/payment processor, order lifecycle |
| `Order` | orders, order lines, selected modifiers, order status, precheck issue intent | реализовано сейчас for `Order -> Precheck` runtime with backend-authoritative selected modifier add/edit validation | service pilot UX hardening if accepted | split/merge/transfer/courses | inventory mutation, payment facts, pricing policy ownership |
| `Precheck` | immutable precheck snapshot, selected modifier snapshot, order lock/cancel lifecycle | реализовано сейчас | print/reporting snapshot polish if accepted | print device orchestration | payment processor and stock movement |
| `Payment` | captured payments, payment attempts, payment methods, provider metadata | реализовано сейчас for manual/trusted `cash/card/other` | PSP boundary decision only if pilot needs it | real PSP integration | fiscalization, order line pricing, inventory, cancellation/refund ledger |
| `Check` | final paid document after full precheck payment, immutable check snapshot | реализовано сейчас | fiscal/tax fields only if policy exists | legal fiscal receipt adapter | PSP authorization and stock consumption |
| `Financial Operations` | append-only cancellation/refund ledger, operation item scopes, inventory disposition, no-over-compensation rules | реализовано сейчас: full/partial cancellation and refund records, bounded POS ledger read endpoints, cashier UI for whole-check and partial `order_line`/quantity actions с явным inventory disposition, `CancellationRecorded`, `RefundRecorded`, Cloud detailed operation projection service/repository | rich partial UI for modifier/service/tip only if pilot acceptance requires it | public Cloud reporting UI/API, richer accounting exports | inventory mutation, PSP refund execution, fiscal correction documents |
| `Inventory` | Cloud-owned stock documents, stock ledger, costing, stop-list authority, inventory worker policy | реализовано сейчас: Cloud-centric Event-Driven Inventory для normalized item payloads; Edge-side manual stock foundation был pre-pilot legacy и удален | recipe expansion, stop-list sync, retro costing | full recipe/costing implementation, ClickHouse OLAP projection | Edge-side stock documents/moves, order/payment/check direct mutation |
| `Production` | KDS tickets, stations, cooking/dish served lifecycle, `ItemServed`, `ProductionCompleted` input events | вне текущего объема | none unless pilot changes | KDS runtime feeding Cloud Inventory Worker | financial close and Cloud ledger writes |
| `CRM` | customer identity/preferences/history | вне текущего объема except `guest_count` | none | full CRM | order/payment lifecycle |
| `Loyalty` | bonuses/coupons/customer promos | вне текущего объема | none | full loyalty engine | backend authoritative totals unless integrated through Pricing |
| `Accounting / Finance / Analytics` | reporting, reconciliation, profit views, immutable business event archive | реализована основа in Cloud projections: event-type stats, coarse shift finance counters and detailed current financial operation projection | ClickHouse `raw_business_events` target contract | ERP/accounting integration, public reporting API/UI, ClickHouse analytics and data lake | runtime capture/check mutation |

## Mandatory Boundaries

- `Catalog` does not own pricing or tax policy.
- `Pricing` does not own catalog identity, recipe composition, order lifecycle or payment facts.
- `Fiscal / Tax` does not own PSP/payment processor state.
- `Payment` and fiscalization are separate boundaries.
- `Order` does not write inventory directly.
- `Inventory` changes only through Cloud Inventory Worker generated stock documents / stock ledger.
- В целевой architecture POS Edge never writes stock documents, stock moves, stock balances or costing rows.
- Balance is an analytic projection and may be negative; it never blocks POS sale.
- `StopList` is the only sale-blocking inventory availability mechanism and syncs Edge <-> Cloud.
- Cancellation/refund records financial compensation and explicit inventory disposition; it does not move stock by itself.
- Finalized checks/payments are not rewritten by compensation flows.
- UI never owns authoritative discount/tax/totals.
- Cloud owns master data authoring; POS Edge uses local read models and ingest.

## Текущая Event/Sync Реальность

Реализовано сейчас:

- Edge cashier operations write local events/outbox for core runtime.
- Cloud -> Edge `mastersync.Service` applies `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`.
- `catalog` sync includes folders, tags, services and modifier groups/options/links; `menu` sync includes menu items. Menu categories remain separate from catalog folders.
- Reprint events use immutable snapshots.
- Cancellation/refund backend flow writes local event/outbox records; `CancellationRecorded` and `RefundRecorded` are current Edge -> Cloud operational events.
- `PaymentRefunded` and `CheckRefunded` remain accepted legacy inbound event types for older payloads; new POS Edge refund runtime emits `RefundRecorded`.
- Просмотр закрытых заказов относится к POS local read model: bounded pagination/filtering реализовано сейчас. Manifest-only export-plan и export-only archive readiness для старых closed orders относятся к operations/data lifecycle и не меняют cashier runtime ownership; destructive retention/apply/delete/compaction остается будущей темой.
- Просмотр financial operations в POS activity detail относится к bounded POS local read model. Cloud detailed projection существует как service/repository read model для current financial operation events, но public Cloud reporting API/UI не реализован сейчас.

Реализована только основа:

- Domain constants mention `recipes` and `inventory_reference`.
- Целевая Edge schema keeps only read-only `recipe_versions`/`recipe_lines` и bidirectional `stop_lists` for inventory availability checks.
- Apply path for `recipes`, `inventory_reference` and `stop_lists` is not implemented in `mastersync.Service`.

## Data Flow

Freezed Principle для immutable event archive:

```text
Edge Outbox
  -> Cloud API (PostgreSQL inbox_events)
  -> Async Batch Forwarder
  -> ClickHouse raw_business_events
```

- Все business events от Edge POS и KDS используют UUIDv7 `event_id`.
- PostgreSQL принимает события в `inbox_events` и остается transactional source of truth для текущего operational state.
- ClickHouse `raw_business_events` является бессрочным archive/source of truth для historical business event trail.
- Request path не делает synchronous dual-write в PostgreSQL и ClickHouse.
- Async Batch Forwarder экспортирует batch от 1 000 до 100 000 rows и после successful export выставляет `processed_for_olap = true`.
- Processed rows старше 3 месяцев можно удалять из PostgreSQL `inbox_events`.

## Pre-Pilot Boundary Decisions

Pricing/Discounts:

- реализовано сейчас: separate backend calculation boundary, unified ordered discount/surcharge pipeline, Tax Always Last, deterministic integer rounding and precheck breakdown persistence;
- реализовано сейчас: Cloud-authored tax/service-charge reference rows and automatic discount/surcharge policies use Edge ingest;
- запланировано далее: policy-id-backed runtime adjustments and manual override boundaries;
- backend authoritative calculation;
- tax profile separate from catalog.

Modifiers:

- реализовано сейчас: Cloud owns modifier master data and publishes modifier groups/options/links;
- реализовано сейчас: POS Edge stores selected modifiers on order lines and immutable precheck/check/reprint snapshots;
- реализовано сейчас: backend pricing includes modifier price, validates group/option/link/count constraints, and cashier UI sends selected modifiers as add/edit backend commands;
- вне текущего объема: modifier-to-recipe expansion, automatic stock consumption and return-to-stock moves.

Recipes/Inventory:

- реализовано сейчас: Edge emits immutable business events, Cloud Inventory Worker computes stock documents and ledger for normalized item payloads;
- реализовано сейчас: `CheckClosed` является финальным batch trigger; KDS `ItemServed` event contract принят Cloud receiver and worker;
- запланировано далее: `ProductionCompleted` creates `PRODUCTION`, and auto-production split expands unavailable semi-finished quantity to raw ingredients;
- реализовано сейчас: `stock_ledger.unit_cost_minor` stores event-time fallback cost; last-known cost and DAG-based retro recalculation запланированы далее;
- UOM остается string-based; separate UOM reference with `code`, `name`, `short_name` and translations запланирована далее только при расширении inventory runtime.

Payment/Fiscal:

- manual/trusted capture is current runtime;
- cancellation/refund ledger is current runtime, PSP refund execution is not;
- payment processor boundary запланирован далее, не реализован;
- fiscal adapter boundary запланирован далее, не реализован.

## Pricing/Fiscal boundary clarification

Статус: реализовано сейчас.

Pricing остается отдельным backend boundary: Catalog и Order не владеют pricing policy. Fiscal/Tax остается отдельным boundary: tax profile/rule reference data приходит из Cloud-owned pricing stream и применяется calculator после всех discounts/surcharges по правилу Tax Always Last.
