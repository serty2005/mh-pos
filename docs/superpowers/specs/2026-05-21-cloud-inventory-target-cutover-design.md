# Cloud Inventory Target Cutover Design

Статус: реализуется сейчас.

## Цель

Перевести склад на целевую Cloud-centric архитектуру одним изменением: Cloud принимает inventory events, durable worker создает Cloud-owned `stock_documents` и `stock_ledger`, а POS Edge перестает иметь legacy runtime для ручных складских документов, движений, остатков и себестоимости.

## Архитектура

POS Edge остается генератором operational events и потребителем read-only reference data. В POS Edge managed SQLite baseline удаляются legacy tables `purchase_receipts`, `purchase_receipt_lines`, `stock_documents`, `stock_moves`, `stock_balances`, `item_costs`; application service `CreateManualStockDocument`, repository methods и tests под `StockDocumentPosted` удаляются. Документация сохраняет историческую пометку: прежний Edge-side manual stock document method использовался в pre-pilot foundation и удален при cutover на Cloud-centric inventory.

Cloud receiver расширяет runtime catalog целевыми event types: `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `StopListUpdated`; текущие `RefundRecorded` и `CancellationRecorded` остаются accepted financial events и становятся входом для inventory worker только при stock-effect disposition. При успешном idempotent receive Cloud в той же PostgreSQL transaction создает durable queue row для inventory-relevant events.

## Worker

Cloud Inventory Worker claim'ит pending queue rows, обрабатывает их идемпотентно и пишет Cloud-owned документы. Первый runtime cutover поддерживает deterministic ledger fallback без recipe expansion: если payload содержит `items[]`, worker пишет движения по `catalog_item_id`, `quantity`, `unit_code` и optional `order_line_id`; если recipe data отсутствует, `CheckClosed`/`ItemServed` списывают сам catalog item. Costing на первом шаге пишет `unit_cost_minor = 0`, `total_cost_minor = 0`, `costing_status = estimated`; retro costing остается отдельной очередью через существующий `stock_recalculation_jobs`.

`RefundRecorded` и `CancellationRecorded` с `no_stock_effect` или `manual_review` не создают складских документов. `return_to_stock` создает `RETURN`/`IN`, `write_off_waste` создает `WASTE`/`OUT`. Whole-check payload без item-level stock data не создает движения и уходит в failed/manual-review queue reason, чтобы не делать общий stock effect без нормализованных строк.

## Ошибки и идемпотентность

Queue row имеет состояния `pending`, `processing`, `processed`, `failed`, attempts, lock metadata и safe `last_error`. Duplicate receive не создает второй queue row и не дублирует ledger. Worker failure не ломает Cloud receiver ACK: raw event уже принят, а inventory processing retry/review идет отдельно.

## Тестирование

Тесты покрывают:

- Cloud contracts принимают target inventory events и отклоняют неизвестные/битые payloads.
- Cloud receiver создает ровно один queue row на idempotent receive.
- Worker создает `SALE`, `RETURN`, `WASTE`, `PURCHASE`, `INVENTORY_COUNT`, `PRODUCTION` documents и ledger rows из нормализованных items.
- Worker не создает движения для `no_stock_effect` и помечает неподдержанный whole-check stock effect как failed/manual review.
- POS Edge baseline больше не содержит legacy stock tables, service/repository methods и `StockDocumentPosted` tests.
- Docs отражают текущее поведение и историческое удаление legacy method.
