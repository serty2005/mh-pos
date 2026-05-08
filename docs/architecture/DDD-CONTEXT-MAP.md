# Карта DDD-контекстов POS Core

## Назначение

Документ фиксирует владение сущностями и границы bounded contexts для modular monolith. Это архитектурный источник истины вместе с `SPECv1.3.md`.

Цель карты — не создать один большой `POSContext`, а явно разделить ответственность между контекстами POS Core MVP и будущими контурами RMS.

## Статусы

В документации используются русские статусы:

- `реализовано сейчас`
- `запланировано далее`
- `вне текущего объема`

Английские статусы `implemented now`, `planned next`, `out of scope` не используются как человекочитаемый текст, кроме случаев, где они являются существующим машинно-читаемым значением.

## Таблица контекстов

| Контекст | Чем владеет | Реализовано сейчас | Запланировано далее до MVP/пилота | Вне текущего объема / после пилота | Какие события публикует | Какие события потребляет | Запрещенное владение / анти-сцепление |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `Organization` | `restaurants`, `devices`, `edge_node_identity`, `client_devices`, `employees`, `roles`, настройки организации и ресторана | `restaurants`, `devices`, `edge_node_identity`, `client_devices`, `employees`, `roles`, основа pairing и auth | настройки ресторана, `business_date_local` policy через настройки таймзоны и начала дня, моделирование workstation/cash zone при необходимости пилота | enterprise org tree, multi-brand и сложные legal entity модели | `DeviceRegistered`, `AuthSessionStarted`, `AuthSessionRevoked` | Cloud-authored master-data streams `restaurants`, `devices`, `staff` | Не владеет заказами, оплатами, сменными операциями и sync-доставкой |
| `Catalog` | `catalog_items`, идентичность menu-visible позиций, units, SKU/barcode/quick code, recipe reference | `catalog_items`, упрощенные `menu_items`, основа `recipe_versions` / `recipe_lines` | modifiers, POS category, preparation/sales place mapping, нормализация units | allergens, nutrition, media-rich catalog, если пилот явно не требует | master-data change events через Cloud provisioning, runtime сейчас не публикует Edge-события каталога | Cloud-authored streams `catalog`, `menu` | В долгосрочной модели не владеет pricing decisions и не должен становиться владельцем налоговой политики |
| `Pricing` | price lists, channel/location pricing, currency/rounding policy, tax-in-price policy | упрощенное хранение `menu_items.price`, money/currency invariants и ISO 4217 precision | явная модель `PriceList` / `PriceListItem` или документированное MVP-владение ценой, чтобы `Catalog` не стал владельцем pricing | happy hours, dynamic pricing, advanced promotions | в текущем runtime отдельных событий нет | изменения каталога и справочников валют как входные данные | Не владеет идентичностью товара, составом рецепта, заказом или фактом оплаты |
| `Order` | `orders`, `order_lines`, состояние заказа, guest count, table reference, intent выпуска precheck | runtime `Order -> Precheck`, редактирование заказа, lock/close lifecycle | modifiers/order line modifiers, void reasons, split/merge/transfer decisions, если нужно пилоту | сложная модель гостей, courses, если пилот не требует | `OrderCreated`, `OrderLineAdded`, `OrderLineQuantityChanged`, `OrderLineVoided`, `OrderClosed` | read model каталога, столы, actor context, precheck/payment orchestration через application service | Не пишет напрямую в `Inventory`; не владеет payment facts, fiscal facts и stock movements |
| `Payment` | `payments`, `payment_attempts`, payment methods, payment allocation, refunds/tips далее | cash/card/other, trusted manual card, partial payments, immutable captured payment | явная документация payment allocation, решение по refund policy | real PSP integration | `PaymentCaptured` | `Precheck` как locked финансовая основа, cash session validation | Не мутирует `Order` / `Check` напрямую, кроме orchestration через application service; не владеет fiscal/legal receipt mapping |
| `Fiscal / Tax` | tax policy, fiscal/legal receipt mapping, fiscal device integration далее | основа line/check totals, rounding policy и immutable check snapshot для reprint | tax profile entity и уточнение immutable print snapshot policy по precheck/final check | real fiscalization integration, если нет отдельного pilot requirement | `CheckCreated`, `CheckReprinted` через текущий check flow | `PaymentCaptured`, `Precheck` snapshot, tax/pricing policy | Не владеет заказом, оплатой, PSP и бухгалтерской сверкой |
| `Production` | kitchen ticket, KDS station, cooking status, fire/hold, prep queues | только архитектурные и schema/foundation references | event contract из `Order` в `Production`, если KDS входит в пилот | KDS runtime, если roadmap оставляет его после пилота | будущие `KitchenTicketCreated`, `DishServed` | `OrderLineAdded`, `OrderLineVoided`, menu preparation mapping | Не списывает inventory напрямую без политики `Inventory`; не владеет финансовым закрытием заказа |
| `Inventory` | `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, recipe consumption | schema/domain foundation для recipes, purchase receipts, stock docs/moves/balances/item costs | stock movement app services и политика consumption по `DishServed` / `OrderClosed`, если пилот требует | complex AVCO/FIFO/batch costing | будущие `StockDocumentPosted`, `StockMoveRecorded` | recipes, purchase receipt foundation, production/order consumption events | `Order` не пишет inventory напрямую; изменения только через stock documents / stock moves |
| `Procurement` | suppliers, supplier orders, purchase receipts, supplier prices | `purchase_receipts` / `purchase_receipt_lines` как foundation на границе `Inventory` и `Procurement` | решить в документации, остается ли purchase receipt частью `Inventory` foundation или становится отдельным контекстом `Procurement` | полный supplier workflow, autopurchase | будущие `PurchaseReceiptPosted` | catalog items, supplier references после появления | Не владеет stock balances и costing; проводки склада делает `Inventory` |
| `CRM` | customer profile, contacts, preferences, history | нет отдельного runtime; только `guest_count` на заказе | `customer_id` reference, если нужно пилоту | полноценный CRM | будущие customer events | order history projections | Не владеет order lifecycle и payment allocation |
| `Loyalty` | bonuses, coupons, promos, discount decisioning | не реализовано | только discount result contract, если нужно пилоту | полноценный loyalty engine | будущие discount/bonus events | customer/order context | Не владеет авторитетными totals в UI; backend остается источником финансового расчета |
| `Delivery / Channel` | sales channel, delivery status, aggregators, commissions | базовый dine-in/table flow без отдельного channel runtime | dining mode/channel enum, если нужно пилоту | aggregators, delivery logistics | будущие channel/delivery events | orders, table/reservation context | Не владеет payment facts, order totals и kitchen execution |
| `Reservation / Table` | halls, tables, reservations, occupancy | `halls`, `tables` и выбор стола в cashier flow | решить, остаются ли `halls`/`tables` как Floor subcontext внутри `Reservation / Table` для MVP | reservations, waitlist, сложная occupancy model | будущие table occupancy events | Cloud-authored stream `floor` | Не владеет order lifecycle после создания заказа |
| `Staff / Shift` | `shifts`, `cash_sessions`, `cash_drawer_events`, manager overrides, operational audit actor context | personal shifts, cash sessions, cash drawer events, manager override audit | propagation `business_date_local` и attendance semantics | full HR/timekeeping | `ShiftOpened`, `ShiftClosed`, `CashSessionOpened`, `CashSessionClosed`, `CashDrawerEventRecorded` | employee/role master data, auth session actor context | `auth_session` не является employee shift; `cash_session` не является login session |
| `Accounting / Finance` | reconciliation, P&L, revenue/cost/profit reporting, cash book | базовые Cloud projections и shift finance foundation подтверждены Cloud ingestion code | reporting projections over operational events | full accounting ERP | будущие reconciliation/reporting events | `PaymentCaptured`, `CheckCreated`, shift/cash events, inventory costs | Не владеет runtime capture payment и не меняет операционные факты |
| `Event / Integration` | `local_event_log`, `pos_sync_outbox`, `SyncEnvelope`, inbox/outbox, retry policy, Cloud sync receiver, master-data provisioning/import, webhooks далее | local event log, outbox, sync envelope, Edge -> Cloud sender, Cloud receiver, item-level ACK batch flow, Cloud -> Edge master-data provisioning/import for supported streams | projection query endpoints, production auth perimeter for provisioning | marketplace/plugin system | переносит доменные события без изменения их смысла | все доменные события и Cloud-authored master-data packages | Sync не содержит бизнес-логику домена; Cloud-owned master data не мутируется Edge runtime app services |

## Межконтекстные правила

- `Order` не пишет напрямую в `Inventory`.
- `Payment` не мутирует `Order` / `Check` напрямую, кроме orchestration через application service.
- `Inventory` меняется только через stock documents / stock moves.
- `Catalog` в долгосрочной модели не владеет pricing decisions.
- Frontend не рассчитывает authoritative financial totals.
- Sync transports events and master-data changes; sync не содержит domain business logic.
- Cloud-owned master data не может мутироваться Edge runtime app services.
- Edge-owned operational data синхронизируется в Cloud через append/event flow.
- `auth_session`, employee shift и `cash_session` являются разными runtime-сущностями.
- `Payment` ссылается на `precheck_id`; возвращаться к legacy модели `Check -> Payment` нельзя.

## Связь с текущим кодом

- `pos-backend/internal/pos/domain` уже разделен на пакеты `restaurant`, `device`, `employee`, `floor`, `catalog`, `menu`, `shift`, `order`, `precheck`, `check`, `cash`, `shared`, `inventory`.
- `pos-backend/internal/pos/domain/aliases.go` является transitional package API/facade над domain packages. Он не является bounded context и не должен превращаться в новый god-domain.
- `pos-backend/internal/pos/app/service.go` агрегирует use case services и выполняет orchestration между контекстами на application layer.
- Каноническая SQLite-схема до первого пилота находится в `pos-backend/migrations/sqlite/001_init.sql`.
