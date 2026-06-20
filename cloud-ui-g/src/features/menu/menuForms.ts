import type { MenuItem } from '../../shared/api/schemas';
import type { LifecycleStatus } from '../catalog/catalogForms';

export type MenuItemFormValues = {
  catalog_item_id: string;
  category_id: string;
  tag_id: string;
  tax_profile_id: string;
  name: string;
  price: number;
  currency: string;
  status: LifecycleStatus;
  runtime_status: string;
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
  tag_id: '',
  tax_profile_id: '',
  name: '',
  price: 0,
  currency: 'RUB',
  status: 'published',
  runtime_status: 'available',
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
    tag_id: item.tag_id,
    tax_profile_id: item.tax_profile_id,
    name: item.name,
    price: item.price,
    currency: item.currency,
    status: item.status,
    runtime_status: item.runtime_status,
    availability_json: item.availability_json,
    station_routing_key: item.station_routing_key,
  };
}

export function normalizeMenuItemValues(values: MenuItemFormValues): MenuItemFormValues {
  return {
    ...values,
    name: values.name.trim(),
    category_id: values.category_id.trim(),
    tag_id: values.tag_id.trim(),
    tax_profile_id: values.tax_profile_id.trim(),
    currency: values.currency.trim().toUpperCase(),
    runtime_status: values.runtime_status.trim() || 'available',
    availability_json: values.availability_json.trim() || '{}',
    station_routing_key: values.station_routing_key.trim(),
  };
}
