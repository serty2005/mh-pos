package logging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Event задает нормализованные поля telemetry для non-HTTP операций.
type Event struct {
	Operation       string
	Action          string
	Result          string
	ErrorCode       string
	RequestID       string
	NodeDeviceID    string
	ClientDeviceID  string
	SessionID       string
	ActorEmployeeID string
}

// Log записывает нормализованное structured telemetry event.
func Log(ctx context.Context, logger *slog.Logger, level slog.Level, message string, e Event, extra ...any) {
	if logger == nil {
		logger = slog.Default()
	}
	args := []any{
		"operation", strings.TrimSpace(e.Operation),
		"action", strings.TrimSpace(e.Action),
		"result", strings.TrimSpace(e.Result),
		"error_code", strings.TrimSpace(e.ErrorCode),
		"request_id", strings.TrimSpace(e.RequestID),
		"node_device_id", MaskID(e.NodeDeviceID),
		"client_device_id", MaskID(e.ClientDeviceID),
		"session_id", MaskID(e.SessionID),
		"actor_employee_id", MaskID(e.ActorEmployeeID),
	}
	args = append(args, extra...)
	logger.Log(ctx, level, message, args...)
}

// ErrorCodeFromStatus сопоставляет status codes с нормализованным error code.
func ErrorCodeFromStatus(status int) string {
	if status >= 200 && status < 400 {
		return ""
	}
	return fmt.Sprintf("HTTP_%d", status)
}

// ResultFromStatus сопоставляет HTTP status с нормализованным operation result.
func ResultFromStatus(status int) string {
	if status >= 200 && status < 300 {
		return "success"
	}
	if status >= 400 && status < 500 {
		return "rejected"
	}
	return "failed"
}

// MaskID маскирует identifiers для безопасных логов, сохраняя возможность correlation.
func MaskID(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if len(v) <= 8 {
		return v
	}
	return v[:8] + "..."
}
