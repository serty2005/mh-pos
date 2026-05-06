package api

import (
	"fmt"
	"net/http"
	"strconv"

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
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(localCORS)
	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/health", h.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/restaurants", h.createRestaurant)
		r.Get("/restaurants", h.listRestaurants)

		r.Post("/devices/register", h.registerDevice)
		r.Get("/devices", h.listDevices)

		r.Post("/roles", h.createRole)
		r.Get("/roles", h.listRoles)

		r.Post("/employees", h.createEmployee)
		r.Get("/employees", h.listEmployees)
		r.Patch("/employees/{id}/archive", h.archiveEmployee)

		r.Post("/auth/pin-login", h.pinLogin)
		r.Post("/auth/logout", h.logout)
		r.Get("/auth/session", h.getAuthSession)

		r.Post("/system/pair", h.pairEdgeNode)
		r.Get("/system/pairing-status", h.getPairingStatus)

		r.Post("/halls", h.createHall)
		r.Get("/halls", h.listHalls)
		r.Patch("/halls/{id}/archive", h.archiveHall)
		r.Post("/tables", h.createTable)
		r.Get("/tables", h.listTables)
		r.Patch("/tables/{id}/archive", h.archiveTable)

		r.Post("/catalog/items", h.createCatalogItem)
		r.Get("/catalog/items", h.listCatalogItems)

		r.Post("/menu/items", h.createMenuItem)
		r.Get("/menu/items", h.listMenuItems)

		r.Post("/shifts/open", h.openShift)
		r.Post("/shifts/{id}/close", h.closeShift)
		r.Get("/shifts/current", h.currentShift)

		r.Post("/orders", h.createOrder)
		r.Get("/orders/current", h.getCurrentOrder)
		r.Get("/orders/{id}", h.getOrder)
		r.Post("/orders/{id}/lines", h.addOrderLine)
		r.Patch("/orders/{id}/lines/{line_id}", h.changeOrderLineQuantity)
		r.Post("/orders/{id}/lines/{line_id}/void", h.voidOrderLine)
		r.Post("/orders/{id}/precheck", h.issuePrecheck)
		r.Get("/orders/{id}/prechecks", h.listPrechecksByOrder)
		r.Post("/orders/{id}/check", h.issuePrecheckFromDeprecatedCheckAlias)
		r.Post("/orders/{id}/close", h.closeOrder)

		r.Get("/prechecks/{id}", h.getPrecheck)
		r.Post("/prechecks/{id}/cancel", h.cancelPrecheck)
		r.Post("/prechecks/{id}/payments", h.capturePrecheckPayment)

		r.Get("/checks/{id}", h.getCheck)
		r.Post("/checks/{id}/payments", h.captureLegacyCheckPayment)

		r.Post("/cash-sessions/open", h.openCashSession)
		r.Post("/cash-sessions/{id}/close", h.closeCashSession)
		r.Get("/cash-sessions/current", h.currentCashSession)
		r.Post("/cash-drawer-events", h.recordCashDrawerEvent)

		r.Get("/sync/outbox", h.listOutbox)
		r.Post("/sync/outbox/{id}/mark-sent", h.markOutboxSent)
		r.Post("/sync/outbox/{id}/mark-failed", h.markOutboxFailed)
		r.Get("/sync/local-events", h.listLocalEvents)
		r.Get("/sync/status", h.syncStatus)
		r.Post("/sync/retry-failed", h.retryFailedOutbox)
	})

	return r
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "false")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Node-Device-ID, X-Client-Device-ID, X-Actor-Employee-ID, X-Session-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) createRestaurant(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRestaurantCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateRestaurant(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listRestaurants(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRestaurants(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	var cmd app.RegisterDeviceCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.RegisterDevice(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListDevices(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRoleCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateRole(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRoles(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) createEmployee(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateEmployeeCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateEmployee(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listEmployees(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListEmployees(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) archiveEmployee(w http.ResponseWriter, r *http.Request) {
	var cmd app.ArchiveEmployeeCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	if err := h.service.ArchiveEmployee(r.Context(), cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (h *Handler) pinLogin(w http.ResponseWriter, r *http.Request) {
	var cmd app.PinLoginCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.PinLogin(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var cmd app.LogoutCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.Logout(r.Context(), cmd)
	writeOK(w, v, err)
}

func (h *Handler) getAuthSession(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetSession(r.Context(), r.URL.Query().Get("session_id"), requestNodeDeviceID(r), requestClientDeviceID(r))
	writeOK(w, v, err)
}

func (h *Handler) pairEdgeNode(w http.ResponseWriter, r *http.Request) {
	var cmd app.PairEdgeNodeCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	v, err := h.service.PairEdgeNode(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) getPairingStatus(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetPairingStatus(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) createHall(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateHallCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateHall(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listHalls(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListHalls(r.Context(), r.URL.Query().Get("restaurant_id"))
	writeOK(w, v, err)
}

func (h *Handler) archiveHall(w http.ResponseWriter, r *http.Request) {
	var cmd app.ArchiveHallCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	if err := h.service.ArchiveHall(r.Context(), cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (h *Handler) createTable(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateTableCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateTable(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListTables(r.Context(), r.URL.Query().Get("restaurant_id"), r.URL.Query().Get("hall_id"))
	writeOK(w, v, err)
}

func (h *Handler) archiveTable(w http.ResponseWriter, r *http.Request) {
	var cmd app.ArchiveTableCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	if err := h.service.ArchiveTable(r.Context(), cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "archived"})
}

func (h *Handler) createCatalogItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCatalogItemCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateCatalogItem(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listCatalogItems(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListCatalogItems(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) createMenuItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateMenuItemCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateMenuItem(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listMenuItems(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListMenuItems(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) openShift(w http.ResponseWriter, r *http.Request) {
	var cmd app.OpenShiftCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.OpenShift(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) closeShift(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseShiftCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	v, err := h.service.CloseShift(r.Context(), cmd)
	writeOK(w, v, err)
}

func (h *Handler) currentShift(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentShift(r.Context(), r.URL.Query().Get("device_id"))
	writeOK(w, v, err)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateOrderCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.CreateOrder(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) getCurrentOrder(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentOrderByTable(r.Context(), requestNodeDeviceID(r), r.URL.Query().Get("table_id"))
	writeOK(w, v, err)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetOrder(r.Context(), chi.URLParam(r, "id"))
	writeOK(w, v, err)
}

func (h *Handler) addOrderLine(w http.ResponseWriter, r *http.Request) {
	var cmd app.AddOrderLineCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.AddOrderLine(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) changeOrderLineQuantity(w http.ResponseWriter, r *http.Request) {
	var cmd app.ChangeOrderLineQuantityCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	cmd.LineID = chi.URLParam(r, "line_id")
	v, err := h.service.ChangeOrderLineQuantity(r.Context(), cmd)
	writeOK(w, v, err)
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
	writeOK(w, v, err)
}

func (h *Handler) issuePrecheck(w http.ResponseWriter, r *http.Request) {
	var cmd app.IssuePrecheckCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.IssuePrecheck(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) issuePrecheckFromDeprecatedCheckAlias(w http.ResponseWriter, r *http.Request) {
	var legacy app.CreateCheckCommand
	if err := httpx.Decode(r, &legacy); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&legacy.CommandMeta, r)
	v, err := h.service.IssuePrecheck(r.Context(), app.IssuePrecheckCommand{
		CommandMeta: legacy.CommandMeta,
		OrderID:     chi.URLParam(r, "id"),
	})
	writeCreated(w, v, err)
}

func (h *Handler) closeOrder(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseOrderCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.CloseOrder(r.Context(), cmd)
	writeOK(w, v, err)
}

func (h *Handler) getPrecheck(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetPrecheck(r.Context(), chi.URLParam(r, "id"))
	writeOK(w, v, err)
}

func (h *Handler) listPrechecksByOrder(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListPrechecksByOrder(r.Context(), chi.URLParam(r, "id"))
	writeOK(w, v, err)
}

func (h *Handler) cancelPrecheck(w http.ResponseWriter, r *http.Request) {
	var cmd app.CancelPrecheckCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.PrecheckID = chi.URLParam(r, "id")
	v, err := h.service.CancelPrecheck(r.Context(), cmd)
	writeOK(w, v, err)
}

func (h *Handler) getCheck(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCheck(r.Context(), chi.URLParam(r, "id"))
	writeOK(w, v, err)
}

func (h *Handler) capturePrecheckPayment(w http.ResponseWriter, r *http.Request) {
	var cmd app.CapturePaymentCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.PrecheckID = chi.URLParam(r, "id")
	v, err := h.service.CapturePayment(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) captureLegacyCheckPayment(w http.ResponseWriter, r *http.Request) {
	httpx.Error(w, fmt.Errorf("%w: legacy check payment endpoint is disabled; use /api/v1/prechecks/{id}/payments", domain.ErrConflict))
}

func (h *Handler) openCashSession(w http.ResponseWriter, r *http.Request) {
	var cmd app.OpenCashSessionCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.OpenCashSession(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) closeCashSession(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseCashSessionCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	cmd.ID = chi.URLParam(r, "id")
	v, err := h.service.CloseCashSession(r.Context(), cmd)
	writeOK(w, v, err)
}

func (h *Handler) currentCashSession(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentCashSession(r.Context(), r.URL.Query().Get("device_id"))
	writeOK(w, v, err)
}

func (h *Handler) recordCashDrawerEvent(w http.ResponseWriter, r *http.Request) {
	var cmd app.RecordCashDrawerEventCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	setRequestMeta(&cmd.CommandMeta, r)
	v, err := h.service.RecordCashDrawerEvent(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listOutbox(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := h.service.ListOutbox(r.Context(), limit)
	writeOK(w, v, err)
}

func (h *Handler) listLocalEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := h.service.ListLocalEvents(r.Context(), app.ListLocalEventsQuery{
		Limit:     limit,
		EventType: r.URL.Query().Get("event_type"),
	})
	writeOK(w, v, err)
}

func (h *Handler) syncStatus(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetSyncStatus(r.Context())
	writeOK(w, v, err)
}

func (h *Handler) retryFailedOutbox(w http.ResponseWriter, r *http.Request) {
	n, err := h.service.RetryFailedOutbox(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int{"retried": n})
}

func (h *Handler) markOutboxSent(w http.ResponseWriter, r *http.Request) {
	if err := h.service.MarkOutboxSent(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) markOutboxFailed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Error string `json:"error"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.service.MarkOutboxFailed(r.Context(), chi.URLParam(r, "id"), body.Error); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "failed"})
}

func writeCreated(w http.ResponseWriter, v any, err error) {
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, v)
}

func writeOK(w http.ResponseWriter, v any, err error) {
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

func setEdgeOrigin(meta *app.CommandMeta) {
	meta.Origin = app.OriginEdgeDevice
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
	return r.URL.Query().Get("device_id")
}

func requestClientDeviceID(r *http.Request) string {
	if v := r.Header.Get("X-Client-Device-ID"); v != "" {
		return v
	}
	return r.URL.Query().Get("client_device_id")
}
