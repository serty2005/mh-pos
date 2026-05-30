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
7. Route-backed manager review для catalog/recipe suggestions и Edge-origin stop-list updates, readiness-only поверхности для inventory operations/costing и OLAP exports без неподтвержденных команд.

запланировано далее:

- вывести связи `catalog item -> menu item -> modifier bindings -> pricing policies` как единый сценарий подготовки продажи;
- показывать версии опубликованного пакета и состояние доставки на Edge, когда backend подтвердит такой контракт.
- до полного пилота превратить readiness-only manager surfaces для inventory operations, costing status, ClickHouse export readiness и OLAP API diagnostics в runtime только после появления подтвержденных Cloud backend routes;

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
- recipe items через `/api/v1/master-data/recipes/items`;
- сценарный editor версий техкарт через `/api/v1/master-data/recipes/versions`, `/api/v1/master-data/recipes/versions/drafts`, `/api/v1/master-data/recipes/versions/{id}/submit`;
- stop-list entries через `/api/v1/master-data/inventory/stop-list`;
- route-backed раздел `Очередь предложений` для Cloud review workflow (`catalog-suggestions`/`recipe-suggestions`/`manager/stop-list-updates`) со списками catalog/recipe suggestions и Edge-origin stop-list updates, detail/diff view, approve/reject/request-changes actions, linked new dish + recipe group display и publication/readiness signal после approve; раздел `Готовность склада` читает `GET /api/v1/sync/readiness/stop-list` для safe stop-list/publication/Edge ACK/sync problem summary; OLAP exports остается readiness-only, хотя backend уже имеет bounded `GET /api/v1/olap/stock-moves` без UI-превью в текущем scope;
- halls и tables;
- menu items;
- menu category create как command-only операция, потому что list/update routes не подтверждены;
- publication summary и явная публикация master data;
- опубликованный snapshot для Edge использует backend Cloud -> POS Edge ingest DTO: top-level modifier groups/options/bindings передаются отдельно от link-only `menu_item_modifier_groups`, без rich/UI projection fields внутри `menu_items`;
- `GET /api/v1/restaurants/{id}/master-data/publication-state` возвращает `200 null` до первой публикации выбранного ресторана; Cloud UI трактует это как empty state панели публикации, а не как ошибку browser console;
- отдельный раздел `События от Edge`, который читает `GET /api/v1/sync/edge-events` и показывает только безопасные receipt metadata/checksum без raw payload; на narrow screens таблица заменяется card/list fallback с теми же безопасными полями;
- resource lists на narrow screens переходят с широких таблиц на карточки, где статус выводится тем же safe status label, а не raw payload или POS runtime detail.

запланировано до полного пилота:

- recipe editor имеет bounded route-backed строки recipe items и реализовано сейчас сценарный editor версий: просмотр текущих версий, draft form с компонентной строкой, save draft / submit, review выполняется в существующей очереди предложений;
- duplicate hints и linked receipt line для catalog suggestion review остаются запланировано далее;
- stop-list panel уже имеет bounded route-backed rows; `Готовность склада` показывает default conflict policy, async projection mode, publication/package metadata, latest Edge ACK metadata и sync problem counters без raw payload; Cloud review queue показывает safe Edge-origin stop-list update summary/diff и approve/reject/request-changes без raw payload. Production-grade assignment/escalation workflow остается запланирован далее;
- реализовано сейчас: inventory readiness surface показывает route-backed `stock-balances` table из Cloud backend с фильтрами `warehouse_id`, `catalog_item_id`, `business_date_to`, `costing_status`, aggregate costing status и readiness signals stop-list без raw payload. Edge-side stock receipts, inventory counts, write-offs and production input are covered by `pos-ui-g` kitchen mode and Cloud ledger/balance read endpoints;
- запланировано далее: stock documents table, full costing/recalculation operator workflow и inventory runtime actions в Cloud UI;
- ClickHouse/OLAP workspace: backend уже имеет read-only export status, bounded stock moves и stock move summary; UI runtime preview, retry/backfill mutation controls и richer analytics остаются запланировано далее;
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
- раздел `Очередь предложений` не выводит raw `payload_json`: detail/diff строится только по whitelist полям catalog proposal (`kind`, `name`, `sku`, `base_unit`, `kitchen_type`, `accounting_category`) и recipe proposal changes (`action`, `from_catalog_item_id`, `to_catalog_item_id`, `quantity`, `unit_code`, `loss_percent`); PIN/token/secret/request dump не отображаются;
- approve/reject/request-changes формы отправляют только `reviewed_by_employee_id`, optional `review_comment` и optional `published_by`; после approve UI перечитывает `publication-state`, потому что apply/publish выполняет backend;
- UX ориентиры полного пилота зафиксированы в `docs/ui/PILOT-UX-MARKET-REVIEW.md`; business workflows не должны требовать ручного ввода UUID/raw JSON для обычного менеджерского сценария;
- пользовательские тексты идут через `vue-i18n`.

## API

реализовано сейчас: API client `cloud-ui/src/shared/api.ts` использует подтвержденные routes из:

- `cloud-backend/internal/provisioning/api/router.go` для Edge-device provisioning;
- `cloud-backend/internal/masterdata/api/router.go` для master data и публикации;
- `cloud-backend/internal/cloudsync/api/router.go` для безопасного списка входящих Edge events.

Реализовано сейчас для `Очередь предложений`:

- `GET /api/v1/master-data/catalog-suggestions?restaurant_id=&status=&limit=&offset=`;
- `GET /api/v1/master-data/recipe-suggestions?restaurant_id=&status=&limit=&offset=`;
- `POST /api/v1/master-data/catalog-suggestions/{id}/approve|reject|request-changes`;
- `POST /api/v1/master-data/recipe-suggestions/{id}/approve|reject|request-changes`.
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

Для entities без подтвержденного `GET list` route UI показывает форму команды и поясняет, что list route не подтвержден.

реализовано сейчас: API client покрывает bounded inventory balance view `GET /api/v1/inventory/stock-balances` и stop-list readiness. Запланировано до полного пилота: API client должен покрыть full costing/recalculation status, подтвержденные ClickHouse export status/stock move summary/`olap_stock_moves` preview endpoints и production-grade operator flows; UI не должен вызывать неподтвержденные mutating retry/backfill или BI endpoints до появления backend contract.

## Runtime Code

реализовано сейчас: runtime backend code изменялся для безопасного `GET /api/v1/sync/edge-events`, `GET /api/v1/sync/readiness/stop-list`, proposal review/apply, Edge-origin stop-list review/apply, ClickHouse `raw_business_events`, kitchen stock events, `StopListUpdated` projection/conflict policy и выравнивания accepted Edge event types со schema baseline.

реализовано сейчас: `cloud-ui/src/App.vue` оставляет orchestration/state/config, а presentation layer вынесен в `cloud-ui/src/components/cloud/*`.

реализовано сейчас: для запуска Cloud UI из браузера `cloud-backend` разрешает local CORS origin `http://localhost:5174`, `http://127.0.0.1:5174` и `http://host.docker.internal:5174`.

реализовано сейчас: `cloud-ui/package.json` содержит `dev`, `build` и `preview`; отдельный `test` script не заявлен, поэтому проверка Cloud UI в текущем scope выполняется через `npm run build`, если тестовая инфраструктура не добавляется отдельной задачей.
