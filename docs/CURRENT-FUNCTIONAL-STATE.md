# Текущее функциональное состояние проекта

Статус: реализовано сейчас по коду, тестам и документации на 2026-05-26; цель полного пилота зафиксирована отдельно и не считается текущим runtime.

Этот документ является сводной картой фактического состояния репозитория. Он не заменяет профильные спецификации: архитектурные инварианты остаются в `SPECv1.3.md`, backend-контракты - в `docs/backend/*`, контракты интерфейсов - в `docs/ui/*`, синхронизация - в `docs/sync/*`.

Источник истины для реализованного runtime: код, миграции и тесты. Если этот документ конфликтует с кодом или тестами, сначала фиксируется фактическое поведение по коду, затем обновляется документация.

## Найдено при ревью

Реализовано сейчас:

- Документация уже описывает ключевой кассовый поток `Order -> Precheck -> Payment -> Check`, предчеки, оплаты, итоговые чеки, возвраты/отмены, синхронизацию и локальный smoke path.
- Основной пробел был не в отдельных POS-инвариантах, а в отсутствии единой русскоязычной сводки по всему репозиторию: POS Edge, Cloud Backend, License Server, POS UI, Cloud UI, миграции, скрипты и тестовое покрытие были описаны в разных документах.
- Кодовая база содержит больше подтвержденных Cloud и provisioning маршрутов, чем явно видно из POS-ориентированных документов; они зафиксированы ниже как реализованная сейчас функциональность.

Не обнаружено сейчас:

- Подтвержденного runtime для delivery, настоящего платежного процессинга, фискального адаптера, ClickHouse pipeline и расширенных KDS flows за пределами ticket lifecycle foundation.
- Публичного Cloud HTTP/API интерфейса отчетов по детальной проекции финансовых операций. Сервисная и repository-основа есть, публичный reporting surface остается запланированным далее.

Цель полной пилотной реализации:

- сохранить текущий cashier runtime как базовый поток;
- stop-list sale blocking на POS Edge с Cloud authoring/publication и offline локальной проверкой уже подтвержден в `pos_stop_list_sale_blocking`;
- расширять mobile-first waiter runtime без payment/refund authority по умолчанию только по подтвержденным backend contracts;
- расширить KDS lifecycle за пределы минимального ticket foundation: cooking events, приемку поставки, catalog proposals, recipe change proposals и stop-list edit;
- зафиксировать POS Edge backend как авторитетный runtime для financial/order/KDS command validation и stop-list sale blocking; POS UI не становится авторитетным слоем;
- добавить Cloud manager flow для recipes, stop-list, catalog/recipe proposal review, inventory operations, publication readiness и sync/problem observability;
- добавить полный Cloud-owned складской движок: stock receipts, inventory counts, production, sale consumption, refund/cancellation dispositions, recipe expansion, balances, costing и retro recalculation DAG;
- добавить ClickHouse runtime как immutable/OLAP storage, async export pipeline и bounded Cloud OLAP API;
- закрыть полный smoke path Cloud setup -> Edge sync -> waiter order -> kitchen served -> cashier payment/check -> Cloud inventory ledger -> ClickHouse export -> OLAP API.

## POS Edge Backend

Реализовано сейчас:

- HTTP API на `chi` с безопасным JSON error contract, request id, структурированным audit log и CORS для локального POS UI.
- PIN-вход, backend-сессии, привязка `node_device_id` / `client_device_id`, проверка actor context и rate limit для PIN.
- RBAC на уровне application services; UI visibility не является границей безопасности.
- Личная смена сотрудника и кассовая смена устройства как разные runtime-понятия.
- Залы, столы, меню и каталог как локальные read models, получаемые из Cloud-owned справочников.
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
- Минимальный KDS ticket lifecycle: `kitchen_tickets` создаются из non-service order lines, `GET /api/v1/kitchen/tickets` поддерживает bounded read/status filter, status actions `accept/start/hold/ready/serve/recall/cancel` проверяют `pos.kitchen.status.change`, пишут `KitchenTicketStatusChanged`, а `serve` дополнительно пишет `ItemServed`.
- Локальный lifecycle SQLite: status, retention dry-run, archive export plan, export-only JSONL archive, read-plan, lookup preview, apply-plan и apply-readiness с поддержкой destructive apply (физическое удаление закрытых orders/checks/financial_operations и связанных при verified JSONL + чистый scoped outbox + отсутствие открытых operational boundaries для cutoff периода) и последующий VACUUM compaction БД.
- Cloud -> Edge master-data ingest для `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`.
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
- `ItemServed` попадает в durable `inventory_event_queue` и Cloud Inventory Worker создает sale ledger идемпотентно по source event; `KitchenTicketStatusChanged` принимается как operational-only event и не ставится в inventory queue.
- Детальная PostgreSQL projection для current `CancellationRecorded` и `RefundRecorded`; legacy `PaymentRefunded`/`CheckRefunded` принимаются, но не наполняют detailed operation projection.
- Безопасный список входящих Edge events для Cloud UI без raw payload.
- Хранилище master-data packages и Cloud -> Edge package retrieval.
- Cloud-owned master data authority: рестораны, роли, сотрудники, PIN lifecycle, каталог, услуги, папки, параметры папок, теги, привязки тегов, группы/опции/привязки модификаторов, policies скидок/надбавок, залы, столы, menu items и публикации.
- Publication flow формирует typed ingest DTO для POS Edge: top-level modifier groups/options/bindings и link-only `menu_item_modifier_groups`.
- Provisioning endpoints: регистрация устройства, список незакрепленных устройств, назначение ресторану, статус назначения и генерация одноразового pairing code через License Server.

Вне текущего объема:

- Production auth/RBAC perimeter для Cloud API.
- Публичный Cloud reporting API/UI по financial operation projection.
- Async ClickHouse forwarder и immutable OLAP archive.
- Recipe expansion, semi-finished auto-production split и retro costing DAG.

## License Server

Реализовано сейчас:

- Health endpoint.
- Регистрация одноразового pairing code.
- Resolve pairing code с проверкой срока действия и consumed status.
- Безопасный error contract для invalid/expired/consumed code.
- Structured logs без раскрытия самого pairing code; логируются только факт наличия и длина.

## POS UI

Реализовано сейчас:

- Vue/Quasar cashier UI с PIN-сессией и backend actor context.
- Рабочий терминал кассира: заказ, зал/столы, смена/касса, активность, отчеты и вспомогательные drawer/dialog surfaces.
- Открытие/закрытие личной смены и кассовой смены.
- Выбор столов, активные заказы по залу, создание заказа, добавление меню/услуг, модификаторы, изменение количества, списание строки.
- Выпуск/отмена/повторная печать предчека.
- Оплата наличными и trusted manual card через backend.
- Закрытые заказы с постраничной выдачей, фильтром по бизнес-дате и деталями итогового чека.
- Финансовые операции закрытого чека: просмотр ledger, full cancellation/refund и partial `order_line`/quantity cancellation/refund с явным `inventory_disposition`.
- Compatibility payment refund как отдельный fallback, визуально отделенный от основного ledger flow.
- Sync drawer для status/outbox/local events с bounded запросами.
- Нормализация безопасных API-ошибок и optional empty states; пользовательский текст идет через `vue-i18n`, а dialog/inline banners показывают безопасный support code (`correlation_id` или stable `error_code`) без raw backend details.
- `/pos/waiter` как mobile-first order/precheck runtime: выбор зала/стола, активные заказы, создание заказа, меню/поиск, добавление строк с модификаторами, изменение quantity, void line, issue/reprint precheck без payment/refund/cash drawer controls по умолчанию; active issued precheck/locked order визуально блокирует add/change/void actions.
- `/pos/waiter` дополнительно стабилизирован под viewport `390x844`: sticky compact context dock для текущего стола/заказа/статуса и границ полномочий, sticky topbar, lock badge на заблокированном меню, touch-friendly table/menu/order rows и scrollable modifier dialog layout без добавления financial authority.
- `/pos/kitchen` как минимальный backend-backed KDS runtime: экран читает `GET /api/v1/kitchen/tickets`, показывает tickets по статусам `new/accepted/in_progress/hold/ready/recall/served/cancelled`, отправляет только подтвержденные backend status actions и перечитывает tickets после ответа backend.
- POS shared UI layer шире используется в cashier/readiness surfaces: loading/error/empty/no-permission states и menu skeleton cards переведены на `PosBanner`, `PosEmptyState` и `PosSkeleton`, top/context actions используют shared button/context primitives, а passive backlog/readiness states используют `PosReadinessCard`.

Вне текущего объема:

- UI для скидок/надбавок/налоговых профилей в кассовом терминале.
- UI для modifier/service/tip scopes в financial operation ledger.
- Складские операции в POS UI, kitchen receipt/proposal/stop-list edit actions, доставка, PSP/fiscal device screens.

## Cloud UI

Реализовано сейчас:

- Отдельный интерфейс для Cloud-owned сценариев, не использующий POS session или POS Edge stores.
- Первый сценарий запуска: readiness panel по restaurant/staff/floor/catalog/menu/modifiers/pricing/Edge/publication, Edge-device flow, master data preparation и публикация snapshot.
- Управление ресторанами, ролями, сотрудниками, каталогом, папками, тегами, модификаторами, policies, залами, столами, menu items и публикациями по подтвержденным Cloud routes.
- Генерация pairing code и назначение Edge-device ресторану.
- Просмотр безопасного списка входящих Edge events без raw payload, включая card/list fallback на narrow screens с metadata/checksum вместо raw event payload; resource lists на narrow screens показывают status label в карточке и не раскрывают raw payload.
- Cloud-owned recipes и stop-list authoring через подтвержденные master-data routes.
- Readiness-only manager surfaces для catalog/recipe proposal review, inventory operations/costing и OLAP exports без имитации отсутствующих Cloud routes.
- Локализованные сообщения, safe error details, no raw payload / PIN / token display; Cloud create/rotate PIN поля используют password input, а списки сотрудников показывают только `pin_configured` и credential version.

Вне текущего объема:

- Cashier runtime в Cloud UI.
- Cloud auth/RBAC UI.
- KDS runtime screens, PSP, fiscalization, delivery и cashier runtime в Cloud UI; inventory runtime actions и OLAP exports должны появиться в Cloud UI только после подтвержденных backend contracts.

## Данные и миграции

Реализовано сейчас:

- POS Edge SQLite является локальным OLTP/source of truth для кассового runtime.
- Cloud PostgreSQL является Cloud OLTP/source of truth для приема событий, projections, master data и provisioning foundation.
- License Server использует локальную SQLite БД.
- Active pre-pilot path использует один managed SQL baseline на runtime-модуль и runtime startup migration/verification.
- UUID runtime генерируется как UUIDv7 там, где код создает новые ids через `idgen`.
- Профильные schema tests проверяют критичные таблицы, индексы, constraints, runtime version gates и migration repair behavior.

Вне текущего объема:

- Data-preserving production migrations после первого внедрения.
- Подтвержденный rollout `sqlc`.
- ClickHouse как запущенный runtime pipeline.

## Скрипты и локальная приемка

Реализовано сейчас:

- Docker compose поднимает Cloud PostgreSQL, Cloud API, License API и POS Edge без POS UI.
- Единственный Python seed script использует HTTP API и не делает прямых записей в PostgreSQL/SQLite.
- `scripts/seed-dev-system.py` проверяет health Cloud/POS/License, создает полный Cloud-owned seed dataset, публикует master data, выполняет license pairing POS Edge и проверяет базовый POS read model.
- `scripts/seed-dev-system.py --run-minimal-flow` выполняет минимальный HTTP-only smoke: Cloud recipes/stop-list publication, Edge sync, waiter order/precheck, KDS served, cashier payment/final check, прием `ItemServed`/`CheckClosed` в Cloud, появление строк Cloud `stock_ledger` по `ItemServed` и отсутствие duplicate `CheckClosed` delta для того же `order_line_id`.
- PowerShell/Bash wrappers и прежние onboarding flows удалены; в `scripts` остается один пользовательский Python seed script.
- HTTP слой скриптов игнорирует proxy для localhost/loopback, чтобы не ломать Docker published ports.

## Покрытие тестами бизнес-логики

Реализовано сейчас:

- POS service tests покрывают RBAC, PIN/auth, смены, кассовые смены, idempotency command id, transaction rollback, order/precheck/payment/check lifecycle, manager override, business date, reprint, modifiers, service items, pricing, financial operation caps, partial cancellation/refund, mixed refunds, outbox, Cloud master-data ingest и storage archive readiness.
- POS API tests покрывают безопасные HTTP errors, сессии, pairing/provisioning, master-data route boundaries, floor/order/precheck/payment/check endpoints, sync/storage endpoints и CORS.
- POS SQLite tests покрывают schema constraints, active managed baseline, payments by `precheck_id`, prechecks, local event log, outbox retry schema, modifiers, отсутствие legacy Edge stock tables и migration repair.
- Cloud sync tests покрывают idempotent receive, item-level batch ACK, authenticated exchange, revision conflicts, current financial operation events, legacy refund events, master-data packages и contract validation.
- Cloud master-data tests покрывают CRUD/validation, PIN reuse rules, role permission validation, catalog/menu/publication shape, service/semi-finished kinds, lifecycle statuses и pricing policies.
- License tests покрывают registration, resolve, consumed/expired/invalid pairing codes.
- UI unit/e2e tests покрывают currency/error/session guards, RBAC, schema parsing, cashier terminal conflict handling, compensation boundaries, modifier flow, payments/refunds, refund после закрытия исходных personal/cash shifts, запрет cancellation после закрытия исходной смены и sync/provisioning flows.
- Script tests покрывают единственный seed flow, отсутствие других user-facing Python entrypoints в `scripts`, отсутствие preassigned IDs в seed dataset и генерацию pairing после публикации master data.

Оставшиеся риски:

- Полный `go test ./...` и `npm run build` нужно запускать после каждого изменения соответствующего кода; этот документ фиксирует покрытие по найденным тестам, а не заменяет запуск CI.
- Cloud Backend теперь имеет отдельный профильный contract: `docs/backend/CLOUD-BACKEND-SPEC.md`; при изменении Cloud routes, provisioning, publication, sync receiver или PostgreSQL schema его нужно обновлять вместе с кодом.
- Legacy Edge-side manual inventory foundation удален из кода и managed SQLite baseline; историческая пометка сохранена в профильных sync/data docs.

## Запланировано далее

- Поддерживать `docs/backend/CLOUD-BACKEND-SPEC.md` как профильный документ Cloud Backend при каждом изменении Cloud routes, payloads, sync/provisioning contracts или schema.
- Публичный Cloud reporting API/UI для detailed financial operation projection.
- До полного пилота: расширение KDS за пределы ticket lifecycle foundation, chef receipt/catalog/recipe proposal flows, полный Cloud Inventory Engine, ClickHouse runtime/OLAP API и `full_pilot` smoke.
- После полного пилота: hardware bump-bar integrations, kitchen printer orchestration, rich BI dashboards, ERP/accounting integrations и внешние delivery/payment/fiscal контуры.
- Data-preserving migrations после первого реального внедрения.
- Production auth/RBAC perimeter для Cloud/License API.

## Вне текущего объема полного пилота

- Delivery/channel integrations.
- Настоящий PSP/payment processor module и PSP refund.
- Fiscal device integration.
- Cashier/KDS/manager mobile variants outside waiter screen.
