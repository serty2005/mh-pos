package memory_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/cloudsync/infra/memory"
)

func TestRepositoryDedupesByIdempotencyKey(t *testing.T) {
	repo := memory.NewRepository()
	receipt := app.EdgeEventReceipt{
		Envelope: contracts.SyncEnvelope{
			Version:      "1",
			EventID:      "event-1",
			CommandID:    "command-1",
			RestaurantID: ptr("restaurant-1"),
			DeviceID:     "device-1",
		},
		IdempotencyKey:   "restaurant-1:device-1:event-1",
		RawPayload:       []byte(`{"event_id":"event-1"}`),
		RawPayloadSHA256: "hash-1",
		CloudReceivedAt:  time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
	}

	first, err := repo.ReceiveEdgeEvent(context.Background(), receipt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.ReceiveEdgeEvent(context.Background(), receipt)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("expected stable ack on duplicate, first=%+v second=%+v", first, second)
	}
	if repo.Count() != 1 {
		t.Fatalf("expected one stored event, got %d", repo.Count())
	}
}

func TestRepositoryTracksProjections(t *testing.T) {
	repo := memory.NewRepository()
	receipt := app.EdgeEventReceipt{
		Envelope: contracts.SyncEnvelope{
			Version:      "1",
			EventID:      "event-payment-1",
			CommandID:    "command-payment-1",
			EventType:    contracts.EventPaymentCaptured,
			RestaurantID: ptr("restaurant-1"),
			DeviceID:     "device-1",
			ShiftID:      ptr("shift-1"),
			OccurredAt:   time.Date(2026, 5, 5, 9, 0, 0, 0, time.UTC),
			Payload:      mustPayloadJSON(t, map[string]any{"id": "payment-1", "amount": 1500}),
		},
		IdempotencyKey:   "restaurant-1:device-1:event-payment-1",
		RawPayload:       []byte(`{"event_id":"event-payment-1"}`),
		RawPayloadSHA256: "hash-1",
		CloudReceivedAt:  time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
	}
	if _, err := repo.ReceiveEdgeEvent(context.Background(), receipt); err != nil {
		t.Fatal(err)
	}
	refund := receipt
	refund.Envelope.EventID = "event-payment-refund-1"
	refund.Envelope.CommandID = "command-payment-refund-1"
	refund.Envelope.EventType = contracts.EventPaymentRefunded
	refund.Envelope.Payload = mustPayloadJSON(t, map[string]any{"id": "payment-1", "amount": 1500, "status": "refunded"})
	refund.IdempotencyKey = "restaurant-1:device-1:event-payment-refund-1"
	refund.RawPayload = []byte(`{"event_id":"event-payment-refund-1"}`)
	refund.RawPayloadSHA256 = "hash-2"
	if _, err := repo.ReceiveEdgeEvent(context.Background(), refund); err != nil {
		t.Fatal(err)
	}
	stats := repo.EventTypeStats()
	if len(stats) != 2 {
		t.Fatalf("unexpected event stats: %+v", stats)
	}
	finance := repo.ShiftFinance()
	if len(finance) != 1 || finance[0].PaymentsCapturedCount != 1 || finance[0].PaymentsCapturedTotal != 1500 || finance[0].PaymentsRefundedCount != 1 || finance[0].PaymentsRefundedTotal != 1500 {
		t.Fatalf("unexpected shift finance projection: %+v", finance)
	}
}

func TestRepositoryUpsertAndGetMasterDataPackage(t *testing.T) {
	repo := memory.NewRepository()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	stored, err := repo.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:         contracts.MasterDataStreamCatalog,
		NodeDeviceID:       "node-1",
		SyncMode:           contracts.SyncModeFullSnapshot,
		FullSnapshotReason: contracts.FullSnapshotReasonTerminalRestaurantChanged,
		CloudVersion:       2,
		PayloadJSON:        json.RawMessage(`{"catalog_items":[{"id":"c-1"}]}`),
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetMasterDataPackage(context.Background(), contracts.MasterDataStreamCatalog, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.CloudVersion != stored.CloudVersion || string(got.PayloadJSON) != string(stored.PayloadJSON) {
		t.Fatalf("unexpected package, stored=%+v got=%+v", stored, got)
	}
}

func ptr(v string) *string {
	return &v
}

func mustPayloadJSON(t *testing.T, data map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"origin": "edge_device",
		"data":   data,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
