package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

// Handler содержит thin HTTP handlers для Cloud master-data API.
type Handler struct {
	service *app.Service
}

// RegisterRoutes подключает Cloud master-data routes к общему API router.
func RegisterRoutes(r chi.Router, service *app.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	r.Route("/master-data", func(r chi.Router) {
		r.Post("/roles", h.createRole)
		r.Post("/employees", h.createEmployee)
		r.Patch("/employees/{id}", h.updateEmployee)
		r.Post("/employees/{id}/suspend", h.suspendEmployee)
		r.Post("/employees/{id}/archive", h.archiveEmployee)
		r.Post("/employees/{id}/role", h.assignEmployeeRole)
		r.Post("/employees/{id}/pin", h.rotateEmployeePIN)
		r.Post("/catalog/items", h.createCatalogItem)
		r.Patch("/catalog/items/{id}", h.updateCatalogItem)
		r.Post("/menu/categories", h.createCategory)
		r.Post("/menu/items", h.createMenuItem)
		r.Patch("/menu/items/{id}", h.updateMenuItem)
		r.Post("/publications", h.publish)
		r.Get("/published", h.getPublished)
	})
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateRoleCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateRole(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) createEmployee(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateEmployeeCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateEmployee(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) updateEmployee(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateEmployeeCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateEmployee(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) suspendEmployee(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.SuspendEmployee(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) archiveEmployee(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ArchiveEmployee(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) assignEmployeeRole(w http.ResponseWriter, r *http.Request) {
	var cmd app.AssignRoleCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.AssignEmployeeRole(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) rotateEmployeePIN(w http.ResponseWriter, r *http.Request) {
	var cmd app.RotatePINCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RotateEmployeePIN(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createCatalogItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCatalogItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateCatalogItem(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) updateCatalogItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateCatalogItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateCatalogItem(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateCategoryCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateCategory(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) createMenuItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateMenuItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateMenuItem(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) updateMenuItem(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateMenuItemCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateMenuItem(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	var cmd app.PublishCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.Publish(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetCurrentPublishedState(r.Context(), r.URL.Query().Get("restaurant_id"))
	write(w, http.StatusOK, v, err)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%w: %v", domain.ErrInvalid, err))
		return false
	}
	return true
}

func write[T any](w http.ResponseWriter, status int, v T, err error) {
	if err != nil {
		code := http.StatusInternalServerError
		switch {
		case errors.Is(err, domain.ErrInvalid):
			code = http.StatusBadRequest
		case errors.Is(err, domain.ErrNotFound):
			code = http.StatusNotFound
		}
		writeError(w, code, err)
		return
	}
	writeJSON(w, status, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
