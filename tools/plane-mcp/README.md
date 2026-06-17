# Plane MCP для mh-pos

## Назначение

`tools/plane-mcp` хранит локальную настройку официального Plane MCP Server для проекта `mh-pos`.

Интеграция нужна, чтобы локальные AI-агенты могли читать Plane work items, комментарии, state/module/cycle/labels, добавлять итоговые комментарии и выполнять ограниченные обновления статуса во время разработки.

Runtime-код `mh-pos` эти файлы не использует.

## Текущий Формат

Основной формат подключения:

1. Secrets и параметры Plane лежат в локальном `tools/plane-mcp/.env`.
2. MCP client запускает только `tools/plane-mcp/run-stdio.sh`.
3. `run-stdio.sh` читает `.env` и выполняет `uvx plane-mcp-server==0.2.8 stdio`.
4. `PLANE_API_KEY` не записывается в `.codex/config.toml`, `mcp.example.json`, README, issue comments или shell history.

Такой формат выбран потому, что MCP clients не всегда наследуют shell env. Локальный launcher делает запуск повторяемым и не дублирует token в конфиг клиента.

## Быстрый Старт

1. Получить доступ к Plane workspace `myhoreca-pos` и project `562fe804-ecc3-41df-b85d-c981e6c13760`.
2. Создать личный Plane Personal Access Token в профиле пользователя.
3. Установить Python 3.10+ и `uv`/`uvx`.
4. Создать локальный env:

```bash
cp tools/plane-mcp/.env.example tools/plane-mcp/.env
```

5. Заполнить `tools/plane-mcp/.env`:

```dotenv
PLANE_BASE_URL=https://dev.serty.top
PLANE_WORKSPACE_SLUG=myhoreca-pos
PLANE_PROJECT_ID=562fe804-ecc3-41df-b85d-c981e6c13760
PLANE_API_KEY=<personal-token>
```

6. Проверить API доступ:

```bash
set -a
source tools/plane-mcp/.env
set +a

curl --fail-with-body \
  --silent \
  --show-error \
  --header "X-API-Key: ${PLANE_API_KEY}" \
  --header "Accept: application/json" \
  "${PLANE_BASE_URL}/api/v1/workspaces/${PLANE_WORKSPACE_SLUG}/projects/${PLANE_PROJECT_ID}/" \
  | python3 -m json.tool
```

Ожидается JSON проекта `POS`. HTML вместо JSON обычно означает неверный base URL, proxy route или redirect на login page.

7. Проверить launcher:

```bash
tools/plane-mcp/run-stdio.sh
```

В ручном запуске процесс ожидает MCP stdio-сообщения. Это нормально. Для выхода нажать `Ctrl+C`.

## Подключение К Codex

Для этого checkout уже используется такой проектный пример:

```toml
[mcp_servers.plane]
command = "/home/master/repos/myhoreca-pos/tools/plane-mcp/run-stdio.sh"
startup_timeout_sec = 30
tool_timeout_sec = 120
```

Для глобального Codex config добавить тот же блок в `/home/master/.codex/config.toml`.

Если репозиторий находится не в `/home/master/repos/myhoreca-pos`, заменить `command` на абсолютный путь к своему `tools/plane-mcp/run-stdio.sh`.

После изменения MCP config нужно открыть новую Codex-сессию: список MCP tools загружается на старте сессии.

## Подключение К Другим MCP Clients

Использовать launcher как stdio command:

```json
{
  "mcpServers": {
    "plane-myhoreca-pos": {
      "command": "/home/master/repos/myhoreca-pos/tools/plane-mcp/run-stdio.sh"
    }
  }
}
```

`mcp.example.json` содержит такой же минимальный пример.

Для Claude Code:

```bash
claude mcp add-json plane-myhoreca-pos '{
  "type": "stdio",
  "command": "/home/master/repos/myhoreca-pos/tools/plane-mcp/run-stdio.sh"
}'
```

Проверка:

```bash
claude mcp list
claude mcp get plane-myhoreca-pos
```

## Официальный Сервер

- Документация: https://developers.plane.so/dev-tools/mcp-server
- Репозиторий: https://github.com/makeplane/plane-mcp-server
- PyPI package: `plane-mcp-server`
- Проверенная версия: `0.2.8`
- Локальный запуск: `uvx plane-mcp-server==0.2.8 stdio`
- Лицензия: MIT

Старый npm-пакет `@makeplane/plane-mcp-server` не используется: Node.js-based версия deprecated, текущий официальный сервер основан на Python/FastMCP.

## Переменные

| Переменная | Назначение |
| --- | --- |
| `PLANE_BASE_URL` | Self-hosted Plane URL, сейчас `https://dev.serty.top` |
| `PLANE_WORKSPACE_SLUG` | Workspace slug, сейчас `myhoreca-pos` |
| `PLANE_PROJECT_ID` | Project UUID для локальной дисциплины scope |
| `PLANE_API_KEY` | Личный PAT, хранится только в локальном `.env` |

`PLANE_PROJECT_ID` официальный MCP server может игнорировать. Агент должен явно передавать `project_id` в tool calls.

## Ожидаемые Tools

Smoke-test текущей настройки возвращает 100+ MCP tools. Основные категории:

- projects;
- work items;
- states;
- labels;
- modules;
- cycles;
- comments;
- links;
- relations;
- activities;
- work logs;
- pages;
- workspace/project members and features.

Минимально нужные tools для разработки:

| Операция | Tool |
| --- | --- |
| Получить project context | `retrieve_project` |
| Получить задачу по identifier | `retrieve_work_item_by_identifier` |
| Найти задачи | `list_work_items`, `search_work_items` |
| Прочитать comments | `list_work_item_comments` |
| Добавить итоговый comment | `create_work_item_comment` |
| Найти state | `list_states` |
| Обновить state назначенной задачи | `update_work_item` |

## Рабочий Цикл Задачи

Каждая задача разработки должна иметь один Plane work item. Агент работает только в scope этой задачи, если пользователь явно не расширил scope.

1. Получить задачу через `retrieve_work_item_by_identifier`.
2. Прочитать description, comments, state, labels, module, cycle, links и relations.
3. Проверить `git status` и профильные документы проекта.
4. Если задача готова к работе и это разрешено, перевести `Ready -> In Progress`.
5. Выполнить изменения и профильные проверки.
6. Добавить итоговый Plane comment.
7. Если это разрешено, перевести `In Progress -> Review`.

`Done` выставляет только владелец задачи после review/merge. Агент не делает merge, release или deployment через Plane MCP.

## Формат Итогового Plane Comment

```text
Выполнено:
- ...

Измененные файлы:
- ...

Проверки:
- ...

Не запускалось:
- ...

Оставшиеся риски:
- ...

Вне scope:
- ...
```

## Разрешенные Операции Агента

- Читать project context, work items, comments, labels, states, modules, cycles, links, relations и activities.
- Искать work items по query и filters.
- Добавлять комментарии к текущей задаче.
- Переводить текущую назначенную задачу `Ready -> In Progress` и `In Progress -> Review`, если это входит в задачу.
- Добавлять links, labels, module или cycle только если это явно указано в work item или подтверждено пользователем.
- Создавать subtask или related work item только по явному требованию пользователя.

## Запрещенные Операции Агента

- Удалять work items, projects, modules, cycles, milestones, labels, states, comments, links, relations и участников.
- Автоматически переводить задачи в `Done`.
- Делать merge, push, release или deployment.
- Менять project/workspace feature flags без отдельного подтверждения.
- Массово переносить задачи между cycles/modules без отдельного подтверждения.
- Выполнять write smoke-test на продуктовой задаче.

## Smoke-Test

Read-only MCP smoke-test без изменения Plane:

1. Открыть новую сессию агента после настройки MCP config.
2. Проверить, что доступен Plane MCP server.
3. Выполнить `retrieve_project` с `project_id=562fe804-ecc3-41df-b85d-c981e6c13760`.
4. Выполнить `list_work_items`.
5. Выполнить `retrieve_work_item_by_identifier` для известной тестовой задачи.
6. Выполнить `list_work_item_comments`.

Write smoke-test выполнять только на отдельной задаче `MCP smoke test` и только после подтверждения пользователя.

## Troubleshooting

| Симптом | Вероятная причина | Что проверить |
| --- | --- | --- |
| Plane tools не появились в новой сессии | MCP config не подключен или сессия не перезапущена | `/home/master/.codex/config.toml`, `.codex/config.toml`, новый старт сессии |
| `PLANE_API_KEY` missing | `.env` не заполнен или launcher запущен не из repo | `tools/plane-mcp/.env`, путь в `command` |
| 401 | Неверный или отозванный PAT | Пересоздать token и повторить API check |
| 403 | Недостаточные права | Проверить роль пользователя в workspace/project |
| 404 | Неверный workspace slug/project id или нет membership | Сверить `.env` и membership в Plane UI |
| HTML вместо JSON | Попали на frontend/login/redirect | Проверить `PLANE_BASE_URL` и `/api/v1/...` route |
| MCP server timeout | `uvx` долго подтягивает package или нет Python/uv | `python3 --version`, `uvx --version`, timeout в MCP config |
| Tools есть, но не та project scope | MCP server не ограничивает project на transport level | Всегда передавать `project_id` из `.env` |

## Стартовый Промпт Для Агента

```text
Начни работу над Plane work item <IDENTIFIER>.

Сначала получи задачу через Plane MCP.
Проверь state, module, cycle, labels, описание, комментарии и зависимости.
Прочитай AGENTS.md и связанные документы.
Проверь git status.
Не расширяй scope.
Если задача готова к работе и это разрешено, переведи её в In Progress.
Выполни изменения.
Запусти проверки.
Добавь итоговый комментарий в Plane.
Если это разрешено, переведи задачу в Review.
Не выполняй merge и не переводи задачу в Done.
```
