# ROADMAP

Статус документа: актуализировано под фактический код, сводную карту текущего состояния и цель полной пилотной реализации на 2026-05-21.

Roadmap фиксирует статусы, блокеры и следующий план. Архитектурный контракт находится в `SPECv1.3.md`, backend contract — в `docs/backend/POS-BACKEND-SPEC.md`. Текущий runtime является cashier-first; целевой полный пилот добавляет manager, waiter, advanced KDS lifecycle, POS-side authoritative financial/inventory checks, stop-list sale blocking и Cloud-managed setup.

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
- Bounded read surfaces ledger: `GET /api/v1/checks/{id}/financial-operations?limit=&offset=` возвращает append-only operations/items для closed-order detail под `pos.check.view`; `GET /api/v1/financial-operations?business_date_from=&business_date_to=&operation_type=&shift_id=&original_shift_id=&check_id=&limit=&offset=` добавляет backend-owned local reporting filter без мутации finalized документов.
- Bounded activity/sync reads: `GET /api/v1/sync/outbox` и `GET /api/v1/sync/local-events` имеют backend default bounded limit, cap oversized requests and are used by POS UI with `limit=5`.
- Основа POS Edge local storage lifecycle: `GET /api/v1/storage/status` и `POST /api/v1/storage/retention/dry-run` дают read-only оценку размера SQLite, объемов closed orders/checks/prechecks/payments/financial operations, business-date окна, active/open blockers и outbox blocking state. `POST /api/v1/storage/archive/export-plan` возвращает manifest-only plan по `checks.business_date_local < cutoff` с `result_mode = plan_only`, deterministic table manifest, protected flags, active/open blockers и blocking outbox state. Destructive apply поддержан через verified JSONL archive, clean scoped outbox и отсутствие open operational boundaries.
- Export-only archive readiness для closed orders: `POST /api/v1/storage/archive/export` создает typed JSONL archive и JSON manifest по тому же exclusive cutoff rule `checks.business_date_local < cutoff`, с counts, business-date range, source node/device metadata если она есть в runtime, SHA-256, `runtime_rows_deleted = false`, protected flags и block reasons для последующего destructive apply, не удаляя и не мутируя source SQLite rows.
- Archive verify/read/lookup preview для closed orders: `POST /api/v1/storage/archive/verify` non-destructive проверяет archive manifest, version, SHA-256, JSONL counts, required identity fields, business-date range consistency, `runtime_rows_deleted = false`, immutable snapshot payload и отсутствие full payload JSON в `local_event_log`/`pos_sync_outbox` summaries. `POST /api/v1/storage/archive/read-plan` возвращает bounded archived closed-order preview с default `limit=50`, max `limit=100`, `offset`, filters `business_date_local`, `order_id`, `check_id`, флагами `runtime_restored = false`/`runtime_rows_deleted = false` и без sync/event payload JSON. `POST /api/v1/storage/archive/lookup` streaming-способом ищет archived check/order по `check_id` или `order_id` и возвращает только immutable check/precheck snapshot preview и связанные счетчики без записи в runtime SQLite.
- Apply-plan/readiness для archive apply: `POST /api/v1/storage/archive/apply-readiness` агрегирует exclusive cutoff, manifest version, archive SHA-256, JSONL counts, snapshot payload presence, current eligible runtime counts, pending Edge -> Cloud outbox и open operational boundaries; при успешном gate возвращает `ready_for_destructive_apply = true`. `POST /api/v1/storage/archive/apply-plan` при тех же verified/safety условиях выполняет физическое удаление scoped `orders`/`checks`/`prechecks`/`payments`/`financial_operations`/связанных rows и запускает `VACUUM`, возвращая `result_mode = destructive_apply`, `runtime_rows_deleted = true`; иначе возвращает `apply_blocked`.
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
- POS Edge Cloud -> Edge ingest for streams `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`, `recipes`, `inventory_reference`.
- POS Edge Cloud -> Edge ingest for catalog folders/tags/item tags, services, modifier groups/options/menu item links and `pricing_policy` tax/service-charge/automatic discount-surcharge reference rows.
- POS Edge outbox/local event foundation for cashier operational events.
- `CancellationRecorded` and `RefundRecorded` are current Edge -> Cloud financial operation events. `PaymentRefunded` and `CheckRefunded` remain accepted legacy operational event types for older payloads.
- Cloud receiver валидирует current `RefundRecorded`/`CancellationRecorded` payload fields, включая совпадение payload `restaurant_id`/`device_id` с envelope, precheck id и reason, stores raw/journal rows idempotently, updates event-type stats plus coarse shift finance refund counters for refunds and maintains detailed `cloud_projection_financial_operations` for current financial operations. Legacy `PaymentRefunded`/`CheckRefunded` remain inbound-compatible but do not populate the detailed operation projection.
- Python 3 local stack smoke runner: отдельные suites `health`, `license_pairing`, `cloud_to_edge_masterdata`, `pos_cashier_runtime`, `pos_refund_after_shift_close`. Runtime suites проверяют Cloud seed -> POS Edge sync -> PIN login -> personal shift -> cash shift -> hall/table/menu reads -> order -> regular line -> modifier line при наличии seed data -> service line при наличии seed data -> precheck -> payment by `precheck_id` -> final check -> bounded closed orders -> get/reprint check -> cancellation ledger в той же смене -> financial operations read -> storage status, а также отдельный refund после закрытия исходных cash/personal shifts с проверкой ledger и closed-order reads.
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
- Inventory: целевая architecture is Cloud-centric Event-Driven Inventory. Edge-side `stock_documents`, `stock_moves`, `stock_balances`, `item_costs` and purchase receipt foundation использовались как pre-pilot legacy и удалены из целевого SQLite baseline.
- Master-data publications: Cloud deterministic package generation и sync storage уже публикуют `recipes` и `inventory_reference` вместе с базовыми stream-пакетами.
- Stop-list sale blocking foundation: POS Edge уже применяет `recipes`/`inventory_reference` ingest и локально блокирует продажу по active stop-list; smoke `pos_stop_list_sale_blocking` подтверждает Cloud authoring -> publish -> Edge import -> runtime blocking путь.

## Аудит 2026-05-15

Реализовано сейчас:

- Документация частично сверена с фактическими POS Edge routes, Cloud routes, миграционными baseline и smoke-скриптами; сверка продолжается по мере выявления расхождений в route lists и формулировках runtime-coverage.
- Результаты сверки фиксируются напрямую в профильных документах и этом roadmap без ссылок на отсутствующие временные отчеты.

Запланировано далее:

- Повторить browser-based UI/UX smoke в окружении с установленным Chromium/Playwright browsers; текущая среда блокирует загрузку браузера proxy/403.

Выполнено:

- POS UI: добавлен primary flow strip `готовность смены -> стол -> заказ -> пречек -> оплата`, secondary operations визуально отделены, blocking states унифицированы, tablet breakpoint пересмотрен так, чтобы checkout/precheck/payment не уходили под active order около 1100px.
- POS UI: верхний cashier context показывает restaurant/actor/node/backend session readiness, а dialog/inline error states показывают безопасный support code без raw backend details.
- POS UI: cashier shell подтвержден как `floor` / `order` / `activity` / `reports` / `cash`; active-looking placeholders для line transfer/split/fractional split, banquet/preorder, mock waiter filters, selected-line placeholder и discount/surcharge editor убраны или переведены в passive/disabled backlog state без backend command.
- POS UI: `PosFloorSection` и `PosMenuGrid` переведены на shared `PosBanner`/`PosEmptyState`/`PosSkeleton`, waiter mobile viewport `390x844` уплотнен с явной status strip полномочий и без payment/refund/cash drawer authority, `/pos/kitchen` оставлен readiness-only с disabled future action chips и activation gates.
- Cloud UI: presentation layer вынесен из монолитного `App.vue` в flow components, launch/readiness checklist стал primary journey с restaurant/staff/floor/catalog/menu/modifiers/pricing/Edge/publication gates, master-data CRUD оставлен secondary/admin layer, добавлен card/list fallback для narrow screens, включая Edge events metadata/checksum без raw payload.

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
  - выполнено: Edge SQLite schema содержит read-only `recipe_versions`, `recipe_lines` и `stop_lists`;
  - выполнено: Cloud Inventory Worker принимает через durable queue `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `RefundRecorded`, `CancellationRecorded`, `StopListUpdated`;
  - выполнено: Cloud PostgreSQL baseline содержит `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists`;
  - выполнено: worker пишет `stock_ledger` with `unit_cost_minor`, `total_cost_minor`, `costing_status` для нормализованных item payloads; retro recalculation jobs остаются следующим шагом;
  - выполнено: POS Edge пишет `CheckClosed` outbox event из immutable `check.Snapshot` при final check после полной оплаты;
  - выполнено: POS Edge использует stop-list как единственный механизм блокировки продаж при add/increase order line; stock balance остается аналитическим и может быть отрицательным.
- Cancellation/refund/reprint hardening:
  - backend ledger, immutable snapshots, no-over-cancel/no-over-refund/no-over-line-amount tests, current `CancellationRecorded`/`RefundRecorded` sync contracts, idempotent Cloud raw/journal receipt checks, coarse Cloud refund counters and detailed Cloud financial operation projection реализованы;
  - cashier UI full whole-check и partial `order_line`/quantity cancellation/refund через ledger endpoints реализован с выбором inventory disposition; compatibility refund по captured payment оставлен отдельным fallback;
  - выполнено: `scripts/run-stack-smoke.py --suite all` включает `pos_cashier_runtime` для cancellation ledger в той же смене, check reprint, bounded closed orders и storage status sanity check;
  - выполнено: отдельная suite `pos_refund_after_shift_close` закрывает исходные personal/cash shifts, открывает новую cash-session boundary для refund, пишет full refund через `/checks/{id}/refunds` и проверяет ledger/closed-order reads;
  - запланировано далее: PSP refund и fiscal integration.
- Documentation freeze:
  - поддерживать `SPECv1.3.md` как contract текущего cashier runtime и цели полного пилота;
  - дальние контуры переносить в roadmap/ADR, а не документировать как реализованное сейчас.

Расширенные блокеры полного пилота:

- Stop-list sale blocking:
  - выполнено: POS Edge lookup active `stop_lists` для самого блюда и обязательных active recipe components при `AddOrderLine` и увеличении quantity;
  - выполнено: POS Edge ingest streams `recipes` и `inventory_reference`; Cloud generic package validation/storage принимает эти streams;
  - выполнено: Cloud UI имеет bounded authoring для recipe items и stop-list entries по подтвержденным master-data routes;
  - добавить conflict policy, сценарный recipe version editor/review и publication readiness поверх этих данных;
  - стабилизировать regression-покрытие `pos_stop_list_sale_blocking` для Cloud publish/import контракта и offline blocking-инварианта.
- Advanced KDS/kitchen lifecycle:
  - выполнено: `/pos/kitchen` заменен с generic shell на readiness-only экран с contract gaps, `запланировано далее`, disabled future action chips и activation gates без активных lifecycle controls;
  - создать POS Edge kitchen ticket lifecycle `new -> accepted -> in_progress -> ready -> served` с `hold`/`recall`/`cancelled` ветками;
  - route `/pos/kitchen` должен стать рабочим KDS после появления backend endpoints, а не readiness-only screen;
  - status actions пишут `KitchenTicketStatusChanged`, `ItemServed` и при необходимости `ProductionCompleted` в outbox; Cloud принимает events идемпотентно и Cloud Inventory Worker не дублирует расход с `CheckClosed`;
  - добавить chef stock receipt flow: `StockReceiptCaptured` с выбором существующего catalog item или `CatalogItemChangeSuggested` для нового/измененного товара;
  - добавить chef recipe proposal flow: просмотр техкарты и `RecipeChangeSuggested` с заменой ингредиента, количеством/единицей/потерями и prep time delta в пределах `recipe_suggestion_max_time_delta_minutes`;
  - добавить kitchen stop-list edit flow и параметр `stop_list_conflict_policy` для порядка применения Cloud/Edge overlay.
- POS-side authoritative financial/inventory logic:
  - POS Edge backend остается авторитетным для offline order/precheck/payment/check commands, financial operation ledger, pricing snapshot, idempotency, cash/session boundaries, stop-list sale blocking и KDS command validation;
  - POS UI не считает authoritative totals и не принимает финансовые/складские решения;
  - Cloud остается авторитетным для master data, stock documents, stock ledger, costing/recalculation state, ClickHouse export и OLAP reads.
- Waiter mobile runtime:
  - выполнено: route `/pos/waiter` стал mobile-first order/precheck flow по существующим order/menu/floor/precheck contracts;
  - waiter mobile является единственным мобильным layout пилота; cashier/KDS/manager не получают mobile variants;
  - waiter role видит floor/menu/order/precheck actions и не получает payment/refund/cash drawer controls без payment permissions;
  - Playwright mobile viewport spec добавлен для создания заказа, модификаторов, выпуска precheck и отсутствия payment/refund/cash drawer controls; локальный запуск требует demo bootstrap.
- Manager pilot operations:
  - выполнено: Cloud UI содержит stop-list/recipe authoring и readiness-only surfaces для proposal review, inventory operations/costing и OLAP exports без CRUD-муляжа;
  - Cloud UI должен довести readiness-only surfaces до runtime только после появления подтвержденных Cloud backend routes;
  - launch readiness учитывает restaurant, staff, floor, catalog, menu, modifiers, pricing, stop-list review, publication и known Edge node.
- Full pilot smoke:
  - добавить suite `full_pilot`, которая проходит Cloud setup -> publication -> Edge sync -> waiter order -> kitchen served -> cashier payment/final check -> reconnect/outbox ACK -> Cloud inventory ledger -> ClickHouse export -> OLAP API reads.
- Full Inventory Engine:
  - реализовать stock receipts, inventory counts, production, sale consumption, refund/cancellation stock disposition, recipe expansion, modifier linked consumption, balances и costing state;
  - реализовать retro recalculation DAG для документов задним числом и отрицательных остатков;
  - добавить Cloud UI/API для ручного ввода складских документов и просмотра balances/costing status.
- ClickHouse OLAP:
  - поднять ClickHouse как обязательный Cloud runtime component для полного пилота;
  - реализовать async forwarder `inbox_events -> raw_business_events` и projection export `stock_ledger -> olap_stock_moves`;
  - добавить bounded read-only Cloud API для OLAP: event archive, stock moves, COGS/margin, sales aggregates и kitchen timing.

## Далее

После закрытия cashier pilot blockers и перед полным пилотом:

- Полный pre-pilot smoke path: поддерживать `scripts/run-stack-smoke.py` как основной Fedora/Linux/Windows-compatible путь; следующий перенос в Python suites — более богатые negative/permission cases.
- Расширять OpenAPI smoke contract, stack smoke suites и demo seed dataset вместе с новыми Cloud-owned справочниками, publication streams и POS read flows, чтобы ручной наглядный тест не отставал от runtime.
- Сверка RBAC matrix с фактическим UI и backend permissions.
- Проверка migration/backup behavior на старой SQLite DB.
- Публичный Cloud reporting API/UI поверх `cloud_projection_financial_operations`, если пилоту потребуется Cloud-side финансовая отчетность beyond service/repository layer.
- Destructive apply/delete/compaction policy для больших локальных SQLite БД закрытых заказов поверх текущего status/dry-run/manifest-only export-plan/export-only/verify/read-plan/lookup/apply-plan foundation.

## После Пилота

После полного пилота:

- Hardware bump-bar integrations, kitchen printer orchestration и rich BI dashboards beyond bounded pilot OLAP/KDS metrics.
- Real PSP/payment processor integrations.
- Fiscal adapter/fiscalization integrations.
- Delivery/channel integrations.
- `sqlc` adoption, если после стабилизации схемы это уменьшит риск persistence layer.
- Full accounting/ERP integrations.

## Вне Текущего Объема

Вне текущего объема полного пилота:

- Real PSP authorization/capture/refund flow.
- Fiscal device integration.
- UI-side authoritative financial calculation.
- Edge-side creation of Cloud-owned master data.
- Cashier/KDS/manager mobile variants outside waiter screen.
- Synchronous dual-write в PostgreSQL и ClickHouse в request path.

## Definition Of Ready For Cashier Pilot

Готовность к первому cashier pilot означает:

- текущий cashier flow проходит smoke/e2e без ручной правки данных;
- документация не обещает runtime, которого нет в коде;
- pricing/modifiers/inventory либо реализованы и протестированы, либо явно исключены из pilot acceptance;
- backend and UI docs согласованы по refund/reprint/current routes;
- cancellation/refund boundaries явно разделены: cancellation внутри открытой исходной смены/дня, refund после закрытия исходной смены или на следующую business date;
- `sqlc` описан только как запланировано далее/после пилота, не как текущий runtime.

## Definition Of Ready For Full Pilot

Готовность к полному пилоту означает:

- cashier flow из `Definition Of Ready For Cashier Pilot` остается зеленым;
- Cloud UI позволяет настроить stop-list и recipes, опубликовать их и увидеть readiness Edge;
- POS Edge применяет `recipes` и `inventory_reference` через managed sync и локально блокирует stop-listed sale offline по локальному `stop_lists`;
- waiter mobile UI проходит Playwright mobile flow без payment/refund authority;
- kitchen UI сначала проходит readiness spec при отсутствии KDS endpoints; после появления backend routes должен проходить Playwright flow по status lifecycle, `ItemServed`, receipt capture, recipe suggestion и stop-list edit;
- Cloud worker создает review/proposal записи из `CatalogItemChangeSuggested` и `RecipeChangeSuggested`, а не применяет их без policy/manager review;
- Cloud принимает `CheckClosed`/`ItemServed`, дедуплицирует replay и Cloud Inventory Worker пишет полный stock document/ledger/balance/costing state;
- Cloud Inventory Engine покрывает stock receipt, inventory count, production, sale consumption, refund/cancellation disposition, recipe expansion, modifier linked consumption, negative-balance costing и retro recalculation DAG;
- ClickHouse runtime поднят как обязательный Cloud component: `raw_business_events`, `olap_stock_moves`, async forwarder, retry/backfill, export checkpoints и bounded OLAP API проходят smoke;
- `scripts/run-stack-smoke.py --suite full_pilot` проходит без ручной правки данных;
- все новые routes, payloads, UI flows, RBAC, DB schema, sync events, error keys и smoke scripts отражены в профильных docs.

## Pricing/tax pilot readiness

Выполнено:

- Cloud-authored pricing policies доставляются в Edge `pricing_policy` stream с manual/permission/application order metadata.
- POS Edge применяет runtime discounts/surcharges by `pricing_policy_id` и сохраняет policy id в adjustment/precheck breakdown.
- Backend calculation сохраняет ordered discounts/surcharges before tax и Tax Always Last.

Далее:

- Расширить Cloud authoring surface для tax profiles/rules и service charge rules отдельными полноценными CRUD, если pilot restaurant требует редактировать их через Cloud UI до первого запуска.
