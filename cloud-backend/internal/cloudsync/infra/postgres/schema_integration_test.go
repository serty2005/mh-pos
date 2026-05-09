package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	platformpg "cloud-backend/internal/platform/postgres"
)

func TestActualCloudMigrationsInitializeEmptyPostgres(t *testing.T) {
	pool := openPostgresIntegrationPool(t)
	ctx := t.Context()
	resetPublicSchema(t, ctx, pool)
	t.Cleanup(func() { resetPublicSchema(t, context.Background(), pool) })

	if err := platformpg.MigrateDirWithPolicy(ctx, pool, actualMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: RequiredSchema(),
	}); err != nil {
		t.Fatalf("empty postgres migration failed: %v", err)
	}
	for _, req := range RequiredSchema() {
		assertTableExists(t, ctx, pool, req.Table)
	}
	assertRecordedMigrationCount(t, ctx, pool, 4)

	if err := platformpg.MigrateDirWithPolicy(ctx, pool, actualMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: RequiredSchema(),
	}); err != nil {
		t.Fatalf("second empty postgres migration failed: %v", err)
	}
	assertRecordedMigrationCount(t, ctx, pool, 4)
}

func TestActualCloudMigrationsRepairOldPostgresMissingRuntimeTable(t *testing.T) {
	pool := openPostgresIntegrationPool(t)
	ctx := t.Context()
	resetPublicSchema(t, ctx, pool)
	t.Cleanup(func() { resetPublicSchema(t, context.Background(), pool) })

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
CREATE TABLE cloud_edge_event_receipts (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  command_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  edge_event_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  envelope_version TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX cloud_edge_event_receipts_edge_event_key
  ON cloud_edge_event_receipts(restaurant_id, device_id, edge_event_id);
CREATE INDEX cloud_edge_event_receipts_event_type_received_at
  ON cloud_edge_event_receipts(event_type, cloud_received_at);
CREATE TABLE cloud_edge_event_raw_payloads (
  receipt_id TEXT PRIMARY KEY REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  raw_payload JSONB NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE cloud_operational_events (
  id TEXT PRIMARY KEY,
  receipt_id TEXT NOT NULL UNIQUE REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  idempotency_key TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL,
  device_id TEXT NOT NULL,
  command_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  edge_event_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  envelope_version TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL,
  replay_status TEXT NOT NULL DEFAULT 'accepted',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX cloud_operational_events_edge_event_key
  ON cloud_operational_events(restaurant_id, device_id, edge_event_id);
CREATE INDEX cloud_operational_events_type_received_at
  ON cloud_operational_events(event_type, cloud_received_at);
CREATE INDEX cloud_operational_events_restaurant_sequence
  ON cloud_operational_events(restaurant_id, device_id, occurred_at, event_id);
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
CREATE TABLE cloud_master_data_packages (
  stream_name TEXT NOT NULL,
  node_device_id TEXT NOT NULL DEFAULT '',
  restaurant_id TEXT,
  sync_mode TEXT NOT NULL,
  cloud_version BIGINT NOT NULL,
  checkpoint_token TEXT,
  cloud_updated_at TIMESTAMPTZ,
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (stream_name, node_device_id)
);
CREATE INDEX cloud_master_data_packages_stream_updated
  ON cloud_master_data_packages(stream_name, updated_at DESC);
CREATE TABLE cloud_currency_reference (
  currency_code INTEGER PRIMARY KEY,
  currency_alpha_code TEXT NOT NULL UNIQUE,
  minor_unit SMALLINT NOT NULL,
  currency_iso_name TEXT NOT NULL,
  currency_symbol TEXT NOT NULL,
  curr_basic_name TEXT NOT NULL,
  curr_add_name TEXT NOT NULL,
  show_add BOOLEAN NOT NULL DEFAULT TRUE,
  show_currency_basic_name BOOLEAN NOT NULL DEFAULT TRUE,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX cloud_currency_reference_alpha_code_idx
  ON cloud_currency_reference(currency_alpha_code);
`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO schema_migrations(version, checksum_sha256, status)
VALUES ($1, $2, 'applied'), ($3, $4, 'applied');
`,
		"001_sync_receiver.sql", checksumFile(t, filepath.Join(actualMigrationsDir(), "001_sync_receiver.sql")),
		"002_projection_event_type_stats.sql", checksumFile(t, filepath.Join(actualMigrationsDir(), "002_projection_event_type_stats.sql")),
	); err != nil {
		t.Fatal(err)
	}

	if err := platformpg.MigrateDirWithPolicy(ctx, pool, actualMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: RequiredSchema(),
	}); err != nil {
		t.Fatalf("old postgres repair migration failed: %v", err)
	}
	assertTableExists(t, ctx, pool, "cloud_projection_shift_finance")
	assertRecordedMigrationCount(t, ctx, pool, 4)

	if err := platformpg.MigrateDirWithPolicy(ctx, pool, actualMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: RequiredSchema(),
	}); err != nil {
		t.Fatalf("second old postgres repair migration failed: %v", err)
	}
	assertRecordedMigrationCount(t, ctx, pool, 4)
}

func actualMigrationsDir() string {
	return filepath.Join("..", "..", "..", "..", "migrations", "postgres")
}

func openPostgresIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("CLOUD_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("CLOUD_POSTGRES_TEST_DSN is not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	lockPostgresIntegration(t, t.Context(), pool)
	return pool
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

func assertRecordedMigrationCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE status = 'applied'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expected %d applied migrations, got %d", want, got)
	}
}

func checksumFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
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
