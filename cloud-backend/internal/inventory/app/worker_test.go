package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	stopListUpdates     []app.StopListProjectionCommand
	processed           []string
	failed              map[string]string
	recipes             map[string][]app.RecipeLine
	modifierOptionLinks map[string]string
	supersededServedIDs map[string]bool
}

func (f *fakeRepo) ClaimPending(context.Context, app.ClaimCommand) ([]app.QueuedEvent, error) {
	out := append([]app.QueuedEvent(nil), f.events...)
	f.events = nil
	return out, nil
}

func (f *fakeRepo) CreateStockDocument(_ context.Context, document app.StockDocument) error {
	for _, existing := range f.documents {
		if existing.SourceEventID == document.SourceEventID && existing.SourceEventType == document.SourceEventType {
			return nil
		}
	}
	f.documents = append(f.documents, document)
	return nil
}

func (f *fakeRepo) ApplyStopListUpdate(_ context.Context, cmd app.StopListProjectionCommand) error {
	for _, existing := range f.stopListUpdates {
		if existing.SourceEventID == cmd.SourceEventID {
			return nil
		}
	}
	f.stopListUpdates = append(f.stopListUpdates, cmd)
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

func (f *fakeRepo) ListServedOrderLineQuantities(_ context.Context, _ string, orderLineIDs []string) (map[string]string, error) {
	wanted := map[string]bool{}
	for _, id := range orderLineIDs {
		wanted[strings.TrimSpace(id)] = true
	}
	totals := map[string]float64{}
	for _, document := range f.documents {
		for _, entry := range document.Ledger {
			orderLineID := strings.TrimSpace(entry.OrderLineID)
			if !wanted[orderLineID] {
				continue
			}
			qty, _ := strconv.ParseFloat(strings.TrimSpace(entry.Quantity), 64)
			switch {
			case entry.SourceEventType == string(contracts.EventItemServed) && entry.MovementType == app.MovementOut:
				totals[orderLineID] += qty
			case entry.SourceEventType == app.SourceEventItemServedCompensation && entry.MovementType == app.MovementIn:
				totals[orderLineID] -= qty
			}
		}
	}
	out := map[string]string{}
	for id, qty := range totals {
		out[id] = fmt.Sprintf("%.3f", qty)
	}
	return out, nil
}

func (f *fakeRepo) HasSupersedingServedEvent(_ context.Context, _ string, _ string, servedEventID string) (bool, error) {
	return f.supersededServedIDs[strings.TrimSpace(servedEventID)], nil
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
	got := ledgerQuantityByCatalogItem(repo.documents[0])
	if got["ing-1"] != "1.000" {
		t.Fatalf("expected recipe component quantity 1.000, got %+v", got)
	}
	if got["mod-item-1"] != "2.000" {
		t.Fatalf("expected linked modifier consumption to scale by sold line quantity, got %+v", got)
	}
}

func TestRunOnceModifierWithoutLinkCreatesNoAdditionalLedgerRow(t *testing.T) {
	repo := &fakeRepo{
		events:              []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventCheckClosed, checkClosedPayloadWithModifier(t))},
		modifierOptionLinks: map[string]string{},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || len(repo.documents[0].Ledger) != 1 {
		t.Fatalf("expected only base sale row when modifier has no link, got %+v", repo.documents)
	}
	if repo.documents[0].Ledger[0].CatalogItemID != "item-1" {
		t.Fatalf("expected fallback base item ledger row, got %+v", repo.documents[0].Ledger[0])
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

func TestRunOnceCreatesSaleLedgerFromItemServedForOrderLine(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventItemServed, itemServedPayload(t, "line-1", "item-1", "1.000"))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || len(repo.documents[0].Ledger) != 1 {
		t.Fatalf("expected one item served document, got %+v", repo.documents)
	}
	entry := repo.documents[0].Ledger[0]
	if repo.documents[0].Type != app.DocumentSale || entry.MovementType != app.MovementOut || entry.OrderLineID != "line-1" || entry.Quantity != "1.000" {
		t.Fatalf("unexpected item served ledger entry: %+v", entry)
	}
}

func TestRunOnceSkipsDuplicateItemServedReplay(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayload(t, "line-1", "item-1", "1.000")),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayload(t, "line-1", "item-1", "1.000")),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
		"018f0000-0000-7000-8000-00000000d002", "018f0000-0000-7000-8000-00000000d102",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected replay to keep one document, got %+v", repo.documents)
	}
	if len(repo.processed) != 2 {
		t.Fatalf("expected both queue rows processed, got %+v", repo.processed)
	}
}

func TestRunOnceSkipsSupersededItemServedWhenRecallServeAgainArrived(t *testing.T) {
	repo := &fakeRepo{
		events: []app.QueuedEvent{
			queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayloadWithServedEventID(t, "served-event-1", "", 1, "line-1", "item-1", "1.000")),
			queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a2", contracts.EventItemServed, itemServedPayloadWithServedEventID(t, "served-event-2", "served-event-1", 2, "line-1", "item-1", "1.000")),
		},
		supersededServedIDs: map[string]bool{"served-event-1": true},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected only latest effective ItemServed document, got %+v", repo.documents)
	}
	if repo.documents[0].SourceEventID != "018f0000-0000-7000-8000-0000000000a2" {
		t.Fatalf("expected latest served event to own stock document, got %+v", repo.documents[0])
	}
	if len(repo.processed) != 2 {
		t.Fatalf("expected both queue rows processed, got %+v", repo.processed)
	}
}

func TestRunOnceCompensatesAlreadyProcessedItemServedOnRecallServeAgain(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayloadWithServedEventID(t, "served-event-1", "", 1, "line-1", "item-1", "1.000")),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a2", contracts.EventItemServed, itemServedPayloadWithServedEventID(t, "served-event-2", "served-event-1", 2, "line-1", "item-1", "1.000")),
		queuedEvent(t, "queue-3", "018f0000-0000-7000-8000-0000000000a3", contracts.EventCheckClosed, checkClosedPayload(t, []map[string]any{{
			"order_line_id":          "line-1",
			"catalog_item_id":        "item-1",
			"quantity":               "2.000",
			"unit_code":              "PC",
			"required_for_inventory": true,
		}})),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
		"018f0000-0000-7000-8000-00000000d002", "018f0000-0000-7000-8000-00000000d102",
		"018f0000-0000-7000-8000-00000000d003", "018f0000-0000-7000-8000-00000000d103",
		"018f0000-0000-7000-8000-00000000d004", "018f0000-0000-7000-8000-00000000d104",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 4 {
		t.Fatalf("expected old serve, compensation, new serve and CheckClosed delta documents, got %+v", repo.documents)
	}
	if repo.documents[1].Type != app.DocumentReturn || repo.documents[1].SourceEventType != app.SourceEventItemServedCompensation {
		t.Fatalf("expected append-only compensation document, got %+v", repo.documents[1])
	}
	compensation := repo.documents[1].Ledger[0]
	if compensation.MovementType != app.MovementIn || compensation.Quantity != "1.000" || compensation.OrderLineID != "line-1" {
		t.Fatalf("unexpected compensation ledger entry: %+v", compensation)
	}
	newServe := repo.documents[2].Ledger[0]
	if repo.documents[2].SourceEventID != "018f0000-0000-7000-8000-0000000000a2" || newServe.MovementType != app.MovementOut || newServe.Quantity != "1.000" {
		t.Fatalf("unexpected replacement serve document: %+v", repo.documents[2])
	}
	checkClosedDelta := repo.documents[3].Ledger[0]
	if checkClosedDelta.MovementType != app.MovementOut || checkClosedDelta.Quantity != "1.000" {
		t.Fatalf("expected CheckClosed to consume only unserved delta after compensation, got %+v", checkClosedDelta)
	}
}

func TestRunOnceRecallServeAgainKeepsSingleConsumptionForSameQuantity(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayloadWithServedEventID(t, "served-event-1", "", 1, "line-1", "item-1", "1.000")),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a2", contracts.EventItemServed, itemServedPayloadWithServedEventID(t, "served-event-2", "served-event-1", 2, "line-1", "item-1", "1.000")),
		queuedEvent(t, "queue-3", "018f0000-0000-7000-8000-0000000000a3", contracts.EventCheckClosed, checkClosedPayload(t, []map[string]any{{
			"order_line_id":          "line-1",
			"catalog_item_id":        "item-1",
			"quantity":               "1.000",
			"unit_code":              "PC",
			"required_for_inventory": true,
		}})),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
		"018f0000-0000-7000-8000-00000000d002", "018f0000-0000-7000-8000-00000000d102",
		"018f0000-0000-7000-8000-00000000d003", "018f0000-0000-7000-8000-00000000d103",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 3 {
		t.Fatalf("expected old serve, compensation and replacement serve only, got %+v", repo.documents)
	}
	var consumed float64
	for _, document := range repo.documents {
		if document.SourceEventType == string(contracts.EventCheckClosed) {
			t.Fatalf("expected CheckClosed to skip fully served line after recall compensation, got %+v", document)
		}
		for _, entry := range document.Ledger {
			if entry.OrderLineID != "line-1" {
				continue
			}
			qty, err := strconv.ParseFloat(entry.Quantity, 64)
			if err != nil {
				t.Fatalf("parse ledger quantity %q: %v", entry.Quantity, err)
			}
			switch entry.MovementType {
			case app.MovementOut:
				consumed += qty
			case app.MovementIn:
				consumed -= qty
			}
		}
	}
	if fmt.Sprintf("%.3f", consumed) != "1.000" {
		t.Fatalf("expected net single consumption after recall/serve-again, got %.3f from %+v", consumed, repo.documents)
	}
}

func TestRunOnceCompensatingItemServedReplayIsIdempotent(t *testing.T) {
	supersedingPayload := itemServedPayloadWithServedEventID(t, "served-event-2", "served-event-1", 2, "line-1", "item-1", "1.000")
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayloadWithServedEventID(t, "served-event-1", "", 1, "line-1", "item-1", "1.000")),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a2", contracts.EventItemServed, supersedingPayload),
		queuedEvent(t, "queue-3", "018f0000-0000-7000-8000-0000000000a2", contracts.EventItemServed, supersedingPayload),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
		"018f0000-0000-7000-8000-00000000d002", "018f0000-0000-7000-8000-00000000d102",
		"018f0000-0000-7000-8000-00000000d003", "018f0000-0000-7000-8000-00000000d103",
		"018f0000-0000-7000-8000-00000000d004", "018f0000-0000-7000-8000-00000000d104",
		"018f0000-0000-7000-8000-00000000d005", "018f0000-0000-7000-8000-00000000d105",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 3 {
		t.Fatalf("expected replay to keep old serve, one compensation and one replacement serve, got %+v", repo.documents)
	}
	countBySourceType := map[string]int{}
	for _, document := range repo.documents {
		countBySourceType[document.SourceEventType]++
	}
	if countBySourceType[app.SourceEventItemServedCompensation] != 1 || countBySourceType[string(contracts.EventItemServed)] != 2 {
		t.Fatalf("unexpected document source types after replay: %+v", countBySourceType)
	}
	if len(repo.processed) != 3 {
		t.Fatalf("expected all queue rows processed, got %+v", repo.processed)
	}
}

func TestRunOnceCheckClosedAfterServedDoesNotDoubleConsumeLine(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayload(t, "line-1", "item-1", "1.000")),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a2", contracts.EventCheckClosed, checkClosedPayload(t, []map[string]any{{
			"order_line_id":          "line-1",
			"catalog_item_id":        "item-1",
			"quantity":               "1.000",
			"unit_code":              "PC",
			"required_for_inventory": true,
		}})),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected CheckClosed to create no duplicate document, got %+v", repo.documents)
	}
	if repo.documents[0].SourceEventType != string(contracts.EventItemServed) {
		t.Fatalf("expected only ItemServed document, got %+v", repo.documents[0])
	}
}

func TestRunOnceCheckClosedAfterServedDoesNotDoubleConsumeLinkedModifier(t *testing.T) {
	repo := &fakeRepo{
		events: []app.QueuedEvent{
			queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayload(t, "line-1", "item-1", "2.000")),
			queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a2", contracts.EventCheckClosed, checkClosedPayloadWithModifier(t)),
		},
		modifierOptionLinks: map[string]string{"mod-opt-1": "mod-item-1"},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected CheckClosed delta to skip line and linked modifier after ItemServed, got %+v", repo.documents)
	}
	if got := ledgerQuantityByCatalogItem(repo.documents[0]); got["mod-item-1"] != "" {
		t.Fatalf("expected no linked modifier row after served line, got %+v", got)
	}
}

func TestRunOnceCheckClosedWritesOnlyUnservedDeltaAndLines(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventItemServed, itemServedPayload(t, "line-1", "item-1", "1.250")),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a2", contracts.EventCheckClosed, checkClosedPayload(t, []map[string]any{
			{
				"order_line_id":          "line-1",
				"catalog_item_id":        "item-1",
				"quantity":               "2.000",
				"unit_code":              "PC",
				"required_for_inventory": true,
			},
			{
				"order_line_id":          "line-2",
				"catalog_item_id":        "item-2",
				"quantity":               "3.000",
				"unit_code":              "PC",
				"required_for_inventory": true,
			},
		})),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
		"018f0000-0000-7000-8000-00000000d002", "018f0000-0000-7000-8000-00000000d102", "018f0000-0000-7000-8000-00000000d103",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 2 || len(repo.documents[1].Ledger) != 2 {
		t.Fatalf("expected served document plus CheckClosed delta document, got %+v", repo.documents)
	}
	got := map[string]string{}
	for _, entry := range repo.documents[1].Ledger {
		got[entry.OrderLineID] = entry.Quantity
	}
	if got["line-1"] != "0.750" || got["line-2"] != "3.000" {
		t.Fatalf("unexpected CheckClosed delta quantities: %+v", got)
	}
}

func TestRunOnceCheckClosedReplayIsIdempotent(t *testing.T) {
	payload := checkClosedPayload(t, []map[string]any{{
		"order_line_id":          "line-1",
		"catalog_item_id":        "item-1",
		"quantity":               "2.000",
		"unit_code":              "PC",
		"required_for_inventory": true,
	}})
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventCheckClosed, payload),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a1", contracts.EventCheckClosed, payload),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
		"018f0000-0000-7000-8000-00000000d002", "018f0000-0000-7000-8000-00000000d102",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected replay to keep one CheckClosed document, got %+v", repo.documents)
	}
	if len(repo.processed) != 2 {
		t.Fatalf("expected both queue rows processed, got %+v", repo.processed)
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

func TestRunOnceCreatesWasteLedgerFromStockWriteOff(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockWriteOffCaptured, stockWriteOffPayload(t))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("expected one stock document, got %+v", repo.documents)
	}
	doc := repo.documents[0]
	if doc.Type != app.DocumentWaste || len(doc.Ledger) != 1 || doc.Ledger[0].MovementType != app.MovementOut {
		t.Fatalf("unexpected write-off document: %+v", doc)
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

func TestRunOnceProjectsStopListUpdatedWithoutStockDocument(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStopListUpdated, stopListUpdatedPayload(t, "edge_overlay_until_next_publication"))}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 0 {
		t.Fatalf("StopListUpdated must not create stock document, got %+v", repo.documents)
	}
	if len(repo.stopListUpdates) != 1 {
		t.Fatalf("expected one stop-list projection command, got %+v", repo.stopListUpdates)
	}
	got := repo.stopListUpdates[0]
	if got.SourceEventID != "018f0000-0000-7000-8000-0000000000a1" || got.CatalogItemID != "item-1" || got.AvailableQuantity != "0.000" {
		t.Fatalf("unexpected stop-list projection: %+v", got)
	}
	if got.ConflictPolicy != contracts.StopListConflictPolicyEdgeOverlayUntilNextPublication {
		t.Fatalf("unexpected conflict policy: %+v", got)
	}
	if len(repo.processed) != 1 || repo.processed[0] != "queue-1" {
		t.Fatalf("expected queue row processed, got %+v", repo.processed)
	}
}

func TestRunOnceDefaultsStopListPolicyToManagerReview(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStopListUpdated, stopListUpdatedPayload(t, ""))}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.stopListUpdates) != 1 {
		t.Fatalf("expected one stop-list projection command, got %+v", repo.stopListUpdates)
	}
	if repo.stopListUpdates[0].ConflictPolicy != contracts.DefaultStopListConflictPolicy {
		t.Fatalf("expected default manager-review policy, got %+v", repo.stopListUpdates[0])
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

func TestRunOnceManualReviewDispositionFailsWithoutStockDocument(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationPayload(t, "refund", "manual_review"))}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 0 {
		t.Fatalf("expected no document for manual review disposition, got %+v", repo.documents)
	}
	if reason := repo.failed["queue-1"]; !strings.Contains(reason, "manual_review") {
		t.Fatalf("expected manual_review queue failure, got %+v", repo.failed)
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
func (f *failingRepo) ApplyStopListUpdate(context.Context, app.StopListProjectionCommand) error {
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
func (f *failingRepo) ListServedOrderLineQuantities(context.Context, string, []string) (map[string]string, error) {
	return nil, f.err
}
func (f *failingRepo) HasSupersedingServedEvent(context.Context, string, string, string) (bool, error) {
	return false, f.err
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

func sampleQueuedEvent(t *testing.T, eventType contracts.EventType, payload json.RawMessage) app.QueuedEvent {
	t.Helper()
	return queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", eventType, payload)
}

func queuedEvent(t *testing.T, queueID, eventID string, eventType contracts.EventType, payload json.RawMessage) app.QueuedEvent {
	t.Helper()
	return app.QueuedEvent{
		ID:           queueID,
		ReceiptID:    "receipt-1",
		RestaurantID: "restaurant-1",
		DeviceID:     "device-1",
		EventID:      eventID,
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

func itemServedPayload(t *testing.T, orderLineID, catalogItemID, quantity string) json.RawMessage {
	t.Helper()
	return itemServedPayloadWithServedEventID(t, "served-event-1", "", 1, orderLineID, catalogItemID, quantity)
}

func itemServedPayloadWithServedEventID(t *testing.T, servedEventID, supersedesServedEventID string, serveSequence int, orderLineID, catalogItemID, quantity string) json.RawMessage {
	t.Helper()
	payload := map[string]any{
		"served_event_id": servedEventID,
		"ticket_id":       "ticket-1",
		"serve_sequence":  serveSequence,
		"order_id":        "order-1",
		"order_line_id":   orderLineID,
		"catalog_item_id": catalogItemID,
		"quantity":        quantity,
		"unit_code":       "PC",
		"served_at":       "2026-05-05T08:50:00Z",
		"station_id":      "kitchen-hot",
	}
	if strings.TrimSpace(supersedesServedEventID) != "" {
		payload["supersedes_served_event_id"] = supersedesServedEventID
	}
	return marshalPayload(t, payload)
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

func stockWriteOffPayload(t *testing.T) json.RawMessage {
	t.Helper()
	return marshalPayload(t, map[string]any{
		"write_off_id":        "writeoff-1",
		"restaurant_id":       "restaurant-1",
		"business_date_local": "2026-05-05",
		"written_off_at":      "2026-05-05T09:00:00Z",
		"reason_code":         "expired",
		"items": []map[string]any{{
			"catalog_item_id": "item-1",
			"quantity":        "2.000",
			"unit_code":       "KG",
		}},
	})
}

func stopListUpdatedPayload(t *testing.T, policy string) json.RawMessage {
	t.Helper()
	data := map[string]any{
		"stop_list_id":       "stop-1",
		"restaurant_id":      "restaurant-1",
		"catalog_item_id":    "item-1",
		"available_quantity": "0.000",
		"active":             true,
		"source":             "edge",
		"reason":             "ingredient_unavailable",
		"updated_at":         "2026-05-05T12:05:00Z",
	}
	if strings.TrimSpace(policy) != "" {
		data["conflict_policy"] = policy
	}
	return marshalPayload(t, data)
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

func ledgerQuantityByCatalogItem(document app.StockDocument) map[string]string {
	out := map[string]string{}
	for _, entry := range document.Ledger {
		out[entry.CatalogItemID] = entry.Quantity
	}
	return out
}
