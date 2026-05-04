CREATE TABLE recipe_versions (
  id TEXT PRIMARY KEY,
  dish_catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  version INTEGER NOT NULL CHECK (version > 0),
  name TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'archived')),
  yield_quantity INTEGER NOT NULL CHECK (yield_quantity > 0),
  yield_unit TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(dish_catalog_item_id, version)
);

CREATE TABLE recipe_lines (
  id TEXT PRIMARY KEY,
  recipe_version_id TEXT NOT NULL REFERENCES recipe_versions(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit TEXT NOT NULL,
  loss_percent INTEGER NOT NULL CHECK (loss_percent >= 0 AND loss_percent <= 100),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(recipe_version_id, catalog_item_id)
);

CREATE TABLE purchase_receipts (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  supplier_name TEXT NOT NULL,
  document_number TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft', 'posted', 'cancelled')),
  received_at TEXT NOT NULL,
  total_amount INTEGER NOT NULL CHECK (total_amount >= 0),
  currency TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE purchase_receipt_lines (
  id TEXT PRIMARY KEY,
  purchase_receipt_id TEXT NOT NULL REFERENCES purchase_receipts(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit TEXT NOT NULL,
  unit_cost INTEGER NOT NULL CHECK (unit_cost >= 0),
  total_cost INTEGER NOT NULL CHECK (total_cost >= 0),
  currency TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE stock_documents (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  document_type TEXT NOT NULL CHECK (document_type IN ('purchase_receipt', 'adjustment', 'transfer', 'write_off', 'production')),
  source_type TEXT,
  source_id TEXT,
  status TEXT NOT NULL CHECK (status IN ('draft', 'posted', 'cancelled')),
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE stock_moves (
  id TEXT PRIMARY KEY,
  stock_document_id TEXT NOT NULL REFERENCES stock_documents(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  location_id TEXT,
  movement_type TEXT NOT NULL CHECK (movement_type IN ('in', 'out', 'adjustment')),
  quantity INTEGER NOT NULL CHECK (quantity <> 0),
  unit TEXT NOT NULL,
  unit_cost INTEGER CHECK (unit_cost IS NULL OR unit_cost >= 0),
  total_cost INTEGER CHECK (total_cost IS NULL OR total_cost >= 0),
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE stock_balances (
  id TEXT PRIMARY KEY,
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  location_id TEXT,
  quantity INTEGER NOT NULL,
  unit TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX stock_balances_catalog_location ON stock_balances(catalog_item_id, location_id) WHERE location_id IS NOT NULL;

CREATE TABLE item_costs (
  id TEXT PRIMARY KEY,
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  cost_type TEXT NOT NULL CHECK (cost_type IN ('last_purchase', 'moving_average')),
  amount INTEGER NOT NULL CHECK (amount >= 0),
  currency TEXT NOT NULL,
  source_type TEXT,
  source_id TEXT,
  effective_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX item_costs_catalog_type_effective_at ON item_costs(catalog_item_id, cost_type, effective_at);

CREATE TRIGGER recipe_versions_dish_catalog_item_insert
BEFORE INSERT ON recipe_versions
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.dish_catalog_item_id AND type = 'dish')
BEGIN
  SELECT RAISE(ABORT, 'recipe version must reference dish catalog item');
END;

CREATE TRIGGER recipe_versions_dish_catalog_item_update
BEFORE UPDATE OF dish_catalog_item_id ON recipe_versions
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.dish_catalog_item_id AND type = 'dish')
BEGIN
  SELECT RAISE(ABORT, 'recipe version must reference dish catalog item');
END;

CREATE TRIGGER recipe_lines_ingredient_or_good_insert
BEFORE INSERT ON recipe_lines
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.catalog_item_id AND type IN ('ingredient', 'good'))
BEGIN
  SELECT RAISE(ABORT, 'recipe line must reference ingredient or good catalog item');
END;

CREATE TRIGGER recipe_lines_ingredient_or_good_update
BEFORE UPDATE OF catalog_item_id ON recipe_lines
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.catalog_item_id AND type IN ('ingredient', 'good'))
BEGIN
  SELECT RAISE(ABORT, 'recipe line must reference ingredient or good catalog item');
END;

CREATE TRIGGER stock_moves_no_update
BEFORE UPDATE ON stock_moves
BEGIN
  SELECT RAISE(ABORT, 'stock_moves are append-only');
END;

CREATE TRIGGER stock_moves_no_delete
BEFORE DELETE ON stock_moves
BEGIN
  SELECT RAISE(ABORT, 'stock_moves are append-only');
END;
