package sqlite_test

import (
	"testing"

	possqlite "pos-backend/internal/pos/infra/sqlite"
)

// POS-84: managed baseline создаёт receipt_printers с Cloud-owned routing config
// и объявляет таблицу в schema verification до старта HTTP/workers.
func TestReceiptPrintersTableAndSchemaContract(t *testing.T) {
	db, ctx := newSchemaDB(t)

	execSchema(t, ctx, db, `INSERT INTO receipt_printers(id,restaurant_id,name,type,address,port,document_types,codepage,paper_cut_type,cpl,is_active,cloud_version,synced_at)
VALUES ('printer-1','restaurant-1','Front','tcp','10.25.1.201',9100,'["precheck","ticket"]','cp437','partial',48,1,7,?)`, schemaTestTime)

	var documentTypes string
	var cpl, active, cloudVersion int
	if err := db.QueryRowContext(ctx, `SELECT document_types,cpl,is_active,cloud_version FROM receipt_printers WHERE id = 'printer-1'`).Scan(&documentTypes, &cpl, &active, &cloudVersion); err != nil {
		t.Fatal(err)
	}
	if documentTypes != `["precheck","ticket"]` || cpl != 48 || active != 1 || cloudVersion != 7 {
		t.Fatalf("unexpected receipt_printers row: document_types=%q cpl=%d active=%d version=%d", documentTypes, cpl, active, cloudVersion)
	}
	execSchema(t, ctx, db, `INSERT INTO receipt_printers(id,restaurant_id,name,type,document_types,codepage,paper_cut_type,cpl,is_active,cloud_version,synced_at)
VALUES ('printer-usb','restaurant-1','USB','usb','["ticket"]','','partial',48,1,7,?)`, schemaTestTime)

	var found bool
	for _, req := range possqlite.RequiredSchema() {
		if req.Table != "receipt_printers" {
			continue
		}
		found = true
		for _, column := range []string{"id", "restaurant_id", "name", "type", "address", "port", "document_types", "codepage", "paper_cut_type", "cpl", "is_active", "cloud_version", "synced_at"} {
			if !containsString(req.Columns, column) {
				t.Fatalf("expected receipt_printers.%s in schema verification contract", column)
			}
		}
		if !containsString(req.Indexes, "receipt_printers_restaurant_active") {
			t.Fatalf("expected receipt_printers_restaurant_active index in schema verification contract, got %+v", req.Indexes)
		}
	}
	if !found {
		t.Fatal("expected receipt_printers in schema verification contract")
	}
}
