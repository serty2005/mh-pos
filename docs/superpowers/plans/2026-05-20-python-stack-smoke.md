# Python Stack Smoke Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a portable Python stack smoke runner that controls Cloud API, POS Edge API and License Server checks from one CLI.

**Architecture:** Keep reusable orchestration in `scripts/lib/mhpos_stack.py`, keep existing master-data behavior in `mhpos_seed.py`, expose `scripts/run-stack-smoke.py`, and use `docs/api/mhpos-local-smoke.openapi.json` operation IDs for HTTP calls.

**Tech Stack:** Python 3 standard library, `unittest`, OpenAPI 3.0 JSON, existing local Docker stack HTTP APIs.

---

### Task 1: Stack Result Model And Runner

**Files:**
- Create: `scripts/tests/test_mhpos_stack.py`
- Create: `scripts/lib/mhpos_stack.py`

- [x] **Step 1: Write failing tests**

Add tests for `StackContext`, `CheckResult`, `StackSmokeResult`, suite selection, failed-result conversion and health suite calls.

- [x] **Step 2: Run RED**

Run: `python -m unittest scripts.tests.test_mhpos_stack`

Expected: FAIL because `mhpos_stack` does not exist.

- [x] **Step 3: Implement minimal library**

Implement result objects as dictionaries/dataclasses, context construction and suite execution.

- [x] **Step 4: Run GREEN**

Run: `python -m unittest scripts.tests.test_mhpos_stack`

Expected: PASS.

### Task 2: License Pairing Suite

**Files:**
- Modify: `docs/api/mhpos-local-smoke.openapi.json`
- Modify: `scripts/lib/mhpos_stack.py`
- Modify: `scripts/tests/test_mhpos_contract.py`
- Modify: `scripts/tests/test_mhpos_stack.py`

- [x] **Step 1: Write failing tests**

Add tests for `registerLicensePairingCode` and `resolveLicensePairingCode` operation resolution, then test that the suite registers, resolves and confirms consumed behavior.

- [x] **Step 2: Run RED**

Run: `python -m unittest scripts.tests.test_mhpos_contract scripts.tests.test_mhpos_stack`

Expected: FAIL until OpenAPI and suite are implemented.

- [x] **Step 3: Implement suite**

Generate one-time code and future `expires_at`, call License Server directly, assert first resolve succeeds and second resolve returns safe failure.

- [x] **Step 4: Run GREEN**

Run: `python -m unittest scripts.tests.test_mhpos_contract scripts.tests.test_mhpos_stack`

Expected: PASS.

### Task 3: CLI And Masterdata Suite

**Files:**
- Create: `scripts/run-stack-smoke.py`
- Modify: `scripts/lib/mhpos_stack.py`
- Modify: `scripts/tests/test_mhpos_stack.py`

- [x] **Step 1: Write failing tests**

Add tests for suite parsing and stack smoke JSON shape.

- [x] **Step 2: Implement CLI**

Expose `--suite`, bases, output paths and wait options. Exit non-zero if any suite failed.

- [x] **Step 3: Run Python tests**

Run: `python -m unittest discover -s scripts/tests`

Expected: PASS.

### Task 4: Documentation And Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/backend/LOCAL-DOCKER-STACK.md`
- Modify: `ROADMAP.md`

- [x] **Step 1: Document standalone module**

Describe `run-stack-smoke.py`, suites and extension rule: new service functionality must add OpenAPI operation and suite coverage.

- [x] **Step 2: Run verification**

Run Python unit tests, Python compile check, targeted Go API tests, and if host HTTP access works, run `python scripts/run-stack-smoke.py --suite all`.
