package cash

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
}

type OpenCashSessionCommand struct {
	shared.CommandMeta
	RestaurantID       string `json:"restaurant_id"`
	OpenedByEmployeeID string `json:"opened_by_employee_id"`
	OpeningCashAmount  int64  `json:"opening_cash_amount"`
}

type CloseCashSessionCommand struct {
	shared.CommandMeta
	ID                 string `json:"id"`
	ClosedByEmployeeID string `json:"closed_by_employee_id"`
	ClosingCashAmount  int64  `json:"closing_cash_amount"`
}

type RecordCashDrawerEventCommand struct {
	shared.CommandMeta
	CashSessionID       string                     `json:"cash_session_id,omitempty"`
	CreatedByEmployeeID string                     `json:"created_by_employee_id"`
	EventType           domain.CashDrawerEventType `json:"event_type"`
	Amount              int64                      `json:"amount"`
	Reason              string                     `json:"reason,omitempty"`
	Note                string                     `json:"note,omitempty"`
}

func (s *Service) GetCurrentCashSession(ctx context.Context, deviceID string) (*domain.CashSession, error) {
	if strings.TrimSpace(deviceID) == "" {
		return nil, fmt.Errorf("%w: device_id is required", domain.ErrInvalid)
	}
	return s.repo.GetOpenCashSessionByDevice(ctx, deviceID)
}

func (s *Service) OpenCashSession(ctx context.Context, cmd OpenCashSessionCommand) (*domain.CashSession, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.OpenedByEmployeeID) == "" || cmd.OpeningCashAmount < 0 {
		return nil, fmt.Errorf("%w: restaurant_id, opened_by_employee_id and non-negative opening_cash_amount are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var session *domain.CashSession
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta); err != nil {
			return err
		}
		if cmd.Origin == domain.OriginEdgeDevice && cmd.ActorEmployeeID != cmd.OpenedByEmployeeID {
			return fmt.Errorf("%w: opened_by_employee_id must match actor_employee_id", domain.ErrForbidden)
		}
		shift, err := s.repo.GetOpenShiftByDevice(ctx, cmd.DeviceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: cannot open cash session without an open shift", domain.ErrConflict)
			}
			return err
		}
		if shift.RestaurantID != cmd.RestaurantID {
			return fmt.Errorf("%w: restaurant_id does not match open shift", domain.ErrConflict)
		}
		if _, err := s.repo.GetOpenCashSessionByDevice(ctx, cmd.DeviceID); err == nil {
			return fmt.Errorf("%w: device already has an open cash session", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		session = &domain.CashSession{
			ID:                 s.ids.NewID(),
			EdgeCashSessionID:  s.ids.NewID(),
			RestaurantID:       shift.RestaurantID,
			DeviceID:           shift.DeviceID,
			ShiftID:            shift.ID,
			OpenedByEmployeeID: cmd.OpenedByEmployeeID,
			Status:             domain.CashSessionOpen,
			OpeningCashAmount:  cmd.OpeningCashAmount,
			OpenedAt:           now,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := s.repo.CreateCashSession(ctx, session); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, session.RestaurantID, session.ShiftID, "CashSession", session.ID, "CashSessionOpened", session)
	})
	return session, err
}

func (s *Service) CloseCashSession(ctx context.Context, cmd CloseCashSessionCommand) (*domain.CashSession, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.ClosedByEmployeeID) == "" || cmd.ClosingCashAmount < 0 {
		return nil, fmt.Errorf("%w: id, closed_by_employee_id and non-negative closing_cash_amount are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var session *domain.CashSession
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta); err != nil {
			return err
		}
		if cmd.Origin == domain.OriginEdgeDevice && cmd.ActorEmployeeID != cmd.ClosedByEmployeeID {
			return fmt.Errorf("%w: closed_by_employee_id must match actor_employee_id", domain.ErrForbidden)
		}
		var err error
		session, err = s.repo.GetCashSession(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if session.Status != domain.CashSessionOpen {
			return fmt.Errorf("%w: cash session is not open", domain.ErrConflict)
		}
		if session.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: cash session belongs to another device", domain.ErrConflict)
		}
		session.Status = domain.CashSessionClosed
		session.ClosedByEmployeeID = trimPtr(cmd.ClosedByEmployeeID)
		session.ClosingCashAmount = &cmd.ClosingCashAmount
		session.ClosedAt = &now
		session.UpdatedAt = now
		if err := s.repo.UpdateCashSessionClosed(ctx, session); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, session.RestaurantID, session.ShiftID, "CashSession", session.ID, "CashSessionClosed", session)
	})
	return session, err
}

func (s *Service) RecordCashDrawerEvent(ctx context.Context, cmd RecordCashDrawerEventCommand) (*domain.CashDrawerEvent, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CreatedByEmployeeID) == "" || cmd.Amount < 0 || !validCashDrawerEventType(cmd.EventType) {
		return nil, fmt.Errorf("%w: created_by_employee_id, event_type and non-negative amount are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var event *domain.CashDrawerEvent
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta); err != nil {
			return err
		}
		if cmd.Origin == domain.OriginEdgeDevice && cmd.ActorEmployeeID != cmd.CreatedByEmployeeID {
			return fmt.Errorf("%w: created_by_employee_id must match actor_employee_id", domain.ErrForbidden)
		}
		session, err := s.cashSessionForEvent(ctx, cmd)
		if err != nil {
			return err
		}
		event = &domain.CashDrawerEvent{
			ID:                    s.ids.NewID(),
			EdgeCashDrawerEventID: s.ids.NewID(),
			CashSessionID:         session.ID,
			RestaurantID:          session.RestaurantID,
			DeviceID:              session.DeviceID,
			ShiftID:               session.ShiftID,
			CreatedByEmployeeID:   cmd.CreatedByEmployeeID,
			EventType:             cmd.EventType,
			Amount:                cmd.Amount,
			Reason:                trimPtr(cmd.Reason),
			Note:                  trimPtr(cmd.Note),
			OccurredAt:            now,
			CreatedAt:             now,
		}
		if err := s.repo.CreateCashDrawerEvent(ctx, event); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, event.RestaurantID, event.ShiftID, "CashDrawerEvent", event.ID, "CashDrawerEventRecorded", event)
	})
	return event, err
}

func (s *Service) cashSessionForEvent(ctx context.Context, cmd RecordCashDrawerEventCommand) (*domain.CashSession, error) {
	var session *domain.CashSession
	var err error
	if strings.TrimSpace(cmd.CashSessionID) == "" {
		session, err = s.repo.GetOpenCashSessionByDevice(ctx, cmd.DeviceID)
	} else {
		session, err = s.repo.GetCashSession(ctx, cmd.CashSessionID)
	}
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: cannot record cash drawer event without active cash session", domain.ErrConflict)
		}
		return nil, err
	}
	if session.Status != domain.CashSessionOpen {
		return nil, fmt.Errorf("%w: cannot record cash drawer event without active cash session", domain.ErrConflict)
	}
	if session.DeviceID != cmd.DeviceID {
		return nil, fmt.Errorf("%w: cash session belongs to another device", domain.ErrConflict)
	}
	return session, nil
}

func validCashDrawerEventType(v domain.CashDrawerEventType) bool {
	switch v {
	case domain.CashDrawerCashIn, domain.CashDrawerCashOut, domain.CashDrawerNoSale, domain.CashDrawerCashCount:
		return true
	default:
		return false
	}
}

func trimPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
