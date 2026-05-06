# POS-UI Vue 3 + Quasar MVP Specification

Статус: Approved MVP Specification
Назначение: Рабочая спецификация для реализации MVP frontend-части POS/RMS платформы
Целевой пакет: `pos-ui`
Frontend stack: Vue 3 + TypeScript + Quasar + TanStack Query
Backend: локальный Edge Go Backend (`pos-backend`)
Архитектурная модель: Edge-first / Offline-aware / Backend-as-source-of-truth

---

## 1. Контекст и ключевые решения

`pos-ui` реализуется как отдельный frontend-пакет внутри монорепозитория POS/RMS платформы.
Frontend работает как локальный web-интерфейс к Edge Go Backend и обслуживает несколько режимов:
* POS Terminal;
* Waiter UI;
* KDS (Kitchen Display System);
* Manager UI;
* Diagnostics;
* Settings;
* Mode Selector.

Текущий backend (до MVP UI) уже содержит foundation для runtime flow:
`Order -> Precheck -> Payment -> Check`

**Backend Gaps (Должно быть сделано в Фазе 0 до старта UI):**
1. Авторизация сотрудника по PIN.
2. Столы, ресторанные залы и схемы залов.
3. Несколько залов и KDS stations.
4. Полное редактирование order lines (изменение количества, удаление позиции).
5. Аудит команд от имени авторизованного сотрудника (actor metadata).

---

## 2. Главная архитектурная позиция

**Frontend не является источником истины (Source of Truth).**

**Source of truth:**
* Edge Go Backend;
* локальная SQLite database.

**Frontend отвечает за:**
* отображение состояния (с кешированием и поллингом через Vue Query);
* PIN-auth UI flow и Lock/Unlock flow;
* вызов backend command endpoints;
* маршрутизацию и touch-friendly UX (через Quasar);
* i18n адаптацию.

**Frontend не должен:**
* напрямую работать с SQLite или оборудованием (ФР, банковские терминалы);
* принимать финансовые решения или считать чек оплаченным без подтверждения backend;
* хранить PIN, хеши PIN или критичные права в localStorage.

---

## 3. Frontend Stack & Tooling

* **Framework:** Vue 3 (Composition API, `<script setup>`)
* **Language:** TypeScript (strict mode)
* **UI Framework:** Quasar Framework (использовать встроенные Quasar CSS Utility Classes, **не использовать Tailwind**)
* **Routing:** Vue Router
* **Local State:** Pinia (UI состояние, сессия, настройки устройства)
* **Server State & Data Fetching:** `@tanstack/vue-query` (Обязательно для всех GET/POST запросов, поддержка caching, invalidation и polling)
* **Real-time (MVP):** Short Polling через TanStack Query (интервал 3-5 сек) для KDS и статусов столов.
* **Validation:** Zod
* **I18n:** `vue-i18n`
* **Build Tool:** Vite / Quasar CLI

---

## 4. Multi-mode routing

`pos-ui` — это multi-mode SPA.

```text
/                 -> Mode Selector
/login            -> PIN Login
/lock             -> Lock Screen
/pos              -> POS Terminal
/waiter           -> Waiter UI
/kds              -> Kitchen Display System
/manager          -> Manager UI
/diagnostics      -> Edge Diagnostics
/settings         -> Local Node Settings
```

Каждый режим имеет собственный layout, набор разрешенных действий и RBAC guards.

---

## 5. Идентификация устройства и Auth MVP

### 5.1 Device ID
При первом запуске Frontend генерирует **UUID v4** в качестве `device_id` и персистентно хранит его в `localStorage`. Этот `device_id` передается во всех API-запросах.

### 5.2 PIN Login Flow
1. Пользователь вводит PIN на экране Login.
2. UI отправляет `POST /api/v1/auth/pin-login` (payload: `device_id`, `pin`).
3. Backend возвращает сессию, данные сотрудника (actor context) и список permission'ов.
4. UI сохраняет `session_id` и `permissions` в памяти (Pinia) и переходит к Mode Selector.
5. При бездействии или нажатии "Lock", сессия очищается из активного контекста, показывается Lock Screen.

### 5.3 Actor Metadata
Все backend-команды вызываются с метаданными актора:
```json
{
  "device_id": "dev_uuid",
  "actor_employee_id": "emp_123",
  "session_id": "sess_abc"
}
```

---

## 6. RBAC и Manager Override

Frontend скрывает элементы интерфейса исключительно ради UX. Вся реальная защита (enforcement) — на бэкенде.
Права в формате: `domain.resource.action` (например, `pos.order.create`, `kds.mode.open`).

Для ограниченных действий (restricted action), например `CancelPrecheck`, UI вызывает команду с блоком `manager_override`:
```json
{
  "actor_employee_id": "emp_cashier",
  "manager_override": {
    "manager_employee_id": "emp_manager",
    "pin": "1234",
    "reason": "Mistake"
  }
}
```

---

## 7. MVP Режимы: Детализация

### 7.1 POS Terminal (`/pos`)
* Layout: адаптация от 800x600 (Compact) до 1080p.
* Функции: выбор зала/стола, создание заказа, добавление позиций, изменение количества, удаление позиций.
* Финансы: Выпуск пречека (Issue Precheck) -> Оплата (Capture Payment) -> Получение чека (Check).

### 7.2 Waiter UI (`/waiter`)
* Layout: Mobile-first (360-430px ширина), bottom actions.
* Функции: выбор стола, добавление позиций, отправка на кухню (Send to kitchen). Запрос пречека.

### 7.3 KDS - Kitchen Display System (`/kds`)
* Функции: Board layout с тикетами фильтрованными по `station_id` (hot, cold, bar).
* Действия: `Start Ticket` -> `Ready Ticket`.
* Синхронизация: Polling `/api/v1/kitchen/tickets?station_id=X` раз в 3-5 секунд.

### 7.4 Diagnostics (`/diagnostics`)
* Функции: Мониторинг здоровья Edge Node, статус `sync_outbox`, кнопки `Retry Sync`.

---

## 8. State Management: Разделение

**TanStack Query (Server State):**
Владеет данными бекенда: Halls, Tables, Menu Items, Orders, KDS Tickets, Sync Status.

**Pinia / LocalStorage (Local UI State):**
* `localStorage.deviceId`
* `localStorage.locale`
* `localStorage.defaultMode`
* `localStorage.kdsStationId`
* `Pinia`: Активная сессия, права, выбранный зал, состояние UI панелей (открыта/закрыта), черновики комментариев.

---

## 9. Структура проекта (Feature-Sliced Design Lite)

```text
pos-ui/
  src/
    app/              # Инициализация (router, pinia, query-client, i18n, layouts)
    shared/           # API клиент, UI компоненты (кнопки, инпуты), утилиты
    entities/         # Модели данных (Order, Menu, Table, Session) + Query Hooks
    features/         # Бизнес-фичи (PIN-login, Add-order-line, Capture-payment)
    modes/            # (Вместо pages) pos-terminal, waiter, kds, manager, settings
```

---

## 10. Roadmap Имплементации

* **Phase 0:** Подготовка Edge Backend (добавление PIN Auth, Halls/Tables API, KDS API, Order Edit API).
* **Phase 1:** Инициализация `pos-ui` (Quasar CLI/Vite, настройка TS, Vue Router, Vue Query, i18n).
* **Phase 2:** Auth & RBAC (PIN Screen, Lock Screen, Guard Router).
* **Phase 3:** Mode Selector & Layouts (Каркасы всех режимов).
* **Phase 4:** POS Terminal Core (Меню, Заказ, Precheck, Payment).
* **Phase 5:** Waiter UI (Mobile Layout, Выбор столов, Send to kitchen).
* **Phase 6:** KDS (Доска тикетов, Polling, Таймеры).
* **Phase 7:** Settings, Manager & Diagnostics.

---

## 11. Acceptance Criteria для UI MVP

1. Приложение стартует и не использует моки для критических путей (только вызовы к Edge backend).
2. UI корректно использует Quasar CSS утилиты (без Tailwind).
3. Приложение работает на разрешении 800x600 (POS) и мобильном (Waiter).
4. i18n используется для всех текстов (нет хардкода).
5. При первом запуске генерируется и сохраняется `device_id`.
6. Vue Query поллит актуальные статусы для столов и KDS.
7. Логика расчетов (суммы, налоги) не дублируется на фронте — UI просто рендерит ответы бекенда.
8. Аутентификация работает, PIN-коды нигде не логируются и не сохраняются.
