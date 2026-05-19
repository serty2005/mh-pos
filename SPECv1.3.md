# SPECv1.3 — frozen cashier pilot contract and inventory target

Статус: заморожено до первого cashier pilot.

Этот документ фиксирует проверенный pilot surface, инварианты и явно принятые pre-pilot boundary decisions. Раздел склада/себестоимости фиксирует замороженную целевую архитектуру для будущей реализации; она не считается текущим runtime, пока не переведена в код, тесты и профильную документацию.

## Источники фактов

Реализовано сейчас подтверждается кодом и миграциями:

- `pos-backend/internal/pos/api/router.go`
- `pos-backend/internal/pos/app/precheck/service.go`
- `pos-backend/internal/pos/app/check/service.go`
- `pos-backend/internal/pos/app/check/financial_operations.go`
- `pos-backend/internal/pos/app/mastersync/service.go`
- `pos-backend/migrations/sqlite/001_init.sql`
- `cloud-backend/migrations/postgres/001_init.sql`
- `docs/adr/ADR-015-persistence-and-analytics-strategy.md`

Если этот документ конфликтует с кодом, источником истины является код.

## Pilot Scope

Реализовано сейчас:

- cashier-first POS Edge runtime;
- PIN login/session/RBAC;
- personal employee shifts;
- cash sessions and cash drawer events;
- halls/tables read model;
- menu/catalog read model;
- order create/read, active lines with selected modifiers, backend-authoritative modifier validation, modifier edit for active open lines, quantity change, void line;
- `Order -> Precheck -> Payment -> Check`;
- service catalog items as sellable POS items;
- cashier modifier selection/edit flow for menu items with modifier groups;
- controlled precheck/check reprint from immutable snapshots;
- append-only cancellation/refund ledger, cashier UI для full whole-check и partial `order_line`/quantity cancellation/refund с явным `inventory_disposition` и compatibility payment refund fallback;
- Edge -> Cloud operational outbox foundation;
- Cloud -> Edge master-data ingest for supported streams.
- Целевая Cloud-centric inventory architecture зафиксирована в `docs/backend/INVENTORY-COSTING-SPEC.md`, но runtime engine не реализован сейчас.

Вне текущего объема:

- KDS runtime;
- delivery/channel integrations;
- real PSP/payment processor module;
- fiscal device integration;
- full inventory engine runtime;
- ClickHouse runtime pipeline;
- `sqlc` as confirmed persistence implementation.

## Current Runtime Contract

### Shift And Cash Session

Реализовано сейчас:

- `POST /api/v1/employee-shifts/open`
- `POST /api/v1/employee-shifts/{id}/close`
- `GET /api/v1/employee-shifts/current`
- `GET /api/v1/employee-shifts/recent`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `GET /api/v1/cash-shifts/current`

Инварианты:

- Personal employee shift (`shifts`) и device cash session (`cash_sessions`) являются разными runtime concepts.
- `GET /api/v1/employee-shifts/current` ищет открытую личную смену по authenticated employee в restaurant context; query/header `node_device_id` является session/device metadata, а не ключом выбора личной смены.
- Если у authenticated employee нет открытой личной смены, `GET /api/v1/employee-shifts/current` возвращает `200 null`.
- `GET /api/v1/cash-shifts/current` ищет открытую cash session по authenticated device context; empty state для cash session остается `404 NOT_FOUND` и трактуется POS UI как optional `null`.

### Order

Реализовано сейчас:

- `POST /api/v1/orders`
- `GET /api/v1/orders/current`
- `GET /api/v1/orders/{id}`
- `GET /api/v1/orders/closed`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/modifiers`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `POST /api/v1/orders/{id}/close`

Order line snapshot содержит `menu_item_id`, `catalog_item_id`, name, quantity, unit price, total price и selected modifiers. `SelectedModifierCommand.Quantity` означает количество выбранной modifier option на всю строку заказа; line total считается как `unit_price * line.quantity + sum(selected_modifier.total_price)`. Add/update modifiers выполняются только для активной строки открытого заказа без active precheck/final check; backend проверяет required/min/max, active group/option, принадлежность option к group, link menu item -> modifier group и неотрицательную цену option.

`GET /api/v1/orders/closed` реализовано сейчас как bounded read для activity UI: default `limit=50`, max `limit=100`, `offset`, stable newest-first sort по закрытию и `id`, фильтры `business_date_local`, `from_business_date_local`, `to_business_date_local`, `shift_id`, `device_id`, `check_id`. Без фильтра API все равно возвращает только bounded latest page. `GET /api/v1/sync/outbox` и `GET /api/v1/sync/local-events` также bounded: backend default `limit=100`, oversized/empty limit returns bounded default, POS UI запрашивает `limit=5`. `POST /api/v1/storage/archive/export` реализовано сейчас только как export-only readiness: создает JSONL archive/manifest для старых closed orders без удаления source rows. Destructive retention/apply, restore/read из архива и compaction закрытых заказов запланированы далее и не являются текущим runtime.

### Precheck

Реализовано сейчас:

- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/orders/{id}/prechecks`
- `GET /api/v1/prechecks/{id}`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/reprint`

Инварианты:

- `IssuePrecheck` разрешен только для `open` order.
- Active issued precheck блокирует order и переводит order в `locked`.
- Precheck snapshot immutable и используется как source для reprint.
- В текущем коде subtotal, discounts, surcharges, taxes и grand total считаются backend `Pricing` boundary по active order lines и сохраненным order pricing adjustments.
- Snapshot сохраняет `currency_code`, `subtotal_minor`, `discount_total_minor`, `surcharge_total_minor`, `tax_total_minor`, `grand_total_minor`, `paid_total_minor`, `remaining_total_minor` и breakdown строк/скидок/надбавок/налогов.
- `CancelPrecheck` требует manager employee id, manager PIN, reason и permissions `pos.precheck.cancel.request` + `pos.precheck.cancel`.
- Cancel unpaid active precheck переводит order обратно в `open`.

### Payment And Check

Реализовано сейчас:

- `POST /api/v1/prechecks/{id}/payments`
- `POST /api/v1/payments/{id}/refund`
- `GET /api/v1/checks/{id}`
- `GET /api/v1/checks/{id}/financial-operations`
- `POST /api/v1/checks/{id}/reprint`
- `POST /api/v1/checks/{id}/cancellations`
- `POST /api/v1/checks/{id}/refunds`

Инварианты:

- Payment ссылается на `precheck_id`, а не на legacy `check_id`.
- Методы оплаты: `cash`, `card`, `other`.
- Manual/trusted card capture допустим для pilot: cashier проводит оплату на автономном терминале и фиксирует факт оплаты в POS.
- Provider metadata (`provider_name`, `provider_transaction_id`, `provider_reference`, `fingerprint_hash`) существует как metadata, а не как подтверждение PSP module.
- Partial payments разрешены до суммы precheck total.
- Final check создается только после полной оплаты active precheck.
- Check snapshot включает precheck snapshot и payments snapshot.
- Check snapshot сохраняет selected modifiers через immutable precheck snapshot; reprint/refund не обращаются к текущему каталогу для восстановления старых modifiers.
- Finalized check/payment/precheck после создания final check не переписываются cancellation/refund flow.
- `/payments/{id}/refund` является compatibility wrapper поверх `/checks/{id}/refunds`: он требует captured payment, finalized check and open current cash session, then records refund operation scope `payment`; статус payment остается `captured`.

### Cancellation / Refund Boundary

Реализовано сейчас:

- Cancellation и refund являются разными operation types в append-only ledger `financial_operations`.
- Operation kind: `full` или `partial`.
- Operation item scopes: `whole_check`, `order_line`, `modifier_line`, `service_charge`, `tip`, `payment`.
- Whole check, specific order line, quantity of order line и split payment allocation проверяются backend service.
- Modifier/service/tip scopes поддержаны как ledger scopes с explicit snapshot; cashier UI уже поддерживает выбор modifiers в заказе, но отдельного UI для partial modifier/service/tip cancellation/refund сейчас нет.
- Cashier UI реализует full whole-check cancellation/refund и partial `order_line`/quantity cancellation/refund поверх ledger endpoints. Line/quantity варианты строятся из immutable `check.snapshot.precheck_snapshot.lines`, но backend остается финальным enforcement layer для суммы, количества, смены и business date.
- Cashier UI для whole-check операций отправляет `command_id`, `operation_kind`, `inventory_disposition` и reason без `items[]`; backend записывает `whole_check` item из immutable check snapshot. Для `order_line`/quantity UI отправляет `items[]` со scope `order_line`, `order_line_id`, `quantity`, `amount`, `currency` и `tax_amount`.
- `GET /api/v1/checks/{id}/financial-operations` реализовано сейчас как read-only ledger view по конкретному final check под `pos.check.view`; activity UI показывает type, kind, amount, reason, employee/approver, business date, inventory disposition и created time.
- Inventory disposition фиксируется явно: `no_stock_effect`, `return_to_stock`, `write_off_waste`, `manual_review`.
- Financial operation не создает `stock_moves` автоматически.
- No-over-refund/no-over-cancel проверяется по сумме check; для `order_line` backend также проверяет selected line amount, уже записанную сумму по line и сумму уже записанных quantities по operation type.
- `CancellationRecorded` и `RefundRecorded` являются текущими Edge -> Cloud operational events для этих операций.

Boundary rules:

- Cancellation применяется в пределах открытой исходной personal shift/current cash session и той же `business_date_local`.
- Refund применяется после закрытия исходной personal shift или на более поздней `business_date_local`; для записи refund все равно нужна текущая open cash session.
- Refund денег не означает возврат товара на склад; stock effect задается только `inventory_disposition` и требует отдельного inventory service, которого в cashier runtime сейчас нет.
- Legacy events `PaymentRefunded` и `CheckRefunded` остаются распознаваемыми Cloud sync event types для старых payloads, но новый POS Edge runtime пишет `RefundRecorded`.
- Cloud receiver validates current `RefundRecorded`/`CancellationRecorded` payload shape for operation id, check id, current/original shift id, amount, currency, business date, inventory disposition and snapshot, stores raw/journal envelopes, updates event-type stats and updates coarse shift finance refund counters for refunds. It is not a full financial operation reporting projection by item scope, inventory disposition or approval policy.

Не реализовано сейчас:

- отдельные runtime aggregates `business_day` и `fiscal_shift`;
- отдельный aggregate `cashier_shift`; текущий cashier shift представлен personal employee shift/table `shifts`;
- fiscal receipt/correction document generation;
- PSP refund integration;
- cashier UI для modifier/service/tip partial cancellation/refund;
- automatic refund-to-original-tender policy beyond captured payment allocation cap.

## Master Data And Sync

Реализовано сейчас:

- Edge read model принимает Cloud -> Edge master data через:
  - `POST /api/v1/sync/master-data/snapshots`
  - `POST /api/v1/sync/master-data/{stream}`
- POS Edge ingest поддерживает только streams:
  - `restaurants`
  - `devices`
  - `staff`
  - `floor`
  - `catalog`
  - `menu`
  - `pricing_policy`
- `pricing_policy` применяет Cloud-authored `tax_profiles`, `tax_rules`, `service_charge_rules` и automatic discount/surcharge `pricing_policies` как reference/read-model data с sync metadata.
- `restaurants` применяет Cloud-authored настройки ресторана и `active`; опубликованный active restaurant сохраняется в Edge read model как active row.
- `catalog` применяет catalog folders, folder parameters, catalog tags, item tags, catalog item kinds `dish`, `good`, `semi_finished`, `service`, modifier groups/options/bindings и effective menu item modifier group links.
- `menu` применяет menu items и menu-visible `item_type`.
- Unknown JSON fields и unsupported stream names отклоняются до partial apply.

Реализовано сейчас / основа:

- Целевая Edge inventory схема содержит только `recipe_versions`, `recipe_lines` в read-only режиме и двусторонний overlay `stop_lists`.
- Целевая Edge schema не должна содержать `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, `purchase_receipts`, `purchase_receipt_lines`.
- Целевой Cloud runtime владеет `stock_documents`, `stock_ledger`, costing state, stop-list authority и очередью Inventory Worker. Реализовано сейчас: Cloud PostgreSQL baseline содержит foundation tables `stock_documents`, `stock_ledger`, `stock_recalculation_jobs`, `stop_lists`; full Inventory Worker остается вне текущего runtime.
- Legacy Edge-side manual stock document foundation должен быть удален при переходе на Cloud-centric Event-Driven Inventory.

## Pricing, Discounts And Tax

Реализовано сейчас:

- В runtime есть отдельный domain/application boundary `Pricing`; он не смешан с Order, Payment или Catalog aggregate.
- Backend считает authoritative totals; UI отправляет команды и отображает результат, но не является financial authority.
- Канонический pipeline:
  - order lines subtotal;
  - unified ordered modifiers pipeline: discounts и surcharges по `application_index ASC`;
  - taxable base;
  - taxes;
  - grand total.
- Все discounts и surcharges используют единое пространство `application_index`; duplicate index между скидкой и надбавкой одного расчета отклоняется.
- Surcharge является отдельной доменной семантикой и не реализуется как negative discount.
- Tax Always Last является текущим backend-инвариантом: налоги считаются только после всех discounts/surcharges.
- Все persistent money values хранятся как `INTEGER` minor units; проценты хранятся как basis points.
- Rounding policy: deterministic integer half-up minor units (`integer_half_up_minor_units_v1`), без float/decimal runtime math.
- Поддержаны line/order manual discounts, percentage/fixed discounts, manual/service/PB1 surcharge foundation, percentage/fixed surcharge.
- Discount cannot exceed target amount; negative line/precheck/check totals rejected by domain calculation.
- Tax foundation поддерживает `tax_profiles`, `tax_rules`, percentage/fixed components, inclusive/exclusive mode, multiple components per line, compound tax foundation и tax exempt profile foundation; inclusive tax попадает в `tax_total_minor`, но не увеличивает `grand_total_minor`.
- `GET /api/v1/orders/{id}/pricing` возвращает calculated preview.
- `POST /api/v1/orders/{id}/discounts` и `POST /api/v1/orders/{id}/surcharges` сохраняют order pricing adjustments для open order; оба payload требуют `application_index`.
- `IssuePrecheck` сохраняет immutable financial snapshot и persistence breakdown в `precheck_lines`, `precheck_discounts`, `precheck_surcharges`, `precheck_taxes`.
- Selected modifiers участвуют в line subtotal и сохраняются в order/precheck/check snapshots.
- Service items продаются как обычные catalog/menu items с ручной ценой и без recipe semantics.
- Cloud-authored tax/service-charge/automatic discount-surcharge reference data приходит через `pricing_policy`.
- Edge manual adjustments остаются runtime-командами с backend permission checks.

Запланировано до пилота:

- Runtime adjustments должны ссылаться на synced policy ids там, где центральная policy уже существует.
- Full Cloud UI/publication workflow для pricing/tax policy должен быть доведен отдельно; текущий шаг подтверждает только generic package storage/apply для `pricing_policy`.
- Manual line override / manual amount override допускается только при явном разрешении policy, отдельном permission boundary и audit trail.

Вне текущего объема до реализации:

- fiscal/legal tax adapter.

## Modifiers

Реализовано сейчас:

- Modifiers являются частью `Catalog/Menu` master data на Cloud и синхронизируются на Edge через supported payload sections.
- Menu item может ссылаться на один или несколько modifier groups; groups/options также могут иметь binding foundation через menu item, catalog item, folder или tag.
- Modifier group хранит `required`, `min_count`, `max_count`, sort order и lifecycle status.
- Modifier option хранит name, real modifier price, currency, active/status и sort order; modifier без техкарты и с нулевой ценой допустим.
- POS Edge order line model хранит selected modifiers в `order_line_modifiers`; открытая активная строка поддерживает полную замену набора modifiers через backend API.
- Modifier price impact входит в authoritative backend calculation; UI не является источником истины для цен, налогов или итогов.
- Precheck/check snapshots и reprint responses содержат выбранные modifiers с name, quantity, unit price и total price; reprint не читает текущий каталог для восстановления старых modifiers.
- Cashier UI открывает modifier selection dialog для menu item с groups при добавлении и редактировании активной строки, инициализирует текущие selected modifiers, показывает required/min/max UX validation и отправляет selected modifiers в backend.

Запланировано далее:

- Rich partial cancellation/refund UI для scope `modifier_line`, если pilot acceptance потребует отдельные операции по конкретным modifier строкам.

Вне текущего объема:

- Modifier-to-recipe expansion, automatic stock consumption и return-to-stock stock moves относятся к recipes/inventory, а не к текущему modifier runtime.

## Recipes And Inventory

Архитектурное решение заморожено:

- POS Edge и KDS являются генераторами immutable business events и не формируют складские документы, складские проводки или себестоимость.
- Cloud является единственным source of truth для склада: Cloud receiver принимает Edge outbox, durable queue передает события Inventory Worker, Worker пишет `stock_documents` и `stock_ledger` в PostgreSQL.
- ClickHouse используется как immutable business event archive и Cloud OLAP/reporting accelerator через batch projection `olap_stock_moves`; он не является transactional source of truth и не входит в POS transaction path.
- Остаток склада является аналитическим показателем, допускает отрицательные значения и не блокирует продажу.
- Продажу блокирует только `StopList`.
- Edge SQLite целевая схема содержит `recipe_versions`, `recipe_lines` read-only и `stop_lists`; Edge-side `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, `purchase_receipts`, `purchase_receipt_lines` должны быть удалены из целевого baseline.
- `StopList` содержит `catalog_item_id` и `available_quantity`; запись может относиться к блюду, ингредиенту или заготовке и синхронизируется Edge <-> Cloud.
- При добавлении позиции Edge локально разворачивает read-only рецептуру и блокирует продажу, если само блюдо или обязательный компонент находится в stop-list с `available_quantity = 0`.
- Modifier на Edge остается ценовой опцией `modifier_option_id`; Cloud-only `ModifierOption.linked_catalog_item_id` приводит к отдельному списанию только в Inventory Worker.
- `CheckClosed` является финальным batch trigger для заказа; Worker делает delta consumption после сверки с уже обработанными KDS событиями `ItemServed`.
- `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted` и `ItemServed` являются Edge/KDS input events, а не Edge stock documents.
- `RefundRecorded` и `CancellationRecorded` должны передавать operation-level `inventory_disposition`: `return_to_stock`, `write_off_waste`, `manual_review` или `no_stock_effect`. Текущий payload не содержит отдельного `items[].inventory_disposition`.

Inventory and costing logic:

- `ProductionCompleted` создает Cloud `PRODUCTION`: приходует заготовку и списывает сырье.
- Auto-production при продаже сначала списывает доступную заготовку, а недостающую часть split-списанием разворачивает по рецепту до сырья.
- `stock_ledger.unit_cost_minor` фиксирует себестоимость на момент события.
- Если списание уходит в минус без истории приходов, `unit_cost_minor = 0`.
- Если есть последняя известная цена, списание в минус использует ее для всего расхода, уводящего остаток ниже нуля.
- Приход задним числом влияет только на события начиная с даты приходного документа.
- Документы в прошлом запускают asynchronous recalculation job; worker строит DAG зависимостей `raw goods -> semi_finished -> dishes` и пересчитывает журнал хронологически.

Профильный контракт:

- `docs/backend/INVENTORY-COSTING-SPEC.md` содержит schema target, ER target, event payloads и алгоритмы дедупликации, auto-production и costing.

Не реализовано сейчас:

- Cloud Inventory Worker;
- `stock_ledger` и `stock_recalculation_jobs`;
- `stop_lists` sync Edge <-> Cloud;
- KDS `ItemServed` / `ProductionCompleted` runtime;
- ClickHouse `olap_stock_moves` projection;
- удаление legacy Edge-side stock tables из текущего runtime baseline.

## Payment Processor And Fiscal Boundary

Реализовано сейчас:

- Manual/trusted capture model для `cash`, `card`, `other`.
- Provider/reference fields являются payment metadata.
- Нет подтвержденного payment processor module.
- Нет fiscal adapter/fiscalization module.
- Final check считается finalized internal POS document после полной оплаты, но не fiscalized document.
- Reprint precheck/check является POS snapshot reprint; fiscal document reprint сейчас не реализован.

Запланировано далее:

- Payment processor boundary и fiscal adapter boundary должны быть отдельными архитектурными зонами.
- Payment processor отвечает за authorization/capture/refund integration с провайдером.
- Fiscal adapter отвечает за legal/fiscal receipt mapping и устройство/сервис фискализации.
- Нельзя смешивать PSP state и fiscal/legal receipt state в одной модели.
- Void/correction/refund fiscal document semantics должны появиться только вместе с fiscal adapter contract.

## Persistence And Analytics

Реализовано сейчас:

- POS Edge использует SQLite.
- Cloud backend использует PostgreSQL.
- Persistence code написан вручную в infrastructure repositories.
- Managed SQL files и startup migration/verification являются текущим canonical path.

Freezed Principle:

- Все Edge POS/KDS business events используют UUIDv7 `event_id`.
- Cloud API принимает Edge outbox batch в PostgreSQL `inbox_events` и отвечает без synchronous ClickHouse write.
- Async Batch Forwarder экспортирует `inbox_events` batch от 1 000 до 100 000 rows в ClickHouse `raw_business_events`.
- ClickHouse `raw_business_events` хранит all business events бессрочно и является source of truth для historical business event trail.
- PostgreSQL остается transactional source of truth для текущего operational state.
- `processed_for_olap = true` rows старше 3 месяцев можно удалять из PostgreSQL `inbox_events`.
- Synchronous dual-write в PostgreSQL и ClickHouse запрещен.

Запланировано далее:

- `sqlc` можно рассматривать после стабилизации схемы и package boundaries.
- ClickHouse добавляется как immutable business event archive and OLAP/reporting accelerator, но не как transactional source of truth и не часть POS transaction path.

Вне текущего объема:

- ClickHouse runtime dependency.
- `sqlc` как уже внедренный persistence implementation.
- ручной ad-hoc SQL как canonical migration path.

## Document Boundaries

- `AGENTS.md` — правила работы агентов и процесса.
- `README.md` — короткий обзор, запуск и навигация.
- `ROADMAP.md` — статусы, этапы, блокеры и следующий план.
- `SPECv1.3.md` — frozen pilot contract.
- `docs/backend/*` — backend/data contracts.
- `docs/ui/*` — UI contracts.
- `docs/architecture/*` — bounded contexts и dependency direction.

## Pilot pricing/tax policy flow

Реализовано сейчас:

- Cloud-authored `pricing_policy` доставляется на Edge с `manual`, `requires_permission`, `application_index`, amount fields и lifecycle-derived `active`.
- POS runtime применяет скидки/надбавки по `pricing_policy_id`, копирует расчетные поля из policy и сохраняет `pricing_policy_id` в runtime adjustment и precheck breakdown.
- Canonical calculation pipeline остается `order lines subtotal -> ordered discounts/surcharges by application_index -> taxable base -> taxes -> grand total`.
- Tax Always Last: налоги считаются только после всех discounts/surcharges.

Вне текущего объема:

- loyalty/promocodes;
- dynamic pricing;
- fiscal adapter;
- UI-side authoritative financial calculation.
