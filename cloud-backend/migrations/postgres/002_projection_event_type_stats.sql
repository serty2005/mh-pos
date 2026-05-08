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
