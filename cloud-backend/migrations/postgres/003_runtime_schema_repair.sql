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
    'CheckReprinted',
    'PaymentCaptured',
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
