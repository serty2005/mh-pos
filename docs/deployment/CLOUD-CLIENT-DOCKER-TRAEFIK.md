# Cloud client Docker/Traefik deployment

Статус: `реализовано сейчас` как минимальный alpha/pre-Kubernetes production path для 1-5 независимых клиентских Cloud-стеков на одной Linux VM.

## Назначение

Этот контур нужен для выставочной альфы без Kubernetes. Один клиентский стек содержит Cloud Backend, PostgreSQL, ClickHouse и Cloud UI. Общим для VM остается только Traefik reverse proxy и внешняя Docker network `traefik_proxy`.

License Server остается внешним центральным authority. Клиентский Cloud Backend подключается к нему через `LICENSE_SERVER_URL`.

## Реализовано Сейчас

- `deploy/traefik/docker-compose.traefik.yml` поднимает Traefik v3 с Docker provider, `exposedbydefault=false`, entrypoints `web`/`websecure`, HTTP -> HTTPS redirect и Let's Encrypt HTTP challenge resolver `le`.
- Dashboard Traefik публикуется только через отдельный host из `TRAEFIK_DASHBOARD_DOMAIN`, HTTPS и basic auth.
- `deploy/cloud-client/docker-compose.cloud-client.yml` описывает один клиентский Cloud-стек без публикации host ports для PostgreSQL, ClickHouse и Cloud Backend.
- Cloud UI и Cloud API работают на одном домене: UI на `/`, API на `/api/v1`; Traefik роутит API по `Host(...) && PathPrefix(/api)`, а `/health` отдельно направляет в Cloud Backend для smoke.
- Client/server isolation держится на паре `CLIENT_SLUG` + `SERVER_SLUG`, Docker Compose project name, volumes, env/config и domains.
- Cloud API image может работать от env без ручной правки JSON внутри контейнера: compose задает `CLOUD_CONFIG_PATH=""`, поэтому встроенный `/app/config/cloud-api.docker.json` не перекрывает production env.
- Cloud UI production image собирается из `cloud-ui-g/Dockerfile` с `VITE_CLOUD_API_BASE=/api/v1`.
- `deploy/cloud-client/docker-compose.cloud-build.yml` собирает и публикует `mhpos-cloud-api` и `mhpos-cloud-ui` по immutable version tag без отдельного shell-скрипта.

## Запланировано Далее

- GitHub Actions/CD для сборки immutable images и выкладки на Docker Hub без ручного запуска скрипта.
- Data-preserving production migrations после первого реального внедрения.
- Регулярные backup jobs, restore rehearsal и мониторинг health/sync/OLAP freshness.
- Kubernetes/Helm/GitOps контур: namespace/release-per-client, Helm values вместо env-файлов и Ingress/IngressRoute вместо Docker labels.

## Вне Текущего Объема

- Kubernetes, Helm charts и GitOps manifests.
- Встроенный secret vault.
- Production auth/RBAC perimeter Cloud API beyond текущего alpha-периметра.
- Развертывание центрального License Server внутри клиентского стека.
- Автоматический DB downgrade. Downgrade не поддерживается.

## Prerequisites VM

- Linux VM с одним белым IP.
- Docker Engine и Docker Compose plugin.
- DNS A records:
  - один домен/поддомен на каждого клиента, например `alpha-client.example.com`;
  - отдельный домен для Traefik dashboard, например `traefik.example.com`.
- Firewall открыт только для `80/tcp`, `443/tcp` и SSH с доверенных IP.
- Доступ VM к Docker Hub для pull private/public images.
- Внешний License Server URL, доступный из контейнеров Cloud Backend.
- Production env/secrets хранятся вне Git, например в `clients/<client>.tenant.env`, `clients/<client>-<server>.env` и `deploy/traefik/traefik.env`.

## Сборка И Публикация Images

Команды не запускаются агентом автоматически и не пушат образы без оператора.

Один раз создайте build env рядом с build compose:

```bash
cp deploy/cloud-client/build.env.example deploy/cloud-client/build.env
```

В `deploy/cloud-client/build.env` задаются:

- `MHPOS_DOCKER_NAMESPACE`;
- `MHPOS_VERSION`;
- `VITE_CLOUD_API_BASE=/api/v1`.

Сборка свежих images:

```bash
docker compose --env-file deploy/cloud-client/build.env -f deploy/cloud-client/docker-compose.cloud-build.yml build
```

Публикация в Docker Hub:

```bash
docker compose --env-file deploy/cloud-client/build.env -f deploy/cloud-client/docker-compose.cloud-build.yml push
```

Опциональный mutable convenience tag `alpha-latest` можно создать вручную после push immutable tag:

```bash
docker tag myhoreca/mhpos-cloud-api:0.1.15-alpha.1 myhoreca/mhpos-cloud-api:alpha-latest
docker tag myhoreca/mhpos-cloud-ui:0.1.15-alpha.1 myhoreca/mhpos-cloud-ui:alpha-latest
docker push myhoreca/mhpos-cloud-api:alpha-latest
docker push myhoreca/mhpos-cloud-ui:alpha-latest
```

Deployment должен ссылаться на explicit immutable `MHPOS_VERSION`, а не на `alpha-latest`.

## Запуск Traefik

```bash
cp deploy/traefik/traefik.env.example deploy/traefik/traefik.env
```

В `deploy/traefik/traefik.env` задайте:

- `TRAEFIK_ACME_EMAIL`;
- `TRAEFIK_DASHBOARD_DOMAIN`;
- `TRAEFIK_DASHBOARD_BASIC_AUTH`.

Basic auth hash:

```bash
htpasswd -nbB admin 'strong-password' | sed -e 's/\$/$$/g'
```

Создайте общую сеть один раз:

```bash
docker network create traefik_proxy
```

Запустите Traefik:

```bash
docker compose -f deploy/traefik/docker-compose.traefik.yml --env-file deploy/traefik/traefik.env up -d
```

Проверьте:

```bash
docker compose -f deploy/traefik/docker-compose.traefik.yml --env-file deploy/traefik/traefik.env ps
```

`acme.json` хранится в Docker volume `traefik_letsencrypt` и не коммитится.

## Добавление Клиента

1. Создайте env из example:

```bash
mkdir -p clients
cp deploy/cloud-client/client.env.example clients/<client>-<server>.env
```

2. Заполните `clients/<client>-<server>.env` реальными значениями:

- `CLIENT_SLUG` для контрагента/tenant;
- `SERVER_SLUG` для конкретного Cloud-сервера/стека этого контрагента;
- общий `CLIENT_BASE_DOMAIN`;
- explicit `MHPOS_VERSION`;
- PostgreSQL и ClickHouse passwords;
- `LICENSE_SERVER_URL`.

Минимальный client env намеренно короткий. По умолчанию из `CLIENT_SLUG=demoalpha`, `SERVER_SLUG=cloud1` и `CLIENT_BASE_DOMAIN=example.com` compose собирает:

- `CLIENT_DOMAIN=cloud1.demoalpha.example.com`;
- `CLOUD_PUBLIC_URL=https://cloud1.demoalpha.example.com`;
- PostgreSQL DB/user: `mhpos_demoalpha_cloud1`;
- ClickHouse DB/user: `mhpos_demoalpha_cloud1`;
- `LICENSE_TENANT_ID=demoalpha`;
- `LICENSE_SERVER_ID=demoalpha-cloud1`.

`CLIENT_SLUG` и `SERVER_SLUG` держите в нижнем регистре из `a-z` и `0-9`: они входят в domain labels, Docker volumes, network, Traefik routers, PostgreSQL и ClickHouse имена. Если нужен нестандартный домен или уже выданные License IDs, задайте optional overrides в client env: `CLIENT_DOMAIN`, `CLOUD_PUBLIC_URL`, `LICENSE_TENANT_ID`, `LICENSE_SERVER_ID`, `CLOUD_POSTGRES_DB`, `CLOUD_POSTGRES_USER`, `CLOUD_POSTGRES_DSN`, `CLOUD_CLICKHOUSE_DATABASE`, `CLOUD_CLICKHOUSE_USER`.

Если на одной production VM живет один контрагент и несколько его серверов, можно вынести tenant-level параметры в отдельный env:

```bash
cp deploy/cloud-client/tenant.env.example clients/<client>.tenant.env
cp deploy/cloud-client/client.env.example clients/<client>-<server>.env
```

Тогда server env может содержать только `SERVER_SLUG`, `MHPOS_VERSION` и secrets конкретного стека, а общий tenant env содержит `CLIENT_SLUG`, `CLIENT_BASE_DOMAIN`, `LICENSE_SERVER_URL` и при необходимости `LICENSE_TENANT_ID`. Compose поддерживает несколько `--env-file`; более поздний файл переопределяет более ранний:

```bash
docker compose \
  --env-file clients/<client>.tenant.env \
  --env-file clients/<client>-<server>.env \
  -p mhpos-<client>-<server> \
  -f deploy/cloud-client/docker-compose.cloud-client.yml config
```

3. Проверьте DNS:

```bash
dig +short <CLIENT_DOMAIN>
```

4. Загрузите images:

```bash
docker compose --env-file clients/<client>-<server>.env -p mhpos-<client>-<server> -f deploy/cloud-client/docker-compose.cloud-client.yml pull
```

5. Запустите стек:

```bash
docker compose --env-file clients/<client>-<server>.env -p mhpos-<client>-<server> -f deploy/cloud-client/docker-compose.cloud-client.yml up -d
```

6. Проверьте health и UI:

```bash
curl -fsS https://<CLIENT_DOMAIN>/health
curl -fsS https://<CLIENT_DOMAIN>/api/v1/sync/readiness/stop-list
```

Cloud UI должен открываться на `https://<CLIENT_DOMAIN>/`.

## Обновление Клиента

1. Соберите и push новые images в Docker Hub по новому immutable tag.
2. Измените `MHPOS_VERSION` в `clients/<client>-<server>.env`.
3. Выполните:

```bash
docker compose --env-file clients/<client>-<server>.env -p mhpos-<client>-<server> -f deploy/cloud-client/docker-compose.cloud-client.yml pull
docker compose --env-file clients/<client>-<server>.env -p mhpos-<client>-<server> -f deploy/cloud-client/docker-compose.cloud-client.yml up -d
```

4. Smoke после обновления:

- `GET https://<CLIENT_DOMAIN>/health`;
- Cloud UI на `/`;
- Cloud API на `/api/v1`;
- Edge pairing через внешний License Server;
- Edge sync exchange и доставка master data;
- проверка свежести OLAP/ClickHouse, если релиз затрагивал analytics.

## Rollback

1. Верните прежний `MHPOS_VERSION` в `clients/<client>-<server>.env`.
2. Выполните:

```bash
docker compose --env-file clients/<client>-<server>.env -p mhpos-<client>-<server> -f deploy/cloud-client/docker-compose.cloud-client.yml pull
docker compose --env-file clients/<client>-<server>.env -p mhpos-<client>-<server> -f deploy/cloud-client/docker-compose.cloud-client.yml up -d
```

Важно: DB downgrade не поддерживается. Если релиз менял schema/runtime version, rollback требует совместимых migrations или backup/restore. Перед schema upgrade нужен production backup PostgreSQL и ClickHouse.

## Backup

PostgreSQL:

- регулярный `pg_dump` для logical backup;
- volume snapshot только при согласованной стратегии quiesce/snapshot;
- хранение backup вне VM;
- периодический restore rehearsal на отдельном окружении.

ClickHouse:

- ClickHouse backup/snapshot strategy по выбранному production способу;
- проверка восстановления таблиц `raw_business_events` и `olap_stock_moves`;
- retention policy для локальных и внешних копий.

Общее:

- backup retention должен быть согласован до первого клиента;
- runtime pre-upgrade JSONL backup не является заменой полноценного production backup PostgreSQL/ClickHouse;
- secrets/env-файлы клиентов не переиспользуются между клиентами и не хранятся в Git.

## Несколько Клиентов На Одной VM

- Каждый контрагент получает свой `CLIENT_SLUG`.
- Каждый сервер/стек контрагента получает свой `SERVER_SLUG`.
- Один `LICENSE_TENANT_ID` может использоваться несколькими серверами одного контрагента на той же VM: это нормальная alpha/pre-Kubernetes схема, если каждый сервер изолирован отдельным stack boundary.
- Каждый сервер/стек запускается с отдельным Compose project name: `-p mhpos-<client>-<server>`.
- Каждый сервер/стек получает домен из `SERVER_SLUG.CLIENT_SLUG.CLIENT_BASE_DOMAIN` или явный override `CLIENT_DOMAIN`.
- Общая только Docker network `traefik_proxy`.
- PostgreSQL, ClickHouse и Cloud API volumes отдельные: имена включают `CLIENT_SLUG` и `SERVER_SLUG`.
- PostgreSQL и ClickHouse host ports не публикуются.
- Cloud Backend host port не публикуется.
- Traefik labels есть только на `cloud-api` и `cloud-ui`.
- `LICENSE_SERVER_ID` должен быть уникален для каждого server stack, даже если `LICENSE_TENANT_ID` общий.
- Secrets, DB passwords, server IDs и server env-файлы не переиспользуются между стеками.

## Перенос В Kubernetes Позже

- Images уже имеют immutable tags и подходят для image registry based rollout.
- Значения из `clients/<client>.tenant.env` и `clients/<client>-<server>.env` переносятся в Helm values и Kubernetes Secrets.
- Volume data переносится через PostgreSQL dump/restore и ClickHouse backup/restore.
- Traefik Docker labels превращаются в Ingress или IngressRoute.
- Client stack boundary сохраняется как namespace/release-per-client.
- Внешний License Server остается отдельным authority и передается как `LICENSE_SERVER_URL`.

## Минимальные Проверки Конфигов

```bash
docker compose --env-file deploy/cloud-client/client.env.example -p mhpos-demoalpha-cloud1 -f deploy/cloud-client/docker-compose.cloud-client.yml config
docker compose --env-file deploy/cloud-client/build.env.example -f deploy/cloud-client/docker-compose.cloud-build.yml config
docker compose --env-file deploy/traefik/traefik.env.example -f deploy/traefik/docker-compose.traefik.yml config
cd cloud-ui-g && npm run build
```

Если менялся Go runtime code, дополнительно:

```bash
cd cloud-backend && go test ./...
```
