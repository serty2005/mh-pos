CREATE TABLE cloud_edge_event_receipts (
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
    'PrecheckCancelled',
    'CheckCreated',
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

CREATE UNIQUE INDEX cloud_edge_event_receipts_edge_event_key
  ON cloud_edge_event_receipts(restaurant_id, device_id, edge_event_id);

CREATE INDEX cloud_edge_event_receipts_event_type_received_at
  ON cloud_edge_event_receipts(event_type, cloud_received_at);

CREATE TABLE cloud_edge_event_raw_payloads (
  receipt_id TEXT PRIMARY KEY REFERENCES cloud_edge_event_receipts(id) ON DELETE RESTRICT,
  raw_payload JSONB NOT NULL,
  raw_payload_sha256_hex TEXT NOT NULL CHECK (raw_payload_sha256_hex <> ''),
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE cloud_operational_events (
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

CREATE UNIQUE INDEX cloud_operational_events_edge_event_key
  ON cloud_operational_events(restaurant_id, device_id, edge_event_id);

CREATE INDEX cloud_operational_events_type_received_at
  ON cloud_operational_events(event_type, cloud_received_at);

CREATE INDEX cloud_operational_events_restaurant_sequence
  ON cloud_operational_events(restaurant_id, device_id, occurred_at, event_id);
