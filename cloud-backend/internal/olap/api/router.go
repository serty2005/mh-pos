package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/olap/app"
	httpx "cloud-backend/internal/platform/httpx"
)

type Handler struct {
	service *app.Service
}

func RegisterRoutes(r chi.Router, service *app.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	r.Get("/olap/raw-business-events", h.listRawBusinessEvents)
	r.Get("/olap/stock-moves", h.listStockMoves)
}

func (h *Handler) listRawBusinessEvents(w http.ResponseWriter, r *http.Request) {
	limit, ok := intQuery(w, r, "limit", 50)
	if !ok {
		return
	}
	offset, ok := intQuery(w, r, "offset", 0)
	if !ok {
		return
	}
	occurredFrom, ok := timeQuery(w, r, "occurred_from")
	if !ok {
		return
	}
	occurredTo, ok := timeQuery(w, r, "occurred_to")
	if !ok {
		return
	}
	items, err := h.service.ListRawBusinessEvents(r.Context(), app.RawBusinessEventFilter{
		RestaurantID: r.URL.Query().Get("restaurant_id"),
		EventType:    r.URL.Query().Get("event_type"),
		OccurredFrom: occurredFrom,
		OccurredTo:   occurredTo,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) listStockMoves(w http.ResponseWriter, r *http.Request) {
	limit, ok := intQuery(w, r, "limit", 50)
	if !ok {
		return
	}
	offset, ok := intQuery(w, r, "offset", 0)
	if !ok {
		return
	}
	items, err := h.service.ListStockMoves(r.Context(), app.StockMoveFilter{
		RestaurantID:     r.URL.Query().Get("restaurant_id"),
		BusinessDateFrom: r.URL.Query().Get("business_date_from"),
		BusinessDateTo:   r.URL.Query().Get("business_date_to"),
		CatalogItemID:    r.URL.Query().Get("catalog_item_id"),
		WarehouseID:      r.URL.Query().Get("warehouse_id"),
		SourceEventType:  r.URL.Query().Get("source_event_type"),
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func intQuery(w http.ResponseWriter, r *http.Request, name string, fallback int) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		httpx.Error(w, fmt.Errorf("%w: %s must be a number", contracts.ErrInvalidEnvelope, name), r)
		return 0, false
	}
	return parsed, true
}

func timeQuery(w http.ResponseWriter, r *http.Request, name string) (*time.Time, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		httpx.Error(w, fmt.Errorf("%w: %s must be RFC3339", contracts.ErrInvalidEnvelope, name), r)
		return nil, false
	}
	parsed = parsed.UTC()
	return &parsed, true
}
