package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/masterdata/app"
)

func (h *Handler) listPrinters(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListPrinters(r.Context(), app.PrinterFilter{
		RestaurantID: r.URL.Query().Get("restaurant_id"),
	})
	write(w, http.StatusOK, v, err)
}

func (h *Handler) createPrinter(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreatePrinterCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreatePrinter(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) updatePrinter(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdatePrinterCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdatePrinter(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) deactivatePrinter(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.DeactivatePrinter(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}
