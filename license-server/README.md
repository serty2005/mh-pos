# MyHoReCa License Server Stub

`license-server` - отдельный локальный stub для production-like Option B pairing code flow. Он не является Cloud master-data authority; задача сервиса - принять pairing code metadata от Cloud и вернуть POS Edge `cloud_url`/`pairing_id` по введенному коду.

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
  "pairing_id": "pairing-id",
  "instance_id": "cloud-instance-id",
  "cloud_url": "http://localhost:8090",
  "restaurant_id": "restaurant-id",
  "expires_at": "2026-05-09T12:00:00Z"
}
```

Resolve pairing code from Edge:

```http
POST /api/v1/pairing-codes/resolve
```

```json
{
  "pairing_code": "123456"
}
```

Реализовано сейчас:

- pairing code хранится как SHA-256 hash;
- code имеет TTL и возвращает `PAIRING_CODE_EXPIRED` после истечения;
- resolve возвращает только `cloud_url`, `pairing_id` и restaurant id; node credentials выдаются Cloud после encrypted consume;
- successful resolve не переводит code в `consumed`, потому что финальное потребление выполняет Cloud после подтверждения Edge node id;
- ошибки возвращаются в structured envelope и выставляют `X-Error-Code`.

## Логирование

Реализовано сейчас:

- request audit пишет `operation=http.request`, `action`, `result`, `status`, `error_code`, `duration_ms` и `remote_ip`;
- register/resolve flow пишет `operation=license.pairing.register` или `operation=license.pairing.resolve`, `result`, masked `restaurant_id`, masked `pairing_id`, `pairing_code_present`, `pairing_code_length` и безопасную `reason` для отказов;
- безопасные причины отказа включают `registration_required_fields_missing`, `registration_expires_at_not_future`, `pairing_code_required`, `pairing_code_not_found` и `pairing_code_expired`;
- pairing code, node token и credentials payload не пишутся в логи.

Вне текущего объема: настоящий внешний licensing/billing, multi-tenant admin UI и production auth perimeter между Cloud и License Server.
