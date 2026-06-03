import type { MenuItem } from '../../shared/api/schemas';
import type { LifecycleStatus } from '../catalog/catalogForms';

export type MenuItemFormValues = {
  catalog_item_id: string;
  category_id: string;
  name: string;
  price: number;
  currency: string;
  status: LifecycleStatus;
  availability_json: string;
  station_routing_key: string;
};

export type MenuCategoryFormValues = {
  restaurant_id: string;
  name: string;
  sort_order: number;
};

export const defaultMenuItemValues: MenuItemFormValues = {
  catalog_item_id: '',
  category_id: '',
  name: '',
  price: 0,
  currency: 'RUB',
  status: 'published',
  availability_json: '{}',
  station_routing_key: '',
};

export const defaultMenuCategoryValues: MenuCategoryFormValues = {
  restaurant_id: '',
  name: '',
  sort_order: 0,
};

export function buildCreateMenuItemPayload(values: MenuItemFormValues) {
  const { status: _status, ...payload } = normalizeMenuItemValues(values);
  return payload;
}

export function toMenuItemValues(item: MenuItem): MenuItemFormValues {
  return {
    catalog_item_id: item.catalog_item_id,
    category_id: item.category_id,
    name: item.name,
    price: item.price,
    currency: item.currency,
    status: item.status,
    availability_json: item.availability_json,
    station_routing_key: item.station_routing_key,
  };
}

export function normalizeMenuItemValues(values: MenuItemFormValues): MenuItemFormValues {
  return {
    ...values,
    name: values.name.trim(),
    currency: values.currency.trim().toUpperCase(),
    availability_json: values.availability_json.trim() || '{}',
    station_routing_key: values.station_routing_key.trim(),
  };
}
