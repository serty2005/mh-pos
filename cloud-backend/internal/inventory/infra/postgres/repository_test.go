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
