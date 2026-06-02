# Cloud UI Spec

## Назначение

реализовано сейчас: `cloud-ui` является отдельным пилотным интерфейсом для `cloud-backend` и Cloud-owned операционных сценариев.

`cloud-ui` не является частью `pos-ui`, не использует POS session, POS Edge runtime endpoints, cashier routes или локальные POS stores.

## Целевой сценарный план

реализовано сейчас:

1. Подключение POS Edge к Cloud через `cloud-backend` provisioning routes.
2. Проверка минимальной готовности ресторана, зала, ролей и сотрудников.
3. Подготовка продаваемого меню поверх существующих Cloud-owned master data.
4. Явная публикация master data package для Edge.
5. Передача опубликованного snapshot на Edge, где далее формируются заказ и продажа.
6. Manager-facing recipes и stop-list authoring по подтвержденным Cloud master-data routes.
7. Route-backed manager review для catalog/recipe suggestions и Edge-origin stop-list updates, bounded read-only поверхности для inventory balances, stock ledger, OLAP export/stock movement slices, backfill job status и kitchen timing summary без неподтвержденных команд.
8. Read-only Cloud reporting по detailed financial operation projection для `CancellationRecorded`/`RefundRecorded`.
9. Минимальный read-only preview bounded OLAP агрегата `sales-kitchen-summary`.

запланировано далее:

- вывести связи `catalog item -> menu item -> modifier bindings -> pricing policies` как единый сценарий подготовки продажи;
- показывать версии опубликованного пакета и состояние доставки на Edge, когда backend подтвердит такой контракт.
- до полного пилота расширять manager surfaces для inventory operations, costing status, ClickHouse export readiness и OLAP API diagnostics только поверх подтвержденных Cloud backend routes; текущий UI уже читает bounded `stock-balances`, `stock-ledger`, OLAP export status, stock moves, stock move summary и `sales-kitchen-summary` без mutating controls или BI dashboard;

вне текущего объема:

- создание справочников как отдельная бизнес-цель;
- cashier runtime, order/payment/check/precheck flows в Cloud UI;
- Cloud auth/RBAC UI до появления подтвержденного backend-контракта.

## Границы

реализовано сейчас:

- план запуска Cloud UI от подключения Edge-device до продажи на Edge-стороне является primary journey первого экрана;
- панель готовности онбординга по выбранному ресторану: ресторан выбран, роли/сотрудники готовы, зал/столы готовы, каталог заполнен, меню продаваемо, модификаторы и pricing policies подготовлены, Edge назначен, публикация создана и snapshot доступен;
- для каждого blocked readiness item показывается next best action button к профильному разделу;
- список незакрепленных Edge-device из `/api/v1/devices/unassigned`;
- назначение Edge-device ресторану через `/api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign`;
- проверка assignment status через `/api/v1/devices/{node_device_id}/assignment-status`;
- генерация pairing code через `/api/v1/restaurants/{restaurant_id}/devices/generate-pairing-code`;
- управление ресторанами;
- роли и сотрудники; роли создаются через предустановленные профили и матрицу POS-прав, а не через ручной ввод `permissions_json`;
- catalog items;
- catalog folders;
- folder parameters;
- catalog tags;
- item tags как command-only привязка;
- modifier groups, options и bindings;
- pricing policies;
- tax/service-charge package authoring через `GET/PUT /api/v1/provisioning/master-data/pricing_policy` с сохранением `tax_profiles`, `tax_rules`, `service_charge_rules` в `payload_json`;
- recipe items через `/api/v1/master-data/recipes/items`;
- сценарный editor версий техкарт через `/api/v1/master-data/recipes/versions`, `/api/v1/master-data/recipes/versions/drafts`, `/api/v1/master-data/recipes/versions/{id}/submit`;
- stop-list entries через `/api/v1/master-data/inventory/stop-list`;
- route-backed раздел `Очередь предложений` для Cloud review workflow (`catalog-suggestions`/`recipe-suggestions`/`manager/stop-list-updates`) со списками catalog/recipe suggestions и Edge-origin stop-list updates, detail/diff view, approve/reject/request-changes actions, safe assignment metadata из backend review DTO, linked new dish + recipe group display и publication/readiness signal после approve; раздел `Готовность склада` читает `GET /api/v1/sync/readiness/stop-list` для safe stop-list/publication/Edge ACK/sync problem summary, `GET /api/v1/inventory/stock-balances` для bounded balance table и `GET /api/v1/inventory/stock-ledger` для первых 50 ledger rows без raw payload; раздел `OLAP exports` читает `GET /api/v1/olap/export-status`, `GET /api/v1/olap/stock-moves`, `GET /api/v1/olap/stock-move-summary`, `GET /api/v1/olap/backfill-jobs` и `GET /api/v1/olap/kitchen-timing-summary` как bounded operator preview без retry/backfill mutation controls; раздел `Sales/kitchen summary` читает bounded `GET /api/v1/olap/sales-kitchen-summary` как минимальный table/card preview без raw payload;
- halls и tables;
- menu items;
- menu category create как command-only операция, потому что list/update routes не подтверждены;
- publication summary и явная публикация master data;
- опубликованный snapshot для Edge использует backend Cloud -> POS Edge ingest DTO: top-level modifier groups/options/bindings передаются отдельно от link-only `menu_item_modifier_groups`, без rich/UI projection fields внутри `menu_items`;
- `GET /api/v1/restaurants/{id}/master-data/publication-state` возвращает `200 null` до первой публикации выбранного ресторана; Cloud UI трактует это как empty state панели публикации, а не как ошибку browser console;
- отдельный раздел `События от Edge`, который читает `GET /api/v1/sync/edge-events` и показывает только безопасные receipt metadata/checksum без raw payload; на narrow screens таблица заменяется card/list fallback с теми же безопасными полями;
- отдельный read-only раздел `Финансовые операции`, который читает `GET /api/v1/reporting/financial-operations` с фильтрами business date from/to, operation type, shift, original shift и check; UI показывает projection metadata/checksum без raw sync payload, snapshot JSON, PIN/token/request dump и без cashier mutations;
- отдельный read-only раздел `Sales/kitchen summary`, который читает `GET /api/v1/olap/sales-kitchen-summary` с `restaurant_id`, `business_date_from`, `business_date_to`, `group_by`, `limit=50` и `offset=0`; UI показывает bounded table/card preview по безопасным aggregate fields (`group_by`, `group_key`, optional business date/event/source event/catalog item, counts, quantities, total movement minor amount и first/last timestamps) без raw payload, snapshot JSON, retry/backfill controls, графиков, BI dashboard и COGS/margin расчетов;
- resource lists на narrow screens переходят с широких таблиц на карточки, где статус выводится тем же safe status label, а не raw payload или POS runtime detail.

запланировано до полного пилота:

- recipe editor имеет bounded route-backed строки recipe items и реализовано сейчас сценарный editor версий: просмотр текущих версий, draft form с компонентной строкой, save draft / submit, review выполняется в существующей очереди предложений;
- duplicate hints и linked receipt line для catalog suggestion review остаются запланировано далее;
- stop-list panel уже имеет bounded route-backed rows; `Готовность склада` показывает default conflict policy, async projection mode, publication/package metadata, latest Edge ACK metadata и sync problem counters без raw payload; Cloud review queue показывает safe Edge-origin stop-list update summary/diff, approve/reject/request-changes и safe assignment metadata без raw payload. Реализовано сейчас: backend routes `POST /api/v1/manager/stop-list-updates/{id}/assign|unassign` поддерживают только `stop_list_update` с UUIDv7 `command_id` и append-only audit. Assignment для catalog/recipe review запланирован далее. Escalation workflow и dashboard refactor запланированы далее. Raw payload exposure вне текущего объема и запрещено;
- реализовано сейчас: inventory readiness surface показывает route-backed `stock-balances` table из Cloud backend с фильтрами `warehouse_id`, `catalog_item_id`, `business_date_to`, `costing_status`, aggregate costing status и readiness signals stop-list без raw payload. Там же есть read-only stock ledger preview по `GET /api/v1/inventory/stock-ledger` с `restaurant_id`, `catalog_item_id`, `source_event_type`, `source_event_id`, `order_line_id`, `limit=50` и `offset=0`; UI показывает только safe table/card поля ledger DTO и не выполняет складские команды. Edge-side stock receipts, inventory counts, write-offs and production input are covered by `pos-ui-g` kitchen mode and Cloud ledger/balance read endpoints;
- запланировано далее: stock documents table, full costing/recalculation operator workflow и inventory runtime actions в Cloud UI;
- реализовано сейчас: ClickHouse/OLAP workspace показывает read-only export status для `raw_business_events` и `stock_moves`, bounded preview `olap_stock_moves`, stock move summary с группировкой `business_date`, `catalog_item`, `warehouse`, backfill job status, kitchen timing summary и минимальный read-only `sales-kitchen-summary` preview. UI не вызывает `POST /api/v1/olap/export-retry` и mutating backfill controls, не показывает COGS/margin расчеты и не является BI dashboard;
- launch readiness должен учитывать stop-list review и публикацию streams `recipes`/`stop_lists`;
- publication panel показывает latest package version и target Edge node; latest known Edge ACK для stop-list отображается в readiness summary, package delivery ACK как отдельный contract остается запланирован далее;
- Edge events/problem events panel должен показывать accepted/rejected/retryable metadata без raw payload.

вне текущего объема полного пилота:

- KDS runtime screens в Cloud UI;
- PSP;
- fiscalization;
- ERP/accounting integrations;
- rich BI dashboards beyond pilot OLAP endpoint previews;
- delivery;
- cashier runtime;
- POS order/payment/check/precheck flows.

## UX

реализовано сейчас: интерфейс перестроен из чистой admin surface в операционный центр с двумя слоями:

- сценарный слой запуска: план внедрения и подключение Edge-device;
- технический слой master data: существующие таблицы и формы для подтвержденных backend routes.

Правила UI:

- первое действие оператора — открыть план запуска или подключить Edge-device;
- выбор ресторана остается обязательным для restaurant-scoped операций;
- UX разбит на presentation components: shell/navigation, launch readiness, Edge-device flow, publication panel, resource list/table, resource form и role permission matrix;
- для narrow screens ключевые launch/Edge/publication/resource states и Edge events имеют card/list fallback, а таблицы остаются desktop/admin представлением; sidebar становится scrollable, чтобы master-data navigation не ломала рабочую область на tablet/narrow viewport;
- технические связи между сущностями выбираются из загруженных справочников; пользователь не вводит ID вручную в подтвержденных связях;
- pairing code flow не требует ввода `node_device_id`: Cloud генерирует device id на backend-стороне;
- publication flow позволяет выбрать известное Edge-устройство из UI-состояния или опубликовать общий пакет без ручного ввода ID;
- роли выбираются из профилей `cashier`, `senior_cashier`, `waiter`, `manager`, `kitchen`, `support_admin`, после чего оператор может изменить права в матрице;
- создание и ротация PIN используют password input; после сохранения Cloud UI показывает только флаги/версии credential lifecycle, а не PIN material;
- Edge-device flow не показывает секреты кроме одноразового pairing code, который возвращает backend;
- command-only разделы не показывают неподтвержденную таблицу;
- Cloud UI показывает безопасные локализованные ошибки возле активного failed step с recovery action: retry, select restaurant или open related section; message key, support code, correlation id и безопасные details выводятся без raw payload, а подозрительные `payload`/`token`/`PIN`/`SQL`/`stack` details редактируются в UI;
- раздел входящих Edge events выводит event metadata и checksum, но не показывает raw payload, sensitive request dumps или payload-derived финансовые details;
- раздел финансовых операций выводит только read-only projection fields из Cloud reporting route; он не вызывает POS Edge endpoints, не создает Cloud cashier commands и не рассчитывает COGS/margin;
- раздел `Sales/kitchen summary` и OLAP kitchen timing выводят только безопасные aggregate fields из bounded Cloud OLAP routes; они не вызывают POS Edge endpoints, не создают cashier commands, не показывают raw payload/request dump/snapshot JSON и не строят BI dashboard, charts, COGS или margin;
- раздел `Очередь предложений` не выводит raw `payload_json`: detail/diff строится только по whitelist полям catalog proposal (`kind`, `name`, `sku`, `base_unit`, `kitchen_type`, `accounting_category`) и recipe proposal changes (`action`, `from_catalog_item_id`, `to_catalog_item_id`, `quantity`, `unit_code`, `loss_percent`); assignment metadata выводится только safe полями (`assigned_to_employee_id`, `assigned_by_employee_id`, `assigned_at`, `assignment_note`); PIN/token/secret/request dump не отображаются;
- approve/reject/request-changes формы отправляют только `reviewed_by_employee_id`, optional `review_comment` и optional `published_by`; после approve UI перечитывает `publication-state`, потому что apply/publish выполняет backend;
- UX ориентиры полного пилота зафиксированы в `docs/ui/PILOT-UX-MARKET-REVIEW.md`; business workflows не должны требовать ручного ввода UUID/raw JSON для обычного менеджерского сценария;
- пользовательские тексты идут через `vue-i18n`.

## API

реализовано сейчас: API client `cloud-ui/src/shared/api.ts` использует подтвержденные routes из:

- `cloud-backend/internal/provisioning/api/router.go` для Edge-device provisioning;
- `cloud-backend/internal/masterdata/api/router.go` для master data и публикации;
- `cloud-backend/internal/cloudsync/api/router.go` для безопасного списка входящих Edge events, stop-list readiness, inventory balances, stock ledger и Cloud financial reporting;
- `cloud-backend/internal/olap/api/router.go` для read-only OLAP export status, stock moves и stock move summary.

Реализовано сейчас для financial reporting:

- `GET /api/v1/reporting/financial-operations?restaurant_id=&business_date_from=&business_date_to=&operation_type=&shift_id=&original_shift_id=&check_id=&limit=&offset=`;
- response schema валидируется в `cloud-ui/src/shared/schemas.ts` и не содержит raw sync payload или snapshot JSON;
- UI surface находится в `cloud-ui/src/components/cloud/FinancialOperationsPanel.vue`.

Реализовано сейчас для `Очередь предложений`:

- `GET /api/v1/master-data/catalog-suggestions?restaurant_id=&status=&limit=&offset=`;
- `GET /api/v1/master-data/recipe-suggestions?restaurant_id=&status=&limit=&offset=`;
- `POST /api/v1/master-data/catalog-suggestions/{id}/approve|reject|request-changes`;
- `POST /api/v1/master-data/recipe-suggestions/{id}/approve|reject|request-changes`.
- `POST /api/v1/manager/stop-list-updates/{id}/assign`;
- `POST /api/v1/manager/stop-list-updates/{id}/unassign`.
- `GET /api/v1/master-data/recipes/versions?restaurant_id=&owner_catalog_item_id=&status=&limit=&offset=`;
- `POST /api/v1/master-data/recipes/versions/drafts`;
- `POST /api/v1/master-data/recipes/versions/{id}/submit`.

Review command body:

```json
{
  "reviewed_by_employee_id": "manager-1",
  "review_comment": "approved",
  "published_by": "cloud-ui"
}
```

`review_comment` и `published_by` optional; `published_by` используется backend approve flow для публикации master data. UI не вызывает неподтвержденных detail endpoints для suggestions.

Assign command body:

```json
{
  "command_id": "UUIDv7",
  "assigned_to_employee_id": "manager-2",
  "assigned_by_employee_id": "manager-1",
  "reason": "operator reason"
}
```

Unassign command body:

```json
{
  "command_id": "UUIDv7",
  "unassigned_by_employee_id": "manager-1",
  "reason": "operator reason"
}
```

Реализовано сейчас: stop-list assignment endpoints возвращают только safe fields (`review_type`, `id`, `status`, assignment metadata) и не возвращают `payload_json`, raw Edge payload, PIN/token/secret/request dump. Catalog/recipe assignment controls запланированы далее. Вне текущего объема и запрещено: raw payload exposure. Запланировано далее: escalation, dashboard и большой UX refactor.

Для entities без подтвержденного `GET list` route UI показывает форму команды и поясняет, что list route не подтвержден.

реализовано сейчас: API client покрывает bounded inventory balance view `GET /api/v1/inventory/stock-balances`, bounded stock ledger preview `GET /api/v1/inventory/stock-ledger`, stop-list readiness, `GET /api/v1/olap/export-status?stream=raw_business_events|stock_moves`, `GET /api/v1/olap/stock-moves`, `GET /api/v1/olap/stock-move-summary`, `GET /api/v1/olap/backfill-jobs`, `GET /api/v1/olap/kitchen-timing-summary` и минимальный bounded preview `GET /api/v1/olap/sales-kitchen-summary`. Ответы валидируются через Zod, query формируется через bounded `limit/offset`, а UI не отображает raw payload fields. Запланировано далее: full costing/recalculation status, production auth/RBAC perimeter для mutating OLAP controls и richer BI endpoints; UI не вызывает изменяющие retry/backfill controls в текущем scope.

## Runtime Code

реализовано сейчас: runtime backend code изменялся для безопасного `GET /api/v1/sync/edge-events`, `GET /api/v1/sync/readiness/stop-list`, proposal review/apply, Edge-origin stop-list review/apply, ClickHouse `raw_business_events`, kitchen stock events, `StopListUpdated` projection/conflict policy и выравнивания accepted Edge event types со schema baseline.

реализовано сейчас: `cloud-ui/src/App.vue` оставляет orchestration/state/config, а presentation layer вынесен в `cloud-ui/src/components/cloud/*`.

реализовано сейчас: для запуска Cloud UI из браузера `cloud-backend` разрешает local CORS origin `http://localhost:5174`, `http://127.0.0.1:5174` и `http://host.docker.internal:5174`.

реализовано сейчас: `cloud-ui/package.json` содержит `dev`, `build`, `preview` и `test`; unit tests запускаются через Vitest, production build выполняется через `vue-tsc --noEmit && vite build`.
