import { describe, expect, it } from 'vitest';
import {
  buildCreateCatalogFolderPayload,
  buildCreateCatalogItemPayload,
  buildCreateCatalogTagPayload,
  buildCreateFolderParameterPayload,
} from './catalogForms';

describe('catalogForms payloads', () => {
  it('omits update-only status from catalog item create payload', () => {
    expect(buildCreateCatalogItemPayload({
      kind: 'dish',
      folder_id: '',
      name: 'Tea',
      sku: 'TEA',
      base_unit: 'portion',
      kitchen_type: '',
      accounting_category: '',
      qr_confirmation_enabled: false,
      validity_mode: '',
      validity_expires_at: '',
      status: 'draft',
    })).toEqual({
      kind: 'dish',
      folder_id: '',
      name: 'Tea',
      sku: 'TEA',
      base_unit: 'portion',
      kitchen_type: '',
      accounting_category: '',
      qr_confirmation_enabled: false,
    });
  });

  it('omits update-only status from catalog folder create payload', () => {
    expect(buildCreateCatalogFolderPayload({
      parent_id: '',
      name: 'Kitchen',
      sort_order: 10,
      status: 'draft',
    })).toEqual({
      parent_id: '',
      name: 'Kitchen',
      sort_order: 10,
    });
  });

  it('omits update-only status from folder parameter create payload', () => {
    expect(buildCreateFolderParameterPayload({
      folder_id: 'folder-1',
      parameter_key: 'temperature',
      value_type: 'string',
      value_json: '"hot"',
      status: 'draft',
    })).toEqual({
      folder_id: 'folder-1',
      parameter_key: 'temperature',
      value_type: 'string',
      value_json: '"hot"',
    });
  });

  it('omits update-only status from catalog tag create payload', () => {
    expect(buildCreateCatalogTagPayload({
      name: 'Seasonal',
      code: 'seasonal',
      status: 'draft',
    })).toEqual({
      name: 'Seasonal',
      code: 'seasonal',
    });
  });
});
