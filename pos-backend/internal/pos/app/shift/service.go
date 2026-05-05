package shift

import (
	"context"
	"errors"
	"fmt"
	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
	"strings"
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

type OpenShiftCommand struct {
	shared.CommandMeta
	RestaurantID       string `json:"restaurant_id"`
	OpenedByEmployeeID string `json:"opened_by_employee_id"`
	OpeningCashAmount  int64  `json:"opening_cash_amount"`
}

type CloseShiftCommand struct {
	shared.CommandMeta
	ID                 string `json:"id"`
	ClosedByEmployeeID string `json:"closed_by_employee_id"`
	ClosingCashAmount  int64  `json:"closing_cash_amount"`
}

func (s *Service) GetCurrentShift(ctx context.Context, deviceID string) (*domain.Shift, error) {
	if strings.TrimSpace(deviceID) == "" {
		return nil, fmt.Errorf("%w: device_id is required", domain.ErrInvalid)
	}
	return s.repo.GetOpenShiftByDevice(ctx, deviceID)
}

func (s *Service) OpenShift(ctx context.Context, cmd OpenShiftCommand) (*domain.Shift, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.DeviceID) == "" || strings.TrimSpace(cmd.OpenedByEmployeeID) == "" || cmd.OpeningCashAmount < 0 {
		return nil, fmt.Errorf("%w: restaurant_id, device_id, opened_by_employee_id and non-negative opening_cash_amount are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Shift{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, DeviceID: cmd.DeviceID, OpenedByEmployeeID: cmd.OpenedByEmployeeID, Status: domain.ShiftOpen, OpenedAt: now, OpeningCashAmount: cmd.OpeningCashAmount, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := s.repo.GetOpenShiftByDevice(ctx, cmd.DeviceID); err == nil {
			return fmt.Errorf("%w: device already has an open shift", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if err := s.repo.CreateShift(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, v.RestaurantID, v.ID, "Shift", v.ID, "ShiftOpened", v)
	})
}

func (s *Service) CloseShift(ctx context.Context, cmd CloseShiftCommand) (*domain.Shift, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.ClosedByEmployeeID) == "" || cmd.ClosingCashAmount < 0 {
		return nil, fmt.Errorf("%w: id, closed_by_employee_id and non-negative closing_cash_amount are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var shift *domain.Shift
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		var err error
		shift, err = s.repo.GetShift(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if shift.Status != domain.ShiftOpen {
			return fmt.Errorf("%w: shift is not open", domain.ErrConflict)
		}
		hasOpenOrders, err := s.repo.HasOpenOrdersForShift(ctx, shift.ID)
		if err != nil {
			return err
		}
		if hasOpenOrders {
			return fmt.Errorf("%w: shift has open orders", domain.ErrConflict)
		}
		shift.Status = domain.ShiftClosed
		shift.ClosedByEmployeeID = &cmd.ClosedByEmployeeID
		shift.ClosedAt = &now
		shift.ClosingCashAmount = &cmd.ClosingCashAmount
		shift.UpdatedAt = now
		if err := s.repo.UpdateShiftClosed(ctx, shift); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, shift.RestaurantID, shift.ID, "Shift", shift.ID, "ShiftClosed", shift)
	})
	return shift, err
}
