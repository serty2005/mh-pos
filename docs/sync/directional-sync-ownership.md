# Directional Sync Ownership

Статус: актуально для текущего cashier runtime и целевого полного пилота.

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
| Recipe reference | Cloud | Edge read-only | Cloud -> Edge `recipes` | реализовано сейчас: `recipe_versions`/`recipe_lines` ingest для KDS UI и stop-list checks |
| Recipe change proposal | Cloud review queue | Edge creates suggestion only | Edge -> Cloud `RecipeChangeSuggested` | реализовано сейчас на POS Edge и Cloud review/apply |
| Catalog change proposal | Cloud review queue | Edge creates suggestion only | Edge -> Cloud `CatalogItemChangeSuggested` | реализовано сейчас на POS Edge и Cloud review/apply |
| Stop-list | Cloud + Edge kitchen/manager input | Edge runtime reads local overlay; Edge edit flow запланирован | Cloud -> Edge `inventory_reference` сейчас; Edge -> Cloud `StopListUpdated` запланировано | реализовано сейчас: sale blocking по local `stop_lists`; conflict policy запланирован далее |
| Employee shift | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Cash session/drawer event | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Order/order line | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Precheck | Edge | Yes | Edge -> Cloud operational events | реализовано сейчас |
| Payment | Edge | Захват относится к текущей кассовой смене кассира, а заказ сохраняет исходную личную смену | Edge -> Cloud operational events for capture | реализовано сейчас |
| Financial operation | Edge | Yes | `CancellationRecorded` and `RefundRecorded` are current Edge -> Cloud operational events | реализовано сейчас |
| Check | Edge | Создается после полной оплаты под кассовой сменой оплаты; финализированные чеки не переписываются через cancellation/refund | `CheckCreated` and `CheckClosed` are current Edge -> Cloud operational events; `CheckRefunded` is legacy accepted | реализовано сейчас |
| Tax/pricing policy reference | Cloud | Edge read model only | `pricing_policy` Cloud -> Edge stream for `tax_profiles`, `tax_rules`, `service_charge_rules`, `pricing_policies` | реализовано сейчас |
| Operational order adjustments | Edge | Yes while order is open | runtime-команды; будущие policy ids могут ограничивать допустимые варианты | реализовано сейчас |
| Stock document/move/ledger | Cloud Inventory Worker | No | Edge business events -> Cloud worker | реализовано сейчас for normalized item payloads; Edge-side stock document service был pre-pilot legacy и удален |
| Kitchen ticket/status | POS Edge/KDS | Yes | Edge -> Cloud `KitchenTicketStatusChanged`/`ItemServed` | реализовано сейчас для ticket lifecycle; receipt/proposal Edge input flows реализованы сейчас; stop-list edit flow запланирован далее |

## Текущий Cloud -> Edge Ingest

`mastersync.Service` сейчас поддерживает:

- `restaurants`
- `devices`
- `staff`
- `floor`
- `catalog`
- `menu`
- `pricing_policy`
- `recipes`
- `inventory_reference`

`catalog` payload включает catalog folders/tags/services и modifier groups/options/bindings/effective links; `menu` payload включает menu items. Menu categories остаются отдельным понятием и не заменяют catalog folders.
`pricing_policy` включает tax/service-charge reference tables и automatic discount/surcharge policies; manual override runtime остается backend RBAC-controlled action.
`recipes` включает `recipe_versions` и `recipe_lines`; `inventory_reference` включает `stop_lists`.

Запланировано до полного пилота:

- добавить Cloud authoring UI/publication workflow для recipes/stop-list;
- не включать Cloud-owned stock documents/moves/balances в Edge ingest.

Реализовано сейчас:

- POS Edge sync sender использует authenticated `POST /api/v1/sync/exchange` как приоритетный Cloud-Edge цикл, когда локальное provisioning state содержит `node_token`.
- Edge отправляет текущие `cloud_master_sync_state` revisions/checkpoints по поддерживаемым streams и получает только более новые Cloud packages.
- Sync sender работает по строгому poll interval. При достижении `POS_SYNC_SENDER_EMERGENCY_PENDING_THRESHOLD` для pending Edge -> Cloud outbox rows следующая итерация выполняется немедленно, чтобы backlog не ждал штатного таймера.
- Cloud ограничивает Cloud -> Edge выдачу `CLOUD_SYNC_MAX_CLOUD_PACKAGES_PER_EXCHANGE`; Edge забирает оставшиеся changed streams последовательными exchange-сессиями после применения предыдущей порции.
- Cloud package apply и commit соответствующего stream checkpoint выполняются существующей transaction boundary `mastersync.Service`.
- Если отдельный Cloud package не проходит локальный apply, Edge фиксирует проблемный stream в `cloud_master_sync_state` со статусом `failed`, продолжает применять остальные packages и не блокирует accepted Edge -> Cloud ACK. Ошибочный package не становится новым stream checkpoint; Edge продолжает объявлять последний успешно примененный checkpoint.
- Если transport/auth exchange не завершился успешно, Edge не помечает outbox rows как `sent`; retry повторяет exchange, а Cloud idempotency возвращает стабильный ACK для уже принятого event.
- После successful pairing/assignment POS Edge не выполняет повторный Cloud device registration/snapshot provisioning loop; фоновая maintenance только регистрирует not configured node или poll-ит `pending_admin_approval`.
- Повторный Cloud assignment-status для уже выданного `node_token` не ротирует token hash, чтобы внешние проверки статуса не приводили к `401 SYNC_UNAUTHORIZED` на последующих `sync/exchange`.
- Пустой exchange без Edge outbox throttled отдельным Cloud pull interval, а появившиеся Edge outbox events отправляются в ближайший worker tick без ожидания этого throttling interval.
- Cloud UI после успешного CRUD Cloud-owned master data автоматически создает новый published package через canonical publication API. Поэтому роль, сотрудник или PIN, созданные оператором в Cloud UI после pairing, попадают на Edge в ближайший Cloud -> Edge exchange. Ручная публикация остается реализована сейчас как явный operator checkpoint.

## Текущий Edge -> Cloud Runtime

Реализовано сейчас:

- cashier commands пишут local event/outbox foundation;
- POS runtime продолжает работу, если Cloud недоступен;
- Cloud receiver/projection foundation существует.
- authenticated `sync/exchange` принимает Edge events с item-level ACK и сохраняет существующую idempotency model;
- проблемные Edge -> Cloud items сохраняются в `cloud_sync_problem_events` и не блокируют остальные items в batch/exchange;
- legacy `/sync/edge-events` и `/sync/edge-events/batch` остаются совместимыми inbound routes.

Замороженный принцип для event archive:

```text
Edge Outbox
  -> Cloud API (PostgreSQL inbox_events)
  -> Async Batch Forwarder
  -> ClickHouse raw_business_events
```

- Все Edge POS/KDS `event_id` должны быть UUIDv7.
- Реализовано сейчас: Cloud API подтверждает прием после записи в PostgreSQL `inbox_events`.
- Реализовано сейчас: ClickHouse write выполняет только async batch forwarder, не request handler.
- Реализовано сейчас: после successful export в ClickHouse row получает `processed_for_olap = true`, а checkpoint/retry state хранится в `olap_export_checkpoints` и `inbox_events`.
- Processed rows старше 3 месяцев можно удалить из PostgreSQL; ClickHouse хранит historical business event trail бессрочно.

Реализовано сейчас:

- `CancellationRecorded` и `RefundRecorded` принимаются Cloud receiver и сохраняются как operational events;
- Cloud receiver валидирует current financial operation payload fields: operation id, edge operation id, совпадение payload `restaurant_id`/`device_id` с envelope, check id, precheck id, original/current shift ids, amount, currency, business date, operation-level inventory disposition, reason и immutable snapshot;
- cashier UI for whole-check and partial `order_line`/quantity cancellation/refund не добавляет новые sync event names; он использует текущие Edge-owned ledger events и отправляет command id, inventory disposition и operation items как payload fields;
- `PaymentRefunded` и `CheckRefunded` остаются legacy accepted event types для старых payloads;
- Cloud event-type stats обновляются для всех accepted operational events;
- Cloud shift finance foundation считает coarse refund totals from current `RefundRecorded` and legacy `PaymentRefunded`/`CheckRefunded`; detailed `cloud_projection_financial_operations` stores current `CancellationRecorded`/`RefundRecorded` operations for reporting filters. Legacy events do not become primary financial operation projection rows.
- Pagination/filtering закрытых заказов является local POS read-model behavior и не добавляет sync ownership или event names.
- Bounded outbox/local-event visibility в POS API/UI является local operational window and does not acknowledge, remove or archive sync rows.
- POS Edge storage lifecycle status/dry-run/manifest-only export-plan/export-only archive/verify/read-plan/lookup/apply-plan является local operational read/export/verification/apply model по exclusive cutoff rule `checks.business_date_local < cutoff_business_date_local` и не добавляет sync event names. Verify/read-plan/lookup читают только ранее созданный локальный archive artifact, проверяют integrity/payload policy и не создают sync envelopes. Destructive apply блокируется при наличии active/open blockers и non-sent `edge_to_cloud` outbox messages; при verified archive и clean runtime scope apply-plan выполняет локальное физическое удаление + `VACUUM`, не создавая sync envelopes.
- Исторически manual Inventory service писал `StockDocumentPosted` как local-only outbox/local event; этот pre-pilot legacy path удален и не входит в Edge -> Cloud operational catalog.

Реализовано сейчас:

- POS Edge генерирует `CheckClosed` при создании final check после полной оплаты; событие строится из immutable `check.Snapshot`;
- Edge/KDS events `CheckClosed`, `KitchenTicketStatusChanged`, `ItemServed`, `StockReceiptCaptured`, `CatalogItemChangeSuggested`, `RecipeChangeSuggested`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted`, `RefundRecorded`, `CancellationRecorded`, `StopListUpdated`;
- Cloud Inventory Worker создает `stock_documents` и `stock_ledger` из accepted events;
- `stock_balances` остаются аналитической проекцией и не блокируют продажи;

Требуется до полного пилота:

- advanced KDS должен генерировать `KitchenTicketStatusChanged`, `ItemServed` и cooking events;
- kitchen receipt/proposal flows должны генерировать `StockReceiptCaptured`, `CatalogItemChangeSuggested` и `RecipeChangeSuggested`;
- Cloud receiver/worker должен сохранить идемпотентность replay и дедупликацию `ItemServed` с `CheckClosed`;
- stop-list changes должны синхронизироваться без raw sensitive payload в UI/API diagnostics.
- полный Cloud Inventory Engine должен обработать receipts, counts, production, sale consumption, refund/cancellation dispositions, balances и costing/recalculation state.

Запланировано до полного пилота:

- Реализовано сейчас: ClickHouse получает immutable `raw_business_events` из Cloud PostgreSQL `inbox_events`.
- Запланировано далее: ClickHouse получает `olap_stock_moves` projection из Cloud inventory data.
- Реализовано сейчас: Cloud OLAP API читает bounded `raw_business_events` metadata из ClickHouse и не участвует в transactional command validation; bounded aggregates запланированы далее.

Запланировано далее:

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

## Pricing and tax ownership

Статус: реализовано сейчас.

Pricing/tax policy ownership остается Cloud -> Edge. POS Edge runtime выбирает опубликованную policy по id и не становится владельцем размера скидки, надбавки, порядка применения или налоговой логики. Runtime order/precheck/check владеют только фактом применения и immutable snapshot результата расчета.
