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

func TestPostBatchEdgeEventsReturnsItemLevelAck(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	valid := []byte(`{
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
	  "payload":{"origin":"edge_device","data":{"id":"order-1","edge_order_id":"edge-order-1","restaurant_id":"restaurant-1","device_id":"device-1","shift_id":"shift-1","status":"open","table_name":"A1","guest_count":2,"opened_at":"2026-05-05T09:00:00Z","created_at":"2026-05-05T09:00:00Z","updated_at":"2026-05-05T09:00:00Z"}}
	}`)
	reqBody := []byte(`{"items":[` + string(valid) + `,{"version":"1"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/edge-events/batch", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var ack contracts.BatchEventAck
	if err := json.Unmarshal(rec.Body.Bytes(), &ack); err != nil {
		t.Fatal(err)
	}
	if ack.Status != "partial" || len(ack.Items) != 2 {
		t.Fatalf("unexpected batch ack: %+v", ack)
	}
	if ack.Items[0].Status != contracts.BatchItemAccepted {
		t.Fatalf("expected first item accepted, got %+v", ack.Items[0])
	}
	if ack.Items[1].Status != contracts.BatchItemRejected || ack.Items[1].ErrorCode != "INVALID_ENVELOPE" {
		t.Fatalf("expected second item rejected by validation, got %+v", ack.Items[1])
	}
}

func TestProvisioningMasterDataPutAndGet(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	putBody := []byte(`{
	  "node_device_id":"node-1",
	  "restaurant_id":"restaurant-1",
	  "sync_mode":"full_snapshot",
	  "cloud_version":12,
	  "payload_json":{"catalog_items":[{"id":"cat-1","name":"Tea"}]}
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/provisioning/master-data/catalog", bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on upsert, got %d: %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/provisioning/master-data/catalog?node_device_id=node-1", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var pkg contracts.MasterDataPackage
	if err := json.Unmarshal(getRec.Body.Bytes(), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.StreamName != contracts.MasterDataStreamCatalog || pkg.CloudVersion != 12 {
		t.Fatalf("unexpected package response: %+v", pkg)
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
