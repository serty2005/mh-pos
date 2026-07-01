-- Чистая инициализация SQLite-схемы POS-Edge.
-- Первая и единственная миграция создаёт БД с нуля.
-- Следующая нумерованная миграция (002_*.sql) — после первого боевого внедрения.

CREATE TABLE restaurants (
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
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(restaurant_id, device_code)
);
CREATE TABLE edge_node_identity (
  id TEXT PRIMARY KEY CHECK (id = 'local'),
  node_device_id TEXT NOT NULL REFERENCES devices(id),
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  status TEXT NOT NULL CHECK (status IN ('paired')),
  pairing_code_hash TEXT NOT NULL CHECK (pairing_code_hash <> ''),
  paired_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE edge_provisioning_state (
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
CREATE TABLE client_devices (
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
CREATE INDEX client_devices_restaurant_node_status ON client_devices(restaurant_id, node_device_id, status);
CREATE TABLE roles (
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
CREATE TABLE employees (
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
CREATE TABLE auth_sessions (
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
CREATE INDEX auth_sessions_device_employee_status ON auth_sessions(node_device_id, client_device_id, employee_id, status);
CREATE TABLE halls (
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
CREATE TABLE tables (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  hall_id TEXT REFERENCES halls(id),
  section_id TEXT NOT NULL REFERENCES restaurant_sections(id),
  name TEXT NOT NULL,
  seats INTEGER NOT NULL CHECK (seats >= 0),
  is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0,1)),
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(section_id, name)
);
CREATE INDEX tables_restaurant_hall ON tables(restaurant_id, hall_id);
CREATE INDEX tables_restaurant_section ON tables(restaurant_id, section_id);
CREATE UNIQUE INDEX tables_one_default_per_restaurant ON tables(restaurant_id) WHERE is_default = 1;
CREATE TRIGGER tables_section_hall_mode_insert
BEFORE INSERT ON tables
FOR EACH ROW
WHEN NOT EXISTS (
  SELECT 1 FROM restaurant_sections s
  WHERE s.id = NEW.section_id
    AND s.restaurant_id = NEW.restaurant_id
    AND s.mode = 'hall_section'
)
BEGIN
  SELECT RAISE(ABORT, 'table must reference hall_section restaurant section');
END;
CREATE TRIGGER tables_section_hall_mode_update
BEFORE UPDATE OF restaurant_id, section_id ON tables
FOR EACH ROW
WHEN NOT EXISTS (
  SELECT 1 FROM restaurant_sections s
  WHERE s.id = NEW.section_id
    AND s.restaurant_id = NEW.restaurant_id
    AND s.mode = 'hall_section'
)
BEGIN
  SELECT RAISE(ABORT, 'table must reference hall_section restaurant section');
END;
CREATE TABLE catalog_folders (
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
CREATE TABLE catalog_folder_parameters (
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
CREATE TABLE catalog_tags (
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
CREATE TABLE catalog_item_tags (
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  tag_id TEXT NOT NULL REFERENCES catalog_tags(id),
  restaurant_id TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  PRIMARY KEY(catalog_item_id, tag_id)
);
CREATE TABLE modifier_groups (
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
CREATE TABLE modifier_options (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  modifier_group_id TEXT NOT NULL REFERENCES modifier_groups(id),
  linked_catalog_item_id TEXT,
  name TEXT NOT NULL,
  price_minor INTEGER NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);
CREATE TABLE modifier_group_bindings (
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
CREATE TABLE menu_item_modifier_groups (
  menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
  modifier_group_id TEXT NOT NULL REFERENCES modifier_groups(id),
  sort_order INTEGER NOT NULL DEFAULT 0,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  PRIMARY KEY(menu_item_id, modifier_group_id)
);
CREATE TABLE tax_profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tax_exempt INTEGER NOT NULL DEFAULT 0 CHECK (tax_exempt IN (0,1)),
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
, cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0), cloud_updated_at TEXT, cloud_deleted_at TEXT, last_synced_at TEXT);
CREATE TABLE tax_rules (
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
, cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0), cloud_updated_at TEXT, cloud_deleted_at TEXT, last_synced_at TEXT);
CREATE INDEX tax_rules_profile_priority ON tax_rules(tax_profile_id, priority, id);
CREATE TABLE menu_items (
  id TEXT PRIMARY KEY,
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  category_id TEXT,
  tag_id TEXT,
  name TEXT NOT NULL,
  price INTEGER NOT NULL CHECK (price >= 0),
  currency TEXT NOT NULL,
  tax_profile_id TEXT REFERENCES tax_profiles(id),
  runtime_status TEXT NOT NULL DEFAULT 'available' CHECK (runtime_status IN ('available','unavailable','hidden')),
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT
);
CREATE TABLE shifts (
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
CREATE UNIQUE INDEX shifts_one_open_per_employee ON shifts(restaurant_id, opened_by_employee_id) WHERE status = 'open';
CREATE TABLE orders (
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
  cancelled_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX orders_closed_restaurant_closed_at ON orders(restaurant_id, status, closed_at, id);
CREATE INDEX orders_closed_shift_closed_at ON orders(shift_id, status, closed_at, id);
CREATE INDEX orders_closed_device_closed_at ON orders(device_id, status, closed_at, id);
CREATE TABLE order_lines (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  category_id TEXT,
  tag_id TEXT,
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
CREATE TABLE order_line_modifiers (
  id TEXT PRIMARY KEY,
  order_line_id TEXT NOT NULL REFERENCES order_lines(id) ON DELETE CASCADE,
  modifier_group_id TEXT NOT NULL REFERENCES modifier_groups(id),
  modifier_option_id TEXT NOT NULL REFERENCES modifier_options(id),
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
  total_price INTEGER NOT NULL CHECK (total_price >= 0)
);
CREATE TABLE kitchen_tickets (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  order_id TEXT NOT NULL REFERENCES orders(id),
  order_line_id TEXT NOT NULL UNIQUE REFERENCES order_lines(id),
  table_name TEXT NOT NULL,
  menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  name TEXT NOT NULL,
  quantity INTEGER NOT NULL CHECK (quantity > 0),
  unit_code TEXT NOT NULL CHECK (unit_code <> ''),
  station_routing_key TEXT NOT NULL DEFAULT '',
  course TEXT,
  comment TEXT,
  status TEXT NOT NULL CHECK (status IN ('new','accepted','in_progress','hold','ready','served','recall','cancelled')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX kitchen_tickets_restaurant_status_created_at ON kitchen_tickets(restaurant_id, status, created_at, id);
CREATE INDEX kitchen_tickets_order_id ON kitchen_tickets(order_id);
CREATE TABLE kitchen_ticket_events (
  id TEXT PRIMARY KEY,
  ticket_id TEXT NOT NULL REFERENCES kitchen_tickets(id),
  order_line_id TEXT NOT NULL REFERENCES order_lines(id),
  from_status TEXT NOT NULL CHECK (from_status IN ('new','accepted','in_progress','hold','ready','served','recall','cancelled')),
  to_status TEXT NOT NULL CHECK (to_status IN ('new','accepted','in_progress','hold','ready','served','recall','cancelled')),
  command_id TEXT NOT NULL,
  actor_employee_id TEXT REFERENCES employees(id),
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX kitchen_ticket_events_ticket_created_at ON kitchen_ticket_events(ticket_id, created_at);
CREATE INDEX kitchen_ticket_events_command_id ON kitchen_ticket_events(command_id);
CREATE TABLE kitchen_proposals (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL CHECK (restaurant_id <> ''),
  proposal_group_id TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL CHECK (kind IN ('catalog','recipe')),
  status TEXT NOT NULL CHECK (status IN ('draft','pending_sync','synced','approved','rejected','changes_requested','failed')),
  action TEXT NOT NULL CHECK (action <> ''),
  owner_catalog_item_id TEXT NOT NULL DEFAULT '',
  owner_catalog_suggestion_id TEXT NOT NULL DEFAULT '',
  recipe_version_id TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
  outbox_command_id TEXT NOT NULL UNIQUE CHECK (outbox_command_id <> ''),
  outbox_event_type TEXT NOT NULL CHECK (outbox_event_type IN ('CatalogItemChangeSuggested','RecipeChangeSuggested')),
  created_by_employee_id TEXT NOT NULL REFERENCES employees(id),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  cloud_version INTEGER,
  cloud_updated_at TEXT
);
CREATE INDEX kitchen_proposals_restaurant_kind_status ON kitchen_proposals(restaurant_id, kind, status, created_at);
CREATE INDEX kitchen_proposals_group ON kitchen_proposals(proposal_group_id);
CREATE INDEX kitchen_proposals_owner_recipe ON kitchen_proposals(owner_catalog_item_id, recipe_version_id);
CREATE TABLE checks (
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
  print_confirmed_at TEXT,
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE prechecks (
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
CREATE UNIQUE INDEX prechecks_one_issued_per_order ON prechecks(order_id) WHERE status = 'issued';
CREATE UNIQUE INDEX prechecks_order_version ON prechecks(order_id, version);
CREATE INDEX prechecks_order_id_created_at ON prechecks(order_id, created_at);
CREATE TABLE order_line_discounts (
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
CREATE INDEX order_line_discounts_order_created_at ON order_line_discounts(order_id, created_at);
CREATE UNIQUE INDEX order_line_discounts_order_application_index ON order_line_discounts(order_id, application_index);
CREATE TABLE order_level_discounts (
  id TEXT PRIMARY KEY,
  order_discount_id TEXT NOT NULL UNIQUE REFERENCES order_line_discounts(id),
  order_id TEXT NOT NULL REFERENCES orders(id),
  created_at TEXT NOT NULL
);
CREATE TABLE order_surcharges (
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
CREATE INDEX order_surcharges_order_created_at ON order_surcharges(order_id, created_at);
CREATE UNIQUE INDEX order_surcharges_order_application_index ON order_surcharges(order_id, application_index);
CREATE TRIGGER order_line_discounts_application_index_unique_insert
BEFORE INSERT ON order_line_discounts
WHEN EXISTS (
  SELECT 1 FROM order_surcharges s
  WHERE s.order_id = NEW.order_id AND s.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;
CREATE TRIGGER order_line_discounts_application_index_unique_update
BEFORE UPDATE OF order_id, application_index ON order_line_discounts
WHEN EXISTS (
  SELECT 1 FROM order_surcharges s
  WHERE s.order_id = NEW.order_id AND s.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;
CREATE TRIGGER order_surcharges_application_index_unique_insert
BEFORE INSERT ON order_surcharges
WHEN EXISTS (
  SELECT 1 FROM order_line_discounts d
  WHERE d.order_id = NEW.order_id AND d.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;
CREATE TRIGGER order_surcharges_application_index_unique_update
BEFORE UPDATE OF order_id, application_index ON order_surcharges
WHEN EXISTS (
  SELECT 1 FROM order_line_discounts d
  WHERE d.order_id = NEW.order_id AND d.application_index = NEW.application_index
)
BEGIN
  SELECT RAISE(ABORT, 'duplicate application_index for order financial modifiers');
END;
CREATE TABLE service_charge_rules (
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
, cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0), cloud_updated_at TEXT, cloud_deleted_at TEXT, last_synced_at TEXT);
CREATE TABLE pricing_policies (
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
CREATE INDEX pricing_policies_restaurant_active ON pricing_policies(restaurant_id, active, application_index);
CREATE TABLE precheck_lines (
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
CREATE TABLE precheck_line_modifiers (
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
CREATE TABLE precheck_discounts (
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
CREATE TABLE precheck_surcharges (
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
CREATE TABLE precheck_taxes (
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
CREATE TABLE payments (
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
CREATE INDEX payments_precheck_id_created_at ON payments(precheck_id, created_at);
CREATE INDEX payments_provider_transaction_id ON payments(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
CREATE INDEX payments_fingerprint_hash ON payments(fingerprint_hash) WHERE fingerprint_hash IS NOT NULL;
CREATE TABLE payment_attempts (
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
CREATE INDEX payment_attempts_payment_id_attempt_no ON payment_attempts(payment_id, attempt_no);
CREATE INDEX payment_attempts_provider_transaction_id ON payment_attempts(provider_name, provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
CREATE TABLE financial_operations (
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
CREATE INDEX financial_operations_check_type_created_at ON financial_operations(check_id, operation_type, created_at);
CREATE INDEX financial_operations_shift_created_at ON financial_operations(shift_id, created_at);
CREATE TABLE financial_operation_items (
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
CREATE INDEX financial_operation_items_operation_id ON financial_operation_items(operation_id);
CREATE INDEX financial_operation_items_payment_id ON financial_operation_items(payment_id) WHERE payment_id IS NOT NULL;
CREATE INDEX financial_operation_items_order_line_id ON financial_operation_items(order_line_id) WHERE order_line_id IS NOT NULL;
CREATE TRIGGER financial_operations_no_update
BEFORE UPDATE ON financial_operations
BEGIN
  SELECT RAISE(ABORT, 'financial_operations are append-only');
END;
CREATE TRIGGER financial_operations_no_delete
BEFORE DELETE ON financial_operations
BEGIN
  SELECT RAISE(ABORT, 'financial_operations are append-only');
END;
CREATE TRIGGER financial_operation_items_no_update
BEFORE UPDATE ON financial_operation_items
BEGIN
  SELECT RAISE(ABORT, 'financial_operation_items are append-only');
END;
CREATE TRIGGER financial_operation_items_no_delete
BEFORE DELETE ON financial_operation_items
BEGIN
  SELECT RAISE(ABORT, 'financial_operation_items are append-only');
END;
CREATE TABLE cash_sessions (
  id TEXT PRIMARY KEY,
  edge_cash_session_id TEXT NOT NULL UNIQUE,
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  sales_point_id TEXT NOT NULL REFERENCES sales_points(id),
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
CREATE UNIQUE INDEX cash_sessions_one_open_per_device ON cash_sessions(device_id) WHERE status = 'open';
CREATE INDEX cash_sessions_shift_id ON cash_sessions(shift_id);
CREATE TABLE cash_drawer_events (
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
CREATE INDEX cash_drawer_events_cash_session_created_at ON cash_drawer_events(cash_session_id, created_at);
CREATE INDEX cash_drawer_events_shift_created_at ON cash_drawer_events(shift_id, created_at);
CREATE TABLE ticket_units (
  id TEXT PRIMARY KEY,
  ticket_number TEXT NOT NULL UNIQUE CHECK (ticket_number <> ''),
  restaurant_id TEXT NOT NULL REFERENCES restaurants(id),
  device_id TEXT NOT NULL REFERENCES devices(id),
  cash_session_id TEXT NOT NULL REFERENCES cash_sessions(id),
  shift_id TEXT NOT NULL REFERENCES shifts(id),
  check_id TEXT NOT NULL REFERENCES checks(id),
  order_id TEXT NOT NULL REFERENCES orders(id),
  order_line_id TEXT NOT NULL UNIQUE REFERENCES order_lines(id),
  catalog_item_id TEXT NOT NULL REFERENCES catalog_items(id),
  menu_item_id TEXT NOT NULL REFERENCES menu_items(id),
  name TEXT NOT NULL CHECK (name <> ''),
  sale_date_local TEXT NOT NULL CHECK (sale_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  timezone TEXT NOT NULL CHECK (timezone <> ''),
  validity_mode TEXT NOT NULL CHECK (validity_mode IN ('cash_session', 'business_date', 'absolute_date')),
  validity_date_local TEXT CHECK (validity_date_local IS NULL OR validity_date_local GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  cash_shift_sequence INTEGER NOT NULL CHECK (cash_shift_sequence > 0),
  qr_payload TEXT NOT NULL CHECK (qr_payload <> ''),
  print_status TEXT NOT NULL CHECK (print_status IN ('pending', 'printed')),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','voided')),
  snapshot TEXT NOT NULL CHECK (json_valid(snapshot)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (cash_session_id, cash_shift_sequence)
);
CREATE INDEX ticket_units_check_id ON ticket_units(check_id, cash_shift_sequence);
CREATE INDEX ticket_units_cash_session_sequence ON ticket_units(cash_session_id, cash_shift_sequence);
CREATE INDEX ticket_units_restaurant_sale_date ON ticket_units(restaurant_id, sale_date_local, created_at);
CREATE TABLE manager_override_audit (
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
  action TEXT NOT NULL CHECK (action IN ('cancel_precheck', 'cancel_unconfirmed_order')),
  reason TEXT NOT NULL CHECK (reason <> ''),
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  CHECK (device_id = node_device_id)
);
CREATE INDEX manager_override_audit_precheck_created_at ON manager_override_audit(precheck_id, created_at);
CREATE INDEX manager_override_audit_manager_created_at ON manager_override_audit(manager_employee_id, created_at);
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
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
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
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  UNIQUE(recipe_version_id, catalog_item_id)
);
CREATE TABLE stop_lists (
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
CREATE UNIQUE INDEX stop_lists_restaurant_item ON stop_lists(restaurant_id, catalog_item_id);
CREATE TABLE warehouse_reference (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  name TEXT NOT NULL CHECK (name <> ''),
  kind TEXT NOT NULL CHECK (kind <> ''),
  is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0,1)),
  active INTEGER NOT NULL CHECK (active IN (0,1)),
  cloud_version INTEGER,
  cloud_updated_at TEXT,
  cloud_deleted_at TEXT,
  last_synced_at TEXT,
  updated_at TEXT NOT NULL
);
CREATE INDEX warehouse_reference_restaurant_active ON warehouse_reference(restaurant_id, active, id);
CREATE UNIQUE INDEX warehouse_reference_one_default ON warehouse_reference(restaurant_id) WHERE is_default = 1 AND active = 1 AND cloud_deleted_at IS NULL;
CREATE TABLE local_event_log (
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
CREATE INDEX local_event_log_created_at ON local_event_log(created_at);
CREATE INDEX local_event_log_event_type_created_at ON local_event_log(event_type, created_at);
CREATE INDEX local_event_log_command_id_created_at ON local_event_log(command_id, created_at);
CREATE TABLE pos_sync_outbox (
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
CREATE INDEX pos_sync_outbox_status_sequence_no ON pos_sync_outbox(status, sequence_no);
CREATE INDEX pos_sync_outbox_pending_retry_sequence ON pos_sync_outbox(next_retry_at, sequence_no) WHERE status = 'pending';
CREATE INDEX pos_sync_outbox_processing_locked_at ON pos_sync_outbox(locked_at) WHERE status = 'processing';
CREATE INDEX pos_sync_outbox_command_id_created_at ON pos_sync_outbox(command_id, created_at);
CREATE INDEX checks_business_date_closed_at ON checks(business_date_local, closed_at, id);
CREATE INDEX checks_order_id_closed_at ON checks(order_id, closed_at);
CREATE INDEX orders_closed_restaurant_created_at ON orders(restaurant_id, status, created_at, id);
CREATE INDEX payments_business_date_shift_created_at ON payments(business_date_local, shift_id, created_at, id);
CREATE INDEX financial_operations_restaurant_business_date_type_created_at ON financial_operations(restaurant_id, business_date_local, operation_type, created_at, id);
CREATE INDEX financial_operations_original_shift_created_at ON financial_operations(original_shift_id, created_at, id);
CREATE INDEX financial_operations_check_created_at ON financial_operations(check_id, created_at, id);
CREATE INDEX local_event_log_occurred_at ON local_event_log(occurred_at, id);
CREATE INDEX pos_sync_outbox_created_at ON pos_sync_outbox(created_at, id);
CREATE TABLE "cloud_master_sync_state" (
  id TEXT PRIMARY KEY,
  restaurant_id TEXT CHECK (restaurant_id IS NULL OR restaurant_id <> ''),
  node_device_id TEXT NOT NULL CHECK (node_device_id <> ''),
  stream_name TEXT NOT NULL CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','recipes','inventory_reference','proposal_feedback','receipt_templates','printers','sales_points','restaurant_sections')),
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
CREATE INDEX cloud_master_sync_state_node_status ON cloud_master_sync_state(node_device_id, status);
CREATE TABLE "catalog_items" (
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
, qr_confirmation_enabled INTEGER NOT NULL DEFAULT 0, single_unit_per_line INTEGER NOT NULL DEFAULT 0, validity_mode TEXT NOT NULL DEFAULT '', validity_expires_at TEXT);
CREATE TRIGGER recipe_versions_owner_catalog_item_insert
BEFORE INSERT ON recipe_versions
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.dish_catalog_item_id AND type IN ('dish', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe version must reference dish or semi_finished catalog item');
END;
CREATE TRIGGER recipe_versions_owner_catalog_item_update
BEFORE UPDATE OF dish_catalog_item_id ON recipe_versions
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.dish_catalog_item_id AND type IN ('dish', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe version must reference dish or semi_finished catalog item');
END;
CREATE TRIGGER recipe_lines_good_or_semi_finished_insert
BEFORE INSERT ON recipe_lines
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.catalog_item_id AND type IN ('good', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe line must reference good or semi_finished catalog item');
END;
CREATE TRIGGER recipe_lines_good_or_semi_finished_update
BEFORE UPDATE OF catalog_item_id ON recipe_lines
FOR EACH ROW
WHEN NOT EXISTS (SELECT 1 FROM catalog_items WHERE id = NEW.catalog_item_id AND type IN ('good', 'semi_finished'))
BEGIN
  SELECT RAISE(ABORT, 'recipe line must reference good or semi_finished catalog item');
END;
CREATE TABLE receipt_templates (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT,
  document_type TEXT NOT NULL,
  name TEXT NOT NULL,
  content TEXT NOT NULL,
  level INTEGER NOT NULL DEFAULT 1,
  cpl INTEGER NOT NULL,
  printer_class TEXT NOT NULL DEFAULT 'generic',
  is_default INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 1,
  synced_at TEXT NOT NULL
);
CREATE INDEX receipt_templates_type_default
  ON receipt_templates (document_type, is_default);
CREATE TABLE receipt_printers (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('tcp','usb')),
  address TEXT,
  port INTEGER,
  document_types TEXT NOT NULL CHECK (json_valid(document_types)),
  codepage TEXT NOT NULL DEFAULT '' CHECK (codepage IN ('','cp437','cp866')),
  paper_cut_type TEXT NOT NULL DEFAULT 'partial' CHECK (paper_cut_type IN ('partial','full')),
  cpl INTEGER NOT NULL CHECK (cpl IN (32,42,48,56,80)),
  is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  synced_at TEXT NOT NULL,
  CHECK ((type = 'tcp' AND address IS NOT NULL AND address <> '' AND port IS NOT NULL AND port > 0) OR type = 'usb')
);
CREATE INDEX receipt_printers_restaurant_active
  ON receipt_printers (restaurant_id, is_active, id);
CREATE TABLE sales_points (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  name TEXT NOT NULL CHECK (name <> ''),
  analytics_tag TEXT NOT NULL CHECK (analytics_tag <> ''),
  default_table_id TEXT NOT NULL REFERENCES tables(id),
  is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  synced_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(restaurant_id, analytics_tag)
);
CREATE INDEX sales_points_restaurant_active
  ON sales_points (restaurant_id, is_active, id);
CREATE TRIGGER sales_points_default_table_insert
BEFORE INSERT ON sales_points
FOR EACH ROW
WHEN NOT EXISTS (
  SELECT 1 FROM tables t
  WHERE t.id = NEW.default_table_id
    AND t.restaurant_id = NEW.restaurant_id
    AND t.active = 1
)
BEGIN
  SELECT RAISE(ABORT, 'sales point must reference active table from same restaurant');
END;
CREATE TRIGGER sales_points_default_table_update
BEFORE UPDATE OF restaurant_id, default_table_id ON sales_points
FOR EACH ROW
WHEN NOT EXISTS (
  SELECT 1 FROM tables t
  WHERE t.id = NEW.default_table_id
    AND t.restaurant_id = NEW.restaurant_id
    AND t.active = 1
)
BEGIN
  SELECT RAISE(ABORT, 'sales point must reference active table from same restaurant');
END;
CREATE TABLE restaurant_sections (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  name TEXT NOT NULL CHECK (name <> ''),
  mode TEXT NOT NULL CHECK (mode IN ('hall_section','kitchen_workshop')),
  hall_id TEXT REFERENCES halls(id),
  kitchen_routing_key TEXT,
  warehouse_id TEXT,
  is_default INTEGER NOT NULL DEFAULT 0 CHECK (is_default IN (0,1)),
  is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  synced_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK ((mode = 'hall_section' AND kitchen_routing_key IS NULL) OR (mode = 'kitchen_workshop' AND hall_id IS NULL))
);
CREATE INDEX restaurant_sections_restaurant_mode_active
  ON restaurant_sections (restaurant_id, mode, is_active, id);
CREATE UNIQUE INDEX restaurant_sections_one_default_hall_section
  ON restaurant_sections (restaurant_id)
  WHERE mode = 'hall_section' AND is_default = 1;
CREATE TABLE print_routes (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  document_type TEXT NOT NULL CHECK (document_type IN ('precheck','check_nonfiscal','ticket','kitchen_service','report')),
  scope_type TEXT NOT NULL CHECK (scope_type IN ('restaurant','sales_point','section')),
  scope_id TEXT,
  printer_id TEXT NOT NULL REFERENCES receipt_printers(id),
  is_required INTEGER NOT NULL DEFAULT 1 CHECK (is_required IN (0,1)),
  sort_order INTEGER NOT NULL DEFAULT 0,
  origin TEXT NOT NULL DEFAULT 'cloud' CHECK (origin IN ('cloud','edge_override')),
  is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  synced_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK ((scope_type = 'restaurant' AND scope_id IS NULL) OR (scope_type IN ('sales_point','section') AND scope_id IS NOT NULL))
);
CREATE INDEX print_routes_scope_active
  ON print_routes (restaurant_id, scope_type, scope_id, document_type, is_active, sort_order);
CREATE INDEX print_routes_printer_active
  ON print_routes (printer_id, is_active);
CREATE UNIQUE INDEX print_routes_unique_active_printer_scope
  ON print_routes (restaurant_id, document_type, scope_type, COALESCE(scope_id, ''), printer_id)
  WHERE is_active = 1;
CREATE TRIGGER print_routes_required_scope_insert
BEFORE INSERT ON print_routes
FOR EACH ROW
WHEN NOT (
  (NEW.document_type = 'check_nonfiscal' AND NEW.scope_type = 'sales_point')
  OR (NEW.document_type IN ('precheck','ticket','kitchen_service') AND NEW.scope_type = 'section')
  OR (NEW.document_type = 'report' AND NEW.scope_type = 'restaurant')
)
BEGIN
  SELECT RAISE(ABORT, 'print route document_type requires a fixed scope_type');
END;
CREATE TRIGGER print_routes_required_scope_update
BEFORE UPDATE OF document_type, scope_type ON print_routes
FOR EACH ROW
WHEN NOT (
  (NEW.document_type = 'check_nonfiscal' AND NEW.scope_type = 'sales_point')
  OR (NEW.document_type IN ('precheck','ticket','kitchen_service') AND NEW.scope_type = 'section')
  OR (NEW.document_type = 'report' AND NEW.scope_type = 'restaurant')
)
BEGIN
  SELECT RAISE(ABORT, 'print route document_type requires a fixed scope_type');
END;
CREATE TRIGGER print_routes_required_section_mode_insert
BEFORE INSERT ON print_routes
FOR EACH ROW
WHEN NEW.scope_type = 'section' AND NOT EXISTS (
  SELECT 1 FROM restaurant_sections s
  WHERE s.id = NEW.scope_id
    AND s.restaurant_id = NEW.restaurant_id
    AND (
      (NEW.document_type IN ('precheck','ticket') AND s.mode = 'hall_section')
      OR (NEW.document_type = 'kitchen_service' AND s.mode = 'kitchen_workshop')
    )
)
BEGIN
  SELECT RAISE(ABORT, 'print route document_type requires matching section mode');
END;
CREATE TRIGGER print_routes_required_section_mode_update
BEFORE UPDATE OF document_type, scope_type, scope_id, restaurant_id ON print_routes
FOR EACH ROW
WHEN NEW.scope_type = 'section' AND NOT EXISTS (
  SELECT 1 FROM restaurant_sections s
  WHERE s.id = NEW.scope_id
    AND s.restaurant_id = NEW.restaurant_id
    AND (
      (NEW.document_type IN ('precheck','ticket') AND s.mode = 'hall_section')
      OR (NEW.document_type = 'kitchen_service' AND s.mode = 'kitchen_workshop')
    )
)
BEGIN
  SELECT RAISE(ABORT, 'print route document_type requires matching section mode');
END;
CREATE TABLE printer_route_override_audit (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  actor_employee_id TEXT,
  action TEXT NOT NULL CHECK (action IN ('create','update','delete')),
  route_id TEXT,
  scope_type TEXT NOT NULL CHECK (scope_type IN ('restaurant','sales_point','section')),
  scope_id TEXT,
  document_type TEXT NOT NULL CHECK (document_type IN ('precheck','check_nonfiscal','ticket','kitchen_service','report')),
  before_json TEXT CHECK (before_json IS NULL OR json_valid(before_json)),
  after_json TEXT CHECK (after_json IS NULL OR json_valid(after_json)),
  outbox_command_id TEXT,
  occurred_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  CHECK ((action = 'create' AND after_json IS NOT NULL) OR (action = 'delete' AND before_json IS NOT NULL) OR (action = 'update' AND before_json IS NOT NULL AND after_json IS NOT NULL))
);
CREATE INDEX printer_route_override_audit_restaurant_created
  ON printer_route_override_audit (restaurant_id, created_at);
CREATE INDEX printer_route_override_audit_outbox_command
  ON printer_route_override_audit (outbox_command_id)
  WHERE outbox_command_id IS NOT NULL;
CREATE TABLE print_jobs (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  document_type TEXT NOT NULL CHECK (document_type IN ('precheck','check_nonfiscal','ticket')),
  scope_id TEXT,
  source_kind TEXT NOT NULL CHECK (source_kind IN ('precheck','check','ticket')),
  source_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','processing','succeeded','failed')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  max_attempts INTEGER NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
  printer_class TEXT NOT NULL DEFAULT 'generic',
  last_error TEXT,
  next_attempt_at TEXT,
  locked_by TEXT,
  locked_at TEXT,
  printed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(document_type, source_id)
);
CREATE INDEX print_jobs_pending_due
  ON print_jobs (status, next_attempt_at, created_at);
CREATE INDEX print_jobs_restaurant_status_created
  ON print_jobs (restaurant_id, status, created_at);
CREATE TABLE print_job_targets (
  id TEXT NOT NULL PRIMARY KEY,
  print_job_id TEXT NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,
  restaurant_id TEXT NOT NULL,
  printer_id TEXT NOT NULL REFERENCES receipt_printers(id),
  scope_type TEXT NOT NULL CHECK (scope_type IN ('restaurant','sales_point','section')),
  scope_id TEXT,
  status TEXT NOT NULL CHECK (status IN ('pending','processing','succeeded','failed')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  max_attempts INTEGER NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
  is_required INTEGER NOT NULL DEFAULT 1 CHECK (is_required IN (0,1)),
  last_error TEXT,
  next_attempt_at TEXT,
  locked_by TEXT,
  locked_at TEXT,
  printed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(print_job_id, printer_id, scope_type, scope_id),
  CHECK ((scope_type = 'restaurant' AND scope_id IS NULL) OR (scope_type IN ('sales_point','section') AND scope_id IS NOT NULL))
);
CREATE INDEX print_job_targets_pending_due
  ON print_job_targets (status, next_attempt_at, created_at);
CREATE INDEX print_job_targets_job_status
  ON print_job_targets (print_job_id, status);
CREATE INDEX print_job_targets_restaurant_status_created
  ON print_job_targets (restaurant_id, status, created_at);
CREATE UNIQUE INDEX print_job_targets_unique_printer_scope
  ON print_job_targets (print_job_id, printer_id, scope_type, COALESCE(scope_id, ''));
