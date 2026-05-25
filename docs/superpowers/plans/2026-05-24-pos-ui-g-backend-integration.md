# POS UI G Backend Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Подключить React-дизайн `pos-ui-g` к существующему POS backend как параллельный UI-вариант без переноса бизнес-логики во фронтенд.

**Architecture:** `pos-ui-g` остается визуальным React/Vite приложением. Новый frontend-слой состоит из API client, runtime auth state, DTO schemas и pure mappers, а backend остается авторитетным источником смен, заказов, пречеков, платежей, возвратов, RBAC и sync state. Компоненты получают прежнюю форму `usePOS()`, но данные и команды приходят из backend.

**Tech Stack:** React 19, Vite, TypeScript, zod, Vitest, existing POS HTTP API `/api/v1`.

---

### Task 1: Testable Contract Layer

**Files:**
- Modify: `pos-ui-g/package.json`
- Create: `pos-ui-g/src/shared/clientIdentity.ts`
- Create: `pos-ui-g/src/shared/schemas.ts`
- Create: `pos-ui-g/src/shared/backendMappers.ts`
- Test: `pos-ui-g/src/shared/backendMappers.test.ts`

- [ ] **Step 1: Write failing mapper tests**

```ts
import { describe, expect, it } from 'vitest';
import { mapMenuItem, mapOrder, mapTable } from './backendMappers';

it('maps backend table and active order to design table state', () => {
  const table = mapTable(
    { id: 'tbl-1', restaurant_id: 'r1', hall_id: 'hall-1', name: 'Стол 7', seats: 4, active: true },
    { id: 'ord-1', edge_order_id: 'e1', restaurant_id: 'r1', device_id: 'dev', shift_id: 'shift', status: 'open', table_id: 'tbl-1', table_name: 'Стол 7', guest_count: 3, total: 1200, opened_at: '2026-05-24T10:00:00Z', closed_at: null, created_at: '2026-05-24T10:00:00Z', updated_at: '2026-05-24T10:00:00Z', lines: [] },
  );
  expect(table).toMatchObject({ id: 'tbl-1', number: 7, status: 'occupied', guestsCount: 3, activeOrderSum: 1200 });
});

it('maps backend modifier groups to design menu item modifiers', () => {
  const item = mapMenuItem({
    id: 'menu-1',
    catalog_item_id: 'cat-1',
    item_type: 'dish',
    name: 'Стейк',
    price: 1450,
    currency: 'RUB',
    active: true,
    created_at: '2026-05-24T10:00:00Z',
    updated_at: '2026-05-24T10:00:00Z',
    modifier_groups: [{
      id: 'grp-1',
      restaurant_id: 'r1',
      name: 'Прожарка',
      required: true,
      min_count: 1,
      max_count: 1,
      active: true,
      options: [{ id: 'opt-1', restaurant_id: 'r1', modifier_group_id: 'grp-1', name: 'Medium', price_minor: 0, active: true }],
    }],
  });
  expect(item.modifierGroups?.[0]).toMatchObject({ id: 'grp-1', minRequired: 1, maxAllowed: 1 });
});

it('maps backend order lines without calculating business totals', () => {
  const order = mapOrder({
    id: 'ord-1',
    edge_order_id: 'e1',
    restaurant_id: 'r1',
    device_id: 'dev',
    shift_id: 'shift',
    status: 'locked',
    table_id: 'tbl-1',
    table_name: 'Стол 1',
    guest_count: 2,
    subtotal: 1000,
    discount_total: 0,
    tax_total: 100,
    total: 1000,
    opened_at: '2026-05-24T10:00:00Z',
    closed_at: null,
    created_at: '2026-05-24T10:00:00Z',
    updated_at: '2026-05-24T10:00:00Z',
    lines: [{
      id: 'line-1',
      order_id: 'ord-1',
      menu_item_id: 'menu-1',
      catalog_item_id: 'cat-1',
      name: 'Стейк',
      quantity: 2,
      unit_price: 500,
      total_price: 1000,
      currency_code: 'RUB',
      tax_profile_id: null,
      course: '2',
      comment: 'без соли',
      modifiers: [],
      status: 'active',
      created_at: '2026-05-24T10:00:00Z',
      updated_at: '2026-05-24T10:00:00Z',
    }],
  }, null);
  expect(order).toMatchObject({ status: 'precheck_issued', subtotal: 1000, tax: 100, total: 1000 });
  expect(order.lines[0]).toMatchObject({ price: 500, quantity: 2, course: 2 });
});
```

- [ ] **Step 2: Run tests and verify failure**

Run: `cd pos-ui-g; npm test -- backendMappers.test.ts`
Expected: FAIL because `vitest`, `backendMappers` and schemas are not implemented.

- [ ] **Step 3: Implement schemas and mappers**

Create zod schemas matching existing `pos-ui/src/shared/schemas.ts`. Create `mapMenuItem`, `mapTable`, `mapOrder`, `mapClosedOrder`, `mapCashSession`, `mapOperator` and refund payload helpers. These functions only translate backend field names and display shapes; they do not authorize, price, close, refund or mutate domain state.

- [ ] **Step 4: Run tests and verify pass**

Run: `cd pos-ui-g; npm test -- backendMappers.test.ts`
Expected: PASS.

### Task 2: Backend API Client

**Files:**
- Create: `pos-ui-g/src/shared/api.ts`
- Test: `pos-ui-g/src/shared/api.test.ts`

- [ ] **Step 1: Write failing API client tests**

Test that `request()` sends `X-Client-Device-ID`, `X-Node-Device-ID`, `X-Session-ID`, `X-Actor-Employee-ID`, parses backend safe error shape, and times out into a safe `ApiError`.

- [ ] **Step 2: Implement client**

Port endpoint functions from `pos-ui/src/shared/api.ts` into React-neutral functions that accept an explicit auth snapshot instead of reading Pinia. Include auth, shifts, cash sessions, halls, tables, menu, orders, prechecks, payments, checks, refunds and sync endpoints.

- [ ] **Step 3: Run tests**

Run: `cd pos-ui-g; npm test -- api.test.ts`
Expected: PASS.

### Task 3: Backend-Backed POS Context

**Files:**
- Replace: `pos-ui-g/src/context/POSContext.tsx`
- Modify: `pos-ui-g/src/main.tsx`
- Modify: `pos-ui-g/src/components/actions/PaymentDialog.tsx`
- Modify: `pos-ui-g/src/components/actions/PrecheckCancelDialog.tsx`

- [ ] **Step 1: Write failing context-facing tests for command results**

Test pure helpers used by context: payment change display, selected modifier payload conversion, active precheck selection, and permission predicates. These tests must fail before implementation.

- [ ] **Step 2: Implement provider**

Provider owns UI-only state: current section, selected hall/table, theme, local loading/error notices and persisted auth snapshot. Provider fetches pairing/provisioning, auth session, current employee shift, cash session, halls, tables, active orders, menu, prechecks, closed orders and sync diagnostics. Provider sends commands to backend then refetches affected resources. Backend remains source of truth for permission, totals, order locks, payments and refunds.

- [ ] **Step 3: Update dialogs for async commands**

`payOrder` and `cancelPrecheck` return promises. Dialogs await backend response and display only safe UI state from the result or normalized `ApiError`.

- [ ] **Step 4: Run tests**

Run: `cd pos-ui-g; npm test`
Expected: PASS.

### Task 4: UI Wiring and Build Verification

**Files:**
- Modify: `pos-ui-g/src/components/floor/POSFloorSection.tsx`
- Modify: `pos-ui-g/src/components/menu/POSOrderSection.tsx`
- Modify: `pos-ui-g/src/components/cash/POSCashSection.tsx`
- Modify: `pos-ui-g/src/components/activity/POSActivitySection.tsx`
- Modify: `pos-ui-g/src/components/reports/POSReportsSection.tsx`
- Modify: `pos-ui-g/src/shared/i18n/index.ts`

- [ ] **Step 1: Replace mock imports**

Remove direct usage of `mockHalls`, `mockTables`, `mockMenuItems` from runtime components. All runtime data comes through `usePOS()`.

- [ ] **Step 2: Move hardcoded user-facing strings to i18n**

Every new or touched user-facing label, error, empty state, modal title and notification goes through `shared/i18n`.

- [ ] **Step 3: Build**

Run: `cd pos-ui-g; npm run build`
Expected: PASS.

### Task 5: End-to-End Smoke

**Files:**
- No backend runtime code unless an existing endpoint mismatch is found and documented.

- [ ] **Step 1: Start backend and React UI**

Run backend according to existing project scripts, then `cd pos-ui-g; npm run dev`.

- [ ] **Step 2: Manual smoke**

Verify cashier login, open employee shift if needed, open cash shift, select table, create order, add item with modifier, issue precheck, pay cash/card, reprint check, record cash drawer event, view closed order, record full or partial compensation, view sync diagnostics.

- [ ] **Step 3: Final checks**

Run: `cd pos-ui-g; npm run lint; npm run build`
Expected: PASS.

## Scope Boundaries

- `pos-ui` remains untouched except as a reference.
- KDS and Waiter flows are outside current scope.
- Frontend may hide buttons for UX based on permissions, but backend authorization and all financial/order state transitions remain authoritative.
- Runtime backend code is outside scope unless existing route contracts are incompatible with the already documented POS flow.
