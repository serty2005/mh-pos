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

type repo struct {
	stockMoves  []app.StockMove
	stockFilter app.StockMoveFilter
}

func (r *repo) ListRawBusinessEvents(context.Context, app.RawBusinessEventFilter) ([]app.RawBusinessEvent, error) {
	return nil, nil
}

func (r *repo) ListStockMoves(_ context.Context, filter app.StockMoveFilter) ([]app.StockMove, error) {
	r.stockFilter = filter
	return r.stockMoves, nil
}
