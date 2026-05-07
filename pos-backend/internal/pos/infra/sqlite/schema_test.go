package sqlite_test

import (
	"context"
	"database/sql"
	"os"
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

func seedFinancialForSchemaTests(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO roles(id,name,permissions_json,active,created_at,updated_at) VALUES ('role-1','cashier','{}',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO employees(id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at) VALUES ('employee-1','restaurant-1','role-1','Anna','hash',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO halls(id,restaurant_id,name,active,created_at,updated_at) VALUES ('hall-1','restaurant-1','Main',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO tables(id,restaurant_id,hall_id,name,seats,active,created_at,updated_at) VALUES ('table-1','restaurant-1','hall-1','A1',2,1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO tables(id,restaurant_id,hall_id,name,seats,active,created_at,updated_at) VALUES ('table-2','restaurant-1','hall-1','A2',2,1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO shifts(id,restaurant_id,device_id,opened_by_employee_id,status,opened_at,opening_cash_amount,created_at,updated_at) VALUES ('shift-1','restaurant-1','device-1','employee-1','open',?,0,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,created_at,updated_at) VALUES ('order-1','edge-order-1','restaurant-1','device-1','shift-1','open','table-1','A1',1,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO checks(id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,created_at,updated_at) VALUES ('check-1','order-1','open',100,0,0,100,0,?,?)`, schemaTestTime, schemaTestTime)
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
	insert := `INSERT INTO prechecks(id,order_id,status,subtotal,discount_total,tax_total,total,created_at,issued_at) VALUES (?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "precheck-1", "order-1", "issued", 100, 0, 0, 100, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "precheck-2", "order-1", "issued", 100, 0, 0, 100, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected second issued precheck for order to fail")
	}
}

func TestPrecheckTotalsMustMatchSnapshotFormula(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)

	_, err := db.ExecContext(ctx, `INSERT INTO prechecks(id,order_id,status,subtotal,discount_total,tax_total,total,created_at,issued_at) VALUES ('precheck-bad','order-1','issued',100,10,5,100,?,?)`, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected inconsistent precheck total to fail")
	}
}

func TestPrecheckLifecycleFoundationConstraints(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	insert := `INSERT INTO prechecks(id,order_id,status,version,subtotal,discount_total,tax_total,total,paid_total,created_at,issued_at,closed_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "precheck-1", "order-1", "cancelled", 1, 100, 0, 0, 100, 0, schemaTestTime, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, insert, "precheck-2", "order-1", "superseded", 2, 100, 0, 0, 100, 0, schemaTestTime, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "precheck-bad-version", "order-1", "cancelled", 0, 100, 0, 0, 100, 0, schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected non-positive precheck version to fail")
	}
	_, err = db.ExecContext(ctx, insert, "precheck-bad-paid-total", "order-1", "cancelled", 3, 100, 0, 0, 100, 101, schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected paid_total above total to fail")
	}
	_, err = db.ExecContext(ctx, insert, "precheck-bad-terminal-time", "order-1", "cancelled", 4, 100, 0, 0, 100, 0, schemaTestTime, schemaTestTime, nil)
	if err == nil {
		t.Fatal("expected terminal precheck without closed_at to fail")
	}
}

func TestCashSessionsAllowOnlyOneOpenSessionPerDevice(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	insert := `INSERT INTO cash_sessions(id,edge_cash_session_id,restaurant_id,device_id,shift_id,opened_by_employee_id,status,opening_cash_amount,opened_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	execSchema(t, ctx, db, insert, "cash-session-1", "edge-cash-session-1", "restaurant-1", "device-1", "shift-1", "employee-1", "open", 100, schemaTestTime, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, insert, "cash-session-2", "edge-cash-session-2", "restaurant-1", "device-1", "shift-1", "employee-1", "open", 200, schemaTestTime, schemaTestTime, schemaTestTime)
	if err == nil {
		t.Fatal("expected second open cash session on device to fail")
	}
}

func TestCashDrawerEventsRequireSessionAndNonNegativeAmount(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedFinancialForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO cash_sessions(id,edge_cash_session_id,restaurant_id,device_id,shift_id,opened_by_employee_id,status,opening_cash_amount,opened_at,created_at,updated_at) VALUES ('cash-session-1','edge-cash-session-1','restaurant-1','device-1','shift-1','employee-1','open',100,?,?,?)`, schemaTestTime, schemaTestTime, schemaTestTime)
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
	execSchema(t, ctx, db, `INSERT INTO prechecks(id,order_id,status,version,subtotal,discount_total,tax_total,total,paid_total,created_at,issued_at) VALUES ('precheck-1','order-1','issued',1,100,0,0,100,0,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO payments(id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,created_at,updated_at) VALUES ('payment-1','edge-payment-1','restaurant-1','device-1','shift-1','precheck-1','cash',100,'RUB','captured',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO payment_attempts(id,payment_id,attempt_no,method,amount,currency,status,attempted_at,created_at) VALUES ('attempt-1','payment-1',1,'cash',100,'RUB','captured',?,?)`, schemaTestTime, schemaTestTime)

	_, err := db.ExecContext(ctx, `INSERT INTO payments(id,edge_payment_id,restaurant_id,device_id,shift_id,precheck_id,method,amount,currency,status,created_at,updated_at) VALUES ('payment-2','edge-payment-2',NULL,'device-1','shift-1','precheck-1','cash',100,'RUB','captured',?,?)`, schemaTestTime, schemaTestTime)
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

func TestActiveSQLiteMigrationPathIsSingleCanonicalFirstLaunchInit(t *testing.T) {
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
		t.Fatalf("expected only canonical first-launch 001_init.sql, got %+v", migrations)
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

func TestCleanInstallRecordsOnlyCanonicalInitMigration(t *testing.T) {
	db, ctx := newSchemaDB(t)
	var version string
	if err := db.QueryRowContext(ctx, `SELECT version FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != "001_init.sql" {
		t.Fatalf("expected schema_migrations to contain 001_init.sql, got %q", version)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected one applied first-launch migration, got %d", n)
	}
}

func TestRetrySafeOutboxSchemaColumnsAndConstraints(t *testing.T) {
	db, ctx := newSchemaDB(t)
	expected := map[string]bool{
		"sequence_no":   false,
		"attempts":      false,
		"next_retry_at": false,
		"locked_at":     false,
		"locked_by":     false,
		"sent_at":       false,
		"last_error":    false,
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
