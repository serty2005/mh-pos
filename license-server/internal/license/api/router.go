package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"license-server/internal/license/app"
)

type Handler struct {
	service *app.Service
}

func NewRouter(service *app.Service) http.Handler {
	h := &Handler{service: service}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/pairing-codes", h.register)
		r.Post("/pairing-codes/resolve", h.resolve)
	})
	return r
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var cmd app.RegisterPairingCodeCommand
	if err := decode(r, &cmd); err != nil {
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_INVALID", "errors.pairing.invalid")
		return
	}
	v, err := h.service.Register(r.Context(), cmd)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	var cmd app.ResolveCommand
	if err := decode(r, &cmd); err != nil {
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_INVALID", "errors.pairing.invalid")
		return
	}
	v, err := h.service.Resolve(r.Context(), cmd)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeAppError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, app.ErrExpired):
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_EXPIRED", "errors.pairing.expired")
	default:
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_INVALID", "errors.pairing.invalid")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, messageKey string) {
	w.Header().Set("X-Error-Code", code)
	writeJSON(w, status, map[string]any{"error": map[string]string{
		"code":           code,
		"message_key":    messageKey,
		"correlation_id": middleware.GetReqID(r.Context()),
	}})
}
