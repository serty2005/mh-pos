import { z } from 'zod';
import { request, requestOptional } from './client';
import {
  catalogItemSchema,
  catalogFolderSchema,
  catalogItemTagSchema,
  catalogTagSchema,
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
  assignmentStatusSchema,
  assignDeviceResultSchema,
  edgeEventSchema,
  pairingCodeResultSchema,
  unassignedEdgeNodeSchema,
  type AssignmentStatus,
  type AssignDeviceResult,
  type CatalogItem,
  type CatalogFolder,
  type CatalogItemTag,
  type CatalogTag,
  type Employee,
  type EdgeEvent,
  type Hall,
  type FolderParameter,
  type MenuItem,
  type ModifierBinding,
  type ModifierGroup,
  type ModifierOption,
  type PairingCodeResult,
  type PricingPolicy,
  type PublicationSummary,
  type Restaurant,
  type RestaurantTable,
  type Role,
  type UnassignedEdgeNode,
} from './schemas';

type Payload = Record<string, unknown>;

function post<T>(path: string, schema: z.ZodType<T>, payload: Payload) {
  return request(path, schema, { method: 'POST', body: JSON.stringify(payload) });
}

function patch<T>(path: string, schema: z.ZodType<T>, payload: Payload) {
  return request(path, schema, { method: 'PATCH', body: JSON.stringify(payload) });
}

function query(restaurantId: string) {
  return `restaurant_id=${encodeURIComponent(restaurantId)}`;
}

export function listRestaurants(): Promise<Restaurant[]> {
  return request('/restaurants', z.array(restaurantSchema));
}

export function createRestaurant(payload: Payload): Promise<Restaurant> {
  return post('/restaurants', restaurantSchema, payload);
}

export function updateRestaurant(id: string, payload: Payload): Promise<Restaurant> {
  return patch(`/restaurants/${encodeURIComponent(id)}`, restaurantSchema, payload);
}

export function archiveRestaurant(id: string): Promise<Restaurant> {
  return post(`/restaurants/${encodeURIComponent(id)}/archive`, restaurantSchema, {});
}

export function listRoles(restaurantId: string): Promise<Role[]> {
  return request(`/master-data/roles?${query(restaurantId)}`, z.array(roleSchema));
}

export function listEmployees(restaurantId: string): Promise<Employee[]> {
  return request(`/master-data/employees?${query(restaurantId)}`, z.array(employeeSchema));
}

export function listCatalogItems(restaurantId: string): Promise<CatalogItem[]> {
  return request(`/master-data/catalog/items?${query(restaurantId)}`, z.array(catalogItemSchema));
}

export function listMenuItems(restaurantId: string): Promise<MenuItem[]> {
  return request(`/master-data/menu/items?${query(restaurantId)}`, z.array(menuItemSchema));
}

export function createCatalogItem(payload: Payload): Promise<CatalogItem> {
  return post('/master-data/catalog/items', catalogItemSchema, payload);
}

export function updateCatalogItem(id: string, payload: Payload): Promise<CatalogItem> {
  return patch(`/master-data/catalog/items/${encodeURIComponent(id)}`, catalogItemSchema, payload);
}

export function archiveCatalogItem(id: string): Promise<CatalogItem> {
  return post(`/master-data/catalog/items/${encodeURIComponent(id)}/archive`, catalogItemSchema, {});
}

export function listCatalogFolders(restaurantId: string): Promise<CatalogFolder[]> {
  return request(`/master-data/catalog/folders?${query(restaurantId)}`, z.array(catalogFolderSchema));
}

export function createCatalogFolder(payload: Payload): Promise<CatalogFolder> {
  return post('/master-data/catalog/folders', catalogFolderSchema, payload);
}

export function updateCatalogFolder(id: string, payload: Payload): Promise<CatalogFolder> {
  return patch(`/master-data/catalog/folders/${encodeURIComponent(id)}`, catalogFolderSchema, payload);
}

export function archiveCatalogFolder(id: string): Promise<CatalogFolder> {
  return post(`/master-data/catalog/folders/${encodeURIComponent(id)}/archive`, catalogFolderSchema, {});
}

export function listFolderParameters(restaurantId: string): Promise<FolderParameter[]> {
  return request(`/master-data/catalog/folder-parameters?${query(restaurantId)}`, z.array(folderParameterSchema));
}

export function createFolderParameter(payload: Payload): Promise<FolderParameter> {
  return post('/master-data/catalog/folder-parameters', folderParameterSchema, payload);
}

export function updateFolderParameter(id: string, payload: Payload): Promise<FolderParameter> {
  return patch(`/master-data/catalog/folder-parameters/${encodeURIComponent(id)}`, folderParameterSchema, payload);
}

export function listCatalogTags(restaurantId: string): Promise<CatalogTag[]> {
  return request(`/master-data/catalog/tags?${query(restaurantId)}`, z.array(catalogTagSchema));
}

export function createCatalogTag(payload: Payload): Promise<CatalogTag> {
  return post('/master-data/catalog/tags', catalogTagSchema, payload);
}

export function updateCatalogTag(id: string, payload: Payload): Promise<CatalogTag> {
  return patch(`/master-data/catalog/tags/${encodeURIComponent(id)}`, catalogTagSchema, payload);
}

export function assignCatalogItemTag(payload: Payload): Promise<CatalogItemTag> {
  return post('/master-data/catalog/item-tags', catalogItemTagSchema, payload);
}

export function listModifierGroups(restaurantId: string): Promise<ModifierGroup[]> {
  return request(`/master-data/modifiers/groups?${query(restaurantId)}`, z.array(modifierGroupSchema));
}

export function listModifierOptions(restaurantId: string): Promise<ModifierOption[]> {
  return request(`/master-data/modifiers/options?${query(restaurantId)}`, z.array(modifierOptionSchema));
}

export function listModifierBindings(restaurantId: string): Promise<ModifierBinding[]> {
  return request(`/master-data/modifiers/bindings?${query(restaurantId)}`, z.array(modifierBindingSchema));
}

export function listPricingPolicies(restaurantId: string): Promise<PricingPolicy[]> {
  return request(`/master-data/pricing/policies?${query(restaurantId)}`, z.array(pricingPolicySchema));
}

export function listHalls(restaurantId: string): Promise<Hall[]> {
  return request(`/master-data/floor/halls?${query(restaurantId)}`, z.array(hallSchema));
}

export function listTables(restaurantId: string): Promise<RestaurantTable[]> {
  return request(`/master-data/floor/tables?${query(restaurantId)}`, z.array(tableSchema));
}

export function listUnassignedDevices(): Promise<UnassignedEdgeNode[]> {
  return request('/devices/unassigned', z.array(unassignedEdgeNodeSchema));
}

export function assignDeviceToRestaurant(restaurantId: string, nodeDeviceId: string): Promise<AssignDeviceResult> {
  return post(
    `/restaurants/${encodeURIComponent(restaurantId)}/devices/${encodeURIComponent(nodeDeviceId)}/assign`,
    assignDeviceResultSchema,
    {},
  );
}

export function getAssignmentStatus(nodeDeviceId: string): Promise<AssignmentStatus> {
  return request(`/devices/${encodeURIComponent(nodeDeviceId)}/assignment-status`, assignmentStatusSchema);
}

export function generatePairingCode(
  restaurantId: string,
  payload: { ttl_seconds?: number },
): Promise<PairingCodeResult> {
  return post(
    `/restaurants/${encodeURIComponent(restaurantId)}/devices/generate-pairing-code`,
    pairingCodeResultSchema,
    payload,
  );
}

export function listEdgeEvents(restaurantId: string, limit = 50): Promise<EdgeEvent[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  params.set('limit', String(limit));
  return request(`/sync/edge-events?${params.toString()}`, z.array(edgeEventSchema));
}

export function getPublicationState(restaurantId: string): Promise<PublicationSummary | null> {
  return requestOptional(
    `/restaurants/${encodeURIComponent(restaurantId)}/master-data/publication-state`,
    publicationSummarySchema,
  );
}

export function publishMasterData(restaurantId: string, payload: Payload): Promise<PublicationSummary> {
  return post(`/restaurants/${encodeURIComponent(restaurantId)}/master-data/publish`, publicationSummarySchema, payload);
}
