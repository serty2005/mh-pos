# ROADMAP MVP RMS/POS Платформы v1.3

> Язык проекта, документации, промптов, комментариев к задачам и пользовательских сценариев: **русский**.  
> Исключение: имена Go-пакетов, структур, методов, SQL-таблиц, HTTP endpoints, enum-значений и технических идентификаторов остаются на английском.

---

## 1. Назначение документа

Этот документ является рабочим roadmap для разработки MVP RMS/POS платформы.

Его нужно прикладывать к каждой новой итерации генерации кода, чтобы агент или разработчик:

- сохранял архитектурные инварианты проекта;
- не возвращался к устаревшей модели `Order -> Check -> Payment`;
- не планировал миграции БД как обязательный этап до первого запуска;
- учитывал, что проект уже имеет реализованный foundation;
- развивал код в сторону первого реального запуска, а не абстрактной enterprise-архитектуры.

---

## 2. Текущий статус проекта

Текущий репозиторий уже содержит foundation для Edge-first POS/RMS:

- `pos-backend` — Go Edge Backend на SQLite;
- `cloud-backend` — минимальный Cloud Sync Receiver;
- `local_event_log`;
- `pos_sync_outbox`;
- `SyncEnvelope`;
- idempotent Cloud receive/dedupe;
- shifts;
- cash sessions;
- cash drawer events;
- prechecks lifecycle foundation: SQLite table, domain model, repository, dormant `IssuePrecheck`, order locking, app-level `CancelPrecheck`, version/paid_total fields;
- базовые orders/checks/payments;
- device registration foundation;
- SQLite migrations для первого запуска локальной БД;
- PostgreSQL migrations для Cloud sync receiver.

Текущая задача — не переписать foundation, а аккуратно перевести проект к новой доменной модели v1.3:

```text
Order -> Precheck -> Payment -> Check
```

Текущее ограничение: precheck foundation added, runtime flow still legacy. App-level `IssuePrecheck` уже locks order, app-level `CancelPrecheck` unlocks order после успешной отмены active issued precheck, но legacy `CreateCheck` и payment-to-check flow intentionally сохранены до полноценного Precheck Core и публичного precheck API.

---

## 3. Главный принцип первого запуска

На текущем этапе **не требуется миграция существующих production БД**.

Проект еще не был запущен в продакшене, нет реально работающих ресторанных БД, которые нужно сохранять или мигрировать.

Поэтому все новые вводные делаются как схема и поведение для **первого запуска**:

- можно менять структуру SQLite migrations;
- можно переименовывать текущие сущности, если это нужно для правильной модели MVP;
- можно заменить текущий `CreateCheck` на `IssuePrecheck`;
- можно привести стартовую схему к финальной форме v1.3;
- не нужно писать сложные data migrations для исторических заказов, оплат и чеков.

Запрещено тратить время MVP на:

- миграцию старых данных;
- backward compatibility для несуществующих production баз;
- dual-write старой и новой модели;
- сохранение устаревшего `Check` как рабочего счета гостя.

---

## 4. Архитектурные инварианты

### 4.1 Edge-first

POS должен работать без интернета.

Cloud не является runtime-зависимостью для:

- открытия смены;
- создания заказа;
- выпуска пречека;
- принятия оплаты;
- создания финального чека;
- отмены пречека менеджером;
- закрытия смены.

### 4.2 Primary Edge Node

В MVP у ресторана один активный Primary Edge Node.

Запрещено проектировать multi-master Edge до отдельного post-MVP этапа.

### 4.3 SQLite как локальный Source of Truth

SQLite на Edge — локальный источник истины для активных POS-операций.

Обязательные настройки SQLite при открытии соединения:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

### 4.4 Все write use cases выполняются транзакционно

Каждая write-операция выполняется в одной SQLite транзакции:

```text
BEGIN
  бизнес-логика
  запись доменных таблиц
  запись local_event_log
  запись pos_sync_outbox
COMMIT
```

Запрещено:

- писать outbox вне транзакции;
- писать local event вне транзакции;
- делать split transaction;
- коммитить бизнес-сущность без sync-события;
- коммитить sync-событие без бизнес-сущности.

### 4.5 UI не содержит бизнес-логики

Frontend, WebView, Android shell и Windows shell не считают:

- налоги;
- скидки;
- итоги заказа;
- статусы оплаты;
- возможность закрыть заказ;
- возможность отменить пречек.

Все бизнес-решения принимает Go Edge Server.


### 4.6 Pilot Topology

Для MVP-0 замораживается инфраструктурная модель:

- one all-in-one terminal;
- POS UI + Go Edge Backend + SQLite на одном устройстве;
- SQLite только на локальном диске;
- network FS и sync folders для active DB запрещены;
- печать — через один supported path: Network ESC/POS Printer.

### 4.7 SQLite Runtime Gate

Startup Edge backend обязан fail-fast проверять:

- `sqlite_version()`;
- `journal_mode`;
- `synchronous`;
- `foreign_keys`;
- `busy_timeout`.

Pilot baseline:

- functional minimum: `SQLite >= 3.37.0`;
- required production baseline for WAL pilot: `SQLite >= 3.51.3` либо pinned fixed backport `3.50.7/3.44.6`.

### 4.8 Print Failure Rule

Ошибка печати не откатывает precheck/payment/check commit. Система обязана поддержать `Reprint` из сохраненного snapshot.

### 4.9 Money, Time & Security Baseline

- Все деньги: signed `INTEGER minor units + currency_code`.
- Все write use cases: `BEGIN IMMEDIATE`.
- `business_date_local` обязателен для смен, cash sessions, prechecks, payments, checks.
- PIN/OTP/tokens не попадают в plaintext storage/logs/events.
- Binding code хранится как keyed verifier, default `HMAC-SHA-256(server_secret, code)`.

---

## 5. Целевая модель MVP-0

MVP-0 должен обеспечить полный ресторанный цикл:

```text
One Restaurant
One Primary Edge Node
SQLite WAL
Local POS UI
Open Shift
Create Order
Add Order Lines
Issue Precheck
Manager Override for Precheck Cancel
Trusted Payment: Cash / Autonomous Card Terminal
Automatic Final Check Generation
Outbox Sync to Cloud
Cloud Idempotency
Local Recovery Checks
First Launch Readiness
```

Критерий успеха MVP-0:

```text
Касса автономно обслуживает гостя от открытия смены до финального чека,
не теряет события при offline-режиме,
не создает дубли в Cloud после восстановления сети,
требует manager PIN для опасных операций,
и готова к первому пилотному запуску без миграции существующих БД.
```

---

## 6. Целевая доменная модель продаж

### 6.1 Order

`Order` — рабочая сущность официанта и кухни.

Допустимые статусы MVP:

```text
open
locked
cancelled
closed
```

Правила:

- заказ создается только при активной смене;
- в `open` можно добавлять позиции;
- после выпуска активного пречека заказ переводится в `locked`;
- в `locked` нельзя тихо добавлять или удалять позиции;
- для изменения после пречека нужен manager override и отмена пречека;
- `closed` заказ нельзя редактировать.

### 6.2 Precheck

`Precheck` — заблокированный финансовый snapshot для гостя.

Допустимые статусы MVP:

```text
issued
superseded
cancelled
paid
```

Правила:

- пречек создается из текущего состояния заказа;
- пречек фиксирует позиции, скидки, налоги и totals;
- активным может быть только один `issued` precheck на заказ;
- пречек нельзя редактировать;
- для изменения заказа после пречека нужно отменить текущий пречек;
- отмена пречека требует manager override.

### 6.3 Payment

`Payment` — immutable финансовый факт.

Правила:

- payment нельзя удалять;
- payment нельзя редактировать;
- ошибка исправляется refund/reversal/correction событием;
- в MVP-0 card payment является trusted payment от автономного терминала;
- raw PSP payloads не хранятся в MVP-0, потому что реальных PSP-интеграций еще нет.

### 6.4 Check

`Check` — финальный расчетный документ.

Правила:

- check создается только после полной оплаты precheck;
- check нельзя создать вручную до оплаты;
- check нельзя использовать как рабочий счет гостя;
- после создания check заказ закрывается.

---

## 7. Обязательные технические решения v1.3

### 7.1 Precheck Versioning

Риск: два официанта одновременно нажимают «Распечатать пречек».

Решение: в MVP race condition решается нативной SQLite транзакцией на одном Primary Edge Node.

Требования к реализации `IssuePrecheck`:

1. Внутри одной транзакции выполнить:

```sql
SELECT MAX(version_no)
FROM prechecks
WHERE order_id = ?;
```

2. Если результата нет — `version_no = 1`.
3. Иначе — `version_no = max + 1`.
4. В этой же транзакции перевести предыдущие активные пречеки заказа в `superseded`:

```sql
UPDATE prechecks
SET status = 'superseded'
WHERE order_id = ?
  AND status = 'issued';
```

5. Создать новый precheck со статусом `issued`.
6. Заблокировать заказ: `orders.status = 'locked'`.
7. Записать `local_event_log`.
8. Записать `pos_sync_outbox`.

Обязательные SQLite constraints:

```sql
CREATE UNIQUE INDEX idx_prechecks_order_version
ON prechecks(order_id, version_no);
```

Желательно также обеспечить один активный `issued` precheck на заказ:

```sql
CREATE UNIQUE INDEX idx_prechecks_one_issued_per_order
ON prechecks(order_id)
WHERE status = 'issued';
```

Acceptance criteria:

- параллельные вызовы `IssuePrecheck` не создают одинаковый `version_no`;
- на заказ не может быть два активных `issued` precheck;
- предыдущий `issued` precheck становится `superseded` при выпуске нового;
- при ошибке записи outbox весь выпуск precheck откатывается.

---

### 7.2 Outbox Guarantees

Риск: долгий offline-режим, лавина событий после восстановления сети, битые сообщения, вечный retry одного poison event.

Требования к `pos_sync_outbox`:

Добавить или закрепить поля:

```text
status: pending | processing | sent | failed | suspended
attempts INT NOT NULL DEFAULT 0
next_retry_at TIMESTAMP NULL
last_error TEXT NULL
locked_at TIMESTAMP NULL
locked_by TEXT NULL
sent_at TIMESTAMP NULL
```

Retry policy:

```text
next_retry_at = now + min(2 ^ attempts seconds, 5 minutes)
```

Правила worker:

- брать только `status = pending`;
- брать только события, у которых `next_retry_at IS NULL OR next_retry_at <= now`;
- отправлять batch-ами;
- не DDoS-ить Cloud после восстановления сети;
- при успешной отправке ставить `sent`;
- при временной ошибке увеличивать `attempts` и выставлять `next_retry_at`;
- при `attempts > 20` ставить `status = 'suspended'`;
- `suspended` автоматически не брать.

MVP dead-letter model:

Отдельная DLQ не нужна. Dead-letter реализуется в SQLite через `status = 'suspended'`.

Manager UI action:

```text
Retry Failed Syncs
```

Действие:

```sql
UPDATE pos_sync_outbox
SET status = 'pending', attempts = 0, next_retry_at = NULL, last_error = NULL
WHERE status IN ('failed', 'suspended');
```

Acceptance criteria:

- offline 8-24 часа не ломает очередь;
- при восстановлении сети outbox отправляется постепенно;
- один битый payload не блокирует всю очередь;
- повторная отправка не создает дублей в Cloud;
- менеджер может вручную вернуть failed/suspended события в retry.

---

### 7.3 Device Identity

Риск: нестабильный `device_id` ломает idempotency и audit trail.

Правило:

Устройство не генерирует себе production `device_id` самостоятельно.

Provisioning flow:

1. Новый планшет/ПК запускает POS UI.
2. UI показывает экран привязки устройства.
3. Менеджер вводит OTP-код привязки.
4. OTP создается в Cloud или Primary Edge, в зависимости от доступности сети и сценария первого запуска.
5. Primary Edge регистрирует устройство.
6. Primary Edge генерирует стабильный `Device UUID`.
7. UUID сохраняется в таблице `devices`.
8. UUID возвращается клиентскому устройству.
9. Клиент сохраняет UUID в устойчивом локальном хранилище:
   - Android Keystore / SharedPreferences с защитой;
   - Windows config file с ограниченными правами;
   - LocalStorage допустим только для раннего dev UI, но не как финальное production-хранилище.
10. Все последующие write-команды и `SyncEnvelope` используют этот UUID.

Правила:

- если устройство сгорело, новое устройство получает новый `device_id`;
- нельзя переиспользовать старый `device_id` без явной процедуры восстановления;
- нельзя брать MAC address, hostname или случайный ID на каждый запуск как production identity;
- `device_id` обязателен во всех write-командах, local events и outbox payloads.

Acceptance criteria:

- перезапуск приложения не меняет `device_id`;
- переподключение сети не меняет `device_id`;
- Cloud idempotency key стабилен;
- все события содержат `device_id`.

---

### 7.4 DishServed Timing

Риск: склад спишет ингредиенты до фактического приготовления.

Правило:

Списание ингредиентов запрещено по `OrderLineAdded`.

Складское событие создается только по `DishServed`.

Источники `DishServed`:

1. Если есть KDS:
   - повар нажимает «Готово» напротив позиции;
   - Edge создает `DishServed`;
   - событие попадает в outbox.

2. Если KDS еще нет в MVP:
   - кассир/официант отмечает позицию как «Выдано»;
   - либо система автоматически генерирует `DishServed` для еще не выданных позиций в момент успешной полной оплаты и создания финального `Check`.

Правила отмены:

- отмена позиции до `DishServed` не создает stock write-off;
- отмена позиции после `DishServed` не возвращает ингредиенты автоматически;
- после `DishServed` корректировка склада делается отдельным документом: акт порчи, списание, correction или reversal.

Acceptance criteria:

- добавление позиции в заказ не меняет склад;
- выпуск пречека не меняет склад;
- оплата сама по себе не должна списывать дважды уже served позиции;
- `DishServed` идемпотентен по `order_line_id` / `dish_served_event_id`;
- Cloud создает inventory ledger documents только на основании `DishServed`.

---

## 8. Roadmap по этапам

## Этап A. Architecture Lock v1.3

Статус: обязательный перед активной генерацией Stage 3/4.

Цель: закрепить новую модель, чтобы код больше не развивался вокруг устаревшего `CreateCheck` как рабочего счета.

Задачи:

- [ ] Добавить `SPECv1.3.md` в корень репозитория.
- [ ] Добавить этот файл как `ROADMAP_MVP.md`.
- [ ] Обновить `AGENTS.md` под v1.3.
- [ ] Обновить `README.md` под фактический статус Cloud и POS Edge.
- [ ] Явно указать: миграция production БД не нужна до первого запуска.
- [ ] Явно указать: `Check` создается только после полной оплаты.
- [ ] Явно указать: `Precheck` заменяет старый рабочий `Check`.
- [ ] Зафиксировать список событий SyncEnvelope v1.3.

- [ ] Зафиксировать pilot topology: one all-in-one terminal.
- [ ] Зафиксировать printing path: Network ESC/POS Printer.
- [ ] Зафиксировать print failure semantics: print failure != transaction rollback.
- [ ] Зафиксировать state machine table для `Order / Precheck / Payment / Check`.
- [ ] Зафиксировать pilot baseline SQLite: `>= 3.37.0` для `STRICT`, `>= 3.51.3` для production WAL pilot.
- [ ] Зафиксировать runtime gate для SQLite environment.
- [ ] Зафиксировать item-level ACK contract для Cloud batch ingest.
- [ ] Зафиксировать refund flow в MVP-0.
- [ ] Зафиксировать keyed verifier format для binding code.

Acceptance criteria:

- новый агент, прочитав `AGENTS.md`, `SPECv1.3.md` и `ROADMAP_MVP.md`, не предлагает старую модель `Order -> Check -> Payment`;
- roadmap содержит отдельный блок First Launch Readiness;
- документация говорит на русском языке.
- новый агент не предлагает direct file copy как supported live backup.
- новый агент не предлагает plaintext OTP/PIN storage.
- документация однозначно описывает printing, sync ordering и recovery.

---

## Этап B. Schema Reset для первого запуска

Статус: начат частично. Таблица `prechecks` уже расширена как preparatory lifecycle foundation; полный schema reset v1.3 еще не завершен.

Цель: привести SQLite и PostgreSQL стартовые схемы к v1.3 без сложных data migrations.

Важно:

Это не production migration. Это обновление стартовой схемы до первого запуска.

Задачи SQLite:

- [x] Добавить минимальную таблицу `prechecks` без переключения runtime flow.
- [ ] Добавить таблицу `precheck_lines`.
- [ ] Добавить таблицу `precheck_tax_lines` или JSON snapshot налогов, если это проще для MVP.
- [ ] Добавить таблицу `manager_override_logs`.
- [ ] Добавить/обновить `tax_profiles`.
- [ ] Добавить связь `payments.precheck_id`.
- [ ] Проверить необходимость сохранения `payments.check_id` как nullable post-finalization reference.
- [ ] Обновить `checks`, чтобы они были финальными документами, а не рабочими счетами.
- [ ] Добавить поля outbox retry policy: `attempts`, `next_retry_at`, `last_error`, `locked_at`, `locked_by`.
- [x] Добавить constraint для одного active `issued` precheck на order.
- [x] Добавить минимальный constraints набор для precheck versioning: `version`, unique `(order_id, version)`, `paid_total`, terminal statuses.
- [ ] Добавить `business_date_local` в `shifts`, `cash_sessions`, `prechecks`, `payments`, `checks`.
- [ ] Перевести money columns на signed `INTEGER minor units`.
- [ ] Добавить `currency_code`.
- [ ] Добавить `payments.entry_kind`.
- [ ] Добавить `payments.original_payment_id`.
- [ ] Добавить `pos_sync_outbox.sequence_no`.
- [ ] Добавить `STRICT` tables для новых финансовых сущностей.

Задачи PostgreSQL Cloud:

- [ ] Обновить accepted event types v1.3.
- [ ] Убедиться, что raw `SyncEnvelope` хранится без потерь.
- [ ] Не строить сложные projections до стабилизации Edge flow.
- [ ] Зафиксировать dedupe по `event_id`.
- [ ] Реализовать per-item ACK contract.
- [ ] Разделить `accepted`, `duplicate`, `retryable_error`, `terminal_error`.

Acceptance criteria:

- `go test ./...` проходит;
- новая пустая SQLite БД создается сразу в v1.3 форме;
- нет обязательных миграций исторических данных;
- старые тесты адаптированы под precheck lifecycle.

---

## Этап C. Precheck Core

Цель: реализовать `Order -> Precheck`.

Статус: начат как dormant foundation. Entity, repository, app-level `IssuePrecheck`, order locking, app-level `CancelPrecheck` и минимальный versioning foundation добавлены, но endpoint, full supersede flow и payment-to-precheck еще не внедрены.

Backend задачи:

- [x] Создать минимальную domain model `Precheck`.
- [ ] Создать domain model `PrecheckLine`.
- [ ] Создать domain model `PrecheckTaxLine` или tax snapshot.
- [x] Добавить минимальные repository ports:
  - `CreatePrecheck`;
  - `GetPrecheck`;
  - `GetActivePrecheckByOrder`;
- [x] Добавить часть расширенных repository ports:
  - `ListPrechecksByOrder`;
  - `UpdatePrecheckLifecycle`;
  - full `SupersedeIssuedPrechecksForOrder`/`NextPrecheckVersionForOrder` еще не выделены отдельными ports.
- [x] Реализовать SQLite repository для `CreatePrecheck`, `GetPrecheck`, `GetActivePrecheckByOrder`, `ListPrechecksByOrder`, `UpdatePrecheckLifecycle`.
- [x] Реализовать dormant use case `IssuePrecheck`.
- [ ] Заменить endpoint `POST /orders/{id}/check` на `POST /orders/{id}/precheck`.
- [ ] Старый endpoint либо удалить до первого запуска, либо оставить временно только как deprecated dev endpoint, который вызывает `IssuePrecheck`.

Бизнес-правила:

- [ ] Нельзя выпустить precheck без активной смены.
- [x] Нельзя выпустить precheck для closed/cancelled order.
- [x] Нельзя добавить order line после active precheck.
- [x] `IssuePrecheck` блокирует заказ.
- [x] `IssuePrecheck` пишет `PrecheckIssued` в `local_event_log` и `pos_sync_outbox`.

События:

```text
PrecheckIssued
PrecheckSuperseded
```

Тесты:

- [x] success: issue first precheck version 1;
- [ ] success: issue second precheck version 2, previous becomes superseded;
- [ ] fail: duplicate version blocked by unique index;
- [x] fail: add order line after issued precheck;
- [x] rollback: outbox write failure откатывает precheck и order lock.

Acceptance criteria:

- `Order -> Precheck` работает локально;
- precheck immutable;
- active precheck только один;
- order locked после precheck.

---

## Этап D. Manager Override & Local RBAC

Цель: опасные offline-операции выполняются только менеджером.

Задачи:

- [ ] Добавить PIN hash для employees.
- [ ] Добавить permission/role flag для manager override.
- [ ] Реализовать локальную проверку PIN на Edge.
- [ ] Реализовать `manager_override_logs`.
- [x] Реализовать temporary backend foundation `CancelPrecheck` без full PIN verification.
- [ ] Реализовать full use case `CancelPrecheckWithManagerOverride`.
- [ ] Реализовать endpoint `POST /prechecks/{id}/cancel`.
- [ ] Добавить reason code.
- [x] После отмены precheck разблокировать order на app/service уровне.
- [ ] Реализовать `RefundPaymentWithManagerOverride`.
- [ ] Добавить `entry_kind = capture | refund`.
- [ ] Добавить `original_payment_id`.
- [ ] Реализовать rate limiting / lockout policy на manager PIN.
- [ ] Реализовать zero-logging policy для PIN / OTP / tokens.

События:

```text
ManagerOverrideGranted
PrecheckCancelled
OrderUnlocked
```

Правила:

- PIN не хранить в открытом виде;
- не отправлять PIN в Cloud;
- в sync event писать только факт manager override и manager_user_id;
- отмена precheck без manager override запрещена.

Тесты:

- [ ] correct manager PIN allows cancel;
- [ ] wrong PIN rejects cancel;
- [ ] employee without permission rejects cancel;
- [x] cancel precheck unlocks order;
- [x] cancel paid precheck запрещен на уровне `prechecks.paid_total` foundation;
- [ ] override log пишется в одной транзакции с cancel.

Acceptance criteria:

- manager override работает offline;
- audit trail сохраняется локально и уходит в Cloud через outbox.

---

## Этап E. Generic Tax Engine

Цель: убрать hardcoded taxes и подготовить Индонезию/PBJT без привязки логики к названию налога.

Задачи:

- [ ] Добавить/уточнить `tax_profiles`.
- [ ] Добавить связь catalog/menu item -> tax profile.
- [ ] Реализовать расчет inclusive/exclusive tax.
- [ ] Сохранять tax snapshot в precheck.
- [ ] Добавить `receipt_label`, например `PB1`, без hardcode в Go.
- [ ] Запретить расчет налогов во frontend.

Минимальная модель:

```text
tax_profiles:
  id
  name
  rate_percent
  is_inclusive
  receipt_label
  active
```

Тесты:

- [ ] exclusive tax calculation;
- [ ] inclusive tax calculation;
- [ ] zero tax profile;
- [ ] tax snapshot не меняется после изменения tax profile;
- [ ] rounding rules deterministic.

Acceptance criteria:

- precheck total считается только на Edge;
- налоговая логика не содержит hardcoded `PB1` / `PBJT`;
- tax snapshot воспроизводим.

---

## Этап F. Payments to Precheck & Final Check

Цель: реализовать полный финансовый цикл.

Новая модель:

```text
Payment capture -> Precheck
RefundPayment    -> same Precheck as negative ledger entry
Full paid Precheck -> Final Check
```

Задачи:

- [ ] Изменить payment capture с `check_id` на `precheck_id`.
- [ ] Поддержать cash payment.
- [ ] Поддержать trusted card terminal payment.
- [ ] Запретить payment без active shift.
- [ ] Запретить payment для cancelled/superseded precheck.
- [ ] Разрешить partial payments.
- [ ] Поддержать refund flow через immutable negative ledger entries.
- [ ] Запретить `CancelPrecheck` при `paid_total_minor > 0`.
- [ ] Разрешить `CancelPrecheck` только после refund до `paid_total_minor = 0`.
- [ ] Добавить reconciliation fields: `provider_reference`, `terminal_id`, `auth_code`, `operator_note`, `business_date_local`.
- [ ] При достижении полной суммы создать final `Check`.
- [ ] Пометить precheck `paid`.
- [ ] Пометить order `closed`.
- [ ] Создать событие `CheckCreated` только после полной оплаты.

События:

```text
PaymentCaptured
PrecheckPaid
CheckCreated
OrderClosed
```

Правила:

- переплата запрещена в MVP, если нет отдельной политики tips/change;
- trusted card payment создается со статусом `captured` и `is_trusted = true`;
- автономный терминал не дает raw PSP payloads в MVP-0;
- check immutable.

Тесты:

- [ ] partial payment keeps precheck issued/partially_paid equivalent;
- [ ] full payment creates final check;
- [ ] overpayment rejected;
- [ ] payment for cancelled precheck rejected;
- [ ] payment for superseded precheck rejected;
- [ ] duplicate command id does not double-capture;
- [ ] outbox failure rolls back payment and check generation.

Acceptance criteria:

- работает цикл `Order -> Precheck -> Payment -> Check`;
- check больше не создается до оплаты;
- Cloud получает полный журнал через outbox.

---

## Этап G. Sync Worker MVP

Цель: превратить outbox foundation в рабочую доставку событий.

Задачи:

- [ ] Добавить `sync-worker` внутри `pos-backend` или отдельный command.
- [ ] Реализовать batch claiming pending events.
- [ ] Реализовать exact constants: `base_delay_ms = 1000`, `max_delay_ms = 300000`, `lease_ttl_seconds = 120`, `attempts > 20 => suspended`.
- [ ] Реализовать exact formula: `delay_ms = min(base_delay_ms * 2^attempts, max_delay_ms) + random(0, 1000)`.
- [ ] Реализовать `sequence_no` ordering.
- [ ] Реализовать stale `processing` reclaim.
- [ ] Реализовать item-level ACK parsing.
- [ ] Разделить retryable и terminal failure semantics.
- [ ] Реализовать suspended status после `attempts > 20`.
- [ ] Реализовать Cloud sender на `POST /api/v1/sync/edge-events`.
- [ ] Реализовать стабильную обработку duplicate ack.
- [ ] Добавить manager action `Retry Failed Syncs`.
- [ ] Добавить operational endpoint для статуса очереди.

Минимальные endpoints:

```text
GET  /api/v1/sync/status
POST /api/v1/sync/retry-failed
```

Acceptance criteria:

- 8 часов offline накапливают события без потерь;
- после восстановления сети события уходят batch-ами;
- Cloud dedupe не создает дублей;
- poison message не блокирует очередь.

---

## Этап H. Device Provisioning

Цель: стабилизировать device identity перед первым запуском.

Задачи:

- [ ] Binding code длиной 8 цифр.
- [ ] TTL 10 минут.
- [ ] Single-use.
- [ ] Максимум 5 неудачных попыток.
- [ ] Resend инвалидирует старый код.
- [ ] Plaintext code не хранится; используется keyed verifier format.
- [ ] Реализовать OTP для привязки устройства.
- [ ] Реализовать endpoint создания binding code.
- [ ] Реализовать endpoint consuming binding code.
- [ ] Возвращать stable `device_id`.
- [ ] Сохранять device metadata.
- [ ] Обновить POS UI flow первого запуска.
- [ ] Запретить write API без stable `device_id`, кроме bootstrap/provisioning endpoints.
- [ ] Реализовать lifecycle `pending -> active -> revoked -> replaced`.
- [ ] Реализовать rebind flow после reinstall / restore / clone.
- [ ] Android storage = keystore-backed.
- [ ] Windows storage = DPAPI-protected.
- [ ] Запретить roaming stores для production `device_id`.
- [ ] Provisioning и sync только по TLS 1.2/1.3.

Возможные endpoints:

```text
POST /api/v1/devices/binding-codes
POST /api/v1/devices/bind
GET  /api/v1/devices/current
```

Acceptance criteria:

- устройство получает стабильный UUID;
- все write-команды используют этот UUID;
- idempotency key стабилен.

---

## Этап I. POS UI MVP

Цель: дать кассиру минимальный интерфейс для первого пилота.

Стек:

```text
Vite workspace
/ui-core
/ui-protocol
/apps/pos-react
```

Правила:

- UI не считает бизнес-итоги;
- UI вызывает Edge API;
- UI показывает состояния, полученные от backend;
- все пользовательские тексты на русском языке;
- бизнес-термины в UI: заказ, пречек, оплата, чек, смена, кассовая сессия.

MVP screens:

- [ ] Первый запуск / привязка устройства.
- [ ] Открытие смены.
- [ ] Открытие cash session.
- [ ] Создание заказа.
- [ ] Добавление позиций.
- [ ] Выпуск пречека.
- [ ] Отмена пречека через PIN менеджера.
- [ ] Оплата наличными.
- [ ] Оплата картой через автономный терминал.
- [ ] Финальный чек.
- [ ] Sync status.
- [ ] Retry failed syncs.
- [ ] Экран refund payment через manager PIN.
- [ ] Экран print error state.
- [ ] Действие `Reprint Precheck`.
- [ ] Действие `Reprint Check`.

Acceptance criteria:

- кассир может провести гостя без Swagger/curl;
- ошибка сети не блокирует продажи;
- dangerous actions требуют PIN.

---

## Этап J. DishServed MVP без полноценного KDS

Цель: подготовить inventory events без разработки полного KDS.

Задачи:

- [ ] Добавить `order_line_status`: ordered / served / cancelled.
- [ ] Реализовать `MarkOrderLineServed`.
- [ ] Генерировать `DishServed`.
- [ ] Сделать idempotency по order line.
- [ ] При финальном Check автоматически генерировать `DishServed` для не-served позиций, если KDS отключен.
- [ ] Не генерировать duplicate DishServed.

События:

```text
DishServed
OrderLineServed
```

Acceptance criteria:

- склад не списывается при добавлении позиции;
- served event создается только в правильный момент;
- automatic served at final check не дублирует ручной served.

---

## Этап K. Cloud Inventory Ledger Foundation

Цель: минимальная immutable inventory модель post-sale.

Не является блокером для первого кассового MVP, но нужна для RMS MVP.

Задачи:

- [ ] Добавить Cloud tables `stock_documents`.
- [ ] Добавить Cloud tables `stock_moves`.
- [ ] Добавить document type `recipe_consumption`.
- [ ] Обрабатывать `DishServed` из raw event storage.
- [ ] Создавать stock document по активной recipe version.
- [ ] Реализовать manual receipt.
- [ ] Реализовать reversal/correction skeleton.

Acceptance criteria:

- inventory меняется только документами;
- нет прямого `UPDATE quantity = quantity - X`;
- `DishServed` приводит к `recipe_consumption`.

---

## Этап L. Backup & First Launch Recovery

Цель: не выходить на пилот без сценария восстановления.

Задачи:

- [ ] Snapshot только через `VACUUM INTO`.
- [ ] Snapshot bundle: `snapshot.db`, `metadata.json`, `sha256`.
- [ ] Валидация checksum перед upload.
- [ ] Smoke test replay pending outbox after restore.
- [ ] Реализовать SQLite snapshot на Edge.
- [ ] Реализовать upload snapshot в Cloud.
- [ ] Реализовать локальную проверку snapshot integrity.
- [ ] Документировать manual recovery procedure.
- [ ] Добавить smoke test восстановления на новой машине.

Acceptance criteria:

- можно восстановить новый Edge из последнего snapshot;
- неотправленные данные можно восстановить ручным переносом SQLite DB;
- процедура восстановления описана на русском языке.

---

## Этап M. Reconciliation MVP

Цель: закрыть trusted terminal payments через банковскую сверку.

Задачи:

- [ ] CSV import банковской выписки в Cloud.
- [ ] Matching по amount/date/reference/operator notes.
- [ ] Статусы:
  - `matched`;
  - `amount_mismatch`;
  - `missing_in_pos`;
  - `missing_in_bank`.
- [ ] UI/endpoint для ручного override.

Acceptance criteria:

- trusted card payments MVP-0 можно сверять после смены;
- расхождения видны менеджеру.

---

## Этап N. First Launch Readiness

Цель: отдельная точка готовности к первому пилотному запуску.

Этот этап обязателен и не должен смешиваться с разработкой новых фич.

Checklist backend:

- [ ] `go test ./...` проходит в `pos-backend`.
- [ ] `go test ./...` проходит в `cloud-backend`.
- [ ] SQLite стартует с чистой БД.
- [ ] PostgreSQL Cloud стартует с чистой БД.
- [ ] Все migrations применяются с нуля.
- [ ] Нет обязательной миграции production данных.
- [ ] Все write use cases пишут local event и outbox в одной транзакции.
- [ ] Outbox retry/backoff работает.
- [ ] Cloud dedupe работает.
- [ ] Device identity стабилен.
- [ ] Manager PIN работает offline.
- [ ] SQLite runtime gate passes.
- [ ] Runtime uses patched WAL-safe SQLite baseline.
- [ ] Refund flow works.
- [ ] Print failure does not break sale commit.
- [ ] Reprint works.

Checklist business flow:

- [ ] Открыть смену.
- [ ] Открыть cash session.
- [ ] Создать заказ.
- [ ] Добавить позиции.
- [ ] Выпустить precheck.
- [ ] Проверить, что заказ locked.
- [ ] Отменить precheck через manager PIN.
- [ ] Выпустить новый precheck version 2.
- [ ] Принять cash payment.
- [ ] Принять trusted card payment.
- [ ] Создать final check после полной оплаты.
- [ ] Закрыть заказ.
- [ ] Закрыть cash session.
- [ ] Закрыть смену.

Checklist offline/sync:

- [ ] Выполнить полный цикл без интернета.
- [ ] Накопить outbox события.
- [ ] Включить интернет.
- [ ] Убедиться, что Cloud получил события без дублей.
- [ ] Повторить replay тех же событий и получить stable ack.
- [ ] Проверить suspended event flow.
- [ ] Проверить manual retry failed syncs.
- [ ] Stale `processing` reclaim works.
- [ ] `duplicate` ACK becomes `sent`.
- [ ] `terminal_error` becomes `failed`.

Checklist recovery:

- [ ] Создать SQLite snapshot.
- [ ] Загрузить snapshot в Cloud.
- [ ] Восстановить Edge из snapshot.
- [ ] Проверить, что текущий ресторанный flow продолжает работать.
- [ ] Replay pending outbox after restore.
- [ ] Verify Cloud dedupe stability.

Результат этапа:

```text
Проект готов к первому техническому запуску в одном ресторане.
```

---

## 9. Post-MVP этапы

Эти задачи не должны блокировать MVP-0.

### 9.1 Полный KDS

- KDS screen;
- kitchen routing;
- WebSocket updates;
- cook station states;
- preparation timing.

### 9.2 Android / Hardware Layer

- Android WebView kiosk;
- Go process supervisor;
- printer bridge;
- power management;
- boot recovery.

### 9.3 Real PSP Integrations

- Adyen;
- Stripe;
- webhook ingestion;
- Payment Evidence Archive;
- PCI filtering.

### 9.4 ClickHouse Analytics

- food cost;
- AVCO;
- dashboards;
- BI reports.

### 9.5 Multi-restaurant / advanced Cloud projections

- richer Cloud projections;
- cross-restaurant reporting;
- advanced catalog governance.

---

## 10. Запрещенные решения до MVP

Запрещено:

- делать multi-master Edge;
- создавать `Check` до полной оплаты;
- редактировать `Precheck`;
- менять `Order` после active precheck без manager override;
- списывать склад по `OrderLineAdded`;
- считать налоги во frontend;
- hardcode `PB1` / `PBJT` в Go logic;
- хранить PIN в открытом виде;
- генерировать unstable `device_id`;
- делать Cloud обязательным для кассовых операций;
- строить PSP evidence archive до реальных PSP интеграций;
- внедрять ClickHouse до стабильного OLTP flow;
- проектировать миграцию production БД до первого запуска.

---

## 11. Правила для AI-генерации кода

Каждая новая итерация должна начинаться с чтения:

```text
AGENTS.md
SPECv1.3.md
ROADMAP_MVP.md
README.md
```

Правила ответа и кода:

- Все объяснения, планы, roadmap, комментарии к задачам и промпты писать на русском языке.
- Имена Go entities, SQL tables, JSON fields, enum values и endpoints писать на английском языке.
- Не предлагать миграцию production БД до первого запуска.
- Не возвращать старую модель `CreateCheck` как создание счета гостя.
- Любой write use case должен содержать local event + outbox в той же транзакции.
- После изменения capabilities обновлять `AGENTS.md` и `README.md`.
- Для Go использовать текущую версию проекта и существующие архитектурные слои.
- Не добавлять бизнес-логику в HTTP handlers.
- Не добавлять бизнес-логику во frontend.

---

## 12. Рекомендуемый порядок ближайших итераций

1. Обновить документацию: `SPECv1.3.md`, `ROADMAP_MVP.md`, `AGENTS.md`, `README.md`.
2. Привести SQLite schema к v1.3 для первого запуска.
3. Реализовать Precheck domain/repository/use case/API.
4. Реализовать Manager Override и отмену precheck.
5. Реализовать Generic Tax Engine.
6. Перевести payments с `check_id` на `precheck_id`.
7. Реализовать automatic final check generation.
8. Реализовать sync-worker retry/backoff/suspended.
9. Реализовать device provisioning flow.
10. Собрать минимальный POS UI.
11. Реализовать DishServed MVP без полного KDS.
12. Провести First Launch Readiness.

---

## 13. Определение готовности MVP-0

MVP-0 считается готовым, когда выполнен сценарий:

```text
1. Чистая БД создается с нуля.
2. Менеджер привязывает устройство.
3. Кассир открывает смену.
4. Кассир открывает cash session.
5. Кассир создает заказ.
6. Кассир добавляет позиции.
7. Кассир выпускает precheck.
8. Система блокирует order.
9. Менеджер может отменить precheck через PIN.
10. Кассир выпускает новый precheck.
11. Кассир принимает оплату наличными или trusted card.
12. Система автоматически создает final check после полной оплаты.
13. Система закрывает order.
14. Все события сохраняются в local_event_log и pos_sync_outbox.
15. При восстановлении сети Cloud получает события без дублей.
16. SQLite snapshot можно создать и восстановить.
```

Главный итог:

```text
Первый ресторан может провести реальный кассовый день на одном Primary Edge Node,
даже если интернет пропадает на несколько часов.
```

---

# Codex Usage Note

Эти файлы являются актуальными pilot-freeze источниками для формирования промптов Codex. Перед генерацией кода агент должен использовать их вместе с `AGENTS.md`, `README.md`, текущей структурой репозитория и существующими тестами. Запрещено возвращаться к старой модели `Order -> Check -> Payment`, хранить деньги не как minor units, делать live backup через прямое копирование active WAL DB, хранить PIN/OTP в plaintext или считать печать частью финансового commit.
