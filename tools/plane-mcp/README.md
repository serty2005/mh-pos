# Plane MCP для mh-pos

## Назначение

Эта папка хранит локальную конфигурацию и инструкции для подключения официального Plane MCP Server к проекту `mh-pos`. Интеграция нужна, чтобы локальные AI-агенты разработчиков могли читать задачи Plane, добавлять комментарии и выполнять ограниченные обновления статуса во время разработки.

Runtime-код `mh-pos` не использует эти файлы.

## Быстрый старт для разработчика

Нужно один раз настроить локальный MCP-клиент, затем начинать каждую задачу с Plane identifier вроде `MHPOS-42`.

1. Получить доступ к Plane workspace `myhoreca-pos` и project `562fe804-ecc3-41df-b85d-c981e6c13760`.
2. Создать личный Plane Personal Access Token в профиле пользователя.
3. Установить Python 3.10+ и `uv`/`uvx`.
4. Скопировать `tools/plane-mcp/.env.example` в локальный `tools/plane-mcp/.env` или выставить env в shell.
5. Подключить MCP server к своему агенту по примеру из `mcp.example.json`.
6. Проверить доступ через `curl` и read smoke-test из `smoke-test.md`.
7. Перед работой дать агенту Plane identifier задачи и стартовый промпт из этого README.

Минимальная shell-настройка для WSL/Linux:

```bash
cd /home/serty/repos/mh-pos
export PLANE_BASE_URL="https://dev.serty.top"
export PLANE_WORKSPACE_SLUG="myhoreca-pos"
export PLANE_PROJECT_ID="562fe804-ecc3-41df-b85d-c981e6c13760"
read -rsp "Plane API key: " PLANE_API_KEY
echo
export PLANE_API_KEY
```

Значение `PLANE_API_KEY` не добавлять в `mcp.example.json`, README, issue comments или shell-команды, которые попадут в history.

## Рабочий цикл задачи

Каждая задача разработки должна иметь один Plane work item. Агент работает только в scope этой задачи, если пользователь явно не расширил его.

1. Найти задачу через `retrieve_work_item_by_identifier`.
2. Прочитать описание, comments, state, labels, module, cycle, links и relations.
3. Проверить `git status` и профильные документы проекта.
4. Если задача готова к работе, перевести ее `Ready -> In Progress`.
5. Выполнить изменения и профильные проверки.
6. Добавить итоговый комментарий в Plane: что изменено, файлы, проверки, риски, что вне scope.
7. Перевести задачу `In Progress -> Review`.

`Done` выставляет только владелец задачи после review/merge. Агент не делает merge, release или deployment через Plane-MCP.

Рекомендуемый формат итогового Plane comment:

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

## Официальный сервер

- Документация: https://developers.plane.so/dev-tools/mcp-server
- Репозиторий: https://github.com/makeplane/plane-mcp-server
- PyPI package: `plane-mcp-server`
- Проверенная версия: `0.2.8`, release от 2026-03-23.
- Дата последней проверки для `mh-pos`: 2026-06-10.
- Лицензия: MIT.
- Запуск для локального self-hosted workflow: `uvx plane-mcp-server==0.2.8 stdio`.

Старый npm-пакет `@makeplane/plane-mcp-server` не используется: официальный README помечает Node.js-based версию как deprecated и рекомендует Python/FastMCP implementation.

## Проверка источника

| Проверка | Результат |
| --- | --- |
| Официальная документация Plane | `developers.plane.so/dev-tools/mcp-server` |
| Официальная GitHub-организация | `github.com/makeplane/plane-mcp-server` |
| Официальный пакет | PyPI `plane-mcp-server`, maintainer `makeplane` |
| История releases | Есть, актуальная проверенная версия `0.2.8` |
| README и license | Есть README, SECURITY.md, CONTRIBUTING.md, MIT license |
| Передача token | Через env для stdio или headers для HTTP/PAT; для `mh-pos` используем env |
| Доступ к БД Plane | Не требуется, сервер работает через Plane HTTPS API |
| Self-hosted Plane | Поддерживается через `PLANE_BASE_URL` |
| Hidden SaaS endpoints | Для локального stdio ожидается только `PLANE_BASE_URL`; hosted HTTP/OAuth использует `mcp.plane.so`, но он вне текущего local self-hosted workflow |

## Transport modes

| Transport | Auth | Применимость для `mh-pos` |
| --- | --- | --- |
| Local stdio | env: `PLANE_API_KEY`, `PLANE_WORKSPACE_SLUG`, `PLANE_BASE_URL` | Основной режим для локальных AI-агентов и self-hosted Plane |
| Streamable HTTP / HTTP with OAuth | Browser OAuth | Для Plane Cloud или собственного hosted MCP, вне текущего локального подключения |
| Streamable HTTP / HTTP with PAT | request headers | Для автоматизации, но не выбран из-за риска хранения токена в client config |
| SSE legacy | OAuth | Только для старых интеграций, не использовать для нового подключения |

## Требования

- Python 3.10+.
- `uv`/`uvx`.
- Доступ к `https://dev.serty.top`.
- Личный Plane Personal Access Token.
- Членство пользователя в workspace `myhoreca-pos` и project `562fe804-ecc3-41df-b85d-c981e6c13760`.

## Переменные окружения

| Переменная | Нужна серверу | Значение |
| --- | --- | --- |
| `PLANE_BASE_URL` | Да для self-hosted | `https://dev.serty.top` |
| `PLANE_WORKSPACE_SLUG` | Да | `myhoreca-pos` |
| `PLANE_API_KEY` | Да | Личный PAT, не хранить в Git |
| `PLANE_PROJECT_ID` | Нет, локальная convention | `562fe804-ecc3-41df-b85d-c981e6c13760` |

`PLANE_PROJECT_ID` добавлен для промптов и локальной дисциплины scope. Официальный MCP Server может его игнорировать, поэтому агенту нужно явно передавать `project_id` в tool calls.

## Как получить токен Plane

1. Открыть Plane instance `https://dev.serty.top`.
2. Перейти в профиль пользователя.
3. Открыть API Tokens или Personal Access Tokens.
4. Создать личный token с минимальными правами, достаточными для чтения задач, комментариев и разрешенных обновлений.
5. Скопировать token один раз и сохранить только в локальном secret storage или локальном `.env`.

Не использовать общий team-token.

## Что нужно для старта конкретной задачи

Разработчику или агенту достаточно получить от владельца задачи:

- Plane identifier задачи, например `MHPOS-42`;
- ожидаемую ветку или правило именования ветки, если оно задано командой;
- подтверждение, можно ли агенту менять state задачи в Plane;
- список обязательных проверок, если задача требует больше стандартного набора из `AGENTS.md`.

Если Plane identifier не передан, агент не должен сам выбирать ближайшую похожую задачу. Сначала нужно попросить пользователя указать identifier или создать отдельный work item.

## Настройка WSL

```bash
export PLANE_BASE_URL="https://dev.serty.top"
export PLANE_WORKSPACE_SLUG="myhoreca-pos"
export PLANE_PROJECT_ID="562fe804-ecc3-41df-b85d-c981e6c13760"
read -rsp "Plane API key: " PLANE_API_KEY
echo
export PLANE_API_KEY
```

Если используется локальный `.env`, создать `tools/plane-mcp/.env` вручную по образцу `.env.example`. Файл игнорируется Git и не должен попадать в commit.

## Проверка токена через curl

```bash
curl --fail-with-body \
  --silent \
  --show-error \
  --header "X-API-Key: ${PLANE_API_KEY}" \
  --header "Accept: application/json" \
  "https://dev.serty.top/api/v1/workspaces/myhoreca-pos/projects/562fe804-ecc3-41df-b85d-c981e6c13760/" \
  | python -m json.tool
```

Ожидается JSON с данными проекта. HTML вместо JSON обычно означает неверный base URL, reverse proxy route или redirect на login page.

## Ручной запуск

```bash
cd /home/serty/repos/mh-pos
export PLANE_BASE_URL="https://dev.serty.top"
export PLANE_WORKSPACE_SLUG="myhoreca-pos"
export PLANE_PROJECT_ID="562fe804-ecc3-41df-b85d-c981e6c13760"
read -rsp "Plane API key: " PLANE_API_KEY
echo
export PLANE_API_KEY
uvx plane-mcp-server==0.2.8 stdio
```

В ручном запуске процесс ожидает MCP stdio-сообщения от клиента. Для реальной проверки удобнее использовать MCP Inspector или локальный агент.

## Подключение к локальному агенту

Базовый пример находится в `mcp.example.json`.

```json
{
  "mcpServers": {
    "plane-myhoreca-pos": {
      "command": "uvx",
      "args": ["plane-mcp-server==0.2.8", "stdio"],
      "env": {
        "PLANE_BASE_URL": "https://dev.serty.top",
        "PLANE_WORKSPACE_SLUG": "myhoreca-pos",
        "PLANE_PROJECT_ID": "562fe804-ecc3-41df-b85d-c981e6c13760"
      }
    }
  }
}
```

`PLANE_API_KEY` намеренно не записан в JSON. Перед запуском агента переменная должна быть выставлена в пользовательском окружении. Если конкретный MCP client не наследует env, токен нужно передать через user-local secret config клиента, который не хранится в репозитории.

Для Claude Code локальный вариант через JSON:

```bash
claude mcp add-json plane-myhoreca-pos '{
  "type": "stdio",
  "command": "uvx",
  "args": ["plane-mcp-server==0.2.8", "stdio"],
  "env": {
    "PLANE_BASE_URL": "https://dev.serty.top",
    "PLANE_WORKSPACE_SLUG": "myhoreca-pos",
    "PLANE_PROJECT_ID": "562fe804-ecc3-41df-b85d-c981e6c13760"
  }
}'
```

После подключения проверить:

```bash
claude mcp list
claude mcp get plane-myhoreca-pos
```

Для Cursor, VS Code, Windsurf, Zed и Claude Desktop схема та же: stdio server запускает `uvx`, `PLANE_BASE_URL` и `PLANE_WORKSPACE_SLUG` лежат в user-local MCP config, `PLANE_API_KEY` передается только через локальный secret/env. Не добавлять `.vscode/mcp.json`, `.cursor/mcp.json` или другие user-local configs в репозиторий, если в них есть token или пользовательские настройки.

## Ожидаемые tools/resources

Официальная документация для версии `0.2.8` описывает 100+ tools across 20 modules. В `MCP Server for Claude Code` страница может показывать более короткий summary 55+ tools; для `mh-pos` считать актуальной подробную tool reference основного MCP Server документа.

Основные категории tools:

| Категория | Tools |
| --- | --- |
| Users | `get_me` |
| Workspaces | `get_workspace_members`, `get_workspace_features`, `update_workspace_features` |
| Projects | `list_projects`, `create_project`, `retrieve_project`, `update_project`, `delete_project`, `get_project_worklog_summary`, `get_project_members`, `get_project_features`, `update_project_features` |
| Work Items | `list_work_items`, `create_work_item`, `retrieve_work_item`, `retrieve_work_item_by_identifier`, `update_work_item`, `delete_work_item`, `search_work_items` |
| Cycles | `list_cycles`, `create_cycle`, `retrieve_cycle`, `update_cycle`, `delete_cycle`, `list_archived_cycles`, `add_work_items_to_cycle`, `remove_work_item_from_cycle`, `list_cycle_work_items`, `transfer_cycle_work_items`, `archive_cycle`, `unarchive_cycle` |
| Modules | `list_modules`, `create_module`, `retrieve_module`, `update_module`, `delete_module`, `list_archived_modules`, `add_work_items_to_module`, `remove_work_item_from_module`, `list_module_work_items`, `archive_module`, `unarchive_module` |
| Initiatives | `list_initiatives`, `create_initiative`, `retrieve_initiative`, `update_initiative`, `delete_initiative` |
| Intake | `list_intake_work_items`, `create_intake_work_item`, `retrieve_intake_work_item`, `update_intake_work_item`, `delete_intake_work_item` |
| Work Item Properties | `list_work_item_properties`, `create_work_item_property`, `retrieve_work_item_property`, `update_work_item_property`, `delete_work_item_property` |
| Epics | `list_epics`, `create_epic`, `retrieve_epic`, `update_epic`, `delete_epic` |
| Milestones | `list_milestones`, `create_milestone`, `retrieve_milestone`, `update_milestone`, `delete_milestone`, `add_work_items_to_milestone`, `remove_work_items_from_milestone`, `list_milestone_work_items` |
| Labels | `list_labels`, `create_label`, `retrieve_label`, `update_label`, `delete_label` |
| States | `list_states`, `create_state`, `retrieve_state`, `update_state`, `delete_state` |
| Work Item Comments | `list_work_item_comments`, `retrieve_work_item_comment`, `create_work_item_comment`, `update_work_item_comment`, `delete_work_item_comment` |
| Work Item Links | `list_work_item_links`, `retrieve_work_item_link`, `create_work_item_link`, `update_work_item_link`, `delete_work_item_link` |
| Work Item Types | `list_work_item_types`, `create_work_item_type`, `retrieve_work_item_type`, `update_work_item_type`, `delete_work_item_type` |
| Work Item Relations | `list_work_item_relations`, `create_work_item_relation`, `remove_work_item_relation` |
| Work Item Activities | `list_work_item_activities`, `retrieve_work_item_activity` |
| Work Logs | `list_work_logs`, `create_work_log`, `update_work_log`, `delete_work_log` |
| Pages | `retrieve_workspace_page`, `retrieve_project_page`, `create_workspace_page`, `create_project_page` |

Официальная документация не фиксирует отдельный список MCP resources. Проверять resources нужно фактическим `list resources` в MCP Inspector для установленной версии.

## Разрешенные операции агента

- Читать project context, work items, comments, labels, states, modules, cycles, links, relations и activities.
- Искать work items по query и filters.
- Добавлять комментарии к текущей задаче.
- Переводить текущую тестовую или назначенную задачу `Ready -> In Progress` и `In Progress -> Review`, если это явно входит в задачу.
- Добавлять links, labels, module или cycle только если это явно указано в work item или подтверждено пользователем.
- Создавать subtask или related work item только по явному требованию пользователя.

## Запрещенные операции агента

- Удалять work items, projects, modules, cycles, milestones, labels, states, comments, links, relations и участников.
- Автоматически переводить задачи в `Done`.
- Автоматически делать merge, push, release или deployment.
- Менять project/workspace feature flags без отдельного подтверждения.
- Массово переносить задачи между cycles/modules без отдельного подтверждения.
- Выполнять write smoke-test на продуктовой задаче.

## Правила синхронизации с Git

- Branch/commit/PR naming не выводится из Plane автоматически, если команда не задала отдельное правило.
- Plane identifier рекомендуется включать в название ветки и PR title, например `feature/MHPOS-42-plane-mcp-onboarding`.
- В Plane comment не писать secrets, raw env, request dumps, PIN, tokens, credentials и sensitive payloads.
- Если изменения затрагивают HTTP routes, payloads, UI flows, permission model, DB schema, sync events, error/logging contracts или migration policy, обновить профильную документацию по правилам `AGENTS.md`.
- Если задача оказалась шире описания Plane, сначала оставить комментарий или спросить владельца, а не расширять scope молча.

## Пригодность под workflow mh-pos

| Нужная операция | Есть в официальном MCP | Tool name | Комментарий |
| --- | --- | --- | --- |
| Получить project context | Да | `retrieve_project`, `get_project_members`, `get_project_features` | Нужно передать `project_id` |
| Получить work item по identifier | Да | `retrieve_work_item_by_identifier` | Использует project identifier и sequence number, например `MHPOS-42` |
| Получить список work items | Да | `list_work_items` | Можно фильтровать по state, label, module, cycle |
| Получить comments | Да | `list_work_item_comments` | Нужен UUID work item |
| Добавить comment | Да | `create_work_item_comment` | Разрешено для итогового отчета агента |
| Изменить state | Да | `list_states`, `update_work_item` | Агент должен сначала найти UUID state |
| Добавить link | Да | `create_work_item_link` | Только по явному требованию |
| Создать subtask или related work item | Да | `create_work_item`, `create_work_item_relation` | Для subtask используется `parent_id`; relation отдельным tool |
| Добавить labels | Да | `list_labels`, `update_work_item` | Создание новых labels запрещено без подтверждения |
| Добавить module | Да | `list_modules`, `add_work_items_to_module` | Если modules disabled, включение feature запрещено без подтверждения |
| Добавить cycle | Да | `list_cycles`, `add_work_items_to_cycle` | Если cycles disabled, включение feature запрещено без подтверждения |

На текущий минимальный workflow официального MCP достаточно. Тонкий wrapper-MCP не нужен. Если позже понадобится операция, отсутствующая в официальном MCP, варианты workaround: выполнить вручную, добавить REST bootstrap-скрипт, открыть issue в Plane MCP или написать узкий wrapper-MCP только для отсутствующей операции после отдельного решения.

## Scope и ограничения

Официальный сервер scope не ограничивает одним project на уровне transport. Он работает в рамках прав Plane PAT и workspace slug. Для `mh-pos` project scope задается дисциплиной промпта, `PLANE_PROJECT_ID` и явным `project_id` в tool calls.

Для self-hosted Plane известный риск: если пользователь имеет workspace-level доступ, но не добавлен в project members, `retrieve_project` и work item tools могут возвращать `404`. Добавить пользователя в project members или проверить роль в Plane UI.

## Smoke-test

Подробный сценарий находится в `smoke-test.md`.

Кратко:

1. Выставить переменные окружения.
2. Проверить токен через `curl`.
3. Подключить MCP Inspector или локальный агент.
4. Проверить список tools/resources.
5. Получить project context и work item по identifier.
6. На отдельной задаче `MCP smoke test` добавить тестовый комментарий.
7. При необходимости перевести тестовую задачу `Ready -> In Progress`.
8. Вернуть state вручную, если требуется.
9. Убедиться, что действия видны в Plane UI.

## Troubleshooting

| Симптом | Вероятная причина | Что проверить |
| --- | --- | --- |
| 401 | Неверный или отозванный `PLANE_API_KEY` | Пересоздать личный PAT и повторить curl |
| 403 | Недостаточные права | Проверить роль пользователя в workspace/project |
| 404 | Неверный workspace slug, project id или нет membership в project | Сверить `myhoreca-pos`, `562fe804-ecc3-41df-b85d-c981e6c13760`, membership |
| HTML вместо JSON | Попали на frontend/login/redirect, а не API response | Проверить `PLANE_BASE_URL`, reverse proxy и `/api/v1/...` route |
| Cycles are not enabled | Cycles выключены в project features | Не включать автоматически; попросить project admin |
| Rate limit | Слишком частые tool calls | Уменьшить polling, повторить позже |
| Неверный workspace slug | Slug отличается от URL | Открыть workspace URL и сверить сегмент пути |
| Неверный project id | Использован identifier вместо UUID или project удален | Взять UUID из Plane UI/API |
| MCP server timeout | `uvx` долго подтягивает package или нет Python/uv | Проверить `python --version`, `uvx --version`, увеличить MCP timeout |
| `PLANE_API_KEY` не виден серверу | MCP client не наследует shell env | Настроить user-local secret config клиента |

## Стартовый промпт для локального агента

```text
Начни работу над Plane work item <IDENTIFIER>.

Сначала получи задачу через Plane MCP.
Проверь state, module, cycle, labels, описание, комментарии и зависимости.
Прочитай AGENTS.md и связанные документы.
Проверь git status.
Не расширяй scope.
Если задача готова к работе, переведи её в In Progress.
Выполни изменения.
Запусти проверки.
Добавь итоговый комментарий в Plane.
Переведи задачу в Review.
Не выполняй merge и не переводи задачу в Done.
```
