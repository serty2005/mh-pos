# ROADMAP

Статус документа: актуализировано под фактический код на 2026-05-16.

Roadmap фиксирует статусы, блокеры и следующий план. Архитектурный контракт находится в `SPECv1.3.md`, backend contract — в `docs/backend/POS-BACKEND-SPEC.md`.

## Выполнено

### Cashier Runtime

Выполнено:

- PIN login/session/RBAC foundation.
- Personal employee shifts.
- Cash sessions and cash drawer events.
- Halls/tables read model.
- Menu/catalog read model.
- Order create/read/current/closed.
- Add/change/void order lines.
- Selected modifiers in order lines, modifier price impact in backend pricing, modifier snapshots in precheck/check and cashier modifier selection UI.
- Service catalog items in Cloud -> Edge sync, POS menu read model, separate cashier UI section and order/pricing/precheck/check flow.
- `IssuePrecheck`.
- List/get prechecks.
- Manager override cancel precheck.
- Reprint precheck from immutable snapshot.
- Precheck-based payments through `precheck_id`.
- Partial payments.
- Final check creation after full payment.
- Reprint final check from immutable snapshot.
- Append-only financial operation ledger для full/partial cancellation и full/partial refund: `financial_operations`, `financial_operation_items`, `CancellationRecorded`, `RefundRecorded`.
- Compatibility payment refund route and cashier UI flow: UI вызывает `/payments/{id}/refund`, backend записывает refund operation по captured payment allocation.
- Cashier rich cancellation/refund dialog для закрытого чека: full check cancellation/refund отправляют `command_id`, `operation_kind`, явный `inventory_disposition` и reason; partial scopes показаны как запланированная область без runtime выбора.
- `business_date_local` for shifts, cash sessions, payments, checks and financial operations.
- Pricing/Discounts boundary: backend `Pricing` domain/application layer, line/order discounts, separate surcharge foundation, unified ordered modifier pipeline по `application_index`, tax-last invariant, tax profile/rule foundation, deterministic integer rounding и immutable precheck breakdown persistence.
- Cloud-authored automatic discount/surcharge policies synced through `pricing_policy`; manual discount/surcharge commands remain backend RBAC-controlled operational actions.
- UI error handling hardening для cashier pilot: current employee shift empty state возвращается как `200 null`, остальные optional current empty states отображаются как `null`, payment `409` показывает localized business error и обновляет order/precheck/check/cash-session состояние без auto-retry оплаты, ru locale содержит backend/API error keys.

### Cloud And Sync Foundation

Выполнено:

- Cloud PostgreSQL sync receiver and operational projections foundation.
- Cloud master-data authority foundation in collapsed PostgreSQL baseline `001_init.sql`.
- Cloud schema foundation for roles, employees, catalog items, dishes, goods, semi-finished products, services, recipe items, menu categories, catalog folders, folder parameters, catalog tags, item tags, modifier groups/options/bindings, menu items, menu assignments and versioned publications.
- POS Edge Cloud -> Edge ingest for streams `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`.
- POS Edge Cloud -> Edge ingest for catalog folders/tags/item tags, services, modifier groups/options/menu item links and `pricing_policy` tax/service-charge/automatic discount-surcharge reference rows.
- POS Edge outbox/local event foundation for cashier operational events.
- `CancellationRecorded` and `RefundRecorded` are current Edge -> Cloud financial operation events. `PaymentRefunded` and `CheckRefunded` remain accepted legacy operational event types for older payloads.
- Cloud receiver stores `RefundRecorded` raw/journal rows idempotently and updates event-type stats plus coarse shift finance refund counters; detailed financial operation projection remains separate from cashier runtime.
- DDD context map exists in `docs/architecture/DDD-CONTEXT-MAP.md`.

### Persistence Policy

Выполнено:

- POS Edge SQLite as local OLTP/source of truth.
- Cloud PostgreSQL as Cloud OLTP/source of truth.
- Managed SQL files and startup migration/verification policy.
- ADR-015 accepted for persistence and analytics strategy.

Не выполнено и не должно считаться завершенным:

- `sqlc` rollout как текущий persistence implementation.
- ClickHouse runtime/projection pipeline.

## Только Основа

Эти зоны имеют schema/domain foundation, но не являются готовым pilot runtime:

- Recipes: SQLite `recipe_versions`, `recipe_lines`; Cloud `cloud_recipe_items`.
- Inventory: SQLite `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, purchase receipt foundation.
- Master-data publications: Cloud package/publication foundation пока шире текущего POS Edge runtime для recipes/inventory.

## Аудит 2026-05-15

Реализовано сейчас:

- Документация сверена с фактическими POS Edge routes, Cloud routes, миграционными baseline, sync contracts и текущими Vue UI entry points. Критичных обещаний отсутствующего runtime в профильных документах не найдено.
- Отдельный audit report зафиксирован в `docs/temp/DOCUMENTATION-AUDIT-2026-05-15.md`.
- Отдельный промпт для UI/UX hardening зафиксирован в `docs/temp/UI-UX-FIX-PROMPT-2026-05-15.md`.

Запланировано далее:

- Повторить browser-based UI/UX smoke в окружении с установленным Chromium/Playwright browsers; текущая среда блокирует загрузку браузера proxy/403.

Выполнено:

- POS UI: добавлен primary flow strip `готовность смены -> стол -> заказ -> пречек -> оплата`, secondary operations визуально отделены, blocking states унифицированы, tablet breakpoint пересмотрен так, чтобы checkout/precheck/payment не уходили под active order около 1100px.
- Cloud UI: presentation layer вынесен из монолитного `App.vue` в flow components, launch/readiness checklist стал primary journey, master-data CRUD оставлен secondary/admin layer, добавлен card/list fallback для narrow screens.

## В Работе / До Пилота

Блокеры пилота:

- Pricing/Discounts publication:
  - synced automatic discount/surcharge policies реализованы как backend calculation input;
  - довести Cloud-authored UI workflow и policy-id-backed manual runtime adjustments, если pilot acceptance требует централизованного управления всеми ручными сценариями;
  - уточнить operator policy для manual discount/surcharge permissions в pilot script.
- Modifiers:
  - runtime, pricing, snapshots and cashier UI flow реализованы сейчас;
  - остается pilot acceptance по UX деталям, печатным формам и audit/sync требованиям.
- Recipes/inventory:
  - старая recipe validation была частичной; новая policy запрещает `dish` как компонент и разрешает `good`/`semi_finished`;
  - решить, входит ли automatic consumption в первый pilot;
  - если входит, реализовать consumption trigger, stock document/move service и snapshot requirements.
- Cancellation/refund/reprint hardening:
  - backend ledger, immutable snapshots, no-over-cancel/no-over-refund tests, current `CancellationRecorded`/`RefundRecorded` sync contracts, idempotent Cloud raw/journal receipt checks and coarse Cloud refund projection реализованы;
  - cashier UI pilot-minimum full check cancellation/refund через ledger endpoints реализован с выбором inventory disposition; compatibility refund по captured payment оставлен отдельным fallback;
  - production-way smoke script покрывает Cloud master data для dishes/services/modifiers, Edge ingest, login, shift/cash session, order, modifiers/services, precheck, payment, check reprint, same-shift cancellation, post-shift full refund and shift close.
- Documentation freeze:
  - поддерживать `SPECv1.3.md` как frozen pilot contract;
  - дальние контуры переносить в roadmap/ADR, а не в pilot spec.

## Далее

После закрытия pilot blockers:

- Полный pre-pilot smoke path: поддерживать `scripts/bootstrap-production-way.ps1 -RunRuntimeSmoke` и `scripts/start-and-test-all.ps1` как acceptance smoke для Cloud master data -> Edge ingest -> login -> shift/cash session -> order -> precheck -> payment -> check -> reprint -> cancellation/refund -> close shifts.
- Сверка RBAC matrix с фактическим UI и backend permissions.
- Проверка migration/backup behavior на старой SQLite DB.
- Богатая financial operation ledger projection для отчетности, если raw/journal payload и текущих event counters недостаточно.

## После Пилота

После пилота:

- KDS runtime and kitchen ticket lifecycle.
- DishServed / production triggers.
- Full inventory engine, recipe expansion, semi-finished consumption policies.
- Real PSP/payment processor integrations.
- Fiscal adapter/fiscalization integrations.
- Delivery/channel integrations.
- ClickHouse OLAP/reporting accelerator and PostgreSQL projection pipeline.
- `sqlc` adoption, если после стабилизации схемы это уменьшит риск persistence layer.
- Full accounting/ERP integrations.

## Вне Текущего Объема

Вне текущего объема первого cashier pilot:

- KDS as required runtime dependency.
- Real PSP authorization/capture/refund flow.
- Fiscal device integration.
- Full inventory/procurement engine.
- ClickHouse runtime dependency.
- UI-side authoritative financial calculation.
- Edge-side creation of Cloud-owned master data.

## Definition Of Ready For Cashier Pilot

Готовность к первому cashier pilot означает:

- текущий cashier flow проходит smoke/e2e без ручной правки данных;
- документация не обещает runtime, которого нет в коде;
- pricing/modifiers/inventory либо реализованы и протестированы, либо явно исключены из pilot acceptance;
- backend and UI docs согласованы по refund/reprint/current routes;
- cancellation/refund boundaries явно разделены: cancellation внутри открытой исходной смены/дня, refund после закрытия исходной смены или на следующую business date;
- `sqlc` и ClickHouse описаны только как запланировано далее/после пилота, не как текущий runtime.
