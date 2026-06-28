package sqlite_test

import (
	"slices"
	"testing"

	possqlite "pos-backend/internal/pos/infra/sqlite"
)

// POS-74: managed baseline создает print_jobs и startup verification проверяет
// очередь до запуска HTTP server/worker.
func TestPrintJobsSchemaAndVerificationContract(t *testing.T) {
	db, ctx := newSchemaDB(t)
	execSchema(t, ctx, db, `INSERT INTO print_jobs(id,restaurant_id,document_type,source_kind,source_id,status,attempts,max_attempts,printer_class,created_at,updated_at)
VALUES ('job-1','restaurant-1','precheck','precheck','precheck-1','pending',0,3,'generic',?,?)`, schemaTestTime, schemaTestTime)

	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM print_jobs WHERE id = 'job-1'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("expected pending print job, got %q", status)
	}

	for _, req := range possqlite.RequiredSchema() {
		if req.Table != "print_jobs" {
			continue
		}
		for _, column := range []string{"id", "restaurant_id", "document_type", "source_kind", "source_id", "status", "attempts", "max_attempts", "printer_class", "next_attempt_at"} {
			if !slices.Contains(req.Columns, column) {
				t.Fatalf("expected print_jobs.%s in schema verification contract", column)
			}
		}
		if !slices.Contains(req.Indexes, "print_jobs_pending_due") || !slices.Contains(req.Indexes, "print_jobs_restaurant_status_created") {
			t.Fatalf("expected print_jobs indexes in schema verification contract, got %+v", req.Indexes)
		}
		return
	}
	t.Fatal("expected print_jobs in schema verification contract")
}
