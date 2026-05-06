package precheck

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
	domainprecheck "pos-backend/internal/pos/domain/precheck"
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

type IssuePrecheckCommand struct {
	shared.CommandMeta
	OrderID string `json:"order_id"`
}

type CancelPrecheckCommand struct {
	shared.CommandMeta
	PrecheckID         string `json:"precheck_id"`
	ManagerEmployeeID  string `json:"manager_employee_id,omitempty"`
	CancellationReason string `json:"cancellation_reason,omitempty"`
}

func (s *Service) GetPrecheck(ctx context.Context, id string) (*domain.Precheck, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: precheck id is required", domain.ErrInvalid)
	}
	return s.repo.GetPrecheck(ctx, id)
}

func (s *Service) ListPrechecksByOrder(ctx context.Context, orderID string) ([]domain.Precheck, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	return s.repo.ListPrechecksByOrder(ctx, orderID)
}

func (s *Service) IssuePrecheck(ctx context.Context, cmd IssuePrecheckCommand) (*domain.Precheck, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var precheck *domain.Precheck
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderOpen {
			return fmt.Errorf("%w: cannot issue precheck for closed order", domain.ErrConflict)
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: precheck device does not match order device", domain.ErrConflict)
		}
		if _, err := s.repo.GetActivePrecheckByOrder(ctx, order.ID); err == nil {
			return fmt.Errorf("%w: order already has active precheck", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		lines, err := s.repo.ListOrderLines(ctx, order.ID)
		if err != nil {
			return err
		}
		existing, err := s.repo.ListPrechecksByOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		version := 1
		var supersedesPrecheckID *string
		for _, item := range existing {
			if item.Version >= version {
				version = item.Version + 1
				id := item.ID
				supersedesPrecheckID = &id
			}
		}
		var subtotal int64
		for _, line := range lines {
			if line.Status == domain.OrderLineActive {
				subtotal += line.TotalPrice
			}
		}
		precheck, err = domainprecheck.NewIssuedVersion(s.ids.NewID(), order.ID, version, supersedesPrecheckID, subtotal, 0, 0, now)
		if err != nil {
			return err
		}
		if err := s.repo.CreatePrecheck(ctx, precheck); err != nil {
			return err
		}
		order.Status = domain.OrderLocked
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderLocked(ctx, order); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Precheck", precheck.ID, "PrecheckIssued", precheck)
	})
	return precheck, err
}

func (s *Service) CancelPrecheck(ctx context.Context, cmd CancelPrecheckCommand) (*domain.Precheck, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.PrecheckID) == "" {
		return nil, fmt.Errorf("%w: precheck_id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var precheck *domain.Precheck
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		var err error
		precheck, err = s.repo.GetPrecheck(ctx, cmd.PrecheckID)
		if err != nil {
			return err
		}
		if err := precheck.Cancel(now, cmd.ManagerEmployeeID, cmd.CancellationReason); err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, precheck.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderLocked {
			return fmt.Errorf("%w: precheck cancellation expects locked order", domain.ErrConflict)
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: precheck device does not match order device", domain.ErrConflict)
		}
		active, err := s.repo.GetActivePrecheckByOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		if active.ID != precheck.ID {
			return fmt.Errorf("%w: precheck is not active for order", domain.ErrConflict)
		}
		if err := s.repo.UpdatePrecheckLifecycle(ctx, precheck); err != nil {
			return err
		}
		order.Status = domain.OrderOpen
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderOpen(ctx, order); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Precheck", precheck.ID, "PrecheckCancelled", precheck)
	})
	return precheck, err
}
