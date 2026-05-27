# POS Backend Spec

Статус: актуальный backend contract для текущего cashier runtime и целевого полного пилота.

Код и тесты являются источником истины. Этот документ не описывает будущие API как реализованные.
Сводная карта фактически реализованного функционала по всем runtime-модулям находится в `docs/CURRENT-FUNCTIONAL-STATE.md`.

## Runtime Modules

Реализовано сейчас:

- POS Edge HTTP API на `chi`.
- SQLite repository/migration/runtime verification.
- Application services для auth, staff shifts, cash sessions, floor/menu/catalog reads, order, pricing, precheck, payment/check, master-data ingest, outbox.
- Manual persistence implementation в infrastructure repositories.
- Selected modifiers runtime для order lines, pricing, precheck/check snapshots.
- API чтения активных заказов зала для статусов столов и панели активных заказов POS.
- Runtime-поля курса подачи и комментария строки заказа, которые не влияют на финансовые итоги.
- Service catalog items as sellable POS items.
- Stop-list sale blocking для добавления order line и увеличения quantity по локальным read model tables `stop_lists`, `recipe_versions`, `recipe_lines`.
- POS-generated `CheckClosed` inventory fact при final check после полной оплаты.

Не реализовано сейчас:

- `sqlc` как текущий persistence implementation;
- Edge-side stock documents/moves/balances/costing;
- chef receipt and proposal generation beyond минимального kitchen ticket lifecycle;
- payment processor module;
- fiscal adapter.

## Public Routes

Реализовано сейчас в `pos-backend/internal/pos/api/router.go`:

- `GET /health`
- `POST /api/v1/auth/pin-login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/session`
- `POST /api/v1/system/pair`
- `GET /api/v1/system/pairing-status`
- `GET /api/v1/system/provisioning-status`
- `POST /api/v1/system/provisioning/register-cloud`
- `POST /api/v1/system/provisioning/pair-via-license`
- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/catalog/items`
- `GET /api/v1/menu/items`
- `POST /api/v1/employee-shifts/open`
- `POST /api/v1/employee-shifts/{id}/close`
- `GET /api/v1/employee-shifts/current`
- `GET /api/v1/employee-shifts/recent`
- `POST /api/v1/orders`
- `GET /api/v1/orders/current`
- `GET /api/v1/orders/active`
- `GET /api/v1/orders/closed`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/modifiers`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/details`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `GET /api/v1/orders/{id}/pricing`
- `POST /api/v1/orders/{id}/discounts`
- `POST /api/v1/orders/{id}/surcharges`
- `GET /api/v1/pricing/policies`
- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/orders/{id}/prechecks`
- `POST /api/v1/orders/{id}/close`
- `GET /api/v1/prechecks/{id}`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/reprint`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/checks/{id}`
- `POST /api/v1/checks/{id}/reprint`
- `POST /api/v1/checks/{id}/cancellations`
- `POST /api/v1/checks/{id}/refunds`
- `GET /api/v1/checks/{id}/financial-operations`
- `GET /api/v1/financial-operations`
- `GET /api/v1/kitchen/order-queue`
- `GET /api/v1/kitchen/tickets`
- `POST /api/v1/kitchen/tickets/{id}/accept`
- `POST /api/v1/kitchen/tickets/{id}/start`
- `POST /api/v1/kitchen/tickets/{id}/hold`
- `POST /api/v1/kitchen/tickets/{id}/ready`
- `POST /api/v1/kitchen/tickets/{id}/serve`
- `POST /api/v1/kitchen/tickets/{id}/recall`
- `POST /api/v1/kitchen/tickets/{id}/cancel`
- `POST /api/v1/kitchen/stock-receipts`
- `POST /api/v1/kitchen/inventory-counts`
- `POST /api/v1/kitchen/stock-write-offs`
- `POST /api/v1/kitchen/productions`
- `POST /api/v1/payments/{id}/refund`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `GET /api/v1/cash-shifts/current`
- `POST /api/v1/cash-drawer-events`
- `GET /api/v1/sync/outbox`
- `GET /api/v1/sync/local-events`
- `GET /api/v1/sync/status`
- `POST /api/v1/sync/retry-failed`
- `POST /api/v1/sync/master-data/snapshots`
- `POST /api/v1/sync/master-data/{stream}`
- `GET /api/v1/storage/status`
- `POST /api/v1/storage/retention/dry-run`
- `POST /api/v1/storage/archive/export-plan`
- `POST /api/v1/storage/archive/export`
- `POST /api/v1/storage/archive/verify`
- `POST /api/v1/storage/archive/read-plan`
- `POST /api/v1/storage/archive/lookup`
- `POST /api/v1/storage/archive/apply-plan`
- `POST /api/v1/storage/archive/apply-readiness`

## Текущие Optional Reads

Реализовано сейчас:

- `GET /api/v1/employee-shifts/current` возвращает `200` с JSON `null`, когда у authenticated employee нет открытой личной смены.
- `GET /api/v1/cash-shifts/current` и `GET /api/v1/orders/current?table_id=...` пока используют `404 NOT_FOUND` как empty state, когда текущая сущность отсутствует.
- Эти empty states не retryable и не означают инфраструктурный сбой. Cashier UI превращает `200 null` или optional `404` в `null` через optional read helper и показывает состояние "нет открытой смены/кассовой смены/активного заказа".
- `GET /api/v1/employee-shifts/current` ищет текущую личную смену по authenticated employee в restaurant context, а не по устройству.
- `GET /api/v1/cash-shifts/current` ищет открытую кассовую смену по устройству из authenticated request context.
- `node_device_id` в query/header остается частью authenticated device/session metadata; он не выбирает personal employee shift вместо actor session.

## Cashier Flow

Реализовано сейчас:

1. Кассир открывает личную смену сотрудника.
2. Кассир открывает кассовую смену устройства.
3. Кассир создает заказ на стол.
4. Кассир добавляет, меняет количество и списывает активные строки заказа, включая selected modifiers для menu items с modifier groups.
5. Кассир может сохранять metadata курса подачи и комментария строки через backend; эти поля возвращаются вместе со строкой и не меняют итоги.
6. Кассир может читать все активные заказы зала через `GET /api/v1/orders/active?hall_id=...`, поэтому статусы столов и панель активных заказов используют backend-данные, а не UI mock-данные.
7. Кассир может применять backend-authoritative discount/surcharge commands и читать pricing preview.
8. Кассир выпускает пречек.
9. Backend блокирует заказ и создает immutable financial precheck snapshot.
10. Кассир проводит одну или несколько оплат через `precheck_id`; заказ сохраняет исходную личную смену оператора, а оплата и final check относятся к текущей кассовой смене кассира.
11. Backend создает final check только после полной оплаты.
12. Кассир или менеджер может повторно напечатать копию precheck/check из immutable snapshot.
13. Авторизованный оператор может записать cancellation/refund operation; текущий cashier UI использует check-level ledger routes для full whole-check и partial `order_line`/quantity cancellation/refund и оставляет compatibility payment refund route как fallback для закрытых заказов.

## Portable Stack Smoke

Реализовано сейчас:

- `scripts/seed-dev-system.py` является единственным локальным seed entrypoint: он создает полный набор Cloud-owned справочников, публикует master data, выполняет license pairing POS Edge и проверяет POS read model.
- Seed script выполняет health check Cloud/POS/License, берет `node_device_id` из POS provisioning status, создает справочники через Cloud API, публикует packages, генерирует license pairing code, вызывает POS `pair-via-license` и проверяет PIN login/menu/floor read model.
- `--run-minimal-flow` выполняет минимальную financial/KDS mutation через HTTP: waiter order/precheck -> KDS served -> cashier payment/final check -> `ItemServed`/`CheckClosed` -> Cloud inventory ledger без double consumption. Refund/cancellation runtime boundaries проверяются отдельными backend/UI тестами. Seed script не делает automatic retry financial mutations и destructive storage actions.
- JSON summary содержит локальные demo IDs, pairing code и PIN-коды; он предназначен только для local/dev проверки и игнорируется git.

Вне текущего объема:

- archive restore в active SQLite;
- PSP refund, fiscal integration;
- замена полноценным e2e/UI тестам.

Read contract закрытых заказов:

- Реализовано сейчас: `GET /api/v1/orders/closed` принимает `limit`, `offset`, `business_date_local`, `from_business_date_local`, `to_business_date_local`, `shift_id`, `device_id`, `check_id`.
- Default `limit` = `50`, max `limit` = `100`; отрицательный `offset` и невалидные business date фильтры отклоняются.
- Сортировка stable newest-first: close timestamp, затем `id`.
- API без фильтра возвращает только bounded latest page, а не всю историю.
- Реализовано сейчас: `GET /api/v1/checks/{id}/financial-operations` принимает `limit`/`offset` и возвращает append-only ledger operations/items по конкретному final check под `pos.check.view`.
- Реализовано сейчас: `GET /api/v1/financial-operations` принимает `business_date_from`, `business_date_to`, `operation_type`, `shift_id`, `original_shift_id`, `check_id`, `limit`, `offset`; это backend-owned локальная отчетная выдача ledger, не UI total calculator.

Operational activity/sync read contract:

- Реализовано сейчас: `GET /api/v1/sync/outbox` принимает optional `limit`; backend repository возвращает bounded default page `100`, если limit пустой, отрицательный, нулевой или больше `500`.
- Реализовано сейчас: `GET /api/v1/sync/local-events` принимает optional `limit` и `event_type`; backend repository возвращает bounded default page `100`, если limit пустой, отрицательный, нулевой или больше `500`, и сортирует по `created_at DESC, id DESC`.
- POS UI sync/activity drawer использует `limit=5` для outbox и local events; эти reads не являются бесконечным журналом отчетности.
- `GET /api/v1/sync/status` агрегирует counts by outbox status and does not return payload history.

Контракт lifecycle локального storage:

- Реализовано сейчас: `GET /api/v1/storage/status` возвращает read-only operational snapshot локальной SQLite БД: page stats (`page_count`, `page_size_bytes`, `freelist_count`, estimated size), high-level table counts, диапазон business date закрытых чеков, закрытые заказы по business date, outbox counts by status/direction и число blocking Edge -> Cloud outbox messages.
- Реализовано сейчас: `POST /api/v1/storage/retention/dry-run` принимает `cutoff_business_date_local` в формате `YYYY-MM-DD`, отклоняет будущий cutoff и считает документы с `checks.business_date_local < cutoff`, которые могли бы войти в будущую archive/retention policy.
- Реализовано сейчас: `POST /api/v1/storage/archive/export-plan` принимает `cutoff_business_date_local` и optional `mode = manifest_only`, отклоняет будущий cutoff, считает тот же closed-order candidate set по `checks.business_date_local < cutoff` и возвращает deterministic manifest без записи archive files и без изменения runtime tables.
- Archive export-plan response возвращает `mode = manifest_only`, `result_mode = plan_only`, `destructive_apply_supported = false`, `blocked = true`, `block_reasons`, `archive_set`, protected flags для ledger/snapshots/local events/outbox, active/open blockers (`active_orders`, `open_shifts`, `open_cash_sessions`), blocking outbox count и manifest `format_version = storage-archive-manifest-v1` с restaurant id, business-date range, cutoff и stable table list.
- Реализовано сейчас: `POST /api/v1/storage/archive/export` принимает `cutoff_business_date_local` в формате `YYYY-MM-DD` и `reason`, отбирает только closed orders с `checks.business_date_local < cutoff`, включая связанные `kitchen_tickets`/`kitchen_ticket_events`, создает typed JSONL archive и JSON manifest в `POS_SQLITE_ARCHIVE_DIR` или default archive directory рядом с SQLite data directory.
- Archive export response возвращает `mode = export_only`, `result_mode = export_only`, `destructive_apply_supported = true`, `runtime_rows_deleted = false`, `archive_id`, `archive_path`, `manifest_path`, archive `sha256`, counts, business-date min/max, source node/device metadata если она есть в runtime, `blocked`, `block_reasons`, `financial_ledger_protected`, `immutable_snapshots_protected` и `export_created`.
- Реализовано сейчас: `POST /api/v1/storage/archive/verify` принимает `archive_id` или `manifest_path`/`archive_path`; endpoint проверяет, что пути находятся внутри configured archive dir, читает manifest и JSONL streaming-способом, проверяет `version = pos_storage_archive_export_v1`, SHA-256, counts manifest vs JSONL, required identity fields, business-date range/cutoff consistency, `runtime_rows_deleted = false`, snapshot payload в `prechecks`/`checks` и summary-only payload policy для `local_event_log`/`pos_sync_outbox`. Response возвращает `valid`, `errors`, counts, business-date range, tables и verification flags; active runtime tables не читаются и не мутируются.
- Реализовано сейчас: `POST /api/v1/storage/archive/read-plan` принимает `archive_id` или `manifest_path`/`archive_path`, optional filters `business_date_local`, `order_id`, `check_id`, `limit`, `offset`; перед preview выполняет ту же archive verification. Response возвращает `result_mode = read_plan_only`, bounded `archived_closed_orders`, default `limit=50`, max `limit=100`, `runtime_restored = false`, `runtime_rows_deleted = false`, summary counts/tables/business-date range и verification flags без sync/event payload JSON.
- Verify/read-plan block reasons включают `archive_manifest_missing`, `archive_manifest_invalid`, `archive_manifest_version_mismatch`, `archive_path_outside_archive_dir`, `archive_missing`, `archive_unreadable`, `archive_jsonl_malformed`, `archive_sha_mismatch`, `archive_manifest_counts_mismatch`, `archive_snapshot_payload_missing`, `archive_identity_fields_missing`, `archive_business_date_range_mismatch`, `archive_runtime_rows_deleted_true`, `archive_sensitive_payload_policy_violation`.
- Реализовано сейчас: `POST /api/v1/storage/archive/lookup` принимает `archive_id` или `manifest_path`/`archive_path` и ровно один lookup key: `check_id` или `order_id`; перед lookup выполняет ту же archive verification, затем streaming-способом ищет archived check/order без загрузки всего archive payload в память.
- Archive lookup response возвращает `result_mode = archive_lookup_preview`, `lookup.found`, `archive_id`, immutable `check.snapshot`, immutable `precheck.snapshot` и `related_counts` для `order_lines`, `payments`, `financial_operations`, `financial_operation_items`. Lookup не разрешает произвольные table names, не раскрывает `local_event_log`/`pos_sync_outbox` payloads и при отсутствии записи возвращает `archive_record_not_found`.
- Реализовано сейчас: `POST /api/v1/storage/archive/apply-plan` принимает `cutoff_business_date_local`, `archive_path`, `manifest_path` и optional `mode = plan_only`; при verified archive и runtime safety выполняет destructive apply в active SQLite и последующий `VACUUM`. Restore в active SQLite не выполняется.
- Archive apply-plan response при успехе возвращает `result_mode = destructive_apply`, `destructive_apply_supported = true`, `runtime_rows_deleted = true`, `blocked = false`, deleted `eligible_counts`, archive counts, protected flags и verification summary. При нарушении gate возвращает `result_mode = apply_blocked`, `runtime_rows_deleted = false`, `blocked = true` и machine-readable `block_reasons`.
- Apply-plan verification читает manifest, проверяет `version = pos_storage_archive_export_v1`, streaming-считает SHA-256 и rows по table из JSONL без загрузки всего archive payload в память, сверяет archive counts с manifest и текущим eligible runtime scope по `checks.business_date_local < cutoff`, проверяет наличие snapshot payload в `prechecks`/`checks`, required identity fields, business-date range/exclusive cutoff consistency, `runtime_rows_deleted = false` и summary-only payload policy.
- Apply-plan blockers включают `archive_manifest_missing`, `archive_manifest_version_mismatch`, `archive_sha_mismatch`, `archive_manifest_counts_mismatch`, `archive_counts_mismatch`, `pending_edge_to_cloud_outbox`, `open_operational_boundary`, `archive_snapshot_payload_missing`; невалидный или будущий cutoff возвращается как blocked plan с `invalid_cutoff` или `future_cutoff`.
- Реализовано сейчас: `POST /api/v1/storage/archive/apply-readiness` принимает тот же read-only input, что apply-plan, и возвращает отдельный policy gate для destructive apply/delete/compaction.
- Archive apply-readiness response возвращает `result_mode = apply_readiness_only`, `destructive_apply_supported = true`, `ready_for_destructive_apply`, `runtime_rows_deleted = false`, `archive_verified`, `manifest_verified`, `snapshot_payload_verified`, `runtime_scope_verified`, `blocking_outbox_count`, `pending_edge_to_cloud_outbox`, `open_operational_boundaries`, `protected_data`, `eligible_counts`, `archive_counts`, verification summary и `block_reasons`.
- `local_event_log` и `pos_sync_outbox` в archive export включаются только как summary/reference rows без `payload_json`; payload остается в active DB, чтобы не выносить потенциально sensitive sync/event data в архивный файл.
- Все storage lifecycle endpoints требуют operator session с `pos.sync.view`; UI visibility не является security boundary.
- `POST /api/v1/storage/retention/dry-run` остается non-destructive legacy readiness surface: response имеет `result_mode = dry_run_only`, защищает ledger/snapshots и возвращает `dry_run_only_no_archive_policy`. Фактическое удаление выполняется только через archive `apply-plan` после verified archive и runtime safety gate.
- Export-only archive, verify, read-plan и lookup не выполняют `DELETE`, `UPDATE`, destructive apply, restore в active SQLite, `VACUUM`, `wal_checkpoint`, startup/background auto-archive или compaction. Non-sent `edge_to_cloud` outbox rows по candidate aggregates возвращаются как block reason; сам export file может быть создан.
- Вне текущего объема: archive tables и restore archived closed checks обратно в active SQLite.

## Precheck Contract

Реализовано сейчас:

- `IssuePrecheckCommand` contains `order_id`.
- Precheck can be issued only for `open` order.
- Order device must match command device.
- Only one active issued precheck per order is allowed.
- Snapshot contains active lines with selected modifiers, currency, discounts, surcharges, taxes, totals and calculation breakdown at issue time.
- Order becomes `locked`.
- `CancelPrecheckCommand` требует `precheck_id`, `manager_pin`, `cancellation_reason`; `manager_employee_id` принимается только как legacy compatibility input и не требуется от UI.
- Отмена требует operator permission `pos.precheck.cancel.request`; backend определяет менеджера по введенному PIN внутри ресторана заказа и требует manager permission `pos.precheck.cancel`.
- Cancel writes manager override audit and returns order to `open`.
- Reprint requires `pos.precheck.reprint` and valid immutable snapshot.

Pricing contract:

- `GET /api/v1/orders/{id}/pricing` returns backend-calculated preview for open/current order state.
- `POST /api/v1/orders/{id}/discounts` supports line/order discounts with `amount_kind = percentage|fixed` and required `application_index`.
- `POST /api/v1/orders/{id}/surcharges` supports `service_charge`, `pb1_service_fee` and `manual` surcharge foundation with `amount_kind = percentage|fixed` and required `application_index`.
- Canonical pipeline is `order lines subtotal -> unified ordered modifiers by application_index -> taxable base -> taxes -> grand total`.
- Discounts and surcharges share one `application_index` space per calculation snapshot; duplicate indexes are rejected.
- Surcharge is a separate domain operation and is not represented as a negative discount.
- Tax Always Last is enforced by the calculation pipeline.
- Rounding policy is deterministic integer half-up minor units (`integer_half_up_minor_units_v1`).
- Tax foundation supports `tax_profiles`, `tax_rules`, percentage/fixed components, inclusive/exclusive mode, compound foundation and tax exempt profile foundation; inclusive tax is included in `tax_total`, but does not increase grand total.
- Precheck issue persists immutable breakdown in `precheck_lines`, `precheck_discounts`, `precheck_surcharges`, `precheck_taxes`.
- Menu price/tax rule changes after precheck issue do not mutate old precheck/check snapshots.
- Selected modifiers are priced by backend calculation and persisted in `order_line_modifiers`, precheck/check snapshots and reprint payloads. `POST /orders/{id}/lines` and `PATCH /orders/{id}/lines/{line_id}/modifiers` validate required/min/max, active group/option, option-group membership, menu item link and non-negative option price. Modifier update is allowed only for active lines of open editable orders without active precheck/finalized check and writes `OrderLineModifiersUpdated` to outbox/local events.
- Service items use the same order/pricing/precheck/check flow as other sellable menu items and do not imply recipe semantics.
- Catalog item lifecycle/availability audit: Edge runtime хранит active catalog/menu read models из Cloud lifecycle status. Temporary unavailability не реализована как глобальный catalog item status; запланировано далее моделировать ее как overlay menu/restaurant/terminal-group, если это будет принято.
- UOM audit: текущий runtime хранит string units (`base_unit`, recipe `unit`, stock move `unit`). Separate UOM reference model with machine `code`, `name`, `short_name` and translations не реализована сейчас, поэтому новый runtime code не должен считать display labels canonical UOM codes.

## Payment, Check, Cancellation And Refund Contract

Реализовано сейчас:

- Payment capture endpoint is `POST /api/v1/prechecks/{id}/payments`.
- Capture command accepts `method`, `amount`, `currency` and optional provider metadata.
- Supported methods are `cash`, `card`, `other`.
- Payment requires open cash session for the same device, shift and restaurant.
- Payment updates `prechecks.paid_total`.
- Payment updates `prechecks.remaining_total`.
- Partial payments are allowed; overpayment is rejected.
- Full payment creates one final check and closes order.
- Check snapshot includes immutable precheck snapshot and payment snapshot.
- Оплата относится к текущей открытой кассовой смене; заказ сохраняет исходную личную смену. `PaymentCaptured`, `CheckCreated` и `CheckClosed` используют кассовую смену оплаты в sync envelope.
- Check reprint/refund use immutable snapshots and do not re-read current menu modifier data.
- Reprint check requires `pos.check.reprint`.
- Cancellation endpoint is `POST /api/v1/checks/{id}/cancellations`.
- Refund endpoint is `POST /api/v1/checks/{id}/refunds`.
- Ledger endpoints принимают `command_id`, `operation_kind`, `inventory_disposition`, reason и optional item scopes. Текущий cashier UI отправляет whole-check commands без item list, поэтому backend записывает `whole_check` из immutable check snapshot; для partial line/quantity UI отправляет `items[]` со scope `order_line`, `order_line_id`, `quantity`, `amount`, `currency` и `tax_amount`.
- Compatibility refund endpoint is `POST /api/v1/payments/{id}/refund`; it records a refund operation with payment allocation and does not mutate finalized payment/check/precheck totals.
- Compatibility refund requires the captured payment to belong to an order that already has a finalized check. Captured partial payment on a still-issued precheck is not refundable through this endpoint.
- Cancellation uses permission `pos.precheck.cancel`; refund uses permission `pos.payment.refund`.
- Cancellation requires the original personal shift to be open, the current cash session to belong to that shift and the same `business_date_local`.
- Refund requires the original personal shift to be closed or the current `business_date_local` to differ from the check business date; a current open cash session is still required.
- Financial operations are append-only rows in `financial_operations` and `financial_operation_items`.
- Operation type is `cancellation` or `refund`; operation kind is `full` or `partial`.
- Item scopes are `whole_check`, `order_line`, `modifier_line`, `service_charge`, `tip`, `payment`.
- Backend rejects over-cancel, over-refund, over-line-amount, over-line-quantity and over-payment-allocation scenarios.
- Operation snapshot embeds immutable check snapshot and operation items.
- Inventory disposition is explicit: `no_stock_effect`, `return_to_stock`, `write_off_waste`, `manual_review`; financial operation does not mutate stock tables.
- Текущие POS Edge events: `CancellationRecorded` и `RefundRecorded`. New refund runtime не emit legacy `PaymentRefunded`/`CheckRefunded`; эти names остаются только Cloud-accepted legacy sync event types.
- Cloud receiver валидирует current financial operation payload fields, включая совпадение payload `restaurant_id`/`device_id` с envelope, `precheck_id`, `reason` и immutable snapshot, stores raw/journal envelopes and event-type stats, and maintains detailed `cloud_projection_financial_operations` for current `CancellationRecorded`/`RefundRecorded`; public Cloud reporting HTTP/UI остается запланирован далее.

Не реализовано сейчас:

- automatic PSP authorization/capture/refund;
- fiscal receipt creation;
- refund manager PIN policy beyond current RBAC permission check;
- cashier UI for modifier/service/tip partial cancellation/refund;
- automatic stock return/write-off;
- separate `business_day` and `fiscal_shift` runtime aggregates.

## Shift, Business Date And Fiscal Boundary

Реализовано сейчас:

- Runtime имеет personal employee shift (`shifts`) и device cash session (`cash_sessions`).
- `cashier_shift` в текущем коде представлен personal employee shift aggregate/table `shifts`; отдельного объекта с именем `cashier_shift` нет.
- `business_date_local` хранится у shifts, cash sessions, payments, checks и financial operations.
- Отдельной сущности `business_day` нет; текущая business date вычисляется из restaurant settings.
- Отдельной сущности `fiscal_shift` нет.
- Final check является finalized internal POS document после полной оплаты.
- Cancellation/refund являются compensating operations и не переписывают finalized check/payment.

Вне текущего объема:

- fiscalized receipt state;
- correction document flow;
- fiscal document reprint vs ordinary POS snapshot reprint distinction in runtime;
- reopen business day policy as a first-class operation.

## Master Data Ingest

Реализовано сейчас:

- `ApplyMasterData` принимает `full_snapshot` и `incremental`.
- `full_snapshot` требует `full_snapshot_reason` со значением `terminal_restaurant_changed` или `node_role_changed`.
- Backup hook для full snapshot существует, если он настроен.
- Поддерживаемые POS Edge ingest streams:
  - `restaurants`
  - `devices`
  - `staff`
  - `floor`
  - `catalog`
  - `menu`
  - `pricing_policy`
- `catalog` применяет catalog folders, folder parameters, catalog tags, item tags, item kinds `dish`, `good`, `semi_finished`, `service` и modifier groups/options/bindings.
- `menu` применяет menu items, menu item `item_type` и effective menu item modifier group links после применения menu items.
- Cloud publication package для POS Edge должен соответствовать typed ingest DTO `ApplyMasterDataCommand`: modifier groups/options/bindings публикуются top-level массивами с `restaurant_id` и без Cloud lifecycle fields, а `menu_item_modifier_groups` остается link-only (`menu_item_id`, `modifier_group_id`, `sort_order`).
- `required`, `min_count`, `max_count`, `active` принадлежат top-level `modifier_groups[]`; эти поля не публикуются внутри `menu_item_modifier_groups[]` и не встраиваются как rich `menu_items[].modifier_groups[]` в ingest payload. Inventory/recipe expansion для modifiers не выполняется в POS order/pricing/precheck/check runtime.
- `restaurants` применяет Cloud-authored settings и `active`; опубликованный active restaurant сохраняется в Edge read model как active row.
- `pricing_policy` применяет `tax_profiles`, `tax_rules`, `service_charge_rules` и automatic discount/surcharge `pricing_policies` с sync metadata.
- `recipes` применяет `recipe_versions` и `recipe_lines` с sync metadata.
- `inventory_reference` применяет `stop_lists` и `warehouses`/`warehouse_reference` с sync metadata.
- Strict JSON decode отклоняет неизвестные request fields; unsupported stream names отклоняются до partial apply.

Только основа:

- Cloud authoring UI/publication workflow для recipes/stop-list еще не является полным runtime; текущий POS Edge принимает package payloads, которые переданы через provisioning/sync exchange.

## Pricing, Modifiers And Inventory Boundaries

Discounts/taxes:

- Реализовано сейчас: separate `Pricing` policy area, backend authoritative calculation, unified ordered discount/surcharge pipeline, tax-last invariant, immutable precheck breakdown and no UI authoritative totals.
- Реализовано сейчас: Cloud-authored tax/service-charge/automatic discount-surcharge reference rows применяются через `pricing_policy`.
- Реализовано сейчас: selected modifiers participate in line/order totals and immutable snapshots.
- Реализовано сейчас: service items are priced as normal sellable order lines.
- Запланировано далее: полный Cloud-authored pricing UI workflow и policy-id-backed manual runtime adjustments.
- Ручные order discounts/surcharges остаются Edge operational commands и требуют pricing permissions; manual policy exceptions требуют отдельный permission/audit boundary до поддержки.

Modifiers:

- Реализовано сейчас: Cloud modifier group/option/binding data is published and ingested into Edge read model.
- Реализовано сейчас: selected modifiers are accepted in add-line commands, stored on order lines, priced by backend and copied into precheck/check snapshots.
- Реализовано сейчас: cashier UI exposes modifier selection for menu items with modifier groups.
- Запланировано далее: modifier print/reporting/audit polish if pilot acceptance requires it.

Recipes/inventory:

- Реализовано сейчас: POS Edge работает как генератор events и UI ввода; он не создает `StockDocument`, `StockMove`, stock balance или costing rows.
- Реализовано сейчас: Edge SQLite содержит read-only `recipe_versions`, `recipe_lines`, `stop_lists` и `warehouse_reference`; legacy Edge-side stock tables удалены из целевого baseline.
- Реализовано сейчас: Cloud Inventory Worker обрабатывает `CheckClosed`, `ItemServed`, `StockReceiptCaptured`, `InventoryCountCaptured`, `ProductionCompleted`, `RefundRecorded`, `CancellationRecorded`, `StopListUpdated` через durable queue.
- Реализовано сейчас: Cloud PostgreSQL хранит `inventory_event_queue`, `stock_documents`, `stock_ledger` with `unit_cost_minor`, `total_cost_minor` and `costing_status`; ClickHouse batch projection `olap_stock_moves` запланирована до полного пилота.
- Реализовано сейчас: cancellation/refund ledger хранит явный `inventory_disposition`; POS runtime не мутирует local stock tables, потому что local stock tables удалены.
- Реализовано сейчас: POS Edge recipe/stop-list ingest, локальная sale blocking проверка active stop-list для sellable catalog item и mandatory active recipe components. Проверка не читает stock balance и не создает stock documents/moves.
- Реализовано сейчас: final check после полной оплаты пишет POS-generated `CheckClosed` outbox envelope из immutable `check.Snapshot`.
- Реализовано сейчас: минимальный POS Edge KDS lifecycle foundation создает kitchen tickets из order lines, пишет `KitchenTicketStatusChanged`, а `serve` пишет `ItemServed`.
- Реализовано сейчас: POS Edge kitchen stock input routes валидируют `warehouse_id`/default warehouse, существующие stock-capable catalog items, receipt supplier/document date/line totals, inventory count `counted_quantity`, write-off reason и production для active `semi_finished` с active recipe; routes пишут только `local_event_log`/`pos_sync_outbox`.
- Не реализовано сейчас: catalog/recipe proposal flows, Edge manager/KDS stop-list edit flow, Cloud-side `StockWriteOffCaptured` receiver/worker, modifier linked catalog item stock consumption, retro costing DAG.
- Запланировано до полного пилота: Cloud authoring/publication UI для recipes/stop-list, `CatalogItemChangeSuggested`, `RecipeChangeSuggested`, `StopListUpdated`, Cloud-side write-off processing и расширение KDS за пределы ticket lifecycle foundation.
- Профильный целевой contract: `docs/backend/INVENTORY-COSTING-SPEC.md`.

## Full Pilot Backend Delta

Запланировано до полного пилота:

- Stop-list API/read model:
  - Cloud-owned stop-list entries применяются на Edge через `sync/exchange` и `mastersync.Service`;
  - sale blocking возвращает стабильный safe error code/message key и не пишет order/outbox rows при отказе;
  - local check выполняется до создания order line и перед увеличением quantity;
  - `stop_list_conflict_policy` определяет порядок применения Cloud package и Edge overlay: `cloud_wins`, `edge_wins`, `last_event_wins` или `most_restrictive`.
- POS-side authoritative boundary:
  - POS Edge backend является authoritative runtime для order/precheck/payment/check commands, financial operation ledger, pricing snapshot, idempotency, cash/session boundaries, stop-list sale blocking и KDS command validation;
  - POS UI не считает authoritative totals и не принимает финансовые/складские решения;
  - POS Edge не создает stock documents, stock ledger, costing rows или ClickHouse rows.
- Advanced Kitchen API реализовано сейчас в минимальном объеме:
  - `GET /api/v1/kitchen/order-queue`;
  - `GET /api/v1/kitchen/tickets`;
  - `POST /api/v1/kitchen/tickets/{id}/accept`;
  - `POST /api/v1/kitchen/tickets/{id}/start`;
  - `POST /api/v1/kitchen/tickets/{id}/hold`;
  - `POST /api/v1/kitchen/tickets/{id}/ready`;
  - `POST /api/v1/kitchen/tickets/{id}/serve`;
  - `POST /api/v1/kitchen/tickets/{id}/recall`;
  - `POST /api/v1/kitchen/tickets/{id}/cancel`;
  - `POST /api/v1/kitchen/stock-receipts`;
  - `POST /api/v1/kitchen/inventory-counts`;
  - `POST /api/v1/kitchen/stock-write-offs`;
  - `POST /api/v1/kitchen/productions`;
  - `GET /api/v1/kitchen/order-queue` требует `pos.kitchen.view`, поддерживает `status` по вычисляемому `kitchen_order_status`, `station`, `limit`, `offset`, default/max limit `50/100`, grouped tickets по order и backend-side `elapsed_seconds`;
  - `GET /api/v1/kitchen/tickets` требует `pos.kitchen.view`, поддерживает `status`, `station`, `limit`, `offset`, default/max limit `50/100` и stable sort `created_at ASC, id ASC`;
  - status actions требуют `pos.kitchen.status.change`, принимают `command_id`, возвращают safe conflict для недопустимого перехода и не считают UI visibility security boundary;
  - replay того же kitchen `command_id` для того же ticket/action возвращает успешный идемпотентный ответ без второго ticket event/outbox event; reuse `command_id` для другой команды остается conflict;
  - tickets создаются из non-service order lines с переносом `order_line_id`, `catalog_item_id`, `menu_item_id`, `quantity`, `unit_code`, `station_routing_key`, `table_name`, `shift_id`, `device_id`, `restaurant_id`, а course/comment синхронизируются из order line details;
  - status actions пишут `KitchenTicketStatusChanged`, а served action дополнительно пишет `ItemServed` в `local_event_log` и `pos_sync_outbox`;
  - повторный цикл `served -> recall -> start -> ready -> serve` реализован; повторный `serve` с новым `command_id` пишет новый `ItemServed` с `ticket_id`, `serve_sequence` и optional `supersedes_served_event_id`.
- Kitchen stock input API реализовано сейчас:
  - chef stock receipt capture / `StockReceiptCaptured` требует `pos.kitchen.stock.receipt`;
  - inventory count / `InventoryCountCaptured` требует `pos.kitchen.stock.inventory_count`;
  - stock write-off / `StockWriteOffCaptured` требует `pos.kitchen.stock.write_off`;
  - production completed / `ProductionCompleted` требует `pos.kitchen.production.complete`;
  - replay того же `command_id` для того же event type возвращает successful replay без второго outbox/local event;
  - POS Edge не создает stock documents/moves/balances/costing rows.
- Kitchen proposal API остается `запланировано далее`:
  - catalog item suggestions / `CatalogItemChangeSuggested`;
  - recipe read/change suggestions / `RecipeChangeSuggested`;
  - kitchen stop-list edit / Edge `StopListUpdated`.
- Inventory facts:
  - реализовано сейчас: final check creation writes current financial events and additional `CheckClosed` inventory event;
  - `CheckClosed` payload includes order line id, catalog item id, quantity, unit code and `required_for_inventory`;
  - replay is protected by existing command/outbox idempotency.
- Cloud inventory/OLAP contract:
  - POS Edge остается генератором events и не пишет stock documents/costing rows;
  - Cloud Inventory Engine и ClickHouse OLAP API являются Cloud-side full pilot requirements;
  - POS API не получает synchronous dependency on ClickHouse.
- Verification gate:
  - every backend task adds focused service/repository/API tests first;
  - after each backend milestone run `cd pos-backend && go test ./...`.

## RBAC

Реализовано сейчас:

- UI visibility is UX only.
- Backend app-layer permissions are authoritative.
- Payment permissions are method-specific:
  - `pos.payment.cash`
  - `pos.payment.card.manual`
  - `pos.payment.other`
- Pricing permissions:
  - `pos.pricing.view`
  - `pos.pricing.discount.apply`
  - `pos.pricing.surcharge.apply`
- Refund uses `pos.payment.refund`.
- Check cancellation ledger uses `pos.precheck.cancel` in current backend service.
- Precheck cancel uses split request/approval:
  - `pos.precheck.cancel.request`
  - `pos.precheck.cancel`
- Reprint permissions:
  - `pos.precheck.reprint`
  - `pos.check.reprint`
- Запланировано до полного пилота:
  - waiter profile keeps order/precheck permissions but no payment/refund permissions by default;
  - kitchen profile gets kitchen view/update permissions only;
  - backend permissions remain authoritative over UI visibility.

## Provisioning

Реализовано сейчас:

- `POST /api/v1/system/provisioning/pair-via-license` принимает одноразовый license pairing code, получает Cloud URL, restaurant id, node token и node device id из License Server.
- Если локальный Edge еще не находится в статусе `paired`, а license response содержит другой `node_device_id`, локальная provisioning identity переключается на `node_device_id` из license response. Это поддерживает Cloud UI flow без ручного ввода пользователем device ID.
- Если локальный Edge уже находится в статусе `paired`, повторный `pair-via-license` и повторный assignment polling являются идемпотентными empty work: backend возвращает текущий paired status без повторного resolve/download/apply/pair и без нового `EdgeNodePaired` local event/outbox row.
- После успешного resolve Edge скачивает Cloud snapshot, применяет master data и переводит provisioning status в `paired`.

## Error And Logging Contract

Реализовано сейчас:

- HTTP panic recovery returns safe JSON error.
- Request audit log records method/path/status/duration and masked IDs.
- Sensitive data such as PINs, tokens and raw payment-sensitive payloads must not be logged.
- Stable permission/error behavior is enforced in backend services; UI must not expose raw Go/SQL errors.

## Cloud-authored pricing policy runtime adjustments

Статус: реализовано сейчас.

POS Edge поддерживает pilot-ready путь применения скидок и надбавок по `pricing_policy_id`: `POST /api/v1/orders/{id}/discounts` и `POST /api/v1/orders/{id}/surcharges` загружают активную Cloud-authored policy из Edge read model, проверяют restaurant boundary, lifecycle, kind, scope, `application_index`, состояние заказа и permission boundary, затем копируют `amount_kind`, `amount_minor`, `value_basis_points`, `scope`, `kind` и `application_index` из policy в runtime adjustment. Пользовательский `reason` допустим только как audit/comment field и не влияет на расчет.

Статус: реализовано сейчас.

`GET /api/v1/pricing/policies` возвращает активные синхронизированные policies для выбора кассиром. POS UI не должен считать authoritative totals и после применения policy обязан перечитать order/pricing/precheck state у backend.

Статус: реализовано сейчас.

Legacy ручные amount fields остаются только как explicit manual override compatibility path под существующими backend permissions `pos.pricing.discount.apply` и `pos.pricing.surcharge.apply`. Pilot UI не должен использовать этот путь для обычного кассира.
