# RBAC-матрица POS UI

## Назначение

Документ фиксирует фактическую permission model для текущего POS UI и backend enforcement.

Код и тесты остаются источником истины для `implemented now`. UI скрывает или блокирует действия только ради UX; финальная проверка всегда выполняется backend app-layer.

## Роли

implemented now:

- `cashier`
- `senior_cashier`
- `waiter`
- `manager`
- `kitchen`
- `support_admin`

Роли закреплены в backend role profiles и возвращаются пользователю через auth/session permissions. Permissions хранятся в `roles.permissions_json`, но валидируются через canonical backend catalog.

## Canonical Permission Catalog

implemented now:

- `pos.employee_shift.open`
- `pos.employee_shift.close`
- `pos.employee_shift.view_current`
- `pos.employee_shift.recent`
- `pos.cash_session.open`
- `pos.cash_session.close`
- `pos.cash_session.view_current`
- `pos.cash_drawer.record_event`
- `pos.catalog.view`
- `pos.floor.view`
- `pos.menu.view`
- `pos.order.create`
- `pos.order.view`
- `pos.order.add_line`
- `pos.order.change_quantity`
- `pos.order.void_line`
- `pos.order.close`
- `pos.precheck.issue`
- `pos.precheck.view`
- `pos.precheck.cancel.request`
- `pos.precheck.cancel`
- `pos.payment.cash`
- `pos.payment.card.manual`
- `pos.payment.other`
- `pos.check.view`
- `pos.sync.view`
- `pos.sync.retry_failed`

out of scope:

- UI-only permission ids вида `ui.*`.
- Permission ids для несуществующих runtime endpoints.

## Manager Override

implemented now:

- `CancelPrecheck` использует split permissions:
  - actor должен иметь `pos.precheck.cancel.request`;
  - approving manager должен иметь `pos.precheck.cancel`;
  - reason и manager PIN обязательны;
  - попытка пишет audit trail.

out of scope:

- override для `order transfer`, `refund`, waiter payment и `cash drawer no sale`;
- restaurant-level policy engine для включения/выключения override per operation.

## Implemented Runtime Matrix

Обозначения:

- `A` = allow
- `O` = allow through implemented manager override
- `-` = deny

| Операция | permission id | cashier | senior_cashier | waiter | manager | kitchen | support_admin |
|---|---|---:|---:|---:|---:|---:|---:|
| Login / active session | session flow | A | A | A | A | A | A |
| Lock / logout | session flow | A | A | A | A | A | A |
| Open personal employee shift | `pos.employee_shift.open` | A | A | A | A | - | - |
| Close personal employee shift | `pos.employee_shift.close` | A | A | A | A | - | - |
| View current personal shift | `pos.employee_shift.view_current` | A | A | A | A | - | - |
| View recent personal shifts | `pos.employee_shift.recent` | A | A | A | A | - | - |
| Open cash shift | `pos.cash_session.open` | A | A | - | A | - | - |
| Close cash shift | `pos.cash_session.close` | - | A | - | A | - | - |
| View current cash shift | `pos.cash_session.view_current` | A | A | - | A | - | - |
| Cash drawer event / no sale | `pos.cash_drawer.record_event` | - | - | - | A | - | - |
| View catalog reference | `pos.catalog.view` | A | A | A | A | - | - |
| Select hall/table | `pos.floor.view` | A | A | A | A | - | - |
| View menu | `pos.menu.view` | A | A | A | A | - | - |
| Create order | `pos.order.create` | A | A | A | A | - | - |
| View order | `pos.order.view` | A | A | A | A | - | - |
| Add order line | `pos.order.add_line` | A | A | A | A | - | - |
| Change quantity before precheck | `pos.order.change_quantity` | A | A | A | A | - | - |
| Void line before precheck | `pos.order.void_line` | A | A | A | A | - | - |
| Close order after final check | `pos.order.close` | A | A | A | A | - | - |
| Issue precheck | `pos.precheck.issue` | A | A | A | A | - | - |
| View precheck | `pos.precheck.view` | A | A | A | A | - | - |
| Cancel precheck | `pos.precheck.cancel.request` + `pos.precheck.cancel` | - | O | - | A | - | - |
| Take cash payment | `pos.payment.cash` | A | A | - | A | - | - |
| Take trusted manual card payment | `pos.payment.card.manual` | A | A | - | A | - | - |
| Other payment method | `pos.payment.other` | - | - | - | A | - | - |
| View final check | `pos.check.view` | A | A | A | A | - | - |
| View sync status/local events/outbox | `pos.sync.view` | - | A | - | A | - | A |
| Retry failed syncs | `pos.sync.retry_failed` | - | - | - | A | - | A |

## Out Of Scope Runtime Rows

out of scope:

- role-based terminal pairing UI;
- view other employee order;
- transfer order to another employee;
- waiter payment override;
- reprint final check;
- refund payment;
- diagnostics screens/actions;
- manager/admin screens for editing halls, tables, menu, catalog, employees and roles.

Эти строки не считаются `implemented now`, пока в коде нет соответствующего route/use-case, permission id, backend enforcement и тестов.

## UX Requirements

implemented now:

- `pos-ui` использует backend permission ids напрямую в `src/shared/rbac.ts`;
- критичные POS-действия в `/pos` скрываются или блокируются по текущему session actor permissions;
- query-запросы к защищенным backend read endpoints не запускаются без соответствующего permission, чтобы не плодить ожидаемые `403` в браузерных devtools.

out of scope:

- считать UI visibility security boundary.

## Evolution Rules

Нельзя добавлять новую UI-операцию без:

- canonical backend permission id;
- строки в этой матрице;
- backend enforcement note;
- теста backend allow/deny;
- UI visibility test или acceptance note;
- документационного статуса `implemented now`, `planned next` или `out of scope`.
