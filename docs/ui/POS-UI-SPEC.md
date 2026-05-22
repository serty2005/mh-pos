# POS UI Spec

Статус: актуальный cashier UI contract и целевой POS UI contract для полного пилота.

UI не является security boundary. Backend RBAC и application-layer checks остаются авторитетными.

## Реализовано Сейчас

Cashier UI in `pos-ui/src/pages/PosPage.vue` разделен на переиспользуемые компоненты терминала кассира в `pos-ui/src/pages/pos/*` и поддерживает:

- PIN login/session-based operator context;
- POSAppShell с фиксированной верхней action-панелью, центральной рабочей областью, правой панелью заказа/деталей там, где она есть, нижней status/navigation bar и скрываемым section menu поверх контента;
- строгие разделы shell из текущего кода: `floor` (`Залы / столы`), `order` (`Заказы`), `activity` (`Активность`), `reports` (`Отчеты`), `cash` (`Касса`);
- selected-order layout: слева поиск/группы/сетка блюд и услуг, справа current order panel; верхний контекст показывает выбранную строку как passive context, реальные действия комментария/курса и редактирования модификаторов, а не неподдержанные карточки редактирования строки;
- hall-orders layout: слева сетка столов, справа список активных заказов; создание заказа доступно только при праве `pos.order.create`, mock-фильтры официантов убраны, банкет показан только как disabled/backlog context badge;
- modal-процессы для оплаты, действий заказа, manager override отмены пречека, cash drawer и cancellation/refund;
- secondary operations (`cash drawer`, `closed orders`, `sync`) визуально отделены от основного пути продажи в отдельных разделах, dialog/drawer;
- employee shift open/close;
- cash session open/close;
- halls/tables selection в dense material-подобной сетке без скруглений;
- active orders by hall из backend для статусов столов и правой панели активных заказов;
- current order lookup by table;
- create order;
- add order line from menu;
- select modifiers for menu items with modifier groups;
- edit modifiers on active open order lines through the same modifier dialog;
- show selected modifiers under active order lines;
- сохранение курса подачи и комментария выбранной строки заказа через POS backend;
- transfer/move/split/fractional split для строки заказа отображаются только как disabled/backlog cards с причиной отсутствия backend/API flow, а не как активные операции;
- sell service items from a separate services section;
- change quantity;
- показывает safe backend error `errors.stopListConflict`, если POS backend отклоняет добавление или увеличение количества из-за active stop-list; UI не рассчитывает availability как источник истины;
- void line;
- issue precheck;
- cancel precheck through manager override dialog;
- reprint precheck copy;
- cash payment через payment modal;
- trusted manual card payment через payment modal;
- final check display after full payment без отдельного штатного шага `Закрыть заказ`;
- reprint final check copy;
- список закрытых заказов и detail/action rail в разделе `Активность`;
- closed orders UI использует bounded backend pagination: business-date filter, `limit=50`, `offset` previous/next controls; клиентский поиск применяется только к текущей странице и не является full-history search;
- detail закрытого заказа читает `GET /api/v1/checks/{id}/financial-operations?limit=50&offset=0` и показывает финансовые операции по выбранному final check: type, kind, amount, reason, employee/approver, business date, inventory disposition и created time;
- compatibility drawer закрытых заказов остается вспомогательной поверхностью для старых точек входа;
- full whole-check cancellation/refund и partial `order_line`/quantity cancellation/refund для closed orders через ledger endpoints с cashier reason и выбором inventory disposition;
- compatibility payment refund показан как явный fallback для захваченных оплат и подчиняется тому же UI refund-boundary guard: право на refund, открытая кассовая смена и текущая кассовая смена, отличная от исходной смены оплаты; backend остается авторитетным для финального чека, бизнес-даты и no-over-allocation enforcement.
- cash drawer events в отдельном dialog;
- sync status, outbox и local events в отдельном drawer; UI запрашивает outbox/local events с `limit=5` и не предполагает unbounded history response.
- unified blocking notice pattern for `noShift`, `noCashSession`, locked order and permission-disabled actions: причина, следующее действие и нужное permission id, если применимо.

UI calls backend APIs for authoritative state and does not compute authoritative totals.

Waiter mobile UI in `pos-ui/src/pages/WaiterPage.vue` и `pos-ui/src/pages/pos/useWaiterTerminal.ts` реализовано сейчас:

- route `/pos/waiter` является mobile-first surface, проверяемой отдельной Playwright spec под viewport `390x844`;
- использует только подтвержденные POS backend contracts: смена сотрудника, залы, столы, активные/current orders, menu items, добавление строки с modifier dialog, изменение quantity, void line, issue/reprint precheck;
- не требует кассовую смену и не показывает payment/refund/cash drawer controls по умолчанию;
- не считает authoritative totals, цены модификаторов, складские остатки или платежные статусы; показывает backend-provided order/precheck totals;
- после active issued precheck или locked order визуально блокирует меню, quantity и void controls; selected table/order/status остаются видимыми в mobile context strip;
- modifier dialog показывает required/min/max правила, validation message и disabled/loading submit state без локальной подмены backend validation;
- empty/loading/error/no-permission states идут через `vue-i18n` и reusable primitives из `pos-ui/src/shared/ui`.

KDS route реализовано сейчас как bounded readiness screen:

- route `/pos/kitchen` больше не generic shell; он показывает `запланировано далее`, отсутствующие backend contracts для kitchen tickets/lifecycle/stations/recall/printer и подготовленные статусы `new`, `accepted`, `in_progress`, `hold`, `ready`, `served`, `recall`, `cancelled`;
- readiness screen группирует будущий lifecycle и activation gates, но не показывает active buttons для `accepted`, `in_progress`, `ready`, `served`, `recall` или `cancelled`;
- активные KDS lifecycle actions не отображаются, потому что в POS backend нет kitchen ticket endpoints;
- hardware bump-bar/printer orchestration не описывается как реализованное.

## Runtime Error And Empty-State Handling

Реализовано сейчас:

- `requestOptional` converts backend optional current reads with `200 null` or `404 NOT_FOUND` to `null`.
- `GET /api/v1/employee-shifts/current` uses `200 null` when no personal employee shift is open, so normal terminal startup does not produce a browser network `404` for this read.
- `GET /api/v1/cash-shifts/current` and `GET /api/v1/orders/current?table_id=...` may still appear as `404` in browser network console; this remains expected backend empty-state behavior for those two endpoints, not a visible UI error.
- `GET /api/v1/orders/active?hall_id=...` возвращает пустой массив, когда в зале нет активных заказов; экран зала больше не генерирует mock активных заказов или mock статусов столов.
- Cashier terminal shows "нет открытой личной смены", "нет открытой кассовой смены" or "нет активного заказа" instead of setting blocking `statusError`/`orderError` for these optional empty states.
- Optional current reads are not retried on expected empty states.
- Payment mutation has no automatic retry. On `409 CONFLICT` from `POST /api/v1/prechecks/{id}/payments`, UI shows the localized backend `message_key` when present, otherwise `errors.conflict`, and invalidates current cash session, current order, order, prechecks, check and closed orders.
- On `409 SALE_STOP_LIST_CONFLICT` from order line add/increase commands, UI shows localized backend `errors.stopListConflict`; UI must not derive sale availability from stock balance or client-side stop-list logic.
- Payment buttons require an active precheck, positive amount, sufficient remaining total, payment permission and an open cash session. If a precheck exists but cash session is absent, UI blocks payment and shows the operator to open a cash session.
- Floor/menu sections distinguish no shift and no-permission states before showing ordinary empty tables/menu states.

## Backend Capability Vs UI Capability

Refund:

- Backend ledger capability реализован через `POST /api/v1/checks/{id}/cancellations` и `POST /api/v1/checks/{id}/refunds`.
- Backend ledger read capability для closed-order detail реализован через bounded `GET /api/v1/checks/{id}/financial-operations`; backend also exposes bounded `GET /api/v1/financial-operations` for local reporting filters, но POS UI не считает authoritative totals из него.
- Cashier UI has full whole-check cancellation through `POST /api/v1/checks/{id}/cancellations` guarded by `pos.precheck.cancel` and an open current cash session that belongs to the original shift.
- Cashier UI has full whole-check refund through `POST /api/v1/checks/{id}/refunds` guarded by `pos.payment.refund`, captured payment presence and an open current cash session different from the original payment shift.
- Диалог cancellation/refund отправляет `command_id`, reason, выбранный `inventory_disposition` и `operation_kind`. Whole-check режим не отправляет `items[]`; partial `order_line`/quantity режим строит выбор из immutable `check.snapshot.precheck_snapshot.lines` и отправляет `items[]` со scope `order_line`, `order_line_id`, `quantity`, `amount`, `currency` и `tax_amount`.
- Backend владеет totals, remaining compensable amount, shift/business-date boundaries and final operation enforcement.
- Ledger scopes `modifier_line`, `service_charge` и `tip` показаны только как unsupported текущего UI flow, потому что backend требует explicit snapshot для этих item scopes.
- Cashier UI держит compatibility route `POST /api/v1/payments/{id}/refund` как явный payment-level fallback для закрытых заказов с захваченными оплатами, визуально отделенный от основных check-level cancellation/refund actions.
- The compatibility route records a refund ledger operation and does not make UI authoritative for payment/check mutation.
- UI не реализует operator-facing archive/retention/export-plan/compaction controls и не предполагает загрузку всех закрытых заказов одной выдачей.
- UI shows refund/cancellation actions only when the required permission and cash session state are present; backend remains final authority for business-date/original-shift checks and no-over-compensation rules.
- UI does not expose refund for active issued prechecks with partial captured payments; refund runtime requires a finalized check.
- Backend remains final enforcement layer.

Reprint:

- Backend precheck/check reprint реализован из immutable snapshots.
- UI has reprint actions guarded by `pos.precheck.reprint` and `pos.check.reprint`.
- UI displays copy readiness through i18n text, not hardcoded source strings outside locale.
- Cancel/refund dialogs use safe operator wording through i18n and do not expose raw backend details, PIN, SQL or stack traces.

## Financial Boundaries

Реализовано сейчас:

- UI displays backend-provided order/precheck/check totals.
- Верхний cashier контекст показывает backend-provided totals скидок/надбавок, если они уже есть в заказе/precheck/check; он не открывает редактор скидок.
- UI sends payment amount and method to backend.
- UI does not calculate authoritative discount/tax/check totals.
- UI does not apply tax rules, discount rules or modifier prices as authoritative financial logic.
- UI validates modifier required/min/max constraints only as UX feedback before add/edit backend command; POS backend remains authoritative for modifier constraints, prices and totals.

Не реализовано сейчас:

- discount/surcharge editor in cashier UI;
- tax profile editor in cashier UI;
- modifier/service-charge/tip cancellation/refund UI;
- recipe expansion, automatic stock consumption and return-to-stock inventory moves for modifiers;
- inventory consumption UI.

Запланировано далее:

- discount/manual override controls must exist only after a confirmed cashier UI contract, permission model and safe policy selection flow;
- no UI-side authoritative financial calculation.

## RBAC Visibility

Реализовано сейчас:

- UI maps backend permission ids in `pos-ui/src/shared/rbac.ts`.
- Critical actions are hidden/disabled based on permissions for UX.
- Backend validates permissions again.

Relevant permissions:

- `pos.order.create`
- `pos.order.add_line`
- `pos.order.change_quantity`
- `pos.order.void_line`
- `pos.precheck.issue`
- `pos.precheck.view`
- `pos.precheck.cancel.request`
- `pos.precheck.reprint`
- `pos.payment.cash`
- `pos.payment.card.manual`
- `pos.payment.other`
- `pos.payment.refund`
- `pos.check.view`
- `pos.check.reprint`

## Locale And Text

Requirements:

- User-visible labels, dialogs, validation messages, notifications and empty states go through `vue-i18n`.
- Russian UI strings belong in locale definitions, not scattered hardcoded source code.
- Error display must not expose raw Go errors, SQL errors, stack traces, request dumps, PINs, tokens or sensitive payloads.

## POS UI Component Standardization / Design-System Rules

Реализовано сейчас:

- Cashier POS UI имеет внутренний reusable presentation layer в `pos-ui/src/shared/ui`.
- Первый слой primitives/composites включает `PosButton`, `PosContextButton`, `PosDialog`, `PosSectionHeader`, `PosTabs`, `PosPagination`, `PosQuantityStepper`, `PosBanner`, `PosEmptyState`, `PosStatusStrip`, `PosMetricCard`, `PosActionRail`, `PosPanel`, `PosDataRow`, `PosFormRow` и `PosSkeleton`.
- Эти компоненты являются dumb/presentational: они принимают labels, variant/state props и callbacks, но не получают `terminal` и не владеют cashier business logic.
- `PosPage`, `WaiterPage`, `KitchenPage`, `PosMenuGrid`, `PosOrderRail`, `PosActivitySection`, `PosCashSection`, `PosReportsSection`, `PosPaymentDialog`, `PosActionsDialog`, `ModifierSelectionDialog`, `RefundDialog`, `PrecheckCancelDialog`, `CashDrawerDialog`, `SyncDrawer` и `ClosedOrdersDrawer` уже используют часть этого слоя для повторяющихся кнопок, tabs/chips, dialog shell, section header, action rail, panels, data rows, status/metric cards, empty/error/loading states и quantity steppers.
- `pos-ui/src/styles.css` содержит общий POS scrollbar contract через `.pos-scrollarea`, `.pos-scrollarea-y`, `.pos-scrollarea-x` и `.pos-scrollbar-thin`: thin scrollbar, semantic colors, touch-friendly overflow и отсутствие неуправляемого горизонтального скролла в основных cashier surfaces.

Правила для следующих изменений:

- Новый POS UI элемент сначала проектируется как reusable primitive/composite или расширение существующего компонента в `pos-ui/src/shared/ui`, если он может повторяться.
- Feature-компоненты не должны накапливать локальные варианты одинаковых кнопок, табов, модалок, карточек, rows, panels, scrollbar и action panels.
- Пользовательский текст для новых компонентов передается из feature layer через `vue-i18n`; primitive не хардкодит человекочитаемые labels.
- Цвета и visual state должны идти через semantic CSS tokens или уже существующие POS utility classes, а не через локальные raw colors.
- Backend-неподдержанные действия остаются disabled/backlog presentation с причиной, а не активными кнопками.

Запланировано далее:

- Постепенно мигрировать оставшиеся legacy/compatibility поверхности (`OrderWorkspace`, `CatalogCheckoutPanel`, `FloorTableSelector`, `PosFloorSection`, старые checkout/floor helper panels и отдельные feature-local table/list rows) на тот же `shared/ui` layer без изменения backend/API behavior.

## Разделение Интерфейсов

Реализовано сейчас:

- `/pos` and `/pos/cashier` load the current cashier pilot terminal.
- `/pos/waiter` loads the current waiter mobile order/precheck runtime without payment/refund/cash drawer authority.
- `/pos/kitchen` loads an honest KDS readiness screen; active KDS runtime remains blocked by absent backend endpoints.
- Код cashier terminal разделен на composable для runtime/API state и presentation components для POS shell, floor, menu grid, order rail, payment/actions modals и utility panels.
- Bottom quick access bar и скрываемое side menu являются основным navigation shell для POS runtime.
- Раздел `order` / `Заказы` является основным рабочим экраном: search/category tabs + dish/service grid + current order panel + payment/actions modal поверх текущего заказа.
- Раздел `floor` / `Залы / столы` является рабочим экраном выбора зала/стола: table grid + active orders panel + modal создания заказа на данных `GET /api/v1/orders/active?hall_id=...`.
- Раздел `activity` / `Активность` показывает закрытые заказы, bounded pagination/filter текущей страницы, детали оплат, financial operations и refund/cancellation/reprint actions по текущим backend-правам.
- Раздел `reports` / `Отчеты` показывает только ограниченные операционные сводки по already-loaded closed orders, оплатам и sync health; Cloud reporting UI не входит в cashier runtime.
- Раздел `cash` / `Касса` использует текущую операционную секцию кассы: личная смена, кассовая смена, cash drawer actions и sync diagnostics.
- Разделов `delivery` и `settings` в текущем cashier shell нет; delivery/channel runtime и backoffice/settings surfaces остаются вне POS cashier UI до отдельного backend/API контракта.
- Основные route components загружаются через lazy imports/code splitting, чтобы снизить нагрузку на initial bundle.

Запланировано до полного пилота:

- `/pos/waiter` должен расширяться только в пределах подтвержденных backend contracts; он остается единственным mobile layout полного пилота, остальные modes не получают мобильные варианты;
- `/pos/kitchen` должен перейти от readiness screen к advanced KDS lifecycle screen после появления backend routes;
- `/pos/manager` остается вне POS UI runtime, если manager операции полностью покрыты Cloud UI;
- kitchen UI читает kitchen tickets, показывает статусы `new`, `accepted`, `in_progress`, `hold`, `ready`, `served`, `recall`, `cancelled` и отправляет status actions, которые backend превращает в `KitchenTicketStatusChanged`/`ItemServed`;
- kitchen UI дает повару сценарии приемки поставки, catalog suggestion, просмотра техкарты, `RecipeChangeSuggested` и редактирования stop-list через backend routes.

## POS Shell Visual Contract

Реализовано сейчас:

- Основной POS экран рассчитан на 1366x768 без горизонтального скролла.
- Основные панели, кнопки, карточки блюд, карточки столов и строки заказа используют прямые углы.
- Side menu открывается по левой кнопке нижней панели, накладывается поверх интерфейса, не сдвигает layout и закрывается после выбора раздела.
- Замок блокировки POS находится в правой части верхней context bar.
- Правая панель заказа не показывает заголовки `Заказ #...` и `Стол ...`; контекст заказа вынесен в нижнюю панель.
- Правая панель `Залы / столы` показывает только активные заказы, сгруппированные по залу.

Запланировано далее:

- Подключить backend/API быстрого чека со столом по умолчанию и проверкой отдельного permission.

Вне текущего объема:

- `/pos/manager` является route shell, пока manager operations покрываются Cloud UI.
- `/pos/kitchen` не является готовым runtime: реализован только readiness screen до появления backend/API contracts.

## Вне Текущего Объема

Вне текущего объема полного пилота:

- delivery/channel screens;
- real PSP terminal integration UI;
- fiscal device operation UI;
- Cloud inventory/procurement back-office UI inside POS UI;
- hardware bump-bar/printer UI and rich KDS analytics beyond bounded pilot timing metrics;
- rich partial cancellation/refund ledger UI beyond current order-line/quantity actions;
- discount/surcharge cashier editor and tax policy UI on top of existing backend pricing foundation.

## Full Pilot POS UI Acceptance

Запланировано до полного пилота:

- waiter mobile viewport `390x844`: login, table selection, active order creation, menu/modifier selection, quantity change, void line, issue/reprint precheck, no payment controls by default;
- kitchen readiness route: absent backend contracts are visible, `запланировано далее` is visible, no wording claims `готовый runtime` / `реализовано сейчас` for KDS lifecycle actions;
- after backend routes appear, kitchen tablet/desktop viewport must cover ticket list by station/status, accept/start/hold/ready/served/recall/cancel actions, receipt capture, recipe suggestion, stop-list edit, safe localized error handling and sync pending indicator;
- cashier/KDS/manager routes are checked at desktop/tablet widths only; mobile acceptance belongs to waiter route;
- cashier regression: current cashier flow remains unchanged and still passes payment/refund/sync e2e tests;
- all new labels, empty states, errors and dialog text are added through `vue-i18n`.

## Выбор скидок и надбавок по policy

Статус: backend/API foundation реализован, cashier UI editor вне текущего объема.

POS API client содержит функции для `pricing_policy` и применения скидок/надбавок, а backend хранит policy-id-backed adjustments. Текущий cashier UI не показывает активный discount/surcharge editor, потому что безопасный cashier contract для выбора policy, permission wording, audit UX и pilot acceptance еще не зафиксирован. В POS shell отображаются только backend-provided totals скидок/надбавок.
