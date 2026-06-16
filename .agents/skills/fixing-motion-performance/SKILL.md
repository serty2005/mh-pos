---
name: fixing-motion-performance
description: Use when adding, changing, auditing, or fixing animations in MyHoreca UI, especially jank, scroll-linked motion, layout thrashing, blur/filter effects, drag interactions, or long-running motion.
---

# Motion Performance

Use this for animation and interaction performance in `pos-ui-g` and `cloud-ui-g`.

## Project Baseline

- Prefer CSS transitions/animations for simple feedback.
- `pos-ui-g` has `motion`; use it only when it is already the local pattern or CSS is not enough.
- `cloud-ui-g` does not have `motion`; do not add it for small transitions.
- Do not add GSAP, Three.js, or another animation library unless explicitly requested.

## Rendering Rules

- Default animated properties: `transform` and `opacity`.
- Avoid animating `top`, `left`, `right`, `bottom`, `width`, `height`, `padding`, `margin`, large shadows, filters, and inherited CSS variables.
- Paint-heavy animation is allowed only on small isolated elements and for short durations.
- Blur/filter animation should be small, short, and isolated. Never animate blur continuously on large surfaces.
- Use `will-change` narrowly and temporarily. Do not promote many large layers.
- Respect `prefers-reduced-motion`.

## Measurement And Layout

- Do not interleave layout reads and writes in the same frame.
- Measure once, then animate via transform.
- For layout-like transitions, prefer FLIP-style behavior: first rect, final rect, inverted transform, then transition to zero.
- Do not repeatedly call layout-reading APIs inside animation loops.

## Scroll And Visibility

- Do not drive animation from `scroll` listeners or continuous `scrollY` polling.
- Use CSS view timelines when suitable, or `IntersectionObserver` for reveal/pause behavior.
- Long-running animations should pause when off-screen.
- Scroll-linked motion must not trigger continuous layout or paint on large surfaces.

## Interaction Patterns

- Button and row feedback: short active/hover transform or opacity change.
- Drag/resize: keep the drag surface stable; do not reflow the whole table on every pointer move.
- Skeletons: animate shimmer only if cheap; static skeletons are acceptable.
- Table hover/focus: avoid row height changes and expensive shadows.

## Common Fixes

```css
/* Before: layout animation */
.panel {
  transition: width 200ms;
}

/* After: compositor-friendly animation */
.panel {
  transition: transform 200ms ease, opacity 200ms ease;
}
```

```ts
// Prefer visibility-based behavior over scroll polling.
const observer = new IntersectionObserver(([entry]) => {
  element.dataset.visible = String(entry.isIntersecting)
})
observer.observe(element)
```

## Review Guidance

- Fix critical jank first; do not redesign the component unless requested.
- Keep the existing animation system.
- State the constraint behind any expensive effect.
- Verify with browser/Playwright when the change affects layout, drag, scrolling, dialogs, or responsive behavior.
