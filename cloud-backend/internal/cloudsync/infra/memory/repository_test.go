package memory_test

import (
	"context"
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

func ptr(v string) *string {
	return &v
}
