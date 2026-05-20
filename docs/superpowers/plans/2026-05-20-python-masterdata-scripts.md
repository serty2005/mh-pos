# Python Masterdata Scripts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build portable Python 3 scripts and thin environment wrappers for Cloud master-data seeding, POS Edge provisioning, and sync verification.

**Architecture:** Keep all behavior in standard-library Python modules under `scripts/lib`, expose small CLI entry points in `scripts/*.py`, and make `.sh`/`.ps1` wrappers pass arguments through. All data mutation goes through Cloud/POS/License HTTP APIs; database access remains diagnostic only.

**Tech Stack:** Python 3 standard library, `unittest`, Bash wrappers, ASCII PowerShell wrappers, existing Cloud/POS HTTP APIs.

---

### Task 1: Python Library Tests

**Files:**
- Create: `scripts/tests/test_mhpos_http.py`
- Create: `scripts/tests/test_mhpos_seed.py`

- [x] **Step 1: Write failing unit tests**

Tests define desired behavior for base URL normalization, Cloud seed call order, POS read-model verification, and polling for a post-pairing sync item.

- [x] **Step 2: Run tests to verify RED**

Run: `python3 -m unittest discover -s scripts/tests`

Expected: FAIL because `mhpos_http` and `mhpos_seed` do not exist yet.

### Task 2: Python Library Implementation

**Files:**
- Create: `scripts/lib/mhpos_http.py`
- Create: `scripts/lib/mhpos_seed.py`

- [x] **Step 1: Implement HTTP helpers**

Add a standard-library JSON client with `get`, `post`, `patch`, `put`, retryable wait helpers, and safe `HttpError`.

- [x] **Step 2: Implement seed/provision/verify helpers**

Add reusable functions for health checks, Cloud seed data, publication, license provisioning, POS auth headers, read-model verification, and post-pairing sync data creation.

- [x] **Step 3: Run unit tests**

Run: `python3 -m unittest discover -s scripts/tests`

Expected: PASS.

### Task 3: CLI Entrypoints And Wrappers

**Files:**
- Create: `scripts/seed-cloud-masterdata.py`
- Create: `scripts/provision-pos-edge.py`
- Create: `scripts/verify-sync.py`
- Create: `scripts/run-local-masterdata-smoke.py`
- Create: `scripts/seed-cloud-masterdata.sh`
- Create: `scripts/provision-pos-edge.sh`
- Create: `scripts/verify-sync.sh`
- Create: `scripts/run-local-masterdata-smoke.sh`
- Create: `scripts/seed-cloud-masterdata.ps1`
- Create: `scripts/provision-pos-edge.ps1`
- Create: `scripts/verify-sync.ps1`
- Create: `scripts/run-local-masterdata-smoke.ps1`

- [x] **Step 1: Add CLI scripts**

Each Python script parses arguments, calls `scripts/lib` helpers, and writes JSON summary to stdout or `--output`.

- [x] **Step 2: Add thin wrappers**

Bash wrappers use `python3`; PowerShell wrappers use `python` and ASCII-only text.

- [x] **Step 3: Run CLI syntax checks**

Run: `python3 -m py_compile scripts/*.py scripts/lib/*.py`

Expected: exit code 0.

### Task 4: Documentation

**Files:**
- Modify: `docs/backend/LOCAL-DOCKER-STACK.md`
- Modify: `README.md`
- Modify: `ROADMAP.md`

- [x] **Step 1: Update local Docker docs**

Document Fedora/Linux commands, Python smoke scripts, wrappers, and the rule that seed data must grow with new master-data surfaces.

- [x] **Step 2: Update README**

Point local bootstrap documentation at Python scripts and keep PowerShell wrappers as compatibility entry points.

- [x] **Step 3: Update roadmap**

Record the script migration as implemented and keep dataset expansion as ongoing pre-pilot maintenance.

### Task 5: Verification

**Files:**
- All changed script and docs files.

- [x] **Step 1: Run unit tests**

Run: `python3 -m unittest discover -s scripts/tests`

- [x] **Step 2: Run syntax checks**

Run: `python3 -m py_compile scripts/*.py scripts/lib/*.py`

- [x] **Step 3: Run targeted documentation search**

Run: `rg "cloud-masterdata-e2e.ps1|bootstrap-production-way.ps1|python3 scripts/run-local-masterdata-smoke.py|seed data|demo seed|PowerShell" README.md docs/backend/LOCAL-DOCKER-STACK.md ROADMAP.md scripts`

- [x] **Step 4: Report integration checks not run**

Docker stack smoke is manual unless services are already running and user asks to execute it.
