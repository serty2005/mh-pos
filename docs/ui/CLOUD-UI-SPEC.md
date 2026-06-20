# Cloud UI Spec

## Назначение

реализовано сейчас: активный Cloud-бэкофис находится в `cloud-ui-g`.

`cloud-ui-g` является отдельным React/Vite/TypeScript интерфейсом для `cloud-backend` и Cloud-owned операционных сценариев. Он не является частью `pos-ui`, не использует POS session, POS Edge runtime endpoints, cashier routes или локальные POS stores.

`cloud-ui` на Vue/Quasar признан устаревшим и удален из runtime tree. Дальнейшая разработка Cloud-бэкофиса идет только в `cloud-ui-g`.

## Статус Каталогов

реализовано сейчас:

- `cloud-ui-g` — активный Cloud UI runtime target.
- `cloud-ui-g/package.json` содержит `dev`, `build`, `preview`, `clean`, `lint` и `test`.
- `cloud-ui-g` использует `VITE_CLOUD_API_BASE`; default из `.env.example` — `http://localhost:8090/api/v1`.
- `cloud-ui` удален; старые Vue/Quasar scripts больше не являются частью проверок.

запланировано далее:

- переносить нужные manager-facing сценарии в `cloud-ui-g` только после сверки с текущими backend routes/DTO;
- поддерживать документацию по активному Cloud UI вокруг `cloud-ui-g`.

вне текущего объема:

- новые Cloud UI фичи вне `cloud-ui-g`;
- перенос React/Vite экранов обратно в Vue/Quasar.

## Реализовано Сейчас В `cloud-ui-g`

реализовано сейчас:

- Shell `CloudManagerApp` с restaurant selector, sidebar navigation и route scopes `global|restaurant`.
- Route-backed разделы:
  - `dashboard`;
  - `restaurants`;
  - `edge-sync`;
  - `catalog`;
  - `menu`;
  - `modifiers`;
  - `pricing-taxes`;
  - `staff-permissions`;
  - `floor`;
  - `publications`.
- Navigation placeholders `inventory` и `reports` существуют, но пока показывают blocked section и не считаются реализованными runtime screens.
- Dashboard readiness проверяет наличие roles/employees, halls/tables, catalog items, menu items, modifiers/pricing, Edge assignment и publication.
- Edge sync показывает server-owned pending devices, restaurant-owned assigned devices, assignment status, pairing code generation, safe Edge events list по выбранному устройству и metadata по отправленным Cloud -> Edge master-data packages без раскрытия raw payload.
- Restaurants раздел управляет restaurant records.
- Staff/permissions раздел управляет tenant-level POS Edge roles/employees, role assignment, employee restaurant memberships, status, PIN rotation и POS permission profiles/matrix; `organization.manage` отображается как доступ ко всем restaurants без отдельных галок; это не Cloud operator RBAC и не production Cloud authorization boundary.
- Catalog раздел управляет catalog items, folders, folder parameters, tags и command-only item tag assignment.
- Menu раздел управляет menu items и command-only menu category create.
- Modifiers раздел управляет modifier groups, options и bindings.
- Pricing/taxes раздел управляет pricing policies и package `pricing_policy` через provisioning route.
- Floor раздел управляет halls/tables и показывает preview зала.
- Publications раздел читает publication state и выполняет publish master data.
- Это текущее, но устаревающее поведение: целевой Cloud UI удаляет Publish action и заменяет Publications экран read-only состоянием автоматической доставки по Edge — Cloud version, acknowledged Edge version, lag и safe error.
- API client валидирует responses через Zod-схемы и использует bounded/safe query там, где route это поддерживает.
- Пользовательские строки находятся в `cloud-ui-g/src/shared/i18n` и `cloud-ui-g/src/i18n`; новые строки не должны добавляться hardcoded в components.

## Подтвержденные API Routes В `cloud-ui-g`

реализовано сейчас:

- `GET /api/v1/restaurants`
- `POST /api/v1/restaurants`
- `PATCH /api/v1/restaurants/{id}`
- `POST /api/v1/restaurants/{id}/archive`
- `GET /api/v1/master-data/roles`
- `POST /api/v1/master-data/roles`
- `PATCH /api/v1/master-data/roles/{id}`
- `POST /api/v1/master-data/roles/{id}/archive`
- `GET /api/v1/master-data/employees`
- `POST /api/v1/master-data/employees`
- `PATCH /api/v1/master-data/employees/{id}`
- `POST /api/v1/master-data/employees/{id}/suspend`
- `POST /api/v1/master-data/employees/{id}/activate`
- `POST /api/v1/master-data/employees/{id}/archive`
- `POST /api/v1/master-data/employees/{id}/role`
- `POST /api/v1/master-data/employees/{id}/pin`
- `GET /api/v1/master-data/catalog/items?restaurant_id=...`
- `POST /api/v1/master-data/catalog/items`
- `PATCH /api/v1/master-data/catalog/items/{id}`
- `POST /api/v1/master-data/catalog/items/{id}/archive`
- `GET /api/v1/master-data/catalog/folders?restaurant_id=...`
- `POST /api/v1/master-data/catalog/folders`
- `PATCH /api/v1/master-data/catalog/folders/{id}`
- `POST /api/v1/master-data/catalog/folders/{id}/archive`
- `GET /api/v1/master-data/catalog/folder-parameters?restaurant_id=...`
- `POST /api/v1/master-data/catalog/folder-parameters`
- `PATCH /api/v1/master-data/catalog/folder-parameters/{id}`
- `GET /api/v1/master-data/catalog/tags?restaurant_id=...`
- `POST /api/v1/master-data/catalog/tags`
- `PATCH /api/v1/master-data/catalog/tags/{id}`
- `POST /api/v1/master-data/catalog/item-tags`
- `GET /api/v1/master-data/menu/items?restaurant_id=...`
- `POST /api/v1/master-data/menu/items`
- `PATCH /api/v1/master-data/menu/items/{id}`
- `POST /api/v1/master-data/menu/items/{id}/archive`
- `POST /api/v1/master-data/menu/categories`
- `GET /api/v1/master-data/modifiers/groups?restaurant_id=...`
- `POST /api/v1/master-data/modifiers/groups`
- `PATCH /api/v1/master-data/modifiers/groups/{id}`
- `GET /api/v1/master-data/modifiers/options?restaurant_id=...`
- `POST /api/v1/master-data/modifiers/options`
- `PATCH /api/v1/master-data/modifiers/options/{id}`
- `GET /api/v1/master-data/modifiers/bindings?restaurant_id=...`
- `POST /api/v1/master-data/modifiers/bindings`
- `PATCH /api/v1/master-data/modifiers/bindings/{id}`
- `GET /api/v1/master-data/pricing/policies?restaurant_id=...`
- `POST /api/v1/master-data/pricing/policies`
- `PATCH /api/v1/master-data/pricing/policies/{id}`
- `GET /api/v1/provisioning/master-data/pricing_policy?node_device_id=...`
- `PUT /api/v1/provisioning/master-data/pricing_policy`
- `GET /api/v1/master-data/floor/halls?restaurant_id=...`
- `POST /api/v1/master-data/floor/halls`
- `PATCH /api/v1/master-data/floor/halls/{id}`
- `POST /api/v1/master-data/floor/halls/{id}/archive`
- `GET /api/v1/master-data/floor/tables?restaurant_id=...`
- `POST /api/v1/master-data/floor/tables`
- `PATCH /api/v1/master-data/floor/tables/{id}`
- `POST /api/v1/master-data/floor/tables/{id}/archive`
- `GET /api/v1/devices/unassigned`
- `GET /api/v1/restaurants/{restaurant_id}/devices`
- `POST /api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign`
- `GET /api/v1/devices/{node_device_id}/assignment-status`
- `POST /api/v1/restaurants/{restaurant_id}/devices/generate-pairing-code`
- `POST /api/v1/devices/pairing/consume`
- `GET /api/v1/sync/edge-events?restaurant_id=&limit=`
- `GET /api/v1/restaurants/{id}/master-data/publication-state`
- `POST /api/v1/restaurants/{id}/master-data/publish`

`POST .../publish` используется только текущим runtime и должен быть удален из пользовательского flow после реализации automatic delivery. UI не вызывает его и не предлагает manual checkpoint.

## Удаленный Legacy `cloud-ui`

реализовано сейчас:

- Удаленный Vue/Quasar `cloud-ui` исторически содержал более широкий manager-facing набор экранов: financial operations, recipe versions, proposal review, inventory readiness, OLAP export/read-only slices, sale preparation links и sales/kitchen summary.
- Источником истины для переноса остаются текущие backend routes/DTO, docs и активный `cloud-ui-g`, а не удаленный runtime.
- Наличие исторического экрана в старом `cloud-ui` не означает, что такой экран реализован в активном `cloud-ui-g`.

запланировано далее:

- переносить только нужные legacy-сценарии в `cloud-ui-g`, обновляя API client, Zod schemas, i18n и тесты активного React приложения.

вне текущего объема:

- поддерживать два равноправных Cloud UI runtime;
- добавлять новые Cloud UI features вне `cloud-ui-g`.

## UX И Безопасность

реализовано сейчас:

- `cloud-ui-g` работает в local pilot perimeter через CORS origins `5174`.
- Выбор ресторана обязателен для restaurant-scoped разделов.
- Обычный оператор не должен вводить UUID вручную там, где UI может выбрать сущность из загруженных справочников.
- Pairing code показывается только как одноразовый секрет, возвращенный backend.
- PIN create/rotate flows используют password input; списки сотрудников показывают только `pin_configured` и credential version.
- Error UI показывает safe message/support context и не раскрывает raw payload, PIN material, token material, SQL errors, stack traces или request dumps.

запланировано далее:

- при переносе inventory/reporting/proposal review из legacy UI сохранять read-only/no-raw-payload границы;
- расширять Cloud UI только поверх подтвержденных backend routes.

вне текущего объема:

- Cashier runtime, KDS runtime screens, POS order/payment/check/precheck flows, PSP и fiscalization в Cloud UI.
- Cloud auth/RBAC UI до появления подтвержденного backend-контракта.
- BI dashboard, charts, COGS/margin и mutating OLAP retry/backfill controls в активном `cloud-ui-g`.
