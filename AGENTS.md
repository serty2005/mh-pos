# AGENTS.md

## Назначение

Этот файл задает правила работы Codex, агентов и инструментов кодогенерации в репозитории `mh-pos`.

`AGENTS.md` является только рабочей инструкцией для разработки. Он не является источником архитектуры, бизнес-спецификации, DDD-модели, roadmap или детального описания домена.

## Язык

- Ответы пользователю по умолчанию пишутся на русском языке.
- Промпты для Codex и агентов по умолчанию пишутся на русском языке.
- Документация проекта пишется на русском языке.
- Комментарии в коде пишутся на русском языке.
- Идентификаторы в коде пишутся на английском языке: packages, modules, functions, methods, variables, routes, payload fields, error codes, permission IDs, event names, test names.
- Имена структурированных полей логов пишутся на английском языке.
- Английский допускается для технических идентификаторов, команд, API/SDK-терминов и точных внешних цитат.
- Русский пользовательский текст допускается только в `ru` locale files, документации, комментариях к коду и прямом общении с оператором.
- Нельзя добавлять русские hardcoded UI strings в source code, кроме `ru` locale files.
- Пользовательские UI-сообщения должны идти через i18n.

## UTF-8 и PowerShell

- Все файлы репозитория сохраняются только в UTF-8.
- При работе в PowerShell включать UTF-8 для вывода и чтения:

```powershell
$env:PYTHONIOENCODING='utf-8'
```

- Для чтения и анализа файлов в этом репозитории использовать Python-скрипты с `encoding='utf-8'`.
- Для массового поиска можно использовать `rg`; итоговое чтение файлов должно сохранять UTF-8.
- Файлы PowerShell (`*.ps1`) также сохраняются в UTF-8; текстовые сообщения в скриптах рекомендуется держать на английском для кросс-платформенной диагностики.


## Безопасная работа с репозиторием

- Перед изменениями перечитать фактический код и профильные документы, относящиеся к задаче.
- Не откатывать чужие изменения в рабочем дереве.
- Если найден dirty state, сначала понять, относится ли он к задаче.
- Не использовать `git reset --hard`, `git checkout --` и другие destructive operations без явного запроса пользователя.
- Не делать большой runtime refactor только ради приведения документации к целевой архитектуре.
- Если документация описывает unsupported behavior как реализованное, исправлять документацию, а не дописывать фичу без отдельного требования.
- До первого пилота не добавлять compatibility tails без владельца, причины, срока удаления, теста и записи в профильной спецификации.

## Источники истины

- `SPECv1.3.md` — архитектура, бизнес-инварианты, DDD/context boundaries и ADR.
- `ROADMAP.md` — прогресс, следующий план, блокеры и задачи после пилота.
- `docs/backend/*` — backend contracts, state transitions, error contract, schema и migration policy.
- `docs/ui/*` — UI contracts, пользовательские сценарии и UI RBAC.
- `docs/sync/*` — sync contracts и directional ownership.
- `docs/architecture/*` — context map, dependency direction и modular monolith rules.
- Код и тесты — источник истины для фактически реализованного runtime.

Если документы конфликтуют с кодом и тестами, сначала зафиксировать фактическое поведение по коду, затем обновить устаревший документ.

## Документация

- Любой PR, который меняет HTTP routes, payloads, UI flows, permission model, DB schema, sync event catalog, error contract, logging contract, migration/reset policy или startup/smoke scripts, обязан обновлять профильную документацию в том же PR.
- В документации использовать русские человекочитаемые статусы:
  - `реализовано сейчас`;
  - `запланировано далее`;
  - `вне текущего объема`.
- В `ROADMAP.md` использовать русские статусы:
  - `выполнено`;
  - `в работе`;
  - `заблокировано`;
  - `далее`;
  - `после пилота`.
- Английские значения статусов допустимы только как машинно-читаемые значения, enum values, labels или цитаты из кода.
- Не документировать будущую или неподдерживаемую функциональность как текущую.
- Реализация UUID только UUID v7

## Миграции и версии БД

- Все изменения схемы SQLite/PostgreSQL выполняются программно при старте runtime-модуля; ручной ad-hoc SQL не является canonical path.
- Active pre-pilot path использует один managed SQL baseline `001_init.sql` на runtime-модуль, описанный в `docs/backend/POS-DATA-AND-MIGRATIONS.md`; существующие dev/test БД до первого клиента пересоздаются, а реальные data-preserving migrations начинаются после первого внедрения.
- Версия и состояние схемы/данных фиксируются в `db_runtime_versions`; изменение active SQL file выполняется через повышение `MH_POS_VERSION` и программный startup upgrade.
- Каждая БД должна иметь `db_runtime_versions`; если таблица отсутствует, модуль считает БД самой старой и запускает upgrade path.
- `schema_migrations` должна хранить active SQL file, checksum, status и время применения, если это поддержано текущим модулем.
- Перед safe schema/data upgrade существующей БД обязателен backup: SQLite `.db/.db-wal/.db-shm`, PostgreSQL безопасный snapshot/hook по текущей реализации.
- После migrations обязательна schema verification критичных tables/columns/indexes до запуска HTTP server, workers и runtime access к business tables.
- `DB version > MH_POS_VERSION` должен завершать startup fail-fast; downgrade не поддерживается.
- Любое изменение managed SQL file, таблиц/колонок/индексов/constraints или загрузки master/reference/configuration данных должно иметь тесты, backup behavior и обновление профильной документации в том же PR.
- UI/admin операция очистки SQLite допустима только как destructive-by-design flow: backup до очистки, явное подтверждение, RBAC/support permission, audit log и документированный rebootstrap/restart path.

## Комментарии к коду

Комментарии обязательны для нетривиального кода, но не должны пересказывать очевидные строки.

Комментировать:

- экспортируемые Go types, interfaces, funcs, methods и constants, если они являются частью package API;
- публичные TypeScript/Vue helpers, composables, stores, API clients и shared models;
- нетривиальные бизнес-правила;
- state transitions для order, precheck, payment, check, shift и cash session;
- RBAC и security-sensitive решения;
- auth/session/client_device_id validation;
- manager override rules;
- safe error responses, correlation/request ID handling и error mapping;
- retry/no-retry policy, особенно для financial mutations;
- sync/outbox/inbox/retry/reclaim logic;
- SQLite transaction boundaries и maintenance operations;
- startup/stop scripts с неочевидным Windows, PowerShell, Docker, PID, process или log behavior;
- тестовые fixtures, если без комментария непонятно, какое правило они проверяют.

Комментарии должны описывать текущий инвариант, причину или операционное ограничение. Не использовать исторические пометки вроде `added`, `updated`, `temporary`, `for now`, если это не часть реального runtime-смысла.

## Go

- Следовать современным Go practices.
- Использовать маленькие interfaces.
- Делать context-aware APIs там, где это уместно.
- Писать table-driven tests.
- Использовать typed/sentinel errors там, где это полезно.
- Не использовать panics в request path.
- Выполнять `gofmt`.
- После изменений Go-кода выполнять:

```powershell
cd pos-backend
go mod tidy
go test ./...
```

```powershell
cd cloud-backend
go mod tidy
go test ./...
```

## TypeScript, Vue и UI

- Пользовательские labels, validation messages, errors, modals, dialogs, notifications и empty states должны идти через `vue-i18n`.
- Если поддерживается несколько locale, новое пользовательское сообщение добавлять во все поддерживаемые locale.
- UI visibility является UX-слоем, а не security boundary.
- Backend RBAC и application-layer checks являются авторитетными.
- При разработке UI использовать Playwright для проверки изменений, если это возможно в текущей среде.
- После изменений UI выполнять:

```powershell
cd pos-ui-g
npm install
npm run build
```

```powershell
cd cloud-ui-g
npm install
npm run build
```

## Ошибки, логи и безопасность

- Backend API должны возвращать стабильный и безопасный error contract: error code, HTTP status, безопасный message key, безопасные details и correlation/request ID, если он доступен.
- Internal causes логируются, но не возвращаются пользователю.
- UI не должен показывать raw Go errors, SQL errors, stack traces, exception messages, request dumps, PINs, tokens, secrets или sensitive payloads.
- Логи должны быть структурированными.
- Поля логов пишутся на английском языке.
- Использовать стабильный `error_code` / `message_key`.
- Не логировать PIN, manager PIN, PIN hash, tokens, secrets, credentials, raw auth payloads с sensitive fields и payment-sensitive payloads.
- Разработка ведется secure-by-default с учетом OWASP Top 10: CSRF, XSS, injection, broken access control, insecure deserialization, session fixation/reuse, sensitive data exposure, unsafe logging и replay/double-submit risks.
- Financial и order state mutations считаются high-risk operations.
- Не делать auto-retry financial mutations без idempotency key или эквивалентного safety mechanism.
- Frontend не принимает авторитетные решения о financial state transitions.

## Проверки перед финальным отчетом

Если окружение позволяет, выполнить:

```powershell
cd pos-backend
go mod tidy
go test ./...
```

```powershell
cd cloud-backend
go mod tidy
go test ./...
```

```powershell
cd pos-ui-g
npm install
npm run build
```

```powershell
cd cloud-ui-g
npm install
npm run build
```

Профильные поиски для документационных задач:

```powershell
rg "Order.*Check.*Payment|Check.*Payment|payment.*check_id|check_id.*payment" .
rg "business_date_local|reprint|повторн.*печать|print snapshot|item-level ACK|batch ACK" SPECv1.3.md ROADMAP.md docs
rg "bounded context|DDD|POSContext|Organization|Catalog|Pricing|Inventory" AGENTS.md SPECv1.3.md ROADMAP.md docs
rg "future|later|maybe|probably|temporary for now|for now" AGENTS.md SPECv1.3.md ROADMAP.md docs
rg "implemented now|planned next|out of scope|Current status|Business rules|Architecture decisions|Pilot blockers|Context owns|Remaining risks" AGENTS.md SPECv1.3.md ROADMAP.md docs
```

Если найденные строки являются кодовыми идентификаторами, enum values, API fields, event names или цитатами из кода, их можно оставить с русским пояснением рядом. Если это обычный текст документации, заменить на русский.

## Финальный отчет

Финальный отчет писать на русском языке.

Кратко указать:

- что найдено;
- что изменено;
- измененные файлы;
- какие проверки запущены;
- какие проверки не удалось запустить;
- оставшиеся риски;
- что запланировано далее;
- что вне текущего объема;
- затрагивался ли runtime code.
- по задаче всегда давать краткий и полный комментарии о выполненых работах
