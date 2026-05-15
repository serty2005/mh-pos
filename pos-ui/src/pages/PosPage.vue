<template>
  <q-page class="pos-page">
    <section class="pos-stage" :class="`section-${activeSection}`">
      <pos-floor-section v-if="activeSection === 'floor'" :terminal="terminal" @open-orders="openSection('orders')" />
      <section v-else-if="activeSection === 'orders'" class="pos-order-screen">
        <pos-menu-grid :terminal="terminal" />
        <pos-order-rail
          :terminal="terminal"
          @open-actions="actionsDialog = true"
          @open-payment="paymentDialog = true"
          @open-floor="openSection('floor')"
        />
      </section>
      <pos-activity-section v-else-if="activeSection === 'activity'" :terminal="terminal" />
      <pos-reports-section v-else-if="activeSection === 'reports'" :terminal="terminal" />
      <pos-cash-section v-else :terminal="terminal" />
    </section>

    <pos-bottom-bar
      :terminal="terminal"
      :active-section="activeSection"
      :menu-open="sectionMenuOpen"
      @toggle-menu="sectionMenuOpen = !sectionMenuOpen"
    />

    <q-dialog v-model="sectionMenuOpen" position="left">
      <nav class="pos-section-menu" :aria-label="terminal.t('pos.sections.title')">
        <button
          v-for="section in sections"
          :key="section.id"
          class="section-menu-item"
          :class="{ active: activeSection === section.id }"
          type="button"
          @click="openSection(section.id)"
        >
          <q-icon :name="section.icon" size="24px" />
          <span>{{ terminal.t(section.labelKey) }}</span>
        </button>
      </nav>
    </q-dialog>

    <pos-payment-dialog v-model="paymentDialog" :terminal="terminal" />
    <pos-actions-dialog v-model="actionsDialog" :terminal="terminal" />
    <closed-orders-drawer :terminal="terminal" />
    <sync-drawer :terminal="terminal" />
    <cash-drawer-dialog :terminal="terminal" />
    <modifier-selection-dialog :terminal="terminal" />
    <precheck-cancel-dialog :terminal="terminal" />
    <refund-dialog :terminal="terminal" />
  </q-page>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

import CashDrawerDialog from './pos/CashDrawerDialog.vue';
import ClosedOrdersDrawer from './pos/ClosedOrdersDrawer.vue';
import ModifierSelectionDialog from './pos/ModifierSelectionDialog.vue';
import PosActionsDialog from './pos/PosActionsDialog.vue';
import PosActivitySection from './pos/PosActivitySection.vue';
import PosBottomBar from './pos/PosBottomBar.vue';
import PosCashSection from './pos/PosCashSection.vue';
import PosFloorSection from './pos/PosFloorSection.vue';
import PosMenuGrid from './pos/PosMenuGrid.vue';
import PosOrderRail from './pos/PosOrderRail.vue';
import PosPaymentDialog from './pos/PosPaymentDialog.vue';
import PosReportsSection from './pos/PosReportsSection.vue';
import PrecheckCancelDialog from './pos/PrecheckCancelDialog.vue';
import RefundDialog from './pos/RefundDialog.vue';
import SyncDrawer from './pos/SyncDrawer.vue';
import { useCashierTerminal } from './pos/useCashierTerminal';

type PosSectionId = 'floor' | 'orders' | 'activity' | 'reports' | 'cash';

const terminal = useCashierTerminal();
const activeSection = ref<PosSectionId>('floor');
const sectionMenuOpen = ref(false);
const paymentDialog = ref(false);
const actionsDialog = ref(false);
const sectionWasInitialized = ref(false);

const sections: Array<{ id: PosSectionId; icon: string; labelKey: string }> = [
  { id: 'floor', icon: 'table_restaurant', labelKey: 'pos.sections.floor' },
  { id: 'orders', icon: 'restaurant_menu', labelKey: 'pos.sections.orders' },
  { id: 'activity', icon: 'history', labelKey: 'pos.sections.activity' },
  { id: 'reports', icon: 'monitoring', labelKey: 'pos.sections.reports' },
  { id: 'cash', icon: 'point_of_sale', labelKey: 'pos.sections.cash' },
];

watch(terminal.activeTables, (tables) => {
  if (sectionWasInitialized.value || terminal.tables.isPending.value) return;
  activeSection.value = tables.length > 1 ? 'floor' : 'orders';
  sectionWasInitialized.value = true;
}, { immediate: true });

function openSection(section: PosSectionId) {
  activeSection.value = section;
  sectionMenuOpen.value = false;
}
</script>
