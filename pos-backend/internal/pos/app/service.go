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
	appauth "pos-backend/internal/pos/app/auth"
	appcash "pos-backend/internal/pos/app/cash"
	appcatalog "pos-backend/internal/pos/app/catalog"
	appcheck "pos-backend/internal/pos/app/check"
	appdevice "pos-backend/internal/pos/app/device"
	appemployee "pos-backend/internal/pos/app/employee"
	appfloor "pos-backend/internal/pos/app/floor"
	appinventory "pos-backend/internal/pos/app/inventory"
	appmastersync "pos-backend/internal/pos/app/mastersync"
	appmenu "pos-backend/internal/pos/app/menu"
	apporder "pos-backend/internal/pos/app/order"
	appprecheck "pos-backend/internal/pos/app/precheck"
	apppricing "pos-backend/internal/pos/app/pricing"
	appprovisioning "pos-backend/internal/pos/app/provisioning"
	apprestaurant "pos-backend/internal/pos/app/restaurant"
	"pos-backend/internal/pos/app/shared"
	appshift "pos-backend/internal/pos/app/shift"
	appstorage "pos-backend/internal/pos/app/storage"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type CommandMeta = shared.CommandMeta

var NormalizeDeviceMeta = shared.NormalizeDeviceMeta

const (
	OriginEdgeDevice = shared.OriginEdgeDevice
	OriginCloudSync  = shared.OriginCloudSync
	OriginSystemSeed = shared.OriginSystemSeed
)

type CreateRestaurantCommand = apprestaurant.CreateRestaurantCommand
type RegisterDeviceCommand = appdevice.RegisterDeviceCommand
type PairEdgeNodeCommand = appdevice.PairEdgeNodeCommand
type CreateRoleCommand = appemployee.CreateRoleCommand
type CreateEmployeeCommand = appemployee.CreateEmployeeCommand
type ArchiveEmployeeCommand = appemployee.ArchiveEmployeeCommand
type PinLoginCommand = appauth.PinLoginCommand
type LogoutCommand = appauth.LogoutCommand
type CreateHallCommand = appfloor.CreateHallCommand
type ArchiveHallCommand = appfloor.ArchiveHallCommand
type CreateTableCommand = appfloor.CreateTableCommand
type ArchiveTableCommand = appfloor.ArchiveTableCommand
type CreateCatalogItemCommand = appcatalog.CreateCatalogItemCommand
type CreateMenuItemCommand = appmenu.CreateMenuItemCommand
type OpenShiftCommand = appshift.OpenShiftCommand
type ListRecentShiftsCommand = appshift.ListRecentShiftsCommand
type CloseShiftCommand = appshift.CloseShiftCommand
type CreateOrderCommand = apporder.CreateOrderCommand
type AddOrderLineCommand = apporder.AddOrderLineCommand
type SelectedModifierCommand = apporder.SelectedModifierCommand
type ChangeOrderLineQuantityCommand = apporder.ChangeOrderLineQuantityCommand
type UpdateOrderLineModifiersCommand = apporder.UpdateOrderLineModifiersCommand
type VoidOrderLineCommand = apporder.VoidOrderLineCommand
type UpdateOrderLineDetailsCommand = apporder.UpdateOrderLineDetailsCommand
type IssuePrecheckCommand = appprecheck.IssuePrecheckCommand
type CancelPrecheckCommand = appprecheck.CancelPrecheckCommand
type ReprintPrecheckCommand = appprecheck.ReprintPrecheckCommand
type AddDiscountCommand = apppricing.AddDiscountCommand
type AddSurchargeCommand = apppricing.AddSurchargeCommand
type CapturePaymentCommand = appcheck.CapturePaymentCommand
type RefundPaymentCommand = appcheck.RefundPaymentCommand
type ListClosedOrdersCommand = appcheck.ListClosedOrdersCommand
type ListFinancialOperationsCommand = appcheck.ListFinancialOperationsCommand
type ReprintCheckCommand = appcheck.ReprintCheckCommand
type FinancialOperationItemCommand = appcheck.FinancialOperationItemCommand
type RecordCheckCancellationCommand = appcheck.RecordCheckCancellationCommand
type RecordCheckRefundCommand = appcheck.RecordCheckRefundCommand
type CreateManualStockDocumentCommand = appinventory.CreateManualStockDocumentCommand
type CreateStockMoveCommand = appinventory.CreateStockMoveCommand
type CloseOrderCommand = apporder.CloseOrderCommand
type OpenCashSessionCommand = appcash.OpenCashSessionCommand
type CloseCashSessionCommand = appcash.CloseCashSessionCommand
type RecordCashDrawerEventCommand = appcash.RecordCashDrawerEventCommand
type ApplyMasterDataCommand = appmastersync.ApplyMasterDataCommand
type ApplyMasterDataResult = appmastersync.ApplyMasterDataResult
type RegisterCloudProvisioningCommand = appprovisioning.RegisterCloudCommand
type PairViaLicenseCommand = appprovisioning.PairViaLicenseCommand
type StorageStatusCommand = appstorage.StorageStatusCommand
type RetentionDryRunCommand = appstorage.RetentionDryRunCommand
type ArchiveExportPlanCommand = appstorage.ArchiveExportPlanCommand
type ArchiveExportCommand = appstorage.ArchiveExportCommand
type ArchiveApplyPlanCommand = appstorage.ArchiveApplyPlanCommand
type ArchiveReadPlanCommand = appstorage.ArchiveReadPlanCommand
type ArchiveLookupCommand = appstorage.ArchiveLookupCommand

// MasterDataBackupRequest содержит безопасные metadata для backup-before-data-load.
type MasterDataBackupRequest = appmastersync.BackupRequest

// MasterDataBackupFunc выполняет backup перед Cloud -> Edge full_snapshot.
type MasterDataBackupFunc = appmastersync.BackupFunc

type ListLocalEventsQuery struct {
	Limit     int
	EventType string
}

type ClaimPendingOutboxCommand struct {
	Limit    int
	LockedBy string
}

type ReclaimStaleOutboxCommand struct {
	StaleBefore time.Time
}

type Service struct {
	repo         ports.Repository
	restaurants  *apprestaurant.Service
	devices      *appdevice.Service
	employees    *appemployee.Service
	auth         *appauth.Service
	floor        *appfloor.Service
	catalog      *appcatalog.Service
	menu         *appmenu.Service
	shifts       *appshift.Service
	orders       *apporder.Service
	prechecks    *appprecheck.Service
	pricing      *apppricing.Service
	checks       *appcheck.Service
	cash         *appcash.Service
	inventory    *appinventory.Service
	masterSync   *appmastersync.Service
	provisioning *appprovisioning.Service
	storage      *appstorage.Service
	localEvents  ports.LocalEventRepository
	outbox       *shared.OutboxService
}

// ServiceOptions задает runtime hooks верхнего POS application service.
type ServiceOptions struct {
	MasterDataBackupBeforeFullSnapshot MasterDataBackupFunc
	CloudProvisioningURL               string
	LicenseServerURL                   string
	CloudProvisioningClient            appprovisioning.CloudClient
	LicenseProvisioningClient          appprovisioning.LicenseClient
	StorageArchiveDir                  string
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return NewServiceWithOptions(repo, tx, ids, clock, ServiceOptions{})
}

// NewServiceWithOptions создает POS application service с дополнительными runtime hooks.
func NewServiceWithOptions(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock, options ServiceOptions) *Service {
	pricingSvc := apppricing.NewService(repo, tx, ids, clock)
	s := &Service{
		repo:        repo,
		restaurants: apprestaurant.NewService(repo, tx, ids, clock),
		devices:     appdevice.NewService(repo, tx, ids, clock),
		employees:   appemployee.NewService(repo, tx, ids, clock),
		auth:        appauth.NewService(repo, tx, ids, clock),
		floor:       appfloor.NewService(repo, tx, ids, clock),
		catalog:     appcatalog.NewService(repo, tx, ids, clock),
		menu:        appmenu.NewService(repo, tx, ids, clock),
		shifts:      appshift.NewService(repo, tx, ids, clock),
		orders:      apporder.NewService(repo, tx, ids, clock),
		pricing:     pricingSvc,
		prechecks:   appprecheck.NewService(repo, tx, ids, clock, pricingSvc),
		checks:      appcheck.NewService(repo, tx, ids, clock),
		cash:        appcash.NewService(repo, tx, ids, clock),
		inventory:   appinventory.NewService(repo, tx, ids, clock),
		masterSync: appmastersync.NewServiceWithOptions(repo, tx, ids, clock, appmastersync.Options{
			BackupBeforeFullSnapshot: options.MasterDataBackupBeforeFullSnapshot,
		}),
		storage:     appstorage.NewService(repo, ids, clock, appstorage.Options{ArchiveDir: options.StorageArchiveDir}),
		localEvents: repo,
		outbox:      shared.NewOutboxService(repo, tx, clock),
	}
	s.provisioning = appprovisioning.NewService(repo, tx, ids, clock, appprovisioning.Options{
		CloudURL:   options.CloudProvisioningURL,
		LicenseURL: options.LicenseServerURL,
		Cloud:      options.CloudProvisioningClient,
		License:    options.LicenseProvisioningClient,
		Apply:      s.masterSync.ApplyMasterData,
		Pair:       s.devices.PairEdgeNode,
	})
	return s
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

func (s *Service) PairEdgeNode(ctx context.Context, cmd PairEdgeNodeCommand) (*domain.EdgeNodeIdentity, error) {
	return s.devices.PairEdgeNode(ctx, cmd)
}

func (s *Service) GetPairingStatus(ctx context.Context) (domain.PairingStatus, error) {
	return s.devices.GetPairingStatus(ctx)
}

func (s *Service) GetProvisioningStatus(ctx context.Context) (domain.ProvisioningStatusView, error) {
	return s.provisioning.GetStatus(ctx)
}

func (s *Service) RegisterCloudProvisioning(ctx context.Context, cmd RegisterCloudProvisioningCommand) (domain.ProvisioningStatusView, error) {
	return s.provisioning.RegisterCloud(ctx, cmd)
}

func (s *Service) PollCloudAssignment(ctx context.Context) (domain.ProvisioningStatusView, error) {
	return s.provisioning.PollAssignment(ctx)
}

// MaintainCloudProvisioning выполняет один безопасный фоновой шаг provisioning.
// После pairing/assignment метод не ходит в Cloud, чтобы не сбрасывать node_token
// и не создавать повторный register/assignment/snapshot цикл.
func (s *Service) MaintainCloudProvisioning(ctx context.Context, cmd RegisterCloudProvisioningCommand) (domain.ProvisioningStatusView, error) {
	status, err := s.GetProvisioningStatus(ctx)
	if err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	switch status.Status {
	case domain.ProvisioningPaired, domain.ProvisioningAssignedDownloadingSnapshot:
		return status, nil
	case domain.ProvisioningUnpairedRegistered:
		return s.PollCloudAssignment(ctx)
	case domain.ProvisioningNotConfigured, domain.ProvisioningError:
		return s.RegisterCloudProvisioning(ctx, cmd)
	default:
		return status, nil
	}
}

func (s *Service) PairViaLicense(ctx context.Context, cmd PairViaLicenseCommand) (domain.ProvisioningStatusView, error) {
	return s.provisioning.PairViaLicense(ctx, cmd)
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

func (s *Service) PinLogin(ctx context.Context, cmd PinLoginCommand) (*domain.PinLoginResult, error) {
	return s.auth.PinLogin(ctx, cmd)
}

func (s *Service) Logout(ctx context.Context, cmd LogoutCommand) (*domain.AuthSession, error) {
	return s.auth.Logout(ctx, cmd)
}

func (s *Service) GetSession(ctx context.Context, sessionID, nodeDeviceID, clientDeviceID string) (*domain.PinLoginResult, error) {
	return s.auth.GetSession(ctx, sessionID, nodeDeviceID, clientDeviceID)
}

func (s *Service) CreateHall(ctx context.Context, cmd CreateHallCommand) (*domain.Hall, error) {
	return s.floor.CreateHall(ctx, cmd)
}

func (s *Service) ListHalls(ctx context.Context, restaurantID string) ([]domain.Hall, error) {
	return s.floor.ListHalls(ctx, restaurantID)
}

// ListHallsAsOperator возвращает halls для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) ListHallsAsOperator(ctx context.Context, restaurantID string, meta CommandMeta) ([]domain.Hall, error) {
	return s.floor.ListHallsAsOperator(ctx, restaurantID, meta)
}

func (s *Service) ArchiveHall(ctx context.Context, cmd ArchiveHallCommand) error {
	return s.floor.ArchiveHall(ctx, cmd)
}

func (s *Service) CreateTable(ctx context.Context, cmd CreateTableCommand) (*domain.Table, error) {
	return s.floor.CreateTable(ctx, cmd)
}

func (s *Service) ListTables(ctx context.Context, restaurantID, hallID string) ([]domain.Table, error) {
	return s.floor.ListTables(ctx, restaurantID, hallID)
}

// ListTablesAsOperator возвращает tables для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) ListTablesAsOperator(ctx context.Context, restaurantID, hallID string, meta CommandMeta) ([]domain.Table, error) {
	return s.floor.ListTablesAsOperator(ctx, restaurantID, hallID, meta)
}

func (s *Service) ArchiveTable(ctx context.Context, cmd ArchiveTableCommand) error {
	return s.floor.ArchiveTable(ctx, cmd)
}

func (s *Service) ListCatalogItems(ctx context.Context) ([]domain.CatalogItem, error) {
	return s.catalog.ListCatalogItems(ctx)
}

// ListCatalogItemsAsOperator возвращает catalog items для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) ListCatalogItemsAsOperator(ctx context.Context, meta CommandMeta) ([]domain.CatalogItem, error) {
	return s.catalog.ListCatalogItemsAsOperator(ctx, meta)
}

func (s *Service) CreateCatalogItem(ctx context.Context, cmd CreateCatalogItemCommand) (*domain.CatalogItem, error) {
	return s.catalog.CreateCatalogItem(ctx, cmd)
}

func (s *Service) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	return s.menu.ListMenuItems(ctx)
}

// ListMenuItemsAsOperator возвращает menu items для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) ListMenuItemsAsOperator(ctx context.Context, meta CommandMeta) ([]domain.MenuItem, error) {
	return s.menu.ListMenuItemsAsOperator(ctx, meta)
}

func (s *Service) CreateMenuItem(ctx context.Context, cmd CreateMenuItemCommand) (*domain.MenuItem, error) {
	return s.menu.CreateMenuItem(ctx, cmd)
}

func (s *Service) GetCurrentShift(ctx context.Context, meta CommandMeta) (*domain.Shift, error) {
	return s.shifts.GetCurrentShift(ctx, meta)
}

func (s *Service) OpenShift(ctx context.Context, cmd OpenShiftCommand) (*domain.Shift, error) {
	return s.shifts.OpenShift(ctx, cmd)
}

func (s *Service) ListRecentShifts(ctx context.Context, cmd ListRecentShiftsCommand) ([]domain.Shift, error) {
	return s.shifts.ListRecentShifts(ctx, cmd)
}

func (s *Service) CloseShift(ctx context.Context, cmd CloseShiftCommand) (*domain.Shift, error) {
	return s.shifts.CloseShift(ctx, cmd)
}

func (s *Service) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return s.orders.GetOrder(ctx, id)
}

func (s *Service) GetOrderAsOperator(ctx context.Context, id string, meta CommandMeta) (*domain.Order, error) {
	return s.orders.GetOrderAsOperator(ctx, id, meta)
}

func (s *Service) GetCurrentOrderByTable(ctx context.Context, deviceID, tableID string) (*domain.Order, error) {
	return s.orders.GetCurrentOrderByTable(ctx, deviceID, tableID)
}

func (s *Service) GetCurrentOrderByTableAsOperator(ctx context.Context, tableID string, meta CommandMeta) (*domain.Order, error) {
	return s.orders.GetCurrentOrderByTableAsOperator(ctx, tableID, meta)
}

func (s *Service) ListActiveOrdersByHallAsOperator(ctx context.Context, hallID string, meta CommandMeta) ([]domain.Order, error) {
	return s.orders.ListActiveOrdersByHallAsOperator(ctx, hallID, meta)
}

func (s *Service) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*domain.Order, error) {
	return s.orders.CreateOrder(ctx, cmd)
}

func (s *Service) AddOrderLine(ctx context.Context, cmd AddOrderLineCommand) (*domain.OrderLine, error) {
	return s.orders.AddOrderLine(ctx, cmd)
}

func (s *Service) ChangeOrderLineQuantity(ctx context.Context, cmd ChangeOrderLineQuantityCommand) (*domain.OrderLine, error) {
	return s.orders.ChangeOrderLineQuantity(ctx, cmd)
}

func (s *Service) UpdateOrderLineModifiers(ctx context.Context, cmd UpdateOrderLineModifiersCommand) (*domain.OrderLine, error) {
	return s.orders.UpdateOrderLineModifiers(ctx, cmd)
}

func (s *Service) VoidOrderLine(ctx context.Context, cmd VoidOrderLineCommand) (*domain.OrderLine, error) {
	return s.orders.VoidOrderLine(ctx, cmd)
}

func (s *Service) UpdateOrderLineDetails(ctx context.Context, cmd UpdateOrderLineDetailsCommand) (*domain.OrderLine, error) {
	return s.orders.UpdateOrderLineDetails(ctx, cmd)
}

func (s *Service) CloseOrder(ctx context.Context, cmd CloseOrderCommand) (*domain.Order, error) {
	return s.orders.CloseOrder(ctx, cmd)
}

func (s *Service) IssuePrecheck(ctx context.Context, cmd IssuePrecheckCommand) (*domain.Precheck, error) {
	return s.prechecks.IssuePrecheck(ctx, cmd)
}

func (s *Service) AddDiscount(ctx context.Context, cmd AddDiscountCommand) (*domain.OrderDiscount, error) {
	return s.pricing.AddDiscount(ctx, cmd)
}

func (s *Service) AddSurcharge(ctx context.Context, cmd AddSurchargeCommand) (*domain.OrderSurcharge, error) {
	return s.pricing.AddSurcharge(ctx, cmd)
}

func (s *Service) ListActivePricingPoliciesAsOperator(ctx context.Context, meta CommandMeta) ([]domain.PricingPolicy, error) {
	return s.pricing.ListActivePricingPoliciesAsOperator(ctx, meta)
}

func (s *Service) GetOrderPricingAsOperator(ctx context.Context, orderID string, meta CommandMeta) (*domain.CalculationResult, error) {
	return s.pricing.GetOrderPricingAsOperator(ctx, orderID, meta)
}

func (s *Service) CalculateOrderPricing(ctx context.Context, orderID string) (domain.CalculationResult, error) {
	return s.pricing.CalculateOrderPricing(ctx, orderID)
}

func (s *Service) GetPrecheck(ctx context.Context, id string) (*domain.Precheck, error) {
	return s.prechecks.GetPrecheck(ctx, id)
}

func (s *Service) GetPrecheckAsOperator(ctx context.Context, id string, meta CommandMeta) (*domain.Precheck, error) {
	return s.prechecks.GetPrecheckAsOperator(ctx, id, meta)
}

func (s *Service) ListPrechecksByOrder(ctx context.Context, orderID string) ([]domain.Precheck, error) {
	return s.prechecks.ListPrechecksByOrder(ctx, orderID)
}

func (s *Service) ListPrechecksByOrderAsOperator(ctx context.Context, orderID string, meta CommandMeta) ([]domain.Precheck, error) {
	return s.prechecks.ListPrechecksByOrderAsOperator(ctx, orderID, meta)
}

func (s *Service) CancelPrecheck(ctx context.Context, cmd CancelPrecheckCommand) (*domain.Precheck, error) {
	return s.prechecks.CancelPrecheck(ctx, cmd)
}

func (s *Service) ReprintPrecheck(ctx context.Context, cmd ReprintPrecheckCommand) (*domain.ReprintDocument, error) {
	return s.prechecks.ReprintPrecheck(ctx, cmd)
}

func (s *Service) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return s.checks.GetCheck(ctx, id)
}

func (s *Service) GetCheckByOrder(ctx context.Context, orderID string) (*domain.Check, error) {
	return s.repo.GetCheckByOrder(ctx, orderID)
}

func (s *Service) GetCheckAsOperator(ctx context.Context, id string, meta CommandMeta) (*domain.Check, error) {
	return s.checks.GetCheckAsOperator(ctx, id, meta)
}

func (s *Service) ListFinancialOperationsByCheckAsOperator(ctx context.Context, checkID string, meta CommandMeta, limit, offset int) ([]domain.FinancialOperation, error) {
	return s.checks.ListFinancialOperationsByCheckAsOperator(ctx, checkID, meta, limit, offset)
}

func (s *Service) ListFinancialOperationsAsOperator(ctx context.Context, cmd ListFinancialOperationsCommand) ([]domain.FinancialOperation, error) {
	return s.checks.ListFinancialOperationsAsOperator(ctx, cmd)
}

func (s *Service) ListClosedOrders(ctx context.Context, cmd ListClosedOrdersCommand) ([]domain.OrderSummary, error) {
	return s.checks.ListClosedOrders(ctx, cmd)
}

func (s *Service) CapturePayment(ctx context.Context, cmd CapturePaymentCommand) (*domain.Payment, error) {
	return s.checks.CapturePayment(ctx, cmd)
}

func (s *Service) RefundPayment(ctx context.Context, cmd RefundPaymentCommand) (*domain.Payment, error) {
	return s.checks.RefundPayment(ctx, cmd)
}

func (s *Service) RecordCancellation(ctx context.Context, cmd RecordCheckCancellationCommand) (*domain.FinancialOperation, error) {
	return s.checks.RecordCancellation(ctx, cmd)
}

func (s *Service) RecordRefund(ctx context.Context, cmd RecordCheckRefundCommand) (*domain.FinancialOperation, error) {
	return s.checks.RecordRefund(ctx, cmd)
}

func (s *Service) ReprintCheck(ctx context.Context, cmd ReprintCheckCommand) (*domain.ReprintDocument, error) {
	return s.checks.ReprintCheck(ctx, cmd)
}

func (s *Service) GetCurrentCashSession(ctx context.Context, deviceID string) (*domain.CashSession, error) {
	return s.cash.GetCurrentCashSession(ctx, deviceID)
}

func (s *Service) GetCurrentCashSessionAsOperator(ctx context.Context, meta CommandMeta) (*domain.CashSession, error) {
	return s.cash.GetCurrentCashSessionAsOperator(ctx, meta)
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

func (s *Service) CreateManualStockDocument(ctx context.Context, cmd CreateManualStockDocumentCommand) (*domain.StockDocument, error) {
	return s.inventory.CreateManualStockDocument(ctx, cmd)
}

func (s *Service) ApplyMasterData(ctx context.Context, cmd ApplyMasterDataCommand) (*ApplyMasterDataResult, error) {
	return s.masterSync.ApplyMasterData(ctx, cmd)
}

func (s *Service) GetSyncExchangeState(ctx context.Context) (domain.SyncExchangeState, error) {
	state, err := s.repo.GetEdgeProvisioningState(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.SyncExchangeState{}, nil
		}
		return domain.SyncExchangeState{}, err
	}
	if strings.TrimSpace(state.NodeDeviceID) == "" || strings.TrimSpace(state.CredentialsToken) == "" {
		return domain.SyncExchangeState{}, nil
	}
	streamStates, err := s.repo.ListMasterDataSyncStates(ctx, state.NodeDeviceID)
	if err != nil {
		return domain.SyncExchangeState{}, err
	}
	byStream := map[domain.MasterDataStream]domain.MasterDataSyncState{}
	for _, item := range streamStates {
		byStream[item.StreamName] = item
	}
	streams := make([]domain.SyncExchangeStreamRequest, 0, len(syncExchangeStreams()))
	for _, streamName := range syncExchangeStreams() {
		local := byStream[streamName]
		checkpoint := ""
		if local.CheckpointToken != nil {
			checkpoint = *local.CheckpointToken
		}
		streams = append(streams, domain.SyncExchangeStreamRequest{
			StreamName:       string(streamName),
			LastCloudVersion: local.LastCloudVersion,
			CheckpointToken:  checkpoint,
		})
	}
	return domain.SyncExchangeState{
		NodeDeviceID: strings.TrimSpace(state.NodeDeviceID),
		RestaurantID: strings.TrimSpace(state.RestaurantID),
		AuthToken:    strings.TrimSpace(state.CredentialsToken),
		Streams:      streams,
	}, nil
}

func (s *Service) RefreshSyncExchangeState(ctx context.Context) error {
	status, err := s.GetProvisioningStatus(ctx)
	if err != nil {
		return err
	}
	if status.Status == domain.ProvisioningUnpairedRegistered {
		_, err = s.PollCloudAssignment(ctx)
		return err
	}
	return nil
}

func (s *Service) ApplySyncExchangeCloudPackages(ctx context.Context, packages []domain.CloudPackage) error {
	for _, pkg := range packages {
		stream := domain.MasterDataStream(strings.TrimSpace(pkg.StreamName))
		if !isSyncExchangeStream(stream) {
			return fmt.Errorf("%w: unsupported exchange stream %q", domain.ErrInvalid, pkg.StreamName)
		}
		state, err := s.repo.GetMasterDataSyncState(ctx, strings.TrimSpace(pkgNodeDeviceID(pkg)), stream)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if state != nil {
			localCheckpoint := ""
			if state.CheckpointToken != nil {
				localCheckpoint = *state.CheckpointToken
			}
			if state.LastCloudVersion > pkg.CloudVersion {
				return fmt.Errorf("%w: local stream %s version is ahead of cloud package", domain.ErrConflict, stream)
			}
			if state.LastCloudVersion == pkg.CloudVersion && localCheckpoint != "" && pkg.CheckpointToken != "" && localCheckpoint != pkg.CheckpointToken {
				return fmt.Errorf("%w: local stream %s checkpoint conflicts with cloud package", domain.ErrConflict, stream)
			}
			if state.LastCloudVersion == pkg.CloudVersion && (localCheckpoint == pkg.CheckpointToken || pkg.CheckpointToken == "") {
				continue
			}
		}
		var cmd ApplyMasterDataCommand
		if err := json.Unmarshal(pkg.PayloadJSON, &cmd); err != nil {
			return fmt.Errorf("%w: exchange package payload is invalid JSON", domain.ErrInvalid)
		}
		cmd.StreamName = stream
		cmd.RestaurantID = strings.TrimSpace(pkg.RestaurantID)
		cmd.SyncMode = domain.SyncMode(strings.TrimSpace(pkg.SyncMode))
		cmd.FullSnapshotReason = strings.TrimSpace(pkg.FullSnapshotReason)
		cmd.CloudVersion = pkg.CloudVersion
		cmd.CheckpointToken = strings.TrimSpace(pkg.CheckpointToken)
		cmd.CloudUpdatedAt = strings.TrimSpace(pkg.CloudUpdatedAt)
		cmd.CommandMeta = CommandMeta{NodeDeviceID: pkgNodeDeviceID(pkg), DeviceID: pkgNodeDeviceID(pkg), Origin: domain.OriginCloudSync}
		if _, err := s.ApplyMasterData(ctx, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	return s.outbox.ListOutbox(ctx, limit)
}

func (s *Service) GetStorageLifecycleStatus(ctx context.Context, cmd StorageStatusCommand) (domain.StorageLifecycleStatus, error) {
	return s.storage.GetStatus(ctx, cmd)
}

func (s *Service) DryRunStorageRetention(ctx context.Context, cmd RetentionDryRunCommand) (domain.StorageRetentionDryRunResult, error) {
	return s.storage.DryRunRetention(ctx, cmd)
}

func (s *Service) BuildStorageArchiveExportPlan(ctx context.Context, cmd ArchiveExportPlanCommand) (domain.StorageArchiveExportPlan, error) {
	return s.storage.BuildArchiveExportPlan(ctx, cmd)
}

func (s *Service) ExportStorageArchive(ctx context.Context, cmd ArchiveExportCommand) (domain.StorageArchiveExportResult, error) {
	return s.storage.ExportArchive(ctx, cmd)
}

func (s *Service) BuildStorageArchiveApplyPlan(ctx context.Context, cmd ArchiveApplyPlanCommand) (domain.StorageArchiveApplyPlan, error) {
	return s.storage.BuildArchiveApplyPlan(ctx, cmd)
}

func (s *Service) BuildStorageArchiveReadPlan(ctx context.Context, cmd ArchiveReadPlanCommand) (domain.StorageArchiveReadPlan, error) {
	return s.storage.BuildArchiveReadPlan(ctx, cmd)
}

func (s *Service) LookupStorageArchivePreview(ctx context.Context, cmd ArchiveLookupCommand) (domain.StorageArchiveLookupPreview, error) {
	return s.storage.LookupArchivePreview(ctx, cmd)
}

func syncExchangeStreams() []domain.MasterDataStream {
	return []domain.MasterDataStream{
		domain.MasterDataStreamRestaurants,
		domain.MasterDataStreamDevices,
		domain.MasterDataStreamStaff,
		domain.MasterDataStreamFloor,
		domain.MasterDataStreamCatalog,
		domain.MasterDataStreamMenu,
		domain.MasterDataStreamPricing,
	}
}

func isSyncExchangeStream(stream domain.MasterDataStream) bool {
	for _, supported := range syncExchangeStreams() {
		if stream == supported {
			return true
		}
	}
	return false
}

func pkgNodeDeviceID(pkg domain.CloudPackage) string {
	return strings.TrimSpace(pkg.NodeDeviceID)
}

func (s *Service) ListOutboxAsOperator(ctx context.Context, meta CommandMeta, limit int) ([]domain.OutboxMessage, error) {
	return s.outbox.ListOutboxAsOperator(ctx, meta, limit)
}

func (s *Service) GetSyncStatus(ctx context.Context) (domain.SyncStatus, error) {
	return s.outbox.GetSyncStatus(ctx)
}

func (s *Service) GetSyncStatusAsOperator(ctx context.Context, meta CommandMeta) (domain.SyncStatus, error) {
	return s.outbox.GetSyncStatusAsOperator(ctx, meta)
}

func (s *Service) RetryFailedOutbox(ctx context.Context) (int, error) {
	return s.outbox.RetryFailedOutbox(ctx)
}

// RetryFailedOutboxAsOperator повторяет failed outbox messages с операторской проверкой RBAC.
func (s *Service) RetryFailedOutboxAsOperator(ctx context.Context, meta CommandMeta) (int, error) {
	return s.outbox.RetryFailedOutboxAsOperator(ctx, meta)
}

func (s *Service) ClaimPendingOutbox(ctx context.Context, cmd ClaimPendingOutboxCommand) ([]domain.OutboxMessage, error) {
	return s.outbox.ClaimPendingOutbox(ctx, cmd.Limit, cmd.LockedBy)
}

func (s *Service) ReclaimStaleProcessingOutbox(ctx context.Context, cmd ReclaimStaleOutboxCommand) (int, error) {
	return s.outbox.ReclaimStaleProcessingOutbox(ctx, cmd.StaleBefore)
}

func (s *Service) ReleaseProcessingOutbox(ctx context.Context, lockedBy string) (int, error) {
	return s.outbox.ReleaseProcessingOutbox(ctx, lockedBy)
}

func (s *Service) ListLocalEvents(ctx context.Context, query ListLocalEventsQuery) ([]domain.LocalEvent, error) {
	return s.localEvents.ListLocalEvents(ctx, query.Limit, query.EventType)
}

// ListLocalEventsAsOperator возвращает local events для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) ListLocalEventsAsOperator(ctx context.Context, meta CommandMeta, query ListLocalEventsQuery) ([]domain.LocalEvent, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionSyncView)); err != nil {
		return nil, err
	}
	return s.ListLocalEvents(ctx, query)
}

func (s *Service) MarkOutboxSent(ctx context.Context, id string) error {
	return s.outbox.MarkOutboxSent(ctx, id)
}

func (s *Service) MarkOutboxFailed(ctx context.Context, id, reason string) error {
	return s.outbox.MarkOutboxFailed(ctx, id, reason)
}

func (s *Service) MarkOutboxRetryableFailure(ctx context.Context, id, reason string) error {
	return s.outbox.MarkOutboxRetryableFailure(ctx, id, reason)
}

func (s *Service) SuspendOutboxMessage(ctx context.Context, id, reason string) error {
	return s.outbox.SuspendOutboxMessage(ctx, id, reason)
}
