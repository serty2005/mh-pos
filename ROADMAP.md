# ROADMAP

Статус документа: актуализировано под фактический код на 2026-05-13.

Roadmap фиксирует статусы, блокеры и следующий план. Архитектурный контракт находится в `SPECv1.3.md`, backend contract — в `docs/backend/POS-BACKEND-SPEC.md`.

## Выполнено

### Cashier Runtime

Выполнено:

- PIN login/session/RBAC foundation.
- Personal employee shifts.
- Cash sessions and cash drawer events.
- Halls/tables read model.
- Menu/catalog read model.
- Order create/read/current/closed.
- Add/change/void order lines.
- `IssuePrecheck`.
- List/get prechecks.
- Manager override cancel precheck.
- Reprint precheck from immutable snapshot.
- Precheck-based payments through `precheck_id`.
- Partial payments.
- Final check creation after full payment.
- Reprint final check from immutable snapshot.
- Payment refund route and cashier UI flow.
- `business_date_local` for shifts, cash sessions, payments and checks.

### Cloud And Sync Foundation

Выполнено:

- Cloud PostgreSQL sync receiver and operational projections foundation.
- Cloud master-data authority foundation in `004_master_data_authority.sql`.
- Cloud schema foundation for roles, employees, catalog items, dishes, goods/raw materials, semi-finished products, recipe items, categories, modifier groups/options, menu items, menu assignments and versioned publications.
- POS Edge Cloud -> Edge ingest for streams `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`.
- POS Edge outbox/local event foundation for cashier operational events.
- DDD context map exists in `docs/architecture/DDD-CONTEXT-MAP.md`.

### Persistence Policy

Выполнено:

- POS Edge SQLite as local OLTP/source of truth.
- Cloud PostgreSQL as Cloud OLTP/source of truth.
- Managed SQL files and startup migration/verification policy.
- ADR-015 accepted for persistence and analytics strategy.

Не выполнено и не должно считаться завершенным:

- `sqlc` rollout как текущий persistence implementation.
- ClickHouse runtime/projection pipeline.

## Foundation Only

Эти зоны имеют schema/domain foundation, но не являются готовым pilot runtime:

- Modifiers: Cloud tables `cloud_modifier_groups`, `cloud_modifier_options`, `cloud_menu_item_modifier_groups`.
- Recipes: SQLite `recipe_versions`, `recipe_lines`; Cloud `cloud_recipe_items`.
- Inventory: SQLite `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`, purchase receipt foundation.
- Master-data publications: Cloud package/publication foundation шире текущего POS Edge ingest.

## В Работе / До Пилота

Pilot blockers:

- Pricing/Discounts boundary:
  - отделить `Pricing` от `Catalog`;
  - описать и реализовать discount/surcharge/tax calculation только если это нужно для pilot acceptance;
  - не выдавать текущие `discount_total` / `tax_total` fields за готовый engine.
- Tax policy:
  - ввести `tax_profile` / tax policy concept при необходимости пилотного налога;
  - зафиксировать порядок расчета после скидок, если такая pilot policy утверждена.
- Modifiers:
  - решить, входят ли modifiers в первый pilot;
  - если входят, добавить Cloud publication payload, POS Edge ingest, order line snapshot, precheck/check snapshot и cashier UI flow.
- Recipes/inventory:
  - решить, входит ли automatic consumption в первый pilot;
  - если входит, реализовать consumption trigger, stock document/move service и snapshot requirements.
- Refund/reprint hardening:
  - backend и UI flow реализованы;
  - требуется финальная проверка operator policy, audit/sync expectations и acceptance tests для pilot script.
- Documentation freeze:
  - поддерживать `SPECv1.3.md` как frozen pilot contract;
  - дальние контуры переносить в roadmap/ADR, а не в pilot spec.

## Далее

После закрытия pilot blockers:

- Полный pre-pilot smoke path: Cloud master data -> Edge ingest -> login -> shift/cash session -> order -> precheck -> payment -> check -> refund/reprint.
- Сверка RBAC matrix с фактическим UI и backend permissions.
- Проверка migration/backup behavior на старой SQLite DB.
- Уточнение sync direction для refund events, если Cloud должен получать возвраты как operational events.

## После Пилота

После пилота:

- KDS runtime and kitchen ticket lifecycle.
- DishServed / production triggers.
- Full inventory engine, recipe expansion, semi-finished consumption policies.
- Real PSP/payment processor integrations.
- Fiscal adapter/fiscalization integrations.
- Delivery/channel integrations.
- ClickHouse OLAP/reporting accelerator and PostgreSQL projection pipeline.
- `sqlc` adoption, если после стабилизации схемы это уменьшит риск persistence layer.
- Full accounting/ERP integrations.

## Вне Текущего Объема

Вне текущего объема первого cashier pilot:

- KDS as required runtime dependency.
- Real PSP authorization/capture/refund flow.
- Fiscal device integration.
- Full inventory/procurement engine.
- ClickHouse runtime dependency.
- UI-side authoritative financial calculation.
- Edge-side creation of Cloud-owned master data.

## Definition Of Ready For Cashier Pilot

Готовность к первому cashier pilot означает:

- текущий cashier flow проходит smoke/e2e без ручной правки данных;
- документация не обещает runtime, которого нет в коде;
- pricing/modifiers/inventory либо реализованы и протестированы, либо явно исключены из pilot acceptance;
- backend and UI docs согласованы по refund/reprint/current routes;
- `sqlc` и ClickHouse описаны только как planned/post-pilot options, не как текущий runtime.
