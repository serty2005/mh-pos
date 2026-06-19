// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import EdgeSyncPage from './EdgeSyncPage';
import { ru } from '../../shared/i18n/ru';
import {
  buttonByText,
  click,
  deferred,
  edgeEvent,
  expectNoRawMarkers,
  installFetchMock,
  masterDataPackage,
  nodeDeviceId,
  renderPage,
  restaurantDevice,
  restaurantId,
  safeApiError,
  text,
  unassignedDevice,
  waitFor,
} from '../../test/pageHarness';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
});

describe('EdgeSyncPage page integration', () => {
  it('shows loading state while edge device reads are pending', async () => {
    const unassigned = deferred<unknown[]>();
    installFetchMock([
      { path: '/devices/unassigned', responder: () => unassigned.promise },
      { path: `/restaurants/${restaurantId}/devices`, responder: () => [] },
    ]);

    const page = await renderPage(<EdgeSyncPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(text(page.container)).toContain(ru.edge.pageTitle);
      expect(text(page.container)).toContain(ru.edge.refreshing);
    });

    unassigned.resolve([]);
    await page.cleanup();
  });

  it('renders empty state when there are no devices, events, or packages', async () => {
    installFetchMock([
      { path: '/devices/unassigned', responder: () => [] },
      { path: `/restaurants/${restaurantId}/devices`, responder: () => [] },
    ]);

    const page = await renderPage(<EdgeSyncPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(text(page.container)).toContain(ru.edge.emptyDevicesTitle);
      expect(text(page.container)).toContain(ru.edge.emptyRestaurantDevicesTitle);
      expect(text(page.container)).toContain(ru.edge.noDeviceSelected);
    });

    await page.cleanup();
  });

  it('renders bounded event and package projections without raw payload values', async () => {
    const api = installFetchMock([
      { path: '/devices/unassigned', responder: () => [] },
      { path: `/restaurants/${restaurantId}/devices`, responder: () => [restaurantDevice()] },
      { path: `/sync/edge-events?restaurant_id=${restaurantId}&limit=50&device_id=${nodeDeviceId}`, responder: () => [edgeEvent()] },
      {
        path: /\/provisioning\/master-data\/catalog\?node_device_id=edge-node-1/,
        responder: () => masterDataPackage({ payload_json: { items: [{ secret: 'raw payload marker' }] } }),
      },
    ]);

    const page = await renderPage(<EdgeSyncPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(text(page.container)).toContain('order.closed');
      expect(text(page.container)).toContain('catalog');
      expect(text(page.container)).toContain('items: 1');
    });
    expect(api.callsFor(/\/provisioning\/master-data\//)).toHaveLength(7);
    expectNoRawMarkers(page.container);

    await page.cleanup();
  });

  it('renders safe errors without raw sync payload markers', async () => {
    installFetchMock([
      { path: '/devices/unassigned', responder: () => { throw safeApiError(); } },
      { path: `/restaurants/${restaurantId}/devices`, responder: () => [] },
    ]);

    const page = await renderPage(<EdgeSyncPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(text(page.container)).toContain(ru.errors.server);
      expect(page.container.querySelector('[role="alert"]')).not.toBeNull();
    });
    expectNoRawMarkers(page.container);

    await page.cleanup();
  });

  it('assigns a selected device once, disables duplicate clicks, and reloads lists after success', async () => {
    const assignPending = deferred<unknown>();
    let unassignedReads = 0;
    let restaurantDeviceReads = 0;
    const api = installFetchMock([
      {
        path: '/devices/unassigned',
        responder: () => {
          unassignedReads += 1;
          return [unassignedDevice()];
        },
      },
      {
        path: `/restaurants/${restaurantId}/devices`,
        responder: () => {
          restaurantDeviceReads += 1;
          return restaurantDeviceReads > 1 ? [restaurantDevice()] : [];
        },
      },
      {
        method: 'POST',
        path: `/restaurants/${restaurantId}/devices/${nodeDeviceId}/assign`,
        responder: () => assignPending.promise,
      },
      {
        path: `/devices/${nodeDeviceId}/assignment-status`,
        responder: () => ({
          node_device_id: nodeDeviceId,
          status: 'assigned',
          restaurant_id: restaurantId,
          cloud_url: 'https://cloud.example.test',
          snapshot_url: 'https://cloud.example.test/snapshot',
        }),
      },
    ]);

    const page = await renderPage(<EdgeSyncPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(page.container)).toContain('Front POS'));

    const radio = page.container.querySelector('input[type="radio"]');
    expect(radio).not.toBeNull();
    await click(radio!);

    const assignButton = buttonByText(page.container, ru.edge.assignAction);
    await click(assignButton);
    await click(assignButton);

    await waitFor(() => {
      expect(api.callsFor(`/restaurants/${restaurantId}/devices/${nodeDeviceId}/assign`, 'POST')).toHaveLength(1);
      expect((buttonByText(page.container, ru.edge.refreshing) as HTMLButtonElement).disabled).toBe(true);
    });

    assignPending.resolve({
      node_device_id: nodeDeviceId,
      restaurant_id: restaurantId,
      status: 'assigned',
      snapshot_url: 'https://cloud.example.test/snapshot',
    });

    await waitFor(() => {
      expect(unassignedReads).toBe(2);
      expect(restaurantDeviceReads).toBe(2);
      expect(text(page.container)).toContain(ru.edge.status.assigned);
    });

    await page.cleanup();
  });
});
