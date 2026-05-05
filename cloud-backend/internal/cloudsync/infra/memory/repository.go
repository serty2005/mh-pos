package memory

import (
	"context"
	"strconv"
	"sync"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
)

type Repository struct {
	mu      sync.Mutex
	nextID  int
	events  map[string]storedEvent
	rawByID map[string][]byte
}

type storedEvent struct {
	ack contracts.EventAck
	raw []byte
}

func NewRepository() *Repository {
	return &Repository{
		events:  map[string]storedEvent{},
		rawByID: map[string][]byte{},
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
	return ack, nil
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
