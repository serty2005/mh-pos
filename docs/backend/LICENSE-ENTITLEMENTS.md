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

Фактические gates существуют только для реализованных runtime surfaces:

- `table-mode`: Cloud floor routes, Edge halls/tables, direct floor ingest и доставка `floor` через sync exchange;
- `kitchen-space`: Cloud recipe routes, Edge kitchen routes, direct recipe ingest и доставка `recipes` через sync exchange;
- `warehouse-mode`: Cloud inventory master-data routes, Edge stock/count/write-off/production routes, direct inventory ingest, доставка `inventory_reference` через sync exchange и Cloud inventory worker.

При `denied` или недоступном authority лицензируемый Cloud package не включается в exchange response, поэтому Edge checkpoint этого stream не продвигается. После восстановления entitlement очередной exchange доставляет актуальную версию без удаления сохраненных данных.

Запланировано далее: `telegram-worker`, отдельный waiter module и `checker-flow` получат gates одновременно с появлением соответствующих workers/routes. Фиктивные endpoints не создаются.

## Хранение и аудит

`license-server` программно создает SQLite table `entitlement_snapshots`; primary key — `(tenant_id, server_id)`. Provider update разрешает только версию выше сохраненной и пишет structured audit без token или entitlement payload.

Локальный seed через HTTP создает active snapshots для `cloud-local` и `edge-local`. Production provider token должен передаваться secret management, а не храниться в tracked config.
