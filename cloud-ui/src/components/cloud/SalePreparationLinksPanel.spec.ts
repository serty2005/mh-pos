// @vitest-environment happy-dom

import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import SalePreparationLinksPanel from './SalePreparationLinksPanel.vue';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (!params) return key;
      return `${key}:${Object.entries(params).map(([name, value]) => `${name}=${value}`).join(',')}`;
    },
  }),
}));

const QuasarInputStub = defineComponent({
  name: 'QInput',
  props: {
    modelValue: { type: String, default: '' },
    label: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('label', { 'data-test': 'q-input', 'data-label': props.label }, [
      h('span', props.label),
      h('input', {
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      }),
    ]);
  },
});

const QuasarSelectStub = defineComponent({
  name: 'QSelect',
  props: {
    modelValue: { type: String, default: '' },
    label: { type: String, default: '' },
    options: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('label', { 'data-test': 'q-select', 'data-label': props.label }, [
      h('span', props.label),
      h('select', {
        value: props.modelValue,
        onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
      }, (props.options as Array<{ label: string; value: string }>).map((option) => (
        h('option', { value: option.value }, option.label)
      ))),
    ]);
  },
});

const QuasarToggleStub = defineComponent({
  name: 'QToggle',
  props: {
    modelValue: { type: Boolean, default: false },
    label: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('label', { 'data-test': 'q-toggle', 'data-label': props.label }, [
      h('span', props.label),
      h('input', {
        checked: props.modelValue,
        type: 'checkbox',
        onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).checked),
      }),
    ]);
  },
});

const QuasarButtonStub = defineComponent({
  name: 'QBtn',
  props: {
    label: { type: String, default: '' },
    loading: { type: Boolean, default: false },
  },
  emits: ['click'],
  setup(props, { emit }) {
    return () => h('button', {
      disabled: props.loading,
      type: 'button',
      onClick: () => emit('click'),
    }, props.label);
  },
});

function createCtx(overrides: Record<string, unknown> = {}) {
  return {
    scopedRows: {
      catalogItems: [
        {
          id: 'catalog-pizza',
          restaurant_id: 'restaurant-1',
          kind: 'dish',
          folder_id: '',
          name: 'Margherita catalog',
          sku: 'PIZZA-1',
          base_unit: 'portion',
          kitchen_type: 'hot',
          accounting_category: 'food',
          status: 'published',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
          payload_json: '{"raw":true}',
        },
        {
          id: 'catalog-draft',
          restaurant_id: 'restaurant-1',
          kind: 'dish',
          folder_id: '',
          name: 'Draft catalog',
          sku: 'DRAFT-1',
          base_unit: 'portion',
          kitchen_type: 'hot',
          accounting_category: 'food',
          status: 'draft',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        },
        {
          id: 'catalog-soup',
          restaurant_id: 'restaurant-1',
          kind: 'dish',
          folder_id: '',
          name: 'Soup catalog',
          sku: 'SOUP-1',
          base_unit: 'portion',
          kitchen_type: 'hot',
          accounting_category: 'food',
          status: 'published',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        },
      ],
      menuItems: [
        {
          id: 'menu-pizza',
          restaurant_id: 'restaurant-1',
          catalog_item_id: 'catalog-pizza',
          category_id: '',
          name: 'Margherita',
          price: 1200,
          currency: 'RUB',
          status: 'published',
          availability_json: '{}',
          station_routing_key: 'hot',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
          raw_payload: '{"unsafe":true}',
        },
        {
          id: 'menu-missing-catalog',
          restaurant_id: 'restaurant-1',
          catalog_item_id: 'catalog-missing',
          category_id: '',
          name: 'Ghost dish',
          price: 900,
          currency: 'RUB',
          status: 'published',
          availability_json: '{}',
          station_routing_key: '',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        },
        {
          id: 'menu-soup',
          restaurant_id: 'restaurant-1',
          catalog_item_id: 'catalog-soup',
          category_id: '',
          name: 'Soup',
          price: 700,
          currency: 'RUB',
          status: 'published',
          availability_json: '{}',
          station_routing_key: 'hot',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        },
      ],
      modifierGroups: [
        {
          id: 'modifier-group-1',
          restaurant_id: 'restaurant-1',
          name: 'Cheese',
          required: false,
          min_count: 0,
          max_count: 2,
          status: 'published',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        },
      ],
      modifierBindings: [
        {
          id: 'binding-1',
          restaurant_id: 'restaurant-1',
          modifier_group_id: 'modifier-group-1',
          target_type: 'menu_item',
          target_id: 'menu-pizza',
          sort_order: 1,
          status: 'published',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        },
      ],
      pricingPolicies: [
        {
          id: 'policy-1',
          restaurant_id: 'restaurant-1',
          name: 'Lunch discount',
          kind: 'discount',
          scope: 'restaurant',
          amount_kind: 'percentage',
          amount_minor: 0,
          value_basis_points: 500,
          application_index: 1,
          manual: false,
          requires_permission: '',
          status: 'published',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        },
      ],
    },
    isLoading: vi.fn().mockReturnValue(false),
    loadSalePreparationLinks: vi.fn().mockResolvedValue(undefined),
    safeOperationalValue: (value: string) => `safe:${value}`,
    statusText: (value: string) => `status:${value}`,
    ...overrides,
  };
}

function mountPanel(ctx = createCtx()) {
  return mount(SalePreparationLinksPanel, {
    props: { ctx },
    global: {
      stubs: {
        'cloud-safe-error-banner': true,
        QBtn: QuasarButtonStub,
        QInput: QuasarInputStub,
        QSelect: QuasarSelectStub,
        QToggle: QuasarToggleStub,
      },
    },
  });
}

function readyCatalogItem(index: number) {
  return {
    id: `catalog-${index}`,
    restaurant_id: 'restaurant-1',
    kind: 'dish',
    folder_id: '',
    name: `Catalog ${index}`,
    sku: `SKU-${index}`,
    base_unit: 'portion',
    kitchen_type: 'hot',
    accounting_category: 'food',
    status: 'published',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
  };
}

function readyMenuItem(index: number) {
  return {
    id: `menu-${index}`,
    restaurant_id: 'restaurant-1',
    catalog_item_id: `catalog-${index}`,
    category_id: '',
    name: `Menu item ${index}`,
    price: 1000 + index,
    currency: 'RUB',
    status: 'published',
    availability_json: '{}',
    station_routing_key: 'hot',
    cloud_version: 1,
    created_at: '2026-06-01T10:00:00Z',
    updated_at: '2026-06-01T10:00:00Z',
  };
}

describe('SalePreparationLinksPanel', () => {
  it('renders ready, missing and warning rows from supplied Cloud master data', () => {
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain('cloud.salePreparation.readiness.ready');
    expect(wrapper.text()).toContain('cloud.salePreparation.readiness.missing');
    expect(wrapper.text()).toContain('cloud.salePreparation.readiness.warning');
    expect(wrapper.text()).toContain('Margherita');
    expect(wrapper.text()).toContain('Margherita catalog');
    expect(wrapper.text()).toContain('Cheese');
    expect(wrapper.text()).toContain('cloud.salePreparation.summaries.pricing:published=1,total=1');
    expect(wrapper.text()).toContain('cloud.salePreparation.hints.missingCatalogItem');
    expect(wrapper.text()).toContain('cloud.salePreparation.hints.noModifierBindings');
  });

  it('renders all visible labels through i18n keys', () => {
    const wrapper = mountPanel();
    const text = wrapper.text();

    expect(text).toContain('cloud.salePreparation.status');
    expect(text).toContain('cloud.salePreparation.title');
    expect(text).toContain('cloud.salePreparation.description');
    expect(text).toContain('cloud.salePreparation.filters.search');
    expect(text).toContain('cloud.salePreparation.filters.status');
    expect(text).toContain('cloud.salePreparation.filters.onlyIncomplete');
    expect(text).toContain('cloud.salePreparation.signals.readOnly');
    expect(text).toContain('cloud.salePreparation.signals.cloudMasterDataOnly');
    expect(text).toContain('cloud.salePreparation.signals.noCashierCommands');
    expect(text).toContain('cloud.salePreparation.signals.noRawPayload');
    expect(text).toContain('cloud.salePreparation.columns.menuItem');
    expect(text).toContain('cloud.salePreparation.columns.catalogItem');
    expect(text).toContain('cloud.salePreparation.columns.modifiers');
    expect(text).toContain('cloud.salePreparation.columns.pricing');
    expect(text).toContain('cloud.salePreparation.columns.readiness');
    expect(text).toContain('cloud.salePreparation.columns.hint');
  });

  it('refresh action calls ctx.loadSalePreparationLinks', async () => {
    const ctx = createCtx();
    const wrapper = mountPanel(ctx);

    await wrapper.find('button').trigger('click');

    expect(ctx.loadSalePreparationLinks).toHaveBeenCalledTimes(1);
  });

  it('bounds preview to 50 rows and shows bounded note', () => {
    const count = 55;
    const catalogItems = Array.from({ length: count }, (_, index) => readyCatalogItem(index + 1));
    const menuItems = Array.from({ length: count }, (_, index) => readyMenuItem(index + 1));
    const modifierBindings = Array.from({ length: count }, (_, index) => ({
      id: `binding-${index + 1}`,
      restaurant_id: 'restaurant-1',
      modifier_group_id: 'modifier-group-1',
      target_type: 'menu_item',
      target_id: `menu-${index + 1}`,
      sort_order: index + 1,
      status: 'published',
      cloud_version: 1,
      created_at: '2026-06-01T10:00:00Z',
      updated_at: '2026-06-01T10:00:00Z',
    }));
    const baseCtx = createCtx();
    const ctx = createCtx({
      scopedRows: {
        ...baseCtx.scopedRows,
        catalogItems,
        menuItems,
        modifierBindings,
      },
    });

    const wrapper = mountPanel(ctx);

    expect(wrapper.findAll('tbody tr')).toHaveLength(50);
    expect(wrapper.text()).toContain('cloud.salePreparation.boundedNote:shown=50,total=55');
    expect(wrapper.text()).toContain('Menu item 50');
    expect(wrapper.text()).not.toContain('Menu item 51');
  });

  it('filters rows by search without backend calls', async () => {
    const ctx = createCtx();
    const wrapper = mountPanel(ctx);

    await wrapper.find('[data-label="cloud.salePreparation.filters.search"] input').setValue('PIZZA-1');

    expect(wrapper.text()).toContain('Margherita');
    expect(wrapper.text()).not.toContain('Ghost dish');
    expect(wrapper.text()).not.toContain('Soup');
    expect(ctx.loadSalePreparationLinks).not.toHaveBeenCalled();
  });

  it('filters rows by readiness status without backend calls', async () => {
    const ctx = createCtx();
    const wrapper = mountPanel(ctx);
    const select = wrapper.find('[data-label="cloud.salePreparation.filters.status"] select');

    await select.setValue('ready');
    expect(wrapper.text()).toContain('Margherita');
    expect(wrapper.text()).not.toContain('Ghost dish');
    expect(wrapper.text()).not.toContain('Soup');

    await select.setValue('missing');
    expect(wrapper.text()).not.toContain('Margherita');
    expect(wrapper.text()).toContain('Ghost dish');
    expect(wrapper.text()).not.toContain('Soup');

    await select.setValue('warning');
    expect(wrapper.text()).not.toContain('Margherita');
    expect(wrapper.text()).not.toContain('Ghost dish');
    expect(wrapper.text()).toContain('Soup');
    expect(ctx.loadSalePreparationLinks).not.toHaveBeenCalled();
  });

  it('filters only incomplete rows', async () => {
    const wrapper = mountPanel();

    await wrapper.find('[data-label="cloud.salePreparation.filters.onlyIncomplete"] input').setValue(true);

    expect(wrapper.text()).not.toContain('Margherita catalog');
    expect(wrapper.text()).toContain('Ghost dish');
    expect(wrapper.text()).toContain('Soup');
  });

  it('keeps missing readiness ahead of warning reasons', () => {
    const baseCtx = createCtx();
    const missingCatalogCtx = createCtx({
      scopedRows: {
        ...baseCtx.scopedRows,
        menuItems: [{
          ...readyMenuItem(1),
          id: 'menu-draft-missing',
          catalog_item_id: 'catalog-missing',
          name: 'Draft missing catalog',
          status: 'draft',
        }],
        modifierBindings: [],
        pricingPolicies: [],
      },
    });
    const unpublishedCatalogCtx = createCtx({
      scopedRows: {
        ...baseCtx.scopedRows,
        catalogItems: [{ ...readyCatalogItem(1), id: 'catalog-draft', status: 'draft' }],
        menuItems: [{ ...readyMenuItem(1), catalog_item_id: 'catalog-draft', name: 'Unpublished catalog item' }],
        modifierBindings: [],
        pricingPolicies: [],
      },
    });
    const unpublishedMenuCtx = createCtx({
      scopedRows: {
        ...baseCtx.scopedRows,
        catalogItems: [readyCatalogItem(1)],
        menuItems: [{ ...readyMenuItem(1), name: 'Unpublished menu item', status: 'draft' }],
        modifierBindings: [],
        pricingPolicies: [],
      },
    });

    expect(mountPanel(missingCatalogCtx).text()).toContain('cloud.salePreparation.hints.missingCatalogItem');
    expect(mountPanel(missingCatalogCtx).text()).not.toContain('cloud.salePreparation.hints.noModifierBindings');
    expect(mountPanel(unpublishedCatalogCtx).text()).toContain('cloud.salePreparation.hints.catalogNotPublished');
    expect(mountPanel(unpublishedCatalogCtx).text()).not.toContain('cloud.salePreparation.hints.noPricingPolicy');
    expect(mountPanel(unpublishedMenuCtx).text()).toContain('cloud.salePreparation.hints.menuItemNotPublished');
    expect(mountPanel(unpublishedMenuCtx).text()).not.toContain('cloud.salePreparation.hints.noModifierBindings');
  });

  it('warns when a published sellable row has no published pricing policy', () => {
    const baseCtx = createCtx();
    const ctx = createCtx({
      scopedRows: {
        ...baseCtx.scopedRows,
        catalogItems: [readyCatalogItem(1)],
        menuItems: [readyMenuItem(1)],
        modifierBindings: [{
          id: 'binding-1',
          restaurant_id: 'restaurant-1',
          modifier_group_id: 'modifier-group-1',
          target_type: 'menu_item',
          target_id: 'menu-1',
          sort_order: 1,
          status: 'published',
          cloud_version: 1,
          created_at: '2026-06-01T10:00:00Z',
          updated_at: '2026-06-01T10:00:00Z',
        }],
        pricingPolicies: [{
          ...baseCtx.scopedRows.pricingPolicies[0],
          status: 'draft',
        }],
      },
    });

    const wrapper = mountPanel(ctx);

    expect(wrapper.text()).toContain('cloud.salePreparation.readiness.warning');
    expect(wrapper.text()).toContain('cloud.salePreparation.hints.noPricingPolicy');
  });

  it('renders card fallback and does not display raw payload-like fields', () => {
    const wrapper = mountPanel();

    expect(wrapper.find('.edge-event-card-list').exists()).toBe(true);
    expect(wrapper.find('.edge-event-card').text()).toContain('safe:menu-pizza');
    expect(wrapper.find('.edge-event-card').text()).toContain('status:published');
    expect(wrapper.find('.edge-event-card').text()).toContain('cloud.salePreparation.summaries.modifiers:count=1,groups=Cheese');
    expect(wrapper.find('.edge-event-card').text()).toContain('cloud.salePreparation.hints.ready');
    expect(wrapper.text()).not.toContain('{"unsafe":true}');
    expect(wrapper.text()).not.toContain('payload_json');
    expect(wrapper.text()).not.toContain('raw_payload');
  });
});
