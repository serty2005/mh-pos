package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"

	"pos-backend/internal/pos/domain"
)

// ErrorBody описывает безопасный контракт ошибки для UI и диагностики поддержки.
type ErrorBody struct {
	Code          string            `json:"code"`
	MessageKey    string            `json:"message_key"`
	Details       map[string]string `json:"details,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
}

// ErrorResponse задает единый JSON-конверт для всех безопасных API-ошибок.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// JSON записывает JSON-ответ без дополнительной бизнес-логики.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// Decode читает JSON-body строго по контракту, чтобы расхождение payload между UI и backend не проходило молча.
func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	return nil
}

// Error возвращает безопасную API-ошибку и пишет внутреннюю причину только в structured log.
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

// ClassifyError преобразует internal/domain error в безопасный HTTP status и стабильный error code.
func ClassifyError(err error) (int, ErrorBody) {
	status := statusForError(err)
	body := ErrorBody{
		Code:       codeForError(err, status),
		MessageKey: messageKeyForStatus(status),
	}
	body.MessageKey = messageKeyForCode(body.Code, body.MessageKey)
	return status, body
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		if isSessionRequiredError(err) {
			return http.StatusUnauthorized
		}
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrForbidden):
		if isSessionRevokedError(err) || isSessionRequiredError(err) {
			return http.StatusUnauthorized
		}
		return http.StatusForbidden
	case errors.Is(err, domain.ErrTooManyRequests):
		return http.StatusTooManyRequests
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrDuplicate), errors.Is(err, domain.ErrDuplicateCommand):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func codeForError(err error, status int) string {
	switch {
	case errors.Is(err, domain.ErrTooManyRequests):
		return "RATE_LIMITED"
	case errors.Is(err, domain.ErrDuplicateCommand):
		return "DUPLICATE_COMMAND"
	case status == http.StatusUnauthorized && isSessionRevokedError(err):
		return "SESSION_REVOKED"
	case status == http.StatusUnauthorized:
		return "SESSION_REQUIRED"
	case errors.Is(err, domain.ErrForbidden) && isSessionContextMismatchError(err):
		return "SESSION_CONTEXT_MISMATCH"
	case errors.Is(err, domain.ErrForbidden) && isPermissionDeniedError(err):
		return "PERMISSION_DENIED"
	case errors.Is(err, domain.ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, domain.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, domain.ErrInvalid):
		return "VALIDATION_FAILED"
	case errors.Is(err, domain.ErrConflict) && containsErrorText(err, "pin must uniquely"):
		return "DUPLICATE_PIN"
	case errors.Is(err, domain.ErrConflict) && containsErrorText(err, "active precheck"):
		return "ACTIVE_PRECHECK_CONFLICT"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrDuplicate):
		return "CONFLICT"
	default:
		return "INTERNAL_ERROR"
	}
}

func messageKeyForStatus(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "errors.session.required"
	case http.StatusForbidden:
		return "errors.permission"
	case http.StatusNotFound:
		return "errors.not_found"
	case http.StatusConflict:
		return "errors.conflict"
	case http.StatusTooManyRequests:
		return "errors.rateLimit"
	case http.StatusBadRequest:
		return "errors.validation"
	default:
		return "errors.server"
	}
}

func messageKeyForCode(code, fallback string) string {
	switch code {
	case "SESSION_REVOKED":
		return "errors.session.revoked"
	case "SESSION_REQUIRED":
		return "errors.session.required"
	case "SESSION_CONTEXT_MISMATCH":
		return "errors.session.contextMismatch"
	case "PERMISSION_DENIED", "FORBIDDEN":
		return "errors.permission"
	case "RATE_LIMITED":
		return "errors.rateLimit"
	case "DUPLICATE_PIN":
		return "errors.conflict_duplicate_pin"
	case "ACTIVE_PRECHECK_CONFLICT":
		return "errors.conflict_active_precheck"
	case "DUPLICATE_COMMAND":
		return "errors.conflict_duplicate_command"
	case "VALIDATION_FAILED":
		return "errors.validation"
	case "NOT_FOUND":
		return "errors.not_found"
	case "INTERNAL_ERROR":
		return "errors.server"
	default:
		return fallback
	}
}

func logAPIError(r *http.Request, status int, code string, err error) {
	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	slog.Log(r.Context(), level, "api вернул безопасную ошибку",
		"request_id", middleware.GetReqID(r.Context()),
		"operation", "http.error",
		"action", r.Method+" "+r.URL.Path,
		"result", "rejected",
		"status", status,
		"error_code", code,
		"node_device_id", maskLogID(requestValue(r, "X-Node-Device-ID", "node_device_id")),
		"client_device_id", maskLogID(requestValue(r, "X-Client-Device-ID", "client_device_id")),
		"session_id", maskLogID(r.Header.Get("X-Session-ID")),
		"actor_employee_id", maskLogID(r.Header.Get("X-Actor-Employee-ID")),
		"internal_error", err.Error(),
	)
}

func isSessionRequiredError(err error) bool {
	return containsErrorText(err, "session_id") || containsErrorText(err, "operator flow")
}

func isSessionRevokedError(err error) bool {
	return containsErrorText(err, "session is not active")
}

func isSessionContextMismatchError(err error) bool {
	return containsErrorText(err, "session context does not match") || containsErrorText(err, "requested device context")
}

func isPermissionDeniedError(err error) bool {
	return containsErrorText(err, "permission ") && containsErrorText(err, " is required")
}

func containsErrorText(err error, needle string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(needle))
}

func requestValue(r *http.Request, header, query string) string {
	if v := r.Header.Get(header); v != "" {
		return v
	}
	return r.URL.Query().Get(query)
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
