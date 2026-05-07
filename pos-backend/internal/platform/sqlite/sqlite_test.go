package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	txctx "pos-backend/internal/platform/tx"
)

func TestOpenAppliesAndVerifiesRuntimeGate(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "pos.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	report, err := EnsureRuntimeGate(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ToLower(report.JournalMode) != "wal" {
		t.Fatalf("expected WAL journal mode, got %q", report.JournalMode)
	}
	if report.Synchronous != requiredSynchronousNormal {
		t.Fatalf("expected synchronous NORMAL, got %d", report.Synchronous)
	}
	if report.ForeignKeys != requiredForeignKeysOn {
		t.Fatalf("expected foreign_keys ON, got %d", report.ForeignKeys)
	}
	if report.BusyTimeoutMS < requiredBusyTimeoutMS {
		t.Fatalf("expected busy_timeout at least %d, got %d", requiredBusyTimeoutMS, report.BusyTimeoutMS)
	}
	if !meetsWALPilotBaseline(report.SQLiteVersion) {
		t.Fatalf("expected sqlite_version %s to satisfy WAL pilot baseline", report.SQLiteVersion)
	}
}

func TestValidateRuntimeReportRejectsInvalidBaseline(t *testing.T) {
	valid := RuntimeReport{
		SQLiteVersion: requiredWALPilotSQLiteVersion,
		JournalMode:   "wal",
		Synchronous:   requiredSynchronousNormal,
		ForeignKeys:   requiredForeignKeysOn,
		BusyTimeoutMS: requiredBusyTimeoutMS,
	}
	cases := []struct {
		name      string
		mutate    func(RuntimeReport) RuntimeReport
		wantError string
	}{
		{
			name: "old sqlite version below functional minimum",
			mutate: func(report RuntimeReport) RuntimeReport {
				report.SQLiteVersion = "3.36.0"
				return report
			},
			wantError: "functional minimum",
		},
		{
			name: "sqlite version below wal pilot baseline",
			mutate: func(report RuntimeReport) RuntimeReport {
				report.SQLiteVersion = "3.51.2"
				return report
			},
			wantError: "production WAL pilot baseline",
		},
		{
			name: "journal mode is not wal",
			mutate: func(report RuntimeReport) RuntimeReport {
				report.JournalMode = "delete"
				return report
			},
			wantError: "journal_mode",
		},
		{
			name: "synchronous is not normal",
			mutate: func(report RuntimeReport) RuntimeReport {
				report.Synchronous = 2
				return report
			},
			wantError: "synchronous",
		},
		{
			name: "foreign keys disabled",
			mutate: func(report RuntimeReport) RuntimeReport {
				report.ForeignKeys = 0
				return report
			},
			wantError: "foreign_keys",
		},
		{
			name: "busy timeout is too low",
			mutate: func(report RuntimeReport) RuntimeReport {
				report.BusyTimeoutMS = 1000
				return report
			},
			wantError: "busy_timeout",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRuntimeReport(tc.mutate(valid))
			if err == nil {
				t.Fatal("expected runtime gate error")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error to contain %q, got %q", tc.wantError, err.Error())
			}
		})
	}
}

func TestValidateRuntimeReportAllowsPinnedBackports(t *testing.T) {
	for version := range allowedPinnedBackportSQLiteVersions {
		report := RuntimeReport{
			SQLiteVersion: version,
			JournalMode:   "wal",
			Synchronous:   requiredSynchronousNormal,
			ForeignKeys:   requiredForeignKeysOn,
			BusyTimeoutMS: requiredBusyTimeoutMS,
		}
		if err := validateRuntimeReport(report); err != nil {
			t.Fatalf("expected pinned backport %s to pass runtime gate: %v", version, err)
		}
	}
}

func TestTxManagerUsesBeginImmediate(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "pos.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(ctx, `CREATE TABLE tx_probe (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}

	otherDB, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = otherDB.Close() })
	if _, err := otherDB.ExecContext(ctx, `PRAGMA busy_timeout = 50`); err != nil {
		t.Fatal(err)
	}

	err = NewTxManager(db).WithinTx(ctx, func(ctx context.Context) error {
		writeCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		if _, err := otherDB.ExecContext(writeCtx, `INSERT INTO tx_probe(id) VALUES ('other-writer')`); err == nil {
			return errors.New("expected concurrent writer to be blocked by BEGIN IMMEDIATE")
		}
		tx, ok := txctx.FromContext(ctx)
		if !ok {
			return errors.New("expected transaction in context")
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO tx_probe(id) VALUES ('owner')`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherDB.ExecContext(ctx, `INSERT INTO tx_probe(id) VALUES ('after-commit')`); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateDirWithPolicyWritesRuntimeVersionAndCreatesBackup(t *testing.T) {
	ctx := t.Context()
	dbPath := filepath.Join(t.TempDir(), "policy.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrationsDir := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_init.sql"), []byte(`CREATE TABLE IF NOT EXISTS migration_probe (id TEXT PRIMARY KEY);`), 0o644); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(t.TempDir(), "backups")
	if err := MigrateDirWithPolicy(ctx, db, dbPath, migrationsDir, MigrationOptions{
		ModuleName:    "pos-backend",
		ModuleVersion: "0.1.0",
		BackupDir:     backupDir,
	}); err != nil {
		t.Fatalf("first migrate failed: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO migration_probe(id) VALUES ('seed')`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateDirWithPolicy(ctx, db, dbPath, migrationsDir, MigrationOptions{
		ModuleName:    "pos-backend",
		ModuleVersion: "0.2.0",
		BackupDir:     backupDir,
	}); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}

	var runtimeVersion string
	if err := db.QueryRowContext(ctx, `SELECT module_version FROM db_runtime_versions WHERE module_name = ?`, "pos-backend").Scan(&runtimeVersion); err != nil {
		t.Fatal(err)
	}
	if runtimeVersion != "0.2.0" {
		t.Fatalf("expected stored runtime version 0.2.0, got %s", runtimeVersion)
	}

	backupEntries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backupEntries) == 0 {
		t.Fatal("expected backup files before version upgrade")
	}
}

func TestCompareModuleVersion(t *testing.T) {
	result, err := compareModuleVersion("0.1.0", "0.2.0")
	if err != nil {
		t.Fatalf("compare failed: %v", err)
	}
	if result >= 0 {
		t.Fatalf("expected 0.1.0 < 0.2.0, got %d", result)
	}
	if _, err := compareModuleVersion("invalid", "0.2.0"); err == nil {
		t.Fatal("expected invalid semantic version to fail")
	}
}
