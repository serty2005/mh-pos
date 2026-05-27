package contracts_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"cloud-backend/internal/cloudsync/contracts"
)

func TestIdempotencyKeyUsesRestaurantDeviceAndEdgeEventID(t *testing.T) {
	envelope := validEnvelope(t, "PaymentCaptured")
	key, err := contracts.IdempotencyKey(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if key != "restaurant-1:device-1:018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("unexpected key %q", key)
	}
	if contracts.EdgeEventID(envelope) != envelope.EventID {
		t.Fatalf("edge_event_id must be the envelope event_id")
	}
}

func TestValidateEnvelopeRejectsUnknownEvent(t *testing.T) {
	envelope := validEnvelope(t, "PaymentCaptured")
	envelope.EventType = "RefundCaptured"
	err := contracts.ValidateEnvelope(envelope)
	if !errors.Is(err, contracts.ErrInvalidEnvelope) {
		t.Fatalf("expected invalid envelope, got %v", err)
	}
}

func TestValidateEnvelopeAcceptsRefundEvents(t *testing.T) {
	for _, eventType := range []contracts.EventType{contracts.EventPaymentRefunded, contracts.EventCheckRefunded} {
		envelope := validEnvelope(t, eventType)
		if eventType == contracts.EventCheckRefunded {
			envelope.AggregateType = "Check"
			envelope.AggregateID = "check-1"
			envelope.Payload = json.RawMessage(`{
				"origin":"edge_device",
				"data":{
					"id":"check-1",
					"order_id":"order-1",
					"status":"refunded",
					"subtotal":1000,
					"discount_total":0,
					"tax_total":0,
					"total":1000,
					"paid_total":0,
					"business_date_local":"2026-05-05",
					"closed_at":"2026-05-05T09:00:00Z",
					"created_at":"2026-05-05T09:00:00Z",
					"updated_at":"2026-05-05T09:00:00Z"
				}
			}`)
		}
		if err := contracts.ValidateEnvelope(envelope); err != nil {
			t.Fatalf("expected %s to be accepted, got %v", eventType, err)
		}
	}
}

func TestValidateEnvelopeAcceptsCurrentFinancialOperationEvents(t *testing.T) {
	tests := []struct {
		name          string
		eventType     contracts.EventType
		operationType string
	}{
		{name: "cancellation", eventType: contracts.EventCancellationRecorded, operationType: "cancellation"},
		{name: "refund", eventType: contracts.EventRefundRecorded, operationType: "refund"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := validFinancialOperationEnvelope(t, tt.eventType, tt.operationType)
			if err := contracts.ValidateEnvelope(envelope); err != nil {
				t.Fatalf("expected %s payload to be accepted, got %v", tt.eventType, err)
			}
		})
	}
}

func TestValidateEnvelopeAcceptsTargetInventoryEvents(t *testing.T) {
	tests := []struct {
		name      string
		eventType contracts.EventType
		payload   json.RawMessage
	}{
		{
			name:      "check closed",
			eventType: contracts.EventCheckClosed,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"check_id":"check-1","order_id":"order-1","precheck_id":"precheck-1","restaurant_id":"restaurant-1","business_date_local":"2026-05-05","closed_at":"2026-05-05T09:00:00Z","items":[{"order_line_id":"line-1","catalog_item_id":"item-1","quantity":"2.000","unit_code":"PC","required_for_inventory":true}]}}`),
		},
		{
			name:      "item served",
			eventType: contracts.EventItemServed,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"served_event_id":"served-1","ticket_id":"ticket-1","serve_sequence":1,"order_id":"order-1","order_line_id":"line-1","catalog_item_id":"item-1","quantity":"1.000","unit_code":"PC","served_at":"2026-05-05T09:00:00Z","station_id":"kitchen-hot"}}`),
		},
		{
			name:      "kitchen status changed",
			eventType: contracts.EventKitchenTicketStatusChanged,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"ticket_id":"ticket-1","order_id":"order-1","order_line_id":"line-1","from_status":"new","to_status":"accepted","changed_at":"2026-05-05T09:00:00Z","station_id":"kitchen-hot"}}`),
		},
		{
			name:      "stock receipt",
			eventType: contracts.EventStockReceiptCaptured,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"receipt_id":"receipt-1","restaurant_id":"restaurant-1","received_at":"2026-05-05T08:00:00Z","business_date_local":"2026-05-05","supplier_id":"supplier-1","items":[{"catalog_item_id":"item-1","quantity":"10.000","unit_code":"KG","unit_cost_minor":5000,"currency":"RUB"}]}}`),
		},
		{
			name:      "inventory count",
			eventType: contracts.EventInventoryCountCaptured,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"count_id":"count-1","restaurant_id":"restaurant-1","counted_at":"2026-05-05T21:00:00Z","business_date_local":"2026-05-05","items":[{"catalog_item_id":"item-1","counted_quantity":"3.250","unit_code":"KG"}]}}`),
		},
		{
			name:      "production completed",
			eventType: contracts.EventProductionCompleted,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"production_id":"production-1","restaurant_id":"restaurant-1","semi_finished_catalog_item_id":"semi-1","quantity":"5.000","unit_code":"KG","completed_at":"2026-05-05T10:15:00Z","business_date_local":"2026-05-05"}}`),
		},
		{
			name:      "stock write-off",
			eventType: contracts.EventStockWriteOffCaptured,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"write_off_id":"writeoff-1","restaurant_id":"restaurant-1","reason_code":"expired","written_off_at":"2026-05-05T11:00:00Z","business_date_local":"2026-05-05","items":[{"catalog_item_id":"item-1","quantity":"1.000","unit_code":"KG"}]}}`),
		},
		{
			name:      "stop list updated",
			eventType: contracts.EventStopListUpdated,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"stop_list_id":"stop-1","restaurant_id":"restaurant-1","catalog_item_id":"item-1","available_quantity":"0.000","active":true,"source":"edge","reason":"ingredient_unavailable","updated_at":"2026-05-05T12:05:00Z"}}`),
		},
		{
			name:      "catalog suggestion",
			eventType: contracts.EventCatalogItemChangeSuggested,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"suggestion_id":"catalog-suggest-1","restaurant_id":"restaurant-1","action":"create_item","reason":"new seasonal item","suggested_at":"2026-05-05T12:30:00Z"}}`),
		},
		{
			name:      "recipe suggestion",
			eventType: contracts.EventRecipeChangeSuggested,
			payload:   json.RawMessage(`{"origin":"edge_device","data":{"suggestion_id":"recipe-suggest-1","restaurant_id":"restaurant-1","action":"update_recipe","reason":"faster prep","suggested_at":"2026-05-05T12:35:00Z"}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := validInventoryEnvelope(t, tt.eventType, tt.payload)
			if err := contracts.ValidateEnvelope(envelope); err != nil {
				t.Fatalf("expected %s to be accepted, got %v", tt.eventType, err)
			}
		})
	}
}

func TestKitchenStatusChangedIsOperationalOnlyAndItemServedIsInventoryRelevant(t *testing.T) {
	if contracts.IsInventoryRelevantEventType(contracts.EventKitchenTicketStatusChanged) {
		t.Fatal("KitchenTicketStatusChanged must stay operational-only and not enter inventory_event_queue")
	}
	if !contracts.IsInventoryRelevantEventType(contracts.EventItemServed) {
		t.Fatal("ItemServed must enter durable inventory_event_queue")
	}
	if contracts.IsInventoryRelevantEventType(contracts.EventCatalogItemChangeSuggested) {
		t.Fatal("CatalogItemChangeSuggested must stay outside inventory_event_queue")
	}
	if contracts.IsInventoryRelevantEventType(contracts.EventRecipeChangeSuggested) {
		t.Fatal("RecipeChangeSuggested must stay outside inventory_event_queue")
	}
}

func TestValidateEnvelopeRejectsInvalidTargetInventoryPayload(t *testing.T) {
	envelope := validInventoryEnvelope(t, contracts.EventCheckClosed, json.RawMessage(`{"origin":"edge_device","data":{"check_id":"check-1","business_date_local":"2026-05-05","closed_at":"2026-05-05T09:00:00Z","items":[{"catalog_item_id":"item-1","quantity":"0.000","unit_code":"PC","required_for_inventory":true}]}}`))
	err := contracts.ValidateEnvelope(envelope)
	if !errors.Is(err, contracts.ErrInvalidEnvelope) {
		t.Fatalf("expected invalid inventory payload, got %v", err)
	}
}

func TestValidateEnvelopeRejectsInvalidFinancialOperationPayload(t *testing.T) {
	envelope := validFinancialOperationEnvelope(t, contracts.EventRefundRecorded, "refund")
	envelope.Payload = json.RawMessage(`{
		"origin":"edge_device",
		"data":{
			"id":"financial-operation-1",
			"edge_operation_id":"edge-financial-operation-1",
			"restaurant_id":"restaurant-1",
			"device_id":"device-1",
			"shift_id":"shift-refund-1",
			"original_shift_id":"shift-sale-1",
			"check_id":"check-1",
			"precheck_id":"precheck-1",
			"operation_type":"cancellation",
			"operation_kind":"full",
			"status":"recorded",
			"amount":1000,
			"currency":"RUB",
			"business_date_local":"2026-05-05",
			"inventory_disposition":"no_stock_effect",
			"reason":"guest return",
			"created_at":"2026-05-05T09:00:00Z"
		}
	}`)
	err := contracts.ValidateEnvelope(envelope)
	if !errors.Is(err, contracts.ErrInvalidEnvelope) {
		t.Fatalf("expected invalid financial operation payload, got %v", err)
	}
}

func TestValidateEnvelopeRejectsFinancialOperationPayloadIdentityMismatch(t *testing.T) {
	tests := []struct {
		name       string
		field      string
		value      any
		envelopeFn func(*contracts.SyncEnvelope)
	}{
		{name: "missing precheck", field: "precheck_id", value: ""},
		{name: "missing reason", field: "reason", value: ""},
		{name: "restaurant mismatch", field: "restaurant_id", value: "restaurant-other"},
		{name: "device mismatch", field: "device_id", value: "device-other"},
		{name: "envelope restaurant mismatch", envelopeFn: func(envelope *contracts.SyncEnvelope) {
			other := "restaurant-other"
			envelope.RestaurantID = &other
		}},
		{name: "envelope device mismatch", envelopeFn: func(envelope *contracts.SyncEnvelope) {
			envelope.DeviceID = "device-other"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := validFinancialOperationEnvelope(t, contracts.EventRefundRecorded, "refund")
			if tt.field != "" {
				setFinancialOperationPayloadField(t, &envelope, tt.field, tt.value)
			}
			if tt.envelopeFn != nil {
				tt.envelopeFn(&envelope)
			}
			err := contracts.ValidateEnvelope(envelope)
			if !errors.Is(err, contracts.ErrInvalidEnvelope) {
				t.Fatalf("expected invalid financial operation payload, got %v", err)
			}
		})
	}
}

func validEnvelope(t *testing.T, eventType contracts.EventType) contracts.SyncEnvelope {
	t.Helper()
	raw := []byte(`{
	  "version":"1",
	  "event_id":"018f0000-0000-7000-8000-000000000001",
	  "command_id":"command-1",
	  "event_type":"PaymentCaptured",
	  "aggregate_type":"Payment",
	  "aggregate_id":"payment-1",
	  "restaurant_id":"restaurant-1",
	  "device_id":"device-1",
	  "node_device_id":"device-1",
	  "client_device_id":"client-1",
	  "actor_employee_id":"employee-1",
	  "session_id":"session-1",
	  "shift_id":"shift-1",
	  "occurred_at":"2026-05-05T09:00:00Z",
	  "payload":{
	    "origin":"edge_device",
	    "data":{
	      "id":"payment-1",
	      "precheck_id":"precheck-1",
	      "method":"cash",
	      "amount":1000,
	      "currency":"RUB",
	      "status":"captured",
	      "created_at":"2026-05-05T09:00:00Z",
	      "updated_at":"2026-05-05T09:00:00Z"
	    }
	  }
	}`)
	var envelope contracts.SyncEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.EventType = eventType
	return envelope
}

func validInventoryEnvelope(t *testing.T, eventType contracts.EventType, payload json.RawMessage) contracts.SyncEnvelope {
	t.Helper()
	restaurantID := "restaurant-1"
	shiftID := "shift-1"
	return contracts.SyncEnvelope{
		Version:       "1",
		EventID:       "018f0000-0000-7000-8000-0000000000a1",
		CommandID:     "command-inventory-1",
		EventType:     eventType,
		AggregateType: "InventoryEvent",
		AggregateID:   "inventory-event-1",
		RestaurantID:  &restaurantID,
		DeviceID:      "device-1",
		NodeDeviceID:  "device-1",
		ShiftID:       &shiftID,
		OccurredAt:    mustParseTime(t, "2026-05-05T09:00:00Z"),
		Payload:       payload,
	}
}

func setFinancialOperationPayloadField(t *testing.T, envelope *contracts.SyncEnvelope, field string, value any) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatal("expected financial operation payload data object")
	}
	data[field] = value
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Payload = raw
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func validFinancialOperationEnvelope(t *testing.T, eventType contracts.EventType, operationType string) contracts.SyncEnvelope {
	t.Helper()
	body := map[string]any{
		"version":           "1",
		"event_id":          "018f0000-0000-7000-8000-0000000000f1",
		"command_id":        "command-financial-operation-1",
		"event_type":        string(eventType),
		"aggregate_type":    "FinancialOperation",
		"aggregate_id":      "financial-operation-1",
		"restaurant_id":     "restaurant-1",
		"device_id":         "device-1",
		"node_device_id":    "device-1",
		"client_device_id":  "client-1",
		"actor_employee_id": "manager-1",
		"session_id":        "session-1",
		"shift_id":          "shift-refund-1",
		"occurred_at":       "2026-05-05T09:00:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"id":                    "financial-operation-1",
				"edge_operation_id":     "edge-financial-operation-1",
				"restaurant_id":         "restaurant-1",
				"device_id":             "device-1",
				"shift_id":              "shift-refund-1",
				"original_shift_id":     "shift-sale-1",
				"check_id":              "check-1",
				"precheck_id":           "precheck-1",
				"operation_type":        operationType,
				"operation_kind":        "full",
				"status":                "recorded",
				"amount":                1000,
				"currency":              "RUB",
				"business_date_local":   "2026-05-05",
				"inventory_disposition": "no_stock_effect",
				"reason":                "guest return",
				"snapshot":              map[string]any{"document_type": "financial_operation", "check_id": "check-1"},
				"created_at":            "2026-05-05T09:00:00Z",
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope contracts.SyncEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}
