package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/platform/clock"
)

type DocumentType string
type MovementType string
type ProcessingStatus string

const (
	DocumentSale           DocumentType = "SALE"
	DocumentReturn         DocumentType = "RETURN"
	DocumentWaste          DocumentType = "WASTE"
	DocumentProduction     DocumentType = "PRODUCTION"
	DocumentPurchase       DocumentType = "PURCHASE"
	DocumentInventoryCount DocumentType = "INVENTORY_COUNT"

	MovementIn  MovementType = "IN"
	MovementOut MovementType = "OUT"

	ProcessingStatusAccepted        ProcessingStatus = "accepted"
	ProcessingStatusPosted          ProcessingStatus = "posted"
	ProcessingStatusPartiallyPosted ProcessingStatus = "partially_posted"
	ProcessingStatusFailed          ProcessingStatus = "failed"

	// SourceEventItemServedCompensation помечает сторно уже обработанного ItemServed при recall/serve-again.
	SourceEventItemServedCompensation = "ItemServedCompensation"
)

type IDGenerator interface {
	NewID() string
}

type RecipeLine struct {
	ComponentCatalogItemID string
	Quantity               string
	UnitCode               string
}

type Repository interface {
	ClaimPending(context.Context, ClaimCommand) ([]QueuedEvent, error)
	BeginProcessingState(context.Context, ProcessingStateCommand) (ProcessingState, error)
	CompleteProcessingState(context.Context, ProcessingStateCommand) error
	FailProcessingState(context.Context, ProcessingStateCommand) error
	CreateStockDocument(context.Context, StockDocument) error
	ApplyStopListUpdate(context.Context, StopListProjectionCommand) error
	MarkProcessed(context.Context, string, time.Time) error
	MarkFailed(context.Context, string, string, time.Time) error
	ListActiveRecipeLines(context.Context, string, string) ([]RecipeLine, error)
	ListModifierOptionLinks(context.Context, string, []string) (map[string]string, error)
	ListServedOrderLineQuantities(context.Context, string, []string) (map[string]string, error)
	GetCurrentQuantity(context.Context, string, string, string, string) (string, error)
	HasSupersedingServedEvent(context.Context, string, string, string) (bool, error)
}

type ClaimCommand struct {
	Limit    int
	LockedBy string
	Now      time.Time
}

type QueuedEvent struct {
	ID           string
	ReceiptID    string
	RestaurantID string
	WarehouseID  string
	DeviceID     string
	EventID      string
	EventType    contracts.EventType
	AggregateID  string
	OccurredAt   time.Time
	Payload      json.RawMessage
}

type StockDocument struct {
	ID                string
	RestaurantID      string
	WarehouseID       string
	Type              DocumentType
	SourceEventID     string
	SourceEventType   string
	BusinessDateLocal string
	OccurredAt        time.Time
	CreatedAt         time.Time
	Ledger            []StockLedgerEntry
	ProcessingState   *ProcessingStateCommand
}

type StockLedgerEntry struct {
	ID                string
	RestaurantID      string
	WarehouseID       string
	StockDocumentID   string
	SourceEventID     string
	SourceEventType   string
	CatalogItemID     string
	OrderLineID       string
	MovementType      MovementType
	Quantity          string
	UnitCode          string
	UnitCostMinor     int64
	TotalCostMinor    int64
	CostingStatus     string
	OccurredAt        time.Time
	BusinessDateLocal string
	CreatedAt         time.Time
}

// StopListProjectionCommand переносит StopListUpdated в безопасную projection без raw payload.
type StopListProjectionCommand struct {
	SourceEventID     string
	QueueID           string
	RestaurantID      string
	DeviceID          string
	StopListID        string
	WarehouseID       string
	CatalogItemID     string
	AvailableQuantity string
	Active            bool
	ConflictPolicy    contracts.StopListConflictPolicy
	Source            string
	Reason            string
	UpdatedAt         time.Time
	OccurredAt        time.Time
	ProjectedAt       time.Time
}

type ProcessingState struct {
	ID                  string
	RestaurantID        string
	SourceEventID       string
	SourceEventType     string
	SourceAggregateID   string
	StockDocumentID     string
	Status              ProcessingStatus
	PostedLedgerCount   int
	ExpectedLedgerCount *int
	CostingStatus       string
	NeedsRecalculation  bool
	FailureCode         string
	FailureMessageKey   string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	PostedAt            *time.Time
}

type ProcessingStateCommand struct {
	ID                  string
	RestaurantID        string
	SourceEventID       string
	SourceEventType     string
	SourceAggregateID   string
	StockDocumentID     string
	Status              ProcessingStatus
	PostedLedgerCount   int
	ExpectedLedgerCount *int
	CostingStatus       string
	NeedsRecalculation  bool
	FailureCode         string
	FailureMessageKey   string
	Now                 time.Time
}

type Config struct {
	WorkerID  string
	BatchSize int
}

type Worker struct {
	repo   Repository
	ids    IDGenerator
	clock  clock.Clock
	config Config
	logger *slog.Logger
}

func NewWorker(repo Repository, ids IDGenerator, clock clock.Clock, config Config) *Worker {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = fmt.Sprintf("cloud-inventory-worker-%d", time.Now().UnixNano())
	}
	if config.BatchSize <= 0 || config.BatchSize > 100 {
		config.BatchSize = 25
	}
	return &Worker{repo: repo, ids: ids, clock: clock, config: config, logger: slog.Default()}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	now := w.clock.Now().UTC()
	events, err := w.repo.ClaimPending(ctx, ClaimCommand{Limit: w.config.BatchSize, LockedBy: w.config.WorkerID, Now: now})
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := w.processEvent(ctx, event, now); err != nil {
			if markErr := w.repo.MarkFailed(ctx, event.ID, safeError(err), now); markErr != nil {
				return markErr
			}
			continue
		}
		if err := w.repo.MarkProcessed(ctx, event.ID, now); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processEvent(ctx context.Context, event QueuedEvent, now time.Time) error {
	if isDocumentProcessingStateEvent(event.EventType) {
		return w.processStatefulDocumentEvent(ctx, event, now)
	}
	documents, err := w.documentsFromEvent(ctx, event, now)
	if err != nil {
		return err
	}
	for _, document := range documents {
		if err := w.repo.CreateStockDocument(ctx, document); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processStatefulDocumentEvent(ctx context.Context, event QueuedEvent, now time.Time) error {
	stateCmd := w.processingStateCommand(event, now)
	state, err := w.repo.BeginProcessingState(ctx, stateCmd)
	if err != nil {
		return err
	}
	if state.Status == ProcessingStatusPosted || state.Status == ProcessingStatusPartiallyPosted || state.Status == ProcessingStatusFailed {
		return nil
	}
	documents, err := w.documentsFromEvent(ctx, event, now)
	if err != nil {
		stateCmd.Status = ProcessingStatusFailed
		stateCmd.FailureCode = "VALIDATION_FAILED"
		stateCmd.FailureMessageKey = "inventory.processing.validation_failed"
		if markErr := w.repo.FailProcessingState(ctx, stateCmd); markErr != nil {
			return markErr
		}
		return nil
	}
	if len(documents) == 0 {
		zero := 0
		stateCmd.Status = ProcessingStatusPosted
		stateCmd.PostedLedgerCount = 0
		stateCmd.ExpectedLedgerCount = &zero
		stateCmd.CostingStatus = "final"
		stateCmd.NeedsRecalculation = false
		return w.repo.CompleteProcessingState(ctx, stateCmd)
	}
	for _, document := range documents {
		document.ProcessingState = &stateCmd
		if err := w.repo.CreateStockDocument(ctx, document); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processingStateCommand(event QueuedEvent, now time.Time) ProcessingStateCommand {
	return ProcessingStateCommand{
		ID:                w.ids.NewID(),
		RestaurantID:      strings.TrimSpace(event.RestaurantID),
		SourceEventID:     strings.TrimSpace(event.EventID),
		SourceEventType:   string(event.EventType),
		SourceAggregateID: strings.TrimSpace(event.AggregateID),
		Status:            ProcessingStatusAccepted,
		CostingStatus:     "estimated",
		Now:               now,
	}
}

func isDocumentProcessingStateEvent(eventType contracts.EventType) bool {
	switch eventType {
	case contracts.EventStockReceiptCaptured, contracts.EventInventoryCountCaptured, contracts.EventStockWriteOffCaptured, contracts.EventProductionCompleted:
		return true
	default:
		return false
	}
}

func (w *Worker) documentsFromEvent(ctx context.Context, event QueuedEvent, now time.Time) ([]StockDocument, error) {
	switch event.EventType {
	case contracts.EventCheckClosed:
		doc, ok, err := w.checkClosedDocument(ctx, event, now)
		return singleDocument(doc, ok, err)
	case contracts.EventItemServed:
		return w.itemServedDocuments(ctx, event, now)
	case contracts.EventStockReceiptCaptured:
		doc, ok, err := w.stockReceiptDocument(event, now)
		return singleDocument(doc, ok, err)
	case contracts.EventInventoryCountCaptured:
		doc, ok, err := w.inventoryCountDocument(ctx, event, now)
		return singleDocument(doc, ok, err)
	case contracts.EventStockWriteOffCaptured:
		doc, ok, err := w.stockWriteOffDocument(event, now)
		return singleDocument(doc, ok, err)
	case contracts.EventProductionCompleted:
		doc, ok, err := w.productionDocument(ctx, event, now)
		return singleDocument(doc, ok, err)
	case contracts.EventRefundRecorded, contracts.EventCancellationRecorded:
		doc, ok, err := w.financialOperationDocument(ctx, event, now)
		return singleDocument(doc, ok, err)
	case contracts.EventStopListUpdated:
		return nil, w.applyStopListUpdated(ctx, event, now)
	default:
		return nil, fmt.Errorf("unsupported inventory event_type %s", event.EventType)
	}
}

func singleDocument(document StockDocument, ok bool, err error) ([]StockDocument, error) {
	if err != nil || !ok {
		return nil, err
	}
	return []StockDocument{document}, nil
}

func (w *Worker) applyStopListUpdated(ctx context.Context, event QueuedEvent, now time.Time) error {
	payload, err := decode[contracts.StopListUpdated](event.Payload)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.RestaurantID) != strings.TrimSpace(event.RestaurantID) {
		return fmt.Errorf("stop-list restaurant mismatch")
	}
	policy := contracts.NormalizeStopListConflictPolicy(data.ConflictPolicy)
	if policy == "" {
		return fmt.Errorf("invalid stop-list conflict policy")
	}
	return w.repo.ApplyStopListUpdate(ctx, StopListProjectionCommand{
		SourceEventID:     strings.TrimSpace(event.EventID),
		QueueID:           strings.TrimSpace(event.ID),
		RestaurantID:      strings.TrimSpace(data.RestaurantID),
		DeviceID:          strings.TrimSpace(event.DeviceID),
		StopListID:        strings.TrimSpace(data.StopListID),
		WarehouseID:       strings.TrimSpace(data.WarehouseID),
		CatalogItemID:     strings.TrimSpace(data.CatalogItemID),
		AvailableQuantity: strings.TrimSpace(data.AvailableQuantity),
		Active:            data.Active,
		ConflictPolicy:    policy,
		Source:            strings.TrimSpace(data.Source),
		Reason:            strings.TrimSpace(data.Reason),
		UpdatedAt:         data.UpdatedAt.UTC(),
		OccurredAt:        event.OccurredAt.UTC(),
		ProjectedAt:       now,
	})
}

func (w *Worker) checkClosedDocument(ctx context.Context, event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.CheckClosed](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	deltaItems, err := w.checkClosedDeltaItems(ctx, event.RestaurantID, payload.Data.Items)
	if err != nil {
		return StockDocument{}, false, err
	}
	items, err := w.expandRecipeItems(ctx, event.RestaurantID, deltaItems)
	if err != nil {
		return StockDocument{}, false, err
	}
	modifierItems, err := w.modifierItemsFromItems(ctx, event.RestaurantID, deltaItems)
	if err != nil {
		return StockDocument{}, false, err
	}
	items = append(items, modifierItems...)
	if len(items) == 0 {
		return StockDocument{}, false, nil
	}
	return w.documentFromItems(event, now, DocumentSale, MovementOut, payload.Data.BusinessDateLocal, items, false)
}

func (w *Worker) itemServedDocuments(ctx context.Context, event QueuedEvent, now time.Time) ([]StockDocument, error) {
	payload, err := decode[contracts.ItemServed](event.Payload)
	if err != nil {
		return nil, err
	}
	servedEventID := strings.TrimSpace(payload.Data.ServedEventID)
	orderLineID := strings.TrimSpace(payload.Data.OrderLineID)
	if servedEventID != "" {
		superseded, err := w.repo.HasSupersedingServedEvent(ctx, event.RestaurantID, orderLineID, servedEventID)
		if err != nil {
			return nil, err
		}
		if superseded {
			return nil, nil
		}
	}
	item := contracts.InventoryItem{
		OrderLineID:   orderLineID,
		CatalogItemID: strings.TrimSpace(payload.Data.CatalogItemID),
		Quantity:      strings.TrimSpace(payload.Data.Quantity),
		UnitCode:      strings.TrimSpace(payload.Data.UnitCode),
	}
	if strings.TrimSpace(payload.Data.SupersedesServedEventID) != "" {
		return w.supersedingItemServedDocuments(ctx, event, now, item)
	}
	effectiveQuantity, err := w.effectiveServedQuantity(ctx, event.RestaurantID, payload.Data.OrderLineID, payload.Data.Quantity)
	if err != nil {
		return nil, err
	}
	if effectiveQuantity == "" {
		return nil, nil
	}
	item.Quantity = effectiveQuantity
	items, err := w.expandRecipeItems(ctx, event.RestaurantID, []contracts.InventoryItem{item})
	if err != nil {
		return nil, err
	}
	doc, ok, err := w.documentFromItems(event, now, DocumentSale, MovementOut, businessDate(event.OccurredAt), items, false)
	return singleDocument(doc, ok, err)
}

func (w *Worker) supersedingItemServedDocuments(ctx context.Context, event QueuedEvent, now time.Time, item contracts.InventoryItem) ([]StockDocument, error) {
	documents := make([]StockDocument, 0, 2)
	servedQuantity, err := w.servedOrderLineQuantity(ctx, event.RestaurantID, item.OrderLineID)
	if err != nil {
		return nil, err
	}
	if positive(servedQuantity) {
		compensationItem := item
		compensationItem.Quantity = servedQuantity
		compensationItems, err := w.expandRecipeItems(ctx, event.RestaurantID, []contracts.InventoryItem{compensationItem})
		if err != nil {
			return nil, err
		}
		doc, ok, err := w.documentFromItemsWithSourceType(event, now, DocumentReturn, MovementIn, businessDate(event.OccurredAt), compensationItems, false, SourceEventItemServedCompensation)
		if err != nil {
			return nil, err
		}
		if ok {
			documents = append(documents, doc)
		}
	}
	if !positive(item.Quantity) {
		return documents, nil
	}
	items, err := w.expandRecipeItems(ctx, event.RestaurantID, []contracts.InventoryItem{item})
	if err != nil {
		return nil, err
	}
	doc, ok, err := w.documentFromItems(event, now, DocumentSale, MovementOut, businessDate(event.OccurredAt), items, false)
	if err != nil {
		return nil, err
	}
	if ok {
		documents = append(documents, doc)
	}
	return documents, nil
}

func (w *Worker) stockWriteOffDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.StockWriteOffCaptured](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	return w.documentFromItems(event, now, DocumentWaste, MovementOut, payload.Data.BusinessDateLocal, payload.Data.Items, false)
}

func (w *Worker) stockReceiptDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.StockReceiptCaptured](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	return w.documentFromItems(event, now, DocumentPurchase, MovementIn, payload.Data.BusinessDateLocal, payload.Data.Items, false)
}

func (w *Worker) inventoryCountDocument(ctx context.Context, event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.InventoryCountCaptured](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	items := make([]contracts.InventoryItem, 0, len(payload.Data.Items))
	for _, item := range payload.Data.Items {
		counted := strings.TrimSpace(item.CountedQuantity)
		if !nonNegative(counted) || strings.TrimSpace(item.CatalogItemID) == "" {
			continue
		}
		unit := strings.TrimSpace(item.UnitCode)
		if unit == "" {
			unit = "PC"
		}
		current, err := w.repo.GetCurrentQuantity(ctx, event.RestaurantID, event.WarehouseID, item.CatalogItemID, unit)
		if err != nil {
			return StockDocument{}, false, err
		}
		delta, movement, ok := countAdjustment(current, counted)
		if !ok {
			continue
		}
		item.Quantity = delta
		item.CountedQuantity = ""
		item.UnitCode = unit
		if movement == MovementOut {
			item.Quantity = "-" + delta
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return StockDocument{}, false, nil
	}
	return w.inventoryCountAdjustmentDocument(event, now, payload.Data.BusinessDateLocal, items)
}

func (w *Worker) productionDocument(ctx context.Context, event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.ProductionCompleted](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	item := contracts.InventoryItem{
		CatalogItemID: payload.Data.SemiFinishedCatalogItemID,
		Quantity:      payload.Data.Quantity,
		UnitCode:      payload.Data.UnitCode,
	}
	doc, ok, err := w.documentFromItems(event, now, DocumentProduction, MovementIn, payload.Data.BusinessDateLocal, []contracts.InventoryItem{item}, false)
	if err != nil || !ok {
		return doc, ok, err
	}
	components, err := w.recipeToLedger(ctx, event.RestaurantID, payload.Data.SemiFinishedCatalogItemID, payload.Data.Quantity, payload.Data.BusinessDateLocal, event, now)
	if err != nil {
		return StockDocument{}, false, err
	}
	doc.Ledger = append(doc.Ledger, components...)
	return doc, true, nil
}

func (w *Worker) financialOperationDocument(ctx context.Context, event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.FinancialOperationRecorded](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	disposition := strings.TrimSpace(payload.Data.InventoryDisposition)
	switch disposition {
	case "no_stock_effect":
		return StockDocument{}, false, nil
	case "manual_review":
		return StockDocument{}, false, fmt.Errorf("inventory disposition manual_review requires operator review")
	case "return_to_stock", "write_off_waste":
	default:
		return StockDocument{}, false, fmt.Errorf("unsupported inventory disposition %s", disposition)
	}
	if len(payload.Data.Items) == 0 {
		return StockDocument{}, false, fmt.Errorf("inventory disposition %s requires item rows", disposition)
	}
	documentType := DocumentReturn
	movement := MovementIn
	if disposition == "write_off_waste" {
		documentType = DocumentWaste
		movement = MovementOut
	}
	baseItems := financialOperationInventoryItems(payload.Data)
	if len(baseItems) == 0 {
		return StockDocument{}, false, nil
	}
	items, err := w.expandRecipeItems(ctx, event.RestaurantID, baseItems)
	if err != nil {
		return StockDocument{}, false, err
	}
	modifierItems, err := w.modifierItemsFromItems(ctx, event.RestaurantID, baseItems)
	if err != nil {
		return StockDocument{}, false, err
	}
	items = append(items, modifierItems...)
	if len(items) == 0 {
		return StockDocument{}, false, nil
	}
	return w.documentFromItems(event, now, documentType, movement, payload.Data.BusinessDateLocal, items, false)
}

func financialOperationInventoryItems(data contracts.FinancialOperationRecorded) []contracts.InventoryItem {
	out := make([]contracts.InventoryItem, 0, len(data.Items))
	for _, item := range data.Items {
		scope := strings.TrimSpace(item.Scope)
		if scope == "" && strings.TrimSpace(item.CatalogItemID) != "" {
			if inventoryItem, ok := directFinancialInventoryItem(item); ok {
				out = append(out, inventoryItem)
			}
			continue
		}
		switch scope {
		case "whole_check":
			raw := item.Snapshot
			if len(raw) == 0 {
				raw = data.Snapshot
			}
			out = append(out, inventoryItemsFromSnapshot(raw, "", "")...)
		case "order_line":
			out = append(out, inventoryItemsFromSnapshot(item.Snapshot, strings.TrimSpace(item.OrderLineID), quantityFromRaw(item.Quantity))...)
		case "modifier_line":
			if inventoryItem, ok := directFinancialInventoryItem(item); ok {
				out = append(out, inventoryItem)
			}
		case "service_charge", "tip", "payment":
			continue
		default:
			if inventoryItem, ok := directFinancialInventoryItem(item); ok {
				out = append(out, inventoryItem)
			}
		}
	}
	return out
}

func directFinancialInventoryItem(item contracts.FinancialOperationItem) (contracts.InventoryItem, bool) {
	quantity := quantityFromRaw(item.Quantity)
	if strings.TrimSpace(item.CatalogItemID) == "" || !positive(quantity) {
		return contracts.InventoryItem{}, false
	}
	unitCode := strings.TrimSpace(item.UnitCode)
	if unitCode == "" {
		unitCode = "PC"
	}
	return contracts.InventoryItem{
		OrderLineID:   strings.TrimSpace(item.OrderLineID),
		CatalogItemID: strings.TrimSpace(item.CatalogItemID),
		Quantity:      quantity,
		UnitCode:      unitCode,
	}, true
}

type financialSnapshotLine struct {
	ID            string                      `json:"id"`
	OrderLineID   string                      `json:"order_line_id"`
	CatalogItemID string                      `json:"catalog_item_id"`
	Quantity      json.RawMessage             `json:"quantity"`
	UnitCode      string                      `json:"unit_code"`
	Modifiers     []financialSnapshotModifier `json:"modifiers,omitempty"`
}

type financialSnapshotModifier struct {
	ID                  string          `json:"id"`
	OrderLineModifierID string          `json:"order_line_modifier_id"`
	ModifierGroupID     string          `json:"modifier_group_id"`
	ModifierOptionID    string          `json:"modifier_option_id"`
	Name                string          `json:"name"`
	Quantity            json.RawMessage `json:"quantity"`
	UnitCode            string          `json:"unit_code"`
}

func inventoryItemsFromSnapshot(raw json.RawMessage, wantedOrderLineID, quantityOverride string) []contracts.InventoryItem {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var snapshot struct {
		CheckSnapshot    json.RawMessage         `json:"check_snapshot"`
		PrecheckSnapshot json.RawMessage         `json:"precheck_snapshot"`
		Lines            []financialSnapshotLine `json:"lines"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil
	}
	if len(snapshot.CheckSnapshot) > 0 && json.Valid(snapshot.CheckSnapshot) {
		if items := inventoryItemsFromSnapshot(snapshot.CheckSnapshot, wantedOrderLineID, quantityOverride); len(items) > 0 {
			return items
		}
	}
	if len(snapshot.PrecheckSnapshot) > 0 && json.Valid(snapshot.PrecheckSnapshot) {
		if items := inventoryItemsFromSnapshot(snapshot.PrecheckSnapshot, wantedOrderLineID, quantityOverride); len(items) > 0 {
			return items
		}
	}
	if len(snapshot.Lines) > 0 {
		out := make([]contracts.InventoryItem, 0, len(snapshot.Lines))
		for _, line := range snapshot.Lines {
			if item, ok := inventoryItemFromSnapshotLine(line, wantedOrderLineID, quantityOverride); ok {
				out = append(out, item)
			}
		}
		return out
	}
	var line financialSnapshotLine
	if err := json.Unmarshal(raw, &line); err != nil {
		return nil
	}
	if item, ok := inventoryItemFromSnapshotLine(line, wantedOrderLineID, quantityOverride); ok {
		return []contracts.InventoryItem{item}
	}
	return nil
}

func inventoryItemFromSnapshotLine(line financialSnapshotLine, wantedOrderLineID, quantityOverride string) (contracts.InventoryItem, bool) {
	orderLineID := strings.TrimSpace(line.OrderLineID)
	if orderLineID == "" {
		orderLineID = strings.TrimSpace(line.ID)
	}
	if strings.TrimSpace(wantedOrderLineID) != "" && orderLineID != strings.TrimSpace(wantedOrderLineID) {
		return contracts.InventoryItem{}, false
	}
	quantity := strings.TrimSpace(quantityOverride)
	if quantity == "" {
		quantity = quantityFromRaw(line.Quantity)
	}
	if strings.TrimSpace(line.CatalogItemID) == "" || !positive(quantity) {
		return contracts.InventoryItem{}, false
	}
	unitCode := strings.TrimSpace(line.UnitCode)
	if unitCode == "" {
		unitCode = "PC"
	}
	item := contracts.InventoryItem{
		OrderLineID:   orderLineID,
		CatalogItemID: strings.TrimSpace(line.CatalogItemID),
		Quantity:      quantity,
		UnitCode:      unitCode,
	}
	for _, modifier := range line.Modifiers {
		optionID := strings.TrimSpace(modifier.ModifierOptionID)
		if optionID == "" {
			continue
		}
		modifierQuantity := quantityFromRaw(modifier.Quantity)
		if modifierQuantity == "" {
			modifierQuantity = "1.000"
		}
		modifierID := strings.TrimSpace(modifier.OrderLineModifierID)
		if modifierID == "" {
			modifierID = strings.TrimSpace(modifier.ID)
		}
		item.Modifiers = append(item.Modifiers, contracts.InventoryModifier{
			OrderLineModifierID: modifierID,
			ModifierGroupID:     strings.TrimSpace(modifier.ModifierGroupID),
			ModifierOptionID:    optionID,
			Name:                strings.TrimSpace(modifier.Name),
			Quantity:            modifierQuantity,
			UnitCode:            strings.TrimSpace(modifier.UnitCode),
		})
	}
	return item, true
}

func quantityFromRaw(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil && number > 0 {
		return fmt.Sprintf("%.3f", number)
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil && positive(text) {
		n, _ := strconv.ParseFloat(strings.TrimSpace(text), 64)
		return fmt.Sprintf("%.3f", n)
	}
	return ""
}

func (w *Worker) documentFromItems(event QueuedEvent, now time.Time, typ DocumentType, movement MovementType, businessDateLocal string, items []contracts.InventoryItem, useCountedQuantity bool) (StockDocument, bool, error) {
	return w.documentFromItemsWithSourceType(event, now, typ, movement, businessDateLocal, items, useCountedQuantity, string(event.EventType))
}

func (w *Worker) documentFromItemsWithSourceType(event QueuedEvent, now time.Time, typ DocumentType, movement MovementType, businessDateLocal string, items []contracts.InventoryItem, useCountedQuantity bool, sourceEventType string) (StockDocument, bool, error) {
	documentID := w.ids.NewID()
	sourceEventType = strings.TrimSpace(sourceEventType)
	document := StockDocument{
		ID:                documentID,
		RestaurantID:      event.RestaurantID,
		WarehouseID:       event.WarehouseID,
		Type:              typ,
		SourceEventID:     event.EventID,
		SourceEventType:   sourceEventType,
		BusinessDateLocal: strings.TrimSpace(businessDateLocal),
		OccurredAt:        event.OccurredAt,
		CreatedAt:         now,
	}
	for _, item := range items {
		quantity := strings.TrimSpace(item.Quantity)
		if useCountedQuantity {
			quantity = strings.TrimSpace(item.CountedQuantity)
		}
		if !positive(quantity) || strings.TrimSpace(item.CatalogItemID) == "" {
			continue
		}
		unitCost := item.UnitCostMinor
		totalCost := totalCostMinor(quantity, unitCost)
		costingStatus := costingStatusForItem(movement, unitCost)
		document.Ledger = append(document.Ledger, StockLedgerEntry{
			ID:                w.ids.NewID(),
			RestaurantID:      event.RestaurantID,
			WarehouseID:       event.WarehouseID,
			StockDocumentID:   documentID,
			SourceEventID:     event.EventID,
			SourceEventType:   sourceEventType,
			CatalogItemID:     strings.TrimSpace(item.CatalogItemID),
			OrderLineID:       strings.TrimSpace(item.OrderLineID),
			MovementType:      movement,
			Quantity:          quantity,
			UnitCode:          strings.TrimSpace(item.UnitCode),
			UnitCostMinor:     unitCost,
			TotalCostMinor:    totalCost,
			CostingStatus:     costingStatus,
			OccurredAt:        event.OccurredAt,
			BusinessDateLocal: strings.TrimSpace(businessDateLocal),
			CreatedAt:         now,
		})
	}
	if len(document.Ledger) == 0 {
		return StockDocument{}, false, fmt.Errorf("inventory event has no ledger rows")
	}
	return document, true, nil
}

func (w *Worker) inventoryCountAdjustmentDocument(event QueuedEvent, now time.Time, businessDateLocal string, items []contracts.InventoryItem) (StockDocument, bool, error) {
	documentID := w.ids.NewID()
	document := StockDocument{
		ID:                documentID,
		RestaurantID:      event.RestaurantID,
		WarehouseID:       event.WarehouseID,
		Type:              DocumentInventoryCount,
		SourceEventID:     event.EventID,
		SourceEventType:   string(event.EventType),
		BusinessDateLocal: strings.TrimSpace(businessDateLocal),
		OccurredAt:        event.OccurredAt,
		CreatedAt:         now,
	}
	for _, item := range items {
		quantity := strings.TrimSpace(item.Quantity)
		movement := MovementIn
		if strings.HasPrefix(quantity, "-") {
			movement = MovementOut
			quantity = strings.TrimPrefix(quantity, "-")
		}
		if !positive(quantity) || strings.TrimSpace(item.CatalogItemID) == "" {
			continue
		}
		document.Ledger = append(document.Ledger, StockLedgerEntry{
			ID:                w.ids.NewID(),
			RestaurantID:      event.RestaurantID,
			WarehouseID:       event.WarehouseID,
			StockDocumentID:   documentID,
			SourceEventID:     event.EventID,
			SourceEventType:   string(event.EventType),
			CatalogItemID:     strings.TrimSpace(item.CatalogItemID),
			MovementType:      movement,
			Quantity:          quantity,
			UnitCode:          strings.TrimSpace(item.UnitCode),
			CostingStatus:     "estimated",
			OccurredAt:        event.OccurredAt,
			BusinessDateLocal: strings.TrimSpace(businessDateLocal),
			CreatedAt:         now,
		})
	}
	if len(document.Ledger) == 0 {
		return StockDocument{}, false, nil
	}
	return document, true, nil
}

func decode[T any](payloadRaw json.RawMessage) (contracts.Payload[T], error) {
	var payload contracts.Payload[T]
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return payload, fmt.Errorf("decode inventory payload: %w", err)
	}
	return payload, nil
}

func businessDate(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

func positive(value string) bool {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && n > 0
}

func nonNegative(value string) bool {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && n >= 0
}

func countAdjustment(current, counted string) (string, MovementType, bool) {
	currentValue, err := strconv.ParseFloat(strings.TrimSpace(current), 64)
	if err != nil {
		currentValue = 0
	}
	countedValue, err := strconv.ParseFloat(strings.TrimSpace(counted), 64)
	if err != nil || countedValue < 0 {
		return "", "", false
	}
	delta := countedValue - currentValue
	if delta > 0.0005 {
		return fmt.Sprintf("%.3f", delta), MovementIn, true
	}
	if delta < -0.0005 {
		return fmt.Sprintf("%.3f", -delta), MovementOut, true
	}
	return "", "", false
}

func costingStatusForItem(movement MovementType, unitCostMinor int64) string {
	if movement == MovementIn && unitCostMinor > 0 {
		return "final"
	}
	return "estimated"
}

func totalCostMinor(quantity string, unitCost int64) int64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(quantity), 64)
	if err != nil || unitCost <= 0 {
		return 0
	}
	return int64(n * float64(unitCost))
}

func (w *Worker) checkClosedDeltaItems(ctx context.Context, restaurantID string, items []contracts.InventoryItem) ([]contracts.InventoryItem, error) {
	orderLineIDs := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		orderLineID := strings.TrimSpace(item.OrderLineID)
		if orderLineID == "" || seen[orderLineID] {
			continue
		}
		seen[orderLineID] = true
		orderLineIDs = append(orderLineIDs, orderLineID)
	}
	served, err := w.repo.ListServedOrderLineQuantities(ctx, restaurantID, orderLineIDs)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.InventoryItem, 0, len(items))
	for _, item := range items {
		orderLineID := strings.TrimSpace(item.OrderLineID)
		if orderLineID == "" {
			out = append(out, item)
			continue
		}
		delta, ok := subtractQuantity(item.Quantity, served[orderLineID])
		if !ok {
			continue
		}
		item.Quantity = delta
		out = append(out, item)
	}
	return out, nil
}

func (w *Worker) effectiveServedQuantity(ctx context.Context, restaurantID, orderLineID, requested string) (string, error) {
	if strings.TrimSpace(orderLineID) == "" {
		if !positive(requested) {
			return "", nil
		}
		return strings.TrimSpace(requested), nil
	}
	served, err := w.repo.ListServedOrderLineQuantities(ctx, restaurantID, []string{orderLineID})
	if err != nil {
		return "", err
	}
	delta, ok := subtractQuantity(requested, served[strings.TrimSpace(orderLineID)])
	if !ok {
		return "", nil
	}
	return delta, nil
}

func (w *Worker) servedOrderLineQuantity(ctx context.Context, restaurantID, orderLineID string) (string, error) {
	orderLineID = strings.TrimSpace(orderLineID)
	if orderLineID == "" {
		return "", nil
	}
	served, err := w.repo.ListServedOrderLineQuantities(ctx, restaurantID, []string{orderLineID})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(served[orderLineID]), nil
}

func subtractQuantity(total, consumed string) (string, bool) {
	totalValue, err := strconv.ParseFloat(strings.TrimSpace(total), 64)
	if err != nil || totalValue <= 0 {
		return "", false
	}
	consumedValue, err := strconv.ParseFloat(strings.TrimSpace(consumed), 64)
	if err != nil {
		consumedValue = 0
	}
	delta := totalValue - consumedValue
	if delta <= 0.0005 {
		return "", false
	}
	return fmt.Sprintf("%.3f", delta), true
}

func safeError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "inventory processing failed"
	}
	if len(msg) > 500 {
		return msg[:500]
	}
	return msg
}

func (w *Worker) expandRecipeItems(ctx context.Context, restaurantID string, items []contracts.InventoryItem) ([]contracts.InventoryItem, error) {
	out := make([]contracts.InventoryItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.CatalogItemID) == "" || !positive(item.Quantity) {
			continue
		}
		lines, err := w.repo.ListActiveRecipeLines(ctx, restaurantID, strings.TrimSpace(item.CatalogItemID))
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			out = append(out, item)
			continue
		}
		for _, line := range lines {
			if !positive(line.Quantity) || strings.TrimSpace(line.UnitCode) == "" {
				return nil, fmt.Errorf("invalid recipe line for %s", item.CatalogItemID)
			}
			q := scaledQuantity(item.Quantity, line.Quantity)
			out = append(out, contracts.InventoryItem{OrderLineID: item.OrderLineID, CatalogItemID: line.ComponentCatalogItemID, Quantity: q, UnitCode: line.UnitCode})
		}
	}
	return out, nil
}

func (w *Worker) recipeToLedger(ctx context.Context, restaurantID, ownerID, baseQty, businessDateLocal string, event QueuedEvent, now time.Time) ([]StockLedgerEntry, error) {
	lines, err := w.repo.ListActiveRecipeLines(ctx, restaurantID, ownerID)
	if err != nil || len(lines) == 0 {
		return nil, err
	}
	entries := make([]StockLedgerEntry, 0, len(lines))
	for _, line := range lines {
		if !positive(line.Quantity) || strings.TrimSpace(line.UnitCode) == "" {
			return nil, fmt.Errorf("invalid recipe line for %s", ownerID)
		}
		entries = append(entries, StockLedgerEntry{ID: w.ids.NewID(), RestaurantID: event.RestaurantID, WarehouseID: event.WarehouseID, SourceEventID: event.EventID, SourceEventType: string(event.EventType), CatalogItemID: line.ComponentCatalogItemID, MovementType: MovementOut, Quantity: scaledQuantity(baseQty, line.Quantity), UnitCode: line.UnitCode, CostingStatus: "estimated", OccurredAt: event.OccurredAt, BusinessDateLocal: businessDateLocal, CreatedAt: now})
	}
	return entries, nil
}

func scaledQuantity(left, right string) string {
	ln, _ := strconv.ParseFloat(strings.TrimSpace(left), 64)
	rn, _ := strconv.ParseFloat(strings.TrimSpace(right), 64)
	return fmt.Sprintf("%.3f", ln*rn)
}

func (w *Worker) modifierItemsFromItems(ctx context.Context, restaurantID string, items []contracts.InventoryItem) ([]contracts.InventoryItem, error) {
	type modRef struct{ qty, unit string }
	refs := map[string][]modRef{}
	optionIDs := make([]string, 0)
	for _, item := range items {
		if !positive(item.Quantity) {
			continue
		}
		for _, modifier := range item.Modifiers {
			optionID := strings.TrimSpace(modifier.ModifierOptionID)
			if optionID == "" {
				continue
			}
			qty := strings.TrimSpace(modifier.Quantity)
			if strings.TrimSpace(qty) == "" {
				qty = "1.000"
			}
			// В inventory invariant выбранная quantity модификатора трактуется как количество на единицу sold line.
			qty = scaledQuantity(item.Quantity, qty)
			refs[optionID] = append(refs[optionID], modRef{qty: qty, unit: modifier.UnitCode})
			if len(refs[optionID]) == 1 {
				optionIDs = append(optionIDs, optionID)
			}
		}
	}
	if len(optionIDs) == 0 {
		return nil, nil
	}
	links, err := w.repo.ListModifierOptionLinks(ctx, restaurantID, optionIDs)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.InventoryItem, 0)
	for _, optionID := range optionIDs {
		cid := strings.TrimSpace(links[optionID])
		if cid == "" {
			continue
		}
		for _, ref := range refs[optionID] {
			unit := strings.TrimSpace(ref.unit)
			if unit == "" {
				unit = "PC"
			}
			out = append(out, contracts.InventoryItem{CatalogItemID: cid, Quantity: ref.qty, UnitCode: unit})
		}
	}
	return out, nil
}
