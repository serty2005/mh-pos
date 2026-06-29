# MyHoReCa License Server

`license-server` - внешний authority для tenant/server entitlement snapshots и production-like Option B pairing code flow. Он не является Cloud master-data authority: Cloud остается владельцем master data, а License Server отвечает за pairing code metadata и module entitlements.

Canonical licensing contract, module IDs и правила stale grace описаны в `../docs/backend/LICENSE-ENTITLEMENTS.md`.

Native Linux deployment для Ubuntu 24.04 описан в `../docs/deployment/LICENSE-SERVER-LINUX.md`.

## Запуск

```powershell
cd license-server
go mod tidy
go test ./...
$env:LICENSE_CONFIG_PATH="config/license-api.json" # optional; файл имеет приоритет над env
$env:LICENSE_HTTP_ADDR=":8095"
$env:LICENSE_SQLITE_PATH="data/license-server.db"
$env:LICENSE_SUPER_ADMIN_LOGIN="admin"
$env:LICENSE_SUPER_ADMIN_PASSWORD="replace-with-strong-password"
go run ./cmd/license-api
```

Реализовано сейчас: License Server также читает optional `config/license-api.json`; пример полного файла находится в `config/license-api.example.json`. Если `LICENSE_CONFIG_PATH` задан явно, файл обязателен. Порядок приоритета: defaults -> env -> JSON-файл. Общий контракт описан в `../docs/backend/RUNTIME-CONFIG.md`.

Перед startup migration сервер делает backup SQLite `.db/.db-wal/.db-shm` в `LICENSE_SQLITE_BACKUP_DIR`. Обновление binary/config не требует сброса данных: существующие таблицы сохраняются, недостающие таблицы добавляются программно.

## Термины

- Контрагент — юридическое лицо или клиент, который платит за обслуживание; в API это `tenant_id`.
- Сервер — конкретный Cloud или POS Edge runtime под контрагентом; в API это `server_id`.
- Один контрагент может иметь несколько серверов, но сервер относится только к одному контрагенту.

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

Entitlement snapshot runtime read:

```http
GET /api/v1/entitlements/{tenant_id}/{server_id}
```

Provider update:

```http
PUT /api/v1/entitlements/{tenant_id}/{server_id}
```

Реализовано сейчас:

- snapshot хранится по `(tenant_id, server_id)`;
- `version` должен монотонно расти;
- `status` принимает `active` или `revoked`;
- `entitlements` содержит canonical hyphen IDs из `../docs/backend/LICENSE-ENTITLEMENTS.md`;
- update/list требуют входа super-admin через `POST /api/v1/admin/login`;
- operator UI доступен на `/` и `/admin`, показывает connected servers, поиск по `tenant_id`, выбор сервера из списка, module toggles и presets;
- runtime read фиксирует подключившийся сервер в списке connected servers и не раскрывает operator credentials.

Admin login:

```http
POST /api/v1/admin/login
```

```json
{
  "username": "admin",
  "password": "replace-with-strong-password"
}
```

Логин и пароль первого super-admin задаются через `LICENSE_SUPER_ADMIN_LOGIN` и `LICENSE_SUPER_ADMIN_PASSWORD`. Password хранится в SQLite как PBKDF2-SHA256 hash с salt; plaintext password не сохраняется.

## Логирование

Реализовано сейчас:

- request audit пишет `operation=http.request`, `action`, `result`, `status`, `error_code`, `duration_ms` и `remote_ip`;
- register/resolve flow пишет `operation=license.pairing.register` или `operation=license.pairing.resolve`, `result`, masked `restaurant_id`, masked `pairing_id`, `pairing_code_present`, `pairing_code_length` и безопасную `reason` для отказов;
- безопасные причины отказа включают `registration_required_fields_missing`, `registration_expires_at_not_future`, `pairing_code_required`, `pairing_code_not_found` и `pairing_code_expired`;
- pairing code, node token и credentials payload не пишутся в логи.

Вне текущего объема: настоящий billing provider, multi-tenant commercial admin UI и production auth perimeter между Cloud и License Server.
