# MyHoReCa POS UI

`pos-ui` - Vue 3 + TypeScript + Quasar интерфейс для модели `pairing -> login -> pos -> lock/logout`.

## Запуск

```powershell
cd pos-ui
npm install
npm run dev
```

По умолчанию UI ходит в `http://localhost:8080/api/v1`. Для другого backend:

```powershell
$env:VITE_POS_API_BASE="http://localhost:8080/api/v1"
npm run dev
```

## Local E2E Prototype Quickstart

implemented now: UI проходит основной cashier flow через настоящий POS Edge backend.

1. Запусти `pos-backend` с `$env:POS_DEV_TOOLS="1"`.
2. Из корня репозитория выполни `.\scripts\bootstrap-pos-demo.ps1`.
3. Открой `http://localhost:5173`.
4. На `/pair` введи `pairing_code` из bootstrap.
5. На `/login` используй cashier PIN `1111`.
6. Для cancel unpaid precheck в manager override введи `manager_employee_id` из bootstrap и manager PIN `2222`.

Ручной UI flow:

```text
pair -> login -> open shift -> open cash session -> select hall/table -> create order -> add lines -> change quantity -> void line -> issue precheck -> cancel precheck -> issue precheck again -> pay -> final check -> close cash session -> close shift -> lock/logout
```

## Local E2E Prototype: получить pairing code и войти в POS UI

implemented now: UI `/pair` and `/login` use real POS backend endpoints.

1. Start POS backend with dev bootstrap enabled:

```powershell
cd pos-backend
$env:POS_DEV_TOOLS="1"
go run ./cmd/pos-edge
```

2. From repo root, get credentials:

```powershell
$demo = .\scripts\bootstrap-pos-demo.ps1
```

3. Start UI:

```powershell
cd pos-ui
npm install
npm run dev
```

4. Open `http://localhost:5173/pair`, enter `$demo.pairing_code`, then log in on `/login` with cashier PIN `1111`. The UI stores the paired `node_device_id` and `restaurant_id` returned by `GET /api/v1/system/pairing-status`, then reads demo hall/table/menu data from backend endpoints.

From repo root, Cloud replay and local sync checks:

```powershell
.\scripts\send-cloud-test-envelope.ps1 -RestaurantId $demo.restaurant_id -NodeDeviceId $demo.node_device_id -ReplayTwice
Invoke-RestMethod http://localhost:8080/api/v1/sync/status
Invoke-RestMethod http://localhost:8080/api/v1/sync/local-events?limit=10
Invoke-RestMethod http://localhost:8080/api/v1/sync/outbox?limit=10
```

out of scope: waiter UI, KDS, inventory, fiscalization, and production sync sender worker.

## Что Реализовано

- `/pair` вызывает реальный `POST /api/v1/system/pair`.
- `/login` вызывает реальный `POST /api/v1/auth/pin-login`.
- `/lock` вызывает реальный `POST /api/v1/auth/logout`, очищает локальную session и требует новый PIN.
- `/pos` реализует POS Terminal Core для одного кассира на одном Primary Edge Node:
  - показывает сотрудника, session, pairing/node status;
  - показывает текущую смену и кассовую сессию;
  - открывает смену и кассовую сессию;
  - закрывает кассовую сессию и показывает безопасное действие закрытия смены;
  - выбирает зал и стол;
  - находит активный заказ по столу через backend;
  - создает заказ на выбранном столе;
  - показывает позиции заказа и backend totals;
  - добавляет позиции из меню;
  - меняет количество и void-ит позиции;
  - выпускает пречек;
  - отменяет unpaid issued пречек через manager override;
  - принимает оплату наличными и trusted manual card;
  - показывает финальный чек после полной оплаты.

Server state хранится только через `@tanstack/vue-query`. Frontend не является source of truth и не принимает бизнес-решения по заказу, пречеку, оплате или чеку.

## Identity Flow

- MVP pairing code имеет временный формат `MHPOS:<restaurant_id>:<node_device_id>`.
- `node_device_id` не генерируется frontend-клиентом; он приходит из pairing payload и обозначает Edge Node backend.
- Каждый browser/tablet client генерирует свой `client_device_id` через `crypto.randomUUID()` и хранит его в `localStorage`.
- Backend auto-registers новый `client_device_id` при PIN login.
- Lock всегда вызывает backend logout.

## Используемые Backend Endpoints

- `GET /api/v1/system/pairing-status`
- `POST /api/v1/system/pair`
- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`
- `GET /api/v1/shifts/current`
- `POST /api/v1/shifts/open`
- `POST /api/v1/shifts/{id}/close`
- `GET /api/v1/cash-sessions/current`
- `POST /api/v1/cash-sessions/open`
- `POST /api/v1/cash-sessions/{id}/close`
- `GET /api/v1/halls`
- `GET /api/v1/tables`
- `GET /api/v1/menu/items`
- `GET /api/v1/orders/current?table_id=...`
- `POST /api/v1/orders`
- `GET /api/v1/orders/{id}`
- `POST /api/v1/orders/{id}/lines`
- `PATCH /api/v1/orders/{id}/lines/{line_id}`
- `POST /api/v1/orders/{id}/lines/{line_id}/void`
- `POST /api/v1/orders/{id}/precheck`
- `GET /api/v1/orders/{id}/prechecks`
- `POST /api/v1/prechecks/{id}/cancel`
- `POST /api/v1/prechecks/{id}/payments`
- `GET /api/v1/checks/{id}`

## Ограничения

- Нет waiter mode.
- Нет KDS runtime.
- Нет refund flow.
- Нет tax engine rewrite.
- Нет print/reprint UI.
- Нет backoffice.
- Trusted card payment - ручная trusted capture запись без PSP integration.
- Денежный ввод в UI показывается в основных единицах валюты, а backend получает integer minor units.
