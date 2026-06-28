import { describe, expect, it } from 'vitest';

import { permissions, type EmployeeShift } from '../types';
import { mapMenuItem, mapOrder, mapTable, roleFromPermissions } from './backendMappers';
import type { BackendMenuItem, BackendOrder, BackendTable } from './schemas';

describe('backend DTO mappers', () => {
  it('maps backend table and active order to design table state', () => {
    const table: BackendTable = {
      id: 'tbl-1',
      restaurant_id: 'r1',
      hall_id: 'hall-1',
      name: 'Стол 7',
      seats: 4,
      active: true,
    };
    const order: BackendOrder = {
      id: 'ord-1',
      edge_order_id: 'edge-1',
      restaurant_id: 'r1',
      device_id: 'dev-1',
      shift_id: 'shift-1',
      status: 'open',
      table_id: 'tbl-1',
      table_name: 'Стол 7',
      guest_count: 3,
      subtotal: 1200,
      discount_total: 0,
      tax_total: 120,
      total: 1200,
      opened_at: '2026-05-24T10:00:00Z',
      closed_at: null,
      created_at: '2026-05-24T10:00:00Z',
      updated_at: '2026-05-24T10:00:00Z',
      lines: [],
    };

    expect(mapTable(table, order)).toMatchObject({
      id: 'tbl-1',
      number: 7,
      hallId: 'hall-1',
      status: 'occupied',
      guestsCount: 3,
      activeOrderSum: 1200,
      currentOrderId: 'ord-1',
    });
  });

  it('maps backend modifier groups to design menu item modifiers', () => {
    const item: BackendMenuItem = {
      id: 'menu-1',
      catalog_item_id: 'cat-1',
      item_type: 'dish',
      name: 'Стейк',
      price: 1450,
      currency: 'RUB',
      single_unit_per_line: false,
      active: true,
      created_at: '2026-05-24T10:00:00Z',
      updated_at: '2026-05-24T10:00:00Z',
      modifier_groups: [
        {
          id: 'grp-1',
          restaurant_id: 'r1',
          name: 'Прожарка',
          required: true,
          min_count: 1,
          max_count: 1,
          active: true,
          options: [
            {
              id: 'opt-1',
              restaurant_id: 'r1',
              modifier_group_id: 'grp-1',
              name: 'Medium',
              price_minor: 0,
              active: true,
            },
          ],
        },
      ],
    };

    expect(mapMenuItem(item)).toMatchObject({
      id: 'menu-1',
      name: 'Стейк',
      price: 1450,
      isAvailable: true,
      modifierGroups: [
        {
          id: 'grp-1',
          name: 'Прожарка',
          minRequired: 1,
          maxAllowed: 1,
          options: [{ id: 'opt-1', name: 'Medium', price: 0 }],
        },
      ],
    });
  });

  it('maps backend order lines without calculating business totals', () => {
    const order: BackendOrder = {
      id: 'ord-1',
      edge_order_id: 'edge-1',
      restaurant_id: 'r1',
      device_id: 'dev-1',
      shift_id: 'shift-1',
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
      lines: [
        {
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
        },
      ],
    };

    const mapped = mapOrder(order, null);

    expect(mapped).toMatchObject({
      id: 'ord-1',
      status: 'precheck_issued',
      subtotal: 1000,
      tax: 100,
      total: 1000,
    });
    expect(mapped.lines[0]).toMatchObject({
      id: 'line-1',
      price: 500,
      quantity: 2,
      course: 2,
      comment: 'без соли',
    });
  });

  it('derives POS UI roles from backend permissions', () => {
    const cases: Array<{ expected: EmployeeShift['role']; permissions: string[] }> = [
      { expected: 'waiter', permissions: [permissions.ORDER_VIEW, permissions.PRECHECK_ISSUE] },
      { expected: 'cashier', permissions: [permissions.ORDER_VIEW, permissions.PAYMENT_CASH] },
      { expected: 'manager', permissions: [permissions.ORDER_VIEW, permissions.PAYMENT_CASH, permissions.PAYMENT_REFUND] },
      { expected: 'kitchen', permissions: [permissions.KITCHEN_VIEW, permissions.KITCHEN_STOP_LIST_UPDATE] },
      { expected: 'support', permissions: [permissions.SYNC_VIEW, permissions.SYNC_RETRY] },
    ];

    cases.forEach((item) => {
      expect(roleFromPermissions(item.permissions)).toBe(item.expected);
    });
  });
});
