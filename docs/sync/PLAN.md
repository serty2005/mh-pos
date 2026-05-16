# SyncExchange v1 Production Plan

## Summary

- Фактический gap: в текущем дереве нет `POST /api/v1/sync/exchange` и Edge `Exchange()`. Есть legacy `/sync/edge-events`, `/sync/edge-events/batch`, Cloud package storage и POS `ApplyMasterData`, который уже атомарно применяет stream payload + `cloud_master_sync_state`.
- Цель реализации: добавить приоритетный authenticated exchange-цикл поверх существующей outbox/master-sync основы, не ломая legacy routes.
- Security default: новый exchange требует `Authorization: Bearer <node_token>` из provisioning credentials; legacy endpoints остаются совместимыми.
- E2E default: Playwright API tests без UI, с docker-compose окружением, API/БД/log assertions и артефактами.

## Public Contract

- Добавить `SyncExchange v1` в `cloud-backend/internal/cloudsync/contracts`:
  - Request: `protocol_version`, `node_device_id`, `restaurant_id`, `edge_events[]`, `streams[]`.
  - `edge_events[]`: `client_item_id` = POS outbox id, `payload` = strict `SyncEnvelope`.
  - `streams[]`: `stream_name`, `last_cloud_version`, optional `checkpoint_token`.
  - Response: `protocol_version`, `status`, `edge_acks[]`, `cloud_packages[]`, `stream_results[]`.
  - ACK statuses: `accepted`, `rejected`, `retryable`; include stable `error_code`, `message_key`, safe `details`.
- Limits:
  - max body `8 MiB`;
  - max edge events `100`;
  - max single envelope `2 MiB`;
  - supported exchange streams only: `restaurants`, `devices`, `staff`, `floor`, `catalog`, `menu`, `pricing_policy`.
- Revision/checkpoint rules:
  - If Cloud package missing: stream result `not_found`, no HTTP failure.
  - If unknown stream: HTTP `400 VALIDATION_FAILED`, no edge ingest.
  - If Edge known version is ahead of Cloud: HTTP `409 SYNC_REVISION_AHEAD`, no edge ingest.
  - If Edge version equals Cloud version but checkpoint differs: HTTP `409 SYNC_CHECKPOINT_CONFLICT`, no edge ingest.
  - If Cloud version is newer: return package; Edge applies it and commits `cloud_master_sync_state` in the existing local transaction boundary.

## Implementation Changes

- Cloud:
  - Add `POST /api/v1/sync/exchange` in `cloud-backend/internal/cloudsync/api`.
  - Add node-token auth for exchange only: validate bearer token against active `cloud_edge_nodes.credentials_hash`, match `node_device_id` and assigned `restaurant_id`; never log token.
  - Add app-level `Exchange(ctx, request)` that validates full request first, then receives edge events idempotently, then builds stream results/packages from `cloud_master_data_packages`.
  - Reuse existing receive/idempotency projection path for Edge events; preserve `/sync/edge-events` and `/batch`.
  - Add structured logs with `operation=sync.exchange`, action/result/error_code/request_id/node_device_id`.
- Edge:
  - Extend `pos-backend/internal/pos/infra/cloudsync.Client` with `Exchange(ctx, req)` and strict response parsing.
  - Add app service methods to expose local exchange identity, token, restaurant id, and `cloud_master_sync_state` without leaking credentials to logs.
  - Change `syncsender.Worker` orchestration to: reclaim stale locks, poll provisioning assignment if needed, claim outbox, build exchange request, call exchange, apply returned Cloud packages, then mark outbox item statuses.
  - If Cloud package apply fails, do not advance outbox statuses; retry repeats Edge events safely via Cloud idempotency.
  - Preserve current retry/reclaim/suspend ordering and wrong-direction protection.
- Documentation:
  - Update `docs/sync/edge-cloud-contracts-v1.md` and `docs/sync/directional-sync-ownership.md`.
  - Update backend error/runtime docs if new error codes/auth config are introduced.
  - Use only required Russian statuses: `реализовано сейчас`, `запланировано далее`, `вне текущего объема`.

## Test Plan

- Cloud Go tests:
  - exchange happy path;
  - partial ACK;
  - invalid envelope item;
  - missing package returns `not_found`;
  - invalid stream rejects whole exchange;
  - idempotent replay;
  - missing/invalid node token;
  - revision ahead and checkpoint conflict.
- Edge Go tests:
  - exchange request serialization includes local revisions/checkpoints and outbox ids;
  - response parsing for packages and item ACKs;
  - retryable HTTP: network, `429`, `5xx`;
  - non-retryable HTTP: `400`, `401`, `403`, `409`;
  - malformed response handling.
- Worker/repository tests:
  - full exchange happy path marks accepted outbox sent and applies package state;
  - package apply failure leaves outbox uncommitted for retry;
  - partial ACK marks accepted/rejected/retryable correctly;
  - stale lock/retry behavior remains ordered;
  - `ApplyMasterData` atomicity verifies no data/checkpoint split after simulated failure.
- E2E Playwright API:
  - add `pos-ui/e2e/sync-exchange.spec.ts`;
  - run docker compose stack;
  - cover online sync, Cloud unavailable, recovery catch-up, partial ACK, idempotent replay;
  - capture Playwright traces on failure plus docker logs and DB/endpoint assertions.

## Verification And Delivery

- Required commands:
  - `cd pos-backend && go mod tidy && go test ./...`
  - `cd cloud-backend && go mod tidy && go test ./...`
  - `cd pos-ui && npm install && npm run build`
  - `cd pos-ui && npx playwright test e2e/sync-exchange.spec.ts`
- Required doc/profile searches from `AGENTS.md` must be run before final report.
- Final report must state touched runtime code, changed files, test results, Playwright artifacts, remaining risks, what is next, what is outside scope.
- Commit message:
  - `feat(sync): add authenticated cloud edge exchange cycle`
- PR text:
  - Summary: authenticated `SyncExchange v1`, Edge checkpointed package apply, worker orchestration, docs/tests/E2E.
  - Risk: exchange auth/config rollout, docker E2E environment dependency.
  - Verification: list exact Go, UI build, Playwright, and doc-search commands.
