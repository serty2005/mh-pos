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
