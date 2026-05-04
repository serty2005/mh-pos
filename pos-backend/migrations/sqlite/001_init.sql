CREATE TABLE restaurants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  timezone TEXT NOT NULL,
  currency TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE devices (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_code TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  active INTEGER NOT NULL,
  registered_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(restaurant_id, device_code)
);

CREATE TABLE roles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  permissions_json TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE employees (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  role_id TEXT NOT NULL REFERENCES roles(id),
  name TEXT NOT NULL,
  pin_hash TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE catalog_items (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL CHECK (type IN ('ingredient', 'dish', 'good')),
  name TEXT NOT NULL,
  sku TEXT NOT NULL UNIQUE,
  base_unit TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE menu_items (
  id TEXT PRIMARY KEY,
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  name TEXT NOT NULL,
  price INTEGER NOT NULL CHECK (price >= 0),
  currency TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE shifts (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  opened_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  closed_by_employee_id TEXT REFERENCES employees(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  opening_cash_amount INTEGER NOT NULL,
  closing_cash_amount INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX shifts_one_open_per_device ON shifts(device_id) WHERE status = 'open';

CREATE TABLE orders (
  id TEXT PRIMARY KEY,
  edge_order_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'closed', 'cancelled')),
  table_name TEXT NOT NULL,
  guest_count INTEGER NOT NULL,
  opened_at TEXT NOT NULL,
  closed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE order_lines (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  total_price INTEGER NOT NULL CHECK (total_price >= 0),
  status TEXT NOT NULL CHECK (status IN ('active', 'cancelled')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE checks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL UNIQUE REFERENCES orders(id),
  status TEXT NOT NULL CHECK (status IN ('open', 'paid', 'refunded', 'voided')),
  subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
  discount_total INTEGER NOT NULL CHECK (discount_total >= 0),
  tax_total INTEGER NOT NULL CHECK (tax_total >= 0),
  total INTEGER NOT NULL CHECK (total >= 0),
  paid_total INTEGER NOT NULL CHECK (paid_total >= 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE payments (
  id TEXT PRIMARY KEY,
  check_id TEXT NOT NULL REFERENCES checks(id),
  method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'other')),
  amount INTEGER NOT NULL CHECK (amount > 0),
  currency TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('captured', 'refunded', 'failed')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE pos_sync_outbox (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT,
  device_id TEXT,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  command_type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
  attempts INTEGER NOT NULL,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX pos_sync_outbox_status_created_at ON pos_sync_outbox(status, created_at);
