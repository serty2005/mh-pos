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
