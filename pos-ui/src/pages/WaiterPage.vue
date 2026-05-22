<template>
  <q-page class="waiter-page">
    <header class="waiter-topbar">
      <div>
        <p class="eyebrow">{{ t('pos.waiterMobile') }}</p>
        <h1>{{ t('pos.waiterWorkspaceTitle') }}</h1>
        <span>{{ waiterSubtitle }}</span>
      </div>
      <q-btn flat round icon="lock" :aria-label="t('actions.lock')" @click="router.push('/lock')" />
    </header>

    <PosBanner v-if="terminal.statusError.value" tone="error" :label="terminal.statusError.value" />
    <PosBanner v-if="terminal.orderError.value" tone="error" :label="terminal.orderError.value" />
    <PosBanner tone="info" :label="t('pos.waiterNoPaymentAuthority')" />
    <PosBanner v-if="terminal.orderIsLocked.value" tone="warning" :label="t('pos.waiterPrecheckLockedCopy')" />

    <PosPanel v-if="!terminal.currentShift.data.value" class="waiter-readiness-panel" :eyebrow="t('pos.serviceReadiness')" :title="t('pos.noShift')">
      <p class="waiter-muted">{{ terminal.canOpenShift.value ? t('pos.waiterOpenShiftHint') : t('pos.blocking.noShift.permissionReason') }}</p>
      <template #footer>
        <PosButton
          v-if="terminal.canOpenShift.value"
          variant="primary"
          icon="work_history"
          :label="t('actions.openShift')"
          :loading="terminal.openShiftMutation.isPending.value"
          @click="terminal.openShiftMutation.mutate()"
        />
      </template>
    </PosPanel>

    <template v-else>
      <PosPanel :eyebrow="t('pos.floorPlan')" :title="t('pos.chooseTable')" class="waiter-floor-panel">
        <template #header-side>
          <span class="waiter-chip">{{ terminal.activeTables.value.length }} {{ t('pos.tables') }}</span>
        </template>

        <PosEmptyState v-if="!terminal.canViewFloor.value" size="wide" :label="t('pos.noPermissionForFloor')" />
        <PosSkeleton v-else-if="terminal.halls.isPending.value || terminal.tables.isPending.value" />
        <PosEmptyState v-else-if="terminal.activeHalls.value.length === 0" size="wide" :label="t('pos.noHalls')" />
        <template v-else>
          <PosTabs
            :model-value="terminal.selectedHallId.value"
            :options="hallOptions"
            :accessibility-label="t('pos.halls')"
            variant="chip"
            @update:model-value="terminal.selectHall"
          />
          <PosEmptyState v-if="terminal.activeTables.value.length === 0" size="wide" :label="t('pos.noTables')" />
          <div v-else class="waiter-table-grid" :aria-label="t('pos.tables')">
            <button
              v-for="table in terminal.activeTables.value"
              :key="table.id"
              class="waiter-table-card"
              :class="{ selected: table.id === terminal.selectedTableId.value, occupied: orderForTable(table.id) }"
              type="button"
              @click="terminal.selectTable(table.id)"
            >
              <strong>{{ table.name }}</strong>
              <span>{{ orderForTable(table.id) ? t('pos.tableStatus.open') : t('pos.free') }}</span>
            </button>
          </div>
        </template>
      </PosPanel>

      <PosPanel :eyebrow="t('pos.activeOrders')" :title="selectedTableTitle" class="waiter-order-panel">
        <template #header-side>
          <PosButton
            variant="primary"
            icon="add"
            :label="t('actions.createOrder')"
            :disabled="!terminal.canCreateOrder.value"
            :loading="terminal.createOrderMutation.isPending.value"
            @click="terminal.createOrderMutation.mutate()"
          />
        </template>

        <div v-if="terminal.selectedTable.value || terminal.activeOrder.value" class="waiter-context-strip">
          <span>
            <small>{{ t('pos.selectedTable') }}</small>
            <strong>{{ terminal.selectedTable.value?.name ?? t('common.empty') }}</strong>
          </span>
          <span>
            <small>{{ t('pos.order') }}</small>
            <strong>{{ activeOrderLabel }}</strong>
          </span>
          <span :class="{ locked: terminal.orderIsLocked.value }">
            <small>{{ t('common.status') }}</small>
            <strong>{{ orderStateLabel }}</strong>
          </span>
        </div>

        <PosEmptyState v-if="!terminal.canViewOrder.value" size="wide" :label="t('pos.waiterNoOrderPermission')" />
        <PosSkeleton v-else-if="terminal.activeOrdersQuery.isPending.value || terminal.tableOrder.isFetching.value || terminal.order.isFetching.value" />
        <template v-else>
          <div v-if="terminal.activeOrders.value.length" class="waiter-active-orders" :aria-label="t('pos.activeOrders')">
            <button
              v-for="order in terminal.activeOrders.value"
              :key="order.id"
              class="waiter-order-pill"
              :class="{ selected: order.id === terminal.activeOrder.value?.id }"
              type="button"
              @click="terminal.selectOrder(order.id)"
            >
              <span>{{ order.table_name }}</span>
              <strong>{{ terminal.money(order.total, terminal.orderCurrency.value) }}</strong>
            </button>
          </div>

          <PosEmptyState v-if="!terminal.activeOrder.value" size="wide" :label="terminal.selectedTable.value ? t('pos.noActiveOrder') : t('pos.chooseTable')" />
          <div v-else class="waiter-order-lines">
            <div class="waiter-order-summary">
              <strong>{{ t('pos.total') }}</strong>
              <span>{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</span>
            </div>
            <article
              v-for="line in terminal.activeLines.value"
              :key="line.id"
              class="waiter-line-row"
              :class="{ selected: line.id === terminal.selectedOrderLine.value?.id, locked: terminal.orderIsLocked.value }"
              @click="terminal.selectOrderLine(line.id)"
            >
              <div>
                <strong>{{ line.name }}</strong>
                <span v-for="modifier in line.modifiers" :key="modifier.id">{{ t('pos.modifierLine', { name: modifier.name }) }}</span>
                <small>{{ terminal.money(line.total_price, line.currency_code) }}</small>
              </div>
              <PosQuantityStepper
                compact
                :value="line.quantity"
                :label="t('pos.quantityInput')"
                :decrement-label="t('actions.remove')"
                :increment-label="t('actions.add')"
                :disabled="!terminal.canChangeOrderLine.value"
                :min="1"
                @decrement="terminal.changeQuantity(line.id, line.quantity - 1)"
                @increment="terminal.changeQuantity(line.id, line.quantity + 1)"
              />
              <q-btn
                flat
                round
                icon="delete_outline"
                :aria-label="t('actions.voidLine')"
                :disable="!terminal.canVoidOrderLine.value"
                @click.stop="terminal.voidLine(line.id)"
              />
            </article>
            <PosEmptyState v-if="terminal.activeLines.value.length === 0" :label="t('pos.emptyOrder')" />
          </div>
        </template>
      </PosPanel>

      <PosPanel :eyebrow="t('pos.menu')" :title="t('pos.menuCategoryAll')" class="waiter-menu-panel">
        <q-input v-model="terminal.menuSearch.value" dense outlined square clearable :label="t('pos.searchMenu')" />
        <p v-if="terminal.orderIsLocked.value" class="waiter-locked-hint">{{ t('pos.lockedAddHint') }}</p>
        <PosEmptyState v-if="!terminal.canViewMenu.value" size="wide" :label="t('pos.noPermissionForMenu')" />
        <PosSkeleton v-else-if="terminal.menu.isPending.value" />
        <PosEmptyState v-else-if="terminal.visibleMenuItems.value.length === 0" size="wide" :label="terminal.menuSearch.value ? t('pos.noMenuMatches') : t('pos.emptyMenu')" />
        <div v-else class="waiter-menu-grid" :aria-label="t('pos.menu')">
          <button
            v-for="item in terminal.visibleMenuItems.value"
            :key="item.id"
            class="waiter-menu-item"
            :class="{ locked: terminal.orderIsLocked.value }"
            type="button"
            :disabled="!terminal.canAddOrderLine.value"
            @click="terminal.openMenuItem(item)"
          >
            <span>{{ item.name }}</span>
            <strong>{{ terminal.money(item.price, item.currency) }}</strong>
            <small v-if="item.modifier_groups.length">{{ t('pos.modifiersAvailable') }}</small>
          </button>
        </div>
      </PosPanel>

      <PosPanel :eyebrow="t('pos.precheck')" :title="precheckTitle" class="waiter-precheck-panel">
        <p class="waiter-muted">{{ precheckCopy }}</p>
        <div class="waiter-action-row">
          <PosButton
            variant="primary"
            icon="receipt_long"
            :label="t('actions.issuePrecheck')"
            :disabled="!terminal.canIssuePrecheck.value"
            :loading="terminal.issuePrecheckMutation.isPending.value"
            @click="terminal.issuePrecheckMutation.mutate()"
          />
          <PosButton
            variant="secondary"
            mode="outline"
            icon="print"
            :label="t('actions.reprintPrecheck')"
            :disabled="!terminal.canReprintPrecheck.value"
            :loading="terminal.reprintPrecheckMutation.isPending.value"
            @click="terminal.reprintPrecheckMutation.mutate()"
          />
        </div>
      </PosPanel>
    </template>

    <PosDialog v-model="terminal.modifierDialog.value" :title="t('pos.modifiers')" :eyebrow="terminal.modifierMenuItem.value?.name">
      <div class="waiter-modifier-list">
        <section v-for="group in terminal.modifierGroupsForDialog.value" :key="group.id" class="waiter-modifier-group">
          <div>
            <strong>{{ group.name }}</strong>
            <span>{{ modifierGroupRule(group) }}</span>
          </div>
          <article v-for="option in group.options.filter((item) => item.active)" :key="option.id" class="waiter-modifier-option">
            <div>
              <strong>{{ option.name }}</strong>
              <span>{{ option.price_minor ? terminal.money(option.price_minor, terminal.orderCurrency.value) : t('pos.freeModifier') }}</span>
            </div>
            <PosQuantityStepper
              compact
              :value="terminal.modifierQuantities.value[option.id] ?? 0"
              :label="t('pos.quantityInput')"
              :decrement-label="t('actions.remove')"
              :increment-label="t('actions.add')"
              :min="0"
              @decrement="terminal.changeModifierQuantity(option.id, (terminal.modifierQuantities.value[option.id] ?? 0) - 1)"
              @increment="terminal.changeModifierQuantity(option.id, (terminal.modifierQuantities.value[option.id] ?? 0) + 1)"
            />
          </article>
        </section>
      </div>
      <PosBanner v-if="terminal.modifierValidationKey.value" tone="warning" :label="t(terminal.modifierValidationKey.value)" />
      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="t('actions.cancel')" @click="terminal.closeModifierDialog" />
        <PosButton
          variant="primary"
          :label="t('actions.add')"
          :disabled="!terminal.canSubmitModifierSelection.value"
          :loading="terminal.addLineMutation.isPending.value"
          @click="terminal.submitModifierSelection"
        />
      </template>
    </PosDialog>
  </q-page>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { useWaiterTerminal } from './pos/useWaiterTerminal';
import { PosBanner, PosButton, PosDialog, PosEmptyState, PosPanel, PosQuantityStepper, PosSkeleton, PosTabs, type PosTabOption } from '../shared/ui';

const { t } = useI18n();
const router = useRouter();
const terminal = useWaiterTerminal();

const hallOptions = computed<PosTabOption[]>(() => terminal.activeHalls.value.map((hall) => ({ value: hall.id, label: hall.name })));
const waiterSubtitle = computed(() => terminal.actorName.value ? t('pos.waiterContext', { name: terminal.actorName.value }) : t('pos.waiter'));
const selectedTableTitle = computed(() => terminal.selectedTable.value ? t('pos.tableWithName', { table: terminal.selectedTable.value.name }) : t('pos.activeOrders'));
const activeOrderLabel = computed(() => terminal.activeOrder.value ? terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) : t('pos.noActiveOrder'));
const orderStateLabel = computed(() => {
  if (!terminal.activeOrder.value) return t('pos.noActiveOrder');
  if (terminal.orderIsLocked.value) return t('pos.lockedOrder');
  return terminal.statusLabel(terminal.activeOrder.value.status);
});
const precheckTitle = computed(() => terminal.activePrecheck.value ? t('pos.precheckIssued') : t('pos.noPrecheck'));
const precheckCopy = computed(() => {
  if (!terminal.canViewPrecheck.value) return t('pos.waiterNoPrecheckPermission');
  if (terminal.activePrecheck.value) return t('pos.waiterPrecheckLockedCopy');
  return t('pos.waiterPrecheckCopy');
});

function orderForTable(tableId: string) {
  return terminal.activeOrders.value.find((order) => order.table_id === tableId);
}

function modifierGroupRule(group: { required: boolean; min_count: number; max_count: number }) {
  if (group.required && group.max_count > 0) return t('pos.modifierGroupRequiredRange', { min: group.min_count, max: group.max_count });
  if (group.required) return t('pos.modifierGroupRequiredMin', { min: group.min_count });
  if (group.max_count > 0) return t('pos.modifierGroupOptionalMax', { max: group.max_count });
  return t('pos.optionalModifierGroup');
}
</script>
