# Runtime Config

Реализовано сейчас: Go runtime-сервисы поддерживают единый порядок загрузки конфигурации:

1. встроенные defaults в entrypoint сервиса;
2. переменные окружения;
3. внешний JSON-файл конфигурации.

Файловый конфиг имеет приоритет над переменными окружения. Это позволяет держать production/dev параметры в файле и использовать env только как bootstrap или emergency override нижнего уровня.

## Формат

Реализовано сейчас: файл конфигурации является плоским JSON object, где ключи совпадают с именами поддерживаемых env-переменных сервиса.

```json
{
  "POS_HTTP_ADDR": ":8080",
  "POS_SQLITE_PATH": "data/pos-edge.db",
  "POS_SQLITE_ARCHIVE_DIR": "data/archives",
  "POS_SYNC_SENDER_ENABLED": true,
  "POS_SYNC_SENDER_BATCH_SIZE": 25,
  "POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES": 120
}
```

Значения допускаются как JSON string, number, boolean или null. Duration-параметры задаются строками Go duration, например `"2s"` или `"5m"`.

## Пути

Реализовано сейчас:

- POS Edge читает `POS_CONFIG_PATH`; default optional path: `config/pos-edge.json`;
- Cloud Backend читает `CLOUD_CONFIG_PATH`; default optional path: `config/cloud-api.json`;
- License Server читает `LICENSE_CONFIG_PATH`; default optional path: `config/license-api.json`.

Если `*_CONFIG_PATH` задан явно, файл обязателен и ошибка чтения останавливает startup fail-fast. Если `*_CONFIG_PATH` не задан и default-файл отсутствует, сервис продолжает старт с env/defaults.

Примеры файлов находятся рядом с сервисами:

- `pos-backend/config/pos-edge.example.json`;
- `cloud-backend/config/cloud-api.example.json`;
- `license-server/config/license-api.example.json`.

Licensing authority config:

- `LICENSE_ADMIN_TOKEN` — обязательный provider secret только для License Server update API;
- `LICENSE_TENANT_ID` и `LICENSE_SERVER_ID` — runtime scope Cloud/Edge;
- `LICENSE_FALLBACK_SERVER_IDS` — реализовано сейчас для POS Edge как deployment-configured список fallback `server_id`, разделенный запятыми, пробелами или `;`; используется только если primary snapshot отсутствует или authority не вернул валидный snapshot, но не обходит явный primary `revoked`/expired snapshot;
- `LICENSE_STALE_GRACE_SECONDS` — provider deployment grace при недоступности authority; клиентский UI его не изменяет.

Контракт и module IDs описаны в `docs/backend/LICENSE-ENTITLEMENTS.md`.

Реализовано сейчас: loader находится в общем локальном Go module `shared/platform` (`module mh-pos-platform`) и подключается сервисами через `replace`.

Реализовано сейчас: POS Edge storage archive export использует `POS_SQLITE_ARCHIVE_DIR`. Если значение не задано, entrypoint выбирает `archives` рядом с active SQLite data directory из `POS_SQLITE_PATH`; сам export не пишет файлы внутрь `.db` file и не запускает destructive apply. Физическое удаление и `VACUUM` выполняются только отдельным `POST /api/v1/storage/archive/apply-plan` после verified archive и runtime safety gate.

Реализовано сейчас: `POS_RECIPE_SUGGESTION_MAX_TIME_DELTA_MINUTES` задает положительный integer limit для `RecipeChangeSuggested.prep_time_delta_minutes`. Если ключ отсутствует или некорректен, POS Edge использует default `120`.

Вне текущего объема: горячая перезагрузка runtime-конфига без рестарта процесса и хранение secrets в зашифрованном vault.

## Sync Runtime

Реализовано сейчас:

- `POS_CLOUD_SYNC_URL` может указывать на legacy `/api/v1/sync/edge-events`; POS Edge client автоматически использует `/api/v1/sync/exchange` для authenticated exchange, когда provisioning state содержит `node_token`.
- `POS_SYNC_SENDER_ENABLED`, `POS_SYNC_SENDER_BATCH_SIZE`, `POS_SYNC_SENDER_POLL_INTERVAL`, `POS_SYNC_SENDER_CLOUD_PULL_INTERVAL`, `POS_SYNC_SENDER_RECLAIM_AFTER`, `POS_SYNC_SENDER_SEND_TIMEOUT`, `POS_SYNC_SENDER_EMERGENCY_PENDING_THRESHOLD` и `POS_SYNC_SENDER_CLOUD_PACKAGE_BURST_THRESHOLD` управляют worker cycle.
- `POS_SYNC_SENDER_POLL_INTERVAL` задает строгую периодику worker-а. `POS_SYNC_SENDER_POLL_JITTER` сохранен как compatibility config key, но не добавляет случайную задержку к sync cycle.
- `POS_SYNC_SENDER_EMERGENCY_PENDING_THRESHOLD` включает немедленную следующую итерацию, если число pending Edge -> Cloud outbox rows достигло high-watermark.
- `POS_SYNC_SENDER_CLOUD_PACKAGE_BURST_THRESHOLD` включает немедленный следующий Cloud pull после bounded Cloud -> Edge response, если Cloud вернул не меньше указанного числа packages.
- `POS_SYNC_SENDER_CLOUD_PULL_INTERVAL` ограничивает пустой authenticated exchange без Edge outbox; если local outbox содержит sendable rows, exchange выполняется на ближайшем worker tick и не ждет этот interval.
- `node_token` хранится в local Edge provisioning state после Cloud/License provisioning и не выводится в HTTP responses или structured logs.
- Повторный Cloud `assignment-status` не должен ротировать уже выданный `node_token`; иначе POS Edge продолжит отправлять сохраненный token и получит `401 SYNC_UNAUTHORIZED`.
- `CLOUD_SYNC_MAX_CLOUD_PACKAGES_PER_EXCHANGE` ограничивает число Cloud -> Edge packages в одном `sync/exchange` response. Остальные changed streams передаются следующими exchange-сессиями после применения предыдущей порции на Edge.

Вне текущего объема:

- публикация `node_token` через operator UI/API;
- production secret vault для Edge credentials.
