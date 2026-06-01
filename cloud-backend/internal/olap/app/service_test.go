package app_test

import (
	"context"
	"testing"
	"time"

	"cloud-backend/internal/olap/app"
)

func TestListRawBusinessEventsAppliesBoundedLimit(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListRawBusinessEvents(context.Background(), app.RawBusinessEventFilter{Limit: 1000}); err != nil {
		t.Fatal(err)
	}
	if repo.filter.Limit != 50 {
		t.Fatalf("expected oversized limit to fall back to 50, got %d", repo.filter.Limit)
	}
}

func TestListRawBusinessEventsRejectsInvalidRange(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)
	from := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)

	if _, err := service.ListRawBusinessEvents(context.Background(), app.RawBusinessEventFilter{OccurredFrom: &from, OccurredTo: &to}); err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestListStockMovesAppliesFiltersAndBoundedLimit(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListStockMoves(context.Background(), app.StockMoveFilter{
		RestaurantID:     " rest-1 ",
		BusinessDateFrom: "2026-05-01",
		BusinessDateTo:   "2026-05-29",
		CatalogItemID:    " item-1 ",
		WarehouseID:      " wh-1 ",
		SourceEventType:  " StockReceiptCaptured ",
		Limit:            1000,
	}); err != nil {
		t.Fatal(err)
	}
	if repo.stockFilter.Limit != 50 {
		t.Fatalf("expected oversized limit to fall back to 50, got %d", repo.stockFilter.Limit)
	}
	if repo.stockFilter.RestaurantID != "rest-1" || repo.stockFilter.CatalogItemID != "item-1" || repo.stockFilter.WarehouseID != "wh-1" {
		t.Fatalf("expected trimmed filters, got %+v", repo.stockFilter)
	}
}

func TestListStockMovesRejectsInvalidBusinessDateRange(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListStockMoves(context.Background(), app.StockMoveFilter{
		BusinessDateFrom: "2026-05-29",
		BusinessDateTo:   "2026-05-01",
	}); err == nil {
		t.Fatal("expected invalid range error")
	}
	if _, err := service.ListStockMoves(context.Background(), app.StockMoveFilter{
		BusinessDateFrom: "2026-05-XX",
	}); err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestListStockMoveSummaryAppliesGroupingFiltersAndBoundedLimit(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListStockMoveSummary(context.Background(), app.StockMoveSummaryFilter{
		RestaurantID:     " rest-1 ",
		BusinessDateFrom: "2026-05-01",
		BusinessDateTo:   "2026-05-29",
		CatalogItemID:    " item-1 ",
		WarehouseID:      " wh-1 ",
		SourceEventType:  " StockReceiptCaptured ",
		GroupBy:          " catalog_item ",
		Limit:            1000,
	}); err != nil {
		t.Fatal(err)
	}
	if repo.summaryFilter.Limit != 50 {
		t.Fatalf("expected oversized limit to fall back to 50, got %d", repo.summaryFilter.Limit)
	}
	if repo.summaryFilter.GroupBy != "catalog_item" || repo.summaryFilter.RestaurantID != "rest-1" || repo.summaryFilter.CatalogItemID != "item-1" {
		t.Fatalf("expected trimmed summary filters, got %+v", repo.summaryFilter)
	}
}

func TestListStockMoveSummaryRejectsInvalidGroupBy(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListStockMoveSummary(context.Background(), app.StockMoveSummaryFilter{GroupBy: "cogs"}); err == nil {
		t.Fatal("expected invalid group_by error")
	}
}

func TestListSalesKitchenSummaryAppliesFiltersGroupingAndBoundedLimit(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListSalesKitchenSummary(context.Background(), app.SalesKitchenSummaryFilter{
		RestaurantID:     " rest-1 ",
		BusinessDateFrom: "2026-05-01",
		BusinessDateTo:   "2026-05-29",
		GroupBy:          " source_event_type ",
		Limit:            1000,
	}); err != nil {
		t.Fatal(err)
	}
	if repo.salesKitchenFilter.Limit != 50 {
		t.Fatalf("expected oversized limit to fall back to 50, got %d", repo.salesKitchenFilter.Limit)
	}
	if repo.salesKitchenFilter.GroupBy != "source_event_type" || repo.salesKitchenFilter.RestaurantID != "rest-1" {
		t.Fatalf("expected trimmed sales/kitchen filters, got %+v", repo.salesKitchenFilter)
	}
}

func TestListSalesKitchenSummaryRejectsInvalidFilters(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	tests := []app.SalesKitchenSummaryFilter{
		{GroupBy: "margin"},
		{BusinessDateFrom: "2026-05-XX"},
		{BusinessDateFrom: "2026-05-29", BusinessDateTo: "2026-05-01"},
		{Offset: -1},
	}
	for _, tt := range tests {
		if _, err := service.ListSalesKitchenSummary(context.Background(), tt); err == nil {
			t.Fatalf("expected invalid filter error for %+v", tt)
		}
	}
}

func TestListKitchenTimingSummaryAppliesFiltersGroupingAndBoundedLimit(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	if _, err := service.ListKitchenTimingSummary(context.Background(), app.KitchenTimingSummaryFilter{
		RestaurantID:     " rest-1 ",
		BusinessDateFrom: "2026-05-01",
		BusinessDateTo:   "2026-05-29",
		StationID:        " hot ",
		GroupBy:          " station ",
		Limit:            1000,
	}); err != nil {
		t.Fatal(err)
	}
	if repo.kitchenTimingFilter.Limit != 50 {
		t.Fatalf("expected oversized limit to fall back to 50, got %d", repo.kitchenTimingFilter.Limit)
	}
	if repo.kitchenTimingFilter.GroupBy != "station" || repo.kitchenTimingFilter.StationID != "hot" {
		t.Fatalf("expected trimmed kitchen timing filters, got %+v", repo.kitchenTimingFilter)
	}
}

func TestListKitchenTimingSummaryRejectsInvalidFilters(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewService(repo)

	tests := []app.KitchenTimingSummaryFilter{
		{GroupBy: "cogs"},
		{BusinessDateFrom: "2026-05-XX"},
		{BusinessDateFrom: "2026-05-29", BusinessDateTo: "2026-05-01"},
		{Offset: -1},
	}
	for _, tt := range tests {
		if _, err := service.ListKitchenTimingSummary(context.Background(), tt); err == nil {
			t.Fatalf("expected invalid filter error for %+v", tt)
		}
	}
}

func TestGetExportStatusRequiresKnownStream(t *testing.T) {
	repo := &rawRepo{}
	service := app.NewServiceWithExportStatus(repo, repo)

	if _, err := service.GetExportStatus(context.Background(), "stock_moves"); err != nil {
		t.Fatal(err)
	}
	if repo.statusStream != "stock_moves" {
		t.Fatalf("expected stock_moves stream, got %q", repo.statusStream)
	}
	if _, err := service.GetExportStatus(context.Background(), "payloads"); err == nil {
		t.Fatal("expected invalid stream error")
	}
}

func TestRequestExportRetryValidatesCommandAndIsIdempotent(t *testing.T) {
	repo := &rawRepo{retryResult: app.ExportRetryResult{
		CommandID:        "018f0000-0000-7000-8000-000000000111",
		Stream:           "stock_moves",
		Mode:             "retry_failed",
		Accepted:         true,
		CheckpointBefore: "ledger-10",
		PendingCount:     2,
		FailedCount:      0,
	}}
	service := app.NewServiceWithControls(repo, repo, repo)

	result, err := service.RequestExportRetry(context.Background(), app.ExportRetryCommand{
		CommandID: "018f0000-0000-7000-8000-000000000111",
		Stream:    " stock_moves ",
		Mode:      " retry_failed ",
		Reason:    " operator requested retry ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stream != "stock_moves" || result.CheckpointBefore != "ledger-10" || repo.retryCommand.Reason != "operator requested retry" {
		t.Fatalf("unexpected retry result=%+v command=%+v", result, repo.retryCommand)
	}
	replay, err := service.RequestExportRetry(context.Background(), app.ExportRetryCommand{
		CommandID: "018f0000-0000-7000-8000-000000000111",
		Stream:    "stock_moves",
		Mode:      "retry_failed",
		Reason:    "operator requested retry",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replay.AlreadyProcessed || replay.CommandID != result.CommandID {
		t.Fatalf("expected idempotent replay result, got %+v", replay)
	}
	if _, err := service.RequestExportRetry(context.Background(), app.ExportRetryCommand{
		CommandID: "not-a-uuid",
		Stream:    "stock_moves",
		Mode:      "retry_failed",
		Reason:    "operator requested retry",
	}); err == nil {
		t.Fatal("expected invalid command_id error")
	}
	if _, err := service.RequestExportRetry(context.Background(), app.ExportRetryCommand{
		CommandID: "018f0000-0000-7000-8000-000000000112",
		Stream:    "payloads",
		Mode:      "retry_failed",
		Reason:    "operator requested retry",
	}); err == nil {
		t.Fatal("expected invalid stream error")
	}
	if _, err := service.RequestExportRetry(context.Background(), app.ExportRetryCommand{
		CommandID: "018f0000-0000-7000-8000-000000000113",
		Stream:    "stock_moves",
		Mode:      "full_backfill",
		Reason:    "operator requested retry",
	}); err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestBackfillJobCommandsValidateAndUseUUIDv7(t *testing.T) {
	repo := &rawRepo{backfillJob: app.BackfillJob{
		ID:        "018f0000-0000-7000-8000-000000000311",
		CommandID: "018f0000-0000-7000-8000-000000000311",
		Stream:    "raw_business_events",
		Status:    "queued",
	}}
	service := app.NewServiceWithOperatorControls(repo, repo, repo, repo)

	_, err := service.CreateBackfillJob(context.Background(), app.BackfillCreateCommand{
		CommandID: "018f0000-0000-7000-8000-000000000311",
		Stream:    " raw_business_events ",
		Reason:    " rebuild archive ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.backfillCreate.Stream != "raw_business_events" || repo.backfillCreate.Reason != "rebuild archive" {
		t.Fatalf("expected normalized create command, got %+v", repo.backfillCreate)
	}
	if _, err := service.CreateBackfillJob(context.Background(), app.BackfillCreateCommand{
		CommandID: "not-a-uuid",
		Stream:    "raw_business_events",
		Reason:    "rebuild archive",
	}); err == nil {
		t.Fatal("expected invalid command_id error")
	}
	if _, err := service.CancelBackfillJob(context.Background(), app.BackfillCancelCommand{
		JobID:     "018f0000-0000-7000-8000-000000000311",
		CommandID: "018f0000-0000-7000-8000-000000000312",
		Reason:    "operator cancel",
	}); err != nil {
		t.Fatal(err)
	}
	if repo.backfillCancel.JobID == "" {
		t.Fatal("expected cancel command to reach repository")
	}
}

type rawRepo struct {
	filter              app.RawBusinessEventFilter
	stockFilter         app.StockMoveFilter
	summaryFilter       app.StockMoveSummaryFilter
	salesKitchenFilter  app.SalesKitchenSummaryFilter
	kitchenTimingFilter app.KitchenTimingSummaryFilter
	statusStream        string
	retryCommand        app.ExportRetryCommand
	retryResult         app.ExportRetryResult
	retryCalls          int
	backfillFilter      app.BackfillJobFilter
	backfillCreate      app.BackfillCreateCommand
	backfillCancel      app.BackfillCancelCommand
	backfillJob         app.BackfillJob
}

func (r *rawRepo) ListRawBusinessEvents(_ context.Context, filter app.RawBusinessEventFilter) ([]app.RawBusinessEvent, error) {
	r.filter = filter
	return nil, nil
}

func (r *rawRepo) ListStockMoves(_ context.Context, filter app.StockMoveFilter) ([]app.StockMove, error) {
	r.stockFilter = filter
	return nil, nil
}

func (r *rawRepo) ListStockMoveSummary(_ context.Context, filter app.StockMoveSummaryFilter) ([]app.StockMoveSummary, error) {
	r.summaryFilter = filter
	return nil, nil
}

func (r *rawRepo) ListSalesKitchenSummary(_ context.Context, filter app.SalesKitchenSummaryFilter) ([]app.SalesKitchenSummary, error) {
	r.salesKitchenFilter = filter
	return nil, nil
}

func (r *rawRepo) ListKitchenTimingSummary(_ context.Context, filter app.KitchenTimingSummaryFilter) ([]app.KitchenTimingSummary, error) {
	r.kitchenTimingFilter = filter
	return nil, nil
}

func (r *rawRepo) GetExportStatus(_ context.Context, stream string, _ time.Time) (app.ExportStatus, error) {
	r.statusStream = stream
	return app.ExportStatus{Stream: stream}, nil
}

func (r *rawRepo) RequestExportRetry(_ context.Context, cmd app.ExportRetryCommand, now time.Time) (app.ExportRetryResult, error) {
	r.retryCommand = cmd
	r.retryCalls++
	if r.retryResult.CommandID == "" {
		r.retryResult = app.ExportRetryResult{
			CommandID:        cmd.CommandID,
			Stream:           cmd.Stream,
			Mode:             cmd.Mode,
			Accepted:         true,
			RetryRequestedAt: now,
		}
	}
	if r.retryCalls > 1 && r.retryResult.CommandID == cmd.CommandID && r.retryResult.Stream == cmd.Stream && r.retryResult.Mode == cmd.Mode {
		r.retryResult.AlreadyProcessed = true
	}
	return r.retryResult, nil
}

func (r *rawRepo) ListBackfillJobs(_ context.Context, filter app.BackfillJobFilter) ([]app.BackfillJob, error) {
	r.backfillFilter = filter
	return []app.BackfillJob{r.backfillJob}, nil
}

func (r *rawRepo) GetBackfillJob(_ context.Context, id string) (app.BackfillJob, error) {
	r.backfillJob.ID = id
	return r.backfillJob, nil
}

func (r *rawRepo) CreateBackfillJob(_ context.Context, cmd app.BackfillCreateCommand, _ time.Time) (app.BackfillJob, error) {
	r.backfillCreate = cmd
	r.backfillJob.ID = cmd.CommandID
	r.backfillJob.CommandID = cmd.CommandID
	r.backfillJob.Stream = cmd.Stream
	r.backfillJob.Status = "queued"
	return r.backfillJob, nil
}

func (r *rawRepo) CancelBackfillJob(_ context.Context, cmd app.BackfillCancelCommand, _ time.Time) (app.BackfillJob, error) {
	r.backfillCancel = cmd
	r.backfillJob.ID = cmd.JobID
	r.backfillJob.Status = "cancelled"
	return r.backfillJob, nil
}
