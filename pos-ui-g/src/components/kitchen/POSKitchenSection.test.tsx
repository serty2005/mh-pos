import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { t } from '../../shared/i18n';

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

  it('renders recipe and proposal tabs', async () => {
    const { POSKitchenSection } = await import('./POSKitchenSection');
    const html = renderToStaticMarkup(<POSKitchenSection section="kitchen" />);

    expect(html).toContain('tab-recipes');
    expect(html).toContain('tab-suggestions');
    expect(html).toContain('tab-my_proposals');
    expect(html).toContain(t.kitchen.loadRecipe);
  });
});
