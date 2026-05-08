# Каталог ошибок POS Backend

## Назначение

Документ фиксирует implemented now контракт безопасных API ошибок POS Edge backend.

Код и тесты остаются источником истины. Этот каталог отражает фактическую реализацию `pos-backend/internal/platform/http/respond.go`, `pos-backend/internal/pos/api/router.go` и покрывающие тесты.

## Error response contract

implemented now:

```json
{
  "error": {
    "code": "PERMISSION_DENIED",
    "message_key": "errors.permission.denied",
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

## Каталог implemented now

| code | HTTP | message_key | Бизнес-смысл | UI behavior | Retryable | log level | Sensitive-data policy |
|---|---:|---|---|---|---|---|---|
| `VALIDATION_FAILED` | 400/422 | `errors.validation` | Некорректный payload или параметры запроса | Inline validation или modal для blocking action | no | WARN | Не возвращать raw payload |
| `SESSION_REQUIRED` | 401 | `errors.session.required` | Для operator/business flow нет active session context | Очистить session state и перейти на `/login` | no | WARN | Не раскрывать session internals |
| `SESSION_REVOKED` | 401 | `errors.session.revoked` | Session найдена, но уже `revoked` | Очистить session state, показать controlled redirect/login dialog | no | WARN | Не раскрывать причину revoke сверх safe key |
| `SESSION_CONTEXT_MISMATCH` | 403 | `errors.session.contextMismatch` | `node_device_id`/`client_device_id` не совпадает с session context | Modal с безопасным объяснением, без destructive logout | no | WARN | Mask device/session ids в логах |
| `PERMISSION_DENIED` | 403 | `errors.permission.denied` | Actor session активна, но permission отсутствует | Modal "Недостаточно прав", не делать logout | no | WARN | Не возвращать required permission id, если это раскрывает внутреннюю политику |
| `FORBIDDEN` | 403 | `errors.permission.denied` | Общий ожидаемый отказ доступа | Modal "Недостаточно прав" | no | WARN | Без секретов |
| `NOT_FOUND` | 404 | `errors.notFound` | Сущность не найдена | Compact notice или modal для blocking flow | no | WARN | Не раскрывать SQL/query details |
| `CONFLICT` | 409 | `errors.conflict.default` | Нарушен текущий state/business invariant | Modal с предложением обновить состояние | no | WARN | Без internal state dump |
| `DUPLICATE_PIN` | 409 | `errors.conflict.duplicatePin` | PIN совпал с несколькими active employees | Modal с бизнес-сообщением, не показывать PIN | no | WARN | PIN не возвращается и не логируется |
| `ACTIVE_PRECHECK_CONFLICT` | 409 | `errors.conflict.activePrecheck` | Для заказа уже есть активный precheck | Modal с предложением обновить заказ/precheck | no | WARN | Без raw domain error |
| `DUPLICATE_COMMAND` | 409 | `errors.conflict.duplicateCommand` | Повтор command id/idempotency conflict | Modal/notice, не auto-retry write command | no | WARN | Без raw payload |
| `RATE_LIMITED` | 429 | `errors.rateLimit` | Превышен лимит PIN login attempts | Notice/modal с рекомендацией подождать | yes, вручную | WARN | PIN не возвращается и не логируется |
| `INTERNAL_ERROR` | 500 | `errors.server` | Неожиданная или инфраструктурная ошибка | Modal с generic текстом и support code | no для write, осторожно для read/status | ERROR | Stack trace и SQL details только в backend log |

## Logging behavior

implemented now:

- HTTP error path пишет structured log с `request_id`, `operation=http.error`, `action`, `result=rejected`, `status`, `error_code`, masked `node_device_id`, `client_device_id`, `session_id`, `actor_employee_id`, `internal_error`.
- Panic recovery возвращает safe `INTERNAL_ERROR` response и пишет stack trace только в backend log.
- Request audit middleware читает stable `X-Error-Code`, поэтому audit logs не деградируют до generic `HTTP_403`.

## Retry policy

implemented now:

- UI может retry network/timeout/server для safe read/status запросов.
- UI mutation defaults отключают auto-retry, чтобы не повторять финансовые write commands без explicit idempotency policy.
- `429 RATE_LIMITED` считается retryable только вручную после ожидания.

out of scope:

- расширенный per-endpoint retry-after contract;
- локализованные backend message strings вместо `message_key`;
- раскрытие internal validation field map, если оно может вернуть raw sensitive payload.
