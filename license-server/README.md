# MyHoReCa License Server Stub

`license-server` - отдельный локальный stub для production-like Option B pairing code flow. Он не является Cloud master-data authority; задача сервиса - принять pairing code/config от Cloud и одноразово отдать config POS Edge по коду.

## Запуск

```powershell
cd license-server
go mod tidy
go test ./...
$env:LICENSE_CONFIG_PATH="config/license-api.json" # optional; файл имеет приоритет над env
$env:LICENSE_HTTP_ADDR=":8095"
$env:LICENSE_SQLITE_PATH="data/license-server.db"
go run ./cmd/license-api
```

Реализовано сейчас: License Server также читает optional `config/license-api.json`; пример полного файла находится в `config/license-api.example.json`. Если `LICENSE_CONFIG_PATH` задан явно, файл обязателен. Порядок приоритета: defaults -> env -> JSON-файл. Общий контракт описан в `../docs/backend/RUNTIME-CONFIG.md`.

## API

Register pairing code from Cloud:

```http
POST /api/v1/pairing-codes
```

```json
{
  "pairing_code": "123456",
  "cloud_url": "http://localhost:8090",
  "restaurant_id": "restaurant-id",
  "node_device_id": "edge-node-id",
  "credentials": {
    "type": "node_token",
    "token": "plaintext-token-from-cloud-response"
  },
  "expires_at": "2026-05-09T12:00:00Z"
}
```

Resolve pairing code from Edge:

```http
POST /api/v1/pairing-codes/resolve
```

```json
{
  "pairing_code": "123456",
  "node_device_id": "edge-node-id"
}
```

Реализовано сейчас:

- pairing code хранится как SHA-256 hash;
- code имеет TTL и возвращает `PAIRING_CODE_EXPIRED` после истечения;
- successful resolve переводит code в `consumed`, повторный resolve возвращает `PAIRING_CODE_INVALID`;
- ошибки возвращаются в structured envelope и выставляют `X-Error-Code`.

## Логирование

Реализовано сейчас:

- request audit пишет `operation=http.request`, `action`, `result`, `status`, `error_code`, `duration_ms` и `remote_ip`;
- register/resolve flow пишет `operation=license.pairing.register` или `operation=license.pairing.resolve`, `result`, masked `restaurant_id`, masked `node_device_id`, `pairing_code_present`, `pairing_code_length` и безопасную `reason` для отказов;
- безопасные причины отказа включают `registration_required_fields_missing`, `registration_expires_at_not_future`, `pairing_code_required`, `pairing_code_not_found`, `pairing_code_consumed`, `pairing_code_expired` и `node_device_id_mismatch`;
- pairing code, node token и credentials payload не пишутся в логи.

Вне текущего объема: настоящий внешний licensing/billing, multi-tenant admin UI и production auth perimeter между Cloud и License Server.
