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
| Modifier group/option | Cloud | No | Cloud -> Edge via `catalog` | реализовано сейчас |
| Recipe reference | Cloud | Edge read-only | Cloud -> Edge planned | запланировано далее: `recipe_versions`/`recipe_lines` для KDS UI и stop-list checks |
| Stop-list | Cloud + Edge manager input | Yes, only stop-list overlay | Bi-directional planned via `StopListUpdated` | запланировано далее |
| Employee shift | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Cash session/drawer event | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Order/order line | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Precheck | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Payment | Edge | Yes | Edge -> Cloud operational events for capture | реализовано сейчас |
| Financial operation | Edge | Yes | `CancellationRecorded` and `RefundRecorded` are current Edge -> Cloud operational events | реализовано сейчас |
| Check | Edge | Generated after full payment; finalized checks are not rewritten by cancellation/refund | `CheckCreated` is current Edge -> Cloud operational event; `CheckRefunded` is legacy accepted | реализовано сейчас |
| Tax/pricing policy reference | Cloud | Edge read model only | `pricing_policy` Cloud -> Edge stream for `tax_profiles`, `tax_rules`, `service_charge_rules`, `pricing_policies` | реализовано сейчас |
| Operational order adjustments | Edge | Yes while order is open | runtime-команды; будущие policy ids могут ограничивать допустимые варианты | реализовано сейчас |
| Stock document/move/ledger | Cloud Inventory Worker | No | Edge business events -> Cloud worker | запланировано далее; Edge-side stock document service должен быть выведен из целевой архитектуры |

## Current Cloud -> Edge Ingest

`mastersync.Service` сейчас поддерживает только:

- `restaurants`
- `devices`
- `staff`
- `floor`
- `catalog`
- `menu`
- `pricing_policy`

`recipes`, `inventory_reference` и `stop_lists` нельзя документировать как реализованные POS Edge ingest streams, пока `mastersync.Service` не применяет их payloads.
`catalog` payload включает catalog folders/tags/services и modifier groups/options/bindings/effective links; `menu` payload включает menu items. Menu categories остаются отдельным понятием и не заменяют catalog folders.
`pricing_policy` включает tax/service-charge reference tables и automatic discount/surcharge policies; manual override runtime остается backend RBAC-controlled action.

Реализовано сейчас:

- POS Edge sync sender использует authenticated `POST /api/v1/sync/exchange` как приоритетный Cloud-Edge цикл, когда локальное provisioning state содержит `node_token`.
- Edge отправляет текущие `cloud_master_sync_state` revisions/checkpoints по поддерживаемым streams и получает только более новые Cloud packages.
- Cloud package apply и commit соответствующего stream checkpoint выполняются существующей transaction boundary `mastersync.Service`.
- Если локальный apply не проходит, Edge не помечает accepted outbox rows как `sent`; retry повторяет exchange, а Cloud idempotency возвращает стабильный ACK для уже принятого event.
- После successful pairing/assignment POS Edge не выполняет повторный Cloud device registration/snapshot provisioning loop; фоновая maintenance только регистрирует not configured node или poll-ит `pending_admin_approval`.
- Пустой exchange без Edge outbox throttled отдельным Cloud pull interval, а появившиеся Edge outbox events отправляются в ближайший worker tick без ожидания этого throttling interval.
- Cloud UI после успешного CRUD Cloud-owned master data автоматически создает новый published package через canonical publication API. Поэтому роль, сотрудник или PIN, созданные оператором в Cloud UI после pairing, попадают на Edge в ближайший Cloud -> Edge exchange. Ручная публикация остается реализована сейчас как явный operator checkpoint.

## Current Edge -> Cloud Runtime

Реализовано сейчас:

- cashier commands пишут local event/outbox foundation;
- POS runtime продолжает работу, если Cloud недоступен;
- Cloud receiver/projection foundation существует.
- authenticated `sync/exchange` принимает Edge events с item-level ACK и сохраняет существующую idempotency model;
- legacy `/sync/edge-events` и `/sync/edge-events/batch` остаются совместимыми inbound routes.

Реализовано сейчас:

- `CancellationRecorded` и `RefundRecorded` принимаются Cloud receiver и сохраняются как operational events;
- cashier UI for whole-check and partial `order_line`/quantity cancellation/refund не добавляет новые sync event names; он использует текущие Edge-owned ledger events и отправляет command id, inventory disposition и operation items как payload fields;
- `PaymentRefunded` и `CheckRefunded` остаются legacy accepted event types для старых payloads;
- Cloud event-type stats обновляются для всех accepted operational events;
- Cloud shift finance foundation считает coarse refund totals from current `RefundRecorded` and legacy `PaymentRefunded`/`CheckRefunded`, but it is not a full projection for financial operation ledger item scopes, inventory disposition, approval policy or original-shift reconciliation.
- Pagination/filtering закрытых заказов является local POS read-model behavior и не добавляет sync ownership или event names.
- Bounded outbox/local-event visibility в POS API/UI является local operational window and does not acknowledge, remove or archive sync rows.
- POS Edge storage lifecycle status/dry-run является local operational read model и не добавляет sync event names. Любая будущая destructive retention/archive policy должна блокироваться при наличии non-sent `edge_to_cloud` outbox messages; текущий runtime только сообщает это blocking state.
- manual Inventory service реализовано сейчас пишет `StockDocumentPosted` как local-only outbox/local event; это не часть Edge -> Cloud operational catalog и должно быть выведено из целевой архитектуры при переходе на Cloud-centric inventory.

Запланировано далее:

- Edge/KDS events `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `RefundRecorded`, `CancellationRecorded`, `StopListUpdated`;
- Cloud Inventory Worker создает `stock_documents` и `stock_ledger` из accepted events;
- `stock_balances` остаются аналитической проекцией и не блокируют продажи;
- ClickHouse получает только OLAP projection из Cloud PostgreSQL.
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
