package postgres

import platformpg "cloud-backend/internal/platform/postgres"

// RequiredSchema returns implemented-now PostgreSQL objects used before Cloud HTTP starts.
func RequiredSchema() []platformpg.SchemaRequirement {
	return []platformpg.SchemaRequirement{
		{
			Table:         "schema_migrations",
			RequiredBy:    "postgres startup migration history",
			MigrationFile: "startup framework ensureSchemaMigrationsTable",
			Columns: []string{
				"version", "checksum_sha256", "status", "applied_at",
			},
		},
		{
			Table:         "db_runtime_versions",
			RequiredBy:    "postgres startup runtime version contract",
			MigrationFile: "startup framework ensureRuntimeVersionTable",
			Columns: []string{
				"module_name", "module_version", "schema_version", "checksum_sha256", "status", "applied_at", "updated_at",
			},
		},
		{
			Table:         "cloud_edge_event_receipts",
			RequiredBy:    "cloudsync postgres repository ReceiveEdgeEvent idempotent receipt storage",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql",
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
			Table:         "cloud_edge_event_raw_payloads",
			RequiredBy:    "cloudsync postgres repository ReceiveEdgeEvent raw payload storage",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql",
			Columns: []string{
				"receipt_id", "raw_payload", "raw_payload_sha256_hex", "created_at",
			},
		},
		{
			Table:         "cloud_operational_events",
			RequiredBy:    "cloudsync postgres repository ReceiveEdgeEvent operational journal",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql",
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
			Table:         "cloud_projection_event_type_stats",
			RequiredBy:    "cloudsync postgres repository applyEventProjections event type stats upsert",
			MigrationFile: "002_projection_event_type_stats.sql, 003_runtime_schema_repair.sql",
			Columns: []string{
				"restaurant_id", "device_id", "event_type", "event_count", "first_occurred_at", "last_occurred_at",
				"last_cloud_received_at", "last_event_id", "last_command_id", "updated_at",
			},
		},
		{
			Table:         "cloud_projection_shift_finance",
			RequiredBy:    "cloudsync postgres repository applyEventProjections shift finance upsert",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql",
			Columns: []string{
				"restaurant_id", "device_id", "shift_id", "payments_captured_count", "payments_captured_total",
				"checks_created_count", "checks_total_amount", "last_event_id", "last_command_id", "last_occurred_at",
				"last_cloud_received_at", "updated_at",
			},
		},
		{
			Table:         "cloud_master_data_packages",
			RequiredBy:    "cloudsync postgres repository master data package storage",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql",
			Columns: []string{
				"stream_name", "node_device_id", "restaurant_id", "sync_mode", "full_snapshot_reason", "cloud_version", "checkpoint_token",
				"cloud_updated_at", "payload_json", "created_at", "updated_at",
			},
			Indexes: []string{"cloud_master_data_packages_stream_updated"},
		},
		{
			Table:         "cloud_currency_reference",
			RequiredBy:    "cloudsync postgres startup currency reference catalog",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql",
			Columns: []string{
				"currency_code", "currency_alpha_code", "minor_unit", "currency_iso_name", "currency_symbol",
				"curr_basic_name", "curr_add_name", "show_add", "show_currency_basic_name", "active", "created_at", "updated_at",
			},
			Indexes: []string{"cloud_currency_reference_alpha_code_idx"},
		},
		{
			Table:         "cloud_roles",
			RequiredBy:    "cloud master-data authority role storage",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "name", "permissions_json", "active", "created_at", "updated_at"},
		},
		{
			Table:         "cloud_employees",
			RequiredBy:    "cloud master-data authority employee lifecycle and PIN credential storage",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "role_id", "name", "status", "pin_hash", "pin_credential_version", "permission_snapshot_json", "suspended_at", "archived_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_employees_restaurant_status"},
		},
		{
			Table:         "cloud_categories",
			RequiredBy:    "cloud master-data authority menu category storage",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "name", "status", "sort_order", "created_at", "updated_at"},
			Indexes:       []string{"cloud_categories_restaurant_status"},
		},
		{
			Table:         "cloud_catalog_items",
			RequiredBy:    "cloud master-data authority catalog item storage",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "kind", "name", "sku", "base_unit", "status", "created_at", "updated_at"},
			Indexes:       []string{"cloud_catalog_items_restaurant_kind_status"},
		},
		{
			Table:         "cloud_dishes",
			RequiredBy:    "cloud catalog dish foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"catalog_item_id", "restaurant_id", "recipe_policy", "updated_at"},
		},
		{
			Table:         "cloud_goods",
			RequiredBy:    "cloud catalog goods/raw material foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"catalog_item_id", "restaurant_id", "stock_tracking_mode", "updated_at"},
		},
		{
			Table:         "cloud_semi_finished_products",
			RequiredBy:    "cloud catalog semi-finished products foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"catalog_item_id", "restaurant_id", "production_unit", "updated_at"},
		},
		{
			Table:         "cloud_recipe_items",
			RequiredBy:    "cloud recipe foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "recipe_owner_catalog_item_id", "component_catalog_item_id", "quantity", "unit", "loss_percent", "created_at", "updated_at"},
		},
		{
			Table:         "cloud_modifier_groups",
			RequiredBy:    "cloud menu modifier group foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "name", "status", "required", "min_count", "max_count", "created_at", "updated_at"},
		},
		{
			Table:         "cloud_modifier_options",
			RequiredBy:    "cloud menu modifier option foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "modifier_group_id", "name", "price_delta", "status", "created_at", "updated_at"},
		},
		{
			Table:         "cloud_menu_items",
			RequiredBy:    "cloud master-data authority menu item storage",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "catalog_item_id", "category_id", "name", "price", "currency", "status", "availability_json", "station_routing_key", "created_at", "updated_at"},
			Indexes:       []string{"cloud_menu_items_restaurant_status"},
		},
		{
			Table:         "cloud_menu_item_modifier_groups",
			RequiredBy:    "future cloud menu modifier assignment foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"menu_item_id", "modifier_group_id", "sort_order"},
		},
		{
			Table:         "cloud_menu_location_assignments",
			RequiredBy:    "future multi-location menu assignment foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"menu_item_id", "location_id", "active"},
		},
		{
			Table:         "cloud_master_data_publications",
			RequiredBy:    "cloud master-data publication versioning and package generation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"id", "restaurant_id", "version", "status", "cloud_version", "published_at", "published_by", "package_json", "package_sha256", "created_at", "updated_at"},
			Indexes:       []string{"cloud_master_data_publications_current"},
		},
	}
}
