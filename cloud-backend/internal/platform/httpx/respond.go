package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/masterdata/domain"
	provisioningdomain "cloud-backend/internal/provisioning/domain"
)

type ErrorBody struct {
	Code          string            `json:"code"`
	MessageKey    string            `json:"message_key"`
	Details       map[string]string `json:"details,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		value := reflect.ValueOf(v)
		if value.Kind() == reflect.Slice && value.IsNil() {
			_, _ = w.Write([]byte("[]\n"))
			return
		}
		_ = json.NewEncoder(w).Encode(v)
	}
}

func Error(w http.ResponseWriter, err error, requests ...*http.Request) {
	status, body := ClassifyError(err)
	var r *http.Request
	if len(requests) > 0 {
		r = requests[0]
	}
	if r != nil {
		body.CorrelationID = middleware.GetReqID(r.Context())
		if body.CorrelationID != "" {
			w.Header().Set("X-Request-ID", body.CorrelationID)
		}
		logAPIError(r, status, body.Code, err)
	}
	w.Header().Set("X-Error-Code", body.Code)
	JSON(w, status, ErrorResponse{Error: body})
}

func ClassifyError(err error) (int, ErrorBody) {
	status := statusForError(err)
	code := codeForError(err, status)
	return status, ErrorBody{Code: code, MessageKey: messageKeyForCode(code, messageKeyForStatus(status))}
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, contracts.ErrSyncUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, contracts.ErrSyncForbidden):
		return http.StatusForbidden
	case errors.Is(err, contracts.ErrSyncRevisionAhead), errors.Is(err, contracts.ErrSyncCheckpointConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrPINAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, provisioningdomain.ErrLicenseServerUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, provisioningdomain.ErrPairingCodeInvalid), errors.Is(err, provisioningdomain.ErrPairingCodeExpired), errors.Is(err, provisioningdomain.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, provisioningdomain.ErrDeviceNotAssigned), errors.Is(err, provisioningdomain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, provisioningdomain.ErrConflict), errors.Is(err, provisioningdomain.ErrSnapshotNotPublished):
		return http.StatusConflict
	case errors.Is(err, domain.ErrInvalid), errors.Is(err, contracts.ErrInvalidEnvelope):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, contracts.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, contracts.ErrPayloadConflict):
		return http.StatusConflict
	default:
		if contains(err, "license server unavailable") {
			return http.StatusServiceUnavailable
		}
		if contains(err, "olap runtime unavailable") {
			return http.StatusServiceUnavailable
		}
		if contains(err, "pairing code expired") || contains(err, "pairing code invalid") {
			return http.StatusBadRequest
		}
		if contains(err, "snapshot not published") {
			return http.StatusConflict
		}
		if contains(err, "device not assigned") {
			return http.StatusNotFound
		}
		return http.StatusInternalServerError
	}
}

func codeForError(err error, status int) string {
	switch {
	case errors.Is(err, contracts.ErrSyncUnauthorized):
		return "SYNC_UNAUTHORIZED"
	case errors.Is(err, contracts.ErrSyncForbidden):
		return "SYNC_FORBIDDEN"
	case errors.Is(err, contracts.ErrSyncRevisionAhead):
		return "SYNC_REVISION_AHEAD"
	case errors.Is(err, contracts.ErrSyncCheckpointConflict):
		return "SYNC_CHECKPOINT_CONFLICT"
	case errors.Is(err, domain.ErrPINAlreadyExists):
		return "PIN_ALREADY_EXISTS"
	case errors.Is(err, provisioningdomain.ErrLicenseServerUnavailable):
		return "LICENSE_SERVER_UNAVAILABLE"
	case errors.Is(err, provisioningdomain.ErrPairingCodeInvalid):
		return "PAIRING_CODE_INVALID"
	case errors.Is(err, provisioningdomain.ErrPairingCodeExpired):
		return "PAIRING_CODE_EXPIRED"
	case errors.Is(err, provisioningdomain.ErrDeviceNotAssigned):
		return "DEVICE_NOT_ASSIGNED"
	case errors.Is(err, provisioningdomain.ErrSnapshotNotPublished):
		return "SNAPSHOT_NOT_PUBLISHED"
	case errors.Is(err, provisioningdomain.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, provisioningdomain.ErrInvalid):
		return "VALIDATION_FAILED"
	case errors.Is(err, provisioningdomain.ErrConflict):
		return "CONFLICT"
	case contains(err, "device already registered"):
		return "DEVICE_ALREADY_REGISTERED"
	case contains(err, "device not assigned"):
		return "DEVICE_NOT_ASSIGNED"
	case contains(err, "pairing code invalid"):
		return "PAIRING_CODE_INVALID"
	case contains(err, "pairing code expired"):
		return "PAIRING_CODE_EXPIRED"
	case contains(err, "license server unavailable"):
		return "LICENSE_SERVER_UNAVAILABLE"
	case contains(err, "olap runtime unavailable"):
		return "OLAP_UNAVAILABLE"
	case contains(err, "snapshot not published"):
		return "SNAPSHOT_NOT_PUBLISHED"
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, contracts.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, domain.ErrInvalid), errors.Is(err, contracts.ErrInvalidEnvelope):
		return "VALIDATION_FAILED"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, contracts.ErrPayloadConflict):
		return "CONFLICT"
	default:
		if status == http.StatusNotFound {
			return "NOT_FOUND"
		}
		return "INTERNAL_ERROR"
	}
}

func messageKeyForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "errors.validation"
	case http.StatusNotFound:
		return "errors.notFound"
	case http.StatusConflict:
		return "errors.conflict"
	case http.StatusUnauthorized:
		return "errors.auth.unauthorized"
	case http.StatusForbidden:
		return "errors.auth.forbidden"
	case http.StatusServiceUnavailable:
		return "errors.infrastructure.unavailable"
	default:
		return "errors.server"
	}
}

func messageKeyForCode(code, fallback string) string {
	switch code {
	case "SYNC_UNAUTHORIZED":
		return "errors.sync.unauthorized"
	case "SYNC_FORBIDDEN":
		return "errors.sync.forbidden"
	case "SYNC_REVISION_AHEAD":
		return "errors.sync.revisionAhead"
	case "SYNC_CHECKPOINT_CONFLICT":
		return "errors.sync.checkpointConflict"
	case "PIN_ALREADY_EXISTS":
		return "errors.employee.pinAlreadyExists"
	case "DEVICE_ALREADY_REGISTERED":
		return "errors.device.alreadyRegistered"
	case "DEVICE_NOT_ASSIGNED":
		return "errors.device.notAssigned"
	case "PAIRING_CODE_INVALID":
		return "errors.pairing.invalid"
	case "PAIRING_CODE_EXPIRED":
		return "errors.pairing.expired"
	case "LICENSE_SERVER_UNAVAILABLE":
		return "errors.license.unavailable"
	case "SNAPSHOT_NOT_PUBLISHED":
		return "errors.masterData.snapshotNotPublished"
	case "VALIDATION_FAILED":
		return "errors.validation"
	case "NOT_FOUND":
		return "errors.notFound"
	case "CONFLICT":
		return "errors.conflict"
	default:
		return fallback
	}
}

func logAPIError(r *http.Request, status int, code string, err error) {
	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	slog.Log(r.Context(), level, "cloud api returned safe error",
		"request_id", middleware.GetReqID(r.Context()),
		"operation", "http.error",
		"action", r.Method+" "+r.URL.Path,
		"result", "rejected",
		"status", status,
		"error_code", code,
		"internal_error", err.Error(),
	)
}

func contains(err error, needle string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(fmt.Sprint(err)), strings.ToLower(needle))
}
