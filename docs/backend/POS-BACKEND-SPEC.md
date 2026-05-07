# Спецификация POS Backend

## Назначение

Этот документ фиксирует:

- текущий публичный backend surface;
- state transitions;
- policy compatibility-хвостов для текущего публичного API;
- event catalog Edge runtime;
- границы между implemented now и target later.

## Архитектурная позиция

Edge backend является source of truth для всех активных POS-операций.

Cloud не является runtime dependency для:

- смен;
- кассовых сессий;
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

### Auth

- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`

### Залы и меню

- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/catalog/items`
- `GET /api/v1/menu/items`

implemented now: halls, tables, catalog and menu are Cloud-owned master data. Public Edge runtime writes to these entities return `403 Forbidden`; local/demo setup uses `POST /api/v1/dev/bootstrap-demo`.

### Смены и касса

- `GET /api/v1/shifts/current`
- `POST /api/v1/shifts/open`
- `POST /api/v1/shifts/{id}/close`

- `GET /api/v1/cash-sessions/current`
- `POST /api/v1/cash-sessions/open`
- `POST /api/v1/cash-sessions/{id}/close`

### Заказы

- `GET /api/v1/orders/current?table_id=...`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`

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

Эти routes больше не являются runtime-supported Edge mutation flow. HTTP layer нормализует такие запросы как Edge runtime origin, application services запрещают mutation Cloud-owned master data и возвращают `403 Forbidden`. Разрешенные write origins для этих use cases: `cloud_sync` и `system_seed`.

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

implemented now: Cloud -> Edge provisioning/configuration имеет schema/app foundation: master-data services принимают `cloud_sync`, master tables имеют sync metadata, `cloud_master_sync_state` хранит stream checkpoints. Production snapshot endpoint/apply flow является planned next.

## Manager override

На текущем этапе manager override обязателен как минимум для отмены пречека.

Минимальный контракт payload для override-операции:

- actor context;
- manager employee id;
- manager PIN;
- reason.

Backend обязан:

- проверить active session actor;
- проверить manager employee и manager permission;
- проверить manager PIN;
- записать audit trail;
- записать sync/local events транзакционно.

## Документационные правила

Любое изменение одного из пунктов ниже обновляет этот файл в том же PR:

- список endpoints;
- контракт запроса/ответа;
- переходы состояния;
- event catalog;
- compatibility tails;
- manager override behavior.

Если меняется только долгосрочная архитектурная цель, но не runtime contract, обновляется `SPECv1.3.md`, а не этот файл.
