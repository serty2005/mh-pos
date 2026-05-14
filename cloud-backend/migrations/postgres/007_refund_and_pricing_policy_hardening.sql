ALTER TABLE cloud_edge_event_receipts
  DROP CONSTRAINT IF EXISTS cloud_edge_event_receipts_event_type_check;

ALTER TABLE cloud_edge_event_receipts
  ADD CONSTRAINT cloud_edge_event_receipts_event_type_check CHECK (event_type IN (
    'ShiftOpened',
    'ShiftClosed',
    'OrderCreated',
    'OrderLineAdded',
    'OrderLineQuantityChanged',
    'OrderLineVoided',
    'PrecheckIssued',
    'PrecheckReprinted',
    'PrecheckCancelled',
    'CheckCreated',
    'CheckRefunded',
    'CheckReprinted',
    'PaymentCaptured',
    'PaymentRefunded',
    'OrderClosed',
    'CashSessionOpened',
    'CashSessionClosed',
    'CashDrawerEventRecorded',
    'AuthSessionStarted',
    'AuthSessionRevoked',
    'DeviceRegistered'
  ));

ALTER TABLE cloud_projection_shift_finance
  ADD COLUMN IF NOT EXISTS payments_refunded_count BIGINT NOT NULL DEFAULT 0 CHECK (payments_refunded_count >= 0),
  ADD COLUMN IF NOT EXISTS payments_refunded_total BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS checks_refunded_count BIGINT NOT NULL DEFAULT 0 CHECK (checks_refunded_count >= 0),
  ADD COLUMN IF NOT EXISTS checks_refunded_total BIGINT NOT NULL DEFAULT 0;

ALTER TABLE cloud_master_data_packages
  DROP CONSTRAINT IF EXISTS cloud_master_data_packages_stream_name_check;

ALTER TABLE cloud_master_data_packages
  ADD CONSTRAINT cloud_master_data_packages_stream_name_check CHECK (stream_name IN ('restaurants','devices','staff','floor','catalog','menu','pricing_policy','currencies'));
