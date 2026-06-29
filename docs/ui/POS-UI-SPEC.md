# POS UI Spec

Статус: актуальный cashier UI contract и целевой POS UI contract для полного пилота. Активный POS UI runtime находится в `pos-ui-g`; старый Vue `pos-ui` удален из runtime tree.

UI не является security boundary. Backend RBAC и application-layer checks остаются авторитетными.

## Реализовано Сейчас

Cashier UI in `pos-ui-g/src/App.tsx` и `pos-ui-g/src/components/**` разделен на переиспользуемые компоненты терминала кассира и поддерживает:

- PIN login/session-based operator context;
- POSAppShell с фиксированной верхней action-панелью, центральной рабочей областью, правой панелью заказа/деталей там, где она есть, нижней status/navigation bar и скрываемым section menu поверх контента;
- верхний контекст кассира показывает фактические backend/session identifiers и readiness: restaurant id, actor, node device, selected table/order, personal shift, cash session и backend session state; значения берутся из backend/auth store и не создают отдельную бизнес-модель;
- строгие разделы shell из текущего кода: `floor` (`Залы / столы`), `order` (`Заказы`), `activity` (`Активность`), `reports` (`Отчеты`), `cash` (`Касса`);
- selected-order layout: слева поиск/группы/сетка блюд и услуг, справа current order panel; верхний контекст показывает выбранную строку как passive context, реальные действия комментария/курса и редактирования модификаторов, а не неподдержанные карточки редактирования строки;
- режим продажи без залов и столов при отсутствии `table-mode` остается в разделе `order`: слева показывается крупная кнопка `+` для создания counter-order, справа показываются широкие строки последних закрытых заказов с типом оплаты и суммой, клик открывает модалку состава заказа;
- после `POST /api/v1/orders` для counter-order UI выбирает созданный заказ по `order_id`, а не через `table_id`, поэтому созданный заказ сразу становится текущим без reload;
- в режиме без залов и столов ручная кнопка пречека не показывается: оплата идет через `POST /api/v1/orders/{id}/counter-payment`, где POS backend автоматически выпускает precheck под капотом и не требует отдельного операторского шага печати;
- нижний блок итогов текущего заказа кликабелен и открывает модалку расшифровки backend totals: подытог, скидка, налог и итог;
- hall-orders layout: слева сетка столов, справа список активных заказов; создание заказа доступно только при праве `pos.order.create`, mock-фильтры официантов убраны, банкет показан только как disabled/backlog context badge; loading/error/empty/no-permission states используют reusable `PosBanner`, `PosEmptyState` и `PosSkeleton`;
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
- issue precheck в table-mode;
- cancel precheck through manager override dialog;
- reprint precheck copy;
- cash payment через payment modal;
- trusted manual card payment через payment modal;
- final check display after full payment без отдельного штатного шага `Закрыть заказ`;
- reprint final check copy;
- список закрытых заказов и панель detail/action rail в разделе `Активность`;
- UI закрытых заказов использует ограниченную backend-пагинацию: фильтр операционной даты, `limit=26` для размера страницы 25 + проверка следующей страницы, `offset` для перехода назад/вперед; клиентский поиск применяется только к текущей странице и не является поиском по всей истории;
- detail закрытого заказа читает `GET /api/v1/checks/{id}/financial-operations?limit=50&offset=0` и показывает финансовые операции по выбранному final check: type, kind, amount, reason, employee/approver, business date, inventory disposition и created time;
- compatibility drawer закрытых заказов остается вспомогательной поверхностью для старых точек входа;
- full whole-check cancellation/refund и partial `order_line`/quantity cancellation/refund для closed orders через ledger endpoints с cashier reason и выбором inventory disposition;
- compatibility payment refund показан как явный fallback для захваченных оплат и подчиняется тому же UI refund-boundary guard: право на refund, открытая кассовая смена и текущая кассовая смена, отличная от исходной смены оплаты; backend остается авторитетным для финального чека, бизнес-даты и no-over-allocation enforcement.
- cash drawer events в отдельном dialog;
- sync status, outbox и local events в отдельном drawer; UI запрашивает outbox/local events с `limit=5` и не предполагает неограниченный history response.
- Единый blocking notice pattern для `noShift`, `noCashSession`, locked order и действий, отключенных по permission: причина, следующее действие и нужное permission id, если применимо.

UI вызывает backend API за авторитетным состоянием и не считает авторитетные totals.

Waiter mobile surface в `pos-ui-g` реализовано сейчас:

- terminal mode `waiter` является mobile-first surface и автоматически выбирается на mobile viewport; он проверяется отдельной Playwright spec под viewport `390x844`;
- использует только подтвержденные POS backend contracts: смена сотрудника, залы, столы, активные/current orders, menu items, добавление строки с modifier dialog, изменение quantity, void line с явной причиной, issue/reprint precheck;
- не требует кассовую смену и не показывает payment/refund/cash drawer/fiscal controls по умолчанию;
- не считает authoritative totals, цены модификаторов, складские остатки или платежные статусы; показывает backend-provided order/precheck totals;
- явно показывает mobile context: текущий стол, заказ, статус и границы полномочий официанта; order/precheck runtime доступен, payment/refund/cash drawer/fiscal authority скрыта;
- после active issued precheck или locked order визуально блокирует меню, quantity и void controls; selected table/order/status остаются видимыми в mobile context strip, а меню дополнительно показывает lock badge без добавления новых действий;
- modifier dialog показывает required/min/max правила, validation message и disabled/loading submit state без локальной подмены backend validation; общий `PosDialog` ограничивает высоту карточки и прокручивает body, чтобы длинный список modifiers оставался usable на mobile;
- viewport `390x844` держит compact context/authority surface, touch-friendly table/menu/order rows и locked-state reason без payment/refund/cash drawer/fiscal controls;
- empty/loading/error/no-permission states идут через `pos-ui-g/src/shared/i18n` и reusable primitives из `pos-ui-g/src/shared/ui`.

`pos-ui-g` KDS mode реализовано сейчас как backend-backed kitchen runtime:

- terminal mode `kds` в `pos-ui-g/src/App.tsx` использует нижний quick access только с разделами `Заказы`, `Склад`, `Кухня`;
- раздел `Заказы` читает `GET /api/v1/kitchen/order-queue`, показывает `Очередь` и `Готово к выдаче`, order tiles с elapsed time, временем создания/последнего изменения, backend `kitchen_order_status`, блюдами и допустимыми ticket actions;
- active buttons отображаются только для допустимых backend transitions `accept`, `start`, `hold`, `ready`, `serve`, `recall`, `cancel`;
- после status action UI не подменяет truth оптимистично, а перечитывает `GET /api/v1/kitchen/order-queue`;
- раздел `Склад` использует full catalog picker поверх `GET /api/v1/catalog/items` и формы `Приемка`, `Ревизия`, `Списание`, `Приготовление`;
- раздел `Кухня` показывает `Техкарты`, `Предложения`, `Стоп-лист`, `Мои предложения`, читает recipe response с `catalog_item`, `recipe_version`, `ingredients`, отправляет `CatalogItemChangeSuggested`/`RecipeChangeSuggested` и отправляет stop-list update commands через `POST /api/v1/kitchen/stop-list-updates`;
- вкладка `Стоп-лист` читает `GET /api/v1/kitchen/stop-list`, показывает только safe local overlay и outbox metadata (`sync_state`, `outbox_status`, `command_id`, attempts) без raw `payload_json`/`last_error`, а sale blocking остается backend-authoritative;
- `Мои предложения` получает Cloud approve/reject/request-changes через `proposal_feedback` после sync и не применяет master-data локально без publication;
- loading/error/empty/no-permission states идут через `pos-ui-g/src/shared/ui` и `pos-ui-g/src/shared/i18n`;
- hardware bump-bar/printer orchestration и расширенная KDS analytics не описываются как реализованные.

## Runtime Error And Empty-State Handling

Реализовано сейчас:

- `requestOptional` преобразует optional current reads backend с `200 null` или `404 NOT_FOUND` в `null`.
- Global dialog и inline banners показывают безопасный support code для оператора и разработчика: backend `correlation_id`, если он есть, иначе стабильный `error_code` вроде `INVALID_RESPONSE`, `NETWORK_ERROR` или backend business code.
- `GET /api/v1/employee-shifts/current` использует `200 null`, когда личная смена сотрудника не открыта, поэтому обычный startup терминала не создает browser network `404` для этого чтения.
- `GET /api/v1/cash-shifts/current` и `GET /api/v1/orders/current?table_id=...` могут оставаться `404` в browser network console; это ожидаемое backend empty-state behavior для этих двух endpoints, а не видимая UI ошибка.
- `GET /api/v1/orders/active?hall_id=...` возвращает пустой массив, когда в зале нет активных заказов; экран зала больше не генерирует mock активных заказов или mock статусов столов.
- Cashier terminal показывает "нет открытой личной смены", "нет открытой кассовой смены" или "нет активного заказа" вместо установки blocking `statusError`/`orderError` для этих optional empty states.
- Optional current reads не повторяются на ожидаемых empty states.
- Payment mutation не имеет automatic retry. При `409 CONFLICT` от `POST /api/v1/prechecks/{id}/payments` UI показывает локализованный backend `message_key`, если он есть, иначе `errors.conflict`, и инвалидирует current cash session, current order, order, prechecks, check и closed orders.
- При `409 SALE_STOP_LIST_CONFLICT` от команд добавления/увеличения order line UI показывает локализованный backend `errors.stopListConflict`; UI не должен выводить sale availability из stock balance или client-side stop-list logic.
- Payment buttons требуют active precheck, положительную сумму, достаточный remaining total, payment permission и открытую cash session. Если precheck есть, но cash session отсутствует, UI блокирует payment и показывает оператору необходимость открыть cash session.
- Floor/menu sections различают no shift и no-permission states до показа обычных empty tables/menu states.

## Backend Capability Vs UI Capability

Refund:

- Backend ledger capability реализован через `POST /api/v1/checks/{id}/cancellations` и `POST /api/v1/checks/{id}/refunds`.
- Backend ledger read capability для closed-order detail реализован через ограниченный `GET /api/v1/checks/{id}/financial-operations`; backend также предоставляет ограниченный `GET /api/v1/financial-operations` для локальных reporting filters, но POS UI не считает authoritative totals из него.
- Cashier UI поддерживает full whole-check cancellation через `POST /api/v1/checks/{id}/cancellations` под guard `pos.precheck.cancel` и при открытой текущей кассовой смене исходной смены.
- Cashier UI поддерживает full whole-check refund через `POST /api/v1/checks/{id}/refunds` под guard `pos.payment.refund`, при наличии captured payment и открытой текущей кассовой смене, отличной от исходной смены оплаты.
- Диалог cancellation/refund отправляет `command_id`, reason, выбранный `inventory_disposition` и `operation_kind`. Whole-check режим не отправляет `items[]`; partial `order_line`/quantity режим строит выбор из immutable `check.snapshot.precheck_snapshot.lines` и отправляет `items[]` со scope `order_line`, `order_line_id`, `quantity`, `amount`, `currency` и `tax_amount`.
- Backend владеет totals, remaining compensable amount, shift/business-date boundaries and final operation enforcement.
- Ledger scopes `modifier_line`, `service_charge` и `tip` показаны только как unsupported текущего UI flow, потому что backend требует explicit snapshot для этих item scopes.
- Cashier UI держит compatibility route `POST /api/v1/payments/{id}/refund` как явный payment-level fallback для закрытых заказов с захваченными оплатами, визуально отделенный от основных check-level cancellation/refund actions.
- Compatibility route записывает refund ledger operation и не делает UI авторитетным для payment/check mutation.
- UI не реализует operator-facing archive/retention/export-plan/compaction controls и не предполагает загрузку всех закрытых заказов одной выдачей.
- UI показывает refund/cancellation actions только при наличии нужного permission и состояния cash session; backend остается final authority для business-date/original-shift checks и правил no-over-compensation.
- UI не показывает refund для active issued prechecks с partial captured payments; refund runtime требует finalized check.
- Backend остается final enforcement layer.

Reprint:

- Backend precheck/check reprint реализован из immutable snapshots.
- UI has reprint actions guarded by `pos.precheck.reprint` and `pos.check.reprint`.
- UI displays copy readiness through i18n text, not hardcoded source strings outside locale.
- Cancel/refund dialogs use safe operator wording through i18n and do not expose raw backend details, PIN, SQL or stack traces.
- Для QR-билетов backend API существует отдельно от базового cashier flow; POS modal заказа для клика по билетной позиции с QR-флагом номенклатуры и ticket reprint остается `запланировано далее`, потому что текущий POS menu DTO не содержит UI-контракт QR-флага позиции.

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

- UI получает backend permission ids из auth/session responses и мапит POS role через `pos-ui-g/src/shared/backendMappers.ts` / `pos-ui-g/src/types.ts`.
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

- User-visible labels, dialogs, validation messages, notifications and empty states go through the active UI locale layer in `pos-ui-g/src/shared/i18n`.
- Russian UI strings belong in locale definitions, not scattered hardcoded source code.
- Error display must not expose raw Go errors, SQL errors, stack traces, request dumps, PINs, tokens or sensitive payloads.

## POS UI Component Standardization / Design-System Rules

Реализовано сейчас:

- Cashier POS UI имеет reusable presentation layer в `pos-ui-g/src/shared/ui`.
- Первый слой React primitives/composites включает `PosButton`, `PosIconButton`, `PosContextButton`, `PosDialog`, `PosSectionHeader`, `PosTabs`, `PosBottomNav`, `PosSegmentedControl`, `PosSelectableChip`, `PosSelectableTile`, `PosSearchInput`, `PosPagination`, `PosQuantityStepper`, `PosBanner`, `PosEmptyState`, `PosStatusBadge`, `PosInlineStatusBadge`, `PosStatusStrip`, `PosMetricCard`, `PosActionRail`, `PosRailHeader`, `PosDataRow`, `PosFormRow`, `PosSkeleton`.
- В `pos-ui-g` реализовано сейчас: shell bottom navigation/icon controls, side drawer mode items, PIN/pairing/payment keypad controls, floor hall tabs/table tiles/active-order rail, order category/search/filter controls/menu tiles/order rail header, action-dialog course/modifier selectors, activity search/check rows/status badges, cash rail header/ledger badges, payment/refund/cash drawer selector controls и KDS catalog picker/status badges используют shared primitives. Отдельные узкоспециализированные keypad rows могут оставаться локальными, когда layout или тестовые IDs завязаны на сценарий.
- POS UI читает effective Edge entitlement snapshot и скрывает floor, KDS, warehouse stock и waiter modes при отсутствии соответствующих grants. Если недоступен только дополнительный module entitlement, базовый cashier terminal не показывается как заблокированный: раздел `order` остается доступен для counter sale при наличии локальных данных. Это UX-слой; Edge backend gates являются авторитетными. Canonical module IDs описаны в `docs/backend/LICENSE-ENTITLEMENTS.md`.
- POS UI перечитывает Edge runtime state и entitlement snapshot в фоне, пока терминал разблокирован. Уже примененные Edge backend изменения из Cloud sync, включая license grants, меню/блюда, залы, столы и pricing policies, появляются в интерфейсе без обновления страницы.
- PIN login screen автоматически отправляет введенный PIN после достижения минимальной длины и короткой паузы ввода; ручная кнопка входа остается fallback action. Неверный auto-attempt не раскрывает PIN и не очищает ввод, чтобы оператор мог добрать более длинный PIN.
- Эти компоненты являются dumb/presentational: они принимают labels, variant/state props и callbacks, но не получают `terminal` и не владеют cashier business logic.
- `PosPage`, `WaiterPage`, `KitchenPage`, `PosMenuGrid`, `PosOrderRail`, `PosFloorSection`, `PosActivitySection`, `PosCashSection`, `PosReportsSection`, `PosPaymentDialog`, `PosActionsDialog`, `ModifierSelectionDialog`, `RefundDialog`, `PrecheckCancelDialog`, `CashDrawerDialog`, `SyncDrawer` и `ClosedOrdersDrawer` уже используют часть этого слоя для повторяющихся кнопок, tabs/chips, dialog shell, section header, action rail, panels, data rows, status/metric/readiness cards, empty/error/loading states, menu skeleton cards и quantity steppers.
- `pos-ui-g/src/index.css` содержит общий POS scrollbar contract через `.pos-scrollarea`, `.pos-scrollarea-y`, `.pos-scrollarea-x` и `.pos-scrollbar-thin`: thin scrollbar, semantic colors, touch-friendly overflow и отсутствие неуправляемого горизонтального скролла в основных cashier surfaces.

Правила для следующих изменений:

- Новый POS UI элемент сначала проектируется как reusable primitive/composite или расширение существующего компонента в `pos-ui-g/src/shared/ui`, если он может повторяться.
- Feature-компоненты не должны накапливать локальные варианты одинаковых кнопок, табов, модалок, карточек, rows, panels, scrollbar и action panels.
- Пользовательский текст для новых компонентов передается из feature layer через `pos-ui-g/src/shared/i18n`; primitive не хардкодит человекочитаемые labels.
- Цвета и visual state должны идти через semantic CSS tokens или уже существующие POS utility classes, а не через локальные raw colors.
- Backend-неподдержанные действия остаются disabled/backlog presentation с причиной, а не активными кнопками.

Запланировано далее:

- Постепенно мигрировать оставшиеся legacy/compatibility поверхности (`OrderWorkspace`, `CatalogCheckoutPanel`, `FloorTableSelector`, старые checkout/floor helper panels и отдельные feature-local table/list rows) на тот же `shared/ui` layer без изменения backend/API behavior.

## Разделение Интерфейсов

Реализовано сейчас:

- `/pos` и `/pos/cashier` загружают текущий cashier pilot terminal.
- Terminal mode `waiter` загружает текущий mobile order/precheck runtime без payment/refund/cash drawer/fiscal authority.
- `pos-ui-g` kitchen mode загружает backend-backed KDS, stock input и proposal runtime.
- Код cashier terminal разделен на composable для runtime/API state и presentation components для POS shell, floor, menu grid, order rail, payment/actions modals и utility panels.
- Bottom quick access bar и скрываемое side menu являются основным navigation shell для POS runtime.
- Раздел `order` / `Заказы` является основным рабочим экраном: search/category tabs + dish/service grid + current order panel + payment/actions modal поверх текущего заказа.
- При `table-mode=false` этот же раздел является основным экраном продажи без залов, столов и ручного пречека: состояние без выбранного заказа показывает крупный `+`, справа последние закрытые заказы, состояние активного заказа показывает меню слева и строки текущего заказа справа.
- Раздел `floor` / `Залы / столы` является рабочим экраном выбора зала/стола: table grid + active orders panel + modal создания заказа на данных `GET /api/v1/orders/active?hall_id=...`.
- Раздел `activity` / `Активность` показывает закрытые заказы, ограниченную пагинацию и фильтр текущей страницы, детали оплат, financial operations и refund/cancellation/reprint actions по текущим backend-правам.
- В React/Vite `pos-ui-g` раздел `activity` получает текущую ограниченную страницу `closedOrders` из `POSContext`; `POSContext` вызывает `listClosedOrders({ businessDateLocal, limit: 26, offset })`, UI показывает previous/next по backend page-size 25, поиск работает только по загруженной странице, а operator-facing archive/retention/export/compaction controls не реализованы.
- Раздел `reports` / `Отчеты` показывает только ограниченные операционные сводки по already-loaded closed orders, оплатам и sync health; Cloud reporting UI не входит в cashier runtime.
- Раздел `cash` / `Касса` использует текущую операционную секцию кассы: личная смена, кассовая смена, cash drawer actions и sync diagnostics.
- Разделов `delivery` и `settings` в текущем cashier shell нет; delivery/channel runtime и backoffice/settings surfaces остаются вне POS cashier UI до отдельного backend/API контракта.
- Основные route components загружаются через lazy imports/code splitting, чтобы снизить нагрузку на initial bundle.

Реализовано сейчас для `pos-ui-g` kitchen mode:

- kitchen UI читает kitchen tickets, показывает статусы `new`, `accepted`, `in_progress`, `hold`, `ready`, `served`, `recall`, `cancelled` и отправляет status actions, которые backend превращает в `KitchenTicketStatusChanged`/`ItemServed`;
- `pos-ui-g` KDS mode реализован как production shell с нижним quick access только `Заказы`, `Склад`, `Кухня`; внутри разделов используются верхние вкладки:
- `Заказы`: `Очередь`, `Готово к выдаче` с order tiles (время, статусы, блюда, actions);
- `Склад`: формы `Приемка`, `Ревизия`, `Списание`, `Приготовление` с full catalog picker;
- `Кухня`: `Техкарты`, `Предложения`, `Стоп-лист`, `Мои предложения`;
- после kitchen action UI перечитывает backend и не держит оптимистичный статус как source of truth;
- ошибки показываются безопасно через локализованные keys в `pos-ui-g/shared/i18n`;
- pure helpers KDS вынесены в `pos-ui-g/src/components/kitchen/kitchenHelpers.ts`; presentational слой очереди, stock forms, recipe workspace, catalog suggestions, stop-list workspace и proposal list вынесен в `KitchenOrdersTab`, `KitchenStockTab`, `KitchenRecipeTab`, `KitchenCatalogSuggestionForm`, `KitchenStopListTab` и `KitchenProposalList`; runtime component остается владельцем state/loaders/submit handlers и не меняет backend contracts или allowed transitions;
- kitchen UI дает повару сценарии приемки поставки, ревизии, списания, приготовления, catalog suggestion, просмотра техкарты и `RecipeChangeSuggested` через backend routes.
- профильный smoke `scripts/seed-dev-system.py --run-kitchen-process-smoke` проверяет UI-backed backend surface для KDS lifecycle, recall/serve-again, stock forms и proposal feedback через HTTP routes.

Запланировано далее:

- Waiter terminal mode должен расширяться только в пределах подтвержденных backend contracts; он остается единственным mobile layout полного пилота, остальные modes не получают мобильные варианты;
- `/pos/manager` остается вне POS UI runtime, если manager операции полностью покрыты Cloud UI;
- `/pos/kitchen` / `pos-ui-g` должен расширяться только поверх подтвержденных backend routes;
- расширенный production workflow polish для stop-list beyond минимальной backend-backed формы и sync indicator.

## POS Shell Visual Contract

Реализовано сейчас:

- Основной POS экран рассчитан на 1366x768 без горизонтального скролла.
- Верхний cashier context bar, hover/focus states и main workspace используют единые shell tokens, чтобы primary flow и второстепенные sections читались как один терминал без изменения backend-команд.
- Основные панели, кнопки, карточки блюд, карточки столов и строки заказа используют прямые углы.
- Side menu открывается по левой кнопке нижней панели, накладывается поверх интерфейса, не сдвигает layout и закрывается после выбора раздела.
- Замок блокировки POS находится в правой части верхней context bar.
- Правая панель заказа не показывает заголовки `Заказ #...` и `Стол ...`; контекст заказа вынесен в нижнюю панель.
- Правая панель `Залы / столы` показывает только активные заказы, сгруппированные по залу.

Запланировано далее:

- Подключить backend/API быстрого чека со столом по умолчанию и проверкой отдельного permission.
- Настройка Edge после оплаты: возвращаться на экран с `+` или автоматически создавать следующий counter-order.
- POS modal закрытого заказа должен показывать QR ticket units для билетных позиций после появления явного QR-флага в POS menu/catalog DTO; повторная печать билета должна использовать дату первого прохода, если ticket activation уже была.

Вне текущего объема:

- `/pos/manager` является route shell, пока manager operations покрываются Cloud UI.
- `/pos/kitchen` не покрывает bump-bar/printer orchestration и расширенную KDS analytics.

## Вне Текущего Объема

Вне текущего объема полного пилота:

- delivery/channel screens;
- real PSP terminal integration UI;
- fiscal device operation UI;
- Cloud inventory/procurement back-office UI inside POS UI;
- hardware bump-bar/printer UI и расширенная KDS analytics beyond bounded pilot timing metrics;
- rich partial cancellation/refund ledger UI beyond current order-line/quantity actions;
- discount/surcharge cashier editor and tax policy UI on top of existing backend pricing foundation.

## Full Pilot POS UI Acceptance

Запланировано до полного пилота:

- waiter mobile viewport `390x844`: login, table selection, active order creation, menu/modifier selection, quantity change, void line, issue/reprint precheck, no payment controls by default;
- kitchen route: backend-backed order queue/status lifecycle, stock forms, recipe view and proposal forms work with safe localized error handling and no UI-authoritative status decisions;
- запланировано далее: bump-bar/printer orchestration и расширенная KDS analytics;
- cashier/KDS/manager routes проверяются только на desktop/tablet ширинах; mobile acceptance относится к waiter route;
- cashier regression: current cashier flow remains unchanged and still passes payment/refund/sync e2e tests;
- all new labels, empty states, errors and dialog text are added through the active UI locale layer (`pos-ui-g/src/shared/i18n`).

## Выбор скидок и надбавок по policy

Статус: backend/API foundation реализован, cashier UI editor вне текущего объема.

POS API client содержит функции для `pricing_policy` и применения скидок/надбавок, а backend хранит policy-id-backed adjustments. Текущий cashier UI не показывает активный discount/surcharge editor, потому что безопасный cashier contract для выбора policy, permission wording, audit UX и pilot acceptance еще не зафиксирован. В POS shell отображаются только backend-provided totals скидок/надбавок.
