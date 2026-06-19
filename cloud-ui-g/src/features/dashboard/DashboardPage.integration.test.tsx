// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import DashboardPage from './DashboardPage';
import PublicationPanel from '../publications/PublicationPanel';
import { ru } from '../../shared/i18n/ru';
import {
  buttonByText,
  change,
  click,
  deferred,
  employee,
  expectNoRawMarkers,
  hall,
  installFetchMock,
  publication,
  renderPage,
  restaurantDevice,
  restaurantId,
  role,
  safeApiError,
  table,
  text,
  waitFor,
} from '../../test/pageHarness';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
});

function readinessRoutes(overrides: Record<string, unknown> = {}) {
  const defaults: Record<string, unknown> = {
    [`/master-data/roles?restaurant_id=${restaurantId}`]: [role()],
    [`/master-data/employees?restaurant_id=${restaurantId}`]: [employee()],
    [`/master-data/floor/halls?restaurant_id=${restaurantId}`]: [hall()],
    [`/master-data/floor/tables?restaurant_id=${restaurantId}`]: [table()],
    [`/master-data/catalog/items?restaurant_id=${restaurantId}`]: [],
    [`/master-data/menu/items?restaurant_id=${restaurantId}`]: [],
    [`/master-data/modifiers/groups?restaurant_id=${restaurantId}`]: [],
    [`/master-data/modifiers/options?restaurant_id=${restaurantId}`]: [],
    [`/master-data/modifiers/bindings?restaurant_id=${restaurantId}`]: [],
    [`/master-data/pricing/policies?restaurant_id=${restaurantId}`]: [],
    [`/restaurants/${restaurantId}/devices`]: [restaurantDevice()],
  };

  return Object.entries({ ...defaults, ...overrides }).map(([path, response]) => ({
    path,
    responder: () => {
      if (response && typeof response === 'object' && 'throwError' in response) throw safeApiError();
      return response;
    },
  }));
}

describe('DashboardPage page integration', () => {
  it('shows a stable loading skeleton while readiness API calls are pending', async () => {
    const roles = deferred<unknown[]>();
    installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => null },
      ...readinessRoutes({ [`/master-data/roles?restaurant_id=${restaurantId}`]: roles.promise }),
    ]);

    const page = await renderPage(<DashboardPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(page.container.querySelector('[aria-busy="true"]')).not.toBeNull();
      expect(text(page.container)).toContain(ru.dashboard.pageTitle);
    });

    roles.resolve([]);
    await page.cleanup();
  });

  it('renders empty readiness and publication states without crashing', async () => {
    installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => null },
      ...readinessRoutes({
        [`/master-data/roles?restaurant_id=${restaurantId}`]: [],
        [`/master-data/employees?restaurant_id=${restaurantId}`]: [],
        [`/master-data/floor/halls?restaurant_id=${restaurantId}`]: [],
        [`/master-data/floor/tables?restaurant_id=${restaurantId}`]: [],
        [`/restaurants/${restaurantId}/devices`]: [],
      }),
    ]);

    const page = await renderPage(<DashboardPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(text(page.container)).toContain(ru.dashboard.readiness.pending);
      expect(text(page.container)).toContain(ru.publications.emptyTitle);
    });

    await page.cleanup();
  });

  it('renders safe summary fields after successful reads', async () => {
    installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => publication({ version: 11 }) },
      ...readinessRoutes(),
    ]);

    const page = await renderPage(<DashboardPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(text(page.container)).toContain(ru.dashboard.kpis.publicationReady);
      expect(text(page.container)).toContain('11');
      expect(text(page.container)).toContain('5/8');
    });

    await page.cleanup();
  });

  it('renders safe errors without backend internals', async () => {
    installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => { throw safeApiError(); } },
      ...readinessRoutes({ [`/master-data/roles?restaurant_id=${restaurantId}`]: { throwError: true } }),
    ]);

    const page = await renderPage(<DashboardPage restaurantId={restaurantId} />);

    await waitFor(() => {
      expect(page.container.querySelector('[role="alert"]')).not.toBeNull();
      expect(text(page.container)).toContain(ru.errors.server);
    });
    expectNoRawMarkers(page.container);

    await page.cleanup();
  });
});

describe('PublicationPanel page integration', () => {
  it('keeps the page stable while current publication read is pending', async () => {
    const current = deferred<unknown>();
    const api = installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => current.promise },
    ]);

    const page = await renderPage(<PublicationPanel restaurantId={restaurantId} canPublish />);

    await waitFor(() => {
      expect(api.callsFor(`/restaurants/${restaurantId}/master-data/publication-state`)).toHaveLength(1);
      expect(text(page.container)).toContain(ru.publications.title);
    });

    current.resolve(null);
    await page.cleanup();
  });

  it('renders empty, current publication, and safe error states', async () => {
    installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => null },
    ]);
    const empty = await renderPage(<PublicationPanel restaurantId={restaurantId} canPublish />);
    await waitFor(() => expect(text(empty.container)).toContain(ru.publications.emptyTitle));
    await empty.cleanup();

    installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => publication({ version: 42 }) },
    ]);
    const success = await renderPage(<PublicationPanel restaurantId={restaurantId} canPublish />);
    await waitFor(() => {
      expect(text(success.container)).toContain('42');
      expect(text(success.container)).toContain('cloud-ui-test');
    });
    await success.cleanup();

    installFetchMock([
      { path: `/restaurants/${restaurantId}/master-data/publication-state`, responder: () => { throw safeApiError(); } },
    ]);
    const failed = await renderPage(<PublicationPanel restaurantId={restaurantId} canPublish />);
    await waitFor(() => expect(text(failed.container)).toContain(ru.errors.server));
    expectNoRawMarkers(failed.container);
    await failed.cleanup();
  });

  it('publishes successfully with operator payload and refreshes current publication', async () => {
    let currentReads = 0;
    const api = installFetchMock([
      {
        path: `/restaurants/${restaurantId}/master-data/publication-state`,
        responder: () => {
          currentReads += 1;
          return currentReads > 1 ? publication({ version: 12, published_by: 'operator-1' }) : null;
        },
      },
      {
        method: 'POST',
        path: `/restaurants/${restaurantId}/master-data/publish`,
        responder: () => publication({ version: 12, published_by: 'operator-1' }),
      },
    ]);

    const page = await renderPage(<PublicationPanel restaurantId={restaurantId} canPublish />);
    await waitFor(() => expect(currentReads).toBe(1));

    const inputs = page.container.querySelectorAll('input');
    await change(inputs[0], 'operator-1');
    await change(inputs[1], 'edge-node-1');
    await click(buttonByText(page.container, ru.publications.publishAction));

    await waitFor(() => {
      expect(api.callsFor(`/restaurants/${restaurantId}/master-data/publish`, 'POST')).toHaveLength(1);
      expect(currentReads).toBe(2);
      expect(text(page.container)).toContain('12');
    });
    expect(api.callsFor(`/restaurants/${restaurantId}/master-data/publish`, 'POST')[0].body).toEqual({
      published_by: 'operator-1',
      node_device_id: 'edge-node-1',
    });

    await page.cleanup();
  });

  it('keeps failed publish visible without duplicate mutation or automatic retry', async () => {
    let currentReads = 0;
    const publishPending = deferred<unknown>();
    const api = installFetchMock([
      {
        path: `/restaurants/${restaurantId}/master-data/publication-state`,
        responder: () => {
          currentReads += 1;
          return null;
        },
      },
      {
        method: 'POST',
        path: `/restaurants/${restaurantId}/master-data/publish`,
        responder: () => publishPending.promise,
      },
    ]);

    const page = await renderPage(<PublicationPanel restaurantId={restaurantId} canPublish />);
    await waitFor(() => expect(currentReads).toBe(1));

    const inputs = page.container.querySelectorAll('input');
    await change(inputs[0], 'operator-1');
    await change(inputs[1], 'edge-node-1');
    await click(buttonByText(page.container, ru.publications.publishAction));
    await waitFor(() => {
      expect(api.callsFor(`/restaurants/${restaurantId}/master-data/publish`, 'POST')).toHaveLength(1);
      expect((buttonByText(page.container, ru.publications.publishAction) as HTMLButtonElement).disabled).toBe(true);
    });
    await click(buttonByText(page.container, ru.publications.publishAction));
    publishPending.reject(safeApiError());

    await waitFor(() => {
      expect(api.callsFor(`/restaurants/${restaurantId}/master-data/publish`, 'POST')).toHaveLength(1);
      expect(text(page.container)).toContain(ru.errors.server);
    });
    expect(api.callsFor(`/restaurants/${restaurantId}/master-data/publish`, 'POST')[0].body).toEqual({
      published_by: 'operator-1',
      node_device_id: 'edge-node-1',
    });
    expect(currentReads).toBe(1);
    expectNoRawMarkers(page.container);

    await page.cleanup();
  });
});
