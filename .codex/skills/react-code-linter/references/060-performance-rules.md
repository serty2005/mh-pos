---
description: Frontend performance thinking for React: renders, requests, bundles
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# PERFORMANCE RULES (FRONTEND)

Always answer the question: **what breaks under 10x load?**

## React performance

- Avoid unnecessary re-renders: stable props, memoization where needed, and no excessive inline objects/functions in hot paths.
- Avoid heavy computations during render; extract and cache them.
- Large lists require virtualization when applicable.

## Network performance

- Deduplicate requests and cache where appropriate.
- Do not create N+1 requests during page render.
- Design pagination and filters for large datasets.

## Bundle performance

- Use code splitting by route or feature if the project supports it.
- Do not add heavy libraries for a small feature without justification.
