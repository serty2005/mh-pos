# POS Backend Specification

## Назначение

Этот документ фиксирует:

- текущий публичный backend surface;
- state transitions;
- current compatibility tails;
- event catalog Edge runtime;
- границы между implemented now и target later.

## Архитектурная позиция

Edge backend является source of truth для всех активных POS-операций.

Cloud не является runtime dependency для:

- shifts
- cash sessions
- orders
- prechecks
- payments
- final checks
- manager override

## Финансовая модель

Каноническая модель:

```text
Order -> Precheck -> Payment -> Check
```

Правила:

- `Order` — рабочая сущность обслуживания;
- `Precheck` — рабочий финансовый snapshot;
- `Payment` — immutable финансовый факт;
- `Check` — финальный расчетный документ после полной оплаты precheck.

## Текущий публичный API

### Health and system

- `GET /health`
- `GET /api/v1/system/pairing-status`
- `POST /api/v1/system/pair`

### Auth

- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`

### Floor and menu

- `GET /api/v1/halls`
- `POST /api/v1/halls`
- `GET /api/v1/tables`
- `POST /api/v1/tables`
- `GET /api/v1/menu/items`

### Shift and cash

- `GET /api/v1/shifts/current`
- `POST /api/v1/shifts/open`
- `POST /api/v1/shifts/{id}/close`

- `GET /api/v1/cash-sessions/current`
- `POST /api/v1/cash-sessions/open`
- `POST /api/v1/cash-sessions/{id}/close`

### Orders

- `GET /api/v1/orders/current?table_id=...`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`

### Prechecks and checks

- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/prechecks/{id}`
- `GET /api/v1/orders/{id}/prechecks`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/checks/{id}`

### Sync operational endpoints

- `GET /api/v1/sync/outbox`
- `GET /api/v1/sync/status`
- `GET /api/v1/sync/local-events`
- `POST /api/v1/sync/retry-failed`

## Explicit compatibility tails

На текущем этапе в backend существуют следующие tails:

### Deprecated alias endpoint

`POST /api/v1/orders/{id}/check`

Смысл:

- временный alias;
- вызывает issue precheck flow;
- не создает legacy working check.

Статус:

- должен быть удален до pilot API freeze.

### Legacy transport alias

`device_id`

Смысл:

- backward-compatible alias для `node_device_id`.

Статус:

- поддерживается транспортно;
- не должен продвигаться в новой документации, payload examples и новых UI-клиентах.

### Manual check creation command

Manual `CreateCheck` path считается disabled.
Финальный check создается только автоматически после полной оплаты precheck.

## State transitions

### Order

- `open` -> `locked` при `IssuePrecheck`
- `locked` -> `open` при `CancelPrecheck`
- `open` или `locked` -> `closed` только после полной оплаты и final check
- `closed` не редактируется

### Precheck

- `issued` -> `cancelled`
- `issued` -> `closed` при полной оплате
- `issued` -> `superseded` зарезервировано для future re-issue flow

### Payment

- создается как immutable факт
- не редактируется
- не удаляется
- correction делается отдельными финансовыми операциями, а не mutate-in-place

### Check

- создается только после полной оплаты precheck
- не является рабочим счетом гостя
- не создается вручную в нормальном runtime flow

## Event catalog Edge runtime

На Edge runtime уже существуют или должны существовать в outbox/local event log события следующих групп:

### System and auth

- `EdgeNodePaired`
- `AuthSessionStarted`
- `AuthSessionRevoked`

### Shift and cash

- `ShiftOpened`
- `ShiftClosed`
- `CashSessionOpened`
- `CashSessionClosed`
- `CashDrawerEventRecorded`

### Orders

- `OrderCreated`
- `OrderLineAdded`
- `OrderLineQuantityChanged`
- `OrderLineVoided`
- `OrderClosed`

### Financial

- `PrecheckIssued`
- `PrecheckCancelled`
- `PaymentCaptured`
- `CheckCreated`

## Sync contract note

Cloud receiver documentation и Edge event emission должны быть синхронизированы.

Пока Cloud не принимает весь фактический event catalog Edge, real sender/worker не считается pilot-ready.

## Manager override

На текущем этапе manager override обязателен как минимум для отмены пречека.

Минимальный payload contract для override operation:

- actor context
- manager employee id
- manager PIN
- reason

Backend обязан:

- проверить active session actor;
- проверить manager employee и manager permission;
- проверить manager PIN;
- записать audit trail;
- записать sync/local events транзакционно.

## Документационные правила

Любое изменение одного из пунктов ниже обновляет этот файл в том же PR:

- endpoint list;
- request/response contract;
- state transitions;
- event catalog;
- compatibility tails;
- manager override behavior.

Если меняется только долгосрочная архитектурная цель, но не runtime contract, обновляется `SPECv1.3.md`, а не этот файл.
