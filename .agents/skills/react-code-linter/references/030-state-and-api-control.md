---
description: State ownership and API discipline for React  loading/error/success, retry, contracts
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# STATE & API CONTROL

## State ownership

- **local state** (`useState` / `useReducer`) is the default.
- **context** is mainly for dependency injection or cross-cutting concerns such as theme, auth, and i18n, not for business state.
- **global state** such as Redux, Zustand, MobX, RTK Query, etc. is allowed only with explicit justification: multiple pages, complex synchronization, or caching.

## Data fetching / API

- All HTTP calls must live only in the data/service layer, such as an API client.
- Components must not know about URLs, headers, or token refresh. They should only know about use-cases or service methods.
- DTO → UI model mapping, and UI model → DTO mapping, must be explicit.

## Async state machine (mandatory)

Every async scenario must account for these states:

- loading
- success
- error
- empty, if applicable

Optional states:

- retry, for transient errors
- partial success, if the page consists of independent widgets

## ❌ Forbidden

- unhandled promises or lost async errors
- silent catch
- updating state after unmount without protection
- request races without cancellation or deduplication, if the scenario allows them
