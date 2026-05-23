<template>
  <section class="menu-workspace" :aria-label="terminal.t('pos.menu')">
    <PosBanner v-if="terminal.statusError.value" tone="error" :label="terminal.statusError.value" />
    <PosBanner v-if="terminal.menu.isError.value" tone="error" :label="terminal.t('common.error')" />

    <div class="menu-search-row">
      <q-input
        v-model="terminal.menuSearch.value"
        outlined
        dense
        square
        clearable
        class="menu-search-field"
        :label="terminal.t('pos.searchMenu')"
        type="search"
      >
        <template #prepend>
          <q-icon name="search" />
        </template>
      </q-input>
    </div>

    <PosTabs :model-value="group" :options="categoryTabs" :accessibility-label="terminal.t('pos.menuGroups')" @update:model-value="setGroup" />

    <div v-if="readonlyNoticeKey" class="menu-readonly-note">
      <q-icon name="lock" size="18px" />
      <span>{{ terminal.t(readonlyNoticeKey) }}</span>
    </div>

    <PosEmptyState v-if="!terminal.canViewMenu.value" size="wide" :label="terminal.t('pos.noPermissionForMenu')" />

    <div v-else-if="terminal.menu.isPending.value" class="dish-grid">
      <PosSkeleton v-for="n in 12" :key="n" kind="card" />
    </div>

    <div v-else-if="visibleItems.length" class="dish-grid">
      <button
        v-for="item in visibleItems"
        :key="item.id"
        class="dish-card"
        type="button"
        :disabled="!terminal.canAddOrderLine.value"
        @click="terminal.openMenuItem(item)"
      >
        <span class="dish-card-media">{{ initials(item.name) }}</span>
        <span class="dish-card-title">{{ item.name }}</span>
        <strong>{{ terminal.money(item.price, item.currency) }}</strong>
        <small v-if="!terminal.canAddOrderLine.value && disabledHintKey">{{ terminal.t(disabledHintKey) }}</small>
      </button>
    </div>

    <PosEmptyState v-else size="wide" :label="terminal.regularMenuItems.value.length || terminal.serviceMenuItems.value.length ? terminal.t('pos.noMenuMatches') : terminal.t('pos.emptyMenu')" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

import type { MenuItem } from '../../shared/schemas';
import { PosBanner, PosEmptyState, PosSkeleton, PosTabs, type PosTabOption } from '../../shared/ui';
import type { CashierTerminal } from './useCashierTerminal';

type MenuGroup = 'all' | 'food' | 'services';

const props = defineProps<{
  terminal: CashierTerminal;
}>();

const group = ref<MenuGroup>('all');

const categoryOptions: Array<{ value: MenuGroup; labelKey: string }> = [
  { value: 'all', labelKey: 'pos.menuCategoryAll' },
  { value: 'food', labelKey: 'pos.menuCategoryFood' },
  { value: 'services', labelKey: 'pos.services' },
];

const categoryTabs = computed<PosTabOption[]>(() => categoryOptions.map((option) => ({
  value: option.value,
  label: props.terminal.t(option.labelKey),
})));

const visibleItems = computed(() => {
  if (group.value === 'food') return props.terminal.visibleMenuItems.value;
  if (group.value === 'services') return props.terminal.visibleServiceItems.value;
  return [...props.terminal.visibleMenuItems.value, ...props.terminal.visibleServiceItems.value];
});

const readonlyNoticeKey = computed(() => {
  if (!props.terminal.currentShift.data.value) return 'pos.noShift';
  if (props.terminal.activePrecheck.value || props.terminal.activeOrder.value?.status === 'locked') return 'pos.lockedAddHint';
  return '';
});

const disabledHintKey = computed(() => {
  if (props.terminal.activePrecheck.value || props.terminal.activeOrder.value?.status === 'locked') return '';
  if (!props.terminal.activeOrder.value) return 'pos.noActiveOrderShort';
  return 'pos.actionUnavailable';
});

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toLocaleUpperCase('ru-RU') ?? '')
    .join('');
}

function setGroup(value: string) {
  if (value === 'all' || value === 'food' || value === 'services') {
    group.value = value;
  }
}
</script>
