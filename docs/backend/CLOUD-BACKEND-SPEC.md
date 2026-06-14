# Cloud Backend Spec

Статус: актуальный Cloud backend contract для текущего cashier runtime и целевого полного пилота на 2026-05-21.

Код, миграции и тесты являются источником истины для фактически реализованного runtime. Этот документ описывает текущий Cloud Backend, восстановленные архитектурные решения и границы, но не документирует будущую функциональность как реализованную.

## Источники фактов

Реализовано сейчас подтверждается:

- `cloud-backend/cmd/cloud-api/main.go`;
- `cloud-backend/internal/cloudsync/*`;
- `cloud-backend/internal/masterdata/*`;
- `cloud-backend/internal/provisioning/*`;
- `cloud-backend/internal/platform/postgres/*`;
- `cloud-backend/migrations/postgres/001_init.sql`;
- `cloud-backend/README.md`;
- `docs/adr/ADR-015-persistence-and-analytics-strategy.md`;
- `docs/adr/ADR-016-clickhouse-immutable-event-store.md`;
- `docs/sync/edge-cloud-contracts-v1.md`;
- `docs/sync/directional-sync-ownership.md`;
- `docs/ui/CLOUD-UI-SPEC.md`;
- `docs/backend/POS-DATA-AND-MIGRATIONS.md`;
- `docs/backend/INVENTORY-COSTING-SPEC.md`;
- история git по Cloud/master-data/sync/provisioning commits.

Если этот документ конфликтует с кодом или тестами, сначала фиксируется фактическое поведение по коду, затем обновляется документация.

## Восстановленные решения из git

Реализовано сейчас:

- `3e2632b` зафиксировал Cloud как production-like authority для master data: роли, сотрудники, каталог, меню и публикации живут в Cloud; POS Edge остается offline-first runtime и получает опубликованные read models.
- `05311bd` расширил Cloud до production onboarding: рестораны, entity list/get/archive, stream `restaurants`, Cloud-side duplicate PIN policy и Edge-ready snapshot endpoint.
- `eae2d32` ввел Zero-to-Cashier: provisioning bounded context, Cloud Approve, License Code flow, floor stream, structured error envelope и безопасный Cloud -> Edge snapshot apply.
- `d3789b0` добавил безопасный Cloud UI журнал входящих Edge events без raw payload и выровнял accepted event catalog со schema baseline.
- `6eb98a0` подтвердил, что транспорт Cloud -> Edge работает через publication version/checkpoint; после Cloud-owned CRUD нужна публикация, иначе Edge честно остается на старой версии.
- `51d95d0` принял ADR-016: PostgreSQL остается транзакционным Cloud store, ClickHouse является будущим бессрочным архивом business events; синхронный dual-write запрещен.
- `689e075` перенес целевую складскую архитектуру в Cloud: stock documents, stock ledger, costing jobs и stop-lists принадлежат Cloud Inventory Worker; Edge-side stock foundation был переходным legacy и удален при cutover.
- `86e1dee` усилил current financial operation payload contract: `CancellationRecorded`/`RefundRecorded` требуют identity fields, check/precheck/shift/date/reason/snapshot и не смешиваются с legacy refund events.

Вне текущего объема:

- production auth/RBAC perimeter для Cloud API;
- rich BI/reporting UI beyond pilot financial operations and OLAP API;

Реализовано сейчас:

- Cloud authoring/publication workflow для streams `recipes` и `inventory_reference` поверх Cloud authority tables;
- review/apply очереди для `CatalogItemChangeSuggested` и `RecipeChangeSuggested`, созданных kitchen worker на Edge;
- обработка `KitchenTicketStatusChanged`, `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted`, `CatalogItemChangeSuggested`, `RecipeChangeSuggested` как business events без synchronous apply в request path; `StopListUpdated` принимается receiver-ом, попадает в durable `inventory_event_queue` и обрабатывается Cloud Inventory Worker в bounded projection без raw payload exposure;
- `stop_list_conflict_policy` для `StopListUpdated`: `cloud_wins`, `edge_overlay_until_next_publication`, `edge_overlay_requires_manager_review`; default `edge_overlay_requires_manager_review`;
- safe readiness API/UI signal для stop-list publication, последнего известного Edge ACK metadata и sync problem counters без raw payload;
- поддержка `CheckClosed`/`ItemServed`/`StockWriteOffCaptured` как pilot inventory facts через текущий receiver и Inventory Worker;
- Cloud Inventory Worker выполняет recipe expansion основной позиции продажи по active recipe version и modifier-linked consumption по nullable `ModifierOption.linked_catalog_item_id`; linked modifier item списывается напрямую, без recipe expansion linked item;
- ClickHouse first slices: managed `raw_business_events`, async forwarder из PostgreSQL `inbox_events`, `processed_for_olap`, retry state, export checkpoint и bounded read-only metadata API; managed `olap_stock_moves`, async export из PostgreSQL `stock_ledger`, bounded stock moves API, read-only export status, минимальный support-only export retry control, async backfill job foundation, первый bounded stock movement summary, первый bounded sales/kitchen summary и bounded kitchen timing summary.

Запланировано до полного пилота:

- full inventory engine beyond текущего bounded worker slice: materialized balances, production-grade receipts/counts, semi-finished auto-production split, costing и retro recalculation; bounded refund/cancellation dispositions `return_to_stock`/`write_off_waste` реализованы сейчас;
- production auth/RBAC perimeter для mutating OLAP controls, richer sales aggregates и COGS/margin после появления достоверной cost basis.

## Назначение

Реализовано сейчас:

- Cloud Backend принимает операционные события POS Edge.
- Cloud Backend хранит PostgreSQL receipts, raw payload checksums, operational journal и проекции.
- Cloud Backend является источником истины для Cloud-authored master/reference/configuration data.
- Cloud Backend публикует Cloud -> Edge packages и Edge-ready snapshots.
- Cloud Backend управляет подключением Edge-устройств через Cloud Approve и License Code.
- Cloud Backend обслуживает Cloud UI на локальном pilot perimeter.

Не реализовано сейчас:

- Cloud не выполняет cashier runtime commands: заказы, предчеки, оплаты и чеки создаются на POS Edge.
- Cloud не является платежным процессором и не выполняет fiscalization.
- Cloud не принимает POS operator session как security boundary.
- Cloud использует ClickHouse для async immutable `raw_business_events` archive, но не делает synchronous dual-write в request path.
- Cloud предоставляет pilot-ready CRUD/publication UI для recipes/stop-list в текущем bounded объеме; реализовано сейчас добавлен сценарный editor версий техкарт с draft, submit в review и approve/apply через Cloud-owned publication path. Production-grade lifecycle polish остается запланированным далее.
- Реализовано сейчас: Cloud предоставляет review/apply runtime для Edge-originated `CatalogItemChangeSuggested`/`RecipeChangeSuggested` и `StopListUpdated` через маршруты `GET/approve/reject/request-changes`; apply выполняется только на approve с последующей публикацией и без прямой мутации Edge runtime.

## Runtime Modules

Реализовано сейчас:

- `cmd/cloud-api` — entrypoint, конфигурация, PostgreSQL startup migration/verification, HTTP server lifecycle.
- `internal/cloudsync` — прием Edge events, batch ACK, authenticated `sync/exchange`, problem item quarantine, master-data package storage, operational projections.
- `internal/masterdata` — Cloud-owned справочники, lifecycle, PIN policy, publication workflow, Edge DTO generation.
- `internal/provisioning` — регистрация/назначение Edge-устройств, pairing code flow через License Server, node token lifecycle.
- `internal/platform/httpx` — безопасный error envelope.
- `internal/platform/postgres` — managed migration policy, backup-before-upgrade, schema verification, runtime version gate.
- `internal/platform/logging` — structured JSON logging.

## Startup And Config

Реализовано сейчас:

- Cloud Backend читает defaults, затем env, затем JSON config file.
- `CLOUD_CONFIG_PATH` задает обязательный файл, если указан явно; default optional path: `config/cloud-api.json`.
- `CLOUD_POSTGRES_DSN` обязателен.
- `CLOUD_HTTP_ADDR` по умолчанию `:8090`.
- `CLOUD_PUBLIC_URL` используется в provisioning/snapshot URLs.
- `LICENSE_SERVER_URL` включает интеграцию с License Server для License Code flow.
- `CLOUD_POSTGRES_MIGRATIONS_DIR` по умолчанию `migrations/postgres`.
- `CLOUD_POSTGRES_BACKUP_DIR` по умолчанию `data/cloud-backups`.
- `MH_POS_VERSION` участвует в runtime version gate.
- `CLOUD_SYNC_MAX_CLOUD_PACKAGES_PER_EXCHANGE` ограничивает Cloud -> Edge packages в одном `sync/exchange` response, чтобы крупные публикации уходили несколькими последовательными сессиями.

Startup path:

1. Загружается конфигурация.
2. Инициализируется JSON logger.
3. Открывается PostgreSQL pool.
4. Выполняется managed migration policy с backup и schema verification.
5. Заполняется canonical currency reference catalog.
6. Создаются repositories и application services.
7. HTTP server стартует только после успешной проверки схемы.

Правила:

- `DB version > MH_POS_VERSION` завершает startup fail-fast.
- Checksum drift при той же версии считается ошибкой.
- Business runtime не должен обращаться к таблицам до startup migration/verification.
- Manual ad-hoc SQL не является canonical repair path.

## Public Routes

Реализовано сейчас в `cloud-backend/internal/cloudsync/api/router.go`, `masterdata/api/router.go`, `provisioning/api/router.go`.

Core:

- `GET /health`

Sync receiver:

- `GET /api/v1/sync/edge-events`
- `GET /api/v1/sync/readiness/stop-list?restaurant_id=&node_device_id=` — safe readiness summary по stop-list publication/package, последнему `StopListUpdated` ACK metadata и `cloud_sync_problem_events` counters без raw payload.
- `POST /api/v1/sync/edge-events`
- `POST /api/v1/sync/edge-events/batch`
- `POST /api/v1/sync/exchange`

Inventory read model:

- `GET /api/v1/inventory/stock-ledger?restaurant_id=&source_event_type=&source_event_id=&order_line_id=&catalog_item_id=&limit=&offset=` — bounded read-only view of Cloud-owned `stock_ledger` без raw Edge payload.
- `GET /api/v1/inventory/stock-balances?restaurant_id=&warehouse_id=&catalog_item_id=&business_date_to=&costing_status=&limit=&offset=` — bounded Cloud-owned aggregate поверх PostgreSQL `stock_ledger`; response содержит `quantity_on_hand`, `unit_code`, aggregate `costing_status`, `needs_recalculation`, `last_movement_at`, `business_date_to` без raw Edge payload, COGS или margin.

	OLAP read model:

- `GET /api/v1/olap/raw-business-events?restaurant_id=&event_type=&occurred_from=&occurred_to=&limit=&offset=` — bounded ClickHouse metadata view без raw payload.
- `GET /api/v1/olap/stock-moves?restaurant_id=&business_date_from=&business_date_to=&catalog_item_id=&warehouse_id=&source_event_type=&limit=&offset=` — bounded ClickHouse stock movement view из `olap_stock_moves` без raw sync payload.
- `GET /api/v1/olap/export-status?stream=raw_business_events|stock_moves` — read-only PostgreSQL checkpoint/retry status без raw payload и без mutation/backfill side effects.
- `POST /api/v1/olap/export-retry` — support-only control для `stream=raw_business_events|stock_moves` и `mode=retry_failed|resume_from_checkpoint`; request требует UUIDv7 `command_id` и operator reason, response не содержит raw payload/reason, mutation меняет только PostgreSQL retry/backoff control state и command log. Cloud UI текущего scope этот endpoint не вызывает.
- `GET /api/v1/olap/stock-move-summary?restaurant_id=&business_date_from=&business_date_to=&catalog_item_id=&warehouse_id=&source_event_type=&group_by=business_date|catalog_item|warehouse&limit=&offset=` — bounded ClickHouse aggregate по `olap_stock_moves`; не является COGS/margin API.
- `GET /api/v1/olap/sales-kitchen-summary?restaurant_id=&business_date_from=&business_date_to=&group_by=business_date|event_type|source_event_type|catalog_item&limit=&offset=` — bounded read-only aggregate по `raw_business_events` и `olap_stock_moves`; response не содержит raw payload/hash, не является BI dashboard, COGS/margin или cashier command API.
- `GET /api/v1/olap/kitchen-timing-summary?restaurant_id=&business_date_from=&business_date_to=&station_id=&group_by=business_date|station&limit=&offset=` — bounded KDS timing aggregate поверх `KitchenTicketStatusChanged`/`ItemServed`; response содержит lifecycle counts и средние длительности без raw payload.
- `GET /api/v1/olap/backfill-jobs?stream=&status=&limit=&offset=`, `POST /api/v1/olap/backfill-jobs`, `GET /api/v1/olap/backfill-jobs/{id}` и `POST /api/v1/olap/backfill-jobs/{id}/cancel` — support/operator-only async backfill foundation с UUIDv7 `command_id`, checkpoint/progress/status/error metadata и audit trail; HTTP handlers не пишут business rows в ClickHouse.

Использование в Cloud UI:

- реализовано сейчас: legacy `cloud-ui` читает `stock-balances`, `olap/export-status`, `olap/stock-moves`, `olap/stock-move-summary`, `olap/backfill-jobs` и `olap/kitchen-timing-summary` как bounded operator surface с safe фильтрами и без raw payload display.
- реализовано сейчас: активный `cloud-ui-g` читает publication state, выполняет publication, работает с Edge-device flow, master data и safe Edge events list; inventory/OLAP/reporting screens в `cloud-ui-g` еще не реализованы.
- реализовано сейчас: publication panel читает только safe `GET /api/v1/restaurants/{id}/master-data/publication-state`; routes `/master-data/packages/*`, `/master-data/snapshot` и `sync/exchange` переносят package payload/snapshot для Edge delivery и не являются Cloud UI read-only delivery-status contract.
- вне текущего объема: Cloud UI не вызывает support-only mutating `POST /api/v1/olap/export-retry` и `POST /api/v1/olap/backfill-jobs`, не показывает COGS/margin и не превращает bounded slices в BI dashboard.
- запланировано далее: отдельный safe read-only package delivery status/Edge package ACK DTO для Cloud UI; до появления такого route UI не должен имитировать delivery state поверх payload routes или raw exchange payload.

Generic Cloud -> Edge package storage:

- `PUT /api/v1/provisioning/master-data/{stream}`
- `GET /api/v1/provisioning/master-data/{stream}`

Provisioning:

- `POST /api/v1/devices/register`
- `GET /api/v1/devices/unassigned`
- `POST /api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign`
- `GET /api/v1/devices/{node_device_id}/assignment-status`
- `POST /api/v1/restaurants/{restaurant_id}/devices/generate-pairing-code`

Master data under canonical namespace:

- `POST /api/v1/master-data/roles`
- `GET /api/v1/master-data/roles`
- `GET /api/v1/master-data/roles/{id}`
- `PATCH /api/v1/master-data/roles/{id}`
- `POST /api/v1/master-data/roles/{id}/archive`
- `POST /api/v1/master-data/employees`
- `GET /api/v1/master-data/employees`
- `GET /api/v1/master-data/employees/{id}`
- `PATCH /api/v1/master-data/employees/{id}`
- `POST /api/v1/master-data/employees/{id}/suspend`
- `POST /api/v1/master-data/employees/{id}/activate`
- `POST /api/v1/master-data/employees/{id}/archive`
- `POST /api/v1/master-data/employees/{id}/role`
- `POST /api/v1/master-data/employees/{id}/pin`
- `POST /api/v1/master-data/employees/{id}/pin/rotate`
- `POST /api/v1/master-data/catalog/items`
- `GET /api/v1/master-data/catalog/items`
- `GET /api/v1/master-data/catalog/items/{id}`
- `PATCH /api/v1/master-data/catalog/items/{id}`
- `POST /api/v1/master-data/catalog/items/{id}/archive`
- `POST /api/v1/master-data/catalog/folders`
- `GET /api/v1/master-data/catalog/folders`
- `PATCH /api/v1/master-data/catalog/folders/{id}`
- `POST /api/v1/master-data/catalog/folders/{id}/archive`
- `POST /api/v1/master-data/catalog/folder-parameters`
- `GET /api/v1/master-data/catalog/folder-parameters`
- `PATCH /api/v1/master-data/catalog/folder-parameters/{id}`
- `POST /api/v1/master-data/catalog/tags`
- `GET /api/v1/master-data/catalog/tags`
- `PATCH /api/v1/master-data/catalog/tags/{id}`
- `POST /api/v1/master-data/catalog/item-tags`
- `POST /api/v1/master-data/modifiers/groups`
- `GET /api/v1/master-data/modifiers/groups`
- `PATCH /api/v1/master-data/modifiers/groups/{id}`
- `POST /api/v1/master-data/modifiers/options`
- `GET /api/v1/master-data/modifiers/options`
- `PATCH /api/v1/master-data/modifiers/options/{id}`
- `POST /api/v1/master-data/modifiers/bindings`
- `GET /api/v1/master-data/modifiers/bindings`
- `PATCH /api/v1/master-data/modifiers/bindings/{id}`
- `POST /api/v1/master-data/pricing/policies`
- `GET /api/v1/master-data/pricing/policies`
- `PATCH /api/v1/master-data/pricing/policies/{id}`
- `PUT /api/v1/provisioning/master-data/pricing_policy`
- `GET /api/v1/provisioning/master-data/pricing_policy?node_device_id=...`
- Реализовано сейчас (route-backed aliases в текущем runtime):
  - `POST /api/v1/master-data/recipes/items`
  - `GET /api/v1/master-data/recipes/items`
  - `PATCH /api/v1/master-data/recipes/items/{id}`
  - `GET /api/v1/master-data/recipes/versions?restaurant_id=&owner_catalog_item_id=&status=&limit=&offset=`
  - `POST /api/v1/master-data/recipes/versions/drafts`
  - `POST /api/v1/master-data/recipes/versions/{id}/submit`
  - `POST /api/v1/master-data/inventory/stop-list`
  - `GET /api/v1/master-data/inventory/stop-list`
  - `PATCH /api/v1/master-data/inventory/stop-list/{id}`
  - `POST /api/v1/master-data/inventory/stop-list/{id}/deactivate`
- Реализовано сейчас:
  - `GET /api/v1/master-data/catalog-suggestions?restaurant_id=&status=&limit=&offset=`
  - `POST /api/v1/master-data/catalog-suggestions/{id}/approve`
  - `POST /api/v1/master-data/catalog-suggestions/{id}/reject`
  - `POST /api/v1/master-data/catalog-suggestions/{id}/request-changes`
  - `GET /api/v1/master-data/recipe-suggestions?restaurant_id=&status=&limit=&offset=`
  - `POST /api/v1/master-data/recipe-suggestions/{id}/approve`
  - `POST /api/v1/master-data/recipe-suggestions/{id}/reject`
  - `POST /api/v1/master-data/recipe-suggestions/{id}/request-changes`
  - Review command body для approve/reject/request-changes: `reviewed_by_employee_id`, optional `review_comment`, optional `published_by`; approve применяет suggestion и создает новую master-data publication, reject/request-changes меняют только review status/comment metadata.
- Реализовано сейчас для Edge-origin stop-list review:
  - `GET /api/v1/manager/stop-list-updates?restaurant_id=&status=&limit=&offset=`
  - `GET /api/v1/manager/stop-list-updates/{id}`
  - `POST /api/v1/manager/stop-list-updates/{id}/approve`
  - `POST /api/v1/manager/stop-list-updates/{id}/reject`
  - `POST /api/v1/manager/stop-list-updates/{id}/request-changes`
  - `POST /api/v1/manager/stop-list-updates/{id}/assign`
  - `POST /api/v1/manager/stop-list-updates/{id}/unassign`
  - `GET /api/v1/manager/stop-list-updates/{id}/audit?limit=&offset=`
- Реализовано сейчас для assignment audit read:
  - `GET /api/v1/manager/catalog-suggestions/{id}/audit?limit=&offset=`
  - `GET /api/v1/manager/recipe-suggestions/{id}/audit?limit=&offset=`

Реализовано сейчас: assignment поддержан только для Edge-origin stop-list review item из `StopListUpdated`. `assign` принимает UUIDv7 `command_id`, `assigned_to_employee_id`, `assigned_by_employee_id` и safe `reason`; `unassign` принимает UUIDv7 `command_id`, `unassigned_by_employee_id` и safe `reason`. Replay того же `command_id` идемпотентен и не добавляет повторную audit row. Terminal statuses `approved` и `rejected` нельзя назначать или снимать с назначения. Responses возвращают только safe assignment metadata и не содержат raw payload. Assignment audit read реализовано сейчас через bounded routes для `stop_list_update`, `catalog_suggestion` и `recipe_suggestion`: default `limit=50`, max `100`, `offset` non-negative, stable sort `occurred_at DESC, event_id DESC`; unknown review id возвращает safe empty list. Response содержит только `event_id`, `review_id`, `review_type`, `action`, `actor_employee_id`, `target_employee_id`, `occurred_at`, `reason` и safe `command_id`, без `payload_json`, raw payload, sync envelope, request dump, token/PIN/SQL details. Assignment runtime для `catalog_suggestion` и `recipe_suggestion` запланирован далее и не заявлен как реализованный. Escalation/dashboard запланированы далее. Raw payload exposure вне текущего объема и запрещено.
  - Stop-list review DTO содержит только safe projection fields из `cloud_projection_stop_list_updates`; raw Edge payload не отдается. Approve пишет Cloud-owned `stop_lists` row и публикацию, reject/request-changes меняют только review audit state.
- `POST /api/v1/master-data/menu/categories`
- `POST /api/v1/master-data/floor/halls`
- `GET /api/v1/master-data/floor/halls`
- `PATCH /api/v1/master-data/floor/halls/{id}`
- `POST /api/v1/master-data/floor/halls/{id}/archive`
- `POST /api/v1/master-data/floor/tables`
- `GET /api/v1/master-data/floor/tables`
- `PATCH /api/v1/master-data/floor/tables/{id}`
- `POST /api/v1/master-data/floor/tables/{id}/archive`
- `POST /api/v1/master-data/menu/items`
- `GET /api/v1/master-data/menu/items`
- `GET /api/v1/master-data/menu/items/{id}`
- `PATCH /api/v1/master-data/menu/items/{id}`
- `POST /api/v1/master-data/menu/items/{id}/archive`
- `POST /api/v1/master-data/publications`
- `GET /api/v1/master-data/published`

Production-oriented aliases:

- `POST /api/v1/restaurants`
- `GET /api/v1/restaurants`
- `GET /api/v1/restaurants/{id}`
- `PATCH /api/v1/restaurants/{id}`
- `POST /api/v1/restaurants/{id}/archive`
- `PATCH /api/v1/restaurants/{id}/archive`
- `POST /api/v1/roles`
- `GET /api/v1/roles`
- `GET /api/v1/roles/{id}`
- `PATCH /api/v1/roles/{id}`
- `POST /api/v1/roles/{id}/archive`
- `PATCH /api/v1/roles/{id}/archive`
- `POST /api/v1/employees`
- `GET /api/v1/employees`
- `GET /api/v1/employees/{id}`
- `PATCH /api/v1/employees/{id}`
- `POST /api/v1/employees/{id}/suspend`
- `POST /api/v1/employees/{id}/activate`
- `POST /api/v1/employees/{id}/archive`
- `POST /api/v1/employees/{id}/pin`
- `POST /api/v1/employees/{id}/pin/rotate`
- `POST /api/v1/catalog/items`
- `GET /api/v1/catalog/items`
- `GET /api/v1/catalog/items/{id}`
- `PATCH /api/v1/catalog/items/{id}`
- `POST /api/v1/catalog/items/{id}/archive`
- `POST /api/v1/menu/items`
- `GET /api/v1/menu/items`
- `GET /api/v1/menu/items/{id}`
- `PATCH /api/v1/menu/items/{id}`
- `POST /api/v1/menu/items/{id}/archive`
- `POST /api/v1/halls`
- `GET /api/v1/halls`
- `PATCH /api/v1/halls/{id}`
- `POST /api/v1/halls/{id}/archive`
- `POST /api/v1/tables`
- `GET /api/v1/tables`
- `PATCH /api/v1/tables/{id}`
- `POST /api/v1/tables/{id}/archive`
- `POST /api/v1/restaurants/{id}/master-data/publish`
- `GET /api/v1/restaurants/{id}/master-data/publication-state`
- `GET /api/v1/restaurants/{id}/master-data/packages/latest`
- `GET /api/v1/restaurants/{id}/master-data/packages/{package_id}`
- `GET /api/v1/restaurants/{id}/edge-nodes/{node_device_id}/master-data/snapshot`

Правила:

- Master-data mutation routes используют strict JSON decode: неизвестные поля отклоняются.
- Некоторые aliases сохранены для production-like сценариев и Cloud UI compatibility; canonical Cloud UI может использовать `/master-data/...` routes.
- `GET /api/v1/restaurants/{id}/master-data/publication-state` до первой публикации возвращает `200 null`.

## Error Contract And Logging

Реализовано сейчас:

- Cloud API возвращает безопасный envelope:

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message_key": "errors.validation",
    "details": {},
    "correlation_id": "request-id"
  }
}
```

- Для ошибок выставляется `X-Error-Code`.
- `request_id` создается middleware и попадает в logs/error envelope.
- Structured logs содержат `operation`, `action`, `result`, `error_code`, `method`, `path`, `status`, `duration_ms`, `remote_ip`.
- Cloud UI CORS разрешает `http://localhost:5174`, `http://127.0.0.1:5174`, `http://host.docker.internal:5174`.

Запрещено:

- Возвращать PIN, `pin_hash`, node token, raw pairing code secret, raw auth payload, raw Edge payload или stack trace в UI-facing response.
- Использовать frontend visibility как security boundary.

## Cloud Master Data Authority

Реализовано сейчас:

- Cloud владеет restaurant identity, business-day config, roles, employees, PIN credentials, floor, catalog, menu, modifiers, pricing policies и publication workflow.
- POS Edge не является production CRUD для этих сущностей.
- POS Edge получает published state через Cloud -> Edge packages/snapshot delivery и локально работает offline.

Сущности:

- Restaurants: `active`, `archived`, timezone, currency, `business_day_mode`, `business_day_boundary_local_time`.
- Roles: scoped by restaurant, permission snapshot, validation canonical permission ids.
- Employees: `active`, `suspended`, `archived`, role assignment, PIN credential version, safe `pin_configured`.
- Floor: halls and tables.
- Catalog: `dish`, `good`, `semi_finished`, `service`; folders, folder parameters, tags, item tags.
- Modifiers: groups, options, bindings by target `menu_item`, `catalog_item`, `folder`, `tag`.
- Pricing policies: discount/surcharge, line/order scope, fixed/percentage amount, `application_index`, `manual`, `requires_permission`, lifecycle.
- Menu: menu items, category foundation, availability JSON, station routing foundation.

Lifecycle:

- Restaurant status: `active`, `archived`.
- Employee status: `active`, `suspended`, `archived`.
- Common lifecycle status: `draft`, `published`, `archived`.
- Catalog item kinds: `dish`, `good`, `semi_finished`, `service`.

PIN policy:

- Plain PIN принимается только на входе create/rotate use cases.
- Cloud UI-facing API не возвращает raw PIN или `pin_hash`.
- `pin_hash` присутствует только в device/system staff package для offline PIN auth на POS Edge.
- PIN должен быть уникален в рамках ресторана среди сотрудников `active` и `suspended`.
- Archived employee не блокирует повторное использование PIN.
- Duplicate active/suspended PIN возвращает conflict с `PIN_ALREADY_EXISTS`.

## Publication Workflow

Реализовано сейчас:

- Publication создает монотонную версию для ресторана.
- Publication сохраняет `cloud_master_data_publications`.
- Publication строит deterministic stream packages и сохраняет их в `cloud_master_data_packages`.
- Активный `cloud-ui-g` вызывает publication API из отдельной publication panel с explicit `published_by`; ручная публикация остается operator checkpoint.
- Edge-ready snapshot endpoint возвращает typed ingest DTO, который POS Edge может применить без PowerShell field stripping.

Текущие published streams:

- `restaurants`;
- `staff`;
- `floor`;
- `catalog`;
- `menu`;
- `pricing_policy`;
- `recipes` через generic package storage/validation;
- `inventory_reference` через generic package storage/validation;
- `currencies` через generic package storage/validation.

Publication DTO правила:

- `catalog` stream содержит catalog items, folders, folder parameters, tags, item tags, modifier groups/options/bindings; `folder_parameters`, `tags` и `item_tags` публикуются с `restaurant_id` для POS Edge restaurant-scoped ingest.
- `menu` stream содержит menu items and link-only `menu_item_modifier_groups`.
- Rich modifier group fields не вкладываются внутрь `menu_items`.
- `modifier_groups[]` передает `id`, `restaurant_id`, `name`, `required`, `min_count`, `max_count`, `active`.
- `modifier_options[]` передает `id`, `restaurant_id`, `modifier_group_id`, optional read-only `linked_catalog_item_id`, `name`, `price_minor`, `active`.
- `modifier_bindings[]` передает `id`, `restaurant_id`, `modifier_group_id`, `target_type`, `target_id`, `sort_order`, `active`.
- `pricing_policy` stream передает `tax_profiles`, `tax_rules`, `service_charge_rules`, `pricing_policies`, когда соответствующий package сохранен/опубликован.
- `recipes` stream передает `recipe_versions` и `recipe_lines`, когда соответствующий package сохранен/опубликован.
- `inventory_reference` stream передает `stop_lists` и Cloud-owned default `warehouses` (`warehouse-main` в local seed), когда соответствующий package сохранен/опубликован.

Не реализовано сейчас:

- Полный Cloud UI сценарий recipes/stop-list, налоговых профилей и service-charge rules.

## Provisioning

Реализовано сейчас:

Cloud Approve:

1. POS Edge регистрирует устройство через `POST /api/v1/devices/register`.
2. Cloud UI видит незакрепленное устройство через `GET /api/v1/devices/unassigned`.
3. Оператор назначает устройство ресторану через `POST /api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign`.
4. Cloud создает/обновляет assigned edge node и возвращает snapshot URL.
5. POS Edge получает assignment status и одноразовый node token.

License Code:

1. Cloud генерирует node id при необходимости, node token и короткий pairing code.
2. Cloud сохраняет hashes/verifiers и регистрирует pairing code в License Server.
3. Plain pairing code возвращается только в HTTP response Cloud UI.
4. POS Edge отправляет code в License Server через собственный pair-via-license flow.
5. POS Edge получает Cloud URL, restaurant id, node device id, credentials и применяет snapshot.

Правила:

- `assign` и `generate-pairing-code` требуют active restaurant.
- Node token хранится в Cloud как hash/verifier, а не как plaintext.
- `assignment-status` после assigned выдает credentials для Edge provisioning только если token еще не был выдан. Повторная проверка статуса не ротирует существующий `credentials_hash` и не возвращает plaintext token.
- License Server недоступен: Cloud возвращает `503 LICENSE_SERVER_UNAVAILABLE`.

Вне текущего объема:

- Production authorization perimeter для provisioning routes.
- Многошаговое подтверждение владельцем организации.
- UI/API для отзыва node token, кроме текущего schema foundation `revoked`.

## Sync Receiver

Реализовано сейчас:

- `POST /api/v1/sync/exchange` является приоритетным Cloud-Edge циклом.
- Endpoint требует `Authorization: Bearer <node_token>`.
- Cloud проверяет token hash, node device id, restaurant id и status `assigned`.
- `401 SYNC_UNAUTHORIZED` означает отсутствующий/пустой Bearer token, неизвестный node, не assigned/revoked node, пустой Cloud credentials hash или несовпадение hash. Несовпадение restaurant id возвращается как forbidden, а не unauthorized.
- Legacy receive endpoints остаются совместимыми: `POST /sync/edge-events`, `POST /sync/edge-events/batch`.
- Один envelope ограничен 2 MiB; batch/exchange body ограничен 8 MiB; batch содержит от 1 до 100 items.
- Duplicate replay возвращает стабильный ACK и не создает второй receipt.
- Batch ACK содержит item-level statuses.

`sync/exchange`:

- Принимает Edge outbox events.
- Возвращает ACK по каждому client item.
- Сравнивает stream revisions/checkpoints.
- Возвращает новые Cloud packages для запрошенных streams, но не больше `CLOUD_SYNC_MAX_CLOUD_PACKAGES_PER_EXCHANGE` за одну сессию.
- Если Edge revision ahead, request отклоняется до приема Edge events.
- Если revision равен, но checkpoint отличается, request отклоняется до приема Edge events.
- Если package отсутствует, stream result получает `not_found`, HTTP request остается успешным.
- Ошибочные Edge items получают item-level `rejected`/`retryable` и сохраняются в `cloud_sync_problem_events` для анализа, не блокируя остальные items.
- Если transport/auth exchange не завершился успешно, POS Edge не коммитит ACK локально как sent; следующий exchange безопасно повторяет Edge events.

Текущие Edge -> Cloud events:

- `ShiftOpened`
- `ShiftClosed`
- `CashSessionOpened`
- `CashSessionClosed`
- `CashDrawerEventRecorded`
- `OrderCreated`
- `OrderLineAdded`
- `OrderLineQuantityChanged`
- `OrderLineVoided`
- `PrecheckIssued`
- `PrecheckReprinted`
- `PrecheckCancelled`
- `PaymentCaptured`
- `CancellationRecorded`
- `RefundRecorded`
- `CheckCreated`
- `CheckReprinted`
- `OrderClosed`
- `AuthSessionStarted`
- `AuthSessionRevoked`
- `DeviceRegistered`

Inbound compatibility:

- `PaymentRefunded`
- `CheckRefunded`

Эти legacy events принимаются для совместимости старых payloads, но не заполняют detailed `cloud_projection_financial_operations`.

## Financial Operation Projection

Реализовано сейчас:

- Текущие `CancellationRecorded` и `RefundRecorded` проходят строгую validation.
- Payload должен содержать operation id, edge operation id, restaurant id, device id, check id, precheck id, current shift id, original shift id, amount, currency, business date, operation type/kind, inventory disposition, reason и snapshot metadata.
- Payload `restaurant_id` и `device_id` должны совпадать с sync envelope.
- Cloud сохраняет raw payload, receipt и operational journal idempotently.
- Cloud обновляет event-type stats.
- Для refunds Cloud обновляет coarse shift finance refund counters.
- Cloud поддерживает service/repository read model `cloud_projection_financial_operations`.
- Cloud предоставляет bounded read-only HTTP reporting endpoint `GET /api/v1/reporting/financial-operations?restaurant_id=&business_date_from=&business_date_to=&operation_type=&shift_id=&original_shift_id=&check_id=&limit=&offset=` поверх `cloud_projection_financial_operations`; endpoint возвращает operation/check/shift/date/type/disposition/reason metadata и `raw_payload_sha256_hex`, но не возвращает raw sync payload или snapshot JSON.
- `RefundRecorded` и `CancellationRecorded` являются inventory-relevant events только при explicit stock disposition. Cloud receiver ставит их в `inventory_event_queue`, если `inventory_disposition != no_stock_effect`; `no_stock_effect` остается accepted/projection event без queue/ledger. Cloud Inventory Worker обрабатывает operation-level `inventory_disposition`: `return_to_stock` создает `RETURN/IN`, `write_off_waste` создает `WASTE/OUT`, `manual_review` переводит queue item в failure для операторского разбора без stock document. Для автоматического движения Worker нормализует текущие POS ledger `items[]` из immutable operation/check/precheck snapshots.

Вне текущего объема:

- PSP refund execution.
- Fiscal correction documents.
- per-line mixed `inventory_disposition` в одном financial operation payload.

## PostgreSQL Data Contract

Managed SQL file, реализовано сейчас:

- `cloud-backend/migrations/postgres/001_init.sql`

Основные таблицы:

- Sync receiver: `cloud_edge_event_receipts`, `cloud_edge_event_raw_payloads`, `cloud_sync_problem_events`, `cloud_operational_events`.
- Projections: `cloud_projection_event_type_stats`, `cloud_projection_shift_finance`, `cloud_projection_financial_operations`.
- Master-data packages: `cloud_master_data_packages`.
- Currency reference: `cloud_currency_reference`.
- Master data: `cloud_restaurants`, `cloud_roles`, `cloud_employees`, `cloud_categories`, `cloud_catalog_items`, `cloud_dishes`, `cloud_goods`, `cloud_semi_finished_products`, `cloud_services`, `cloud_catalog_folders`, `cloud_catalog_folder_parameters`, `cloud_catalog_tags`, `cloud_catalog_item_tags`, `cloud_recipe_items`, `cloud_recipe_versions`, `cloud_recipe_lines`, `cloud_modifier_groups`, `cloud_modifier_options` с nullable `linked_catalog_item_id`, `cloud_modifier_group_bindings`, `cloud_pricing_policies`, `cloud_menu_items`, `cloud_menu_item_modifier_groups`, `cloud_menu_location_assignments`, `cloud_master_data_publications`, `cloud_review_assignment_audit_events`.
- Provisioning: `cloud_edge_nodes`, `cloud_unassigned_edge_nodes`, `cloud_pairing_codes`.
- Inventory runtime: `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists`.
- OLAP control state: `olap_export_checkpoints`, `olap_export_retry_commands`.

Schema verification:

- Проверяет required receiver/journal/projection/package/currency/master-data/provisioning/inventory foundation tables and columns.
- Missing table/column должен обнаруживаться на startup, а не в середине request path.

## Inventory And Analytics Boundaries

Реализована только основа:

- PostgreSQL baseline содержит Cloud inventory runtime tables.
- `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists` являются целевой Cloud-owned основой.

Реализовано сейчас:

- Cloud Inventory Worker создает stock documents and stock ledger из accepted normalized item events.
- `GET /api/v1/inventory/stock-ledger` возвращает bounded read-only rows из Cloud-owned `stock_ledger` для smoke/операционной проверки `CheckClosed`/`ItemServed` processing; endpoint не раскрывает raw sync payload и не является OLAP API.
- `GET /api/v1/inventory/stock-balances` реализовано сейчас как bounded read-only balance view поверх Cloud-owned PostgreSQL `stock_ledger`: отрицательные остатки допустимы, sale blocking не использует stock balance, aggregate costing status ограничен `final`, `estimated`, `needs_recalculation`, `mixed`, `unknown`.
- OLAP Stock Moves Forwarder асинхронно экспортирует новые `stock_ledger` rows в ClickHouse `olap_stock_moves` по checkpoint `olap_export_checkpoints.id = 'olap_stock_moves'`; retry state хранится в той же checkpoint table через `last_error`, `consecutive_failures` и `next_retry_at`.
- Async OLAP Backfill Worker выполняет jobs из `olap_backfill_jobs` вне HTTP request path: для `raw_business_events` переэкспортирует выбранный range из PostgreSQL `inbox_events`, для `stock_moves` переэкспортирует range из `stock_ledger`; ClickHouse `ReplacingMergeTree` и stable row ids защищают read model от видимых дублей при повторном backfill.
- `GET /api/v1/olap/stock-moves` возвращает bounded read-only rows из ClickHouse `olap_stock_moves` с фильтрами `restaurant_id`, business date range, `catalog_item_id`, `warehouse_id`, `source_event_type`, `limit`, `offset`; response не содержит raw payload.
- `GET /api/v1/olap/export-status` реализовано сейчас как read-only observability над `olap_export_checkpoints`, `inbox_events` и `stock_ledger`: response содержит stream, checkpoint, last exported id/time, counters, last error metadata, consecutive failures, next retry и retry_blocked без raw payload.
- `POST /api/v1/olap/export-retry` реализовано сейчас как минимальный support-only mutating control: идемпотентность хранится в `olap_export_retry_commands`, `retry_failed` переводит failed `inbox_events` обратно в pending для `raw_business_events`, `resume_from_checkpoint` дополнительно снимает processing locks, а для `stock_moves` снимается checkpoint backoff; endpoint не сбрасывает checkpoint вручную, не пишет ClickHouse business rows и не подключен к Cloud UI.
- `GET /api/v1/olap/stock-move-summary` реализовано сейчас как первый bounded агрегированный ClickHouse read: фильтры совпадают с stock moves, `group_by` ограничен `business_date`, `catalog_item`, `warehouse`, ordering deterministic, response содержит quantities/cost totals без COGS/margin wording.
- `GET /api/v1/olap/sales-kitchen-summary` реализовано сейчас как первый bounded sales/kitchen aggregate: фильтры `restaurant_id`, `business_date_from`, `business_date_to`, `limit`, `offset`, `group_by=business_date|event_type|source_event_type|catalog_item`; endpoint читает существующие `raw_business_events` и `olap_stock_moves`, не выбирает raw payload, не добавляет новые ClickHouse tables/materialized views и не пишет ClickHouse из request path.
- `GET /api/v1/olap/kitchen-timing-summary` реализовано сейчас как bounded kitchen timing API: группировки `business_date|station`, фильтр `station_id`, lifecycle counts и средние секунды `accepted -> ready`, `in_progress -> ready`, `ready -> served`; endpoint читает только confirmed event streams без raw payload.
- `GET/POST /api/v1/olap/backfill-jobs` и `POST /api/v1/olap/backfill-jobs/{id}/cancel` реализованы сейчас как foundation для operator workflow: jobs имеют checkpoint/progress/status/error metadata, idempotency по UUIDv7 `command_id`, audit trail `olap_operator_audit_events`, bounded list/get и background execution без synchronous ClickHouse write в HTTP path.
- `CatalogItemChangeSuggested` создает Cloud review item; upsert в catalog выполняется только после manager approve текущими `catalog-suggestions` routes.
- `RecipeChangeSuggested` создает Cloud review item с diff по ingredients, quantities, units, loss percent и prep time; published recipe не меняется до approve/apply текущими `recipe-suggestions` routes. Реализовано сейчас: Cloud-authored recipe version draft при submit создает pending `RecipeChangeSuggested` с `action = publish_recipe_version`; approve активирует draft version, архивирует предыдущую active version для owner item и публикует `recipes` package.
- Реализовано сейчас: Edge-origin stop-list review item для `StopListUpdated` поддерживает назначение на менеджера и снятие назначения через `POST /api/v1/manager/stop-list-updates/{id}/assign|unassign`. Assignment state хранится в `cloud_projection_stop_list_updates` (`assigned_to_employee_id`, `assigned_by_employee_id`, `assigned_at`, `assignment_note`), а действия `assigned`/`unassigned` пишутся в append-only `cloud_review_assignment_audit_events` с `event_id` UUIDv7, `review_id`, `actor_employee_id`, `target_employee_id`, `reason` и `occurred_at`. Реализовано сейчас: bounded чтение assignment audit доступно для `stop_list_update`, `catalog_suggestion` и `recipe_suggestion` через `GET /api/v1/manager/stop-list-updates/{id}/audit?limit=&offset=`, `GET /api/v1/manager/catalog-suggestions/{id}/audit?limit=&offset=` и `GET /api/v1/manager/recipe-suggestions/{id}/audit?limit=&offset=` без raw payload; unknown review id возвращает safe empty list. Assignment runtime для catalog/recipe запланирован далее и не заявлен как реализованный. Escalation/dashboard запланированы далее. Raw payload exposure вне текущего объема и запрещено.
- Canonical seed/smoke для Cloud-owned сценариев находится только в `scripts/seed-dev-system.py`: при добавлении Cloud-owned справочника, review flow, publication stream/package или bounded POS read flow тем же PR обновляются seed dataset, publication assertion, smoke assertion/read check, script guard `CLOUD_OWNED_SEED_SURFACES` и профильная документация. Seed/smoke работает через HTTP API и не пишет напрямую в PostgreSQL/ClickHouse.
- `StopListUpdated` обрабатывается асинхронно через `inventory_event_queue`: worker пишет `cloud_projection_stop_list_updates` без raw payload. `edge_overlay_until_next_publication` обновляет bounded `stop_lists` overlay, `cloud_wins` не перетирает Cloud-owned row, `edge_overlay_requires_manager_review` фиксирует безопасную projection для bounded manager review.
- Bounded manager review для `edge_overlay_requires_manager_review` реализован сейчас: list/detail имеют stable bounded paging, decisions идемпотентны, invalid transition возвращает conflict, approve применяет изменение только через Cloud-owned `stop_lists` + publication, reject/request-changes не меняют runtime stop-list authority.
- `GET /api/v1/sync/readiness/stop-list` реализовано сейчас как safe readiness summary: publication/package metadata, latest accepted `StopListUpdated` ACK metadata и агрегат `cloud_sync_problem_events` по кодам ошибок без raw payload.
- Отдельный safe read-only route для package delivery status по restaurant/device/package сейчас не подтвержден кодом; доступные package/snapshot routes возвращают provisioning payload, а `sync/exchange` возвращает Cloud packages для Edge import. Cloud UI может показывать только `publication-state` и stop-list readiness ACK metadata, но не должен заявлять общий package delivery ACK.
- `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured` и `ProductionCompleted` создают Cloud-owned stock documents/ledger rows.
- `KitchenTicketStatusChanged` и `ItemServed` используются для kitchen timing и inventory deduplication, но не меняют finalized checks.
- Если Cloud уже принял superseding `ItemServed` для той же order line до обработки очереди, Inventory Worker пропускает superseded served fact.
- Если старый `ItemServed` уже создал stock document до recall/serve-again, superseding `ItemServed` создает append-only `RETURN/IN` compensation document с `source_event_type = ItemServedCompensation`, затем новый `SALE/OUT` document. Replay защищен unique `(source_event_id, source_event_type)`, raw Edge payload в read APIs не раскрывается.
- ClickHouse `raw_business_events` реализовано сейчас как бессрочный архив business events.
- Async Batch Forwarder переносит accepted events из PostgreSQL `inbox_events` в ClickHouse и после successful export выставляет `processed_for_olap = true`.
- ClickHouse `olap_stock_moves` реализовано сейчас как первый bounded read model для складских движений; он не является source of truth и наполняется только async export из PostgreSQL `stock_ledger`.
- `GET /api/v1/olap/raw-business-events`, `GET /api/v1/olap/stock-moves`, `GET /api/v1/olap/export-status`, `GET /api/v1/olap/stock-move-summary`, `GET /api/v1/olap/sales-kitchen-summary`, `GET /api/v1/olap/kitchen-timing-summary` и bounded backfill job endpoints реализованы сейчас без raw payload.

Запланировано до полного пилота:

- Cloud authoring/publication workflow для stop-list/recipes становится штатным источником sale-blocking availability overlay; POS Edge runtime уже блокирует продажи по локальному `stop_lists`.
- Receipt line с pending catalog suggestion остается запланировано далее.
- Полный materialized balance engine, semi-finished auto-production split и retro costing DAG остаются частью дальнейшего Cloud Inventory Engine; bounded `stock-balances` read из `stock_ledger`, recipe expansion основной продажи и modifier linked catalog item consumption реализованы сейчас.
- Расширенные sales/kitchen/costing-dependent projections beyond first bounded sales/kitchen summary запланированы далее.
- Расширенный manager review workflow для Edge-origin stop-list изменений остается запланирован далее; текущий runtime уже имеет bounded review/apply, assignment/unassignment и audit без raw payload, но без escalation/dashboard workflow.

Вне текущего объема:

- Synchronous dual-write PostgreSQL + ClickHouse.

## Cloud UI Boundary

Реализовано сейчас:

- Активный Cloud UI target — `cloud-ui-g`; устаревший Vue/Quasar `cloud-ui` остается legacy/reference-only и не принимает новые Cloud-бэкофисные фичи.
- `cloud-ui-g` использует Cloud Backend routes для launch readiness, Edge-device flow, master data, publication и safe Edge events list.
- Legacy `cloud-ui` читает `GET /api/v1/sync/readiness/stop-list` в inventory readiness panel и показывает только counts/status/checkpoint/ACK metadata без raw sync payload.
- Legacy `cloud-ui` читает bounded `GET /api/v1/manager/stop-list-updates` и вызывает approve/reject/request-changes для safe Edge-origin stop-list review; raw Edge payload не рендерится.
- Реализовано сейчас: backend assignment routes доступны только для `stop_list_update` через `POST /api/v1/manager/stop-list-updates/{id}/assign|unassign`; bounded audit read доступен для `stop_list_update`, `catalog_suggestion` и `recipe_suggestion` через manager audit routes; assignment runtime для `catalog_suggestion` и `recipe_suggestion`, escalation и dashboard запланированы далее.
- Cloud UI не использует POS session, POS Edge runtime endpoints или cashier stores.
- Cloud UI не показывает raw payloads, PIN material, token material или sensitive request dumps.
- Cloud UI работает в local pilot perimeter через CORS origins `5174`.

Вне текущего объема полного пилота:

- Cashier runtime в Cloud UI.
- Cloud auth/RBAC UI.
- KDS runtime screens в Cloud UI.
- PSP, fiscalization, delivery и advanced procurement planning screens beyond pilot stock receipt/count/production input.

## Тестовое покрытие

Реализовано сейчас:

- Cloud sync API tests: duplicate envelope, batch ACK, authenticated exchange, provisioning package read/write, CORS, safe Edge events list.
- Cloud sync service tests: idempotent receive, item-level ACK, exchange packages, revision conflicts, current/legacy refund boundaries, master-data package validation, `StopListUpdated` replay queue idempotency и readiness no-raw-payload contract.
- POS syncsender tests: temporary `sync/exchange` failure, retryable outbox state, повторная отправка до item-level ACK и прекращение pending resend после ACK.
- Cloud sync contract tests: idempotency key, supported event catalog, financial operation payload validation, identity mismatch rejection.
- Cloud master-data API tests: employee responses without PIN material, publication summary, `200 null` before first publication, production publication/snapshot endpoints.
- Cloud master-data service tests: restaurant CRUD, employee lifecycle, PIN uniqueness, permission validation, catalog/menu publication shape, service/semi-finished kinds, lifecycle status preservation, pricing policy validation.
- Cloud PostgreSQL tests: migration ordering, checksum behavior, runtime version policy, schema repair, schema verification errors.
- Cloud OLAP tests: bounded read validation, read-only export status, export-retry validation/API no-payload contract, stock move summary and sales-kitchen summary grouping and forwarder retry behavior.
- Cloud schema tests: currency reference, inventory foundation, `cloud_projection_stop_list_updates`, financial operation projection, OLAP checkpoint/retry command tables.

## Запланировано далее

- Production authorization and tenant perimeter для Cloud API.
- До полного пилота: расширить bounded Edge-origin stop-list manager review до production workflow, full recipe/costing inventory engine, richer sales/kitchen/costing OLAP API beyond first bounded endpoint, production-grade OLAP backfill jobs/operator UI и расширенная observability UI.
- После полного пилота: rich BI/reporting UI beyond pilot financial operations and OLAP API.
- Data-preserving PostgreSQL migrations после первого реального внедрения.
- Cloud UI сценарий налогов/service-charge rules, если пилот требует централизованное управление.

## Вне текущего объема полного пилота

- POS cashier runtime commands в Cloud Backend.
- Payment processor and fiscal adapter.
- POS-side KDS runtime screens and hardware bump-bar/printer integrations.
- Delivery/channel integrations.
- ClickHouse как transactional source of truth.
- Manual ad-hoc SQL repair as canonical path.
- Раскрытие PIN/token/raw payload в UI-facing API.
