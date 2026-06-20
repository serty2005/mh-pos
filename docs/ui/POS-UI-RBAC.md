# POS UI RBAC

Статус: синхронизировано с текущим `pos-backend`, активным POS UI `pos-ui-g` и активным Cloud UI `cloud-ui-g`.

UI visibility является только UX-слоем. Backend application-layer RBAC остается авторитетным security boundary. Legacy `pos-ui` не является целевым UI для новых правок и в этой матрице не учитывается.

## Реализовано Сейчас

`pos-ui-g` получает permissions из POS Edge auth/session actor context и использует их для UX visibility/disabled states. Все финансовые, складские, KDS, sync и storage действия повторно проверяются backend service layer.

`cloud-ui-g` управляет справочниками, сотрудниками и POS role permission profiles через Cloud Backend master-data routes. Это не production Cloud operator RBAC perimeter: Cloud auth/RBAC для самих операторов Cloud UI остается вне текущего runtime.

## Backend Permission Catalog

Фактически используемые POS Edge permissions:

- Auth/session: PIN login возвращает actor permissions; отдельного permission на сам login нет.
- Employee shifts: `pos.employee_shift.open`, `pos.employee_shift.close`, `pos.employee_shift.view_current`, `pos.employee_shift.recent`.
- Cash shifts/cash drawer: `pos.cash_session.open`, `pos.cash_session.close`, `pos.cash_session.view_current`, `pos.cash_drawer.record_event`.
- Floor/menu/catalog reads: `pos.floor.view`, `pos.menu.view`, `pos.catalog.view`.
- Order/precheck/payment/check: `pos.order.create`, `pos.order.view`, `pos.order.add_line`, `pos.order.change_quantity`, `pos.order.void_line`, `pos.order.close`, `pos.precheck.issue`, `pos.precheck.view`, `pos.precheck.reprint`, `pos.precheck.cancel.request`, `pos.precheck.cancel`, `pos.payment.cash`, `pos.payment.card.manual`, `pos.payment.other`, `pos.payment.refund`, `pos.check.view`, `pos.check.reprint`.
- Pricing: `pos.pricing.view`, `pos.pricing.discount.apply`, `pos.pricing.surcharge.apply`; `requires_permission` у policy дополнительно проверяется backend.
- Kitchen/KDS: `pos.kitchen.view`, `pos.kitchen.status.change`.
- Kitchen stock input: `pos.kitchen.stock.receipt`, `pos.kitchen.stock.inventory_count`, `pos.kitchen.stock.write_off`, `pos.kitchen.production.complete`.
- Kitchen proposals: `pos.kitchen.catalog.view`, `pos.kitchen.recipe.view`, `pos.kitchen.recipe.suggest`, `pos.kitchen.catalog.suggest`.
- Stop-list: `pos.kitchen.stop_list.view`, `pos.kitchen.stop_list.update`.
- Sync/storage/support operations: `pos.sync.view`, `pos.sync.retry_failed`; storage destructive operations остаются support/RBAC-sensitive backend routes и не дают cashier/waiter/kitchen финансовых полномочий.

## Role Matrix

| Роль | Backend permissions | Доступные UI sections/actions | Явно запрещено по умолчанию | Gaps / спорные места | Статус |
| --- | --- | --- | --- | --- | --- |
| `cashier` | employee shift open/close/view/recent; cash session open/view; floor/menu/catalog; order create/view/add/change/void/close; pricing view/discount/surcharge apply; precheck issue/view/reprint; payment cash/card; check view | `pos-ui-g` POS mode: floor, order, activity, reports, cash; cash/card payment; issue/reprint precheck; policy discount/surcharge controls only when backend exposes matching permissions | refund, check reprint, cash drawer event, cash session close, payment other, sync retry, kitchen/KDS stock/proposals/stop-list | `senior_cashier` существует в backend как расширенный профиль, но не является отдельной целевой ролью этой матрицы | реализовано сейчас |
| `waiter` | employee shift open/close/view/recent; floor/menu/catalog; order create/view/add/change/void/close; pricing view; precheck issue/view/reprint; check view | `pos-ui-g` POS mode can show order/precheck workflow without active payment/refund/cash drawer controls; waiter handoff mode показывает QR/handoff shell | payment cash/card/other, refund, cash drawer, cash session, check reprint, sync retry, kitchen stock/proposals/stop-list | Полноценный mobile waiter route из старого `pos-ui` не считается целевым runtime; для `pos-ui-g` текущий waiter surface ограничен handoff/permission-safe POS shell | реализовано сейчас |
| `kitchen` | employee shift view_current in canonical backend profile; catalog view; kitchen view/status; kitchen catalog/recipe read; recipe/catalog suggestions; stock receipt/count/write-off/production; stop-list view/update | `pos-ui-g` KDS mode: order queue, status actions, stock forms, recipe view/suggestions, catalog suggestions, my proposals, stop-list view/update | payment/refund/check/cash drawer/cash session/sync retry/cashier controls | Dev/system seed выдает kitchen также employee shift open/close/recent для smoke/runtime удобства; это не добавляет финансовых полномочий | реализовано сейчас |
| `manager` | employee shift all; cash session open/close/view; cash drawer; floor/menu/catalog; order all; pricing all; precheck issue/view/reprint/cancel request/cancel; payment cash/card/other/refund; check view/reprint; sync view/retry | `pos-ui-g` POS mode: cashier flow плюс refund/cancel/reprint/cash drawer/sync retry controls; `cloud-ui-g` staff/permissions/backoffice sections are route-backed management UI, not Cloud operator RBAC | unsupported business functions: PSP/fiscal device controls, order split/merge/transfer, unsupported OLAP mutating controls in active `cloud-ui-g` | Manager Cloud UI controls do not imply production Cloud auth/RBAC perimeter | реализовано сейчас |
| `support` (`support_admin` в backend) | `pos.sync.view`, `pos.sync.retry_failed` | `pos-ui-g` sync/status diagnostics and retry where backend allows; support-only backend operations remain explicitly separated from cashier/waiter/kitchen | order/payment/refund/check/cash drawer/cash session/kitchen controls | User-facing role name `support` maps to current backend role id `support_admin`; active `cloud-ui-g` does not implement production support login/RBAC | реализовано сейчас |

## UI Visibility In `pos-ui-g`

Реализовано сейчас:

- Payment button is active only with at least one of `pos.payment.cash`, `pos.payment.card.manual`, `pos.payment.other`.
- Refund button is active only with `pos.payment.refund` and an open cash session.
- Final check reprint is active only with `pos.check.reprint`.
- Precheck issue/cancel request buttons are active only with `pos.precheck.issue` / `pos.precheck.cancel.request`.
- Pricing policy dialog shows only policies allowed by `pos.pricing.discount.apply`, `pos.pricing.surcharge.apply` and optional policy `requires_permission`.
- Cash drawer event, cash session close/open and sync retry controls are gated by their exact permissions.
- KDS/stock/proposal/stop-list surfaces are gated by the corresponding kitchen permission IDs.

Запланировано далее:

- Дополнительные component/e2e проверки для role-specific visibility в `pos-ui-g`.
- Уточнение UX для read-only manager/support diagnostics, если появятся новые backend support routes.

Вне текущего объема:

- Использовать frontend visibility как security boundary.
- Добавлять cashier/waiter/kitchen финансовые полномочия через UI без backend permission.
- Реализовывать order transfer/split/merge, PSP/fiscal device screens, hardware bump-bar, rich KDS analytics.
- Реанимировать legacy `pos-ui` как целевой runtime.

## Cloud UI Boundary

Реализовано сейчас:

- `cloud-ui-g` имеет route-backed sections: dashboard, restaurants, edge-sync, catalog, menu, modifiers, pricing-taxes, staff-permissions, floor, publications.
- `staff-permissions` редактирует tenant-level POS Edge roles/employees, authoritative restaurant memberships и permission profiles; `organization.manage` охватывает все restaurants, permission IDs и last-membership invariant валидирует Cloud Backend.
- `inventory` и `reports` в активном `cloud-ui-g` остаются blocked/planned placeholders и не являются активными runtime actions.

Запланировано далее:

- Production Cloud operator auth/RBAC perimeter для Cloud UI и mutating operator controls.
- Перенос подтвержденных manager/review/reporting surfaces из legacy `cloud-ui` в `cloud-ui-g` только после сверки backend routes/DTO.

Вне текущего объема:

- Считать Cloud UI staff-permissions экран Cloud operator authorization boundary.
- Включать unsupported inventory/reporting actions как активные runtime controls без backend route/permission boundary.
