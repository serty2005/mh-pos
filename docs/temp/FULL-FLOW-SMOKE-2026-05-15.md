# Full-flow smoke audit на 2026-05-15

## Повторный ручной прогон после очистки volumes и контейнеров

Статус: ручная проверка через Playwright по запущенному local stack. Bootstrap-скрипты не использовались. Runtime backend code не изменялся. Продажа и возврат заказа заблокированы на этапе открытия личной смены.

Проверялся путь:

1. Cloud UI: создание ресторана, роли, сотрудника, блюда, зала, стола и menu item через формы UI.
2. Cloud UI: генерация pairing-code, публикация master data package.
3. POS UI: регистрация Edge через pairing-code.
4. POS UI: PIN login сотрудника.
5. POS UI: открытие личной смены, далее планировались cash session, стол, заказ, пречек, оплата, закрытый заказ и возврат.

Фактически выполнено:

- Cloud UI создан ресторан `Manual Flow 1778857981075`;
- создана роль `Flow Manager` с POS permissions;
- создан сотрудник `Flow Operator`, PIN в отчете не сохраняется;
- создано блюдо `Manual Dish`, зал `Main Hall`, стол `T1`, menu item с ценой `1234 RUB`;
- pairing-code сгенерирован через Cloud UI, значение в отчете не сохраняется;
- master data опубликованы, publication version `2`, статус `published`;
- POS Edge зарегистрирован через pairing-code и получил `paired` state;
- POS Edge применил snapshot: `restaurants`, `staff`, `catalog`, `floor`, `menu` имеют `status = applied`;
- POS UI выполнил PIN login и открыл cashier terminal для `Flow Operator`.

### Critical blocker: открытие смены

`POST /api/v1/employee-shifts/open` возвращает `400 VALIDATION_FAILED`.

UI показал безопасный i18n-диалог с correlation id и не показал raw backend/internal error. В логах POS Edge причина:

```text
internal_error="invalid domain operation: invalid restaurant timezone"
```

Root cause подтвержден:

- Cloud UI по умолчанию создает ресторан с timezone `Europe/Moscow`;
- POS Edge read model содержит `timezone = Europe/Moscow`;
- `pos-backend/internal/pos/app/shared/business_date.go` вычисляет `business_date_local` через `time.LoadLocation(strings.TrimSpace(restaurant.Timezone))`;
- контейнер `mh-pos-local-pos-edge-1` не содержит zoneinfo:

```text
/usr/share/zoneinfo/Europe/Moscow: No such file or directory
/usr/share/zoneinfo/UTC: No such file or directory
```

Dockerfile использует `alpine:3.22`, но не устанавливает `tzdata`:

```text
pos-backend/docker/Dockerfile:8 FROM alpine:3.22
```

Это блокирует canonical sale flow до открытия кассовой смены, поэтому продажа и возврат в этом чистом прогоне не выполнялись. Live SQLite не правилась, чтобы не маскировать production-way blocker.

### Request/log avalanche после pairing

После успешной регистрации через pairing-code лавина воспроизводится на чистых volumes.

За последние 10 минут на POS Edge:

- `EdgeNodePaired`: `185`;
- `SYNC_DIRECTION_BLOCKED`: `92`;
- `employee-shifts/open` log lines: `3`.

Характер логов:

```text
domain action committed action="EdgeNodePaired"
POS sync sender suspended outbox message error_code="SYNC_DIRECTION_BLOCKED" event_type="EdgeNodePaired" reason="outbox row direction is \"local_only\""
```

Вероятная причина остается прежней: polling/provisioning повторно проходит already paired state, снова пишет `EdgeNodePaired` и создает local-only outbox rows, которые затем обрабатываются sender-ом как blocked.

### Snapshot/read-model observations

POS Edge read model после snapshot:

```json
{
  "edge_node_identity": {
    "node_device_id": "bb749310-bc0a-465e-b1d2-87cf10a4426c",
    "restaurant_id": "29f9856e-67c0-4a33-b2ca-bf1aa0a0b5fa",
    "status": "paired"
  },
  "restaurant": {
    "name": "Manual Flow 1778857981075",
    "timezone": "Europe/Moscow",
    "currency": "RUB",
    "business_day_mode": "standard",
    "business_day_boundary_local_time": "04:00",
    "active": 0,
    "cloud_version": 2
  },
  "device": {
    "restaurant_id": "29f9856e-67c0-4a33-b2ca-bf1aa0a0b5fa",
    "active": 1
  }
}
```

Для этого чистого прогона device/identity mismatch не воспроизвелся: `devices.restaurant_id` совпадает с `edge_node_identity.restaurant_id`, PIN login прошел. Но ресторан по-прежнему приходит на Edge как `active = 0`, что остается отдельным contract/read-model risk.

### Browser diagnostics

Machine-readable отчет по ручному прогону:

- `docs/temp/full-flow-manual-browser-diagnostics-2026-05-15.json`.

Скриншоты сохранены вне репозитория:

- `%TEMP%\mh-pos-manual-flow-2026-05-15\01-cloud-launch.png`;
- `%TEMP%\mh-pos-manual-flow-2026-05-15\02-cloud-pairing-code.png`;
- `%TEMP%\mh-pos-manual-flow-2026-05-15\03-cloud-published.png`;
- `%TEMP%\mh-pos-manual-flow-2026-05-15\06-pos-after-login.png`.

Зафиксированные browser/API noise:

- Cloud UI до первой публикации получает ожидаемый, но шумный `404` на `GET /restaurants/{id}/master-data/publication-state`; UI обрабатывает безопасно, браузер пишет `Failed to load resource`;
- POS UI после логина получает ожидаемый, но шумный `404` на `GET /employee-shifts/current`; UI обрабатывает безопасно, браузер пишет `Failed to load resource`.

### Что нужно исправить перед повтором sale/refund

1. Добавить timezone data в POS Edge runtime image либо встроить Go tzdata так, чтобы `time.LoadLocation("Europe/Moscow")` работал в контейнере.
2. Сделать provisioning/pairing idempotent после paired state и прекратить повторную запись `EdgeNodePaired` без изменения assignment/snapshot checkpoint.
3. Исправить Cloud -> Edge restaurant projection: активный опубликованный ресторан не должен попадать в Edge read model как `active = 0`, либо контракт должен явно описывать другое поведение.
4. Решить, должны ли expected empty states (`publication-state`, `current shift`) возвращать 404 с browser noise или отдельный безопасный empty-state контракт.
5. После исправления повторить ручной path: pairing-code -> PIN login -> open shift -> open cash session -> table -> order -> precheck -> payment -> closed order -> refund.

## Предыдущий bootstrap-прогон

Статус: диагностический отчет по запущенному local stack. Runtime backend code не изменялся. Продажа на POS Edge заблокирована до входа кассира.

## Объем проверки

Проверялся путь:

1. Cloud: создание ресторана, ролей, сотрудников, зала, стола, каталога, menu items.
2. Cloud: публикация master data package.
3. Cloud/Edge: assignment текущего Edge node к ресторану и получение snapshot.
4. POS Edge: применение snapshot, pairing state, PIN login.
5. POS Edge: тестовая продажа.

Фронты были доступны:

- Cloud UI: `http://127.0.0.1:5174/`;
- POS UI: `http://127.0.0.1:5173/login`;
- Cloud API: `http://localhost:8090/api/v1`;
- POS Edge API: `http://localhost:8080/api/v1`.

## Выполненные шаги

Создан тестовый ресторан `Codex Full Flow 20260515170236` через production-way bootstrap сценарий:

```powershell
.\scripts\bootstrap-production-way.ps1 -CloudBaseUrl 'http://localhost:8090' -EdgeBaseUrl 'http://localhost:8080' -UiBaseUrl 'http://localhost:5173' -RestaurantName 'Codex Full Flow 20260515170236' -CashierPin '[REDACTED]' -ManagerPin '[REDACTED]' -RunRuntimeSmoke -VerboseOutput
```

Cloud side после bootstrap и повторной публикации содержит готовый набор данных для выбранного ресторана:

- roles: `2`;
- employees: `2`;
- catalog/items: `2`;
- halls: `1`;
- tables: `1`;
- menu/items: `2`;
- publication state: `published`, version `2`;
- publication counts: restaurants `1`, roles `2`, employees `2`, catalog_items `2`, menu_items `2`, halls `1`, tables `1`.

Edge assignment status для текущего `node_device_id` возвращал `assigned`, `restaurant_id` нового ресторана и snapshot URL. Credentials/token в отчете не сохранялись.

## Blocker

Тестовая продажа на POS Edge не дошла до сценария `смена -> стол -> заказ -> пречек -> оплата`, потому что `POST /api/v1/auth/pin-login` возвращает `403 FORBIDDEN`.

Безопасность UI проверена отдельно: POS UI показывает i18n-диалог «Доступ / Недостаточно прав» и support correlation id. Raw backend text `node device is archived or mismatched` в UI не показан.

API response:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message_key": "errors.permission.denied",
    "correlation_id": "..."
  }
}
```

Root cause по локальной SQLite read model:

```json
{
  "edge_node_identity": [
    {
      "node_device_id": "f147ae4f-a149-497e-8e38-89c207b958ad",
      "restaurant_id": "09a844a9-dc4f-4279-8c51-5e2dcebc4486",
      "status": "paired"
    }
  ],
  "devices": [
    {
      "id": "f147ae4f-a149-497e-8e38-89c207b958ad",
      "restaurant_id": "0f6d6421-e3d5-460d-9f26-f8f4f08dd372",
      "active": 1
    }
  ]
}
```

То есть `edge_node_identity.restaurant_id` уже указывает на новый ресторан, а строка `devices.restaurant_id` осталась на старом ресторане. Auth service затем блокирует вход:

- `pos-backend/internal/pos/app/auth/service.go:78` читает device по `identity.NodeDeviceID`;
- `pos-backend/internal/pos/app/auth/service.go:82` требует `device.Active && device.RestaurantID == identity.RestaurantID`;
- при mismatch возвращается `domain.ErrForbidden`.

Причина mismatch в pairing path:

- `pos-backend/internal/pos/app/device/service.go:91-110` создает `devices` row только если device не найден;
- при перепривязке уже существующего `node_device_id` к другому ресторану существующая строка `devices` не обновляется;
- `pos-backend/internal/pos/app/device/service.go:111-118` при этом обновляет `edge_node_identity` и пишет новый `EdgeNodePaired`.

## Request avalanche

За 20 минут после assignment/pairing зафиксирована лавина polling/write events:

POS Edge logs:

- `EdgeNodePaired`: `240`;
- `SYNC_DIRECTION_BLOCKED` для `EdgeNodePaired` с reason `outbox row direction is "local_only"`: `240`;
- `pin-login` mismatch: `3`.

Cloud API logs:

- `POST /api/v1/devices/register`: `263`;
- `GET /api/v1/devices/{node_device_id}/assignment-status`: `264`;
- snapshot downloads: `243`;
- Cloud API errors: `0`.

Вероятная причина: `PollAssignment` не останавливается после paired state и каждый цикл снова вызывает `finishAssigned`, скачивает snapshot, применяет его и вызывает `PairEdgeNode`.

Кодовые точки:

- `pos-backend/internal/pos/app/provisioning/service.go:151-173`: при `assignment.Status == "assigned"` всегда вызывает `finishAssigned`;
- `pos-backend/internal/pos/app/provisioning/service.go:205-236`: `finishAssigned` каждый раз скачивает snapshot, применяет master data и вызывает `pair`;
- `pos-backend/internal/pos/app/device/service.go:114-118`: `PairEdgeNode` каждый раз пишет `EdgeNodePaired` в outbox.

## Snapshot/read-model issues

Cloud publication/snapshot сейчас не несет stream `devices`, хотя POS Edge `ApplyMasterDataCommand` поддерживает `devices`.

Кодовые точки:

- `pos-backend/internal/pos/app/mastersync/service.go`: `ApplyMasterDataCommand.Devices` существует;
- `cloud-backend/internal/masterdata/app/service.go:1682-1788`: stream packages формируются только для `restaurants`, `staff`, `catalog`, `floor`, `menu`, `pricing_policy`;
- `devices` stream не формируется, поэтому snapshot не может исправить `devices.restaurant_id` на Edge.

Дополнительно Cloud projection ресторана в Edge snapshot не содержит `active/status`:

- `cloud-backend/internal/masterdata/domain/types.go:420-429`: `EdgeRestaurant` не имеет `active/status`;
- `cloud-backend/internal/masterdata/app/service.go:1790-1805`: `edgeRestaurants` не выставляет активность.

Фактическое состояние POS Edge после snapshot:

```json
{
  "restaurants": [
    {
      "id": "09a844a9-dc4f-4279-8c51-5e2dcebc4486",
      "name": "Codex Full Flow 20260515170236",
      "active": 0,
      "cloud_version": 2
    },
    {
      "id": "0f6d6421-e3d5-460d-9f26-f8f4f08dd372",
      "name": "testrest",
      "active": 0,
      "cloud_version": 2
    }
  ]
}
```

Это не было непосредственной причиной `pin-login` 403, но является отдельным read-model risk для дальнейших Edge flows.

## Browser diagnostics

Сохранен machine-readable отчет:

- `docs/temp/full-flow-browser-diagnostics-2026-05-15.json`.

Cloud UI на первичной загрузке launch plan за 12 секунд:

- API requests total: `16`;
- повторов одного endpoint не обнаружено;
- failed requests: `0`;
- console errors/warnings: `0`.

Важно: Cloud UI без сохраненного выбора ресторана выбрал первый ресторан из списка (`testrest`), а не последний созданный `Codex Full Flow ...`. Для smoke/оператора это риск неверной интерпретации readiness в multi-restaurant окружении.

POS UI login:

- API requests total: `2`;
- `GET /system/pairing-status`: `1`;
- `POST /auth/pin-login`: `1`;
- response: `403`;
- UI показывает safe i18n dialog;
- raw text `node device is archived or mismatched` не показан.

POS pair page:

- API requests total: `2`;
- failed requests: `0`;
- console errors/warnings: `0`.

## Выводы

### Критично

Перепривязка уже существующего POS Edge node к новому ресторану ломает PIN login, потому что `edge_node_identity` и `devices` расходятся по `restaurant_id`.

Рекомендуемое исправление: сделать pairing idempotent и явно синхронизировать локальную `devices` row при смене restaurant assignment либо доставлять authoritative assigned device stream из Cloud snapshot. Решение должно сопровождаться backend tests на re-pair same node to another restaurant.

### Высокий риск

Provisioning polling после assigned/paired state вызывает повторное скачивание snapshot и повторную запись `EdgeNodePaired`, что создает request/log/outbox avalanche.

Рекомендуемое исправление: добавить idempotency guard в `PollAssignment/finishAssigned`: если Edge уже paired с тем же `node_device_id`, `restaurant_id`, `cloud_url`, `checkpoint_token/cloud_version`, не скачивать snapshot и не писать новый `EdgeNodePaired`. Отдельно проверить, что `local_only` outbox rows не отправляются Cloud sender-ом как rejected/suspended на каждом цикле.

### Средний риск

Cloud UI launch readiness может показывать readiness не того ресторана в multi-restaurant окружении, если оператор не заметил выбранный restaurant filter. Это не backend blocker, но smoke и onboarding flow должны явно подсвечивать выбранный ресторан и последнюю созданную/назначенную связку Edge.

### Средний риск

Cloud -> Edge `restaurants` projection не переносит active/status, поэтому POS Edge хранит рестораны из snapshot как inactive. Нужна явная contract-правка: либо `active/status` добавляется в Edge projection, либо POS ingest должен трактовать наличие ресторана в опубликованном active package как active.

## Не завершено

Тестовая продажа на POS Edge не выполнена: canonical path заблокирован на `pin-login`. Ручная правка live SQLite не выполнялась, чтобы не маскировать production-way blocker.

После исправления blocker нужно повторить smoke:

1. PIN login cashier/manager для нового ресторана.
2. Открытие смены и cash session.
3. Выбор стола.
4. Добавление опубликованного menu item.
5. Пречек.
6. Оплата.
7. Закрытый чек, reprint/refund wording.
8. Проверка, что polling не создает повторные snapshot downloads/outbox events.
