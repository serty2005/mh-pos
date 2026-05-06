package order

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

type CreateOrderCommand struct {
	shared.CommandMeta
	RestaurantID string `json:"restaurant_id"`
	ShiftID      string `json:"shift_id"`
	TableName    string `json:"table_name"`
	GuestCount   int    `json:"guest_count"`
}

type AddOrderLineCommand struct {
	shared.CommandMeta
	OrderID    string `json:"order_id"`
	MenuItemID string `json:"menu_item_id"`
	Quantity   int64  `json:"quantity"`
}

type CloseOrderCommand struct {
	shared.CommandMeta
	OrderID string `json:"order_id"`
}

func (s *Service) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	lines, err := s.repo.ListOrderLines(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Lines = lines
	check, err := s.repo.GetCheckByOrder(ctx, id)
	if err == nil {
		order.Check = check
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	return order, nil
}

func (s *Service) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if cmd.GuestCount < 0 {
		return nil, fmt.Errorf("%w: guest_count must be non-negative", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var order *domain.Order
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		var shift *domain.Shift
		var err error
		if strings.TrimSpace(cmd.ShiftID) != "" {
			shift, err = s.repo.GetShift(ctx, cmd.ShiftID)
		} else {
			shift, err = s.repo.GetOpenShiftByDevice(ctx, cmd.DeviceID)
		}
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: cannot create order without an open shift", domain.ErrConflict)
			}
			return err
		}
		if shift.Status != domain.ShiftOpen || shift.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: cannot create order without an open shift on device", domain.ErrConflict)
		}
		if restaurantID := strings.TrimSpace(cmd.RestaurantID); restaurantID != "" && restaurantID != shift.RestaurantID {
			return fmt.Errorf("%w: restaurant_id does not match open shift", domain.ErrConflict)
		}
		order = &domain.Order{ID: s.ids.NewID(), EdgeOrderID: s.ids.NewID(), RestaurantID: shift.RestaurantID, DeviceID: cmd.DeviceID, ShiftID: shift.ID, Status: domain.OrderOpen, TableName: cmd.TableName, GuestCount: cmd.GuestCount, OpenedAt: now, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.CreateOrder(ctx, order); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderCreated", order)
	})
	return order, err
}

func (s *Service) AddOrderLine(ctx context.Context, cmd AddOrderLineCommand) (*domain.OrderLine, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" || strings.TrimSpace(cmd.MenuItemID) == "" || cmd.Quantity <= 0 {
		return nil, fmt.Errorf("%w: order_id, menu_item_id and positive quantity are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var line *domain.OrderLine
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderOpen {
			return fmt.Errorf("%w: cannot add line to non-open order", domain.ErrConflict)
		}
		if _, err := s.repo.GetActivePrecheckByOrder(ctx, order.ID); err == nil {
			return fmt.Errorf("%w: cannot change order with active precheck", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		menuItem, err := s.repo.GetMenuItem(ctx, cmd.MenuItemID)
		if err != nil {
			return err
		}
		if !menuItem.Active {
			return fmt.Errorf("%w: menu item is archived", domain.ErrConflict)
		}
		line = &domain.OrderLine{ID: s.ids.NewID(), OrderID: order.ID, MenuItemID: menuItem.ID, CatalogItemID: menuItem.CatalogItemID, Name: menuItem.Name, Quantity: cmd.Quantity, UnitPrice: menuItem.Price, TotalPrice: menuItem.Price * cmd.Quantity, Status: domain.OrderLineActive, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.CreateOrderLine(ctx, line); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderLineAdded", line)
	})
	return line, err
}

func (s *Service) CloseOrder(ctx context.Context, cmd CloseOrderCommand) (*domain.Order, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var order *domain.Order
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		var err error
		order, err = s.repo.GetOrder(ctx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderOpen {
			return fmt.Errorf("%w: order is not open", domain.ErrConflict)
		}
		check, err := s.repo.GetCheckByOrder(ctx, order.ID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: cannot close order without check", domain.ErrConflict)
			}
			return err
		}
		if check.PaidTotal != check.Total {
			return fmt.Errorf("%w: cannot close order without full payment", domain.ErrConflict)
		}
		order.Status = domain.OrderClosed
		order.ClosedAt = &now
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderClosed(ctx, order); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderClosed", order)
	})
	return order, err
}
