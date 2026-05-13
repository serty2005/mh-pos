---
description: Universal anti-hallucination protection for React
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# 🚨 ANTI-HALLUCINATION (CRITICAL)

If you are not 100% sure about:

- browser or Node APIs
- React / Vite / Router / library versions
- fetch / axios / cache / retry behavior
- backend contract formats
- CORS / Auth / CSRF rules

## ❌ FORBIDDEN

- inventing APIs, hooks, or config options
- guessing signatures
- using similar-looking methods without verification
- referencing non-existent capabilities

## ✅ INSTEAD

- Write explicitly: **"The version must be clarified / the documentation must be checked."**
- If there is a compatibility risk: **"There may be an incompatibility — the version must be checked."**
- If the solution depends on context, list assumptions and propose 2–3 options.

Hallucination is a critical error.
