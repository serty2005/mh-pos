package postgres

import (
	"context"
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
	assertRecordedMigrationCount(t, ctx, pool, 1)

	if err := platformpg.MigrateDirWithPolicy(ctx, pool, actualMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: RequiredSchema(),
	}); err != nil {
		t.Fatalf("second empty postgres migration failed: %v", err)
	}
	assertRecordedMigrationCount(t, ctx, pool, 1)
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
