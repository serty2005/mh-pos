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
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql, 007_refund_and_pricing_policy_hardening.sql",
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
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql, 007_refund_and_pricing_policy_hardening.sql",
			Columns: []string{
				"restaurant_id", "device_id", "shift_id", "payments_captured_count", "payments_captured_total",
				"payments_refunded_count", "payments_refunded_total", "checks_created_count", "checks_total_amount",
				"checks_refunded_count", "checks_refunded_total", "last_event_id", "last_command_id",
				"last_occurred_at", "last_cloud_received_at", "updated_at",
			},
		},
		{
			Table:         "cloud_master_data_packages",
			RequiredBy:    "cloudsync postgres repository master data package storage",
			MigrationFile: "001_sync_receiver.sql, 003_runtime_schema_repair.sql, 007_refund_and_pricing_policy_hardening.sql",
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
			Table:         "cloud_restaurants",
			RequiredBy:    "cloud restaurant onboarding and master-data restaurants publication",
			MigrationFile: "005_master_data_restaurants_api.sql",
			Columns:       []string{"id", "name", "timezone", "currency", "business_day_mode", "business_day_boundary_local_time", "status", "cloud_version", "archived_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_restaurants_status_updated"},
		},
		{
			Table:         "cloud_roles",
			RequiredBy:    "cloud master-data authority role storage",
			MigrationFile: "004_master_data_authority.sql, 005_master_data_restaurants_api.sql",
			Columns:       []string{"id", "restaurant_id", "name", "permissions_json", "active", "cloud_version", "archived_at", "created_at", "updated_at"},
		},
		{
			Table:         "cloud_employees",
			RequiredBy:    "cloud master-data authority employee lifecycle and PIN credential storage",
			MigrationFile: "004_master_data_authority.sql, 005_master_data_restaurants_api.sql",
			Columns:       []string{"id", "restaurant_id", "role_id", "name", "status", "pin_hash", "pin_credential_version", "permission_snapshot_json", "cloud_version", "suspended_at", "archived_at", "created_at", "updated_at"},
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
			MigrationFile: "004_master_data_authority.sql, 005_master_data_restaurants_api.sql",
			Columns:       []string{"id", "restaurant_id", "kind", "name", "sku", "base_unit", "status", "cloud_version", "archived_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_catalog_items_restaurant_kind_status", "cloud_catalog_items_active_sku"},
		},
		{
			Table:         "cloud_dishes",
			RequiredBy:    "cloud catalog dish foundation",
			MigrationFile: "004_master_data_authority.sql",
			Columns:       []string{"catalog_item_id", "restaurant_id", "recipe_policy", "updated_at"},
		},
		{
			Table:         "cloud_goods",
			RequiredBy:    "cloud catalog goods/ingredient foundation",
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
			MigrationFile: "004_master_data_authority.sql, 005_master_data_restaurants_api.sql",
			Columns:       []string{"id", "restaurant_id", "catalog_item_id", "category_id", "name", "price", "currency", "status", "availability_json", "station_routing_key", "cloud_version", "archived_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_menu_items_restaurant_status"},
		},
		{
			Table:         "cloud_halls",
			RequiredBy:    "cloud floor master-data publication",
			MigrationFile: "006_zero_to_cashier_provisioning.sql",
			Columns:       []string{"id", "restaurant_id", "name", "status", "cloud_version", "archived_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_halls_active_name"},
		},
		{
			Table:         "cloud_tables",
			RequiredBy:    "cloud floor master-data publication",
			MigrationFile: "006_zero_to_cashier_provisioning.sql",
			Columns:       []string{"id", "restaurant_id", "hall_id", "name", "seats", "status", "cloud_version", "archived_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_tables_active_name"},
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
		{
			Table:         "cloud_edge_nodes",
			RequiredBy:    "cloud edge device provisioning",
			MigrationFile: "006_zero_to_cashier_provisioning.sql",
			Columns:       []string{"id", "restaurant_id", "node_device_id", "display_name", "status", "credentials_hash", "last_seen_at", "assigned_at", "revoked_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_edge_nodes_restaurant_status"},
		},
		{
			Table:         "cloud_unassigned_edge_nodes",
			RequiredBy:    "cloud edge device pending approval queue",
			MigrationFile: "006_zero_to_cashier_provisioning.sql",
			Columns:       []string{"id", "node_device_id", "claimed_cloud_url", "display_name", "app_version", "status", "first_seen_at", "last_seen_at", "assigned_restaurant_id", "assigned_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_unassigned_edge_nodes_status_seen"},
		},
		{
			Table:         "cloud_pairing_codes",
			RequiredBy:    "cloud license server pairing code registration",
			MigrationFile: "006_zero_to_cashier_provisioning.sql",
			Columns:       []string{"id", "pairing_code_hash", "restaurant_id", "node_device_id", "cloud_url", "status", "expires_at", "consumed_at", "created_at", "updated_at"},
			Indexes:       []string{"cloud_pairing_codes_restaurant_status"},
		},
	}
}
