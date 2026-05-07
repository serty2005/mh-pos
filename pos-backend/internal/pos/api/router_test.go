package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platformsqlite "pos-backend/internal/platform/sqlite"
	"pos-backend/internal/pos/api"
	"pos-backend/internal/pos/app"
	appshared "pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	possqlite "pos-backend/internal/pos/infra/sqlite"
)

type apiTestIDs struct {
	n int
}

func (g *apiTestIDs) NewID() string {
	g.n++
	return fmt.Sprintf("api-id-%03d", g.n)
}

type apiFixedClock struct{}

func (apiFixedClock) Now() time.Time {
	return time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
}

type apiFixture struct {
	ctx        context.Context
	db         *sql.DB
	repo       *possqlite.Repository
	service    *app.Service
	router     http.Handler
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

func newAPIFixture(t *testing.T) *apiFixture {
	t.Helper()
	ctx := context.Background()
	db, err := platformsqlite.Open(filepath.Join(t.TempDir(), "pos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := platformsqlite.MigrateDir(ctx, db, filepath.Join("..", "..", "..", "migrations", "sqlite")); err != nil {
		t.Fatal(err)
	}
	repo := possqlite.NewRepository(db)
	service := app.NewService(repo, platformsqlite.NewTxManager(db), &apiTestIDs{}, apiFixedClock{})
	f := &apiFixture{ctx: ctx, db: db, repo: repo, service: service, router: api.NewRouter(service), clientID: "api-client-1"}
	f.seed(t)
	return f
}

func (f *apiFixture) seed(t *testing.T) {
	t.Helper()
	var err error
	f.restaurant, err = f.service.CreateRestaurant(f.ctx, app.CreateRestaurantCommand{CommandMeta: apiSeedMeta("bootstrap-device"), Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{CommandMeta: apiSeedMeta("bootstrap-device"), Name: "cashier", PermissionsJSON: `{"pos":true}`})
	if err != nil {
		t.Fatal(err)
	}
	managerRole, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{CommandMeta: apiSeedMeta("bootstrap-device"), Name: "manager", PermissionsJSON: `{"precheck.cancel":true}`})
	if err != nil {
		t.Fatal(err)
	}
	f.device, err = f.service.RegisterDevice(f.ctx, app.RegisterDeviceCommand{CommandMeta: apiSeedMeta("bootstrap-device"), RestaurantID: f.restaurant.ID, DeviceCode: "POS-1", Name: "Main", Type: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.PairEdgeNode(f.ctx, app.PairEdgeNodeCommand{PairingCode: "MHPOS:" + f.restaurant.ID + ":" + f.device.ID}); err != nil {
		t.Fatal(err)
	}
	cashierPINHash, err := appshared.HashPIN("1111", []byte("api-cashier-salt"))
	if err != nil {
		t.Fatal(err)
	}
	f.employee, err = f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{CommandMeta: apiSeedMeta(f.device.ID), RestaurantID: f.restaurant.ID, RoleID: role.ID, Name: "Anna", PINHash: cashierPINHash})
	if err != nil {
		t.Fatal(err)
	}
	managerPINHash, err := appshared.HashPIN("2468", []byte("api-manager-salt"))
	if err != nil {
		t.Fatal(err)
	}
	f.manager, err = f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{CommandMeta: apiSeedMeta(f.device.ID), RestaurantID: f.restaurant.ID, RoleID: managerRole.ID, Name: "Mira", PINHash: managerPINHash})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{CommandMeta: app.CommandMeta{CommandID: "cmd-api-seed-login", NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, Origin: app.OriginEdgeDevice}, PIN: "1111"})
	if err != nil {
		t.Fatal(err)
	}
	f.session = &login.Session
	f.hall, err = f.service.CreateHall(f.ctx, app.CreateHallCommand{CommandMeta: apiSeedMeta(f.device.ID), RestaurantID: f.restaurant.ID, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	f.table, err = f.service.CreateTable(f.ctx, app.CreateTableCommand{CommandMeta: apiSeedMeta(f.device.ID), RestaurantID: f.restaurant.ID, HallID: f.hall.ID, Name: "A1", Seats: 2})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{CommandMeta: apiSeedMeta(f.device.ID), Type: domain.CatalogItemDish, Name: "Soup", SKU: "SOUP", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	f.menuItem, err = f.service.CreateMenuItem(f.ctx, app.CreateMenuItemCommand{CommandMeta: apiSeedMeta(f.device.ID), CatalogItemID: catalog.ID, Name: "Soup", Price: 1000, Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
}

func apiSeedMeta(deviceID string) app.CommandMeta {
	return app.CommandMeta{DeviceID: deviceID, Origin: app.OriginSystemSeed}
}

func (f *apiFixture) edgeMeta() app.CommandMeta {
	return app.CommandMeta{NodeDeviceID: f.device.ID, DeviceID: f.device.ID, ClientDeviceID: f.clientID, ActorEmployeeID: f.employee.ID, SessionID: f.session.ID, Origin: app.OriginEdgeDevice}
}

func (f *apiFixture) createOrderWithLine(t *testing.T) *domain.Order {
	t.Helper()
	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, TableName: "A1", GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	return order
}

func (f *apiFixture) postJSON(t *testing.T, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	f.setOperatorHeaders(req)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

func (f *apiFixture) patchJSON(t *testing.T, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	f.setOperatorHeaders(req)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

func (f *apiFixture) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	f.setOperatorHeaders(req)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

func (f *apiFixture) setOperatorHeaders(req *http.Request) {
	if f.device == nil || f.session == nil {
		return
	}
	req.Header.Set("X-Node-Device-ID", f.device.ID)
	req.Header.Set("X-Client-Device-ID", f.clientID)
	req.Header.Set("X-Actor-Employee-ID", f.employee.ID)
	req.Header.Set("X-Session-ID", f.session.ID)
}

func decodeAPIResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, rr.Body.String())
	}
	return out
}

func countAPIRows(t *testing.T, f *apiFixture, table string) int {
	t.Helper()
	switch table {
	case "prechecks", "checks", "payments", "payment_attempts", "pos_sync_outbox", "local_event_log", "manager_override_audit", "auth_sessions", "halls", "tables":
	default:
		t.Fatalf("unexpected table %q", table)
	}
	var n int
	if err := f.db.QueryRowContext(f.ctx, fmt.Sprintf("SELECT COUNT(1) FROM %s", table)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPinLoginAndSessionAPI(t *testing.T) {
	f := newAPIFixture(t)
	rr := f.postJSON(t, "/api/v1/auth/pin-login", `{"command_id":"cmd-api-pin-login","node_device_id":"`+f.device.ID+`","pin":"1111"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	result := decodeAPIResponse[domain.PinLoginResult](t, rr)
	if result.Session.EmployeeID != f.employee.ID || result.Actor.EmployeeID != f.employee.ID {
		t.Fatalf("unexpected login result: %+v", result)
	}
	if strings.Contains(rr.Body.String(), "1111") {
		t.Fatal("expected PIN not to be returned in login response")
	}
	if strings.Contains(rr.Body.String(), "pin_hash") {
		t.Fatal("expected pin_hash not to be returned in login response")
	}

	current := f.get(t, "/api/v1/auth/session?node_device_id="+f.device.ID+"&session_id="+result.Session.ID)
	if current.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", current.Code, current.Body.String())
	}
	got := decodeAPIResponse[domain.PinLoginResult](t, current)
	if got.Session.ID != result.Session.ID || got.Actor.EmployeeID != f.employee.ID {
		t.Fatalf("unexpected current session: %+v", got)
	}
}

func TestFloorAndOrderLineEditingAPI(t *testing.T) {
	f := newAPIFixture(t)
	hallResp := f.postJSON(t, "/api/v1/halls", `{"node_device_id":"`+f.device.ID+`","restaurant_id":"`+f.restaurant.ID+`","name":"Terrace"}`)
	if hallResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", hallResp.Code, hallResp.Body.String())
	}
	hall := decodeAPIResponse[domain.Hall](t, hallResp)
	tableResp := f.postJSON(t, "/api/v1/tables", `{"node_device_id":"`+f.device.ID+`","restaurant_id":"`+f.restaurant.ID+`","hall_id":"`+hall.ID+`","name":"T1","seats":4}`)
	if tableResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", tableResp.Code, tableResp.Body.String())
	}
	table := decodeAPIResponse[domain.Table](t, tableResp)
	listTables := f.get(t, "/api/v1/tables?restaurant_id="+f.restaurant.ID+"&hall_id="+hall.ID)
	if listTables.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listTables.Code, listTables.Body.String())
	}
	tables := decodeAPIResponse[[]domain.Table](t, listTables)
	if len(tables) != 1 || tables[0].ID != table.ID {
		t.Fatalf("unexpected table list: %+v", tables)
	}

	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, OpenedByEmployeeID: f.employee.ID, OpeningCashAmount: 0}); err != nil {
		t.Fatal(err)
	}
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: table.ID, GuestCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	line, err := f.service.AddOrderLine(f.ctx, app.AddOrderLineCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID, MenuItemID: f.menuItem.ID, Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	change := f.patchJSON(t, "/api/v1/orders/"+order.ID+"/lines/"+line.ID, `{"node_device_id":"`+f.device.ID+`","quantity":3}`)
	if change.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", change.Code, change.Body.String())
	}
	changed := decodeAPIResponse[domain.OrderLine](t, change)
	if changed.Quantity != 3 || changed.TotalPrice != 3000 {
		t.Fatalf("unexpected changed line: %+v", changed)
	}
	voidedResp := f.postJSON(t, "/api/v1/orders/"+order.ID+"/lines/"+line.ID+"/void", `{"node_device_id":"`+f.device.ID+`","reason":"mistake"}`)
	if voidedResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", voidedResp.Code, voidedResp.Body.String())
	}
	voided := decodeAPIResponse[domain.OrderLine](t, voidedResp)
	if voided.Status != domain.OrderLineVoided {
		t.Fatalf("expected voided line, got %+v", voided)
	}
}

func apiOutboxIDs(t *testing.T, f *apiFixture, n int) []string {
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

func TestSyncStatusAPI(t *testing.T) {
	f := newAPIFixture(t)
	ids := apiOutboxIDs(t, f, 2)
	now := appshared.DBTime(apiFixedClock{}.Now())
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = 1, last_error = 'temporary', updated_at = ? WHERE id = ?`, now, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'processing', locked_at = ?, locked_by = 'api-worker', updated_at = ? WHERE id = ?`, now, now, ids[1]); err != nil {
		t.Fatal(err)
	}

	rr := f.get(t, "/api/v1/sync/status")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	status := decodeAPIResponse[domain.SyncStatus](t, rr)
	if status.Total != countAPIRows(t, f, "pos_sync_outbox") || status.Failed != 1 || status.Processing != 1 {
		t.Fatalf("unexpected sync status: %+v", status)
	}
}

func TestRetryFailedAPIResetsFailedAndSuspendedButNotSent(t *testing.T) {
	f := newAPIFixture(t)
	ids := apiOutboxIDs(t, f, 3)
	now := appshared.DBTime(apiFixedClock{}.Now())
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = 1, last_error = 'temporary', updated_at = ? WHERE id = ?`, now, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'suspended', attempts = 4, last_error = 'threshold', updated_at = ? WHERE id = ?`, now, ids[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'sent', sent_at = ?, updated_at = ? WHERE id = ?`, now, now, ids[2]); err != nil {
		t.Fatal(err)
	}

	rr := f.postJSON(t, "/api/v1/sync/retry-failed", `{}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	got := decodeAPIResponse[map[string]int](t, rr)
	if got["retried"] != 2 {
		t.Fatalf("expected retried=2, got %+v", got)
	}
	var failedStatus, suspendedStatus, sentStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM pos_sync_outbox WHERE id = ?`, ids[0]).Scan(&failedStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM pos_sync_outbox WHERE id = ?`, ids[1]).Scan(&suspendedStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM pos_sync_outbox WHERE id = ?`, ids[2]).Scan(&sentStatus); err != nil {
		t.Fatal(err)
	}
	if failedStatus != "pending" || suspendedStatus != "pending" || sentStatus != "sent" {
		t.Fatalf("unexpected retry statuses: failed=%s suspended=%s sent=%s", failedStatus, suspendedStatus, sentStatus)
	}
}

func TestDevBootstrapDemoIsGatedAndCreatesLoginData(t *testing.T) {
	f := newAPIFixture(t)

	disabledReq := httptest.NewRequest(http.MethodPost, "/api/v1/dev/bootstrap-demo", nil)
	disabled := httptest.NewRecorder()
	f.router.ServeHTTP(disabled, disabledReq)
	if disabled.Code != http.StatusForbidden {
		t.Fatalf("expected disabled bootstrap to return 403, got %d: %s", disabled.Code, disabled.Body.String())
	}

	t.Setenv("POS_DEV_TOOLS", "1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/bootstrap-demo", nil)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var boot app.DemoBootstrapResult
	if err := json.Unmarshal(rr.Body.Bytes(), &boot); err != nil {
		t.Fatal(err)
	}
	if boot.PairingCode == "" || boot.CashierPIN != "1111" || boot.ManagerPIN != "2222" || boot.ManagerEmployeeID == "" || len(boot.TableIDs) == 0 || len(boot.MenuItemIDs) == 0 {
		t.Fatalf("unexpected bootstrap result: %+v", boot)
	}

	loginBody := `{"node_device_id":"` + boot.NodeDeviceID + `","client_device_id":"api-demo-client","pin":"1111"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/pin-login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	login := httptest.NewRecorder()
	f.router.ServeHTTP(login, loginReq)
	if login.Code != http.StatusCreated {
		t.Fatalf("expected demo cashier PIN login to work, got %d: %s", login.Code, login.Body.String())
	}
}

func TestCORSPreflightForPairingAPI(t *testing.T) {
	f := newAPIFixture(t)

	for _, origin := range []string{"http://localhost:5173", "http://host.docker.internal:5173"} {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/system/pair", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type,x-client-device-id")
		rr := httptest.NewRecorder()
		f.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected preflight 204 for %s, got %d: %s", origin, rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("expected CORS origin header %q, got %q", origin, got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") || !strings.Contains(got, "OPTIONS") {
			t.Fatalf("expected CORS methods to include POST and OPTIONS, got %q", got)
		}
	}
}

func TestIssueFirstPrecheckThroughPublicAPI(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	prechecksBefore := countAPIRows(t, f, "prechecks")
	checksBefore := countAPIRows(t, f, "checks")
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")

	rr := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-issue-precheck","node_device_id":"`+f.device.ID+`"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, rr)
	if precheck.OrderID != order.ID || precheck.Status != domain.PrecheckIssued || precheck.Version != 1 || precheck.Total != 2000 {
		t.Fatalf("unexpected precheck: %+v", precheck)
	}
	lockedOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if lockedOrder.Status != domain.OrderLocked {
		t.Fatalf("expected order locked, got %s", lockedOrder.Status)
	}
	if prechecks := countAPIRows(t, f, "prechecks"); prechecks != prechecksBefore+1 {
		t.Fatalf("expected one precheck row, before=%d after=%d", prechecksBefore, prechecks)
	}
	if checks := countAPIRows(t, f, "checks"); checks != checksBefore {
		t.Fatalf("expected issue precheck not to create final check, before=%d after=%d", checksBefore, checks)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one local event row, before=%d after=%d", eventsBefore, events)
	}
}

func TestGetPrecheckByIDThroughPublicAPI(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-get-precheck","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)

	rr := f.get(t, "/api/v1/prechecks/"+precheck.ID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	got := decodeAPIResponse[domain.Precheck](t, rr)
	if got.ID != precheck.ID || got.OrderID != order.ID || got.Status != domain.PrecheckIssued {
		t.Fatalf("unexpected precheck: %+v", got)
	}
}

func TestListPrechecksByOrderThroughPublicAPI(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-list-prechecks","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)

	rr := f.get(t, "/api/v1/orders/"+order.ID+"/prechecks")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	got := decodeAPIResponse[[]domain.Precheck](t, rr)
	if len(got) != 1 || got[0].ID != precheck.ID {
		t.Fatalf("expected one listed precheck, got %+v", got)
	}
}

func TestAddOrderLineAfterIssuedPrecheckFailsThroughPublicAPI(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-lock-order","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", issued.Code, issued.Body.String())
	}

	rr := f.postJSON(t, "/api/v1/orders/"+order.ID+"/lines", `{"command_id":"cmd-api-add-line-after-precheck","node_device_id":"`+f.device.ID+`","menu_item_id":"`+f.menuItem.ID+`","quantity":1}`)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDuplicatePrecheckCommandIDDoesNotCreateSecondPrecheckThroughPublicAPI(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	body := `{"command_id":"cmd-api-duplicate-precheck","node_device_id":"` + f.device.ID + `"}`
	rr := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	prechecksBefore := countAPIRows(t, f, "prechecks")
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")

	duplicate := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", body)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", duplicate.Code, duplicate.Body.String())
	}
	if prechecks := countAPIRows(t, f, "prechecks"); prechecks != prechecksBefore {
		t.Fatalf("expected no second precheck, before=%d after=%d", prechecksBefore, prechecks)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no second outbox row, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no second local event row, before=%d after=%d", eventsBefore, events)
	}
}

func TestCancelPrecheckThroughPublicAPIRequiresManagerOverride(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-cancel-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	body := `{"command_id":"cmd-api-cancel-precheck","node_device_id":"` + f.device.ID + `","manager_employee_id":"` + f.manager.ID + `","manager_pin":"2468","cancellation_reason":"guest changed order"}`
	rr := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/cancel", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	cancelled := decodeAPIResponse[domain.Precheck](t, rr)
	if cancelled.Status != domain.PrecheckCancelled {
		t.Fatalf("expected cancelled precheck, got %+v", cancelled)
	}
	if audit := countAPIRows(t, f, "manager_override_audit"); audit != 1 {
		t.Fatalf("expected one manager override audit row, got %d", audit)
	}
}

func TestCapturePaymentThroughPrecheckAPICreatesFinalCheck(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-payment-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	rr := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/payments", `{"command_id":"cmd-api-payment-full","node_device_id":"`+f.device.ID+`","method":"cash","amount":`+fmt.Sprint(precheck.Total)+`,"currency":"RUB"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	payment := decodeAPIResponse[domain.Payment](t, rr)
	if payment.PrecheckID != precheck.ID {
		t.Fatalf("expected payment for precheck %s, got %+v", precheck.ID, payment)
	}
	if checks := countAPIRows(t, f, "checks"); checks != 1 {
		t.Fatalf("expected one final check, got %d", checks)
	}
	gotOrder, err := f.repo.GetOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOrder.Status != domain.OrderClosed {
		t.Fatalf("expected order closed, got %s", gotOrder.Status)
	}
}
