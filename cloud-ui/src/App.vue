<template>
  <cloud-shell :ctx="cloudCtx">
    <header class="cloud-main-head">
      <div>
        <p class="eyebrow">{{ t(activeGroupLabelKey) }}</p>
        <h2>{{ t(activeTitleKey) }}</h2>
        <p class="cloud-copy">{{ t(activeDescriptionKey) }}</p>
      </div>
      <q-btn
        v-if="canCreateActive"
        color="primary"
        unelevated
        icon="add"
        :label="t('actions.add')"
        @click="startCreate"
      />
    </header>

    <q-banner v-if="successKey" class="success-banner dense-banner">{{ t(successKey) }}</q-banner>
    <launch-readiness-panel v-if="activeKey === 'launchPlan'" :ctx="cloudCtx" />
    <edge-device-panel v-else-if="activeKey === 'edgeDevices'" :ctx="cloudCtx" />
    <edge-events-panel v-else-if="activeKey === 'edgeEvents'" :ctx="cloudCtx" />
    <section v-else-if="activeKey !== 'restaurants' && !selectedRestaurantId" class="empty-state wide">
      {{ t('cloud.empty.selectRestaurant') }}
    </section>
    <publication-panel v-else-if="activeKey === 'publications'" :ctx="cloudCtx" />
    <resource-workspace v-else :ctx="cloudCtx" />
  </cloud-shell>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import CloudShell from './components/cloud/CloudShell.vue';
import EdgeDevicePanel from './components/cloud/EdgeDevicePanel.vue';
import EdgeEventsPanel from './components/cloud/EdgeEventsPanel.vue';
import LaunchReadinessPanel from './components/cloud/LaunchReadinessPanel.vue';
import PublicationPanel from './components/cloud/PublicationPanel.vue';
import ResourceWorkspace from './components/cloud/ResourceWorkspace.vue';
import {
  activateEmployee,
  archiveCatalogFolder,
  archiveCatalogItem,
  archiveEmployee,
  archiveHall,
  archiveMenuItem,
  archiveRestaurant,
  archiveRole,
  archiveTable,
  assignCatalogItemTag,
  assignDeviceToRestaurant,
  assignEmployeeRole,
  createCatalogFolder,
  createCatalogItem,
  createCatalogTag,
  createCategory,
  createEmployee,
  createFolderParameter,
  createHall,
  createMenuItem,
  createModifierBinding,
  createModifierGroup,
  createModifierOption,
  createPricingPolicy,
  createRecipeItem,
  createRestaurant,
  createRole,
  createTable,
  deactivateStopListEntry,
  generatePairingCode,
  getAssignmentStatus,
  getPublicationState,
  listCatalogFolders,
  listCatalogItems,
  listCatalogTags,
  listEmployees,
  listEdgeEvents,
  listFolderParameters,
  listHalls,
  listMenuItems,
  listModifierBindings,
  listModifierGroups,
  listModifierOptions,
  listPricingPolicies,
  listRecipeItems,
  listRestaurants,
  listRoles,
  listStopListEntries,
  listTables,
  listUnassignedDevices,
  publishMasterData,
  rotateEmployeePIN,
  suspendEmployee,
  updateCatalogFolder,
  updateCatalogItem,
  updateCatalogTag,
  updateEmployee,
  updateFolderParameter,
  updateHall,
  updateMenuItem,
  updateModifierBinding,
  updateModifierGroup,
  updateModifierOption,
  updatePricingPolicy,
  updateRecipeItem,
  updateRestaurant,
  updateRole,
  updateStopListEntry,
  updateTable,
  upsertStopListEntry,
  ApiError,
} from './shared/api';
import type { AssignmentStatus, EdgeEvent, PairingCodeResult, PublicationSummary, Restaurant, UnassignedEdgeNode } from './shared/schemas';

type ScenarioKey = 'launchPlan' | 'edgeDevices' | 'edgeEvents';
type ResourceKey =
  | ScenarioKey
  | 'restaurants'
  | 'roles'
  | 'employees'
  | 'catalogItems'
  | 'catalogFolders'
  | 'folderParameters'
  | 'catalogTags'
  | 'itemTags'
  | 'modifierGroups'
  | 'modifierOptions'
  | 'modifierBindings'
  | 'pricingPolicies'
  | 'recipeItems'
  | 'stopLists'
  | 'halls'
  | 'tables'
  | 'menuItems'
  | 'categories'
  | 'publications';

type MasterResourceKey = Exclude<ResourceKey, ScenarioKey | 'publications'>;
type ScopedResourceKey = Exclude<MasterResourceKey, 'restaurants' | 'itemTags' | 'categories'>;
type FormMode = 'create' | 'edit';
type RowValue = string | number | boolean | string[] | null | undefined;
type Row = Record<string, RowValue>;
type SelectOption = { label: string; value: string | number | boolean };

const autoPublishResourceKeys = new Set<ResourceKey>([
  'restaurants',
  'roles',
  'employees',
  'catalogItems',
  'catalogFolders',
  'folderParameters',
  'catalogTags',
  'modifierGroups',
  'modifierOptions',
  'modifierBindings',
  'pricingPolicies',
  'recipeItems',
  'stopLists',
  'halls',
  'tables',
  'menuItems',
]);

type FieldConfig = {
  key: string;
  labelKey: string;
  type?: 'text' | 'number' | 'textarea' | 'checkbox' | 'permissionMatrix';
  rows?: number;
  options?: string;
  createOnly?: boolean;
  updateOnly?: boolean;
  defaultValue?: RowValue;
};

type PermissionGroup = {
  key: string;
  labelKey: string;
  permissions: string[];
};

type RolePreset = {
  key: string;
  labelKey: string;
  permissions: string[];
};

type ResourceConfig = {
  key: MasterResourceKey;
  groupKey: string;
  titleKey: string;
  descriptionKey: string;
  columns: { key: string; labelKey: string }[];
  fields: FieldConfig[];
  commandOnly?: boolean;
  create?: (payload: Row) => Promise<unknown>;
  update?: (id: string, payload: Row) => Promise<unknown>;
  archive?: (id: string) => Promise<unknown>;
};

const { t, tm } = useI18n();

const apiBaseLabel = (import.meta.env.VITE_CLOUD_API_BASE ?? 'http://localhost:8090/api/v1').replace(/\/$/, '');
const selectedRestaurantId = ref('');
const activeKey = ref<ResourceKey>('launchPlan');
const selectedRowId = ref('');
const search = ref('');
const mode = ref<FormMode>('create');
const actionPin = ref('');
const errorKey = ref('');
const errorCode = ref('');
const errorCorrelationId = ref('');
const errorDetails = ref<Record<string, string>>({});
const successKey = ref('');
const loading = ref<string[]>([]);
const restaurants = ref<Restaurant[]>([]);
const publication = ref<PublicationSummary | null>(null);
const edgeDevices = ref<UnassignedEdgeNode[]>([]);
const edgeEvents = ref<EdgeEvent[]>([]);
const selectedEdgeNodeId = ref('');
const assignmentResult = ref<{ status: string; snapshot_url: string } | null>(null);
const assignmentStatus = ref<AssignmentStatus | null>(null);
const pairingResult = ref<PairingCodeResult | null>(null);
const publishForm = reactive({ published_by: '', node_device_id: '' });
const pairingForm = reactive({ display_name: '', expires_in_minutes: 30 });
const form = reactive<Row>({});

const permissionGroups: PermissionGroup[] = [
  {
    key: 'shift',
    labelKey: 'cloud.permissions.groups.shift',
    permissions: ['pos.employee_shift.open', 'pos.employee_shift.close', 'pos.employee_shift.view_current', 'pos.employee_shift.recent'],
  },
  {
    key: 'cash',
    labelKey: 'cloud.permissions.groups.cash',
    permissions: ['pos.cash_session.open', 'pos.cash_session.close', 'pos.cash_session.view_current', 'pos.cash_drawer.record_event'],
  },
  {
    key: 'order',
    labelKey: 'cloud.permissions.groups.order',
    permissions: ['pos.catalog.view', 'pos.floor.view', 'pos.menu.view', 'pos.order.create', 'pos.order.view', 'pos.order.add_line', 'pos.order.change_quantity', 'pos.order.void_line', 'pos.order.close'],
  },
  {
    key: 'pricing',
    labelKey: 'cloud.permissions.groups.pricing',
    permissions: ['pos.pricing.view', 'pos.pricing.discount.apply', 'pos.pricing.surcharge.apply'],
  },
  {
    key: 'precheck',
    labelKey: 'cloud.permissions.groups.precheck',
    permissions: ['pos.precheck.issue', 'pos.precheck.view', 'pos.precheck.reprint', 'pos.precheck.cancel.request', 'pos.precheck.cancel'],
  },
  {
    key: 'payment',
    labelKey: 'cloud.permissions.groups.payment',
    permissions: ['pos.payment.cash', 'pos.payment.card.manual', 'pos.payment.other', 'pos.payment.refund', 'pos.check.view', 'pos.check.reprint'],
  },
  {
    key: 'sync',
    labelKey: 'cloud.permissions.groups.sync',
    permissions: ['pos.sync.view', 'pos.sync.retry_failed'],
  },
];

const rolePresets: RolePreset[] = [
  {
    key: 'cashier',
    labelKey: 'cloud.rolePresets.cashier',
    permissions: [
      'pos.employee_shift.open',
      'pos.employee_shift.close',
      'pos.employee_shift.view_current',
      'pos.employee_shift.recent',
      'pos.cash_session.open',
      'pos.cash_session.view_current',
      'pos.catalog.view',
      'pos.floor.view',
      'pos.menu.view',
      'pos.order.create',
      'pos.order.view',
      'pos.order.add_line',
      'pos.order.change_quantity',
      'pos.order.void_line',
      'pos.order.close',
      'pos.pricing.view',
      'pos.pricing.discount.apply',
      'pos.pricing.surcharge.apply',
      'pos.precheck.issue',
      'pos.precheck.view',
      'pos.precheck.reprint',
      'pos.payment.cash',
      'pos.payment.card.manual',
      'pos.check.view',
    ],
  },
  {
    key: 'senior_cashier',
    labelKey: 'cloud.rolePresets.senior_cashier',
    permissions: [
      'pos.employee_shift.open',
      'pos.employee_shift.close',
      'pos.employee_shift.view_current',
      'pos.employee_shift.recent',
      'pos.cash_session.open',
      'pos.cash_session.close',
      'pos.cash_session.view_current',
      'pos.catalog.view',
      'pos.floor.view',
      'pos.menu.view',
      'pos.order.create',
      'pos.order.view',
      'pos.order.add_line',
      'pos.order.change_quantity',
      'pos.order.void_line',
      'pos.order.close',
      'pos.pricing.view',
      'pos.pricing.discount.apply',
      'pos.pricing.surcharge.apply',
      'pos.precheck.issue',
      'pos.precheck.view',
      'pos.precheck.reprint',
      'pos.precheck.cancel.request',
      'pos.payment.cash',
      'pos.payment.card.manual',
      'pos.payment.refund',
      'pos.check.view',
      'pos.sync.view',
    ],
  },
  {
    key: 'waiter',
    labelKey: 'cloud.rolePresets.waiter',
    permissions: [
      'pos.employee_shift.open',
      'pos.employee_shift.close',
      'pos.employee_shift.view_current',
      'pos.employee_shift.recent',
      'pos.catalog.view',
      'pos.floor.view',
      'pos.menu.view',
      'pos.order.create',
      'pos.order.view',
      'pos.order.add_line',
      'pos.order.change_quantity',
      'pos.order.void_line',
      'pos.order.close',
      'pos.pricing.view',
      'pos.precheck.issue',
      'pos.precheck.view',
      'pos.precheck.reprint',
      'pos.check.view',
    ],
  },
  {
    key: 'manager',
    labelKey: 'cloud.rolePresets.manager',
    permissions: allPermissions(),
  },
  {
    key: 'kitchen',
    labelKey: 'cloud.rolePresets.kitchen',
    permissions: [],
  },
  {
    key: 'support_admin',
    labelKey: 'cloud.rolePresets.support_admin',
    permissions: ['pos.sync.view', 'pos.sync.retry_failed'],
  },
];

const scopedRows = reactive<Record<ScopedResourceKey, Row[]>>({
  roles: [],
  employees: [],
  catalogItems: [],
  catalogFolders: [],
  folderParameters: [],
  catalogTags: [],
  modifierGroups: [],
  modifierOptions: [],
  modifierBindings: [],
  pricingPolicies: [],
  recipeItems: [],
  stopLists: [],
  halls: [],
  tables: [],
  menuItems: [],
});

const lifecycleField: FieldConfig = field('status', { options: 'lifecycleStatuses', updateOnly: true });

const resourceConfigs: ResourceConfig[] = [
  {
    key: 'restaurants',
    groupKey: 'cloud.groups.organization',
    titleKey: 'cloud.resources.restaurants',
    descriptionKey: 'cloud.descriptions.restaurants',
    columns: columns(['name', 'timezone', 'currency', 'business_day_mode', 'status', 'cloud_version']),
    fields: [
      field('name'),
      field('timezone', { defaultValue: 'Europe/Moscow' }),
      field('currency', { defaultValue: 'RUB' }),
      field('business_day_mode', { options: 'businessDayModes', defaultValue: 'standard' }),
      field('business_day_boundary_local_time', { defaultValue: '04:00' }),
      field('status', { options: 'restaurantStatuses', updateOnly: true }),
    ],
    create: createRestaurant,
    update: updateRestaurant,
    archive: archiveRestaurant,
  },
  {
    key: 'roles',
    groupKey: 'cloud.groups.staff',
    titleKey: 'cloud.resources.roles',
    descriptionKey: 'cloud.descriptions.roles',
    columns: columns(['name', 'active', 'cloud_version', 'updated_at']),
    fields: [
      field('name'),
      field('role_profile', { options: 'rolePresets', defaultValue: 'cashier' }),
      field('permission_ids', { type: 'permissionMatrix', defaultValue: [] }),
      field('active', { type: 'checkbox', updateOnly: true }),
    ],
    create: createRole,
    update: updateRole,
    archive: archiveRole,
  },
  {
    key: 'employees',
    groupKey: 'cloud.groups.staff',
    titleKey: 'cloud.resources.employees',
    descriptionKey: 'cloud.descriptions.employees',
    columns: columns(['name', 'role_id', 'status', 'pin_configured', 'pin_credential_version']),
    fields: [field('role_id', { options: 'roles' }), field('name'), field('pin', { createOnly: true }), field('status', { options: 'employeeStatuses', updateOnly: true })],
    create: createEmployee,
    update: updateEmployee,
  },
  {
    key: 'catalogItems',
    groupKey: 'cloud.groups.catalog',
    titleKey: 'cloud.resources.catalogItems',
    descriptionKey: 'cloud.descriptions.catalogItems',
    columns: columns(['name', 'kind', 'sku', 'base_unit', 'folder_id', 'status']),
    fields: [
      field('kind', { options: 'catalogKinds', defaultValue: 'dish' }),
      field('folder_id', { options: 'folders' }),
      field('name'),
      field('sku'),
      field('base_unit', { defaultValue: 'portion' }),
      field('kitchen_type'),
      field('accounting_category'),
      lifecycleField,
    ],
    create: createCatalogItem,
    update: updateCatalogItem,
    archive: archiveCatalogItem,
  },
  {
    key: 'catalogFolders',
    groupKey: 'cloud.groups.catalog',
    titleKey: 'cloud.resources.catalogFolders',
    descriptionKey: 'cloud.descriptions.catalogFolders',
    columns: columns(['name', 'parent_id', 'sort_order', 'status']),
    fields: [field('parent_id', { options: 'folders' }), field('name'), field('sort_order', { type: 'number', defaultValue: 0 }), lifecycleField],
    create: createCatalogFolder,
    update: updateCatalogFolder,
    archive: archiveCatalogFolder,
  },
  {
    key: 'folderParameters',
    groupKey: 'cloud.groups.catalog',
    titleKey: 'cloud.resources.folderParameters',
    descriptionKey: 'cloud.descriptions.folderParameters',
    columns: columns(['parameter_key', 'folder_id', 'value_type', 'status']),
    fields: [field('folder_id', { options: 'folders', createOnly: true }), field('parameter_key', { createOnly: true }), field('value_type', { defaultValue: 'json' }), field('value_json', { type: 'textarea', rows: 4, defaultValue: '{}' }), lifecycleField],
    create: createFolderParameter,
    update: updateFolderParameter,
  },
  {
    key: 'catalogTags',
    groupKey: 'cloud.groups.catalog',
    titleKey: 'cloud.resources.catalogTags',
    descriptionKey: 'cloud.descriptions.catalogTags',
    columns: columns(['name', 'code', 'status', 'cloud_version']),
    fields: [field('name'), field('code'), lifecycleField],
    create: createCatalogTag,
    update: updateCatalogTag,
  },
  {
    key: 'itemTags',
    groupKey: 'cloud.groups.catalog',
    titleKey: 'cloud.resources.itemTags',
    descriptionKey: 'cloud.descriptions.itemTags',
    columns: [],
    fields: [field('catalog_item_id', { options: 'catalogItems' }), field('tag_id', { options: 'tags' })],
    commandOnly: true,
    create: assignCatalogItemTag,
  },
  {
    key: 'modifierGroups',
    groupKey: 'cloud.groups.modifiers',
    titleKey: 'cloud.resources.modifierGroups',
    descriptionKey: 'cloud.descriptions.modifierGroups',
    columns: columns(['name', 'required', 'min_count', 'max_count', 'status']),
    fields: [field('name'), field('required', { type: 'checkbox' }), field('min_count', { type: 'number', defaultValue: 0 }), field('max_count', { type: 'number', defaultValue: 0 }), lifecycleField],
    create: createModifierGroup,
    update: updateModifierGroup,
  },
  {
    key: 'modifierOptions',
    groupKey: 'cloud.groups.modifiers',
    titleKey: 'cloud.resources.modifierOptions',
    descriptionKey: 'cloud.descriptions.modifierOptions',
    columns: columns(['name', 'modifier_group_id', 'price_minor', 'status']),
    fields: [field('modifier_group_id', { options: 'modifierGroups' }), field('name'), field('price_minor', { type: 'number', defaultValue: 0 }), lifecycleField],
    create: createModifierOption,
    update: updateModifierOption,
  },
  {
    key: 'modifierBindings',
    groupKey: 'cloud.groups.modifiers',
    titleKey: 'cloud.resources.modifierBindings',
    descriptionKey: 'cloud.descriptions.modifierBindings',
    columns: columns(['modifier_group_id', 'target_type', 'target_id', 'sort_order', 'status']),
    fields: [field('modifier_group_id', { options: 'modifierGroups' }), field('target_type', { options: 'modifierTargetTypes', defaultValue: 'menu_item' }), field('target_id', { options: 'targetIds' }), field('sort_order', { type: 'number', defaultValue: 0 }), lifecycleField],
    create: createModifierBinding,
    update: updateModifierBinding,
  },
  {
    key: 'pricingPolicies',
    groupKey: 'cloud.groups.pricing',
    titleKey: 'cloud.resources.pricingPolicies',
    descriptionKey: 'cloud.descriptions.pricingPolicies',
    columns: columns(['name', 'kind', 'amount_kind', 'amount_minor', 'value_basis_points', 'status']),
    fields: [field('name'), field('kind', { options: 'pricingKinds', defaultValue: 'discount' }), field('scope'), field('amount_kind', { options: 'amountKinds', defaultValue: 'fixed' }), field('amount_minor', { type: 'number', defaultValue: 0 }), field('value_basis_points', { type: 'number', defaultValue: 0 }), field('application_index', { type: 'number', defaultValue: 1 }), field('manual', { type: 'checkbox' }), field('requires_permission', { options: 'permissions' }), lifecycleField],
    create: createPricingPolicy,
    update: updatePricingPolicy,
  },
  {
    key: 'recipeItems',
    groupKey: 'cloud.groups.inventory',
    titleKey: 'cloud.resources.recipeItems',
    descriptionKey: 'cloud.descriptions.recipeItems',
    columns: columns(['recipe_owner_catalog_item_id', 'component_catalog_item_id', 'quantity', 'unit', 'loss_percent']),
    fields: [
      field('recipe_owner_catalog_item_id', { options: 'catalogItems' }),
      field('component_catalog_item_id', { options: 'catalogItems' }),
      field('quantity', { type: 'number', defaultValue: 1 }),
      field('unit', { defaultValue: 'g' }),
      field('loss_percent', { type: 'number', defaultValue: 0 }),
    ],
    create: createRecipeItem,
    update: updateRecipeItem,
  },
  {
    key: 'stopLists',
    groupKey: 'cloud.groups.inventory',
    titleKey: 'cloud.resources.stopLists',
    descriptionKey: 'cloud.descriptions.stopLists',
    columns: columns(['catalog_item_id', 'available_quantity', 'active', 'source', 'reason', 'updated_at']),
    fields: [
      field('catalog_item_id', { options: 'catalogItems', createOnly: true }),
      field('available_quantity', { type: 'number', defaultValue: 0 }),
      field('reason'),
      field('active', { type: 'checkbox', defaultValue: true }),
    ],
    create: upsertStopListEntry,
    update: updateStopListEntry,
    archive: deactivateStopListEntry,
  },
  {
    key: 'halls',
    groupKey: 'cloud.groups.floor',
    titleKey: 'cloud.resources.halls',
    descriptionKey: 'cloud.descriptions.halls',
    columns: columns(['name', 'status', 'cloud_version']),
    fields: [field('name'), lifecycleField],
    create: createHall,
    update: updateHall,
    archive: archiveHall,
  },
  {
    key: 'tables',
    groupKey: 'cloud.groups.floor',
    titleKey: 'cloud.resources.tables',
    descriptionKey: 'cloud.descriptions.tables',
    columns: columns(['name', 'hall_id', 'seats', 'status']),
    fields: [field('hall_id', { options: 'halls' }), field('name'), field('seats', { type: 'number', defaultValue: 0 }), lifecycleField],
    create: createTable,
    update: updateTable,
    archive: archiveTable,
  },
  {
    key: 'menuItems',
    groupKey: 'cloud.groups.menu',
    titleKey: 'cloud.resources.menuItems',
    descriptionKey: 'cloud.descriptions.menuItems',
    columns: columns(['name', 'catalog_item_id', 'price', 'currency', 'status']),
    fields: [field('catalog_item_id', { options: 'catalogItems' }), field('name'), field('price', { type: 'number', defaultValue: 0 }), field('currency'), field('availability_json', { type: 'textarea', rows: 4, defaultValue: '{}' }), field('station_routing_key'), lifecycleField],
    create: createMenuItem,
    update: updateMenuItem,
    archive: archiveMenuItem,
  },
  {
    key: 'categories',
    groupKey: 'cloud.groups.menu',
    titleKey: 'cloud.resources.categories',
    descriptionKey: 'cloud.descriptions.categories',
    columns: [],
    fields: [field('name'), field('sort_order', { type: 'number', defaultValue: 0 })],
    commandOnly: true,
    create: createCategory,
  },
];

const publicationNav = {
  key: 'publications' as ResourceKey,
  groupKey: 'cloud.groups.publication',
  titleKey: 'cloud.resources.publications',
  descriptionKey: 'cloud.descriptions.publications',
};

const scenarioNav = [
  { key: 'launchPlan' as ResourceKey, groupKey: 'cloud.groups.scenarios', titleKey: 'cloud.resources.launchPlan', descriptionKey: 'cloud.descriptions.launchPlan' },
  { key: 'edgeDevices' as ResourceKey, groupKey: 'cloud.groups.scenarios', titleKey: 'cloud.resources.edgeDevices', descriptionKey: 'cloud.descriptions.edgeDevices' },
  { key: 'edgeEvents' as ResourceKey, groupKey: 'cloud.groups.scenarios', titleKey: 'cloud.resources.edgeEvents', descriptionKey: 'cloud.descriptions.edgeEvents' },
];

const navGroups = computed(() => {
  const groupKeys = ['cloud.groups.scenarios', 'cloud.groups.organization', 'cloud.groups.staff', 'cloud.groups.catalog', 'cloud.groups.modifiers', 'cloud.groups.pricing', 'cloud.groups.inventory', 'cloud.groups.floor', 'cloud.groups.menu', 'cloud.groups.publication'];
  return groupKeys.map((labelKey) => ({
    key: labelKey,
    labelKey,
    items: [
      ...(labelKey === 'cloud.groups.scenarios' ? scenarioNav : []),
      ...resourceConfigs.filter((item) => item.groupKey === labelKey),
      ...(labelKey === publicationNav.groupKey ? [publicationNav] : []),
    ],
  }));
});

const activeConfig = computed(() => resourceConfigs.find((item) => item.key === activeKey.value));
const activeScenario = computed(() => scenarioNav.find((item) => item.key === activeKey.value));
const activeTitleKey = computed(() => activeConfig.value?.titleKey ?? activeScenario.value?.titleKey ?? publicationNav.titleKey);
const activeGroupLabelKey = computed(() => activeConfig.value?.groupKey ?? activeScenario.value?.groupKey ?? publicationNav.groupKey);
const activeDescriptionKey = computed(() => activeConfig.value?.descriptionKey ?? activeScenario.value?.descriptionKey ?? publicationNav.descriptionKey);
const activeColumns = computed(() => activeConfig.value?.columns ?? []);
const visibleFields = computed(() => (activeConfig.value?.fields ?? []).filter((item) => !(mode.value === 'create' && item.updateOnly) && !(mode.value === 'edit' && item.createOnly)));
const permissionFields = computed(() => visibleFields.value.filter((item) => item.type === 'permissionMatrix'));
const currentRows = computed<Row[]>(() => {
  if (activeKey.value === 'restaurants') return restaurants.value as unknown as Row[];
  if (activeKey.value === 'launchPlan' || activeKey.value === 'edgeDevices' || activeKey.value === 'edgeEvents' || activeKey.value === 'publications' || activeKey.value === 'itemTags' || activeKey.value === 'categories') return [];
  return scopedRows[activeKey.value as ScopedResourceKey];
});
const filteredRows = computed(() => {
  const needle = search.value.trim().toLowerCase();
  if (!needle) return currentRows.value;
  return currentRows.value.filter((row) => JSON.stringify(row).toLowerCase().includes(needle));
});
const selectedRestaurant = computed(() => restaurants.value.find((item) => item.id === selectedRestaurantId.value));
const restaurantOptions = computed(() => restaurants.value.map((item) => ({ label: item.name, value: item.id })));
const activeLoading = computed(() => isLoading(activeKey.value));
const canCreateActive = computed(() => Boolean(activeConfig.value?.create && (activeKey.value === 'restaurants' || selectedRestaurantId.value)));
const canSubmitActive = computed(() => Boolean(activeConfig.value?.create && mode.value === 'create') || Boolean(activeConfig.value?.update && mode.value === 'edit'));
const publicationCounts = computed(() => Object.entries(publication.value?.counts ?? {}).sort(([a], [b]) => a.localeCompare(b)));
const filteredEdgeDevices = computed(() => {
  const needle = search.value.trim().toLowerCase();
  if (!needle) return edgeDevices.value;
  return edgeDevices.value.filter((node) => JSON.stringify(node).toLowerCase().includes(needle));
});
const unassignedEdgeOptions = computed(() => edgeDevices.value.map((node) => ({ label: nodeDisplayName(node.node_device_id), value: node.node_device_id })));
const knownNodeOptions = computed(() => {
  const ids = new Set<string>();
  for (const node of edgeDevices.value) ids.add(node.node_device_id);
  if (selectedEdgeNodeId.value) ids.add(selectedEdgeNodeId.value);
  if (assignmentResult.value?.status) ids.add(selectedEdgeNodeId.value);
  if (assignmentStatus.value?.node_device_id) ids.add(assignmentStatus.value.node_device_id);
  if (pairingResult.value?.node_device_id) ids.add(pairingResult.value.node_device_id);
  if (publishForm.node_device_id) ids.add(publishForm.node_device_id);
  return [...ids].filter(Boolean).map((id) => ({ label: nodeDisplayName(id), value: id }));
});
const selectedPermissions = computed(() => {
  const value = form.permission_ids;
  return Array.isArray(value) ? value : [];
});
const errorDetailsList = computed(() => Object.entries(errorDetails.value).map(([key, value]) => ({ key, label: readableDetailLabel(key), value })));
const onboardingChecks = computed(() => {
  const roles = scopedRows.roles.length;
  const employees = scopedRows.employees.length;
  const halls = scopedRows.halls.length;
  const tables = scopedRows.tables.length;
  const menuItems = scopedRows.menuItems.length;
  const stopLists = scopedRows.stopLists.length;
  const nodeReady = Boolean(assignmentResult.value || assignmentStatus.value?.status === 'assigned');
  const snapshotReady = Boolean(assignmentResult.value?.snapshot_url || assignmentStatus.value?.snapshot_url || publication.value?.package_sha256);
  return [
    { key: 'restaurant', ready: Boolean(selectedRestaurant.value), titleKey: 'cloud.onboarding.restaurant.title', descriptionKey: 'cloud.onboarding.restaurant.description', params: { count: restaurants.value.length }, target: 'restaurants' as ResourceKey, actionKey: 'cloud.onboarding.actions.open', icon: 'storefront' },
    { key: 'staff', ready: roles > 0 && employees > 0, titleKey: 'cloud.onboarding.staff.title', descriptionKey: 'cloud.onboarding.staff.description', params: { roles, employees }, target: 'roles' as ResourceKey, actionKey: 'cloud.onboarding.actions.configure', icon: 'badge' },
    { key: 'floor', ready: halls > 0 && tables > 0, titleKey: 'cloud.onboarding.floor.title', descriptionKey: 'cloud.onboarding.floor.description', params: { halls, tables }, target: 'halls' as ResourceKey, actionKey: 'cloud.onboarding.actions.configure', icon: 'table_restaurant' },
    { key: 'menu', ready: menuItems > 0, titleKey: 'cloud.onboarding.menu.title', descriptionKey: 'cloud.onboarding.menu.description', params: { menuItems }, target: 'menuItems' as ResourceKey, actionKey: 'cloud.onboarding.actions.configure', icon: 'restaurant_menu' },
    { key: 'inventory', ready: stopLists > 0, titleKey: 'cloud.onboarding.inventory.title', descriptionKey: 'cloud.onboarding.inventory.description', params: { stopLists }, target: 'stopLists' as ResourceKey, actionKey: 'cloud.onboarding.actions.configure', icon: 'block' },
    { key: 'edge', ready: nodeReady, titleKey: 'cloud.onboarding.edge.title', descriptionKey: 'cloud.onboarding.edge.description', params: { count: knownNodeOptions.value.length }, target: 'edgeDevices' as ResourceKey, actionKey: 'cloud.onboarding.actions.connect', icon: 'devices' },
    { key: 'publication', ready: Boolean(publication.value), titleKey: 'cloud.onboarding.publication.title', descriptionKey: 'cloud.onboarding.publication.description', params: { version: publication.value?.version ?? '-' }, target: 'publications' as ResourceKey, actionKey: 'cloud.onboarding.actions.publish', icon: 'publish' },
    { key: 'snapshot', ready: snapshotReady, titleKey: 'cloud.onboarding.snapshot.title', descriptionKey: 'cloud.onboarding.snapshot.description', params: { code: publication.value?.package_sha256 ? shortId(publication.value.package_sha256) : '-' }, target: 'publications' as ResourceKey, actionKey: 'cloud.onboarding.actions.open', icon: 'inventory' },
  ];
});

const launchSteps = [
  { key: 'edge', status: 'current', badgeKey: 'cloud.launchPlan.badges.now', titleKey: 'cloud.launchPlan.steps.edge.title', descriptionKey: 'cloud.launchPlan.steps.edge.description' },
  { key: 'organization', status: 'next', badgeKey: 'cloud.launchPlan.badges.next', titleKey: 'cloud.launchPlan.steps.organization.title', descriptionKey: 'cloud.launchPlan.steps.organization.description' },
  { key: 'menu', status: 'next', badgeKey: 'cloud.launchPlan.badges.next', titleKey: 'cloud.launchPlan.steps.menu.title', descriptionKey: 'cloud.launchPlan.steps.menu.description' },
  { key: 'publish', status: 'next', badgeKey: 'cloud.launchPlan.badges.next', titleKey: 'cloud.launchPlan.steps.publish.title', descriptionKey: 'cloud.launchPlan.steps.publish.description' },
  { key: 'sale', status: 'later', badgeKey: 'cloud.launchPlan.badges.later', titleKey: 'cloud.launchPlan.steps.sale.title', descriptionKey: 'cloud.launchPlan.steps.sale.description' },
];

const playbookItems = [
  { kickerKey: 'cloud.edgeDevices.claimedFlow', titleKey: 'cloud.launchPlan.playbook.claimed.title', descriptionKey: 'cloud.launchPlan.playbook.claimed.description' },
  { kickerKey: 'cloud.edgeDevices.licenseFlow', titleKey: 'cloud.launchPlan.playbook.pairing.title', descriptionKey: 'cloud.launchPlan.playbook.pairing.description' },
  { kickerKey: 'cloud.groups.publication', titleKey: 'cloud.launchPlan.playbook.publish.title', descriptionKey: 'cloud.launchPlan.playbook.publish.description' },
];

const publicationFields = computed(() => {
  if (!publication.value) return [];
  return [
    { key: 'version', labelKey: 'cloud.fields.version', value: publication.value.version },
    { key: 'status', labelKey: 'cloud.fields.status', value: publication.value.status },
    { key: 'cloud_version', labelKey: 'cloud.fields.cloud_version', value: publication.value.cloud_version },
    { key: 'published_at', labelKey: 'cloud.fields.published_at', value: formatDate(publication.value.published_at) },
    { key: 'published_by', labelKey: 'cloud.fields.published_by', value: publication.value.published_by },
    { key: 'package_sha256', labelKey: 'cloud.fields.package_sha256', value: shortId(publication.value.package_sha256) },
  ];
});

const cloudCtx = {
  apiBaseLabel,
  selectedRestaurantId,
  activeKey,
  selectedRowId,
  search,
  mode,
  actionPin,
  errorKey,
  errorCode,
  errorCorrelationId,
  successKey,
  publication,
  edgeEvents,
  selectedEdgeNodeId,
  assignmentResult,
  assignmentStatus,
  pairingResult,
  publishForm,
  pairingForm,
  form,
  permissionGroups,
  navGroups,
  restaurantOptions,
  activeConfig,
  activeColumns,
  visibleFields,
  permissionFields,
  filteredRows,
  activeLoading,
  canSubmitActive,
  publicationCounts,
  filteredEdgeDevices,
  unassignedEdgeOptions,
  knownNodeOptions,
  selectedPermissions,
  errorDetailsList,
  onboardingChecks,
  launchSteps,
  playbookItems,
  publicationFields,
  setActive,
  navCount,
  rowKey,
  selectRow,
  startCreate,
  inputModelValue,
  setFormValue,
  selectOptions,
  isSelectDisabled,
  loadPublication,
  reloadActive,
  submitForm,
  archiveSelected,
  employeeAction,
  assignSelectedEmployeeRole,
  rotateSelectedEmployeePIN,
  selectEdgeNode,
  assignSelectedEdgeDevice,
  loadSelectedAssignmentStatus,
  generateSelectedPairingCode,
  loadEdgeEvents,
  publishSelectedRestaurant,
  isLoading,
  formatCell,
  statusText,
  edgeStatusText,
  formatDate,
  permissionLabel,
  togglePermission,
  nodeDisplayName,
};

watch(selectedRestaurantId, async () => {
  clearMessages();
  if (!selectedRestaurantId.value) return;
  await loadScopedData();
  if (activeKey.value === 'publications' || activeKey.value === 'launchPlan') await loadPublication();
  if (activeKey.value === 'edgeDevices' || activeKey.value === 'launchPlan') await loadEdgeDevices();
  if (activeKey.value === 'edgeEvents') await loadEdgeEvents();
  resetSelection();
});

watch(activeKey, async () => {
  clearMessages();
  search.value = '';
  resetSelection();
  if (activeKey.value === 'publications') await loadPublication();
  if (activeKey.value === 'edgeDevices') await loadEdgeDevices();
  if (activeKey.value === 'edgeEvents') await loadEdgeEvents();
  if (activeKey.value === 'launchPlan') {
    await Promise.all([loadPublication(), loadEdgeDevices()]);
  }
  if (selectedRestaurantId.value && isScopedResourceKey(activeKey.value)) {
    await loadScopedData();
  }
});

onMounted(async () => {
  await loadRestaurants();
});

function field(key: string, overrides: Partial<FieldConfig> = {}): FieldConfig {
  return { key, labelKey: `cloud.fields.${key}`, type: 'text', ...overrides };
}

function columns(keys: string[]) {
  return keys.map((key) => ({ key, labelKey: `cloud.fields.${key}` }));
}

function setActive(key: ResourceKey) {
  activeKey.value = key;
}

function isScopedResourceKey(key: ResourceKey): key is ScopedResourceKey {
  return !['launchPlan', 'edgeDevices', 'edgeEvents', 'restaurants', 'publications', 'itemTags', 'categories'].includes(key);
}

function navCount(key: ResourceKey) {
  if (key === 'launchPlan') return launchSteps.length;
  if (key === 'edgeDevices') return edgeDevices.value.length;
  if (key === 'edgeEvents') return edgeEvents.value.length;
  if (key === 'restaurants') return restaurants.value.length;
  if (key === 'publications') return publication.value ? publication.value.version : '-';
  if (key === 'itemTags' || key === 'categories') return '+';
  return scopedRows[key as ScopedResourceKey]?.length ?? 0;
}

function rowKey(row: Row) {
  return typeof row.id === 'string' ? row.id : JSON.stringify(row);
}

function selectRow(row: Row) {
  selectedRowId.value = rowKey(row);
  mode.value = 'edit';
  resetForm(row);
}

function startCreate() {
  mode.value = 'create';
  selectedRowId.value = '';
  resetForm();
}

function resetSelection() {
  mode.value = 'create';
  selectedRowId.value = '';
  resetForm();
}

function resetForm(source?: Row) {
  for (const key of Object.keys(form)) delete form[key];
  for (const item of activeConfig.value?.fields ?? []) {
    if (mode.value === 'create' && item.updateOnly) continue;
    if (mode.value === 'edit' && item.createOnly) continue;
    form[item.key] = source?.[item.key] ?? defaultFieldValue(item);
  }
  if (activeKey.value === 'roles') {
    const permissions = permissionsFromJSON(typeof source?.permissions_json === 'string' ? source.permissions_json : rolePresetByKey(String(form.role_profile ?? 'cashier'))?.permissions ?? []);
    form.permission_ids = permissions;
    form.role_profile = rolePresetForPermissions(permissions)?.key ?? 'custom';
  }
  if (activeKey.value === 'menuItems' && mode.value === 'create') {
    form.currency = selectedRestaurant.value?.currency ?? 'RUB';
  }
}

function defaultFieldValue(item: FieldConfig): RowValue {
  if (item.defaultValue !== undefined) return item.defaultValue;
  if (item.type === 'number') return 0;
  if (item.type === 'checkbox') return false;
  return '';
}

function inputModelValue(key: string): string | number | null | undefined {
  const value = form[key];
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  if (Array.isArray(value)) return value.join(', ');
  return value;
}

function setFormValue(key: string, value: RowValue) {
  form[key] = value;
  if (key === 'role_profile') {
    const preset = rolePresetByKey(String(value ?? ''));
    if (preset) form.permission_ids = [...preset.permissions].sort();
  }
}

function selectOptions(item: FieldConfig): SelectOption[] {
  switch (item.options) {
    case 'businessDayModes':
      return enumOptions(['standard', '24_7'], 'cloud.businessDayModes');
    case 'restaurantStatuses':
      return enumOptions(['active', 'archived'], 'cloud.statuses');
    case 'employeeStatuses':
      return enumOptions(['active', 'suspended', 'archived'], 'cloud.statuses');
    case 'lifecycleStatuses':
      return enumOptions(['draft', 'published', 'archived'], 'cloud.statuses');
    case 'catalogKinds':
      return enumOptions(['dish', 'good', 'semi_finished', 'service'], 'cloud.catalogKinds');
    case 'modifierTargetTypes':
      return enumOptions(['menu_item', 'catalog_item', 'folder', 'tag'], 'cloud.modifierTargets');
    case 'pricingKinds':
      return enumOptions(['discount', 'surcharge'], 'cloud.pricingKinds');
    case 'amountKinds':
      return enumOptions(['fixed', 'percentage'], 'cloud.amountKinds');
    case 'rolePresets':
      return [...rolePresets.map((preset) => ({ label: t(preset.labelKey), value: preset.key })), { label: t('cloud.rolePresets.custom'), value: 'custom' }];
    case 'permissions':
      return allPermissions().map((permission) => ({ label: permissionLabel(permission), value: permission }));
    case 'roles':
      return rowOptions(scopedRows.roles);
    case 'folders':
      return rowOptions(scopedRows.catalogFolders);
    case 'catalogItems':
      return rowOptions(scopedRows.catalogItems);
    case 'tags':
      return rowOptions(scopedRows.catalogTags);
    case 'modifierGroups':
      return rowOptions(scopedRows.modifierGroups);
    case 'halls':
      return rowOptions(scopedRows.halls);
    case 'targetIds':
      return targetOptions(String(form.target_type ?? ''));
    default:
      return [];
  }
}

function enumOptions(values: string[], prefix: string) {
  return values.map((value) => ({ label: t(`${prefix}.${value}`), value }));
}

function rowOptions(rows: Row[]) {
  return rows.map((item) => ({
    label: `${String(item.name ?? item.code ?? item.id)} (${shortId(String(item.id ?? ''))})`,
    value: String(item.id ?? ''),
  }));
}

function targetOptions(targetType: string) {
  if (targetType === 'menu_item') return rowOptions(scopedRows.menuItems);
  if (targetType === 'catalog_item') return rowOptions(scopedRows.catalogItems);
  if (targetType === 'folder') return rowOptions(scopedRows.catalogFolders);
  if (targetType === 'tag') return rowOptions(scopedRows.catalogTags);
  return [];
}

async function loadRestaurants() {
  await withLoading('restaurants', async () => {
    restaurants.value = await listRestaurants();
    if (!selectedRestaurantId.value && restaurants.value.length > 0) selectedRestaurantId.value = restaurants.value[0].id;
  });
}

async function loadScopedData() {
  if (!selectedRestaurantId.value) return;
  await Promise.all((Object.keys(scopedRows) as ScopedResourceKey[]).map((key) => loadScopedResource(key)));
}

async function loadScopedResource(key: ScopedResourceKey) {
  if (!selectedRestaurantId.value) return;
  await withLoading(key, async () => {
    const restaurantId = selectedRestaurantId.value;
    if (key === 'roles') scopedRows.roles = (await listRoles(restaurantId)) as unknown as Row[];
    if (key === 'employees') scopedRows.employees = (await listEmployees(restaurantId)) as unknown as Row[];
    if (key === 'catalogItems') scopedRows.catalogItems = (await listCatalogItems(restaurantId)) as unknown as Row[];
    if (key === 'catalogFolders') scopedRows.catalogFolders = (await listCatalogFolders(restaurantId)) as unknown as Row[];
    if (key === 'folderParameters') scopedRows.folderParameters = (await listFolderParameters(restaurantId)) as unknown as Row[];
    if (key === 'catalogTags') scopedRows.catalogTags = (await listCatalogTags(restaurantId)) as unknown as Row[];
    if (key === 'modifierGroups') scopedRows.modifierGroups = (await listModifierGroups(restaurantId)) as unknown as Row[];
    if (key === 'modifierOptions') scopedRows.modifierOptions = (await listModifierOptions(restaurantId)) as unknown as Row[];
    if (key === 'modifierBindings') scopedRows.modifierBindings = (await listModifierBindings(restaurantId)) as unknown as Row[];
    if (key === 'pricingPolicies') scopedRows.pricingPolicies = (await listPricingPolicies(restaurantId)) as unknown as Row[];
    if (key === 'recipeItems') scopedRows.recipeItems = (await listRecipeItems(restaurantId)) as unknown as Row[];
    if (key === 'stopLists') scopedRows.stopLists = (await listStopListEntries(restaurantId)) as unknown as Row[];
    if (key === 'halls') scopedRows.halls = (await listHalls(restaurantId)) as unknown as Row[];
    if (key === 'tables') scopedRows.tables = (await listTables(restaurantId)) as unknown as Row[];
    if (key === 'menuItems') scopedRows.menuItems = (await listMenuItems(restaurantId)) as unknown as Row[];
  });
}

async function loadPublication() {
  if (!selectedRestaurantId.value) return;
  await withLoading('publication', async () => {
    publication.value = await getPublicationState(selectedRestaurantId.value);
  });
}

async function reloadActive() {
  if (activeKey.value === 'launchPlan') return;
  if (activeKey.value === 'edgeDevices') return loadEdgeDevices();
  if (activeKey.value === 'edgeEvents') return loadEdgeEvents();
  if (activeKey.value === 'restaurants') return loadRestaurants();
  if (activeKey.value === 'publications') return loadPublication();
  if (activeKey.value === 'itemTags' || activeKey.value === 'categories') return;
  return loadScopedResource(activeKey.value as ScopedResourceKey);
}

async function submitForm() {
  const config = activeConfig.value;
  if (!config?.create) return;
  const validationErrorKey = validateForm(config);
  if (validationErrorKey) {
    showLocalError(validationErrorKey, 'CLIENT_VALIDATION_FAILED');
    return;
  }
  await withLoading('submit', async () => {
    const payload = formPayload(config);
    let nextSuccessKey = 'cloud.messages.saved';
    if (mode.value === 'create') {
      const created = await config.create?.(payload);
      if (config.key === 'restaurants' && created && typeof (created as Row).id === 'string') {
        selectedRestaurantId.value = String((created as Row).id);
      }
      nextSuccessKey = 'cloud.messages.created';
    } else if (selectedRowId.value && config.update) {
      await config.update(selectedRowId.value, payload);
    }
    await reloadAfterMutation(config.key);
    await autoPublishAfterMutation(config.key);
    successKey.value = nextSuccessKey;
    startCreate();
  });
}

function validateForm(config: ResourceConfig) {
  for (const item of config.fields) {
    if (item.key === 'value_json' || item.key === 'availability_json') {
      const value = String(form[item.key] ?? '').trim();
      if (value && !isValidJSON(value)) return 'errors.form.invalidJson';
    }
  }
  if (config.key === 'roles' && !Array.isArray(form.permission_ids)) {
    return 'errors.form.permissionsRequired';
  }
  return '';
}

function isValidJSON(value: string) {
  try {
    JSON.parse(value);
    return true;
  } catch {
    return false;
  }
}

function isSelectDisabled(item: FieldConfig) {
  const options = selectOptions(item);
  if (options.length > 0) return false;
  return ['roles', 'catalogItems', 'tags', 'modifierGroups', 'halls', 'targetIds'].includes(item.options ?? '');
}

function formPayload(config: ResourceConfig) {
  const payload: Row = {};
  if (config.key !== 'restaurants') payload.restaurant_id = selectedRestaurantId.value;
  for (const item of config.fields) {
    if (mode.value === 'create' && item.updateOnly) continue;
    if (mode.value === 'edit' && item.createOnly) continue;
    if (!(item.key in form)) continue;
    if (config.key === 'roles' && (item.key === 'role_profile' || item.key === 'permission_ids')) continue;
    payload[item.key] = normalizedValue(item, form[item.key]);
  }
  if (config.key === 'roles') {
    payload.permissions_json = permissionsJSON(selectedPermissions.value);
  }
  return payload;
}

function normalizedValue(item: FieldConfig, value: RowValue): RowValue {
  if (item.type === 'number') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  if (item.type === 'checkbox') return Boolean(value);
  if (Array.isArray(value)) return value;
  return typeof value === 'string' ? value.trim() : value;
}

async function archiveSelected() {
  const config = activeConfig.value;
  if (!config?.archive || !selectedRowId.value) return;
  await withLoading('archive', async () => {
    await config.archive?.(selectedRowId.value);
    await reloadAfterMutation(config.key);
    await autoPublishAfterMutation(config.key);
    successKey.value = 'cloud.messages.archived';
    startCreate();
  });
}

async function employeeAction(action: 'suspend' | 'activate' | 'archive') {
  if (!selectedRowId.value) return;
  await withLoading('employee-action', async () => {
    if (action === 'suspend') await suspendEmployee(selectedRowId.value);
    if (action === 'activate') await activateEmployee(selectedRowId.value);
    if (action === 'archive') await archiveEmployee(selectedRowId.value);
    await loadScopedResource('employees');
    await autoPublishAfterMutation('employees');
    successKey.value = 'cloud.messages.saved';
  });
}

async function assignSelectedEmployeeRole() {
  if (!selectedRowId.value || !form.role_id) return;
  await withLoading('employee-action', async () => {
    await assignEmployeeRole(selectedRowId.value, String(form.role_id));
    await loadScopedResource('employees');
    await autoPublishAfterMutation('employees');
    successKey.value = 'cloud.messages.saved';
  });
}

async function rotateSelectedEmployeePIN() {
  if (!selectedRowId.value || !actionPin.value.trim()) return;
  await withLoading('employee-action', async () => {
    await rotateEmployeePIN(selectedRowId.value, actionPin.value.trim());
    actionPin.value = '';
    await loadScopedResource('employees');
    await autoPublishAfterMutation('employees');
    successKey.value = 'cloud.messages.saved';
  });
}


async function loadEdgeEvents() {
  await withLoading('edge-events', async () => {
    edgeEvents.value = await listEdgeEvents(selectedRestaurantId.value, 50);
  });
}

async function loadEdgeDevices() {
  await withLoading('edge-devices', async () => {
    edgeDevices.value = await listUnassignedDevices();
  });
}

function selectEdgeNode(node: UnassignedEdgeNode) {
  selectedEdgeNodeId.value = node.node_device_id;
  assignmentResult.value = null;
  assignmentStatus.value = null;
}

async function assignSelectedEdgeDevice() {
  if (!selectedRestaurantId.value || !selectedEdgeNodeId.value.trim()) return;
  await withLoading('edge-assign', async () => {
    assignmentResult.value = await assignDeviceToRestaurant(selectedRestaurantId.value, selectedEdgeNodeId.value.trim());
    assignmentStatus.value = null;
    publishForm.node_device_id = selectedEdgeNodeId.value.trim();
    successKey.value = 'cloud.messages.deviceAssigned';
    await loadEdgeDevices();
  });
}

async function loadSelectedAssignmentStatus() {
  if (!selectedEdgeNodeId.value.trim()) return;
  await withLoading('edge-status', async () => {
    assignmentStatus.value = await getAssignmentStatus(selectedEdgeNodeId.value.trim());
  });
}

async function generateSelectedPairingCode() {
  if (!selectedRestaurantId.value) return;
  await withLoading('pairing-code', async () => {
    pairingResult.value = await generatePairingCode(selectedRestaurantId.value, {
      display_name: pairingForm.display_name.trim(),
      expires_in_minutes: Number(pairingForm.expires_in_minutes) || 30,
    });
    selectedEdgeNodeId.value = pairingResult.value.node_device_id;
    publishForm.node_device_id = pairingResult.value.node_device_id;
    successKey.value = 'cloud.messages.pairingGenerated';
  });
}

async function publishSelectedRestaurant() {
  if (!selectedRestaurantId.value || !publishForm.published_by.trim()) return;
  await withLoading('publication-submit', async () => {
    publication.value = await publishMasterData(selectedRestaurantId.value, {
      published_by: publishForm.published_by.trim(),
      node_device_id: publishForm.node_device_id.trim(),
    });
    successKey.value = 'cloud.messages.published';
  });
}

async function autoPublishAfterMutation(key: ResourceKey) {
  if (!selectedRestaurantId.value || !autoPublishResourceKeys.has(key)) return;
  // Cloud UI является операторским контуром пилота: после CRUD сразу создаем новый package,
  // чтобы Edge получил свежий checkpoint в ближайшем exchange без отдельного шага публикации.
  publication.value = await publishMasterData(selectedRestaurantId.value, {
    published_by: 'cloud-ui',
    node_device_id: publishForm.node_device_id.trim(),
  });
}

async function reloadAfterMutation(key: ResourceKey) {
  if (key === 'restaurants') return loadRestaurants();
  if (key === 'itemTags' || key === 'categories') return;
  return loadScopedResource(key as ScopedResourceKey);
}

async function withLoading(key: string, fn: () => Promise<void>) {
  clearMessages();
  loading.value = [...loading.value, key];
  try {
    await fn();
  } catch (error) {
    handleError(error);
  } finally {
    loading.value = loading.value.filter((item) => item !== key);
  }
}

function isLoading(key: string) {
  return loading.value.includes(key);
}

function handleError(error: unknown) {
  if (error instanceof ApiError) {
    errorKey.value = error.messageKey;
    errorCode.value = error.code;
    errorCorrelationId.value = error.correlationId;
    errorDetails.value = error.details;
    return;
  }
  errorKey.value = 'errors.unexpected';
  errorCode.value = '';
  errorCorrelationId.value = '';
  errorDetails.value = {};
}

function showLocalError(messageKey: string, code: string) {
  clearMessages();
  errorKey.value = messageKey;
  errorCode.value = code;
  errorCorrelationId.value = '';
  errorDetails.value = {};
}

function clearMessages() {
  errorKey.value = '';
  errorCode.value = '';
  errorCorrelationId.value = '';
  errorDetails.value = {};
  successKey.value = '';
}

function formatCell(key: string, value: unknown) {
  if (value === null || value === undefined || value === '') return '-';
  if (typeof value === 'boolean') return t(value ? 'common.yes' : 'common.no');
  if (key.endsWith('_at') && typeof value === 'string') return formatDate(value);
  if (typeof value === 'string') {
    const reference = referenceLabel(key, value);
    if (reference) return reference;
  }
  if ((key.endsWith('_id') || key === 'id') && typeof value === 'string') return shortId(value);
  return String(value);
}

function statusText(value: unknown) {
  return t(`cloud.statuses.${String(value)}`);
}

function edgeStatusText(value: unknown) {
  return t(`cloud.edgeStatuses.${String(value)}`);
}

function formatDate(value: string) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' }).format(date);
}

function shortId(value: string) {
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function allPermissions() {
  return [...new Set(permissionGroups.flatMap((group) => group.permissions))].sort();
}

function rolePresetByKey(key: string) {
  return rolePresets.find((preset) => preset.key === key);
}

function rolePresetForPermissions(permissions: string[]) {
  const body = permissionsJSON(permissions);
  return rolePresets.find((preset) => permissionsJSON(preset.permissions) === body);
}

function permissionsFromJSON(value: string | string[]) {
  if (Array.isArray(value)) return [...value].sort();
  try {
    const raw = JSON.parse(value) as Record<string, unknown>;
    const out = new Set<string>();
    for (const [key, allowed] of Object.entries(raw)) {
      if (allowed === true && allPermissions().includes(key)) out.add(key);
    }
    if (Array.isArray(raw.permissions)) {
      for (const permission of raw.permissions) {
        if (typeof permission === 'string' && allPermissions().includes(permission)) out.add(permission);
      }
    }
    return [...out].sort();
  } catch {
    return [];
  }
}

function permissionsJSON(permissions: string[]) {
  const allowed = new Set(allPermissions());
  const keys = [...new Set(permissions.filter((permission) => allowed.has(permission)))].sort();
  const body: Record<string, boolean> = {};
  for (const permission of keys) body[permission] = true;
  return JSON.stringify(body);
}

function permissionLabel(permission: string) {
  const labels = tm('cloud.permissions.items') as Record<string, string>;
  return labels[permission] ?? permission;
}

function togglePermission(permission: string, checked: boolean) {
  const values = new Set(selectedPermissions.value);
  if (checked) values.add(permission);
  else values.delete(permission);
  form.permission_ids = [...values].sort();
  form.role_profile = rolePresetForPermissions(form.permission_ids)?.key ?? 'custom';
}

function referenceLabel(key: string, value: string) {
  if (key === 'role_id') return rowLabel(scopedRows.roles, value);
  if (key === 'folder_id' || key === 'parent_id') return rowLabel(scopedRows.catalogFolders, value);
  if (key === 'catalog_item_id' || key === 'recipe_owner_catalog_item_id' || key === 'component_catalog_item_id') return rowLabel(scopedRows.catalogItems, value);
  if (key === 'tag_id') return rowLabel(scopedRows.catalogTags, value);
  if (key === 'modifier_group_id') return rowLabel(scopedRows.modifierGroups, value);
  if (key === 'hall_id') return rowLabel(scopedRows.halls, value);
  if (key === 'target_id') return targetLabel(value);
  return '';
}

function rowLabel(rows: Row[], id: string) {
  const row = rows.find((item) => item.id === id);
  if (!row) return '';
  return `${String(row.name ?? row.code ?? id)} (${shortId(id)})`;
}

function targetLabel(id: string) {
  return rowLabel(scopedRows.menuItems, id)
    || rowLabel(scopedRows.catalogItems, id)
    || rowLabel(scopedRows.catalogFolders, id)
    || rowLabel(scopedRows.catalogTags, id);
}

function nodeDisplayName(nodeDeviceId: string) {
  const node = edgeDevices.value.find((item) => item.node_device_id === nodeDeviceId);
  const name = node?.display_name || t('cloud.edgeDevices.generatedNode');
  return `${name} (${shortId(nodeDeviceId)})`;
}

function readableDetailLabel(key: string) {
  const translationKey = `errors.details.${key}`;
  const translated = t(translationKey);
  return translated === translationKey ? key : translated;
}
</script>
