import { z } from 'zod';

export const lifecycleStatusSchema = z.enum(['draft', 'published', 'archived']);
export const restaurantStatusSchema = z.enum(['active', 'archived']);
export const employeeStatusSchema = z.enum(['active', 'suspended', 'archived']);
export const edgeDeviceStatusSchema = z.enum(['pending', 'assigned', 'rejected', 'expired', 'unassigned', 'revoked']);

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
  role_id: z.string(),
  restaurant_ids: z.array(z.string()),
  all_restaurants: z.boolean(),
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
  linked_catalog_item_id: z.string().optional(),
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

export const recipeItemSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  recipe_owner_catalog_item_id: z.string(),
  component_catalog_item_id: z.string(),
  quantity: z.number(),
  unit: z.string(),
  loss_percent: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const stopListEntrySchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  catalog_item_id: z.string(),
  available_quantity: z.number().nullable().optional(),
  source: z.string(),
  reason: z.string().optional().default(''),
  active: z.boolean(),
  cloud_version: z.number().optional(),
  updated_at: z.string(),
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

export const pricingPolicyPackageSchema = z.object({
  node_device_id: z.string(),
  restaurant_id: z.string(),
  sync_mode: z.string(),
  full_snapshot_reason: z.string().optional().default(''),
  cloud_version: z.number(),
  payload_json: z.object({
    tax_profiles: z.array(z.record(z.string(), z.unknown())).default([]),
    tax_rules: z.array(z.record(z.string(), z.unknown())).default([]),
    service_charge_rules: z.array(z.record(z.string(), z.unknown())).default([]),
    pricing_policies: z.array(z.record(z.string(), z.unknown())).default([]),
  }),
});

export const masterDataPackageSchema = z.object({
  stream_name: z.string(),
  node_device_id: z.string().optional().default(''),
  restaurant_id: z.string().optional().default(''),
  sync_mode: z.string(),
  full_snapshot_reason: z.string().optional().default(''),
  cloud_version: z.number(),
  checkpoint_token: z.string().optional().default(''),
  cloud_updated_at: z.string().optional(),
  payload_json: z.unknown(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const restaurantEdgeNodeSchema = z.object({
  id: z.string(),
  restaurant_id: z.string().optional().default(''),
  node_device_id: z.string(),
  display_name: z.string(),
  status: edgeDeviceStatusSchema,
  last_seen_at: z.string().optional(),
  assigned_at: z.string().optional(),
  revoked_at: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const unassignedEdgeNodeSchema = z.object({
  id: z.string(),
  node_device_id: z.string(),
  claimed_cloud_url: z.string(),
  display_name: z.string(),
  app_version: z.string(),
  status: edgeDeviceStatusSchema,
  first_seen_at: z.string(),
  last_seen_at: z.string(),
  assigned_restaurant_id: z.string().optional().default(''),
  assigned_at: z.string().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const assignDeviceResultSchema = z.object({
  node_device_id: z.string(),
  restaurant_id: z.string(),
  status: edgeDeviceStatusSchema,
  snapshot_url: z.string(),
});

export const assignmentStatusSchema = z.object({
  node_device_id: z.string(),
  status: edgeDeviceStatusSchema,
  restaurant_id: z.string().optional().default(''),
  cloud_url: z.string().optional().default(''),
  snapshot_url: z.string().optional().default(''),
});

export const pairingCodeResultSchema = z.object({
  pairing_code: z.string(),
  pairing_id: z.string(),
  restaurant_id: z.string(),
  expires_at: z.string(),
});

export const edgeEventSchema = z.object({
  cloud_receipt_id: z.string(),
  idempotency_key: z.string(),
  restaurant_id: z.string(),
  device_id: z.string(),
  command_id: z.string(),
  event_id: z.string(),
  edge_event_id: z.string(),
  event_type: z.string(),
  aggregate_type: z.string(),
  aggregate_id: z.string(),
  envelope_version: z.string(),
  occurred_at: z.string(),
  cloud_received_at: z.string(),
  raw_payload_sha256_hex: z.string(),
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
export type RecipeItem = z.infer<typeof recipeItemSchema>;
export type StopListEntry = z.infer<typeof stopListEntrySchema>;
export type Hall = z.infer<typeof hallSchema>;
export type RestaurantTable = z.infer<typeof tableSchema>;
export type MenuItem = z.infer<typeof menuItemSchema>;
export type Category = z.infer<typeof categorySchema>;
export type PricingPolicyPackage = z.infer<typeof pricingPolicyPackageSchema>;
export type MasterDataPackage = z.infer<typeof masterDataPackageSchema>;
export type RestaurantEdgeNode = z.infer<typeof restaurantEdgeNodeSchema>;
export type EdgeEvent = z.infer<typeof edgeEventSchema>;
export type PublicationSummary = z.infer<typeof publicationSummarySchema>;
export type UnassignedEdgeNode = z.infer<typeof unassignedEdgeNodeSchema>;
export type AssignDeviceResult = z.infer<typeof assignDeviceResultSchema>;
export type AssignmentStatus = z.infer<typeof assignmentStatusSchema>;
export type PairingCodeResult = z.infer<typeof pairingCodeResultSchema>;
