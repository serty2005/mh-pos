package cloudsync

import (
	"context"
	"encoding/json"
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

func TestExchangeSendsBearerTokenAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sync/exchange" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer node-token" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		var req syncsender.SyncExchangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.ProtocolVersion != syncsender.SyncExchangeProtocolVersion || req.NodeDeviceID != "node-1" || len(req.EdgeEvents) != 1 || len(req.Streams) != 1 {
			t.Fatalf("unexpected exchange request: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{
			"protocol_version":"sync_exchange.v1",
			"status":"partial",
			"edge_acks":[{"client_item_id":"outbox-1","status":"accepted","ack":{"status":"accepted","event_id":"event-1"}}],
			"stream_results":[{"stream_name":"catalog","status":"changed","cloud_version":9,"checkpoint_token":"catalog:9"}],
			"cloud_packages":[{"stream_name":"catalog","sync_mode":"incremental","cloud_version":9,"checkpoint_token":"catalog:9","payload_json":{"catalog_items":[{"id":"cat-1","name":"Tea"}]}}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL + "/api/v1/sync/edge-events")
	resp, err := client.Exchange(context.Background(), syncsender.SyncExchangeRequest{
		ProtocolVersion: syncsender.SyncExchangeProtocolVersion,
		NodeDeviceID:    "node-1",
		RestaurantID:    "restaurant-1",
		AuthToken:       "node-token",
		EdgeEvents: []syncsender.SyncExchangeEdgeEvent{
			{ClientItemID: "outbox-1", Payload: json.RawMessage(`{"event_id":"event-1"}`)},
		},
		Streams: []syncsender.SyncExchangeStreamRequest{
			{StreamName: "catalog", LastCloudVersion: 8, CheckpointToken: "catalog:8"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.EdgeAcks) != 1 || resp.EdgeAcks[0].Status != syncsender.BatchSendAccepted {
		t.Fatalf("unexpected edge acks: %+v", resp.EdgeAcks)
	}
	if len(resp.CloudPackages) != 1 || resp.CloudPackages[0].CloudVersion != 9 {
		t.Fatalf("unexpected cloud packages: %+v", resp.CloudPackages)
	}
}

func TestExchangeClassifiesHTTPFailures(t *testing.T) {
	cases := []struct {
		name          string
		status        int
		wantRetryable bool
	}{
		{name: "too many requests", status: http.StatusTooManyRequests, wantRetryable: true},
		{name: "server error", status: http.StatusInternalServerError, wantRetryable: true},
		{name: "bad request", status: http.StatusBadRequest, wantRetryable: false},
		{name: "unauthorized", status: http.StatusUnauthorized, wantRetryable: false},
		{name: "conflict", status: http.StatusConflict, wantRetryable: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":{"code":"TEST_ERROR","message_key":"errors.test"}}`))
			}))
			defer server.Close()
			_, err := NewClient(server.URL+"/api/v1/sync/edge-events").Exchange(context.Background(), syncsender.SyncExchangeRequest{
				ProtocolVersion: syncsender.SyncExchangeProtocolVersion,
				NodeDeviceID:    "node-1",
				RestaurantID:    "restaurant-1",
				AuthToken:       "node-token",
			})
			if err == nil {
				t.Fatal("expected exchange error")
			}
			_, nonRetryable := err.(syncsender.NonRetryableError)
			if tc.wantRetryable && nonRetryable {
				t.Fatalf("expected retryable error, got %T %v", err, err)
			}
			if !tc.wantRetryable && !nonRetryable {
				t.Fatalf("expected non-retryable error, got %T %v", err, err)
			}
		})
	}
}
