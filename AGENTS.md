# AGENTS.md

## Назначение

Этот файл задает правила работы с документацией и архитектурными границами репозитория `mh-pos`.

Цель проста:

- не смешивать в одном документе текущий runtime, целевую архитектуру и план работ;
- не тащить compatibility-хвосты до первого пилота;
- не допускать расхождения между кодом, тестами и документацией.

Проект еще не имеет production-эксплуатации. До первого пилота действуют правила first-launch development, а не legacy-support development.

## Правила агентской кодогенерации

- Все ответы пользователю, комментарии в коде, логи, тексты ошибок и новые документы пишутся на русском языке.
- Нельзя придумывать детали реализации внешних систем, бизнес-правил и неоднозначных требований. Если фактов нет в коде, документации или явном ответе пользователя, нужно задать уточняющий вопрос.
- Комментарии в коде должны описывать только текущую функциональность. Не использовать пометки вроде "добавлено", "обновлено", "временно", если это не часть реального runtime-смысла.
- При работе в PowerShell принудительно включать UTF-8 для вывода и чтения, например `$env:PYTHONIOENCODING='utf-8'`, чтобы кириллица не превращалась в mojibake.
- Для чтения и анализа файлов в этом репозитории использовать Python-скрипты с `encoding='utf-8'`. Для массового поиска можно вызывать быстрые инструменты из Python, но итоговое чтение файлов должно сохранять UTF-8.
- Все файлы репозитория (код, тесты, документация, миграции, конфиги, скрипты) сохраняются исключительно в UTF-8. Использование любых других кодировок запрещено.
- Не откатывать чужие изменения в рабочем дереве. Если найден dirty state, сначала понять, относится ли он к задаче.
- При разработке Go-кода использовать скилл `go-modern-guidelines`.
- При разработке UI использовать доступный инструмент Playwright для проверки изменений UI.
- При разработке кода использовать secure-by-default подход: максимально учитывать риски CSRF, XSS, injection, broken access control и другие классы атак из OWASP Top 10; при проектировании/изменении фич оценивать severity по CVSS 4.0 и снижать риск до приемлемого уровня до merge.
- Валютный каталог должен опираться на полный active ISO 4217 список (не локальный минимальный subset). Особенно обязательно покрытие валют ЮВА (`IDR`, `THB`, `VND`, `MYR`, `SGD`, `PHP` и др.).
- Изменения валютного каталога всегда выполняются end-to-end: backend validation, cloud reference/template, UI precision/formatting, тесты и профильная документация в одном PR.

## Документы и их владельцы

### Этот файл отвечает за

- границы между документами;
- порядок приоритетов при конфликте источников;
- правила поддержки документации;
- правила clean-before-pilot;
- требования к compatibility tails.

### Этот файл не отвечает за

- полный перечень HTTP endpoints;
- полный перечень экранов UI;
- полное описание схемы БД;
- детальный roadmap по стадиям.

Для этого существуют отдельные документы.

## Карта источников истины

### Код и тесты

Код и тесты — источник истины для того, что **реально реализовано сейчас**.

Если документация утверждает одно, а runtime и тесты делают другое, то для описания текущего состояния приоритет у кода и тестов.
- Для всех новых/измененных экспортируемых сущностей добавь GoDoc/TS doc comments.
- Для нетривиальной логики добавь краткие объясняющие комментарии “почему”, не “что”.
- Избегай шума и очевидных комментариев.
- Используй средства логгирования операций и действий, для анализа проблем.
- максимально описывать методы, свойства и переменные , что выполняется, получаемый результат, для метода описывать входные и выходные данные, для переменных давать комментарий, что за переменная

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
- требований к БД.

### Roadmap

`ROADMAP.md` — источник истины для:

- what is done / next / blocked;
- рисков;
- pilot gates;
- sequencing.

### Local E2E Prototype Quickstart

`README.md` — источник истины для локального запуска `pos-ui -> pos-backend -> cloud-backend`, demo bootstrap и smoke scripts. Dev bootstrap endpoint `POST /api/v1/dev/bootstrap-demo` относится только к local/dev режиму и требует `POS_DEV_TOOLS=1`.

## Порядок приоритета при конфликте

Если есть конфликт между документами, использовать такой порядок:

1. Безопасность и архитектурные инварианты: `SPECv1.3.md`.
2. Текущее фактическое поведение: код и тесты.
3. UI / backend surface contracts: профильные документы в `docs/ui/` и `docs/backend/`.
4. План выполнения и статусы: `ROADMAP.md`.
5. `README.md` — только обзорный документ, не финальный арбитр архитектуры.

## Clean-before-pilot policy

До первого пилота запрещено:

- поддерживать старое поведение "на всякий случай", если его не существует в production;
- добавлять dual-write;
- добавлять исторические DB migrations ради обратной совместимости dev-схем;
- тащить legacy aliases без владельца и срока удаления;
- документировать unsupported future behavior как будто оно уже доступно.

Разрешено:

- переписывать canonical first-launch schema;
- удалять deprecated transport/API tails;
- пересоздавать dev/test БД;
- переименовывать сущности и endpoints так, чтобы модель становилась чище.

## Политика схемы БД до первого пилота

До первого пилота SQLite развивается по правилу:

- один канонический `001_init.sql`;
- никакой обязательной исторической цепочки `002`, `003`, `004` для локального runtime;
- любое изменение схемы до пилота делается через обновление canonical init;
- dev/test databases регенерируются с нуля.

Если потребуется редкий временный migration script для локальной разработки, он не становится частью canonical pilot path без отдельного архитектурного решения.

### Политика версий модулей и миграций БД (implemented now)

- Все runtime-модули используют единую версию продукта через `MH_POS_VERSION` (fallback: `0.1.0`).
- Контекст БД включает любую используемую БД решения (`SQLite` и `PostgreSQL`).
- Любые миграции/обновления схемы выполняются программно при старте модуля, ручной ad-hoc upgrade path не считается canonical.
- Каждая БД обязана иметь runtime-таблицу версий модулей (`db_runtime_versions`).
- Если модуль видит, что версия БД ниже версии модуля, он обязан:
  - сначала выполнить backup текущего состояния;
  - только после backup запускать schema upgrade / apply canonical migration;
  - после успешного обновления записать новую версию модуля в таблицу версий.

## Политика compatibility tails

Любой compatibility tail допустим только если у него есть:

- владелец;
- причина существования;
- срок удаления или milestone удаления;
- тест, который подтверждает текущее поведение;
- запись в профильной спецификации backend/UI.

Если у compatibility tail нет срока удаления, он не должен быть merged.

## Что считается compatibility tail

Примеры:

- deprecated alias endpoint;
- legacy transport field;
- старое enum name;
- временный adapter слой между old/new payload;
- документация со старым названием сущности, оставленная "ради привычки".

## Правило синхронного обновления документации

Любой PR, который меняет одно из следующих, обязан менять профильную документацию в том же PR:

- HTTP routes or payloads;
- UI screens or user flows;
- permission model;
- DB schema / invariants;
- sync event catalog;
- migration/reset policy.

## Минимальный чек перед merge

Перед merge проверить:

- тесты не спорят с документацией;
- `README.md` не обещает лишнего;
- профильный документ обновлен;
- deprecated tails либо удалены, либо помечены с kill date;
- документация отделяет `implemented now` от `planned next` и `out of scope`.

## Правило формулировок

В документации использовать только три формулировки статуса:

- `implemented now`
- `planned next`
- `out of scope`

Формулировки вида:

- `temporary for now`
- `legacy but maybe keep`
- `later we will see`

не использовать.

## Logging policy (implemented now)

Для backend операций и действий использовать structured logging с единым контрактом полей:

- `request_id`
- `operation`
- `action`
- `result`
- `duration_ms`
- `error_code`
- `node_device_id` (masked)
- `client_device_id` (masked)
- `session_id` (masked)
- `actor_employee_id` (masked)

Уровни логирования:

- `TRACE` — детальные внутренние шаги операции (выключено по умолчанию).
- `DEBUG` — диагностический технический контекст.
- `INFO` — успешные бизнес-операции и state transitions.
- `WARN` — ожидаемые отклонения и отклоненные запросы (`4xx`, включая `403/409/429`).
- `ERROR` — неожиданные сбои и инфраструктурные ошибки (`5xx`).

Запрещено логировать:

- PIN;
- manager PIN;
- hash PIN;
- raw auth payload с чувствительными полями.

### Worker telemetry policy (implemented now)

- Non-HTTP background workers must use shared telemetry helper for normalized fields:
  - `operation`
  - `action`
  - `result`
  - `error_code`
  - masked correlation ids when available (`node_device_id`, `client_device_id`, `session_id`, `actor_employee_id`).
- TRACE logs are required for worker lifecycle internal steps (batch claim/process/send/retry/reclaim).
- Temporary local artifacts (example: `test_pipe/`) are not part of managed runtime path and must be removed before merge unless explicitly documented and owned.
