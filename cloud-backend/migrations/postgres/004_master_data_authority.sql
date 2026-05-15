CREATE TABLE IF NOT EXISTS cloud_roles (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  permissions_json JSONB NOT NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, name)
);

CREATE TABLE IF NOT EXISTS cloud_employees (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  role_id TEXT NOT NULL REFERENCES cloud_roles(id),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('active','suspended','archived')),
  pin_hash TEXT NOT NULL CHECK (pin_hash <> ''),
  pin_credential_version BIGINT NOT NULL DEFAULT 1 CHECK (pin_credential_version > 0),
  permission_snapshot_json JSONB NOT NULL,
  suspended_at TIMESTAMPTZ,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_employees_restaurant_status
  ON cloud_employees(restaurant_id, status);

CREATE TABLE IF NOT EXISTS cloud_categories (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  sort_order BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_categories_restaurant_status
  ON cloud_categories(restaurant_id, status, sort_order);

CREATE TABLE IF NOT EXISTS cloud_catalog_items (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  kind TEXT NOT NULL CHECK (kind IN ('dish','good','semi_finished','service')),
  folder_id TEXT,
  name TEXT NOT NULL CHECK (name <> ''),
  sku TEXT NOT NULL CHECK (sku <> ''),
  base_unit TEXT NOT NULL CHECK (base_unit <> ''),
  kitchen_type TEXT,
  accounting_category TEXT,
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, sku)
);

CREATE INDEX IF NOT EXISTS cloud_catalog_items_restaurant_kind_status
  ON cloud_catalog_items(restaurant_id, kind, status);

CREATE TABLE IF NOT EXISTS cloud_dishes (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  recipe_policy TEXT NOT NULL DEFAULT 'none' CHECK (recipe_policy IN ('none','optional','required')),
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_goods (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  stock_tracking_mode TEXT NOT NULL DEFAULT 'none' CHECK (stock_tracking_mode IN ('none','quantity')),
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_semi_finished_products (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  production_unit TEXT NOT NULL DEFAULT 'portion',
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_services (
  catalog_item_id TEXT PRIMARY KEY REFERENCES cloud_catalog_items(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  fixed_unit TEXT NOT NULL DEFAULT 'service',
  updated_at TIMESTAMPTZ NOT NULL
);

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

CREATE TABLE IF NOT EXISTS cloud_recipe_items (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  recipe_owner_catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  component_catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  quantity BIGINT NOT NULL CHECK (quantity > 0),
  unit TEXT NOT NULL CHECK (unit <> ''),
  loss_percent BIGINT NOT NULL DEFAULT 0 CHECK (loss_percent >= 0 AND loss_percent <= 100),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (recipe_owner_catalog_item_id, component_catalog_item_id)
);

CREATE TABLE IF NOT EXISTS cloud_modifier_groups (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  name TEXT NOT NULL CHECK (name <> ''),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  required BOOLEAN NOT NULL DEFAULT false,
  min_count BIGINT NOT NULL DEFAULT 0 CHECK (min_count >= 0),
  max_count BIGINT NOT NULL DEFAULT 1 CHECK (max_count >= 0),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cloud_modifier_options (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  modifier_group_id TEXT NOT NULL REFERENCES cloud_modifier_groups(id),
  name TEXT NOT NULL CHECK (name <> ''),
  price_minor BIGINT NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

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

CREATE TABLE IF NOT EXISTS cloud_menu_items (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  catalog_item_id TEXT NOT NULL REFERENCES cloud_catalog_items(id),
  category_id TEXT REFERENCES cloud_categories(id),
  name TEXT NOT NULL CHECK (name <> ''),
  price BIGINT NOT NULL CHECK (price >= 0),
  currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  status TEXT NOT NULL CHECK (status IN ('draft','published','archived')),
  availability_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  station_routing_key TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_menu_items_restaurant_status
  ON cloud_menu_items(restaurant_id, status);

CREATE TABLE IF NOT EXISTS cloud_menu_item_modifier_groups (
  menu_item_id TEXT NOT NULL REFERENCES cloud_menu_items(id) ON DELETE CASCADE,
  modifier_group_id TEXT NOT NULL REFERENCES cloud_modifier_groups(id) ON DELETE RESTRICT,
  sort_order BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (menu_item_id, modifier_group_id)
);

CREATE TABLE IF NOT EXISTS cloud_menu_location_assignments (
  menu_item_id TEXT NOT NULL REFERENCES cloud_menu_items(id) ON DELETE CASCADE,
  location_id TEXT NOT NULL CHECK (location_id <> ''),
  active BOOLEAN NOT NULL DEFAULT true,
  PRIMARY KEY (menu_item_id, location_id)
);

CREATE TABLE IF NOT EXISTS cloud_master_data_publications (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  version BIGINT NOT NULL CHECK (version > 0),
  status TEXT NOT NULL CHECK (status IN ('published','archived')),
  cloud_version BIGINT NOT NULL CHECK (cloud_version > 0),
  published_at TIMESTAMPTZ NOT NULL,
  published_by TEXT NOT NULL CHECK (published_by <> ''),
  package_json JSONB NOT NULL,
  package_sha256 TEXT NOT NULL CHECK (package_sha256 <> ''),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (restaurant_id, version)
);

CREATE INDEX IF NOT EXISTS cloud_master_data_publications_current
  ON cloud_master_data_publications(restaurant_id, version DESC)
  WHERE status = 'published';
