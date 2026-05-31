# Pilot UX Market Review

Статус: запланировано далее для полного пилота.

Документ фиксирует UX-ориентиры для advanced KDS lifecycle, waiter mobile и Cloud manager web app. Это не источник runtime-истины; backend/API контракты остаются в `docs/backend/*`, UI-контракты - в `docs/ui/*`, sync events - в `docs/sync/*`.

## Источники

- Toast KDS: official support описывает all-day view, production item counts, advanced prep station routing, kitchen productivity reporting, prep-time firing, color-coded modifiers, prep station / expediter modes, recently fulfilled tickets и recall. Источник: https://support.toasttab.com/en/article/Get-Started-With-the-Kitchen-Display-System
- Square KDS setup/routing/completion: official support описывает Prep и Expo device types, completion/recall behavior per station/all devices, configurable layout, timers/alerts, routing by POS/online sources, prioritization and grouping identical items. Источники: https://squareup.com/help/us/en/article/7944-get-started-with-square-kds-android, https://squareup.com/help/us/en/article/7959-route-orders-with-your-kds, https://squareup.com/help/us/en/article/8171-complete-orders-with-square-kds, https://squareup.com/help/us/en/article/8168-prioritize-orders-with-square-kds
- Lightspeed Restaurant recipes: official support описывает recipe list/search, ingredient lookup, made-to-order/made-in-batches recipe types, ingredient picker, unit/amount editing, instructions, gross profit / suggested price, local/shared recipe publishing. Источник: https://k-series-support.lightspeedhq.com/hc/en-us/articles/4407511552155-Creating-and-managing-recipes
- Oracle Simphony: official product page описывает all-in-one restaurant management, real-time table management, reporting/analytics, inventory/menu management and multi-channel KDS across stations and order channels. Источник: https://www.oracle.com/food-beverage/micros/

## KDS Выводы Для Пилота

Рыночный минимум для KDS не ограничивается кнопкой `served`. Для полного пилота требуется рабочий lifecycle:

- station/expo split: экран станции видит только свои позиции, expo/manager view видит весь заказ;
- item-level actions: `accept`, `start`, `hold`, `ready`, `served`, `recall`, `cancel`;
- status timeline and audit: каждое действие создает `KitchenTicketStatusChanged`, а `served` дополнительно создает `ItemServed`;
- priority and timing: oldest-first default, ручной priority, SLA/timer color state, warning/late thresholds;
- all-day/production counts: агрегированные counts по блюдам/ингредиентам для активных tickets;
- safe undo/recall: короткое undo window для ошибочного status action и явный recall после закрытия;
- offline continuity: Edge KDS продолжает принимать POS-created orders и писать outbox events без Cloud.

В полном пилоте не обязательны аппаратные bump bars, kitchen printer orchestration и rich BI dashboards. Они остаются вне текущего объема полного пилота.

## Chef Inventory And Recipe UX

Поварский поток должен быть операционным, а не админским:

- receipt capture: поставка вводится через supplier/date/items, товар выбирается из catalog picker; если товара нет, создается `CatalogItemChangeSuggested`;
- catalog proposal: форма показывает name, kind, unit, SKU/optional fields, duplicate hints и linked receipt line;
- recipe viewer: техкарта открывается из KDS item с ingredients, quantities, units, losses, prep time and instructions;
- recipe proposal: замена ингредиента или правка количества/единицы/потерь/времени создает `RecipeChangeSuggested`; Edge не применяет изменение локально;
- prep time delta: UI валидирует предел `recipe_suggestion_max_time_delta_minutes`, backend повторно валидирует;
- stop-list edit: повар может поставить item/component в stop-list, указать `available_quantity` и reason; конфликт применения задается `stop_list_conflict_policy`.

## Waiter Mobile UX

Mobile layout в полном пилоте принадлежит только официанту:

- `/pos/waiter` оптимизируется под viewport `390x844`;
- primary navigation: halls/tables, active orders, compact analytics;
- order flow: table -> menu/category -> modifiers -> quantity -> void line -> precheck issue/reprint;
- payment/refund/cash drawer controls скрыты по умолчанию и появляются только при явных backend permissions;
- cashier, KDS и manager modes не получают отдельные mobile variants в рамках полного пилота.

## Cloud Manager UX

Cloud UI должен стать менеджерским приложением, а не raw-admin:

- guided setup: restaurant, staff, floor, catalog/menu/modifiers/pricing, recipes, stop-list, publication, Edge delivery readiness;
- interactive selectors вместо UUID/raw JSON для связей catalog/menu/modifier/recipe/stop-list;
- proposal queues: `CatalogItemChangeSuggested` and `RecipeChangeSuggested` показываются как review workflow с approve/reject, diff and source event metadata;
- inventory workspace: stock receipts, counts, production, ledger/balances/costing status, recalculation status;
- OLAP workspace: ClickHouse export health, retry/backfill state and read-only previews for sales/stock moves; COGS/margin and kitchen timing remain costing-dependent future analytics after reliable cost basis and backend/UI contracts;
- sync observability: accepted/rejected/retryable event metadata, checksums and support codes без raw payload, PIN/token/secret values or sensitive request dumps.

## Design Code Requirements

- UI labels, empty states, validation, toasts and dialogs must use `vue-i18n`.
- Business flows must not require manual UUID entry when a referenced entity can be selected from loaded data.
- KDS actions must use icons/buttons with stable dimensions and visible status/timer signals.
- Waiter mobile must be tested only on waiter route; no responsive mobile promise is made for cashier/KDS/manager routes.
- Playwright coverage must include waiter mobile, KDS tablet/desktop, Cloud manager proposal review and Cloud OLAP/export readiness states.
