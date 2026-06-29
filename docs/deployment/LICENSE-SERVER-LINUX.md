# License Server на Ubuntu 24.04

Статус: runbook для альфа-развертывания `license-server` как native binary без Docker.

## Термины

- Контрагент — клиент или юридическое лицо, которое платит за обслуживание; runtime поле `tenant_id`.
- Сервер — конкретный Cloud или POS Edge runtime под контрагентом; runtime поле `server_id`.
- Один контрагент может иметь несколько серверов. Один сервер относится только к одному контрагенту.

## Сетевой периметр

Реализовано сейчас: сервер лицензирования слушает HTTP на `ip:port`. Домен и TLS вне текущей готовности, поэтому admin UI нельзя оставлять открытым всему интернету. Минимальный безопасный альфа-периметр: открыть только SSH и порт License Server, а доступ к порту License Server ограничить IP офиса/VPN/оператора.

Пример ниже использует SSH `22`, License Server `8095` и один доверенный IP оператора `203.0.113.10`. Заменить IP перед выполнением.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp
sudo ufw allow from 203.0.113.10 to any port 8095 proto tcp
sudo ufw enable
sudo ufw status verbose
```

Если временно нужно открыть License Server всем для smoke, команда ниже допустима только на короткое время:

```bash
sudo ufw allow 8095/tcp
```

Закрыть обратно:

```bash
sudo ufw delete allow 8095/tcp
sudo ufw allow from 203.0.113.10 to any port 8095 proto tcp
```

## Первый сервер

Создать пользователя и каталоги:

```bash
sudo useradd --system --home /var/lib/myhoreca/license-server --shell /usr/sbin/nologin mh-license
sudo install -d -o mh-license -g mh-license -m 0750 /var/lib/myhoreca/license-server
sudo install -d -o mh-license -g mh-license -m 0750 /var/backups/myhoreca/license-server/startup
sudo install -d -m 0755 /opt/myhoreca/license-server/releases
sudo install -d -m 0755 /etc/myhoreca/license-server
```

Установить production config:

```bash
sudo install -m 0640 -o root -g mh-license deploy/license-server/license-api.env.example /etc/myhoreca/license-server/license-api.env
sudoedit /etc/myhoreca/license-server/license-api.env
```

Минимальные значения:

```env
LICENSE_HTTP_ADDR=:8095
LICENSE_SQLITE_PATH=/var/lib/myhoreca/license-server/license-server.db
LICENSE_SQLITE_BACKUP_DIR=/var/backups/myhoreca/license-server/startup
LICENSE_SUPER_ADMIN_LOGIN=admin
LICENSE_SUPER_ADMIN_PASSWORD=replace-with-strong-password
```

Установить systemd unit:

```bash
sudo install -m 0644 deploy/license-server/license-api.service /etc/systemd/system/license-api.service
sudo systemctl daemon-reload
```

Если CD будет работать отдельным SSH-пользователем, создать deploy user и добавить его публичный ключ:

```bash
sudo adduser --disabled-password --gecos "" deploy-user
sudo install -d -m 0700 -o deploy-user -g deploy-user /home/deploy-user/.ssh
sudoedit /home/deploy-user/.ssh/authorized_keys
sudo chown deploy-user:deploy-user /home/deploy-user/.ssh/authorized_keys
sudo chmod 0600 /home/deploy-user/.ssh/authorized_keys
```

Первый binary можно поставить вручную:

```bash
cd license-server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o license-api ./cmd/license-api
cd ..
release="$(date -u +%Y%m%dT%H%M%SZ)-manual"
sudo install -d -m 0755 "/opt/myhoreca/license-server/releases/$release"
sudo install -m 0755 license-server/license-api "/opt/myhoreca/license-server/releases/$release/license-api"
sudo ln -sfn "/opt/myhoreca/license-server/releases/$release" /opt/myhoreca/license-server/current
sudo systemctl enable --now license-api
sudo systemctl status license-api --no-pager --full
```

Smoke:

```bash
curl -fsS http://127.0.0.1:8095/health
curl -fsS -c /tmp/license.cookies \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"replace-with-strong-password"}' \
  http://127.0.0.1:8095/api/v1/admin/login
curl -sS http://127.0.0.1:8095/api/v1/entitlements/tenant-alpha/cloud-alpha || true
curl -fsS -b /tmp/license.cookies http://127.0.0.1:8095/api/v1/servers
rm -f /tmp/license.cookies
```

Cloud/Edge runtime config до появления домена должен указывать на IP и порт:

```env
LICENSE_SERVER_URL=http://<server-ip>:8095
```

## Обновление без сброса данных

Runtime SQLite живет в `/var/lib/myhoreca/license-server/license-server.db`. Перед startup migration сервис делает backup `.db/.db-wal/.db-shm` в `LICENSE_SQLITE_BACKUP_DIR`. Обновление binary не должно удалять `/var/lib/myhoreca/license-server` и `/var/backups/myhoreca/license-server`.

Ручное обновление:

```bash
sudo systemctl stop license-api
release="$(date -u +%Y%m%dT%H%M%SZ)-manual"
sudo install -d -m 0755 "/opt/myhoreca/license-server/releases/$release"
sudo install -m 0755 license-api "/opt/myhoreca/license-server/releases/$release/license-api"
sudo ln -sfn "/opt/myhoreca/license-server/releases/$release" /opt/myhoreca/license-server/current
sudo systemctl start license-api
sudo systemctl status license-api --no-pager --full
curl -fsS http://127.0.0.1:8095/health
```

Rollback на предыдущий release:

```bash
ls -1 /opt/myhoreca/license-server/releases
sudo ln -sfn /opt/myhoreca/license-server/releases/<previous-release> /opt/myhoreca/license-server/current
sudo systemctl restart license-api
```

## CD через GitHub Actions

Подготовлен production-ready workflow template `deploy/license-server/deploy-license-server.workflow.yml`. Пока у пользователя нет GitHub permission на создание workflow files, активная папка `.github` не хранится в ветке. После выдачи прав нужно скопировать template в `.github/workflows/deploy-license-server.yml`.

Workflow запускается при push в ветку `production`, только если изменены `license-server/**`, `shared/platform/**`, deploy artifacts или сам workflow. Также доступен ручной `workflow_dispatch`.

Нужные GitHub Actions secrets:

| Secret | Назначение |
| --- | --- |
| `LICENSE_DEPLOY_HOST` | IP Linux VM |
| `LICENSE_DEPLOY_USER` | SSH user для деплоя |
| `LICENSE_DEPLOY_PORT` | SSH port, обычно `22` |
| `LICENSE_DEPLOY_SSH_KEY` | private key deploy user |
| `LICENSE_DEPLOY_KNOWN_HOSTS` | строка из `ssh-keyscan` для host key pinning |

Сформировать `known_hosts`:

```bash
ssh-keyscan -p 22 <server-ip>
```

Deploy user должен иметь SSH-доступ и passwordless sudo только на нужные команды. Минимальный sudoers-файл:

```bash
sudo visudo -f /etc/sudoers.d/myhoreca-license-deploy
```

```text
deploy-user ALL=(root) NOPASSWD: /usr/bin/install, /usr/bin/ln, /usr/bin/rm, /usr/bin/systemctl restart license-api, /usr/bin/systemctl status license-api
```

Создать ветку `production`, когда готовы включить CD:

```bash
mkdir -p .github/workflows
cp deploy/license-server/deploy-license-server.workflow.yml .github/workflows/deploy-license-server.yml
git checkout -b production
git push origin production
```

После этого push с изменениями в `license-server/**` соберет binary, прогонит `go test ./...`, загрузит release по SSH, переключит symlink и перезапустит `license-api`.

## Что остается перед постоянной эксплуатацией

- Домен и TLS через reverse proxy. До этого admin UI должен быть закрыт firewall/VPN/IP allowlist.
- Регулярный внешний backup и restore rehearsal.
- Durable audit изменений лицензий в БД, если structured logs недостаточно для владельца.
- Управление дополнительными operator users из UI/API.
