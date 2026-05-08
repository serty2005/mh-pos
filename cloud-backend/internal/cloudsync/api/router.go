package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
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
	r.Use(middleware.Recoverer)

	r.Get("/health", h.health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/sync/edge-events", h.receiveEdgeEvent)
		r.Post("/sync/edge-events/batch", h.receiveEdgeEventBatch)
		r.Put("/provisioning/master-data/{stream}", h.upsertMasterDataPackage)
		r.Get("/provisioning/master-data/{stream}", h.getMasterDataPackage)
	})
	return r
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

func (h *Handler) upsertMasterDataPackage(w http.ResponseWriter, r *http.Request) {
	streamName := chi.URLParam(r, "stream")
	var req struct {
		NodeDeviceID       string          `json:"node_device_id"`
		RestaurantID       string          `json:"restaurant_id"`
		SyncMode           string          `json:"sync_mode"`
		FullSnapshotReason string          `json:"full_snapshot_reason"`
		CloudVersion       int64           `json:"cloud_version"`
		CheckpointToken    string          `json:"checkpoint_token"`
		CloudUpdatedAt     *time.Time      `json:"cloud_updated_at"`
		PayloadJSON        json.RawMessage `json:"payload_json"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%w: %v", contracts.ErrInvalidEnvelope, err))
		return
	}
	v, err := h.service.UpsertMasterDataPackage(r.Context(), contracts.MasterDataPackage{
		StreamName:         streamName,
		NodeDeviceID:       req.NodeDeviceID,
		RestaurantID:       req.RestaurantID,
		SyncMode:           req.SyncMode,
		FullSnapshotReason: req.FullSnapshotReason,
		CloudVersion:       req.CloudVersion,
		CheckpointToken:    req.CheckpointToken,
		CloudUpdatedAt:     req.CloudUpdatedAt,
		PayloadJSON:        req.PayloadJSON,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, contracts.ErrInvalidEnvelope) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) getMasterDataPackage(w http.ResponseWriter, r *http.Request) {
	streamName := chi.URLParam(r, "stream")
	nodeDeviceID := r.URL.Query().Get("node_device_id")
	v, err := h.service.GetMasterDataPackage(r.Context(), streamName, nodeDeviceID)
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
	writeJSON(w, http.StatusOK, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
