import { describe, expect, it } from 'vitest';
import { buildCreateMenuItemPayload } from './menuForms';

describe('menuForms payloads', () => {
  it('omits update-only status from menu item create payload', () => {
    expect(buildCreateMenuItemPayload({
      catalog_item_id: 'catalog-1',
      category_id: '',
      tag_id: 'tag-vip',
      tax_profile_id: 'tax-active',
      name: 'Tea',
      price: 1000,
      currency: 'RUB',
      status: 'published',
      runtime_status: 'available',
      availability_json: '{}',
      station_routing_key: '',
    })).toEqual({
      catalog_item_id: 'catalog-1',
      category_id: '',
      tag_id: 'tag-vip',
      tax_profile_id: 'tax-active',
      name: 'Tea',
      price: 1000,
      currency: 'RUB',
      runtime_status: 'available',
      availability_json: '{}',
      station_routing_key: '',
    });
  });
});
