import type { CatalogFolder, CatalogItem, CatalogTag, FolderParameter } from '../../shared/api/schemas';

export type CatalogKind = 'dish' | 'good' | 'semi_finished' | 'service';
export type LifecycleStatus = 'draft' | 'published' | 'archived';

export type CatalogItemFormValues = {
  kind: CatalogKind;
  folder_id: string;
  name: string;
  sku: string;
  base_unit: string;
  kitchen_type: string;
  accounting_category: string;
  status: LifecycleStatus;
};

export type CatalogFolderFormValues = {
  parent_id: string;
  name: string;
  sort_order: number;
  status: LifecycleStatus;
};

export type FolderParameterFormValues = {
  folder_id: string;
  parameter_key: string;
  value_type: string;
  value_json: string;
  status: LifecycleStatus;
};

export type CatalogTagFormValues = {
  name: string;
  code: string;
  status: LifecycleStatus;
};

export type ItemTagCommandFormValues = {
  catalog_item_id: string;
  tag_id: string;
};

export const defaultCatalogItemValues: CatalogItemFormValues = {
  kind: 'dish',
  folder_id: '',
  name: '',
  sku: '',
  base_unit: 'pcs',
  kitchen_type: '',
  accounting_category: '',
  status: 'draft',
};

export const defaultCatalogFolderValues: CatalogFolderFormValues = {
  parent_id: '',
  name: '',
  sort_order: 0,
  status: 'draft',
};

export const defaultFolderParameterValues: FolderParameterFormValues = {
  folder_id: '',
  parameter_key: '',
  value_type: 'string',
  value_json: '{}',
  status: 'draft',
};

export const defaultCatalogTagValues: CatalogTagFormValues = {
  name: '',
  code: '',
  status: 'draft',
};

export const defaultItemTagCommandValues: ItemTagCommandFormValues = {
  catalog_item_id: '',
  tag_id: '',
};

export function toCatalogItemValues(item: CatalogItem): CatalogItemFormValues {
  return {
    kind: item.kind,
    folder_id: item.folder_id ?? '',
    name: item.name,
    sku: item.sku,
    base_unit: item.base_unit,
    kitchen_type: item.kitchen_type ?? '',
    accounting_category: item.accounting_category ?? '',
    status: item.status,
  };
}

export function toCatalogFolderValues(folder: CatalogFolder): CatalogFolderFormValues {
  return {
    parent_id: folder.parent_id ?? '',
    name: folder.name,
    sort_order: folder.sort_order,
    status: folder.status,
  };
}

export function toFolderParameterValues(parameter: FolderParameter): FolderParameterFormValues {
  return {
    folder_id: parameter.folder_id,
    parameter_key: parameter.parameter_key,
    value_type: parameter.value_type,
    value_json: parameter.value_json,
    status: parameter.status,
  };
}

export function toCatalogTagValues(tag: CatalogTag): CatalogTagFormValues {
  return {
    name: tag.name,
    code: tag.code,
    status: tag.status,
  };
}
