package sqlite_test

import (
	"slices"
	"testing"

	possqlite "pos-backend/internal/pos/infra/sqlite"
)

// POS-85: managed baseline создает Edge data model для точек продаж,
// секций ресторана, назначений печати, audit override и per-printer targets.
func TestPrintRoutingSchemaAndVerificationContract(t *testing.T) {
	db, ctx := newSchemaDB(t)
	seedCatalogForSchemaTests(t, ctx, db)
	execSchema(t, ctx, db, `INSERT INTO halls(id,restaurant_id,name,active,created_at,updated_at)
VALUES ('hall-1','restaurant-1','Main',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO receipt_printers(id,restaurant_id,name,type,address,port,document_types,codepage,paper_cut_type,cpl,is_active,cloud_version,synced_at)
VALUES ('printer-1','restaurant-1','Cash printer','tcp','127.0.0.1',9100,'["precheck","check_nonfiscal"]','','partial',48,1,1,?)`, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO restaurant_sections(id,restaurant_id,name,mode,hall_id,is_default,created_at,updated_at)
VALUES ('section-1','restaurant-1','Main hall','hall_section','hall-1',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO restaurant_sections(id,restaurant_id,name,mode,kitchen_routing_key,created_at,updated_at)
VALUES ('section-2','restaurant-1','Hot workshop','kitchen_workshop','hot',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO tables(id,restaurant_id,hall_id,section_id,name,seats,is_default,active,created_at,updated_at)
VALUES ('table-1','restaurant-1','hall-1','section-1','A1',2,1,1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO sales_points(id,restaurant_id,name,analytics_tag,default_table_id,is_active,created_at,updated_at)
VALUES ('sales-point-1','restaurant-1','Front cashier','front','table-1',1,?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO print_routes(id,restaurant_id,document_type,scope_type,scope_id,printer_id,origin,created_at,updated_at)
VALUES ('route-1','restaurant-1','check_nonfiscal','sales_point','sales-point-1','printer-1','edge_override',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO printer_route_override_audit(id,restaurant_id,action,route_id,scope_type,scope_id,document_type,after_json,occurred_at,created_at)
VALUES ('audit-1','restaurant-1','create','route-1','sales_point','sales-point-1','check_nonfiscal','{"printer_id":"printer-1"}',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO print_jobs(id,restaurant_id,document_type,source_kind,source_id,scope_id,status,attempts,max_attempts,printer_class,created_at,updated_at)
VALUES ('job-1','restaurant-1','check_nonfiscal','check','check-1','sales-point-1','pending',0,3,'generic',?,?)`, schemaTestTime, schemaTestTime)
	execSchema(t, ctx, db, `INSERT INTO print_job_targets(id,print_job_id,restaurant_id,printer_id,scope_type,scope_id,status,created_at,updated_at)
VALUES ('target-1','job-1','restaurant-1','printer-1','sales_point','sales-point-1','pending',?,?)`, schemaTestTime, schemaTestTime)

	var targetStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM print_job_targets WHERE id = 'target-1'`).Scan(&targetStatus); err != nil {
		t.Fatal(err)
	}
	if targetStatus != "pending" {
		t.Fatalf("expected pending print target, got %q", targetStatus)
	}

	expected := map[string]struct {
		columns []string
		indexes []string
	}{
		"sales_points": {
			columns: []string{"id", "restaurant_id", "name", "analytics_tag", "default_table_id", "is_active"},
			indexes: []string{"sales_points_restaurant_active"},
		},
		"restaurant_sections": {
			columns: []string{"id", "restaurant_id", "name", "mode", "hall_id", "kitchen_routing_key", "is_default"},
			indexes: []string{"restaurant_sections_restaurant_mode_active", "restaurant_sections_one_default_hall_section"},
		},
		"tables": {
			columns: []string{"section_id", "is_default"},
			indexes: []string{"tables_restaurant_section", "tables_one_default_per_restaurant"},
		},
		"print_routes": {
			columns: []string{"id", "restaurant_id", "document_type", "scope_type", "scope_id", "printer_id", "origin"},
			indexes: []string{"print_routes_scope_active", "print_routes_printer_active", "print_routes_unique_active_printer_scope"},
		},
		"printer_route_override_audit": {
			columns: []string{"id", "restaurant_id", "action", "scope_type", "document_type", "before_json", "after_json"},
			indexes: []string{"printer_route_override_audit_restaurant_created", "printer_route_override_audit_outbox_command"},
		},
		"print_job_targets": {
			columns: []string{"id", "print_job_id", "restaurant_id", "printer_id", "scope_type", "scope_id", "status", "attempts", "max_attempts", "is_required"},
			indexes: []string{"print_job_targets_pending_due", "print_job_targets_job_status", "print_job_targets_restaurant_status_created", "print_job_targets_unique_printer_scope"},
		},
		"print_jobs": {
			columns: []string{"scope_id"},
		},
	}
	for table, want := range expected {
		assertSchemaRequirement(t, table, want.columns, want.indexes)
	}
	assertSchemaRequirement(t, "cash_sessions", []string{"sales_point_id"}, nil)
}

func assertSchemaRequirement(t *testing.T, table string, columns []string, indexes []string) {
	t.Helper()
	for _, req := range possqlite.RequiredSchema() {
		if req.Table != table {
			continue
		}
		for _, column := range columns {
			if !slices.Contains(req.Columns, column) {
				t.Fatalf("expected %s.%s in schema verification contract", table, column)
			}
		}
		for _, index := range indexes {
			if !slices.Contains(req.Indexes, index) {
				t.Fatalf("expected %s index in schema verification contract for %s", index, table)
			}
		}
		return
	}
	t.Fatalf("expected %s in schema verification contract", table)
}
