package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunMaintenanceRequiresForceForVacuum(t *testing.T) {
	err := runMaintenance(t.Context(), maintenanceOptions{
		dbPath: filepath.Join(t.TempDir(), "pos.db"),
		vacuum: true,
	})
	if err == nil {
		t.Fatal("expected explicit force requirement")
	}
}

func TestRunMaintenanceOptimizeCheckpointAndVacuumInto(t *testing.T) {
	ctx := t.Context()
	dbPath := filepath.Join(t.TempDir(), "pos.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode = WAL; CREATE TABLE probe(id TEXT PRIMARY KEY); INSERT INTO probe(id) VALUES ('one');`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	snapshotPath := filepath.Join(t.TempDir(), "snapshot.db")
	if err := runMaintenance(ctx, maintenanceOptions{
		dbPath:      dbPath,
		optimize:    true,
		checkpoint:  true,
		vacuumInto:  snapshotPath,
		forceVacuum: true,
	}); err != nil {
		t.Fatalf("maintenance failed: %v", err)
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("expected VACUUM INTO snapshot: %v", err)
	}
}
