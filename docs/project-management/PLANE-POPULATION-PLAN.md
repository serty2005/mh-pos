# План заселения Plane

Статус: план переноса прогресса разработки `mh-pos` в Plane для последовательного выполнения через Codex.

Дата аудита: 2026-06-10.

Документ описывает, как перенести текущий прогресс проекта в Plane без потери фактического состояния, архитектурных границ и уже закрытой работы. Он является рабочим runbook для агентов и владельца проекта. Источником истины по runtime остаются код, тесты и профильные документы.

## Цель

Перевести управление разработкой `mh-pos` в Plane так, чтобы каждая дальнейшая работа выполнялась по конкретному work item, а Codex мог последовательно брать задачи по Plane identifier, читать контекст, выполнять изменения, запускать проверки и возвращать результат в Plane.

Итоговое состояние:

- Plane содержит карту всех модулей проекта;
- текущий реализованный прогресс зафиксирован baseline-задачами;
- открытые roadmap gaps разложены на атомарные work items;
- задачи имеют единый стандарт описания и критериев приемки;
- агентский workflow работает через `Ready -> In Progress -> Review`;
- `Done` выставляет владелец после review/merge/приемки;
- Git-документация не заменяется Plane, а Plane ссылается на профильные документы.

## Текущий Plane-контур

Реализовано сейчас:

- Workspace: `myhoreca-pos`.
- Project: `POS`.
- Project id: `562fe804-ecc3-41df-b85d-c981e6c13760`.
- Фактический Plane identifier проекта: `POS`.
- Включены modules, cycles, views, pages и intake.
- Work item types через MCP сейчас недоступны, поэтому базовый процесс опирается на modules, parent/child work items, labels, states, links и comments.
- Созданы кастомные states:
  - `Specified`;
  - `Ready`;
  - `Blocked`;
  - `Review`;
  - `Validation`.
- Созданы labels для backend/frontend/database/api/security/tests/docs/event-contract/research/agent-ready/manual-validation и других типов работ.
- Создан cycle `2026-06 — Plane Bootstrap`.
- Созданы первые work items вокруг Plane bootstrap и `Storage and Archiving`.

Запланировано далее:

- Привести примерные упоминания `MHPOS-42` в локальных инструкциях к фактическому идентификатору `POS-N` или явно оставить их как пример.
- Заполнить Plane baseline-задачами по всем модулям.
- Перенести незакрытые задачи из `ROADMAP.md`, `SPECv1.3.md`, `docs/CURRENT-FUNCTIONAL-STATE.md` и профильных документов.

## Источники для переноса

Использовать в таком порядке:

1. `README.md` - краткая карта репозитория и текущее состояние.
2. `docs/CURRENT-FUNCTIONAL-STATE.md` - фактическая сводка runtime.
3. `ROADMAP.md` - выполненное, в работе, далее, после пилота.
4. `SPECv1.3.md` - frozen runtime contract и full pilot target.
5. `docs/architecture/DDD-CONTEXT-MAP.md` - bounded contexts и ownership.
6. `docs/backend/*` - backend contracts, data, migrations, errors, runtime config.
7. `docs/ui/*` - UI contracts, RBAC, UX rules.
8. `docs/sync/*` - sync ownership и Edge/Cloud contracts.
9. Код, миграции и тесты - финальная проверка фактической реализации.
10. `tools/plane-mcp/README.md` - правила работы агента с Plane.

Если документ и код расходятся, сначала фиксировать фактическое поведение по коду/тестам, затем создавать задачу на обновление документации или runtime, не смешивая эти работы без причины.

## Что хранится в Plane

Plane хранит:

- атомарные задачи разработки;
- baseline-аудиты модулей;
- зависимости между задачами;
- текущий статус работы;
- ссылки на документы, файлы и PR;
- итоговые комментарии агентов;
- остаточные риски;
- ручные validation tasks;
- blocked reasons.

Plane не хранит как источник истины:

- архитектурный contract вместо `SPECv1.3.md`;
- backend/UI/sync спецификации вместо `docs/*`;
- бизнес-инварианты без ссылки на код или документ;
- secrets, tokens, raw payloads, PIN, request dumps;
- большие фрагменты кода;
- неподтвержденную будущую функциональность как текущую.

## Статусы

Использовать единый flow:

```text
Backlog -> Specified -> Ready -> In Progress -> Review -> Validation -> Done
```

Правила:

- `Backlog`: идея, gap или риск без полной спецификации.
- `Specified`: требования описаны, но есть открытые решения, зависимости или не хватает acceptance.
- `Ready`: задача готова для Codex; scope, критерии приемки, документы и проверки определены.
- `In Progress`: агент или разработчик выполняет задачу.
- `Blocked`: работа остановлена внешней зависимостью или решением владельца.
- `Review`: изменения готовы, проверки выполнены, нужен review.
- `Validation`: требуется ручная, продуктовая, интеграционная или smoke-проверка.
- `Done`: выставляет владелец после review/merge/приемки.
- `Cancelled`: задача отменена как неактуальная или заменена другой задачей.

Агент может переводить только назначенную задачу:

- `Ready -> In Progress`;
- `In Progress -> Review`;
- при реальном блокере: `In Progress -> Blocked` с комментарием.

Агент не переводит задачу в `Done`.

## Labels

Использовать labels как тип изменения и риск:

- `backend` - Go backend logic.
- `frontend` - UI.
- `database` - schema, queries, persistence.
- `migration` - managed SQL, runtime version, upgrade behavior.
- `api` - HTTP routes, payloads, OpenAPI.
- `event-contract` - sync events, envelopes, outbox/inbox.
- `security` - auth, RBAC, safe errors, audit.
- `performance` - latency, large DB, resource usage.
- `documentation` - specs, README, docs.
- `tests` - unit/integration/e2e/smoke.
- `technical-debt` - debt with explicit owner/reason.
- `bug` - confirmed behavior regression or mismatch.
- `research` - investigation without mandatory implementation.
- `agent-ready` - task is safe for local Codex execution.
- `manual-validation` - owner/operator must manually verify.
- `blocked-external` - blocked by environment, owner decision or external system.
- `breaking-change` - incompatible API/behavior/schema change.

Не создавать новые labels автоматически без отдельного решения владельца.

## Modules

Plane modules должны отражать доменные и технические ownership boundaries. Текущий список уже достаточен для переноса:

- `POS Core`;
- `POS Cashier UI`;
- `Waiter`;
- `Orders`;
- `Pricing`;
- `Payments`;
- `Fiscalization`;
- `Shifts and Business Day`;
- `KDS`;
- `Catalog and Menu`;
- `Recipes`;
- `Inventory`;
- `Costing`;
- `Stop Lists`;
- `Edge-Cloud Synchronization`;
- `Cloud Backoffice`;
- `Reporting and OLAP`;
- `Storage and Archiving`;
- `Licensing and Pairing`;
- `Platform and Infrastructure`;
- `Security and RBAC`;
- `Testing and Quality`;
- `Documentation`.

Правило: один work item может лежать в одном основном module. Если задача пересекает несколько областей, выбрать module по главному ownership, а смежные области указать labels и relations.

## Work Item Standard

Каждый work item, который можно отдавать Codex, должен иметь структуру:

```text
Цель
Коротко, какой результат должен быть достигнут.

Пользовательский или операционный сценарий
Кто выполняет действие и зачем.

Текущее поведение
Что уже есть сейчас. Указать подтверждение кодом, тестами или документом.

Требуемое поведение
Что должно измениться.

Входит в scope
Четкие границы.

Не входит в scope
Что агент не должен делать.

Архитектурные инварианты
Критичные правила из SPEC/DDD/backend/ui/sync docs.

Критерии приемки
Проверяемые пункты.

Требуемые тесты
Какие unit/integration/e2e/smoke/build проверки нужны.

Документы
Какие docs обновить, если меняются контракты.

Риски
Оставшиеся риски и ручная validation.

Зависимости
Связанные Plane tasks или решения владельца.

Подтверждение из репозитория
Файлы, символы, тесты, docs.
```

Для baseline work item допустимо вместо реализации описывать:

- `реализовано сейчас`;
- `запланировано далее`;
- `вне текущего объема`;
- `оставшиеся риски`;
- `следующие задачи`.

## Agent Workflow

Codex-прогон по одной задаче:

1. Получить Plane identifier от пользователя, например `POS-8`.
2. Прочитать задачу через `retrieve_work_item_by_identifier`.
3. Проверить state, module, cycle, labels, description, comments, links и relations.
4. Прочитать `AGENTS.md`.
5. Прочитать профильные документы по module.
6. Проверить `git status`.
7. Если задача в `Ready`, перевести ее в `In Progress`.
8. Выполнить работу строго в scope.
9. Запустить профильные проверки.
10. Обновить профильные docs, если менялись routes, payloads, UI flows, RBAC, schema, sync events, errors, logs, migrations или smoke scripts.
11. Добавить итоговый Plane comment.
12. Перевести задачу в `Review`.

Итоговый Plane comment:

```text
Выполнено:
- ...

Измененные файлы:
- ...

Проверки:
- ...

Не запускалось:
- ...

Оставшиеся риски:
- ...

Вне scope:
- ...

Runtime code:
- да/нет, какие области
```

Запрещено агенту:

- брать задачу без Plane identifier;
- расширять scope без согласования;
- переводить в `Done`;
- выполнять destructive Plane operations;
- удалять чужие изменения;
- писать secrets или raw sensitive payloads в Plane comments;
- менять runtime code при документационной задаче, если это явно запрещено.

## Этап 0. Bootstrap Plane Process

Цель: закрепить сам процесс до массового переноса.

Создать или завершить задачи:

1. `Plane Bootstrap — document project management structure`
   - Module: `Documentation`.
   - Labels: `documentation`, `research`, `agent-ready`.
   - State после подготовки: `Review`.
   - Результат: документы в `docs/project-management/*`.

2. `Plane Bootstrap — align local MCP README with POS identifier`
   - Module: `Documentation`.
   - Labels: `documentation`.
   - Scope: уточнить, что фактический project identifier сейчас `POS`, а `MHPOS-42` является примером, если identifier не меняется в Plane.

3. `Plane Bootstrap — validate read/write smoke on dedicated MCP smoke test work item`
   - Module: `Platform and Infrastructure`.
   - Labels: `manual-validation`, `blocked-external`.
   - Scope: выполнять write smoke-test только на отдельной тестовой задаче.

Definition of Done этапа:

- агентский workflow зафиксирован;
- формат work item принят;
- baseline-задачи можно создавать пачками без изменения runtime-кода;
- owner подтвердил, что identifier `POS-N` подходит или отдельно поменял project identifier в Plane.

## Этап 1. Baseline По Всем Модулям

Цель: перенести уже выполненный прогресс как проверенный baseline, чтобы агенты не переоткрывали закрытые части.

Для каждого module создать parent work item:

```text
<Module> — verified implementation baseline
```

State:

- `Validation`, если baseline требует ручного подтверждения владельца;
- `Done`, если владелец после review подтверждает перенос;
- не ставить `Done` агентом.

Минимальное содержимое baseline:

- реализовано сейчас;
- запланировано далее;
- вне текущего объема;
- подтверждение кодом;
- подтверждение тестами;
- связанные документы;
- риски;
- рекомендуемые дочерние задачи.

Порядок baseline-аудита:

1. `POS Core`
   - Источники: `pos-backend/internal/pos`, `docs/backend/POS-BACKEND-SPEC.md`, `SPECv1.3.md`.
   - Зафиксировать auth/session/device context, safe errors, local runtime boundaries.

2. `Orders`
   - Источники: order/precheck/check services, `SPECv1.3.md`.
   - Зафиксировать `Order -> Precheck -> Payment -> Check`, modifiers, closed orders bounded reads.

3. `Payments`
   - Источники: check/payment/financial operations code и docs.
   - Зафиксировать precheck-based payments, partial payments, immutable checks, cancellation/refund ledger.

4. `Shifts and Business Day`
   - Зафиксировать personal shifts, cash sessions, business date, boundaries cancellation/refund.

5. `Pricing`
   - Зафиксировать discounts/surcharges, tax-last, integer rounding, pricing_policy ingest.

6. `Catalog and Menu`
   - Зафиксировать Cloud-owned master data, services, folders/tags, modifiers, menu publication.

7. `KDS`
   - Зафиксировать ticket lifecycle, `KitchenTicketStatusChanged`, `ItemServed`, stock input/proposal routes.

8. `Waiter`
   - Зафиксировать `/pos/waiter`, mobile-first flow, отсутствие payment/refund/cash drawer authority.

9. `POS Cashier UI`
   - Зафиксировать active React/Vite `pos-ui-g` и legacy Vue `pos-ui` статус, cashier shell, activity, refunds, shared UI.

10. `Edge-Cloud Synchronization`
    - Зафиксировать master-data ingest, outbox, exchange, ACK/retry, current/legacy event contracts.

11. `Inventory`
    - Зафиксировать Cloud-centric inventory foundation, Worker ledger rows, stock balances read model.

12. `Recipes`
    - Зафиксировать read-only Edge recipes, Cloud authoring/review foundation, recipe expansion gaps.

13. `Stop Lists`
    - Зафиксировать sale blocking, Edge overlay, Cloud review, sync readiness.

14. `Costing`
    - Зафиксировать pilot costing fields и то, что full costing/retro DAG не реализованы.

15. `Reporting and OLAP`
    - Зафиксировать ClickHouse raw events, olap_stock_moves, bounded summaries, backfill foundation.

16. `Cloud Backoffice`
   - Зафиксировать active `cloud-ui-g`, удаление legacy `cloud-ui` и migration gaps.

17. `Storage and Archiving`
    - Уже начато. Довести baseline `POS-2` и дочерние задачи.

18. `Licensing and Pairing`
    - Зафиксировать license server, pairing code, provisioning routes.

19. `Security and RBAC`
    - Зафиксировать backend authority, UI visibility non-security boundary, manager override, safe errors/logging.

20. `Testing and Quality`
    - Зафиксировать Go tests, UI builds, Playwright, seed smoke, Docker smoke.

21. `Platform and Infrastructure`
    - Зафиксировать Docker compose, local ports, runtime config, buildx blocker.

22. `Documentation`
    - Зафиксировать docs ownership и правила обновления.

23. `Fiscalization`
    - Зафиксировать как backlog/post-pilot или architecture-only до отдельного решения.

Definition of Done этапа:

- у каждого module есть baseline work item;
- текущий runtime не смешан с будущими планами;
- для каждого module есть список дочерних задач или явно указано, что новых задач нет;
- все baseline work items связаны с профильными документами.

## Этап 2. Перенос Открытых Roadmap Gaps

Цель: превратить незакрытые пункты в атомарные задачи.

Правило нарезки:

- одна задача - один contract или один UI flow или один backend slice;
- если нужно менять backend + UI + docs + smoke, parent work item должен иметь отдельные child work items;
- если задача требует решения владельца, она остается `Specified` или `Blocked`, а не `Ready`;
- research-задачи не должны тихо превращаться в реализацию.

Основные parent work items:

1. `Cloud Backoffice — migrate legacy manager scenarios to cloud-ui-g`
   - Children:
     - inventory stock balances screen;
     - stock ledger screen;
     - OLAP export status read-only screen;
     - stock moves and summaries;
     - sales/kitchen summary;
     - kitchen timing summary;
     - proposal review queues;
     - recipe version editor/review;
     - stop-list review polish.

2. `Inventory — full Cloud Inventory Engine`
   - Children:
     - materialized balances design;
     - production-grade stock receipts/counts/production state;
     - sale consumption with recipe expansion;
     - refund/cancellation disposition processing;
     - modifier-linked consumption;
     - negative balance costing policy;
     - retro recalculation DAG;
     - fault-injection reconnect/outbox ACK smoke.

3. `Costing — costing status and recalculation foundation`
   - Children:
     - costing status state machine;
     - cost basis source policy;
     - recalculation job contract;
     - no COGS/margin guard until final cost basis.

4. `Reporting and OLAP — active Cloud UI reporting surfaces`
   - Children:
     - bounded raw event metadata screen;
     - stock moves screen;
     - stock movement summary screen;
     - sales/kitchen summary screen;
     - kitchen timing summary screen;
     - backfill job status screen;
     - production RBAC for mutating retry/backfill controls.

5. `Stop Lists — production review polish`
   - Children:
     - Cloud manager review polish;
     - assignment/escalation/dashboard decision;
     - Edge overlay conflict UX;
     - sync readiness visibility;
     - no raw payload validation.

6. `Recipes — full pilot recipe lifecycle`
   - Children:
     - recipe version editor parity in `cloud-ui-g`;
     - proposal review parity;
     - recipe expansion implementation;
     - semi-finished auto-production split decision.

7. `KDS — advanced kitchen lifecycle polish`
   - Children:
     - station/priority contract;
     - cooking events decision;
     - KDS analytics extensions;
     - hardware bump-bar/printer explicitly post-pilot.

8. `Security and RBAC — matrix acceptance`
   - Children:
     - backend permission catalog review;
     - role seed permission review;
     - UI visibility vs backend authority audit;
     - destructive storage permission;
     - Cloud API production auth/RBAC perimeter.

9. `Platform and Infrastructure — full local acceptance`
   - Children:
     - Docker smoke repeatability;
     - host port override docs;
     - buildx requirement docs;
     - CI command matrix;
     - seed script portability check.

10. `Payments / Fiscalization — architecture-only pilot-hardening`
    - Children:
      - payment provider abstraction contract;
      - fiscalization abstraction contract;
      - payment/refund/fiscal status separation;
      - reconciliation queue model;
      - PSP/fiscal real integration marked after pilot.

Definition of Done этапа:

- каждый open roadmap gap имеет Plane work item или explicit "outside scope" baseline note;
- нет задач с неподтвержденным runtime как "implemented";
- у каждого `Ready` work item есть acceptance criteria и проверки.

## Этап 3. Приоритизация По Cycles

Рекомендуемые cycles:

1. `2026-06 — Plane Bootstrap`
   - Документы управления.
   - Baseline по модулям.
   - Storage and Archiving уже начатые задачи.

2. `2026-06/07 — Cloud UI React Parity`
   - Перенос нужных исторических legacy-сценариев в `cloud-ui-g`.
   - Только поверх подтвержденных backend routes.

3. `2026-07 — Inventory Engine Foundation`
   - Balances/costing/status.
   - Worker slices.
   - Receipt/count/production state.
   - Smoke extensions.

4. `2026-07 — Reporting and OLAP UI`
   - Read-only reporting surfaces в `cloud-ui-g`.
   - RBAC decisions для mutating controls.

5. `2026-07/08 — Full Pilot Hardening`
   - Stop-list/recipe/KDS polish.
   - RBAC matrix.
   - Full smoke stabilization.

6. `After Pilot — PSP Fiscal Delivery Hardware`
   - PSP.
   - Fiscalization.
   - Delivery.
   - Hardware bump-bar/printers.
   - ERP/accounting integrations.

Cycles создавать только после подтверждения владельцем календаря и состава работ. До этого можно использовать текущий bootstrap cycle и module backlog.

## Этап 4. Детализация Ready-задач

Перед переводом задачи в `Ready` проверить:

- задача атомарна;
- нет скрытого изменения нескольких bounded contexts без parent task;
- описан текущий кодовый факт;
- перечислены конкретные файлы или области;
- указаны профильные документы;
- есть acceptance criteria;
- указаны проверки;
- указано, что вне scope;
- есть labels;
- есть module;
- есть dependencies или явно указано "нет";
- нет secrets или raw payloads.

Готовая задача для Codex должна быть выполнима одним focused turn или понятной серией turn-ов без пересогласования scope.

## Этап 5. Массовое Создание Задач

Массовое создание выполнять малыми партиями:

1. Сначала 3-5 baseline work items.
2. Проверить читаемость в Plane UI.
3. Проверить, что Codex корректно получает их через MCP.
4. Создать оставшиеся baseline work items.
5. Создать open-gap parent tasks.
6. Создать child work items только для ближайшего cycle.

Не создавать сразу весь post-pilot backlog до уровня атомарных задач. Достаточно parent tasks с `Backlog`/`Specified`, чтобы не зашумлять Plane.

## Этап 6. Контроль Качества Plane Backlog

Еженедельная проверка:

- нет `Ready` без acceptance criteria;
- нет `In Progress` без активного исполнителя или комментария;
- нет `Review` без проверок;
- нет `Validation` без описания ручной проверки;
- нет задач с `Done`, выставленным агентом без владельца;
- нет задач, где runtime и future plan смешаны;
- нет секретов в comments/descriptions;
- нет задач без module;
- parent/child связи не противоречат ownership.

Для документационных задач дополнительно проверить:

```bash
rg "future|later|maybe|probably|temporary for now|for now" AGENTS.md SPECv1.3.md ROADMAP.md docs
rg "implemented now|planned next|out of scope|Current status|Business rules|Architecture decisions|Pilot blockers|Context owns|Remaining risks" AGENTS.md SPECv1.3.md ROADMAP.md docs
```

Обычный русский текст документации должен использовать статусы:

- `реализовано сейчас`;
- `запланировано далее`;
- `вне текущего объема`;
- `после пилота`.

## Минимальные Проверки По Типам Задач

Backend Go:

```bash
cd pos-backend
go mod tidy
go test ./...
```

```bash
cd cloud-backend
go mod tidy
go test ./...
```

UI:

```bash
cd pos-ui-g
npm install
npm run build
```

```bash
cd cloud-ui-g
npm install
npm run build
```

Seed/smoke:

```bash
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.seed-dev-system-summary.json \
  --run-minimal-flow \
  --run-kitchen-process-smoke
```

Документационные задачи без runtime changes:

- `git diff --check`;
- проверить относительные ссылки;
- проверить, что изменены только Markdown/документные файлы;
- не запускать тяжелые runtime тесты без необходимости, но указать это в отчете.

## Рекомендуемые Первые Codex-прогоны

1. `POS-1` - подготовить project-management docs.
2. `POS-2` - завершить validation baseline по Storage and Archiving.
3. `Storage and Archiving — update smoke OpenAPI and docs gaps`.
4. `Storage and Archiving — cover archive write atomicity and destructive apply failure tests`.
5. `Plane Bootstrap — create POS Core baseline`.
6. `Plane Bootstrap — create Orders/Payments baseline`.
7. `Plane Bootstrap — create Cloud Backend Sync/Inventory baseline`.
8. `Plane Bootstrap — create Cloud UI active-vs-legacy baseline`.
9. `Cloud Backoffice — specify inventory/reporting migration to cloud-ui-g`.
10. `Security and RBAC — specify RBAC matrix acceptance`.

После этих прогонов Plane станет достаточно полным, чтобы новая разработка шла только через задачи.

## Риски

- Можно случайно перенести future plan как уже реализованное. Митигировать через baseline-аудит с подтверждением кодом/тестами.
- Можно перегрузить Plane слишком большим количеством post-pilot задач. Митигировать parent tasks для дальних направлений.
- Можно создать `Ready` задачи без реального acceptance. Митигировать weekly backlog QA.
- Можно смешать удаленный legacy `cloud-ui` и active `cloud-ui-g`. Митигировать явным статусом: runtime только `cloud-ui-g`.
- Можно расширить POS Edge складскую ответственность. Митигировать инвариантом: POS Edge не пишет stock documents/moves/balances/costing.
- Можно дать агенту destructive Plane/write operations шире текущей задачи. Митигировать правилом: write только для назначенного work item.

## Вне Текущего Объема Заселения Plane

- Автоматический массовый импорт всех пунктов roadmap без review.
- Изменение production/runtime-кода.
- Изменение миграций.
- Изменение API.
- Создание новых Plane labels/states/modules без подтверждения владельца.
- Перевод задач в `Done` агентом.
- Реальные PSP/fiscal/delivery/hardware integrations.
