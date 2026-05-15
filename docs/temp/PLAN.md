# Implementation Plan: Catalog v2, Modifiers Runtime и Sync Contract

## Summary
Реализация пойдёт кодом до документации: сначала schema/domain/API/publication, затем Edge ingest/runtime/pricing/UI, затем тесты и только после этого docs/ROADMAP. Новые данные публикуются в существующих streams: `catalog`, `menu`, `pricing_policy`. Старый Edge со strict decode будет отклонять новый payload как несовместимый; новый Edge принимает и сохраняет новые секции.

Canonical catalog kinds становятся: `dish`, `good`, `semi_finished`, `service`. `ingredient` полностью убирается из Cloud/Edge enum, без data migration/backfill, так как production БД сейчас нет.

## Key Changes
- **Cloud master data**
  - Добавить PostgreSQL migration `008_*` и обновить first-install master-data schema: `cloud_catalog_folders`, `cloud_catalog_folder_parameters`, `cloud_catalog_tags`, `cloud_catalog_item_tags`, `cloud_modifier_group_bindings`, `cloud_services`, Cloud discount/surcharge policy tables.
  - `cloud_categories` оставить только menu categories; folders не связывать с menu categories.
  - У catalog item добавить `folder_id`, `kitchen_type`, `accounting_category`; folder parameters хранить расширяемо через `parameter_key`, `value_type`, `value_json`, с публикацией effective values вниз по дереву.
  - `cloud_modifier_options.price_delta` заменить канонически на `price_minor`; модификатор может быть без техкарты и с ценой `0`.
  - CRUD foundation в masterdata service/router/repository для folders, folder parameters, tags, item tags, services, modifier bindings, pricing policies.

- **Publication and sync contract**
  - `catalog` payload расширить: `catalog_items`, `folders`, `folder_parameters`, `tags`, `item_tags`, `modifier_groups`, `modifier_options`, `modifier_bindings`, effective `menu_item_modifier_groups`.
  - `menu` payload оставить для `menu_items` и menu category semantics; `cloud_categories` не публиковать как folders.
  - `pricing_policy` payload расширить discount/surcharge policies alongside `tax_profiles`, `tax_rules`, `service_charge_rules`.
  - Обновить `MasterDataPacket`, `cloudsync` payload validation, stream package generation и package tests.

- **Edge schema, ingest and catalog model**
  - Добавить SQLite migration `004_*` и обновить baseline schema: `catalog_items.type IN ('dish','good','semi_finished','service')`, folder/tag/modifier/policy tables, `order_line_modifiers`, precheck modifier snapshot rows.
  - Edge ingest расширить новыми arrays и upsert methods; strict unknown-field behavior сохранить.
  - Recipe model поменять на owner `dish|semi_finished`, component только `good|semi_finished`; `dish`, `service` и удалённый `ingredient` запрещены DB triggers + app validation.

- **Order, pricing, precheck/check runtime**
  - `AddOrderLineCommand` принимает `selected_modifiers[]` с `modifier_group_id`, `modifier_option_id`, `quantity`.
  - Runtime валидирует active/effective modifiers по menu item через bindings `menu_item|catalog_item|folder|tag`, min/max/required.
  - `OrderLine` хранит selected modifier snapshots; изменение quantity пересчитывает totals.
  - Pricing calculator считает line subtotal как base item + selected modifiers, затем применяет synced policies, manual discounts/surcharges и taxes. Manual amount остаётся только по RBAC.
  - Precheck/check snapshots и reprint/refund сохраняют выбранные modifiers и их финансовый эффект.

- **POS UI**
  - В API schemas/types добавить catalog item type, service menu items, selected modifiers.
  - В cashier flow добавить dialog выбора modifiers перед добавлением блюда.
  - В active order показывать modifiers под строкой заказа.
  - Добавить отдельную вкладку/секцию услуг; услуги не смешивать с обычным меню.
  - Все новые UI strings добавить через i18n, без hardcoded пользовательских строк.

## Test Plan
- Cloud unit/API/repository tests: CRUD folders/tags/services/modifier bindings/policies, publication contains new sections, menu categories не смешиваются с folders.
- Cloud contract tests: allowed payload fields for expanded `catalog/menu/pricing_policy`, unknown fields rejected.
- Edge schema/ingest tests: new payload sections persist; `service`, `semi_finished`, folders/tags/modifiers/policies сохраняются; unsupported streams and unknown fields still rejected.
- Recipe tests: `good|semi_finished` accepted; `dish`, `service`, `ingredient` rejected.
- Pricing/runtime tests: modifiers affect totals; service line sells normally; synced discount/surcharge policies apply; manual override permissions enforced.
- Order/precheck/check tests: selected modifiers persist into order line, precheck snapshot, check snapshot; reprint/refund continue working.
- UI tests where current practice exists: modifier selection flow, services section, active order modifier display.
- Verification commands after implementation: `go test ./...` in `pos-backend` and `cloud-backend`; `npm install && npm run build` in `pos-ui`.

## Assumptions
- Production DB/data migration is not required; existing `ingredient` rows do not need conversion.
- New Edge compatibility policy: old strict Edge rejects new payload; new Edge accepts expanded streams.
- Modifier binding canonical model is target bindings with `target_type=menu_item|catalog_item|folder|tag`; publication also provides effective menu item links for fast Edge validation.
- Documentation updates happen only after code and tests reflect the implemented behavior.
