<template>
  <aside class="current-order-panel" :aria-label="terminal.t('pos.currentOrderRail')">
    <PosBanner v-if="terminal.orderError.value" tone="error" :label="terminal.orderError.value" />
    <PosSkeleton v-if="terminal.orderLoading.value" kind="rail" />

    <template v-else-if="terminal.activeOrder.value">
      <PosBanner v-if="terminal.finalCheckData.value" tone="success" :label="terminal.t('pos.paymentCompleteCheckClosed')" />

      <blocking-notice
        v-if="terminal.activeOrder.value.status === 'locked' || terminal.activePrecheck.value"
        :terminal="terminal"
        :title="terminal.t('pos.blocking.lockedOrder.title')"
        :reason="terminal.t('pos.blocking.lockedOrder.reason')"
        :permission="terminal.canCancelPrecheck.value ? '' : 'pos.precheck.cancel.request'"
        icon="lock"
      />

      <div class="order-items-list">
        <article
          v-for="line in terminal.activeLines.value"
          :key="line.id"
          class="order-item-row"
          :class="{ selected: line.id === selectedLineId }"
          @click="terminal.selectOrderLine(line.id)"
        >
          <div class="row-swipe-action row-delete-action">
            <q-icon name="delete" size="22px" />
          </div>
          <div class="row-swipe-action row-more-action">
            <q-icon name="more_horiz" size="22px" />
          </div>

          <div class="order-line-main">
            <strong>{{ line.name }}</strong>
            <span>{{ line.quantity }}</span>
            <strong>{{ terminal.money(line.total_price, terminal.orderCurrency.value) }}</strong>
            <PosButton variant="neutral" mode="flat" dense square icon="more_vert" class="line-menu-button" :aria-label="terminal.t('pos.lineActions')" @click.stop="$emit('open-line-actions')" />
          </div>

          <ul v-if="line.modifiers.length" class="line-modifiers">
            <li v-for="modifier in line.modifiers" :key="modifier.id">
              <span>{{ terminal.t('pos.modifierLine', { name: modifier.name }) }}</span>
              <strong>{{ terminal.money(modifier.total_price, terminal.orderCurrency.value) }}</strong>
            </li>
          </ul>
          <p v-if="line.id === selectedLineId && line.course" class="line-note">{{ terminal.t('pos.courseValue', { value: line.course }) }}</p>
          <p v-if="line.id === selectedLineId && line.comment" class="line-note">{{ line.comment }}</p>
        </article>
        <PosEmptyState v-if="!terminal.activeLines.value.length" :label="terminal.t('pos.emptyOrder')" />
      </div>

      <div class="order-panel-total">
        <span>{{ terminal.t('pos.total') }}</span>
        <strong>{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</strong>
      </div>

      <PosQuantityStepper
        :value="selectedLine?.quantity ?? 0"
        :label="selectedLineName"
        :value-label="selectedLine ? terminal.t('pos.quantityPieces', { count: selectedLine.quantity }) : terminal.t('pos.noSelectedLine')"
        :decrement-label="terminal.t('actions.remove')"
        :increment-label="terminal.t('actions.add')"
        :edit-label="terminal.t('actions.editModifiers')"
        :disabled="!selectedLine"
        :decrement-disabled="!terminal.canChangeOrderLine.value || Boolean(selectedLine && selectedLine.quantity <= 1)"
        :increment-disabled="!terminal.canChangeOrderLine.value"
        :edit-disabled="!selectedLine || !terminal.canChangeOrderLine.value || !terminal.canEditLineModifiers(selectedLine.id)"
        editable
        show-edit
        @decrement="changeSelectedQuantity(-1)"
        @increment="changeSelectedQuantity(1)"
        @edit-value="quantityDialog = true"
        @edit="selectedLine && terminal.editLineModifiers(selectedLine.id)"
      />

      <div class="rail-actions order-rail-actions">
        <template v-if="terminal.activePrecheck.value || terminal.activeOrder.value.status === 'locked'">
          <PosButton
            variant="primary"
            primary
            icon="point_of_sale"
            :label="terminal.t('pos.cashier')"
            :disabled="terminal.remainingPayment.value <= 0"
            @click="$emit('open-payment')"
          />
          <PosButton
            variant="danger"
            mode="outline"
            icon="lock_open"
            :label="terminal.t('pos.cancelPrecheck')"
            :disabled="Boolean(terminal.activePrecheck.value && terminal.activePrecheck.value.paid_total > 0) || !terminal.canCancelPrecheck.value"
            @click="$emit('open-cancel-precheck')"
          />
        </template>
        <template v-else>
          <PosButton
            variant="secondary"
            mode="outline"
            icon="tune"
            :label="terminal.t('pos.actions')"
            :disabled="!terminal.activeLines.value.length"
            @click="$emit('open-actions')"
          />
          <PosButton
            variant="primary"
            primary
            icon="receipt_long"
            :label="terminal.t('pos.precheck')"
            :disabled="!terminal.canIssuePrecheck.value"
            :loading="terminal.issuePrecheckMutation.isPending.value"
            @click="terminal.activeOrder.value?.id && terminal.issuePrecheckMutation.mutate(terminal.activeOrder.value.id)"
          />
        </template>
      </div>
    </template>

    <div v-else class="rail-empty">
      <p>{{ terminal.selectedTableId.value ? terminal.t('pos.noActiveOrder') : terminal.t('pos.chooseTable') }}</p>
      <PosButton variant="primary" primary icon="receipt_long" :label="terminal.t('actions.createOrder')" :disabled="!terminal.canCreateOrder.value" :loading="terminal.createOrderMutation.isPending.value" @click="terminal.createOrderMutation.mutate()" />
      <blocking-notice
        v-if="terminal.selectedTableId.value && terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)"
        :terminal="terminal"
        :title="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.titleKey ?? '')"
        :reason="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.reasonKey ?? '')"
        :permission="terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.permission"
        icon="lock"
      />
    </div>

    <PosDialog v-model="quantityDialog" :title="terminal.t('pos.quantityInput')">
      <q-input v-model.number="quantityDraft" type="number" min="1" outlined square inputmode="numeric" :label="terminal.t('pos.quantity')" />
      <template #actions>
        <PosButton variant="neutral" mode="flat" :label="terminal.t('actions.cancel')" @click="quantityDialog = false" />
        <PosButton variant="primary" :label="terminal.t('actions.save')" @click="submitQuantity" />
      </template>
    </PosDialog>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';

import { PosBanner, PosButton, PosDialog, PosEmptyState, PosQuantityStepper, PosSkeleton } from '../../shared/ui';
import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

const props = defineProps<{
  terminal: CashierTerminal;
}>();

defineEmits<{
  (event: 'open-line-actions'): void;
  (event: 'open-payment'): void;
  (event: 'open-actions'): void;
  (event: 'open-cancel-precheck'): void;
}>();

const quantityDialog = ref(false);
const quantityDraft = ref(1);

const selectedLine = computed(() => props.terminal.selectedOrderLine.value);
const selectedLineId = computed(() => selectedLine.value?.id ?? '');
const selectedLineName = computed(() => selectedLine.value?.name ?? props.terminal.t('pos.noSelectedLine'));

watch(selectedLine, (line) => {
  quantityDraft.value = line?.quantity ?? 1;
}, { immediate: true });

function changeSelectedQuantity(delta: number) {
  const line = selectedLine.value;
  if (!line) return;
  props.terminal.changeQuantity(line.id, line.quantity + delta);
}

function submitQuantity() {
  const line = selectedLine.value;
  const nextQuantity = Number(quantityDraft.value);
  if (!line || !Number.isFinite(nextQuantity) || nextQuantity < 1) return;
  props.terminal.changeQuantity(line.id, nextQuantity);
  quantityDialog.value = false;
}
</script>
