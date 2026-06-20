# SPECv1.3 — current POS runtime and full pilot target

Статус: актуализировано под фактический cashier/waiter UI runtime, минимальный backend-backed KDS ticket lifecycle и целевой полный пилот.

Этот документ фиксирует проверенный runtime surface, инварианты и явно принятые boundary decisions. Полный пилот включает cashier, manager, waiter, advanced KDS lifecycle, stop-list sale blocking, Cloud-managed setup, полный Cloud-owned складской движок и ClickHouse runtime как immutable/OLAP storage с Cloud API-ручками для OLAP-движка; все элементы полного пилота, которых нет в коде, считаются `запланировано далее`, пока не переведены в код, тесты, smoke-приемку и профильную документацию.

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
- основной cashier flow является нелицензируемым базовым runtime: локальные смены, меню, заказ, precheck, payment, final check, reprint и financial operation ledger должны работать без module entitlement, если Edge имеет локальные данные;
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
- cashier POS UI shell состоит из разделов `floor`, `order`, `activity`, `reports`, `cash`; storage/archive/backoffice, Cloud reporting, delivery runtime и settings не входят в operator-facing cashier flow;
- cashier POS UI top context показывает backend/session readiness: restaurant id, actor, node device, selected table/order, personal shift, cash session и backend session state; UI не создает отдельную бизнес-модель этих контекстов;
- cashier POS UI не показывает active commands для table/order transfer, split/fractional split, banquet/preorder, mock waiter filters или discount/surcharge editor без отдельного backend/API/UI contract;
- `/pos/waiter` реализован как mobile-first order/precheck UI route: выбор зала/стола, активные заказы, создание заказа, меню/поиск, добавление строк с модификаторами, quantity, void line, issue/reprint precheck; payment/refund/cash drawer authority не входит в default waiter surface;
- `/pos/kitchen` реализован как минимальный backend-backed KDS ticket lifecycle: tickets создаются из order lines, экран читает `GET /api/v1/kitchen/tickets`, status actions `accept/start/hold/ready/serve/recall/cancel` проходят backend RBAC, а UI перечитывает backend truth после команды;
- Edge -> Cloud operational outbox foundation;
- Cloud -> Edge master-data ingest for supported streams.
- POS Edge stop-list sale blocking при `AddOrderLine` и увеличении quantity по direct `catalog_item_id` и mandatory active recipe components из локальных `recipe_versions`/`recipe_lines`.
- Целевая Cloud-centric inventory architecture зафиксирована в `docs/backend/INVENTORY-COSTING-SPEC.md`. Реализовано сейчас: bounded worker, `stock_ledger`, materialized `stock-balances`, Cloud-состояние обработки для receipt/count/write-off/production и ограниченный async retro recalculation DAG/job lifecycle для costing fields. COGS/margin, production-grade balance rebuild и Cloud UI operator workflow не реализованы сейчас.

Цель полной пилотной реализации:

- cashier runtime остается обязательным базовым потоком и не расширяется фискализацией/PSP-интеграцией до отдельного решения;
- post-MVP бесплатный автономный POS Edge должен работать без внешнего Cloud: локальный владелец создает простые позиции меню на самом Edge, кассир продает их через базовый cashier flow, а подключение внешнего Cloud, tenant management, automatic delivery, analytics и расширенных рабочих пространств включается только покупкой лицензий;
- manager runtime должен позволять через Cloud UI подготовить ресторан, роли, сотрудников, зал/столы, catalog/menu/modifiers/pricing, recipes и stop-list, опубликовать master-data и увидеть readiness/sync состояние Edge;
- waiter runtime должен расширяться как mobile-first POS UI route для выбора стола, создания/изменения заказа, выбора модификаторов, выпуска и повторной печати precheck без права оплаты, если роль не имеет cashier payment permissions;
- advanced KDS runtime уже поддерживает ticket lifecycle `new -> accepted -> in_progress -> ready -> served` с ветками `hold`, `recall` и `cancelled`; POS Edge backend и `pos-ui-g` для receipt/count/write-off/production, recipe read, catalog/recipe proposal input и bounded stop-list edit form поверх `POST /api/v1/kitchen/stop-list-updates` реализованы сейчас, а расширения по станциям/приоритету, cooking events и production-grade stop-list review polish остаются `запланировано далее`;
- kitchen worker может принять поставку/ревизию/списание/заготовку на Edge и создать предложение нового catalog item; Cloud worker превращает stock events в Cloud-owned documents/ledger, а catalog/recipe proposals применяет только после manager approve;
- kitchen worker может видеть техкарту и отправлять `RecipeChangeSuggested` с заменой ингредиента, правкой количества/единицы/потерь и изменением prep time в пределах параметра `POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES`; правка не применяется на Edge и не меняет Cloud recipe до review/apply шага;
- kitchen worker должен видеть и редактировать stop-list; конфликт Cloud/Edge изменений разрешается параметром `stop_list_conflict_policy`, а один catalog item может быть добавлен Cloud-стороной и одновременно ограничен/исключен Edge-стороной через active overlay и `available_quantity`;
- waiter runtime должен быть единственным mobile layout в POS UI: mobile-first route `/pos/waiter` для залов, заказов и ограниченной аналитики; cashier/KDS/manager modes не получают отдельные mobile variants в полном пилоте;
- `waiter-space` лицензирует отдельный официантский доступ, а не shared order/precheck/payment backend. Текущие order routes используются кассиром и не должны закрываться целиком по `waiter-space`; backend enforcement нужен на waiter-only facade/route/worker или другом backend-owned признаке официантского контекста. UI route или frontend header не являются security boundary.
- stop-list является единственным механизмом runtime-блокировки продаж: POS Edge должен локально блокировать блюдо или обязательный recipe component из active stop-list даже offline;
- `CheckClosed` и `ItemServed` являются текущими inventory facts: Cloud принимает их через `sync/exchange`, дедуплицирует и передает Cloud Inventory Worker; при наличии уже принятого superseding `ItemServed` для той же order line Worker пропускает superseded served fact, а если старый served fact уже обработан, пишет append-only `ItemServedCompensation` return ledger перед новой подачей; `KitchenTicketStatusChanged` остается operational-only event без складской проводки;
- Cloud Inventory Engine должен закрыть полный пилотный складской контур: recipes, stop-list, stock receipts, inventory counts, production, consumption, cancellation/refund dispositions, stock ledger, stock documents, stock balances и costing/recalculation state;
- ClickHouse runtime должен хранить `raw_business_events` и OLAP projections, а Cloud API должен отдавать bounded OLAP/read-only ручки для продаж, склада, себестоимости и kitchen speed analytics.

Вне текущего объема полного пилота:

- hardware bump-bar integrations, kitchen printer orchestration and rich BI dashboards beyond bounded pilot OLAP/KDS metrics;
- delivery/channel integrations;
- real PSP/payment processor module;
- fiscal device integration;
- ERP/accounting integrations;
- `sqlc` as confirmed persistence implementation.

## Текущий Runtime Contract

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
- `GET /api/v1/orders/active`
- `GET /api/v1/orders/{id}`
- `GET /api/v1/orders/closed`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/modifiers`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/details`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `POST /api/v1/orders/{id}/close`
- `GET /api/v1/pricing/policies`

Order line snapshot содержит `menu_item_id`, `catalog_item_id`, name, quantity, unit price, total price и selected modifiers. `SelectedModifierCommand.Quantity` означает количество выбранной modifier option на всю строку заказа; line total считается как `unit_price * line.quantity + sum(selected_modifier.total_price)`. Add/update modifiers выполняются только для активной строки открытого заказа без active precheck/final check; backend проверяет required/min/max, active group/option, принадлежность option к group, link menu item -> modifier group и неотрицательную цену option.

`GET /api/v1/orders/closed` реализовано сейчас как bounded read для activity UI: default `limit=50`, max `limit=100`, `offset`, stable newest-first sort по закрытию и `id`, фильтры `business_date_local`, `from_business_date_local`, `to_business_date_local`, `shift_id`, `device_id`, `check_id`. Без фильтра API все равно возвращает только bounded latest page. `GET /api/v1/checks/{id}/financial-operations` и `GET /api/v1/financial-operations` также bounded: default `limit=50`, max `limit=200`, `offset`; общий ledger endpoint поддерживает `business_date_from`, `business_date_to`, `operation_type`, `shift_id`, `original_shift_id`, `check_id`. `GET /api/v1/sync/outbox` и `GET /api/v1/sync/local-events` также bounded: backend default `limit=100`, oversized/empty limit returns bounded default, POS UI запрашивает `limit=5`. `POST /api/v1/storage/retention/dry-run`, `POST /api/v1/storage/archive/export-plan`, `POST /api/v1/storage/archive/export`, `POST /api/v1/storage/archive/apply-plan` и `POST /api/v1/storage/archive/apply-readiness` используют единое exclusive правило cutoff: candidate closed orders имеют `checks.business_date_local < cutoff_business_date_local`; невалидный или будущий cutoff отклоняется либо возвращается как blocked apply-plan/readiness. `POST /api/v1/storage/archive/export-plan` реализовано сейчас как manifest-only readiness: возвращает `result_mode = plan_only`, deterministic table manifest, protected flags, active/open blockers и outbox blocking state без записи файлов. `POST /api/v1/storage/archive/export` реализовано сейчас как export-only readiness: создает JSONL archive/manifest для старых closed orders с `runtime_rows_deleted = false`, source node/device metadata если она есть в runtime и без удаления source rows. `POST /api/v1/storage/archive/verify` реализовано сейчас как non-destructive integrity check: проверяет manifest/version/SHA/counts, required row identity fields, business-date range, exclusive cutoff consistency, `runtime_rows_deleted = false`, immutable snapshot payload и summary-only policy для `local_event_log`/`pos_sync_outbox`. `POST /api/v1/storage/archive/read-plan` реализовано сейчас как bounded archived closed-order preview: default `limit=50`, max `limit=100`, `offset`, filters `business_date_local`, `order_id`, `check_id`, без восстановления в active SQLite и без sync/event payload JSON. `POST /api/v1/storage/archive/apply-readiness` реализовано сейчас как read-only policy gate: возвращает `ready_for_destructive_apply = true`, только если archive verified, scoped Edge -> Cloud outbox clean и нет open operational boundaries для cutoff. `POST /api/v1/storage/archive/apply-plan` реализовано сейчас как destructive apply: при verified archive и runtime safety выполняет физическое удаление scoped runtime rows, затем `VACUUM`; ответ возвращает `result_mode = destructive_apply`, `runtime_rows_deleted = true`. При нарушении safety gate apply-plan возвращает `result_mode = apply_blocked` и не удаляет runtime rows.

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
- `GET /api/v1/financial-operations`
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
- `GET /api/v1/checks/{id}/financial-operations?limit=&offset=` реализовано сейчас как read-only ledger view по конкретному final check под `pos.check.view`; `GET /api/v1/financial-operations?business_date_from=&business_date_to=&operation_type=&shift_id=&original_shift_id=&check_id=&limit=&offset=` реализован как backend-owned local reporting read. Activity UI показывает type, kind, amount, reason, employee/approver, business date, inventory disposition и created time.
- Inventory disposition фиксируется явно: `no_stock_effect`, `return_to_stock`, `write_off_waste`, `manual_review`.
- POS financial operation ledger сам по себе не создает Edge-side `stock_moves`. Реализовано сейчас: Cloud receiver ставит `CancellationRecorded`/`RefundRecorded` в durable `inventory_event_queue` только при `inventory_disposition != no_stock_effect`, а Cloud Inventory Worker асинхронно создает Cloud-owned `RETURN/IN` для `return_to_stock` и `WASTE/OUT` для `write_off_waste`.
- No-over-refund/no-over-cancel проверяется по сумме check; для `order_line` backend также проверяет selected line amount, уже записанную сумму по line и сумму уже записанных quantities по operation type.
- `CancellationRecorded` и `RefundRecorded` являются текущими Edge -> Cloud operational events для этих операций.

Boundary rules:

- Cancellation применяется в пределах открытой исходной personal shift/current cash session и той же `business_date_local`.
- Refund применяется после закрытия исходной personal shift или на более поздней `business_date_local`; для записи refund все равно нужна текущая open cash session.
- Refund денег не означает возврат товара на склад; stock effect задается только `inventory_disposition`. POS Edge не создает складские документы или проводки, а Cloud Inventory Worker применяет `return_to_stock`/`write_off_waste` append-only по immutable financial operation snapshot. `no_stock_effect` не создает складского эффекта; `manual_review` не создает автоматического движения и оставляет queue item в failure state для операторского разбора.
- Legacy events `PaymentRefunded` и `CheckRefunded` остаются распознаваемыми Cloud sync event types для старых payloads, но новый POS Edge runtime пишет `RefundRecorded`.
- Cloud receiver валидирует current `RefundRecorded`/`CancellationRecorded` payload shape: operation id, edge operation id, совпадение `restaurant_id`/`device_id` payload с envelope, check id, precheck id, current/original shift id, amount, currency, business date, inventory disposition, reason и snapshot; затем сохраняет raw/journal envelopes, обновляет event-type stats и coarse shift finance refund counters для refunds. Detailed Cloud projection `cloud_projection_financial_operations` реализована сейчас для current `CancellationRecorded`/`RefundRecorded` with operation/check/precheck/shift/date/type/disposition/reason/snapshot metadata and service/repository filters. `GET /api/v1/reporting/financial-operations?restaurant_id=&business_date_from=&business_date_to=&operation_type=&shift_id=&original_shift_id=&check_id=&limit=&offset=` реализовано сейчас как bounded read-only Cloud reporting API; Cloud UI показывает эту projection без raw sync payload, snapshot JSON, PIN/token/request dump и без Cloud cashier commands.

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
- Целевой Cloud runtime владеет `stock_documents`, `stock_ledger`, materialized balances, costing state, stop-list authority и очередью Inventory Worker. Реализовано сейчас: Cloud PostgreSQL baseline содержит `inventory_event_queue`, `stock_documents`, `stock_ledger`, `inventory_stock_balances`, `stock_recalculation_jobs`, `stop_lists`; Cloud Inventory Worker обрабатывает нормализованные item payloads с fallback costing `estimated` и транзакционно обновляет materialized balances.
- Legacy Edge-side manual stock document foundation использовался в pre-pilot runtime и удален при переходе на Cloud-centric Event-Driven Inventory.

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
- Effective Cloud master-data changes доставляются подключенным Edge автоматически через scheduled exchange. Operator publish запрещен; без назначенных Edge delivery packages не накапливаются, а first assignment собирает актуальный full batch.
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

- Modifier-to-recipe expansion в POS modifier runtime, Edge-side automatic stock consumption и return-to-stock stock moves относятся к recipes/inventory, а не к текущему modifier pricing runtime.

## Recipes And Inventory

Архитектурное решение заморожено:

- POS Edge и KDS являются генераторами immutable business events и не формируют складские документы, складские проводки или себестоимость.
- POS Edge backend является авторитетным runtime для offline order/precheck/payment/check commands, pricing snapshots, financial operation ledger, idempotency, cash/session boundaries, stop-list sale blocking и KDS command validation. POS UI не является авторитетным слоем и только отправляет команды/показывает ответы backend.
- Cloud является единственным source of truth для склада: Cloud receiver принимает Edge outbox, durable queue передает события Inventory Worker, Worker пишет `stock_documents`, `stock_ledger` и materialized `inventory_stock_balances` в PostgreSQL.
- Cloud остается авторитетным источником master/reference/configuration data, складских документов, stock ledger, materialized balances, costing/recalculation state, ClickHouse export и OLAP reads.
- ClickHouse используется как immutable business event archive и Cloud OLAP/reporting accelerator через batch projection `olap_stock_moves`; он не является transactional source of truth и не входит в POS transaction path.
- Остаток склада является аналитическим показателем, допускает отрицательные значения и не блокирует продажу.
- Продажу блокирует только `StopList`.
- Edge SQLite целевая схема содержит `recipe_versions`, `recipe_lines` read-only и `stop_lists`; Edge-side `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, `purchase_receipts`, `purchase_receipt_lines` удалены из целевого baseline.
- `StopList` содержит `catalog_item_id` и `available_quantity`; запись может относиться к блюду, ингредиенту или заготовке и синхронизируется Edge <-> Cloud.
- При добавлении позиции и увеличении quantity Edge локально разворачивает read-only active recipe version и блокирует продажу, если само блюдо или обязательный компонент находится в active stop-list с `available_quantity = 0` или `NULL`.
- Modifier на Edge остается ценовой опцией `modifier_option_id`; Cloud-only `ModifierOption.linked_catalog_item_id` приводит к отдельному списанию только в Inventory Worker. POS Edge хранит эту связь только как read-only reference и не получает stock authority.
- `CheckClosed` является финальным batch trigger для заказа; Worker делает delta consumption после сверки с уже обработанными KDS событиями `ItemServed`.
- `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured`, `ProductionCompleted` и `ItemServed` являются Edge/KDS input events, а не Edge stock documents.
- `KitchenTicketStatusChanged` является operational-only Edge/KDS event без Cloud Inventory Worker проводки. `CatalogItemChangeSuggested` и `RecipeChangeSuggested` реализованы сейчас на POS Edge как proposal events с локальным статусом `pending_sync`; Cloud review/apply реализован сейчас через manager approve/reject/request-changes routes. `StopListUpdated` реализован сейчас как Edge -> Cloud audit/projection event: receiver validates/enqueues, Inventory Worker writes safe projection without raw payload, and `stop_list_conflict_policy` supports `cloud_wins`, `edge_overlay_until_next_publication`, `edge_overlay_requires_manager_review` with default `edge_overlay_requires_manager_review`.
- `RefundRecorded` и `CancellationRecorded` передают operation-level `inventory_disposition`: `return_to_stock`, `write_off_waste`, `manual_review` или `no_stock_effect`. Текущий payload не содержит отдельного `items[].inventory_disposition`. Реализовано сейчас: Cloud нормализует `whole_check` и `order_line` из immutable snapshots; `service_charge`, `tip`, `payment` и `modifier_line` без authoritative linked catalog item не создают складское движение.
- License boundary для Edge -> Cloud: базовые cashier financial facts остаются частью подключенного Cloud-контура, но module-owned события не должны попадать в batch и worker processing при выключенном entitlement. `kitchen-space` закрывает KDS/kitchen events и proposals, `warehouse-mode` закрывает receipt/count/write-off/production и Cloud inventory worker, будущий `waiter-space` закрывает waiter-only commands/events после выделения backend-discriminated waiter surface. Выключение модуля не удаляет уже сохраненные локальные данные.

Inventory and costing logic:

- Реализовано сейчас: `StockReceiptCaptured`, `InventoryCountCaptured`, `StockWriteOffCaptured` и `ProductionCompleted` имеют Cloud-owned `inventory_document_processing_state` с уникальным `(restaurant_id, source_event_id, source_event_type)`. Повтор source event не создает повторные stock documents/ledger/balance mutations; safe validation failures фиксируются как `failed` state с безопасными failure code/message key без raw payload.
- Реализовано сейчас: `InventoryCountCaptured` приводит Cloud materialized balance к counted quantity через deterministic `IN`/`OUT` adjustment; если counted equals current, Cloud создает posted processing state с нулем ledger rows и без stock document.
- `ProductionCompleted` создает Cloud `PRODUCTION`: приходует заготовку и списывает сырье.
- Реализовано сейчас: если active recipe/cost basis для `ProductionCompleted` отсутствуют, worker сохраняет факт прихода готовой позиции `IN` и помечает costing visibility как `estimated`/`needs_recalculation`; bounded retro recalculation worker может позднее пересчитать affected costing fields, если появляется backdated trigger и reliable cost basis.
- Реализовано сейчас: продажа основной позиции разворачивается Cloud Inventory Worker по active recipe version, если она есть; иначе списывается сам `catalog_item_id`.
- Реализовано сейчас: selected modifier с Cloud-authoritative `linked_catalog_item_id` создает отдельное прямое `SALE/OUT` списание linked item; linked modifier item не разворачивается в recipe в этой итерации.
- Auto-production при продаже сначала списывает доступную заготовку, а недостающую часть split-списанием разворачивает по рецепту до сырья.
- `stock_ledger.unit_cost_minor` фиксирует себестоимость на момент события.
- Если списание уходит в минус без истории приходов, `unit_cost_minor = 0`.
- Если есть последняя известная цена, списание в минус использует ее для всего расхода, уводящего остаток ниже нуля.
- Приход задним числом влияет только на события начиная с даты приходного документа.
- Реализовано сейчас: документы receipt/count/production/write-off в прошлом запускают asynchronous recalculation job, если есть affected ledger rows позднее trigger date; worker строит deterministic DAG зависимостей `raw goods -> semi_finished -> dishes`, валидирует cycles safe failed job, проходит rows по `business_date_local`, `occurred_at`, `id` и обновляет только costing fields/status.

Профильный контракт:

- `docs/backend/INVENTORY-COSTING-SPEC.md` содержит schema target, ER target, event payloads и алгоритмы дедупликации, auto-production и costing.

Не реализовано сейчас:

- production-grade assignment/escalation для Edge-origin stop-list review и расширенный stop-list UX за пределами bounded KDS form;
- semi-finished auto-production split, COGS/margin, production-grade balance rebuild и Cloud UI operator workflow.

Запланировано до полного пилота:

- реализовано сейчас: Cloud authoring/UI для recipes и stop-list поверх поддержанного package contract/storage, включая route-backed recipe version editor и stop-list entries;
- smoke для Cloud package -> Edge sync -> offline sale blocking реализован сейчас в `scripts/seed-dev-system.py --run-minimal-flow`;
- полный kitchen/process smoke для Cloud seed, Edge sync, KDS recall/serve-again, ClickHouse trail, stock events, proposal approve и Edge feedback реализован сейчас в `scripts/seed-dev-system.py --run-kitchen-process-smoke`;
- реализовано сейчас: bounded Edge stop-list edit form и manager review flow поверх `StopListUpdated`; запланировано далее production-grade assignment/escalation, расширенный conflict UX и operator polish;
- full inventory engine beyond текущего bounded worker/recalculation slice: semi-finished auto-production split, production-grade purchase/receipt input, richer costing math и production-grade balance rebuild; bounded materialized balances, retro costing job/DAG lifecycle и refund/cancellation stock dispositions `return_to_stock`/`write_off_waste` реализованы сейчас без PSP/fiscal/COGS/margin;
- реализовано сейчас: ClickHouse first slice с managed schema `raw_business_events`, async forwarder `inbox_events -> raw_business_events`, export state, retry state и bounded metadata API;
- реализовано сейчас: первый ClickHouse stock moves slice с managed schema `olap_stock_moves`, async forwarder `stock_ledger -> olap_stock_moves`, checkpoint/retry state и bounded API `GET /api/v1/olap/stock-moves` без raw payload;
- реализовано сейчас: read-only `GET /api/v1/olap/export-status?stream=raw_business_events|stock_moves`, первый bounded агрегат `GET /api/v1/olap/stock-move-summary` по `olap_stock_moves` и минимальный support-only `POST /api/v1/olap/export-retry` для `retry_failed|resume_from_checkpoint` без raw payload и без synchronous ClickHouse dual-write;
- запланировано далее: промышленные backfill jobs/operator UI, sales/kitchen aggregates и costing-dependent COGS/margin;
- smoke-проверка offline waiter order flow, reconnect, Cloud inventory ledger, ClickHouse export и OLAP read endpoints из `CheckClosed`/`ItemServed`.

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

Замороженный принцип:

- Все Edge POS/KDS business events используют UUIDv7 `event_id`.
- Реализовано сейчас: Cloud API принимает Edge outbox batch в PostgreSQL `inbox_events` и отвечает без synchronous ClickHouse write.
- Реализовано сейчас: Async Batch Forwarder экспортирует `inbox_events` bounded batch от 1 до 100 000 rows в ClickHouse `raw_business_events`.
- ClickHouse `raw_business_events` хранит all business events бессрочно как immutable historical event archive; transactional source of truth остается PostgreSQL/runtime services.
- PostgreSQL остается transactional source of truth для текущего operational state.
- Реализовано сейчас: Cloud OLAP API читает bounded `raw_business_events` metadata, bounded `olap_stock_moves`, read-only export status, первый bounded `stock-move-summary` aggregate, первый bounded `sales-kitchen-summary`, bounded `kitchen-timing-summary` и backfill job state без raw payload. Transactional command validation остается в PostgreSQL/runtime services.
- `processed_for_olap = true` rows старше 3 месяцев можно удалять из PostgreSQL `inbox_events`.
- Synchronous dual-write в PostgreSQL и ClickHouse запрещен.

Запланировано до полного пилота:

- ClickHouse добавляется как immutable business event archive and OLAP/reporting accelerator, но не как transactional source of truth и не часть POS transaction path.
- Реализовано сейчас: Cloud OLAP API отдает bounded read-only metadata endpoint для event archive, bounded stock moves endpoint, stock move summary, sales/kitchen summary, kitchen timing summary, минимальный support-only export retry control и async backfill jobs foundation. Запланировано далее: COGS/margin и richer sales aggregates после достоверной cost basis.

Запланировано далее:

- `sqlc` можно рассматривать после стабилизации схемы и package boundaries.

Вне текущего объема:

- `sqlc` как уже внедренный persistence implementation.
- ручной ad-hoc SQL как canonical migration path.

## Document Boundaries

- `AGENTS.md` — правила работы агентов и процесса.
- `README.md` — короткий обзор, запуск и навигация.
- `ROADMAP.md` — статусы, этапы, блокеры и следующий план.
- `SPECv1.3.md` — текущий cashier runtime contract и целевой полный pilot contract.
- `docs/backend/*` — backend/data contracts.
- `docs/ui/*` — UI contracts.
- `docs/architecture/*` — bounded contexts и dependency direction.

## Pilot pricing/tax policy flow

Реализовано сейчас:

- Cloud-authored `pricing_policy` доставляется на Edge с `manual`, `requires_permission`, `application_index`, amount fields и lifecycle-derived `active`.
- POS runtime применяет скидки/надбавки по `pricing_policy_id`, копирует расчетные поля из policy и сохраняет `pricing_policy_id` в runtime adjustment и precheck breakdown.
- Текущий cashier UI показывает backend-provided totals скидок/надбавок, но не содержит active discount/surcharge editor.
- Canonical calculation pipeline остается `order lines subtotal -> ordered discounts/surcharges by application_index -> taxable base -> taxes -> grand total`.
- Tax Always Last: налоги считаются только после всех discounts/surcharges.

Вне текущего объема:

- loyalty/promocodes;
- dynamic pricing;
- fiscal adapter;
- UI-side authoritative financial calculation.
