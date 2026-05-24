import { z } from 'zod';

const optionalNullableString = z.string().nullable().optional();

export const edgeNodeIdentitySchema = z.object({
  id: z.string(),
  node_device_id: z.string(),
  restaurant_id: z.string(),
  status: z.literal('paired'),
  paired_at: z.string(),
});

export const pairingStatusSchema = z.object({
  paired: z.boolean(),
  identity: edgeNodeIdentitySchema.optional(),
  node_device_id: z.string().optional(),
  restaurant_id: z.string().optional(),
});

export const provisioningStatusSchema = z.object({
  node_device_id: z.string(),
  cloud_url: z.string().optional(),
  license_url: z.string().optional(),
  restaurant_id: z.string().optional(),
  status: z.enum(['not_configured', 'pending_admin_approval', 'assigned_downloading_snapshot', 'paired', 'error']),
  paired: z.boolean(),
  last_error: z.string().optional(),
});

export const authSessionSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  node_device_id: z.string(),
  client_device_id: z.string(),
  employee_id: z.string(),
  status: z.enum(['active', 'revoked']),
  started_at: z.string(),
  last_seen_at: z.string(),
  revoked_at: z.string().optional(),
});

export const actorContextSchema = z.object({
  employee_id: z.string(),
  restaurant_id: z.string(),
  role_id: z.string(),
  name: z.string(),
  permissions: z.array(z.string()),
});

export const pinLoginResultSchema = z.object({
  session: authSessionSchema,
  actor: actorContextSchema,
  permissions: z.array(z.string()),
});

export const hallSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  name: z.string(),
  active: z.boolean(),
});

export const tableSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  hall_id: z.string(),
  name: z.string(),
  seats: z.number(),
  active: z.boolean(),
});

export const shiftSchema = z.object({
  id: z.string(),
  restaurant_id: z.string(),
  device_id: z.string(),
  opened_by_employee_id: z.string(),
  closed_by_employee_id: optionalNullableString,
  status: z.enum(['open', 'closed']),
  opened_at: z.string(),
  closed_at: optionalNullableString,
  opening_cash_amount: z.number().optional().default(0),
  closing_cash_amount: z.number().nullable().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const cashSessionSchema = z.object({
  id: z.string(),
  edge_cash_session_id: z.string(),
  restaurant_id: z.string(),
  device_id: z.string(),
  shift_id: z.string(),
  opened_by_employee_id: z.string(),
  closed_by_employee_id: optionalNullableString,
  status: z.enum(['open', 'closed']),
  opening_cash_amount: z.number(),
  closing_cash_amount: z.number().nullable().optional(),
  opened_at: z.string(),
  closed_at: optionalNullableString,
  created_at: z.string(),
  updated_at: z.string(),
});

export const cashDrawerEventSchema = z.object({
  id: z.string(),
  edge_cash_drawer_event_id: z.string(),
  cash_session_id: z.string(),
  restaurant_id: z.string(),
  device_id: z.string(),
  shift_id: z.string(),
  created_by_employee_id: z.string(),
  event_type: z.enum(['cash_in', 'cash_out', 'no_sale', 'cash_count']),
  amount: z.number(),
  reason: optionalNullableString,
  note: optionalNullableString,
  occurred_at: z.string(),
  created_at: z.string(),
});

export const menuItemSchema = z.object({
  id: z.string(),
  catalog_item_id: z.string(),
  item_type: z.enum(['dish', 'good', 'semi_finished', 'service']).optional().default('dish'),
  name: z.string(),
  price: z.number(),
  currency: z.string(),
  modifier_groups: z.array(z.object({
    id: z.string(),
    restaurant_id: z.string(),
    name: z.string(),
    required: z.boolean(),
    min_count: z.number(),
    max_count: z.number(),
    active: z.boolean(),
    options: z.array(z.object({
      id: z.string(),
      restaurant_id: z.string(),
      modifier_group_id: z.string(),
      name: z.string(),
      price_minor: z.number(),
      active: z.boolean(),
    })).optional().default([]),
  })).optional().default([]),
  active: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const orderLineModifierSchema = z.object({
  id: z.string(),
  order_line_id: z.string(),
  modifier_group_id: z.string(),
  modifier_option_id: z.string(),
  name: z.string(),
  quantity: z.number(),
  unit_price: z.number(),
  total_price: z.number(),
});

export const orderLineSchema = z.object({
  id: z.string(),
  order_id: z.string(),
  menu_item_id: z.string(),
  catalog_item_id: z.string(),
  name: z.string(),
  quantity: z.number(),
  unit_price: z.number(),
  total_price: z.number(),
  currency_code: z.string(),
  tax_profile_id: optionalNullableString,
  course: optionalNullableString,
  comment: optionalNullableString,
  modifiers: z.array(orderLineModifierSchema).optional().default([]),
  status: z.enum(['active', 'cancelled', 'voided']),
  created_at: z.string(),
  updated_at: z.string(),
});

export const paymentSchema = z.object({
  id: z.string(),
  edge_payment_id: z.string(),
  restaurant_id: z.string(),
  device_id: z.string(),
  shift_id: z.string(),
  precheck_id: z.string(),
  method: z.enum(['cash', 'card', 'other']),
  amount: z.number(),
  currency: z.string(),
  status: z.enum(['captured', 'refunded', 'failed']),
  business_date_local: z.string(),
  provider_name: optionalNullableString,
  provider_transaction_id: optionalNullableString,
  provider_reference: optionalNullableString,
  fingerprint_hash: optionalNullableString,
  created_at: z.string(),
  updated_at: z.string(),
});

export const checkSchema = z.object({
  id: z.string(),
  order_id: z.string(),
  status: z.enum(['open', 'paid', 'refunded', 'voided']),
  currency_code: z.string(),
  subtotal: z.number(),
  discount_total: z.number(),
  surcharge_total: z.number(),
  tax_total: z.number(),
  total: z.number(),
  paid_total: z.number(),
  remaining_total: z.number(),
  business_date_local: z.string(),
  closed_at: z.string(),
  snapshot: z.unknown().optional(),
  payments: z.array(paymentSchema).optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const orderSchema = z.object({
  id: z.string(),
  edge_order_id: z.string(),
  restaurant_id: z.string(),
  device_id: z.string(),
  shift_id: z.string(),
  status: z.enum(['open', 'locked', 'closed', 'cancelled']),
  table_id: z.string(),
  table_name: z.string(),
  guest_count: z.number(),
  subtotal: z.number().optional().default(0),
  discount_total: z.number().optional().default(0),
  tax_total: z.number().optional().default(0),
  total: z.number().optional().default(0),
  opened_at: z.string(),
  closed_at: optionalNullableString,
  created_at: z.string(),
  updated_at: z.string(),
  lines: z.array(orderLineSchema).optional().default([]),
  check: checkSchema.optional(),
});

export const closedOrderSchema = z.object({
  id: z.string(),
  table_name: z.string(),
  opened_at: z.string(),
  closed_at: optionalNullableString,
  total: z.number().optional().default(0),
  status: z.enum(['open', 'locked', 'closed', 'cancelled']),
  check: checkSchema.optional(),
});

export const precheckSchema = z.object({
  id: z.string(),
  order_id: z.string(),
  status: z.enum(['issued', 'closed', 'cancelled', 'superseded']),
  version: z.number(),
  supersedes_precheck_id: optionalNullableString,
  currency_code: z.string(),
  subtotal: z.number(),
  discount_total: z.number(),
  surcharge_total: z.number(),
  tax_total: z.number(),
  total: z.number(),
  paid_total: z.number(),
  remaining_total: z.number(),
  snapshot: z.unknown().optional(),
  created_at: z.string(),
  issued_at: z.string(),
  closed_at: optionalNullableString,
  cancelled_by_employee_id: optionalNullableString,
  cancellation_reason: optionalNullableString,
});

export const reprintDocumentSchema = z.object({
  document_type: z.enum(['precheck', 'check']),
  source_id: z.string(),
  copy_marker: z.string(),
  actor_employee_id: z.string().optional(),
  reprinted_at: z.string(),
  snapshot: z.unknown(),
});

export const financialOperationItemSchema = z.object({
  id: z.string(),
  operation_id: z.string(),
  scope: z.enum(['whole_check', 'order_line', 'modifier_line', 'service_charge', 'tip', 'payment']),
  order_line_id: optionalNullableString,
  payment_id: optionalNullableString,
  quantity: z.number().nullable().optional(),
  amount: z.number(),
  currency: z.string(),
  tax_amount: z.number(),
  snapshot: z.unknown().optional(),
  created_at: z.string(),
});

export const financialOperationSchema = z.object({
  id: z.string(),
  edge_operation_id: z.string(),
  restaurant_id: z.string(),
  device_id: z.string(),
  shift_id: z.string(),
  original_shift_id: z.string(),
  check_id: z.string(),
  precheck_id: z.string(),
  operation_type: z.enum(['cancellation', 'refund']),
  operation_kind: z.enum(['full', 'partial']),
  status: z.literal('recorded'),
  amount: z.number(),
  currency: z.string(),
  business_date_local: z.string(),
  inventory_disposition: z.enum(['no_stock_effect', 'return_to_stock', 'write_off_waste', 'manual_review']),
  reason: z.string(),
  created_by_employee_id: z.string(),
  approved_by_employee_id: optionalNullableString,
  snapshot: z.unknown().optional(),
  items: z.array(financialOperationItemSchema).optional().default([]),
  created_at: z.string(),
});

export const syncStatusSchema = z.object({
  total: z.number(),
  pending: z.number(),
  processing: z.number(),
  sent: z.number(),
  failed: z.number(),
  suspended: z.number(),
  oldest_pending_sequence_no: z.number().optional(),
});

export const outboxMessageSchema = z.object({
  id: z.string(),
  command_id: z.string(),
  sequence_no: z.number(),
  origin: z.string(),
  aggregate_type: z.string(),
  aggregate_id: z.string(),
  command_type: z.string(),
  sync_direction: z.string(),
  status: z.enum(['pending', 'processing', 'sent', 'failed', 'suspended']),
  attempts: z.number(),
  last_error: optionalNullableString,
  created_at: z.string(),
  updated_at: z.string(),
});

export const retryFailedOutboxResultSchema = z.object({
  retried: z.number(),
});

export type BackendActorContext = z.infer<typeof actorContextSchema>;
export type BackendCashDrawerEvent = z.infer<typeof cashDrawerEventSchema>;
export type BackendCashSession = z.infer<typeof cashSessionSchema>;
export type BackendCheck = z.infer<typeof checkSchema>;
export type BackendClosedOrder = z.infer<typeof closedOrderSchema>;
export type BackendFinancialOperation = z.infer<typeof financialOperationSchema>;
export type BackendHall = z.infer<typeof hallSchema>;
export type BackendMenuItem = z.infer<typeof menuItemSchema>;
export type BackendOrder = z.infer<typeof orderSchema>;
export type BackendOrderLine = z.infer<typeof orderLineSchema>;
export type BackendPairingStatus = z.infer<typeof pairingStatusSchema>;
export type BackendPayment = z.infer<typeof paymentSchema>;
export type BackendPinLoginResult = z.infer<typeof pinLoginResultSchema>;
export type BackendPrecheck = z.infer<typeof precheckSchema>;
export type BackendProvisioningStatus = z.infer<typeof provisioningStatusSchema>;
export type BackendShift = z.infer<typeof shiftSchema>;
export type BackendSyncStatus = z.infer<typeof syncStatusSchema>;
export type BackendTable = z.infer<typeof tableSchema>;
