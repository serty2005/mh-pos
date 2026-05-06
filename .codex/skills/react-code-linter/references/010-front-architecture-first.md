---
description: Architecture-first approach and alternative analysis for React
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# 🏗 ARCHITECTURE FIRST (CRITICAL)

## ❗ Absolute rule

DO NOT start by writing code, except for a micro-fix of 1–2 lines that does not affect architecture.

Before implementation, you must describe:

- the layers: UI / logic / data-access / contracts
- the data flow: from where → to where → who owns state
- error handling: network / auth / validation
- completion criteria: what counts as done

## ⚖️ MULTIPLE SOLUTIONS RULE

Always propose at least **2–3** solution options:

- brief description
- pros / cons
- when to choose it

Then choose the best option and explain WHY.

## 🔍 HIDDEN REQUIREMENTS DETECTION

Before proposing a solution, list:

- what is undefined: versions, API contracts, UX details
- possible hidden requirements: auth, roles, i18n, offline, accessibility, performance
- potential constraints: legacy, backward compatibility, SEO, analytics
