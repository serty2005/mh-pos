package contracts_test

import (
	"encoding/json"
	"errors"
	"testing"

	"cloud-backend/internal/cloudsync/contracts"
)

func TestIdempotencyKeyUsesRestaurantDeviceAndEdgeEventID(t *testing.T) {
	envelope := validEnvelope(t, "PaymentCaptured")
	key, err := contracts.IdempotencyKey(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if key != "restaurant-1:device-1:event-1" {
		t.Fatalf("unexpected key %q", key)
	}
	if contracts.EdgeEventID(envelope) != envelope.EventID {
		t.Fatalf("edge_event_id must be the envelope event_id")
	}
}

func TestValidateEnvelopeRejectsUnknownEvent(t *testing.T) {
	envelope := validEnvelope(t, "PaymentCaptured")
	envelope.EventType = "RefundCaptured"
	err := contracts.ValidateEnvelope(envelope)
	if !errors.Is(err, contracts.ErrInvalidEnvelope) {
		t.Fatalf("expected invalid envelope, got %v", err)
	}
}

func validEnvelope(t *testing.T, eventType contracts.EventType) contracts.SyncEnvelope {
	t.Helper()
	raw := []byte(`{
	  "version":"1",
	  "event_id":"event-1",
	  "command_id":"command-1",
	  "event_type":"PaymentCaptured",
	  "aggregate_type":"Payment",
	  "aggregate_id":"payment-1",
	  "restaurant_id":"restaurant-1",
	  "device_id":"device-1",
	  "node_device_id":"device-1",
	  "client_device_id":"client-1",
	  "actor_employee_id":"employee-1",
	  "session_id":"session-1",
	  "shift_id":"shift-1",
	  "occurred_at":"2026-05-05T09:00:00Z",
	  "payload":{
	    "origin":"edge_device",
	    "data":{
	      "id":"payment-1",
	      "precheck_id":"precheck-1",
	      "method":"cash",
	      "amount":1000,
	      "currency":"RUB",
	      "status":"captured",
	      "created_at":"2026-05-05T09:00:00Z",
	      "updated_at":"2026-05-05T09:00:00Z"
	    }
	  }
	}`)
	var envelope contracts.SyncEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.EventType = eventType
	return envelope
}
