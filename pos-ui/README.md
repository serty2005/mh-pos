# MyHoReCa POS UI

`pos-ui` - Vue 3 + TypeScript + Quasar интерфейс для модели `provisioning/pairing -> login -> pos -> lock/logout`.

## Запуск

```powershell
cd pos-ui
npm install
npm run dev
```

## Скрипты

Реализовано сейчас:

- `npm run dev` - Vite dev server.
- `npm run build` - `vue-tsc --noEmit` и production build.
- `npm run preview` - локальный preview production build.
- `npm run test` - Vitest unit tests.
- `npm run test:e2e` - Playwright tests; для backend-backed сценариев нужен `POS_E2E_BOOTSTRAP_JSON`.

По умолчанию UI ходит в `http://localhost:8080/api/v1`. Для другого backend:

```powershell
$env:VITE_POS_API_BASE="http://localhost:8080/api/v1"
npm run dev
```

## Zero-to-Cashier Quickstart

Реализовано сейчас: `/pair` поддерживает два режима production onboarding без dev bootstrap.

- Cloud approval: экран показывает `node_device_id`, `cloud_url`, статус регистрации и polling; после Cloud assign Edge скачивает snapshot и UI автоматически переходит на `/login`.
- License code: оператор вводит code, UI вызывает `POST /api/v1/system/provisioning/pair-via-license`; после успешного snapshot apply UI переходит на `/login`.

Локальная проверка:

```powershell
.\scripts\zero-to-cashier-option-a.ps1
.\scripts\zero-to-cashier-option-b.ps1
```

После provisioning войди на `/login` с cashier PIN `1111`.

## Локальный E2E Prototype Quickstart

Реализовано сейчас: UI проходит основной cashier flow через настоящий POS Edge backend и production-way Cloud -> Edge bootstrap.

1. Запусти `cloud-backend` и `pos-backend`.
2. Из корня репозитория выполни `.\scripts\bootstrap-production-way.ps1`.
3. Открой `http://localhost:5173`.
4. На `/pair` используй Cloud provisioning/license code, если он был выдан; при Cloud-approved assignment Edge уже paired.
5. На `/login` используй cashier PIN `1111`.
6. Для cancel unpaid precheck в manager override введи manager PIN `2222`; backend сам определяет менеджера по PIN и проверяет право отмены.

Playwright specs читают `POS_E2E_BOOTSTRAP_JSON` как JSON-строку или как путь к JSON-файлу. Для Docker devbox canonical path:

```bash
POS_E2E_BOOTSTRAP_JSON=/workspace/myhoreca-pos/.e2e/bootstrap.json
```

Пример формы лежит в `.e2e/bootstrap.example.json`. Реальный `.e2e/bootstrap.json` создается локальным bootstrap/smoke script и не коммитится.

Docker devbox flow:

```bash
docker compose -f docker-compose.local.yml --profile devbox build devbox
docker compose -f docker-compose.local.yml up --build -d cloud-postgres license-api cloud-api pos-edge
docker compose -f docker-compose.local.yml --profile devbox up -d devbox
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd pos-ui && npm install'
docker compose -f docker-compose.local.yml exec devbox bash -lc 'mkdir -p .e2e && python3 scripts/run-stack-smoke.py --suite cloud_to_edge_masterdata --cloud-base http://cloud-api:8090 --pos-base http://pos-edge:8080 --license-base http://license-api:8095 --output .e2e/bootstrap.json'
```

Внутри devbox backend доступен через Docker service DNS: `http://pos-edge:8080/api/v1`. Поэтому devbox задает `VITE_POS_API_BASE=http://pos-edge:8080/api/v1` и `POS_E2E_API_BASE=http://pos-edge:8080/api/v1`. Для UI, открытого host browser, используй `http://localhost:8080/api/v1`.

E2E smoke:

```bash
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd pos-ui && npm run dev'
docker compose -f docker-compose.local.yml exec devbox bash -lc 'cd pos-ui && npx playwright test e2e/waiter-mobile-flow.spec.ts e2e/kitchen-flow.spec.ts'
```

Ручной UI flow:

```text
pair -> login -> open personal shift -> open cash shift -> select hall/table -> create order -> add lines/services/modifiers -> change quantity -> void line -> issue precheck -> cancel precheck -> issue precheck again -> pay -> final check -> reprint -> full check cancellation or refund from closed-order activity -> close cash shift -> close personal shift -> lock/logout
```

## Локальный E2E Prototype: получить pairing code и войти в POS UI

Реализовано сейчас: UI `/pair` и `/login` используют реальные POS backend endpoints.

1. Запусти POS backend:

```powershell
cd pos-backend
$env:POS_CLOUD_SYNC_URL="http://localhost:8090"
go run ./cmd/pos-edge
```

2. Из корня репозитория получи учетные данные:

```powershell
$demo = .\scripts\bootstrap-production-way.ps1
```

3. Запусти UI:

```powershell
cd pos-ui
npm install
npm run dev
```

4. Открой `http://localhost:5173/pair`, введи `$demo.pairing_code`, если требуется license flow, затем войди на `/login` с cashier PIN `1111`. UI сохраняет paired `node_device_id` и `restaurant_id`, возвращенные `GET /api/v1/system/pairing-status`, затем читает Cloud-authored hall/table/menu data из Edge read endpoints.

Из корня репозитория можно проверить Cloud replay и локальный sync:

```powershell
.\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
$login = Invoke-RestMethod -Method Post http://localhost:8080/api/v1/auth/pin-login -ContentType "application/json" -Body (@{
  node_device_id = $demo.node_device_id
  client_device_id = "dev-ui-readme-client"
  pin = "2222"
} | ConvertTo-Json)
$headers = @{
  "X-Node-Device-ID" = $demo.node_device_id
  "X-Client-Device-ID" = "dev-ui-readme-client"
  "X-Session-ID" = $login.session.id
  "X-Actor-Employee-ID" = $login.actor.employee_id
}
Invoke-RestMethod http://localhost:8080/api/v1/sync/status -Headers $headers
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=10 -Headers $headers
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=10 -Headers $headers
```

Вне текущего объема: KDS runtime, modifier/service/tip ledger scopes, inventory consumption, PSP и fiscalization. `/pos/waiter` реализован как order/precheck mobile readiness/runtime surface без payment/refund/cash drawer authority по умолчанию, а `/pos/kitchen` остается readiness-only до появления KDS backend contracts.

## Что реализовано

- `/pair` показывает Cloud approval status, license code form и после `paired` ведет на `/login`.
- `/login` вызывает реальный `POST /api/v1/auth/pin-login`.
- `/lock` вызывает реальный `POST /api/v1/auth/logout`, очищает локальную session и требует новый PIN.
- `/pos` реализует POS Terminal Core для одного кассира на одном Primary Edge Node:
  - показывает сотрудника, ресторан, node/client device, backend session, личную смену, кассовую смену и pairing/node status;
  - показывает текущую личную смену и кассовую смену;
  - открывает личную смену и кассовую смену;
  - закрывает кассовую смену и показывает безопасное действие закрытия личной смены;
  - выбирает зал и стол;
  - находит активный заказ по столу через backend;
  - создает заказ на выбранном столе;
  - показывает позиции заказа и backend totals;
  - добавляет позиции из меню;
  - показывает services отдельной секцией и добавляет service menu items;
  - открывает выбор modifiers для menu items с modifier groups и отправляет selected modifiers в backend;
  - меняет количество и void-ит позиции;
  - выпускает пречек;
  - отменяет unpaid issued пречек через manager override;
  - принимает оплату наличными и trusted manual card;
  - показывает финальный чек после полной оплаты;
  - перепечатывает precheck/check по backend immutable snapshot;
  - записывает full check cancellation/refund через ledger endpoints из closed-order activity;
  - оставляет payment-level refund как explicit compatibility fallback.

Server state хранится только через `@tanstack/vue-query`. Frontend не является source of truth и не принимает бизнес-решения по заказу, пречеку, оплате или чеку.

## Поток identity

- Production pairing использует Cloud approval или License Server code; legacy MVP pairing code имеет формат `MHPOS:<restaurant_id>:<node_device_id>`.
- `node_device_id` не генерируется frontend-клиентом; он хранится в POS Edge backend и обозначает Edge Node backend.
- Каждый browser/tablet client генерирует свой `client_device_id` через `crypto.randomUUID()` и хранит его в `localStorage`.
- Backend auto-registers новый `client_device_id` при PIN login.
- Lock всегда вызывает backend logout.

## Error handling

Реализовано сейчас:

- `src/shared/api.ts` является единым API client и различает `401/403/404/409/429/5xx/network/timeout`.
- Backend error envelope нормализуется в `ApiError` с `code`, `messageKey`, `category` и `correlationId`.
- Critical/business errors показываются через global Quasar dialog из `src/stores/errorDialog.ts`.
- Dialog/inline banners показывают безопасный support code: `correlation_id`, если он есть, иначе стабильный `error_code` вроде `INVALID_RESPONSE`, `NETWORK_ERROR` или backend code.
- Все user-facing ошибки идут через `vue-i18n` keys.
- `401` очищает local session и ведет к login flow.
- `403` показывает "Недостаточно прав" и не выполняет logout.
- Network/timeout сообщает, что POS Edge backend недоступен, но не удаляет `client_device_id`.
- TanStack mutations не используют auto-retry для write/financial commands.

## Используемые Backend Endpoints

- `GET /api/v1/system/pairing-status`
- `POST /api/v1/system/pair`
- `GET /api/v1/system/provisioning-status`
- `POST /api/v1/system/provisioning/register-cloud`
- `POST /api/v1/system/provisioning/pair-via-license`
- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`
- `GET /api/v1/employee-shifts/current`
- `POST /api/v1/employee-shifts/open`
- `POST /api/v1/employee-shifts/{id}/close`
- `GET /api/v1/cash-shifts/current`
- `POST /api/v1/cash-shifts/open`
- `POST /api/v1/cash-shifts/{id}/close`
- `POST /api/v1/cash-drawer-events`
- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/menu/items`
- `GET /api/v1/orders/current?table_id=...`
- `GET /api/v1/orders/active?hall_id=...`
- `GET /api/v1/orders/closed`
- `POST /api/v1/orders`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/modifiers`
- `PATCH /api/v1/orders/{id}/lines/{line_id}/details`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/orders/{id}/prechecks`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/reprint`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/checks/{id}`
- `POST /api/v1/checks/{id}/reprint`
- `POST /api/v1/checks/{id}/cancellations`
- `POST /api/v1/checks/{id}/refunds`
- `GET /api/v1/checks/{id}/financial-operations?limit=&offset=`
- `POST /api/v1/payments/{id}/refund` (compatibility-only)
- `GET /api/v1/sync/status`
- `GET /api/v1/sync/outbox?limit=`
- `GET /api/v1/sync/local-events?limit=&event_type=`
- `POST /api/v1/sync/outbox/retry-failed`

## Ограничения

- Waiter route `/pos/waiter` реализован только как mobile order/precheck surface по существующим POS contracts; payment/refund/cash drawer authority в нем не включена по умолчанию.
- KDS route `/pos/kitchen` является readiness-only экраном, не KDS runtime.
- Refund/cancellation pilot flow реализован для закрытых заказов: full whole-check и partial `order_line`/quantity cancellation/refund через `/checks/{id}/cancellations` и `/checks/{id}/refunds`, compatibility refund по captured payment через `/payments/{id}/refund`.
- Нет tax engine rewrite.
- Precheck/check reprint UI использует backend immutable snapshot endpoints.
- Нет backoffice.
- Trusted card payment - ручная trusted capture запись без PSP integration.
- Денежный ввод в UI показывается в основных единицах валюты, а backend получает integer minor units.

## Pricing policy selection

Статус: реализовано сейчас.

POS UI API layer содержит typed helpers для списка активных pricing policies и применения discount/surcharge policy по `pricing_policy_id`. UI не должен отправлять произвольные amount fields для обычного pilot cashier flow.
