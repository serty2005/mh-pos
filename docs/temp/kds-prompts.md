Ниже набор универсальных промптов. Их можно выдавать агенту по одному, последовательно.

**Общий Префикс**

```text
Мы работаем в репозитории /home/serty/repos/mh-pos над RMS-POS для ресторанов.

Перед изменениями обязательно:
1. Прочитай AGENTS.md.
2. Прочитай docs/backend/KITCHEN-PROCESSES-SPEC.md.
3. Проверь текущее состояние кода и документации, не полагайся на старые выводы.
4. Используй CodeGraph для структурного анализа, rg только для текстовых поисков.
5. Не откатывай чужие изменения.
6. После реализации обнови профильную документацию, затронутую изменением.
7. В финале укажи: что найдено, что изменено, файлы, проверки, риски, что дальше, что вне объема, затрагивался ли runtime code.
```

**Итерация 1: POS Edge KDS Lifecycle**
Выполнена.

**Итерация 2: POS Edge Kitchen Stock Events**
Выполнена.

**Итерация 3: POS Edge Catalog And Recipe Proposals**
Выполнена.

**Итерация 4: Cloud Sync Contracts, ClickHouse Trail, Inventory Analyzer**

Выполнена.

**Итерация 5: Cloud Proposal Review And Feedback**
Выполнена.

**Итерация 6: pos-ui-g Kitchen Mode**
Выполнена.

**Итерация 7: Cloud UI Manager Review**
Выполнена.

**Итерация 8: End-To-End Smoke And Documentation Alignment**
Выполнена для runtime/script/docs scope. `scripts/seed-dev-system.py --run-kitchen-process-smoke` присутствует, summary пишет отдельные `minimal_flow` и `kitchen_process_smoke`, а Python регрессионные тесты покрывают этот путь. Полный Docker smoke в текущей локальной проверке остается не подтвержден из-за окружения: buildx plugin является требованием Docker CLI/Compose, port blocker локализован через host-port overrides в `docker-compose.local.yml` для `5432`/`8123`/`9000`/`8090`/`8080`/`8095`.

**Итерация 9: ClickHouse raw_business_events**
Выполнена. `raw_business_events` реализован как managed bounded metadata read без raw payload exposure.

**Итерация 10: Stock ledger to OLAP stock moves**
Выполнена. `stock_ledger -> olap_stock_moves` реализован async bounded slice, `GET /api/v1/olap/stock-moves` читает bounded rows без raw sync payload.

**Итерация 11: OLAP export status and retry**
Выполнена. `GET /api/v1/olap/export-status?stream=raw_business_events|stock_moves` реализован read-only, `POST /api/v1/olap/export-retry` реализован как support-only control для `retry_failed|resume_from_checkpoint`.

**Итерация 12: sync/exchange ACK regression**
Выполнена. POS syncsender regression покрывает temporary `sync/exchange` failure, retry того же outbox item, item-level ACK и прекращение повторной отправки после ACK.

**Итерация 13: Stock move summary**
Выполнена. `GET /api/v1/olap/stock-move-summary` реализован как первый bounded aggregate по `olap_stock_moves`.

**Итерация 14: Stop-List Projection Conflict Policy + Readiness Signals**
Выполнена в bounded объеме. `StopListUpdated` валидируется Cloud receiver-ом, обрабатывается Cloud Inventory Worker через durable queue в safe projection без raw payload, `stop_list_conflict_policy` поддерживает `cloud_wins`, `edge_overlay_until_next_publication`, `edge_overlay_requires_manager_review`, а `GET /api/v1/sync/readiness/stop-list` дает safe readiness по publication/package, latest Edge ACK metadata и sync problem counters. Следующая итерация добавила backend-first Edge stop-list command и bounded Cloud manager review; полноценный Edge stop-list edit UI и production-grade review workflow остаются запланированы далее.

**Итерация 15: Edge-Origin Stop-List Manager Review + Kitchen Stop-List Edit Slice**
Выполнена в bounded backend-first объеме. Cloud добавил safe manager review routes для Edge-origin `StopListUpdated` без raw payload, approve применяет изменение через Cloud-owned stop-list authority/publication path, reject/request-changes не мутируют runtime authority. POS Edge добавил kitchen backend command `POST /api/v1/kitchen/stop-list-updates` с идемпотентностью по `command_id`, локальным overlay и outbox event. Cloud UI показывает bounded review surface; полноценный KDS stop-list edit UI, production workflow и full inventory/costing остаются запланированы далее.
