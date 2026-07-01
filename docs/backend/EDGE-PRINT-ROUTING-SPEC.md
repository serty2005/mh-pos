# Edge print routing: точки продаж, секции и targets

Статус: `реализовано сейчас` — жёсткая маршрутизация печати по document_type/scope_type,
per-printer FIFO worker, Edge HTTP API для `print_routes`/targets, Cloud-owned CRUD+sync для
точек продаж/секций/столов (включая bootstrap-провижининг и защиты от удаления), печать как
гейт подтверждения оплаты (`checks.print_confirmed_at`) с retry и cancel-unconfirmed веткой.
`запланировано далее`: Edge settings UI для `print_routes`/targets (POS-88, переформулирована),
Cloud UI для точек продаж/секций (POS-102), Edge → Cloud override audit projection (POS-87),
offline-флоу автономного Edge (POS-101), exhibition smoke на реальном принтере (POS-89),
вынесение обнаружения/адресации физических принтеров (USB PnP + сетевой скан портов
9100/6001/custom) во внешний exe-адаптер `windows-printers` и Edge-local физическая
привязка принтера поверх текущего Cloud-owned `receipt_printers` (ADR-018,
`EDGE-HARDWARE-ADAPTER-PROTOCOL.md`).

Дата проверки: 2026-07-01 (Windows/Docker Desktop сессия — live-прогон `seed-dev-system.py`
против чистого docker-стека выполнен и зелёный, Playwright прошёл по Cloud UI/POS UI поверх
реального браузера, найден и исправлен CORS-баг деактивации принтеров, см. "Оставшиеся
риски"). Backend, Playwright и live e2e проверки закрыты; Windows-инсталлятор не собирался
из-за отсутствия npm/NSIS на этой машине.

## Назначение

Этот документ фиксирует Edge-side ресторанную схему печати для выставочного запуска: как
физический принтер (`receipt_printers`, Cloud-owned) назначается на документ через точку
продаж или секцию (`print_routes`, Edge-local), и как печать становится условием
подтверждения оплаты гостю.

Финансовые операции, payment, ticket issuance и check state не повторяются при retry печати.
Повтор печати меняет только локальные `print_jobs`/`print_job_targets` и audit диагностики.

## Модель владения данными

- `sales_points`, `restaurant_sections`, `tables.section_id` — **Cloud-owned master data**.
  Создание/деактивация — только через Cloud CRUD (`cloud-backend`, `organization.manage`).
  Edge получает их исключительно read-only через mastersync (streams `sales_points`,
  `restaurant_sections`, расширенный `floor`). Никакого write API для этих трёх сущностей на
  Edge нет.
- `print_routes` — **полностью Edge-local**: CRUD через `/api/v1/print-routing/routes`
  (`pos.print_routing.manage`/`pos.print_routing.view`), каждая мутация пишет
  `printer_route_override_audit` (`origin='edge_override'`). Push override в Cloud — задача
  `POS-87`, в этой итерации не реализован.
- `receipt_printers` (логическая конфигурация: имя, CPL, codepage, тип отреза) остаётся
  Cloud-owned как сегодня. Физический адрес/устройство под этой логической конфигурацией —
  запланированная отдельная Edge-local сущность (`printer_physical_bindings`), которую
  Edge заполняет через обнаружение адаптером `windows-printers`, а не через Cloud-поле
  `address`/`port`. См. ADR-018 и `EDGE-HARDWARE-ADAPTER-PROTOCOL.md`.

## Жёсткая маршрутизация document_type → scope_type (без fallback)

Закодировано один раз в `receipt.RequiredScopeType`/`receipt.RequiredSectionMode`
(`pos-backend/internal/pos/domain/receipt/routing.go`) и продублировано DB-триггерами
`print_routes_required_scope_*`/`print_routes_required_section_mode_*`:

| document_type     | scope_type   | restaurant_sections.mode | Статус использования |
| ----------------- | ------------ | ------------------------ | --------------------- |
| `check_nonfiscal` | `sales_point`| —                         | worker создаёт targets |
| `precheck`        | `section`    | `hall_section`            | worker создаёт targets |
| `ticket`          | `section`    | `hall_section`            | worker создаёт targets |
| `kitchen_service` | `section`    | `kitchen_workshop`        | конфигурация принимается, worker пока не создаёт такие jobs |
| `report`          | `restaurant` | —                         | конфигурация принимается, worker пока не создаёт такие jobs |

Старый fallback на `receipt_printers.document_types` полностью убран из worker'а.

## scope_id резолвится один раз при enqueue

`print_jobs.scope_id` заполняется в момент постановки job в очередь (внутри транзакции
`CapturePayment`/`IssuePrecheck`), не в момент обработки worker'ом:

- `check_nonfiscal` → `cashSession.SalesPointID` (сессия всегда привязана к точке продаж).
- `precheck`/`ticket` → `table.SectionID`, где `table = GetTable(order.TableID)` — прямой
  путь без особых случаев, так как стол обязан принадлежать секции (см. ниже).

## Стол обязан принадлежать секции; `EnsureSystemFloor`/`GetSystemTable` ретированы

- `tables.section_id NOT NULL REFERENCES restaurant_sections(id)`; `tables.hall_id` и
  `restaurant_sections.hall_id` — чисто декоративные UI-поля (схема зала), не участвуют в
  routing. DB-триггеры `tables_section_hall_mode_*` проверяют, что секция того же ресторана
  и `mode='hall_section'`.
- Cloud при создании ресторана безусловно (вне зависимости от лицензии `table-mode`)
  провижинит бутстрэп-секцию (`is_default=1`, `hall_id=NULL`) и бутстрэп-стол внутри неё
  (`tables.is_default=1`) — `bootstrapDefaultSectionAndTable` в `cloud-backend`.
- `sales_points.default_table_id NOT NULL REFERENCES tables(id)` — при создании точки
  продаж, если стол не указан явно, подставляется бутстрэп-стол ресторана.
- Старый Edge-local `EnsureSystemFloor`/`GetSystemTable`/`__counter__` (POS-53) полностью
  удалён из `pos-backend`. Заказ без явно выбранного гостевого стола резолвит стол как
  `cashSession.SalesPointID → salesPoint.DefaultTableID`.
- Защиты от удаления (cloud-backend, safe error): нельзя деактивировать/архивировать
  `is_default` стол или секцию; нельзя архивировать стол, если он `default_table_id` активной
  точки продаж; нельзя деактивировать секцию, если в ней есть такой стол; нельзя
  деактивировать последнюю активную точку продаж ресторана; `hall_section` нельзя
  активировать без хотя бы одного стола.

## Обязательность точки продаж у кассовой сессии

`cash_sessions.sales_point_id NOT NULL`. Команда открытия смены (`POST /api/v1/cash-shifts/open`)
требует `sales_point_id`, валидирует что точка активна и принадлежит ресторану. Точка продаж
никогда не создаётся автоматически — только вручную через Cloud CRUD или seed-скрипт.
`seed-dev-system.py` создаёт точку продаж сразу после создания ресторана и передаёт её id в
`cash-shifts/open`. Текущий `pos-ui-g` экран открытия смены **не обновлён** в этой итерации —
это отдельная, известная работа по UI вне scope `POS-86`.

## Per-printer FIFO буфер

Claim — на уровне `print_job_targets`, не `print_jobs` (`ClaimDuePrintJobTarget` в
`infra/sqlite/print_repository.go`): один и тот же принтер не может иметь два target'а в
статусе `processing` одновременно, порядок — `ORDER BY printer_id, created_at`.
`print_jobs.status`/`attempts` агрегируются из дочерних targets (`aggregatePrintJobStatus`):
succeeded — когда все required targets succeeded; failed — когда хотя бы один required target
исчерпал попытки; иначе pending с ближайшим `next_attempt_at`.

## Печать как гейт подтверждения оплаты

- После полной оплаты в очередь встаёт `document_type=check_nonfiscal` (источник — Check), не
  `precheck`. `IssuePrecheck` дополнительно сама ставит `precheck`-job (печать пречека гостю
  до оплаты, маршрут — секция).
- `checks.print_confirmed_at` стампится worker'ом (`MarkCheckPrintConfirmedIfReady`), когда
  check_nonfiscal-job и все ticket-jobs этого чека succeeded — не HTTP-таймаутом.
- `CapturePayment` коммитит Payment/Check/Ticket безусловно и быстро, затем ждёт
  `print_confirmed_at` bounded таймаутом (`ServiceOptions.PrintConfirmationWait`, production
  default 3s через `POS_PRINT_CONFIRMATION_WAIT_SECONDS`, в тестах — 0 = мгновенная проверка
  без polling, чтобы не подвешивать unit-тесты на реальное время). Ответ включает
  `payment.print_confirmation.{confirmed, targets}`.
- `GET /api/v1/checks/{id}/print-confirmation` — статус после таймаута.
- `POST /api/v1/checks/{id}/print-confirmation/retry` — job-level retry: пересобирает targets
  всех print_jobs чека (check_nonfiscal + ticket) из ТЕКУЩИХ активных `print_routes` — если
  принтер заменили, следующая попытка уйдёт на новый.
- `POST /api/v1/orders/{id}/cancel-unconfirmed` — только пока `print_confirmed_at IS NULL`,
  manager PIN (право `pos.order.cancel_unconfirmed`, по образцу `CancelPrecheck`).
  Транзакционно: `FinancialOperationCancellation` (kind=full, тот же shift/business date —
  `FinancialOperationRefund` здесь неприменим по `ensureFinancialBoundary`, он рассчитан на
  другой business date), void всех активных `ticket_units` чека, soft-cancel заказа
  (`orders.status='cancelled'`, `cancelled_at`). Три отдельных outbox-события:
  `CheckPrintUnconfirmedRefunded`, `TicketVoided` (на каждый билет), `OrderCancelled`. Заказ
  пропадает из активных списков, остаётся в БД со всей историей.
- POS UI для этого флоу (спиннер, модалка retry/отменить на экране оплаты) не построен —
  backend-контракт самодостаточен, экран — отдельная будущая работа (см.
  `CURRENT-FUNCTIONAL-STATE.md`).

## Edge HTTP API

- `GET /api/v1/print-routing/printers` — read-only список `receipt_printers`.
- `GET /api/v1/print-routing/sales-points`, `GET /api/v1/print-routing/sections` — read-only
  отражение Cloud-synced данных.
- `GET/POST/PATCH/DELETE /api/v1/print-routing/routes` — CRUD `print_routes`.
- `GET /api/v1/print/jobs/{id}` — включает `targets`.
- `POST /api/v1/print/jobs/{id}/retry` — job-level retry (пересобирает targets).
- `POST /api/v1/print/jobs/{id}/targets/{target_id}/retry` — retry одного target без
  пересборки routing.
- `GET /api/v1/checks/{id}/print-confirmation`, `POST .../retry`,
  `POST /api/v1/orders/{id}/cancel-unconfirmed`.

RBAC: `pos.print_routing.view`/`pos.print_routing.manage` (RoleManager — оба;
RoleSupportAdmin — только view), `pos.order.cancel_unconfirmed` (RoleManager, manager PIN).

## Cloud-backend

- Postgres schema: `sales_points` (`default_table_id NOT NULL`), `restaurant_sections`
  (`is_default`, `hall_id` nullable-декоративный), `tables.section_id NOT NULL`.
- CRUD: `POST/GET/PATCH/archive /api/v1/sales-points`, `/api/v1/restaurant-sections` (и
  зеркальные `/api/v1/master-data/...` пути), `organization.manage`.
- Mastersync streams `sales_points`, `restaurant_sections`; `floor` расширен `section_id`.
- Бутстрэп-провижининг секции+стола при `POST /api/v1/restaurants`.

## Тесты

- `pos-backend/internal/pos/infra/sqlite/print_routing_schema_test.go` — schema baseline.
- `pos-backend/internal/pos/app/service_test.go` — routing match, per-printer FIFO,
  target/job lifecycle и aggregation (`TestPrintQueue*`), mastersync ingest новых стримов
  (`TestApplyMasterDataSnapshotUpsertsRowsStateAndDoesNotCreateOutbox`).
- `pos-backend/internal/pos/app/print_confirmation_test.go` — bounded wait (happy path и
  timeout), job-level retry с пересборкой routing, полный `cancel-unconfirmed` (refund + void
  + soft-cancel + идемпотентность), RBAC/gate-проверки.
- `cloud-backend/internal/masterdata/app` — CRUD sales_points/restaurant_sections/tables,
  bootstrap-провижининг, защиты от удаления.

`go test ./...` зелёный в `pos-backend` и `cloud-backend`; `go vet ./...` чистый в обоих;
`git diff --check` чистый.

## Оставшиеся риски / manual-validation

Статус проверки: **закрыто живьём 2026-07-01** (Windows/Docker Desktop сессия, продолжение
доводки POS-86). Полный чек-лист ниже прогнан вживую через Playwright MCP по реальному
браузеру поверх поднятого `docker-compose.local.yml`; найденные регрессии либо подтверждены
как ожидаемые и safe (открытие смены), либо исправлены (CORS DELETE).

- `scripts/seed-dev-system.py` прогнан живьём (`--run-minimal-flow --run-kitchen-process-smoke`)
  против чистого docker-стека (down -v/up -d --build) — зелёный полностью, создаёт точку
  продаж, `print_routes` для `check_nonfiscal`/`precheck`/`ticket`, оба print job дошли до
  терминального `failed` (ожидаемо без реального принтера, worker не завис).
- POS UI (`pos-ui-g`): экран открытия смены (`src/components/cash/POSCashSection.tsx`)
  подтверждён живьём — не передаёт `sales_point_id`, `POST /cash-shifts/open` возвращает 400
  с safe error contract (`VALIDATION_FAILED`/`errors.validation`/`correlation_id`, без raw
  Go/SQL деталей). Обойдя регрессию прямым API-вызовом с явным `sales_point_id`, полный флоу
  заказ → позиция с модификатором → precheck → оплата (наличные) пройден до конца через
  реальный UI; экран оплаты не падает и показывает "Оплата принята, заказ закрыт" несмотря на
  физический сбой печати. Экран оплаты по-прежнему не показывает print-confirmation
  gate/retry/cancel-unconfirmed UI (вне scope, см. POS-88).
- **Исправлено в этой сессии:** форма создания стола в Cloud UI
  (`cloud-ui-g/src/features/floor/TablesPanel.tsx`, `floorForms.ts`) теперь собирает
  `section_id` (было исправлено кодом ещё до этой сессии) — подтверждено живьём: создание
  стола через Playwright возвращает 201, регресс не воспроизвёлся.
- **Найдено и исправлено в этой сессии:** экран Printers (POS-83) в Cloud UI — create/edit
  прошли штатно, но деактивация (`DELETE /api/v1/printers/{id}`) ловила CORS-ошибку в
  браузере (`Method DELETE is not allowed by Access-Control-Allow-Methods in preflight
  response`). Причина — `localCORS` middleware в
  `cloud-backend/internal/cloudsync/api/router.go` не включал `DELETE` в
  `Access-Control-Allow-Methods` (только `GET, POST, PATCH, PUT, OPTIONS`), хотя backend route
  `DELETE /printers/{id}` (`cloud-backend/internal/masterdata/api/router.go`) существовал и
  работал. Исправлено добавлением `DELETE` в список методов; деактивация подтверждена живьём
  (200 OK).
- `POS-88` (Edge print_routes/targets UI) и `POS-102` (cloud-ui-g точки продаж/секции) не
  реализованы — переформулированы/заведены, но не начаты.
- Физический hardware acceptance с реальным ESC/POS принтером под новой моделью маршрутизации
  не переподтверждался в этой итерации (последнее подтверждение — Wave 3/POS-64 до POS-86).
- Windows-инсталлятор (`scripts/build-pos-edge-installer.ps1`) не собирался в этой сессии —
  на машине нет `npm`/Node.js и NSIS (`makensis`) в PATH, только Go и Docker Desktop. Baseline
  конфиг `pos-backend/config/pos-edge.windows.json` (используемый инсталлятором как
  `config\pos-edge.install.json`) проверен по git-истории: фикс `POS_PRINT_WORKER_ENABLED:
  true` и `MH_POS_VERSION: 0.1.11` подтверждён в коммите `2d810f5`, но сам инсталлятор с этим
  фиксом ещё ни разу не собирался и не запускался как установленный сервис.
- **Побочная находка (не исправлено, вне scope POS-86):** в `pos-ui-g/src/context/POSContext.tsx`
  `refreshCurrentPrechecks` (используется в отдельном `useEffect` без try/catch на строке
  ~514) может выбрасывать unhandled promise rejection (`ApiError: INVALID_RESPONSE`) при
  фоновом poll каждые 2.5с после создания/оплаты заказа — воспроизведено многократно в
  browser console во время order→precheck→payment прогона. Экран при этом не падает и не
  теряет данные, это чисто console-шум, но стоит завести отдельную задачу на разбор точной
  причины несовпадения response/schema и на добавление `.catch(handleError)` по аналогии с
  соседними `refresh*`-функциями.

## Промпт: полная проверка и отладка POS-86 на чистом стеке

Статус: **не выполнялось** ни в основной, ни в доводочной сессии POS-86 — обе ограничились
`go test`/`go vet`/`git diff --check` и построчным код-ревью. Ни живой докер-стек, ни
Playwright, ни фактический прогон `seed-dev-system.py` не запускались. Следующая сессия
должна закрыть именно этот разрыв.

```text
Проверь и отладь реализацию POS-86 (жёсткая маршрутизация печати, Cloud-owned точки продаж/
секции/столы, print-confirmation gate, cancel-unconfirmed) на ПОЛНОСТЬЮ ЧИСТОМ докер-стеке
в репозитории /home/master/repos/myhoreca-pos. Код и миграции уже реализованы (задача POS-86
в Plane сейчас в Review) — цель не дописать функциональность, а найти и исправить всё, что
не работает в реальном end-to-end окружении, чего не показывают unit/integration тесты на
SQLite in-memory/fixture.

Перед стартом прочитай: docs/backend/EDGE-PRINT-ROUTING-SPEC.md целиком (особенно "Оставшиеся
риски"), docs/project-management/ALPHA-LAUNCH-CODEGEN-ITERATIONS.md раздел "Итерация 8g",
итоговый комментарий на POS-86 в Plane (там список того, что уже нашли и починили в прошлый
раз — не переоткрывай уже закрытые пункты, если не найдёшь обратного).

## Шаг 1 — базовая сверка кода (быстро, до поднятия стека)

cd pos-backend && go mod tidy && go test ./... && go vet ./...
cd cloud-backend && go mod tidy && go test ./... && go vet ./...
git diff --check

Если что-то из этого красное — почему-то регрессировало с прошлой сессии, разберись прежде
чем идти дальше.

## Шаг 2 — чистый докер-стек

Подними docker-compose.local.yml с нуля (пересоздай volumes, чтобы БД были гарантированно
пустыми — это pre-pilot окружение, старые данные не защищены):

docker compose -f docker-compose.local.yml down -v
docker compose -f docker-compose.local.yml up -d --build
# дождись healthy для cloud-postgres и cloud-clickhouse (healthcheck уже настроен в compose)

Порты: cloud-api :8090, pos-edge :8080, license-api :8095, postgres :5432, clickhouse :8123/:9000
(см. CLAUDE.md). Пройдись по /health на каждом сервисе прежде чем продолжать.

## Шаг 3 — живой прогон seed-скрипта (обязательно, это и есть главный пробел)

cd scripts
python3 seed-dev-system.py --run-minimal-flow --run-kitchen-process-smoke

Это первый реальный прогон новой логики (создание sales_point сразу после ресторана,
section_id в create table, sales_point_id в open cash session) через настоящий HTTP/Cloud/Edge
контур, а не через Go-тесты с моками. Ожидай найти расхождения — например: поле называется
не так, как в Go DTO; порядок операций не совпадает с тем, что реально требует backend;
`verify_pos_ready` не дожидается прихода стримов `sales_points`/`restaurant_sections`, если
её не расширяли под них (проверь).

Если скрипт падает — не обходи ошибку, а прочитай actual response body, сопоставь с Go
кодом (particularly cloud-backend/internal/masterdata/app/service.go CreateSalesPoint/
CreateRestaurantSection/CreateTable, pos-backend/internal/pos/app/cash/service.go
OpenCashSession) и почини либо скрипт, либо (если найдёшь баг) сам backend. Прогоняй заново
до зелёного полного смоука, не только до первой прошедшей команды.

Дополнительно проверь end-to-end физическую логику печати на этом стеке: после
`--run-minimal-flow` найди созданные `print_jobs`/`print_job_targets` через
`GET /api/v1/print/jobs` на pos-edge (с валидным manager PIN сессией) и убедись, что
`check_nonfiscal`-job и `ticket`-job реально дошли до какого-то терминального статуса (даже
если `failed` из-за отсутствия реального принтера — статус должен быть `PRINT_ROUTING_NOT_CONFIGURED`
только если print_routes не настроены сидом, либо реальная попытка отправки, если настроены).
Если seed не создаёт print_routes — это отдельный пробел, который стоит закрыть (сид должен
создать хотя бы `check_nonfiscal→sales_point` и `precheck`/`ticket`→бутстрэп-секция маршруты
через POST /api/v1/print-routing/routes на pos-edge, иначе print-confirmation gate никогда не
подтвердится и это будет мешать сквозной проверке).

Проверь также `POST /api/v1/orders/{id}/cancel-unconfirmed` и
`POST /api/v1/checks/{id}/print-confirmation/retry` вручную через curl на этом стеке (не
только в Go-тестах) хотя бы один раз каждый, чтобы исключить расхождение между тестовым http
клиентом и реальным сервером (заголовки, content-type, RBAC middleware).

## Шаг 4 — Playwright: ручная проверка UI (используй Playwright MCP)

Подними pos-ui-g и cloud-ui-g dev-сервера, указав на этот же докер-стек (переменные окружения
VITE_CLOUD_API_BASE / соответствующие для pos-ui-g — проверь актуальные имена в vite.config/.env
примерах каждого проекта). Через Playwright MCP пройди руками:

1. Cloud UI: залогинься, открой экран Floor/Tables (`TablesPanel.tsx`) для ресторана,
   попробуй создать новый стол. ОЖИДАЕТСЯ РЕГРЕССИЯ: форма не собирает `section_id`, backend
   должен вернуть 400. Если воспроизвелось — почини форму (добавь выбор секции, подставляй
   дефолтную/бутстрэп секцию как значение по умолчанию) и артефакт зафиксируй в
   CURRENT-FUNCTIONAL-STATE.md/этом файле как исправленный, а не как оставшийся риск.
   Если НЕ воспроизвелось — разберись почему (может, где-то есть fallback, о котором мы не
   знали) и опиши фактическое поведение.
2. Cloud UI: проверь существующий экран Printers (POS-83) — регрессий по идее быть не должно,
   но пройди create/edit/deactivate руками один раз для контроля.
3. POS UI: залогинься PIN'ом кассира/менеджера, попробуй открыть кассовую смену через обычный
   экран (`POSCashSection.tsx`). ОЖИДАЕТСЯ РЕГРЕССИЯ: запрос падает без `sales_point_id`.
   Зафиксируй точный текст ошибки, которую видит кассир — должна быть safe error (не raw Go
   error/500), если это не так — почини error handling на этом конкретном пути. Реализация
   самого выбора точки продаж в UI — вне scope (POS-88 будущая задача), но безопасный,
   понятный отказ — не вне scope, это часть AGENTS.md error contract.
4. Если получится обойти пункт 3 (например, открыть смену через прямой API-вызов с
   sales_point_id, а потом продолжить в UI) — пройди обычный флоу: заказ → precheck → оплата
   → убедись что UI не падает, даже если печать физически не настроена (ошибка печати не
   должна ломать экран оплаты кассира).

## Шаг 5 — привести в порядок

- Почини все найденные регрессии (не только задокументируй) — особенно cloud-ui-g
  TablesPanel, если она правда ломается, это уже отгруженный клиенту экран.
- Обнови docs/backend/EDGE-PRINT-ROUTING-SPEC.md ("Оставшиеся риски") и
  docs/CURRENT-FUNCTIONAL-STATE.md под то, что реально проверено/исправлено — не оставляй
  пункты "не проверено" висеть, если ты их только что проверил.
- Прогони `cd pos-ui-g && npm run build` и `cd cloud-ui-g && npm run build`, если правил
  frontend.
- Останови докер-стек: `docker compose -f docker-compose.local.yml down -v` (если это чисто
  проверочный прогон и volumes не нужно сохранять — уточни у пользователя перед `down -v`,
  если стек мог использоваться и для чего-то ещё).

## Отчёт

Оставь итоговый комментарий на POS-86 в Plane (не Done, задача уже в Review — если что-то
принципиально сломано, верни в In Progress с точным списком; если всё подтвердилось рабочим —
отдельным комментарием подтверди live-прохождение smoke и закрой пункт "manual-validation" в
описании риска). Явно укажи: что проверено вживую, что исправлено, что осталось нерабочим и
почему (например, если POS UI осознанно не чинится в этом прогоне).
```

## Следующие промпты

### POS-87

```text
Используй универсальный промпт для POS-87 после POS-86 (Review/Done).

Фокус: Edge -> Cloud sync для printer override audit и Cloud effective read model.
Edge-side изменение схемы печати (print_routes) применяется локально сразу, пишет audit и
outbox event. Cloud принимает событие как факт/проекцию, не как proposal, хранит
видимость effective routes/override audit и отдает bounded read model для оператора.

Не менять локальный routing алгоритм сверх контракта POS-86. Не добавлять POS UI.

Проверки: cd pos-backend && go mod tidy && go test ./...; cd cloud-backend &&
go mod tidy && go test ./...; git diff --check.
Документация: sync ownership, CLOUD-BACKEND-SPEC.md, EDGE-PRINT-ROUTING-SPEC.md.
```

### POS-88 (переформулирована после POS-86)

```text
Используй универсальный промпт для POS-88 в репозитории /home/master/repos/myhoreca-pos.

Фокус: POS UI settings ТОЛЬКО для print_routes (назначение уже синхронизированных с Cloud
принтеров/точек продаж/секций на печать по document_type) и очереди print_job_targets
(статус, retry отдельного target, диагностика последней ошибки). Не включать
создание/редактирование/удаление точек продаж, секций или столов — это Cloud-owned,
управляется через cloud-ui-g (POS-102), Edge их только читает через
GET /api/v1/print-routing/sales-points и /sections. Все пользовательские строки через
vue-i18n. Backend остаётся authoritative; UI visibility не является security boundary.

Нужны состояния loading/empty/error, safe error banner, retry отдельного target,
диагностика последней ошибки и ручной checklist для физического принтера.

Проверки: cd pos-ui-g && npm install && npm run build; при возможности
Playwright smoke settings flow; git diff --check.
Документация: POS-UI-SPEC.md, EDGE-PRINT-ROUTING-SPEC.md, CURRENT-FUNCTIONAL-STATE.md.
```

### POS-89

```text
Используй универсальный промпт для POS-89 после POS-86 (Done), POS-87 и POS-88.

Фокус: exhibition smoke sales point + section printer routing. Smoke должен
создать/проверить точку продаж с cash printer, секцию зала с precheck printer, кухонную
секцию с kitchen_service printer, выполнить bounded sale/precheck/print flow и подтвердить
target-level statuses и print_confirmed_at.

Перед стартом прогнать live `scripts/seed-dev-system.py` против поднятого стека (не было
сделано в POS-86-сессии) — если найдутся расхождения, поправить сначала seed, потом smoke.

Если физический принтер недоступен, не заявлять hardware acceptance: оставить
manual-validation checklist с host/port/model/CPL и ожидаемыми документами.

Проверки: профильные backend/UI checks, seed/smoke, git diff --check.
Документация: CURRENT-FUNCTIONAL-STATE.md и go/no-go evidence.
```
