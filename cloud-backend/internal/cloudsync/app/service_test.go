package app_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/cloudsync/infra/memory"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

func TestReceiveDuplicateEnvelopeReturnsStableAckAndKeepsOneReceipt(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	raw := sampleEnvelope(t)

	first, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("expected stable ack on replay\nfirst=%+v\nsecond=%+v", first, second)
	}
	if repo.Count() != 1 {
		t.Fatalf("expected one receipt row, got %d", repo.Count())
	}
	if got := string(repo.RawPayload(first.CloudReceiptID)); got != string(raw) {
		t.Fatalf("raw payload was not preserved\nwant=%s\ngot=%s", raw, got)
	}
	if first.IdempotencyKey != "restaurant-1:device-1:event-1" {
		t.Fatalf("unexpected idempotency key %q", first.IdempotencyKey)
	}
	if first.EdgeEventID != first.EventID || first.EventID != "event-1" {
		t.Fatalf("expected edge_event_id to equal event_id, got %+v", first)
	}
}

func TestReceiveBatchReturnsItemLevelAck(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	valid := sampleEnvelope(t)
	invalid := []byte(`{"version":"1","event_type":"OrderCreated"}`)

	ack := service.ReceiveBatch(context.Background(), [][]byte{valid, invalid})
	if ack.Status != "partial" {
		t.Fatalf("expected partial status, got %+v", ack)
	}
	if len(ack.Items) != 2 {
		t.Fatalf("expected 2 ack items, got %+v", ack)
	}
	if ack.Items[0].Status != contracts.BatchItemAccepted || ack.Items[0].Ack == nil {
		t.Fatalf("expected first item accepted, got %+v", ack.Items[0])
	}
	if ack.Items[1].Status != contracts.BatchItemRejected || ack.Items[1].ErrorCode != "INVALID_ENVELOPE" {
		t.Fatalf("expected second item rejected by validation, got %+v", ack.Items[1])
	}
}

func TestUpsertAndGetMasterDataPackage(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	payload := json.RawMessage(`{"catalog_items":[{"id":"c-1","name":"Tea"}]}`)

	stored, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:   contracts.MasterDataStreamCatalog,
		NodeDeviceID: "node-1",
		SyncMode:     contracts.SyncModeIncremental,
		CloudVersion: 10,
		PayloadJSON:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stored.StreamName != contracts.MasterDataStreamCatalog || stored.CloudVersion != 10 {
		t.Fatalf("unexpected stored package: %+v", stored)
	}
	got, err := service.GetMasterDataPackage(context.Background(), contracts.MasterDataStreamCatalog, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.PayloadJSON) != string(payload) {
		t.Fatalf("unexpected payload: %s", got.PayloadJSON)
	}
}

func TestUpsertMasterDataPackageValidatesCurrenciesPayload(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	payload := json.RawMessage(`{
		"currencies":[
			{
				"currency_code":643,
				"currency_alpha_code":"RUB",
				"minor_unit":2,
				"currency_iso_name":"Russian Ruble",
				"currency_symbol":"₽",
				"curr_basic_name":"р",
				"curr_add_name":"коп.",
				"show_add":true,
				"show_currency_basic_name":true
			}
		]
	}`)
	if _, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:         contracts.MasterDataStreamCurrencies,
		SyncMode:           contracts.SyncModeFullSnapshot,
		FullSnapshotReason: contracts.FullSnapshotReasonNodeRoleChanged,
		CloudVersion:       1,
		PayloadJSON:        payload,
	}); err != nil {
		t.Fatalf("expected valid currencies package, got %v", err)
	}
	if _, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:         contracts.MasterDataStreamCurrencies,
		SyncMode:           contracts.SyncModeFullSnapshot,
		FullSnapshotReason: contracts.FullSnapshotReasonNodeRoleChanged,
		CloudVersion:       2,
		PayloadJSON:        json.RawMessage(`{"currencies":[]}`),
	}); err == nil {
		t.Fatal("expected empty currencies list to be rejected")
	}
}

func sampleEnvelope(t *testing.T) []byte {
	t.Helper()
	body := map[string]any{
		"version":           "1",
		"event_id":          "event-1",
		"command_id":        "command-1",
		"event_type":        "OrderCreated",
		"aggregate_type":    "Order",
		"aggregate_id":      "order-1",
		"restaurant_id":     "restaurant-1",
		"device_id":         "device-1",
		"node_device_id":    "device-1",
		"client_device_id":  "client-1",
		"actor_employee_id": "employee-1",
		"session_id":        "session-1",
		"shift_id":          "shift-1",
		"occurred_at":       "2026-05-05T09:00:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"id":            "order-1",
				"edge_order_id": "edge-order-1",
				"restaurant_id": "restaurant-1",
				"device_id":     "device-1",
				"shift_id":      "shift-1",
				"status":        "open",
				"table_name":    "A1",
				"guest_count":   2,
				"opened_at":     "2026-05-05T09:00:00Z",
				"created_at":    "2026-05-05T09:00:00Z",
				"updated_at":    "2026-05-05T09:00:00Z",
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
