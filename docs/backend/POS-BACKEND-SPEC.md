# Спецификация POS Backend

## Назначение

Этот документ фиксирует:

- текущий публичный backend surface;
- state transitions;
- policy compatibility-хвостов для текущего публичного API;
- event catalog Edge runtime;
- границы между implemented now, planned next и out of scope.

## Архитектурная позиция

Edge backend является source of truth для всех активных POS-операций.

Cloud не является runtime dependency для:

- смен;
- кассовых смен;
- заказов;
- пречеков;
- оплат;
- финальных чеков;
- manager override.

## Финансовая модель

Каноническая модель:

```text
Order -> Precheck -> Payment -> Check
```

Правила:

- `Order` - рабочая сущность обслуживания;
- `Precheck` - рабочий финансовый snapshot;
- `Payment` - immutable финансовый факт;
- `Check` - финальный расчетный документ после полной оплаты precheck.

## Текущий публичный API

### Health и system

- `GET /health`
- `GET /api/v1/system/pairing-status`
- `POST /api/v1/system/pair`

implemented now: `POST /api/v1/system/pair` сохраняет verifier pairing code в keyed format `pairing.hmac-sha256.v1`; plaintext pairing code не сохраняется.

### Auth

- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`

implemented now: PIN login is rate-limited per `node_device_id + client_device_id`.
implemented now: repeated invalid PIN attempts return `429 Too Many Requests`.
implemented now: PIN values are never echoed back in response payloads.
implemented now: PIN login must resolve exactly one active employee in the paired restaurant; duplicate active PIN matches return conflict instead of choosing an arbitrary employee.

### Залы и меню

- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/catalog/items`
- `GET /api/v1/menu/items`

implemented now: halls, tables, catalog and menu are Cloud-owned master data. Public Edge runtime writes to these entities return `403 Forbidden`; local/demo setup uses `POST /api/v1/dev/bootstrap-demo`.

### Смены и касса

- `GET /api/v1/employee-shifts/current`
- `GET /api/v1/employee-shifts/recent`
- `POST /api/v1/employee-shifts/open`
- `POST /api/v1/employee-shifts/{id}/close`

- `GET /api/v1/cash-shifts/current`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `POST /api/v1/cash-drawer-events`

implemented now:

- `auth_sessions` остаются техническим login/logout-контекстом устройства и клиента.
- `shifts` используются как личные смены сотрудника: открытая смена ищется по `restaurant_id + employee_id`, а не по устройству.
- Без открытой личной смены сотрудника business/runtime операции запрещены; доступны только открытие личной смены и чтение последних личных смен текущего actor.
- Заказы, позиции и пречеки требуют открытую личную смену, но не требуют открытую кассовую смену.
- Оплаты и cash drawer events требуют открытую кассовую смену на устройстве.
- `cash_sessions` являются текущей runtime-сущностью кассовой смены.

planned next:

- Данные личной смены сотрудника будут использоваться для учета рабочего времени post-MVP.

### Заказы

- `GET /api/v1/orders/current?table_id=...`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `POST /api/v1/orders/{id}/close`

### Пречеки и чеки

- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/prechecks/{id}`
- `GET /api/v1/orders/{id}/prechecks`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/checks/{id}`

### Operational sync endpoints

- `GET /api/v1/sync/outbox`
- `GET /api/v1/sync/status`
- `GET /api/v1/sync/local-events`
- `POST /api/v1/sync/retry-failed`

implemented now: operator-facing sync endpoints enforce app-layer RBAC:

- `GET /api/v1/sync/outbox` requires `pos.sync.view`;
- `GET /api/v1/sync/status` requires `pos.sync.view`;
- `GET /api/v1/sync/local-events` requires `pos.sync.view`;
- `POST /api/v1/sync/retry-failed` requires `pos.sync.retry_failed`.

### Cloud -> Edge master-data ingest endpoints

implemented now:

- `POST /api/v1/sync/master-data/snapshots`
- `POST /api/v1/sync/master-data/{stream}`

Supported streams: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.

Payload accepts `node_device_id`, optional `restaurant_id`, `sync_mode` (`full_snapshot` or `incremental`), optional `checkpoint_token`, `cloud_version`, optional `cloud_updated_at`, and stream arrays: `restaurants`, `devices`, `roles`, `employees`, `halls`, `tables`, `catalog_items`, `menu_items`.

implemented now: these endpoints are Cloud -> Edge ingest, not POS runtime mutation APIs. Handler sets origin `cloud_sync`, calls app-layer master sync use case, writes master rows and `cloud_master_sync_state` in one transaction, and does not create `local_event_log` or `pos_sync_outbox` rows.

### Dev/local bootstrap

implemented now:

- `POST /api/v1/dev/bootstrap-demo`
- доступен только при `POS_DEV_TOOLS=1`;
- создает demo restaurant, paired Edge Node, cashier/manager roles, сотрудников с PIN `1111`/`2222`, hall/table и menu items;
- возвращает `pairing_code` и `manager_employee_id` для ручного POS UI smoke flow;
- не является production path.

### Master-data mutation boundary

implemented now:

- `POST /api/v1/restaurants`
- `POST /api/v1/devices/register`
- `POST /api/v1/roles`
- `POST /api/v1/employees`
- `PATCH /api/v1/employees/{id}/archive`
- `POST /api/v1/halls`
- `PATCH /api/v1/halls/{id}/archive`
- `POST /api/v1/tables`
- `PATCH /api/v1/tables/{id}/archive`
- `POST /api/v1/catalog/items`
- `POST /api/v1/menu/items`

Эти routes больше не являются runtime-supported Edge mutation flow. implemented now: HTTP layer держит их как dev-only seed/admin helpers за `POS_DEV_TOOLS=1`; без dev tools они возвращают `403 Forbidden`. В dev mode handler использует origin `system_seed`. Production Cloud-authored master data должна входить через `POST /api/v1/sync/master-data/snapshots` или `POST /api/v1/sync/master-data/{stream}` с origin `cloud_sync`.

## Policy compatibility-хвостов

implemented now: публичные compatibility tails удалены из backend API surface.

`device_id` остается domain/storage field для POS Edge node identity в operational payloads. Новые transport examples используют явные `node_device_id` и `client_device_id`, когда нужен actor/device context.

## Переходы состояния

### Order

- `open` -> `locked` при `IssuePrecheck`;
- `locked` -> `open` при `CancelPrecheck`;
- `open` или `locked` -> `closed` только после полной оплаты и final check;
- `closed` не редактируется.

### Precheck

- `issued` -> `cancelled`;
- `issued` -> `closed` при полной оплате;
- `issued` -> `superseded` зарезервировано для future re-issue flow.

### Payment

- создается как immutable факт;
- не редактируется;
- не удаляется;
- correction делается отдельными финансовыми операциями, а не mutate-in-place.

### Check

- создается только после полной оплаты precheck;
- не является рабочим счетом гостя;
- не создается вручную в нормальном runtime flow.

## Каталог событий Edge runtime

На Edge runtime уже существуют или должны существовать в outbox/local event log события следующих групп:

### System и auth

- `EdgeNodePaired`
- `AuthSessionStarted`
- `AuthSessionRevoked`
- `DeviceRegistered`

### Смены и касса

- `ShiftOpened`
- `ShiftClosed`
- `CashSessionOpened`
- `CashSessionClosed`
- `CashDrawerEventRecorded`

### Заказы

- `OrderCreated`
- `OrderLineAdded`
- `OrderLineQuantityChanged`
- `OrderLineVoided`
- `OrderClosed`

### Финансы

- `PrecheckIssued`
- `PrecheckCancelled`
- `PaymentCaptured`
- `CheckCreated`

## Примечание к sync contract

Документация Cloud receiver и Edge event emission должны быть синхронизированы.

implemented now: production sender path отправляет только Edge -> Cloud operational events. Cloud-managed/configuration events, например изменения restaurant, employee, role, catalog, menu, hall и table, не отправляются sender-ом вверх; они помечаются `suspended` с явной sync-direction причиной.

implemented now: Cloud принимает operational sender catalog, описанный в `docs/sync/edge-cloud-contracts-v1.md`, и хранит raw envelopes плюс `cloud_operational_events`. Ownership matrix и directional sync rules описаны в `docs/sync/directional-sync-ownership.md`.

implemented now: Cloud -> Edge provisioning/configuration имеет backend apply flow: `internal/pos/app/mastersync` принимает `cloud_sync`, master tables имеют sync metadata, `cloud_master_sync_state` хранит stream checkpoints, а dedicated sync endpoints применяют full snapshot/incremental payloads для supported streams. Full snapshot replacement policy beyond upserted payload rows является planned next.

## Manager override

На текущем этапе manager override обязателен как минимум для отмены пречека.

Минимальный контракт payload для override-операции:

- actor context;
- manager employee id;
- manager PIN;
- reason.

Backend обязан:

- проверить active session actor;
- проверить actor permission `pos.precheck.cancel.request` для инициации override-операции;
- проверить manager employee и manager permission;
- проверить manager PIN;
- записать audit trail;
- записать sync/local events транзакционно.

## RBAC enforcement (implemented now)

implemented now:

- backend uses a canonical permission catalog in app-layer checks;
- role permissions remain stored as JSON, but enforcement uses stable permission ids;
- key cashier runtime operations are enforced in app services via `EnsureOperatorSession(...requiredPermissions...)`.

Canonical permission ids used by implemented now runtime:

- `pos.shift.open`
- `pos.shift.close`
- `pos.shift.view_current`
- `pos.shift.recent`
- `pos.cash_session.open`
- `pos.cash_session.close`
- `pos.cash_session.view_current`
- `pos.cash_drawer.record_event`
- `pos.floor.view`
- `pos.menu.view`
- `pos.order.create`
- `pos.order.view`
- `pos.order.add_line`
- `pos.order.change_quantity`
- `pos.order.void_line`
- `pos.precheck.issue`
- `pos.precheck.view`
- `pos.precheck.cancel.request` (override actor permission)
- `pos.precheck.cancel` (manager override approver permission)
- `pos.payment.capture`
- `pos.check.view`
- `pos.sync.view` (required for operator-triggered `GET /api/v1/sync/outbox`, `GET /api/v1/sync/status`, `GET /api/v1/sync/local-events`)
- `pos.sync.retry_failed` (required for operator-triggered `POST /api/v1/sync/retry-failed`)

Error behavior:

- missing permission returns domain `forbidden` and HTTP `403`;
- authorization errors do not include sensitive auth fields (PIN, manager PIN, PIN hash).

planned next:

- extend canonical backend enforcement to the full UI RBAC matrix (`docs/ui/POS-UI-RBAC.md`) beyond the current runtime RBAC slice.

## Currency policy (implemented now)

implemented now:

- backend validates runtime currency codes against a canonical pilot ISO 4217 profile catalog;
- pilot catalog explicitly supports both 2-decimal and 3-decimal currencies;
- pricing/payment domain amounts continue to use integer minor units (no floating-point storage);
- unsupported currency code is rejected as domain `invalid`.

## Документационные правила

Любое изменение одного из пунктов ниже обновляет этот файл в том же PR:

- список endpoints;
- контракт запроса/ответа;
- переходы состояния;
- event catalog;
- compatibility tails;
- manager override behavior.

Если меняется только долгосрочная архитектурная цель, но не runtime contract, обновляется `SPECv1.3.md`, а не этот файл.

## Operational logging (implemented now)

- Backend writes structured operation logs with levels `TRACE|DEBUG|INFO|WARN|ERROR`.
- Request audit logs include `request_id`, `operation`, `action`, `result`, `duration_ms`, `error_code` and masked actor/device/session identifiers.
- Sensitive auth fields (`pin`, `manager_pin`, pin hash, raw auth payload) must not be logged.

## Sync sender telemetry (implemented now)

- `internal/pos/syncsender` emits normalized worker telemetry for non-HTTP paths with fields `operation`, `action`, `result`, `error_code` and masked correlation ids.
- TRACE-level lifecycle events are emitted for reclaim, batch claim, per-message processing, send attempt, ack and retry decision steps.
