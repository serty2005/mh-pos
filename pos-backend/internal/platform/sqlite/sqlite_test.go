package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
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
