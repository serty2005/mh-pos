# Альфа-пилот продаж и печати билетов на выставках

Статус: требования первого выставочного запуска согласованы; QR-проверка билетов перенесена в следующий post-deploy цикл.

Дата актуализации: 2026-06-30.

Документ фиксирует выставки как частный случай общей RMS-POS платформы. Отдельный клиентский fork, exhibition-only runtime и hardcoded product profile запрещены. Конечный функционал Cloud и Edge собирается лицензиями.

Фактическая готовность ведется в `ROADMAP.md`, `docs/CURRENT-FUNCTIONAL-STATE.md` и Plane. Код и тесты остаются источником истины для реализованного runtime.

## 1. Управленческое резюме

| Вопрос | Решение первого запуска |
| --- | --- |
| Что продаем | Билет как catalog service через общий поток `Order -> Precheck -> Payment -> Check`. |
| Как моделируем клиента | Каждая выставка является обычным restaurant внутри tenant. |
| Каталог и меню | Catalog items принадлежат tenant; ресторан собирает собственное меню и задает название, цену, тег, налог и папку. |
| Билет | Одна единица QR-enabled service создает отдельный ticket с уникальным номером и печатным QR. |
| QR в первом запуске | Выпуск и печать QR обязательны; lookup, scan, confirm, revoke и checker infrastructure перенесены после запуска. |
| Печать | Нефискальные чеки и билеты печатаются на реальном ESC/POS-принтере по стандартным версионируемым шаблонам. |
| Аналитика | Продажи видны на главной Cloud-бэкофиса по restaurant, категории билета, service, business date и смене. |
| Telegram | Отчеты отправляются по расписанию и/или после закрытия кассовой смены согласно настройке ресторана. |
| Сотрудники и роли | Роли и сотрудники принадлежат tenant; доступ к ресторанам задается employee memberships. |
| Лицензии | Базовая кассовая продажа не отключается лицензиями; License authority включает Cloud/Edge модули поверх нее, UI скрывает, а backend блокирует нелицензированный функционал. |

## 2. Главный сценарий приемки

1. Управляющий организацией создает tenant-level роли, сотрудников и catalog services.
2. Управляющий включает сотрудникам доступ к ресторанам; `organization.manage` автоматически охватывает все рестораны tenant.
3. Менеджер ресторана выбирает catalog service и создает menu item со своими названием, ценой, тегом, налогом и папкой.
4. Для услуги включаются `qr_confirmation_enabled`, validity policy и зависимый признак `single_unit_per_line`.
5. Поставщик назначает tenant лицензии для дополнительных модулей; базовая кассовая продажа остается доступной на Edge при наличии локальных данных.
6. После подключения Edge Cloud собирает актуальный разрешенный batch, а Edge забирает его при плановой синхронизации без ручной публикации.
7. Кассир входит по PIN в разрешенном ресторане, открывает личную и кассовую смены.
8. Кассир продает один или несколько билетов как услуги. Каждая единица остается отдельной order line quantity `1`; без `table-mode` checkout автоматически выпускает backend-authoritative precheck без отдельного экрана залов, столов и пречека.
9. После полной оплаты Edge закрывает check и один раз создает ticket unit с UUIDv7 и уникальным порядковым номером в рамках кассовой смены.
10. Print subsystem печатает нефискальный чек и отдельный билет на реальном ESC/POS-принтере.
11. Билет содержит QR, название из menu item услуги, дату продажи, срок действия и сменный порядковый номер.
12. Edge синхронизирует продажу и ticket issuance в Cloud с restaurant, service/category и shift dimensions.
13. Главная Cloud-бэкофиса показывает реальные KPI и bounded drill-down по ресторану и категории билета.
14. Telegram worker отправляет ресторанный отчет по расписанию и/или после закрытия кассовой смены.
15. Backup/restore, printer failure/retry, license stale grace и acceptance smoke проходят на первом хосте.

## 3. Фактическое состояние кода

### 3.1. Реализовано сейчас

- тип catalog item `service` и его участие в menu/order/payment/check flow;
- restaurant identity в Edge commands, checks и Cloud sync envelopes;
- PIN login, roles, permissions, personal shifts и cash sessions;
- immutable precheck/check snapshots и controlled reprint response;
- автоматическая Cloud -> Edge доставка без manual Publish: packages создаются после Edge assignment/pairing и обновляются после подтвержденных Cloud commits только для назначенных Edge;
- tenant-level roles/employees, employee restaurant memberships, `organization.manage` и authoritative restaurant scope enforcement;
- tenant-level catalog identity и restaurant menu overrides для name, price, tag, active tax, menu folder, availability и runtime status;
- `qr_confirmation_enabled`, auto-derived `single_unit_per_line`, validity modes `cash_session`/`business_date`/`absolute_date`, Cloud UI форма и POS Edge validation quantity > 1;
- ticket issuance после полной оплаты final check: одна immutable `ticket_units` единица на QR-enabled line, UUIDv7 ticket number, sequence внутри cash shift, QR payload `MHT1:<ticket_number>`, reprint с COPY marker без создания второй единицы;
- `TicketIssued` Edge -> Cloud принимается как accepted audit; operational sales projection/API остается `POS-40`;
- нефискальная physical ESC/POS печать `precheck` и `ticket` через Edge print queue/worker, Cloud-owned receipt templates и Cloud-owned printer config stream; clean-stack стенд 2026-06-28 отправил оба документа на TCP-принтер `10.25.1.201:9100`, оператор подтвердил выход двух чеков;
- Cloud route-backed `receipt-templates` editor с live SVG preview и `printers` management в `cloud-ui-g`;
- Edge outbox, Cloud event receive, PostgreSQL operational storage и async ClickHouse export;
- bounded OLAP foundation и Cloud dashboard shell;
- внешний License Server как authority для versioned entitlement snapshots, canonical module toggles/presets, stale grace и backend gates реализованных `cloud-subscription`/`table-mode`/`kitchen-space`/`warehouse-mode`/`ticket-mode` surfaces;
- Edge pairing через License Server.

### 3.2. Не реализовано и обязательно до запуска

- реальные sales/ticket projections и KPI главной Cloud-бэкофиса;
- restaurant-level Telegram settings, безопасный recipient onboarding и worker;
- production deployment smoke, backup/restore rehearsal и финальная hardware acceptance.

Остающийся print-risk: после исправления ESC/POS/SVG renderer для крупного `{f:double}` текста на 48CPL требуется повторное визуальное подтверждение ticket на реальном принтере. License gates для будущих waiter/telegram runtime добавляются вместе с соответствующими backend-discriminated routes/workers. Telegram runtime отсутствует.

## 4. Организация, рестораны и master data

Один развернутый сервер обслуживает один tenant организации. Внутри tenant создаются рестораны, включая выставки.

Tenant владеет:

- каталогом и catalog item identity;
- ролями и permission sets;
- сотрудниками и PIN credential lifecycle;
- лицензиями и entitlement snapshot.

Ресторан владеет:

- меню и menu item overrides;
- залами, столами, секциями и устройствами, если модуль включен;
- сменами, заказами, продажами и печатными заданиями;
- настройками Telegram-отчетов после реализации `POS-63`;
- restaurant-scoped analytics.

Текущие обязательные `restaurant_id` в catalog, roles и employees являются migration gap. До первого клиента active pre-pilot baseline меняется программно при startup согласно общей migration policy.

## 5. Каталог и ресторанное меню

Catalog item создается один раз на tenant level. Один item может использоваться в меню нескольких ресторанов.

Menu item принадлежит ресторану и ссылается на tenant catalog item. Ресторан задает:

- собственное отображаемое название;
- цену;
- тег;
- действующий налог;
- папку меню;
- availability и publication status.

Restaurant override не изменяет catalog item и другие меню. Продажа и отчет сохраняют оба идентификатора: tenant `catalog_item_id` и restaurant `menu_item_id`.

Изменение effective menu/catalog data автоматически становится доступно подключенным Edge на ближайшей плановой синхронизации. Без подключенных Edge Cloud не копит delivery packages. При первом подключении batch собирается из актуального tenant/restaurant state. Менеджер не видит и не выполняет Publish action.

Категория билета для аналитики должна иметь стабильную identity. До реализации нужно использовать явное поле/справочник, а не выводить категорию из названия, папки или произвольного тега.

Cloud UI целевой flow:

- раздел называется `Каталог и меню` и доступен до создания/выбора ресторана;
- без выбранного ресторана менеджер заранее заполняет общий tenant catalog: папки каталога, позиции, services и tags;
- после создания и выбора ресторана тот же раздел показывает tenant catalog с restaurant menu overlay;
- позиции catalog item, выставленные на продажу в выбранном ресторане, подсвечиваются как `выставлено на продажу` и показывают menu item overrides;
- `Только каталог` является режимом по умолчанию и строит иерархию по `catalog_folders`;
- `Только меню` доступен при выбранном ресторане и строит иерархию по restaurant menu categories;
- menu category является папкой ресторанного меню, а не папкой общего catalog;
- все create/edit формы для catalog folders/items, menu categories/items и tags открываются как modal dialogs.

## 6. Сотрудники, роли и доступ к ресторанам

Role и Employee являются tenant-level справочниками. Restaurant access хранится отдельными employee memberships.

Правила:

- сотрудник без `organization.manage` обязан иметь минимум одно active restaurant membership;
- login и restaurant-scoped command разрешены только для active membership;
- роль не включается отдельно для ресторана;
- `organization.manage` всегда дает доступ ко всем текущим и будущим ресторанам tenant;
- управляющий рестораном использует обычную роль и явные memberships;
- Cloud UI фильтры не заменяют backend authorization;
- публикация на Edge содержит только сотрудников, которым разрешен этот restaurant, плюс актуальный permission snapshot.

## 6.1. Базовая касса и лицензируемые рабочие пространства

Продажа билетов первого запуска идет через базовый cashier flow `Order -> Precheck -> Payment -> Check`. Этот flow не должен зависеть от `waiter-space`, `kitchen-space`, `warehouse-mode`, `telegram-worker` или post-deploy checker runtime внутри `ticket-mode`.

Лицензии включают только дополнительные Cloud/Edge возможности. Canonical module IDs, ownership и правила enforcement описаны в `docs/backend/LICENSE-ENTITLEMENTS.md`. Если модуль выключен, новые module-owned данные не создаются в Edge commands, Edge batch и Cloud workers. Уже сохраненные данные не удаляются. Базовые cashier financial facts остаются частью подключенного Cloud-контура.

## 7. QR-enabled service и ticket issuance

QR-поведение включается явным `qr_confirmation_enabled` у service. При его включении backend автоматически включает `single_unit_per_line`; пользователь не может отключить зависимый признак отдельно.

Если кассир добавляет несколько одинаковых билетов, backend создает отдельную order line quantity `1` для каждой единицы. Quantity mutation выше `1` отклоняется стабильной business error.

После полной оплаты и закрытия check Edge один раз создает ticket unit:

- `id` UUIDv7;
- tenant, restaurant, Edge, cash session, check, order и order line IDs;
- catalog item и menu item IDs;
- immutable name, sale date, timezone и resolved validity snapshot;
- уникальный ticket number;
- sequence number внутри кассовой смены;
- QR payload, построенный из уникального ticket number без PIN, token или payment-sensitive данных;
- print status и timestamps.

Повторная печать использует тот же ticket number и QR, помечается как копия и не создает новый ticket unit.

## 8. Срок действия билета

До запуска поддерживаются согласованные validity modes:

- `cash_session` — кассовая смена продажи;
- `business_date` — `business_date_local` продажи;
- `absolute_date` — одна заданная локальная календарная дата ресторана.

При выпуске ticket unit сохраняются immutable resolved validity и timezone. Изменение service/menu/restaurant settings не переписывает уже проданный билет.

## 9. Стандарт печати

Print subsystem является общим модулем RMS-POS и поддерживает типы `precheck`, `check`, `ticket` и `kitchen_order` без выставочного fork.

Требования первого запуска:

- нефискальная печать;
- ESC/POS по TCP/network;
- ESC/POS через USB-принтер, установленный в Windows;
- версионируемые стандартные шаблоны и typed document model;
- preview/test print только через реальный backend route;
- очередь, timeout, bounded retry, status и безопасная ошибка оператора;
- audit для initial print, retry и reprint;
- отсутствие auto-retry финансовой операции: повторяется только print job;
- реальный hardware smoke на целевых моделях.

Razor/XML-шаблоны iiko/Syrve используются как reference для состава документов. Их runtime engine и формат не становятся обязательной зависимостью.

Билет печатается отдельным документом на каждую ticket unit и содержит минимум:

- QR;
- ресторан и название услуги из immutable sale snapshot;
- дату продажи;
- срок действия;
- порядковый номер в рамках кассовой смены;
- признак копии при reprint.

## 10. Продажи и Cloud-аналитика

Каждая продажа сохраняется и синхронизируется с обязательным `restaurant_id`. Cloud строит operational projection до ClickHouse export, чтобы dashboard не зависел только от OLAP lag.

Главная Cloud-бэкофиса показывает:

- количество проданных билетов;
- gross revenue, refunds и net revenue в minor units;
- средний чек;
- разрез по restaurant;
- разрез по категории билета, catalog item и menu item;
- business date и cash shift;
- freshness и incomplete-data marker.

Агрегаты допускают bounded drill-down до check/order line/ticket unit/financial operation. `organization.manage` видит все рестораны tenant; остальные сотрудники — только memberships.

## 11. Telegram-отчеты

Настройки принадлежат ресторану и требуют лицензии `telegram-worker`.

Ресторан независимо включает:

- отправку по расписанию;
- отправку после закрытия кассовой смены;
- оба режима одновременно;
- timezone, schedule и recipients.

Telegram Bot API не гарантирует начало диалога по username. Canonical recipient — подтвержденный `chat_id`, полученный после bot onboarding; username может храниться только как display metadata.

Отчет содержит restaurant, период/смену, количество билетов и разрез по категориям, gross/refund/net amounts и freshness marker. Повторная доставка одного report occurrence идемпотентна. Bot token не возвращается в UI и не пишется в логи.

## 12. Внешнее лицензирование

License Server является внешним authority для tenant/server entitlements. Подробный контракт licensing catalog, canonical IDs, stale grace, `table-mode` off, UI visibility и backend/workers/sync enforcement находится в `docs/backend/LICENSE-ENTITLEMENTS.md`.

Для выставочного запуска важно продуктовое правило: базовая кассовая продажа не блокируется отсутствием дополнительных module entitlements, а выставка остается конфигурацией общей RMS-POS через tenant/restaurant data и entitlement snapshot.

## 13. Deployment и эксплуатация

Первый запуск выполняется на отдельном хосте. Обязательны:

- documented single-host topology;
- TLS, secrets и backup/restore;
- PostgreSQL, SQLite и printer configuration backup;
- health/readiness для Cloud, Edge, License Server, Telegram worker и print worker;
- RPO/RTO и owner восстановления;
- переносимость контейнеров в будущий общий Kubernetes-контур без смены доменной модели.

## 14. Go/No-Go первого запуска

До допуска клиента обязательны:

- tenant catalog и restaurant menu overrides — реализовано сейчас, требуется smoke на целевых данных;
- автоматическая Cloud -> Edge доставка без ручной публикации и без накопления packages до подключения Edge — реализовано сейчас, требуется go/no-go smoke;
- tenant roles/employees, memberships и `organization.manage` — реализовано сейчас, требуется go/no-go smoke;
- продажа QR-enabled service с `single_unit_per_line` — реализовано сейчас;
- неизменяемый ticket number, validity snapshot и QR — реализовано сейчас;
- реальная печать нефискального чека и билета через целевой ESC/POS TCP-принтер — реализовано сейчас; Windows USB и повторное визуальное подтверждение ticket после `{f:double}` fix остаются acceptance gate;
- безопасный retry/reprint без duplicate ticket;
- Cloud sales projections и dashboard reconciliation;
- Telegram schedule и cash-shift-close delivery;
- enforcement всех назначенных licenses на UI и backend — canonical flow `POS-95` в `Review`;
- stale grace/fail-closed smoke;
- backup/restore rehearsal и client acceptance comment в Plane.

QR lookup/confirm не является критерием первого запуска.

## 15. Следующий post-deploy цикл: QR-проверка

В следующий цикл после запуска переносятся:

- checker device enrollment и license binding;
- Cloud-hosted checker UI;
- scanner и manual code input;
- Cloud-Edge typed relay;
- ticket lookup, confirm, one-use guard и command-result idempotency;
- `TicketCodeChecked`, `TicketServiceUsed`, revoke/refund state integration;
- checker reporting и cross-Edge rules.

Выпущенные первым релизом ticket number и QR должны быть совместимы с последующей проверкой, но первый запуск не объявляет их проверяемыми в runtime.

## 16. Plane Development Map

Активный цикл первого запуска содержит:

- `POS-36` — requirements sync в `Review`; `POS-46` — финальная синхронизация Git/Plane/public page, `Ready`;
- `POS-61`/`POS-62`/`POS-65`/`POS-48`/`POS-52`/`POS-53`/`POS-42` — `Done`;
- `POS-64` — umbrella physical printing, `Ready`; дочерние `POS-68…74`, `POS-81…84` — `Done`, `POS-67`/`POS-80` — `Review`;
- `POS-40` — sales projections API, `Ready`; `POS-41` — главная Cloud-аналитика, `Specified`;
- `POS-63` — Telegram reports, `Ready`;
- `POS-43`/`POS-44`/`POS-45` — deployment, backup/restore и E2E go/no-go, `Specified`;
- `POS-91`/`POS-95` — найденные UI/license blockers из ручного аудита, `Review`;
- `POS-92`/`POS-93`/`POS-94` — Cloud raw forms, guided setup и accessibility cleanup, `Specified`.

QR checker, relay, usage/revoke events и device enrollment исключаются из активного цикла и переносятся в следующий post-deploy cycle без удаления истории задач.

## 17. Вне текущего объема

- QR lookup/confirm до первого запуска;
- fiscal device и PSP integration;
- generic report constructor;
- accounting/ERP integration;
- generic remote administration tunnel;
- Kubernetes operator/control plane;
- отдельный exhibition codebase или product fork.
