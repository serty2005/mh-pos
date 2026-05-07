import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError, listMenuItems } from './api';

const authState = {
  clientDeviceId: 'client-1',
  nodeDeviceId: 'node-1',
  sessionId: 'session-1',
  actor: { employee_id: 'employee-1' },
};

vi.mock('../stores/auth', () => ({
  useAuthStore: () => authState,
}));

describe('api request helpers', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('sends required auth/device headers for runtime reads', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => '[]',
    });
    vi.stubGlobal('fetch', fetchMock);

    await listMenuItems();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    const headers = new Headers(init.headers);
    expect(headers.get('X-Client-Device-ID')).toBe('client-1');
    expect(headers.get('X-Node-Device-ID')).toBe('node-1');
    expect(headers.get('X-Session-ID')).toBe('session-1');
    expect(headers.get('X-Actor-Employee-ID')).toBe('employee-1');
  });

  it('maps backend error payload into ApiError', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 429,
      statusText: 'Too Many Requests',
      text: async () => JSON.stringify({ error: 'too many requests: retry later' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    let thrown: unknown;
    try {
      await listMenuItems();
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toBeInstanceOf(ApiError);
    expect(thrown).toMatchObject({ status: 429, message: 'too many requests: retry later' });
  });
});
