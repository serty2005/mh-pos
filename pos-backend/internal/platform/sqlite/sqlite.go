package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	functionalMinimumSQLiteVersion = "3.37.0"
	requiredWALPilotSQLiteVersion  = "3.51.3"
	requiredBusyTimeoutMS          = 5000
	requiredSynchronousNormal      = 1
	requiredForeignKeysOn          = 1
)

var allowedPinnedBackportSQLiteVersions = map[string]struct{}{
	"3.50.7": {},
	"3.44.6": {},
}

type RuntimeReport struct {
	SQLiteVersion string
	JournalMode   string
	Synchronous   int
	ForeignKeys   int
	BusyTimeoutMS int
}

// MigrationOptions задает startup policy для module/schema version, backup и проверки схемы.
type MigrationOptions struct {
	ModuleName         string
	ModuleVersion      string
	BackupDir          string
	SchemaRequirements []SchemaRequirement
}

// SchemaRequirement описывает критичные runtime-объекты SQLite, без которых модуль не должен стартовать.
type SchemaRequirement struct {
	Table          string
	Columns        []string
	Indexes        []string
	RequiredBy     string
	MigrationFile  string
	RecoveryAction string
}

// SchemaVerificationError хранит безопасные детали отсутствующего runtime-объекта для structured logs.
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
		return fmt.Sprintf("sqlite schema verification failed: missing column %s.%s", e.Table, e.Column)
	case "index":
		return fmt.Sprintf("sqlite schema verification failed: missing index %s on %s", e.Index, e.Table)
	default:
		return fmt.Sprintf("sqlite schema verification failed: missing table %s", e.Table)
	}
}

type migrationFile struct {
	Name           string
	Path           string
	Body           []byte
	ChecksumSHA256 string
}

type sqliteRepairBlock struct {
	Kind   string
	Table  string
	Column string
	Index  string
	SQL    string
}

type sqliteRunner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if err := configureRuntime(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure sqlite runtime: %w", err)
	}
	if _, err := EnsureRuntimeGate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func EnsureRuntimeGate(ctx context.Context, db *sql.DB) (RuntimeReport, error) {
	report, err := inspectRuntime(ctx, db)
	if err != nil {
		return RuntimeReport{}, err
	}
	if err := validateRuntimeReport(report); err != nil {
		return report, err
	}
	return report, nil
}

func configureRuntime(ctx context.Context, db *sql.DB) error {
	var journalMode string
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode = WAL`).Scan(&journalMode); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA synchronous = NORMAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;`); err != nil {
		return err
	}
	return nil
}

func inspectRuntime(ctx context.Context, db *sql.DB) (RuntimeReport, error) {
	var report RuntimeReport
	if err := db.QueryRowContext(ctx, `SELECT sqlite_version()`).Scan(&report.SQLiteVersion); err != nil {
		return RuntimeReport{}, fmt.Errorf("sqlite runtime gate: sqlite_version() is unavailable: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&report.JournalMode); err != nil {
		return RuntimeReport{}, fmt.Errorf("sqlite runtime gate: PRAGMA journal_mode failed: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA synchronous`).Scan(&report.Synchronous); err != nil {
		return RuntimeReport{}, fmt.Errorf("sqlite runtime gate: PRAGMA synchronous failed: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&report.ForeignKeys); err != nil {
		return RuntimeReport{}, fmt.Errorf("sqlite runtime gate: PRAGMA foreign_keys failed: %w", err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&report.BusyTimeoutMS); err != nil {
		return RuntimeReport{}, fmt.Errorf("sqlite runtime gate: PRAGMA busy_timeout failed: %w", err)
	}
	return report, nil
}

func validateRuntimeReport(report RuntimeReport) error {
	if compare, err := compareSQLiteVersions(report.SQLiteVersion, functionalMinimumSQLiteVersion); err != nil {
		return fmt.Errorf("sqlite runtime gate failed: invalid sqlite_version %q: %w", report.SQLiteVersion, err)
	} else if compare < 0 {
		return fmt.Errorf("sqlite runtime gate failed: sqlite_version %s is below functional minimum %s; required production WAL pilot baseline is %s or pinned backport %s", report.SQLiteVersion, functionalMinimumSQLiteVersion, requiredWALPilotSQLiteVersion, pinnedBackportsLabel())
	}
	if !meetsWALPilotBaseline(report.SQLiteVersion) {
		return fmt.Errorf("sqlite runtime gate failed: sqlite_version %s does not satisfy required production WAL pilot baseline %s or pinned backport %s; functional minimum is %s", report.SQLiteVersion, requiredWALPilotSQLiteVersion, pinnedBackportsLabel(), functionalMinimumSQLiteVersion)
	}
	if strings.ToLower(report.JournalMode) != "wal" {
		return fmt.Errorf("sqlite runtime gate failed: PRAGMA journal_mode = %q, want WAL", report.JournalMode)
	}
	if report.Synchronous != requiredSynchronousNormal {
		return fmt.Errorf("sqlite runtime gate failed: PRAGMA synchronous = %d, want %d (NORMAL)", report.Synchronous, requiredSynchronousNormal)
	}
	if report.ForeignKeys != requiredForeignKeysOn {
		return fmt.Errorf("sqlite runtime gate failed: PRAGMA foreign_keys = %d, want %d (ON)", report.ForeignKeys, requiredForeignKeysOn)
	}
	if report.BusyTimeoutMS < requiredBusyTimeoutMS {
		return fmt.Errorf("sqlite runtime gate failed: PRAGMA busy_timeout = %d, want at least %d", report.BusyTimeoutMS, requiredBusyTimeoutMS)
	}
	return nil
}

func meetsWALPilotBaseline(version string) bool {
	if _, ok := allowedPinnedBackportSQLiteVersions[version]; ok {
		return true
	}
	compare, err := compareSQLiteVersions(version, requiredWALPilotSQLiteVersion)
	return err == nil && compare >= 0
}

func compareSQLiteVersions(left, right string) (int, error) {
	leftParts, err := parseSQLiteVersion(left)
	if err != nil {
		return 0, err
	}
	rightParts, err := parseSQLiteVersion(right)
	if err != nil {
		return 0, err
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

func parseSQLiteVersion(version string) ([3]int, error) {
	var parsed [3]int
	parts := strings.Split(version, ".")
	if len(parts) != len(parsed) {
		return parsed, fmt.Errorf("expected major.minor.patch")
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return parsed, err
		}
		parsed[i] = n
	}
	return parsed, nil
}

func pinnedBackportsLabel() string {
	versions := make([]string, 0, len(allowedPinnedBackportSQLiteVersions))
	for version := range allowedPinnedBackportSQLiteVersions {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return strings.Join(versions, "/")
}

func MigrateDir(ctx context.Context, db *sql.DB, dir string) error {
	return MigrateDirWithPolicy(ctx, db, "", dir, MigrationOptions{})
}

// MigrateDirWithPolicy применяет ordered migrations под SQLite write lock до запуска runtime-кода.
func MigrateDirWithPolicy(ctx context.Context, db *sql.DB, dbPath, dir string, options MigrationOptions) (err error) {
	started := time.Now()
	moduleName := strings.TrimSpace(options.ModuleName)
	targetVersion := strings.TrimSpace(options.ModuleVersion)
	previousVersion := ""
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
			"db_type", "sqlite",
			"db_path", dbPath,
			"module_name", moduleName,
			"target_version", targetVersion,
			"current_version", previousVersion,
			"migration_dir", dir,
			"duration_ms", time.Since(started).Milliseconds(),
		}
		if err != nil {
			attrs = append(attrs, "internal_error", err.Error())
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
		}
		slog.Log(ctx, level, "sqlite startup migration завершена", attrs...)
	}()

	versionTableExisted, err := sqliteTableExists(ctx, db, "db_runtime_versions")
	if err != nil {
		return fmt.Errorf("sqlite migration: inspect runtime version table: %w", err)
	}
	schemaTableExisted, err := sqliteTableExists(ctx, db, "schema_migrations")
	if err != nil {
		return fmt.Errorf("sqlite migration: inspect schema_migrations table: %w", err)
	}
	existingTables, err := countSQLiteUserTables(ctx, db)
	if err != nil {
		return fmt.Errorf("sqlite migration: inspect existing tables: %w", err)
	}
	if !versionTableExisted {
		slog.WarnContext(ctx, "sqlite db_runtime_versions отсутствует, БД считается самой старой", "operation", "db.migration", "action", "inspect_version_table", "result", "oldest", "db_type", "sqlite", "db_path", dbPath, "module_name", moduleName, "migration_dir", dir)
	}

	if versionTableExisted {
		previousVersion, err = readRuntimeVersion(ctx, db, moduleName)
		if err != nil {
			return fmt.Errorf("sqlite migration: read runtime version: %w", err)
		}
	}
	needsVersionUpgrade, err := shouldUpgradeVersion(previousVersion, targetVersion)
	if err != nil {
		return err
	}
	files, err := readMigrationFiles(dir)
	if err != nil {
		return fmt.Errorf("sqlite migration: read migration dir %s: %w", dir, err)
	}
	allowCanonicalUpgrade := allowManagedChecksumUpgrade(needsVersionUpgrade, versionTableExisted)
	pending, err := pendingMigrations(ctx, db, files, schemaTableExisted, allowCanonicalUpgrade)
	if err != nil {
		return err
	}
	needsBackup := (needsVersionUpgrade || len(pending) > 0) && (previousVersion != "" || !versionTableExisted || existingTables > 0) && existingTables > 0 && strings.TrimSpace(dbPath) != ""
	if needsBackup {
		if err := backupSQLiteBeforeUpgrade(ctx, db, dbPath, options.BackupDir, moduleName, previousVersion, targetVersion); err != nil {
			return fmt.Errorf("sqlite migration: backup before upgrade: %w", err)
		}
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("sqlite migration: acquire connection: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("sqlite migration: acquire write lock: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), `ROLLBACK`)
		}
	}()

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return fmt.Errorf("sqlite migration: ensure schema_migrations: %w", err)
	}
	if err := ensureRuntimeVersionTable(ctx, conn); err != nil {
		return fmt.Errorf("sqlite migration: ensure db_runtime_versions: %w", err)
	}
	// После BEGIN IMMEDIATE план пересчитывается, чтобы concurrent process не оставил устаревший pending set.
	pending, err = pendingMigrations(ctx, conn, files, true, allowCanonicalUpgrade)
	if err != nil {
		return err
	}
	// Старые pre-pilot БД без history считаются oldest DB: managed files должны
	// выполняться idempotent-образом, иначе CREATE IF NOT EXISTS не создаст
	// недостающие объекты перед repair migration и schema verification.
	adoptBase := false
	if err := applyPendingMigrations(ctx, conn, pending, adoptBase); err != nil {
		return err
	}
	if err := VerifySchema(ctx, conn, options.SchemaRequirements); err != nil {
		return err
	}
	latestName, latestChecksum := latestMigrationIdentity(files)
	if moduleName != "" && targetVersion != "" {
		if err := writeRuntimeVersion(ctx, conn, moduleName, targetVersion, latestName, latestChecksum, "applied"); err != nil {
			return fmt.Errorf("sqlite migration: write runtime version: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("sqlite migration: commit: %w", err)
	}
	committed = true
	return nil
}

func ensureSchemaMigrationsTable(ctx context.Context, runner sqliteRunner) error {
	if _, err := runner.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, checksum_sha256 TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'applied', applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	return ensureSQLiteColumns(ctx, runner, "schema_migrations", map[string]string{
		"checksum_sha256": `ALTER TABLE schema_migrations ADD COLUMN checksum_sha256 TEXT NOT NULL DEFAULT ''`,
		"status":          `ALTER TABLE schema_migrations ADD COLUMN status TEXT NOT NULL DEFAULT 'applied'`,
		"applied_at":      `ALTER TABLE schema_migrations ADD COLUMN applied_at TEXT NOT NULL DEFAULT ''`,
	})
}

func ensureRuntimeVersionTable(ctx context.Context, runner sqliteRunner) error {
	if _, err := runner.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS db_runtime_versions (module_name TEXT PRIMARY KEY, module_version TEXT NOT NULL, schema_version TEXT NOT NULL DEFAULT '', checksum_sha256 TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'applied', applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	return ensureSQLiteColumns(ctx, runner, "db_runtime_versions", map[string]string{
		"schema_version":  `ALTER TABLE db_runtime_versions ADD COLUMN schema_version TEXT NOT NULL DEFAULT ''`,
		"checksum_sha256": `ALTER TABLE db_runtime_versions ADD COLUMN checksum_sha256 TEXT NOT NULL DEFAULT ''`,
		"status":          `ALTER TABLE db_runtime_versions ADD COLUMN status TEXT NOT NULL DEFAULT 'applied'`,
		"applied_at":      `ALTER TABLE db_runtime_versions ADD COLUMN applied_at TEXT NOT NULL DEFAULT ''`,
		"updated_at":      `ALTER TABLE db_runtime_versions ADD COLUMN updated_at TEXT NOT NULL DEFAULT ''`,
	})
}

func ensureSQLiteColumns(ctx context.Context, runner sqliteRunner, table string, alterByColumn map[string]string) error {
	for column, stmt := range alterByColumn {
		exists, err := sqliteColumnExists(ctx, runner, table, column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := runner.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func readRuntimeVersion(ctx context.Context, runner sqliteRunner, moduleName string) (string, error) {
	if strings.TrimSpace(moduleName) == "" {
		return "", nil
	}
	var version string
	err := runner.QueryRowContext(ctx, `SELECT module_version FROM db_runtime_versions WHERE module_name = ?`, moduleName).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(version), nil
}

func writeRuntimeVersion(ctx context.Context, runner sqliteRunner, moduleName, moduleVersion, schemaVersion, checksum, status string) error {
	_, err := runner.ExecContext(ctx, `
INSERT INTO db_runtime_versions(module_name,module_version,schema_version,checksum_sha256,status,applied_at,updated_at)
VALUES (?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
ON CONFLICT(module_name) DO UPDATE SET
  module_version = excluded.module_version,
  schema_version = excluded.schema_version,
  checksum_sha256 = excluded.checksum_sha256,
  status = excluded.status,
  applied_at = excluded.applied_at,
  updated_at = CURRENT_TIMESTAMP
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

func pendingMigrations(ctx context.Context, runner sqliteRunner, files []migrationFile, schemaTableExisted bool, allowCanonicalUpgrade bool) ([]migrationFile, error) {
	if !schemaTableExisted {
		return files, nil
	}
	hasChecksumColumn, err := sqliteColumnExists(ctx, runner, "schema_migrations", "checksum_sha256")
	if err != nil {
		return nil, fmt.Errorf("sqlite migration: inspect schema_migrations checksum column: %w", err)
	}
	pending := make([]migrationFile, 0, len(files))
	for _, file := range files {
		if !hasChecksumColumn {
			var n int
			err := runner.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, file.Name).Scan(&n)
			if err != nil {
				return nil, fmt.Errorf("sqlite migration: read legacy migration marker for %s: %w", file.Name, err)
			}
			if n == 0 || allowCanonicalUpgrade {
				pending = append(pending, file)
			}
			continue
		}
		var storedChecksum string
		err := runner.QueryRowContext(ctx, `SELECT checksum_sha256 FROM schema_migrations WHERE version = ?`, file.Name).Scan(&storedChecksum)
		if errors.Is(err, sql.ErrNoRows) {
			pending = append(pending, file)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("sqlite migration: read checksum for %s: %w", file.Name, err)
		}
		storedChecksum = strings.TrimSpace(storedChecksum)
		if storedChecksum == "" {
			if allowCanonicalUpgrade {
				pending = append(pending, file)
				continue
			}
			if _, err := runner.ExecContext(ctx, `UPDATE schema_migrations SET checksum_sha256 = ? WHERE version = ?`, file.ChecksumSHA256, file.Name); err != nil {
				return nil, fmt.Errorf("sqlite migration: adopt legacy checksum for %s: %w", file.Name, err)
			}
			continue
		}
		if storedChecksum != file.ChecksumSHA256 {
			if allowCanonicalUpgrade {
				pending = append(pending, file)
				continue
			}
			return nil, fmt.Errorf("sqlite migration checksum mismatch for %s: database=%s file=%s", file.Name, storedChecksum, file.ChecksumSHA256)
		}
	}
	return pending, nil
}

func applyPendingMigrations(ctx context.Context, runner sqliteRunner, files []migrationFile, adoptExistingBase bool) error {
	for i, file := range files {
		started := time.Now()
		if adoptExistingBase && i == 0 {
			if _, err := runner.ExecContext(ctx, `INSERT INTO schema_migrations(version,checksum_sha256,status,applied_at) VALUES (?,?,'adopted',CURRENT_TIMESTAMP) ON CONFLICT(version) DO NOTHING`, file.Name, file.ChecksumSHA256); err != nil {
				return fmt.Errorf("sqlite migration: adopt existing base schema %s: %w", file.Name, err)
			}
			slog.WarnContext(ctx, "sqlite migration baseline adopted по фактической схеме", "operation", "db.migration", "action", "adopt_base_migration", "result", "adopted", "db_type", "sqlite", "migration_file", file.Name, "migration_checksum", file.ChecksumSHA256, "duration_ms", time.Since(started).Milliseconds())
			continue
		}
		if err := applySQLiteMigrationFile(ctx, runner, file); err != nil {
			return fmt.Errorf("sqlite migration: apply %s: %w", file.Name, err)
		}
		if _, err := runner.ExecContext(ctx, `
INSERT INTO schema_migrations(version,checksum_sha256,status,applied_at)
VALUES (?,?,'applied',CURRENT_TIMESTAMP)
ON CONFLICT(version) DO UPDATE SET
  checksum_sha256 = excluded.checksum_sha256,
  status = excluded.status,
  applied_at = excluded.applied_at
`, file.Name, file.ChecksumSHA256); err != nil {
			return fmt.Errorf("sqlite migration: record %s: %w", file.Name, err)
		}
		slog.InfoContext(ctx, "sqlite migration применена", "operation", "db.migration", "action", "apply_migration", "result", "success", "db_type", "sqlite", "migration_file", file.Name, "migration_checksum", file.ChecksumSHA256, "duration_ms", time.Since(started).Milliseconds())
	}
	return nil
}

func applySQLiteMigrationFile(ctx context.Context, runner sqliteRunner, file migrationFile) error {
	preamble, blocks, err := parseSQLiteRepairBlocks(string(file.Body))
	if err != nil {
		return err
	}
	if strings.TrimSpace(preamble) != "" {
		if _, err := runner.ExecContext(ctx, preamble); err != nil {
			return err
		}
	}
	for _, block := range blocks {
		shouldApply, err := shouldApplySQLiteRepairBlock(ctx, runner, block)
		if err != nil {
			return err
		}
		if !shouldApply {
			continue
		}
		if strings.TrimSpace(block.SQL) == "" {
			continue
		}
		if _, err := runner.ExecContext(ctx, block.SQL); err != nil {
			return err
		}
	}
	return nil
}

func shouldApplySQLiteRepairBlock(ctx context.Context, runner sqliteRunner, block sqliteRepairBlock) (bool, error) {
	switch block.Kind {
	case "repair-column":
		tableExists, err := sqliteTableExists(ctx, runner, block.Table)
		if err != nil {
			return false, fmt.Errorf("sqlite repair: inspect table %s: %w", block.Table, err)
		}
		if !tableExists {
			return false, nil
		}
		columnExists, err := sqliteColumnExists(ctx, runner, block.Table, block.Column)
		if err != nil {
			return false, fmt.Errorf("sqlite repair: inspect column %s.%s: %w", block.Table, block.Column, err)
		}
		return !columnExists, nil
	case "repair-index":
		indexExists, err := sqliteIndexExists(ctx, runner, block.Index)
		if err != nil {
			return false, fmt.Errorf("sqlite repair: inspect index %s: %w", block.Index, err)
		}
		return !indexExists, nil
	case "repair-sql":
		return true, nil
	default:
		return false, fmt.Errorf("sqlite repair: unsupported directive %q", block.Kind)
	}
}

func parseSQLiteRepairBlocks(body string) (string, []sqliteRepairBlock, error) {
	var preamble strings.Builder
	var blocks []sqliteRepairBlock
	var current *sqliteRepairBlock
	var currentSQL strings.Builder
	flush := func() {
		if current == nil {
			return
		}
		current.SQL = currentSQL.String()
		blocks = append(blocks, *current)
		current = nil
		currentSQL.Reset()
	}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- sqlite:repair-") {
			flush()
			block, err := parseSQLiteRepairDirective(trimmed)
			if err != nil {
				return "", nil, err
			}
			current = &block
			continue
		}
		if current == nil {
			preamble.WriteString(line)
			preamble.WriteByte('\n')
			continue
		}
		currentSQL.WriteString(line)
		currentSQL.WriteByte('\n')
	}
	flush()
	return preamble.String(), blocks, nil
}

func parseSQLiteRepairDirective(line string) (sqliteRepairBlock, error) {
	fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "-- sqlite:"))
	if len(fields) == 0 {
		return sqliteRepairBlock{}, fmt.Errorf("sqlite repair: empty directive")
	}
	switch fields[0] {
	case "repair-column":
		if len(fields) != 3 {
			return sqliteRepairBlock{}, fmt.Errorf("sqlite repair: repair-column requires table and column")
		}
		return sqliteRepairBlock{Kind: fields[0], Table: fields[1], Column: fields[2]}, nil
	case "repair-index":
		if len(fields) != 2 {
			return sqliteRepairBlock{}, fmt.Errorf("sqlite repair: repair-index requires index")
		}
		return sqliteRepairBlock{Kind: fields[0], Index: fields[1]}, nil
	case "repair-sql":
		if len(fields) != 1 {
			return sqliteRepairBlock{}, fmt.Errorf("sqlite repair: repair-sql does not accept arguments")
		}
		return sqliteRepairBlock{Kind: fields[0]}, nil
	default:
		return sqliteRepairBlock{}, fmt.Errorf("sqlite repair: unsupported directive %q", fields[0])
	}
}

// VerifySchema проверяет критичные таблицы, колонки и индексы до запуска HTTP/workers.
func VerifySchema(ctx context.Context, runner sqliteRunner, requirements []SchemaRequirement) error {
	started := time.Now()
	for _, req := range requirements {
		if strings.TrimSpace(req.Table) == "" {
			continue
		}
		exists, err := sqliteTableExists(ctx, runner, req.Table)
		if err != nil {
			return fmt.Errorf("sqlite schema verification: inspect table %s: %w", req.Table, err)
		}
		if !exists {
			return missingSQLiteSchemaObject(req, "table", "", "")
		}
		for _, column := range req.Columns {
			if strings.TrimSpace(column) == "" {
				continue
			}
			columnExists, err := sqliteColumnExists(ctx, runner, req.Table, column)
			if err != nil {
				return fmt.Errorf("sqlite schema verification: inspect column %s.%s: %w", req.Table, column, err)
			}
			if !columnExists {
				return missingSQLiteSchemaObject(req, "column", column, "")
			}
		}
		for _, index := range req.Indexes {
			if strings.TrimSpace(index) == "" {
				continue
			}
			indexExists, err := sqliteIndexExists(ctx, runner, index)
			if err != nil {
				return fmt.Errorf("sqlite schema verification: inspect index %s: %w", index, err)
			}
			if !indexExists {
				return missingSQLiteSchemaObject(req, "index", "", index)
			}
		}
	}
	slog.InfoContext(ctx, "sqlite schema verification завершена", "operation", "db.schema_verification", "action", "verify_schema", "result", "success", "db_type", "sqlite", "duration_ms", time.Since(started).Milliseconds())
	return nil
}

func missingSQLiteSchemaObject(req SchemaRequirement, objectType, column, index string) error {
	recovery := strings.TrimSpace(req.RecoveryAction)
	if recovery == "" {
		recovery = "run startup migrations with the configured migration directory; for local/dev, recreate the SQLite database from migrations if repair is impossible"
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

func sqliteTableExists(ctx context.Context, runner sqliteRunner, table string) (bool, error) {
	var n int
	err := runner.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
	return n > 0, err
}

func sqliteColumnExists(ctx context.Context, runner sqliteRunner, table, column string) (bool, error) {
	rows, err := runner.QueryContext(ctx, `PRAGMA table_info(`+quoteSQLiteIdent(table)+`)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func sqliteIndexExists(ctx context.Context, runner sqliteRunner, index string) (bool, error) {
	var n int
	err := runner.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&n)
	return n > 0, err
}

func countSQLiteUserTables(ctx context.Context, runner sqliteRunner) (int, error) {
	var n int
	err := runner.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name NOT IN ('schema_migrations','db_runtime_versions')`).Scan(&n)
	return n, err
}

func quoteSQLiteIdent(raw string) string {
	return `"` + strings.ReplaceAll(raw, `"`, `""`) + `"`
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

func backupSQLiteBeforeUpgrade(ctx context.Context, db *sql.DB, dbPath, backupDir, moduleName, previousVersion, targetVersion string) error {
	started := time.Now()
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(FULL)`); err != nil {
		return fmt.Errorf("sqlite backup checkpoint failed: %w", err)
	}
	if strings.TrimSpace(backupDir) == "" {
		backupDir = filepath.Join(filepath.Dir(dbPath), "backups")
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	stem := fmt.Sprintf("%s_%s_to_%s_%s", sanitizeFilenameToken(moduleName), sanitizeFilenameToken(previousVersion), sanitizeFilenameToken(targetVersion), stamp)
	baseTarget := filepath.Join(backupDir, stem+".db")
	copied := 0
	for _, suffix := range []string{"", "-wal", "-shm"} {
		source := dbPath + suffix
		info, err := os.Stat(source)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}
		target := baseTarget + suffix
		if err := copyFile(source, target, info.Mode()); err != nil {
			return err
		}
		copied++
	}
	if copied == 0 {
		return fmt.Errorf("sqlite backup failed: no db files copied for %s", dbPath)
	}
	slog.InfoContext(ctx, "sqlite backup перед миграцией создан", "operation", "db.backup", "action", "backup_before_upgrade", "result", "success", "db_type", "sqlite", "db_path", dbPath, "backup_path", baseTarget, "module_name", moduleName, "current_version", previousVersion, "target_version", targetVersion, "duration_ms", time.Since(started).Milliseconds())
	return nil
}

func copyFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
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
