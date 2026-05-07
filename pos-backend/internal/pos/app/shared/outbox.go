package shared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type OutboxService struct {
	repo  ports.OutboxRepository
	tx    txmanager.Manager
	clock clock.Clock
}

const (
	DefaultOutboxMaxAttempts = 20
	defaultOutboxRetryDelay  = time.Minute
	defaultOutboxMaxDelay    = 30 * time.Minute
)

type eventOutboxRepository interface {
	ports.OutboxRepository
	ports.LocalEventRepository
}

func NewOutboxService(repo ports.OutboxRepository, tx txmanager.Manager, clock clock.Clock) *OutboxService {
	return &OutboxService{repo: repo, tx: tx, clock: clock}
}

func (s *OutboxService) ListOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	return s.repo.ListOutbox(ctx, limit)
}

func (s *OutboxService) GetSyncStatus(ctx context.Context) (domain.SyncStatus, error) {
	return s.repo.GetSyncStatus(ctx)
}

func (s *OutboxService) RetryFailedOutbox(ctx context.Context) (int, error) {
	var count int
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		count, err = s.repo.RetryFailedOutbox(ctx, DBTime(s.clock.Now()))
		return err
	})
	return count, err
}

func (s *OutboxService) ClaimPendingOutbox(ctx context.Context, limit int, lockedBy string) ([]domain.OutboxMessage, error) {
	if strings.TrimSpace(lockedBy) == "" {
		return nil, fmt.Errorf("%w: locked_by is required", domain.ErrInvalid)
	}
	var out []domain.OutboxMessage
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.repo.ClaimPendingOutbox(ctx, limit, strings.TrimSpace(lockedBy), DBTime(s.clock.Now()))
		return err
	})
	return out, err
}

func (s *OutboxService) ReclaimStaleProcessingOutbox(ctx context.Context, staleBefore time.Time) (int, error) {
	if staleBefore.IsZero() {
		return 0, fmt.Errorf("%w: stale_before is required", domain.ErrInvalid)
	}
	var count int
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		count, err = s.repo.ReclaimStaleProcessingOutbox(ctx, DBTime(staleBefore), DBTime(s.clock.Now()))
		return err
	})
	return count, err
}

func (s *OutboxService) ReleaseProcessingOutbox(ctx context.Context, lockedBy string) (int, error) {
	if strings.TrimSpace(lockedBy) == "" {
		return 0, fmt.Errorf("%w: locked_by is required", domain.ErrInvalid)
	}
	var count int
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		count, err = s.repo.ReleaseProcessingOutbox(ctx, strings.TrimSpace(lockedBy), DBTime(s.clock.Now()))
		return err
	})
	return count, err
}

func (s *OutboxService) MarkOutboxSent(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: outbox id is required", domain.ErrInvalid)
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		return s.repo.MarkOutboxSent(ctx, id, DBTime(s.clock.Now()))
	})
}

func (s *OutboxService) MarkOutboxRetryableFailure(ctx context.Context, id, reason string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: outbox id and error are required", domain.ErrInvalid)
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		msg, err := s.repo.GetOutboxByID(ctx, strings.TrimSpace(id))
		if err != nil {
			return err
		}
		now := s.clock.Now()
		nextRetryAt := DBTime(now.Add(outboxBackoffDelay(msg.Attempts + 1)))
		return s.repo.MarkOutboxRetryableFailure(ctx, id, reason, &nextRetryAt, DBTime(now), DefaultOutboxMaxAttempts)
	})
}

func (s *OutboxService) MarkOutboxFailed(ctx context.Context, id, reason string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: outbox id and error are required", domain.ErrInvalid)
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		now := s.clock.Now()
		nextRetryAt := DBTime(now.Add(defaultOutboxRetryDelay))
		return s.repo.MarkOutboxFailed(ctx, id, reason, &nextRetryAt, DBTime(now), DefaultOutboxMaxAttempts)
	})
}

func (s *OutboxService) SuspendOutboxMessage(ctx context.Context, id, reason string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: outbox id and reason are required", domain.ErrInvalid)
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		return s.repo.SuspendOutboxMessage(ctx, strings.TrimSpace(id), reason, DBTime(s.clock.Now()))
	})
}

func outboxBackoffDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return defaultOutboxRetryDelay
	}
	delay := defaultOutboxRetryDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= defaultOutboxMaxDelay {
			return defaultOutboxMaxDelay
		}
	}
	return delay
}

func EnsureCommandNotProcessed(ctx context.Context, repo ports.OutboxRepository, commandID string) error {
	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return nil
	}
	if _, err := repo.GetOutboxByCommandID(ctx, commandID); err == nil {
		return fmt.Errorf("%w: %s", domain.ErrDuplicateCommand, commandID)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	return nil
}

func WriteOutbox(ctx context.Context, repo eventOutboxRepository, ids idgen.Generator, clock clock.Clock, meta CommandMeta, restaurantID, shiftID, aggregateType, aggregateID, commandType string, payload any) error {
	NormalizeDeviceMeta(&meta)
	commandID := strings.TrimSpace(meta.CommandID)
	if commandID == "" {
		commandID = ids.NewID()
	}
	origin := NormalizeOrigin(meta.Origin)
	nodeDeviceID := EffectiveNodeDeviceID(meta)
	eventID := ids.NewID()
	now := clock.Now()
	payloadBody := struct {
		Origin domain.CommandOrigin `json:"origin"`
		Data   any                  `json:"data"`
	}{
		Origin: origin,
		Data:   payload,
	}
	envelope := domain.SyncEnvelope{
		Version:         domain.SyncEnvelopeVersion,
		EventID:         eventID,
		CommandID:       commandID,
		EventType:       commandType,
		AggregateType:   aggregateType,
		AggregateID:     aggregateID,
		RestaurantID:    OptionalID(restaurantID),
		DeviceID:        nodeDeviceID,
		NodeDeviceID:    nodeDeviceID,
		ClientDeviceID:  OptionalID(meta.ClientDeviceID),
		ShiftID:         OptionalID(shiftID),
		ActorEmployeeID: OptionalID(meta.ActorEmployeeID),
		SessionID:       OptionalID(meta.SessionID),
		OccurredAt:      now,
		Payload:         payloadBody,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	localEvent := &domain.LocalEvent{
		ID:              ids.NewID(),
		EventID:         eventID,
		CommandID:       commandID,
		EnvelopeVersion: domain.SyncEnvelopeVersion,
		EventType:       commandType,
		AggregateType:   aggregateType,
		AggregateID:     aggregateID,
		RestaurantID:    OptionalID(restaurantID),
		DeviceID:        nodeDeviceID,
		NodeDeviceID:    nodeDeviceID,
		ClientDeviceID:  OptionalID(meta.ClientDeviceID),
		ShiftID:         OptionalID(shiftID),
		ActorEmployeeID: OptionalID(meta.ActorEmployeeID),
		SessionID:       OptionalID(meta.SessionID),
		PayloadJSON:     string(body),
		OccurredAt:      now,
		CreatedAt:       now,
	}
	if err := repo.CreateLocalEvent(ctx, localEvent); err != nil {
		return err
	}
	msg := &domain.OutboxMessage{
		ID:              ids.NewID(),
		CommandID:       commandID,
		Origin:          origin,
		RestaurantID:    OptionalID(restaurantID),
		DeviceID:        nodeDeviceID,
		NodeDeviceID:    nodeDeviceID,
		ClientDeviceID:  OptionalID(meta.ClientDeviceID),
		ActorEmployeeID: OptionalID(meta.ActorEmployeeID),
		SessionID:       OptionalID(meta.SessionID),
		AggregateType:   aggregateType,
		AggregateID:     aggregateID,
		CommandType:     commandType,
		PayloadJSON:     string(body),
		Status:          domain.OutboxPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return repo.CreateOutboxMessage(ctx, msg)
}
