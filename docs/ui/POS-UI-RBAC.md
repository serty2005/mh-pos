# POS UI RBAC

Статус: синхронизировано с текущим cashier UI/backend permissions.

UI visibility is UX only. Backend app-layer permissions remain authoritative.

Интерфейс кассира теперь разделен на переиспользуемые компоненты POS shell в `pos-ui/src/pages/pos/*`, но все visibility guards по-прежнему используют backend permission ids из `pos-ui/src/shared/rbac.ts`.

Нижняя quick access bar, скрываемое меню разделов, redesigned `Залы и столы`, `Заказы`, `Активность`, `Отчеты` и `Касса` являются UX-навигацией. Они не добавляют новых backend permission ids и не заменяют backend application-layer checks.

`Активность` показывает paginated/filtered closed orders, детали оплат, reprint, whole-check и partial `order_line`/quantity cancellation/refund и compatibility refund только по существующим правам `pos.check.view`, `pos.check.reprint`, `pos.precheck.cancel`, `pos.payment.refund` и состоянию открытой кассовой смены. `Отчеты` показывает только ограниченные операционные сводки на основе уже доступных reads и не вводит отдельные report permissions до backend-контракта.

## Реализовано Сейчас

Permission ids used by cashier UI:

- `pos.employee_shift.open`
- `pos.employee_shift.close`
- `pos.employee_shift.view_current`
- `pos.employee_shift.recent`
- `pos.cash_session.open`
- `pos.cash_session.close`
- `pos.cash_session.view_current`
- `pos.cash_drawer.record_event`
- `pos.floor.view`
- `pos.menu.view`
- `pos.catalog.view`
- `pos.order.create`
- `pos.order.view`
- `pos.order.add_line`
- `pos.order.change_quantity`
- `pos.order.void_line`
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
- `pos.sync.retry_failed`

## Cashier UI Actions

| UI action | Permission | Статус |
| --- | --- | --- |
| Open employee shift | `pos.employee_shift.open` | реализовано сейчас |
| Close employee shift | `pos.employee_shift.close` | реализовано сейчас |
| Open cash session | `pos.cash_session.open` | реализовано сейчас |
| Close cash session | `pos.cash_session.close` | реализовано сейчас |
| View floor/tables | `pos.floor.view` | реализовано сейчас |
| View menu/catalog | `pos.menu.view`, `pos.catalog.view` | реализовано сейчас |
| Create order | `pos.order.create` | реализовано сейчас |
| Add order line | `pos.order.add_line` | реализовано сейчас |
| Select modifiers while adding order line | `pos.order.add_line` | реализовано сейчас |
| Edit modifiers on active open order line | `pos.order.change_quantity` | реализовано сейчас |
| Change line quantity | `pos.order.change_quantity` | реализовано сейчас |
| Void line | `pos.order.void_line` | реализовано сейчас |
| Issue precheck | `pos.precheck.issue` | реализовано сейчас |
| Cancel precheck request | `pos.precheck.cancel.request` | реализовано сейчас |
| Manager approve precheck cancel | `pos.precheck.cancel` | реализовано сейчас |
| Reprint precheck | `pos.precheck.reprint` | реализовано сейчас |
| Capture cash payment | `pos.payment.cash` | реализовано сейчас |
| Capture manual card payment | `pos.payment.card.manual` | реализовано сейчас |
| Full check cancellation through ledger route | `pos.precheck.cancel` | реализовано сейчас |
| Full check refund through ledger route | `pos.payment.refund` | реализовано сейчас |
| Refund captured payment through compatibility route | `pos.payment.refund` | реализовано сейчас |
| Check cancellation/refund ledger UI by `order_line`/quantity scope | `pos.precheck.cancel`, `pos.payment.refund` | реализовано сейчас |
| View final check / closed orders | `pos.check.view` | реализовано сейчас |
| Reprint final check | `pos.check.reprint` | реализовано сейчас |

Pagination/filter controls закрытых заказов не вводят новые permission ids; backend `pos.check.view` остается authoritative для read.

## Вне Текущего UI Объема

- waiter payment without cashier permissions;
- waiter mobile runtime;
- order transfer/split/merge;
- partial modifier/service/tip cancellation/refund ledger screens beyond current order-line/quantity actions;
- discount/surcharge/tax override controls;
- inventory/procurement operations;
- KDS screens;
- manager tools runtime beyond cashier-visible sync/closed-orders/cash-drawer panels;
- PSP terminal/fiscal device operation screens.

## Notes

- Refund/cancellation больше не является compatibility-only сценарием в cashier UI: closed-order activity показывает whole-check и partial `order_line`/quantity cancellation/refund через backend ledger endpoints с явным выбором inventory disposition.
- The compatibility payment refund button remains visible only for closed orders with captured payments and is disabled without `pos.payment.refund` or current open cash session.
- Cancellation/refund policy still needs pilot acceptance for operator workflow and fiscal wording.
- UI must not show raw backend/internal errors or calculate authoritative financial totals.
