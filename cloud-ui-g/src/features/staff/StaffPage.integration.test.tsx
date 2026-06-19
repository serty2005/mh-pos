// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import StaffPage from './StaffPage';
import { permissionCatalog } from './permissionCatalog';
import { roleProfileById } from './roleProfiles';
import { ru } from '../../shared/i18n/ru';
import {
  buttonByAriaLabel,
  buttonByText,
  change,
  click,
  deferred,
  employee,
  expectNoRawMarkers,
  installFetchMock,
  renderPage,
  restaurantId,
  role,
  safeApiError,
  text,
  waitFor,
} from '../../test/pageHarness';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
});

describe('StaffPage page integration', () => {
  it('covers loading, empty employees/roles, and blocked route errors safely', async () => {
    const roles = deferred<unknown[]>();
    installFetchMock([
      { path: `/master-data/roles?restaurant_id=${restaurantId}`, responder: () => roles.promise },
      { path: `/master-data/employees?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);

    const loading = await renderPage(<StaffPage restaurantId={restaurantId} />);
    await waitFor(() => {
      expect(loading.container.querySelector('[aria-busy="true"]')).not.toBeNull();
      expect(text(loading.container)).toContain(ru.staff.pageTitle);
    });
    roles.resolve([]);
    await loading.cleanup();

    installFetchMock([
      { path: `/master-data/roles?restaurant_id=${restaurantId}`, responder: () => [] },
      { path: `/master-data/employees?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const empty = await renderPage(<StaffPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(empty.container)).toContain(ru.staff.employees.emptyTitle));
    await click(buttonByText(empty.container, ru.staff.tabs.roles));
    await waitFor(() => expect(text(empty.container)).toContain(ru.staff.roles.empty));
    await empty.cleanup();

    installFetchMock([
      { path: `/master-data/roles?restaurant_id=${restaurantId}`, responder: () => { throw safeApiError(); } },
      { path: `/master-data/employees?restaurant_id=${restaurantId}`, responder: () => [] },
    ]);
    const blocked = await renderPage(<StaffPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(blocked.container)).toContain(ru.staff.blockedTitle));
    expectNoRawMarkers(blocked.container);
    await blocked.cleanup();
  });

  it('creates a role with canonical permissions JSON and reloads lists once', async () => {
    let roleReads = 0;
    let employeeReads = 0;
    const api = installFetchMock([
      {
        path: `/master-data/roles?restaurant_id=${restaurantId}`,
        responder: () => {
          roleReads += 1;
          return roleReads > 1 ? [role({ id: 'role-created', name: ru.staff.roleProfiles.manager })] : [];
        },
      },
      {
        path: `/master-data/employees?restaurant_id=${restaurantId}`,
        responder: () => {
          employeeReads += 1;
          return [];
        },
      },
      {
        method: 'POST',
        path: '/master-data/roles',
        responder: (request) => role({
          id: 'role-created',
          name: String((request.body as { name: string }).name),
        }),
      },
    ]);

    const page = await renderPage(<StaffPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(page.container)).toContain(ru.staff.employees.emptyTitle));

    await click(buttonByText(page.container, ru.staff.tabs.roles));
    const profileSelect = page.container.querySelector('select');
    expect(profileSelect).not.toBeNull();
    await change(profileSelect!, 'manager');
    await click(buttonByText(page.container, ru.staff.roles.actions.create));

    await waitFor(() => {
      expect(api.callsFor('/master-data/roles', 'POST')).toHaveLength(1);
      expect(roleReads).toBe(2);
      expect(employeeReads).toBe(2);
      expect(text(page.container)).toContain(ru.staff.roleProfiles.manager);
    });

    const body = api.callsFor('/master-data/roles', 'POST')[0].body as {
      restaurant_id: string;
      name: string;
      permissions_json: string;
    };
    const permissions = JSON.parse(body.permissions_json) as Record<string, boolean>;
    const managerProfile = roleProfileById.get('manager');
    expect(body.restaurant_id).toBe(restaurantId);
    expect(body.name).toBe(ru.staff.roleProfiles.manager);
    expect(Object.keys(permissions).sort()).toEqual([...(managerProfile?.permissionIds ?? [])].sort());

    await page.cleanup();
  });

  it('updates role permissions without dropping selected ids', async () => {
    const existingRole = role({
      permissions_json: '{"pos.catalog.view":true,"pos.sync.view":true}',
    });
    const targetPermission = permissionCatalog.find((permission) => permission.id === 'pos.floor.view');
    expect(targetPermission).toBeDefined();
    const api = installFetchMock([
      { path: `/master-data/roles?restaurant_id=${restaurantId}`, responder: () => [existingRole] },
      { path: `/master-data/employees?restaurant_id=${restaurantId}`, responder: () => [] },
      {
        method: 'PATCH',
        path: `/master-data/roles/${existingRole.id}`,
        responder: () => existingRole,
      },
    ]);

    const page = await renderPage(<StaffPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(page.container)).toContain(ru.staff.employees.emptyTitle));
    await click(buttonByText(page.container, ru.staff.tabs.roles));
    await waitFor(() => expect(text(page.container)).toContain(existingRole.name));
    await click(buttonByAriaLabel(page.container, `${existingRole.name}: ${ru.staff.permissions.items.floorView}`));

    await waitFor(() => expect(api.callsFor(`/master-data/roles/${existingRole.id}`, 'PATCH')).toHaveLength(1));
    const body = api.callsFor(`/master-data/roles/${existingRole.id}`, 'PATCH')[0].body as { permissions_json: string };
    const permissions = JSON.parse(body.permissions_json) as Record<string, boolean>;
    expect(permissions['pos.catalog.view']).toBe(true);
    expect(permissions['pos.sync.view']).toBe(true);
    expect(permissions['pos.floor.view']).toBe(true);

    await page.cleanup();
  });

  it('creates employees with scoped body, no visible PIN leakage, and safe failed mutation errors', async () => {
    const createPending = deferred<unknown>();
    let roleReads = 0;
    let employeeReads = 0;
    const api = installFetchMock([
      {
        path: `/master-data/roles?restaurant_id=${restaurantId}`,
        responder: () => {
          roleReads += 1;
          return [role()];
        },
      },
      {
        path: `/master-data/employees?restaurant_id=${restaurantId}`,
        responder: () => {
          employeeReads += 1;
          return employeeReads > 1 ? [employee()] : [];
        },
      },
      {
        method: 'POST',
        path: '/master-data/employees',
        responder: () => createPending.promise,
      },
    ]);

    const page = await renderPage(<StaffPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(page.container)).toContain(ru.staff.employees.emptyTitle));

    await click(buttonByText(page.container, ru.staff.employees.actions.new));
    const modal = page.container.querySelector('.fixed');
    expect(modal).not.toBeNull();
    const inputs = modal!.querySelectorAll('input');
    const selects = modal!.querySelectorAll('select');
    await change(inputs[0], 'Bob Cashier');
    await change(selects[0], 'role-manager');
    await change(inputs[1], '1234');
    await click(buttonByText(modal!, ru.staff.employees.actions.create));
    await click(buttonByText(modal!, ru.staff.employees.actions.create));

    await waitFor(() => expect(api.callsFor('/master-data/employees', 'POST')).toHaveLength(1));
    createPending.resolve(employee({ id: 'employee-created', name: 'Bob Cashier' }));

    await waitFor(() => {
      expect(roleReads).toBe(2);
      expect(employeeReads).toBe(2);
      expect(text(page.container)).not.toContain('1234');
    });

    const body = api.callsFor('/master-data/employees', 'POST')[0].body as {
      restaurant_id: string;
      name: string;
      role_id: string;
      pin: string;
    };
    expect(body).toEqual({
      restaurant_id: restaurantId,
      name: 'Bob Cashier',
      role_id: 'role-manager',
      pin: '1234',
    });

    await page.cleanup();

    installFetchMock([
      { path: `/master-data/roles?restaurant_id=${restaurantId}`, responder: () => [role()] },
      { path: `/master-data/employees?restaurant_id=${restaurantId}`, responder: () => [] },
      { method: 'POST', path: '/master-data/employees', responder: () => { throw safeApiError(); } },
    ]);

    const failed = await renderPage(<StaffPage restaurantId={restaurantId} />);
    await waitFor(() => expect(text(failed.container)).toContain(ru.staff.employees.emptyTitle));
    await click(buttonByText(failed.container, ru.staff.employees.actions.new));
    const failedModal = failed.container.querySelector('.fixed');
    expect(failedModal).not.toBeNull();
    const failedInputs = failedModal!.querySelectorAll('input');
    const failedSelects = failedModal!.querySelectorAll('select');
    await change(failedInputs[0], 'Pin Leak Check');
    await change(failedSelects[0], 'role-manager');
    await change(failedInputs[1], '9999');
    await click(buttonByText(failedModal!, ru.staff.employees.actions.create));

    await waitFor(() => {
      expect(text(failed.container)).toContain(ru.errors.server);
      expect(text(failed.container)).not.toContain('9999');
    });
    expectNoRawMarkers(failed.container);
    await failed.cleanup();
  });
});
