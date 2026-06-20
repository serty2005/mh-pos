# Внешние лицензии и module gates

## Статус

Реализовано сейчас:

- внешний `license-server` является authority для versioned snapshot по паре `tenant_id/server_id`;
- snapshot содержит monotonic `version`, `active|revoked`, extensible `entitlements`, `issued_at` и `expires_at`;
- provider update `PUT /api/v1/entitlements/{tenant_id}/{server_id}` требует `Authorization: Bearer <LICENSE_ADMIN_TOKEN>`;
- runtime read `GET /api/v1/entitlements/{tenant_id}/{server_id}` не возвращает credentials или provider token;
- Cloud и POS Edge получают snapshot по HTTP, работают fail-closed и возвращают `LICENSE_ENTITLEMENT_REQUIRED` (`403`) либо `LICENSE_AUTHORITY_UNAVAILABLE` (`503`);
- stale grace применяется только при недоступности authority, задается deployment config и не продлевает snapshot при успешном ответе `expired`/`revoked`;
- выключение entitlement не удаляет halls, tables, recipes, kitchen, warehouse или financial data.

Canonical IDs: `table-mode`, `telegram-worker`, `kitchen-space`, `waiter-space`, `checker-flow`, `warehouse-mode`. Дополнительные IDs допускают lowercase letters, digits, `-` и `.`.

## Базовая касса и платные модули

Основной кассовый поток является базовой частью продукта и не лицензируется:

- локальный вход, смены сотрудника и кассовая смена устройства;
- локальное меню, если оно уже есть на Edge;
- создание заказа кассиром;
- добавление/изменение/void строк;
- `Order -> Precheck -> Payment -> Check`;
- повторная печать предчека/чека из сохраненного snapshot;
- локальный financial operation ledger для отмен и возвратов;
- локальный outbox базовых финансовых фактов, если Edge подключен к Cloud.

Это решение нужно для текущего MVP и для post-MVP бесплатного автономного режима. После MVP POS Edge должен уметь работать полностью локально без внешнего Cloud: владелец локально создает простые позиции меню для этого Edge, кассир продает их через базовый поток, данные остаются в локальной SQLite и локальных backup/archive. Покупка лицензии подключает внешний Cloud/tenant management, автоматическую доставку master data, дополнительные роли, рабочие пространства и workers. Бесплатный автономный режим не получает Cloud-owned tenant catalog, межустройственную доставку, Cloud analytics, warehouse engine, waiter mobile, KDS/advanced kitchen, Telegram, checker или другие лицензируемые поверхности.

`waiter-space` лицензирует не сам факт заказа, а отдельный официантский доступ: mobile-first waiter UI/API, waiter-only navigation, waiter worker/facade и связанные события. Текущие order/precheck HTTP routes являются shared cashier backend surface, поэтому их нельзя закрывать `waiter-space` целиком: это сломает нелицензируемую базовую кассу. Enforcement для `waiter-space` должен появляться там, где backend однозначно видит официантский контекст, например через отдельный waiter API surface или backend-owned waiter command facade. UI route `/pos/waiter` и любые frontend headers сами по себе не являются security boundary.

Фактические gates существуют только для реализованных runtime surfaces:

- `table-mode`: Cloud floor routes, Edge halls/tables, direct floor ingest и доставка `floor` через sync exchange;
- `kitchen-space`: Cloud recipe routes, Edge kitchen routes, direct recipe ingest и доставка `recipes` через sync exchange;
- `warehouse-mode`: Cloud inventory master-data routes, Edge stock/count/write-off/production routes, direct inventory ingest, доставка `inventory_reference` через sync exchange и Cloud inventory worker.

При `denied` или недоступном authority лицензируемый Cloud package не включается в exchange response, поэтому Edge checkpoint этого stream не продвигается. После восстановления entitlement очередной exchange доставляет актуальную версию без удаления сохраненных данных.

Архитектурное правило для Edge -> Cloud: batch, Cloud receiver и Cloud workers не должны формировать данные для выключенного модуля. Базовые cashier financial facts могут синхронизироваться как часть подключенного Cloud-контура, но module-owned события должны отсеиваться по entitlement до отправки batch или до обработки worker:

- `kitchen-space` владеет KDS/kitchen operational events и kitchen proposal surfaces;
- `warehouse-mode` владеет receipt/count/write-off/production и Cloud inventory worker processing;
- `waiter-space` владеет waiter-only commands/events после появления отдельного backend-discriminated waiter surface;
- `telegram-worker` и `checker-flow` владеют своими worker queues и routes после реализации.

Если модуль выключен, Edge не должен создавать новые module-owned commands через закрытые routes, sync sender не должен включать module-owned outbox rows в отправляемый batch, а Cloud не должен создавать module projections/worker rows из таких событий. Уже сохраненные данные не удаляются; после восстановления лицензии обработка продолжается с текущего безопасного checkpoint.

Запланировано далее: `telegram-worker`, отдельный waiter module и `checker-flow` получат gates одновременно с появлением соответствующих workers/routes. Дополнительно требуется довести module-owned outbox filtering для Edge -> Cloud batch и прямые Cloud provisioning package routes до того же entitlement mapping. Фиктивные endpoints не создаются.

## Хранение и аудит

`license-server` программно создает SQLite table `entitlement_snapshots`; primary key — `(tenant_id, server_id)`. Provider update разрешает только версию выше сохраненной и пишет structured audit без token или entitlement payload.

Локальный seed через HTTP создает active snapshots для `cloud-local` и `edge-local`. Production provider token должен передаваться secret management, а не храниться в tracked config.
