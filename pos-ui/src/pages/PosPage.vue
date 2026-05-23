<template>
  <q-page class="pos-page pos-app-shell" :class="`section-${activeSection}`">
    <header class="pos-context-bar" :aria-label="terminal.t('pos.topContext')">
      <div v-if="activeSection === 'order' && terminal.activeOrder.value" class="context-actions order-context-actions">
        <PosContextButton :label="selectedLineName" passive selected :aria-label="terminal.t('pos.selectedLine')" />
        <PosContextButton
          icon="tune"
          :label="terminal.t('pos.lineModifier')"
          :disabled="!canEditSelectedLineModifiers"
          :title="editSelectedLineModifiersTitle"
          @click="editSelectedLineModifiers"
        />
        <PosContextButton
          icon="notes"
          :label="terminal.t('pos.lineComment')"
          :disabled="!canEditSelectedLineDetails"
          @click="lineCommentDialog = true"
        />
        <q-btn-dropdown
          flat
          square
          class="context-button course-button"
          icon="add_circle_outline"
          :label="terminal.t('pos.course')"
          :disable="!canEditSelectedLineDetails"
        >
          <q-list dense>
            <q-item v-for="course in courseOptions" :key="course" v-close-popup clickable @click="saveCourse(course)">
              <q-item-section>{{ course }}</q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
      </div>

      <div v-else-if="activeSection === 'floor'" class="context-actions floor-context-actions">
        <PosButton
          variant="primary"
          class="context-primary-left"
          icon="add"
          :label="terminal.t('pos.createOrderShort')"
          :disabled="!terminal.activeTables.value.length || !terminal.canStartOrderFromFloor.value"
          :title="terminal.activeTables.value.length ? terminal.t('pos.createOrder') : terminal.t('pos.noTables')"
          @click="createOrderDialog = true"
        />
        <q-btn-dropdown flat square class="context-button" :label="selectedHallName">
          <q-list dense>
            <q-item v-for="hall in terminal.activeHalls.value" :key="hall.id" v-close-popup clickable @click="terminal.selectHall(hall.id)">
              <q-item-section>{{ hall.name }}</q-item-section>
            </q-item>
          </q-list>
        </q-btn-dropdown>
        <PosContextButton icon="person" :label="waiterContextLabel" passive />
        <PosContextButton icon="event" :label="terminal.t('pos.banquetBacklog')" passive backlog :title="terminal.t('pos.backlogFeatureReason')" />
      </div>

      <div v-else class="context-actions">
        <PosContextButton :label="terminal.t(currentSectionTitleKey)" selected />
      </div>

      <div v-if="activeSection === 'order'" class="top-status-grid order-status-grid">
        <span class="top-status-cell">
          <small>{{ terminal.t('pos.orderNumberLabel') }}</small>
          <strong>{{ terminal.activeOrder.value ? terminal.shortId(terminal.activeOrder.value.id) : '-' }}</strong>
        </span>
        <span class="top-status-cell">
          <small>{{ terminal.t('pos.hallTable') }}</small>
          <strong>{{ hallTableLabel }}</strong>
        </span>
        <span class="top-status-cell">
          <small>{{ terminal.t('pos.waiter') }}</small>
          <strong>{{ terminal.actorName.value || '-' }}</strong>
        </span>
        <span class="top-status-cell two-line-cell">
          <small>{{ openedLabel }}</small>
          <strong>{{ pricingAdjustmentsLabel }}</strong>
        </span>
        <span class="top-status-cell" :class="{ good: terminal.currentShift.data.value }">
          <small>{{ terminal.t('pos.shift') }}</small>
          <strong>{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</strong>
        </span>
        <span class="top-status-cell" :class="{ good: terminal.currentCashSession.data.value }">
          <small>{{ terminal.t('pos.cashSession') }}</small>
          <strong>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</strong>
        </span>
        <span class="top-status-cell technical-cell">
          <small>{{ terminal.t('pos.session') }}</small>
          <strong>{{ terminal.backendSessionLabel.value }}</strong>
        </span>
      </div>

      <div v-else-if="activeSection === 'floor'" class="top-status-grid floor-status-grid">
        <span class="top-status-cell">
          <small>{{ terminal.t('pos.restaurant') }}</small>
          <strong>{{ terminal.shortId(terminal.auth.restaurantId || '-') }}</strong>
        </span>
        <span class="top-status-cell">
          <small>{{ terminal.t('pos.actor') }}</small>
          <strong>{{ terminal.actorName.value || '-' }}</strong>
        </span>
        <span class="top-status-cell">
          <small>{{ terminal.t('common.node') }}</small>
          <strong>{{ terminal.shortId(terminal.auth.nodeDeviceId || '-') }}</strong>
        </span>
        <span class="top-status-cell" :class="{ good: terminal.currentShift.data.value }">
          <small>{{ terminal.t('pos.shift') }}</small>
          <strong>{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</strong>
        </span>
        <span class="top-status-cell" :class="{ good: terminal.currentCashSession.data.value }">
          <small>{{ terminal.t('pos.cashSession') }}</small>
          <strong>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</strong>
        </span>
        <span class="top-status-cell technical-cell">
          <small>{{ terminal.t('pos.session') }}</small>
          <strong>{{ terminal.backendSessionLabel.value }}</strong>
        </span>
      </div>

      <div v-else class="top-status-grid">
        <span class="top-status-cell">
          <small>{{ terminal.t(currentSectionTitleKey) }}</small>
          <strong>{{ sectionStatusLabel }}</strong>
        </span>
        <span class="top-status-cell">
          <small>{{ terminal.t('pos.restaurant') }}</small>
          <strong>{{ terminal.shortId(terminal.auth.restaurantId || '-') }}</strong>
        </span>
        <span class="top-status-cell">
          <small>{{ terminal.t('pos.actor') }}</small>
          <strong>{{ terminal.actorName.value || '-' }}</strong>
        </span>
        <span class="top-status-cell" :class="{ good: terminal.currentShift.data.value }">
          <small>{{ terminal.t('pos.shift') }}</small>
          <strong>{{ terminal.currentShift.data.value ? terminal.t('status.open') : terminal.t('pos.noShift') }}</strong>
        </span>
        <span class="top-status-cell" :class="{ good: terminal.currentCashSession.data.value }">
          <small>{{ terminal.t('pos.cashSession') }}</small>
          <strong>{{ terminal.currentCashSession.data.value ? terminal.t('status.open') : terminal.t('pos.noCashSession') }}</strong>
        </span>
        <span class="top-status-cell technical-cell">
          <small>{{ terminal.t('pos.session') }}</small>
          <strong>{{ terminal.backendSessionLabel.value }}</strong>
        </span>
      </div>

      <q-btn flat square icon="lock" class="top-lock-button" :aria-label="terminal.t('actions.lock')" @click="terminal.lockTerminal" />
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
          @open-actions="actionsDialog = true"
          @open-payment="paymentDialog = true"
          @open-cancel-precheck="terminal.cancelDialog.value = true"
        />
      </template>
      <component v-else :is="sectionComponent" :terminal="terminal" />
    </section>

    <pos-bottom-bar
      :terminal="terminal"
      :active-section="activeSection"
      :menu-open="sectionMenuOpen"
      @toggle-menu="sectionMenuOpen = !sectionMenuOpen"
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

    <PosDialog v-model="lineCommentDialog" :eyebrow="selectedLineName" :title="terminal.t('pos.lineComment')">
      <q-input v-model="terminal.lineCommentDraft.value" type="textarea" outlined square autogrow :label="terminal.t('pos.lineCommentInput')" />
      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.cancel')" @click="lineCommentDialog = false" />
        <PosButton variant="primary" :label="terminal.t('actions.save')" :loading="terminal.lineDetailsMutation.isPending.value" @click="saveLineComment" />
      </template>
    </PosDialog>

    <PosDialog v-model="lineActionsDialog" :eyebrow="selectedLineName" :title="terminal.t('pos.lineActions')" body-class="line-action-grid">
      <article v-for="item in lineActionItems" :key="item.labelKey" class="backlog-action-card" aria-disabled="true">
        <q-icon :name="item.icon" size="20px" />
        <span>
          <strong>{{ terminal.t(item.labelKey) }}</strong>
          <small>{{ terminal.t(item.reasonKey) }}</small>
        </span>
      </article>
      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.close')" @click="lineActionsDialog = false" />
      </template>
    </PosDialog>

    <PosDialog v-model="createOrderDialog" :title="terminal.t('pos.createOrder')" card-class="create-order-dialog" body-class="create-order-body">
      <PosTabs
        :model-value="terminal.selectedHallId.value"
        :options="hallTabOptions"
        :accessibility-label="terminal.t('pos.halls')"
        variant="chip"
        @update:model-value="terminal.selectHall"
      />
      <div class="floor-table-grid dialog-table-grid">
        <button v-for="table in terminal.activeTables.value" :key="table.id" class="floor-table-tile is-free" type="button" @click="createOrderAtTable(table.id)">
          <span>{{ table.name }}</span>
          <small>{{ terminal.t('pos.free') }}</small>
        </button>
      </div>
    </PosDialog>

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
import { computed, nextTick, ref, watch } from 'vue';

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
import { PosButton, PosContextButton, PosDialog, PosTabs, type PosTabOption } from '../shared/ui';

type PosSectionId = 'floor' | 'order' | 'activity' | 'reports' | 'cash';

const terminal = useCashierTerminal();
const activeSection = ref<PosSectionId>('floor');
const sectionMenuOpen = ref(false);
const paymentDialog = ref(false);
const actionsDialog = ref(false);
const lineCommentDialog = ref(false);
const lineActionsDialog = ref(false);
const createOrderDialog = ref(false);
const sectionWasInitialized = ref(false);

const sections: Array<{ id: PosSectionId; icon: string; labelKey: string }> = [
  { id: 'floor', icon: 'grid_view', labelKey: 'pos.sections.floor' },
  { id: 'order', icon: 'apps', labelKey: 'pos.sections.order' },
  { id: 'activity', icon: 'history', labelKey: 'pos.sections.activity' },
  { id: 'reports', icon: 'bar_chart', labelKey: 'pos.sections.reports' },
  { id: 'cash', icon: 'point_of_sale', labelKey: 'pos.sections.cash' },
];

const courseOptions = ['1', '2', '3', '4', '5', 'VIP'];
const lineActionItems = [
  { labelKey: 'pos.moveToAnotherTable', reasonKey: 'pos.lineActionBacklogReason', icon: 'table_restaurant' },
  { labelKey: 'pos.moveToAnotherOrder', reasonKey: 'pos.lineActionBacklogReason', icon: 'receipt_long' },
  { labelKey: 'pos.splitDish', reasonKey: 'pos.lineActionBacklogReason', icon: 'call_split' },
  { labelKey: 'pos.enableFractionalSplit', reasonKey: 'pos.lineActionBacklogReason', icon: 'splitscreen' },
];

const selectedLineName = computed(() => terminal.selectedOrderLine.value?.name ?? terminal.t('pos.noSelectedLine'));
const canEditSelectedLineDetails = computed(() => Boolean(terminal.selectedOrderLine.value && terminal.canChangeOrderLine.value));
const canEditSelectedLineModifiers = computed(() => {
  const lineId = terminal.selectedOrderLine.value?.id;
  return Boolean(lineId && terminal.canChangeOrderLine.value && terminal.canEditLineModifiers(lineId));
});
const editSelectedLineModifiersTitle = computed(() => canEditSelectedLineModifiers.value ? terminal.t('actions.editModifiers') : terminal.t('pos.modifierEditUnavailable'));
const waiterContextLabel = computed(() => terminal.actorName.value ? terminal.t('pos.waiterContext', { name: terminal.actorName.value }) : terminal.t('pos.waiter'));
const selectedHallName = computed(() => terminal.activeHalls.value.find((hall) => hall.id === terminal.selectedHallId.value)?.name ?? terminal.t('pos.halls'));
const hallTabOptions = computed<PosTabOption[]>(() => terminal.activeHalls.value.map((hall) => ({ value: hall.id, label: hall.name })));
const currentSectionTitleKey = computed(() => sections.find((section) => section.id === activeSection.value)?.labelKey ?? 'pos.sections.order');
const hallTableLabel = computed(() => {
  const hall = terminal.activeHalls.value.find((item) => item.id === terminal.selectedHallId.value)?.name ?? terminal.t('pos.currentHall');
  const table = terminal.selectedTable.value?.name ?? '-';
  return `${hall} / ${table}`;
});
const openedLabel = computed(() => {
  const openedAt = terminal.activeOrder.value?.opened_at;
  if (!openedAt) return terminal.t('pos.openedEmpty');
  return terminal.t('pos.openedAt', { value: formatOpenedAt(openedAt) });
});
const pricingAdjustmentsLabel = computed(() => {
  const discount = terminal.activePrecheck.value?.discount_total ?? terminal.finalCheckData.value?.discount_total ?? terminal.activeOrder.value?.discount_total ?? 0;
  const surcharge = terminal.activePrecheck.value?.surcharge_total ?? terminal.finalCheckData.value?.surcharge_total ?? 0;
  if (discount === 0 && surcharge === 0) return terminal.t('pos.pricingAdjustmentsNone');
  return terminal.t('pos.pricingAdjustmentsValue', {
    discount: terminal.money(discount, terminal.orderCurrency.value),
    surcharge: terminal.money(surcharge, terminal.orderCurrency.value),
  });
});
const completedCount = computed(() => (terminal.closedOrders.data.value ?? []).length);
const sectionStatusLabel = computed(() => {
  if (activeSection.value === 'cash') {
    return terminal.currentCashSession.data.value ? terminal.t('pos.cashSessionOpen') : terminal.t('pos.noCashSession');
  }
  if (activeSection.value === 'activity' || activeSection.value === 'reports') {
    return terminal.t('pos.closedOrdersCount', { count: completedCount.value });
  }
  return terminal.t(currentSectionTitleKey.value);
});
const sectionComponent = computed(() => {
  if (activeSection.value === 'activity') return PosActivitySection;
  if (activeSection.value === 'reports') return PosReportsSection;
  return PosCashSection;
});

watch(terminal.activeTables, (tables) => {
  if (sectionWasInitialized.value || terminal.tables.isPending.value) return;
  activeSection.value = tables.length > 1 ? 'floor' : 'order';
  sectionWasInitialized.value = true;
}, { immediate: true });

function openSection(section: PosSectionId) {
  activeSection.value = section;
  sectionMenuOpen.value = false;
}

watch(lineCommentDialog, (open) => {
  if (open) terminal.primeLineDetailsDraft();
});

function saveCourse(course: string) {
  if (!canEditSelectedLineDetails.value) return;
  terminal.lineCourseDraft.value = course;
  terminal.lineCommentDraft.value = terminal.selectedOrderLine.value?.comment ?? '';
  terminal.saveSelectedLineDetails();
}

function saveLineComment() {
  if (!canEditSelectedLineDetails.value) return;
  terminal.saveSelectedLineDetails();
  lineCommentDialog.value = false;
}

function editSelectedLineModifiers() {
  const lineId = terminal.selectedOrderLine.value?.id;
  if (!lineId || !canEditSelectedLineModifiers.value) return;
  terminal.editLineModifiers(lineId);
}

function createOrderAtTable(tableId: string) {
  terminal.selectTable(tableId);
  createOrderDialog.value = false;
  void nextTick(() => {
    if (!terminal.canCreateOrder.value) return;
    terminal.createOrderMutation.mutate();
    openSection('order');
  });
}

function formatOpenedAt(value: string) {
  const date = new Date(value);
  const now = new Date();
  const sameDay = date.toDateString() === now.toDateString();
  return new Intl.DateTimeFormat('ru-RU', sameDay ? { hour: '2-digit', minute: '2-digit' } : { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }).format(date);
}
</script>
