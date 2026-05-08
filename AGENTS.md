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
- Промпты для Codex/агентов по умолчанию пишутся на русском языке. Английский допускается только для точных внешних цитат, API/SDK-терминов, команд, идентификаторов и случаев, где русская формулировка ухудшает однозначность технического требования.
- Идентификаторы в коде пишутся на английском языке:
  - имена packages/modules;
  - имена functions;
  - имена methods;
  - имена variables;
  - имена routes;
  - имена payload fields;
  - error codes;
  - permission IDs;
  - event names;
  - test names.
- Комментарии в коде пишутся на русском языке.
- Имена структурированных полей логов пишутся на английском языке.
- Вся документация проекта пишется на русском языке.
- Русский пользовательский текст допускается:
  - в `ru` locale files;
  - в документации;
  - в комментариях к коду;
  - в прямом общении с пользователем/оператором.
- Нельзя добавлять русские hardcoded UI strings в source code, кроме `ru` locale files. Русские комментарии к коду разрешены и обязательны по правилам ниже.
- Пользовательские UI-сообщения не хардкодятся в компонентах. Они должны идти через i18n.
- Логи должны быть структурированными: поля логов на английском, бизнес-смысл через стабильный `error_code` / `message_key`, без sensitive данных.
- Нельзя придумывать детали реализации внешних систем, бизнес-правил и неоднозначных требований. Если фактов нет в коде, тестах, документации или явном ответе пользователя, нужно задать уточняющий вопрос.
- Комментарии в коде должны описывать текущую функциональность и текущие инварианты. Не использовать пометки вроде `added`, `updated`, `temporary`, `for now`, если это не часть реального runtime-смысла.
- При работе в PowerShell принудительно включать UTF-8 для вывода и чтения, например `$env:PYTHONIOENCODING='utf-8'`, чтобы кириллица не превращалась в mojibake.
- Для чтения и анализа файлов в этом репозитории использовать Python-скрипты с `encoding='utf-8'`. Для массового поиска можно вызывать быстрые инструменты из Python, но итоговое чтение файлов должно сохранять UTF-8.
- Все файлы репозитория — код, тесты, документация, миграции, конфиги, скрипты — сохраняются исключительно в UTF-8. Использование других кодировок запрещено.
- Не откатывать чужие изменения в рабочем дереве. Если найден dirty state, сначала понять, относится ли он к задаче.
- При разработке Go-кода следовать современным Go practices:
  - маленькие interfaces;
  - context-aware APIs там, где это уместно;
  - table-driven tests;
  - typed/sentinel errors там, где это полезно;
  - без panics в request path;
  - `gofmt`;
  - `go mod tidy`;
  - `go test ./...`.
- При разработке UI использовать доступный инструмент Playwright для проверки изменений UI, если это возможно в текущей среде.
- При разработке кода использовать secure-by-default подход: учитывать CSRF, XSS, injection, broken access control и другие классы атак из OWASP Top 10.
- При проектировании/изменении фич оценивать security severity по CVSS 4.0 там, где это применимо, и снижать риск до приемлемого уровня до merge.
- Валютный каталог должен опираться на полный active ISO 4217 список, а не локальный минимальный subset. Особенно обязательно покрытие валют ЮВА: `IDR`, `THB`, `VND`, `MYR`, `SGD`, `PHP` и др.
- Изменения валютного каталога всегда выполняются end-to-end: backend validation, cloud reference/template, UI precision/formatting, тесты и профильная документация в одном PR.

---

## Языковая политика

- Общение с оператором/пользователем: русский язык по умолчанию.
- Промпты для Codex/агентов: русский язык по умолчанию; английский допускается только для внешних технических терминов, цитат, команд и идентификаторов.
- Комментарии в коде: русский язык.
- Документация проекта: русский язык.
- Внутренние идентификаторы: английский язык.
- Error codes и message keys: английский язык.
- Permission IDs: английский язык.
- API routes и payload field names: английский язык.
- Event names и sync contract names: английский язык.
- Имена структурированных полей логов: английский язык.
- Русский текст допустим:
  - `ru` locale files;
  - документация;
  - комментарии к коду;
  - прямое общение с пользователем/оператором.

Не добавлять русский UI-текст напрямую в Vue components, Go handlers, scripts, tests, logs или seed data, если это не locale/test data. Русские комментарии и русская документация не считаются UI-текстом.

---

## Политика комментариев к коду

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

- Комментарии должны объяснять причину, инвариант, бизнес-правило или операционное ограничение.
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
// Увеличить счетчик.
i++
```

```ts
// Вызвать API.
await api.get(...)
```

```go
// Вернуть ошибку.
return err
```

### Хорошие комментарии

```go
// Операторские записи требуют действующую сессию сотрудника и тот же
// client_device_id, который создал сессию. Это не дает переиспользовать
// скопированный session id на другом POS-терминале.
```

```ts
// Финансовые мутации не повторяются автоматически: без idempotency key
// повторный запрос может создать дублирующую оплату или лишний переход
// состояния заказа.
```

```go
// Системные sync endpoints намеренно обходят employee RBAC: они проходят
// аутентификацию как device/runtime flows, а не как кассовые операции.
```

```powershell
# VACUUM — явная maintenance operation. Запуск на каждом старте может
# заблокировать доступность POS и требует дополнительное свободное место
# на больших SQLite databases.
```

---

## Политика UI-текста и i18n

- Пользовательский текст нельзя хардкодить прямо во Vue components, если в соседнем коде нет явного существующего исключения.
- Пользовательские labels, validation messages, errors, modals, dialogs, notifications и empty states должны использовать `vue-i18n`.
- Если проект поддерживает несколько locale, каждое новое пользовательское сообщение должно добавляться во все поддерживаемые locale.
- Бизнес-сообщения об ошибках должны быть понятными, безопасными и подсказывать действие.
- Технические детали нужно писать в логи, а не показывать напрямую в UI.
- Критические или блокирующие бизнес-ошибки нужно показывать через modal/dialog flow, а не только inline red banner.
- Некритичные validation errors могут показываться на уровне поля или inline.
- Backend error responses должны отдавать стабильные error codes/message keys, а не сырые internal errors.
- UI должен сопоставлять backend error codes/message keys с локализованными сообщениями.
- Нельзя показывать в UI raw Go errors, SQL errors, stack traces, exception messages, request dumps, PINs, tokens, secrets или sensitive payloads.

---

## Политика обработки ошибок

Технические ошибки и бизнес-ошибки должны обрабатываться по-разному.

### Политика backend-ошибок

Backend APIs должны возвращать стабильный, типизированный и безопасный error contract:

- стабильный error code;
- HTTP status;
- безопасный user-facing message key;
- опциональные безопасные details;
- correlation/request ID, если он доступен.

Internal causes нужно логировать, а не возвращать пользователю.

Backend должен различать:

- validation errors;
- authentication errors;
- permission errors;
- conflict/state errors;
- rate limit errors;
- infrastructure/database errors;
- unexpected internal errors.

Unexpected errors должны возвращать безопасные `5xx` responses и писать подробные internal logs.

Backend responses не должны включать:

- PIN;
- manager PIN;
- PIN hash;
- tokens;
- secrets;
- full sensitive payloads;
- raw SQL errors;
- raw Go errors;
- stack traces.

### Политика UI-ошибок

UI должен использовать центральный flow нормализации и отображения ошибок.

UI должен различать:

- `400/422` validation errors;
- `401` unauthenticated/revoked session;
- `403` permission denied;
- `404` not found;
- `409` business/state conflict;
- `429` rate limit;
- `5xx` server error;
- network/timeout/backend unavailable.

Обязательное поведение:

- `401` или revoked session должны идти по auth/session recovery policy.
- `403` показывает permission-denied UX без logout, если сама session остается валидной.
- `409` показывает бизнес-понятную причину конфликта.
- `429` показывает подсказку по rate limit.
- Network/backend-unavailable errors сами по себе не должны запускать destructive logout.
- Critical/business-blocking errors должны использовать modal/dialog UX.
- Non-critical validation errors могут использовать field-level или inline messages.

---

## Политика логирования

Backend operations и actions должны использовать structured logging с единым field contract.

Обязательные поля, где применимо:

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

Уровни логов:

- `TRACE` — подробные internal operation steps, выключены по умолчанию.
- `DEBUG` — diagnostic technical context.
- `INFO` — успешные business operations и state transitions.
- `WARN` — ожидаемые rejections и denied requests, включая `4xx`, `403`, `409`, `429`.
- `ERROR` — unexpected failures и infrastructure errors, включая `5xx`.

Никогда не логировать:

- PIN;
- manager PIN;
- PIN hash;
- tokens;
- secrets;
- credentials;
- raw auth payloads с sensitive fields;
- full payment-sensitive payloads.

---

## Политика telemetry для workers

Non-HTTP background workers должны использовать общие telemetry helpers для нормализованных полей:

- `operation`
- `action`
- `result`
- `error_code`
- masked correlation IDs, если они доступны:
  - `node_device_id`
  - `client_device_id`
  - `session_id`
  - `actor_employee_id`

TRACE logs обязательны для внутренних шагов worker lifecycle:

- batch claim;
- process;
- send;
- retry;
- reclaim;
- shutdown.

Временные локальные артефакты, например `test_pipe/`, не являются частью managed runtime path и должны удаляться до merge, если они явно не документированы и не имеют владельца.

---

## Политика безопасности

Разработка должна быть secure-by-default.

Для каждой фичи или изменения учитывать:

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

Security-sensitive decisions должны enforced на backend. UI visibility не является security boundary.

RBAC и session checks должны enforced в application/use-case logic там, где это возможно, а не только в HTTP handlers.

---

## Безопасность финансовых операций

Financial и order state mutations должны считаться high-risk operations.

- Не делать auto-retry payment/order write operations без idempotency key или эквивалентного safety mechanism.
- Не позволять frontend принимать решения о financial state transitions.
- Frontend может отображать состояние и запрашивать действия, но backend владеет business rules и state transitions.
- Payment, precheck, check, order, shift и cash session transitions должны валидироваться backend invariants.
- Duplicate submit protection обязателен в UI для financial mutations.
- Backend всё равно должен защищаться от duplicate или invalid transitions.

---

## Документы и их владельцы

### Этот файл отвечает за

- границы между документами;
- порядок приоритетов при конфликте источников;
- правила поддержки документации;
- правила clean-before-pilot;
- требования к compatibility tails;
- языковая политика;
- политика комментариев к коду;
- базовые политики logging/error/i18n.

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

Architecture docs должны явно разделять:

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

## Политика clean-before-pilot

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

implemented now:

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

## Политика SQLite maintenance

SQLite maintenance должна быть явной и безопасной.

- Не запускать `VACUUM` внутри active write transaction.
- Не запускать `VACUUM` автоматически на каждом старте.
- `VACUUM` — это maintenance/dev/reset operation, а не часть обычного POS write path.
- `VACUUM` может выполняться долго и требовать дополнительное свободное место на диске.
- `VACUUM INTO` можно использовать для compact snapshot/backup flows, когда это уместно.
- Maintenance scripts должны быть явными, Windows-compatible и safe by default.
- Потенциально destructive maintenance commands должны требовать явное намерение оператора.
- SQLite maintenance policy должна быть документирована в `docs/backend/POS-DATA-AND-MIGRATIONS.md`.

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

## Политика DDD bounded contexts

Target architecture для POS Core — modular monolith с явными bounded contexts.

Архитектура может позже развиваться в сторону отдельных services, но текущая реализация должна избегать premature service extraction.

Правила:

- Не создавать один generic `POSContext`, который владеет всем.
- Каждый bounded context должен иметь ясное domain ownership.
- Cross-context communication должно происходить через contracts, APIs или events.
- Избегать прямой cross-context database coupling.
- Architecture docs должны различать current implementation и target architecture.
- Не выполнять большой package refactor только ради соответствия target DDD docs, если задача явно этого не требует.

Основные целевые contexts:

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

## Граница ответственности UI/backend

Backend является source of truth для:

- business rules;
- state transitions;
- permission enforcement;
- financial calculations;
- order/precheck/payment/check lifecycle;
- shift/cash session lifecycle;
- sync/outbox behavior.

Frontend отвечает за:

- rendering state;
- collecting user intent;
- local UI state;
- client identity storage;
- user-friendly error display;
- permission-based visibility только как UX.

Frontend не должен:

- bypass backend invariants;
- принимать final permission decisions;
- рассчитывать final financial state;
- самостоятельно решать order/payment/check transitions;
- считать hidden/disabled UI security enforcement.

---

## Политика RBAC

RBAC должен enforced на backend.

- UI visibility — это только UX, не security.
- Business/operator write operations требуют active employee session там, где применимо.
- Operator/business flows должны валидировать:
  - session;
  - actor employee;
  - client_device_id;
  - required permission.
- System/device flows могут обходить employee session только если это явно спроектировано и документировано.
- Permission IDs должны приходить из canonical permission catalog.
- Избегать ad-hoc permission strings в runtime code.
- Role matrices должны быть документированы и покрыты тестами.
- Permission denied events нужно audit log без sensitive data.

Canonical roles должны включать:

- `cashier`
- `senior_cashier`
- `waiter`
- `manager`
- `kitchen`
- `support_admin`

---

## Политика local dev scripts

Scripts должны быть предсказуемыми и безопасными.

Для PowerShell scripts:

- использовать `$PSScriptRoot` для script-relative paths;
- использовать UTF-8-aware reading/writing;
- писать полезные logs для background services;
- писать PID files для запущенных background processes;
- предоставлять stop scripts или ясные stop instructions;
- показывать tail logs при падении health checks;
- избегать тихого запуска duplicate services;
- проверять required ports перед startup.

Для `.cmd` scripts:

- использовать `set "VAR=value"`;
- использовать quoted paths;
- использовать `%~dp0` для script-relative paths;
- использовать `call` при вызове другого `.cmd`, `npm` или batch-style command;
- использовать `exit /b 1` для errors;
- не использовать PowerShell syntax вроде `$env:VAR="..."`.

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

## Обязательные проверки

Когда применимо, запускать:

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

Если `package.json` содержит lint/typecheck/test scripts, запускать их, когда они относятся к изменению.

Если check нельзя выполнить в текущей среде, final report должен указать:

- какой check не был выполнен;
- почему он не был выполнен;
- что было проверено вместо него.

---

## Требования к финальному отчету agent tasks

Каждая существенная agent task должна завершаться отчетом, содержащим:

- что найдено;
- что изменено;
- какие файлы изменены;
- какие tests/checks запущены;
- какие checks не удалось запустить и почему;
- remaining risks;
- `planned next`;
- `out of scope`.

Для hardening tasks добавлять отдельные sections:

- RBAC status;
- error handling status;
- UI/backend contract status;
- logging/audit status;
- i18n status;
- documentation sync status;
- SQLite maintenance status, если применимо;
- local scripts/smoke status, если применимо.
