import { describe, expect, it } from 'vitest';
import {
  buildCreateEmployeePayload,
  buildCreateRolePayload,
  buildUpdateEmployeePayload,
  buildUpdateRolePayload,
  parsePermissionIds,
  permissionsJsonFromIds,
} from './staffForms';

describe('staffForms payloads', () => {
  it('serializes selected permission ids into canonical permissions_json', () => {
    expect(permissionsJsonFromIds([
      'pos.order.create',
      'pos.menu.view',
      'pos.order.create',
      '',
    ])).toBe('{"pos.menu.view":true,"pos.order.create":true}');
  });

  it('builds role create and update payloads from selected permissions', () => {
    expect(buildCreateRolePayload({
      name: ' Manager ',
      active: true,
      permission_ids: ['pos.sync.view', 'pos.menu.view'],
    })).toEqual({
      name: 'Manager',
      permissions_json: '{"pos.menu.view":true,"pos.sync.view":true}',
    });

    expect(buildUpdateRolePayload({
      name: 'Manager',
      active: false,
      permission_ids: ['pos.sync.view'],
    })).toEqual({
      name: 'Manager',
      active: false,
      permissions_json: '{"pos.sync.view":true}',
    });
  });

  it('never includes PIN in employee update payload', () => {
    expect(buildCreateEmployeePayload({
      name: ' Anna ',
      role_id: 'role-1',
      restaurant_ids: ['restaurant-1'],
      pin: '1234',
    })).toEqual({
      name: 'Anna',
      role_id: 'role-1',
      restaurant_ids: ['restaurant-1'],
      pin: '1234',
    });

    expect(buildUpdateEmployeePayload({
      name: 'Anna',
      role_id: 'role-2',
      restaurant_ids: ['restaurant-2'],
      status: 'suspended',
    })).toEqual({
      name: 'Anna',
      role_id: 'role-2',
      restaurant_ids: ['restaurant-2'],
      status: 'suspended',
    });
  });

  it('parses both canonical object and array permissions_json shapes', () => {
    expect(parsePermissionIds('{"pos.menu.view":true,"pos.order.create":false,"pos.sync.view":true}')).toEqual([
      'pos.menu.view',
      'pos.sync.view',
    ]);
    expect(parsePermissionIds('{"permissions":["pos.floor.view","pos.menu.view"]}')).toEqual([
      'pos.floor.view',
      'pos.menu.view',
    ]);
    expect(parsePermissionIds('not-json')).toEqual([]);
  });
});
