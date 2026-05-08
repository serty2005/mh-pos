package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/platform/clock"
)

type Repository interface {
	ReceiveEdgeEvent(context.Context, EdgeEventReceipt) (contracts.EventAck, error)
	UpsertMasterDataPackage(context.Context, contracts.MasterDataPackage) (contracts.MasterDataPackage, error)
	GetMasterDataPackage(context.Context, string, string) (contracts.MasterDataPackage, error)
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

// ReceiveBatch принимает batch SyncEnvelope и возвращает item-level ACK decisions.
func (s *Service) ReceiveBatch(ctx context.Context, raws [][]byte) contracts.BatchEventAck {
	items := make([]contracts.BatchEventAckItem, 0, len(raws))
	allAccepted := true
	for i, raw := range raws {
		ack, err := s.Receive(ctx, raw)
		if err == nil {
			items = append(items, contracts.BatchEventAckItem{
				Index:  i,
				Status: contracts.BatchItemAccepted,
				Ack:    &ack,
			})
			continue
		}
		allAccepted = false
		item := contracts.BatchEventAckItem{
			Index: i,
			Error: err.Error(),
		}
		switch {
		case errors.Is(err, contracts.ErrInvalidEnvelope):
			item.Status = contracts.BatchItemRejected
			item.ErrorCode = "INVALID_ENVELOPE"
		case errors.Is(err, contracts.ErrPayloadConflict):
			item.Status = contracts.BatchItemRejected
			item.ErrorCode = "PAYLOAD_CONFLICT"
		default:
			item.Status = contracts.BatchItemRetryable
			item.ErrorCode = "INTERNAL"
		}
		items = append(items, item)
	}
	status := "accepted"
	if !allAccepted {
		status = "partial"
	}
	return contracts.BatchEventAck{
		Status: status,
		Items:  items,
	}
}

// UpsertMasterDataPackage сохраняет Cloud-authored master/reference/configuration payload для Edge import.
func (s *Service) UpsertMasterDataPackage(ctx context.Context, v contracts.MasterDataPackage) (contracts.MasterDataPackage, error) {
	now := s.clock.Now().UTC()
	v.StreamName = strings.TrimSpace(v.StreamName)
	v.NodeDeviceID = strings.TrimSpace(v.NodeDeviceID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.SyncMode = contracts.NormalizeSyncMode(v.SyncMode)
	v.FullSnapshotReason = strings.TrimSpace(strings.ToLower(v.FullSnapshotReason))
	v.PayloadJSON = bytes.TrimSpace(v.PayloadJSON)
	if err := contracts.ValidateMasterDataPackage(v); err != nil {
		return contracts.MasterDataPackage{}, err
	}
	if v.CloudUpdatedAt != nil {
		updated := v.CloudUpdatedAt.UTC()
		v.CloudUpdatedAt = &updated
	}
	v.CreatedAt = now
	v.UpdatedAt = now
	return s.repo.UpsertMasterDataPackage(ctx, v)
}

// GetMasterDataPackage возвращает Cloud-authored package для запрошенных stream/node.
func (s *Service) GetMasterDataPackage(ctx context.Context, streamName, nodeDeviceID string) (contracts.MasterDataPackage, error) {
	streamName = strings.TrimSpace(streamName)
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	if err := contracts.ValidateMasterDataStream(streamName); err != nil {
		return contracts.MasterDataPackage{}, err
	}
	return s.repo.GetMasterDataPackage(ctx, streamName, nodeDeviceID)
}
