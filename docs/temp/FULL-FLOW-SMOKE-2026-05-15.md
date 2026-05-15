# Full-flow smoke audit на 2026-05-15

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
