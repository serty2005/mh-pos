package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	platformsqlite "pos-backend/internal/platform/sqlite"
)

const schemaTestTime = "2026-05-04T20:00:00Z"

func newSchemaDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()
	ctx := context.Background()
	db, err := platformsqlite.Open(filepath.Join(t.TempDir(), "pos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := platformsqlite.MigrateDir(ctx, db, filepath.Join("..", "..", "..", "..", "migrations", "sqlite")); err != nil {
		t.Fatal(err)
	}
	return db, ctx
}

func seedCatalogForSchemaTests(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	execSchema(t, ctx, db, `INSERT INTO restaurants(id,name,timezone,currency,active,created_at,updated_at) VALUES ('restaurant-1','Demo','UTC','RUB',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO devices(id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at) VALUES ('device-1','restaurant-1','POS-1','Main','windows',1,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES ('dish-1','dish','Soup','DISH-1','portion',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES ('ingredient-1','ingredient','Potato','ING-1','g',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES ('good-1','good','Bottle','GOOD-1','pcs',1,?,?)`, schemaTestTime, schemaTestTime)
}

func execSchema(t *testing.T, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func TestPhase2FoundationTablesExist(t *testing.T) {
	db, ctx := newSchemaDB(t)
	tables := []string{
		"recipe_versions",
		"recipe_lines",
		"purchase_receipts",
		"purchase_receipt_lines",
		"stock_documents",
		"stock_moves",
		"stock_balances",
		"item_costs",
	}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			var n int
			err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
			if err != nil {
				t.Fatal(err)
			}
			if n != 1 {
				t.Fatalf("expected table %s to exist", table)
			}
		})
	}
}

func TestLocalEventLogFoundationTableExists(t *testing.T) {
	db, ctx := newSchemaDB(t)
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'local_event_log'`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("expected local_event_log table to exist")
	}
}

func TestLocalEventLogRequiresUniqueEdgeEventIdentity(t *testing.T) {
	db, ctx := newSchemaDB(t)
	insert := `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,shift_id,payload_json,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "local-event-1", "edge-event-1", "cmd-1", "1", "OrderCreated", "Order", "order-1", "restaurant-1", "device-1", "shift-1", `{"event_id":"edge-event-1"}`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "local-event-2", "edge-event-1", "cmd-2", "1", "OrderCreated", "Order", "order-1", "restaurant-1", "device-1", "shift-1", `{"event_id":"edge-event-1"}`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected duplicate event_id to fail")
	}
}

func TestLocalEventLogRequiresUniqueCommandID(t *testing.T) {
	db, ctx := newSchemaDB(t)
	insert := `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,shift_id,payload_json,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "local-event-1", "edge-event-1", "cmd-1", "1", "OrderCreated", "Order", "order-1", "restaurant-1", "device-1", "shift-1", `{"command_id":"cmd-1"}`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "local-event-2", "edge-event-2", "cmd-1", "1", "OrderCreated", "Order", "order-2", "restaurant-1", "device-1", "shift-1", `{"command_id":"cmd-1"}`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected duplicate command_id to fail")
	}
}

func TestLocalEventLogRequiresDeviceID(t *testing.T) {
	db, ctx := newSchemaDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,shift_id,payload_json,occurred_at,created_at) VALUES ('local-event-1','edge-event-1','cmd-1','1','OrderCreated','Order','order-1','restaurant-1','','shift-1','{}',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected empty device_id to fail")
	}
}

func TestStockMovesCannotHaveZeroQuantity(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO stock_documents(id,restaurant_id,device_id,document_type,status,occurred_at,created_at,updated_at) VALUES ('stock-doc-1','restaurant-1','device-1','adjustment','posted',?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO stock_moves(id,stock_document_id,catalog_item_id,movement_type,quantity,unit,occurred_at,created_at) VALUES ('stock-move-1','stock-doc-1','ingredient-1','adjustment',0,'g',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected zero quantity stock move to fail")
	}
}

func TestStockBalancesUniqueByCatalogItemLocationWhenLocationExists(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO stock_balances(id,catalog_item_id,location_id,quantity,unit,updated_at) VALUES ('balance-1','ingredient-1','kitchen',10,'g',?)`, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO stock_balances(id,catalog_item_id,location_id,quantity,unit,updated_at) VALUES ('balance-2','ingredient-1','kitchen',20,'g',?)`, schemaTestTime)
	if err == nil {
		t.Fatal("expected duplicate catalog/location stock balance to fail")
	}
}

func TestRecipeVersionReferencesDishCatalogItem(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)

	_, err := db.ExecContext(ctx, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES ('recipe-bad','ingredient-1',1,'Bad','draft',1,'portion',1,?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected recipe version for non-dish catalog item to fail")
	}
	execSchema(t, ctx, db, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES ('recipe-1','dish-1',1,'Soup v1','draft',1,'portion',1,?,?)`, schemaTestTime, schemaTestTime)
}

func TestRecipeLinesReferenceIngredientOrGoodCatalogItems(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES ('recipe-1','dish-1',1,'Soup v1','draft',1,'portion',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES ('recipe-line-1','recipe-1','ingredient-1',100,'g',0,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES ('recipe-line-2','recipe-1','good-1',1,'pcs',0,?,?)`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES ('recipe-line-bad','recipe-1','dish-1',1,'portion',0,?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected recipe line for dish catalog item to fail")
	}
}
