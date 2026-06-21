package api

import (
	"bytes"
	"context"
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
	"mh-pos-platform/licensegate"

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
	gate    licensegate.Gate
	master  *masterapp.Service
}

type financialOperationReportItem struct {
	OperationID          string    `json:"operation_id"`
	EdgeOperationID      string    `json:"edge_operation_id"`
	EventID              string    `json:"event_id"`
	ReceiptID            string    `json:"receipt_id"`
	RestaurantID         string    `json:"restaurant_id"`
	DeviceID             string    `json:"device_id"`
	NodeDeviceID         *string   `json:"node_device_id,omitempty"`
	ClientDeviceID       *string   `json:"client_device_id,omitempty"`
	ActorEmployeeID      *string   `json:"actor_employee_id,omitempty"`
	SessionID            *string   `json:"session_id,omitempty"`
	ShiftID              string    `json:"shift_id"`
	OriginalShiftID      string    `json:"original_shift_id"`
	CheckID              string    `json:"check_id"`
	PrecheckID           string    `json:"precheck_id"`
	OperationType        string    `json:"operation_type"`
	OperationKind        string    `json:"operation_kind"`
	Amount               int64     `json:"amount"`
	Currency             string    `json:"currency"`
	BusinessDateLocal    string    `json:"business_date_local"`
	InventoryDisposition string    `json:"inventory_disposition"`
	Reason               string    `json:"reason"`
	CreatedByEmployeeID  string    `json:"created_by_employee_id,omitempty"`
	ApprovedByEmployeeID *string   `json:"approved_by_employee_id,omitempty"`
	OperationCreatedAt   time.Time `json:"operation_created_at"`
	OccurredAt           time.Time `json:"occurred_at"`
	CloudReceivedAt      time.Time `json:"cloud_received_at"`
	RawPayloadSHA256Hex  string    `json:"raw_payload_sha256_hex"`
}

func NewRouter(service *app.Service, masterServices ...*masterapp.Service) http.Handler {
	return NewRouterWithProvisioning(service, nil, masterServices...)
}

func NewRouterWithProvisioning(service *app.Service, provisioningService *provisioningapp.Service, masterServices ...*masterapp.Service) http.Handler {
	return NewRouterWithProvisioningAndOLAP(service, provisioningService, nil, masterServices...)
}

func NewRouterWithProvisioningAndOLAP(service *app.Service, provisioningService *provisioningapp.Service, olapService *olapapp.Service, masterServices ...*masterapp.Service) http.Handler {
	return NewRouterWithProvisioningOLAPAndLicense(service, provisioningService, olapService, nil, masterServices...)
}

func NewRouterWithProvisioningOLAPAndLicense(service *app.Service, provisioningService *provisioningapp.Service, olapService *olapapp.Service, gate licensegate.Gate, masterServices ...*masterapp.Service) http.Handler {
	h := &Handler{service: service, gate: gate}
	if len(masterServices) > 0 {
		h.master = masterServices[0]
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestAuditLog)
	r.Use(middleware.Recoverer)
	r.Use(localCORS)
	if gate != nil {
		r.Use(licensegate.Middleware(gate, cloudModuleForRequest))
	}
	r.Options("/*", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/health", h.health)
	if gate != nil {
		r.Get("/api/v1/license/entitlements", licensegate.StatusHandler(gate))
	}
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/sync/edge-events", h.listEdgeEvents)
		r.Get("/sync/readiness/stop-list", h.stopListReadiness)
		r.Get("/reporting/financial-operations", h.listFinancialOperations)
		r.Get("/inventory/stock-ledger", h.listInventoryLedger)
		r.Get("/inventory/stock-balances", h.listInventoryStockBalances)
		r.Get("/inventory/recalculation-jobs", h.listInventoryRecalculationJobs)
		r.Get("/inventory/recalculation-jobs/{id}", h.getInventoryRecalculationJob)
		r.Post("/sync/edge-events", h.receiveEdgeEvent)
		r.Post("/sync/edge-events/batch", h.receiveEdgeEventBatch)
		r.Post("/sync/exchange", h.exchange)
		provisioningapi.RegisterMasterDataPackageRoutes(r, service)
		if len(masterServices) > 0 {
			masterapi.RegisterRoutes(r, masterServices[0])
		}
		olapapi.RegisterRoutes(r, olapService)
		provisioningapi.RegisterRoutes(r, provisioningService)
	})
	return r
}

func cloudModuleForRequest(r *http.Request) string {
	path := r.URL.Path
	switch {
	case strings.Contains(path, "/provisioning/master-data/floor"):
		return licensegate.TableMode
	case strings.Contains(path, "/provisioning/master-data/recipes"):
		return licensegate.KitchenSpace
	case strings.Contains(path, "/provisioning/master-data/inventory_reference"):
		return licensegate.WarehouseMode
	case strings.Contains(path, "/master-data/floor/") || strings.Contains(path, "/master-data/halls") || strings.Contains(path, "/master-data/tables"):
		return licensegate.TableMode
	case strings.Contains(path, "/master-data/recipes/") || strings.Contains(path, "/master-data/recipe-suggestions"):
		return licensegate.KitchenSpace
	case strings.Contains(path, "/master-data/inventory/") || strings.HasPrefix(path, "/api/v1/inventory/"):
		return licensegate.WarehouseMode
	default:
		return ""
	}
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

func (h *Handler) listFinancialOperations(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.service.ListFinancialOperations(r.Context(), app.FinancialOperationProjectionFilter{
		RestaurantID:     r.URL.Query().Get("restaurant_id"),
		BusinessDateFrom: r.URL.Query().Get("business_date_from"),
		BusinessDateTo:   r.URL.Query().Get("business_date_to"),
		OperationType:    r.URL.Query().Get("operation_type"),
		ShiftID:          r.URL.Query().Get("shift_id"),
		OriginalShiftID:  r.URL.Query().Get("original_shift_id"),
		CheckID:          r.URL.Query().Get("check_id"),
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, contracts.ErrInvalidEnvelope) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	out := make([]financialOperationReportItem, 0, len(items))
	for _, item := range items {
		out = append(out, financialOperationReportItem{
			OperationID:          item.OperationID,
			EdgeOperationID:      item.EdgeOperationID,
			EventID:              item.EventID,
			ReceiptID:            item.ReceiptID,
			RestaurantID:         item.RestaurantID,
			DeviceID:             item.DeviceID,
			NodeDeviceID:         item.NodeDeviceID,
			ClientDeviceID:       item.ClientDeviceID,
			ActorEmployeeID:      item.ActorEmployeeID,
			SessionID:            item.SessionID,
			ShiftID:              item.ShiftID,
			OriginalShiftID:      item.OriginalShiftID,
			CheckID:              item.CheckID,
			PrecheckID:           item.PrecheckID,
			OperationType:        item.OperationType,
			OperationKind:        item.OperationKind,
			Amount:               item.Amount,
			Currency:             item.Currency,
			BusinessDateLocal:    item.BusinessDateLocal,
			InventoryDisposition: item.InventoryDisposition,
			Reason:               item.Reason,
			CreatedByEmployeeID:  item.CreatedByEmployeeID,
			ApprovedByEmployeeID: item.ApprovedByEmployeeID,
			OperationCreatedAt:   item.OperationCreatedAt,
			OccurredAt:           item.OccurredAt,
			CloudReceivedAt:      item.CloudReceivedAt,
			RawPayloadSHA256Hex:  item.RawPayloadSHA256Hex,
		})
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
		RestaurantID:   r.URL.Query().Get("restaurant_id"),
		WarehouseID:    r.URL.Query().Get("warehouse_id"),
		CatalogItemID:  r.URL.Query().Get("catalog_item_id"),
		BusinessDateTo: r.URL.Query().Get("business_date_to"),
		CostingStatus:  r.URL.Query().Get("costing_status"),
		Limit:          limit,
		Offset:         offset,
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

func (h *Handler) listInventoryRecalculationJobs(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.service.ListInventoryRecalculationJobs(r.Context(), app.InventoryRecalculationJobFilter{
		RestaurantID: r.URL.Query().Get("restaurant_id"),
		Status:       r.URL.Query().Get("status"),
		TriggerType:  r.URL.Query().Get("trigger_type"),
		Limit:        limit,
		Offset:       offset,
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

func (h *Handler) getInventoryRecalculationJob(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetInventoryRecalculationJob(r.Context(), chi.URLParam(r, "id"))
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
	writeJSON(w, http.StatusOK, item)
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
	if h.master != nil {
		if err := h.master.RetryDeliveryForNode(r.Context(), req.RestaurantID, req.NodeDeviceID); err != nil {
			slog.ErrorContext(r.Context(), "master-data delivery retry failed", "operation", "sync.exchange.delivery_retry", "restaurant_id", req.RestaurantID, "node_device_id", req.NodeDeviceID, "error", err)
		}
	}
	platformExchangeLog(r, "exchange", "attempt", "", req.NodeDeviceID)
	resp, err := h.service.Exchange(r.Context(), req)
	if err != nil {
		platformExchangeLog(r, "exchange", "rejected", errorCodeForExchange(err), req.NodeDeviceID)
		httpx.Error(w, err, r)
		return
	}
	resp.CloudPackages = h.licensedCloudPackages(r.Context(), resp.CloudPackages)
	platformExchangeLog(r, "exchange", "success", "", req.NodeDeviceID)
	writeJSON(w, http.StatusAccepted, resp)
}

// licensedCloudPackages не отдает Edge данные модуля, отключенного внешним authority.
func (h *Handler) licensedCloudPackages(ctx context.Context, packages []contracts.SyncExchangeCloudPackage) []contracts.SyncExchangeCloudPackage {
	if h.gate == nil {
		return packages
	}
	result := make([]contracts.SyncExchangeCloudPackage, 0, len(packages))
	for _, pkg := range packages {
		moduleID := ""
		switch pkg.StreamName {
		case contracts.MasterDataStreamFloor:
			moduleID = licensegate.TableMode
		case contracts.MasterDataStreamRecipes:
			moduleID = licensegate.KitchenSpace
		case contracts.MasterDataStreamInventory:
			moduleID = licensegate.WarehouseMode
		}
		if moduleID == "" || h.gate.Require(ctx, moduleID) == nil {
			result = append(result, pkg)
		}
	}
	return result
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	httpx.JSON(w, status, v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	_ = status
	httpx.Error(w, err)
}
