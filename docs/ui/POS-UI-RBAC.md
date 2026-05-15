# POS UI RBAC

Статус: синхронизировано с текущим cashier UI/backend permissions.

UI visibility is UX only. Backend app-layer permissions remain authoritative.

Интерфейс кассира теперь разделен на переиспользуемые компоненты в `pos-ui/src/pages/pos/*`, но все visibility guards по-прежнему используют backend permission ids из `pos-ui/src/shared/rbac.ts`.

## Реализовано Сейчас

Permission ids used by cashier UI:

- `pos.shift.open`
- `pos.shift.close`
- `pos.cash_session.open`
- `pos.cash_session.close`
- `pos.cash_drawer.record`
- `pos.floor.view`
- `pos.menu.view`
- `pos.catalog.view`
- `pos.order.create`
- `pos.order.view`
- `pos.order.line.add`
- `pos.order.line.update`
- `pos.order.line.void`
- `pos.order.close`
- `pos.precheck.issue`
- `pos.precheck.view`
- `pos.precheck.reprint`
- `pos.precheck.cancel.request`
- `pos.precheck.cancel`
- `pos.payment.cash`
- `pos.payment.card.manual`
- `pos.payment.other`
- `pos.payment.refund`
- `pos.check.view`
- `pos.check.reprint`
- `pos.sync.view`
- `pos.sync.retry`

## Cashier UI Actions

| UI action | Permission | Статус |
| --- | --- | --- |
| Open employee shift | `pos.shift.open` | реализовано сейчас |
| Close employee shift | `pos.shift.close` | реализовано сейчас |
| Open cash session | `pos.cash_session.open` | реализовано сейчас |
| Close cash session | `pos.cash_session.close` | реализовано сейчас |
| View floor/tables | `pos.floor.view` | реализовано сейчас |
| View menu/catalog | `pos.menu.view`, `pos.catalog.view` | реализовано сейчас |
| Create order | `pos.order.create` | реализовано сейчас |
| Add order line | `pos.order.line.add` | реализовано сейчас |
| Select modifiers while adding order line | `pos.order.line.add` | реализовано сейчас |
| Change line quantity | `pos.order.line.update` | реализовано сейчас |
| Void line | `pos.order.line.void` | реализовано сейчас |
| Issue precheck | `pos.precheck.issue` | реализовано сейчас |
| Cancel precheck request | `pos.precheck.cancel.request` | реализовано сейчас |
| Manager approve precheck cancel | `pos.precheck.cancel` | реализовано сейчас |
| Reprint precheck | `pos.precheck.reprint` | реализовано сейчас |
| Capture cash payment | `pos.payment.cash` | реализовано сейчас |
| Capture manual card payment | `pos.payment.card.manual` | реализовано сейчас |
| Refund captured payment through compatibility route | `pos.payment.refund` | реализовано сейчас |
| Check cancellation/refund ledger UI by line/quantity/scope | `pos.precheck.cancel`, `pos.payment.refund` | запланировано далее |
| View final check / closed orders | `pos.check.view` | реализовано сейчас |
| Reprint final check | `pos.check.reprint` | реализовано сейчас |

## Вне Текущего UI Объема

- waiter payment without cashier permissions;
- waiter mobile runtime;
- order transfer/split/merge;
- check-level cancellation/refund ledger screens beyond payment-level refund;
- discount/surcharge/tax override controls;
- inventory/procurement operations;
- KDS screens;
- manager tools runtime beyond cashier-visible sync/closed-orders/cash-drawer panels;
- PSP terminal/fiscal device operation screens.

## Notes

- Refund is not completely absent: backend ledger exists, while cashier UI currently exposes only payment-level compatibility refund for closed orders.
- Cancellation/refund policy still needs pilot acceptance for operator workflow and fiscal wording.
- UI must not show raw backend/internal errors or calculate authoritative financial totals.
