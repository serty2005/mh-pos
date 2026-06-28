package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/masterdata/app"
)

func (h *Handler) createReceiptTemplate(w http.ResponseWriter, r *http.Request) {
	var cmd app.CreateReceiptTemplateCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.CreateReceiptTemplate(r.Context(), cmd)
	write(w, http.StatusCreated, v, err)
}

func (h *Handler) listReceiptTemplates(w http.ResponseWriter, r *http.Request) {
	filter := app.ReceiptTemplateFilter{
		OrgID:        r.URL.Query().Get("org_id"),
		RestaurantID: r.URL.Query().Get("restaurant_id"),
		DocumentType: r.URL.Query().Get("document_type"),
		IsDefault:    boolQuery(r, "is_default"),
		IsActive:     boolQuery(r, "is_active"),
	}
	v, err := h.service.ListReceiptTemplates(r.Context(), filter)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) getReceiptTemplate(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetReceiptTemplate(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

func (h *Handler) updateReceiptTemplate(w http.ResponseWriter, r *http.Request) {
	var cmd app.UpdateReceiptTemplateCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.UpdateReceiptTemplate(r.Context(), chi.URLParam(r, "id"), cmd)
	write(w, http.StatusOK, v, err)
}

func (h *Handler) deleteReceiptTemplate(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.DeactivateReceiptTemplate(r.Context(), chi.URLParam(r, "id"))
	write(w, http.StatusOK, v, err)
}

// boolQuery возвращает указатель на bool-фильтр, если query-параметр задан валидно.
func boolQuery(r *http.Request, key string) *bool {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &v
}
