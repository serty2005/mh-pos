CREATE TABLE local_event_log_next (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE,
  command_id TEXT NOT NULL UNIQUE,
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
  id,
  event_id,
  command_id,
  envelope_version,
  event_type,
  aggregate_type,
  aggregate_id,
  restaurant_id,
  device_id,
  shift_id,
  payload_json,
  occurred_at,
  created_at
)
SELECT
  id,
  event_id,
  event_id,
  envelope_version,
  event_type,
  aggregate_type,
  aggregate_id,
  restaurant_id,
  device_id,
  shift_id,
  payload_json,
  occurred_at,
  created_at
FROM local_event_log;

DROP TABLE local_event_log;
ALTER TABLE local_event_log_next RENAME TO local_event_log;

CREATE INDEX local_event_log_created_at ON local_event_log(created_at);
CREATE INDEX local_event_log_event_type_created_at ON local_event_log(event_type, created_at);
CREATE INDEX local_event_log_command_id_created_at ON local_event_log(command_id, created_at);
