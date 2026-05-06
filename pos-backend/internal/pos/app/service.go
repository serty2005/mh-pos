package app

import (
	"context"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	appcash "pos-backend/internal/pos/app/cash"
	appcatalog "pos-backend/internal/pos/app/catalog"
	appcheck "pos-backend/internal/pos/app/check"
	appdevice "pos-backend/internal/pos/app/device"
	appemployee "pos-backend/internal/pos/app/employee"
	appmenu "pos-backend/internal/pos/app/menu"
	apporder "pos-backend/internal/pos/app/order"
	appprecheck "pos-backend/internal/pos/app/precheck"
	apprestaurant "pos-backend/internal/pos/app/restaurant"
	"pos-backend/internal/pos/app/shared"
	appshift "pos-backend/internal/pos/app/shift"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type CommandMeta = shared.CommandMeta

const (
	OriginEdgeDevice = shared.OriginEdgeDevice
	OriginCloudSync  = shared.OriginCloudSync
	OriginSystemSeed = shared.OriginSystemSeed
)

type CreateRestaurantCommand = apprestaurant.CreateRestaurantCommand
type RegisterDeviceCommand = appdevice.RegisterDeviceCommand
type CreateRoleCommand = appemployee.CreateRoleCommand
type CreateEmployeeCommand = appemployee.CreateEmployeeCommand
type ArchiveEmployeeCommand = appemployee.ArchiveEmployeeCommand
type CreateCatalogItemCommand = appcatalog.CreateCatalogItemCommand
type CreateMenuItemCommand = appmenu.CreateMenuItemCommand
type OpenShiftCommand = appshift.OpenShiftCommand
type CloseShiftCommand = appshift.CloseShiftCommand
type CreateOrderCommand = apporder.CreateOrderCommand
type AddOrderLineCommand = apporder.AddOrderLineCommand
type IssuePrecheckCommand = appprecheck.IssuePrecheckCommand
type CancelPrecheckCommand = appprecheck.CancelPrecheckCommand
type CreateCheckCommand = appcheck.CreateCheckCommand
type CapturePaymentCommand = appcheck.CapturePaymentCommand
type CloseOrderCommand = apporder.CloseOrderCommand
type OpenCashSessionCommand = appcash.OpenCashSessionCommand
type CloseCashSessionCommand = appcash.CloseCashSessionCommand
type RecordCashDrawerEventCommand = appcash.RecordCashDrawerEventCommand

type ListLocalEventsQuery struct {
	Limit     int
	EventType string
}

type Service struct {
	restaurants *apprestaurant.Service
	devices     *appdevice.Service
	employees   *appemployee.Service
	catalog     *appcatalog.Service
	menu        *appmenu.Service
	shifts      *appshift.Service
	orders      *apporder.Service
	prechecks   *appprecheck.Service
	checks      *appcheck.Service
	cash        *appcash.Service
	localEvents ports.LocalEventRepository
	outbox      *shared.OutboxService
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{
		restaurants: apprestaurant.NewService(repo, tx, ids, clock),
		devices:     appdevice.NewService(repo, tx, ids, clock),
		employees:   appemployee.NewService(repo, tx, ids, clock),
		catalog:     appcatalog.NewService(repo, tx, ids, clock),
		menu:        appmenu.NewService(repo, tx, ids, clock),
		shifts:      appshift.NewService(repo, tx, ids, clock),
		orders:      apporder.NewService(repo, tx, ids, clock),
		prechecks:   appprecheck.NewService(repo, tx, ids, clock),
		checks:      appcheck.NewService(repo, tx, ids, clock),
		cash:        appcash.NewService(repo, tx, ids, clock),
		localEvents: repo,
		outbox:      shared.NewOutboxService(repo, clock),
	}
}

func (s *Service) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	return s.restaurants.ListRestaurants(ctx)
}

func (s *Service) CreateRestaurant(ctx context.Context, cmd CreateRestaurantCommand) (*domain.Restaurant, error) {
	return s.restaurants.CreateRestaurant(ctx, cmd)
}

func (s *Service) ListDevices(ctx context.Context) ([]domain.Device, error) {
	return s.devices.ListDevices(ctx)
}

func (s *Service) RegisterDevice(ctx context.Context, cmd RegisterDeviceCommand) (*domain.Device, error) {
	return s.devices.RegisterDevice(ctx, cmd)
}

func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.employees.ListRoles(ctx)
}

func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCommand) (*domain.Role, error) {
	return s.employees.CreateRole(ctx, cmd)
}

func (s *Service) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	return s.employees.ListEmployees(ctx)
}

func (s *Service) CreateEmployee(ctx context.Context, cmd CreateEmployeeCommand) (*domain.Employee, error) {
	return s.employees.CreateEmployee(ctx, cmd)
}

func (s *Service) ArchiveEmployee(ctx context.Context, cmd ArchiveEmployeeCommand) error {
	return s.employees.ArchiveEmployee(ctx, cmd)
}

func (s *Service) ListCatalogItems(ctx context.Context) ([]domain.CatalogItem, error) {
	return s.catalog.ListCatalogItems(ctx)
}

func (s *Service) CreateCatalogItem(ctx context.Context, cmd CreateCatalogItemCommand) (*domain.CatalogItem, error) {
	return s.catalog.CreateCatalogItem(ctx, cmd)
}

func (s *Service) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	return s.menu.ListMenuItems(ctx)
}

func (s *Service) CreateMenuItem(ctx context.Context, cmd CreateMenuItemCommand) (*domain.MenuItem, error) {
	return s.menu.CreateMenuItem(ctx, cmd)
}

func (s *Service) GetCurrentShift(ctx context.Context, deviceID string) (*domain.Shift, error) {
	return s.shifts.GetCurrentShift(ctx, deviceID)
}

func (s *Service) OpenShift(ctx context.Context, cmd OpenShiftCommand) (*domain.Shift, error) {
	return s.shifts.OpenShift(ctx, cmd)
}

func (s *Service) CloseShift(ctx context.Context, cmd CloseShiftCommand) (*domain.Shift, error) {
	return s.shifts.CloseShift(ctx, cmd)
}

func (s *Service) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return s.orders.GetOrder(ctx, id)
}

func (s *Service) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error) {
	return s.orders.CreateOrder(ctx, cmd)
}

func (s *Service) AddOrderLine(ctx context.Context, cmd AddOrderLineCommand) (*domain.OrderLine, error) {
	return s.orders.AddOrderLine(ctx, cmd)
}

func (s *Service) CloseOrder(ctx context.Context, cmd CloseOrderCommand) (*domain.Order, error) {
	return s.orders.CloseOrder(ctx, cmd)
}

func (s *Service) IssuePrecheck(ctx context.Context, cmd IssuePrecheckCommand) (*domain.Precheck, error) {
	return s.prechecks.IssuePrecheck(ctx, cmd)
}

func (s *Service) GetPrecheck(ctx context.Context, id string) (*domain.Precheck, error) {
	return s.prechecks.GetPrecheck(ctx, id)
}

func (s *Service) ListPrechecksByOrder(ctx context.Context, orderID string) ([]domain.Precheck, error) {
	return s.prechecks.ListPrechecksByOrder(ctx, orderID)
}

func (s *Service) CancelPrecheck(ctx context.Context, cmd CancelPrecheckCommand) (*domain.Precheck, error) {
	return s.prechecks.CancelPrecheck(ctx, cmd)
}

func (s *Service) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return s.checks.GetCheck(ctx, id)
}

func (s *Service) CreateCheck(ctx context.Context, cmd CreateCheckCommand) (*domain.Check, error) {
	return s.checks.CreateCheck(ctx, cmd)
}

func (s *Service) CapturePayment(ctx context.Context, cmd CapturePaymentCommand) (*domain.Payment, error) {
	return s.checks.CapturePayment(ctx, cmd)
}

func (s *Service) GetCurrentCashSession(ctx context.Context, deviceID string) (*domain.CashSession, error) {
	return s.cash.GetCurrentCashSession(ctx, deviceID)
}

func (s *Service) OpenCashSession(ctx context.Context, cmd OpenCashSessionCommand) (*domain.CashSession, error) {
	return s.cash.OpenCashSession(ctx, cmd)
}

func (s *Service) CloseCashSession(ctx context.Context, cmd CloseCashSessionCommand) (*domain.CashSession, error) {
	return s.cash.CloseCashSession(ctx, cmd)
}

func (s *Service) RecordCashDrawerEvent(ctx context.Context, cmd RecordCashDrawerEventCommand) (*domain.CashDrawerEvent, error) {
	return s.cash.RecordCashDrawerEvent(ctx, cmd)
}

func (s *Service) ListOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	return s.outbox.ListOutbox(ctx, limit)
}

func (s *Service) ListLocalEvents(ctx context.Context, query ListLocalEventsQuery) ([]domain.LocalEvent, error) {
	return s.localEvents.ListLocalEvents(ctx, query.Limit, query.EventType)
}

func (s *Service) MarkOutboxSent(ctx context.Context, id string) error {
	return s.outbox.MarkOutboxSent(ctx, id)
}

func (s *Service) MarkOutboxFailed(ctx context.Context, id, reason string) error {
	return s.outbox.MarkOutboxFailed(ctx, id, reason)
}
