---
description: Frontend zero-trust security for React: input, auth, tokens, XSS/CSRF, storage
globs:
  - "*.{ts,tsx,js,jsx}"
alwaysApply: true
---

# SECURITY (FRONTEND, ZERO TRUST)

## Principle

Every input — UI input, query params, localStorage, backend response — is potentially malicious or inconsistent.

## Mandatory

- Validate and normalize user input before sending it.
- Escape or sanitize user-generated content before displaying it.
- Do not use `dangerouslySetInnerHTML` without strict sanitization and justification.
- Tokens and secrets: do not log them and do not store them in plain text.
- Do not use localStorage for highly sensitive tokens if it can be avoided. Prefer httpOnly cookies if the backend is designed that way.

## Forbidden

- Hardcoding secrets, keys, tokens, or private URLs in the repository.
- Showing sensitive details in UI or logs, such as stack traces, tokens, or raw responses.
