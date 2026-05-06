---
description: React testing rules: unit/component/e2e and mocking discipline
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# TESTING RULES (FRONTEND)

## Test pyramid

- Unit: pure functions, mapping, validation, use-cases.
- Component: UI behavior such as rendering, loading/error states, and user actions.
- E2E: critical flows such as auth, settings, and creating/updating entities.

## What to mock

- Mock only external dependencies: HTTP client, time, storage, feature flags, analytics.
- DO NOT mock DTOs, value-like objects, or simple pure functions.

## Mandatory coverage

- happy path
- 3–5 edge cases
- invalid input / error mapping, if applicable

## Quality criterion

A test must verify behavior, not implementation details.
