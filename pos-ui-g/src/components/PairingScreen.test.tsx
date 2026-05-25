import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const usePOSMock = vi.fn();

vi.mock('../context/POSContext', () => ({
  usePOS: () => usePOSMock(),
}));

describe('PairingScreen', () => {
  beforeEach(() => {
    usePOSMock.mockReset();
  });

  it('renders cloud registration and license pairing controls', async () => {
    usePOSMock.mockReturnValue({
      authSnapshot: {
        clientDeviceId: 'client-1',
        nodeDeviceId: 'node-1',
        restaurantId: '',
        sessionId: '',
        actorEmployeeId: '',
      },
      provisioningStatus: {
        node_device_id: 'node-1',
        cloud_url: 'http://cloud.local',
        status: 'pending_admin_approval',
        paired: false,
      },
      provisioningLoading: false,
      provisioningError: '',
      refreshProvisioningStatus: vi.fn(),
      registerCloudProvisioning: vi.fn(),
      pairViaLicense: vi.fn(),
    });

    const { PairingScreen } = await import('./PairingScreen');
    const html = renderToStaticMarkup(<PairingScreen />);

    expect(html).toContain('pair-register-cloud-btn');
    expect(html).toContain('pair-mode-license-btn');
    expect(html).toContain('http://cloud.local');
  });
});
