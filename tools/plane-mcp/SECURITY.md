# Security для Plane MCP

## Правила обращения с токеном

- Не хранить `PLANE_API_KEY` в Git, README, `.env.example`, `mcp.example.json`, логах, тестах, shell history или issue-комментариях.
- Каждый разработчик использует личный Plane Personal Access Token.
- Не использовать общий team-token или общий Workspace Access Token для локальных AI-агентов.
- Не передавать токен через CLI arguments, потому что аргументы могут попасть в shell history и список процессов.
- Передавать токен только через локальную переменную окружения `PLANE_API_KEY` или локальный `.env`, который игнорируется Git.
- Не логировать полный `env`, MCP config, request headers или debug dumps, если там может быть токен.

## Ограничение прав агента

- Не давать MCP больше прав, чем нужно для текущей задачи разработки.
- Не подключать Plane MCP к непроверенным локальным или удаленным агентам.
- Не разрешать агенту удаление work items, projects, modules, cycles, milestones, labels, states, comments, links, relations и участников.
- Не разрешать агенту автоматический переход задачи в `Done`.
- Не разрешать агенту автоматический merge, push или release.
- Любые destructive operations выполняются только вручную владельцем задачи после отдельного подтверждения.

## Инциденты

Если токен мог попасть в Git, логи, историю терминала, чат, скриншот или внешний сервис:

1. Немедленно отозвать PAT в Plane.
2. Создать новый личный PAT.
3. Проверить `git status`, `git diff`, логи терминала и локальные MCP configs.
4. Если токен попал в историю Git, считать его скомпрометированным даже после удаления строки из файла.

## Проверенный источник

Для `mh-pos` разрешен только официальный Plane MCP Server:

- документация: https://developers.plane.so/dev-tools/mcp-server
- репозиторий: https://github.com/makeplane/plane-mcp-server
- PyPI package: `plane-mcp-server`

Старый npm-пакет `@makeplane/plane-mcp-server` не использовать для нового подключения: в README официального репозитория он помечен как deprecated.
