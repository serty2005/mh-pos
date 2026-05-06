# Итоговая Спецификация и Архитектура RMS/POS Платформы v1.3

Статус: актуальная pilot-freeze спецификация для MVP-0 и первого запуска  
Язык проекта, документации, промптов и комментариев задач: русский  
Дата фиксации версии: 2026-05-06

---

## 0. Назначение документа

Этот документ является архитектурным источником истины для разработки RMS/POS платформы.

Его нужно использовать при каждой новой итерации генерации кода вместе с:

```text
AGENTS.md
ROADMAP_MVP.md
README.md
pos-backend/README.md
docs/sync/edge-cloud-contracts-v1.md
```

Документ фиксирует:

- целевую архитектуру MVP-0;
- доменные границы;
- инварианты финансовой безопасности;
- правила Edge-first / Offline-first;
- SyncEnvelope и Outbox модель;
- модель Order → Precheck → Payment → Check;
- правила Manager Override;
- правила Device Identity;
- правила DishServed и Inventory Ledger;
- ограничения текущего этапа;
- запреты и anti-patterns.

Важно: проект еще не был запущен в production. Реальных рабочих БД, которые нужно мигрировать, нет. Все новые вводные v1.3 внедряются как схема и логика для первого запуска, а не как production migration существующих данных.

---

# 1. Executive Summary

Платформа представляет собой распределенную ресторанную учетную систему RMS/POS с автономным Edge-контуром в ресторане и облачным учетным ядром.

Ключевая парадигма:

```text
Edge-first POS/KDS
Immutable Ledger & SyncEnvelope
Primary Edge Node as Local Source of Truth
Cloud Accounting Core
OLAP Analytics
```

Кассы, кухня, очереди, клиентские экраны и базовые операции ресторана должны работать без интернета. Cloud отвечает за учет, синхронизацию, справочники, инвентарь, аналитику, интеграции, reconciliation и долговременное хранение данных.

Главный принцип разработки:

```text
Сначала строим надежный локальный источник истины и финансовую целостность.
Потом добавляем UI, кухню, склад, реальные банки и аналитику.
```

---

# 2. Текущий статус проекта

На момент фиксации v1.3 проект уже продвинулся дальше чистого skeleton.

Реализовано или заложено в репозитории:

```text
pos-backend/
  Go Edge Server
  SQLite storage foundation
  domain/app/ports/infra layering
  local_event_log
  pos_sync_outbox
  SyncEnvelope foundation
  Orders foundation
  Checks/Payments foundation по старой модели
  Shifts foundation
  Cash Sessions foundation
  Cash Drawer Events foundation
  payment_attempts foundation
  Inventory/Recipes schema foundation

cloud-backend/
  Go Cloud Sync Receiver
  PostgreSQL bootstrap
  /health
  POST /api/v1/sync/edge-events
  idempotent receive/dedupe
  raw SyncEnvelope storage
```

Фактическая стадия:

```text
Stage 0/1: Edge Core Skeleton & Sync Foundation — DONE
Stage 2: Cash & Shifts Core — DONE / mostly done
Stage 3: Orders, Prechecks & Taxes — NEXT / in progress
Stage 4: Payments & Final Checks — NEXT
```

В v1.3 старая модель `Order → Check → Payment` заменяется целевой финансовой моделью:

```text
Order → Precheck → Payment → Check
```

---

# 3. Non-Goals текущего этапа

На текущем этапе не делаем:

```text
production DB migration
multi-master Edge
real PSP integration
ClickHouse analytics
complex AVCO costing
multi-restaurant network operations
fiscalization integrations
full backoffice ERP
Kubernetes/Helm production deployment
```

Отдельно: миграция БД не нужна, потому что проект еще не запускался в production и нет рабочих БД с данными клиентов. Все изменения v1.3 проектируются для первого запуска.

---

# 4. Architecture Decision Records

## ADR-001: Instance-per-Tenant

Решение: каждый клиент получает изолированный Cloud-инстанс:

```text
Go Backend + PostgreSQL
```

Иерархия:

```text
organization → restaurant → warehouse / device / shift
```

Следствие: в MVP не усложняем каждую таблицу тяжелой multi-tenant логикой. Изоляция клиента достигается уровнем Cloud-инстанса.

---

## ADR-002: Master / Branch Hub-and-Spoke

Решение: Cloud/Master хранит справочники, каталоги, рецепты, налоговые профили и настройки. Edge/Branch генерирует заказы, пречеки, оплаты, смены, KDS-события и локальный audit log.

Связь:

```text
UUID everywhere
Transactional Outbox
SyncEnvelope
Idempotency Key
```

---

## ADR-003: Edge-first POS & Web UI Shell

Решение: Go Edge Server хранит 100% бизнес-логики и работает локально с SQLite.

Android/Windows/Desktop слой является оболочкой:

```text
WebView / browser shell
hardware bridge
printer bridge
device storage
power/sleep handling
```

Запрещено переносить расчет заказов, налогов, скидок, оплат или финальных чеков во frontend.

---

## ADR-004: UI Workspace Architecture — Vite Framework-Agnostic

Решение: frontend кассы не привязывается к React на архитектурном уровне. Разработка MVP ведется на Vite и монорепозитории:

```text
/ui-core
  design system, controls, grids, no business logic

/ui-protocol
  generated DTO
  Edge/Cloud API clients
  SyncEnvelope validators

/apps/pos-react
  React POS application for MVP
```

Цель: позволить в будущем заменить UI на Vue/Svelte/другой renderer без переписывания бизнес-ядра.

---

## ADR-005: Operational SQLite on Edge

Решение: локальная SQLite является Source of Truth для активных POS-операций.

На уровне Go-драйвера фиксируются pragmas:

```text
journal_mode=WAL
synchronous=NORMAL
foreign_keys=ON
busy_timeout=5000
```

Пояснение:

- WAL нужен для конкурентного чтения/записи;
- synchronous=NORMAL — баланс скорости и устойчивости;
- foreign_keys=ON — обязательная целостность;
- busy_timeout=5000 — защита от краткоживущих локов.

Backup:

```text
Edge periodically creates SQLite snapshot
Edge uploads snapshot to Cloud
Cloud stores recoverable artifact
```

Если устройство сгорает, новый Edge можно поднять из последнего snapshot. Неотправленные данные восстанавливаются ручным переносом локальной БД, если физически доступно старое устройство.

---

## ADR-006: Precheck as Locked Snapshot

Решение: отказываемся от прямого редактирования финального чека.

Целевая модель:

```text
Order    — рабочая сущность официанта и кухни
Precheck — заблокированный финансовый snapshot для гостя
Payment  — immutable финансовый факт
Check    — финальный неизменяемый расчетный документ
```

Check создается только после полной оплаты Precheck.

---

## ADR-007: Mandatory Manager Override via Local RBAC

Решение: опасные операции в offline требуют локального PIN менеджера.

Операции:

```text
cancel_precheck
refund_payment
void_after_dish_served
manual_stock_write_off
forced_shift_close
retry_failed_syncs, если политика ресторана требует manager role
```

Edge Server хэширует введенный PIN, проверяет его по локальной таблице сотрудников и ролей, выполняет действие и пишет immutable audit event.

---

## ADR-008: Payment as First-Class Immutable Entity

Решение: Payment — отдельная неизменяемая сущность. Ее нельзя удалять или редактировать.

Для автономных терминалов MVP-0 оплата фиксируется доверенно:

```text
Payment(method='card', status='captured', is_trusted=true)
```

Это означает: кассир физически провел оплату на банковском терминале, а POS фиксирует факт со слов кассира.

---

## ADR-009: Primary Edge Node Only in MVP

Решение: в одном ресторане для MVP есть один Primary Edge Node.

Все остальные устройства:

```text
POS terminals
KDS screens
customer displays
manager tablets
```

являются клиентами Primary Edge.

Multi-master запрещен в MVP.

Причины:

```text
double payments
duplicate printing
split-brain
conflicting order state
terminal session conflicts
clock drift
complex conflict resolution
```

---

## ADR-010: No Production DB Migration Before First Launch

Решение: на текущем этапе не строим миграционную стратегию для production data.

Причина: production еще не было, реальных клиентских БД нет.

Следствие:

- можно менять SQLite schema и PostgreSQL schema под v1.3;
- можно переименовывать старые `check` use cases в `precheck` use cases;
- можно пересобрать dev DB с нуля;
- обязательно описывать новые схемы как first-launch schema.

Нельзя тратить время MVP на backward-compatible migration старой dev-схемы, если это мешает архитектурной чистоте.

---


## ADR-011: Strict Financial Math & Rounding

Решение:

1. Все денежные суммы в системе (БД, API, события) передаются и хранятся исключительно в **minor units (`INTEGER`)**.
2. Использование `REAL`, `FLOAT` или `DECIMAL/NUMERIC` для денег в persistent schema запрещено.
3. Каждая денежная величина обязана иметь `currency_code`.
4. Цена и скидка фиксируются в момент `OrderLineAdded`. Если после этого цена в меню изменилась, позиция в открытом заказе не пересчитывается.
5. Порядок расчета чека: **Discount Before Tax**. Скидка применяется к сабтоталу позиции, и только на получившуюся сумму (`taxable amount`) рассчитывается налог.
6. Округление происходит математически (`round half up`) строго на уровне каждой позиции (`line-level rounding`), а итог чека — это сумма уже округленных позиций.

---

## ADR-012: SQLite Write Transactions & Durability

Решение:

1. Все конкурентно-чувствительные write use cases стартуют транзакцию строго через `BEGIN IMMEDIATE`, а не стандартный `BEGIN` (`DEFERRED`).
2. Функциональный минимум SQLite для `STRICT` tables — `>= 3.37.0`.
3. Pilot-required baseline для production WAL use — `SQLite >= 3.51.3`.
4. Допускаются backported fixed builds `3.50.7` или `3.44.6` только при явном pin в сборке и CI.
5. Все новые финансовые таблицы (`orders`, `prechecks`, `payments`, `checks`, snapshots, outbox) должны создаваться с `STRICT`, если это не ломает совместимость текущего драйвера/мигратора.
6. Резервное копирование SQLite для отправки в Cloud делается исключительно через `VACUUM INTO 'temp_snapshot.db'` во временный новый файл с последующей checksum/metadata валидацией.
7. Прямое OS-level копирование active `.db`, `.db-wal`, `.db-shm` во время работы кассы не является supported backup path.
8. `PRAGMA synchronous = NORMAL` принимается как осознанный pilot trade-off:
   - система остается консистентной в WAL mode;
   - при power loss может потеряться durability последнего commit.
9. Если pilot требует максимальной durability на каждом commit, это оформляется отдельным deployment override на `synchronous = FULL` с обязательным performance test report.

---

## ADR-013: SQLite Runtime Gate

Решение:

1. При старте Edge backend обязан проверять фактическое окружение SQLite, а не только выполнять `PRAGMA`.
2. Запуск запрещен, если не выполнены одновременно:
   - `sqlite_version()` соответствует pilot baseline;
   - `PRAGMA journal_mode` вернул `wal`;
   - `PRAGMA synchronous` вернул `1 (NORMAL)` или отдельное разрешенное deployment override;
   - `PRAGMA foreign_keys` вернул `1`;
   - `PRAGMA busy_timeout` не меньше `5000`.
3. Проверка выполняется fail-fast на старте процесса.
4. `PRAGMA foreign_key_check` является частью bootstrap smoke test после миграций.

---

## ADR-014: Device Binding Secret Storage

Решение:

1. Binding code никогда не хранится в plaintext.
2. Для verifier-side хранения используется не plain hash, а keyed verifier format:
   - рекомендуемый default: `binding_code_hmac = HMAC-SHA-256(server_secret, code)`;
   - memory-hard hash допустим только если это отдельно обосновано.
3. Android production storage — keystore-backed local storage.
4. Windows production storage — local DPAPI-protected storage.
5. Roaming credential stores для production `device_id` запрещены.

---

# 5. Core Domain Boundaries

Целевые bounded contexts:

```text
identity / employees / local RBAC
catalog & recipes
taxes
orders_and_sales
shifts & cash_sessions
payments
reconciliation
inventory
sync
devices
kds
printing
analytics
```

Правила границ:

```text
orders_and_sales не пишет напрямую в inventory
payments не изменяет order/check напрямую без application service
inventory меняется только через stock documents / stock moves
sync принимает и доставляет события, но не содержит бизнес-логику домена
frontend не считает налоги, скидки, totals, payments allocation
```

---

# 6. Layering и правила кода

В Go backend сохраняется Clean Architecture:

```text
domain → app → ports → infra
```

## domain

Разрешено:

```text
business types
state machines
invariants
errors
pure calculations
```

Запрещено:

```text
HTTP
SQL
database/sql
SQLite/PostgreSQL imports
filesystem
network
```

## app

Разрешено:

```text
use case orchestration
transactions
repository calls
outbox/local_event orchestration
```

Запрещено:

```text
direct SQL
HTTP mapping
UI concerns
```

## ports

Только интерфейсы репозиториев и внешних сервисов.

## infra

Реализация портов:

```text
SQLite repositories
PostgreSQL repositories
filesystem snapshot storage
HTTP clients if needed
```

## api

HTTP handlers должны быть тонкими:

```text
parse request
validate transport fields
call app service
map response/error
```

---

# 7. Edge Runtime Architecture

```text
Restaurant LAN

Primary Edge Host
  ├─ Go Edge Server
  ├─ SQLite WAL DB
  ├─ HTTP API
  ├─ WebSocket Hub
  ├─ Sync Worker
  ├─ Tax Engine
  ├─ Payment Recording Engine
  ├─ Printer Router
  ├─ KDS Event Router
  ├─ Device Registry
  └─ Static Web App Hosting

Clients
  ├─ POS Web UI
  ├─ KDS Web UI
  ├─ Customer Display
  ├─ Manager Tablet
  └─ Android/Windows shell
```

Все UI обращаются только к локальному Edge:

```text
http://edge.local
ws://edge.local/ws
```

Запрещено:

```text
POS UI → Cloud напрямую
KDS UI → Cloud напрямую
Frontend business calculations
```

---


## 7.1 Топология пилота (Pilot Topology)

Для первого запуска (MVP-0) замораживается следующая инфраструктурная модель:

1. **Один хост / all-in-one terminal:** POS UI, Go Edge Backend и SQLite физически находятся на одном устройстве.
2. **Запрет Network FS:** размещение SQLite БД на сетевых дисках (`SMB/NFS/WebDAV`) и в sync-папках (`Google Drive / Dropbox / OneDrive`) строго запрещено.
3. **Путь печати:** для MVP-0 фиксируется один supported path — **Network ESC/POS Printer** через локальную сеть.
4. Сетевой принтер является периферией. Он не является частью локального хранения БД и не меняет правило all-in-one для runtime-ядра.

## 7.2 Print Failure Semantics

Печать не является частью финансовой транзакции.

1. Ошибка печати не откатывает `IssuePrecheck`, `CapturePayment`, `CheckCreated` и `OrderClosed`.
2. После неудачной печати UI обязан показать оператору статус ошибки и действие `Reprint`.
3. Backend обязан повторно сформировать печатный payload из сохраненного snapshot, а не пересчитывать суммы заново.
4. Повторная печать precheck/check обязательна для MVP-0.

# 8. Synchronization Protocol

В системе закреплен корпоративный стандарт обмена — SyncEnvelope.

Любая write-операция выполняется в одной транзакции:

```text
BEGIN IMMEDIATE
  business logic writes
  local_event_log append
  pos_sync_outbox append
COMMIT
```

Если событие попало в бизнес-таблицу, но не попало в `local_event_log` / `pos_sync_outbox` — это bug.

---

## 8.1 SyncEnvelope

Минимальные поля:

```text
version
event_id
command_id
event_type
aggregate_type
aggregate_id
restaurant_id
device_id
shift_id nullable
occurred_at_utc
payload
```

`event_id` является уникальным Edge event identifier. Каждый write API должен принимать стабильный `command_id`, чтобы client retries не создавали повторные бизнес-факты.

---

## 8.2 Idempotency Key для Cloud

Стандарт v1.3 pilot freeze:

```text
restaurant_id : device_id : event_id
```

Cloud обязан:

- принимать повтор того же payload как idempotent replay;
- возвращать стабильный ack;
- не создавать дублей;
- конфликтовать, если idempotency key тот же, но payload отличается.

---

## 8.3 Outbox гарантия

Таблица `pos_sync_outbox` должна поддерживать:

```text
id
event_id
command_id
envelope_version
aggregate_type
aggregate_id
event_type
payload
status                 -- pending | processing | sent | failed | suspended
sequence_no            -- monotonic local ordering key
attempts
next_retry_at
last_error
locked_at nullable
locked_by nullable
created_at_utc
updated_at_utc
sent_at_utc nullable
```

`sequence_no` — канонический ключ порядка доставки. Worker всегда выбирает батчи так:

```sql
ORDER BY sequence_no ASC
```

Для `sequence_no` допускается `INTEGER PRIMARY KEY AUTOINCREMENT`, потому что outbox нужен не переиспользуемый ordering key даже после purge старых `sent` записей.

---

## 8.4 Retry Policy

Для MVP-0 фиксируются значения:

```text
base_delay_ms = 1000
max_delay_ms  = 300000
lease_ttl_seconds = 120
max_attempts_before_suspended = 20
```

Формула:

```text
delay_ms = min(base_delay_ms * 2^attempts, max_delay_ms) + random(0, 1000)
```

При `attempts > 20` статус меняется на `suspended`.

Worker выбирает только:

```text
status = 'pending'
AND (next_retry_at IS NULL OR next_retry_at <= now_utc)
```

---

## 8.5 Item-level ACK от Cloud

Cloud не возвращает `all-or-nothing`. Для каждого события в батче Cloud обязан вернуть индивидуальный статус:

- `accepted`: Edge ставит `status = 'sent'`.
- `duplicate`: Edge ставит `status = 'sent'`.
- `retryable_error`: Edge увеличивает `attempts`, рассчитывает `next_retry_at` и возвращает запись в `pending`.
- `terminal_error`: Edge ставит `status = 'failed'`.

Минимальный формат batch ACK:

```json
{
  "batch_id": "uuid",
  "results": [
    {
      "event_id": "uuid",
      "status": "accepted | duplicate | retryable_error | terminal_error",
      "error_code": "optional_machine_code",
      "message": "optional_human_message"
    }
  ]
}
```

---

## 8.6 Retry Classification

По умолчанию retryable:

- timeout;
- network unavailable;
- DNS/TLS transient failure;
- HTTP `408`, `425`, `429`, `500`, `502`, `503`, `504`.

По умолчанию terminal:

- schema validation error;
- malformed JSON envelope;
- unsupported `event_type` / `event_version`;
- signature/authentication failure на payload level.

---

## 8.7 Lease Recovery для Outbox

Для предотвращения зависания событий со статусом `processing`:

- Worker при взятии батча обновляет `status = 'processing'`, `locked_at = now_utc`, `locked_by = worker_instance_id`.
- При следующем цикле Worker имеет право reclaim-события:

```sql
WHERE status = 'processing'
  AND locked_at < now_utc - interval '120 seconds'
```

- При reclaim запись возвращается в `pending`, а `locked_at` и `locked_by` очищаются.

---

## 8.8 Manual Retry Failed Syncs

В UI менеджера должна быть операция:

```text
Retry Failed Syncs
```

Она делает:

```text
status = 'pending'
attempts = 0
next_retry_at = null
last_error = null or preserved separately
locked_at = null
locked_by = null
```

Операция может требовать Manager Override в зависимости от политики ресторана.

---

# 9. Device Identity

`device_id` участвует в idempotency key, audit trail, сменах, платежах и печати. Он не может генерироваться случайно при каждом запуске.

---

## 9.1 Риск

Если `device_id` берется из нестабильного источника:

```text
MAC address
random UUID on startup
browser fingerprint
OS installation id
```

то Cloud может:

- создать дубли;
- смешать события разных устройств;
- потерять idempotency guarantees.

---

## 9.2 Provisioning Flow

Устройство не придумывает себе ID само.

Первый запуск:

```text
1. POS UI показывает экран привязки.
2. Менеджер вводит OTP / binding code.
3. Primary Edge или Cloud проверяет код.
4. Primary Edge регистрирует устройство.
5. Генерируется стабильный Device UUID.
6. UUID сохраняется в devices table.
7. UUID возвращается устройству.
8. Устройство сохраняет UUID в persistent local storage.
```

Хранилища:

```text
Android: Android Keystore / app protected storage
Windows: config file with filesystem permissions
Browser shell: local config controlled by native shell
Development browser: localStorage допустим только для dev
```

---

## 9.3 Контракт

Полученный `device_id` используется во всех:

```text
SyncEnvelope
orders
prechecks
checks
payments
shifts
cash_sessions
cash_drawer_events
manager_override_logs
print_jobs
kds events
```

Если устройство сгорело, новое устройство проходит привязку заново и получает новый `device_id`.

Нельзя переиспользовать старый `device_id` без процедуры восстановления, явно подтвержденной менеджером/администратором.

---


## 9.4 Безопасность Binding Flow и жизненный цикл

Для защиты процесса привязки устройства вводятся строгие ограничения.

### Binding Code

- Длина: 8 цифровых символов.
- TTL: строго 10 минут.
- Single-use: код сгорает сразу после первого успешного применения.
- Rate limiting: максимум 5 неудачных попыток, после чего код инвалидируется.
- Resend: генерация нового кода немедленно инвалидирует предыдущий.
- Запрет логирования: binding code никогда не пишется в plaintext ни в логи, ни в БД.
- Verifier-side хранение: `binding_code_hmac` или иной keyed verifier format.

### Транспорт

- Процесс привязки и вся дальнейшая синхронизация работают только по TLS 1.2/1.3.
- Production sync/provisioning по открытому HTTP запрещены.

### Жизненный цикл устройства

```text
pending -> active -> revoked -> replaced
```

### Защита от клонирования

- При restore ОС, клонировании диска или reinstall приложения старый `device_id` запрещено переиспользовать молча.
- Приложение обязано потребовать повторную авторизацию (`Rebind`) менеджером.
- Только authoritative registrar может перевести старый `device_id` в `replaced` и выдать новый.

### Хранение identity на клиенте

- Android: keystore-backed local storage.
- Windows: local DPAPI-protected storage.
- Roaming credential stores для production `device_id` запрещены.

---

# 10. Orders, Prechecks, Checks & Taxes

Это ядро финансовой безопасности v1.3.

---

## 10.1 Основная модель

```text
Order
  рабочая сущность официанта и кухни
  можно добавлять позиции до locked

Precheck
  locked financial snapshot
  выдается гостю
  фиксирует lines, discounts, taxes, totals

Payment
  immutable financial fact
  привязан к Precheck

Check
  final immutable document
  создается только после полной оплаты Precheck
```

---

## 10.2 Order lifecycle

Статусы:

```text
open
locked
cancelled
closed
```

Правила:

- `open` можно редактировать;
- `locked` нельзя редактировать без отмены активного Precheck через Manager Override;
- `closed` нельзя редактировать;
- `cancelled` нельзя редактировать.

---

## 10.3 Precheck lifecycle

Статусы:

```text
issued
superseded
cancelled
paid
```

Правила:

- активным может быть только один `issued` Precheck на Order;
- `paid` означает, что финальный Check уже создан или создается в той же транзакции;
- `cancelled` требует Manager Override;
- `superseded` используется при выпуске новой версии snapshot.

---

## 10.4 Check lifecycle

Check создается автоматически, когда сумма captured payments по Precheck достигает total.

Правила:

```text
No orphan checks
No check before full payment
No direct check editing
No silent mutation after check generation
```

---

## 10.5 Precheck Versioning

Риск: два официанта одновременно нажимают «Распечатать пречек» на разных устройствах. Если версии задублируются, ломаются отмены, оплаты и audit trail.

Решение: в MVP есть один Primary Edge Node и SQLite, поэтому race condition решается нативной транзакцией БД.

Правила для `IssuePrecheck`:

```text
BEGIN
  SELECT MAX(version_no) FROM prechecks WHERE order_id = ?
  version = COALESCE(max, 0) + 1
  UPDATE prechecks SET status = 'superseded'
    WHERE order_id = ? AND status = 'issued'
  INSERT INTO prechecks(... version_no = version, status = 'issued' ...)
  UPDATE orders SET status = 'locked' WHERE id = ?
  INSERT local_event_log
  INSERT pos_sync_outbox
COMMIT
```

Обязательный SQLite constraint:

```sql
CREATE UNIQUE INDEX idx_prechecks_order_version
ON prechecks(order_id, version_no);
```

Дополнительно желательно обеспечить один активный issued Precheck на order:

```sql
CREATE UNIQUE INDEX idx_prechecks_one_issued_per_order
ON prechecks(order_id)
WHERE status = 'issued';
```

Если второй конкурентный запрос не сможет вставить версию из-за constraint, application service должен вернуть доменную ошибку или повторить IssuePrecheck внутри новой транзакции по явной политике. Для MVP предпочтительно вернуть понятную ошибку и попросить UI обновить состояние заказа.

---

## 10.6 Split после Precheck

Если Precheck уже issued:

```text
1. Manager вводит PIN.
2. Active Precheck переводится в cancelled.
3. Order разблокируется.
4. Order делится на два новых Order или две logical groups по политике MVP.
5. Выпускаются новые Prechecks.
```

Запрещено тихо менять issued Precheck.

---

## 10.7 Ошибка после полной оплаты

Reopen «как есть» запрещен.

Правильный flow:

```text
1. Manager Override.
2. Refund original payment или partial refund.
3. Original Check остается immutable.
4. Order копируется в новый Order.
5. Выпускается новый Precheck.
6. Принимается новая Payment.
7. Генерируется новый Check.
```

---


## 10.8 Refund Flow, блокировка отмены и частичные оплаты

Refund включен в MVP-0 как минимальный compensating flow.

1. Если precheck имеет `paid_total_minor > 0`, его отмена через Manager Override строго запрещена.
2. Менеджер инициирует `RefundPayment` через Manager PIN.
3. Создается новая immutable ledger row в `payments`.
4. Refund использует отрицательную сумму: `amount_minor < 0`.
5. Refund обязан иметь `original_payment_id`, указывающий на исходный capture.
6. Refund обязан иметь `entry_kind = 'refund'`.
7. Исходный платеж capture обязан иметь `entry_kind = 'capture'`.
8. `paid_total_minor = sum(capture.amount_minor) + sum(refund.amount_minor)`.
9. Только когда `paid_total_minor = 0`, менеджер может выполнить `CancelPrecheck` и разблокировать заказ для редактирования.

Partial payment в MVP-0 не вводит отдельный статус `partially_paid`: precheck остается `issued`, а факт частичной оплаты определяется как:

```text
status = 'issued' AND paid_total_minor > 0
```

---

## 10.9 Матрица состояний и блокировки

| Entity | From | Command | To | Actor | Инварианты |
|---|---|---|---|---|---|
| Order | `open` | `IssuePrecheck` | `locked` | cashier | active shift, snapshot created, outbox written |
| Order | `locked` | `CancelPrecheck` | `open` | manager | only when `paid_total_minor = 0` |
| Order | `locked` | `CreateFinalCheck` | `closed` | system | only after full payment |
| Precheck | none | `IssuePrecheck` | `issued` | cashier | one issued precheck per order |
| Precheck | `issued` | `SupersedePrecheck` | `superseded` | system | forbidden if `paid_total_minor > 0` |
| Precheck | `issued` | `CancelPrecheck` | `cancelled` | manager | forbidden if `paid_total_minor > 0` |
| Precheck | `issued` | `CapturePayment` partial | `issued` | cashier | paid amount tracked by ledger sum |
| Precheck | `issued` | `CapturePayment` full | `paid` | system | final check created in same transaction |
| Payment | none | `CapturePayment` | `captured` | cashier | immutable positive ledger row |
| Payment | `captured` | `RefundPayment` | unchanged | manager | new negative ledger row, original row unchanged |
| Check | none | `CreateFinalCheck` | `created` | system | only after full payment; immutable |

Запрещенные переходы:

- `Check` до полной оплаты.
- `Payment` без active shift.
- `Payment` для `cancelled` или `superseded` precheck.
- `CancelPrecheck` при `paid_total_minor > 0`.
- Изменение `Order` после active precheck без cancel/refund flow.

---

## 10.10 Payment Data Shape for MVP-0

Минимальные обязательные поля платежа:

```text
payment_id
precheck_id
original_payment_id NULL
entry_kind          -- capture | refund
method              -- cash | trusted_card
status              -- captured | refunded
amount_minor        -- signed INTEGER
currency_code
provider_reference NULL
terminal_id NULL
auth_code NULL
operator_note NULL
captured_at_utc
business_date_local
device_id
shift_id
cash_session_id
operator_user_id
```

Для cash, если пилот включает наличные с выдачей сдачи, дополнительно фиксируются:

```text
tendered_amount_minor
change_amount_minor
```

---

## 10.11 Generic Tax Engine

Налоги не хардкодятся.

Настраиваются `tax_profiles`, применяемые к `catalog_items`.

Поля профиля:

```text
id
name
rate_percent
is_inclusive
receipt_label
active
created_at
updated_at
```

Пример для Индонезии:

```text
legal name: PBJT
receipt_label: PB1
rate_percent: restaurant-specific / jurisdiction-specific
is_inclusive: true or false by settings
```

Важно: `PB1` нельзя хардкодить в Go-логике. Это только label из tax profile.

Precheck должен сохранять tax snapshot, чтобы будущие изменения налогов не меняли исторические документы.

---

# 11. Shifts & Cash Sessions

Кассовая смена обязательна.

Без активной смены нельзя:

```text
create order
issue precheck
capture payment
create final check
record cash drawer event
perform refund
close day
```

---

## 11.1 Shift

Shift — операционная смена ресторана/устройства.

Инварианты:

```text
на device может быть только одна active shift
нельзя создать order без active shift
нельзя закрыть shift с open/locked orders
нельзя закрыть shift с active cash session
```

---

## 11.2 Cash Session

Cash Session — финансовая сессия для наличных.

Инварианты:

```text
нельзя открыть cash session без active shift
нельзя иметь две active cash sessions на device/cash drawer
cash drawer events требуют active cash session
```

---

## 11.3 Cash Drawer Events

Типы:

```text
cash_in
cash_out
drop
correction
open_drawer
```

Все события пишутся append-only и синхронизируются через outbox.

---


## 11.4 Business Date и время

Кассовый день не совпадает с календарным.

- Для всех бизнес-сущностей (`shifts`, `cash_sessions`, `prechecks`, `payments`, `checks`) вводится обязательное поле `business_date_local`.
- Формат: строка `YYYY-MM-DD` в таймзоне ресторана на момент открытия смены.
- Все системные timestamps (`created_at`, `occurred_at`, `locked_at`, `sent_at`) в БД и событиях хранятся в UTC формате RFC3339Nano.
- Опора на локальное не-нормализованное время сервера запрещена.
- Cloud всегда пишет отдельный `received_at_utc`.

---

# 12. Payments & Financial Integrity Layer

---

## 12.1 MVP-0: Trusted Terminal Payments

В MVP-0 нет интеграции с банковским API.

Flow:

```text
1. Кассир вводит сумму на физическом банковском терминале.
2. Терминал печатает банковский slip.
3. Кассир нажимает в POS «Оплачено картой».
4. POS создает Payment(method='card', status='captured', is_trusted=true).
```

Для cash:

```text
Payment(method='cash', status='captured', is_trusted=true)
```

---

## 12.2 Payment invariants

```text
Payment нельзя удалить
Payment нельзя редактировать
исправление = refund/reversal/adjustment
Payment должен быть связан с Precheck
Payment должен быть связан с Shift
нельзя переплатить Precheck без явной политики tips/overpayment
```

Для MVP tips/overpayment лучше запретить до отдельной политики.

---

## 12.3 Final Check Generation

В одной транзакции после payment capture:

```text
BEGIN
  INSERT payment
  SELECT SUM(captured payments) FOR precheck
  IF sum == precheck.total:
    INSERT check
    UPDATE precheck SET status = 'paid'
    UPDATE order SET status = 'closed'
  INSERT local_event_log
  INSERT pos_sync_outbox
COMMIT
```

Если сумма меньше total:

```text
Precheck остается issued
Order остается locked
```

Если сумма больше total:

```text
reject unless explicit tips/overpayment policy exists
```

---

## 12.4 Refund

Refund требует Manager Override.

Refund не удаляет исходный Payment.

Создается отдельный финансовый факт:

```text
Payment(type='refund', amount negative or refund table record)
```

Конкретная schema может быть выбрана при реализации, но immutable principle обязателен.

---

## 12.5 Post-MVP PSP Integrations

После MVP:

```text
Adyen
Stripe
local Indonesian PSPs
QRIS/e-wallets
```

Тогда добавляются:

```text
provider_transaction_id
provider_reference
provider_rrn
provider_auth_code
raw webhook payload archive
signature validation
PCI-safe payload cleaning
```

Raw payload нельзя хранить с PAN/SAD/PII без очистки. Payment Evidence Archive проектируется отдельно.

---


## 12.6 Security & Manager PIN Handling

1. Менеджерский PIN хэшируется алгоритмом **Argon2id** или documented fallback (`scrypt` / `PBKDF2`).
2. Plaintext PIN хранить запрещено.
3. PIN, OTP, access token, refresh token и любые аутентификационные секреты никогда не должны попадать в `local_event_log`, `pos_sync_outbox`, `manager_override_logs` или application logs.
4. Override действует на **одно действие**. Одно действие = один ввод PIN.
5. В `manager_override_logs` обязательно пишутся `target_id`, `target_type`, `reason_code`, `manager_user_id`, `cashier_user_id`, `device_id`, `created_at_utc`, `outcome`.

---

## 12.7 Payment Data Compliance

1. `CVV/CVC`, track data, PIN block и полный `PAN` не хранятся в БД и не передаются в Cloud ни при каких условиях.
2. `provider_reference` используется только для безопасных reference-значений эквайера / терминала.
3. Если в будущем потребуется хранить отображаемый card fragment, для этого используется отдельное display-only поле `masked_pan`, а не `provider_reference`.
4. Raw PSP payload storage проектируется отдельно и только после PCI-safe filtering.

---

# 13. Reconciliation

MVP-0 trusted terminal payments закрываются через сверку банковской выписки.

Источники:

```text
CSV bank statement
terminal batch report
manual upload
future PSP settlement API
```

Статусы:

```text
matched
amount_mismatch
missing_in_pos
missing_in_bank
duplicate
manual_override
```

Reconciliation выполняется в Cloud, не блокирует локальную кассу.

---

# 14. KDS & DishServed

---

## 14.1 DishServed Timing

Ингредиенты физически считаются потраченными только когда блюдо реально приготовлено и отдано.

Запрещено списывать inventory по:

```text
OrderLineAdded
OrderCreated
SendToKitchen
IssuePrecheck
```

---

## 14.2 Когда генерируется DishServed

Вариант 1 — с KDS:

```text
Cook presses Done on KDS item
→ Edge records DishServed
→ Outbox event
→ Cloud creates recipe_consumption document
```

Вариант 2 — MVP без KDS:

```text
Cashier marks item as Served/Issued in POS UI
```

или, если отдельной отметки нет:

```text
Final Check generated after full payment
→ system auto-generates DishServed for not-yet-served eligible lines
```

Автоматическая генерация при final Check допустима только как MVP fallback и должна быть явно помечена в payload:

```text
served_source = 'auto_on_check'
```

---

## 14.3 Cancel before/after DishServed

Если позиция отменяется до DishServed:

```text
позиция удаляется/void из заказа
inventory не меняется
```

Если позиция отменяется после DishServed:

```text
inventory автоматически не возвращается
manager оформляет акт порчи/списания/коррекции
```

Причина: продукты уже реально потрачены.

---

# 15. Inventory Ledger

Остатки меняются только через immutable documents.

Запрещено:

```sql
UPDATE stock SET quantity = quantity - X;
```

---

## 15.1 Stock Documents

Типы:

```text
receipt
recipe_consumption
transfer
adjustment
write_off
reversal
```

---

## 15.2 Stock Moves

Каждый документ создает движения:

```text
stock_document
  → stock_moves
```

Поля движения:

```text
warehouse_id
item_id
quantity_delta
unit_cost optional
source_event_id
created_at
```

---

## 15.3 DishServed → Recipe Consumption

```text
DishServed
  → Cloud receives event
  → find active recipe version
  → create stock_document(type='recipe_consumption')
  → create negative stock_moves for ingredients
```

---

## 15.4 Corrections

Ошибки учета исправляются новым документом:

```text
reversal
adjustment
write_off
```

Нельзя удалять или редактировать проведенные stock documents.

---

## 15.5 Costing

MVP:

```text
Last Purchase Price
```

Post-MVP analytics:

```text
AVCO in ClickHouse
```

---

# 16. Backup & Recovery

---

## 16.1 SQLite Snapshot

Edge периодически делает snapshot SQLite и отправляет в Cloud.

Snapshot должен включать:

```text
business tables
local_event_log
pos_sync_outbox
schema version
device identity
created_at
hash/checksum
```

---

## 16.2 Restore

Восстановление:

```text
1. Установить новый Edge.
2. Привязать ресторан/устройство через recovery procedure.
3. Скачать последний snapshot.
4. Запустить integrity checks.
5. Продолжить sync outbox replay.
```

Нельзя автоматически переиспользовать старый device_id без явного recovery flow.

---


## 16.3 Безопасный Snapshot

Поскольку база работает в режиме WAL, `.db`, `.db-wal` и `.db-shm` нельзя считать произвольно копируемым live backup contract.

- Edge выполняет online snapshot только через `VACUUM INTO 'temp_snapshot.db'`.
- Целевой файл snapshot не должен существовать заранее.
- После успешного завершения snapshot вычисляется SHA-256 checksum.
- Только после успешной валидации snapshot архивируется и отправляется в Cloud.
- Простое OS-level копирование active DB не считается supported recovery procedure.

---

## 16.4 Metadata Bundle

Snapshot отправляется не “голым” файлом, а metadata bundle:

```text
snapshot.db
metadata.json
sha256
```

`metadata.json` минимум:

```json
{
  "schema_version": "...",
  "app_version": "...",
  "sqlite_version": "...",
  "device_id": "...",
  "restaurant_id": "...",
  "created_at_utc": "..."
}
```

---

## 16.5 Recovery Flow

При восстановлении базы из snapshot на новом железе часть событий в `pos_sync_outbox` может оставаться в `pending`.

Это штатная ситуация:

- Edge отправит их повторно.
- Cloud безопасно проигнорирует дубликаты благодаря `event_id`-based idempotency.
- Повторная отправка после restore считается частью нормального recovery, а не инцидентом.

---

# 17. Edge Data Model v1.3

Ниже целевая first-launch schema-level модель. Конкретные имена таблиц могут сохранять суффиксы `_local`, если это уже принято в коде, но доменная семантика должна соответствовать v1.3.

---

## 17.1 Core operational tables

```text
restaurants
devices
employees
roles
employee_roles
shifts
cash_sessions
cash_drawer_events
orders
order_lines
prechecks
precheck_lines
precheck_tax_lines
payments
checks
check_lines
manager_override_logs
local_event_log
pos_sync_outbox
```

---

## 17.2 Prechecks

```text
prechecks (
  id UUID PRIMARY KEY,
  order_id UUID NOT NULL,
  restaurant_id UUID NOT NULL,
  device_id UUID NOT NULL,
  shift_id UUID NOT NULL,
  version_no INTEGER NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('issued','superseded','cancelled','paid')),
  subtotal NUMERIC NOT NULL,
  discount_total NUMERIC NOT NULL DEFAULT 0,
  tax_total NUMERIC NOT NULL,
  total NUMERIC NOT NULL,
  issued_by_user_id UUID NOT NULL,
  cancelled_by_user_id UUID NULL,
  manager_override_user_id UUID NULL,
  cancel_reason_code TEXT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
)
```

Required indexes:

```sql
CREATE UNIQUE INDEX idx_prechecks_order_version
ON prechecks(order_id, version_no);

CREATE UNIQUE INDEX idx_prechecks_one_issued_per_order
ON prechecks(order_id)
WHERE status = 'issued';
```

---

## 17.3 Precheck lines

```text
precheck_lines (
  id UUID PRIMARY KEY,
  precheck_id UUID NOT NULL,
  order_line_id UUID NOT NULL,
  catalog_item_id UUID NOT NULL,
  name_snapshot TEXT NOT NULL,
  quantity NUMERIC NOT NULL,
  unit_price NUMERIC NOT NULL,
  subtotal NUMERIC NOT NULL,
  tax_total NUMERIC NOT NULL,
  total NUMERIC NOT NULL,
  created_at TIMESTAMP NOT NULL
)
```

---

## 17.4 Precheck tax lines

```text
precheck_tax_lines (
  id UUID PRIMARY KEY,
  precheck_id UUID NOT NULL,
  precheck_line_id UUID NULL,
  tax_profile_id UUID NOT NULL,
  tax_name_snapshot TEXT NOT NULL,
  receipt_label_snapshot TEXT NOT NULL,
  rate_percent NUMERIC NOT NULL,
  is_inclusive BOOLEAN NOT NULL,
  taxable_amount NUMERIC NOT NULL,
  tax_amount NUMERIC NOT NULL,
  created_at TIMESTAMP NOT NULL
)
```

---

## 17.5 Manager override logs

```text
manager_override_logs (
  id UUID PRIMARY KEY,
  restaurant_id UUID NOT NULL,
  device_id UUID NOT NULL,
  shift_id UUID NULL,
  manager_user_id UUID NOT NULL,
  action_type TEXT NOT NULL,
  reason_code TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL
)
```

Action types:

```text
cancel_precheck
refund_payment
void_served_item
forced_shift_close
retry_failed_syncs
manual_stock_write_off
```

---

## 17.6 Tax profiles

```text
tax_profiles (
  id UUID PRIMARY KEY,
  restaurant_id UUID NOT NULL,
  name TEXT NOT NULL,
  rate_percent NUMERIC NOT NULL,
  is_inclusive BOOLEAN NOT NULL,
  receipt_label TEXT NOT NULL,
  active BOOLEAN NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
)
```

---

## 17.7 Outbox additions

`pos_sync_outbox` должна иметь:

```text
attempts INTEGER NOT NULL DEFAULT 0
next_retry_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
last_error TEXT NULL
```

---

# 18. Required Domain Events

MVP-0 / near-MVP events:

```text
ShiftOpened
ShiftClosed
CashSessionOpened
CashSessionClosed
CashDrawerEventRecorded
OrderCreated
OrderLineAdded
OrderLineVoided
PrecheckIssued
PrecheckCancelled
PrecheckSuperseded
PaymentCaptured
PaymentRefunded
CheckCreated
OrderClosed
ManagerOverrideGranted
DishServed
DeviceBound
SyncFailedSuspended
FailedSyncRetryRequested
```

Post-MVP:

```text
RecipeVersionActivated
StockDocumentPosted
BankStatementImported
PaymentReconciliationMatched
PaymentReconciliationMismatch
PSPWebhookReceived
PaymentEvidenceArchived
```

---

# 19. API Surface MVP

Конкретные URL могут быть адаптированы под существующий router, но смысловые use cases обязательны.

---

## 19.1 Health / sync operational

```text
GET /health
GET /api/v1/sync/outbox?limit=50
GET /api/v1/sync/local-events?limit=50&event_type=...
POST /api/v1/sync/retry-failed
```

---

## 19.2 Shifts & cash

```text
POST /api/v1/shifts/open
POST /api/v1/shifts/close
POST /api/v1/cash-sessions/open
POST /api/v1/cash-sessions/close
POST /api/v1/cash-drawer-events
```

---

## 19.3 Orders & prechecks

```text
POST /api/v1/orders
POST /api/v1/orders/{order_id}/lines
POST /api/v1/orders/{order_id}/lines/{line_id}/void
POST /api/v1/orders/{order_id}/prechecks
POST /api/v1/prechecks/{precheck_id}/cancel
GET  /api/v1/orders/{order_id}
GET  /api/v1/prechecks/{precheck_id}
```

---

## 19.4 Payments & checks

```text
POST /api/v1/prechecks/{precheck_id}/payments
POST /api/v1/payments/{payment_id}/refund
GET  /api/v1/checks/{check_id}
```

---

## 19.5 Device binding

```text
POST /api/v1/devices/binding-codes
POST /api/v1/devices/bind
GET  /api/v1/devices/current
```

---

## 19.6 KDS / DishServed

```text
POST /api/v1/order-lines/{line_id}/served
GET  /api/v1/kds/tasks
POST /api/v1/kds/tasks/{task_id}/done
```

KDS API может быть отложен, но `DishServed` event model должен быть учтен до inventory.

---

# 20. MVP-0 Definition

Первый технически значимый MVP для пилота в Индонезии:

```text
One Restaurant
One Primary Edge Node
Framework-Agnostic UI, React/Vite for MVP
Stable Device Identity
Shift Opening/Closing
Cash Session
Order Creation
Order Line Management
Issue Precheck as Snapshot
Local Manager Override PIN for Precheck Cancel
Trusted Payment: Cash/Card terminal manual confirmation
Final Check Generation after full payment
SQLite WAL
Transactional local_event_log + pos_sync_outbox
Cloud Sync Receiver with idempotency
Outbox retry/backoff/failed status
Basic SQLite snapshot backup
```

MVP-0 успешен, если:

```text
касса автономно обслуживает полный цикл гостя
интернет не нужен для критических операций
менеджерский PIN требуется для опасных действий
финальный чек создается только после полной оплаты
Cloud без потерь и дублей собирает журнал операций после восстановления сети
failed sync можно вручную вернуть в pending
данные можно восстановить из snapshot
```

---

# 21. First Launch Readiness

Отдельный checkpoint перед первым пилотным запуском.

Важно: это не migration checkpoint. Это readiness для первого реального запуска на чистой схеме v1.3.

---

## 21.1 Functional readiness

Должно работать:

```text
open shift
open cash session
create order
add order lines
issue precheck
cancel precheck via manager PIN
capture cash payment
capture trusted card payment
auto-create final check after full payment
close order
close cash session
close shift
```

---

## 21.2 Offline readiness

Проверки:

```text
создать полный guest flow без интернета
перезапустить Edge во время offline
проверить сохранность SQLite
после возврата сети отправить outbox
проверить отсутствие дублей в Cloud
```

---

## 21.3 Financial integrity readiness

Проверки:

```text
нельзя создать order без active shift
нельзя issue precheck без active shift
нельзя изменить locked order
нельзя создать check до полной оплаты
нельзя переплатить precheck
нельзя отменить precheck без manager PIN
нельзя refund без manager PIN
Payment immutable
Check immutable
```

---

## 21.4 Sync readiness

Проверки:

```text
idempotent replay returns stable ack
same idempotency key with different payload returns conflict
retry backoff works
attempts increments
next_retry_at respected
attempts > 20 becomes failed
Retry Failed Syncs returns events to pending
```

---

## 21.5 Device readiness

Проверки:

```text
new device cannot operate without binding
bound device has stable UUID
device_id appears in all SyncEnvelope payloads
device_id survives restart
replacement device gets new UUID
```

---

## 21.6 Recovery readiness

Проверки:

```text
SQLite snapshot created
snapshot uploaded to Cloud or stored locally for upload
snapshot checksum verified
restore procedure documented
outbox survives restore
```

---

# 22. Roadmap Development Sequence

Подробный roadmap живет в `ROADMAP_MVP.md`. Здесь фиксируется архитектурная последовательность.

---

## Stage 0 & 1 — Edge Core Skeleton & Cloud Sync

Статус: done.

Состав:

```text
Go Edge Server
SQLite
local_event_log
pos_sync_outbox
SyncEnvelope
Cloud Sync Receiver
PostgreSQL receiver storage
idempotency dedupe
```

---

## Stage 2 — Cash & Shifts Core

Статус: done / mostly done.

Состав:

```text
Open/Close Shift
Cash Sessions
Cash Drawer Events
payment_attempts foundation
```

---

## Stage 3 — Orders, Prechecks & Taxes

Статус: next / current.

Состав:

```text
replace CreateCheck with IssuePrecheck
prechecks table
precheck_lines
precheck_tax_lines
precheck versioning
one issued precheck per order
order locked state
manager override for cancel_precheck
generic tax_profiles
snapshot tax calculation
```

---

## Stage 4 — Payments & Final Checks

Состав:

```text
trusted cash/card payments
payment linked to precheck
partial payments if needed
prevent overpayment
final Check generation after full payment
refund via manager override
immutability tests
```

---

## Stage 5 — Catalog, Recipes & Menu Updates

Состав:

```text
Cloud catalog source
Cloud → Edge menu/tax snapshot
catalog item tax profile assignment
recipe versions foundation
```

---

## Stage 6 — KDS & DishServed

Состав:

```text
KDS task routing
mark dish done
DishServed event
MVP fallback auto_on_check
```

---

## Stage 7 — Inventory Ledger

Состав:

```text
Cloud immutable stock ledger
receipt documents
recipe_consumption from DishServed
manual adjustment/write_off
reversal
```

---

## Stage 8 — Android/Hardware Layer

Состав:

```text
WebView kiosk
Go process supervisor
printer bridge
device persistent storage
power/sleep handling
boot recovery
```

---

## Stage 9 — Backup & Reconciliation

Состав:

```text
SQLite snapshot upload
restore procedure
bank statement CSV import
trusted terminal payment matching
```

---

## Stage 10 — ClickHouse Analytics

Состав:

```text
OLAP schema
food cost
AVCO
sales dashboards
inventory dashboards
```

---

## Stage 11 — Real PSP Integrations & Evidence Archive

Состав:

```text
Adyen/Stripe/local PSPs
webhooks
signature validation
PCI-safe raw payload storage
Payment Evidence Archive
```

---

# 23. Anti-Patterns

Строгие запреты:

```text
business logic in React/Vite
business logic in Android/Kotlin shell
UI directly calls Cloud
Create Check before full payment
Silent mutation of issued Precheck
Editing locked Order without Manager Override
Deleting Payment
Editing Payment
Deleting Check
Deleting posted Stock Document
Hardcoded taxes/PB1 in Go logic
Inventory UPDATE counters directly
Multi-master Edge in MVP
Outbox write outside transaction
local_event_log write outside transaction
split transaction for one business use case
random device_id on startup
MAC address as device_id
DishServed on OrderLineAdded
Cloud as runtime dependency for POS writes
```

---

# 24. Testing Requirements

Каждый новый write use case должен иметь тесты:

```text
happy path
invalid state transition
transactional outbox/local_event_log presence
idempotency or duplicate protection where relevant
boundary case
```

Обязательные тесты для Stage 3/4:

```text
IssuePrecheck creates version 1
IssuePrecheck increments version
IssuePrecheck supersedes previous issued precheck
unique order/version constraint exists
one issued precheck per order constraint exists
locked order cannot accept new lines
cancel precheck requires manager override
cancel precheck unlocks order
payment cannot exceed precheck total
check created only after full payment
payment immutable
refund requires manager override
```

Обязательные sync tests:

```text
outbox event created in same transaction
retry backoff sets next_retry_at
attempts > 20 sets failed
failed not picked by worker
manual retry resets failed to pending
Cloud duplicate replay stable ack
Cloud payload conflict returns conflict
```

---

# 25. Правила для AI-итераций и генерации кода

Все ответы, промпты, комментарии задач и документация по проекту должны быть на русском языке, если пользователь явно не попросил иначе.

Перед генерацией кода AI должен учитывать:

```text
AGENTS.md
ROADMAP_MVP.md
SPECv1.3.md
README.md
актуальную структуру репозитория
```

При изменении backend capabilities нужно обновлять:

```text
AGENTS.md
README.md
pos-backend/README.md, если изменились команды/API
ROADMAP_MVP.md, если изменился статус этапов
SPECv1.3.md, если изменилось архитектурное решение
```

Код должен сохранять:

```text
Clean Architecture
SQLite transaction per write use case
local_event_log + outbox in same transaction
domain invariants
Russian task descriptions/prompts/docs
```

---

# 26. Final Conclusion

SPEC v1.3 готова к разработке MVP-0.

Ключевой переход версии:

```text
from: Order → Check → Payment
  to: Order → Precheck → Payment → Check
```

Ключевые технические фиксации v1.3:

```text
Precheck versioning under SQLite transaction
unique order/version constraint
only one active issued precheck per order
outbox exponential backoff
SQLite dead-letter via failed status
manual Retry Failed Syncs
stable Device UUID via binding flow
DishServed only when dish is actually served
no DB migration before first launch
```

Главный фокус ближайшей разработки:

```text
Stage 3: Orders, Prechecks & Taxes
Stage 4: Payments & Final Checks
First Launch Readiness
```

---

# Codex Usage Note

Эти файлы являются актуальными pilot-freeze источниками для формирования промптов Codex. Перед генерацией кода агент должен использовать их вместе с `AGENTS.md`, `README.md`, текущей структурой репозитория и существующими тестами. Запрещено возвращаться к старой модели `Order -> Check -> Payment`, хранить деньги не как minor units, делать live backup через прямое копирование active WAL DB, хранить PIN/OTP в plaintext или считать печать частью финансового commit.
