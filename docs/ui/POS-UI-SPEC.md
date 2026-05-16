# POS UI Spec

Статус: актуальный cashier UI contract для frozen pilot.

UI не является security boundary. Backend RBAC и application-layer checks остаются авторитетными.

## Реализовано Сейчас

Cashier UI in `pos-ui/src/pages/PosPage.vue` разделен на переиспользуемые компоненты терминала кассира в `pos-ui/src/pages/pos/*` и поддерживает:

- PIN login/session-based operator context;
- POSAppShell с фиксированной верхней action-панелью, центральной рабочей областью, стабильной правой панелью, нижней status/navigation bar и скрываемым side menu поверх контента;
- строгие разделы shell: `ЗАКАЗ`, `Залы / столы`, `Доставка`, `Смена`, `Аналитика`, `Настройки`;
- selected-order layout: слева категории и сетка блюд, справа current order panel фиксированной ширины, верхняя кнопка `Сохранить`/`Пречек`/`Чек` выровнена по ширине правой панели;
- hall-orders layout: слева сетка столов, справа только список активных заказов, верхняя кнопка `Быстрый чек` выровнена по ширине правой панели;
- modal-процессы для оплаты, действий заказа, manager override отмены пречека, cash drawer и refund;
- secondary operations (`cash drawer`, `closed orders`, `sync`) визуально отделены от основного пути продажи в отдельных разделах, dialog/drawer;
- employee shift open/close;
- cash session open/close;
- halls/tables selection в dense material-подобной сетке без скруглений;
- active orders by hall из backend для статусов столов и правой панели активных заказов;
- current order lookup by table;
- create order;
- add order line from menu;
- select modifiers for menu items with modifier groups;
- show selected modifiers under active order lines;
- сохранение курса подачи и комментария выбранной строки заказа через POS backend;
- sell service items from a separate services section;
- change quantity;
- void line;
- issue precheck;
- cancel precheck through manager override dialog;
- reprint precheck copy;
- cash payment через payment modal;
- trusted manual card payment через payment modal;
- final check display after full payment без отдельного штатного шага `Закрыть заказ`;
- reprint final check copy;
- список закрытых заказов и detail/action rail в разделе `Активность`;
- compatibility drawer закрытых заказов остается вспомогательной поверхностью для старых точек входа;
- compatibility refund for captured payment from closed orders when operator has permission and cash session is open.
- cash drawer events в отдельном dialog;
- sync status, outbox и local events в отдельном drawer.
- unified blocking notice pattern for `noShift`, `noCashSession`, locked order and permission-disabled actions: причина, следующее действие и нужное permission id, если применимо.

UI calls backend APIs for authoritative state and does not compute authoritative totals.

## Runtime Error And Empty-State Handling

Реализовано сейчас:

- `requestOptional` converts backend optional current reads with `200 null` or `404 NOT_FOUND` to `null`.
- `GET /api/v1/employee-shifts/current` uses `200 null` when no personal employee shift is open, so normal terminal startup does not produce a browser network `404` for this read.
- `GET /api/v1/cash-shifts/current` and `GET /api/v1/orders/current?table_id=...` may still appear as `404` in browser network console; this remains expected backend empty-state behavior for those two endpoints, not a visible UI error.
- `GET /api/v1/orders/active?hall_id=...` возвращает пустой массив, когда в зале нет активных заказов; экран зала больше не генерирует mock активных заказов или mock статусов столов.
- Cashier terminal shows "нет открытой личной смены", "нет открытой кассовой смены" or "нет активного заказа" instead of setting blocking `statusError`/`orderError` for these optional empty states.
- Optional current reads are not retried on expected empty states.
- Payment mutation has no automatic retry. On `409 CONFLICT` from `POST /api/v1/prechecks/{id}/payments`, UI shows the localized backend `message_key` when present, otherwise `errors.conflict`, and invalidates current cash session, current order, order, prechecks, check and closed orders.
- Payment buttons require an active precheck, positive amount, sufficient remaining total, payment permission and an open cash session. If a precheck exists but cash session is absent, UI blocks payment and shows the operator to open a cash session.

## Backend Capability Vs UI Capability

Refund:

- Backend ledger capability is implemented through `POST /api/v1/checks/{id}/cancellations` and `POST /api/v1/checks/{id}/refunds`.
- Cashier UI capability is implemented only through the compatibility route `POST /api/v1/payments/{id}/refund` for closed orders with captured payments.
- The compatibility route records a refund ledger operation and does not make UI authoritative for payment/check mutation.
- UI shows refund action only when `pos.payment.refund` is granted and current cash session exists.
- UI does not expose refund for active issued prechecks with partial captured payments; refund runtime requires a finalized check.
- Backend remains final enforcement layer.

Reprint:

- Backend precheck/check reprint is implemented from immutable snapshots.
- UI has reprint actions guarded by `pos.precheck.reprint` and `pos.check.reprint`.
- UI displays copy readiness through i18n text, not hardcoded source strings outside locale.
- Cancel/refund dialogs use safe operator wording through i18n and do not expose raw backend details, PIN, SQL or stack traces.

## Financial Boundaries

Реализовано сейчас:

- UI displays backend-provided order/precheck/check totals.
- UI sends payment amount and method to backend.
- UI does not calculate authoritative discount/tax/check totals.
- UI does not apply tax rules, discount rules or modifier prices as authoritative financial logic.
- UI validates modifier required/min/max constraints only as UX feedback before sending the backend command.

Не реализовано сейчас:

- discount/surcharge editor in cashier UI;
- tax profile editor in cashier UI;
- check-level full/partial cancellation UI;
- line/quantity/modifier/service-charge/tip refund UI;
- inventory consumption UI.

Запланировано далее:

- discount/manual override controls must exist only if backend policy/API exists;
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

## Разделение Интерфейсов

Реализовано сейчас:

- `/pos` and `/pos/cashier` load the current cashier pilot terminal.
- Код cashier terminal разделен на composable для runtime/API state и presentation components для POS shell, floor, menu grid, order rail, payment/actions modals и utility panels.
- Bottom quick access bar и скрываемое side menu являются основным navigation shell для POS runtime.
- Раздел `ЗАКАЗ` является основным redesigned рабочим экраном: category tabs + dish grid + current order panel + payment/actions modal поверх текущего заказа.
- Раздел `Залы / столы` является redesigned рабочим экраном выбора зала/стола: table grid + active orders panel + modal создания заказа на данных `GET /api/v1/orders/active?hall_id=...`.
- Раздел `Аналитика` использует текущую ограниченную операционную сводку: личная смена, кассовая смена, сводки закрытых заказов и оплат, sync health. Расширенные отчеты отмечены как `запланировано далее`, без активных неподдержанных кнопок.
- Раздел `Смена` использует текущую операционную секцию кассы: личная смена, кассовая смена, cash drawer actions, sync diagnostics, lock/logout.
- Разделы `Доставка` и `Настройки` пока используют существующие безопасные placeholder/utility surfaces до появления отдельных backend/API contracts.
- Основные route components загружаются через lazy imports/code splitting, чтобы снизить нагрузку на initial bundle.

## POS Shell Visual Contract

Реализовано сейчас:

- Основной POS экран рассчитан на 1366x768 без горизонтального скролла.
- Основные панели, кнопки, карточки блюд, карточки столов и строки заказа используют прямые углы.
- Side menu открывается по левой кнопке нижней панели, накладывается поверх интерфейса, не сдвигает layout и закрывается после выбора раздела.
- Замок блокировки POS находится в правом нижнем углу нижней панели.
- Правая панель заказа не показывает заголовки `Заказ #...` и `Стол ...`; контекст заказа вынесен в нижнюю панель.
- Правая панель `Залы / столы` показывает только активные заказы, сгруппированные по залу.

Запланировано далее:

- Подключить backend/API редактирования модификаторов уже добавленной строки; добавление строки с модификаторами и сохранение course/comment уже реализованы сейчас.
- Подключить backend/API быстрого чека со столом по умолчанию и проверкой отдельного permission.

Вне текущего объема:

- `/pos/waiter`, `/pos/kitchen` и `/pos/manager` являются только route shells. Они не реализуют waiter mobile, KDS или manager runtime без backend/API contracts.

## Вне Текущего Объема

Вне текущего cashier pilot UI:

- KDS runtime screens;
- delivery/channel screens;
- real PSP terminal integration UI;
- fiscal device operation UI;
- full inventory/procurement UI;
- rich cancellation/refund ledger UI beyond captured-payment compatibility action;
- discount/surcharge cashier editor and tax policy UI on top of existing backend pricing foundation.
