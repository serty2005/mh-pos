# Edge Hardware Adapter Protocol

Статус: запланировано далее (архитектура зафиксирована, реализация не начата).

Связано: ADR-018 (внешние hardware-адаптеры), `RECEIPT-PRINT-SPEC.md`,
`EDGE-PRINT-ROUTING-SPEC.md`, `CLOUD-BACKEND-SPEC.md` (Printers, POS-82/83),
физическая проверка POS-86 на реальном USB-принтере 2026-07-01.

Этот документ — единственный источник истины для протокола связи между `pos-edge.exe`
(core) и внешними hardware-адаптерами (первый — `windows-printers`). Он написан так, чтобы
под него можно было подключить второй, третий адаптер (другая ОС, другой класс устройств —
фискальный регистратор, весы, эквайринг) без изменения core-протокола.

## 1. Роли

- **Core (`pos-edge.exe`)** — владеет бизнес-логикой: routing, print_jobs/targets, retry,
  RBAC, sync с Cloud. Не содержит ни одной Windows-специфичной системной вызовы работы с
  принтером. Выступает protocol client и process supervisor по отношению к адаптерам.
- **Адаптер (`windows-printers.exe`, далее другие)** — отдельный exe. Владеет всей
  низкоуровневой, ОС/hardware-специфичной логикой: обнаружение устройств (USB PnP
  enumeration, сетевой port-scan), управление Windows print queue/driver там, где это
  нужно для активации устройства (см. находки POS-86: `Add-PrinterDriver`, `Add-Printer`
  переводят `USBPRINT`-класс устройства из `Unknown` в `OK`), и непосредственная отправка
  уже отрендеренных байт на устройство. Адаптер не знает про `print_jobs`, `print_routes`,
  restaurant/print context — он оперирует только "устройство" и "байты".
- Core может запускать несколько адаптеров одновременно (по одному на класс устройств);
  адаптер обслуживает один class/kind (`windows-printers` — только принтеры).

**Принцип: адаптер не принимает решений о конфигурации.** Адаптер — исполнитель низкого
уровня, а не источник истины ни для одной настройки:

- Для `transport: "tcp"` адаптер **не хранит и не выбирает** ни адрес, ни порт сам —
  `address`/`port` уже сегодня Cloud-owned значения в `receipt_printers` (POS-82) и в каждом
  `print.send` приходят от core явно, беря их из этого уже существующего Cloud-owned поля.
  Адаптер просто подключается к переданному `address:port` и пишет байты; он не хранит их
  между вызовами, не переиспользует "последний известный" адрес и не пытается сам решить,
  на какой порт слать, если явные `address`/`port` не переданы — это ошибка вызова, а не
  повод для адаптера угадывать.
- Для `transport: "usb"` у core физически нет портируемого адреса (см. ADR-018 и находки
  POS-86), поэтому core передаёт `binding_ref` — непрозрачный идентификатор, который сам
  адаптер ранее сообщил через `discover.result`. Это не "решение" адаптера о конфигурации,
  а просто резолв ранее сообщённого им же идентификатора в текущий OS-хендл устройства.
- Discovery (`discover.request`/`discover.result`, сетевой скан включительно) — единственная
  функция адаптера, где он самостоятельно "смотрит и сообщает, что видит". Эта информация
  используется core/Cloud только для того, чтобы оператору было из чего выбрать; сама
  печать (`print.send`) никогда не полагается на данные скана как на источник настроек —
  только на то, что явно передано в конкретном вызове.

## 2. Транспорт

- Local-only. Адаптер и core работают на одной машине.
- Windows: named pipe (`\\.\pipe\mh-pos-edge-adapter-<kind>-<instance>`) как основной
  транспорт — не требует открытого TCP-порта, ACL ограничивает доступ до текущего
  пользователя/сервисной учётной записи Edge.
  Loopback TCP (`127.0.0.1:0`, ephemeral port, сообщается адаптеру через argv/env при
  запуске) — fallback-транспорт и единственный вариант для будущих не-Windows core/адаптеров;
  включается только если named pipe недоступен. Никогда не биндиться на `0.0.0.0`.
- Кадрирование сообщений: длина-префикс (4 байта, big-endian uint32) + JSON UTF-8 payload.
  JSON выбран (не protobuf/gRPC), чтобы не тянуть codegen-тулинг в маленький Windows-adapter
  и держать протокол легко читаемым при отладке.
- Один conceptual "channel" = одно duplex-соединение на время жизни адаптера. Реконнект
  после падения адаптера — новое соединение с новым handshake (см. §4).

## 3. Жизненный цикл процесса адаптера

1. Core запускает адаптер как child-процесс при старте (или at first use, если адаптер
   не установлен как autostart-компонент — конкретика вынесена в installer/deployment doc
   будущей итерации), передавая через argv/env: pipe name или loopback address, одноразовый
   auth token, `core_version`.
2. Адаптер подключается к каналу и присылает `adapter.hello` (см. §5).
3. Core держит heartbeat (`health.ping`/`health.pong`) с интервалом (конфигурируемо,
   аналогично `POS_PRINT_WORKER_POLL_INTERVAL`); N пропущенных pong подряд → адаптер
   считается зависшим.
4. Если адаптер падает, зависает или не проходит heartbeat — core убивает процесс (если ещё
   жив), помечает адаптер `offline`, все in-flight запросы к нему заканчиваются
   контролируемой ошибкой (см. §7), и перезапускает адаптер с exponential backoff
   (аналогично `POS_SYNC_SENDER_RECLAIM_AFTER`-класса настроек, но отдельный конфиг:
   `POS_ADAPTER_RESTART_BACKOFF_*`).
5. Graceful shutdown: core посылает `adapter.shutdown`, ждёт `adapter.bye` с таймаутом,
   иначе kill.
6. Один и тот же адаптер kind может быть переустановлен/обновлён независимо от `pos-edge.exe`
   — core обязан работать и с адаптером более новой минорной версии, и без адаптера вовсе
   (адаптер недоступен → discovery/print через него недоступны, safe error, не panic).

## 4. Envelope сообщений

```json
{
  "v": 1,
  "id": "corr-...",
  "type": "discover.request",
  "payload": { }
}
```

- `v` — версия протокола (не версия адаптера); core отклоняет несовместимый major.
- `id` — correlation id, генерируется отправителем запроса; ответ содержит тот же `id`.
- `type` — см. §5.
- Ошибки — отдельный `type: "error"` с `payload.error_code` (стабильный, safe, без raw
  internal details — тот же принцип, что и HTTP error contract в `AGENTS.md`) и `in_reply_to`.

## 5. Типы сообщений

### Handshake / health

- `adapter.hello` (adapter → core): `{ kind: "windows-printers", adapter_version, capabilities: ["usb_discovery","network_scan","raw_print","queue_management"], instance_id }`.
- `adapter.shutdown` (core → adapter), `adapter.bye` (adapter → core).
- `health.ping` (core → adapter) / `health.pong` (adapter → core).

### Discovery (adapter обнаруживает устройства)

- `discover.request` (core → adapter): `{ scan_usb: bool, scan_network: bool, network_ports: [9100, 6001, ...], network_targets: ["auto_subnet"] | ["192.168.1.0/24"], timeout_ms }`.
  Сетевой скан — явный, по запросу (из Edge UI/Cloud-triggered poll), не непрерывный
  background job, чтобы не создавать шум/нагрузку/security-вопросы на сети ресторана без
  явного действия оператора.
- `discover.result` (adapter → core), ответ на `discover.request`: `{ devices: [DiscoveredDevice, ...] }`.
- `discover.event` (adapter → core, unsolicited, опционально на будущее): hot-plug событие
  (устройство подключено/отключено) без явного request — тот же `DiscoveredDevice` shape.

`DiscoveredDevice`:

```json
{
  "binding_ref": "usb:1fc9:2016:0020416A82A8",
  "transport": "usb",
  "display_vendor": "Xprinter",
  "display_model": "XP-365B",
  "display_label": "Xprinter XP-365B (USB)",
  "queue_state": "unconfigured|queue_ready",
  "seen_at": "2026-07-01T12:00:00Z"
}
```

Для `transport: "tcp"`: `binding_ref` = `tcp:<host>:<port>` (сам host:port МОЖЕТ входить в
`binding_ref`, так как это не секрет и не хрупкий OS-путь — в отличие от USB device path,
IP:port валиден вне контекста конкретной ОС), `display_label` — то, что удалось получить
(reverse DNS / просто адрес, если имя недоступно).

`binding_ref` — стабильный (по VID/PID/serial для USB, по host:port для TCP), детерминированный
identifier уровня адаптера. Он **никогда не превращается в Cloud-owned значение** (см. §7 и
ADR-018) — Cloud видит только `display_*` поля через external-событие; `binding_ref` целиком
остаётся Edge-local и известен только core + адаптеру на этой машине.

### Печать (core просит адаптер отправить уже готовые байты)

- `print.send` (core → adapter), payload зависит от `transport` и всегда содержит явные
  connection-параметры — адаптер не хранит и не выбирает их сам (см. принцип в §1):

  ```json
  // transport: "tcp" — address/port приходят от core (Cloud-owned receipt_printers),
  // адаptер только подключается и пишет, ничего не запоминает между вызовами.
  { "transport": "tcp", "address": "10.25.1.201", "port": 9100, "payload_base64": "...", "timeout_ms": 5000 }

  // transport: "usb" — адаптер резолвит ранее сообщённый им же binding_ref в текущий
  // OS-хендл устройства; это не "решение о конфигурации", а просто lookup.
  { "transport": "usb", "binding_ref": "usb:1fc9:2016:0020416A82A8", "payload_base64": "...", "timeout_ms": 5000 }
  ```

  `payload_base64` — уже отрендеренные ESC/POS-байты (рендеринг остаётся в core/
  `shared/platform/receipt`, адаптер не парсит и не рендерит документ).
- `print.result` (adapter → core): `{ status: "sent"|"failed", error_code?, error_detail_safe? }`.
  `sent` означает, что адаптер успешно записал байты в устройство (`WriteFile` вернул
  успех) — не гарантия, что бумага физически вышла (аппаратные ошибки принтера типа "нет
  бумаги" ESC/POS обычно не репортит синхронно); это ограничение остаётся таким же, каким
  было при прямом `escpos.WriteRaw` до вынесения в адаптер.

### Управление очередью (только для USB, Windows-specific операции)

- `queue.ensure` (core → adapter): `{ binding_ref }` — для устройства, у которого
  `queue_state = "unconfigured"`, адаптер создаёт Windows print queue (эквивалент найденной
  во время POS-86 hardware-проверки последовательности: подобрать/поставить драйвер
  "Generic / Text Only", `Add-Printer` на нужный dynamic port), чтобы устройство стало
  реально открываемым по device interface path. Идемпотентно: повторный вызов на уже
  готовую очередь — no-op.
- `queue.ensure.result` (adapter → core): `{ status: "ready"|"failed", error_code? }`.

## 6. Edge-local физическая привязка принтера (core-side, не протокол, но напрямую зависит от него)

Core хранит в Edge SQLite новую таблицу (managed baseline, обычная migration policy)
`printer_physical_bindings`:

```
printer_id      TEXT NOT NULL REFERENCES receipt_printers(id)  -- логический Cloud-owned принтер
adapter_kind    TEXT NOT NULL   -- 'windows-printers'
binding_ref     TEXT NOT NULL   -- opaque, из DiscoveredDevice; только для transport='usb'
transport       TEXT NOT NULL CHECK (transport = 'usb')  -- см. ниже: tcp сюда не попадает
display_label   TEXT            -- для UI, не участвует в логике печати
bound_at        TIMESTAMPTZ NOT NULL
bound_by_employee_id TEXT
UNIQUE (printer_id)
```

`printer_physical_bindings` существует **только для `transport='usb'`**. Для `tcp` эта
таблица не используется и не заполняется: `address`/`port` уже сегодня Cloud-owned поля
`receipt_printers` (POS-82) — Edge и так знает их авторитетно, без всякого discovery/binding
шага, и адаптер не должен ничего "запоминать" за Edge (см. принцип в §1).

Print worker при обработке `print_job_targets` резолвит физическое назначение так:

1. `receipt_printers.type = 'tcp'`: `address`/`port` берутся напрямую из
   `receipt_printers` (как сегодня) и передаются либо (a) в текущий прямой
   `escpos.WriteRaw` в процессе `pos-edge.exe` без адаптера, либо (b) в `print.send` с
   `transport: "tcp"` и явными `address`/`port`, если печать в этой итерации переведена на
   адаптер. В обоих случаях источник настроек — Cloud, а не адаптер и не какой-либо
   локальный кэш; выбор (a) или (b) — конфигурационный, не меняет, откуда берутся
   `address`/`port`.
2. `receipt_printers.type = 'usb'`: обязателен `printer_physical_bindings` для этого
   `printer_id`; печать идёт только через адаптер (`print.send` с `transport: "usb"` и этим
   `binding_ref`). Если привязки нет — печать не пытается угадать адрес, job уходит в
   обычный retry/failed с safe error_code `PRINT_ROUTING_INVALID`/аналогом (Edge-оператору
   нужно явно привязать физическое устройство через `POST .../binding`, см. ниже).
3. Явно НЕ реализуется автоматический fallback между режимами (adapter↔direct-write для
   tcp, или "угадать usb-адрес" при отсутствии binding) внутри одной попытки — поведение
   печати должно оставаться предсказуемым и объяснимым по логам (`error_code` явно говорит,
   через что шла попытка), а не skip/cascade-логикой.

Edge HTTP API (RBAC `pos.print_routing.manage`, тот же permission, что уже владеет
`print_routes`):

- `GET /api/v1/print-routing/adapters/{kind}/discovered` — проксирует `discover.request` к
  адаптеру (usb и/или tcp-скан, по параметрам запроса) и возвращает `DiscoveredDevice[]`
  для локального UI выбора. Для `tcp`-кандидатов результат — справочная информация
  ("вот что видно в сети сейчас"); он не меняет и не заменяет `receipt_printers.address/port`
  автоматически — оператор по-прежнему явно подтверждает/вводит `address`/`port` там, где
  это Cloud-owned поле, скан лишь подсказывает значение.
- `POST /api/v1/print-routing/printers/{id}/binding` — только для `receipt_printers.type = 'usb'`;
  `{ adapter_kind, binding_ref }`; создаёт/обновляет `printer_physical_bindings`; если
  `queue_state` устройства — `unconfigured`, сначала выполняет `queue.ensure`. Для `type = 'tcp'`
  этот endpoint возвращает `400 ErrInvalid` — привязка не нужна и не поддерживается.
- `DELETE /api/v1/print-routing/printers/{id}/binding` — снимает привязку USB-принтера;
  печать не переключается ни на какой fallback-адрес автоматически — job уходит в
  `failed`/`PRINT_ROUTING_INVALID` до новой явной привязки (см. п.2 выше).

Это даёт последний пункт из требований: "пользователь может со стороны Edge выбрать другой
физический USB под тот же логический принтер, установить новую очередь или выбрать
существующую" — реализуется через `discovered` + `binding` endpoints, без участия Cloud.

## 7. Edge → Cloud external-события обнаружения (для Cloud-side выбора при создании принтера)

Обнаруженные устройства — это **не** `receipt_printers` master data и не отдельная
подсистема поверх существующего sync: они переиспользуют уже существующий Edge → Cloud
event/outbox механизм (тот же путь, которым Edge уже отправляет в Cloud доменные события),
добавляя новый `command_type`/`event_type`, например `HardwareDeviceDiscovered`:

```json
{
  "node_device_id": "...",
  "adapter_kind": "windows-printers",
  "binding_ref": "usb:1fc9:2016:0020416A82A8",
  "transport": "usb",
  "display_vendor": "Xprinter",
  "display_model": "XP-365B",
  "display_label": "Xprinter XP-365B (USB)",
  "seen_at": "2026-07-01T12:00:00Z"
}
```

Важно, и здесь `usb` и `tcp` принципиально различаются:

- Для `transport: "usb"` — `binding_ref` не адрес, который Cloud может использовать для
  печати; это просто стабильный идентификатор "устройство, увиденное этим Edge-узлом",
  нужный только для того, чтобы при последующей привязке (`printer_physical_bindings`)
  Edge-оператор мог узнать то же устройство в списке, которое видел Cloud-оператор при
  создании принтера. Cloud никогда не хранит этот идентификатор как настройку печати и
  никогда не инициирует печать по нему напрямую — печатью по-прежнему управляет только Edge.
- Для `transport: "tcp"` — `binding_ref` это по сути сам `host:port` (см. DiscoveredDevice
  в §5), не opaque-токен: сетевой адрес принтера не является хрупкой OS-специфичной
  сущностью, в отличие от USB device path. Поэтому при создании принтера в Cloud UI
  выбранный tcp-кандидат может быть напрямую скопирован в Cloud-owned `address`/`port`
  (см. §8) — это не нарушает принцип "адаптер не решает за Edge/Cloud": адаптер лишь
  сообщил, что увидел на сети, а решение сохранить это как `address`/`port` принимает
  оператор в Cloud UI, как и при ручном вводе.

Cloud-side обработка (не master data, а *ephemeral candidate projection*, отдельная от
`cloud_printers`):

- Новая таблица/проекция `printer_discovery_candidates(node_device_id, adapter_kind,
  binding_ref, transport, display_vendor, display_model, display_label, first_seen_at,
  last_seen_at)`, upsert по `(node_device_id, adapter_kind, binding_ref)` при получении
  каждого `HardwareDeviceDiscovered`.
- Кандидаты не считаются "надёжными" бесконечно: если устройство не появлялось в
  последнем скане свежее какого-то TTL (конкретное значение — предмет отдельной
  codegen-итерации), UI помечает его как stale, но не удаляет молча (оператор должен видеть,
  что принтер "не виден сейчас", а не терять историю).
- Только для отображения оператору в Cloud UI при создании/редактировании принтера — не
  участвует в `printers` mastersync stream к Edge и не хранится в `cloud_printers`.

## 8. Cloud UX-поток создания принтера (продуктовое ограничение для будущей codegen-итерации)

1. Оператор открывает форму создания принтера в Cloud UI (POS-83, существующий экран).
   Вместо ручного ввода `type/address/port` видит список обнаруженных устройств
   (сгруппированный по Edge-узлу/терминалу, так как у одного ресторана может быть
   несколько Edge-машин с разными USB-принтерами) — источник: `printer_discovery_candidates`.
2. Оператор выбирает конкретное обнаруженное устройство. Для `usb` он не видит и не вводит
   `binding_ref`/device path — эти детали остаются opaque и на Edge-стороне. Для `tcp` выбор
   кандидата подставляет `address`/`port` в форму как обычное Cloud-owned значение (оператор
   может увидеть/поправить IP при желании — это не хрупкий OS-путь, а простой сетевой адрес),
   но ему не нужно узнавать этот адрес самостоятельно или лазить в сеть — скан адаптера уже
   его нашёл.
3. Оператор задаёт: имя принтера в системе (`name`), CPL, отступы/margins (если рендер их
   поддерживает на момент реализации — уточнить относительно текущего `RenderOptions`), тип
   отреза. `document_types`/routing назначаются отдельно через `print_routes` (уже
   существующий Edge-local механизм, не меняется). Явно вне этого шага: ручной ввод сложных
   технических идентификаторов (device path, binding_ref) оператором.
4. После создания принтер уходит в обычный `printers` mastersync stream к Edge (как
   сегодня, без изменений транспортного механизма) — для `tcp` в пакете уже есть готовый к
   использованию `address`/`port`, дополнительных шагов на Edge не требуется.
5. Только для `usb`: Edge получает логическую конфигурацию принтера обычным apply-путём
   (`receipt_printers`), но физическая привязка остаётся пустой, пока Edge-оператор явно не
   свяжет её через `POST .../binding` (см. §6) — это может быть тот же Edge-узел, что был
   выбран на шаге 1 (типичный случай), либо другой (оператор Edge физически переставил
   принтер на другой терминал/порт).

Конкретная разметка форм, копирайтинг, i18n-ключи — не фиксируются этим документом,
остаются на усмотрение будущей UI-codegen-итерации (см. `docs/project-management/
ALPHA-LAUNCH-CODEGEN-ITERATIONS.md`, Wave 4).

## 9. Явно вне scope на этом этапе

- Непрерывный background-scan сети/USB без явного запроса — только on-demand.
- Автоматический silent-fallback между адаптером и direct-write в рамках одной попытки
  печати (см. §6, п.3).
- Адаптеры не на текущей машине (remote adapter over network) — протокол это не запрещает
  архитектурно, но эксплуатационно не поддерживается и не тестируется в этой итерации.
- TTL/очистка `printer_discovery_candidates` — конкретное число и механизм очистки решает
  codegen-итерация, реализующая Cloud-сторону.
- Учёт нескольких адаптеров одного kind на одной машине (например, два экземпляра
  `windows-printers`) — не предусмотрено; один kind = один запущенный процесс на Edge-узел.
