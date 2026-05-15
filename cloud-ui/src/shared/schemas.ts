import { z } from 'zod';

export const lifecycleStatusSchema = z.enum(['draft', 'published', 'archived']);
export const restaurantStatusSchema = z.enum(['active', 'archived']);
export const employeeStatusSchema = z.enum(['active', 'suspended', 'archived']);

export const restaurantSchema = z.object({
  id: z.string(),
  name: z.string(),
  timezone: z.string(),
  currency: z.string(),
  business_day_mode: z.string(),
  business_day_boundary_local_time: z.string(),
  status: restaurantStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const roleSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  permissions_json: z.string(),
  active: z.boolean(),
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const employeeSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  role_id: z.string(),
  name: z.string(),
  status: employeeStatusSchema,
  pin_configured: z.boolean(),
  pin_credential_version: z.number(),
  permission_snapshot_json: z.string(),
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  suspended_at: z.string().optional(),
  archived_at: z.string().optional(),
});

export const catalogItemSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  kind: z.enum(['dish', 'good', 'semi_finished', 'service']),
  folder_id: z.string().optional().default(''),
  name: z.string(),
  sku: z.string(),
  base_unit: z.string(),
  kitchen_type: z.string().optional().default(''),
  accounting_category: z.string().optional().default(''),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const catalogFolderSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  parent_id: z.string().optional().default(''),
  name: z.string(),
  sort_order: z.number(),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const folderParameterSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  folder_id: z.string(),
  parameter_key: z.string(),
  value_type: z.string(),
  value_json: z.string(),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const catalogTagSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  code: z.string(),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const catalogItemTagSchema = z.object({
  restaurant_id: z.string(),
  catalog_item_id: z.string(),
  tag_id: z.string(),
  cloud_version: z.number(),
  created_at: z.string(),
});

export const modifierGroupSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  status: lifecycleStatusSchema,
  required: z.boolean(),
  min_count: z.number(),
  max_count: z.number(),
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const modifierOptionSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  modifier_group_id: z.string(),
  name: z.string(),
  price_minor: z.number(),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const modifierBindingSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  modifier_group_id: z.string(),
  target_type: z.enum(['menu_item', 'catalog_item', 'folder', 'tag']),
  target_id: z.string(),
  sort_order: z.number(),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const pricingPolicySchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  kind: z.enum(['discount', 'surcharge']),
  scope: z.string(),
  amount_kind: z.enum(['fixed', 'percentage']),
  amount_minor: z.number().optional().default(0),
  value_basis_points: z.number().optional().default(0),
  application_index: z.number(),
  manual: z.boolean(),
  requires_permission: z.string().optional().default(''),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const hallSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const tableSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  hall_id: z.string(),
  name: z.string(),
  seats: z.number(),
  status: lifecycleStatusSchema,
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const menuItemSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  catalog_item_id: z.string(),
  category_id: z.string().optional().default(''),
  name: z.string(),
  price: z.number(),
  currency: z.string(),
  status: lifecycleStatusSchema,
  availability_json: z.string(),
  station_routing_key: z.string().optional().default(''),
  cloud_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  archived_at: z.string().optional(),
});

export const categorySchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  status: lifecycleStatusSchema,
  sort_order: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const publicationSummarySchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  version: z.number(),
  status: z.string(),
  cloud_version: z.number(),
  published_at: z.string(),
  published_by: z.string(),
  package_sha256: z.string(),
  counts: z.record(z.string(), z.number()),
});

export type Restaurant = z.infer<typeof restaurantSchema>;
export type Role = z.infer<typeof roleSchema>;
export type Employee = z.infer<typeof employeeSchema>;
export type CatalogItem = z.infer<typeof catalogItemSchema>;
export type CatalogFolder = z.infer<typeof catalogFolderSchema>;
export type FolderParameter = z.infer<typeof folderParameterSchema>;
export type CatalogTag = z.infer<typeof catalogTagSchema>;
export type CatalogItemTag = z.infer<typeof catalogItemTagSchema>;
export type ModifierGroup = z.infer<typeof modifierGroupSchema>;
export type ModifierOption = z.infer<typeof modifierOptionSchema>;
export type ModifierBinding = z.infer<typeof modifierBindingSchema>;
export type PricingPolicy = z.infer<typeof pricingPolicySchema>;
export type Hall = z.infer<typeof hallSchema>;
export type RestaurantTable = z.infer<typeof tableSchema>;
export type MenuItem = z.infer<typeof menuItemSchema>;
export type Category = z.infer<typeof categorySchema>;
export type PublicationSummary = z.infer<typeof publicationSummarySchema>;
