package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
)

type repositoryFixedClock struct{}

func (repositoryFixedClock) Now() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

func TestRepositoryReceiveEdgeEventIdempotencyAndRawPayloadPersistence(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openRepositoryWithBaseline(t, ctx)
	defer cleanup()
	service := app.NewService(repo, repositoryFixedClock{})
	raw := samplePaymentCapturedEnvelope(t, "018f0000-0000-7000-8000-000000100001", "command-payment-1", "payment-1", 1500, "shift-payment-1", "2026-05-05")

	first, err := service.Receive(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Receive(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	assertStableAck(t, first, second)
	if first.Status != "accepted" || first.CloudReceiptID == "" || first.RawPayloadSHA256Hex != rawSHA256(raw) {
		t.Fatalf("unexpected ACK: %+v", first)
	}

	for _, tt := range []struct {
		table string
		want  int64
	}{
		{table: "cloud_edge_event_receipts", want: 1},
		{table: "cloud_edge_event_raw_payloads", want: 1},
		{table: "inbox_events", want: 1},
		{table: "cloud_operational_events", want: 1},
		{table: "cloud_projection_event_type_stats", want: 1},
		{table: "cloud_projection_shift_finance", want: 1},
		{table: "cloud_projection_financial_operations", want: 0},
		{table: "inventory_event_queue", want: 0},
	} {
		t.Run(tt.table, func(t *testing.T) {
			if got := countTableRows(t, ctx, pool, tt.table); got != tt.want {
				t.Fatalf("unexpected %s row count: got %d want %d", tt.table, got, tt.want)
			}
		})
	}
	assertStoredJSONPayload(t, ctx, pool, "cloud_edge_event_raw_payloads", "receipt_id", first.CloudReceiptID, raw, first.RawPayloadSHA256Hex)
	assertStoredJSONPayload(t, ctx, pool, "inbox_events", "receipt_id", first.CloudReceiptID, raw, first.RawPayloadSHA256Hex)

	views, err := repo.ListEdgeEvents(ctx, app.EdgeEventListFilter{RestaurantID: "restaurant-1", DeviceID: "device-1", EventType: string(contracts.EventPaymentCaptured), Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].CloudReceiptID != first.CloudReceiptID || views[0].RawPayloadSHA256Hex != first.RawPayloadSHA256Hex {
		t.Fatalf("unexpected safe edge event view: %+v", views)
	}
	marshaled, err := json.Marshal(views)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(marshaled), "edge_payment_id") || strings.Contains(string(marshaled), `"payload":`) {
		t.Fatalf("edge event read model exposed raw payload: %s", marshaled)
	}
}

func TestRepositoryReceiveBatchItemLevelAckAndInvalidPayloadConsistency(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openRepositoryWithBaseline(t, ctx)
	defer cleanup()
	service := app.NewService(repo, repositoryFixedClock{})

	ack := service.ReceiveBatch(ctx, [][]byte{
		sampleOrderCreatedEnvelope(t, "018f0000-0000-7000-8000-000000200001", "command-order-batch-1", "order-batch-1"),
		[]byte(`{"version":"1"}`),
	})
	if ack.Status != "partial" || len(ack.Items) != 2 {
		t.Fatalf("unexpected batch ACK: %+v", ack)
	}
	if ack.Items[0].Status != contracts.BatchItemAccepted || ack.Items[0].Ack == nil {
		t.Fatalf("expected accepted first item ACK, got %+v", ack.Items[0])
	}
	if ack.Items[1].Status != contracts.BatchItemRejected || ack.Items[1].ErrorCode != "INVALID_ENVELOPE" {
		t.Fatalf("expected rejected invalid item, got %+v", ack.Items[1])
	}
	if got := countTableRows(t, ctx, pool, "cloud_edge_event_receipts"); got != 1 {
		t.Fatalf("expected only valid batch item receipt, got %d", got)
	}
	if got := countTableRows(t, ctx, pool, "cloud_sync_problem_events"); got != 1 {
		t.Fatalf("expected one problem event for invalid batch item, got %d", got)
	}

	before := tableCounts(t, ctx, pool, "cloud_edge_event_receipts", "cloud_edge_event_raw_payloads", "inbox_events", "cloud_operational_events", "cloud_projection_event_type_stats")
	bad := malformedProjectionReceipt(t)
	_, err := repo.ReceiveEdgeEvent(ctx, bad)
	if !errors.Is(err, contracts.ErrInvalidEnvelope) {
		t.Fatalf("expected invalid envelope projection error, got %v", err)
	}
	after := tableCounts(t, ctx, pool, "cloud_edge_event_receipts", "cloud_edge_event_raw_payloads", "inbox_events", "cloud_operational_events", "cloud_projection_event_type_stats")
	if !equalCounts(before, after) {
		t.Fatalf("malformed projection payload must roll back atomically, before=%v after=%v", before, after)
	}
}

func TestRepositoryProjectionRowsReplayExactlyOnce(t *testing.T) {
	tests := []struct {
		name                    string
		raw                     func(*testing.T) []byte
		eventType               contracts.EventType
		wantStats               int64
		wantPaymentCount        int64
		wantPaymentTotal        int64
		wantCheckRefundCount    int64
		wantCheckRefundTotal    int64
		wantShiftFinanceRows    int64
		wantFinancialOperations int64
		operationType           string
		shiftID                 string
	}{
		{
			name: "payment captured updates stats and shift finance",
			raw: func(t *testing.T) []byte {
				return samplePaymentCapturedEnvelope(t, "018f0000-0000-7000-8000-000000300001", "command-payment-proj-1", "payment-proj-1", 1500, "shift-payment-proj-1", "2026-05-05")
			},
			eventType:            contracts.EventPaymentCaptured,
			wantStats:            1,
			wantPaymentCount:     1,
			wantPaymentTotal:     1500,
			wantShiftFinanceRows: 1,
			shiftID:              "shift-payment-proj-1",
		},
		{
			name: "refund recorded updates stats shift finance and financial operation",
			raw: func(t *testing.T) []byte {
				return sampleFinancialOperationEnvelope(t, contracts.EventRefundRecorded, "018f0000-0000-7000-8000-000000300002", "command-refund-proj-1", "financial-operation-refund-proj-1", "refund", 1000, "shift-refund-proj-1", "2026-05-06")
			},
			eventType:               contracts.EventRefundRecorded,
			wantStats:               1,
			wantCheckRefundCount:    1,
			wantCheckRefundTotal:    1000,
			wantShiftFinanceRows:    1,
			wantFinancialOperations: 1,
			operationType:           "refund",
			shiftID:                 "shift-refund-proj-1",
		},
		{
			name: "cancellation recorded updates stats and financial operation only",
			raw: func(t *testing.T) []byte {
				return sampleFinancialOperationEnvelope(t, contracts.EventCancellationRecorded, "018f0000-0000-7000-8000-000000300003", "command-cancel-proj-1", "financial-operation-cancel-proj-1", "cancellation", 700, "shift-cancel-proj-1", "2026-05-07")
			},
			eventType:               contracts.EventCancellationRecorded,
			wantStats:               1,
			wantShiftFinanceRows:    0,
			wantFinancialOperations: 1,
			operationType:           "cancellation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			pool, repo, cleanup := openRepositoryWithBaseline(t, ctx)
			defer cleanup()
			service := app.NewService(repo, repositoryFixedClock{})
			raw := tt.raw(t)

			first, err := service.Receive(ctx, raw)
			if err != nil {
				t.Fatal(err)
			}
			second, err := service.Receive(ctx, raw)
			if err != nil {
				t.Fatal(err)
			}
			assertStableAck(t, first, second)
			assertEventTypeStats(t, ctx, pool, string(tt.eventType), tt.wantStats)
			assertShiftFinance(t, ctx, pool, tt.shiftID, tt.wantShiftFinanceRows, tt.wantPaymentCount, tt.wantPaymentTotal, tt.wantCheckRefundCount, tt.wantCheckRefundTotal)
			if got := countTableRows(t, ctx, pool, "cloud_projection_financial_operations"); got != tt.wantFinancialOperations {
				t.Fatalf("unexpected financial operation projection count: got %d want %d", got, tt.wantFinancialOperations)
			}
			if tt.wantFinancialOperations == 0 {
				return
			}
			items, err := repo.ListFinancialOperations(ctx, app.FinancialOperationProjectionFilter{
				RestaurantID:  "restaurant-1",
				OperationType: tt.operationType,
				Limit:         10,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 1 || items[0].OperationType != tt.operationType || items[0].ReceiptID != first.CloudReceiptID || items[0].RawPayloadSHA256Hex != first.RawPayloadSHA256Hex {
				t.Fatalf("unexpected financial operation read model: %+v", items)
			}
			marshaled, err := json.Marshal(items)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(marshaled), `"payload":`) || strings.Contains(string(marshaled), `"origin"`) {
				t.Fatalf("financial operation read model exposed raw event payload: %s", marshaled)
			}
		})
	}
}

func TestRepositoryRecordProblemEdgeEventStoresDiagnosticsAndSafeSummary(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openRepositoryWithBaseline(t, ctx)
	defer cleanup()
	raw := []byte(`{"node_token":"secret-node-token","diagnostic":"bad envelope"}`)

	if err := repo.RecordProblemEdgeEvent(ctx, app.ProblemEdgeEvent{
		Direction:        "edge_to_cloud",
		NodeDeviceID:     "node-1",
		RestaurantID:     "restaurant-1",
		ClientItemID:     "outbox-1",
		ErrorCode:        "INVALID_ENVELOPE",
		ErrorMessage:     "safe validation failure",
		RawPayload:       raw,
		RawPayloadSHA256: rawSHA256(raw),
		CreatedAt:        time.Date(2026, 5, 5, 10, 30, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	var stored struct {
		Direction string
		Node      string
		Rest      string
		Client    string
		Code      string
		Message   string
		Raw       string
		SHA       string
	}
	if err := pool.QueryRow(ctx, `
SELECT direction,COALESCE(node_device_id,''),COALESCE(restaurant_id,''),COALESCE(client_item_id,''),error_code,error_message,raw_payload,raw_payload_sha256_hex
FROM cloud_sync_problem_events`).Scan(&stored.Direction, &stored.Node, &stored.Rest, &stored.Client, &stored.Code, &stored.Message, &stored.Raw, &stored.SHA); err != nil {
		t.Fatal(err)
	}
	if stored.Direction != "edge_to_cloud" || stored.Node != "node-1" || stored.Rest != "restaurant-1" || stored.Client != "outbox-1" || stored.Code != "INVALID_ENVELOPE" || stored.SHA != rawSHA256(raw) {
		t.Fatalf("unexpected stored problem diagnostics: %+v", stored)
	}
	if stored.Raw != string(raw) {
		t.Fatalf("problem event raw payload was not preserved for diagnostics: %s", stored.Raw)
	}

	readiness, err := repo.GetStopListReadiness(ctx, app.StopListReadinessFilter{RestaurantID: "restaurant-1", NodeDeviceID: "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	if readiness.ProblemEvents.Total != 1 || len(readiness.ProblemEvents.ByErrorCode) != 1 || readiness.ProblemEvents.ByErrorCode[0].ErrorCode != "INVALID_ENVELOPE" {
		t.Fatalf("unexpected safe problem summary: %+v", readiness.ProblemEvents)
	}
	marshaled, err := json.Marshal(readiness)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(marshaled), "secret-node-token") || strings.Contains(string(marshaled), "raw_payload") {
		t.Fatalf("safe readiness view exposed problem raw payload: %s", marshaled)
	}
}

func TestRepositoryUpsertAndGetMasterDataPackage(t *testing.T) {
	tests := []struct {
		name         string
		initial      contracts.MasterDataPackage
		update       contracts.MasterDataPackage
		getNode      string
		wantNode     string
		wantVersion  int64
		wantPayload  string
		wantRevision string
	}{
		{
			name: "specific node update",
			initial: contracts.MasterDataPackage{
				StreamName:      contracts.MasterDataStreamCatalog,
				NodeDeviceID:    "node-1",
				RestaurantID:    "restaurant-1",
				SyncMode:        contracts.SyncModeIncremental,
				CloudVersion:    1,
				CheckpointToken: "catalog:1",
				PayloadJSON:     json.RawMessage(`{"catalog_items":[{"id":"item-old","type":"dish","name":"Tea","sku":"tea","base_unit":"pc","active":true}]}`),
			},
			update: contracts.MasterDataPackage{
				StreamName:      contracts.MasterDataStreamCatalog,
				NodeDeviceID:    "node-1",
				RestaurantID:    "restaurant-1",
				SyncMode:        contracts.SyncModeIncremental,
				CloudVersion:    2,
				CheckpointToken: "catalog:2",
				PayloadJSON:     json.RawMessage(`{"catalog_items":[{"id":"item-new","type":"dish","name":"Coffee","sku":"coffee","base_unit":"pc","active":true}]}`),
			},
			getNode:      "node-1",
			wantNode:     "node-1",
			wantVersion:  2,
			wantPayload:  `"item-new"`,
			wantRevision: "catalog:2",
		},
		{
			name: "generic package fallback",
			initial: contracts.MasterDataPackage{
				StreamName:      contracts.MasterDataStreamProposalFeedback,
				RestaurantID:    "restaurant-1",
				SyncMode:        contracts.SyncModeIncremental,
				CloudVersion:    3,
				CheckpointToken: "proposal_feedback:3",
				PayloadJSON:     json.RawMessage(`{"catalog_suggestions":[{"suggestion_id":"old","status":"pending"}]}`),
			},
			update: contracts.MasterDataPackage{
				StreamName:      contracts.MasterDataStreamProposalFeedback,
				RestaurantID:    "restaurant-1",
				SyncMode:        contracts.SyncModeIncremental,
				CloudVersion:    4,
				CheckpointToken: "proposal_feedback:4",
				PayloadJSON:     json.RawMessage(`{"catalog_suggestions":[{"suggestion_id":"new","status":"approved"}]}`),
			},
			getNode:      "node-2",
			wantVersion:  4,
			wantPayload:  `"new"`,
			wantRevision: "proposal_feedback:4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			_, repo, cleanup := openRepositoryWithBaseline(t, ctx)
			defer cleanup()
			now := time.Date(2026, 5, 5, 11, 0, 0, 0, time.UTC)
			tt.initial.CreatedAt = now
			tt.initial.UpdatedAt = now
			tt.update.CreatedAt = now.Add(time.Minute)
			tt.update.UpdatedAt = now.Add(time.Minute)
			cloudUpdatedAt := now.Add(2 * time.Minute)
			tt.update.CloudUpdatedAt = &cloudUpdatedAt

			if _, err := repo.UpsertMasterDataPackage(ctx, tt.initial); err != nil {
				t.Fatal(err)
			}
			stored, err := repo.UpsertMasterDataPackage(ctx, tt.update)
			if err != nil {
				t.Fatal(err)
			}
			got, err := repo.GetMasterDataPackage(ctx, tt.initial.StreamName, tt.getNode)
			if err != nil {
				t.Fatal(err)
			}
			if stored.CloudVersion != tt.wantVersion || got.CloudVersion != tt.wantVersion || got.NodeDeviceID != tt.wantNode || got.CheckpointToken != tt.wantRevision || !strings.Contains(string(got.PayloadJSON), tt.wantPayload) {
				t.Fatalf("unexpected package round-trip: stored=%+v got=%+v", stored, got)
			}
			if got.CloudUpdatedAt == nil || !got.CloudUpdatedAt.Equal(cloudUpdatedAt) {
				t.Fatalf("expected cloud_updated_at round-trip, got %+v", got.CloudUpdatedAt)
			}
		})
	}

	ctx := t.Context()
	_, repo, cleanup := openRepositoryWithBaseline(t, ctx)
	defer cleanup()
	_, err := repo.GetMasterDataPackage(ctx, contracts.MasterDataStreamMenu, "missing-node")
	if !errors.Is(err, contracts.ErrNotFound) {
		t.Fatalf("expected contract not found error, got %v", err)
	}
}

func TestRepositoryAuthenticateNodeToken(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openRepositoryWithBaseline(t, ctx)
	defer cleanup()
	insertRestaurant(t, ctx, pool, "restaurant-1")
	insertEdgeNode(t, ctx, pool, "node-1", "restaurant-1", "assigned", "stable-node-token")

	if err := repo.AuthenticateNodeToken(ctx, "node-1", "restaurant-1", "stable-node-token"); err != nil {
		t.Fatalf("expected valid node token, got %v", err)
	}

	tests := []struct {
		name         string
		nodeDeviceID string
		restaurantID string
		token        string
		want         error
	}{
		{name: "wrong token", nodeDeviceID: "node-1", restaurantID: "restaurant-1", token: "wrong-node-token", want: contracts.ErrSyncUnauthorized},
		{name: "wrong restaurant", nodeDeviceID: "node-1", restaurantID: "restaurant-other", token: "stable-node-token", want: contracts.ErrSyncForbidden},
		{name: "unknown node", nodeDeviceID: "node-missing", restaurantID: "restaurant-1", token: "stable-node-token", want: contracts.ErrSyncUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.AuthenticateNodeToken(ctx, tt.nodeDeviceID, tt.restaurantID, tt.token)
			if !errors.Is(err, tt.want) {
				t.Fatalf("unexpected auth error: got %v want %v", err, tt.want)
			}
			if err != nil && strings.Contains(err.Error(), tt.token) {
				t.Fatalf("auth error leaked token")
			}
		})
	}
}

func openRepositoryWithBaseline(t *testing.T, ctx context.Context) (*pgxpool.Pool, *Repository, func()) {
	t.Helper()
	pool, cleanup := openPostgresWithBaseline(t, ctx)
	return pool, NewRepository(pool), cleanup
}

func sampleOrderCreatedEnvelope(t *testing.T, eventID, commandID, orderID string) []byte {
	t.Helper()
	return marshalEnvelope(t, map[string]any{
		"version":           "1",
		"event_id":          eventID,
		"command_id":        commandID,
		"event_type":        string(contracts.EventOrderCreated),
		"aggregate_type":    "Order",
		"aggregate_id":      orderID,
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
				"id":            orderID,
				"edge_order_id": "edge-" + orderID,
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
	})
}

func samplePaymentCapturedEnvelope(t *testing.T, eventID, commandID, paymentID string, amount int64, shiftID, businessDate string) []byte {
	t.Helper()
	return marshalEnvelope(t, map[string]any{
		"version":           "1",
		"event_id":          eventID,
		"command_id":        commandID,
		"event_type":        string(contracts.EventPaymentCaptured),
		"aggregate_type":    "Payment",
		"aggregate_id":      paymentID,
		"restaurant_id":     "restaurant-1",
		"device_id":         "device-1",
		"node_device_id":    "device-1",
		"client_device_id":  "client-1",
		"actor_employee_id": "cashier-1",
		"session_id":        "session-1",
		"shift_id":          shiftID,
		"occurred_at":       businessDate + "T09:00:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"id":                  paymentID,
				"edge_payment_id":     "edge-" + paymentID,
				"restaurant_id":       "restaurant-1",
				"device_id":           "device-1",
				"shift_id":            shiftID,
				"precheck_id":         "precheck-1",
				"method":              "cash",
				"amount":              amount,
				"currency":            "RUB",
				"status":              "captured",
				"business_date_local": businessDate,
				"created_at":          businessDate + "T09:00:00Z",
				"updated_at":          businessDate + "T09:00:00Z",
			},
		},
	})
}

func sampleFinancialOperationEnvelope(t *testing.T, eventType contracts.EventType, eventID, commandID, operationID, operationType string, amount int64, shiftID, businessDate string) []byte {
	t.Helper()
	return marshalEnvelope(t, map[string]any{
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
	})
}

func malformedProjectionReceipt(t *testing.T) app.EdgeEventReceipt {
	t.Helper()
	raw := samplePaymentCapturedEnvelope(t, "018f0000-0000-7000-8000-000000200002", "command-malformed-projection-1", "payment-bad", 1200, "shift-bad", "2026-05-05")
	receipt, err := receiptFromRaw(raw, repositoryFixedClock{}.Now())
	if err != nil {
		t.Fatal(err)
	}
	receipt.Envelope.Payload = json.RawMessage(`{"origin":`)
	return receipt
}

func receiptFromRaw(raw []byte, receivedAt time.Time) (app.EdgeEventReceipt, error) {
	var envelope contracts.SyncEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return app.EdgeEventReceipt{}, err
	}
	key, err := contracts.IdempotencyKey(envelope)
	if err != nil {
		return app.EdgeEventReceipt{}, err
	}
	return app.EdgeEventReceipt{
		Envelope:         envelope,
		IdempotencyKey:   key,
		RawPayload:       append([]byte(nil), raw...),
		RawPayloadSHA256: rawSHA256(raw),
		CloudReceivedAt:  receivedAt,
	}, nil
}

func marshalEnvelope(t *testing.T, body map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func rawSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func assertStableAck(t *testing.T, first, second contracts.EventAck) {
	t.Helper()
	if first.Status != second.Status ||
		first.IdempotencyKey != second.IdempotencyKey ||
		first.CloudReceiptID != second.CloudReceiptID ||
		first.CommandID != second.CommandID ||
		first.EventID != second.EventID ||
		first.EdgeEventID != second.EdgeEventID ||
		first.EnvelopeVersion != second.EnvelopeVersion ||
		first.RawPayloadSHA256Hex != second.RawPayloadSHA256Hex ||
		!first.CloudReceivedAt.Equal(second.CloudReceivedAt) {
		t.Fatalf("expected stable ACK on replay\nfirst=%+v\nsecond=%+v", first, second)
	}
}

func countTableRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string) int64 {
	t.Helper()
	var got int64
	if err := pool.QueryRow(ctx, "SELECT COUNT(1) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return got
}

func tableCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tables ...string) map[string]int64 {
	t.Helper()
	out := make(map[string]int64, len(tables))
	for _, table := range tables {
		out[table] = countTableRows(t, ctx, pool, table)
	}
	return out
}

func equalCounts(a, b map[string]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		if b[key] != av {
			return false
		}
	}
	return true
}

func assertStoredJSONPayload(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, keyColumn, keyValue string, raw []byte, wantSHA string) {
	t.Helper()
	var same bool
	var gotSHA string
	query := "SELECT raw_payload = $1::jsonb, raw_payload_sha256_hex FROM " + table + " WHERE " + keyColumn + " = $2"
	if err := pool.QueryRow(ctx, query, string(raw), keyValue).Scan(&same, &gotSHA); err != nil {
		t.Fatalf("read %s payload: %v", table, err)
	}
	if !same || gotSHA != wantSHA {
		t.Fatalf("unexpected %s payload persistence: same=%v sha=%s want_sha=%s", table, same, gotSHA, wantSHA)
	}
}

func assertEventTypeStats(t *testing.T, ctx context.Context, pool *pgxpool.Pool, eventType string, wantCount int64) {
	t.Helper()
	var got int64
	if err := pool.QueryRow(ctx, `SELECT event_count FROM cloud_projection_event_type_stats WHERE restaurant_id = 'restaurant-1' AND device_id = 'device-1' AND event_type = $1`, eventType).Scan(&got); err != nil {
		t.Fatalf("read event type stats: %v", err)
	}
	if got != wantCount {
		t.Fatalf("unexpected event count for %s: got %d want %d", eventType, got, wantCount)
	}
}

func assertShiftFinance(t *testing.T, ctx context.Context, pool *pgxpool.Pool, shiftID string, wantRows, wantPaymentCount, wantPaymentTotal, wantCheckRefundCount, wantCheckRefundTotal int64) {
	t.Helper()
	var rows, paymentCount, paymentTotal, checkRefundCount, checkRefundTotal int64
	if err := pool.QueryRow(ctx, `
SELECT COUNT(1),
       COALESCE(SUM(payments_captured_count),0),
       COALESCE(SUM(payments_captured_total),0),
       COALESCE(SUM(checks_refunded_count),0),
       COALESCE(SUM(checks_refunded_total),0)
FROM cloud_projection_shift_finance
WHERE ($1 = '' OR shift_id = $1)`, shiftID).Scan(&rows, &paymentCount, &paymentTotal, &checkRefundCount, &checkRefundTotal); err != nil {
		t.Fatalf("read shift finance projection: %v", err)
	}
	if rows != wantRows || paymentCount != wantPaymentCount || paymentTotal != wantPaymentTotal || checkRefundCount != wantCheckRefundCount || checkRefundTotal != wantCheckRefundTotal {
		t.Fatalf("unexpected shift finance projection: rows=%d payments=%d/%d refunds=%d/%d", rows, paymentCount, paymentTotal, checkRefundCount, checkRefundTotal)
	}
}

func insertRestaurant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_restaurants(id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,status,created_at,updated_at)
VALUES ($1,'Test Restaurant','Europe/Moscow','RUB','standard','06:00','active',$2,$2)`, id, repositoryFixedClock{}.Now()); err != nil {
		t.Fatalf("insert restaurant: %v", err)
	}
}

func insertEdgeNode(t *testing.T, ctx context.Context, pool *pgxpool.Pool, nodeDeviceID, restaurantID, status, token string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_edge_nodes(id,restaurant_id,node_device_id,display_name,status,credentials_hash,assigned_at,created_at,updated_at)
VALUES ($1,$2,$3,'Node 1',$4,$5,$6,$6,$6)`,
		"edge-node-"+nodeDeviceID,
		restaurantID,
		nodeDeviceID,
		status,
		secretHash(token),
		repositoryFixedClock{}.Now(),
	); err != nil {
		t.Fatalf("insert edge node: %v", err)
	}
}
