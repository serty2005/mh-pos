import { beforeEach, describe, expect, it, vi } from 'vitest';

import { addOrderLine, ApiError, listMenuItems } from './api';

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
      headers: new Headers({ 'X-Request-ID': 'req-1' }),
      text: async () => JSON.stringify({
        error: {
          code: 'RATE_LIMITED',
          message_key: 'errors.rateLimit',
          correlation_id: 'req-1',
        },
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    let thrown: unknown;
    try {
      await listMenuItems();
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toBeInstanceOf(ApiError);
    expect(thrown).toMatchObject({
      status: 429,
      code: 'RATE_LIMITED',
      messageKey: 'errors.rateLimit',
      category: 'rate_limit',
      correlationId: 'req-1',
    });
  });

  it('maps network failure without clearing client identity in api layer', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('failed to fetch')));

    await expect(listMenuItems()).rejects.toMatchObject({
      code: 'NETWORK_ERROR',
      messageKey: 'errors.network.unavailable',
      category: 'network',
      retryable: true,
    });
  });

  it('sends selected modifiers when adding an order line', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify({
        id: 'line-1',
        order_id: 'order-1',
        menu_item_id: 'menu-1',
        catalog_item_id: 'catalog-1',
        name: 'Soup',
        quantity: 1,
        unit_price: 1000,
        total_price: 1250,
        currency_code: 'RUB',
        tax_profile_id: null,
        modifiers: [{
          id: 'line-modifier-1',
          order_line_id: 'line-1',
          modifier_group_id: 'group-1',
          modifier_option_id: 'option-1',
          name: 'Hot',
          quantity: 1,
          unit_price: 250,
          total_price: 250,
        }],
        status: 'active',
        created_at: '2026-05-09T10:00:00Z',
        updated_at: '2026-05-09T10:00:00Z',
      }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await addOrderLine('order-1', 'menu-1', 1, [{ modifier_group_id: 'group-1', modifier_option_id: 'option-1', quantity: 1 }]);

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toMatchObject({
      menu_item_id: 'menu-1',
      quantity: 1,
      selected_modifiers: [{ modifier_group_id: 'group-1', modifier_option_id: 'option-1', quantity: 1 }],
    });
  });
});
