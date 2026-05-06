---
description: Frontend error discipline and observability for React
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# ERROR HANDLING (FRONTEND)

## ❌ Forbidden

- silent catch
- ignoring network-layer errors
- swallowing errors with `console.log` without any UX reaction

## ✅ Mandatory

- For every request: loading / error / empty / success UI.
- Categorize errors: auth (401/403), validation (400/422), not found (404), conflict (409), server (5xx), network/timeout.
- Auth errors must follow a centralized scenario in the API layer: refresh / logout / redirect.
- Transient errors must use retry/backoff when it is safe.

## Observability

- Logs must never include sensitive data such as tokens, passwords, or PII.
- Important errors must use centralized reporting such as Sentry or an equivalent tool, if the project uses one.
