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
  "POS_SYNC_SENDER_BATCH_SIZE": 25
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

Реализовано сейчас: loader находится в общем локальном Go module `shared/platform` (`module mh-pos-platform`) и подключается сервисами через `replace`.

Реализовано сейчас: POS Edge storage archive export использует `POS_SQLITE_ARCHIVE_DIR`. Если значение не задано, entrypoint выбирает `archives` рядом с active SQLite data directory из `POS_SQLITE_PATH`; export не пишет файлы внутрь `.db` file и не запускает destructive cleanup.

Вне текущего объема: горячая перезагрузка runtime-конфига без рестарта процесса и хранение secrets в зашифрованном vault.

## Sync Runtime

Реализовано сейчас:

- `POS_CLOUD_SYNC_URL` может указывать на legacy `/api/v1/sync/edge-events`; POS Edge client автоматически использует `/api/v1/sync/exchange` для authenticated exchange, когда provisioning state содержит `node_token`.
- `POS_SYNC_SENDER_ENABLED`, `POS_SYNC_SENDER_BATCH_SIZE`, `POS_SYNC_SENDER_POLL_INTERVAL`, `POS_SYNC_SENDER_POLL_JITTER`, `POS_SYNC_SENDER_CLOUD_PULL_INTERVAL`, `POS_SYNC_SENDER_RECLAIM_AFTER` и `POS_SYNC_SENDER_SEND_TIMEOUT` управляют worker cycle.
- `POS_SYNC_SENDER_CLOUD_PULL_INTERVAL` ограничивает пустой authenticated exchange без Edge outbox; если local outbox содержит sendable rows, exchange выполняется на ближайшем worker tick и не ждет этот interval.
- `node_token` хранится в local Edge provisioning state после Cloud/License provisioning и не выводится в HTTP responses или structured logs.

Вне текущего объема:

- публикация `node_token` через operator UI/API;
- production secret vault для Edge credentials.
