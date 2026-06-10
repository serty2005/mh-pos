# Smoke-test Plane MCP

Тест проверяет, что локальный агент может читать Plane и выполнять ограниченные write-операции на отдельной тестовой задаче.

Не выполнять write smoke-test на реальной продуктовой задаче без подтверждения пользователя. Для write-проверки пользователь должен создать отдельный work item `MCP smoke test`.

## 1. Подготовить окружение

```bash
export PLANE_BASE_URL="https://dev.serty.top"
export PLANE_WORKSPACE_SLUG="myhoreca-pos"
export PLANE_PROJECT_ID="562fe804-ecc3-41df-b85d-c981e6c13760"
read -rsp "Plane API key: " PLANE_API_KEY
echo
export PLANE_API_KEY
```

Проверить токен через Plane API:

```bash
curl --fail-with-body \
  --silent \
  --show-error \
  --header "X-API-Key: ${PLANE_API_KEY}" \
  --header "Accept: application/json" \
  "https://dev.serty.top/api/v1/workspaces/myhoreca-pos/projects/562fe804-ecc3-41df-b85d-c981e6c13760/" \
  | python -m json.tool
```

## 2. Запустить MCP server

```bash
uvx plane-mcp-server==0.2.8 stdio
```

Для ручного запуска команда будет ждать MCP stdio-сообщения от клиента. Это нормально.

## 3. Подключить MCP Inspector или локальный агент

Использовать пример из `mcp.example.json`. Если клиент не наследует переменные окружения родительского shell, добавить `PLANE_API_KEY` только в user-local secret config клиента, не в файлы репозитория.

## 4. Проверить tools/resources

В MCP Inspector или локальном агенте открыть список tools. Ожидаются категории:

- users;
- workspaces;
- projects;
- work items;
- cycles;
- modules;
- initiatives;
- intake;
- work item properties;
- epics;
- milestones;
- labels;
- states;
- work item comments;
- work item links;
- work item types;
- work item relations;
- work item activities;
- work logs;
- pages.

Официальная документация не фиксирует отдельный список MCP resources; если Inspector показывает resources, сверить их с текущей версией сервера.

## 5. Read smoke-test

1. Получить project context через `retrieve_project` с `project_id=562fe804-ecc3-41df-b85d-c981e6c13760`.
2. Получить список work items через `list_work_items`.
3. Получить work item по identifier первой тестовой задачи через `retrieve_work_item_by_identifier`.
4. Получить comments через `list_work_item_comments`.
5. Проверить, что данные совпадают с Plane UI.

## 6. Write smoke-test

Выполнять только на задаче `MCP smoke test`.

1. Добавить тестовый comment через `create_work_item_comment`.
2. Найти state `In Progress` через `list_states`.
3. Если задача находится в `Ready`, перевести ее в `In Progress` через `update_work_item`.
4. Если нужно, вручную вернуть исходное состояние в Plane UI.
5. Убедиться, что comment и state change видны в Plane UI.

## 7. Что не проверять без отдельного подтверждения

- delete operations;
- archive/unarchive operations;
- изменение project/module/cycle settings;
- массовые transfer operations между cycles;
- перевод в `Done`;
- любые действия на продуктовых задачах.
