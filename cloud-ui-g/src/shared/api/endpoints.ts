import { z } from 'zod';
import { request, requestOptional } from './client';
import {
  catalogItemSchema,
  catalogFolderSchema,
  catalogItemTagSchema,
  catalogTagSchema,
  categorySchema,
  deliveryStatusSchema,
  employeeSchema,
  folderParameterSchema,
  hallSchema,
  menuItemSchema,
  modifierBindingSchema,
  modifierGroupSchema,
  modifierOptionSchema,
  pricingPolicyPackageSchema,
  pricingPolicySchema,
  masterDataPackageSchema,
  printerSchema,
  publicationSummarySchema,
  receiptTemplateSchema,
  restaurantSchema,
  restaurantEdgeNodeSchema,
  restaurantSectionSchema,
  roleSchema,
  tableSchema,
  assignmentStatusSchema,
  assignDeviceResultSchema,
  edgeEventSchema,
  pairingCodeResultSchema,
  unassignedEdgeNodeSchema,
  entitlementSnapshotSchema,
  type AssignmentStatus,
  type AssignDeviceResult,
  type CatalogItem,
  type CatalogFolder,
  type CatalogItemTag,
  type CatalogTag,
  type Category,
  type Employee,
  type EdgeEvent,
  type Hall,
  type FolderParameter,
  type MenuItem,
  type ModifierBinding,
  type ModifierGroup,
  type ModifierOption,
  type PairingCodeResult,
  type Printer,
  type PricingPolicyPackage,
  type MasterDataPackage,
  type PricingPolicy,
  type PublicationSummary,
  type ReceiptTemplate,
  type Restaurant,
  type RestaurantEdgeNode,
  type RestaurantSection,
  type RestaurantTable,
  type Role,
  type UnassignedEdgeNode,
  type EntitlementSnapshot,
} from './schemas';

type Payload = Record<string, unknown>;

function post<T>(path: string, schema: z.ZodType<T>, payload: Payload) {
  return request(path, schema, { method: 'POST', body: JSON.stringify(payload) });
}

function patch<T>(path: string, schema: z.ZodType<T>, payload: Payload) {
  return request(path, schema, { method: 'PATCH', body: JSON.stringify(payload) });
}

function put<T>(path: string, schema: z.ZodType<T>, payload: Payload) {
  return request(path, schema, { method: 'PUT', body: JSON.stringify(payload) });
}

function query(restaurantId: string) {
  return `restaurant_id=${encodeURIComponent(restaurantId)}`;
}

function optionalRestaurantQuery(restaurantId?: string) {
  const trimmed = restaurantId?.trim() ?? '';
  return trimmed ? `?${query(trimmed)}` : '';
}

export function listRestaurants(): Promise<Restaurant[]> {
  return request('/restaurants', z.array(restaurantSchema));
}

export function getEntitlements(): Promise<EntitlementSnapshot> {
  return request('/license/entitlements', entitlementSnapshotSchema);
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

export function listRoles(): Promise<Role[]> {
  return request('/master-data/roles', z.array(roleSchema));
}

export function createRole(payload: Payload): Promise<Role> {
  return post('/master-data/roles', roleSchema, payload);
}

export function updateRole(id: string, payload: Payload): Promise<Role> {
  return patch(`/master-data/roles/${encodeURIComponent(id)}`, roleSchema, payload);
}

export function archiveRole(id: string): Promise<Role> {
  return post(`/master-data/roles/${encodeURIComponent(id)}/archive`, roleSchema, {});
}

export function listEmployees(): Promise<Employee[]> {
  return request('/master-data/employees', z.array(employeeSchema));
}

export function createEmployee(payload: Payload): Promise<Employee> {
  return post('/master-data/employees', employeeSchema, payload);
}

export function updateEmployee(id: string, payload: Payload): Promise<Employee> {
  return patch(`/master-data/employees/${encodeURIComponent(id)}`, employeeSchema, payload);
}

export function suspendEmployee(id: string): Promise<Employee> {
  return post(`/master-data/employees/${encodeURIComponent(id)}/suspend`, employeeSchema, {});
}

export function activateEmployee(id: string): Promise<Employee> {
  return post(`/master-data/employees/${encodeURIComponent(id)}/activate`, employeeSchema, {});
}

export function archiveEmployee(id: string): Promise<Employee> {
  return post(`/master-data/employees/${encodeURIComponent(id)}/archive`, employeeSchema, {});
}

export function assignEmployeeRole(id: string, roleId: string): Promise<Employee> {
  return post(`/master-data/employees/${encodeURIComponent(id)}/role`, employeeSchema, { role_id: roleId });
}

export function rotateEmployeePIN(id: string, pin: string): Promise<Employee> {
  return post(`/master-data/employees/${encodeURIComponent(id)}/pin`, employeeSchema, { pin });
}

export function listCatalogItems(restaurantId?: string): Promise<CatalogItem[]> {
  return request(`/master-data/catalog/items${optionalRestaurantQuery(restaurantId)}`, z.array(catalogItemSchema));
}

export function listMenuItems(restaurantId: string): Promise<MenuItem[]> {
  return request(`/master-data/menu/items?${query(restaurantId)}`, z.array(menuItemSchema));
}

export function listMenuCategories(restaurantId: string): Promise<Category[]> {
  return request(`/master-data/menu/categories?${query(restaurantId)}`, z.array(categorySchema));
}

export function createMenuCategory(payload: Payload): Promise<Category> {
  return post('/master-data/menu/categories', categorySchema, payload);
}

export function updateMenuCategory(id: string, payload: Payload): Promise<Category> {
  return patch(`/master-data/menu/categories/${encodeURIComponent(id)}`, categorySchema, payload);
}

export function archiveMenuCategory(id: string): Promise<Category> {
  return post(`/master-data/menu/categories/${encodeURIComponent(id)}/archive`, categorySchema, {});
}

export function createMenuItem(payload: Payload): Promise<MenuItem> {
  return post('/master-data/menu/items', menuItemSchema, payload);
}

export function updateMenuItem(id: string, payload: Payload): Promise<MenuItem> {
  return patch(`/master-data/menu/items/${encodeURIComponent(id)}`, menuItemSchema, payload);
}

export function archiveMenuItem(id: string): Promise<MenuItem> {
  return post(`/master-data/menu/items/${encodeURIComponent(id)}/archive`, menuItemSchema, {});
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

export function listCatalogFolders(restaurantId?: string): Promise<CatalogFolder[]> {
  return request(`/master-data/catalog/folders${optionalRestaurantQuery(restaurantId)}`, z.array(catalogFolderSchema));
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

export function listCatalogTags(restaurantId?: string): Promise<CatalogTag[]> {
  return request(`/master-data/catalog/tags${optionalRestaurantQuery(restaurantId)}`, z.array(catalogTagSchema));
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

export function createModifierGroup(payload: Payload): Promise<ModifierGroup> {
  return post('/master-data/modifiers/groups', modifierGroupSchema, payload);
}

export function updateModifierGroup(id: string, payload: Payload): Promise<ModifierGroup> {
  return patch(`/master-data/modifiers/groups/${encodeURIComponent(id)}`, modifierGroupSchema, payload);
}

export function listModifierOptions(restaurantId: string): Promise<ModifierOption[]> {
  return request(`/master-data/modifiers/options?${query(restaurantId)}`, z.array(modifierOptionSchema));
}

export function createModifierOption(payload: Payload): Promise<ModifierOption> {
  return post('/master-data/modifiers/options', modifierOptionSchema, payload);
}

export function updateModifierOption(id: string, payload: Payload): Promise<ModifierOption> {
  return patch(`/master-data/modifiers/options/${encodeURIComponent(id)}`, modifierOptionSchema, payload);
}

export function listModifierBindings(restaurantId: string): Promise<ModifierBinding[]> {
  return request(`/master-data/modifiers/bindings?${query(restaurantId)}`, z.array(modifierBindingSchema));
}

export function createModifierBinding(payload: Payload): Promise<ModifierBinding> {
  return post('/master-data/modifiers/bindings', modifierBindingSchema, payload);
}

export function updateModifierBinding(id: string, payload: Payload): Promise<ModifierBinding> {
  return patch(`/master-data/modifiers/bindings/${encodeURIComponent(id)}`, modifierBindingSchema, payload);
}

export function listPricingPolicies(restaurantId: string): Promise<PricingPolicy[]> {
  return request(`/master-data/pricing/policies?${query(restaurantId)}`, z.array(pricingPolicySchema));
}

export function createPricingPolicy(payload: Payload): Promise<PricingPolicy> {
  return post('/master-data/pricing/policies', pricingPolicySchema, payload);
}

export function updatePricingPolicy(id: string, payload: Payload): Promise<PricingPolicy> {
  return patch(`/master-data/pricing/policies/${encodeURIComponent(id)}`, pricingPolicySchema, payload);
}

export function getPricingPolicyPackage(nodeDeviceId: string): Promise<PricingPolicyPackage | null> {
  return requestOptional(
    `/provisioning/master-data/pricing_policy?node_device_id=${encodeURIComponent(nodeDeviceId)}`,
    pricingPolicyPackageSchema,
  );
}

export function getMasterDataPackage(streamName: string, nodeDeviceId: string): Promise<MasterDataPackage | null> {
  return requestOptional(
    `/provisioning/master-data/${encodeURIComponent(streamName)}?node_device_id=${encodeURIComponent(nodeDeviceId)}`,
    masterDataPackageSchema,
  );
}

export function putPricingPolicyPackage(payload: Payload): Promise<PricingPolicyPackage> {
  return put('/provisioning/master-data/pricing_policy', pricingPolicyPackageSchema, payload);
}

export function listHalls(restaurantId: string): Promise<Hall[]> {
  return request(`/master-data/floor/halls?${query(restaurantId)}`, z.array(hallSchema));
}

export function createHall(payload: Payload): Promise<Hall> {
  return post('/master-data/floor/halls', hallSchema, payload);
}

export function updateHall(id: string, payload: Payload): Promise<Hall> {
  return patch(`/master-data/floor/halls/${encodeURIComponent(id)}`, hallSchema, payload);
}

export function archiveHall(id: string): Promise<Hall> {
  return post(`/master-data/floor/halls/${encodeURIComponent(id)}/archive`, hallSchema, {});
}

export function listTables(restaurantId: string): Promise<RestaurantTable[]> {
  return request(`/master-data/floor/tables?${query(restaurantId)}`, z.array(tableSchema));
}

export function listRestaurantSections(restaurantId: string): Promise<RestaurantSection[]> {
  return request(`/master-data/restaurant-sections?${query(restaurantId)}`, z.array(restaurantSectionSchema));
}

export function createTable(payload: Payload): Promise<RestaurantTable> {
  return post('/master-data/floor/tables', tableSchema, payload);
}

export function updateTable(id: string, payload: Payload): Promise<RestaurantTable> {
  return patch(`/master-data/floor/tables/${encodeURIComponent(id)}`, tableSchema, payload);
}

export function archiveTable(id: string): Promise<RestaurantTable> {
  return post(`/master-data/floor/tables/${encodeURIComponent(id)}/archive`, tableSchema, {});
}

export function listUnassignedDevices(): Promise<UnassignedEdgeNode[]> {
  return request('/devices/unassigned', z.array(unassignedEdgeNodeSchema));
}

export function listRestaurantDevices(restaurantId: string): Promise<RestaurantEdgeNode[]> {
  return request(`/restaurants/${encodeURIComponent(restaurantId)}/devices`, z.array(restaurantEdgeNodeSchema));
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

export function listEdgeEvents(restaurantId: string, limit = 50, deviceId = ''): Promise<EdgeEvent[]> {
  const params = new URLSearchParams();
  params.set('restaurant_id', restaurantId);
  params.set('limit', String(limit));
  if (deviceId.trim()) {
    params.set('device_id', deviceId.trim());
  }
  return request(`/sync/edge-events?${params.toString()}`, z.array(edgeEventSchema));
}

export function getPublicationState(restaurantId: string): Promise<PublicationSummary | null> {
  return requestOptional(
    `/restaurants/${encodeURIComponent(restaurantId)}/master-data/publication-state`,
    publicationSummarySchema,
  );
}

export function listDeliveryStatuses(restaurantId: string) {
  return request(
    `/restaurants/${encodeURIComponent(restaurantId)}/master-data/delivery-status`,
    z.array(deliveryStatusSchema),
  );
}

export function listReceiptTemplates(filter?: { restaurant_id?: string; document_type?: string }): Promise<ReceiptTemplate[]> {
  const params = new URLSearchParams();
  if (filter?.restaurant_id) params.set('restaurant_id', filter.restaurant_id);
  if (filter?.document_type) params.set('document_type', filter.document_type);
  const qs = params.toString();
  return request(`/receipt-templates${qs ? `?${qs}` : ''}`, z.array(receiptTemplateSchema));
}

export function createReceiptTemplate(payload: Payload): Promise<ReceiptTemplate> {
  return post('/receipt-templates', receiptTemplateSchema, payload);
}

export function updateReceiptTemplate(id: string, payload: Payload): Promise<ReceiptTemplate> {
  return patch(`/receipt-templates/${encodeURIComponent(id)}`, receiptTemplateSchema, payload);
}

export function deactivateReceiptTemplate(id: string): Promise<ReceiptTemplate> {
  return post(`/receipt-templates/${encodeURIComponent(id)}/deactivate`, receiptTemplateSchema, {});
}

export async function previewReceiptTemplate(payload: {
  content: string;
  document_type: string;
  cpl: number;
}): Promise<string> {
  const res = await fetch('/api/v1/receipts/preview', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(`Preview failed: ${res.status}`);
  return res.text();
}

export function listPrinters(restaurantId: string): Promise<Printer[]> {
  return request(`/printers?${query(restaurantId)}`, z.array(printerSchema));
}

export function createPrinter(payload: Payload): Promise<Printer> {
  return post('/printers', printerSchema, payload);
}

export function updatePrinter(id: string, payload: Payload): Promise<Printer> {
  return patch(`/printers/${encodeURIComponent(id)}`, printerSchema, payload);
}

export function deactivatePrinter(id: string): Promise<Printer> {
  return request(`/printers/${encodeURIComponent(id)}`, printerSchema, { method: 'DELETE' });
}
