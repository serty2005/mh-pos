# AGENTS.md

## Назначение

Этот файл задает правила работы агентов, Codex и других инструментов кодогенерации в репозитории `mh-pos`.

Цель:

- не смешивать в одном документе текущий runtime, целевую архитектуру и план работ;
- не тащить compatibility-хвосты до первого пилота;
- не допускать расхождения между кодом, тестами и документацией;
- сохранить код понятным для следующего разработчика и следующей итерации Codex;
- обеспечить безопасную обработку ошибок: технические детали в логах, бизнес-пользователю — понятное сообщение.

Проект еще не имеет production-эксплуатации. До первого пилота действуют правила **first-launch development**, а не **legacy-support development**.

---

## Правила агентской кодогенерации

- Ответы пользователю по умолчанию пишутся на русском языке.
- Промпты для Codex/агентов могут быть написаны на английском языке, если это повышает точность технических требований.
- Идентификаторы в коде пишутся на английском языке:
  - package/module names;
  - function names;
  - method names;
  - variable names;
  - route names;
  - payload field names;
  - error codes;
  - permission IDs;
  - event names;
  - test names.
- Комментарии в коде пишутся на английском языке.
- Имена структурированных полей логов пишутся на английском языке.
- Техническая документация пишется на английском языке, если конкретный документ явно не предназначен для русскоязычной аудитории.
- Русский пользовательский текст допускается только:
  - в `ru` locale files;
  - в документах, которые явно ведутся на русском языке;
  - в прямом общении с пользователем/оператором.
- Нельзя добавлять русские hardcoded strings в source code, кроме `ru` locale files.
- Пользовательские UI-сообщения не хардкодятся в компонентах. Они должны идти через i18n.
- Логи должны быть структурированными: поля логов на английском, бизнес-смысл через стабильный `error_code` / `message_key`, без sensitive данных.
- Нельзя придумывать детали реализации внешних систем, бизнес-правил и неоднозначных требований. Если фактов нет в коде, тестах, документации или явном ответе пользователя, нужно задать уточняющий вопрос.
- Комментарии в коде должны описывать текущую функциональность и текущие инварианты. Не использовать пометки вроде `added`, `updated`, `temporary`, `for now`, если это не часть реального runtime-смысла.
- При работе в PowerShell принудительно включать UTF-8 для вывода и чтения, например `$env:PYTHONIOENCODING='utf-8'`, чтобы кириллица не превращалась в mojibake.
- Для чтения и анализа файлов в этом репозитории использовать Python-скрипты с `encoding='utf-8'`. Для массового поиска можно вызывать быстрые инструменты из Python, но итоговое чтение файлов должно сохранять UTF-8.
- Все файлы репозитория — код, тесты, документация, миграции, конфиги, скрипты — сохраняются исключительно в UTF-8. Использование других кодировок запрещено.
- Не откатывать чужие изменения в рабочем дереве. Если найден dirty state, сначала понять, относится ли он к задаче.
- При разработке Go-кода следовать современным Go practices:
  - small interfaces;
  - context-aware APIs where appropriate;
  - table-driven tests;
  - typed/sentinel errors where useful;
  - no panics in request path;
  - `gofmt`;
  - `go mod tidy`;
  - `go test ./...`.
- При разработке UI использовать доступный инструмент Playwright для проверки изменений UI, если это возможно в текущей среде.
- При разработке кода использовать secure-by-default подход: учитывать CSRF, XSS, injection, broken access control и другие классы атак из OWASP Top 10.
- При проектировании/изменении фич оценивать security severity по CVSS 4.0 там, где это применимо, и снижать риск до приемлемого уровня до merge.
- Валютный каталог должен опираться на полный active ISO 4217 список, а не локальный минимальный subset. Особенно обязательно покрытие валют ЮВА: `IDR`, `THB`, `VND`, `MYR`, `SGD`, `PHP` и др.
- Изменения валютного каталога всегда выполняются end-to-end: backend validation, cloud reference/template, UI precision/formatting, тесты и профильная документация в одном PR.

---

## Language policy

- Operator/user communication: Russian by default.
- Codex/agent prompts: English is allowed and preferred for precise technical work.
- Code comments: English.
- Internal identifiers: English.
- Error codes and message keys: English.
- Permission IDs: English.
- API routes and payload field names: English.
- Event names and sync contract names: English.
- Structured log field names: English.
- Technical documentation: English by default.
- Russian text belongs only in:
  - `ru` locale files;
  - Russian-only documentation;
  - direct user/operator communication.

Do not mix Russian UI text into Vue components, Go handlers, scripts, tests, logs, or seed data unless it is explicitly locale/test data.

---

## Code comments policy

Код и тесты — источник истины для того, что реально реализовано сейчас.

Если документация утверждает одно, а runtime и тесты делают другое, то для описания текущего состояния приоритет у кода и тестов.

Комментарии обязательны для нетривиального кода, но комментарии не должны превращаться в шум.

### Что обязательно комментировать

Добавлять осмысленные комментарии для:

- экспортируемых Go-сущностей: types, interfaces, funcs, methods, constants, если они являются частью package API;
- публичных TypeScript/Vue helpers, composables, stores, API clients и shared models;
- нетривиальных бизнес-правил;
- state transitions:
  - order;
  - precheck;
  - payment;
  - check;
  - shift;
  - cash session;
- RBAC и security-sensitive решений;
- auth/session/client_device_id validation;
- manager override rules;
- error mapping, safe error responses, correlation/request ID handling;
- retry/no-retry policy, особенно для financial mutations;
- sync/outbox/inbox/retry/reclaim logic;
- SQLite transaction boundaries и maintenance operations;
- startup/stop scripts, если есть неочевидное поведение Windows, PowerShell, Docker, PID, process или log handling;
- тестовых fixtures, если без комментария непонятно, какое бизнес-правило проверяется.

### Как писать комментарии

- Комментарии должны объяснять `why`, invariant, business rule или operational constraint.
- Комментарии не должны пересказывать очевидный код.
- Комментарии должны описывать текущее поведение, а не историю изменений.
- TODO допускается только если он конкретный:
  - что осталось сделать;
  - почему это не сделано сейчас;
  - какой planned-next scope должен это закрыть.
- При изменении кода устаревшие комментарии обязательно обновлять или удалять.
- Комментарии не должны содержать PIN, manager PIN, tokens, secrets, credentials или sensitive payloads.

### Плохие комментарии

Не добавлять комментарии такого вида:

```go
// Increment counter.
i++
```

```ts
// Call API.
await api.get(...)
```

```go
// Return error.
return err
```

### Хорошие комментарии

```go
// Operator writes require both a valid employee session and the same client_device_id
// that created the session. This prevents a copied session id from being reused on
// another POS terminal.
```

```ts
// Financial mutations are not retried automatically because duplicate requests can
// create duplicate payments or order state transitions without an idempotency key.
```

```go
// System sync endpoints intentionally bypass employee RBAC because they authenticate
// as device/runtime flows, not as cashier operations.
```

```powershell
# VACUUM is an explicit maintenance operation. Running it on every startup can block
# POS availability and requires extra free disk space on large SQLite databases.
```

---

## UI text and i18n policy

- No user-facing text may be hardcoded directly in Vue components unless the surrounding codebase has an explicit existing exception.
- User-facing labels, validation messages, errors, modals, dialogs, notifications and empty states must use `vue-i18n`.
- If the project supports both English and Russian locales, every new user-facing message must be added to both locales.
- Business-facing error messages must be readable, actionable and safe.
- Technical details must be logged, not shown directly in the UI.
- Critical or business-blocking errors should be shown via modal/dialog flows, not only inline red banners.
- Non-critical validation errors may use field-level or inline messages.
- Backend error responses should expose stable error codes/message keys, not raw internal errors.
- UI should map backend error codes/message keys to localized messages.
- Do not expose raw Go errors, SQL errors, stack traces, exception messages, request dumps, PINs, tokens, secrets or sensitive payloads in UI.

---

## Error handling policy

Technical errors and business errors must be handled differently.

### Backend error policy

Backend APIs should return a stable, typed, safe error contract with:

- stable error code;
- HTTP status;
- safe user-facing message key;
- optional safe details;
- correlation/request ID when available.

Internal causes must be logged, not returned to the user.

Backend must distinguish:

- validation errors;
- authentication errors;
- permission errors;
- conflict/state errors;
- rate limit errors;
- infrastructure/database errors;
- unexpected internal errors.

Unexpected errors must return safe `5xx` responses and write detailed internal logs.

Backend responses must not include:

- PIN;
- manager PIN;
- PIN hash;
- tokens;
- secrets;
- full sensitive payloads;
- raw SQL errors;
- raw Go errors;
- stack traces.

### UI error policy

UI must use a central error normalization and display flow.

UI must distinguish:

- `400/422` validation errors;
- `401` unauthenticated/revoked session;
- `403` permission denied;
- `404` not found;
- `409` business/state conflict;
- `429` rate limit;
- `5xx` server error;
- network/timeout/backend unavailable.

Required behavior:

- `401` or revoked session follows the auth/session recovery policy.
- `403` shows permission-denied UX without logout if the session is otherwise valid.
- `409` shows a business-readable conflict reason.
- `429` shows rate-limit guidance.
- Network/backend-unavailable errors must not trigger destructive logout by themselves.
- Critical/business-blocking errors should use modal/dialog UX.
- Non-critical validation errors may use field-level or inline messages.

---

## Logging policy

Backend operations and actions must use structured logging with a consistent field contract.

Required fields where applicable:

- `request_id`
- `operation`
- `action`
- `result`
- `duration_ms`
- `error_code`
- `node_device_id` masked
- `client_device_id` masked
- `session_id` masked
- `actor_employee_id` masked

Log levels:

- `TRACE` — detailed internal operation steps, disabled by default.
- `DEBUG` — diagnostic technical context.
- `INFO` — successful business operations and state transitions.
- `WARN` — expected rejections and denied requests, including `4xx`, `403`, `409`, `429`.
- `ERROR` — unexpected failures and infrastructure errors, including `5xx`.

Never log:

- PIN;
- manager PIN;
- PIN hash;
- tokens;
- secrets;
- credentials;
- raw auth payloads with sensitive fields;
- full payment-sensitive payloads.

---

## Worker telemetry policy

Non-HTTP background workers must use shared telemetry helpers for normalized fields:

- `operation`
- `action`
- `result`
- `error_code`
- masked correlation IDs where available:
  - `node_device_id`
  - `client_device_id`
  - `session_id`
  - `actor_employee_id`

TRACE logs are required for worker lifecycle internal steps:

- batch claim;
- process;
- send;
- retry;
- reclaim;
- shutdown.

Temporary local artifacts, for example `test_pipe/`, are not part of the managed runtime path and must be removed before merge unless explicitly documented and owned.

---

## Security policy

Development must be secure by default.

For every feature or change, consider:

- broken access control;
- session fixation/reuse;
- client_device_id spoofing;
- CSRF;
- XSS;
- SQL injection;
- command injection;
- insecure deserialization;
- sensitive data exposure;
- unsafe logging;
- replay/double-submit risks;
- unsafe retry of financial operations.

Security-sensitive decisions must be enforced on the backend. UI visibility is not a security boundary.

RBAC and session checks must be enforced in application/use-case logic where possible, not only in HTTP handlers.

---

## Financial operation safety

Financial and order state mutations must be treated as high-risk operations.

- Do not auto-retry payment/order write operations unless an idempotency key or equivalent safety mechanism exists.
- Do not let frontend decide financial state transitions.
- Frontend may display and request actions, but backend owns business rules and state transitions.
- Payment, precheck, check, order, shift and cash session transitions must be validated by backend invariants.
- Duplicate submit protection is required in UI for financial mutations.
- Backend must still protect against duplicate or invalid transitions.

---

## Документы и их владельцы

### Этот файл отвечает за

- границы между документами;
- порядок приоритетов при конфликте источников;
- правила поддержки документации;
- правила clean-before-pilot;
- требования к compatibility tails;
- language policy;
- code comments policy;
- logging/error/i18n baseline policies.

### Этот файл не отвечает за

- полный перечень HTTP endpoints;
- полный перечень экранов UI;
- полное описание схемы БД;
- детальный roadmap по стадиям;
- полный sync contract;
- полную RBAC matrix.

Для этого существуют профильные документы.

---

## Карта источников истины

### Код и тесты

Код и тесты — источник истины для того, что **реально реализовано сейчас**.

Если документация утверждает одно, а runtime и тесты делают другое, то для описания текущего состояния приоритет у кода и тестов.

### SPECv1.3

`SPECv1.3.md` — источник истины для:

- архитектурных инвариантов;
- финансовой модели;
- identity model;
- sync model;
- security baseline;
- pilot topology.

### UI-спецификация

`docs/ui/POS-UI-SPEC.md` — источник истины для:

- текущих и целевых экранов;
- пользовательских сценариев;
- UI flow;
- границ frontend responsibility;
- supported и unsupported UX surface.

### UI RBAC

`docs/ui/POS-UI-RBAC.md` — источник истины для:

- ролей сотрудников;
- permission catalog;
- матрицы прав по UI-операциям;
- правил manager override.

### Backend-спецификация

`docs/backend/POS-BACKEND-SPEC.md` — источник истины для:

- публичного API;
- state transitions;
- event catalog;
- текущих compatibility tails.

### Backend data/migrations

`docs/backend/POS-DATA-AND-MIGRATIONS.md` — источник истины для:

- ключевых сущностей;
- связей между сущностями;
- first-launch schema policy;
- reset/migration policy;
- требований к БД;
- SQLite maintenance policy.

### Error catalog

`docs/backend/POS-ERROR-CATALOG.md`, если существует, — источник истины для:

- backend error codes;
- HTTP status mapping;
- message keys;
- business meaning;
- retryability;
- recommended UI behavior;
- logging level;
- sensitive-data policy.

### Architecture docs

`docs/architecture/` — источник истины для:

- target architecture;
- modular monolith rules;
- DDD bounded contexts;
- dependency direction;
- long-term service extraction boundaries.

Architecture docs must clearly separate:

- `implemented now`
- `planned next`
- `out of scope`

### Sync contracts

`docs/sync/edge-cloud-contracts-v1.md` — источник истины для:

- edge-cloud sync contracts;
- domain event envelopes;
- outbox/inbox behavior;
- cloud sync API semantics.

### Roadmap

`ROADMAP.md` — источник истины для:

- what is done / next / blocked;
- рисков;
- pilot gates;
- sequencing.

### Local E2E Prototype Quickstart

`README.md` — источник истины для локального запуска:

- `pos-ui -> pos-backend -> cloud-backend`;
- demo bootstrap;
- smoke scripts;
- dev environment notes.

Dev bootstrap endpoint `POST /api/v1/dev/bootstrap-demo` относится только к local/dev режиму и требует `POS_DEV_TOOLS=1`.

---

## Порядок приоритета при конфликте

Если есть конфликт между документами, использовать такой порядок:

1. Безопасность и архитектурные инварианты: `SPECv1.3.md`.
2. Текущее фактическое поведение: код и тесты.
3. UI / backend surface contracts: профильные документы в `docs/ui/` и `docs/backend/`.
4. Sync contracts: `docs/sync/`.
5. План выполнения и статусы: `ROADMAP.md`.
6. `README.md` — обзорный quickstart-документ, не финальный арбитр архитектуры.

При разрешении конфликта:

- обновить устаревший документ;
- не сохранять противоречия молча;
- явно использовать статусы `implemented now`, `planned next`, `out of scope`.

---

## Clean-before-pilot policy

До первого пилота запрещено:

- поддерживать старое поведение "на всякий случай", если его не существует в production;
- добавлять dual-write ради legacy compatibility;
- добавлять исторические DB migrations ради обратной совместимости dev-схем;
- тащить legacy aliases без владельца и срока удаления;
- документировать unsupported future behavior как будто оно уже доступно.

До первого пилота разрешено:

- переписывать canonical first-launch schema;
- удалять deprecated transport/API tails;
- пересоздавать dev/test БД;
- переименовывать сущности и endpoints так, чтобы модель становилась чище.

---

## Политика схемы БД до первого пилота

До первого пилота SQLite развивается по правилу:

- один канонический `001_init.sql`;
- никакой обязательной исторической цепочки `002`, `003`, `004` для локального runtime;
- любое изменение схемы до пилота делается через обновление canonical init;
- dev/test databases регенерируются с нуля.

Если потребуется редкий временный migration script для локальной разработки, он не становится частью canonical pilot path без отдельного архитектурного решения.

---

## Политика версий модулей и миграций БД

Implemented now:

- Все runtime-модули используют единую версию продукта через `MH_POS_VERSION`, fallback: `0.1.0`.
- Контекст БД включает любую используемую БД решения: `SQLite` и `PostgreSQL`.
- Любые миграции/обновления схемы выполняются программно при старте модуля.
- Ручной ad-hoc upgrade path не считается canonical.
- Каждая БД обязана иметь runtime-таблицу версий модулей: `db_runtime_versions`.
- Если модуль видит, что версия БД ниже версии модуля, он обязан:
  - сначала выполнить backup текущего состояния;
  - только после backup запускать schema upgrade / apply canonical migration;
  - после успешного обновления записать новую версию модуля в таблицу версий.

---

## SQLite maintenance policy

SQLite maintenance must be explicit and safe.

- Do not run `VACUUM` inside an active write transaction.
- Do not run `VACUUM` automatically on every startup.
- `VACUUM` is a maintenance/dev/reset operation, not part of the normal POS write path.
- `VACUUM` can be long-running and can require extra free disk space.
- `VACUUM INTO` may be used for compact snapshot/backup flows when appropriate.
- Maintenance scripts must be explicit, Windows-compatible and safe by default.
- Potentially destructive maintenance commands must require clear operator intent.
- SQLite maintenance policy must be documented in `docs/backend/POS-DATA-AND-MIGRATIONS.md`.

---

## Политика compatibility tails

Любой compatibility tail допустим только если у него есть:

- владелец;
- причина существования;
- срок удаления или milestone удаления;
- тест, который подтверждает текущее поведение;
- запись в профильной спецификации backend/UI.

Если у compatibility tail нет срока удаления, он не должен быть merged.

### Что считается compatibility tail

Примеры:

- deprecated alias endpoint;
- legacy transport field;
- старое enum name;
- temporary adapter layer between old/new payload;
- документация со старым названием сущности, оставленная "ради привычки".

---

## Правило синхронного обновления документации

Любой PR, который меняет одно из следующих, обязан менять профильную документацию в том же PR:

- HTTP routes or payloads;
- UI screens or user flows;
- permission model;
- DB schema / invariants;
- sync event catalog;
- error contract;
- logging contract;
- migration/reset policy;
- local dev startup/smoke scripts.

---

## Правило формулировок в документации

В документации использовать только три формулировки статуса:

- `implemented now`
- `planned next`
- `out of scope`

Не использовать расплывчатые формулировки:

- `temporary for now`
- `legacy but maybe keep`
- `later we will see`
- `probably`
- `maybe supported`

---

## DDD bounded contexts policy

The POS Core target architecture is a modular monolith with explicit bounded contexts.

The architecture may evolve toward separate services later, but the current implementation should avoid premature service extraction.

Rules:

- Do not create a single generic `POSContext` that owns everything.
- Each bounded context should have clear domain ownership.
- Cross-context communication should happen through contracts, APIs or events.
- Avoid direct cross-context database coupling.
- Architecture docs must distinguish current implementation from target architecture.
- Do not perform a large package refactor only to match target DDD docs unless the task explicitly requires it.

Primary target contexts:

- Organization
- Catalog
- Pricing
- Order
- Payment
- Fiscal / Tax
- Production
- Inventory
- Procurement
- CRM
- Loyalty
- Delivery / Channel
- Reservation / Table
- Staff / Shift
- Accounting / Finance
- Event / Integration

---

## UI/backend responsibility boundary

Backend is the source of truth for:

- business rules;
- state transitions;
- permission enforcement;
- financial calculations;
- order/precheck/payment/check lifecycle;
- shift/cash session lifecycle;
- sync/outbox behavior.

Frontend is responsible for:

- rendering state;
- collecting user intent;
- local UI state;
- client identity storage;
- user-friendly error display;
- permission-based visibility as UX only.

Frontend must not:

- bypass backend invariants;
- make final permission decisions;
- calculate final financial state;
- decide order/payment/check transitions independently;
- treat hidden/disabled UI as security enforcement.

---

## RBAC policy

RBAC must be enforced on the backend.

- UI visibility is only UX, not security.
- Business/operator write operations require active employee session where applicable.
- Operator/business flows must validate:
  - session;
  - actor employee;
  - client_device_id;
  - required permission.
- System/device flows may bypass employee session only when explicitly designed and documented.
- Permission IDs must come from a canonical permission catalog.
- Avoid ad-hoc permission strings in runtime code.
- Role matrices must be documented and tested.
- Permission denied events should be audit logged without sensitive data.

Canonical roles should include:

- `cashier`
- `senior_cashier`
- `waiter`
- `manager`
- `kitchen`
- `support_admin`

---

## Local dev scripts policy

Scripts must be predictable and safe.

For PowerShell scripts:

- use `$PSScriptRoot` for script-relative paths;
- use UTF-8-aware reading/writing;
- write useful logs for background services;
- write PID files for started background processes;
- provide stop scripts or clear stop instructions;
- show tail logs when health checks fail;
- avoid silently spawning duplicate services;
- check required ports before startup.

For `.cmd` scripts:

- use `set "VAR=value"`;
- use quoted paths;
- use `%~dp0` for script-relative paths;
- use `call` when invoking another `.cmd`, `npm`, or batch-style command;
- use `exit /b 1` for errors;
- do not use PowerShell syntax such as `$env:VAR="..."`.

---

## Минимальный чек перед merge

Перед merge проверить:

- тесты не спорят с документацией;
- `README.md` не обещает unsupported behavior;
- профильный документ обновлен;
- deprecated tails либо удалены, либо помечены с owner и kill date;
- документация отделяет `implemented now` от `planned next` и `out of scope`;
- user-facing UI strings идут через i18n;
- critical/business errors используют safe, readable UX;
- technical errors логируются, а не показываются пользователю;
- code comments осмысленные и не шумные;
- sensitive data не логируется и не показывается;
- local scripts не оставляют unmanaged processes без stop-инструкций.

---

## Required checks

When applicable, run:

### POS backend

```bash
cd pos-backend
go mod tidy
go test ./...
```

### Cloud backend

```bash
cd cloud-backend
go mod tidy
go test ./...
```

### POS UI

```bash
cd pos-ui
npm install
npm run build
```

If `package.json` contains lint/typecheck/test scripts, run them when relevant.

If a check cannot be executed in the current environment, the final report must state:

- which check was not executed;
- why it was not executed;
- what was checked instead.

---

## Final report requirements for agent tasks

Every substantial agent task must finish with a report containing:

- what was found;
- what was changed;
- which files were changed;
- which tests/checks were run;
- which checks could not be run and why;
- remaining risks;
- `planned next`;
- `out of scope`.

For hardening tasks, include separate sections for:

- RBAC status;
- error handling status;
- UI/backend contract status;
- logging/audit status;
- i18n status;
- documentation sync status;
- SQLite maintenance status, if relevant;
- local scripts/smoke status, if relevant.
