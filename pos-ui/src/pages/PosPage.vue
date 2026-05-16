<template>
  <q-page class="pos-page pos-app-shell" :class="`section-${activeSection}`">
    <header class="pos-context-bar" :aria-label="terminal.t('pos.topContext')">
      <div v-if="activeSection === 'order'" class="context-actions order-context-actions">
        <button class="context-button selected-item-button" type="button" :disabled="!terminal.selectedOrderLine.value" @click="selectedItemDialog = true">
          {{ selectedLineName }}
        </button>
        <button class="context-button" type="button" :disabled="!terminal.selectedOrderLine.value" @click="lineModifierDialog = true">
          <q-icon name="construction" size="20px" />
          <span>{{ terminal.t('pos.lineModifier') }}</span>
        </button>
        <button class="context-button" type="button" :disabled="!terminal.selectedOrderLine.value" @click="lineCommentDialog = true">
          <q-icon name="notes" size="20px" />
          <span>{{ terminal.t('pos.lineComment') }}</span>
        </button>
        <q-btn-dropdown
          flat
          square
          class="context-button course-button"
          icon="add_circle_outline"
          :label="terminal.t('pos.course')"
          :disable="!terminal.selectedOrderLine.value"
        >
          <q-list dense>
            <q-item v-for="course in courseOptions" :key="course" v-close-popup clickable @click="selectedCourse = course">
              <q-item-section>{{ course }}</q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
      </div>

      <div v-else-if="activeSection === 'floor'" class="context-actions floor-context-actions">
        <q-btn color="primary" unelevated square class="context-primary-left" icon="add" :label="terminal.t('pos.createOrderShort')" :disable="!terminal.activeTables.value.length" @click="createOrderDialog = true" />
        <q-btn-dropdown flat square class="context-button" :label="selectedHallName">
          <q-list dense>
            <q-item v-for="hall in terminal.activeHalls.value" :key="hall.id" v-close-popup clickable @click="terminal.selectHall(hall.id)">
              <q-item-section>{{ hall.name }}</q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
        <button class="context-button" type="button" @click="waiterFilterDialog = true">
          <span>{{ terminal.t('pos.waiterFilter') }}</span>
        </button>
        <button class="context-button" type="button" @click="banquetDialog = true">
          <q-icon name="add" size="20px" />
          <span>{{ terminal.t('pos.banquet') }}</span>
        </button>
      </div>

      <div v-else class="context-actions">
        <button class="context-button selected-item-button" type="button">
          {{ terminal.t(currentSectionTitleKey) }}
        </button>
      </div>

      <div class="context-main-action">
        <template v-if="activeSection === 'order' && terminal.activePrecheck.value">
          <q-btn square unelevated color="negative" class="main-split-action" :label="terminal.t('pos.cancelPrecheck')" :disable="terminal.activePrecheck.value.paid_total > 0 || !terminal.canCancelPrecheck.value" @click="terminal.cancelDialog.value = true" />
          <q-btn square unelevated color="primary" class="main-split-action" :label="terminal.t('pos.check')" :disable="terminal.remainingPayment.value <= 0" @click="paymentDialog = true" />
        </template>
        <q-btn
          v-else-if="activeSection === 'order'"
          square
          unelevated
          color="primary"
          class="main-wide-action"
          :label="orderMainActionLabel"
          :disable="!terminal.activeOrder.value || (!terminal.activePrecheck.value && !terminal.canIssuePrecheck.value)"
          :loading="terminal.issuePrecheckMutation.isPending.value"
          @click="runOrderMainAction"
        />
        <q-btn
          v-else-if="activeSection === 'floor'"
          square
          unelevated
          color="primary"
          class="main-wide-action"
          :label="terminal.t('pos.quickCheck')"
          :disable="!quickCheckAvailable"
          :title="quickCheckAvailable ? '' : terminal.t('pos.quickCheckNeedsDefaultTable')"
          @click="createQuickCheck"
        />
        <button v-else class="main-wide-action placeholder-action" type="button">
          {{ terminal.t('pos.operationalNow') }}
        </button>
      </div>
    </header>

    <section class="pos-main-workspace" :aria-label="terminal.t('pos.workspace')">
      <template v-if="activeSection === 'floor'">
        <pos-floor-section :terminal="terminal" @open-orders="openSection('order')" />
      </template>
      <template v-else-if="activeSection === 'order'">
        <pos-menu-grid :terminal="terminal" />
        <pos-order-rail
          :terminal="terminal"
          @open-line-actions="lineActionsDialog = true"
          @open-payment="paymentDialog = true"
        />
      </template>
      <section v-else class="section-workspace">
        <component :is="fallbackComponent" :terminal="terminal" />
      </section>
    </section>

    <pos-bottom-bar
      :terminal="terminal"
      :active-section="activeSection"
      :menu-open="sectionMenuOpen"
      @toggle-menu="sectionMenuOpen = !sectionMenuOpen"
      @open-discounts="discountDialog = true"
    />

    <div v-if="sectionMenuOpen" class="pos-section-menu-layer" @click.self="sectionMenuOpen = false">
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
    </div>

    <q-dialog v-model="selectedItemDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section>
          <p class="eyebrow">{{ terminal.t('pos.selectedLine') }}</p>
          <h2>{{ selectedLineName }}</h2>
        </q-card-section>
        <q-card-section class="dialog-copy">{{ terminal.t('pos.selectedLinePlaceholder') }}</q-card-section>
        <q-card-actions align="right"><q-btn flat :label="terminal.t('actions.close')" @click="selectedItemDialog = false" /></q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="lineModifierDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section>
          <p class="eyebrow">{{ selectedLineName }}</p>
          <h2>{{ terminal.t('pos.lineModifier') }}</h2>
        </q-card-section>
        <q-card-section class="dialog-copy">{{ terminal.t('pos.lineModifierPlaceholder') }}</q-card-section>
        <q-card-actions align="right"><q-btn flat :label="terminal.t('actions.close')" @click="lineModifierDialog = false" /></q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="lineCommentDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section>
          <p class="eyebrow">{{ selectedLineName }}</p>
          <h2>{{ terminal.t('pos.lineComment') }}</h2>
        </q-card-section>
        <q-card-section>
          <q-input v-model="lineCommentDraft" type="textarea" outlined square autogrow :label="terminal.t('pos.lineCommentInput')" />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat :label="terminal.t('actions.cancel')" @click="lineCommentDialog = false" />
          <q-btn color="primary" unelevated square :label="terminal.t('actions.save')" @click="lineCommentDialog = false" />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="lineActionsDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section>
          <p class="eyebrow">{{ selectedLineName }}</p>
          <h2>{{ terminal.t('pos.lineActions') }}</h2>
        </q-card-section>
        <q-card-section class="line-action-grid">
          <button v-for="key in lineActionKeys" :key="key" class="line-action-button" type="button">{{ terminal.t(key) }}</button>
        </q-card-section>
        <q-card-actions align="right"><q-btn flat :label="terminal.t('actions.close')" @click="lineActionsDialog = false" /></q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="createOrderDialog">
      <q-card class="create-order-dialog pos-square-dialog">
        <q-card-section>
          <h2>{{ terminal.t('pos.createOrder') }}</h2>
        </q-card-section>
        <q-card-section class="create-order-body">
          <nav class="hall-tabs">
            <button v-for="hall in terminal.activeHalls.value" :key="hall.id" class="hall-chip" :class="{ selected: hall.id === terminal.selectedHallId.value }" type="button" @click="terminal.selectHall(hall.id)">
              {{ hall.name }}
            </button>
          </nav>
          <div class="floor-table-grid dialog-table-grid">
            <button v-for="table in terminal.activeTables.value" :key="table.id" class="floor-table-tile is-free" type="button" @click="createOrderAtTable(table.id)">
              <span>{{ table.name }}</span>
              <small>{{ terminal.t('pos.free') }}</small>
            </button>
          </div>
        </q-card-section>
      </q-card>
    </q-dialog>

    <q-dialog v-model="waiterFilterDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section><h2>{{ terminal.t('pos.waiterFilter') }}</h2></q-card-section>
        <q-card-section class="waiter-filter-list">
          <q-checkbox v-for="waiter in waiterFilters" :key="waiter.nameKey" v-model="waiter.selected" :label="`${terminal.t(waiter.nameKey)} (${waiter.count})`" />
        </q-card-section>
        <q-card-actions align="right"><q-btn flat :label="terminal.t('actions.close')" @click="waiterFilterDialog = false" /></q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="banquetDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section>
          <h2>{{ terminal.t('pos.banquetPlannedTitle') }}</h2>
        </q-card-section>
        <q-card-section class="dialog-copy">
          <p>{{ terminal.t('pos.banquetPlannedBody') }}</p>
          <ul class="planned-list">
            <li>{{ terminal.t('pos.banquetPlanTables') }}</li>
            <li>{{ terminal.t('pos.banquetPlanTime') }}</li>
            <li>{{ terminal.t('pos.banquetPlanPrepayment') }}</li>
            <li>{{ terminal.t('pos.banquetPlanPreorder') }}</li>
            <li>{{ terminal.t('pos.banquetPlanFutureOrder') }}</li>
          </ul>
        </q-card-section>
        <q-card-actions align="right"><q-btn flat :label="terminal.t('actions.close')" @click="banquetDialog = false" /></q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="discountDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section><h2>{{ terminal.t('pos.discountSurcharge') }}</h2></q-card-section>
        <q-card-section class="dialog-copy">{{ terminal.t('pos.discountSurchargePlaceholder') }}</q-card-section>
        <q-card-actions align="right"><q-btn flat :label="terminal.t('actions.close')" @click="discountDialog = false" /></q-card-actions>
      </q-card>
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
import { computed, nextTick, reactive, ref, shallowRef, watch } from 'vue';

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

type PosSectionId = 'order' | 'floor' | 'delivery' | 'shift' | 'analytics' | 'settings';

const terminal = useCashierTerminal();
const activeSection = ref<PosSectionId>('floor');
const sectionMenuOpen = ref(false);
const paymentDialog = ref(false);
const actionsDialog = ref(false);
const selectedItemDialog = ref(false);
const lineModifierDialog = ref(false);
const lineCommentDialog = ref(false);
const lineActionsDialog = ref(false);
const createOrderDialog = ref(false);
const waiterFilterDialog = ref(false);
const banquetDialog = ref(false);
const discountDialog = ref(false);
const sectionWasInitialized = ref(false);
const selectedCourse = ref('2');
const lineCommentDraft = ref('');

const fallbackComponent = shallowRef(PosCashSection);

const sections: Array<{ id: PosSectionId; icon: string; labelKey: string }> = [
  { id: 'order', icon: 'apps', labelKey: 'pos.sections.order' },
  { id: 'floor', icon: 'grid_view', labelKey: 'pos.sections.floor' },
  { id: 'delivery', icon: 'local_shipping', labelKey: 'pos.sections.delivery' },
  { id: 'shift', icon: 'schedule', labelKey: 'pos.sections.shift' },
  { id: 'analytics', icon: 'bar_chart', labelKey: 'pos.sections.analytics' },
  { id: 'settings', icon: 'settings', labelKey: 'pos.sections.settings' },
];

const courseOptions = ['1', '2', '3', '4', '5', 'VIP'];
const lineActionKeys = [
  'pos.moveToAnotherTable',
  'pos.moveToAnotherOrder',
  'pos.splitDish',
  'pos.enableFractionalSplit',
];

const waiterFilters = reactive([
  { nameKey: 'pos.mockWaiters.oleg', count: 12, selected: true },
  { nameKey: 'pos.mockWaiters.anna', count: 8, selected: true },
  { nameKey: 'pos.mockWaiters.ivan', count: 3, selected: false },
]);

const selectedLineName = computed(() => terminal.selectedOrderLine.value?.name ?? terminal.t('pos.noSelectedLine'));
const selectedHallName = computed(() => terminal.activeHalls.value.find((hall) => hall.id === terminal.selectedHallId.value)?.name ?? terminal.t('pos.halls'));
const currentSectionTitleKey = computed(() => sections.find((section) => section.id === activeSection.value)?.labelKey ?? 'pos.sections.order');
const quickCheckAvailable = computed(() => Boolean(terminal.activeTables.value[0] && terminal.currentShift.data.value));
const orderMainActionLabel = computed(() => {
  if (!terminal.activeOrder.value) return terminal.t('actions.save');
  if (terminal.finalCheckData.value) return terminal.t('pos.check');
  return terminal.t('pos.precheck');
});

watch(terminal.activeTables, (tables) => {
  if (sectionWasInitialized.value || terminal.tables.isPending.value) return;
  activeSection.value = tables.length > 1 ? 'floor' : 'order';
  sectionWasInitialized.value = true;
}, { immediate: true });

function openSection(section: PosSectionId) {
  activeSection.value = section;
  sectionMenuOpen.value = false;
  if (section === 'shift') fallbackComponent.value = PosCashSection;
  if (section === 'analytics') fallbackComponent.value = PosReportsSection;
  if (section === 'settings' || section === 'delivery') fallbackComponent.value = PosActivitySection;
}

function runOrderMainAction() {
  if (terminal.activePrecheck.value || terminal.finalCheckData.value) {
    paymentDialog.value = true;
    return;
  }
  if (terminal.activeOrder.value?.id) terminal.issuePrecheckMutation.mutate(terminal.activeOrder.value.id);
}

async function createQuickCheck() {
  const defaultTable = terminal.activeTables.value[0];
  if (!defaultTable) return;
  terminal.selectTable(defaultTable.id);
  await nextTick();
  terminal.createOrderMutation.mutate();
  openSection('order');
}

function createOrderAtTable(tableId: string) {
  terminal.selectTable(tableId);
  createOrderDialog.value = false;
  void nextTick(() => {
    terminal.createOrderMutation.mutate();
    openSection('order');
  });
}
</script>
