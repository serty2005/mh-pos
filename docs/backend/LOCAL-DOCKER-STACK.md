# Локальный Docker stack без POS UI

Документ описывает локальный запуск `cloud-postgres`, `cloud-api`, `license-api` и `pos-edge` через общий Docker Compose. `pos-ui` намеренно не входит в compose: его удобнее запускать локально через Vite/Quasar dev server.

## Что запускается

Реализовано сейчас:

- `cloud-postgres` - PostgreSQL 16 для Cloud Backend;
- `cloud-api` - Cloud Sync Receiver и Cloud master-data authority;
- `license-api` - локальный License Server stub для Option B pairing code flow;
- `pos-edge` - POS Edge backend с SQLite, IANA timezone data (`tzdata`) для `business_date_local` и включенным sync sender.

Именованные volumes:

- `cloud_postgres_data` - данные PostgreSQL;
- `cloud_api_data` - Cloud runtime data и PostgreSQL backup snapshots;
- `license_sqlite_data` - SQLite БД license-server;
- `pos_edge_sqlite_data` - SQLite БД POS Edge и backup files.

## Конфиги

Docker-oriented JSON-конфиги лежат рядом с сервисами:

- `cloud-backend/config/cloud-api.docker.json`;
- `license-server/config/license-api.docker.json`;
- `pos-backend/config/pos-edge.docker.json`.

Важные значения:

```text
CLOUD_POSTGRES_HOST_PORT=5432
CLOUD_POSTGRES_DSN=postgres://postgres:postgres@cloud-postgres:5432/mh_pos_cloud?sslmode=disable
CLOUD_PUBLIC_URL=http://cloud-api:8090
LICENSE_SERVER_URL=http://license-api:8095
POS_CLOUD_SYNC_URL=http://cloud-api:8090/api/v1/sync/edge-events
POS_SQLITE_PATH=/app/data/pos-edge.db
```

Файловый конфиг имеет приоритет над env. Общий контракт описан в `docs/backend/RUNTIME-CONFIG.md`.

## Запуск

Из корня репозитория:

```powershell
$env:PYTHONIOENCODING='utf-8'
docker compose -f docker-compose.local.yml up --build -d
```

Если локальный `5432` уже занят другой PostgreSQL-инстанцией, можно поменять только host binding, не меняя внутренний DSN между контейнерами:

```powershell
$env:CLOUD_POSTGRES_HOST_PORT='55432'
docker compose -f docker-compose.local.yml up --build -d
```

Проверка health endpoints:

```powershell
Invoke-RestMethod http://localhost:8090/health
Invoke-RestMethod http://localhost:8095/health
Invoke-RestMethod http://localhost:8080/health
```

Логи:

```powershell
docker compose -f docker-compose.local.yml logs -f cloud-api
docker compose -f docker-compose.local.yml logs -f pos-edge
docker compose -f docker-compose.local.yml logs -f license-api
```

Остановка без удаления данных:

```powershell
docker compose -f docker-compose.local.yml down
```

Полный reset локальных Docker БД и файлов является destructive-by-design операцией:

```powershell
docker compose -f docker-compose.local.yml down -v
```

## Заполнение Cloud и проверка POS

Для быстрого smoke/e2e пути через Cloud CRUD, publication, Cloud -> Edge snapshot и POS login:

```powershell
.\scripts\cloud-masterdata-e2e.ps1 `
  -CloudApiBase "http://localhost:8090/api/v1" `
  -PosApiBase "http://localhost:8080/api/v1"
```

Скрипт создает ресторан, роль, сотрудника с PIN `1357`, catalog/menu item, modifier group/option/binding, публикует typed Cloud -> POS Edge snapshot, применяет его на POS Edge без PowerShell field stripping, выполняет local pairing и проверяет, что POS видит Cloud-created данные.

Для production-like Zero-to-Cashier через Cloud Approve:

```powershell
.\scripts\zero-to-cashier-option-a.ps1 `
  -CloudApiBase "http://localhost:8090/api/v1" `
  -PosApiBase "http://localhost:8080/api/v1"
```

Для production-like Zero-to-Cashier через License Code:

```powershell
.\scripts\zero-to-cashier-option-b.ps1 `
  -CloudApiBase "http://localhost:8090/api/v1" `
  -PosApiBase "http://localhost:8080/api/v1"
```

Оба Zero-to-Cashier скрипта создают Cloud master data, привязывают POS Edge и проверяют PIN login. По умолчанию cashier PIN: `1111`.

## Ручная проверка через POS UI

После запуска Docker stack запусти UI локально:

```powershell
cd pos-ui
npm install
npm run dev
```

Открой `http://localhost:5173`. POS UI ходит в POS Edge на `http://localhost:8080/api/v1`.

Если перед этим был выполнен `zero-to-cashier-option-a.ps1` или `zero-to-cashier-option-b.ps1`, Edge уже paired, а войти можно PIN `1111`. Если был выполнен `cloud-masterdata-e2e.ps1`, PIN по умолчанию `1357`.

## Проверка PostgreSQL и sync

Cloud operational events:

```powershell
docker compose -f docker-compose.local.yml exec cloud-postgres `
  psql -U postgres -d mh_pos_cloud `
  -c "select event_type, count(*) from cloud_operational_events group by event_type order by event_type;"
```

Последние Cloud receipts:

```powershell
docker compose -f docker-compose.local.yml exec cloud-postgres `
  psql -U postgres -d mh_pos_cloud `
  -c "select idempotency_key, event_type, cloud_received_at from cloud_edge_event_receipts order by cloud_received_at desc limit 10;"
```

POS sync sender доставляет Edge -> Cloud operational rows автоматически, когда `POS_SYNC_SENDER_ENABLED=true`. Недоступность Cloud не блокирует POS runtime writes.

реализовано сейчас:

- после pairing через license code или Cloud assignment `pos-edge` прекращает повторный device registration/snapshot provisioning poll;
- локальные Edge outbox rows отправляются через authenticated `sync/exchange` на ближайшем worker tick;
- пустой `sync/exchange` для Cloud -> Edge pull ограничен отдельным interval, чтобы локальный Docker stack не создавал шумный Cloud access log при отсутствии локальных событий.
- Cloud UI после successful master-data CRUD автоматически вызывает publication API от имени `cloud-ui`; ручная публикация в разделе Publication state остается доступна для явного operator checkpoint.

вне текущего объема: запуск `pos-ui` внутри этого compose и production auth perimeter для Cloud/License API.
