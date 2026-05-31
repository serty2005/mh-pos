package clickhouse_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud-backend/internal/olap/app"
	"cloud-backend/internal/olap/infra/clickhouse"
)

func TestListStockMovesAcceptsQuotedClickHouseInt64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ledger_entry_id":"ledger-1","restaurant_id":"rest-1","warehouse_id":"warehouse-main","stock_document_id":"doc-1","source_event_id":"event-1","source_event_type":"ItemServed","catalog_item_id":"item-1","order_line_id":"line-1","movement_type":"OUT","quantity":"1.000","unit_code":"portion","unit_cost_minor":"125","total_cost_minor":"125","costing_status":"estimated","occurred_at":"2026-05-29 10:00:00.000","business_date_local":"2026-05-29","ledger_created_at":"2026-05-29 10:00:01.000"}` + "\n"))
	}))
	defer server.Close()

	repo := clickhouse.NewRepository(clickhouse.Config{URL: server.URL, Database: "mh_pos_cloud"})
	items, err := repo.ListStockMoves(context.Background(), app.StockMoveFilter{
		RestaurantID:    "rest-1",
		SourceEventType: "ItemServed",
		Limit:           10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UnitCostMinor != 125 || items[0].TotalCostMinor != 125 {
		t.Fatalf("unexpected stock move rows: %+v", items)
	}
}

func TestListStockMoveSummaryUsesFiltersGroupingAndStableLimit(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		query = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"group_by":"catalog_item","group_key":"item-1","business_date_local":"","catalog_item_id":"item-1","warehouse_id":"","move_count":"2","in_quantity":"3.000","out_quantity":"1.000","net_quantity":"2.000","total_cost_minor":"1200","first_occurred_at":"2026-05-29 10:00:00.000","last_occurred_at":"2026-05-29 11:00:00.000"}` + "\n"))
	}))
	defer server.Close()

	repo := clickhouse.NewRepository(clickhouse.Config{URL: server.URL, Database: "mh_pos_cloud"})
	items, err := repo.ListStockMoveSummary(context.Background(), app.StockMoveSummaryFilter{
		RestaurantID:     "rest-1",
		BusinessDateFrom: "2026-05-01",
		BusinessDateTo:   "2026-05-29",
		CatalogItemID:    "item-1",
		WarehouseID:      "warehouse-main",
		SourceEventType:  "StockReceiptCaptured",
		GroupBy:          "catalog_item",
		Limit:            10,
		Offset:           5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GroupKey != "item-1" || items[0].MoveCount != 2 {
		t.Fatalf("unexpected summary rows: %+v", items)
	}
	for _, want := range []string{
		"catalog_item_id AS group_key",
		"restaurant_id = 'rest-1'",
		"business_date_local >= toDate('2026-05-01')",
		"warehouse_id = 'warehouse-main'",
		"source_event_type = 'StockReceiptCaptured'",
		"GROUP BY catalog_item_id ORDER BY catalog_item_id ASC LIMIT 10 OFFSET 5",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected query to contain %q, got %s", want, query)
		}
	}
	if strings.Contains(query, "payload") {
		t.Fatalf("stock summary query must not select raw payload: %s", query)
	}
}

func TestListSalesKitchenSummaryUsesExistingDatasetsAndSafeGrouping(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		query = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"group_by":"source_event_type","group_key":"ItemServed","business_date_local":"","event_type":"","source_event_type":"ItemServed","catalog_item_id":"","event_count":"3","stock_move_count":"2","sale_event_count":"0","kitchen_event_count":"3","in_quantity":"0.000","out_quantity":"2.000","net_quantity":"-2.000","total_cost_minor":"250","first_occurred_at":"2026-05-29 10:00:00.000","last_occurred_at":"2026-05-29 11:00:00.000"}` + "\n"))
	}))
	defer server.Close()

	repo := clickhouse.NewRepository(clickhouse.Config{URL: server.URL, Database: "mh_pos_cloud"})
	items, err := repo.ListSalesKitchenSummary(context.Background(), app.SalesKitchenSummaryFilter{
		RestaurantID:     "rest-1",
		BusinessDateFrom: "2026-05-01",
		BusinessDateTo:   "2026-05-29",
		GroupBy:          "source_event_type",
		Limit:            10,
		Offset:           5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GroupKey != "ItemServed" || items[0].EventCount != 3 || items[0].StockMoveCount != 2 {
		t.Fatalf("unexpected sales/kitchen summary rows: %+v", items)
	}
	for _, want := range []string{
		".raw_business_events FINAL",
		".olap_stock_moves FINAL",
		"event_type IN ('KitchenTicketStatusChanged','ItemServed'",
		"source_event_type AS group_key",
		"restaurant_id = 'rest-1'",
		"toDate(occurred_at) >= toDate('2026-05-01')",
		"business_date_local >= toDate('2026-05-01')",
		"GROUP BY group_by, group_key ORDER BY group_key ASC LIMIT 10 OFFSET 5",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected query to contain %q, got %s", want, query)
		}
	}
	if strings.Contains(query, " payload") || strings.Contains(query, "margin") || strings.Contains(query, "COGS") {
		t.Fatalf("sales/kitchen summary query must not select raw payload or BI costing labels: %s", query)
	}
}
