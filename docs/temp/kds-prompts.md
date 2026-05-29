Ниже набор универсальных промптов. Их можно выдавать агенту по одному, последовательно.

**Общий Префикс**

```text
Мы работаем в репозитории /home/serty/repos/mh-pos над RMS-POS для ресторанов.

Перед изменениями обязательно:
1. Прочитай AGENTS.md.
2. Прочитай docs/backend/KITCHEN-PROCESSES-SPEC.md.
3. Проверь текущее состояние кода и документации, не полагайся на старые выводы.
4. Используй CodeGraph для структурного анализа, rg только для текстовых поисков.
5. Не откатывай чужие изменения.
6. После реализации обнови профильную документацию, затронутую изменением.
7. В финале укажи: что найдено, что изменено, файлы, проверки, риски, что дальше, что вне объема, затрагивался ли runtime code.
```

**Итерация 1: POS Edge KDS Lifecycle**
Выполнена.

**Итерация 2: POS Edge Kitchen Stock Events**
Выполнена.

**Итерация 3: POS Edge Catalog And Recipe Proposals**
Выполнена.

**Итерация 4: Cloud Sync Contracts, ClickHouse Trail, Inventory Analyzer**

Выполнена.

**Итерация 5: Cloud Proposal Review And Feedback**
Выполнена.

**Итерация 6: pos-ui-g Kitchen Mode**
Выполнена.

**Итерация 7: Cloud UI Manager Review**
Выполнена.

**Итерация 8: End-To-End Smoke And Documentation Alignment**
В работе. Кодовый blocker `seed_full_system(..., run_kitchen_process_smoke=...)` в текущей ветке не воспроизводится: сигнатура принимает параметр, summary пишет отдельные `minimal_flow` и `kitchen_process_smoke`, а Python регрессионные тесты проходят. Полный Docker smoke не подтвержден в текущем окружении: `docker compose -f docker-compose.local.yml up --build -d` завис после предупреждения `Docker Compose requires buildx plugin to be installed`, резервный запуск `docker compose -f docker-compose.local.yml up -d` остановился на `Bind for 127.0.0.1:5432 failed: port is already allocated`.
