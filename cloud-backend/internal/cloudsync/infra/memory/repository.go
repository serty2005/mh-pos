package memory

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
)

type Repository struct {
	mu                sync.Mutex
	nextID            int
	events            map[string]storedEvent
	rawByID           map[string][]byte
	masterDataByKey   map[string]contracts.MasterDataPackage
	eventStatsByKey   map[string]EventTypeProjection
	shiftFinanceByKey map[string]ShiftFinanceProjection
	financialOpsByID  map[string]contracts.FinancialOperationProjection
	inventoryQueue    map[string]contracts.EventAck
	authorizedNodes   map[string]authorizedNode
}

type authorizedNode struct {
	RestaurantID    string
	CredentialsHash string
	Status          string
}

type storedEvent struct {
	ack  contracts.EventAck
	view contracts.EdgeEventView
	raw  []byte
}

type EventTypeProjection struct {
	RestaurantID      string
	DeviceID          string
	EventType         string
	EventCount        int64
	FirstOccurredAt   time.Time
	LastOccurredAt    time.Time
	LastCloudReceived time.Time
	LastEventID       string
	LastCommandID     string
	UpdatedAt         time.Time
}

type ShiftFinanceProjection struct {
	RestaurantID          string
	DeviceID              string
	ShiftID               string
	PaymentsCapturedCount int64
	PaymentsCapturedTotal int64
	PaymentsRefundedCount int64
	PaymentsRefundedTotal int64
	ChecksCreatedCount    int64
	ChecksTotalAmount     int64
	ChecksRefundedCount   int64
	ChecksRefundedTotal   int64
	LastEventID           string
	LastCommandID         string
	LastOccurredAt        time.Time
	LastCloudReceived     time.Time
	UpdatedAt             time.Time
}

func NewRepository() *Repository {
	return &Repository{
		events:            map[string]storedEvent{},
		rawByID:           map[string][]byte{},
		masterDataByKey:   map[string]contracts.MasterDataPackage{},
		eventStatsByKey:   map[string]EventTypeProjection{},
		shiftFinanceByKey: map[string]ShiftFinanceProjection{},
		financialOpsByID:  map[string]contracts.FinancialOperationProjection{},
		inventoryQueue:    map[string]contracts.EventAck{},
		authorizedNodes:   map[string]authorizedNode{},
	}
}

func (r *Repository) ReceiveEdgeEvent(_ context.Context, receipt app.EdgeEventReceipt) (contracts.EventAck, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.events[receipt.IdempotencyKey]; ok {
		if existing.ack.RawPayloadSHA256Hex != receipt.RawPayloadSHA256 {
			return contracts.EventAck{}, contracts.ErrPayloadConflict
		}
		return existing.ack, nil
	}
	r.nextID++
	ack := contracts.EventAck{
		Status:              "accepted",
		IdempotencyKey:      receipt.IdempotencyKey,
		CloudReceiptID:      "mem-receipt-" + strconv.Itoa(r.nextID),
		CommandID:           receipt.Envelope.CommandID,
		EventID:             receipt.Envelope.EventID,
		EdgeEventID:         contracts.EdgeEventID(receipt.Envelope),
		EnvelopeVersion:     receipt.Envelope.Version,
		CloudReceivedAt:     receipt.CloudReceivedAt,
		RawPayloadSHA256Hex: receipt.RawPayloadSHA256,
	}
	r.events[receipt.IdempotencyKey] = storedEvent{
		ack: ack,
		view: contracts.EdgeEventView{
			CloudReceiptID:      ack.CloudReceiptID,
			IdempotencyKey:      ack.IdempotencyKey,
			RestaurantID:        *receipt.Envelope.RestaurantID,
			DeviceID:            receipt.Envelope.DeviceID,
			CommandID:           receipt.Envelope.CommandID,
			EventID:             receipt.Envelope.EventID,
			EdgeEventID:         ack.EdgeEventID,
			EventType:           string(receipt.Envelope.EventType),
			AggregateType:       receipt.Envelope.AggregateType,
			AggregateID:         receipt.Envelope.AggregateID,
			EnvelopeVersion:     receipt.Envelope.Version,
			OccurredAt:          receipt.Envelope.OccurredAt,
			CloudReceivedAt:     receipt.CloudReceivedAt,
			RawPayloadSHA256Hex: receipt.RawPayloadSHA256,
		},
		raw: append([]byte(nil), receipt.RawPayload...),
	}
	r.rawByID[ack.CloudReceiptID] = append([]byte(nil), receipt.RawPayload...)
	r.applyEventTypeProjection(receipt)
	r.applyFinancialOperationProjection(receipt, ack.CloudReceiptID)
	r.applyShiftFinanceProjection(receipt)
	if contracts.IsInventoryRelevantEventType(receipt.Envelope.EventType) {
		r.inventoryQueue[ack.CloudReceiptID] = ack
	}
	return ack, nil
}

// ListEdgeEvents возвращает последние принятые events из memory storage без raw payload.
func (r *Repository) ListEdgeEvents(_ context.Context, filter app.EdgeEventListFilter) ([]contracts.EdgeEventView, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := make([]contracts.EdgeEventView, 0, min(limit, len(r.events)))
	for _, stored := range r.events {
		view := stored.view
		if filter.RestaurantID != "" && view.RestaurantID != filter.RestaurantID {
			continue
		}
		if filter.DeviceID != "" && view.DeviceID != filter.DeviceID {
			continue
		}
		if filter.EventType != "" && view.EventType != filter.EventType {
			continue
		}
		out = append(out, view)
	}
	slices.SortFunc(out, func(a, b contracts.EdgeEventView) int {
		if cmp := b.CloudReceivedAt.Compare(a.CloudReceivedAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(b.CloudReceiptID, a.CloudReceiptID)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListFinancialOperations возвращает current financial operation projection с bounded pagination.
func (r *Repository) ListFinancialOperations(_ context.Context, filter app.FinancialOperationProjectionFilter) ([]contracts.FinancialOperationProjection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	out := make([]contracts.FinancialOperationProjection, 0, min(limit, len(r.financialOpsByID)))
	for _, item := range r.financialOpsByID {
		if filter.RestaurantID != "" && item.RestaurantID != filter.RestaurantID {
			continue
		}
		if filter.BusinessDateFrom != "" && item.BusinessDateLocal < filter.BusinessDateFrom {
			continue
		}
		if filter.BusinessDateTo != "" && item.BusinessDateLocal > filter.BusinessDateTo {
			continue
		}
		if filter.OperationType != "" && item.OperationType != filter.OperationType {
			continue
		}
		if filter.ShiftID != "" && item.ShiftID != filter.ShiftID {
			continue
		}
		if filter.OriginalShiftID != "" && item.OriginalShiftID != filter.OriginalShiftID {
			continue
		}
		if filter.CheckID != "" && item.CheckID != filter.CheckID {
			continue
		}
		out = append(out, copyFinancialOperationProjection(item))
	}
	slices.SortFunc(out, func(a, b contracts.FinancialOperationProjection) int {
		if cmp := b.OperationCreatedAt.Compare(a.OperationCreatedAt); cmp != 0 {
			return cmp
		}
		return strings.Compare(b.OperationID, a.OperationID)
	})
	if offset >= len(out) {
		return []contracts.FinancialOperationProjection{}, nil
	}
	out = out[offset:]
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *Repository) UpsertMasterDataPackage(_ context.Context, v contracts.MasterDataPackage) (contracts.MasterDataPackage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := masterDataKey(v.StreamName, v.NodeDeviceID)
	now := v.UpdatedAt
	existing, ok := r.masterDataByKey[key]
	if ok {
		v.CreatedAt = existing.CreatedAt
	} else if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}
	r.masterDataByKey[key] = copyMasterDataPackage(v)
	return copyMasterDataPackage(v), nil
}

func (r *Repository) GetMasterDataPackage(_ context.Context, streamName, nodeDeviceID string) (contracts.MasterDataPackage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.masterDataByKey[masterDataKey(streamName, nodeDeviceID)]; ok {
		return copyMasterDataPackage(v), nil
	}
	if v, ok := r.masterDataByKey[masterDataKey(streamName, "")]; ok {
		return copyMasterDataPackage(v), nil
	}
	return contracts.MasterDataPackage{}, contracts.ErrNotFound
}

func (r *Repository) AuthenticateNodeToken(_ context.Context, nodeDeviceID, restaurantID, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	node, ok := r.authorizedNodes[strings.TrimSpace(nodeDeviceID)]
	if !ok || node.Status != "assigned" {
		return contracts.ErrSyncUnauthorized
	}
	if strings.TrimSpace(node.RestaurantID) != strings.TrimSpace(restaurantID) {
		return contracts.ErrSyncForbidden
	}
	if subtle.ConstantTimeCompare([]byte(node.CredentialsHash), []byte(secretHash(token))) != 1 {
		return contracts.ErrSyncUnauthorized
	}
	return nil
}

func (r *Repository) AuthorizeNodeForTest(nodeDeviceID, restaurantID, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authorizedNodes[strings.TrimSpace(nodeDeviceID)] = authorizedNode{
		RestaurantID:    strings.TrimSpace(restaurantID),
		CredentialsHash: secretHash(token),
		Status:          "assigned",
	}
	return nil
}

func (r *Repository) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func (r *Repository) RawPayload(receiptID string) []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]byte(nil), r.rawByID[receiptID]...)
}

func (r *Repository) EventTypeStats() []EventTypeProjection {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]EventTypeProjection, 0, len(r.eventStatsByKey))
	for _, item := range r.eventStatsByKey {
		out = append(out, item)
	}
	return out
}

func (r *Repository) ShiftFinance() []ShiftFinanceProjection {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ShiftFinanceProjection, 0, len(r.shiftFinanceByKey))
	for _, item := range r.shiftFinanceByKey {
		out = append(out, item)
	}
	return out
}

func (r *Repository) InventoryQueueCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.inventoryQueue)
}

func (r *Repository) applyEventTypeProjection(receipt app.EdgeEventReceipt) {
	key := strings.Join([]string{
		strings.TrimSpace(*receipt.Envelope.RestaurantID),
		strings.TrimSpace(receipt.Envelope.DeviceID),
		string(receipt.Envelope.EventType),
	}, "|")
	current := r.eventStatsByKey[key]
	current.RestaurantID = strings.TrimSpace(*receipt.Envelope.RestaurantID)
	current.DeviceID = strings.TrimSpace(receipt.Envelope.DeviceID)
	current.EventType = string(receipt.Envelope.EventType)
	current.EventCount++
	if current.FirstOccurredAt.IsZero() || receipt.Envelope.OccurredAt.Before(current.FirstOccurredAt) {
		current.FirstOccurredAt = receipt.Envelope.OccurredAt
	}
	if receipt.Envelope.OccurredAt.After(current.LastOccurredAt) || current.LastOccurredAt.IsZero() {
		current.LastOccurredAt = receipt.Envelope.OccurredAt
	}
	current.LastCloudReceived = receipt.CloudReceivedAt
	current.LastEventID = receipt.Envelope.EventID
	current.LastCommandID = receipt.Envelope.CommandID
	current.UpdatedAt = receipt.CloudReceivedAt
	r.eventStatsByKey[key] = current
}

func (r *Repository) applyFinancialOperationProjection(receipt app.EdgeEventReceipt, receiptID string) {
	if receipt.Envelope.EventType != contracts.EventCancellationRecorded && receipt.Envelope.EventType != contracts.EventRefundRecorded {
		return
	}
	var payload contracts.Payload[contracts.FinancialOperationRecorded]
	if err := json.Unmarshal(receipt.Envelope.Payload, &payload); err != nil {
		return
	}
	data := payload.Data
	operationID := strings.TrimSpace(data.ID)
	if operationID == "" {
		return
	}
	if _, exists := r.financialOpsByID[operationID]; exists {
		return
	}
	r.financialOpsByID[operationID] = contracts.FinancialOperationProjection{
		OperationID:          operationID,
		EdgeOperationID:      strings.TrimSpace(data.EdgeOperationID),
		EventID:              strings.TrimSpace(receipt.Envelope.EventID),
		ReceiptID:            receiptID,
		RestaurantID:         strings.TrimSpace(data.RestaurantID),
		DeviceID:             strings.TrimSpace(data.DeviceID),
		NodeDeviceID:         stringPtr(receipt.Envelope.NodeDeviceID),
		ClientDeviceID:       trimStringPtr(receipt.Envelope.ClientDeviceID),
		ActorEmployeeID:      trimStringPtr(receipt.Envelope.ActorEmployeeID),
		SessionID:            trimStringPtr(receipt.Envelope.SessionID),
		ShiftID:              strings.TrimSpace(data.ShiftID),
		OriginalShiftID:      strings.TrimSpace(data.OriginalShiftID),
		CheckID:              strings.TrimSpace(data.CheckID),
		PrecheckID:           strings.TrimSpace(data.PrecheckID),
		OperationType:        strings.TrimSpace(data.OperationType),
		OperationKind:        strings.TrimSpace(data.OperationKind),
		Amount:               data.Amount,
		Currency:             strings.TrimSpace(data.Currency),
		BusinessDateLocal:    strings.TrimSpace(data.BusinessDateLocal),
		InventoryDisposition: strings.TrimSpace(data.InventoryDisposition),
		Reason:               strings.TrimSpace(data.Reason),
		CreatedByEmployeeID:  strings.TrimSpace(data.CreatedByEmployeeID),
		ApprovedByEmployeeID: trimStringPtr(data.ApprovedByEmployeeID),
		Snapshot:             append(json.RawMessage(nil), data.Snapshot...),
		OperationCreatedAt:   data.CreatedAt,
		OccurredAt:           receipt.Envelope.OccurredAt,
		CloudReceivedAt:      receipt.CloudReceivedAt,
		RawPayloadSHA256Hex:  receipt.RawPayloadSHA256,
	}
}

func (r *Repository) applyShiftFinanceProjection(receipt app.EdgeEventReceipt) {
	shiftID := ""
	if receipt.Envelope.ShiftID != nil {
		shiftID = strings.TrimSpace(*receipt.Envelope.ShiftID)
	}
	if shiftID == "" {
		return
	}
	key := strings.Join([]string{
		strings.TrimSpace(*receipt.Envelope.RestaurantID),
		strings.TrimSpace(receipt.Envelope.DeviceID),
		shiftID,
	}, "|")
	current := r.shiftFinanceByKey[key]
	current.RestaurantID = strings.TrimSpace(*receipt.Envelope.RestaurantID)
	current.DeviceID = strings.TrimSpace(receipt.Envelope.DeviceID)
	current.ShiftID = shiftID
	switch receipt.Envelope.EventType {
	case contracts.EventPaymentCaptured:
		var payload contracts.Payload[contracts.PaymentCaptured]
		if err := json.Unmarshal(receipt.Envelope.Payload, &payload); err == nil {
			current.PaymentsCapturedCount++
			current.PaymentsCapturedTotal += payload.Data.Amount
		}
	case contracts.EventPaymentRefunded:
		var payload contracts.Payload[contracts.PaymentRefunded]
		if err := json.Unmarshal(receipt.Envelope.Payload, &payload); err == nil {
			current.PaymentsRefundedCount++
			current.PaymentsRefundedTotal += payload.Data.Amount
		}
	case contracts.EventCheckCreated:
		var payload contracts.Payload[contracts.CheckCreated]
		if err := json.Unmarshal(receipt.Envelope.Payload, &payload); err == nil {
			current.ChecksCreatedCount++
			current.ChecksTotalAmount += payload.Data.Total
		}
	case contracts.EventCheckRefunded:
		var payload contracts.Payload[contracts.CheckRefunded]
		if err := json.Unmarshal(receipt.Envelope.Payload, &payload); err == nil {
			current.ChecksRefundedCount++
			current.ChecksRefundedTotal += payload.Data.PaidTotal
		}
	case contracts.EventRefundRecorded:
		var payload contracts.Payload[contracts.FinancialOperationRecorded]
		if err := json.Unmarshal(receipt.Envelope.Payload, &payload); err == nil {
			current.ChecksRefundedCount++
			current.ChecksRefundedTotal += payload.Data.Amount
		}
	default:
		return
	}
	current.LastEventID = receipt.Envelope.EventID
	current.LastCommandID = receipt.Envelope.CommandID
	current.LastOccurredAt = receipt.Envelope.OccurredAt
	current.LastCloudReceived = receipt.CloudReceivedAt
	current.UpdatedAt = receipt.CloudReceivedAt
	r.shiftFinanceByKey[key] = current
}

func masterDataKey(streamName, nodeDeviceID string) string {
	return strings.TrimSpace(streamName) + "|" + strings.TrimSpace(nodeDeviceID)
}

func copyMasterDataPackage(v contracts.MasterDataPackage) contracts.MasterDataPackage {
	copyValue := v
	copyValue.PayloadJSON = append([]byte(nil), v.PayloadJSON...)
	if v.CloudUpdatedAt != nil {
		t := *v.CloudUpdatedAt
		copyValue.CloudUpdatedAt = &t
	}
	return copyValue
}

func copyFinancialOperationProjection(v contracts.FinancialOperationProjection) contracts.FinancialOperationProjection {
	copyValue := v
	copyValue.NodeDeviceID = copyStringPtr(v.NodeDeviceID)
	copyValue.ClientDeviceID = copyStringPtr(v.ClientDeviceID)
	copyValue.ActorEmployeeID = copyStringPtr(v.ActorEmployeeID)
	copyValue.SessionID = copyStringPtr(v.SessionID)
	copyValue.ApprovedByEmployeeID = copyStringPtr(v.ApprovedByEmployeeID)
	copyValue.Snapshot = append(json.RawMessage(nil), v.Snapshot...)
	return copyValue
}

func stringPtr(v string) *string {
	value := strings.TrimSpace(v)
	if value == "" {
		return nil
	}
	return &value
}

func trimStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	return stringPtr(*v)
}

func copyStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	value := *v
	return &value
}

func secretHash(v string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(v)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
