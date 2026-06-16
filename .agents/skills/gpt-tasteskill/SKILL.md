---
name: gpt-tasteskill
description: Use only when the user explicitly asks for an experimental, cinematic, marketing, showcase, or demo page. Do not use for MyHoreca POS runtime screens, admin screens, tables, forms, backend work, or ordinary UI polish.
---

# Experimental Showcase UI

This skill is intentionally narrow in this project. It is for non-runtime visual experiments only.

## Use Only When

- The user explicitly asks for a landing page, marketing page, cinematic demo, scroll storytelling, animated showcase, or visual concept.
- The work is outside normal POS/admin workflows.
- The user accepts that the result may be more expressive than the operational product UI.

## Do Not Use For

- `pos-ui-g` or `cloud-ui-g` runtime screens.
- Tables, forms, dialogs, RBAC, auth, licensing, order/check/payment/shift workflows.
- Accessibility or performance fixes.
- Ordinary redesign/polish of existing product screens. Use `redesign-existing-projects` and `minimalist-ui`.

## Implementation Constraints

- Do not add GSAP, Three.js, image libraries, or animation dependencies unless the user explicitly asks and accepts the dependency.
- Prefer installed CSS/Tailwind features first.
- In `pos-ui-g`, `motion` is available; use it only when needed and isolated.
- In `cloud-ui-g`, do not assume `motion` exists.
- Generated pages must still be responsive, accessible, and free of product-app hardcoded user-facing strings.

## Visual Direction

- Strong typography, clear contrast, real imagery or generated bitmap assets when the page needs visual impact.
- No generic SaaS card rows, purple AI gradients, decorative blobs, or empty atmospheric stock backgrounds.
- If animation is used, isolate it and provide cleanup for JS-driven effects.
