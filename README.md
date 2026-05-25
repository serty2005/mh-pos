# myhoreca-pos

Короткая карта репозитория `ASMaslovMH/myhoreca-pos`. Подробные правила, контракты и планы находятся в профильных документах, а не в README.

## Текущее состояние

Реализовано сейчас:

- POS Edge backend поддерживает cashier runtime `Order -> Precheck -> Payment -> Check`.
- Текущая личная смена сотрудника ищется по authenticated employee; отсутствие открытой личной смены возвращается как optional `200 null`, а не как runtime error.
- `IssuePrecheck` блокирует заказ, создает immutable financial snapshot precheck и фиксирует `currency_code`, subtotal, discounts, surcharges, taxes, grand total, paid/remaining totals и breakdown строк/налогов/скидок/надбавок.
- POS Edge backend содержит MVP `Pricing` boundary: line/order discounts, synced automatic discount/surcharge policies, manual/service/PB1 surcharge foundation, единый ordered discount/surcharge pipeline по `application_index`, percentage/fixed amounts, percentage/fixed tax rules, inclusive/exclusive tax foundation и deterministic integer rounding.
- POS Edge order runtime хранит selected modifiers в строках заказа, учитывает цену modifiers в backend authoritative totals и сохраняет modifiers в precheck/check snapshots.
- POS cashier UI показывает отдельную секцию услуг, открывает выбор modifiers для позиций с modifier groups и отображает выбранные modifiers в активном заказе.
- POS cashier UI использует текущий shell `floor` / `order` / `activity` / `reports` / `cash`; delivery, settings, storage/archive/retention и Cloud reporting не являются operator-facing cashier flows.
- POS waiter UI реализован как mobile-first route `/pos/waiter`: зал/стол, активные заказы, создание заказа, меню/поиск, modifiers при добавлении строки, quantity, void line и issue/reprint precheck без payment/refund/cash drawer authority по умолчанию.
- POS kitchen route `/pos/kitchen` реализован сейчас только как honest readiness screen: он показывает `запланировано далее` и отсутствующие KDS backend contracts, не активный KDS lifecycle runtime.
- Active-looking POS UI placeholders для переноса/разделения строки, banquet/preorder, mock waiter filters и discount/surcharge editor не считаются реализованным runtime: они скрыты, passive или disabled/backlog до появления backend/API/UI contract; повторяющиеся backlog/readiness cards отображаются через reusable `PosReadinessCard`.
- `CancelPrecheck` требует manager override, проверяет PIN/permission и возвращает unpaid active precheck order в `open`.
- Оплата выполняется через `precheck_id`; partial payments разрешены; final check создается только после полной оплаты.
- `POST /api/v1/checks/{id}/cancellations` и `POST /api/v1/checks/{id}/refunds` пишут append-only ledger `financial_operations`/`financial_operation_items` для full/partial cancellation и refund без мутации finalized payment/precheck/check.
- POS cashier UI вызывает check cancellation/refund ledger endpoints из закрытого заказа: поддержаны full whole-check операции и partial `order_line`/quantity операции по immutable check/precheck snapshot; UI отправляет `command_id`, `operation_kind`, `inventory_disposition`, reason и `items[]`, а backend остается источником истины для enforcement. Compatibility payment refund остается отдельным fallback.
- `GET /api/v1/checks/{id}/financial-operations?limit=&offset=` и `GET /api/v1/financial-operations?business_date_from=&business_date_to=&operation_type=&shift_id=&original_shift_id=&check_id=&limit=&offset=` реализованы сейчас как bounded read-only ledger surfaces; POS activity detail показывает type, kind, amount, reason, employee/approver, business date, inventory disposition и created time.
- `GET /api/v1/orders/closed` реализовано сейчас как bounded read: default `limit=50`, max `limit=100`, `offset`, фильтры `business_date_local`, `from_business_date_local`, `to_business_date_local`, `shift_id`, `device_id`, `check_id`, stable newest-first sort. POS UI использует бизнес-дату и постраничную навигацию.
- `GET /api/v1/sync/outbox` и `GET /api/v1/sync/local-events` также являются bounded operational reads: backend default `limit=100`, oversized/empty limit falls back to bounded default, UI sync/activity drawer запрашивает `limit=5`.
- POS Edge backend имеет безопасный lifecycle surface локальной SQLite БД: `GET /api/v1/storage/status` и `POST /api/v1/storage/retention/dry-run` считают объемы закрытых заказов, financial ledger, active/open blockers и outbox/sync состояния без удаления данных; `POST /api/v1/storage/archive/export-plan` возвращает manifest-only план кандидатов, protected flags и `result_mode = plan_only`; `POST /api/v1/storage/archive/export` создает export-only JSONL archive и manifest для closed orders с `checks.business_date_local < cutoff_business_date_local` и `runtime_rows_deleted = false`; `POST /api/v1/storage/archive/verify` non-destructive проверяет manifest/version/SHA/counts, required identity fields, business-date range, exclusive cutoff consistency, `runtime_rows_deleted = false` и payload policy для `local_event_log`/`pos_sync_outbox` summaries; `POST /api/v1/storage/archive/read-plan` возвращает bounded preview archived closed orders с filters `business_date_local`, `order_id`, `check_id`, `limit`, `offset` без восстановления в SQLite и без sync/event payload JSON; `POST /api/v1/storage/archive/lookup` streaming-способом возвращает immutable check/precheck snapshot preview по `check_id` или `order_id`; `POST /api/v1/storage/archive/apply-plan` проверяет archive/runtime safety (verified JSONL, scoped sent outbox, no open operational boundaries для cutoff) и при прохождении gate выполняет destructive apply с `result_mode = destructive_apply`, `runtime_rows_deleted = true`; `POST /api/v1/storage/archive/apply-readiness` возвращает `ready_for_destructive_apply = true` при прохождении всех проверок. Физическое удаление + VACUUM compaction реализованы сейчас (реализовано).
- `POST /api/v1/payments/{id}/refund` оставлен как compatibility wrapper: он требует finalized check, записывает `RefundRecorded` operation по payment allocation и не переводит payment/check обратно в mutable состояние.
- Cloud receiver принимает current `CancellationRecorded`/`RefundRecorded` и legacy inbound-only `PaymentRefunded`/`CheckRefunded`; для current events validation сверяет `restaurant_id`/`device_id` payload с envelope и требует поля operation/check/precheck/shift/date/type/disposition/reason/snapshot. Реализована detailed PostgreSQL/service projection `cloud_projection_financial_operations` с фильтрами restaurant/date/type/shift/original shift/check. Публичный Cloud HTTP reporting API/UI для этой projection вне текущего runtime.
- Reprint precheck/check строится из immutable snapshot.
- `scripts/seed-dev-system.py` является единственным локальным Python seed script: он создает полный Cloud-owned dataset, публикует master data, выполняет license pairing POS Edge и проверяет базовый POS read model.
- Runtime cashier/refund/stop-list сценарии проверяются профильными backend/UI тестами; seed script не заменяет e2e runtime smoke и не выполняет destructive storage actions.
- Python stack smoke содержит suite `pos_cashier_runtime`, которая проверяет backend путь после Cloud -> Edge master-data sync: PIN login, личную смену, cash shift, hall/table/menu read models, заказ, обычную строку, modifiers при наличии, service item при наличии, precheck, оплату по `precheck_id`, final check, bounded closed orders, check get/reprint, same-shift cancellation ledger, financial operations read и `GET /api/v1/storage/status`.
- Python stack smoke содержит suite `pos_refund_after_shift_close`, которая создает отдельную POS sale, закрывает исходные cash shift и personal employee shift, открывает новую сменную границу для refund под менеджером, записывает full refund через `/checks/{id}/refunds`, проверяет ledger через `/checks/{id}/financial-operations` и bounded closed-order/check reads без PSP, fiscal или destructive storage действий.
- Python stack smoke содержит suite `pos_stop_list_sale_blocking`: Cloud authoring recipes/stop-list -> publication -> Edge import -> локальная блокировка продажи по direct stop-list и recipe component stop-list.
- Cloud -> Edge master-data ingest в POS Edge runtime поддерживает потоки `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`, `recipes`, `inventory_reference`.
- POS Edge backend локально блокирует продажу при добавлении order line и при увеличении quantity, если продаваемый `catalog_item_id` или обязательный компонент active recipe version находится в active `stop_lists` с `available_quantity = 0` или `NULL`; stock balance для sale blocking не используется.
- Cloud/Edge master data разделяет menu categories, catalog folders и tags; `catalog` stream передает folders, folder parameters, tags, item tags, services и modifier groups/options/bindings, а `menu` stream передает menu items и effective modifier links.
- Cloud publication snapshot для POS Edge публикуется как typed ingest DTO: `modifier_groups[]` сохраняет `required`, `min_count`, `max_count`, `active`, а `menu_item_modifier_groups[]` остается link-only без rich/UI fields. Единый seed flow отправляет опубликованный Cloud snapshot на POS Edge без PowerShell field stripping.
- Inventory runtime переведен на Cloud-centric cutover: POS Edge больше не содержит manual stock document service и SQLite tables `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, `purchase_receipts`, `purchase_receipt_lines`; исторически этот pre-pilot Edge-side метод использовался как foundation и удален при переходе.
- Cloud принимает inventory events через sync receiver, кладет их в durable `inventory_event_queue`, а Cloud Inventory Worker пишет Cloud-owned `stock_documents` и `stock_ledger` для нормализованных item payloads. Cloud package contracts/storage принимают `recipes` и `inventory_reference`; Cloud UI уже имеет manager-facing authoring для recipe items и stop-list по подтвержденным master-data routes. Proposal review, inventory operations/costing и OLAP exports пока показаны как readiness-only surfaces без имитации отсутствующих endpoints.

Вне текущего runtime:

- automatic recipe expansion / stock consumption engine;
- recipe-expanded stock return/write-off from financial operations beyond normalized item payloads;
- Cloud proposal review, inventory operations/costing UI и OLAP export runtime;
- PSP refund smoke и fiscal integration;
- operator-facing storage/archive/retention UI, archive restore в active SQLite и ручной destructive retention flow вне подтвержденного backend archive apply contract;
- fiscal shift/business day сущности как отдельные runtime aggregates;
- real payment processor module, PSP webhooks и fiscal adapter;
- ClickHouse runtime pipeline;
- подтвержденный `sqlc` persistence rollout.

## Структура

- `pos-backend/` — POS Edge Go backend, SQLite runtime, cashier API.
- `pos-ui/` — Vue/Quasar cashier UI.
- `cloud-backend/` — Cloud API, PostgreSQL sync receiver и master-data authority foundation.
- `cloud-ui/` — Cloud web UI (admin/операционные экраны, см. `docs/ui/CLOUD-UI-SPEC.md`).
- `license-server/` — license/pairing support service.
- `shared/` — общие platform helpers.
- `scripts/` — единственный локальный seed script `seed-dev-system.py`.

## Главные документы

- `docs/CURRENT-FUNCTIONAL-STATE.md` — сводка фактически реализованного функционала, тестового покрытия и границ текущего runtime.
- `SPECv1.3.md` — frozen cashier pilot contract до первого pilot.
- `ROADMAP.md` — статусы, блокеры и следующий план.
- `docs/backend/CLOUD-BACKEND-SPEC.md` — фактический Cloud backend contract.
- `docs/backend/POS-BACKEND-SPEC.md` — фактический POS backend contract.
- `docs/backend/POS-DATA-AND-MIGRATIONS.md` — SQLite/PostgreSQL schema и migration policy.
- `docs/ui/POS-UI-SPEC.md` — фактический cashier UI contract.
- `docs/architecture/DDD-CONTEXT-MAP.md` — bounded contexts и ownership boundaries.
- `docs/adr/ADR-015-persistence-and-analytics-strategy.md` — решение по persistence/analytics strategy.
- `AGENTS.md` — только правила работы агентов и процесса разработки.

## Локальный запуск

Docker stack:

```bash
docker compose -f docker-compose.local.yml up --build -d
```

UI/E2E devbox с Playwright Chromium и Docker volumes для `node_modules`:

```bash
docker compose -f docker-compose.local.yml --profile devbox build devbox
docker compose -f docker-compose.local.yml --profile devbox up -d devbox
```

Подробный порядок запуска backend, seed `.e2e/bootstrap.json`, Vite и Playwright описан в `docs/backend/LOCAL-DOCKER-STACK.md`.

Единое заполнение локальной системы начальными данными:

```bash
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.seed-dev-system-summary.json
```

`scripts/seed-dev-system.py` является единственным пользовательским скриптом в `scripts`. Он создает полный набор текущих Cloud-owned справочников через HTTP API: ресторан, роли, сотрудников с PIN, залы и столы, catalog folders/parameters/tags/items, menu categories/items, modifier groups/options/bindings, pricing policies, recipe items и stop-list examples. После создания и публикации всех сущностей скрипт генерирует license pairing code для POS Edge, выполняет `pair-via-license`, проверяет POS read model и выводит `restaurant_id`, `node_device_id`, pairing code и все PIN-коды для проверки ролей.

Seed-вход содержит только пользовательские данные: названия, имена, PIN, цены, количества, места и права. `restaurant_id`, `role_id`, `employee_id`, `catalog_item_id`, `menu_item_id`, `node_device_id`, generated SKU и остальные технические значения берутся из backend responses или генерируются системой скрипта как производные значения.

`scripts/.seed-dev-system-summary.json` содержит локальные demo credentials и игнорируется git. Повторный запуск рассчитан на чистые backend volumes; если POS Edge уже paired, скрипт завершится fail-fast и попросит пересоздать локальные данные.
POS UI:

```powershell
cd pos-ui
npm install
npm run dev
```

POS backend:

```powershell
cd pos-backend
go mod tidy
go test ./...
```

Cloud backend:

```powershell
cd cloud-backend
go mod tidy
go test ./...
```

Cloud UI:

```powershell
cd cloud-ui
npm install
npm run dev
```

Скрипты `dev`/`build` для `cloud-ui` определены в `cloud-ui/package.json`; локальное заполнение данных выполняется только через `scripts/seed-dev-system.py`.

UI build:

```powershell
cd pos-ui
npm install
npm run build
```

## Документационное правило

Если код и документ расходятся, фактический runtime проверяется по коду и тестам. Документ после этого обновляется под подтвержденное поведение. Planned decisions должны быть явно помечены как `запланировано до пилота`, `запланировано далее`, `после пилота` или `вне текущего объема`, а не как реализованные функции.
