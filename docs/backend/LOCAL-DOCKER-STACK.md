# Локальный Docker stack и UI запуск

Документ описывает локальный запуск `cloud-postgres`, `cloud-clickhouse`, `cloud-api`, `license-api` и `pos-edge` через общий Docker Compose. Текущий `docker-compose.local.yml` не содержит UI/devbox services; frontend dev/build/test запускаются из соответствующих локальных каталогов.

## Что запускается

Реализовано сейчас:

- `cloud-postgres` - PostgreSQL 16 для Cloud Backend;
- `cloud-clickhouse` - ClickHouse для bounded Cloud OLAP slices;
- `cloud-api` - Cloud Sync Receiver и Cloud master-data authority;
- `license-api` - локальный License Server для Option B pairing code flow и entitlement snapshots;
- `pos-edge` - POS Edge backend с SQLite, IANA timezone data (`tzdata`) для `business_date_local` и включенным sync sender.

Не реализовано сейчас в `docker-compose.local.yml`:

- `devbox` service/profile;
- `pos-ui-g` или `cloud-ui-g` services.

Именованные volumes:

- `cloud_postgres_data` - данные PostgreSQL;
- `cloud_clickhouse_data` - данные ClickHouse;
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
CLOUD_CLICKHOUSE_HOST_PORT=8123
CLOUD_CLICKHOUSE_NATIVE_HOST_PORT=9000
CLOUD_API_HOST_PORT=8090
POS_EDGE_HOST_PORT=8080
LICENSE_API_HOST_PORT=8095
CLOUD_POSTGRES_DSN=postgres://postgres:postgres@cloud-postgres:5432/mh_pos_cloud?sslmode=disable
CLOUD_PUBLIC_URL=http://cloud-api:8090
LICENSE_SERVER_URL=http://license-api:8095
POS_CLOUD_SYNC_URL=http://cloud-api:8090/api/v1/sync/edge-events
POS_SQLITE_PATH=/app/data/pos-edge.db
```

Файловый конфиг имеет приоритет над env. Общий контракт описан в `docs/backend/RUNTIME-CONFIG.md`.

Локальный E2E environment:

```text
POS_E2E_BOOTSTRAP_JSON=.e2e/bootstrap.json
POS_E2E_UI_BASE=http://localhost:5173
POS_E2E_API_BASE=http://localhost:8080/api/v1
POS_E2E_CLOUD_BASE=http://localhost:8090/api/v1
VITE_POS_API_BASE=http://localhost:8080/api/v1
VITE_CLOUD_API_BASE=http://localhost:8090/api/v1
```

Если Playwright запускается с host machine, backend API доступен через published ports `localhost:8080` и `localhost:8090`. Docker service DNS вроде `pos-edge:8080` и `cloud-api:8090` доступен только внутри compose network.

## Запуск

Из корня репозитория:

```bash
docker compose -f docker-compose.local.yml up --build -d
```

Если локальные `5432`, `8123`, `9000`, `8090`, `8080` или `8095` уже заняты, можно поменять только host binding, не меняя внутренние DSN/URLs между контейнерами:

```bash
CLOUD_POSTGRES_HOST_PORT=55432 \
CLOUD_CLICKHOUSE_HOST_PORT=18123 \
CLOUD_CLICKHOUSE_NATIVE_HOST_PORT=19000 \
CLOUD_API_HOST_PORT=18090 \
POS_EDGE_HOST_PORT=18080 \
LICENSE_API_HOST_PORT=18095 \
docker compose -f docker-compose.local.yml up --build -d
```

При таком override health/seed с host machine нужно вызывать по новым host ports, например `http://localhost:18090`, `http://localhost:18080`, `http://localhost:18095`. Внутри Docker network остаются service DNS `cloud-api:8090`, `pos-edge:8080`, `license-api:8095`, `cloud-postgres:5432` и `cloud-clickhouse:8123`.

Если `docker compose ... up --build` останавливается на сообщении про отсутствующий buildx plugin, это blocker локального Docker CLI/Compose окружения, а не runtime code. Исправление: установить/включить Docker buildx plugin для используемого Docker CLI или собрать/запустить stack в окружении Docker Desktop/Compose, где `docker buildx version` проходит. `docker compose -f docker-compose.local.yml up -d` без `--build` является fallback только если нужные images уже были успешно собраны ранее.

Проверка health endpoints:

```bash
curl -fsS http://localhost:8090/health
curl -fsS http://localhost:8095/health
curl -fsS http://localhost:8080/health
```

## UI build/unit/e2e

Реализовано сейчас: UI проверки запускаются локально из каталогов приложений, а не через `docker-compose.local.yml`.

POS UI:

```bash
cd pos-ui-g
npm install
npm run build
```

Активный Cloud UI:

```bash
cd cloud-ui-g
npm install
npm run lint
npm run test
npm run build
```

npm install
npm run test
npm run build
```

Перед POS UI E2E нужен seed-файл. Он содержит локальные demo identifiers, pairing code и PIN для проверки ролей и не коммитится:

```bash
mkdir -p .e2e
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output .e2e/bootstrap.json
```

Файл `.e2e/bootstrap.example.json` показывает ожидаемую форму. Реальный `.e2e/bootstrap.json` игнорируется git.

Для browser-based E2E запусти Vite локально, затем тесты во втором shell, если соответствующие specs перенесены в `pos-ui-g`:

```bash
cd pos-ui-g
npm run dev
```

```bash
cd pos-ui-g
POS_E2E_BOOTSTRAP_JSON=../.e2e/bootstrap.json \
POS_E2E_UI_BASE=http://localhost:5173 \
POS_E2E_API_BASE=http://localhost:8080/api/v1 \
POS_E2E_CLOUD_BASE=http://localhost:8090/api/v1 \
npx playwright test e2e/waiter-mobile-flow.spec.ts e2e/kitchen-flow.spec.ts
```

Для API-only E2E specs `POS_E2E_BOOTSTRAP_JSON` можно передать как путь к JSON или содержимое JSON-строки.

Если UI открывается из host browser по `http://localhost:5173`, используй `VITE_POS_API_BASE=http://localhost:8080/api/v1`; Docker service DNS `pos-edge:8080` доступен только внутри compose network.

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

Реализовано сейчас: канонический локальный путь использует один самодостаточный Python 3 скрипт без внешних Python dependencies. `scripts/seed-dev-system.py` является единственным user-facing demo/seed entrypoint для Fedora/Linux/Windows-compatible локального контура: он создает полный набор текущих Cloud-owned справочников через Cloud HTTP API, публикует master-data packages, генерирует license pairing code, выполняет POS Edge `pair-via-license` и проверяет POS read model через POS HTTP API. Прямые записи в PostgreSQL/SQLite/ClickHouse не используются.

Полный seed для поднятого Docker stack:

```bash
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.seed-dev-system-summary.json
```

Скрипт создает ресторан, роли cashier/senior cashier/waiter/manager/kitchen/support, сотрудников с PIN `1111`/`2222`/`3333`/`4444`/`5555`/`9999`, залы и столы, catalog folders/folder parameters/tags/items, menu categories/items, service item, modifier groups/options/bindings, pricing policies, active recipe versions через manager draft -> submit -> approve flow, stop-list examples и publication. После создания всех сущностей он генерирует pairing code через Cloud/License flow, привязывает POS Edge и проверяет, что POS видит Cloud-created halls/menu.

Минимальный сквозной smoke для поднятого stack:

```bash
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.seed-dev-system-summary.json \
  --run-minimal-flow
```

Реализовано сейчас: флаг `--run-minimal-flow` после seed/pairing выполняет HTTP-only сценарий `Cloud recipes/stop-list publication -> Edge sync -> waiter order/precheck -> KDS served -> cashier final check -> ItemServed/CheckClosed -> Cloud inventory ledger -> ClickHouse/OLAP bounded reads`. Сценарий проверяет stop-list rejection для demo sold-out item, создает заказ официантом, проводит KDS ticket через `accept/start/ready/serve`, выпускает precheck, закрывает его оплатой кассира, ожидает `ItemServed` и `CheckClosed` в Cloud safe event log, проверяет `stock_ledger` для `ItemServed`, отсутствие duplicate `CheckClosed` delta по тому же `order_line_id`, экспорт `ItemServed`/`CheckClosed` в ClickHouse `raw_business_events`, bounded `olap_stock_moves` для `ItemServed`, `stock-move-summary` и `sales-kitchen-summary` без raw payload. Финансовая мутация оплаты выполняется single-shot без automatic retry.

Полный kitchen/process smoke для поднятого stack:

```bash
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.seed-dev-system-summary.json \
  --run-kitchen-process-smoke
```

Реализовано сейчас: флаг `--run-kitchen-process-smoke` после seed/pairing проверяет Cloud publication для `catalog`/`menu`/`recipes`/`inventory_reference` включая default warehouse `warehouse-main`, Edge sync полного каталога и техкарт, waiter order с блюдом, KDS order tile, `accept/start/ready/serve`, `recall/start/ready/serve`, прием `KitchenTicketStatusChanged`/`ItemServed` в Cloud, наличие kitchen event trail в ClickHouse `raw_business_events`, Cloud `stock_ledger`, bounded Cloud `stock-balances` read без raw payload и bounded ClickHouse `olap_stock_moves` read для `ItemServed`/`StockReceiptCaptured`/`InventoryCountCaptured`/`StockWriteOffCaptured`/`ProductionCompleted`, создание catalog/recipe suggestions на Edge, Cloud manager approve и возврат `proposal_feedback` на Edge.

Полный профильный запуск обеих smoke-веток требует чистого POS Edge pairing state, поэтому перед ним выполняется reset volumes:

```bash
docker compose -f docker-compose.local.yml down -v
docker compose -f docker-compose.local.yml up --build -d
python3 scripts/seed-dev-system.py \
  --cloud-base http://localhost:8090 \
  --pos-base http://localhost:8080 \
  --license-base http://localhost:8095 \
  --output scripts/.seed-dev-system-summary.json \
  --run-minimal-flow \
  --run-kitchen-process-smoke
```

Если включены оба флага, итоговый JSON содержит независимые секции `minimal_flow` и `kitchen_process_smoke`. Минимальная ветка подтверждает полный cashier/waiter/KDS/check/inventory/OLAP хвост и может использовать резервный PIN с KDS authority для подачи блюда, а полный kitchen/process smoke должен использовать опубликованную kitchen role/PIN `5555`, чтобы проверить фактические backend RBAC и Cloud маршруты.

Seed-вход содержит только пользовательские данные: названия, имена, PIN, цены, количества, места и наборы прав. ID, `node_device_id`, generated SKU и остальные технические значения берутся из backend responses или генерируются системно. `scripts/.seed-dev-system-summary.json` содержит локальные demo credentials и добавлен в `.gitignore`; не коммить этот файл.

Правило расширения: новый Cloud-owned справочник, publication stream или POS read flow добавляется в canonical `scripts/seed-dev-system.py` тем же PR, что и runtime/doc изменение. Обязательный checklist: seed dataset, publication stream/package, POS read flow или smoke assertion, script guard `CLOUD_OWNED_SEED_SURFACES`, профильные документы. Отдельные пользовательские seed/smoke entrypoints не добавляются без явного архитектурного решения.

Повторный запуск рассчитан на чистые backend volumes. Если POS Edge уже находится в `paired`, скрипт завершится fail-fast: для нового полного seed нужно пересоздать локальные Docker volumes через `docker compose -f docker-compose.local.yml down -v` и поднять stack заново.

Python HTTP layer игнорирует системные proxy-переменные для `localhost`/loopback адресов. Это важно для Windows/Linux окружений, где `HTTP_PROXY`/`HTTPS_PROXY` могут уводить запросы к Docker published ports в корпоративный proxy.

## Ручная проверка через POS UI

После запуска Docker stack запусти UI локально:

```bash
cd pos-ui-g
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
- Реализовано сейчас: canonical seed/smoke не вызывает manual publish API. После pairing/assignment Cloud автоматически собирает current batch, а последующие Cloud CRUD changes обновляют latest packages для назначенных Edge и приходят на Edge через scheduled exchange.

вне текущего объема: production serving `pos-ui-g`/`cloud-ui-g` из Docker Compose, devbox service в текущем compose и production auth perimeter для Cloud/License API.
