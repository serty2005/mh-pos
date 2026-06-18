package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"license-server/internal/license/api"
	"license-server/internal/license/app"
	"license-server/internal/license/infra/sqlite"

	_ "modernc.org/sqlite"
)

const (
	plaintextCodeMarker = "PAIRING-SECRET-MARKER-1234"
	hashMarker          = "hash-secret-marker"
)

func TestHTTPRegisterAndResolvePairingCodeUsesSQLiteWithoutSecretLeakage(t *testing.T) {
	f := newHTTPFixture(t)
	code := plaintextCodeMarker
	registerBody := `{
		"pairing_code":"` + code + `",
		"pairing_id":"pairing-http-1",
		"instance_id":"cloud-instance-http-1",
		"cloud_url":"https://cloud.example.test",
		"restaurant_id":"restaurant-http-1",
		"expires_at":"2099-05-04T10:30:00Z"
	}`

	registered := f.postJSON(t, "/api/v1/pairing-codes", registerBody)
	assertJSONStatus(t, registered, http.StatusCreated)
	assertAllowedJSONFields(t, registered.Body.Bytes(), "status", "expires_at")
	assertBodyDoesNotContain(t, registered.Body.String(), plaintextCodeMarker, hashMarker, "pairing_code_hash")
	if registered.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON content type, got %q", registered.Header().Get("Content-Type"))
	}

	persisted, err := f.repo.GetByHash(t.Context(), app.Hash(code))
	if err != nil {
		t.Fatalf("expected HTTP register to persist pairing code: %v", err)
	}
	if persisted.PairingID != "pairing-http-1" || persisted.InstanceID != "cloud-instance-http-1" || persisted.CloudURL != "https://cloud.example.test" || persisted.RestaurantID != "restaurant-http-1" {
		t.Fatalf("unexpected persisted pairing code metadata: %+v", persisted)
	}
	if persisted.ConsumedAt != nil {
		t.Fatalf("expected registered pairing code to be active, got consumed_at=%v", persisted.ConsumedAt)
	}

	resolved := f.postJSON(t, "/api/v1/pairing-codes/resolve", `{"pairing_code":"`+code+`"}`)
	assertJSONStatus(t, resolved, http.StatusOK)
	assertAllowedJSONFields(t, resolved.Body.Bytes(), "pairing_id", "cloud_url", "restaurant_id", "expires_at")
	assertBodyContains(t, resolved.Body.String(), "pairing-http-1", "https://cloud.example.test", "restaurant-http-1")
	assertBodyDoesNotContain(t, resolved.Body.String(), plaintextCodeMarker, hashMarker, "pairing_code_hash", "internal_error", "stack", "sqlite", "sql:")

	afterResolve, err := f.repo.GetByHash(t.Context(), app.Hash(code))
	if err != nil {
		t.Fatalf("get pairing code after resolve: %v", err)
	}
	if afterResolve.ConsumedAt != nil {
		t.Fatalf("current resolve contract should not consume pairing code, got consumed_at=%v", afterResolve.ConsumedAt)
	}
	repeated := f.postJSON(t, "/api/v1/pairing-codes/resolve", `{"pairing_code":"`+code+`"}`)
	assertJSONStatus(t, repeated, http.StatusOK)
	assertBodyDoesNotContain(t, repeated.Body.String(), plaintextCodeMarker, hashMarker, "pairing_code_hash")

	assertBodyDoesNotContain(t, f.logs.String(), plaintextCodeMarker, hashMarker, app.Hash(code))
}

func TestHTTPRegisterRejectsInvalidBodiesWithSafeErrors(t *testing.T) {
	f := newHTTPFixture(t)
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed_json", body: `{"pairing_code":`},
		{name: "unknown_field", body: `{"pairing_code":"value","unexpected":"field"}`},
		{name: "missing_required_fields", body: `{"pairing_code":"` + plaintextCodeMarker + `"}`},
		{name: "large_invalid_body", body: strings.Repeat("x", 1<<20+128)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := f.postJSON(t, "/api/v1/pairing-codes", tc.body)
			assertJSONStatus(t, res, http.StatusBadRequest)
			assertSafeErrorEnvelope(t, res, "PAIRING_CODE_INVALID", "errors.pairing.invalid")
			assertBodyDoesNotContain(t, res.Body.String(), plaintextCodeMarker, hashMarker, "internal_error", "sqlite", "sql:", "stack", "panic", "constraint")
		})
	}
}

func TestHTTPResolveRejectsInvalidExpiredAndConsumedCodesSafely(t *testing.T) {
	f := newHTTPFixture(t)
	validFuture := time.Date(2099, 5, 4, 10, 30, 0, 0, time.UTC)
	expiredCode := plaintextCodeMarker + "-expired"
	consumedCode := plaintextCodeMarker + "-consumed"
	if err := f.repo.Save(t.Context(), app.PairingCode{
		PairingCodeHash: app.Hash(expiredCode),
		PairingID:       "pairing-expired",
		InstanceID:      "instance-expired",
		CloudURL:        "https://expired.example.test",
		RestaurantID:    "restaurant-expired",
		ExpiresAt:       time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		CreatedAt:       time.Date(1999, 1, 2, 3, 4, 5, 0, time.UTC),
	}); err != nil {
		t.Fatalf("seed expired pairing code: %v", err)
	}
	if err := f.repo.Save(t.Context(), app.PairingCode{
		PairingCodeHash: app.Hash(consumedCode),
		PairingID:       "pairing-consumed",
		InstanceID:      "instance-consumed",
		CloudURL:        "https://consumed.example.test",
		RestaurantID:    "restaurant-consumed",
		ExpiresAt:       validFuture,
		CreatedAt:       time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("seed consumed pairing code: %v", err)
	}
	if err := f.repo.MarkConsumed(t.Context(), app.Hash(consumedCode), time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("pre-mark consumed pairing code: %v", err)
	}

	tests := []struct {
		name       string
		body       string
		statusCode int
		code       string
		messageKey string
	}{
		{name: "unknown", body: `{"pairing_code":"unknown-code"}`, statusCode: http.StatusBadRequest, code: "PAIRING_CODE_INVALID", messageKey: "errors.pairing.invalid"},
		{name: "malformed_empty", body: `{"pairing_code":""}`, statusCode: http.StatusBadRequest, code: "PAIRING_CODE_INVALID", messageKey: "errors.pairing.invalid"},
		{name: "malformed_whitespace", body: `{"pairing_code":"   "}`, statusCode: http.StatusBadRequest, code: "PAIRING_CODE_INVALID", messageKey: "errors.pairing.invalid"},
		{name: "expired", body: `{"pairing_code":"` + expiredCode + `"}`, statusCode: http.StatusBadRequest, code: "PAIRING_CODE_EXPIRED", messageKey: "errors.pairing.expired"},
		{name: "consumed", body: `{"pairing_code":"` + consumedCode + `"}`, statusCode: http.StatusBadRequest, code: "PAIRING_CODE_INVALID", messageKey: "errors.pairing.invalid"},
		{name: "invalid_json", body: `{"pairing_code":`, statusCode: http.StatusBadRequest, code: "PAIRING_CODE_INVALID", messageKey: "errors.pairing.invalid"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := f.postJSON(t, "/api/v1/pairing-codes/resolve", tc.body)
			assertJSONStatus(t, res, tc.statusCode)
			assertSafeErrorEnvelope(t, res, tc.code, tc.messageKey)
			assertBodyDoesNotContain(t, res.Body.String(), plaintextCodeMarker, hashMarker, "pairing_code_hash", "internal_error", "sqlite", "sql:", "stack", "panic", "constraint", "database")
		})
	}
}

func TestHTTPMethodAndPathSafetyForPairingCodeRoutes(t *testing.T) {
	f := newHTTPFixture(t)
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "get_register_route", method: http.MethodGet, path: "/api/v1/pairing-codes", want: http.StatusMethodNotAllowed},
		{name: "get_resolve_route", method: http.MethodGet, path: "/api/v1/pairing-codes/resolve", want: http.StatusMethodNotAllowed},
		{name: "unknown_path", method: http.MethodPost, path: "/api/v1/pairing-codes/unknown", body: `{}`, want: http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			f.router.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("expected HTTP %d, got %d", tc.want, res.Code)
			}
			assertBodyDoesNotContain(t, res.Body.String(), plaintextCodeMarker, hashMarker, "sqlite", "sql:", "stack", "panic")
		})
	}
}

func TestHTTPAndRepositoryErrorStringsDoNotExposeSecrets(t *testing.T) {
	f := newHTTPFixture(t)
	res := f.postJSON(t, "/api/v1/pairing-codes/resolve", `{"pairing_code":"`+plaintextCodeMarker+`"}`)
	assertJSONStatus(t, res, http.StatusBadRequest)
	assertSafeErrorEnvelope(t, res, "PAIRING_CODE_INVALID", "errors.pairing.invalid")
	assertBodyDoesNotContain(t, res.Body.String(), plaintextCodeMarker, hashMarker, app.Hash(plaintextCodeMarker), "internal_error")

	_, err := f.repo.GetByHash(t.Context(), hashMarker)
	if !errors.Is(err, app.ErrInvalid) {
		t.Fatalf("expected invalid repository error, got %v", err)
	}
	assertErrorDoesNotContain(t, err, plaintextCodeMarker, hashMarker)
	assertBodyDoesNotContain(t, f.logs.String(), plaintextCodeMarker, hashMarker, app.Hash(plaintextCodeMarker))
}

type httpFixture struct {
	db     *sql.DB
	repo   *sqlite.Repository
	router http.Handler
	logs   *bytes.Buffer
}

func newHTTPFixture(t *testing.T) *httpFixture {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "license-http-test.db"))
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewRepository(db)
	if err := repo.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return &httpFixture{
		db:     db,
		repo:   repo,
		router: api.NewRouter(app.NewService(repo)),
		logs:   &logs,
	}
}

func (f *httpFixture) postJSON(t *testing.T, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func assertJSONStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected HTTP %d, got %d: %s", want, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
}

func assertAllowedJSONFields(t *testing.T, body []byte, allowed ...string) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
	allowedSet := map[string]bool{}
	for _, field := range allowed {
		allowedSet[field] = true
	}
	for field := range fields {
		if !allowedSet[field] {
			t.Fatalf("unexpected public response field %q", field)
		}
	}
	for _, field := range allowed {
		if fields[field] == nil {
			t.Fatalf("expected public response field %q", field)
		}
	}
}

func assertSafeErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder, code, messageKey string) {
	t.Helper()
	if got := rec.Header().Get("X-Error-Code"); got != code {
		t.Fatalf("expected X-Error-Code %q, got %q", code, got)
	}
	var envelope struct {
		Error map[string]string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if len(envelope.Error) != 3 {
		t.Fatalf("expected safe error envelope with three fields, got %d", len(envelope.Error))
	}
	if envelope.Error["code"] != code || envelope.Error["message_key"] != messageKey {
		t.Fatalf("unexpected error contract: %+v", envelope.Error)
	}
	if envelope.Error["correlation_id"] == "" {
		t.Fatal("expected correlation_id in error response")
	}
	for field := range envelope.Error {
		switch field {
		case "code", "message_key", "correlation_id":
		default:
			t.Fatalf("unexpected unsafe error field %q", field)
		}
	}
}

func assertBodyContains(t *testing.T, body string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(body, value) {
			t.Fatalf("expected response body to contain %q", value)
		}
	}
}

func assertBodyDoesNotContain(t *testing.T, body string, forbidden ...string) {
	t.Helper()
	lowerBody := strings.ToLower(body)
	for _, value := range forbidden {
		if value == "" {
			continue
		}
		if strings.Contains(lowerBody, strings.ToLower(value)) {
			t.Fatal("body or log output leaked forbidden marker")
		}
	}
}

func assertErrorDoesNotContain(t *testing.T, err error, forbidden ...string) {
	t.Helper()
	if err == nil {
		return
	}
	lowerError := strings.ToLower(err.Error())
	for _, value := range forbidden {
		if strings.Contains(lowerError, strings.ToLower(value)) {
			t.Fatal("error string leaked forbidden marker")
		}
	}
}
