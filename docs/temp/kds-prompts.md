Ниже набор универсальных промптов. Их можно выдавать агенту по одному, последовательно.

**Общий Префикс**

```text
Мы работаем в репозитории /home/serty/repos/mh-pos над RMS-POS для ресторанов.

Перед изменениями обязательно:
1. Прочитай AGENTS.md.
2. Прочитай docs/backend/KITCHEN-PROCESSES-SPEC.md.
3. Проверь текущее состояние кода и документации, не полагайся на старые выводы.
4. Используй CodeGraph для структурного анализа, rg только для текстовых поисков.
5. Не откатывай чужие изменения.
6. После реализации обнови профильную документацию, затронутую изменением.
7. В финале укажи: что найдено, что изменено, файлы, проверки, риски, что дальше, что вне объема, затрагивался ли runtime code.
```

**Итерация 1: POS Edge KDS Lifecycle**
Выполнена.

**Итерация 2: POS Edge Kitchen Stock Events**
Выполнена.

**Итерация 3: POS Edge Catalog And Recipe Proposals**
Выполнена.

**Итерация 4: Cloud Sync Contracts, ClickHouse Trail, Inventory Analyzer**

Выполнена.

**Итерация 5: Cloud Proposal Review And Feedback**
Выполнена.

**Итерация 6: pos-ui-g Kitchen Mode**

```text
Задача: реализовать настоящий kitchen mode в pos-ui-g вместо placeholder.

Сначала проверь pos-ui-g/src/App.tsx, POSContext, shared/api, schemas, i18n, shared/ui и текущий дизайн. Не работай в legacy pos-ui, если задача явно про pos-ui-g.

Нужно реализовать:
- нижний quick access только с тремя разделами: Заказы, Склад, Кухня;
- верхние вкладки внутри разделов;
- Заказы: очередь и готовые к выдаче;
- Склад: приемка, ревизия, списание, приготовление;
- Кухня: техкарты, предложения, мои предложения;
- order tile с временем, статусами, блюдами и actions;
- no optimistic status truth, после action перечитать backend;
- full catalog picker;
- safe localized errors через pos-ui-g i18n;
- формы receipt/count/write-off/production/proposals.

Обязательно обнови:
- docs/ui/POS-UI-SPEC.md;
- docs/backend/KITCHEN-PROCESSES-SPEC.md, если UI contract уточнился;
- docs/ui/POS-UI-RBAC.md;
- ROADMAP.md.

Проверки:
cd pos-ui-g && npm install && npm run build
При возможности добавить/запустить Playwright или component tests для kitchen flow.
```

**Итерация 7: Cloud UI Manager Review**

```text
Задача: реализовать Cloud UI surfaces для manager review catalog/recipe suggestions.

Сначала проверь cloud-ui текущую архитектуру, API clients, schemas, i18n и CLOUD-UI-SPEC. UI не должен имитировать отсутствующие backend routes.

Нужно реализовать:
- список catalog suggestions;
- список recipe suggestions;
- detail/diff view;
- approve/reject/request changes;
- linked new dish + recipe group display;
- safe error handling;
- no raw payload/PIN/token display;
- publication/readiness signal after approve.

Обязательно обнови:
- docs/ui/CLOUD-UI-SPEC.md;
- docs/backend/CLOUD-BACKEND-SPEC.md, если API contract уточнился;
- ROADMAP.md.

Проверки:
cd cloud-ui && npm install && npm run build
```

**Итерация 8: End-To-End Smoke And Documentation Alignment**

```text
Задача: собрать полный kitchen/process smoke и выровнять документацию с фактическим runtime.

Сначала проверь scripts/seed-dev-system.py, docker-compose.local.yml, LOCAL-DOCKER-STACK.md, текущие smoke tests и docs.

Нужно покрыть сценарий:
- Cloud seed publishes catalog/menu/recipes/inventory_reference;
- POS Edge sync receives full catalog and recipes;
- cashier/waiter creates order with dish;
- kitchen sees order tile;
- kitchen accept/start/ready/serve;
- kitchen recall served line and serve again;
- ClickHouse contains kitchen event trail;
- Cloud analyzer uses latest effective ItemServed;
- kitchen stock receipt/count/write-off/production events reach Cloud;
- Cloud stock ledger reflects expected stock documents;
- kitchen creates catalog + recipe suggestion;
- Cloud manager approves;
- Edge receives updated catalog/recipes/proposal feedback.

Обязательно обнови:
- SPECv1.3.md;
- ROADMAP.md;
- docs/CURRENT-FUNCTIONAL-STATE.md;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/backend/CLOUD-BACKEND-SPEC.md;
- docs/backend/INVENTORY-COSTING-SPEC.md;
- docs/backend/LOCAL-DOCKER-STACK.md;
- docs/sync/edge-cloud-contracts-v1.md;
- docs/sync/directional-sync-ownership.md;
- docs/ui/POS-UI-SPEC.md;
- docs/ui/CLOUD-UI-SPEC.md.

Проверки:
cd pos-backend && go mod tidy && go test ./...
cd cloud-backend && go mod tidy && go test ./...
cd pos-ui-g && npm install && npm run build
cd cloud-ui && npm install && npm run build
Запусти профильный smoke script, если окружение Docker доступно.
```