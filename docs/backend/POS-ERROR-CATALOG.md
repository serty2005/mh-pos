# Каталог ошибок POS Backend

## Назначение

Документ фиксирует реализованный сейчас контракт безопасных API ошибок POS Edge backend.

Код и тесты остаются источником истины. Этот каталог отражает фактическую реализацию `pos-backend/internal/platform/http/respond.go`, `pos-backend/internal/pos/api/router.go` и покрывающие тесты.

## Контракт error response

Реализовано сейчас:

```json
{
  "error": {
    "code": "PERMISSION_DENIED",
    "message_key": "errors.permission",
    "details": {},
    "correlation_id": "request-id"
  }
}
```

- `code` - stable machine-readable код ошибки.
- `message_key` - safe i18n key для UI.
- `details` - только безопасные детали, без PIN, hash, manager PIN, raw auth payload, SQL details и stack trace.
- `correlation_id` - request id для support/debug; backend также возвращает его в `X-Request-ID`, если он есть в request context.
- `X-Error-Code` дублирует stable `code` для audit middleware.

Internal cause пишется только в structured backend log.

## Реализованный каталог

| code | HTTP | message_key | Бизнес-смысл | UI behavior | Retryable | log level | Sensitive-data policy |
|---|---:|---|---|---|---|---|---|
| `VALIDATION_FAILED` | 400/422 | `errors.validation` | Некорректный payload или параметры запроса | Inline validation или modal для blocking action | no | WARN | Не возвращать raw payload |
| `SESSION_REQUIRED` | 401 | `errors.session.required` | Для operator/business flow нет active session context | Очистить session state и перейти на `/login` | no | WARN | Не раскрывать session internals |
| `SESSION_REVOKED` | 401 | `errors.session.revoked` | Session найдена, но уже `revoked` | Очистить session state, показать controlled redirect/login dialog | no | WARN | Не раскрывать причину revoke сверх safe key |
| `SESSION_CONTEXT_MISMATCH` | 403 | `errors.session.contextMismatch` | `node_device_id`/`client_device_id` не совпадает с session context | Modal с безопасным объяснением, без destructive logout | no | WARN | Mask device/session ids в логах |
| `PERMISSION_DENIED` | 403 | `errors.permission` | Actor session активна, но permission отсутствует | Modal "Недостаточно прав", не делать logout | no | WARN | Не возвращать required permission id, если это раскрывает внутреннюю политику |
| `FORBIDDEN` | 403 | `errors.permission` | Общий ожидаемый отказ доступа | Modal "Недостаточно прав" | no | WARN | Без секретов |
| `NOT_FOUND` | 404 | `errors.not_found` | Сущность не найдена | Compact notice или modal для blocking flow; optional cash-shift/order current reads UI может трактовать как empty state | no | WARN | Не раскрывать SQL/query details |
| `CONFLICT` | 409 | `errors.conflict` | Нарушен текущий state/business invariant | Modal с предложением обновить состояние; payment 409 требует refetch order/precheck/check/cash session без auto-retry оплаты | no | WARN | Без internal state dump |
| `DUPLICATE_PIN` | 409 | `errors.conflict_duplicate_pin` | PIN совпал с несколькими active employees | Modal с бизнес-сообщением, не показывать PIN | no | WARN | PIN не возвращается и не логируется |
| `ACTIVE_PRECHECK_CONFLICT` | 409 | `errors.conflict_active_precheck` | Для заказа уже есть активный precheck | Modal с предложением обновить заказ/precheck | no | WARN | Без raw domain error |
| `DUPLICATE_COMMAND` | 409 | `errors.conflict_duplicate_command` | Повтор command id/idempotency conflict | Modal/notice, не auto-retry write command | no | WARN | Без raw payload |
| `SALE_STOP_LIST_CONFLICT` | 409 | `errors.stopListConflict` | Позиция или обязательный recipe component находится в active stop-list | UI показывает localized business error и не рассчитывает availability на клиенте | no | WARN | Без stock/internal query details |
| `KITCHEN_WAREHOUSE_REQUIRED` | 400 | `errors.kitchen.warehouseRequired` | Kitchen stock command не передал valid `warehouse_id`, и default warehouse отсутствует в локальном `warehouse_reference` | Показать blocking validation, предложить дождаться sync/admin setup | no | WARN | Не раскрывать raw inventory reference query |
| `KITCHEN_RECEIPT_LINE_ITEM_REQUIRED` | 400 | `errors.kitchen.receiptLineItemRequired` | Receipt line не содержит поддержанный текущим contract `catalog_item_id` | Inline validation строки receipt | no | WARN | Не возвращать raw form payload |
| `KITCHEN_RECEIPT_LINE_TOTAL_REQUIRED` | 400 | `errors.kitchen.receiptLineTotalRequired` | Receipt line не содержит положительный `line_total_minor` или содержит некорректную цену | Inline validation строки receipt | no | WARN | Не возвращать raw form payload |
| `KITCHEN_WRITE_OFF_REASON_REQUIRED` | 400 | `errors.kitchen.writeOffReasonRequired` | Stock write-off не содержит reason/reason_code | Inline validation формы списания | no | WARN | Не возвращать raw form payload |
| `KITCHEN_INVENTORY_COUNT_EMPTY` | 400 | `errors.kitchen.inventoryCountEmpty` | Inventory count отправлен без строк | Inline validation формы ревизии | no | WARN | Не возвращать raw form payload |
| `KITCHEN_PRODUCTION_RECIPE_REQUIRED` | 400 | `errors.kitchen.productionRecipeRequired` | Production command выбран для заготовки без active recipe на Edge | Blocking validation, обновить recipes sync/reference | no | WARN | Не раскрывать raw recipe query details |
| `RATE_LIMITED` | 429 | `errors.rateLimit` | Превышен лимит PIN login attempts | Notice/modal с рекомендацией подождать | yes, вручную | WARN | PIN не возвращается и не логируется |
| `INTERNAL_ERROR` | 500 | `errors.server` | Неожиданная или инфраструктурная ошибка | Modal с generic текстом и support code | no для write, осторожно для read/status | ERROR | Stack trace и SQL details только в backend log |

Примечания:

- реализовано сейчас: `POST /api/v1/system/provisioning/pair-via-license` мапит ожидаемые `PAIRING_CODE_INVALID` и `PAIRING_CODE_EXPIRED` от License Server в `400 VALIDATION_FAILED`, а не в `500 INTERNAL_ERROR`; внутренняя причина остается только в structured log и `edge_provisioning_state.last_error`.
- реализовано сейчас: `GET /api/v1/employee-shifts/current` при отсутствии открытой личной смены возвращает `200 null`, а не `404 NOT_FOUND`; это empty state, а не error contract.
- реализовано сейчас: `POST /api/v1/prechecks/{id}/payments` возвращает `409 CONFLICT` / `errors.conflict` для state conflicts, включая отсутствие открытой кассовой смены, несовпадение ресторана кассовой смены с заказом, stale/inactive precheck, overpayment и уже созданный final check. Backend не возвращает raw internal reason в response; UI обязан показать безопасное бизнес-сообщение, обновить состояние заказа/precheck/check/current cash session и не повторять оплату автоматически.
- реализовано сейчас: add/increase order line commands возвращают `409 SALE_STOP_LIST_CONFLICT` / `errors.stopListConflict`, если продаваемая позиция или обязательный recipe component находится в active локальном stop-list. Backend не возвращает stock balance, raw SQL или internal query details.
- реализовано сейчас: kitchen stock input routes возвращают stable kitchen validation codes для отсутствующего склада, receipt line item/total, пустой ревизии, отсутствующей причины списания и production без active recipe. POS Edge не возвращает raw Go/SQL errors и не создает local stock documents при отказе.

## Поведение логирования

Реализовано сейчас:

- HTTP error path пишет structured log с `request_id`, `operation=http.error`, `action`, `result=rejected`, `status`, `error_code`, masked `node_device_id`, `client_device_id`, `session_id`, `actor_employee_id`, `internal_error`.
- Panic recovery возвращает safe `INTERNAL_ERROR` response и пишет stack trace только в backend log.
- Request audit middleware читает stable `X-Error-Code`, поэтому audit logs не деградируют до generic `HTTP_403`.

## Retry policy

Реализовано сейчас:

- UI может retry network/timeout/server для safe read/status запросов.
- UI mutation defaults отключают auto-retry, чтобы не повторять финансовые write commands без explicit idempotency policy.
- `429 RATE_LIMITED` считается retryable только вручную после ожидания.

Вне текущего объема:

- расширенный per-endpoint retry-after contract;
- локализованные backend message strings вместо `message_key`;
- раскрытие internal validation field map, если оно может вернуть raw sensitive payload.
