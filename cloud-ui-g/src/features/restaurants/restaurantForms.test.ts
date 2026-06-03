import { describe, expect, it } from 'vitest';
import { buildCreateRestaurantPayload, type RestaurantFormValues } from './RestaurantForm';

const values: RestaurantFormValues = {
  name: 'Manual Cafe',
  timezone: 'Europe/Moscow',
  currency: 'RUB',
  business_day_mode: 'standard',
  business_day_boundary_local_time: '04:00',
  status: 'active',
};

describe('RestaurantForm payloads', () => {
  it('omits update-only status from create payload', () => {
    expect(buildCreateRestaurantPayload(values)).toEqual({
      name: 'Manual Cafe',
      timezone: 'Europe/Moscow',
      currency: 'RUB',
      business_day_mode: 'standard',
      business_day_boundary_local_time: '04:00',
    });
  });
});
