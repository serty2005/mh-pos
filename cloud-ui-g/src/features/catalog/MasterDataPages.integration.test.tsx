// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import CatalogPage from './CatalogPage';
import MenuPage from '../menu/MenuPage';
import PricingPage from '../pricing/PricingPage';
import FloorPage from '../floor/FloorPage';
import { ru } from '../../shared/i18n/ru';
import {
  buttonByText,
  catalogFolder,
  catalogItem,
  change,
  click,
  deferred,
  expectNoRawMarkers,
  hall,
  inputByPlaceholder,
  installFetchMock,
  menuItem,
  pricingPolicy,
  renderPage,
  restaurantId,
  safeApiError,
  table,
  text,
  waitFor,
} from '../../test/pageHarness';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
  vi.restoreAllMocks();
});

describe('CatalogPage page integration', () => {
  it('shows loading, empty, success with missing optional fields, and safe error states', async () => {
    const items = deferred<unknown[]>();
    installFetchMock([
      { path: `/master-data/catalog/items?restaurant_id=${restaurantId}`, responder: () => items.promise },
      { path: `/master-data/catalog/folders?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/folder-parameters?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/tags?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const loading = await renderPage(<CatalogPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(loading.container)).toContain(ru.status.loading));
    items.resolve([]);
    await loading.cleanup();

    installFetchMock([
      { path: `/master-data/catalog/items?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/folders?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/folder-parameters?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/tags?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const empty = await renderPage(<CatalogPage restaurantId={restaurantId} />);
    await waitFor(() => {
      expect(text(empty.container)).toContain(ru.catalog.items.empty);
      expect(text(empty.container)).toContain(ru.catalog.folders.empty);
    });
    await empty.cleanup();

    installFetchMock([
      {
        path: `/master-data/catalog/items?restaurant_id=${restaurantId}`,
        responder: () => [{ ...catalogItem(), folder_id: undefined, kitchen_type: undefined, accounting_category: undefined }],
      },
      { path: `/master-data/catalog/folders?restaurant_id=${restaurantId}`, responder: () => [catalogFolder()] },
      { path: `/master-data/catalog/folder-parameters?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/tags?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const success = await renderPage(<CatalogPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(success.container)).toContain('Tea'));
    await success.cleanup();

    installFetchMock([
      { path: `/master-data/catalog/items?restaurant_id=${restaurantId}`, responder: () => { throw safeApiError(); } },
      { path: `/master-data/catalog/folders?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/folder-parameters?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/tags?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const failed = await renderPage(<CatalogPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(failed.container)).toContain(ru.catalog.blockedTitle));
    expectNoRawMarkers(failed.container);
    await failed.cleanup();
  });

  it('creates catalog item once and refreshes all catalog lists after success', async () => {
    let itemReads = 0;
    const api = installFetchMock([
      {
        path: `/master-data/catalog/items?restaurant_id=${restaurantId}`,
        responder: () => {
          itemReads += 1;
          return itemReads > 1 ? [catalogItem({ id: 'catalog-coffee', name: 'Coffee', sku: 'COF' })] : [];
        },
      },
      { path: `/master-data/catalog/folders?restaurant_id=${restaurantId}`, responder: () => [catalogFolder()] },
      { path: `/master-data/catalog/folder-parameters?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/catalog/tags?restaurant_id=${restaurantId}`, responder: () => [] },
      { method: 'POST', path: '/master-data/catalog/items', responder: () => catalogItem({ id: 'catalog-coffee', name: 'Coffee', sku: 'COF' }) },
    ]);

    const page = await renderPage(<CatalogPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(page.container)).toContain(ru.catalog.items.empty));

    const itemPanel = page.container.querySelector('section section');
    expect(itemPanel).not.toBeNull();
    const inputs = itemPanel!.querySelectorAll('input');
    await change(inputs[0], 'Coffee');
    await change(inputs[1], 'COF');
    await change(inputs[2], 'portion');
    await click(buttonByText(itemPanel!, ru.catalog.items.actions.create));
    await click(buttonByText(itemPanel!, ru.catalog.items.actions.create));

    await waitFor(() => {
      expect(api.callsFor('/master-data/catalog/items', 'POST')).toHaveLength(1);
      expect(itemReads).toBe(2);
      expect(text(page.container)).toContain('Coffee');
    });
    expect(api.callsFor('/master-data/catalog/items', 'POST')[0].body).toMatchObject({
      restaurant_id: restaurantId,
      kind: 'dish',
      folder_id: '',
      name: 'Coffee',
      sku: 'COF',
      base_unit: 'portion',
    });

    await page.cleanup();
  });
});

describe('MenuPage page integration', () => {
  it('covers loading, empty, success filter for archived catalog items, and safe error states', async () => {
    const menu = deferred<unknown[]>();
    installFetchMock([
      { path: `/master-data/menu/items?restaurant_id=${restaurantId}`, responder: () => menu.promise },
      { path: `/master-data/catalog/items?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const loading = await renderPage(<MenuPage restaurantId={restaurantId} restaurantCurrency="RUB" />);
    await waitFor(() => expect(text(loading.container)).toContain(ru.status.loading));
    menu.resolve([]);
    await loading.cleanup();

    installFetchMock([
      { path: `/master-data/menu/items?restaurant_id=${restaurantId}`, responder: () => [] },
      {
        path: `/master-data/catalog/items?restaurant_id=${restaurantId}`,
        responder: () => [catalogItem(), catalogItem({ id: 'catalog-old', name: 'Archived dish', status: 'archived' })],
      },
    ]);
    const empty = await renderPage(<MenuPage restaurantId={restaurantId} restaurantCurrency="RUB" />);
    await waitFor(() => expect(text(empty.container)).toContain(ru.menu.items.empty));
    const catalogOptions = Array.from(empty.container.querySelectorAll('select')[0].querySelectorAll('option')).map((option) => option.textContent);
    expect(catalogOptions).toContain('Tea');
    expect(catalogOptions).not.toContain('Archived dish');
    await empty.cleanup();

    installFetchMock([
      { path: `/master-data/menu/items?restaurant_id=${restaurantId}`, responder: () => [menuItem()] },
      { path: `/master-data/catalog/items?restaurant_id=${restaurantId}`, responder: () => [catalogItem()] },
    ]);
    const success = await renderPage(<MenuPage restaurantId={restaurantId} restaurantCurrency="RUB" />);
    await waitFor(() => expect(text(success.container)).toContain('Tea service'));
    await success.cleanup();

    installFetchMock([
      { path: `/master-data/menu/items?restaurant_id=${restaurantId}`, responder: () => { throw safeApiError(); } },
      { path: `/master-data/catalog/items?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const failed = await renderPage(<MenuPage restaurantId={restaurantId} restaurantCurrency="RUB" />);
    await waitFor(() => expect(text(failed.container)).toContain(ru.menu.blockedTitle));
    expectNoRawMarkers(failed.container);
    await failed.cleanup();
  });

  it('creates menu item with restaurant scope, linked catalog item, and refresh after success', async () => {
    let menuReads = 0;
    const api = installFetchMock([
      {
        path: `/master-data/menu/items?restaurant_id=${restaurantId}`,
        responder: () => {
          menuReads += 1;
          return menuReads > 1 ? [menuItem({ id: 'menu-coffee', name: 'Coffee menu' })] : [];
        },
      },
      { path: `/master-data/catalog/items?restaurant_id=${restaurantId}`, responder: () => [catalogItem()] },
      { method: 'POST', path: '/master-data/menu/items', responder: () => menuItem({ id: 'menu-coffee', name: 'Coffee menu' }) },
    ]);

    const page = await renderPage(<MenuPage restaurantId={restaurantId} restaurantCurrency="RUB" />);
    await waitFor(() => expect(text(page.container)).toContain(ru.menu.items.empty));

    const selects = page.container.querySelectorAll('select');
    const inputs = page.container.querySelectorAll('input');
    await change(selects[0], 'catalog-tea');
    await change(inputs[0], 'Coffee menu');
    await change(page.container.querySelector('input[type="number"]')!, '250');
    await change(Array.from(inputs).find((input) => input.value === 'RUB')!, 'rub');
    await click(buttonByText(page.container, ru.menu.items.actions.create));

    await waitFor(() => {
      expect(api.callsFor('/master-data/menu/items', 'POST')).toHaveLength(1);
      expect(menuReads).toBe(2);
      expect(text(page.container)).toContain('Coffee menu');
    });
    expect(api.callsFor('/master-data/menu/items', 'POST')[0].body).toMatchObject({
      restaurant_id: restaurantId,
      catalog_item_id: 'catalog-tea',
      name: 'Coffee menu',
      price: 250,
      currency: 'RUB',
    });

    await page.cleanup();
  });
});

describe('PricingPage page integration', () => {
  it('covers loading, empty, success, and safe SQL error states', async () => {
    const policies = deferred<unknown[]>();
    installFetchMock([
      { path: `/master-data/pricing/policies?restaurant_id=${restaurantId}`, responder: () => policies.promise },
    ]);
    const loading = await renderPage(<PricingPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(loading.container)).toContain(ru.status.loading));
    policies.resolve([]);
    await loading.cleanup();

    installFetchMock([
      { path: `/master-data/pricing/policies?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const empty = await renderPage(<PricingPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(empty.container)).toContain(ru.pricing.policies.empty));
    await empty.cleanup();

    installFetchMock([
      { path: `/master-data/pricing/policies?restaurant_id=${restaurantId}`, responder: () => [pricingPolicy()] },
    ]);
    const success = await renderPage(<PricingPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(success.container)).toContain('Service charge'));
    await success.cleanup();

    installFetchMock([
      { path: `/master-data/pricing/policies?restaurant_id=${restaurantId}`, responder: () => { throw safeApiError(); } },
    ]);
    const failed = await renderPage(<PricingPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(failed.container)).toContain(ru.pricing.blockedTitle));
    expectNoRawMarkers(failed.container);
    await failed.cleanup();
  });

  it('creates percentage pricing policy with normalized values and refresh after success', async () => {
    let reads = 0;
    const api = installFetchMock([
      {
        path: `/master-data/pricing/policies?restaurant_id=${restaurantId}`,
        responder: () => {
          reads += 1;
          return reads > 1 ? [pricingPolicy({ id: 'policy-happy-hour', name: 'Happy hour' })] : [];
        },
      },
      { method: 'POST', path: '/master-data/pricing/policies', responder: () => pricingPolicy({ id: 'policy-happy-hour', name: 'Happy hour' }) },
    ]);

    const page = await renderPage(<PricingPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(page.container)).toContain(ru.pricing.policies.empty));

    await change(inputByPlaceholder(page.container, ru.pricing.policies.fields.name), 'Happy hour');
    const inputs = page.container.querySelectorAll('input');
    await change(inputs[3], '1250');
    await change(inputs[4], '30');
    await change(inputByPlaceholder(page.container, ru.pricing.policies.fields.permission), 'pos.pricing.discount.apply');
    await click(buttonByText(page.container, ru.pricing.policies.actions.create));

    await waitFor(() => {
      expect(api.callsFor('/master-data/pricing/policies', 'POST')).toHaveLength(1);
      expect(reads).toBe(2);
      expect(text(page.container)).toContain('Happy hour');
    });
    expect(api.callsFor('/master-data/pricing/policies', 'POST')[0].body).toMatchObject({
      restaurant_id: restaurantId,
      name: 'Happy hour',
      kind: 'discount',
      scope: 'order',
      amount_kind: 'percentage',
      amount_minor: 0,
      value_basis_points: 1250,
      application_index: 30,
      manual: true,
      requires_permission: 'pos.pricing.discount.apply',
    });

    await page.cleanup();
  });
});

describe('FloorPage page integration', () => {
  it('covers loading, empty, archived hall rendering, and safe error states', async () => {
    const halls = deferred<unknown[]>();
    installFetchMock([
      { path: `/master-data/floor/halls?restaurant_id=${restaurantId}`, responder: () => halls.promise },
      { path: `/master-data/floor/tables?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const loading = await renderPage(<FloorPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(loading.container)).toContain(ru.status.loading));
    halls.resolve([]);
    await loading.cleanup();

    installFetchMock([
      { path: `/master-data/floor/halls?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/floor/tables?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const empty = await renderPage(<FloorPage restaurantId={restaurantId} />);
    await waitFor(() => {
      expect(text(empty.container)).toContain(ru.floor.halls.empty);
      expect(text(empty.container)).toContain(ru.floor.tables.noHalls);
    });
    await empty.cleanup();

    installFetchMock([
      {
        path: `/master-data/floor/halls?restaurant_id=${restaurantId}`,
        responder: () => [hall(), hall({ id: 'hall-archived', name: 'Archived hall', status: 'archived' })],
      },
      { path: `/master-data/floor/tables?restaurant_id=${restaurantId}`, responder: () => [table()] },
    ]);
    const success = await renderPage(<FloorPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(success.container)).toContain('Archived hall'));
    const tableHallOptions = Array.from(success.container.querySelectorAll('select')[0].querySelectorAll('option')).map((option) => option.textContent);
    expect(tableHallOptions).toContain('Main hall');
    expect(tableHallOptions).not.toContain('Archived hall');
    await success.cleanup();

    installFetchMock([
      { path: `/master-data/floor/halls?restaurant_id=${restaurantId}`, responder: () => { throw safeApiError(); } },
      { path: `/master-data/floor/tables?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const failed = await renderPage(<FloorPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(failed.container)).toContain(ru.floor.blockedTitle));
    expectNoRawMarkers(failed.container);
    await failed.cleanup();
  });

  it('creates a table preserving hall, name, and seats, then refreshes floor lists', async () => {
    let tableReads = 0;
    const api = installFetchMock([
      { path: `/master-data/floor/halls?restaurant_id=${restaurantId}`, responder: () => [hall()] },
      {
        path: `/master-data/floor/tables?restaurant_id=${restaurantId}`,
        responder: () => {
          tableReads += 1;
          return tableReads > 1 ? [table({ id: 'table-2', name: 'T2', seats: 6 })] : [];
        },
      },
      { method: 'POST', path: '/master-data/floor/tables', responder: () => table({ id: 'table-2', name: 'T2', seats: 6 }) },
    ]);

    const page = await renderPage(<FloorPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(page.container)).toContain(ru.floor.tables.empty));

    const tablePanel = Array.from(page.container.querySelectorAll('section')).find((section) => section.querySelector('h3')?.textContent === ru.floor.tables.title);
    expect(tablePanel).not.toBeNull();
    const inputs = tablePanel!.querySelectorAll('input');
    await change(inputs[0], 'T2');
    await change(inputs[1], '6');
    await click(buttonByText(tablePanel!, ru.floor.tables.actions.create));

    await waitFor(() => {
      expect(api.callsFor('/master-data/floor/tables', 'POST')).toHaveLength(1);
      expect(tableReads).toBe(2);
      expect(text(page.container)).toContain('T2');
    });
    expect(api.callsFor('/master-data/floor/tables', 'POST')[0].body).toEqual({
      restaurant_id: restaurantId,
      hall_id: 'hall-main',
      name: 'T2',
      seats: 6,
    });

    await page.cleanup();
  });
});
