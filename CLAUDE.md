@AGENTS.md

## Plane MCP

Задачи ведутся в Plane (проект `myhoreca-pos`). Для работы с задачами доступны инструменты MCP `plane-myhoreca-pos`:

- `retrieve_work_item_by_identifier` — получить задачу по ID (например, `POS-85`)
- `update_work_item_state` — сменить статус (`Ready → In Progress → Review`)
- `create_work_item_comment` — добавить итоговый комментарий

Workflow: получить задачу → перевести в `In Progress` → выполнить → добавить comment в формате AGENTS.md → перевести в `Review`. Никогда не переводить в `Done`, не делать merge.

## Playwright MCP

Для проверки UI-изменений доступен Playwright MCP (`playwright`). Использовать при изменениях в `pos-ui-g` или `cloud-ui-g`.

## CodeGraph MCP

Для навигации по кодовой базе доступен CodeGraph MCP (`codegraph`). Инструменты: `codegraph_explore`, `codegraph_search`, `codegraph_node`, `codegraph_callers`, `codegraph_callees`, `codegraph_impact`, `codegraph_files`, `codegraph_status`.

## Порты dev-стека

- cloud-backend: 8090
- pos-backend: 8080
- license-server: 8095
- PostgreSQL: 5432
- ClickHouse: 8123 / 9000
