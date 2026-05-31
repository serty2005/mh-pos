package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/olap/api"
	"cloud-backend/internal/olap/app"
)

func TestListStockMovesReturnsSafeProjectionWithoutRawPayload(t *testing.T) {
	repo := &repo{
		stockMoves: []app.StockMove{{
			LedgerEntryID:     "ledger-1",
			RestaurantID:      "rest-1",
			WarehouseID:       "warehouse-main",
			StockDocumentID:   "doc-1",
			SourceEventID:     "event-1",
			SourceEventType:   "StockReceiptCaptured",
			CatalogItemID:     "item-1",
			MovementType:      "IN",
			Quantity:          "1.000",
			UnitCode:          "PC",
			CostingStatus:     "estimated",
			OccurredAt:        time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC),
			BusinessDateLocal: "2026-05-29",
			LedgerCreatedAt:   time.Date(2026, 5, 29, 10, 0, 1, 0, time.UTC),
		}},
	}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewService(repo))

	req := httptest.NewRequest(http.MethodGet, "/olap/stock-moves?restaurant_id=rest-1&business_date_from=2026-05-01&limit=500", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "payload") {
		t.Fatalf("stock moves response must not expose raw payload: %s", rec.Body.String())
	}
	var items []app.StockMove
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].LedgerEntryID != "ledger-1" {
		t.Fatalf("unexpected stock moves response: %+v", items)
	}
	if repo.stockFilter.Limit != 50 {
		t.Fatalf("expected bounded route limit, got %d", repo.stockFilter.Limit)
	}
}

func TestListStockMoveSummaryReturnsBoundedAggregateWithoutRawPayload(t *testing.T) {
	repo := &repo{
		summaries: []app.StockMoveSummary{{
			GroupBy:        "catalog_item",
			GroupKey:       "item-1",
			CatalogItemID:  "item-1",
			MoveCount:      2,
			InQuantity:     "3.000",
			OutQuantity:    "1.000",
			NetQuantity:    "2.000",
			TotalCostMinor: 1200,
		}},
	}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewService(repo))

	req := httptest.NewRequest(http.MethodGet, "/olap/stock-move-summary?restaurant_id=rest-1&group_by=catalog_item&limit=500", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "payload") || strings.Contains(rec.Body.String(), "COGS") {
		t.Fatalf("stock summary response must not expose raw payload or costing BI labels: %s", rec.Body.String())
	}
	var items []app.StockMoveSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GroupKey != "item-1" {
		t.Fatalf("unexpected stock summary response: %+v", items)
	}
	if repo.summaryFilter.Limit != 50 || repo.summaryFilter.GroupBy != "catalog_item" {
		t.Fatalf("expected bounded summary route filter, got %+v", repo.summaryFilter)
	}
}

func TestListSalesKitchenSummaryReturnsSafeBoundedAggregate(t *testing.T) {
	repo := &repo{
		salesKitchenSummaries: []app.SalesKitchenSummary{{
			GroupBy:           "business_date",
			GroupKey:          "2026-05-31",
			BusinessDateLocal: "2026-05-31",
			EventCount:        12,
			StockMoveCount:    4,
			SaleEventCount:    3,
			KitchenEventCount: 5,
			OutQuantity:       "4.000",
			InQuantity:        "0.000",
			NetQuantity:       "-4.000",
			TotalCostMinor:    12345,
		}},
	}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewService(repo))

	req := httptest.NewRequest(http.MethodGet, "/olap/sales-kitchen-summary?restaurant_id=rest-1&business_date_from=2026-05-01&group_by=business_date&limit=500", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "payload") || strings.Contains(rec.Body.String(), "margin") || strings.Contains(rec.Body.String(), "COGS") {
		t.Fatalf("sales/kitchen summary response must not expose raw payload or BI costing labels: %s", rec.Body.String())
	}
	var items []app.SalesKitchenSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GroupKey != "2026-05-31" || items[0].EventCount != 12 {
		t.Fatalf("unexpected sales/kitchen summary response: %+v", items)
	}
	if repo.salesKitchenFilter.Limit != 50 || repo.salesKitchenFilter.GroupBy != "business_date" {
		t.Fatalf("expected bounded sales/kitchen route filter, got %+v", repo.salesKitchenFilter)
	}
}

func TestGetExportStatusReturnsSafeState(t *testing.T) {
	repo := &repo{status: app.ExportStatus{Stream: "raw_business_events", PendingCount: 3, FailedCount: 1, LastError: "clickhouse down"}}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewServiceWithExportStatus(repo, repo))

	req := httptest.NewRequest(http.MethodGet, "/olap/export-status?stream=raw_business_events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "payload") {
		t.Fatalf("export status response must not expose raw payload: %s", rec.Body.String())
	}
	var status app.ExportStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Stream != "raw_business_events" || status.PendingCount != 3 || repo.statusStream != "raw_business_events" {
		t.Fatalf("unexpected status response=%+v stream=%q", status, repo.statusStream)
	}
}

func TestPostExportRetryReturnsAcceptedSafeResult(t *testing.T) {
	repo := &repo{retryResult: app.ExportRetryResult{
		CommandID:        "018f0000-0000-7000-8000-000000000211",
		Stream:           "raw_business_events",
		Mode:             "retry_failed",
		Accepted:         true,
		CheckpointBefore: "receipt-10",
		PendingCount:     4,
		FailedCount:      0,
	}}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewServiceWithControls(repo, repo, repo))

	body := `{"command_id":"018f0000-0000-7000-8000-000000000211","stream":"raw_business_events","mode":"retry_failed","reason":"operator retry after ClickHouse outage"}`
	req := httptest.NewRequest(http.MethodPost, "/olap/export-retry", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "payload") || strings.Contains(rec.Body.String(), "operator retry") {
		t.Fatalf("export retry response must not expose raw payload or operator reason: %s", rec.Body.String())
	}
	var result app.ExportRetryResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.CommandID != repo.retryCommand.CommandID || result.PendingCount != 4 || result.FailedCount != 0 {
		t.Fatalf("unexpected retry result=%+v command=%+v", result, repo.retryCommand)
	}
}

func TestListStockMoveSummaryEmptyStateReturnsEmptyArray(t *testing.T) {
	repo := &repo{}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewService(repo))

	req := httptest.NewRequest(http.MethodGet, "/olap/stock-move-summary?group_by=business_date", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("expected empty array, got %s", rec.Body.String())
	}
}

type repo struct {
	stockMoves            []app.StockMove
	summaries             []app.StockMoveSummary
	salesKitchenSummaries []app.SalesKitchenSummary
	status                app.ExportStatus
	retryResult           app.ExportRetryResult
	stockFilter           app.StockMoveFilter
	summaryFilter         app.StockMoveSummaryFilter
	salesKitchenFilter    app.SalesKitchenSummaryFilter
	statusStream          string
	retryCommand          app.ExportRetryCommand
}

func (r *repo) ListRawBusinessEvents(context.Context, app.RawBusinessEventFilter) ([]app.RawBusinessEvent, error) {
	return nil, nil
}

func (r *repo) ListStockMoves(_ context.Context, filter app.StockMoveFilter) ([]app.StockMove, error) {
	r.stockFilter = filter
	return r.stockMoves, nil
}

func (r *repo) ListStockMoveSummary(_ context.Context, filter app.StockMoveSummaryFilter) ([]app.StockMoveSummary, error) {
	r.summaryFilter = filter
	return r.summaries, nil
}

func (r *repo) ListSalesKitchenSummary(_ context.Context, filter app.SalesKitchenSummaryFilter) ([]app.SalesKitchenSummary, error) {
	r.salesKitchenFilter = filter
	return r.salesKitchenSummaries, nil
}

func (r *repo) GetExportStatus(_ context.Context, stream string, _ time.Time) (app.ExportStatus, error) {
	r.statusStream = stream
	if r.status.Stream == "" {
		r.status.Stream = stream
	}
	return r.status, nil
}

func (r *repo) RequestExportRetry(_ context.Context, cmd app.ExportRetryCommand, now time.Time) (app.ExportRetryResult, error) {
	r.retryCommand = cmd
	if r.retryResult.CommandID == "" {
		r.retryResult = app.ExportRetryResult{
			CommandID:        cmd.CommandID,
			Stream:           cmd.Stream,
			Mode:             cmd.Mode,
			Accepted:         true,
			RetryRequestedAt: now,
		}
	}
	return r.retryResult, nil
}
