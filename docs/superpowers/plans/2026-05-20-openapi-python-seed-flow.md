# OpenAPI Python Seed Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make local Python seed/smoke scripts execute through an OpenAPI-defined contract instead of hardcoded endpoint strings.

**Architecture:** Add a focused OpenAPI spec for the Cloud/POS/License endpoints used by the local smoke flow, load it with a standard-library Python helper, and call HTTP operations by stable `operationId`. Keep the Go runtime unchanged; this task only aligns the portable test/seed layer with existing API routes.

**Tech Stack:** Python 3 standard library, `unittest`, OpenAPI 3.0 JSON, existing Go HTTP API routers.

---

### Task 1: Contract Tests

**Files:**
- Modify: `scripts/tests/test_mhpos_seed.py`
- Create: `scripts/tests/test_mhpos_contract.py`

- [x] **Step 1: Add failing tests**

Add tests that expect seed helpers to call canonical Cloud master-data endpoints (`/master-data/...`) and expect the OpenAPI contract to resolve `operationId` values into method/path pairs.

- [x] **Step 2: Verify RED**

Run: `python -m unittest discover -s scripts/tests`

Expected: FAIL because `mhpos_contract` and the OpenAPI spec do not exist yet.

### Task 2: OpenAPI Contract Loader

**Files:**
- Create: `docs/api/mhpos-local-smoke.openapi.json`
- Create: `scripts/lib/mhpos_contract.py`

- [x] **Step 1: Add OpenAPI spec**

Define the local smoke endpoints used by Cloud seeding, POS provisioning, POS read-model checks, sync status, and root health checks.

- [x] **Step 2: Add Python loader**

Implement `load_default_contract()`, `operation()`, `build_request()`, path parameter substitution, query encoding, expected success status extraction, and required request-body field validation.

- [x] **Step 3: Run tests**

Run: `python -m unittest discover -s scripts/tests`

Expected: only seed helper path tests still fail until the helpers are migrated.

### Task 3: Seed Helper Migration

**Files:**
- Modify: `scripts/lib/mhpos_seed.py`

- [x] **Step 1: Replace raw HTTP paths with operation IDs**

Create a small `_call()` helper around the OpenAPI contract and update create/publish/provision/login/read/sync functions to use it.

- [x] **Step 2: Prefer canonical Cloud master-data paths**

Use `/master-data/roles`, `/master-data/employees`, `/master-data/catalog/items`, `/master-data/floor/halls`, `/master-data/floor/tables`, `/master-data/menu/items`, `/master-data/modifiers/*`, and `/master-data/publications`.

- [x] **Step 3: Run Python tests**

Run: `python -m unittest discover -s scripts/tests`

Expected: PASS.

### Task 4: Documentation And Verification

**Files:**
- Modify: `README.md`
- Modify: `docs/backend/LOCAL-DOCKER-STACK.md`
- Modify: `ROADMAP.md`

- [x] **Step 1: Document contract-first seed flow**

Describe the OpenAPI spec as the source used by Python smoke/seed calls.

- [x] **Step 2: Run verification**

Run:

```powershell
$env:PYTHONIOENCODING='utf-8'
python -m unittest discover -s scripts/tests
python -m py_compile (Get-ChildItem scripts -Filter *.py).FullName (Get-ChildItem scripts/lib -Filter *.py).FullName
cd cloud-backend; go test ./internal/masterdata/api ./internal/provisioning/api
cd ../pos-backend; go test ./internal/pos/api
```

Expected: PASS, unless local environment lacks the needed toolchain.
