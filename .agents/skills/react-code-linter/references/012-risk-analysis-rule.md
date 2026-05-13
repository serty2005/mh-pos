---
description: Mandatory risk assessment before React implementation
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# ⚠️ RISK ANALYSIS RULE (CRITICAL)

## ❗ Absolute rule

Before implementation, you MUST perform a risk analysis.

## ✅ Mandatory identification and description

### ⚠️ Technical risks

- architecture mistakes
- scaling issues
- potential bottlenecks
- performance issues

### ⚠️ Logical risks

- incorrect business logic
- edge cases
- inconsistent states

### ⚠️ Integration risks

- API incompatibility
- contract changes
- dependency on external services

## Format

### ⚠️ Risks

- ...

### 🔧 Mitigations

- ...

## ❌ Forbidden

- ignoring risks
- assuming a solution is obviously safe
- ignoring edge cases

## 🧠 Principle

Every solution has risks. If you do not see them, you have missed them.
