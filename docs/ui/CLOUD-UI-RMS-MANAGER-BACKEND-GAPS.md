# Cloud UI: gaps по референсу RMS Manager

## Назначение

Документ фиксирует функциональность из `docs/ui/myhoreca-rms-manager/`, которую нельзя переносить в `cloud-ui-g/` как runtime UI без новых Cloud backend контрактов.

Подробная спецификация требуемых Cloud routes: `docs/ui/CLOUD-UI-RMS-MANAGER-CLOUD-API-SPEC.md`.

## реализовано сейчас

- Выбор ресторана, CRUD ресторанов и route-backed разделы master-data доступны в `cloud-ui-g`.
- Каталог, меню, модификаторы, цены/налоги, роли/сотрудники, залы/столы, публикации master-data и базовая привязка Edge используют существующие Cloud API.
- Раздельный пользовательский опыт Catalog/Menu в `cloud-ui-g` заменен единым разделом `Каталог и меню`; tenant catalog доступен без выбранного ресторана, а выбранный ресторан добавляет menu overlay.
- Menu categories имеют backend lifecycle для режима `Только меню`: `GET /master-data/menu/categories?restaurant_id=...`, `PATCH /master-data/menu/categories/{id}` и `POST /master-data/menu/categories/{id}/archive`.
- Экран Edge sync различает server-owned pending устройства и restaurant-owned assigned Edge nodes через `GET /devices/unassigned` и `GET /restaurants/{restaurant_id}/devices`; sync log фильтруется по выбранному устройству.
- UI не копирует моковые симуляторы продаж, терминалов, склада и stop-list из референса.

## запланировано далее

- Catalog/Menu:
  - расширить toolbar единого раздела `Каталог и меню` фильтрами, tags filter, настройками вида и delivery status;
  - подсветка `выставлено на продажу` цветом ресторана остается хотелкой до появления стабильного restaurant color/accent DTO;
  - destructive/support actions в context menu остаются запрещены до отдельного backend RBAC/idempotency/audit contract.
- Stop-list:
  - list active stop-list entries по ресторану;
  - create/update/deactivate entry;
  - связь entry с catalog item/menu item;
  - audit поля `source`, `reason`, `available_quantity`, `active`, `updated_at`;
  - RBAC permissions для просмотра и изменения stop-list.
- Sales analytics:
  - агрегаты выручки, чеков, среднего чека и платежных методов по business date;
  - динамика по часам;
  - top items/categories;
  - фильтр по ресторану и периоду.
- Edge hardware inventory:
  - список физических устройств Edge: POS terminal, fiscal device, printer, KDS;
  - hardware status, firmware, serial number, network endpoint;
  - read-only health diagnostics;
  - operator commands только после отдельного idempotency/audit contract.
- Warehouse/TTN:
  - warehouse items, balances, receipts, write-offs, inventory count;
  - TTN import/acceptance workflow;
  - costing hooks согласно `docs/backend/INVENTORY-COSTING-SPEC.md`.
- Sync operations:
  - журнал синхронизации с typed status/event categories;
  - safe retry failed sync command с idempotency key;
  - clear/log maintenance только как audit-backed support action.

## вне текущего объема

- Моковые симуляции продаж, обрывов связи, печати тестовых чеков и локальных складских операций.
- Авторитетные financial/order state transitions на frontend.
- Любые destructive/support operations без отдельного RBAC, audit log и backend safety contract.
