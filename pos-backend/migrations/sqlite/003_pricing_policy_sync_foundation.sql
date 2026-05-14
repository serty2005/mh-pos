-- sqlite:repair-column tax_profiles cloud_version
ALTER TABLE tax_profiles ADD COLUMN cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0);

-- sqlite:repair-column tax_profiles cloud_updated_at
ALTER TABLE tax_profiles ADD COLUMN cloud_updated_at TEXT;

-- sqlite:repair-column tax_profiles cloud_deleted_at
ALTER TABLE tax_profiles ADD COLUMN cloud_deleted_at TEXT;

-- sqlite:repair-column tax_profiles last_synced_at
ALTER TABLE tax_profiles ADD COLUMN last_synced_at TEXT;

-- sqlite:repair-column tax_rules cloud_version
ALTER TABLE tax_rules ADD COLUMN cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0);

-- sqlite:repair-column tax_rules cloud_updated_at
ALTER TABLE tax_rules ADD COLUMN cloud_updated_at TEXT;

-- sqlite:repair-column tax_rules cloud_deleted_at
ALTER TABLE tax_rules ADD COLUMN cloud_deleted_at TEXT;

-- sqlite:repair-column tax_rules last_synced_at
ALTER TABLE tax_rules ADD COLUMN last_synced_at TEXT;

-- sqlite:repair-column service_charge_rules cloud_version
ALTER TABLE service_charge_rules ADD COLUMN cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0);

-- sqlite:repair-column service_charge_rules cloud_updated_at
ALTER TABLE service_charge_rules ADD COLUMN cloud_updated_at TEXT;

-- sqlite:repair-column service_charge_rules cloud_deleted_at
ALTER TABLE service_charge_rules ADD COLUMN cloud_deleted_at TEXT;

-- sqlite:repair-column service_charge_rules last_synced_at
ALTER TABLE service_charge_rules ADD COLUMN last_synced_at TEXT;

-- sqlite:repair-sql
CREATE TABLE IF NOT EXISTS cloud_master_sync_state (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  stream_name TEXT NOT NULL CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','recipes','inventory_reference')),
  direction TEXT NOT NULL CHECK (direction = 'cloud_to_edge'),
  sync_mode TEXT NOT NULL CHECK (sync_mode IN ('full_snapshot','incremental')),
  checkpoint_token TEXT CHECK (checkpoint_token IS NULL OR checkpoint_token <> ''),
  last_cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (last_cloud_version >= 0),
  last_cloud_updated_at TEXT,
  last_applied_at TEXT,
  status TEXT NOT NULL CHECK (status IN ('never_synced','applying','applied','failed')),
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(node_device_id, stream_name)
);

CREATE TABLE IF NOT EXISTS cloud_master_sync_state_v2 (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  stream_name TEXT NOT NULL CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','recipes','inventory_reference')),
  direction TEXT NOT NULL CHECK (direction = 'cloud_to_edge'),
  sync_mode TEXT NOT NULL CHECK (sync_mode IN ('full_snapshot','incremental')),
  checkpoint_token TEXT CHECK (checkpoint_token IS NULL OR checkpoint_token <> ''),
  last_cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (last_cloud_version >= 0),
  last_cloud_updated_at TEXT,
  last_applied_at TEXT,
  status TEXT NOT NULL CHECK (status IN ('never_synced','applying','applied','failed')),
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(node_device_id, stream_name)
);

INSERT OR IGNORE INTO cloud_master_sync_state_v2(
  id,restaurant_id,node_device_id,stream_name,direction,sync_mode,checkpoint_token,last_cloud_version,
  last_cloud_updated_at,last_applied_at,status,last_error,created_at,updated_at
)
SELECT
  id,restaurant_id,node_device_id,stream_name,direction,sync_mode,checkpoint_token,last_cloud_version,
  last_cloud_updated_at,last_applied_at,status,last_error,created_at,updated_at
FROM cloud_master_sync_state;

DROP TABLE cloud_master_sync_state;
ALTER TABLE cloud_master_sync_state_v2 RENAME TO cloud_master_sync_state;
CREATE INDEX IF NOT EXISTS cloud_master_sync_state_node_status ON cloud_master_sync_state(node_device_id, status);
