# Cloud Inventory Target Cutover Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Cloud-owned inventory event intake and worker processing while removing POS Edge manual stock document runtime.

**Architecture:** Cloud sync accepts target inventory events and writes durable queue rows in the receive transaction. A Cloud Inventory Worker claims queue rows and writes idempotent `stock_documents` and `stock_ledger`; POS Edge keeps recipe/stop-list reference only and removes legacy stock document/move/balance/costing tables and services.

**Tech Stack:** Go 1.26.2, PostgreSQL via pgx, SQLite via modernc, existing Cloud sync receiver, existing managed SQL baseline.

---

### Task 1: Cloud Inventory Contracts

**Files:**
- Modify: `cloud-backend/internal/cloudsync/contracts/types.go`
- Modify: `cloud-backend/internal/cloudsync/contracts/types_test.go`
- Modify: `cloud-backend/migrations/postgres/001_init.sql`

- [x] Write failing tests for `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `StopListUpdated`.
- [x] Run `cd cloud-backend && go test ./internal/cloudsync/contracts`.
- [x] Add event constants, payload structs and validation.
- [x] Add event names to PostgreSQL event type checks.
- [x] Re-run `cd cloud-backend && go test ./internal/cloudsync/contracts`.

### Task 2: Durable Inventory Queue

**Files:**
- Modify: `cloud-backend/migrations/postgres/001_init.sql`
- Modify: `cloud-backend/internal/cloudsync/infra/postgres/schema.go`
- Modify: `cloud-backend/internal/cloudsync/infra/postgres/schema_test.go`
- Modify: `cloud-backend/internal/cloudsync/infra/postgres/repository.go`
- Modify: `cloud-backend/internal/cloudsync/infra/memory/repository.go`
- Modify: `cloud-backend/internal/cloudsync/app/service_test.go`

- [x] Write failing test that receiving an inventory event creates one queue row and replay keeps one row.
- [x] Run `cd cloud-backend && go test ./internal/cloudsync/app ./internal/cloudsync/infra/postgres`.
- [x] Add `inventory_event_queue` table, indexes and schema verification.
- [x] Enqueue inventory-relevant receipts in the same receive transaction.
- [x] Re-run focused tests.

### Task 3: Cloud Inventory Worker

**Files:**
- Create: `cloud-backend/internal/inventory/app/worker.go`
- Create: `cloud-backend/internal/inventory/app/worker_test.go`
- Create: `cloud-backend/internal/inventory/infra/postgres/repository.go`
- Modify: `cloud-backend/cmd/cloud-api/main.go`

- [x] Write failing worker tests for `CheckClosed` sale ledger, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `RefundRecorded return_to_stock`, `CancellationRecorded write_off_waste`, idempotent replay, and failed/manual-review whole-check stock effect without item rows.
- [x] Run `cd cloud-backend && go test ./internal/inventory/...`.
- [x] Implement repository claim/mark methods and document/ledger inserts.
- [x] Implement worker event mapping and minimal costing fallback.
- [x] Start worker from `cloud-api` after migrations and before HTTP shutdown wait.
- [x] Re-run `cd cloud-backend && go test ./internal/inventory/... ./internal/cloudsync/...`.

### Task 4: POS Edge Legacy Removal

**Files:**
- Delete: `pos-backend/internal/pos/app/inventory/service.go`
- Modify: `pos-backend/internal/pos/app/service.go`
- Modify: `pos-backend/internal/pos/ports/repository.go`
- Modify: `pos-backend/internal/pos/ports/inventory_repository.go`
- Modify: `pos-backend/internal/pos/infra/sqlite/inventory_repository.go`
- Modify: `pos-backend/internal/pos/domain/inventory/inventory.go`
- Modify: `pos-backend/internal/pos/domain/aliases.go`
- Modify: `pos-backend/migrations/sqlite/001_init.sql`
- Modify: `pos-backend/internal/pos/infra/sqlite/storage_repository.go`
- Modify: `pos-backend/internal/pos/domain/storage/lifecycle.go`
- Modify: `pos-backend/internal/pos/app/service_test.go`
- Modify: `pos-backend/internal/pos/infra/sqlite/schema_test.go`

- [x] Write/adjust failing tests asserting Edge baseline has no legacy stock tables and application service has no manual stock document surface.
- [x] Run `cd pos-backend && go test ./internal/pos/...`.
- [x] Remove manual stock document service and repository methods.
- [x] Keep recipe/stop-list reference repository methods only.
- [x] Remove legacy SQLite tables and storage count fields.
- [x] Re-run `cd pos-backend && go test ./internal/pos/...`.

### Task 5: Docs And Verification

**Files:**
- Modify: `README.md`
- Modify: `pos-backend/README.md`
- Modify: `cloud-backend/README.md`
- Modify: `ROADMAP.md`
- Modify: `SPECv1.3.md`
- Modify: `docs/backend/INVENTORY-COSTING-SPEC.md`
- Modify: `docs/backend/POS-BACKEND-SPEC.md`
- Modify: `docs/backend/POS-DATA-AND-MIGRATIONS.md`
- Modify: `docs/sync/edge-cloud-contracts-v1.md`
- Modify: `docs/sync/directional-sync-ownership.md`
- Modify: `docs/CURRENT-FUNCTIONAL-STATE.md`

- [x] Update docs to say target Cloud inventory worker is implemented for normalized event items and Edge legacy manual stock method was used previously, then removed.
- [x] Run profile searches from `AGENTS.md`.
- [x] Run `cd cloud-backend && go mod tidy && go test ./...`.
- [x] Run `cd pos-backend && go mod tidy && go test ./...`.
- [x] Run `cd pos-ui && npm install && npm run build` only if UI/docs contracts require it. Not required for this runtime/docs backend-only cutover.
