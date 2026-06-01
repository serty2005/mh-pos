package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
	r.Get("/olap/export-status", h.getExportStatus)
	r.Post("/olap/export-retry", h.requestExportRetry)
	r.Get("/olap/raw-business-events", h.listRawBusinessEvents)
	r.Get("/olap/stock-moves", h.listStockMoves)
	r.Get("/olap/stock-move-summary", h.listStockMoveSummary)
	r.Get("/olap/sales-kitchen-summary", h.listSalesKitchenSummary)
	r.Get("/olap/kitchen-timing-summary", h.listKitchenTimingSummary)
	r.Get("/olap/backfill-jobs", h.listBackfillJobs)
	r.Post("/olap/backfill-jobs", h.createBackfillJob)
	r.Get("/olap/backfill-jobs/{id}", h.getBackfillJob)
	r.Post("/olap/backfill-jobs/{id}/cancel", h.cancelBackfillJob)
}

func (h *Handler) getExportStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetExportStatus(r.Context(), r.URL.Query().Get("stream"))
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, status)
}

func (h *Handler) requestExportRetry(w http.ResponseWriter, r *http.Request) {
	var cmd app.ExportRetryCommand
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cmd); err != nil {
		httpx.Error(w, fmt.Errorf("%w: invalid export retry request", contracts.ErrInvalidEnvelope), r)
		return
	}
	result, err := h.service.RequestExportRetry(r.Context(), cmd)
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	slog.InfoContext(r.Context(), "olap export retry command accepted",
		"operation", "olap.export_control",
		"action", "export_retry",
		"result", "accepted",
		"command_id", result.CommandID,
		"stream", result.Stream,
		"mode", result.Mode,
		"already_processed", result.AlreadyProcessed,
		"pending_count", result.PendingCount,
		"failed_count", result.FailedCount,
	)
	httpx.JSON(w, http.StatusAccepted, result)
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

func (h *Handler) listStockMoveSummary(w http.ResponseWriter, r *http.Request) {
	limit, ok := intQuery(w, r, "limit", 50)
	if !ok {
		return
	}
	offset, ok := intQuery(w, r, "offset", 0)
	if !ok {
		return
	}
	items, err := h.service.ListStockMoveSummary(r.Context(), app.StockMoveSummaryFilter{
		RestaurantID:     r.URL.Query().Get("restaurant_id"),
		BusinessDateFrom: r.URL.Query().Get("business_date_from"),
		BusinessDateTo:   r.URL.Query().Get("business_date_to"),
		CatalogItemID:    r.URL.Query().Get("catalog_item_id"),
		WarehouseID:      r.URL.Query().Get("warehouse_id"),
		SourceEventType:  r.URL.Query().Get("source_event_type"),
		GroupBy:          r.URL.Query().Get("group_by"),
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) listSalesKitchenSummary(w http.ResponseWriter, r *http.Request) {
	limit, ok := intQuery(w, r, "limit", 50)
	if !ok {
		return
	}
	offset, ok := intQuery(w, r, "offset", 0)
	if !ok {
		return
	}
	items, err := h.service.ListSalesKitchenSummary(r.Context(), app.SalesKitchenSummaryFilter{
		RestaurantID:     r.URL.Query().Get("restaurant_id"),
		BusinessDateFrom: r.URL.Query().Get("business_date_from"),
		BusinessDateTo:   r.URL.Query().Get("business_date_to"),
		GroupBy:          r.URL.Query().Get("group_by"),
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) listKitchenTimingSummary(w http.ResponseWriter, r *http.Request) {
	limit, ok := intQuery(w, r, "limit", 50)
	if !ok {
		return
	}
	offset, ok := intQuery(w, r, "offset", 0)
	if !ok {
		return
	}
	items, err := h.service.ListKitchenTimingSummary(r.Context(), app.KitchenTimingSummaryFilter{
		RestaurantID:     r.URL.Query().Get("restaurant_id"),
		BusinessDateFrom: r.URL.Query().Get("business_date_from"),
		BusinessDateTo:   r.URL.Query().Get("business_date_to"),
		StationID:        r.URL.Query().Get("station_id"),
		GroupBy:          r.URL.Query().Get("group_by"),
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) listBackfillJobs(w http.ResponseWriter, r *http.Request) {
	limit, ok := intQuery(w, r, "limit", 50)
	if !ok {
		return
	}
	offset, ok := intQuery(w, r, "offset", 0)
	if !ok {
		return
	}
	items, err := h.service.ListBackfillJobs(r.Context(), app.BackfillJobFilter{
		Stream: r.URL.Query().Get("stream"),
		Status: r.URL.Query().Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (h *Handler) getBackfillJob(w http.ResponseWriter, r *http.Request) {
	job, err := h.service.GetBackfillJob(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, job)
}

func (h *Handler) createBackfillJob(w http.ResponseWriter, r *http.Request) {
	var cmd app.BackfillCreateCommand
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cmd); err != nil {
		httpx.Error(w, fmt.Errorf("%w: invalid backfill job request", contracts.ErrInvalidEnvelope), r)
		return
	}
	job, err := h.service.CreateBackfillJob(r.Context(), cmd)
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	slog.InfoContext(r.Context(), "olap backfill job accepted",
		"operation", "olap.backfill",
		"action", "create_job",
		"result", "accepted",
		"command_id", job.CommandID,
		"job_id", job.ID,
		"stream", job.Stream,
		"status", job.Status,
		"already_processed", job.AlreadyProcessed,
	)
	httpx.JSON(w, http.StatusAccepted, job)
}

func (h *Handler) cancelBackfillJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CommandID   string `json:"command_id"`
		Reason      string `json:"reason"`
		RequestedBy string `json:"requested_by,omitempty"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, fmt.Errorf("%w: invalid backfill cancel request", contracts.ErrInvalidEnvelope), r)
		return
	}
	job, err := h.service.CancelBackfillJob(r.Context(), app.BackfillCancelCommand{
		JobID:       chi.URLParam(r, "id"),
		CommandID:   body.CommandID,
		Reason:      body.Reason,
		RequestedBy: body.RequestedBy,
	})
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	slog.InfoContext(r.Context(), "olap backfill job cancellation accepted",
		"operation", "olap.backfill",
		"action", "cancel_job",
		"result", "accepted",
		"command_id", body.CommandID,
		"job_id", job.ID,
		"stream", job.Stream,
		"status", job.Status,
	)
	httpx.JSON(w, http.StatusAccepted, job)
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
