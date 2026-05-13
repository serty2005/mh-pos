# POS UI Spec

Статус: актуальный cashier UI contract для frozen pilot.

UI не является security boundary. Backend RBAC и application-layer checks остаются авторитетными.

## Реализовано Сейчас

Cashier UI in `pos-ui/src/pages/PosPage.vue` supports:

- PIN login/session-based operator context;
- employee shift open/close;
- cash session open/close;
- halls/tables selection;
- current order lookup by table;
- create order;
- add order line from menu;
- change quantity;
- void line;
- issue precheck;
- cancel precheck through manager override dialog;
- reprint precheck copy;
- cash payment;
- trusted manual card payment;
- final check display after full payment;
- reprint final check copy;
- closed orders list;
- refund captured payment from closed orders when operator has permission and cash session is open.

UI calls backend APIs for authoritative state and does not compute authoritative totals.

## Backend Capability Vs UI Capability

Refund:

- Backend capability is implemented through `POST /api/v1/payments/{id}/refund`.
- Cashier UI capability is also implemented for closed orders with captured payments.
- UI shows refund action only when `pos.payment.refund` is granted and current cash session exists.
- Backend remains final enforcement layer.

Reprint:

- Backend precheck/check reprint is implemented from immutable snapshots.
- UI has reprint actions guarded by `pos.precheck.reprint` and `pos.check.reprint`.
- UI displays copy readiness through i18n text, not hardcoded source strings outside locale.

## Financial Boundaries

Реализовано сейчас:

- UI displays backend-provided order/precheck/check totals.
- UI sends payment amount and method to backend.
- UI does not calculate authoritative discount/tax/check totals.
- UI does not apply tax rules, discount rules or modifier price deltas.

Не реализовано сейчас:

- discount/surcharge editor in cashier UI;
- tax profile editor in cashier UI;
- modifier selection in cashier order flow;
- inventory consumption UI.

Запланировано до пилота, if accepted:

- modifier selection must submit a command to backend and render backend-calculated totals;
- discount/manual override controls must exist only if backend policy/API exists;
- no UI-side authoritative financial calculation.

## RBAC Visibility

Реализовано сейчас:

- UI maps backend permission ids in `pos-ui/src/shared/rbac.ts`.
- Critical actions are hidden/disabled based on permissions for UX.
- Backend validates permissions again.

Relevant permissions:

- `pos.order.create`
- `pos.order.line.add`
- `pos.order.line.update`
- `pos.order.line.void`
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

## Вне Текущего Объема

Вне текущего cashier pilot UI:

- KDS runtime screens;
- delivery/channel screens;
- real PSP terminal integration UI;
- fiscal device operation UI;
- full inventory/procurement UI;
- modifiers UI until backend order runtime exists;
- discount/surcharge/tax policy UI until backend pricing policy exists.
