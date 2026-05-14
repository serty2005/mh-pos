package memory

import (
	"context"
	"encoding/json"
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
}

type storedEvent struct {
	ack contracts.EventAck
	raw []byte
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
	r.events[receipt.IdempotencyKey] = storedEvent{ack: ack, raw: append([]byte(nil), receipt.RawPayload...)}
	r.rawByID[ack.CloudReceiptID] = append([]byte(nil), receipt.RawPayload...)
	r.applyEventTypeProjection(receipt)
	r.applyShiftFinanceProjection(receipt)
	return ack, nil
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
