# Directional Sync Ownership

Статус: актуально для frozen cashier pilot.

## Ownership Matrix

| Aggregate/Data | Owner | Edge mutation | Direction | Статус |
| --- | --- | --- | --- | --- |
| Restaurant | Cloud | No | Cloud -> Edge | реализовано сейчас for ingest stream `restaurants` |
| Device | Cloud | Pairing/provisioning only | Cloud -> Edge | реализовано сейчас for ingest stream `devices` |
| Role/Employee | Cloud | No | Cloud -> Edge | реализовано сейчас for ingest stream `staff` |
| Hall/Table | Cloud | No | Cloud -> Edge | реализовано сейчас for ingest stream `floor` |
| Catalog item | Cloud | No | Cloud -> Edge | реализовано сейчас for ingest stream `catalog` |
| Menu item | Cloud | No | Cloud -> Edge | реализовано сейчас for ingest stream `menu` |
| Modifier group/option | Cloud | No | Cloud -> Edge planned | foundation only in Cloud schema |
| Recipe/reference inventory | Cloud | No | Cloud -> Edge planned | foundation only in schema/constants; POS Edge apply path not implemented |
| Employee shift | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Cash session/drawer event | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Order/order line | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Precheck | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Payment | Edge | Yes | Edge -> Cloud operational events for capture | реализовано сейчас |
| Payment refund | Edge | Yes | sync/reporting policy needs final hardening | backend/UI реализовано сейчас |
| Check | Edge | Generated after full payment | Edge -> Cloud operational events | реализовано сейчас |
| Stock document/move | Inventory context | Not from cashier runtime | planned | foundation only |

## Current Cloud -> Edge Ingest

`mastersync.Service` currently supports only:

- `restaurants`
- `devices`
- `staff`
- `floor`
- `catalog`
- `menu`

`recipes` and `inventory_reference` must not be documented as supported POS Edge ingest streams until `mastersync.Service` applies their payloads.

## Current Edge -> Cloud Runtime

Реализовано сейчас:

- cashier commands write local event/outbox foundation;
- POS runtime can continue while Cloud is unavailable;
- Cloud receiver/projection foundation exists.

Needs final hardening:

- explicit Cloud reporting treatment for refund events;
- modifiers/pricing/inventory event contracts only after runtime is implemented.

## Master Data Rule

Cloud owns authoring of master data. POS Edge uses local read model and sync ingest. Edge runtime mutation APIs for Cloud-owned master data are not part of supported pilot runtime.
