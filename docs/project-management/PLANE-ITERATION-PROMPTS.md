# Промпты для итераций заселения Plane

Статус: рабочие промпты для последовательного выполнения плана из `docs/project-management/PLANE-POPULATION-PLAN.md` через Codex.

Дата: 2026-06-10.

Этот документ содержит готовые промпты для следующих Codex-прогонов. Каждый прогон должен работать с одним Plane work item или с явно указанной небольшой группой связанных work items. Перед запуском заменить placeholders в угловых скобках.

## Общие правила

Каждый промпт ниже предполагает:

- работа ведется в репозитории `/home/serty/repos/mh-pos`;
- ответы и документация пишутся на русском языке;
- сначала читаются `AGENTS.md`, `docs/project-management/PLANE-POPULATION-PLAN.md`, `tools/plane-mcp/README.md` и профильные документы;
- Plane write-операции выполняются только для явно назначенной задачи;
- runtime code не меняется в документационных/baseline итерациях;
- `Done` не выставляется агентом;
- итоговый комментарий в Plane обязателен, если работа идет по существующему Plane item.

Финальный отчет каждой итерации должен кратко указать:

- что найдено;
- что изменено;
- измененные файлы;
- какие проверки запущены;
- какие проверки не удалось запустить;
- оставшиеся риски;
- что запланировано далее;
- что вне текущего объема;
- затрагивался ли runtime code;
- краткий и полный комментарии о выполненных работах.

## Универсальный промпт для любой Plane-задачи

```text
Начни работу над Plane work item <POS-N>.

Сначала получи задачу через Plane MCP по identifier <POS-N>.
Проверь state, module, cycle, labels, описание, комментарии, links и relations.
Прочитай AGENTS.md, docs/project-management/PLANE-POPULATION-PLAN.md, tools/plane-mcp/README.md и профильные документы, связанные с module задачи.
Проверь git status.

Не расширяй scope задачи.
Если задача не готова к разработке, не начинай реализацию: добавь в ответ список недостающих требований и предложи, как перевести задачу в Ready.
Если задача в Ready и в описании не запрещено менять state, переведи ее в In Progress.

Выполни работу строго в scope.
Если меняются HTTP routes, payloads, UI flows, permission model, DB schema, sync events, error/logging contracts, migration/reset policy или startup/smoke scripts, обнови профильную документацию тем же изменением.
Запусти профильные проверки. Если проверку нельзя запустить, укажи причину.

Добавь итоговый комментарий в Plane:
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

Переведи задачу в Review только если работа выполнена и проверки/ограничения отражены.
Не выполняй commit, push, merge, release или deployment.
Не переводи задачу в Done.
```

## Итерация 1. Завершить POS-1 Project Management Docs

Использовать для задачи `POS-1`, если она остается основной задачей подготовки управления через Plane.

```text
Начни работу над Plane work item POS-1.

Цель итерации: завершить документационный foundation для управления разработкой mh-pos через Plane и Codex.

Сначала прочитай POS-1 через Plane MCP: state, module, cycle, labels, описание и comments.
Затем прочитай:
- AGENTS.md;
- tools/plane-mcp/README.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- README.md;
- ROADMAP.md;
- docs/CURRENT-FUNCTIONAL-STATE.md.

Проверь git status.
Runtime code, миграции и API не менять.

Нужно создать или актуализировать документы:
- docs/project-management/PLANE-STRUCTURE.md;
- docs/project-management/WORK-ITEM-STANDARD.md;
- docs/project-management/AGENT-WORKFLOW.md;
- docs/project-management/PROJECT-MODULE-MAP.md;
- docs/project-management/ROADMAP-MIGRATION-PLAN.md.

Если часть информации уже покрыта в PLANE-POPULATION-PLAN.md, не дублируй ее механически: вынеси reusable standards в отдельные документы, а plan оставь как runbook.

Критерии приемки:
- фактическая карта Plane project отражает текущий project identifier POS;
- modules, labels, states и cycles описаны;
- определено, что хранится в Plane, а что остается в Git;
- описан стандарт work item;
- описан workflow агента;
- есть карта модулей проекта с active/legacy статусами;
- есть план миграции roadmap gaps в Plane;
- все утверждения подтверждены кодом, Plane read-аудитом или профильными документами;
- изменены только Markdown-файлы.

Проверки:
- git diff --check;
- проверить относительные ссылки;
- git status --short.

Добавь итоговый комментарий в POS-1 и переведи в Review, если критерии закрыты.
Не переводи POS-1 в Done.
```

## Итерация 2. Storage and Archiving Baseline Validation

Использовать для `POS-2` или соответствующей baseline-задачи Storage and Archiving.

```text
Начни работу над Plane work item POS-2.

Цель итерации: завершить validation baseline по Storage and Archiving и убедиться, что Plane-задачи отражают фактический код, тесты, документы и оставшиеся риски.

Сначала прочитай POS-2 через Plane MCP, включая child tasks, comments, labels, state, module и cycle.
Прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- docs/backend/POS-DATA-AND-MIGRATIONS.md;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/backend/RUNTIME-CONFIG.md;
- docs/ui/POS-UI-SPEC.md;
- docs/sync/directional-sync-ownership.md;
- SPECv1.3.md;
- ROADMAP.md;
- docs/CURRENT-FUNCTIONAL-STATE.md.

Используй CodeGraph для структурного контекста Storage and Archiving перед любым grep:
- storage service;
- storage domain lifecycle;
- sqlite storage repository;
- storage API routes;
- sqlite-maintenance command;
- POS UI storage status usage.

Не меняй runtime code в этой итерации, если Plane item не требует реализации.

Нужно:
- проверить, что baseline description в POS-2 не заявляет неподтвержденное поведение;
- проверить, что child tasks покрывают найденные риски: permission/audit, restore policy, atomicity/failure tests, VACUUM/large DB, OpenAPI/docs, manual disposable SQLite validation;
- предложить недостающие child tasks, но не создавать их без явного указания в Plane item или подтверждения пользователя;
- обновить Markdown-документы только если найдены явные расхождения.

Критерии приемки:
- baseline перечисляет реализовано сейчас / запланировано далее / вне текущего объема;
- ссылки на код/тесты/доки указаны;
- remaining risks вынесены в child tasks;
- runtime code не затронут без необходимости.

Проверки:
- git diff --check;
- git status --short;
- если менялись только docs, Go/UI тесты не запускать и указать это.

Добавь итоговый комментарий в POS-2 и переведи в Review или Validation по фактическому состоянию.
Не переводи в Done.
```

## Итерация 3. Storage OpenAPI And Docs Gaps

Использовать для work item `Storage and Archiving — update smoke OpenAPI and docs gaps`.

```text
Начни работу над Plane work item <POS-N>, который соответствует задаче "Storage and Archiving — update smoke OpenAPI and docs gaps".

Цель итерации: синхронизировать smoke OpenAPI и профильные документы с фактическими Storage and Archiving routes.

Сначала прочитай задачу через Plane MCP.
Прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- docs/api/mhpos-local-smoke.openapi.json;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/backend/POS-DATA-AND-MIGRATIONS.md;
- pos-backend/README.md;
- SPECv1.3.md;
- ROADMAP.md.

Используй CodeGraph для router/storage route context.

Scope:
- обновить только OpenAPI/docs gaps, подтвержденные кодом;
- не менять runtime behavior;
- не добавлять неподтвержденные UI flows;
- не менять archive JSONL contract, если задача этого не требует.

Проверить расхождения:
- /storage/retention/dry-run;
- /storage/archive/export-plan;
- sqlite-maintenance support в README;
- текущий status/export/verify/read/lookup/apply contract.

Критерии приемки:
- smoke OpenAPI содержит подтвержденные Storage routes;
- README/docs не противоречат наличию sqlite-maintenance;
- docs используют русские статусы "реализовано сейчас", "запланировано далее", "вне текущего объема";
- runtime code не изменен.

Проверки:
- git diff --check;
- проверить JSON валидность OpenAPI через доступный formatter/parser;
- git status --short.

Добавь итоговый комментарий в Plane и переведи задачу в Review.
```

## Итерация 4. Storage Atomicity And Destructive Apply Tests

Использовать для work item `Storage and Archiving — cover archive write atomicity and destructive apply failure tests`.

```text
Начни работу над Plane work item <POS-N>, который соответствует задаче "Storage and Archiving — cover archive write atomicity and destructive apply failure tests".

Цель итерации: добавить или усилить тесты archive write atomicity и destructive apply failure paths.

Сначала прочитай задачу через Plane MCP.
Прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- docs/backend/POS-DATA-AND-MIGRATIONS.md;
- docs/backend/POS-BACKEND-SPEC.md;
- pos-backend/internal/pos/app/service_test.go;
- pos-backend/internal/pos/api/router_test.go;
- pos-backend/cmd/sqlite-maintenance/main_test.go.

Используй CodeGraph:
- storage service;
- BuildArchiveApplyPlan;
- ApplyStorageArchiveDestructive;
- storage repository;
- archive export/verify/apply tests.

Scope:
- добавлять focused tests;
- менять production code только если тест выявит реальный bug и исправление входит в задачу;
- не менять archive format без необходимости;
- не добавлять operator UI.

Критерии приемки:
- есть тест на частичный/ошибочный archive write или подтверждено, почему он невозможен на текущей архитектуре;
- есть тест на destructive apply failure path без удаления runtime rows;
- существующие storage tests остаются зелеными;
- docs обновлены, если уточнен contract.

Проверки:
- cd pos-backend && go test ./internal/pos/app ./internal/pos/api ./cmd/sqlite-maintenance;
- при изменении shared code рассмотреть cd pos-backend && go test ./...;
- git diff --check.

Добавь итоговый комментарий в Plane и переведи задачу в Review.
```

## Итерация 5. POS Core Baseline

Использовать для создания или выполнения задачи `Plane Bootstrap — create POS Core baseline`.

```text
Подготовь Plane baseline для module "POS Core".

Если Plane work item уже создан, начни с identifier <POS-N>.
Если work item еще не создан и пользователь подтвердил создание, создай его в project POS с названием:
"POS Core — verified implementation baseline".

Сначала прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- README.md;
- SPECv1.3.md;
- docs/CURRENT-FUNCTIONAL-STATE.md;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/backend/POS-ERROR-CATALOG.md;
- docs/backend/RUNTIME-CONFIG.md;
- docs/architecture/DDD-CONTEXT-MAP.md.

Используй CodeGraph для контекста POS Core:
- API router;
- auth/session/device;
- permission catalog;
- shared app services;
- error handling;
- runtime config.

Runtime code не менять.

Нужно сформировать baseline description для Plane:
- реализовано сейчас;
- запланировано далее;
- вне текущего объема;
- подтверждение кодом;
- подтверждение тестами;
- связанные документы;
- риски;
- рекомендуемые child tasks.

Особо проверить:
- backend authoritative session/RBAC;
- safe JSON error contract;
- request/correlation id;
- PIN/session/device boundaries;
- UI visibility не является security boundary;
- UUID v7 rule.

Проверки:
- git diff --check;
- git status --short.

Если работа идет по существующему item, добавь итоговый комментарий и переведи в Review или Validation.
```

## Итерация 6. Orders And Payments Baseline

Использовать для `Plane Bootstrap — create Orders/Payments baseline`.

```text
Подготовь Plane baseline для modules "Orders" и "Payments".

Если Plane work item уже создан, начни с identifier <POS-N>.
Если work item еще не создан и пользователь подтвердил создание, создай parent baseline:
"Orders and Payments — verified implementation baseline".

Сначала прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- SPECv1.3.md;
- ROADMAP.md;
- docs/CURRENT-FUNCTIONAL-STATE.md;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/ui/POS-UI-SPEC.md;
- docs/sync/edge-cloud-contracts-v1.md.

Используй CodeGraph для:
- order service;
- precheck service;
- payment/check service;
- financial operations;
- pricing integration in precheck/check;
- relevant router endpoints.

Runtime code не менять.

Нужно сформировать baseline:
- Order -> Precheck -> Payment -> Check;
- active order lines, quantity, void, modifiers;
- immutable precheck/check snapshots;
- precheck-based payments;
- partial payments;
- final check after full payment;
- cancellation/refund append-only ledger;
- compatibility payment refund wrapper;
- current Edge -> Cloud financial operation events;
- known gaps: PSP, fiscal, modifier/service/tip partial UI if not required.

Критерии приемки:
- cancellation и refund явно разделены;
- financial operation не создает stock moves автоматически;
- finalized checks/payments не мутируются;
- UI не является financial authority;
- все planned/future items отмечены как "запланировано далее" или "вне текущего объема".

Проверки:
- git diff --check;
- git status --short.

Добавь итоговый Plane comment и переведи baseline в Review/Validation.
```

## Итерация 7. Cloud Sync And Inventory Baseline

Использовать для `Plane Bootstrap — create Cloud Backend Sync/Inventory baseline`.

```text
Подготовь Plane baseline для "Edge-Cloud Synchronization" и "Inventory".

Если Plane work item уже создан, начни с identifier <POS-N>.
Если work item еще не создан и пользователь подтвердил создание, создай parent baseline:
"Cloud Sync and Inventory — verified implementation baseline".

Сначала прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- docs/backend/CLOUD-BACKEND-SPEC.md;
- docs/backend/INVENTORY-COSTING-SPEC.md;
- docs/sync/edge-cloud-contracts-v1.md;
- docs/sync/directional-sync-ownership.md;
- docs/architecture/DDD-CONTEXT-MAP.md;
- ROADMAP.md;
- docs/CURRENT-FUNCTIONAL-STATE.md.

Используй CodeGraph для:
- cloudsync API/router/service;
- inventory worker/service/repository;
- masterdata publication;
- POS mastersync;
- POS syncsender;
- Cloud sync exchange.

Runtime code не менять.

Нужно сформировать baseline:
- Cloud -> Edge streams;
- Edge outbox и sync/exchange;
- item-level ACK/retry;
- inbox_events;
- inventory_event_queue;
- Cloud Inventory Worker;
- stock_documents/stock_ledger foundation;
- stock_balances read endpoint;
- current inventory-relevant events;
- StopListUpdated processing;
- no POS Edge stock documents/moves/balances/costing.

Критерии приемки:
- Cloud ownership clearly stated;
- POS Edge only emits events and validates local runtime;
- stock balance does not block sale;
- stop-list is only sale blocking mechanism;
- full costing/retro DAG marked planned, not implemented.

Проверки:
- git diff --check;
- git status --short.

Добавь итоговый Plane comment и переведи в Review/Validation.
```

## Итерация 8. Cloud UI Active vs Legacy Baseline

Использовать для `Plane Bootstrap — create Cloud UI active-vs-legacy baseline`.

```text
Подготовь Plane baseline для module "Cloud Backoffice".

Если Plane work item уже создан, начни с identifier <POS-N>.
Если work item еще не создан и пользователь подтвердил создание, создай baseline:
"Cloud Backoffice — active cloud-ui-g vs legacy cloud-ui baseline".

Сначала прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- docs/ui/CLOUD-UI-SPEC.md;
- README.md;
- ROADMAP.md;
- docs/CURRENT-FUNCTIONAL-STATE.md;
- docs/backend/CLOUD-BACKEND-SPEC.md.

Осмотри структуру:
- cloud-ui-g/src/features;
- cloud-ui/src/components/cloud;
- cloud-ui-g/src/shared/api;
- cloud-ui-g/src/i18n или shared/i18n.

Runtime code не менять.

Нужно сформировать baseline:
- active Cloud UI target: cloud-ui-g;
- legacy cloud-ui: reference-only;
- route-backed features already in cloud-ui-g;
- legacy-only features that must be migrated separately;
- blocked placeholders inventory/reports in active UI;
- i18n and safe error expectations;
- no Cloud cashier/POS runtime authority.

Критерии приемки:
- не смешаны active и legacy статусы;
- legacy functionality не заявлена как реализованная в cloud-ui-g;
- migration child tasks предложены по подтвержденным backend routes;
- scope future BI/COGS/retry controls отмечен осторожно.

Проверки:
- git diff --check;
- git status --short.

Добавь итоговый Plane comment и переведи в Review/Validation.
```

## Итерация 9. Cloud Backoffice Inventory/Reporting Migration Spec

Использовать для `Cloud Backoffice — specify inventory/reporting migration to cloud-ui-g`.

```text
Начни работу над Plane work item <POS-N>, который соответствует задаче "Cloud Backoffice — specify inventory/reporting migration to cloud-ui-g".

Цель итерации: подготовить спецификацию и Plane child tasks для переноса inventory/reporting screens из legacy cloud-ui в active cloud-ui-g.

Сначала прочитай задачу через Plane MCP.
Прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- docs/ui/CLOUD-UI-SPEC.md;
- docs/backend/CLOUD-BACKEND-SPEC.md;
- docs/backend/INVENTORY-COSTING-SPEC.md;
- ROADMAP.md;
- docs/CURRENT-FUNCTIONAL-STATE.md.

Осмотри:
- cloud-ui legacy screens для inventory/OLAP/reporting;
- cloud-ui-g features/shared api/schema/i18n;
- подтвержденные Cloud backend routes.

Runtime code не менять, если задача именно specification.

Нужно:
- сформировать migration spec в docs/project-management или профильном docs/ui документе;
- разбить перенос на child work items;
- для каждого screen указать backend routes, DTO, labels, tests, docs;
- явно запретить raw payload, PIN/token/request dumps;
- mutating retry/backfill controls оставить blocked до production RBAC decision.

Минимальные child tasks:
- stock balances screen;
- stock ledger screen;
- OLAP export status screen;
- stock moves screen;
- stock move summary screen;
- sales/kitchen summary screen;
- kitchen timing summary screen;
- backfill job status read-only screen;
- proposal/recipe/stop-list review parity, если входит в parent scope.

Проверки:
- git diff --check;
- проверить Markdown links;
- git status --short.

Добавь итоговый Plane comment и переведи задачу в Review.
```

## Итерация 10. Security And RBAC Matrix Acceptance

Использовать для `Security and RBAC — specify RBAC matrix acceptance`.

```text
Начни работу над Plane work item <POS-N>, который соответствует задаче "Security and RBAC — specify RBAC matrix acceptance".

Цель итерации: подготовить acceptance plan для сверки backend permissions, seeded roles, UI visibility и destructive/high-risk operations.

Сначала прочитай задачу через Plane MCP.
Прочитай:
- AGENTS.md;
- docs/project-management/PLANE-POPULATION-PLAN.md;
- docs/ui/POS-UI-RBAC.md;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/backend/POS-ERROR-CATALOG.md;
- docs/backend/CLOUD-BACKEND-SPEC.md;
- docs/architecture/DDD-CONTEXT-MAP.md;
- SPECv1.3.md.

Используй CodeGraph для:
- permission catalog;
- role/permission seed or masterdata handling;
- manager override;
- auth/session services;
- storage destructive apply permission if present;
- POS UI permission checks.

Runtime code не менять, если задача specification.

Нужно:
- описать RBAC matrix acceptance в docs/project-management или профильном RBAC документе;
- выделить high-risk operations: payments, refunds, cancellations, storage destructive apply, KDS status change, stop-list updates, OLAP mutating controls;
- отделить UI visibility от backend authority;
- указать required tests для backend permission enforcement;
- создать или предложить Plane child tasks для найденных gaps.

Критерии приемки:
- у каждой high-risk operation есть authoritative backend permission decision или explicit gap;
- нет raw Go/SQL/internal error exposure in UI plan;
- destructive operations требуют отдельного решения/audit;
- Cloud production auth/RBAC отмечен отдельно от local pilot foundation.

Проверки:
- git diff --check;
- проверить ссылки;
- git status --short.

Добавь итоговый Plane comment и переведи задачу в Review.
```

## Промпт для создания следующей партии baseline tasks

Использовать после завершения первых baseline-прогонов.

```text
Продолжи заселение Plane по docs/project-management/PLANE-POPULATION-PLAN.md.

Цель итерации: создать следующую небольшую партию baseline work items для modules:
<MODULE_LIST>

Перед созданием задач:
- прочитай AGENTS.md;
- прочитай tools/plane-mcp/README.md;
- прочитай docs/project-management/PLANE-POPULATION-PLAN.md;
- получи список текущих Plane modules и work items через Plane MCP;
- убедись, что baseline для этих modules еще не создан.

Создавай не больше 3-5 work items за один прогон.
Для каждого work item:
- name: "<Module> — verified implementation baseline";
- module: соответствующий Plane module;
- state: Specified или Ready, если описание уже полное;
- labels: documentation, research, agent-ready только если задача действительно готова для агента;
- description: использовать Work Item Standard из плана;
- не переводить в Done.

После создания:
- добавь краткий отчет со списком созданных задач;
- не создавай child tasks без отдельного подтверждения;
- не меняй runtime code;
- не выполняй commit/push.
```

## Промпт для QA backlog после заселения

```text
Проведи QA Plane backlog для проекта POS.

Цель: проверить, что заселение Plane соответствует docs/project-management/PLANE-POPULATION-PLAN.md и готово для дальнейшей разработки через Codex.

Сначала прочитай:
- AGENTS.md;
- tools/plane-mcp/README.md;
- docs/project-management/PLANE-POPULATION-PLAN.md.

Через Plane MCP получи:
- modules;
- states;
- labels;
- cycles;
- work items по project POS.

Проверь:
- у каждого work item есть module;
- Ready-задачи имеют criteria, scope, out-of-scope и проверки;
- In Progress задачи имеют исполнителя или свежий комментарий;
- Review задачи имеют комментарий о проверках;
- Validation задачи имеют manual/integration validation plan;
- Done не выставлен агентом без owner review comment;
- future plan не описан как реализовано сейчас;
- legacy cloud-ui не смешан с active cloud-ui-g;
- destructive/high-risk tasks имеют security/audit consideration;
- нет secrets/raw payloads в descriptions/comments, если доступно проверить.

Не меняй Plane без отдельного подтверждения.
Сформируй отчет:
- найдено;
- критичные проблемы;
- задачи, которые нужно вернуть из Ready в Specified;
- задачи без module/labels;
- рекомендуемые следующие действия.
```
