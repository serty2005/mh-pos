# Edge / Cloud Sync Contracts v1

Статус: актуализировано под frozen cashier pilot.

## Direction Model

Реализовано сейчас:

- POS Edge owns cashier operational runtime data.
- Cloud owns master/reference/configuration authoring.
- POS Edge can continue cashier runtime while Cloud is unavailable.
- Directional ownership matrix is maintained in `docs/sync/directional-sync-ownership.md`.

## Cloud -> Edge Master Data Ingest

POS Edge endpoints:

```text
POST /api/v1/sync/master-data/snapshots
POST /api/v1/sync/master-data/{stream}
```

Supported POS Edge apply streams in `mastersync.Service`:

```text
restaurants
devices
staff
floor
catalog
menu
```

Request body shape currently supported by POS Edge:

```json
{
  "restaurant_id": "restaurant-id",
  "stream": "catalog",
  "sync_mode": "incremental",
  "full_snapshot_reason": "",
  "checkpoint_token": "optional-cloud-checkpoint",
  "cloud_version": 42,
  "cloud_updated_at": "2026-05-07T10:00:00Z",
  "restaurants": [],
  "devices": [],
  "roles": [],
  "employees": [],
  "halls": [],
  "tables": [],
  "catalog_items": [],
  "menu_items": []
}
```

Rules:

- `sync_mode` is `incremental` by default.
- Supported values are `incremental` and `full_snapshot`.
- `full_snapshot` requires `full_snapshot_reason` of `terminal_restaurant_changed` or `node_role_changed`.
- Unsupported streams are rejected.
- `recipes`, `inventory_reference`, modifiers, pricing rules and tax profiles are not supported POS Edge apply payloads yet.

Foundation only:

- Cloud schema has modifier/recipe/menu publication foundations.
- SQLite schema has recipe/inventory foundation.
- These foundations must not be documented as supported POS Edge runtime ingest until `mastersync.Service` applies them.

## Edge -> Cloud Operational Events

Реализовано сейчас:

- POS Edge writes local operational events and outbox rows for cashier runtime commands.
- Sender/cloud receiver details are implementation-specific; documentation must not claim Cloud reporting semantics for events that are not in the confirmed edge-to-cloud catalog.

Confirmed current edge-to-cloud catalog in domain boundary:

```text
ShiftOpened
ShiftClosed
CashSessionOpened
CashSessionClosed
CashDrawerEventRecorded
OrderCreated
OrderLineAdded
OrderLineQuantityChanged
OrderLineVoided
PrecheckIssued
PrecheckReprinted
PrecheckCancelled
PaymentCaptured
CheckCreated
CheckReprinted
OrderClosed
AuthSessionStarted
AuthSessionRevoked
DeviceRegistered
```

Refund note:

- Backend refund flow is implemented and writes local event/outbox records.
- `PaymentRefunded` / `CheckRefunded` should not be documented as confirmed Cloud reporting events until sync direction and Cloud receiver behavior are explicitly hardened.

## Financial Payload Boundaries

Реализовано сейчас:

- `PaymentCaptured` and `CheckCreated` payloads include backend-owned `business_date_local`.
- Precheck/check reprint uses immutable snapshot payload.
- Payment references `precheck_id`, not legacy `check_id`.

Не реализовано сейчас:

- discount/surcharge/tax policy payloads;
- modifier selections in operational snapshots;
- inventory consumption events;
- PSP/fiscal event streams.

## Planned Boundaries

Запланировано до пилота only if accepted:

- pricing/tax publication payloads after backend policy exists;
- modifier publication and order snapshot support;
- refund sync/reporting hardening.

После пилота:

- KDS/Production events;
- inventory stock movement events;
- PSP/fiscal integration events;
- richer reporting projections and optional ClickHouse acceleration.
