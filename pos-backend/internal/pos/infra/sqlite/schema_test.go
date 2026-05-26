package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	platformsqlite "pos-backend/internal/platform/sqlite"
	possqlite "pos-backend/internal/pos/infra/sqlite"
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
	execSchema(t, ctx, db, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES ('semi-finished-1','semi_finished','Potato prep','SEMI-1','g',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES ('good-1','good','Bottle','GOOD-1','pcs',1,?,?)`, schemaTestTime, schemaTestTime)
}

func seedFinancialForSchemaTests(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO roles(id,name,permissions_json,active,created_at,updated_at) VALUES ('role-1','cashier','{}',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO employees(id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at) VALUES ('employee-1','restaurant-1','role-1','Anna','hash',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO halls(id,restaurant_id,name,active,created_at,updated_at) VALUES ('hall-1','restaurant-1','Main',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO tables(id,restaurant_id,hall_id,name,seats,active,created_at,updated_at) VALUES ('table-1','restaurant-1','hall-1','A1',2,1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO tables(id,restaurant_id,hall_id,name,seats,active,created_at,updated_at) VALUES ('table-2','restaurant-1','hall-1','A2',2,1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO shifts(id,restaurant_id,device_id,opened_by_employee_id,status,business_date_local,opened_at,opening_cash_amount,created_at,updated_at) VALUES ('shift-1','restaurant-1','device-1','employee-1','open','2026-05-04',?,0,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,created_at,updated_at) VALUES ('order-1','edge-order-1','restaurant-1','device-1','shift-1','open','table-1','A1',1,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO checks(id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,business_date_local,closed_at,snapshot,created_at,updated_at) VALUES ('check-1','order-1','open',100,0,0,100,0,'2026-05-04',?,'{}',?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
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
		"stop_lists",
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

func TestEdgeLegacyStockTablesAreRemovedFromCleanInstall(t *testing.T) {
	db, ctx := newSchemaDB(t)
	tables := []string{"purchase_receipts", "purchase_receipt_lines", "stock_documents", "stock_moves", "stock_balances", "item_costs"}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			var n int
			err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
			if err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Fatalf("expected legacy Edge stock table %s to be absent", table)
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

func TestCashAndPaymentAttemptFoundationTablesExist(t *testing.T) {
	db, ctx := newSchemaDB(t)
	tables := []string{"cash_sessions", "cash_drawer_events", "payment_attempts"}
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

func TestPrecheckFoundationTableExists(t *testing.T) {
	db, ctx := newSchemaDB(t)
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'prechecks'`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("expected prechecks table to exist")
	}
}

func TestKitchenFoundationTablesExist(t *testing.T) {
	db, ctx := newSchemaDB(t)
	tables := []string{"kitchen_tickets", "kitchen_ticket_events"}
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

func TestManagerOverrideAuditTableExists(t *testing.T) {
	db, ctx := newSchemaDB(t)
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'manager_override_audit'`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("expected manager_override_audit table to exist")
	}
}

func TestOrdersAllowLockedStatus(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)

	execSchema(t, ctx, db, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,created_at,updated_at) VALUES ('order-locked','edge-order-locked','restaurant-1','device-1','shift-1','locked','table-2','A2',1,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = 'order-locked'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "locked" {
		t.Fatalf("expected locked status, got %s", status)
	}
}

func TestAuthSessionsAndActorMetadataReferenceLocalEmployees(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO auth_sessions(id,restaurant_id,device_id,node_device_id,client_device_id,employee_id,status,started_at,last_seen_at,created_at,updated_at) VALUES ('session-1','restaurant-1','device-1','device-1','client-1','employee-1','active',?,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,client_device_id,shift_id,actor_employee_id,session_id,payload_json,occurred_at,created_at) VALUES ('local-event-actor','edge-event-actor','cmd-actor','1','OrderCreated','Order','order-1','restaurant-1','device-1','device-1','client-1','shift-1','employee-1','session-1','{}',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,node_device_id,client_device_id,actor_employee_id,session_id,aggregate_type,aggregate_id,command_type,payload_json,status,created_at,updated_at) VALUES ('outbox-actor','cmd-actor',1,'edge_device','restaurant-1','device-1','device-1','client-1','employee-1','session-1','Order','order-1','OrderCreated','{}','pending',?,?)`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,actor_employee_id,payload_json,occurred_at,created_at) VALUES ('local-event-bad-actor','edge-event-bad-actor','cmd-bad-actor','1','OrderCreated','Order','order-1','restaurant-1','device-1','device-1','missing-employee','{}',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected invalid actor_employee_id foreign key")
	}
}

func TestOrdersRequireTableEntity(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)

	_, err := db.ExecContext(ctx, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,created_at,updated_at) VALUES ('order-bad-table','edge-order-bad-table','restaurant-1','device-1','shift-1','open','missing-table','Ghost',1,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected invalid table_id foreign key")
	}
}

func TestPrechecksAllowOnlyOneIssuedPrecheckPerOrder(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	insert := `INSERT INTO prechecks(id,order_id,status,subtotal,discount_total,tax_total,total,snapshot,created_at,issued_at) VALUES (?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "precheck-1", "order-1", "issued", 100, 0, 0, 100, "{}", schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "precheck-2", "order-1", "issued", 100, 0, 0, 100, "{}", schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected second issued precheck for order to fail")
	}
}

func TestPrecheckTotalsAllowInclusiveTaxSnapshot(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)

	execSchema(t, ctx, db, `INSERT INTO prechecks(id,order_id,status,subtotal,discount_total,tax_total,total,snapshot,created_at,issued_at) VALUES ('precheck-inclusive','order-1','issued',100,0,20,100,'{}',?,?)`, schemaTestTime, schemaTestTime)
}

func TestPrecheckLifecycleFoundationConstraints(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	insert := `INSERT INTO prechecks(id,order_id,status,version,subtotal,discount_total,tax_total,total,paid_total,snapshot,created_at,issued_at,closed_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "precheck-1", "order-1", "cancelled", 1, 100, 0, 0, 100, 0, "{}", schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, insert, "precheck-2", "order-1", "superseded", 2, 100, 0, 0, 100, 0, "{}", schemaTestTime, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "precheck-bad-version", "order-1", "cancelled", 0, 100, 0, 0, 100, 0, "{}", schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected non-positive precheck version to fail")
	}
	_, err = db.ExecContext(ctx, insert, "precheck-bad-paid-total", "order-1", "cancelled", 3, 100, 0, 0, 100, 101, "{}", schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected paid_total above total to fail")
	}
	_, err = db.ExecContext(ctx, insert, "precheck-bad-terminal-time", "order-1", "cancelled", 4, 100, 0, 0, 100, 0, "{}", schemaTestTime, schemaTestTime, nil)
	if err == nil {
		t.Fatal("expected terminal precheck without closed_at to fail")
	}
}

func TestCashSessionsAllowOnlyOneOpenSessionPerDevice(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	insert := `INSERT INTO cash_sessions(id,edge_cash_session_id,restaurant_id,device_id,shift_id,opened_by_employee_id,status,business_date_local,opening_cash_amount,opened_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "cash-session-1", "edge-cash-session-1", "restaurant-1", "device-1", "shift-1", "employee-1", "open", "2026-05-04", 100, schemaTestTime, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "cash-session-2", "edge-cash-session-2", "restaurant-1", "device-1", "shift-1", "employee-1", "open", "2026-05-04", 200, schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected second open cash session on device to fail")
	}
}

func TestCashDrawerEventsRequireSessionAndNonNegativeAmount(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO cash_sessions(id,edge_cash_session_id,restaurant_id,device_id,shift_id,opened_by_employee_id,status,business_date_local,opening_cash_amount,opened_at,created_at,updated_at) VALUES ('cash-session-1','edge-cash-session-1','restaurant-1','device-1','shift-1','employee-1','open','2026-05-04',100,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO cash_drawer_events(id,edge_cash_drawer_event_id,cash_session_id,restaurant_id,device_id,shift_id,created_by_employee_id,event_type,amount,occurred_at,created_at) VALUES ('cash-event-1','edge-cash-event-1','cash-session-1','restaurant-1','device-1','shift-1','employee-1','cash_in',100,?,?)`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO cash_drawer_events(id,edge_cash_drawer_event_id,cash_session_id,restaurant_id,device_id,shift_id,created_by_employee_id,event_type,amount,occurred_at,created_at) VALUES ('cash-event-2','edge-cash-event-2','missing','restaurant-1','device-1','shift-1','employee-1','cash_in',100,?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected cash drawer event without session to fail")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO cash_drawer_events(id,edge_cash_drawer_event_id,cash_session_id,restaurant_id,device_id,shift_id,created_by_employee_id,event_type,amount,occurred_at,created_at) VALUES ('cash-event-3','edge-cash-event-3','cash-session-1','restaurant-1','device-1','shift-1','employee-1','cash_out',-1,?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected negative cash drawer event amount to fail")
	}
}

func TestPaymentsRequireEdgeContextAndPaymentAttemptsReferencePayment(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO prechecks(id,order_id,status,version,subtotal,discount_total,tax_total,total,paid_total,snapshot,created_at,issued_at) VALUES ('precheck-1','order-1','issued',1,100,0,0,100,0,'{}',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO payments(id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,business_date_local,created_at,updated_at) VALUES ('payment-1','edge-payment-1','restaurant-1','device-1','shift-1','precheck-1','cash',100,'RUB','captured','2026-05-04',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO payment_attempts(id,payment_id,attempt_no,method,amount,currency,status,attempted_at,created_at) VALUES ('attempt-1','payment-1',1,'cash',100,'RUB','captured',?,?)`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO payments(id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,business_date_local,created_at,updated_at) VALUES ('payment-2','edge-payment-2',NULL,'device-1','shift-1','precheck-1','cash',100,'RUB','captured','2026-05-04',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected payment without restaurant_id to fail")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO payment_attempts(id,payment_id,attempt_no,method,amount,currency,status,attempted_at,created_at) VALUES ('attempt-2','missing',1,'cash',100,'RUB','captured',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected payment attempt without payment to fail")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO payment_attempts(id,payment_id,attempt_no,method,amount,currency,status,attempted_at,created_at) VALUES ('attempt-3','payment-1',0,'cash',100,'RUB','captured',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected payment attempt with zero attempt_no to fail")
	}
}

func TestLocalEventLogRequiresUniqueEdgeEventIdentity(t *testing.T) {
	db, ctx := newSchemaDB(t)
	insert := `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,shift_id,payload_json,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "local-event-1", "edge-event-1", "cmd-1", "1", "OrderCreated", "Order", "order-1", "restaurant-1", "device-1", "device-1", "shift-1", `{"event_id":"edge-event-1"}`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "local-event-2", "edge-event-1", "cmd-2", "1", "OrderCreated", "Order", "order-1", "restaurant-1", "device-1", "device-1", "shift-1", `{"event_id":"edge-event-1"}`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected duplicate event_id to fail")
	}
}

func TestLocalEventLogAllowsMultipleEventsForOneCommandID(t *testing.T) {
	db, ctx := newSchemaDB(t)
	insert := `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,shift_id,payload_json,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "local-event-1", "edge-event-1", "cmd-1", "1", "OrderCreated", "Order", "order-1", "restaurant-1", "device-1", "device-1", "shift-1", `{"command_id":"cmd-1"}`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, insert, "local-event-2", "edge-event-2", "cmd-1", "1", "CheckCreated", "Check", "check-1", "restaurant-1", "device-1", "device-1", "shift-1", `{"command_id":"cmd-1"}`, schemaTestTime, schemaTestTime)
}

func TestLocalEventLogRequiresDeviceID(t *testing.T) {
	db, ctx := newSchemaDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,shift_id,payload_json,occurred_at,created_at) VALUES ('local-event-1','edge-event-1','cmd-1','1','OrderCreated','Order','order-1','restaurant-1','','','shift-1','{}',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected empty device_id to fail")
	}
}

func TestActiveSQLiteMigrationPathUsesSingleManagedCanonicalFile(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "..", "..", "migrations", "sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			migrations = append(migrations, entry.Name())
		}
	}
	if len(migrations) != 1 || migrations[0] != "001_init.sql" {
		t.Fatalf("expected single managed sqlite baseline migration file, got %+v", migrations)
	}
}

func TestCleanInstallPaymentsUsePrecheckIDWithoutLegacyCheckID(t *testing.T) {
	db, ctx := newSchemaDB(t)
	columns := tableColumns(t, ctx, db, "payments")
	if !columns["precheck_id"] {
		t.Fatal("expected payments.precheck_id in first-launch schema")
	}
	if columns["check_id"] {
		t.Fatal("did not expect payments.check_id in first-launch schema")
	}
}

func TestCleanInstallRecordsCanonicalInitMigration(t *testing.T) {
	db, ctx := newSchemaDB(t)
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected one managed migration row, got %d", n)
	}
	rows, err := db.QueryContext(ctx, `SELECT version, checksum_sha256, status FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var versions []string
	for rows.Next() {
		var version, checksum, status string
		if err := rows.Scan(&version, &checksum, &status); err != nil {
			t.Fatal(err)
		}
		if len(checksum) != 64 {
			t.Fatalf("expected stored checksum for %s, got %q", version, checksum)
		}
		if status != "applied" {
			t.Fatalf("expected migration %s status applied, got %s", version, status)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0] != "001_init.sql" {
		t.Fatalf("expected collapsed baseline migration history, got %+v", versions)
	}
}

func TestRequiredSchemaContractMatchesCleanInstall(t *testing.T) {
	db, ctx := newSchemaDB(t)
	if err := platformsqlite.VerifySchema(ctx, db, possqlite.RequiredSchema()); err != nil {
		t.Fatalf("required schema contract does not match clean install: %v", err)
	}
}

func TestRuntimeSchemaRepairMigratesLegacyBusinessDateColumns(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-pos.db")
	db, err := platformsqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	execSchema(t, ctx, db, `
CREATE TABLE restaurants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  timezone TEXT NOT NULL,
  currency TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE shifts (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  opened_by_employee_id TEXT NOT NULL,
  closed_by_employee_id TEXT,
  status TEXT NOT NULL,
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  opening_cash_amount INTEGER NOT NULL,
  closing_cash_amount INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE prechecks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL,
  status TEXT NOT NULL,
  version INTEGER NOT NULL,
  supersedes_precheck_id TEXT,
  subtotal INTEGER NOT NULL,
  discount_total INTEGER NOT NULL,
  tax_total INTEGER NOT NULL,
  total INTEGER NOT NULL,
  paid_total INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  issued_at TEXT NOT NULL,
  closed_at TEXT,
  cancelled_by_employee_id TEXT,
  cancellation_reason TEXT
);
CREATE TABLE checks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL,
  subtotal INTEGER NOT NULL,
  discount_total INTEGER NOT NULL,
  tax_total INTEGER NOT NULL,
  total INTEGER NOT NULL,
  paid_total INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE payments (
  id TEXT PRIMARY KEY,
  edge_payment_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  shift_id TEXT NOT NULL,
  precheck_id TEXT NOT NULL,
  method TEXT NOT NULL,
  amount INTEGER NOT NULL,
  currency TEXT NOT NULL,
  status TEXT NOT NULL,
  provider_name TEXT,
  provider_transaction_id TEXT,
  provider_reference TEXT,
  fingerprint_hash TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE cash_sessions (
  id TEXT PRIMARY KEY,
  edge_cash_session_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  shift_id TEXT NOT NULL,
  opened_by_employee_id TEXT NOT NULL,
  closed_by_employee_id TEXT,
  status TEXT NOT NULL,
  opening_cash_amount INTEGER NOT NULL,
  closing_cash_amount INTEGER,
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
`)

	if err := platformsqlite.MigrateDirWithPolicy(ctx, db, dbPath, filepath.Join("..", "..", "..", "..", "migrations", "sqlite"), platformsqlite.MigrationOptions{
		ModuleName:         "pos-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: possqlite.RequiredSchema(),
	}); err != nil {
		t.Fatalf("legacy sqlite repair migration failed: %v", err)
	}
	for table, columns := range map[string][]string{
		"restaurants":   {"business_day_mode", "business_day_boundary_local_time"},
		"shifts":        {"business_date_local"},
		"prechecks":     {"snapshot"},
		"checks":        {"business_date_local", "closed_at", "snapshot"},
		"payments":      {"business_date_local"},
		"cash_sessions": {"business_date_local"},
	} {
		found := tableColumns(t, ctx, db, table)
		for _, column := range columns {
			if !found[column] {
				t.Fatalf("expected repair migration to add %s.%s", table, column)
			}
		}
	}
	if err := platformsqlite.MigrateDirWithPolicy(ctx, db, dbPath, filepath.Join("..", "..", "..", "..", "migrations", "sqlite"), platformsqlite.MigrationOptions{
		ModuleName:         "pos-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: possqlite.RequiredSchema(),
	}); err != nil {
		t.Fatalf("second legacy sqlite repair migration failed: %v", err)
	}
	var appliedCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE status = 'applied'`).Scan(&appliedCount); err != nil {
		t.Fatal(err)
	}
	if appliedCount != 1 {
		t.Fatalf("expected second startup to keep one applied migration, got %d", appliedCount)
	}
}

func TestRetrySafeOutboxSchemaColumnsAndConstraints(t *testing.T) {
	db, ctx := newSchemaDB(t)
	expected := map[string]bool{
		"sequence_no":    false,
		"sync_direction": false,
		"attempts":       false,
		"next_retry_at":  false,
		"locked_at":      false,
		"locked_by":      false,
		"sent_at":        false,
		"last_error":     false,
	}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(pos_sync_outbox)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("expected pos_sync_outbox column %s", name)
		}
	}

	execSchema(t, ctx, db, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,node_device_id,aggregate_type,aggregate_id,command_type,payload_json,status,created_at,updated_at) VALUES ('outbox-ok','cmd-ok',1,'edge_device','restaurant-1','device-1','device-1','Order','order-1','OrderCreated','{}','pending',?,?)`, schemaTestTime, schemaTestTime)
	var attempts int
	if err := db.QueryRowContext(ctx, `SELECT attempts FROM pos_sync_outbox WHERE id = 'outbox-ok'`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatalf("expected default attempts=0, got %d", attempts)
	}
	var syncDirection string
	if err := db.QueryRowContext(ctx, `SELECT sync_direction FROM pos_sync_outbox WHERE id = 'outbox-ok'`).Scan(&syncDirection); err != nil {
		t.Fatal(err)
	}
	if syncDirection != "edge_to_cloud" {
		t.Fatalf("expected default sync_direction=edge_to_cloud, got %s", syncDirection)
	}

	_, err = db.ExecContext(ctx, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,node_device_id,aggregate_type,aggregate_id,command_type,payload_json,status,created_at,updated_at) VALUES ('outbox-bad-status','cmd-bad-status',2,'edge_device','restaurant-1','device-1','device-1','Order','order-1','OrderCreated','{}','unknown',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected invalid outbox status to fail")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,node_device_id,aggregate_type,aggregate_id,command_type,payload_json,status,locked_at,locked_by,created_at,updated_at) VALUES ('outbox-bad-lock','cmd-bad-lock',3,'edge_device','restaurant-1','device-1','device-1','Order','order-1','OrderCreated','{}','pending',?,'worker',?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected non-processing outbox lock to fail")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,node_device_id,aggregate_type,aggregate_id,command_type,payload_json,status,created_at,updated_at) VALUES ('outbox-bad-sequence','cmd-bad-sequence',0,'edge_device','restaurant-1','device-1','device-1','Order','order-1','OrderCreated','{}','pending',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected non-positive sequence_no to fail")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,node_device_id,aggregate_type,aggregate_id,command_type,sync_direction,payload_json,status,created_at,updated_at) VALUES ('outbox-bad-direction','cmd-bad-direction',4,'edge_device','restaurant-1','device-1','device-1','Order','order-1','OrderCreated','sideways','{}','pending',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected invalid sync_direction to fail")
	}
}

func TestCloudMasterDataSyncFoundationSchema(t *testing.T) {
	db, ctx := newSchemaDB(t)
	for _, table := range []string{"restaurants", "devices", "roles", "employees", "halls", "tables", "catalog_items", "menu_items", "tax_profiles", "tax_rules", "service_charge_rules", "recipe_versions", "recipe_lines", "stop_lists"} {
		columns := tableColumns(t, ctx, db, table)
		for _, column := range []string{"cloud_version", "cloud_updated_at", "cloud_deleted_at", "last_synced_at"} {
			if !columns[column] {
				t.Fatalf("expected %s.%s for Cloud -> Edge master sync metadata", table, column)
			}
		}
	}

	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'cloud_master_sync_state'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("expected cloud_master_sync_state table to exist")
	}
	execSchema(t, ctx, db, `INSERT INTO cloud_master_sync_state(id,restaurant_id,node_device_id,stream_name,direction,sync_mode,last_cloud_version,status,created_at,updated_at) VALUES ('sync-state-1','restaurant-1','device-1','menu','cloud_to_edge','full_snapshot',1,'applied',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO cloud_master_sync_state(id,restaurant_id,node_device_id,stream_name,direction,sync_mode,last_cloud_version,status,created_at,updated_at) VALUES ('sync-state-pricing','restaurant-1','device-1','pricing_policy','cloud_to_edge','incremental',2,'applied',?,?)`, schemaTestTime, schemaTestTime)
	_, err := db.ExecContext(ctx, `INSERT INTO cloud_master_sync_state(id,node_device_id,stream_name,direction,sync_mode,status,created_at,updated_at) VALUES ('sync-state-bad','device-1','menu','edge_to_cloud','incremental','applied',?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected Cloud master sync state to reject non cloud_to_edge direction")
	}
}

func tableColumns(t *testing.T, ctx context.Context, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func TestRecipeVersionReferencesDishCatalogItem(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)

	_, err := db.ExecContext(ctx, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES ('recipe-bad','good-1',1,'Bad','draft',1,'portion',1,?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected recipe version for non-dish catalog item to fail")
	}
	execSchema(t, ctx, db, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES ('recipe-1','dish-1',1,'Soup v1','draft',1,'portion',1,?,?)`, schemaTestTime, schemaTestTime)
}

func TestRecipeLinesReferenceGoodOrSemiFinishedCatalogItems(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES ('recipe-1','dish-1',1,'Soup v1','draft',1,'portion',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES ('recipe-line-1','recipe-1','semi-finished-1',100,'g',0,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES ('recipe-line-2','recipe-1','good-1',1,'pcs',0,?,?)`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES ('recipe-line-bad','recipe-1','dish-1',1,'portion',0,?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected recipe line for dish catalog item to fail")
	}
}

func TestStopListsFoundation(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO stop_lists(id,restaurant_id,catalog_item_id,available_quantity,source,reason,active,cloud_version,updated_at) VALUES ('stop-1','restaurant-1','dish-1',NULL,'cloud','maintenance',1,10,?)`, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO stop_lists(id,restaurant_id,catalog_item_id,available_quantity,source,reason,active,cloud_version,updated_at) VALUES ('stop-2','restaurant-1','good-1',0,'edge',NULL,1,NULL,?)`, schemaTestTime)
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM stop_lists WHERE active = 1`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 active stop-list rows, got %d", n)
	}
}
