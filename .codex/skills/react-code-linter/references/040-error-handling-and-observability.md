---
description: Error discipline + observability (frontend) для React
globs:
  - *.{ts,tsx,js,jsx}"
alwaysApply: true
---

# ERROR HANDLING (FRONTEND)

## ❌ Запрещено

- silent catch
- игнор ошибок сетевого слоя
- «проглатывание» ошибок в `console.log` без UX реакции

## ✅ Обязательно

- Для любых запросов: loading/error/empty/success UI.
- Ошибки делить на категории: auth (401/403), validation (400/422), not found (404), conflict (409), server (5xx), network/timeout.
- Для auth ошибок: единый сценарий (refresh/logout/redirect) — централизованно в API слое.
- Для transient ошибок: retry/backoff (если безопасно).

## Observability

- Логи только без чувствительных данных (token, пароль, PII).
- Для важных ошибок — централизованный репортинг (Sentry/аналог), если используется в проекте.
