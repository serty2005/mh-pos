package postgres

import platformpg "cloud-backend/internal/platform/postgres"

// RequiredSchema возвращает минимальный набор объектов PostgreSQL, которые cloud runtime использует до старта HTTP.
func RequiredSchema() []platformpg.SchemaRequirement {
	return []platformpg.SchemaRequirement{
		{
			Table: "schema_migrations",
			Columns: []string{
				"version", "checksum_sha256", "status", "applied_at",
			},
		},
		{
			Table: "db_runtime_versions",
			Columns: []string{
				"module_name", "module_version", "schema_version", "checksum_sha256", "status", "applied_at", "updated_at",
			},
		},
		{
			Table: "cloud_edge_event_receipts",
			Columns: []string{
				"id", "idempotency_key", "restaurant_id", "device_id", "command_id", "event_id", "edge_event_id",
				"event_type", "aggregate_type", "aggregate_id", "envelope_version", "occurred_at", "cloud_received_at",
				"raw_payload_sha256_hex", "created_at",
			},
			Indexes: []string{
				"cloud_edge_event_receipts_edge_event_key", "cloud_edge_event_receipts_event_type_received_at",
			},
		},
		{
			Table: "cloud_edge_event_raw_payloads",
			Columns: []string{
				"receipt_id", "raw_payload", "raw_payload_sha256_hex", "created_at",
			},
		},
		{
			Table: "cloud_operational_events",
			Columns: []string{
				"id", "receipt_id", "idempotency_key", "restaurant_id", "device_id", "command_id", "event_id", "edge_event_id",
				"event_type", "aggregate_type", "aggregate_id", "envelope_version", "occurred_at", "cloud_received_at",
				"raw_payload_sha256_hex", "replay_status", "created_at",
			},
			Indexes: []string{
				"cloud_operational_events_edge_event_key", "cloud_operational_events_type_received_at", "cloud_operational_events_restaurant_sequence",
			},
		},
		{
			Table: "cloud_projection_event_type_stats",
			Columns: []string{
				"restaurant_id", "device_id", "event_type", "event_count", "first_occurred_at", "last_occurred_at",
				"last_cloud_received_at", "last_event_id", "last_command_id", "updated_at",
			},
		},
		{
			Table: "cloud_projection_shift_finance",
			Columns: []string{
				"restaurant_id", "device_id", "shift_id", "payments_captured_count", "payments_captured_total",
				"checks_created_count", "checks_total_amount", "last_event_id", "last_command_id", "last_occurred_at",
				"last_cloud_received_at", "updated_at",
			},
		},
		{
			Table: "cloud_master_data_packages",
			Columns: []string{
				"stream_name", "node_device_id", "restaurant_id", "sync_mode", "cloud_version", "checkpoint_token",
				"cloud_updated_at", "payload_json", "created_at", "updated_at",
			},
			Indexes: []string{"cloud_master_data_packages_stream_updated"},
		},
		{
			Table: "cloud_currency_reference",
			Columns: []string{
				"currency_code", "currency_alpha_code", "minor_unit", "currency_iso_name", "currency_symbol",
				"curr_basic_name", "curr_add_name", "show_add", "show_currency_basic_name", "active", "created_at", "updated_at",
			},
			Indexes: []string{"cloud_currency_reference_alpha_code_idx"},
		},
	}
}
