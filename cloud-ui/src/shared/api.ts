import { z } from 'zod';

import {
  catalogFolderSchema,
  catalogItemSchema,
  catalogItemTagSchema,
  catalogSuggestionSchema,
  catalogTagSchema,
  categorySchema,
  employeeSchema,
  edgeEventSchema,
  financialOperationReportItemSchema,
  folderParameterSchema,
  hallSchema,
  inventoryStockBalanceSchema,
  inventoryStockLedgerEntrySchema,
  kitchenTimingSummaryItemSchema,
  menuItemSchema,
  modifierBindingSchema,
  modifierGroupSchema,
  modifierOptionSchema,
  olapBackfillJobSchema,
  olapExportStatusSchema,
  olapStockMoveSchema,
  olapStockMoveSummarySchema,
  pricingPolicySchema,
  recipeItemSchema,
  recipeVersionViewSchema,
  reviewAssignmentResponseSchema,
  salesKitchenSummaryItemSchema,
  recipeSuggestionSchema,
  stopListUpdateReviewSchema,
  assignDeviceResultSchema,
  assignmentStatusSchema,
  pairingCodeResultSchema,
  publicationSummarySchema,
  restaurantSchema,
  roleSchema,
  stopListReadinessSchema,
  stopListEntrySchema,
  tableSchema,
  unassignedEdgeNodeSchema,
  type AssignDeviceResult,
  type AssignmentStatus,
  type CatalogFolder,
  type CatalogItem,
  type CatalogItemTag,
  type CatalogSuggestion,
  type CatalogTag,
  type Category,
  type Employee,
  type EdgeEvent,
  type FinancialOperationReportItem,
  type FolderParameter,
  type Hall,
  type InventoryStockBalance,
  type InventoryStockLedgerEntry,
  type KitchenTimingSummaryGroupBy,
  type KitchenTimingSummaryItem,
  type OlapBackfillJob,
  type MenuItem,
  type ModifierBinding,
  type ModifierGroup,
  type ModifierOption,
  type OlapExportStatus,
  type OlapStockMove,
  type OlapStockMoveSummary,
  type PairingCodeResult,
  type PricingPolicy,
  type RecipeItem,
  type RecipeVersionView,
  type ReviewAssignmentResponse,
  type ReviewAssignmentType,
  type SalesKitchenSummaryGroupBy,
  type SalesKitchenSummaryItem,
  type RecipeSuggestion,
  type StopListUpdateReview,
  type PublicationSummary,
  type Restaurant,
  type RestaurantTable,
  type Role,
  type StopListReadiness,
  type StopListEntry,
  type UnassignedEdgeNode,
} from './schemas';

type Payload = Record<string, unknown>;
type BoundedListFilters = { limit?: number; offset?: number };
export type OlapStream = 'raw_business_events' | 'stock_moves';
export type OlapStockMoveFilters = BoundedListFilters & {
  businessDateFrom?: string;
  businessDateTo?: string;
  catalogItemId?: string;
  warehouseId?: string;
  sourceEventType?: string;
};
export type OlapStockMoveSummaryFilters = OlapStockMoveFilters & {
  groupBy?: 'business_date' | 'catalog_item' | 'warehouse';
};
export type KitchenTimingSummaryFilters = BoundedListFilters & {
  businessDateFrom?: string;
  businessDateTo?: string;
  stationId?: string;
  groupBy?: KitchenTimingSummaryGroupBy;
};

export type ApiErrorCategory = 'validation' | 'not_found' | 'conflict' | 'server' | 'network' | 'timeout' | 'unexpected';

export type ApiErrorOptions = {
  status: number;
  code: string;
  messageKey: string;
  category: ApiErrorCategory;
  details?: Record<string, string>;
  correlationId?: string;
  retryable?: boolean;
};

export type SuggestionReviewPayload = {
  reviewed_by_employee_id: string;
  review_comment?: string;
  published_by?: string;
};

export type ReviewAssignPayload = {
  command_id: string;
  assigned_to_employee_id: string;
  assigned_by_employee_id: string;
  reason: string;
};

export type ReviewUnassignPayload = {
  command_id: string;
  unassigned_by_employee_id: string;
  reason: string;
};

export class ApiError extends Error {
  status: number;
  code: string;
  messageKey: string;
  category: ApiErrorCategory;
  details: Record<string, string>;
  correlationId: string;
  retryable: boolean;

  constructor(options: ApiErrorOptions) {
    super(options.code);
    this.name = 'ApiError';
    this.status = options.status;
    this.code = options.code;
    this.messageKey = options.messageKey;
    this.category = options.category;
    this.details = options.details ?? {};
    this.correlationId = options.correlationId ?? '';
    this.retryable = options.retryable ?? false;
  }
}

function defaultApiBase() {
  const hostname = globalThis.location?.hostname;
  if (hostname === 'host.docker.internal') {
    return 'http://host.docker.internal:8090/api/v1';
  }
  return 'http://localhost:8090/api/v1';
}

const apiBase = (import.meta.env.VITE_CLOUD_API_BASE ?? defaultApiBase()).replace(/\/$/, '');
const defaultTimeoutMs = 15_000;

const backendErrorSchema = z.object({
  error: z.union([
    z.string(),
    z.object({
      code: z.string().optional(),
      message_key: z.string().optional(),
      details: z.record(z.string(), z.string()).optional(),
      correlation_id: z.string().optional(),
    }),
  ]),
});

async function request<T>(path: string, schema: z.ZodType<T>, init: RequestInit = {}) {
  const headers = new Headers(init.headers);
  if (init.body !== undefined && init.body !== null && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const controller = new AbortController();
  const timeout = globalThis.setTimeout(() => controller.abort(), defaultTimeoutMs);
  let response: Response;
  try {
    response = await fetch(`${apiBase}${path}`, { ...init, headers, signal: controller.signal });
  } catch (error) {
    throw networkApiError(error);
  } finally {
    globalThis.clearTimeout(timeout);
  }

  const data = await parseResponseBody(response);
  if (!response.ok) {
    throw apiErrorFromResponse(response, data);
  }
  try {
    return schema.parse(data);
  } catch {
    throw new ApiError({
      status: 0,
      code: 'INVALID_RESPONSE',
      messageKey: 'errors.response.invalid',
      category: 'unexpected',
    });
  }
}

async function requestOptional<T>(path: string, schema: z.ZodType<T>) {
  try {
    return await request(path, schema.nullable());
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}

async function parseResponseBody(response: Response) {
  const text = await response.text();
  if (!text.trim()) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    throw new ApiError({
      status: response.status,
      code: 'INVALID_JSON',
      messageKey: 'errors.response.invalid',
      category: 'unexpected',
      correlationId: response.headers.get('X-Request-ID') ?? '',
    });
  }
}

function apiErrorFromResponse(response: Response, data: unknown) {
  const parsed = backendErrorSchema.safeParse(data);
  const errorValue = parsed.success ? parsed.data.error : null;
  const structured = typeof errorValue === 'object' && errorValue !== null ? errorValue : null;
  const code = structured?.code ?? codeForStatus(response.status);
  return new ApiError({
    status: response.status,
    code,
    messageKey: structured?.message_key ?? messageKeyForStatus(response.status),
    category: categoryForStatus(response.status),
    details: structured?.details,
    correlationId: structured?.correlation_id ?? response.headers.get('X-Request-ID') ?? '',
    retryable: response.status === 0 || response.status >= 500,
  });
}

function networkApiError(error: unknown) {
  const aborted = typeof DOMException !== 'undefined' && error instanceof DOMException && error.name === 'AbortError';
  return new ApiError({
    status: 0,
    code: aborted ? 'REQUEST_TIMEOUT' : 'NETWORK_ERROR',
    messageKey: aborted ? 'errors.network.timeout' : 'errors.network.unavailable',
    category: aborted ? 'timeout' : 'network',
    retryable: true,
  });
}

function codeForStatus(status: number) {
  if (status === 400 || status === 422) return 'VALIDATION_FAILED';
  if (status === 404) return 'NOT_FOUND';
  if (status === 409) return 'CONFLICT';
  return status >= 500 ? 'INTERNAL_ERROR' : 'UNKNOWN_ERROR';
}

function messageKeyForStatus(status: number) {
  if (status === 400 || status === 422) return 'errors.validation';
  if (status === 404) return 'errors.notFound';
  if (status === 409) return 'errors.conflict';
  return status >= 500 ? 'errors.server' : 'errors.unknown';
}

function categoryForStatus(status: number): ApiErrorCategory {
  if (status === 400 || status === 422) return 'validation';
  if (status === 404) return 'not_found';
  if (status === 409) return 'conflict';
  if (status >= 500) return 'server';
  return 'unexpected';
}

function query(restaurantId: string) {
  return `restaurant_id=${encodeURIComponent(restaurantId)}`;
}

function suggestionQuery(restaurantId: string, status: string, limit: number, offset: number) {
  const params = new URLSearchParams();
  if (restaurantId) params.set('restaurant_id', restaurantId);
  if (status) params.set('status', status);
  params.set('limit', String(limit));
  params.set('offset', String(offset));
  return params.toString();
}

function post<T>(path: string, schema: z.ZodType<T>, payload: Payload) {
  return request(path, schema, { method: 'POST', body: JSON.stringify(payload) });
}

function patch<T>(path: string, schema: z.ZodType<T>, payload: Payload) {
  return request(path, schema, { method: 'PATCH', body: JSON.stringify(payload) });
}

export function listRestaurants(): Promise<Restaurant[]> {
  return request('/restaurants', z.array(restaurantSchema));
}

export function createRestaurant(payload: Payload) {
  return post('/restaurants', restaurantSchema, payload);
}

export function updateRestaurant(id: string, payload: Payload) {
  return patch(`/restaurants/${encodeURIComponent(id)}`, restaurantSchema, payload);
}

export function archiveRestaurant(id: string) {
  return post(`/restaurants/${encodeURIComponent(id)}/archive`, restaurantSchema, {});
}

export function listRoles(restaurantId: string): Promise<Role[]> {
  return request(`/master-data/roles?${query(restaurantId)}`, z.array(roleSchema));
}

export function createRole(payload: Payload) {
  return post('/master-data/roles', roleSchema, payload);
}

export function updateRole(id: string, payload: Payload) {
  return patch(`/master-data/roles/${encodeURIComponent(id)}`, roleSchema, payload);
}

export function archiveRole(id: string) {
  return post(`/master-data/roles/${encodeURIComponent(id)}/archive`, roleSchema, {});
}

export function listEmployees(restaurantId: string): Promise<Employee[]> {
  return request(`/master-data/employees?${query(restaurantId)}`, z.array(employeeSchema));
}

export function createEmployee(payload: Payload) {
  return post('/master-data/employees', employeeSchema, payload);
}

export function updateEmployee(id: string, payload: Payload) {
  return patch(`/master-data/employees/${encodeURIComponent(id)}`, employeeSchema, payload);
}

export function suspendEmployee(id: string) {
  return post(`/master-data/employees/${encodeURIComponent(id)}/suspend`, employeeSchema, {});
}

export function activateEmployee(id: string) {
  return post(`/master-data/employees/${encodeURIComponent(id)}/activate`, employeeSchema, {});
}

export function archiveEmployee(id: string) {
  return post(`/master-data/employees/${encodeURIComponent(id)}/archive`, employeeSchema, {});
}

export function assignEmployeeRole(id: string, roleId: string) {
  return post(`/master-data/employees/${encodeURIComponent(id)}/role`, employeeSchema, { role_id: roleId });
}

export function rotateEmployeePIN(id: string, pin: string) {
  return post(`/master-data/employees/${encodeURIComponent(id)}/pin`, employeeSchema, { pin });
}

export function listCatalogItems(restaurantId: string): Promise<CatalogItem[]> {
  return request(`/master-data/catalog/items?${query(restaurantId)}`, z.array(catalogItemSchema));
}

export function createCatalogItem(payload: Payload) {
  return post('/master-data/catalog/items', catalogItemSchema, payload);
}

export function updateCatalogItem(id: string, payload: Payload) {
  return patch(`/master-data/catalog/items/${encodeURIComponent(id)}`, catalogItemSchema, payload);
}

export function archiveCatalogItem(id: string) {
  return post(`/master-data/catalog/items/${encodeURIComponent(id)}/archive`, catalogItemSchema, {});
}

export function listCatalogFolders(restaurantId: string): Promise<CatalogFolder[]> {
  return request(`/master-data/catalog/folders?${query(restaurantId)}`, z.array(catalogFolderSchema));
}

export function createCatalogFolder(payload: Payload) {
  return post('/master-data/catalog/folders', catalogFolderSchema, payload);
}

export function updateCatalogFolder(id: string, payload: Payload) {
  return patch(`/master-data/catalog/folders/${encodeURIComponent(id)}`, catalogFolderSchema, payload);
}

export function archiveCatalogFolder(id: string) {
  return post(`/master-data/catalog/folders/${encodeURIComponent(id)}/archive`, catalogFolderSchema, {});
}

export function listFolderParameters(restaurantId: string): Promise<FolderParameter[]> {
  return request(`/master-data/catalog/folder-parameters?${query(restaurantId)}`, z.array(folderParameterSchema));
}

export function createFolderParameter(payload: Payload) {
  return post('/master-data/catalog/folder-parameters', folderParameterSchema, payload);
}

export function updateFolderParameter(id: string, payload: Payload) {
  return patch(`/master-data/catalog/folder-parameters/${encodeURIComponent(id)}`, folderParameterSchema, payload);
}

export function listCatalogTags(restaurantId: string): Promise<CatalogTag[]> {
  return request(`/master-data/catalog/tags?${query(restaurantId)}`, z.array(catalogTagSchema));
}

export function createCatalogTag(payload: Payload) {
  return post('/master-data/catalog/tags', catalogTagSchema, payload);
}

export function updateCatalogTag(id: string, payload: Payload) {
  return patch(`/master-data/catalog/tags/${encodeURIComponent(id)}`, catalogTagSchema, payload);
}

export function assignCatalogItemTag(payload: Payload): Promise<CatalogItemTag> {
  return post('/master-data/catalog/item-tags', catalogItemTagSchema, payload);
}

export function listModifierGroups(restaurantId: string): Promise<ModifierGroup[]> {
  return request(`/master-data/modifiers/groups?${query(restaurantId)}`, z.array(modifierGroupSchema));
}

export function createModifierGroup(payload: Payload) {
  return post('/master-data/modifiers/groups', modifierGroupSchema, payload);
}

export function updateModifierGroup(id: string, payload: Payload) {
  return patch(`/master-data/modifiers/groups/${encodeURIComponent(id)}`, modifierGroupSchema, payload);
}

export function listModifierOptions(restaurantId: string): Promise<ModifierOption[]> {
  return request(`/master-data/modifiers/options?${query(restaurantId)}`, z.array(modifierOptionSchema));
}

export function createModifierOption(payload: Payload) {
  return post('/master-data/modifiers/options', modifierOptionSchema, payload);
}

export function updateModifierOption(id: string, payload: Payload) {
  return patch(`/master-data/modifiers/options/${encodeURIComponent(id)}`, modifierOptionSchema, payload);
}

export function listModifierBindings(restaurantId: string): Promise<ModifierBinding[]> {
  return request(`/master-data/modifiers/bindings?${query(restaurantId)}`, z.array(modifierBindingSchema));
}

export function createModifierBinding(payload: Payload) {
  return post('/master-data/modifiers/bindings', modifierBindingSchema, payload);
}

export function updateModifierBinding(id: string, payload: Payload) {
  return patch(`/master-data/modifiers/bindings/${encodeURIComponent(id)}`, modifierBindingSchema, payload);
}

export function listPricingPolicies(restaurantId: string): Promise<PricingPolicy[]> {
  return request(`/master-data/pricing/policies?${query(restaurantId)}`, z.array(pricingPolicySchema));
}

export function createPricingPolicy(payload: Payload) {
  return post('/master-data/pricing/policies', pricingPolicySchema, payload);
}

export function updatePricingPolicy(id: string, payload: Payload) {
  return patch(`/master-data/pricing/policies/${encodeURIComponent(id)}`, pricingPolicySchema, payload);
}

export function listRecipeItems(restaurantId: string): Promise<RecipeItem[]> {
  return request(`/master-data/recipes/items?${query(restaurantId)}`, z.array(recipeItemSchema));
}

export function createRecipeItem(payload: Payload) {
  return post('/master-data/recipes/items', recipeItemSchema, payload);
}

export function updateRecipeItem(id: string, payload: Payload) {
  return patch(`/master-data/recipes/items/${encodeURIComponent(id)}`, recipeItemSchema, payload);
}

export function listRecipeVersions(restaurantId: string, ownerCatalogItemId = '', status = '', limit = 50, offset = 0): Promise<RecipeVersionView[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (ownerCatalogItemId) params.set('owner_catalog_item_id', ownerCatalogItemId);
  if (status) params.set('status', status);
  params.set('limit', String(limit));
  params.set('offset', String(offset));
  return request(`/master-data/recipes/versions?${params.toString()}`, z.array(recipeVersionViewSchema));
}

export function createRecipeVersionDraft(payload: Payload): Promise<RecipeVersionView> {
  return post('/master-data/recipes/versions/drafts', recipeVersionViewSchema, payload);
}

export function submitRecipeVersion(id: string, payload: Payload): Promise<RecipeSuggestion> {
  return post(`/master-data/recipes/versions/${encodeURIComponent(id)}/submit`, recipeSuggestionSchema, payload);
}

export function listCatalogSuggestions(restaurantId: string, status = 'pending', limit = 100, offset = 0): Promise<CatalogSuggestion[]> {
  return request(`/master-data/catalog-suggestions?${suggestionQuery(restaurantId, status, limit, offset)}`, z.array(catalogSuggestionSchema));
}

export function approveCatalogSuggestion(id: string, payload: SuggestionReviewPayload): Promise<CatalogSuggestion> {
  return post(`/master-data/catalog-suggestions/${encodeURIComponent(id)}/approve`, catalogSuggestionSchema, payload);
}

export function rejectCatalogSuggestion(id: string, payload: SuggestionReviewPayload): Promise<CatalogSuggestion> {
  return post(`/master-data/catalog-suggestions/${encodeURIComponent(id)}/reject`, catalogSuggestionSchema, payload);
}

export function requestChangesCatalogSuggestion(id: string, payload: SuggestionReviewPayload): Promise<CatalogSuggestion> {
  return post(`/master-data/catalog-suggestions/${encodeURIComponent(id)}/request-changes`, catalogSuggestionSchema, payload);
}

export function listRecipeSuggestions(restaurantId: string, status = 'pending', limit = 100, offset = 0): Promise<RecipeSuggestion[]> {
  return request(`/master-data/recipe-suggestions?${suggestionQuery(restaurantId, status, limit, offset)}`, z.array(recipeSuggestionSchema));
}

export function approveRecipeSuggestion(id: string, payload: SuggestionReviewPayload): Promise<RecipeSuggestion> {
  return post(`/master-data/recipe-suggestions/${encodeURIComponent(id)}/approve`, recipeSuggestionSchema, payload);
}

export function rejectRecipeSuggestion(id: string, payload: SuggestionReviewPayload): Promise<RecipeSuggestion> {
  return post(`/master-data/recipe-suggestions/${encodeURIComponent(id)}/reject`, recipeSuggestionSchema, payload);
}

export function requestChangesRecipeSuggestion(id: string, payload: SuggestionReviewPayload): Promise<RecipeSuggestion> {
  return post(`/master-data/recipe-suggestions/${encodeURIComponent(id)}/request-changes`, recipeSuggestionSchema, payload);
}

export function listStopListUpdateReviews(restaurantId: string, status = 'pending', limit = 100, offset = 0): Promise<StopListUpdateReview[]> {
  return request(`/manager/stop-list-updates?${suggestionQuery(restaurantId, status, limit, offset)}`, z.array(stopListUpdateReviewSchema));
}

export function approveStopListUpdateReview(id: string, payload: SuggestionReviewPayload): Promise<StopListUpdateReview> {
  return post(`/manager/stop-list-updates/${encodeURIComponent(id)}/approve`, stopListUpdateReviewSchema, payload);
}

export function rejectStopListUpdateReview(id: string, payload: SuggestionReviewPayload): Promise<StopListUpdateReview> {
  return post(`/manager/stop-list-updates/${encodeURIComponent(id)}/reject`, stopListUpdateReviewSchema, payload);
}

export function requestChangesStopListUpdateReview(id: string, payload: SuggestionReviewPayload): Promise<StopListUpdateReview> {
  return post(`/manager/stop-list-updates/${encodeURIComponent(id)}/request-changes`, stopListUpdateReviewSchema, payload);
}

export function assignReviewItem(reviewType: ReviewAssignmentType, id: string, payload: ReviewAssignPayload): Promise<ReviewAssignmentResponse> {
  if (reviewType !== 'stop_list_update') {
    return Promise.reject(new Error('review assignment is supported only for stop_list_update'));
  }
  return post(`/manager/stop-list-updates/${encodeURIComponent(id)}/assign`, reviewAssignmentResponseSchema, payload);
}

export function unassignReviewItem(reviewType: ReviewAssignmentType, id: string, payload: ReviewUnassignPayload): Promise<ReviewAssignmentResponse> {
  if (reviewType !== 'stop_list_update') {
    return Promise.reject(new Error('review assignment is supported only for stop_list_update'));
  }
  return post(`/manager/stop-list-updates/${encodeURIComponent(id)}/unassign`, reviewAssignmentResponseSchema, payload);
}

export function listStopListEntries(restaurantId: string): Promise<StopListEntry[]> {
  return request(`/master-data/inventory/stop-list?${query(restaurantId)}`, z.array(stopListEntrySchema));
}

export function getStopListReadiness(restaurantId: string, nodeDeviceId = ''): Promise<StopListReadiness> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (nodeDeviceId) params.set('node_device_id', nodeDeviceId);
  return request(`/sync/readiness/stop-list?${params.toString()}`, stopListReadinessSchema);
}

export function listInventoryStockBalances(
  restaurantId: string,
  filters: { warehouseId?: string; catalogItemId?: string; businessDateTo?: string; costingStatus?: string; limit?: number; offset?: number } = {},
): Promise<InventoryStockBalance[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (filters.warehouseId) params.set('warehouse_id', filters.warehouseId);
  if (filters.catalogItemId) params.set('catalog_item_id', filters.catalogItemId);
  if (filters.businessDateTo) params.set('business_date_to', filters.businessDateTo);
  if (filters.costingStatus) params.set('costing_status', filters.costingStatus);
  params.set('limit', String(filters.limit ?? 50));
  params.set('offset', String(filters.offset ?? 0));
  return request(`/inventory/stock-balances?${params.toString()}`, z.array(inventoryStockBalanceSchema));
}

export function listInventoryStockLedger(
  restaurantId: string,
  filters: { sourceEventType?: string; sourceEventId?: string; orderLineId?: string; catalogItemId?: string; limit?: number; offset?: number } = {},
): Promise<InventoryStockLedgerEntry[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (filters.sourceEventType) params.set('source_event_type', filters.sourceEventType);
  if (filters.sourceEventId) params.set('source_event_id', filters.sourceEventId);
  if (filters.orderLineId) params.set('order_line_id', filters.orderLineId);
  if (filters.catalogItemId) params.set('catalog_item_id', filters.catalogItemId);
  params.set('limit', String(filters.limit ?? 50));
  params.set('offset', String(filters.offset ?? 0));
  return request(`/inventory/stock-ledger?${params.toString()}`, z.array(inventoryStockLedgerEntrySchema));
}

export function getOlapExportStatus(stream: OlapStream): Promise<OlapExportStatus> {
  const params = new URLSearchParams();
  params.set('stream', stream);
  return request(`/olap/export-status?${params.toString()}`, olapExportStatusSchema);
}

export function listOlapStockMoves(
  restaurantId: string,
  filters: OlapStockMoveFilters = {},
): Promise<OlapStockMove[]> {
  const params = olapStockMoveParams(restaurantId, filters);
  return request(`/olap/stock-moves?${params.toString()}`, z.array(olapStockMoveSchema));
}

export function listOlapStockMoveSummary(
  restaurantId: string,
  filters: OlapStockMoveSummaryFilters = {},
): Promise<OlapStockMoveSummary[]> {
  const params = olapStockMoveParams(restaurantId, filters);
  if (filters.groupBy) params.set('group_by', filters.groupBy);
  return request(`/olap/stock-move-summary?${params.toString()}`, z.array(olapStockMoveSummarySchema));
}

export function listOlapBackfillJobs(filters: { stream?: OlapStream; status?: string; limit?: number; offset?: number } = {}): Promise<OlapBackfillJob[]> {
  const params = new URLSearchParams();
  if (filters.stream) params.set('stream', filters.stream);
  if (filters.status) params.set('status', filters.status);
  params.set('limit', String(filters.limit ?? 50));
  params.set('offset', String(filters.offset ?? 0));
  return request(`/olap/backfill-jobs?${params.toString()}`, z.array(olapBackfillJobSchema));
}

export function listKitchenTimingSummary(
  restaurantId: string,
  filters: KitchenTimingSummaryFilters = {},
): Promise<KitchenTimingSummaryItem[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (filters.businessDateFrom) params.set('business_date_from', filters.businessDateFrom);
  if (filters.businessDateTo) params.set('business_date_to', filters.businessDateTo);
  if (filters.stationId) params.set('station_id', filters.stationId);
  if (filters.groupBy) params.set('group_by', filters.groupBy);
  params.set('limit', String(filters.limit ?? 50));
  params.set('offset', String(filters.offset ?? 0));
  return request(`/olap/kitchen-timing-summary?${params.toString()}`, z.array(kitchenTimingSummaryItemSchema));
}

function olapStockMoveParams(restaurantId: string, filters: OlapStockMoveFilters) {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (filters.businessDateFrom) params.set('business_date_from', filters.businessDateFrom);
  if (filters.businessDateTo) params.set('business_date_to', filters.businessDateTo);
  if (filters.catalogItemId) params.set('catalog_item_id', filters.catalogItemId);
  if (filters.warehouseId) params.set('warehouse_id', filters.warehouseId);
  if (filters.sourceEventType) params.set('source_event_type', filters.sourceEventType);
  params.set('limit', String(filters.limit ?? 50));
  params.set('offset', String(filters.offset ?? 0));
  return params;
}

export function listFinancialOperations(
  restaurantId: string,
  filters: { businessDateFrom?: string; businessDateTo?: string; operationType?: string; shiftId?: string; originalShiftId?: string; checkId?: string; limit?: number; offset?: number } = {},
): Promise<FinancialOperationReportItem[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (filters.businessDateFrom) params.set('business_date_from', filters.businessDateFrom);
  if (filters.businessDateTo) params.set('business_date_to', filters.businessDateTo);
  if (filters.operationType) params.set('operation_type', filters.operationType);
  if (filters.shiftId) params.set('shift_id', filters.shiftId);
  if (filters.originalShiftId) params.set('original_shift_id', filters.originalShiftId);
  if (filters.checkId) params.set('check_id', filters.checkId);
  params.set('limit', String(filters.limit ?? 50));
  params.set('offset', String(filters.offset ?? 0));
  return request(`/reporting/financial-operations?${params.toString()}`, z.array(financialOperationReportItemSchema));
}

export function listSalesKitchenSummary(
  restaurantId: string,
  filters: { businessDateFrom?: string; businessDateTo?: string; groupBy?: SalesKitchenSummaryGroupBy; limit?: number; offset?: number } = {},
): Promise<SalesKitchenSummaryItem[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  if (filters.businessDateFrom) params.set('business_date_from', filters.businessDateFrom);
  if (filters.businessDateTo) params.set('business_date_to', filters.businessDateTo);
  if (filters.groupBy) params.set('group_by', filters.groupBy);
  params.set('limit', String(filters.limit ?? 50));
  params.set('offset', String(filters.offset ?? 0));
  return request(`/olap/sales-kitchen-summary?${params.toString()}`, z.array(salesKitchenSummaryItemSchema));
}

export function upsertStopListEntry(payload: Payload) {
  return post('/master-data/inventory/stop-list', stopListEntrySchema, payload);
}

export function updateStopListEntry(id: string, payload: Payload) {
  return patch(`/master-data/inventory/stop-list/${encodeURIComponent(id)}`, stopListEntrySchema, payload);
}

export function deactivateStopListEntry(id: string) {
  return post(`/master-data/inventory/stop-list/${encodeURIComponent(id)}/deactivate`, stopListEntrySchema, {});
}

export function createCategory(payload: Payload): Promise<Category> {
  return post('/master-data/menu/categories', categorySchema, payload);
}

export function listHalls(restaurantId: string): Promise<Hall[]> {
  return request(`/master-data/floor/halls?${query(restaurantId)}`, z.array(hallSchema));
}

export function createHall(payload: Payload) {
  return post('/master-data/floor/halls', hallSchema, payload);
}

export function updateHall(id: string, payload: Payload) {
  return patch(`/master-data/floor/halls/${encodeURIComponent(id)}`, hallSchema, payload);
}

export function archiveHall(id: string) {
  return post(`/master-data/floor/halls/${encodeURIComponent(id)}/archive`, hallSchema, {});
}

export function listTables(restaurantId: string): Promise<RestaurantTable[]> {
  return request(`/master-data/floor/tables?${query(restaurantId)}`, z.array(tableSchema));
}

export function createTable(payload: Payload) {
  return post('/master-data/floor/tables', tableSchema, payload);
}

export function updateTable(id: string, payload: Payload) {
  return patch(`/master-data/floor/tables/${encodeURIComponent(id)}`, tableSchema, payload);
}

export function archiveTable(id: string) {
  return post(`/master-data/floor/tables/${encodeURIComponent(id)}/archive`, tableSchema, {});
}

export function listMenuItems(restaurantId: string): Promise<MenuItem[]> {
  return request(`/master-data/menu/items?${query(restaurantId)}`, z.array(menuItemSchema));
}

export function createMenuItem(payload: Payload) {
  return post('/master-data/menu/items', menuItemSchema, payload);
}

export function updateMenuItem(id: string, payload: Payload) {
  return patch(`/master-data/menu/items/${encodeURIComponent(id)}`, menuItemSchema, payload);
}

export function archiveMenuItem(id: string) {
  return post(`/master-data/menu/items/${encodeURIComponent(id)}/archive`, menuItemSchema, {});
}

export function publishMasterData(restaurantId: string, payload: Payload): Promise<PublicationSummary> {
  return post(`/restaurants/${encodeURIComponent(restaurantId)}/master-data/publish`, publicationSummarySchema, payload);
}

export function getPublicationState(restaurantId: string): Promise<PublicationSummary | null> {
  return requestOptional(`/restaurants/${encodeURIComponent(restaurantId)}/master-data/publication-state`, publicationSummarySchema);
}


export function listUnassignedDevices(): Promise<UnassignedEdgeNode[]> {
  return request('/devices/unassigned', z.array(unassignedEdgeNodeSchema));
}

export function assignDeviceToRestaurant(restaurantId: string, nodeDeviceId: string): Promise<AssignDeviceResult> {
  return post(`/restaurants/${encodeURIComponent(restaurantId)}/devices/${encodeURIComponent(nodeDeviceId)}/assign`, assignDeviceResultSchema, {});
}

export function getAssignmentStatus(nodeDeviceId: string): Promise<AssignmentStatus> {
  return request(`/devices/${encodeURIComponent(nodeDeviceId)}/assignment-status`, assignmentStatusSchema);
}

export function generatePairingCode(restaurantId: string, payload: Payload): Promise<PairingCodeResult> {
  return post(`/restaurants/${encodeURIComponent(restaurantId)}/devices/generate-pairing-code`, pairingCodeResultSchema, payload);
}

export function listEdgeEvents(restaurantId: string, limit = 50): Promise<EdgeEvent[]> {
  const params = new URLSearchParams();
  if (restaurantId) params.set('restaurant_id', restaurantId);
  params.set('limit', String(limit));
  return request(`/sync/edge-events?${params.toString()}`, z.array(edgeEventSchema));
}
