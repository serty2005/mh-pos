package postgres

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	syncpg "cloud-backend/internal/cloudsync/infra/postgres"
	"cloud-backend/internal/inventory/app"
	platformpg "cloud-backend/internal/platform/postgres"
)

func TestCreateStockDocumentUpdatesMaterializedBalancesIdempotently(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openInventoryPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)

	receipt := stockDocumentForTest("doc-receipt", "event-receipt", app.DocumentPurchase, app.MovementIn, "3.000", "final", now)
	if err := repo.CreateStockDocument(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	sale := stockDocumentForTest("doc-sale", "event-sale", app.DocumentSale, app.MovementOut, "5.000", "estimated", now.Add(time.Hour))
	if err := repo.CreateStockDocument(ctx, sale); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateStockDocument(ctx, sale); err != nil {
		t.Fatal(err)
	}

	var quantity, costingStatus, lastLedgerEntryID string
	var needsRecalculation bool
	if err := pool.QueryRow(ctx, `
SELECT quantity_on_hand::text,costing_status,needs_recalculation,last_ledger_entry_id
FROM inventory_stock_balances
WHERE restaurant_id = 'restaurant-1' AND warehouse_id = 'warehouse-main' AND catalog_item_id = 'item-1' AND unit_code = 'PC'`).Scan(
		&quantity, &costingStatus, &needsRecalculation, &lastLedgerEntryID,
	); err != nil {
		t.Fatal(err)
	}
	if quantity != "-2.000" || costingStatus != "estimated" || !needsRecalculation || lastLedgerEntryID != "ledger-doc-sale" {
		t.Fatalf("unexpected materialized balance: quantity=%s status=%s needs=%v last=%s", quantity, costingStatus, needsRecalculation, lastLedgerEntryID)
	}
	var ledgerCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM stock_ledger`).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if ledgerCount != 2 {
		t.Fatalf("duplicate source event must not create duplicate ledger rows, got %d", ledgerCount)
	}
}

func TestCreateStockDocumentPostsProcessingStateIdempotently(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openInventoryPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)

	document := stockDocumentForTest("doc-receipt", "event-receipt", app.DocumentPurchase, app.MovementIn, "3.000", "final", now)
	document.SourceEventType = "StockReceiptCaptured"
	document.Ledger[0].SourceEventType = "StockReceiptCaptured"
	document.ProcessingState = &app.ProcessingStateCommand{
		ID:                "state-receipt",
		RestaurantID:      "restaurant-1",
		SourceEventID:     "event-receipt",
		SourceEventType:   "StockReceiptCaptured",
		SourceAggregateID: "receipt-1",
		Now:               now,
	}
	if err := repo.CreateStockDocument(ctx, document); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateStockDocument(ctx, document); err != nil {
		t.Fatal(err)
	}

	var status, documentID, costingStatus string
	var postedCount, expectedCount int
	var needs bool
	if err := pool.QueryRow(ctx, `
SELECT status,COALESCE(stock_document_id,''),posted_ledger_count,expected_ledger_count,costing_status,needs_recalculation
FROM inventory_document_processing_state
WHERE restaurant_id = 'restaurant-1' AND source_event_id = 'event-receipt' AND source_event_type = 'StockReceiptCaptured'`).Scan(
		&status, &documentID, &postedCount, &expectedCount, &costingStatus, &needs,
	); err != nil {
		t.Fatal(err)
	}
	if status != "posted" || documentID != "doc-receipt" || postedCount != 1 || expectedCount != 1 || costingStatus != "final" || needs {
		t.Fatalf("unexpected processing state: status=%s doc=%s posted=%d expected=%d costing=%s needs=%v", status, documentID, postedCount, expectedCount, costingStatus, needs)
	}
	var stateCount, ledgerCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM inventory_document_processing_state`).Scan(&stateCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM stock_ledger`).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if stateCount != 1 || ledgerCount != 1 {
		t.Fatalf("replay must keep one state and one ledger row, state=%d ledger=%d", stateCount, ledgerCount)
	}
}

func TestCreateStockDocumentRollsBackProcessingStateWithLedgerFailure(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openInventoryPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)

	document := stockDocumentForTest("doc-bad", "event-bad", app.DocumentPurchase, app.MovementIn, "3.000", "final", now)
	document.SourceEventType = "StockReceiptCaptured"
	document.Ledger[0].SourceEventType = "StockReceiptCaptured"
	document.Ledger[0].UnitCode = ""
	document.ProcessingState = &app.ProcessingStateCommand{
		ID:              "state-bad",
		RestaurantID:    "restaurant-1",
		SourceEventID:   "event-bad",
		SourceEventType: "StockReceiptCaptured",
		Now:             now,
	}
	if err := repo.CreateStockDocument(ctx, document); err == nil {
		t.Fatal("expected invalid ledger row to fail")
	}
	var stateCount, documentCount, ledgerCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM inventory_document_processing_state`).Scan(&stateCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM stock_documents`).Scan(&documentCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM stock_ledger`).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if stateCount != 0 || documentCount != 0 || ledgerCount != 0 {
		t.Fatalf("document/ledger/state must roll back together, state=%d doc=%d ledger=%d", stateCount, documentCount, ledgerCount)
	}
}

func TestCreateStockDocumentAggregatesBalanceCostingStatusConservatively(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openInventoryPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)

	for _, document := range []app.StockDocument{
		stockDocumentForTest("doc-final", "event-final", app.DocumentPurchase, app.MovementIn, "1.000", "final", now),
		stockDocumentForTest("doc-recalculated", "event-recalculated", app.DocumentPurchase, app.MovementIn, "1.000", "recalculated", now.Add(time.Hour)),
	} {
		document.Ledger[0].CatalogItemID = "item-status"
		if err := repo.CreateStockDocument(ctx, document); err != nil {
			t.Fatal(err)
		}
	}
	assertBalanceStatus(t, ctx, pool, "item-status", "recalculated", false)

	estimated := stockDocumentForTest("doc-estimated", "event-estimated", app.DocumentSale, app.MovementOut, "1.000", "estimated", now.Add(2*time.Hour))
	estimated.Ledger[0].CatalogItemID = "item-status"
	if err := repo.CreateStockDocument(ctx, estimated); err != nil {
		t.Fatal(err)
	}
	assertBalanceStatus(t, ctx, pool, "item-status", "estimated", true)

	needs := stockDocumentForTest("doc-needs", "event-needs", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", now.Add(3*time.Hour))
	needs.Ledger[0].CatalogItemID = "item-status"
	if err := repo.CreateStockDocument(ctx, needs); err != nil {
		t.Fatal(err)
	}
	assertBalanceStatus(t, ctx, pool, "item-status", "needs_recalculation", true)

	failed := stockDocumentForTest("doc-failed", "event-failed", app.DocumentSale, app.MovementOut, "1.000", "failed", now.Add(4*time.Hour))
	failed.Ledger[0].CatalogItemID = "item-status"
	if err := repo.CreateStockDocument(ctx, failed); err != nil {
		t.Fatal(err)
	}
	assertBalanceStatus(t, ctx, pool, "item-status", "failed", false)
}

func TestRecalculationJobLifecycleUpdatesCostingFieldsIdempotently(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openInventoryPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)

	future := stockDocumentForTest("doc-future-sale", "event-future-sale", app.DocumentSale, app.MovementOut, "2.000", "estimated", now.Add(24*time.Hour))
	if err := repo.CreateStockDocument(ctx, future); err != nil {
		t.Fatal(err)
	}
	receipt := stockDocumentForTest("doc-backdated-receipt", "event-backdated-receipt", app.DocumentPurchase, app.MovementIn, "3.000", "final", now)
	receipt.SourceEventType = "StockReceiptCaptured"
	receipt.Ledger[0].SourceEventType = "StockReceiptCaptured"
	receipt.Ledger[0].UnitCostMinor = 150
	receipt.Ledger[0].TotalCostMinor = 450
	if err := repo.CreateStockDocument(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	cmd := app.RecalculationTriggerCommand{
		ID:               "018f0000-0000-7000-8000-00000000aa01",
		RestaurantID:     "restaurant-1",
		SourceDocumentID: receipt.ID,
		TriggerType:      "StockReceiptCaptured",
		TriggerEventID:   "event-backdated-receipt",
		BusinessDateFrom: "2026-06-15",
		OccurredAt:       receipt.OccurredAt,
		Ledger:           receipt.Ledger,
		Now:              now,
	}
	if err := repo.CreateRecalculationJob(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRecalculationJob(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	var jobCount, itemCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM stock_recalculation_jobs`).Scan(&jobCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM stock_recalculation_job_items`).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if jobCount != 1 || itemCount != 1 {
		t.Fatalf("expected idempotent job/items, jobs=%d items=%d", jobCount, itemCount)
	}
	var markedStatus string
	if err := pool.QueryRow(ctx, `SELECT costing_status FROM stock_ledger WHERE id = 'ledger-doc-future-sale'`).Scan(&markedStatus); err != nil {
		t.Fatal(err)
	}
	if markedStatus != "needs_recalculation" {
		t.Fatalf("expected future row marked for recalculation, got %s", markedStatus)
	}

	job, ok, err := repo.ClaimRecalculationJob(ctx, app.RecalculationClaimCommand{LockedBy: "worker-1", Now: now.Add(time.Minute)})
	if err != nil || !ok {
		t.Fatalf("expected claimed job, ok=%v err=%v", ok, err)
	}
	if _, ok, err := repo.ClaimRecalculationJob(ctx, app.RecalculationClaimCommand{LockedBy: "worker-2", Now: now.Add(time.Minute)}); err != nil || ok {
		t.Fatalf("second claim must not get running job, ok=%v err=%v", ok, err)
	}
	if err := repo.ValidateRecalculationDAG(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ListRecalculationLedgerRows(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "ledger-doc-future-sale" {
		t.Fatalf("unexpected deterministic recalculation rows: %+v", rows)
	}
	basis, ok, err := repo.LatestCostBasis(ctx, app.CostBasisQuery{
		RestaurantID:  rows[0].RestaurantID,
		WarehouseID:   rows[0].WarehouseID,
		CatalogItemID: rows[0].CatalogItemID,
		UnitCode:      rows[0].UnitCode,
		OccurredAt:    rows[0].OccurredAt,
		LedgerID:      rows[0].ID,
	})
	if err != nil || !ok || basis != 150 {
		t.Fatalf("expected receipt basis 150, ok=%v basis=%d err=%v", ok, basis, err)
	}
	if err := repo.UpdateRecalculationLedgerRow(ctx, app.RecalculationLedgerUpdate{
		JobID:          job.ID,
		LedgerID:       rows[0].ID,
		UnitCostMinor:  basis,
		TotalCostMinor: 300,
		CostingStatus:  "recalculated",
		CompletedSteps: 1,
		Now:            now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteRecalculationJob(ctx, app.RecalculationJobProgress{JobID: job.ID, TotalSteps: 1, CompletedSteps: 1, Now: now.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	var status string
	var completed int
	if err := pool.QueryRow(ctx, `SELECT status,completed_steps FROM stock_recalculation_jobs WHERE id = $1`, job.ID).Scan(&status, &completed); err != nil {
		t.Fatal(err)
	}
	if status != "completed" || completed != 1 {
		t.Fatalf("unexpected job completion: status=%s completed=%d", status, completed)
	}
	var unitCost int64
	if err := pool.QueryRow(ctx, `SELECT unit_cost_minor,costing_status FROM stock_ledger WHERE id = 'ledger-doc-future-sale'`).Scan(&unitCost, &markedStatus); err != nil {
		t.Fatal(err)
	}
	if unitCost != 150 || markedStatus != "recalculated" {
		t.Fatalf("unexpected recalculated ledger row: cost=%d status=%s", unitCost, markedStatus)
	}
	assertBalanceStatus(t, ctx, pool, "item-1", "recalculated", false)
}

func TestRecalculationJobFailsSafelyOnRecipeCycle(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openInventoryPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_catalog_items(id,restaurant_id,kind,name,base_unit,status,cloud_version,created_at,updated_at)
VALUES ('item-a','restaurant-1','semi_finished','A','PC','active',1,$1,$1),
       ('item-b','restaurant-1','semi_finished','B','PC','active',1,$1,$1);
INSERT INTO cloud_recipe_versions(id,restaurant_id,owner_catalog_item_id,version,name,status,yield_quantity,yield_unit,created_at,updated_at)
VALUES ('recipe-a','restaurant-1','item-a',1,'A','active',1,'PC',$1,$1),
       ('recipe-b','restaurant-1','item-b',1,'B','active',1,'PC',$1,$1);
INSERT INTO cloud_recipe_lines(id,recipe_version_id,component_catalog_item_id,quantity,unit,sort_order,created_at,updated_at)
VALUES ('line-a-b','recipe-a','item-b',1,'PC',1,$1,$1),
       ('line-b-a','recipe-b','item-a',1,'PC',1,$1,$1)`, now); err != nil {
		t.Fatal(err)
	}
	future := stockDocumentForTest("doc-cycle-future", "event-cycle-future", app.DocumentSale, app.MovementOut, "1.000", "estimated", now.Add(24*time.Hour))
	future.Ledger[0].CatalogItemID = "item-b"
	if err := repo.CreateStockDocument(ctx, future); err != nil {
		t.Fatal(err)
	}
	trigger := stockDocumentForTest("doc-cycle-trigger", "event-cycle-trigger", app.DocumentPurchase, app.MovementIn, "1.000", "final", now)
	trigger.Ledger[0].CatalogItemID = "item-a"
	trigger.Ledger[0].UnitCostMinor = 100
	trigger.Ledger[0].TotalCostMinor = 100
	if err := repo.CreateStockDocument(ctx, trigger); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRecalculationJob(ctx, app.RecalculationTriggerCommand{
		ID:               "018f0000-0000-7000-8000-00000000aa02",
		RestaurantID:     "restaurant-1",
		SourceDocumentID: trigger.ID,
		TriggerType:      "StockReceiptCaptured",
		TriggerEventID:   "event-cycle-trigger",
		BusinessDateFrom: "2026-06-15",
		OccurredAt:       trigger.OccurredAt,
		Ledger:           trigger.Ledger,
		Now:              now,
	}); err != nil {
		t.Fatal(err)
	}
	job, ok, err := repo.ClaimRecalculationJob(ctx, app.RecalculationClaimCommand{LockedBy: "worker-1", Now: now})
	if err != nil || !ok {
		t.Fatalf("expected cycle job, ok=%v err=%v", ok, err)
	}
	if err := repo.ValidateRecalculationDAG(ctx, job.ID); err == nil {
		t.Fatal("expected recipe dependency cycle")
	}
	if err := repo.FailRecalculationJob(ctx, app.RecalculationJobFailure{JobID: job.ID, FailureCode: "RECIPE_DEPENDENCY_CYCLE", FailureMessageKey: "inventory.recalculation.recipe_cycle", Now: now}); err != nil {
		t.Fatal(err)
	}
	var status, code, key string
	if err := pool.QueryRow(ctx, `SELECT status,COALESCE(failure_code,''),COALESCE(failure_message_key,'') FROM stock_recalculation_jobs WHERE id = $1`, job.ID).Scan(&status, &code, &key); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || code != "RECIPE_DEPENDENCY_CYCLE" || key != "inventory.recalculation.recipe_cycle" {
		t.Fatalf("unexpected safe failure metadata: status=%s code=%s key=%s", status, code, key)
	}
}

func stockDocumentForTest(id, eventID string, documentType app.DocumentType, movement app.MovementType, quantity, costingStatus string, occurredAt time.Time) app.StockDocument {
	return app.StockDocument{
		ID:                id,
		RestaurantID:      "restaurant-1",
		WarehouseID:       "warehouse-main",
		Type:              documentType,
		SourceEventID:     eventID,
		SourceEventType:   string(documentType),
		BusinessDateLocal: occurredAt.Format("2006-01-02"),
		OccurredAt:        occurredAt,
		CreatedAt:         occurredAt,
		Ledger: []app.StockLedgerEntry{{
			ID:                "ledger-" + id,
			RestaurantID:      "restaurant-1",
			WarehouseID:       "warehouse-main",
			SourceEventID:     eventID,
			SourceEventType:   string(documentType),
			CatalogItemID:     "item-1",
			MovementType:      movement,
			Quantity:          quantity,
			UnitCode:          "PC",
			CostingStatus:     costingStatus,
			OccurredAt:        occurredAt,
			BusinessDateLocal: occurredAt.Format("2006-01-02"),
			CreatedAt:         occurredAt,
		}},
	}
}

func assertBalanceStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, catalogItemID, wantStatus string, wantNeeds bool) {
	t.Helper()
	var gotStatus string
	var gotNeeds bool
	if err := pool.QueryRow(ctx, `
SELECT costing_status,needs_recalculation
FROM inventory_stock_balances
WHERE restaurant_id = 'restaurant-1' AND warehouse_id = 'warehouse-main' AND catalog_item_id = $1 AND unit_code = 'PC'`, catalogItemID).Scan(&gotStatus, &gotNeeds); err != nil {
		t.Fatal(err)
	}
	if gotStatus != wantStatus || gotNeeds != wantNeeds {
		t.Fatalf("unexpected balance costing for %s: status=%s needs=%v", catalogItemID, gotStatus, gotNeeds)
	}
}

func openInventoryPostgresWithBaseline(t *testing.T, ctx context.Context) (*pgxpool.Pool, func()) {
	t.Helper()
	dsn := os.Getenv("CLOUD_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("CLOUD_POSTGRES_TEST_DSN is not set")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(72905102)`); err != nil {
		pool.Close()
		t.Fatalf("lock postgres integration db: %v", err)
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		pool.Close()
		t.Fatalf("reset public schema: %v", err)
	}
	migrationsDir := filepath.Join("..", "..", "..", "..", "migrations", "postgres")
	if err := platformpg.MigrateDirWithPolicy(ctx, pool, migrationsDir, platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: syncpg.RequiredSchema(),
	}); err != nil {
		pool.Close()
		t.Fatalf("migrate postgres baseline: %v", err)
	}
	return pool, func() {
		_, _ = pool.Exec(context.Background(), `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`)
		_, _ = pool.Exec(context.Background(), `SELECT pg_advisory_unlock(72905102)`)
		pool.Close()
	}
}
