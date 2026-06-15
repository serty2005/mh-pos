package postgres

import (
	"context"
	"strings"
	"testing"

	"cloud-backend/internal/cloudsync/app"
)

func TestRequiredSchemaIncludesCurrencyReference(t *testing.T) { /* unchanged */
	var found bool
	for _, req := range RequiredSchema() {
		if req.Table != "cloud_currency_reference" {
			continue
		}
		found = true
		columns := map[string]bool{}
		for _, column := range req.Columns {
			columns[column] = true
		}
		for _, column := range []string{"currency_code", "currency_alpha_code", "minor_unit", "currency_iso_name", "currency_symbol", "curr_basic_name", "curr_add_name", "show_add", "show_currency_basic_name", "active"} {
			if !columns[column] {
				t.Fatalf("expected cloud_currency_reference.%s in schema verification contract", column)
			}
		}
	}
	if !found {
		t.Fatal("expected cloud_currency_reference in schema verification contract")
	}
}

func TestRequiredSchemaIncludesCloudInventoryFoundationTables(t *testing.T) {
	reqs := map[string]map[string]bool{}
	for _, req := range RequiredSchema() {
		cols := map[string]bool{}
		for _, c := range req.Columns {
			cols[c] = true
		}
		reqs[req.Table] = cols
	}
	for table, cols := range map[string][]string{
		"inventory_event_queue":                {"id", "receipt_id", "restaurant_id", "warehouse_id", "device_id", "event_id", "event_type", "status", "attempts", "occurred_at", "created_at", "updated_at"},
		"stock_documents":                      {"id", "restaurant_id", "warehouse_id", "document_type", "source_event_id", "source_event_type", "business_date_local", "occurred_at", "created_at"},
		"inventory_document_processing_state":  {"id", "restaurant_id", "source_event_id", "source_event_type", "source_aggregate_id", "stock_document_id", "status", "posted_ledger_count", "expected_ledger_count", "costing_status", "needs_recalculation", "failure_code", "failure_message_key", "created_at", "updated_at", "posted_at"},
		"stock_ledger":                         {"id", "restaurant_id", "warehouse_id", "stock_document_id", "source_event_id", "source_event_type", "catalog_item_id", "order_line_id", "movement_type", "quantity", "unit_code", "unit_cost_minor", "total_cost_minor", "costing_status", "occurred_at", "business_date_local", "created_at"},
		"inventory_stock_balances":             {"restaurant_id", "warehouse_id", "catalog_item_id", "unit_code", "quantity_on_hand", "last_movement_at", "last_ledger_entry_id", "costing_status", "needs_recalculation", "created_at", "updated_at"},
		"stock_recalculation_jobs":             {"id", "restaurant_id", "source_document_id", "trigger_type", "trigger_event_id", "trigger_command_id", "status", "business_date_from", "business_date_to", "affected_catalog_item_count", "affected_warehouse_count", "total_steps", "completed_steps", "failure_code", "failure_message_key", "created_at", "started_at", "finished_at", "updated_at"},
		"stock_recalculation_job_items":        {"job_id", "catalog_item_id", "warehouse_id", "unit_code", "business_date_from", "business_date_to", "created_at"},
		"stock_recalculation_edges":            {"job_id", "dependency_catalog_item_id", "dependent_catalog_item_id", "edge_type", "sort_order", "created_at"},
		"cloud_projection_stop_list_updates":   {"source_event_id", "queue_id", "restaurant_id", "device_id", "stop_list_id", "catalog_item_id", "available_quantity", "active", "conflict_policy", "source", "projection_action", "review_status", "review_comment", "reviewed_by_employee_id", "reviewed_at", "assigned_to_employee_id", "assigned_by_employee_id", "assigned_at", "assignment_note", "applied_stop_list_id", "updated_at", "occurred_at", "projected_at"},
		"stop_lists":                           {"id", "restaurant_id", "catalog_item_id", "available_quantity", "source", "reason", "active", "cloud_version", "updated_at"},
		"cloud_review_assignment_audit_events": {"event_id", "command_id", "review_type", "review_id", "action", "actor_employee_id", "target_employee_id", "reason", "occurred_at"},
	} {
		found, ok := reqs[table]
		if !ok {
			t.Fatalf("expected %s in schema verification contract", table)
		}
		for _, c := range cols {
			if !found[c] {
				t.Fatalf("expected %s.%s in schema verification contract", table, c)
			}
		}
	}
}

func TestRequiredSchemaIncludesCloudInventoryIndexes(t *testing.T) {
	reqs := map[string]map[string]bool{}
	for _, req := range RequiredSchema() {
		indexes := map[string]bool{}
		for _, index := range req.Indexes {
			indexes[index] = true
		}
		reqs[req.Table] = indexes
	}
	for table, indexes := range map[string][]string{
		"inventory_event_queue":               {"inventory_event_queue_status_retry", "inventory_event_queue_event_type", "inventory_event_queue_restaurant_warehouse_order"},
		"stock_documents":                     {"stock_documents_restaurant_occurred_at", "stock_documents_restaurant_warehouse_occurred_at", "stock_documents_source_event_unique"},
		"inventory_document_processing_state": {"inventory_document_processing_state_source_event_unique", "inventory_document_processing_state_restaurant_type_status", "inventory_document_processing_state_document"},
		"stock_ledger":                        {"stock_ledger_restaurant_occurred_at", "stock_ledger_restaurant_warehouse_occurred_at", "stock_ledger_source_event", "stock_ledger_order_line_consumption"},
		"inventory_stock_balances": {
			"inventory_stock_balances_pkey",
			"inventory_stock_balances_restaurant_warehouse_item",
			"inventory_stock_balances_restaurant_last_movement",
			"inventory_stock_balances_costing_status",
		},
		"stock_recalculation_jobs":      {"stock_recalculation_jobs_restaurant_status", "stock_recalculation_jobs_trigger_event_unique", "stock_recalculation_jobs_trigger_command_unique"},
		"stock_recalculation_job_items": {"stock_recalculation_job_items_pkey", "stock_recalculation_job_items_item"},
		"stock_recalculation_edges":     {"stock_recalculation_edges_pkey", "stock_recalculation_edges_job_order"},
	} {
		found, ok := reqs[table]
		if !ok {
			t.Fatalf("expected %s in schema verification contract", table)
		}
		for _, index := range indexes {
			if !found[index] {
				t.Fatalf("expected %s index in schema verification contract", index)
			}
		}
	}
}

func TestProcessingStateSchemaHasNoRawPayloadColumn(t *testing.T) {
	for _, req := range RequiredSchema() {
		if req.Table != "inventory_document_processing_state" {
			continue
		}
		for _, column := range req.Columns {
			if column == "raw_payload" || column == "payload_json" {
				t.Fatalf("processing state must not expose raw payload column: %+v", req.Columns)
			}
		}
		return
	}
	t.Fatal("expected inventory_document_processing_state in schema verification contract")
}

func TestRecalculationSchemaHasNoRawPayloadColumns(t *testing.T) {
	for _, req := range RequiredSchema() {
		switch req.Table {
		case "stock_recalculation_jobs", "stock_recalculation_job_items", "stock_recalculation_edges":
			for _, column := range req.Columns {
				if strings.Contains(column, "payload") || column == "raw_payload" || column == "payload_json" {
					t.Fatalf("%s must not expose payload column: %+v", req.Table, req.Columns)
				}
			}
		}
	}
}

func TestRequiredSchemaIncludesFinancialOperationProjection(t *testing.T) {
	var found bool
	for _, req := range RequiredSchema() {
		if req.Table != "cloud_projection_financial_operations" {
			continue
		}
		found = true
		columns := map[string]bool{}
		for _, column := range req.Columns {
			columns[column] = true
		}
		for _, column := range []string{"operation_id", "edge_operation_id", "event_id", "receipt_id", "restaurant_id", "device_id", "shift_id", "original_shift_id", "check_id", "precheck_id", "operation_type", "operation_kind", "amount", "currency", "business_date_local", "inventory_disposition", "reason", "snapshot_json", "operation_created_at", "cloud_received_at"} {
			if !columns[column] {
				t.Fatalf("expected cloud_projection_financial_operations.%s in schema verification contract", column)
			}
		}
	}
	if !found {
		t.Fatal("expected cloud_projection_financial_operations in schema verification contract")
	}
}

func TestRequiredSchemaIncludesOlapInboxAndCheckpointTables(t *testing.T) {
	reqs := map[string]map[string]bool{}
	indexes := map[string]map[string]bool{}
	for _, req := range RequiredSchema() {
		cols := map[string]bool{}
		for _, c := range req.Columns {
			cols[c] = true
		}
		idx := map[string]bool{}
		for _, name := range req.Indexes {
			idx[name] = true
		}
		reqs[req.Table] = cols
		indexes[req.Table] = idx
	}
	for _, column := range []string{"id", "receipt_id", "tenant_id", "restaurant_id", "device_id", "event_id", "event_type", "raw_payload", "processed_for_olap", "olap_export_status", "olap_export_attempts", "olap_next_retry_at", "olap_processed_at"} {
		if !reqs["inbox_events"][column] {
			t.Fatalf("expected inbox_events.%s in schema verification contract", column)
		}
	}
	for _, index := range []string{"inbox_events_event_unique", "inbox_events_olap_pending"} {
		if !indexes["inbox_events"][index] {
			t.Fatalf("expected %s index in schema verification contract", index)
		}
	}
	for _, column := range []string{"id", "last_exported_inbox_id", "last_exported_event_id", "last_exported_at", "last_error", "consecutive_failures", "next_retry_at", "updated_at"} {
		if !reqs["olap_export_checkpoints"][column] {
			t.Fatalf("expected olap_export_checkpoints.%s in schema verification contract", column)
		}
	}
	for _, column := range []string{"command_id", "stream", "mode", "reason", "accepted", "checkpoint_before", "retry_requested_at", "pending_count", "failed_count", "created_at"} {
		if !reqs["olap_export_retry_commands"][column] {
			t.Fatalf("expected olap_export_retry_commands.%s in schema verification contract", column)
		}
	}
	if !indexes["olap_export_retry_commands"]["olap_export_retry_commands_stream_created"] {
		t.Fatal("expected olap_export_retry_commands_stream_created index in schema verification contract")
	}
}

func TestCloudInventoryConstraintsRejectInvalidValues(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	if _, err := pool.Exec(ctx, `INSERT INTO stock_documents(id,restaurant_id,document_type,source_event_id,source_event_type,business_date_local,occurred_at,created_at) VALUES ('inv-doc-valid','rest-1','SALE','event-1','CheckClosed','2026-05-19',now(),now())`); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO stock_documents(id,restaurant_id,document_type,source_event_id,source_event_type,business_date_local,occurred_at,created_at) VALUES ('inv-doc-invalid','rest-1','BAD','event-2','CheckClosed','2026-05-19',now(),now())`,
		`INSERT INTO stock_ledger(id,restaurant_id,stock_document_id,source_event_id,source_event_type,catalog_item_id,movement_type,quantity,unit_code,unit_cost_minor,total_cost_minor,costing_status,occurred_at,business_date_local,created_at) VALUES ('ledger-bad-movement','rest-1','inv-doc-valid','event-3','CheckClosed','item-1','SIDE',1,'PC',10,10,'final',now(),'2026-05-19',now())`,
		`INSERT INTO stock_ledger(id,restaurant_id,stock_document_id,source_event_id,source_event_type,catalog_item_id,movement_type,quantity,unit_code,unit_cost_minor,total_cost_minor,costing_status,occurred_at,business_date_local,created_at) VALUES ('ledger-bad-costing','rest-1','inv-doc-valid','event-4','CheckClosed','item-1','OUT',1,'PC',10,10,'bad',now(),'2026-05-19',now())`,
		`INSERT INTO inventory_document_processing_state(id,restaurant_id,source_event_id,source_event_type,status,posted_ledger_count,costing_status,needs_recalculation,created_at,updated_at) VALUES ('state-bad-type','rest-1','event-5','CheckClosed','accepted',0,'estimated',false,now(),now())`,
		`INSERT INTO inventory_document_processing_state(id,restaurant_id,source_event_id,source_event_type,status,posted_ledger_count,costing_status,needs_recalculation,created_at,updated_at) VALUES ('state-bad-status','rest-1','event-6','StockReceiptCaptured','retrying',0,'estimated',false,now(),now())`,
		`INSERT INTO stock_recalculation_jobs(id,restaurant_id,source_document_id,trigger_type,status,business_date_from,business_date_to,created_at,updated_at) VALUES ('recalc-bad-status','rest-1','inv-doc-valid','StockReceiptCaptured','retrying','2026-05-19','2026-05-19',now(),now())`,
		`INSERT INTO stock_recalculation_edges(job_id,dependency_catalog_item_id,dependent_catalog_item_id,edge_type) VALUES ('missing-job','item-1','item-2','bad')`,
	} {
		if _, err := pool.Exec(ctx, q); err == nil {
			t.Fatal("expected constraint violation")
		}
	}
}

func TestListInventoryLedgerReadsBaselineDateAsText(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()

	if _, err := pool.Exec(ctx, `INSERT INTO stock_documents(id,restaurant_id,document_type,source_event_id,source_event_type,business_date_local,occurred_at,created_at) VALUES ('inv-doc-1','rest-1','SALE','event-check-closed','CheckClosed','2026-05-19',now(),now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO stock_ledger(id,restaurant_id,stock_document_id,source_event_id,source_event_type,catalog_item_id,order_line_id,movement_type,quantity,unit_code,unit_cost_minor,total_cost_minor,costing_status,occurred_at,business_date_local,created_at) VALUES ('ledger-1','rest-1','inv-doc-1','event-check-closed','CheckClosed','item-1','line-1','OUT',1,'PC',10,10,'estimated',now(),'2026-05-19',now())`); err != nil {
		t.Fatal(err)
	}

	items, err := NewRepository(pool).ListInventoryLedger(ctx, app.InventoryLedgerFilter{
		RestaurantID:    "rest-1",
		SourceEventType: "CheckClosed",
		OrderLineID:     "line-1",
		Limit:           10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].BusinessDateLocal != "2026-05-19" {
		t.Fatalf("unexpected ledger response: %+v", items)
	}
}

func TestListInventoryStockBalancesReadsMaterializedBaselineState(t *testing.T) {
	ctx := context.Background()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()

	if _, err := pool.Exec(ctx, `INSERT INTO stock_documents(id,restaurant_id,warehouse_id,document_type,source_event_id,source_event_type,business_date_local,occurred_at,created_at) VALUES ('inv-doc-balance','rest-1','warehouse-main','PURCHASE','event-receipt','StockReceiptCaptured','2026-05-19','2026-05-19T09:00:00Z','2026-05-19T09:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO stock_ledger(id,restaurant_id,warehouse_id,stock_document_id,source_event_id,source_event_type,catalog_item_id,movement_type,quantity,unit_code,unit_cost_minor,total_cost_minor,costing_status,occurred_at,business_date_local,created_at) VALUES ('ledger-balance','rest-1','warehouse-main','inv-doc-balance','event-receipt','StockReceiptCaptured','item-1','IN',2,'PC',10,20,'recalculated','2026-05-19T09:00:00Z','2026-05-19','2026-05-19T09:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO inventory_stock_balances(restaurant_id,warehouse_id,catalog_item_id,unit_code,quantity_on_hand,last_movement_at,last_ledger_entry_id,costing_status,needs_recalculation,created_at,updated_at) VALUES ('rest-1','warehouse-main','item-1','PC',2,'2026-05-19T09:00:00Z','ledger-balance','recalculated',false,'2026-05-19T09:00:00Z','2026-05-19T09:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	items, err := NewRepository(pool).ListInventoryStockBalances(ctx, app.InventoryStockBalanceFilter{
		RestaurantID:   "rest-1",
		WarehouseID:    "warehouse-main",
		CatalogItemID:  "item-1",
		BusinessDateTo: "2026-05-19",
		CostingStatus:  "recalculated",
		Limit:          10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].QuantityOnHand != "2.000" || items[0].CostingStatus != "recalculated" || items[0].BusinessDateTo != "2026-05-19" {
		t.Fatalf("unexpected materialized balance response: %+v", items)
	}
}
