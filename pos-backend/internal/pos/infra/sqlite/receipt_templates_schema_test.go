package sqlite_test

import (
	"testing"

	possqlite "pos-backend/internal/pos/infra/sqlite"
)

// POS-71: managed baseline создаёт receipt_templates с колонками Edge read model и
// объявляет таблицу в schema verification до старта HTTP/workers.
func TestReceiptTemplatesTableAndSchemaContract(t *testing.T) {
	db, ctx := newSchemaDB(t)

	execSchema(t, ctx, db, `INSERT INTO receipt_templates(id,restaurant_id,document_type,name,content,level,cpl,printer_class,is_default,version,synced_at)
VALUES ('tmpl-1','restaurant-1','precheck','Пречек','{header}',1,48,'generic',1,1,?)`, schemaTestTime)

	var documentType string
	var cpl, isDefault int
	if err := db.QueryRowContext(ctx, `SELECT document_type,cpl,is_default FROM receipt_templates WHERE id = 'tmpl-1'`).Scan(&documentType, &cpl, &isDefault); err != nil {
		t.Fatal(err)
	}
	if documentType != "precheck" || cpl != 48 || isDefault != 1 {
		t.Fatalf("unexpected receipt_templates row: type=%q cpl=%d is_default=%d", documentType, cpl, isDefault)
	}

	// tenant-level default: restaurant_id допускает NULL.
	execSchema(t, ctx, db, `INSERT INTO receipt_templates(id,document_type,name,content,level,cpl,printer_class,is_default,version,synced_at)
VALUES ('tmpl-tenant','ticket','Билет','{header}',1,32,'generic',1,1,?)`, schemaTestTime)

	var found bool
	for _, req := range possqlite.RequiredSchema() {
		if req.Table != "receipt_templates" {
			continue
		}
		found = true
		for _, column := range []string{"id", "restaurant_id", "document_type", "name", "content", "level", "cpl", "printer_class", "is_default", "version", "synced_at"} {
			if !containsString(req.Columns, column) {
				t.Fatalf("expected receipt_templates.%s in schema verification contract", column)
			}
		}
	}
	if !found {
		t.Fatal("expected receipt_templates in schema verification contract")
	}
}
