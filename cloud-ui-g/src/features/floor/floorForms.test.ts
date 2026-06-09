import { describe, expect, it } from 'vitest';
import {
  buildCreateHallPayload,
  buildCreateTablePayload,
  buildUpdateHallPayload,
  buildUpdateTablePayload,
  toHallValues,
  toTableValues,
} from './floorForms';

describe('floorForms payloads', () => {
  it('normalizes hall create and update payloads', () => {
    expect(buildCreateHallPayload({ name: ' Main hall ' })).toEqual({
      name: 'Main hall',
    });

    expect(buildUpdateHallPayload({ name: ' Terrace ', status: 'archived' })).toEqual({
      name: 'Terrace',
      status: 'archived',
    });
  });

  it('normalizes table payloads and preserves hall assignment', () => {
    expect(buildCreateTablePayload({
      hall_id: 'hall-1',
      name: ' T-7 ',
      seats: 4,
      status: 'published',
    })).toEqual({
      hall_id: 'hall-1',
      name: 'T-7',
      seats: 4,
    });

    expect(buildUpdateTablePayload({
      hall_id: 'hall-2',
      name: ' Bar 1 ',
      seats: 2,
      status: 'draft',
    })).toEqual({
      hall_id: 'hall-2',
      name: 'Bar 1',
      seats: 2,
      status: 'draft',
    });
  });

  it('maps backend halls and tables into editable values', () => {
    expect(toHallValues({
      id: 'hall-1',
      restaurant_id: 'restaurant-1',
      name: 'Main',
      status: 'published',
      cloud_version: 1,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })).toEqual({ name: 'Main', status: 'published' });

    expect(toTableValues({
      id: 'table-1',
      restaurant_id: 'restaurant-1',
      hall_id: 'hall-1',
      name: 'A1',
      seats: 6,
      status: 'draft',
      cloud_version: 1,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })).toEqual({
      hall_id: 'hall-1',
      name: 'A1',
      seats: 6,
      status: 'draft',
    });
  });
});
