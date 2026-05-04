package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	httpx "pos-backend/internal/platform/http"
	"pos-backend/internal/pos/app"
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

		r.Post("/catalog/items", h.createCatalogItem)
		r.Get("/catalog/items", h.listCatalogItems)

		r.Post("/menu/items", h.createMenuItem)
		r.Get("/menu/items", h.listMenuItems)

		r.Post("/shifts/open", h.openShift)
		r.Post("/shifts/{id}/close", h.closeShift)
		r.Get("/shifts/current", h.currentShift)

		r.Post("/orders", h.createOrder)
		r.Get("/orders/{id}", h.getOrder)
		r.Post("/orders/{id}/lines", h.addOrderLine)
		r.Post("/orders/{id}/check", h.createCheck)
		r.Post("/orders/{id}/close", h.closeOrder)

		r.Get("/checks/{id}", h.getCheck)
		r.Post("/checks/{id}/payments", h.capturePayment)

		r.Get("/sync/outbox", h.listOutbox)
		r.Post("/sync/outbox/{id}/mark-sent", h.markOutboxSent)
		r.Post("/sync/outbox/{id}/mark-failed", h.markOutboxFailed)
	})

	return r
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createRestaurant(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRestaurantCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
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
	cmd.ID = chi.URLParam(r, "id")
	if err := h.service.ArchiveEmployee(r.Context(), cmd); err != nil {
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
	v, err := h.service.OpenShift(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) closeShift(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseShiftCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
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
	v, err := h.service.CreateOrder(r.Context(), cmd)
	writeCreated(w, v, err)
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
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.AddOrderLine(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) createCheck(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCheckCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.CreateCheck(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) closeOrder(w http.ResponseWriter, r *http.Request) {
	var cmd app.CloseOrderCommand
	if r.Body != nil {
		_ = httpx.Decode(r, &cmd)
	}
	cmd.OrderID = chi.URLParam(r, "id")
	v, err := h.service.CloseOrder(r.Context(), cmd)
	writeOK(w, v, err)
}

func (h *Handler) getCheck(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCheck(r.Context(), chi.URLParam(r, "id"))
	writeOK(w, v, err)
}

func (h *Handler) capturePayment(w http.ResponseWriter, r *http.Request) {
	var cmd app.CapturePaymentCommand
	if err := httpx.Decode(r, &cmd); err != nil {
		httpx.Error(w, err)
		return
	}
	cmd.CheckID = chi.URLParam(r, "id")
	v, err := h.service.CapturePayment(r.Context(), cmd)
	writeCreated(w, v, err)
}

func (h *Handler) listOutbox(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := h.service.ListOutbox(r.Context(), limit)
	writeOK(w, v, err)
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
