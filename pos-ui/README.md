# MyHoReCa POS UI

`pos-ui` - минимальный Vue 3 + TypeScript shell для новой модели `pairing -> login -> pos -> lock/logout`.

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

## Identity Flow

- `/pair` вызывает реальный `POST /api/v1/system/pair`.
- MVP pairing code имеет честный временный формат `MHPOS:<restaurant_id>:<node_device_id>`.
- `node_device_id` не генерируется frontend-клиентом; он приходит из pairing payload и обозначает Edge Node backend.
- Каждый browser/tablet client генерирует свой `client_device_id` через `crypto.randomUUID()` и хранит его в `localStorage`.
- Backend auto-registers новый `client_device_id` при PIN login.
- `/login` вызывает `POST /api/v1/auth/pin-login`.
- `/lock` вызывает `POST /api/v1/auth/logout`, очищает локальный session state и требует новый PIN login.
- `/pos` читает текущую session, halls и tables только через backend API.

## Используемые Backend Endpoints

- `GET /api/v1/system/pairing-status`
- `POST /api/v1/system/pair`
- `POST /api/v1/auth/pin-login`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`
- `GET /api/v1/halls`
- `GET /api/v1/tables`

## MVP Ограничения

- Нет manager approve для планшетов.
- Нет waiter UI, KDS runtime, refunds, tax engine или sync worker.
- Pairing пока не общается с production Cloud orchestration; код содержит выданный извне `node_device_id`.
- Frontend не принимает бизнес-решения и не является source of truth.
