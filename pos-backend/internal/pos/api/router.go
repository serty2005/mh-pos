package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	httpx "pos-backend/internal/platform/http"
	"pos-backend/internal/pos/app"
	"pos-backend/internal/pos/domain"
)

type Handler struct {
	service *app.Service
}

func NewRouter(service *app.Service) http.Handler {
	h := &Handler{service: service}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestAuditLog)
	r.Use(recoverJSON)
	r.Use(localCORS)
	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/health", h.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/pin-login", h.pinLogin)
		r.Post("/auth/logout", h.logout)
		r.Get("/auth/session", h.getAuthSession)

		r.Post("/system/pair", h.pairEdgeNode)
		r.Get("/system/pairing-status", h.getPairingStatus)
		r.Get("/system/provisioning-status", h.getProvisioningStatus)
		r.Post("/system/provisioning/register-cloud", h.registerCloudProvisioning)
		r.Post("/system/provisioning/pair-via-license", h.pairViaLicense)

		r.Get("/halls", h.listHalls)
		r.Get("/tables", h.listTables)

		r.Get("/catalog/items", h.listCatalogItems)

		r.Get("/menu/items", h.listMenuItems)

		r.Post("/employee-shifts/open", h.openShift)
		r.Post("/employee-shifts/{id}/close", h.closeShift)
		r.Get("/employee-shifts/current", h.currentShift)
		r.Get("/employee-shifts/recent", h.recentShifts)

		r.Post("/orders", h.createOrder)
		r.Get("/orders/current", h.getCurrentOrder)
		r.Get("/orders/active", h.listActiveOrders)
		r.Get("/orders/closed", h.listClosedOrders)
		r.Get("/orders/{id}", h.getOrder)
		r.Post("/orders/{id}/lines", h.addOrderLine)
		r.Patch("/orders/{id}/lines/{line_id}", h.changeOrderLineQuantity)
		r.Patch("/orders/{id}/lines/{line_id}/details", h.updateOrderLineDetails)
		r.Post("/orders/{id}/lines/{line_id}/void", h.voidOrderLine)
		r.Post("/orders/{id}/discounts", h.addOrderDiscount)
		r.Post("/orders/{id}/surcharges", h.addOrderSurcharge)
		r.Get("/pricing/policies", h.listActivePricingPolicies)
		r.Get("/orders/{id}/pricing", h.getOrderPricing)
		r.Post("/orders/{id}/precheck", h.issuePrecheck)
		r.Get("/orders/{id}/prechecks", h.listPrechecksByOrder)
		r.Post("/orders/{id}/close", h.closeOrder)

		r.Get("/prechecks/{id}", h.getPrecheck)
		r.Post("/prechecks/{id}/cancel", h.cancelPrecheck)
		r.Post("/prechecks/{id}/reprint", h.reprintPrecheck)
		r.Post("/prechecks/{id}/payments", h.capturePrecheckPayment)

		r.Get("/checks/{id}", h.getCheck)
		r.Post("/checks/{id}/reprint", h.reprintCheck)
		r.Post("/checks/{id}/cancellations", h.recordCheckCancellation)
		r.Post("/checks/{id}/refunds", h.recordCheckRefund)
		r.Post("/payments/{id}/refund", h.refundPayment)

		r.Post("/cash-shifts/open", h.openCashSession)
		r.Post("/cash-shifts/{id}/close", h.closeCashSession)
		r.Get("/cash-shifts/current", h.currentCashSession)
		r.Post("/cash-drawer-events", h.recordCashDrawerEvent)

		r.Get("/sync/outbox", h.listOutbox)
		r.Get("/sync/local-events", h.listLocalEvents)
		r.Get("/sync/status", h.syncStatus)
		r.Post("/sync/retry-failed", h.retryFailedOutbox)
		r.Post("/sync/master-data/snapshots", h.applyMasterDataSnapshot)
		r.Post("/sync/master-data/{stream}", h.applyMasterDataStream)

		r.Get("/storage/status", h.storageStatus)
		r.Post("/storage/retention/dry-run", h.dryRunStorageRetention)
	})

	return r
}

func requireDevTools(w http.ResponseWriter, r *http.Request) bool {
	httpx.Error(w, fmt.Errorf("%w: Edge master-data mutation APIs are not supported; use Cloud API and Cloud->Edge sync ingest", domain.ErrForbidden), r)
	return false
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" || origin == "http://host.docker.internal:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "false")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Node-Device-ID, X-Client-Device-ID, X-Actor-Employee-ID, X-Session-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Log(r.Context(), slog.LevelError, "panic http обработан безопасным ответом",
					"request_id", middleware.GetReqID(r.Context()),
					"operation", "http.recover",
					"action", r.Method+" "+r.URL.Path,
					"result", "failed",
					"error_code", "INTERNAL_ERROR",
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
					"node_device_id", maskID(requestNodeDeviceID(r)),
					"client_device_id", maskID(requestClientDeviceID(r)),
					"session_id", maskID(r.Header.Get("X-Session-ID")),
					"actor_employee_id", maskID(r.Header.Get("X-Actor-Employee-ID")),
				)
				httpx.Error(w, fmt.Errorf("внутренняя ошибка сервера"), r)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestAuditLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Log(r.Context(), slog.LevelDebug, "http request started",
			"request_id", middleware.GetReqID(r.Context()),
			"operation", "http.request",
			"action", r.Method+" "+r.URL.Path,
			"node_device_id", maskID(requestNodeDeviceID(r)),
			"client_device_id", maskID(requestClientDeviceID(r)),
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
		args := []any{
			"request_id", middleware.GetReqID(r.Context()),
			"operation", "http.request",
			"action", r.Method + " " + r.URL.Path,
			"result", requestResult(rec.status),
			"error_code", requestErrorCode(rec.status, rec.Header().Get("X-Error-Code")),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", duration.Milliseconds(),
			"remote_ip", r.RemoteAddr,
			"node_device_id", maskID(requestNodeDeviceID(r)),
			"client_device_id", maskID(requestClientDeviceID(r)),
			"session_id", maskID(r.Header.Get("X-Session-ID")),
			"actor_employee_id", maskID(r.Header.Get("X-Actor-Employee-ID")),
		}
		if q := r.URL.RawQuery; q != "" {
			args = append(args, "raw_query", q)
		}
		slog.Log(r.Context(), level, "http request completed", args...)
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

func requestResult(status int) string {
	if status >= 200 && status < 300 {
		return "success"
	}
	if status >= 400 && status < 500 {
		return "rejected"
	}
	return "failed"
}

func requestErrorCode(status int, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if status >= 200 && status < 400 {
		return ""
	}
	return fmt.Sprintf("HTTP_%d", status)
}

func (h *Handler) createRestaurant(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.CreateRestaurantCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateRestaurant(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listRestaurants(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	v, err := h.service.ListRestaurants(r.Context())
	writeOK(w, r, v, err)
}

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.RegisterDeviceCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.RegisterDevice(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	v, err := h.service.ListDevices(r.Context())
	writeOK(w, r, v, err)
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.CreateRoleCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateRole(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	v, err := h.service.ListRoles(r.Context())
	writeOK(w, r, v, err)
}

func (h *Handler) createEmployee(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.CreateEmployeeCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateEmployee(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listEmployees(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	v, err := h.service.ListEmployees(r.Context())
	writeOK(w, r, v, err)
}

func (h *Handler) archiveEmployee(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.ArchiveEmployeeCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	if err := h.service.ArchiveEmployee(r.Context(), cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (h *Handler) pinLogin(w http.ResponseWriter, r *http.Request) {
	var cmd app.PinLoginCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.PinLogin(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var cmd app.LogoutCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.Logout(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) getAuthSession(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetSession(r.Context(), r.URL.Query().Get("session_id"), requestNodeDeviceID(r), requestClientDeviceID(r))
	writeOK(w, r, v, err)
}

func (h *Handler) pairEdgeNode(w http.ResponseWriter, r *http.Request) {
	var cmd app.PairEdgeNodeCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	v, err := h.service.PairEdgeNode(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) getPairingStatus(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetPairingStatus(r.Context())
	writeOK(w, r, v, err)
}

func (h *Handler) getProvisioningStatus(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetProvisioningStatus(r.Context())
	if err == nil && v.Status == domain.ProvisioningUnpairedRegistered {
		v, err = h.service.PollCloudAssignment(r.Context())
	}
	writeOK(w, r, v, err)
}

func (h *Handler) registerCloudProvisioning(w http.ResponseWriter, r *http.Request) {
	var cmd app.RegisterCloudProvisioningCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	v, err := h.service.RegisterCloudProvisioning(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) pairViaLicense(w http.ResponseWriter, r *http.Request) {
	var cmd app.PairViaLicenseCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	v, err := h.service.PairViaLicense(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) createHall(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.CreateHallCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateHall(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listHalls(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListHallsAsOperator(r.Context(), r.URL.Query().Get("restaurant_id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) archiveHall(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.ArchiveHallCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	if err := h.service.ArchiveHall(r.Context(), cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (h *Handler) createTable(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.CreateTableCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateTable(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListTablesAsOperator(r.Context(), r.URL.Query().Get("restaurant_id"), r.URL.Query().Get("hall_id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) archiveTable(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.ArchiveTableCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	if err := h.service.ArchiveTable(r.Context(), cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (h *Handler) createCatalogItem(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.CreateCatalogItemCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateCatalogItem(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listCatalogItems(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListCatalogItemsAsOperator(r.Context(), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) createMenuItem(w http.ResponseWriter, r *http.Request) {
	if !requireDevTools(w, r) {
		return
	}
	var cmd app.CreateMenuItemCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setSystemSeedRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateMenuItem(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listMenuItems(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListMenuItemsAsOperator(r.Context(), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) openShift(w http.ResponseWriter, r *http.Request) {
	var cmd app.OpenShiftCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.OpenShift(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) closeShift(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseShiftCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	v, err := h.service.CloseShift(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) currentShift(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetCurrentShift(r.Context(), meta)
	writeOptionalOK(w, r, v, err)
}

func (h *Handler) recentShifts(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cmd := app.ListRecentShiftsCommand{Limit: limit}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.ListRecentShifts(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateOrderCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateOrder(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) getCurrentOrder(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetCurrentOrderByTableAsOperator(r.Context(), r.URL.Query().Get("table_id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetOrderAsOperator(r.Context(), chi.URLParam(r, "id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) listActiveOrders(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListActiveOrdersByHallAsOperator(r.Context(), r.URL.Query().Get("hall_id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) listClosedOrders(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, err := optionalNonNegativeInt(query.Get("limit"), "limit")
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	offset, err := optionalNonNegativeInt(query.Get("offset"), "offset")
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	cmd := app.ListClosedOrdersCommand{
		BusinessDateLocal:     query.Get("business_date_local"),
		FromBusinessDateLocal: query.Get("from_business_date_local"),
		ToBusinessDateLocal:   query.Get("to_business_date_local"),
		ShiftID:               query.Get("shift_id"),
		DeviceID:              query.Get("device_id"),
		CheckID:               query.Get("check_id"),
		Limit:                 limit,
		Offset:                offset,
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.ListClosedOrders(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func optionalNonNegativeInt(raw, name string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%w: %s must be a non-negative integer", domain.ErrInvalid, name)
	}
	return value, nil
}

func (h *Handler) addOrderLine(w http.ResponseWriter, r *http.Request) {
	var cmd app.AddOrderLineCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.AddOrderLine(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) changeOrderLineQuantity(w http.ResponseWriter, r *http.Request) {
	var cmd app.ChangeOrderLineQuantityCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	cmd.LineID = chi.URLParam(r, "line_id")
	v, err := h.service.ChangeOrderLineQuantity(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) updateOrderLineDetails(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateOrderLineDetailsCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	cmd.LineID = chi.URLParam(r, "line_id")
	v, err := h.service.UpdateOrderLineDetails(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) voidOrderLine(w http.ResponseWriter, r *http.Request) {
	var cmd app.VoidOrderLineCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	cmd.LineID = chi.URLParam(r, "line_id")
	v, err := h.service.VoidOrderLine(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) issuePrecheck(w http.ResponseWriter, r *http.Request) {
	var cmd app.IssuePrecheckCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.IssuePrecheck(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) addOrderDiscount(w http.ResponseWriter, r *http.Request) {
	var cmd app.AddDiscountCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.AddDiscount(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) addOrderSurcharge(w http.ResponseWriter, r *http.Request) {
	var cmd app.AddSurchargeCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.AddSurcharge(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listActivePricingPolicies(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListActivePricingPoliciesAsOperator(r.Context(), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) getOrderPricing(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetOrderPricingAsOperator(r.Context(), chi.URLParam(r, "id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) closeOrder(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseOrderCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.CloseOrder(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) getPrecheck(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetPrecheckAsOperator(r.Context(), chi.URLParam(r, "id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) listPrechecksByOrder(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListPrechecksByOrderAsOperator(r.Context(), chi.URLParam(r, "id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) cancelPrecheck(w http.ResponseWriter, r *http.Request) {
	var cmd app.CancelPrecheckCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.PrecheckID = chi.URLParam(r, "id")
	v, err := h.service.CancelPrecheck(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) reprintPrecheck(w http.ResponseWriter, r *http.Request) {
	var cmd app.ReprintPrecheckCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.PrecheckID = chi.URLParam(r, "id")
	v, err := h.service.ReprintPrecheck(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) getCheck(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetCheckAsOperator(r.Context(), chi.URLParam(r, "id"), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) reprintCheck(w http.ResponseWriter, r *http.Request) {
	var cmd app.ReprintCheckCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.CheckID = chi.URLParam(r, "id")
	v, err := h.service.ReprintCheck(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) capturePrecheckPayment(w http.ResponseWriter, r *http.Request) {
	var cmd app.CapturePaymentCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.PrecheckID = chi.URLParam(r, "id")
	v, err := h.service.CapturePayment(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) refundPayment(w http.ResponseWriter, r *http.Request) {
	var cmd app.RefundPaymentCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.PaymentID = chi.URLParam(r, "id")
	v, err := h.service.RefundPayment(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) recordCheckCancellation(w http.ResponseWriter, r *http.Request) {
	var cmd app.RecordCheckCancellationCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.CheckID = chi.URLParam(r, "id")
	v, err := h.service.RecordCancellation(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) recordCheckRefund(w http.ResponseWriter, r *http.Request) {
	var cmd app.RecordCheckRefundCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.CheckID = chi.URLParam(r, "id")
	v, err := h.service.RecordRefund(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) openCashSession(w http.ResponseWriter, r *http.Request) {
	var cmd app.OpenCashSessionCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.OpenCashSession(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) closeCashSession(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseCashSessionCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	v, err := h.service.CloseCashSession(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) currentCashSession(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetCurrentCashSessionAsOperator(r.Context(), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) recordCashDrawerEvent(w http.ResponseWriter, r *http.Request) {
	var cmd app.RecordCashDrawerEventCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.RecordCashDrawerEvent(r.Context(), cmd)
	writeCreated(w, r, v, err)
}

func (h *Handler) listOutbox(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListOutboxAsOperator(r.Context(), meta, limit)
	writeOK(w, r, v, err)
}

func (h *Handler) listLocalEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.ListLocalEventsAsOperator(r.Context(), meta, app.ListLocalEventsQuery{
		Limit:     limit,
		EventType: r.URL.Query().Get("event_type"),
	})
	writeOK(w, r, v, err)
}

func (h *Handler) syncStatus(w http.ResponseWriter, r *http.Request) {
	var meta app.CommandMeta
	setRequestMeta(&meta, r)
	v, err := h.service.GetSyncStatusAsOperator(r.Context(), meta)
	writeOK(w, r, v, err)
}

func (h *Handler) retryFailedOutbox(w http.ResponseWriter, r *http.Request) {
	var cmd app.CommandMeta
	setRequestMeta(&cmd, r)
	n, err := h.service.RetryFailedOutboxAsOperator(r.Context(), cmd)
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int{"retried": n})
}

func (h *Handler) storageStatus(w http.ResponseWriter, r *http.Request) {
	var cmd app.StorageStatusCommand
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.GetStorageLifecycleStatus(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) dryRunStorageRetention(w http.ResponseWriter, r *http.Request) {
	var cmd app.RetentionDryRunCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.DryRunStorageRetention(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) applyMasterDataSnapshot(w http.ResponseWriter, r *http.Request) {
	var cmd app.ApplyMasterDataCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	setCloudSyncRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.ApplyMasterData(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func (h *Handler) applyMasterDataStream(w http.ResponseWriter, r *http.Request) {
	var cmd app.ApplyMasterDataCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err, r)
		return
	}
	cmd.StreamName = domain.MasterDataStream(chi.URLParam(r, "stream"))
	setCloudSyncRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.ApplyMasterData(r.Context(), cmd)
	writeOK(w, r, v, err)
}

func writeCreated(w http.ResponseWriter, r *http.Request, v any, err error) {
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusCreated, v)
}

func writeOK(w http.ResponseWriter, r *http.Request, v any, err error) {
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func writeOptionalOK(w http.ResponseWriter, r *http.Request, v any, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("null\n"))
		return
	}
	writeOK(w, r, v, err)
}

func setEdgeOrigin(meta *app.CommandMeta) {
	meta.Origin = app.OriginEdgeDevice
	app.NormalizeDeviceMeta(meta)
}

func setSystemSeedRequestMeta(meta *app.CommandMeta, r *http.Request) {
	meta.Origin = app.OriginSystemSeed
	if meta.NodeDeviceID == "" {
		meta.NodeDeviceID = requestNodeDeviceID(r)
		meta.DeviceID = meta.NodeDeviceID
	}
	if meta.NodeDeviceID == "" {
		meta.NodeDeviceID = "dev-bootstrap"
		meta.DeviceID = meta.NodeDeviceID
	}
	app.NormalizeDeviceMeta(meta)
}

func setCloudSyncRequestMeta(meta *app.CommandMeta, r *http.Request) {
	meta.Origin = app.OriginCloudSync
	if meta.NodeDeviceID == "" {
		meta.NodeDeviceID = requestNodeDeviceID(r)
		meta.DeviceID = meta.NodeDeviceID
	}
	app.NormalizeDeviceMeta(meta)
}

func setRequestMeta(meta *app.CommandMeta, r *http.Request) {
	setEdgeOrigin(meta)
	if meta.NodeDeviceID == "" {
		meta.NodeDeviceID = requestNodeDeviceID(r)
		meta.DeviceID = meta.NodeDeviceID
	}
	if meta.ClientDeviceID == "" {
		meta.ClientDeviceID = requestClientDeviceID(r)
	}
	if meta.ActorEmployeeID == "" {
		meta.ActorEmployeeID = r.Header.Get("X-Actor-Employee-ID")
	}
	if meta.SessionID == "" {
		meta.SessionID = r.Header.Get("X-Session-ID")
	}
	app.NormalizeDeviceMeta(meta)
}

func requestNodeDeviceID(r *http.Request) string {
	if v := r.Header.Get("X-Node-Device-ID"); v != "" {
		return v
	}
	if v := r.URL.Query().Get("node_device_id"); v != "" {
		return v
	}
	return ""
}

func requestClientDeviceID(r *http.Request) string {
	if v := r.Header.Get("X-Client-Device-ID"); v != "" {
		return v
	}
	return r.URL.Query().Get("client_device_id")
}
