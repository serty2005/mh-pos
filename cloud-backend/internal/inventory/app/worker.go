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

const (
	DocumentSale           DocumentType = "SALE"
	DocumentReturn         DocumentType = "RETURN"
	DocumentWaste          DocumentType = "WASTE"
	DocumentProduction     DocumentType = "PRODUCTION"
	DocumentPurchase       DocumentType = "PURCHASE"
	DocumentInventoryCount DocumentType = "INVENTORY_COUNT"

	MovementIn  MovementType = "IN"
	MovementOut MovementType = "OUT"
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
	CreateStockDocument(context.Context, StockDocument) error
	MarkProcessed(context.Context, string, time.Time) error
	MarkFailed(context.Context, string, string, time.Time) error
	ListActiveRecipeLines(context.Context, string, string) ([]RecipeLine, error)
	ListModifierOptionLinks(context.Context, string, []string) (map[string]string, error)
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
	DeviceID     string
	EventID      string
	EventType    contracts.EventType
	OccurredAt   time.Time
	Payload      json.RawMessage
}

type StockDocument struct {
	ID                string
	RestaurantID      string
	Type              DocumentType
	SourceEventID     string
	SourceEventType   string
	BusinessDateLocal string
	OccurredAt        time.Time
	CreatedAt         time.Time
	Ledger            []StockLedgerEntry
}

type StockLedgerEntry struct {
	ID                string
	RestaurantID      string
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
	document, ok, err := w.documentFromEvent(event, now)
	if err != nil || !ok {
		return err
	}
	return w.repo.CreateStockDocument(ctx, document)
}

func (w *Worker) documentFromEvent(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	switch event.EventType {
	case contracts.EventCheckClosed:
		return w.checkClosedDocument(event, now)
	case contracts.EventItemServed:
		return w.itemServedDocument(event, now)
	case contracts.EventStockReceiptCaptured:
		return w.stockReceiptDocument(event, now)
	case contracts.EventInventoryCountCaptured:
		return w.inventoryCountDocument(event, now)
	case contracts.EventProductionCompleted:
		return w.productionDocument(event, now)
	case contracts.EventRefundRecorded, contracts.EventCancellationRecorded:
		return w.financialOperationDocument(event, now)
	case contracts.EventStopListUpdated:
		return StockDocument{}, false, nil
	default:
		return StockDocument{}, false, fmt.Errorf("unsupported inventory event_type %s", event.EventType)
	}
}

func (w *Worker) checkClosedDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.CheckClosed](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	items, err := w.expandRecipeItems(context.Background(), event.RestaurantID, payload.Data.Items)
	if err != nil {
		return StockDocument{}, false, err
	}
	modifierItems, err := w.modifierItemsFromPayload(context.Background(), event.RestaurantID, event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	items = append(items, modifierItems...)
	return w.documentFromItems(event, now, DocumentSale, MovementOut, payload.Data.BusinessDateLocal, items, false)
}

func (w *Worker) itemServedDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.ItemServed](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	item := contracts.InventoryItem{
		OrderLineID:   payload.Data.OrderLineID,
		CatalogItemID: payload.Data.CatalogItemID,
		Quantity:      payload.Data.Quantity,
		UnitCode:      payload.Data.UnitCode,
	}
	items, err := w.expandRecipeItems(context.Background(), event.RestaurantID, []contracts.InventoryItem{item})
	if err != nil {
		return StockDocument{}, false, err
	}
	return w.documentFromItems(event, now, DocumentSale, MovementOut, businessDate(event.OccurredAt), items, false)
}

func (w *Worker) stockReceiptDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.StockReceiptCaptured](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	return w.documentFromItems(event, now, DocumentPurchase, MovementIn, payload.Data.BusinessDateLocal, payload.Data.Items, false)
}

func (w *Worker) inventoryCountDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
	payload, err := decode[contracts.InventoryCountCaptured](event.Payload)
	if err != nil {
		return StockDocument{}, false, err
	}
	return w.documentFromItems(event, now, DocumentInventoryCount, MovementIn, payload.Data.BusinessDateLocal, payload.Data.Items, true)
}

func (w *Worker) productionDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
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
	components, err := w.recipeToLedger(context.Background(), event.RestaurantID, payload.Data.SemiFinishedCatalogItemID, payload.Data.Quantity, payload.Data.BusinessDateLocal, event, now)
	if err != nil {
		return StockDocument{}, false, err
	}
	doc.Ledger = append(doc.Ledger, components...)
	return doc, true, nil
}

func (w *Worker) financialOperationDocument(event QueuedEvent, now time.Time) (StockDocument, bool, error) {
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
	items := make([]contracts.InventoryItem, 0, len(payload.Data.Items))
	for _, item := range payload.Data.Items {
		items = append(items, contracts.InventoryItem{
			OrderLineID:   item.OrderLineID,
			CatalogItemID: item.CatalogItemID,
			Quantity:      item.Quantity,
			UnitCode:      strings.TrimSpace(item.UnitCode),
		})
	}
	return w.documentFromItems(event, now, documentType, movement, payload.Data.BusinessDateLocal, items, false)
}

func (w *Worker) documentFromItems(event QueuedEvent, now time.Time, typ DocumentType, movement MovementType, businessDateLocal string, items []contracts.InventoryItem, useCountedQuantity bool) (StockDocument, bool, error) {
	documentID := w.ids.NewID()
	document := StockDocument{
		ID:                documentID,
		RestaurantID:      event.RestaurantID,
		Type:              typ,
		SourceEventID:     event.EventID,
		SourceEventType:   string(event.EventType),
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
		document.Ledger = append(document.Ledger, StockLedgerEntry{
			ID:                w.ids.NewID(),
			RestaurantID:      event.RestaurantID,
			StockDocumentID:   documentID,
			SourceEventID:     event.EventID,
			SourceEventType:   string(event.EventType),
			CatalogItemID:     strings.TrimSpace(item.CatalogItemID),
			OrderLineID:       strings.TrimSpace(item.OrderLineID),
			MovementType:      movement,
			Quantity:          quantity,
			UnitCode:          strings.TrimSpace(item.UnitCode),
			UnitCostMinor:     unitCost,
			TotalCostMinor:    totalCost,
			CostingStatus:     "estimated",
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

func totalCostMinor(quantity string, unitCost int64) int64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(quantity), 64)
	if err != nil || unitCost <= 0 {
		return 0
	}
	return int64(n * float64(unitCost))
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
		entries = append(entries, StockLedgerEntry{ID: w.ids.NewID(), RestaurantID: event.RestaurantID, SourceEventID: event.EventID, SourceEventType: string(event.EventType), CatalogItemID: line.ComponentCatalogItemID, MovementType: MovementOut, Quantity: scaledQuantity(baseQty, line.Quantity), UnitCode: line.UnitCode, CostingStatus: "estimated", OccurredAt: event.OccurredAt, BusinessDateLocal: businessDateLocal, CreatedAt: now})
	}
	return entries, nil
}

func scaledQuantity(left, right string) string {
	ln, _ := strconv.ParseFloat(strings.TrimSpace(left), 64)
	rn, _ := strconv.ParseFloat(strings.TrimSpace(right), 64)
	return fmt.Sprintf("%.3f", ln*rn)
}

func (w *Worker) modifierItemsFromPayload(ctx context.Context, restaurantID string, raw json.RawMessage) ([]contracts.InventoryItem, error) {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return nil, nil
	}
	data, _ := root["data"].(map[string]any)
	items, _ := data["items"].([]any)
	type modRef struct{ qty, unit string }
	refs := map[string][]modRef{}
	optionIDs := make([]string, 0)
	for _, iv := range items {
		im, _ := iv.(map[string]any)
		mods, _ := im["modifiers"].([]any)
		for _, mv := range mods {
			m, _ := mv.(map[string]any)
			optionID, _ := m["modifier_option_id"].(string)
			optionID = strings.TrimSpace(optionID)
			if optionID == "" {
				continue
			}
			qty, _ := m["quantity"].(string)
			if strings.TrimSpace(qty) == "" {
				qty = "1.000"
			}
			unit, _ := m["unit_code"].(string)
			refs[optionID] = append(refs[optionID], modRef{qty: qty, unit: unit})
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
