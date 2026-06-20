# Текущее функциональное состояние проекта

Статус: реализовано сейчас по коду, тестам и документации на 2026-06-20; цели полного и выставочного альфа-пилотов зафиксированы отдельно и не считаются текущим runtime.

Этот документ является сводной картой фактического состояния репозитория. Он не заменяет профильные спецификации: архитектурные инварианты остаются в `SPECv1.3.md`, backend-контракты - в `docs/backend/*`, контракты интерфейсов - в `docs/ui/*`, синхронизация - в `docs/sync/*`.

Источник истины для реализованного runtime: код, миграции и тесты. Если этот документ конфликтует с кодом или тестами, сначала фиксируется фактическое поведение по коду, затем обновляется документация.

## Найдено при ревью

Реализовано сейчас:

- Документация уже описывает ключевой кассовый поток `Order -> Precheck -> Payment -> Check`, предчеки, оплаты, итоговые чеки, возвраты/отмены, синхронизацию и локальный smoke path.
- Основной пробел был не в отдельных POS-инвариантах, а в отсутствии единой русскоязычной сводки по всему репозиторию: POS Edge, Cloud Backend, License Server, POS UI, Cloud UI, миграции, скрипты и тестовое покрытие были описаны в разных документах.
- Кодовая база содержит больше подтвержденных Cloud и provisioning маршрутов, чем явно видно из POS-ориентированных документов; они зафиксированы ниже как реализованная сейчас функциональность.

Не обнаружено сейчас:

- Подтвержденного runtime для delivery, настоящего платежного процессинга, фискального адаптера, COGS/margin OLAP reads, production BI, production-grade kitchen timing API и расширенных cooking events за пределами ticket lifecycle foundation.
- Автоматическая Cloud -> Edge доставка без manual publish, QR-enabled ticket issuance, physical ESC/POS printing, Telegram reports и sales dashboard пока не реализованы. External License Server, versioned entitlement snapshots и backend gates для существующих table/kitchen/warehouse surfaces реализованы сейчас; будущие telegram/waiter/checker runtime получают gates вместе со своими routes/workers. Tenant-level roles/employees, employee restaurant memberships, `organization.manage`, tenant catalog identity и restaurant menu overrides реализованы сейчас. Master-data CRUD не обновляет Edge package автоматически, Cloud UI вызывает manual publish route; reprint возвращает snapshot без printer orchestration. QR checker/enrollment/relay/confirm перенесены в post-deploy цикл. Целевой контракт описан в `docs/project-management/EXHIBITION-ALPHA-PILOT-REQUIREMENTS.md`.

Цель полной пилотной реализации:

- сохранить текущий cashier runtime как базовый поток;
- stop-list sale blocking на POS Edge с Cloud authoring/publication и offline локальной проверкой уже подтвержден в `pos_stop_list_sale_blocking`;
- расширять mobile-first waiter runtime без payment/refund authority по умолчанию только по подтвержденным backend contracts;
- расширить KDS lifecycle за пределы текущего backend-backed ticket/stock/proposal/stop-list foundation: cooking events, station priority и operator analytics;
- зафиксировать POS Edge backend как авторитетный runtime для financial/order/KDS command validation и stop-list sale blocking; POS UI не становится авторитетным слоем;
- добавить Cloud manager flow для production-grade recipe lifecycle polish, stop-list escalation polish, inventory operations, publication readiness и sync/problem observability;
- добавить полный Cloud-owned складской движок beyond текущего bounded worker/materialized-balance/recalculation slice: production-grade stock receipts/counts/production state, semi-finished auto-production split, richer costing math и production-grade balance rebuild; bounded materialized balances, retro recalculation job/DAG lifecycle и refund/cancellation dispositions уже реализованы Cloud-side для `return_to_stock`/`write_off_waste`;
- расширить ClickHouse runtime от первых bounded `stock-move-summary`/`sales-kitchen-summary` endpoints и минимального retry control до richer sales/kitchen/costing aggregates и production-grade backfill jobs;
- поддерживать полный smoke path Cloud setup -> Edge sync -> waiter order/precheck -> KDS served -> cashier payment/final check -> Edge outbox -> Cloud inventory ledger -> ClickHouse export -> bounded OLAP API; сейчас полный хвост после финального чека покрывает `scripts/seed-dev-system.py --run-minimal-flow`, а advanced kitchen/process ветку покрывает `--run-kitchen-process-smoke`.

## POS Edge Backend

Реализовано сейчас:

- HTTP API на `chi` с безопасным JSON error contract, request id, структурированным audit log и CORS для локального POS UI.
- PIN-вход, backend-сессии, привязка `node_device_id` / `client_device_id`, проверка actor context и rate limit для PIN.
- RBAC на уровне application services; UI visibility не является границей безопасности.
- Личная смена сотрудника и кассовая смена устройства как разные runtime-понятия.
- Залы, столы, меню и каталог как локальные read models, получаемые из Cloud-owned справочников. Catalog хранит tenant-level `catalog_item_id`, menu хранит restaurant-effective `menu_item_id`, overrides name/price/tag/tax/menu folder/availability/runtime status и сохраняет stable category/tag identity downstream в order lines.
- Создание заказов, чтение текущего/активных/закрытых заказов, добавление строк, изменение количества, списание строки, курс подачи и комментарий строки.
- Выбор и редактирование модификаторов активной строки заказа; backend проверяет активность группы/опции, связь с menu item, required/min/max и цену.
- Backend authoritative pricing: скидки, надбавки, automatic policies из Cloud, единый порядок применения по `application_index`, налог последним шагом и целочисленное округление.
- Выпуск предчека с immutable snapshot и блокировкой заказа.
- Отмена unpaid active предчека через manager override с PIN и правами.
- Оплаты по `precheck_id`, частичные оплаты, итоговый чек только после полной оплаты.
- Повторная печать предчека и итогового чека из immutable snapshot.
- Append-only ledger финансовых операций: `CancellationRecorded` и `RefundRecorded` для полных и частичных операций, без мутации уже финализированных payment/precheck/check.
- Compatibility route `POST /api/v1/payments/{id}/refund`, который записывает refund operation по payment allocation, но не возвращает оплату или чек в изменяемое состояние.
- Ограниченные read endpoints для закрытых заказов, financial operations, outbox и local events.
- Backend-backed KDS ticket lifecycle: `kitchen_tickets` создаются из non-service order lines, `GET /api/v1/kitchen/order-queue` и `GET /api/v1/kitchen/tickets` поддерживают bounded read/status filter, status actions `accept/start/hold/ready/serve/recall/cancel` проверяют `pos.kitchen.status.change`, пишут `KitchenTicketStatusChanged`, а `serve` дополнительно пишет `ItemServed`; повторная подача после recall пишет новый `ItemServed` с `serve_sequence` и `supersedes_served_event_id`.
- Kitchen stock/proposal/stop-list runtime: POS Edge принимает receipt/count/write-off/production, recipe read, catalog/recipe suggestions, stop-list update commands, safe stop-list state read для UI indicator и proposal feedback read model без создания Edge-side stock documents.
- Локальный lifecycle SQLite: status, retention dry-run, archive export plan, export-only JSONL archive, read-plan, lookup preview, apply-plan и apply-readiness с поддержкой destructive apply (физическое удаление закрытых orders/checks/financial_operations и связанных при verified JSONL + чистый scoped outbox + отсутствие открытых operational boundaries для cutoff периода) и последующий VACUUM compaction БД.
- Cloud -> Edge master-data ingest для `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`. `menu` принимает restaurant-effective `category_id`, `tag_id`, `tax_profile_id` и `runtime_status`, а `catalog` принимает tenant-level folders/tags/item-tag links без обязательного restaurant binding.
- Sync sender через authenticated `sync/exchange`, item-level ACK, retry/reclaim/backoff и безопасную обработку неподдержанных направлений.
- Cloud-centric Inventory foundation: Cloud sync receiver принимает целевые складские события, durable `inventory_event_queue` передает их Cloud Inventory Worker, POS Edge legacy manual stock foundation удален из runtime.

Вне текущего объема:

- Recipe-based автоматическое списание склада из продажи.
- Полный ретроспективный расчет себестоимости.
- Настоящий платежный процессинг, webhooks, фискальные смены и фискальный адаптер.

## Cloud Backend

Реализовано сейчас:

- HTTP API на `chi` с локальным CORS для Cloud UI и структурированным request log.
- Прием Edge events: `POST /api/v1/sync/edge-events`, batch прием и `POST /api/v1/sync/exchange`.
- `sync/exchange` проверяет bearer `node_token`, assigned restaurant и device status.
- Idempotent receipt для Edge events, raw payload checksum, event type stats и coarse shift finance projection.
- Bounded read-only Cloud inventory ledger endpoint `GET /api/v1/inventory/stock-ledger` для проверки обработанных Cloud Inventory Worker строк без raw sync payload.
- Bounded read-only Cloud inventory balances endpoint `GET /api/v1/inventory/stock-balances` поверх PostgreSQL `inventory_stock_balances`: фильтры по ресторану/складу/товару/UTC-дате `last_movement_at`/costing status, отрицательные остатки допустимы, deterministic `costing_status` и `needs_recalculation` видны без raw payload, COGS или margin.
- Bounded read-only Cloud inventory recalculation endpoints `GET /api/v1/inventory/recalculation-jobs` и `GET /api/v1/inventory/recalculation-jobs/{id}` показывают async costing job status/progress/safe failure metadata без raw payload и без mutating retry/cancel controls.
- `ItemServed` попадает в durable `inventory_event_queue` и Cloud Inventory Worker создает sale ledger идемпотентно по source event; superseded served fact пропускается, если superseding `ItemServed` уже принят до обработки очереди; если старый served fact уже обработан, superseding `ItemServed` пишет append-only `ItemServedCompensation` return ledger и затем новый sale ledger; `KitchenTicketStatusChanged` принимается как operational-only event и не ставится в inventory queue.
- `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured` и `ProductionCompleted` принимаются Cloud receiver и превращаются Cloud Inventory Worker в stock documents/ledger rows.
- Для `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured` и `ProductionCompleted` реализовано сейчас отдельное Cloud-состояние обработки `inventory_document_processing_state`: повтор защищен уникальным `(restaurant_id, source_event_id, source_event_type)`, успешный posting транзакционно согласован со stock document/ledger/balance, безопасная validation failure фиксируется как `failed` с безопасными `failure_code`/`failure_message_key`, raw Edge payload наружу и в state table не попадает.
- `InventoryCountCaptured` реализовано сейчас как детерминированная корректировка к текущему Cloud materialized balance: `IN`, `OUT` или no-op state `posted` с `posted_ledger_count = 0`; отрицательный текущий balance допустим.
- `ProductionCompleted` реализовано сейчас как `PRODUCTION/IN` для готовой/полуготовой позиции и, при наличии active recipe, ingredient `OUT`; при отсутствии recipe/cost basis worker сохраняет факт production с `estimated`/`needs_recalculation`, а bounded retro recalculation worker позднее пересчитывает affected costing fields при появлении backdated trigger и reliable basis.
- Ограниченный Inventory Engine v2 retro recalculation DAG реализовано сейчас: backdated receipt/count/production/write-off создает idempotent job только при affected future ledger rows, хранит catalog/warehouse/unit date ranges и recipe dependency edges, выполняется background worker-ом вне HTTP request path, cycles переводит в safe failed job, а business facts/source events/stock documents/quantities не переписывает.
- `RefundRecorded` и `CancellationRecorded` принимаются Cloud receiver как append-only financial operation facts; `no_stock_effect` не создает складского эффекта, `return_to_stock` и `write_off_waste` попадают в `inventory_event_queue` и асинхронно создают Cloud-owned `RETURN/IN` или `WASTE/OUT`, а `manual_review` не создает автоматическое движение и остается failed queue item для операторского разбора.
- Financial cancellation/refund disposition использует immutable operation/check/precheck snapshots; POS Edge не создает stock documents, stock ledger, balances или costing state, а sale ledger не переписывается.
- `StopListUpdated` принимается Cloud receiver-ом, ставится в durable `inventory_event_queue` и обрабатывается Cloud Inventory Worker в `cloud_projection_stop_list_updates` без raw payload; минимальный `stop_list_conflict_policy` поддерживает `cloud_wins`, `edge_overlay_until_next_publication`, `edge_overlay_requires_manager_review`, default `edge_overlay_requires_manager_review`.
- Bounded Cloud manager review для Edge-origin stop-list updates реализован через `GET /api/v1/manager/stop-list-updates`, detail и `approve/reject/request-changes`: API отдает только safe summary/diff, approve применяет изменение через Cloud-owned authority/publication path, reject/request-changes не мутируют runtime stop-list authority.
- `GET /api/v1/sync/readiness/stop-list` возвращает safe readiness по stop-list publication/package, latest accepted Edge ACK metadata и sync problem counters без raw payload.
- Детальная PostgreSQL projection для current `CancellationRecorded` и `RefundRecorded`; legacy `PaymentRefunded`/`CheckRefunded` принимаются, но не наполняют detailed operation projection.
- Bounded read-only Cloud reporting endpoint `GET /api/v1/reporting/financial-operations` читает detailed financial operation projection с фильтрами restaurant/date/type/shift/original shift/check, `limit`/`offset`, без raw sync payload, snapshot JSON и cashier mutations.
- Безопасный список входящих Edge events для Cloud UI без raw payload.
- PostgreSQL `inbox_events` как transactional delivery queue для accepted Edge events; Cloud API отвечает после PostgreSQL commit и не пишет в ClickHouse в request path.
- ClickHouse managed schema для `raw_business_events`, async forwarder `inbox_events -> raw_business_events`, `processed_for_olap`, retry/backoff state и checkpoint table `olap_export_checkpoints`.
- Bounded read-only metadata endpoint `GET /api/v1/olap/raw-business-events` читает ClickHouse без раскрытия raw payload.
- ClickHouse managed schema для `olap_stock_moves`, async forwarder `stock_ledger -> olap_stock_moves` с checkpoint/retry state и bounded endpoint `GET /api/v1/olap/stock-moves?restaurant_id=&business_date_from=&business_date_to=&catalog_item_id=&warehouse_id=&source_event_type=&limit=&offset=` без raw sync payload.
- Read-only OLAP export status endpoint `GET /api/v1/olap/export-status?stream=raw_business_events|stock_moves` возвращает checkpoint, pending/processing/failed counters, last error metadata и retry/backoff state без raw payload.
- Минимальный support-only mutating control `POST /api/v1/olap/export-retry` принимает `command_id` UUIDv7, `stream=raw_business_events|stock_moves`, `mode=retry_failed|resume_from_checkpoint` и `reason`, идемпотентно снимает retry/backoff state в PostgreSQL, не возвращает raw payload/reason и не пишет business rows в ClickHouse.
- Первый bounded агрегат `GET /api/v1/olap/stock-move-summary?restaurant_id=&business_date_from=&business_date_to=&catalog_item_id=&warehouse_id=&source_event_type=&group_by=business_date|catalog_item|warehouse&limit=&offset=` читает ClickHouse `olap_stock_moves` без raw payload и не является COGS/margin расчетом.
- Первый bounded sales/kitchen агрегат `GET /api/v1/olap/sales-kitchen-summary?restaurant_id=&business_date_from=&business_date_to=&group_by=business_date|event_type|source_event_type|catalog_item&limit=&offset=` читает ClickHouse `raw_business_events` и `olap_stock_moves` без raw payload, без synchronous ClickHouse write и без COGS/margin расчета.
- Bounded kitchen timing агрегат `GET /api/v1/olap/kitchen-timing-summary?restaurant_id=&business_date_from=&business_date_to=&station_id=&group_by=business_date|station&limit=&offset=` читает confirmed KDS streams `KitchenTicketStatusChanged`/`ItemServed` без raw payload и возвращает lifecycle counts/average transition seconds.
- Async backfill job foundation `GET/POST /api/v1/olap/backfill-jobs`, `GET /api/v1/olap/backfill-jobs/{id}` и `POST /api/v1/olap/backfill-jobs/{id}/cancel` хранит operator jobs в PostgreSQL с UUIDv7 `command_id`, progress/checkpoint/error metadata и audit trail; фактический backfill выполняет background worker без synchronous ClickHouse write в HTTP request path.
- Хранилище master-data packages и Cloud -> Edge package retrieval.
- Cloud-owned master data authority: рестораны, роли, сотрудники, PIN lifecycle, tenant-level каталог, услуги, папки, параметры папок, теги, привязки тегов, группы/опции/привязки модификаторов, policies скидок/надбавок, залы, столы, restaurant-scoped menu items и публикации.
- Restaurant menu overrides реализованы сейчас: один `catalog_item_id` может использоваться несколькими ресторанами через разные `menu_item_id`; menu item хранит override name, price, tag, active tax profile, menu folder/category, availability JSON и runtime status. Cloud publication отправляет на Edge только menu items выбранного ресторана, при этом catalog read model остается Cloud-owned tenant/restaurant scope для кухни и склада.
- Реализованный publication workflow требует operator action после Cloud CRUD; provisioning создает первый snapshot только при его отсутствии. Это фактический gap относительно целевой автоматической доставки.
- Publication flow формирует typed ingest DTO для POS Edge: top-level modifier groups/options/bindings и link-only `menu_item_modifier_groups`.
- Cloud Inventory Worker выполняет recipe expansion основной позиции продажи по active Cloud recipe version и modifier-linked consumption по nullable `ModifierOption.linked_catalog_item_id`; linked modifier item списывается напрямую, без recipe expansion linked item. `CheckClosed` после обработанного `ItemServed` списывает только unserved delta и не дублирует linked modifier rows для полностью served order line.
- Provisioning endpoints: регистрация устройства, список незакрепленных устройств, назначение ресторану, статус назначения и генерация одноразового pairing code через License Server.

Вне текущего объема:

- Production auth/RBAC perimeter для Cloud API.
- Расширенные sales/kitchen/costing агрегаты beyond first bounded endpoint, production-grade backfill jobs/operator UI, production-grade balance rebuild tooling и full inventory costing math.
- Semi-finished auto-production split, COGS/margin и Cloud UI operator workflow.

## License Server

Реализовано сейчас:

- Health endpoint.
- Регистрация одноразового pairing code.
- Resolve pairing code с проверкой срока действия и consumed status.
- Безопасный error contract для invalid/expired/consumed code.
- Structured logs без раскрытия самого pairing code; логируются только факт наличия и длина.

## POS UI

Реализовано сейчас:

- React/Vite `pos-ui-g` cashier UI с PIN-сессией и backend actor context.
- Рабочий терминал кассира: заказ, зал/столы, смена/касса, активность, отчеты и вспомогательные drawer/dialog surfaces.
- Открытие/закрытие личной смены и кассовой смены.
- Выбор столов, активные заказы по залу, создание заказа, добавление меню/услуг, модификаторы, изменение количества, списание строки.
- Выпуск/отмена/повторная печать предчека.
- Оплата наличными и trusted manual card через backend.
- Закрытые заказы с постраничной выдачей, фильтром по бизнес-дате и деталями итогового чека.
- Финансовые операции закрытого чека: просмотр ledger, full cancellation/refund и partial `order_line`/quantity cancellation/refund с явным `inventory_disposition`.
- Compatibility payment refund как отдельный fallback, визуально отделенный от основного ledger flow.
- Sync drawer для status/outbox/local events с bounded запросами.
- Нормализация безопасных API-ошибок и optional empty states; пользовательский текст идет через `pos-ui-g/src/shared/i18n`, а dialog/inline banners показывают безопасный support code (`correlation_id` или stable `error_code`) без raw backend details.
- Waiter terminal mode в `pos-ui-g` как mobile-first order/precheck runtime: выбор зала/стола, активные заказы, создание заказа, меню/поиск, добавление строк с модификаторами, изменение quantity, void line с явной причиной, issue/reprint precheck без payment/refund/cash drawer/fiscal controls по умолчанию; active issued precheck/locked order визуально блокирует add/change/void actions.
- Waiter terminal mode дополнительно стабилизирован под viewport `390x844`: compact context для текущего стола/заказа/статуса и границ полномочий, lock badge на заблокированном меню, touch-friendly table/menu/order rows и scrollable modifier dialog layout без добавления financial authority.
- `/pos/kitchen` как backend-backed KDS runtime: экран читает kitchen queue/tickets, показывает tickets по статусам `new/accepted/in_progress/hold/ready/recall/served/cancelled`, отправляет только подтвержденные backend status actions и перечитывает tickets после ответа backend; `pos-ui-g` kitchen mode также показывает stop-list edit form поверх `POST /api/v1/kitchen/stop-list-updates` и sync indicator из safe `GET /api/v1/kitchen/stop-list` DTO.
- POS shared UI layer шире используется в cashier/readiness surfaces: loading/error/empty/no-permission states и menu skeleton cards переведены на `PosBanner`, `PosEmptyState` и `PosSkeleton`, top/context actions используют shared button/context primitives, а passive backlog/readiness states используют `PosReadinessCard`. В React/Vite `pos-ui-g` подтвержден lightweight shared layer для shell bottom navigation/icon controls, side drawer mode items, tabs/chips/segmented controls, search inputs, selectable tiles, inline status badges, rail headers, dialog selectors и bounded empty/loading/error patterns; KDS pure helpers вынесены в `components/kitchen/kitchenHelpers.ts`, а presentational order/stock/recipe/suggestion/stop-list/proposal rendering вынесен в `KitchenOrdersTab`, `KitchenStockTab`, `KitchenRecipeTab`, `KitchenCatalogSuggestionForm`, `KitchenStopListTab` и `KitchenProposalList`. `POSContext` читает закрытые заказы bounded страницами `listClosedOrders({ businessDateLocal, limit: 26, offset })`, `POSActivitySection` показывает previous/next по backend page-size 25, поддерживает фильтр бизнес-даты и сообщает оператору, что поиск работает только по загруженной странице. Бизнес-команды, платежи, KDS transitions и stop-list enforcement остаются backend-authoritative.

Вне текущего объема:

- UI для скидок/надбавок/налоговых профилей в кассовом терминале.
- UI для modifier/service/tip scopes в financial operation ledger.
- доставка, PSP/fiscal device screens и Cloud-owned складские документы в POS UI.

## Cloud UI

Реализовано сейчас:

- Активный Cloud-бэкофис находится в `cloud-ui-g`: это отдельное React/Vite/TypeScript приложение, не использующее POS session, POS Edge stores или cashier routes.
- `cloud-ui-g` работает с `cloud-backend` через `VITE_CLOUD_API_BASE`, по умолчанию `http://localhost:8090/api/v1`.
- В `cloud-ui-g` реализованы route-backed разделы dashboard, restaurants, Edge sync, catalog, menu, modifiers, pricing/taxes, staff/permissions, floor и publications.
- Dashboard показывает readiness по выбранному ресторану: roles/employees, halls/tables, catalog, menu, modifiers/pricing, Edge assignment и publication.
- Edge sync в `cloud-ui-g` читает незакрепленные устройства, выполняет assign device to restaurant, запрашивает assignment status, генерирует pairing code и показывает безопасный список Edge events без raw payload.
- Master-data разделы `cloud-ui-g` работают поверх подтвержденных Cloud routes для restaurants, roles, employees, catalog items/folders/parameters/tags/item-tags, menu categories/items, modifier groups/options/bindings, pricing policies, halls/tables и publication state/publish. Menu items UI редактирует restaurant overrides для category/menu folder, tag, active tax profile и runtime status поверх стабильных `catalog_item_id`/`menu_item_id`.
- Pricing/taxes в `cloud-ui-g` дополнительно читает и обновляет package `pricing_policy` через provisioning route.
- UI strings в `cloud-ui-g` идут через локальный i18n слой, API responses валидируются Zod-схемами, safe error banner не должен показывать raw payload, PIN/token/request dump или backend internals.
- `cloud-ui-g` имеет navigation placeholders `inventory` и `reports`, но соответствующие React runtime screens сейчас не реализованы и показываются как blocked sections.
- Устаревший `cloud-ui` на Vue/Quasar удален из runtime tree. Исторически он содержал более широкие manager-facing экраны для financial operations, recipes/stop-list, proposal review, inventory readiness, OLAP/read-only reporting и `sales-kitchen-summary`, но новые Cloud UI правки выполняются только в `cloud-ui-g`.

Запланировано далее:

- Дальнейшая разработка Cloud-бэкофиса идет только в `cloud-ui-g`.
- Перенос нужных исторических legacy-сценариев в `cloud-ui-g` выполняется постепенно и только поверх подтвержденных backend routes/DTO.
- Inventory/reporting/OLAP/proposal review экраны в `cloud-ui-g` добавляются отдельно; нельзя считать их реализованными в активном React UI только потому, что похожий код есть в устаревшем Vue UI.

Вне текущего объема:

- Новая разработка Cloud-бэкофиса вне `cloud-ui-g`.
- Cashier runtime, KDS runtime screens, PSP, fiscalization, delivery и POS order/payment/check/precheck flows в Cloud UI.
- Cloud auth/RBAC UI до появления подтвержденного backend-контракта.
- Inventory runtime actions, mutating OLAP retry/backfill controls, BI dashboards, charts и COGS/margin аналитика в активном `cloud-ui-g`.

## Данные и миграции

Реализовано сейчас:

- POS Edge SQLite является локальным OLTP/source of truth для кассового runtime.
- Cloud PostgreSQL является Cloud OLTP/source of truth для приема событий, projections, master data и provisioning foundation.
- ClickHouse является immutable archive для `raw_business_events`; экспорт выполняет только async forwarder из PostgreSQL `inbox_events`.
- License Server использует локальную SQLite БД.
- Active pre-pilot path использует один managed SQL baseline на runtime-модуль и runtime startup migration/verification.
- UUID runtime генерируется как UUIDv7 там, где код создает новые ids через `idgen`.
- Профильные schema tests проверяют критичные таблицы, индексы, constraints, runtime version gates и migration repair behavior.

Вне текущего объема:

- Data-preserving production migrations после первого внедрения.
- Подтвержденный rollout `sqlc`.
- Production-grade ClickHouse backfill/retention jobs, richer sales/kitchen analytics и COGS/margin OLAP reads.

## Скрипты и локальная приемка

Реализовано сейчас:

- Docker compose поднимает Cloud PostgreSQL, ClickHouse, Cloud API, License API и POS Edge без POS UI.
- `scripts/seed-dev-system.py` является единственным user-facing Python demo/seed entrypoint для Fedora/Linux/Windows-compatible локального контура. Helper/test-only scripts в текущем `scripts/` отсутствуют; прежние wrapper/onboarding paths не являются актуальными пользовательскими сценариями.
- Единственный Python seed script использует HTTP API Cloud/POS/License и не делает прямых записей в PostgreSQL/SQLite/ClickHouse.
- `scripts/seed-dev-system.py` проверяет health Cloud/POS/License, создает полный Cloud-owned seed dataset, создает active recipe versions через manager draft -> submit -> approve flow, публикует master data, выполняет license pairing POS Edge и проверяет базовый POS read model.
- `scripts/seed-dev-system.py --run-minimal-flow` выполняет минимальный HTTP-only smoke: Cloud recipes/stop-list publication, Edge sync, waiter order/precheck, KDS served, cashier payment/final check, прием `ItemServed`/`CheckClosed` в Cloud, появление строк Cloud `stock_ledger` по `ItemServed`, появление materialized `stock-balances`, отсутствие duplicate `CheckClosed` delta для того же `order_line_id`, экспорт событий в ClickHouse `raw_business_events`, экспорт складских движений в `olap_stock_moves` и bounded reads `stock-move-summary`/`sales-kitchen-summary` без raw payload.
- `scripts/seed-dev-system.py --run-kitchen-process-smoke` выполняет профильный kitchen/process smoke: Cloud seed publication для catalog/menu/recipes/inventory_reference, Edge sync, waiter order, kitchen order tile, `accept/start/ready/serve`, `recall/start/ready/serve`, ClickHouse `raw_business_events`, Cloud stock ledger и `olap_stock_moves` read для receipt/count/write-off/production, catalog/recipe suggestions, Cloud manager approve и Edge proposal feedback. При одновременном запуске `--run-minimal-flow` и `--run-kitchen-process-smoke` summary содержит отдельные секции `minimal_flow` и `kitchen_process_smoke`; полный kitchen/process smoke использует kitchen role/PIN, а не manager PIN.
- PowerShell/Bash wrappers и прежние onboarding flows удалены; в `scripts` остается один пользовательский Python seed script.
- HTTP слой скриптов игнорирует proxy для localhost/loopback, чтобы не ломать Docker published ports.
- Seed/smoke guards закрепляют отсутствие direct DB client imports, запрет destructive storage/archive routes и отсутствие automatic retry financial mutations.

## Покрытие тестами бизнес-логики

Реализовано сейчас:

- POS service tests покрывают RBAC, PIN/auth, смены, кассовые смены, idempotency command id, transaction rollback, order/precheck/payment/check lifecycle, manager override, business date, reprint, modifiers, service items, pricing, financial operation caps, partial cancellation/refund, mixed refunds, outbox, Cloud master-data ingest и storage archive readiness.
- POS API tests покрывают безопасные HTTP errors, сессии, pairing/provisioning, master-data route boundaries, floor/order/precheck/payment/check endpoints, sync/storage endpoints и CORS.
- POS SQLite tests покрывают schema constraints, active managed baseline, payments by `precheck_id`, prechecks, local event log, outbox retry schema, modifiers, отсутствие legacy Edge stock tables и migration repair.
- Cloud/POS sync tests покрывают idempotent receive, item-level batch ACK, authenticated exchange, temporary exchange failure -> retry -> ACK на POS sender, revision conflicts, current financial operation events, legacy refund events, master-data packages, `StopListUpdated` replay queue idempotency, stop-list readiness no-raw-payload contract и contract validation.
- Cloud OLAP tests покрывают raw event и stock moves forwarder success/retry, bounded read validation, read-only export status, минимальный export-retry control validation/API, stock move summary и sales-kitchen summary limit/filter/grouping/empty state, а также отсутствие raw payload в OLAP API; PostgreSQL schema tests покрывают `inbox_events`, checkpoint contract, `olap_export_retry_commands` и `cloud_projection_stop_list_updates`.
- Cloud master-data tests покрывают CRUD/validation, PIN reuse rules, role permission validation, catalog/menu/publication shape, service/semi-finished kinds, lifecycle statuses и pricing policies.
- License tests покрывают registration, resolve, consumed/expired/invalid pairing codes.
- UI unit/e2e tests покрывают currency/error/session guards, RBAC, schema parsing, cashier terminal conflict handling, compensation boundaries, modifier flow, payments/refunds, refund после закрытия исходных personal/cash shifts, запрет cancellation после закрытия исходной смены и sync/provisioning flows.
- Script tests покрывают единственный seed flow, materialized balance assertion в minimal flow, отсутствие других user-facing Python entrypoints в `scripts`, отсутствие preassigned IDs в seed dataset, dev-only summary рядом со скриптом, отключение proxy для localhost/loopback, отсутствие direct DB client imports, запрет destructive storage/archive routes, single-shot financial mutation request, карту расширения Cloud-owned seed surfaces и генерацию pairing после публикации master data.

Оставшиеся риски:

- Полный `go test ./...` и `npm run build` нужно запускать после каждого изменения соответствующего кода; этот документ фиксирует покрытие по найденным тестам, а не заменяет запуск CI.
- Полный локальный Docker smoke `--run-minimal-flow --run-kitchen-process-smoke` подтвержден 01.06.2026 в окружении с доступным Docker Compose/buildx. Local compose поддерживает переопределение host ports для PostgreSQL, ClickHouse HTTP/native, Cloud API, POS Edge и License API; buildx blocker остается требованием локального Docker CLI/Compose окружения.
- Cloud Backend теперь имеет отдельный профильный contract: `docs/backend/CLOUD-BACKEND-SPEC.md`; при изменении Cloud routes, provisioning, publication, sync receiver или PostgreSQL schema его нужно обновлять вместе с кодом.
- Legacy Edge-side manual inventory foundation удален из кода и managed SQLite baseline; историческая пометка сохранена в профильных sync/data docs.

## Запланировано далее

- До первого выставочного запуска: автоматическая per-Edge batch assembly без Publish action, ticket issuance/QR printing, ESC/POS subsystem, Cloud sales dashboard и Telegram worker. Внешний licensing authority и gates существующих table/kitchen/warehouse surfaces реализованы сейчас; tenant-level roles/employees, memberships, tenant catalog identity и restaurant menu overrides также реализованы сейчас.
- После первого выставочного запуска: checker enrollment, scanner UI, typed Cloud-Edge relay, QR lookup/confirm/revoke и usage reporting.
- Поддерживать `docs/backend/CLOUD-BACKEND-SPEC.md` как профильный документ Cloud Backend при каждом изменении Cloud routes, payloads, sync/provisioning contracts или schema.
- При добавлении Cloud-owned сценария обновлять `scripts/seed-dev-system.py`, publication stream/package, POS read flow/smoke assertion и профильные документы в одном PR.
- До полного пилота: полный Cloud Inventory Engine, richer sales/costing OLAP API beyond current bounded endpoints и production auth/RBAC perimeter для mutating OLAP operator controls.
- После полного пилота: hardware bump-bar integrations, kitchen printer orchestration, rich BI dashboards, ERP/accounting integrations и внешние delivery/payment/fiscal контуры.
- Data-preserving migrations после первого реального внедрения.
- Production auth/RBAC perimeter для Cloud/License API.

## Вне текущего объема полного пилота

- Delivery/channel integrations.
- Настоящий PSP/payment processor module и PSP refund.
- Fiscal device integration.
- Cashier/KDS/manager mobile variants outside waiter screen.
