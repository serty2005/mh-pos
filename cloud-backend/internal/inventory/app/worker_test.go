package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/inventory/app"
)

type fixedIDs struct {
	values []string
	next   int
}

func (f *fixedIDs) NewID() string {
	if f.next >= len(f.values) {
		panic("fixed id exhausted")
	}
	value := f.values[f.next]
	f.next++
	return value
}

type fakeRepo struct {
	events              []app.QueuedEvent
	documents           []app.StockDocument
	processed           []string
	failed              map[string]string
	recipes             map[string][]app.RecipeLine
	modifierOptionLinks map[string]string
}

func (f *fakeRepo) ClaimPending(context.Context, app.ClaimCommand) ([]app.QueuedEvent, error) {
	out := append([]app.QueuedEvent(nil), f.events...)
	f.events = nil
	return out, nil
}

func (f *fakeRepo) CreateStockDocument(_ context.Context, document app.StockDocument) error {
	f.documents = append(f.documents, document)
	return nil
}

func (f *fakeRepo) MarkProcessed(_ context.Context, queueID string, _ time.Time) error {
	f.processed = append(f.processed, queueID)
	return nil
}

func (f *fakeRepo) ListActiveRecipeLines(_ context.Context, _ string, catalogItemID string) ([]app.RecipeLine, error) {
	if f.recipes == nil {
		return nil, nil
	}
	return f.recipes[catalogItemID], nil
}

func (f *fakeRepo) ListModifierOptionLinks(_ context.Context, _ string, optionIDs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range optionIDs {
		if linked := f.modifierOptionLinks[id]; linked != "" {
			out[id] = linked
		}
	}
	return out, nil
}

func (f *fakeRepo) MarkFailed(_ context.Context, queueID, reason string, _ time.Time) error {
	if f.failed == nil {
		f.failed = map[string]string{}
	}
	f.failed[queueID] = reason
	return nil
}

func TestRunOnceCreatesSaleLedgerFromCheckClosed(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventCheckClosed, checkClosedPayload(t, []map[string]any{{
		"order_line_id":          "line-1",
		"catalog_item_id":        "item-1",
		"quantity":               "2.000",
		"unit_code":              "PC",
		"required_for_inventory": true,
	}}))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected one stock document, got %+v", repo.documents)
	}
	doc := repo.documents[0]
	if doc.Type != app.DocumentSale || len(doc.Ledger) != 1 || doc.Ledger[0].MovementType != app.MovementOut || doc.Ledger[0].CatalogItemID != "item-1" {
		t.Fatalf("unexpected sale document: %+v", doc)
	}
	if len(repo.processed) != 1 || repo.processed[0] != "queue-1" {
		t.Fatalf("expected queue row processed, got %+v", repo.processed)
	}
}

func TestRunOnceExpandsRecipeAndModifiersForCheckClosed(t *testing.T) {
	repo := &fakeRepo{
		events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventCheckClosed, checkClosedPayloadWithModifier(t))},
		recipes: map[string][]app.RecipeLine{
			"item-1": {{ComponentCatalogItemID: "ing-1", Quantity: "0.500", UnitCode: "KG"}},
		},
		modifierOptionLinks: map[string]string{"mod-opt-1": "mod-item-1"},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101", "018f0000-0000-7000-8000-00000000d102"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || len(repo.documents[0].Ledger) != 2 {
		t.Fatalf("unexpected docs %+v", repo.documents)
	}
}

func TestRunOnceFailsOnInvalidRecipeLine(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventCheckClosed, checkClosedPayload(t, []map[string]any{{"order_line_id": "line-1", "catalog_item_id": "item-1", "quantity": "1.000", "unit_code": "PC", "required_for_inventory": true}}))}, recipes: map[string][]app.RecipeLine{"item-1": {{ComponentCatalogItemID: "ing-1", Quantity: "0", UnitCode: "KG"}}}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repo.failed["queue-1"] == "" {
		t.Fatalf("expected failed queue row")
	}
}

func TestRunOnceCreatesPurchaseLedgerFromStockReceipt(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockReceiptCaptured, stockReceiptPayload(t))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected one stock document, got %+v", repo.documents)
	}
	doc := repo.documents[0]
	if doc.Type != app.DocumentPurchase || len(doc.Ledger) != 1 || doc.Ledger[0].MovementType != app.MovementIn {
		t.Fatalf("unexpected purchase document: %+v", doc)
	}
	if doc.Ledger[0].UnitCostMinor != 120 || doc.Ledger[0].TotalCostMinor != 360 {
		t.Fatalf("expected estimated costs from receipt item, got %+v", doc.Ledger[0])
	}
}

func TestRunOnceCreatesInventoryCountLedgerFromCountedQuantity(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventInventoryCountCaptured, inventoryCountPayload(t))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected one stock document, got %+v", repo.documents)
	}
	doc := repo.documents[0]
	if doc.Type != app.DocumentInventoryCount || len(doc.Ledger) != 1 || doc.Ledger[0].MovementType != app.MovementIn || doc.Ledger[0].Quantity != "7.500" {
		t.Fatalf("unexpected inventory count document: %+v", doc)
	}
}

func TestRunOnceCreatesProductionLedgerForSemiFinishedItem(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventProductionCompleted, productionPayload(t))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected one stock document, got %+v", repo.documents)
	}
	doc := repo.documents[0]
	if doc.Type != app.DocumentProduction || len(doc.Ledger) != 1 || doc.Ledger[0].MovementType != app.MovementIn || doc.Ledger[0].CatalogItemID != "semi-1" {
		t.Fatalf("unexpected production document: %+v", doc)
	}
}

func TestRunOnceMapsRefundReturnAndCancellationWaste(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationPayload(t, "refund", "return_to_stock")),
		sampleQueuedEvent(t, contracts.EventCancellationRecorded, financialOperationPayload(t, "cancellation", "write_off_waste")),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
		"018f0000-0000-7000-8000-00000000d002", "018f0000-0000-7000-8000-00000000d102",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 2 {
		t.Fatalf("expected two stock documents, got %+v", repo.documents)
	}
	if repo.documents[0].Type != app.DocumentReturn || repo.documents[0].Ledger[0].MovementType != app.MovementIn {
		t.Fatalf("unexpected refund return document: %+v", repo.documents[0])
	}
	if repo.documents[1].Type != app.DocumentWaste || repo.documents[1].Ledger[0].MovementType != app.MovementOut {
		t.Fatalf("unexpected cancellation waste document: %+v", repo.documents[1])
	}
}

func TestRunOnceNoStockEffectIsProcessedWithoutDocument(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationPayload(t, "refund", "no_stock_effect"))}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 0 || len(repo.processed) != 1 {
		t.Fatalf("expected no document and processed queue, documents=%+v processed=%+v", repo.documents, repo.processed)
	}
}

func TestRunOnceMarksWholeCheckStockEffectWithoutItemsFailed(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationPayloadWithoutItems(t, "refund", "return_to_stock"))}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 0 {
		t.Fatalf("expected no document for whole-check stock effect without items, got %+v", repo.documents)
	}
	if reason := repo.failed["queue-1"]; reason == "" {
		t.Fatalf("expected failed queue row, got %+v", repo.failed)
	}
}

func TestRunOnceReturnsRepositoryError(t *testing.T) {
	want := errors.New("database down")
	repo := &failingRepo{err: want}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); !errors.Is(err, want) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

type failingRepo struct{ err error }

func (f *failingRepo) ClaimPending(context.Context, app.ClaimCommand) ([]app.QueuedEvent, error) {
	return nil, f.err
}
func (f *failingRepo) CreateStockDocument(context.Context, app.StockDocument) error {
	return f.err
}
func (f *failingRepo) MarkProcessed(context.Context, string, time.Time) error      { return f.err }
func (f *failingRepo) MarkFailed(context.Context, string, string, time.Time) error { return f.err }
func (f *failingRepo) ListActiveRecipeLines(context.Context, string, string) ([]app.RecipeLine, error) {
	return nil, f.err
}
func (f *failingRepo) ListModifierOptionLinks(context.Context, string, []string) (map[string]string, error) {
	return nil, f.err
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

func sampleQueuedEvent(t *testing.T, eventType contracts.EventType, payload json.RawMessage) app.QueuedEvent {
	t.Helper()
	return app.QueuedEvent{
		ID:           "queue-1",
		ReceiptID:    "receipt-1",
		RestaurantID: "restaurant-1",
		DeviceID:     "device-1",
		EventID:      "018f0000-0000-7000-8000-0000000000a1",
		EventType:    eventType,
		OccurredAt:   time.Date(2026, 5, 5, 9, 0, 0, 0, time.UTC),
		Payload:      payload,
	}
}

func checkClosedPayload(t *testing.T, items []map[string]any) json.RawMessage {
	t.Helper()
	return marshalPayload(t, map[string]any{
		"check_id":            "check-1",
		"order_id":            "order-1",
		"precheck_id":         "precheck-1",
		"restaurant_id":       "restaurant-1",
		"business_date_local": "2026-05-05",
		"closed_at":           "2026-05-05T09:00:00Z",
		"items":               items,
	})
}

func stockReceiptPayload(t *testing.T) json.RawMessage {
	t.Helper()
	return marshalPayload(t, map[string]any{
		"receipt_id":          "stock-receipt-1",
		"restaurant_id":       "restaurant-1",
		"supplier_id":         "supplier-1",
		"business_date_local": "2026-05-05",
		"received_at":         "2026-05-05T09:00:00Z",
		"items": []map[string]any{{
			"catalog_item_id":  "item-1",
			"quantity":         "3.000",
			"unit_code":        "KG",
			"unit_cost_minor":  120,
			"total_cost_minor": 360,
		}},
	})
}

func inventoryCountPayload(t *testing.T) json.RawMessage {
	t.Helper()
	return marshalPayload(t, map[string]any{
		"count_id":            "inventory-count-1",
		"restaurant_id":       "restaurant-1",
		"business_date_local": "2026-05-05",
		"counted_at":          "2026-05-05T09:00:00Z",
		"items": []map[string]any{{
			"catalog_item_id":  "item-1",
			"quantity":         "0.000",
			"counted_quantity": "7.500",
			"unit_code":        "KG",
		}},
	})
}

func productionPayload(t *testing.T) json.RawMessage {
	t.Helper()
	return marshalPayload(t, map[string]any{
		"production_id":                 "production-1",
		"restaurant_id":                 "restaurant-1",
		"business_date_local":           "2026-05-05",
		"completed_at":                  "2026-05-05T09:00:00Z",
		"semi_finished_catalog_item_id": "semi-1",
		"quantity":                      "4.000",
		"unit_code":                     "KG",
	})
}

func financialOperationPayload(t *testing.T, operationType, disposition string) json.RawMessage {
	t.Helper()
	data := financialOperationData(operationType, disposition)
	data["items"] = []map[string]any{{
		"order_line_id":         "line-1",
		"catalog_item_id":       "item-1",
		"quantity":              "1.000",
		"unit_code":             "PC",
		"inventory_disposition": disposition,
	}}
	return marshalPayload(t, data)
}

func financialOperationPayloadWithoutItems(t *testing.T, operationType, disposition string) json.RawMessage {
	t.Helper()
	return marshalPayload(t, financialOperationData(operationType, disposition))
}

func financialOperationData(operationType, disposition string) map[string]any {
	return map[string]any{
		"id":                    "financial-operation-1",
		"edge_operation_id":     "edge-financial-operation-1",
		"restaurant_id":         "restaurant-1",
		"device_id":             "device-1",
		"shift_id":              "shift-1",
		"original_shift_id":     "shift-sale-1",
		"check_id":              "check-1",
		"precheck_id":           "precheck-1",
		"operation_type":        operationType,
		"operation_kind":        "partial",
		"status":                "recorded",
		"amount":                1000,
		"currency":              "RUB",
		"business_date_local":   "2026-05-05",
		"inventory_disposition": disposition,
		"reason":                "guest return",
		"snapshot":              map[string]any{"document_type": "financial_operation", "check_id": "check-1"},
		"created_at":            "2026-05-05T09:00:00Z",
	}
}

func marshalPayload(t *testing.T, data map[string]any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"origin": "edge_device", "data": data})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func checkClosedPayloadWithModifier(t *testing.T) json.RawMessage {
	t.Helper()
	return marshalPayload(t, map[string]any{
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
			"modifiers": []map[string]any{{
				"modifier_option_id": "mod-opt-1",
				"quantity":           "1.000",
				"unit_code":          "PC",
			}},
		}},
	})
}
