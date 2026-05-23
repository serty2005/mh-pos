package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	httpx "pos-backend/internal/platform/http"
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

func apiContainsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

type apiFixedClock struct {
	now time.Time
}

func newAPIFixedClock() *apiFixedClock {
	return &apiFixedClock{now: time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)}
}

func (c *apiFixedClock) Now() time.Time {
	return c.now
}

func (c *apiFixedClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
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
	archiveDir string
	clock      *apiFixedClock
}

func newAPIFixture(t *testing.T) *apiFixture {
	t.Helper()
	ctx := context.Background()
	rootDir := t.TempDir()
	db, err := platformsqlite.Open(filepath.Join(rootDir, "pos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := platformsqlite.MigrateDir(ctx, db, filepath.Join("..", "..", "..", "migrations", "sqlite")); err != nil {
		t.Fatal(err)
	}
	repo := possqlite.NewRepository(db)
	testClock := newAPIFixedClock()
	archiveDir := filepath.Join(rootDir, "archives")
	service := app.NewServiceWithOptions(repo, platformsqlite.NewTxManager(db), &apiTestIDs{}, testClock, app.ServiceOptions{StorageArchiveDir: archiveDir})
	f := &apiFixture{ctx: ctx, db: db, repo: repo, service: service, router: api.NewRouter(service), clientID: "api-client-1", archiveDir: archiveDir, clock: testClock}
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
	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     apiSeedMeta("bootstrap-device"),
		Name:            string(appshared.RoleCashier),
		PermissionsJSON: appshared.RolePermissionsJSON(appshared.RoleCashier),
	})
	if err != nil {
		t.Fatal(err)
	}
	managerRole, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     apiSeedMeta("bootstrap-device"),
		Name:            string(appshared.RoleManager),
		PermissionsJSON: appshared.RolePermissionsJSON(appshared.RoleManager),
	})
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

func (f *apiFixture) makeOrderOlderThanArchiveCutoff(t *testing.T, orderID string) {
	t.Helper()
	if _, err := f.db.ExecContext(f.ctx, `UPDATE checks SET business_date_local = '2026-05-03' WHERE order_id = ?`, orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `
UPDATE payments
SET business_date_local = '2026-05-03'
WHERE precheck_id IN (
  SELECT id
  FROM prechecks
  WHERE order_id = ?
)`, orderID); err != nil {
		t.Fatal(err)
	}
}

func (f *apiFixture) openCashSession(t *testing.T) *domain.CashSession {
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

func (f *apiFixture) useManagerOperator(t *testing.T) {
	t.Helper()
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-api-manager-login",
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
	f.employee = f.manager
	f.session = &login.Session
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
	case "prechecks", "checks", "payments", "payment_attempts", "financial_operations", "financial_operation_items", "pos_sync_outbox", "local_event_log", "manager_override_audit", "auth_sessions", "halls", "tables", "catalog_items", "cloud_master_sync_state":
	default:
		t.Fatalf("unexpected table %q", table)
	}
	var n int
	if err := f.db.QueryRowContext(f.ctx, fmt.Sprintf("SELECT COUNT(1) FROM %s", table)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func assertSafeConflictAPIError(t *testing.T, rr *httptest.ResponseRecorder, forbiddenSubstrings ...string) {
	t.Helper()
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected safe conflict 409, got %d: %s", rr.Code, rr.Body.String())
	}
	errBody := decodeAPIResponse[httpx.ErrorResponse](t, rr)
	if errBody.Error.Code != "CONFLICT" || errBody.Error.MessageKey != "errors.conflict" {
		t.Fatalf("expected safe conflict error contract, got %+v", errBody.Error)
	}
	if got := rr.Header().Get("X-Error-Code"); got != "CONFLICT" {
		t.Fatalf("expected X-Error-Code CONFLICT, got %q", got)
	}
	if len(errBody.Error.Details) != 0 {
		t.Fatalf("expected conflict response without unsafe details, got %+v", errBody.Error.Details)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode conflict envelope: %v; body=%s", err, rr.Body.String())
	}
	if len(envelope) != 1 || envelope["error"] == nil {
		t.Fatalf("expected only top-level error envelope, got %s", rr.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope["error"], &fields); err != nil {
		t.Fatalf("decode conflict error object: %v; body=%s", err, rr.Body.String())
	}
	allowed := map[string]bool{
		"code":           true,
		"message_key":    true,
		"details":        true,
		"correlation_id": true,
	}
	for field := range fields {
		if !allowed[field] {
			t.Fatalf("unexpected unsafe conflict error field %q in %s", field, rr.Body.String())
		}
	}

	rawBody := strings.ToLower(rr.Body.String())
	leaks := append([]string{
		"internal_error",
		"constraint",
		"foreign key",
		"sqlite",
		"sql:",
		"stack",
		"panic",
		"domain invariant",
		"financial operation",
		"operation line",
	}, forbiddenSubstrings...)
	for _, leak := range leaks {
		if strings.Contains(rawBody, strings.ToLower(leak)) {
			t.Fatalf("expected safe conflict response not to expose %q: %s", leak, rr.Body.String())
		}
	}
}

type apiFinancialOperationOutboxEnvelope struct {
	Version         string  `json:"version"`
	EventID         string  `json:"event_id"`
	CommandID       string  `json:"command_id"`
	EventType       string  `json:"event_type"`
	AggregateType   string  `json:"aggregate_type"`
	AggregateID     string  `json:"aggregate_id"`
	RestaurantID    *string `json:"restaurant_id"`
	DeviceID        string  `json:"device_id"`
	NodeDeviceID    string  `json:"node_device_id"`
	ClientDeviceID  *string `json:"client_device_id"`
	ShiftID         *string `json:"shift_id"`
	ActorEmployeeID *string `json:"actor_employee_id"`
	SessionID       *string `json:"session_id"`
	Payload         struct {
		Origin domain.CommandOrigin      `json:"origin"`
		Data   domain.FinancialOperation `json:"data"`
	} `json:"payload"`
}

func assertAPIFinancialOperationOutboxEnvelope(t *testing.T, f *apiFixture, commandID, eventType string, operation domain.FinancialOperation) {
	t.Helper()
	msg, err := f.repo.GetOutboxByCommandID(f.ctx, commandID)
	if err != nil {
		t.Fatal(err)
	}
	if msg.CommandType != eventType || msg.AggregateType != "FinancialOperation" || msg.AggregateID != operation.ID {
		t.Fatalf("unexpected financial operation outbox row: %+v", msg)
	}
	if msg.Origin != domain.OriginEdgeDevice || msg.SyncDirection != domain.SyncDirectionEdgeToCloud || msg.Status != domain.OutboxPending {
		t.Fatalf("unexpected financial operation outbox delivery state: origin=%s direction=%s status=%s", msg.Origin, msg.SyncDirection, msg.Status)
	}
	if msg.RestaurantID == nil || *msg.RestaurantID != f.restaurant.ID || msg.DeviceID != f.device.ID || msg.NodeDeviceID != f.device.ID {
		t.Fatalf("unexpected financial operation outbox device scope: %+v", msg)
	}
	if msg.ClientDeviceID == nil || *msg.ClientDeviceID != f.clientID || msg.ActorEmployeeID == nil || *msg.ActorEmployeeID != operation.CreatedByEmployeeID || msg.SessionID == nil || *msg.SessionID != f.session.ID {
		t.Fatalf("unexpected financial operation outbox actor scope: %+v", msg)
	}

	var envelope apiFinancialOperationOutboxEnvelope
	if err := json.Unmarshal([]byte(msg.PayloadJSON), &envelope); err != nil {
		t.Fatalf("decode financial operation outbox envelope: %v; payload=%s", err, msg.PayloadJSON)
	}
	if envelope.Version != domain.SyncEnvelopeVersion || envelope.EventID == "" || envelope.CommandID != commandID || envelope.EventType != eventType || envelope.AggregateType != "FinancialOperation" || envelope.AggregateID != operation.ID {
		t.Fatalf("unexpected financial operation outbox envelope identity: %+v", envelope)
	}
	if envelope.RestaurantID == nil || *envelope.RestaurantID != f.restaurant.ID || envelope.DeviceID != f.device.ID || envelope.NodeDeviceID != f.device.ID {
		t.Fatalf("unexpected financial operation outbox envelope device scope: %+v", envelope)
	}
	if envelope.ClientDeviceID == nil || *envelope.ClientDeviceID != f.clientID || envelope.ShiftID == nil || *envelope.ShiftID != operation.ShiftID || envelope.ActorEmployeeID == nil || *envelope.ActorEmployeeID != operation.CreatedByEmployeeID || envelope.SessionID == nil || *envelope.SessionID != f.session.ID {
		t.Fatalf("unexpected financial operation outbox envelope actor scope: %+v", envelope)
	}
	if envelope.Payload.Origin != domain.OriginEdgeDevice {
		t.Fatalf("expected edge_device payload origin, got %s", envelope.Payload.Origin)
	}
	if envelope.Payload.Data.ID != operation.ID || envelope.Payload.Data.CheckID != operation.CheckID || envelope.Payload.Data.Type != operation.Type || envelope.Payload.Data.Kind != operation.Kind || envelope.Payload.Data.Amount != operation.Amount {
		t.Fatalf("unexpected financial operation outbox payload: %+v", envelope.Payload.Data)
	}

	var localEvents int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM local_event_log WHERE event_id = ? AND command_id = ? AND envelope_version = ? AND event_type = ? AND aggregate_type = 'FinancialOperation' AND aggregate_id = ? AND shift_id = ?`, envelope.EventID, commandID, domain.SyncEnvelopeVersion, eventType, operation.ID, operation.ShiftID).Scan(&localEvents); err != nil {
		t.Fatal(err)
	}
	if localEvents != 1 {
		t.Fatalf("expected one matching local event for %s outbox envelope, got %d", eventType, localEvents)
	}
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

func TestRevokedSessionReturnsSafeUnauthorizedError(t *testing.T) {
	f := newAPIFixture(t)
	if _, err := f.service.Logout(f.ctx, app.LogoutCommand{
		CommandMeta: f.edgeMeta(),
		SessionID:   f.session.ID,
	}); err != nil {
		t.Fatal(err)
	}

	current := f.get(t, "/api/v1/auth/session?node_device_id="+f.device.ID+"&session_id="+f.session.ID)
	if current.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked session, got %d: %s", current.Code, current.Body.String())
	}
	body := decodeAPIResponse[httpx.ErrorResponse](t, current)
	if body.Error.Code != "SESSION_REVOKED" || body.Error.MessageKey != "errors.session.revoked" {
		t.Fatalf("expected session revoked error contract, got: %+v", body.Error)
	}
	if strings.Contains(current.Body.String(), "pin") || strings.Contains(current.Body.String(), "hash") {
		t.Fatalf("expected auth error not to expose sensitive data: %s", current.Body.String())
	}
}

func TestWrongClientDeviceReturnsSafeForbiddenError(t *testing.T) {
	f := newAPIFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/employee-shifts/current?node_device_id="+f.device.ID, nil)
	req.Header.Set("X-Node-Device-ID", f.device.ID)
	req.Header.Set("X-Client-Device-ID", "wrong-client")
	req.Header.Set("X-Actor-Employee-ID", f.employee.ID)
	req.Header.Set("X-Session-ID", f.session.ID)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong client device, got %d: %s", rr.Code, rr.Body.String())
	}
	body := decodeAPIResponse[httpx.ErrorResponse](t, rr)
	if body.Error.Code != "SESSION_CONTEXT_MISMATCH" || body.Error.MessageKey != "errors.session.contextMismatch" {
		t.Fatalf("expected session context mismatch contract, got: %+v", body.Error)
	}
}

func TestCurrentShiftAPIReturnsNullWhenEmployeeHasNoOpenShift(t *testing.T) {
	f := newAPIFixture(t)

	rr := f.get(t, "/api/v1/employee-shifts/current?node_device_id="+f.device.ID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for optional current shift empty state, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "null" {
		t.Fatalf("expected JSON null body for optional current shift empty state, got %q", rr.Body.String())
	}
}

func TestCurrentShiftAPIReturnsAuthenticatedEmployeeOpenShift(t *testing.T) {
	f := newAPIFixture(t)
	shift, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	rr := f.get(t, "/api/v1/employee-shifts/current?node_device_id="+f.device.ID)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for current shift, got %d: %s", rr.Code, rr.Body.String())
	}
	got := decodeAPIResponse[domain.Shift](t, rr)
	if got.ID != shift.ID || got.OpenedByEmployeeID != f.employee.ID || got.Status != domain.ShiftOpen {
		t.Fatalf("unexpected current shift: %+v", got)
	}
}

func TestRequestAuditLogContainsContractFieldsAndNoPINLeak(t *testing.T) {
	f := newAPIFixture(t)
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	body := `{"pairing_code":"MHPOS:<restaurant_id>:demo-edge-node-1","pin":"9999"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/pair", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Device-ID", f.device.ID)
	req.Header.Set("X-Client-Device-ID", f.clientID)
	req.Header.Set("X-Actor-Employee-ID", f.employee.ID)
	req.Header.Set("X-Session-ID", f.session.ID)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	raw := logs.String()
	for _, required := range []string{
		`"operation":"http.request"`,
		`"action":"POST /api/v1/system/pair"`,
		`"result":"rejected"`,
		`"error_code":"VALIDATION_FAILED"`,
		`"request_id":"`,
		`"duration_ms":`,
		`"node_device_id":"`,
		`"client_device_id":"`,
		`"session_id":"`,
		`"actor_employee_id":"`,
	} {
		if !strings.Contains(raw, required) {
			t.Fatalf("expected log to contain %q, logs=%s", required, raw)
		}
	}
	if strings.Contains(raw, "9999") || strings.Contains(raw, `"pin"`) || strings.Contains(raw, "manager_pin") {
		t.Fatalf("expected secret fields not to be logged, logs=%s", raw)
	}
}

func TestPairingRejectsPlaceholderIDs(t *testing.T) {
	f := newAPIFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/pair", bytes.NewBufferString(`{"pairing_code":"MHPOS:<restaurant_id>:demo-edge-node-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for placeholder pairing code, got %d: %s", rr.Code, rr.Body.String())
	}
	body := decodeAPIResponse[httpx.ErrorResponse](t, rr)
	if body.Error.Code != "VALIDATION_FAILED" || body.Error.MessageKey != "errors.validation" {
		t.Fatalf("expected validation error contract, got: %+v", body.Error)
	}
	if strings.Contains(rr.Body.String(), "MHPOS:<restaurant_id>") {
		t.Fatalf("expected raw pairing payload not to be exposed, got: %s", rr.Body.String())
	}
}

func TestPairingRejectsUnknownRestaurantID(t *testing.T) {
	f := newAPIFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/pair", bytes.NewBufferString(`{"pairing_code":"MHPOS:unknown-restaurant:demo-edge-node-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown restaurant in pairing code, got %d: %s", rr.Code, rr.Body.String())
	}
	body := decodeAPIResponse[httpx.ErrorResponse](t, rr)
	if body.Error.Code != "VALIDATION_FAILED" || body.Error.MessageKey != "errors.validation" {
		t.Fatalf("expected validation error contract, got: %+v", body.Error)
	}
}

func TestPinLoginRateLimitReturnsTooManyRequests(t *testing.T) {
	f := newAPIFixture(t)
	loginBody := `{"command_id":"cmd-api-pin-rate-limit","node_device_id":"` + f.device.ID + `","client_device_id":"` + f.clientID + `","pin":"9999"}`
	for i := 0; i < 4; i++ {
		rr := f.postJSON(t, "/api/v1/auth/pin-login", loginBody)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 before limit reached, got %d: %s", rr.Code, rr.Body.String())
		}
		if strings.Contains(rr.Body.String(), "9999") {
			t.Fatalf("expected login error body not to expose attempted pin: %s", rr.Body.String())
		}
	}
	limited := f.postJSON(t, "/api/v1/auth/pin-login", loginBody)
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after repeated attempts, got %d: %s", limited.Code, limited.Body.String())
	}
	errBody := decodeAPIResponse[httpx.ErrorResponse](t, limited)
	if strings.Contains(limited.Body.String(), "9999") {
		t.Fatalf("expected rate-limit error not to expose pin, got: %s", limited.Body.String())
	}
	if errBody.Error.Code != "RATE_LIMITED" || errBody.Error.MessageKey != "errors.rateLimit" {
		t.Fatalf("expected rate limit error contract, got: %+v", errBody.Error)
	}
}

func TestPinLoginRateLimitResetsAfterLockoutWindow(t *testing.T) {
	f := newAPIFixture(t)
	invalid := `{"command_id":"cmd-api-pin-rate-window","node_device_id":"` + f.device.ID + `","client_device_id":"` + f.clientID + `","pin":"0000"}`
	for i := 0; i < 5; i++ {
		_ = f.postJSON(t, "/api/v1/auth/pin-login", invalid)
	}
	limited := f.postJSON(t, "/api/v1/auth/pin-login", invalid)
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 while lockout active, got %d: %s", limited.Code, limited.Body.String())
	}
	f.clock.Advance(16 * time.Minute)
	valid := `{"command_id":"cmd-api-pin-rate-window-valid","node_device_id":"` + f.device.ID + `","client_device_id":"` + f.clientID + `","pin":"1111"}`
	recovered := f.postJSON(t, "/api/v1/auth/pin-login", valid)
	if recovered.Code != http.StatusCreated {
		t.Fatalf("expected successful login after lockout window, got %d: %s", recovered.Code, recovered.Body.String())
	}
}

func TestMasterDataWriteAPIsRejectEdgeRuntimeMutation(t *testing.T) {
	f := newAPIFixture(t)
	hallResp := f.postJSON(t, "/api/v1/halls", `{"node_device_id":"`+f.device.ID+`","restaurant_id":"`+f.restaurant.ID+`","name":"Terrace"}`)
	if hallResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for removed Edge hall mutation route, got %d: %s", hallResp.Code, hallResp.Body.String())
	}
	menuResp := f.postJSON(t, "/api/v1/menu/items", `{"node_device_id":"`+f.device.ID+`","catalog_item_id":"`+f.menuItem.CatalogItemID+`","name":"Tea","price":3000,"currency":"RUB"}`)
	if menuResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for removed Edge menu mutation route, got %d: %s", menuResp.Code, menuResp.Body.String())
	}
}

func TestMasterDataListAPIsAreNotRuntimeRoutes(t *testing.T) {
	f := newAPIFixture(t)
	for _, path := range []string{"/api/v1/restaurants", "/api/v1/devices", "/api/v1/roles", "/api/v1/employees"} {
		rr := f.get(t, path)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for removed %s runtime route, got %d: %s", path, rr.Code, rr.Body.String())
		}
	}
}

func TestCatalogReadAPIRequiresCatalogViewPermission(t *testing.T) {
	f := newAPIFixture(t)
	allowed := f.get(t, "/api/v1/catalog/items")
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected 200 for catalog read with catalog view, got %d: %s", allowed.Code, allowed.Body.String())
	}

	role, err := f.service.CreateRole(f.ctx, app.CreateRoleCommand{
		CommandMeta:     apiSeedMeta(f.device.ID),
		Name:            "no-catalog-view",
		PermissionsJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	pinHash, err := appshared.HashPIN("9753", []byte("api-no-catalog-salt"))
	if err != nil {
		t.Fatal(err)
	}
	employee, err := f.service.CreateEmployee(f.ctx, app.CreateEmployeeCommand{
		CommandMeta:  apiSeedMeta(f.device.ID),
		RestaurantID: f.restaurant.ID,
		RoleID:       role.ID,
		Name:         "No Catalog",
		PINHash:      pinHash,
	})
	if err != nil {
		t.Fatal(err)
	}
	login, err := f.service.PinLogin(f.ctx, app.PinLoginCommand{
		CommandMeta: app.CommandMeta{
			CommandID:      "cmd-api-login-no-catalog",
			NodeDeviceID:   f.device.ID,
			DeviceID:       f.device.ID,
			ClientDeviceID: f.clientID,
			Origin:         app.OriginEdgeDevice,
		},
		PIN: "9753",
	})
	if err != nil {
		t.Fatal(err)
	}
	f.employee = employee
	f.session = &login.Session
	denied := f.get(t, "/api/v1/catalog/items")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without catalog view permission, got %d: %s", denied.Code, denied.Body.String())
	}
}

func TestMasterDataIngestAPIAppliesCloudAuthoredCatalogWithoutOutbox(t *testing.T) {
	f := newAPIFixture(t)
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")
	catalogBefore := countAPIRows(t, f, "catalog_items")
	body := `{
		"node_device_id":"` + f.device.ID + `",
		"sync_mode":"incremental",
		"cloud_version":55,
		"catalog_items":[{
			"id":"cloud-api-catalog-1",
			"type":"dish",
			"name":"Cloud API Tea",
			"sku":"CLOUD-API-TEA",
			"base_unit":"portion",
			"active":true
		}]
	}`

	rr := f.postJSON(t, "/api/v1/sync/master-data/catalog", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	result := decodeAPIResponse[app.ApplyMasterDataResult](t, rr)
	if len(result.AppliedStreams) != 1 || result.AppliedStreams[0] != domain.MasterDataStreamCatalog || result.Counts["catalog"] != 1 {
		t.Fatalf("unexpected ingest result: %+v", result)
	}
	if catalog := countAPIRows(t, f, "catalog_items"); catalog != catalogBefore+1 {
		t.Fatalf("expected one catalog item created, before=%d after=%d", catalogBefore, catalog)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no outbox from master ingest, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no local events from master ingest, before=%d after=%d", eventsBefore, events)
	}
	var direction, mode string
	var version int64
	if err := f.db.QueryRowContext(f.ctx, `SELECT direction,sync_mode,last_cloud_version FROM cloud_master_sync_state WHERE node_device_id = ? AND stream_name = 'catalog'`, f.device.ID).Scan(&direction, &mode, &version); err != nil {
		t.Fatal(err)
	}
	if direction != "cloud_to_edge" || mode != "incremental" || version != 55 {
		t.Fatalf("unexpected sync state direction=%s mode=%s version=%d", direction, mode, version)
	}
}

func TestMasterDataIngestAPIRejectsUnknownJSONField(t *testing.T) {
	f := newAPIFixture(t)
	rr := f.postJSON(t, "/api/v1/sync/master-data/pricing_policy", `{
		"node_device_id":"`+f.device.ID+`",
		"cloud_version":56,
		"tax_profiles":[{"id":"tax-api-1","name":"VAT","active":true}],
		"unknown_payload_shape":true
	}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	body := decodeAPIResponse[httpx.ErrorResponse](t, rr)
	if body.Error.Code != "VALIDATION_FAILED" {
		t.Fatalf("expected validation error contract, got %+v", body.Error)
	}
}

func TestFloorReadAndOrderLineEditingAPI(t *testing.T) {
	f := newAPIFixture(t)
	listTables := f.get(t, "/api/v1/tables?restaurant_id="+f.restaurant.ID+"&hall_id="+f.hall.ID)
	if listTables.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listTables.Code, listTables.Body.String())
	}
	tables := decodeAPIResponse[[]domain.Table](t, listTables)
	if len(tables) != 1 || tables[0].ID != f.table.ID {
		t.Fatalf("unexpected table list: %+v", tables)
	}

	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{CommandMeta: f.edgeMeta(), RestaurantID: f.restaurant.ID, OpenedByEmployeeID: f.employee.ID, OpeningCashAmount: 0}); err != nil {
		t.Fatal(err)
	}
	order, err := f.service.CreateOrder(f.ctx, app.CreateOrderCommand{CommandMeta: f.edgeMeta(), TableID: f.table.ID, GuestCount: 1})
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
	details := f.patchJSON(t, "/api/v1/orders/"+order.ID+"/lines/"+line.ID+"/details", `{"node_device_id":"`+f.device.ID+`","course":"2","comment":"no onion"}`)
	if details.Code != http.StatusOK {
		t.Fatalf("expected 200 for line details, got %d: %s", details.Code, details.Body.String())
	}
	detailed := decodeAPIResponse[domain.OrderLine](t, details)
	if detailed.Course == nil || *detailed.Course != "2" || detailed.Comment == nil || *detailed.Comment != "no onion" {
		t.Fatalf("unexpected line details: %+v", detailed)
	}
	otherDeviceID := "api-floor-device-2"
	otherShiftID := "api-floor-shift-2"
	otherOrderID := "api-floor-order-2"
	otherLineID := "api-floor-line-2"
	otherOpenedAt := appshared.DBTime(f.clock.Now().Add(time.Minute))
	if _, err := f.db.ExecContext(f.ctx, `INSERT INTO devices(id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		otherDeviceID, f.restaurant.ID, "POS-2", "Second node", "windows", 1, otherOpenedAt, otherOpenedAt, otherOpenedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `INSERT INTO shifts(id,restaurant_id,device_id,opened_by_employee_id,status,business_date_local,opened_at,opening_cash_amount,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		otherShiftID, f.restaurant.ID, otherDeviceID, f.manager.ID, "open", "2026-05-04", otherOpenedAt, 0, otherOpenedAt, otherOpenedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		otherOrderID, "edge-"+otherOrderID, f.restaurant.ID, otherDeviceID, otherShiftID, "open", f.table.ID, f.table.Name, 1, otherOpenedAt, otherOpenedAt, otherOpenedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `INSERT INTO order_lines(id,order_id,menu_item_id,catalog_item_id,name,quantity,unit_price,total_price,currency_code,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		otherLineID, otherOrderID, f.menuItem.ID, f.menuItem.CatalogItemID, "Soup", 1, 1000, 1000, "RUB", "active", otherOpenedAt, otherOpenedAt); err != nil {
		t.Fatal(err)
	}
	activeResp := f.get(t, "/api/v1/orders/active?hall_id="+f.hall.ID)
	if activeResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for active hall orders, got %d: %s", activeResp.Code, activeResp.Body.String())
	}
	activeOrders := decodeAPIResponse[[]domain.Order](t, activeResp)
	activeIDs := make(map[string]int)
	for _, activeOrder := range activeOrders {
		activeIDs[activeOrder.ID] = len(activeOrder.Lines)
	}
	if activeIDs[order.ID] != 1 || activeIDs[otherOrderID] != 1 || len(activeOrders) != 2 {
		t.Fatalf("unexpected active orders response: %+v", activeOrders)
	}
	if activeOrders[0].Lines[0].Comment == nil || *activeOrders[0].Lines[0].Comment != "no onion" {
		t.Fatalf("expected active orders to include line details, got %+v", activeOrders[0].Lines[0])
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
	clock := &apiFixedClock{}
	now := appshared.DBTime(clock.Now())

	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = 1, last_error = 'temporary', updated_at = ? WHERE id = ?`, now, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'processing', locked_at = ?, locked_by = 'api-worker', updated_at = ? WHERE id = ?`, now, now, ids[1]); err != nil {
		t.Fatal(err)
	}

	rr := f.get(t, "/api/v1/sync/status")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier sync status access, got %d: %s", rr.Code, rr.Body.String())
	}
	f.useManagerOperator(t)
	rr = f.get(t, "/api/v1/sync/status")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	status := decodeAPIResponse[domain.SyncStatus](t, rr)
	if status.Total != countAPIRows(t, f, "pos_sync_outbox") || status.Failed != 1 || status.Processing != 1 {
		t.Fatalf("unexpected sync status: %+v", status)
	}
}

func TestListOutboxAPIRequiresSyncViewPermission(t *testing.T) {
	f := newAPIFixture(t)
	rr := f.get(t, "/api/v1/sync/outbox?limit=5")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier outbox access, got %d: %s", rr.Code, rr.Body.String())
	}
	f.useManagerOperator(t)
	rr = f.get(t, "/api/v1/sync/outbox?limit=5")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager outbox access, got %d: %s", rr.Code, rr.Body.String())
	}
	items := decodeAPIResponse[[]domain.OutboxMessage](t, rr)
	if len(items) == 0 {
		t.Fatal("expected non-empty outbox list")
	}
}

func TestListOutboxAPIRemainsBoundedWithoutClientLimit(t *testing.T) {
	f := newAPIFixture(t)
	for i := 0; i < 120; i++ {
		if _, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{
			CommandMeta: apiSeedMeta(f.device.ID),
			Type:        domain.CatalogItemDish,
			Name:        fmt.Sprintf("Outbox Dish %03d", i),
			SKU:         fmt.Sprintf("OUTBOX-DISH-%03d", i),
			BaseUnit:    "portion",
		}); err != nil {
			t.Fatal(err)
		}
	}

	f.useManagerOperator(t)
	rr := f.get(t, "/api/v1/sync/outbox")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	items := decodeAPIResponse[[]domain.OutboxMessage](t, rr)
	if len(items) != 100 {
		t.Fatalf("expected default bounded outbox page of 100, got %d", len(items))
	}

	rr = f.get(t, "/api/v1/sync/outbox?limit=9999")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for capped outbox read, got %d: %s", rr.Code, rr.Body.String())
	}
	items = decodeAPIResponse[[]domain.OutboxMessage](t, rr)
	if len(items) != 100 {
		t.Fatalf("expected oversized outbox limit to fall back to bounded default 100, got %d", len(items))
	}
}

func TestListLocalEventsAPIRequiresSyncViewPermission(t *testing.T) {
	f := newAPIFixture(t)
	rr := f.get(t, "/api/v1/sync/local-events?limit=5")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier local events access, got %d: %s", rr.Code, rr.Body.String())
	}
	f.useManagerOperator(t)
	rr = f.get(t, "/api/v1/sync/local-events?limit=5")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager local events access, got %d: %s", rr.Code, rr.Body.String())
	}
	items := decodeAPIResponse[[]domain.LocalEvent](t, rr)
	if len(items) == 0 {
		t.Fatal("expected non-empty local events list")
	}
}

func TestListLocalEventsAPIRemainsBoundedWithoutClientLimit(t *testing.T) {
	f := newAPIFixture(t)
	for i := 0; i < 120; i++ {
		if _, err := f.service.CreateCatalogItem(f.ctx, app.CreateCatalogItemCommand{
			CommandMeta: apiSeedMeta(f.device.ID),
			Type:        domain.CatalogItemDish,
			Name:        fmt.Sprintf("Event Dish %03d", i),
			SKU:         fmt.Sprintf("EVENT-DISH-%03d", i),
			BaseUnit:    "portion",
		}); err != nil {
			t.Fatal(err)
		}
	}

	f.useManagerOperator(t)
	rr := f.get(t, "/api/v1/sync/local-events")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	events := decodeAPIResponse[[]domain.LocalEvent](t, rr)
	if len(events) != 100 {
		t.Fatalf("expected default bounded local event page of 100, got %d", len(events))
	}

	rr = f.get(t, "/api/v1/sync/local-events?limit=9999")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for capped local event read, got %d: %s", rr.Code, rr.Body.String())
	}
	events = decodeAPIResponse[[]domain.LocalEvent](t, rr)
	if len(events) != 100 {
		t.Fatalf("expected oversized local event limit to fall back to bounded default 100, got %d", len(events))
	}
}

func TestStorageStatusAPIRequiresSyncViewPermission(t *testing.T) {
	f := newAPIFixture(t)
	rr := f.get(t, "/api/v1/storage/status")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier storage status access, got %d: %s", rr.Code, rr.Body.String())
	}

	f.useManagerOperator(t)
	rr = f.get(t, "/api/v1/storage/status")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager storage status access, got %d: %s", rr.Code, rr.Body.String())
	}
	status := decodeAPIResponse[domain.StorageLifecycleStatus](t, rr)
	if status.SQLite.PageCount <= 0 || status.Retention.Mode != "archive_apply_supported" || !status.Retention.DestructiveApplySupported {
		t.Fatalf("unexpected storage status: %+v", status)
	}
}

func TestStorageRetentionDryRunAPI(t *testing.T) {
	f := newAPIFixture(t)
	rr := f.postJSON(t, "/api/v1/storage/retention/dry-run", `{"cutoff_business_date_local":"2026-05-04"}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier retention dry-run, got %d: %s", rr.Code, rr.Body.String())
	}

	f.useManagerOperator(t)
	rr = f.postJSON(t, "/api/v1/storage/retention/dry-run", `{"cutoff_business_date_local":"2026-05-04"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager retention dry-run, got %d: %s", rr.Code, rr.Body.String())
	}
	result := decodeAPIResponse[domain.StorageRetentionDryRunResult](t, rr)
	if !result.Blocked || result.Mode != "dry_run_only" || result.ResultMode != "dry_run_only" || result.DestructiveApplySupported {
		t.Fatalf("unexpected retention dry-run result: %+v", result)
	}
}

func TestStorageArchiveExportPlanAPIRequiresSyncViewAndReturnsManifestOnly(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	f.openCashSession(t)
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{
		CommandMeta: f.edgeMeta(),
		OrderID:     order.ID,
	})
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
	f.makeOrderOlderThanArchiveCutoff(t, order.ID)

	body := `{"cutoff_business_date_local":"2026-05-04","mode":"manifest_only"}`
	rr := f.postJSON(t, "/api/v1/storage/archive/export-plan", body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier archive export-plan, got %d: %s", rr.Code, rr.Body.String())
	}

	f.useManagerOperator(t)
	rr = f.postJSON(t, "/api/v1/storage/archive/export-plan", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager archive export-plan, got %d: %s", rr.Code, rr.Body.String())
	}
	result := decodeAPIResponse[domain.StorageArchiveExportPlan](t, rr)
	if result.Mode != "manifest_only" || result.ResultMode != "plan_only" || result.DestructiveApplySupported || !result.Blocked || result.ArchiveSet.ClosedOrders != 1 {
		t.Fatalf("unexpected archive export-plan response: %+v", result)
	}
	if result.OpenShifts < 1 || result.OpenCashSessions < 1 ||
		!apiContainsString(result.BlockReasons, "open_shifts") ||
		!apiContainsString(result.BlockReasons, "open_cash_sessions") {
		t.Fatalf("expected operational blockers in archive export-plan response: %+v", result)
	}
	if result.Manifest.FormatVersion != "storage-archive-manifest-v1" || len(result.Manifest.Tables) == 0 {
		t.Fatalf("unexpected archive export-plan manifest: %+v", result.Manifest)
	}
}

func TestStorageArchiveExportAPIRequiresSyncViewAndCreatesArchive(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	f.openCashSession(t)
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	f.makeOrderOlderThanArchiveCutoff(t, order.ID)

	body := `{"cutoff_business_date_local":"2026-05-04","reason":"operator export from API"}`
	rr := f.postJSON(t, "/api/v1/storage/archive/export", body)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier archive export, got %d: %s", rr.Code, rr.Body.String())
	}

	f.useManagerOperator(t)
	rr = f.postJSON(t, "/api/v1/storage/archive/export", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for manager archive export, got %d: %s", rr.Code, rr.Body.String())
	}
	result := decodeAPIResponse[domain.StorageArchiveExportResult](t, rr)
	if result.Mode != "export_only" || result.ResultMode != "export_only" || !result.DestructiveApplySupported || result.RuntimeRowsDeleted || !result.ExportCreated || result.Counts.ClosedOrders != 1 {
		t.Fatalf("unexpected archive export response: %+v", result)
	}
	if _, err := os.Stat(result.ArchivePath); err != nil {
		t.Fatalf("expected archive file to exist: %v", err)
	}
	if _, err := os.Stat(result.ManifestPath); err != nil {
		t.Fatalf("expected manifest file to exist: %v", err)
	}
}

func TestStorageArchiveExportAPIRejectsFutureCutoff(t *testing.T) {
	f := newAPIFixture(t)
	f.useManagerOperator(t)
	rr := f.postJSON(t, "/api/v1/storage/archive/export", `{"cutoff_business_date_local":"2026-05-05","reason":"future cutoff"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for future cutoff, got %d: %s", rr.Code, rr.Body.String())
	}
	body := decodeAPIResponse[httpx.ErrorResponse](t, rr)
	if body.Error.Code != "VALIDATION_FAILED" || body.Error.MessageKey != "errors.validation" {
		t.Fatalf("expected validation error contract, got: %+v", body.Error)
	}
}

func TestStorageArchiveApplyPlanAPIRequiresSyncViewAndReturnsBlockedPlan(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	f.openCashSession(t)
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	f.makeOrderOlderThanArchiveCutoff(t, order.ID)
	cashier := f.employee
	cashierSession := f.session
	f.useManagerOperator(t)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.edgeMeta(),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "api apply plan fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := f.employee
	managerSession := f.session
	body := struct {
		CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
		ArchivePath             string `json:"archive_path"`
		ManifestPath            string `json:"manifest_path"`
		Mode                    string `json:"mode"`
	}{
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	f.employee = cashier
	f.session = cashierSession
	rr := f.postJSON(t, "/api/v1/storage/archive/apply-plan", string(rawBody))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier archive apply-plan, got %d: %s", rr.Code, rr.Body.String())
	}

	f.employee = manager
	f.session = managerSession
	rr = f.postJSON(t, "/api/v1/storage/archive/apply-plan", string(rawBody))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager archive apply-plan, got %d: %s", rr.Code, rr.Body.String())
	}
	result := decodeAPIResponse[domain.StorageArchiveApplyPlan](t, rr)
	if !result.Blocked || result.ResultMode != "apply_blocked" || !result.DestructiveApplySupported || result.RuntimeRowsDeleted {
		t.Fatalf("unexpected archive apply-plan response: %+v", result)
	}
	if !apiContainsString(result.BlockReasons, "pending_edge_to_cloud_outbox") {
		t.Fatalf("expected blocked-by-default apply-plan reasons, got %+v", result.BlockReasons)
	}
}

func TestStorageArchiveApplyReadinessAPIRequiresSyncViewAndReturnsReadOnlyGate(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	f.openCashSession(t)
	precheck, err := f.service.IssuePrecheck(f.ctx, app.IssuePrecheckCommand{CommandMeta: f.edgeMeta(), OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CapturePayment(f.ctx, app.CapturePaymentCommand{CommandMeta: f.edgeMeta(), PrecheckID: precheck.ID, Method: domain.PaymentCash, Amount: precheck.Total, Currency: "RUB"}); err != nil {
		t.Fatal(err)
	}
	f.makeOrderOlderThanArchiveCutoff(t, order.ID)
	cashier := f.employee
	cashierSession := f.session
	f.useManagerOperator(t)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.edgeMeta(),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "api apply readiness fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := f.employee
	managerSession := f.session
	body := struct {
		CutoffBusinessDateLocal string `json:"cutoff_business_date_local"`
		ArchivePath             string `json:"archive_path"`
		ManifestPath            string `json:"manifest_path"`
		Mode                    string `json:"mode"`
	}{
		CutoffBusinessDateLocal: "2026-05-04",
		ArchivePath:             exported.ArchivePath,
		ManifestPath:            exported.ManifestPath,
		Mode:                    "plan_only",
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	f.employee = cashier
	f.session = cashierSession
	rr := f.postJSON(t, "/api/v1/storage/archive/apply-readiness", string(rawBody))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier archive apply-readiness, got %d: %s", rr.Code, rr.Body.String())
	}

	f.employee = manager
	f.session = managerSession
	rr = f.postJSON(t, "/api/v1/storage/archive/apply-readiness", string(rawBody))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager archive apply-readiness, got %d: %s", rr.Code, rr.Body.String())
	}
	result := decodeAPIResponse[domain.StorageArchiveApplyReadiness](t, rr)
	if result.ResultMode != "apply_readiness_only" || !result.DestructiveApplySupported || result.ReadyForDestructiveApply || result.RuntimeRowsDeleted {
		t.Fatalf("unexpected archive apply-readiness response: %+v", result)
	}
	if !result.ArchiveVerified || !result.ManifestVerified || !result.SnapshotPayloadVerified ||
		!result.PendingEdgeToCloudOutbox || result.OpenOperationalBoundaries.Open {
		t.Fatalf("expected verified archive and runtime blockers, got %+v", result)
	}
	if !apiContainsString(result.BlockReasons, "pending_edge_to_cloud_outbox") {
		t.Fatalf("expected pending outbox readiness reason, got %+v", result.BlockReasons)
	}
}

func TestStorageArchiveReadPlanAndLookupAPIRequireSyncView(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	f.openCashSession(t)
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
	f.makeOrderOlderThanArchiveCutoff(t, order.ID)
	cashier := f.employee
	cashierSession := f.session
	f.useManagerOperator(t)
	exported, err := f.service.ExportStorageArchive(f.ctx, app.ArchiveExportCommand{
		CommandMeta:             f.edgeMeta(),
		CutoffBusinessDateLocal: "2026-05-04",
		Reason:                  "api archive read fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := f.employee
	managerSession := f.session

	readBody := struct {
		ArchivePath  string `json:"archive_path"`
		ManifestPath string `json:"manifest_path"`
	}{ArchivePath: exported.ArchivePath, ManifestPath: exported.ManifestPath}
	rawReadBody, err := json.Marshal(readBody)
	if err != nil {
		t.Fatal(err)
	}
	lookupBody := struct {
		ArchivePath  string `json:"archive_path"`
		ManifestPath string `json:"manifest_path"`
		CheckID      string `json:"check_id"`
	}{ArchivePath: exported.ArchivePath, ManifestPath: exported.ManifestPath, CheckID: check.ID}
	rawLookupBody, err := json.Marshal(lookupBody)
	if err != nil {
		t.Fatal(err)
	}

	f.employee = cashier
	f.session = cashierSession
	rr := f.postJSON(t, "/api/v1/storage/archive/verify", string(rawReadBody))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier archive verify, got %d: %s", rr.Code, rr.Body.String())
	}
	rr = f.postJSON(t, "/api/v1/storage/archive/read-plan", string(rawReadBody))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier archive read-plan, got %d: %s", rr.Code, rr.Body.String())
	}
	rr = f.postJSON(t, "/api/v1/storage/archive/lookup", string(rawLookupBody))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier archive lookup, got %d: %s", rr.Code, rr.Body.String())
	}

	f.employee = manager
	f.session = managerSession
	rr = f.postJSON(t, "/api/v1/storage/archive/verify", string(rawReadBody))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager archive verify, got %d: %s", rr.Code, rr.Body.String())
	}
	verify := decodeAPIResponse[domain.StorageArchiveVerifyResult](t, rr)
	if !verify.Valid || verify.ArchiveID != exported.ArchiveID || verify.Counts.ClosedOrders != 1 {
		t.Fatalf("unexpected archive verify API response: %+v", verify)
	}
	rr = f.postJSON(t, "/api/v1/storage/archive/read-plan", string(rawReadBody))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager archive read-plan, got %d: %s", rr.Code, rr.Body.String())
	}
	readPlan := decodeAPIResponse[domain.StorageArchiveReadPlan](t, rr)
	if readPlan.Blocked || readPlan.ResultMode != "read_plan_only" || readPlan.ArchiveID != exported.ArchiveID ||
		readPlan.Returned != 1 || len(readPlan.ArchivedClosedOrders) != 1 {
		t.Fatalf("unexpected archive read-plan API response: %+v", readPlan)
	}
	rr = f.postJSON(t, "/api/v1/storage/archive/lookup", string(rawLookupBody))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for manager archive lookup, got %d: %s", rr.Code, rr.Body.String())
	}
	lookup := decodeAPIResponse[domain.StorageArchiveLookupPreview](t, rr)
	if lookup.Blocked || !lookup.Lookup.Found || lookup.Check == nil || lookup.Precheck == nil {
		t.Fatalf("unexpected archive lookup API response: %+v", lookup)
	}
}

func TestRetryFailedAPIResetsFailedAndSuspendedButNotSent(t *testing.T) {
	f := newAPIFixture(t)
	ids := apiOutboxIDs(t, f, 3)
	clock := &apiFixedClock{}
	now := appshared.DBTime(clock.Now())
	//now := appshared.DBTime(apiFixedClock{}.Now())
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = 1, last_error = 'temporary', updated_at = ? WHERE id = ?`, now, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'suspended', attempts = 4, last_error = 'threshold', updated_at = ? WHERE id = ?`, now, ids[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `UPDATE pos_sync_outbox SET status = 'sent', sent_at = ?, updated_at = ? WHERE id = ?`, now, now, ids[2]); err != nil {
		t.Fatal(err)
	}

	f.useManagerOperator(t)
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

func TestRetryFailedAPIRequiresSyncRetryPermission(t *testing.T) {
	f := newAPIFixture(t)
	rr := f.postJSON(t, "/api/v1/sync/retry-failed", `{}`)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRemovedLocalBootstrapRouteReturnsNotFound(t *testing.T) {
	f := newAPIFixture(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/bootstrap-demo", nil)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected removed bootstrap route to return 404, got %d: %s", rr.Code, rr.Body.String())
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
	f.useManagerOperator(t)
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
	f.openCashSession(t)
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

func TestListCheckFinancialOperationsThroughPublicAPI(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	f.openCashSession(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-ledger-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected precheck 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	paid := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/payments", `{"command_id":"cmd-api-ledger-payment","node_device_id":"`+f.device.ID+`","method":"cash","amount":`+fmt.Sprint(precheck.Total)+`,"currency":"RUB"}`)
	if paid.Code != http.StatusCreated {
		t.Fatalf("expected payment 201, got %d: %s", paid.Code, paid.Body.String())
	}
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	f.useManagerOperator(t)
	cancelled := f.postJSON(t, "/api/v1/checks/"+check.ID+"/cancellations", `{"command_id":"cmd-api-ledger-cancel","node_device_id":"`+f.device.ID+`","operation_kind":"full","inventory_disposition":"manual_review","reason":"pilot read endpoint"}`)
	if cancelled.Code != http.StatusCreated {
		t.Fatalf("expected cancellation 201, got %d: %s", cancelled.Code, cancelled.Body.String())
	}
	operation := decodeAPIResponse[domain.FinancialOperation](t, cancelled)

	rr := f.get(t, "/api/v1/checks/"+check.ID+"/financial-operations?limit=10&offset=0")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	operations := decodeAPIResponse[[]domain.FinancialOperation](t, rr)
	if len(operations) != 1 || operations[0].ID != operation.ID || operations[0].Type != domain.FinancialOperationCancellation {
		t.Fatalf("unexpected operations response: %+v", operations)
	}
	if len(operations[0].Items) != 1 || operations[0].Items[0].Scope != domain.FinancialItemWholeCheck {
		t.Fatalf("expected whole-check operation item, got %+v", operations[0].Items)
	}

	rr = f.get(t, "/api/v1/financial-operations?business_date_from=2026-05-04&business_date_to=2026-05-04&operation_type=cancellation&shift_id="+operation.ShiftID+"&check_id="+check.ID+"&limit=1")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from financial operation report endpoint, got %d: %s", rr.Code, rr.Body.String())
	}
	reportOperations := decodeAPIResponse[[]domain.FinancialOperation](t, rr)
	if len(reportOperations) != 1 || reportOperations[0].ID != operation.ID {
		t.Fatalf("unexpected report endpoint response: %+v", reportOperations)
	}

	rr = f.get(t, "/api/v1/financial-operations?operation_type=void&limit=1")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid operation type to return 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRecordRefundAboveCheckTotalThroughPublicAPIReturnsConflict(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	cashSession := f.openCashSession(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-over-refund-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected precheck 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	paid := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/payments", `{"command_id":"cmd-api-over-refund-payment","node_device_id":"`+f.device.ID+`","method":"cash","amount":`+fmt.Sprint(precheck.Total)+`,"currency":"RUB"}`)
	if paid.Code != http.StatusCreated {
		t.Fatalf("expected payment 201, got %d: %s", paid.Code, paid.Body.String())
	}
	payment := decodeAPIResponse[domain.Payment](t, paid)
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	cashier := f.employee
	cashierSession := f.session
	f.useManagerOperator(t)
	manager := f.employee
	managerSession := f.session
	if _, err := f.service.CloseCashSession(f.ctx, app.CloseCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 cashSession.ID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = cashier
	f.session = cashierSession
	if _, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 order.ShiftID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = manager
	f.session = managerSession
	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	operationsBefore := countAPIRows(t, f, "financial_operations")
	itemsBefore := countAPIRows(t, f, "financial_operation_items")
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")

	body := fmt.Sprintf(`{"command_id":"cmd-api-refund-above-check-total","node_device_id":"%s","operation_kind":"partial","inventory_disposition":"no_stock_effect","reason":"over refund must be rejected","items":[{"scope":"whole_check","amount":%d,"currency":"RUB"}]}`, f.device.ID, check.Total+1)
	rr := f.postJSON(t, "/api/v1/checks/"+check.ID+"/refunds", body)
	assertSafeConflictAPIError(t, rr, "financial operation exceeds remaining check amount")
	if operations := countAPIRows(t, f, "financial_operations"); operations != operationsBefore {
		t.Fatalf("expected no refund operation write, before=%d after=%d", operationsBefore, operations)
	}
	if items := countAPIRows(t, f, "financial_operation_items"); items != itemsBefore {
		t.Fatalf("expected no refund item write, before=%d after=%d", itemsBefore, items)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no refund outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no refund local event write, before=%d after=%d", eventsBefore, events)
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

func TestRecordRefundAfterCancellationAboveRemainingThroughPublicAPIReturnsConflict(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	cashSession := f.openCashSession(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-mixed-over-refund-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected precheck 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	paid := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/payments", `{"command_id":"cmd-api-mixed-over-refund-payment","node_device_id":"`+f.device.ID+`","method":"cash","amount":`+fmt.Sprint(precheck.Total)+`,"currency":"RUB"}`)
	if paid.Code != http.StatusCreated {
		t.Fatalf("expected payment 201, got %d: %s", paid.Code, paid.Body.String())
	}
	payment := decodeAPIResponse[domain.Payment](t, paid)
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	cashier := f.employee
	cashierSession := f.session
	f.useManagerOperator(t)
	cancellationAmount := check.Total / 2
	cancelBody := fmt.Sprintf(`{"command_id":"cmd-api-mixed-cancel-partial","node_device_id":"%s","operation_kind":"partial","inventory_disposition":"manual_review","reason":"prepared item cancelled","items":[{"scope":"whole_check","amount":%d,"currency":"RUB"}]}`, f.device.ID, cancellationAmount)
	cancelled := f.postJSON(t, "/api/v1/checks/"+check.ID+"/cancellations", cancelBody)
	if cancelled.Code != http.StatusCreated {
		t.Fatalf("expected cancellation 201, got %d: %s", cancelled.Code, cancelled.Body.String())
	}
	cancellation := decodeAPIResponse[domain.FinancialOperation](t, cancelled)
	if cancellation.Type != domain.FinancialOperationCancellation || cancellation.Amount != cancellationAmount {
		t.Fatalf("unexpected cancellation operation: %+v", cancellation)
	}

	manager := f.employee
	managerSession := f.session
	if _, err := f.service.CloseCashSession(f.ctx, app.CloseCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 cashSession.ID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = cashier
	f.session = cashierSession
	if _, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 order.ShiftID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = manager
	f.session = managerSession
	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	operationsBefore := countAPIRows(t, f, "financial_operations")
	itemsBefore := countAPIRows(t, f, "financial_operation_items")
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")

	remaining := check.Total - cancellation.Amount
	body := fmt.Sprintf(`{"command_id":"cmd-api-refund-above-remaining-after-cancel","node_device_id":"%s","operation_kind":"partial","inventory_disposition":"no_stock_effect","reason":"refund above remaining must be rejected","items":[{"scope":"whole_check","amount":%d,"currency":"RUB"}]}`, f.device.ID, remaining+1)
	rr := f.postJSON(t, "/api/v1/checks/"+check.ID+"/refunds", body)
	assertSafeConflictAPIError(t, rr, "financial operation exceeds remaining check amount")
	if operations := countAPIRows(t, f, "financial_operations"); operations != operationsBefore {
		t.Fatalf("expected no refund operation write, before=%d after=%d", operationsBefore, operations)
	}
	if items := countAPIRows(t, f, "financial_operation_items"); items != itemsBefore {
		t.Fatalf("expected no refund item write, before=%d after=%d", itemsBefore, items)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no refund outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no refund local event write, before=%d after=%d", eventsBefore, events)
	}
	operations, err := f.repo.ListFinancialOperationsByCheck(f.ctx, check.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || operations[0].ID != cancellation.ID || operations[0].Type != domain.FinancialOperationCancellation {
		t.Fatalf("expected only the original cancellation operation, got %+v", operations)
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

func TestRecordRefundAfterLineCancellationAboveRemainingQuantityThroughPublicAPIReturnsConflict(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	lines, err := f.repo.ListOrderLines(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Quantity != 2 {
		t.Fatalf("expected one order line with quantity 2, got %+v", lines)
	}
	line := lines[0]
	cashSession := f.openCashSession(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-line-over-refund-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected precheck 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	paid := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/payments", `{"command_id":"cmd-api-line-over-refund-payment","node_device_id":"`+f.device.ID+`","method":"cash","amount":`+fmt.Sprint(precheck.Total)+`,"currency":"RUB"}`)
	if paid.Code != http.StatusCreated {
		t.Fatalf("expected payment 201, got %d: %s", paid.Code, paid.Body.String())
	}
	payment := decodeAPIResponse[domain.Payment](t, paid)
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	cashier := f.employee
	cashierSession := f.session
	f.useManagerOperator(t)
	unitAmount := line.TotalPrice / line.Quantity
	cancelBody := fmt.Sprintf(`{"command_id":"cmd-api-line-cancel-one","node_device_id":"%s","operation_kind":"partial","inventory_disposition":"manual_review","reason":"one prepared item cancelled","items":[{"scope":"order_line","order_line_id":"%s","quantity":1,"amount":%d,"currency":"RUB"}]}`, f.device.ID, line.ID, unitAmount)
	cancelled := f.postJSON(t, "/api/v1/checks/"+check.ID+"/cancellations", cancelBody)
	if cancelled.Code != http.StatusCreated {
		t.Fatalf("expected cancellation 201, got %d: %s", cancelled.Code, cancelled.Body.String())
	}
	cancellation := decodeAPIResponse[domain.FinancialOperation](t, cancelled)
	if cancellation.Type != domain.FinancialOperationCancellation || len(cancellation.Items) != 1 || cancellation.Items[0].OrderLineID == nil || *cancellation.Items[0].OrderLineID != line.ID {
		t.Fatalf("unexpected line cancellation operation: %+v", cancellation)
	}

	manager := f.employee
	managerSession := f.session
	if _, err := f.service.CloseCashSession(f.ctx, app.CloseCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 cashSession.ID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = cashier
	f.session = cashierSession
	if _, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 order.ShiftID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = manager
	f.session = managerSession
	if _, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	operationsBefore := countAPIRows(t, f, "financial_operations")
	itemsBefore := countAPIRows(t, f, "financial_operation_items")
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")

	body := fmt.Sprintf(`{"command_id":"cmd-api-line-refund-over-quantity","node_device_id":"%s","operation_kind":"partial","inventory_disposition":"no_stock_effect","reason":"line refund quantity above remaining must be rejected","items":[{"scope":"order_line","order_line_id":"%s","quantity":2,"amount":%d,"currency":"RUB"}]}`, f.device.ID, line.ID, unitAmount)
	rr := f.postJSON(t, "/api/v1/checks/"+check.ID+"/refunds", body)
	assertSafeConflictAPIError(t, rr, "operation line quantity exceeds remaining line quantity")
	if operations := countAPIRows(t, f, "financial_operations"); operations != operationsBefore {
		t.Fatalf("expected no refund operation write, before=%d after=%d", operationsBefore, operations)
	}
	if items := countAPIRows(t, f, "financial_operation_items"); items != itemsBefore {
		t.Fatalf("expected no refund item write, before=%d after=%d", itemsBefore, items)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no refund outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no refund local event write, before=%d after=%d", eventsBefore, events)
	}
	operations, err := f.repo.ListFinancialOperationsByCheck(f.ctx, check.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || operations[0].ID != cancellation.ID || operations[0].Type != domain.FinancialOperationCancellation {
		t.Fatalf("expected only the original line cancellation operation, got %+v", operations)
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

func TestRecordCancellationLineAmountAboveSelectedQuantityThroughPublicAPIReturnsConflict(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	lines, err := f.repo.ListOrderLines(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Quantity != 2 {
		t.Fatalf("expected one order line with quantity 2, got %+v", lines)
	}
	line := lines[0]
	f.openCashSession(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-line-over-amount-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected precheck 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	paid := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/payments", `{"command_id":"cmd-api-line-over-amount-payment","node_device_id":"`+f.device.ID+`","method":"cash","amount":`+fmt.Sprint(precheck.Total)+`,"currency":"RUB"}`)
	if paid.Code != http.StatusCreated {
		t.Fatalf("expected payment 201, got %d: %s", paid.Code, paid.Body.String())
	}
	payment := decodeAPIResponse[domain.Payment](t, paid)
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	f.useManagerOperator(t)
	operationsBefore := countAPIRows(t, f, "financial_operations")
	itemsBefore := countAPIRows(t, f, "financial_operation_items")
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")

	unitAmount := line.TotalPrice / line.Quantity
	body := fmt.Sprintf(`{"command_id":"cmd-api-line-cancel-over-amount","node_device_id":"%s","operation_kind":"partial","inventory_disposition":"manual_review","reason":"line amount above selected quantity must be rejected","items":[{"scope":"order_line","order_line_id":"%s","quantity":1,"amount":%d,"currency":"RUB"}]}`, f.device.ID, line.ID, unitAmount+1)
	rr := f.postJSON(t, "/api/v1/checks/"+check.ID+"/cancellations", body)
	assertSafeConflictAPIError(t, rr, "operation line amount exceeds selected line amount")
	if operations := countAPIRows(t, f, "financial_operations"); operations != operationsBefore {
		t.Fatalf("expected no cancellation operation write, before=%d after=%d", operationsBefore, operations)
	}
	if items := countAPIRows(t, f, "financial_operation_items"); items != itemsBefore {
		t.Fatalf("expected no cancellation item write, before=%d after=%d", itemsBefore, items)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore {
		t.Fatalf("expected no cancellation outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore {
		t.Fatalf("expected no cancellation local event write, before=%d after=%d", eventsBefore, events)
	}
	operations, err := f.repo.ListFinancialOperationsByCheck(f.ctx, check.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 0 {
		t.Fatalf("expected no financial operations after rejected cancellation, got %+v", operations)
	}
	var paymentStatus, checkStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM payments WHERE id = ?`, payment.ID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM checks WHERE id = ?`, check.ID).Scan(&checkStatus); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != string(domain.PaymentCaptured) || checkStatus != string(domain.CheckPaid) {
		t.Fatalf("rejected cancellation must not mutate finalized docs, payment=%s check=%s", paymentStatus, checkStatus)
	}
}

func TestRecordCancellationThenRefundRemainingThroughPublicAPISucceeds(t *testing.T) {
	f := newAPIFixture(t)
	order := f.createOrderWithLine(t)
	cashSession := f.openCashSession(t)
	issued := f.postJSON(t, "/api/v1/orders/"+order.ID+"/precheck", `{"command_id":"cmd-api-mixed-success-issue","node_device_id":"`+f.device.ID+`"}`)
	if issued.Code != http.StatusCreated {
		t.Fatalf("expected precheck 201, got %d: %s", issued.Code, issued.Body.String())
	}
	precheck := decodeAPIResponse[domain.Precheck](t, issued)
	paid := f.postJSON(t, "/api/v1/prechecks/"+precheck.ID+"/payments", `{"command_id":"cmd-api-mixed-success-payment","node_device_id":"`+f.device.ID+`","method":"cash","amount":`+fmt.Sprint(precheck.Total)+`,"currency":"RUB"}`)
	if paid.Code != http.StatusCreated {
		t.Fatalf("expected payment 201, got %d: %s", paid.Code, paid.Body.String())
	}
	payment := decodeAPIResponse[domain.Payment](t, paid)
	check, err := f.repo.GetCheckByOrder(f.ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}

	cashier := f.employee
	cashierSession := f.session
	f.useManagerOperator(t)
	outboxBeforeCancellation := countAPIRows(t, f, "pos_sync_outbox")
	eventsBeforeCancellation := countAPIRows(t, f, "local_event_log")
	cancellationAmount := check.Total / 2
	cancelBody := fmt.Sprintf(`{"command_id":"cmd-api-mixed-success-cancel","node_device_id":"%s","operation_kind":"partial","inventory_disposition":"manual_review","reason":"partial cancellation before refund","items":[{"scope":"whole_check","amount":%d,"currency":"RUB"}]}`, f.device.ID, cancellationAmount)
	cancelled := f.postJSON(t, "/api/v1/checks/"+check.ID+"/cancellations", cancelBody)
	if cancelled.Code != http.StatusCreated {
		t.Fatalf("expected cancellation 201, got %d: %s", cancelled.Code, cancelled.Body.String())
	}
	cancellation := decodeAPIResponse[domain.FinancialOperation](t, cancelled)
	if cancellation.Type != domain.FinancialOperationCancellation || cancellation.Kind != domain.FinancialOperationPartial || cancellation.Amount != cancellationAmount {
		t.Fatalf("unexpected cancellation operation: %+v", cancellation)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBeforeCancellation+1 {
		t.Fatalf("expected one cancellation outbox write, before=%d after=%d", outboxBeforeCancellation, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBeforeCancellation+1 {
		t.Fatalf("expected one cancellation local event write, before=%d after=%d", eventsBeforeCancellation, events)
	}
	assertAPIFinancialOperationOutboxEnvelope(t, f, "cmd-api-mixed-success-cancel", "CancellationRecorded", cancellation)

	manager := f.employee
	managerSession := f.session
	if _, err := f.service.CloseCashSession(f.ctx, app.CloseCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 cashSession.ID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = cashier
	f.session = cashierSession
	if _, err := f.service.CloseShift(f.ctx, app.CloseShiftCommand{
		CommandMeta:        f.edgeMeta(),
		ID:                 order.ShiftID,
		ClosedByEmployeeID: f.employee.ID,
		ClosingCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	f.employee = manager
	f.session = managerSession
	refundShift, err := f.service.OpenShift(f.ctx, app.OpenShiftCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.OpenCashSession(f.ctx, app.OpenCashSessionCommand{
		CommandMeta:        f.edgeMeta(),
		RestaurantID:       f.restaurant.ID,
		OpenedByEmployeeID: f.employee.ID,
		OpeningCashAmount:  0,
	}); err != nil {
		t.Fatal(err)
	}
	operationsBefore := countAPIRows(t, f, "financial_operations")
	itemsBefore := countAPIRows(t, f, "financial_operation_items")
	outboxBefore := countAPIRows(t, f, "pos_sync_outbox")
	eventsBefore := countAPIRows(t, f, "local_event_log")

	remaining := check.Total - cancellation.Amount
	refundBody := fmt.Sprintf(`{"command_id":"cmd-api-mixed-success-refund","node_device_id":"%s","operation_kind":"full","inventory_disposition":"no_stock_effect","reason":"refund remaining after cancellation","items":[{"scope":"whole_check","amount":%d,"currency":"RUB"}]}`, f.device.ID, remaining)
	refunded := f.postJSON(t, "/api/v1/checks/"+check.ID+"/refunds", refundBody)
	if refunded.Code != http.StatusCreated {
		t.Fatalf("expected refund 201, got %d: %s", refunded.Code, refunded.Body.String())
	}
	refund := decodeAPIResponse[domain.FinancialOperation](t, refunded)
	if refund.Type != domain.FinancialOperationRefund || refund.Kind != domain.FinancialOperationFull || refund.Amount != remaining {
		t.Fatalf("unexpected refund operation: %+v", refund)
	}
	if refund.ShiftID != refundShift.ID || refund.OriginalShiftID != order.ShiftID {
		t.Fatalf("unexpected refund shift boundary: shift_id=%s original_shift_id=%s", refund.ShiftID, refund.OriginalShiftID)
	}
	if operations := countAPIRows(t, f, "financial_operations"); operations != operationsBefore+1 {
		t.Fatalf("expected one refund operation write, before=%d after=%d", operationsBefore, operations)
	}
	if items := countAPIRows(t, f, "financial_operation_items"); items != itemsBefore+1 {
		t.Fatalf("expected one refund item write, before=%d after=%d", itemsBefore, items)
	}
	if outbox := countAPIRows(t, f, "pos_sync_outbox"); outbox != outboxBefore+1 {
		t.Fatalf("expected one refund outbox write, before=%d after=%d", outboxBefore, outbox)
	}
	if events := countAPIRows(t, f, "local_event_log"); events != eventsBefore+1 {
		t.Fatalf("expected one refund local event write, before=%d after=%d", eventsBefore, events)
	}
	assertAPIFinancialOperationOutboxEnvelope(t, f, "cmd-api-mixed-success-refund", "RefundRecorded", refund)
	operations, err := f.repo.ListFinancialOperationsByCheck(f.ctx, check.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 2 || operations[0].ID != cancellation.ID || operations[1].ID != refund.ID {
		t.Fatalf("expected cancellation followed by refund, got %+v", operations)
	}
	var paymentStatus, checkStatus string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM payments WHERE id = ?`, payment.ID).Scan(&paymentStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM checks WHERE id = ?`, check.ID).Scan(&checkStatus); err != nil {
		t.Fatal(err)
	}
	if paymentStatus != string(domain.PaymentCaptured) || checkStatus != string(domain.CheckPaid) {
		t.Fatalf("mixed compensation must not mutate finalized docs, payment=%s check=%s", paymentStatus, checkStatus)
	}
}
