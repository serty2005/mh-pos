# ROADMAP

Статус документа: актуализировано под фактический код и целевую inventory architecture на 2026-05-19.

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
- Selected modifiers in order lines, backend-authoritative required/min/max/active/link validation, modifier edit for active open lines, modifier price impact in backend pricing, modifier snapshots/reprint payloads in precheck/check and cashier modifier selection/edit UI.
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
- Bounded read закрытых заказов: `GET /api/v1/orders/closed` поддерживает безопасный default/max limit, `offset`, фильтры по business date/range, shift, device и check, стабильную сортировку newest-first и SQLite indexes.
- Bounded activity/sync reads: `GET /api/v1/sync/outbox` и `GET /api/v1/sync/local-events` имеют backend default bounded limit, cap oversized requests and are used by POS UI with `limit=5`.
- Основа POS Edge local storage lifecycle: `GET /api/v1/storage/status` и `POST /api/v1/storage/retention/dry-run` дают read-only оценку размера SQLite, объемов closed orders/checks/prechecks/payments/financial operations, business-date окна и outbox blocking state. Retention mode сейчас `dry_run_only`; физическое удаление/архивирование не выполняется.
- Compatibility payment refund route сохранен как fallback: `/payments/{id}/refund` записывает refund operation по captured payment allocation, но не является primary cashier model.
- Cashier rich cancellation/refund dialog для закрытого чека: full whole-check cancellation/refund отправляют `command_id`, `operation_kind`, явный `inventory_disposition` и reason; partial `order_line`/quantity выбирается из immutable check/precheck snapshot и отправляет `items[]`. Modifier/service/tip scopes остаются вне текущего UI flow.
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

- Recipes: целевая Edge SQLite схема хранит read-only `recipe_versions`, `recipe_lines`; Cloud остается authoring/source.
- Inventory: целевая architecture is Cloud-centric Event-Driven Inventory. Edge-side `stock_documents`, `stock_moves`, `stock_balances`, `item_costs` and purchase receipt foundation являются legacy для roadmap и должны быть удалены из целевого baseline.
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
  - runtime, backend validation, active-line edit API/UI, pricing, snapshots, reprint payloads and cashier UI flow реализованы сейчас;
  - modifier-to-recipe expansion, automatic stock consumption and return-to-stock moves вне текущего объема pilot modifier acceptance.
- Recipes/inventory:
  - целевой contract зафиксирован в `docs/backend/INVENTORY-COSTING-SPEC.md`;
  - Edge должен стать только генератором events и UI ввода, без stock documents/moves/balances/costing;
  - целевая Edge SQLite schema: read-only `recipe_versions`, `recipe_lines`, двусторонний `stop_lists`;
  - Cloud Inventory Worker должен обрабатывать `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `RefundRecorded`, `CancellationRecorded`, `StopListUpdated` (запланировано далее);
  - выполнено: Cloud PostgreSQL baseline уже содержит foundation tables `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists`;
  - реализовать `stock_ledger` with `unit_cost_minor`, `total_cost_minor`, `costing_status` and retro recalculation jobs;
  - реализовать stop-list как единственный механизм блокировки продаж; stock balance остается аналитическим и может быть отрицательным.
- Cancellation/refund/reprint hardening:
  - backend ledger, immutable snapshots, no-over-cancel/no-over-refund/no-over-line-amount tests, current `CancellationRecorded`/`RefundRecorded` sync contracts, idempotent Cloud raw/journal receipt checks and coarse Cloud refund projection реализованы;
  - cashier UI full whole-check и partial `order_line`/quantity cancellation/refund через ledger endpoints реализован с выбором inventory disposition; compatibility refund по captured payment оставлен отдельным fallback;
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
- Safe archive/export/apply policy для больших локальных SQLite БД закрытых заказов поверх текущего status/dry-run foundation.

## После Пилота

После пилота:

- KDS runtime and kitchen ticket lifecycle.
- `ItemServed` / `ProductionCompleted` triggers.
- Cloud Inventory Worker, recipe expansion, semi-finished auto-production split policies.
- Stop-list bi-directional sync and Edge local recipe-based stop-list checks.
- Costing Engine with negative balance rules and retro recalculation DAG.
- ClickHouse immutable event store `raw_business_events` на UUIDv7:
  - anti-fraud audit: сохранить trail `ItemAdded`/`ItemRemoved` до финального `CheckClosed`;
  - Speed of Service: считать median/percentiles между `CheckClosed` и `ItemServed`;
  - Data Lake: ABC analysis, cohort analysis и BI без нагрузки на PostgreSQL.
- Real PSP/payment processor integrations.
- Fiscal adapter/fiscalization integrations.
- Delivery/channel integrations.
- ClickHouse `olap_stock_moves` OLAP/reporting accelerator and PostgreSQL projection pipeline.
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
- Synchronous dual-write в PostgreSQL и ClickHouse в request path.

## Definition Of Ready For Cashier Pilot

Готовность к первому cashier pilot означает:

- текущий cashier flow проходит smoke/e2e без ручной правки данных;
- документация не обещает runtime, которого нет в коде;
- pricing/modifiers/inventory либо реализованы и протестированы, либо явно исключены из pilot acceptance;
- backend and UI docs согласованы по refund/reprint/current routes;
- cancellation/refund boundaries явно разделены: cancellation внутри открытой исходной смены/дня, refund после закрытия исходной смены или на следующую business date;
- `sqlc` и ClickHouse описаны только как запланировано далее/после пилота, не как текущий runtime.

## Pricing/tax pilot readiness

Выполнено:

- Cloud-authored pricing policies доставляются в Edge `pricing_policy` stream с manual/permission/application order metadata.
- POS Edge применяет runtime discounts/surcharges by `pricing_policy_id` и сохраняет policy id в adjustment/precheck breakdown.
- Backend calculation сохраняет ordered discounts/surcharges before tax и Tax Always Last.

Далее:

- Расширить Cloud authoring surface для tax profiles/rules и service charge rules отдельными полноценными CRUD, если pilot restaurant требует редактировать их через Cloud UI до первого запуска.
