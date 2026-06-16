package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

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
	r.Use(middleware.RealIP)
	r.Use(requestAuditLog)
	r.Use(recoverJSON)
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
		logPairingEvent(r, slog.LevelWarn, "license.pairing.register", "rejected", "decode_failed", "", "", "", cmd.PairingCode)
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_INVALID", "errors.pairing.invalid", err)
		return
	}
	logPairingEvent(r, slog.LevelInfo, "license.pairing.register", "started", "", cmd.RestaurantID, cmd.PairingID, cmd.CloudURL, cmd.PairingCode)
	v, err := h.service.Register(r.Context(), cmd)
	if err != nil {
		logPairingEvent(r, slog.LevelWarn, "license.pairing.register", "rejected", app.SafeErrorReason(err), cmd.RestaurantID, cmd.PairingID, cmd.CloudURL, cmd.PairingCode)
		writeAppError(w, r, err)
		return
	}
	logPairingEvent(r, slog.LevelInfo, "license.pairing.register", "success", "", cmd.RestaurantID, cmd.PairingID, cmd.CloudURL, cmd.PairingCode)
	writeJSON(w, http.StatusCreated, v)
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	var cmd app.ResolveCommand
	if err := decode(r, &cmd); err != nil {
		logPairingEvent(r, slog.LevelWarn, "license.pairing.resolve", "rejected", "decode_failed", "", "", "", cmd.PairingCode)
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_INVALID", "errors.pairing.invalid", err)
		return
	}
	logPairingEvent(r, slog.LevelInfo, "license.pairing.resolve", "started", "", "", "", "", cmd.PairingCode)
	v, err := h.service.Resolve(r.Context(), cmd)
	if err != nil {
		logPairingEvent(r, levelForAppError(err), "license.pairing.resolve", "rejected", app.SafeErrorReason(err), "", "", "", cmd.PairingCode)
		writeAppError(w, r, err)
		return
	}
	logPairingEvent(r, slog.LevelInfo, "license.pairing.resolve", "success", "", v.RestaurantID, v.PairingID, v.CloudURL, cmd.PairingCode)
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
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_EXPIRED", "errors.pairing.expired", err)
	case errors.Is(err, app.ErrInvalid), errors.Is(err, app.ErrConsumed):
		writeError(w, r, http.StatusBadRequest, "PAIRING_CODE_INVALID", "errors.pairing.invalid", err)
	default:
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.server", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, messageKey string, internalErr error) {
	w.Header().Set("X-Error-Code", code)
	logAPIError(r, status, code, internalErr)
	writeJSON(w, status, map[string]any{"error": map[string]string{
		"code":           code,
		"message_key":    messageKey,
		"correlation_id": middleware.GetReqID(r.Context()),
	}})
}

func levelForAppError(err error) slog.Level {
	if errors.Is(err, app.ErrInvalid) || errors.Is(err, app.ErrConsumed) || errors.Is(err, app.ErrExpired) {
		return slog.LevelWarn
	}
	return slog.LevelError
}

func logPairingEvent(r *http.Request, level slog.Level, operation, result, reason, restaurantID, pairingID, cloudURL, pairingCode string) {
	args := []any{
		"request_id", middleware.GetReqID(r.Context()),
		"operation", operation,
		"action", r.Method + " " + r.URL.Path,
		"result", result,
		"restaurant_id", maskLogID(restaurantID),
		"pairing_id", maskLogID(pairingID),
		"pairing_code_present", strings.TrimSpace(pairingCode) != "",
		"pairing_code_length", len(strings.TrimSpace(pairingCode)),
	}
	if reason != "" {
		args = append(args, "reason", reason)
	}
	if cloudURL != "" {
		args = append(args, "cloud_url", safeLogURL(cloudURL))
	}
	slog.Log(r.Context(), level, "license pairing flow event", args...)
}

func logAPIError(r *http.Request, status int, code string, err error) {
	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	args := []any{
		"request_id", middleware.GetReqID(r.Context()),
		"operation", "http.error",
		"action", r.Method + " " + r.URL.Path,
		"result", "rejected",
		"status", status,
		"error_code", code,
		"remote_ip", r.RemoteAddr,
	}
	if err != nil {
		args = append(args, "reason", app.SafeErrorReason(err), "internal_error", err.Error())
	}
	slog.Log(r.Context(), level, "license api returned safe error", args...)
}

func recoverJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Log(r.Context(), slog.LevelError, "license api panic handled with safe response",
					"request_id", middleware.GetReqID(r.Context()),
					"operation", "http.recover",
					"action", r.Method+" "+r.URL.Path,
					"result", "failed",
					"error_code", "INTERNAL_ERROR",
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
				)
				writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "errors.server", fmt.Errorf("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestAuditLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		slog.Log(r.Context(), level, "license http request completed",
			"request_id", middleware.GetReqID(r.Context()),
			"operation", "http.request",
			"action", r.Method+" "+r.URL.Path,
			"result", requestResult(rec.status),
			"error_code", requestErrorCode(rec.status, rec.Header().Get("X-Error-Code")),
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

func requestErrorCode(status int, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if status >= 200 && status < 400 {
		return ""
	}
	return fmt.Sprintf("HTTP_%d", status)
}

func maskLogID(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if len(v) <= 8 {
		return v
	}
	return v[:8] + "..."
}

func safeLogURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
