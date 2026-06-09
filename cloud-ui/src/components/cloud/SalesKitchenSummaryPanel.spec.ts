// @vitest-environment happy-dom

import { mount } from '@vue/test-utils';
import { defineComponent, h, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import SalesKitchenSummaryPanel from './SalesKitchenSummaryPanel.vue';

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
    salesKitchenSummaryRows: ref([
      {
        group_by: 'catalog_item',
        group_key: 'catalog-item-1',
        business_date_local: '2026-05-30',
        event_type: 'CheckClosed',
        source_event_type: 'ItemServed',
        catalog_item_id: 'catalog-item-1',
        event_count: 3,
        stock_move_count: 2,
        sale_event_count: 1,
        kitchen_event_count: 2,
        out_quantity: '2.000',
        in_quantity: '0.000',
        net_quantity: '-2.000',
        total_cost_minor: -500,
        first_occurred_at: '2026-05-30T10:00:00Z',
        last_occurred_at: '2026-05-30T10:30:00Z',
      },
    ]),
    salesKitchenSummaryFilters: {
      businessDateFrom: '',
      businessDateTo: '',
      groupBy: 'business_date',
    },
    formatCell: (_field: string, value: string) => value || '-',
    isLoading: vi.fn().mockReturnValue(false),
    loadSalesKitchenSummary: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

function mountPanel(ctx = createCtx()) {
  return mount(SalesKitchenSummaryPanel, {
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

describe('SalesKitchenSummaryPanel', () => {
  it('renders title, description and status signals through i18n', () => {
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.status');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.title');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.description');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.signals.readOnly');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.signals.noRawPayload');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.signals.noCostingBi');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.signals.noCharts');
  });

  it('renders date filters and group_by select', () => {
    const wrapper = mountPanel();

    expect(wrapper.find('[data-label="cloud.reporting.salesKitchen.filters.businessDateFrom"] input').exists()).toBe(true);
    expect(wrapper.find('[data-label="cloud.reporting.salesKitchen.filters.businessDateTo"] input').exists()).toBe(true);
    expect(wrapper.find('[data-label="cloud.reporting.salesKitchen.filters.groupBy"] select').exists()).toBe(true);
    expect(wrapper.find('option[value="business_date"]').exists()).toBe(true);
    expect(wrapper.find('option[value="event_type"]').exists()).toBe(true);
    expect(wrapper.find('option[value="source_event_type"]').exists()).toBe(true);
    expect(wrapper.find('option[value="catalog_item"]').exists()).toBe(true);
  });

  it('calls ctx.loadSalesKitchenSummary on refresh and apply', async () => {
    const ctx = createCtx();
    const wrapper = mountPanel(ctx);
    const buttons = wrapper.findAll('button');

    await buttons[0].trigger('click');
    await buttons[1].trigger('click');

    expect(ctx.loadSalesKitchenSummary).toHaveBeenCalledTimes(2);
  });

  it('renders empty state when rows are empty', () => {
    const wrapper = mountPanel(createCtx({ salesKitchenSummaryRows: ref([]) }));

    expect(wrapper.find('table.cloud-table').exists()).toBe(false);
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.empty');
  });

  it('renders a bounded aggregate row with safe summary fields', () => {
    const wrapper = mountPanel();

    expect(wrapper.find('table.cloud-table').exists()).toBe(true);
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.groupBy.catalogItem');
    expect(wrapper.text()).toContain('catalog-item-1');
    expect(wrapper.text()).toContain('2026-05-30');
    expect(wrapper.text()).toContain('CheckClosed');
    expect(wrapper.text()).toContain('ItemServed');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.counts.events:total=3,sales=1,kitchen=2');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.counts.quantities:out=2.000,in=0.000,net=-2.000');
    expect(wrapper.text()).toContain('-500');
    expect(wrapper.text()).toContain('cloud.reporting.salesKitchen.counts.period');
  });

  it('does not render raw payload, snapshot, retry, backfill or chart controls', () => {
    const wrapper = mountPanel(createCtx({
      salesKitchenSummaryRows: ref([{
        group_by: 'business_date',
        group_key: '2026-05-30',
        business_date_local: '2026-05-30',
        event_type: '',
        source_event_type: '',
        catalog_item_id: '',
        event_count: 1,
        stock_move_count: 0,
        sale_event_count: 1,
        kitchen_event_count: 0,
        out_quantity: '0.000',
        in_quantity: '0.000',
        net_quantity: '0.000',
        total_cost_minor: 0,
        payload: '{"unsafe":true}',
        raw_payload_sha256_hex: 'hash',
        snapshot_json: '{}',
      }]),
    }));
    const text = wrapper.text();

    expect(text).not.toContain('{"unsafe":true}');
    expect(text).not.toContain('raw_payload_sha256_hex');
    expect(text).not.toContain('snapshot_json');
    expect(text).not.toContain('cloud.readiness.olap.retry');
    expect(text).not.toContain('cloud.readiness.olap.backfill');
    expect(wrapper.find('canvas').exists()).toBe(false);
    expect(wrapper.find('svg').exists()).toBe(false);
  });
});
