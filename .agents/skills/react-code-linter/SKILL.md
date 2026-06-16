---
name: react-code-linter
description: Use after React/TypeScript UI changes, when npm build/typecheck/test fails, or before final reporting for UI work in pos-ui-g or cloud-ui-g. Uses this project's npm scripts, not yarn.
---

# UI Verification

Use the smallest check that can catch the risk introduced by the change.

## Commands

- POS UI build: `cd pos-ui-g && npm run build`
- POS UI typecheck: `cd pos-ui-g && npm run lint`
- POS UI unit tests: `cd pos-ui-g && npm run test`
- POS UI e2e: `cd pos-ui-g && npm run test:e2e`
- Cloud UI build: `cd cloud-ui-g && npm run build`
- Cloud UI typecheck: `cd cloud-ui-g && npm run lint`
- Cloud UI unit tests: `cd cloud-ui-g && npm run test`

## What To Run

- Style-only change: build, plus Playwright/screenshot if visual layout changed.
- Type/API/client change: build and typecheck.
- Store/state/business UI logic: build and unit tests if relevant tests exist.
- Dialog, table, navigation, auth/session, license, order/check/payment/shift workflow: build and Playwright when environment allows.
- i18n change: build and verify all supported locale files were updated.

## Rules

- Do not run `yarn`; this project uses npm scripts.
- Do not run every command by default. Match checks to changed files and risk.
- If a command fails, capture the first actionable error and fix it when in scope.
- Do not paste full logs into the final report. Summarize the failing command and key error.
- If the environment prevents a check, state that clearly.
