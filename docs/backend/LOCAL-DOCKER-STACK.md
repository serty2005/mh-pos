# Локальный Docker stack и UI devbox

Документ описывает локальный запуск `cloud-postgres`, `cloud-api`, `license-api` и `pos-edge` через общий Docker Compose. Для UI build/unit/e2e есть отдельный opt-in `devbox` profile: он не поднимается обычным backend stack и нужен только для разработки/проверок frontend.

## Что запускается

Реализовано сейчас:

- `cloud-postgres` - PostgreSQL 16 для Cloud Backend;
- `cloud-api` - Cloud Sync Receiver и Cloud master-data authority;
- `license-api` - локальный License Server stub для Option B pairing code flow;
- `pos-edge` - POS Edge backend с SQLite, IANA timezone data (`tzdata`) для `business_date_local` и включенным sync sender.

Опционально через profile `devbox`:

- `devbox` - Node/Playwright контейнер на `mcr.microsoft.com/playwright:v1.59.1-noble` с Chromium и Linux dependencies, установленными на этапе build image.

Именованные volumes:

- `cloud_postgres_data` - данные PostgreSQL;
- `cloud_api_data` - Cloud runtime data и PostgreSQL backup snapshots;
- `license_sqlite_data` - SQLite БД license-server;
- `pos_edge_sqlite_data` - SQLite БД POS Edge и backup files.
- `pos_ui_node_modules` - `pos-ui/node_modules` внутри Docker volume, а не Windows bind mount;
- `cloud_ui_node_modules` - `cloud-ui/node_modules` внутри Docker volume;
- `devbox_npm_cache` - npm cache пользователя `pwuser`.

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

Devbox E2E environment:

```text
POS_E2E_BOOTSTRAP_JSON=/workspace/myhoreca-pos/.e2e/bootstrap.json
POS_E2E_UI_BASE=http://localhost:5173
POS_E2E_API_BASE=http://pos-edge:8080/api/v1
POS_E2E_CLOUD_BASE=http://cloud-api:8090/api/v1
VITE_POS_API_BASE=http://pos-edge:8080/api/v1
VITE_CLOUD_API_BASE=http://cloud-api:8090/api/v1
```

Внутри `devbox` Playwright browser и Vite dev server живут в одном контейнере, поэтому UI base остается `http://localhost:5173`. Backend API при этом доступен не через `localhost:8080`, а через Docker service DNS `http://pos-edge:8080/api/v1`. Host gateway для этого сценария не нужен.

## Запуск

Из корня репозитория:

```bash
docker compose -f docker-compose.local.yml up --build -d
```

Если локальный `5432` уже занят другой PostgreSQL-инстанцией, можно поменять только host binding, не меняя внутренний DSN между контейнерами:

```bash
CLOUD_POSTGRES_HOST_PORT=55432 \
docker compose -f docker-compose.local.yml up --build -d
```

Проверка health endpoints:

```bash
curl -fsS http://localhost:8090/health
curl -fsS http://localhost:8095/health
curl -fsS http://localhost:8080/health
```

## UI devbox для build/unit/e2e

Собери devbox один раз; Chromium и системные зависимости Playwright устанавливаются в image build, а не при каждом запуске Codex:

```bash
docker compose -f docker-compose.local.yml --profile devbox build devbox
```

Запусти backend stack и devbox:

```bash
docker compose -f docker-compose.local.yml up --build -d cloud-postgres license-api cloud-api pos-edge
docker compose -f docker-compose.local.yml --profile devbox up -d devbox
```

Установи UI dependencies в Docker volumes, не в Windows-mounted `node_modules`:

```bash
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd pos-ui && npm install'
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd cloud-ui && npm install'
```

Проверки build/unit:

```bash
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd pos-ui && npm run build'
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd pos-ui && npm run test'
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd cloud-ui && npm run build'
```

Перед POS UI E2E нужен bootstrap файл. Он содержит demo identifiers/PIN для локального стенда и не коммитится:

```bash
docker compose -f docker-compose.local.yml exec devbox bash -lc '
  mkdir -p .e2e &&
  python3 scripts/run-stack-smoke.py \
    --suite cloud_to_edge_masterdata \
    --cloud-base http://cloud-api:8090 \
    --pos-base http://pos-edge:8080 \
    --license-base http://license-api:8095 \
    --output .e2e/bootstrap.json
'
```

Файл `.e2e/bootstrap.example.json` показывает ожидаемую форму. Реальный `.e2e/bootstrap.json` игнорируется git.

Для browser-based E2E запусти Vite в devbox, затем тесты во втором shell:

```bash
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd pos-ui && npm run dev'
```

```bash
docker compose -f docker-compose.local.yml exec devbox bash -lc '
  cd pos-ui &&
  npx playwright test e2e/waiter-mobile-flow.spec.ts e2e/kitchen-flow.spec.ts
'
```

Для API-only E2E specs тот же `POS_E2E_BOOTSTRAP_JSON` путь уже задан service environment. Если нужно передать JSON напрямую, `POS_E2E_BOOTSTRAP_JSON` также принимает содержимое JSON-строки.

Если UI открывается из host browser по `http://localhost:5173`, не используй devbox service DNS в browser bundle: запусти UI локально или переопредели `VITE_POS_API_BASE=http://localhost:8080/api/v1`. Service DNS `pos-edge:8080` предназначен для Playwright, запущенного внутри devbox.

Логи:

```bash
docker compose -f docker-compose.local.yml logs -f cloud-api
docker compose -f docker-compose.local.yml logs -f pos-edge
docker compose -f docker-compose.local.yml logs -f license-api
```

Остановка без удаления данных:

```bash
docker compose -f docker-compose.local.yml down
```

Полный reset локальных Docker БД и файлов является destructive-by-design операцией:

```bash
docker compose -f docker-compose.local.yml down -v
```

## Заполнение Cloud и проверка POS на Linux/Fedora

Реализовано сейчас: канонический локальный путь использует Python 3 scripts без внешних Python dependencies. Скрипты создают demo справочники через Cloud HTTP API, выполняют POS Edge provisioning через License/Cloud API, затем проверяют POS read model через POS HTTP API. HTTP calls в Python ядре строятся из OpenAPI contract `docs/api/mhpos-local-smoke.openapi.json` по `operationId`; прямые записи в PostgreSQL/SQLite не используются.

Полный semi-automatic smoke для поднятого Docker stack:

```bash
python3 scripts/run-local-masterdata-smoke.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.local-masterdata-summary.json
```

Полный stack smoke, который проверяет Cloud API, POS Edge API и License Server одной Python-утилитой:

```bash
python3 scripts/run-stack-smoke.py \
  --suite all \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.local-masterdata-summary.json \
  --json-output scripts/.stack-smoke-result.json
```

Реализованные suites:

- `health` - проверяет root health endpoint Cloud, POS Edge и License Server;
- `license_pairing` - напрямую регистрирует одноразовый pairing code в License Server, resolve-ит его и проверяет, что повторный resolve отклоняется;
- `cloud_to_edge_masterdata` - создает Cloud-owned demo master data, выполняет POS Edge provisioning, проверяет POS read model и post-pairing Cloud -> Edge sync.

Правило расширения: когда в Cloud API, POS Edge API или License Server появляется новая функциональность, которая должна входить в локальную приемку, добавить OpenAPI operation в `docs/api/mhpos-local-smoke.openapi.json`, отдельную suite или шаг suite в `scripts/lib/mhpos_stack.py`, unit test в `scripts/tests` и обновить этот раздел документации.

То же через thin Bash wrapper:

```bash
./scripts/run-local-masterdata-smoke.sh \
  --output scripts/.local-masterdata-summary.json
```

Сценарий создает ресторан, роли, сотрудников с PIN `1111`/`2222`, зал/стол, catalog/menu items, service item, modifier group/option/binding, публикует typed Cloud -> POS Edge package, выполняет pairing/provisioning и проверяет, что POS видит Cloud-created данные. После pairing скрипт добавляет дополнительную Cloud menu позицию, повторно публикует master data и ждет, пока POS Edge sync sender получит ее через authenticated `sync/exchange`.

Раздельные шаги для отладки:

```bash
python3 scripts/seed-cloud-masterdata.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --output scripts/.local-masterdata-summary.json

python3 scripts/provision-pos-edge.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --summary scripts/.local-masterdata-summary.json

python3 scripts/verify-sync.py \
  --pos-base http://localhost:8080 \
  --summary scripts/.local-masterdata-summary.json
```

Windows-compatible wrappers остаются тонкими оболочками над тем же Python ядром:

```powershell
.\scripts\run-local-masterdata-smoke.ps1 --output scripts/.local-masterdata-summary.json
```

`scripts/.local-masterdata-summary.json` содержит локальные demo PIN для последующих автоматических шагов и добавлен в `.gitignore`; не коммить этот файл. `scripts/.stack-smoke-result.json` содержит безопасный JSON-отчет `run-stack-smoke.py` и тоже игнорируется git.

Повторный запуск `run-stack-smoke.py --suite all` на уже provisioned Edge использует существующий `--output` summary, если `restaurant_id` и `node_device_id` совпадают с текущей POS Edge привязкой. В этом режиме suite не пересоздает pairing, а проверяет текущий POS read model и публикует новый post-pairing Cloud -> Edge item в тот же ресторан. Если summary отсутствует или относится к другому ресторану, suite завершается fail-fast: нужно либо передать корректный summary, либо пересоздать локальные Docker volumes.

Python HTTP layer игнорирует системные proxy-переменные для `localhost`/loopback адресов. Это важно для Windows/Linux окружений, где `HTTP_PROXY`/`HTTPS_PROXY` могут уводить запросы к Docker published ports в корпоративный proxy. Если post-pairing sync не доходит до POS Edge, отчет дополнительно выводит `sync_status` и последний `last_error` из Edge outbox, например `SYNC_FORBIDDEN`.

Важно для ручного наглядного теста: demo seed dataset должен расширяться вместе с развитием проекта. Когда появляются новые Cloud-owned справочники, publication streams или POS read flows, их нужно добавлять в OpenAPI contract, Python seed/sync сценарии и эту документацию в том же PR.

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

Оба legacy Zero-to-Cashier PowerShell скрипта создают Cloud master data, привязывают POS Edge и проверяют PIN login. По умолчанию cashier PIN: `1111`. Для Fedora/Linux и нового semi-automatic master-data smoke использовать Python scripts выше.

## Ручная проверка через POS UI

После запуска Docker stack запусти UI локально:

```bash
cd pos-ui
npm install
npm run dev
```

Открой `http://localhost:5173`. POS UI ходит в POS Edge на `http://localhost:8080/api/v1`.

Если перед этим был выполнен новый Python master-data smoke или legacy Zero-to-Cashier скрипт, Edge уже paired, а войти можно PIN `1111`. Для manager сценариев в Python smoke используется PIN `2222`.

## Проверка PostgreSQL и sync

Cloud operational events:

```bash
docker compose -f docker-compose.local.yml exec cloud-postgres \
  psql -U postgres -d mh_pos_cloud \
  -c "select event_type, count(*) from cloud_operational_events group by event_type order by event_type;"
```

Последние Cloud receipts:

```bash
docker compose -f docker-compose.local.yml exec cloud-postgres \
  psql -U postgres -d mh_pos_cloud \
  -c "select idempotency_key, event_type, cloud_received_at from cloud_edge_event_receipts order by cloud_received_at desc limit 10;"
```

POS sync sender доставляет Edge -> Cloud operational rows автоматически, когда `POS_SYNC_SENDER_ENABLED=true`. Недоступность Cloud не блокирует POS runtime writes.

реализовано сейчас:

- после pairing через license code или Cloud assignment `pos-edge` прекращает повторный device registration/snapshot provisioning poll;
- локальные Edge outbox rows отправляются через authenticated `sync/exchange` на ближайшем worker tick;
- пустой `sync/exchange` для Cloud -> Edge pull ограничен отдельным interval, чтобы локальный Docker stack не создавал шумный Cloud access log при отсутствии локальных событий.
- Cloud UI после successful master-data CRUD автоматически вызывает publication API от имени `cloud-ui`; ручная публикация в разделе Publication state остается доступна для явного operator checkpoint.

вне текущего объема: production serving `pos-ui` из Docker Compose и production auth perimeter для Cloud/License API.
