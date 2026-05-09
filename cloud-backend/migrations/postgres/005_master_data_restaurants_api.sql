CREATE TABLE IF NOT EXISTS cloud_restaurants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL CHECK (name <> ''),
  timezone TEXT NOT NULL CHECK (timezone <> ''),
  currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  business_day_mode TEXT NOT NULL CHECK (business_day_mode IN ('standard','24_7')),
  business_day_boundary_local_time TEXT NOT NULL CHECK (business_day_boundary_local_time ~ '^[0-2][0-9]:[0-5][0-9]'),
  status TEXT NOT NULL CHECK (status IN ('active','archived')),
  cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0),
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS cloud_restaurants_status_updated
  ON cloud_restaurants(status, updated_at DESC);

ALTER TABLE cloud_roles
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_roles
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE cloud_employees
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_catalog_items
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_catalog_items
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE cloud_catalog_items
  DROP CONSTRAINT IF EXISTS cloud_catalog_items_kind_check;

ALTER TABLE cloud_catalog_items
  ADD CONSTRAINT cloud_catalog_items_kind_check
  CHECK (kind IN ('dish','good','ingredient','raw_material','semi_finished'));

ALTER TABLE cloud_menu_items
  ADD COLUMN IF NOT EXISTS cloud_version BIGINT NOT NULL DEFAULT 1 CHECK (cloud_version > 0);

ALTER TABLE cloud_menu_items
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE cloud_catalog_items
  DROP CONSTRAINT IF EXISTS cloud_catalog_items_restaurant_id_sku_key;

CREATE UNIQUE INDEX IF NOT EXISTS cloud_catalog_items_active_sku
  ON cloud_catalog_items(restaurant_id, sku)
  WHERE status <> 'archived';
