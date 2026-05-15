-- sqlite:repair-sql
PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS recipe_versions_dish_catalog_item_insert;
DROP TRIGGER IF EXISTS recipe_versions_dish_catalog_item_update;
DROP TRIGGER IF EXISTS recipe_lines_ingredient_or_good_insert;
DROP TRIGGER IF EXISTS recipe_lines_ingredient_or_good_update;
DROP TRIGGER IF EXISTS recipe_versions_owner_catalog_item_insert;
DROP TRIGGER IF EXISTS recipe_versions_owner_catalog_item_update;
DROP TRIGGER IF EXISTS recipe_lines_good_or_semi_finished_insert;
DROP TRIGGER IF EXISTS recipe_lines_good_or_semi_finished_update;

CREATE TABLE IF NOT EXISTS catalog_items_v2 (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL CHECK (type IN ('dish', 'good', 'semi_finished', 'service')),
  folder_id TEXT,
  name TEXT NOT NULL,
  sku TEXT NOT NULL UNIQUE,
  base_unit TEXT NOT NULL,
  kitchen_type TEXT NOT NULL DEFAULT '',
  accounting_category TEXT NOT NULL DEFAULT '',
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

INSERT OR IGNORE INTO catalog_items_v2(
  id,type,folder_id,name,sku,base_unit,kitchen_type,accounting_category,active,created_at,updated_at,
  cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at
)
SELECT
  id,
  CASE WHEN type = 'ingredient' THEN 'good' ELSE type END,
  NULL,
  name,sku,base_unit,'','',active,created_at,updated_at,
  cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at
FROM catalog_items
WHERE type IN ('ingredient','dish','good','semi_finished','service');

DROP TABLE catalog_items;
ALTER TABLE catalog_items_v2 RENAME TO catalog_items;

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS catalog_folders (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  parent_id TEXT REFERENCES catalog_folders(id),
  name TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE TABLE IF NOT EXISTS catalog_folder_parameters (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  folder_id TEXT NOT NULL REFERENCES catalog_folders(id),
  parameter_key TEXT NOT NULL,
  value_type TEXT NOT NULL,
  value_json TEXT NOT NULL CHECK (json_valid(value_json)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(folder_id, parameter_key)
);

CREATE TABLE IF NOT EXISTS catalog_tags (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  code TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(restaurant_id, code)
);

CREATE TABLE IF NOT EXISTS catalog_item_tags (
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  tag_id TEXT NOT NULL REFERENCES catalog_tags(id),
  restaurant_id TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  PRIMARY KEY(catalog_item_id, tag_id)
);

CREATE TABLE IF NOT EXISTS modifier_groups (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  required INTEGER NOT NULL DEFAULT 0 CHECK (required IN (0,1)),
  min_count INTEGER NOT NULL DEFAULT 0 CHECK (min_count >= 0),
  max_count INTEGER NOT NULL DEFAULT 0 CHECK (max_count >= 0),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  CHECK (max_count = 0 OR max_count >= min_count)
);

CREATE TABLE IF NOT EXISTS modifier_options (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  modifier_group_id TEXT NOT NULL REFERENCES modifier_groups(id),
  name TEXT NOT NULL,
  price_minor INTEGER NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE TABLE IF NOT EXISTS modifier_group_bindings (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  modifier_group_id TEXT NOT NULL REFERENCES modifier_groups(id),
  target_type TEXT NOT NULL CHECK (target_type IN ('menu_item','catalog_item','folder','tag')),
  target_id TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(modifier_group_id, target_type, target_id)
);

CREATE TABLE IF NOT EXISTS menu_item_modifier_groups (
  menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
  modifier_group_id TEXT NOT NULL REFERENCES modifier_groups(id),
  sort_order INTEGER NOT NULL DEFAULT 0,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  PRIMARY KEY(menu_item_id, modifier_group_id)
);

CREATE TABLE IF NOT EXISTS order_line_modifiers (
  id TEXT PRIMARY KEY,
  order_line_id TEXT NOT NULL REFERENCES order_lines(id) ON DELETE CASCADE,
  modifier_group_id TEXT NOT NULL REFERENCES modifier_groups(id),
  modifier_option_id TEXT NOT NULL REFERENCES modifier_options(id),
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  total_price INTEGER NOT NULL CHECK (total_price >= 0)
);

CREATE TABLE IF NOT EXISTS pricing_policies (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  kind TEXT NOT NULL CHECK (kind IN ('discount','surcharge')),
  name TEXT NOT NULL,
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  requires_permission TEXT NOT NULL DEFAULT '',
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE INDEX IF NOT EXISTS pricing_policies_restaurant_active ON pricing_policies(restaurant_id, active, application_index);

CREATE TABLE IF NOT EXISTS precheck_line_modifiers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  order_line_id TEXT NOT NULL,
  modifier_group_id TEXT NOT NULL,
  modifier_option_id TEXT NOT NULL,
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price_minor INTEGER NOT NULL CHECK (unit_price_minor >= 0),
  total_minor INTEGER NOT NULL CHECK (total_minor >= 0)
);

CREATE TRIGGER IF NOT EXISTS recipe_versions_owner_catalog_item_insert
BEFORE INSERT ON recipe_versions
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.dish_catalog_item_id AND type IN ('dish', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe version must reference dish or semi_finished catalog item');
END;

CREATE TRIGGER IF NOT EXISTS recipe_versions_owner_catalog_item_update
BEFORE UPDATE OF dish_catalog_item_id ON recipe_versions
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.dish_catalog_item_id AND type IN ('dish', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe version must reference dish or semi_finished catalog item');
END;

CREATE TRIGGER IF NOT EXISTS recipe_lines_good_or_semi_finished_insert
BEFORE INSERT ON recipe_lines
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.catalog_item_id AND type IN ('good', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe line must reference good or semi_finished catalog item');
END;

CREATE TRIGGER IF NOT EXISTS recipe_lines_good_or_semi_finished_update
BEFORE UPDATE OF catalog_item_id ON recipe_lines
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.catalog_item_id AND type IN ('good', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe line must reference good or semi_finished catalog item');
END;
