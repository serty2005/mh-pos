package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/platform/httpx"
	"cloud-backend/internal/provisioning/app"
)

type Handler struct {
	service *app.Service
}

// MasterDataPackageService описывает общий storage Cloud-authored provisioning packages.
type MasterDataPackageService interface {
	UpsertMasterDataPackage(context.Context, contracts.MasterDataPackage) (contracts.MasterDataPackage, error)
	GetMasterDataPackage(context.Context, string, string) (contracts.MasterDataPackage, error)
}

func RegisterRoutes(r chi.Router, service *app.Service) {
	if service == nil {
		return
	}
	h := &Handler{service: service}
	r.Post("/devices/register", h.registerDevice)
	r.Get("/devices/unassigned", h.listUnassigned)
	r.Get("/restaurants/{restaurant_id}/devices", h.listRestaurantDevices)
	r.Post("/restaurants/{restaurant_id}/devices/{node_device_id}/assign", h.assignDevice)
	r.Get("/devices/{node_device_id}/assignment-status", h.assignmentStatus)
	r.Post("/restaurants/{restaurant_id}/devices/generate-pairing-code", h.generatePairingCode)
	r.Post("/devices/pairing/consume", h.consumePairingCode)
}

// RegisterMasterDataPackageRoutes подключает generic GET/PUT для Cloud-authored provisioning packages.
func RegisterMasterDataPackageRoutes(r chi.Router, service MasterDataPackageService) {
	if service == nil {
		return
	}
	h := &masterDataPackageHandler{service: service}
	r.Put("/provisioning/master-data/{stream}", h.upsertMasterDataPackage)
	r.Get("/provisioning/master-data/{stream}", h.getMasterDataPackage)
}

type masterDataPackageHandler struct {
	service MasterDataPackageService
}

func (h *masterDataPackageHandler) upsertMasterDataPackage(w http.ResponseWriter, r *http.Request) {
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
		httpx.Error(w, fmt.Errorf("%w: %v", contracts.ErrInvalidEnvelope, err), r)
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
	writePackage(w, r, http.StatusOK, v, err)
}

func (h *masterDataPackageHandler) getMasterDataPackage(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.GetMasterDataPackage(r.Context(), chi.URLParam(r, "stream"), r.URL.Query().Get("node_device_id"))
	writePackage(w, r, http.StatusOK, v, err)
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

func (h *Handler) listRestaurantDevices(w http.ResponseWriter, r *http.Request) {
	v, err := h.service.ListRestaurantDevices(r.Context(), chi.URLParam(r, "restaurant_id"))
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

func (h *Handler) consumePairingCode(w http.ResponseWriter, r *http.Request) {
	var cmd app.PairingConsumeCommand
	if !decode(w, r, &cmd) {
		return
	}
	v, err := h.service.ConsumePairingCode(r.Context(), cmd)
	write(w, r, http.StatusOK, v, err)
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

func writePackage(w http.ResponseWriter, r *http.Request, status int, v contracts.MasterDataPackage, err error) {
	if err != nil {
		httpx.Error(w, err, r)
		return
	}
	httpx.JSON(w, status, v)
}
