# Cloud UI G React Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Превратить `cloud-ui-g` из Google AI Studio React-прототипа с моками в production-ready менеджерский Cloud UI для RMS/POS, подключенный к реализованным `cloud-backend` API.

**Architecture:** Целевой UI остается отдельным Vite/React приложением в `cloud-ui-g`, использует только Cloud Backend endpoints под `/api/v1`, typed API client с `zod`, локализованные UI-строки и route-backed workflows. Старый `cloud-ui` на Vue/Quasar используется как источник подтвержденных контрактов и UX-инвариантов, но не как runtime target.

**Tech Stack:** React 19, Vite, Tailwind CSS v4, TypeScript, `zod`, native Fetch API, локальный i18n слой, Playwright для smoke/visual QA.

---

## Основной Заголовок Для Агентского Промпта

```markdown
# RMS Cloud Manager React Migration: сделать cloud-ui-g полноценным менеджерским Cloud UI
```

## Общий Промпт-Контекст Для Каждого Агента

Перед каждым итерационным промптом добавляй этот блок. Он задает границы проекта и не должен сокращаться.

```markdown
Ты работаешь в репозитории `c:\safe\repos\myhoreca-pos`.

Цель спринта: полностью перевести новый прототип `cloud-ui-g` на React в рабочий менеджерский Cloud UI для RMS/POS. Нельзя переносить runtime на старый Vue `cloud-ui`; его можно использовать только как источник уже подтвержденных API schemas, error handling и UX-инвариантов.

Обязательные правила репозитория:
- Ответы, документация и комментарии по умолчанию на русском.
- Идентификаторы, API fields, route names, env vars и test names на английском.
- Пользовательские UI-строки не хардкодить в TSX. Все labels, empty states, validation, errors, dialogs и notifications должны идти через i18n locale files.
- Не показывать raw Go errors, SQL errors, stack traces, request dumps, PIN, PIN hash, tokens, secrets, raw sync payloads или payment-sensitive payloads.
- Frontend не принимает авторитетных решений о финансовых state transitions.
- UI не должен имитировать отсутствующие backend routes. Если route не подтвержден, показывай route-backed readiness/blocked state, а не fake CRUD.
- Не добавляй сложные новые backend routes. Если для полного сценария не хватает только маленького расширения существующего route, сначала явно зафиксируй это в документации и тестах.
- Не откатывай чужие изменения в рабочем дереве.

Главные источники истины:
- `docs/ui/CLOUD-UI-SPEC.md`
- `docs/backend/CLOUD-BACKEND-SPEC.md`
- `docs/sync/edge-cloud-contracts-v1.md`
- `cloud-ui/src/shared/api.ts`
- `cloud-ui/src/shared/schemas.ts`
- `cloud-backend/internal/masterdata/api/router.go`
- `cloud-backend/internal/provisioning/api/router.go`
- `cloud-backend/internal/cloudsync/api/router.go`
- `cloud-backend/internal/olap/api/router.go`

Текущие факты:
- `cloud-ui-g` сейчас React/Tailwind прототип с моками и Google AI Studio следами.
- `cloud-ui` сейчас Vue/Quasar и уже подключен к большей части Cloud API.
- Целевой dev URL для Cloud UI должен остаться `http://localhost:5174`, потому что `cloud-backend` CORS уже разрешает 5174.
- API base по умолчанию: `http://localhost:8090/api/v1`, override через `VITE_CLOUD_API_BASE`.

Финальная проверка каждой итерации:
- `cd cloud-ui-g && npm install`
- `cd cloud-ui-g && npm run lint`
- `cd cloud-ui-g && npm run build`
- Если менялся Go backend: `cd cloud-backend && go mod tidy && go test ./...`
- Если менялась профильная документация: проверить, что статусы написаны по-русски (`реализовано сейчас`, `запланировано далее`, `вне текущего объема`).
```

## Спринтовая Граница

реализовано сейчас в рамках спринта:

- React app `cloud-ui-g` очищен от Google AI Studio, Gemini и мокового runtime.
- Все основные менеджерские разделы используют реальные Cloud API.
- UI покрывает создание ресторана, настройки ресторана, Edge pairing/assignment, sync events, каталог, меню, модификаторы, pricing policies, tax/service-charge package authoring, роли, сотрудники, матрицу прав ролей, lifecycle/PIN, залы/столы, публикацию, recipes, stop-list, proposal review, inventory/OLAP read-only reports.
- Dashboard и Reports не показывают выдуманные продажи. Revenue KPI показываются только после подтвержденного агрегированного backend API; до этого используются безопасные metadata reports.

запланировано далее:

- Rich BI dashboards с выручкой, средним чеком и топами продаж после появления агрегированных OLAP endpoints.
- Полный employee-specific permission override, если будет принято решение не расширять существующий employee update route в этом спринте.
- Drag-and-drop план зала после базового route-backed hall/table editor.

вне текущего объема:

- POS cashier runtime в Cloud UI.
- KDS runtime screens в Cloud UI.
- PSP/fiscalization/delivery flows.
- Новые сложные backend подсистемы.

## Подтвержденные API Для Подключения

### Restaurants

- `POST /api/v1/restaurants`
- `GET /api/v1/restaurants`
- `GET /api/v1/restaurants/{id}`
- `PATCH /api/v1/restaurants/{id}`
- `POST /api/v1/restaurants/{id}/archive`

### Staff And Permissions

- `POST /api/v1/master-data/roles`
- `GET /api/v1/master-data/roles?restaurant_id=...`
- `PATCH /api/v1/master-data/roles/{id}`
- `POST /api/v1/master-data/roles/{id}/archive`
- `POST /api/v1/master-data/employees`
- `GET /api/v1/master-data/employees?restaurant_id=...`
- `PATCH /api/v1/master-data/employees/{id}`
- `POST /api/v1/master-data/employees/{id}/suspend`
- `POST /api/v1/master-data/employees/{id}/activate`
- `POST /api/v1/master-data/employees/{id}/archive`
- `POST /api/v1/master-data/employees/{id}/role`
- `POST /api/v1/master-data/employees/{id}/pin`
- `POST /api/v1/master-data/employees/{id}/pin/rotate`

Важный риск: отдельный employee-level permission override endpoint не подтвержден. Текущий backend пересчитывает `permission_snapshot_json` из роли. Для true per-employee matrix нужен минимальный backend change в существующий `PATCH /api/v1/master-data/employees/{id}` или отдельное решение владельца.

### Catalog, Menu, Modifiers, Pricing

- `POST/GET/PATCH /api/v1/master-data/catalog/items`
- `POST/GET/PATCH /api/v1/master-data/catalog/folders`
- `POST/GET/PATCH /api/v1/master-data/catalog/folder-parameters`
- `POST/GET/PATCH /api/v1/master-data/catalog/tags`
- `POST /api/v1/master-data/catalog/item-tags`
- `POST/GET/PATCH /api/v1/master-data/modifiers/groups`
- `POST/GET/PATCH /api/v1/master-data/modifiers/options`
- `POST/GET/PATCH /api/v1/master-data/modifiers/bindings`
- `POST/GET/PATCH /api/v1/master-data/pricing/policies`
- `POST /api/v1/master-data/menu/categories`
- `POST/GET/PATCH /api/v1/master-data/menu/items`

### Tax And Service-Charge Package Storage

- `PUT /api/v1/provisioning/master-data/pricing_policy`
- `GET /api/v1/provisioning/master-data/pricing_policy?node_device_id=...`

Payload shape:

```json
{
  "node_device_id": "node-1",
  "restaurant_id": "restaurant-1",
  "sync_mode": "full_snapshot",
  "full_snapshot_reason": "terminal_restaurant_changed",
  "cloud_version": 1,
  "payload_json": {
    "tax_profiles": [],
    "tax_rules": [],
    "service_charge_rules": [],
    "pricing_policies": []
  }
}
```

### Floor

- `POST/GET/PATCH /api/v1/master-data/floor/halls`
- `POST /api/v1/master-data/floor/halls/{id}/archive`
- `POST/GET/PATCH /api/v1/master-data/floor/tables`
- `POST /api/v1/master-data/floor/tables/{id}/archive`

### Edge Provisioning And Sync

- `GET /api/v1/devices/unassigned`
- `POST /api/v1/restaurants/{restaurant_id}/devices/{node_device_id}/assign`
- `GET /api/v1/devices/{node_device_id}/assignment-status`
- `POST /api/v1/restaurants/{restaurant_id}/devices/generate-pairing-code`
- `GET /api/v1/sync/edge-events?restaurant_id=...&limit=...`

### Publication, Recipes, Stop-List, Proposals, Reports

- `POST /api/v1/restaurants/{id}/master-data/publish`
- `GET /api/v1/restaurants/{id}/master-data/publication-state`
- `POST/GET/PATCH /api/v1/master-data/recipes/items`
- `POST/GET/PATCH /api/v1/master-data/inventory/stop-list`
- `POST /api/v1/master-data/inventory/stop-list/{id}/deactivate`
- `GET /api/v1/master-data/catalog-suggestions`
- `POST /api/v1/master-data/catalog-suggestions/{id}/approve`
- `POST /api/v1/master-data/catalog-suggestions/{id}/reject`
- `POST /api/v1/master-data/catalog-suggestions/{id}/request-changes`
- `GET /api/v1/master-data/recipe-suggestions`
- `POST /api/v1/master-data/recipe-suggestions/{id}/approve`
- `POST /api/v1/master-data/recipe-suggestions/{id}/reject`
- `POST /api/v1/master-data/recipe-suggestions/{id}/request-changes`
- `GET /api/v1/inventory/stock-ledger`
- `GET /api/v1/olap/raw-business-events`

## Целевая Структура `cloud-ui-g`

```text
cloud-ui-g/
  README.md
  package.json
  vite.config.ts
  index.html
  src/
    App.tsx
    main.tsx
    index.css
    app/
      CloudManagerApp.tsx
      routes.ts
      navigation.ts
    shared/
      api/
        client.ts
        endpoints.ts
        errors.ts
        schemas.ts
        types.ts
      i18n/
        I18nProvider.tsx
        ru.ts
        keys.ts
      ui/
        SafeErrorBanner.tsx
        EmptyState.tsx
        LoadingSkeleton.tsx
        ConfirmDialog.tsx
        FormField.tsx
        DataTable.tsx
        StatusBadge.tsx
      utils/
        format.ts
        ids.ts
        json.ts
    features/
      dashboard/
      restaurants/
      edge/
      catalog/
      menu/
      modifiers/
      pricing/
      staff/
      floor/
      publications/
      inventory/
      reports/
```

## Итерации Промптов Для Кодовых Агентов

### Итерация 1: Очистить Google AI Studio И Зафиксировать React Baseline

Выполнена

### Итерация 2: API Client, Schemas, Safe Errors, I18n

Выполнена

### Итерация 3: App Shell, Навигация, Restaurant Scope

Выполнена.

### Итерация 4: Restaurants, Settings, Launch Readiness, Publication

Выполнена

### Итерация 5: Edge Provisioning И Поток Синхронизации

Выполнена

### Итерация 6: Catalog Workspace Полного Объема

```markdown
## Задача
Реализуй полноценное управление каталогом: items, folders, folder parameters, tags, item-tags command.

## Файлы
- Create: `cloud-ui-g/src/features/catalog/CatalogPage.tsx`
- Create: `cloud-ui-g/src/features/catalog/CatalogItemsPanel.tsx`
- Create: `cloud-ui-g/src/features/catalog/CatalogFoldersPanel.tsx`
- Create: `cloud-ui-g/src/features/catalog/CatalogTagsPanel.tsx`
- Create: `cloud-ui-g/src/features/catalog/FolderParametersPanel.tsx`
- Create: `cloud-ui-g/src/features/catalog/ItemTagCommandPanel.tsx`
- Create: `cloud-ui-g/src/features/catalog/catalogForms.ts`
- Modify: `cloud-ui-g/src/shared/api/endpoints.ts`
- Modify: `cloud-ui-g/src/shared/i18n/ru.ts`

## API
- `catalog/items`, `catalog/folders`, `catalog/folder-parameters`, `catalog/tags`, `catalog/item-tags`.

## Требования
- `kind`: `dish`, `good`, `semi_finished`, `service`.
- `status`: `draft`, `published`, `archived`.
- Select inputs используют загруженные справочники, не ручной ввод UUID для обычного manager flow.
- Item-tags не имеет list route; UI должен быть command-only form с понятным success state.
- Archive actions требуют confirm dialog.
- После mutation обновлять соответствующий список и readiness.

## Проверки
- Создать/изменить/архивировать catalog item.
- Создать folder и parameter.
- Создать tag и выполнить item-tag command.
```

### Итерация 7: Menu, Modifiers, Pricing Policies, Taxes

Выполнена

```markdown
## Задача
Реализуй рабочие менеджерские UI для продаваемого меню, модификаторов, скидок/надбавок и налоговых/service-charge reference packages.

## Файлы
- Create: `cloud-ui-g/src/features/menu/MenuPage.tsx`
- Create: `cloud-ui-g/src/features/menu/MenuItemsPanel.tsx`
- Create: `cloud-ui-g/src/features/menu/MenuCategoryCommandPanel.tsx`
- Create: `cloud-ui-g/src/features/modifiers/ModifiersPage.tsx`
- Create: `cloud-ui-g/src/features/modifiers/ModifierGroupsPanel.tsx`
- Create: `cloud-ui-g/src/features/modifiers/ModifierOptionsPanel.tsx`
- Create: `cloud-ui-g/src/features/modifiers/ModifierBindingsPanel.tsx`
- Create: `cloud-ui-g/src/features/pricing/PricingPage.tsx`
- Create: `cloud-ui-g/src/features/pricing/PricingPoliciesPanel.tsx`
- Create: `cloud-ui-g/src/features/pricing/TaxPackagePanel.tsx`
- Create: `cloud-ui-g/src/features/pricing/taxPackageTypes.ts`
- Modify: `cloud-ui-g/src/shared/api/endpoints.ts`
- Modify: `cloud-ui-g/src/shared/api/schemas.ts`
- Modify: `cloud-ui-g/src/shared/i18n/ru.ts`

## API
- `menu/items`, `menu/categories`
- `modifiers/groups`, `modifiers/options`, `modifiers/bindings`
- `pricing/policies`
- `PUT/GET /provisioning/master-data/pricing_policy`

## Требования
- Menu item form: catalog item select, name, price minor, currency, availability JSON editor, station routing key, status.
- Menu category remains command-only because list/update route is not confirmed.
- Modifier binding target options depend on `target_type`: menu item, catalog item, folder, tag.
- Pricing policies cover discount/surcharge, fixed/percentage, `application_index`, manual flag, required permission.
- Detect duplicate `application_index` client-side before submit and show validation error.
- Tax package UI must be structured forms for `tax_profiles`, `tax_rules`, `service_charge_rules`; no raw JSON-only editor as primary workflow.
- Tax package must serialize into `payload_json` for `pricing_policy` stream and preserve existing package rows when editing.
- If `GET pricing_policy` returns 404, show empty state and create first package.

## Проверки
- Создать menu item from catalog item.
- Создать modifier group/option/binding.
- Создать discount и surcharge с разными application indexes.
- Создать tax profile, tax rule и service charge package через generic package API.
```

### Итерация 8: Staff, Roles, Permission Matrix, PIN Lifecycle

Выполнена

```markdown
## Задача
Реализуй управление ролями, сотрудниками, матрицей прав, lifecycle и PIN без раскрытия sensitive material.

## Файлы
- Create: `cloud-ui-g/src/features/staff/StaffPage.tsx`
- Create: `cloud-ui-g/src/features/staff/RolesPanel.tsx`
- Create: `cloud-ui-g/src/features/staff/EmployeesPanel.tsx`
- Create: `cloud-ui-g/src/features/staff/PermissionMatrix.tsx`
- Create: `cloud-ui-g/src/features/staff/roleProfiles.ts`
- Create: `cloud-ui-g/src/features/staff/permissionCatalog.ts`
- Modify: `cloud-ui-g/src/shared/api/endpoints.ts`
- Modify: `cloud-ui-g/src/shared/i18n/ru.ts`

## API
- Roles and employees endpoints under `/master-data`.

## Требования
- Role profiles: `cashier`, `senior_cashier`, `waiter`, `manager`, `kitchen`, `support_admin`.
- Role form stores permissions as stable `permissions_json` generated from selected permission ids.
- Employee create requires role, name, PIN. PIN input `type=password`, autocomplete `new-password`.
- Employee edit allows name, role, status, suspend/activate/archive, role assignment, PIN rotate.
- UI response displays `pin_configured` and `pin_credential_version`, never PIN/hash.
- Employee permission snapshot currently comes from role. Show employee `permission_snapshot_json` as read-only matrix unless backend extension is implemented.

## Decision Point: Employee-Specific Matrix
Для полного требования "матрица прав для отдельных сотрудников" есть два пути:
1. Без backend changes: read-only employee snapshot + role assignment + role clone workflow. Это честно работает на текущем API.
2. Минимальное backend extension: расширить существующий `PATCH /api/v1/master-data/employees/{id}` полем `permission_snapshot_json` или `permission_ids`, добавить service/repository tests и docs. Не добавлять новый route.

Если владелец спринта требует true employee-specific overrides, реализуй путь 2 отдельным коммитом и обнови `docs/backend/CLOUD-BACKEND-SPEC.md`, `docs/ui/CLOUD-UI-SPEC.md`.

## Проверки
- Создать роль из профиля и изменить матрицу.
- Создать сотрудника, назначить роль, suspend/activate/archive.
- Rotate PIN не выводит PIN material.
- Employee snapshot view не дает ложного впечатления, что override сохранен, если backend extension не сделан.
```

### Итерация 9: Floor, Halls, Tables

```markdown
## Задача
Реализуй управление залами и столами для менеджера.

## Файлы
- Create: `cloud-ui-g/src/features/floor/FloorPage.tsx`
- Create: `cloud-ui-g/src/features/floor/HallsPanel.tsx`
- Create: `cloud-ui-g/src/features/floor/TablesPanel.tsx`
- Create: `cloud-ui-g/src/features/floor/FloorPreview.tsx`
- Modify: `cloud-ui-g/src/shared/api/endpoints.ts`
- Modify: `cloud-ui-g/src/shared/i18n/ru.ts`

## API
- `master-data/floor/halls`
- `master-data/floor/tables`

## Требования
- Hall CRUD with archive.
- Table CRUD with hall select, seats, status, archive.
- Floor preview groups tables by hall and shows status/readiness.
- Не реализовывать fake drag-and-drop persistence, если backend координаты столов не подтверждены.

## Проверки
- Создать зал.
- Создать стол в зале.
- Изменить seats/status.
- Archive hall/table.
```

### Итерация 10: Recipes, Stop-List, Proposal Review, Inventory

```markdown
## Задача
Добавь manager-facing inventory surfaces, которые уже имеют Cloud routes: recipes, stop-list, proposal review, stock ledger read.

## Файлы
- Create: `cloud-ui-g/src/features/inventory/InventoryPage.tsx`
- Create: `cloud-ui-g/src/features/inventory/RecipesPanel.tsx`
- Create: `cloud-ui-g/src/features/inventory/StopListPanel.tsx`
- Create: `cloud-ui-g/src/features/inventory/ProposalReviewPanel.tsx`
- Create: `cloud-ui-g/src/features/inventory/StockLedgerPanel.tsx`
- Modify: `cloud-ui-g/src/shared/api/endpoints.ts`
- Modify: `cloud-ui-g/src/shared/api/schemas.ts`
- Modify: `cloud-ui-g/src/shared/i18n/ru.ts`

## API
- `recipes/items`
- `inventory/stop-list`
- `catalog-suggestions`
- `recipe-suggestions`
- `inventory/stock-ledger`

## Требования
- Recipes: owner catalog item, component catalog item, quantity, unit, loss_percent.
- Stop-list: catalog item, available_quantity nullable, source, reason, active, deactivate action.
- Proposal review: list catalog and recipe suggestions; actions approve/reject/request changes.
- Stock ledger is read-only report; no manual stock mutation UI unless route is confirmed.
- All review actions must refresh related list and show safe success/error state.

## Проверки
- Создать/изменить recipe item.
- Upsert/deactivate stop-list row.
- Review action sends correct endpoint.
- Stock ledger empty/error states work.
```

### Итерация 11: Analytics Dashboard And Reports Без Моков

```markdown
## Задача
Реализуй dashboard и отдельный reports раздел на имеющихся read-only API без выдуманных revenue metrics.

## Файлы
- Create: `cloud-ui-g/src/features/reports/ReportsPage.tsx`
- Create: `cloud-ui-g/src/features/reports/OlapRawEventsPanel.tsx`
- Create: `cloud-ui-g/src/features/reports/SyncHealthPanel.tsx`
- Create: `cloud-ui-g/src/features/reports/ReportFilters.tsx`
- Modify: `cloud-ui-g/src/features/dashboard/DashboardPage.tsx`
- Modify: `cloud-ui-g/src/shared/api/endpoints.ts`
- Modify: `cloud-ui-g/src/shared/api/schemas.ts`
- Modify: `cloud-ui-g/src/shared/i18n/ru.ts`

## API
- `GET /olap/raw-business-events`
- `GET /sync/edge-events`
- `GET /inventory/stock-ledger`

## Требования
- Dashboard shows route-backed operational metrics: restaurant readiness, publication status, edge device counts, event counts by type, latest sync events, stock ledger row count.
- Do not show fake revenue, average check, payment method split, popular items or hourly sales unless backend returns aggregate data.
- Reports page supports filters: restaurant, event_type, occurred_from, occurred_to, limit, offset.
- OLAP raw event rows show safe metadata only: event_id, tenant_id, restaurant_id, device_id, employee_id, event_type, occurred_at, cloud_received_at, raw_payload_sha256_hex.
- Add clear readiness panel for aggregate BI: status `запланировано далее`, not implemented as mock.

## Проверки
- Empty OLAP response renders cleanly.
- Invalid date filter shows validation before request.
- No raw payload appears in DOM.
```

### Итерация 12: Replace Current Cloud UI Entry In Docs And Local Workflow

```markdown
## Задача
Сделай `cloud-ui-g` официальным React Cloud UI target для разработки и документации, не удаляя старый `cloud-ui` без отдельного решения владельца.

## Файлы
- Modify: `cloud-ui-g/README.md`
- Modify: `docs/ui/CLOUD-UI-SPEC.md`
- Modify: `docs/backend/LOCAL-DOCKER-STACK.md`
- Modify: `docker-compose.local.yml` if devbox/cloud-ui command references old module
- Modify: `docker-compose.igor.yml` if devbox/cloud-ui command references old module

## Требования
- Документация должна сказать: `cloud-ui-g` реализовано сейчас как React Cloud manager UI.
- Старый Vue `cloud-ui` пометить как legacy/pilot reference, если он остается в repo.
- Не документировать неподдержанные aggregate BI, employee override или drag/drop floor как реализованные.
- Все статусы в документации по-русски: `реализовано сейчас`, `запланировано далее`, `вне текущего объема`.
- Docker/devbox должен запускать UI на 5174.

## Проверки
- `rg -n "cloud-ui" docs docker-compose*.yml` и вручную проверить, где нужен old/new wording.
- `rg -n "implemented now|planned next|out of scope|future|later|temporary for now|for now" docs/ui/CLOUD-UI-SPEC.md docs/backend/LOCAL-DOCKER-STACK.md`
```

### Итерация 13: End-To-End QA, Browser Smoke, Sprint Acceptance

```markdown
## Задача
Проведи финальную проверку React Cloud UI против Cloud backend и зафиксируй остаточные риски.

## Файлы
- Create: `cloud-ui-g/e2e/cloud-manager-smoke.spec.ts` if Playwright infra is added.
- Modify: `cloud-ui-g/package.json` scripts if adding e2e.
- Modify: `cloud-ui-g/README.md` with verification commands.

## Smoke Flow
1. App loads at `http://localhost:5174`.
2. Create/select restaurant.
3. Create role and employee.
4. Create hall/table.
5. Create catalog item.
6. Create menu item from catalog item.
7. Create modifier group/option/binding.
8. Create pricing policy.
9. Generate pairing code or assign unassigned device if present.
10. Open sync events.
11. Publish master data.
12. Open reports.

## Checks
- Page identity: correct title and URL.
- Not blank: meaningful dashboard content.
- No framework overlay.
- Console: no relevant app errors/warnings.
- Desktop viewport and one mobile viewport.
- Text does not overlap controls.
- Loading, empty and error states verified for at least one list and one form.

## Commands
- `cd cloud-ui-g && npm install`
- `cd cloud-ui-g && npm run lint`
- `cd cloud-ui-g && npm run build`
- `cd cloud-backend && go mod tidy && go test ./...` if backend changed.
- Optional local stack: `docker compose -f docker-compose.local.yml up --build -d cloud-postgres license-api cloud-api pos-edge`

## Acceptance
- No Google/Gemini artifacts remain.
- No mock data drives required manager workflows.
- All implemented sections call Cloud backend endpoints.
- UI does not expose sensitive payloads.
- Missing backend contracts are documented as `запланировано далее`, not silently faked.
```

## Риски И Решения Владельца До Старта Реализации

1. Employee-level permissions:
   - Текущий backend поддерживает role permissions и employee permission snapshot from role.
   - Для true per-employee override нужен минимальный backend change или явное решение оставить read-only snapshot в этом спринте.

2. Revenue analytics:
   - `GET /api/v1/olap/raw-business-events` не возвращает суммы, топы продаж или средний чек.
   - Dashboard может показывать operational analytics, но не финансовые KPI без нового aggregate endpoint.

3. Tax/service-charge authoring:
   - CRUD endpoints для tax profiles/rules не выделены.
   - Реализуемый путь на текущем backend: structured UI поверх generic `PUT /provisioning/master-data/pricing_policy`.

4. Floor drag-and-drop:
   - Backend table coordinates не подтверждены.
   - В спринте делать grouped floor preview и table CRUD; drag/drop persistence оставить `запланировано далее`.

5. Old `cloud-ui` lifecycle:
   - Этот план не удаляет Vue module автоматически.
   - После успешной React миграции владелец должен решить: оставить как reference, удалить или архивировать отдельным PR.

## Финальный Definition Of Done

- `cloud-ui-g` является основным React Cloud UI для менеджеров.
- Все обязательные сценарии из запроса доступны как route-backed UI или честно помечены как blocked by backend contract.
- `npm run build` и `npm run lint` проходят в `cloud-ui-g`.
- Если backend минимально расширялся, `go test ./...` проходит в `cloud-backend`, документация обновлена.
- В UI нет Google, Gemini, AI Studio, Google Fonts и мокового runtime.
- В TSX нет хардкодных русских UI-строк вне i18n.
- Sensitive data redaction работает для ошибок и sync/report surfaces.
- План QA содержит результаты desktop/mobile smoke.
