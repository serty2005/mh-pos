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
| Catalog folder/tag | Cloud | No | Cloud -> Edge via `catalog` | реализовано сейчас |
| Modifier group/option | Cloud | No | Cloud -> Edge via `menu` | реализовано сейчас |
| Recipe/reference inventory | Cloud | No | Cloud -> Edge planned | реализована только основа in schema/constants; POS Edge apply path not implemented |
| Employee shift | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Cash session/drawer event | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Order/order line | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Precheck | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Payment | Edge | Yes | Edge -> Cloud operational events for capture | реализовано сейчас |
| Financial operation | Edge | Yes | `CancellationRecorded` and `RefundRecorded` are current Edge -> Cloud operational events | реализовано сейчас |
| Check | Edge | Generated after full payment; finalized checks are not rewritten by cancellation/refund | `CheckCreated` is current Edge -> Cloud operational event; `CheckRefunded` is legacy accepted | реализовано сейчас |
| Tax/pricing policy reference | Cloud | Edge read model only | `pricing_policy` Cloud -> Edge stream for `tax_profiles`, `tax_rules`, `service_charge_rules`, `pricing_policies` | реализовано сейчас |
| Operational order adjustments | Edge | Yes while order is open | runtime-команды; будущие policy ids могут ограничивать допустимые варианты | реализовано сейчас |
| Stock document/move | Inventory context | Not from cashier runtime | planned | реализована только основа |

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
`catalog` и `menu` payloads включают catalog folders/tags/services и modifier groups/options/links; menu categories остаются отдельным понятием и не заменяют catalog folders.
`pricing_policy` включает tax/service-charge reference tables и automatic discount/surcharge policies; manual override runtime остается backend RBAC-controlled action.

Реализовано сейчас:

- POS Edge sync sender использует authenticated `POST /api/v1/sync/exchange` как приоритетный Cloud-Edge цикл, когда локальное provisioning state содержит `node_token`.
- Edge отправляет текущие `cloud_master_sync_state` revisions/checkpoints по поддерживаемым streams и получает только более новые Cloud packages.
- Cloud package apply и commit соответствующего stream checkpoint выполняются существующей transaction boundary `mastersync.Service`.
- Если локальный apply не проходит, Edge не помечает accepted outbox rows как `sent`; retry повторяет exchange, а Cloud idempotency возвращает стабильный ACK для уже принятого event.

## Current Edge -> Cloud Runtime

Реализовано сейчас:

- cashier commands пишут local event/outbox foundation;
- POS runtime продолжает работу, если Cloud недоступен;
- Cloud receiver/projection foundation существует.
- authenticated `sync/exchange` принимает Edge events с item-level ACK и сохраняет существующую idempotency model;
- legacy `/sync/edge-events` и `/sync/edge-events/batch` остаются совместимыми inbound routes.

Реализовано сейчас:

- `CancellationRecorded` и `RefundRecorded` принимаются Cloud receiver и сохраняются как operational events;
- `PaymentRefunded` и `CheckRefunded` остаются legacy accepted event types для старых payloads;
- Cloud event-type stats обновляются для всех accepted operational events;
- Cloud shift finance foundation считает coarse refund totals from current `RefundRecorded` and legacy `PaymentRefunded`/`CheckRefunded`, but it is not a full projection for financial operation ledger item scopes, inventory disposition, approval policy or original-shift reconciliation.

Запланировано далее:

- inventory event contracts только после реализации runtime.
- richer modifier/pricing reporting projections после pilot acceptance.

## Master Data Rule

Cloud владеет authoring master data. POS Edge использует local read model и sync ingest. Edge runtime mutation APIs для Cloud-owned master data не входят в поддерживаемый pilot runtime.

## Pricing Policy Boundary

Реализовано сейчас:

- Edge operational adjustments являются cashier runtime commands для line/order discounts и surcharges на открытом order.
- Cloud-authored tax/pricing policy reference data, including automatic discount/surcharge policies, применяется через `pricing_policy` и хранится с `cloud_version`, `cloud_updated_at`, `cloud_deleted_at` и `last_synced_at`.
- Manual surcharge/discount commands остаются runtime actions и требуют существующих pricing permissions.

Запланировано далее:

- Runtime adjustments должны ссылаться на synced policy ids там, где существует central policy.
- Manual override flows для policy exceptions должны иметь отдельный permission boundary и audit trail до того, как они станут supported pilot behavior.
