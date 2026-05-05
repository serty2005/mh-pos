package shared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type OutboxService struct {
	repo  ports.OutboxRepository
	clock clock.Clock
}

type eventOutboxRepository interface {
	ports.OutboxRepository
	ports.LocalEventRepository
}

func NewOutboxService(repo ports.OutboxRepository, clock clock.Clock) *OutboxService {
	return &OutboxService{repo: repo, clock: clock}
}

func (s *OutboxService) ListOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	return s.repo.ListOutbox(ctx, limit)
}

func (s *OutboxService) MarkOutboxSent(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: outbox id is required", domain.ErrInvalid)
	}
	return s.repo.MarkOutboxSent(ctx, id, DBTime(s.clock.Now()))
}

func (s *OutboxService) MarkOutboxFailed(ctx context.Context, id, reason string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: outbox id and error are required", domain.ErrInvalid)
	}
	return s.repo.MarkOutboxFailed(ctx, id, reason, DBTime(s.clock.Now()))
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
	commandID := strings.TrimSpace(meta.CommandID)
	if commandID == "" {
		commandID = ids.NewID()
	}
	origin := NormalizeOrigin(meta.Origin)
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
		Version:       domain.SyncEnvelopeVersion,
		EventID:       eventID,
		CommandID:     commandID,
		EventType:     commandType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		RestaurantID:  OptionalID(restaurantID),
		DeviceID:      strings.TrimSpace(meta.DeviceID),
		ShiftID:       OptionalID(shiftID),
		OccurredAt:    now,
		Payload:       payloadBody,
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
		DeviceID:        strings.TrimSpace(meta.DeviceID),
		ShiftID:         OptionalID(shiftID),
		PayloadJSON:     string(body),
		OccurredAt:      now,
		CreatedAt:       now,
	}
	if err := repo.CreateLocalEvent(ctx, localEvent); err != nil {
		return err
	}
	msg := &domain.OutboxMessage{
		ID:            ids.NewID(),
		CommandID:     commandID,
		Origin:        origin,
		RestaurantID:  OptionalID(restaurantID),
		DeviceID:      strings.TrimSpace(meta.DeviceID),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		CommandType:   commandType,
		PayloadJSON:   string(body),
		Status:        domain.OutboxPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return repo.CreateOutboxMessage(ctx, msg)
}
