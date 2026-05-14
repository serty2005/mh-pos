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
pricing_policy
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
  "menu_items": [],
  "tax_profiles": [],
  "tax_rules": [],
  "service_charge_rules": []
}
```

Правила:

- `sync_mode` по умолчанию равен `incremental`.
- Поддерживаемые значения: `incremental` и `full_snapshot`.
- `full_snapshot` требует `full_snapshot_reason` со значением `terminal_restaurant_changed` или `node_role_changed`.
- Unsupported streams отклоняются.
- `pricing_policy` применяет Cloud-authored `tax_profiles`, `tax_rules` и `service_charge_rules` в Edge read-model tables с sync metadata.
- Unsupported JSON fields отклоняются strict decode; неизвестные stream names не применяются.
- `recipes`, `inventory_reference` и modifiers пока не являются поддерживаемыми POS Edge apply payloads.

Только foundation:

- Cloud schema содержит modifier/recipe/menu publication foundation.
- SQLite schema содержит recipe/inventory foundation.
- Эти foundation нельзя документировать как поддерживаемый POS Edge runtime ingest, пока `mastersync.Service` не применяет их payloads.

## Edge -> Cloud Operational Events

Реализовано сейчас:

- POS Edge пишет local operational events и outbox rows для cashier runtime commands.
- Детали sender/cloud receiver являются implementation-specific; документация не должна обещать Cloud reporting semantics для events, которых нет в подтвержденном Edge -> Cloud catalog.

Подтвержденный текущий Edge -> Cloud catalog в domain boundary:

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
PaymentRefunded
CheckCreated
CheckRefunded
CheckReprinted
OrderClosed
AuthSessionStarted
AuthSessionRevoked
DeviceRegistered
```

Refund sync behavior:

- `PaymentRefunded` и `CheckRefunded` являются подтвержденными Edge -> Cloud operational events.
- Cloud receiver валидирует эти event types, сохраняет raw envelope/journal rows и обновляет event-type stats.
- Shift finance projection хранит payment refund count/total и check refunded count/current refunded paid-total foundation. Подробное отображение возвратов должно читать stored raw/journal payloads, пока не добавлена более богатая refund ledger projection.

## Financial Payload Boundaries

Реализовано сейчас:

- Payloads `PaymentCaptured`, `PaymentRefunded`, `CheckCreated` и `CheckRefunded` включают backend-owned `business_date_local`, если он есть у source aggregate.
- Precheck/check reprint использует immutable snapshot payload.
- Payment ссылается на `precheck_id`, а не на legacy `check_id`.

Не реализовано сейчас:

- modifier selections в operational snapshots;
- inventory consumption events;
- PSP/fiscal event streams.

## Planned Boundaries

Запланировано до пилота только при отдельном принятии:

- Cloud-authored pricing/tax UI и полный publication workflow поверх generic `pricing_policy` package storage/apply;
- modifier publication и order snapshot support;

После пилота:

- KDS/Production events;
- inventory stock movement events;
- PSP/fiscal integration events;
- richer reporting projections and optional ClickHouse acceleration.
