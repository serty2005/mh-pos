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

```text
Задача: реализовать POS Edge backend routes для кухонных складских событий.

Сначала проверь текущие inventory/cloudsync contracts и существующие Cloud worker capabilities. Реализуй только Edge-side command/input слой, POS Edge не должен создавать stock_documents/stock_ledger.

Нужно добавить routes:
- POST /api/v1/kitchen/stock-receipts;
- POST /api/v1/kitchen/inventory-counts;
- POST /api/v1/kitchen/stock-write-offs;
- POST /api/v1/kitchen/productions.

Нужно обеспечить:
- warehouse_id/default warehouse validation;
- receipt supplier/counterparty, document date, line totals;
- inventory count counted_quantity;
- write-off reason;
- production for semi_finished item;
- outbox events StockReceiptCaptured, InventoryCountCaptured, StockWriteOffCaptured, ProductionCompleted;
- idempotency by command_id;
- RBAC permissions from spec;
- no POS-side stock documents/moves/balances.

Обязательно обнови:
- docs/backend/KITCHEN-PROCESSES-SPEC.md;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/backend/INVENTORY-COSTING-SPEC.md;
- docs/backend/POS-ERROR-CATALOG.md;
- docs/sync/edge-cloud-contracts-v1.md;
- ROADMAP.md.

Проверки:
cd pos-backend && go mod tidy && go test ./...
```
**Итерация 2: POS Edge Kitchen Stock Events**
Выполнена.

**Итерация 3: POS Edge Catalog And Recipe Proposals**

```text
Задача: реализовать POS Edge backend для просмотра техкарт и создания предложений кухни.

Сначала проверь текущие catalog, recipe_versions, recipe_lines, master sync и catalog item read APIs.

Нужно реализовать:
- GET /api/v1/kitchen/catalog/items/{catalog_item_id}/recipe;
- POST /api/v1/kitchen/catalog-suggestions;
- POST /api/v1/kitchen/recipe-suggestions;
- GET /api/v1/kitchen/proposals.

Нужно обеспечить:
- полный каталог из POS Edge read model, не только меню;
- recipe read с ingredient names из catalog_items;
- CatalogItemChangeSuggested;
- RecipeChangeSuggested;
- proposal_group_id для нового блюда + техкарты;
- prep_time_delta validation;
- локальные proposal statuses;
- Edge не применяет предложения к catalog/recipe до Cloud approve/publication.

Обязательно обнови:
- docs/backend/KITCHEN-PROCESSES-SPEC.md;
- docs/backend/POS-BACKEND-SPEC.md;
- docs/sync/edge-cloud-contracts-v1.md;
- docs/sync/directional-sync-ownership.md;
- docs/ui/POS-UI-RBAC.md;
- ROADMAP.md.

Проверки:
cd pos-backend && go mod tidy && go test ./...
```
**Итерация 3: POS Edge Catalog And Recipe Proposals**
Выполнена.

**Итерация 4: Cloud Sync Contracts, ClickHouse Trail, Inventory Analyzer**

```text
Задача: расширить Cloud backend для новых kitchen/inventory/proposal events и ClickHouse kitchen event trail.

Сначала проверь cloud-backend/internal/cloudsync, inventory worker, ClickHouse forwarder, migrations и docs. Не ломай существующий sync/exchange ACK path.

Нужно обеспечить:
- contracts validation для CatalogItemChangeSuggested, RecipeChangeSuggested, StockWriteOffCaptured;
- расширенные поля ItemServed, StockReceiptCaptured, InventoryCountCaptured, ProductionCompleted, StopListUpdated;
- все kitchen events попадают в PostgreSQL inbox/journal и ClickHouse raw_business_events;
- proposal events не попадают в inventory_event_queue;
- inventory events попадают в durable processing;
- Cloud analyzer использует ClickHouse stream для latest effective ItemServed;
- recalled served events не дают duplicate stock consumption;
- warehouse sequence по restaurant_id + warehouse_id.

Обязательно обнови:
- docs/backend/CLOUD-BACKEND-SPEC.md;
- docs/backend/INVENTORY-COSTING-SPEC.md;
- docs/sync/edge-cloud-contracts-v1.md;
- docs/sync/directional-sync-ownership.md;
- SPECv1.3.md;
- ROADMAP.md.

Проверки:
cd cloud-backend && go mod tidy && go test ./...
```

**Итерация 5: Cloud Proposal Review And Feedback**

```text
Задача: реализовать Cloud review/apply workflow для предложений кухни.

Сначала проверь masterdata module, Cloud publications, Cloud -> Edge streams и Cloud UI contracts.

Нужно реализовать:
- cloud_catalog_suggestions;
- cloud_recipe_suggestions;
- cloud_recipe_suggestion_changes;
- cloud_suggestion_review_events;
- GET/approve/reject/request-changes routes для catalog suggestions;
- GET/approve/reject/request-changes routes для recipe suggestions;
- apply catalog suggestion only on manager approve;
- apply recipe suggestion only on manager approve;
- linked new dish + recipe proposal group transaction;
- proposal_feedback Cloud -> Edge stream;
- publication after approve/apply.

Обязательно обнови:
- docs/backend/CLOUD-BACKEND-SPEC.md;
- docs/backend/KITCHEN-PROCESSES-SPEC.md;
- docs/sync/edge-cloud-contracts-v1.md;
- docs/ui/CLOUD-UI-SPEC.md;
- ROADMAP.md.

Проверки:
cd cloud-backend && go mod tidy && go test ./...
```

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