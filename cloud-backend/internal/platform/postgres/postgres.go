package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationOptions задает политику версионирования и backup перед обновлением схемы БД.
type MigrationOptions struct {
	ModuleName    string
	ModuleVersion string
	BackupDir     string
}

func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func MigrateDir(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	return MigrateDirWithPolicy(ctx, pool, dir, MigrationOptions{})
}

// MigrateDirWithPolicy применяет миграции и обновляет module version contract с backup-before-upgrade.
func MigrateDirWithPolicy(ctx context.Context, pool *pgxpool.Pool, dir string, options MigrationOptions) error {
	if err := ensureSchemaMigrationsTable(ctx, pool); err != nil {
		return err
	}
	if err := ensureRuntimeVersionTable(ctx, pool); err != nil {
		return err
	}
	currentModule := strings.TrimSpace(options.ModuleName)
	currentVersion := strings.TrimSpace(options.ModuleVersion)
	previousVersion, err := readRuntimeVersion(ctx, pool, currentModule)
	if err != nil {
		return err
	}
	needsUpgrade, err := shouldUpgradeVersion(previousVersion, currentVersion)
	if err != nil {
		return err
	}
	if needsUpgrade && previousVersion != "" {
		if err := backupPostgresBeforeUpgrade(ctx, pool, options.BackupDir, currentModule, previousVersion, currentVersion); err != nil {
			return err
		}
	}
	if err := applyMigrationsDir(ctx, pool, dir); err != nil {
		return err
	}
	if currentModule != "" && currentVersion != "" {
		if err := writeRuntimeVersion(ctx, pool, currentModule, currentVersion); err != nil {
			return err
		}
	}
	return nil
}

func ensureSchemaMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	return nil
}

func applyMigrationsDir(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) != 1 {
		return fmt.Errorf("postgres first-launch migration path must contain exactly one canonical SQL file, got %d", len(names))
	}
	for _, name := range names {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		var exists int
		if err := tx.QueryRow(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = $1`, name).Scan(&exists); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if exists > 0 {
			_ = tx.Rollback(ctx)
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func ensureRuntimeVersionTable(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS db_runtime_versions (module_name TEXT PRIMARY KEY, module_version TEXT NOT NULL, updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	return nil
}

func readRuntimeVersion(ctx context.Context, pool *pgxpool.Pool, moduleName string) (string, error) {
	if strings.TrimSpace(moduleName) == "" {
		return "", nil
	}
	var version string
	err := pool.QueryRow(ctx, `SELECT module_version FROM db_runtime_versions WHERE module_name = $1`, moduleName).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(version), nil
}

func writeRuntimeVersion(ctx context.Context, pool *pgxpool.Pool, moduleName, moduleVersion string) error {
	if _, err := pool.Exec(ctx, `
INSERT INTO db_runtime_versions(module_name,module_version,updated_at)
VALUES ($1,$2,now())
ON CONFLICT(module_name) DO UPDATE SET
  module_version = EXCLUDED.module_version,
  updated_at = now()
`, moduleName, moduleVersion); err != nil {
		return err
	}
	return nil
}

func shouldUpgradeVersion(previous, current string) (bool, error) {
	if strings.TrimSpace(current) == "" {
		return false, nil
	}
	if strings.TrimSpace(previous) == "" {
		return true, nil
	}
	compare, err := compareModuleVersion(previous, current)
	if err != nil {
		return false, err
	}
	return compare < 0, nil
}

func compareModuleVersion(left, right string) (int, error) {
	leftParts, err := parseModuleVersion(left)
	if err != nil {
		return 0, fmt.Errorf("invalid module version %q: %w", left, err)
	}
	rightParts, err := parseModuleVersion(right)
	if err != nil {
		return 0, fmt.Errorf("invalid module version %q: %w", right, err)
	}
	for i := range leftParts {
		if leftParts[i] < rightParts[i] {
			return -1, nil
		}
		if leftParts[i] > rightParts[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseModuleVersion(raw string) ([3]int, error) {
	var parsed [3]int
	normalized := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	normalized, _, _ = strings.Cut(normalized, "+")
	normalized, _, _ = strings.Cut(normalized, "-")
	parts := strings.Split(normalized, ".")
	if len(parts) != len(parsed) {
		return parsed, fmt.Errorf("expected semantic version major.minor.patch")
	}
	for i := range parsed {
		value, err := strconv.Atoi(parts[i])
		if err != nil || value < 0 {
			return parsed, fmt.Errorf("invalid semantic segment %q", parts[i])
		}
		parsed[i] = value
	}
	return parsed, nil
}

func backupPostgresBeforeUpgrade(ctx context.Context, pool *pgxpool.Pool, backupDir, moduleName, previousVersion, targetVersion string) error {
	if strings.TrimSpace(backupDir) == "" {
		backupDir = filepath.Join("data", "cloud-backups")
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	filePath := filepath.Join(backupDir, fmt.Sprintf("%s_%s_to_%s_%s.jsonl", sanitizeFilenameToken(moduleName), sanitizeFilenameToken(previousVersion), sanitizeFilenameToken(targetVersion), stamp))
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	rows, err := pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename`)
	if err != nil {
		return err
	}
	tables := make([]string, 0, 64)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, table := range tables {
		query := fmt.Sprintf("SELECT row_to_json(t) FROM %s t", pgx.Identifier{"public", table}.Sanitize())
		tableRows, err := pool.Query(ctx, query)
		if err != nil {
			return err
		}
		for tableRows.Next() {
			var raw []byte
			if err := tableRows.Scan(&raw); err != nil {
				tableRows.Close()
				return err
			}
			line := fmt.Appendf(nil, `{"table":"%s","row":%s}`+"\n", table, string(raw))
			if _, err := file.Write(line); err != nil {
				tableRows.Close()
				return err
			}
		}
		if err := tableRows.Err(); err != nil {
			tableRows.Close()
			return err
		}
		tableRows.Close()
	}
	return file.Sync()
}

func sanitizeFilenameToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
