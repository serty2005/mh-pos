# ROADMAP

Статус документа: актуализировано под фактический код, сводную карту текущего состояния и цель полной пилотной реализации на 30.05.2026.

Этот документ объединяет:
- исходную детальную структуру старого `ROADMAP.md`;
- фактические изменения, закрытые последующими итерациями;
- текущие открытые блокеры полного пилота;
- осторожную маркировку спорных пунктов, которые есть в старом тексте, но не подтверждены текущим кодом.

Архитектурный контракт находится в `SPECv1.3.md`, backend-контракты — в `docs/backend/*`, UI-контракты — в `docs/ui/*`, sync-контракты — в `docs/sync/*`.

Код и тесты являются источником истины для фактически реализованного runtime. Если документация описывает поведение как реализованное, но код/тесты это не подтверждают, документация должна быть исправлена под фактическое состояние, а не наоборот.

## Текущий контур

Текущий runtime остается cashier-first, но к 30.05.2026 пилотный контур уже расширен за счет:

- manager / Cloud UI operations;
- waiter mobile runtime;
- advanced KDS lifecycle;
- Cloud-managed setup;
- Cloud-owned recipes/stop-list/publication;
- Edge-origin proposal/review flows;
- bounded Cloud inventory ledger;
- bounded ClickHouse / OLAP slices;
- KDS kitchen stock input;
- KDS stop-list edit UI;
- Cloud-owned recipe version editor/review.

## Принципы

- POS Edge остается авторитетным для offline-команд заказа, пречека, оплаты, финального чека, финансового журнала, pricing snapshot, idempotency, границ смен/кассы, stop-list sale blocking и KDS command validation.
- POS UI не считает authoritative totals и не принимает финансовые или складские решения.
- POS Edge и KDS являются генераторами неизменяемых business events и интерфейсом ввода складских фактов.
- POS Edge не создает Cloud-owned stock documents, stock moves, stock balances и costing state.
- Cloud является источником истины для master data, stock documents, stock ledger, costing/recalculation state, ClickHouse export и OLAP reads.
- ClickHouse используется как immutable event archive и bounded OLAP слой, но не как transactional source of truth.
- Stop-list блокирует продажу; stock balance является аналитическим показателем и может быть отрицательным.
- Синхронная двойная запись в PostgreSQL и ClickHouse в request path запрещена.

---

# Выполнено

## Cashier Runtime

Выполнено:

- PIN login/session/RBAC foundation.
- Personal employee shifts.
- Cash sessions and cash drawer events.
- Halls/tables read model.
- Menu/catalog read model.
- Service catalog items в Cloud -> Edge sync, POS menu read model, отдельная cashier UI секция и участие service items в order/pricing/precheck/check flow.
- Order create/read/current/closed.
- Add/change/void order lines.
- Selected modifiers in order lines.
- Backend-authoritative required/min/max/active/link validation для modifiers.
- Modifier edit для active open lines.
- Modifier price impact in backend pricing.
- Modifier snapshots/reprint payloads in precheck/check.
- Cashier modifier selection/edit UI.
- `IssuePrecheck`.
- List/get prechecks.
- Manager override cancel precheck.
- Reprint precheck from immutable snapshot.
- Precheck-based payments через `precheck_id`.
- Partial payments.
- Final check creation after full payment.
- Reprint final check from immutable snapshot.
- Append-only financial operation ledger для full/partial cancellation и full/partial refund:
  - `financial_operations`;
  - `financial_operation_items`;
  - `CancellationRecorded`;
  - `RefundRecorded`.
- Bounded read закрытых заказов:
  - `GET /api/v1/orders/closed`;
  - безопасный default/max limit;
  - `offset`;
  - фильтры по business date/range, shift, device и check;
  - стабильная сортировка newest-first;
  - SQLite indexes.
- Bounded read surfaces ledger:
  - `GET /api/v1/checks/{id}/financial-operations?limit=&offset=`;
  - `GET /api/v1/financial-operations?business_date_from=&business_date_to=&operation_type=&shift_id=&original_shift_id=&check_id=&limit=&offset=`.
- Bounded activity/sync reads:
  - `GET /api/v1/sync/outbox`;
  - `GET /api/v1/sync/local-events`;
  - backend default bounded limit;
  - cap oversized requests;
  - POS UI использует `limit=5`.
- Основа POS Edge local storage lifecycle:
  - `GET /api/v1/storage/status`;
  - `POST /api/v1/storage/retention/dry-run`;
  - read-only оценка размера SQLite;
  - объемы closed orders/checks/prechecks/payments/financial operations;
  - business-date окна;
  - active/open blockers;
  - outbox blocking state.
- Archive export-plan:
  - `POST /api/v1/storage/archive/export-plan`;
  - manifest-only plan по правилу `checks.business_date_local < cutoff`;
  - `result_mode = plan_only`;
  - deterministic table manifest;
  - protected flags;
  - active/open blockers;
  - blocking outbox state.
- Export-only archive readiness для closed orders:
  - `POST /api/v1/storage/archive/export`;
  - typed JSONL archive;
  - JSON manifest;
  - exclusive cutoff rule `checks.business_date_local < cutoff`;
  - counts;
  - business-date range;
  - source node/device metadata, если есть в runtime;
  - SHA-256;
  - `runtime_rows_deleted = false`;
  - protected flags;
  - block reasons;
  - source SQLite rows не удаляются и не мутируются.
- Archive verify/read/lookup preview:
  - `POST /api/v1/storage/archive/verify`;
  - `POST /api/v1/storage/archive/read-plan`;
  - `POST /api/v1/storage/archive/lookup`;
  - non-destructive проверка archive manifest;
  - bounded archived closed-order preview;
  - streaming lookup archived check/order по `check_id` или `order_id`;
  - без записи в runtime SQLite.
- Apply-plan/readiness для archive apply:
  - `POST /api/v1/storage/archive/apply-readiness`;
  - `POST /api/v1/storage/archive/apply-plan`;
  - gate по cutoff, manifest version, archive SHA-256, JSONL counts, snapshot payload presence, eligible runtime counts, pending Edge -> Cloud outbox и open operational boundaries;
  - destructive apply удаляет scoped `orders`/`checks`/`prechecks`/`payments`/`financial_operations`/связанные rows и запускает `VACUUM`;
  - при блокировке возвращает `apply_blocked`.
- Compatibility payment refund route сохранен как fallback:
  - `/payments/{id}/refund`;
  - пишет refund operation по captured payment allocation;
  - не является primary cashier model.
- Cashier rich cancellation/refund dialog для закрытого чека:
  - full whole-check cancellation/refund;
  - partial `order_line`/quantity cancellation/refund;
  - `command_id`;
  - `operation_kind`;
  - явный `inventory_disposition`;
  - reason;
  - выбор partial items из immutable check/precheck snapshot.
- Modifier/service/tip scopes остаются вне текущего UI flow.
- `business_date_local` for shifts, cash sessions, payments, checks and financial operations.
- Pricing/Discounts boundary:
  - backend `Pricing` domain/application layer;
  - line/order discounts;
  - separate surcharge foundation;
  - unified ordered modifier pipeline по `application_index`;
  - tax-last invariant;
  - tax profile/rule foundation;
  - deterministic integer rounding;
  - immutable precheck breakdown persistence.
- Cloud-authored automatic discount/surcharge policies synced through `pricing_policy`.
- Manual discount/surcharge commands remain backend RBAC-controlled operational actions.
- UI error handling hardening для cashier pilot:
  - current employee shift empty state возвращается как `200 null`;
  - остальные optional current empty states отображаются как `null`;
  - payment `409` показывает localized business error;
  - order/precheck/check/cash-session состояние обновляется без auto-retry оплаты;
  - ru locale содержит backend/API error keys.

## Cloud And Sync Foundation

Выполнено:

- Cloud PostgreSQL sync receiver and operational projections foundation.
- Cloud master-data authority foundation in PostgreSQL baseline `001_init.sql`.
- Cloud schema foundation для:
  - roles;
  - employees;
  - catalog items;
  - dishes;
  - goods;
  - semi-finished products;
  - services;
  - recipe items;
  - menu categories;
  - catalog folders;
  - folder parameters;
  - catalog tags;
  - item tags;
  - modifier groups/options/bindings;
  - menu items;
  - menu assignments;
  - versioned publications.
- POS Edge Cloud -> Edge ingest for streams:
  - `restaurants`;
  - `devices`;
  - `staff`;
  - `floor`;
  - `catalog`;
  - `menu`;
  - `pricing_policy`;
  - `recipes`;
  - `inventory_reference`.
- POS Edge Cloud -> Edge ingest для:
  - catalog folders;
  - catalog tags;
  - item tags;
  - services;
  - modifier groups/options/menu item links;
  - `pricing_policy` tax/service-charge/automatic discount-surcharge reference rows.
- POS Edge outbox/local event foundation для cashier/KDS/kitchen operational events.
- Current Edge -> Cloud financial operation events:
  - `CancellationRecorded`;
  - `RefundRecorded`.
- Legacy operational event types остаются inbound-compatible:
  - `PaymentRefunded`;
  - `CheckRefunded`.
- Cloud receiver валидирует current `RefundRecorded`/`CancellationRecorded` payload fields:
  - совпадение payload `restaurant_id` / `device_id` с envelope;
  - precheck id;
  - reason.
- Cloud receiver idempotently stores raw/journal rows, updates event-type stats, coarse shift finance refund counters and detailed `cloud_projection_financial_operations`.
- Cloud reporting `GET /api/v1/reporting/financial-operations` и Cloud UI read-only surface поверх `cloud_projection_financial_operations` реализованы сейчас с фильтрами restaurant/date/type/shift/original shift/check, `limit`/`offset` и без raw sync payload.
- Legacy `PaymentRefunded`/`CheckRefunded` принимаются, но не наполняют detailed operation projection.
- `POST /api/v1/sync/edge-events`.
- Batch receive.
- `POST /api/v1/sync/exchange`.
- `sync/exchange` проверяет bearer `node_token`, assigned restaurant и device status.
- POS syncsender regression покрывает:
  - temporary `sync/exchange` failure;
  - retry того же outbox item;
  - item-level ACK;
  - прекращение повторной отправки после ACK.
- Python 3 local seed runner `scripts/seed-dev-system.py`:
  - создает полный Cloud-owned dataset;
  - публикует packages для POS Edge streams;
  - выполняет license pairing;
  - проверяет базовый POS read model;
  - `--run-minimal-flow` выполняет waiter order/precheck -> KDS served -> cashier final check -> `ItemServed`/`CheckClosed` -> Cloud inventory ledger -> ClickHouse/OLAP bounded reads smoke;
  - `--run-kitchen-process-smoke` выполняет полный kitchen/process smoke без destructive storage actions.
- DDD context map exists in `docs/architecture/DDD-CONTEXT-MAP.md`.

## Persistence Policy

Выполнено:

- POS Edge SQLite as local OLTP/source of truth.
- Cloud PostgreSQL as Cloud OLTP/source of truth.
- Managed SQL files and startup migration/verification policy.
- ADR-015 accepted for persistence and analytics strategy.
- ClickHouse добавлен как обязательный Cloud runtime component для bounded analytics slices.
- Cloud UI реализовано сейчас читает bounded inventory/OLAP operator slices: `stock-balances`, `stock-ledger`, OLAP export status, `olap_stock_moves` preview и `stock-move-summary`.

Не выполнено и не должно считаться завершенным:

- `sqlc` rollout как текущий persistence implementation.
- Production auth/RBAC perimeter для mutating ClickHouse backfill/operator controls.
- OLAP projections шире текущих bounded:
  - `raw_business_events`;
  - `olap_stock_moves`;
  - `stock-move-summary`;
  - `sales-kitchen-summary`.

## Только Основа

Эти зоны имеют schema/domain foundation, но не являются завершенным full pilot runtime:

- Recipes:
  - целевая Edge SQLite схема хранит read-only `recipe_versions`, `recipe_lines`;
  - Cloud остается authoring/source.
- Inventory:
  - целевая architecture is Cloud-centric Event-Driven Inventory;
  - Edge-side `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, purchase receipt foundation использовались как pre-pilot legacy и удалены из целевого SQLite baseline.
- Master-data publications:
  - Cloud deterministic package generation и sync storage уже публикуют `recipes` и `inventory_reference` вместе с базовыми stream-пакетами.
- Stop-list sale blocking foundation:
  - POS Edge применяет `recipes` / `inventory_reference` ingest;
  - POS Edge локально блокирует продажу по active stop-list;
  - regression `pos_stop_list_sale_blocking` подтверждает Cloud authoring -> publish -> Edge import -> runtime blocking path.

## Аудит 2026-05-15

Реализовано сейчас:

- Документация частично сверена с фактическими POS Edge routes, Cloud routes, migration baseline и единым seed path.
- Сверка продолжается по мере выявления расхождений в route lists и формулировках runtime coverage.
- Результаты сверки фиксируются напрямую в профильных документах и этом roadmap без ссылок на отсутствующие временные отчеты.

Запланировано далее:

- Повторить browser-based UI/UX smoke в окружении с установленными Chromium/Playwright browsers; прежняя среда блокировала загрузку браузера proxy/403.

Выполнено по UI-аудиту:

- POS UI: добавлен primary flow strip `готовность смены -> стол -> заказ -> пречек -> оплата`.
- POS UI: secondary operations визуально отделены.
- POS UI: blocking states унифицированы.
- POS UI: tablet breakpoint пересмотрен так, чтобы checkout/precheck/payment не уходили под active order около 1100px.
- POS UI: верхний cashier context показывает restaurant/actor/node/backend session readiness.
- POS UI: dialog/inline error states показывают безопасный support code без raw backend details.
- POS UI: cashier shell подтвержден как `floor` / `order` / `activity` / `reports` / `cash`.
- Active-looking placeholders для line transfer/split/fractional split, banquet/preorder, mock waiter filters, selected-line placeholder и discount/surcharge editor убраны или переведены в passive/disabled backlog state без backend command.
- `PosFloorSection` и `PosMenuGrid` переведены на shared `PosBanner`/`PosEmptyState`/`PosSkeleton`.
- Passive backlog/readiness states переведены на `PosReadinessCard`.
- Waiter mobile viewport `390x844` уплотнен с sticky context/authority dock, lock badge и scrollable modifier dialog без payment/refund/cash drawer authority.
- `pos-ui-g` kitchen mode переведен на backend-backed runtime с queue/ready order tiles, stock forms, full catalog picker, recipe view и proposals.
- Cloud UI: presentation layer вынесен из монолитного `App.vue` в flow components.
- Cloud UI: launch/readiness checklist стал primary journey с restaurant/staff/floor/catalog/menu/modifiers/pricing/Edge/publication gates.
- Cloud UI: master-data CRUD оставлен secondary/admin layer.
- Cloud UI: добавлен card/list fallback для narrow screens, включая resource status cards и Edge events metadata/checksum без raw payload.

---

# В работе / До пилота

## Блокеры пилота

### Pricing/Discounts publication

Реализовано сейчас:

- Synced automatic discount/surcharge policies реализованы как backend calculation input.

Запланировано далее:

- Довести Cloud-authored UI workflow и policy-id-backed manual runtime adjustments, если pilot acceptance требует централизованного управления всеми ручными сценариями.
- Уточнить operator policy для manual discount/surcharge permissions в pilot script.

### Modifiers

Реализовано сейчас:

- Runtime.
- Backend validation.
- Active-line edit API/UI.
- Pricing.
- Snapshots.
- Reprint payloads.
- Cashier UI flow.

Вне текущего объема pilot modifier acceptance:

- Modifier-to-recipe expansion.
- Automatic stock consumption.
- Return-to-stock moves.

### Recipes / Inventory

Реализовано сейчас:

- Целевой contract зафиксирован в `docs/backend/INVENTORY-COSTING-SPEC.md`.
- Edge должен быть только генератором events и UI ввода, без stock documents/moves/balances/costing.
- Edge SQLite schema содержит read-only `recipe_versions`, `recipe_lines`, `stop_lists`, `warehouse_reference`.
- Cloud Inventory Worker принимает через durable queue:
  - `CheckClosed`;
  - `ItemServed`;
  - `StockReceiptCaptured`;
  - `InventoryCountCaptured`;
  - `StockWriteOffCaptured`;
  - `ProductionCompleted`;
  - `RefundRecorded`;
  - `CancellationRecorded`;
  - `StopListUpdated`.
- Cloud PostgreSQL baseline содержит:
  - `inventory_event_queue`;
  - `stock_documents`;
  - `stock_ledger`;
  - `stock_recalculation_jobs`;
  - `stop_lists`.
- Worker пишет `stock_ledger` with `unit_cost_minor`, `total_cost_minor`, `costing_status` для нормализованных item payloads.
- Retro recalculation jobs остаются следующим шагом.
- Cloud Inventory Worker дедуплицирует `ItemServed` replay и `CheckClosed` replay.
- `CheckClosed` после обработанного `ItemServed` списывает только положительную unserved delta по `order_line_id`.
- `StopListUpdated` обрабатывается async через `inventory_event_queue` в bounded `cloud_projection_stop_list_updates` без raw payload.
- `stop_list_conflict_policy` поддерживает:
  - `cloud_wins`;
  - `edge_overlay_until_next_publication`;
  - `edge_overlay_requires_manager_review`;
  - default `edge_overlay_requires_manager_review`.
- Cloud manager review для Edge-origin stop-list updates имеет bounded routes:
  - list;
  - detail;
  - approve;
  - reject;
  - request-changes.
- Cloud manager review assignment реализовано сейчас только для Edge-origin `stop_list_update`:
  - `POST /api/v1/manager/stop-list-updates/{id}/assign`;
  - `POST /api/v1/manager/stop-list-updates/{id}/unassign`;
  - UUIDv7 `command_id` idempotency;
  - append-only audit events `assigned` / `unassigned` с `event_id`, `review_id`, `actor_employee_id`, `target_employee_id`, `occurred_at` и safe `reason`;
  - response без raw Edge payload.
- Assignment audit read реализовано сейчас:
  - `GET /api/v1/manager/stop-list-updates/{id}/audit?limit=&offset=`;
  - `GET /api/v1/manager/catalog-suggestions/{id}/audit?limit=&offset=`;
  - `GET /api/v1/manager/recipe-suggestions/{id}/audit?limit=&offset=`;
  - default `limit=50`, max `100`, `offset` non-negative;
  - stable sort `occurred_at DESC, event_id DESC`;
  - unknown review id возвращает safe empty list;
  - response содержит safe audit fields и `command_id`, без raw payload, sync envelope, request dump, token/PIN/SQL details.
- Assignment runtime для catalog/recipe запланирован далее и не заявлен как реализованный.
- Escalation/dashboard запланированы далее.
- Raw payload exposure вне текущего объема и запрещено.
- Cloud UI assignment controls для очереди предложений реализовано сейчас:
  - безопасные assignment metadata в строках и detail view;
  - выбор менеджера из списка сотрудников ресторана;
  - обязательная reason/comment перед assign/unassign;
  - перечитывание queue state после команды;
  - отключение controls для terminal statuses `approved` и `rejected`.
- Stop-list review API отдает safe DTO без raw payload.
- Stop-list review decisions идемпотентны.
- Approve применяет изменения только через Cloud-owned `stop_lists` + publication path.
- `GET /api/v1/sync/readiness/stop-list` возвращает stop-list publication/package readiness, latest accepted Edge ACK metadata и sync problem counters без raw payload.
- POS Edge пишет `CheckClosed` outbox event из immutable `check.Snapshot` при final check после полной оплаты.
- POS Edge kitchen stock input routes пишут `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted` в `local_event_log` / `pos_sync_outbox` без POS-side stock documents/moves/balances/costing.
- Replay того же stock `command_id` возвращает сохраненный результат без повторной записи events.
- POS Edge использует stop-list как единственный механизм блокировки продаж при add/increase order line.
- Stock balance остается аналитическим и может быть отрицательным.
- Минимальный HTTP-only smoke `scripts/seed-dev-system.py --run-minimal-flow` проверяет Cloud recipes/stop-list publication -> Edge sync -> waiter order/precheck -> KDS served -> cashier final check -> `ItemServed`/`CheckClosed` -> Cloud `stock_ledger` -> ClickHouse `raw_business_events`/`olap_stock_moves` -> bounded `stock-move-summary` и `sales-kitchen-summary`.
- Полный kitchen/process smoke `scripts/seed-dev-system.py --run-kitchen-process-smoke` проверяет Cloud seed publication для catalog/menu/recipes/inventory_reference, Edge sync, waiter order, KDS tile, `accept/start/ready/serve`, `recall/start/ready/serve`, ClickHouse `raw_business_events`, stock receipt/count/write-off/production ledger rows, catalog/recipe suggestions, manager approve и Edge proposal feedback.

### Cancellation/refund/reprint hardening

Реализовано сейчас:

- Backend ledger.
- Immutable snapshots.
- No-over-cancel/no-over-refund/no-over-line-amount tests.
- Current `CancellationRecorded` / `RefundRecorded` sync contracts.
- Idempotent Cloud raw/journal receipt checks.
- Coarse Cloud refund counters.
- Detailed Cloud financial operation projection.
- Bounded Cloud reporting API/UI для detailed financial operation projection.
- Cashier UI full whole-check and partial `order_line`/quantity cancellation/refund through ledger endpoints.
- Inventory disposition selection.
- Compatibility refund по captured payment оставлен отдельным fallback.
- `scripts/seed-dev-system.py --run-minimal-flow` проверяет минимальный runtime sale path с waiter order/precheck, KDS served, cashier payment/final check, `ItemServed`/`CheckClosed`, Cloud `stock_ledger`, ClickHouse raw archive, `olap_stock_moves` и bounded OLAP агрегаты.
- Refund/cancellation остаются в профильных backend/UI e2e, а не в seed smoke.
- Playwright `payments-refunds.spec.ts` закрывает исходные personal/cash shifts, открывает новую сменную границу, проверяет refund ledger read после закрытой смены и ожидаемый запрет cancellation после закрытия исходной смены.

Запланировано далее:

- PSP refund.
- Fiscal integration.

### Documentation freeze

Требования:

- Поддерживать `SPECv1.3.md` как contract текущего cashier runtime и цели полного пилота.
- Дальние контуры переносить в roadmap/ADR, а не документировать как реализованное сейчас.

---

# Расширенные блокеры полного пилота

## Stop-list sale blocking

Выполнено:

- POS Edge lookup active `stop_lists` для самого блюда и обязательных active recipe components при `AddOrderLine` и увеличении quantity.
- POS Edge ingest streams `recipes` и `inventory_reference`.
- Cloud generic package validation/storage принимает streams `recipes` и `inventory_reference`.
- POS Edge применяет `warehouses` из `inventory_reference` в `warehouse_reference`.
- POS Edge использует default warehouse для kitchen stock command validation.
- Cloud UI имеет bounded authoring для recipe items, сценарный editor версий техкарт и stop-list entries по подтвержденным master-data routes.
- Минимальный `stop_list_conflict_policy`.
- Bounded Edge-origin stop-list manager review flow.
- Assignment/unassignment для Cloud manager review items реализовано сейчас на backend и подключено в Cloud UI controls для assignment queue; escalation и dashboard workflow вне текущего объема.
- Safe readiness API/UI signals для stop-list publication, Edge ACK metadata и sync problem counters.
- Сценарный Cloud-owned recipe version editor/review:
  - draft versions;
  - submit в `RecipeChangeSuggested`;
  - approve/apply через Cloud authority/publication path;
  - read-only Edge publication.
- POS Edge regression закрепляет, что Cloud-imported active stop-list для компонента активной техкарты не обходится локальным inactive Edge overlay.

Запланировано далее:

- Расширенный stop-list review polish без escalation/dashboard refactor.

## Advanced KDS / Kitchen Lifecycle

Выполнено:

- POS Edge создает `kitchen_tickets` из order lines.
- POS Edge предоставляет:
  - `GET /api/v1/kitchen/order-queue`;
  - `GET /api/v1/kitchen/tickets`;
  - status endpoints `accept/start/hold/ready/serve/recall/cancel`.
- Lifecycle `new -> accepted -> in_progress -> ready -> served` поддерживает ветки:
  - `hold`;
  - `recall`;
  - `cancelled`.
- Повторный цикл `served -> recall -> start -> ready -> serve` поддержан.
- Backend проверяет `pos.kitchen.view` / `pos.kitchen.status.change`.
- Status actions пишут `KitchenTicketStatusChanged`.
- `serve` дополнительно пишет `ItemServed` в `local_event_log` и `pos_sync_outbox`.
- Replay того же kitchen `command_id` идемпотентен.
- Повторный `serve` новым `command_id` пишет новый `ItemServed` с `serve_sequence` и optional `supersedes_served_event_id`.
- `pos-ui-g` kitchen mode читает backend order queue, показывает queue/ready order tiles, безопасные loading/error/empty/no-permission states и после action перечитывает backend truth без UI-authoritative статусов.
- Принятый `ItemServed` в Cloud Worker идемпотентно создает SALE ledger по `order_line_id`.
- Последующий `CheckClosed` пишет только unserved delta.
- Superseded `ItemServed` пропускается, если superseding served fact уже принят Cloud до обработки очереди.
- Если старый `ItemServed` уже обработан до recall/serve-again, superseding `ItemServed` пишет append-only `ItemServedCompensation` `RETURN/IN`, затем новый `SALE/OUT`.
- Edge-side chef stock input routes для receipt/count/write-off/production:
  - валидируют warehouse;
  - валидируют catalog item;
  - валидируют receipt line totals;
  - валидируют counted quantity;
  - валидируют write-off reason;
  - валидируют semi-finished production recipe;
  - пишут outbox events без local stock documents.
- Canonical kitchen role получает `pos.catalog.view`, чтобы `pos-ui-g` full catalog picker мог читать `GET /api/v1/catalog/items` без расширения финансовых или cashier полномочий.
- POS Edge recipe/proposal backend routes:
  - возвращают техкарту с ingredient names из полного `catalog_items`;
  - сохраняют локальные `kitchen_proposals`;
  - пишут `CatalogItemChangeSuggested` / `RecipeChangeSuggested`;
  - поддерживают `proposal_group_id` для нового блюда + техкарты;
  - валидируют prep time delta через `POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES`.
- Cloud-side `StockWriteOffCaptured` receiver/worker, включая durable processing через `inventory_event_queue`.
- Cloud review/apply для `CatalogItemChangeSuggested` / `RecipeChangeSuggested`:
  - list/detail;
  - approve;
  - reject;
  - request-changes;
  - apply только на approve;
  - публикация `proposal_feedback` вместе с `catalog` / `recipes`.
- POS Edge regression закрепляет, что Cloud-imported active stop-list для компонента активной техкарты не обходится локальным inactive Edge overlay.
- `pos-ui-g` kitchen mode использует backend routes для:
  - queue/ready order tiles;
  - ticket actions;
  - stock forms (`receipt/count/write-off/production`);
  - full catalog picker;
  - recipe view;
  - локальные kitchen proposals/suggestions;
  - безопасную локализацию ошибок.
- POS Edge backend route `POST /api/v1/kitchen/stop-list-updates` пишет local overlay и `StopListUpdated` в `local_event_log` / `pos_sync_outbox`, идемпотентно по `command_id`.
- `GET /api/v1/kitchen/stop-list` возвращает safe local overlay + outbox status DTO без raw payload.
- `pos-ui-g` KDS kitchen tab имеет stop-list edit form и pending/ack/problem indicator поверх backend state.

## POS-side authoritative financial/inventory logic

Выполнено:

- POS Edge backend остается авторитетным для:
  - offline order/precheck/payment/check commands;
  - financial operation ledger;
  - pricing snapshot;
  - idempotency;
  - cash/session boundaries;
  - stop-list sale blocking;
  - KDS command validation.
- POS UI не считает authoritative totals и не принимает финансовые/складские решения.
- Cloud остается авторитетным для:
  - master data;
  - stock documents;
  - stock ledger;
  - costing/recalculation state;
  - ClickHouse export;
  - OLAP reads.

## Waiter mobile runtime

Выполнено:

- Route `/pos/waiter` стал mobile-first order/precheck flow по существующим order/menu/floor/precheck contracts.
- Waiter mobile является единственным мобильным layout пилота.
- Cashier/KDS/manager не получают mobile variants.
- Waiter role видит floor/menu/order/precheck actions.
- Waiter role не получает payment/refund/cash drawer controls без payment permissions.
- Playwright mobile viewport spec добавлен для:
  - создания заказа;
  - выбора модификаторов;
  - выпуска precheck;
  - проверки отсутствия payment/refund/cash drawer controls.
- Локальный запуск требует demo bootstrap.

## Manager pilot operations

Выполнено:

- Cloud UI содержит stop-list/recipe authoring.
- Cloud UI содержит route-backed manager review surfaces для:
  - `CatalogItemChangeSuggested`;
  - `RecipeChangeSuggested`;
  - Edge-origin stop-list updates.
- Review surfaces включают:
  - списки;
  - detail/diff view;
  - approve;
  - reject;
  - request-changes;
  - linked new dish + recipe group display для proposal groups;
  - safe error handling;
  - publication/readiness signal after approve.
- Launch readiness учитывает:
  - restaurant;
  - staff;
  - floor;
  - catalog;
  - menu;
  - modifiers;
  - pricing;
  - stop-list review;
  - publication;
  - known Edge node.
- Cloud UI содержит route-backed recipe version editor.
- Cloud UI содержит минимальный read-only preview `sales-kitchen-summary`: фильтры business date from/to и `group_by`, bounded запрос `limit=50&offset=0`, table/card вывод безопасных агрегированных полей без raw payload, графиков, BI dashboard, COGS/margin и retry/backfill controls.

Запланировано далее:

- Runtime surfaces для inventory operations/costing за пределами текущих read-only `stock-balances` и `stock-ledger` после появления подтвержденных Cloud backend routes.
- Изменяющие surfaces для OLAP exports/operator controls только после production-grade backend jobs; текущий Cloud UI остается read-only для export status, stock moves, stock move summary и `sales-kitchen-summary`.

## Full pilot smoke

Выполнено сейчас:

- Минимальный runtime smoke проходит:
  - Cloud setup;
  - seed publication;
  - Edge sync;
  - waiter order/precheck;
  - KDS served;
  - cashier payment/final check;
  - Cloud inventory ledger;
  - ClickHouse raw event archive;
  - bounded `olap_stock_moves`, `stock-move-summary` и `sales-kitchen-summary` reads.
- Kitchen/process smoke проверяет:
  - KDS recall/serve-again;
  - ClickHouse event trail;
  - Cloud stock ledger;
  - bounded `olap_stock_moves` read для kitchen stock events;
  - proposal approve/feedback.
- POS syncsender regression покрывает:
  - temporary `sync/exchange` failure;
  - retry того же outbox item;
  - item-level ACK;
  - прекращение повторной отправки после ACK.

Локализовано по окружению:

- Полный Docker smoke `--run-minimal-flow --run-kitchen-process-smoke` подтвержден в локальной проверке 01.06.2026 при доступном Docker Compose/buildx.
- `docker-compose.local.yml` поддерживает host-port overrides для:
  - `5432`;
  - `8123`;
  - `9000`;
  - `8090`;
  - `8080`;
  - `8095`.
- Buildx blocker остается требованием Docker CLI/Compose окружения.

## Full Inventory Engine

Реализовано сейчас:

- Cloud PostgreSQL baseline содержит inventory schema foundation.
- Worker пишет pilot `stock_ledger` rows with costing fields.
- Bounded Cloud inventory ledger endpoint существует для проверки обработанных worker rows; Cloud UI показывает первые 50 строк как read-only preview без raw payload и складских команд.
- `GET /api/v1/inventory/stock-balances` подтвержден по runtime-коду и тестам как bounded Cloud-owned balance read model поверх PostgreSQL `stock_ledger`; route объявлен в `cloud-backend/internal/cloudsync/api/router.go`, реализован в service/repository слое, покрыт API tests на агрегацию, границы выдачи, фильтр статуса, пустой результат и safe no-raw-payload response, а Cloud UI показывает bounded balances/costing status table.

Запланировано далее:

- Materialized balances.
- Production-grade stock receipts/counts/production state.
- Sale consumption.
- Refund/cancellation stock disposition.
- Recipe expansion.
- Modifier-linked consumption.
- Full costing state.
- Retro recalculation DAG для документов задним числом и отрицательных остатков.
- Cloud UI/API для ручного ввода складских документов.
- Full costing/recalculation status.
- COGS/margin только после появления достоверной cost basis.

Следующая рекомендуемая итерация:

**Cloud Inventory Balances + Costing Status Foundation**

Цель bounded slice:

- использовать уже подтвержденный route `GET /api/v1/inventory/stock-balances` как текущий bounded Cloud-owned inventory balances read endpoint поверх `stock_ledger`;
- показать costing status visibility без COGS/margin;
- не делать full retro recalculation DAG;
- не переносить stock balances/costing authority на POS Edge;
- расширять Cloud UI inventory/costing surface только поверх подтвержденных backend routes; текущая bounded `stock-balances` table уже route-backed, full costing/recalculation operator workflow остается `запланировано далее`.

Ожидаемый минимальный contract:

```text
GET /api/v1/inventory/stock-balances?restaurant_id=&warehouse_id=&catalog_item_id=&business_date_to=&costing_status=&limit=&offset=
GET /api/v1/inventory/stock-ledger?restaurant_id=&source_event_type=&source_event_id=&order_line_id=&catalog_item_id=&limit=&offset=
```

Правила:

- читать из Cloud-owned PostgreSQL state, не из POS Edge;
- не раскрывать raw Edge payload;
- default/max limit;
- stable sort;
- отрицательные остатки разрешены;
- sale blocking не должен использовать stock balance;
- ответ должен показывать quantity, unit, last movement, costing status summary and `needs_recalculation`, если это выводимо из данных.

## ClickHouse OLAP

Выполнено:

- ClickHouse добавлен в local Cloud runtime component с managed `raw_business_events`.
- Async forwarder `inbox_events -> raw_business_events`.
- Retry state.
- `processed_for_olap`.
- Checkpoint storage.
- Bounded metadata API `GET /api/v1/olap/raw-business-events` без raw payload.
- Первый bounded stock moves slice `stock_ledger -> olap_stock_moves`.
- `GET /api/v1/olap/stock-moves` без raw payload.
- Read-only `GET /api/v1/olap/export-status?stream=raw_business_events|stock_moves`.
- Первый bounded агрегат `GET /api/v1/olap/stock-move-summary` по `olap_stock_moves` с группировкой:
  - `business_date`;
  - `catalog_item`;
  - `warehouse`.
- Первый bounded sales/kitchen агрегат `GET /api/v1/olap/sales-kitchen-summary` поверх `raw_business_events` и `olap_stock_moves` с группировкой:
  - `business_date`;
  - `event_type`;
  - `source_event_type`;
  - `catalog_item`.
  Endpoint read-only, bounded, не раскрывает raw payload, не является BI dashboard и не считает COGS/margin.
- Минимальный support-only `POST /api/v1/olap/export-retry`:
  - `retry_failed`;
  - `resume_from_checkpoint`;
  - streams `raw_business_events|stock_moves`;
  - idempotency по UUIDv7 `command_id`;
  - без raw payload;
  - без synchronous ClickHouse dual-write.
- Async backfill jobs foundation:
  - `GET /api/v1/olap/backfill-jobs`;
  - `POST /api/v1/olap/backfill-jobs`;
  - `GET /api/v1/olap/backfill-jobs/{id}`;
  - `POST /api/v1/olap/backfill-jobs/{id}/cancel`;
  - jobs имеют UUIDv7 `command_id`, status/progress/checkpoint/error metadata и audit trail;
  - фактический backfill выполняет background worker без synchronous ClickHouse write в HTTP request path.
- Bounded kitchen timing aggregate `GET /api/v1/olap/kitchen-timing-summary` поверх `KitchenTicketStatusChanged`/`ItemServed` с группировкой `business_date|station`, lifecycle counts и средними transition durations без raw payload.
- Cloud UI реализовано сейчас показывает read-only `export-status`, bounded `stock-moves`, `stock-move-summary`, backfill job status и kitchen timing summary; support-only retry/backfill mutating controls не вызываются из UI.

Далее:

- Production auth/RBAC perimeter для mutating OLAP controls.
- Расширенные sales aggregates beyond current bounded endpoints.
- Costing-dependent COGS/margin после появления достоверной cost basis.

---

# Далее

После закрытия cashier pilot blockers и перед полным пилотом:

- Поддерживать `scripts/seed-dev-system.py` как единственный Fedora/Linux/Windows-compatible путь заполнения данных.
- Новые Cloud-owned справочники, publication streams и POS read flows добавлять в seed script и документацию тем же PR.
- Расширять demo seed dataset вместе с новыми Cloud-owned справочниками, publication streams и POS read flows, чтобы ручной наглядный тест не отставал от runtime.
- Сверить RBAC matrix с фактическим UI и backend permissions.
- Проверить migration/backup behavior на старой SQLite DB.
- Продолжить destructive apply/delete/compaction policy для больших локальных SQLite БД закрытых заказов поверх текущего status/dry-run/manifest-only export-plan/export-only/verify/read-plan/lookup/apply-plan foundation.

---

# После пилота

После полного пилота:

- Hardware bump-bar integrations.
- Kitchen printer orchestration.
- Rich BI dashboards beyond bounded pilot OLAP/KDS metrics.
- Real PSP/payment processor integrations.
- Fiscal adapter/fiscalization integrations.
- Delivery/channel integrations.
- `sqlc` adoption, если после стабилизации схемы это уменьшит риск persistence layer.
- Full accounting/ERP integrations.

## Payments / PSP / Fiscalization

### До полного пилота / pilot-hardening

- Уточнить целевую payment architecture:
  - offline/локальный cashier payment flow остается текущим pilot runtime;
  - real PSP integration не должна ломать existing cash/terminal/manual payment flow;
  - payment status, refund status и fiscal status должны быть разделены в модели.
- Зафиксировать contract для payment provider abstraction:
  - authorization;
  - capture;
  - cancel/void;
  - refund;
  - partial refund;
  - provider reference;
  - idempotency key;
  - retry/error states.
- Зафиксировать contract для fiscalization abstraction:
  - fiscal receipt request;
  - fiscal receipt status;
  - fiscal refund/correction receipt;
  - fiscal device/provider response;
  - retry/error states;
  - связь с immutable check/precheck/payment/refund operation.
- Зафиксировать payment/fiscalization orchestration как policy-driven workflow, а не один жесткий порядок:
  - порядок операций задается fiscal/payment policy для страны, ресторана, провайдера и типа оплаты;
  - offline/локальный cashier payment flow остается текущим pilot runtime;
  - real PSP integration не должна ломать existing cash/terminal/manual payment flow;
  - payment status, refund status и fiscal status должны быть разделены в модели;
  - fiscal document может создаваться до оплаты, после оплаты или как часть provider-specific двухфазного сценария;
  - payment operation должна уметь ссылаться на fiscal document;
  - fiscal document должен уметь ссылаться на payment/refund operation, если платеж уже известен;
  - допускается состояние, где fiscal receipt создан, а payment еще pending/failed;
  - допускается состояние, где payment captured, а fiscalization pending/failed;
  - такие расхождения должны попадать в reconciliation queue / operator review, а не ломать immutable check/payment history.
- Согласовать варианты sequencing по policy:
  - fiscal receipt before payment — например для сценариев, где фискальный чек должен быть создан до или во время приема оплаты;
  - payment before fiscal receipt — например для провайдеров, где сначала подтверждается платеж, затем печатается/регистрируется чек;
  - fiscal receipt and payment in one provider flow — например integrated terminal/fiscal device;
  - refund before fiscal correction receipt;
  - fiscal correction receipt before refund;
  - cancellation before fiscalization;
  - cancellation after fiscalization;
  - shift close boundaries;
  - offline fallback and reconciliation.
- Добавить в документацию явные статусы:
  - `payment_status`;
  - `refund_status`;
  - `fiscal_status`;
  - `fiscal_receipt_id`;
  - `provider_payment_id`;
  - `provider_refund_id`.

### После полного пилота

- Real PSP authorization/capture/refund flow.
- Fiscal adapter/fiscalization integrations.
- Fiscal device integrations.
- Provider-specific terminal integrations.
- Fiscal reporting/export integrations.


---

# Вне текущего объема

Вне текущего объема полного пилота:

- Real PSP authorization/capture/refund flow.
- Fiscal device integration.
- UI-side authoritative financial calculation.
- Edge-side creation of Cloud-owned master data.
- Cashier/KDS/manager mobile variants outside waiter screen.
- Synchronous dual-write в PostgreSQL и ClickHouse в request path.
- POS Edge stock documents/moves/balances/costing.
- COGS/margin до появления достоверной cost basis.
- Full retro recalculation DAG в текущем bounded inventory-balances step.

---

# Definition Of Ready For Cashier Pilot

Готовность к первому cashier pilot означает:

- текущий cashier flow проходит smoke/e2e без ручной правки данных;
- документация не обещает runtime, которого нет в коде;
- pricing/modifiers/inventory либо реализованы и протестированы, либо явно исключены из pilot acceptance;
- backend and UI docs согласованы по refund/reprint/current routes;
- cancellation/refund boundaries явно разделены:
  - cancellation внутри открытой исходной смены/дня;
  - refund после закрытия исходной смены или на следующую business date;
- `sqlc` описан только как запланировано далее/после пилота, не как текущий runtime.

# Definition Of Ready For Full Pilot

Готовность к полному пилоту означает:

- cashier flow из Definition Of Ready For Cashier Pilot остается зеленым;
- Cloud UI позволяет настроить stop-list и recipes, опубликовать их и увидеть readiness Edge;
- POS Edge применяет `recipes` и `inventory_reference` через managed sync;
- POS Edge локально блокирует stop-listed sale offline по локальному `stop_lists`;
- POS Edge валидирует kitchen stock commands по `warehouse_reference`;
- waiter mobile UI проходит Playwright mobile flow без payment/refund authority;
- kitchen UI проходит Playwright/component flow по backend-backed status lifecycle, `ItemServed`, receipt/count/write-off/production forms, recipe/catalog suggestions и stop-list edit UI поверх POS Edge backend routes;
- Cloud worker создает review/proposal записи из `CatalogItemChangeSuggested` и `RecipeChangeSuggested`, а не применяет их без policy/manager review;
- Cloud принимает `CheckClosed`/`ItemServed`, дедуплицирует replay и Cloud Inventory Worker пишет полный stock document/ledger/balance/costing state;
- Cloud Inventory Engine покрывает stock receipt, inventory count, production, sale consumption, refund/cancellation disposition, recipe expansion, modifier-linked consumption, negative-balance costing и retro recalculation DAG;
- ClickHouse runtime поднят как обязательный Cloud component:
  - `raw_business_events`;
  - `olap_stock_moves`;
  - async forwarder;
  - retry/export checkpoints;
  - минимальный support-only retry control;
  - bounded OLAP API.
- `scripts/seed-dev-system.py` создает full-pilot seed dataset без ручной правки данных;
- `--run-kitchen-process-smoke` является текущим профильным smoke для kitchen/process контура;
- все новые routes, payloads, UI flows, RBAC, DB schema, sync events, error keys и seed/e2e paths отражены в профильных docs.

---

# Pricing/tax pilot readiness

Выполнено:

- Cloud-authored pricing policies доставляются в Edge `pricing_policy` stream с manual/permission/application order metadata.
- POS Edge применяет runtime discounts/surcharges by `pricing_policy_id`.
- POS Edge сохраняет policy id в adjustment/precheck breakdown.
- Backend calculation сохраняет ordered discounts/surcharges before tax.
- Tax Always Last сохраняется.

Далее:

- Расширить Cloud authoring surface для tax profiles/rules и service charge rules отдельными полноценными CRUD, если pilot restaurant требует редактировать их через Cloud UI до первого запуска.
