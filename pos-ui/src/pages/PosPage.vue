<template>
  <q-page class="pos-page">
    <cashier-status-bar :terminal="terminal" />

    <section class="terminal-grid">
      <floor-table-selector :terminal="terminal" />
      <order-workspace :terminal="terminal" />
      <catalog-checkout-panel :terminal="terminal" />
    </section>

    <closed-orders-drawer :terminal="terminal" />
    <sync-drawer :terminal="terminal" />
    <cash-drawer-dialog :terminal="terminal" />
    <modifier-selection-dialog :terminal="terminal" />
    <precheck-cancel-dialog :terminal="terminal" />
    <refund-dialog :terminal="terminal" />
  </q-page>
</template>

<script setup lang="ts">
import CashDrawerDialog from './pos/CashDrawerDialog.vue';
import CashierStatusBar from './pos/CashierStatusBar.vue';
import CatalogCheckoutPanel from './pos/CatalogCheckoutPanel.vue';
import ClosedOrdersDrawer from './pos/ClosedOrdersDrawer.vue';
import FloorTableSelector from './pos/FloorTableSelector.vue';
import ModifierSelectionDialog from './pos/ModifierSelectionDialog.vue';
import OrderWorkspace from './pos/OrderWorkspace.vue';
import PrecheckCancelDialog from './pos/PrecheckCancelDialog.vue';
import RefundDialog from './pos/RefundDialog.vue';
import SyncDrawer from './pos/SyncDrawer.vue';
import { useCashierTerminal } from './pos/useCashierTerminal';

const terminal = useCashierTerminal();
</script>
