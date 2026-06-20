# Итерации кодогенерации выставочного запуска

Статус: готово к последовательной реализации задач первого запуска.

Дата проверки: 2026-06-20.

Этот документ является runbook для непосредственной генерации кода. Он не копирует Plane: перед каждой итерацией агент обязан заново прочитать work item, relations и комментарии. Plane хранит текущее состояние работы, Git — требования, код и тесты.

## Вердикт

Кодогенерацию можно начинать с первой незаблокированной задачи в состоянии `Ready`.

На момент проверки первый безопасный порядок:

1. `POS-61` — tenant roles/employees и restaurant memberships.
2. `POS-62` — tenant catalog и restaurant menu overrides.
3. `POS-42` — внешний licensing authority и module gates.
4. `POS-65` — автоматическая Cloud -> Edge доставка.

Весь launch cycle еще не готов к непрерывному выполнению: `POS-38`, `POS-41`, `POS-43`, `POS-44`, `POS-45`, `POS-64` остаются `Specified` до закрытия зависимостей или внешних решений. Агент не начинает такую задачу и не переводит ее в `Ready` предположением.

## Результат аудита Plane

Проверено 65 work items, 3 cycles, 23 modules, 18 labels и state workflow проекта `POS`.

- launch cycle содержит 18 задач;
- post-deploy QR cycle содержит 8 задач и не смешан с первым запуском;
- `POS-48`, `POS-52`, `POS-53` отвязаны от post-deploy parent `POS-39`;
- launch dependencies заведены как `blocked_by` relations;
- `POS-61…65` привязаны к профильным modules;
- Ready-задачи launch foundation имеют label `agent-ready`;
- `POS-36` переведена в `Review` с итоговым комментарием, `Done` не выставлен;
- отсутствие assignee не блокирует агентский workflow: исполнитель определяется при запуске задачи;
- legacy bootstrap items `POS-6`, `POS-8`, `POS-9` не имеют labels, а `POS-6…9` имеют неполные descriptions. Они не входят в launch cycle и не блокируют кодогенерацию, но требуют отдельной backlog hygiene, если будут возвращены в работу.

Текущий Plane MCP записывает relations, но его read-модель ожидает массив UUID, тогда как Plane server возвращает relation objects. `list_work_item_relations` поэтому может завершиться schema validation error. Gap записан в backlog `POS-66`. До исправления connector агент использует таблицу ниже, перечитывает состояния указанных задач по identifier и отмечает ограничение в стартовом комментарии.

## Граф зависимостей

| Задача | `blocked_by` |
| --- | --- |
| `POS-61` | — |
| `POS-62` | — |
| `POS-42` | — |
| `POS-65` | `POS-61`, `POS-62`, `POS-42` |
| `POS-52` | `POS-62` |
| `POS-53` | `POS-42` |
| `POS-48` | `POS-52`, `POS-53` |
| `POS-64` | `POS-48` |
| `POS-40` | `POS-48`, `POS-65` |
| `POS-41` | `POS-40` |
| `POS-63` | `POS-40`, `POS-42` |
| `POS-38` | `POS-61`, `POS-62`, `POS-65`, `POS-42`, `POS-52`, `POS-53`, `POS-48` |
| `POS-43` | `POS-42`, `POS-63`, `POS-64`, `POS-65` |
| `POS-44` | `POS-43` |
| `POS-45` | `POS-38`, `POS-41`, `POS-63`, `POS-64`, `POS-44` |
| `POS-47` | `POS-45` |
| `POS-46` | `POS-47` |

## Проверенный baseline

Перед созданием runbook успешно выполнены:

- `cd pos-backend && go test ./...`;
- `cd cloud-backend && go test ./...`;
- `cd license-server && go test ./...`;
- `cd pos-ui-g && npm run build`;
- `cd cloud-ui-g && npm run build`.

`pos-ui-g` собирается с предупреждением Vite о chunk больше 500 KB. Это не блокирует начало launch-разработки и не должно исправляться попутно без отдельной задачи.

## Источники истины

Каждая итерация обязана читать:

- `AGENTS.md`;
- соответствующий Plane work item, relations и комментарии;
- `docs/project-management/EXHIBITION-ALPHA-PILOT-REQUIREMENTS.md`;
- `docs/project-management/MVP-PILOT-REQUIREMENTS.md`;
- `ROADMAP.md`;
- `docs/CURRENT-FUNCTIONAL-STATE.md`;
- профильные документы из `docs/backend`, `docs/ui`, `docs/sync`, `docs/architecture`;
- фактический runtime-код и тесты затрагиваемых модулей.

Если текст задачи расходится с кодом, агент сначала фиксирует фактическое поведение и обновляет task description или оставляет Plane comment. Будущее поведение не объявляется реализованным.

## Обязательный Plane workflow

### Начало

Агент:

1. Получает задачу по `POS-N` через Plane MCP.
2. Проверяет cycle, module, state, labels, `blocked_by`, parent, description и последние comments.
3. Начинает код только если задача `Ready` и все `blocked_by` из Plane или fallback-таблицы закрыты после проверки фактических состояний зависимостей.
4. Проверяет `git status` и не откатывает чужие изменения.
5. Переводит задачу в `In Progress`.
6. Оставляет комментарий:

```text
Работа начата.

Проверено:
- требования и текущее состояние кода;
- зависимости Plane;
- dirty state рабочего дерева.

План:
- ...

Ожидаемые проверки:
- ...
```

### Во время работы

Комментарий о прогрессе обязателен, если:

- найдено расхождение с task description;
- меняется API/schema/sync/RBAC contract;
- появилась внешняя блокировка;
- работа переносится на следующий прогон.

```text
Промежуточный прогресс:
- выполнено: ...
- подтверждено тестами: ...
- осталось: ...
- риск или блокер: ...
```

При реальном внешнем блокере агент оставляет точное условие разблокировки и переводит задачу в `Blocked`. Наличие обычной зависимости, уже записанной в `blocked_by`, не требует начинать задачу или менять ее state.

### Завершение агентом

Агент не выставляет `Done`. После реализации и доступных проверок он оставляет итоговый комментарий и переводит задачу в `Review`:

```text
Реализация завершена, готово к проверке человеком.

Выполнено:
- ...

Измененные файлы:
- ...

Проверки:
- команда: результат

Не запускалось:
- проверка: причина

Документация:
- ...

Оставшиеся риски:
- ...

Вне scope:
- ...

Runtime code:
- да/нет; затронутые области
```

Если обязательная автоматическая проверка падает из-за внесенного изменения, задача остается `In Progress`. Если автоматические проверки прошли, но требуется принтер, restore rehearsal или другая ручная приемка, задача переходит в `Review` с label `manual-validation` и точным сценарием проверки.

Человек после review переводит задачу в `Done` либо возвращает в `In Progress` с комментарием. Агент не выполняет commit, push, merge, release или deployment без отдельного запроса.

## Универсальный промпт

```text
Реализуй Plane work item <POS-N> в репозитории /home/master/repos/myhoreca-pos.

Сначала через Plane MCP прочитай work item, state, module, cycle, labels, parent, relations и comments. Начинай код только если state = Ready и фактические blocked_by закрыты. Переведи задачу в In Progress и оставь стартовый комментарий с планом и проверками.

Прочитай AGENTS.md, docs/project-management/ALPHA-LAUNCH-CODEGEN-ITERATIONS.md, EXHIBITION-ALPHA-PILOT-REQUIREMENTS.md, MVP-PILOT-REQUIREMENTS.md, ROADMAP.md, CURRENT-FUNCTIONAL-STATE.md и профильные backend/UI/sync/architecture документы. Затем прочитай фактический код и тесты. Не расширяй scope и не откатывай чужие изменения.

Используй текущий managed baseline 001_init.sql и startup migration policy. UUID — только UUIDv7. Backend является authoritative для RBAC, licenses, financial state и sync. Пользовательские UI-строки идут через i18n. При изменении route, payload, schema, RBAC, sync, errors, logs, startup или smoke обнови профильные документы в том же изменении.

Реализуй минимальный законченный diff, добавь необходимые тесты и запусти профильные проверки. После реализации оставь итоговый Plane comment по шаблону runbook и переведи задачу в Review. Не выставляй Done.
```

## Wave 1. Tenant foundation

Эти итерации выполняются последовательно: `POS-61` и `POS-62` меняют один Cloud baseline и master-data contracts.

### Итерация 1 — `POS-61`

```text
Используй универсальный промпт для POS-61.

Фокус: перенести roles/employees на tenant scope, добавить employee restaurant memberships и permission organization.manage. Не ослаблять PIN/session security. Cloud API, publication filtering, Edge staff read model и UI должны использовать одну authoritative membership model.

Минимальная приемка: organization manager видит все restaurants; остальные сотрудники имеют минимум одно membership; revoke membership блокирует restaurant login/commands; Edge получает только eligible staff. Обязательны migration backup/schema verification, backend tests, Cloud UI build и профильные RBAC/sync docs.
```

### Итерация 2 — `POS-62`

Статус выполнения на 2026-06-20: реализовано сейчас в коде, тестах и профильных документах; Plane переводится в `Review` после локальных проверок.

```text
Используй универсальный промпт для POS-62.

Фокус: tenant catalog identity и restaurant menu overrides для name, price, tag, active tax, menu folder, availability и runtime status. Сохранять catalog_item_id и menu_item_id в downstream contracts; ticket category должна иметь стабильную identity, а не вычисляться из текста.

Минимальная приемка: один catalog item используется двумя restaurants с разными menu values; изменение одного menu не меняет другое; Edge получает только restaurant-effective menu. Обязательны schema/API/repository/sync/UI/seed tests и документы.
```

### Итерация 3 — `POS-42`

```text
Используй универсальный промпт для POS-42.

Фокус: внешний licensing authority и расширяемые entitlements table-mode, telegram-worker, kitchen-space, waiter-space, checker-flow, warehouse-mode. UI hiding не заменяет Cloud/Edge backend enforcement. Stale grace задается только deployment config поставщика.

Минимальная приемка: enable/disable/revoke/stale-expiry проверены; нелицензированные routes, commands, workers и streams возвращают stable safe error; данные при отключении не удаляются. Обязательны security tests для License Server, Cloud, Edge и UI build.
```

## Wave 2. Delivery и sale model

### Итерация 4 — `POS-65`

```text
Используй универсальный промпт для POS-65 после закрытия POS-61, POS-62 и POS-42.

Фокус: удалить manual Publish action. До Edge assignment не создавать delivery packages. При assignment/first connection собрать current full batch; после effective Cloud commit обновлять latest package только для назначенных Edge. Scheduled sync/exchange доставляет version новее node checkpoint.

Не создавать event backlog на каждое изменение: текущего latest row на node/stream достаточно. Draft/review и нелицензированные rows не доставлять. Заменить Publications UI на read-only Cloud version/Edge ACK/lag/error. Обновить canonical seed/smoke так, чтобы он не вызывал publish API.
```

### Итерация 5 — `POS-52`

```text
Используй универсальный промпт для POS-52 после POS-62.

Фокус: qr_confirmation_enabled, зависимый single_unit_per_line и validity modes cash_session, business_date, absolute_date. Backend автоматически включает single-unit rule и запрещает quantity > 1; повторное добавление создает новую line.

Не реализовывать checker lookup/confirm. Обновить tenant catalog, restaurant menu publication, Edge validation, UI/i18n, migration и tests.
```

### Итерация 6 — `POS-53`

```text
Используй универсальный промпт для POS-53 после POS-42.

Фокус: при выключенном table-mode скрыть halls/tables/precheck UI и Cloud settings, но сохранить общую Order/Precheck/Payment model через system restaurant hall/table и automatic authoritative precheck перед payment.

Не открывать hidden precheck actions через API. Проверить license on/off, restaurant isolation, counter checkout и обычный table-mode regression.
```

### Итерация 7 — `POS-48`

```text
Используй универсальный промпт для POS-48 после POS-52 и POS-53.

Фокус: транзакционно создать одну ticket unit после final check для каждой QR-enabled line. UUIDv7, unique ticket number, cash-shift sequence, immutable restaurant/menu name, sale date/timezone/validity и безопасный QR payload.

Replay не создает второй ticket. Reprint использует тот же QR и COPY marker. Не реализовывать lookup/use/revoke. Проверить financial transaction boundary, refund-safe behavior, migration, event/sync contracts и tests.
```

## Wave 3. Output и reporting

### Итерация 8 — `POS-64`

```text
Запускай универсальный промпт для POS-64 только после перевода задачи в Ready и закрытия POS-48.

Фокус: общий нефискальный ESC/POS subsystem для check и ticket, network TCP и Windows USB printer, typed versioned templates, queue, timeout, bounded retry, status и audit. Print retry не повторяет payment и не создает ticket.

До Review обязательны unit/integration tests. В итоговом Plane comment оставить точный manual hardware checklist и модели проверенных принтеров. Без реального принтера не заявлять hardware acceptance.
```

### Итерация 9 — `POS-40`

```text
Используй универсальный промпт для POS-40 после POS-48 и POS-65.

Фокус: bounded operational projection/API для sold ticket units, gross/refund/net, average check, restaurant, stable ticket category, catalog/menu item, business date и cash shift. Доступ только по organization.manage или memberships.

Добавить deterministic paging, freshness/incomplete marker и drill-down до check/order line/ticket/financial operation. Не включать checker usage analytics.
```

### Итерация 10 — `POS-41`

```text
Запускай универсальный промпт для POS-41 после POS-40 и перевода задачи в Ready.

Фокус: заменить dashboard placeholders реальными API data. Реализовать authorized restaurant filter, ticket category/service/date/shift filters, sold/gross/refund/net/average check, freshness, loading/empty/error states и bounded drill-down.

Backend RBAC authoritative. Все строки через i18n. Обязательны component tests, npm build и Playwright для основного dashboard flow.
```

### Итерация 11 — `POS-63`

```text
Используй универсальный промпт для POS-63 после POS-40 и POS-42.

Фокус: restaurant Telegram settings, schedule trigger и cash-shift-close trigger, confirmed chat_id onboarding, idempotent report occurrence, retry/backoff и safe secrets. Username хранится только как display metadata.

Без telegram-worker UI/routes/worker недоступны. Отчет использует POS-40 projection и содержит freshness marker. Для Telegram API сначала получи актуальную документацию через Context7 согласно AGENTS.md.
```

## Wave 4. Seed и эксплуатация

### Итерация 12 — `POS-38`

```text
Запускай универсальный промпт для POS-38 после закрытия его Plane dependencies и перевода в Ready.

Фокус: HTTP-only seed двух restaurants с tenant roles/employees/catalog, memberships, разными menu overrides, licenses, QR-enabled services, Edge assignment и automatic sync. Не вызывать publish API и не писать напрямую в БД.

Скрипт должен fail-fast на грязном окружении, не сохранять secrets в tracked output и подготовить данные для launch smoke без checker dependencies.
```

### Итерация 13 — `POS-43`

```text
Запускай универсальный промпт для POS-43 только после Ready и закрытия POS-42, POS-63, POS-64, POS-65.

Фокус: single-host runbook для Cloud, Edge, PostgreSQL, ClickHouse, License Server, print worker и Telegram worker. Зафиксировать TLS, secrets, health/readiness, logs, rollback, printer connectivity и будущую переносимость в Kubernetes без изменения доменной модели.

Не выполнять production deployment. Неподтвержденные DNS/registry/RPO/RTO оставить явными blockers, а не придумывать.
```

### Итерация 14 — `POS-44`

```text
Запускай универсальный промпт для POS-44 после POS-43 и перевода задачи в Ready.

Фокус: backup/restore PostgreSQL, SQLite, entitlement snapshot, Telegram settings и printer/template config. Зафиксировать RPO/RTO, retention, owner и безопасный restore order.

До Review выполнить доступный disposable restore rehearsal. Невыполненную production/manual часть перечислить в Plane comment и оставить manual-validation.
```

## Wave 5. Сквозная приемка

### Итерация 15 — `POS-45`

```text
Запускай универсальный промпт для POS-45 только после закрытия всех blocked_by и перевода в Ready.

Реализуй один canonical E2E smoke: tenant setup → licenses/memberships/menu → Edge assignment/full batch → automatic change delivery → ticket sale → physical print job → Cloud dashboard → Telegram report → backup signal.

Финансовые mutations single-shot; print/report retries идемпотентны. Автоматическую часть выполнить полностью, hardware/restore evidence оформить отдельным manual checklist. Checker flow не добавлять.
```

### Итерация 16 — `POS-47`

```text
Используй универсальный промпт для POS-47 после POS-45.

Фокус: собрать evidence по каждому Go/No-Go пункту из exhibition requirements. Не заменять тест ссылкой на код и не считать mock printer hardware acceptance.

Если автоматические проверки пройдены, перевести в Review и приложить human checklist. Окончательный Go/No-Go и Done выставляет только владелец продукта.
```

### Итерация 17 — `POS-46`

```text
Используй универсальный промпт для POS-46 после POS-47.

Фокус: финально синхронизировать Git requirements, ROADMAP, CURRENT-FUNCTIONAL-STATE, профильные contracts, Plane work items и public page с фактическим runtime. Удалить claims о manual Publish, если POS-65 реализован, и не объявлять post-deploy checker готовым.

Runtime code не менять. Выполнить профильные rg, link check, git diff --check и оставить итоговый Plane comment со списком source-of-truth документов.
```

## Post-deploy boundary

`POS-37`, `POS-39`, `POS-49`, `POS-50`, `POS-51`, `POS-54`, `POS-55`, `POS-56` не выполняются до завершения launch Go/No-Go. Агент не переносит их в launch cycle и не добавляет checker code попутно в ticket issuance, printing или waiter tasks.

## Готовность итерации

Итерация готова к запуску, когда одновременно выполнено:

- task state `Ready`;
- все `blocked_by` закрыты человеком в `Done` либо явно приняты владельцем как завершенные;
- description содержит проверяемые acceptance criteria;
- отсутствует неразрешенный hardware/product decision;
- рабочее дерево проверено;
- агент понимает минимальный набор тестов и документов.

Если хотя бы одно условие не выполнено, агент не генерирует код: оставляет Plane comment с точным gap и завершает прогон.
