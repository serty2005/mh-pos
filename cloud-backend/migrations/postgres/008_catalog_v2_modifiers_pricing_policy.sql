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
    (amount_kind = 'fixed' AND amount_minor > 0 AND value_basis_points = 0)
    OR (amount_kind = 'percentage' AND value_basis_points > 0 AND amount_minor = 0)
  )
);

ALTER TABLE cloud_master_data_packages
  DROP CONSTRAINT IF EXISTS cloud_master_data_packages_stream_name_check;

ALTER TABLE cloud_master_data_packages
  ADD CONSTRAINT cloud_master_data_packages_stream_name_check CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','currencies'));
