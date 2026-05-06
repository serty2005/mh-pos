package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
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
	for _, name := range names {
		var exists int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, name).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (?)`, name); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
