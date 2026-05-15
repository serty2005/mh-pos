<template>
  <footer class="pos-bottom-bar" :aria-label="terminal.t('pos.quickAccess')">
    <button class="bottom-section-button" :class="{ active: menuOpen }" type="button" @click="$emit('toggle-menu')">
      <q-icon name="apps" size="24px" />
      <span>{{ terminal.t(activeSectionLabelKey) }}</span>
    </button>

    <div class="context-chip-row">
      <span v-if="terminal.selectedTable.value" class="context-chip">{{ terminal.t('pos.table') }} {{ terminal.selectedTable.value.name }}</span>
      <span v-if="terminal.activeOrder.value" class="context-chip">{{ terminal.t('pos.order') }} {{ terminal.shortId(terminal.activeOrder.value.id) }}</span>
      <span v-if="terminal.activePrecheck.value" class="context-chip warning">{{ terminal.t('pos.precheckIssued') }}</span>
      <span v-if="terminal.finalCheckData.value" class="context-chip good">{{ terminal.t('pos.checkCreated') }}</span>
      <span v-if="terminal.activeOrder.value" class="context-chip total">{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</span>
    </div>

    <div class="bottom-status-cluster">
      <button v-if="terminal.canViewSync.value" class="bottom-status" type="button" @click="terminal.syncDrawer.value = true">
        <q-icon name="sync" size="20px" />
        <span>{{ terminal.syncProblems.value > 0 ? terminal.syncProblems.value : terminal.t('status.sent') }}</span>
      </button>
      <span class="bottom-status">{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</span>
      <span class="bottom-status">{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</span>
      <span class="bottom-actor">{{ terminal.actorName.value }}</span>
      <q-btn flat round icon="lock" class="icon-touch bottom-icon" :aria-label="terminal.t('actions.lock')" @click="terminal.lockTerminal" />
      <q-btn
        flat
        round
        icon="logout"
        class="icon-touch bottom-icon"
        :aria-label="terminal.t('actions.logout')"
        :loading="terminal.logoutMutation.isPending.value"
        @click="terminal.logoutMutation.mutate()"
      />
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
</script>
