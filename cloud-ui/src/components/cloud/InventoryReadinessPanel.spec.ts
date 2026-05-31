// @vitest-environment happy-dom

import { mount } from '@vue/test-utils';
import { defineComponent, h, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import InventoryReadinessPanel from './InventoryReadinessPanel.vue';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
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
    activeLoading: ref(false),
    stopListReadiness: ref({
      restaurant_id: 'restaurant-1',
      default_conflict_policy: 'manager_review',
      projection_mode: 'cloud_authoritative',
      active_stop_list_entries: 2,
      total_stop_list_entries: 5,
      latest_publication: {
        version: 12,
        published_at: '2026-05-30T10:00:00Z',
      },
      latest_stop_list_edge_ack: {
        status: 'applied',
        event_id: 'edge-event-1',
      },
      problem_events: {
        total: 3,
        by_error_code: [],
      },
      package_ack_status: 'pending',
      package_ack_status_reason_key: 'cloud.readiness.stopList.pending',
    }),
    inventoryStockBalances: ref([
      {
        restaurant_id: 'restaurant-1',
        warehouse_id: 'warehouse-1',
        catalog_item_id: 'catalog-item-1',
        quantity_on_hand: '12.500',
        unit_code: 'kg',
        costing_status: 'final',
        needs_recalculation: false,
        last_movement_at: '2026-05-30T10:00:00Z',
        business_date_to: '2026-05-30',
      },
    ]),
    inventoryStockLedger: ref([
      {
        id: 'ledger-entry-1',
        restaurant_id: 'restaurant-1',
        warehouse_id: 'warehouse-1',
        stock_document_id: 'stock-document-1',
        source_event_id: 'source-event-1',
        source_event_type: 'OrderLineServed',
        catalog_item_id: 'catalog-item-1',
        order_line_id: 'order-line-1',
        movement_type: 'sale',
        quantity: '-2.000',
        unit_code: 'kg',
        unit_cost_minor: 125,
        total_cost_minor: -250,
        costing_status: 'estimated',
        occurred_at: '2026-05-30T10:05:00Z',
        business_date_local: '2026-05-30',
        created_at: '2026-05-30T10:06:00Z',
      },
    ]),
    scopedRows: {
      catalogItems: [{ id: 'catalog-item-1', name: 'Milk' }],
      stopLists: [{ warehouse_id: 'warehouse-1' }],
    },
    safeOperationalValue: (value: string) => `safe:${value}`,
    formatDate: (value: string) => `fmt:${value}`,
    loadStopListReadiness: vi.fn().mockResolvedValue(undefined),
    loadInventoryStockBalances: vi.fn().mockResolvedValue(undefined),
    loadInventoryStockLedger: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

function mountPanel(ctx = createCtx()) {
  return mount(InventoryReadinessPanel, {
    props: { ctx },
    global: {
      stubs: {
        'cloud-safe-error-banner': true,
        QBtn: QuasarButtonStub,
        QInput: QuasarInputStub,
        QSelect: QuasarSelectStub,
      },
    },
  });
}

describe('InventoryReadinessPanel', () => {
  it('renders stop-list readiness summary from ctx.stopListReadiness', () => {
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain('manager_review');
    expect(wrapper.text()).toContain('cloud_authoritative');
    expect(wrapper.text()).toContain('2 / 5');
    expect(wrapper.text()).toContain('pending');
    expect(wrapper.text()).toContain('12 / fmt:2026-05-30T10:00:00Z');
    expect(wrapper.text()).toContain('applied / safe:edge-event-1');
    expect(wrapper.text()).toContain('3');
  });

  it('renders inventory stock balances from ctx.inventoryStockBalances', () => {
    const wrapper = mountPanel();

    expect(wrapper.find('table.cloud-table').exists()).toBe(true);
    expect(wrapper.text()).toContain('warehouse-1');
    expect(wrapper.text()).toContain('safe:catalog-item-1');
    expect(wrapper.text()).toContain('12.500 kg');
    expect(wrapper.text()).toContain('cloud.readiness.inventory.costingStatuses.final');
    expect(wrapper.text()).toContain('fmt:2026-05-30T10:00:00Z');
  });

  it('renders empty state when inventory stock balances are empty', () => {
    const wrapper = mountPanel(createCtx({
      inventoryStockBalances: ref([]),
      inventoryStockLedger: ref([]),
    }));

    expect(wrapper.find('table.cloud-table').exists()).toBe(false);
    expect(wrapper.text()).toContain('cloud.readiness.inventory.emptyBalances');
  });

  it('refresh action calls readiness and balance loaders', async () => {
    const ctx = createCtx();
    const wrapper = mountPanel(ctx);

    await wrapper.find('button').trigger('click');

    expect(ctx.loadStopListReadiness).toHaveBeenCalledTimes(1);
    expect(ctx.loadInventoryStockBalances).toHaveBeenCalledTimes(1);
  });

  it('passes inventory filters to loadInventoryStockBalances with camelCase contract', async () => {
    const ctx = createCtx();
    const wrapper = mountPanel(ctx);

    await wrapper.find('[data-label="cloud.readiness.inventory.filters.warehouse"] select').setValue('warehouse-1');
    await wrapper.find('[data-label="cloud.readiness.inventory.filters.catalogItem"] select').setValue('catalog-item-1');
    await wrapper.find('[data-label="cloud.readiness.inventory.filters.businessDateTo"] input').setValue('2026-05-30');
    await wrapper.find('[data-label="cloud.readiness.inventory.filters.costingStatus"] select').setValue('final');
    await wrapper.find('button').trigger('click');

    expect(ctx.loadInventoryStockBalances).toHaveBeenCalledWith({
      warehouseId: 'warehouse-1',
      catalogItemId: 'catalog-item-1',
      businessDateTo: '2026-05-30',
      costingStatus: 'final',
    });
  });

  it('renders inventory stock ledger preview from ctx.inventoryStockLedger', () => {
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain('cloud.readiness.inventory.ledger.title');
    expect(wrapper.text()).toContain('OrderLineServed');
    expect(wrapper.text()).toContain('Milk (safe:catalog-item-1)');
    expect(wrapper.text()).toContain('-2.000 kg');
    expect(wrapper.text()).toContain('estimated');
    expect(wrapper.text()).toContain('-250');
    expect(wrapper.text()).toContain('fmt:2026-05-30T10:05:00Z');
  });

  it('passes inventory stock ledger filters with camelCase contract', async () => {
    const ctx = createCtx();
    const wrapper = mountPanel(ctx);

    await wrapper.find('[data-label="cloud.readiness.inventory.ledger.filters.catalogItem"] select').setValue('catalog-item-1');
    await wrapper.find('[data-label="cloud.readiness.inventory.ledger.filters.sourceEventType"] input').setValue('OrderLineServed');
    await wrapper.find('[data-label="cloud.readiness.inventory.ledger.filters.sourceEventId"] input').setValue('source-event-1');
    await wrapper.find('[data-label="cloud.readiness.inventory.ledger.filters.orderLineId"] input').setValue('order-line-1');
    const buttons = wrapper.findAll('button');
    expect(buttons.length).toBeGreaterThan(1);
    await buttons[buttons.length - 1].trigger('click');

    expect(ctx.loadInventoryStockLedger).toHaveBeenCalledWith({
      catalogItemId: 'catalog-item-1',
      sourceEventType: 'OrderLineServed',
      sourceEventId: 'source-event-1',
      orderLineId: 'order-line-1',
    });
  });
});
