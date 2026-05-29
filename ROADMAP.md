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
- Python 3 local seed runner `scripts/seed-dev-system.py` создает полный Cloud-owned dataset, публикует packages для POS Edge streams, выполняет license pairing и проверяет базовый POS read model; флаг `--run-minimal-flow` выполняет минимальный waiter order -> KDS served -> cashier final check -> `ItemServed`/`CheckClosed` -> Cloud inventory ledger smoke, а `--run-kitchen-process-smoke` выполняет полный kitchen/process smoke без destructive storage actions.
- DDD context map exists in `docs/architecture/DDD-CONTEXT-MAP.md`.

### Persistence Policy

Выполнено:

- POS Edge SQLite as local OLTP/source of truth.
- Cloud PostgreSQL as Cloud OLTP/source of truth.
- Managed SQL files and startup migration/verification policy.
- ADR-015 accepted for persistence and analytics strategy.

Не выполнено и не должно считаться завершенным:

- `sqlc` rollout как текущий persistence implementation.
- Промышленные ClickHouse backfill/operator jobs и OLAP projections шире первого bounded `olap_stock_moves` slice.

## Только Основа

Эти зоны имеют schema/domain foundation, но не являются готовым pilot runtime:

- Recipes: целевая Edge SQLite схема хранит read-only `recipe_versions`, `recipe_lines`; Cloud остается authoring/source.
- Inventory: целевая architecture is Cloud-centric Event-Driven Inventory. Edge-side `stock_documents`, `stock_moves`, `stock_balances`, `item_costs` and purchase receipt foundation использовались как pre-pilot legacy и удалены из целевого SQLite baseline.
- Master-data publications: Cloud deterministic package generation и sync storage уже публикуют `recipes` и `inventory_reference` вместе с базовыми stream-пакетами.
- Stop-list sale blocking foundation: POS Edge уже применяет `recipes`/`inventory_reference` ingest и локально блокирует продажу по active stop-list; smoke `pos_stop_list_sale_blocking` подтверждает Cloud authoring -> publish -> Edge import -> runtime blocking путь.

## Аудит 2026-05-15

Реализовано сейчас:

- Документация частично сверена с фактическими POS Edge routes, Cloud routes, миграционными baseline и единым seed-путем; сверка продолжается по мере выявления расхождений в route lists и формулировках runtime-coverage.
- Результаты сверки фиксируются напрямую в профильных документах и этом roadmap без ссылок на отсутствующие временные отчеты.

Запланировано далее:

- Повторить browser-based UI/UX smoke в окружении с установленным Chromium/Playwright browsers; текущая среда блокирует загрузку браузера proxy/403.

Выполнено:

- POS UI: добавлен primary flow strip `готовность смены -> стол -> заказ -> пречек -> оплата`, secondary operations визуально отделены, blocking states унифицированы, tablet breakpoint пересмотрен так, чтобы checkout/precheck/payment не уходили под active order около 1100px.
- POS UI: верхний cashier context показывает restaurant/actor/node/backend session readiness, а dialog/inline error states показывают безопасный support code без raw backend details.
- POS UI: cashier shell подтвержден как `floor` / `order` / `activity` / `reports` / `cash`; active-looking placeholders для line transfer/split/fractional split, banquet/preorder, mock waiter filters, selected-line placeholder и discount/surcharge editor убраны или переведены в passive/disabled backlog state без backend command.
- POS UI: `PosFloorSection` и `PosMenuGrid` переведены на shared `PosBanner`/`PosEmptyState`/`PosSkeleton`, passive backlog/readiness states переведены на `PosReadinessCard`, waiter mobile viewport `390x844` уплотнен с sticky context/authority dock, lock badge и scrollable modifier dialog без payment/refund/cash drawer authority, `pos-ui-g` kitchen mode переведен на backend-backed runtime с queue/ready order tiles, stock forms, full catalog picker, recipe view и proposals.
- Cloud UI: presentation layer вынесен из монолитного `App.vue` в flow components, launch/readiness checklist стал primary journey с restaurant/staff/floor/catalog/menu/modifiers/pricing/Edge/publication gates, master-data CRUD оставлен secondary/admin layer, добавлен card/list fallback для narrow screens, включая resource status cards и Edge events metadata/checksum без raw payload.

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
  - выполнено: Edge SQLite schema содержит read-only `recipe_versions`, `recipe_lines`, `stop_lists` и `warehouse_reference`;
  - выполнено: Cloud Inventory Worker принимает через durable queue `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted`, `RefundRecorded`, `CancellationRecorded`, `StopListUpdated`;
  - выполнено: Cloud PostgreSQL baseline содержит `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists`;
  - выполнено: worker пишет `stock_ledger` with `unit_cost_minor`, `total_cost_minor`, `costing_status` для нормализованных item payloads; retro recalculation jobs остаются следующим шагом;
  - выполнено: Cloud Inventory Worker дедуплицирует `ItemServed` replay и `CheckClosed` replay, а `CheckClosed` после обработанного `ItemServed` списывает только положительную unserved delta по `order_line_id`;
  - выполнено: `StopListUpdated` обрабатывается async через `inventory_event_queue` в bounded `cloud_projection_stop_list_updates` без raw payload; `stop_list_conflict_policy` поддерживает `cloud_wins`, `edge_overlay_until_next_publication`, `edge_overlay_requires_manager_review`, default `edge_overlay_requires_manager_review`;
  - выполнено: `GET /api/v1/sync/readiness/stop-list` возвращает stop-list publication/package readiness, latest accepted Edge ACK metadata и sync problem counters без raw payload;
  - выполнено: POS Edge пишет `CheckClosed` outbox event из immutable `check.Snapshot` при final check после полной оплаты;
  - выполнено: POS Edge kitchen stock input routes пишут `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured` и `ProductionCompleted` в `local_event_log`/`pos_sync_outbox` без POS-side stock documents/moves/balances/costing; replay того же stock `command_id` возвращает сохраненный результат без повторной записи событий;
  - выполнено: POS Edge использует stop-list как единственный механизм блокировки продаж при add/increase order line; stock balance остается аналитическим и может быть отрицательным;
  - выполнено: минимальный HTTP-only smoke `scripts/seed-dev-system.py --run-minimal-flow` проверяет Cloud recipes/stop-list publication -> Edge sync -> waiter order -> KDS served -> cashier final check -> `ItemServed`/`CheckClosed` -> Cloud `stock_ledger`;
  - выполнено: полный kitchen/process smoke `scripts/seed-dev-system.py --run-kitchen-process-smoke` проверяет Cloud seed publication для catalog/menu/recipes/inventory_reference, Edge sync, waiter order, KDS tile, `accept/start/ready/serve`, `recall/start/ready/serve`, ClickHouse `raw_business_events`, stock receipt/count/write-off/production ledger rows, catalog/recipe suggestions, manager approve и Edge proposal feedback.
- Cancellation/refund/reprint hardening:
  - backend ledger, immutable snapshots, no-over-cancel/no-over-refund/no-over-line-amount tests, current `CancellationRecorded`/`RefundRecorded` sync contracts, idempotent Cloud raw/journal receipt checks, coarse Cloud refund counters and detailed Cloud financial operation projection реализованы;
  - cashier UI full whole-check и partial `order_line`/quantity cancellation/refund через ledger endpoints реализован с выбором inventory disposition; compatibility refund по captured payment оставлен отдельным fallback;
  - выполнено: `scripts/seed-dev-system.py --run-minimal-flow` проверяет минимальный runtime sale path с waiter order/precheck, KDS served, cashier payment/final check, `ItemServed`/`CheckClosed` и Cloud `stock_ledger`; refund/cancellation остаются в профильных backend/UI e2e, а не в seed smoke;
  - выполнено: Playwright `payments-refunds.spec.ts` закрывает исходные personal/cash shifts, открывает новую сменную границу, проверяет refund ledger read после закрытой смены и ожидаемый запрет cancellation после закрытия исходной смены;
  - запланировано далее: PSP refund и fiscal integration.
- Documentation freeze:
  - поддерживать `SPECv1.3.md` как contract текущего cashier runtime и цели полного пилота;
  - дальние контуры переносить в roadmap/ADR, а не документировать как реализованное сейчас.

Расширенные блокеры полного пилота:

- Stop-list sale blocking:
  - выполнено: POS Edge lookup active `stop_lists` для самого блюда и обязательных active recipe components при `AddOrderLine` и увеличении quantity;
  - выполнено: POS Edge ingest streams `recipes` и `inventory_reference`; Cloud generic package validation/storage принимает эти streams;
  - выполнено: POS Edge применяет `warehouses` из `inventory_reference` в `warehouse_reference` и использует default warehouse для kitchen stock command validation;
  - выполнено: Cloud UI имеет bounded authoring для recipe items и stop-list entries по подтвержденным master-data routes;
  - выполнено: минимальный `stop_list_conflict_policy` и safe readiness API/UI signals для stop-list publication, Edge ACK metadata и sync problem counters;
  - добавить полноценный Edge-origin stop-list manager review flow и сценарный recipe version editor/review поверх этих данных;
  - стабилизировать regression-покрытие `pos_stop_list_sale_blocking` для Cloud publish/import контракта и offline blocking-инварианта.
- Advanced KDS/kitchen lifecycle:
  - выполнено: POS Edge создает `kitchen_tickets` из order lines, предоставляет `GET /api/v1/kitchen/order-queue`, `GET /api/v1/kitchen/tickets` и status endpoints `accept/start/hold/ready/serve/recall/cancel`;
  - выполнено: lifecycle `new -> accepted -> in_progress -> ready -> served` поддерживает ветки `hold`, `recall`, `cancelled` и повторный цикл `served -> recall -> start -> ready -> serve`; backend проверяет `pos.kitchen.view` / `pos.kitchen.status.change`;
  - выполнено: status actions пишут `KitchenTicketStatusChanged`, а `serve` дополнительно пишет `ItemServed` в `local_event_log` и `pos_sync_outbox`; replay того же kitchen `command_id` идемпотентен, повторный `serve` новым `command_id` пишет новый `ItemServed` с `serve_sequence` и optional `supersedes_served_event_id`;
  - выполнено: `pos-ui-g` kitchen mode читает backend order queue, показывает queue/ready order tiles, безопасные loading/error/empty/no-permission states и после action перечитывает backend truth без UI-authoritative статусов;
  - выполнено для Cloud worker: принятый `ItemServed` идемпотентно создает SALE ledger по `order_line_id`, последующий `CheckClosed` пишет только unserved delta, а superseded `ItemServed` пропускается, если superseding served fact уже принят в Cloud до обработки очереди;
  - выполнено: Edge-side chef stock input routes для receipt/count/write-off/production валидируют warehouse, catalog item, receipt line totals, counted quantity, write-off reason и semi-finished production recipe, затем пишут outbox events без local stock documents;
  - выполнено: canonical kitchen role получает `pos.catalog.view`, чтобы `pos-ui-g` full catalog picker мог читать `GET /api/v1/catalog/items` без расширения финансовых или cashier полномочий;
  - выполнено: POS Edge recipe/proposal backend routes возвращают техкарту с ingredient names из полного `catalog_items`, сохраняют локальные `kitchen_proposals`, пишут `CatalogItemChangeSuggested`/`RecipeChangeSuggested`, поддерживают `proposal_group_id` для нового блюда + техкарты и валидируют prep time delta через `POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES`;
  - выполнено: Cloud-side `StockWriteOffCaptured` receiver/worker, включая durable processing через `inventory_event_queue`;
  - выполнено: Cloud review/apply для `CatalogItemChangeSuggested`/`RecipeChangeSuggested` с `GET/approve/reject/request-changes`, apply только на approve и публикацией `proposal_feedback` вместе с `catalog`/`recipes`;
  - выполнено: `pos-ui-g` kitchen mode использует backend routes для queue/ready order tiles, ticket actions, stock forms (`receipt/count/write-off/production`), full catalog picker, recipe view и локальные kitchen proposals/suggestions с безопасной локализацией ошибок;
  - добавить kitchen stop-list edit flow; bounded `stop_list_conflict_policy` для Cloud/Edge overlay уже реализован, полноценный review workflow остается далее.
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
  - выполнено: Cloud UI содержит stop-list/recipe authoring и route-backed manager review surfaces для `CatalogItemChangeSuggested`/`RecipeChangeSuggested`: списки catalog/recipe suggestions, detail/diff view, approve/reject/request-changes, linked new dish + recipe group display, safe error handling и publication/readiness signal после approve;
  - Cloud UI должен довести readiness-only surfaces для inventory operations/costing и OLAP exports до runtime только после появления подтвержденных Cloud backend routes;
  - launch readiness учитывает restaurant, staff, floor, catalog, menu, modifiers, pricing, stop-list review, publication и known Edge node.
- Full pilot smoke:
  - выполнено сейчас: минимальный runtime smoke проходит Cloud setup -> seed publication -> Edge sync -> waiter order/precheck -> KDS served -> cashier payment/final check -> Cloud inventory ledger;
  - выполнено сейчас: kitchen/process smoke проверяет KDS recall/serve-again, ClickHouse event trail, Cloud stock ledger и bounded `olap_stock_moves` read для kitchen stock events, proposal approve/feedback;
	  - выполнено: POS syncsender regression покрывает temporary `sync/exchange` failure, retry того же outbox item, item-level ACK и прекращение повторной отправки после ACK;
	  - заблокировано/локализовано в текущей локальной проверке: полный Docker smoke не был подтвержден из-за окружения; `docker-compose.local.yml` поддерживает host-port overrides для `5432`/`8123`/`9000`/`8090`/`8080`/`8095`, а buildx blocker остается требованием Docker CLI/Compose окружения.
- Full Inventory Engine:
  - реализовать stock receipts, inventory counts, production, sale consumption, refund/cancellation stock disposition, recipe expansion, modifier linked consumption, balances и costing state;
  - реализовать retro recalculation DAG для документов задним числом и отрицательных остатков;
  - добавить Cloud UI/API для ручного ввода складских документов и просмотра balances/costing status.
- ClickHouse OLAP:
  - выполнено: ClickHouse добавлен в local Cloud runtime component с managed `raw_business_events`;
  - выполнено: async forwarder `inbox_events -> raw_business_events`, retry state, `processed_for_olap` и checkpoint storage;
  - выполнено: bounded metadata API `GET /api/v1/olap/raw-business-events` без raw payload;
	  - выполнено: первый bounded stock moves slice `stock_ledger -> olap_stock_moves` через async forwarder с checkpoint/retry state и `GET /api/v1/olap/stock-moves` без raw payload;
	  - выполнено: read-only `GET /api/v1/olap/export-status?stream=raw_business_events|stock_moves` для checkpoint/retry counters без raw payload;
	  - выполнено: первый bounded агрегат `GET /api/v1/olap/stock-move-summary` по `olap_stock_moves` с группировкой `business_date|catalog_item|warehouse`;
	  - выполнено: минимальный support-only `POST /api/v1/olap/export-retry` для `retry_failed|resume_from_checkpoint` по `raw_business_events|stock_moves` с идемпотентностью по UUIDv7 `command_id`, без raw payload и без synchronous ClickHouse dual-write;
	  - далее: production-grade backfill jobs/operator UI, sales/kitchen aggregates и costing-dependent COGS/margin после появления достоверной cost basis.

## Далее

После закрытия cashier pilot blockers и перед полным пилотом:

- Полный pre-pilot seed path: поддерживать `scripts/seed-dev-system.py` как единственный Fedora/Linux/Windows-compatible путь заполнения данных; новые Cloud-owned справочники, publication streams и POS read flows добавлять в этот скрипт и документацию тем же PR.
- Расширять `scripts/seed-dev-system.py` и demo seed dataset вместе с новыми Cloud-owned справочниками, publication streams и POS read flows, чтобы ручной наглядный тест не отставал от runtime.
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
- POS Edge применяет `recipes` и `inventory_reference` через managed sync, локально блокирует stop-listed sale offline по локальному `stop_lists` и валидирует kitchen stock commands по `warehouse_reference`;
- waiter mobile UI проходит Playwright mobile flow без payment/refund authority;
- kitchen UI должен проходить Playwright/component flow по backend-backed status lifecycle, `ItemServed`, receipt/count/write-off/production forms и recipe/catalog suggestions; stop-list edit остается следующим сценарием после backend route foundation;
- Cloud worker создает review/proposal записи из `CatalogItemChangeSuggested` и `RecipeChangeSuggested`, а не применяет их без policy/manager review;
- Cloud принимает `CheckClosed`/`ItemServed`, дедуплицирует replay и Cloud Inventory Worker пишет полный stock document/ledger/balance/costing state;
- Cloud Inventory Engine покрывает stock receipt, inventory count, production, sale consumption, refund/cancellation disposition, recipe expansion, modifier linked consumption, negative-balance costing и retro recalculation DAG;
- ClickHouse runtime поднят как обязательный Cloud component: `raw_business_events`, `olap_stock_moves`, async forwarder, retry/export checkpoints, минимальный support-only retry control и bounded OLAP API проходят smoke;
- `scripts/seed-dev-system.py` создает full-pilot seed dataset без ручной правки данных; `--run-kitchen-process-smoke` является текущим профильным smoke для kitchen/process контура.
- все новые routes, payloads, UI flows, RBAC, DB schema, sync events, error keys и seed/e2e paths отражены в профильных docs.

## Pricing/tax pilot readiness

Выполнено:

- Cloud-authored pricing policies доставляются в Edge `pricing_policy` stream с manual/permission/application order metadata.
- POS Edge применяет runtime discounts/surcharges by `pricing_policy_id` и сохраняет policy id в adjustment/precheck breakdown.
- Backend calculation сохраняет ordered discounts/surcharges before tax и Tax Always Last.

Далее:

- Расширить Cloud authoring surface для tax profiles/rules и service charge rules отдельными полноценными CRUD, если pilot restaurant требует редактировать их через Cloud UI до первого запуска.
