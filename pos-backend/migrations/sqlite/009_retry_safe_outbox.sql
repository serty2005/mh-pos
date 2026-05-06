CREATE TABLE pos_sync_outbox_next (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL,
  sequence_no INTEGER NOT NULL UNIQUE CHECK (sequence_no > 0),
  origin TEXT NOT NULL CHECK (origin IN ('edge_device', 'cloud_sync', 'system_seed')),
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  command_type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'sent', 'failed', 'suspended')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  next_retry_at TEXT,
  locked_at TEXT,
  locked_by TEXT CHECK (locked_by IS NULL OR locked_by <> ''),
  sent_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (status = 'processing' OR (locked_at IS NULL AND locked_by IS NULL)),
  CHECK ((locked_at IS NULL AND locked_by IS NULL) OR (locked_at IS NOT NULL AND locked_by IS NOT NULL)),
  CHECK (sent_at IS NULL OR status = 'sent')
);

INSERT INTO pos_sync_outbox_next (
  id,
  command_id,
  sequence_no,
  origin,
  restaurant_id,
  device_id,
  aggregate_type,
  aggregate_id,
  command_type,
  payload_json,
  status,
  attempts,
  last_error,
  created_at,
  updated_at
)
SELECT
  id,
  command_id,
  ROW_NUMBER() OVER (ORDER BY created_at, id),
  origin,
  restaurant_id,
  device_id,
  aggregate_type,
  aggregate_id,
  command_type,
  payload_json,
  status,
  attempts,
  last_error,
  created_at,
  updated_at
FROM pos_sync_outbox;

DROP TABLE pos_sync_outbox;
ALTER TABLE pos_sync_outbox_next RENAME TO pos_sync_outbox;

CREATE INDEX pos_sync_outbox_status_sequence_no ON pos_sync_outbox(status, sequence_no);
CREATE INDEX pos_sync_outbox_pending_retry_sequence ON pos_sync_outbox(next_retry_at, sequence_no) WHERE status = 'pending';
CREATE INDEX pos_sync_outbox_processing_locked_at ON pos_sync_outbox(locked_at) WHERE status = 'processing';
CREATE INDEX pos_sync_outbox_command_id_created_at ON pos_sync_outbox(command_id, created_at);
