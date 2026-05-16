<template>
  <section class="pos-menu-area" :aria-label="terminal.t('pos.menu')">
    <div class="pos-section-head">
      <div>
        <p class="eyebrow">{{ terminal.t('pos.sections.orders') }}</p>
        <h1>{{ terminal.t('pos.menu') }}</h1>
      </div>
      <q-btn flat round icon="refresh" class="icon-touch" :aria-label="terminal.t('actions.retry')" @click="terminal.refetchMenu" />
    </div>

    <q-banner v-if="terminal.statusError.value" class="error-banner dense-banner" rounded>{{ terminal.statusError.value }}</q-banner>
    <q-banner v-if="terminal.menu.isError.value" class="error-banner dense-banner" rounded>{{ terminal.t('common.error') }}</q-banner>
    <blocking-notice
      v-if="menuLockedNotice"
      :terminal="terminal"
      :title="terminal.t(menuLockedNotice.titleKey)"
      :reason="terminal.t(menuLockedNotice.reasonKey)"
      :permission="menuLockedNotice.permission"
      icon="lock"
    />

    <div class="menu-work-surface">
      <aside class="menu-category-rail" :aria-label="terminal.t('pos.menuGroups')">
        <button v-for="option in groupOptions" :key="option.value" class="menu-filter-chip" :class="{ active: group === option.value }" type="button" @click="group = option.value">
          {{ terminal.t(option.labelKey) }}
        </button>
      </aside>

      <div class="menu-product-surface">
        <q-input
          v-model="terminal.menuSearch.value"
          dense
          outlined
          clearable
          debounce="120"
          class="menu-search square-search"
          :label="terminal.t('pos.searchMenu')"
        >
          <template #prepend>
            <q-icon name="search" />
          </template>
        </q-input>

        <div v-if="terminal.menu.isPending.value" class="pos-menu-grid">
          <q-skeleton v-for="n in 12" :key="n" class="menu-tile menu-tile-skeleton" />
        </div>

        <div v-else-if="visibleItems.length" class="pos-menu-grid">
          <button
            v-for="item in visibleItems"
            :key="item.id"
            class="menu-tile"
            type="button"
            :disabled="!terminal.canAddOrderLine.value"
            @click="terminal.openMenuItem(item)"
          >
            <span class="menu-tile-media">{{ initials(item.name) }}</span>
            <span class="menu-tile-title">{{ item.name }}</span>
            <span v-if="item.modifier_groups.length" class="menu-tile-hint">{{ terminal.t('pos.modifiersAvailable') }}</span>
            <strong>{{ terminal.money(item.price, item.currency) }}</strong>
            <small v-if="!terminal.canAddOrderLine.value">{{ terminal.t(disabledHintKey) }}</small>
          </button>
        </div>

        <div v-else class="empty-state wide">{{ terminal.regularMenuItems.value.length || terminal.serviceMenuItems.value.length ? terminal.t('pos.noMenuMatches') : terminal.t('pos.emptyMenu') }}</div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';

import type { MenuItem } from '../../shared/schemas';
import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

type MenuGroup = 'all' | 'food' | 'services';

const props = defineProps<{
  terminal: CashierTerminal;
}>();

const group = ref<MenuGroup>('all');

const groupOptions: Array<{ value: MenuGroup; labelKey: string }> = [
  { value: 'all', labelKey: 'pos.menuGroupAll' },
  { value: 'food', labelKey: 'pos.menuGroupFood' },
  { value: 'services', labelKey: 'pos.services' },
];

const visibleItems = computed(() => {
  if (group.value === 'food') return props.terminal.visibleMenuItems.value;
  if (group.value === 'services') return props.terminal.visibleServiceItems.value;
  return [...props.terminal.visibleMenuItems.value, ...props.terminal.visibleServiceItems.value];
});

const menuLockedNotice = computed(() => {
  if (!props.terminal.activeOrder.value) return null;
  return props.terminal.actionBlocker('pos.order.add_line', props.terminal.canAddOrderLine.value);
});

const disabledHintKey = computed(() => {
  if (props.terminal.activePrecheck.value || props.terminal.activeOrder.value?.status === 'locked') return 'pos.lockedAddHint';
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
