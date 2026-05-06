---
description: State ownership + API discipline (loading/error/success, retry, contracts) для React
globs:
  - *.{ts,tsx,js,jsx}"
alwaysApply: true
---

# STATE & API CONTROL

## State ownership

- **local state** (useState/useReducer) → по умолчанию.
- **context** → в основном для DI / cross-cutting (theme, auth, i18n), а не для бизнес‑state.
- **global state** (redux/zustand/mobx/rtk-query и т.п.) → только при явном обосновании (несколько страниц, сложная синхронизация, кэш).

## Data fetching / API

- Все HTTP вызовы — только в data/service слое (API client).
- Компоненты не должны знать про URL/headers/token refresh — только про «use-case»/метод сервиса.
- DTO → маппинг в UI model (и обратно) должен быть явным.

## Async state machine (обязательное)

Для любого async сценария должны быть учтены состояния:

- loading
- success
- error
- empty (если применимо)

Опционально:

- retry (для transient ошибок)
- partial success (если страница состоит из независимых виджетов)

## ❌ Запрещено

- unhandled promises / «потерянные» async ошибки
- silent catch
- обновление state после unmount без защиты
- гонки запросов без отмены/дедупликации (если сценарий допускает)
