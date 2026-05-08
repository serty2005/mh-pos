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
	OpeningCashAmount  int64  `json:"-"`
}

// ListRecentShiftsCommand запрашивает последние личные смены текущего сотрудника.
type ListRecentShiftsCommand struct {
	shared.CommandMeta
	RestaurantID string `json:"restaurant_id,omitempty"`
	EmployeeID   string `json:"employee_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type CloseShiftCommand struct {
	shared.CommandMeta
	ID                 string `json:"id"`
	ClosedByEmployeeID string `json:"closed_by_employee_id"`
	ClosingCashAmount  int64  `json:"-"`
}

func (s *Service) GetCurrentShift(ctx context.Context, meta shared.CommandMeta) (*domain.Shift, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionEmployeeShiftViewCurrent))
	if err != nil {
		return nil, err
	}
	return s.repo.GetOpenShiftByEmployee(ctx, operator.Employee.RestaurantID, operator.Employee.ID)
}

func (s *Service) ListRecentShifts(ctx context.Context, cmd ListRecentShiftsCommand) ([]domain.Shift, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionEmployeeShiftRecent))
	if err != nil {
		return nil, err
	}
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	if restaurantID == "" {
		restaurantID = operator.Employee.RestaurantID
	}
	employeeID := strings.TrimSpace(cmd.EmployeeID)
	if employeeID == "" {
		employeeID = operator.Employee.ID
	}
	if restaurantID != operator.Employee.RestaurantID || employeeID != operator.Employee.ID {
		return nil, fmt.Errorf("%w: последние личные смены доступны только текущему сотруднику", domain.ErrForbidden)
	}
	limit := cmd.Limit
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	return s.repo.ListRecentShiftsByEmployee(ctx, restaurantID, employeeID, limit)
}

func (s *Service) OpenShift(ctx context.Context, cmd OpenShiftCommand) (*domain.Shift, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.DeviceID) == "" || strings.TrimSpace(cmd.OpenedByEmployeeID) == "" {
		return nil, fmt.Errorf("%w: restaurant_id, device_id и opened_by_employee_id обязательны", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Shift{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, DeviceID: cmd.DeviceID, OpenedByEmployeeID: cmd.OpenedByEmployeeID, Status: domain.ShiftOpen, OpenedAt: now, OpeningCashAmount: cmd.OpeningCashAmount, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionEmployeeShiftOpen)); err != nil {
			return err
		}
		if cmd.Origin == domain.OriginEdgeDevice && cmd.ActorEmployeeID != cmd.OpenedByEmployeeID {
			return fmt.Errorf("%w: opened_by_employee_id must match actor_employee_id", domain.ErrForbidden)
		}
		if _, err := s.repo.GetOpenShiftByEmployee(ctx, cmd.RestaurantID, cmd.OpenedByEmployeeID); err == nil {
			return fmt.Errorf("%w: у сотрудника уже есть открытая личная смена", domain.ErrConflict)
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
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.ClosedByEmployeeID) == "" {
		return nil, fmt.Errorf("%w: id и closed_by_employee_id обязательны", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var shift *domain.Shift
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionEmployeeShiftClose)); err != nil {
			return err
		}
		if cmd.Origin == domain.OriginEdgeDevice && cmd.ActorEmployeeID != cmd.ClosedByEmployeeID {
			return fmt.Errorf("%w: closed_by_employee_id must match actor_employee_id", domain.ErrForbidden)
		}
		var err error
		shift, err = s.repo.GetShift(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if shift.Status != domain.ShiftOpen {
			return fmt.Errorf("%w: shift is not open", domain.ErrConflict)
		}
		if cmd.Origin == domain.OriginEdgeDevice && shift.OpenedByEmployeeID != cmd.ActorEmployeeID {
			return fmt.Errorf("%w: личная смена принадлежит другому сотруднику", domain.ErrForbidden)
		}
		hasOpenOrders, err := s.repo.HasOpenOrdersForShift(ctx, shift.ID)
		if err != nil {
			return err
		}
		if hasOpenOrders {
			return fmt.Errorf("%w: shift has open orders", domain.ErrConflict)
		}
		if _, err := s.repo.GetOpenCashSessionByDevice(ctx, shift.DeviceID); err == nil {
			return fmt.Errorf("%w: у личной смены есть открытая кассовая смена", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
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
