---
description: Frontend feature flags and gradual rollout kill switch, targeting, sunset plan
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# FEATURE FLAGS (FRONTEND) — 2026

Feature flags are used for UI migrations, canary releases, dark launches, and A/B tests.

## Rules

- A flag selects an implementation, route, or behavior, but must not spread business logic across the UI.
- Every flag must have a **sunset plan**: when and by whom it will be removed.
- Switching points must use a single helper/adapter, not hundreds of `if (flag)` checks across the codebase.

## Patterns

- Kill switch: immediately disable a feature.
- Canary: enable for a percentage of users or tenants.
- Migration toggle: v1 → v2.

## Forbidden

- permanent flags without removal
- flags without observability, at least a log or metric in a single centralized point
