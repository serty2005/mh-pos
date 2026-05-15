# Аудит документации и UI/UX на 2026-05-15

Статус: рабочий audit report для pre-pilot синхронизации документации, roadmap и фактического кода.

Runtime code не изменялся. Проверка выполнена по исходникам, профильной документации, сборкам UI и доступным тестам. Browser-based Playwright-сценарий не был завершен из-за отсутствия браузера в окружении и блокировки загрузки Chromium proxy/403; выводы по UI/UX основаны на source review, сборке и текущей структуре Vue/Quasar компонентов.

## Проверенные источники

Документация:

- `SPECv1.3.md`;
- `ROADMAP.md`;
- `docs/backend/POS-BACKEND-SPEC.md`;
- `docs/backend/POS-DATA-AND-MIGRATIONS.md`;
- `docs/backend/POS-ERROR-CATALOG.md`;
- `docs/backend/RUNTIME-CONFIG.md`;
- `docs/sync/edge-cloud-contracts-v1.md`;
- `docs/sync/directional-sync-ownership.md`;
- `docs/architecture/DDD-CONTEXT-MAP.md`;
- `docs/ui/POS-UI-SPEC.md`;
- `docs/ui/CLOUD-UI-SPEC.md`;
- `docs/adr/ADR-015-persistence-and-analytics-strategy.md`.

Кодовые источники для сверки:

- `pos-backend/internal/pos/api/router.go`;
- `pos-backend/internal/pos/app/*`;
- `pos-backend/internal/pos/domain/*`;
- `pos-backend/internal/pos/infra/sqlite/*`;
- `pos-backend/migrations/sqlite/001_init.sql`;
- `cloud-backend/internal/cloudsync/api/router.go`;
- `cloud-backend/internal/masterdata/api/router.go`;
- `cloud-backend/internal/provisioning/api/router.go`;
- `cloud-backend/migrations/postgres/001_init.sql`;
- `pos-ui/src/router.ts`;
- `pos-ui/src/pages/PosPage.vue`;
- `pos-ui/src/pages/pos/*`;
- `cloud-ui/src/App.vue`;
- `cloud-ui/src/shared/*`.

## Краткий вывод

Документация в целом соответствует текущему коду по главным pilot-инвариантам:

- cashier runtime описан как `Order -> Precheck -> Payment -> Check`, а не legacy check-first/payment-by-check flow;
- payments привязаны к `precheck_id`;
- refund/cancellation описаны как append-only financial operation ledger, а не переписывание finalized документов;
- Cloud остается authority для master data, POS Edge — локальный runtime/read model;
- `sqlc` и ClickHouse не заявлены как текущий runtime;
- recipes/inventory корректно описаны как основа, а не готовый runtime;
- POS UI и Cloud UI разведены по назначению и не обещают неподтвержденные cashier flows в Cloud.

Roadmap был дополнен отдельным блоком аудита 2026-05-15: фактические статусы оставлены, а UI/UX и smoke/e2e вынесены в pre-pilot hardening, чтобы не считать визуальную приемку завершенной без browser-based проверки.

## Сверка backend-документации с кодом

### POS Edge runtime

Реализовано сейчас и корректно отражено в документации:

- auth/session endpoints: `POST /api/v1/auth/pin-login`, `POST /api/v1/auth/logout`, `GET /api/v1/auth/session`;
- provisioning/pairing endpoints Edge side;
- halls/tables/menu/catalog read APIs;
- employee shifts, cash sessions and cash drawer events;
- order create/read/current/closed, line add/change/void;
- order discount/surcharge commands and pricing preview;
- precheck issue/list/get/cancel/reprint;
- payments through `POST /api/v1/prechecks/{id}/payments`;
- checks get/reprint;
- append-only cancellation/refund routes: `POST /api/v1/checks/{id}/cancellations`, `POST /api/v1/checks/{id}/refunds`;
- compatibility payment refund route: `POST /api/v1/payments/{id}/refund`;
- sync outbox/local-events/status/retry and master-data ingest.

Фактическое состояние, которое важно не переобещать:

- backend имеет check-level cancellation/refund routes, но cashier UI пока exposes только payment-level compatibility refund из закрытых заказов;
- manual discount/surcharge commands есть на backend, но cashier UI editor для них не реализован;
- recipes/inventory schema/domain foundation есть, но automatic stock consumption не реализован как pilot runtime;
- fiscal adapter and real PSP не реализованы.

### Cloud backend and sync

Реализовано сейчас и корректно отражено в документации:

- Cloud sync receiver: single and batch Edge events;
- provisioning master-data stream endpoints;
- provisioning device registration, unassigned devices, restaurant assignment, assignment status and pairing code flow;
- Cloud master-data CRUD for restaurants, roles, employees, catalog/menu/floor/modifiers/pricing/publications;
- published package/snapshot endpoints;
- accepted operational event catalog includes current `CancellationRecorded`/`RefundRecorded` and legacy accepted `PaymentRefunded`/`CheckRefunded`.

Фактическое состояние, которое важно не переобещать:

- Cloud UI остается pilot/admin operational center without Cloud auth/RBAC UI;
- delivery state of published package to Edge is not confirmed as a full UI contract;
- inventory/recipes are not ready cashier runtime flows.

## Документационные несоответствия и риски

Критичных несоответствий, где документация прямо обещает отсутствующий runtime, не найдено.

Оставшиеся риски:

1. Некоторые человекочитаемые bullet points в roadmap и docs остаются смешанными русско-английскими. Это допустимо для API/SDK/DDD терминов, но перед frozen pilot стоит унифицировать статусы и пользовательские формулировки.
2. `docs/temp/PLAN.md` является историческим планом и не должен восприниматься как источник текущего runtime. Если документ остается в репозитории, рядом нужен явный статус «исторический план» либо перенос актуальных пунктов в roadmap.
3. Backend capability по cancellation/refund шире текущей cashier UI capability. Это уже отражено в `docs/ui/POS-UI-SPEC.md`, но должно оставаться видимым в acceptance script.
4. Browser-based UI/UX проверка не завершена из-за ограничения окружения, поэтому визуальные выводы требуют повторной проверки в окружении с установленным Chromium/Playwright browsers.

## UI/UX аудит POS Edge UI

Реализовано сейчас:

- `/pos` and `/pos/cashier` ведут в cashier terminal;
- shell route pages для waiter/kitchen/manager показывают «вне текущего объема»;
- терминал разделен на status bar, floor/table selector, order workspace, catalog/checkout panel и dialogs/drawers;
- тексты идут через `vue-i18n`;
- основные опасные действия gated by permissions in UI, backend остается authority;
- responsive breakpoint collapses three-column terminal to two columns and then one column.

Выявленные UX-проблемы и зоны hardening:

1. **Слишком высокая когнитивная плотность для кассира.** На одном экране одновременно присутствуют смена, кассовая смена, столы, заказ, меню, услуги, пречек, оплата, sync, closed orders и cash drawer. Для пилота нужен task-first режим: «готовность смены -> стол -> заказ -> пречек -> оплата» с второстепенными действиями в отдельном utility rail/drawer.
2. **Нет полноценной browser-based smoke проверки.** Сборка проходит, но Playwright не стартует без браузера. Нужно добавить воспроизводимый локальный/CI сценарий с seeded backend или mock layer для `login -> shift/cash session -> order -> precheck -> payment -> check -> refund/reprint`.
3. **Средний tablet breakpoint рискованный.** На ширине до 1100px action pane уходит вниз, что может отделить checkout от заказа и увеличить скролл в самом важном cashier flow.
4. **Manager/refund/cash drawer dialogs требуют acceptance copy review.** Нужна финальная проверка wording для fiscal/operator policy и запрет показа raw technical errors.
5. **Нет отдельной визуальной иерархии для блокирующих условий.** `noShift`, `noCashSession`, locked order and permission-disabled actions должны иметь одинаковый pattern: причина, следующее действие, кто может исправить.

## UI/UX аудит Cloud UI

Реализовано сейчас:

- Cloud UI является отдельным Vite/Vue/Quasar приложением;
- есть сценарный launch plan, onboarding checks, Edge-device flow, publication flow and technical master-data tables/forms;
- restaurant-scoped операции gated by selected restaurant;
- Cloud UI не вызывает POS Edge cashier endpoints and не использует POS session store;
- тексты идут через `vue-i18n`.

Выявленные UX-проблемы и зоны hardening:

1. **`cloud-ui/src/App.vue` является монолитным компонентом.** В одном файле сосредоточены navigation, onboarding, tables, forms, role permissions, provisioning and publication flows. Это повышает риск regression при UX-правках и усложняет тестирование.
2. **Техническая admin surface конкурирует со сценарным launch flow.** Launch plan существует, но master-data CRUD still dominates navigation. Для оператора первого запуска нужен wizard/checklist with next best action, а технические таблицы должны быть secondary.
3. **Readiness checks не являются полноценной acceptance panel.** Нужны явные blocked/ready states: ресторан выбран, роли/сотрудники готовы, зал/столы готовы, меню продаваемо, Edge назначен, publication создана, snapshot доступен.
4. **Табличный UX плохо масштабируется на mobile/narrow screens.** Таблицы имеют `min-width: 720px`; это приемлемо для admin desktop, но для narrow browser нужно card/list fallback для ключевых launch flows.
5. **Ошибка backend/API показывается единым верхним alert.** Нужны contextual inline recovery actions near failed step: retry, select restaurant, open related section, copy safe diagnostic id.

## Рекомендуемый порядок исправления UI/UX

1. Не менять backend contracts в UX PR без отдельного требования.
2. Разделить Cloud UI на компоненты по bounded flows: shell/navigation, launch readiness, edge devices, publications, resource table, resource form, role permission matrix.
3. Добавить deterministic smoke fixture/mock для обоих UI, чтобы Playwright мог пройти без ручного заполнения данных.
4. POS UI: уплотнить secondary operations в utility drawer, а checkout/precheck/payment держать рядом с активным заказом на tablet widths.
5. Cloud UI: превратить launch plan в основной actionable wizard/checklist; master-data tables оставить доступными, но не primary journey.
6. Повторить visual/a11y smoke в Chromium после установки Playwright browsers.

## Выполненные проверки

- `rg "Order.*Check.*Payment|Check.*Payment|payment.*check_id|check_id.*payment" .` — сверка legacy payment/check формулировок.
- `rg "business_date_local|reprint|повторн.*печать|print snapshot|item-level ACK|batch ACK" SPECv1.3.md ROADMAP.md docs` — сверка business date/reprint/sync wording.
- `rg "bounded context|DDD|POSContext|Organization|Catalog|Pricing|Inventory" AGENTS.md SPECv1.3.md ROADMAP.md docs` — сверка DDD/context wording.
- `rg "future|later|maybe|probably|temporary for now|for now" AGENTS.md SPECv1.3.md ROADMAP.md docs` — поиск запрещенных временных формулировок.
- `rg "implemented now|planned next|out of scope|Current status|Business rules|Architecture decisions|Pilot blockers|Context owns|Remaining risks" AGENTS.md SPECv1.3.md ROADMAP.md docs` — поиск английских человекочитаемых статусов.
- `cd pos-ui && npm run build` — production build passed.
- `cd cloud-ui && npm run build` — production build passed.
- `cd pos-ui && npm run test` — unit tests passed.
- `cd pos-ui && npx playwright install chromium` — failed due proxy/403, browser-based UI test blocked by environment.
- `apt-get update && apt-get install -y chromium` — failed due proxy/403, fallback system browser install blocked by environment.
- `cd pos-backend && go mod tidy && go test ./...` — failed due blocked Go module downloads from `proxy.golang.org`.
- `cd cloud-backend && go mod tidy && go test ./...` — failed due blocked Go module downloads from `proxy.golang.org`.
