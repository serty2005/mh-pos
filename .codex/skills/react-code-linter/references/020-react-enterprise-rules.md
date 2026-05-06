---
description: Enterprise-level React rules for architecture, components, and layers
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# REACT HARD RULES

## Architecture

- Use functional components only. Class components are forbidden.
- Maintain strict isolation of layers and responsibilities.
- Define explicit state ownership and component boundaries.

## Layers (mandatory discipline)

- **UI**: rendering and UI event handlers only; no business logic.
- **Logic (domain/use-cases)**: business rules, transformations, orchestration.
- **Data (API/storage)**: HTTP, localStorage/sessionStorage, DTO mapping, caching.
- **Contracts**: request/response types and validation schemas, if present.

## ❌ Forbidden

- business logic inside UI components
- direct API calls from components, except thin page orchestrator components with strict justification
- mixing DTO / transport models with UI models without explicit mapping
- side effects during render; every effect must go through `useEffect` or hooks

## Components

- Keep components small, predictable, and reusable.
- Keep props minimal and typed.
- Avoid god components; a 500+ line component is a decomposition signal.

## Routing

- Route-level components are the page composition root: they connect data and logic, while UI is extracted into separate components.
