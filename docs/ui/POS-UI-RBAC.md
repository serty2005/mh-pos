# RBAC-матрица POS UI

## Назначение

Этот документ задает **матрицу прав сотрудников по UI-операциям**.

Он отвечает на вопрос:
какая роль может выполнить какое действие на UI
и где требуется manager override.

## Базовые принципы

- UI скрывает или дизейблит действия ради UX.
- Backend является финальным enforcement layer.
- Любая опасная операция должна быть описана здесь до появления в UI.
- Shared employee credential model запрещен.
- Каждому сотруднику соответствует отдельная учетная запись и отдельный audit trail.

## Роли

Поддерживаемые роли pilot baseline:

- `cashier`
- `senior_cashier`
- `waiter`
- `manager`
- `kitchen`
- `support_admin`

## Типы разрешений

Используются три класса решения:

- `allow` - операция возможна без дополнительного подтверждения;
- `override` - операция инициируется сотрудником, но требует manager override;
- `deny` - операция недоступна.

## Каталог permission id

### Session и terminal

- `ui.session.login`
- `ui.session.lock`
- `ui.system.pair`
- `ui.system.logout`

### Смены и касса

- `ui.pos.shift.open`
- `ui.pos.shift.close`
- `ui.pos.cash_session.open`
- `ui.pos.cash_session.close`
- `ui.pos.cash_drawer.no_sale`

### Order lifecycle

- `ui.pos.order.select_table`
- `ui.pos.order.create`
- `ui.pos.order.add_line`
- `ui.pos.order.change_quantity`
- `ui.pos.order.void_line`
- `ui.pos.order.transfer`
- `ui.pos.order.view_other_employee_order`

### Финансовые действия

- `ui.pos.precheck.issue`
- `ui.pos.precheck.cancel`
- `ui.pos.payment.cash`
- `ui.pos.payment.card.manual`
- `ui.pos.check.view`
- `ui.pos.check.reprint`
- `ui.pos.payment.refund`

### Manager и service operations

- `ui.manager.sync.view`
- `ui.manager.sync.retry_failed`
- `ui.manager.diagnostics.view`
- `ui.manager.diagnostics.actions`
- `ui.manager.catalog.edit`
- `ui.manager.floor.edit`
- `ui.manager.roles.edit`
- `ui.manager.employees.edit`

## Правила manager override

Операция через override допустима только если:

- actor имеет право инициировать запрос override;
- approving manager имеет право подтверждать эту операцию;
- reason обязателен;
- PIN manager обязателен;
- попытка записывается в immutable audit trail.

Для pilot baseline manager override обязателен как минимум для:

- `ui.pos.precheck.cancel`
- `ui.pos.order.transfer`
- `ui.pos.payment.refund`
- `ui.manager.sync.retry_failed` если ресторанская политика это требует
- `ui.pos.cash_drawer.no_sale` если политика ресторана требует контроль менеджера

## Матрица ролей к операциям

Обозначения:

- `A` = allow
- `O` = allow only via manager override
- `-` = deny

| Операция | cashier | senior_cashier | waiter | manager | kitchen | support_admin |
|---|---:|---:|---:|---:|---:|---:|
| Login to POS UI | A | A | A | A | A | A |
| Pair terminal | - | - | - | A | - | A |
| Open shift | A | A | - | A | - | - |
| Close shift | - | A | - | A | - | - |
| Open cash session | A | A | - | A | - | - |
| Close cash session | - | A | - | A | - | - |
| No sale / drawer open | - | O | - | A | - | - |
| Select hall/table | A | A | A | A | - | - |
| Create order | A | A | A | A | - | - |
| Add order line | A | A | A | A | - | - |
| Change quantity before precheck | A | A | A | A | - | - |
| Void line before precheck | A | A | A | A | - | - |
| View other employee order | - | A | - | A | - | - |
| Transfer order to another employee | - | O | O | A | - | - |
| Issue precheck | A | A | A | A | - | - |
| Cancel precheck | - | O | O | A | - | - |
| Take cash payment | A | A | O | A | - | - |
| Take trusted manual card payment | A | A | O | A | - | - |
| View final check | A | A | A | A | - | - |
| Reprint final check | - | A | - | A | - | - |
| Refund payment | - | - | - | A | - | - |
| View sync status | - | A | - | A | - | A |
| Retry failed syncs | - | - | - | A | - | A |
| View diagnostics | - | - | - | A | - | A |
| Запустить diagnostics actions | - | - | - | A | - | A |
| Edit halls/tables | - | - | - | A | - | A |
| Edit menu/catalog | - | - | - | A | - | A |
| Edit employees/roles | - | - | - | A | - | A |

## Рекомендуемые связки роли и режима

### Cashier

Видит:

- `pair` только если это отдельная pilot policy не нужна;
- `login`
- `pos`
- `lock`

Не видит:

- manager UI
- diagnostics
- catalog/settings/admin screens

### Waiter

Целевая роль для будущего waiter mode.

В cashier pilot runtime waiter не должен получать кассовые и сервисные операции по умолчанию.

### Manager

Может:

- подтверждать override;
- выполнять опасные операции напрямую;
- видеть sync/diagnostics;
- управлять людьми и настройками.

### Support admin

Это не ресторанная операционная роль.
Ее задача - pairing, service access, diagnostics, техническое обслуживание.
Она не должна использоваться как обычная кассовая роль.

## UX-требования к RBAC

UI обязан:

- скрывать операции, которые роль не может выполнить вообще;
- показывать операции override как требующие manager approval;
- не показывать “тихие” fallback paths;
- не подменять backend decision локальной логикой.

## Правила эволюции матрицы

Нельзя добавлять новую UI-операцию без:

- permission id;
- строки в этой матрице;
- backend enforcement note;
- указания, нужна ли manager override;
- теста или acceptance note для UX visibility.

## Backend sync status

implemented now:

- backend enforces a canonical RBAC slice for cashier runtime operations:
  - `pos.shift.open`, `pos.shift.close`
  - `pos.cash_session.open`, `pos.cash_session.close`
  - `pos.cash_drawer.record_event`
  - `pos.order.create`, `pos.order.add_line`, `pos.order.change_quantity`, `pos.order.void_line`
  - `pos.precheck.issue`
  - `pos.payment.capture`
  - `pos.sync.retry_failed` for operator-triggered retry API
- manager override approver validation for precheck cancel uses `pos.precheck.cancel`.

planned next:

- complete backend enforcement for the full matrix in this document (including waiter/senior_cashier override variants and non-cashier surfaces).

out of scope:

- treating UI visibility as a security boundary without backend authorization checks.
