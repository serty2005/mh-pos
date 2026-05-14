# ADR-015: Стратегия хранения данных и аналитики

Статус: принято для текущего pre-pilot runtime.

## Контекст

Проект использует локальный POS Edge runtime и Cloud backend. Для текущего pilot scope важны предсказуемость OLTP-хранилища, offline-first работа Edge и безопасное восстановление после проблем с локальной БД.

## Решение

Реализовано сейчас:

- SQLite используется как локальный POS Edge OLTP/source of truth для активных POS операций.
- PostgreSQL используется как Cloud OLTP/source of truth для sync receiver, operational projections и master/reference packages.
- Active migration path использует managed SQL migration files, которые применяются runtime startup path.
- POS Edge migration files сейчас включают `001_init.sql` ... `003_pricing_policy_sync_foundation.sql`.
- Cloud PostgreSQL migration files сейчас включают `001_sync_receiver.sql` ... `007_refund_and_pricing_policy_hardening.sql`.
- Версия и состояние БД фиксируются runtime migration/versioning механизмом; schema verification выполняется до business runtime access.
- Persistence implementation сейчас написан вручную в repository/infrastructure layer.

Запланировано далее:

- `sqlc` может быть использован как основной persistence-подход после стабилизации схемы и package boundaries.
- ClickHouse может быть добавлен в Cloud как OLAP/reporting accelerator, наполняемый асинхронно из PostgreSQL/projection pipeline.
- Backup-before-data-load должен оставаться обязательной границей для Cloud -> Edge full snapshot/master-data import.
- UI/admin SQLite cleanup/reset flow должен быть реализован как destructive-by-design операция с backup, явным подтверждением, RBAC/audit и rebootstrap/restart path.

Вне текущего объема:

- `sqlc` как уже внедренный persistence implementation.
- GORM/Ent для POS Core financial/offline/sync-critical flows.
- ClickHouse как source of truth.
- ClickHouse как POS transaction path dependency.
- Ручной ad-hoc SQL как canonical upgrade/repair path.
- Online zero-downtime production migration orchestration.

## Последствия

- Runtime не должен обращаться к business tables до startup upgrade и schema verification.
- Проблемы уровня missing table должны обнаруживаться на старте как управляемая диагностика, а не как случайная runtime SQL error.
- Любая загрузка данных, которая может заменить локальную read model или привести к коллизиям, должна иметь recoverable backup path.
