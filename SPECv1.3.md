# SPECv1.3 — frozen cashier pilot contract

Статус: заморожено до первого cashier pilot.

Этот документ фиксирует только проверенный pilot surface, инварианты и явно принятые pre-pilot boundary decisions. Дальняя архитектура не считается частью текущего контракта, пока не переведена в код, тесты и профильную документацию.

## Источники фактов

Реализовано сейчас подтверждается кодом и миграциями:

- `pos-backend/internal/pos/api/router.go`
- `pos-backend/internal/pos/app/precheck/service.go`
- `pos-backend/internal/pos/app/check/service.go`
- `pos-backend/internal/pos/app/mastersync/service.go`
- `pos-backend/migrations/sqlite/001_init.sql`
- `cloud-backend/migrations/postgres/004_master_data_authority.sql`
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
- payment refund route and UI flow;
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

Инварианты:

- Payment ссылается на `precheck_id`, а не на legacy `check_id`.
- Методы оплаты: `cash`, `card`, `other`.
- Manual/trusted card capture допустим для pilot: cashier проводит оплату на автономном терминале и фиксирует факт оплаты в POS.
- Provider metadata (`provider_name`, `provider_transaction_id`, `provider_reference`, `fingerprint_hash`) существует как metadata, а не как подтверждение PSP module.
- Partial payments разрешены до суммы precheck total.
- Final check создается только после полной оплаты active precheck.
- Refund переводит captured payment в `refunded`, уменьшает `paid_total` precheck и, если check уже есть, уменьшает `paid_total` check.
- Refund пишет подтвержденные Edge -> Cloud operational events `PaymentRefunded` и, если затронут final check, `CheckRefunded`.
- Check snapshot включает precheck snapshot и payments snapshot.
- Check snapshot сохраняет selected modifiers через immutable precheck snapshot; reprint/refund не обращаются к текущему каталогу для восстановления старых modifiers.

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
- `catalog` применяет catalog folders, folder parameters, catalog tags, item tags и catalog item kinds `dish`, `good`, `semi_finished`, `service`.
- `menu` применяет menu items, menu-visible `item_type`, modifier groups/options and menu item modifier group links.
- Unknown JSON fields и unsupported stream names отклоняются до partial apply.

Остается только основа:

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
- stock movement app services for cashier runtime.

Запланировано до пилота как boundary decision, если inventory входит в pilot acceptance:

- Recipe должна быть versioned сущностью.
- Dish, modifier option и semi-finished/preparation могут ссылаться на recipe/preparation semantics.
- Recommended pilot policy: consumption after final check creation, если `KDS/DishServed` не введен как обязательный runtime trigger.
- Для semi-finished/preparation сначала списывается баланс заготовки; разворачивание в компоненты при нехватке разрешается только если эта policy отдельно утверждена.
- Order/precheck/check snapshots должны хранить достаточно данных, чтобы inventory/fiscal/reporting logic не зависела от текущего menu state.
- Inventory changes должны происходить только через immutable stock documents / stock moves, а не прямым mutation counters.

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

Запланировано далее:

- Payment processor boundary и fiscal adapter boundary должны быть отдельными архитектурными зонами.
- Payment processor отвечает за authorization/capture/refund integration с провайдером.
- Fiscal adapter отвечает за legal/fiscal receipt mapping и устройство/сервис фискализации.
- Нельзя смешивать PSP state и fiscal/legal receipt state в одной модели.

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
