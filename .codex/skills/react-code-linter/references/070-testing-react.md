---
description: Тестирование (React): unit/component/e2e, что мокать 
globs:
  - *.{ts,tsx,js,jsx}"
alwaysApply: true
---

# TESTING RULES (FRONTEND)

## Пирамида тестов

- Unit: чистые функции, маппинг, валидация, use-cases.
- Component: поведение UI (рендер, состояния loading/error, пользовательские действия).
- E2E: критические флоу (auth, настройки, создание/обновление сущностей).

## Что мокать

- мокать только внешние зависимости (HTTP client, время, storage, feature flags, analytics)
- НЕ мокать: DTO, value-like объекты, простые pure functions

## Обязательное покрытие

- happy path
- 3–5 edge cases
- invalid input / error mapping (если есть)

## Критерий качества

Тест должен проверять **поведение**, а не детали реализации.
