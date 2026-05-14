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
| Payment | Edge | Yes | Edge -> Cloud operational events for capture and refund | реализовано сейчас |
| Payment refund | Edge | Yes | `PaymentRefunded` is confirmed Edge -> Cloud operational event | реализовано сейчас |
| Check | Edge | Generated after full payment/refund status | `CheckCreated` and `CheckRefunded` are confirmed Edge -> Cloud operational events | реализовано сейчас |
| Tax/pricing policy reference | Cloud | Edge read model only | `pricing_policy` Cloud -> Edge stream for `tax_profiles`, `tax_rules`, `service_charge_rules` | реализовано сейчас |
| Operational order adjustments | Edge | Yes while order is open | runtime-команды; будущие policy ids могут ограничивать допустимые варианты | реализовано сейчас |
| Stock document/move | Inventory context | Not from cashier runtime | planned | foundation only |

## Current Cloud -> Edge Ingest

`mastersync.Service` сейчас поддерживает только:

- `restaurants`
- `devices`
- `staff`
- `floor`
- `catalog`
- `menu`
- `pricing_policy`

`recipes` и `inventory_reference` нельзя документировать как поддерживаемые POS Edge ingest streams, пока `mastersync.Service` не применяет их payloads.
`pricing_policy` намеренно ограничен tax/service-charge reference tables; он не включает modifiers runtime или Cloud-authored advanced pricing.

## Current Edge -> Cloud Runtime

Реализовано сейчас:

- cashier commands пишут local event/outbox foundation;
- POS runtime продолжает работу, если Cloud недоступен;
- Cloud receiver/projection foundation существует.

Реализовано сейчас:

- `PaymentRefunded` и `CheckRefunded` принимаются Cloud receiver и сохраняются как operational events;
- Cloud shift finance foundation хранит refund counters/totals из этих events.

Запланировано далее:

- modifiers/pricing/inventory event contracts только после реализации runtime.

## Master Data Rule

Cloud владеет authoring master data. POS Edge использует local read model и sync ingest. Edge runtime mutation APIs для Cloud-owned master data не входят в поддерживаемый pilot runtime.

## Pricing Policy Boundary

Реализовано сейчас:

- Edge operational adjustments являются cashier runtime commands для line/order discounts и surcharges на открытом order.
- Cloud-authored tax/pricing policy reference data применяется через `pricing_policy` и хранится с `cloud_version`, `cloud_updated_at`, `cloud_deleted_at` и `last_synced_at`.
- Manual surcharge/discount commands остаются runtime actions и требуют существующих pricing permissions.

Запланировано далее:

- Runtime adjustments должны ссылаться на synced policy ids там, где существует central policy.
- Manual override flows для policy exceptions должны иметь отдельный permission boundary и audit trail до того, как они станут supported pilot behavior.
