package app

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

type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
}

type CommandMeta struct {
	CommandID string               `json:"command_id,omitempty"`
	DeviceID  string               `json:"device_id,omitempty"`
	Origin    domain.CommandOrigin `json:"origin,omitempty"`
}

const (
	OriginEdgeDevice = domain.OriginEdgeDevice
	OriginCloudSync  = domain.OriginCloudSync
	OriginSystemSeed = domain.OriginSystemSeed
)

type CreateRestaurantCommand struct {
	CommandMeta
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
	Currency string `json:"currency"`
}

type RegisterDeviceCommand struct {
	CommandMeta
	RestaurantID string `json:"restaurant_id"`
	DeviceCode   string `json:"device_code"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type CreateRoleCommand struct {
	CommandMeta
	Name            string `json:"name"`
	PermissionsJSON string `json:"permissions_json"`
}

type CreateEmployeeCommand struct {
	CommandMeta
	RestaurantID string `json:"restaurant_id"`
	RoleID       string `json:"role_id"`
	Name         string `json:"name"`
	PINHash      string `json:"pin_hash"`
}

type ArchiveEmployeeCommand struct {
	CommandMeta
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
}

type CreateCatalogItemCommand struct {
	CommandMeta
	Type     domain.CatalogItemType `json:"type"`
	Name     string                 `json:"name"`
	SKU      string                 `json:"sku"`
	BaseUnit string                 `json:"base_unit"`
}

type CreateMenuItemCommand struct {
	CommandMeta
	CatalogItemID string `json:"catalog_item_id"`
	Name          string `json:"name"`
	Price         int64  `json:"price"`
	Currency      string `json:"currency"`
}

type OpenShiftCommand struct {
	CommandMeta
	RestaurantID       string `json:"restaurant_id"`
	OpenedByEmployeeID string `json:"opened_by_employee_id"`
	OpeningCashAmount  int64  `json:"opening_cash_amount"`
}

type CloseShiftCommand struct {
	CommandMeta
	ID                 string `json:"id"`
	ClosedByEmployeeID string `json:"closed_by_employee_id"`
	ClosingCashAmount  int64  `json:"closing_cash_amount"`
}

type CreateOrderCommand struct {
	CommandMeta
	RestaurantID string `json:"restaurant_id"`
	ShiftID      string `json:"shift_id"`
	TableName    string `json:"table_name"`
	GuestCount   int    `json:"guest_count"`
}

type AddOrderLineCommand struct {
	CommandMeta
	OrderID    string `json:"order_id"`
	MenuItemID string `json:"menu_item_id"`
	Quantity   int64  `json:"quantity"`
}

type CreateCheckCommand struct {
	CommandMeta
	OrderID       string `json:"order_id"`
	DiscountTotal int64  `json:"discount_total"`
	TaxTotal      int64  `json:"tax_total"`
}

type CapturePaymentCommand struct {
	CommandMeta
	CheckID  string               `json:"check_id"`
	Method   domain.PaymentMethod `json:"method"`
	Amount   int64                `json:"amount"`
	Currency string               `json:"currency"`
}

type CloseOrderCommand struct {
	CommandMeta
	OrderID string `json:"order_id"`
}

func (s *Service) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	return s.repo.ListRestaurants(ctx)
}

func (s *Service) ListDevices(ctx context.Context) ([]domain.Device, error) {
	return s.repo.ListDevices(ctx)
}

func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	return s.repo.ListEmployees(ctx)
}

func (s *Service) ListCatalogItems(ctx context.Context) ([]domain.CatalogItem, error) {
	return s.repo.ListCatalogItems(ctx)
}

func (s *Service) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	return s.repo.ListMenuItems(ctx)
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

func (s *Service) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return s.repo.GetCheck(ctx, id)
}

func (s *Service) GetCurrentShift(ctx context.Context, deviceID string) (*domain.Shift, error) {
	if strings.TrimSpace(deviceID) == "" {
		return nil, fmt.Errorf("%w: device_id is required", domain.ErrInvalid)
	}
	return s.repo.GetOpenShiftByDevice(ctx, deviceID)
}

func (s *Service) CreateRestaurant(ctx context.Context, cmd CreateRestaurantCommand) (*domain.Restaurant, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.Timezone) == "" || strings.TrimSpace(cmd.Currency) == "" {
		return nil, fmt.Errorf("%w: name, timezone and currency are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Restaurant{ID: s.ids.NewID(), Name: cmd.Name, Timezone: cmd.Timezone, Currency: strings.ToUpper(cmd.Currency), Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateRestaurant(ctx, v); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, v.ID, "Restaurant", v.ID, "RestaurantCreated", v)
	})
}

func (s *Service) RegisterDevice(ctx context.Context, cmd RegisterDeviceCommand) (*domain.Device, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.DeviceCode) == "" || strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.Type) == "" {
		return nil, fmt.Errorf("%w: restaurant_id, device_code, name and type are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Device{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, DeviceCode: cmd.DeviceCode, Name: cmd.Name, Type: cmd.Type, Active: true, RegisteredAt: now, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateDevice(ctx, v); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, v.RestaurantID, "Device", v.ID, "DeviceRegistered", v)
	})
}

func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCommand) (*domain.Role, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	permissions := strings.TrimSpace(cmd.PermissionsJSON)
	if permissions == "" {
		permissions = "{}"
	}
	if strings.TrimSpace(cmd.Name) == "" || !json.Valid([]byte(permissions)) {
		return nil, fmt.Errorf("%w: name and valid permissions_json are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Role{ID: s.ids.NewID(), Name: cmd.Name, PermissionsJSON: permissions, Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateRole(ctx, v); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, "", "Role", v.ID, "RoleCreated", v)
	})
}

func (s *Service) CreateEmployee(ctx context.Context, cmd CreateEmployeeCommand) (*domain.Employee, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.RoleID) == "" || strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.PINHash) == "" {
		return nil, fmt.Errorf("%w: restaurant_id, role_id, name and pin_hash are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Employee{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, RoleID: cmd.RoleID, Name: cmd.Name, PINHash: cmd.PINHash, Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateEmployee(ctx, v); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, v.RestaurantID, "Employee", v.ID, "EmployeeCreated", v)
	})
}

func (s *Service) ArchiveEmployee(ctx context.Context, cmd ArchiveEmployeeCommand) error {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ID) == "" {
		return fmt.Errorf("%w: employee id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.ArchiveEmployee(ctx, cmd.ID, dbTime(now)); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, cmd.RestaurantID, "Employee", cmd.ID, "EmployeeArchived", cmd)
	})
}

func (s *Service) CreateCatalogItem(ctx context.Context, cmd CreateCatalogItemCommand) (*domain.CatalogItem, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if cmd.Type != domain.CatalogItemIngredient && cmd.Type != domain.CatalogItemDish && cmd.Type != domain.CatalogItemGood {
		return nil, fmt.Errorf("%w: unsupported catalog item type", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.SKU) == "" || strings.TrimSpace(cmd.BaseUnit) == "" {
		return nil, fmt.Errorf("%w: name, sku and base_unit are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.CatalogItem{ID: s.ids.NewID(), Type: cmd.Type, Name: cmd.Name, SKU: cmd.SKU, BaseUnit: cmd.BaseUnit, Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateCatalogItem(ctx, v); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, "", "CatalogItem", v.ID, "CatalogItemCreated", v)
	})
}

func (s *Service) CreateMenuItem(ctx context.Context, cmd CreateMenuItemCommand) (*domain.MenuItem, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CatalogItemID) == "" || strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.Currency) == "" || cmd.Price < 0 {
		return nil, fmt.Errorf("%w: catalog_item_id, name, currency and non-negative price are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.MenuItem{ID: s.ids.NewID(), CatalogItemID: cmd.CatalogItemID, Name: cmd.Name, Price: cmd.Price, Currency: strings.ToUpper(cmd.Currency), Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		catalogItem, err := s.repo.GetCatalogItem(ctx, cmd.CatalogItemID)
		if err != nil {
			return err
		}
		if !catalogItem.Active {
			return fmt.Errorf("%w: catalog item is archived", domain.ErrConflict)
		}
		if err := s.repo.CreateMenuItem(ctx, v); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, "", "MenuItem", v.ID, "MenuItemCreated", v)
	})
}

func (s *Service) OpenShift(ctx context.Context, cmd OpenShiftCommand) (*domain.Shift, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.DeviceID) == "" || strings.TrimSpace(cmd.OpenedByEmployeeID) == "" || cmd.OpeningCashAmount < 0 {
		return nil, fmt.Errorf("%w: restaurant_id, device_id, opened_by_employee_id and non-negative opening_cash_amount are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Shift{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, DeviceID: cmd.DeviceID, OpenedByEmployeeID: cmd.OpenedByEmployeeID, Status: domain.ShiftOpen, OpenedAt: now, OpeningCashAmount: cmd.OpeningCashAmount, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
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
		return s.outbox(ctx, cmd.CommandMeta, v.RestaurantID, "Shift", v.ID, "ShiftOpened", v)
	})
}

func (s *Service) CloseShift(ctx context.Context, cmd CloseShiftCommand) (*domain.Shift, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.ClosedByEmployeeID) == "" || cmd.ClosingCashAmount < 0 {
		return nil, fmt.Errorf("%w: id, closed_by_employee_id and non-negative closing_cash_amount are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var shift *domain.Shift
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
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
		return s.outbox(ctx, cmd.CommandMeta, shift.RestaurantID, "Shift", shift.ID, "ShiftClosed", shift)
	})
	return shift, err
}

func (s *Service) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if cmd.GuestCount < 0 {
		return nil, fmt.Errorf("%w: guest_count must be non-negative", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var order *domain.Order
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
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
		return s.outbox(ctx, cmd.CommandMeta, order.RestaurantID, "Order", order.ID, "OrderCreated", order)
	})
	return order, err
}

func (s *Service) AddOrderLine(ctx context.Context, cmd AddOrderLineCommand) (*domain.OrderLine, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" || strings.TrimSpace(cmd.MenuItemID) == "" || cmd.Quantity <= 0 {
		return nil, fmt.Errorf("%w: order_id, menu_item_id and positive quantity are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var line *domain.OrderLine
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderOpen {
			return fmt.Errorf("%w: cannot add line to closed order", domain.ErrConflict)
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
		return s.outbox(ctx, cmd.CommandMeta, order.RestaurantID, "Order", order.ID, "OrderLineAdded", line)
	})
	return line, err
}

func (s *Service) CreateCheck(ctx context.Context, cmd CreateCheckCommand) (*domain.Check, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" || cmd.DiscountTotal < 0 || cmd.TaxTotal < 0 {
		return nil, fmt.Errorf("%w: order_id, non-negative discount_total and tax_total are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var check *domain.Check
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderOpen {
			return fmt.Errorf("%w: cannot create check for closed order", domain.ErrConflict)
		}
		if _, err := s.repo.GetCheckByOrder(ctx, order.ID); err == nil {
			return fmt.Errorf("%w: order already has a check", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		lines, err := s.repo.ListOrderLines(ctx, order.ID)
		if err != nil {
			return err
		}
		var subtotal int64
		for _, line := range lines {
			if line.Status == domain.OrderLineActive {
				subtotal += line.TotalPrice
			}
		}
		total := subtotal - cmd.DiscountTotal + cmd.TaxTotal
		if total < 0 {
			return fmt.Errorf("%w: check total cannot be negative", domain.ErrInvalid)
		}
		check = &domain.Check{ID: s.ids.NewID(), OrderID: order.ID, Status: domain.CheckOpen, Subtotal: subtotal, DiscountTotal: cmd.DiscountTotal, TaxTotal: cmd.TaxTotal, Total: total, PaidTotal: 0, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.CreateCheck(ctx, check); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, order.RestaurantID, "Check", check.ID, "CheckCreated", check)
	})
	return check, err
}

func (s *Service) CapturePayment(ctx context.Context, cmd CapturePaymentCommand) (*domain.Payment, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CheckID) == "" || cmd.Amount <= 0 || strings.TrimSpace(cmd.Currency) == "" {
		return nil, fmt.Errorf("%w: check_id, positive amount and currency are required", domain.ErrInvalid)
	}
	if cmd.Method != domain.PaymentCash && cmd.Method != domain.PaymentCard && cmd.Method != domain.PaymentOther {
		return nil, fmt.Errorf("%w: unsupported payment method", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var payment *domain.Payment
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
			return err
		}
		check, err := s.repo.GetCheck(ctx, cmd.CheckID)
		if err != nil {
			return err
		}
		if check.Status != domain.CheckOpen && check.Status != domain.CheckPaid {
			return fmt.Errorf("%w: check cannot accept payments", domain.ErrConflict)
		}
		if check.PaidTotal+cmd.Amount > check.Total {
			return fmt.Errorf("%w: check overpayment is not allowed", domain.ErrConflict)
		}
		order, err := s.repo.GetOrder(ctx, check.OrderID)
		if err != nil {
			return err
		}
		payment = &domain.Payment{ID: s.ids.NewID(), CheckID: check.ID, Method: cmd.Method, Amount: cmd.Amount, Currency: strings.ToUpper(cmd.Currency), Status: domain.PaymentCaptured, CreatedAt: now, UpdatedAt: now}
		if err := s.repo.CreatePayment(ctx, payment); err != nil {
			return err
		}
		check.PaidTotal += cmd.Amount
		if check.PaidTotal == check.Total {
			check.Status = domain.CheckPaid
		}
		check.UpdatedAt = now
		if err := s.repo.UpdateCheckPaidTotal(ctx, check); err != nil {
			return err
		}
		return s.outbox(ctx, cmd.CommandMeta, order.RestaurantID, "Payment", payment.ID, "PaymentCaptured", payment)
	})
	return payment, err
}

func (s *Service) CloseOrder(ctx context.Context, cmd CloseOrderCommand) (*domain.Order, error) {
	if err := validateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var order *domain.Order
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensureCommandNotProcessed(ctx, cmd.CommandID); err != nil {
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
		return s.outbox(ctx, cmd.CommandMeta, order.RestaurantID, "Order", order.ID, "OrderClosed", order)
	})
	return order, err
}

func (s *Service) ListOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	return s.repo.ListOutbox(ctx, limit)
}

func (s *Service) MarkOutboxSent(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: outbox id is required", domain.ErrInvalid)
	}
	return s.repo.MarkOutboxSent(ctx, id, dbTime(s.clock.Now()))
}

func (s *Service) MarkOutboxFailed(ctx context.Context, id, reason string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: outbox id and error are required", domain.ErrInvalid)
	}
	return s.repo.MarkOutboxFailed(ctx, id, reason, dbTime(s.clock.Now()))
}

func (s *Service) outbox(ctx context.Context, meta CommandMeta, restaurantID, aggregateType, aggregateID, commandType string, payload any) error {
	commandID := strings.TrimSpace(meta.CommandID)
	if commandID == "" {
		commandID = s.ids.NewID()
	}
	body, err := json.Marshal(struct {
		Origin domain.CommandOrigin `json:"origin"`
		Data   any                  `json:"data"`
	}{
		Origin: meta.Origin,
		Data:   payload,
	})
	if err != nil {
		return err
	}
	now := s.clock.Now()
	msg := &domain.OutboxMessage{
		ID:            s.ids.NewID(),
		CommandID:     commandID,
		Origin:        meta.Origin,
		RestaurantID:  optionalID(restaurantID),
		DeviceID:      strings.TrimSpace(meta.DeviceID),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		CommandType:   commandType,
		PayloadJSON:   string(body),
		Status:        domain.OutboxPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return s.repo.CreateOutboxMessage(ctx, msg)
}

func validateWriteMeta(meta CommandMeta) error {
	if strings.TrimSpace(meta.DeviceID) == "" {
		return fmt.Errorf("%w: device_id is required", domain.ErrInvalid)
	}
	switch meta.Origin {
	case domain.OriginEdgeDevice, domain.OriginCloudSync, domain.OriginSystemSeed:
		return nil
	default:
		return fmt.Errorf("%w: valid origin is required", domain.ErrInvalid)
	}
}

func (s *Service) ensureCommandNotProcessed(ctx context.Context, commandID string) error {
	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return nil
	}
	if _, err := s.repo.GetOutboxByCommandID(ctx, commandID); err == nil {
		return fmt.Errorf("%w: %s", domain.ErrDuplicateCommand, commandID)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	return nil
}

func optionalID(id string) *string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	return &id
}

func dbTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
