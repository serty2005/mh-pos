# MyHoReCa Cloud UI

`cloud-ui` — устаревший Vue/Quasar frontend для управления `cloud-backend`.

## Статус разработки

реализовано сейчас:

- код каталога сохранен как legacy/reference-only runtime surface для уже реализованных Cloud UI сценариев;
- build и unit tests остаются доступными для проверки существующего кода.

запланировано далее:

- все новые правки Cloud-бэкофиса выполняются в `cloud-ui-g`.

вне текущего объема:

- развитие новых Cloud UI сценариев в `cloud-ui`;
- перенос новых React/Vite экранов обратно в Vue/Quasar.

## Статус

реализовано сейчас:

- отдельный Vite/Vue/Quasar модуль, не связанный с `pos-ui`;
- работа с Cloud master-data API на `http://localhost:8090/api/v1` по умолчанию;
- схематичный admin-style интерфейс для ресторанов, персонала, каталога, модификаторов, pricing policies, зала, menu items и публикации master data;
- только подтвержденные операции из `cloud-backend/internal/masterdata/api/router.go`.

вне текущего объема:

- KDS;
- PSP;
- fiscalization;
- inventory runtime;
- recipe consumption;
- delivery;
- cashier runtime и любые POS Edge операции.

## Запуск

```powershell
cd cloud-ui
npm install
npm run dev
```

Открой `http://localhost:5174`.

Для активной разработки Cloud-бэкофиса используй `cloud-ui-g`.

## Скрипты

Реализовано сейчас:

- `npm run dev` - Vite dev server для Cloud admin UI.
- `npm run build` - `vue-tsc --noEmit` и production build.
- `npm run preview` - локальный preview production build.

Реализовано сейчас:

- `npm run test` - unit tests через Vitest.

Для другого Cloud API:

```powershell
$env:VITE_CLOUD_API_BASE="http://localhost:8090/api/v1"
npm run dev
```

## Подтвержденные Cloud API routes

реализовано сейчас:

- `POST /api/v1/restaurants`
- `GET /api/v1/restaurants`
- `GET /api/v1/restaurants/{id}`
- `PATCH /api/v1/restaurants/{id}`
- `POST /api/v1/restaurants/{id}/archive`
- `POST /api/v1/roles`
- `GET /api/v1/roles?restaurant_id=...`
- `GET /api/v1/roles/{id}`
- `PATCH /api/v1/roles/{id}`
- `POST /api/v1/roles/{id}/archive`
- `POST /api/v1/employees`
- `GET /api/v1/employees?restaurant_id=...`
- `GET /api/v1/employees/{id}`
- `PATCH /api/v1/employees/{id}`
- `POST /api/v1/employees/{id}/suspend`
- `POST /api/v1/employees/{id}/activate`
- `POST /api/v1/employees/{id}/archive`
- `POST /api/v1/master-data/employees/{id}/role`
- `POST /api/v1/employees/{id}/pin`
- `POST /api/v1/catalog/items`
- `GET /api/v1/catalog/items?restaurant_id=...`
- `GET /api/v1/catalog/items/{id}`
- `PATCH /api/v1/catalog/items/{id}`
- `POST /api/v1/catalog/items/{id}/archive`
- `POST /api/v1/master-data/catalog/folders`
- `GET /api/v1/master-data/catalog/folders?restaurant_id=...`
- `PATCH /api/v1/master-data/catalog/folders/{id}`
- `POST /api/v1/master-data/catalog/folders/{id}/archive`
- `POST /api/v1/master-data/catalog/folder-parameters`
- `GET /api/v1/master-data/catalog/folder-parameters?restaurant_id=...`
- `PATCH /api/v1/master-data/catalog/folder-parameters/{id}`
- `POST /api/v1/master-data/catalog/tags`
- `GET /api/v1/master-data/catalog/tags?restaurant_id=...`
- `PATCH /api/v1/master-data/catalog/tags/{id}`
- `POST /api/v1/master-data/catalog/item-tags`
- `POST /api/v1/master-data/modifiers/groups`
- `GET /api/v1/master-data/modifiers/groups?restaurant_id=...`
- `PATCH /api/v1/master-data/modifiers/groups/{id}`
- `POST /api/v1/master-data/modifiers/options`
- `GET /api/v1/master-data/modifiers/options?restaurant_id=...`
- `PATCH /api/v1/master-data/modifiers/options/{id}`
- `POST /api/v1/master-data/modifiers/bindings`
- `GET /api/v1/master-data/modifiers/bindings?restaurant_id=...`
- `PATCH /api/v1/master-data/modifiers/bindings/{id}`
- `POST /api/v1/master-data/pricing/policies`
- `GET /api/v1/master-data/pricing/policies?restaurant_id=...`
- `PATCH /api/v1/master-data/pricing/policies/{id}`
- `POST /api/v1/halls`
- `GET /api/v1/halls?restaurant_id=...`
- `PATCH /api/v1/halls/{id}`
- `POST /api/v1/halls/{id}/archive`
- `POST /api/v1/tables`
- `GET /api/v1/tables?restaurant_id=...`
- `PATCH /api/v1/tables/{id}`
- `POST /api/v1/tables/{id}/archive`
- `POST /api/v1/menu/items`
- `GET /api/v1/menu/items?restaurant_id=...`
- `GET /api/v1/menu/items/{id}`
- `PATCH /api/v1/menu/items/{id}`
- `POST /api/v1/menu/items/{id}/archive`
- `POST /api/v1/master-data/menu/categories`
- `POST /api/v1/restaurants/{id}/master-data/publish`
- `GET /api/v1/restaurants/{id}/master-data/publication-state`

запланировано далее:

- отдельная авторизация Cloud UI и Cloud RBAC после появления подтвержденного backend-контракта.
