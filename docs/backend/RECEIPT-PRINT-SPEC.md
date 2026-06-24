# Спецификация системы шаблонов и нефискальной печати

Статус: принято. Реализовано сейчас: POS-64 в работе; конкретные компоненты помечены.

Дата: 2026-06-24.

Архитектурное решение: `docs/adr/ADR-017-receipt-templating-and-printing.md`.

Этот документ является source of truth для:
- схемы `receipt_template` (Cloud PostgreSQL и Edge SQLite);
- контракта ingest-потока `receipt_templates`;
- JSON-схемы print context для каждого типа документа;
- print queue schema и HTTP API;
- printer routing config.

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

Запланировано до пилота (реализуется в P1-5):

```sql
CREATE TABLE receipt_templates (
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

CREATE UNIQUE INDEX receipt_templates_default_uq
    ON receipt_templates (org_id, COALESCE(restaurant_id,''), document_type)
    WHERE is_default = TRUE AND is_active = TRUE;

CREATE INDEX receipt_templates_org_type
    ON receipt_templates (org_id, document_type, is_active);
```

### Edge SQLite schema

Хранит только `is_active = TRUE` строки. Добавляется в `migrations/sqlite/001_init.sql`
в рамках P1-5:

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

Запланировано до пилота (реализуется в P1-5):

```
GET    /api/v1/receipt-templates                    — список шаблонов (filter: document_type, is_default, is_active)
POST   /api/v1/receipt-templates                    — создать шаблон
GET    /api/v1/receipt-templates/{id}               — получить шаблон
PUT    /api/v1/receipt-templates/{id}               — обновить (инкрементирует version)
DELETE /api/v1/receipt-templates/{id}               — soft-delete (is_active = FALSE)
GET    /api/v1/receipts/preview                     — SVG-предпросмотр (см. ниже)
```

RBAC: `cloud.templates.manage` (organization.manage или поддержка). Preview доступен
всем аутентифицированным пользователям Cloud.

### Ingest stream `receipt_templates`

Запланировано до пилота (реализуется в P1-5):

Новый stream в `mastersync.Service`:

- Cloud собирает package: все `is_active = TRUE` шаблоны для `restaurant_id` ресторана
  плюс tenant-level defaults (`restaurant_id IS NULL`). Restaurant-specific шаблон
  перекрывает tenant-level default для того же `document_type`.
- Package версионируется по `MAX(updated_at)` активных строк.
- Edge apply: заменяет все строки `receipt_templates` для этого ресторана атомарно.
- Если entitlement ресторана отозван или stream `receipt_templates` отключён
  в delivery config — stream не включается в exchange response.

Запись в `directional-sync-ownership.md`:

| Данные               | Владелец | Направление     | Статус          |
|----------------------|----------|-----------------|-----------------|
| Receipt templates    | Cloud    | Cloud -> Edge   | запланировано до пилота |

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

## SVG Preview Endpoint

Запланировано до пилота (реализуется в P1-4):

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
- `400 CONTEXT_SCHEMA_ERROR` — print context не соответствует схеме document_type.
- `403` — недостаточно прав.

## Print Queue HTTP API

Запланировано до пилота (реализуется в P1-8):

```
GET  /api/v1/print/jobs/{id}        — статус задачи
POST /api/v1/print/jobs/{id}/retry  — ручной ретрай failed задачи
GET  /api/v1/print/jobs             — список задач (filter: status, limit)
```

RBAC: `pos.print.status` (read), `pos.print.retry` (mutate).

Ответ `GET /api/v1/print/jobs/{id}`:

```jsonc
{
  "id": "019...",
  "document_type": "ticket",
  "status": "printed",               // pending|printing|printed|failed|cancelled
  "attempts": 1,
  "last_error": null,
  "created_at": "2026-06-24T10:15:01",
  "printed_at": "2026-06-24T10:15:02"
}
```

Retry policy:
- Попытка 1: немедленно.
- Попытка 2: через 2 секунды.
- Попытка 3: через 5 секунд.
- После 3 неудач: статус `failed`; доступен ручной retry через HTTP.
- Print retry НЕ повторяет payment, НЕ создаёт ticket и НЕ изменяет финансовый state.

## Printer Config

Конфигурация принтеров задаётся в deployment config (env vars или JSON-файл):

```json
{
  "printers": [
    {
      "id": "main",
      "type": "tcp",
      "address": "192.168.1.100",
      "port": 9100,
      "cpl": 48,
      "printer_class": "generic",
      "raster_only": false,
      "paper_cut_type": "full",
      "connect_timeout_ms": 3000,
      "write_timeout_ms": 5000
    },
    {
      "id": "usb-main",
      "type": "usb",
      "address": "\\\\.\\USB001",
      "cpl": 32,
      "printer_class": "generic",
      "raster_only": false,
      "paper_cut_type": "partial",
      "write_timeout_ms": 5000
    }
  ],
  "routing": [
    { "document_type": "precheck",        "printer_id": "main" },
    { "document_type": "check_nonfiscal", "printer_id": "main" },
    { "document_type": "ticket",          "printer_id": "main" },
    { "document_type": "kitchen_service", "printer_id": "main" },
    { "document_type": "cash_in_out",     "printer_id": "main" },
    { "document_type": "acceptance",      "printer_id": "main" }
  ]
}
```

`printer_class` значения: `generic`, `epson`, `star`, `bixolon`, `custom`. Влияет на
выбор ESC/POS команд там, где производители расходятся в деталях (QR-model, cut).

## ReceiptLine Level 1 — эталонные шаблоны

### Default precheck template (CPL=48)

```
---
{a:center}{{.restaurant_name}}
{a:center}{{.restaurant_address}}
---
{a:center}ПРЕДЧЕК
{a:center}№ {{.precheck_number}}
---
{w:auto,6,8}{a:left,left,right}Наименование\tКол\tСумма
---
{each:lines}
{w:auto,6,8}{a:left,left,right}{{.name}}\t{{.quantity}}\t{{.total_minor | money}}
{if:modifiers}{each:modifiers}
  {a:left}  + {{.name}}: {{.price_minor | money}}
{/each}{/if}
{/each}
---
{w:auto,12}{a:left,right}Итого:\t{{.total_minor | money}} {{.currency_code}}
{if:discount_total_minor}
{w:auto,12}{a:left,right}Скидка:\t-{{.discount_total_minor | money}}
{/if}
{if:taxes}{each:taxes}
{w:auto,12}{a:left,right}{{.name}}:\t{{.amount_minor | money}}
{/each}{/if}
---
{a:center}{{.cashier_name}}
{a:center}{{.business_date}} | Смена №{{.shift_number}}
{if:is_copy}{a:center}*** КОПИЯ ***{/if}
{cut}
```

### Default ticket template (CPL=48)

```
{a:center}{{.restaurant_name}}
---
{a:center}{f:double}БИЛЕТ
{a:center}№ {{.ticket_display_number}}
---
{a:center}{f:double}{{.service_name}}
---
{qr:{{.qr_payload}}}
---
{a:center}{{.sale_date_local}} {{.sale_time_local}}
{a:center}Сумма: {{.price_minor | money}} {{.currency_code}}
{if:validity_date_local}
{a:center}Действителен до: {{.validity_date_local}}
{/if}
{if:is_copy}{a:center}*** КОПИЯ ***{/if}
{a:center}{{.cashier_name}} | Смена №{{.shift_number}}
{cut}
```

## `shared/platform/receipt` — структура пакета

Запланировано до пилота:

```
shared/platform/receipt/
  ir/
    block.go          -- IR типы (Block interface, TextBlock, QRBlock, ...)
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
    layout.go         -- shared column layout logic (используется обоими renderers)
    layout_test.go
    fixtures/         -- fixture IR + CPL -> ожидаемый SVG
  engine/
    engine.go         -- template engine: content + print context -> IR
    engine_test.go
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
- `docs/sync/directional-sync-ownership.md` (stream `receipt_templates`);
- `docs/backend/CLOUD-BACKEND-SPEC.md` (routes Cloud API);
- `docs/backend/POS-BACKEND-SPEC.md` (print queue routes, print_jobs schema);
- `docs/backend/POS-DATA-AND-MIGRATIONS.md` (Edge SQLite schema);
- `docs/CURRENT-FUNCTIONAL-STATE.md` (после завершения компонента).

## Оставшиеся риски и открытые вопросы

- Hardware acceptance (реальный принтер TCP/USB): обязателен до Done на POS-64.
  В итоговом Plane comment указать проверенные модели.
- CP866 принтер-specific: некоторые принтеры не поддерживают codepage 17 (PC866) и
  требуют codepage 16 (PC852) или другой. Нужна проверка на целевом принтере и
  документирование в printer_class.
- QR нативный ESC/POS (`GS ( k`): не все дешёвые принтеры поддерживают model 2.
  Растровый QR через `shared/platform/receipt/escpos` является fallback.
- Windows USB: `\\.\USB001` путь зависит от порядка подключения. В production
  использовать конфигурируемый `address` в printer config.
- `COPY` marker при reprint: должен быть виден на оба типа документов (precheck, ticket).
