ALTER TABLE cloud_edge_event_receipts
  DROP CONSTRAINT IF EXISTS cloud_edge_event_receipts_event_type_check;

ALTER TABLE cloud_edge_event_receipts
  ADD CONSTRAINT cloud_edge_event_receipts_event_type_check
  CHECK (event_type IN (
    'ShiftOpened',
    'ShiftClosed',
    'OrderCreated',
    'OrderLineAdded',
    'CheckCreated',
    'PaymentCaptured',
    'OrderClosed',
    'CashSessionOpened',
    'CashSessionClosed',
    'CashDrawerEventRecorded'
  ));
