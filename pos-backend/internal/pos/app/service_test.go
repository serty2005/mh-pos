package app_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pos-backend/internal/platform/clock"
	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/pos/app"
	appmastersync "pos-backend/internal/pos/app/mastersync"
	appprovisioning "pos-backend/internal/pos/app/provisioning"
	appshared "pos-backend/internal/pos/app/shared"
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

type checkCreatedOutboxFailingRepo struct {
	ports.Repository
}

func (r checkCreatedOutboxFailingRepo) CreateOutboxMessage(ctx context.Context, msg *domain.OutboxMessage) error {
	if msg.CommandType == "CheckCreated" {
		return errInjectedOutbox
	}
	return r.Repository.CreateOutboxMessage(ctx, msg)
}

type countingProvisioningCloud struct {
	registerCalls int
	statusCalls   int
	snapshotCalls int
}

func (c *countingProvisioningCloud) RegisterDevice(context.Context, string, appprovisioning.CloudRegisterRequest) (appprovisioning.CloudRegisterResponse, error) {
	c.registerCalls++
	return appprovisioning.CloudRegisterResponse{NodeDeviceID: "node-1", Status: "pending_admin_approval"}, nil
}

func (c *countingProvisioningCloud) AssignmentStatus(context.Context, string, string) (appprovisioning.CloudAssignmentStatus, error) {
	c.statusCalls++
	return appprovisioning.CloudAssignmentStatus{NodeDeviceID: "node-1", Status: "pending_admin_approval"}, nil
}

func (c *countingProvisioningCloud) DownloadSnapshot(context.Context, string) (appmastersync.ApplyMasterDataCommand, error) {
	c.snapshotCalls++
	return appmastersync.ApplyMasterDataCommand{}, nil
}

type countingProvisioningLicense struct {
	resolveCalls int
}

func (c *countingProvisioningLicense) Resolve(context.Context, string, appprovisioning.LicenseResolveRequest) (appprovisioning.LicenseResolveResponse, error) {
	c.resolveCalls++
	return appprovisioning.LicenseResolveResponse{}, nil
}

type fixture struct {
	ctx            context.Context
	db             *sql.DB
	repo           *possqlite.Repository
	service        *app.Service
	restaurant     *domain.Restaurant
	device         *domain.Device
	employee       *domain.Employee
	manager        *domain.Employee
	session        *domain.AuthSession
	managerSession *domain.AuthSession
	hall           *domain.Hall
	table          *domain.Table
	menuItem       *domain.MenuItem
	clientID       string
	archiveDir     string
}

const bootstrapDeviceID = "bootstrap-device"
const testClientDeviceID = "client-device-1"

func seedMeta(deviceID string) app.CommandMeta {
	return app.CommandMeta{DeviceID: deviceID, Origin: app.OriginSystemSeed}
}

func edgeMeta(deviceID string) app.CommandMeta {
	return app.CommandMeta{NodeDeviceID: deviceID, DeviceID: deviceID, Origin: app.OriginEdgeDevice}
}

func (f *fixture) edgeMeta() app.CommandMeta {
	meta := edgeMeta(f.device.ID)
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = f.employee.ID
	meta.SessionID = f.session.ID
	return meta
}

func (f *fixture) edgeMetaCommand(commandID string) app.CommandMeta {
	meta := f.edgeMeta()
	meta.CommandID = commandID
	return meta
}

func (f *fixture) managerEdgeMetaCommand(t *testing.T, commandID string) app.CommandMeta {
	t.Helper()
	if f.managerSession == nil {
		login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
			CommandMeta: app.CommandMeta{
				CommandID:      "cmd-login-manager-" + commandID,
				NodeDeviceID:   f.device.ID,
				DeviceID:       f.device.ID,
				ClientDeviceID: f.clientID,
				Origin:         app.OriginEdgeDevice,
			},
			PIN: "2468",
		})
		if err != nil {
			t.Fatal(err)
		}
		f.managerSession = &login.Session
	}
	return f.managerMetaCommand(commandID)
}

func (f *fixture) managerMetaCommand(commandID string) app.CommandMeta {
	meta := edgeMeta(f.device.ID)
	meta.CommandID = commandID
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = f.manager.ID
	meta.SessionID = f.managerSession.ID
	return meta
}

func (f *fixture) cancelPrecheckCommand(commandID, precheckID string) app.CancelPrecheckCommand {
	return app.CancelPrecheckCommand{
		CommandMeta:        f.managerMetaCommand(commandID),
		PrecheckID:         precheckID,
		ManagerPIN:         "2468",
		CancellationReason: "guest changed order",
	}
}

func testPINHash(t *testing.T, pin, salt string) string {
	t.Helper()
	hash, err := appshared.HashPIN(pin, []byte(salt))
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	rootDir := t.TempDir()
	dbPath := filepath.Join(rootDir, "pos.db")
	archiveDir := filepath.Join(rootDir, "archives")
	db, err := platformsqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := platformsqlite.MigrateDir(ctx, db, filepath.Join("..", "..", "..", "migrations", "sqlite")); err != nil {
		t.Fatal(err)
	}
	repo := possqlite.NewRepository(db)
	service := app.NewServiceWithOptions(repo, platformsqlite.NewTxManager(db), &testIDs{}, fixedClock{}, app.ServiceOptions{StorageArchiveDir: archiveDir})
	f := &fixture{ctx: ctx, db: db, repo: repo, service: service, archiveDir: archiveDir}
	f.clientID = testClientDeviceID
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
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     seedMeta(bootstrapDeviceID),
		Name:            string(appshared.RoleCashier),
		PermissionsJSON: appshared.RolePermissionsJSON(appshared.RoleCashier),
	})
	if err != nil {
		t.Fatal(err)
	}
	managerRole, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     seedMeta(bootstrapDeviceID),
		Name:            string(appshared.RoleManager),
		PermissionsJSON: appshared.RolePermissionsJSON(appshared.RoleManager),
	})
	if err != nil {
		t.Fatal(err)
	}
	f.device, err = f.service.RegisterDevice(f.ctx, app.RegisterDeviceCommand{CommandMeta: seedMeta(bootstrapDeviceID), RestaurantID: f.restaurant.ID, DeviceCode: "POS-1", Name: "Main", Type: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.PairEdgeNode(f.ctx, app.PairEdgeNodeCommand{PairingCode: "MHPOS:" + f.restaurant.ID + ":" + f.device.ID}); err != nil {
		t.Fatal(err)
	}
	f.employee, err = f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, RoleID: role.ID, Name: "Anna", PINHash: testPINHash(t, "1111", "cashier-salt")})
	if err != nil {
		t.Fatal(err)
	}
	f.manager, err = f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, RoleID: managerRole.ID, Name: "Mira", PINHash: testPINHash(t, "2468", "manager-salt")})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{CommandMeta: app.CommandMeta{CommandID: "cmd-seed-login", NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, Origin: app.OriginEdgeDevice}, PIN: "1111"})
	if err != nil {
		t.Fatal(err)
	}
	f.session = &login.Session
	managerLogin, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{CommandMeta: app.CommandMeta{CommandID: "cmd-seed-login-manager", NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, Origin: app.OriginEdgeDevice}, PIN: "2468"})
	if err != nil {
		t.Fatal(err)
	}
	f.managerSession = &managerLogin.Session
	f.hall, err = f.service.CreateHall(f.ctx, app.CreateHallCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	f.table, err = f.service.CreateTable(f.ctx, app.CreateTableCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, HallID: f.hall.ID, Name: "A1", Seats: 2})
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

func TestPairEdgeNodeStoresKeyedPairingVerifier(t *testing.T) {
	f := newFixture(t)
	var verifier string
	if err := f.db.QueryRowContext(f.ctx, `SELECT pairing_code_hash FROM edge_node_identity WHERE id = 'local'`).Scan(&verifier); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(verifier, "pairing.hmac-sha256.v1:") {
		t.Fatalf("expected keyed pairing verifier format, got %q", verifier)
	}
	if strings.HasPrefix(verifier, "sha256:") {
		t.Fatalf("expected pairing verifier not to use plain sha256 format, got %q", verifier)
	}
	if strings.Contains(verifier, "MHPOS:") || strings.Contains(verifier, f.restaurant.ID) || strings.Contains(verifier, f.device.ID) {
		t.Fatalf("expected pairing verifier not to expose raw pairing payload, got %q", verifier)
	}
}

func TestMaintainCloudProvisioningSkipsCloudRegistrationWhenAlreadyPaired(t *testing.T) {
	f := newFixture(t)
	cloud := &countingProvisioningCloud{}
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 9000}, fixedClock{}, app.ServiceOptions{
		CloudProvisioningURL:    "http://cloud.example",
		CloudProvisioningClient: cloud,
	})

	status, err := service.MaintainCloudProvisioning(f.ctx, app.RegisterCloudProvisioningCommand{
		CloudURL:    "http://cloud.example",
		DisplayName: "POS Terminal",
		AppVersion:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != domain.ProvisioningPaired || !status.Paired {
		t.Fatalf("expected paired provisioning status, got %+v", status)
	}
	if cloud.registerCalls != 0 || cloud.statusCalls != 0 || cloud.snapshotCalls != 0 {
		t.Fatalf("expected paired maintenance not to call Cloud, got register=%d status=%d snapshot=%d", cloud.registerCalls, cloud.statusCalls, cloud.snapshotCalls)
	}
}

func TestProvisioningPollAndLicensePairAreIdempotentWhenAlreadyPaired(t *testing.T) {
	f := newFixture(t)
	cloud := &countingProvisioningCloud{}
	license := &countingProvisioningLicense{}
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 9100}, fixedClock{}, app.ServiceOptions{
		CloudProvisioningURL:      "http://cloud.example",
		CloudProvisioningClient:   cloud,
		LicenseServerURL:          "http://license.example",
		LicenseProvisioningClient: license,
	})
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	polled, err := service.PollCloudAssignment(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	paired, err := service.PairViaLicense(f.ctx, app.PairViaLicenseCommand{PairingCode: "MHPOS:" + f.restaurant.ID + ":" + f.device.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !polled.Paired || polled.Status != domain.ProvisioningPaired || !paired.Paired || paired.Status != domain.ProvisioningPaired {
		t.Fatalf("expected paired idempotent status, poll=%+v pair=%+v", polled, paired)
	}
	if cloud.registerCalls != 0 || cloud.statusCalls != 0 || cloud.snapshotCalls != 0 || license.resolveCalls != 0 {
		t.Fatalf("expected paired provisioning not to call external services, cloud=%+v license=%+v", cloud, license)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected paired provisioning not to create outbox rows, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected paired provisioning not to create local events, before=%d after=%d", eventsBefore, events)
	}
}

func TestPairEdgeNodeIsIdempotentForCurrentIdentity(t *testing.T) {
	f := newFixture(t)
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	identity, err := f.service.PairEdgeNode(f.ctx, app.PairEdgeNodeCommand{PairingCode: "MHPOS:" + f.restaurant.ID + ":" + f.device.ID})
	if err != nil {
		t.Fatal(err)
	}
	if identity.NodeDeviceID != f.device.ID || identity.RestaurantID != f.restaurant.ID {
		t.Fatalf("unexpected idempotent identity: %+v", identity)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected repeated pairing not to create outbox rows, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected repeated pairing not to create local events, before=%d after=%d", eventsBefore, events)
	}
}

func TestPinLoginRejectsDuplicateActivePIN(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       f.employee.RoleID,
		Name:         "Duplicate Cashier",
		PINHash:      testPINHash(t, "1111", "duplicate-cashier-salt"),
	}); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-duplicate-pin",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "1111",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected duplicate active PIN login to conflict, got %v", err)
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

func (f *fixture) openCashSession(t *testing.T) *domain.CashSession {
	t.Helper()
	session, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func (f *fixture) closeCashSessionAndShift(t *testing.T, shift *domain.Shift, cashSession *domain.CashSession, suffix string) {
	t.Helper()
	if _, err := f.service.CloseCashSession(f.ctx, app.CloseCashSessionCommand{
		CommandMeta:        f.managerEdgeMetaCommand(t, "cmd-close-cash-"+suffix),
		ID:                 cashSession.ID,
		ClosedByEmployeeID: f.manager.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{
		CommandMeta:        f.edgeMetaCommand("cmd-close-shift-" + suffix),
		ID:                 shift.ID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) createPaidOrder(t *testing.T) (*domain.Order, *domain.Check) {
	t.Helper()
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	return order, check
}

func makeCheckOlderThanArchiveCutoff(t *testing.T, f *fixture, checkID string) {
	t.Helper()
	execTestSQL(t, f, `UPDATE checks SET business_date_local = '2026-05-03' WHERE id = ?`, checkID)
	execTestSQL(t, f, `
UPDATE payments
SET business_date_local = '2026-05-03'
WHERE precheck_id = (
  SELECT p.id
  FROM prechecks p
  JOIN checks c ON c.order_id = p.order_id
  WHERE c.id = ?
  LIMIT 1
)`, checkID)
}

func insertClosedOrderFixture(t *testing.T, f *fixture, shiftID, deviceID, orderID, checkID, businessDate string, closedAt time.Time) {
	t.Helper()
	openedAt := closedAt.Add(-30 * time.Minute)
	execTestSQL(t, f, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		orderID, "edge-"+orderID, f.restaurant.ID, deviceID, shiftID, string(domain.OrderClosed), f.table.ID, f.table.Name, 1, appshared.DBTime(openedAt), appshared.DBTime(closedAt), appshared.DBTime(openedAt), appshared.DBTime(closedAt))
	execTestSQL(t, f, `INSERT INTO checks(id,order_id,status,currency_code,subtotal,discount_total,surcharge_total,tax_total,total,paid_total,remaining_total,business_date_local,closed_at,snapshot,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		checkID, orderID, string(domain.CheckPaid), "RUB", int64(1000), int64(0), int64(0), int64(0), int64(1000), int64(1000), int64(0), businessDate, appshared.DBTime(closedAt), `{"document_type":"check","precheck_snapshot":{"lines":[]}}`, appshared.DBTime(closedAt), appshared.DBTime(closedAt))
}

func countRows(t *testing.T, f *fixture, table string) int {
	t.Helper()
	switch table {
	case "restaurants", "devices", "orders", "order_lines", "order_line_modifiers", "prechecks", "checks", "payments", "payment_attempts", "cash_sessions", "cash_drawer_events", "pos_sync_outbox", "local_event_log", "manager_override_audit", "roles", "employees", "catalog_items", "catalog_folders", "catalog_tags", "catalog_item_tags", "modifier_groups", "modifier_options", "modifier_group_bindings", "menu_item_modifier_groups", "menu_items", "tax_profiles", "tax_rules", "service_charge_rules", "pricing_policies", "auth_sessions", "halls", "tables", "cloud_master_sync_state", "order_line_discounts", "order_surcharges", "precheck_lines", "precheck_line_modifiers", "precheck_discounts", "precheck_surcharges", "precheck_taxes", "financial_operations", "financial_operation_items":
	default:
		t.Fatalf("unexpected table %q", table)
	}
	var n int
	if err := f.db.QueryRowContext(f.ctx, fmt.Sprintf("SELECT COUNT(1) FROM %s", table)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func countRowsWhere(t *testing.T, f *fixture, table, where string, args ...any) int {
	t.Helper()
	switch table {
	case "orders", "checks", "financial_operations", "financial_operation_items", "pos_sync_outbox", "local_event_log":
	default:
		t.Fatalf("unexpected table %q", table)
	}
	var n int
	if err := f.db.QueryRowContext(f.ctx, fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE %s", table, where), args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func execTestSQL(t *testing.T, f *fixture, query string, args ...any) {
	t.Helper()
	if _, err := f.db.ExecContext(f.ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}

func markEdgeOutboxSent(t *testing.T, f *fixture) {
	t.Helper()
	execTestSQL(t, f, `UPDATE pos_sync_outbox SET status = 'sent', sent_at = COALESCE(sent_at, updated_at), locked_at = NULL, locked_by = NULL WHERE sync_direction = 'edge_to_cloud'`)
}

func insertStopList(t *testing.T, f *fixture, id, catalogItemID string, availableQuantity *float64, active bool) {
	t.Helper()
	activeValue := 0
	if active {
		activeValue = 1
	}
	var available any
	if availableQuantity != nil {
		available = *availableQuantity
	}
	execTestSQL(t, f, `INSERT INTO stop_lists(id,restaurant_id,catalog_item_id,available_quantity,source,reason,active,cloud_version,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		id, f.restaurant.ID, catalogItemID, available, "test", "fixture", activeValue, int64(1), appshared.DBTime(fixedClock{}.Now()))
}

func insertRecipe(t *testing.T, f *fixture, recipeID, ownerCatalogItemID, componentCatalogItemID string) {
	t.Helper()
	execTestSQL(t, f, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at,cloud_version) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		recipeID, ownerCatalogItemID, 1, "Fixture recipe", string(domain.RecipeVersionActive), 1, "portion", 1, appshared.DBTime(fixedClock{}.Now()), appshared.DBTime(fixedClock{}.Now()), int64(1))
	execTestSQL(t, f, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at,cloud_version) VALUES (?,?,?,?,?,?,?,?,?)`,
		recipeID+"-line", recipeID, componentCatalogItemID, 100, "g", 0, appshared.DBTime(fixedClock{}.Now()), appshared.DBTime(fixedClock{}.Now()), int64(1))
}

func floatPtr(v float64) *float64 {
	return &v
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
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

func outboxIDs(t *testing.T, f *fixture, n int) []string {
	t.Helper()
	rows, err := f.db.QueryContext(f.ctx, `SELECT id FROM pos_sync_outbox ORDER BY sequence_no LIMIT ?`, n)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ids) != n {
		t.Fatalf("expected %d outbox ids, got %d", n, len(ids))
	}
	return ids
}

func outboxStatusAttempts(t *testing.T, f *fixture, id string) (domain.OutboxStatus, int) {
	t.Helper()
	var status string
	var attempts int
	if err := f.db.QueryRowContext(f.ctx, `SELECT status, attempts FROM pos_sync_outbox WHERE id = ?`, id).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	return domain.OutboxStatus(status), attempts
}

func TestRetryFailedOutboxResetsFailedAndSuspendedButNotSent(t *testing.T) {
	f := newFixture(t)
	ids := outboxIDs(t, f, 3)
	now := appshared.DBTime(fixedClock{}.Now())
	_, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = 2, last_error = 'temporary', updated_at = ? WHERE id = ?`, now, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'suspended', attempts = 4, last_error = 'threshold', updated_at = ? WHERE id = ?`, now, ids[1])
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'sent', sent_at = ?, updated_at = ? WHERE id = ?`, now, now, ids[2])
	if err != nil {
		t.Fatal(err)
	}

	retried, err := f.service.RetryFailedOutbox(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if retried != 2 {
		t.Fatalf("expected 2 retried messages, got %d", retried)
	}
	for _, id := range ids[:2] {
		status, attempts := outboxStatusAttempts(t, f, id)
		if status != domain.OutboxPending || attempts != 0 {
			t.Fatalf("expected %s to be pending with attempts=0, got status=%s attempts=%d", id, status, attempts)
		}
	}
	status, _ := outboxStatusAttempts(t, f, ids[2])
	if status != domain.OutboxSent {
		t.Fatalf("expected sent outbox row to stay sent, got %s", status)
	}
}

func TestRetryFailedOutboxAsOperatorRequiresPermission(t *testing.T) {
	f := newFixture(t)
	ids := outboxIDs(t, f, 2)
	now := appshared.DBTime(fixedClock{}.Now())
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = 1, last_error = 'temporary', updated_at = ? WHERE id = ?`, now, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'suspended', attempts = 2, last_error = 'threshold', updated_at = ? WHERE id = ?`, now, ids[1]); err != nil {
		t.Fatal(err)
	}

	_, err := f.service.RetryFailedOutboxAsOperator(f.ctx, f.edgeMetaCommand("cmd-retry-cashier-denied"))
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier retry, got %v", err)
	}
	for _, id := range ids {
		status, _ := outboxStatusAttempts(t, f, id)
		if status == domain.OutboxPending {
			t.Fatalf("expected retry status to remain unchanged for %s", id)
		}
	}

	retried, err := f.service.RetryFailedOutboxAsOperator(f.ctx, f.managerEdgeMetaCommand(t, "cmd-retry-manager-allow"))
	if err != nil {
		t.Fatal(err)
	}
	if retried != 2 {
		t.Fatalf("expected 2 retried messages for manager, got %d", retried)
	}
}

func TestGetSyncStatusAsOperatorRequiresPermission(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.GetSyncStatusAsOperator(f.ctx, f.edgeMetaCommand("cmd-sync-status-cashier-denied")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier sync status view, got %v", err)
	}
	if _, err := f.service.GetSyncStatusAsOperator(f.ctx, f.managerEdgeMetaCommand(t, "cmd-sync-status-manager-allow")); err != nil {
		t.Fatalf("expected manager sync status access, got %v", err)
	}
}

func TestListOutboxAsOperatorRequiresPermission(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.ListOutboxAsOperator(f.ctx, f.edgeMetaCommand("cmd-list-outbox-cashier-denied"), 10); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier outbox view, got %v", err)
	}
	items, err := f.service.ListOutboxAsOperator(f.ctx, f.managerEdgeMetaCommand(t, "cmd-list-outbox-manager-allow"), 10)
	if err != nil {
		t.Fatalf("expected manager outbox access, got %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected non-empty outbox list for manager")
	}
}

func TestListLocalEventsAsOperatorRequiresPermission(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.ListLocalEventsAsOperator(f.ctx, f.edgeMetaCommand("cmd-local-events-cashier-denied"), app.ListLocalEventsQuery{Limit: 5}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier local events access, got %v", err)
	}
	items, err := f.service.ListLocalEventsAsOperator(f.ctx, f.managerEdgeMetaCommand(t, "cmd-local-events-manager-allow"), app.ListLocalEventsQuery{Limit: 5})
	if err != nil {
		t.Fatalf("expected manager local events access, got %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected non-empty local events list for manager")
	}
}

func TestStorageLifecycleStatusRequiresPermissionAndSummarizesRuntimeState(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.GetStorageLifecycleStatus(f.ctx, app.StorageStatusCommand{CommandMeta: f.edgeMetaCommand("cmd-storage-status-cashier-denied")}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier storage status, got %v", err)
	}

	shift := f.openShift(t)
	closedAt := fixedClock{}.Now().Add(-48 * time.Hour)
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "order-old-storage", "check-old-storage", "2026-05-02", closedAt)
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "order-new-storage", "check-new-storage", "2026-05-04", fixedClock{}.Now())

	status, err := f.service.GetStorageLifecycleStatus(f.ctx, app.StorageStatusCommand{CommandMeta: f.managerEdgeMetaCommand(t, "cmd-storage-status-manager-allow")})
	if err != nil {
		t.Fatal(err)
	}
	if status.SQLite.PageCount <= 0 || status.SQLite.PageSizeBytes <= 0 || status.SQLite.EstimatedSizeBytes <= 0 {
		t.Fatalf("expected sqlite page stats, got %+v", status.SQLite)
	}
	if status.Tables.ClosedOrders < 2 || status.Tables.Checks < 2 {
		t.Fatalf("expected closed order/check counts, got %+v", status.Tables)
	}
	if status.ClosedOrderBusinessDateRange.Oldest != "2026-05-02" || status.ClosedOrderBusinessDateRange.Newest != "2026-05-04" {
		t.Fatalf("unexpected closed date range: %+v", status.ClosedOrderBusinessDateRange)
	}
	if len(status.ClosedOrdersByBusinessDate) < 2 {
		t.Fatalf("expected closed orders grouped by business date, got %+v", status.ClosedOrdersByBusinessDate)
	}
	if status.Retention.Mode != "archive_apply_supported" || !status.Retention.DestructiveApplySupported {
		t.Fatalf("expected archive apply retention capability, got %+v", status.Retention)
	}
}

func TestStorageRetentionDryRunCountsEligibleRowsWithoutMutatingProtectedTables(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	if _, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-storage-cancel-for-dry-run"),
		CheckID:              check.ID,
		Reason:               "guest returned immediately",
		InventoryDisposition: domain.InventoryNoStockEffect,
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   check.Total,
			Currency: check.CurrencyCode,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	activeOrder, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-storage-active-order"), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}

	beforeOrders := countRows(t, f, "orders")
	beforeChecks := countRows(t, f, "checks")
	beforeFinancialOperations := countRows(t, f, "financial_operations")
	beforeFinancialItems := countRows(t, f, "financial_operation_items")
	beforeOutbox := countRows(t, f, "pos_sync_outbox")

	result, err := f.service.DryRunStorageRetention(f.ctx, app.RetentionDryRunCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-retention-dry-run"),
		CutoffBusinessDateLocal: "2026-05-04",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || result.Mode != "dry_run_only" || result.ResultMode != "dry_run_only" || result.DestructiveApplySupported {
		t.Fatalf("expected blocked dry-run-only result, got %+v", result)
	}
	if !containsString(result.BlockReasons, "dry_run_only_no_archive_policy") ||
		!containsString(result.BlockReasons, "pending_edge_to_cloud_outbox") ||
		!containsString(result.BlockReasons, "active_orders") ||
		!containsString(result.BlockReasons, "open_shifts") ||
		!containsString(result.BlockReasons, "open_cash_sessions") {
		t.Fatalf("expected archive-policy, operational and pending-outbox block reasons, got %+v", result.BlockReasons)
	}
	if result.Eligible.ClosedOrders < 1 || result.Eligible.Checks < 1 || result.Eligible.Prechecks < 1 || result.Eligible.Payments < 1 {
		t.Fatalf("expected eligible financial documents for order %s, got %+v", order.ID, result.Eligible)
	}
	if result.Eligible.FinancialOperations < 1 || result.Eligible.FinancialOperationItems < 1 {
		t.Fatalf("expected eligible financial operation rows, got %+v", result.Eligible)
	}
	if result.ActiveOrders < 1 || activeOrder.ID == "" {
		t.Fatalf("expected active orders to be reported, result=%+v active=%+v", result, activeOrder)
	}
	if !result.FinancialLedgerProtected || !result.ImmutableSnapshotsProtected {
		t.Fatalf("expected protected ledger/snapshot flags, got %+v", result)
	}
	if got := countRows(t, f, "orders"); got != beforeOrders {
		t.Fatalf("dry-run mutated orders, before=%d after=%d", beforeOrders, got)
	}
	if got := countRows(t, f, "checks"); got != beforeChecks {
		t.Fatalf("dry-run mutated checks, before=%d after=%d", beforeChecks, got)
	}
	if got := countRows(t, f, "financial_operations"); got != beforeFinancialOperations {
		t.Fatalf("dry-run mutated financial_operations, before=%d after=%d", beforeFinancialOperations, got)
	}
	if got := countRows(t, f, "financial_operation_items"); got != beforeFinancialItems {
		t.Fatalf("dry-run mutated financial_operation_items, before=%d after=%d", beforeFinancialItems, got)
	}
	if got := countRows(t, f, "pos_sync_outbox"); got != beforeOutbox {
		t.Fatalf("dry-run mutated outbox, before=%d after=%d", beforeOutbox, got)
	}
}

func TestStorageRetentionDryRunRejectsInvalidAndFutureCutoff(t *testing.T) {
	f := newFixture(t)
	for _, tc := range []struct {
		name   string
		cutoff string
	}{
		{name: "invalid", cutoff: "2026/05/05"},
		{name: "future", cutoff: "2026-05-05"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.service.DryRunStorageRetention(f.ctx, app.RetentionDryRunCommand{
				CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-retention-"+tc.name),
				CutoffBusinessDateLocal: tc.cutoff,
			})
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("expected invalid cutoff error, got %v", err)
			}
		})
	}
}

func TestStorageArchiveExportPlanRequiresPermissionAndRejectsInvalidInput(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.BuildStorageArchiveExportPlan(f.ctx, app.ArchiveExportPlanCommand{
		CommandMeta:             f.edgeMetaCommand("cmd-storage-archive-plan-cashier-denied"),
		CutoffBusinessDateLocal: "2026-05-04",
		Mode:                    "manifest_only",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier archive plan, got %v", err)
	}

	_, err = f.service.BuildStorageArchiveExportPlan(f.ctx, app.ArchiveExportPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-plan-invalid-cutoff"),
		CutoffBusinessDateLocal: "2026/05/05",
		Mode:                    "manifest_only",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid cutoff error, got %v", err)
	}

	_, err = f.service.BuildStorageArchiveExportPlan(f.ctx, app.ArchiveExportPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-plan-future-cutoff"),
		CutoffBusinessDateLocal: "2026-05-05",
		Mode:                    "manifest_only",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected future cutoff error, got %v", err)
	}

	_, err = f.service.BuildStorageArchiveExportPlan(f.ctx, app.ArchiveExportPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-plan-invalid-mode"),
		CutoffBusinessDateLocal: "2026-05-04",
		Mode:                    "delete",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid mode error, got %v", err)
	}
}

func TestStorageArchiveExportPlanCountsProtectedRowsBlocksOutboxAndDoesNotMutate(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	if _, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-storage-archive-plan-cancel"),
		CheckID:              check.ID,
		Reason:               "guest returned immediately",
		InventoryDisposition: domain.InventoryNoStockEffect,
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   check.Total,
			Currency: check.CurrencyCode,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	activeOrder, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{
		CommandMeta: f.edgeMetaCommand("cmd-storage-archive-plan-active-order"),
		TableID:     f.table.ID,
		TableName:   "A1",
		GuestCount:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]int{
		"orders":                    countRows(t, f, "orders"),
		"checks":                    countRows(t, f, "checks"),
		"prechecks":                 countRows(t, f, "prechecks"),
		"payments":                  countRows(t, f, "payments"),
		"financial_operations":      countRows(t, f, "financial_operations"),
		"financial_operation_items": countRows(t, f, "financial_operation_items"),
		"local_event_log":           countRows(t, f, "local_event_log"),
		"pos_sync_outbox":           countRows(t, f, "pos_sync_outbox"),
	}

	result, err := f.service.BuildStorageArchiveExportPlan(f.ctx, app.ArchiveExportPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-plan"),
		CutoffBusinessDateLocal: "2026-05-04",
		Mode:                    "manifest_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "manifest_only" || result.ResultMode != "plan_only" || !result.Blocked || result.DestructiveApplySupported {
		t.Fatalf("unexpected archive plan flags: %+v", result)
	}
	if !containsString(result.BlockReasons, "dry_run_only_no_archive_policy") ||
		!containsString(result.BlockReasons, "pending_edge_to_cloud_outbox") ||
		!containsString(result.BlockReasons, "active_orders") ||
		!containsString(result.BlockReasons, "open_shifts") ||
		!containsString(result.BlockReasons, "open_cash_sessions") {
		t.Fatalf("expected dry-run, operational and outbox block reasons, got %+v", result.BlockReasons)
	}
	if result.ArchiveSet.ClosedOrders != 1 || result.ArchiveSet.Checks != 1 || result.ArchiveSet.Prechecks != 1 ||
		result.ArchiveSet.Payments != 1 || result.ArchiveSet.OrderLines != 1 {
		t.Fatalf("expected one eligible closed order graph for %s, got %+v", order.ID, result.ArchiveSet)
	}
	if result.ArchiveSet.FinancialOperations != 1 || result.ArchiveSet.FinancialOperationItems != 1 {
		t.Fatalf("expected eligible financial operation rows, got %+v", result.ArchiveSet)
	}
	if !result.Protected.FinancialLedgerProtected || !result.Protected.ImmutableSnapshotsProtected ||
		!result.Protected.LocalEventsProtected || !result.Protected.OutboxProtected {
		t.Fatalf("expected protected flags, got %+v", result.Protected)
	}
	if result.ActiveOrders < 1 || result.OpenShifts < 1 || result.OpenCashSessions < 1 {
		t.Fatalf("expected operational blockers in archive plan, got active=%d shifts=%d cash=%d", result.ActiveOrders, result.OpenShifts, result.OpenCashSessions)
	}
	if result.Manifest.FormatVersion != "storage-archive-manifest-v1" ||
		result.Manifest.CutoffBusinessDateLocal != "2026-05-04" || result.Manifest.RestaurantID != f.restaurant.ID {
		t.Fatalf("unexpected manifest metadata: %+v", result.Manifest)
	}
	if result.Manifest.BusinessDateRange.Oldest != "2026-05-03" || result.Manifest.BusinessDateRange.Newest != "2026-05-03" {
		t.Fatalf("unexpected manifest business date range: %+v", result.Manifest.BusinessDateRange)
	}
	if len(result.Manifest.Tables) != 14 || result.Manifest.Tables[0].Name != "orders" ||
		result.Manifest.Tables[0].Rows != 1 ||
		result.Manifest.Tables[len(result.Manifest.Tables)-1].Name != "financial_operation_items" {
		t.Fatalf("unexpected deterministic table manifest: %+v", result.Manifest.Tables)
	}
	if activeOrder.ID == "" {
		t.Fatal("expected active order fixture")
	}
	for table, want := range before {
		if got := countRows(t, f, table); got != want {
			t.Fatalf("archive export-plan mutated %s, before=%d after=%d", table, want, got)
		}
	}
}

func TestStorageArchiveExportRejectsInvalidAndFutureCutoff(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-invalid"),
		CutoffBusinessDateLocal: "2026/05/04",
		Reason:                  "invalid format",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid cutoff error, got %v", err)
	}
	_, err = f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-future"),
		CutoffBusinessDateLocal: "2026-05-05",
		Reason:                  "future cutoff",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected future cutoff error, got %v", err)
	}
}

func TestStorageArchiveExportCreatesEmptyNoopArtifact(t *testing.T) {
	f := newFixture(t)
	result, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-empty"),
		CutoffBusinessDateLocal: "2026-05-03",
		Reason:                  "operator empty export check",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "export_only" || !result.ExportCreated || result.Blocked || !result.DestructiveApplySupported {
		t.Fatalf("unexpected empty archive result: %+v", result)
	}
	if result.Counts.ArchivedRows != 0 || result.Counts.ClosedOrders != 0 || result.BusinessDateRange.Oldest != "" || result.BusinessDateRange.Newest != "" {
		t.Fatalf("expected empty export scope, got counts=%+v range=%+v", result.Counts, result.BusinessDateRange)
	}
	rawArchive, err := os.ReadFile(result.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rawArchive) != 0 {
		t.Fatalf("expected empty JSONL archive, got %q", string(rawArchive))
	}
	if result.SHA256 != hex.EncodeToString(sha256.New().Sum(nil)) {
		t.Fatalf("unexpected empty archive sha: %s", result.SHA256)
	}
}

func TestStorageArchiveCutoffIsExclusiveAcrossPlanExportAndApplyPlan(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	insertClosedOrderFixture(
		t, f, shift.ID, f.device.ID,
		"archive-old-order", "archive-old-check",
		"2026-05-03", fixedClock{}.Now().Add(-24*time.Hour),
	)
	insertClosedOrderFixture(
		t, f, shift.ID, f.device.ID,
		"archive-boundary-order", "archive-boundary-check",
		"2026-05-04", fixedClock{}.Now(),
	)

	dryRun, err := f.service.DryRunStorageRetention(f.ctx, app.RetentionDryRunCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-exclusive-dry-run"),
		CutoffBusinessDateLocal: "2026-05-04",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Eligible.ClosedOrders != 1 || dryRun.Eligible.Checks != 1 {
		t.Fatalf("expected dry-run cutoff to be exclusive, got %+v", dryRun.Eligible)
	}

	plan, err := f.service.BuildStorageArchiveExportPlan(f.ctx, app.ArchiveExportPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-exclusive-export-plan"),
		CutoffBusinessDateLocal: "2026-05-04",
		Mode:                    "manifest_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ArchiveSet.ClosedOrders != 1 ||
		plan.Manifest.BusinessDateRange.Oldest != "2026-05-03" ||
		plan.Manifest.BusinessDateRange.Newest != "2026-05-03" {
		t.Fatalf("expected export-plan cutoff to be exclusive, got archive_set=%+v range=%+v", plan.ArchiveSet, plan.Manifest.BusinessDateRange)
	}

	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-exclusive-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "exclusive cutoff fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if exported.Counts.ClosedOrders != 1 ||
		exported.Counts.Checks != 1 ||
		exported.BusinessDateRange.Oldest != "2026-05-03" ||
		exported.BusinessDateRange.Newest != "2026-05-03" {
		t.Fatalf("expected export cutoff to be exclusive, got counts=%+v range=%+v", exported.Counts, exported.BusinessDateRange)
	}
	rawArchive, err := os.ReadFile(exported.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawArchive), "archive-old-order") || strings.Contains(string(rawArchive), "archive-boundary-order") {
		t.Fatalf("archive cutoff must include old order and exclude boundary order: %s", string(rawArchive))
	}

	applyPlan, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-exclusive-apply-plan"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if applyPlan.EligibleCounts.ClosedOrders != 1 ||
		applyPlan.ArchiveCounts.ClosedOrders != 1 ||
		containsString(applyPlan.BlockReasons, "archive_counts_mismatch") {
		t.Fatalf("expected apply-plan runtime/archive counts to use exclusive cutoff, got %+v", applyPlan)
	}
}

func TestStorageArchiveExportIncludesClosedOrderGraphLedgerAndManifestWithoutMutatingSource(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	var checkSnapshotBefore string
	if err := f.db.QueryRowContext(f.ctx, `SELECT snapshot FROM checks WHERE id = ?`, check.ID).Scan(&checkSnapshotBefore); err != nil {
		t.Fatal(err)
	}
	operation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-storage-archive-cancel"),
		CheckID:              check.ID,
		Reason:               "guest returned immediately",
		InventoryDisposition: domain.InventoryNoStockEffect,
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   check.Total,
			Currency: check.CurrencyCode,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)

	before := map[string]int{
		"orders":                    countRows(t, f, "orders"),
		"order_lines":               countRows(t, f, "order_lines"),
		"prechecks":                 countRows(t, f, "prechecks"),
		"checks":                    countRows(t, f, "checks"),
		"payments":                  countRows(t, f, "payments"),
		"financial_operations":      countRows(t, f, "financial_operations"),
		"financial_operation_items": countRows(t, f, "financial_operation_items"),
		"pos_sync_outbox":           countRows(t, f, "pos_sync_outbox"),
		"local_event_log":           countRows(t, f, "local_event_log"),
	}

	result, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-archive-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "operator requested export before pilot cleanup policy",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "export_only" || result.ResultMode != "export_only" || !result.DestructiveApplySupported || result.RuntimeRowsDeleted || !result.ExportCreated {
		t.Fatalf("unexpected archive mode flags: %+v", result)
	}
	if !result.FinancialLedgerProtected || !result.ImmutableSnapshotsProtected {
		t.Fatalf("expected protected flags, got %+v", result)
	}
	if !containsString(result.BlockReasons, "pending_edge_to_cloud_outbox") {
		t.Fatalf("unexpected block reasons: %+v", result.BlockReasons)
	}
	if result.Counts.ClosedOrders != 1 || result.Counts.Checks != 1 || result.Counts.Prechecks != 1 || result.Counts.Payments != 1 || result.Counts.OrderLines != 1 {
		t.Fatalf("expected closed order/precheck/check/payment graph counts, got %+v", result.Counts)
	}
	if result.Counts.FinancialOperations != 1 || result.Counts.FinancialOperationItems != 1 || result.Counts.OutboxMessageReferences == 0 || result.Counts.LocalEventReferences == 0 || result.Counts.BlockingOutboxMessages == 0 {
		t.Fatalf("expected ledger and sync references, got %+v", result.Counts)
	}
	if result.BusinessDateRange.Oldest != "2026-05-03" || result.BusinessDateRange.Newest != "2026-05-03" {
		t.Fatalf("unexpected archive business date range: %+v", result.BusinessDateRange)
	}
	if result.Source.SourceNodeDeviceID != f.device.ID || result.Source.SourceDeviceCode != f.device.DeviceCode {
		t.Fatalf("expected source node metadata in archive manifest, got %+v", result.Source)
	}

	archiveRaw, err := os.ReadFile(result.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(archiveRaw)
	if result.SHA256 != hex.EncodeToString(hash[:]) {
		t.Fatalf("archive sha mismatch: response=%s computed=%s", result.SHA256, hex.EncodeToString(hash[:]))
	}
	var manifest domain.StorageArchiveManifest
	manifestRaw, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SHA256 != result.SHA256 || manifest.Counts.ArchivedRows != result.Counts.ArchivedRows ||
		manifest.CutoffBusinessDateLocal != "2026-05-04" || manifest.RuntimeRowsDeleted {
		t.Fatalf("manifest mismatch: manifest=%+v result=%+v", manifest, result)
	}
	byTable := decodeArchiveJSONLByTable(t, archiveRaw)
	if byTable["orders"] != result.Counts.ClosedOrders || byTable["checks"] != result.Counts.Checks || byTable["financial_operations"] != result.Counts.FinancialOperations || byTable["financial_operation_items"] != result.Counts.FinancialOperationItems {
		t.Fatalf("archive JSONL counts do not match manifest counts: byTable=%+v result=%+v", byTable, result.Counts)
	}
	if byTable["local_event_log_summary"] != result.Counts.LocalEventReferences || byTable["pos_sync_outbox_summary"] != result.Counts.OutboxMessageReferences {
		t.Fatalf("expected summary reference counts to match archive lines: byTable=%+v result=%+v", byTable, result.Counts)
	}
	if strings.Contains(string(archiveRaw), "payload_json") {
		t.Fatalf("expected local event/outbox payload_json to be omitted from archive summaries")
	}
	if !strings.Contains(string(archiveRaw), operation.ID) || !strings.Contains(string(archiveRaw), order.ID) {
		t.Fatalf("expected archive to include order %s and financial operation %s", order.ID, operation.ID)
	}

	for table, want := range before {
		if got := countRows(t, f, table); got != want {
			t.Fatalf("archive export mutated %s, before=%d after=%d", table, want, got)
		}
	}
	var checkSnapshotAfter string
	if err := f.db.QueryRowContext(f.ctx, `SELECT snapshot FROM checks WHERE id = ?`, check.ID).Scan(&checkSnapshotAfter); err != nil {
		t.Fatal(err)
	}
	if checkSnapshotAfter != checkSnapshotBefore {
		t.Fatalf("archive export mutated check snapshot")
	}
}

func TestStorageArchiveApplyPlanWithoutManifestBlocksAndDoesNotMutate(t *testing.T) {
	f := newFixture(t)
	before := protectedStorageCounts(t, f)
	result, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-missing-manifest"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             filepath.Join(f.archiveDir, "missing", "archive.jsonl"),
		ManifestPath:            filepath.Join(f.archiveDir, "missing", "manifest.json"),
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || result.ResultMode != "apply_blocked" || !result.DestructiveApplySupported || result.RuntimeRowsDeleted {
		t.Fatalf("expected blocked apply plan, got %+v", result)
	}
	if !containsString(result.BlockReasons, "archive_manifest_missing") {
		t.Fatalf("expected missing manifest/default block reasons, got %+v", result.BlockReasons)
	}
	assertProtectedStorageCounts(t, f, before, "apply-plan without manifest")
}

func TestStorageArchiveApplyReadinessWithoutManifestBlocksAndDoesNotMutate(t *testing.T) {
	f := newFixture(t)
	before := protectedStorageCounts(t, f)
	result, err := f.service.BuildStorageArchiveApplyReadiness(f.ctx, app.ArchiveApplyReadinessCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-readiness-missing-manifest"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             filepath.Join(f.archiveDir, "missing", "archive.jsonl"),
		ManifestPath:            filepath.Join(f.archiveDir, "missing", "manifest.json"),
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResultMode != "apply_readiness_only" || !result.DestructiveApplySupported || result.ReadyForDestructiveApply || result.RuntimeRowsDeleted {
		t.Fatalf("expected blocked readiness, got %+v", result)
	}
	if result.ArchiveVerified || result.ManifestVerified || !result.RuntimeScopeVerified {
		t.Fatalf("expected missing manifest to fail archive verification only, got %+v", result)
	}
	if !containsString(result.BlockReasons, "archive_manifest_missing") {
		t.Fatalf("expected missing manifest/default block reasons, got %+v", result.BlockReasons)
	}
	assertProtectedStorageCounts(t, f, before, "apply-readiness without manifest")
}

func TestStorageArchiveApplyPlanBlocksInvalidAndFutureCutoff(t *testing.T) {
	f := newFixture(t)
	for _, tc := range []struct {
		name   string
		cutoff string
		reason string
	}{
		{name: "invalid", cutoff: "2026/05/04", reason: "invalid_cutoff"},
		{name: "future", cutoff: "2026-05-05", reason: "future_cutoff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
				CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-"+tc.name),
				CutoffBusinessDateLocal: tc.cutoff,
				ArchivePath:             filepath.Join(f.archiveDir, "missing", "archive.jsonl"),
				ManifestPath:            filepath.Join(f.archiveDir, "missing", "manifest.json"),
				Mode:                    "plan_only",
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Blocked || !containsString(result.BlockReasons, tc.reason) {
				t.Fatalf("expected %s block reason, got %+v", tc.reason, result)
			}
		})
	}
}

func TestStorageArchiveApplyPlanVerifiesArchiveSHAAndCounts(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "apply plan fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := appendArchiveLine(exported.ArchivePath, `{"table":"unknown","row":{}}`); err != nil {
		t.Fatal(err)
	}

	result, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-sha"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(result.BlockReasons, "archive_sha_mismatch") {
		t.Fatalf("expected archive_sha_mismatch, got %+v", result.BlockReasons)
	}
	if !result.Verification.ManifestVersionMatched || result.Verification.SHA256Matched {
		t.Fatalf("unexpected verification summary after sha mismatch: %+v", result.Verification)
	}

	manifest := readArchiveManifest(t, exported.ManifestPath)
	manifest.Counts.ClosedOrders++
	writeArchiveManifest(t, exported.ManifestPath, manifest)
	result, err = f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-manifest-counts"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(result.BlockReasons, "archive_manifest_counts_mismatch") {
		t.Fatalf("expected archive_manifest_counts_mismatch, got %+v", result.BlockReasons)
	}
}

func TestStorageArchiveApplyReadinessAggregatesIntegrityAndRuntimeBlockers(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-readiness-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "readiness fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := protectedStorageCounts(t, f)
	result, err := f.service.BuildStorageArchiveApplyReadiness(f.ctx, app.ArchiveApplyReadinessCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-readiness"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ResultMode != "apply_readiness_only" || !result.DestructiveApplySupported || result.ReadyForDestructiveApply || result.RuntimeRowsDeleted {
		t.Fatalf("unexpected readiness mode flags: %+v", result)
	}
	if !result.ArchiveVerified || !result.ManifestVerified || !result.SnapshotPayloadVerified {
		t.Fatalf("expected verified archive scope, got %+v", result)
	}
	if !result.PendingEdgeToCloudOutbox || result.BlockingOutboxCount == 0 || result.OpenOperationalBoundaries.Open {
		t.Fatalf("expected scoped outbox blocker without current-day open boundary blockers, got %+v", result)
	}
	if !result.ProtectedData.FinancialLedgerProtected ||
		!result.ProtectedData.ImmutableSnapshotsProtected ||
		!result.ProtectedData.LocalEventsProtected ||
		!result.ProtectedData.OutboxProtected {
		t.Fatalf("expected protected data flags, got %+v", result.ProtectedData)
	}
	if !containsString(result.BlockReasons, "pending_edge_to_cloud_outbox") ||
		containsString(result.BlockReasons, "open_operational_boundary") {
		t.Fatalf("expected scoped outbox block reason only, got %+v", result.BlockReasons)
	}
	assertProtectedStorageCounts(t, f, before, "apply-readiness")
}

func TestStorageArchiveApplyReadinessAndDestructiveApplyDeletesRuntimeRowsButKeepsArchivePreview(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	operation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-storage-apply-cancel"),
		CheckID:              check.ID,
		Reason:               "guest returned immediately",
		InventoryDisposition: domain.InventoryNoStockEffect,
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   check.Total,
			Currency: check.CurrencyCode,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	markEdgeOutboxSent(t, f)

	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-export-ready"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "destructive apply fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := f.service.BuildStorageArchiveApplyReadiness(f.ctx, app.ArchiveApplyReadinessCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-readiness-ready"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !readiness.ReadyForDestructiveApply || readiness.BlockingOutboxCount != 0 || readiness.OpenOperationalBoundaries.Open {
		t.Fatalf("expected ready destructive apply gate, got %+v", readiness)
	}

	beforeOrders := countRows(t, f, "orders")
	beforeFinancialOperations := countRows(t, f, "financial_operations")
	beforeSpecificOrder := countRowsWhere(t, f, "orders", "id = ?", order.ID)
	t.Logf("DEBUG before apply: total_orders=%d specific_order_rows=%d", beforeOrders, beforeSpecificOrder)
	applied, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-run"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied.Blocked || applied.ResultMode != "destructive_apply" || !applied.DestructiveApplySupported || !applied.RuntimeRowsDeleted {
		t.Fatalf("expected destructive apply success, got %+v", applied)
	}
	t.Logf("DEBUG applied eligible counts: %+v", applied.EligibleCounts)
	afterSpecificOrder := countRowsWhere(t, f, "orders", "id = ?", order.ID)
	t.Logf("DEBUG after apply: specific_order_rows=%d", afterSpecificOrder)
	if afterSpecificOrder != 0 {
		t.Fatalf("expected runtime order %s to be deleted, got %d rows", order.ID, afterSpecificOrder)
	}
	if got := countRowsWhere(t, f, "checks", "id = ?", check.ID); got != 0 {
		t.Fatalf("expected runtime check %s to be deleted, got %d rows", check.ID, got)
	}
	if got := countRowsWhere(t, f, "financial_operations", "id = ?", operation.ID); got != 0 {
		t.Fatalf("expected financial operation %s to be deleted, got %d rows", operation.ID, got)
	}
	if got := countRowsWhere(t, f, "financial_operation_items", "operation_id = ?", operation.ID); got != 0 {
		t.Fatalf("expected financial operation items for %s to be deleted, got %d rows", operation.ID, got)
	}
	if got := countRows(t, f, "orders"); got >= beforeOrders {
		t.Fatalf("expected order table to shrink after apply, before=%d after=%d", beforeOrders, got)
	}
	if got := countRows(t, f, "financial_operations"); got >= beforeFinancialOperations {
		t.Fatalf("expected financial_operations to shrink after apply, before=%d after=%d", beforeFinancialOperations, got)
	}

	closed, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{
		CommandMeta: f.edgeMetaCommand("cmd-storage-apply-closed-orders-after"),
		CheckID:     check.ID,
		Limit:       10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(closed) != 0 {
		t.Fatalf("expected deleted check to disappear from /orders/closed read model, got %+v", closed)
	}
	readPlan, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-apply-read-plan-after"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		CheckID:      check.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if readPlan.Blocked || readPlan.Returned != 1 || readPlan.ArchivedClosedOrders[0].CheckID != check.ID {
		t.Fatalf("expected archive read-plan to preview deleted check, got %+v", readPlan)
	}
	lookup, err := f.service.LookupStorageArchivePreview(f.ctx, app.ArchiveLookupCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-apply-lookup-after"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		CheckID:      check.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !lookup.Lookup.Found || lookup.Check == nil || lookup.Check.ID != check.ID {
		t.Fatalf("expected archive lookup to find deleted check, got %+v", lookup)
	}
}

func TestStorageArchiveApplyReadinessBlocksRepeatApplyAfterRuntimeRowsDeleted(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	operation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-storage-repeat-apply-cancel"),
		CheckID:              check.ID,
		Reason:               "guest returned immediately",
		InventoryDisposition: domain.InventoryNoStockEffect,
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   check.Total,
			Currency: check.CurrencyCode,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	markEdgeOutboxSent(t, f)

	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-repeat-apply-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "repeat destructive apply fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstApply, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-repeat-apply-first"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if firstApply.Blocked || !firstApply.RuntimeRowsDeleted || firstApply.ResultMode != "destructive_apply" {
		t.Fatalf("expected first destructive apply to succeed, got %+v", firstApply)
	}
	if got := countRowsWhere(t, f, "orders", "id = ?", order.ID); got != 0 {
		t.Fatalf("expected runtime order %s to be deleted by first apply, got %d rows", order.ID, got)
	}
	if got := countRowsWhere(t, f, "financial_operations", "id = ?", operation.ID); got != 0 {
		t.Fatalf("expected runtime financial operation %s to be deleted by first apply, got %d rows", operation.ID, got)
	}

	beforeRepeat := protectedStorageCounts(t, f)
	readiness, err := f.service.BuildStorageArchiveApplyReadiness(f.ctx, app.ArchiveApplyReadinessCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-repeat-apply-readiness"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if readiness.ReadyForDestructiveApply || readiness.RuntimeScopeVerified || !readiness.ArchiveVerified || !readiness.ManifestVerified {
		t.Fatalf("expected repeat readiness to block on runtime/archive counts mismatch only, got %+v", readiness)
	}
	if !containsString(readiness.BlockReasons, "archive_counts_mismatch") ||
		containsString(readiness.BlockReasons, "pending_edge_to_cloud_outbox") ||
		containsString(readiness.BlockReasons, "open_operational_boundary") {
		t.Fatalf("expected repeat readiness count mismatch without runtime blockers, got %+v", readiness.BlockReasons)
	}
	if readiness.EligibleCounts.ClosedOrders != 0 || readiness.ArchiveCounts.ClosedOrders != 1 ||
		readiness.ArchiveCounts.FinancialOperations != 1 || readiness.ArchiveCounts.FinancialOperationItems != 1 {
		t.Fatalf("expected empty runtime scope against archived compensation ledger, got eligible=%+v archive=%+v", readiness.EligibleCounts, readiness.ArchiveCounts)
	}
	assertProtectedStorageCounts(t, f, beforeRepeat, "repeat apply-readiness after destructive apply")

	repeatApply, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-repeat-apply-blocked"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repeatApply.Blocked || repeatApply.RuntimeRowsDeleted || repeatApply.ResultMode != "apply_blocked" {
		t.Fatalf("expected repeat destructive apply to be blocked without deleting rows, got %+v", repeatApply)
	}
	if !containsString(repeatApply.BlockReasons, "archive_counts_mismatch") {
		t.Fatalf("expected repeat apply archive_counts_mismatch, got %+v", repeatApply.BlockReasons)
	}
	assertProtectedStorageCounts(t, f, beforeRepeat, "repeat destructive apply")
}

func TestStorageArchiveDestructiveApplyRechecksPendingOutboxInsideTransaction(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)

	_, err := f.repo.ApplyStorageArchiveDestructive(f.ctx, "2026-05-04")
	if err == nil {
		t.Fatal("expected destructive apply to reject scoped pending outbox")
	}
	if !strings.Contains(err.Error(), "pending_edge_to_cloud_outbox") {
		t.Fatalf("expected pending outbox blocker, got %v", err)
	}
	if got := countRowsWhere(t, f, "orders", "id = ?", order.ID); got != 1 {
		t.Fatalf("expected order %s to remain after blocked apply, got %d rows", order.ID, got)
	}
	if got := countRowsWhere(t, f, "checks", "id = ?", check.ID); got != 1 {
		t.Fatalf("expected check %s to remain after blocked apply, got %d rows", check.ID, got)
	}
}

func TestStorageArchiveApplyReadinessReportsSHAAndCountsMismatch(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-readiness-mismatch-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "readiness mismatch fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := appendArchiveLine(exported.ArchivePath, `{"table":"unknown","row":{}}`); err != nil {
		t.Fatal(err)
	}
	manifest := readArchiveManifest(t, exported.ManifestPath)
	manifest.Counts.ClosedOrders++
	writeArchiveManifest(t, exported.ManifestPath, manifest)

	result, err := f.service.BuildStorageArchiveApplyReadiness(f.ctx, app.ArchiveApplyReadinessCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-readiness-mismatch"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ArchiveVerified || result.RuntimeScopeVerified || result.ReadyForDestructiveApply {
		t.Fatalf("expected mismatched archive to fail readiness, got %+v", result)
	}
	if !containsString(result.BlockReasons, "archive_sha_mismatch") ||
		!containsString(result.BlockReasons, "archive_manifest_counts_mismatch") ||
		!containsString(result.BlockReasons, "archive_counts_mismatch") {
		t.Fatalf("expected sha/count mismatch reasons, got %+v", result.BlockReasons)
	}
}

func TestStorageArchiveApplyPlanBlocksManifestVersionMismatch(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-version-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "manifest version fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readArchiveManifest(t, exported.ManifestPath)
	manifest.Version = "unsupported_archive_version"
	writeArchiveManifest(t, exported.ManifestPath, manifest)

	result, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-version"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(result.BlockReasons, "archive_manifest_version_mismatch") || result.Verification.ManifestVersionMatched {
		t.Fatalf("expected archive_manifest_version_mismatch, got result=%+v verification=%+v", result, result.Verification)
	}
}

func TestStorageArchiveApplyPlanBlocksRuntimeMismatchOutboxAndOpenBoundary(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	activeOrder, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{
		CommandMeta: f.edgeMetaCommand("cmd-storage-apply-active-order"),
		TableID:     f.table.ID,
		TableName:   f.table.Name,
		GuestCount:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	execTestSQL(t, f, `UPDATE shifts SET business_date_local = '2026-05-03' WHERE id = ?`, activeOrder.ShiftID)
	execTestSQL(t, f, `UPDATE cash_sessions SET business_date_local = '2026-05-03' WHERE shift_id = ? AND status = 'open'`, activeOrder.ShiftID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-empty-export"),
		CutoffBusinessDateLocal: "2026-05-03",
		Reason:                  "runtime mismatch fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := protectedStorageCounts(t, f)
	result, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-runtime-mismatch"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if activeOrder.ID == "" || result.ActiveOrders < 1 || result.OpenShifts < 1 || result.OpenCashSessions < 1 {
		t.Fatalf("expected open operational boundary in apply plan, got %+v", result)
	}
	if !containsString(result.BlockReasons, "archive_counts_mismatch") ||
		!containsString(result.BlockReasons, "pending_edge_to_cloud_outbox") ||
		!containsString(result.BlockReasons, "open_operational_boundary") {
		t.Fatalf("expected runtime mismatch/outbox/open boundary block reasons, got %+v", result.BlockReasons)
	}
	assertProtectedStorageCounts(t, f, before, "apply-plan runtime mismatch")
}

func TestStorageArchiveApplyPlanDetectsMissingSnapshotPayload(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-snapshot-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "snapshot fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(exported.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := strings.Replace(string(raw), `"snapshot":`, `"snapshot_removed":`, 1)
	if err := os.WriteFile(exported.ArchivePath, []byte(rewritten), 0o640); err != nil {
		t.Fatal(err)
	}
	sha := sha256.Sum256([]byte(rewritten))
	manifest := readArchiveManifest(t, exported.ManifestPath)
	manifest.SHA256 = hex.EncodeToString(sha[:])
	writeArchiveManifest(t, exported.ManifestPath, manifest)

	result, err := f.service.BuildStorageArchiveApplyPlan(f.ctx, app.ArchiveApplyPlanCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-apply-snapshot"),
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(result.BlockReasons, "archive_snapshot_payload_missing") || result.Verification.SnapshotPayloadPresent {
		t.Fatalf("expected archive_snapshot_payload_missing, got result=%+v verification=%+v", result, result.Verification)
	}
}

func TestStorageArchiveVerifyRequiresPermissionAndValidatesExport(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-verify-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "verify fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.VerifyStorageArchive(f.ctx, app.ArchiveVerifyCommand{
		CommandMeta:  f.edgeMetaCommand("cmd-storage-verify-cashier-denied"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier archive verify, got %v", err)
	}
	before := protectedStorageCounts(t, f)
	result, err := f.service.VerifyStorageArchive(f.ctx, app.ArchiveVerifyCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-verify-manager"),
		ArchiveID:    exported.ArchiveID,
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || len(result.Errors) != 0 || result.ArchiveID != exported.ArchiveID || result.Counts.ClosedOrders != 1 {
		t.Fatalf("unexpected archive verify result: %+v", result)
	}
	if !result.Verification.ManifestVersionMatched || !result.Verification.SHA256Matched ||
		!result.Verification.CountsMatchedManifest || !result.Verification.IdentityFieldsPresent ||
		!result.Verification.BusinessDateConsistent || !result.Verification.RuntimeRowsNotDeleted ||
		!result.Verification.PayloadPolicyPreserved {
		t.Fatalf("unexpected verification flags: %+v", result.Verification)
	}
	assertProtectedStorageCounts(t, f, before, "archive verify")
}

func TestStorageArchiveVerifyReportsMissingAndMalformedArtifacts(t *testing.T) {
	f := newFixture(t)
	missing, err := f.service.VerifyStorageArchive(f.ctx, app.ArchiveVerifyCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-verify-missing"),
		ArchivePath:  filepath.Join(f.archiveDir, "missing", "archive.jsonl"),
		ManifestPath: filepath.Join(f.archiveDir, "missing", "manifest.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Valid || !containsString(missing.Errors, "archive_manifest_missing") {
		t.Fatalf("expected missing manifest verification error, got %+v", missing)
	}

	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-verify-malformed-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "malformed fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := appendArchiveLine(exported.ArchivePath, `{"table":"checks","row":`); err != nil {
		t.Fatal(err)
	}
	malformed, err := f.service.VerifyStorageArchive(f.ctx, app.ArchiveVerifyCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-verify-malformed"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if malformed.Valid || !containsString(malformed.Errors, "archive_jsonl_malformed") {
		t.Fatalf("expected malformed JSONL verification error, got %+v", malformed)
	}
}

func TestStorageArchiveVerifyDetectsIdentityDateRuntimeAndPayloadPolicyViolations(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-verify-policy-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "policy fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(exported.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := strings.Replace(string(raw), `"order_id":`, `"order_id_removed":`, 1)
	rewritten = strings.Replace(rewritten, `"payload_policy":"summary_without_payload"`, `"payload_json":{},"payload_policy":"full_payload"`, 1)
	if err := os.WriteFile(exported.ArchivePath, []byte(rewritten), 0o640); err != nil {
		t.Fatal(err)
	}
	sha := sha256.Sum256([]byte(rewritten))
	manifest := readArchiveManifest(t, exported.ManifestPath)
	manifest.SHA256 = hex.EncodeToString(sha[:])
	manifest.RuntimeRowsDeleted = true
	manifest.BusinessDateRange.Newest = "2026-05-01"
	writeArchiveManifest(t, exported.ManifestPath, manifest)

	result, err := f.service.VerifyStorageArchive(f.ctx, app.ArchiveVerifyCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-verify-policy"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{
		"archive_runtime_rows_deleted_true",
		"archive_identity_fields_missing",
		"archive_business_date_range_mismatch",
		"archive_sensitive_payload_policy_violation",
	} {
		if !containsString(result.Errors, reason) {
			t.Fatalf("expected %s in verification errors, got %+v", reason, result.Errors)
		}
	}
}

func TestStorageArchiveReadPlanHappyPathAfterExport(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-read-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "read plan fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := protectedStorageCounts(t, f)

	result, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-read-plan"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked || result.ResultMode != "read_plan_only" || result.ArchiveID != exported.ArchiveID ||
		result.CutoffBusinessDateLocal != "2026-05-04" || result.Counts.ClosedOrders != 1 {
		t.Fatalf("unexpected archive read-plan result: %+v", result)
	}
	if result.ArchiveSHA256 != exported.SHA256 || result.ComputedSHA256 != exported.SHA256 ||
		!result.Verification.ManifestVersionMatched || !result.Verification.SHA256Matched ||
		!result.Verification.CountsMatchedManifest || !result.Verification.SnapshotPayloadPresent {
		t.Fatalf("unexpected archive read-plan verification: %+v", result.Verification)
	}
	if len(result.Tables) == 0 || result.Tables[0].Name != "orders" {
		t.Fatalf("expected archive table summary, got %+v", result.Tables)
	}
	if result.Limit != 50 || result.Offset != 0 || result.Returned != 1 || result.RuntimeRowsDeleted || result.RuntimeRestored {
		t.Fatalf("unexpected bounded archive read-plan metadata: %+v", result)
	}
	if len(result.ArchivedClosedOrders) != 1 || result.ArchivedClosedOrders[0].DocumentState != "archived_preview" ||
		result.ArchivedClosedOrders[0].RuntimeRestored || !result.ArchivedClosedOrders[0].CheckSnapshotPresent {
		t.Fatalf("unexpected archived closed order preview: %+v", result.ArchivedClosedOrders)
	}
	assertProtectedStorageCounts(t, f, before, "read-plan happy path")
}

func TestStorageArchiveReadPlanBoundsAndFiltersArchivedClosedOrders(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "archive-order-1", "archive-check-1", "2026-05-02", fixedClock{}.Now().Add(-48*time.Hour))
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "archive-order-2", "archive-check-2", "2026-05-03", fixedClock{}.Now().Add(-24*time.Hour))
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "archive-order-3", "archive-check-3", "2026-05-04", fixedClock{}.Now())
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-read-bounds-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "read bounds fixture",
	})
	if err != nil {
		t.Fatal(err)
	}

	page, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-read-bounds"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		Limit:        2,
		Offset:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Limit != 2 || page.Offset != 1 || page.Returned != 1 ||
		page.ArchivedClosedOrders[0].OrderID != "archive-order-2" {
		t.Fatalf("unexpected bounded archive read page: %+v", page)
	}
	filtered, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:       f.managerEdgeMetaCommand(t, "cmd-storage-read-filter-date"),
		ArchivePath:       exported.ArchivePath,
		ManifestPath:      exported.ManifestPath,
		BusinessDateLocal: "2026-05-03",
		Limit:             500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Limit != 100 || filtered.Returned != 1 || filtered.ArchivedClosedOrders[0].CheckID != "archive-check-2" {
		t.Fatalf("unexpected business date filtered archive read-plan: %+v", filtered)
	}
	byCheck, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-read-filter-check"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		CheckID:      "archive-check-3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if byCheck.Returned != 0 {
		t.Fatalf("unexpected check filtered archive read-plan: %+v", byCheck)
	}
	raw, err := json.Marshal(filtered)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "payload_json") {
		t.Fatalf("read-plan must not expose sync/event payload_json: %s", string(raw))
	}
}

func TestStorageArchiveReadPlanRejectsPathOutsideArchiveDir(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-read-outside-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "outside path fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	outsideArchive := filepath.Join(t.TempDir(), "archive.jsonl")
	result, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-read-outside"),
		ArchivePath:  outsideArchive,
		ManifestPath: exported.ManifestPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || !containsString(result.BlockReasons, "archive_path_outside_archive_dir") {
		t.Fatalf("expected outside archive dir block reason, got %+v", result)
	}
}

func TestStorageArchiveReadPlanDetectsSHAMismatch(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-read-sha-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "sha mismatch fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := appendArchiveLine(exported.ArchivePath, `{"table":"unknown","row":{}}`); err != nil {
		t.Fatal(err)
	}

	result, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-read-sha"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || !containsString(result.BlockReasons, "archive_sha_mismatch") || result.Verification.SHA256Matched {
		t.Fatalf("expected archive_sha_mismatch, got %+v", result)
	}
}

func TestStorageArchiveReadPlanDetectsManifestCountsMismatch(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-read-counts-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "counts mismatch fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readArchiveManifest(t, exported.ManifestPath)
	manifest.Counts.ClosedOrders++
	writeArchiveManifest(t, exported.ManifestPath, manifest)

	result, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-read-counts"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || !containsString(result.BlockReasons, "archive_manifest_counts_mismatch") ||
		result.Verification.CountsMatchedManifest {
		t.Fatalf("expected archive_manifest_counts_mismatch, got %+v", result)
	}
}

func TestStorageArchiveLookupByCheckIDReturnsImmutableSnapshots(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-lookup-check-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "lookup by check fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := protectedStorageCounts(t, f)

	result, err := f.service.LookupStorageArchivePreview(f.ctx, app.ArchiveLookupCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-lookup-check"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		CheckID:      check.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked || result.ResultMode != "archive_lookup_preview" || !result.Lookup.Found ||
		result.Lookup.CheckID != check.ID || result.Lookup.OrderID != order.ID {
		t.Fatalf("unexpected lookup result: %+v", result)
	}
	if result.Check == nil || result.Check.ID != check.ID || len(result.Check.Snapshot) == 0 {
		t.Fatalf("expected check snapshot preview, got %+v", result.Check)
	}
	if result.Precheck == nil || len(result.Precheck.Snapshot) == 0 {
		t.Fatalf("expected precheck snapshot preview, got %+v", result.Precheck)
	}
	if result.RelatedCounts.OrderLines != 1 || result.RelatedCounts.Payments != 1 {
		t.Fatalf("unexpected related counts: %+v", result.RelatedCounts)
	}
	assertProtectedStorageCounts(t, f, before, "lookup by check_id")
}

func TestStorageArchiveLookupByOrderIDReturnsMatchingArchivedPreview(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-lookup-order-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "lookup by order fixture",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := f.service.LookupStorageArchivePreview(f.ctx, app.ArchiveLookupCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-lookup-order"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		OrderID:      order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Blocked || !result.Lookup.Found || result.Lookup.OrderID != order.ID || result.Lookup.CheckID != check.ID {
		t.Fatalf("unexpected lookup by order result: %+v", result)
	}
	if result.Check == nil || result.Check.ID != check.ID || result.Precheck == nil {
		t.Fatalf("expected matching check/precheck preview, got check=%+v precheck=%+v", result.Check, result.Precheck)
	}
}

func TestStorageArchiveLookupRejectsEmptyAndAmbiguousKeys(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.LookupStorageArchivePreview(f.ctx, app.ArchiveLookupCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "cmd-storage-lookup-empty-key"),
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid empty lookup key, got %v", err)
	}
	_, err = f.service.LookupStorageArchivePreview(f.ctx, app.ArchiveLookupCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "cmd-storage-lookup-both-keys"),
		CheckID:     "check-1",
		OrderID:     "order-1",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid ambiguous lookup key, got %v", err)
	}
}

func TestStorageArchiveLookupReturnsStructuredNotFound(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-lookup-not-found-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "lookup not found fixture",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := f.service.LookupStorageArchivePreview(f.ctx, app.ArchiveLookupCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-lookup-not-found"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		CheckID:      "missing-check",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Blocked || result.Lookup.Found || !containsString(result.BlockReasons, "archive_record_not_found") {
		t.Fatalf("expected structured not-found result, got %+v", result)
	}
}

func TestStorageArchiveReadPlanAndLookupKeepRuntimeCountsUnchanged(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	makeCheckOlderThanArchiveCutoff(t, f, check.ID)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.managerEdgeMetaCommand(t, "cmd-storage-zero-delete-export"),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "zero deletion fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := protectedStorageCounts(t, f)
	if _, err := f.service.BuildStorageArchiveReadPlan(f.ctx, app.ArchiveReadPlanCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-zero-delete-read"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.LookupStorageArchivePreview(f.ctx, app.ArchiveLookupCommand{
		CommandMeta:  f.managerEdgeMetaCommand(t, "cmd-storage-zero-delete-lookup"),
		ArchivePath:  exported.ArchivePath,
		ManifestPath: exported.ManifestPath,
		CheckID:      check.ID,
	}); err != nil {
		t.Fatal(err)
	}
	if order.ID == "" {
		t.Fatal("expected archived order fixture")
	}
	assertProtectedStorageCounts(t, f, before, "read-plan and lookup")
}

func decodeArchiveJSONLByTable(t *testing.T, raw []byte) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row struct {
			Table string `json:"table"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("decode archive line: %v; line=%s", err, line)
		}
		out[row.Table]++
	}
	return out
}

func appendArchiveLine(path, line string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(line + "\n"); err != nil {
		return err
	}
	return nil
}

func readArchiveManifest(t *testing.T, path string) domain.StorageArchiveManifest {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest domain.StorageArchiveManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func writeArchiveManifest(t *testing.T, path string, manifest domain.StorageArchiveManifest) {
	t.Helper()
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o640); err != nil {
		t.Fatal(err)
	}
}

func protectedStorageCounts(t *testing.T, f *fixture) map[string]int {
	t.Helper()
	tables := []string{
		"orders",
		"order_lines",
		"prechecks",
		"payments",
		"checks",
		"financial_operations",
		"financial_operation_items",
		"local_event_log",
		"pos_sync_outbox",
	}
	out := map[string]int{}
	for _, table := range tables {
		out[table] = countRows(t, f, table)
	}
	return out
}

func assertProtectedStorageCounts(t *testing.T, f *fixture, before map[string]int, operation string) {
	t.Helper()
	for table, want := range before {
		if got := countRows(t, f, table); got != want {
			t.Fatalf("%s mutated %s, before=%d after=%d", operation, table, want, got)
		}
	}
}

func TestFloorAndMenuReadAsOperatorRequiresPermissions(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.ListHallsAsOperator(f.ctx, f.restaurant.ID, f.edgeMetaCommand("cmd-halls-cashier-allow")); err != nil {
		t.Fatalf("expected cashier halls access, got %v", err)
	}
	if _, err := f.service.ListTablesAsOperator(f.ctx, f.restaurant.ID, f.hall.ID, f.edgeMetaCommand("cmd-tables-cashier-allow")); err != nil {
		t.Fatalf("expected cashier tables access, got %v", err)
	}
	if _, err := f.service.ListMenuItemsAsOperator(f.ctx, f.edgeMetaCommand("cmd-menu-cashier-allow")); err != nil {
		t.Fatalf("expected cashier menu access, got %v", err)
	}

	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     seedMeta(bootstrapDeviceID),
		Name:            "read-restricted",
		PermissionsJSON: appshared.PermissionsJSON(appshared.PermissionOrderCreate),
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "Restricted",
		PINHash:      testPINHash(t, "1357", "restricted-salt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-restricted",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "1357",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := edgeMeta(f.device.ID)
	meta.CommandID = "cmd-floor-menu-restricted-denied"
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = employee.ID
	meta.SessionID = login.Session.ID

	if _, err := f.service.ListHallsAsOperator(f.ctx, f.restaurant.ID, meta); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected halls forbidden for restricted role, got %v", err)
	}
	if _, err := f.service.ListTablesAsOperator(f.ctx, f.restaurant.ID, f.hall.ID, meta); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected tables forbidden for restricted role, got %v", err)
	}
	if _, err := f.service.ListMenuItemsAsOperator(f.ctx, meta); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected menu forbidden for restricted role, got %v", err)
	}
}

func TestClaimPendingOutboxSkipsFutureNextRetryAt(t *testing.T) {
	f := newFixture(t)
	future := appshared.DBTime(fixedClock{}.Now().Add(time.Hour))
	now := appshared.DBTime(fixedClock{}.Now())
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET next_retry_at = ?, updated_at = ? WHERE status = 'pending'`, future, now); err != nil {
		t.Fatal(err)
	}

	claimed, err := f.service.ClaimPendingOutbox(f.ctx, app.ClaimPendingOutboxCommand{Limit: 10, LockedBy: "sync-test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 0 {
		t.Fatalf("expected no claimed messages while next_retry_at is in future, got %d", len(claimed))
	}
}

func TestClaimPendingOutboxUsesSequenceOrderAndLocksRows(t *testing.T) {
	f := newFixture(t)
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET next_retry_at = NULL, updated_at = ? WHERE status = 'pending'`, appshared.DBTime(fixedClock{}.Now())); err != nil {
		t.Fatal(err)
	}

	claimed, err := f.service.ClaimPendingOutbox(f.ctx, app.ClaimPendingOutboxCommand{Limit: 3, LockedBy: "sync-test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 3 {
		t.Fatalf("expected 3 claimed messages, got %d", len(claimed))
	}
	for i, msg := range claimed {
		if msg.Status != domain.OutboxProcessing || msg.LockedBy == nil || *msg.LockedBy != "sync-test" || msg.LockedAt == nil {
			t.Fatalf("expected claimed message to be processing and locked, got %+v", msg)
		}
		if i > 0 && claimed[i-1].SequenceNo >= msg.SequenceNo {
			t.Fatalf("expected sequence order, got %d before %d", claimed[i-1].SequenceNo, msg.SequenceNo)
		}
	}
}

func TestReclaimStaleProcessingOutboxReturnsOldLocksToPending(t *testing.T) {
	f := newFixture(t)
	ids := outboxIDs(t, f, 2)
	oldLock := appshared.DBTime(fixedClock{}.Now().Add(-2 * time.Hour))
	freshLock := appshared.DBTime(fixedClock{}.Now().Add(-time.Minute))
	now := appshared.DBTime(fixedClock{}.Now())
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'processing', locked_at = ?, locked_by = 'old-worker', updated_at = ? WHERE id = ?`, oldLock, now, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'processing', locked_at = ?, locked_by = 'fresh-worker', updated_at = ? WHERE id = ?`, freshLock, now, ids[1]); err != nil {
		t.Fatal(err)
	}

	reclaimed, err := f.service.ReclaimStaleProcessingOutbox(f.ctx, app.ReclaimStaleOutboxCommand{StaleBefore: fixedClock{}.Now().Add(-time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed != 1 {
		t.Fatalf("expected one stale processing lock reclaimed, got %d", reclaimed)
	}
	status, _ := outboxStatusAttempts(t, f, ids[0])
	if status != domain.OutboxPending {
		t.Fatalf("expected stale row to return to pending, got %s", status)
	}
	status, _ = outboxStatusAttempts(t, f, ids[1])
	if status != domain.OutboxProcessing {
		t.Fatalf("expected fresh lock to stay processing, got %s", status)
	}
}

func TestMarkOutboxFailedSuspendsAfterAttemptsExceedThreshold(t *testing.T) {
	f := newFixture(t)
	id := outboxIDs(t, f, 1)[0]
	for i := 0; i < appshared.DefaultOutboxMaxAttempts+1; i++ {
		if err := f.service.MarkOutboxFailed(f.ctx, id, "cloud unavailable"); err != nil {
			t.Fatal(err)
		}
	}
	status, attempts := outboxStatusAttempts(t, f, id)
	if status != domain.OutboxSuspended {
		t.Fatalf("expected suspended after threshold exceeded, got %s", status)
	}
	if attempts != appshared.DefaultOutboxMaxAttempts+1 {
		t.Fatalf("expected attempts=%d, got %d", appshared.DefaultOutboxMaxAttempts+1, attempts)
	}
}

func TestMarkOutboxRetryableFailureKeepsMessagePendingWithBackoff(t *testing.T) {
	f := newFixture(t)
	id := outboxIDs(t, f, 1)[0]
	if _, err := f.service.ClaimPendingOutbox(f.ctx, app.ClaimPendingOutboxCommand{Limit: 1, LockedBy: "sync-test"}); err != nil {
		t.Fatal(err)
	}
	if err := f.service.MarkOutboxRetryableFailure(f.ctx, id, "cloud unavailable"); err != nil {
		t.Fatal(err)
	}
	status, attempts := outboxStatusAttempts(t, f, id)
	if status != domain.OutboxPending || attempts != 1 {
		t.Fatalf("expected pending retry with attempts=1, got status=%s attempts=%d", status, attempts)
	}
	var nextRetryAt, lockedAt, lockedBy string
	if err := f.db.QueryRowContext(f.ctx, `SELECT COALESCE(next_retry_at,''), COALESCE(locked_at,''), COALESCE(locked_by,'') FROM pos_sync_outbox WHERE id = ?`, id).Scan(&nextRetryAt, &lockedAt, &lockedBy); err != nil {
		t.Fatal(err)
	}
	if nextRetryAt == "" {
		t.Fatal("expected next_retry_at to be set")
	}
	if lockedAt != "" || lockedBy != "" {
		t.Fatalf("expected retryable failure to release lock, got locked_at=%q locked_by=%q", lockedAt, lockedBy)
	}
}

func TestGetSyncStatusAggregatesOutboxRows(t *testing.T) {
	f := newFixture(t)
	ids := outboxIDs(t, f, 4)
	now := appshared.DBTime(fixedClock{}.Now())
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'processing', locked_at = ?, locked_by = 'worker', updated_at = ? WHERE id = ?`, now, now, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = 1, last_error = 'temporary', updated_at = ? WHERE id = ?`, now, ids[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'suspended', attempts = 4, last_error = 'threshold', updated_at = ? WHERE id = ?`, now, ids[2]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'sent', sent_at = ?, updated_at = ? WHERE id = ?`, now, now, ids[3]); err != nil {
		t.Fatal(err)
	}

	status, err := f.service.GetSyncStatus(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Total != countRows(t, f, "pos_sync_outbox") || status.Processing != 1 || status.Failed != 1 || status.Suspended != 1 || status.Sent != 1 {
		t.Fatalf("unexpected sync status: %+v", status)
	}
	if status.Pending == 0 || status.OldestPendingSequenceNo == nil {
		t.Fatalf("expected pending rows with oldest sequence, got %+v", status)
	}
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
	_, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1"})
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
		CommandMeta:         f.managerEdgeMetaCommand(t, "cmd-cash-drawer-no-session"),
		CreatedByEmployeeID: f.manager.ID,
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

func TestCashDrawerEventRequiresPermission(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	session, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMetaCommand("cmd-open-cash-before-denied-drawer"),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  100,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.RecordCashDrawerEvent(f.ctx, app.RecordCashDrawerEventCommand{
		CommandMeta:         f.edgeMetaCommand("cmd-cash-drawer-denied"),
		CashSessionID:       session.ID,
		CreatedByEmployeeID: f.employee.ID,
		EventType:           domain.CashDrawerNoSale,
		Amount:              0,
		Reason:              "cashier check",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for cashier cash drawer event, got %v", err)
	}
}

func TestDuplicateCashSessionCommandIDDoesNotCreateDuplicateRows(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	sessionsBefore := countRows(t, f, "cash_sessions")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	cmd := app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMetaCommand("cmd-open-cash-session-1"),
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
		CommandMeta: f.edgeMetaCommand("cmd-create-order-1"),
		TableID:     f.table.ID,
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
		CommandMeta: f.edgeMetaCommand("cmd-local-event-fails"),
		TableID:     f.table.ID,
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
		CommandMeta: f.edgeMetaCommand("cmd-outbox-fails"),
		TableID:     f.table.ID,
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
		CommandMeta:        f.edgeMetaCommand("cmd-cash-outbox-fails"),
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

func TestCreateRoleRejectsUnknownPermissionID(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     seedMeta(f.device.ID),
		Name:            "bad-role",
		PermissionsJSON: `{"pos.order.create":true,"pos.permission.unknown":true}`,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid unknown permission id, got %v", err)
	}
}

func TestCurrencyValidationRejectsUnsupportedISOCode(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{
		CommandMeta: seedMeta(f.device.ID),
		Name:        "Unsupported Currency Restaurant",
		Timezone:    "Europe/Moscow",
		Currency:    "AAA",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid unsupported restaurant currency, got %v", err)
	}
	if _, err := f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{
		CommandMeta:   seedMeta(f.device.ID),
		CatalogItemID: f.menuItem.CatalogItemID,
		Name:          "Unknown Currency Item",
		Price:         300,
		Currency:      "AAA",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid unsupported menu currency, got %v", err)
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

func TestPinLoginCreatesLocalSessionAndActorMetadataWithoutPINLeak(t *testing.T) {
	f := newFixture(t)
	sessionsBefore := countRows(t, f, "auth_sessions")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	result, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-pin-login-1", NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, Origin: app.OriginEdgeDevice},
		PIN:         "1111",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.EmployeeID != f.employee.ID || result.Actor.EmployeeID != f.employee.ID || result.Session.NodeDeviceID != f.device.ID || result.Session.ClientDeviceID != f.clientID {
		t.Fatalf("unexpected login result: %+v", result)
	}
	if sessions := countRows(t, f, "auth_sessions"); sessions != sessionsBefore+1 {
		t.Fatalf("expected one auth session, before=%d after=%d", sessionsBefore, sessions)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one auth outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one auth local event, before=%d after=%d", eventsBefore, events)
	}
	var actorID, sessionID, payload string
	if err := f.db.QueryRowContext(f.ctx, `SELECT actor_employee_id, session_id, payload_json FROM local_event_log WHERE command_id = ?`, "cmd-pin-login-1").Scan(&actorID, &sessionID, &payload); err != nil {
		t.Fatal(err)
	}
	if actorID != f.employee.ID || sessionID != result.Session.ID {
		t.Fatalf("expected actor/session metadata, got actor=%s session=%s", actorID, sessionID)
	}
	if strings.Contains(payload, "1111") {
		t.Fatal("expected PIN not to be written to local event payload")
	}
	current, err := f.service.GetSession(f.ctx, result.Session.ID, f.device.ID, f.clientID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Actor.EmployeeID != f.employee.ID {
		t.Fatalf("unexpected current session actor: %+v", current.Actor)
	}
}

func TestLogoutRevokesBackendSession(t *testing.T) {
	f := newFixture(t)
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-login-before-logout", NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, Origin: app.OriginEdgeDevice},
		PIN:         "1111",
	})
	if err != nil {
		t.Fatal(err)
	}
	logout, err := f.service.Logout(f.ctx, app.LogoutCommand{
		CommandMeta: app.CommandMeta{CommandID: "cmd-logout", NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, Origin: app.OriginEdgeDevice},
		SessionID:   login.Session.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if logout.Status != domain.AuthSessionRevoked || logout.RevokedAt == nil {
		t.Fatalf("expected revoked session, got %+v", logout)
	}
	if _, err := f.service.GetSession(f.ctx, login.Session.ID, f.device.ID, f.clientID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected revoked session to be rejected, got %v", err)
	}
}

func TestPinLoginRejectsInvalidPINWithoutSessionWrite(t *testing.T) {
	f := newFixture(t)
	before := countRows(t, f, "auth_sessions")

	_, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{CommandMeta: f.edgeMeta(), PIN: "9999"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if sessions := countRows(t, f, "auth_sessions"); sessions != before {
		t.Fatalf("expected no auth session write, before=%d after=%d", before, sessions)
	}
}

func TestCreateEmployeeDoesNotWritePINHashToOutboxOrLocalEvent(t *testing.T) {
	f := newFixture(t)
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{CommandMeta: seedMeta(f.device.ID), Name: "auditor", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	hash := testPINHash(t, "1357", "auditor-salt")
	_, err = f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  app.CommandMeta{CommandID: "cmd-create-employee-no-pin-leak", DeviceID: f.device.ID, Origin: app.OriginSystemSeed},
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "Oleg",
		PINHash:      hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	var eventPayload, outboxPayload string
	if err := f.db.QueryRowContext(f.ctx, `SELECT payload_json FROM local_event_log WHERE command_id = ?`, "cmd-create-employee-no-pin-leak").Scan(&eventPayload); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT payload_json FROM pos_sync_outbox WHERE command_id = ?`, "cmd-create-employee-no-pin-leak").Scan(&outboxPayload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(eventPayload, "pin_hash") || strings.Contains(eventPayload, hash) || strings.Contains(outboxPayload, "pin_hash") || strings.Contains(outboxPayload, hash) {
		t.Fatal("expected employee PIN hash not to be written to local event or outbox payload")
	}
}

func TestCloudOrSeedCreateAndArchiveHallAndTableUseCloudToEdgeOutbox(t *testing.T) {
	f := newFixture(t)
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	hall, err := f.service.CreateHall(f.ctx, app.CreateHallCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, Name: "Terrace"})
	if err != nil {
		t.Fatal(err)
	}
	table, err := f.service.CreateTable(f.ctx, app.CreateTableCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, HallID: hall.ID, Name: "T1", Seats: 4})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.ArchiveTable(f.ctx, app.ArchiveTableCommand{CommandMeta: seedMeta(f.device.ID), ID: table.ID}); err != nil {
		t.Fatal(err)
	}
	if err := f.service.ArchiveHall(f.ctx, app.ArchiveHallCommand{CommandMeta: seedMeta(f.device.ID), ID: hall.ID}); err != nil {
		t.Fatal(err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+4 {
		t.Fatalf("expected four floor outbox rows, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+4 {
		t.Fatalf("expected four floor local events, before=%d after=%d", eventsBefore, events)
	}
	tables, err := f.service.ListTables(f.ctx, f.restaurant.ID, hall.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || tables[0].Active {
		t.Fatalf("expected archived table in read model, got %+v", tables)
	}
	var cloudToEdge int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM pos_sync_outbox WHERE aggregate_id IN (?, ?) AND command_type IN ('HallCreated','TableCreated','TableArchived','HallArchived') AND sync_direction = 'cloud_to_edge'`, hall.ID, table.ID).Scan(&cloudToEdge); err != nil {
		t.Fatal(err)
	}
	if cloudToEdge != 4 {
		t.Fatalf("expected floor master-data outbox rows to be cloud_to_edge, got %d", cloudToEdge)
	}
}

func TestCannotCreateTableInArchivedHall(t *testing.T) {
	f := newFixture(t)
	hall, err := f.service.CreateHall(f.ctx, app.CreateHallCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, Name: "Closed room"})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.ArchiveHall(f.ctx, app.ArchiveHallCommand{CommandMeta: seedMeta(f.device.ID), ID: hall.ID}); err != nil {
		t.Fatal(err)
	}

	_, err = f.service.CreateTable(f.ctx, app.CreateTableCommand{CommandMeta: seedMeta(f.device.ID), RestaurantID: f.restaurant.ID, HallID: hall.ID, Name: "C1", Seats: 2})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestEdgeRuntimeCannotMutateCloudOwnedMasterData(t *testing.T) {
	f := newFixture(t)
	rolesBefore := countRows(t, f, "roles")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	commandID := "cmd-role-with-device"

	_, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     app.CommandMeta{CommandID: commandID, DeviceID: f.device.ID},
		Name:            "supervisor",
		PermissionsJSON: `{}`,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if roles := countRows(t, f, "roles"); roles != rolesBefore {
		t.Fatalf("expected no role row, before=%d after=%d", rolesBefore, roles)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no outbox row, before=%d after=%d", outboxBefore, outbox)
	}
}

func TestCannotCloseShiftWithOpenOrders(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	if _, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1"}); err != nil {
		t.Fatal(err)
	}
	_, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{CommandMeta: f.edgeMeta(), ID: shift.ID, ClosedByEmployeeID: f.employee.ID, ClosingCashAmount: 0})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestOpenShiftRequiresPermission(t *testing.T) {
	f := newFixture(t)
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     seedMeta(f.device.ID),
		Name:            "no-shift-open",
		PermissionsJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "Denied Operator",
		PINHash:      testPINHash(t, "3579", "denied-operator-salt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-denied-open-shift",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "3579",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := edgeMeta(f.device.ID)
	meta.CommandID = "cmd-open-shift-denied"
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = employee.ID
	meta.SessionID = login.Session.ID
	_, err = f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        meta,
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: employee.ID,
		OpeningCashAmount:  0,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
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
	got, err := f.service.GetCurrentShift(f.ctx, f.edgeMeta())
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
	_, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotCloseOrderWithoutFullPayment(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CloseOrder(f.ctx, app.CloseOrderCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCloseOrderRequiresOrderClosePermission(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{
		CommandMeta: f.edgeMetaCommand("cmd-create-order-close-permission"),
		TableID:     f.table.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta: seedMeta(f.device.ID),
		Name:        "order-close-denied",
		PermissionsJSON: appshared.PermissionsJSON(
			appshared.PermissionOrderCreate,
			appshared.PermissionOrderView,
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "Denied Order Closer",
		PINHash:      testPINHash(t, "7391", "denied-order-close-salt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-denied-order-close",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "7391",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := edgeMeta(f.device.ID)
	meta.CommandID = "cmd-close-order-denied"
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = employee.ID
	meta.SessionID = login.Session.ID
	_, err = f.service.CloseOrder(f.ctx, app.CloseOrderCommand{CommandMeta: meta, OrderID: order.ID})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestIssuePrecheckCreatesDormantSnapshotAndLocksOrderWithoutLegacyCheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
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
		CommandMeta: f.edgeMetaCommand("cmd-issue-precheck-1"),
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
		t.Fatalf("expected final checks to remain unchanged, before=%d after=%d", checksBefore, checks)
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

func TestIssuePrecheckPersistsPricingBreakdownAndCheckUsesSnapshotTotals(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	now := fixedClock{}.Now().Format(time.RFC3339Nano)
	if _, err := f.db.ExecContext(f.ctx, `INSERT INTO tax_profiles(id,name,tax_exempt,active,created_at,updated_at) VALUES ('tax-profile-1','VAT',0,1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `INSERT INTO tax_rules(id,tax_profile_id,name,kind,mode,rate_basis_points,amount_minor,compound,priority,active,created_at,updated_at) VALUES ('tax-rule-1','tax-profile-1','VAT 10','percentage','exclusive',1000,0,0,1,1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE menu_items SET tax_profile_id = 'tax-profile-1' WHERE id = ?`, f.menuItem.ID); err != nil {
		t.Fatal(err)
	}
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddDiscount(f.ctx, app.AddDiscountCommand{
		CommandMeta:      f.edgeMetaCommand("cmd-pricing-discount"),
		OrderID:          order.ID,
		OrderLineID:      line.ID,
		Scope:            domain.DiscountScopeLine,
		ApplicationIndex: 10,
		AmountKind:       domain.AmountFixed,
		AmountMinor:      300,
		Reason:           "manual",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddSurcharge(f.ctx, app.AddSurchargeCommand{
		CommandMeta:      f.edgeMetaCommand("cmd-pricing-surcharge"),
		OrderID:          order.ID,
		Kind:             domain.SurchargeServiceCharge,
		ApplicationIndex: 20,
		AmountKind:       domain.AmountPercentage,
		ValueBasisPoints: 1000,
		Reason:           "service",
	}); err != nil {
		t.Fatal(err)
	}
	pricing, err := f.service.GetOrderPricingAsOperator(f.ctx, order.ID, f.edgeMeta())
	if err != nil {
		t.Fatal(err)
	}
	if pricing.SubtotalMinor != 2000 || pricing.DiscountTotalMinor != 300 || pricing.SurchargeTotalMinor != 170 || pricing.TaxTotalMinor != 187 || pricing.GrandTotalMinor != 2057 {
		t.Fatalf("unexpected pricing preview: %+v", pricing)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-issue-pricing-precheck"), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if precheck.CurrencyCode != "RUB" || precheck.Subtotal != 2000 || precheck.DiscountTotal != 300 || precheck.SurchargeTotal != 170 || precheck.TaxTotal != 187 || precheck.Total != 2057 || precheck.RemainingTotal != 2057 {
		t.Fatalf("unexpected priced precheck: %+v", precheck)
	}
	for table, want := range map[string]int{
		"precheck_lines":      1,
		"precheck_discounts":  1,
		"precheck_surcharges": 1,
		"precheck_taxes":      1,
	} {
		if got := countRows(t, f, table); got != want {
			t.Fatalf("expected %s rows %d, got %d", table, want, got)
		}
	}
	var discountIndex, surchargeIndex int
	if err := f.db.QueryRowContext(f.ctx, `SELECT application_index FROM precheck_discounts WHERE precheck_id = ?`, precheck.ID).Scan(&discountIndex); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT application_index FROM precheck_surcharges WHERE precheck_id = ?`, precheck.ID).Scan(&surchargeIndex); err != nil {
		t.Fatal(err)
	}
	if discountIndex != 10 || surchargeIndex != 20 {
		t.Fatalf("expected precheck modifier application indexes 10/20, got %d/%d", discountIndex, surchargeIndex)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE menu_items SET price = 9999 WHERE id = ?`, f.menuItem.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE tax_rules SET rate_basis_points = 5000 WHERE id = 'tax-rule-1'`); err != nil {
		t.Fatal(err)
	}
	storedPrecheck, err := f.service.GetPrecheck(f.ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedPrecheck.Total != 2057 || storedPrecheck.TaxTotal != 187 {
		t.Fatalf("precheck snapshot was mutated by menu/tax changes: %+v", storedPrecheck)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-pay-priced-precheck"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check.Total != precheck.Total || check.TaxTotal != precheck.TaxTotal || check.SurchargeTotal != precheck.SurchargeTotal || check.PaidTotal != precheck.Total {
		t.Fatalf("check did not use precheck snapshot totals: check=%+v precheck=%+v", check, precheck)
	}
}

func TestPricingRejectsDuplicateApplicationIndexAcrossDiscountsAndSurcharges(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddDiscount(f.ctx, app.AddDiscountCommand{
		CommandMeta:      f.edgeMetaCommand("cmd-duplicate-index-discount"),
		OrderID:          order.ID,
		Scope:            domain.DiscountScopeOrder,
		ApplicationIndex: 10,
		AmountKind:       domain.AmountFixed,
		AmountMinor:      100,
	}); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.AddSurcharge(f.ctx, app.AddSurchargeCommand{
		CommandMeta:      f.edgeMetaCommand("cmd-duplicate-index-surcharge"),
		OrderID:          order.ID,
		Kind:             domain.SurchargeManual,
		ApplicationIndex: 10,
		AmountKind:       domain.AmountFixed,
		AmountMinor:      50,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected duplicate application_index to be invalid, got %v", err)
	}
	if got := countRows(t, f, "order_surcharges"); got != 0 {
		t.Fatalf("expected duplicate surcharge not to be persisted, got %d rows", got)
	}
}

func TestCannotAddLineToLockedOrder(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
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

func TestChangeOrderLineQuantityUpdatesTotalAndWritesAuditMetadata(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{CommandMeta: f.edgeMeta(), PIN: "1111"})
	if err != nil {
		t.Fatal(err)
	}
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}

	meta := f.edgeMetaCommand("cmd-change-line-quantity")
	meta.SessionID = login.Session.ID
	changed, err := f.service.ChangeOrderLineQuantity(f.ctx, app.ChangeOrderLineQuantityCommand{
		CommandMeta: meta,
		OrderID:     order.ID,
		LineID:      line.ID,
		Quantity:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Quantity != 3 || changed.TotalPrice != 3000 || changed.Status != domain.OrderLineActive {
		t.Fatalf("unexpected changed line: %+v", changed)
	}
	var actorID, sessionID string
	if err := f.db.QueryRowContext(f.ctx, `SELECT actor_employee_id, session_id FROM local_event_log WHERE command_id = ? AND event_type = 'OrderLineQuantityChanged'`, "cmd-change-line-quantity").Scan(&actorID, &sessionID); err != nil {
		t.Fatal(err)
	}
	if actorID != f.employee.ID || sessionID != login.Session.ID {
		t.Fatalf("expected line edit actor metadata, got actor=%s session=%s", actorID, sessionID)
	}
}

func TestCannotChangeOrderLineQuantityForLockedOrder(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}

	_, err = f.service.ChangeOrderLineQuantity(f.ctx, app.ChangeOrderLineQuantityCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, LineID: line.ID, Quantity: 2})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestVoidOrderLineKeepsRowAndBlocksSecondVoid(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	linesBefore := countRows(t, f, "order_lines")

	voided, err := f.service.VoidOrderLine(f.ctx, app.VoidOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, LineID: line.ID, Reason: "mistake"})
	if err != nil {
		t.Fatal(err)
	}
	if voided.Status != domain.OrderLineVoided {
		t.Fatalf("expected voided line, got %+v", voided)
	}
	if lines := countRows(t, f, "order_lines"); lines != linesBefore {
		t.Fatalf("expected void to keep order line row, before=%d after=%d", linesBefore, lines)
	}
	_, err = f.service.VoidOrderLine(f.ctx, app.VoidOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, LineID: line.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict on second void, got %v", err)
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
			order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
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
				CommandMeta: f.edgeMetaCommand("cmd-precheck-fails-" + tc.name),
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
		CommandMeta:        f.managerMetaCommand("cmd-cancel-missing-precheck"),
		PrecheckID:         "missing-precheck",
		ManagerPIN:         "2468",
		CancellationReason: "guest changed order",
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
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-before-cancel"),
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	auditBefore := countRows(t, f, "manager_override_audit")

	cancelled, err := f.service.CancelPrecheck(f.ctx, f.cancelPrecheckCommand("cmd-cancel-precheck-1", precheck.ID))
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
	if audit := countRows(t, f, "manager_override_audit"); audit != auditBefore+1 {
		t.Fatalf("expected one manager override audit row, before=%d after=%d", auditBefore, audit)
	}
	var eventCount int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE command_id = ? AND event_type = 'PrecheckCancelled'`, "cmd-cancel-precheck-1").Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("expected PrecheckCancelled local event, got %d", eventCount)
	}
	var auditPINCount int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM manager_override_audit WHERE precheck_id = ? AND manager_employee_id = ? AND action = 'cancel_precheck' AND reason = 'guest changed order'`, precheck.ID, f.manager.ID).Scan(&auditPINCount); err != nil {
		t.Fatal(err)
	}
	if auditPINCount != 1 {
		t.Fatalf("expected manager override audit without pin payload, got %d", auditPINCount)
	}
}

func TestCancelPrecheckRejectsWrongManagerPIN(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-before-wrong-pin"),
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := f.cancelPrecheckCommand("cmd-cancel-wrong-pin", precheck.ID)
	cmd.ManagerPIN = "0000"
	_, err = f.service.CancelPrecheck(f.ctx, cmd)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	got, err := f.repo.GetPrecheck(f.ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.PrecheckIssued {
		t.Fatalf("expected precheck to stay issued, got %s", got.Status)
	}
	if audit := countRows(t, f, "manager_override_audit"); audit != 0 {
		t.Fatalf("expected no manager override audit for wrong pin, got %d", audit)
	}
}

func TestCancelPrecheckRejectsPINWithoutManagerPermission(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-before-no-permission"),
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := f.cancelPrecheckCommand("cmd-cancel-no-permission", precheck.ID)
	cmd.ManagerPIN = "1111"
	_, err = f.service.CancelPrecheck(f.ctx, cmd)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestGetCurrentShiftRequiresPermission(t *testing.T) {
	f := newFixture(t)
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     seedMeta(f.device.ID),
		Name:            "no-shift-read",
		PermissionsJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "Denied Shift Reader",
		PINHash:      testPINHash(t, "2580", "denied-shift-reader-salt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-denied-shift-reader",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "2580",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := edgeMeta(f.device.ID)
	meta.CommandID = "cmd-current-shift-denied"
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = employee.ID
	meta.SessionID = login.Session.ID
	if _, err := f.service.GetCurrentShift(f.ctx, meta); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestListRecentShiftsRequiresPermission(t *testing.T) {
	f := newFixture(t)
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     seedMeta(f.device.ID),
		Name:            "no-shift-recent",
		PermissionsJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "Denied Recent Reader",
		PINHash:      testPINHash(t, "8520", "denied-recent-reader-salt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-denied-recent-reader",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "8520",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := edgeMeta(f.device.ID)
	meta.CommandID = "cmd-recent-shift-denied"
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = employee.ID
	meta.SessionID = login.Session.ID
	_, err = f.service.ListRecentShifts(f.ctx, app.ListRecentShiftsCommand{CommandMeta: meta, Limit: 5})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestCancelPrecheckRejectsActorWithoutOverridePermission(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-before-actor-without-permission"),
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := app.CancelPrecheckCommand{
		CommandMeta:        f.edgeMetaCommand("cmd-cancel-actor-without-permission"),
		PrecheckID:         precheck.ID,
		ManagerPIN:         "2468",
		CancellationReason: "actor has no override permission",
	}
	_, err = f.service.CancelPrecheck(f.ctx, cmd)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestCannotCancelNonIssuedPrecheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-for-non-issued-cancel"),
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CancelPrecheck(f.ctx, f.cancelPrecheckCommand("cmd-cancel-once", precheck.ID)); err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	_, err = f.service.CancelPrecheck(f.ctx, f.cancelPrecheckCommand("cmd-cancel-non-issued", precheck.ID))
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
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-paid-foundation"),
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

	_, err = f.service.CancelPrecheck(f.ctx, f.cancelPrecheckCommand("cmd-cancel-paid-foundation", precheck.ID))
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
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-before-cancel-outbox-fails"),
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	auditBefore := countRows(t, f, "manager_override_audit")
	service := app.NewService(outboxFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 6000}, fixedClock{})

	_, err = service.CancelPrecheck(f.ctx, f.cancelPrecheckCommand("cmd-cancel-outbox-fails", precheck.ID))
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
	if audit := countRows(t, f, "manager_override_audit"); audit != auditBefore {
		t.Fatalf("expected no partial manager override audit write, before=%d after=%d", auditBefore, audit)
	}
}

func TestDuplicateCancelPrecheckCommandIDDoesNotDoubleCancel(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-before-duplicate-cancel"),
		OrderID:     order.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	cmd := app.CancelPrecheckCommand{
		CommandMeta:        f.managerMetaCommand("cmd-cancel-duplicate"),
		PrecheckID:         precheck.ID,
		ManagerPIN:         "2468",
		CancellationReason: "guest changed order",
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
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
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
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-precheck-1"),
		OrderID:     order.ID,
	}); err != nil {
		t.Fatal(err)
	}

	_, err = f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMetaCommand("cmd-issue-precheck-2"),
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
	other, err := f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{CommandMeta: seedMeta(f.device.ID), Name: "Other", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), RestaurantID: other.ID, TableID: f.table.ID})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCannotOverpayPrecheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total + 1, Currency: "RUB"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestCapturePaymentCreatesFirstAttemptWithEdgeContext(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta:           f.edgeMeta(),
		PrecheckID:            precheck.ID,
		Method:                domain.PaymentCard,
		Amount:                precheck.Total,
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
	if payment.PrecheckID != precheck.ID {
		t.Fatalf("expected payment to reference precheck %s, got %s", precheck.ID, payment.PrecheckID)
	}
	var attempts int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM payment_attempts WHERE payment_id = ? AND attempt_no = 1 AND provider_transaction_id = 'txn-1'`, payment.ID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("expected first payment attempt, got %d", attempts)
	}
}

func TestBusinessDateLocalStandardBoundaryAppliesToPaymentAndCheck(t *testing.T) {
	f := newFixture(t)
	if _, err := f.db.ExecContext(f.ctx, `UPDATE restaurants SET business_day_mode = 'standard', business_day_boundary_local_time = '23:30' WHERE id = ?`, f.restaurant.ID); err != nil {
		t.Fatal(err)
	}
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	if payment.BusinessDateLocal != "2026-05-03" {
		t.Fatalf("expected standard boundary business date 2026-05-03, got %s", payment.BusinessDateLocal)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check.BusinessDateLocal != payment.BusinessDateLocal || check.ClosedAt.IsZero() {
		t.Fatalf("expected check business date and closed_at, got %+v", check)
	}
}

func TestBusinessDateLocal24x7UsesLocalCalendarDate(t *testing.T) {
	f := newFixture(t)
	if _, err := f.db.ExecContext(f.ctx, `UPDATE restaurants SET business_day_mode = '24_7', business_day_boundary_local_time = '23:30' WHERE id = ?`, f.restaurant.ID); err != nil {
		t.Fatal(err)
	}
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	if payment.BusinessDateLocal != "2026-05-04" {
		t.Fatalf("expected 24/7 local calendar business date 2026-05-04, got %s", payment.BusinessDateLocal)
	}
}

func TestReprintPrecheckUsesSnapshotAndWritesAuditEvent(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	document, err := f.service.ReprintPrecheck(f.ctx, app.ReprintPrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-reprint-precheck"), PrecheckID: precheck.ID})
	if err != nil {
		t.Fatal(err)
	}
	if document.CopyMarker != "COPY" || document.DocumentType != "precheck" || !json.Valid(document.Snapshot) {
		t.Fatalf("expected copy reprint document from snapshot, got %+v", document)
	}
	var eventCount int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE command_id = 'cmd-reprint-precheck' AND event_type = 'PrecheckReprinted' AND actor_employee_id = ?`, f.employee.ID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("expected one PrecheckReprinted audit event, got %d", eventCount)
	}
}

func TestReprintCheckRequiresManagerAndWritesAuditEvent(t *testing.T) {
	f := newFixture(t)
	_, check := f.createPaidOrder(t)
	if _, err := f.service.ReprintCheck(f.ctx, app.ReprintCheckCommand{CommandMeta: f.edgeMetaCommand("cmd-reprint-check-denied"), CheckID: check.ID}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected cashier check reprint to be forbidden, got %v", err)
	}
	document, err := f.service.ReprintCheck(f.ctx, app.ReprintCheckCommand{CommandMeta: f.managerMetaCommand("cmd-reprint-check"), CheckID: check.ID})
	if err != nil {
		t.Fatal(err)
	}
	if document.CopyMarker != "COPY" || document.DocumentType != "check" || !json.Valid(document.Snapshot) {
		t.Fatalf("expected check copy reprint document from snapshot, got %+v", document)
	}
	var eventCount int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE command_id = 'cmd-reprint-check' AND event_type = 'CheckReprinted' AND actor_employee_id = ?`, f.manager.ID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("expected one CheckReprinted audit event, got %d", eventCount)
	}
}

func TestCardPaymentRequiresManualCardPermission(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta: seedMeta(f.device.ID),
		Name:        "cash-only-payment",
		PermissionsJSON: appshared.PermissionsJSON(
			appshared.PermissionPaymentCash,
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  seedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "Cash Only",
		PINHash:      testPINHash(t, "8642", "cash-only-salt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-login-cash-only-payment",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "8642",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := edgeMeta(f.device.ID)
	meta.CommandID = "cmd-card-payment-denied"
	meta.ClientDeviceID = f.clientID
	meta.ActorEmployeeID = employee.ID
	meta.SessionID = login.Session.ID
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: meta, PrecheckID: precheck.ID, Method: domain.PaymentCard, Amount: precheck.Total, Currency: "RUB"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for card payment without card permission, got %v", err)
	}
}

func TestCapturePaymentRollbackRemovesAttemptPaymentOutboxAndLocalEvent(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	paymentsBefore := countRows(t, f, "payments")
	attemptsBefore := countRows(t, f, "payment_attempts")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	service := app.NewService(outboxFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 4000}, fixedClock{})

	_, err = service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMetaCommand("cmd-payment-outbox-fails"),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
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
	if err := f.db.QueryRowContext(f.ctx, `SELECT paid_total,status FROM prechecks WHERE id = ?`, precheck.ID).Scan(&paidTotal, &status); err != nil {
		t.Fatal(err)
	}
	if paidTotal != 0 || status != string(domain.PrecheckIssued) {
		t.Fatalf("expected precheck rollback, paid_total=%d status=%s", paidTotal, status)
	}
	if checks := countRows(t, f, "checks"); checks != 0 {
		t.Fatalf("expected no final check after rollback, got %d", checks)
	}
}

func TestRefundPaymentRejectsActiveIssuedPrecheckWithoutFinalCheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMetaCommand("cmd-refund-active-precheck-payment-capture"),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      1000,
		Currency:    "RUB",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.GetCheckByOrder(f.ctx, order.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected no final check for partial payment, got %v", err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	_, err = f.service.RefundPayment(f.ctx, app.RefundPaymentCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "cmd-refund-active-issued-precheck"),
		PaymentID:   payment.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected refund without finalized check to conflict, got %v", err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no refund outbox for active precheck payment, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no refund local event for active precheck payment, before=%d after=%d", eventsBefore, events)
	}
}

func TestRefundPaymentRequiresClosedOriginalShift(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMeta(),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "rub",
	})
	if err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check == nil {
		t.Fatal("expected check after full payment")
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	_, err = f.service.RefundPayment(f.ctx, app.RefundPaymentCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "refund-cmd"),
		PaymentID:   payment.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected refund before original shift close to conflict, got %v", err)
	}
	var precheckPaidTotal int64
	var precheckStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT paid_total, status FROM prechecks WHERE id = ?`, precheck.ID).Scan(&precheckPaidTotal, &precheckStatus); err != nil {
		t.Fatal(err)
	}
	if precheckPaidTotal != precheck.Total {
		t.Fatalf("expected immutable precheck paid_total %d, got %d", precheck.Total, precheckPaidTotal)
	}
	if precheckStatus != string(domain.PrecheckClosed) {
		t.Fatalf("expected precheck status closed, got %s", precheckStatus)
	}
	var paymentStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM payments WHERE id = ?`, payment.ID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != string(domain.PaymentCaptured) {
		t.Fatalf("expected payment to remain captured, got %s", paymentStatus)
	}
	var checkStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM checks WHERE id = ?`, check.ID).Scan(&checkStatus); err != nil {
		t.Fatal(err)
	}
	if checkStatus != string(domain.CheckPaid) {
		t.Fatalf("expected check to remain paid, got %s", checkStatus)
	}
	var attempts int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM payment_attempts WHERE payment_id = ? AND status = 'refunded'`, payment.ID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatalf("expected no refunded attempts before boundary, got %d", attempts)
	}
	if operations := countRows(t, f, "financial_operations"); operations != 0 {
		t.Fatalf("expected no financial operation before original shift %s is closed, got %d", shift.ID, operations)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no outbox write on rejected refund, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no local event on rejected refund, before=%d after=%d", eventsBefore, events)
	}
}

func TestRefundPaymentRequiresOpenCashSession(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMetaCommand("cmd-refund-no-cash-payment"),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "RUB",
	})
	if err != nil {
		t.Fatal(err)
	}
	f.closeCashSessionAndShift(t, shift, cashSession, "refund-no-open-cash-session")
	f.openShift(t)
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	_, err = f.service.RefundPayment(f.ctx, app.RefundPaymentCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "cmd-refund-without-open-cash-session"),
		PaymentID:   payment.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected refund without open cash session to conflict, got %v", err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no outbox write without cash session, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no local event without cash session, before=%d after=%d", eventsBefore, events)
	}
}

func TestRefundPaymentAfterFullPaymentWithCheck(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMeta(),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "rub",
	})
	if err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check == nil {
		t.Fatal("expected check after full payment")
	}
	f.closeCashSessionAndShift(t, shift, cashSession, "before-refund")
	refundShift := f.openShift(t)
	f.openCashSession(t)
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	refundedPayment, err := f.service.RefundPayment(f.ctx, app.RefundPaymentCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "refund-cmd"),
		PaymentID:   payment.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if refundedPayment.Status != domain.PaymentCaptured {
		t.Fatalf("expected legacy refund wrapper to keep payment captured, got %s", refundedPayment.Status)
	}
	var precheckPaidTotal int64
	var precheckStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT paid_total, status FROM prechecks WHERE id = ?`, precheck.ID).Scan(&precheckPaidTotal, &precheckStatus); err != nil {
		t.Fatal(err)
	}
	if precheckPaidTotal != precheck.Total {
		t.Fatalf("expected immutable precheck paid_total %d after refund ledger write, got %d", precheck.Total, precheckPaidTotal)
	}
	if precheckStatus != string(domain.PrecheckClosed) {
		t.Fatalf("expected precheck status closed, got %s", precheckStatus)
	}
	var checkPaidTotal int64
	var checkStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT paid_total, status FROM checks WHERE id = ?`, check.ID).Scan(&checkPaidTotal, &checkStatus); err != nil {
		t.Fatal(err)
	}
	if checkPaidTotal != check.Total {
		t.Fatalf("expected immutable check paid_total %d after refund ledger write, got %d", check.Total, checkPaidTotal)
	}
	if checkStatus != string(domain.CheckPaid) {
		t.Fatalf("expected check status paid, got %s", checkStatus)
	}
	var attempts int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM payment_attempts WHERE payment_id = ? AND status = 'refunded'`, payment.ID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatalf("expected no mutable refunded attempts, got %d", attempts)
	}
	operations, err := f.repo.ListFinancialOperationsByCheck(f.ctx, check.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 {
		t.Fatalf("expected one refund operation, got %d", len(operations))
	}
	operation := operations[0]
	if operation.Type != domain.FinancialOperationRefund || operation.Kind != domain.FinancialOperationFull || operation.Amount != payment.Amount || operation.InventoryDisposition != domain.InventoryNoStockEffect {
		t.Fatalf("unexpected refund operation: type=%s kind=%s amount=%d disposition=%s", operation.Type, operation.Kind, operation.Amount, operation.InventoryDisposition)
	}
	if operation.ShiftID != refundShift.ID || operation.OriginalShiftID != shift.ID {
		t.Fatalf("unexpected operation shift boundary: shift_id=%s original_shift_id=%s", operation.ShiftID, operation.OriginalShiftID)
	}
	if operation.Reason != "legacy_payment_refund" {
		t.Fatalf("expected default legacy reason, got %q", operation.Reason)
	}
	if len(operation.Items) != 1 || operation.Items[0].Scope != domain.FinancialItemPayment || operation.Items[0].PaymentID == nil || *operation.Items[0].PaymentID != payment.ID {
		t.Fatalf("expected one payment-scope operation item, got %+v", operation.Items)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one refund outbox envelope, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one refund local event, before=%d after=%d", eventsBefore, events)
	}
	var refundEvents int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE event_type = 'RefundRecorded' AND aggregate_type = 'FinancialOperation' AND aggregate_id = ?`, operation.ID).Scan(&refundEvents); err != nil {
		t.Fatal(err)
	}
	if refundEvents != 1 {
		t.Fatalf("expected one RefundRecorded local event, got %d", refundEvents)
	}
	var refundOutbox int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM pos_sync_outbox WHERE command_id = 'refund-cmd' AND command_type = 'RefundRecorded' AND aggregate_type = 'FinancialOperation' AND aggregate_id = ? AND sync_direction = 'edge_to_cloud'`, operation.ID).Scan(&refundOutbox); err != nil {
		t.Fatal(err)
	}
	if refundOutbox != 1 {
		t.Fatalf("expected one edge_to_cloud RefundRecorded outbox row, got %d", refundOutbox)
	}
	var legacyEvents int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE command_id = 'refund-cmd' AND event_type IN ('PaymentRefunded','CheckRefunded')`).Scan(&legacyEvents); err != nil {
		t.Fatal(err)
	}
	if legacyEvents != 0 {
		t.Fatalf("expected compatibility refund to emit RefundRecorded, not legacy PaymentRefunded/CheckRefunded, got %d legacy events", legacyEvents)
	}
	var orderStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM orders WHERE id = ?`, order.ID).Scan(&orderStatus); err != nil {
		t.Fatal(err)
	}
	if orderStatus != string(domain.OrderClosed) {
		t.Fatalf("expected refund to preserve closed order status, got %s", orderStatus)
	}
	outboxAfterRefund := countRows(t, f, "pos_sync_outbox")
	eventsAfterRefund := countRows(t, f, "local_event_log")
	operationsAfterRefund := countRows(t, f, "financial_operations")
	_, err = f.service.RefundPayment(f.ctx, app.RefundPaymentCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "refund-cmd-repeat"),
		PaymentID:   payment.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected repeated full refund to conflict, got %v", err)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxAfterRefund {
		t.Fatalf("expected repeated refund not to write outbox, before=%d after=%d", outboxAfterRefund, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsAfterRefund {
		t.Fatalf("expected repeated refund not to write local event, before=%d after=%d", eventsAfterRefund, events)
	}
	if operations := countRows(t, f, "financial_operations"); operations != operationsAfterRefund {
		t.Fatalf("expected repeated refund not to write operation, before=%d after=%d", operationsAfterRefund, operations)
	}
}

func TestRecordCancellationDuringOpenShiftSupportsFullAndRejectsOverCancel(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMeta(),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "RUB",
	})
	if err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	operation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-full"),
		CheckID:              check.ID,
		Reason:               "guest cancelled before shift close",
		InventoryDisposition: domain.InventoryNoStockEffect,
	})
	if err != nil {
		t.Fatal(err)
	}
	if operation.Type != domain.FinancialOperationCancellation || operation.Kind != domain.FinancialOperationFull || operation.Amount != check.Total {
		t.Fatalf("unexpected full cancellation operation: type=%s kind=%s amount=%d", operation.Type, operation.Kind, operation.Amount)
	}
	if operation.Status != domain.FinancialOperationRecorded || len(operation.Items) != 1 || operation.Items[0].Scope != domain.FinancialItemWholeCheck {
		t.Fatalf("unexpected cancellation ledger shape: status=%s items=%+v", operation.Status, operation.Items)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one cancellation outbox envelope, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one cancellation local event, before=%d after=%d", eventsBefore, events)
	}
	var cancellationEvents int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE event_type = 'CancellationRecorded' AND aggregate_type = 'FinancialOperation' AND aggregate_id = ?`, operation.ID).Scan(&cancellationEvents); err != nil {
		t.Fatal(err)
	}
	if cancellationEvents != 1 {
		t.Fatalf("expected one CancellationRecorded local event, got %d", cancellationEvents)
	}
	var cancellationOutbox int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM pos_sync_outbox WHERE command_id = 'cmd-cancel-full' AND command_type = 'CancellationRecorded' AND aggregate_type = 'FinancialOperation' AND aggregate_id = ? AND sync_direction = 'edge_to_cloud'`, operation.ID).Scan(&cancellationOutbox); err != nil {
		t.Fatal(err)
	}
	if cancellationOutbox != 1 {
		t.Fatalf("expected one edge_to_cloud CancellationRecorded outbox row, got %d", cancellationOutbox)
	}
	var legacyEvents int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE command_id = 'cmd-cancel-full' AND event_type IN ('PaymentRefunded','CheckRefunded')`).Scan(&legacyEvents); err != nil {
		t.Fatal(err)
	}
	if legacyEvents != 0 {
		t.Fatalf("expected cancellation to emit CancellationRecorded only, got %d legacy events", legacyEvents)
	}
	if _, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "cmd-cancel-full"),
		CheckID:     check.ID,
		Reason:      "same offline command replay",
	}); !errors.Is(err, domain.ErrDuplicateCommand) {
		t.Fatalf("expected duplicate cancellation command to be rejected, got %v", err)
	}
	if _, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta: f.managerEdgeMetaCommand(t, "cmd-cancel-over"),
		CheckID:     check.ID,
		Reason:      "second cancellation exceeds check",
	}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected over-cancel to conflict, got %v", err)
	}
	if operations := countRows(t, f, "financial_operations"); operations != 1 {
		t.Fatalf("expected one cancellation operation, got %d", operations)
	}
	var paymentStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM payments WHERE id = ?`, payment.ID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != string(domain.PaymentCaptured) {
		t.Fatalf("expected finalized payment to remain captured, got %s", paymentStatus)
	}
	var checkStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM checks WHERE id = ?`, check.ID).Scan(&checkStatus); err != nil {
		t.Fatal(err)
	}
	if checkStatus != string(domain.CheckPaid) {
		t.Fatalf("expected finalized check to remain paid, got %s", checkStatus)
	}
}

func TestListFinancialOperationsAsOperatorFiltersAndPaginates(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	lines, err := f.repo.ListOrderLines(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected one order line, got %+v", lines)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMetaCommand("cmd-ledger-list-payment"),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "RUB",
	})
	if err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	cancellation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-ledger-list-cancel"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryManualReview,
		Reason:               "partial cancellation for list filters",
		Items: []app.FinancialOperationItemCommand{{
			Scope:       domain.FinancialItemOrderLine,
			OrderLineID: lines[0].ID,
			Quantity:    1,
			Amount:      lines[0].UnitPrice,
			Currency:    "RUB",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	f.closeCashSessionAndShift(t, shift, cashSession, "ledger-list")
	refundShift := f.openShift(t)
	f.openCashSession(t)
	refund, err := f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-ledger-list-refund"),
		CheckID:              check.ID,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "remaining refund for list filters",
	})
	if err != nil {
		t.Fatal(err)
	}

	all, err := f.service.ListFinancialOperationsAsOperator(f.ctx, app.ListFinancialOperationsCommand{
		CommandMeta:      f.managerMetaCommand("cmd-ledger-list-read-all"),
		BusinessDateFrom: "2026-05-04",
		BusinessDateTo:   "2026-05-04",
		CheckID:          check.ID,
		Limit:            10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected two ledger operations, got %+v", all)
	}
	refunds, err := f.service.ListFinancialOperationsAsOperator(f.ctx, app.ListFinancialOperationsCommand{
		CommandMeta:   f.managerMetaCommand("cmd-ledger-list-read-refunds"),
		OperationType: domain.FinancialOperationRefund,
		ShiftID:       refundShift.ID,
		Limit:         10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(refunds) != 1 || refunds[0].ID != refund.ID || refunds[0].Type != domain.FinancialOperationRefund {
		t.Fatalf("unexpected refund filter result: %+v", refunds)
	}
	originalShift, err := f.service.ListFinancialOperationsAsOperator(f.ctx, app.ListFinancialOperationsCommand{
		CommandMeta:     f.managerMetaCommand("cmd-ledger-list-read-original-shift"),
		OriginalShiftID: shift.ID,
		Limit:           10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(originalShift) != 2 {
		t.Fatalf("expected both operations by original shift, got %+v", originalShift)
	}
	secondPage, err := f.service.ListFinancialOperationsAsOperator(f.ctx, app.ListFinancialOperationsCommand{
		CommandMeta: f.managerMetaCommand("cmd-ledger-list-read-page"),
		CheckID:     check.ID,
		Limit:       1,
		Offset:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage) != 1 || secondPage[0].ID == refund.ID {
		t.Fatalf("expected second page to advance past latest refund, got %+v", secondPage)
	}
	if cancellation.Type != domain.FinancialOperationCancellation {
		t.Fatalf("expected cancellation operation fixture, got %+v", cancellation)
	}

	var checkStatus, precheckStatus, paymentStatus string
	var checkPaidTotal, precheckPaidTotal int64
	if err := f.db.QueryRowContext(f.ctx, `SELECT status, paid_total FROM checks WHERE id = ?`, check.ID).Scan(&checkStatus, &checkPaidTotal); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT status, paid_total FROM prechecks WHERE id = ?`, precheck.ID).Scan(&precheckStatus, &precheckPaidTotal); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM payments WHERE id = ?`, payment.ID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if checkStatus != string(domain.CheckPaid) || checkPaidTotal != check.Total || precheckStatus != string(domain.PrecheckClosed) || precheckPaidTotal != precheck.Total || paymentStatus != string(domain.PaymentCaptured) {
		t.Fatalf("ledger reads/writes must not mutate finalized docs, check=%s/%d precheck=%s/%d payment=%s", checkStatus, checkPaidTotal, precheckStatus, precheckPaidTotal, paymentStatus)
	}
}

func TestFinancialOperationBoundarySeparatesCancellationAndRefund(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-same-open-shift"),
		CheckID:              check.ID,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "same shift refund should be cancellation",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected same business-day refund in original open shift to conflict, got %v", err)
	}

	f.closeCashSessionAndShift(t, shift, cashSession, "before-cancel-boundary")
	f.openShift(t)
	f.openCashSession(t)
	_, err = f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-after-shift-close"),
		CheckID:              check.ID,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "cancellation after original shift close",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected cancellation after original shift close to conflict, got %v", err)
	}
}

func TestRecordRefundAllowsLaterBusinessDateWithOpenCashSession(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	execTestSQL(t, f, `UPDATE checks SET business_date_local = '2026-05-03' WHERE id = ?`, check.ID)

	operation, err := f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-later-business-date"),
		CheckID:              check.ID,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "later business date refund",
	})
	if err != nil {
		t.Fatal(err)
	}
	if operation.Type != domain.FinancialOperationRefund || operation.BusinessDateLocal != "2026-05-04" {
		t.Fatalf("unexpected later business date refund operation: %+v", operation)
	}
}

func TestRecordRefundRejectsAmountAboveCheckTotal(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.closeCashSessionAndShift(t, shift, cashSession, "before-over-refund")
	f.openShift(t)
	f.openCashSession(t)
	operationsBefore := countRows(t, f, "financial_operations")
	itemsBefore := countRows(t, f, "financial_operation_items")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	_, err = f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-above-check-total"),
		CheckID:              check.ID,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "over refund must be rejected",
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   check.Total + 1,
			Currency: "RUB",
		}},
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected refund above check total to conflict, got %v", err)
	}
	if operations := countRows(t, f, "financial_operations"); operations != operationsBefore {
		t.Fatalf("expected no refund operation write, before=%d after=%d", operationsBefore, operations)
	}
	if items := countRows(t, f, "financial_operation_items"); items != itemsBefore {
		t.Fatalf("expected no refund item write, before=%d after=%d", itemsBefore, items)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no refund outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no refund local event write, before=%d after=%d", eventsBefore, events)
	}
	total, err := f.repo.SumFinancialOperationAmountByCheck(f.ctx, check.ID, domain.FinancialOperationRefund)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("expected refund ledger to stay empty, got total=%d", total)
	}
	var paymentStatus, checkStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM payments WHERE id = ?`, payment.ID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM checks WHERE id = ?`, check.ID).Scan(&checkStatus); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != string(domain.PaymentCaptured) || checkStatus != string(domain.CheckPaid) {
		t.Fatalf("rejected refund must not mutate finalized docs, payment=%s check=%s", paymentStatus, checkStatus)
	}
}

func TestRecordCancellationSupportsPartialLineQuantityAndInventoryDisposition(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMeta(),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "RUB",
	}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	quantity := int64(1)
	_, err = f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-line-over-amount"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryManualReview,
		Reason:               "line amount exceeds selected quantity",
		Items: []app.FinancialOperationItemCommand{{
			Scope:       domain.FinancialItemOrderLine,
			OrderLineID: line.ID,
			Quantity:    quantity,
			Amount:      1001,
			Currency:    "RUB",
		}},
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected line amount over-cancel to conflict, got %v", err)
	}
	operation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-partial-line"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryWriteOffWaste,
		Reason:               "one prepared item cancelled",
		Items: []app.FinancialOperationItemCommand{{
			Scope:       domain.FinancialItemOrderLine,
			OrderLineID: line.ID,
			Quantity:    quantity,
			Amount:      1000,
			Currency:    "RUB",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if operation.Kind != domain.FinancialOperationPartial || operation.InventoryDisposition != domain.InventoryWriteOffWaste {
		t.Fatalf("unexpected partial cancellation operation: kind=%s disposition=%s", operation.Kind, operation.InventoryDisposition)
	}
	if len(operation.Items) != 1 || operation.Items[0].Quantity == nil || *operation.Items[0].Quantity != quantity || operation.Items[0].OrderLineID == nil || *operation.Items[0].OrderLineID != line.ID {
		t.Fatalf("unexpected partial line item: %+v", operation.Items)
	}
	_, err = f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-line-over-quantity"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryManualReview,
		Reason:               "line quantity replay",
		Items: []app.FinancialOperationItemCommand{{
			Scope:       domain.FinancialItemOrderLine,
			OrderLineID: line.ID,
			Quantity:    2,
			Amount:      500,
			Currency:    "RUB",
		}},
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected line quantity over-cancel to conflict, got %v", err)
	}
}

func TestRecordRefundAfterShiftCloseSupportsRepeatedPartialRefunds(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.closeCashSessionAndShift(t, shift, cashSession, "partial-refunds")
	f.openShift(t)
	f.openCashSession(t)
	for i, amount := range []int64{700, 300} {
		operation, err := f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
			CommandMeta:          f.managerEdgeMetaCommand(t, fmt.Sprintf("cmd-refund-partial-%d", i)),
			CheckID:              check.ID,
			OperationKind:        domain.FinancialOperationPartial,
			InventoryDisposition: domain.InventoryManualReview,
			Reason:               "partial guest return",
			Items: []app.FinancialOperationItemCommand{{
				Scope:    domain.FinancialItemWholeCheck,
				Amount:   amount,
				Currency: "RUB",
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if operation.Type != domain.FinancialOperationRefund || operation.Kind != domain.FinancialOperationPartial || operation.Amount != amount {
			t.Fatalf("unexpected partial refund operation: type=%s kind=%s amount=%d", operation.Type, operation.Kind, operation.Amount)
		}
	}
	_, err = f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-over"),
		CheckID:              check.ID,
		InventoryDisposition: domain.InventoryManualReview,
		Reason:               "over refund",
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   1001,
			Currency: "RUB",
		}},
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected over-refund to conflict, got %v", err)
	}
	total, err := f.repo.SumFinancialOperationAmountByCheck(f.ctx, check.ID, domain.FinancialOperationRefund)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1000 {
		t.Fatalf("expected recorded partial refunds to sum to 1000, got %d", total)
	}
}

func TestMixedCancellationAndRefundShareCheckAndLineQuantityCaps(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-mixed-line"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryManualReview,
		Reason:               "prepared item cancellation",
		Items: []app.FinancialOperationItemCommand{{
			Scope:       domain.FinancialItemOrderLine,
			OrderLineID: line.ID,
			Quantity:    1,
			Amount:      1000,
			Currency:    "RUB",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	f.closeCashSessionAndShift(t, shift, cashSession, "mixed-line-refund")
	f.openShift(t)
	f.openCashSession(t)
	_, err = f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-mixed-line-over-quantity"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryManualReview,
		Reason:               "over mixed line quantity",
		Items: []app.FinancialOperationItemCommand{{
			Scope:       domain.FinancialItemOrderLine,
			OrderLineID: line.ID,
			Quantity:    2,
			Amount:      1000,
			Currency:    "RUB",
		}},
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected mixed line quantity cap conflict, got %v", err)
	}

	_, err = f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-mixed-check-over"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryManualReview,
		Reason:               "over mixed check amount",
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   1001,
			Currency: "RUB",
		}},
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected mixed check amount cap conflict, got %v", err)
	}
}

func TestRecordRefundSupportsMixedPaymentAllocations(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	cashPayment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-split-cash"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: 400, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	cardPayment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-split-card"), PrecheckID: precheck.ID, Method: domain.PaymentCard, Amount: 600, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.closeCashSessionAndShift(t, shift, cashSession, "mixed-refund")
	f.openShift(t)
	f.openCashSession(t)
	operation, err := f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-cash-payment"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "cash tender return",
		Items: []app.FinancialOperationItemCommand{{
			Scope:     domain.FinancialItemPayment,
			PaymentID: cashPayment.ID,
			Amount:    cashPayment.Amount,
			Currency:  "RUB",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(operation.Items) != 1 || operation.Items[0].PaymentID == nil || *operation.Items[0].PaymentID != cashPayment.ID {
		t.Fatalf("expected refund item to be allocated to cash payment, got %+v", operation.Items)
	}
	if cardPayment.Status != domain.PaymentCaptured {
		t.Fatalf("expected card payment to remain captured, got %s", cardPayment.Status)
	}
	_, err = f.service.RecordRefund(f.ctx, app.RecordCheckRefundCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-refund-cash-payment-over"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "cash tender duplicate",
		Items: []app.FinancialOperationItemCommand{{
			Scope:     domain.FinancialItemPayment,
			PaymentID: cashPayment.ID,
			Amount:    1,
			Currency:  "RUB",
		}},
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected repeated refund over original tender to conflict, got %v", err)
	}
}

func TestFinancialOperationLedgerIsAppendOnlyAndPreservesSnapshot(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE menu_items SET name = 'Changed soup', price = 9999 WHERE id = ?`, f.menuItem.ID); err != nil {
		t.Fatal(err)
	}
	operation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-snapshot"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryReturnToStock,
		Reason:               "snapshot and inventory disposition check",
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   100,
			Currency: "RUB",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := string(operation.Snapshot)
	if !strings.Contains(snapshot, `"name":"Soup"`) {
		t.Fatalf("expected financial operation snapshot to preserve original commercial name, got %s", snapshot)
	}
	if strings.Contains(snapshot, "Changed soup") || strings.Contains(snapshot, "9999") {
		t.Fatalf("expected financial operation snapshot not to use changed menu data, got %s", snapshot)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE financial_operations SET reason = 'edited' WHERE id = ?`, operation.ID); err == nil {
		t.Fatal("expected financial_operations update to be rejected by append-only trigger")
	}
	if _, err := f.db.ExecContext(f.ctx, `DELETE FROM financial_operation_items WHERE operation_id = ?`, operation.ID); err == nil {
		t.Fatal("expected financial_operation_items delete to be rejected by append-only trigger")
	}
}

func TestFinancialOperationModifierAndTipScopesRequireExplicitSnapshot(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-modifier-no-snapshot"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "modifier scope without snapshot",
		Items: []app.FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemModifierLine,
			Amount:   50,
			Currency: "RUB",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "snapshot") {
		t.Fatalf("expected explicit snapshot validation for modifier/tip scopes, got %v", err)
	}
	operation, err := f.service.RecordCancellation(f.ctx, app.RecordCheckCancellationCommand{
		CommandMeta:          f.managerEdgeMetaCommand(t, "cmd-cancel-modifier-tip"),
		CheckID:              check.ID,
		OperationKind:        domain.FinancialOperationPartial,
		InventoryDisposition: domain.InventoryNoStockEffect,
		Reason:               "modifier and tip scope ledger",
		Items: []app.FinancialOperationItemCommand{
			{
				Scope:    domain.FinancialItemModifierLine,
				Amount:   50,
				Currency: "RUB",
				Snapshot: json.RawMessage(`{"modifier_name":"extra herbs","unit_price_minor":50}`),
			},
			{
				Scope:    domain.FinancialItemTip,
				Amount:   50,
				Currency: "RUB",
				Snapshot: json.RawMessage(`{"tip_policy":"manual","amount_minor":50}`),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(operation.Items) != 2 {
		t.Fatalf("expected modifier and tip operation items, got %+v", operation.Items)
	}
	if operation.Items[0].Scope != domain.FinancialItemModifierLine || !strings.Contains(string(operation.Items[0].Snapshot), "extra herbs") {
		t.Fatalf("unexpected modifier snapshot: %+v", operation.Items[0])
	}
	if operation.Items[1].Scope != domain.FinancialItemTip || !strings.Contains(string(operation.Items[1].Snapshot), "tip_policy") {
		t.Fatalf("unexpected tip snapshot: %+v", operation.Items[1])
	}
}

func TestRefundPaymentRequiresPermission(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	payment, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMeta(),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "rub",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Attempt refund with cashier permissions (which lack pos.payment.refund)
	_, err = f.service.RefundPayment(f.ctx, app.RefundPaymentCommand{
		CommandMeta: f.edgeMeta(),
		PaymentID:   payment.ID,
	})
	if err == nil {
		t.Fatal("expected permission denied error for refund without pos.payment.refund permission")
	}
	if !strings.Contains(err.Error(), "permission") {
		t.Fatalf("expected permission error, got %v", err)
	}
}

func TestFullPaymentRollsBackFinalCheckWhenCheckCreatedOutboxFails(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	paymentsBefore := countRows(t, f, "payments")
	checksBefore := countRows(t, f, "checks")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	service := app.NewService(checkCreatedOutboxFailingRepo{Repository: f.repo}, platformsqlite.NewTxManager(f.db), &testIDs{n: 7000}, fixedClock{})

	_, err = service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMetaCommand("cmd-payment-check-outbox-fails"),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "RUB",
	})
	if !errors.Is(err, errInjectedOutbox) {
		t.Fatalf("expected injected outbox failure, got %v", err)
	}
	if payments := countRows(t, f, "payments"); payments != paymentsBefore {
		t.Fatalf("expected payment rollback, before=%d after=%d", paymentsBefore, payments)
	}
	if checks := countRows(t, f, "checks"); checks != checksBefore {
		t.Fatalf("expected final check rollback, before=%d after=%d", checksBefore, checks)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected outbox rollback, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected local event rollback, before=%d after=%d", eventsBefore, events)
	}
	got, err := f.repo.GetPrecheck(f.ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaidTotal != 0 || got.Status != domain.PrecheckIssued {
		t.Fatalf("expected precheck rollback, got %+v", got)
	}
}

func TestPartialPaymentKeepsPrecheckOpenAndDoesNotCreateFinalCheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: 1000, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	got, err := f.repo.GetPrecheck(f.ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.PrecheckIssued || got.PaidTotal != 1000 {
		t.Fatalf("expected partial paid issued precheck, got %+v", got)
	}
	if checks := countRows(t, f, "checks"); checks != 0 {
		t.Fatalf("expected no final check before full payment, got %d", checks)
	}
	gotOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOrder.Status != domain.OrderLocked {
		t.Fatalf("expected order to stay locked, got %s", gotOrder.Status)
	}
}

func TestFullPaymentCreatesFinalCheckAndClosesOrder(t *testing.T) {
	f := newFixture(t)
	order, check := f.createPaidOrder(t)
	if check.Status != domain.CheckPaid || check.PaidTotal != check.Total {
		t.Fatalf("expected paid final check, got %+v", check)
	}
	gotOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOrder.Status != domain.OrderClosed || gotOrder.ClosedAt == nil {
		t.Fatalf("expected closed order, got %+v", gotOrder)
	}
}

func TestFullPaymentWritesCheckClosedOutboxFromImmutableCheckSnapshot(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-checkclosed-precheck"), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{
		CommandMeta: f.edgeMetaCommand("cmd-checkclosed-payment"),
		PrecheckID:  precheck.ID,
		Method:      domain.PaymentCash,
		Amount:      precheck.Total,
		Currency:    "RUB",
	}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	var snapshot struct {
		CheckID           string `json:"check_id"`
		OrderID           string `json:"order_id"`
		PrecheckID        string `json:"precheck_id"`
		RestaurantID      string `json:"restaurant_id"`
		BusinessDateLocal string `json:"business_date_local"`
		PrecheckSnapshot  struct {
			Lines []struct {
				OrderLineID   string `json:"order_line_id"`
				CatalogItemID string `json:"catalog_item_id"`
				Quantity      int64  `json:"quantity"`
			} `json:"lines"`
		} `json:"precheck_snapshot"`
	}
	if err := json.Unmarshal(check.Snapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.CheckID != check.ID || snapshot.OrderID != order.ID || snapshot.PrecheckID != precheck.ID {
		t.Fatalf("unexpected immutable check snapshot identity: %+v", snapshot)
	}
	if len(snapshot.PrecheckSnapshot.Lines) != 1 {
		t.Fatalf("expected one immutable snapshot line, got %+v", snapshot.PrecheckSnapshot.Lines)
	}
	snapshotLine := snapshot.PrecheckSnapshot.Lines[0]
	if snapshotLine.OrderLineID != line.ID || snapshotLine.CatalogItemID != f.menuItem.CatalogItemID || snapshotLine.Quantity != 2 {
		t.Fatalf("unexpected immutable snapshot line: %+v", snapshotLine)
	}

	var payloadJSON string
	if err := f.db.QueryRowContext(f.ctx, `SELECT payload_json FROM pos_sync_outbox WHERE command_type = 'CheckClosed' AND aggregate_type = 'Check' AND aggregate_id = ?`, check.ID).Scan(&payloadJSON); err != nil {
		t.Fatalf("expected CheckClosed outbox envelope for final check %s: %v", check.ID, err)
	}
	var envelope struct {
		Version       string `json:"version"`
		EventType     string `json:"event_type"`
		AggregateType string `json:"aggregate_type"`
		AggregateID   string `json:"aggregate_id"`
		RestaurantID  string `json:"restaurant_id"`
		DeviceID      string `json:"device_id"`
		Payload       struct {
			Origin string `json:"origin"`
			Data   struct {
				CheckID           string `json:"check_id"`
				OrderID           string `json:"order_id"`
				PrecheckID        string `json:"precheck_id"`
				RestaurantID      string `json:"restaurant_id"`
				BusinessDateLocal string `json:"business_date_local"`
				Items             []struct {
					OrderLineID          string `json:"order_line_id"`
					CatalogItemID        string `json:"catalog_item_id"`
					Quantity             string `json:"quantity"`
					UnitCode             string `json:"unit_code"`
					RequiredForInventory bool   `json:"required_for_inventory"`
				} `json:"items"`
			} `json:"data"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Version != domain.SyncEnvelopeVersion || envelope.EventType != "CheckClosed" || envelope.AggregateType != "Check" || envelope.AggregateID != check.ID {
		t.Fatalf("unexpected CheckClosed envelope metadata: %+v", envelope)
	}
	if envelope.Payload.Data.CheckID != snapshot.CheckID ||
		envelope.Payload.Data.OrderID != snapshot.OrderID ||
		envelope.Payload.Data.PrecheckID != snapshot.PrecheckID ||
		envelope.Payload.Data.BusinessDateLocal != snapshot.BusinessDateLocal {
		t.Fatalf("CheckClosed identity must come from immutable check snapshot, snapshot=%+v envelope=%+v", snapshot, envelope.Payload.Data)
	}
	if len(envelope.Payload.Data.Items) != len(snapshot.PrecheckSnapshot.Lines) {
		t.Fatalf("CheckClosed items must mirror immutable snapshot lines, snapshot=%+v envelope=%+v", snapshot.PrecheckSnapshot.Lines, envelope.Payload.Data.Items)
	}
	item := envelope.Payload.Data.Items[0]
	if item.OrderLineID != snapshotLine.OrderLineID || item.CatalogItemID != snapshotLine.CatalogItemID || item.Quantity != "2.000" || item.UnitCode == "" || !item.RequiredForInventory {
		t.Fatalf("CheckClosed item must be built from immutable snapshot line, snapshot=%+v item=%+v", snapshotLine, item)
	}
}

func TestListClosedOrdersUsesBoundedPaginationAndStableNewestFirstSort(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	base := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 105; i++ {
		insertClosedOrderFixture(t, f, shift.ID, f.device.ID, fmt.Sprintf("bulk-order-%03d", i), fmt.Sprintf("bulk-check-%03d", i), "2026-05-04", base.Add(time.Duration(i)*time.Minute))
	}

	defaultPage, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta()})
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultPage) != 50 {
		t.Fatalf("expected default page to be capped at 50, got %d", len(defaultPage))
	}
	if defaultPage[0].ID != "bulk-order-104" || defaultPage[49].ID != "bulk-order-055" {
		t.Fatalf("expected newest-first stable default page, first=%s last=%s", defaultPage[0].ID, defaultPage[49].ID)
	}

	maxPage, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len(maxPage) != 100 {
		t.Fatalf("expected max page to be capped at 100, got %d", len(maxPage))
	}

	offsetPage, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), Limit: 10, Offset: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(offsetPage) != 5 || offsetPage[0].ID != "bulk-order-004" || offsetPage[4].ID != "bulk-order-000" {
		t.Fatalf("unexpected offset page: %+v", offsetPage)
	}
}

func TestListClosedOrdersSupportsDateShiftDeviceAndCheckFilters(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "closed-1", "check-1", "2026-05-04", time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC))
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "closed-2", "check-2", "2026-05-05", time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC))
	insertClosedOrderFixture(t, f, shift.ID, f.device.ID, "closed-3", "check-3", "2026-05-06", time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC))

	sameDate, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), BusinessDateLocal: "2026-05-05", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(sameDate) != 1 || sameDate[0].Check == nil || sameDate[0].Check.ID != "check-2" {
		t.Fatalf("unexpected business date filter result: %+v", sameDate)
	}

	dateRange, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), FromBusinessDateLocal: "2026-05-04", ToBusinessDateLocal: "2026-05-05", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(dateRange) != 2 || dateRange[0].ID != "closed-2" || dateRange[1].ID != "closed-1" {
		t.Fatalf("unexpected date range result: %+v", dateRange)
	}

	byShift, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), ShiftID: shift.ID, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(byShift) != 3 {
		t.Fatalf("expected three rows for shift filter, got %d", len(byShift))
	}

	byDevice, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), DeviceID: f.device.ID, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(byDevice) != 3 {
		t.Fatalf("expected three rows for device filter, got %d", len(byDevice))
	}

	byCheck, err := f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), CheckID: "check-1", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(byCheck) != 1 || byCheck[0].ID != "closed-1" {
		t.Fatalf("unexpected check filter result: %+v", byCheck)
	}

	_, err = f.service.ListClosedOrders(f.ctx, app.ListClosedOrdersCommand{CommandMeta: f.edgeMeta(), BusinessDateLocal: "05-05-2026"})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid business date to be rejected, got %v", err)
	}
}

func TestPaymentForCancelledPrecheckRejected(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CancelPrecheck(f.ctx, f.cancelPrecheckCommand("cmd-cancel-before-payment", precheck.ID)); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestPaymentForSupersededPrecheckRejected(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := precheck.Supersede(fixedClock{}.Now()); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.UpdatePrecheckLifecycle(f.ctx, precheck); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if !errors.Is(err, domain.ErrNotFound) && !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected not found or conflict for inactive precheck, got %v", err)
	}
}

func TestDuplicatePaymentCommandIDDoesNotDoubleCapture(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	cmd := app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-payment-duplicate"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: 1000, Currency: "RUB"}
	if _, err := f.service.CapturePayment(f.ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, cmd); !errors.Is(err, domain.ErrDuplicateCommand) {
		t.Fatalf("expected duplicate command, got %v", err)
	}
	if payments := countRows(t, f, "payments"); payments != 1 {
		t.Fatalf("expected one payment, got %d", payments)
	}
	got, err := f.repo.GetPrecheck(f.ctx, precheck.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaidTotal != 1000 {
		t.Fatalf("expected paid_total 1000, got %d", got.PaidTotal)
	}
}

func TestPaymentRequiresActiveShiftAndMatchingDevice(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	wrongDeviceMeta := f.edgeMetaCommand("cmd-payment-wrong-device")
	wrongDeviceMeta.NodeDeviceID = "other-device"
	wrongDeviceMeta.DeviceID = "other-device"
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: wrongDeviceMeta, PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected session/device forbidden, got %v", err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE cash_sessions SET status = 'closed', closed_at = ?, updated_at = ? WHERE id = ?`, appshared.DBTime(fixedClock{}.Now()), appshared.DBTime(fixedClock{}.Now()), cashSession.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-payment-no-active-cash-session"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected active cash session conflict, got %v", err)
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

func TestOpenShiftUsesRestaurantIANATimezone(t *testing.T) {
	f := newFixture(t)
	shift, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if shift.BusinessDateLocal != "2026-05-04" {
		t.Fatalf("expected business date from Europe/Moscow timezone, got %s", shift.BusinessDateLocal)
	}
}

func TestApplyMasterDataSnapshotUpsertsRowsStateAndDoesNotCreateOutbox(t *testing.T) {
	f := newFixture(t)
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")
	applied, err := f.service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:        app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		SyncMode:           domain.SyncModeFullSnapshot,
		FullSnapshotReason: "terminal_restaurant_changed",
		CloudVersion:       42,
		Restaurants: []domain.Restaurant{{
			ID:       "cloud-restaurant-1",
			Name:     "Cloud Bistro",
			Timezone: "Europe/Moscow",
			Currency: "rub",
			Active:   true,
		}},
		Devices: []domain.Device{{
			ID:           "cloud-device-1",
			RestaurantID: "cloud-restaurant-1",
			DeviceCode:   "CLOUD-POS-1",
			Name:         "Cloud POS",
			Type:         "terminal",
			Active:       true,
		}},
		Roles: []domain.Role{{
			ID:              "cloud-role-1",
			Name:            "cloud-cashier",
			PermissionsJSON: appshared.PermissionsJSON(appshared.PermissionOrderCreate),
			Active:          true,
		}},
		Employees: []domain.Employee{{
			ID:           "cloud-employee-1",
			RestaurantID: "cloud-restaurant-1",
			RoleID:       "cloud-role-1",
			Name:         "Cloud Anna",
			PINHash:      testPINHash(t, "3333", "cloud-salt"),
			Active:       true,
		}},
		Halls: []domain.Hall{{
			ID:           "cloud-hall-1",
			RestaurantID: "cloud-restaurant-1",
			Name:         "Cloud Hall",
			Active:       true,
		}},
		Tables: []domain.Table{{
			ID:           "cloud-table-1",
			RestaurantID: "cloud-restaurant-1",
			HallID:       "cloud-hall-1",
			Name:         "C1",
			Seats:        4,
			Active:       true,
		}},
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-catalog-1",
			Type:     domain.CatalogItemDish,
			Name:     "Cloud Soup",
			SKU:      "CLOUD-SOUP",
			BaseUnit: "portion",
			Active:   true,
		}},
		MenuItems: []domain.MenuItem{{
			ID:            "cloud-menu-1",
			CatalogItemID: "cloud-catalog-1",
			Name:          "Cloud Soup",
			Price:         1200,
			Currency:      "rub",
			Active:        true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(applied.AppliedStreams), 6; got != want {
		t.Fatalf("expected %d applied streams, got %d: %+v", want, got, applied.AppliedStreams)
	}
	if outbox := countRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected master ingest not to create outbox rows, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected master ingest not to create local events, before=%d after=%d", eventsBefore, events)
	}
	var restaurantName, menuCurrency string
	var restaurantActive int
	var restaurantCloudVersion, menuCloudVersion int64
	if err := f.db.QueryRowContext(f.ctx, `SELECT name,active,cloud_version FROM restaurants WHERE id = 'cloud-restaurant-1'`).Scan(&restaurantName, &restaurantActive, &restaurantCloudVersion); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT currency,cloud_version FROM menu_items WHERE id = 'cloud-menu-1'`).Scan(&menuCurrency, &menuCloudVersion); err != nil {
		t.Fatal(err)
	}
	if restaurantName != "Cloud Bistro" || restaurantActive != 1 || restaurantCloudVersion != 42 || menuCurrency != "RUB" || menuCloudVersion != 42 {
		t.Fatalf("unexpected applied rows: restaurant=%q active=%d version=%d menu_currency=%q menu_version=%d", restaurantName, restaurantActive, restaurantCloudVersion, menuCurrency, menuCloudVersion)
	}
	if states := countRows(t, f, "cloud_master_sync_state"); states != 6 {
		t.Fatalf("expected six master sync states, got %d", states)
	}
	var wrongDirection int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM cloud_master_sync_state WHERE direction <> 'cloud_to_edge' OR status <> 'applied'`).Scan(&wrongDirection); err != nil {
		t.Fatal(err)
	}
	if wrongDirection != 0 {
		t.Fatalf("expected all master sync states applied/cloud_to_edge, got %d wrong rows", wrongDirection)
	}
}

func TestApplyMasterDataAcceptsCloudCatalogStreamPackageShape(t *testing.T) {
	f := newFixture(t)
	payload := []byte(`{
		"node_device_id":"` + f.device.ID + `",
		"restaurant_id":"cloud-restaurant-1",
		"stream":"catalog",
		"sync_mode":"incremental",
		"checkpoint_token":"master-data:cloud-restaurant-1:43",
		"cloud_version":43,
		"cloud_updated_at":"2026-05-09T10:00:00Z",
		"folders":[{"id":"folder-services","restaurant_id":"` + f.restaurant.ID + `","name":"Services","sort_order":10,"active":true}],
		"folder_parameters":[{"id":"folder-param-1","restaurant_id":"` + f.restaurant.ID + `","folder_id":"folder-services","parameter_key":"accounting_category","value_type":"string","value_json":"\"services\""}],
		"tags":[{"id":"tag-delivery","restaurant_id":"` + f.restaurant.ID + `","name":"Delivery","code":"delivery","active":true}],
		"catalog_items":[{
			"id":"cloud-service-1",
			"type":"service",
			"folder_id":"folder-services",
			"name":"Cloud Delivery",
			"sku":"CLOUD-DELIVERY",
			"base_unit":"service",
			"accounting_category":"services",
			"active":true,
			"created_at":"2026-05-09T10:00:00Z",
			"updated_at":"2026-05-09T10:00:00Z"
		}],
		"item_tags":[{"catalog_item_id":"cloud-service-1","tag_id":"tag-delivery","restaurant_id":"` + f.restaurant.ID + `"}],
		"modifier_groups":[{"id":"modifier-group-1","restaurant_id":"` + f.restaurant.ID + `","name":"Sauce","required":false,"min_count":0,"max_count":2,"active":true}],
		"modifier_options":[{"id":"modifier-option-1","restaurant_id":"` + f.restaurant.ID + `","modifier_group_id":"modifier-group-1","name":"Extra sauce","price_minor":0,"active":true}],
		"modifier_bindings":[{"id":"modifier-binding-1","restaurant_id":"` + f.restaurant.ID + `","modifier_group_id":"modifier-group-1","target_type":"tag","target_id":"tag-delivery","sort_order":10,"active":true}],
		"menu_item_modifier_groups":[{"menu_item_id":"` + f.menuItem.ID + `","modifier_group_id":"modifier-group-1","sort_order":10}]
	}`)
	if strings.Contains(string(payload), `"categories"`) {
		t.Fatalf("Cloud catalog stream fixture must not contain unsupported categories field: %s", payload)
	}
	var cmd app.ApplyMasterDataCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		t.Fatal(err)
	}
	cmd.CommandMeta = app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync}
	applied, err := f.service.ApplyMasterData(f.ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied.AppliedStreams) != 1 || applied.AppliedStreams[0] != domain.MasterDataStreamCatalog || applied.Counts["catalog"] != 9 {
		t.Fatalf("expected catalog stream apply, got %+v", applied)
	}
	var itemType, sku, folderID, accountingCategory string
	if err := f.db.QueryRowContext(f.ctx, `SELECT type,sku,folder_id,accounting_category FROM catalog_items WHERE id = 'cloud-service-1'`).Scan(&itemType, &sku, &folderID, &accountingCategory); err != nil {
		t.Fatal(err)
	}
	if itemType != string(domain.CatalogItemService) || sku != "CLOUD-DELIVERY" || folderID != "folder-services" || accountingCategory != "services" {
		t.Fatalf("expected service catalog row, got type=%q sku=%q folder=%q accounting=%q", itemType, sku, folderID, accountingCategory)
	}
	var modifierPrice int64
	if err := f.db.QueryRowContext(f.ctx, `SELECT price_minor FROM modifier_options WHERE id = 'modifier-option-1'`).Scan(&modifierPrice); err != nil {
		t.Fatal(err)
	}
	if modifierPrice != 0 {
		t.Fatalf("expected zero-price modifier option, got %d", modifierPrice)
	}
}

func TestApplySyncExchangeCloudPackagesQuarantinesBadPackageAndAppliesRest(t *testing.T) {
	f := newFixture(t)
	err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{
		{
			StreamName:      string(domain.MasterDataStreamCatalog),
			NodeDeviceID:    f.device.ID,
			RestaurantID:    f.restaurant.ID,
			SyncMode:        string(domain.SyncModeIncremental),
			CloudVersion:    70,
			CheckpointToken: "catalog:70",
			PayloadJSON:     json.RawMessage(`{"catalog_items":[{"id":"bad-catalog","type":"dish","name":"Broken","base_unit":"pc","active":true}]}`),
		},
		{
			StreamName:      string(domain.MasterDataStreamRestaurants),
			NodeDeviceID:    f.device.ID,
			RestaurantID:    f.restaurant.ID,
			SyncMode:        string(domain.SyncModeIncremental),
			CloudVersion:    71,
			CheckpointToken: "restaurants:71",
			PayloadJSON:     json.RawMessage(`{"restaurants":[{"id":"cloud-restaurant-ok","name":"Cloud OK","timezone":"Europe/Moscow","currency":"RUB","active":true}]}`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var restaurantName string
	if err := f.db.QueryRowContext(f.ctx, `SELECT name FROM restaurants WHERE id = 'cloud-restaurant-ok'`).Scan(&restaurantName); err != nil {
		t.Fatal(err)
	}
	if restaurantName != "Cloud OK" {
		t.Fatalf("expected valid package applied, got restaurant name %q", restaurantName)
	}
	var failedStatus, failedError string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status,last_error FROM cloud_master_sync_state WHERE node_device_id = ? AND stream_name = 'catalog'`, f.device.ID).Scan(&failedStatus, &failedError); err != nil {
		t.Fatal(err)
	}
	if failedStatus != "failed" || !strings.Contains(failedError, "catalog item") {
		t.Fatalf("expected catalog package quarantined, status=%q error=%q", failedStatus, failedError)
	}
}

func TestApplyMasterDataFullPackageAppliesMenuItemsBeforeModifierLinks(t *testing.T) {
	f := newFixture(t)
	payload := []byte(`{
		"node_device_id":"` + f.device.ID + `",
		"restaurant_id":"` + f.restaurant.ID + `",
		"sync_mode":"incremental",
		"checkpoint_token":"master-data:` + f.restaurant.ID + `:44",
		"cloud_version":44,
		"cloud_updated_at":"2026-05-09T10:00:00Z",
		"catalog_items":[{
			"id":"cloud-tea-catalog",
			"type":"dish",
			"name":"Cloud Tea",
			"sku":"CLOUD-TEA",
			"base_unit":"portion",
			"active":true,
			"created_at":"2026-05-09T10:00:00Z",
			"updated_at":"2026-05-09T10:00:00Z"
		}],
		"modifier_groups":[{"id":"cloud-modifier-group","restaurant_id":"` + f.restaurant.ID + `","name":"Add-ons","required":true,"min_count":1,"max_count":2,"active":true}],
		"modifier_options":[{"id":"cloud-modifier-option","restaurant_id":"` + f.restaurant.ID + `","modifier_group_id":"cloud-modifier-group","name":"Lemon","price_minor":3000,"active":true}],
		"modifier_bindings":[{"id":"cloud-modifier-binding","restaurant_id":"` + f.restaurant.ID + `","modifier_group_id":"cloud-modifier-group","target_type":"menu_item","target_id":"cloud-menu-item","sort_order":1,"active":true}],
		"menu_item_modifier_groups":[{"menu_item_id":"cloud-menu-item","modifier_group_id":"cloud-modifier-group","sort_order":1}],
		"menu_items":[{
			"id":"cloud-menu-item",
			"catalog_item_id":"cloud-tea-catalog",
			"name":"Cloud Tea",
			"price":15000,
			"currency":"RUB",
			"active":true,
			"created_at":"2026-05-09T10:00:00Z",
			"updated_at":"2026-05-09T10:00:00Z"
		}]
	}`)
	var cmd app.ApplyMasterDataCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		t.Fatal(err)
	}
	cmd.CommandMeta = app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync}
	applied, err := f.service.ApplyMasterData(f.ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied.AppliedStreams) != 2 || applied.AppliedStreams[0] != domain.MasterDataStreamCatalog || applied.AppliedStreams[1] != domain.MasterDataStreamMenu {
		t.Fatalf("expected catalog then menu streams, got %+v", applied.AppliedStreams)
	}
	var linkCount int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM menu_item_modifier_groups WHERE menu_item_id = 'cloud-menu-item' AND modifier_group_id = 'cloud-modifier-group'`).Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 1 {
		t.Fatalf("expected menu item modifier link to be applied after menu item, got %d rows", linkCount)
	}
}

func TestApplyMasterDataFullSnapshotCreatesBackupBeforeApply(t *testing.T) {
	f := newFixture(t)
	backupCalls := 0
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 8000}, fixedClock{}, app.ServiceOptions{
		MasterDataBackupBeforeFullSnapshot: func(ctx context.Context, req app.MasterDataBackupRequest) error {
			backupCalls++
			if req.NodeDeviceID != f.device.ID || req.CloudVersion != 77 || len(req.Streams) != 1 || req.Streams[0] != domain.MasterDataStreamCatalog {
				t.Fatalf("unexpected backup request: %+v", req)
			}
			var rowsBeforeApply int
			if err := f.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM catalog_items WHERE id = 'cloud-backup-before-apply'`).Scan(&rowsBeforeApply); err != nil {
				t.Fatal(err)
			}
			if rowsBeforeApply != 0 {
				t.Fatalf("expected backup before catalog row apply, got %d rows", rowsBeforeApply)
			}
			var statesBeforeApply int
			if err := f.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM cloud_master_sync_state WHERE node_device_id = ? AND stream_name = 'catalog'`, f.device.ID).Scan(&statesBeforeApply); err != nil {
				t.Fatal(err)
			}
			if statesBeforeApply != 0 {
				t.Fatalf("expected backup before sync state apply, got %d states", statesBeforeApply)
			}
			return nil
		},
	})

	if _, err := service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:        app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		SyncMode:           domain.SyncModeFullSnapshot,
		FullSnapshotReason: "terminal_restaurant_changed",
		CloudVersion:       77,
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-backup-before-apply",
			Type:     domain.CatalogItemDish,
			Name:     "Backup Tea",
			SKU:      "BACKUP-TEA",
			BaseUnit: "portion",
			Active:   true,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if backupCalls != 1 {
		t.Fatalf("expected one backup call, got %d", backupCalls)
	}
	if got := countRows(t, f, "cloud_master_sync_state"); got != 1 {
		t.Fatalf("expected one sync state after apply, got %d", got)
	}
}

func TestApplyMasterDataBackupErrorDoesNotWriteRowsOrState(t *testing.T) {
	f := newFixture(t)
	errBackup := errors.New("injected backup failure")
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 8100}, fixedClock{}, app.ServiceOptions{
		MasterDataBackupBeforeFullSnapshot: func(context.Context, app.MasterDataBackupRequest) error {
			return errBackup
		},
	})
	catalogBefore := countRows(t, f, "catalog_items")
	stateBefore := countRows(t, f, "cloud_master_sync_state")

	_, err := service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:        app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		SyncMode:           domain.SyncModeFullSnapshot,
		FullSnapshotReason: "terminal_restaurant_changed",
		CloudVersion:       78,
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-backup-fails",
			Type:     domain.CatalogItemDish,
			Name:     "Backup Fail Tea",
			SKU:      "BACKUP-FAIL-TEA",
			BaseUnit: "portion",
			Active:   true,
		}},
	})
	if !errors.Is(err, errBackup) {
		t.Fatalf("expected backup error, got %v", err)
	}
	if catalog := countRows(t, f, "catalog_items"); catalog != catalogBefore {
		t.Fatalf("expected backup error not to write catalog rows, before=%d after=%d", catalogBefore, catalog)
	}
	if states := countRows(t, f, "cloud_master_sync_state"); states != stateBefore {
		t.Fatalf("expected backup error not to write sync state, before=%d after=%d", stateBefore, states)
	}
}

func TestApplyMasterDataIncrementalDoesNotCreateBackup(t *testing.T) {
	f := newFixture(t)
	backupCalls := 0
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 8200}, fixedClock{}, app.ServiceOptions{
		MasterDataBackupBeforeFullSnapshot: func(context.Context, app.MasterDataBackupRequest) error {
			backupCalls++
			return nil
		},
	})

	if _, err := service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:  app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		SyncMode:     domain.SyncModeIncremental,
		CloudVersion: 79,
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-incremental-no-backup",
			Type:     domain.CatalogItemDish,
			Name:     "Incremental Tea",
			SKU:      "INCREMENTAL-TEA",
			BaseUnit: "portion",
			Active:   true,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if backupCalls != 0 {
		t.Fatalf("expected incremental ingest not to call backup, got %d calls", backupCalls)
	}
}

func TestApplyMasterDataEmptySyncModeDefaultsToIncremental(t *testing.T) {
	f := newFixture(t)
	backupCalls := 0
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 8250}, fixedClock{}, app.ServiceOptions{
		MasterDataBackupBeforeFullSnapshot: func(context.Context, app.MasterDataBackupRequest) error {
			backupCalls++
			return nil
		},
	})

	applied, err := service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:  app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		CloudVersion: 79,
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-default-incremental",
			Type:     domain.CatalogItemDish,
			Name:     "Default Incremental Tea",
			SKU:      "DEFAULT-INCREMENTAL-TEA",
			BaseUnit: "portion",
			Active:   true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if backupCalls != 0 {
		t.Fatalf("expected default incremental ingest not to call backup, got %d calls", backupCalls)
	}
	if len(applied.SyncStates) != 1 || applied.SyncStates[0].SyncMode != domain.SyncModeIncremental {
		t.Fatalf("expected default sync mode incremental, got %+v", applied.SyncStates)
	}
}

func TestApplyMasterDataFullSnapshotRequiresReason(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:  app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		SyncMode:     domain.SyncModeFullSnapshot,
		CloudVersion: 80,
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-full-snapshot-no-reason",
			Type:     domain.CatalogItemDish,
			Name:     "No Reason Tea",
			SKU:      "NO-REASON-TEA",
			BaseUnit: "portion",
			Active:   true,
		}},
	})
	if err == nil {
		t.Fatal("expected full_snapshot without reason to be rejected")
	}
}

func TestApplyMasterDataIncrementalRejectsFullSnapshotReason(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:        app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		SyncMode:           domain.SyncModeIncremental,
		FullSnapshotReason: "terminal_restaurant_changed",
		CloudVersion:       80,
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-incremental-with-reason",
			Type:     domain.CatalogItemDish,
			Name:     "Reason Tea",
			SKU:      "REASON-TEA",
			BaseUnit: "portion",
			Active:   true,
		}},
	})
	if err == nil {
		t.Fatal("expected incremental with full_snapshot_reason to be rejected")
	}
}

func TestApplyMasterDataInvalidPayloadDoesNotCreateBackup(t *testing.T) {
	f := newFixture(t)
	backupCalls := 0
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 8300}, fixedClock{}, app.ServiceOptions{
		MasterDataBackupBeforeFullSnapshot: func(context.Context, app.MasterDataBackupRequest) error {
			backupCalls++
			return nil
		},
	})

	_, err := service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:        app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		SyncMode:           domain.SyncModeFullSnapshot,
		FullSnapshotReason: "terminal_restaurant_changed",
		CloudVersion:       80,
		CatalogItems: []domain.CatalogItem{{
			ID:       "cloud-invalid-no-backup",
			Type:     domain.CatalogItemDish,
			Name:     "Invalid Tea",
			BaseUnit: "portion",
			Active:   true,
		}},
	})
	if err == nil {
		t.Fatal("expected invalid payload error")
	}
	if backupCalls != 0 {
		t.Fatalf("expected invalid payload not to call backup, got %d calls", backupCalls)
	}
	var rows int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM catalog_items WHERE id = 'cloud-invalid-no-backup'`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("expected invalid payload not to write catalog row, got %d", rows)
	}
}

func TestApplyMasterDataEmptyFullSnapshotDoesNotCreateBackup(t *testing.T) {
	f := newFixture(t)
	backupCalls := 0
	service := app.NewServiceWithOptions(f.repo, platformsqlite.NewTxManager(f.db), &testIDs{n: 8400}, fixedClock{}, app.ServiceOptions{
		MasterDataBackupBeforeFullSnapshot: func(context.Context, app.MasterDataBackupRequest) error {
			backupCalls++
			return nil
		},
	})
	stateBefore := countRows(t, f, "cloud_master_sync_state")

	_, err := service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:        app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		StreamName:         domain.MasterDataStreamCatalog,
		SyncMode:           domain.SyncModeFullSnapshot,
		FullSnapshotReason: "terminal_restaurant_changed",
		CloudVersion:       81,
	})
	if err == nil {
		t.Fatal("expected empty full_snapshot error")
	}
	if backupCalls != 0 {
		t.Fatalf("expected empty full_snapshot not to call backup, got %d calls", backupCalls)
	}
	if states := countRows(t, f, "cloud_master_sync_state"); states != stateBefore {
		t.Fatalf("expected empty full_snapshot not to write sync state, before=%d after=%d", stateBefore, states)
	}
}

func TestApplyMasterDataPricingPolicyAppliesReferenceRowsAndSyncState(t *testing.T) {
	f := newFixture(t)

	applied, err := f.service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:    app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		RestaurantID:   f.restaurant.ID,
		StreamName:     domain.MasterDataStreamPricing,
		SyncMode:       domain.SyncModeIncremental,
		CloudVersion:   91,
		CloudUpdatedAt: "2026-05-04T19:00:00Z",
		TaxProfiles: []domain.TaxProfile{{
			ID:        "tax-vat-10",
			Name:      "VAT 10",
			TaxExempt: false,
			Active:    true,
		}},
		TaxRules: []domain.TaxRule{{
			ID:              "tax-rule-vat-10",
			TaxProfileID:    "tax-vat-10",
			Name:            "VAT 10 exclusive",
			Kind:            domain.TaxRulePercentage,
			Mode:            domain.TaxModeExclusive,
			RateBasisPoints: 1000,
			Priority:        10,
			Active:          true,
		}},
		ServiceChargeRules: []domain.ServiceChargeRule{{
			ID:               "service-charge-10",
			RestaurantID:     f.restaurant.ID,
			Name:             "Service charge 10",
			Kind:             domain.SurchargeServiceCharge,
			AmountKind:       domain.AmountPercentage,
			ValueBasisPoints: 1000,
			Active:           true,
		}},
		PricingPolicies: []domain.PricingPolicy{{
			ID:               "discount-cloud-5",
			RestaurantID:     f.restaurant.ID,
			Kind:             domain.PricingPolicyDiscount,
			Name:             "Cloud discount 5",
			Scope:            domain.DiscountScopeOrder,
			AmountKind:       domain.AmountPercentage,
			ValueBasisPoints: 500,
			ApplicationIndex: 30,
			Active:           true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(applied.AppliedStreams) != 1 || applied.AppliedStreams[0] != domain.MasterDataStreamPricing {
		t.Fatalf("expected pricing_policy stream, got %+v", applied.AppliedStreams)
	}
	if applied.Counts[string(domain.MasterDataStreamPricing)] != 4 {
		t.Fatalf("expected four pricing policy rows, got %+v", applied.Counts)
	}
	for table, id := range map[string]string{
		"tax_profiles":         "tax-vat-10",
		"tax_rules":            "tax-rule-vat-10",
		"service_charge_rules": "service-charge-10",
		"pricing_policies":     "discount-cloud-5",
	} {
		var cloudVersion int64
		var lastSyncedAt string
		if err := f.db.QueryRowContext(f.ctx, `SELECT cloud_version,last_synced_at FROM `+table+` WHERE id = ?`, id).Scan(&cloudVersion, &lastSyncedAt); err != nil {
			t.Fatal(err)
		}
		if cloudVersion != 91 || lastSyncedAt == "" {
			t.Fatalf("expected sync metadata for %s.%s, got version=%d last_synced_at=%q", table, id, cloudVersion, lastSyncedAt)
		}
	}
	state, err := f.repo.GetMasterDataSyncState(f.ctx, f.device.ID, domain.MasterDataStreamPricing)
	if err != nil {
		t.Fatal(err)
	}
	if state.LastCloudVersion != 91 || state.Status != "applied" || state.SyncMode != domain.SyncModeIncremental {
		t.Fatalf("unexpected pricing sync state: %+v", state)
	}
}

func TestOrderLineSelectedModifiersAffectPricingAndPrecheckSnapshot(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	execTestSQL(t, f, `INSERT INTO modifier_groups(id,restaurant_id,name,required,min_count,max_count,active,cloud_version) VALUES ('group-spice',?,'Spice',0,0,2,1,1)`, f.restaurant.ID)
	execTestSQL(t, f, `INSERT INTO modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,active,cloud_version) VALUES ('option-hot',?,'group-spice','Hot',250,1,1)`, f.restaurant.ID)
	execTestSQL(t, f, `INSERT INTO menu_item_modifier_groups(menu_item_id,modifier_group_id,sort_order,cloud_version) VALUES (?, 'group-spice', 10, 1)`, f.menuItem.ID)

	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{
		CommandMeta: f.edgeMetaCommand("cmd-add-line-with-modifier"),
		OrderID:     order.ID,
		MenuItemID:  f.menuItem.ID,
		Quantity:    1,
		SelectedModifiers: []app.SelectedModifierCommand{{
			ModifierGroupID:  "group-spice",
			ModifierOptionID: "option-hot",
			Quantity:         1,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if line.TotalPrice != 1250 || len(line.Modifiers) != 1 || line.Modifiers[0].TotalPrice != 250 {
		t.Fatalf("expected modifier to be persisted on order line, got %+v", line)
	}
	calculation, err := f.service.CalculateOrderPricing(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if calculation.SubtotalMinor != 1250 || len(calculation.Lines) != 1 || len(calculation.Lines[0].Modifiers) != 1 {
		t.Fatalf("expected modifier in pricing calculation, got %+v", calculation)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-precheck-with-modifier"), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	var modifierRows int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM precheck_line_modifiers WHERE precheck_id = ? AND modifier_option_id = 'option-hot'`, precheck.ID).Scan(&modifierRows); err != nil {
		t.Fatal(err)
	}
	if modifierRows != 1 || !strings.Contains(string(precheck.Snapshot), `"modifiers"`) {
		t.Fatalf("expected selected modifier in precheck snapshot, rows=%d snapshot=%s", modifierRows, precheck.Snapshot)
	}
}

func TestOrderLineQuantityChangeKeepsSelectedModifierTotal(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	execTestSQL(t, f, `INSERT INTO modifier_groups(id,restaurant_id,name,required,min_count,max_count,active,cloud_version) VALUES ('group-spice',?,'Spice',0,0,3,1,1)`, f.restaurant.ID)
	execTestSQL(t, f, `INSERT INTO modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,active,cloud_version) VALUES ('option-hot',?,'group-spice','Hot',250,1,1)`, f.restaurant.ID)
	execTestSQL(t, f, `INSERT INTO menu_item_modifier_groups(menu_item_id,modifier_group_id,sort_order,cloud_version) VALUES (?, 'group-spice', 10, 1)`, f.menuItem.ID)

	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{
		CommandMeta: f.edgeMetaCommand("cmd-add-line-with-modifier-qty"),
		OrderID:     order.ID,
		MenuItemID:  f.menuItem.ID,
		Quantity:    1,
		SelectedModifiers: []app.SelectedModifierCommand{{
			ModifierGroupID:  "group-spice",
			ModifierOptionID: "option-hot",
			Quantity:         2,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := f.service.ChangeOrderLineQuantity(f.ctx, app.ChangeOrderLineQuantityCommand{
		CommandMeta: f.edgeMetaCommand("cmd-change-line-with-modifier-qty"),
		OrderID:     order.ID,
		LineID:      line.ID,
		Quantity:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed.TotalPrice != 3500 {
		t.Fatalf("expected base quantity total plus fixed selected modifier total, got %+v", changed)
	}
	calculation, err := f.service.CalculateOrderPricing(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if calculation.SubtotalMinor != 3500 || len(calculation.Lines) != 1 || len(calculation.Lines[0].Modifiers) != 1 || calculation.Lines[0].Modifiers[0].TotalMinor != 500 {
		t.Fatalf("expected recalculated pricing with preserved modifier total, got %+v", calculation)
	}
}

func TestCheckSnapshotIncludesSelectedModifiersFromPrecheckSnapshot(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	execTestSQL(t, f, `INSERT INTO modifier_groups(id,restaurant_id,name,required,min_count,max_count,active,cloud_version) VALUES ('group-spice',?,'Spice',0,0,2,1,1)`, f.restaurant.ID)
	execTestSQL(t, f, `INSERT INTO modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,active,cloud_version) VALUES ('option-hot',?,'group-spice','Hot',250,1,1)`, f.restaurant.ID)
	execTestSQL(t, f, `INSERT INTO menu_item_modifier_groups(menu_item_id,modifier_group_id,sort_order,cloud_version) VALUES (?, 'group-spice', 10, 1)`, f.menuItem.ID)

	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{
		CommandMeta: f.edgeMetaCommand("cmd-add-line-check-modifier"),
		OrderID:     order.ID,
		MenuItemID:  f.menuItem.ID,
		Quantity:    1,
		SelectedModifiers: []app.SelectedModifierCommand{{
			ModifierGroupID:  "group-spice",
			ModifierOptionID: "option-hot",
			Quantity:         1,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-precheck-check-modifier"), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-pay-check-modifier"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(check.Snapshot), `"precheck_snapshot"`) || !strings.Contains(string(check.Snapshot), `"modifiers"`) || check.Total != 1250 {
		t.Fatalf("expected check snapshot and totals to preserve selected modifiers, check=%+v snapshot=%s", check, check.Snapshot)
	}
}

func TestServiceCatalogItemSellsThroughOrderPricingPrecheckAndCheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	f.openCashSession(t)
	serviceCatalog, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{
		CommandMeta: seedMeta(f.device.ID),
		Type:        domain.CatalogItemService,
		Name:        "Delivery",
		SKU:         "DELIVERY",
		BaseUnit:    "service",
	})
	if err != nil {
		t.Fatal(err)
	}
	serviceMenuItem, err := f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{
		CommandMeta:   seedMeta(f.device.ID),
		CatalogItemID: serviceCatalog.ID,
		Name:          "Delivery",
		Price:         300,
		Currency:      "RUB",
	})
	if err != nil {
		t.Fatal(err)
	}
	menuItems, err := f.service.ListMenuItemsAsOperator(f.ctx, f.edgeMeta())
	if err != nil {
		t.Fatal(err)
	}
	var foundService bool
	for _, item := range menuItems {
		if item.ID == serviceMenuItem.ID && item.ItemType == string(domain.CatalogItemService) {
			foundService = true
		}
	}
	if !foundService {
		t.Fatalf("expected service menu item to expose item_type=service, got %+v", menuItems)
	}

	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-add-service-line"), OrderID: order.ID, MenuItemID: serviceMenuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	if line.CatalogItemID != serviceCatalog.ID || line.TotalPrice != 600 {
		t.Fatalf("expected service item line to sell as normal catalog item, got %+v", line)
	}
	calculation, err := f.service.CalculateOrderPricing(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if calculation.SubtotalMinor != 600 || calculation.GrandTotalMinor != 600 {
		t.Fatalf("expected service item pricing total, got %+v", calculation)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-service-precheck"), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if precheck.Total != 600 {
		t.Fatalf("expected service item precheck total, got %+v", precheck)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-service-payment"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if check.Total != 600 || !strings.Contains(string(check.Snapshot), `"Delivery"`) {
		t.Fatalf("expected service item check snapshot and total, check=%+v snapshot=%s", check, check.Snapshot)
	}
}

func TestApplyMasterDataRejectsUnsupportedStreamWithoutPartialWrite(t *testing.T) {
	f := newFixture(t)
	stateBefore := countRows(t, f, "cloud_master_sync_state")
	taxBefore := countRows(t, f, "tax_profiles")

	_, err := f.service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:  app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		StreamName:   domain.MasterDataStream("unknown_payload"),
		SyncMode:     domain.SyncModeIncremental,
		CloudVersion: 92,
		TaxProfiles: []domain.TaxProfile{{
			ID:     "tax-should-not-write",
			Name:   "Should not write",
			Active: true,
		}},
	})
	if err == nil {
		t.Fatal("expected unsupported stream to be rejected")
	}
	if states := countRows(t, f, "cloud_master_sync_state"); states != stateBefore {
		t.Fatalf("expected unsupported stream not to write sync state, before=%d after=%d", stateBefore, states)
	}
	if rows := countRows(t, f, "tax_profiles"); rows != taxBefore {
		t.Fatalf("expected unsupported stream not to write tax rows, before=%d after=%d", taxBefore, rows)
	}
}

func TestListLocalEventsThroughServiceSupportsLimitAndFilter(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
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

func TestAddOrderLineBlockedByDirectStopList(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	insertStopList(t, f, "stop-soup", f.menuItem.CatalogItemID, nil, true)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-stop-create-order"), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-stop-add-soup"), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if !errors.Is(err, domain.ErrSaleUnavailable) || !strings.Contains(err.Error(), "active stop-list") {
		t.Fatalf("expected stop-list sale conflict, got %v", err)
	}

	catalog, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{CommandMeta: seedMeta(f.device.ID), Type: domain.CatalogItemDish, Name: "Salad", SKU: "SALAD", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	menuItem, err := f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{CommandMeta: seedMeta(f.device.ID), CatalogItemID: catalog.ID, Name: "Salad", Price: 1200, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-stop-add-salad"), OrderID: order.ID, MenuItemID: menuItem.ID, Quantity: 1}); err != nil {
		t.Fatalf("expected unblocked item to be added, got %v", err)
	}
}

func TestChangeOrderLineQuantityBlocksOnlyIncreaseForStopList(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-qty-stop-create-order"), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-qty-stop-add"), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	insertStopList(t, f, "stop-soup-qty", f.menuItem.CatalogItemID, floatPtr(0), true)

	if _, err := f.service.ChangeOrderLineQuantity(f.ctx, app.ChangeOrderLineQuantityCommand{CommandMeta: f.edgeMetaCommand("cmd-qty-stop-decrease"), OrderID: order.ID, LineID: line.ID, Quantity: 1}); err != nil {
		t.Fatalf("expected quantity decrease to remain available, got %v", err)
	}
	_, err = f.service.ChangeOrderLineQuantity(f.ctx, app.ChangeOrderLineQuantityCommand{CommandMeta: f.edgeMetaCommand("cmd-qty-stop-increase"), OrderID: order.ID, LineID: line.ID, Quantity: 3})
	if !errors.Is(err, domain.ErrSaleUnavailable) {
		t.Fatalf("expected stop-list conflict on increase, got %v", err)
	}
}

func TestAddOrderLineBlockedByRecipeComponentStopList(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	component, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{CommandMeta: seedMeta(f.device.ID), Type: domain.CatalogItemGood, Name: "Potato", SKU: "POTATO", BaseUnit: "g"})
	if err != nil {
		t.Fatal(err)
	}
	insertRecipe(t, f, "recipe-soup", f.menuItem.CatalogItemID, component.ID)
	insertStopList(t, f, "stop-potato", component.ID, nil, true)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-recipe-stop-create-order"), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-recipe-stop-add"), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if !errors.Is(err, domain.ErrSaleUnavailable) || !strings.Contains(err.Error(), "recipe component") {
		t.Fatalf("expected recipe component stop-list conflict, got %v", err)
	}
}

func TestApplyMasterDataRecipesAndInventoryReferenceStreams(t *testing.T) {
	f := newFixture(t)
	component, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{CommandMeta: seedMeta(f.device.ID), Type: domain.CatalogItemGood, Name: "Carrot", SKU: "CARROT", BaseUnit: "g"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:    app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		RestaurantID:   f.restaurant.ID,
		StreamName:     domain.MasterDataStreamRecipes,
		SyncMode:       domain.SyncModeIncremental,
		CloudVersion:   101,
		CloudUpdatedAt: appshared.DBTime(fixedClock{}.Now()),
		RecipeVersions: []domain.RecipeVersion{{
			ID:                "recipe-cloud-soup",
			DishCatalogItemID: f.menuItem.CatalogItemID,
			Version:           1,
			Name:              "Soup recipe",
			Status:            domain.RecipeVersionActive,
			YieldQuantity:     1,
			YieldUnit:         "portion",
			Active:            true,
		}},
		RecipeLines: []domain.RecipeLine{{
			ID:              "recipe-cloud-line-carrot",
			RecipeVersionID: "recipe-cloud-soup",
			CatalogItemID:   component.ID,
			Quantity:        100,
			Unit:            "g",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	recipe, err := f.repo.GetActiveRecipeVersionByCatalogItem(f.ctx, f.menuItem.CatalogItemID)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := f.repo.ListRecipeLines(f.ctx, recipe.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recipe.CloudVersion != 101 || len(lines) != 1 || lines[0].CatalogItemID != component.ID {
		t.Fatalf("expected recipe stream upsert, recipe=%+v lines=%+v", recipe, lines)
	}

	_, err = f.service.ApplyMasterData(f.ctx, app.ApplyMasterDataCommand{
		CommandMeta:    app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, Origin: app.OriginCloudSync},
		RestaurantID:   f.restaurant.ID,
		StreamName:     domain.MasterDataStreamInventory,
		SyncMode:       domain.SyncModeIncremental,
		CloudVersion:   102,
		CloudUpdatedAt: appshared.DBTime(fixedClock{}.Now()),
		StopListEntries: []domain.StopListEntry{{
			ID:            "stop-cloud-carrot",
			RestaurantID:  f.restaurant.ID,
			CatalogItemID: component.ID,
			Source:        "cloud",
			Active:        true,
			UpdatedAt:     fixedClock{}.Now(),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	stop, err := f.repo.GetBlockingStopListEntry(f.ctx, f.restaurant.ID, component.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stop.ID != "stop-cloud-carrot" || stop.CloudVersion == nil || *stop.CloudVersion != 102 {
		t.Fatalf("expected blocking stop-list entry from inventory_reference stream, got %+v", stop)
	}
}

func TestKeyWritesCreateLocalEventsAndMatchingOutboxEnvelopes(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
	cashSession := f.openCashSession(t)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-key-write-full-payment"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	managerMeta := f.managerEdgeMetaCommand(t, "cmd-key-write-close-cash")
	if _, err := f.service.CloseCashSession(f.ctx, app.CloseCashSessionCommand{CommandMeta: managerMeta, ID: cashSession.ID, ClosedByEmployeeID: f.manager.ID, ClosingCashAmount: 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{CommandMeta: f.edgeMeta(), ID: shift.ID, ClosedByEmployeeID: f.employee.ID, ClosingCashAmount: 0}); err != nil {
		t.Fatal(err)
	}

	eventTypes := []string{"ShiftOpened", "ShiftClosed", "OrderCreated", "OrderLineAdded", "PrecheckIssued", "OrderClosed", "CheckCreated", "PaymentCaptured"}
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

func seedSpiceModifierFixture(t *testing.T, f *fixture, required bool, minCount, maxCount int, optionActive bool, priceMinor int64) {
	t.Helper()
	requiredValue := 0
	if required {
		requiredValue = 1
	}
	activeValue := 0
	if optionActive {
		activeValue = 1
	}
	execTestSQL(t, f, `INSERT INTO modifier_groups(id,restaurant_id,name,required,min_count,max_count,active,cloud_version) VALUES ('group-spice',?,'Spice',?,?,?,?,1)`, f.restaurant.ID, requiredValue, minCount, maxCount, 1)
	execTestSQL(t, f, `INSERT INTO modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,active,cloud_version) VALUES ('option-hot',?,'group-spice','Hot',?,?,1)`, f.restaurant.ID, priceMinor, activeValue)
	execTestSQL(t, f, `INSERT INTO menu_item_modifier_groups(menu_item_id,modifier_group_id,sort_order,cloud_version) VALUES (?, 'group-spice', 10, 1)`, f.menuItem.ID)
}

func TestAddOrderLineModifierAuthoritativeValidation(t *testing.T) {
	tests := []struct {
		name      string
		seed      func(t *testing.T, f *fixture)
		selected  []app.SelectedModifierCommand
		wantError string
	}{
		{
			name: "required group missing",
			seed: func(t *testing.T, f *fixture) {
				seedSpiceModifierFixture(t, f, true, 1, 2, true, 250)
			},
			wantError: "required modifier group is missing",
		},
		{
			name: "max count exceeded",
			seed: func(t *testing.T, f *fixture) {
				seedSpiceModifierFixture(t, f, false, 0, 1, true, 250)
			},
			selected:  []app.SelectedModifierCommand{{ModifierGroupID: "group-spice", ModifierOptionID: "option-hot", Quantity: 2}},
			wantError: "max_count",
		},
		{
			name: "inactive option rejected",
			seed: func(t *testing.T, f *fixture) {
				seedSpiceModifierFixture(t, f, false, 0, 2, false, 250)
			},
			selected:  []app.SelectedModifierCommand{{ModifierGroupID: "group-spice", ModifierOptionID: "option-hot", Quantity: 1}},
			wantError: "option is not active",
		},
		{
			name: "option must belong to linked group",
			seed: func(t *testing.T, f *fixture) {
				seedSpiceModifierFixture(t, f, false, 0, 2, true, 250)
				execTestSQL(t, f, `INSERT INTO modifier_groups(id,restaurant_id,name,required,min_count,max_count,active,cloud_version) VALUES ('group-sauce',?,'Sauce',0,0,2,1,1)`, f.restaurant.ID)
				execTestSQL(t, f, `INSERT INTO modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,active,cloud_version) VALUES ('option-bbq',?,'group-sauce','BBQ',100,1,1)`, f.restaurant.ID)
			},
			selected:  []app.SelectedModifierCommand{{ModifierGroupID: "group-spice", ModifierOptionID: "option-bbq", Quantity: 1}},
			wantError: "option is not active in group",
		},
		{
			name: "group must be linked to menu item",
			seed: func(t *testing.T, f *fixture) {
				execTestSQL(t, f, `INSERT INTO modifier_groups(id,restaurant_id,name,required,min_count,max_count,active,cloud_version) VALUES ('group-spice',?,'Spice',0,0,2,1,1)`, f.restaurant.ID)
				execTestSQL(t, f, `INSERT INTO modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,active,cloud_version) VALUES ('option-hot',?,'group-spice','Hot',250,1,1)`, f.restaurant.ID)
			},
			selected:  []app.SelectedModifierCommand{{ModifierGroupID: "group-spice", ModifierOptionID: "option-hot", Quantity: 1}},
			wantError: "menu item has no modifiers",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.openShift(t)
			tt.seed(t, f)
			order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-create-" + strings.ReplaceAll(tt.name, " ", "-")), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-add-" + strings.ReplaceAll(tt.name, " ", "-")), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1, SelectedModifiers: tt.selected})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
			}
		})
	}
}

func TestUpdateOrderLineModifiersRepricesPersistsAndWritesOutbox(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	seedSpiceModifierFixture(t, f, false, 0, 3, true, 250)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-create-update-modifiers"), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-add-update-modifiers"), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := f.service.UpdateOrderLineModifiers(f.ctx, app.UpdateOrderLineModifiersCommand{
		CommandMeta: f.edgeMetaCommand("cmd-update-modifiers"),
		OrderID:     order.ID,
		LineID:      line.ID,
		SelectedModifiers: []app.SelectedModifierCommand{{
			ModifierGroupID:  "group-spice",
			ModifierOptionID: "option-hot",
			Quantity:         2,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.TotalPrice != 2500 || len(updated.Modifiers) != 1 || updated.Modifiers[0].TotalPrice != 500 {
		t.Fatalf("expected repriced selected modifiers, got %+v", updated)
	}
	var modifierRows, outboxRows int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM order_line_modifiers WHERE order_line_id = ? AND modifier_option_id = 'option-hot' AND total_price = 500`, line.ID).Scan(&modifierRows); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM pos_sync_outbox WHERE command_type = 'OrderLineModifiersUpdated' AND aggregate_id = ?`, order.ID).Scan(&outboxRows); err != nil {
		t.Fatal(err)
	}
	if modifierRows != 1 || outboxRows != 1 {
		t.Fatalf("expected persisted modifier and outbox event, modifierRows=%d outboxRows=%d", modifierRows, outboxRows)
	}
	calculation, err := f.service.CalculateOrderPricing(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if calculation.SubtotalMinor != 2500 || len(calculation.Lines[0].Modifiers) != 1 {
		t.Fatalf("expected pricing with updated modifiers, got %+v", calculation)
	}
}

func TestUpdateOrderLineModifiersBlockedByActivePrecheck(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
	seedSpiceModifierFixture(t, f, false, 0, 3, true, 250)
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMetaCommand("cmd-create-locked-modifiers"), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMetaCommand("cmd-add-locked-modifiers"), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMetaCommand("cmd-precheck-locked-modifiers"), OrderID: order.ID}); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.UpdateOrderLineModifiers(f.ctx, app.UpdateOrderLineModifiersCommand{CommandMeta: f.edgeMetaCommand("cmd-update-locked-modifiers"), OrderID: order.ID, LineID: line.ID})
	if err == nil || !strings.Contains(err.Error(), "cannot change non-open order") {
		t.Fatalf("expected locked order conflict, got %v", err)
	}
}
