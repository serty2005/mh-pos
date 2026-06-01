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
    'CheckClosed',
    'KitchenTicketStatusChanged',
    'ItemServed',
    'StockReceiptCaptured',
    'InventoryCountCaptured',
    'StockWriteOffCaptured',
    'ProductionCompleted',
    'StopListUpdated',
    'CatalogItemChangeSuggested',
    'RecipeChangeSuggested',
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

CREATE TABLE IF NOT EXISTS inbox_events (
  id TEXT PRIMARY KEY REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  receipt_id TEXT NOT NULL UNIQUE REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  idempotency_key TEXT NOT NULL UNIQUE,
  tenant_id TEXT NOT NULL CHECK (tenant_id <> ''),
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  employee_id TEXT NOT NULL DEFAULT '',
  command_id TEXT NOT NULL CHECK (command_id <> ''),
  event_id TEXT NOT NULL CHECK (event_id <> ''),
  edge_event_id TEXT NOT NULL CHECK (edge_event_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  aggregate_type TEXT NOT NULL CHECK (aggregate_type <> ''),
  aggregate_id TEXT NOT NULL CHECK (aggregate_id <> ''),
  envelope_version TEXT NOT NULL CHECK (envelope_version = '1'),
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload JSONB NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  processed_for_olap BOOLEAN NOT NULL DEFAULT false,
  olap_export_status TEXT NOT NULL DEFAULT 'pending' CHECK (olap_export_status IN ('pending','processing','processed','failed')),
  olap_export_attempts BIGINT NOT NULL DEFAULT 0 CHECK (olap_export_attempts >= 0),
  olap_next_retry_at TIMESTAMPTZ,
  olap_locked_at TIMESTAMPTZ,
  olap_locked_by TEXT,
  olap_processed_at TIMESTAMPTZ,
  olap_last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS inbox_events_event_unique
  ON inbox_events(restaurant_id, device_id, event_id);

CREATE INDEX IF NOT EXISTS inbox_events_olap_pending
  ON inbox_events(processed_for_olap, olap_export_status, olap_next_retry_at, cloud_received_at, id);

CREATE INDEX IF NOT EXISTS inbox_events_restaurant_received
  ON inbox_events(restaurant_id, cloud_received_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS inbox_events_event_type_received
  ON inbox_events(event_type, cloud_received_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS olap_export_checkpoints (
  id TEXT PRIMARY KEY,
  worker_id TEXT NOT NULL DEFAULT '',
  last_exported_inbox_id TEXT NOT NULL DEFAULT '',
  last_exported_event_id TEXT NOT NULL DEFAULT '',
  last_exported_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  consecutive_failures BIGINT NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
  next_retry_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE olap_export_checkpoints
  ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS olap_export_retry_commands (
  command_id TEXT PRIMARY KEY CHECK (command_id <> ''),
  stream TEXT NOT NULL CHECK (stream IN ('raw_business_events','stock_moves')),
  mode TEXT NOT NULL CHECK (mode IN ('retry_failed','resume_from_checkpoint')),
  reason TEXT NOT NULL CHECK (reason <> ''),
  accepted BOOLEAN NOT NULL DEFAULT true,
  checkpoint_before TEXT NOT NULL DEFAULT '',
  retry_requested_at TIMESTAMPTZ NOT NULL,
  pending_count BIGINT NOT NULL DEFAULT 0 CHECK (pending_count >= 0),
  failed_count BIGINT NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS olap_export_retry_commands_stream_created
  ON olap_export_retry_commands(stream, created_at DESC);

CREATE TABLE IF NOT EXISTS olap_backfill_jobs (
  id TEXT PRIMARY KEY CHECK (id <> ''),
  command_id TEXT NOT NULL UNIQUE CHECK (command_id <> ''),
  stream TEXT NOT NULL CHECK (stream IN ('raw_business_events','stock_moves')),
  status TEXT NOT NULL CHECK (status IN ('queued','running','completed','failed','cancelled')),
  requested_from TIMESTAMPTZ,
  requested_to TIMESTAMPTZ,
  checkpoint_cursor TEXT NOT NULL DEFAULT '',
  batch_size INTEGER NOT NULL DEFAULT 1000 CHECK (batch_size > 0),
  total_rows BIGINT NOT NULL DEFAULT 0 CHECK (total_rows >= 0),
  processed_rows BIGINT NOT NULL DEFAULT 0 CHECK (processed_rows >= 0),
  last_error TEXT NOT NULL DEFAULT '',
  cancel_requested BOOLEAN NOT NULL DEFAULT false,
  reason TEXT NOT NULL CHECK (reason <> ''),
  requested_by TEXT NOT NULL DEFAULT '',
  locked_by TEXT NOT NULL DEFAULT '',
  locked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS olap_backfill_jobs_stream_status_created
  ON olap_backfill_jobs(stream, status, created_at DESC);

CREATE TABLE IF NOT EXISTS olap_operator_audit_events (
  id TEXT PRIMARY KEY CHECK (id <> ''),
  command_id TEXT NOT NULL CHECK (command_id <> ''),
  action TEXT NOT NULL CHECK (action IN ('create_backfill_job','cancel_backfill_job')),
  stream TEXT NOT NULL CHECK (stream IN ('raw_business_events','stock_moves')),
  job_id TEXT NOT NULL REFERENCES olap_backfill_jobs(id) ON DELETE RESTRICT,
  requested_by TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS olap_operator_audit_events_job_created
  ON olap_operator_audit_events(job_id, created_at DESC);

CREATE TABLE IF NOT EXISTS cloud_sync_problem_events (
  id TEXT PRIMARY KEY,
  direction TEXT NOT NULL CHECK (direction IN ('edge_to_cloud','cloud_to_edge')),
  node_device_id TEXT,
  restaurant_id TEXT,
  client_item_id TEXT,
  error_code TEXT NOT NULL CHECK (error_code <> ''),
  error_message TEXT NOT NULL CHECK (error_message <> ''),
  raw_payload TEXT NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_sync_problem_events_created_at
  ON cloud_sync_problem_events(created_at DESC);

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

CREATE TABLE IF NOT EXISTS cloud_projection_financial_operations (
  operation_id TEXT PRIMARY KEY CHECK (operation_id <> ''),
  edge_operation_id TEXT NOT NULL CHECK (edge_operation_id <> ''),
  event_id TEXT NOT NULL UNIQUE CHECK (event_id <> ''),
  receipt_id TEXT NOT NULL UNIQUE REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  node_device_id TEXT CHECK (node_device_id IS NULL OR node_device_id <> ''),
  client_device_id TEXT CHECK (client_device_id IS NULL OR client_device_id <> ''),
  actor_employee_id TEXT CHECK (actor_employee_id IS NULL OR actor_employee_id <> ''),
  session_id TEXT CHECK (session_id IS NULL OR session_id <> ''),
  shift_id TEXT NOT NULL CHECK (shift_id <> ''),
  original_shift_id TEXT NOT NULL CHECK (original_shift_id <> ''),
  check_id TEXT NOT NULL CHECK (check_id <> ''),
  precheck_id TEXT NOT NULL CHECK (precheck_id <> ''),
  operation_type TEXT NOT NULL CHECK (operation_type IN ('cancellation','refund')),
  operation_kind TEXT NOT NULL CHECK (operation_kind IN ('full','partial')),
  amount BIGINT NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  business_date_local TEXT NOT NULL CHECK (business_date_local ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'),
  inventory_disposition TEXT NOT NULL CHECK (inventory_disposition IN ('no_stock_effect','return_to_stock','write_off_waste','manual_review')),
  reason TEXT NOT NULL CHECK (reason <> ''),
  created_by_employee_id TEXT CHECK (created_by_employee_id IS NULL OR created_by_employee_id <> ''),
  approved_by_employee_id TEXT CHECK (approved_by_employee_id IS NULL OR approved_by_employee_id <> ''),
  snapshot_json JSONB NOT NULL,
  operation_created_at TIMESTAMPTZ NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_projection_financial_operations_edge_operation
  ON cloud_projection_financial_operations(restaurant_id, device_id, edge_operation_id);

CREATE INDEX IF NOT EXISTS cloud_projection_financial_operations_restaurant_date_type
  ON cloud_projection_financial_operations(restaurant_id, business_date_local, operation_type, operation_created_at DESC);

CREATE INDEX IF NOT EXISTS cloud_projection_financial_operations_shift
  ON cloud_projection_financial_operations(restaurant_id, shift_id, operation_created_at DESC);

CREATE INDEX IF NOT EXISTS cloud_projection_financial_operations_original_shift
  ON cloud_projection_financial_operations(restaurant_id, original_shift_id, operation_created_at DESC);

CREATE INDEX IF NOT EXISTS cloud_projection_financial_operations_check
  ON cloud_projection_financial_operations(restaurant_id, check_id, operation_created_at DESC);

CREATE TABLE IF NOT EXISTS cloud_master_data_packages (
  stream_name TEXT NOT NULL CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','recipes','inventory_reference','currencies')),
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
    'CheckClosed',
    'KitchenTicketStatusChanged',
    'ItemServed',
    'StockReceiptCaptured',
    'InventoryCountCaptured',
    'StockWriteOffCaptured',
    'ProductionCompleted',
    'StopListUpdated',
    'CatalogItemChangeSuggested',
    'RecipeChangeSuggested',
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
  stream_name TEXT NOT NULL CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','recipes','inventory_reference','currencies')),
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

CREATE TABLE IF NOT EXISTS cloud_recipe_versions (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  owner_catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  version BIGINT NOT NULL CHECK (version > 0),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','review_pending','active','archived')),
  yield_quantity BIGINT NOT NULL DEFAULT 1 CHECK (yield_quantity > 0),
  yield_unit TEXT NOT NULL CHECK (yield_unit <> ''),
  created_by_employee_id TEXT,
  submitted_by_employee_id TEXT,
  approved_by_employee_id TEXT,
  submitted_at TIMESTAMPTZ,
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, owner_catalog_item_id, version)
);

CREATE INDEX IF NOT EXISTS cloud_recipe_versions_owner_status
  ON cloud_recipe_versions(restaurant_id, owner_catalog_item_id, status, version DESC);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_recipe_versions_one_active
  ON cloud_recipe_versions(restaurant_id, owner_catalog_item_id)
  WHERE status = 'active';

CREATE TABLE IF NOT EXISTS cloud_recipe_lines (
  id TEXT PRIMARY KEY,
  recipe_version_id TEXT NOT NULL REFERENCES cloud_recipe_versions(id) ON DELETE CASCADE,
  component_catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  quantity BIGINT NOT NULL CHECK (quantity > 0),
  unit TEXT NOT NULL CHECK (unit <> ''),
  loss_percent BIGINT NOT NULL DEFAULT 0 CHECK (loss_percent >= 0 AND loss_percent <= 100),
  sort_order BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (recipe_version_id, component_catalog_item_id)
);

CREATE INDEX IF NOT EXISTS cloud_recipe_lines_version_order
  ON cloud_recipe_lines(recipe_version_id, sort_order, id);

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

CREATE TABLE IF NOT EXISTS cloud_catalog_suggestions (
  id TEXT PRIMARY KEY,
  suggestion_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL,
  catalog_item_id TEXT,
  proposal_group_id TEXT,
  action TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('pending','approved','rejected','changes_requested')),
  review_comment TEXT NOT NULL DEFAULT '',
  reviewed_by_employee_id TEXT NOT NULL DEFAULT '',
  reviewed_at TIMESTAMPTZ,
  assigned_to_employee_id TEXT NOT NULL DEFAULT '',
  assigned_by_employee_id TEXT NOT NULL DEFAULT '',
  assigned_at TIMESTAMPTZ,
  assignment_note TEXT NOT NULL DEFAULT '',
  applied_catalog_item_id TEXT NOT NULL DEFAULT '',
  source_event_id TEXT NOT NULL DEFAULT '',
  suggested_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_recipe_suggestions (
  id TEXT PRIMARY KEY,
  suggestion_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL,
  recipe_version_id TEXT,
  owner_catalog_item_id TEXT,
  owner_catalog_suggestion_id TEXT,
  proposal_group_id TEXT,
  action TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  prep_time_delta_minutes BIGINT NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN ('pending','approved','rejected','changes_requested')),
  review_comment TEXT NOT NULL DEFAULT '',
  reviewed_by_employee_id TEXT NOT NULL DEFAULT '',
  reviewed_at TIMESTAMPTZ,
  assigned_to_employee_id TEXT NOT NULL DEFAULT '',
  assigned_by_employee_id TEXT NOT NULL DEFAULT '',
  assigned_at TIMESTAMPTZ,
  assignment_note TEXT NOT NULL DEFAULT '',
  source_event_id TEXT NOT NULL DEFAULT '',
  suggested_at TIMESTAMPTZ NOT NULL,
  cloud_received_at TIMESTAMPTZ NOT NULL,
  payload_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_recipe_suggestion_changes (
  id TEXT PRIMARY KEY,
  recipe_suggestion_id TEXT NOT NULL REFERENCES cloud_recipe_suggestions(id) ON DELETE CASCADE,
  line_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  from_catalog_item_id TEXT NOT NULL DEFAULT '',
  to_catalog_item_id TEXT NOT NULL DEFAULT '',
  quantity TEXT NOT NULL DEFAULT '',
  unit_code TEXT NOT NULL DEFAULT '',
  loss_percent TEXT NOT NULL DEFAULT '',
  sort_order BIGINT NOT NULL DEFAULT 0,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_suggestion_review_events (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  suggestion_kind TEXT NOT NULL CHECK (suggestion_kind IN ('catalog','recipe')),
  suggestion_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','approved','rejected','changes_requested')),
  reviewed_by_employee_id TEXT NOT NULL DEFAULT '',
  review_comment TEXT NOT NULL DEFAULT '',
  reviewed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE cloud_catalog_suggestions
  ADD COLUMN IF NOT EXISTS assigned_to_employee_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS assigned_by_employee_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS assignment_note TEXT NOT NULL DEFAULT '';

ALTER TABLE cloud_recipe_suggestions
  ADD COLUMN IF NOT EXISTS assigned_to_employee_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS assigned_by_employee_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS assignment_note TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS cloud_review_assignment_audit_events (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL UNIQUE CHECK (command_id <> ''),
  review_type TEXT NOT NULL CHECK (review_type IN ('catalog_suggestion','recipe_suggestion','stop_list_update')),
  review_id TEXT NOT NULL CHECK (review_id <> ''),
  action TEXT NOT NULL CHECK (action IN ('assigned','unassigned')),
  assigned_to_employee_id TEXT NOT NULL DEFAULT '',
  actor_employee_id TEXT NOT NULL CHECK (actor_employee_id <> ''),
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_review_assignment_audit_events_review_created
  ON cloud_review_assignment_audit_events(review_type, review_id, created_at DESC);

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
    'CheckClosed',
    'KitchenTicketStatusChanged',
    'ItemServed',
    'StockReceiptCaptured',
    'InventoryCountCaptured',
    'StockWriteOffCaptured',
    'ProductionCompleted',
    'StopListUpdated',
    'CatalogItemChangeSuggested',
    'RecipeChangeSuggested',
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
  ADD CONSTRAINT cloud_master_data_packages_stream_name_check CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','recipes','inventory_reference','currencies','proposal_feedback'));

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
  ADD CONSTRAINT cloud_master_data_packages_stream_name_check CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','recipes','inventory_reference','currencies','proposal_feedback'));


-- === 004_cloud_inventory_foundation.sql ===
CREATE TABLE IF NOT EXISTS inventory_event_queue (
  id TEXT PRIMARY KEY,
  receipt_id TEXT NOT NULL UNIQUE REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  warehouse_id TEXT,
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  event_id TEXT NOT NULL CHECK (event_id <> ''),
  event_type TEXT NOT NULL CHECK (event_type <> ''),
  aggregate_type TEXT NOT NULL CHECK (aggregate_type <> ''),
  aggregate_id TEXT NOT NULL CHECK (aggregate_id <> ''),
  status TEXT NOT NULL CHECK (status IN ('pending','processing','processed','failed')),
  attempts BIGINT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  next_retry_at TIMESTAMPTZ,
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  processed_at TIMESTAMPTZ,
  last_error TEXT,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS inventory_event_queue_status_retry
  ON inventory_event_queue(status, next_retry_at, occurred_at, id);

CREATE INDEX IF NOT EXISTS inventory_event_queue_event_type
  ON inventory_event_queue(event_type, occurred_at, id);

CREATE INDEX IF NOT EXISTS inventory_event_queue_restaurant_warehouse_order
  ON inventory_event_queue(restaurant_id, warehouse_id, occurred_at, id);

CREATE TABLE IF NOT EXISTS stock_documents (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  warehouse_id TEXT,
  document_type TEXT NOT NULL CHECK (document_type IN ('SALE','RETURN','WASTE','PRODUCTION','PURCHASE','ADJUSTMENT','TRANSFER','INVENTORY_COUNT')),
  source_event_id TEXT NOT NULL CHECK (source_event_id <> ''),
  source_event_type TEXT NOT NULL CHECK (source_event_type <> ''),
  business_date_local DATE NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS stock_documents_restaurant_occurred_at
  ON stock_documents(restaurant_id, occurred_at, id);

CREATE INDEX IF NOT EXISTS stock_documents_restaurant_warehouse_occurred_at
  ON stock_documents(restaurant_id, warehouse_id, occurred_at, id);

CREATE UNIQUE INDEX IF NOT EXISTS stock_documents_source_event_unique
  ON stock_documents(source_event_id, source_event_type);

CREATE TABLE IF NOT EXISTS stock_ledger (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  warehouse_id TEXT,
  stock_document_id TEXT NOT NULL REFERENCES stock_documents(id) ON DELETE RESTRICT,
  source_event_id TEXT NOT NULL CHECK (source_event_id <> ''),
  source_event_type TEXT NOT NULL CHECK (source_event_type <> ''),
  catalog_item_id TEXT NOT NULL CHECK (catalog_item_id <> ''),
  order_line_id TEXT,
  movement_type TEXT NOT NULL CHECK (movement_type IN ('IN','OUT')),
  quantity NUMERIC(14,3) NOT NULL CHECK (quantity > 0),
  unit_code TEXT NOT NULL CHECK (unit_code <> ''),
  unit_cost_minor BIGINT NOT NULL CHECK (unit_cost_minor >= 0),
  total_cost_minor BIGINT NOT NULL,
  costing_status TEXT NOT NULL CHECK (costing_status IN ('final','estimated','needs_recalculation','recalculated','failed')),
  occurred_at TIMESTAMPTZ NOT NULL,
  business_date_local DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS stock_ledger_restaurant_occurred_at
  ON stock_ledger(restaurant_id, occurred_at, id);

CREATE INDEX IF NOT EXISTS stock_ledger_restaurant_warehouse_occurred_at
  ON stock_ledger(restaurant_id, warehouse_id, occurred_at, id);

CREATE INDEX IF NOT EXISTS stock_ledger_source_event
  ON stock_ledger(source_event_id, source_event_type);

CREATE INDEX IF NOT EXISTS stock_ledger_order_line_consumption
  ON stock_ledger(restaurant_id, order_line_id, source_event_type, movement_type);

CREATE TABLE IF NOT EXISTS stock_recalculation_jobs (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  source_document_id TEXT NOT NULL REFERENCES stock_documents(id) ON DELETE RESTRICT,
  status TEXT NOT NULL CHECK (status <> ''),
  recalculate_from TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS stock_recalculation_jobs_restaurant_status
  ON stock_recalculation_jobs(restaurant_id, status, recalculate_from);

CREATE TABLE IF NOT EXISTS cloud_projection_stop_list_updates (
  source_event_id TEXT PRIMARY KEY CHECK (source_event_id <> ''),
  queue_id TEXT NOT NULL CHECK (queue_id <> ''),
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  stop_list_id TEXT NOT NULL CHECK (stop_list_id <> ''),
  warehouse_id TEXT,
  catalog_item_id TEXT NOT NULL CHECK (catalog_item_id <> ''),
  available_quantity NUMERIC(14,3),
  active BOOLEAN NOT NULL,
  conflict_policy TEXT NOT NULL CHECK (conflict_policy IN ('cloud_wins','edge_overlay_until_next_publication','edge_overlay_requires_manager_review')),
  source TEXT NOT NULL CHECK (source <> ''),
  reason TEXT,
  projection_action TEXT NOT NULL CHECK (projection_action IN ('applied_edge_overlay','ignored_cloud_wins','requires_manager_review')),
  review_status TEXT NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending','approved','rejected','changes_requested')),
  review_comment TEXT NOT NULL DEFAULT '',
  reviewed_by_employee_id TEXT NOT NULL DEFAULT '',
  reviewed_at TIMESTAMPTZ,
  assigned_to_employee_id TEXT NOT NULL DEFAULT '',
  assigned_by_employee_id TEXT NOT NULL DEFAULT '',
  assigned_at TIMESTAMPTZ,
  assignment_note TEXT NOT NULL DEFAULT '',
  applied_stop_list_id TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  projected_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS cloud_projection_stop_list_updates_restaurant_updated
  ON cloud_projection_stop_list_updates(restaurant_id, updated_at DESC, source_event_id);

CREATE INDEX IF NOT EXISTS cloud_projection_stop_list_updates_action
  ON cloud_projection_stop_list_updates(projection_action, projected_at DESC);

ALTER TABLE cloud_projection_stop_list_updates
  ADD COLUMN IF NOT EXISTS assigned_to_employee_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS assigned_by_employee_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS assignment_note TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS stop_lists (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  catalog_item_id TEXT NOT NULL CHECK (catalog_item_id <> ''),
  available_quantity NUMERIC(14,3),
  source TEXT NOT NULL CHECK (source <> ''),
  reason TEXT,
  active BOOLEAN NOT NULL,
  cloud_version BIGINT,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS stop_lists_restaurant_item
  ON stop_lists(restaurant_id, catalog_item_id);
