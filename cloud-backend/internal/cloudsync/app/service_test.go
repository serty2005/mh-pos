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
	if first.IdempotencyKey != "restaurant-1:device-1:018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("unexpected idempotency key %q", first.IdempotencyKey)
	}
	if first.EdgeEventID != first.EventID || first.EventID != "018f0000-0000-7000-8000-000000000001" {
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

func TestExchangeAcksServedAndClosedInventoryEventsIdempotently(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	served := sampleItemServedEnvelope(t)
	closed := sampleCheckClosedEnvelope(t)

	exchange := func() contracts.SyncExchangeResponse {
		resp, err := service.Exchange(context.Background(), contracts.SyncExchangeRequest{
			ProtocolVersion: contracts.SyncExchangeProtocolVersion,
			NodeDeviceID:    "device-1",
			RestaurantID:    "restaurant-1",
			EdgeEvents: []contracts.SyncExchangeEdgeEvent{
				{ClientItemID: "outbox-served", Payload: json.RawMessage(served)},
				{ClientItemID: "outbox-closed", Payload: json.RawMessage(closed)},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	first := exchange()
	replay := exchange()
	if first.Status != contracts.SyncExchangeStatusAccepted || replay.Status != contracts.SyncExchangeStatusAccepted {
		t.Fatalf("expected accepted exchanges, first=%+v replay=%+v", first, replay)
	}
	if len(first.EdgeAcks) != 2 || len(replay.EdgeAcks) != 2 {
		t.Fatalf("expected per-item ACK for served and closed events, first=%+v replay=%+v", first.EdgeAcks, replay.EdgeAcks)
	}
	for i := range first.EdgeAcks {
		if first.EdgeAcks[i].Status != contracts.BatchItemAccepted || replay.EdgeAcks[i].Status != contracts.BatchItemAccepted {
			t.Fatalf("expected accepted item ACKs, first=%+v replay=%+v", first.EdgeAcks, replay.EdgeAcks)
		}
		if first.EdgeAcks[i].Ack == nil || replay.EdgeAcks[i].Ack == nil {
			t.Fatalf("expected stable ACK payloads, first=%+v replay=%+v", first.EdgeAcks, replay.EdgeAcks)
		}
		if first.EdgeAcks[i].Ack.CloudReceiptID != replay.EdgeAcks[i].Ack.CloudReceiptID {
			t.Fatalf("expected replay to reuse cloud receipt id for %s, first=%+v replay=%+v", first.EdgeAcks[i].ClientItemID, first.EdgeAcks[i].Ack, replay.EdgeAcks[i].Ack)
		}
	}
	if got := repo.InventoryQueueCount(); got != 2 {
		t.Fatalf("expected ItemServed and CheckClosed to enter inventory queue once each after replay, got %d", got)
	}
	if repo.Count() != 2 {
		t.Fatalf("expected replay to keep two accepted receipts, got %d", repo.Count())
	}
}

func TestExchangeLimitsCloudPackagesPerSession(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewServiceWithOptions(repo, fixedClock{}, app.Options{MaxCloudPackagesPerExchange: 2})
	streams := []string{
		contracts.MasterDataStreamRestaurants,
		contracts.MasterDataStreamCatalog,
		contracts.MasterDataStreamMenu,
	}
	for idx, stream := range streams {
		if _, err := service.UpsertMasterDataPackage(context.Background(), contracts.MasterDataPackage{
			StreamName:      stream,
			NodeDeviceID:    "device-1",
			RestaurantID:    "restaurant-1",
			SyncMode:        contracts.SyncModeIncremental,
			CloudVersion:    int64(10 + idx),
			CheckpointToken: stream + ":new",
			PayloadJSON:     payloadForStream(stream),
		}); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := service.Exchange(context.Background(), contracts.SyncExchangeRequest{
		ProtocolVersion: contracts.SyncExchangeProtocolVersion,
		NodeDeviceID:    "device-1",
		RestaurantID:    "restaurant-1",
		Streams: []contracts.SyncExchangeStreamRequest{
			{StreamName: contracts.MasterDataStreamRestaurants, LastCloudVersion: 1},
			{StreamName: contracts.MasterDataStreamCatalog, LastCloudVersion: 1},
			{StreamName: contracts.MasterDataStreamMenu, LastCloudVersion: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.CloudPackages) != 2 {
		t.Fatalf("expected two cloud packages in bounded exchange, got %+v", resp.CloudPackages)
	}
	if len(resp.StreamResults) != 3 {
		t.Fatalf("expected stream results for all requested streams, got %+v", resp.StreamResults)
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

func TestReceiveBatchStoresProblemItemsForAnalysis(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})

	ack := service.ReceiveBatch(context.Background(), [][]byte{
		sampleEnvelope(t),
		[]byte(`{"version":"1"}`),
	})
	if ack.Status != "partial" {
		t.Fatalf("expected partial status, got %+v", ack)
	}
	problems := repo.ProblemEdgeEvents()
	if len(problems) != 1 {
		t.Fatalf("expected one stored problem item, got %+v", problems)
	}
	if problems[0].Direction != "edge_to_cloud" || problems[0].ErrorCode != "INVALID_ENVELOPE" {
		t.Fatalf("unexpected problem item: %+v", problems[0])
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
	projected, err := service.ListFinancialOperations(context.Background(), app.FinancialOperationProjectionFilter{
		RestaurantID:     "restaurant-1",
		BusinessDateFrom: "2026-05-06",
		BusinessDateTo:   "2026-05-06",
		OperationType:    "refund",
		ShiftID:          "shift-refund-1",
		CheckID:          "check-1",
		Limit:            10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(projected) != 1 || projected[0].OperationID != "financial-operation-1" || projected[0].EdgeOperationID == "" || projected[0].Amount != 1000 {
		t.Fatalf("unexpected refund operation projection: %+v", projected)
	}
}

func TestReceiveInventoryEventEnqueuesOnceOnReplay(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	raw := sampleCheckClosedEnvelope(t)

	first, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Receive(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("expected stable ack on CheckClosed replay\nfirst=%+v\nsecond=%+v", first, second)
	}
	if got := repo.InventoryQueueCount(); got != 1 {
		t.Fatalf("expected one inventory queue row after replay, got %d", got)
	}
}

func TestReceiveCancellationRecordedReplaysIdempotentlyAndKeepsCurrentEventStats(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	raw := sampleFinancialOperationEnvelope(t, contracts.EventCancellationRecorded, "018f0000-0000-7000-8000-0000000000c1", "command-cancel-1", "financial-operation-cancel-1", "cancellation", 1000, "shift-sale-1", "2026-05-05")

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
	if got := string(repo.RawPayload(first.CloudReceiptID)); got != string(raw) {
		t.Fatalf("raw cancellation payload was not preserved\nwant=%s\ngot=%s", raw, got)
	}
	stats := repo.EventTypeStats()
	if len(stats) != 1 || stats[0].EventType != string(contracts.EventCancellationRecorded) || stats[0].EventCount != 1 {
		t.Fatalf("unexpected cancellation event stats: %+v", stats)
	}
	if finance := repo.ShiftFinance(); len(finance) != 0 {
		t.Fatalf("cancellation should not update coarse refund shift finance projection, got %+v", finance)
	}
	projected, err := service.ListFinancialOperations(context.Background(), app.FinancialOperationProjectionFilter{
		RestaurantID:    "restaurant-1",
		OperationType:   "cancellation",
		OriginalShiftID: "shift-sale-1",
		Limit:           10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(projected) != 1 || projected[0].OperationType != "cancellation" || projected[0].CheckID != "check-1" {
		t.Fatalf("unexpected cancellation operation projection: %+v", projected)
	}
	refunds, err := service.ListFinancialOperations(context.Background(), app.FinancialOperationProjectionFilter{
		RestaurantID:  "restaurant-1",
		OperationType: "refund",
		Limit:         10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(refunds) != 0 {
		t.Fatalf("cancellation must not appear in refund projection filter, got %+v", refunds)
	}
}

func TestLegacyRefundEventsDoNotPopulateFinancialOperationProjection(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	if _, err := service.Receive(context.Background(), sampleLegacyPaymentRefundedEnvelope(t)); err != nil {
		t.Fatal(err)
	}
	projected, err := service.ListFinancialOperations(context.Background(), app.FinancialOperationProjectionFilter{
		RestaurantID: "restaurant-1",
		Limit:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(projected) != 0 {
		t.Fatalf("legacy PaymentRefunded must not become financial operation projection, got %+v", projected)
	}
	finance := repo.ShiftFinance()
	if len(finance) != 1 || finance[0].PaymentsRefundedCount != 1 || finance[0].PaymentsRefundedTotal != 1000 {
		t.Fatalf("legacy refund compatibility should remain only in coarse counters, got %+v", finance)
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
		"event_id":          "018f0000-0000-7000-8000-000000000001",
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

func payloadForStream(stream string) json.RawMessage {
	switch stream {
	case contracts.MasterDataStreamRestaurants:
		return json.RawMessage(`{"restaurants":[{"id":"restaurant-1","name":"Cafe","timezone":"Europe/Moscow","currency":"RUB","active":true}]}`)
	case contracts.MasterDataStreamCatalog:
		return json.RawMessage(`{"catalog_items":[{"id":"cat-1","type":"dish","name":"Tea","sku":"tea","base_unit":"pc","active":true}]}`)
	case contracts.MasterDataStreamMenu:
		return json.RawMessage(`{"menu_items":[{"id":"menu-1","catalog_item_id":"cat-1","name":"Tea","price":100,"currency":"RUB","active":true}]}`)
	default:
		return json.RawMessage(`{}`)
	}
}

func sampleCheckClosedEnvelope(t *testing.T) []byte {
	t.Helper()
	body := map[string]any{
		"version":        "1",
		"event_id":       "018f0000-0000-7000-8000-0000000000a1",
		"command_id":     "command-check-closed-1",
		"event_type":     string(contracts.EventCheckClosed),
		"aggregate_type": "Check",
		"aggregate_id":   "check-1",
		"restaurant_id":  "restaurant-1",
		"device_id":      "device-1",
		"node_device_id": "device-1",
		"shift_id":       "shift-1",
		"occurred_at":    "2026-05-05T09:00:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"check_id":            "check-1",
				"order_id":            "order-1",
				"precheck_id":         "precheck-1",
				"restaurant_id":       "restaurant-1",
				"business_date_local": "2026-05-05",
				"closed_at":           "2026-05-05T09:00:00Z",
				"items": []map[string]any{{
					"order_line_id":          "line-1",
					"catalog_item_id":        "item-1",
					"quantity":               "2.000",
					"unit_code":              "PC",
					"required_for_inventory": true,
				}},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func sampleItemServedEnvelope(t *testing.T) []byte {
	t.Helper()
	body := map[string]any{
		"version":        "1",
		"event_id":       "018f0000-0000-7000-8000-0000000000a0",
		"command_id":     "command-item-served-1",
		"event_type":     string(contracts.EventItemServed),
		"aggregate_type": "KitchenTicket",
		"aggregate_id":   "ticket-1",
		"restaurant_id":  "restaurant-1",
		"device_id":      "device-1",
		"node_device_id": "device-1",
		"shift_id":       "shift-1",
		"occurred_at":    "2026-05-05T08:55:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"served_event_id": "018f0000-0000-7000-8000-0000000000a0",
				"order_id":        "order-1",
				"order_line_id":   "line-1",
				"catalog_item_id": "item-1",
				"quantity":        "2.000",
				"unit_code":       "PC",
				"served_at":       "2026-05-05T08:55:00Z",
				"station_id":      "kitchen-hot",
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
	return sampleFinancialOperationEnvelope(t, contracts.EventRefundRecorded, "018f0000-0000-7000-8000-0000000000f1", "command-refund-1", "financial-operation-1", "refund", 1000, "shift-refund-1", "2026-05-06")
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
				"id":                     operationID,
				"edge_operation_id":      "edge-" + operationID,
				"restaurant_id":          "restaurant-1",
				"device_id":              "device-1",
				"shift_id":               shiftID,
				"original_shift_id":      "shift-sale-1",
				"check_id":               "check-1",
				"precheck_id":            "precheck-1",
				"operation_type":         operationType,
				"operation_kind":         "full",
				"status":                 "recorded",
				"amount":                 amount,
				"currency":               "RUB",
				"business_date_local":    businessDate,
				"inventory_disposition":  "no_stock_effect",
				"reason":                 "guest return",
				"created_by_employee_id": "manager-1",
				"snapshot":               map[string]any{"document_type": "financial_operation", "check_id": "check-1"},
				"created_at":             businessDate + "T09:00:00Z",
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func sampleLegacyPaymentRefundedEnvelope(t *testing.T) []byte {
	t.Helper()
	body := map[string]any{
		"version":           "1",
		"event_id":          "018f0000-0000-7000-8000-0000000000e1",
		"command_id":        "command-legacy-refund-1",
		"event_type":        string(contracts.EventPaymentRefunded),
		"aggregate_type":    "Payment",
		"aggregate_id":      "payment-1",
		"restaurant_id":     "restaurant-1",
		"device_id":         "device-1",
		"node_device_id":    "device-1",
		"client_device_id":  "client-1",
		"actor_employee_id": "manager-1",
		"session_id":        "session-1",
		"shift_id":          "shift-refund-1",
		"occurred_at":       "2026-05-06T09:00:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"id":                  "payment-1",
				"edge_payment_id":     "edge-payment-1",
				"restaurant_id":       "restaurant-1",
				"device_id":           "device-1",
				"shift_id":            "shift-refund-1",
				"precheck_id":         "precheck-1",
				"method":              "cash",
				"amount":              1000,
				"currency":            "RUB",
				"status":              "refunded",
				"business_date_local": "2026-05-06",
				"created_at":          "2026-05-06T09:00:00Z",
				"updated_at":          "2026-05-06T09:00:00Z",
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
