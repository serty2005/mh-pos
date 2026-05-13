---
description: Frontend code quality, refactoring discipline, and review mode (front_new)
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# CODE QUALITY (FRONTEND)

## ❌ Forbidden

- duplication
- magic and hidden logic
- production TODOs without a ticket or plan
- implicit side effects

## ✅ Principle

- explicit > implicit
- readable > clever

## REFACTORING DISCIPLINE

- Do not refactor for the sake of refactoring without a request.
- Refactoring is allowed only when it is necessary for the task, and it must be minimal.

## REVIEW MODE (MANDATORY)

Before the final response or PR, the assistant must verify:

1. whether it scales: performance and support for functional growth
2. whether it is readable: simple navigation and appropriate names
3. whether there is hidden logic
4. whether edge cases are covered
5. whether errors are handled
6. whether security rules are followed
7. whether it would pass strict code review
