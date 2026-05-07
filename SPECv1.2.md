# Итоговая Спецификация и Архитектура RMS/POS Платформы v1.2

## Краткое резюме

Платформа представляет собой распределенную ресторанную учетную систему RMS/POS с автономным Edge-контуром в ресторане и облачным учетным ядром.

**Ключевая парадигма:**

```text
Edge-first POS/KDS
Immutable Ledger
Payment Integrity
Primary Edge Node
Cloud Accounting Core
OLAP Analytics
```

Кассы, кухня, очереди и экраны должны работать без интернета. Cloud отвечает за учет, синхронизацию, справочники, платежное подтверждение, инвентарь, аналитику и юридически значимое хранение данных.

---

# 1. Architecture Decision Records

## ADR-001: Instance-per-Tenant

**Решение:**  
Каждый клиент получает изолированный Cloud-инстанс: Go Backend + отдельная PostgreSQL БД.

**Следствие:**  
`tenant_id` в таблицах не используется.

Иерархия:

```text
organization → restaurant → warehouse / device / shift
```

---

## ADR-002: Master / Branch Hub-and-Spoke

**Решение:**  
Для сетей используется Master Node и Branch Nodes.

```text
Master Node:
  catalog
  recipes
  tax rules
  settings

Branch Node:
  checks
  payments
  shifts
  dish events
  inventory operations
```

**Следствие:**  
Глобальная идентификация сущностей идет строго через UUID.

---

## ADR-003: Edge-first POS — Go Business Core + Web UI + Native Supervisor

**Решение:**  
Go Edge Server является хранителем бизнес-логики, локального состояния и процессов.

```text
Go Edge Server:
  orders
  checks
  payments
  shifts
  tax calculation
  KDS events
  outbox
  sync
  SQLite
```

Kotlin Android-приложение не содержит бизнес-логики. Оно является Android-слоем:

```text
Kotlin Android Layer:
  foreground service
  WebView kiosk
  Go process supervisor
  printer bridge
  payment terminal SDK bridge
  USB/Bluetooth/LAN access
  power management
  boot recovery
```

Для остальных платформ приоритет:

```text
Web first
Browser / WebView / WebView2
Native layer only for hardware access
```

---

## ADR-004: Deduct on Prep

**Решение:**  
Складское списание ингредиентов происходит по факту приготовления блюда — событию `DishServed`.

```text
DishServed → Cloud → recipe_consumption stock document
```

Cloud принимает событие `DishServed`, находит активную версию техкарты и формирует документ списания ингредиентов.

**Важно:**  
Если блюдо отменено после приготовления, автоматического возврата на склад нет. Менеджер оформляет отдельный ручной акт списания/коррекции.

---

## ADR-005: OLTP vs OLAP

**Решение:**

```text
PostgreSQL = учетные документы, сырые события, юридически значимые данные
ClickHouse = тяжелая аналитика, AVCO, отчеты
```

---

## ADR-006: Expiry Tracking без FEFO

**Решение:**  
В MVP ведутся сроки годности:

```text
production_date
expiry_date
```

Они используются для:

```text
KDS-ротации
операционного контроля
автоматических событий просрочки
```

Финансовое списание идет не через FEFO, а через учетные документы.

---

## ADR-007: Payment as First-Class Immutable Entity

**Решение:**  
Оплата является отдельной неизменяемой сущностью, независимой от чека.

```text
check = агрегатор оплат
payment = самостоятельный финансовый факт
```

Оплату нельзя удалять или редактировать. Можно только создавать новые события:

```text
refund
reversal
adjustment
dispute
reconciliation result
```

---

## ADR-008: Primary Edge Node

**Решение:**  
В ресторане один активный Primary Edge Server.

```text
Primary Edge:
  SQLite authoritative DB
  orders
  payments
  shifts
  KDS state
  printer routing
  payment terminal sessions
  outbox
```

Остальные устройства — клиенты:

```text
POS terminals
KDS screens
queue screens
customer displays
manager tablets
```

---

## ADR-009: No Multi-Master in MVP

**Решение:**  
Полный multi-master между кассами запрещен в MVP.

Причины:

```text
double payments
duplicate printing
split-brain
conflicting order state
payment terminal conflicts
clock drift
complex conflict resolution
```

Допускается:

```text
Primary Edge
Optional Standby Edge
Manual / semi-auto failover
```

---

# 2. Technology Stack

## Edge

```text
Backend: Go
Local DB: SQLite
UI: React/Vue SPA
Protocol: HTTP + WebSocket
Sync: Transactional Outbox
```

## Android

```text
Kotlin
Foreground Service
WebView kiosk
Go binary supervisor
Hardware bridge
Power management
```

## Windows

```text
WebView2 / browser shell
Optional .NET hardware bridge
```

## Cloud

```text
Go Modular Monolith
PostgreSQL
Docker
Kubernetes
Helm
```

## Analytics

```text
ClickHouse
CDC / ETL from PostgreSQL
```

---

# 3. Core Domain Boundaries

```text
identity
catalog
recipes
taxes
orders_and_sales
shifts
payments
reconciliation
inventory
sync
devices
analytics
```

Правило:

```text
orders не пишет напрямую в inventory
payments не меняет чек напрямую без доменного use case
sync принимает события и вызывает соответствующие application services
inventory меняется только через stock_documents / stock_moves
```

---

# 4. Edge Runtime Architecture

```text
Restaurant LAN

Primary Edge Host
  ├─ Go Edge Server
  ├─ SQLite
  ├─ WebSocket Hub
  ├─ Sync Worker
  ├─ Payment Engine
  ├─ Tax Engine
  ├─ Printer Router
  ├─ Device Lock Manager
  └─ Static Web App Hosting

Clients:
  POS Web App
  KDS Web App
  Queue Screen
  Customer Display
  Manager Panel
```

Все UI работают через локальный Edge:

```text
http://edge.local
ws://edge.local/ws
```

**Запрещено:** UI напрямую обращается в Cloud.

---

# 5. Модель данных Edge

```text
edge_nodes
edge_devices
device_locks

local_event_log
pos_sync_outbox

shifts_local
cash_sessions_local
cash_drawer_events_local

orders_local
checks_local
check_lines_local

payments_local
payment_attempts_local
payment_allocations_local

kds_tasks_local
dish_served_events_local

print_jobs_local
```

---

# 6. Synchronization Protocol

## Edge → Cloud

Используется Transactional Outbox.

Любая бизнес-операция в одной SQLite-транзакции:

```text
1. пишет факт в локальную таблицу
2. пишет событие в local_event_log
3. пишет команду в pos_sync_outbox
```

Пример:

```text
close check
capture payment
dish served
open shift
close shift
cash drawer event
```

Cloud проверяет idempotency key:

```text
organization_id + restaurant_id + device_id + edge_event_id
```

---

## Cloud → Edge

Edge периодически запрашивает изменения:

```text
GET /api/v1/sync/updates?device_id=...&after_version=...
```

Cloud возвращает:

```text
menu snapshots
tax rules
recipe versions
device config
payment provider config
feature flags
```

---

# 7. Shifts & Cash Sessions

## Обязательное правило

Кассовая смена является обязательным атрибутом всех кассовых и платежных операций.

Без активной смены нельзя:

```text
создать чек
принять оплату
открыть денежный ящик
оформить возврат
закрыть день
```

---

## Модель данных

```text
shifts (
  id UUID,
  restaurant_id UUID,
  device_id UUID,
  cashier_id UUID,
  status: opened | closed | forced_closed,
  opened_at,
  closed_at,
  opening_cash_amount,
  closing_cash_amount,
  expected_cash_amount,
  actual_cash_amount
)
```

```text
cash_sessions (
  id UUID,
  shift_id UUID,
  device_id UUID,
  cashier_id UUID,
  status,
  opened_at,
  closed_at
)
```

```text
cash_drawer_events (
  id UUID,
  shift_id UUID,
  device_id UUID,
  cashier_id UUID,
  type: open | cash_in | cash_out | payout | drop | correction,
  amount,
  reason,
  created_at
)
```

---

# 8. Orders, Checks & Taxes

## Order Lifecycle

```text
created
confirmed
sent_to_kitchen
partially_served
served
cancelled
closed
```

## Check Lifecycle

```text
open
partially_paid
paid
refunded
voided
```

## Tax Engine

Налоговая логика считается только на Go Edge Server.

Для Индонезии:

```text
base price
+ service charge
= subtotal_1
+ PB1
= total
```

```text
tax_profiles
tax_rules
tax_calculation_snapshots
```

В чек сохраняется snapshot расчета налогов, чтобы будущие изменения правил не меняли исторические чеки.

---

# 9. Payments & Financial Integrity Layer

## Payment Lifecycle

```text
initiated
authorized
captured
settled
failed
cancelled
refunded
disputed
```

## Payment Methods

```text
cash
card_terminal
qr_static
qr_dynamic
e_wallet
bank_transfer
delivery_aggregator
voucher
loyalty_points
mixed
```

---

## Модель данных Cloud

```text
payments (
  id UUID,
  restaurant_id UUID,
  device_id UUID,
  shift_id UUID,
  check_id UUID,

  edge_payment_id UUID,
  idempotency_key TEXT,

  method,
  amount,
  currency,
  status,

  provider_name,
  provider_merchant_id,
  provider_terminal_id,
  provider_transaction_id,
  provider_reference,
  provider_rrn,
  provider_auth_code,

  fingerprint_hash,
  evidence_hash,

  paid_at,
  edge_created_at,
  cloud_received_at,
  created_at
)
```

---

## Payment Uniqueness

Основной ключ:

```text
organization_id + restaurant_id + device_id + edge_payment_id
```

Для внешних провайдеров:

```text
provider_name + provider_merchant_id + provider_transaction_id
```

Fallback fingerprint:

```text
sha256(
  restaurant_id +
  device_id +
  shift_id +
  check_id +
  amount +
  currency +
  method +
  local_sequence +
  created_at
)
```

---

## Payment Attempts

```text
payment_attempts (
  id UUID,
  payment_id UUID,
  attempt_no,
  request_hash,
  response_hash,
  provider_request_id,
  status,
  error_code,
  created_at
)
```

---

## Provider Events

```text
payment_provider_events (
  id UUID,
  provider_name,
  provider_event_id,
  event_type,
  raw_payload JSONB,
  payload_hash,
  signature_valid BOOLEAN,
  received_at
)
```

**Правило:** raw payload не теряется.

---

## Payment Allocations

```text
payment_allocations (
  payment_id UUID,
  check_id UUID,
  amount
)
```

Поддерживает:

```text
split payment
partial payment
one payment for multiple checks
tips
service charge split
refund allocation
```

---

## Refunds

```text
refunds (
  id UUID,
  payment_id UUID,
  shift_id UUID,
  amount,
  reason,
  provider_ref,
  status,
  created_at
)
```

---

# 10. Reconciliation

## Sources

```text
PSP settlement reports
bank statements
terminal batch reports
manual CSV/XLSX upload
webhook history
cash shift closing
```

## Statuses

```text
unmatched
matched
amount_mismatch
duplicate
missing_in_pos
missing_in_bank
manual_override
```

## Модель данных

```text
payment_reconciliations (
  id UUID,
  payment_id UUID,
  source_type,
  source_reference,
  matched_amount,
  status,
  matched_at,
  notes
)
```

---

# 11. Хранение юридически значимых доказательств

```text
payment_evidence_archive (
  id UUID,
  payment_id UUID,
  type,
  storage_uri,
  sha256_hash,
  created_at
)
```

Обязательные элементы доказуемости:

```text
raw provider payload
provider IDs
signature validation result
device_id
shift_id
cashier_id
edge timestamp
cloud received timestamp
idempotency key
payload hash
reconciliation status
```

---

# 12. Catalog & Recipes

```text
items (
  id UUID,
  type: ingredient | dish | good | semi_finished,
  base_unit_id
)
```

```text
recipe_versions (
  id UUID,
  item_id UUID,
  effective_from,
  status
)
```

```text
recipe_lines (
  recipe_version_id UUID,
  ingredient_id UUID,
  quantity,
  loss_percent
)
```

Branch-узлы получают read-only snapshots от Master.

---

# 13. KDS & Dish Lifecycle

## Flow

```text
POS creates order
Edge creates preparation tasks
KDS receives task via WebSocket
Cook presses Done
Edge records DishServed
Edge puts event into outbox
Cloud receives DishServed
Cloud creates recipe consumption document
```

## DishServed Event

```text
dish_served_events (
  id UUID,
  restaurant_id UUID,
  shift_id UUID,
  order_id UUID,
  check_id UUID,
  item_id UUID,
  recipe_version_id UUID,
  served_at
)
```

---

# 14. Inventory Ledger

## Immutable Ledger

Остатки меняются только через:

```text
stock_documents
stock_moves
```

Запрещено:

```text
UPDATE quantity = quantity - X
```

---

## Stock Documents

```text
stock_documents (
  id UUID,
  type,
  status,
  source_event_id,
  posted_at
)
```

Типы:

```text
receipt
recipe_consumption
transfer
adjustment
reversal
expiry_write_off
manual_write_off
```

---

## Stock Moves

```text
stock_moves (
  id UUID,
  document_id UUID,
  warehouse_id UUID,
  item_id UUID,
  quantity_delta,
  unit_cost,
  expiry_date
)
```

---

## DishServed → Recipe Consumption

```text
DishServed
  → recipe_version lookup
  → stock_document(type=recipe_consumption)
  → negative stock_moves for ingredients
```

---

## Production Issue / Expiry Event

Событие `production_issue` используется для истечения срока годности/производственного списания.

```text
production_issue event
  → Cloud
  → stock_document(type=expiry_write_off)
  → negative stock_moves
```

Это отдельный поток и не смешивается с `DishServed`.

---

## Reversal

Ошибочный документ не удаляется.

```text
original stock_document
reversal stock_document
corrected stock_document
```

---

## Costing

В MVP:

```text
Last Purchase Price
```

Для аналитики:

```text
AVCO в ClickHouse
```

---

# 15. Device Coordination

## Device Registry

```text
edge_devices (
  id UUID,
  type: pos | kds | printer | payment_terminal | customer_display,
  role,
  status,
  last_seen_at
)
```

---

## Lease Locks

```text
device_locks (
  device_id UUID,
  owner_node_id UUID,
  lock_type,
  lease_until
)
```

---

## Print Jobs

```text
print_jobs (
  id UUID,
  printer_id UUID,
  check_id UUID,
  job_type,
  payload_hash,
  status,
  attempts,
  created_at
)
```

Правило:

```text
printer_id + job_id + idempotency_key
```

---

## Payment Terminal Sessions

```text
payment_terminal_sessions (
  terminal_id UUID,
  payment_id UUID,
  owner_node_id UUID,
  status,
  locked_until
)
```

Терминал нельзя перехватывать во время активной оплаты без timeout/cancel.

---

# 16. Go Module Structure

```text
/cmd
  /api-server
  /edge-server
  /sync-worker
  /cdc-exporter

/internal
  /platform
    /postgres
    /sqlite
    /outbox
    /observability
    /idgen
    /config

  /modules
    /identity
    /catalog
    /recipes
    /taxes
    /orders
    /checks
    /shifts
    /payments
    /reconciliation
    /inventory
    /sync
    /devices
    /analytics
```

---

# 17. Roadmap Development Sequence

## Этап 0: Architecture Lock & Domain Contracts

Цель: зафиксировать контракты до активной разработки UI.

Результаты:

```text
final ADRs
OpenAPI contracts
event schemas
idempotency rules
SQLite schema v1
PostgreSQL schema v1
migration strategy
```

Обязательные контракты:

```text
ShiftOpened
ShiftClosed
OrderCreated
CheckCreated
PaymentCaptured
DishServed
ProductionIssue
PrintJobCreated
SyncEnvelope
```

---

## Этап 1: Edge Core Skeleton

```text
Go Edge Server
SQLite
HTTP API
WebSocket Hub
local_event_log
pos_sync_outbox
healthcheck
device registration
basic config
```

Цель: Edge должен переживать рестарт и сохранять события.

---

## Этап 2: Cloud Sync Receiver

```text
Cloud Go API
PostgreSQL
idempotency table
edge event receiver
raw event storage
device state
sync acknowledgements
```

Цель: доказать надежный сценарий:

```text
8 часов offline
массовый слив outbox
без дублей
без потерь
```

---

## Этап 3: Shifts & Payments Core

```text
open shift
close shift
cash session
cash drawer events
payments_local
payment_attempts
payment_allocations
mock PSP adapter
cash payment
sync payments to Cloud
```

Цель: финансовый фундамент до полноценного POS.

---

## Этап 4: Checks, Orders & Tax Engine

```text
create order
add items
calculate taxes
create check
partial payment
mixed payment
close check
refund skeleton
```

Цель: первый end-to-end MVP:

```text
shift → order → check → payment → sync → cloud
```

---

## Этап 5: Catalog, Recipes & Cloud → Edge Updates

```text
items
menu snapshots
tax profiles
service charge + PB1
recipe_versions
Cloud → Edge pull updates
```

---

## Этап 6: KDS & DishServed

```text
preparation tasks
Web-KDS
WebSocket updates
DishServed event
sync to Cloud
recipe consumption document
```

---

## Этап 7: Inventory Ledger

```text
receipts
stock_documents
stock_moves
recipe_consumption
expiry_write_off from production_issue
manual_write_off
reversal
last_purchase_price cache
```

---

## Этап 8: Android Hardware Layer

```text
Kotlin foreground service
Go process supervisor
WebView kiosk
printer bridge
payment terminal bridge
power management
boot recovery
logs export
```

---

## Этап 9: Reconciliation & Legal Evidence

```text
PSP reports
bank statements
terminal batch close
cash reconciliation
evidence archive
manual resolution
```

---

## Этап 10: Reporting & ClickHouse

```text
CDC
ClickHouse schema
AVCO jobs
stock on hand
food cost
sales dashboards
shift reports
```

---

## Этап 11: Pilot & Stabilization

```text
real printers
real payment providers
offline stress test
power loss test
network loss test
Android sleep test
mass outbox replay
pilot in Indonesia
```

---

# 18. Anti-Patterns

Запрещено:

```text
бизнес-логика в React
бизнес-логика в Kotlin
прямой UI → Cloud
Redis как учетное хранилище
UPDATE остатков
удаление проведенных документов
multi-master Edge в MVP
оплата без shift_id
чек без shift_id
потеря raw provider payload
печать без idempotent print job
```

---

# 19. Определение MVP-0

Первый технически значимый MVP:

```text
One restaurant
One Primary Edge
One POS Web UI
Go Edge Server
SQLite
Open/Close Shift
Create Order
Create Check
Cash Payment
Mock Card Payment
Outbox Sync
Cloud Receiver
PostgreSQL Storage
Idempotency
No duplicates after offline mode
```

MVP-0 считается успешным, если:

```text
Edge работает offline
события не теряются
оплаты не теряются
дубли не создаются
смена закрывается
Cloud получает полный журнал
```

---

# 20. Итоговый вывод

Спецификация готова к разработке после фиксации контрактов Этапа 0.

Правильный порядок:

```text
1. Контракты и схемы
2. Edge Core
3. Cloud Sync Receiver
4. Shifts + Payments
5. Orders + Checks + Taxes
6. Catalog + Recipes
7. KDS
8. Inventory
9. Android hardware
10. Reconciliation
11. ClickHouse
12. Pilot
```

Главный принцип разработки:

> Сначала строим надежный локальный источник истины и финансовую целостность. Потом добавляем UI, кухню, склад, реальные банки и аналитику.
