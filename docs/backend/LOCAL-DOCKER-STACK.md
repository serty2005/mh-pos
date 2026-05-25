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

Перед POS UI E2E нужен seed-файл. Он содержит локальные demo identifiers, pairing code и PIN для проверки ролей и не коммитится:

```bash
docker compose -f docker-compose.local.yml exec devbox bash -lc '
  mkdir -p .e2e &&
  python3 scripts/seed-dev-system.py \
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

## Заполнение Cloud и POS начальными данными

Реализовано сейчас: канонический локальный путь использует один самодостаточный Python 3 скрипт без внешних Python dependencies. `scripts/seed-dev-system.py` создает полный набор текущих Cloud-owned справочников через Cloud HTTP API, публикует master-data packages, генерирует license pairing code, выполняет POS Edge `pair-via-license` и проверяет POS read model через POS HTTP API. Прямые записи в PostgreSQL/SQLite не используются.

Полный seed для поднятого Docker stack:

```bash
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.seed-dev-system-summary.json
```

Скрипт создает ресторан, роли cashier/senior cashier/waiter/manager/kitchen/support, сотрудников с PIN `1111`/`2222`/`3333`/`4444`/`5555`/`9999`, залы и столы, catalog folders/folder parameters/tags/items, menu categories/items, service item, modifier groups/options/bindings, pricing policies, recipe items, stop-list examples и publication. После создания всех сущностей он генерирует pairing code через Cloud/License flow, привязывает POS Edge и проверяет, что POS видит Cloud-created halls/menu.

Seed-вход содержит только пользовательские данные: названия, имена, PIN, цены, количества, места и наборы прав. ID, `node_device_id`, generated SKU и остальные технические значения берутся из backend responses или генерируются системно. `scripts/.seed-dev-system-summary.json` содержит локальные demo credentials и добавлен в `.gitignore`; не коммить этот файл.

Повторный запуск рассчитан на чистые backend volumes. Если POS Edge уже находится в `paired`, скрипт завершится fail-fast: для нового полного seed нужно пересоздать локальные Docker volumes через `docker compose -f docker-compose.local.yml down -v` и поднять stack заново.

Python HTTP layer игнорирует системные proxy-переменные для `localhost`/loopback адресов. Это важно для Windows/Linux окружений, где `HTTP_PROXY`/`HTTPS_PROXY` могут уводить запросы к Docker published ports в корпоративный proxy.

## Ручная проверка через POS UI

После запуска Docker stack запусти UI локально:

```bash
cd pos-ui
npm install
npm run dev
```

Открой `http://localhost:5173`. POS UI ходит в POS Edge на `http://localhost:8080/api/v1`.

Если перед этим был выполнен `scripts/seed-dev-system.py`, Edge уже paired, а войти можно PIN `1111`. Для manager сценариев используется PIN `2222`; остальные PIN перечислены в `scripts/.seed-dev-system-summary.json`.

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
