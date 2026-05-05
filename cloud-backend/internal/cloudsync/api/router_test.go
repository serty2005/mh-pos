package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud-backend/internal/cloudsync/api"
	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/cloudsync/infra/memory"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

func TestPostDuplicateEnvelopeDoesNotCreateDuplicateReceipt(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	raw := []byte(`{
	  "version":"1",
	  "event_id":"event-1",
	  "command_id":"command-1",
	  "event_type":"OrderCreated",
	  "aggregate_type":"Order",
	  "aggregate_id":"order-1",
	  "restaurant_id":"restaurant-1",
	  "device_id":"device-1",
	  "shift_id":"shift-1",
	  "occurred_at":"2026-05-05T09:00:00Z",
	  "payload":{
	    "origin":"edge_device",
	    "data":{
	      "id":"order-1",
	      "edge_order_id":"edge-order-1",
	      "restaurant_id":"restaurant-1",
	      "device_id":"device-1",
	      "shift_id":"shift-1",
	      "status":"open",
	      "table_name":"A1",
	      "guest_count":2,
	      "opened_at":"2026-05-05T09:00:00Z",
	      "created_at":"2026-05-05T09:00:00Z",
	      "updated_at":"2026-05-05T09:00:00Z"
	    }
	  }
	}`)

	first := postEnvelope(t, router, raw)
	second := postEnvelope(t, router, raw)
	if first != second {
		t.Fatalf("expected stable ack on replay\nfirst=%+v\nsecond=%+v", first, second)
	}
	if repo.Count() != 1 {
		t.Fatalf("expected one business receipt, got %d", repo.Count())
	}
	if got := string(repo.RawPayload(first.CloudReceiptID)); got != string(bytes.TrimSpace(raw)) {
		t.Fatalf("raw payload was not preserved\nwant=%s\ngot=%s", bytes.TrimSpace(raw), got)
	}
}

func postEnvelope(t *testing.T, h http.Handler, raw []byte) contracts.EventAck {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/edge-events", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var ack contracts.EventAck
	if err := json.Unmarshal(rec.Body.Bytes(), &ack); err != nil {
		t.Fatal(err)
	}
	return ack
}
