CREATE TABLE local_event_log (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE,
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

CREATE INDEX local_event_log_created_at ON local_event_log(created_at);
CREATE INDEX local_event_log_event_type_created_at ON local_event_log(event_type, created_at);
