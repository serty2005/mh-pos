# Traefik production VM deployment

Статус: `реализовано сейчас` как общий reverse proxy для alpha/pre-Kubernetes Cloud-стеков.

Traefik публикует наружу только `80/443`, читает Docker labels и не публикует контейнеры автоматически, потому что включен `providers.docker.exposedbydefault=false`.

## Запуск

```bash
cp deploy/traefik/traefik.env.example deploy/traefik/traefik.env
docker network create traefik_proxy
docker compose -f deploy/traefik/docker-compose.traefik.yml --env-file deploy/traefik/traefik.env up -d
```

`deploy/traefik/traefik.env` содержит email ACME и basic auth dashboard, поэтому рабочий файл не хранится в Git. Состояние Let's Encrypt лежит в Docker volume `traefik_letsencrypt`; `acme.json` не коммитится.

## Dashboard

Dashboard доступен только по отдельному host из `TRAEFIK_DASHBOARD_DOMAIN`, через HTTPS и basic auth. Для production сгенерируйте hash:

```bash
htpasswd -nbB admin 'strong-password' | sed -e 's/\$/$$/g'
```

Полученную строку положите в `TRAEFIK_DASHBOARD_BASIC_AUTH`.
