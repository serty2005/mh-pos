---
name: redesign-existing-projects
description: Use when the user asks to redesign, visually upgrade, polish, or audit an existing MyHoreca UI screen. Focus on improving the current product UI without rewriting behavior or changing stack.
---

# Redesign Existing MyHoreca Screens

Use this after `design-taste-frontend` when the request is primarily visual redesign, polish, or UX audit.

## Goal

Upgrade the existing screen in place. Keep behavior, routes, API contracts, permission logic, i18n, and domain flow intact unless explicitly requested.

## Workflow

1. Scan the current screen, shared components, style files, locale strings, and screenshots if available.
2. Identify the actual problem: density, hierarchy, contrast, flow, state coverage, responsiveness, or interaction friction.
3. Choose the smallest set of changes that improves scanability and trust.
4. Apply focused edits using the current stack.
5. Verify build and, when possible, Playwright/screenshot at desktop and mobile widths.

## Audit Checklist

- Information hierarchy: primary task is obvious; secondary actions do not compete with primary actions.
- Density: screen uses space efficiently without becoming cramped.
- Tables: stable columns, readable rows, clear status, useful empty/loading/error states.
- Forms: labels, validation, grouping, disabled states, submit/cancel hierarchy.
- Actions: destructive operations stand out and require appropriate confirmation.
- Navigation: current location and return path are clear.
- Responsiveness: no horizontal scroll, clipped buttons, overlapping text, or broken toolbar wrapping.
- Accessibility: focus states, semantic controls, labels, keyboard paths.
- Visual noise: remove nested cards, excessive borders, decorative gradients, redundant badges, and inconsistent icons.

## Fix Priority

1. Broken layout, overlap, clipped text, inaccessible controls.
2. Missing loading/empty/error states.
3. Poor action hierarchy or dangerous destructive affordances.
4. Table density, filters, and scan paths.
5. Typography, spacing, border, and color cleanup.
6. Motion and micro-interaction polish.

## Constraints

- Do not rebuild the app from scratch.
- Do not add a UI kit or new design dependency.
- Do not turn operational screens into marketing pages.
- Do not replace domain copy with generic sales copy.
- Do not remove security, RBAC, validation, or audit-related UI cues.

## Taste

- Use `minimalist-ui` for visual choices.
- Keep POS/admin screens compact, quiet, and structured.
- Prefer fewer stronger layout decisions over many decorative details.
