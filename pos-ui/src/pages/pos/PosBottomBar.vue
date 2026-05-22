<template>
  <footer class="pos-bottom-bar" :aria-label="terminal.t('pos.quickAccess')">
    <button class="bottom-section-button" :class="{ active: menuOpen }" type="button" @click="$emit('toggle-menu')">
      <q-icon name="apps" size="24px" />
      <span>{{ terminal.t(activeSectionLabelKey) }}</span>
    </button>

    <div class="context-chip-row" :aria-label="terminal.t('pos.currentContext')">
      <span v-for="chip in contextChips" :key="chip.key" class="context-chip" :class="chip.tone">
        <q-icon class="context-chip-icon" :name="chip.icon" size="18px" />
        <span class="context-chip-label">{{ chip.label }}</span>
      </span>
    </div>

    <div class="bottom-status-area" :aria-label="terminal.t('pos.terminalStatus')">
      <button
        v-for="status in statusItems"
        :key="status.key"
        class="bottom-status-item"
        :class="[{ clickable: status.clickable }, status.tone]"
        type="button"
        :disabled="!status.clickable"
        @click="status.action?.()"
      >
        <q-icon :name="status.icon" size="18px" />
        <span>{{ status.label }}</span>
      </button>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import type { CashierTerminal } from './useCashierTerminal';

type BottomChip = {
  key: string;
  icon: string;
  label: string;
  tone?: 'good' | 'warning' | 'total';
};

type BottomStatusItem = {
  key: string;
  icon: string;
  label: string;
  tone?: 'good' | 'warning';
  clickable?: boolean;
  action?: () => void;
};

const props = defineProps<{
  terminal: CashierTerminal;
  activeSection: string;
  menuOpen: boolean;
}>();

defineEmits<{
  (event: 'toggle-menu'): void;
}>();

const activeSectionLabelKey = computed(() => `pos.sections.${props.activeSection}`);

const contextChips = computed<BottomChip[]>(() => {
  const chips: BottomChip[] = [];
  const tableName = props.terminal.selectedTable.value?.name;
  const order = props.terminal.activeOrder.value;

  chips.push({
    key: 'table',
    icon: 'table_restaurant',
    label: tableName ? props.terminal.t('pos.tableWithName', { table: tableName }) : props.terminal.t('pos.chooseTable'),
  });

  if (order) {
    chips.push({
      key: 'order',
      icon: 'receipt_long',
      label: props.terminal.t('pos.orderShort', { number: props.terminal.shortId(order.id) }),
    });
    chips.push({
      key: 'total',
      icon: 'payments',
      label: props.terminal.money(order.total, props.terminal.orderCurrency.value),
      tone: 'total',
    });
  } else {
    chips.push({
      key: 'order-empty',
      icon: 'receipt_long',
      label: props.terminal.t('pos.noActiveOrderShort'),
    });
  }

  if (props.terminal.activePrecheck.value || order?.status === 'locked') {
    chips.push({
      key: 'precheck',
      icon: 'lock',
      label: props.terminal.t('pos.precheckIssued'),
      tone: 'warning',
    });
  }

  return chips;
});

const statusItems = computed<BottomStatusItem[]>(() => [
  {
    key: 'shift',
    icon: 'schedule',
    label: props.terminal.currentShift.data.value ? props.terminal.t('status.open') : props.terminal.t('pos.noShift'),
    tone: props.terminal.currentShift.data.value ? 'good' : 'warning',
  },
  {
    key: 'cash',
    icon: 'point_of_sale',
    label: props.terminal.currentCashSession.data.value ? props.terminal.t('status.open') : props.terminal.t('pos.noCashSession'),
    tone: props.terminal.currentCashSession.data.value ? 'good' : 'warning',
  },
  {
    key: 'sync',
    icon: 'sync',
    label: props.terminal.syncProblems.value > 0 ? props.terminal.t('pos.syncFailed') : props.terminal.t('status.sent'),
    tone: props.terminal.syncProblems.value > 0 ? 'warning' : 'good',
    clickable: props.terminal.canViewSync.value,
    action: () => {
      props.terminal.syncDrawer.value = true;
    },
  },
  {
    key: 'actor',
    icon: 'person',
    label: props.terminal.actorName.value || props.terminal.t('pos.actor'),
  },
]);
</script>
