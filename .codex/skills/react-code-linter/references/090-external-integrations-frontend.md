---
description: Frontend external integration patterns contracts, resilience, idempotency
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# EXTERNAL INTEGRATIONS (FRONTEND)

## Contracts and compatibility

- API contracts must be typed as DTOs and explicitly mapped to UI models.
- Backward compatibility: the UI must tolerate missing new fields and the appearance of additional fields.

## Resilience

- Use request timeouts and cancellation if supported by the client.
- Retry only **idempotent** requests and transient errors such as network/5xx, with backoff.
- Circuit breaker is allowed only if there is a shared layer/library and it is justified. Otherwise, do not invent it.

## Idempotency / double submit

- Protect against double submits: disable buttons while loading, deduplicate requests by key, and use explicit in-flight flags.

## Forbidden

- storing raw responses in UI state without normalization/mapping
- hiding integration errors from the user without a UX solution
