# Спецификация системы шаблонов и нефискальной печати

Статус: принято. Реализовано сейчас: P1-2 IR/parser, P1-3 ESC/POS renderer/printer
primitives, P1-6 print context projection, P1-7 template engine/default templates,
P1-8 Edge print queue/routing/worker и POS-84 Cloud-owned stream `printers` без
секций, точек продаж и per-printer targets. Clean-stack аудит 2026-06-28 исправил
Cloud baseline/exchange blocker для `printers`; Cloud runtime version `0.1.15`.

Дата: 2026-06-24.

Архитектурное решение: `docs/adr/ADR-017-receipt-templating-and-printing.md`.

Этот документ является source of truth для:
- схемы `receipt_template` (Cloud PostgreSQL и Edge SQLite);
- контракта ingest-потока `receipt_templates`;
- JSON-схемы print context для каждого типа документа;
- print queue schema и HTTP API;
- Cloud-owned printer routing config на Edge.

Если документ конфликтует с кодом и тестами, сначала фиксируется фактическое поведение по
коду, затем обновляется документация.

## Фискальная граница

Print subsystem владеет только нефискальными документами. Фискальное тело чека (54-ФЗ)
формирует фискальный регистратор вне этой системы.

Допустимые значения `document_type`:

| Тип                | Описание                                              |
|--------------------|-------------------------------------------------------|
| `precheck`         | Предчек (нефискальный); immutable precheck snapshot   |
| `check_nonfiscal`  | Нефискальный чек (необязательный pre/post-fiscal slip)|
| `ticket`           | QR-билет (service с `qr_confirmation_enabled`)        |
| `kitchen_service`  | Сервисный кухонный чек (заказ на кухню)               |
| `cash_in_out`      | Внесение / изъятие денег                              |
| `acceptance`       | Документ приёмки товара                               |

Фискальные значения (`fiscal_check`, `fiscal_correction` и т.д.) в enum запрещены.

## `receipt_template` — Cloud master-data

### Cloud PostgreSQL schema

Реализовано сейчас (POS-71). Таблица в managed baseline `cloud-backend/migrations/postgres/001_init.sql`
называется `cloud_receipt_templates` (по cloud master-data конвенции `cloud_*`); форма и индексы
соответствуют схеме ниже. `org_id` пустой (`''`) для текущего single-tenant cloud:

```sql
CREATE TABLE cloud_receipt_templates (
    id              TEXT NOT NULL PRIMARY KEY,     -- UUIDv7
    org_id          TEXT NOT NULL,
    restaurant_id   TEXT,                          -- NULL = tenant-level default
    document_type   TEXT NOT NULL CHECK (document_type IN (
                        'precheck','check_nonfiscal','ticket',
                        'kitchen_service','cash_in_out','acceptance')),
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL,                 -- ReceiptLine Level 1 markup
    level           INTEGER NOT NULL DEFAULT 1,    -- 1=ReceiptLine, 2=go text/template
    cpl             INTEGER NOT NULL CHECK (cpl IN (32, 40, 48, 58)),
    printer_class   TEXT NOT NULL DEFAULT 'generic',
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    version         INTEGER NOT NULL DEFAULT 1,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX cloud_receipt_templates_default_uq
    ON cloud_receipt_templates (org_id, COALESCE(restaurant_id,''), document_type)
    WHERE is_default = TRUE AND is_active = TRUE;

CREATE INDEX cloud_receipt_templates_org_type
    ON cloud_receipt_templates (org_id, document_type, is_active);
```

### Edge SQLite schema

Реализовано сейчас (POS-71). Хранит только `is_active = TRUE` строки, добавлено в
`pos-backend/migrations/sqlite/001_init.sql`:

```sql
CREATE TABLE IF NOT EXISTS receipt_templates (
    id              TEXT NOT NULL PRIMARY KEY,
    restaurant_id   TEXT,
    document_type   TEXT NOT NULL,
    name            TEXT NOT NULL,
    content         TEXT NOT NULL,
    level           INTEGER NOT NULL DEFAULT 1,
    cpl             INTEGER NOT NULL,
    printer_class   TEXT NOT NULL DEFAULT 'generic',
    is_default      INTEGER NOT NULL DEFAULT 0,
    version         INTEGER NOT NULL DEFAULT 1,
    synced_at       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS receipt_templates_type_default
    ON receipt_templates (document_type, is_default);
```

### Cloud API (CRUD)

Реализовано сейчас (POS-71):

```
GET    /api/v1/receipt-templates                    — список шаблонов (filter: org_id, restaurant_id, document_type, is_default, is_active)
POST   /api/v1/receipt-templates                    — создать шаблон
GET    /api/v1/receipt-templates/{id}               — получить шаблон
PUT    /api/v1/receipt-templates/{id}               — обновить (инкрементирует version)
DELETE /api/v1/receipt-templates/{id}               — soft-delete (is_active = FALSE)
GET    /api/v1/receipts/preview                     — SVG-предпросмотр (тело в JSON-body) — legacy
POST   /api/v1/receipts/preview                     — SVG-предпросмотр (preferred; POS-80 Cloud UI editor)
```

RBAC: `cloud.templates.manage` (organization.manage или поддержка); permission id зафиксирован
как `domain.PermissionReceiptTemplatesManage` и зарегистрирован в каталоге прав. Per-request RBAC
middleware для cloud master-data routes ещё не подключён (как и для sibling routes restaurants/halls/...),
поэтому авторитетная проверка выполняется на cloud admin boundary. Preview доступен всем
аутентифицированным пользователям Cloud. Error contract — стабильный safe code/message через `httpx`.

### Ingest stream `receipt_templates`

Реализовано сейчас (POS-71). Stream в `mastersync.Service`:

- Cloud собирает package: все `is_active = TRUE` шаблоны для `restaurant_id` ресторана
  плюс tenant-level defaults (`restaurant_id IS NULL`). Restaurant-specific шаблон
  перекрывает tenant-level default для того же `document_type`.
- Stream-specific checkpoint token (`receipt-templates:<restaurant>:<MAX(updated_at) ms>:<count>`)
  фиксирует версию активного набора; `cloud_version` пакета — монотонная версия публикации.
- Edge apply: заменяет все строки `receipt_templates` для этого ресторана атомарно (в master-data apply tx).
- Если entitlement ресторана отозван или stream `receipt_templates` отключён
  в delivery config — stream не включается в exchange response.
- Default templates POS-73 (`precheck`, `ticket`) сидируются через `scripts/seed-dev-system.py`
  HTTP-only: CRUD `/api/v1/receipt-templates`, без прямой записи в БД и без publish API.
- POS-81 smoke в `scripts/seed-dev-system.py --run-minimal-flow` подтверждает, что POS Edge
  видит доставленные шаблоны через sync status. **Обновлено POS-86:** точки auto-enqueue
  изменились — `precheck`-job ставится при `IssuePrecheck` (пречек гостю до оплаты), а после
  полной оплаты ставится `check_nonfiscal`-job (источник — Check, не Precheck; фикс
  давно известного gap POS-72, где `check_nonfiscal` никогда не создавался автоматически) +
  `ticket`-jobs для выданных ticket units. Полная модель маршрутизации, per-printer FIFO и
  print-confirmation gate — `docs/backend/EDGE-PRINT-ROUTING-SPEC.md`. Физическая отправка на
  принтер остается hardware acceptance POS-64.

Запись в `directional-sync-ownership.md`:

| Данные               | Владелец | Направление     | Статус          |
|----------------------|----------|-----------------|-----------------|
| Receipt templates    | Cloud    | Cloud -> Edge   | реализовано сейчас (POS-71) |

## Print Context — JSON-схема

Print context — плоская View-model, строго проецируемая из immutable snapshot.
Не читает текущий каталог, текущее меню или текущие цены.

### PrecheckPrintContext

Используется для `document_type: precheck`.

```jsonc
{
  // Шапка
  "restaurant_name": "Выставка «TechWorld 2026»",
  "restaurant_address": "г. Москва, Проспект Мира 119с57",
  "cashier_name": "Иван Петров",
  "shift_number": 42,                    // порядковый номер кассовой смены
  "business_date": "2026-06-24",         // YYYY-MM-DD
  "opened_at": "2026-06-24T10:15:00",   // ISO 8601 без TZ (local)
  "precheck_number": "00001",            // display number

  // Строки заказа
  "lines": [
    {
      "name": "Стандартный билет",
      "quantity": 1,
      "unit_price_minor": 50000,         // копейки
      "total_minor": 50000,
      "modifiers": [
        { "name": "VIP зона", "price_minor": 10000 }
      ],
      "is_service": true,
      "qr_enabled": true
    }
  ],

  // Итоги
  "subtotal_minor": 50000,
  "discount_total_minor": 0,
  "surcharge_total_minor": 0,
  "tax_total_minor": 9091,               // НДС включён в итог (inclusive)
  "total_minor": 50000,
  "currency_code": "RUB",

  // Налоговые компоненты (массив для детализации)
  "taxes": [
    { "name": "НДС 20%", "base_minor": 41667, "amount_minor": 8333 }
  ],

  // Скидки (массив, может быть пустым)
  "discounts": [
    { "name": "Скидка 10%", "amount_minor": 5000 }
  ],

  // Метаданные
  "is_copy": false,                      // true при reprint
  "copy_marker": "",                     // "COPY" при reprint
  "printed_at": "2026-06-24T10:16:00"
}
```

### ServiceTicketPrintContext

Используется для `document_type: ticket`.

```jsonc
{
  // Идентификация билета
  "ticket_number": "019044ab-0000-7000-0000-000000000001",  // UUIDv7 = ticket_number
  "ticket_display_number": "0001",       // cash_shift_sequence (порядковый в смене)
  "qr_payload": "MHT1:019044ab-0000-7000-0000-000000000001",

  // Что продано
  "service_name": "Стандартный билет",
  "category_name": "Билеты",            // из menu category
  "price_minor": 50000,
  "currency_code": "RUB",

  // Когда продано
  "sale_date_local": "2026-06-24",      // YYYY-MM-DD
  "sale_time_local": "10:15:00",        // HH:MM:SS
  "timezone": "Europe/Moscow",

  // Срок действия
  "validity_mode": "cash_session",      // cash_session|business_date|absolute_date
  "validity_date_local": null,          // YYYY-MM-DD или null для cash_session

  // Ресторан / мероприятие
  "restaurant_name": "Выставка «TechWorld 2026»",
  "restaurant_address": "г. Москва, Проспект Мира 119с57",

  // Кассир
  "cashier_name": "Иван Петров",
  "shift_number": 42,
  "business_date": "2026-06-24",

  // Метаданные
  "is_copy": false,
  "copy_marker": ""
}
```

### KitchenServicePrintContext

Используется для `document_type: kitchen_service`. Запланировано до пилота (P2-1):

```jsonc
{
  "order_number": "00042",
  "table_name": "Стол 5",
  "hall_name": "Зал A",
  "waiter_name": "Мария Сидорова",
  "printed_at": "2026-06-24T10:20:00",
  "lines": [
    { "name": "Борщ", "quantity": 2, "comment": "без лука", "course": 1 }
  ]
}
```

### CashInOutPrintContext

Используется для `document_type: cash_in_out`. Запланировано до пилота (P2-1):

```jsonc
{
  "operation_type": "cash_in",           // cash_in|cash_out
  "amount_minor": 100000,
  "currency_code": "RUB",
  "cashier_name": "Иван Петров",
  "shift_number": 42,
  "business_date": "2026-06-24",
  "restaurant_name": "Выставка «TechWorld 2026»",
  "comment": "Начальная выручка",
  "printed_at": "2026-06-24T10:05:00"
}
```

### AcceptancePrintContext

Используется для `document_type: acceptance`. Запланировано до пилота (P2-1):

```jsonc
{
  "document_number": "ПРИ-0001",
  "supplier_name": "ООО Поставщик",
  "restaurant_name": "Выставка «TechWorld 2026»",
  "accepted_by": "Мария Сидорова",
  "accepted_at": "2026-06-24T09:00:00",
  "lines": [
    { "name": "Стакан пластиковый", "quantity": 1000, "unit": "шт", "unit_price_minor": 150, "total_minor": 150000 }
  ],
  "total_minor": 150000,
  "currency_code": "RUB"
}
```

## Template engine

Реализовано сейчас (POS-73): `shared/platform/receipt/engine.Render(templateContent, printContext)`
детерминированно рендерит ReceiptLine Level 1 template в IR:

- сначала использует POS-68 parser для получения width-independent IR;
- раскрывает `{if:<field>}` по truthy-значению поля print context;
- раскрывает `{each:<field>}` для массивов/slices с вложенным scope элемента;
- подставляет `{{.field}}` по JSON field name или Go field name;
- поддерживает pipe `{{.amount_minor | money}}`, который форматирует minor units через
  `FormatMoneyMinor` и берёт `currency_code` из текущего/root scope; для `RUB`
  используется CP866-safe суффикс `RUB`, а не символ `₽`;
- возвращает IR без `IfBlock`/`EachBlock`, готовый для существующих SVG/ESC/POS renderer-ов.

Ограничения текущего POS-73/POS-74:

- поддерживается только Level 1 ReceiptLine; Level 2 (`go text/template`) вне текущего объема;
- engine не читает БД, каталог, текущее время, printer config, fiscal state или очередь печати;
- реализованы `document_type` `precheck`, `check_nonfiscal` и `ticket` для Edge print worker;
- Cloud UI editor, fiscal logic, PNG/e-receipt и Level 2 templates вне текущего объема.

## SVG Preview Endpoint

Реализовано сейчас для Level 1 preview:

```
GET /api/v1/receipts/preview
Content-Type: application/json
```

Запрос:

```jsonc
{
  "template_content": "---\n{{.restaurant_name}}\n...",
  "document_type": "precheck",
  "cpl": 48,
  "print_context": { /* PrecheckPrintContext */ }
}
```

Ответ: `Content-Type: image/svg+xml`, `200 OK`. SVG-документ с rendered document.

Ошибки:
- `400 TEMPLATE_PARSE_ERROR` — невалидный ReceiptLine синтаксис.
- `400 CONTEXT_SCHEMA_ERROR` — `document_type`, `cpl` или `print_context` не соответствуют текущему контракту preview.
- Preview доступен всем аутентифицированным пользователям Cloud и сейчас не требует module license gate.

Ограничения текущего preview:
- endpoint валидирует `print_context` как JSON object и допустимый `document_type`, затем
  использует `engine.Render` для Level 1 template interpolation;
- endpoint не выполняет POS-72 context projection: caller передаёт уже готовый print context;
- поддерживается `cpl` 32 или 48;
- QR preview учитывает `QRBlock.Size` из синтаксиса `{qr:size=1..8:<payload>}`.

## Print Queue HTTP API

Реализовано сейчас (POS-74):

```
GET  /api/v1/print/jobs/{id}        — статус задачи
POST /api/v1/print/jobs/{id}/retry  — ручной retry failed/pending задачи
GET  /api/v1/print/jobs             — список задач (filter: status, document_type, limit)
```

RBAC: `pos.print.status` (read), `pos.print.retry` (mutate).

Ответ `GET /api/v1/print/jobs/{id}`:

```jsonc
{
  "id": "019...",
  "document_type": "ticket",
  "source_kind": "ticket",
  "source_id": "019...",
  "status": "succeeded",             // pending|processing|succeeded|failed
  "attempts": 1,
  "max_attempts": 3,
  "printer_class": "ticket",
  "last_error": null,
  "next_attempt_at": null,
  "created_at": "2026-06-24T10:15:01",
  "printed_at": "2026-06-24T10:15:02"
}
```

### Edge SQLite `print_jobs`

Реализовано сейчас (POS-74) в managed baseline `pos-backend/migrations/sqlite/001_init.sql`.
Очередь не хранит print payload: worker заново читает immutable snapshot источника
(`prechecks`, `checks`, `ticket_units`) и default template из Edge `receipt_templates`.

```sql
CREATE TABLE IF NOT EXISTS print_jobs (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  document_type TEXT NOT NULL CHECK (document_type IN ('precheck','check_nonfiscal','ticket')),
  source_kind TEXT NOT NULL CHECK (source_kind IN ('precheck','check','ticket')),
  source_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','processing','succeeded','failed')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  max_attempts INTEGER NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
  printer_class TEXT NOT NULL DEFAULT 'generic',
  last_error TEXT,
  next_attempt_at TEXT,
  locked_by TEXT,
  locked_at TEXT,
  printed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(document_type, source_id)
);
```

Индексы: `print_jobs_pending_due` для worker claim и
`print_jobs_restaurant_status_created` для status/list endpoints. Таблица входит в
startup schema verification до запуска HTTP server/worker.

Retry policy:
- До 3 total attempts.
- После неудачной попытки worker ставит backoff `2s`, затем `5s`; policy sequence
  хранит `2s/5s/15s`, но при `max_attempts=3` статус `failed` выставляется после
  третьей неудачной отправки.
- Manual retry переводит `failed` или `pending` в `pending`, сбрасывает attempts/error/lock.
- Print retry НЕ повторяет payment, НЕ создаёт ticket и НЕ изменяет финансовый state.

Worker pipeline:

```
print_jobs pending -> claim processing
  -> lookup default receipt_templates(document_type)
  -> POS-72 projection из immutable snapshot
  -> engine.Render(template.content, printContext)
  -> escpos.Render(IR, printer.RenderOptions())
  -> escpos.WriteRaw / injected sender
  -> succeeded или pending/failed
```

### Edge SQLite `print_job_targets`

Реализовано сейчас (POS-85): `print_jobs` остается родительской задачей печати
документа, а storage для каждой фактической отправки на физический принтер выделен
в отдельную строку `print_job_targets`.

Поля target:

- `id`, `print_job_id`, `restaurant_id`;
- `printer_id`;
- `scope_type` (`restaurant`, `sales_point`, `section`);
- `scope_id`;
- `status` (`pending`, `processing`, `succeeded`, `failed`);
- `attempts`, `max_attempts`, `last_error`, `next_attempt_at`, `locked_by`,
  `locked_at`, `printed_at`, `created_at`, `updated_at`.

Запланировано далее (POS-86): worker создает targets по `print_routes`, retry
выполняется на уровне target, родительский `print_jobs.status` становится
`succeeded`, когда все обязательные targets успешны, и `failed`, когда хотя бы
один обязательный target исчерпал retry policy. Ручной retry должен уметь
перезапустить один target без повторного payment, ticket issuance или изменения
финансового state.

## Printer Config

Реализовано сейчас (POS-84): конфигурация принтеров является Cloud-owned master data.
Cloud stream `printers` доставляет все активные принтеры ресторана, а POS Edge
атомарно заменяет строки `receipt_printers` для `restaurant_id` внутри master-data apply tx.
Deployment/runtime config больше не задает routing; если legacy routing key задан
при старте, POS Edge пишет warning `LEGACY_PRINTER_ROUTING_IGNORED` и игнорирует значение.

### Edge SQLite `receipt_printers`

```sql
CREATE TABLE IF NOT EXISTS receipt_printers (
  id TEXT NOT NULL PRIMARY KEY,
  restaurant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('tcp','usb')),
  address TEXT,
  port INTEGER,
  document_types TEXT NOT NULL CHECK (json_valid(document_types)),
  codepage TEXT NOT NULL DEFAULT '' CHECK (codepage IN ('','cp437','cp866')),
  paper_cut_type TEXT NOT NULL DEFAULT 'partial' CHECK (paper_cut_type IN ('partial','full')),
  cpl INTEGER NOT NULL CHECK (cpl IN (32,42,48,56,80)),
  is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
  cloud_version INTEGER NOT NULL DEFAULT 0 CHECK (cloud_version >= 0),
  synced_at TEXT NOT NULL
);
```

Индекс: `receipt_printers_restaurant_active`. Таблица входит в startup schema
verification до запуска HTTP server/worker.

### Ingest stream `printers`

- Cloud package содержит `printers[]`: `id`, `restaurant_id`, `name`, `type`, `address`,
  `port`, `document_types`, `codepage`, `paper_cut_type`, `cpl`, `version`.
- Cloud baseline `cloud_master_data_packages`, Cloud exchange validation и master-data
  payload validation принимают stream `printers`; это подтверждено clean-stack smoke
  2026-06-28 после повышения Cloud runtime до `0.1.15`.
- Edge сохраняет `version` как `cloud_version`; stream-specific sync state хранит
  `checkpoint_token` вида `printers:{restaurant_id}:{MAX(updated_at).UnixMilli()}:{count}`.
- Soft-delete в Cloud убирает принтер из следующего active package; Edge replace удаляет
  отсутствующие строки ресторана.
- Один `document_type` может быть указан у нескольких активных принтеров; worker отправляет
  один print job на каждый подходящий принтер.
- Если для `restaurant_id + document_type` нет активных строк, worker не panic-ует и переводит
  print job по обычной retry policy в `failed` с безопасным кодом `PRINT_ROUTING_NOT_CONFIGURED`.

Runtime config keys:

- `POS_PRINT_WORKER_ENABLED` — включает in-process worker (`false` по умолчанию);
- `POS_PRINT_WORKER_ID`;
- `POS_PRINT_WORKER_POLL_INTERVAL`;
- `POS_PRINT_WORKER_SEND_TIMEOUT`.

Поддержано сейчас: TCP raw printer (`type=tcp`, `address`, `port`, обычно 9100) и
Windows USB/device path (`type=usb`, `address=...`) через существующие
`escpos.Open`/`escpos.WriteRaw` primitives (`os.OpenFile` по `address`).

Проверено на реальном физическом принтере (Xprinter XP-365B, USB, Windows 11,
2026-07-01): путь `\\.\USB001`, который раньше был указан здесь как рабочий формат
`address`, **не открывается** на современных Windows (usbprint.sys больше не создает
такой legacy compatibility symlink; `CreateFile`/`os.OpenFile` возвращают
"file not found" даже при установленной "Generic / Text Only" очереди на этот порт).
Реально открываемый Win32-путь для USB ESC/POS-устройства на Windows 10/11 — это
SetupAPI device interface path вида
`\\?\USB#VID_xxxx&PID_yyyy#<serial>#{28d78fad-5a12-11d1-ae5b-0000f803a8c2}`
(GUID — `GUID_DEVINTERFACE_USBPRINT`). Его можно получить на целевой машине через
`Get-PnpDevice -Class USBPrint` (registry `HKLM\SYSTEM\CurrentControlSet\Control\
DeviceClasses\{28d78fad-...}`, значение ключа с префиксом `##?#`, где `#` заменяется
на `\`). Именно этот путь нужно вводить в поле `address` при настройке `type=usb`
принтера, а не `USB001`/`\\.\USB001`. С этим путем сквозной прогон
routing → template engine → ESC/POS renderer → print worker подтвержден реальной
печатью документа `check_nonfiscal` через существующий Go-код `escpos.WriteRaw` без
изменений runtime.

Запланировано далее (ADR-018, `EDGE-HARDWARE-ADAPTER-PROTOCOL.md`): ручной подбор этого
device path оператором не масштабируется и не подходит для Cloud-first UX. Обнаружение
USB-устройств и сетевых ESC/POS-принтеров (скан портов 9100/6001/вручную указанных),
а также сама отправка байт на устройство переносятся в отдельный exe-адаптер
(`windows-printers`), работающий как child-процесс `pos-edge.exe` по protocol channel;
`escpos.Open`/`escpos.WriteRaw` остаются как fallback-путь для уже сконфигурированных
вручную TCP-принтеров и для переходного периода, пока физическая привязка через адаптер
не задана.

## Схема печати POS-85

Реализовано сейчас на уровне Edge SQLite baseline: физический принтер и назначение
печати разделены. `receipt_printers` описывает физическое устройство, а `print_routes`
связывает документ с областью применения и одним или несколькими принтерами.

Целевые области применения:

- `restaurant` — принтер отчетов ресторана;
- `sales_point` — принтер кассы конкретной точки продаж;
- `section` — сервисный принтер секции ресторана.

Типы назначений:

- принтер кассы печатает чек при оплате. Сейчас это нефискальный `check_nonfiscal`;
  после реализации фискального адаптера та же точка продаж должна указывать
  фискальный route/device без генерации фискального тела через ReceiptLine;
- сервисный принтер секции-зала печатает `precheck`;
- сервисный принтер секции-цеха печатает `kitchen_service`;
- принтер отчетов ресторана печатает документы отчетов после появления report
  document type и typed report context.

`sales_points` является restaurant-scoped справочником с обязательными `name` и
`analytics_tag`. `cash_sessions.sales_point_id` добавлен как nullable подготовительная
связь; обязательное открытие кассовой смены в рамках точки продаж включается в
сервисном слое позже. Device identity остается технической привязкой и не заменяет
точку продаж.

`restaurant_section` является restaurant-scoped справочником с режимом
`hall_section` или `kitchen_workshop`. Секция-зал связывается с `hall_id` и печатает
предчеки по этому залу. Секция-цех связывается с кухонной маршрутизацией и печатает
кухонные заказы. В print context должен передаваться `section_id`.

Edge settings запланированы далее: они должны позволять добавлять физические
принтеры, назначать их на точку продаж или секцию, просматривать очередь и
повторять отдельный target. Edge-side изменение схемы печати должно применяться
сразу локально, записываться в `printer_route_override_audit` и отправляться в
Cloud как Edge-originated override. Это не proposal flow.

Реализовано сейчас в P1-3:

- `shared/platform/receipt/escpos` рендерит IR в ESC/POS bytes с инициализацией
  принтера. **По умолчанию (POS-80): CP437 (`ESC t 0`) — PC437 USA/Standard Europe**,
  подходит для Английского/Индонезийского текста и большинства non-Russian ESC/POS принтеров.
  Для кириллицы нужен явный `"codepage": "cp866"` в конфиге принтера.
- CP437 encoder поддерживает ASCII 0x20-0x7E + типографические нормализации;
  CP866 encoder поддерживает ASCII + русскую кириллицу, `Ё/ё`, `№`.
  `PrinterConfig.Codepage` (`""` / `"cp437"` = CP437, `"cp866"` = CP866).
  generic ESC/POS path нормализует печатно-безопасную типографику в ASCII
  (`«»`/smart quotes, dash variants, `…`, `₽` → `RUB`, non-breaking space), чтобы
  default templates не падали до отправки на принтер;
- текстовые блоки, правила, пробелы, QR, barcode, 1-bit ImageBlock (`GS v 0`),
  cut, drawer, IfBlock и EachBlock покрыты unit/fixture tests;
- `QRBlock.Size` уже учитывается ESC/POS renderer-ом; P1-7 должен открыть это через
  ReceiptLine/default templates, чтобы размер QR можно было менять без правки Go-кода;
- raw write primitives поддерживают TCP (`host:port`, обычно 9100) и USB/device path
  (`\\.\USB001` на Windows) без очереди, retry orchestration и routing.

Реализовано сейчас в POS-74/POS-84: print queue, document_type routing из
Cloud-owned `receipt_printers`, backend routes, schema, retry policy и in-process worker.
Реализовано сейчас в POS-80: Cloud UI редактор шаблонов (route `/receipt-templates`
в Cloud Manager) с двухпанельным интерфейсом — список шаблонов с фильтром по типу
документа и редактор с live SVG preview через POST /api/v1/receipts/preview (debounced).
Вне текущего объема: полноценный raster fallback для текста вне CP437/CP866
или `RasterOnly:true`, fiscal logic, PNG/e-receipt и Level 2 templates.

## ReceiptLine Level 1 — эталонные шаблоны

### Default precheck template (CPL=48)

Шаблон переведён на English (Indonesia launch). Файл: `shared/platform/receipt/engine/templates/default_precheck.rl`.

```
---
{a:center}{{.restaurant_name}}
{a:center}{{.restaurant_address}}
---
{a:center}RECEIPT
{a:center}No. {{.precheck_number}}
---
{w:auto,6,12}{a:left,right,right}Item	Qty	Total
---
{each:lines}
{w:auto,6,12}{a:left,right,right}{{.name}}	{{.quantity}}	{{.total_minor | money}}
{if:modifiers}{each:modifiers}
{a:left}+ {{.name}}: {{.price_minor | money}}
{/each}{/if}
{/each}
---
{w:auto,16}{a:left,right}Subtotal:	{{.subtotal_minor | money}}
{if:discount_total_minor}
{w:auto,16}{a:left,right}Discount:	-{{.discount_total_minor | money}}
{/if}
{if:taxes}{each:taxes}
{w:auto,16}{a:left,right}{{.name}}:	{{.amount_minor | money}}
{/each}{/if}
---
{w:auto,16}{a:left,right}TOTAL:	{{.total_minor | money}}
{a:center}{{.cashier_name}}
{a:center}{{.business_date}} | Shift No.{{.shift_number}}
{if:is_copy}{a:center}*** COPY ***{/if}
{s:4}
{cut}
```

### Default ticket template (CPL=48)

Шаблон переведён на English (Indonesia launch). Файл: `shared/platform/receipt/engine/templates/default_ticket.rl`.

```
{a:center}{{.restaurant_name}}
{a:center}{{.restaurant_address}}
---
{a:center}{f:double}TICKET
{a:center}No. {{.ticket_display_number}}
---
{a:center}{f:double}{{.service_name}}
{a:center}{{.category_name}}
---
{qr:size=6:{{.qr_payload}}}
---
{a:center}{{.sale_date_local}} {{.sale_time_local}}
{a:center}Amount: {{.price_minor | money}}
{if:validity_date_local}
{a:center}Valid until: {{.validity_date_local}}
{/if}
{if:is_copy}{a:center}*** COPY ***{/if}
{a:center}{{.cashier_name}} | Shift No.{{.shift_number}}
{s:4}
{cut}
```

## `shared/platform/receipt` — структура пакета

Реализовано сейчас для `ir`, `parser`, `engine`, `escpos`, `layout`, `svg` и POS Edge
print queue/worker:

```
shared/platform/receipt/
  ir/
    block.go          -- IR типы (Block interface, TextBlock, QRBlock, IfBlock, EachBlock, ...)
    block_test.go
  parser/
    parser.go         -- ReceiptLine Level 1 -> IR
    parser_test.go
    fixtures/         -- *.rl фикстуры + ожидаемый IR JSON
  escpos/
    renderer.go       -- IR -> []byte ESC/POS команды
    renderer_test.go
    cp866.go          -- CP866 encoder/decoder
    cp866_test.go
    printer.go        -- PrinterConfig, PrinterInterface (TCP + USB)
    printer_test.go
    fixtures/         -- fixture IR -> ожидаемый []byte ESC/POS
  svg/
    renderer.go       -- IR -> SVG string
    renderer_test.go
    fixtures/         -- fixture IR + CPL -> ожидаемый SVG
  layout/
    layout.go         -- shared column layout logic (используется ESC/POS и SVG)
    layout_test.go
  engine/
    engine.go         -- template engine: content + print context -> IR
    engine_test.go
    templates/        -- default_precheck.rl, default_ticket.rl (сидируются в Cloud)
    context.go        -- PrecheckPrintContext, ServiceTicketPrintContext, ...
    projector.go      -- precheck snapshot -> PrecheckPrintContext
    projector_test.go
```

## Тестовые требования

По каждому компоненту до Review:

| Компонент      | Требование                                                           |
|----------------|----------------------------------------------------------------------|
| Parser         | Table-driven: каждая Level 1 конструкция; паника на невалидном вводе запрещена |
| ESC/POS render | Fixture roundtrip parse → IR → ESC/POS; CP866 roundtrip для кириллицы; CPL 32 и 48 |
| SVG render     | Layout equality с ESC/POS рендером: одинаковые column widths и wrap points |
| Engine         | Render из fixture context + default template → проверенный IR       |
| Projector      | Fixture precheck snapshot → ожидаемый PrecheckPrintContext           |
| Print queue    | Lifecycle: enqueue → print → success; enqueue → fail 3 раза → failed; retry |
| Ingest stream  | Cloud create → package assembly → Edge apply → Edge read             |
| Preview API    | 200 SVG; 400 TEMPLATE_PARSE_ERROR; 400 CONTEXT_SCHEMA_ERROR          |

## Документы для обновления при изменении

При изменении route, payload, schema, stream или print contract обновлять:

- `docs/backend/RECEIPT-PRINT-SPEC.md` (этот файл);
- `docs/adr/ADR-017-receipt-templating-and-printing.md`;
- `docs/sync/directional-sync-ownership.md` (streams `receipt_templates`, `printers`);
- `docs/backend/CLOUD-BACKEND-SPEC.md` (routes Cloud API);
- `docs/backend/POS-BACKEND-SPEC.md` (print queue routes, print_jobs schema, printer routing);
- `docs/backend/POS-DATA-AND-MIGRATIONS.md` (Edge SQLite schema);
- `docs/CURRENT-FUNCTIONAL-STATE.md` (после завершения компонента).

## Оставшиеся риски и открытые вопросы

- Hardware стенд POS-64: физический network ESC/POS принтер, модель неизвестна,
  `10.25.1.201:9100`, `cpl=48`, `printer_class=generic`, `paper_cut_type=partial`.
  Clean-stack прогон 2026-06-28 создал Cloud printer через `/api/v1/printers`,
  доставил stream `printers` до Edge (`status=synced`, lag `0`) и после включения
  worker отправил один `precheck` и один `ticket`; оба `print_jobs` получили
  `status=succeeded`, `attempts=1`. Оператор подтвердил выход двух чеков; ticket
  печатался со съездом крупного `{f:double}` текста на 48CPL. Исправлено в
  ESC/POS/SVG renderer: строки двойной ширины используют эффективную ширину `CPL/2`,
  поэтому default ticket template может печатать крупные `TICKET` и `service_name`
  без выхода за бумагу.
- Предыдущий manual flow 2026-06-28 через `pos-ui-g`/Playwright создал заказ на
  две отдельные single-line `Service Fee` позиции (`qty=1` каждая), выпустил
  precheck и закрыл заказ cash payment. Результат backend/worker: `print_jobs`
  созданы как `precheck: 1`, `ticket: 2`; после исправления CP866-safe рендера
  и manual retry все три job получили `status=succeeded`, `attempts=1`.
- Acceptance-дефект первого прогона: jobs падали до TCP-send на `escpos render failed`,
  потому что generic CP866 renderer не мог закодировать `₽` и часть типографских
  символов из print context. Исправлено: `FormatMoneyMinor(RUB)` печатает `RUB`,
  CP866 encoder нормализует безопасную типографику; default precheck/ticket templates
  покрыты ESC/POS render tests на CPL 32 и 48.
- После hardware check P1-3 зафиксирован нюанс: рез идет слишком близко к barcode/QR.
  Default templates POS-73 уже добавляют управляемые пустые строки `{s:4}` перед `{cut}`;
  hardware acceptance на конкретной модели принтера может потребовать printer-specific cut feed в POS-74.
- Размер QR настраивается из шаблона/default template через Level 1 синтаксис
  `{qr:size=1..8:<payload>}`; значение попадает в `QRBlock.Size`, а ESC/POS и SVG renderers
  дают сопоставимый визуальный размер.
- CP866 принтер-specific: некоторые принтеры не поддерживают codepage 17 (PC866) и
  требуют codepage 16 (PC852) или другой. Нужна проверка на целевом принтере и
  документирование в printer_class.
- QR нативный ESC/POS (`GS ( k`): не все дешёвые принтеры поддерживают model 2.
  Растровый QR через `shared/platform/receipt/escpos` является fallback.
- Windows USB: `\\.\USB001` не является рабочим device path на Windows 10/11 (см. раздел
  выше про физическую проверку 2026-07-01) и дополнительно зависит от порядка подключения
  на старых Windows, где он еще существовал. В production использовать реальный SetupAPI
  device interface path (`\\?\USB#VID_...&PID_...#<serial>#{28d78fad-...}`), сконфигурированный
  в `address` printer config для конкретной физической машины.
- `COPY` marker при reprint: должен быть виден на оба типа документов (precheck, ticket).
