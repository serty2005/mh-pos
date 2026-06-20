package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCompareModuleVersion(t *testing.T) {
	result, err := compareModuleVersion("0.1.0", "0.2.0")
	if err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	if result >= 0 {
		t.Fatalf("expected 0.1.0 < 0.2.0, got %d", result)
	}
	if _, err := compareModuleVersion("broken", "0.2.0"); err == nil {
		t.Fatal("expected invalid semantic version to fail")
	}
}

func TestShouldUpgradeVersion(t *testing.T) {
	needsUpgrade, err := shouldUpgradeVersion("0.1.0", "0.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !needsUpgrade {
		t.Fatal("expected upgrade when runtime version is lower")
	}
	needsUpgrade, err = shouldUpgradeVersion("0.1.1", "0.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if needsUpgrade {
		t.Fatal("expected no upgrade when versions are equal")
	}
	if _, err := shouldUpgradeVersion("0.2.0", "0.1.0"); err == nil {
		t.Fatal("expected newer database version to fail fast")
	}
}

func TestSanitizeFilenameToken(t *testing.T) {
	if got := sanitizeFilenameToken(" cloud backend / 0.1.0 "); got != "cloud_backend___0.1.0" {
		t.Fatalf("unexpected sanitized token %q", got)
	}
}

func TestReadMigrationFilesSortedAndChecksummed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "002_second.sql"), []byte("SELECT 2;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "001_first.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := readMigrationFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two sql migrations, got %d", len(files))
	}
	if files[0].Name != "001_first.sql" || files[1].Name != "002_second.sql" {
		t.Fatalf("expected lexicographic order, got %s then %s", files[0].Name, files[1].Name)
	}
	for _, file := range files {
		if len(file.ChecksumSHA256) != 64 {
			t.Fatalf("expected sha256 checksum for %s, got %q", file.Name, file.ChecksumSHA256)
		}
	}
}

func TestPendingMigrationsReappliesLegacyMarkerWithoutChecksumColumn(t *testing.T) {
	files := []migrationFile{{Name: "001_sync_receiver.sql", ChecksumSHA256: strings.Repeat("a", 64)}}
	pending, err := pendingMigrations(context.Background(), postgresCatalogStub{
		hasChecksumColumn: false,
		legacyMarkers:     map[string]bool{"001_sync_receiver.sql": true},
	}, files, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Name != "001_sync_receiver.sql" {
		t.Fatalf("expected legacy marker to re-run canonical migration, got %+v", pending)
	}
}

func TestPendingMigrationsReappliesLegacyMarkerWithEmptyChecksum(t *testing.T) {
	files := []migrationFile{{Name: "001_sync_receiver.sql", ChecksumSHA256: strings.Repeat("b", 64)}}
	pending, err := pendingMigrations(context.Background(), postgresCatalogStub{
		hasChecksumColumn: true,
		checksums:         map[string]string{"001_sync_receiver.sql": ""},
	}, files, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Name != "001_sync_receiver.sql" {
		t.Fatalf("expected empty checksum marker to re-run canonical migration, got %+v", pending)
	}
}

func TestPendingMigrationsSkipsAppliedChecksum(t *testing.T) {
	checksum := strings.Repeat("c", 64)
	files := []migrationFile{{Name: "001_sync_receiver.sql", ChecksumSHA256: checksum}}
	pending, err := pendingMigrations(context.Background(), postgresCatalogStub{
		hasChecksumColumn: true,
		checksums:         map[string]string{"001_sync_receiver.sql": checksum},
	}, files, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected applied checksum to stay idempotent, got %+v", pending)
	}
}

func TestCloudMigrationDirUsesOrderedManagedFiles(t *testing.T) {
	files, err := readMigrationFiles(filepath.Join("..", "..", "..", "migrations", "postgres"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "001_init.sql" {
		t.Fatalf("expected single managed postgres baseline migration file, got %+v", files)
	}
	baseBody := string(files[0].Body)
	for _, required := range []string{
		"cloud_currency_reference",
		"cloud_currency_reference_alpha_code_idx",
		"cloud_projection_event_type_stats",
		"PRIMARY KEY (restaurant_id, device_id, event_type)",
		"cloud_projection_shift_finance",
		"cloud_master_data_packages",
		"cloud_employees",
		"cloud_employee_restaurant_memberships",
		"cloud_catalog_items",
		"cloud_menu_items",
		"cloud_master_data_publications",
		"cloud_restaurants",
		"cloud_catalog_items_active_sku",
		"cloud_version",
		"cloud_halls",
		"cloud_tables",
		"cloud_edge_nodes",
		"cloud_unassigned_edge_nodes",
		"cloud_pairing_codes",
		"PaymentRefunded",
		"CheckRefunded",
		"CancellationRecorded",
		"RefundRecorded",
		"pricing_policy",
		"payments_refunded_count",
		"cloud_catalog_folders",
		"cloud_catalog_tags",
		"cloud_services",
		"cloud_modifier_group_bindings",
		"cloud_pricing_policies",
		"price_minor",
		"inventory_stock_balances",
		"inventory_stock_balances_restaurant_warehouse_item",
	} {
		if !strings.Contains(baseBody, required) {
			t.Fatalf("expected canonical postgres baseline to manage %s", required)
		}
	}
	if !strings.Contains(baseBody, "kind IN ('dish','good','semi_finished','service')") {
		t.Fatalf("expected first-launch catalog item kind check to use canonical catalog v2 enum")
	}
}

func TestMigrateDirWithPolicyRepairsLegacyPostgresRuntimeSchema(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CLOUD_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("CLOUD_POSTGRES_TEST_DSN is not set")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	lockPostgresIntegration(t, ctx, pool)
	resetPublicSchema(t, ctx, pool)
	t.Cleanup(func() { resetPublicSchema(t, context.Background(), pool) })

	migrationsDir := t.TempDir()
	migration001 := `
CREATE TABLE IF NOT EXISTS cloud_edge_event_receipts (id TEXT PRIMARY KEY);
`
	migration002 := `
CREATE TABLE IF NOT EXISTS cloud_projection_event_type_stats (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  event_count BIGINT NOT NULL CHECK (event_count >= 0),
  first_occurred_at TIMESTAMPTZ NOT NULL,
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, event_type)
);
`
	migration003 := `
CREATE TABLE IF NOT EXISTS migration_apply_log (
  id BIGSERIAL PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO migration_apply_log DEFAULT VALUES;
CREATE TABLE IF NOT EXISTS cloud_projection_event_type_stats (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  event_count BIGINT NOT NULL CHECK (event_count >= 0),
  first_occurred_at TIMESTAMPTZ NOT NULL,
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, event_type)
);
CREATE TABLE IF NOT EXISTS cloud_projection_shift_finance (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  shift_id TEXT NOT NULL CHECK (shift_id <> ''),
  payments_captured_count BIGINT NOT NULL DEFAULT 0 CHECK (payments_captured_count >= 0),
  payments_captured_total BIGINT NOT NULL DEFAULT 0,
  checks_created_count BIGINT NOT NULL DEFAULT 0 CHECK (checks_created_count >= 0),
  checks_total_amount BIGINT NOT NULL DEFAULT 0,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, shift_id)
);
`
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_sync_receiver.sql"), []byte(migration001), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "002_projection_event_type_stats.sql"), []byte(migration002), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "003_runtime_schema_repair.sql"), []byte(migration003), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := readMigrationFiles(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
CREATE TABLE schema_migrations (
  version TEXT PRIMARY KEY,
  checksum_sha256 TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'applied',
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE db_runtime_versions (
  module_name TEXT PRIMARY KEY,
  module_version TEXT NOT NULL,
  schema_version TEXT NOT NULL DEFAULT '',
  checksum_sha256 TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'applied',
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO db_runtime_versions(module_name,module_version)
VALUES ('cloud-backend','0.1.0');
CREATE TABLE cloud_edge_event_receipts (id TEXT PRIMARY KEY);
CREATE TABLE cloud_projection_event_type_stats (
  restaurant_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  event_count BIGINT NOT NULL,
  first_occurred_at TIMESTAMPTZ NOT NULL,
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  last_event_id TEXT NOT NULL,
  last_command_id TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, event_type)
);
`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO schema_migrations(version, checksum_sha256, status)
VALUES ($1, $2, 'applied'), ($3, $4, 'applied');
`, files[0].Name, files[0].ChecksumSHA256, files[1].Name, files[1].ChecksumSHA256); err != nil {
		t.Fatal(err)
	}
	requirements := []SchemaRequirement{
		{
			Table:         "cloud_projection_event_type_stats",
			RequiredBy:    "cloudsync postgres repository applyEventProjections event type stats upsert",
			MigrationFile: "002_projection_event_type_stats.sql, 003_runtime_schema_repair.sql",
			Columns: []string{
				"restaurant_id", "device_id", "event_type", "event_count", "first_occurred_at", "last_occurred_at",
				"last_cloud_received_at", "last_event_id", "last_command_id", "updated_at",
			},
		},
		{
			Table:         "cloud_projection_shift_finance",
			RequiredBy:    "cloudsync postgres repository applyEventProjections shift finance upsert",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql",
			Columns: []string{
				"restaurant_id", "device_id", "shift_id", "payments_captured_count", "payments_captured_total",
				"checks_created_count", "checks_total_amount", "last_event_id", "last_command_id", "last_occurred_at",
				"last_cloud_received_at", "updated_at",
			},
		},
	}

	if err := MigrateDirWithPolicy(ctx, pool, migrationsDir, MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: requirements,
	}); err != nil {
		t.Fatalf("legacy postgres migrate failed: %v", err)
	}
	assertTableExists(t, ctx, pool, "cloud_projection_event_type_stats")
	assertTableExists(t, ctx, pool, "cloud_projection_shift_finance")
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM schema_migrations WHERE version = '003_runtime_schema_repair.sql'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "applied" {
		t.Fatalf("expected migration history status applied, got %s", status)
	}

	if err := MigrateDirWithPolicy(ctx, pool, migrationsDir, MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: requirements,
	}); err != nil {
		t.Fatalf("second postgres migrate failed: %v", err)
	}
	var appliedCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM migration_apply_log`).Scan(&appliedCount); err != nil {
		t.Fatal(err)
	}
	if appliedCount != 1 {
		t.Fatalf("expected second startup to skip already recorded migration, got %d applies", appliedCount)
	}
}

func TestVerifySchemaMissingTableReturnsStructuredError(t *testing.T) {
	err := VerifySchema(context.Background(), postgresCatalogStub{
		tables: map[string]bool{"schema_migrations": true},
	}, []SchemaRequirement{{
		Table:         "cloud_projection_event_type_stats",
		RequiredBy:    "cloudsync postgres repository applyEventProjections event type stats upsert",
		MigrationFile: "002_projection_event_type_stats.sql",
	}})
	var verificationErr *SchemaVerificationError
	if !errors.As(err, &verificationErr) {
		t.Fatalf("expected structured schema verification error, got %T %v", err, err)
	}
	if verificationErr.Table != "cloud_projection_event_type_stats" || verificationErr.MigrationFile != "002_projection_event_type_stats.sql" {
		t.Fatalf("unexpected verification error details: %+v", verificationErr)
	}
	if !strings.Contains(err.Error(), "missing table cloud_projection_event_type_stats") {
		t.Fatalf("expected clear missing table message, got %q", err.Error())
	}
}

func resetPublicSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset public schema: %v", err)
	}
}

func assertTableExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected table %s to exist", table)
	}
}

func lockPostgresIntegration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(72905101)`); err != nil {
		t.Fatalf("lock postgres integration db: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `SELECT pg_advisory_unlock(72905101)`); err != nil {
			t.Logf("unlock postgres integration db: %v", err)
		}
	})
}

type postgresCatalogStub struct {
	hasChecksumColumn bool
	legacyMarkers     map[string]bool
	checksums         map[string]string
	tables            map[string]bool
	columns           map[string]bool
	indexes           map[string]bool
}

func (s postgresCatalogStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (s postgresCatalogStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (s postgresCatalogStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (s postgresCatalogStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	switch {
	case strings.Contains(query, "to_regclass"):
		if len(args) == 1 {
			table := strings.TrimPrefix(fmt.Sprint(args[0]), "public.")
			return stubRow{values: []any{s.tables[table]}}
		}
		return stubRow{values: []any{false}}
	case strings.Contains(query, "information_schema.columns"):
		if len(args) == 2 && args[0] == "schema_migrations" && args[1] == "checksum_sha256" && s.hasChecksumColumn {
			return stubRow{values: []any{1}}
		}
		if len(args) == 2 && s.columns[fmt.Sprintf("%s.%s", args[0], args[1])] {
			return stubRow{values: []any{1}}
		}
		return stubRow{values: []any{0}}
	case strings.Contains(query, "pg_indexes"):
		if len(args) == 2 && s.indexes[fmt.Sprintf("%s.%s", args[0], args[1])] {
			return stubRow{values: []any{1}}
		}
		return stubRow{values: []any{0}}
	case strings.Contains(query, "SELECT COUNT(1) FROM schema_migrations WHERE version"):
		if len(args) == 1 && s.legacyMarkers[fmt.Sprint(args[0])] {
			return stubRow{values: []any{1}}
		}
		return stubRow{values: []any{0}}
	case strings.Contains(query, "SELECT checksum_sha256 FROM schema_migrations WHERE version"):
		if len(args) == 1 {
			if checksum, ok := s.checksums[fmt.Sprint(args[0])]; ok {
				return stubRow{values: []any{checksum}}
			}
		}
		return stubRow{err: pgx.ErrNoRows}
	default:
		return stubRow{err: fmt.Errorf("unexpected query: %s", query)}
	}
}

type stubRow struct {
	values []any
	err    error
}

func (r stubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan destination count %d does not match values %d", len(dest), len(r.values))
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *int:
			v, ok := r.values[i].(int)
			if !ok {
				return fmt.Errorf("value %d is not int", i)
			}
			*d = v
		case *string:
			v, ok := r.values[i].(string)
			if !ok {
				return fmt.Errorf("value %d is not string", i)
			}
			*d = v
		case *bool:
			v, ok := r.values[i].(bool)
			if !ok {
				return fmt.Errorf("value %d is not bool", i)
			}
			*d = v
		default:
			return fmt.Errorf("unsupported scan destination %T", d)
		}
	}
	return nil
}
