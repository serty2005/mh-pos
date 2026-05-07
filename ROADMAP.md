# ROADMAP

## Назначение

Этот документ описывает:

- что уже завершено;
- что обязательно закрыть до первого пилота;
- что остается после пилота;
- основные риски и меры снижения риска.

## Статусы

Используются только статусы:

- `done`
- `in_progress`
- `blocked`
- `next`
- `post_pilot`

## Что уже сделано

### Foundation

Статус: `done`

- Edge backend на Go + SQLite
- canonical SQLite first-launch path
- SQLite runtime gate
- `local_event_log`
- `pos_sync_outbox`
- retry-safe outbox foundation
- explicit directional sync ownership foundation
- Cloud -> Edge master sync metadata/checkpoint schema foundation
- Cloud sync receiver foundation
- pairing foundation
- auth session foundation
- halls/tables foundation
- shifts foundation
- cash sessions foundation
- cash drawer events foundation
- local E2E demo bootstrap и smoke scripts

### Sales runtime

Статус: `done`

- публичный runtime `Order -> Precheck -> Payment -> Check`
- issue precheck
- list/get prechecks
- manager override cancel precheck
- precheck-based payments
- automatic final check
- automatic order close

### UI cashier slice

Статус: `done`

- `/pair`
- `/login`
- `/pos`
- `/lock`
- hall/table selection
- order editing
- issue/cancel precheck
- cash payment
- trusted manual card payment
- final check display

## Что обязательно закрыть до первого пилота

### Documentation freeze

Статус: `next`

Нужно:

- ввести новый `AGENTS.md`
- заменить устаревший UI spec на текущий cashier-first spec
- добавить отдельный UI RBAC document
- добавить отдельный backend API/spec document
- добавить отдельный backend data/migration policy document
- перестать документировать future modes как current runtime

### Очистка compatibility-хвостов

Статус: `done`

Сделано:

- удалены public compatibility endpoints старой check/payment модели;
- `device_id` больше не описывается как transport compatibility alias; он остается domain/storage field для POS Edge node identity в operational payload;
- canonical transport examples используют `node_device_id` и `client_device_id`.

### Выравнивание sync contract

Статус: `done`

Сделано:

- Cloud принимает фактический Edge -> Cloud operational event catalog;
- production sender path имеет direction gate и не отправляет Cloud-managed/configuration события вверх;
- `pos_sync_outbox.sync_direction` явно разделяет `edge_to_cloud`, `cloud_to_edge` и `local_only`;
- Edge runtime mutation Cloud-owned master data запрещен application boundary;
- ownership matrix добавлена в `docs/sync/directional-sync-ownership.md`;
- canonical Edge/Cloud sync contract обновлен в `docs/sync/edge-cloud-contracts-v1.md`;
- POS sender включен как отдельный background worker с retry/backoff, stale lock reclaim и idempotent resend;
- Cloud хранит raw envelopes и append-safe operational event journal.

planned next:

- item-level ACK plan;
- richer Cloud projections поверх `cloud_operational_events`;
- production Cloud -> Edge provisioning/import endpoints для master/reference/configuration данных.

### Security hardening

Статус: `next`

Нужно:

- перевести pairing verifier на keyed format
- зафиксировать policy уникальности PIN либо employee selection login flow
- добавить/задокументировать rate limiting для PIN attempts
- проверить, что PIN и manager PIN не попадают в logs/events/storage

### RBAC hardening

Статус: `next`

Нужно:

- перейти от ad-hoc permission strings к canonical permission catalog
- описать роли cashier / senior_cashier / waiter / manager / kitchen / support_admin
- привязать UI visibility к permission model
- расширить backend enforcement beyond current manager override minimum

### Pilot scope hardening

Статус: `next`

Нужно явно решить до пилота:

- поддерживаются ли только валюты с 2 decimal places;
- вводится ли `business_date_local` как pilot blocker;
- нужен ли reprint в pilot scope;
- допускается ли waiter payment path в pilot scope;
- какие diagnostics доступны менеджеру, а какие только support/admin.

## Что можно оставить после пилота

Статус: `post_pilot`

- waiter UI runtime
- KDS runtime
- manager runtime
- settings runtime
- diagnostics runtime expansion
- PSP integration
- refund ledger flow
- print adapter layer
- inventory write-off from `DishServed`
- full Cloud projections
- advanced analytics
- multi-device / multi-client coordination beyond pilot topology

## Мильстоуны

### Pilot docs freeze

Статус: `next`

Критерий:

- весь runtime surface описан отдельными документами;
- нет устаревших основных спецификаций;
- нет противоречий между README, UI docs, backend docs и roadmap.

### Pilot API freeze

Статус: `next`

Критерий:

- compatibility endpoints удалены;
- event catalog опубликован;
- first-launch API не содержит unresolved public compatibility tails.

### Pilot hardening freeze

Статус: `next`

Критерий:

- pairing/PIN policy закрыта;
- RBAC matrix утверждена;
- supported currency/business-date policy зафиксирована;
- print/reprint policy зафиксирована.

### Pilot readiness

Статус: `blocked`

Критерий:

- sync contract aligned;
- security hardening closed;
- docs freeze closed;
- no unresolved critical compatibility tails.

## Риски и mitigation

| Риск | Влияние | Вероятность | Митигирующее действие |
|---|---|---|---|
| Документация обещает больше, чем реально поддерживает runtime | Высокое | Высокая | Разделить docs по владельцам и обновлять их в одном PR |
| Старый compatibility endpoint вернется в public surface | Среднее | Средняя | Проверять `rg` по API routes/docs перед freeze |
| Edge/Cloud event catalog снова расходится | Высокое | Средняя | Поддерживать canonical catalog в `docs/sync/edge-cloud-contracts-v1.md` и тестировать sender direction gate |
| Pairing verifier остается plain hash | Высокое | Средняя | Перейти на keyed verifier до пилота |
| Duplicate PIN / ambiguous login | Высокое | Средняя | Ввести уникальность PIN или employee-first login |
| RBAC остается неявным | Среднее | Высокая | Утвердить permission catalog и matrix |
| Пилотные assumptions по валюте и business date не зафиксированы | Высокое | Средняя | Зафиксировать policy в backend/data docs |
| Reprint нужен операционно, но не описан и не реализован | Среднее | Средняя | Либо убрать из pilot scope, либо реализовать и зафиксировать |

## Последовательность работ

```mermaid
flowchart LR
    Docs["Docs freeze"] --> Tails["Compatibility tail cleanup"]
    Tails --> Sync["Sync contract alignment"]
    Sync --> Security["Security hardening"]
    Security --> RBAC["RBAC hardening"]
    RBAC --> Pilot["Pilot readiness gate"]
```

## Правило stop-doing

До первого пилота нельзя тратить время на:

- historical DB migrations для несуществующего production;
- dual-write;
- сохранение obsolete API ради “может пригодится”;
- расширение future modes без фиксации текущего cashier pilot scope.

## Критерии готовности pre-pilot изменений

Изменение считается завершенным только если:

- код и тесты обновлены;
- профильная документация обновлена;
- roadmap status изменен;
- compatibility tail удален из public surface;
- изменение не создало новый historical хвост.
