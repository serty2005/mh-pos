<template>
  <section class="menu-workspace" :aria-label="terminal.t('pos.menu')">
    <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner">{{ terminal.statusError.value }}</q-banner>
    <q-banner v-if="terminal.menu.isError.value" class="error-banner dense-banner">{{ terminal.t('common.error') }}</q-banner>

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

    <nav class="menu-category-tabs" :aria-label="terminal.t('pos.menuGroups')">
      <button
        v-for="option in categoryOptions"
        :key="option.value"
        class="menu-category-tab"
        :class="{ active: group === option.value }"
        type="button"
        @click="group = option.value"
      >
        {{ terminal.t(option.labelKey) }}
      </button>
    </nav>

    <div v-if="readonlyNoticeKey" class="menu-readonly-note">
      <q-icon name="lock" size="18px" />
      <span>{{ terminal.t(readonlyNoticeKey) }}</span>
    </div>

    <div v-if="!terminal.canViewMenu.value" class="empty-state wide">{{ terminal.t('pos.noPermissionForMenu') }}</div>

    <div v-else-if="terminal.menu.isPending.value" class="dish-grid">
      <q-skeleton v-for="n in 12" :key="n" class="dish-card dish-card-skeleton" />
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

    <div v-else class="empty-state wide">{{ terminal.regularMenuItems.value.length || terminal.serviceMenuItems.value.length ? terminal.t('pos.noMenuMatches') : terminal.t('pos.emptyMenu') }}</div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

import type { MenuItem } from '../../shared/schemas';
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
</script>
