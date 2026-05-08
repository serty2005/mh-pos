# ADR-015: Стратегия хранения данных и аналитики

Статус: принято для текущего pre-pilot runtime.

## Контекст

Проект использует локальный POS Edge runtime и Cloud Sync Receiver. Для текущего pilot scope важны предсказуемость OLTP-хранилища, offline-first работа Edge и безопасное восстановление после проблем с локальной БД.

## Решение

Реализовано сейчас:

- SQLite используется как локальный POS Edge OLTP/source of truth для активных POS операций.
- PostgreSQL используется как Cloud OLTP/source of truth для sync receiver, operational journal, projections и master/reference packages.
- Active migration path до пилота использует один managed SQL file на модуль: `pos-backend/migrations/sqlite/001_init.sql` и `cloud-backend/migrations/postgres/001_sync_receiver.sql`.
- Версия и состояние БД фиксируются в `db_runtime_versions`; примененный managed SQL file фиксируется в `schema_migrations` с checksum/status.
- Изменение managed SQL file выполняется через повышение `MH_POS_VERSION`, backup-before-upgrade и schema verification до запуска runtime.
- Cloud `cloud_currency_reference` является частью managed PostgreSQL file и затем наполняется canonical active ISO 4217 catalog при startup.

Запланировано далее:

- sqlc может быть использован как основной persistence-подход после стабилизации схемы и package boundaries.
- ClickHouse может быть добавлен в Cloud как OLAP/reporting accelerator, наполняемый асинхронно из PostgreSQL/projection pipeline.
- Backup-before-data-load должен быть добавлен для Cloud -> Edge full snapshot/master-data import.
- UI/admin SQLite cleanup/reset flow должен быть реализован как destructive-by-design операция с backup, явным подтверждением, RBAC/audit и rebootstrap/restart path.

Вне текущего объема:

- GORM/Ent для POS Core financial/offline/sync-critical flows.
- ClickHouse как source of truth.
- Ручной ad-hoc SQL как canonical upgrade/repair path.
- Online zero-downtime production migration orchestration.

## Последствия

- Runtime не должен обращаться к business tables до startup upgrade и schema verification.
- Проблемы уровня missing table должны обнаруживаться на старте как управляемая диагностика, а не как случайная runtime SQL error.
- Любая загрузка данных, которая может заменить локальную read model или привести к коллизиям, должна иметь recoverable backup path.
