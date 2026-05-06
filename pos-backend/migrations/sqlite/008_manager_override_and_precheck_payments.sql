CREATE TABLE local_event_log_next (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE,
  command_id TEXT NOT NULL,
  envelope_version TEXT NOT NULL,
  event_type TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  shift_id TEXT CHECK (shift_id IS NULL OR shift_id <> ''),
  payload_json TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

INSERT INTO local_event_log_next (
  id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,shift_id,payload_json,occurred_at,created_at
)
SELECT id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,shift_id,payload_json,occurred_at,created_at
FROM local_event_log;

DROP TABLE local_event_log;
ALTER TABLE local_event_log_next RENAME TO local_event_log;

CREATE INDEX local_event_log_created_at ON local_event_log(created_at);
CREATE INDEX local_event_log_event_type_created_at ON local_event_log(event_type, created_at);
CREATE INDEX local_event_log_command_id_created_at ON local_event_log(command_id, created_at);

CREATE TABLE pos_sync_outbox_next (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL,
  origin TEXT NOT NULL CHECK (origin IN ('edge_device', 'cloud_sync', 'system_seed')),
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  command_type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
  attempts INTEGER NOT NULL,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

INSERT INTO pos_sync_outbox_next (
  id,command_id,origin,restaurant_id,device_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,last_error,created_at,updated_at
)
SELECT id,command_id,origin,restaurant_id,device_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,last_error,created_at,updated_at
FROM pos_sync_outbox;

DROP TABLE pos_sync_outbox;
ALTER TABLE pos_sync_outbox_next RENAME TO pos_sync_outbox;

CREATE INDEX pos_sync_outbox_status_created_at ON pos_sync_outbox(status, created_at);
CREATE INDEX pos_sync_outbox_command_id_created_at ON pos_sync_outbox(command_id, created_at);

DROP TABLE payment_attempts;
DROP TABLE payments;

CREATE TABLE payments (
  id TEXT PRIMARY KEY,
  edge_payment_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'other')),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('captured', 'refunded', 'failed')),
  provider_name TEXT CHECK (provider_name IS NULL OR provider_name <> ''),
  provider_transaction_id TEXT CHECK (provider_transaction_id IS NULL OR provider_transaction_id <> ''),
  provider_reference TEXT CHECK (provider_reference IS NULL OR provider_reference <> ''),
  fingerprint_hash TEXT CHECK (fingerprint_hash IS NULL OR fingerprint_hash <> ''),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX payments_precheck_id_created_at ON payments(precheck_id, created_at);
CREATE INDEX payments_provider_transaction_id ON payments(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
CREATE INDEX payments_fingerprint_hash ON payments(fingerprint_hash) WHERE fingerprint_hash IS NOT NULL;

CREATE TABLE payment_attempts (
  id TEXT PRIMARY KEY,
  payment_id TEXT NOT NULL REFERENCES payments(id),
  attempt_no INTEGER NOT NULL CHECK (attempt_no > 0),
  method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'other')),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('captured', 'failed')),
  provider_name TEXT CHECK (provider_name IS NULL OR provider_name <> ''),
  provider_transaction_id TEXT CHECK (provider_transaction_id IS NULL OR provider_transaction_id <> ''),
  provider_reference TEXT CHECK (provider_reference IS NULL OR provider_reference <> ''),
  fingerprint_hash TEXT CHECK (fingerprint_hash IS NULL OR fingerprint_hash <> ''),
  attempted_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(payment_id, attempt_no)
);

CREATE INDEX payment_attempts_payment_id_attempt_no ON payment_attempts(payment_id, attempt_no);
CREATE INDEX payment_attempts_provider_transaction_id ON payment_attempts(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;

CREATE TABLE manager_override_audit (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  order_id TEXT NOT NULL REFERENCES orders(id),
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  manager_employee_id TEXT NOT NULL REFERENCES employees(id),
  action TEXT NOT NULL CHECK (action IN ('cancel_precheck')),
  reason TEXT NOT NULL CHECK (reason <> ''),
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX manager_override_audit_precheck_created_at ON manager_override_audit(precheck_id, created_at);
CREATE INDEX manager_override_audit_manager_created_at ON manager_override_audit(manager_employee_id, created_at);
