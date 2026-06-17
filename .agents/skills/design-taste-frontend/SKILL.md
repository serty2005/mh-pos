---
name: design-taste-frontend
description: "Use for any React/Vite/Tailwind UI work in pos-ui-g or cloud-ui-g: screens, forms, tables, navigation, i18n, accessibility, loading/empty/error states, workflow ergonomics, API integration, and Playwright checks. Primary frontend skill for this project."
---

# MyHoreca Frontend Engineering

Use this for product UI work in `pos-ui-g` and `cloud-ui-g`. These apps are operational POS/admin tools, not marketing sites.

## Project Baseline

- Stack: React 19, Vite 6, TypeScript, Tailwind CSS 4.
- Installed icons: `lucide-react`. Use it unless the edited file already uses another installed source.
- `pos-ui-g` has `motion`; `cloud-ui-g` does not. Prefer CSS for simple transitions.
- User-facing text must go through i18n locale files. Do not hardcode Russian UI strings in source code.
- Frontend visibility is UX only. Backend RBAC and application-layer checks are authoritative.
- Keep changes inside the existing routing/component/store/API patterns. Do not add UI kits, state libraries, routers, animation stacks, or date picker libraries without an explicit request.

## When This Skill Triggers

- Building or changing product screens, panels, dialogs, drawers, tables, filters, forms, navigation, settings, auth/session UI, license/support UI, catalog/order/check/payment/shift workflows.
- Adding frontend API calls or changing request/response handling.
- Fixing layout, responsiveness, accessibility, empty/loading/error states, or user workflow friction.

## Before Editing

1. Read the target component and its nearest shared components.
2. Read the related store/composable/API client and types.
3. Read relevant locale files before adding or changing user-facing copy.
4. Check `package.json` before importing anything new.
5. Preserve existing behavior unless the user explicitly asked to change it.

## Product UI Principles

- Start with the real workflow as the first screen. No landing-page wrapper for internal tools.
- Optimize for repeated use: scanning, comparing, filtering, editing, confirming, recovering.
- Prefer dense but calm layouts: stable toolbars, compact filters, clear tables, predictable detail panels.
- Avoid marketing composition: oversized heroes, decorative card grids, abstract blobs, meaningless badges, purple/blue AI gradients.
- Use cards only for repeated items, modals, and genuinely framed tools. Do not nest cards.
- Use familiar controls:
  - icon buttons for tools;
  - segmented controls for modes;
  - toggles/checkboxes for boolean settings;
  - inputs/steppers/sliders for numeric values;
  - tabs for stable views;
  - menus/selects for option sets;
  - text buttons only for clear commands.

## Data And State Handling

- Every async surface must have loading, empty, error, and success/recovery behavior.
- Loading states should preserve layout dimensions. Prefer skeletons or reserved space over spinners that cause jump.
- Empty states should be actionable but compact. Do not write marketing copy.
- Error UI must use safe message keys/details. Never show raw Go errors, SQL errors, stack traces, tokens, PINs, credentials, request dumps, or sensitive payloads.
- High-risk operations such as financial/order state changes need clear affordance, idempotency awareness where relevant, disabled duplicate submit, and explicit confirmation when destructive.
- Do not make frontend-authoritative decisions for financial state transitions.

## Accessibility And Input

- Use semantic buttons/links, not clickable divs.
- Keep visible focus states.
- Provide labels for form fields and accessible names for icon-only buttons.
- Preserve keyboard navigation through dialogs, menus, tabs, and table actions.
- Text must not clip, overlap, or escape its container on mobile or desktop.
- Dialogs and drawers need safe close/cancel behavior and predictable focus return where the local pattern supports it.

## Tables And Operational Screens

- Keep column widths stable; avoid hover states that resize rows or headers.
- Use tabular figures for money, counts, dates, durations, and metrics.
- Keep row actions discoverable but not noisy. Prefer a compact action column or contextual menu.
- Filters should be close to the data they affect and should preserve scanability when wrapping.
- If a table supports selection, sorting, drag, resize, or column visibility, avoid conflicting click/drag zones.
- Preserve existing saved user preferences and layout profiles unless asked to change them.

## Styling

- Use `minimalist-ui` whenever the task changes visual language, density, color, typography, spacing, cards, or controls.
- Use `fixing-motion-performance` for non-trivial animation, scroll-linked effects, jank, or long-running motion.
- Keep icons visually consistent: same size family, aligned optical center, no mixed stroke weights in the same control cluster.

## Verification

- For `pos-ui-g`: run the smallest relevant command, usually `npm run build`; use `npm run lint`, `npm run test`, or `npm run test:e2e` when the change justifies it.
- For `cloud-ui-g`: run `npm run build`; use `npm run lint` or `npm run test` when relevant.
- Use Playwright/screenshot checks when behavior, responsive layout, dialogs, workflows, drag/drop, tables, or visual framing changed.
