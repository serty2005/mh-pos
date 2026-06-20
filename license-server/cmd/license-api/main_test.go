package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupSQLiteFilesCopiesDatabaseSidecars(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "license.db")
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.WriteFile(dbPath+suffix, []byte("data"+suffix), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	backupDir := filepath.Join(dir, "backups")
	if err := backupSQLiteFiles(dbPath, backupDir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("backup dirs=%d err=%v", len(entries), err)
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if _, err := os.Stat(filepath.Join(backupDir, entries[0].Name(), filepath.Base(dbPath+suffix))); err != nil {
			t.Fatal(err)
		}
	}
}
