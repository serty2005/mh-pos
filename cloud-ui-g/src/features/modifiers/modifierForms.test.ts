import { describe, expect, it } from 'vitest';
import {
  buildCreateModifierBindingPayload,
  buildCreateModifierGroupPayload,
  buildCreateModifierOptionPayload,
} from './modifierForms';

describe('modifierForms payloads', () => {
  it('omits update-only status from modifier group create payload', () => {
    expect(buildCreateModifierGroupPayload({
      name: 'Milk',
      required: false,
      min_count: 0,
      max_count: 1,
      status: 'published',
    })).toEqual({
      name: 'Milk',
      required: false,
      min_count: 0,
      max_count: 1,
    });
  });

  it('omits update-only status from modifier option create payload', () => {
    expect(buildCreateModifierOptionPayload({
      modifier_group_id: 'group-1',
      name: 'Oat milk',
      price_minor: 100,
      status: 'published',
    })).toEqual({
      modifier_group_id: 'group-1',
      name: 'Oat milk',
      price_minor: 100,
    });
  });

  it('omits update-only status from modifier binding create payload', () => {
    expect(buildCreateModifierBindingPayload({
      modifier_group_id: 'group-1',
      target_type: 'menu_item',
      target_id: 'menu-1',
      sort_order: 10,
      status: 'published',
    })).toEqual({
      modifier_group_id: 'group-1',
      target_type: 'menu_item',
      target_id: 'menu-1',
      sort_order: 10,
    });
  });
});
