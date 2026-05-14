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
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  reason TEXT,
  created_at TEXT NOT NULL,
  CHECK (scope = 'order' OR order_line_id IS NOT NULL)
);

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
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL DEFAULT 0 CHECK (amount_minor >= 0),
  value_basis_points INTEGER NOT NULL DEFAULT 0 CHECK (value_basis_points >= 0),
  reason TEXT,
  created_at TEXT NOT NULL
);

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
  scope TEXT NOT NULL CHECK (scope IN ('line','order')),
  order_line_id TEXT,
  amount_kind TEXT NOT NULL CHECK (amount_kind IN ('percentage','fixed')),
  amount_minor INTEGER NOT NULL CHECK (amount_minor >= 0),
  reason TEXT
);

CREATE TABLE IF NOT EXISTS precheck_surcharges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  precheck_id TEXT NOT NULL REFERENCES prechecks(id),
  surcharge_id TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('service_charge','pb1_service_fee','manual')),
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
