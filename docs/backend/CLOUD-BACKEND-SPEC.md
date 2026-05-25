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
- публичный Cloud reporting UI beyond pilot OLAP API;

Запланировано до полного пилота:

- Cloud authoring/publication workflow для streams `recipes` и `inventory_reference` поверх Cloud authority tables;
- projection update для `StopListUpdated` из Edge/Cloud без raw payload exposure;
- review/apply очереди для `CatalogItemChangeSuggested` и `RecipeChangeSuggested`, созданных kitchen worker на Edge;
- обработка `KitchenTicketStatusChanged`, `StockReceiptCaptured`, `CatalogItemChangeSuggested`, `RecipeChangeSuggested` и `StopListUpdated` как business events без synchronous apply в request path;
- параметр `stop_list_conflict_policy` для порядка применения Cloud-authored stop-list и Edge overlay;
- readiness API/UI signals для stop-list publication, Edge ACK и sync problem events;
- поддержка `CheckClosed`/`ItemServed` как pilot inventory facts через текущий receiver и Inventory Worker;
- full inventory engine для receipts, counts, production, consumption, refund/cancellation dispositions, balances, costing и retro recalculation;
- ClickHouse runtime: async forwarder, `raw_business_events`, `olap_stock_moves`, retry/backfill/export checkpoints;
- bounded read-only Cloud OLAP API для event archive, stock moves, sales aggregates, COGS/margin и kitchen timing.

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
- Cloud не использует ClickHouse в текущем runtime, но ClickHouse является обязательным компонентом полного пилота.
- Cloud пока не предоставляет pilot-ready CRUD/publication UI для recipes/stop-list; generic package storage/contracts уже принимают `recipes` и `inventory_reference`.
- Cloud пока не предоставляет review/apply runtime для Edge-originated catalog/recipe proposals.

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
- `POST /api/v1/sync/edge-events`
- `POST /api/v1/sync/edge-events/batch`
- `POST /api/v1/sync/exchange`

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
- Запланировано до полного пилота:
  - `POST /api/v1/master-data/recipes`
  - `GET /api/v1/master-data/recipes`
  - `PATCH /api/v1/master-data/recipes/{id}`
  - `GET /api/v1/master-data/recipe-change-suggestions`
  - `POST /api/v1/master-data/recipe-change-suggestions/{id}/approve`
  - `POST /api/v1/master-data/recipe-change-suggestions/{id}/reject`
  - `GET /api/v1/master-data/catalog-suggestions`
  - `POST /api/v1/master-data/catalog-suggestions/{id}/approve`
  - `POST /api/v1/master-data/catalog-suggestions/{id}/reject`
  - `POST /api/v1/master-data/stop-lists`
  - `GET /api/v1/master-data/stop-lists`
  - `PATCH /api/v1/master-data/stop-lists/{id}`
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
- Cloud UI после successful master-data CRUD может автоматически вызвать publication API; ручная публикация остается operator checkpoint.
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
- `modifier_options[]` передает `id`, `restaurant_id`, `modifier_group_id`, `name`, `price_minor`, `active`.
- `modifier_bindings[]` передает `id`, `restaurant_id`, `modifier_group_id`, `target_type`, `target_id`, `sort_order`, `active`.
- `pricing_policy` stream передает `tax_profiles`, `tax_rules`, `service_charge_rules`, `pricing_policies`, когда соответствующий package сохранен/опубликован.
- `recipes` stream передает `recipe_versions` и `recipe_lines`, когда соответствующий package сохранен/опубликован.
- `inventory_reference` stream передает `stop_lists`, когда соответствующий package сохранен/опубликован.

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
- `assignment-status` после assigned может выдать credentials для Edge provisioning; UI не должен показывать token.
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

Вне текущего объема:

- Публичный HTTP reporting endpoint для `cloud_projection_financial_operations`.
- Cloud UI reporting screen.
- PSP refund execution.
- Fiscal correction documents.
- Automatic inventory movement based on `inventory_disposition`.

## PostgreSQL Data Contract

Managed SQL file, реализовано сейчас:

- `cloud-backend/migrations/postgres/001_init.sql`

Основные таблицы:

- Sync receiver: `cloud_edge_event_receipts`, `cloud_edge_event_raw_payloads`, `cloud_sync_problem_events`, `cloud_operational_events`.
- Projections: `cloud_projection_event_type_stats`, `cloud_projection_shift_finance`, `cloud_projection_financial_operations`.
- Master-data packages: `cloud_master_data_packages`.
- Currency reference: `cloud_currency_reference`.
- Master data: `cloud_restaurants`, `cloud_roles`, `cloud_employees`, `cloud_categories`, `cloud_catalog_items`, `cloud_dishes`, `cloud_goods`, `cloud_semi_finished_products`, `cloud_services`, `cloud_catalog_folders`, `cloud_catalog_folder_parameters`, `cloud_catalog_tags`, `cloud_catalog_item_tags`, `cloud_recipe_items`, `cloud_modifier_groups`, `cloud_modifier_options`, `cloud_modifier_group_bindings`, `cloud_pricing_policies`, `cloud_menu_items`, `cloud_menu_item_modifier_groups`, `cloud_menu_location_assignments`, `cloud_master_data_publications`.
- Provisioning: `cloud_edge_nodes`, `cloud_unassigned_edge_nodes`, `cloud_pairing_codes`.
- Inventory runtime: `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists`.

Schema verification:

- Проверяет required receiver/journal/projection/package/currency/master-data/provisioning/inventory foundation tables and columns.
- Missing table/column должен обнаруживаться на startup, а не в середине request path.

## Inventory And Analytics Boundaries

Реализована только основа:

- PostgreSQL baseline содержит Cloud inventory runtime tables.
- `inventory_event_queue`, `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists` являются целевой Cloud-owned основой.

Реализовано сейчас:

- Cloud Inventory Worker создает stock documents and stock ledger из accepted normalized item events.

Запланировано до полного пилота:

- Cloud authoring/publication workflow для stop-list/recipes становится штатным источником sale-blocking availability overlay; POS Edge runtime уже блокирует продажи по локальному `stop_lists`.
- `CatalogItemChangeSuggested` создает Cloud review item; upsert в catalog разрешен только при policy `auto_apply_catalog_suggestions = true` или после manager approve.
- `RecipeChangeSuggested` создает Cloud review item с diff по ingredients, quantities, units, loss percent и prep time; published recipe не меняется до approve/apply.
- `StockReceiptCaptured` создает Cloud-owned receipt document и может ссылаться на pending catalog suggestion, если товар еще не утвержден.
- `KitchenTicketStatusChanged` и `ItemServed` используются для kitchen timing и inventory deduplication, но не меняют finalized checks.
- ClickHouse `raw_business_events` становится бессрочным архивом business events.
- Async Batch Forwarder переносит accepted events из PostgreSQL inbox buffer в ClickHouse.
- Recipe expansion, modifier linked catalog item consumption, stock balances and retro costing DAG становятся частью Cloud Inventory Engine.
- Cloud OLAP API читает bounded aggregates из ClickHouse projections.

Вне текущего объема:

- Synchronous dual-write PostgreSQL + ClickHouse.

## Cloud UI Boundary

Реализовано сейчас:

- Cloud UI использует Cloud Backend routes для launch readiness, Edge-device flow, master data, publication и safe Edge events list.
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
- Cloud sync service tests: idempotent receive, item-level ACK, exchange packages, revision conflicts, current/legacy refund boundaries, master-data package validation.
- Cloud sync contract tests: idempotency key, supported event catalog, financial operation payload validation, identity mismatch rejection.
- Cloud master-data API tests: employee responses without PIN material, publication summary, `200 null` before first publication, production publication/snapshot endpoints.
- Cloud master-data service tests: restaurant CRUD, employee lifecycle, PIN uniqueness, permission validation, catalog/menu publication shape, service/semi-finished kinds, lifecycle status preservation, pricing policy validation.
- Cloud PostgreSQL tests: migration ordering, checksum behavior, runtime version policy, schema repair, schema verification errors.
- Cloud schema tests: currency reference, inventory foundation, financial operation projection.

## Запланировано далее

- Production authorization and tenant perimeter для Cloud API.
- До полного пилота: recipes/stop-list authoring UI, deterministic publication from Cloud authority tables, Edge-origin stop-list sync/conflict policy, full recipe/costing inventory engine, ClickHouse async forwarder, `olap_stock_moves` export, OLAP API и readiness/observability UI.
- После полного пилота: Public Cloud reporting UI beyond pilot OLAP API.
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
