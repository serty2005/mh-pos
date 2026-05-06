CREATE TABLE prechecks_next (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL REFERENCES orders(id),
  status TEXT NOT NULL CHECK (status IN ('issued', 'closed', 'cancelled', 'superseded')),
  version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
  supersedes_precheck_id TEXT CHECK (supersedes_precheck_id IS NULL OR supersedes_precheck_id <> ''),
  subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
  discount_total INTEGER NOT NULL CHECK (discount_total >= 0),
  tax_total INTEGER NOT NULL CHECK (tax_total >= 0),
  total INTEGER NOT NULL CHECK (total >= 0),
  paid_total INTEGER NOT NULL DEFAULT 0 CHECK (paid_total >= 0),
  created_at TEXT NOT NULL,
  issued_at TEXT NOT NULL,
  closed_at TEXT,
  cancelled_by_employee_id TEXT REFERENCES employees(id),
  cancellation_reason TEXT CHECK (cancellation_reason IS NULL OR cancellation_reason <> ''),
  CHECK (total = subtotal - discount_total + tax_total),
  CHECK (paid_total <= total),
  CHECK (closed_at IS NULL OR status IN ('closed', 'cancelled', 'superseded')),
  CHECK (closed_at IS NOT NULL OR status = 'issued')
);

INSERT INTO prechecks_next (
  id,
  order_id,
  status,
  version,
  subtotal,
  discount_total,
  tax_total,
  total,
  paid_total,
  created_at,
  issued_at,
  closed_at
)
SELECT
  id,
  order_id,
  status,
  1,
  subtotal,
  discount_total,
  tax_total,
  total,
  0,
  created_at,
  issued_at,
  closed_at
FROM prechecks;

DROP TABLE prechecks;
ALTER TABLE prechecks_next RENAME TO prechecks;

CREATE UNIQUE INDEX prechecks_one_issued_per_order ON prechecks(order_id) WHERE status = 'issued';
CREATE UNIQUE INDEX prechecks_order_version ON prechecks(order_id, version);
CREATE INDEX prechecks_order_id_created_at ON prechecks(order_id, created_at);
