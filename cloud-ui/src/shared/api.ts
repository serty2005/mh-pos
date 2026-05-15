import { z } from 'zod';

import {
  catalogFolderSchema,
  catalogItemSchema,
  catalogItemTagSchema,
  catalogTagSchema,
  categorySchema,
  employeeSchema,
  folderParameterSchema,
  hallSchema,
  menuItemSchema,
  modifierBindingSchema,
  modifierGroupSchema,
  modifierOptionSchema,
  pricingPolicySchema,
  publicationSummarySchema,
  restaurantSchema,
  roleSchema,
  tableSchema,
  type CatalogFolder,
  type CatalogItem,
  type CatalogItemTag,
  type CatalogTag,
  type Category,
  type Employee,
  type FolderParameter,
  type Hall,
  type MenuItem,
  type ModifierBinding,
  type ModifierGroup,
  type ModifierOption,
  type PricingPolicy,
  type PublicationSummary,
  type Restaurant,
  type RestaurantTable,
  type Role,
} from './schemas';

type Payload = Record<string, unknown>;

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
    return await request(path, schema);
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
  return request(`/roles?${query(restaurantId)}`, z.array(roleSchema));
}

export function createRole(payload: Payload) {
  return post('/roles', roleSchema, payload);
}

export function updateRole(id: string, payload: Payload) {
  return patch(`/roles/${encodeURIComponent(id)}`, roleSchema, payload);
}

export function archiveRole(id: string) {
  return post(`/roles/${encodeURIComponent(id)}/archive`, roleSchema, {});
}

export function listEmployees(restaurantId: string): Promise<Employee[]> {
  return request(`/employees?${query(restaurantId)}`, z.array(employeeSchema));
}

export function createEmployee(payload: Payload) {
  return post('/employees', employeeSchema, payload);
}

export function updateEmployee(id: string, payload: Payload) {
  return patch(`/employees/${encodeURIComponent(id)}`, employeeSchema, payload);
}

export function suspendEmployee(id: string) {
  return post(`/employees/${encodeURIComponent(id)}/suspend`, employeeSchema, {});
}

export function activateEmployee(id: string) {
  return post(`/employees/${encodeURIComponent(id)}/activate`, employeeSchema, {});
}

export function archiveEmployee(id: string) {
  return post(`/employees/${encodeURIComponent(id)}/archive`, employeeSchema, {});
}

export function assignEmployeeRole(id: string, roleId: string) {
  return post(`/master-data/employees/${encodeURIComponent(id)}/role`, employeeSchema, { role_id: roleId });
}

export function rotateEmployeePIN(id: string, pin: string) {
  return post(`/employees/${encodeURIComponent(id)}/pin`, employeeSchema, { pin });
}

export function listCatalogItems(restaurantId: string): Promise<CatalogItem[]> {
  return request(`/catalog/items?${query(restaurantId)}`, z.array(catalogItemSchema));
}

export function createCatalogItem(payload: Payload) {
  return post('/catalog/items', catalogItemSchema, payload);
}

export function updateCatalogItem(id: string, payload: Payload) {
  return patch(`/catalog/items/${encodeURIComponent(id)}`, catalogItemSchema, payload);
}

export function archiveCatalogItem(id: string) {
  return post(`/catalog/items/${encodeURIComponent(id)}/archive`, catalogItemSchema, {});
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

export function createCategory(payload: Payload): Promise<Category> {
  return post('/master-data/menu/categories', categorySchema, payload);
}

export function listHalls(restaurantId: string): Promise<Hall[]> {
  return request(`/halls?${query(restaurantId)}`, z.array(hallSchema));
}

export function createHall(payload: Payload) {
  return post('/halls', hallSchema, payload);
}

export function updateHall(id: string, payload: Payload) {
  return patch(`/halls/${encodeURIComponent(id)}`, hallSchema, payload);
}

export function archiveHall(id: string) {
  return post(`/halls/${encodeURIComponent(id)}/archive`, hallSchema, {});
}

export function listTables(restaurantId: string): Promise<RestaurantTable[]> {
  return request(`/tables?${query(restaurantId)}`, z.array(tableSchema));
}

export function createTable(payload: Payload) {
  return post('/tables', tableSchema, payload);
}

export function updateTable(id: string, payload: Payload) {
  return patch(`/tables/${encodeURIComponent(id)}`, tableSchema, payload);
}

export function archiveTable(id: string) {
  return post(`/tables/${encodeURIComponent(id)}/archive`, tableSchema, {});
}

export function listMenuItems(restaurantId: string): Promise<MenuItem[]> {
  return request(`/menu/items?${query(restaurantId)}`, z.array(menuItemSchema));
}

export function createMenuItem(payload: Payload) {
  return post('/menu/items', menuItemSchema, payload);
}

export function updateMenuItem(id: string, payload: Payload) {
  return patch(`/menu/items/${encodeURIComponent(id)}`, menuItemSchema, payload);
}

export function archiveMenuItem(id: string) {
  return post(`/menu/items/${encodeURIComponent(id)}/archive`, menuItemSchema, {});
}

export function publishMasterData(restaurantId: string, payload: Payload): Promise<PublicationSummary> {
  return post(`/restaurants/${encodeURIComponent(restaurantId)}/master-data/publish`, publicationSummarySchema, payload);
}

export function getPublicationState(restaurantId: string): Promise<PublicationSummary | null> {
  return requestOptional(`/restaurants/${encodeURIComponent(restaurantId)}/master-data/publication-state`, publicationSummarySchema);
}
