CREATE TABLE prechecks (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  status TEXT NOT NULL CHECK (status IN ('issued', 'closed', 'cancelled')),
  subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
  discount_total INTEGER NOT NULL CHECK (discount_total >= 0),
  tax_total INTEGER NOT NULL CHECK (tax_total >= 0),
  total INTEGER NOT NULL CHECK (total >= 0),
  created_at TEXT NOT NULL,
  issued_at TEXT NOT NULL,
  closed_at TEXT,
  CHECK (total = subtotal - discount_total + tax_total),
  CHECK (closed_at IS NULL OR status IN ('closed', 'cancelled'))
);

CREATE UNIQUE INDEX prechecks_one_issued_per_order ON prechecks(order_id) WHERE status = 'issued';
CREATE INDEX prechecks_order_id_created_at ON prechecks(order_id, created_at);
