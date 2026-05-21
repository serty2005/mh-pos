-- Схлопнутый pre-pilot baseline. До первого клиента dev/test базы пересоздаются,
-- реальные data-preserving migrations начинаются после первого внедрения.

-- === 001_init.sql ===
CREATE TABLE IF NOT EXISTS restaurants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  timezone TEXT NOT NULL,
  currency TEXT NOT NULL,
  business_day_mode TEXT NOT NULL DEFAULT 'standard' CHECK (business_day_mode IN ('standard','24_7')),
  business_day_boundary_local_time TEXT NOT NULL DEFAULT '05:00' CHECK (business_day_boundary_local_time GLOB '[0-1][0-9]:[0-5][0-9]' OR business_day_boundary_local_time GLOB '2[0-3]:[0-5][0-9]'),
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_code TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  active INTEGER NOT NULL,
  registered_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(restaurant_id, device_code)
);

CREATE TABLE IF NOT EXISTS edge_node_identity (
  id TEXT PRIMARY KEY CHECK (id = 'local'),
  node_device_id TEXT NOT NULL REFERENCES devices(id),
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  status TEXT NOT NULL CHECK (status IN ('paired')),
  pairing_code_hash TEXT NOT NULL CHECK (pairing_code_hash <> ''),
  paired_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS edge_provisioning_state (
  id TEXT PRIMARY KEY CHECK (id = 'local'),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  cloud_url TEXT,
  license_url TEXT,
  restaurant_id TEXT,
  status TEXT NOT NULL CHECK (status IN ('not_configured','pending_admin_approval','assigned_downloading_snapshot','paired','error')),
  credentials_type TEXT,
  credentials_token TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS client_devices (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  node_device_id TEXT NOT NULL REFERENCES devices(id),
  client_device_id TEXT NOT NULL CHECK (client_device_id <> ''),
  status TEXT NOT NULL CHECK (status IN ('active')),
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(node_device_id, client_device_id)
);

CREATE INDEX IF NOT EXISTS client_devices_restaurant_node_status ON client_devices(restaurant_id, node_device_id, status);

CREATE TABLE IF NOT EXISTS roles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  permissions_json TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE TABLE IF NOT EXISTS employees (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  role_id TEXT NOT NULL REFERENCES roles(id),
  name TEXT NOT NULL,
  pin_hash TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE TABLE IF NOT EXISTS auth_sessions (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  node_device_id TEXT NOT NULL REFERENCES devices(id),
  client_device_id TEXT NOT NULL CHECK (client_device_id <> ''),
  employee_id TEXT NOT NULL REFERENCES employees(id),
  status TEXT NOT NULL CHECK (status IN ('active', 'revoked')),
  started_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  expires_at TEXT,
  revoked_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (device_id = node_device_id)
);

CREATE INDEX IF NOT EXISTS auth_sessions_device_employee_status ON auth_sessions(node_device_id, client_device_id, employee_id, status);

CREATE TABLE IF NOT EXISTS halls (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  name TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(restaurant_id, name)
);

CREATE TABLE IF NOT EXISTS tables (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  hall_id TEXT NOT NULL REFERENCES halls(id),
  name TEXT NOT NULL,
  seats INTEGER NOT NULL CHECK (seats >= 0),
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(hall_id, name)
);

CREATE INDEX IF NOT EXISTS tables_restaurant_hall ON tables(restaurant_id, hall_id);

CREATE TABLE IF NOT EXISTS catalog_items (
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

CREATE TABLE IF NOT EXISTS tax_profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tax_exempt INTEGER NOT NULL DEFAULT 0 CHECK (tax_exempt IN (0,1)),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tax_rules (
  id TEXT PRIMARY KEY,
  tax_profile_id TEXT NOT NULL REFERENCES tax_profiles(id),
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('percentage','fixed')),
  mode TEXT NOT NULL CHECK (mode IN ('inclusive','exclusive')),
  rate_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (rate_basis_points >= 0),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  compound INTEGER NOT NULL DEFAULT 0 CHECK (compound IN (0,1)),
  priority INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS tax_rules_profile_priority ON tax_rules(tax_profile_id, priority, id);

CREATE TABLE IF NOT EXISTS menu_items (
  id TEXT PRIMARY KEY,
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  name TEXT NOT NULL,
  price INTEGER NOT NULL CHECK (price >= 0),
  currency TEXT NOT NULL,
  tax_profile_id TEXT REFERENCES tax_profiles(id),
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE TABLE IF NOT EXISTS shifts (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  opened_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  closed_by_employee_id TEXT REFERENCES employees(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  opening_cash_amount INTEGER NOT NULL,
  closing_cash_amount INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS shifts_one_open_per_employee ON shifts(restaurant_id, opened_by_employee_id) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  edge_order_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'locked', 'closed', 'cancelled')),
  table_id TEXT NOT NULL REFERENCES tables(id),
  table_name TEXT NOT NULL,
  guest_count INTEGER NOT NULL,
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS orders_closed_restaurant_closed_at ON orders(restaurant_id, status, closed_at, id);
CREATE INDEX IF NOT EXISTS orders_closed_shift_closed_at ON orders(shift_id, status, closed_at, id);
CREATE INDEX IF NOT EXISTS orders_closed_device_closed_at ON orders(device_id, status, closed_at, id);

CREATE TABLE IF NOT EXISTS order_lines (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  total_price INTEGER NOT NULL CHECK (total_price >= 0),
  currency_code TEXT NOT NULL DEFAULT 'RUB' CHECK (currency_code GLOB '[A-Z][A-Z][A-Z]'),
  tax_profile_id TEXT REFERENCES tax_profiles(id),
  course TEXT,
  comment TEXT,
  status TEXT NOT NULL CHECK (status IN ('active', 'cancelled', 'voided')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
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

CREATE TABLE IF NOT EXISTS checks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL UNIQUE REFERENCES orders(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'paid', 'refunded', 'voided')),
  currency_code TEXT NOT NULL DEFAULT 'RUB' CHECK (currency_code GLOB '[A-Z][A-Z][A-Z]'),
  subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
  discount_total INTEGER NOT NULL CHECK (discount_total >= 0),
  surcharge_total INTEGER NOT NULL DEFAULT 0 CHECK (surcharge_total >= 0),
  tax_total INTEGER NOT NULL CHECK (tax_total >= 0),
  total INTEGER NOT NULL CHECK (total >= 0),
  paid_total INTEGER NOT NULL CHECK (paid_total >= 0),
  remaining_total INTEGER NOT NULL DEFAULT 0 CHECK (remaining_total >= 0),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  closed_at TEXT NOT NULL,
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS prechecks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  status TEXT NOT NULL CHECK (status IN ('issued', 'closed', 'cancelled', 'superseded')),
  version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
  supersedes_precheck_id TEXT CHECK (supersedes_precheck_id IS NULL OR supersedes_precheck_id <> ''),
  currency_code TEXT NOT NULL DEFAULT 'RUB' CHECK (currency_code GLOB '[A-Z][A-Z][A-Z]'),
  subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
  discount_total INTEGER NOT NULL CHECK (discount_total >= 0),
  surcharge_total INTEGER NOT NULL DEFAULT 0 CHECK (surcharge_total >= 0),
  tax_total INTEGER NOT NULL CHECK (tax_total >= 0),
  total INTEGER NOT NULL CHECK (total >= 0),
  paid_total INTEGER NOT NULL DEFAULT 0 CHECK (paid_total >= 0),
  remaining_total INTEGER NOT NULL DEFAULT 0 CHECK (remaining_total >= 0),
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL,
  issued_at TEXT NOT NULL,
  closed_at TEXT,
  cancelled_by_employee_id TEXT REFERENCES employees(id),
  cancellation_reason TEXT CHECK (cancellation_reason IS NULL OR cancellation_reason <> ''),
  CHECK (paid_total <= total),
  CHECK (closed_at IS NULL OR status IN ('closed', 'cancelled', 'superseded')),
  CHECK (closed_at IS NOT NULL OR status = 'issued')
);

CREATE UNIQUE INDEX IF NOT EXISTS prechecks_one_issued_per_order ON prechecks(order_id) WHERE status = 'issued';
CREATE UNIQUE INDEX IF NOT EXISTS prechecks_order_version ON prechecks(order_id, version);
CREATE INDEX IF NOT EXISTS prechecks_order_id_created_at ON prechecks(order_id, created_at);

CREATE TABLE IF NOT EXISTS order_line_discounts (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  order_line_id TEXT REFERENCES order_lines(id),
  pricing_policy_id TEXT,
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  reason TEXT,
  created_at TEXT NOT NULL,
  CHECK (scope = 'order' OR order_line_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS order_line_discounts_order_created_at ON order_line_discounts(order_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS order_line_discounts_order_application_index ON order_line_discounts(order_id, application_index);

CREATE TABLE IF NOT EXISTS order_level_discounts (
  id TEXT PRIMARY KEY,
  order_discount_id TEXT NOT NULL UNIQUE REFERENCES order_line_discounts(id),
  order_id TEXT NOT NULL REFERENCES orders(id),
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_surcharges (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  pricing_policy_id TEXT,
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  reason TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS order_surcharges_order_created_at ON order_surcharges(order_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS order_surcharges_order_application_index ON order_surcharges(order_id, application_index);

CREATE TRIGGER IF NOT EXISTS order_line_discounts_application_index_unique_insert
BEFORE INSERT ON order_line_discounts
WHEN EXISTS (
  SELECT 1 FROM order_surcharges s
  WHERE s.order_id = NEW.order_id AND s.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TRIGGER IF NOT EXISTS order_line_discounts_application_index_unique_update
BEFORE UPDATE OF order_id, application_index ON order_line_discounts
WHEN EXISTS (
  SELECT 1 FROM order_surcharges s
  WHERE s.order_id = NEW.order_id AND s.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TRIGGER IF NOT EXISTS order_surcharges_application_index_unique_insert
BEFORE INSERT ON order_surcharges
WHEN EXISTS (
  SELECT 1 FROM order_line_discounts d
  WHERE d.order_id = NEW.order_id AND d.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TRIGGER IF NOT EXISTS order_surcharges_application_index_unique_update
BEFORE UPDATE OF order_id, application_index ON order_surcharges
WHEN EXISTS (
  SELECT 1 FROM order_line_discounts d
  WHERE d.order_id = NEW.order_id AND d.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TABLE IF NOT EXISTS service_charge_rules (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
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
  manual INTEGER NOT NULL DEFAULT 0 CHECK (manual IN (0,1)),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);


CREATE INDEX IF NOT EXISTS pricing_policies_restaurant_active ON pricing_policies(restaurant_id, active, application_index);

CREATE TABLE IF NOT EXISTS precheck_lines (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  order_line_id TEXT NOT NULL,
  menu_item_id TEXT NOT NULL,
  catalog_item_id TEXT NOT NULL,
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price_minor INTEGER NOT NULL CHECK (unit_price_minor >= 0),
  subtotal_minor INTEGER NOT NULL CHECK (subtotal_minor >= 0),
  discount_total_minor INTEGER NOT NULL CHECK (discount_total_minor >= 0),
  surcharge_total_minor INTEGER NOT NULL CHECK (surcharge_total_minor >= 0),
  taxable_base_minor INTEGER NOT NULL CHECK (taxable_base_minor >= 0),
  tax_total_minor INTEGER NOT NULL CHECK (tax_total_minor >= 0),
  tax_added_minor INTEGER NOT NULL DEFAULT 0 CHECK (tax_added_minor >= 0),
  total_minor INTEGER NOT NULL CHECK (total_minor >= 0),
  currency_code TEXT NOT NULL CHECK (currency_code GLOB '[A-Z][A-Z][A-Z]'),
  tax_profile_id TEXT
);

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

CREATE TABLE IF NOT EXISTS precheck_discounts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  discount_id TEXT NOT NULL,
  pricing_policy_id TEXT,
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  order_line_id TEXT,
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL CHECK (amount_minor >= 0),
  reason TEXT
);

CREATE TABLE IF NOT EXISTS precheck_surcharges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  surcharge_id TEXT NOT NULL,
  pricing_policy_id TEXT,
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL CHECK (amount_minor >= 0),
  reason TEXT
);

CREATE TABLE IF NOT EXISTS precheck_taxes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  order_line_id TEXT NOT NULL,
  tax_profile_id TEXT NOT NULL,
  tax_rule_id TEXT NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('percentage','fixed')),
  mode TEXT NOT NULL CHECK (mode IN ('inclusive','exclusive')),
  rate_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (rate_basis_points >= 0),
  taxable_base_minor INTEGER NOT NULL CHECK (taxable_base_minor >= 0),
  tax_amount_minor INTEGER NOT NULL CHECK (tax_amount_minor >= 0),
  compound INTEGER NOT NULL DEFAULT 0 CHECK (compound IN (0,1)),
  priority INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS payments (
  id TEXT PRIMARY KEY,
  edge_payment_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'other')),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('captured', 'refunded', 'failed')),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  provider_name TEXT CHECK (provider_name IS NULL OR provider_name <> ''),
  provider_transaction_id TEXT CHECK (provider_transaction_id IS NULL OR provider_transaction_id <> ''),
  provider_reference TEXT CHECK (provider_reference IS NULL OR provider_reference <> ''),
  fingerprint_hash TEXT CHECK (fingerprint_hash IS NULL OR fingerprint_hash <> ''),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS payments_precheck_id_created_at ON payments(precheck_id, created_at);
CREATE INDEX IF NOT EXISTS payments_provider_transaction_id ON payments(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS payments_fingerprint_hash ON payments(fingerprint_hash) WHERE fingerprint_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS payment_attempts (
  id TEXT PRIMARY KEY,
  payment_id TEXT NOT NULL REFERENCES payments(id),
  attempt_no INTEGER NOT NULL CHECK (attempt_no > 0),
  method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'other')),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('captured', 'refunded', 'failed')),
  provider_name TEXT CHECK (provider_name IS NULL OR provider_name <> ''),
  provider_transaction_id TEXT CHECK (provider_transaction_id IS NULL OR provider_transaction_id <> ''),
  provider_reference TEXT CHECK (provider_reference IS NULL OR provider_reference <> ''),
  fingerprint_hash TEXT CHECK (fingerprint_hash IS NULL OR fingerprint_hash <> ''),
  attempted_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(payment_id, attempt_no)
);

CREATE INDEX IF NOT EXISTS payment_attempts_payment_id_attempt_no ON payment_attempts(payment_id, attempt_no);
CREATE INDEX IF NOT EXISTS payment_attempts_provider_transaction_id ON payment_attempts(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS financial_operations (
  id TEXT PRIMARY KEY,
  edge_operation_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  original_shift_id TEXT NOT NULL REFERENCES shifts(id),
  check_id TEXT NOT NULL REFERENCES checks(id),
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  operation_type TEXT NOT NULL CHECK (operation_type IN ('cancellation','refund')),
  operation_kind TEXT NOT NULL CHECK (operation_kind IN ('full','partial')),
  status TEXT NOT NULL CHECK (status = 'recorded'),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL CHECK (currency GLOB '[A-Z][A-Z][A-Z]'),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  inventory_disposition TEXT NOT NULL CHECK (inventory_disposition IN ('no_stock_effect','return_to_stock','write_off_waste','manual_review')),
  reason TEXT NOT NULL CHECK (reason <> ''),
  created_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  approved_by_employee_id TEXT REFERENCES employees(id),
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS financial_operations_check_type_created_at ON financial_operations(check_id, operation_type, created_at);
CREATE INDEX IF NOT EXISTS financial_operations_shift_created_at ON financial_operations(shift_id, created_at);

CREATE TABLE IF NOT EXISTS financial_operation_items (
  id TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL REFERENCES financial_operations(id),
  scope TEXT NOT NULL CHECK (scope IN ('whole_check','order_line','modifier_line','service_charge','tip','payment')),
  order_line_id TEXT REFERENCES order_lines(id),
  payment_id TEXT REFERENCES payments(id),
  quantity INTEGER CHECK (quantity IS NULL OR quantity > 0),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL CHECK (currency GLOB '[A-Z][A-Z][A-Z]'),
  tax_amount INTEGER NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL,
  CHECK (scope <> 'order_line' OR order_line_id IS NOT NULL),
  CHECK (scope <> 'payment' OR payment_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS financial_operation_items_operation_id ON financial_operation_items(operation_id);
CREATE INDEX IF NOT EXISTS financial_operation_items_payment_id ON financial_operation_items(payment_id) WHERE payment_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS financial_operation_items_order_line_id ON financial_operation_items(order_line_id) WHERE order_line_id IS NOT NULL;

CREATE TRIGGER IF NOT EXISTS financial_operations_no_update
BEFORE UPDATE ON financial_operations
BEGIN
  SELECT RAISE(ABORT, 'financial_operations are append-only');
END;

CREATE TRIGGER IF NOT EXISTS financial_operations_no_delete
BEFORE DELETE ON financial_operations
BEGIN
  SELECT RAISE(ABORT, 'financial_operations are append-only');
END;

CREATE TRIGGER IF NOT EXISTS financial_operation_items_no_update
BEFORE UPDATE ON financial_operation_items
BEGIN
  SELECT RAISE(ABORT, 'financial_operation_items are append-only');
END;

CREATE TRIGGER IF NOT EXISTS financial_operation_items_no_delete
BEFORE DELETE ON financial_operation_items
BEGIN
  SELECT RAISE(ABORT, 'financial_operation_items are append-only');
END;

CREATE TABLE IF NOT EXISTS cash_sessions (
  id TEXT PRIMARY KEY,
  edge_cash_session_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  opened_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  closed_by_employee_id TEXT REFERENCES employees(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  opening_cash_amount INTEGER NOT NULL CHECK (opening_cash_amount >= 0),
  closing_cash_amount INTEGER CHECK (closing_cash_amount IS NULL OR closing_cash_amount >= 0),
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS cash_sessions_one_open_per_device ON cash_sessions(device_id) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS cash_sessions_shift_id ON cash_sessions(shift_id);

CREATE TABLE IF NOT EXISTS cash_drawer_events (
  id TEXT PRIMARY KEY,
  edge_cash_drawer_event_id TEXT NOT NULL UNIQUE,
  cash_session_id TEXT NOT NULL REFERENCES cash_sessions(id),
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  created_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  event_type TEXT NOT NULL CHECK (event_type IN ('cash_in', 'cash_out', 'no_sale', 'cash_count')),
  amount INTEGER NOT NULL CHECK (amount >= 0),
  reason TEXT CHECK (reason IS NULL OR reason <> ''),
  note TEXT CHECK (note IS NULL OR note <> ''),
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS cash_drawer_events_cash_session_created_at ON cash_drawer_events(cash_session_id, created_at);
CREATE INDEX IF NOT EXISTS cash_drawer_events_shift_created_at ON cash_drawer_events(shift_id, created_at);

CREATE TABLE IF NOT EXISTS manager_override_audit (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  node_device_id TEXT NOT NULL REFERENCES devices(id),
  client_device_id TEXT CHECK (client_device_id IS NULL OR client_device_id <> ''),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  order_id TEXT NOT NULL REFERENCES orders(id),
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  manager_employee_id TEXT NOT NULL REFERENCES employees(id),
  actor_employee_id TEXT REFERENCES employees(id),
  session_id TEXT REFERENCES auth_sessions(id),
  action TEXT NOT NULL CHECK (action IN ('cancel_precheck')),
  reason TEXT NOT NULL CHECK (reason <> ''),
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  CHECK (device_id = node_device_id)
);

CREATE INDEX IF NOT EXISTS manager_override_audit_precheck_created_at ON manager_override_audit(precheck_id, created_at);
CREATE INDEX IF NOT EXISTS manager_override_audit_manager_created_at ON manager_override_audit(manager_employee_id, created_at);

CREATE TABLE IF NOT EXISTS recipe_versions (
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
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(dish_catalog_item_id, version)
);

CREATE TABLE IF NOT EXISTS recipe_lines (
  id TEXT PRIMARY KEY,
  recipe_version_id TEXT NOT NULL REFERENCES recipe_versions(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit TEXT NOT NULL,
  loss_percent INTEGER NOT NULL CHECK (loss_percent >= 0 AND loss_percent <= 100),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(recipe_version_id, catalog_item_id)
);

CREATE TABLE IF NOT EXISTS stop_lists (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  catalog_item_id TEXT NOT NULL,
  available_quantity REAL,
  source TEXT NOT NULL,
  reason TEXT,
  active INTEGER NOT NULL CHECK (active IN (0,1)),
  cloud_version INTEGER,
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS stop_lists_restaurant_item ON stop_lists(restaurant_id, catalog_item_id);

CREATE TABLE IF NOT EXISTS cloud_master_sync_state (
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

CREATE INDEX IF NOT EXISTS cloud_master_sync_state_node_status ON cloud_master_sync_state(node_device_id, status);

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

CREATE TABLE IF NOT EXISTS local_event_log (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE,
  command_id TEXT NOT NULL,
  envelope_version TEXT NOT NULL,
  event_type TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  client_device_id TEXT CHECK (client_device_id IS NULL OR client_device_id <> ''),
  shift_id TEXT CHECK (shift_id IS NULL OR shift_id <> ''),
  actor_employee_id TEXT REFERENCES employees(id),
  session_id TEXT REFERENCES auth_sessions(id),
  payload_json TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  CHECK (device_id = node_device_id)
);

CREATE INDEX IF NOT EXISTS local_event_log_created_at ON local_event_log(created_at);
CREATE INDEX IF NOT EXISTS local_event_log_event_type_created_at ON local_event_log(event_type, created_at);
CREATE INDEX IF NOT EXISTS local_event_log_command_id_created_at ON local_event_log(command_id, created_at);

CREATE TABLE IF NOT EXISTS pos_sync_outbox (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL,
  sequence_no INTEGER NOT NULL UNIQUE CHECK (sequence_no > 0),
  origin TEXT NOT NULL CHECK (origin IN ('edge_device', 'cloud_sync', 'system_seed')),
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  device_id TEXT NOT NULL CHECK (device_id <> ''),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  client_device_id TEXT CHECK (client_device_id IS NULL OR client_device_id <> ''),
  actor_employee_id TEXT REFERENCES employees(id),
  session_id TEXT REFERENCES auth_sessions(id),
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  command_type TEXT NOT NULL,
  sync_direction TEXT NOT NULL DEFAULT 'edge_to_cloud' CHECK (sync_direction IN ('edge_to_cloud','cloud_to_edge','local_only')),
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'sent', 'failed', 'suspended')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  next_retry_at TEXT,
  locked_at TEXT,
  locked_by TEXT CHECK (locked_by IS NULL OR locked_by <> ''),
  sent_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (status = 'processing' OR (locked_at IS NULL AND locked_by IS NULL)),
  CHECK ((locked_at IS NULL AND locked_by IS NULL) OR (locked_at IS NOT NULL AND locked_by IS NOT NULL)),
  CHECK (sent_at IS NULL OR status = 'sent'),
  CHECK (device_id = node_device_id)
);

CREATE INDEX IF NOT EXISTS pos_sync_outbox_status_sequence_no ON pos_sync_outbox(status, sequence_no);
CREATE INDEX IF NOT EXISTS pos_sync_outbox_pending_retry_sequence ON pos_sync_outbox(next_retry_at, sequence_no) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS pos_sync_outbox_processing_locked_at ON pos_sync_outbox(locked_at) WHERE status = 'processing';
CREATE INDEX IF NOT EXISTS pos_sync_outbox_command_id_created_at ON pos_sync_outbox(command_id, created_at);

-- === 002_runtime_schema_repair.sql ===
-- POS Edge SQLite runtime schema repair для старых pre-pilot БД.
-- SQLite не поддерживает ADD COLUMN IF NOT EXISTS, поэтому блоки ниже
-- выполняются migration framework только если указанная колонка отсутствует.

CREATE TABLE IF NOT EXISTS restaurants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  timezone TEXT NOT NULL,
  currency TEXT NOT NULL CHECK (currency GLOB '[A-Z][A-Z][A-Z]'),
  business_day_mode TEXT NOT NULL DEFAULT 'standard' CHECK (business_day_mode IN ('standard', '24_7')),
  business_day_boundary_local_time TEXT NOT NULL DEFAULT '06:00' CHECK (business_day_boundary_local_time GLOB '[0-2][0-9]:[0-5][0-9]'),
  active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER,
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);

CREATE TABLE IF NOT EXISTS shifts (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  opened_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  closed_by_employee_id TEXT REFERENCES employees(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  opening_cash_amount INTEGER NOT NULL,
  closing_cash_amount INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS shifts_one_open_per_employee ON shifts(restaurant_id, opened_by_employee_id) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS checks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL UNIQUE REFERENCES orders(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'closed', 'voided')),
  subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
  discount_total INTEGER NOT NULL CHECK (discount_total >= 0),
  tax_total INTEGER NOT NULL CHECK (tax_total >= 0),
  total INTEGER NOT NULL CHECK (total >= 0),
  paid_total INTEGER NOT NULL CHECK (paid_total >= 0),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  closed_at TEXT NOT NULL,
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS prechecks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  status TEXT NOT NULL CHECK (status IN ('issued', 'cancelled', 'superseded')),
  version INTEGER NOT NULL CHECK (version > 0),
  supersedes_precheck_id TEXT REFERENCES prechecks(id),
  subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
  discount_total INTEGER NOT NULL CHECK (discount_total >= 0),
  tax_total INTEGER NOT NULL CHECK (tax_total >= 0),
  total INTEGER NOT NULL CHECK (total >= 0),
  paid_total INTEGER NOT NULL DEFAULT 0 CHECK (paid_total >= 0),
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL,
  issued_at TEXT NOT NULL,
  closed_at TEXT,
  cancelled_by_employee_id TEXT REFERENCES employees(id),
  cancellation_reason TEXT,
  CHECK (subtotal - discount_total + tax_total = total),
  CHECK (status = 'issued' OR closed_at IS NOT NULL),
  CHECK (status <> 'cancelled' OR cancelled_by_employee_id IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS edge_provisioning_state (
  id TEXT PRIMARY KEY CHECK (id = 'local'),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  cloud_url TEXT,
  license_url TEXT,
  restaurant_id TEXT,
  status TEXT NOT NULL CHECK (status IN ('not_configured','pending_admin_approval','assigned_downloading_snapshot','paired','error')),
  credentials_type TEXT,
  credentials_token TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS prechecks_one_issued_per_order ON prechecks(order_id) WHERE status = 'issued';
CREATE UNIQUE INDEX IF NOT EXISTS prechecks_order_version ON prechecks(order_id, version);
CREATE INDEX IF NOT EXISTS prechecks_order_id_created_at ON prechecks(order_id, created_at);

CREATE TABLE IF NOT EXISTS payments (
  id TEXT PRIMARY KEY,
  edge_payment_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'other')),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('captured', 'refunded', 'failed')),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  provider_name TEXT CHECK (provider_name IS NULL OR provider_name <> ''),
  provider_transaction_id TEXT CHECK (provider_transaction_id IS NULL OR provider_transaction_id <> ''),
  provider_reference TEXT CHECK (provider_reference IS NULL OR provider_reference <> ''),
  fingerprint_hash TEXT CHECK (fingerprint_hash IS NULL OR fingerprint_hash <> ''),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS payments_precheck_id_created_at ON payments(precheck_id, created_at);
CREATE INDEX IF NOT EXISTS payments_provider_transaction_id ON payments(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS payments_fingerprint_hash ON payments(fingerprint_hash) WHERE fingerprint_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS payment_attempts (
  id TEXT PRIMARY KEY,
  payment_id TEXT NOT NULL REFERENCES payments(id),
  attempt_no INTEGER NOT NULL CHECK (attempt_no > 0),
  method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'other')),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('captured', 'refunded', 'failed')),
  provider_name TEXT CHECK (provider_name IS NULL OR provider_name <> ''),
  provider_transaction_id TEXT CHECK (provider_transaction_id IS NULL OR provider_transaction_id <> ''),
  provider_reference TEXT CHECK (provider_reference IS NULL OR provider_reference <> ''),
  fingerprint_hash TEXT CHECK (fingerprint_hash IS NULL OR fingerprint_hash <> ''),
  attempted_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS payment_attempts_payment_id_attempt_no ON payment_attempts(payment_id, attempt_no);
CREATE INDEX IF NOT EXISTS payment_attempts_provider_transaction_id ON payment_attempts(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS cash_sessions (
  id TEXT PRIMARY KEY,
  edge_cash_session_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  opened_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  closed_by_employee_id TEXT REFERENCES employees(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  business_date_local TEXT NOT NULL CHECK (business_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  opening_cash_amount INTEGER NOT NULL CHECK (opening_cash_amount >= 0),
  closing_cash_amount INTEGER CHECK (closing_cash_amount IS NULL OR closing_cash_amount >= 0),
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS cash_sessions_one_open_per_device ON cash_sessions(device_id) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS cash_sessions_shift_id ON cash_sessions(shift_id);

-- sqlite:repair-column restaurants business_day_mode
ALTER TABLE restaurants ADD COLUMN business_day_mode TEXT NOT NULL DEFAULT 'standard';

-- sqlite:repair-column restaurants business_day_boundary_local_time
ALTER TABLE restaurants ADD COLUMN business_day_boundary_local_time TEXT NOT NULL DEFAULT '06:00';

-- sqlite:repair-column shifts business_date_local
ALTER TABLE shifts ADD COLUMN business_date_local TEXT NOT NULL DEFAULT '1970-01-01';

-- sqlite:repair-column prechecks snapshot
ALTER TABLE prechecks ADD COLUMN snapshot TEXT NOT NULL DEFAULT '{}';

-- sqlite:repair-column checks business_date_local
ALTER TABLE checks ADD COLUMN business_date_local TEXT NOT NULL DEFAULT '1970-01-01';

-- sqlite:repair-column checks closed_at
ALTER TABLE checks ADD COLUMN closed_at TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z';

-- sqlite:repair-column checks snapshot
ALTER TABLE checks ADD COLUMN snapshot TEXT NOT NULL DEFAULT '{}';

-- sqlite:repair-column payments business_date_local
ALTER TABLE payments ADD COLUMN business_date_local TEXT NOT NULL DEFAULT '1970-01-01';

-- sqlite:repair-column cash_sessions business_date_local
ALTER TABLE cash_sessions ADD COLUMN business_date_local TEXT NOT NULL DEFAULT '1970-01-01';

-- sqlite:repair-column menu_items tax_profile_id
ALTER TABLE menu_items ADD COLUMN tax_profile_id TEXT;

-- sqlite:repair-column order_lines currency_code
ALTER TABLE order_lines ADD COLUMN currency_code TEXT NOT NULL DEFAULT 'RUB';

-- sqlite:repair-column order_lines tax_profile_id
ALTER TABLE order_lines ADD COLUMN tax_profile_id TEXT;

-- sqlite:repair-column order_lines course
ALTER TABLE order_lines ADD COLUMN course TEXT;

-- sqlite:repair-column order_lines comment
ALTER TABLE order_lines ADD COLUMN comment TEXT;

-- sqlite:repair-column prechecks currency_code
ALTER TABLE prechecks ADD COLUMN currency_code TEXT NOT NULL DEFAULT 'RUB';

-- sqlite:repair-column prechecks surcharge_total
ALTER TABLE prechecks ADD COLUMN surcharge_total INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-column prechecks remaining_total
ALTER TABLE prechecks ADD COLUMN remaining_total INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-column checks currency_code
ALTER TABLE checks ADD COLUMN currency_code TEXT NOT NULL DEFAULT 'RUB';

-- sqlite:repair-column checks surcharge_total
ALTER TABLE checks ADD COLUMN surcharge_total INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-column checks remaining_total
ALTER TABLE checks ADD COLUMN remaining_total INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-sql
CREATE TABLE IF NOT EXISTS tax_profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tax_exempt INTEGER NOT NULL DEFAULT 0 CHECK (tax_exempt IN (0,1)),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tax_rules (
  id TEXT PRIMARY KEY,
  tax_profile_id TEXT NOT NULL REFERENCES tax_profiles(id),
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('percentage','fixed')),
  mode TEXT NOT NULL CHECK (mode IN ('inclusive','exclusive')),
  rate_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (rate_basis_points >= 0),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  compound INTEGER NOT NULL DEFAULT 0 CHECK (compound IN (0,1)),
  priority INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS tax_rules_profile_priority ON tax_rules(tax_profile_id, priority, id);

CREATE TABLE IF NOT EXISTS order_line_discounts (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  order_line_id TEXT REFERENCES order_lines(id),
  pricing_policy_id TEXT,
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  reason TEXT,
  created_at TEXT NOT NULL,
  CHECK (scope = 'order' OR order_line_id IS NOT NULL)
);

-- sqlite:repair-column order_line_discounts application_index
ALTER TABLE order_line_discounts ADD COLUMN application_index INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-column order_line_discounts pricing_policy_id
ALTER TABLE order_line_discounts ADD COLUMN pricing_policy_id TEXT;

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS order_line_discounts_order_created_at ON order_line_discounts(order_id, created_at);

CREATE TABLE IF NOT EXISTS order_level_discounts (
  id TEXT PRIMARY KEY,
  order_discount_id TEXT NOT NULL UNIQUE REFERENCES order_line_discounts(id),
  order_id TEXT NOT NULL REFERENCES orders(id),
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_surcharges (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  pricing_policy_id TEXT,
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  reason TEXT,
  created_at TEXT NOT NULL
);

-- sqlite:repair-column order_surcharges application_index
ALTER TABLE order_surcharges ADD COLUMN application_index INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-column order_surcharges pricing_policy_id
ALTER TABLE order_surcharges ADD COLUMN pricing_policy_id TEXT;

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS order_surcharges_order_created_at ON order_surcharges(order_id, created_at);

CREATE TABLE IF NOT EXISTS service_charge_rules (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS precheck_lines (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  order_line_id TEXT NOT NULL,
  menu_item_id TEXT NOT NULL,
  catalog_item_id TEXT NOT NULL,
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price_minor INTEGER NOT NULL CHECK (unit_price_minor >= 0),
  subtotal_minor INTEGER NOT NULL CHECK (subtotal_minor >= 0),
  discount_total_minor INTEGER NOT NULL CHECK (discount_total_minor >= 0),
  surcharge_total_minor INTEGER NOT NULL CHECK (surcharge_total_minor >= 0),
  taxable_base_minor INTEGER NOT NULL CHECK (taxable_base_minor >= 0),
  tax_total_minor INTEGER NOT NULL CHECK (tax_total_minor >= 0),
  tax_added_minor INTEGER NOT NULL DEFAULT 0 CHECK (tax_added_minor >= 0),
  total_minor INTEGER NOT NULL CHECK (total_minor >= 0),
  currency_code TEXT NOT NULL CHECK (currency_code GLOB '[A-Z][A-Z][A-Z]'),
  tax_profile_id TEXT
);

CREATE TABLE IF NOT EXISTS precheck_discounts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  discount_id TEXT NOT NULL,
  pricing_policy_id TEXT,
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  order_line_id TEXT,
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL CHECK (amount_minor >= 0),
  reason TEXT
);

-- sqlite:repair-column precheck_discounts application_index
ALTER TABLE precheck_discounts ADD COLUMN application_index INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-column precheck_discounts pricing_policy_id
ALTER TABLE precheck_discounts ADD COLUMN pricing_policy_id TEXT;

-- sqlite:repair-sql
CREATE TABLE IF NOT EXISTS precheck_surcharges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  surcharge_id TEXT NOT NULL,
  pricing_policy_id TEXT,
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
  application_index INTEGER NOT NULL CHECK (application_index > 0),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL CHECK (amount_minor >= 0),
  reason TEXT
);

-- sqlite:repair-column precheck_surcharges application_index
ALTER TABLE precheck_surcharges ADD COLUMN application_index INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-column precheck_surcharges pricing_policy_id
ALTER TABLE precheck_surcharges ADD COLUMN pricing_policy_id TEXT;

-- sqlite:repair-sql
WITH ordered_modifiers AS (
  SELECT modifier_type, id, ROW_NUMBER() OVER (PARTITION BY order_id ORDER BY created_at, modifier_type, id) * 10 AS next_application_index
  FROM (
    SELECT 'discount' AS modifier_type, id, order_id, created_at
    FROM order_line_discounts
    WHERE application_index <= 0
    UNION ALL
    SELECT 'surcharge' AS modifier_type, id, order_id, created_at
    FROM order_surcharges
    WHERE application_index <= 0
  )
)
UPDATE order_line_discounts
SET application_index = (
  SELECT next_application_index
  FROM ordered_modifiers
  WHERE ordered_modifiers.modifier_type = 'discount'
    AND ordered_modifiers.id = order_line_discounts.id
)
WHERE application_index <= 0;

-- sqlite:repair-sql
WITH ordered_modifiers AS (
  SELECT modifier_type, id, ROW_NUMBER() OVER (PARTITION BY order_id ORDER BY created_at, modifier_type, id) * 10 AS next_application_index
  FROM (
    SELECT 'discount' AS modifier_type, id, order_id, created_at
    FROM order_line_discounts
    UNION ALL
    SELECT 'surcharge' AS modifier_type, id, order_id, created_at
    FROM order_surcharges
    WHERE application_index <= 0
  )
)
UPDATE order_surcharges
SET application_index = (
  SELECT next_application_index
  FROM ordered_modifiers
  WHERE ordered_modifiers.modifier_type = 'surcharge'
    AND ordered_modifiers.id = order_surcharges.id
)
WHERE application_index <= 0;

-- sqlite:repair-sql
UPDATE precheck_discounts
SET application_index = COALESCE(
  (SELECT application_index FROM order_line_discounts WHERE order_line_discounts.id = precheck_discounts.discount_id),
  100000 + id * 20
)
WHERE application_index <= 0;

-- sqlite:repair-sql
UPDATE precheck_surcharges
SET application_index = COALESCE(
  (SELECT application_index FROM order_surcharges WHERE order_surcharges.id = precheck_surcharges.surcharge_id),
  100000 + id * 20 + 10
)
WHERE application_index <= 0;

CREATE UNIQUE INDEX IF NOT EXISTS order_line_discounts_order_application_index ON order_line_discounts(order_id, application_index);
CREATE UNIQUE INDEX IF NOT EXISTS order_surcharges_order_application_index ON order_surcharges(order_id, application_index);

CREATE TRIGGER IF NOT EXISTS order_line_discounts_application_index_unique_insert
BEFORE INSERT ON order_line_discounts
WHEN EXISTS (
  SELECT 1 FROM order_surcharges s
  WHERE s.order_id = NEW.order_id AND s.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TRIGGER IF NOT EXISTS order_line_discounts_application_index_unique_update
BEFORE UPDATE OF order_id, application_index ON order_line_discounts
WHEN EXISTS (
  SELECT 1 FROM order_surcharges s
  WHERE s.order_id = NEW.order_id AND s.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TRIGGER IF NOT EXISTS order_surcharges_application_index_unique_insert
BEFORE INSERT ON order_surcharges
WHEN EXISTS (
  SELECT 1 FROM order_line_discounts d
  WHERE d.order_id = NEW.order_id AND d.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TRIGGER IF NOT EXISTS order_surcharges_application_index_unique_update
BEFORE UPDATE OF order_id, application_index ON order_surcharges
WHEN EXISTS (
  SELECT 1 FROM order_line_discounts d
  WHERE d.order_id = NEW.order_id AND d.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;

CREATE TABLE IF NOT EXISTS precheck_taxes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  order_line_id TEXT NOT NULL,
  tax_profile_id TEXT NOT NULL,
  tax_rule_id TEXT NOT NULL,
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('percentage','fixed')),
  mode TEXT NOT NULL CHECK (mode IN ('inclusive','exclusive')),
  rate_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (rate_basis_points >= 0),
  taxable_base_minor INTEGER NOT NULL CHECK (taxable_base_minor >= 0),
  tax_amount_minor INTEGER NOT NULL CHECK (tax_amount_minor >= 0),
  compound INTEGER NOT NULL DEFAULT 0 CHECK (compound IN (0,1)),
  priority INTEGER NOT NULL DEFAULT 0
);

-- sqlite:repair-column precheck_lines tax_added_minor
ALTER TABLE precheck_lines ADD COLUMN tax_added_minor INTEGER NOT NULL DEFAULT 0;

-- sqlite:repair-sql
UPDATE restaurants
SET business_day_mode = 'standard'
WHERE business_day_mode IS NULL OR business_day_mode = '';

-- sqlite:repair-sql
UPDATE restaurants
SET business_day_boundary_local_time = '06:00'
WHERE business_day_boundary_local_time IS NULL OR business_day_boundary_local_time = '';

-- sqlite:repair-sql
UPDATE shifts
SET business_date_local = substr(opened_at, 1, 10)
WHERE (business_date_local = '1970-01-01' OR business_date_local = '')
  AND opened_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]*';

-- sqlite:repair-sql
UPDATE checks
SET business_date_local = substr(COALESCE(NULLIF(closed_at, '1970-01-01T00:00:00Z'), created_at), 1, 10)
WHERE (business_date_local = '1970-01-01' OR business_date_local = '')
  AND COALESCE(NULLIF(closed_at, '1970-01-01T00:00:00Z'), created_at) GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]*';

-- sqlite:repair-sql
UPDATE checks
SET closed_at = created_at
WHERE (closed_at = '1970-01-01T00:00:00Z' OR closed_at = '')
  AND created_at IS NOT NULL
  AND created_at <> '';

-- sqlite:repair-sql
UPDATE payments
SET business_date_local = substr(created_at, 1, 10)
WHERE (business_date_local = '1970-01-01' OR business_date_local = '')
  AND created_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]*';

-- sqlite:repair-sql
UPDATE cash_sessions
SET business_date_local = substr(opened_at, 1, 10)
WHERE (business_date_local = '1970-01-01' OR business_date_local = '')
  AND opened_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]*';

-- sqlite:repair-sql
UPDATE prechecks
SET remaining_total = total - paid_total
WHERE remaining_total = 0 AND total >= paid_total;

-- sqlite:repair-sql
UPDATE checks
SET remaining_total = total - paid_total
WHERE remaining_total = 0 AND total >= paid_total;

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS checks_business_date_closed_at ON checks(business_date_local, closed_at, id);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS checks_order_id_closed_at ON checks(order_id, closed_at);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS orders_closed_restaurant_created_at ON orders(restaurant_id, status, created_at, id);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS payments_business_date_shift_created_at ON payments(business_date_local, shift_id, created_at, id);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS financial_operations_restaurant_business_date_type_created_at ON financial_operations(restaurant_id, business_date_local, operation_type, created_at, id);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS financial_operations_original_shift_created_at ON financial_operations(original_shift_id, created_at, id);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS financial_operations_check_created_at ON financial_operations(check_id, created_at, id);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS local_event_log_occurred_at ON local_event_log(occurred_at, id);

-- sqlite:repair-sql
CREATE INDEX IF NOT EXISTS pos_sync_outbox_created_at ON pos_sync_outbox(created_at, id);

-- === 003_pricing_policy_sync_foundation.sql ===
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

-- === 004_catalog_v2_modifiers_runtime.sql ===
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
  manual INTEGER NOT NULL DEFAULT 0 CHECK (manual IN (0,1)),
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
