<template>
  <q-layout view="hHh LpR fFf" class="cloud-layout">
    <q-header class="cloud-header">
      <q-toolbar class="cloud-toolbar">
        <q-toolbar-title class="cloud-brand">{{ t('app.title') }}</q-toolbar-title>
        <span class="cloud-api">{{ apiBaseLabel }}</span>
        <q-btn flat dense icon="refresh" :label="t('actions.refresh')" :loading="activeLoading" @click="reloadActive" />
      </q-toolbar>
    </q-header>

    <q-page-container>
      <q-page class="cloud-page">
        <aside class="cloud-sidebar">
          <div class="cloud-sidebar-head">
            <p class="eyebrow">{{ t('cloud.scope') }}</p>
            <h1>{{ t('cloud.title') }}</h1>
          </div>

          <q-select
            v-model="selectedRestaurantId"
            dense
            outlined
            emit-value
            map-options
            :loading="isLoading('restaurants')"
            :label="t('cloud.restaurantFilter')"
            :options="restaurantOptions"
          />

          <section v-for="group in navGroups" :key="group.key" class="cloud-nav-group">
            <p class="cloud-nav-label">{{ t(group.labelKey) }}</p>
            <button
              v-for="item in group.items"
              :key="item.key"
              type="button"
              class="cloud-nav-item"
              :class="{ selected: activeKey === item.key }"
              @click="setActive(item.key)"
            >
              <span>{{ t(item.titleKey) }}</span>
              <strong>{{ navCount(item.key) }}</strong>
            </button>
          </section>
        </aside>

        <main class="cloud-main">
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

          <q-banner v-if="errorKey" class="error-banner dense-banner">
            {{ t(errorKey) }} <span v-if="errorCode">{{ errorCode }}</span>
          </q-banner>
          <q-banner v-if="successKey" class="success-banner dense-banner">{{ t(successKey) }}</q-banner>

          <section v-if="activeKey !== 'restaurants' && !selectedRestaurantId" class="empty-state wide">
            {{ t('cloud.empty.selectRestaurant') }}
          </section>

          <section v-else-if="activeKey === 'publications'" class="cloud-publication-grid">
            <div class="cloud-panel">
              <div class="section-head">
                <h2>{{ t('cloud.publications.currentState') }}</h2>
                <q-btn flat dense icon="refresh" :label="t('actions.refresh')" :loading="isLoading('publication')" @click="loadPublication" />
              </div>
              <div v-if="publication" class="cloud-state-grid">
                <div v-for="field in publicationFields" :key="field.key">
                  <span>{{ t(field.labelKey) }}</span>
                  <strong>{{ field.value }}</strong>
                </div>
              </div>
              <div v-else class="empty-state">{{ t('cloud.empty.noPublication') }}</div>
              <div v-if="publication" class="cloud-counts">
                <div v-for="[key, value] in publicationCounts" :key="key" class="cloud-count-row">
                  <span>{{ key }}</span>
                  <strong>{{ value }}</strong>
                </div>
              </div>
            </div>

            <form class="cloud-panel cloud-form-panel" @submit.prevent="publishSelectedRestaurant">
              <div class="section-head">
                <h2>{{ t('cloud.publications.publish') }}</h2>
              </div>
              <q-input v-model="publishForm.published_by" dense outlined :label="t('cloud.fields.published_by')" />
              <q-input v-model="publishForm.node_device_id" dense outlined :label="t('cloud.fields.node_device_id')" />
              <q-btn
                color="primary"
                unelevated
                icon="publish"
                type="submit"
                :loading="isLoading('publication-submit')"
                :label="t('cloud.publications.publishAction')"
              />
            </form>
          </section>

          <section v-else class="cloud-workspace">
            <div class="cloud-panel cloud-list-panel">
              <div class="cloud-list-tools">
                <q-input v-model="search" dense outlined clearable debounce="120" :label="t('cloud.search')" />
                <span>{{ t('cloud.rows') }}: {{ filteredRows.length }}</span>
              </div>
              <div v-if="activeConfig?.commandOnly" class="empty-state wide">
                {{ t('cloud.empty.commandOnly') }}
              </div>
              <div v-else-if="activeLoading" class="cloud-skeleton-list">
                <q-skeleton v-for="index in 6" :key="index" class="skeleton-row" />
              </div>
              <div v-else-if="filteredRows.length === 0" class="empty-state wide">
                {{ t('common.empty') }}
              </div>
              <div v-else class="cloud-table-wrap">
                <table class="cloud-table">
                  <thead>
                    <tr>
                      <th v-for="column in activeColumns" :key="column.key">{{ t(column.labelKey) }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in filteredRows"
                      :key="rowKey(row)"
                      :class="{ selected: selectedRowId === rowKey(row) }"
                      @click="selectRow(row)"
                    >
                      <td v-for="column in activeColumns" :key="column.key">
                        <span v-if="column.key === 'status'" class="cloud-status" :class="String(row[column.key])">
                          {{ statusText(row[column.key]) }}
                        </span>
                        <span v-else>{{ formatCell(column.key, row[column.key]) }}</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <form class="cloud-panel cloud-form-panel" @submit.prevent="submitForm">
              <div class="section-head">
                <h2>{{ t(mode === 'create' ? 'cloud.form.create' : 'cloud.form.edit') }}</h2>
                <q-btn v-if="mode === 'edit'" flat dense icon="add" :label="t('cloud.form.new')" @click="startCreate" />
              </div>

              <template v-for="field in visibleFields" :key="field.key">
                <q-checkbox v-if="field.type === 'checkbox'" v-model="form[field.key]" :label="t(field.labelKey)" />
                <q-select
                  v-else-if="selectOptions(field).length > 0"
                  v-model="form[field.key]"
                  dense
                  outlined
                  emit-value
                  map-options
                  :label="t(field.labelKey)"
                  :options="selectOptions(field)"
                />
                <q-input
                  v-else
                  :model-value="inputModelValue(field.key)"
                  @update:model-value="(value) => setFormValue(field.key, value)"
                  dense
                  outlined
                  :type="field.type === 'textarea' ? 'textarea' : field.type === 'number' ? 'number' : 'text'"
                  :rows="field.type === 'textarea' ? field.rows ?? 4 : undefined"
                  :label="t(field.labelKey)"
                />
              </template>

              <div class="cloud-form-actions">
                <q-btn
                  color="primary"
                  unelevated
                  icon="save"
                  type="submit"
                  :loading="isLoading('submit')"
                  :disable="!canSubmitActive"
                  :label="t(mode === 'create' ? 'cloud.form.createAction' : 'actions.save')"
                />
                <q-btn
                  v-if="mode === 'edit' && activeConfig?.archive"
                  flat
                  color="negative"
                  icon="archive"
                  :loading="isLoading('archive')"
                  :label="t('cloud.actions.archive')"
                  @click="archiveSelected"
                />
              </div>

              <div v-if="activeKey === 'employees' && mode === 'edit'" class="cloud-extra-actions">
                <q-separator />
                <div class="inline-action horizontal">
                  <q-btn flat icon="pause" :label="t('cloud.actions.suspend')" @click="employeeAction('suspend')" />
                  <q-btn flat icon="play_arrow" :label="t('cloud.actions.activate')" @click="employeeAction('activate')" />
                </div>
                <q-btn flat color="negative" icon="archive" :label="t('cloud.actions.archive')" @click="employeeAction('archive')" />
                <q-btn flat icon="assignment_ind" :label="t('cloud.actions.assignRole')" @click="assignSelectedEmployeeRole" />
                <div class="inline-action horizontal">
                  <q-input v-model="actionPin" dense outlined type="password" :label="t('cloud.fields.pin')" />
                  <q-btn flat icon="vpn_key" :label="t('cloud.actions.rotatePin')" @click="rotateSelectedEmployeePIN" />
                </div>
              </div>
            </form>
          </section>
        </main>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

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
  createRestaurant,
  createRole,
  createTable,
  getPublicationState,
  listCatalogFolders,
  listCatalogItems,
  listCatalogTags,
  listEmployees,
  listFolderParameters,
  listHalls,
  listMenuItems,
  listModifierBindings,
  listModifierGroups,
  listModifierOptions,
  listPricingPolicies,
  listRestaurants,
  listRoles,
  listTables,
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
  updateRestaurant,
  updateRole,
  updateTable,
  ApiError,
} from './shared/api';
import type { PublicationSummary, Restaurant } from './shared/schemas';

type ResourceKey =
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
  | 'halls'
  | 'tables'
  | 'menuItems'
  | 'categories'
  | 'publications';

type ScopedResourceKey = Exclude<ResourceKey, 'restaurants' | 'itemTags' | 'categories' | 'publications'>;
type FormMode = 'create' | 'edit';
type RowValue = string | number | boolean | null | undefined;
type Row = Record<string, RowValue>;
type SelectOption = { label: string; value: string | number | boolean };

type FieldConfig = {
  key: string;
  labelKey: string;
  type?: 'text' | 'number' | 'textarea' | 'checkbox';
  rows?: number;
  options?: string;
  createOnly?: boolean;
  updateOnly?: boolean;
  defaultValue?: RowValue;
};

type ResourceConfig = {
  key: Exclude<ResourceKey, 'publications'>;
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

const { t } = useI18n();

const apiBaseLabel = (import.meta.env.VITE_CLOUD_API_BASE ?? 'http://localhost:8090/api/v1').replace(/\/$/, '');
const selectedRestaurantId = ref('');
const activeKey = ref<ResourceKey>('restaurants');
const selectedRowId = ref('');
const search = ref('');
const mode = ref<FormMode>('create');
const actionPin = ref('');
const errorKey = ref('');
const errorCode = ref('');
const successKey = ref('');
const loading = ref<string[]>([]);
const restaurants = ref<Restaurant[]>([]);
const publication = ref<PublicationSummary | null>(null);
const publishForm = reactive({ published_by: '', node_device_id: '' });
const form = reactive<Row>({});

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
    fields: [field('name'), field('permissions_json', { type: 'textarea', rows: 5, defaultValue: '{}' }), field('active', { type: 'checkbox', updateOnly: true })],
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
    fields: [field('name'), field('kind', { options: 'pricingKinds', defaultValue: 'discount' }), field('scope'), field('amount_kind', { options: 'amountKinds', defaultValue: 'fixed' }), field('amount_minor', { type: 'number', defaultValue: 0 }), field('value_basis_points', { type: 'number', defaultValue: 0 }), field('application_index', { type: 'number', defaultValue: 1 }), field('manual', { type: 'checkbox' }), field('requires_permission'), lifecycleField],
    create: createPricingPolicy,
    update: updatePricingPolicy,
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
    fields: [field('catalog_item_id', { options: 'catalogItems' }), field('category_id'), field('name'), field('price', { type: 'number', defaultValue: 0 }), field('currency'), field('availability_json', { type: 'textarea', rows: 4, defaultValue: '{}' }), field('station_routing_key'), lifecycleField],
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

const navGroups = computed(() => {
  const groupKeys = ['cloud.groups.organization', 'cloud.groups.staff', 'cloud.groups.catalog', 'cloud.groups.modifiers', 'cloud.groups.pricing', 'cloud.groups.floor', 'cloud.groups.menu', 'cloud.groups.publication'];
  return groupKeys.map((labelKey) => ({
    key: labelKey,
    labelKey,
    items: [
      ...resourceConfigs.filter((item) => item.groupKey === labelKey),
      ...(labelKey === publicationNav.groupKey ? [publicationNav] : []),
    ],
  }));
});

const activeConfig = computed(() => resourceConfigs.find((item) => item.key === activeKey.value));
const activeTitleKey = computed(() => activeConfig.value?.titleKey ?? publicationNav.titleKey);
const activeGroupLabelKey = computed(() => activeConfig.value?.groupKey ?? publicationNav.groupKey);
const activeDescriptionKey = computed(() => activeConfig.value?.descriptionKey ?? publicationNav.descriptionKey);
const activeColumns = computed(() => activeConfig.value?.columns ?? []);
const visibleFields = computed(() => (activeConfig.value?.fields ?? []).filter((item) => !(mode.value === 'create' && item.updateOnly) && !(mode.value === 'edit' && item.createOnly)));
const currentRows = computed<Row[]>(() => {
  if (activeKey.value === 'restaurants') return restaurants.value as unknown as Row[];
  if (activeKey.value === 'publications' || activeKey.value === 'itemTags' || activeKey.value === 'categories') return [];
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

watch(selectedRestaurantId, async () => {
  clearMessages();
  if (!selectedRestaurantId.value) return;
  await loadScopedData();
  if (activeKey.value === 'publications') await loadPublication();
  resetSelection();
});

watch(activeKey, async () => {
  clearMessages();
  search.value = '';
  resetSelection();
  if (activeKey.value === 'publications') await loadPublication();
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

function navCount(key: ResourceKey) {
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
  return value;
}

function setFormValue(key: string, value: string | number | null) {
  form[key] = value;
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
  if (activeKey.value === 'restaurants') return loadRestaurants();
  if (activeKey.value === 'publications') return loadPublication();
  if (activeKey.value === 'itemTags' || activeKey.value === 'categories') return;
  return loadScopedResource(activeKey.value as ScopedResourceKey);
}

async function submitForm() {
  const config = activeConfig.value;
  if (!config?.create) return;
  await withLoading('submit', async () => {
    const payload = formPayload(config);
    if (mode.value === 'create') {
      await config.create?.(payload);
      successKey.value = 'cloud.messages.created';
    } else if (selectedRowId.value && config.update) {
      await config.update(selectedRowId.value, payload);
      successKey.value = 'cloud.messages.saved';
    }
    await reloadAfterMutation(config.key);
    startCreate();
  });
}

function formPayload(config: ResourceConfig) {
  const payload: Row = {};
  if (config.key !== 'restaurants') payload.restaurant_id = selectedRestaurantId.value;
  for (const item of config.fields) {
    if (mode.value === 'create' && item.updateOnly) continue;
    if (mode.value === 'edit' && item.createOnly) continue;
    if (!(item.key in form)) continue;
    payload[item.key] = normalizedValue(item, form[item.key]);
  }
  return payload;
}

function normalizedValue(item: FieldConfig, value: RowValue): RowValue {
  if (item.type === 'number') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  if (item.type === 'checkbox') return Boolean(value);
  return typeof value === 'string' ? value.trim() : value;
}

async function archiveSelected() {
  const config = activeConfig.value;
  if (!config?.archive || !selectedRowId.value) return;
  await withLoading('archive', async () => {
    await config.archive?.(selectedRowId.value);
    successKey.value = 'cloud.messages.archived';
    await reloadAfterMutation(config.key);
    startCreate();
  });
}

async function employeeAction(action: 'suspend' | 'activate' | 'archive') {
  if (!selectedRowId.value) return;
  await withLoading('employee-action', async () => {
    if (action === 'suspend') await suspendEmployee(selectedRowId.value);
    if (action === 'activate') await activateEmployee(selectedRowId.value);
    if (action === 'archive') await archiveEmployee(selectedRowId.value);
    successKey.value = 'cloud.messages.saved';
    await loadScopedResource('employees');
  });
}

async function assignSelectedEmployeeRole() {
  if (!selectedRowId.value || !form.role_id) return;
  await withLoading('employee-action', async () => {
    await assignEmployeeRole(selectedRowId.value, String(form.role_id));
    successKey.value = 'cloud.messages.saved';
    await loadScopedResource('employees');
  });
}

async function rotateSelectedEmployeePIN() {
  if (!selectedRowId.value || !actionPin.value.trim()) return;
  await withLoading('employee-action', async () => {
    await rotateEmployeePIN(selectedRowId.value, actionPin.value.trim());
    actionPin.value = '';
    successKey.value = 'cloud.messages.saved';
    await loadScopedResource('employees');
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
    return;
  }
  errorKey.value = 'errors.unexpected';
  errorCode.value = '';
}

function clearMessages() {
  errorKey.value = '';
  errorCode.value = '';
  successKey.value = '';
}

function formatCell(key: string, value: unknown) {
  if (value === null || value === undefined || value === '') return '-';
  if (typeof value === 'boolean') return t(value ? 'common.yes' : 'common.no');
  if (key.endsWith('_at') && typeof value === 'string') return formatDate(value);
  if ((key.endsWith('_id') || key === 'id') && typeof value === 'string') return shortId(value);
  return String(value);
}

function statusText(value: unknown) {
  return t(`cloud.statuses.${String(value)}`);
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
</script>
