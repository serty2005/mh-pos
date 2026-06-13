import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { t } from '../../shared/i18n';
import { permissions } from '../../types';

const usePOSMock = vi.fn();

vi.mock('../../context/POSContext', () => ({
  usePOS: () => usePOSMock(),
}));

describe('POSKitchenSection', () => {
  beforeEach(() => {
    usePOSMock.mockReset();
    usePOSMock.mockReturnValue({
      authSnapshot: {
        clientDeviceId: 'client-1',
        nodeDeviceId: 'node-1',
        restaurantId: 'restaurant-1',
        sessionId: 'session-1',
        actorEmployeeId: 'employee-1',
      },
      currentOperator: {
        id: 'shift-1',
        employeeName: 'Chef',
        role: 'kitchen',
        permissions: [
          permissions.CATALOG_VIEW,
          permissions.KITCHEN_VIEW,
          permissions.KITCHEN_STATUS_CHANGE,
          permissions.KITCHEN_RECIPE_VIEW,
          permissions.KITCHEN_RECIPE_SUGGEST,
          permissions.KITCHEN_CATALOG_SUGGEST,
          permissions.KITCHEN_STOCK_RECEIPT,
          permissions.KITCHEN_STOCK_INVENTORY_COUNT,
          permissions.KITCHEN_STOCK_WRITE_OFF,
          permissions.KITCHEN_PRODUCTION_COMPLETE,
          permissions.KITCHEN_STOP_LIST_VIEW,
          permissions.KITCHEN_STOP_LIST_UPDATE,
        ],
        openTime: '2026-05-24T10:00:00Z',
        status: 'open',
      },
    });
  });

  it('renders order queue and ready tabs', async () => {
    const { POSKitchenSection } = await import('./POSKitchenSection');
    const html = renderToStaticMarkup(<POSKitchenSection section="orders" />);

    expect(html).toContain('tab-queue');
    expect(html).toContain('tab-ready');
    expect(html).toContain(t.kitchen.title);
  });

  it('renders stock capture tabs', async () => {
    const { POSKitchenSection } = await import('./POSKitchenSection');
    const html = renderToStaticMarkup(<POSKitchenSection section="stock" />);

    expect(html).toContain('tab-receipt');
    expect(html).toContain('tab-count');
    expect(html).toContain('tab-writeoff');
    expect(html).toContain('tab-production');
  });

  it('renders recipe, stop-list and proposal tabs', async () => {
    const { POSKitchenSection } = await import('./POSKitchenSection');
    const html = renderToStaticMarkup(<POSKitchenSection section="kitchen" />);

    expect(html).toContain('tab-recipes');
    expect(html).toContain('tab-suggestions');
    expect(html).toContain('tab-stop_list');
    expect(html).toContain('tab-my_proposals');
    expect(html).toContain(t.kitchen.loadRecipe);
  });

  it('shows no-permission state without kitchen permissions', async () => {
    usePOSMock.mockReturnValue({
      authSnapshot: {
        clientDeviceId: 'client-1',
        nodeDeviceId: 'node-1',
        restaurantId: 'restaurant-1',
        sessionId: 'session-1',
        actorEmployeeId: 'employee-1',
      },
      currentOperator: {
        id: 'shift-1',
        employeeName: 'Support',
        role: 'support',
        permissions: [permissions.SYNC_VIEW, permissions.SYNC_RETRY],
        openTime: '2026-05-24T10:00:00Z',
        status: 'open',
      },
    });

    const { POSKitchenSection } = await import('./POSKitchenSection');
    const html = renderToStaticMarkup(<POSKitchenSection section="orders" />);

    expect(html).toContain(t.errors.noPermission);
    expect(html).not.toContain('tab-queue');
  });
});
