import { describe, expect, it } from 'vitest';
import { buildCreateMenuItemPayload } from './menuForms';

describe('menuForms payloads', () => {
  it('omits update-only status from menu item create payload', () => {
    expect(buildCreateMenuItemPayload({
      catalog_item_id: 'catalog-1',
      category_id: '',
      name: 'Tea',
      price: 1000,
      currency: 'RUB',
      status: 'published',
      availability_json: '{}',
      station_routing_key: '',
    })).toEqual({
      catalog_item_id: 'catalog-1',
      category_id: '',
      name: 'Tea',
      price: 1000,
      currency: 'RUB',
      availability_json: '{}',
      station_routing_key: '',
    });
  });
});
