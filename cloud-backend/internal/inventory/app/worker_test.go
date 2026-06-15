package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
		f.next++
		return fmt.Sprintf("018f0000-0000-7000-8000-%012d", f.next)
	}
	value := f.values[f.next]
	f.next++
	return value
}

type fakeRepo struct {
	events              []app.QueuedEvent
	documents           []app.StockDocument
	processingStates    map[string]app.ProcessingState
	stopListUpdates     []app.StopListProjectionCommand
	processed           []string
	failed              map[string]string
	recipes             map[string][]app.RecipeLine
	modifierOptionLinks map[string]string
	supersededServedIDs map[string]bool
	recalculationJobs   []app.RecalculationJob
	recalculationRows   map[string][]app.RecalculationLedgerRow
	recalculationEdges  map[string][]recalculationEdge
	recalculationOrder  []string
	recalculationErrors map[string]app.RecalculationJobFailure
}

type recalculationEdge struct {
	from string
	to   string
}

func (f *fakeRepo) ClaimPending(context.Context, app.ClaimCommand) ([]app.QueuedEvent, error) {
	out := append([]app.QueuedEvent(nil), f.events...)
	f.events = nil
	return out, nil
}

func (f *fakeRepo) CreateStockDocument(_ context.Context, document app.StockDocument) error {
	if document.ProcessingState != nil {
		if state := f.processingState(*document.ProcessingState); state.Status == app.ProcessingStatusPosted || state.Status == app.ProcessingStatusPartiallyPosted || state.Status == app.ProcessingStatusFailed {
			return nil
		}
	}
	for _, existing := range f.documents {
		if existing.SourceEventID == document.SourceEventID && existing.SourceEventType == document.SourceEventType {
			return nil
		}
	}
	f.documents = append(f.documents, document)
	if document.ProcessingState != nil {
		expected := len(document.Ledger)
		state := f.processingState(*document.ProcessingState)
		state.StockDocumentID = document.ID
		state.Status = app.ProcessingStatusPosted
		state.PostedLedgerCount = len(document.Ledger)
		state.ExpectedLedgerCount = &expected
		state.CostingStatus, state.NeedsRecalculation = aggregateTestCosting(document.Ledger)
		f.setProcessingState(state)
	}
	return nil
}

func (f *fakeRepo) CreateRecalculationJob(_ context.Context, cmd app.RecalculationTriggerCommand) error {
	for _, existing := range f.recalculationJobs {
		if existing.RestaurantID == cmd.RestaurantID && existing.TriggerType == cmd.TriggerType && existing.TriggerEventID == cmd.TriggerEventID && cmd.TriggerEventID != "" {
			return nil
		}
	}
	affected := map[string]bool{}
	warehouses := map[string]bool{}
	for _, entry := range cmd.Ledger {
		affected[entry.CatalogItemID] = true
		warehouses[entry.WarehouseID] = true
		f.collectRecipeDependents(entry.CatalogItemID, affected, map[string]bool{}, nil)
	}
	rows := make([]app.RecalculationLedgerRow, 0)
	for _, document := range f.documents {
		for _, entry := range document.Ledger {
			if document.ID == cmd.SourceDocumentID {
				continue
			}
			if entry.RestaurantID != cmd.RestaurantID || !affected[entry.CatalogItemID] || !entry.OccurredAt.After(cmd.OccurredAt) {
				continue
			}
			rows = append(rows, app.RecalculationLedgerRow{
				ID:            entry.ID,
				RestaurantID:  entry.RestaurantID,
				WarehouseID:   entry.WarehouseID,
				CatalogItemID: entry.CatalogItemID,
				MovementType:  entry.MovementType,
				Quantity:      entry.Quantity,
				UnitCode:      entry.UnitCode,
				UnitCostMinor: entry.UnitCostMinor,
				OccurredAt:    entry.OccurredAt,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	job := app.RecalculationJob{ID: cmd.ID, RestaurantID: cmd.RestaurantID, TriggerType: cmd.TriggerType, TriggerEventID: cmd.TriggerEventID, Status: app.RecalculationStatusQueued, TotalSteps: len(rows)}
	f.recalculationJobs = append(f.recalculationJobs, job)
	if f.recalculationRows == nil {
		f.recalculationRows = map[string][]app.RecalculationLedgerRow{}
	}
	f.recalculationRows[job.ID] = rows
	if f.recalculationEdges == nil {
		f.recalculationEdges = map[string][]recalculationEdge{}
	}
	for item := range affected {
		f.collectRecipeDependents(item, map[string]bool{}, map[string]bool{}, func(from, to string) {
			f.recalculationEdges[job.ID] = append(f.recalculationEdges[job.ID], recalculationEdge{from: from, to: to})
		})
	}
	_ = warehouses
	return nil
}

func (f *fakeRepo) collectRecipeDependents(component string, affected, visiting map[string]bool, edge func(string, string)) {
	for owner, lines := range f.recipes {
		for _, line := range lines {
			if line.ComponentCatalogItemID != component {
				continue
			}
			if edge != nil {
				edge(component, owner)
			}
			if affected != nil {
				affected[owner] = true
			}
			if visiting[owner] {
				continue
			}
			visiting[owner] = true
			f.collectRecipeDependents(owner, affected, visiting, edge)
			delete(visiting, owner)
		}
	}
}

func (f *fakeRepo) ClaimRecalculationJob(_ context.Context, _ app.RecalculationClaimCommand) (app.RecalculationJob, bool, error) {
	for i := range f.recalculationJobs {
		if f.recalculationJobs[i].Status != app.RecalculationStatusQueued {
			continue
		}
		f.recalculationJobs[i].Status = app.RecalculationStatusRunning
		return f.recalculationJobs[i], true, nil
	}
	return app.RecalculationJob{}, false, nil
}

func (f *fakeRepo) ValidateRecalculationDAG(_ context.Context, jobID string) error {
	edges := f.recalculationEdges[jobID]
	graph := map[string][]string{}
	for _, edge := range edges {
		graph[edge.from] = append(graph[edge.from], edge.to)
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(node string) bool {
		if visiting[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visiting[node] = true
		for _, next := range graph[node] {
			if visit(next) {
				return true
			}
		}
		delete(visiting, node)
		visited[node] = true
		return false
	}
	for node := range graph {
		if visit(node) {
			return errors.New("recipe dependency cycle")
		}
	}
	return nil
}

func (f *fakeRepo) ListRecalculationLedgerRows(_ context.Context, jobID string) ([]app.RecalculationLedgerRow, error) {
	rows := append([]app.RecalculationLedgerRow(nil), f.recalculationRows[jobID]...)
	slices.SortFunc(rows, func(a, b app.RecalculationLedgerRow) int {
		if da, db := a.OccurredAt.Format("2006-01-02"), b.OccurredAt.Format("2006-01-02"); da != db {
			return strings.Compare(da, db)
		}
		if cmp := a.OccurredAt.Compare(b.OccurredAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ID, b.ID)
	})
	return rows, nil
}

func (f *fakeRepo) LatestCostBasis(_ context.Context, q app.CostBasisQuery) (int64, bool, error) {
	var best *app.StockLedgerEntry
	for _, document := range f.documents {
		for i := range document.Ledger {
			entry := &document.Ledger[i]
			if entry.ID == q.LedgerID || entry.RestaurantID != q.RestaurantID || entry.WarehouseID != q.WarehouseID || entry.CatalogItemID != q.CatalogItemID || entry.UnitCode != q.UnitCode || entry.UnitCostMinor <= 0 || entry.OccurredAt.After(q.OccurredAt) {
				continue
			}
			if best == nil || entry.OccurredAt.After(best.OccurredAt) || (entry.OccurredAt.Equal(best.OccurredAt) && entry.ID > best.ID) {
				best = entry
			}
		}
	}
	if best == nil {
		return 0, false, nil
	}
	return best.UnitCostMinor, true, nil
}

func (f *fakeRepo) UpdateRecalculationLedgerRow(_ context.Context, update app.RecalculationLedgerUpdate) error {
	f.recalculationOrder = append(f.recalculationOrder, update.LedgerID)
	for di := range f.documents {
		for ei := range f.documents[di].Ledger {
			if f.documents[di].Ledger[ei].ID != update.LedgerID {
				continue
			}
			f.documents[di].Ledger[ei].UnitCostMinor = update.UnitCostMinor
			f.documents[di].Ledger[ei].TotalCostMinor = update.TotalCostMinor
			f.documents[di].Ledger[ei].CostingStatus = update.CostingStatus
		}
	}
	for i := range f.recalculationJobs {
		if f.recalculationJobs[i].ID == update.JobID {
			f.recalculationJobs[i].CompletedSteps = update.CompletedSteps
		}
	}
	return nil
}

func (f *fakeRepo) CompleteRecalculationJob(_ context.Context, progress app.RecalculationJobProgress) error {
	for i := range f.recalculationJobs {
		if f.recalculationJobs[i].ID == progress.JobID {
			f.recalculationJobs[i].Status = app.RecalculationStatusCompleted
			f.recalculationJobs[i].TotalSteps = progress.TotalSteps
			f.recalculationJobs[i].CompletedSteps = progress.CompletedSteps
		}
	}
	return nil
}

func (f *fakeRepo) FailRecalculationJob(_ context.Context, failure app.RecalculationJobFailure) error {
	if f.recalculationErrors == nil {
		f.recalculationErrors = map[string]app.RecalculationJobFailure{}
	}
	f.recalculationErrors[failure.JobID] = failure
	for i := range f.recalculationJobs {
		if f.recalculationJobs[i].ID == failure.JobID {
			f.recalculationJobs[i].Status = app.RecalculationStatusFailed
			f.recalculationJobs[i].CompletedSteps = failure.CompletedSteps
		}
	}
	return nil
}

func (f *fakeRepo) BeginProcessingState(_ context.Context, cmd app.ProcessingStateCommand) (app.ProcessingState, error) {
	state := f.processingState(cmd)
	if state.ID == "" {
		state = app.ProcessingState{
			ID:              cmd.ID,
			RestaurantID:    cmd.RestaurantID,
			SourceEventID:   cmd.SourceEventID,
			SourceEventType: cmd.SourceEventType,
			Status:          app.ProcessingStatusAccepted,
			CostingStatus:   "estimated",
			CreatedAt:       cmd.Now,
			UpdatedAt:       cmd.Now,
		}
		f.setProcessingState(state)
	}
	return state, nil
}

func (f *fakeRepo) CompleteProcessingState(_ context.Context, cmd app.ProcessingStateCommand) error {
	state := f.processingState(cmd)
	state.Status = cmd.Status
	state.PostedLedgerCount = cmd.PostedLedgerCount
	state.ExpectedLedgerCount = cmd.ExpectedLedgerCount
	state.CostingStatus = cmd.CostingStatus
	state.NeedsRecalculation = cmd.NeedsRecalculation
	state.UpdatedAt = cmd.Now
	f.setProcessingState(state)
	return nil
}

func (f *fakeRepo) FailProcessingState(_ context.Context, cmd app.ProcessingStateCommand) error {
	state := f.processingState(cmd)
	state.Status = app.ProcessingStatusFailed
	state.FailureCode = cmd.FailureCode
	state.FailureMessageKey = cmd.FailureMessageKey
	state.UpdatedAt = cmd.Now
	f.setProcessingState(state)
	return nil
}

func (f *fakeRepo) processingState(cmd app.ProcessingStateCommand) app.ProcessingState {
	if f.processingStates == nil {
		f.processingStates = map[string]app.ProcessingState{}
	}
	return f.processingStates[processingStateKey(cmd.RestaurantID, cmd.SourceEventID, cmd.SourceEventType)]
}

func (f *fakeRepo) setProcessingState(state app.ProcessingState) {
	if f.processingStates == nil {
		f.processingStates = map[string]app.ProcessingState{}
	}
	f.processingStates[processingStateKey(state.RestaurantID, state.SourceEventID, state.SourceEventType)] = state
}

func processingStateKey(restaurantID, sourceEventID, sourceEventType string) string {
	return strings.TrimSpace(restaurantID) + "|" + strings.TrimSpace(sourceEventID) + "|" + strings.TrimSpace(sourceEventType)
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

func (f *fakeRepo) GetCurrentQuantity(_ context.Context, restaurantID, warehouseID, catalogItemID, unitCode string) (string, error) {
	var total float64
	for _, document := range f.documents {
		if strings.TrimSpace(document.RestaurantID) != strings.TrimSpace(restaurantID) || strings.TrimSpace(document.WarehouseID) != strings.TrimSpace(warehouseID) {
			continue
		}
		for _, entry := range document.Ledger {
			if strings.TrimSpace(entry.CatalogItemID) != strings.TrimSpace(catalogItemID) || strings.TrimSpace(entry.UnitCode) != strings.TrimSpace(unitCode) {
				continue
			}
			qty, _ := strconv.ParseFloat(strings.TrimSpace(entry.Quantity), 64)
			if entry.MovementType == app.MovementOut {
				total -= qty
			} else {
				total += qty
			}
		}
	}
	return fmt.Sprintf("%.3f", total), nil
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
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

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
	state := repo.processingStates[processingStateKey("restaurant-1", "018f0000-0000-7000-8000-0000000000a1", string(contracts.EventStockReceiptCaptured))]
	if state.Status != app.ProcessingStatusPosted || state.PostedLedgerCount != 1 || state.CostingStatus != "final" || state.NeedsRecalculation {
		t.Fatalf("unexpected receipt processing state: %+v", state)
	}
}

func TestRunOnceReceiptWithoutCostMarksProcessingStateEstimated(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockReceiptCaptured, marshalPayload(t, map[string]any{
		"receipt_id":          "stock-receipt-1",
		"restaurant_id":       "restaurant-1",
		"business_date_local": "2026-05-05",
		"received_at":         "2026-05-05T09:00:00Z",
		"items": []map[string]any{{
			"catalog_item_id": "item-1",
			"quantity":        "3.000",
			"unit_code":       "KG",
		}},
	}))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	state := repo.processingStates[processingStateKey("restaurant-1", "018f0000-0000-7000-8000-0000000000a1", string(contracts.EventStockReceiptCaptured))]
	if state.CostingStatus != "estimated" || !state.NeedsRecalculation {
		t.Fatalf("receipt without cost must be safely estimated, got %+v", state)
	}
}

func TestRunOnceCreatesInventoryCountLedgerFromCountedQuantity(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventInventoryCountCaptured, inventoryCountPayload(t))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

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
	state := repo.processingStates[processingStateKey("restaurant-1", "018f0000-0000-7000-8000-0000000000a1", string(contracts.EventInventoryCountCaptured))]
	if state.Status != app.ProcessingStatusPosted || state.PostedLedgerCount != 1 || state.CostingStatus != "estimated" || !state.NeedsRecalculation {
		t.Fatalf("unexpected count processing state: %+v", state)
	}
}

func TestRunOnceInventoryCountCreatesOutAdjustmentWhenCurrentExceedsCounted(t *testing.T) {
	repo := &fakeRepo{
		events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventInventoryCountCaptured, inventoryCountPayload(t))},
		documents: []app.StockDocument{{
			ID:           "existing-doc",
			RestaurantID: "restaurant-1",
			Type:         app.DocumentPurchase,
			Ledger: []app.StockLedgerEntry{{
				ID:            "existing-ledger",
				RestaurantID:  "restaurant-1",
				CatalogItemID: "item-1",
				MovementType:  app.MovementIn,
				Quantity:      "10.000",
				UnitCode:      "KG",
			}},
		}},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	doc := repo.documents[1]
	if doc.Type != app.DocumentInventoryCount || len(doc.Ledger) != 1 || doc.Ledger[0].MovementType != app.MovementOut || doc.Ledger[0].Quantity != "2.500" {
		t.Fatalf("expected deterministic OUT adjustment from current 10.000 to counted 7.500, got %+v", doc)
	}
}

func TestRunOnceInventoryCountNoopPostsProcessingStateWithoutDocument(t *testing.T) {
	repo := &fakeRepo{
		events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventInventoryCountCaptured, inventoryCountPayload(t))},
		documents: []app.StockDocument{{
			ID:           "existing-doc",
			RestaurantID: "restaurant-1",
			Type:         app.DocumentPurchase,
			Ledger: []app.StockLedgerEntry{{
				ID:            "existing-ledger",
				RestaurantID:  "restaurant-1",
				CatalogItemID: "item-1",
				MovementType:  app.MovementIn,
				Quantity:      "7.500",
				UnitCode:      "KG",
			}},
		}},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 {
		t.Fatalf("count no-op must not create a stock document, got %+v", repo.documents)
	}
	state := repo.processingStates[processingStateKey("restaurant-1", "018f0000-0000-7000-8000-0000000000a1", string(contracts.EventInventoryCountCaptured))]
	if state.Status != app.ProcessingStatusPosted || state.PostedLedgerCount != 0 || state.ExpectedLedgerCount == nil || *state.ExpectedLedgerCount != 0 || state.NeedsRecalculation {
		t.Fatalf("unexpected no-op count state: %+v", state)
	}
}

func TestRunOnceCreatesProductionLedgerForSemiFinishedItem(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventProductionCompleted, productionPayload(t))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

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
	state := repo.processingStates[processingStateKey("restaurant-1", "018f0000-0000-7000-8000-0000000000a1", string(contracts.EventProductionCompleted))]
	if state.Status != app.ProcessingStatusPosted || state.PostedLedgerCount != 1 || !state.NeedsRecalculation {
		t.Fatalf("production without recipe/cost must post deterministically with recalculation marker, got %+v", state)
	}
}

func TestRunOnceProductionCreatesFinishedInAndIngredientOut(t *testing.T) {
	repo := &fakeRepo{
		events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventProductionCompleted, productionPayload(t))},
		recipes: map[string][]app.RecipeLine{
			"semi-1": {{ComponentCatalogItemID: "ing-1", Quantity: "0.250", UnitCode: "KG"}},
		},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101", "018f0000-0000-7000-8000-00000000d102"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || len(repo.documents[0].Ledger) != 2 {
		t.Fatalf("expected one production document with finished and ingredient rows, got %+v", repo.documents)
	}
	got := map[string]app.StockLedgerEntry{}
	for _, entry := range repo.documents[0].Ledger {
		got[entry.CatalogItemID] = entry
	}
	if got["semi-1"].MovementType != app.MovementIn || got["semi-1"].Quantity != "4.000" {
		t.Fatalf("expected finished IN row, got %+v", got["semi-1"])
	}
	if got["ing-1"].MovementType != app.MovementOut || got["ing-1"].Quantity != "1.000" {
		t.Fatalf("expected ingredient OUT row, got %+v", got["ing-1"])
	}
	state := repo.processingStates[processingStateKey("restaurant-1", "018f0000-0000-7000-8000-0000000000a1", string(contracts.EventProductionCompleted))]
	if state.PostedLedgerCount != 2 || state.Status != app.ProcessingStatusPosted {
		t.Fatalf("unexpected production processing state: %+v", state)
	}
}

func TestRunOnceCreatesWasteLedgerFromStockWriteOff(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockWriteOffCaptured, stockWriteOffPayload(t))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

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

func TestBackdatedReceiptCreatesQueuedRecalculationJob(t *testing.T) {
	future := stockDocumentForTest("doc-future-sale", "event-future-sale", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockReceiptCaptured, stockReceiptPayload(t))},
		documents: []app.StockDocument{future},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.recalculationJobs) != 1 || repo.recalculationJobs[0].Status != app.RecalculationStatusQueued {
		t.Fatalf("expected queued recalculation job, got %+v", repo.recalculationJobs)
	}
	if got := len(repo.recalculationRows[repo.recalculationJobs[0].ID]); got != 1 {
		t.Fatalf("expected one affected future ledger row, got %d", got)
	}
}

func TestNonBackdatedReceiptDoesNotCreateRecalculationJob(t *testing.T) {
	past := stockDocumentForTest("doc-past-sale", "event-past-sale", app.DocumentSale, app.MovementOut, "1.000", "estimated", time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockReceiptCaptured, stockReceiptPayload(t))},
		documents: []app.StockDocument{past},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.recalculationJobs) != 0 {
		t.Fatalf("non-backdated receipt must not create job, got %+v", repo.recalculationJobs)
	}
}

func TestBackdatedCountCreatesJobWithAffectedRange(t *testing.T) {
	future := stockDocumentForTest("doc-future-counted", "event-future-counted", app.DocumentSale, app.MovementOut, "2.000", "needs_recalculation", time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC))
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventInventoryCountCaptured, inventoryCountPayload(t))},
		documents: []app.StockDocument{future},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.recalculationJobs) != 1 {
		t.Fatalf("expected count to create one recalculation job, got %+v", repo.recalculationJobs)
	}
	rows := repo.recalculationRows[repo.recalculationJobs[0].ID]
	if len(rows) != 1 || rows[0].ID != "ledger-doc-future-counted" {
		t.Fatalf("unexpected affected range rows: %+v", rows)
	}
}

func TestBackdatedProductionCreatesDependencyEdges(t *testing.T) {
	future := stockDocumentForTest("doc-future-dish", "event-future-dish", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 11, 0, 0, 0, time.UTC))
	future.Ledger[0].CatalogItemID = "dish-1"
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventProductionCompleted, productionPayload(t))},
		documents: []app.StockDocument{future},
		recipes: map[string][]app.RecipeLine{
			"semi-1": {{ComponentCatalogItemID: "ing-1", Quantity: "0.500", UnitCode: "KG"}},
			"dish-1": {{ComponentCatalogItemID: "semi-1", Quantity: "1.000", UnitCode: "KG"}},
		},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.recalculationJobs) != 1 {
		t.Fatalf("expected production dependency job, got %+v", repo.recalculationJobs)
	}
	edges := repo.recalculationEdges[repo.recalculationJobs[0].ID]
	if len(edges) == 0 {
		t.Fatalf("expected dependency edges, got none")
	}
}

func TestBackdatedWriteOffAffectingFutureLedgerCreatesJob(t *testing.T) {
	future := stockDocumentForTest("doc-future-writeoff", "event-future-writeoff", app.DocumentSale, app.MovementOut, "2.000", "needs_recalculation", time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockWriteOffCaptured, stockWriteOffPayload(t))},
		documents: []app.StockDocument{future},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.recalculationJobs) != 1 {
		t.Fatalf("expected write-off recalculation job, got %+v", repo.recalculationJobs)
	}
}

func TestDuplicateRecalculationTriggerDoesNotCreateDuplicateJob(t *testing.T) {
	future := stockDocumentForTest("doc-future-dup", "event-future-dup", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepo{
		events: []app.QueuedEvent{
			queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventStockReceiptCaptured, stockReceiptPayload(t)),
			queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a1", contracts.EventStockReceiptCaptured, stockReceiptPayload(t)),
		},
		documents: []app.StockDocument{future},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.recalculationJobs) != 1 {
		t.Fatalf("duplicate trigger must keep one job, got %+v", repo.recalculationJobs)
	}
}

func TestRecalculationJobRunsInDeterministicOrderAndUpdatesProgress(t *testing.T) {
	late := stockDocumentForTest("doc-late", "event-late", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC))
	early := stockDocumentForTest("doc-early", "event-early", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC))
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockReceiptCaptured, stockReceiptPayload(t))},
		documents: []app.StockDocument{late, early},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.RunRecalculationOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(repo.recalculationOrder, ",") != "ledger-doc-early,ledger-doc-late" {
		t.Fatalf("unexpected deterministic order: %+v", repo.recalculationOrder)
	}
	job := repo.recalculationJobs[0]
	if job.Status != app.RecalculationStatusCompleted || job.TotalSteps != 2 || job.CompletedSteps != 2 {
		t.Fatalf("unexpected progress: %+v", job)
	}
}

func TestRecalculationUpdatesRowsWithCostBasisAndLeavesMissingBasisSafe(t *testing.T) {
	withBasis := stockDocumentForTest("doc-with-basis", "event-with-basis", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC))
	withoutBasis := stockDocumentForTest("doc-without-basis", "event-without-basis", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC))
	withBasis.Ledger[0].UnitCode = "KG"
	withoutBasis.Ledger[0].UnitCode = "KG"
	withoutBasis.Ledger[0].CatalogItemID = "item-missing-basis"
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockReceiptCaptured, stockReceiptPayload(t))},
		documents: []app.StockDocument{withBasis, withoutBasis},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.RunRecalculationOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	for _, doc := range repo.documents {
		for _, entry := range doc.Ledger {
			statuses[entry.ID] = entry.CostingStatus
		}
	}
	if statuses["ledger-doc-with-basis"] != "recalculated" || statuses["ledger-doc-without-basis"] != "needs_recalculation" {
		t.Fatalf("unexpected recalculated statuses: %+v", statuses)
	}
}

func TestRecipeDependencyCycleFailsRecalculationJobSafely(t *testing.T) {
	future := stockDocumentForTest("doc-future-cycle", "event-future-cycle", app.DocumentSale, app.MovementOut, "1.000", "needs_recalculation", time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC))
	future.Ledger[0].CatalogItemID = "cycle-b"
	repo := &fakeRepo{
		events:    []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockReceiptCaptured, stockReceiptPayload(t))},
		documents: []app.StockDocument{future},
		recipes: map[string][]app.RecipeLine{
			"cycle-a": {{ComponentCatalogItemID: "cycle-b", Quantity: "1.000", UnitCode: "PC"}},
			"cycle-b": {{ComponentCatalogItemID: "item-1", Quantity: "1.000", UnitCode: "PC"}, {ComponentCatalogItemID: "cycle-a", Quantity: "1.000", UnitCode: "PC"}},
		},
	}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.RunRecalculationOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	job := repo.recalculationJobs[0]
	if job.Status != app.RecalculationStatusFailed {
		t.Fatalf("expected failed job for cycle, got %+v", job)
	}
	if got := repo.recalculationErrors[job.ID]; got.FailureCode != "RECIPE_DEPENDENCY_CYCLE" || got.FailureMessageKey == "" {
		t.Fatalf("expected safe failure metadata, got %+v", got)
	}
}

func TestRunOnceStockReceiptReplayDoesNotDuplicateStateDocumentOrLedger(t *testing.T) {
	payload := stockReceiptPayload(t)
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventStockReceiptCaptured, payload),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a1", contracts.EventStockReceiptCaptured, payload),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101", "018f0000-0000-7000-8000-00000000c002"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || len(repo.processingStates) != 1 || len(repo.documents[0].Ledger) != 1 {
		t.Fatalf("receipt replay must keep one state/document/ledger set, states=%+v docs=%+v", repo.processingStates, repo.documents)
	}
	if len(repo.processed) != 2 {
		t.Fatalf("expected both replay queue rows processed, got %+v", repo.processed)
	}
}

func TestRunOnceStockWriteOffReplayDoesNotDoubleWrite(t *testing.T) {
	payload := stockWriteOffPayload(t)
	repo := &fakeRepo{events: []app.QueuedEvent{
		queuedEvent(t, "queue-1", "018f0000-0000-7000-8000-0000000000a1", contracts.EventStockWriteOffCaptured, payload),
		queuedEvent(t, "queue-2", "018f0000-0000-7000-8000-0000000000a1", contracts.EventStockWriteOffCaptured, payload),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101", "018f0000-0000-7000-8000-00000000c002"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || repo.documents[0].Ledger[0].MovementType != app.MovementOut {
		t.Fatalf("write-off replay must not double write, got %+v", repo.documents)
	}
}

func TestRunOnceInvalidStockWriteOffCreatesSafeFailedProcessingState(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventStockWriteOffCaptured, marshalPayload(t, map[string]any{
		"write_off_id":        "writeoff-1",
		"restaurant_id":       "restaurant-1",
		"business_date_local": "2026-05-05",
		"written_off_at":      "2026-05-05T09:00:00Z",
		"reason_code":         "expired",
		"items": []map[string]any{{
			"catalog_item_id": "item-1",
			"quantity":        "0.000",
			"unit_code":       "KG",
		}},
	}))}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{"018f0000-0000-7000-8000-00000000c001", "018f0000-0000-7000-8000-00000000d001"}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 0 || len(repo.failed) != 0 || len(repo.processed) != 1 {
		t.Fatalf("safe validation failure must not create document or retry queue, docs=%+v failed=%+v processed=%+v", repo.documents, repo.failed, repo.processed)
	}
	state := repo.processingStates[processingStateKey("restaurant-1", "018f0000-0000-7000-8000-0000000000a1", string(contracts.EventStockWriteOffCaptured))]
	if state.Status != app.ProcessingStatusFailed || state.FailureCode != "VALIDATION_FAILED" || state.FailureMessageKey != "inventory.processing.validation_failed" {
		t.Fatalf("unexpected failed processing state: %+v", state)
	}
}

func TestRunOnceNonInventoryEventCreatesNoProcessingState(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{sampleQueuedEvent(t, contracts.EventKitchenTicketStatusChanged, marshalPayload(t, map[string]any{
		"status_event_id": "status-event-1",
		"restaurant_id":   "restaurant-1",
		"order_id":        "order-1",
		"order_line_id":   "line-1",
		"from_status":     "new",
		"to_status":       "accepted",
		"changed_at":      "2026-05-05T09:00:00Z",
	}))}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.processingStates) != 0 {
		t.Fatalf("non-inventory event must not create processing state, got %+v", repo.processingStates)
	}
	if repo.failed["queue-1"] == "" {
		t.Fatalf("unexpected queue outcome for unsupported non-inventory event: %+v", repo.failed)
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

func TestRunOnceFinancialOrderLineUsesOperationItemQuantityFromSnapshot(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationOrderLineSnapshotPayload(t, "refund", "return_to_stock", "1")),
	}}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || len(repo.documents[0].Ledger) != 1 {
		t.Fatalf("expected one refund return ledger row, got %+v", repo.documents)
	}
	entry := repo.documents[0].Ledger[0]
	if entry.CatalogItemID != "item-1" || entry.OrderLineID != "line-1" || entry.Quantity != "1.000" || entry.MovementType != app.MovementIn {
		t.Fatalf("partial order_line must use operation item quantity and immutable snapshot, got %+v", entry)
	}
}

func TestRunOnceFinancialWholeCheckExpandsSnapshotLinesAndRecipes(t *testing.T) {
	repo := &fakeRepo{
		events: []app.QueuedEvent{
			sampleQueuedEvent(t, contracts.EventCancellationRecorded, financialOperationWholeCheckSnapshotPayload(t, "cancellation", "write_off_waste")),
		},
		recipes: map[string][]app.RecipeLine{
			"item-1": {{ComponentCatalogItemID: "ing-1", Quantity: "0.500", UnitCode: "KG"}},
		},
		modifierOptionLinks: map[string]string{"mod-opt-1": "mod-item-1"},
	}
	worker := app.NewWorker(repo, &fixedIDs{values: []string{
		"018f0000-0000-7000-8000-00000000d001", "018f0000-0000-7000-8000-00000000d101", "018f0000-0000-7000-8000-00000000d102",
	}}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 1 || repo.documents[0].Type != app.DocumentWaste {
		t.Fatalf("expected one WASTE document, got %+v", repo.documents)
	}
	got := ledgerQuantityByCatalogItem(repo.documents[0])
	if got["ing-1"] != "1.000" || got["mod-item-1"] != "2.000" {
		t.Fatalf("whole-check disposition must expand immutable snapshot lines through current inventory model, got %+v", got)
	}
	for _, entry := range repo.documents[0].Ledger {
		if entry.MovementType != app.MovementOut {
			t.Fatalf("write_off_waste must create OUT ledger rows, got %+v", entry)
		}
	}
}

func TestRunOnceFinancialNonInventoryScopesCreateNoMovement(t *testing.T) {
	for _, scope := range []string{"service_charge", "tip", "payment"} {
		t.Run(scope, func(t *testing.T) {
			repo := &fakeRepo{events: []app.QueuedEvent{
				sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationScopePayload(t, "refund", "return_to_stock", scope)),
			}}
			worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
			if err := worker.RunOnce(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(repo.documents) != 0 || len(repo.processed) != 1 {
				t.Fatalf("%s must not create stock movement, documents=%+v processed=%+v", scope, repo.documents, repo.processed)
			}
		})
	}
}

func TestRunOnceFinancialModifierLineWithoutCatalogCreatesNoMovement(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationModifierLineWithoutCatalogPayload(t)),
	}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 0 || len(repo.processed) != 1 || len(repo.failed) != 0 {
		t.Fatalf("modifier_line without authoritative catalog item must be a safe no-movement outcome, documents=%+v processed=%+v failed=%+v", repo.documents, repo.processed, repo.failed)
	}
}

func TestRunOnceFinancialMissingCatalogItemCreatesNoUnsafeMovement(t *testing.T) {
	repo := &fakeRepo{events: []app.QueuedEvent{
		sampleQueuedEvent(t, contracts.EventRefundRecorded, financialOperationOrderLineMissingCatalogPayload(t)),
	}}
	worker := app.NewWorker(repo, &fixedIDs{}, fixedClock{}, app.Config{WorkerID: "worker-1", BatchSize: 10})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.documents) != 0 || len(repo.processed) != 1 {
		t.Fatalf("missing catalog item must be a safe no-movement outcome, documents=%+v processed=%+v", repo.documents, repo.processed)
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
func (f *failingRepo) CreateRecalculationJob(context.Context, app.RecalculationTriggerCommand) error {
	return f.err
}
func (f *failingRepo) ClaimRecalculationJob(context.Context, app.RecalculationClaimCommand) (app.RecalculationJob, bool, error) {
	return app.RecalculationJob{}, false, f.err
}
func (f *failingRepo) ValidateRecalculationDAG(context.Context, string) error {
	return f.err
}
func (f *failingRepo) ListRecalculationLedgerRows(context.Context, string) ([]app.RecalculationLedgerRow, error) {
	return nil, f.err
}
func (f *failingRepo) LatestCostBasis(context.Context, app.CostBasisQuery) (int64, bool, error) {
	return 0, false, f.err
}
func (f *failingRepo) UpdateRecalculationLedgerRow(context.Context, app.RecalculationLedgerUpdate) error {
	return f.err
}
func (f *failingRepo) CompleteRecalculationJob(context.Context, app.RecalculationJobProgress) error {
	return f.err
}
func (f *failingRepo) FailRecalculationJob(context.Context, app.RecalculationJobFailure) error {
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
func (f *failingRepo) GetCurrentQuantity(context.Context, string, string, string, string) (string, error) {
	return "", f.err
}
func (f *failingRepo) HasSupersedingServedEvent(context.Context, string, string, string) (bool, error) {
	return false, f.err
}
func (f *failingRepo) BeginProcessingState(context.Context, app.ProcessingStateCommand) (app.ProcessingState, error) {
	return app.ProcessingState{}, f.err
}
func (f *failingRepo) CompleteProcessingState(context.Context, app.ProcessingStateCommand) error {
	return f.err
}
func (f *failingRepo) FailProcessingState(context.Context, app.ProcessingStateCommand) error {
	return f.err
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
		AggregateID:  "aggregate-1",
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

func financialOperationOrderLineSnapshotPayload(t *testing.T, operationType, disposition string, quantity any) json.RawMessage {
	t.Helper()
	data := financialOperationData(operationType, disposition)
	data["items"] = []map[string]any{{
		"id":            "operation-item-1",
		"operation_id":  "financial-operation-1",
		"scope":         "order_line",
		"order_line_id": "line-1",
		"quantity":      quantity,
		"amount":        1000,
		"currency":      "RUB",
		"snapshot": map[string]any{
			"id":              "line-1",
			"catalog_item_id": "item-1",
			"quantity":        2,
			"modifiers": []map[string]any{{
				"id":                 "line-mod-1",
				"modifier_option_id": "mod-opt-1",
				"quantity":           1,
			}},
		},
	}}
	return marshalPayload(t, data)
}

func financialOperationWholeCheckSnapshotPayload(t *testing.T, operationType, disposition string) json.RawMessage {
	t.Helper()
	data := financialOperationData(operationType, disposition)
	checkSnapshot := map[string]any{
		"check_id":            "check-1",
		"order_id":            "order-1",
		"precheck_id":         "precheck-1",
		"restaurant_id":       "restaurant-1",
		"business_date_local": "2026-05-05",
		"closed_at":           "2026-05-05T09:00:00Z",
		"precheck_snapshot": map[string]any{
			"lines": []map[string]any{{
				"order_line_id":   "line-1",
				"catalog_item_id": "item-1",
				"quantity":        2,
				"modifiers": []map[string]any{{
					"modifier_option_id": "mod-opt-1",
					"quantity":           1,
					"unit_code":          "PC",
				}},
			}},
		},
	}
	data["snapshot"] = map[string]any{
		"document_type":  "financial_operation",
		"operation_id":   "financial-operation-1",
		"check_snapshot": checkSnapshot,
	}
	data["items"] = []map[string]any{{
		"id":           "operation-item-1",
		"operation_id": "financial-operation-1",
		"scope":        "whole_check",
		"amount":       1000,
		"currency":     "RUB",
		"snapshot":     checkSnapshot,
	}}
	return marshalPayload(t, data)
}

func financialOperationScopePayload(t *testing.T, operationType, disposition, scope string) json.RawMessage {
	t.Helper()
	data := financialOperationData(operationType, disposition)
	data["items"] = []map[string]any{{
		"id":           "operation-item-1",
		"operation_id": "financial-operation-1",
		"scope":        scope,
		"amount":       1000,
		"currency":     "RUB",
	}}
	return marshalPayload(t, data)
}

func financialOperationModifierLineWithoutCatalogPayload(t *testing.T) json.RawMessage {
	t.Helper()
	data := financialOperationData("refund", "return_to_stock")
	data["items"] = []map[string]any{{
		"id":            "operation-item-1",
		"operation_id":  "financial-operation-1",
		"scope":         "modifier_line",
		"order_line_id": "line-1",
		"quantity":      1,
		"amount":        1000,
		"currency":      "RUB",
		"snapshot": map[string]any{
			"modifier_option_id": "mod-opt-1",
			"quantity":           1,
			"unit_code":          "PC",
		},
	}}
	return marshalPayload(t, data)
}

func financialOperationOrderLineMissingCatalogPayload(t *testing.T) json.RawMessage {
	t.Helper()
	data := financialOperationData("refund", "return_to_stock")
	data["items"] = []map[string]any{{
		"id":            "operation-item-1",
		"operation_id":  "financial-operation-1",
		"scope":         "order_line",
		"order_line_id": "line-1",
		"quantity":      1,
		"amount":        1000,
		"currency":      "RUB",
		"snapshot": map[string]any{
			"id":       "line-1",
			"quantity": 1,
		},
	}}
	return marshalPayload(t, data)
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

func stockDocumentForTest(id, eventID string, documentType app.DocumentType, movement app.MovementType, quantity, costingStatus string, occurredAt time.Time) app.StockDocument {
	return app.StockDocument{
		ID:                id,
		RestaurantID:      "restaurant-1",
		Type:              documentType,
		SourceEventID:     eventID,
		SourceEventType:   string(documentType),
		BusinessDateLocal: occurredAt.Format("2006-01-02"),
		OccurredAt:        occurredAt,
		CreatedAt:         occurredAt,
		Ledger: []app.StockLedgerEntry{{
			ID:                "ledger-" + id,
			RestaurantID:      "restaurant-1",
			SourceEventID:     eventID,
			SourceEventType:   string(documentType),
			CatalogItemID:     "item-1",
			MovementType:      movement,
			Quantity:          quantity,
			UnitCode:          "PC",
			CostingStatus:     costingStatus,
			OccurredAt:        occurredAt,
			BusinessDateLocal: occurredAt.Format("2006-01-02"),
			CreatedAt:         occurredAt,
		}},
	}
}

func aggregateTestCosting(ledger []app.StockLedgerEntry) (string, bool) {
	status := "final"
	for _, entry := range ledger {
		switch entry.CostingStatus {
		case "failed":
			return "failed", false
		case "needs_recalculation":
			status = "needs_recalculation"
		case "estimated":
			if status != "needs_recalculation" {
				status = "estimated"
			}
		case "recalculated":
			if status == "final" {
				status = "recalculated"
			}
		}
	}
	return status, status == "estimated" || status == "needs_recalculation"
}
