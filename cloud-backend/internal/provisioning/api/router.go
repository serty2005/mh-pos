package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/platform/httpx"
	"cloud-backend/internal/provisioning/app"
)

type Handler struct {
	service *app.Service
}

func RegisterRoutes(r chi.Router, service *app.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	r.Post("/devices/register", h.registerDevice)
	r.Get("/devices/unassigned", h.listUnassigned)
	r.Post("/restaurants/{restaurant_id}/devices/{node_device_id}/assign", h.assignDevice)
	r.Get("/devices/{node_device_id}/assignment-status", h.assignmentStatus)
	r.Post("/restaurants/{restaurant_id}/devices/generate-pairing-code", h.generatePairingCode)
}

func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	var cmd app.RegisterDeviceCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.RegisterDevice(r.Context(), cmd)
	write(w, r, http.StatusOK, v, err)
}

func (h *Handler) listUnassigned(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListUnassigned(r.Context())
	write(w, r, http.StatusOK, v, err)
}

func (h *Handler) assignDevice(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.AssignDevice(r.Context(), chi.URLParam(r, "restaurant_id"), chi.URLParam(r, "node_device_id"))
	write(w, r, http.StatusOK, v, err)
}

func (h *Handler) assignmentStatus(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.AssignmentStatus(r.Context(), chi.URLParam(r, "node_device_id"))
	write(w, r, http.StatusOK, v, err)
}

func (h *Handler) generatePairingCode(w http.ResponseWriter, r *http.Request) {
	var cmd app.GeneratePairingCodeCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.GeneratePairingCode(r.Context(), chi.URLParam(r, "restaurant_id"), cmd)
	write(w, r, http.StatusCreated, v, err)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		httpx.Error(w, err, r)
		return false
	}
	return true
}

func write(w http.ResponseWriter, r *http.Request, status int, v any, err error) {
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, status, v)
}
