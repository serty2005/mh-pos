---
name: use-modern-go
description: "Use for any Go backend work in pos-backend or cloud-backend: handlers, services, storage, migrations, tests, RBAC, errors, logging, sync, startup, maintenance, and database code. Project targets Go 1.26.2."
---

# Modern Go For MyHoreca POS

Use this for Go changes in `pos-backend` and `cloud-backend`.

## Project Baseline

- Target Go version: 1.26.2.
- Prefer current standard library features when they reduce code or clarify intent.
- Keep APIs context-aware where useful.
- Use small interfaces only when there is a real boundary or a test seam that earns its keep.
- Avoid speculative abstractions, factories, and compatibility layers.
- No panics in request paths.
- Run `gofmt` on changed Go files.

## Architecture And Boundaries

- Respect existing module ownership and context boundaries from project docs.
- Keep HTTP handlers thin: parse/auth/validate/call service/map response.
- Put business rules in service/application code, not in transport or UI assumptions.
- Storage code owns SQL details and transaction boundaries.
- Use explicit transaction scopes for order, payment, shift, cash, migration, and sync mutations.
- Do not introduce cross-module dependencies that violate the current direction.

## Error Contract

- API errors must map to stable safe responses: error code, HTTP status, safe message key, safe details, and correlation/request ID when available.
- Internal causes are logged, not returned to clients.
- Never expose raw Go errors, SQL errors, stack traces, tokens, PINs, credentials, or sensitive payloads.
- Prefer typed/sentinel errors where callers need branching.
- Use `errors.Is`, `errors.As`, wrapping, and `errors.Join` where useful.

## Security And Risk

- Financial and order state mutations are high-risk operations.
- Do not auto-retry financial mutations without idempotency or an equivalent safety mechanism.
- Backend RBAC and application-layer checks are authoritative.
- Validate auth/session/client device data at trust boundaries.
- Do not log PINs, manager PINs, PIN hashes, tokens, secrets, credentials, raw auth payloads, or payment-sensitive payloads.
- Structured log fields use English names.

## Data And Migrations

- Runtime modules manage schema changes programmatically at startup.
- Active pre-pilot SQL baseline is the managed `001_init.sql` per runtime module.
- Changes to schema, constraints, indexes, master/reference data, or managed SQL require tests and relevant docs updates.
- `DB version > MH_POS_VERSION` must fail fast; downgrade is unsupported.
- Existing DB safe upgrade requires backup before mutation.
- Startup must verify critical tables, columns, indexes, and constraints before serving HTTP/workers.
- SQLite destructive cleanup requires backup, explicit confirmation, support/RBAC permission, audit log, and documented rebootstrap/restart path.

## Modern Go Defaults

- Use `any` instead of `interface{}`.
- Use `strings.Cut`, `CutPrefix`, `CutSuffix`, `SplitSeq`, and `FieldsSeq` where appropriate.
- Use `slices`, `maps`, `cmp`, `min`, `max`, and `clear` instead of manual loops when clearer.
- Use typed atomics from `sync/atomic`.
- Use `context.WithCancelCause`, `WithTimeoutCause`, `Cause`, and `AfterFunc` when cancellation reason matters.
- Use `sync.OnceFunc` and `sync.OnceValue` when they remove boilerplate.
- In tests, use `t.Context()` when a test needs a context.
- In benchmarks, use `b.Loop()`.

## Tests

- Prefer table-driven tests for business rules, state transitions, validators, error mapping, and storage edge cases.
- Keep fixtures small and explain non-obvious business meaning.
- Add tests for migration/backup behavior when schema or managed data changes.
- For high-risk mutations, test no-retry/idempotency behavior where applicable.

## Checks

- POS backend: `cd pos-backend && go mod tidy && go test ./...`
- Cloud backend: `cd cloud-backend && go mod tidy && go test ./...`
- Run only the relevant backend check unless the change crosses both modules.
