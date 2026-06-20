# Промпт для продолжения адаптации `cloud-ui-g` под RMS Manager reference

Мы продолжаем разработку RMS-POS системы для ресторанов.

Нужно продолжить адаптацию активного Cloud back-office UI в `cloud-ui-g/` под согласованный заказчиком референс менеджерского RMS-интерфейса из `docs/ui/myhoreca-rms-manager/`.

Референс является визуальным и UX-ориентиром. Его нельзя подключать напрямую как runtime UI без переработки data layer: внутри референса есть mock state, локальные симуляторы продаж, терминалов, склада, sync и stop-list. В `cloud-ui-g` нужно переносить внешний вид, композицию, плотность, состояния и UX-паттерны, но использовать только реальные Cloud routes или явно документировать недостающие backend contracts.

## Обязательный старт каждой новой сессии

Перед изменениями прочитать:

- `AGENTS.md`;
- `docs/ui/myhoreca-rms-manager/prompt-to-continue.md`;
- `docs/ui/myhoreca-rms-manager/src/App.tsx`;
- `docs/ui/myhoreca-rms-manager/src/components/Sidebar.tsx`;
- профильный reference component из `docs/ui/myhoreca-rms-manager/src/components/`;
- текущий target screen/panel в `cloud-ui-g/src/features/**`;
- `cloud-ui-g/src/app/CloudManagerApp.tsx`;
- `cloud-ui-g/src/app/Sidebar.tsx`;
- `cloud-ui-g/src/shared/i18n/ru.ts`;
- `docs/ui/CLOUD-UI-SPEC.md`;
- `docs/ui/CLOUD-UI-RMS-MANAGER-BACKEND-GAPS.md`;
- `docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.

Перед правками выполнить:

```bash
git status --short
```

Если в рабочем дереве есть изменения, не откатывать их. Понять, относятся ли они к задаче, и работать поверх них аккуратно.

## Текущий статус

Реализовано сейчас:

- `cloud-ui-g` является активным React/Vite/Tailwind Cloud UI runtime.
- Базовый shell уже приближен к референсу:
  - темный slate sidebar;
  - бренд `MyHoreca RMS`;
  - restaurant selector перенесен в sidebar;
  - светлая рабочая область;
  - dashboard page header;
  - responsive sidebar desktop/mobile;
  - Inter/JetBrains Mono;
  - blue accent для active/primary states.
- Route-backed разделы уже существуют:
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
- Placeholders `inventory` и `reports` есть в navigation, но полноценными runtime screens не считаются.
- Backend gaps по референсу зафиксированы в `docs/ui/CLOUD-UI-RMS-MANAGER-BACKEND-GAPS.md`.
- Подробная спецификация требуемых Cloud routes зафиксирована в `docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.
- Выполнен первый визуальный срез:
  - shared `PanelHeader`;
  - улучшенный `LoadingSkeleton`;
  - dashboard route-backed KPI/status blocks без финансовых моков;
  - readiness progress summary;
  - publications operator checkpoint;
  - polished restaurants form/list layout.
- Выполнен staff/permissions срез:
  - compact table списка сотрудников;
  - create/edit сотрудника через модалку;
  - role-backed матрица прав с короткими кодами permission;
  - строки сотрудников в матрице пересчитываются от текущих прав должности;
  - поиск и связанное выделение права в справочнике и столбце матрицы.

Запланировано далее:

- Следующий этап: приблизить `edge-sync` к reference `EdgeDevicesPanel` и `SyncPanel`, используя только реальные Cloud routes.
- Подключать уже существующие read-only Cloud routes там, где они есть.
- Не имитировать отсутствующие backend capabilities моками.
- Любые новые необходимые ручки фиксировать в `docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.

Вне текущего объема:

- Переписывать backend/runtime code без отдельного требования.
- Подключать reference app целиком.
- Переносить mock simulations из reference UI как production behavior.
- Делать frontend авторитетным источником financial/order/stock state.

## Главные правила реализации

- Ответы и финальный отчет писать на русском.
- Runtime/backend code не трогать, если пользователь явно не попросил backend implementation.
- Не добавлять русские hardcoded UI strings в компоненты. Все пользовательские строки добавлять в `cloud-ui-g/src/shared/i18n/ru.ts`.
- Не добавлять новые зависимости без острой необходимости. Использовать существующий React/Tailwind/lucide stack.
- Не переписывать бизнес-логику и API client без причины.
- Не копировать `mockData.ts`, `simulateSale`, `triggerForceSync`, `handleSimulateOutage`, `handlePrintTest` и другие reference-only симуляторы.
- Использовать реальные DTO и Zod schemas из `cloud-ui-g/src/shared/api/schemas.ts`.
- Для новых UI data needs сначала искать существующие endpoints в `cloud-ui-g/src/shared/api/endpoints.ts`, `docs/ui/CLOUD-UI-SPEC.md`, `docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.
- Если backend route отсутствует, сделать UI empty/blocked state и обновить backend API spec, а не писать моковую фичу.
- Для financial, stock, sync и device commands не делать auto-retry или destructive controls без idempotency key, RBAC/support permission и audit contract.
- Для полной и достоверной проверки рабочего Cloud UI можно и нужно запускать или перезапускать реальный `cloud-backend`, если экран зависит от route-backed данных. Проверка только на blocked/error states недостаточна для финального UX-вывода по реализованному экрану.

## Визуальное направление

Ориентироваться на `docs/ui/myhoreca-rms-manager`:

- плотный management dashboard, не landing page;
- белые панели на светло-сером фоне;
- темный `#0f172a` sidebar;
- blue accent для active states, primary actions и service indicators;
- карточки `rounded-2xl`, тонкие borders, мягкие shadows;
- dashboard-like KPI cards и compact status blocks;
- tables с аккуратными headers, hover rows, bounded overflow;
- forms в светлых command panels;
- empty/loading/error states в едином стиле;
- lucide icons в заголовках, action buttons и status cards;
- mobile верстка без overlap/overflow.

Избегать:

- маркетинговых hero sections;
- декоративных моковых графиков без данных;
- огромных landing-like headings внутри panels;
- UI cards внутри cards;
- русских строк прямо в TSX;
- dark/blue монотонности во всей рабочей области.

## План сессий

Каждая сессия должна быть небольшой и проверяемой. Лучше закончить один раздел качественно, чем тронуть все.

### Сессия 1: shared visual foundation

Статус: выполнено.

Цель: улучшить общие UI-паттерны, чтобы все разделы стали ближе к референсу минимальным diff.

Прочитать:

- `cloud-ui-g/src/index.css`;
- `cloud-ui-g/src/shared/ui/EmptyState.tsx`;
- `cloud-ui-g/src/shared/ui/LoadingSkeleton.tsx`;
- `cloud-ui-g/src/shared/ui/SafeErrorBanner.tsx`;
- 2-3 текущих panels из `cloud-ui-g/src/features/**`.

Сделать:

- унифицировать panel header pattern;
- улучшить shared empty/loading/error states;
- проверить buttons, inputs, tables и article cards;
- не ломать локальные классы feature forms.

Проверить:

```bash
cd cloud-ui-g
npm run build
```

### Сессия 2: restaurants + dashboard + publications

Статус: выполнено.

Цель: довести стартовую зону Cloud manager до визуального уровня reference analytics/dashboard.

Прочитать:

- `docs/ui/myhoreca-rms-manager/src/components/AnalyticsPanel.tsx`;
- `cloud-ui-g/src/features/dashboard/DashboardPage.tsx`;
- `cloud-ui-g/src/features/dashboard/LaunchReadinessPanel.tsx`;
- `cloud-ui-g/src/features/restaurants/RestaurantsPage.tsx`;
- `cloud-ui-g/src/features/publications/PublicationPanel.tsx`.

Сделать:

- dashboard header/KPI/readiness cards приблизить к reference density;
- manual publication panel не развивать; целевой экран показывает только automatic delivery status по Edge после появления backend DTO;
- restaurants list/form привести к более polished management style;
- если нужны sales KPI routes, не мокать, а сверить и обновить `CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.

Проверено:

- `npm install`;
- `npm run build`;
- `npm run lint`;
- desktop/mobile render через headless Chromium и CDP interaction для mobile menu.

Оставшийся риск:

- data-loaded состояния не проверялись на реальном `cloud-backend`; следующие route-backed экраны нужно проверять с поднятым backend.

### Сессия 3: staff add/edit employee screen

Статус: выполнено.

Цель: полностью реализовать экран добавления и редактирования сотрудников по примеру reference `StaffPanel`, сохранив текущую Cloud API модель сотрудников, ролей, lifecycle, назначения роли и PIN rotation.

Прочитать:

- `docs/ui/myhoreca-rms-manager/src/components/StaffPanel.tsx`;
- `cloud-ui-g/src/features/staff/**`;
- `cloud-ui-g/src/features/staff/permissionCatalog.ts`;
- `cloud-ui-g/src/shared/api/schemas.ts`;
- `cloud-ui-g/src/shared/api/endpoints.ts`;
- `docs/ui/POS-UI-RBAC.md`;
- `docs/ui/CLOUD-UI-SPEC.md`;
- `docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.

Сделать:

- переработать add/edit employee flow как полноценный management screen, а не минимальную форму;
- приблизить layout, density, staff cards/list/table и command panels к reference `StaffPanel`;
- сделать role selection, employee status, PIN configured/version и permission snapshot понятными оператору;
- сохранить distinction: это POS Edge employee permissions, не Cloud operator RBAC;
- не добавлять `email`, `phone`, `on_shift` и другие поля, которых backend не возвращает;
- не показывать PIN, PIN hash, raw payload или sensitive details;
- все новые пользовательские строки добавить в `cloud-ui-g/src/shared/i18n/ru.ts`;
- если для полного UX нужны shift presence или Cloud operator RBAC, оставить это как `запланировано далее` в API spec, не мокать.

Проверить:

- запустить или перезапустить реальный `cloud-backend`;
- поднять `cloud-ui-g`;
- создать тестовый restaurant, role и employee через UI или существующие seed/dev данные;
- проверить create employee, edit employee, assign role, suspend/activate, rotate PIN на реальных Cloud routes;
- проверить desktop `1440x900`, mobile `390x844` и раскрытое mobile menu;
- выполнить `cd cloud-ui-g && npm install && npm run build`;
- если менялись form helpers/tests, выполнить `npm run test`;
- после UI проверки выполнить профильную документационную сверку и обновить spec, если обнаружены missing contracts.

Выполнено:

- add/edit сотрудника перенесен в модалку;
- список сотрудников приведен к компактной table layout без персональных матриц прав;
- сохранены текущие Cloud API поля сотрудников, lifecycle, назначение роли и PIN rotation;
- права сотрудника не редактируются в карточке/строке сотрудника.

Проверено:

- `cd cloud-ui-g && npm run lint`;
- `cd cloud-ui-g && npm run build`;
- `cd cloud-ui-g && npm run test`.

Оставшийся риск:

- data-loaded flow с реальным `cloud-backend` и Playwright-рендер не проверялись в этой сессии.

### Сессия 4: edge sync + device readiness

Цель: приблизить `edge-sync` к reference `EdgeDevicesPanel` и `SyncPanel`, используя только реальные routes.

Прочитать:

- `docs/ui/myhoreca-rms-manager/src/components/EdgeDevicesPanel.tsx`;
- `docs/ui/myhoreca-rms-manager/src/components/SyncPanel.tsx`;
- `cloud-ui-g/src/features/edge/EdgeSyncPage.tsx`;
- `cloud-ui-g/src/features/edge/UnassignedDevicesPanel.tsx`;
- `cloud-ui-g/src/features/edge/PairingCodePanel.tsx`;
- `cloud-ui-g/src/features/edge/EdgeEventsPanel.tsx`.

Сделать:

- улучшить layout: слева device/pairing cards, справа safe event console или log panel;
- добавить более выразительные status badges;
- не добавлять ping/test-print controls без backend route;
- если нужен hardware inventory screen, использовать spec entries из `CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.

Вне текущего объема:

- mock port scan;
- mock device outage;
- mock test print.

### Сессия 5: catalog + menu

Цель: приблизить catalog/menu screens к reference `MenuPanel`.

Прочитать:

- `docs/ui/myhoreca-rms-manager/src/components/MenuPanel.tsx`;
- `cloud-ui-g/src/features/catalog/**`;
- `cloud-ui-g/src/features/menu/**`;
- `cloud-ui-g/src/shared/api/schemas.ts`;
- `cloud-ui-g/src/shared/api/endpoints.ts`.

Сделать:

- улучшить menu item cards/table;
- добавить compact filters/search только если они работают по локально загруженным real API data;
- сделать forms visually consistent;
- не копировать tech-card modal, если recipes routes не реализованы;
- не отображать category list как будто он есть, пока нет `GET /master-data/menu/categories`.

Документировать:

- если нужен category list/update/archive или recipes/tech cards, обновить API spec.

### Сессия 6: modifiers + pricing/taxes

Цель: сделать operational screens для modifiers/pricing цельными и похожими на reference management panels.

Прочитать:

- `cloud-ui-g/src/features/modifiers/**`;
- `cloud-ui-g/src/features/pricing/**`;
- relevant parts of `docs/ui/myhoreca-rms-manager/src/components/MenuPanel.tsx`.

Сделать:

- улучшить group/option/binding cards;
- сделать visual hierarchy для required/min/max/status;
- pricing policies оформить как читаемый rule stack;
- tax package panel сделать безопасным operator form, не показывать raw unsafe payload из backend events.

Проверить:

- form validation tests, если менялись form helpers;
- `npm run build`.

### Сессия 7: staff + permissions polish / role matrix

Статус: выполнено.

Цель: приблизить staff UI к reference `StaffPanel`, сохранив текущую POS Edge permission model.

Прочитать:

- `docs/ui/myhoreca-rms-manager/src/components/StaffPanel.tsx`;
- `cloud-ui-g/src/features/staff/**`;
- `docs/ui/POS-UI-RBAC.md`;
- `cloud-ui-g/src/features/staff/permissionCatalog.ts`.

Сделать:

- улучшить staff cards/list/table;
- сделать role matrix визуально компактнее;
- явно сохранить distinction: POS Edge employee permissions, не Cloud operator RBAC;
- не добавлять email/phone/on_shift как будто backend их возвращает.

Документировать:

- shift presence и Cloud operator RBAC остаются `запланировано далее` в API spec.

Выполнено:

- staff cards заменены на compact table;
- role matrix преобразована в iiko-like матрицу с правами в столбцах и ролями/сотрудниками в строках;
- каждому permission задан короткий код 3-4 символа;
- строки сотрудников в матрице read-only и динамически отражают текущие права назначенной должности;
- добавлен поиск по праву и связанное выделение строки справочника/столбца матрицы;
- distinction POS Edge employee permissions vs Cloud operator RBAC сохранен.

Проверено:

- `cd cloud-ui-g && npm run lint`;
- `cd cloud-ui-g && npm run build`;
- `cd cloud-ui-g && npm run test`.

### Сессия 8: floor

Цель: привести halls/tables/floor preview к reference `TablesPanel`.

Прочитать:

- `docs/ui/myhoreca-rms-manager/src/components/TablesPanel.tsx`;
- `cloud-ui-g/src/features/floor/**`.

Сделать:

- улучшить hall/table forms и list cards;
- сделать preview визуально похожим на схему зала;
- не делать drag-and-drop сохранение координат без backend route;
- если добавлять локальное drag-only preview, явно не сохранять и не выдавать за runtime behavior.

Документировать:

- `PATCH /master-data/floor/tables/{id}/layout` остается planned API.

### Сессия 9: inventory/reports placeholders

Цель: заменить generic blocked placeholder на честные route-backed или planned states.

Прочитать:

- `docs/ui/myhoreca-rms-manager/src/components/WarehousePanel.tsx`;
- `docs/backend/INVENTORY-COSTING-SPEC.md`;
- `docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`;
- existing Cloud read-only routes:
  - `/inventory/stock-ledger`;
  - `/inventory/stock-balances`;
  - `/inventory/recalculation-jobs`;
  - `/olap/stock-moves`;
  - `/olap/stock-move-summary`;
  - `/olap/sales-kitchen-summary`;
  - `/olap/kitchen-timing-summary`.

Сделать:

- если подключаешь existing read-only routes, добавить Zod schemas и panels в `cloud-ui-g`;
- если полноценного flow нет, сделать premium blocked/planned state с ссылкой на required backend routes;
- не делать TTN/import/posting UI без backend implementation.

## Как отвечать на вопрос про прямое использование референса

Короткий ответ:

Нет, напрямую использовать reference app только заменой mock data и TypeScript types нельзя.

Почему:

- reference `App.tsx` держит всю бизнес-логику в local React state;
- reference components ожидают mock DTO, которых нет в Cloud API;
- часть действий является симуляцией, а не безопасным backend contract;
- нет Zod response validation;
- нет текущего `cloud-ui-g` safe error handling;
- нет текущей route scope/navigation модели;
- есть hardcoded русские строки;
- нет связки с `VITE_CLOUD_API_BASE`, publication workflow и Cloud error envelope.

Что можно:

- использовать reference как визуальный источник;
- переносить JSX/layout/classes по разделам;
- адаптировать под текущие `cloud-ui-g` endpoints, schemas, i18n и safety rules;
- добавлять недостающие backend contracts в `CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.

Практический путь:

- продолжать переносить разделами в `cloud-ui-g`;
- не монтировать reference app целиком;
- сначала shell/shared UI, затем route-backed sections, затем planned API sections.

## Обязательные проверки перед финалом

Для UI изменений:

```bash
cd cloud-ui-g
npm install
npm run build
```

Если менялись form helpers/tests:

```bash
cd cloud-ui-g
npm run test
```

Если Playwright доступен:

- для route-backed экранов сначала запустить или перезапустить реальный `cloud-backend`, чтобы проверять рабочий интерфейс с фактическими DTO, validation и safe error envelope;
- поднять dev server;
- проверить desktop `1440x900`;
- проверить mobile `390x844`;
- проверить раскрытое mobile menu;
- убедиться, что нет явных overlap/overflow;
- console errors от недоступного backend API допустимы только для planned/blocked states; для реализуемого route-backed экрана backend должен быть запущен.

Для документационных изменений без runtime code:

- `git diff --stat`;
- `rg` по статусам `реализовано сейчас|запланировано далее|вне текущего объема`;
- build/tests можно не запускать, но нужно явно сказать об этом в финальном отчете.

## Финальный отчет

Финальный отчет писать на русском языке.

Кратко указать:

- что найдено;
- что изменено;
- измененные файлы;
- какие проверки запущены;
- какие проверки не удалось или не требовалось запускать;
- оставшиеся риски;
- что запланировано далее;
- что вне текущего объема;
- затрагивался ли runtime code;
- дать краткий и полный комментарий о выполненных работах.
