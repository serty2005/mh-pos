import type { ModifierBinding, ModifierGroup, ModifierOption } from '../../shared/api/schemas';
import type { LifecycleStatus } from '../catalog/catalogForms';

export type ModifierTargetType = 'menu_item' | 'catalog_item' | 'folder' | 'tag';

export type ModifierGroupFormValues = {
  name: string;
  required: boolean;
  min_count: number;
  max_count: number;
  status: LifecycleStatus;
};

export type ModifierOptionFormValues = {
  modifier_group_id: string;
  name: string;
  price_minor: number;
  status: LifecycleStatus;
};

export type ModifierBindingFormValues = {
  modifier_group_id: string;
  target_type: ModifierTargetType;
  target_id: string;
  sort_order: number;
  status: LifecycleStatus;
};

export const defaultModifierGroupValues: ModifierGroupFormValues = {
  name: '',
  required: false,
  min_count: 0,
  max_count: 1,
  status: 'published',
};

export const defaultModifierOptionValues: ModifierOptionFormValues = {
  modifier_group_id: '',
  name: '',
  price_minor: 0,
  status: 'published',
};

export const defaultModifierBindingValues: ModifierBindingFormValues = {
  modifier_group_id: '',
  target_type: 'menu_item',
  target_id: '',
  sort_order: 0,
  status: 'published',
};

export function toModifierGroupValues(group: ModifierGroup): ModifierGroupFormValues {
  return {
    name: group.name,
    required: group.required,
    min_count: group.min_count,
    max_count: group.max_count,
    status: group.status,
  };
}

export function toModifierOptionValues(option: ModifierOption): ModifierOptionFormValues {
  return {
    modifier_group_id: option.modifier_group_id,
    name: option.name,
    price_minor: option.price_minor,
    status: option.status,
  };
}

export function toModifierBindingValues(binding: ModifierBinding): ModifierBindingFormValues {
  return {
    modifier_group_id: binding.modifier_group_id,
    target_type: binding.target_type,
    target_id: binding.target_id,
    sort_order: binding.sort_order,
    status: binding.status,
  };
}
