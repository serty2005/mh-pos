# Итерации кодогенерации выставочного запуска

Статус: Wave 1–2 закрыты; Wave 3 receipt-printing (P1-декомпозиция `POS-64`) в приемочной проверке. `POS-68/69/70/71/72/73/74/81/82/83/84` = `Done`, `POS-67/80` = `Review`. Clean-stack аудит 2026-06-28 нашёл и исправил POS-84 blocker: Cloud baseline/exchange не принимал stream `printers`; Cloud runtime поднят до `0.1.15`. Физический TCP ESC/POS стенд: `10.25.1.201:9100`, CPL=48. Backend/worker отправил `precheck` и `ticket` успешно; оператор подтвердил выход двух чеков, после чего ESC/POS/SVG renderer исправлен для крупного `{f:double}` текста через эффективную ширину `CPL/2`. Зонтичная `POS-64` остаётся открытой до закрытия POS-67/POS-80 и повторного операторского подтверждения ticket. ESC/POS по умолчанию CP437 (Indonesia/English launch). Добавлены `POS-82/83/84` — Cloud-managed printer config (CRUD + mastersync + Edge consumer), заменяет `legacy deployment printer routing` env var; итерации 8c/8d/8e. Зафиксирован следующий gap после Wave3: полноценная ресторанная схема печати с точками продаж, секциями, Edge override и `print_job_targets` не реализована сейчас и требует отдельной декомпозиции `POS-85…POS-89`. Обновлено 2026-06-28. См. «Итерация 8».

Дата проверки: 2026-06-28.

Этот документ является runbook для непосредственной генерации кода. Он не копирует Plane: перед каждой итерацией агент обязан заново прочитать work item, relations и комментарии. Plane хранит текущее состояние работы, Git — требования, код и тесты.

## Срез License Server deployment/CD

Обновлено 2026-06-30.

Реализовано сейчас:

- License Server переведен с admin token на super-admin login/password из стартового конфига; password хранится hash+salt.
- Operator UI выбирает connected server из списка, поддерживает поиск по `tenant_id`, toggles/presets и advanced JSON.
- Подготовлен native Linux deployment для Ubuntu 24.04 без Docker: systemd unit, env example, production JSON example и runbook `docs/deployment/LICENSE-SERVER-LINUX.md`.
- Подготовлен GitHub Actions CD template `deploy/license-server/deploy-license-server.workflow.yml`; активная папка `.github` временно не хранится в ветке до выдачи GitHub permission на workflow files. После копирования template в `.github/workflows/deploy-license-server.yml` push в ветку `production` при изменениях `license-server/**`, `shared/platform/**` или deploy artifacts запускает tests, build linux-amd64 binary, SSH deploy, symlink switch и restart `license-api`.
- До появления домена runtime URL задается как `http://<server-ip>:8095`; firewall закрывает входящие соединения кроме SSH и порта License Server для доверенного IP/VPN.

Запланировано далее:

- создать ветку `production`, GitHub secrets и SSH deploy user на конечной VM;
- выполнить первый ручной install, smoke и затем первый CD deploy;
- настроить регулярный внешний backup и restore rehearsal.

Вне текущего объема:

- домен/TLS/reverse proxy;
- billing provider;
- полноценный commercial admin с тарифами, договорами и ролями операторов.

## Срез POS Edge Windows + webwallpaper

Обновлено 2026-06-30.

Реализовано сейчас:

- NSIS installer с `-WebWallpaperExe` создает active `webwallpaper/config.pos-edge.json` на install-time с URL `http://127.0.0.1:<POS backend port>/`.
- Installer создает shortcut `MyHoreca\POS Edge Display`, который запускает `gowebwallpaper.exe` с этим config path.
- `gowebwallpaper` принимает config path первым аргументом командной строки; installer-generated config поэтому подавляет URL prompt на первом запуске.
- Если config не содержит выбранный монитор, `gowebwallpaper` выбирает primary screen из подключенных мониторов и сохраняет этот выбор перед autostart.

Запланировано далее:

- собрать новый Windows installer с актуальным `gowebwallpaper.exe` и проверить на реальной Windows POS-станции: install, первый запуск `POS Edge`, первый запуск `POS Edge Display`, primary-screen fullscreen и повторный запуск без prompt.

Вне текущего объема:

- автозапуск POS Edge и kiosk host после установки;
- Windows service/MSI/updater flow;
- упаковка WebView2 runtime внутрь POS Edge installer.

## Итог аудита Wave 3 / POS-64

Статус: `POS-64` не закрыт. Кодовая база доведена до корректной основы для продолжения `POS-85…POS-89`, но Plane и ручная приемка ещё не совпадают полностью: `POS-67` и `POS-80` остаются на человеческой проверке, а ticket после исправления эффективной ширины для `{f:double}` требует повторного physical-print подтверждения на `10.25.1.201:9100`.

Реализовано сейчас:

- Cloud-owned physical printer routing: Cloud CRUD `/api/v1/printers`, stream `printers`, Edge ingest в `receipt_printers`, worker routing по `restaurant_id + document_type`.
- Print queue `print_jobs` на Edge для `precheck`, `ticket`, `check_nonfiscal`; automatic enqueue после оплаты сейчас создаёт `precheck` и `ticket` для issued ticket units, `check_nonfiscal` поддержан как тип job/context, но не создаётся автоматически после оплаты.
- Template engine/default templates, SVG preview и Cloud UI template editor реализованы для ReceiptLine Level 1; ESC/POS renderer поддерживает TCP/USB, CP437 по умолчанию, CP866 явно, native QR и bounded failure path.
- `POS_PRINTER_ROUTING_JSON` больше не задаёт routing: POS Edge пишет `LEGACY_PRINTER_ROUTING_IGNORED` и игнорирует значение.

Не реализовано сейчас и не входит в закрытие Wave3:

- `sales_points`, cash printer per sales point и cash session внутри sales point;
- `restaurant_sections`, service printer per section и report printer per restaurant;
- Edge-side immediate override, audit override changes и Cloud projection override state;
- `print_job_targets` и отдельный retry/status/error на каждый физический принтер.

Состояние задач `POS-67…POS-84` на 2026-06-28:

| Задача | Фактический статус | Остаток |
| --- | --- | --- |
| `POS-67` ADR/spec receipt print | частично закрыто процессно | ADR/spec есть в Git; Plane state `Review`, нужно человеческое подтверждение. |
| `POS-68` Receipt IR/parser | реализовано сейчас | Остатков по Wave3 не найдено. |
| `POS-69` ESC/POS renderer | реализовано сейчас | Printer-specific codepage/cut-feed остаётся риском hardware acceptance. |
| `POS-70` SVG preview | реализовано сейчас | Остатков по Wave3 не найдено. |
| `POS-71` receipt_template Cloud master-data + Edge ingest | реализовано сейчас | Остатков по Wave3 не найдено. |
| `POS-72` print context projection | реализовано сейчас | `check_nonfiscal` не auto-enqueue после оплаты. |
| `POS-73` template engine/default templates | реализовано сейчас | ESC/POS/SVG renderer исправлен: крупный `{f:double}` текст использует эффективную ширину `CPL/2`, default ticket template снова печатает крупные `TICKET` и `service_name`. Level 2 templates вне текущего объема. |
| `POS-74` Edge print queue + routing + worker | реализовано сейчас | Per-printer targets запланированы далее. |
| `POS-75…POS-79` P2/следующие задачи | вне текущего объема | Не смешивать с Wave3. |
| `POS-80` Cloud UI template editor | частично закрыто процессно | Код и build пройдены; остаётся человеческая UI-проверка. |
| `POS-81` seed/smoke | реализовано сейчас | Clean-stack smoke прошёл после исправления stream `printers`. |
| `POS-82` Cloud printer CRUD | реализовано сейчас | Остатков по Wave3 не найдено. |
| `POS-83` Cloud UI printer management | реализовано сейчас | Остатков по Wave3 не найдено. |
| `POS-84` Edge printer config from Cloud sync | реализовано сейчас после audit fix | Был найден clean-stack blocker в Cloud constraints/exchange для stream `printers`; исправлено в `0.1.15` и подтверждено smoke. |

## Вердикт

Кодогенерацию можно продолжать с `POS-65`: зависимости `POS-61`, `POS-62` и `POS-42` проверены человеком и переведены в `Done`.

На момент проверки первый безопасный порядок:

1. `POS-61` — tenant roles/employees и restaurant memberships.
2. `POS-62` — tenant catalog и restaurant menu overrides.
3. `POS-42` — внешний licensing authority и module gates.
4. `POS-65` — автоматическая Cloud -> Edge доставка.

Весь launch cycle еще не готов к непрерывному выполнению: `POS-38`, `POS-41`, `POS-43`, `POS-44`, `POS-45`, `POS-64` остаются `Specified` до закрытия зависимостей или внешних решений. Агент не начинает такую задачу и не переводит ее в `Ready` предположением.

## Срез после итерации 3

Проверено 2026-06-21 после коммита `e0e8357` и последующих документальных уточнений лицензируемых границ.

- `POS-61`, `POS-62` и `POS-42` находятся в `Done` в Plane после ручной проверки.
- POS-42 реализован сейчас в runtime code, security tests, локальном Docker smoke и профильной документации.
- POS-42 Plane item прочитан: state `Review`, labels `agent-ready`, `tests`, `security`, `frontend`, `backend`; итоговый комментарий фиксирует License Server, fail-closed Cloud/Edge gates, stale grace, revoke/expiry, backup, UI visibility, sync filtering и worker enforcement.
- `list_work_item_relations` для POS-42 все еще падает на MCP schema validation: Plane server возвращает relation objects, connector ожидает строки UUID. Ограничение остается в `POS-66`; для текущего review использована fallback-таблица зависимостей ниже.

### Docker POS Edge

Отдельный `pos_edge_sqlite_data` volume в `docker-compose.igor.yml` является только локальным Docker/smoke harness для POS Edge. Он сохраняет `/app/data` между restart/rebuild контейнера: `pos-edge.db`, backups и archives. Это нужно для проверки POS-42 acceptance `данные при отключении не удаляются`, revoke/rebuild сценариев и воспроизводимого full-stack smoke с Cloud, License Server, PostgreSQL и ClickHouse.

Это не целевой deployment contract для клиента. Целевой POS Edge остается исполняемым файлом в Windows-окружении POS-станции с локальными путями данных, backup и archive из deployment config. Для Windows-specific поведения, путей, service wrapper и локального SQLite надежнее дополнительно запускать `pos-edge` локально вне Docker; Docker-окружение пока оставляется как интеграционный стенд, а не замена локальной проверки.

### POS-42 Закрепленные Решения

Перед стартом POS-65 приняты следующие границы:

Актуальный canonical contract по module IDs, product bundles и enforcement перенесен в `docs/backend/LICENSE-ENTITLEMENTS.md`; пункты ниже оставлены как исторический контекст POS-42.

1. Первый review finding по `waiter-space` уточнен: основной cashier flow является нелицензируемой базовой частью POS Edge и должен оставаться доступным всегда при наличии локальных данных. Нельзя закрывать `waiter-space` все `/api/v1/orders...`, `/api/v1/menu/items`, precheck/payment/check routes целиком, потому что это shared cashier backend surface. `waiter-space` должен применяться к отдельному waiter-доступу: mobile waiter route/API facade, waiter-only commands/events или другому backend-owned признаку официантского контекста. UI route `/pos/waiter` или frontend header не является security boundary.
2. Post-MVP целевое решение: полностью бесплатный автономный POS Edge без внешнего Cloud. Edge локально хранит данные в SQLite, локальный владелец создает простые позиции меню для собственного Edge, кассир продает через базовый `Order -> Precheck -> Payment -> Check`, backup/archive остаются локальными. Покупка `cloud-subscription` подключает внешний Cloud, tenant management, автоматическую доставку master data и Cloud analytics; waiter mobile, KDS/advanced kitchen, warehouse engine, Telegram, ticket/checker и будущие модули подключаются отдельными entitlement IDs. Это тот же runtime/profile, а не fork продукта.
3. Cloud generic provisioning package routes `/api/v1/provisioning/master-data/{stream}` пока не сопоставлены с module entitlements в `cloudModuleForRequest`. Sync exchange фильтрует disabled `floor`, `recipes` и `inventory` streams, но прямой package GET/PUT может читать или записывать package для выключенного модуля. Это важно перед POS-65, потому что следующая итерация меняет доставку Cloud -> Edge.
4. Edge -> Cloud data boundary нужно довести до общего правила: Cloud-side не должен формировать данные из выключенных workers или заблокированных routes, а Edge batch должен содержать только module-owned события включенных лицензий. Базовые cashier financial facts остаются синхронизируемым ядром подключенного Cloud-контура. `kitchen-space` владеет KDS/kitchen events и proposals; `warehouse-mode` владеет receipt/count/write-off/production и Cloud inventory worker; будущий `waiter-space` владеет waiter-only commands/events после выделения backend-discriminated waiter surface.
5. Дополнительно на Edge-стороне нужно закрыть работу с module-owned данными по entitlement: выключенный модуль не открывает новые commands/UI-backed routes и не добавляет новые module-owned outbox rows. Уже сохраненные данные не удаляются.
6. Browser Playwright по POS-42 не выполнен из-за закрытия transport окружения. UI покрыт component/helper tests и production builds; браузерная проверка не блокирует старт POS-65 после ручной проверки первых трех итераций.
7. npm audit предупреждения остаются внешним риском текущего frontend dependency state: Cloud UI — 1 high; POS UI — 2 low, 1 moderate, 1 high. Они не блокируют старт POS-65.

### Проверки POS-42

По итоговому Plane comment и последнему коммиту пройдены:

- `cd license-server && go mod tidy && go test ./...`;
- `cd cloud-backend && go mod tidy && go test ./...`;
- `cd pos-backend && go mod tidy && go test ./...`;
- `cd shared/platform && go test ./...`;
- Python seed tests: 16/16;
- POS UI: lint, tests 65/65, build;
- Cloud UI: lint, tests 46/46, build;
- полный seed, minimal flow и kitchen smoke;
- Docker rebuild, health endpoints, `git diff --check`.

После этого review runtime code не изменялся. Изменения плана фиксируют текущее состояние и разрешают старт POS-65.

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
| `POS-82` | — (Cloud printer CRUD + mastersync stream) |
| `POS-83` | `POS-82` (Cloud UI printer management) |
| `POS-84` | `POS-82`, `POS-74` (POS Edge printer от sync) |
| `POS-40` | `POS-48`, `POS-65` |
| `POS-41` | `POS-40` |
| `POS-63` | `POS-40`, `POS-42` |
| `POS-38` | `POS-61`, `POS-62`, `POS-65`, `POS-42`, `POS-52`, `POS-53`, `POS-48` |
| `POS-43` | `POS-42`, `POS-63`, `POS-64`, `POS-65`, `POS-84` |
| `POS-44` | `POS-43` |
| `POS-45` | `POS-38`, `POS-41`, `POS-63`, `POS-64`, `POS-44`, `POS-84` |
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

Статус выполнения на 2026-06-20: реализовано сейчас в runtime, security tests, локальном Docker smoke и профильной документации; Plane переводится в `Review` после итоговых проверок.

```text
Используй универсальный промпт для POS-42.

Фокус: внешний licensing authority и расширяемые entitlements из `docs/backend/LICENSE-ENTITLEMENTS.md`: cloud-subscription, table-mode, telegram-worker, kitchen-space, waiter-space, ticket-mode, warehouse-mode. UI hiding не заменяет Cloud/Edge backend enforcement. Stale grace задается только deployment config поставщика.

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

### Итерация 8 — `POS-64` (зонтичная) → P1 receipt-printing decomposition

`POS-64` «Физическая ESC-POS печать чеков и билетов» — зонтичная задача. Фактическая работа выполняется P1-итерациями `POS-67…POS-74`, `POS-80`, `POS-81` (ADR-017 + RECEIPT-PRINT-SPEC.md). Прежний монолитный промпт под `POS-64` устарел и заменён этой декомпозицией. Hardware стенд 2026-06-28: реальный ESC/POS TCP-принтер `10.25.1.201:9100`, модель неизвестна, CPL=48; backend/worker успешно отправил документы, оператор подтвердил выход бумаги, но ticket потребовал исправления эффективной ширины для крупного текста. `POS-64` остаётся открытой до закрытия POS-67/POS-80 и повторного подтверждения ticket.

Снимок статуса на 2026-06-28 (перечитывать Plane перед каждой итерацией):

- `POS-67` P1-1 (ADR-017 + spec) — фактически выполнено в Git; Plane state `Specified`.
- `POS-68` P1-2 (IR + ReceiptLine parser) — `Done`.
- `POS-69` P1-3 (IR → ESC/POS, CP437/CP866, TCP/USB) — `Done`; hardware acceptance зафиксирован: TCP `10.25.1.201:9100`, модель неизвестна, CPL=48. **ESC/POS по умолчанию CP437** (`ESC t 0`, USA/Standard Europe); CP866 через `"codepage": "cp866"` в PrinterConfig.
- `POS-70` P1-4 (IR → SVG + Cloud preview) — `Done`. POST /receipts/preview добавлен (POS-80).
- `POS-72` P1-6 (print context projection) — `Done`.
- `POS-71` P1-5 (receipt_template Cloud master-data + Edge stream) — `Done`.
- `POS-73` P1-7 (template engine + default-шаблоны) — `Done`. Default шаблоны переведены на English (Indonesia launch).
- `POS-74` P1-8 (Edge print queue + routing + worker + endpoints) — `Done`.
- `POS-80` P1-9 (Cloud UI template editor + live preview) — `Review`. Реализован: route `/receipt-templates` в Cloud Manager, двухпанельный интерфейс (список + редактор), live SVG preview через POST /receipts/preview (debounced 600ms), CRUD шаблонов. Ожидает человеческого review.
- `POS-81` P1-10 (seed-dev-system.py + receipt_templates smoke) — `Done`. Clean-stack smoke 2026-06-28 прошёл после исправления Cloud stream `printers` в baseline/exchange.

Рекомендованный порядок остатка: закрыть `POS-67` после проверки ADR/spec, закрыть `POS-80` после человеческой UI-проверки, обновить Cloud ticket template из Git и получить повторное операторское подтверждение ticket на `10.25.1.201:9100` → закрытие зонтичной `POS-64`.

#### Итерация 8a — `POS-73` (P1-7)

```text
Используй универсальный промпт для POS-73.

Начинай код только если POS-73 = Ready и зависимости закрыты: POS-68 и POS-72 = Done; POS-71 = Review или Done и её код фактически в рабочем дереве/смержен (receipt_templates таблицы и stream). Если условие не выполнено — оставь Plane comment с точным gap и остановись.

Фокус: template engine engine.Render поверх POS-68 IR/parser и POS-72 PrintContext — детерминированный рендер ReceiptLine Level 1 в IR для document_type precheck и ticket; default-шаблоны precheck и ticket и их идемпотентное сидирование в Cloud receipt_templates через seed-dev-system.py (HTTP-only, без прямой записи в БД и без publish API). Сиды используют CRUD /api/v1/receipt-templates из POS-71 и доставляются на Edge существующим stream receipt_templates.

Не включать: POS-74 print queue/routing/worker/HTTP endpoints; изменение ESC/POS (POS-69) или SVG (POS-70) рендеров сверх необходимого вызова engine.Render; Cloud UI editor (POS-80); fiscal logic, PNG/e-receipt, Level 2 (go text/template) шаблоны; QR-size/cut-feed follow-up как отдельную фичу (учитывать требования из POS-69 комментария только на уровне default-шаблонов, без новой подсистемы).

Соблюдай AGENTS migration policy и UUIDv7. Обязательны unit-тесты engine.Render (precheck/ticket фикстуры, CPL 32/48), тесты default-шаблонов и seed smoke (receipt_templates присутствуют после сидирования и доходят до Edge). Документация в том же изменении: RECEIPT-PRINT-SPEC.md (engine + default templates статус), CURRENT-FUNCTIONAL-STATE.md, при необходимости POS-BACKEND-SPEC/seed docs.

Проверки: cd cloud-backend && go mod tidy && go test ./...; cd pos-backend && go mod tidy && go test ./...; shared/platform go test ./...; seed/smoke по сидированию шаблонов; git diff --check. Оставь итоговый Plane comment по runbook, переведи POS-73 в Review, не Done. Commit/push/deploy без отдельного запроса не делай.
```

#### Итерация 8b — `POS-74` (P1-8)

```text
Используй универсальный промпт для POS-74 после POS-73.

Начинай код только если POS-74 = Ready и зависимости закрыты: POS-69 (ESC/POS renderer) и POS-72 (print context) = Done; POS-73 = Review или Done и её код в рабочем дереве (engine.Render + default-шаблоны). Иначе — Plane comment с gap и стоп.

Фокус: Edge print queue, printer routing config, print worker и print HTTP endpoints (status/retry) для нефискальных check и ticket. Рендер берётся из POS-69 ESC/POS renderer поверх IR из POS-73 engine.Render. Network TCP и Windows USB printer, bounded retry, timeout, status и audit. Print retry идемпотентен: не повторяет payment, не создаёт ticket, не меняет финансовое состояние.

Не включать: Cloud UI editor (POS-80); seed smoke сверх POS-73; fiscal logic, PNG/e-receipt, Level 2 шаблоны; checker flow.

Обязательны unit/integration tests (queue lifecycle, routing, retry/idempotency, status/endpoints, schema-изменения по migration policy). Hardware acceptance не заявлять без реального принтера: в итоговом Plane comment оставить точный manual hardware checklist и модели/порты для проверки. Документация профильная в том же изменении.

Проверки: cd cloud-backend && go mod tidy && go test ./...; cd pos-backend && go mod tidy && go test ./...; git diff --check. Переведи POS-74 в Review (label manual-validation, если нужна printer-приёмка), не Done. Commit/push/deploy без отдельного запроса не делай.
```

#### Итерация 8c — `POS-82` (Cloud Backend printer CRUD + mastersync stream)

```text
Реализуй Plane work item POS-82 в репозитории /home/master/repos/myhoreca-pos.

Сначала через Plane MCP прочитай work item, state, labels, blocked_by, description и comments. Начинай код только если state = Ready. Переведи в In Progress, оставь стартовый комментарий с планом.

Прочитай AGENTS.md, ALPHA-LAUNCH-CODEGEN-ITERATIONS.md, CURRENT-FUNCTIONAL-STATE.md, docs/backend/CLOUD-BACKEND-SPEC.md, docs/sync/ и фактический код receipt_templates (Cloud CRUD + mastersync) как образец для нового printer stream.

Фокус: Cloud-owned управление принтерами ESC/POS — принтер настраивается в Cloud и доставляется на Edge.

Schema (managed baseline, startup migration policy, UUIDv7):
  cloud_printers: id, org_id, restaurant_id, name, type (tcp|usb), address, port (NULL для USB),
  document_types (JSON array: precheck|ticket|kitchen_service|cash_in_out|acceptance),
  codepage (''|cp437|cp866, default ''), paper_cut_type (full|partial, default partial),
  cpl (32|42|48|56|80), is_active BOOL, version INT, created_at, updated_at.

Routes (RBAC: organization.manage):
  GET    /api/v1/printers?restaurant_id=
  POST   /api/v1/printers
  PATCH  /api/v1/printers/{id}
  DELETE /api/v1/printers/{id}   — soft-delete: is_active=FALSE, версия bumped

Mastersync stream printers:
  — Package: все is_active=TRUE принтеры для restaurant_id после assignment
  — Edge apply: атомарная замена строк receipt_printers в master-data apply tx
  — Checkpoint token: printers:{restaurant_id}:{MAX(updated_at)}:{count}
  — Soft-delete убирает принтер из следующего package; существующие print jobs не прерываются

Не включать: Cloud UI (POS-83); Edge consumer (POS-84); print queue или routing logic.

Проверки: cd cloud-backend && go mod tidy && go test ./...; shared/platform go test ./...; git diff --check.
Документация в том же изменении: CLOUD-BACKEND-SPEC.md (printers API table + stream), CURRENT-FUNCTIONAL-STATE.md.
Оставь итоговый Plane comment, переведи POS-82 в Review. Не Done.
```

#### Итерация 8d — `POS-83` (Cloud UI — управление принтерами ресторана)

```text
Реализуй Plane work item POS-83 в репозитории /home/master/repos/myhoreca-pos.

Начинай код только если POS-83 = Ready и POS-82 = Done (Cloud Backend CRUD + stream реализован).
Читай Plane полностью. Переведи в In Progress, оставь стартовый комментарий.

Прочитай AGENTS.md, ALPHA-LAUNCH-CODEGEN-ITERATIONS.md, CURRENT-FUNCTIONAL-STATE.md и фактический cloud-ui-g код. Используй receipt-templates feature как образец структуры.

Фокус: раздел Printers в Cloud Manager, restaurant scope.
— Route id printers в routes.ts, icon Printer из lucide-react в Sidebar.tsx
— Zod-схема printerSchema в shared/api/schemas.ts
— endpoints: listPrinters, createPrinter, updatePrinter, deactivatePrinter в shared/api/endpoints.ts
— i18n строки в ru.ts: nav.printers, printers.* (pageTitle, listTitle, editorTitle, form.*, documentTypes.*)
— PrintersPage.tsx: список принтеров ресторана + create/edit форма (inline или панельная)
— printerForms.ts: типы, defaults, валидация, payload builders
— printerForms.test.ts: toForm, build, validate (аналог receiptTemplateForms.test.ts)

Форма создания/редактирования:
  Название, Тип (tcp|usb), Адрес (IP для tcp, device path для usb), Порт (TCP),
  Типы документов (multiselect), Кодировка (CP437/CP866), CPL (32/42/48/56/80), Тип отреза (partial|full)

Все строки через i18n. Zod-валидация response. Safe error banner без raw payload.
Нет mock данных в production path.

Проверки: cd cloud-ui-g && npm run test -- --run && npm run build; git diff --check.
Документация: CURRENT-FUNCTIONAL-STATE.md (route printers в Cloud UI).
Оставь итоговый Plane comment, переведи POS-83 в Review. Не Done.
```

#### Итерация 8e — `POS-84` (POS Edge — printer config из Cloud sync, убрать env var)

```text
Реализуй Plane work item POS-84 в репозитории /home/master/repos/myhoreca-pos.

Начинай код только если POS-84 = Ready и POS-82 = Done (Cloud stream printers готов) и POS-74 = Done (print worker существует).
Читай Plane полностью. Переведи в In Progress, оставь стартовый комментарий.

Прочитай AGENTS.md, ALPHA-LAUNCH-CODEGEN-ITERATIONS.md, CURRENT-FUNCTIONAL-STATE.md, docs/backend/RECEIPT-PRINT-SPEC.md, фактический код pos-backend: mastersync apply, print worker и routing.

Фокус: POS Edge получает конфиг принтеров из Cloud sync stream printers вместо legacy deployment printer routing.

Изменения:
1. Managed baseline: таблица receipt_printers в SQLite pos-edge.db
   (id, restaurant_id, name, type, address, port, document_types JSON, codepage, paper_cut_type, cpl,
    is_active BOOL, cloud_version INT, synced_at TIMESTAMPTZ)
2. Mastersync apply: stream printers → атомарная замена receipt_printers для restaurant_id в apply tx
3. Print worker routing: читает принтеры из receipt_printers по restaurant_id + document_type
   Один document_type может иметь несколько принтеров — send job to each.
   Если подходящих принтеров нет → print job failed с safe error_code, не panic.
4. legacy deployment printer routing: удалить из runtime, примеров, docker-compose и документов.
   Если задан при старте — logged warning, игнорируется.

Не включать: изменение print queue lifecycle; fiscal logic; Cloud-side код (POS-82).

Обязательны тесты: mastersync ingest shape, routing logic (document_type matching, no-printer safe error),
migration baseline. go test ./... зелёный.

Проверки: cd pos-backend && go mod tidy && go test ./...; shared/platform go test ./...; git diff --check.
Документация в том же изменении: RECEIPT-PRINT-SPEC.md (убрать legacy deployment printer routing, добавить receipt_printers sync), CURRENT-FUNCTIONAL-STATE.md.
Оставь итоговый Plane comment с manual hardware checklist для проверки routing через реальный принтер.
Переведи POS-84 в Review (label manual-validation). Не Done.
```

### Итерация 9 — `POS-40`

```text
Реализуй Plane work item POS-40 в репозитории /home/master/repos/myhoreca-pos.

Сначала через Plane MCP прочитай work item, state, module, cycle, labels, parent, relations и comments. Начинай код только если state = Ready и фактические blocked_by закрыты (POS-48 и POS-65 = Done). Переведи задачу в In Progress и оставь стартовый комментарий с планом и проверками.

Прочитай AGENTS.md, ALPHA-LAUNCH-CODEGEN-ITERATIONS.md, EXHIBITION-ALPHA-PILOT-REQUIREMENTS.md, MVP-PILOT-REQUIREMENTS.md, ROADMAP.md, CURRENT-FUNCTIONAL-STATE.md, docs/backend/CLOUD-BACKEND-SPEC.md и фактический код ticket_units, financial_operations и outbox.

Фокус: bounded operational projection/API для проданных ticket units.
Нужны:
  GET /api/v1/reporting/sales?restaurant_id=&business_date_from=&business_date_to=&shift_number=&limit=&offset=
  Данные: sold tickets, gross/refund/net minor, average check, stable ticket category/service name
  RBAC: organization.manage или restaurant membership
  Deterministic paging, freshness/incomplete marker
  Drill-down до ticket → financial_operation

Не включать: checker usage analytics, ClickHouse OLAP, графики, CSV export.

UUID — только UUIDv7. Backend authoritative.

Проверки: cd cloud-backend && go mod tidy && go test ./...; cd cloud-ui-g && npm run build; git diff --check.
Документация в том же изменении: CLOUD-BACKEND-SPEC.md (reporting API table), CURRENT-FUNCTIONAL-STATE.md.
Оставь итоговый Plane comment, переведи POS-40 в Review. Не Done.
```

### Итерация 10 — `POS-41`

```text
Реализуй Plane work item POS-41 в репозитории /home/master/repos/myhoreca-pos.

Начинай код только если POS-41 = Ready и POS-40 = Done (sales API backend реализован).
Читай Plane полностью. Переведи в In Progress, оставь стартовый комментарий.

Прочитай AGENTS.md, ALPHA-LAUNCH-CODEGEN-ITERATIONS.md, CURRENT-FUNCTIONAL-STATE.md, docs/ui/ и фактический cloud-ui-g код. Прочитай POS-40 endpoint контракт из CLOUD-BACKEND-SPEC.md.

Фокус: заменить dashboard placeholders реальными данными из POS-40 sales API.
— Authorized restaurant filter (sidebar selector)
— Ticket category/service/date/shift фильтры
— Sold/gross/refund/net/average check, freshness marker
— Loading/empty/error states
— Bounded drill-down: check → order line → ticket → financial operation

Backend RBAC authoritative. Все строки через i18n (ru.ts). Zod-схемы для новых API responses.
Safe error banner без raw payload. Нет mock данных в production path.

Проверки: cd cloud-ui-g && npm run test -- --run && npm run build; git diff --check.
При наличии Playwright E2E возможности — добавить smoke test основного dashboard flow.
Документация: CURRENT-FUNCTIONAL-STATE.md (Cloud UI раздел).
Оставь итоговый Plane comment, переведи POS-41 в Review. Не Done.
```

### Итерация 11 — `POS-63`

```text
Реализуй Plane work item POS-63 в репозитории /home/master/repos/myhoreca-pos.

Начинай только если POS-63 = Ready, POS-40 = Done и POS-42 = Done.
Читай Plane полностью. Переведи в In Progress, оставь стартовый комментарий.

Прочитай AGENTS.md, ALPHA-LAUNCH-CODEGEN-ITERATIONS.md, CURRENT-FUNCTIONAL-STATE.md, docs/backend/, docs/architecture/; фактический код Cloud workers и telegram-worker entitlement gate.
Для актуального Telegram Bot API сначала получи документацию через Context7 согласно AGENTS.md.

Фокус:
— Restaurant Telegram settings (bot token, chat_id) через Cloud UI + CRUD route
— Confirmed chat_id onboarding: send test message → confirm → сохранить
— Schedule trigger (daily/shift-close) + cash-shift-close trigger
— Idempotent report occurrence: одна запись per restaurant+trigger+business_date, retry/backoff
— Safe secrets: bot token не логируется, не раскрывается в API response
— Отчёт использует POS-40 projection, содержит freshness marker
— telegram-worker недоступен без entitlement telegram-worker (gate из POS-42)

Username хранится только как display metadata.

Проверки: cd cloud-backend && go mod tidy && go test ./...; cd cloud-ui-g && npm run build; git diff --check.
Документация в том же изменении: CLOUD-BACKEND-SPEC.md, CURRENT-FUNCTIONAL-STATE.md.
Оставь итоговый Plane comment, переведи POS-63 в Review. Не Done.
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
