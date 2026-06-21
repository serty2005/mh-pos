# Cloud API для RMS Manager reference UI

## Назначение

Документ перечисляет Cloud backend routes, нужные для полноценной реализации UI-UX из `docs/ui/myhoreca-rms-manager/` в активном `cloud-ui-g`.

Статусы:

- `реализовано сейчас` — route есть в Cloud backend или уже используется `cloud-ui-g`.
- `запланировано далее` — route нужен для референсного UI, но его нужно добавить или расширить.
- `вне текущего объема` — моковое или небезопасное поведение референса, которое нельзя переносить как runtime contract.

## Общие правила контрактов

реализовано сейчас:

- Base URL: `VITE_CLOUD_API_BASE`, default `http://localhost:8090/api/v1`.
- Ошибки возвращаются как безопасный envelope `{ error: { code, message_key, details, correlation_id } }`.
- UI-facing responses не возвращают raw sync payload, PIN, token, PIN hash, stack trace, SQL error или request dump.
- Мутирующие operator commands должны иметь `command_id` UUID v7, idempotency behavior, audit trail и backend RBAC/support permission.

запланировано далее:

- Все новые списочные routes должны поддерживать `limit`, `offset` и bounded defaults.
- Все restaurant-scoped routes принимают `restaurant_id`.
- Временные диапазоны для отчетов используют `business_date_from`, `business_date_to`; точные timestamps возвращаются ISO-8601 UTC.

## Уже используемые Cloud UI routes

реализовано сейчас:

- Restaurants: `GET/POST/PATCH /restaurants`, `POST /restaurants/{id}/archive`.
- Staff/RBAC для POS Edge: `/master-data/roles`, `/master-data/employees`.
- Staff/RBAC routes работают в tenant scope: role не содержит `restaurant_id`, employee содержит authoritative `restaurant_ids` и `all_restaurants`; `organization.manage` вычисляется backend и всегда охватывает все restaurants.
- Catalog/menu/modifiers/pricing/floor master-data: `/master-data/...`.
- Provisioning/pairing: `/devices/unassigned`, `/restaurants/{restaurant_id}/devices`, `/restaurants/{restaurant_id}/devices/{node_device_id}/assign`, `/devices/{node_device_id}/assignment-status`, `/restaurants/{restaurant_id}/devices/generate-pairing-code`, `/devices/pairing/consume`.
- Sync event log: `GET /sync/edge-events`.
- Automatic delivery status: реализовано сейчас как read-only `publication-state` и `GET /restaurants/{id}/master-data/delivery-status` без manual publish action. Delivery DTO содержит per-Edge Cloud/ACK version, lag, last sync, safe error code и retry metadata.

Полный список текущих route-backed endpoints зафиксирован в `docs/ui/CLOUD-UI-SPEC.md`.

## Операционный обзор и аналитика продаж

### GET `/reporting/sales-overview`

запланировано далее.

Нужно для KPI карточек референса: выручка, количество чеков, средний чек, margin/food cost.

Query:

- `restaurant_id`
- `business_date_from`
- `business_date_to`
- `compare_to_previous=true|false`

Response:

```json
{
  "restaurant_id": "uuid-v7",
  "business_date_from": "2026-06-15",
  "business_date_to": "2026-06-15",
  "currency": "RUB",
  "total_revenue_minor": 2488000,
  "order_count": 147,
  "average_check_minor": 16925,
  "gross_margin_basis_points": 6820,
  "previous_total_revenue_minor": 2214000,
  "previous_order_count": 133,
  "previous_average_check_minor": 16646,
  "previous_gross_margin_basis_points": 6710
}
```

Правила:

- `gross_margin_basis_points` можно возвращать только если backend имеет подтвержденные COGS/costing projections; иначе поле отсутствует.
- Frontend не считает финансовые итоги из raw events.

### GET `/reporting/sales-hourly`

запланировано далее.

Нужно для line chart выручки по часам.

Query:

- `restaurant_id`
- `business_date_from`
- `business_date_to`
- `timezone`

Response item:

```json
{
  "hour_local": "18:00",
  "revenue_minor": 426000,
  "order_count": 24
}
```

### GET `/reporting/sales-breakdown/categories`

запланировано далее.

Нужно для category bars.

Query:

- `restaurant_id`
- `business_date_from`
- `business_date_to`
- `limit`

Response item:

```json
{
  "category_id": "uuid-v7",
  "category_name": "Горячие блюда",
  "revenue_minor": 920000,
  "order_line_count": 82,
  "share_basis_points": 3698
}
```

### GET `/reporting/sales-breakdown/items`

запланировано далее.

Нужно для top dishes/items.

Query:

- `restaurant_id`
- `business_date_from`
- `business_date_to`
- `limit`

Response item:

```json
{
  "catalog_item_id": "uuid-v7",
  "menu_item_id": "uuid-v7",
  "name": "Стейк Рибай",
  "quantity": "18",
  "revenue_minor": 333000,
  "currency": "RUB"
}
```

### GET `/reporting/payment-methods`

запланировано далее.

Нужно для payment method distribution.

Response item:

```json
{
  "payment_method": "card",
  "amount_minor": 1492000,
  "payment_count": 91,
  "share_basis_points": 5997
}
```

## Реализованные read-only reporting routes, которые можно подключить в UI

реализовано сейчас:

- `GET /reporting/financial-operations`
- `GET /olap/sales-kitchen-summary`
- `GET /olap/kitchen-timing-summary`
- `GET /olap/stock-moves`
- `GET /olap/stock-move-summary`
- `GET /inventory/stock-ledger`
- `GET /inventory/stock-balances`
- `GET /inventory/recalculation-jobs`
- `GET /inventory/recalculation-jobs/{id}`
- `GET /sync/readiness/stop-list`

Эти routes полезны для отчетных и складских экранов, но не полностью заменяют sales KPI из референса: текущий `sales-kitchen-summary` не является полноценным revenue/check/payment-method API.

## Меню, категории, availability и tech cards

### GET `/master-data/menu/categories`

запланировано далее.

Сейчас `cloud-ui-g` имеет command-only `POST /master-data/menu/categories`. Для референсного menu UI нужен список категорий.

Query:

- `restaurant_id`
- `status=draft|published|archived`

Response item:

```json
{
  "id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "name": "Горячие блюда",
  "slug": "mains",
  "icon": "Soup",
  "sort_order": 10,
  "status": "published",
  "created_at": "2026-06-15T09:00:00Z",
  "updated_at": "2026-06-15T09:00:00Z"
}
```

### PATCH `/master-data/menu/categories/{id}`

запланировано далее.

Body:

```json
{
  "name": "Горячие блюда",
  "slug": "mains",
  "icon": "Soup",
  "sort_order": 10,
  "status": "published"
}
```

### POST `/master-data/menu/categories/{id}/archive`

запланировано далее.

Нужно для симметричного lifecycle category management.

### PATCH `/master-data/menu/items/{id}/availability`

запланировано далее.

Нужно для быстрых действий stop-list/menu availability без редактирования всей позиции.

Body:

```json
{
  "available": false,
  "available_quantity": "8",
  "reason": "Закончилась заготовка",
  "source": "cloud_manager"
}
```

Response: обновленный `menu_item` и связанный `stop_list_entry`, если backend создает stop-list overlay.

### Recipes/tech cards

запланировано далее:

- `GET /master-data/recipes?restaurant_id=&catalog_item_id=&status=`
- `POST /master-data/recipes`
- `PATCH /master-data/recipes/{id}`
- `POST /master-data/recipes/{id}/activate`
- `POST /master-data/recipes/{id}/archive`

Минимальный response:

```json
{
  "id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "recipe_owner_catalog_item_id": "uuid-v7",
  "version_name": "Летнее меню 2026",
  "status": "active",
  "date_from": "2026-05-01",
  "instructions": "Технология приготовления",
  "lines": [
    {
      "component_catalog_item_id": "uuid-v7",
      "quantity": "0.350",
      "unit": "kg",
      "loss_percent": "4.5",
      "cost_per_unit_minor": 58000
    }
  ],
  "created_at": "2026-06-15T09:00:00Z",
  "updated_at": "2026-06-15T09:00:00Z"
}
```

## Stop-list

### GET `/inventory/stop-list`

запланировано далее.

Нужно для раздела `Активные стоп-листы`.

Query:

- `restaurant_id`
- `active=true|false`
- `catalog_item_id`
- `source`
- `limit`
- `offset`

Response item:

```json
{
  "id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "warehouse_id": "warehouse-main",
  "catalog_item_id": "uuid-v7",
  "menu_item_id": "uuid-v7",
  "item_name": "Суп Том Ям",
  "available_quantity": "8",
  "unit_code": "portion",
  "active": true,
  "source": "cloud_manager",
  "reason": "Лимит заготовки",
  "conflict_policy": "edge_overlay_requires_manager_review",
  "cloud_version": 14,
  "updated_at": "2026-06-15T09:00:00Z"
}
```

### POST `/inventory/stop-list`

запланировано далее.

Body:

```json
{
  "command_id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "warehouse_id": "warehouse-main",
  "catalog_item_id": "uuid-v7",
  "available_quantity": "8",
  "active": true,
  "source": "cloud_manager",
  "reason": "Лимит заготовки"
}
```

### PATCH `/inventory/stop-list/{id}`

запланировано далее.

Body поддерживает `available_quantity`, `active`, `reason`, `conflict_policy`.

### POST `/inventory/stop-list/{id}/deactivate`

запланировано далее.

Body:

```json
{
  "command_id": "uuid-v7",
  "reason": "Позиция снова доступна"
}
```

Правила:

- Изменение stop-list должно попадать в `inventory_reference` publication stream или отдельный sync package.
- Edge overlay conflicts не скрываются: UI показывает conflict/readiness через `GET /sync/readiness/stop-list`.

## Залы и столы как редактор схемы

реализовано сейчас:

- `GET/POST/PATCH/archive /master-data/floor/halls`
- `GET/POST/PATCH/archive /master-data/floor/tables`

запланировано далее:

### PATCH `/master-data/floor/tables/{id}/layout`

Нужно, чтобы референсный drag/layout editor сохранял координаты и форму стола.

Body:

```json
{
  "x_percent": "42.5",
  "y_percent": "31.0",
  "shape": "rectangle",
  "rotation_degrees": 0
}
```

### GET `/floor/table-runtime-state`

Нужно только для manager overview, не как авторитетный order state editor.

Query:

- `restaurant_id`
- `business_date_local`

Response item:

```json
{
  "table_id": "uuid-v7",
  "status": "free",
  "current_order_id": "uuid-v7",
  "waiter_employee_id": "uuid-v7",
  "updated_at": "2026-06-15T09:00:00Z"
}
```

## Персонал и доступ

реализовано сейчас:

- Role CRUD, employee lifecycle, role assignment и PIN rotation.

запланировано далее:

### GET `/staff/shift-presence`

Нужно для статусов `on_shift`, `shiftStart` из референса.

Query:

- `restaurant_id`
- `business_date_local`

Response item:

```json
{
  "employee_id": "uuid-v7",
  "shift_id": "uuid-v7",
  "status": "on_shift",
  "opened_at": "2026-06-15T07:00:00Z",
  "closed_at": null
}
```

### Cloud operator RBAC

запланировано далее:

- `GET /cloud-operators`
- `POST /cloud-operators`
- `PATCH /cloud-operators/{id}`
- `POST /cloud-operators/{id}/suspend`
- `GET /cloud-roles`
- `POST /cloud-roles`
- `PATCH /cloud-roles/{id}`

Это отдельный контур от POS Edge employee permissions. До появления production auth/RBAC perimeter не использовать frontend visibility как security boundary.

## Edge devices, периферия и ККТ

реализовано сейчас:

- Unassigned/assignment/pairing routes для Edge node onboarding.
- `GET /restaurants/{restaurant_id}/devices` для restaurant-owned Edge nodes после назначения.
- `POST /devices/pairing/consume` выполняется Edge-стороной после resolve в License Server и только на этом шаге назначает Edge node ресторану.
- `GET /sync/edge-events` для safe event log.
- `GET /provisioning/master-data/{stream}?node_device_id=...` для просмотра metadata отправленных Cloud -> Edge packages без раскрытия raw sync event payload.

### GET `/restaurants/{restaurant_id}/devices`

реализовано сейчас.

Возвращает Edge nodes, которые уже принадлежат выбранному ресторану. Этот список отделен от server-owned pending устройств из `GET /devices/unassigned`.

Response item:

```json
{
  "id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "node_device_id": "edge-msk-01",
  "display_name": "POS Edge Node",
  "status": "assigned",
  "last_seen_at": "2026-06-15T09:00:00Z",
  "assigned_at": "2026-06-15T09:00:00Z",
  "created_at": "2026-06-15T09:00:00Z",
  "updated_at": "2026-06-15T09:00:00Z"
}
```

### POST `/devices/pairing/consume`

реализовано сейчас.

Используется POS Edge после ввода pairing code и resolve в License Server. Cloud UI не вызывает этот endpoint.

Request:

```json
{
  "pairing_id": "uuid-v7",
  "nonce": "base64url",
  "ciphertext": "base64url"
}
```

`ciphertext` содержит `node_device_id`, `display_name`, `app_version` и `request_id`, зашифрованные AES-GCM ключом, выведенным из pairing code и `pairing_id`. Успешный consume создает/обновляет restaurant-owned Edge node, помечает pending row как assigned и возвращает bootstrap metadata, snapshot URL и node credentials.

запланировано далее:

- cloud-side rebind intent для перепривязки устройства к другому ресторану;
- Edge backup acknowledgment до destructive reset/rebootstrap;
- restore backup path при повторной привязке к тому же ресторану.

### GET `/edge/devices`

Нужно для экрана `Edge-терминалы, периферия и ККТ`.

Query:

- `restaurant_id`
- `type=pos_terminal|printer|kds|fiscal_device`
- `status=online|offline|warning|error`
- `limit`
- `offset`

Response item:

```json
{
  "id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "node_device_id": "edge-msk-01",
  "display_name": "Основной фискальный регистратор",
  "type": "fiscal_device",
  "status": "online",
  "ip_address": "192.168.1.180",
  "port": "5560",
  "firmware": "f-v5.8.1",
  "serial_number": "SN-778401928",
  "paper_level_percent": 85,
  "fn_expires_at": "2027-11-10",
  "last_seen_at": "2026-06-15T09:00:00Z"
}
```

### GET `/edge/devices/{id}/health`

запланировано далее.

Response:

```json
{
  "device_id": "uuid-v7",
  "status": "online",
  "latency_ms": 45,
  "last_check_at": "2026-06-15T09:00:00Z",
  "checks": [
    { "code": "network", "status": "ok", "message_key": "edge.health.network.ok" }
  ]
}
```

### POST `/edge/devices/{id}/diagnostics/ping`

запланировано далее.

Body:

```json
{
  "command_id": "uuid-v7",
  "requested_by": "cloud-manager"
}
```

### POST `/edge/devices/{id}/diagnostics/test-print`

запланировано далее.

Только для printer/fiscal device, с idempotency, audit log и RBAC/support permission.

## Sync operations

реализовано сейчас:

- `GET /sync/edge-events`
- `POST /sync/edge-events`
- `POST /sync/edge-events/batch`
- `POST /sync/exchange`

запланировано далее:

### GET `/sync/status`

Нужно для dashboard карточек по терминалам.

Query:

- `restaurant_id`
- `node_device_id`

Response item:

```json
{
  "node_device_id": "edge-msk-01",
  "display_name": "Терминал Бар-Касса",
  "status": "online",
  "last_sync_at": "2026-06-15T09:00:00Z",
  "pending_edge_event_count": 0,
  "failed_event_count": 0,
  "last_cloud_package_version": 18
}
```

### GET `/sync/problem-events`

запланировано далее.

Нужно для operator troubleshooting без raw payload.

Response item:

```json
{
  "id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "node_device_id": "edge-msk-01",
  "event_type": "CheckClosed",
  "error_code": "VALIDATION_FAILED",
  "message_key": "errors.sync.invalidEnvelope",
  "retryable": false,
  "created_at": "2026-06-15T09:00:00Z"
}
```

### POST `/sync/problem-events/{id}/retry`

запланировано далее.

Body:

```json
{
  "command_id": "uuid-v7",
  "reason": "Manual retry after mapping fix"
}
```

## Warehouse/TTN

реализовано сейчас:

- Read-only ledger/balances/recalculation diagnostics:
  - `GET /inventory/stock-ledger`
  - `GET /inventory/stock-balances`
  - `GET /inventory/recalculation-jobs`
  - `GET /inventory/recalculation-jobs/{id}`

запланировано далее:

### GET `/inventory/warehouses`

Response item:

```json
{
  "id": "warehouse-main",
  "restaurant_id": "uuid-v7",
  "name": "Основной склад",
  "status": "active"
}
```

### GET `/inventory/documents`

Query:

- `restaurant_id`
- `warehouse_id`
- `type=receipt|write_off|inventory_count|production|ttn`
- `status=draft|posted|cancelled`
- `business_date_from`
- `business_date_to`

Response item:

```json
{
  "id": "uuid-v7",
  "restaurant_id": "uuid-v7",
  "warehouse_id": "warehouse-main",
  "type": "receipt",
  "status": "posted",
  "business_date_local": "2026-06-15",
  "document_number": "REC-42",
  "created_by_employee_id": "uuid-v7",
  "posted_at": "2026-06-15T09:00:00Z"
}
```

### POST `/inventory/documents`

запланировано далее.

Body содержит `command_id`, document header и lines.

### PATCH `/inventory/documents/{id}`

запланировано далее.

Только для `draft`.

### POST `/inventory/documents/{id}/post`

запланировано далее.

Posting создает ledger движения async/safely; не должен выполнять тяжелый costing rebuild в HTTP request path.

### POST `/inventory/documents/{id}/cancel`

запланировано далее.

Только с audit reason и безопасной reversal policy.

### TTN routes

запланировано далее:

- `GET /inventory/ttn`
- `POST /inventory/ttn/import`
- `POST /inventory/ttn/{id}/accept`
- `POST /inventory/ttn/{id}/reject`

## Publications/readiness для расширенных stream

реализовано сейчас:

- Publication workflow уже поддерживает streams `recipes`, `inventory_reference`, `currencies` на уровне package/publication foundation.

запланировано далее:

### GET `/restaurants/{id}/master-data/readiness`

Единый агрегат для dashboard вместо множества UI-side запросов.

Response:

```json
{
  "restaurant_id": "uuid-v7",
  "checks": [
    { "key": "staff", "ready": true, "count": 12 },
    { "key": "floor", "ready": true, "count": 24 },
    { "key": "stop_list", "ready": true, "count": 3 }
  ],
  "last_publication_version": 18,
  "last_publication_at": "2026-06-15T09:00:00Z"
}
```

## Что не переносить из референса как backend contract

вне текущего объема:

- `simulateSale`, случайная генерация чеков и изменение stock из frontend.
- Симуляция отключения терминала только через локальный React state.
- Очистка sync log в браузере без backend audit.
- Ping/test print без idempotency, RBAC и audit.
- Frontend-calculated authoritative revenue, margin, stock balance или order/payment state.

## Можно ли подключить референс напрямую

Короткий ответ: нет, не только заменой mock data и TypeScript types.

Можно использовать референс как визуальный исходник и частично как набор React components, но перед прямым подключением к Cloud backend нужно:

- заменить `App.tsx` mock-state orchestration на route-backed data layer через `VITE_CLOUD_API_BASE`;
- удалить `mockData.ts`, `simulateSale`, `triggerForceSync`, `handleSimulateOutage`, `handlePrintTest` и другие browser-only симуляторы;
- заменить reference types на Zod-валидируемые DTO из `cloud-ui-g/src/shared/api/schemas.ts`;
- разнести экраны по Cloud routes и restaurant scope, как в `cloud-ui-g/src/app/navigation.ts`;
- перевести все пользовательские строки в `cloud-ui-g/src/shared/i18n/ru.ts`;
- заменить hardcoded restaurant list на `GET /restaurants`;
- заменить menu category/tech card/stop-list/warehouse/analytics/device screens на реальные API из этой спецификации;
- сохранить safe error handling: не показывать raw backend errors, raw payload, PIN/token material;
- добавить loading/empty/error states для каждого route-backed экрана;
- добавить tests/build checks для форм и API schema parsing.

Практический путь: продолжать переносить UI-слои из референса по разделам в `cloud-ui-g`, а не монтировать reference app целиком. Прямой fork референса станет крупной переписью data layer и безопасности, тогда как текущий `cloud-ui-g` уже имеет правильный API client, i18n, scopes и безопасные ограничения.
