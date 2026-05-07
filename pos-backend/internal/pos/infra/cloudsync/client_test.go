package cloudsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/syncsender"
)

func TestSendBatchMapsItemLevelAckStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sync/edge-events/batch" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{
			"status":"partial",
			"items":[
				{"index":0,"status":"accepted","ack":{"status":"accepted","event_id":"event-1"}},
				{"index":1,"status":"rejected","error":"bad envelope"},
				{"index":2,"status":"retryable","error":"cloud temporary"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL + "/api/v1/sync/edge-events")
	results, err := client.SendBatch(context.Background(), []domain.OutboxMessage{
		{ID: "o1", PayloadJSON: `{"event_id":"event-1"}`},
		{ID: "o2", PayloadJSON: `{"event_id":"event-2"}`},
		{ID: "o3", PayloadJSON: `{"event_id":"event-3"}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 batch results, got %+v", results)
	}
	if results[0].Status != syncsender.BatchSendAccepted {
		t.Fatalf("expected accepted result, got %+v", results[0])
	}
	if results[1].Status != syncsender.BatchSendRejected {
		t.Fatalf("expected rejected result, got %+v", results[1])
	}
	if results[2].Status != syncsender.BatchSendRetryable {
		t.Fatalf("expected retryable result, got %+v", results[2])
	}
}
