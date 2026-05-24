import { afterEach, describe, expect, it, vi } from 'vitest';

import { ApiError, createApiClient, type AuthSnapshot } from './api';

const auth: AuthSnapshot = {
  clientDeviceId: 'client-1',
  nodeDeviceId: 'node-1',
  restaurantId: 'restaurant-1',
  sessionId: 'session-1',
  actorEmployeeId: 'employee-1',
};

describe('POS API client', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('sends POS identity headers with requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('[]', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');
    await api.listMenuItems();

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Headers;
    expect(fetchMock.mock.calls[0][0]).toBe('http://pos.local/api/v1/menu/items');
    expect(headers.get('X-Client-Device-ID')).toBe('client-1');
    expect(headers.get('X-Node-Device-ID')).toBe('node-1');
    expect(headers.get('X-Session-ID')).toBe('session-1');
    expect(headers.get('X-Actor-Employee-ID')).toBe('employee-1');
  });

  it('normalizes backend safe error response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: {
        code: 'PERMISSION_DENIED',
        message_key: 'errors.permission',
        correlation_id: 'req-1',
      },
    }), { status: 403 })));

    const api = createApiClient(() => auth, 'http://pos.local/api/v1');

    await expect(api.listMenuItems()).rejects.toMatchObject({
      code: 'PERMISSION_DENIED',
      messageKey: 'errors.permission',
      category: 'permission',
      correlationId: 'req-1',
    });
  });
});
