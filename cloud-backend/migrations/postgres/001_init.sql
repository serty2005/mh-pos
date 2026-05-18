-- Схлопнутый pre-pilot baseline. До первого клиента dev/test базы пересоздаются,
-- реальные data-preserving migrations начинаются после первого внедрения.

-- === 001_sync_receiver.sql ===
CREATE TABLE IF NOT EXISTS cloud_edge_event_receipts (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  command_id TEXT NOT NULL CHECK (command_id <> ''),
  event_id TEXT NOT NULL CHECK (event_id <> ''),
  edge_event_id TEXT NOT NULL CHECK (edge_event_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type IN (
    'ShiftOpened',
    'ShiftClosed',
    'OrderCreated',
    'OrderLineAdded',
    'OrderLineQuantityChanged',
    'OrderLineVoided',
    'PrecheckIssued',
    'PrecheckReprinted',
    'PrecheckCancelled',
    'CheckCreated',
    'CheckRefunded',
    'CheckReprinted',
    'PaymentCaptured',
    'PaymentRefunded',
    'CancellationRecorded',
    'RefundRecorded',
    'OrderClosed',
    'CashSessionOpened',
    'CashSessionClosed',
    'CashDrawerEventRecorded',
    'AuthSessionStarted',
    'AuthSessionRevoked',
    'DeviceRegistered'
  )),
  aggregate_type TEXT NOT NULL CHECK (aggregate_type <> ''),
  aggregate_id TEXT NOT NULL CHECK (aggregate_id <> ''),
  envelope_version TEXT NOT NULL CHECK (envelope_version = '1'),
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_edge_event_receipts_edge_event_key
  ON cloud_edge_event_receipts(restaurant_id, device_id, edge_event_id);

CREATE INDEX IF NOT EXISTS cloud_edge_event_receipts_event_type_received_at
  ON cloud_edge_event_receipts(event_type, cloud_received_at);

CREATE TABLE IF NOT EXISTS cloud_edge_event_raw_payloads (
  receipt_id TEXT PRIMARY KEY REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  raw_payload JSONB NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_operational_events (
  id TEXT PRIMARY KEY,
  receipt_id TEXT NOT NULL UNIQUE REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  idempotency_key TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  command_id TEXT NOT NULL CHECK (command_id <> ''),
  event_id TEXT NOT NULL CHECK (event_id <> ''),
  edge_event_id TEXT NOT NULL CHECK (edge_event_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  aggregate_type TEXT NOT NULL CHECK (aggregate_type <> ''),
  aggregate_id TEXT NOT NULL CHECK (aggregate_id <> ''),
  envelope_version TEXT NOT NULL CHECK (envelope_version = '1'),
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  replay_status TEXT NOT NULL DEFAULT 'accepted' CHECK (replay_status IN ('accepted')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_operational_events_edge_event_key
  ON cloud_operational_events(restaurant_id, device_id, edge_event_id);

CREATE INDEX IF NOT EXISTS cloud_operational_events_type_received_at
  ON cloud_operational_events(event_type, cloud_received_at);

CREATE INDEX IF NOT EXISTS cloud_operational_events_restaurant_sequence
  ON cloud_operational_events(restaurant_id, device_id, occurred_at, event_id);

CREATE TABLE IF NOT EXISTS cloud_projection_event_type_stats (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  event_count BIGINT NOT NULL CHECK (event_count >= 0),
  first_occurred_at TIMESTAMPTZ NOT NULL,
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, event_type)
);

CREATE TABLE IF NOT EXISTS cloud_projection_shift_finance (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  shift_id TEXT NOT NULL CHECK (shift_id <> ''),
  payments_captured_count BIGINT NOT NULL DEFAULT 0 CHECK (payments_captured_count >= 0),
  payments_captured_total BIGINT NOT NULL DEFAULT 0,
  checks_created_count BIGINT NOT NULL DEFAULT 0 CHECK (checks_created_count >= 0),
  checks_total_amount BIGINT NOT NULL DEFAULT 0,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, shift_id)
);

CREATE TABLE IF NOT EXISTS cloud_master_data_packages (
  stream_name TEXT NOT NULL CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','currencies')),
  node_device_id TEXT NOT NULL DEFAULT '',
  restaurant_id TEXT,
  sync_mode TEXT NOT NULL CHECK (sync_mode IN ('full_snapshot','incremental')),
  full_snapshot_reason TEXT NOT NULL DEFAULT '' CHECK (full_snapshot_reason IN ('','terminal_restaurant_changed','node_role_changed')),
  cloud_version BIGINT NOT NULL CHECK (cloud_version > 0),
  checkpoint_token TEXT,
  cloud_updated_at TIMESTAMPTZ,
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (stream_name, node_device_id)
);

CREATE INDEX IF NOT EXISTS cloud_master_data_packages_stream_updated
  ON cloud_master_data_packages(stream_name, updated_at DESC);

CREATE TABLE IF NOT EXISTS cloud_currency_reference (
  currency_code INTEGER PRIMARY KEY CHECK (currency_code > 0),
  currency_alpha_code TEXT NOT NULL UNIQUE CHECK (currency_alpha_code ~ '^[A-Z]{3}$'),
  minor_unit SMALLINT NOT NULL CHECK (minor_unit BETWEEN 0 AND 4),
  currency_iso_name TEXT NOT NULL CHECK (currency_iso_name <> ''),
  currency_symbol TEXT NOT NULL CHECK (currency_symbol <> ''),
  curr_basic_name TEXT NOT NULL CHECK (curr_basic_name <> ''),
  curr_add_name TEXT NOT NULL CHECK (curr_add_name <> ''),
  show_add BOOLEAN NOT NULL DEFAULT TRUE,
  show_currency_basic_name BOOLEAN NOT NULL DEFAULT TRUE,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_currency_reference_alpha_code_idx
  ON cloud_currency_reference(currency_alpha_code);

-- === 002_projection_event_type_stats.sql ===
CREATE TABLE IF NOT EXISTS cloud_projection_event_type_stats (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  event_count BIGINT NOT NULL CHECK (event_count >= 0),
  first_occurred_at TIMESTAMPTZ NOT NULL,
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, event_type)
);

-- === 003_runtime_schema_repair.sql ===
CREATE TABLE IF NOT EXISTS cloud_edge_event_receipts (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  command_id TEXT NOT NULL CHECK (command_id <> ''),
  event_id TEXT NOT NULL CHECK (event_id <> ''),
  edge_event_id TEXT NOT NULL CHECK (edge_event_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type IN (
    'ShiftOpened',
    'ShiftClosed',
    'OrderCreated',
    'OrderLineAdded',
    'OrderLineQuantityChanged',
    'OrderLineVoided',
    'PrecheckIssued',
    'PrecheckReprinted',
    'PrecheckCancelled',
    'CheckCreated',
    'CheckRefunded',
    'CheckReprinted',
    'PaymentCaptured',
    'PaymentRefunded',
    'CancellationRecorded',
    'RefundRecorded',
    'OrderClosed',
    'CashSessionOpened',
    'CashSessionClosed',
    'CashDrawerEventRecorded',
    'AuthSessionStarted',
    'AuthSessionRevoked',
    'DeviceRegistered'
  )),
  aggregate_type TEXT NOT NULL CHECK (aggregate_type <> ''),
  aggregate_id TEXT NOT NULL CHECK (aggregate_id <> ''),
  envelope_version TEXT NOT NULL CHECK (envelope_version = '1'),
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_edge_event_receipts_edge_event_key
  ON cloud_edge_event_receipts(restaurant_id, device_id, edge_event_id);

CREATE INDEX IF NOT EXISTS cloud_edge_event_receipts_event_type_received_at
  ON cloud_edge_event_receipts(event_type, cloud_received_at);

CREATE TABLE IF NOT EXISTS cloud_edge_event_raw_payloads (
  receipt_id TEXT PRIMARY KEY REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  raw_payload JSONB NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_operational_events (
  id TEXT PRIMARY KEY,
  receipt_id TEXT NOT NULL UNIQUE REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  idempotency_key TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  command_id TEXT NOT NULL CHECK (command_id <> ''),
  event_id TEXT NOT NULL CHECK (event_id <> ''),
  edge_event_id TEXT NOT NULL CHECK (edge_event_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  aggregate_type TEXT NOT NULL CHECK (aggregate_type <> ''),
  aggregate_id TEXT NOT NULL CHECK (aggregate_id <> ''),
  envelope_version TEXT NOT NULL CHECK (envelope_version = '1'),
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  replay_status TEXT NOT NULL DEFAULT 'accepted' CHECK (replay_status IN ('accepted')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_operational_events_edge_event_key
  ON cloud_operational_events(restaurant_id, device_id, edge_event_id);

CREATE INDEX IF NOT EXISTS cloud_operational_events_type_received_at
  ON cloud_operational_events(event_type, cloud_received_at);

CREATE INDEX IF NOT EXISTS cloud_operational_events_restaurant_sequence
  ON cloud_operational_events(restaurant_id, device_id, occurred_at, event_id);

CREATE TABLE IF NOT EXISTS cloud_projection_event_type_stats (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  event_count BIGINT NOT NULL CHECK (event_count >= 0),
  first_occurred_at TIMESTAMPTZ NOT NULL,
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, event_type)
);

CREATE TABLE IF NOT EXISTS cloud_projection_shift_finance (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  shift_id TEXT NOT NULL CHECK (shift_id <> ''),
  payments_captured_count BIGINT NOT NULL DEFAULT 0 CHECK (payments_captured_count >= 0),
  payments_captured_total BIGINT NOT NULL DEFAULT 0,
  checks_created_count BIGINT NOT NULL DEFAULT 0 CHECK (checks_created_count >= 0),
  checks_total_amount BIGINT NOT NULL DEFAULT 0,
  last_event_id TEXT NOT NULL CHECK (last_event_id <> ''),
  last_command_id TEXT NOT NULL CHECK (last_command_id <> ''),
  last_occurred_at TIMESTAMPTZ NOT NULL,
  last_cloud_received_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (restaurant_id, device_id, shift_id)
);

CREATE TABLE IF NOT EXISTS cloud_master_data_packages (
  stream_name TEXT NOT NULL CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','currencies')),
  node_device_id TEXT NOT NULL DEFAULT '',
  restaurant_id TEXT,
  sync_mode TEXT NOT NULL CHECK (sync_mode IN ('full_snapshot','incremental')),
  full_snapshot_reason TEXT NOT NULL DEFAULT '' CHECK (full_snapshot_reason IN ('','terminal_restaurant_changed','node_role_changed')),
  cloud_version BIGINT NOT NULL CHECK (cloud_version > 0),
  checkpoint_token TEXT,
  cloud_updated_at TIMESTAMPTZ,
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (stream_name, node_device_id)
);

ALTER TABLE cloud_master_data_packages
  ADD COLUMN IF NOT EXISTS full_snapshot_reason TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS cloud_master_data_packages_stream_updated
  ON cloud_master_data_packages(stream_name, updated_at DESC);

CREATE TABLE IF NOT EXISTS cloud_currency_reference (
  currency_code INTEGER PRIMARY KEY CHECK (currency_code > 0),
  currency_alpha_code TEXT NOT NULL UNIQUE CHECK (currency_alpha_code ~ '^[A-Z]{3}$'),
  minor_unit SMALLINT NOT NULL CHECK (minor_unit BETWEEN 0 AND 4),
  currency_iso_name TEXT NOT NULL CHECK (currency_iso_name <> ''),
  currency_symbol TEXT NOT NULL CHECK (currency_symbol <> ''),
  curr_basic_name TEXT NOT NULL CHECK (curr_basic_name <> ''),
  curr_add_name TEXT NOT NULL CHECK (curr_add_name <> ''),
  show_add BOOLEAN NOT NULL DEFAULT TRUE,
  show_currency_basic_name BOOLEAN NOT NULL DEFAULT TRUE,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_currency_reference_alpha_code_idx
  ON cloud_currency_reference(currency_alpha_code);

-- === 004_master_data_authority.sql ===
CREATE TABLE IF NOT EXISTS cloud_roles (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  permissions_json JSONB NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, name)
);

CREATE TABLE IF NOT EXISTS cloud_employees (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  role_id TEXT NOT NULL REFERENCES cloud_roles(id),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('active','suspended','archived')),
  pin_hash TEXT NOT NULL CHECK (pin_hash <> ''),
  pin_credential_version BIGINT NOT NULL DEFAULT 1 CHECK (pin_credential_version > 0),
  permission_snapshot_json JSONB NOT NULL,
  suspended_at TIMESTAMPTZ,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_employees_restaurant_status
  ON cloud_employees(restaurant_id, status);

CREATE TABLE IF NOT EXISTS cloud_categories (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  sort_order BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_categories_restaurant_status
  ON cloud_categories(restaurant_id, status, sort_order);

CREATE TABLE IF NOT EXISTS cloud_catalog_items (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  kind TEXT NOT NULL CHECK (kind IN ('dish','good','semi_finished','service')),
  folder_id TEXT,
  name TEXT NOT NULL CHECK (name <> ''),
  sku TEXT NOT NULL CHECK (sku <> ''),
  base_unit TEXT NOT NULL CHECK (base_unit <> ''),
  kitchen_type TEXT,
  accounting_category TEXT,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, sku)
);

CREATE INDEX IF NOT EXISTS cloud_catalog_items_restaurant_kind_status
  ON cloud_catalog_items(restaurant_id, kind, status);

CREATE TABLE IF NOT EXISTS cloud_dishes (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  recipe_policy TEXT NOT NULL DEFAULT 'none' CHECK (recipe_policy IN ('none','optional','required')),
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_goods (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  stock_tracking_mode TEXT NOT NULL DEFAULT 'none' CHECK (stock_tracking_mode IN ('none','quantity')),
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_semi_finished_products (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  production_unit TEXT NOT NULL DEFAULT 'portion',
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_services (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  fixed_unit TEXT NOT NULL DEFAULT 'service',
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_catalog_folders (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  parent_id TEXT REFERENCES cloud_catalog_folders(id),
  name TEXT NOT NULL CHECK (name <> ''),
  sort_order BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_catalog_folders_parent_sort
  ON cloud_catalog_folders(restaurant_id, parent_id, sort_order, id);

CREATE TABLE IF NOT EXISTS cloud_catalog_folder_parameters (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  folder_id TEXT NOT NULL REFERENCES cloud_catalog_folders(id) ON DELETE CASCADE,
  parameter_key TEXT NOT NULL CHECK (parameter_key <> ''),
  value_type TEXT NOT NULL CHECK (value_type <> ''),
  value_json JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (folder_id, parameter_key)
);

CREATE TABLE IF NOT EXISTS cloud_catalog_tags (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  code TEXT NOT NULL CHECK (code <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, code)
);

CREATE TABLE IF NOT EXISTS cloud_catalog_item_tags (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES cloud_catalog_tags(id) ON DELETE CASCADE,
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (catalog_item_id, tag_id)
);

CREATE TABLE IF NOT EXISTS cloud_recipe_items (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  recipe_owner_catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  component_catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  quantity BIGINT NOT NULL CHECK (quantity > 0),
  unit TEXT NOT NULL CHECK (unit <> ''),
  loss_percent BIGINT NOT NULL DEFAULT 0 CHECK (loss_percent >= 0 AND loss_percent <= 100),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (recipe_owner_catalog_item_id, component_catalog_item_id)
);

CREATE TABLE IF NOT EXISTS cloud_modifier_groups (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  required BOOLEAN NOT NULL DEFAULT false,
  min_count BIGINT NOT NULL DEFAULT 0 CHECK (min_count >= 0),
  max_count BIGINT NOT NULL DEFAULT 1 CHECK (max_count >= 0),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_modifier_options (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  modifier_group_id TEXT NOT NULL REFERENCES cloud_modifier_groups(id),
  name TEXT NOT NULL CHECK (name <> ''),
  price_minor BIGINT NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_modifier_group_bindings (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  modifier_group_id TEXT NOT NULL REFERENCES cloud_modifier_groups(id) ON DELETE CASCADE,
  target_type TEXT NOT NULL CHECK (target_type IN ('menu_item','catalog_item','folder','tag')),
  target_id TEXT NOT NULL CHECK (target_id <> ''),
  sort_order BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (modifier_group_id, target_type, target_id)
);

CREATE TABLE IF NOT EXISTS cloud_pricing_policies (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  kind TEXT NOT NULL CHECK (kind IN ('discount','surcharge')),
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points BIGINT NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  application_index BIGINT NOT NULL CHECK (application_index > 0),
  manual BOOLEAN NOT NULL DEFAULT false,
  requires_permission TEXT,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (
    (amount_kind = 'fixed' AND amount_minor >= 0 AND value_basis_points = 0)
    OR (amount_kind = 'percentage' AND value_basis_points > 0 AND amount_minor = 0)
  )
);

CREATE INDEX IF NOT EXISTS cloud_pricing_policies_restaurant_active
  ON cloud_pricing_policies(restaurant_id, status, application_index);

CREATE TABLE IF NOT EXISTS cloud_menu_items (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  category_id TEXT REFERENCES cloud_categories(id),
  name TEXT NOT NULL CHECK (name <> ''),
  price BIGINT NOT NULL CHECK (price >= 0),
  currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  availability_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  station_routing_key TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_menu_items_restaurant_status
  ON cloud_menu_items(restaurant_id, status);

CREATE TABLE IF NOT EXISTS cloud_menu_item_modifier_groups (
  menu_item_id TEXT NOT NULL REFERENCES cloud_menu_items(id) ON DELETE CASCADE,
  modifier_group_id TEXT NOT NULL REFERENCES cloud_modifier_groups(id) ON DELETE RESTRICT,
  sort_order BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (menu_item_id, modifier_group_id)
);

CREATE TABLE IF NOT EXISTS cloud_menu_location_assignments (
  menu_item_id TEXT NOT NULL REFERENCES cloud_menu_items(id) ON DELETE CASCADE,
  location_id TEXT NOT NULL CHECK (location_id <> ''),
  active BOOLEAN NOT NULL DEFAULT true,
  PRIMARY KEY (menu_item_id, location_id)
);

CREATE TABLE IF NOT EXISTS cloud_master_data_publications (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  version BIGINT NOT NULL CHECK (version > 0),
  status TEXT NOT NULL CHECK (status IN ('published','archived')),
  cloud_version BIGINT NOT NULL CHECK (cloud_version > 0),
  published_at TIMESTAMPTZ NOT NULL,
  published_by TEXT NOT NULL CHECK (published_by <> ''),
  package_json JSONB NOT NULL,
  package_sha256 TEXT NOT NULL CHECK (package_sha256 <> ''),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, version)
);

CREATE INDEX IF NOT EXISTS cloud_master_data_publications_current
  ON cloud_master_data_publications(restaurant_id, version DESC)
  WHERE status = 'published';

-- === 005_master_data_restaurants_api.sql ===
CREATE TABLE IF NOT EXISTS cloud_restaurants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL CHECK (name <> ''),
  timezone TEXT NOT NULL CHECK (timezone <> ''),
  currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  business_day_mode TEXT NOT NULL CHECK (business_day_mode IN ('standard','24_7')),
  business_day_boundary_local_time TEXT NOT NULL CHECK (business_day_boundary_local_time ~ '^[0-2][0-9]:[0-5][0-9]'),
  status TEXT NOT NULL CHECK (status IN ('active','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_restaurants_status_updated
  ON cloud_restaurants(status, updated_at DESC);

ALTER TABLE cloud_roles
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_roles
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE cloud_employees
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_catalog_items
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_catalog_items
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE cloud_catalog_items
  DROP CONSTRAINT IF EXISTS cloud_catalog_items_kind_check;

UPDATE cloud_catalog_items
SET kind = 'good'
WHERE kind = 'raw_material';

ALTER TABLE cloud_catalog_items
  ADD CONSTRAINT cloud_catalog_items_kind_check
  CHECK (kind IN ('dish','good','semi_finished','service'));

ALTER TABLE cloud_menu_items
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_menu_items
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE cloud_catalog_items
  DROP CONSTRAINT IF EXISTS cloud_catalog_items_restaurant_id_sku_key;

CREATE UNIQUE INDEX IF NOT EXISTS cloud_catalog_items_active_sku
  ON cloud_catalog_items(restaurant_id, sku)
  WHERE status <> 'archived';

-- === 006_zero_to_cashier_provisioning.sql ===
CREATE TABLE IF NOT EXISTS cloud_halls (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES cloud_restaurants(id),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_halls_active_name
  ON cloud_halls(restaurant_id, name)
  WHERE status <> 'archived';

CREATE TABLE IF NOT EXISTS cloud_tables (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES cloud_restaurants(id),
  hall_id TEXT NOT NULL REFERENCES cloud_halls(id),
  name TEXT NOT NULL CHECK (name <> ''),
  seats BIGINT NOT NULL DEFAULT 0 CHECK (seats >= 0),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_tables_active_name
  ON cloud_tables(hall_id, name)
  WHERE status <> 'archived';

CREATE TABLE IF NOT EXISTS cloud_edge_nodes (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT REFERENCES cloud_restaurants(id),
  node_device_id TEXT NOT NULL UNIQUE CHECK (node_device_id <> ''),
  display_name TEXT NOT NULL CHECK (display_name <> ''),
  status TEXT NOT NULL CHECK (status IN ('unassigned','assigned','revoked')),
  credentials_hash TEXT CHECK (credentials_hash IS NULL OR credentials_hash <> ''),
  last_seen_at TIMESTAMPTZ,
  assigned_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_edge_nodes_restaurant_status
  ON cloud_edge_nodes(restaurant_id, status);

CREATE TABLE IF NOT EXISTS cloud_unassigned_edge_nodes (
  id TEXT PRIMARY KEY,
  node_device_id TEXT NOT NULL UNIQUE CHECK (node_device_id <> ''),
  claimed_cloud_url TEXT NOT NULL CHECK (claimed_cloud_url <> ''),
  display_name TEXT NOT NULL CHECK (display_name <> ''),
  app_version TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('pending','assigned','rejected','expired')),
  first_seen_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  assigned_restaurant_id TEXT REFERENCES cloud_restaurants(id),
  assigned_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_unassigned_edge_nodes_status_seen
  ON cloud_unassigned_edge_nodes(status, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS cloud_pairing_codes (
  id TEXT PRIMARY KEY,
  pairing_code_hash TEXT NOT NULL UNIQUE CHECK (pairing_code_hash <> ''),
  restaurant_id TEXT NOT NULL REFERENCES cloud_restaurants(id),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  cloud_url TEXT NOT NULL CHECK (cloud_url <> ''),
  status TEXT NOT NULL CHECK (status IN ('active','consumed','expired','revoked')),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_pairing_codes_restaurant_status
  ON cloud_pairing_codes(restaurant_id, status, expires_at);

-- === 007_refund_and_pricing_policy_hardening.sql ===
ALTER TABLE cloud_edge_event_receipts
  DROP CONSTRAINT IF EXISTS cloud_edge_event_receipts_event_type_check;

ALTER TABLE cloud_edge_event_receipts
  ADD CONSTRAINT cloud_edge_event_receipts_event_type_check CHECK (event_type IN (
    'ShiftOpened',
    'ShiftClosed',
    'OrderCreated',
    'OrderLineAdded',
    'OrderLineQuantityChanged',
    'OrderLineVoided',
    'PrecheckIssued',
    'PrecheckReprinted',
    'PrecheckCancelled',
    'CheckCreated',
    'CheckRefunded',
    'CheckReprinted',
    'PaymentCaptured',
    'PaymentRefunded',
    'CancellationRecorded',
    'RefundRecorded',
    'OrderClosed',
    'CashSessionOpened',
    'CashSessionClosed',
    'CashDrawerEventRecorded',
    'AuthSessionStarted',
    'AuthSessionRevoked',
    'DeviceRegistered'
  ));

ALTER TABLE cloud_projection_shift_finance
  ADD COLUMN IF NOT EXISTS payments_refunded_count BIGINT NOT NULL DEFAULT 0 CHECK (payments_refunded_count >= 0),
  ADD COLUMN IF NOT EXISTS payments_refunded_total BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS checks_refunded_count BIGINT NOT NULL DEFAULT 0 CHECK (checks_refunded_count >= 0),
  ADD COLUMN IF NOT EXISTS checks_refunded_total BIGINT NOT NULL DEFAULT 0;

ALTER TABLE cloud_master_data_packages
  DROP CONSTRAINT IF EXISTS cloud_master_data_packages_stream_name_check;

ALTER TABLE cloud_master_data_packages
  ADD CONSTRAINT cloud_master_data_packages_stream_name_check CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','currencies'));

-- === 008_catalog_v2_modifiers_pricing_policy.sql ===
ALTER TABLE cloud_catalog_items
  DROP CONSTRAINT IF EXISTS cloud_catalog_items_kind_check;

ALTER TABLE cloud_catalog_items
  ADD COLUMN IF NOT EXISTS folder_id TEXT,
  ADD COLUMN IF NOT EXISTS kitchen_type TEXT,
  ADD COLUMN IF NOT EXISTS accounting_category TEXT;

ALTER TABLE cloud_catalog_items
  ADD CONSTRAINT cloud_catalog_items_kind_check
  CHECK (kind IN ('dish','good','semi_finished','service'));

CREATE TABLE IF NOT EXISTS cloud_catalog_folders (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  parent_id TEXT REFERENCES cloud_catalog_folders(id),
  name TEXT NOT NULL CHECK (name <> ''),
  sort_order BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_catalog_folders_parent_sort
  ON cloud_catalog_folders(restaurant_id, parent_id, sort_order, id);

CREATE TABLE IF NOT EXISTS cloud_catalog_folder_parameters (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  folder_id TEXT NOT NULL REFERENCES cloud_catalog_folders(id) ON DELETE CASCADE,
  parameter_key TEXT NOT NULL CHECK (parameter_key <> ''),
  value_type TEXT NOT NULL CHECK (value_type <> ''),
  value_json JSONB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (folder_id, parameter_key)
);

CREATE TABLE IF NOT EXISTS cloud_catalog_tags (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  code TEXT NOT NULL CHECK (code <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, code)
);

CREATE TABLE IF NOT EXISTS cloud_catalog_item_tags (
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES cloud_catalog_tags(id) ON DELETE CASCADE,
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (catalog_item_id, tag_id)
);

CREATE TABLE IF NOT EXISTS cloud_services (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  fixed_unit TEXT NOT NULL DEFAULT 'service',
  updated_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE cloud_modifier_groups
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE cloud_modifier_options
  ADD COLUMN IF NOT EXISTS price_minor BIGINT NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'cloud_modifier_options' AND column_name = 'price_delta'
  ) THEN
    EXECUTE 'UPDATE cloud_modifier_options SET price_minor = price_delta';
  END IF;
END $$;

ALTER TABLE cloud_modifier_options
  DROP COLUMN IF EXISTS price_delta;

CREATE TABLE IF NOT EXISTS cloud_modifier_group_bindings (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  modifier_group_id TEXT NOT NULL REFERENCES cloud_modifier_groups(id) ON DELETE CASCADE,
  target_type TEXT NOT NULL CHECK (target_type IN ('menu_item','catalog_item','folder','tag')),
  target_id TEXT NOT NULL CHECK (target_id <> ''),
  sort_order BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (modifier_group_id, target_type, target_id)
);

CREATE TABLE IF NOT EXISTS cloud_pricing_policies (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  kind TEXT NOT NULL CHECK (kind IN ('discount','surcharge')),
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor BIGINT NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points BIGINT NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  application_index BIGINT NOT NULL CHECK (application_index > 0),
  manual BOOLEAN NOT NULL DEFAULT false,
  requires_permission TEXT,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (
    (amount_kind = 'fixed' AND amount_minor >= 0 AND value_basis_points = 0)
    OR (amount_kind = 'percentage' AND value_basis_points > 0 AND amount_minor = 0)
  )
);

ALTER TABLE cloud_master_data_packages
  DROP CONSTRAINT IF EXISTS cloud_master_data_packages_stream_name_check;

ALTER TABLE cloud_master_data_packages
  ADD CONSTRAINT cloud_master_data_packages_stream_name_check CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','currencies'));
