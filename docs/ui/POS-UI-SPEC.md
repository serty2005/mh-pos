# POS UI Spec

Статус: актуальный cashier UI contract для frozen pilot.

UI не является security boundary. Backend RBAC и application-layer checks остаются авторитетными.

## Реализовано Сейчас

Cashier UI in `pos-ui/src/pages/PosPage.vue` разделен на переиспользуемые компоненты терминала кассира в `pos-ui/src/pages/pos/*` и поддерживает:

- PIN login/session-based operator context;
- Square-like POS shell: постоянная нижняя quick access bar, скрываемое side menu разделов и рабочие разделы `Залы и столы`, `Заказы`, `Активность`, `Отчеты`, `Касса`;
- waiter-first layout выбранного заказа: слева menu grid примерно 3/4 ширины, справа current order/action rail примерно 1/4 ширины;
- modal-процессы для оплаты, действий заказа, manager override отмены пречека, cash drawer и refund;
- secondary operations (`cash drawer`, `closed orders`, `sync`) визуально отделены от основного пути продажи в отдельных разделах, dialog/drawer;
- employee shift open/close;
- cash session open/close;
- halls/tables selection;
- current order lookup by table;
- create order;
- add order line from menu;
- select modifiers for menu items with modifier groups;
- show selected modifiers under active order lines;
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
- Раздел `Заказы` является первым redesigned рабочим экраном: menu grid + active order rail + payment/actions modal поверх текущего заказа.
- Раздел `Активность` является полноценной секцией со списком и детальной панелью: закрытые заказы, компактные фильтры, детали оплат, reprint check и compatibility refund через уже реализованные backend-возможности.
- Раздел `Отчеты` является ограниченной операционной сводкой: личная смена, кассовая смена, сводки закрытых заказов и оплат, sync health. Расширенные отчеты отмечены как `запланировано далее`, без активных неподдержанных кнопок.
- Раздел `Касса` является полноценной операционной секцией: личная смена, кассовая смена, cash drawer actions, sync diagnostics, lock/logout.
- Основные route components загружаются через lazy imports/code splitting, чтобы снизить нагрузку на initial bundle.

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
