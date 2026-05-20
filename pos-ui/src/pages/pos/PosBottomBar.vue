<template>
  <footer class="pos-bottom-bar" :aria-label="terminal.t('pos.quickAccess')">
    <button class="bottom-section-button" :class="{ active: menuOpen }" type="button" @click="$emit('toggle-menu')">
      <q-icon name="apps" size="24px" />
      <span>{{ terminal.t(activeSectionLabelKey) }}</span>
    </button>

    <div v-if="hasSectionActions" class="bottom-action-area">
      <slot name="actions" />
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import type { CashierTerminal } from './useCashierTerminal';

const props = defineProps<{
  terminal: CashierTerminal;
  activeSection: string;
  menuOpen: boolean;
}>();

defineEmits<{
  (event: 'toggle-menu'): void;
}>();

const activeSectionLabelKey = computed(() => `pos.sections.${props.activeSection}`);
const hasSectionActions = computed(() => ['order', 'floor', 'shift'].includes(props.activeSection));
</script>
