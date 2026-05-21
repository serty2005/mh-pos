# ADR-016: ClickHouse Immutable Event Store На UUIDv7

Статус: принято как замороженный принцип для дальнейшей реализации.

## Контекст

Cloud-centric Event-Driven Inventory и будущая KDS/POS аналитика требуют полного архива business events. PostgreSQL должен оставаться транзакционным Cloud OLTP store, а не бесконечным журналом всех промежуточных действий кассира, кухни и менеджера.

Требуется хранить не только финальные чеки, но и весь event trail: `OrderOpened`, `ItemAdded`, `ItemRemoved`, `CheckClosed`, `RefundRecorded`, `ItemServed`, `StopListUpdated` и другие domain events.

## Решение

ClickHouse становится immutable event archive для всех business events, созданных на Edge POS или KDS. PostgreSQL хранит текущий транзакционный срез, финальные документы, operational projections и очередь `inbox_events`.

Система не выполняет synchronous dual-write в PostgreSQL и ClickHouse при обработке HTTP/sync request от кассы.

Целевой data flow:

```text
Edge Outbox
  -> Cloud API (PostgreSQL inbox_events)
  -> Async Batch Forwarder
  -> ClickHouse raw_business_events
```

Порядок записи:

1. Edge POS отправляет batch событий из локального `outbox`.
2. Cloud API принимает batch, сохраняет события в PostgreSQL `inbox_events` и сразу отвечает `200 OK`.
3. Async Batch Forwarder читает `inbox_events`, собирает batch от 1 000 до 100 000 rows и загружает его в ClickHouse.
4. После успешного export worker помечает rows в PostgreSQL как `processed_for_olap = true`.
5. Обработанные rows старше retention window могут быть удалены из PostgreSQL, потому что ClickHouse является вечным архивом business events.

## UUIDv7 Standard

Все `event_id` должны быть UUIDv7.

Причина:

- UUIDv7 содержит millisecond timestamp в первых 48 bits.
- Natural order UUIDv7 совпадает с хронологией создания событий.
- ClickHouse MergeTree получает inserts, которые естественно дописываются ближе к концу дисковых блоков.
- Это снижает write amplification, предотвращает деградацию вставок при росте event store и ускоряет выборки по датам.
- `occurred_at` в ClickHouse извлекается из UUIDv7 event id.

## ClickHouse Table

Целевая таблица: `raw_business_events`.

Engine:

```sql
MergeTree
```

Обязательные колонки:

| Column | Type | Назначение |
| --- | --- | --- |
| `event_id` | UUID | UUIDv7 event id |
| `tenant_id` | UUID | tenant boundary |
| `restaurant_id` | UUID | restaurant boundary |
| `device_id` | UUID | source Edge/KDS device |
| `employee_id` | UUID | actor employee |
| `event_type` | String | domain event type |
| `occurred_at` | DateTime64 | timestamp extracted from UUIDv7 |
| `payload` | String | full original event body as JSON string |

Sorting key:

```sql
ORDER BY (tenant_id, event_type, event_id)
```

Partitioning:

```sql
PARTITION BY toYYYYMM(occurred_at)
```

Новые колонки под каждый event type не добавляются. Event-specific data хранится в `payload`.

## Retention And Archiving

PostgreSQL `inbox_events` является delivery queue и short-term operational buffer, а не вечным event archive.

Правила:

- `processed_for_olap = false` rows нельзя удалять из PostgreSQL.
- `processed_for_olap = true` rows можно удалять из PostgreSQL после retention window.
- Базовый retention window для PostgreSQL `inbox_events`: 3 месяца.
- Перед удалением worker или maintenance job должен полагаться на successful ClickHouse export state.
- ClickHouse `raw_business_events` хранит business events бессрочно.

## Use Cases

Anti-fraud audit:

- PostgreSQL может хранить только финальный чек на 10.
- ClickHouse сохраняет trail: `ItemAdded` на 100 и последующие `ItemRemoved` на 90 до `CheckClosed`.
- Это позволяет искать подозрительные паттерны void/remove/refund без нагрузки на OLTP.

Speed of Service:

- ClickHouse позволяет считать median и percentiles между `CheckClosed` на POS и `ItemServed` в KDS.
- Расчет не требует читать PostgreSQL transactional tables.

Data Lake:

- `raw_business_events` является основой для ABC analysis, cohort analysis, kitchen analytics, COGS audit и произвольной BI-аналитики.
- Новые отчеты строятся из immutable event trail без расширения OLTP schema под каждый аналитический вопрос.

## Последствия

- Любой event generator на Edge POS, KDS или Cloud boundary обязан создавать UUIDv7 `event_id`.
- Cloud API подтверждает прием Edge events после записи в PostgreSQL `inbox_events`, а не после ClickHouse export.
- ClickHouse downtime не должен блокировать POS sync acceptance, если PostgreSQL `inbox_events` доступен.
- Дубликаты должны отбрасываться по `event_id` на уровне ingestion/export policy.
- Для аналитической истории ClickHouse является source of truth; для transactional commands и текущего operational state source of truth остается PostgreSQL.

## Вне Текущего Объема

- Синхронная запись в PostgreSQL и ClickHouse в одном request path.
- Использование ClickHouse для transactional command validation.
- Моделирование отдельных ClickHouse колонок под каждый event type.
