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

func TestListKitchenTimingSummaryReturnsSafeBoundedAggregate(t *testing.T) {
	repo := &repo{
		kitchenTimingSummaries: []app.KitchenTimingSummary{{
			GroupBy:                 "station",
			GroupKey:                "hot",
			StationID:               "hot",
			TicketCount:             2,
			ReadyCount:              2,
			ServedCount:             1,
			AvgStartToReadySeconds:  420,
			AvgReadyToServedSeconds: 60,
		}},
	}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewService(repo))

	req := httptest.NewRequest(http.MethodGet, "/olap/kitchen-timing-summary?restaurant_id=rest-1&business_date_from=2026-05-01&group_by=station&limit=500", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "payload") || strings.Contains(rec.Body.String(), "margin") || strings.Contains(rec.Body.String(), "COGS") {
		t.Fatalf("kitchen timing response must not expose raw payload or costing labels: %s", rec.Body.String())
	}
	var items []app.KitchenTimingSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].GroupKey != "hot" || items[0].AvgStartToReadySeconds != 420 {
		t.Fatalf("unexpected kitchen timing response: %+v", items)
	}
	if repo.kitchenTimingFilter.Limit != 50 || repo.kitchenTimingFilter.GroupBy != "station" {
		t.Fatalf("expected bounded kitchen timing route filter, got %+v", repo.kitchenTimingFilter)
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

func TestBackfillJobRoutesReturnSafeOperatorState(t *testing.T) {
	repo := &repo{backfillJob: app.BackfillJob{
		ID:            "018f0000-0000-7000-8000-000000000411",
		CommandID:     "018f0000-0000-7000-8000-000000000411",
		Stream:        "raw_business_events",
		Status:        "queued",
		TotalRows:     10,
		ProcessedRows: 0,
		Reason:        "operator rebuild",
	}}
	router := chi.NewRouter()
	api.RegisterRoutes(router, app.NewServiceWithOperatorControls(repo, repo, repo, repo))

	body := `{"command_id":"018f0000-0000-7000-8000-000000000411","stream":"raw_business_events","reason":"operator rebuild","requested_by":"support"}`
	req := httptest.NewRequest(http.MethodPost, "/olap/backfill-jobs", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "payload") {
		t.Fatalf("backfill response must not expose raw payload: %s", rec.Body.String())
	}
	var job app.BackfillJob
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if job.ID != repo.backfillCreate.CommandID || job.Stream != "raw_business_events" {
		t.Fatalf("unexpected backfill create response=%+v command=%+v", job, repo.backfillCreate)
	}

	req = httptest.NewRequest(http.MethodGet, "/olap/backfill-jobs?stream=raw_business_events&limit=500", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.backfillFilter.Limit != 50 {
		t.Fatalf("expected bounded backfill list filter, got %+v", repo.backfillFilter)
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
	stockMoves             []app.StockMove
	summaries              []app.StockMoveSummary
	salesKitchenSummaries  []app.SalesKitchenSummary
	kitchenTimingSummaries []app.KitchenTimingSummary
	status                 app.ExportStatus
	retryResult            app.ExportRetryResult
	stockFilter            app.StockMoveFilter
	summaryFilter          app.StockMoveSummaryFilter
	salesKitchenFilter     app.SalesKitchenSummaryFilter
	kitchenTimingFilter    app.KitchenTimingSummaryFilter
	statusStream           string
	retryCommand           app.ExportRetryCommand
	backfillFilter         app.BackfillJobFilter
	backfillCreate         app.BackfillCreateCommand
	backfillCancel         app.BackfillCancelCommand
	backfillJob            app.BackfillJob
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

func (r *repo) ListKitchenTimingSummary(_ context.Context, filter app.KitchenTimingSummaryFilter) ([]app.KitchenTimingSummary, error) {
	r.kitchenTimingFilter = filter
	return r.kitchenTimingSummaries, nil
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

func (r *repo) ListBackfillJobs(_ context.Context, filter app.BackfillJobFilter) ([]app.BackfillJob, error) {
	r.backfillFilter = filter
	return []app.BackfillJob{r.backfillJob}, nil
}

func (r *repo) GetBackfillJob(_ context.Context, id string) (app.BackfillJob, error) {
	r.backfillJob.ID = id
	return r.backfillJob, nil
}

func (r *repo) CreateBackfillJob(_ context.Context, cmd app.BackfillCreateCommand, _ time.Time) (app.BackfillJob, error) {
	r.backfillCreate = cmd
	r.backfillJob.ID = cmd.CommandID
	r.backfillJob.CommandID = cmd.CommandID
	r.backfillJob.Stream = cmd.Stream
	r.backfillJob.Status = "queued"
	return r.backfillJob, nil
}

func (r *repo) CancelBackfillJob(_ context.Context, cmd app.BackfillCancelCommand, _ time.Time) (app.BackfillJob, error) {
	r.backfillCancel = cmd
	r.backfillJob.ID = cmd.JobID
	r.backfillJob.Status = "cancelled"
	return r.backfillJob, nil
}
