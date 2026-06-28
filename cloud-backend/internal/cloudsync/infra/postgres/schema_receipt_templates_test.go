package postgres

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// POS-71: managed baseline cloud_receipt_templates обязана проходить schema verification
// до запуска HTTP/workers, иначе stream receipt_templates соберёт пакет на отсутствующей таблице.
func TestRequiredSchemaIncludesReceiptTemplates(t *testing.T) {
	for _, req := range RequiredSchema() {
		if req.Table != "cloud_receipt_templates" {
			continue
		}
		for _, column := range []string{"id", "org_id", "restaurant_id", "document_type", "content", "level", "cpl", "printer_class", "is_default", "version", "is_active"} {
			if !slices.Contains(req.Columns, column) {
				t.Fatalf("expected cloud_receipt_templates.%s in schema verification contract", column)
			}
		}
		if !slices.Contains(req.Indexes, "cloud_receipt_templates_default_uq") {
			t.Fatalf("expected partial default unique index in schema verification contract: %+v", req)
		}
		return
	}
	t.Fatal("expected cloud_receipt_templates in schema verification contract")
}

func TestBaselineAllowsReceiptTemplatesAndPrintersPackageStreams(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "migrations", "postgres", "001_init.sql"))
	if err != nil {
		t.Fatalf("read postgres baseline: %v", err)
	}
	sql := string(raw)
	if !strings.Contains(sql, "cloud_master_data_packages_stream_name_check CHECK") ||
		!strings.Contains(sql, "'receipt_templates'") ||
		!strings.Contains(sql, "'printers'") {
		t.Fatal("expected cloud_master_data_packages stream constraint to allow receipt_templates and printers")
	}
}
