# Внешнее лицензирование и module entitlements

Статус: canonical contract для лицензирования RMS-POS. Остальные документы должны ссылаться на этот файл и не дублировать детальные правила.

## Назначение

Лицензирование собирает клиентское решение из общей кодовой базы RMS-POS без fork, exhibition-only runtime или hardcoded product profile. Выставка, ресторанный пилот и локальная касса отличаются tenant/restaurant data, deployment profile и entitlement snapshot, а не отдельным кодом.

Базовый cashier runtime является бесплатным и нелицензируемым. Product licenses включают только дополнительные Cloud/Edge возможности, рабочие пространства, workers, delivery streams и интеграции.

## Реализовано сейчас

- внешний `license-server` является authority для versioned snapshot по паре `tenant_id/server_id`;
- snapshot содержит monotonic `version`, `active|revoked`, extensible `entitlements`, `issued_at` и `expires_at`;
- operator update `PUT /api/v1/entitlements/{tenant_id}/{server_id}` требует входа super-admin через login/password и HttpOnly session cookie;
- runtime read `GET /api/v1/entitlements/{tenant_id}/{server_id}` не возвращает credentials или operator session;
- License Server фиксирует подключившиеся `tenant_id/server_id` в списке connected servers, чтобы оператор выбирал сервер из списка и искал по `tenant_id`;
- Cloud и POS Edge получают snapshot по HTTP, работают fail-closed и возвращают `LICENSE_ENTITLEMENT_REQUIRED` (`403`) либо `LICENSE_AUTHORITY_UNAVAILABLE` (`503`);
- stale grace применяется только при недоступности authority, задается deployment config и не продлевает snapshot при успешном ответе `expired`/`revoked`;
- выключение entitlement не удаляет halls, tables, recipes, kitchen, warehouse, ticket или financial data;
- backend gates реализованы для существующих `cloud-subscription`, `table-mode`, `kitchen-space`, `warehouse-mode` и `ticket-mode` surfaces;
- License Server operator page в стандартном сценарии управляет canonical modules через toggles/presets, raw JSON отделен в advanced support mode;
- Cloud `sync/exchange` не отдает Edge package streams без `cloud-subscription`, а module streams `floor`, `recipes` и `inventory_reference` дополнительно требуют `table-mode`, `kitchen-space` и `warehouse-mode`;
- Edge -> Cloud sender и Cloud receiver не отправляют/не принимают module-owned events выключенных `kitchen-space`, `warehouse-mode` и `ticket-mode`;
- Cloud inventory worker запускается только при доступном `warehouse-mode`;
- POS UI и Cloud UI скрывают часть navigation по текущему snapshot, но это только UX-слой.

## Канонические entitlement IDs

Идентификаторы лицензий являются machine-readable contract и пишутся в lowercase с `-`:

| Entitlement ID | Назначение | Статус runtime |
| --- | --- | --- |
| `cloud-subscription` | Tenant-level Cloud: Cloud backoffice, tenant management, Cloud-owned catalog/roles/employees, automatic Cloud -> Edge delivery, Cloud analytics/OLAP/dashboard, multi-Edge/device limits, receipt/printer/template management и остальные Cloud-сервисы, которые не выделены в отдельный module ID. | реализовано сейчас для текущих tenant-level Cloud HTTP surfaces и `receipt_templates`/`printers` delivery |
| `table-mode` | Залы, столы, table-bound flow, отдельный table/precheck UI, Cloud floor settings и delivery stream `floor`. | реализовано сейчас |
| `kitchen-space` | KDS/kitchen UI, kitchen routes/actions, recipe delivery, kitchen events и proposals. | реализовано сейчас для существующих kitchen routes |
| `warehouse-mode` | Складские формы, inventory reference delivery, stock receipt/count/write-off/production, Cloud Inventory Worker, stock ledger/balances/costing/recalculation. | реализовано сейчас частично |
| `waiter-space` | Отдельный mobile-first официантский доступ поверх backend-recognized waiter surface. | запланировано далее для backend-discriminated waiter facade |
| `telegram-worker` | Telegram settings/routes/worker и автоматическая отправка отчетов. | запланировано далее |
| `ticket-mode` | QR-enabled service items, ticket issuance, ticket templates/printing, checker enrollment, ticket lookup/confirm/revoke/use runtime, scanner/manual code input и checker reporting. | реализовано сейчас для ticket issuance, ticket read/reprint route gates и Edge -> Cloud `TicketIssued`; checker runtime запланирован далее |

Это полный согласованный на текущий момент список отдельно лицензируемых product modules. Новые module IDs добавляются только через обновление этого раздела, тесты, seed/smoke и профильную задачу Plane.

Дополнительные IDs допускают lowercase letters, digits, `-` и `.`. Underscore IDs вроде `table_mode`, `stock_flow`, `qr_flow`, `kitchen_flow`, `analytic_mode` не являются canonical contract для текущего продукта: текущий валидатор License Server их не принимает. Если продукту понадобятся marketing aliases, они должны быть display metadata поверх canonical IDs, а не вторым runtime contract.

## Термины для владельца продукта

- Контрагент — юридическое лицо или клиент, который платит за обслуживание. В runtime contract это `tenant_id`.
- Сервер — конкретный Cloud или POS Edge runtime, на который распространяется license snapshot. В runtime contract это `server_id`.
- Один контрагент может владеть несколькими серверами: например production Cloud, демонстрационный Edge и отдельный Edge на площадке выставки.
- Один сервер относится только к одному контрагенту. Поэтому License Server хранит snapshot по паре `tenant_id/server_id`, а не только по одному server id.
- Entitlement snapshot — текущий набор включенных лицензируемых модулей для конкретного сервера конкретного контрагента.

## Продуктовые пакеты

Product owner может продавать пакеты, которые включают один или несколько canonical IDs. Пакет не является runtime entitlement сам по себе, пока он не записан в snapshot как concrete IDs.

| Пакет | Canonical IDs |
| --- | --- |
| Базовая касса | нет лицензируемых IDs |
| Tenant Cloud | `cloud-subscription` |
| Работа с залом | `table-mode` |
| Кухня | `kitchen-space` |
| Склад | `warehouse-mode` |
| Официанты | `waiter-space` |
| Telegram-отчеты | `telegram-worker` |
| Билеты и проверка | `ticket-mode` |
| Полный ресторанный пилот | `cloud-subscription`, `table-mode`, `kitchen-space`, `warehouse-mode`, `waiter-space`, затем `telegram-worker`/`ticket-mode` по scope |

## Базовая бесплатная касса

Основной кассовый поток является базовой частью продукта и не лицензируется:

- локальный PIN-вход и backend session;
- личные смены сотрудников и кассовые смены устройства;
- локальное меню, если оно уже есть на Edge;
- создание заказа кассиром;
- добавление, изменение и void строк;
- `Order -> Precheck -> Payment -> Check`;
- повторная печать предчека, итогового чека и сохраненных snapshots;
- локальный financial operation ledger для отмен и возвратов;
- локальный outbox базовых финансовых фактов, если Edge подключен к Cloud.

После MVP POS Edge должен уметь работать полностью локально без внешнего Cloud: владелец локально создает простые позиции меню для этого Edge, кассир продает их через базовый поток, данные остаются в локальной SQLite и локальных backup/archive. Покупка `cloud-subscription` подключает внешний Cloud/tenant management, автоматическую доставку master data, Cloud analytics и остальные tenant-level Cloud services. Остальные рабочие пространства, workers и ticket/checker возможности подключаются своими module IDs.

## Сборка клиентского решения

Клиентский deployment собирается так:

1. Поставщик выбирает deployment profile: автономный Edge, Cloud-connected Edge, выставочный tenant или полный ресторанный пилот.
2. Cloud/License Server получают tenant/server identity.
3. License Server хранит active snapshot для `cloud-local`/production Cloud server и для каждого Edge server id.
4. Snapshot включает только canonical IDs, которые куплены или временно выданы для пилота.
5. UI читает `/api/v1/license/entitlements` и скрывает недоступные разделы.
6. Backend, workers и sync delivery проверяют те же IDs как security boundary.
7. При отключении лицензии новые module-owned commands/events/workers блокируются, но сохраненные данные остаются в storage.

## Правила enforcement

UI visibility является только UX-слоем. Security boundary находится в Cloud backend, POS Edge backend, workers и sync delivery.

| Boundary | Требование |
| --- | --- |
| Cloud routes | Module-owned routes возвращают safe `403 LICENSE_ENTITLEMENT_REQUIRED` или `503 LICENSE_AUTHORITY_UNAVAILABLE`. |
| POS Edge routes | Module-owned routes/commands fail-closed; базовый cashier flow не блокируется product module gates. |
| Cloud -> Edge delivery | Packages выключенных module streams не включаются в exchange response, checkpoint stream не продвигается. |
| Edge -> Cloud batch | Module-owned events выключенного модуля не попадают в отправляемый batch; сохраненные outbox rows остаются retryable и не удаляются. |
| Cloud receiver/workers | Cloud reject-ит module-owned events выключенного модуля item-level ACK и не создает module projections, stock documents, ledger rows или worker state. |
| UI | Разделы скрываются или переводятся в понятный blocked/empty state без raw backend details. |

При `denied` или недоступном authority лицензируемый Cloud package не включается в exchange response. После восстановления entitlement очередной exchange доставляет актуальную версию без удаления сохраненных данных.

## Владение модулей

- `table-mode` владеет halls/tables, table-bound navigation, floor settings и stream `floor`.
- `kitchen-space` владеет KDS/kitchen operational events, kitchen routes, recipe read/proposals и stream `recipes`.
- `warehouse-mode` владеет receipt/count/write-off/production, inventory reference stream, Cloud Inventory Worker, stock ledger/balances/costing/recalculation.
- `waiter-space` будет владеть waiter-only commands/events после появления backend-discriminated waiter surface. Текущие shared order/precheck/payment routes остаются базовой кассой.
- `telegram-worker` будет владеть Telegram settings, queues, worker и delivery audit.
- `ticket-mode` владеет QR-enabled service flags, ticket units, ticket templates/printing, checker enrollment, lookup/confirm/revoke/use commands и checker reporting.
- `cloud-subscription` владеет tenant-level Cloud services: Cloud backoffice, tenant management, Cloud-owned catalog/roles/employees, automatic delivery, Cloud analytics/OLAP/dashboard, multi-Edge/device limits, receipt/printer/template management и будущие Cloud-only products, пока для них не выделен отдельный module ID.

## `waiter-space` boundary

`waiter-space` лицензирует не сам факт заказа, а отдельный официантский доступ: mobile-first waiter UI/API, waiter-only navigation, waiter worker/facade и связанные события. Текущие order/precheck HTTP routes являются shared cashier backend surface, поэтому их нельзя закрывать `waiter-space` целиком: это сломает нелицензируемую базовую кассу.

Enforcement для `waiter-space` должен появляться там, где backend однозначно видит официантский контекст, например через отдельный waiter API surface или backend-owned waiter command facade. UI route `/pos/waiter` и frontend headers сами по себе не являются security boundary.

## `table-mode` off

Без `table-mode` UI не показывает залы, столы и отдельный precheck/table flow. Базовая продажа через стойку должна сохранять общую финансовую модель: backend использует системный restaurant hall/table или другой backend-owned internal context и автоматически выпускает authoritative precheck перед payment. Это не создает отдельную order model и не открывает клиенту нелицензированные halls/tables/precheck surfaces.

## `ticket-mode`

`ticket-mode` объединяет QR-ticket-checker в один лицензируемый модуль. Модуль включает:

- QR-enabled service item controls;
- `single_unit_per_line` и ticket validity policy;
- ticket unit issuance после оплаты;
- ticket templates и ticket printing/reprint;
- checker enrollment;
- QR lookup, confirm, revoke/use и one-use guard;
- scanner/manual code input;
- checker reporting.

Первый выставочный запуск реализует issuance/printing без checker runtime. Post-deploy checker cycle расширяет тот же `ticket-mode`, а не вводит отдельный `checker-flow`.

## `cloud-subscription`

`cloud-subscription` является отдельной tenant-level лицензией на Cloud-контур. Все дополнительные продукты, которые не выделены в отдельный module ID, входят в эту подписку:

- Cloud backoffice и guided setup;
- tenant roles/employees/catalog и restaurant menu overrides;
- automatic Cloud -> Edge delivery;
- Cloud analytics, OLAP/dashboard и bounded reporting;
- multi-Edge/device limits;
- receipt templates, printer management и Cloud-owned print configuration;
- Cloud-only operational screens и будущие Cloud-сервисы до отдельного product decision.

Бесплатный автономный Edge не требует `cloud-subscription` и не получает Cloud-owned tenant services.

## Stale grace

Stale grace задается поставщиком в deployment config сервера (`LICENSE_STALE_GRACE_SECONDS`) и недоступен клиенту в UI. Он применяется только когда authority временно недоступен, и только к последнему валидному active snapshot. Успешный ответ authority со статусом `revoked`, истекшим сроком или выключенным entitlement всегда блокирует module-owned functionality.

## Хранение и аудит

`license-server` программно создает SQLite tables `entitlement_snapshots`, `license_servers`, `admin_users` и `admin_sessions`; primary key snapshot — `(tenant_id, server_id)`. Operator update разрешает только версию выше сохраненной и пишет structured audit без session cookie, password или entitlement payload. Перед startup migration сервер делает backup SQLite `.db/.db-wal/.db-shm`, поэтому обновление binary/config не требует сброса данных.

Первый super-admin задается стартовым конфигом `LICENSE_SUPER_ADMIN_LOGIN` и `LICENSE_SUPER_ADMIN_PASSWORD`; password хранится в SQLite как PBKDF2-SHA256 hash с salt. Дальнейшие operator users запланированы далее, schema уже не привязана к одному пользователю.

## Запланировано далее

- добавить explicit gates для будущих `telegram-worker` и backend-discriminated `waiter-space` routes/workers вместе с их runtime;
- расширить smoke: enabled, denied, revoked, authority unavailable, stale grace и no data deletion для каждого реализованного module boundary.
