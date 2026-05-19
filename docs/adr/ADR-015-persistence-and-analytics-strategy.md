# ADR-015: Стратегия хранения данных и аналитики

Статус: принято для текущего pre-pilot runtime; дополнено ADR-016 для ClickHouse immutable event archive.

## Контекст

Проект использует локальный POS Edge runtime и Cloud backend. Для текущего pilot scope важны предсказуемость OLTP-хранилища, offline-first работа Edge и безопасное восстановление после проблем с локальной БД.

## Решение

Реализовано сейчас:

- SQLite используется как локальный POS Edge OLTP/source of truth для активных POS операций.
- PostgreSQL используется как Cloud OLTP/source of truth для sync receiver, operational projections и master/reference packages.
- Active pre-pilot migration path использует один managed SQL baseline на runtime-модуль, который применяется runtime startup path.
- POS Edge migration files сейчас включают `migrations/sqlite/001_init.sql`.
- Cloud PostgreSQL migration files сейчас включают `migrations/postgres/001_init.sql`.
- До первого клиента существующие dev/test БД пересоздаются; data-preserving migrations начинаются после первого внедрения.
- Версия и состояние БД фиксируются runtime migration/versioning механизмом; schema verification выполняется до business runtime access.
- Persistence implementation сейчас написан вручную в repository/infrastructure layer.

Запланировано далее:

- `sqlc` может быть использован как основной persistence-подход после стабилизации схемы и package boundaries.
- ClickHouse должен быть добавлен в Cloud как immutable archive для всех business events и OLAP/reporting accelerator; детали UUIDv7, `raw_business_events`, no dual-write и retention зафиксированы в `docs/adr/ADR-016-clickhouse-immutable-event-store.md`.
- Backup-before-data-load должен оставаться обязательной границей для Cloud -> Edge full snapshot/master-data import.
- UI/admin SQLite cleanup/reset flow должен быть реализован как destructive-by-design операция с backup, явным подтверждением, RBAC/audit и rebootstrap/restart path.

Вне текущего объема:

- `sqlc` как уже внедренный persistence implementation.
- GORM/Ent для POS Core financial/offline/sync-critical flows.
- ClickHouse как transactional source of truth для command validation или текущего operational state.
- ClickHouse как POS transaction path dependency.
- Ручной ad-hoc SQL как canonical upgrade/repair path.
- Online zero-downtime production migration orchestration.

## Последствия

- Runtime не должен обращаться к business tables до startup upgrade и schema verification.
- Проблемы уровня missing table должны обнаруживаться на старте как управляемая диагностика, а не как случайная runtime SQL error.
- Любая загрузка данных, которая может заменить локальную read model или привести к коллизиям, должна иметь recoverable backup path.
- PostgreSQL остается Cloud transactional source of truth; ClickHouse становится source of truth для immutable historical business event trail.
