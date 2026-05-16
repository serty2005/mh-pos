<template>
  <aside class="current-order-panel" :aria-label="terminal.t('pos.currentOrderRail')">
    <q-banner v-if="terminal.orderError.value" class="error-banner dense-banner">{{ terminal.orderError.value }}</q-banner>
    <q-skeleton v-if="terminal.orderLoading.value" class="order-skeleton rail-skeleton" />

    <template v-else-if="terminal.activeOrder.value">
      <q-banner v-if="terminal.finalCheckData.value" class="success-banner">
        {{ terminal.t('pos.paymentCompleteCheckClosed') }}
      </q-banner>

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
            <q-btn flat dense square icon="more_vert" class="line-menu-button" :aria-label="terminal.t('pos.lineActions')" @click.stop="$emit('open-line-actions')" />
          </div>

          <ul v-if="line.modifiers.length" class="line-modifiers">
            <li v-for="modifier in line.modifiers" :key="modifier.id">
              <span>{{ terminal.t('pos.modifierLine', { name: modifier.name }) }}</span>
              <strong>{{ terminal.money(modifier.total_price, terminal.orderCurrency.value) }}</strong>
            </li>
          </ul>
          <p v-if="line.id === selectedLineId" class="line-note">{{ terminal.t('pos.courseValue', { value: 2 }) }}</p>
        </article>
        <div v-if="!terminal.activeLines.value.length" class="empty-state">{{ terminal.t('pos.emptyOrder') }}</div>
      </div>

      <div class="order-panel-total">
        <span>{{ terminal.t('pos.total') }}</span>
        <strong>{{ terminal.money(terminal.activeOrder.value.total, terminal.orderCurrency.value) }}</strong>
      </div>

      <div class="quantity-control" :aria-label="selectedLineName">
        <q-btn flat square icon="remove" class="quantity-button" :aria-label="terminal.t('actions.remove')" :disable="!selectedLine || !terminal.canChangeOrderLine.value || selectedLine.quantity <= 1" @click="changeSelectedQuantity(-1)" />
        <button class="quantity-value" type="button" :disabled="!selectedLine" @click="quantityDialog = true">
          {{ selectedLine ? terminal.t('pos.quantityPieces', { count: selectedLine.quantity }) : terminal.t('pos.noSelectedLine') }}
        </button>
        <q-btn flat square icon="add" class="quantity-button" :aria-label="terminal.t('actions.add')" :disable="!selectedLine || !terminal.canChangeOrderLine.value" @click="changeSelectedQuantity(1)" />
      </div>
    </template>

    <div v-else class="rail-empty">
      <p>{{ terminal.selectedTableId.value ? terminal.t('pos.noActiveOrder') : terminal.t('pos.chooseTable') }}</p>
      <q-btn color="primary" unelevated square class="touch-button primary-action" icon="receipt_long" :label="terminal.t('actions.createOrder')" :disable="!terminal.canCreateOrder.value" :loading="terminal.createOrderMutation.isPending.value" @click="terminal.createOrderMutation.mutate()" />
      <blocking-notice
        v-if="terminal.selectedTableId.value && terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)"
        :terminal="terminal"
        :title="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.titleKey ?? '')"
        :reason="terminal.t(terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.reasonKey ?? '')"
        :permission="terminal.actionBlocker('pos.order.create', terminal.canCreateOrder.value)?.permission"
        icon="lock"
      />
    </div>

    <q-dialog v-model="quantityDialog">
      <q-card class="dialog-card pos-square-dialog">
        <q-card-section>
          <h2>{{ terminal.t('pos.quantityInput') }}</h2>
        </q-card-section>
        <q-card-section>
          <q-input v-model.number="quantityDraft" type="number" min="1" outlined square inputmode="numeric" :label="terminal.t('pos.quantity')" />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat :label="terminal.t('actions.cancel')" @click="quantityDialog = false" />
          <q-btn color="primary" unelevated square :label="terminal.t('actions.save')" @click="submitQuantity" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';

import BlockingNotice from './BlockingNotice.vue';
import type { CashierTerminal } from './useCashierTerminal';

const props = defineProps<{
  terminal: CashierTerminal;
}>();

defineEmits<{
  (event: 'open-line-actions'): void;
  (event: 'open-payment'): void;
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
