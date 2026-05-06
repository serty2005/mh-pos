# Edge to Cloud Sync Contracts v1

This document describes the minimal implemented contract between the existing POS Edge outbox and the new Cloud Sync Receiver.

## Endpoint

```text
POST /api/v1/sync/edge-events
Content-Type: application/json
```

The request body is one `SyncEnvelope`. The current receiver accepts the events already emitted by `pos-backend`:

```text
ShiftOpened
ShiftClosed
OrderCreated
OrderLineAdded
CheckCreated
PaymentCaptured
OrderClosed
CashSessionOpened
CashSessionClosed
CashDrawerEventRecorded
```

## SyncEnvelope

```json
{
  "version": "1",
  "event_id": "edge-generated-event-id",
  "command_id": "edge-command-id",
  "event_type": "OrderCreated",
  "aggregate_type": "Order",
  "aggregate_id": "order-id",
  "restaurant_id": "restaurant-id",
  "device_id": "device-id",
  "shift_id": "shift-id",
  "occurred_at": "2026-05-05T09:00:00Z",
  "payload": {
    "origin": "edge_device",
    "data": {}
  }
}
```

`payload.data` is the JSON representation of the corresponding edge domain object:

```text
ShiftOpened     -> Shift
ShiftClosed     -> Shift
OrderCreated    -> Order
OrderLineAdded  -> OrderLine
CheckCreated    -> Check
PaymentCaptured -> Payment
OrderClosed     -> Order
CashSessionOpened -> CashSession
CashSessionClosed -> CashSession
CashDrawerEventRecorded -> CashDrawerEvent
```

Cloud validates the envelope version, known event type, required routing fields, and the basic payload shape for the selected event type. Cloud stores the raw full envelope before any future projection logic.

## Idempotency Rules

Current MVP is instance-per-tenant, so there is no `organization_id` or `tenant_id` in the implemented key yet.

```text
idempotency_key = restaurant_id + ":" + device_id + ":" + edge_event_id
edge_event_id = SyncEnvelope.event_id
```

Relationships:

```text
command_id
  generated once per edge write use case
  may be shared by multiple local events from the same write use case
  stored in local_event_log rows
  stored in pos_sync_outbox rows
  copied into each SyncEnvelope.command_id

event_id
  generated once for the local edge event
  stored in local_event_log
  copied into SyncEnvelope.event_id

edge_event_id
  Cloud-side name for SyncEnvelope.event_id
  used in the Cloud idempotency key
```

Replay behavior:

```text
same idempotency_key + same raw envelope hash -> return original ack
same idempotency_key + different raw envelope hash -> reject as conflict
```

## Ack

Cloud returns HTTP `202 Accepted` for both the first successful receive and safe duplicate replay.

```json
{
  "status": "accepted",
  "idempotency_key": "restaurant-id:device-id:edge-event-id",
  "cloud_receipt_id": "cloud-generated-receipt-id",
  "command_id": "edge-command-id",
  "event_id": "edge-generated-event-id",
  "edge_event_id": "edge-generated-event-id",
  "envelope_version": "1",
  "cloud_received_at": "2026-05-05T10:00:00Z",
  "raw_payload_sha256_hex": "..."
}
```

The ack is stable on replay: duplicate POST of the same envelope returns the same `cloud_receipt_id`, timestamps, ids, and payload hash.
