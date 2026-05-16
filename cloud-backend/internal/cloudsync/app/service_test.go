package app_test

import (
	"context"
	"encoding/json"
	"errors"
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

func TestExchangeReturnsItemAckAndNewerCloudPackage(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	if err := repo.AuthorizeNodeForTest("device-1", "restaurant-1", "node-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:      contracts.MasterDataStreamCatalog,
		NodeDeviceID:    "device-1",
		RestaurantID:    "restaurant-1",
		SyncMode:        contracts.SyncModeIncremental,
		CloudVersion:    7,
		CheckpointToken: "catalog:7",
		PayloadJSON:     json.RawMessage(`{"catalog_items":[{"id":"c-1","name":"Tea"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := service.Exchange(context.Background(), contracts.SyncExchangeRequest{
		ProtocolVersion: contracts.SyncExchangeProtocolVersion,
		NodeDeviceID:    "device-1",
		RestaurantID:    "restaurant-1",
		EdgeEvents: []contracts.SyncExchangeEdgeEvent{
			{ClientItemID: "outbox-1", Payload: json.RawMessage(sampleEnvelope(t))},
			{ClientItemID: "outbox-bad", Payload: json.RawMessage(`{"version":"1"}`)},
		},
		Streams: []contracts.SyncExchangeStreamRequest{
			{StreamName: contracts.MasterDataStreamCatalog, LastCloudVersion: 6, CheckpointToken: "catalog:6"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != contracts.SyncExchangeStatusPartial {
		t.Fatalf("expected partial exchange status, got %+v", resp)
	}
	if len(resp.EdgeAcks) != 2 || resp.EdgeAcks[0].Status != contracts.BatchItemAccepted || resp.EdgeAcks[1].Status != contracts.BatchItemRejected {
		t.Fatalf("unexpected edge acks: %+v", resp.EdgeAcks)
	}
	if len(resp.CloudPackages) != 1 || resp.CloudPackages[0].StreamName != contracts.MasterDataStreamCatalog || resp.CloudPackages[0].CloudVersion != 7 {
		t.Fatalf("expected one newer catalog package, got %+v", resp.CloudPackages)
	}
	if len(resp.StreamResults) != 1 || resp.StreamResults[0].Status != contracts.SyncExchangeStreamChanged {
		t.Fatalf("expected changed stream result, got %+v", resp.StreamResults)
	}
}

func TestExchangeRejectsRevisionAheadBeforeReceivingEdgeEvents(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	if _, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:      contracts.MasterDataStreamMenu,
		NodeDeviceID:    "node-1",
		RestaurantID:    "restaurant-1",
		SyncMode:        contracts.SyncModeIncremental,
		CloudVersion:    3,
		CheckpointToken: "menu:3",
		PayloadJSON:     json.RawMessage(`{"menu_items":[{"id":"m-1"}]}`),
	}); err != nil {
		t.Fatal(err)
	}

	_, err := service.Exchange(context.Background(), contracts.SyncExchangeRequest{
		ProtocolVersion: contracts.SyncExchangeProtocolVersion,
		NodeDeviceID:    "node-1",
		RestaurantID:    "restaurant-1",
		EdgeEvents:      []contracts.SyncExchangeEdgeEvent{{ClientItemID: "outbox-1", Payload: json.RawMessage(sampleEnvelope(t))}},
		Streams:         []contracts.SyncExchangeStreamRequest{{StreamName: contracts.MasterDataStreamMenu, LastCloudVersion: 4, CheckpointToken: "menu:4"}},
	})
	if !errors.Is(err, contracts.ErrSyncRevisionAhead) {
		t.Fatalf("expected revision-ahead error, got %v", err)
	}
	if repo.Count() != 0 {
		t.Fatalf("edge events must not be received when stream preflight fails, got %d", repo.Count())
	}
}

func TestReceiveRefundRecordedReplaysIdempotentlyAndUpdatesShiftFinance(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	raw := sampleRefundRecordedEnvelope(t)

	first, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("expected stable ack on RefundRecorded replay\nfirst=%+v\nsecond=%+v", first, second)
	}
	if repo.Count() != 1 {
		t.Fatalf("expected one refund receipt after replay, got %d", repo.Count())
	}
	if got := string(repo.RawPayload(first.CloudReceiptID)); got != string(raw) {
		t.Fatalf("raw refund payload was not preserved\nwant=%s\ngot=%s", raw, got)
	}
	stats := repo.EventTypeStats()
	if len(stats) != 1 || stats[0].EventType != string(contracts.EventRefundRecorded) || stats[0].EventCount != 1 {
		t.Fatalf("unexpected refund event stats: %+v", stats)
	}
	finance := repo.ShiftFinance()
	if len(finance) != 1 || finance[0].ChecksRefundedCount != 1 || finance[0].ChecksRefundedTotal != 1000 {
		t.Fatalf("unexpected refund shift finance projection: %+v", finance)
	}
}

func TestReceiveCancellationRecordedReplaysIdempotentlyAndKeepsCurrentEventStats(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	raw := sampleFinancialOperationEnvelope(t, contracts.EventCancellationRecorded, "event-cancel-1", "command-cancel-1", "financial-operation-cancel-1", "cancellation", 1000, "shift-sale-1", "2026-05-05")

	first, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("expected stable ack on CancellationRecorded replay\nfirst=%+v\nsecond=%+v", first, second)
	}
	if repo.Count() != 1 {
		t.Fatalf("expected one cancellation receipt after replay, got %d", repo.Count())
	}
	stats := repo.EventTypeStats()
	if len(stats) != 1 || stats[0].EventType != string(contracts.EventCancellationRecorded) || stats[0].EventCount != 1 {
		t.Fatalf("unexpected cancellation event stats: %+v", stats)
	}
	if finance := repo.ShiftFinance(); len(finance) != 0 {
		t.Fatalf("cancellation should not update coarse refund shift finance projection, got %+v", finance)
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

func TestUpsertAndGetPricingPolicyMasterDataPackage(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	payload := json.RawMessage(`{
		"tax_profiles":[{"id":"tax-vat-10","name":"VAT 10","tax_exempt":false,"active":true}],
		"tax_rules":[{"id":"tax-rule-10","tax_profile_id":"tax-vat-10","name":"VAT 10","kind":"percentage","mode":"exclusive","rate_basis_points":1000,"active":true}],
		"service_charge_rules":[{"id":"svc-10","restaurant_id":"restaurant-1","name":"Service 10","kind":"service_charge","amount_kind":"percentage","value_basis_points":1000,"active":true}]
	}`)

	stored, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:   contracts.MasterDataStreamPricing,
		NodeDeviceID: "node-1",
		RestaurantID: "restaurant-1",
		SyncMode:     contracts.SyncModeIncremental,
		CloudVersion: 11,
		PayloadJSON:  payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.GetMasterDataPackage(context.Background(), contracts.MasterDataStreamPricing, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.StreamName != contracts.MasterDataStreamPricing || got.CloudVersion != 11 || string(got.PayloadJSON) != string(payload) {
		t.Fatalf("unexpected pricing policy package: stored=%+v got=%+v", stored, got)
	}
}

func TestUpsertMasterDataPackageRejectsUnknownPayloadShape(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	_, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
		StreamName:   contracts.MasterDataStreamMenu,
		SyncMode:     contracts.SyncModeIncremental,
		CloudVersion: 12,
		PayloadJSON:  json.RawMessage(`{"menu_items":[{"id":"m-1"}],"shadow_stream":[{"id":"x"}]}`),
	})
	if err == nil {
		t.Fatal("expected unknown payload shape to be rejected")
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

func sampleRefundRecordedEnvelope(t *testing.T) []byte {
	t.Helper()
	return sampleFinancialOperationEnvelope(t, contracts.EventRefundRecorded, "event-refund-1", "command-refund-1", "financial-operation-1", "refund", 1000, "shift-refund-1", "2026-05-06")
}

func sampleFinancialOperationEnvelope(t *testing.T, eventType contracts.EventType, eventID, commandID, operationID, operationType string, amount int64, shiftID, businessDate string) []byte {
	t.Helper()
	body := map[string]any{
		"version":           "1",
		"event_id":          eventID,
		"command_id":        commandID,
		"event_type":        string(eventType),
		"aggregate_type":    "FinancialOperation",
		"aggregate_id":      operationID,
		"restaurant_id":     "restaurant-1",
		"device_id":         "device-1",
		"node_device_id":    "device-1",
		"client_device_id":  "client-1",
		"actor_employee_id": "manager-1",
		"session_id":        "session-1",
		"shift_id":          shiftID,
		"occurred_at":       businessDate + "T09:00:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"id":                    operationID,
				"edge_operation_id":     "edge-" + operationID,
				"restaurant_id":         "restaurant-1",
				"device_id":             "device-1",
				"shift_id":              shiftID,
				"original_shift_id":     "shift-sale-1",
				"check_id":              "check-1",
				"precheck_id":           "precheck-1",
				"operation_type":        operationType,
				"operation_kind":        "full",
				"status":                "recorded",
				"amount":                amount,
				"currency":              "RUB",
				"business_date_local":   businessDate,
				"inventory_disposition": "no_stock_effect",
				"reason":                "guest return",
				"created_at":            businessDate + "T09:00:00Z",
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
