# Full Pilot Boundary Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Довести репозиторий от текущего cashier-pilot состояния до полного пилотного контура: кассир, менеджер, официант, advanced KDS lifecycle, POS-side authoritative financial/inventory checks, stop-list sale blocking, Cloud-managed setup, полный Cloud Inventory Engine и ClickHouse OLAP API.

**Architecture:** Существующий cashier runtime остается основой. Новые границы добавляются поверх текущих DDD/context boundaries: Cloud владеет authoring, публикациями master-data, складскими документами, costing и OLAP; POS Edge backend авторитетен для offline order/precheck/payment/check commands, financial ledger, pricing snapshots, stop-list sale blocking and KDS command validation; POS UI не принимает авторитетные финансовые/складские решения. KDS/waiter создают Edge business events, Cloud Inventory Worker обрабатывает складские факты асинхронно.

**Tech Stack:** Go 1.26.2, SQLite managed baseline для POS Edge, PostgreSQL managed baseline для Cloud, Vue 3/TypeScript/Vite, vue-i18n, Playwright, существующий `sync/exchange`.

---

## Анализ Текущего Состояния

Найдено по `docs/temp/deep-research-report (11).md` и сверке с кодом:

- Реализовано сейчас: основной кассовый поток `Order -> Precheck -> Payment -> Check`, partial payment, final check, refund/cancellation ledger, роли/PIN, cash shift, Cloud master-data CRUD, публикация master-data, Edge outbox и `sync/exchange`.
- Реализовано сейчас частично: Cloud Inventory Worker принимает целевые inventory events; POS Edge генерирует `CheckClosed` из immutable check snapshot, но еще не генерирует `KitchenTicketStatusChanged`/`ItemServed` как пилотные KDS факты.
- Только foundation: POS SQLite содержит `recipe_versions`, `recipe_lines`, `stop_lists`, но `AddOrderLine` не проверяет stop-list и `mastersync.Service` не применяет streams `recipes`/`stop_lists`.
- Только route shell: `/pos/waiter`, `/pos/kitchen`, `/pos/manager` ведут на `WorkspaceShellPage.vue` и не реализуют waiter mobile, KDS или manager runtime.
- Cloud UI уже покрывает базовые CRUD, но нет stop-list/recipe manager flow и нет пилотной операционной панели синхронизации stop-list/KDS.

Целевая трактовка пилота в этом плане шире текущего cashier runtime: требуется не только касса, но и waiter mobile, advanced KDS, chef receipt/catalog/recipe proposal flows, stop-list conflict policy, Cloud Inventory Engine and ClickHouse OLAP.

## Файловая Карта

POS Edge backend:

- `pos-backend/internal/pos/app/order/service.go` - локальная блокировка добавления/изменения строк заказа по stop-list и рецепту.
- `pos-backend/internal/pos/app/check/service.go` - генерация `CheckClosed` inventory event при финальном чеке.
- `pos-backend/internal/pos/app/mastersync/service.go` - применение Cloud -> Edge streams `recipes` и `stop_lists`.
- `pos-backend/internal/pos/app/kitchen/` - новый advanced kitchen ticket service.
- `pos-backend/internal/pos/app/kitchen/receipts.go` - приемка поставки и `StockReceiptCaptured`.
- `pos-backend/internal/pos/app/kitchen/proposals.go` - `CatalogItemChangeSuggested` и `RecipeChangeSuggested`.
- `pos-backend/internal/pos/app/stoplist/` - новый Edge manager input для stop-list overlay, если пилот требует локальное управление без Cloud.
- `pos-backend/internal/pos/ports/*.go` - repository contracts для recipe lookup, stop-list lookup, kitchen tickets.
- `pos-backend/internal/pos/infra/sqlite/*.go` - SQLite repository реализация и schema verification.
- `pos-backend/internal/pos/api/router.go` - routes для stop-list reads/mutations и KDS runtime.
- `pos-backend/migrations/sqlite/001_init.sql` - managed baseline для новых ticket/status tables, индексов и version bump через `MH_POS_VERSION`.

Cloud backend:

- `cloud-backend/internal/masterdata/app/service.go` - Cloud authoring stop-list и recipes.
- `cloud-backend/internal/masterdata/api/router.go` - CRUD routes для stop-list/recipes и включение в публикации.
- `cloud-backend/internal/masterdata/infra/postgres/repository.go` - PostgreSQL storage для stop-list/recipe CRUD.
- `cloud-backend/internal/cloudsync/app/service.go` - прием `StopListUpdated` как projection update, не только raw receipt.
- `cloud-backend/internal/masterdata/app/proposals.go` - review/apply flow для catalog и recipe suggestions.
- `cloud-backend/internal/inventory/app/worker.go` - использовать уже существующий worker path для `CheckClosed`/`ItemServed`.
- `cloud-backend/migrations/postgres/001_init.sql` - недостающие columns/indexes для publication-ready recipes/stop-list state.

POS UI:

- `pos-ui/src/router.ts` - заменить shell routes `/pos/waiter` и `/pos/kitchen` на реальные страницы.
- `pos-ui/src/pages/WaiterPage.vue` - новый mobile-first waiter runtime.
- `pos-ui/src/pages/KitchenPage.vue` - новый advanced KDS.
- `pos-ui/src/pages/pos/useCashierTerminal.ts` - переиспользовать shared order flow без payment authority для waiter.
- `pos-ui/src/shared/api.ts` и `pos-ui/src/shared/schemas.ts` - typed clients/schemas для stop-list/KDS events.
- `pos-ui/src/shared/i18n.ts` - все новые UI strings через locale.
- `pos-ui/e2e/*.spec.ts` - pilot flows для waiter/KDS/stop-list/offline.

Cloud UI:

- `cloud-ui/src/shared/api.ts`, `cloud-ui/src/shared/schemas.ts` - typed clients/schemas для recipes/stop-list.
- `cloud-ui/src/App.vue` - добавить ресурсы `recipes` и `stopLists` в текущую resource workspace модель.
- `cloud-ui/src/components/cloud/*` - при необходимости выделить специализированные формы для recipe lines и stop-list.
- `cloud-ui/src/shared/i18n.ts` - все новые manager UI labels.

Документация и приемка:

- `SPECv1.3.md`, `ROADMAP.md`, `docs/backend/POS-BACKEND-SPEC.md`, `docs/backend/CLOUD-BACKEND-SPEC.md`, `docs/backend/INVENTORY-COSTING-SPEC.md`
- `docs/ui/POS-UI-SPEC.md`, `docs/ui/CLOUD-UI-SPEC.md`, `docs/ui/POS-UI-RBAC.md`
- `docs/ui/PILOT-UX-MARKET-REVIEW.md`
- `docs/sync/directional-sync-ownership.md`, `docs/sync/edge-cloud-contracts-v1.md`
- `scripts/run-stack-smoke.py`, `scripts/tests/*`

## Milestone A: Stop-List Safety First

### Task 1: POS Edge Stop-List Repository And Error Contract

**Files:**
- Modify: `pos-backend/internal/pos/domain/inventory/inventory.go`
- Modify: `pos-backend/internal/pos/ports/inventory_repository.go`
- Modify: `pos-backend/internal/pos/infra/sqlite/inventory_repository.go`
- Modify: `pos-backend/internal/pos/infra/sqlite/schema.go`
- Modify: `pos-backend/internal/pos/infra/sqlite/schema_test.go`
- Modify: `docs/backend/POS-ERROR-CATALOG.md`

- [ ] Добавить domain types `StopListEntry`, `SaleBlockReason`, `SaleAvailability`.
- [ ] Добавить repository methods `GetActiveStopListEntry(ctx, restaurantID, catalogItemID)` и `ListActiveRecipeLinesForDish(ctx, dishCatalogItemID)`.
- [ ] Добавить SQLite tests: active stop-list row returns entry; inactive row ignored; missing row returns `domain.ErrNotFound`; active recipe version returns component lines.
- [ ] Добавить stable error code/message key для sale blocking: `SALE_ITEM_STOP_LISTED`, `pos.errors.saleItemStopListed`.
- [ ] Выполнить `cd pos-backend && go test ./internal/pos/infra/sqlite`.

### Task 2: POS Edge Sale Blocking In Order Mutations

**Files:**
- Modify: `pos-backend/internal/pos/app/order/service.go`
- Modify: `pos-backend/internal/pos/app/service_test.go`
- Modify: `pos-backend/internal/pos/api/router_test.go`
- Modify: `pos-ui/src/shared/errorHandling.ts`
- Modify: `pos-ui/src/shared/i18n.ts`

- [ ] Написать failing service test: `AddOrderLine` возвращает conflict, если `menuItem.CatalogItemID` есть в active `stop_lists` с `available_quantity = 0` или `NULL`.
- [ ] Написать failing service test: `AddOrderLine` блокируется, если active recipe component находится в stop-list.
- [ ] Написать failing service test: `ChangeOrderLineQuantity` повторно проверяет stop-list, чтобы нельзя было увеличить уже добавленную позицию после обновления stop-list.
- [ ] Реализовать helper `ensureSaleAllowed(ctx, restaurantID, catalogItemID, quantity)` внутри order service или выделенного app helper без UI-authoritative логики.
- [ ] Вернуть безопасную business error response без raw SQL/Go error.
- [ ] Добавить ru locale key для UI-ошибки stop-list blocking.
- [ ] Выполнить `cd pos-backend && go test ./...`.
- [ ] Выполнить `cd pos-ui && npm run build`.

### Task 3: Cloud Stop-List Authoring API

**Files:**
- Modify: `cloud-backend/internal/masterdata/domain/types.go`
- Modify: `cloud-backend/internal/masterdata/app/service.go`
- Modify: `cloud-backend/internal/masterdata/app/service_test.go`
- Modify: `cloud-backend/internal/masterdata/infra/postgres/repository.go`
- Modify: `cloud-backend/internal/masterdata/api/router.go`
- Modify: `cloud-backend/internal/masterdata/api/router_test.go`
- Modify: `cloud-backend/migrations/postgres/001_init.sql`

- [ ] Добавить Cloud commands: create/update/list stop-list entry by `restaurant_id`, `catalog_item_id`, `available_quantity`, `active`, `reason`.
- [ ] Гарантировать unique `(restaurant_id, catalog_item_id)` и monotonically increasing `cloud_version`.
- [ ] Добавить routes `POST /api/v1/master-data/stop-lists`, `GET /api/v1/master-data/stop-lists`, `PATCH /api/v1/master-data/stop-lists/{id}`.
- [ ] Добавить tests на restaurant boundary, duplicate upsert, archive/inactive behavior и publication version bump.
- [ ] Выполнить `cd cloud-backend && go test ./internal/masterdata/...`.

### Task 4: Recipes And Stop-Lists In Cloud Publications And POS Ingest

**Files:**
- Modify: `cloud-backend/internal/masterdata/app/service.go`
- Modify: `cloud-backend/internal/masterdata/infra/postgres/repository.go`
- Modify: `cloud-backend/internal/masterdata/api/router_test.go`
- Modify: `pos-backend/internal/pos/app/mastersync/service.go`
- Modify: `pos-backend/internal/pos/app/service_test.go`
- Modify: `pos-backend/internal/pos/domain/shared/sync_boundary.go`
- Modify: `docs/sync/edge-cloud-contracts-v1.md`

- [ ] Добавить Cloud package streams `recipes` и `stop_lists` с `cloud_version`, `cloud_updated_at`, `cloud_deleted_at`, `last_synced_at`.
- [ ] Добавить `syncExchangeStreams()` support для новых streams.
- [ ] Добавить `mastersync.Service` apply path для `recipe_versions`, `recipe_lines`, `stop_lists`.
- [ ] Добавить tests: Cloud publish содержит streams; Edge snapshot applies rows; unsupported/malformed package marks stream failed without blocking accepted Edge ACK.
- [ ] Выполнить `cd cloud-backend && go test ./...`.
- [ ] Выполнить `cd pos-backend && go test ./...`.

## Milestone B: Inventory Event Facts

### Task 5: POS Emits CheckClosed Inventory Event

**Files:**
- Modify: `pos-backend/internal/pos/app/check/service.go`
- Modify: `pos-backend/internal/pos/app/service_test.go`
- Modify: `cloud-backend/internal/cloudsync/contracts/types_test.go`
- Modify: `docs/sync/edge-cloud-contracts-v1.md`

- [x] Написать failing POS test: full payment writes `CheckClosed` outbox envelope with `items[]` from immutable check snapshot.
- [x] Ensure `CheckClosed.items[]` includes `order_line_id`, `catalog_item_id`, `quantity`, `unit_code`, `required_for_inventory`.
- [x] Keep existing `CheckCreated`, `PaymentCaptured`, `OrderClosed` events for current financial projections.
- [ ] Verify Cloud receiver accepts replay idempotently and queues one inventory event.
- [x] Выполнить `cd pos-backend && go test ./internal/pos/app`.
- [x] Выполнить `cd cloud-backend && go test ./internal/cloudsync/... ./internal/inventory/...`.

### Task 6: Cloud Inventory Pilot Projection

**Files:**
- Modify: `cloud-backend/internal/inventory/app/worker.go`
- Modify: `cloud-backend/internal/inventory/app/worker_test.go`
- Modify: `cloud-backend/internal/inventory/infra/postgres/repository.go`
- Modify: `docs/backend/INVENTORY-COSTING-SPEC.md`

- [ ] Проверить, что `CheckClosed` и `ItemServed` не создают двойное списание для одной order line.
- [ ] Добавить test: `ItemServed` до `CheckClosed` deduplicates line consumption; `CheckClosed` consumes only unserved lines.
- [ ] Добавить test: `manual_review` inventory disposition у refund/cancellation не создает автоматическое движение.
- [ ] Зафиксировать в docs, что stock balance в пилоте аналитический и может быть отрицательным; sale blocking делает только stop-list.
- [ ] Выполнить `cd cloud-backend && go test ./internal/inventory/...`.

## Milestone C: Advanced Kitchen Runtime

### Task 7: POS Kitchen Ticket Backend

**Files:**
- Create: `pos-backend/internal/pos/app/kitchen/service.go`
- Create: `pos-backend/internal/pos/app/kitchen/receipts.go`
- Create: `pos-backend/internal/pos/app/kitchen/proposals.go`
- Create: `pos-backend/internal/pos/ports/kitchen_repository.go`
- Create: `pos-backend/internal/pos/infra/sqlite/kitchen_repository.go`
- Modify: `pos-backend/migrations/sqlite/001_init.sql`
- Modify: `pos-backend/internal/pos/infra/sqlite/schema.go`
- Modify: `pos-backend/internal/pos/api/router.go`
- Modify: `pos-backend/internal/pos/app/service.go`
- Modify: `pos-backend/internal/pos/app/service_test.go`

- [ ] Добавить tables `kitchen_tickets`, `kitchen_ticket_items` и `kitchen_ticket_status_events` со статусами `new`, `accepted`, `in_progress`, `hold`, `ready`, `served`, `recall`, `cancelled`.
- [ ] Создавать kitchen ticket item при `OrderLineAdded` для sellable dish/good/semi_finished lines, исключая service items.
- [ ] Добавить routes `GET /api/v1/kitchen/tickets`, `POST /api/v1/kitchen/ticket-items/{id}/accept`, `POST /api/v1/kitchen/ticket-items/{id}/start`, `POST /api/v1/kitchen/ticket-items/{id}/hold`, `POST /api/v1/kitchen/ticket-items/{id}/ready`, `POST /api/v1/kitchen/ticket-items/{id}/served`, `POST /api/v1/kitchen/ticket-items/{id}/recall`, `POST /api/v1/kitchen/ticket-items/{id}/cancel`.
- [ ] При каждом status action писать `KitchenTicketStatusChanged`, а при served дополнительно писать `ItemServed` outbox с UUIDv7 `event_id` через существующий `WriteOutbox`.
- [ ] Добавить routes `POST /api/v1/kitchen/receipts`, `POST /api/v1/kitchen/catalog-suggestions`, `GET /api/v1/kitchen/recipes/{catalog_item_id}`, `POST /api/v1/kitchen/recipe-change-suggestions`, `GET /api/v1/kitchen/stop-list`, `PATCH /api/v1/kitchen/stop-list/{catalog_item_id}`.
- [ ] Валидировать `recipe_suggestion_max_time_delta_minutes` и `stop_list_conflict_policy`.
- [ ] Добавить RBAC permissions для kitchen view/update, receipt capture, catalog suggestion, recipe view/suggestion and stop-list update.
- [ ] Выполнить `cd pos-backend && go test ./...`.

### Task 8: POS Kitchen UI

**Files:**
- Create: `pos-ui/src/pages/KitchenPage.vue`
- Modify: `pos-ui/src/router.ts`
- Modify: `pos-ui/src/shared/api.ts`
- Modify: `pos-ui/src/shared/schemas.ts`
- Modify: `pos-ui/src/shared/i18n.ts`
- Modify: `pos-ui/e2e/cashier-terminal-ux.spec.ts`
- Create: `pos-ui/e2e/kitchen-flow.spec.ts`

- [ ] Replace `/pos/kitchen` shell with `KitchenPage.vue`.
- [ ] Show ticket list grouped by station/status with large touch targets, timers, priority and all-day counts.
- [ ] Add accept/start/hold/ready/served/recall/cancel actions; disable them without kitchen permission.
- [ ] Add receipt capture, catalog suggestion, recipe view/suggestion and stop-list edit panels.
- [ ] Show safe localized errors and sync pending indicator.
- [ ] Add Playwright flow: cashier adds dish, kitchen sees ticket, moves through status lifecycle, captures receipt, suggests catalog/recipe change, edits stop-list, outbox contains `KitchenTicketStatusChanged`, `ItemServed`, `StockReceiptCaptured`, `CatalogItemChangeSuggested`, `RecipeChangeSuggested` and `StopListUpdated`.
- [ ] Выполнить `cd pos-ui && npm run build`.
- [ ] Выполнить `cd pos-ui && npx playwright test e2e/kitchen-flow.spec.ts`.

## Milestone D: Waiter Mobile Runtime

### Task 9: Waiter Role Boundary And API Use

**Files:**
- Modify: `pos-backend/internal/pos/app/shared/permission_catalog.go`
- Modify: `pos-backend/internal/pos/app/shared/permission_catalog_test.go`
- Modify: `docs/ui/POS-UI-RBAC.md`
- Modify: `docs/backend/POS-BACKEND-SPEC.md`

- [ ] Зафиксировать waiter permission set: open/view shift, floor/menu/order create/add/change/void, precheck issue/reprint.
- [ ] Explicitly keep payment permissions out of waiter profile unless pilot owner accepts waiter payment scope.
- [ ] Add tests that waiter cannot capture payment/refund/cancel closed check.
- [ ] Выполнить `cd pos-backend && go test ./internal/pos/app/shared ./internal/pos/app`.

### Task 10: Mobile Waiter UI

**Files:**
- Create: `pos-ui/src/pages/WaiterPage.vue`
- Create: `pos-ui/src/pages/pos/useWaiterTerminal.ts`
- Modify: `pos-ui/src/router.ts`
- Modify: `pos-ui/src/shared/api.ts`
- Modify: `pos-ui/src/shared/schemas.ts`
- Modify: `pos-ui/src/shared/i18n.ts`
- Create: `pos-ui/e2e/waiter-mobile-flow.spec.ts`

- [ ] Replace `/pos/waiter` shell with `WaiterPage.vue`.
- [ ] Implement mobile hall/table selection, active order list, menu grid, modifier dialog, quantity change, void line and issue/reprint precheck.
- [ ] Зафиксировать `/pos/waiter` как единственный mobile layout полного пилота; cashier/KDS/manager routes не получают mobile variants.
- [ ] Hide payment/refund/cash drawer controls for waiter role.
- [ ] Ensure all labels/errors go through `vue-i18n`.
- [ ] Add Playwright mobile viewport flow `390x844`: login as waiter, create table order, add modifier line, issue precheck, verify no payment controls.
- [ ] Выполнить `cd pos-ui && npm run build`.
- [ ] Выполнить `cd pos-ui && npx playwright test e2e/waiter-mobile-flow.spec.ts`.

## Milestone E: Manager Pilot Operations

### Task 11: Cloud Recipe Authoring

**Files:**
- Modify: `cloud-backend/internal/masterdata/domain/types.go`
- Modify: `cloud-backend/internal/masterdata/app/service.go`
- Modify: `cloud-backend/internal/masterdata/infra/postgres/repository.go`
- Modify: `cloud-backend/internal/masterdata/api/router.go`
- Modify: `cloud-backend/internal/masterdata/app/service_test.go`
- Modify: `cloud-backend/internal/masterdata/api/router_test.go`

- [ ] Add CRUD for `cloud_recipe_items` by owner catalog item.
- [ ] Validate owner/component restaurant boundary and positive quantity/unit.
- [ ] Include recipe items in publication stream used by POS stop-list checks and Cloud inventory expansion.
- [ ] Выполнить `cd cloud-backend && go test ./internal/masterdata/...`.

### Task 12: Cloud UI Stop-List And Recipes

**Files:**
- Modify: `cloud-ui/src/shared/api.ts`
- Modify: `cloud-ui/src/shared/schemas.ts`
- Modify: `cloud-ui/src/shared/i18n.ts`
- Modify: `cloud-ui/src/App.vue`
- Modify: `cloud-ui/src/components/cloud/ResourceWorkspace.vue`
- Create: `cloud-ui/src/components/cloud/RecipeEditor.vue`
- Create: `cloud-ui/src/components/cloud/StopListPanel.vue`
- Create: `cloud-ui/src/components/cloud/ProposalReviewQueue.vue`
- Modify: `docs/ui/PILOT-UX-MARKET-REVIEW.md`

- [ ] Add typed API clients for stop-list and recipe endpoints.
- [ ] Add resource navigation entries `recipes`, `stopLists`, `catalogSuggestions`, `recipeSuggestions` and `inventoryOperations`.
- [ ] Implement recipe editor as owner catalog item + component rows with unit/quantity/loss.
- [ ] Implement proposal review queue with approve/reject actions, duplicate hints, source event metadata and safe error details.
- [ ] Implement stop-list panel with active toggle, available quantity, reason, and publish status.
- [ ] Implement manager flows without manual UUID/raw JSON entry for normal business operations.
- [ ] Auto-publish or clearly show pending publication after manager changes, matching existing Cloud UI patterns.
- [ ] Выполнить `cd cloud-ui && npm run build`.

### Task 13: Manager Sync And Device Observability

**Files:**
- Modify: `cloud-ui/src/components/cloud/EdgeEventsPanel.vue`
- Modify: `cloud-ui/src/components/cloud/LaunchReadinessPanel.vue`
- Modify: `cloud-backend/internal/cloudsync/api/router.go`
- Modify: `cloud-backend/internal/cloudsync/api/router_test.go`
- Modify: `docs/ui/CLOUD-UI-SPEC.md`

- [ ] Show latest stop-list package version and Edge package ACK status.
- [ ] Show rejected/retryable sync problem events without raw payload.
- [ ] Add launch readiness check: restaurant, staff, floor, menu, pricing, stop-list review, publication, known Edge node.
- [ ] Выполнить `cd cloud-backend && go test ./internal/cloudsync/...`.
- [ ] Выполнить `cd cloud-ui && npm run build`.

## Milestone F: End-To-End Pilot Acceptance

### Task 14: Offline And Sync Smoke Coverage

**Files:**
- Modify: `scripts/run-stack-smoke.py`
- Modify: `scripts/tests/test_mhpos_seed.py`
- Modify: `scripts/tests/test_mhpos_contract.py`
- Modify: `scripts/tests/test_mhpos_stack.py`
- Modify: `docs/temp/FULL-FLOW-SMOKE-2026-05-16.md`

- [ ] Extend seed data with cashier, waiter, kitchen, manager roles; one stopped dish; one stopped component; one active recipe.
- [ ] Add smoke step: Cloud authoring -> publish -> Edge sync -> blocked sale returns safe error.
- [ ] Add smoke step: waiter creates order and precheck offline while sync sender cannot reach Cloud; outbox remains pending.
- [ ] Add smoke step: reconnect Cloud; outbox ACKs `OrderLineAdded`, `PrecheckIssued`, `CheckClosed`, `ItemServed`.
- [ ] Add smoke step: Cloud inventory worker writes ledger rows from `CheckClosed`/`ItemServed`.
- [ ] Выполнить `python scripts/run-stack-smoke.py --suite full_pilot`.

### Task 15: Documentation Alignment

**Files:**
- Modify: `SPECv1.3.md`
- Modify: `ROADMAP.md`
- Modify: `docs/CURRENT-FUNCTIONAL-STATE.md`
- Modify: `docs/backend/POS-BACKEND-SPEC.md`
- Modify: `docs/backend/CLOUD-BACKEND-SPEC.md`
- Modify: `docs/backend/INVENTORY-COSTING-SPEC.md`
- Modify: `docs/ui/POS-UI-SPEC.md`
- Modify: `docs/ui/CLOUD-UI-SPEC.md`
- Modify: `docs/ui/POS-UI-RBAC.md`
- Modify: `docs/sync/directional-sync-ownership.md`
- Modify: `docs/sync/edge-cloud-contracts-v1.md`

- [ ] Replace outdated “вне текущего cashier pilot” wording where full pilot acceptance now includes waiter/KDS/stop-list.
- [ ] Keep statuses in Russian: `реализовано сейчас`, `запланировано далее`, `вне текущего объема`.
- [ ] Document unsupported post-pilot scope: fiscalization, PSP integration, delivery, ERP/accounting integrations, hardware bump-bar/printer integrations and rich BI dashboards.
- [ ] Run profile searches from `AGENTS.md` and fix ordinary English status text in docs.

### Task 16: Release Gate

**Files:**
- Modify: `ROADMAP.md`
- Modify: `docs/CURRENT-FUNCTIONAL-STATE.md`

- [ ] Execute backend verification:
  - `cd pos-backend && go mod tidy && go test ./...`
  - `cd cloud-backend && go mod tidy && go test ./...`
- [ ] Execute frontend verification:
  - `cd pos-ui && npm install && npm run build`
  - `cd cloud-ui && npm install && npm run build`
- [ ] Execute full smoke:
  - `python scripts/run-stack-smoke.py --suite full_pilot`
- [ ] Record known accepted pilot limitations in `ROADMAP.md`.
- [ ] Mark runtime code touched: yes, across POS Edge, Cloud Backend, POS UI and Cloud UI.

## Зависимости И Порядок

1. Stop-list safety must land before waiter/KDS go live. Иначе новый UI расширит риск продажи запрещенных позиций.
2. `recipes`/`stop_lists` publication and Edge ingest must land before recipe-based sale blocking can pass smoke.
3. `CheckClosed`/`ItemServed` events can be developed after stop-list blocking, because Cloud worker foundation already exists.
4. Waiter UI can reuse existing POS order APIs but must wait for RBAC boundary confirmation.
5. Kitchen UI depends on backend kitchen ticket routes.
6. Cloud UI manager polish follows backend contracts; no hardcoded Russian UI strings outside locale files.

## Строгая Последовательность Исполнения

Каждый шаг закрывается локальной проверкой до перехода к следующему шагу. Если проверка не проходит, следующий шаг не начинается.

1. **Documentation baseline**
   - Files: `SPECv1.3.md`, `ROADMAP.md`, `docs/CURRENT-FUNCTIONAL-STATE.md`, `docs/backend/*`, `docs/ui/*`, `docs/sync/*`.
   - Implement: зафиксировать полный pilot target как `запланировано далее`, не меняя `реализовано сейчас` для отсутствующего runtime.
   - Local check: `rg "полного пилота|до полного пилота|full_pilot|stop-list sale blocking|waiter mobile|advanced KDS|RecipeChangeSuggested|CatalogItemChangeSuggested" SPECv1.3.md ROADMAP.md docs`.
   - Local check: `rg "implemented now|planned next|out of scope|Current status|Business rules|Architecture decisions|Pilot blockers|Context owns|Remaining risks" AGENTS.md SPECv1.3.md ROADMAP.md docs`.

2. **POS Edge stop-list repository**
   - Files: `pos-backend/internal/pos/domain/inventory/inventory.go`, `pos-backend/internal/pos/ports/inventory_repository.go`, `pos-backend/internal/pos/infra/sqlite/inventory_repository.go`, `pos-backend/internal/pos/infra/sqlite/schema_test.go`.
   - Implement: `StopListEntry`, active stop-list lookup, active recipe component lookup.
   - Local check: `cd pos-backend && go test ./internal/pos/infra/sqlite`.
   - Functional proof: tests show active stop-list row blocks lookup, inactive row ignored, missing row returns `ErrNotFound`, active recipe returns component lines.

3. **POS Edge sale blocking**
   - Files: `pos-backend/internal/pos/app/order/service.go`, `pos-backend/internal/pos/app/service_test.go`, `pos-backend/internal/pos/api/router_test.go`, `pos-ui/src/shared/i18n.ts`.
   - Implement: block `AddOrderLine` and quantity increase when dish or mandatory recipe component is active in stop-list; return stable safe error key.
   - Local check: `cd pos-backend && go test ./internal/pos/app ./internal/pos/api`.
   - Functional proof: rejected sale writes no order line, no local event and no outbox row.

4. **Cloud stop-list API**
   - Files: `cloud-backend/internal/masterdata/*`, `cloud-backend/migrations/postgres/001_init.sql`.
   - Implement: create/list/update stop-list entries with restaurant boundary and cloud versioning.
   - Local check: `cd cloud-backend && go test ./internal/masterdata/...`.
   - Functional proof: duplicate `(restaurant_id,catalog_item_id)` updates existing logical entry and publication version changes.

5. **Recipes and stop-lists publication/ingest**
   - Files: `cloud-backend/internal/masterdata/*`, `pos-backend/internal/pos/app/mastersync/service.go`, `docs/sync/*`.
   - Implement: streams `recipes` and `stop_lists`, Edge apply and checkpoint updates.
   - Local check: `cd cloud-backend && go test ./...`.
   - Local check: `cd pos-backend && go test ./...`.
   - Functional proof: Cloud package applies to Edge recipe/stop-list tables; malformed package marks stream failed without blocking accepted Edge ACK.

6. **CheckClosed inventory fact**
   - Files: `pos-backend/internal/pos/app/check/service.go`, `pos-backend/internal/pos/app/service_test.go`, `cloud-backend/internal/cloudsync/*`.
   - Done: final check writes `CheckClosed` outbox envelope from immutable `check.Snapshot` in addition to current financial events.
   - Done: `cd pos-backend && go test ./internal/pos/app`.
   - Done: `cd cloud-backend && go test ./internal/cloudsync/... ./internal/inventory/...`.
   - Functional proof: replayed envelope is idempotent and Cloud queues one inventory event.

7. **Full Cloud Inventory Engine**
   - Files: `cloud-backend/internal/inventory/app/worker.go`, `cloud-backend/internal/inventory/infra/postgres/repository.go`, `cloud-backend/migrations/postgres/001_init.sql`, `docs/backend/INVENTORY-COSTING-SPEC.md`.
   - Implement: receipts, counts, production, sale consumption, refund/cancellation dispositions, recipe expansion, modifier linked consumption, balances, costing state and retro recalculation DAG.
   - Local check: `cd cloud-backend && go test ./internal/inventory/...`.
   - Functional proof: worker writes stock documents, ledger rows, balances and recalculation jobs for sale, return, waste, production, receipt and inventory count cases.

8. **ClickHouse OLAP runtime**
   - Files: `cloud-backend/internal/olap/`, `cloud-backend/internal/cloudsync/`, `cloud-backend/config/cloud-api.example.json`, `docker-compose.local.yml`, `docs/backend/POS-DATA-AND-MIGRATIONS.md`.
   - Implement: ClickHouse connection/config, managed schema for `raw_business_events` and `olap_stock_moves`, async forwarder, retry/backfill/export checkpoints.
   - Local check: `cd cloud-backend && go test ./internal/olap/... ./internal/cloudsync/...`.
   - Functional proof: accepted sync events export to `raw_business_events`, stock ledger exports to `olap_stock_moves`, failed ClickHouse writes retry without blocking POS sync.

9. **Cloud OLAP API**
   - Files: `cloud-backend/internal/olap/api/router.go`, `cloud-backend/internal/olap/app/service.go`, `cloud-backend/internal/olap/infra/clickhouse/repository.go`, `docs/backend/CLOUD-BACKEND-SPEC.md`.
   - Implement: bounded read-only endpoints for event archive, stock moves, sales aggregates, COGS/margin and kitchen timing.
   - Local check: `cd cloud-backend && go test ./internal/olap/...`.
   - Functional proof: APIs reject unbounded ranges, do not expose raw sensitive payloads and return deterministic aggregates from ClickHouse fixtures.

10. **Kitchen backend**
   - Files: `pos-backend/internal/pos/app/kitchen/service.go`, `pos-backend/internal/pos/app/kitchen/receipts.go`, `pos-backend/internal/pos/app/kitchen/proposals.go`, `pos-backend/internal/pos/infra/sqlite/kitchen_repository.go`, `pos-backend/internal/pos/api/router.go`, `pos-backend/migrations/sqlite/001_init.sql`.
   - Implement: advanced kitchen tickets, ticket item statuses, `KitchenTicketStatusChanged`, `ItemServed`, receipt capture, catalog suggestions, recipe change suggestions and stop-list edit.
   - Local check: `cd pos-backend && go test ./...`.
   - Functional proof: added dish creates ticket item; status transitions write KDS events; served transition writes one `ItemServed`; receipt/proposal flows write dedicated outbox events.

11. **Kitchen UI**
   - Files: `pos-ui/src/pages/KitchenPage.vue`, `pos-ui/src/shared/api.ts`, `pos-ui/src/shared/schemas.ts`, `pos-ui/e2e/kitchen-flow.spec.ts`.
   - Implement: ticket list, lifecycle transitions, receipt capture, catalog suggestion, recipe suggestion, stop-list edit, localized error/sync state.
   - Local check: `cd pos-ui && npm run build`.
   - Local check: `cd pos-ui && npx playwright test e2e/kitchen-flow.spec.ts`.
   - Functional proof: cashier-added dish appears in KDS; lifecycle, receipt, recipe suggestion and stop-list edit produce visible states and outbox evidence.

12. **Waiter RBAC and API boundary**
   - Files: `pos-backend/internal/pos/app/shared/permission_catalog.go`, `pos-backend/internal/pos/app/shared/permission_catalog_test.go`, `docs/ui/POS-UI-RBAC.md`.
   - Implement: waiter profile includes order/precheck permissions and excludes payment/refund by default.
   - Local check: `cd pos-backend && go test ./internal/pos/app/shared ./internal/pos/app`.
   - Functional proof: waiter cannot capture payment/refund/cancel closed check.

13. **Waiter mobile UI**
    - Files: `pos-ui/src/pages/WaiterPage.vue`, `pos-ui/src/pages/pos/useWaiterTerminal.ts`, `pos-ui/src/router.ts`, `pos-ui/e2e/waiter-mobile-flow.spec.ts`.
    - Implement: mobile floor/table/order/menu/modifier/precheck flow; no payment controls by default.
    - Local check: `cd pos-ui && npm run build`.
    - Local check: `cd pos-ui && npx playwright test e2e/waiter-mobile-flow.spec.ts`.
    - Functional proof: `390x844` viewport completes order/precheck flow and payment controls are absent.

14. **Cloud inventory manager**
    - Files: `cloud-backend/internal/masterdata/*`, `cloud-ui/src/shared/api.ts`, `cloud-ui/src/App.vue`, `cloud-ui/src/components/cloud/RecipeEditor.vue`.
    - Implement: recipe CRUD, stock receipts, inventory counts, production input, balances/costing status and manager UI editor.
    - Local check: `cd cloud-backend && go test ./internal/masterdata/...`.
    - Local check: `cd cloud-ui && npm run build`.
    - Functional proof: manager creates recipe, publishes package, records stock receipt/count/production and sees balances/costing status.

15. **Cloud manager proposal, stop-list, ClickHouse and OLAP observability**
    - Files: `cloud-ui/src/components/cloud/StopListPanel.vue`, `cloud-ui/src/components/cloud/LaunchReadinessPanel.vue`, `cloud-ui/src/components/cloud/EdgeEventsPanel.vue`.
    - Implement: stop-list editing, catalog/recipe proposal review, publication readiness, safe sync problem metadata, ClickHouse export health and OLAP endpoint previews.
    - Local check: `cd cloud-ui && npm run build`.
    - Functional proof: readiness blocks until stop-list/proposal review, publication and ClickHouse export health exist; problem events show metadata only.

16. **Full pilot smoke**
    - Files: `scripts/run-stack-smoke.py`, `scripts/tests/*`, `docs/temp/FULL-FLOW-SMOKE-2026-05-16.md`.
    - Implement: suite `full_pilot` covering Cloud setup, Edge sync, stop-list block, waiter order, advanced kitchen lifecycle, chef receipt/proposals, cashier payment, Cloud inventory ledger/balances/costing, ClickHouse export and OLAP API reads.
    - Local check: `python scripts/run-stack-smoke.py --suite full_pilot`.
    - Functional proof: suite passes from clean seed without manual DB edits.

17. **Release gate**
    - Files: `ROADMAP.md`, `docs/CURRENT-FUNCTIONAL-STATE.md`.
    - Implement: update statuses from `запланировано далее` to `реализовано сейчас` only for features that passed local verification.
    - Local check: `cd pos-backend && go mod tidy && go test ./...`.
    - Local check: `cd cloud-backend && go mod tidy && go test ./...`.
    - Local check: `cd pos-ui && npm install && npm run build`.
    - Local check: `cd cloud-ui && npm install && npm run build`.
    - Local check: `python scripts/run-stack-smoke.py --suite full_pilot`.
    - Functional proof: all gates pass and docs contain no unsupported behavior marked as current.

## Pilot Definition Of Done

- Кассир может открыть смены, создать order, добавить позиции/модификаторы, выпустить precheck, принять cash/card manual payment, получить final check, выполнить cancellation/refund ledger и закрыть смены.
- Официант на mobile viewport может выбрать стол, создать/изменить order, выпустить/reprint precheck и не видит payment/refund controls без прав.
- Кухня видит активные kitchen ticket items, проходит lifecycle `new -> accepted -> in_progress -> ready -> served`, может выполнить hold/recall/cancel, принять поставку, предложить catalog/recipe changes и отредактировать stop-list; события уходят в Edge outbox.
- Менеджер в Cloud может настроить ресторан, персонал, зал, меню, модификаторы, pricing, recipes и stop-list, рассмотреть catalog/recipe proposals, опубликовать master-data и увидеть sync readiness.
- Edge блокирует продажу блюда и обязательного recipe component из active stop-list локально, offline.
- Cloud принимает Edge events через `sync/exchange`, дедуплицирует replay и worker пишет full inventory documents, ledger, balances and costing state без synchronous stock mutation в POS request path.
- ClickHouse runtime экспортирует `raw_business_events` и `olap_stock_moves`; Cloud OLAP API возвращает bounded aggregates для продаж, склада, себестоимости и kitchen timing.
- Все новые HTTP routes, payloads, UI flows, RBAC, schema, sync events, error keys и smoke scripts задокументированы в профильных документах.

## Вне Текущего Объема

- PSP/fiscal device integration.
- Hardware bump-bar/printer integrations and rich BI dashboards beyond bounded pilot OLAP/KDS metrics.
- Delivery runtime.
- Визуальный drag-and-drop редактор зала как обязательное условие пилота.
- ERP/accounting integrations.
