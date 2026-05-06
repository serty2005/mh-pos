package app_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pos-backend/internal/platform/clock"
	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/pos/app"
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

type fixture struct {
	ctx        context.Context
	db         *sql.DB
	repo       *possqlite.Repository
	service    *app.Service
	restaurant *domain.Restaurant
	device     *domain.Device
	employee   *domain.Employee
	manager    *domain.Employee
	session    *domain.AuthSession
	hall       *domain.Hall
	table      *domain.Table
	menuItem   *domain.MenuItem
	clientID   string
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

func (f *fixture) cancelPrecheckCommand(commandID, precheckID string) app.CancelPrecheckCommand {
	return app.CancelPrecheckCommand{
		CommandMeta:        f.edgeMetaCommand(commandID),
		PrecheckID:         precheckID,
		ManagerEmployeeID:  f.manager.ID,
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
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{CommandMeta: seedMeta(bootstrapDeviceID), Name: "cashier", PermissionsJSON: `{"pos":true}`})
	if err != nil {
		t.Fatal(err)
	}
	managerRole, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{CommandMeta: seedMeta(bootstrapDeviceID), Name: "manager", PermissionsJSON: `{"precheck.cancel":true}`})
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

func countRows(t *testing.T, f *fixture, table string) int {
	t.Helper()
	switch table {
	case "orders", "order_lines", "prechecks", "checks", "payments", "payment_attempts", "cash_sessions", "cash_drawer_events", "pos_sync_outbox", "local_event_log", "manager_override_audit", "roles", "catalog_items", "menu_items", "auth_sessions", "halls", "tables":
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
		CommandMeta: app.CommandMeta{CommandID: "cmd-logout-session", NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, Origin: app.OriginEdgeDevice},
		SessionID:   login.Session.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if logout.Status != domain.AuthSessionRevoked || logout.RevokedAt == nil {
		t.Fatalf("expected revoked session, got %+v", logout)
	}
	current, err := f.service.GetSession(f.ctx, login.Session.ID, f.device.ID, f.clientID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Session.Status != domain.AuthSessionRevoked {
		t.Fatalf("expected session restore to show revoked status, got %+v", current.Session)
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
		CommandMeta:  app.CommandMeta{CommandID: "cmd-create-employee-no-pin-leak", DeviceID: f.device.ID, Origin: app.OriginEdgeDevice},
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

func TestCreateAndArchiveHallAndTableUseOutbox(t *testing.T) {
	f := newFixture(t)
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	eventsBefore := countRows(t, f, "local_event_log")

	hall, err := f.service.CreateHall(f.ctx, app.CreateHallCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, Name: "Terrace"})
	if err != nil {
		t.Fatal(err)
	}
	table, err := f.service.CreateTable(f.ctx, app.CreateTableCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, HallID: hall.ID, Name: "T1", Seats: 4})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.ArchiveTable(f.ctx, app.ArchiveTableCommand{CommandMeta: f.edgeMeta(), ID: table.ID}); err != nil {
		t.Fatal(err)
	}
	if err := f.service.ArchiveHall(f.ctx, app.ArchiveHallCommand{CommandMeta: f.edgeMeta(), ID: hall.ID}); err != nil {
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
}

func TestCannotCreateTableInArchivedHall(t *testing.T) {
	f := newFixture(t)
	hall, err := f.service.CreateHall(f.ctx, app.CreateHallCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, Name: "Closed room"})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.service.ArchiveHall(f.ctx, app.ArchiveHallCommand{CommandMeta: f.edgeMeta(), ID: hall.ID}); err != nil {
		t.Fatal(err)
	}

	_, err = f.service.CreateTable(f.ctx, app.CreateTableCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, HallID: hall.ID, Name: "C1", Seats: 2})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestValidDeviceIDCreatesBusinessAndOutboxRowsWithDefaultOrigin(t *testing.T) {
	f := newFixture(t)
	rolesBefore := countRows(t, f, "roles")
	outboxBefore := countRows(t, f, "pos_sync_outbox")
	commandID := "cmd-role-with-device"

	_, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     app.CommandMeta{CommandID: commandID, DeviceID: f.device.ID},
		Name:            "supervisor",
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
	if _, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1"}); err != nil {
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
		CommandMeta:        f.edgeMetaCommand("cmd-cancel-missing-precheck"),
		PrecheckID:         "missing-precheck",
		ManagerEmployeeID:  f.manager.ID,
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

func TestCancelPrecheckRejectsEmployeeWithoutPermission(t *testing.T) {
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
	cmd.ManagerEmployeeID = f.employee.ID
	cmd.ManagerPIN = "1111"
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
		CommandMeta:        f.edgeMetaCommand("cmd-cancel-duplicate"),
		PrecheckID:         precheck.ID,
		ManagerEmployeeID:  f.manager.ID,
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
	other, err := f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{CommandMeta: f.edgeMeta(), Name: "Other", Timezone: "Europe/Moscow", Currency: "RUB"})
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

func TestCapturePaymentRollbackRemovesAttemptPaymentOutboxAndLocalEvent(t *testing.T) {
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

func TestFullPaymentRollsBackFinalCheckWhenCheckCreatedOutboxFails(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
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

func TestPaymentForCancelledPrecheckRejected(t *testing.T) {
	f := newFixture(t)
	f.openShift(t)
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
	if _, err := f.db.ExecContext(f.ctx, `UPDATE shifts SET status = 'closed', closed_at = ?, updated_at = ? WHERE id = ?`, appshared.DBTime(fixedClock{}.Now()), appshared.DBTime(fixedClock{}.Now()), order.ShiftID); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMetaCommand("cmd-payment-no-active-shift"), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected active shift conflict, got %v", err)
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

func TestKeyWritesCreateLocalEventsAndMatchingOutboxEnvelopes(t *testing.T) {
	f := newFixture(t)
	shift := f.openShift(t)
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
