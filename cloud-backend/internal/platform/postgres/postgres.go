package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const postgresMigrationLockKey int64 = 5501978001

// MigrationOptions задает startup policy для module/schema version, backup и проверки схемы.
type MigrationOptions struct {
	ModuleName         string
	ModuleVersion      string
	BackupDir          string
	SchemaRequirements []SchemaRequirement
}

// SchemaRequirement describes implemented-now runtime schema that must exist before HTTP/workers start.
type SchemaRequirement struct {
	Table          string
	Columns        []string
	Indexes        []string
	RequiredBy     string
	MigrationFile  string
	RecoveryAction string
}

type migrationFile struct {
	Name           string
	Path           string
	Body           []byte
	ChecksumSHA256 string
}

type SchemaVerificationError struct {
	ObjectType     string
	Table          string
	Column         string
	Index          string
	RequiredBy     string
	MigrationFile  string
	RecoveryAction string
}

func (e *SchemaVerificationError) Error() string {
	switch e.ObjectType {
	case "column":
		return fmt.Sprintf("postgres schema verification failed: missing column %s.%s", e.Table, e.Column)
	case "index":
		return fmt.Sprintf("postgres schema verification failed: missing index %s on %s", e.Index, e.Table)
	default:
		return fmt.Sprintf("postgres schema verification failed: missing table %s", e.Table)
	}
}

type migrationFileError struct {
	Operation string
	File      string
	Err       error
}

func (e *migrationFileError) Error() string {
	return fmt.Sprintf("postgres migration: %s %s: %v", e.Operation, e.File, e.Err)
}

func (e *migrationFileError) Unwrap() error {
	return e.Err
}

type pgRunner interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
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

// MigrateDirWithPolicy применяет ordered migrations до запуска runtime-кода и проверяет итоговую схему.
func MigrateDirWithPolicy(ctx context.Context, pool *pgxpool.Pool, dir string, options MigrationOptions) (err error) {
	started := time.Now()
	moduleName := strings.TrimSpace(options.ModuleName)
	targetVersion := strings.TrimSpace(options.ModuleVersion)
	previousVersion := ""
	databaseName := ""
	defer func() {
		result := "success"
		level := slog.LevelInfo
		errorCode := ""
		if err != nil {
			result = "failed"
			level = slog.LevelError
			errorCode = "DB_MIGRATION_FAILED"
			var verificationErr *SchemaVerificationError
			if errors.As(err, &verificationErr) {
				errorCode = "DB_SCHEMA_VERIFICATION_FAILED"
			}
		}
		attrs := []any{
			"operation", "db.migration",
			"action", "startup_upgrade",
			"result", result,
			"error_code", errorCode,
			"db_type", "postgres",
			"database", databaseName,
			"module_name", moduleName,
			"target_version", targetVersion,
			"current_version", previousVersion,
			"migration_dir", dir,
			"duration_ms", time.Since(started).Milliseconds(),
		}
		if err != nil {
			attrs = append(attrs, "sql_state", pgSQLState(err), "internal_error", err.Error())
			var verificationErr *SchemaVerificationError
			if errors.As(err, &verificationErr) {
				attrs = append(attrs,
					"missing_table", verificationErr.Table,
					"missing_column", verificationErr.Column,
					"missing_index", verificationErr.Index,
					"required_by", verificationErr.RequiredBy,
					"migration_file", verificationErr.MigrationFile,
					"recovery_action", verificationErr.RecoveryAction,
				)
			}
			var fileErr *migrationFileError
			if errors.As(err, &fileErr) {
				attrs = append(attrs, "migration_file", fileErr.File)
			}
		}
		slog.Log(ctx, level, "postgres startup migration завершена", attrs...)
	}()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("postgres migration: acquire connection: %w", err)
	}
	defer conn.Release()

	databaseName = currentDatabaseName(ctx, conn)
	versionTableExisted, err := postgresTableExists(ctx, conn, "db_runtime_versions")
	if err != nil {
		return fmt.Errorf("postgres migration: inspect runtime version table: %w", err)
	}
	schemaTableExisted, err := postgresTableExists(ctx, conn, "schema_migrations")
	if err != nil {
		return fmt.Errorf("postgres migration: inspect schema_migrations table: %w", err)
	}
	existingTables, err := countPublicUserTables(ctx, conn)
	if err != nil {
		return fmt.Errorf("postgres migration: inspect existing public tables: %w", err)
	}
	if !versionTableExisted {
		slog.WarnContext(ctx, "postgres db_runtime_versions отсутствует, БД считается самой старой", "operation", "db.migration", "action", "inspect_version_table", "result", "oldest", "db_type", "postgres", "database", databaseName, "module_name", moduleName, "migration_dir", dir)
	}

	if versionTableExisted {
		previousVersion, err = readRuntimeVersion(ctx, conn, moduleName)
		if err != nil {
			return fmt.Errorf("postgres migration: read runtime version: %w", err)
		}
	}
	needsVersionUpgrade, err := shouldUpgradeVersion(previousVersion, targetVersion)
	if err != nil {
		return err
	}

	files, err := readMigrationFiles(dir)
	if err != nil {
		return fmt.Errorf("postgres migration: read migration dir %s: %w", dir, err)
	}
	allowCanonicalUpgrade := allowManagedChecksumUpgrade(needsVersionUpgrade, versionTableExisted)
	pending, err := pendingMigrations(ctx, conn, files, schemaTableExisted, allowCanonicalUpgrade)
	if err != nil {
		return err
	}
	needsBackup := (needsVersionUpgrade || len(pending) > 0) && (previousVersion != "" || !versionTableExisted || existingTables > 0) && existingTables > 0
	if needsBackup {
		if err := backupPostgresBeforeUpgrade(ctx, conn, options.BackupDir, moduleName, previousVersion, targetVersion); err != nil {
			return fmt.Errorf("postgres migration: backup before upgrade: %w", err)
		}
	}

	if err := acquirePostgresMigrationLock(ctx, conn); err != nil {
		return fmt.Errorf("postgres migration: acquire advisory lock: %w", err)
	}
	defer releasePostgresMigrationLock(context.Background(), conn)

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return fmt.Errorf("postgres migration: ensure schema_migrations: %w", err)
	}
	if err := ensureRuntimeVersionTable(ctx, conn); err != nil {
		return fmt.Errorf("postgres migration: ensure db_runtime_versions: %w", err)
	}

	// После advisory lock план пересчитывается, чтобы второй instance не применил уже завершенные migrations повторно.
	pending, err = pendingMigrations(ctx, conn, files, true, allowCanonicalUpgrade)
	if err != nil {
		return err
	}
	if err := applyPendingMigrations(ctx, conn, pending); err != nil {
		return err
	}
	if err := VerifySchema(ctx, conn, options.SchemaRequirements); err != nil {
		return err
	}
	latestName, latestChecksum := latestMigrationIdentity(files)
	if moduleName != "" && targetVersion != "" {
		if err := writeRuntimeVersion(ctx, conn, moduleName, targetVersion, latestName, latestChecksum, "applied"); err != nil {
			return fmt.Errorf("postgres migration: write runtime version: %w", err)
		}
	}
	return nil
}

func ensureSchemaMigrationsTable(ctx context.Context, runner pgRunner) error {
	if _, err := runner.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, checksum_sha256 TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'applied', applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	for _, stmt := range []string{
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum_sha256 TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'applied'`,
		`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT now()`,
	} {
		if _, err := runner.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func ensureRuntimeVersionTable(ctx context.Context, runner pgRunner) error {
	if _, err := runner.Exec(ctx, `CREATE TABLE IF NOT EXISTS db_runtime_versions (module_name TEXT PRIMARY KEY, module_version TEXT NOT NULL, schema_version TEXT NOT NULL DEFAULT '', checksum_sha256 TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'applied', applied_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	for _, stmt := range []string{
		`ALTER TABLE db_runtime_versions ADD COLUMN IF NOT EXISTS schema_version TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE db_runtime_versions ADD COLUMN IF NOT EXISTS checksum_sha256 TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE db_runtime_versions ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'applied'`,
		`ALTER TABLE db_runtime_versions ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT now()`,
		`ALTER TABLE db_runtime_versions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`,
	} {
		if _, err := runner.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func readRuntimeVersion(ctx context.Context, runner pgRunner, moduleName string) (string, error) {
	if strings.TrimSpace(moduleName) == "" {
		return "", nil
	}
	var version string
	err := runner.QueryRow(ctx, `SELECT module_version FROM db_runtime_versions WHERE module_name = $1`, moduleName).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(version), nil
}

func writeRuntimeVersion(ctx context.Context, runner pgRunner, moduleName, moduleVersion, schemaVersion, checksum, status string) error {
	_, err := runner.Exec(ctx, `
INSERT INTO db_runtime_versions(module_name,module_version,schema_version,checksum_sha256,status,applied_at,updated_at)
VALUES ($1,$2,$3,$4,$5,now(),now())
ON CONFLICT(module_name) DO UPDATE SET
  module_version = EXCLUDED.module_version,
  schema_version = EXCLUDED.schema_version,
  checksum_sha256 = EXCLUDED.checksum_sha256,
  status = EXCLUDED.status,
  applied_at = EXCLUDED.applied_at,
  updated_at = now()
`, moduleName, moduleVersion, schemaVersion, checksum, status)
	return err
}

func readMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	files := make([]migrationFile, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(body)
		files = append(files, migrationFile{Name: name, Path: path, Body: body, ChecksumSHA256: hex.EncodeToString(sum[:])})
	}
	return files, nil
}

func pendingMigrations(ctx context.Context, runner pgRunner, files []migrationFile, schemaTableExisted bool, allowCanonicalUpgrade bool) ([]migrationFile, error) {
	if !schemaTableExisted {
		return files, nil
	}
	hasChecksumColumn, err := postgresColumnExists(ctx, runner, "schema_migrations", "checksum_sha256")
	if err != nil {
		return nil, fmt.Errorf("postgres migration: inspect schema_migrations checksum column: %w", err)
	}
	pending := make([]migrationFile, 0, len(files))
	for _, file := range files {
		if !hasChecksumColumn {
			var n int
			err := runner.QueryRow(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = $1`, file.Name).Scan(&n)
			if err != nil {
				return nil, fmt.Errorf("postgres migration: read legacy migration marker for %s: %w", file.Name, err)
			}
			// A legacy marker without checksum is not proof that implemented-now schema exists.
			// Re-run idempotent SQL before verification so missing runtime tables can be repaired.
			pending = append(pending, file)
			continue
		}
		var storedChecksum string
		err := runner.QueryRow(ctx, `SELECT checksum_sha256 FROM schema_migrations WHERE version = $1`, file.Name).Scan(&storedChecksum)
		if errors.Is(err, pgx.ErrNoRows) {
			pending = append(pending, file)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("postgres migration: read checksum for %s: %w", file.Name, err)
		}
		storedChecksum = strings.TrimSpace(storedChecksum)
		if storedChecksum == "" {
			// Empty checksum is a legacy marker, not proof of current schema.
			// The migration history is updated only after the idempotent file succeeds.
			pending = append(pending, file)
			continue
		}
		if storedChecksum != file.ChecksumSHA256 {
			if allowCanonicalUpgrade {
				pending = append(pending, file)
				continue
			}
			return nil, fmt.Errorf("postgres migration checksum mismatch for %s: database=%s file=%s", file.Name, storedChecksum, file.ChecksumSHA256)
		}
	}
	return pending, nil
}

func applyPendingMigrations(ctx context.Context, runner pgRunner, files []migrationFile) error {
	for _, file := range files {
		started := time.Now()
		tx, err := runner.Begin(ctx)
		if err != nil {
			return &migrationFileError{Operation: "begin", File: file.Name, Err: err}
		}
		if _, err := tx.Exec(ctx, string(file.Body)); err != nil {
			_ = tx.Rollback(ctx)
			return &migrationFileError{Operation: "apply", File: file.Name, Err: err}
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO schema_migrations(version,checksum_sha256,status,applied_at)
VALUES ($1,$2,'applied',now())
ON CONFLICT(version) DO UPDATE SET
  checksum_sha256 = EXCLUDED.checksum_sha256,
  status = EXCLUDED.status,
  applied_at = EXCLUDED.applied_at
`, file.Name, file.ChecksumSHA256); err != nil {
			_ = tx.Rollback(ctx)
			return &migrationFileError{Operation: "record", File: file.Name, Err: err}
		}
		if err := tx.Commit(ctx); err != nil {
			return &migrationFileError{Operation: "commit", File: file.Name, Err: err}
		}
		slog.InfoContext(ctx, "postgres migration применена", "operation", "db.migration", "action", "apply_migration", "result", "success", "db_type", "postgres", "migration_file", file.Name, "migration_checksum", file.ChecksumSHA256, "duration_ms", time.Since(started).Milliseconds())
	}
	return nil
}

// VerifySchema проверяет критичные таблицы, колонки и индексы до запуска HTTP/workers.
func VerifySchema(ctx context.Context, runner pgRunner, requirements []SchemaRequirement) error {
	started := time.Now()
	for _, req := range requirements {
		if strings.TrimSpace(req.Table) == "" {
			continue
		}
		exists, err := postgresTableExists(ctx, runner, req.Table)
		if err != nil {
			return fmt.Errorf("postgres schema verification: inspect table %s: %w", req.Table, err)
		}
		if !exists {
			return missingSchemaObject(req, "table", "", "")
		}
		for _, column := range req.Columns {
			if strings.TrimSpace(column) == "" {
				continue
			}
			columnExists, err := postgresColumnExists(ctx, runner, req.Table, column)
			if err != nil {
				return fmt.Errorf("postgres schema verification: inspect column %s.%s: %w", req.Table, column, err)
			}
			if !columnExists {
				return missingSchemaObject(req, "column", column, "")
			}
		}
		for _, index := range req.Indexes {
			if strings.TrimSpace(index) == "" {
				continue
			}
			indexExists, err := postgresIndexExists(ctx, runner, req.Table, index)
			if err != nil {
				return fmt.Errorf("postgres schema verification: inspect index %s: %w", index, err)
			}
			if !indexExists {
				return missingSchemaObject(req, "index", "", index)
			}
		}
	}
	slog.InfoContext(ctx, "postgres schema verification завершена", "operation", "db.schema_verification", "action", "verify_schema", "result", "success", "db_type", "postgres", "duration_ms", time.Since(started).Milliseconds())
	return nil
}

func missingSchemaObject(req SchemaRequirement, objectType, column, index string) error {
	recovery := strings.TrimSpace(req.RecoveryAction)
	if recovery == "" {
		recovery = "run startup migrations with the configured migration directory; for local/dev, recreate the database from migrations if repair is impossible"
	}
	return &SchemaVerificationError{
		ObjectType:     objectType,
		Table:          req.Table,
		Column:         column,
		Index:          index,
		RequiredBy:     req.RequiredBy,
		MigrationFile:  req.MigrationFile,
		RecoveryAction: recovery,
	}
}

func acquirePostgresMigrationLock(ctx context.Context, runner pgRunner) error {
	_, err := runner.Exec(ctx, `SELECT pg_advisory_lock($1)`, postgresMigrationLockKey)
	return err
}

func releasePostgresMigrationLock(ctx context.Context, runner pgRunner) {
	if _, err := runner.Exec(ctx, `SELECT pg_advisory_unlock($1)`, postgresMigrationLockKey); err != nil {
		slog.ErrorContext(ctx, "postgres migration advisory lock не освобожден", "operation", "db.migration", "action", "release_lock", "result", "failed", "db_type", "postgres", "error_code", "DB_MIGRATION_LOCK_RELEASE_FAILED", "sql_state", pgSQLState(err), "internal_error", err.Error())
	}
}

func postgresTableExists(ctx context.Context, runner pgRunner, table string) (bool, error) {
	var exists bool
	err := runner.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists)
	return exists, err
}

func postgresColumnExists(ctx context.Context, runner pgRunner, table, column string) (bool, error) {
	var n int
	err := runner.QueryRow(ctx, `SELECT COUNT(1) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`, table, column).Scan(&n)
	return n > 0, err
}

func postgresIndexExists(ctx context.Context, runner pgRunner, table, index string) (bool, error) {
	var n int
	err := runner.QueryRow(ctx, `SELECT COUNT(1) FROM pg_indexes WHERE schemaname = 'public' AND tablename = $1 AND indexname = $2`, table, index).Scan(&n)
	return n > 0, err
}

func countPublicUserTables(ctx context.Context, runner pgRunner) (int, error) {
	var n int
	err := runner.QueryRow(ctx, `SELECT COUNT(1) FROM pg_tables WHERE schemaname = 'public' AND tablename NOT IN ('schema_migrations','db_runtime_versions')`).Scan(&n)
	return n, err
}

func currentDatabaseName(ctx context.Context, runner pgRunner) string {
	var name string
	if err := runner.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return ""
	}
	return name
}

func latestMigrationIdentity(files []migrationFile) (string, string) {
	if len(files) == 0 {
		return "", ""
	}
	latest := files[len(files)-1]
	return latest.Name, latest.ChecksumSHA256
}

func allowManagedChecksumUpgrade(needsVersionUpgrade bool, versionTableExisted bool) bool {
	return needsVersionUpgrade || !versionTableExisted
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
	if compare > 0 {
		return false, fmt.Errorf("database schema is newer than application; downgrade is not supported: database=%s application=%s", previous, current)
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

func backupPostgresBeforeUpgrade(ctx context.Context, runner pgRunner, backupDir, moduleName, previousVersion, targetVersion string) error {
	started := time.Now()
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

	rows, err := runner.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename`)
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
		tableRows, err := runner.Query(ctx, query)
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
	if err := file.Sync(); err != nil {
		return err
	}
	slog.InfoContext(ctx, "postgres backup перед миграцией создан", "operation", "db.backup", "action", "backup_before_upgrade", "result", "success", "db_type", "postgres", "backup_path", filePath, "module_name", moduleName, "current_version", previousVersion, "target_version", targetVersion, "duration_ms", time.Since(started).Milliseconds())
	return nil
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

func pgSQLState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.SQLState()
	}
	return ""
}
