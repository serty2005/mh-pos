package app_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"pos-backend/internal/platform/clock"
	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/pos/app"
	"pos-backend/internal/pos/domain"
	possqlite "pos-backend/internal/pos/infra/sqlite"
	"pos-backend/internal/pos/ports"
)

type testIDs struct {
	n int
}

func (g *testIDs) NewID() string {
	g.n++
	return fmt.Sprintf("id-%03d", g.n)
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
}

var (
	errInjectedLocalEvent = errors.New("injected local event failure")
	errInjectedOutbox     = errors.New("injected outbox failure")
)

type localEventFailingRepo struct {
	ports.Repository
}

func (r localEventFailingRepo) CreateLocalEvent(context.Context, *domain.LocalEvent) error {
	return errInjectedLocalEvent
}

type outboxFailingRepo struct {
	ports.Repository
}

func (r outboxFailingRepo) CreateOutboxMessage(context.Context, *domain.OutboxMessage) error {
	return errInjectedOutbox
}

type fixture struct {
	ctx        context.Context
	db         *sql.DB
	repo       *possqlite.Repository
	service    *app.Service
	restaurant *domain.Restaurant
	device     *domain.Device
	employee   *domain.Employee
	menuItem   *domain.MenuItem
}

const bootstrapDeviceID = "bootstrap-device"

func seedMeta(deviceID string) app.CommandMeta {
	return app.CommandMeta{DeviceID: deviceID, Origin: app.OriginSystemSeed}
}

func edgeMeta(deviceID string) app.CommandMeta {
	return app.CommandMeta{DeviceID: deviceID, Origin: app.OriginEdgeDevice}
}

func (f *fixture) edgeMeta() app.CommandMeta {
	return edgeMeta(f.device.ID)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "pos.db")
	db, err := platformsqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := platformsqlite.MigrateDir(ctx, db, filepath.Join("..", "..", "..", "migrations", "sqlite")); err != nil {
		t.Fatal(err)
	}
	repo := possqlite.NewRepository(db)
	service := app.NewService(repo, platformsqlite.NewTxManager(db), &testIDs{}, fixedClock{})
	f := &fixture{ctx: ctx, db: db, repo: repo, service: service}
	f.seed(t)
	return f
}

func (f *fixture) seed(t *testing.T) {
	t.Helper()
	var err error
	f.restaurant, err = f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{CommandMeta: seedMeta(bootstrapDeviceID), Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{CommandMeta: seedMeta(bootstrapDeviceID), Name: "cashier", PermissionsJSON: `{"pos":true}`})
	if err != nil {
		t.Fatal(err)
	}
	f.device, err = f.service.RegisterDevice(f.ctx, app.RegisterDeviceCommand{CommandMeta: seedMeta(bootstrapDeviceID), RestaurantID: f.restaurant.ID, DeviceCode: "POS-1", Name: "Main", Type: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	f.employee, err = f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, RoleID: role.ID, Name: "Anna", PINHash: "hash"})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{CommandMeta: seedMeta(f.device.ID), Type: domain.CatalogItemDish, Name: "Soup", SKU: "SOUP", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	f.menuItem, err = f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{CommandMeta: seedMeta(f.device.ID), CatalogItemID: catalog.ID, Name: "Soup", Price: 1000, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) openShift(t *testing.T) *domain.Shift {
	t.Helper()
	shift, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return shift
}

func (f *fixture) createPaidOrder(t *testing.T) (*domain.Order, *domain.Check) {
	t.Helper()
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	check, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), CheckID: check.ID, Method: domain.PaymentCash, Amount: check.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	return order, check
}

func countRows(t *testing.T, f *fixture, table string) int {
	t.Helper()
	switch table {
	case "orders", "order_lines", "prechecks", "checks", "payments", "payment_attempts", "cash_sessions", "cash_drawer_events", "pos_sync_outbox", "local_event_log", "roles", "catalog_items", "menu_items":
	default:
		t.Fatalf("unexpected table %q", table)
	}
	var n int
	if err := f.db.QueryRowContext(f.ctx, fmt.Sprintf("SELECT COUNT(1) FROM %s", table)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func outboxHasDeviceAndOrigin(t *testing.T, f *fixture, commandID, deviceID string, origin domain.CommandOrigin) bool {
	t.Helper()
	var n int
	err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM pos_sync_outbox WHERE command_id = ? AND device_id = ? AND origin = ?`, commandID, deviceID, string(origin)).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func TestCannotOpenTwoShiftsOnDevice(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	_, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, OpenedByEmployeeID: f.employee.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotCreateOrderWithoutOpenShift(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotOpenCashSessionWithoutOpenShift(t *testing.T) {
	f := newFixture(t)
	before := countRows(t, f, "cash_sessions")

	_, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  100,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if after := countRows(t, f, "cash_sessions"); after != before {
		t.Fatalf("expected no cash session write, before=%d after=%d", before, after)
	}
}

func TestCannotRecordCashDrawerEventWithoutActiveCashSession(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)

	_, err := f.service.RecordCashDrawerEvent(f.ctx, app.RecordCashDrawerEventCommand{
		CommandMeta:         f.edgeMeta(),
		CreatedByEmployeeID: f.employee.ID,
		EventType:           domain.CashDrawerCashIn,
		Amount:              100,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if events := countRows(t, f, "cash_drawer_events"); events != 0 {
		t.Fatalf("expected no cash drawer events, got %d", events)
	}
}

func TestDuplicateCashSessionCommandIDDoesNotCreateDuplicateRows(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	sessionsBefore := countRows(t, f, "cash_sessions")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	cmd := app.OpenCashSessionCommand{
		CommandMeta:        app.CommandMeta{CommandID: "cmd-open-cash-session-1", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  100,
	}
	if _, err := f.service.OpenCashSession(f.ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.OpenCashSession(f.ctx, cmd); !errors.Is(err, domain.ErrDuplicateCommand) {
		t.Fatalf("expected duplicate command, got %v", err)
	}
	if sessions := countRows(t, f, "cash_sessions"); sessions != sessionsBefore+1 {
		t.Fatalf("expected one cash session row, before=%d after=%d", sessionsBefore, sessions)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one local event row, before=%d after=%d", eventsBefore, events)
	}
}

func TestDuplicateCommandIDDoesNotCreateDuplicateOrderOrOutbox(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	ordersBefore := countRows(t, f, "orders")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	cmd := app.CreateOrderCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-create-order-1", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		TableName:   "A1",
		GuestCount:  1,
	}
	if _, err := f.service.CreateOrder(f.ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateOrder(f.ctx, cmd); !errors.Is(err, domain.ErrDuplicateCommand) {
		t.Fatalf("expected duplicate command, got %v", err)
	}
	if orders := countRows(t, f, "orders"); orders != ordersBefore+1 {
		t.Fatalf("expected one business row, before=%d after=%d", ordersBefore, orders)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one local event row, before=%d after=%d", eventsBefore, events)
	}
}

func TestRollbackRemovesDomainWriteWhenLocalEventWriteFails(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	ordersBefore := countRows(t, f, "orders")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	service := app.NewService(localEventFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 1000}, fixedClock{})

	_, err := service.CreateOrder(f.ctx, app.CreateOrderCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-local-event-fails", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		TableName:   "A1",
		GuestCount:  1,
	})
	if !errors.Is(err, errInjectedLocalEvent) {
		t.Fatalf("expected injected local event failure, got %v", err)
	}
	if orders := countRows(t, f, "orders"); orders != ordersBefore {
		t.Fatalf("expected no partial order write, before=%d after=%d", ordersBefore, orders)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no partial outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no partial local event write, before=%d after=%d", eventsBefore, events)
	}
}

func TestRollbackRemovesDomainAndLocalEventWhenOutboxWriteFails(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	ordersBefore := countRows(t, f, "orders")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	service := app.NewService(outboxFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 2000}, fixedClock{})

	_, err := service.CreateOrder(f.ctx, app.CreateOrderCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-outbox-fails", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		TableName:   "A1",
		GuestCount:  1,
	})
	if !errors.Is(err, errInjectedOutbox) {
		t.Fatalf("expected injected outbox failure, got %v", err)
	}
	if orders := countRows(t, f, "orders"); orders != ordersBefore {
		t.Fatalf("expected no partial order write, before=%d after=%d", ordersBefore, orders)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no partial local event write, before=%d after=%d", eventsBefore, events)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no partial outbox write, before=%d after=%d", outboxBefore, outbox)
	}
}

func TestRollbackRemovesCashSessionWhenOutboxWriteFails(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	sessionsBefore := countRows(t, f, "cash_sessions")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	service := app.NewService(outboxFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 3000}, fixedClock{})

	_, err := service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        app.CommandMeta{CommandID: "cmd-cash-outbox-fails", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  100,
	})
	if !errors.Is(err, errInjectedOutbox) {
		t.Fatalf("expected injected outbox failure, got %v", err)
	}
	if sessions := countRows(t, f, "cash_sessions"); sessions != sessionsBefore {
		t.Fatalf("expected no partial cash session write, before=%d after=%d", sessionsBefore, sessions)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no partial local event write, before=%d after=%d", eventsBefore, events)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no partial outbox write, before=%d after=%d", outboxBefore, outbox)
	}
}

func TestReferenceCreatesRequireDeviceID(t *testing.T) {
	f := newFixture(t)

	if _, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{Name: "manager", PermissionsJSON: `{}`}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid role command, got %v", err)
	}
	if _, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{Type: domain.CatalogItemGood, Name: "Tea", SKU: "TEA", BaseUnit: "portion"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid catalog command, got %v", err)
	}
	if _, err := f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{CatalogItemID: f.menuItem.CatalogItemID, Name: "Tea", Price: 300, Currency: "RUB"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid menu command, got %v", err)
	}
}

func TestWriteRejectsInvalidOrigin(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     app.CommandMeta{DeviceID: f.device.ID, Origin: domain.CommandOrigin("bad_origin")},
		Name:            "manager",
		PermissionsJSON: `{}`,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid origin, got %v", err)
	}
}

func TestValidDeviceIDCreatesBusinessAndOutboxRowsWithDefaultOrigin(t *testing.T) {
	f := newFixture(t)
	rolesBefore := countRows(t, f, "roles")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	commandID := "cmd-role-with-device"

	_, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     app.CommandMeta{CommandID: commandID, DeviceID: f.device.ID},
		Name:            "manager",
		PermissionsJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if roles := countRows(t, f, "roles"); roles != rolesBefore+1 {
		t.Fatalf("expected one role row, before=%d after=%d", rolesBefore, roles)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if !outboxHasDeviceAndOrigin(t, f, commandID, f.device.ID, domain.OriginEdgeDevice) {
		t.Fatal("expected outbox row to contain device_id and origin")
	}
}

func TestCannotCloseShiftWithOpenOrders(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	if _, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1"}); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{CommandMeta: f.edgeMeta(), ID: shift.ID, ClosedByEmployeeID: f.employee.ID, ClosingCashAmount: 0})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCloseShiftSucceedsWithoutOpenOrdersAndActiveCashSession(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)

	closed, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 shift.ID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != domain.ShiftClosed {
		t.Fatalf("expected closed shift, got %s", closed.Status)
	}
}

func TestCannotCloseShiftWithActiveCashSession(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	if _, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  100,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 shift.ID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  100,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	got, err := f.service.GetCurrentShift(f.ctx, f.device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != shift.ID || got.Status != domain.ShiftOpen {
		t.Fatalf("expected shift to remain open, got %+v", got)
	}
}

func TestCannotAddLineToClosedOrder(t *testing.T) {
	f := newFixture(t)
	order, _ := f.createPaidOrder(t)
	if _, err := f.service.CloseOrder(f.ctx, app.CloseOrderCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotCloseOrderWithoutFullPayment(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CloseOrder(f.ctx, app.CloseOrderCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestIssuePrecheckCreatesDormantSnapshotAndLocksOrderWithoutLegacyCheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	prechecksBefore := countRows(t, f, "prechecks")
	checksBefore := countRows(t, f, "checks")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-precheck-1", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if precheck.OrderID != order.ID || precheck.Status != domain.PrecheckIssued || precheck.Subtotal != 2000 || precheck.Total != 2000 {
		t.Fatalf("unexpected precheck: %+v", precheck)
	}
	if prechecks := countRows(t, f, "prechecks"); prechecks != prechecksBefore+1 {
		t.Fatalf("expected one precheck row, before=%d after=%d", prechecksBefore, prechecks)
	}
	if checks := countRows(t, f, "checks"); checks != checksBefore {
		t.Fatalf("expected legacy checks to remain unchanged, before=%d after=%d", checksBefore, checks)
	}
	lockedOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if lockedOrder.Status != domain.OrderLocked {
		t.Fatalf("expected order to be locked, got %s", lockedOrder.Status)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one local event row, before=%d after=%d", eventsBefore, events)
	}
	active, err := f.repo.GetActivePrecheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != precheck.ID {
		t.Fatalf("expected active precheck %s, got %s", precheck.ID, active.ID)
	}
}

func TestCannotAddLineToLockedOrder(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}

	_, err = f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestIssuePrecheckRollbackKeepsOrderOpenWhenLocalEventOrOutboxFails(t *testing.T) {
	cases := []struct {
		name    string
		repo    func(*possqlite.Repository) ports.Repository
		wantErr error
	}{
		{
			name:    "local-event",
			repo:    func(repo *possqlite.Repository) ports.Repository { return localEventFailingRepo{Repository: repo} },
			wantErr: errInjectedLocalEvent,
		},
		{
			name:    "outbox",
			repo:    func(repo *possqlite.Repository) ports.Repository { return outboxFailingRepo{Repository: repo} },
			wantErr: errInjectedOutbox,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.openShift(t)
			order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
				t.Fatal(err)
			}
			prechecksBefore := countRows(t, f, "prechecks")
			outboxBefore := countRows(t, f, "pos_sync_outbox")
			eventsBefore := countRows(t, f, "local_event_log")
			service := app.NewService(tc.repo(f.repo), platformsqlite.NewTxManager(f.db), &testIDs{n: 5000}, fixedClock{})

			_, err = service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
				CommandMeta: app.CommandMeta{CommandID: "cmd-precheck-fails-" + tc.name, DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
				OrderID:     order.ID,
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected injected failure, got %v", err)
			}
			if prechecks := countRows(t, f, "prechecks"); prechecks != prechecksBefore {
				t.Fatalf("expected no partial precheck write, before=%d after=%d", prechecksBefore, prechecks)
			}
			if events := countRows(t, f, "local_event_log"); events != eventsBefore {
				t.Fatalf("expected no partial local event write, before=%d after=%d", eventsBefore, events)
			}
			if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
				t.Fatalf("expected no partial outbox write, before=%d after=%d", outboxBefore, outbox)
			}
			got, err := f.repo.GetOrder(f.ctx, order.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != domain.OrderOpen {
				t.Fatalf("expected order to remain open, got %s", got.Status)
			}
			if _, err := f.repo.GetActivePrecheckByOrder(f.ctx, order.ID); !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("expected no active precheck, got %v", err)
			}
		})
	}
}

func TestCannotCancelMissingPrecheck(t *testing.T) {
	f := newFixture(t)
	beforeOutbox := countRows(t, f, "pos_sync_outbox")
	beforeEvents := countRows(t, f, "local_event_log")

	_, err := f.service.CancelPrecheck(f.ctx, app.CancelPrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-cancel-missing-precheck", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		PrecheckID:  "missing-precheck",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != beforeOutbox {
		t.Fatalf("expected no outbox write, before=%d after=%d", beforeOutbox, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != beforeEvents {
		t.Fatalf("expected no local event write, before=%d after=%d", beforeEvents, events)
	}
}

func TestCancelPrecheckUnlocksOrderAndWritesOutbox(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-before-cancel", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	cancelled, err := f.service.CancelPrecheck(f.ctx, app.CancelPrecheckCommand{
		CommandMeta:        app.CommandMeta{CommandID: "cmd-cancel-precheck-1", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		PrecheckID:         precheck.ID,
		ManagerEmployeeID:  f.employee.ID,
		CancellationReason: "guest changed order",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != domain.PrecheckCancelled || cancelled.ClosedAt == nil {
		t.Fatalf("expected cancelled precheck, got %+v", cancelled)
	}
	gotOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOrder.Status != domain.OrderOpen {
		t.Fatalf("expected order to unlock to open, got %s", gotOrder.Status)
	}
	if _, err := f.repo.GetActivePrecheckByOrder(f.ctx, order.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected no active precheck after cancel, got %v", err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one local event row, before=%d after=%d", eventsBefore, events)
	}
	var eventCount int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE command_id = ? AND event_type = 'PrecheckCancelled'`, "cmd-cancel-precheck-1").Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("expected PrecheckCancelled local event, got %d", eventCount)
	}
}

func TestCannotCancelNonIssuedPrecheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-for-non-issued-cancel", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CancelPrecheck(f.ctx, app.CancelPrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-cancel-once", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		PrecheckID:  precheck.ID,
	}); err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	_, err = f.service.CancelPrecheck(f.ctx, app.CancelPrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-cancel-non-issued", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		PrecheckID:  precheck.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no local event write, before=%d after=%d", eventsBefore, events)
	}
}

func TestCannotCancelPrecheckWithPaidTotalFoundation(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-paid-foundation", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE prechecks SET paid_total = 1 WHERE id = ?`, precheck.ID); err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	_, err = f.service.CancelPrecheck(f.ctx, app.CancelPrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-cancel-paid-foundation", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		PrecheckID:  precheck.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	gotOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOrder.Status != domain.OrderLocked {
		t.Fatalf("expected paid precheck cancel failure to keep order locked, got %s", gotOrder.Status)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no local event write, before=%d after=%d", eventsBefore, events)
	}
}

func TestCancelPrecheckRollbackKeepsIssuedAndOrderLockedWhenOutboxFails(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-before-cancel-outbox-fails", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	service := app.NewService(outboxFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 6000}, fixedClock{})

	_, err = service.CancelPrecheck(f.ctx, app.CancelPrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-cancel-outbox-fails", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		PrecheckID:  precheck.ID,
	})
	if !errors.Is(err, errInjectedOutbox) {
		t.Fatalf("expected injected outbox failure, got %v", err)
	}
	gotPrecheck, err := f.repo.GetPrecheck(f.ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotPrecheck.Status != domain.PrecheckIssued {
		t.Fatalf("expected precheck to remain issued after rollback, got %s", gotPrecheck.Status)
	}
	gotOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOrder.Status != domain.OrderLocked {
		t.Fatalf("expected order to remain locked after rollback, got %s", gotOrder.Status)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no partial outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no partial local event write, before=%d after=%d", eventsBefore, events)
	}
}

func TestDuplicateCancelPrecheckCommandIDDoesNotDoubleCancel(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-before-duplicate-cancel", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	cmd := app.CancelPrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-cancel-duplicate", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		PrecheckID:  precheck.ID,
	}
	if _, err := f.service.CancelPrecheck(f.ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CancelPrecheck(f.ctx, cmd); !errors.Is(err, domain.ErrDuplicateCommand) {
		t.Fatalf("expected duplicate command, got %v", err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one local event row, before=%d after=%d", eventsBefore, events)
	}
	var cancelEvents int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE command_id = ? AND event_type = 'PrecheckCancelled'`, cmd.CommandID).Scan(&cancelEvents); err != nil {
		t.Fatal(err)
	}
	if cancelEvents != 1 {
		t.Fatalf("expected one PrecheckCancelled event, got %d", cancelEvents)
	}
}

func TestCannotCloseShiftWithLockedOrders(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}

	_, err = f.service.CloseShift(f.ctx, app.CloseShiftCommand{CommandMeta: f.edgeMeta(), ID: shift.ID, ClosedByEmployeeID: f.employee.ID, ClosingCashAmount: 0})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotIssueSecondActivePrecheckForOrder(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-precheck-1", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	}); err != nil {
		t.Fatal(err)
	}

	_, err = f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-issue-precheck-2", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		OrderID:     order.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if prechecks := countRows(t, f, "prechecks"); prechecks != 1 {
		t.Fatalf("expected one precheck row, got %d", prechecks)
	}
}

func TestCreateOrderRejectsMismatchedRestaurantID(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	other, err := f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{CommandMeta: f.edgeMeta(), Name: "Other", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), RestaurantID: other.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotOverpayCheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	check, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), CheckID: check.ID, Method: domain.PaymentCash, Amount: check.Total + 1, Currency: "RUB"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCapturePaymentCreatesFirstAttemptWithEdgeContext(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	check, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta:           f.edgeMeta(),
		CheckID:               check.ID,
		Method:                domain.PaymentCard,
		Amount:                check.Total,
		Currency:              "rub",
		ProviderName:          "demo-psp",
		ProviderTransactionID: "txn-1",
		FingerprintHash:       "fingerprint-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if payment.EdgePaymentID == "" || payment.RestaurantID != f.restaurant.ID || payment.DeviceID != f.device.ID || payment.ShiftID == "" {
		t.Fatalf("expected payment edge context, got %+v", payment)
	}
	var attempts int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM payment_attempts WHERE payment_id = ? AND attempt_no = 1 AND provider_transaction_id = 'txn-1'`, payment.ID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("expected first payment attempt, got %d", attempts)
	}
}

func TestCapturePaymentRollbackRemovesAttemptPaymentOutboxAndLocalEvent(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	check, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	paymentsBefore := countRows(t, f, "payments")
	attemptsBefore := countRows(t, f, "payment_attempts")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	service := app.NewService(outboxFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 4000}, fixedClock{})

	_, err = service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-payment-outbox-fails", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
		CheckID:     check.ID,
		Method:      domain.PaymentCash,
		Amount:      check.Total,
		Currency:    "RUB",
	})
	if !errors.Is(err, errInjectedOutbox) {
		t.Fatalf("expected injected outbox failure, got %v", err)
	}
	if payments := countRows(t, f, "payments"); payments != paymentsBefore {
		t.Fatalf("expected no partial payment write, before=%d after=%d", paymentsBefore, payments)
	}
	if attempts := countRows(t, f, "payment_attempts"); attempts != attemptsBefore {
		t.Fatalf("expected no partial payment attempt write, before=%d after=%d", attemptsBefore, attempts)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no partial local event write, before=%d after=%d", eventsBefore, events)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no partial outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	var paidTotal int64
	var status string
	if err := f.db.QueryRowContext(f.ctx, `SELECT paid_total,status FROM checks WHERE id = ?`, check.ID).Scan(&paidTotal, &status); err != nil {
		t.Fatal(err)
	}
	if paidTotal != 0 || status != string(domain.CheckOpen) {
		t.Fatalf("expected check rollback, paid_total=%d status=%s", paidTotal, status)
	}
}

func TestOutboxEntryCreatedForEachWriteAction(t *testing.T) {
	f := newFixture(t)
	before, err := f.repo.CountOutbox(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, OpenedByEmployeeID: f.employee.ID}); err != nil {
		t.Fatal(err)
	}
	after, err := f.repo.CountOutbox(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after != before+1 {
		t.Fatalf("expected one outbox record for write action, before=%d after=%d", before, after)
	}
}

func TestListLocalEventsThroughServiceSupportsLimitAndFilter(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}

	events, err := f.service.ListLocalEvents(f.ctx, app.ListLocalEventsQuery{EventType: "OrderCreated", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("expected %d event, got %d", want, got)
	}
	if events[0].EventType != "OrderCreated" || events[0].AggregateID != order.ID {
		t.Fatalf("unexpected event: type=%s aggregate_id=%s", events[0].EventType, events[0].AggregateID)
	}
}

func TestKeyWritesCreateLocalEventsAndMatchingOutboxEnvelopes(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	check, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), CheckID: check.ID, Method: domain.PaymentCash, Amount: check.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CloseOrder(f.ctx, app.CloseOrderCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{CommandMeta: f.edgeMeta(), ID: shift.ID, ClosedByEmployeeID: f.employee.ID, ClosingCashAmount: 0}); err != nil {
		t.Fatal(err)
	}

	eventTypes := []string{"ShiftOpened", "ShiftClosed", "OrderCreated", "OrderLineAdded", "OrderClosed", "CheckCreated", "PaymentCaptured"}
	local := localEventCommandIDsByType(t, f, eventTypes, shift.ID)
	outbox := outboxEventCommandIDsByType(t, f, eventTypes)
	for _, eventType := range eventTypes {
		localCommands := local[eventType]
		outboxCommands := outbox[eventType]
		if len(localCommands) != 1 {
			t.Fatalf("expected one local %s event, got %d", eventType, len(localCommands))
		}
		if len(outboxCommands) != 1 {
			t.Fatalf("expected one outbox %s envelope, got %d", eventType, len(outboxCommands))
		}
		for eventID, commandID := range localCommands {
			outboxCommandID, ok := outboxCommands[eventID]
			if !ok {
				t.Fatalf("local event %s for %s missing from outbox envelope", eventID, eventType)
			}
			if outboxCommandID != commandID {
				t.Fatalf("command_id mismatch for %s event %s: local=%s outbox=%s", eventType, eventID, commandID, outboxCommandID)
			}
		}
	}
}

type syncEnvelopeProbe struct {
	Version      string  `json:"version"`
	EventID      string  `json:"event_id"`
	CommandID    string  `json:"command_id"`
	EventType    string  `json:"event_type"`
	RestaurantID *string `json:"restaurant_id"`
	DeviceID     string  `json:"device_id"`
	ShiftID      *string `json:"shift_id"`
}

func localEventCommandIDsByType(t *testing.T, f *fixture, wanted []string, shiftID string) map[string]map[string]string {
	t.Helper()
	want := make(map[string]bool, len(wanted))
	out := make(map[string]map[string]string, len(wanted))
	for _, eventType := range wanted {
		want[eventType] = true
		out[eventType] = map[string]string{}
	}
	rows, err := f.db.QueryContext(f.ctx, `SELECT event_type,event_id,command_id,payload_json,restaurant_id,device_id,shift_id FROM local_event_log`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var eventType, eventID, commandID, payload, deviceID string
		var restaurantID, gotShiftID sql.NullString
		if err := rows.Scan(&eventType, &eventID, &commandID, &payload, &restaurantID, &deviceID, &gotShiftID); err != nil {
			t.Fatal(err)
		}
		if !want[eventType] {
			continue
		}
		var envelope syncEnvelopeProbe
		if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Version != domain.SyncEnvelopeVersion || envelope.EventID != eventID || envelope.CommandID != commandID || envelope.EventType != eventType {
			t.Fatalf("local event envelope mismatch for %s", eventType)
		}
		if !restaurantID.Valid || restaurantID.String != f.restaurant.ID || envelope.RestaurantID == nil || *envelope.RestaurantID != f.restaurant.ID {
			t.Fatalf("expected restaurant_id in %s envelope", eventType)
		}
		if deviceID != f.device.ID || envelope.DeviceID != f.device.ID {
			t.Fatalf("expected device_id in %s envelope", eventType)
		}
		if !gotShiftID.Valid || gotShiftID.String != shiftID || envelope.ShiftID == nil || *envelope.ShiftID != shiftID {
			t.Fatalf("expected shift_id in %s envelope", eventType)
		}
		out[eventType][eventID] = commandID
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func outboxEventCommandIDsByType(t *testing.T, f *fixture, wanted []string) map[string]map[string]string {
	t.Helper()
	want := make(map[string]bool, len(wanted))
	out := make(map[string]map[string]string, len(wanted))
	for _, eventType := range wanted {
		want[eventType] = true
		out[eventType] = map[string]string{}
	}
	rows, err := f.db.QueryContext(f.ctx, `SELECT command_type,command_id,payload_json FROM pos_sync_outbox`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var eventType, commandID, payload string
		if err := rows.Scan(&eventType, &commandID, &payload); err != nil {
			t.Fatal(err)
		}
		if !want[eventType] {
			continue
		}
		var envelope syncEnvelopeProbe
		if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Version != domain.SyncEnvelopeVersion || envelope.EventType != eventType || envelope.EventID == "" || envelope.CommandID != commandID {
			t.Fatalf("outbox envelope mismatch for %s", eventType)
		}
		out[eventType][envelope.EventID] = commandID
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

var _ clock.Clock = fixedClock{}
