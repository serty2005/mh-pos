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

Вне текущего объема: горячая перезагрузка runtime-конфига без рестарта процесса и хранение secrets в зашифрованном vault.
