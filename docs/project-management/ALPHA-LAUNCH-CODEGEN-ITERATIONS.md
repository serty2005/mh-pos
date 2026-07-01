# Итерации кодогенерации выставочного запуска

Статус: Wave 1–2 закрыты; `POS-65`, `POS-48`, `POS-52`, `POS-53`, `POS-61`, `POS-62` и `POS-42` находятся в `Done`. Wave 3 receipt-printing (P1-декомпозиция `POS-64`) в приемочной проверке: `POS-68/69/70/71/72/73/74/81/82/83/84` = `Done`, `POS-67/80` = `Review`, umbrella `POS-64` = `Ready`. Clean-stack аудит 2026-06-28 нашёл и исправил POS-84 blocker: Cloud baseline/exchange не принимал stream `printers`; Cloud runtime поднят до `0.1.15`. Физический TCP ESC/POS стенд: `10.25.1.201:9100`, CPL=48. Backend/worker отправил `precheck` и `ticket` успешно; оператор подтвердил выход двух чеков, после чего ESC/POS/SVG renderer исправлен для крупного `{f:double}` текста через эффективную ширину `CPL/2`. Зонтичная `POS-64` остаётся открытой до закрытия POS-67/POS-80 и повторного операторского подтверждения ticket. ESC/POS по умолчанию CP437 (Indonesia/English launch). 30.06.2026 добавлены Cloud client Docker/Traefik, License Server Linux/systemd/CD-template и POS Edge Windows/webwallpaper deployment paths. POS-85 стартовал как Edge schema baseline для sales points, restaurant sections, print routes, override audit и print targets; POS runtime version поднят до `0.1.10`. Вместо финансовой ветки ближайший фокус — `POS-86…POS-89`; затем остаются `POS-40` Cloud sales API, `POS-41` Cloud dashboard, `POS-63` Telegram reports, deployment/backup/go-no-go tasks `POS-43…45`, а также review задач `POS-91` и `POS-95`.

Дата проверки: 2026-06-30.

Этот документ является runbook для непосредственной генерации кода. Он не копирует Plane: перед каждой итерацией агент обязан заново прочитать work item, relations и комментарии. Plane хранит текущее состояние работы, Git — требования, код и тесты.

## Срез Cloud client Docker/Traefik deployment

Обновлено 2026-06-30.

Реализовано сейчас:

- Подготовлен alpha/pre-Kubernetes deployment path для нескольких независимых клиентских Cloud-стеков на одной Linux production VM: общий Traefik v3 reverse proxy, отдельный Docker Compose project на пару `CLIENT_SLUG` + `SERVER_SLUG`, отдельные volumes/env/domains и общая только сеть `traefik_proxy`.
- Один клиентский стек описан в `deploy/cloud-client/docker-compose.cloud-client.yml`: Cloud Backend, PostgreSQL, ClickHouse и Cloud UI без публикации host ports для PostgreSQL, ClickHouse и Cloud Backend.
- Cloud UI и Cloud API работают на одном клиентском домене: UI на `/`, API через Traefik `PathPrefix(/api)` на Cloud Backend port `8090`; production build Cloud UI использует `VITE_CLOUD_API_BASE=/api/v1`.
- Cloud API production compose задает runtime config через env и `CLOUD_CONFIG_PATH=""`, чтобы не править `/app/config/cloud-api.docker.json` внутри контейнера и не перекрывать production DSN/ClickHouse/License settings.
- Добавлен production Dockerfile для `cloud-ui-g` и `deploy/cloud-client/docker-compose.cloud-build.yml` для сборки/push `mhpos-cloud-api` и `mhpos-cloud-ui` в Docker Hub по immutable `MHPOS_VERSION` без отдельного shell-скрипта.
- Runbook `docs/deployment/CLOUD-CLIENT-DOCKER-TRAEFIK.md` фиксирует запуск Traefik, добавление/обновление/rollback клиента, backup требования, multi-client isolation и будущую миграцию в Kubernetes.

Обновление клиентов:

- оператор собирает и публикует новые Docker images через `docker compose --env-file deploy/cloud-client/build.env -f deploy/cloud-client/docker-compose.cloud-build.yml build` и `push` по explicit immutable tag;
- в `clients/<client>-<server>.env` меняется `MHPOS_VERSION`;
- выполняется `docker compose pull` и `docker compose up -d` с тем же `-p mhpos-<client>-<server>`;
- smoke включает `/health`, Cloud UI, Cloud API `/api/v1`, Edge pairing через внешний License Server и Edge sync.

Запланировано далее:

- перенести этот path в CI/CD: build/push immutable images и controlled VM rollout;
- подготовить Kubernetes/Helm/GitOps контур, где текущие env values станут Helm values/Secrets, Traefik labels превратятся в Ingress/IngressRoute, а client stack boundary сохранится как namespace/release-per-client.

Вне текущего объема:

- Kubernetes/Helm manifests;
- хранение production secrets в Git;
- DB downgrade и автоматический restore без отдельной процедуры.

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
- POS-85 Edge schema baseline: `sales_points`, `restaurant_sections`, `print_routes`, `printer_route_override_audit`, `print_job_targets` и nullable `cash_sessions.sales_point_id`; POS runtime version `0.1.10`.

Не реализовано сейчас и не входит в закрытие Wave3/POS-85:

- обязательное открытие cash session внутри sales point на service layer;
- service printer/report printer routing через `print_routes` в worker;
- Edge-side immediate override API, outbox push и Cloud projection override state;
- автоматическое создание/исполнение `print_job_targets` и отдельный retry/status/error на каждый физический принтер.

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

`POS-65` уже закрыта в Plane как `Done`, поэтому не запускать ее повторно.

Ближайший кодовый путь:

1. `POS-40` — Cloud sales API, сейчас `Ready`; зависимости `POS-48` и `POS-65` закрыты.
2. `POS-63` — Telegram reports, сейчас `Ready`, но зависит от данных POS-40 для содержательного отчета; если начинать раньше, явно зафиксировать ограничение.
3. `POS-41` — Cloud dashboard, остается `Specified` до готовности POS-40.
4. `POS-64` — umbrella physical printing, сейчас `Ready`, но фактический остаток процессный: закрыть `POS-67`/`POS-80` и повторить ticket print acceptance после `{f:double}` fix.
5. `POS-85…POS-89` — ресторанная схема печати на Edge: точки продаж, секции, route assignments, per-printer targets, Edge override, settings UI и exhibition smoke. Выполнять до финансовой ветки `POS-40`.
6. `POS-43`/`POS-44`/`POS-45` — deployment, backup/restore и E2E go/no-go, пока `Specified`; часть deployment runbook уже добавлена 30.06.2026.

Отдельно не терять review-хвосты ручного аудита: `POS-91` (созданный заказ не становился текущим) и `POS-95` (canonical licensing flow) находятся в `Review`; `POS-92`/`POS-93`/`POS-94` остаются `Specified`.

## Исторический срез после итерации 3

Проверено 2026-06-21 после коммита `e0e8357` и последующих документальных уточнений лицензируемых границ.

- `POS-61`, `POS-62` и `POS-42` находятся в `Done` в Plane после ручной проверки.
- POS-42 реализован сейчас в runtime code, security tests, локальном Docker smoke и профильной документации.
- POS-42 Plane item прочитан: state `Review`, labels `agent-ready`, `tests`, `security`, `frontend`, `backend`; итоговый комментарий фиксирует License Server, fail-closed Cloud/Edge gates, stale grace, revoke/expiry, backup, UI visibility, sync filtering и worker enforcement.
- `list_work_item_relations` для POS-42 все еще падает на MCP schema validation: Plane server возвращает relation objects, connector ожидает строки UUID. Ограничение остается в `POS-66`; для текущего review использована fallback-таблица зависимостей ниже.

Этот раздел оставлен как история принятия решений перед `POS-65`. По состоянию на 30.06.2026 `POS-65` уже `Done`; нижеуказанные ограничения не являются разрешением заново запускать итерацию.

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

После этого review runtime code уже менялся в последующих задачах; строка выше оставлена как исторический контекст 21.06.2026, а не как текущий статус.

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

Статус на 30.06.2026: выполнено, Plane state `Done`. Промпт ниже оставлен только как historical audit trail.

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

## Wave 3b. Edge restaurant print routing

Этот блок выполняется до финансовой ветки `POS-40…POS-41`, потому что выставочный
стенд сначала должен уметь настраивать печать по точкам продаж и секциям со стороны
POS Edge.

Состояние Plane на 2026-06-30: `POS-85…POS-89` были `Specified`, descriptions и
comments пустые. MCP read relations возвращает schema validation error, но из
ошибки видна фактическая цепочка: `POS-85` блокирует `POS-86`, `POS-87`, `POS-88`,
`POS-89`; `POS-86` блокирует `POS-87`, `POS-88`, `POS-89`; `POS-88` блокирует
`POS-89`; `POS-89` является финальным smoke.

Профильная документация: `docs/backend/EDGE-PRINT-ROUTING-SPEC.md`.

### Итерация 8f — `POS-85`

Статус выполнения на 2026-06-30: реализовано сейчас на уровне Edge SQLite managed
baseline, startup schema verification, tests и документации. Runtime worker routing
не переведен на targets в этой итерации.

```text
Используй универсальный промпт для POS-85.

Фокус: Edge data model для точек продаж, секций, маршрутов печати и
print_job_targets. Добавить managed SQLite baseline и startup schema verification:
sales_points, nullable cash_sessions.sales_point_id, restaurant_sections,
print_routes, printer_route_override_audit, print_job_targets. Поднять POS runtime
version. Добавить schema tests и профильную документацию.

Не включать: POS Edge HTTP API/services, worker target lifecycle, Edge -> Cloud
override sync, POS UI settings, физический smoke, fiscal adapter.

Проверки: cd pos-backend && go mod tidy && go test ./...; git diff --check.
Документация: EDGE-PRINT-ROUTING-SPEC.md, RECEIPT-PRINT-SPEC.md,
POS-DATA-AND-MIGRATIONS.md, POS-BACKEND-SPEC.md, CURRENT-FUNCTIONAL-STATE.md.
```

### Итерация 8g — `POS-86` (расширенный scope, зафиксировано 2026-07-01)

Статус на 2026-07-01: **реализовано и доведено до зелёных проверок**. Объём вырос далеко за
пределы исходного текста задачи в Plane — это сделано осознанно, после серии уточняющих
вопросов продукту, зафиксировано комментарием в самой задаче `POS-86`. Реализация прошла в
две сессии: основной прогон (коммит `c8743d7`) и последующая доводочная сессия, которая
прошлась по всей реализации пункт за пунктом, нашла и закрыла реальные пробелы (агрегация
attempts у print_jobs, hardcoded 3-секундный real-time wait в тестах, пропущенный validate-case
и порядок mastersync-стримов, отсутствовавший целиком `cancel-unconfirmed`/`print-confirmation
retry`, CHECK-constraint на `manager_override_audit.action`, несовместимость `seed-dev-system.py`
с новой обязательной моделью точек продаж/секций) и добавила недостающие тесты. `go test ./...`
зелёный в `pos-backend` и `cloud-backend`, `go vet` чистый, `git diff --check` чистый.
Подробности фактической реализации — в `docs/backend/EDGE-PRINT-ROUTING-SPEC.md`. Промпт ниже
оставлен как исторический план итерации, а не как чек-лист "что ещё сделать".

Физическая проверка на реальном USB-принтере 2026-07-01: Xprinter XP-365B (USB\VID_1FC9&PID_2016),
подключенный к текущему Windows 11 хосту. Найден и задокументирован реальный gap в
`RECEIPT-PRINT-SPEC.md`: адрес `\\.\USB001`, который документация раньше называла рабочим
device path для `type=usb`, на Windows 10/11 не открывается (`os.OpenFile`/`CreateFile` →
file not found), так как `usbprint.sys` больше не создает такой legacy compatibility symlink.
Рабочий Win32 path — SetupAPI device interface (`\\?\USB#VID_...&PID_...#<serial>#{28d78fad-...}`,
`GUID_DEVINTERFACE_USBPRINT`), получаемый через `Get-PnpDevice -Class USBPrint`/реестр. С этим
путем без изменений runtime-кода подтвержден сквозной прогон через реальную печать:
raw `escpos.WriteRaw` из Go, затем полный production pipeline (`print_routes` → template
engine → ESC/POS renderer → per-printer FIFO worker) через временный manual-тест в
`pos-backend/internal/pos/app` (env-gated, не закоммичен, удален после проверки) — print job
`check_nonfiscal` перешел в `succeeded`, документ физически напечатан оператором подтвержден.
Документация исправлена в том же изменении (`RECEIPT-PRINT-SPEC.md`); значение `\\.\USB001`
больше не рекомендуется как рабочий address для новых USB-принтеров.

```text
Используй универсальный промпт для POS-86 в репозитории /home/master/repos/myhoreca-pos.

Начинай код только если POS-86 = Ready, POS-85 = Review/Done в Git есть Edge schema
baseline с sales_points, restaurant_sections, print_routes, printer_route_override_audit
и print_job_targets. Сначала перечитай актуальный POS-86 (description и comments — там
зафиксирована история решений), POS-82/83/84, и этот раздел целиком.

Все архитектурные вопросы по состоянию на 2026-07-01 закрыты — открытых развилок,
требующих решения перед стартом, в этом промпте не осталось. Если в процессе реализации
обнаружится новое расхождение с этим планом — остановиться и оставить Plane comment с
точным gap, а не угадывать дальше.

## Бизнес-модель маршрутизации печати — строго, без fallback

document_type печатается через ровно один scope_type, без каскадов и без fallback на
старую логику receipt_printers.document_types (этот fallback убирается из worker'а целиком,
ничего из старого routing-кода не остаётся как запасной путь):

| document_type     | scope_type   | restaurant_sections.mode | Назначение |
|---|---|---|---|
| check_nonfiscal    | sales_point  | —                 | квитанция об оплате / нефискальный чек точки продаж (фискальное устройство — отдельно, позже) |
| precheck           | section      | hall_section      | пречек гостю до оплаты + спецдокументы секции зала |
| ticket              | section      | hall_section      | QR-билет печатается там же, где секция зала |
| kitchen_service     | section      | kitchen_workshop  | кухонный чек цеха (схема готова, но в этой итерации НИКАКОЙ источник не создаёт такие print_jobs — конфигурация принимается заранее, worker её не использует) |
| report              | restaurant   | —                 | отчёты ресторана (аналогично — конфигурация принимается, но не используется) |

Правило закодировать один раз (например `receipt.RequiredScopeType(documentType)` +
`receipt.RequiredSectionMode(documentType)`), использовать и в валидации print_routes, и в
worker routing match, и продублировать DB-триггером на print_routes (по аналогии с уже
существующими триггерами вроде `recipe_lines_good_or_semi_finished_*` в `001_init.sql`) —
несовместимую комбинацию document_type/scope_type/section.mode нельзя записать в БД даже в
обход application layer.

## scope_id резолвится один раз при enqueue, не в момент обработки job

Не резолвить scope в момент обработки job через "текущую открытую кассовую смену" или
другую угадайку постфактум — резолвить один раз в момент постановки job в очередь, пока
есть авторитетный контекст транзакции:

- `check_nonfiscal` → `scope_id = cashSession.SalesPointID` (сессия теперь ВСЕГДА привязана
  к точке продаж — см. ниже, поле никогда не NULL для новых сессий).
- `precheck`/`ticket` → `scope_id = table.SectionID`, где `table = GetTable(order.TableID)`.
  Без особых случаев и без fallback: стол ОБЯЗАН принадлежать секции (см. раздел "Стол
  обязан принадлежать секции" ниже) — значит `table.SectionID` существует всегда, для
  любого заказа, включая заказы без явно выбранного гостевого стола. Резолв — прямой
  единственный путь, никакого ветвления по имени стола или по `hall_id` больше нет.

`print_jobs` получает новую nullable колонку `scope_id TEXT` (nullable только на случай
будущих document_type, для которых правило ещё не задано; для `check_nonfiscal`/
`precheck`/`ticket` она всегда заполняется по правилу выше). Если когда-либо scope не
резолвился — print job всё равно создаётся с `scope_id=NULL` и позже завершается worker'ом
безопасной ошибкой при разборе routing; финансовая операция (оплата/check/ticket issuance)
к этому моменту уже закоммичена и не блокируется и не откатывается этим шагом.

## Per-printer буфер — claim на уровне target, а не job

Один и тот же `printer_id` может встречаться в нескольких `print_routes` (разные
scope/document_type — уже поддержано существующим unique index). Claim переключить с
`print_jobs` на `print_job_targets`, эксклюзивность по `printer_id` зашить прямо в SQL:

```sql
UPDATE print_job_targets
SET status='processing', locked_by=?, locked_at=?
WHERE id = (
  SELECT t.id FROM print_job_targets t
  WHERE t.status='pending' AND (t.next_attempt_at IS NULL OR t.next_attempt_at <= ?)
    AND NOT EXISTS (
      SELECT 1 FROM print_job_targets c
      WHERE c.printer_id = t.printer_id AND c.status='processing'
    )
  ORDER BY t.printer_id, t.created_at
  LIMIT 1
)
```

`print_jobs.status` агрегируется из дочерних targets: succeeded когда все required targets
succeeded; failed если хотя бы один required target исчерпал попытки; иначе pending с
ближайшим `next_attempt_at` среди ещё не терминальных targets.

## Владение данными: Cloud vs Edge

- `sales_points`, `restaurant_sections` — полностью Cloud-owned master data, по образцу
  `receipt_printers`/POS-82-84: новый cloud-backend CRUD (create/update/archive, RBAC
  `organization.manage`), новые mastersync streams `sales_points` и `restaurant_sections`
  (Postgres schema в cloud-backend, аналогично существующему стриму `printers`). Edge —
  только read-only ingest через mastersync apply в локальные read-replica таблицы. Никакого
  create/update/delete HTTP API для этих двух сущностей на Edge нет вообще в этой итерации
  (кроме отдельного offline-флоу — см. "Вне scope" ниже, это отдельная Plane-задача).
- cloud-ui-g экран для управления ими **не строится** в этой итерации — заведена/должна
  быть заведена отдельная Plane-задача (follow-up, blocked_by `POS-86`).
- `print_routes` остаётся **полностью Edge-local** (CRUD на Edge, RBAC
  `pos.print_routing.manage`/`pos.print_routing.view`), `origin='edge_override'` для всех
  записей в этой итерации; `origin='cloud'` — задел в схеме на будущее, ничего сейчас его
  не заполняет. Каждая мутация route пишет строку в существующую
  `printer_route_override_audit` (без outbox push в Cloud — это `POS-87`).
- Из `sales_points` убрать колонку `cash_printer_id` (рудимент POS-85): она дублирует то,
  что теперь полностью выражает `print_routes` — должен остаться один источник истины для
  назначения принтера.

## Стол обязан принадлежать секции (закрыто 2026-07-01, финальная версия)

Более ранняя версия этого раздела (вариант с `is_default`-секцией без стола и
сохранением `EnsureSystemFloor`) **устарела и заменена целиком** следующим решением —
оно решает ту же проблему элегантнее и заодно убирает источник проблемы полностью, а не
обходит его.

Суть: `hall_id` — это исключительно UI-категоризация стола для визуальной схемы зала
(сегодня даже без координат — просто именованная группировка), и не должен быть входом
ни для какой бизнес-логики. Бизнес-логике (печать, заказ, оплата) нужна не схема зала, а
**секция** — обязательная операционная привязка стола.

Новый, единственный источник истины:

- **Стол обязан принадлежать ровно одной секции.** `tables.hall_id` становится nullable
  (чисто декоративное, опциональное поле для UI-группировки, никогда не участвует в
  routing-логике) и добавляется `tables.section_id TEXT NOT NULL REFERENCES
  restaurant_sections(id)` — обязательная ссылка, валидируется (app-layer или trigger),
  что секция того же `restaurant_id` и имеет `mode='hall_section'`. Стол не может
  существовать без секции — секцию для стола можно сменить (move), но нельзя обнулить.
  "Удалить стол из секции" в UI означает удалить/деактивировать сам стол, а не оставить
  его без секции.
- При создании ресторана Cloud **автоматически и безусловно** (не зависит от лицензии
  `table-mode` — это теперь базовая инфраструктура любого кассового flow, а не
  гостевая функция выбора стола) создаёт:
  - одну секцию `mode='hall_section'`, `hall_id=NULL`, `is_default=1`;
  - один стол внутри неё, `tables.is_default=1`.
  Партиционные unique-индексы: `UNIQUE(restaurant_id) WHERE
  restaurant_sections.mode='hall_section' AND is_default=1` и `UNIQUE(restaurant_id) WHERE
  tables.is_default=1` — ровно одна дефолтная секция и один дефолтный стол на ресторан.
- `sales_points` получает `default_table_id TEXT NOT NULL REFERENCES tables(id)` (тот же
  restaurant_id). При создании точки продаж, если менеджер не указал стол явно,
  автоматически подставляется дефолтный (`is_default=1`) стол ресторана; менеджер может
  позже назначить точке продаж любой другой существующий стол.
- Защита от удаления (cloud-backend, application-level safe error, не просто soft-delete):
  - нельзя деактивировать/удалить стол, если он `is_default=1`, ИЛИ если он указан как
    `default_table_id` хотя бы у одной точки продаж;
  - нельзя деактивировать/удалить секцию, если она `is_default=1`, ИЛИ если среди её
    столов есть стол, указанный как `default_table_id` хотя бы у одной точки продаж.
- Новая секция, созданная менеджером вручную, создаётся как черновик (`is_active=false`)
  без столов. Активировать (`is_active=true`) секцию с `mode='hall_section'` нельзя, пока
  у неё нет ни одного стола — иначе система не может считать секцию пригодной к работе.
  (Аналогичное правило для `kitchen_workshop` через склад — относится к кухонному/
  складскому контуру, не детализируется в этой итерации.)
- DB-триггер на `print_routes` (document_type↔scope_type↔section.mode) не меняется — он
  проверяет только `mode`, не `hall_id`.

### Ретируется `EnsureSystemFloor`/`GetSystemTable`/`__counter__` (POS-53)

Раз у каждой точки продаж есть собственный реальный (Cloud-owned) `default_table_id`, а
кассовая сессия ВСЕГДА привязана к точке продаж — синтетический Edge-local `__counter__`
hall/table (`EnsureSystemFloor` в `infra/sqlite/floor_repository.go`, `GetSystemTable`,
вызов в `mastersync/service.go` при apply restaurant-stream) **полностью ретируется** в
этой итерации: удалить функции, вызовы и тесты на них. Создание заказа без явно
выбранного гостевого стола (`app/order/service.go`, текущая строка ~209) резолвит стол
как `cashSession.SalesPointID → salesPoint.DefaultTableID` вместо `GetSystemTable()`. Мы
pre-pilot — существующие dev/test БД пересоздаются по политике AGENTS.md, чистый разрыв
без миграции данных безопасен.

Это затрагивает существующий, протестированный код POS-53 — ожидаемо потребуется
поправить существующие тесты `app/order`, `infra/sqlite/floor_repository_test.go` и
аналогичные. Это намеренное, согласованное расширение scope, а не случайная поломка.

### Зависимость от лицензии table-mode

Бутстрэп-секция и бутстрэп-стол существуют и обслуживают базовый кассовый flow **вне
зависимости от лицензии `table-mode`** — без них невозможна ни одна оплата ни в одном
режиме. Лицензия `table-mode` по-прежнему гейтит на Edge только **дополнительные**, явно
создаваемые менеджером halls/tables/sections для гостевого выбора стола (как и сейчас —
`edgeModuleForRequest` для `GET /api/v1/halls`/`/tables`); внутренний резолв
routing/`default_table_id` через эти таблицы не должен проходить через этот гейт.

## Обязательность точки продаж у кассовой сессии

- `cash_sessions.sales_point_id` становится `NOT NULL` в managed baseline (пересоздаём
  pre-pilot baseline, бампим `MH_POS_VERSION`).
- Команда открытия кассовой смены требует `sales_point_id`, валидирует, что точка продаж
  активна и принадлежит ресторану. Никакого автосоздания точки продаж по умолчанию —
  единственный путь её появления — Cloud CRUD (вручную менеджером) или seed-скрипт.
- Текущий `pos-ui-g` не передаёт `sales_point_id` при открытии смены — открытие смены
  сломается, пока (a) менеджер не создаст точку продаж через Cloud CRUD/seed и (b) pos-ui-g
  не научится её передавать (это отдельная, не входящая в `POS-86` работа по UI —
  зафиксировать как явный известный разрыв в Plane-комментарии и в
  `CURRENT-FUNCTIONAL-STATE.md`, не пытаться чинить pos-ui-g в этой задаче).
- Защита: нельзя деактивировать/удалить через Cloud CRUD последнюю активную точку продаж
  ресторана — application-level safe-guard в cloud-backend.
- Дефолтная точка продаж не создаётся автоматически никогда и нигде в штатном режиме.

## Печать как гейт подтверждения оплаты

Заодно чиним давно задокументированный, но не закрытый gap POS-72: после полной оплаты
вместо `document_type=precheck` в очередь встаёт `document_type=check_nonfiscal` (источник —
Check, не Precheck) + ticket-jobs. Дополнительно — новая точка enqueue: `IssuePrecheck`
теперь сама ставит `precheck`-job (печать пречека гостю до оплаты, маршрутизируется через
section).

- `checks` получает `print_confirmed_at TEXT` (nullable; NULL = не подтверждено). Стампится
  НЕ HTTP-таймаутом, а worker'ом: когда check_nonfiscal-job и все ticket-jobs этого чека
  достигают `succeeded`, worker проставляет `print_confirmed_at`. Работает независимо от
  того, успели дождаться в рамках HTTP wait или подтверждение пришло позже через retry.
- `CapturePrecheckPayment` коммитит Payment/Check/Ticket как сейчас, безусловно и быстро
  (НЕ держать открытую БД-транзакцию на время печати/сетевого I/O к принтеру). После коммита
  — bounded wait (конфигурируемый таймаут, по умолчанию несколько секунд) в ожидании
  `print_confirmed_at`, опрашивая статус targets без удержания транзакции. Ответ включает
  `print_confirmation: {confirmed: bool, targets: [...]}`.
- Если не подтвердилось за таймаут — оплата остаётся проведённой (деньги не трогаем),
  `print_confirmed_at` остаётся NULL.
- `GET /checks/{id}/print-confirmation` — поллинг статуса после таймаута (для будущего UI).
- `POST /checks/{id}/print-confirmation/retry` — пересборка targets из ТЕКУЩИХ активных
  `print_routes` (если принтер физически заменили — подхватится новый), сброс
  check_nonfiscal/ticket jobs этого чека. RBAC: `pos.print.retry`.
- `POST /orders/{id}/cancel-unconfirmed` — только пока `print_confirmed_at IS NULL`,
  требует manager PIN (по образцу `CancelPrecheck`/`resolveManagerOverrideByPIN`), новое
  право `pos.order.cancel_unconfirmed`. Транзакционно: полноценный refund (переиспользовать
  существующий `recordFinancialOperation`/refund-механизм, kind=full), void всех
  `ticket_units` чека (новая колонка `ticket_units.status` `active|voided`), soft-cancel
  заказа (`orders.status='cancelled'`, новый repo-метод `UpdateOrderCancelled`, новая
  колонка `orders.cancelled_at`). Заказ пропадает из активных списков
  (`ListActiveOrders`/`GetCurrentOrder`), но остаётся в БД и синкается в Cloud со всей
  историей событий. Восстановление из UI не предусмотрено.
- Три отдельных события outbox, не переиспользующих существующие Payment/Check события:
  `CheckPrintUnconfirmedRefunded`, `TicketVoided` (на каждый ticket), `OrderCancelled`.
- POS UI для этого флоу (спиннер ожидания, модалка retry/удалить на экране оплаты)
  сознательно не строится в этой итерации (не входит ни в исходный `POS-86`, ни в `POS-88`,
  который про настройки, а не про экран оплаты). Backend-контракт должен быть
  самодостаточным для будущего UI; зафиксировать в `CURRENT-FUNCTIONAL-STATE.md`, что без
  этого UI фичу можно проверить только через API/тесты, не руками кассира.

## Edge HTTP API (новое в этой итерации)

- `GET /api/v1/print-routing/printers` — read-only список `receipt_printers` (нет общего
  списка сегодня, только `ListReceiptPrinters` отфильтрованный по document_type).
- `GET /api/v1/print-routing/sales-points`, `GET /api/v1/print-routing/sections` —
  read-only отражение Cloud-synced данных (для select'ов в будущем print_routes UI).
- `GET/POST/PATCH/DELETE /api/v1/print-routing/routes` — полный CRUD print_routes
  (DELETE = soft-deactivate, audit action=delete).
- `POST /api/v1/print/jobs/{id}/targets/{target_id}/retry` — retry одного target без
  пересборки routing (тонкий инструмент, в отличие от job-level retry, который пересобирает
  routing).
- `GET /api/v1/print/jobs/{id}` — дополнить ответ списком targets.
- `GET /api/v1/checks/{id}/print-confirmation`, `POST .../retry`,
  `POST /api/v1/orders/{id}/cancel-unconfirmed` — см. секцию про гейт оплаты выше.

RBAC: новые права `pos.print_routing.view`, `pos.print_routing.manage` (RoleManager — оба;
RoleSupportAdmin — только view), `pos.order.cancel_unconfirmed` (RoleManager, manager PIN
override).

## Cloud-backend (новое в этой итерации)

- Postgres schema: `sales_points` (включая `default_table_id NOT NULL`),
  `restaurant_sections` (включая `is_default`, `hall_id` nullable-декоративный) — зеркально
  Edge-схеме из POS-85, но без `cloud_version`/`synced_at` (Edge-side поля; Cloud хранит
  canonical version). Расширение существующей (POS-52/61/62) таблицы `tables`:
  `section_id NOT NULL REFERENCES restaurant_sections`, `is_default BOOLEAN`, `hall_id`
  становится nullable.
- CRUD: `POST/GET/PATCH/(archive) /api/v1/sales-points`, `/api/v1/restaurant-sections`, RBAC
  `organization.manage`. Существующий `POST/PATCH /api/v1/floor/tables` (и
  `/api/v1/tables`) расширяется обязательным `section_id` в payload — это меняет контракт
  уже работающего endpoint'а, обновить существующие тесты cloud-backend table CRUD.
  Защиты: `is_default` (стол и секция) от деактивации; стол/секция, на которые ссылается
  хотя бы одна `sales_points.default_table_id`, от деактивации; активация секции
  `mode='hall_section'` без столов — ошибка; защита "последняя активная точка продаж
  ресторана" от деактивации.
- Новые mastersync streams `sales_points`, `restaurant_sections`: package = все is_active
  строки ресторана, checkpoint token по образцу
  `printers:{restaurant_id}:{MAX(updated_at)}:{count}`. Существующий стрим `floor`
  расширяется полем `section_id`/`is_default` в `EdgeTable` payload.
- Идемпотентное провижинирование при создании ресторана: бутстрэп-секция (`hall_id=NULL`,
  `is_default=1`) + бутстрэп-стол внутри неё (`is_default=1`) — безусловно, не зависит от
  лицензии `table-mode` (см. "Зависимость от лицензии table-mode" выше).

## Edge SQLite — зеркальные изменения схемы (легко пропустить, явно проверить)

Все Cloud-схемные изменения выше зеркалятся в `pos-backend/migrations/sqlite/001_init.sql`
(Edge read-replica, заполняется только через mastersync apply, без локального write API для
sales_points/restaurant_sections/tables) — без этого `table.SectionID`/
`salesPoint.DefaultTableID` физически негде прочитать на Edge:

- `tables`: сегодня `hall_id TEXT NOT NULL REFERENCES halls(id)` и
  `UNIQUE(hall_id, name)` (строки 128-142) — `hall_id` сделать nullable, добавить
  `section_id TEXT NOT NULL REFERENCES restaurant_sections(id)`, `is_default INTEGER NOT
  NULL DEFAULT 0`, partial unique index `UNIQUE(restaurant_id) WHERE is_default=1`. Отдельно
  проверить, не ломает ли nullable `hall_id` существующий `UNIQUE(hall_id, name)` (NULL в
  SQLite не считается равным NULL — задвоение имени стола без hall в рамках одной секции
  технически возможно, явно решить, нужен ли отдельный constraint на уровне `(section_id,
  name)`).
- `sales_points` (POS-85 baseline): добавить `default_table_id TEXT NOT NULL REFERENCES
  tables(id)`, убрать `cash_printer_id`.
- `restaurant_sections` (POS-85 baseline): добавить `is_default INTEGER NOT NULL DEFAULT 0`,
  partial unique index `UNIQUE(restaurant_id) WHERE mode='hall_section' AND is_default=1`.
- `mastersync/service.go`: `normalizeTable`/`validateTable`/`UpsertMasterTable` (стрим
  `floor`) и новые `UpsertMasterSalesPoint`/`UpsertMasterRestaurantSection` (новые стримы)
  обновить под новые поля; добавить новые case-ветки `MasterDataStreamSalesPoints`,
  `MasterDataStreamRestaurantSections` в `applyStream`.
- `infra/sqlite` schema verification (`RequiredSchema()`, см. паттерн POS-85
  `print_routing_schema_test.go`) — обновить под новые колонки/индексы; это отдельный
  обязательный пункт в "Тесты" ниже.

## Seed/smoke

`seed-dev-system.py` и pos-backend smoke обязаны обновиться в этом же изменении (fallback
больше нет — без этого существующий печатный smoke сломается):
- точка продаж создаётся через новый Cloud CRUD (бутстрэп-стол ресторана подставится в
  `default_table_id` автоматически, если не указан явно);
- открыть тестовую кассовую смену с `sales_point_id`;
- создать print_routes (check_nonfiscal → точка продаж; precheck/ticket → бутстрэп-секция
  или любая другая, на которой стоит стол заказа) через Edge API;
- проверить полный цикл: issue precheck (auto-print) → payment → check_nonfiscal+ticket
  auto-print → `print_confirmed_at` проставлен.
- проверить, что существующий smoke/seed, ранее полагавшийся на `__counter__`/
  `GetSystemTable` для counter-заказов без table-mode, теперь работает через
  `salesPoint.DefaultTableID` — обновить соответствующие фикстуры.

## Тесты (обязательные)

- routing mapping + DB-триггер (отказ на несовместимый document_type/scope_type/section.mode);
- per-printer FIFO claim (две jobs на один printer_id не processing одновременно, порядок by
  created_at);
- target lifecycle (retry одного target не трогает другие; job-level retry пересобирает
  targets из текущих print_routes);
- job status aggregation (required vs non-required target);
- `print_confirmed_at` стампится worker'ом, не HTTP-хендлером;
- cancel-unconfirmed: refund + ticket void + order soft-cancel + три отдельных события, RBAC
  и manager PIN;
- cash-session-open: отказ без sales_point_id, отказ на чужой/неактивной точке продаж;
- стол обязан иметь section_id (CHECK/validation на создание/обновление), секция
  `mode='hall_section'` не активируется без столов;
- защита от деактивации: `is_default` стол/секция, последняя активная точка продаж,
  стол/секция с `default_table_id` точки продаж;
- создание точки продаж без явного стола подставляет бутстрэп-стол ресторана;
- заказ без явно выбранного стола резолвит `salesPoint.DefaultTableID` (новый путь),
  `GetSystemTable`/`EnsureSystemFloor` удалены и нигде не вызываются;
- cloud-backend: CRUD + mastersync streams `sales_points`/`restaurant_sections`, расширенный
  `tables`/`floor` стрим;
- Edge startup schema verification (`RequiredSchema()`) обновлена под новые
  колонки/индексы `tables.section_id`/`is_default`, `sales_points.default_table_id`,
  `restaurant_sections.is_default`, `print_jobs.scope_id`, `checks.print_confirmed_at`,
  `ticket_units.status`, `orders.cancelled_at` — без этого startup verification не поймает
  рассинхрон схемы при будущих изменениях (см. AGENTS.md про обязательную schema
  verification после migrations).

## Вне scope этой итерации

- Offline-флоу: когда лицензия `cloud-subscription` выключена и Edge работает полностью
  автономно, Edge сам локально создаёт себе служебную hall-секцию, служебный стол по
  умолчанию И точку продаж с этим столом в качестве `default_table_id` (Cloud недоступен,
  значит делать это вручную через Cloud-бэкофис, как в штатном режиме, невозможно), а
  дальше открывает модалку настройки имени точки продаж и назначения обнаруженных/
  добавленных принтеров. Зависит от модели маршрутизации и "стол обязан принадлежать
  секции" из `POS-86`. Заведена отдельным линкованным Plane work item (`POS-101`),
  намеренно вне цикла альфа-подготовки.
- cloud-ui-g экран управления точками продаж/секциями — отдельный follow-up Plane work item
  после cloud-backend части `POS-86`.
- POS UI экрана оплаты (спиннер/retry/delete modal) — отдельная будущая работа, не заведена
  как задача в этом прогоне, см. напоминание в `CURRENT-FUNCTIONAL-STATE.md`.
- `POS-87` (Edge override audit → Cloud projection), `POS-88` (требует переформулировки
  после смены модели владения — теперь это только print_routes/targets UI, не CRUD
  точек/секций), фискальный адаптер, checker/redemption flow — как и раньше.

Документация в этом же изменении: `EDGE-PRINT-ROUTING-SPEC.md` (переписать под эту модель),
`POS-BACKEND-SPEC.md`, `CLOUD-BACKEND-SPEC.md` (новые stream/CRUD), `POS-DATA-AND-MIGRATIONS.md`
(новые колонки/таблицы, version bump), `RECEIPT-PRINT-SPEC.md` (смена точек auto-enqueue),
`CURRENT-FUNCTIONAL-STATE.md`.

Проверки: `cd pos-backend && go mod tidy && go test ./...`; `cd cloud-backend && go mod tidy
&& go test ./...`; seed/smoke; `git diff --check`. cloud-ui-g/pos-ui-g не трогаются — их
build-проверки не требуются в этой итерации.

Оставь итоговый Plane comment по шаблону runbook, переведи POS-86 в Review только после
реальной реализации и прохождения тестов. Не Done.
```

### Итерация 8h — `POS-87`

```text
Используй универсальный промпт для POS-87 после POS-86.

Фокус: Edge -> Cloud sync для printer override audit и Cloud effective read model.
Edge-side изменение схемы печати применяется локально сразу, пишет audit и outbox
event. Cloud принимает событие как факт/проекцию, не как proposal, хранит видимость
effective routes/override audit и отдает bounded read model для оператора.

Не менять локальный routing алгоритм сверх контракта POS-86. Не добавлять POS UI.

Проверки: cd pos-backend && go mod tidy && go test ./...; cd cloud-backend &&
go mod tidy && go test ./...; git diff --check.
Документация: sync ownership, CLOUD-BACKEND-SPEC.md, EDGE-PRINT-ROUTING-SPEC.md.
```

### Итерация 8i — `POS-88`

```text
Используй универсальный промпт для POS-88 после POS-86.

Фокус: POS UI settings для физических принтеров, точек продаж, секций, назначений
и очереди targets. Все пользовательские строки через vue-i18n. Backend остается
authoritative; UI visibility не является security boundary.

Нужны состояния loading/empty/error, safe error banner, retry отдельного target,
диагностика последней ошибки и ручной checklist для физического принтера.

Проверки: cd pos-ui-g && npm install && npm run build; при возможности Playwright
smoke settings flow; git diff --check.
Документация: POS-UI-SPEC.md, EDGE-PRINT-ROUTING-SPEC.md,
CURRENT-FUNCTIONAL-STATE.md.
```

### Итерация 8j — `POS-89`

```text
Используй универсальный промпт для POS-89 после POS-86, POS-87 и POS-88.

Фокус: exhibition smoke sales point + section printer routing. Smoke должен
создать/проверить точку продаж с cash printer, секцию зала с precheck printer,
кухонную секцию с kitchen_service printer, выполнить bounded sale/precheck/print
flow и подтвердить target-level statuses.

Если физический принтер недоступен, не заявлять hardware acceptance: оставить
manual-validation checklist с host/port/model/CPL и ожидаемыми документами.

Проверки: профильные backend/UI checks, seed/smoke, git diff --check.
Документация: CURRENT-FUNCTIONAL-STATE.md и go/no-go evidence.
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

## Wave 6. Внешние hardware-адаптеры (`windows-printers`)

Зафиксировано 2026-07-01 по итогам физической проверки POS-86 на реальном USB-принтере
(Xprinter XP-365B, Windows 11): текущий формат `address="\\.\USB001"` не работает на
современной Windows, а Cloud API уже сознательно не принимает `address`/`port` для
`type=usb` (POS-82) — то есть архитектура давно предполагала, что физический адрес
USB-принтера не может быть Cloud-owned, но механизма обнаружения и Edge-side управления
им не было. Решение — ADR-018 и `docs/backend/EDGE-HARDWARE-ADAPTER-PROTOCOL.md`:
низкоуровневая работа с принтерами на Windows (USB-обнаружение, сетевой скан портов
9100/6001/custom, управление Windows print queue, отправка байт на устройство) выносится
в отдельный exe-адаптер `windows-printers`, который `pos-edge.exe` запускает и
супервизирует как child-процесс по generic protocol channel (см. документ — транспорт,
кадрирование сообщений, discovery/print/queue контракты).

Эта волна декомпозирована на 5 итераций с зависимостями `A → {B, C}`, `C → D`, `C → E`.
Перед стартом каждой итерации завести отдельный Plane work item (`POS-N` неизвестен на
момент написания этого раздела — создаётся в целевом окружении кодогенерации) и
перечитать оба source-of-truth документа целиком, а не только этот промпт.

### Итерация 6a — Protocol host + supervisor в `pos-edge.exe` (без Windows-специфики)

```text
Реализуй Edge-side часть hardware adapter protocol из docs/backend/EDGE-HARDWARE-ADAPTER-PROTOCOL.md
и ADR-018 в репозитории mh-pos, модуль pos-backend.

Прочитай сначала оба документа целиком, затем docs/backend/RECEIPT-PRINT-SPEC.md и
docs/backend/EDGE-PRINT-ROUTING-SPEC.md (разделы про адаптер), затем фактический код
pos-backend/internal/pos/app/print и shared/platform/receipt/escpos.

Фокус: транспортно-независимый protocol client/host внутри pos-edge.exe:
- envelope (v/id/type/payload), длина-префикс + JSON поверх net.Conn-интерфейса
  (реализация транспорта — named pipe на Windows, loopback TCP как fallback; для тестов
  использовать net.Pipe() или in-memory transport, не поднимать реальные ОС-примитивы
  в unit-тестах);
- process supervisor: запуск child-процесса адаптера по конфигурируемому пути exe,
  handshake (adapter.hello), health.ping/pong с таймаутом, restart-with-backoff при
  падении/зависании, graceful shutdown (adapter.shutdown/adapter.bye), учет
  adapter_kind/instance/capabilities;
- typed client API поверх протокола: DiscoverDevices(ctx, kind, params) DiscoveredDevice[],
  SendPrint(ctx, kind, bindingRef, payload) error, EnsureQueue(ctx, kind, bindingRef) error;
  все методы возвращают safe error_code при недоступном/незапущенном адаптере, не panic;
- НЕ реализуй сам windows-printers.exe и НЕ пиши реальный Windows syscall-код — это
  отдельная итерация 6b. Для этой итерации используй mock/fake адаптер (отдельный тестовый
  Go-процесс или in-process fake, реализующий протокол) для end-to-end проверки supervisor/
  client логики.

Обязательны unit/integration tests: handshake, heartbeat timeout → offline, restart backoff,
discover/print/queue round-trip через fake-адаптер, graceful shutdown.

Проверки: cd pos-backend && go mod tidy && go test ./...; git diff --check.
Документация: обнови docs/backend/EDGE-HARDWARE-ADAPTER-PROTOCOL.md, если реализация
потребовала уточнить протокол (не меняй архитектурные решения ADR-018 без нового ADR).
Оставь итоговый Plane comment, переведи задачу в Review. Не Done.
```

### Итерация 6b — `windows-printers` адаптер (exe, Windows-специфика)

```text
Реализуй windows-printers/ как отдельный Go-модуль/exe в репозитории mh-pos.

Начинай только после итерации 6a (protocol host/client в pos-edge.exe существует и
протестирован через fake-адаптер). Прочитай docs/backend/EDGE-HARDWARE-ADAPTER-PROTOCOL.md
целиком и находки физической проверки POS-86 (RECEIPT-PRINT-SPEC.md, раздел про
\\?\USB#VID_...#{28d78fad-5a12-11d1-ae5b-0000f803a8c2} device interface path и
Add-PrinterDriver/Add-Printer как способ активировать USBPRINT-класс устройства).

Фокус:
- реализация протокола (client-конец теперь на стороне адаптера: connect к pipe/loopback,
  adapter.hello, health.pong, обработка discover.request/print.send/queue.ensure);
- USB discovery: перечисление USBPRINT-класса устройств (SetupAPI/PnP через
  golang.org/x/sys/windows или эквивалент), построение стабильного binding_ref
  (usb:<vid>:<pid>:<serial>), определение queue_state (unconfigured/queue_ready) по
  наличию Windows print queue на соответствующем dynamic-порте;
- queue.ensure: идемпотентная установка драйвера "Generic / Text Only" (если не
  установлен) и Add-Printer на нужный порт — воспроизвести вручную проверенную
  последовательность из POS-86 hardware-сессии программно (не через shell-обёртку
  PowerShell-скриптов, а через нативные Windows API/пакеты, если это разумно;
  PowerShell-фallback допустим только как явно задокументированное решение с обоснованием);
- сетевой discovery: TCP connect-scan с коротким таймаутом на портах 9100, 6001 и
  произвольном списке из discover.request.network_ports, по подсети/списку целей из
  network_targets; НЕ делать continuous background scan — только по явному запросу;
- print.send: адаптер не принимает решений о конфигурации соединения (см.
  EDGE-HARDWARE-ADAPTER-PROTOCOL.md §1, принцип "адаптер не решает"). Для transport=usb —
  резолвить переданный binding_ref в реальный device interface path (\\?\USB#...). Для
  transport=tcp — использовать ТОЛЬКО address/port, явно переданные core в этом конкретном
  вызове (они приходят из Cloud-owned receipt_printers.address/port), без собственного
  кеша/выбора/угадывания порта адаптером. В обоих случаях — записать payload_base64 как raw
  bytes, вернуть print.result с safe error_code при ошибке;
- health.pong на health.ping с разумным таймаутом ответа.

Не включай: бизнес-логику print_jobs/routing (остаётся в pos-edge.exe); рендеринг ESC/POS
(адаптер получает уже готовые байты); UI.

Обязательны unit tests для парсинга protocol messages и binding_ref detection logic с
фикстурами (без реального USB-устройства). Ручной hardware checklist для итогового Plane
comment: подключить реальный USB ESC/POS-принтер, вызвать discover, проверить binding_ref/
queue_state, вызвать queue.ensure на новом устройстве, отправить print.send и подтвердить
физический выход документа (аналогично трём проверкам, сделанным вручную в POS-86 hardware
сессии 2026-07-01).

Проверки: cd windows-printers && go build ./... (Windows target); go test ./...; git diff --check.
Документация: обнови EDGE-HARDWARE-ADAPTER-PROTOCOL.md, если протокол потребовал уточнений;
задокументируй сборку/установку адаптера как отдельного компонента (аналогично
docs/deployment/*, если такой файл заводится в этой итерации).
Оставь итоговый Plane comment с manual hardware checklist, переведи в Review (label
manual-validation). Не Done.
```

### Итерация 6c — Edge-local физическая привязка принтера + push обнаружения в Cloud

```text
Реализуй Edge-local printer_physical_bindings и Edge → Cloud push обнаруженных устройств
из docs/backend/EDGE-HARDWARE-ADAPTER-PROTOCOL.md, разделы 6 и 7.

Начинай только после итерации 6a (protocol client в pos-edge.exe готов; адаптер для теста
может быть fake из 6a, реальный windows-printers из 6b не обязателен для этой итерации,
но должен быть план интеграции).

Фокус:
- managed baseline: таблица printer_physical_bindings (printer_id, adapter_kind,
  binding_ref, transport, display_label, bound_at, bound_by_employee_id, UNIQUE(printer_id));
  используется ТОЛЬКО для receipt_printers.type='usb' (см. EDGE-HARDWARE-ADAPTER-PROTOCOL.md
  §6) — для type='tcp' эта таблица не заполняется и не читается;
- print worker (pos-backend/internal/pos/app/print) резолвит физическое назначение по
  transport, а не по наличию/отсутствию строки в одной общей таблице: type=tcp — всегда
  address/port из receipt_printers (текущий escpos.WriteRaw-путь не удалять; если печать в
  этой итерации переводится на адаптер — передавать эти address/port явно в print.send,
  адаптер их не хранит и не выбирает сам, см. §1/§5); type=usb — обязателен
  printer_physical_bindings, иначе job уходит в failed с safe error_code, без попытки
  угадать адрес. Явно без silent fallback между режимами внутри одной попытки (§6, пункт 3);
- Edge HTTP API (RBAC pos.print_routing.manage): GET .../adapters/{kind}/discovered,
  POST/DELETE .../printers/{id}/binding — как описано в §6;
- Edge → Cloud push: новое доменное событие HardwareDeviceDiscovered в существующий
  outbox/exchange механизм (тот же, которым Edge уже отправляет доменные события в Cloud),
  триггерится после успешного discover.request (ручной запрос через API из этой же
  итерации, не автоматический периодический polling, если не обосновано иначе).

Не включай: реализацию windows-printers.exe (6b); Cloud-side обработку события и
Cloud UI (6d); Edge settings UI выбора устройства (6e, только HTTP API в этой итерации).

Обязательны unit/integration tests: binding CRUD, print worker resolution с/без binding,
fallback path не ломается для существующих TCP-принтеров, outbox event формируется
корректно. Migration backup/schema verification по стандартной politике.

Проверки: cd pos-backend && go mod tidy && go test ./...; git diff --check.
Документация: EDGE-HARDWARE-ADAPTER-PROTOCOL.md (если потребовались уточнения),
EDGE-PRINT-ROUTING-SPEC.md, POS-DATA-AND-MIGRATIONS.md, CURRENT-FUNCTIONAL-STATE.md.
Оставь итоговый Plane comment, переведи в Review. Не Done.
```

### Итерация 6d — Cloud-side ingestion обнаруженных устройств + обновление Printers UI

```text
Реализуй Cloud-side приём HardwareDeviceDiscovered и обновление printer creation flow
из docs/backend/EDGE-HARDWARE-ADAPTER-PROTOCOL.md §7-8 и docs/backend/CLOUD-BACKEND-SPEC.md
(раздел Printers, обновлённый указатель на ADR-018).

Начинай только после итерации 6c (Edge отправляет HardwareDeviceDiscovered в существующий
exchange/outbox ingestion пайплайн).

Фокус:
- новая таблица/проекция printer_discovery_candidates (node_device_id, adapter_kind,
  binding_ref, transport, display_vendor, display_model, display_label, first_seen_at,
  last_seen_at), upsert по (node_device_id, adapter_kind, binding_ref) при получении события;
  НЕ часть printers mastersync stream, не master data;
- GET-эндпоинт для Cloud UI, отдающий текущие кандидаты по restaurant_id (через node →
  restaurant связь), сгруппированные по Edge-узлу, с признаком stale (TTL — выбери и
  обоснуй конкретное значение);
- обновление POS-83 Cloud UI формы создания/редактирования принтера (cloud-ui-g):
  вместо ручного type/address/port — список обнаруженных устройств для восстановления
  выбора; поля, которые реально задаёт оператор — name, cpl, paper_cut_type, отступы
  (если поддержаны текущим RenderOptions; если нет — зафиксировать это как gap, не
  придумывать несуществующее поле); ручной advanced-режим для tcp-принтеров, не увиденных
  сканом, оставить как escape hatch.

Не включай: сам протокол/адаптер (6a/6b); Edge-side binding API (6c, уже готово, только
потребляется как источник данных для показанного оператору списка); Edge settings UI (6e).

Обязательны backend tests (ingestion upsert, TTL/stale marking) и cloud-ui-g tests
(форма создания принтера с выбором кандидата, zod-схема, i18n). Все строки через i18n.

Проверки: cd cloud-backend && go mod tidy && go test ./...; cd cloud-ui-g && npm run test -- --run && npm run build; git diff --check.
Документация: CLOUD-BACKEND-SPEC.md, EDGE-HARDWARE-ADAPTER-PROTOCOL.md (если уточнён TTL/поля),
CURRENT-FUNCTIONAL-STATE.md.
Оставь итоговый Plane comment, переведи в Review. Не Done.
```

### Итерация 6e — Edge settings UI: выбор/смена физического принтера под логический

```text
Реализуй pos-ui-g экран управления физической привязкой принтера поверх Edge HTTP API
из итерации 6c (GET .../adapters/{kind}/discovered, POST/DELETE .../printers/{id}/binding).

Начинай только после 6c (API готово) и, по возможности, 6b (реальный windows-printers для
осмысленной ручной проверки; если 6b ещё не готов, используй fake-адаптер и явно пометь
итог как manual-validation с hardware checklist на потом).

Фокус: в существующем Edge print routing settings UI (POS-88, если уже реализован к этому
моменту) или отдельный раздел — для каждого логического принтера, синхронизированного из
Cloud, показать текущую физическую привязку (если есть) и кнопку "обнаружить устройства" →
список DiscoveredDevice → выбор → создание/смена привязки, включая создание новой Windows
print queue для ещё не сконфигурированного USB-устройства (queue.ensure через 6b) или выбор
уже готового. Оператор не видит binding_ref/device path напрямую, только display_label.

Все строки через i18n. RBAC pos.print_routing.manage на действия, pos.print_routing.view на
просмотр. Safe error banner без raw payload/device path в UI при ошибках адаптера.

Обязательны component/helper tests. Playwright-проверка через MCP, если доступна реальная
или fake-адаптер среда.

Проверки: cd pos-ui-g && npm run test -- --run && npm run build; git diff --check.
Документация: docs/ui/*, CURRENT-FUNCTIONAL-STATE.md.
Оставь итоговый Plane comment с manual hardware checklist, переведи в Review (label
manual-validation, если проверялось без реального адаптера/принтера). Не Done.
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
