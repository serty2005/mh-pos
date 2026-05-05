package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/platform/clock"
)

type Repository interface {
	ReceiveEdgeEvent(context.Context, EdgeEventReceipt) (contracts.EventAck, error)
}

type EdgeEventReceipt struct {
	Envelope         contracts.SyncEnvelope
	IdempotencyKey   string
	RawPayload       []byte
	RawPayloadSHA256 string
	CloudReceivedAt  time.Time
}

type Service struct {
	repo  Repository
	clock clock.Clock
}

func NewService(repo Repository, clock clock.Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

func (s *Service) Receive(ctx context.Context, raw []byte) (contracts.EventAck, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return contracts.EventAck{}, fmt.Errorf("%w: empty body", contracts.ErrInvalidEnvelope)
	}
	var envelope contracts.SyncEnvelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&envelope); err != nil {
		return contracts.EventAck{}, fmt.Errorf("%w: %v", contracts.ErrInvalidEnvelope, err)
	}
	key, err := contracts.IdempotencyKey(envelope)
	if err != nil {
		return contracts.EventAck{}, err
	}
	sum := sha256.Sum256(raw)
	receivedAt := s.clock.Now().UTC()
	return s.repo.ReceiveEdgeEvent(ctx, EdgeEventReceipt{
		Envelope:         envelope,
		IdempotencyKey:   key,
		RawPayload:       append([]byte(nil), raw...),
		RawPayloadSHA256: hex.EncodeToString(sum[:]),
		CloudReceivedAt:  receivedAt,
	})
}
