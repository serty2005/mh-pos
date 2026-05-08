# Спецификация POS Backend

## Назначение

Этот документ фиксирует:

- текущий публичный backend surface;
- state transitions;
- policy compatibility-хвостов для текущего публичного API;
- event catalog Edge runtime;
- границы между статусами `реализовано сейчас`, `запланировано далее` и `вне текущего объема`.

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

## DB startup и schema verification

Реализовано сейчас:

- POS Edge до запуска HTTP server и sync worker открывает SQLite, проверяет runtime gate (`WAL`, `foreign_keys`, `busy_timeout`, SQLite version), применяет один managed canonical SQL file `001_init.sql` при version-gated upgrade и выполняет schema verification критичных таблиц/колонок/индексов.
- POS Edge использует `db_runtime_versions` и `schema_migrations`; если `db_runtime_versions` отсутствует, БД считается самой старой и запускается upgrade path.
- Перед safe schema/data upgrade существующей SQLite БД создается backup `.db/.db-wal/.db-shm` в `POS_SQLITE_BACKUP_DIR` после WAL checkpoint.
- Cloud backend до запуска HTTP server применяет один managed canonical PostgreSQL SQL file `001_sync_receiver.sql` под advisory lock и выполняет schema verification runtime-таблиц.
- Cloud backend использует `db_runtime_versions` и `schema_migrations`; если `db_runtime_versions` отсутствует, БД считается самой старой и запускается upgrade path.
- Если PostgreSQL `schema_migrations` отсутствует или содержит старую запись без checksum, startup повторно применяет idempotent `001_sync_receiver.sql`, чтобы создать недостающие реализованные сейчас runtime tables до schema verification.
- Перед safe schema/data upgrade существующей PostgreSQL схемы создается JSONL snapshot таблиц `public` в `CLOUD_POSTGRES_BACKUP_DIR`.
- `schema_migrations` хранит имя active SQL file, SHA-256 checksum, status и `applied_at`; checksum drift при той же версии завершает startup fail-fast, а при `db version < MH_POS_VERSION` применяется как управляемый upgrade.
- `DB version > MH_POS_VERSION` завершает startup fail-fast, downgrade не поддерживается.
- Ошибки открытия БД, lock, backup, migration и schema verification логируются structured logs с безопасным контекстом (`db_type`, `module_name`, `operation`, `action`, `result`, `migration_file`, `migration_dir`, `error_code`, `duration_ms`).
- Runtime-код не должен обращаться к business tables до успешных migrations и schema verification.

Запланировано далее:

- production-grade backup retention/restore policy и отдельный observability report для миграций;
- backup-before-data-load для Cloud -> Edge full snapshot/master-data import;
- административная UI-операция очистки/пересоздания SQLite с backup, явным подтверждением, RBAC/audit и restart/rebootstrap flow.

Вне текущего объема:

- online zero-downtime production migration orchestration и rollback arbitrary destructive migrations.

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

Реализовано сейчас: `POST /api/v1/system/pair` сохраняет verifier pairing code в keyed format `pairing.hmac-sha256.v1`; plaintext pairing code не сохраняется.

### Auth

- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`

Реализовано сейчас: PIN login имеет rate limit по `node_device_id + client_device_id`.
Реализовано сейчас: повторные неверные PIN-попытки возвращают `429 Too Many Requests`.
Реализовано сейчас: PIN values никогда не возвращаются в response payloads.
Реализовано сейчас: PIN login должен найти ровно одного active employee в paired restaurant; duplicate active PIN matches возвращают conflict.
Реализовано сейчас: `GET /api/v1/auth/session` возвращает безопасную `401 SESSION_REVOKED` ошибку для revoked sessions вместо revoked session data.

### Залы и меню

- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/catalog/items`
- `GET /api/v1/menu/items`

Реализовано сейчас: halls, tables, catalog и menu являются Cloud-owned master data. Public Edge runtime writes к этим сущностям возвращают `403 Forbidden`; local/demo setup использует `POST /api/v1/dev/bootstrap-demo`.

### Смены и касса

- `GET /api/v1/employee-shifts/current`
- `GET /api/v1/employee-shifts/recent`
- `POST /api/v1/employee-shifts/open`
- `POST /api/v1/employee-shifts/{id}/close`

- `GET /api/v1/cash-shifts/current`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `POST /api/v1/cash-drawer-events`

Реализовано сейчас:

- `auth_sessions` остаются техническим login/logout-контекстом устройства и клиента.
- `shifts` используются как личные смены сотрудника: открытая смена ищется по `restaurant_id + employee_id`, а не по устройству.
- Без открытой личной смены сотрудника business/runtime операции запрещены; доступны только открытие личной смены и чтение последних личных смен текущего actor.
- Заказы, позиции и пречеки требуют открытую личную смену, но не требуют открытую кассовую смену.
- Оплаты и cash drawer events требуют открытую кассовую смену на устройстве.
- `cash_sessions` являются текущей runtime-сущностью кассовой смены.

Запланировано далее:

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
- `POST /api/v1/prechecks/{id}/reprint`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/checks/{id}`
- `POST /api/v1/checks/{id}/reprint`

Реализовано сейчас:

- reprint precheck требует `pos.precheck.reprint` и возвращает copy-document payload из immutable `prechecks.snapshot`;
- reprint final check требует `pos.check.reprint` и возвращает copy-document payload из immutable `checks.snapshot`;
- reprint не использует текущее состояние order как source of truth;
- reprint response содержит `copy_marker = "COPY"`, а русская UI-метка `КОПИЯ` задается через i18n;
- reprint пишет audit events `PrecheckReprinted` / `CheckReprinted` в `local_event_log` и outbox.

### Operational sync endpoints

- `GET /api/v1/sync/outbox`
- `GET /api/v1/sync/status`
- `GET /api/v1/sync/local-events`
- `POST /api/v1/sync/retry-failed`

Реализовано сейчас: operator-facing sync endpoints enforced через app-layer RBAC:

- `GET /api/v1/sync/outbox` requires `pos.sync.view`;
- `GET /api/v1/sync/status` requires `pos.sync.view`;
- `GET /api/v1/sync/local-events` requires `pos.sync.view`;
- `POST /api/v1/sync/retry-failed` requires `pos.sync.retry_failed`.

### Cloud -> Edge master-data ingest endpoints

Реализовано сейчас:

- `POST /api/v1/sync/master-data/snapshots`
- `POST /api/v1/sync/master-data/{stream}`

Supported streams: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.

Payload accepts `node_device_id`, optional `restaurant_id`, `sync_mode` (`full_snapshot` or `incremental`), optional `checkpoint_token`, `cloud_version`, optional `cloud_updated_at`, and stream arrays: `restaurants`, `devices`, `roles`, `employees`, `halls`, `tables`, `catalog_items`, `menu_items`.

Реализовано сейчас: эти endpoints являются Cloud -> Edge ingest, а не POS runtime mutation APIs. Handler задает origin `cloud_sync`, вызывает app-layer master sync use case, пишет master rows и `cloud_master_sync_state` в одной транзакции и не создает строки `local_event_log` или `pos_sync_outbox`.

### Dev/local bootstrap

Реализовано сейчас:

- `POST /api/v1/dev/bootstrap-demo`
- доступен только при `POS_DEV_TOOLS=1`;
- создает demo restaurant, paired Edge Node, cashier/manager roles, сотрудников с PIN `1111`/`2222`, hall/table и menu items;
- возвращает `pairing_code` и `manager_employee_id` для ручного POS UI smoke flow;
- не является production path.

### Master-data mutation boundary

Реализовано сейчас:

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

Эти routes больше не являются runtime-supported Edge mutation flow. Реализовано сейчас: HTTP layer держит их как dev-only seed/admin helpers за `POS_DEV_TOOLS=1`; без dev tools они возвращают `403 Forbidden`. В dev mode handler использует origin `system_seed`. Production Cloud-authored master data должна входить через `POST /api/v1/sync/master-data/snapshots` или `POST /api/v1/sync/master-data/{stream}` с origin `cloud_sync`.

## Policy compatibility-хвостов

Реализовано сейчас: публичные compatibility tails удалены из backend API surface.

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
- `issued` -> `superseded` зарезервировано для будущего re-issue flow.

### Payment

- создается как immutable факт;
- не редактируется;
- не удаляется;
- correction делается отдельными финансовыми операциями, а не mutate-in-place.

### Check

- создается только после полной оплаты precheck;
- не является рабочим счетом гостя;
- не создается вручную в нормальном runtime flow;
- `business_date_local` вычисляется backend в момент создания final check и после этого immutable;
- `closed_at` фиксирует фактическое время закрытия.

### Business date

Реализовано сейчас:

- restaurant config содержит `business_day_mode` (`standard` или `24_7`) и `business_day_boundary_local_time`;
- в `standard` режиме учетный день вычисляется по локальному времени ресторана с учетом ресторанной границы дня;
- в `24_7` режиме учетный день равен локальной календарной дате финансового события;
- финансовая принадлежность определяется моментом capture payment / final check creation, а не временем создания order;
- открытый order может пережить новую смену, но `business_date_local` для созданных checks/payments не меняется;
- ручной перенос закрытых orders/payments в другой business date является вне текущего объема.

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
- `PrecheckReprinted`
- `PrecheckCancelled`
- `PaymentCaptured`
- `CheckCreated`
- `CheckReprinted`

## Примечание к sync contract

Документация Cloud receiver и Edge event emission должны быть синхронизированы.

Реализовано сейчас: production sender path отправляет только Edge -> Cloud operational events. Cloud-managed/configuration events, например изменения restaurant, employee, role, catalog, menu, hall и table, не отправляются sender-ом вверх; они помечаются `suspended` с явной sync-direction причиной.

Реализовано сейчас: Cloud принимает operational sender catalog, описанный в `docs/sync/edge-cloud-contracts-v1.md`, и хранит raw envelopes плюс `cloud_operational_events`. Ownership matrix и directional sync rules описаны в `docs/sync/directional-sync-ownership.md`.

Реализовано сейчас: Cloud -> Edge provisioning/configuration имеет backend apply flow: `internal/pos/app/mastersync` принимает `cloud_sync`, master tables имеют sync metadata, `cloud_master_sync_state` хранит stream checkpoints, а dedicated sync endpoints применяют full snapshot/incremental payloads для supported streams. Full snapshot replacement policy beyond upserted payload rows запланирована далее.

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

## RBAC enforcement

Реализовано сейчас:

- backend uses canonical permission catalog and canonical role profiles for `cashier`, `senior_cashier`, `waiter`, `manager`, `kitchen`, `support_admin`;
- role permissions remain stored as JSON, but role creation/import rejects unknown permission ids;
- implemented POS runtime operations are enforced in app services via `EnsureOperatorSession(...requiredPermissions...)`;
- master-data list endpoints for restaurants/devices/roles/employees are dev-only behind `POS_DEV_TOOLS=1`;
- `GET /api/v1/catalog/items` is an operator endpoint and requires `pos.catalog.view`.

Canonical permission IDs, используемые текущим runtime:

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
- `pos.precheck.reprint`
- `pos.precheck.cancel.request` (override actor permission)
- `pos.precheck.cancel` (manager override approver permission)
- `pos.payment.cash`
- `pos.payment.card.manual`
- `pos.payment.other`
- `pos.check.view`
- `pos.check.reprint`
- `pos.sync.view` (required for operator-triggered `GET /api/v1/sync/outbox`, `GET /api/v1/sync/status`, `GET /api/v1/sync/local-events`)
- `pos.sync.retry_failed` (required for operator-triggered `POST /api/v1/sync/retry-failed`)

Role behavior:

- `cashier`: cashier POS flow, cash/card payment, no cash session close and no sync/service permissions.
- `senior_cashier`: cashier POS flow plus cash session close and sync read.
- `waiter`: order/precheck/check read/write flow without cash session/payment permissions; precheck reprint allowed.
- `manager`: full implemented POS runtime permissions, precheck cancel approval, final check reprint, sync retry.
- `kitchen`: no implemented POS runtime permissions.
- `support_admin`: sync read and retry service permissions only.

Error behavior:

- missing permission returns domain `forbidden` and safe HTTP `403` error code `PERMISSION_DENIED`;
- revoked sessions return `401 SESSION_REVOKED`;
- wrong `client_device_id`/session context returns `403 SESSION_CONTEXT_MISMATCH`;
- authorization errors do not include sensitive auth fields (PIN, manager PIN, PIN hash) or raw permission internals in response payloads.

Вне текущего объема:

- order transfer;
- payment refund;
- diagnostics/admin UI routes;
- waiter payment override and restaurant-level override policy engine.

## Currency policy

Реализовано сейчас:

- backend validates runtime currency codes against canonical active ISO 4217 profile catalog;
- catalog coverage is full active ISO list (including SEA currencies such as `IDR`, `THB`, `VND`, `MYR`, `SGD`, `PHP`);
- precision is currency-code driven and supports minor units `0/2/3/4` where defined by ISO profile;
- pricing/payment domain amounts continue to use integer minor units (no floating-point storage);
- unsupported currency code is rejected as domain `invalid`.

## API error contract

Реализовано сейчас:

- API errors use one JSON envelope: `{ "error": { "code", "message_key", "details", "correlation_id" } }`;
- `code` is stable and machine-readable; `message_key` is safe for UI i18n;
- `X-Error-Code` carries the same stable code for audit middleware;
- `X-Request-ID`/`correlation_id` is returned when request context has a request id;
- internal Go/SQL/domain error text is logged, but not returned to UI;
- panic recovery returns safe `500 INTERNAL_ERROR` and writes stack trace only to backend log.

Реализованные error codes и UI behavior описаны в `docs/backend/POS-ERROR-CATALOG.md`.

## Документационные правила

Любое изменение одного из пунктов ниже обновляет этот файл в том же PR:

- список endpoints;
- контракт запроса/ответа;
- переходы состояния;
- event catalog;
- compatibility tails;
- manager override behavior.

Если меняется только долгосрочная архитектурная цель, но не runtime contract, обновляется `SPECv1.3.md`, а не этот файл.

## Operational logging

- Backend writes structured operation logs with levels `TRACE|DEBUG|INFO|WARN|ERROR`.
- Request audit logs include `request_id`, `operation`, `action`, `result`, `duration_ms`, `error_code` and masked actor/device/session identifiers.
- Sensitive auth fields (`pin`, `manager_pin`, pin hash, raw auth payload) must not be logged.

## Sync sender telemetry

- `internal/pos/syncsender` emits normalized worker telemetry for non-HTTP paths with fields `operation`, `action`, `result`, `error_code` and masked correlation ids.
- TRACE-level lifecycle events are emitted for reclaim, batch claim, per-message processing, send attempt, ack and retry decision steps.
