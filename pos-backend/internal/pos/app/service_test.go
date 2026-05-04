package app_test

import (
	"context"
	"database/sql"
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
	f.restaurant, err = f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{Name: "cashier", PermissionsJSON: `{"pos":true}`})
	if err != nil {
		t.Fatal(err)
	}
	f.device, err = f.service.RegisterDevice(f.ctx, app.RegisterDeviceCommand{RestaurantID: f.restaurant.ID, DeviceCode: "POS-1", Name: "Main", Type: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	f.employee, err = f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{RestaurantID: f.restaurant.ID, RoleID: role.ID, Name: "Anna", PINHash: "hash"})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{Type: domain.CatalogItemDish, Name: "Soup", SKU: "SOUP", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	f.menuItem, err = f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{CatalogItemID: catalog.ID, Name: "Soup", Price: 1000, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) openShift(t *testing.T) *domain.Shift {
	t.Helper()
	shift, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		RestaurantID:       f.restaurant.ID,
		DeviceID:           f.device.ID,
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
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{DeviceID: f.device.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	check, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CheckID: check.ID, Method: domain.PaymentCash, Amount: check.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	return order, check
}

func countRows(t *testing.T, f *fixture, table string) int {
	t.Helper()
	switch table {
	case "orders", "pos_sync_outbox", "roles", "catalog_items", "menu_items":
	default:
		t.Fatalf("unexpected table %q", table)
	}
	var n int
	if err := f.db.QueryRowContext(f.ctx, fmt.Sprintf("SELECT COUNT(1) FROM %s", table)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func outboxDeviceIDIsNull(t *testing.T, f *fixture, commandID string) bool {
	t.Helper()
	var n int
	err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM pos_sync_outbox WHERE command_id = ? AND device_id IS NULL`, commandID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func TestCannotOpenTwoShiftsOnDevice(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	_, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{RestaurantID: f.restaurant.ID, DeviceID: f.device.ID, OpenedByEmployeeID: f.employee.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotCreateOrderWithoutOpenShift(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{DeviceID: f.device.ID, TableName: "A1"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestDuplicateCommandIDDoesNotCreateDuplicateOrderOrOutbox(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	ordersBefore := countRows(t, f, "orders")
	outboxBefore := countRows(t, f, "pos_sync_outbox")

	cmd := app.CreateOrderCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-create-order-1"},
		DeviceID:    f.device.ID,
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
}

func TestReferenceCreatesAllowNullOutboxDeviceID(t *testing.T) {
	f := newFixture(t)

	if _, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     app.CommandMeta{CommandID: "cmd-role-without-device"},
		Name:            "manager",
		PermissionsJSON: `{}`,
	}); err != nil {
		t.Fatal(err)
	}
	if !outboxDeviceIDIsNull(t, f, "cmd-role-without-device") {
		t.Fatal("expected role outbox device_id to be NULL")
	}

	catalog, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-catalog-without-device"},
		Type:        domain.CatalogItemGood,
		Name:        "Tea",
		SKU:         "TEA",
		BaseUnit:    "portion",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outboxDeviceIDIsNull(t, f, "cmd-catalog-without-device") {
		t.Fatal("expected catalog item outbox device_id to be NULL")
	}

	if _, err := f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{
		CommandMeta:   app.CommandMeta{CommandID: "cmd-menu-without-device"},
		CatalogItemID: catalog.ID,
		Name:          "Tea",
		Price:         300,
		Currency:      "RUB",
	}); err != nil {
		t.Fatal(err)
	}
	if !outboxDeviceIDIsNull(t, f, "cmd-menu-without-device") {
		t.Fatal("expected menu item outbox device_id to be NULL")
	}
}

func TestCannotCloseShiftWithOpenOrders(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	if _, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{DeviceID: f.device.ID, TableName: "A1"}); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{ID: shift.ID, ClosedByEmployeeID: f.employee.ID, ClosingCashAmount: 0})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotAddLineToClosedOrder(t *testing.T) {
	f := newFixture(t)
	order, _ := f.createPaidOrder(t)
	if _, err := f.service.CloseOrder(f.ctx, app.CloseOrderCommand{OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotCloseOrderWithoutFullPayment(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{DeviceID: f.device.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CloseOrder(f.ctx, app.CloseOrderCommand{OrderID: order.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCreateOrderRejectsMismatchedRestaurantID(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	other, err := f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{Name: "Other", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CreateOrder(f.ctx, app.CreateOrderCommand{RestaurantID: other.ID, DeviceID: f.device.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotOverpayCheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{DeviceID: f.device.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	check, err := f.service.CreateCheck(f.ctx, app.CreateCheckCommand{OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CheckID: check.ID, Method: domain.PaymentCash, Amount: check.Total + 1, Currency: "RUB"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestOutboxEntryCreatedForEachWriteAction(t *testing.T) {
	f := newFixture(t)
	before, err := f.repo.CountOutbox(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{RestaurantID: f.restaurant.ID, DeviceID: f.device.ID, OpenedByEmployeeID: f.employee.ID}); err != nil {
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

var _ clock.Clock = fixedClock{}
