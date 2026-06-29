package sqlite_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"license-server/internal/license/app"
	"license-server/internal/license/infra/sqlite"

	_ "modernc.org/sqlite"
)

func TestMigrateCreatesPairingCodeSchemaAndIsIdempotent(t *testing.T) {
	ctx := t.Context()
	db := openTestDB(t)
	repo := sqlite.NewRepository(db)

	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("migrate pairing code schema: %v", err)
	}
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("repeat migrate pairing code schema: %v", err)
	}

	columns := pairingCodeColumns(t, db)
	for _, name := range []string{
		"pairing_code_hash",
		"pairing_id",
		"instance_id",
		"cloud_url",
		"restaurant_id",
		"node_device_id",
		"credentials_json",
		"expires_at",
		"consumed_at",
		"created_at",
	} {
		if !columns[name] {
			t.Fatalf("expected pairing_codes.%s to exist", name)
		}
	}
	if !hasUniquePairingCodeIndex(t, db) {
		t.Fatal("expected pairing_codes to have a unique primary-key index")
	}
}

func TestSavePersistsPairingCodeAndUpsertsDuplicateHash(t *testing.T) {
	ctx := t.Context()
	db, repo := migratedTestRepo(t)
	first := testPairingCode("sha256:repo-save-1")
	first.ConsumedAt = ptrTime(time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC))

	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("save pairing code: %v", err)
	}
	assertStoredPairingCode(t, db, first, false)

	updated := app.PairingCode{
		PairingCodeHash: first.PairingCodeHash,
		PairingID:       "pairing-updated",
		InstanceID:      "instance-updated",
		CloudURL:        "https://cloud-updated.example.test",
		RestaurantID:    "restaurant-updated",
		NodeDeviceID:    "node-updated",
		ExpiresAt:       time.Date(2026, 5, 6, 13, 14, 15, 0, time.UTC),
		CreatedAt:       time.Date(2026, 5, 6, 10, 9, 8, 0, time.UTC),
	}
	if err := repo.Save(ctx, updated); err != nil {
		t.Fatalf("upsert duplicate pairing code hash: %v", err)
	}

	var rows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM pairing_codes WHERE pairing_code_hash = ?`, first.PairingCodeHash).Scan(&rows); err != nil {
		t.Fatalf("count pairing code rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected duplicate save to keep one row, got %d", rows)
	}
	assertStoredPairingCode(t, db, updated, false)
	assertPlaintextNotPersisted(t, db, "PAIRING-SECRET-MARKER-1234")
}

func TestGetByHashReturnsRoundTripStateAndSafeMissingError(t *testing.T) {
	ctx := t.Context()
	_, repo := migratedTestRepo(t)
	consumedAt := time.Date(2026, 5, 4, 15, 16, 17, 0, time.UTC)
	item := testPairingCode("sha256:repo-get-1")
	item.ExpiresAt = time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	item.ConsumedAt = &consumedAt
	if err := repo.Save(ctx, item); err != nil {
		t.Fatalf("save pairing code: %v", err)
	}
	if err := repo.MarkConsumed(ctx, item.PairingCodeHash, consumedAt); err != nil {
		t.Fatalf("mark pairing code consumed: %v", err)
	}

	got, err := repo.GetByHash(ctx, item.PairingCodeHash)
	if err != nil {
		t.Fatalf("get pairing code by hash: %v", err)
	}
	assertPairingCodeEqual(t, item, got, true)

	const forbiddenHashMarker = "hash-secret-marker"
	_, err = repo.GetByHash(ctx, forbiddenHashMarker)
	if !errors.Is(err, app.ErrInvalid) {
		t.Fatalf("expected app invalid error for missing hash, got %v", err)
	}
	assertErrorDoesNotContain(t, err, forbiddenHashMarker)
	assertErrorDoesNotContain(t, err, "sql:")
	assertErrorDoesNotContain(t, err, "sqlite")
}

func TestMarkConsumedPersistsTimestampIsRepeatableAndDoesNotTouchOtherRows(t *testing.T) {
	ctx := t.Context()
	_, repo := migratedTestRepo(t)
	first := testPairingCode("sha256:repo-consume-1")
	second := testPairingCode("sha256:repo-consume-2")
	second.PairingID = "pairing-other"
	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("save first pairing code: %v", err)
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("save second pairing code: %v", err)
	}

	consumedAt := time.Date(2026, 5, 7, 8, 9, 10, 0, time.UTC)
	if err := repo.MarkConsumed(ctx, first.PairingCodeHash, consumedAt); err != nil {
		t.Fatalf("mark pairing code consumed: %v", err)
	}
	got, err := repo.GetByHash(ctx, first.PairingCodeHash)
	if err != nil {
		t.Fatalf("get consumed pairing code: %v", err)
	}
	if got.ConsumedAt == nil || !got.ConsumedAt.Equal(consumedAt) {
		t.Fatalf("expected consumed timestamp to round-trip, got %v", got.ConsumedAt)
	}

	repeatConsumedAt := consumedAt.Add(time.Hour)
	if err := repo.MarkConsumed(ctx, first.PairingCodeHash, repeatConsumedAt); err != nil {
		t.Fatalf("repeat mark consumed should remain successful: %v", err)
	}
	got, err = repo.GetByHash(ctx, first.PairingCodeHash)
	if err != nil {
		t.Fatalf("get repeatedly consumed pairing code: %v", err)
	}
	if got.ConsumedAt == nil || !got.ConsumedAt.Equal(repeatConsumedAt) {
		t.Fatalf("expected repeated mark to update consumed timestamp, got %v", got.ConsumedAt)
	}

	other, err := repo.GetByHash(ctx, second.PairingCodeHash)
	if err != nil {
		t.Fatalf("get unrelated pairing code: %v", err)
	}
	if other.ConsumedAt != nil {
		t.Fatalf("expected unrelated pairing code to stay active, got consumed_at=%v", other.ConsumedAt)
	}

	const missingHashMarker = "hash-secret-marker"
	if err := repo.MarkConsumed(ctx, missingHashMarker, consumedAt); err != nil {
		t.Fatalf("current missing-hash contract is no-op success, got %v", err)
	}
}

func TestEntitlementSnapshotsRoundTripAndList(t *testing.T) {
	ctx := t.Context()
	_, repo := migratedTestRepo(t)
	first := app.EntitlementSnapshot{
		TenantID:     "tenant-a",
		ServerID:     "cloud-a",
		Version:      1,
		Status:       "active",
		Entitlements: map[string]bool{"table-mode": true, "kitchen-space": false},
		IssuedAt:     time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC),
		ExpiresAt:    time.Date(2099, 6, 20, 10, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 6, 20, 10, 1, 0, 0, time.UTC),
	}
	second := app.EntitlementSnapshot{
		TenantID:     "tenant-b",
		ServerID:     "edge-b",
		Version:      2,
		Status:       "revoked",
		Entitlements: map[string]bool{"warehouse-mode": false},
		IssuedAt:     time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC),
		ExpiresAt:    time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 6, 21, 10, 1, 0, 0, time.UTC),
	}
	if err := repo.SaveEntitlements(ctx, first); err != nil {
		t.Fatalf("save first entitlement snapshot: %v", err)
	}
	if err := repo.SaveEntitlements(ctx, second); err != nil {
		t.Fatalf("save second entitlement snapshot: %v", err)
	}

	got, err := repo.GetEntitlements(ctx, first.TenantID, first.ServerID)
	if err != nil {
		t.Fatalf("get entitlement snapshot: %v", err)
	}
	if got.Version != first.Version || got.Status != first.Status || !got.Entitlements["table-mode"] || got.Entitlements["kitchen-space"] {
		t.Fatalf("unexpected entitlement round-trip: %+v", got)
	}

	list, err := repo.ListEntitlements(ctx)
	if err != nil {
		t.Fatalf("list entitlement snapshots: %v", err)
	}
	if len(list) != 2 || list[0].TenantID != first.TenantID || list[1].TenantID != second.TenantID {
		t.Fatalf("unexpected entitlement list: %+v", list)
	}

	servers, err := repo.ListConnectedServers(ctx)
	if err != nil {
		t.Fatalf("list connected servers: %v", err)
	}
	if len(servers) != 2 || servers[0].Snapshot == nil || servers[1].Snapshot == nil {
		t.Fatalf("expected entitlement saves to appear in connected server list: %+v", servers)
	}
}

func TestAdminUsersSessionsAndConnectedServersRoundTrip(t *testing.T) {
	ctx := t.Context()
	_, repo := migratedTestRepo(t)
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	user := app.AdminUser{
		Username:     "admin",
		PasswordHash: "hash",
		Salt:         "salt",
		Iterations:   210000,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.SaveAdminUser(ctx, user); err != nil {
		t.Fatalf("save admin user: %v", err)
	}
	gotUser, err := repo.GetAdminUser(ctx, "admin")
	if err != nil || gotUser.PasswordHash != user.PasswordHash || gotUser.Iterations != user.Iterations {
		t.Fatalf("unexpected admin user: %+v err=%v", gotUser, err)
	}
	session := app.AdminSession{TokenHash: "sha256:session", Username: "admin", ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := repo.SaveAdminSession(ctx, session); err != nil {
		t.Fatalf("save admin session: %v", err)
	}
	gotSession, err := repo.GetAdminSession(ctx, session.TokenHash)
	if err != nil || gotSession.Username != "admin" {
		t.Fatalf("unexpected admin session: %+v err=%v", gotSession, err)
	}
	if err := repo.DeleteAdminSession(ctx, session.TokenHash); err != nil {
		t.Fatalf("delete admin session: %v", err)
	}
	if _, err := repo.GetAdminSession(ctx, session.TokenHash); !errors.Is(err, app.ErrAdminAuth) {
		t.Fatalf("expected deleted session to be invalid, got %v", err)
	}
	if err := repo.SaveConnectedServer(ctx, app.ConnectedServer{TenantID: "tenant-a", ServerID: "edge-a", FirstSeenAt: now, LastSeenAt: now}); err != nil {
		t.Fatalf("save connected server: %v", err)
	}
	servers, err := repo.ListConnectedServers(ctx)
	if err != nil || len(servers) != 1 || servers[0].TenantID != "tenant-a" || servers[0].Snapshot != nil {
		t.Fatalf("unexpected connected server list: %+v err=%v", servers, err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "license-test.db"))
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func migratedTestRepo(t *testing.T) (*sql.DB, *sqlite.Repository) {
	t.Helper()
	ctx := t.Context()
	db := openTestDB(t)
	repo := sqlite.NewRepository(db)
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("migrate test repository: %v", err)
	}
	return db, repo
}

func testPairingCode(hash string) app.PairingCode {
	return app.PairingCode{
		PairingCodeHash: hash,
		PairingID:       "pairing-1",
		InstanceID:      "cloud-instance-1",
		CloudURL:        "https://cloud.example.test",
		RestaurantID:    "restaurant-1",
		NodeDeviceID:    "node-1",
		ExpiresAt:       time.Date(2026, 5, 5, 12, 30, 0, 0, time.UTC),
		CreatedAt:       time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC),
	}
}

func pairingCodeColumns(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), `PRAGMA table_info(pairing_codes)`)
	if err != nil {
		t.Fatalf("read pairing_codes table info: %v", err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan pairing_codes table info: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pairing_codes table info: %v", err)
	}
	return columns
}

func hasUniquePairingCodeIndex(t *testing.T, db *sql.DB) bool {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), `PRAGMA index_list(pairing_codes)`)
	if err != nil {
		t.Fatalf("read pairing_codes index list: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var seq, unique, partial int
		var name, origin string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan pairing_codes index list: %v", err)
		}
		if unique == 1 && origin == "pk" {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pairing_codes index list: %v", err)
	}
	return false
}

func assertStoredPairingCode(t *testing.T, db *sql.DB, want app.PairingCode, wantConsumed bool) {
	t.Helper()
	var got app.PairingCode
	var expiresAt, createdAt string
	var consumedAt sql.NullString
	var credentials string
	if err := db.QueryRowContext(t.Context(), `
SELECT pairing_code_hash,pairing_id,instance_id,cloud_url,restaurant_id,node_device_id,credentials_json,expires_at,consumed_at,created_at
FROM pairing_codes
WHERE pairing_code_hash = ?`, want.PairingCodeHash).
		Scan(&got.PairingCodeHash, &got.PairingID, &got.InstanceID, &got.CloudURL, &got.RestaurantID, &got.NodeDeviceID, &credentials, &expiresAt, &consumedAt, &createdAt); err != nil {
		t.Fatalf("read stored pairing code: %v", err)
	}
	got.ExpiresAt = mustParseTime(t, expiresAt)
	got.CreatedAt = mustParseTime(t, createdAt)
	if consumedAt.Valid {
		parsed := mustParseTime(t, consumedAt.String)
		got.ConsumedAt = &parsed
	}
	assertPairingCodeEqual(t, want, got, wantConsumed)
	if credentials != "{}" {
		t.Fatalf("expected credentials_json to stay empty JSON object, got %q", credentials)
	}
}

func assertPairingCodeEqual(t *testing.T, want, got app.PairingCode, compareConsumed bool) {
	t.Helper()
	if got.PairingCodeHash != want.PairingCodeHash ||
		got.PairingID != want.PairingID ||
		got.InstanceID != want.InstanceID ||
		got.CloudURL != want.CloudURL ||
		got.RestaurantID != want.RestaurantID ||
		got.NodeDeviceID != want.NodeDeviceID ||
		!got.ExpiresAt.Equal(want.ExpiresAt) ||
		!got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("unexpected pairing code round-trip: got=%+v want=%+v", got, want)
	}
	if !compareConsumed {
		if got.ConsumedAt != nil {
			t.Fatalf("expected consumed_at to be nil, got %v", got.ConsumedAt)
		}
		return
	}
	if (got.ConsumedAt == nil) != (want.ConsumedAt == nil) {
		t.Fatalf("unexpected consumed_at presence: got=%v want=%v", got.ConsumedAt, want.ConsumedAt)
	}
	if got.ConsumedAt != nil && !got.ConsumedAt.Equal(*want.ConsumedAt) {
		t.Fatalf("unexpected consumed_at: got=%v want=%v", got.ConsumedAt, want.ConsumedAt)
	}
}

func assertPlaintextNotPersisted(t *testing.T, db *sql.DB, forbidden string) {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), `SELECT pairing_code_hash,pairing_id,instance_id,cloud_url,restaurant_id,node_device_id,credentials_json FROM pairing_codes`)
	if err != nil {
		t.Fatalf("scan persisted pairing code fields: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var values [7]string
		if err := rows.Scan(&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6]); err != nil {
			t.Fatalf("scan persisted pairing code row: %v", err)
		}
		for _, value := range values {
			if strings.Contains(value, forbidden) {
				t.Fatal("plaintext pairing code marker leaked into repository row")
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate persisted pairing code rows: %v", err)
	}
}

func assertErrorDoesNotContain(t *testing.T, err error, forbidden string) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), strings.ToLower(forbidden)) {
		t.Fatal("error string leaked forbidden marker")
	}
}

func mustParseTime(t *testing.T, raw string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Fatalf("parse stored timestamp: %v", err)
	}
	return v
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
