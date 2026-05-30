package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
	masterapi "cloud-backend/internal/masterdata/api"
	masterapp "cloud-backend/internal/masterdata/app"
	olapapi "cloud-backend/internal/olap/api"
	olapapp "cloud-backend/internal/olap/app"
	httpx "cloud-backend/internal/platform/httpx"
	provisioningapi "cloud-backend/internal/provisioning/api"
	provisioningapp "cloud-backend/internal/provisioning/app"
)

type Handler struct {
	service *app.Service
}

func NewRouter(service *app.Service, masterServices ...*masterapp.Service) http.Handler {
	return NewRouterWithProvisioning(service, nil, masterServices...)
}

func NewRouterWithProvisioning(service *app.Service, provisioningService *provisioningapp.Service, masterServices ...*masterapp.Service) http.Handler {
	return NewRouterWithProvisioningAndOLAP(service, provisioningService, nil, masterServices...)
}

func NewRouterWithProvisioningAndOLAP(service *app.Service, provisioningService *provisioningapp.Service, olapService *olapapp.Service, masterServices ...*masterapp.Service) http.Handler {
	h := &Handler{service: service}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestAuditLog)
	r.Use(middleware.Recoverer)
	r.Use(localCORS)
	r.Options("/*", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/health", h.health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/sync/edge-events", h.listEdgeEvents)
		r.Get("/sync/readiness/stop-list", h.stopListReadiness)
		r.Get("/inventory/stock-ledger", h.listInventoryLedger)
		r.Get("/inventory/stock-balances", h.listInventoryStockBalances)
		r.Post("/sync/edge-events", h.receiveEdgeEvent)
		r.Post("/sync/edge-events/batch", h.receiveEdgeEventBatch)
		r.Post("/sync/exchange", h.exchange)
		r.Put("/provisioning/master-data/{stream}", h.upsertMasterDataPackage)
		r.Get("/provisioning/master-data/{stream}", h.getMasterDataPackage)
		if len(masterServices) > 0 {
			masterapi.RegisterRoutes(r, masterServices[0])
		}
		olapapi.RegisterRoutes(r, olapService)
		provisioningapi.RegisterRoutes(r, provisioningService)
	})
	return r
}

func requestAuditLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Log(r.Context(), slog.LevelDebug, "http request started",
			"request_id", middleware.GetReqID(r.Context()),
			"operation", "http.request",
			"action", r.Method+" "+r.URL.Path,
		)
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(start)
		level := slog.LevelInfo
		if rec.status >= 500 {
			level = slog.LevelError
		} else if rec.status >= 400 {
			level = slog.LevelWarn
		}
		slog.Log(r.Context(), level, "http request completed",
			"request_id", middleware.GetReqID(r.Context()),
			"operation", "http.request",
			"action", r.Method+" "+r.URL.Path,
			"result", requestResult(rec.status),
			"error_code", requestErrorCode(rec.status),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", duration.Milliseconds(),
			"remote_ip", r.RemoteAddr,
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func requestResult(status int) string {
	if status >= 200 && status < 300 {
		return "success"
	}
	if status >= 400 && status < 500 {
		return "rejected"
	}
	return "failed"
}

func requestErrorCode(status int) string {
	if status >= 200 && status < 400 {
		return ""
	}
	return fmt.Sprintf("HTTP_%d", status)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5174" || origin == "http://127.0.0.1:5174" || origin == "http://host.docker.internal:5174" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "false")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) listEdgeEvents(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%w: limit must be a number", contracts.ErrInvalidEnvelope))
			return
		}
		limit = parsed
	}
	items, err := h.service.ListEdgeEvents(r.Context(), app.EdgeEventListFilter{
		RestaurantID: r.URL.Query().Get("restaurant_id"),
		DeviceID:     r.URL.Query().Get("device_id"),
		EventType:    r.URL.Query().Get("event_type"),
		Limit:        limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) stopListReadiness(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetStopListReadiness(r.Context(), app.StopListReadinessFilter{
		RestaurantID: r.URL.Query().Get("restaurant_id"),
		NodeDeviceID: r.URL.Query().Get("node_device_id"),
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, contracts.ErrInvalidEnvelope) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) listInventoryLedger(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%w: limit must be a number", contracts.ErrInvalidEnvelope))
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%w: offset must be a number", contracts.ErrInvalidEnvelope))
			return
		}
		offset = parsed
	}
	items, err := h.service.ListInventoryLedger(r.Context(), app.InventoryLedgerFilter{
		RestaurantID:    r.URL.Query().Get("restaurant_id"),
		SourceEventType: r.URL.Query().Get("source_event_type"),
		SourceEventID:   r.URL.Query().Get("source_event_id"),
		OrderLineID:     r.URL.Query().Get("order_line_id"),
		CatalogItemID:   r.URL.Query().Get("catalog_item_id"),
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, contracts.ErrInvalidEnvelope) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) listInventoryStockBalances(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%w: limit must be a number", contracts.ErrInvalidEnvelope))
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%w: offset must be a number", contracts.ErrInvalidEnvelope))
			return
		}
		offset = parsed
	}
	items, err := h.service.ListInventoryStockBalances(r.Context(), app.InventoryStockBalanceFilter{
		RestaurantID:    r.URL.Query().Get("restaurant_id"),
		WarehouseID:     r.URL.Query().Get("warehouse_id"),
		CatalogItemID:   r.URL.Query().Get("catalog_item_id"),
		BusinessDateTo:  r.URL.Query().Get("business_date_to"),
		CostingStatus:   r.URL.Query().Get("costing_status"),
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, contracts.ErrInvalidEnvelope) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) receiveEdgeEvent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ack, err := h.service.Receive(r.Context(), body)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, contracts.ErrInvalidEnvelope):
			status = http.StatusBadRequest
		case errors.Is(err, contracts.ErrPayloadConflict):
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusAccepted, ack)
}

func (h *Handler) receiveEdgeEventBatch(w http.ResponseWriter, r *http.Request) {
	var req contracts.BatchReceiveRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%w: %v", contracts.ErrInvalidEnvelope, err))
		return
	}
	if len(req.Items) == 0 || len(req.Items) > 100 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%w: items length must be between 1 and 100", contracts.ErrInvalidEnvelope))
		return
	}
	raws := make([][]byte, 0, len(req.Items))
	for _, item := range req.Items {
		raw := bytes.TrimSpace(item)
		if len(raw) == 0 || string(raw) == "null" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("%w: batch item payload is required", contracts.ErrInvalidEnvelope))
			return
		}
		raws = append(raws, raw)
	}
	ack := h.service.ReceiveBatch(r.Context(), raws)
	writeJSON(w, http.StatusAccepted, ack)
}

func (h *Handler) exchange(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		httpx.Error(w, fmt.Errorf("%w: exchange body read failed", contracts.ErrInvalidEnvelope), r)
		return
	}
	req, err := contracts.DecodeSyncExchangeRequest(bytes.TrimSpace(body))
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	token := bearerToken(r.Header.Get("Authorization"))
	if err := h.service.AuthenticateNodeToken(r.Context(), req.NodeDeviceID, req.RestaurantID, token); err != nil {
		platformExchangeLog(r, "auth", "rejected", errorCodeForExchange(err), req.NodeDeviceID)
		httpx.Error(w, err, r)
		return
	}
	platformExchangeLog(r, "exchange", "attempt", "", req.NodeDeviceID)
	resp, err := h.service.Exchange(r.Context(), req)
	if err != nil {
		platformExchangeLog(r, "exchange", "rejected", errorCodeForExchange(err), req.NodeDeviceID)
		httpx.Error(w, err, r)
		return
	}
	platformExchangeLog(r, "exchange", "success", "", req.NodeDeviceID)
	writeJSON(w, http.StatusAccepted, resp)
}

func bearerToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(raw, prefix))
}

func platformExchangeLog(r *http.Request, action, result, errorCode, nodeDeviceID string) {
	slog.Log(r.Context(), slog.LevelInfo, "cloud sync exchange",
		"request_id", middleware.GetReqID(r.Context()),
		"operation", "sync.exchange",
		"action", action,
		"result", result,
		"error_code", errorCode,
		"node_device_id", maskID(nodeDeviceID),
	)
}

func errorCodeForExchange(err error) string {
	switch {
	case errors.Is(err, contracts.ErrSyncUnauthorized):
		return "SYNC_UNAUTHORIZED"
	case errors.Is(err, contracts.ErrSyncForbidden):
		return "SYNC_FORBIDDEN"
	case errors.Is(err, contracts.ErrSyncRevisionAhead):
		return "SYNC_REVISION_AHEAD"
	case errors.Is(err, contracts.ErrSyncCheckpointConflict):
		return "SYNC_CHECKPOINT_CONFLICT"
	case errors.Is(err, contracts.ErrInvalidEnvelope):
		return "VALIDATION_FAILED"
	default:
		return "INTERNAL_ERROR"
	}
}

func maskID(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if len(v) <= 8 {
		return v
	}
	return v[:8] + "..."
}

func (h *Handler) upsertMasterDataPackage(w http.ResponseWriter, r *http.Request) {
	streamName := chi.URLParam(r, "stream")
	var req struct {
		NodeDeviceID       string          `json:"node_device_id"`
		RestaurantID       string          `json:"restaurant_id"`
		SyncMode           string          `json:"sync_mode"`
		FullSnapshotReason string          `json:"full_snapshot_reason"`
		CloudVersion       int64           `json:"cloud_version"`
		CheckpointToken    string          `json:"checkpoint_token"`
		CloudUpdatedAt     *time.Time      `json:"cloud_updated_at"`
		PayloadJSON        json.RawMessage `json:"payload_json"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%w: %v", contracts.ErrInvalidEnvelope, err))
		return
	}
	v, err := h.service.UpsertMasterDataPackage(r.Context(), contracts.MasterDataPackage{
		StreamName:         streamName,
		NodeDeviceID:       req.NodeDeviceID,
		RestaurantID:       req.RestaurantID,
		SyncMode:           req.SyncMode,
		FullSnapshotReason: req.FullSnapshotReason,
		CloudVersion:       req.CloudVersion,
		CheckpointToken:    req.CheckpointToken,
		CloudUpdatedAt:     req.CloudUpdatedAt,
		PayloadJSON:        req.PayloadJSON,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, contracts.ErrInvalidEnvelope) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) getMasterDataPackage(w http.ResponseWriter, r *http.Request) {
	streamName := chi.URLParam(r, "stream")
	nodeDeviceID := r.URL.Query().Get("node_device_id")
	v, err := h.service.GetMasterDataPackage(r.Context(), streamName, nodeDeviceID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, contracts.ErrInvalidEnvelope):
			status = http.StatusBadRequest
		case errors.Is(err, contracts.ErrNotFound):
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	httpx.JSON(w, status, v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	_ = status
	httpx.Error(w, err)
}
