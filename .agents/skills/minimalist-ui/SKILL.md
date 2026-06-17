---
name: minimalist-ui
description: "Use when changing MyHoreca POS visual style: color, typography, spacing, density, cards, tables, forms, buttons, icons, responsive layout, and polished operational UI details. Style companion for design-taste-frontend."
---

# MyHoreca UI Style System

Use this as the visual decision layer for `pos-ui-g` and `cloud-ui-g`.

## Direction

- MyHoreca UI should feel like a premium operational product: calm, exact, structured, fast to scan.
- It should not feel like a SaaS landing page, creative portfolio, or generic AI dashboard.
- The design should support restaurant/POS work: orders, checks, shifts, cash, catalog, staff, licenses, support, settings.

## Palette

- Base surfaces: white, near-white, zinc/slate/stone neutrals, and subtle panel backgrounds.
- Use one restrained accent per view or workflow. Accent color must carry meaning: active, selected, warning, destructive, success, focus.
- Avoid dominant purple/blue gradients, neon, decorative glows, beige-only palettes, and large saturated color blocks.
- Prefer borders and subtle surface contrast over heavy shadows.
- Use semantic colors consistently:
  - success: completed/healthy/paid;
  - warning: needs attention but recoverable;
  - danger: destructive, blocked, failed, or unsafe;
  - neutral: inactive, secondary, metadata.

## Typography

- Use the existing project font stack unless the local theme already defines a better option.
- Do not introduce remote fonts casually.
- Panel headings should be compact and useful, not hero-scale.
- Prefer sentence case for UI labels and headings.
- Use tabular numbers for money, quantities, durations, dates, counts, stock, tax, payment, shift totals, and table metrics.
- Avoid all-caps labels unless the local design already uses them for small metadata.

## Density And Spacing

- Use a 4/8px rhythm.
- Default to compact spacing for work surfaces, with enough breathing room around grouped controls.
- Preserve scan paths: title/context, filters, primary actions, data, details.
- Toolbars should wrap intentionally, not randomly. At narrow widths, group secondary actions into menus.
- Mobile layouts must collapse without horizontal scroll or clipped controls.

## Components

- Tables:
  - stable row height and column widths;
  - sticky headers only when useful;
  - no layout shift on hover;
  - align numeric columns to support comparison;
  - keep row status readable without color alone.
- Forms:
  - label above field;
  - error below field;
  - helper text only when useful;
  - group related fields with spacing or dividers rather than nested cards.
- Buttons:
  - clear hierarchy: primary, secondary, tertiary, danger;
  - icon-only buttons need accessible names and tooltips if meaning is not obvious;
  - destructive buttons must be visually distinct.
- Cards:
  - radius 8px or less unless the existing design system says otherwise;
  - use cards for repeated entities, modals, and framed tools;
  - do not put cards inside cards.
- Badges:
  - compact, semantic, readable;
  - not decorative;
  - status text must remain understandable without relying only on color.
- Dialogs/drawers:
  - focused task scope;
  - visible cancel/close path;
  - clear primary action;
  - no unnecessary explanatory text.

## Icons

- Use `lucide-react` for new icons.
- Match size, stroke, and vertical alignment inside each control group.
- Prefer familiar icons for common commands: save, download, refresh, search, filter, settings, edit, delete, close, back.
- Do not use emoji as UI icons.

## Motion

- Motion should clarify feedback or orientation, never decorate the work surface.
- Use short opacity/transform transitions.
- Avoid constant ambient motion in POS/admin screens.
- Respect `prefers-reduced-motion`.
- For complex animation or performance problems, use `fixing-motion-performance`.

## Anti-Patterns To Remove

- Huge hero sections inside the product app.
- Decorative blobs/orbs and generic gradients.
- Nested cards and repeated bordered boxes.
- Toolbars made of many equal-weight buttons.
- Empty states with marketing copy.
- Clipped text, overlapping controls, unstable hover states.
- Unexplained icon-only controls.
