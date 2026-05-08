# Модель данных POS и policy миграций

## Назначение

Документ описывает:

- ключевые сущности локального Edge runtime;
- связи между сущностями;
- обязательные инварианты данных;
- first-launch migration policy;
- правила изменения схемы до первого пилота.

## Главный принцип

До первого пилота действует **reset policy**, а не legacy migration policy.

Это означает:

- нет production data, которую нужно сохранять;
- canonical clean schema остается в first-launch init script;
- runtime upgrade выполняется только программно на старте модулей;
- dev/test базы можно пересоздавать, но startup больше не должен падать голой SQL-ошибкой из-за отсутствующей runtime-таблицы.

## Канонический SQLite path

Pilot path для SQLite:

- `001_init.sql` остается managed baseline SQL file для clean init;
- `002_runtime_schema_repair.sql` является ordered repair migration для implemented-now runtime columns, которые появились после ранних pre-pilot SQLite БД (`business_date_local`, business-day settings, immutable snapshots);
- служебные `schema_migrations` и `db_runtime_versions` создаются кодом startup framework;
- `schema_migrations` содержит имя каждого managed SQL file, checksum, status и `applied_at`;
- checksum drift допускается только как часть version upgrade (`db version < MH_POS_VERSION`), при той же версии drift завершает startup fail-fast;
- дополнительные SQL files добавляются только для implemented-now runtime schema repair/upgrade с tests, backup behavior и профильной документацией.

## PostgreSQL startup migration path

implemented now:

- Cloud PostgreSQL использует ordered managed SQL files из `cloud-backend/migrations/postgres`;
- `001_sync_receiver.sql` содержит baseline receiver, operational journal, shift finance, master-data package и currency reference storage;
- `002_projection_event_type_stats.sql` создает required runtime table `cloud_projection_event_type_stats`, потому что Cloud receiver runtime выполняет `INSERT ... ON CONFLICT` в эту projection при приеме Edge events;
- `003_runtime_schema_repair.sql` idempotent-образом довыравнивает весь implemented-now Cloud runtime schema set для старых БД, где history уже содержит ранние migrations, но отдельные runtime tables отсутствуют;
- `schema_migrations` хранит отдельную запись с checksum/status для каждого SQL file, поэтому повторный startup не применяет уже recorded migrations повторно;
- missing `db_runtime_versions` означает oldest DB и запускает upgrade path, а не immediate runtime crash;
- schema verification проверяет только implemented-now runtime schema.

planned next:

- projection query endpoints для ops dashboards;
- richer reporting/analytics projections поверх PostgreSQL/ClickHouse pipeline.

out of scope:

- manual SQL patching outside startup migration framework;
- BI/OLAP/reporting engine в runtime startup path.

## Владение сущностями

Архитектурная карта владения находится в `docs/architecture/DDD-CONTEXT-MAP.md`. Этот документ описывает только схему, связи, инварианты и migration/reset policy.

Краткая привязка таблиц к контекстам:

- `Organization`: `restaurants`, `devices`, `edge_node_identity`, `client_devices`, `roles`, `employees`, `auth_sessions`.
- `Reservation / Table`: `halls`, `tables`.
- `Catalog`: `catalog_items`, `menu_items`, `recipe_versions`, `recipe_lines`.
- `Pricing`: сейчас использует упрощенное хранение `menu_items.price`; это MVP price surface, а не финальное владение pricing внутри `Catalog`.
- `Order`: `orders`, `order_lines`, `prechecks`.
- `Payment`: `payments`, `payment_attempts`.
- `Fiscal / Tax`: `checks` как финальный immutable document foundation; real fiscalization вне текущего runtime.
- `Staff / Shift`: `shifts`, `cash_sessions`, `cash_drawer_events`, `manager_override_audit`.
- `Inventory`: `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, recipe consumption foundation.
- `Procurement`: `purchase_receipts`, `purchase_receipt_lines` существуют как inventory/procurement foundation; полный `Procurement` context остается после пилота, если не будет отдельного решения.
- `Event / Integration`: `local_event_log`, `pos_sync_outbox`, `cloud_master_sync_state`.

## Ключевые runtime-сущности

### Identity и организация

- `restaurants`
- `devices`
- `edge_node_identity`
- `client_devices`
- `roles`
- `employees`
- `auth_sessions`

### Залы и sales runtime

- `halls`
- `tables`
- `catalog_items`
- `menu_items`
- `shifts` как личные смены сотрудников
- `orders`
- `order_lines`
- `prechecks`
- `checks`
- `payments`
- `payment_attempts`

### Касса и sync

- `cash_sessions` как кассовые смены
- `cash_drawer_events`
- `manager_override_audit`
- `cloud_master_sync_state`
- `local_event_log`
- `pos_sync_outbox`

### Foundation будущего inventory

- `recipe_versions`
- `recipe_lines`
- `purchase_receipts`
- `purchase_receipt_lines`
- `stock_documents`
- `stock_moves`
- `stock_balances`
- `item_costs`

## Схема связей

Реализовано сейчас:

- `auth_sessions` фиксируют техническую авторизацию `node_device_id + client_device_id + employee_id` и не являются рабочей сменой.
- `shifts` фиксируют личную смену сотрудника. Открытая личная смена уникальна для пары `restaurant_id + opened_by_employee_id`.
- `cash_sessions` фиксируют кассовую смену устройства и открываются только при открытой личной смене текущего сотрудника.
- `orders.shift_id`, `payments.shift_id`, `cash_sessions.shift_id`, `cash_drawer_events.shift_id` указывают на личную смену сотрудника.

Запланировано далее:

- Личная смена сотрудника станет входной сущностью для post-MVP учета рабочего времени.

```mermaid
erDiagram
    RESTAURANTS ||--o{ DEVICES : has
    RESTAURANTS ||--o{ EMPLOYEES : employs
    ROLES ||--o{ EMPLOYEES : grants
    DEVICES ||--o{ CLIENT_DEVICES : registers
    DEVICES ||--o{ SHIFTS : records
    EMPLOYEES ||--o{ SHIFTS : works
    SHIFTS ||--o{ CASH_SESSIONS : opens
    RESTAURANTS ||--o{ HALLS : has
    HALLS ||--o{ TABLES : contains
    RESTAURANTS ||--o{ ORDERS : owns
    TABLES ||--o{ ORDERS : serves
    SHIFTS ||--o{ ORDERS : records
    ORDERS ||--o{ ORDER_LINES : contains
    ORDERS ||--o{ PRECHECKS : snapshots
    PRECHECKS ||--o{ PAYMENTS : allocates
    ORDERS ||--o| CHECKS : finalizes
    PAYMENTS ||--o{ PAYMENT_ATTEMPTS : attempts
    ORDERS ||--o{ MANAGER_OVERRIDE_AUDIT : audited
    PRECHECKS ||--o{ MANAGER_OVERRIDE_AUDIT : audited
    AUTH_SESSIONS ||--o{ LOCAL_EVENT_LOG : tags
    AUTH_SESSIONS ||--o{ POS_SYNC_OUTBOX : tags
```

## Обязательные текущие инварианты

### Orders

- только один активный order per selected runtime context;
- order открывается только при активной смене;
- order блокируется при issue precheck;
- редактирование order запрещено при active issued precheck;
- order закрывается только после полной оплаты и final check.

### Prechecks

- активным может быть только один `issued` precheck на order;
- у precheck должен быть положительный `version`;
- `paid_total` не может превышать `total`;
- terminal precheck state требует `closed_at`.

### Payments

- payment immutable;
- payment ссылается на `precheck_id`, а не на legacy `check_id`;
- `business_date_local` вычисляется backend в момент capture payment и после записи immutable;
- payment attempt - отдельная сущность истории попыток.

### Checks

- final check immutable после создания;
- `business_date_local` вычисляется backend в момент создания final check;
- `closed_at` хранит фактическое время закрытия;
- `snapshot` хранит immutable JSON source для reprint.

### Prechecks

- `snapshot` хранит immutable JSON source для reprint precheck;
- reprint не использует текущее состояние order.

### Outbox

- `sequence_no` - канонический local ordering key;
- `sync_direction` явно фиксирует `edge_to_cloud`, `cloud_to_edge` или `local_only`;
- запись в business tables, `local_event_log` и `pos_sync_outbox` должна быть транзакционной;
- failed/suspended retry выполняется через явный operational path;
- item-level batch ACK реализован: Edge sender применяет per-item результаты Cloud batch endpoint к lifecycle строк `pos_sync_outbox`.

### Directional ownership

Реализовано сейчас:

- Cloud-owned master tables: `restaurants`, `devices`, `roles`, `employees`, `halls`, `tables`, `catalog_items`, `menu_items`, `recipe_versions`, `recipe_lines`, `item_costs`.
- Эти таблицы являются локальной read model на POS Edge; application services запрещают Edge runtime mutation и принимают только `origin = cloud_sync` или `origin = system_seed`.
- Реализовано сейчас: Cloud-authored rows применяются на POS Edge через `POST /api/v1/sync/master-data/snapshots` или `POST /api/v1/sync/master-data/{stream}`. Supported Edge ingest streams: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.
- Вне текущего объема: POS Edge apply for `currencies` stream. Cloud backend owns canonical ISO 4217 reference/provisioning, while current POS Edge currency validation uses the local canonical catalog.
- Cloud-owned master tables имеют `cloud_version`, `cloud_updated_at`, `cloud_deleted_at`, `last_synced_at`.
- `cloud_master_sync_state` хранит Cloud -> Edge stream checkpoint: stream, mode, checkpoint token, last Cloud version/update, last apply time, status/error.
- Master-data ingest writes master rows and `cloud_master_sync_state` in one transaction and does not write `local_event_log` or `pos_sync_outbox`.
- Edge-owned operational tables: `shifts`, `cash_sessions`, `cash_drawer_events`, `orders`, `order_lines`, `prechecks`, `payments`, `payment_attempts`, `checks`, `manager_override_audit`, `auth_sessions`, `local_event_log`, `pos_sync_outbox`.

## Обязательные policy-решения до первого пилота

### Money contract

Для новых и меняемых финансовых полей использовать:

- signed integer minor units;
- explicit currency code;
- no REAL/FLOAT money storage.

### Business date

Реализовано сейчас:

- `restaurants.business_day_mode` задает режим `standard` или `24_7`;
- `restaurants.business_day_boundary_local_time` хранит ресторанную границу дня в формате `HH:MM`;
- `checks.business_date_local` и `payments.business_date_local` являются обязательными финансовыми полями;
- `shifts.business_date_local` и `cash_sessions.business_date_local` сохраняют учетный день открытия для связности отчетов;
- в `standard` режиме backend вычисляет учетный день по локальному времени ресторана с учетом границы дня;
- в `24_7` режиме backend использует локальную календарную дату события;
- после создания checks/payments `business_date_local` immutable.

### Print snapshots

Реализовано сейчас:

- `prechecks.snapshot` хранит immutable JSON snapshot на момент issue precheck;
- `checks.snapshot` хранит immutable JSON snapshot на момент final check creation;
- reprint precheck/check читает только snapshot;
- reprint events `PrecheckReprinted` / `CheckReprinted` пишутся в `local_event_log` и `pos_sync_outbox`.

### Pairing secret verifier

Реализовано сейчас: verifier-side storage pairing code использует keyed format `pairing.hmac-sha256.v1`.
Plain hash для pairing verifier запрещен в canonical first-launch schema/runtime.

### Employee credential policy

Реализовано сейчас: PIN login должен однозначно определить одного active employee в пределах paired restaurant.
Если один PIN совпал с несколькими active employees, login отклоняется как policy conflict.
Employee selection login flow остается вне текущего объема для текущего cashier-first pilot surface.

## Вне текущего объема до отдельного пилотного решения

Без отдельного пилотного решения не считаются реализованными сейчас:

- `precheck_lines` snapshots;
- `precheck_tax` snapshots;
- полный refund ledger flow;
- hardware printer adapter layer;
- ручной перенос closed order/payment в другой business date;
- broadly enforced STRICT tables across all financial tables.

## Правило изменения схемы

Любое изменение схемы до первого пилота делается так:

1. меняется managed SQL path (ordered SQL files в `pos-backend/migrations/sqlite` для POS Edge SQLite и `cloud-backend/migrations/postgres` для Cloud PostgreSQL);
2. повышается `MH_POS_VERSION` для version-gated upgrade существующей БД;
3. обновляются schema verification requirements и tests;
4. при необходимости обновляются seed/tests;
5. обновляются backend docs;
6. dev/test DBs можно пересоздать, но ручной SQL upgrade не становится canonical path.

Нельзя:

- добавлять второй/третий SQL migration file only because “так привычнее”, без implemented-now runtime необходимости, startup framework, tests, backup и schema verification;
- сохранять legacy поля ради несуществующей production совместимости;
- добавлять новый financial path рядом со старым.

## Runtime version contract и backup-before-upgrade

implemented now:

- для `SQLite` и `PostgreSQL` поддерживается таблица `db_runtime_versions` (module name -> module version, schema version, checksum, status);
- `schema_migrations` хранит managed SQL files, checksum, status и время применения;
- модульные версии берутся из единого `MH_POS_VERSION` (fallback `0.1.0`);
- если `db_runtime_versions` отсутствует, БД считается самой старой и запускается upgrade path, а не голый runtime crash;
- если `schema_migrations` отсутствует или содержит старую запись без checksum, startup framework повторно применяет idempotent managed SQL и записывает migration history только после успешного apply;
- upgrade выполняется программно на старте модулей до HTTP/workers;
- если обнаружена ситуация `db version < module version`, отсутствующая version table или изменение данных через startup/data-load path, перед обновлением обязателен backup:
  - POS Edge (`SQLite`): файловый backup DB/WAL/SHM после WAL checkpoint;
  - Cloud (`PostgreSQL`): JSONL snapshot таблиц `public` схемы;
- `db version > MH_POS_VERSION` завершает startup fail-fast, downgrade не поддерживается;
- после managed SQL apply выполняется schema verification только implemented-now runtime tables/columns/indexes;
- checksum drift active SQL file при той же версии завершает startup fail-fast;
- checksum drift active SQL file при `db version < MH_POS_VERSION` считается управляемым upgrade и фиксируется в `schema_migrations` после успешного apply.

planned next:

- добавить retention/purge policy для backup-артефактов и централизованный мониторинг успешности backup-before-upgrade;
- добавить backup-before-data-load для Cloud -> Edge full snapshot/master-data import, чтобы загрузка данных не могла заменить локальную read model без recoverable snapshot;
- добавить административную UI-операцию очистки/пересоздания SQLite для случаев коллизий, повреждения БД или неустранимого конфликта загрузки данных. Операция должна быть destructive-by-design: backend-admin endpoint, `support_admin`/service permission, явное подтверждение, backup до очистки, аудит и понятный restart/rebootstrap flow.

## SQLite maintenance: VACUUM / VACUUM INTO

Реализовано сейчас:

- `VACUUM`, `VACUUM INTO`, `PRAGMA optimize` и `PRAGMA wal_checkpoint(TRUNCATE)` считаются явными maintenance/snapshot операциями.
- Эти операции не выполняются автоматически на каждом startup и не входят в обычный POS write path.
- `VACUUM` и `VACUUM INTO` запускаются только с явным подтверждением `-force`, чтобы не создать долгую блокирующую операцию случайно.
- PowerShell wrapper: `scripts/maintain-sqlite.ps1`.
- Go CLI: `pos-backend/cmd/sqlite-maintenance`.
- Команды выполняются вне активной write transaction.
- `VACUUM INTO` используется для compact snapshot/backup file, если нужен новый целевой файл; существующий target file не перезаписывается.

Примеры:

```powershell
.\scripts\maintain-sqlite.ps1 -DatabasePath "pos-backend\data\pos-edge.db" -Optimize -WalCheckpoint
.\scripts\maintain-sqlite.ps1 -DatabasePath "pos-backend\data\pos-edge.db" -Vacuum -Force
.\scripts\maintain-sqlite.ps1 -DatabasePath "pos-backend\data\pos-edge.db" -VacuumInto "pos-backend\data\snapshots\pos-edge.compact.db" -Force
```

Риски:

- `VACUUM` может занять заметное время на большой базе и требует свободное место.
- `VACUUM` не должен запускаться внутри active write transaction.
- Для production-like данных перед тяжелой maintenance-операцией нужен отдельный backup/snapshot.

Запланировано далее:

- добавить retention policy для maintenance snapshots;
- добавить отдельный health/report output с размером DB/WAL до и после операции.

## Правило compatibility-хвостов на уровне данных

Запрещены бессрочные data-model tails:

- legacy foreign key path;
- duplicate meaning columns;
- old/new enum names without removal plan;
- shadow table for obsolete runtime.

Если compatibility column временно нужен для transport layer, это должно быть явно отражено и иметь milestone удаления.

## Минимальный verification set

После изменений схемы разработчик обязан проверить:

- clean install проходит;
- `schema_migrations` содержит applied managed SQL files и checksum;
- runtime tests не создают legacy payment-to-check coupling;
- precheck lifecycle constraints и outbox constraints не сломаны;
- schema verification contract совпадает с фактическим managed migration path;
- implemented now POS Edge runtime columns из business-date/reprint итерации покрыты `002_runtime_schema_repair.sql`, потому что старая SQLite таблица не меняется через `CREATE TABLE IF NOT EXISTS`;
- implemented now Cloud projections (`cloud_projection_event_type_stats`, `cloud_projection_shift_finance`) имеют таблицы в PostgreSQL migrations и покрыты `003_runtime_schema_repair.sql`, потому что receiver runtime пишет в них при приеме Edge events;
- документация отражает новое состояние схемы.
