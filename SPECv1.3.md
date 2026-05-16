# SPECv1.3 — frozen cashier pilot contract

Статус: заморожено до первого cashier pilot.

Этот документ фиксирует только проверенный pilot surface, инварианты и явно принятые pre-pilot boundary decisions. Дальняя архитектура не считается частью текущего контракта, пока не переведена в код, тесты и профильную документацию.

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
- order create/read, active lines with selected modifiers, quantity change, void line;
- `Order -> Precheck -> Payment -> Check`;
- service catalog items as sellable POS items;
- cashier modifier selection flow for menu items with modifier groups;
- controlled precheck/check reprint from immutable snapshots;
- append-only cancellation/refund ledger, pilot-minimum full check cancellation/refund UI с явным `inventory_disposition` и compatibility payment refund fallback;
- Edge -> Cloud operational outbox foundation;
- Cloud -> Edge master-data ingest for supported streams.

Вне текущего объема:

- KDS runtime;
- delivery/channel integrations;
- real PSP/payment processor module;
- fiscal device integration;
- full inventory engine;
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
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `POST /api/v1/orders/{id}/close`

Order line snapshot содержит `menu_item_id`, `catalog_item_id`, name, quantity, unit price, total price и selected modifiers. `SelectedModifierCommand.Quantity` означает количество выбранной modifier option на всю строку заказа; line total считается как `unit_price * line.quantity + sum(selected_modifier.total_price)`.

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
- Cashier UI реализует только full check cancellation/refund поверх ledger endpoints; partial line/quantity/modifier/service/tip UI остается вне текущего runtime.
- Cashier UI для full check операций отправляет `command_id`, `operation_kind`, `inventory_disposition` и reason; UI не рассчитывает authoritative amount/items и не передает full operation items.
- Inventory disposition фиксируется явно: `no_stock_effect`, `return_to_stock`, `write_off_waste`, `manual_review`.
- Financial operation не создает `stock_moves` автоматически.
- No-over-refund/no-over-cancel проверяется по сумме check; для order line quantity проверяется сумма уже записанных quantities по operation type.
- `CancellationRecorded` и `RefundRecorded` являются текущими Edge -> Cloud operational events для этих операций.

Boundary rules:

- Cancellation применяется в пределах открытой исходной personal shift/current cash session и той же `business_date_local`.
- Refund применяется после закрытия исходной personal shift или на более поздней `business_date_local`; для записи refund все равно нужна текущая open cash session.
- Refund денег не означает возврат товара на склад; stock effect задается только `inventory_disposition` и требует отдельного inventory service, которого в cashier runtime сейчас нет.
- Legacy events `PaymentRefunded` и `CheckRefunded` остаются распознаваемыми Cloud sync event types для старых payloads, но новый POS Edge runtime пишет `RefundRecorded`.
- Cloud receiver stores raw/journal envelopes for `RefundRecorded`, updates event-type stats and updates coarse shift finance refund counters. It is not a full financial operation reporting projection by item scope, inventory disposition or approval policy.

Не реализовано сейчас:

- отдельные runtime aggregates `business_day` и `fiscal_shift`;
- отдельный aggregate `cashier_shift`; текущий cashier shift представлен personal employee shift/table `shifts`;
- fiscal receipt/correction document generation;
- PSP refund integration;
- cashier UI для line/quantity/modifier/service/tip partial cancellation/refund;
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
- `catalog` применяет catalog folders, folder parameters, catalog tags, item tags и catalog item kinds `dish`, `good`, `semi_finished`, `service`.
- `menu` применяет menu items, menu-visible `item_type`, modifier groups/options and menu item modifier group links.
- Unknown JSON fields и unsupported stream names отклоняются до partial apply.

Реализована только основа:

- SQLite schema содержит `recipe_versions`, `recipe_lines`, `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`.
- Cloud schema содержит recipe/inventory-adjacent foundation для recipe items, semi-finished products и publications.
- Наличие recipe/inventory таблиц и типов не означает готовый POS runtime для recipe expansion или stock consumption.

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
- POS Edge order line model хранит selected modifiers.
- Modifier price impact входит в authoritative backend calculation.
- Precheck/check snapshots содержат выбранные modifiers и их финансовый эффект.
- Cashier UI открывает modifier selection dialog для menu item с groups, отправляет selected modifiers в backend и отображает выбранные modifiers в активном заказе.

Запланировано далее:

- Печатные формы, reporting projections и audit details для modifiers должны быть уточнены отдельно под pilot acceptance.
- Modifier-to-recipe expansion относится к recipes/inventory, а не к текущему modifier runtime.

## Recipes And Inventory

Реализована только основа:

- SQLite содержит `recipe_versions`, `recipe_lines`, `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, `purchase_receipts`, `purchase_receipt_lines`.
- Cloud schema содержит `cloud_recipe_items`, `cloud_semi_finished_products` и catalog kind foundation.
- Recipe validation запрещает `dish` как компонент; approved components: `good` и `semi_finished`.
- `stock_moves` защищены append-only triggers.

Не реализовано сейчас:

- automatic stock consumption engine;
- recipe expansion runtime;
- modifier-to-recipe expansion;
- inventory consumption trigger from check/KDS;
- automatic stock return on cancellation/refund;
- stock movement app services for cashier runtime.

Запланировано до пилота как boundary decision, если inventory входит в pilot acceptance:

- Recipe должна быть versioned сущностью.
- Dish, modifier option и semi-finished/preparation могут ссылаться на recipe/preparation semantics.
- Recommended pilot policy: consumption after final check creation, если `KDS/DishServed` не введен как обязательный runtime trigger.
- Для semi-finished/preparation сначала списывается баланс заготовки; разворачивание в компоненты при нехватке разрешается только если эта policy отдельно утверждена.
- Order/precheck/check snapshots должны хранить достаточно данных, чтобы inventory/fiscal/reporting logic не зависела от текущего menu state.
- Inventory changes должны происходить только через immutable stock documents / stock moves, а не прямым mutation counters.
- Cancellation/refund flow может фиксировать только `inventory_disposition`; фактическое движение склада должно выполняться отдельным Inventory service.

После пилота:

- KDS-driven `DishServed` trigger;
- batch/FIFO/AVCO costing;
- procurement workflow beyond foundation.

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

Запланировано далее:

- `sqlc` можно рассматривать после стабилизации схемы и package boundaries.
- ClickHouse может быть добавлен только как Cloud OLAP/reporting accelerator, не source of truth и не часть POS transaction path.

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
