package postgres

import "testing"

func TestRequiredSchemaIncludesCurrencyReference(t *testing.T) {
	var found bool
	for _, req := range RequiredSchema() {
		if req.Table != "cloud_currency_reference" {
			continue
		}
		found = true
		columns := map[string]bool{}
		for _, column := range req.Columns {
			columns[column] = true
		}
		for _, column := range []string{"currency_code", "currency_alpha_code", "minor_unit", "currency_iso_name", "currency_symbol", "curr_basic_name", "curr_add_name", "show_add", "show_currency_basic_name", "active"} {
			if !columns[column] {
				t.Fatalf("expected cloud_currency_reference.%s in schema verification contract", column)
			}
		}
		indexes := map[string]bool{}
		for _, index := range req.Indexes {
			indexes[index] = true
		}
		if !indexes["cloud_currency_reference_alpha_code_idx"] {
			t.Fatal("expected named currency alpha-code index in schema verification contract")
		}
	}
	if !found {
		t.Fatal("expected cloud_currency_reference in schema verification contract")
	}
}

func TestRequiredSchemaIncludesRuntimeProjectionTables(t *testing.T) {
	requirements := map[string]map[string]bool{}
	for _, req := range RequiredSchema() {
		columns := map[string]bool{}
		for _, column := range req.Columns {
			columns[column] = true
		}
		requirements[req.Table] = columns
	}
	for table, columns := range map[string][]string{
		"cloud_projection_event_type_stats": {
			"restaurant_id", "device_id", "event_type", "event_count", "first_occurred_at", "last_occurred_at",
			"last_cloud_received_at", "last_event_id", "last_command_id", "updated_at",
		},
		"cloud_projection_shift_finance": {
			"restaurant_id", "device_id", "shift_id", "payments_captured_count", "payments_captured_total",
			"checks_created_count", "checks_total_amount", "last_event_id", "last_command_id", "last_occurred_at",
			"last_cloud_received_at", "updated_at",
		},
	} {
		foundColumns, ok := requirements[table]
		if !ok {
			t.Fatalf("expected %s in schema verification contract", table)
		}
		for _, column := range columns {
			if !foundColumns[column] {
				t.Fatalf("expected %s.%s in schema verification contract", table, column)
			}
		}
	}
}
