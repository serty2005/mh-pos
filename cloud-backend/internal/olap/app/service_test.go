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

type rawRepo struct {
	filter        app.RawBusinessEventFilter
	stockFilter   app.StockMoveFilter
	summaryFilter app.StockMoveSummaryFilter
	statusStream  string
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

func (r *rawRepo) GetExportStatus(_ context.Context, stream string, _ time.Time) (app.ExportStatus, error) {
	r.statusStream = stream
	return app.ExportStatus{Stream: stream}, nil
}
