package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestMaskIDMasksLongValues(t *testing.T) {
	if got := MaskID("1234567890abcdef"); got != "12345678..." {
		t.Fatalf("expected masked value, got %q", got)
	}
	if got := MaskID("short"); got != "short" {
		t.Fatalf("expected short value unchanged, got %q", got)
	}
}

func TestResultAndErrorCodeMapping(t *testing.T) {
	if got := ResultFromStatus(200); got != "success" {
		t.Fatalf("expected success, got %q", got)
	}
	if got := ResultFromStatus(429); got != "rejected" {
		t.Fatalf("expected rejected, got %q", got)
	}
	if got := ResultFromStatus(500); got != "failed" {
		t.Fatalf("expected failed, got %q", got)
	}
	if got := ErrorCodeFromStatus(200); got != "" {
		t.Fatalf("expected empty error code, got %q", got)
	}
	if got := ErrorCodeFromStatus(403); got != "HTTP_403" {
		t.Fatalf("expected HTTP_403, got %q", got)
	}
}

func TestLogWritesNormalizedFieldsAndMasksIDs(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: LevelTrace}))
	Log(context.Background(), logger, slog.LevelInfo, "test event", Event{
		Operation:       "sync.sender",
		Action:          "message.ack",
		Result:          "success",
		ErrorCode:       "",
		RequestID:       "req-1",
		NodeDeviceID:    "node-device-123456",
		ClientDeviceID:  "client-device-123456",
		SessionID:       "session-123456",
		ActorEmployeeID: "employee-123456",
	}, "event_type", "OrderCreated")
	raw := out.String()
	for _, want := range []string{
		`"operation":"sync.sender"`,
		`"action":"message.ack"`,
		`"result":"success"`,
		`"request_id":"req-1"`,
		`"node_device_id":"node-dev..."`,
		`"client_device_id":"client-d..."`,
		`"session_id":"session-..."`,
		`"actor_employee_id":"employee..."`,
		`"event_type":"OrderCreated"`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("expected log to contain %q, logs=%s", want, raw)
		}
	}
}
