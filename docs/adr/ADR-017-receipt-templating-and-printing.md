# ADR-017: Система шаблонов и нефискальной печати

Статус: принято.

Дата: 2026-06-24.

Ссылки: ADR-015 (хранение и миграции), ADR-016 (ClickHouse), `SPECv1.3.md`, POS-64, `docs/backend/RECEIPT-PRINT-SPEC.md`.

## Контекст

POS Edge обязан печатать нефискальные документы на физическом ESC/POS-принтере:
предчек, нефискальный чек, QR-билет, сервисный кухонный чек, внесение/изъятие денег,
документ приёмки. Reprint-flow уже возвращает immutable snapshot, но не имеет print
orchestration, template engine и printer interface.

**Главное бизнес-требование:** из одного источника — разметки шаблона — должны получаться
и экранный SVG-предпросмотр, и команды печати. Расхождение предпросмотра с реальной
печатью недопустимо.

**Фискальная граница:** фискальное тело чека (54-ФЗ) формирует фискальный регистратор
вне этой системы. Print subsystem владеет только нефискальными документами:
`precheck`, `check_nonfiscal`, `ticket`, `kitchen_service`, `cash_in_out`, `acceptance`.
Поле `document_type` содержит только эти значения; фискальные типы в enum запрещены.

**Ограничения среды:**
- POS Edge работает на Windows; Node.js-рантайм в Edge-процессе неприемлем.
- Кириллица на принтере требует CP866/PC866 для нативного ESC/POS-текста.
- Шаблоны width-independent; ширина ленты (32/48 CPL) задаётся при рендере.
- Cloud является master-data authority; Edge работает с локальной копией.

## Рассмотренные варианты

### A. Прямая генерация ESC/POS без шаблонного слоя

Каждый тип документа — отдельный Go-код, жёстко генерирующий байты ESC/POS.

- ✅ Просто начать.
- ❌ Preview требует полной дублирующей реализации.
- ❌ Редактирование только через деплой кода.

### B. HTML/CSS → растровое изображение → ESC/POS

Headless-браузер (Puppeteer/wkhtmltopdf) рендерит HTML, результат — растр для ESC/POS.

- ✅ CSS-вёрстка без дополнительного DSL.
- ❌ Требует браузерного движка (тяжело, нет на Windows без Node.js/Chromium).
- ❌ Медленно; нативный текстовый ESC/POS невозможен.
- ❌ Кириллица в растре сложнее.

### C. IR + ReceiptLine-inspired Level 1 (Go-native) + Level 2 escape-hatch

Определить Intermediate Representation (IR) — width-independent список блоков. Реализовать
Go-native парсер ReceiptLine-синтаксиса (MIT-spec, не JS-код). Два рендерера из IR:
ESC/POS (нативный текст + CP866, растровый fallback) и SVG (preview). Shared Go-пакет
`shared/platform/receipt` используется и Edge, и Cloud. Level 2: `go text/template` как
gated escape-hatch для редких сложных шаблонов.

- ✅ Один источник → preview + печать; layout физически одинаков.
- ✅ Нет Node.js на Edge; работает на Windows.
- ✅ Width-independent — один шаблон для 58мм и 80мм.
- ✅ CP866 нативный путь + растровый fallback.
- ✅ Техподдержка редактирует текстовые шаблоны; Cloud UI показывает SVG-preview.
- ⚠️ Нужен Go-парсер по спецификации (~400 строк).
- ⚠️ ReceiptLine spec не покрывает все ESC/POS-функции; достаточен для нашего scope.

## Решение: Вариант C

### ReceiptLine Level 1

Язык шаблонов уровня 1. Мы реализуем Go-native парсер по открытой спецификации
ReceiptLine (MIT, https://github.com/receiptline/receiptline). JS-код не копируется,
реализуется spec. Attribution: README пакета и этот ADR.

Поддерживаемые конструкции Level 1 (полный ReceiptLine spec):

| Конструкция         | Описание                                                |
|---------------------|---------------------------------------------------------|
| Текстовые строки    | Обычный текст; выравнивание через `{a}` директиву      |
| Многоколоночные     | `col1\tcol2\tcol3` — разделитель `\t`                  |
| Ширина колонок      | `{w:N,M}` или `{w:auto}`                               |
| Выравнивание        | `{a:left\|center\|right}` на строку или блок           |
| Шрифт               | `{f:normal\|double\|smaller}` (расширение/сужение)     |
| Жирный текст        | `{b}` / `**bold**`                                     |
| Горизонтальная черта| `---` — разделитель                                    |
| QR-код              | `{qr:<payload>}` — нативный ESC/POS QR или растровый  |
| Штрихкод            | `{barcode:<type>:<data>}` — Code39/EAN/UPC/ITF         |
| Изображение         | `{image:<path\|base64>}` — растровый 1-bit bitmap      |
| Перевод строки / пробел | пустая строка, `{s:N}` пробел N строк              |
| Рез бумаги          | `{cut}` — полный рез                                   |
| Частичный рез       | `{cut:partial}` — частичный рез                        |
| Открытие ящика      | `{drawer}` — импульс на кассовый ящик                  |
| Условие             | `{if:<expr>}...{/if}` — простое bool-условие           |
| Цикл                | `{each:<key>}...{/each}` — итерация по списку          |

Переменные подставляются из print context через `{{.FieldName}}`.

### Level 2 (escape-hatch)

`go text/template` для редких сложных шаблонов. Недоступен через Cloud UI редактор без
явного флага `level: 2` в метаданных шаблона + `POS_TEMPLATE_LEVEL2_ENABLED=true` в
deployment config. По умолчанию отключён; Level 1 всегда достаточен для нашего scope.

### Intermediate Representation (IR)

Width-independent Go-типы в `shared/platform/receipt/ir`:

```go
// Block — один элемент документа; рендерер получает целевой CPL.
type Block interface{ blockMarker() }

type TextBlock struct {
    Lines     []TextLine
    Alignment Alignment // Left, Center, Right
    Font      Font      // Normal, DoubleWidth, Smaller
    Bold      bool
}

type TextLine struct {
    Columns []Column
}

type Column struct {
    Text  string
    Width int    // 0 = auto
    Align Alignment
}

type RuleBlock struct{} // горизонтальная черта

type SpaceBlock struct{ Lines int }

type QRBlock struct {
    Payload string
    Size    int // 1–8, 0 = auto
    Model   int // 1 or 2 (default 2)
}

type BarcodeBlock struct {
    Type string // code39, ean13, ean8, upca, upce, itf, codabar
    Data string
    HRI  bool // human-readable interpretation below
}

type ImageBlock struct {
    Data   []byte // 1-bit packed bitmap, width aligned to 8
    Width  int    // pixels
    Height int    // pixels
}

type CutBlock struct{ Partial bool }

type DrawerBlock struct{}
```

### Рендерер ESC/POS

Go, `shared/platform/receipt/escpos`:

- Нативный текстовый путь для ASCII + CP866 (codepage 17 / PC866).
- Растровый fallback (PNG/bitmap → ESC/POS `GS v 0`) для:
  - символов вне CP866;
  - принтеров с флагом `RasterOnly: true`;
  - блоков `Image`, `QR` (если принтер без встроенного QR), `Barcode`.
- QR через нативный `GS ( k` (model 2, error correction M) с fallback на растр.
- Поддержка инициализации принтера, сброса форматирования, partial/full cut.
- Printer interface: TCP (`net.Dial("tcp", addr)`) + Windows USB (raw port write через
  `\\.\USB001` или аналог; `os.OpenFile` на Windows).
- `PrinterConfig` содержит: `Type` (tcp|usb), `Address`, `Port`, `CPL`, `PrinterClass`,
  `RasterOnly`, `PaperCutType` (full|partial|none).

### Рендерер SVG

Go, `shared/platform/receipt/svg`:

- Рендерит IR в строку `<svg>` с заданным CPL.
- Layout физически идентичен ESC/POS: одинаковые ширины колонок, одинаковые переносы
  строк при одном CPL.
- Cloud Backend endpoint `GET /api/v1/receipts/preview`:
  принимает `{template_content, document_type, cpl, print_context}`;
  возвращает `Content-Type: image/svg+xml`.
- React Cloud UI вызывает этот endpoint и отображает SVG тегом `<img>`.

### Print context

Плоская Go-struct, проецируемая из immutable precheck/check/ticket snapshot. Стабильный
контракт между доменом и шаблонами; не читает текущий каталог.

Полная JSON-схема задокументирована в `docs/backend/RECEIPT-PRINT-SPEC.md`.

### `receipt_template` как Cloud master-data

PostgreSQL table `receipt_templates`:

```sql
id              UUID PRIMARY KEY,   -- UUIDv7
org_id          UUID NOT NULL,
restaurant_id   UUID,               -- NULL = tenant-level default
document_type   TEXT NOT NULL,      -- precheck|check_nonfiscal|ticket|kitchen_service|cash_in_out|acceptance
name            TEXT NOT NULL,
description     TEXT,
content         TEXT NOT NULL,      -- ReceiptLine markup
level           INT NOT NULL DEFAULT 1,  -- 1=ReceiptLine, 2=go text/template
cpl             INT NOT NULL,       -- 32 or 48
printer_class   TEXT NOT NULL DEFAULT 'generic',
is_default      BOOL NOT NULL DEFAULT FALSE,
version         INT NOT NULL DEFAULT 1,
is_active       BOOL NOT NULL DEFAULT TRUE,
created_at      TIMESTAMPTZ NOT NULL,
updated_at      TIMESTAMPTZ NOT NULL
```

UNIQUE `(org_id, restaurant_id, document_type, is_default)` WHERE `is_default = TRUE`.

Новый ingest stream `receipt_templates` в Cloud -> Edge sync exchange.
Edge хранит в SQLite `receipt_templates` (только `is_active = TRUE` строки).

### Print queue на Edge

SQLite `print_jobs`:

```sql
id              TEXT PRIMARY KEY,   -- UUIDv7
document_type   TEXT NOT NULL,
template_id     TEXT NOT NULL,
print_context   TEXT NOT NULL,      -- JSON
status          TEXT NOT NULL,      -- pending|printing|printed|failed|cancelled
attempts        INT NOT NULL DEFAULT 0,
last_error      TEXT,
created_at      TEXT NOT NULL,
updated_at      TEXT NOT NULL,
printed_at      TEXT
```

Print worker: dequeue → template lookup → context projection → Level 1/2 render → IR →
ESC/POS render → send to printer. Retry: до 3 попыток, exponential backoff 2s/5s/15s.
Print retry НЕ повторяет payment и НЕ создаёт ticket.

HTTP:
- `GET /api/v1/print/jobs/{id}` — статус задачи (RBAC: `pos.print.status`).
- `POST /api/v1/print/jobs/{id}/retry` — ручной ретрай (RBAC: `pos.print.retry`).

### Printer routing

Конфигурационная таблица в Edge (из `pos-backend/config/`):

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
      "paper_cut_type": "full"
    }
  ],
  "routing": [
    { "document_type": "precheck",         "printer_id": "main" },
    { "document_type": "check_nonfiscal",  "printer_id": "main" },
    { "document_type": "ticket",           "printer_id": "main" },
    { "document_type": "kitchen_service",  "printer_id": "main" },
    { "document_type": "cash_in_out",      "printer_id": "main" },
    { "document_type": "acceptance",       "printer_id": "main" }
  ]
}
```

Фаза 1: один принтер, routing по document_type. Фаза 2: routing по station_id/section_id.

## Последствия

- Новый Go-модуль `shared/platform/receipt` импортируется в `pos-backend` и
  `cloud-backend`; циклических зависимостей нет (`shared/platform` не импортирует
  backend-specific пакеты).
- Новый Cloud -> Edge stream `receipt_templates` добавляется в `mastersync.Service`,
  `directional-sync-ownership.md`, `CLOUD-BACKEND-SPEC.md`, `POS-BACKEND-SPEC.md`.
- `seed-dev-system.py` создаёт default шаблоны через Cloud API.
- `CLOUD_OWNED_SEED_SURFACES` обновляется при добавлении receipt_templates endpoint.
- Hardware acceptance (реальный принтер: TCP или Windows USB) обязателен перед
  переводом POS-64 в Done.
- После Фазы 1: обновить `docs/CURRENT-FUNCTIONAL-STATE.md` и `ROADMAP.md`.

## Фазы реализации

### Фаза 1 — до первого выставочного запуска (POS-64)

P1-1: ADR-017 + RECEIPT-PRINT-SPEC.md
P1-2: `shared/platform/receipt` — IR + ReceiptLine Level 1 parser
P1-3: IR → ESC/POS renderer (CP866, TCP/USB)
P1-4: IR → SVG renderer + Cloud preview endpoint
P1-5: `receipt_template` Cloud master-data + ingest stream Edge
P1-6: Print context projection из immutable snapshot
P1-7: Template engine + default шаблоны (precheck, ticket)
P1-8: Edge print queue + printer routing
P1-9: Cloud UI template editor + live SVG preview
P1-10: seed-dev-system.py + receipt smoke

### Фаза 2 — далее (после запуска)

P2-1: Дополнительные типы документов (kitchen_service, check_nonfiscal, cash_in_out, acceptance)
P2-2: QR нативный ESC/POS + растровый fallback для всех принтеров
P2-3: Level 2 шаблоны (go text/template, gated)
P2-4: e-receipt/PNG из Cloud (PNG endpoint поверх SVG renderer)
P2-5: Расширенный routing (station_id, мультипринтер per station)
